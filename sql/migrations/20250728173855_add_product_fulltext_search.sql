-- +goose Up
-- +goose StatementBegin

-- Add tsvector columns for full-text search
ALTER TABLE products ADD COLUMN search_vector tsvector;
ALTER TABLE categories ADD COLUMN search_vector tsvector;

-- Create function to update product search vector
CREATE OR REPLACE FUNCTION update_product_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Combine product name, description, long_description with category name (with different weights)
    NEW.search_vector := 
        setweight(to_tsvector('spanish', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('spanish', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('spanish', COALESCE(NEW.long_description, '')), 'B') ||
        setweight(to_tsvector('spanish', COALESCE(
            (SELECT name FROM categories WHERE id = NEW.category), ''
        )), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create function to update category search vector
CREATE OR REPLACE FUNCTION update_category_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('spanish', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('spanish', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers to automatically update search vectors
CREATE TRIGGER update_products_search_vector
    BEFORE INSERT OR UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_product_search_vector();

CREATE TRIGGER update_categories_search_vector
    BEFORE INSERT OR UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_category_search_vector();

-- Update existing records
UPDATE products SET search_vector = 
    setweight(to_tsvector('spanish', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('spanish', COALESCE(description, '')), 'B') ||
    setweight(to_tsvector('spanish', COALESCE(long_description, '')), 'B') ||
    setweight(to_tsvector('spanish', COALESCE(
        (SELECT name FROM categories WHERE id = category), ''
    )), 'C');

UPDATE categories SET search_vector = 
    setweight(to_tsvector('spanish', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('spanish', COALESCE(description, '')), 'B');

-- Create GIN indexes for fast full-text search
CREATE INDEX idx_products_search_vector ON products USING gin(search_vector);
CREATE INDEX idx_categories_search_vector ON categories USING gin(search_vector);

-- Create index for category filtering (if not exists)
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

-- Drop existing catalog views to recreate them with search vectors
DROP VIEW IF EXISTS catalog_products;
DROP VIEW IF EXISTS catalog_categories;

-- Recreate catalog_categories view with search vector
CREATE VIEW catalog_categories AS
SELECT 
    c.id,
    c.name,
    c.search_vector,
    COUNT(p.id) as product_count
FROM categories c
LEFT JOIN products p ON c.id = p.category
GROUP BY c.id, c.name, c.search_vector
ORDER BY c.name;

-- Recreate catalog_products view with search vector
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
    p.search_vector,
    -- Aggregate gallery images as JSON array
    COALESCE(
        (
            SELECT json_agg(i.filename ORDER BY i.filename)
            FROM images_products ip
            JOIN images i ON ip.image_id = i.id
            WHERE ip.product_id = p.id
        ),
        '[]'::json
    ) as images
FROM products p
LEFT JOIN categories c ON p.category = c.id
LEFT JOIN images main_img ON p.main_img = main_img.id
ORDER BY p.name;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop views
DROP VIEW IF EXISTS catalog_products;
DROP VIEW IF EXISTS catalog_categories;

-- Recreate original views without search vectors
CREATE VIEW catalog_categories AS
SELECT 
    c.id,
    c.name,
    COUNT(p.id) as product_count
FROM categories c
LEFT JOIN products p ON c.id = p.category
GROUP BY c.id, c.name
ORDER BY c.name;

CREATE VIEW catalog_products AS
SELECT 
    p.id,
    p.name,
    p.description,
    p.description as long_description,
    p.category as category_id,
    c.name as category_name,
    COALESCE(main_img.filename, '') as image_url,
    p.available,
    COALESCE(
        (
            SELECT json_agg(i.filename ORDER BY i.filename)
            FROM images_products ip
            JOIN images i ON ip.image_id = i.id
            WHERE ip.product_id = p.id
        ),
        '[]'::json
    ) as images
FROM products p
LEFT JOIN categories c ON p.category = c.id
LEFT JOIN images main_img ON p.main_img = main_img.id
ORDER BY p.name;

-- Drop indexes
DROP INDEX IF EXISTS idx_products_search_vector;
DROP INDEX IF EXISTS idx_categories_search_vector;

-- Drop triggers
DROP TRIGGER IF EXISTS update_products_search_vector ON products;
DROP TRIGGER IF EXISTS update_categories_search_vector ON categories;

-- Drop functions
DROP FUNCTION IF EXISTS update_product_search_vector();
DROP FUNCTION IF EXISTS update_category_search_vector();

-- Drop columns
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
ALTER TABLE categories DROP COLUMN IF EXISTS search_vector;

-- +goose StatementEnd
