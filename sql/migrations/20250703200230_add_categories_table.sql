-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name VARCHAR(512) NOT NULL,
    slug VARCHAR(512) UNIQUE NOT NULL,
    description VARCHAR(512) NOT NULL,
    header_img UUID REFERENCES images(id) ON DELETE SET NULL,
    display_img UUID REFERENCES images(id) ON DELETE SET NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE categories;
-- +goose StatementEnd
