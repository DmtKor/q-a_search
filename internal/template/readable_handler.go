package template

import (
	"encoding/json"
	"net/http"
)

// ReadableRequest is the body for template-readable (template only).
type ReadableRequest struct {
	Template string `json:"template"`
}

// ReadableResponse is the response (list of segments for human-readable view).
type ReadableResponse struct {
	Segments []ReadableSegment `json:"segments"`
}

// ReadableHandler handles POST with ReadableRequest and returns ReadableResponse.
// Use RequireStaff when mounting.
type ReadableHandler struct{}

// ServeHTTP expects JSON body { "template": "..." }, returns { "segments": [...] }.
func (ReadableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":{"code":"method_not_allowed","message":"Method not allowed","details":null}}`))
		return
	}

	var req ReadableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"validation_error","message":"Invalid JSON body","details":null}}`))
		return
	}

	segments := ToReadableSegments(req.Template)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ReadableResponse{Segments: segments})
}
