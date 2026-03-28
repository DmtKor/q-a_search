package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	"github.com/yourusername/project/internal/cases"
)

// Handler handles /api/v1/cases (list, create, get, update, delete, status, import, export).
// Mount at /api/v1/cases; path is relative to that (e.g. "", "import", "export", "{id}", "{id}/status").
// Expects RequireStaff and principal in context.
type Handler struct {
	Service cases.CaseService
}

// CategoriesHandler serves GET only and returns list of categories. Use with RequireAppOrStaff so search form can load categories with app token.
type CategoriesHandler struct {
	Service cases.CaseService
}

// ServeHTTP responds to GET with {"categories": ["a", "b", ...]}.
func (h *CategoriesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		authmw.WriteError(w, "validation_error", "Method not allowed", http.StatusMethodNotAllowed, nil)
		return
	}
	list, err := h.Service.ListCategories(r.Context())
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"categories": list})
}

// ServeHTTP dispatches by method and path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cases")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	principal := auth.PrincipalFromContext(r.Context())
	if principal == nil {
		authmw.Write403(w, "Access denied")
		return
	}

	switch {
	case path == "" || path == "/":
		if r.Method == http.MethodGet {
			h.List(w, r, principal)
			return
		}
		if r.Method == http.MethodPost {
			h.Create(w, r, principal)
			return
		}
	case len(parts) == 1 && parts[0] == "categories":
		if r.Method == http.MethodGet {
			h.ListCategories(w, r)
			return
		}
	case len(parts) == 1 && parts[0] == "import":
		if r.Method == http.MethodPost {
			h.Import(w, r, principal)
			return
		}
	case len(parts) == 1 && parts[0] == "export":
		if r.Method == http.MethodGet {
			h.Export(w, r, principal)
			return
		}
	case len(parts) == 1 && isUUID(parts[0]):
		id := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.Get(w, r, id, principal)
			return
		case http.MethodPut:
			h.Update(w, r, id, principal)
			return
		case http.MethodDelete:
			h.Delete(w, r, id, principal)
			return
		}
	case len(parts) == 2 && isUUID(parts[0]) && parts[1] == "status":
		if r.Method == http.MethodPost {
			h.ChangeStatus(w, r, parts[0], principal)
			return
		}
	}

	authmw.WriteError(w, "validation_error", "Method not allowed", http.StatusMethodNotAllowed, nil)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request, principal *auth.Principal) {
	filters := cases.ListFilters{
		Status:   r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"),
		Mine:     r.URL.Query().Get("mine") == "true" || r.URL.Query().Get("mine") == "1",
	}
	list, err := h.Service.List(r.Context(), filters, principal)
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.ListCategories(r.Context())
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"categories": list})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request, principal *auth.Principal) {
	var body cases.CaseCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	c, err := h.Service.Create(r.Context(), body, principal.TokenID)
	if err != nil {
		if errors.Is(err, cases.ErrValidation) {
			authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, id string, principal *auth.Principal) {
	c, err := h.Service.Get(r.Context(), id, principal)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Case not found", http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, cases.ErrForbidden) {
			authmw.Write403(w, "Access denied")
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id string, principal *auth.Principal) {
	var body cases.CaseUpdate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	// Ignore status if sent in body (PUT cannot change status)
	c, err := h.Service.Update(r.Context(), id, body, principal)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Case not found", http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, cases.ErrForbidden) {
			authmw.Write403(w, "Access denied")
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, id string, principal *auth.Principal) {
	err := h.Service.Delete(r.Context(), id, principal)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Case not found", http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, cases.ErrForbidden) {
			authmw.Write403(w, "Access denied")
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request, id string, principal *auth.Principal) {
	var req cases.StatusChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	if req.Status == "" {
		authmw.WriteError(w, "validation_error", "status is required", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "status", "reason": "required",
		})
		return
	}
	c, err := h.Service.ChangeStatus(r.Context(), id, req, principal)
	if err != nil {
		if errors.Is(err, cases.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Case not found", http.StatusNotFound, nil)
			return
		}
		if errors.Is(err, cases.ErrForbidden) {
			authmw.Write403(w, "Access denied")
			return
		}
		if errors.Is(err, cases.ErrInvalidStatus) {
			authmw.WriteError(w, "validation_error", "Invalid status transition", http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(c)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request, principal *auth.Principal) {
	modeStr := r.URL.Query().Get("mode")
	if modeStr == "" {
		authmw.WriteError(w, "validation_error", "mode is required (merge|draft|replace)", http.StatusUnprocessableEntity, nil)
		return
	}
	mode := cases.ImportMode(strings.ToLower(modeStr))
	if mode != cases.ImportModeMerge && mode != cases.ImportModeDraft && mode != cases.ImportModeReplace {
		authmw.WriteError(w, "validation_error", "invalid mode", http.StatusUnprocessableEntity, nil)
		return
	}
	var items []cases.CaseImportItem
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	if items == nil {
		items = []cases.CaseImportItem{}
	}
	result, err := h.Service.Import(r.Context(), mode, items, principal)
	if err != nil {
		if errors.Is(err, cases.ErrValidation) {
			authmw.WriteError(w, "validation_error", err.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request, principal *auth.Principal) {
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")
	list, err := h.Service.Export(r.Context(), category, status, principal)
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	if list == nil {
		list = []cases.Case{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
