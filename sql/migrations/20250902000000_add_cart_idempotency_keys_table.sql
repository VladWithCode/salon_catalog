-- +goose Up
-- +goose StatementBegin
CREATE TABLE cart_idempotency_keys (
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    key_hash BYTEA NOT NULL,
    request_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT cart_idempotency_keys_pkey PRIMARY KEY (cart_id, key_hash),
    CONSTRAINT cart_idempotency_keys_key_hash_length CHECK (octet_length(key_hash) = 32),
    CONSTRAINT cart_idempotency_keys_request_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT cart_idempotency_keys_expires_after_created CHECK (expires_at > created_at)
);

CREATE INDEX idx_cart_idempotency_keys_expires_at ON cart_idempotency_keys(expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cart_idempotency_keys_expires_at;

DROP TABLE cart_idempotency_keys;
-- +goose StatementEnd
