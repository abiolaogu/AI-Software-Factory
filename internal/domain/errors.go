package domain

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorCode string

const (
	ErrorCodeNotFound       ErrorCode = "NOT_FOUND"
	ErrorCodeUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrorCodeValidation     ErrorCode = "VALIDATION_ERROR"
	ErrorCodeTenantRequired ErrorCode = "TENANT_REQUIRED"
	ErrorCodeConflict       ErrorCode = "CONFLICT"
	ErrorCodeInternal       ErrorCode = "INTERNAL_ERROR"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrValidation     = errors.New("validation failed")
	ErrTenantRequired = errors.New("tenant required")
	ErrConflict       = errors.New("conflict")
)

type AppError struct {
	Code    ErrorCode
	Op      string
	Message string
	Details []ErrorDetail
	Err     error
}

func (e *AppError) Error() string {
	parts := make([]string, 0, 3)
	if strings.TrimSpace(e.Op) != "" {
		parts = append(parts, e.Op)
	}
	if strings.TrimSpace(e.Message) != "" {
		parts = append(parts, e.Message)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	if len(parts) == 0 {
		return string(e.Code)
	}

	return fmt.Sprintf("%s (%s)", strings.Join(parts, ": "), e.Code)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewValidationError(op, message string, details ...ErrorDetail) *AppError {
	return &AppError{
		Code:    ErrorCodeValidation,
		Op:      op,
		Message: message,
		Details: details,
		Err:     ErrValidation,
	}
}

func NewUnauthorizedError(op, message string) *AppError {
	return &AppError{
		Code:    ErrorCodeUnauthorized,
		Op:      op,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

func NewTenantRequiredError(op string) *AppError {
	return &AppError{
		Code:    ErrorCodeTenantRequired,
		Op:      op,
		Message: "missing tenant scope",
		Err:     ErrTenantRequired,
	}
}

func NewNotFoundError(op, message string) *AppError {
	return &AppError{
		Code:    ErrorCodeNotFound,
		Op:      op,
		Message: message,
		Err:     ErrNotFound,
	}
}

func NewConflictError(op, message string) *AppError {
	return &AppError{
		Code:    ErrorCodeConflict,
		Op:      op,
		Message: message,
		Err:     ErrConflict,
	}
}

func WrapError(op string, err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Op:      op,
		Message: message,
		Err:     err,
	}
}
