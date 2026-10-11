package apikit

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
)

// Logger defines the interface for logging errors in generated handlers.
// Implementations should follow the slog-style key-value pattern for structured logging.
type Logger interface {
	Error(ctx context.Context, msg string, args ...any)
}

// DebugLogger is a Logger that can also write below error. When the registered
// logger implements it, a request refused for the client's own fault — a body
// that does not parse, a field that fails validation, a handler error that
// renders as 4xx — is written at debug. A Logger without it does not see those
// requests at all: they are not the process failing, and the access log
// already has them with their status.
type DebugLogger interface {
	Logger
	Debug(ctx context.Context, msg string, args ...any)
}

// noopLogger is the default logger that discards all log messages.
type noopLogger struct{}

func (noopLogger) Error(context.Context, string, ...any) {}

// loggerBox lets an interface value live in an atomic.Pointer.
type loggerBox struct{ Logger }

// globalLogger is the package-level logger; empty means the no-op default. It
// is read on every request and may be replaced while requests are in flight — a
// test that installs its own, for one — so it is swapped atomically.
var globalLogger atomic.Pointer[loggerBox]

// SetLogger overrides the global logger used by all generated handlers. A nil
// logger restores the default, which discards everything.
func SetLogger(l Logger) {
	if l == nil {
		l = noopLogger{}
	}
	globalLogger.Store(&loggerBox{l})
}

// GetLogger returns the current global logger.
func GetLogger() Logger {
	if b := globalLogger.Load(); b != nil {
		return b.Logger
	}
	return noopLogger{}
}

// LoggerEnabled returns true if a real (non-noop) logger has been set.
func LoggerEnabled() bool {
	_, noop := GetLogger().(noopLogger)
	return !noop
}

// NewSlogLogger adapts a *slog.Logger to Logger and DebugLogger, keeping the
// request context so a handler that reads trace or request identifiers from it
// still sees them.
func NewSlogLogger(l *slog.Logger) DebugLogger {
	return slogLogger{l: l}
}

type slogLogger struct{ l *slog.Logger }

func (s slogLogger) Error(ctx context.Context, msg string, args ...any) {
	s.l.ErrorContext(ctx, msg, args...)
}

func (s slogLogger) Debug(ctx context.Context, msg string, args ...any) {
	s.l.DebugContext(ctx, msg, args...)
}

// LogFailure records a request a generated handler could not serve: the body
// did not parse, validation refused it, or the handler returned err. It names
// the handler, the method and the path; it does not record the payload, which
// is what the client sent and may carry anything — an address, a document — a
// log is no place for.
//
// The level follows whose fault it was. An error that renders as 4xx is the
// client's, and goes to debug when the logger has one (see DebugLogger); any
// other is the process's, and goes to error.
func LogFailure(r *http.Request, handler, msg string, err error) {
	ctx := r.Context()
	args := []any{"handler", handler, "method", r.Method, "path", r.URL.Path, "error", err}
	l := GetLogger()
	if statusOf(err) < http.StatusInternalServerError {
		if d, ok := l.(DebugLogger); ok {
			d.Debug(ctx, msg, args...)
		}
		return
	}
	l.Error(ctx, msg, args...)
}

// statusOf returns the status an error declares through StatusCode, or 500.
func statusOf(err error) int {
	var sc statusCoder
	if errors.As(err, &sc) {
		if code := sc.StatusCode(); code >= 100 && code <= 999 {
			return code
		}
	}
	return http.StatusInternalServerError
}
