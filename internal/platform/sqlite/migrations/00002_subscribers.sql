-- +goose Up
CREATE TABLE subscribers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    email      TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

-- +goose Down
DROP TABLE subscribers;
