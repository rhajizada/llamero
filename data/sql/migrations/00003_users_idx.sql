-- +goose Up
CREATE UNIQUE INDEX users_provider_sub_idx
    ON users (provider, sub);
CREATE UNIQUE INDEX users_email_idx
    ON users (email);

-- +goose Down
DROP INDEX users_email_idx;
DROP INDEX users_provider_sub_idx;
