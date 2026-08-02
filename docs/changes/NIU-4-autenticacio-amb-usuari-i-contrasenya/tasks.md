---
artefact: tasks
key: "NIU-4"
title: "Autenticació amb usuari i contrasenya"
status: "approved"
owner: "task-planner"
design_path: "./design.md"
requirements_path: "./requirements.md"
task_count: 37
ac_coverage:
  - ac: "AC-01"
    tasks: ["T-12", "T-20", "T-21", "T-26", "T-37"]
  - ac: "AC-02"
    tasks: ["T-12", "T-26", "T-37"]
  - ac: "AC-03"
    tasks: ["T-12", "T-26", "T-37"]
  - ac: "AC-04"
    tasks: ["T-13", "T-21", "T-27", "T-37"]
  - ac: "AC-05"
    tasks: ["T-11", "T-16", "T-22", "T-23", "T-27", "T-37"]
  - ac: "AC-06"
    tasks: ["T-15", "T-16", "T-18", "T-22", "T-30", "T-37"]
  - ac: "AC-07"
    tasks: ["T-15", "T-16", "T-18", "T-30", "T-37"]
  - ac: "AC-08"
    tasks: ["T-10", "T-28", "T-37"]
  - ac: "AC-09"
    tasks: ["T-10", "T-13", "T-27", "T-37"]
  - ac: "AC-10"
    tasks: ["T-06", "T-12", "T-24", "T-29", "T-37"]
  - ac: "AC-11"
    tasks: ["T-07", "T-12", "T-17", "T-25", "T-26", "T-37"]
  - ac: "AC-12"
    tasks: ["T-10", "T-11", "T-14", "T-32", "T-37"]
  - ac: "AC-13"
    tasks: ["T-02", "T-03", "T-19", "T-33", "T-37"]
  - ac: "AC-14"
    tasks: ["T-36", "T-37"]
ec_coverage:
  - ec: "EC-01"
    tasks: ["T-27"]
  - ec: "EC-02"
    tasks: ["T-27"]
  - ec: "EC-03"
    tasks: ["T-15", "T-18", "T-30"]
  - ec: "EC-04"
    tasks: ["T-10", "T-26"]
  - ec: "EC-05"
    tasks: ["T-13", "T-27"]
  - ec: "EC-06"
    tasks: ["T-06", "T-24", "T-29"]
  - ec: "EC-07"
    tasks: ["T-06", "T-24", "T-29"]
  - ec: "EC-08"
    tasks: ["T-08", "T-31"]
  - ec: "EC-09"
    tasks: ["T-08", "T-31"]
  - ec: "EC-10"
    tasks: ["T-14", "T-32"]
  - ec: "EC-11"
    tasks: ["T-27"]
  - ec: "EC-12"
    tasks: ["T-03", "T-19", "T-33"]
nfr_coverage:
  - nfr: "NFR-01"
    tasks: ["T-01", "T-09", "T-34"]
  - nfr: "NFR-02"
    tasks: ["T-10"]
  - nfr: "NFR-03"
    tasks: ["T-09", "T-25"]
  - nfr: "NFR-04"
    tasks: ["T-03", "T-37"]
  - nfr: "NFR-05"
    tasks: ["T-04", "T-05", "T-06", "T-24"]
  - nfr: "NFR-06"
    tasks: ["T-34"]
  - nfr: "NFR-07"
    tasks: ["T-12", "T-35"]
  - nfr: "NFR-08"
    tasks: ["T-14", "T-32"]
  - nfr: "NFR-09"
    tasks: ["T-03", "T-37"]
sources:
  - "GitHub-style checklist (Markdown task lists)"
  - "Fikua AC↔tasks traceability matrix"
created: "2026-08-02"
updated: "2026-08-02"
---

# Tasks — Autenticació amb usuari i contrasenya

> **Què és això.** El full de ruta d'implementació per a NIU-4. Cada
> tasca és petita (≤ ~1h), autocontinguda, i traçada a almenys un
> criteri d'acceptació de `requirements.md`. **Cap tasca sense un AC/EC/
> NFR que la cobreixi; cap AC/EC/NFR sense almenys una tasca.** Aquest
> fitxer és l'únic artefacte mutable durant `/code` — la resta són
> bloquejats (`design.md`/`requirements.md` aprovats).
>
> Traducció mecànica de `design.md` (5 ADRs aprovats) — cap decisió de
> disseny nova. On `design.md` calla sobre un detall d'implementació que
> una tasca necessitaria, es marca a §7 (Preguntes obertes) en lloc de
> decidir-ho aquí.

