-- +goose Up
-- +goose StatementBegin
CREATE TABLE products (
    id UUID PRIMARY KEY,
    name VARCHAR(512) NOT NULL,
    slug VARCHAR(512) UNIQUE NOT NULL,
    description VARCHAR(512) NOT NULL,
    main_img UUID REFERENCES images(id) ON DELETE SET NULL,
    price INT DEFAULT 0 NOT NULL,
    features JSONB NOT NULL,
    category UUID REFERENCES categories(id) ON DELETE RESTRICT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE products;
-- +goose StatementEnd
