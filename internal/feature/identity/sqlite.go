package identity

import (
	"context"
	"database/sql"
	"errors"
	"time"

	platformsqlite "github.com/esrid/mon-template-go/internal/platform/sqlite"
)

type SQLiteStore struct{ db *platformsqlite.DB }

func NewSQLiteStore(db *platformsqlite.DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) CreatePasswordUser(ctx context.Context, email, passwordHash string, now time.Time) (User, error) {
	row := s.db.SQL().QueryRowContext(ctx, `INSERT INTO users(email,password_hash,created_at,updated_at) VALUES(?,?,?,?) RETURNING id,email,email_verified_at,created_at,updated_at,password_hash`, email, passwordHash, now.Unix(), now.Unix())
	user, _, err := scanUser(row.Scan)
	if platformsqlite.IsUniqueViolation(err) {
		return User{}, ErrEmailTaken
	}
	if err != nil {
		return User{}, platformsqlite.DecorateError(err, "identity.CreatePasswordUser")
	}
	return user, nil
}
func (s *SQLiteStore) ByEmail(ctx context.Context, email string) (User, string, error) {
	row := s.db.SQL().QueryRowContext(ctx, `SELECT id,email,email_verified_at,created_at,updated_at,password_hash FROM users WHERE email=?`, email)
	user, hash, err := scanUser(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	if err != nil {
		return User{}, "", platformsqlite.DecorateError(err, "identity.ByEmail")
	}
	return user, hash, nil
}
func (s *SQLiteStore) ByID(ctx context.Context, id int64) (User, error) {
	row := s.db.SQL().QueryRowContext(ctx, `SELECT id,email,email_verified_at,created_at,updated_at,password_hash FROM users WHERE id=?`, id)
	user, _, err := scanUser(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, platformsqlite.DecorateError(err, "identity.ByID")
	}
	return user, nil
}
func (s *SQLiteStore) UpsertGoogleUser(ctx context.Context, external GoogleIdentity, now time.Time) (User, error) {
	var user User
	err := s.db.WithinTransaction(ctx, func(tx *sql.Tx) error {
		var userID int64
		err := tx.QueryRowContext(ctx, `SELECT user_id FROM user_identities WHERE provider='google' AND provider_subject=?`, external.Subject).Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			var verifiedAt any
			if external.EmailVerified {
				verifiedAt = now.Unix()
			}
			err = tx.QueryRowContext(ctx, `INSERT INTO users(email,email_verified_at,created_at,updated_at) VALUES(?,?,?,?) ON CONFLICT(email) DO UPDATE SET email_verified_at=COALESCE(users.email_verified_at,excluded.email_verified_at),updated_at=excluded.updated_at RETURNING id`, external.Email, verifiedAt, now.Unix(), now.Unix()).Scan(&userID)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO user_identities(user_id,provider,provider_subject,provider_email,created_at) VALUES(?,'google',?,?,?)`, userID, external.Subject, external.Email, now.Unix()); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		row := tx.QueryRowContext(ctx, `SELECT id,email,email_verified_at,created_at,updated_at,password_hash FROM users WHERE id=?`, userID)
		var hash string
		user, hash, err = scanUser(row.Scan)
		_ = hash
		return err
	})
	if err != nil {
		return User{}, platformsqlite.DecorateError(err, "identity.UpsertGoogleUser")
	}
	return user, nil
}
func (s *SQLiteStore) SaveToken(ctx context.Context, kind TokenKind, userID int64, hash []byte, expiresAt, now time.Time) error {
	table, err := tokenTable(kind)
	if err != nil {
		return err
	}
	_, err = s.db.SQL().ExecContext(ctx, `DELETE FROM `+table+` WHERE user_id=?`, userID)
	if err != nil {
		return platformsqlite.DecorateError(err, "identity.SaveToken.delete")
	}
	_, err = s.db.SQL().ExecContext(ctx, `INSERT INTO `+table+`(token_hash,user_id,expires_at,created_at) VALUES(?,?,?,?)`, hash, userID, expiresAt.Unix(), now.Unix())
	if err != nil {
		return platformsqlite.DecorateError(err, "identity.SaveToken.insert")
	}
	return nil
}
func (s *SQLiteStore) ConsumeToken(ctx context.Context, kind TokenKind, hash []byte, now time.Time) (User, error) {
	table, err := tokenTable(kind)
	if err != nil {
		return User{}, err
	}
	var user User
	err = s.db.WithinTransaction(ctx, func(tx *sql.Tx) error {
		var userID int64
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM `+table+` WHERE token_hash=? AND expires_at>?`, hash, now.Unix()).Scan(&userID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE token_hash=?`, hash); err != nil {
			return err
		}
		if kind == EmailVerificationToken {
			if _, err := tx.ExecContext(ctx, `UPDATE users SET email_verified_at=?,updated_at=? WHERE id=?`, now.Unix(), now.Unix(), userID); err != nil {
				return err
			}
		}
		row := tx.QueryRowContext(ctx, `SELECT id,email,email_verified_at,created_at,updated_at,password_hash FROM users WHERE id=?`, userID)
		user, _, err = scanUser(row.Scan)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrInvalidToken
	}
	if err != nil {
		return User{}, platformsqlite.DecorateError(err, "identity.ConsumeToken")
	}
	return user, nil
}
func (s *SQLiteStore) UpdatePassword(ctx context.Context, userID int64, passwordHash string, now time.Time) error {
	result, err := s.db.SQL().ExecContext(ctx, `UPDATE users SET password_hash=?,updated_at=? WHERE id=?`, passwordHash, now.Unix(), userID)
	if err != nil {
		return platformsqlite.DecorateError(err, "identity.UpdatePassword")
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
func tokenTable(kind TokenKind) (string, error) {
	switch kind {
	case EmailVerificationToken:
		return "email_verification_tokens", nil
	case PasswordResetToken:
		return "password_reset_tokens", nil
	default:
		return "", ErrInvalidToken
	}
}
func scanUser(into func(...any) error) (User, string, error) {
	var u User
	var verified sql.NullInt64
	var created, updated int64
	var hash sql.NullString
	if err := into(&u.ID, &u.Email, &verified, &created, &updated, &hash); err != nil {
		return User{}, "", err
	}
	u.CreatedAt = time.Unix(created, 0).UTC()
	u.UpdatedAt = time.Unix(updated, 0).UTC()
	if verified.Valid {
		v := time.Unix(verified.Int64, 0).UTC()
		u.EmailVerifiedAt = &v
	}
	return u, hash.String, nil
}