## 1. Task list

### Foundations

- [x] **T-01** — Afegir la dependència `golang.org/x/crypto/bcrypt` a
  `app/go.mod`/`go.sum` (`go get golang.org/x/crypto/bcrypt`) · *covers:* NFR-01
- [x] **T-02** — Estendre `internal/config/config.go`: afegir camps
  `SessionSecret`, `UserAName`, `UserADisplay`, `UserAHash`, `UserBName`,
  `UserBDisplay`, `UserBHash` a `Config`, llegits amb
  `NIU_SESSION_SECRET`, `NIU_USER_A_NAME`, `NIU_USER_A_DISPLAY`,
  `NIU_USER_A_HASH`, `NIU_USER_B_NAME`, `NIU_USER_B_DISPLAY`,
  `NIU_USER_B_HASH` (`PLAN.md` §6) · *covers:* AC-13
- [x] **T-03** — Afegir validació fail-fast a `config.Load()`: error si
  falta qualsevol dels sis camps nous o si `len(SessionSecret) < 32`
  bytes; el procés no arrenca en cap cas parcial (ADR-03 NIU-1 pattern,
  `design.md` §2 punt 9) · *covers:* AC-13, EC-12, NFR-04, NFR-09
- [x] **T-04** — Crear `internal/auth/ratelimiter.go`: estructura
  `RateLimiter` amb `map[string]*bucket` protegit per `sync.Mutex`,
  `bucket{count int, windowStart time.Time}`; mètodes `Allow(key string,
  limit int) bool` i `RecordFailure(key string)` implementant la
  finestra fixa reiniciable de 5 minuts (ADR-01) · *covers:* NFR-05 (base)
- [x] **T-05** — Afegir a `RateLimiter` el mètode `Cleanup()` que esborra
  entrades del mapa amb `time.Since(windowStart) > 5*time.Minute`
  (ADR-01, reutilitzat pel ticker de T-18) · *covers:* NFR-05 (base)
- [x] **T-06** — Cablejar a `internal/auth` les dues claus independents
  del rate limiter (usuari normalitzat, llindar 10/5min; IP via
  `Cf-Connecting-Ip` amb fallback a `r.RemoteAddr`, llindar 20/5min),
  consultades totes dues abans de bcrypt (ADR-01) · *covers:* AC-10, EC-06, EC-07, NFR-05
- [x] **T-07** — Crear `internal/auth/password.go`: precalcular a la
  construcció de `PasswordAuthenticator` un `dummyHash` bcrypt fix (cost
  12) d'una contrasenya dummy embeguda com a constant, generat un sol cop
  (ADR-02) · *covers:* AC-11
- [x] **T-08** — Afegir a `internal/httpapi` la validació d'entrada pura
  del payload de login (`username != ""` post-`TrimSpace`, `password !=
  ""` sense trim) com a pas independent, previ a qualsevol crida al rate
  limiter (ADR-03, pas 2) · *covers:* EC-08, EC-09

### Implementation

- [x] **T-09** — Implementar `PasswordAuthenticator.Login(username,
  password string) (token string, err error)` a `internal/auth`: cerca
  d'usuari per nom normalitzat + crida `bcrypt.CompareHashAndPassword`
  **sempre** (contra `user.PasswordHash` o `dummyHash` de T-07, mai una
  branca condicional que la salti) (ADR-02) · *covers:* NFR-01, NFR-03
- [x] **T-10** — Implementar `CreateSession(userID int64) (token string,
  err error)` a `internal/auth`: genera 256 bits amb `crypto/rand`,
  calcula `SHA-256(token)`, `INSERT INTO sessions (token_hash, user_id,
  expires_at)` amb `expires_at = now + 30 dies`; el token en clar mai es
  persisteix, només viu a la resposta HTTP (`design.md` §5 Flux 1 pas 7) · *covers:* AC-08, AC-09, AC-12, EC-04, NFR-02
