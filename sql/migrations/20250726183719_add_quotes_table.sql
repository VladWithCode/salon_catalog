-- +goose Up
-- +goose StatementBegin
CREATE TABLE quotes (
    id UUID PRIMARY KEY,
    customer_name VARCHAR(256) NOT NULL,
    customer_phone VARCHAR(256) NOT NULL,
    time_start TIMESTAMP,
    time_end TIMESTAMP,
    cart_id UUID REFERENCES carts(id) ON DELETE SET NULL,
    request_type VARCHAR(64) NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'pendiente',
    comments VARCHAR(512),
    event_kind_id UUID REFERENCES event_kinds(id) ON DELETE RESTRICT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER quotes_set_updated_at BEFORE UPDATE ON quotes
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER quotes_set_updated_at ON quotes;

DROP TABLE quotes;
-- +goose StatementEnd
