package requestlog

import (
	"log"
	"net/http"
	"time"
)

// Middleware logs each request and response. Level: "none" (no log), "minimal" (method, path, status, duration), "detailed" (+ sizes).
func Middleware(level string, next http.Handler) http.Handler {
	if level == "none" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		dur := time.Since(start).Milliseconds()
		if level == "minimal" {
			log.Printf("[request] %s %s %d %dms", r.Method, r.URL.Path, rw.status, dur)
			return
		}
		if level == "detailed" {
			reqLen := r.Header.Get("Content-Length")
			if reqLen == "" {
				reqLen = "-"
			}
			log.Printf("[request] %s %s %d %dms req_len=%s res_len=%d", r.Method, r.URL.Path, rw.status, dur, reqLen, rw.written)
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
	written int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(p []byte) (n int, err error) {
	n, err = w.ResponseWriter.Write(p)
	w.written += int64(n)
	return n, err
}
