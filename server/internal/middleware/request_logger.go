package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/heywinit/wiant/server/internal/logctx"
)

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
			requestLogger := logger.With(
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"uri", r.RequestURI,
				"host", r.Host,
			)

			ctx := logctx.WithLogger(r.Context(), requestLogger)
			requestLogger.InfoContext(ctx, "request started")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
