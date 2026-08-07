-- +goose Up
-- +goose StatementBegin

-- Enable pg_trgm extension for text similarity (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create a materialized view for faster similarity calculations
CREATE MATERIALIZED VIEW product_similarities AS
WITH product_terms AS (
    -- Extract the most important terms from each product for comparison
    SELECT 
        p.id,
        p.category,
        p.name,
        p.description,
        -- Combine searchable text for trigram similarity
        LOWER(p.name || ' ' || COALESCE(p.description, '') || ' ' || COALESCE(p.long_description, '')) as searchable_text
    FROM products p
    WHERE p.available = true
)
SELECT 
    p1.id as product_id,
    p2.id as related_id,
    -- Calculate similarity score using multiple strategies
    (
        -- 1. Same category gets the highest weight (0.4)
        CASE WHEN p1.category = p2.category THEN 0.4 ELSE 0 END +
        
        -- 2. Text similarity using trigram similarity (0.4 weight)
        -- This compares the overall text content between products
        COALESCE(
            similarity(p1.searchable_text, p2.searchable_text) * 0.4,
            0
        ) +
        
        -- 3. Name similarity bonus (0.2 weight)
        -- Extra weight for products with similar names
        COALESCE(
            similarity(p1.name, p2.name) * 0.2,
            0
        )
    ) as similarity_score
FROM product_terms p1
CROSS JOIN product_terms p2
WHERE p1.id != p2.id;

-- Create indexes for fast lookups
CREATE INDEX idx_product_similarities_lookup 
    ON product_similarities(product_id, similarity_score DESC);

CREATE INDEX idx_product_similarities_related
    ON product_similarities(related_id);

-- Create unique index to allow CONCURRENTLY refresh
CREATE UNIQUE INDEX idx_product_similarities_unique 
    ON product_similarities(product_id, related_id);

-- Refresh function (call this periodically or after product updates)
CREATE OR REPLACE FUNCTION refresh_product_similarities()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY product_similarities;
END;
$$ LANGUAGE plpgsql;

-- Alternative: Simpler version without pg_trgm extension
-- Use this if pg_trgm is not available or you prefer a simpler approach
CREATE OR REPLACE VIEW product_similarities_simple AS
SELECT 
    p1.id as product_id,
    p2.id as related_id,
    -- Simple scoring based on category and searchable terms
    (
        -- Same category
        CASE WHEN p1.category = p2.category THEN 0.6 ELSE 0 END +
        
        -- Check for common words in names (simple approach)
        CASE 
            WHEN p1.name ILIKE '%' || split_part(p2.name, ' ', 1) || '%' 
                AND LENGTH(split_part(p2.name, ' ', 1)) > 3 THEN 0.2
            ELSE 0
        END +
        
        -- Check if they share search vector terms (using text search)
        CASE 
            WHEN p1.search_vector @@ plainto_tsquery('spanish', p2.name) THEN 0.2
            ELSE 0
        END
    ) as similarity_score
FROM products p1
CROSS JOIN products p2
WHERE p1.id != p2.id
    AND p2.available = true
    AND p1.category = p2.category;  -- Only compare within categories for performance

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS product_similarities CASCADE;
DROP VIEW IF EXISTS product_similarities_simple CASCADE;
DROP FUNCTION IF EXISTS refresh_product_similarities();
-- +goose StatementEnd
