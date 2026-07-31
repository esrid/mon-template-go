// Package identity owns local users, credentials, external identities, and
// the application-level authentication workflows.
package identity

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidEmail       = errors.New("identity: invalid email")
	ErrInvalidPassword    = errors.New("identity: invalid password")
	ErrPasswordTooLong    = errors.New("identity: password exceeds bcrypt limit")
	ErrEmailTaken         = errors.New("identity: email already registered")
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrNotFound           = errors.New("identity: user not found")
	ErrInvalidToken       = errors.New("identity: invalid or expired token")
	ErrGoogleDisabled     = errors.New("identity: google login is disabled")
)

type User struct {
	ID              int64      `json:"id"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
}

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }
