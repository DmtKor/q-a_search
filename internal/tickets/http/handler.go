package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/yourusername/project/internal/auth"
	authmw "github.com/yourusername/project/internal/http/middleware/auth"
	"github.com/yourusername/project/internal/tickets"
)

// Handler handles /api/v1/tickets (list, create, get, update, convert-to-case).
// Mount at /api/v1; path is relative to that (e.g. tickets, tickets/{id}, tickets/{id}/convert-to-case).
// Expects RequireStaff and principal in context.
type Handler struct {
	Service *tickets.Service
}

// ServeHTTP dispatches by method and path.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/tickets")
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
	case len(parts) == 2 && isUUID(parts[0]) && parts[1] == "convert-to-case":
		if r.Method == http.MethodPost {
			h.ConvertToCase(w, r, parts[0], principal)
			return
		}
	}

	authmw.WriteError(w, "validation_error", "Method not allowed", http.StatusMethodNotAllowed, nil)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filters := tickets.ListFilters{
		Status:   r.URL.Query().Get("status"),
		Category: r.URL.Query().Get("category"),
	}
	list, err := h.Service.List(r.Context(), filters)
	if err != nil {
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	if list == nil {
		list = []tickets.Ticket{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(list)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var body tickets.TicketCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	t, err := h.Service.Create(r.Context(), body)
	if err != nil {
		if errors.Is(err, tickets.ErrValidation) {
			authmw.WriteError(w, "validation_error", "query is required", http.StatusUnprocessableEntity, map[string]interface{}{
				"field": "query", "reason": "required",
			})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, id string) {
	t, err := h.Service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, tickets.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{
				"resource": "ticket",
			})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var body tickets.TicketUpdate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		authmw.WriteError(w, "validation_error", "Invalid request body", http.StatusUnprocessableEntity, map[string]interface{}{
			"field": "body", "reason": err.Error(),
		})
		return
	}
	t, err := h.Service.Update(r.Context(), id, body)
	if err != nil {
		if errors.Is(err, tickets.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{
				"resource": "ticket",
			})
			return
		}
		if errors.Is(err, tickets.ErrValidation) {
			authmw.WriteError(w, "validation_error", "Invalid status", http.StatusUnprocessableEntity, map[string]interface{}{
				"field": "status", "reason": "must be one of open, in_progress, resolved, closed",
			})
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(t)
}

func (h *Handler) ConvertToCase(w http.ResponseWriter, r *http.Request, id string, principal *auth.Principal) {
	var body tickets.ConvertToCaseRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	resp, err := h.Service.ConvertToCase(r.Context(), id, body, principal.TokenID)
	if err != nil {
		if errors.Is(err, tickets.ErrNotFound) {
			authmw.WriteError(w, "not_found", "Resource not found", http.StatusNotFound, map[string]interface{}{
				"resource": "ticket",
			})
			return
		}
		if errors.Is(err, tickets.ErrConflict) {
			authmw.WriteError(w, "conflict", "Ticket already converted to case", http.StatusConflict, nil)
			return
		}
		authmw.WriteError(w, "internal_error", "An internal error occurred", http.StatusInternalServerError, nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
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
