-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_products_category ON public.products(category);
CREATE INDEX idx_images_products_product_id ON public.images_products(product_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_products_category;
DROP INDEX IF EXISTS idx_images_products_product_id;
-- +goose StatementEnd
