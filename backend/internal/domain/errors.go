package domain

import (
	"errors"
	"strings"
)

var (
	ErrAPIKeyUnauthorized = errors.New("api key unauthorized")
	ErrPermissionDenied   = errors.New("permission denied")
)

type FieldError struct {
	Field   string
	Message string
	Code    string
}

type ValidationError struct {
	Code        string
	Detail      string
	FieldErrors []FieldError
}

func (err *ValidationError) Error() string {
	if err == nil {
		return ""
	}
	if strings.TrimSpace(err.Detail) != "" {
		return err.Detail
	}
	if len(err.FieldErrors) > 0 {
		return err.FieldErrors[0].Message
	}
	return "validation failed"
}

func (err *ValidationError) Add(field string, message string, code string) {
	err.FieldErrors = append(err.FieldErrors, FieldError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

func (err *ValidationError) HasFieldErrors() bool {
	return len(err.FieldErrors) > 0
}
