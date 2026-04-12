package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		start := time.Now()
		next.ServeHTTP(recorder, r)
		latency := time.Since(start)

		switch {
		case recorder.status >= http.StatusInternalServerError:
			slog.Error("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "latency", latency)
		case recorder.status >= http.StatusBadRequest:
			slog.Warn("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "latency", latency)
		default:
			slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "latency", latency)
		}
	})
}
