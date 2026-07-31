package identity

import (
	"context"
	"time"
)

type Store interface {
	CreatePasswordUser(ctx context.Context, email, passwordHash string, now time.Time) (User, error)
	ByEmail(ctx context.Context, email string) (User, string, error)
	ByID(ctx context.Context, id int64) (User, error)
	UpsertGoogleUser(ctx context.Context, identity GoogleIdentity, now time.Time) (User, error)
	SaveToken(ctx context.Context, kind TokenKind, userID int64, hash []byte, expiresAt, now time.Time) error
	ConsumeToken(ctx context.Context, kind TokenKind, hash []byte, now time.Time) (User, error)
	UpdatePassword(ctx context.Context, userID int64, passwordHash string, now time.Time) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
	NeedsRehash(hash string) bool
}

type Clock func() time.Time
