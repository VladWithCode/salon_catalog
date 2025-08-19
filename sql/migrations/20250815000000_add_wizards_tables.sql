-- +goose Up
-- +goose StatementBegin
CREATE TABLE wizards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description VARCHAR(512),
    event_kind_id UUID REFERENCES event_kinds(id) ON DELETE RESTRICT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER wizards_set_updated_at BEFORE UPDATE ON wizards
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

CREATE TABLE wizard_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description VARCHAR(512),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    step_order INTEGER NOT NULL DEFAULT 1,
    multi_select BOOLEAN NOT NULL DEFAULT FALSE,
    min_selected INTEGER NOT NULL DEFAULT 0,
    max_selected INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER wizard_steps_set_updated_at BEFORE UPDATE ON wizard_steps
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

CREATE TABLE wizard_steps_wizards (
    wizard_id UUID REFERENCES wizards(id) ON DELETE CASCADE,
    wizard_step_id UUID REFERENCES wizard_steps(id) ON DELETE CASCADE,
    -- This columns are for custom settings per wizard's step (that is, the related wizard step not the standalone step)
    required BOOLEAN,
    step_order INTEGER,
    multi_select BOOLEAN,
    min_selected INTEGER,
    max_selected INTEGER,
    PRIMARY KEY (wizard_id, wizard_step_id)
);

CREATE TABLE wizard_step_categories (
    wizard_step_id UUID REFERENCES wizard_steps(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (wizard_step_id, category_id)
);

-- Create indexes for better performance
CREATE INDEX idx_wizards_event_kind_id ON wizards(event_kind_id);

-- Add search vectors for full-text search (optional, following product pattern)
ALTER TABLE wizards ADD COLUMN search_vector tsvector;

CREATE OR REPLACE FUNCTION update_wizard_search_vector() RETURNS trigger AS $$
BEGIN
    NEW.search_vector := to_tsvector('spanish', COALESCE(NEW.name, ''));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER wizards_search_vector_update 
    BEFORE INSERT OR UPDATE ON wizards
    FOR EACH ROW EXECUTE PROCEDURE update_wizard_search_vector();

CREATE INDEX idx_wizards_search_vector ON wizards USING gin(search_vector);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS wizards_search_vector_update ON wizards;
DROP FUNCTION IF EXISTS update_wizard_search_vector();
DROP INDEX IF EXISTS idx_wizards_search_vector;
DROP INDEX IF EXISTS idx_wizard_steps_order;
DROP INDEX IF EXISTS idx_wizard_steps_wizard_id;
DROP INDEX IF EXISTS idx_wizards_event_kind_id;

DROP TRIGGER IF EXISTS wizard_steps_set_updated_at ON wizard_steps;
DROP TRIGGER IF EXISTS wizards_set_updated_at ON wizards;

DROP TABLE IF EXISTS wizard_step_categories;
DROP TABLE IF EXISTS wizard_steps;
DROP TABLE IF EXISTS wizards;
-- +goose StatementEnd
