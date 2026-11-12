package handler

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func requestLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			requestID := chimiddleware.GetReqID(r.Context())
			writer.Header().Set("X-Request-ID", requestID)

			defer func() {
				status := writer.Status()
				if status == 0 {
					status = http.StatusOK
				}

				logger.InfoContext(
					r.Context(),
					"http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", requestID),
				)
			}()

			next.ServeHTTP(writer, r)
		})
	}
}

func recoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(
						r.Context(),
						"request panic",
						slog.Any("panic", recovered),
						slog.String("stack", string(debug.Stack())),
						slog.String(
							"request_id",
							chimiddleware.GetReqID(r.Context()),
						),
					)

					if writer.Status() == 0 {
						writeJSON(writer, http.StatusInternalServerError, errorResponse{
							Error: "internal server error",
							Code:  "INTERNAL_ERROR",
						})
					}
				}
			}()

			next.ServeHTTP(writer, r)
		})
	}
}
