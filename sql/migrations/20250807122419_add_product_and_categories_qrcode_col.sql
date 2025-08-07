-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN qrcode_filename TEXT DEFAULT '';
ALTER TABLE categories ADD COLUMN qrcode_filename TEXT DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP COLUMN qrcode_filename;
ALTER TABLE categories DROP COLUMN qrcode_filename;
-- +goose StatementEnd