- [x] **T-11** — Implementar `PasswordAuthenticator.CurrentUser(r
  *http.Request) (auth.User, error)`: llegeix la cookie `niu_session`,
  calcula `SHA-256(token)`, la cerca a `sessions`, comprova `expires_at >
  now`; substitueix `StubAuthenticator` darrere de la mateixa interfície
  `auth.Authenticator` (ADR-03 NIU-1, `design.md` §4) · *covers:* AC-05, AC-12
- [x] **T-12** — Implementar el pipeline complet de `handleLogin` a nou
  fitxer `internal/httpapi/auth_handlers.go`, seguint l'ordre estricte
  d'ADR-03: (1) decodificació JSON → `400`; (2) validació d'entrada
  (T-08) → `400 validation_failed`, rate limiter no tocat; (3) consulta
  `RateLimiter.Allow` per usuari i per IP (T-06) → `429 rate_limited`
  sense cridar bcrypt; (4) `auth.Login` (T-09) → si falla, `RecordFailure`
  a totes dues claus + `401 invalid_credentials` amb cos idèntic
  (`design.md` §6.1); (5) èxit → `CreateSession` (T-10) + token CSRF
  (T-15) + `Set-Cookie` ×2 + `200` amb `{"user": {...}}`; registrar cada
  intent amb `slog.Info("login attempt", "username", ..., "result",
  ..., "ip", ...)` sense mai el valor de `password` (`design.md` §8) · *covers:* AC-01, AC-02, AC-03, AC-10, AC-11, NFR-07
- [x] **T-13** — Implementar `Logout(token string) error` a
  `internal/auth` (`DELETE FROM sessions WHERE token_hash = ?`) i
  `handleLogout` a `auth_handlers.go`: resol la sessió via
  `WithCurrentUser` (ja passa per aquest middleware), crida `auth.Logout`,
  respon `204` (`design.md` §5 Flux 3) · *covers:* AC-04, AC-09, EC-05
- [x] **T-14** — Afegir `CleanupExpired(ctx context.Context) error` a
  `internal/auth`: `DELETE FROM sessions WHERE expires_at <
  CURRENT_TIMESTAMP`, reutilitzant la mateixa connexió `*sql.DB` (ADR-04) · *covers:* AC-12, EC-10, NFR-08
- [x] **T-15** — Crear `internal/httpapi/csrf.go`: a l'èxit de login,
  generar un segon valor aleatori de 128 bits i codificar-lo com
  `Set-Cookie: niu_csrf=<hmac>; Secure; Path=/; SameSite=Strict;
  Max-Age=2592000` (sense `HttpOnly`), derivat com
  `HMAC-SHA256(NIU_SESSION_SECRET, token_hash)` en base64 URL-safe
  (ADR-05) — no es persisteix enlloc, es recalcula sota demanda · *covers:* AC-06, AC-07, EC-03
- [x] **T-16** — Implementar el middleware `RequireCSRF` a
  `internal/httpapi/csrf.go`: recalcula l'HMAC esperat a partir del
  `token_hash` de la sessió ja resolta per `WithCurrentUser` i el compara
  amb `hmac.Equal` contra la capçalera `X-CSRF-Token`; discrepància
  (absent, buida, no coincident) → `403 csrf_failed`, handler mai
  s'executa (ADR-05) · *covers:* AC-05, AC-06, AC-07
- [x] **T-17** — Afegir a `internal/httpapi/errors.go` els nous codis
  d'error de l'envelope `apiError`: `invalid_credentials`,
  `rate_limited`, `csrf_failed` (reutilitzar `validation_failed` i
  `unauthenticated` ja existents) amb els missatges exactes de
  `design.md` §6.1 · *covers:* AC-11
- [x] **T-18** — **Modificar `router.go`** (fitxer ja enviat a
  producció a NIU-1 — canvi quirúrgic, no reescriptura): registrar `POST
  /api/v1/auth/login` i `POST /api/v1/auth/logout` **fora** de
  `WithCurrentUser` per a login (encara no hi ha sessió) però `logout` sí
  hi passa; muntar `RequireCSRF` (T-16) al grup de rutes de mutació
  existent d'`/api/v1/items` (`POST`/`PATCH`/`DELETE`) i a
  `/api/v1/auth/logout`, mai a `/api/v1/auth/login` ni a cap `GET`
  (`design.md` §4, risc R-07). **`items_handlers.go` no es toca.** · *covers:* AC-06, AC-07, EC-03
