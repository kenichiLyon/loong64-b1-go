package teaching

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const WeightTotalBPS = 10000

type ErrorKind string

const (
	KindValidation   ErrorKind = "validation"
	KindUnauthorized ErrorKind = "unauthorized"
	KindForbidden    ErrorKind = "forbidden"
	KindNotFound     ErrorKind = "not_found"
	KindConflict     ErrorKind = "conflict"
	KindUnavailable  ErrorKind = "unavailable"
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func newError(kind ErrorKind, code, message string, err error) error {
	return &Error{Kind: kind, Code: code, Message: message, Err: err}
}

func validationError(message string) error {
	return newError(KindValidation, "validation_error", message, nil)
}
func forbiddenError(message string) error { return newError(KindForbidden, "forbidden", message, nil) }
func unauthorizedError(message string) error {
	return newError(KindUnauthorized, "unauthorized", message, nil)
}
func notFoundError(message string) error { return newError(KindNotFound, "not_found", message, nil) }
func conflictError(message string) error { return newError(KindConflict, "conflict", message, nil) }
func unavailableError(message string, err error) error {
	return newError(KindUnavailable, "service_unavailable", message, err)
}

func ErrorKindOf(err error) ErrorKind {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Kind
	}
	return ""
}

func ErrorCodeOf(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Code != "" {
		return appErr.Code
	}
	return "internal_error"
}

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
)

func ParseRole(value string) (Role, error) {
	role := Role(strings.ToLower(strings.TrimSpace(value)))
	switch role {
	case RoleAdmin, RoleTeacher, RoleStudent:
		return role, nil
	default:
		return "", validationError("invalid role: " + value)
	}
}

func ParseRoleList(values []string) ([]Role, error) {
	roles := make([]Role, 0, len(values))
	seen := make(map[Role]struct{}, len(values))
	for _, value := range values {
		role, err := ParseRole(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return nil, validationError("at least one role is required")
	}
	return roles, nil
}

type Actor struct {
	ID    string
	Roles map[Role]struct{}
}

func NewActor(id string, roles []Role) (Actor, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Actor{}, unauthorizedError("actor id is required")
	}
	if len(roles) == 0 {
		return Actor{}, unauthorizedError("actor role is required")
	}
	set := make(map[Role]struct{}, len(roles))
	for _, role := range roles {
		switch role {
		case RoleAdmin, RoleTeacher, RoleStudent:
			set[role] = struct{}{}
		default:
			return Actor{}, validationError("invalid actor role")
		}
	}
	return Actor{ID: id, Roles: set}, nil
}

func (a Actor) Has(role Role) bool {
	_, ok := a.Roles[role]
	return ok
}

func (a Actor) RoleValues() []Role {
	roles := make([]Role, 0, len(a.Roles))
	for _, role := range []Role{RoleAdmin, RoleTeacher, RoleStudent} {
		if a.Has(role) {
			roles = append(roles, role)
		}
	}
	return roles
}

func (a Actor) Require(role Role) error {
	if !a.Has(role) {
		return forbiddenError("required role: " + string(role))
	}
	return nil
}

type WeightMode string

const (
	WeightModeStrict100  WeightMode = "strict_100"
	WeightModeNormalized WeightMode = "normalized"
)

func ParseWeightMode(value string) (WeightMode, error) {
	mode := WeightMode(strings.ToLower(strings.TrimSpace(value)))
	switch mode {
	case WeightModeStrict100, WeightModeNormalized:
		return mode, nil
	default:
		return "", validationError("invalid weight mode")
	}
}

func ValidateMetrics(mode WeightMode, metrics []MetricInput) error {
	if len(metrics) == 0 {
		return validationError("at least one rubric metric is required")
	}
	if mode != WeightModeStrict100 && mode != WeightModeNormalized {
		return validationError("invalid weight mode")
	}
	codes := make(map[string]struct{}, len(metrics))
	orders := make(map[int]struct{}, len(metrics))
	total := 0
	for _, metric := range metrics {
		code := normalizeCode(metric.Code)
		if code == "" {
			return validationError("metric code is required")
		}
		if _, ok := codes[code]; ok {
			return validationError("metric code must be unique")
		}
		codes[code] = struct{}{}
		if strings.TrimSpace(metric.Name) == "" {
			return validationError("metric name is required")
		}
		if metric.WeightBPS < 0 {
			return validationError("metric weight_bps must not be negative")
		}
		if metric.MaxScore <= 0 {
			return validationError("metric max_score must be greater than zero")
		}
		if metric.SortOrder <= 0 {
			return validationError("metric sort_order must be greater than zero")
		}
		if _, ok := orders[metric.SortOrder]; ok {
			return validationError("metric sort_order must be unique")
		}
		orders[metric.SortOrder] = struct{}{}
		total += metric.WeightBPS
		if len(metric.RequiredEvidence) > 0 && !json.Valid(metric.RequiredEvidence) {
			return validationError("metric required_evidence must be valid JSON")
		}
	}
	switch mode {
	case WeightModeStrict100:
		if total != WeightTotalBPS {
			return validationError(fmt.Sprintf("strict_100 metric weights must sum to %d", WeightTotalBPS))
		}
	case WeightModeNormalized:
		if total <= 0 {
			return validationError("normalized metric weights must sum to more than zero")
		}
	}
	return nil
}

func ValidateTimeWindow(startAt, dueAt *time.Time) error {
	if startAt != nil && dueAt != nil && !dueAt.After(*startAt) {
		return validationError("due_at must be after start_at")
	}
	return nil
}

func NewID(prefix string) string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}
	return prefix + "_" + hex.EncodeToString(buf[:])
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeStatus(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value
}

func defaultJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
