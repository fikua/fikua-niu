-- +goose Up
CREATE TABLE projects (
  id               INTEGER PRIMARY KEY,
  name             TEXT NOT NULL,
  name_normalized  TEXT NOT NULL,
  state            TEXT NOT NULL CHECK (state IN ('idea','decidit','fet')) DEFAULT 'idea',
  budget           TEXT,
  target_date      TEXT,                     -- ISO-8601 YYYY-MM-DD, NULL si no informada
  added_by         INTEGER REFERENCES users(id),
  last_updated_by  INTEGER REFERENCES users(id),
  created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- EC-03: unicitat sobre el nom normalitzat a través de TOTS els estats
-- (a diferència de l'índex d'items, aquest NO necessita cap clàusula
-- addicional perquè no hi ha soft-delete ni distinció d'estat a excloure —
-- un DELETE dur ja treu la fila de la comprovació, EC-04).
CREATE UNIQUE INDEX idx_projects_name_normalized ON projects(name_normalized);

-- +goose Down
DROP TABLE projects;
