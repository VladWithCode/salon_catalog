-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL,
    slug VARCHAR(200) UNIQUE NOT NULL,
    description VARCHAR(128) NOT NULL,
    long_description VARCHAR(512),
    main_img UUID REFERENCES images(id) ON DELETE SET NULL,
    category UUID REFERENCES categories(id) ON DELETE RESTRICT,
    available BOOLEAN NOT NULL DEFAULT TRUE,
    quantity INTEGER NOT NULL DEFAULT 1
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE products;
-- +goose StatementEnd
