// Package observability provides production-grade observability primitives
// (structured logging, distributed tracing, Prometheus metrics, and health
// probes) shared by every Astra-Service backend microservice.
package observability

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// contextKey is a private type used to avoid collisions with other context keys.
type contextKey struct{}

var fieldsKey = &contextKey{}

// ContextFields are the observability dimensions that propagate alongside a
// request context. They are emitted on every log line and attached to traces
// so a single kiosk transaction can be followed end-to-end.
type ContextFields struct {
	TraceID  string
	SpanID   string
	LaneID   string
	KioskID  string
	TenantID string
}

// ContextWithFields returns a context enriched with observability dimensions.
func ContextWithFields(ctx context.Context, f ContextFields) context.Context {
	return context.WithValue(ctx, fieldsKey, f)
}

// ExtractFields pulls observability dimensions from the context. If an
// OpenTelemetry span is active, trace_id and span_id are derived from it and
// merged with any explicitly stored fields.
func ExtractFields(ctx context.Context) ContextFields {
	var f ContextFields
	if v, ok := ctx.Value(fieldsKey).(ContextFields); ok {
		f = v
	}
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		sc := span.SpanContext()
		f.TraceID = sc.TraceID().String()
		f.SpanID = sc.SpanID().String()
	}
	return f
}

var (
	defaultLogger *slog.Logger
	loggerOnce    sync.Once
)

// Logger returns the package-default structured JSON logger. The logger
// redacts PANs and biometric identifiers from string values and messages.
func Logger() *slog.Logger {
	loggerOnce.Do(func() {
		if defaultLogger == nil {
			level := parseLevel(os.Getenv("ASTRA_LOG_LEVEL"))
			defaultLogger = NewLogger(level)
		}
	})
	return defaultLogger
}

// SetLogger replaces the package-default logger. Used by tests and by
// services that need a custom handler.
func SetLogger(l *slog.Logger) {
	defaultLogger = l
}

// NewLogger builds a JSON slog.Handler that redacts PII from every string
// attribute and from the message itself.
func NewLogger(level slog.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Value = slog.StringValue(RedactPII(a.Value.String()))
				return a
			}
			if a.Value.Kind() == slog.KindString {
				a.Value = slog.StringValue(RedactPII(a.Value.String()))
			}
			return a
		},
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func attrsFromContext(ctx context.Context) []slog.Attr {
	f := ExtractFields(ctx)
	var attrs []slog.Attr
	if f.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", f.TraceID))
	}
	if f.SpanID != "" {
		attrs = append(attrs, slog.String("span_id", f.SpanID))
	}
	if f.LaneID != "" {
		attrs = append(attrs, slog.String("lane_id", f.LaneID))
	}
	if f.KioskID != "" {
		attrs = append(attrs, slog.String("kiosk_id", f.KioskID))
	}
	if f.TenantID != "" {
		attrs = append(attrs, slog.String("tenant_id", f.TenantID))
	}
	return attrs
}

// Info logs an info-level message with context-derived observability fields.
func Info(ctx context.Context, msg string, attrs ...slog.Attr) {
	Logger().LogAttrs(ctx, slog.LevelInfo, msg, append(attrsFromContext(ctx), attrs...)...)
}

// Warn logs a warning-level message with context-derived observability fields.
func Warn(ctx context.Context, msg string, attrs ...slog.Attr) {
	Logger().LogAttrs(ctx, slog.LevelWarn, msg, append(attrsFromContext(ctx), attrs...)...)
}

// Error logs an error-level message with context-derived observability fields.
func Error(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	merged := append(attrsFromContext(ctx), slog.String("error", err.Error()))
	merged = append(merged, attrs...)
	Logger().LogAttrs(ctx, slog.LevelError, msg, merged...)
}

// Debug logs a debug-level message with context-derived observability fields.
func Debug(ctx context.Context, msg string, attrs ...slog.Attr) {
	Logger().LogAttrs(ctx, slog.LevelDebug, msg, append(attrsFromContext(ctx), attrs...)...)
}

// PII redaction patterns. PANs are matched by digit runs of 13-19 characters
// and validated with the Luhn checksum to reduce false positives. Biometric
// identifiers are matched by common key names followed by a hash/template.
var (
	// panRegex matches 13-19 digit runs that may include spaces or dashes.
	panRegex = regexp.MustCompile(`\b(?:\d[\s-]*?){13,19}\b`)

	// biometricRegex matches key-value pairs for biometric hashes/templates.
	biometricRegex = regexp.MustCompile(`(?i)("?(?:biometric|fingerprint|faceprint|iris|vein|face|palm)_?(?:hash|template|signature|vector|feature|print|data|token)"?\s*[:=]\s*)["']?(?:[a-f0-9]{32,128}|base64,[A-Za-z0-9+/=]{40,})["']?`)
)

// RedactPII removes or masks PANs and biometric identifiers from a string.
func RedactPII(input string) string {
	out := panRegex.ReplaceAllStringFunc(input, func(match string) string {
		digits := digitsOnly(match)
		if len(digits) >= 13 && len(digits) <= 19 && luhnCheck(digits) {
			return maskPAN(digits)
		}
		return match
	})
	out = biometricRegex.ReplaceAllString(out, `${1}[REDACTED]`)
	return out
}

func digitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// luhnCheck validates a digit string using the Luhn algorithm.
func luhnCheck(digits string) bool {
	n := len(digits)
	if n < 13 {
		return false
	}
	sum := 0
	double := false
	for i := n - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

// maskPAN keeps the first six (BIN) and last four digits, masking the rest.
func maskPAN(pan string) string {
	if len(pan) <= 10 {
		return strings.Repeat("*", len(pan))
	}
	return pan[:6] + strings.Repeat("*", len(pan)-10) + pan[len(pan)-4:]
}

func parseLevel(v string) slog.Level {
	switch strings.ToLower(v) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
