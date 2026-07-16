//! Telemetry stack for the Astra sync daemon.
//!
//! Provides structured JSON logging to stderr and optional OpenTelemetry OTLP
//! trace export.  All log records are annotated with `trace_id`, `lane_id`,
//! `kiosk_id`, and `tenant_id` when those values are available in the current
//! span context.  Payment card numbers (PANs) and biometric hashes are
//! redacted from log output.

#![deny(unsafe_code)]

pub mod baggage;
pub mod metrics;

use std::io::{self, Write};
use std::sync::Arc;

use opentelemetry::baggage::BaggageExt;
use opentelemetry::propagation::TextMapCompositePropagator;
use opentelemetry::trace::Span as _;
use opentelemetry::trace::TracerProvider as _;
use opentelemetry_sdk::propagation::{BaggagePropagator, TraceContextPropagator};
use opentelemetry_sdk::export::trace::SpanData;
use opentelemetry_sdk::runtime::Tokio;
use opentelemetry_sdk::trace::{Config, SpanProcessor};
use opentelemetry_sdk::Resource;
use tracing_subscriber::layer::{Layer, SubscriberExt};
use tracing_subscriber::{EnvFilter, Registry};

/// Initializes the sync daemon's telemetry stack.
///
/// * JSON log output to stderr with configurable level.
/// * OpenTelemetry OTLP trace exporter when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
/// * Automatic redaction of PAN-like sequences in log output.
///
/// The returned shutdown guard should be awaited during graceful shutdown so
/// that spans are flushed instead of dropped.
pub fn init(
    service_name: &str,
    service_version: &str,
    environment: &str,
    log_level: &str,
) -> Result<ShutdownGuard, TelemetryInitError> {
    let env_filter = EnvFilter::try_new(log_level)
        .or_else(|_| EnvFilter::try_new("info"))
        .map_err(|e| TelemetryInitError::InvalidFilter(e.to_string()))?;

    let fmt_layer = tracing_subscriber::fmt::layer()
        .json()
        .with_target(true)
        .with_thread_ids(true)
        .with_line_number(true)
        .flatten_event(true)
        .with_writer(move || RedactingWriter::new(io::stderr()))
        .with_filter(env_filter);

    let resource = Resource::new(vec![
        opentelemetry::KeyValue::new("service.name", service_name.to_string()),
        opentelemetry::KeyValue::new("service.version", service_version.to_string()),
        opentelemetry::KeyValue::new("deployment.environment.name", environment.to_string()),
    ]);

    set_global_propagator();

    if std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT").is_ok() {
        let exporter = opentelemetry_otlp::new_exporter()
            .tonic()
            .build_span_exporter()
            .map_err(TelemetryInitError::OpenTelemetry)?;

        let tracer_provider = opentelemetry_sdk::trace::TracerProvider::builder()
            .with_span_processor(BaggageSpanProcessor::new())
            .with_batch_exporter(exporter, Tokio)
            .with_config(Config::default().with_resource(resource))
            .build();

        let tracer = tracer_provider.tracer("astra-syncd");
        let otel_layer = tracing_opentelemetry::layer().with_tracer(tracer);

        let subscriber = Registry::default().with(fmt_layer).with(otel_layer);
        tracing::subscriber::set_global_default(subscriber)
            .map_err(|e| TelemetryInitError::SetSubscriber(e.to_string()))?;
    } else {
        let subscriber = Registry::default().with(fmt_layer);
        tracing::subscriber::set_global_default(subscriber)
            .map_err(|e| TelemetryInitError::SetSubscriber(e.to_string()))?;
    }

    Ok(ShutdownGuard {})
}

fn endpoint() -> String {
    std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT").unwrap_or_else(|_| "http://localhost:4317".into())
}

/// Set the global propagator to a composite of TraceContext + Baggage.
fn set_global_propagator() {
    let composite = TextMapCompositePropagator::new(vec![
        Box::new(TraceContextPropagator::new()),
        Box::new(BaggagePropagator::new()),
    ]);
    opentelemetry::global::set_text_map_propagator(composite);
}

/// A span processor that copies baggage entries onto span attributes.
///
/// When a span is created in a context that carries baggage, this processor
/// enriches the span with key OTel attributes (kiosk_id, lane_id, tenant_id,
/// transaction_id) so that they appear in OTLP export without manual wiring.
#[derive(Debug)]
pub struct BaggageSpanProcessor;

impl BaggageSpanProcessor {
    pub fn new() -> Self {
        Self
    }
}

impl SpanProcessor for BaggageSpanProcessor {
    fn on_start(&self, span: &mut opentelemetry_sdk::trace::Span, cx: &opentelemetry::Context) {
        let baggage = cx.baggage();
        for key in &[
            baggage::keys::KIOSK_ID,
            baggage::keys::LANE_ID,
            baggage::keys::TENANT_ID,
            baggage::keys::TRANSACTION_ID,
        ] {
            if let Some(value) = baggage.get(key) {
                span.set_attribute(opentelemetry::KeyValue::new(*key, value.to_string()));
            }
        }
    }

    fn on_end(&self, _span: SpanData) {}

