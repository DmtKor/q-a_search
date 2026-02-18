package search

import "errors"

var (
	ErrInvalidTopK = errors.New("top_k must be between 1 and 50")
)
