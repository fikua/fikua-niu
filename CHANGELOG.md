# Changelog

Tots els canvis destacables d'aquest projecte es documenten en aquest fitxer.

El format segueix [Keep a Changelog](https://keepachangelog.com/ca/1.1.0/),
i el projecte s'adhereix a [Semantic Versioning](https://semver.org/lang/ca/).

## [Unreleased]

### Added

- Brief mestre (`PLAN.md`) amb arquitectura, seguretat, desplegament i pla de proves.
- Manifest `fikua.yml` per al framework Fikua SDD.
- Backlog inicial amb quatre items (NIU-1 … NIU-4).
- **NIU-1 — Llista de la compra ↔ rebost (auth stubbed):** implementació
  completa del backend Go (`internal/config`, `internal/store`,
  `internal/items`, `internal/auth`, `internal/httpapi`) i del frontend
  vanilla (`app/web/`) amb FLIP, actualització optimista, confeti d'un
  sol tret, pestanyes mòbil i anuncis `aria-live`. Migracions goose
  embedded, SQLite pur Go (`modernc.org/sqlite`), normalització NFC de
  noms d'ítem (ADR-02), seguretat S3/S7/S8, i seam d'autenticació
  (`auth.Authenticator`) llest per NIU-4. Suite de tests: unitaris i
  d'integració en Go (26 tests) + E2E Playwright (11 tests, inclou
  auditoria axe-core WCAG 2.2 AA) + procediment manual `killtest`
  (10 execucions de `SIGKILL` sense corrupció, ADR-04/REL-01).
- **NIU-4 — Autenticació amb usuari i contrasenya:** `auth.PasswordAuthenticator`
  (bcrypt cost 12) substitueix `StubAuthenticator` darrere de la mateixa
  interfície `auth.Authenticator` (ADR-03 NIU-1). Sessió opaca de 256
  bits (`crypto/rand`), TTL de 30 dies, mai persistida en clar (només
  `SHA-256(token)`, AC-08). Resistència a enumeració d'usuaris (ADR-02):
  `bcrypt.CompareHashAndPassword` es crida sempre, contra un hash dummy
  precalculat si l'usuari no existeix — cos d'error byte-idèntic per a
  usuari inexistent i contrasenya incorrecta (AC-11/S5). Rate limiting en
  memòria (`internal/auth/ratelimiter.go`, ADR-01): 10 intents fallits/5min
  per usuari normalitzat, 20/5min per IP (`Cf-Connecting-Ip`). CSRF de
  doble-submit sense taula nova (ADR-05): cookie `niu_csrf` no-`HttpOnly`
  amb un HMAC derivat del `token_hash` de la sessió, verificat pel nou
  middleware `RequireCSRF` — muntat, per primer cop, sobre les rutes de
  mutació d'`/api/v1/items` ja enviades a producció a NIU-1 (retrofit de
  seguretat, `items_handlers.go` sense canvis de forma). Neteja de
  sessions expirades i de `buckets` de rate limiting via una única
  goroutine amb `time.Ticker` horari (ADR-04). Credencials sembrades a
  l'arrencada des de `NIU_SESSION_SECRET`/`NIU_USER_*_HASH`/`NIU_USER_*_NAME`/
  `NIU_USER_*_DISPLAY` amb validació fail-fast a `config.Load()` (EC-12) —
  cap secret ni hash real en cap fitxer committejat (S11/NFR-09). Pantalla
  de login nova (`web/login.html`, `web/js/auth.js`) i redirecció
  centralitzada a `401` (`web/js/api.js` `handleUnauthenticated`,
  `web/js/main.js`). Suite de tests: 76 tests Go (unitaris + integració,
  inclou els 7 casos 🟢 NIU-4 del pla de proves S1b/S2a/S2b/S2c/S4/S5/S6) +
  19 tests E2E Playwright (18 de NIU-1 sense regressions + 1 de cicle
  complet login → ús → logout, AC-14).
