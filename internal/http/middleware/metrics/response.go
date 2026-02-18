package metrics

import "net/http"

// responseWriter wraps http.ResponseWriter to capture status code.
// If WriteHeader is never called, status is 200 (net/http default).
type responseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wrote {
		rw.wrote = true
		rw.status = code
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Status() int {
	if !rw.wrote {
		return http.StatusOK
	}
	return rw.status
}
