-- +goose Up
CREATE TABLE links (
    id           BIGSERIAL   PRIMARY KEY,
    code         TEXT        NOT NULL UNIQUE,
    original_url TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE links;
