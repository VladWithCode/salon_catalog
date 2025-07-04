-- +goose Up
-- +goose StatementBegin
CREATE TABLE images_products (
    image_id UUID REFERENCES images(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE images_products;
-- +goose StatementEnd
