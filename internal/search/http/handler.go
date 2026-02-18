package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	"github.com/yourusername/project/internal/search"
)

// Handler handles POST /api/v1/search. Expects principal in context (use after Authenticate + RequireAppOrStaff).
type Handler struct {
	Service *search.Service
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		authmw.WriteError(w, "validation_error", "Method not allowed", http.StatusMethodNotAllowed, nil)
		return
	}

	var req search.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}

	if req.Query == "" {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "query", "reason": "required",
		})
		return
	}

	if req.TopK != nil && (*req.TopK < 1 || *req.TopK > 50) {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "top_k", "reason": "must be between 1 and 50",
		})
		return
	}

	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		authmw.Write403(w, "Access denied")
		return
	}

	resp, err := h.Service.Search(r.Context(), req, principal)
	if err != nil {
		if errors.Is(err, search.ErrInvalidTopK) {
			authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
				"field": "top_k", "reason": "must be between 1 and 50",
			})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
