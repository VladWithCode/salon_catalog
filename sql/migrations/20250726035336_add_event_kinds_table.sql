-- +goose Up
-- +goose StatementBegin
CREATE TABLE event_kinds (
    id UUID PRIMARY KEY,
    name VARCHAR(256) UNIQUE NOT NULL,
    description VARCHAR(512)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE event_kinds;
-- +goose StatementEnd
