-- +goose Up
-- +goose StatementBegin
CREATE TABLE event_kinds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(256) UNIQUE NOT NULL,
    description VARCHAR(512),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER event_kinds_set_updated_at BEFORE UPDATE ON event_kinds
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_kinds;
-- +goose StatementEnd
