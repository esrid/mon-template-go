package identity

import (
	"context"
	"net/mail"
	"time"
)

type Service struct {
	store     Store
	passwords PasswordHasher
	now       Clock
}

func New(store Store, passwords PasswordHasher, now Clock) *Service {
	return &Service{store: store, passwords: passwords, now: now}
}

func (s *Service) Register(ctx context.Context, email, password string) (User, error) {
	email = normalizeEmail(email)
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return User{}, ErrInvalidEmail
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return User{}, err
	}
	now := s.now().UTC()
	return s.store.CreatePasswordUser(ctx, email, hash, now)
}

func (s *Service) Login(ctx context.Context, email, password string) (User, error) {
	user, hash, err := s.store.ByEmail(ctx, normalizeEmail(email))
	if err != nil || hash == "" {
		return User{}, ErrInvalidCredentials
	}
	if err := s.passwords.Compare(hash, password); err != nil {
		return User{}, ErrInvalidCredentials
	}
	if s.passwords.NeedsRehash(hash) {
		if upgraded, err := s.passwords.Hash(password); err == nil {
			_ = s.store.UpdatePassword(ctx, user.ID, upgraded, s.now().UTC())
		}
	}
	return user, nil
}

func (s *Service) ByID(ctx context.Context, id int64) (User, error) { return s.store.ByID(ctx, id) }

func (s *Service) LoginWithGoogle(ctx context.Context, external GoogleIdentity) (User, error) {
	external.Email = normalizeEmail(external.Email)
	if external.Subject == "" || external.Email == "" || !external.EmailVerified {
		return User{}, ErrInvalidCredentials
	}
	return s.store.UpsertGoogleUser(ctx, external, s.now().UTC())
}

func (s *Service) IssueToken(ctx context.Context, kind TokenKind, userID int64, lifetime time.Duration) (string, error) {
	plain, hash, err := newToken()
	if err != nil {
		return "", err
	}
	now := s.now().UTC()
	if err := s.store.SaveToken(ctx, kind, userID, hash, now.Add(lifetime), now); err != nil {
		return "", err
	}
	return plain, nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) (User, error) {
	return s.store.ConsumeToken(ctx, EmailVerificationToken, tokenHash(token), s.now().UTC())
}

// ResetPassword consumes a reset token and returns the user whose password changed,
// so callers can revoke that user's existing sessions.
func (s *Service) ResetPassword(ctx context.Context, token, password string) (int64, error) {
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return 0, err
	}
	user, err := s.store.ConsumeToken(ctx, PasswordResetToken, tokenHash(token), s.now().UTC())
	if err != nil {
		return 0, err
	}
	return user.ID, s.store.UpdatePassword(ctx, user.ID, hash, s.now().UTC())
}
