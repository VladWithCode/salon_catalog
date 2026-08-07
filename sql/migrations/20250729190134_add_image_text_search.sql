-- +goose Up
-- +goose StatementBegin

-- Add tsvector column for full-text search on images
ALTER TABLE images ADD COLUMN search_vector tsvector;

-- Create function to update image search vector
CREATE OR REPLACE FUNCTION update_image_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    -- Combine image name and filename (name has higher weight)
    NEW.search_vector := 
        setweight(to_tsvector('spanish', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('spanish', COALESCE(NEW.filename, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update search vector
CREATE TRIGGER update_images_search_vector
    BEFORE INSERT OR UPDATE ON images
    FOR EACH ROW EXECUTE FUNCTION update_image_search_vector();

-- Update existing records with search vectors
UPDATE images SET search_vector = 
    setweight(to_tsvector('spanish', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('spanish', COALESCE(filename, '')), 'B');

-- Create GIN index for fast full-text search
CREATE INDEX idx_images_search_vector ON images USING gin(search_vector);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop index
DROP INDEX IF EXISTS idx_images_search_vector;

-- Drop trigger
DROP TRIGGER IF EXISTS update_images_search_vector ON images;

-- Drop function
DROP FUNCTION IF EXISTS update_image_search_vector();

-- Drop column
ALTER TABLE images DROP COLUMN IF EXISTS search_vector;

-- +goose StatementEnd
