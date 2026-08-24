-- +goose Up

ALTER TABLE saved_titles
ADD COLUMN url TEXT;

ALTER TABLE chapters
ADD COLUMN url TEXT;

ALTER TABLE chapters
DROP COLUMN language;


-- +goose Down

ALTER TABLE chapters
ADD COLUMN language TEXT;

ALTER TABLE chapters
DROP COLUMN url;

ALTER TABLE saved_titles
DROP COLUMN url;
