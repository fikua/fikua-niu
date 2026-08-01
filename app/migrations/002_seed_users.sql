-- +goose Up
-- Usuaris placeholder (S11 — cap dada real committejada). Hash bcrypt
-- d'una contrasenya arbitrària que mai s'utilitzarà mentre l'auth sigui
-- stubbed; NIU-4 reemplaça aquestes files via variables d'entorn a
-- l'arrencada (upsert), no via una nova migració.
INSERT INTO users (id, name, display_name, password_hash, avatar_emoji)
VALUES
  (1, 'usuari_a', 'Usuari A', '$2a$12$placeholderplaceholderplaceholderplaceholde', '🐦'),
  (2, 'usuari_b', 'Usuari B', '$2a$12$placeholderplaceholderplaceholderplaceholde', '🦊');

-- +goose Down
DELETE FROM users WHERE id IN (1, 2);
