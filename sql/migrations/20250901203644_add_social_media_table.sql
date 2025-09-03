-- +goose Up
-- +goose StatementBegin
CREATE TABLE social_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(512) NOT NULL,
    link VARCHAR(1024) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_timestamp BEFORE UPDATE ON social_links 
    FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- This table will serve to add dynamic sections where we can add social links later.
-- For now they'll be "hard created" in the database.
CREATE TABLE social_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(512) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE social_links_sections (
    link_id UUID REFERENCES social_links(id) NOT NULL,
    section_id UUID REFERENCES social_sections(id) NOT NULL,
    icon_id UUID REFERENCES images(id) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (link_id, section_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE social_links;
-- +goose StatementEnd
