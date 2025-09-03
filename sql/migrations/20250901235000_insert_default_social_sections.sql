-- +goose Up
-- +goose StatementBegin
INSERT INTO social_sections (id, name) 
SELECT gen_random_uuid(), 'footer'
WHERE NOT EXISTS (SELECT 1 FROM social_sections WHERE name = 'footer');

INSERT INTO social_sections (id, name) 
SELECT gen_random_uuid(), 'mobile_nav'
WHERE NOT EXISTS (SELECT 1 FROM social_sections WHERE name = 'mobile_nav');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM social_sections WHERE name IN ('footer', 'mobile_nav');
-- +goose StatementEnd