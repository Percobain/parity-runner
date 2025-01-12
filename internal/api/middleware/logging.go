package middleware

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/virajbhartiya/parity-protocol/pkg/logger"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a logger with request context
		log := logger.Get().With().
			Str("request_id", uuid.New().String()).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Str("remote_addr", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Str("referer", r.Referer()).
			Logger()

		// Log the request
		log.Info().Msg("Request started")

		// Create response wrapper to capture status code
		ww := &responseWriter{w: w, status: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(ww, r)

		// Log the response
		duration := time.Since(start)
		log.Info().
			Int("status", ww.status).
			Dur("duration", duration).
			Str("duration_human", duration.String()).
			Msg("Request completed")
	})
}

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	w      http.ResponseWriter
	status int
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.status = statusCode
	rw.w.WriteHeader(statusCode)
}
