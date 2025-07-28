-- +goose Up
-- +goose StatementBegin

-- Add tsvector columns for full-text search
ALTER TABLE products ADD COLUMN search_vector tsvector;
ALTER TABLE categories ADD COLUMN search_vector tsvector;

-- Create function to update product search vector
CREATE OR REPLACE FUNCTION update_product_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Combine product name, description with category name (with different weights)
    NEW.search_vector := 
        setweight(to_tsvector('spanish', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('spanish', COALESCE(NEW.description, '')), 'B') ||
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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

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
