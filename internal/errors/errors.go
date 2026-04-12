package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ErrorCode represents the type of application error
type ErrorCode int

const (
	ErrNotFound ErrorCode = iota + 1
	ErrForbidden
	ErrBadRequest
	ErrUnauthorized
	ErrConflict
)

// AppError is a custom error type that carries an HTTP status code
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code for this error
func (e *AppError) StatusCode() int {
	switch e.Code {
	case ErrNotFound:
		return http.StatusNotFound
	case ErrForbidden:
		return http.StatusForbidden
	case ErrBadRequest:
		return http.StatusBadRequest
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// ErrorResponse represents a standardized JSON error response
type ErrorResponse struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

// WriteJSONError writes a JSON error response with the proper headers and status code
func WriteJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// WriteAppError writes an AppError as a JSON response
func WriteAppError(w http.ResponseWriter, appErr *AppError) {
	WriteJSONError(w, appErr.Message, appErr.StatusCode())
}

// Helper functions to create errors easily
func NewBadRequest(msg string) *AppError {
	return &AppError{Code: ErrBadRequest, Message: msg}
}

func NewForbidden(msg string) *AppError {
	return &AppError{Code: ErrForbidden, Message: msg}
}

func NewNotFound(msg string) *AppError {
	return &AppError{Code: ErrNotFound, Message: msg}
}

func NewConflict(msg string) *AppError {
	return &AppError{Code: ErrConflict, Message: msg}
}

func WrapAppError(err error, code ErrorCode, msg string) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}
