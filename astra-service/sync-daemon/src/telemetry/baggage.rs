//! OpenTelemetry baggage propagation for the libp2p mesh.
//!
//! Serializes W3C Baggage context into the `SyncMessage` envelope so that
//! distributed traces can span the Kiosk UI → Rust daemon → Cloud Tier
//! boundary.  Baggage key/value pairs are carried as a wire-format header
//! string inside each P2P message.
//!
//! On the receiving side the baggage is rehydrated into the tracing span
//! context so that downstream OTLP exports include the original trace
//! attributes (kiosk_id, lane_id, tenant_id, transaction_id).

use opentelemetry::baggage::BaggageExt;
use opentelemetry::Context;
use tracing::warn;

/// Well-known baggage keys used across the Astra mesh.
pub mod keys {
    /// The kiosk that originated the request.
    pub const KIOSK_ID: &str = "astra.kiosk_id";
    /// The lane/checkout-lane identifier.
    pub const LANE_ID: &str = "astra.lane_id";
    /// The tenant (retailer) identifier.
    pub const TENANT_ID: &str = "astra.tenant_id";
    /// The transaction identifier.
    pub const TRANSACTION_ID: &str = "astra.transaction_id";
    /// The upstream trace ID (W3C Trace-Context trace-id).
    pub const TRACE_ID: &str = "astra.trace_id";
}

/// Maximum wire length for a serialised baggage header (4 KiB).
pub const MAX_BAGGAGE_BYTES: usize = 4096;

/// Encodes the current OTel context's baggage into a wire string suitable
/// for embedding in a `SyncMessage`.
///
/// The resulting string is in W3C Baggage format:
///   `key1=val1,key2=val2`
/// Values are URL-encoded per the spec.
pub fn encode_baggage(ctx: &Context) -> String {
    let baggage = ctx.baggage();
    if baggage.len() == 0 {
        return String::new();
    }

    let entries: Vec<String> = baggage
        .iter()
        .map(|(k, (v, metadata))| {
            let key = percent_encode(k.as_str());
            let value = percent_encode(&v.to_string());
            if !metadata.as_str().is_empty() {
                format!("{}={};{}", key, value, metadata.as_str())
            } else {
                format!("{}={}", key, value)
            }
        })
        .collect();

    let wire = entries.join(",");
    if wire.len() > MAX_BAGGAGE_BYTES {
        warn!(
            len = wire.len(),
            max = MAX_BAGGAGE_BYTES,
            "Baggage header exceeds maximum size; truncating"
        );
        wire[..MAX_BAGGAGE_BYTES].to_string()
    } else {
        wire
    }
}

/// Decodes a W3C Baggage wire string into entries and attaches them to the
/// given OTel context, returning the updated context.
///
/// Calling this on the receiving side of a P2P message ensures that spans
/// created during message processing inherit the origin's trace attributes.
pub fn decode_baggage(ctx: &Context, wire: &str) -> Context {
    if wire.is_empty() {
        return ctx.clone();
    }

    use opentelemetry::KeyValue;
    let mut entries: Vec<KeyValue> = Vec::new();
    for pair in wire.split(',') {
        let pair = pair.trim();
        if pair.is_empty() {
            continue;
        }
        // Each pair: key=value or key=value;metadata
        let (kv, _metadata_str) = match pair.split_once(';') {
            Some((kv, _meta)) => (kv, None::<&str>),
            None => (pair, None),
        };
        let (key, value) = match kv.split_once('=') {
            Some((k, v)) => (percent_decode(k), percent_decode(v)),
            None => continue,
        };
        entries.push(KeyValue::new(key, value));
    }

    ctx.with_baggage(entries)
}

