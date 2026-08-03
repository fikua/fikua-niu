-- +goose Up
CREATE TABLE activity_ideas (
  id              INTEGER PRIMARY KEY,
  url             TEXT NOT NULL,
  title           TEXT,
  image_url       TEXT,
  description     TEXT,
  preview_status  TEXT NOT NULL CHECK (preview_status IN ('pending','ready','partial','failed')) DEFAULT 'pending',
  added_by        INTEGER REFERENCES users(id),
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- EC-06: cap índex únic sobre url — deduplicació explícitament fora
-- d'abast; dues files amb el mateix enllaç són vàlides.

-- +goose Down
DROP TABLE activity_ideas;
