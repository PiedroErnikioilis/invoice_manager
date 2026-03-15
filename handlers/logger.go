package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logger returns a middleware that logs HTTP requests using slog.
func Logger(l *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			t1 := time.Now()
			defer func() {
				t2 := time.Now()
				
				status := ww.Status()
				attrs := []slog.Attr{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.Duration("duration", t2.Sub(t1)),
					slog.String("ip", r.RemoteAddr),
				}

				if status >= 400 {
					l.LogAttrs(r.Context(), slog.LevelError, "request failed", attrs...)
				} else {
					l.LogAttrs(r.Context(), slog.LevelInfo, "request processed", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}
