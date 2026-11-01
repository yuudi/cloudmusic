-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY,
    username text,
    display_name text NOT NULL DEFAULT '',
    is_guest boolean NOT NULL DEFAULT true,
    last_active_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (username IS NULL OR username ~ '^[A-Za-z0-9_]{3,32}$'),
    CHECK ((is_guest AND username IS NULL) OR (NOT is_guest AND username IS NOT NULL) OR deleted_at IS NOT NULL)
);
CREATE UNIQUE INDEX users_username_unique ON users (lower(username)) WHERE username IS NOT NULL;
CREATE INDEX users_guest_expiry ON users (last_active_at) WHERE is_guest AND deleted_at IS NULL;
CREATE TABLE password_credentials (
    user_id uuid PRIMARY KEY REFERENCES users(id),
    password_hash text NOT NULL
);
CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id ON sessions (user_id);
CREATE INDEX sessions_expiry ON sessions (expires_at);
-- +goose Down
DROP TABLE sessions;
DROP TABLE password_credentials;
DROP TABLE users;
