package logger

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// HTTPMiddleware создает middleware для логирования HTTP-запросов
func HTTPMiddleware(logger *Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Оборачиваем response writer, чтобы поймать код статуса
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Добавляем логгер в контекст
			ctx := ToContext(r.Context(), logger)

			// Обрабатываем запрос
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Логируем запрос
			logger.WithString("method", r.Method).
				WithString("path", r.URL.Path).
				WithString("remote_addr", r.RemoteAddr).
				WithInt("status", ww.Status()).
				WithInt("bytes", ww.BytesWritten()).
				WithAny("duration", time.Since(start)).
				WithString("user_agent", r.UserAgent()).
				Info("http request")
		})
	}
}
