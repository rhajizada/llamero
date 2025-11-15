-- +goose Up
CREATE TABLE tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    scopes TEXT[] NOT NULL CHECK (array_length(scopes, 1) > 0),
    token_type TEXT NOT NULL DEFAULT 'pat',
    jti TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX tokens_user_id_idx ON tokens(user_id);
CREATE INDEX tokens_jti_idx ON tokens(jti);

-- +goose Down
DROP INDEX IF EXISTS tokens_jti_idx;
DROP INDEX IF EXISTS tokens_user_id_idx;
DROP TABLE IF EXISTS tokens;