/// Helper: attach well-known Astra baggage entries from a `TelemetryContext`.
pub fn inject_telemetry_context(
    ctx: &Context,
    telemetry_ctx: &crate::telemetry::TelemetryContext,
) -> Context {
    use opentelemetry::KeyValue;
    let mut entries: Vec<KeyValue> = ctx
        .baggage()
        .iter()
        .map(|(k, (v, _))| KeyValue::new(k.as_str().to_string(), v.to_string()))
        .collect();
    entries.push(KeyValue::new(
        keys::KIOSK_ID,
        telemetry_ctx.kiosk_id.clone(),
    ));
    entries.push(KeyValue::new(keys::LANE_ID, telemetry_ctx.lane_id.clone()));
    entries.push(KeyValue::new(
        keys::TENANT_ID,
        telemetry_ctx.tenant_id.clone(),
    ));
    entries.push(KeyValue::new(
        keys::TRACE_ID,
        telemetry_ctx.trace_id.clone(),
    ));
    ctx.with_baggage(entries)
}

/// Extracts key Astra fields from baggage for logging / span attribution.
pub fn extract_telemetry_context(ctx: &Context) -> Option<crate::telemetry::TelemetryContext> {
    let baggage = ctx.baggage();
    let kiosk_id = baggage.get(keys::KIOSK_ID).map(|v| v.to_string());
    let lane_id = baggage.get(keys::LANE_ID).map(|v| v.to_string());
    let tenant_id = baggage.get(keys::TENANT_ID).map(|v| v.to_string());
    let trace_id = baggage.get(keys::TRACE_ID).map(|v| v.to_string());

    match (kiosk_id, lane_id, tenant_id, trace_id) {
        (Some(k), Some(l), Some(t), Some(tr)) => {
            Some(crate::telemetry::TelemetryContext::new(tr, l, k, t))
        }
        _ => None,
    }
}

/// Minimal percent-encoding for Baggage value characters that are not
/// allowed unencoded per W3C spec: commas, semicolons, equals, spaces.
fn percent_encode(s: &str) -> String {
    s.replace('%', "%25")
        .replace(',', "%2C")
        .replace(';', "%3B")
        .replace('=', "%3D")
        .replace(' ', "%20")
}

/// Minimal percent-decoding.
fn percent_decode(s: &str) -> String {
    let mut out = String::with_capacity(s.len());
    let mut chars = s.chars();
    while let Some(c) = chars.next() {
        if c == '%' {
            let hex: String = chars.by_ref().take(2).collect();
            if let Ok(byte) = u8::from_str_radix(&hex, 16) {
                out.push(byte as char);
            } else {
                out.push('%');
                out.push_str(&hex);
            }
        } else {
            out.push(c);
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use opentelemetry::{baggage::BaggageExt, Context};

    #[test]
    fn roundtrip_baggage() {
        let ctx = Context::current();
        let mut baggage = opentelemetry::baggage::Baggage::new();
        baggage.insert(keys::KIOSK_ID, "kiosk-42");
        baggage.insert(keys::LANE_ID, "L7");
        let ctx = ctx.with_baggage(baggage);

        let wire = encode_baggage(&ctx);
        assert!(!wire.is_empty());

        let restored_ctx = decode_baggage(&Context::current(), &wire);
        let restored_baggage = restored_ctx.baggage();
        assert_eq!(
            restored_baggage.get(keys::KIOSK_ID).map(|v| v.to_string()),
            Some("kiosk-42".to_string())
        );
        assert_eq!(
            restored_baggage.get(keys::LANE_ID).map(|v| v.to_string()),
            Some("L7".to_string())
        );
    }

    #[test]
    fn empty_baggage_produces_empty_string() {
        let ctx = Context::current();
        assert_eq!(encode_baggage(&ctx), "");
    }

    #[test]
    fn decode_empty_string_is_noop() {
        let ctx = Context::current();
        let restored = decode_baggage(&ctx, "");
        assert_eq!(restored.baggage().len(), 0);
    }

    #[test]
    fn percent_encoding_handles_special_chars() {
        let ctx = Context::current();
        let mut baggage = opentelemetry::baggage::Baggage::new();
        baggage.insert("key", "val,ue;foo=bar");
        let ctx = ctx.with_baggage(baggage);

        let wire = encode_baggage(&ctx);
        assert!(!wire.contains(',')); // raw comma would break W3C format
        assert!(!wire.contains('='));
        assert!(!wire.contains(';'));

        let restored = decode_baggage(&Context::current(), &wire);
        assert_eq!(
            restored.baggage().get("key").map(|v| v.to_string()),
            Some("val,ue;foo=bar".to_string())
        );
    }
}
