package helper

import (
	"context"
	"fmt"
	"strings"
)

// ValidationError is a structured error with field-level messages.
type ValidationError struct {
	Entity string
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s validation error: %v", e.Entity, e.Errors)
}

// NotFoundError is returned when a resource does not exist.
type NotFoundError struct {
	Entity string
	ID     string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s not found: %s", e.Entity, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Entity)
}

// AlreadyExistsError is returned when creating a resource that already exists.
type AlreadyExistsError struct {
	Entity string
	Field  string
	Value  string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists: %s=%s", e.Entity, e.Field, e.Value)
}

// UnauthorizedError is returned when a caller lacks credentials.
type UnauthorizedError struct {
	Reason string
}

func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", e.Reason)
}

// NormalizePhone strips formatting and normalizes French numbers to E.164.
func NormalizePhone(raw string) string {
	s := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, raw)
	if strings.HasPrefix(s, "+") {
		return s
	}
	// French overseas: 0596... → +596..., 0696... → +596696...
	if strings.HasPrefix(s, "0596") || strings.HasPrefix(s, "0696") || strings.HasPrefix(s, "0697") {
		return "+596" + s[1:]
	}
	// French metropolitan: 0X... → +33X...
	if strings.HasPrefix(s, "0") {
		return "+33" + s[1:]
	}
	return s
}

// ── Tenant context ──────────────────────────────────────────────────────

type tenantKey struct{}

// WithTenant stores the tenant identity in the context.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, strings.TrimSpace(tenantID))
}

// TenantFromContext reads the tenant identity from the context.
func TenantFromContext(ctx context.Context) (string, error) {
	tenantID, _ := ctx.Value(tenantKey{}).(string)
	if tenantID == "" {
		return "", &UnauthorizedError{Reason: "tenant context required"}
	}
	return tenantID, nil
}