- [x] **T-19** — **Modificar `cmd/niu/main.go`**: (1) després
  d'`store.Open`, executar dues `UPDATE users SET password_hash = ? WHERE
  name = ?` (Usuari A, Usuari B) dins una transacció amb els valors de
  `cfg`, verificant `RowsAffected == 1` per cada `UPDATE` i fallant
  l'arrencada si no (`design.md` §6.2, risc R-05); (2) substituir
  `auth.StubAuthenticator{UserID: seedUserAID}` per
  `auth.NewPasswordAuthenticator(st.DB, cfg.SessionSecret)`; (3) llançar
  la goroutine de neteja amb `time.NewTicker(1 * time.Hour)` que crida
  `CleanupExpired` (T-14) + una primera passada immediata a l'arrencada,
  aturant-se netament amb el `context.Context` de shutdown ja existent
  (ADR-04) · *covers:* AC-13, EC-12
- [x] **T-20** — Crear `app/web/login.html`: document separat
  d'`index.html`, mateix `<head>` (fonts self-hosted, `app.css`),
  formulari mínim (`username`, `password type="password"`, botó submit)
  reutilitzant les classes de `component-add-input.html` del
  design-system i un botó primari existent, `<label>` associades
  (`for`/`id`), sense maqueta dedicada (`design.md` §7/§8) · *covers:* AC-01
- [x] **T-21** — Crear `app/web/js/auth.js`: `login(username, password)`
  crida `fetch('/api/v1/auth/login', {credentials:'same-origin', ...})`,
  en `200` llegeix `?next=` (per defecte `/`) i hi redirigeix; en
  `401`/`429`/`400` mostra el `message` del cos d'error sota el
  formulari; `logout()` crida `POST /api/v1/auth/logout` amb la
  capçalera CSRF llegida de `document.cookie` i redirigeix a
  `/login.html` (`design.md` §7) · *covers:* AC-01, AC-04
- [x] **T-22** — Estendre `app/web/js/api.js`: afegir `getCsrfToken()`
  (llegeix `niu_csrf` de `document.cookie`, sense llibreria externa);
  cada wrapper de mutació (`addItem`, `moveItem`, `deleteItem`) inclou la
  capçalera `X-CSRF-Token`; afegir `handleUnauthenticated()` centralitzat
  que redirigeix a `/login.html?next=<ruta actual>`, cridat per **tots**
  els wrappers (mutació i lectura) en rebre `401` (`design.md` §7) · *covers:* AC-05, AC-06
- [x] **T-23** — Estendre `app/web/js/main.js`: a l'arrencada, abans de
  muntar la UI, crida `getMe()`; si `401`, redirigeix immediatament a
  `/login.html?next=/` sense renderitzar cap fila (evita parpelleig,
  `design.md` §7) · *covers:* AC-05

### Verification

- [x] **T-24** — Afegir tests unitaris a `internal/auth` per al rate
  limiter (`ratelimiter_test.go`): expiració de `bucket` (finestra de 5
  minuts es reinicia), independència de comptador per usuari vs. per IP
  (bloquejar un no bloqueja l'altre), i que l'11è intent es rebutja
  mentre el 10è encara passa · *covers:* AC-10, EC-06, EC-07, NFR-05
- [x] **T-25** — Afegir test unitari dedicat a `internal/auth`
  (`password_test.go`) per al mecanisme d'ADR-02: assert que
  `bcrypt.CompareHashAndPassword` es crida exactament una vegada tant si
  l'usuari existeix com si no (mitjançant mesura de temps d'execució
  amb marge ampli entre els dos camins, mateix criteri que NFR-03), i que
  la comparació contra `dummyHash` sempre retorna fals · *covers:* AC-11, NFR-03
- [x] **T-26** — Afegir tests d'integració a
  `tests/integration/auth_test.go` per AC-01/AC-02/AC-03/AC-11/S5: login
  amb credencials correctes obre sessió amb cookies `HttpOnly; Secure;
  Path=/; SameSite=Strict`; login amb contrasenya incorrecta i amb
  usuari inexistent produeixen cos de resposta byte-idèntic (llegit com
  a bytes crus, no camps individuals) · *covers:* AC-01, AC-02, AC-03, AC-11
- [x] **T-27** — Afegir tests d'integració per AC-04/AC-05/AC-09/EC-01/
  EC-02/EC-05/EC-11/S2a/S2b (a `auth_test.go`): petició sense cookie →
  `401`; cookie amb un caràcter mutat → `401` amb el mateix cos que sense
  cookie; logout invalida la sessió al servidor; reutilitzar token
  post-logout → `401`; sessió esborrada entremig → `401` · *covers:* AC-04, AC-05, AC-09, EC-01, EC-02, EC-05, EC-11
- [x] **T-28** — Afegir test d'integració per S2c (`auth_test.go`):
  login real, capturar el token en clar de la `Set-Cookie` de resposta,
  obrir `srv.Store.DB` (mateix patró que
  `TestSQLInjectionPayload_StoredLiterally_TableSurvives`), `SELECT
  token_hash FROM sessions`, assert que cap fila coincideix amb el token
  en clar i que `SHA256(token) == token_hash` · *covers:* AC-08
- [x] **T-29** — Afegir tests d'integració per AC-10/EC-06/EC-07/S4
  (`auth_test.go`): bucle de 10 intents fallits contra el mateix usuari
  seguit d'un onzè intent **amb la contrasenya correcta** → rebutjat
  igualment per `429 rate_limited` (no per credencials); cas simètric per
  llindar per IP (EC-06, atac contra usuaris diferents des de la mateixa
  procedència) i per usuari des de procedències diferents (EC-07) · *covers:* AC-10, EC-06, EC-07
- [x] **T-30** — Afegir tests d'integració per AC-06/AC-07/EC-03/S1b
  (`csrf_test.go`): mutació amb token CSRF vàlid → processada amb
  normalitat; mutació sense token o amb token arbitrari no emès pel
  servidor → `403 csrf_failed`, `GET` posterior confirma que no s'ha
  produït cap efecte; retrofit explícit sobre almenys `POST
  /api/v1/items` i un `PATCH`/`DELETE` addicional (codi ja enviat a
  producció a NIU-1, ara protegit per primer cop) · *covers:* AC-06, AC-07, EC-03
- [x] **T-31** — Afegir test d'integració per EC-08/EC-09 (`auth_test.go`):
  payload de login sense `password` o sense `username` (buit o absent)
  → `400 validation_failed`, i confirmar que el rate limiter no ha
  registrat cap intent (login posterior amb credencials correctes
  segueix funcionant sense estar limitat) · *covers:* EC-08, EC-09
- [x] **T-32** — Afegir test d'integració per AC-12/EC-10
  (`auth_test.go`): sembrar directament una sessió amb `expires_at` en
  el passat via `srv.Store.DB`, confirmar que una petició protegida amb
  aquell token es rebutja (`401`), i forçar l'execució de
  `CleanupExpired` (sense esperar el ticker real) confirmant que la fila
  desapareix de `sessions` · *covers:* AC-12, EC-10, NFR-08
- [x] **T-33** — Afegir test d'integració per AC-13/EC-12
  (`config_test.go` o `tests/integration/`): arrencada del procés (o
  crida directa a `config.Load()`) sense `NIU_SESSION_SECRET`, sense
  `NIU_USER_A_HASH`/`NIU_USER_B_HASH`, o amb `NIU_SESSION_SECRET` <32
  bytes → falla amb error clar, mai arrenca en estat parcial; amb totes
  les variables presents → els dos usuaris poden fer login amb èxit
  contra els hashos configurats · *covers:* AC-13, EC-12
- [x] **T-34** — Afegir test d'integració per NFR-01/NFR-06
  (`auth_test.go`): mesura directa del temps de resposta del login camí
  feliç — assert > 200ms (cost bcrypt real, no mockejat) i < 1s p95 en
  diverses repeticions · *covers:* NFR-01, NFR-06
- [x] **T-35** — Afegir test d'integració per NFR-07 (`auth_test.go`):
  capturar l'output de `log/slog` (mateix mecanisme ja usat a NIU-1)
  durant un intent de login fallit i un de limitat per rate limit,
  assert que el log conté usuari + resultat + IP i **mai** el valor de
  `password` ni el token en clar · *covers:* NFR-07
- [x] **T-36** — Afegir test E2E Playwright a
  `tests/e2e/specs/login-cycle.spec.js` per AC-14: cicle complet
  login (formulari real, credencials de test) → acció protegida (llistar
  ítems, ja visible després del login) → logout (botó/enllaç real) →
  assert que després del logout una recàrrega redirigeix a
  `/login.html`, sense cap error inesperat en cap pas · *covers:* AC-14
- [x] **T-37** — Executar `commands.test` (`cd app && go test ./...`),
  `commands.lint` (`gofmt -l`) i `commands.typecheck` (`cd app && go vet
  ./...`) del manifest; confirmar que els 7 casos 🟢 NIU-4 del pla de
  proves (S1b, S2a, S2b, S2c, S4, S5, S6, `docs/test-plan.md` §2.1/
  §2.1.1) estan verds a CI local, juntament amb la suite completa de
  NIU-1 (31 tests Go + 18 Playwright) sense regressions · *covers:* AC-01, AC-02, AC-03, AC-04, AC-05, AC-06, AC-07, AC-08, AC-09, AC-10, AC-11, AC-12, AC-13, AC-14, NFR-04, NFR-09

### Closing (universal — all changes)

- [ ] **C-01** — Append changelog entry (`docs.changelog` from manifest)
- [ ] **C-02** — Transition backlog item to `Human Review` via the adapter
- [ ] **C-03** — Propose semver bump (ASK USER — never apply unattended)

## 2. AC ↔ tasks traceability matrix

| AC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| AC-01 | Login amb credencials correctes obre sessió | T-12, T-20, T-21, T-26, T-37 |
| AC-02 | Login amb contrasenya incorrecta és rebutjat | T-12, T-26, T-37 |
| AC-03 | Login amb usuari inexistent és rebutjat | T-12, T-26, T-37 |
| AC-04 | Logout tanca la sessió activa | T-13, T-21, T-27, T-37 |
| AC-05 | Accés a endpoint protegit sense sessió es rebutja | T-11, T-16, T-22, T-23, T-27, T-37 |
| AC-06 | Mutacions requereixen token CSRF de doble-submit | T-15, T-16, T-18, T-22, T-30, T-37 |
| AC-07 | Mutació sense token CSRF es rebutja | T-15, T-16, T-18, T-30, T-37 |
| AC-08 | Token de sessió mai recuperable en clar de la BD | T-10, T-28, T-37 |
| AC-09 | Cada login emet token nou; logout l'invalida | T-10, T-13, T-27, T-37 |
| AC-10 | Força bruta contra el login es limita | T-06, T-12, T-24, T-29, T-37 |
| AC-11 | Error de login no distingeix usuari/contrasenya | T-07, T-12, T-17, T-25, T-26, T-37 |
| AC-12 | Sessió expira i deixa de ser vàlida | T-10, T-11, T-14, T-32, T-37 |
| AC-13 | Credencials sembrades des de config, no de codi/BD | T-02, T-03, T-19, T-33, T-37 |
| AC-14 | Cicle complet login → ús → logout | T-36, T-37 |

## 3. Edge cases ↔ tasks

| EC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| EC-01 | Cookie de sessió manipulada es rebutja (S2) | T-27 |
| EC-02 | Petició sense cap cookie es rebutja (S2) | T-27 |
| EC-03 | Falsificació de token CSRF amb valor conegut es rebutja (S1) | T-15, T-18, T-30 |
| EC-04 | Fixació de sessió — token pre-login queda inservible (S6) | T-10, T-26 |
| EC-05 | Reutilització del token després de logout (S6) | T-13, T-27 |
| EC-06 | Ratxa contra usuaris diferents des de la mateixa IP (S4) | T-06, T-24, T-29 |
| EC-07 | Ratxa contra el mateix usuari des d'IPs diferents (S4) | T-06, T-24, T-29 |
| EC-08 | Contrasenya buida/absent no consumeix intent | T-08, T-31 |
| EC-09 | Usuari buit/absent — error equivalent a AC-11 | T-08, T-31 |
| EC-10 | Sessions expirades s'esborren, no s'acumulen | T-14, T-32 |
| EC-11 | Cookie vàlida però sessió ja esborrada entremig | T-27 |
| EC-12 | Arrencada sense variables d'entorn requerides | T-03, T-19, T-33 |

## 4. NFRs ↔ tasks

| NFR | Statement (short) | Covering tasks |
|-----|--------------------|----------------|
| NFR-01 | bcrypt cost 12, >200ms per verificació | T-01, T-09, T-34 |
| NFR-02 | Token de sessió amb 256 bits d'entropia (CSPRNG) | T-10 |
| NFR-03 | Comparació de credencials/tokens resistent a temporització | T-09, T-25 |
| NFR-04 | Cap secret en fitxer committejat ni imatge publicada | T-03, T-37 |
| NFR-05 | Rate limiting amb backoff, mínim 10/usuari en finestra curta | T-04, T-05, T-06, T-24 |
| NFR-06 | Login camí feliç < 1s p95 malgrat bcrypt | T-34 |
| NFR-07 | Intents fallits registrats, mai la contrasenya en clar | T-12, T-35 |
| NFR-08 | Sessions expirades no s'acumulen indefinidament | T-14, T-32 |
| NFR-09 | Cap dada personal real en cap fitxer committejat | T-03, T-37 |

## 5. Out of scope (mirrored from design)

- Registre de nous usuaris.
- Recuperació o restabliment de contrasenya.
- Gestió d'usuaris (afegir, editar, desactivar comptes).
- Rols o permisos diferenciats entre Usuari A i Usuari B.
- Autenticació multifactor.
- Autenticació mòbil amb `Authorization: Bearer` (reservat per a un futur
  ítem si arriba una app mòbil).
- Cloudflare Access com a mecanisme d'autenticació o protecció addicional.
- Rotació automàtica de `NIU_SESSION_SECRET` en calent.
- Docker/compose/CI/CD/DNS (NIU-2).
- OTEL/observabilitat avançada (NIU-3) — només el logging bàsic de
  NFR-07 (T-12, T-35) és d'abast de NIU-4.

## 6. Notes for the developer

- **`items_handlers.go` no canvia de forma.** Només `router.go` (afegir
  rutes + muntar `RequireCSRF`) i `main.go` (wiring) es toquen entre el
  codi ja enviat a producció de NIU-1 — flag-ho explícitament al PR: és
  un retrofit de seguretat sobre codi existent, no una regressió.
- **Cap migració `003_*.sql` nova.** L'esquema `sessions`/
  `users.password_hash` ja existeix des de la migració 1 de NIU-1
  (`design.md` §6.2). El seed de credencials és un `UPDATE`
  idempotent a l'arrencada (T-19), no una migració.
- **Ordre d'execució dins de `handleLogin` és estricte** (ADR-03):
  JSON → validació → rate limiter → bcrypt. Cap pas es pot reordenar
  sense tornar a `software-architect`.
- **Patró de test ja establert:** `newTestServer`/`doJSON` de
  `tests/integration/helpers_test.go` i el patró d'obrir `srv.Store.DB`
  directament (`sql_static_test.go`) es reutilitzen sense modificació
  per a T-26 a T-33 — no inventar un altre harness.
- **Comandes locals:** `cd app && go test ./...` (unitat + integració),
  `cd app && go vet ./...`, `gofmt -l .`; Playwright: `cd
  app/tests/e2e && npx playwright test` (T-36 s'hi afegeix com a
  `login-cycle.spec.js`).
- **Credencials de test:** generar un hash bcrypt de test amb
  `go run` puntual o una utilitat de test — mai un hash/contrasenya
  real, ni tan sols en fixtures (S11/NFR-09).
- **Mecanisme CSRF (ADR-05, HMAC derivat sense taula nova)** és la
  decisió tal com està a `design.md` — la pregunta oberta §10 de
  `design.md` sobre si el propietari humà prefereix un token persistit
  en lloc de derivat queda **oberta i no bloquejant**; si la resposta
  humana canvia el mecanisme abans de `/code`, cal tornar a
  `software-architect` (canviaria l'esquema) i regenerar aquest fitxer
  amb `/define NIU-4 --extend`.
