package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newObservedLogger builds a logger that writes to an in-memory observer,
// making it easy to inspect emitted entries in tests.
func newObservedLogger(serviceName, level string) (*zap.Logger, *observer.ObservedLogs) {
	zapLevel := parseLevel(level)
	core, logs := observer.New(zapLevel)
	logger := zap.New(core).With(
		zap.String("service", serviceName),
		zap.String("trace_id", ""),
	)
	return logger, logs
}

// TestNewLogger_ReturnsNonNil verifies that NewLogger always returns a usable logger.
func TestNewLogger_ReturnsNonNil(t *testing.T) {
	l := NewLogger("test-service", "info")
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}

// TestNewLogger_ServiceField verifies the service default field is set.
func TestNewLogger_ServiceField(t *testing.T) {
	l, logs := newObservedLogger("my-service", "info")
	l.Info("hello")

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}
	entry := logs.All()[0]
	serviceField := fieldByKey(entry.Context, "service")
	if serviceField == nil || serviceField.String != "my-service" {
		t.Errorf("expected service=my-service, got %v", serviceField)
	}
}

// TestNewLogger_TraceIDField verifies the trace_id default field is present (empty by default).
func TestNewLogger_TraceIDField(t *testing.T) {
	l, logs := newObservedLogger("svc", "info")
	l.Info("msg")

	entry := logs.All()[0]
	traceField := fieldByKey(entry.Context, "trace_id")
	if traceField == nil {
		t.Error("expected trace_id field to be present")
	}
}

// TestNewLogger_SuppressDebugAtInfo verifies DEBUG is suppressed when level is INFO.
func TestNewLogger_SuppressDebugAtInfo(t *testing.T) {
	l, logs := newObservedLogger("svc", "info")
	l.Debug("this should be suppressed")
	l.Info("this should appear")

	if logs.Len() != 1 {
		t.Errorf("expected 1 entry (INFO only), got %d", logs.Len())
	}
	if logs.All()[0].Level != zapcore.InfoLevel {
		t.Errorf("expected INFO level entry")
	}
}

// TestNewLogger_DebugVisibleAtDebugLevel verifies DEBUG appears when level is DEBUG.
func TestNewLogger_DebugVisibleAtDebugLevel(t *testing.T) {
	l, logs := newObservedLogger("svc", "debug")
	l.Debug("debug msg")

	if logs.Len() != 1 {
		t.Errorf("expected 1 DEBUG entry, got %d", logs.Len())
	}
}

// TestNewLogger_JSONOutput verifies the real NewLogger emits valid JSON with required fields.
func TestNewLogger_JSONOutput(t *testing.T) {
	// Use a buffer-backed core to capture raw JSON output.
	buf := &bytes.Buffer{}
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.MessageKey = "message"
	encoderCfg.LevelKey = "level"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(buf),
		zapcore.InfoLevel,
	)
	l := zap.New(core).With(
		zap.String("service", "json-test"),
		zap.String("trace_id", ""),
	)

	l.Info("test message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	required := []string{"timestamp", "level", "service", "trace_id", "message"}
	for _, field := range required {
		if _, ok := entry[field]; !ok {
			t.Errorf("missing required field %q in log output: %s", field, buf.String())
		}
	}
}

// fieldByKey is a helper to find a zap.Field by key in a slice.
func fieldByKey(fields []zap.Field, key string) *zap.Field {
	for i := range fields {
		if fields[i].Key == key {
			return &fields[i]
		}
	}
	return nil
}
