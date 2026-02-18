package auth

import (
	"encoding/json"
	"net/http"
)

// ErrorEnvelope matches OpenAPI components/schemas/ErrorEnvelope.
type ErrorEnvelope struct {
	Error struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Details interface{} `json:"details"`
	} `json:"error"`
}

// WriteError writes a JSON error response with the given code, message and status code.
func WriteError(w http.ResponseWriter, code, message string, statusCode int, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	env := ErrorEnvelope{}
	env.Error.Code = code
	env.Error.Message = message
	env.Error.Details = details
	_ = json.NewEncoder(w).Encode(env)
}

// Write401 writes 401 Unauthorized with code "unauthorized" (OpenAPI).
func Write401(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Missing or invalid authorization token"
	}
	WriteError(w, "unauthorized", message, http.StatusUnauthorized, nil)
}

// Write403 writes 403 Forbidden with code "forbidden" (OpenAPI).
func Write403(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Access denied"
	}
	WriteError(w, "forbidden", message, http.StatusForbidden, nil)
}
