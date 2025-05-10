package stremio

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// NewLogger creates a new logger with the specified level and encoding.
// The level can be "debug", "info", "warn", or "error".
// The encoding can be "json" or "console".
func NewLogger(level string, encoding string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	if encoding == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		})
	}

	return slog.New(handler)
}

// createLoggingMiddleware creates a middleware that logs requests.
func createLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create request-specific fields
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			}

			// Log request start
			logger.Info("Request started", attrs...)

			// Create response writer that captures status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Handle request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log request completion
			attrs = append(attrs,
				"status", rw.statusCode,
				"duration", duration,
			)
			logger.Info("Request completed", attrs...)
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// createHealthHandler creates a handler for health check requests.
func createHealthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("Health check request received")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}
