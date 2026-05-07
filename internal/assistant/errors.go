package assistant

import "github.com/kenichiLyon/loong64-b1-go/internal/teaching"

func validationError(message string) error {
	return &teaching.Error{Kind: teaching.KindValidation, Code: "validation_error", Message: message}
}

func forbiddenError(message string) error {
	return &teaching.Error{Kind: teaching.KindForbidden, Code: "forbidden", Message: message}
}

func conflictError(message string) error {
	return &teaching.Error{Kind: teaching.KindConflict, Code: "conflict", Message: message}
}

func unavailableError(message string, err error) error {
	return &teaching.Error{Kind: teaching.KindUnavailable, Code: "service_unavailable", Message: message, Err: err}
}

func notFoundError(message string) error {
	return &teaching.Error{Kind: teaching.KindNotFound, Code: "not_found", Message: message}
}
