package apps

import "errors"

var (
	ErrNotFound = errors.New("app not found")
	ErrConflict = errors.New("app name already exists")
)
