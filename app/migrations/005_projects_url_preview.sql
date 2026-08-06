-- +goose Up
ALTER TABLE projects ADD COLUMN url TEXT;
ALTER TABLE projects ADD COLUMN title TEXT;
ALTER TABLE projects ADD COLUMN image_url TEXT;
ALTER TABLE projects ADD COLUMN description TEXT;
-- preview_status is NULL-able here, unlike activity_ideas' NOT NULL
-- DEFAULT 'pending' (tasks.md T-01): every project that already existed
-- before this migration, and every project created afterwards without a
-- url, has no preview pending at all — 'pending' for them would be a lie
-- and would show the UI an eternal spinner for a project that will never
-- resolve.
ALTER TABLE projects ADD COLUMN preview_status TEXT CHECK (preview_status IN ('pending','ready','partial','failed'));

-- +goose Down
ALTER TABLE projects DROP COLUMN preview_status;
ALTER TABLE projects DROP COLUMN description;
ALTER TABLE projects DROP COLUMN image_url;
ALTER TABLE projects DROP COLUMN title;
ALTER TABLE projects DROP COLUMN url;
