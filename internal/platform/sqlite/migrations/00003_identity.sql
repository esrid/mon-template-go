-- +goose Up
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL COLLATE NOCASE UNIQUE,
    password_hash TEXT,
    email_verified_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    CHECK (password_hash IS NOT NULL OR email_verified_at IS NOT NULL)
);

CREATE TABLE user_identities (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_subject TEXT NOT NULL,
    provider_email TEXT,
    created_at INTEGER NOT NULL,
    UNIQUE(provider, provider_subject)
);
CREATE INDEX user_identities_user_id_idx ON user_identities(user_id);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    expiry REAL NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);

CREATE TABLE email_verification_tokens (
    token_hash BLOB PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX email_verification_tokens_user_id_idx ON email_verification_tokens(user_id);

CREATE TABLE password_reset_tokens (
    token_hash BLOB PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens(user_id);

-- +goose Down
DROP TABLE password_reset_tokens;
DROP TABLE email_verification_tokens;
DROP TABLE sessions;
DROP TABLE user_identities;
DROP TABLE users;
