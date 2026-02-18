package cases

import "errors"

var (
	ErrNotFound      = errors.New("case not found")
	ErrForbidden     = errors.New("access denied")
	ErrInvalidStatus = errors.New("invalid status transition")
	ErrValidation    = errors.New("validation error")
)
