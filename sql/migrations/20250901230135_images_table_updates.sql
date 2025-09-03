-- +goose Up
-- +goose StatementBegin
ALTER TABLE images 
    ADD COLUMN file_type VARCHAR(120) NOT NULL DEFAULT 'image/jpeg',
    ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX images_filetype_idx ON images (file_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE images;
-- +goose StatementEnd
