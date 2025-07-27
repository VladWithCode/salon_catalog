-- +goose Up
-- +goose StatementBegin
CREATE TABLE carts (
    id UUID PRIMARY KEY,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER carts_set_updated_at BEFORE UPDATE ON carts
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER carts_set_updated_at ON carts;

DROP TABLE carts;
-- +goose StatementEnd
