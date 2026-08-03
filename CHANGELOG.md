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
- **NIU-5 — Compres grans i projectes de casa:** nou espai, separat de la
  llista de la compra, amb un cicle de vida de tres estats (`idea` →
  `decidit` → `fet`, reversible en totes dues direccions) i pressupost/data
  objectiu opcionals. Domini nou `internal/projects` (ADR-01), estructural-
  ment paral·lel a `internal/items` — reutilitza `items.NormalizeName` per
  a la normalització de duplicats (ADR-02), amb un abast més ampli: un
  duplicat es rebutja **a través de tots els estats**, no només entre
  caixes actives. Migració goose `003_projects.sql` (taula `projects`,
  índex únic sobre `name_normalized`, `CHECK` de tres estats). Mateix patró
  `BEGIN IMMEDIATE` que NIU-1 per als canvis d'estat concurrents (ADR-01
  NIU-1, cap error 5xx, última escriptura guanya). Nous endpoints
  `GET/POST /api/v1/projects` i `PATCH/DELETE /api/v1/projects/{id}`, dins
  del mateix grup `/api/v1` protegit per `WithCurrentUser`/`RequireCSRF`
  (cap superfície d'autenticació nova). Nova secció de navegació ("🏠
  Projectes", `web/projects.html`) amb accent terracota (en lloc del verd
  molsa de la llista de la compra, ADR-04), sense animació de transició
  d'estat (ADR-03, NFR-08 no aplicable) — llista única amb selector de
  tres opcions per canviar d'estat en qualsevol direcció, avatars d'autoria
  i regió `aria-live` amb el format "{nom} ara està {estat}". `overview.md`
  actualitzat com a font única de veritat (AC-13). Suite de tests: 112
  tests Go en total (33 nous — 16 unitaris de validació/normalització a
  `internal/projects`, 17 d'integració d'estat/concurrència/esdeveniments/
  seguretat a `tests/integration/`) + 26 tests E2E Playwright (23 de
  NIU-1/NIU-4 sense regressions + 3 nous — diferenciació visual, teclat/
  aria-live/estat buit/mòbil, auditoria axe-core i XSS aplicats a l'espai
  de projectes).
- **NIU-6 — Idees d'activitats amb previsualització de link:** tercer
  espai, separat de la compra i dels projectes, per desar enllaços
  d'activitats amb previsualització automàtica (títol/imatge/descripció
  Open Graph). Domini nou `internal/ideas` (ADR-01), sense cicle de vida
  ni deduplicació d'enllaços (EC-06). **Component central de seguretat:**
  `internal/fetchsafe` (ADR-02, esmenat post-revisió de seguretat —
  F-01 a F-07), l'única porta d'entrada de tot Niu a peticions HTTP cap a
  una URL introduïda per l'usuari — validació d'esquema `http(s)` abans de
  qualsevol xarxa, denylist de noms d'amfitrió (`niu.fikua.com` +
  `NIU_PUBLIC_HOST` + serveis de `traefik-public`) independent de la
  validació d'IP, `net.Dialer.ControlContext` com a únic punt de validació
  d'IP amb criteri d'allowlist (`IsGlobalUnicast() && !IsPrivate()`) post-
  `Unmap()` (rebuig explícit de formes IPv4-mapejades-a-IPv6 i de prefixos
  NAT64/6to4), `DisableKeepAlives` + revalidació d'esquema a
  `CheckRedirect` (defensa en dues capes contra el bypass de connexió
  reutilitzada), timeout dur de 5s, límit de 2 MiB en streaming, client
  HTTP dedicat sense cap credencial de Niu. Scraping asíncron amb worker
  pool acotat de 6 workers (ADR-03, esmenat — límit de concurrència/
  memòria): `POST /api/v1/ideas` respon `201` immediatament amb la idea en
  `pending`, la previsualització es resol en segon pla i mai bloqueja la
  petició original. Parsing Open Graph amb `golang.org/x/net/html`
  (ADR-04, sense dependència de tercers). Nova entrada de navegació
  ("💡 Idees") amb l'accent `--color-mel` i graella de targetes de quatre
  estats (completa/fallback/parcial/pendent) mapejats a `preview_status`.
  `overview.md` actualitzat. Suite de tests: 171 tests Go en total (59
  nous — validació/parsing/worker pool a `internal/ideas`/`internal/
  fetchsafe`, i a `tests/integration/` un conjunt dedicat de regressió
  SSRF contra un servidor de test real amb comptador de connexions TCP
  acceptades, incloent els dos casos de regressió F-01 (redirecció
  mateix-host) i F-02 (adreça IPv4-mapejada-a-IPv6) explícitament exigits
  per `security-engineer`) + 33 tests E2E Playwright (22 existents sense
  regressions + 11 nous — diferenciació visual, teclat, targeta accessible
  amb fallback, `aria-live`, viewport mòbil i auditoria axe-core aplicats
  a l'espai d'idees).
