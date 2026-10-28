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
	// ErrMailRejected means the email provider refused a message for good;
	// retrying will not help.
	ErrMailRejected = errors.New("the email provider rejected the message")

	ErrInvalidCredentials = errors.New("the email or password is not correct")
	ErrInvalidCode        = errors.New("the verification code is not correct")
	ErrInvalidLink        = errors.New("this link is not valid or has expired, please ask for a new one")
	ErrAccountLocked      = errors.New("too many failed sign-in attempts, please try again in a few minutes")
)
