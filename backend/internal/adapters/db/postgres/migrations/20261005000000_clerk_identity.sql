-- +goose Up
-- Clerk becomes the only identity system (ADR-0015), so no existing user has an
-- Auth Identity to map to. Deleting every user is intentional ("start fresh",
-- no live users): it cascades to user_profiles, collections and inventory
-- entries, and runs on the production database too.
DELETE FROM users;

DROP TABLE refresh_tokens;
DROP TABLE sessions;

DROP INDEX idx_users_email;
ALTER TABLE users
    DROP COLUMN password_hash,
    DROP COLUMN verified,
    ADD COLUMN auth_provider TEXT NOT NULL,
    ADD COLUMN auth_subject TEXT NOT NULL;

CREATE UNIQUE INDEX idx_users_auth_identity ON users (auth_provider, auth_subject);
-- Email is no longer a login key, so it is neither unique nor looked up on the
-- hot path; the index only serves operators finding an account.
CREATE INDEX idx_users_email ON users (email);

-- +goose Down
-- Restores the shape only; the deleted users are gone for good.
DELETE FROM users;

DROP INDEX idx_users_auth_identity;
DROP INDEX idx_users_email;
ALTER TABLE users
    DROP COLUMN auth_provider,
    DROP COLUMN auth_subject,
    ADD COLUMN password_hash TEXT NOT NULL,
    ADD COLUMN verified BOOLEAN NOT NULL DEFAULT false;
CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    session_id UUID NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX idx_refresh_tokens_session_id ON refresh_tokens (session_id);
