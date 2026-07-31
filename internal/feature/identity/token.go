package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

type TokenKind string

const (
	EmailVerificationToken TokenKind = "email_verification"
	PasswordResetToken     TokenKind = "password_reset"
)

func newToken() (plain string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}

func tokenHash(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}
