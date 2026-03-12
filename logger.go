package apikit

import "context"

// Logger defines the interface for logging errors in generated handlers.
// Implementations should follow the slog-style key-value pattern for structured logging.
type Logger interface {
	Error(ctx context.Context, msg string, args ...any)
}

// noopLogger is the default logger that discards all log messages.
type noopLogger struct{}

func (noopLogger) Error(context.Context, string, ...any) {}

// globalLogger is the package-level logger, defaults to no-op.
var globalLogger Logger = noopLogger{}

// SetLogger overrides the global logger used by all generated handlers.
func SetLogger(l Logger) {
	globalLogger = l
}

// GetLogger returns the current global logger.
func GetLogger() Logger {
	return globalLogger
}
