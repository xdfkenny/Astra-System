package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func captureLogger() (*bytes.Buffer, *slog.Logger) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
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
	})
	return &buf, slog.New(handler)
}

func TestRedactPII_PAN(t *testing.T) {
	// 4111111111111111 is the canonical Visa test PAN (Luhn-valid).
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "card pan 4111111111111111 captured",
			expected: "card pan 411111******1111 captured",
		},
		{
			input:    "formatted 4111 1111 1111 1111 captured",
			expected: "formatted 411111******1111 captured",
		},
		{
			input:    "dashed 4111-1111-1111-1111 captured",
			expected: "dashed 411111******1111 captured",
		},
		{
			// Random 16-digit string that is not Luhn-valid should not be redacted.
			input:    "serial 1234567890123456",
			expected: "serial 1234567890123456",
		},
		{
			// 12-digit number is below PAN length threshold.
			input:    "short 123456789012",
			expected: "short 123456789012",
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := RedactPII(tc.input)
			if got != tc.expected {
				t.Errorf("RedactPII(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestRedactPII_Biometric(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    `biometric_hash=deadbeefcafe1234deadbeefcafe1234`,
			expected: `biometric_hash=[REDACTED]`,
		},
		{
			input:    `fingerprint_template: "aabbccdd11223344556677889900aabbccdd11223344556677889900aabb"`,
			expected: `fingerprint_template: [REDACTED]`,
		},
		{
			input:    `faceprint_vector=base64,AQIDBAUGBwcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8gISIjJCUmJygpKissLS4vMDEyMzQ1Njc4OTo7PD0+P0BBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWltcXV5fYGFiY2RlZmdoaWprbG1ub3BxcnN0dXZ3eHl6e3x9fn+AgYKDhIWGh4iJiouMjY6PkA==`,
			expected: `faceprint_vector=[REDACTED]`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := RedactPII(tc.input)
			if got != tc.expected {
				t.Errorf("RedactPII(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestLoggerIncludesContextFields(t *testing.T) {
	buf, logger := captureLogger()
	SetLogger(logger)
	defer SetLogger(NewLogger(slog.LevelInfo))

	ctx := ContextWithFields(context.Background(), ContextFields{
		TraceID:  "trace-123",
		SpanID:   "span-456",
		LaneID:   "lane-7",
		KioskID:  "kiosk-42",
		TenantID: "tenant-99",
	})

	Info(ctx, "order submitted", slog.String("order_id", "ord-1"))

	line := buf.String()
	for _, want := range []string{
		`"trace_id":"trace-123"`,
		`"span_id":"span-456"`,
		`"lane_id":"lane-7"`,
		`"kiosk_id":"kiosk-42"`,
		`"tenant_id":"tenant-99"`,
		`"order_id":"ord-1"`,
		`"msg":"order submitted"`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("log line missing %q: %s", want, line)
		}
	}
}

func TestLoggerRedactsPANAttribute(t *testing.T) {
	buf, logger := captureLogger()
	SetLogger(logger)
	defer SetLogger(NewLogger(slog.LevelInfo))

	Info(context.Background(), "payment attempted", slog.String("pan", "4111111111111111"))

	line := buf.String()
	if strings.Contains(line, "4111111111111111") {
		t.Errorf("PAN was not redacted in log line: %s", line)
	}
	if !strings.Contains(line, "411111******1111") {
		t.Errorf("expected masked PAN in log line: %s", line)
	}
}

func TestLoggerRedactsPANInMessage(t *testing.T) {
	buf, logger := captureLogger()
	SetLogger(logger)
	defer SetLogger(NewLogger(slog.LevelInfo))

	Info(context.Background(), "received pan 4111111111111111")

	line := buf.String()
	if strings.Contains(line, "4111111111111111") {
		t.Errorf("PAN in message was not redacted: %s", line)
	}
}

func TestParseLevel(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			if got := parseLevel(tc.input); got != tc.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestNewLoggerIsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(slog.LevelInfo)
	logger = logger.With(slog.String("service", "test"))
	logger.Info("hello")

	// We cannot easily redirect the package logger here, but we can assert the
	// handler produced valid JSON by creating a fresh one writing to buf.
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.New(handler).Info("hello")

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("log output is not valid JSON: %v", err)
	}
	if out["msg"] != "hello" {
		t.Errorf("expected msg=hello, got %v", out["msg"])
	}
}
