-- +goose Up
-- +goose StatementBegin
ALTER TABLE carts ADD COLUMN customer_name VARCHAR(160);
ALTER TABLE carts ADD COLUMN customer_email VARCHAR(256);
ALTER TABLE carts ADD COLUMN customer_phone VARCHAR(16);
ALTER TABLE carts ADD COLUMN is_submitted BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_carts_customer_name ON carts(customer_name);
CREATE INDEX idx_carts_customer_email ON carts(customer_email);
CREATE INDEX idx_carts_customer_phone ON carts(customer_phone);
CREATE INDEX idx_carts_is_submitted ON carts(is_submitted);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_carts_customer_name;
DROP INDEX IF EXISTS idx_carts_customer_email;
DROP INDEX IF EXISTS idx_carts_customer_phone;
DROP INDEX IF EXISTS idx_carts_is_submitted;

ALTER TABLE carts DROP COLUMN IF EXISTS customer_name;
ALTER TABLE carts DROP COLUMN IF EXISTS customer_email;
ALTER TABLE carts DROP COLUMN IF EXISTS customer_phone;
ALTER TABLE carts DROP COLUMN IF EXISTS is_submitted;
-- +goose StatementEnd
