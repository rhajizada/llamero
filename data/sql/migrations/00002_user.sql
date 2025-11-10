-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sub TEXT NOT NULL,
    provider TEXT NOT NULL,
    email CITEXT NOT NULL,
    display_name TEXT,
    role TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    groups TEXT[] NOT NULL,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE users;
