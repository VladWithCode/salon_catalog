-- +goose Up
-- +goose StatementBegin
CREATE TABLE images_products (
    image_id UUID REFERENCES images(id),
    product_id UUID REFERENCES products(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE images_products;
-- +goose StatementEnd
