package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrConflict           = errors.New("conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already taken")
	ErrSelfLink           = errors.New("cannot link ticket to itself")
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
