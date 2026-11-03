-- +goose Up
-- Who has turned the weekly digest email off. Everyone with access gets it
-- unless they have a row here saying otherwise.
CREATE TABLE digest_preferences (
    user_id    UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    weekly     BOOLEAN     NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
