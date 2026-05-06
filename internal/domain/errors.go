package domain

import "errors"

var (
	// ErrNotFound means the requested aggregate does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict means a uniqueness or state invariant was violated.
	ErrConflict = errors.New("conflict")
	// ErrInvalid means input validation failed (map to HTTP 422).
	ErrInvalid = errors.New("invalid input")
)
