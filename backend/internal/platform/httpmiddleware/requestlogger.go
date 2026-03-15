package httpmiddleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// RequestLogger writes one structured log line per HTTP request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			wrapped := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

			next.ServeHTTP(wrapped, request)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}

			args := []any{
				"request_id", middleware.GetReqID(request.Context()),
				"method", request.Method,
				"path", request.URL.Path,
				"query", request.URL.RawQuery,
				"status", status,
				"bytes", wrapped.BytesWritten(),
				"duration_ms", time.Since(started).Milliseconds(),
				"remote_addr", request.RemoteAddr,
				"user_agent", request.UserAgent(),
			}

			switch {
			case status >= http.StatusInternalServerError:
				logger.Error("http request", args...)
			case status >= http.StatusBadRequest:
				logger.Warn("http request", args...)
			default:
				logger.Info("http request", args...)
			}
		})
	}
}
