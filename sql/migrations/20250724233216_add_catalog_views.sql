-- +goose Up
-- +goose StatementBegin

-- View for catalog categories with product counts
CREATE VIEW catalog_categories AS
SELECT 
    c.id,
    c.name,
    COUNT(p.id) as product_count
FROM public.categories c
LEFT JOIN public.products p ON c.id = p.category
GROUP BY c.id, c.name
ORDER BY c.name;

-- View for catalog products with all related data
CREATE VIEW catalog_products AS
SELECT 
    p.id,
    p.name,
    p.description,
    p.long_description,
    p.category as category_id,
    p.slug,
    c.name as category_name,
    COALESCE(main_img.filename, '') as image_url,
    p.available,
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
DROP VIEW IF EXISTS catalog_products;
DROP VIEW IF EXISTS catalog_categories;
-- +goose StatementEnd
