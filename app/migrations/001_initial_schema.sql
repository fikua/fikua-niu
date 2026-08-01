-- +goose Up
CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  name          TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  avatar_emoji  TEXT NOT NULL DEFAULT '🐦',
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
  token_hash  TEXT PRIMARY KEY,
  user_id     INTEGER NOT NULL REFERENCES users(id),
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at  TIMESTAMP NOT NULL
);

CREATE TABLE items (
  id               INTEGER PRIMARY KEY,
  name             TEXT NOT NULL,
  name_normalized  TEXT NOT NULL,
  location         TEXT NOT NULL CHECK (location IN ('shopping','pantry')),
  position         REAL NOT NULL,
  added_by         INTEGER REFERENCES users(id),
  moved_by         INTEGER REFERENCES users(id),
  moved_at         TIMESTAMP,
  created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- EC-06: unicitat sobre el nom normalitzat (retallat + minúscules
-- Unicode-aware, calculat a Go — vegeu ADR-02), a través de totes dues
-- caixes (per això NO inclou `location` a l'índex).
CREATE UNIQUE INDEX idx_items_name_normalized ON items(name_normalized);

CREATE TABLE events (
  id         INTEGER PRIMARY KEY,
  user_id    INTEGER REFERENCES users(id),
  kind       TEXT NOT NULL,
  payload    TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE events;
DROP TABLE items;
DROP TABLE sessions;
DROP TABLE users;
