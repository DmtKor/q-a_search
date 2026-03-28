package template

import (
	"encoding/json"
	"net/http"
)

// PreviewRequest is the body for render-preview (template + user_context).
type PreviewRequest struct {
	Template    string                 `json:"template"`
	UserContext map[string]interface{} `json:"user_context"`
}

// PreviewResponse is the response body (rendered text).
type PreviewResponse struct {
	Text string `json:"text"`
}

// PreviewHandler handles POST with PreviewRequest and returns PreviewResponse.
// Use RequireStaff when mounting. Renderer must not be nil.
type PreviewHandler struct {
	Renderer *Renderer
}

// ServeHTTP expects JSON body { "template": "...", "user_context": {...} }, returns { "text": "..." }.
func (h *PreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"error":{"code":"method_not_allowed","message":"Method not allowed","details":null}}`))
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"validation_error","message":"Invalid JSON body","details":null}}`))
		return
	}

	text, err := h.Renderer.Render(r.Context(), req.Template, req.UserContext)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		body, _ := json.Marshal(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "template_error",
				"message": err.Error(),
				"details": nil,
			},
		})
		_, _ = w.Write(body)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := PreviewResponse{Text: text}
	_ = json.NewEncoder(w).Encode(resp)
}
