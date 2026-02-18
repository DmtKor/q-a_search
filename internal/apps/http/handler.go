package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yourusername/project/internal/apps"
	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
)

// Handler handles /api/v1/apps and /api/v1/apps/{id}/settings, export, import.
// Mount at /api/v1/apps; path is relative to that. Expects RequireStaff and principal in context.
type Handler struct {
	Service *apps.Service
}

// ServeHTTP dispatches by method and path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/apps")
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
			h.List(w, r)
			return
		}
		if r.Method == http.MethodPost {
			h.Create(w, r)
			return
		}
	case len(parts) == 1 && isUUID(parts[0]):
		id := parts[0]
		switch r.Method {
		case http.MethodGet:
			h.Get(w, r, id)
			return
		case http.MethodPut:
			h.Update(w, r, id)
			return
		}
	case len(parts) == 2 && isUUID(parts[0]) && parts[1] == "settings":
		id := parts[0]
		if r.Method == http.MethodGet {
			h.GetSettings(w, r, id)
			return
		}
		if r.Method == http.MethodPut {
			h.UpdateSettings(w, r, id)
			return
		}
	case len(parts) == 3 && isUUID(parts[0]) && parts[1] == "settings" && parts[2] == "export":
		if r.Method == http.MethodGet {
			h.Export(w, r, parts[0])
			return
		}
	case len(parts) == 3 && isUUID(parts[0]) && parts[1] == "settings" && parts[2] == "import":
		if r.Method == http.MethodPost {
			h.Import(w, r, parts[0])
			return
		}
	}

	authmw.WriteError(w, "validation_error", "Method not allowed", http.StatusMethodNotAllowed, nil)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.Service.List(r.Context())
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	if list == nil {
		list = []apps.App{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body apps.AppCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	a, err := h.Service.Create(r.Context(), body)
	if err != nil {
		if errors.Is(err, apps.ErrConflict) {
			authmw.WriteError(w, "conflict", "App name already exists", http.StatusConflict, nil)
			return
		}
		if apps.IsValidationError(err) {
			authmw.WriteError(w, "validation_error", err.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(a)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, id string) {
	a, err := h.Service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var body apps.AppUpdate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	a, err := h.Service.Update(r.Context(), id, body)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		if errors.Is(err, apps.ErrConflict) {
			authmw.WriteError(w, "conflict", "App name already exists", http.StatusConflict, nil)
			return
		}
		if apps.IsValidationError(err) {
			authmw.WriteError(w, "validation_error", err.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(a)
}

func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request, id string) {
	settings, err := h.Service.GetSettings(r.Context(), id)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(settings)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request, id string) {
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	out, err := h.Service.UpdateSettings(r.Context(), id, settings)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		if apps.IsValidationError(err) {
			authmw.WriteError(w, "validation_error", err.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request, id string) {
	settings, err := h.Service.Export(r.Context(), id)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(settings)
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request, id string) {
	var settings map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	out, err := h.Service.Import(r.Context(), id, settings)
	if err != nil {
		if errors.Is(err, apps.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{"resource": "app"})
			return
		}
		if apps.IsValidationError(err) {
			authmw.WriteError(w, "validation_error", err.Error(), http.StatusUnprocessableEntity, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
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
