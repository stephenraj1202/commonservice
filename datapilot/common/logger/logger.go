package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a production JSON logger with the given service name and log level.
// Every log entry includes timestamp, level, service, trace_id, and message fields.
// DEBUG entries are suppressed when level is "info", "warn", or "error".
func NewLogger(serviceName, level string) *zap.Logger {
	zapLevel := parseLevel(level)

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)

	// Use "timestamp" key instead of default "ts" to match the required field name.
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Use "message" key instead of default "msg".
	cfg.EncoderConfig.MessageKey = "message"

	// Use "level" key (already the default, but be explicit).
	cfg.EncoderConfig.LevelKey = "level"

	logger, err := cfg.Build()
	if err != nil {
		// Fallback to a no-op logger if build fails (should never happen in practice).
		return zap.NewNop()
	}

	// Add default fields: service and trace_id.
	// trace_id is empty by default; middleware will inject it per-request via With().
	logger = logger.With(
		zap.String("service", serviceName),
		zap.String("trace_id", ""),
	)

	return logger
}

// parseLevel converts a string log level to a zapcore.Level.
// Defaults to InfoLevel for unrecognised values.
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug", "DEBUG":
		return zapcore.DebugLevel
	case "warn", "WARN", "warning", "WARNING":
		return zapcore.WarnLevel
	case "error", "ERROR":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