    fn force_flush(&self) -> opentelemetry_sdk::export::trace::ExportResult {
        Ok(())
    }

    fn shutdown(&self) -> opentelemetry_sdk::export::trace::ExportResult {
        Ok(())
    }
}

/// Guard returned by [`init`]. Awaiting it flushes and shuts down the OTel
/// tracer provider.
pub struct ShutdownGuard;

impl ShutdownGuard {
    pub async fn shutdown(self) {
        opentelemetry::global::shutdown_tracer_provider();
    }
}

/// Errors that can occur while initializing telemetry.
#[derive(Debug, thiserror::Error)]
pub enum TelemetryInitError {
    #[error("invalid log filter: {0}")]
    InvalidFilter(String),
    #[error("failed to set global subscriber: {0}")]
    SetSubscriber(String),
    #[error("opentelemetry error: {0}")]
    OpenTelemetry(#[from] opentelemetry::trace::TraceError),
}

/// Operational context fields that should be attached to every log record.
///
/// Create a span with these fields at the start of a request and all events
/// emitted inside the span will inherit them in JSON output.
#[derive(Debug, Clone)]
pub struct TelemetryContext {
    pub trace_id: String,
    pub lane_id: String,
    pub kiosk_id: String,
    pub tenant_id: String,
}

impl TelemetryContext {
    /// Creates a new telemetry context.
    pub fn new(
        trace_id: impl Into<String>,
        lane_id: impl Into<String>,
        kiosk_id: impl Into<String>,
        tenant_id: impl Into<String>,
    ) -> Self {
        Self {
            trace_id: trace_id.into(),
            lane_id: lane_id.into(),
            kiosk_id: kiosk_id.into(),
            tenant_id: tenant_id.into(),
        }
    }

    /// Returns a `tracing` span that carries the context fields.
    ///
    /// The supplied `name` is stored as the `span_name` field; the actual span
    /// name is fixed so that the macro can construct the callsite at compile time.
    pub fn span(&self, _name: &'static str) -> tracing::Span {
        tracing::info_span!(
            "astra.context",
            span_name = _name,
            trace_id = %self.trace_id,
            lane_id = %self.lane_id,
            kiosk_id = %self.kiosk_id,
            tenant_id = %self.tenant_id,
        )
    }
}

/// A writer that redacts sensitive payment and biometric data from log lines.
///
/// Redaction rules:
/// * 13-19 digit sequences (PANs).
/// * Hex strings of length 32, 40, or 64 that may represent biometric hashes.
/// * Common test card prefixes are replaced with `[REDACTED_PAN]`.
#[derive(Clone)]
struct RedactingWriter {
    inner: Arc<std::sync::Mutex<dyn Write + Send + 'static>>,
}

impl RedactingWriter {
    fn new<W: Write + Send + 'static>(inner: W) -> Self {
        Self {
            inner: Arc::new(std::sync::Mutex::new(inner)),
        }
    }
}

impl Write for RedactingWriter {
    fn write(&mut self, buf: &[u8]) -> io::Result<usize> {
        let redacted = redact_sensitive(std::str::from_utf8(buf).unwrap_or(""));
        let mut guard = self
            .inner
            .lock()
            .map_err(|e| io::Error::other(format!("mutex poisoned: {e}")))?;
        guard.write_all(redacted.as_bytes())?;
        Ok(buf.len())
    }

    fn flush(&mut self) -> io::Result<()> {
        let mut guard = self
            .inner
            .lock()
            .map_err(|e| io::Error::other(format!("mutex poisoned: {e}")))?;
        guard.flush()
    }
}

/// Redacts sensitive patterns from a log line.
fn redact_sensitive(input: &str) -> String {
    use regex::Regex;
    use std::sync::OnceLock;

    static PAN_RE: OnceLock<Regex> = OnceLock::new();
    static HASH_RE: OnceLock<Regex> = OnceLock::new();

    let pan_re = PAN_RE.get_or_init(|| Regex::new(r"\b\d{13,19}\b").expect("valid PAN regex"));
    let hash_re = HASH_RE.get_or_init(|| {
        Regex::new(r"\b[0-9a-fA-F]{32}\b|\b[0-9a-fA-F]{40}\b|\b[0-9a-fA-F]{64}\b")
            .expect("valid hash regex")
    });

    let mut out = pan_re.replace_all(input, "[REDACTED_PAN]").to_string();
    out = hash_re.replace_all(&out, "[REDACTED_HASH]").to_string();
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn redacts_pan() {
        let line = "card=4111111111111111 status=ok";
        assert_eq!(redact_sensitive(line), "card=[REDACTED_PAN] status=ok");
    }

    #[test]
    fn redacts_hash() {
        let line = "hash=a3f5c9e2a3f5c9e2a3f5c9e2a3f5c9e2 other";
        assert_eq!(redact_sensitive(line), "hash=[REDACTED_HASH] other");
    }

    #[test]
    fn leaves_safe_text_untouched() {
        let line = "kiosk_id=kiosk-42 amount_cents=1234";
        assert_eq!(redact_sensitive(line), line);
    }
}
