-- +goose Up
-- +goose StatementBegin

-- View for catalog categories with product counts
CREATE VIEW catalog_categories AS
SELECT 
    c.id,
    c.name,
    COUNT(p.id) as product_count
FROM categories c
LEFT JOIN products p ON c.id = p.category
GROUP BY c.id, c.name
ORDER BY c.name;

-- View for catalog products with all related data
CREATE VIEW catalog_products AS
SELECT 
    p.id,
    p.name,
    p.description,
    p.description as long_description, -- You may want to add a separate long_description column later
    p.category as category_id,
    c.name as category_name,
    COALESCE(main_img, '') as image_url,
    p.available,
    -- Aggregate gallery images as JSON array
    COALESCE(
        (
            SELECT json_agg(i.filename ORDER BY i.filename)
            FROM images_products ip
            JOIN images i ON ip.image_id = i.id
            WHERE ip.product_id = p.id
        ),
        '[]'::json
    ) as images,
    -- Transform features JSONB to specifications array format
    COALESCE(
        (
            SELECT json_agg(
                json_build_object(
                    'name', key,
                    'value', value::text
                )
            )
            FROM jsonb_each_text(p.features)
        ),
        '[]'::json
    ) as specifications
FROM products p
LEFT JOIN categories c ON p.category = c.id
LEFT JOIN images main_img ON p.main_img = main_img.id
ORDER BY p.name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS catalog_products;
DROP VIEW IF EXISTS catalog_categories;
-- +goose StatementEnd
