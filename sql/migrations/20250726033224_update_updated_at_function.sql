-- +goose Up
-- +goose StatementBegin

-- Create a reusable function that updates the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose StatementEnd
