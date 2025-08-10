-- +goose Up
-- +goose StatementBegin

-- Drop existing catalog_products view
DROP VIEW IF EXISTS catalog_products;

-- Recreate catalog_products view with quantity column
CREATE VIEW catalog_products AS
SELECT 
    p.id,
    p.name,
    p.description,
    p.long_description,
    p.category as category_id,
    c.name as category_name,
    COALESCE(main_img.filename, '') as image_url,
    p.available,
    p.slug,
    p.quantity,
    p.search_vector,
    -- Aggregate gallery images as JSON array
    COALESCE(
        (
            SELECT json_agg(i.filename ORDER BY i.filename)
            FROM public.images_products ip
            JOIN public.images i ON ip.image_id = i.id
            WHERE ip.product_id = p.id
        ),
        '[]'::json
    ) as images
FROM public.products p
LEFT JOIN public.categories c ON p.category = c.id
LEFT JOIN public.images main_img ON p.main_img = main_img.id
ORDER BY p.name;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop catalog_products view
DROP VIEW IF EXISTS catalog_products;

-- Recreate catalog_products view without quantity column
CREATE VIEW catalog_products AS
SELECT 
    p.id,
    p.name,
    p.description,
    p.long_description,
    p.category as category_id,
    c.name as category_name,
    COALESCE(main_img.filename, '') as image_url,
    p.available,
    p.slug,
    p.search_vector,
    -- Aggregate gallery images as JSON array
    COALESCE(
        (
            SELECT json_agg(i.filename ORDER BY i.filename)
            FROM public.images_products ip
            JOIN public.images i ON ip.image_id = i.id
            WHERE ip.product_id = p.id
        ),
        '[]'::json
    ) as images
FROM public.products p
LEFT JOIN public.categories c ON p.category = c.id
LEFT JOIN public.images main_img ON p.main_img = main_img.id
ORDER BY p.name;

-- +goose StatementEnd
