package tickets

import "errors"

var (
	ErrNotFound   = errors.New("ticket not found")
	ErrConflict   = errors.New("ticket already converted to case")
	ErrValidation = errors.New("validation error")
)
