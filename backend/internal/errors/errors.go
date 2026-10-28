// Package apperrors holds the sentinel errors the application speaks in.
// Messages are worded for the person who will read them. Wrap with %w and
// map to HTTP status in one place only (transport/http/errors.go).
package apperrors

import "errors"

var (
	ErrNotFound     = errors.New("we couldn't find what you were looking for")
	ErrInvalidInput = errors.New("some of the information provided is not valid")
	ErrConflict     = errors.New("this conflicts with a change that was just made, please refresh and try again")
	ErrUnauthorized = errors.New("please sign in to continue")
	ErrForbidden    = errors.New("you do not have permission to do this")
	ErrUnavailable  = errors.New("the service is temporarily unavailable, please try again shortly")
)
