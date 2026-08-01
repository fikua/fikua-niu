---
artefact: tasks
key: "NIU-1"
title: "Llista de la compra ↔ rebost (auth stubbed)"
status: "approved"
owner: "task-planner"
design_path: "./design.md"
requirements_path: "./requirements.md"
task_count: 35
ac_coverage:
  - ac: "AC-01"
    tasks: ["T-06", "T-07", "T-20", "T-21"]
  - ac: "AC-02"
    tasks: ["T-07", "T-13", "T-20", "T-25"]
  - ac: "AC-03"
    tasks: ["T-07", "T-13", "T-20", "T-25"]
  - ac: "AC-04"
    tasks: ["T-07", "T-20"]
  - ac: "AC-05"
    tasks: ["T-08", "T-20"]
  - ac: "AC-06"
    tasks: ["T-04", "T-05", "T-15", "T-21"]
  - ac: "AC-07"
    tasks: ["T-09", "T-16", "T-21"]
  - ac: "AC-08"
    tasks: ["T-17", "T-22"]
  - ac: "AC-09"
    tasks: ["T-07", "T-22"]
  - ac: "AC-10"
    tasks: ["T-18", "T-23"]
  - ac: "AC-11"
    tasks: ["T-18", "T-23"]
  - ac: "AC-12"
    tasks: ["T-19", "T-23"]
  - ac: "AC-13"
    tasks: ["T-19", "T-23"]
  - ac: "AC-14"
    tasks: ["T-19", "T-23"]
  - ac: "AC-15"
    tasks: ["T-15", "T-24"]
  - ac: "AC-16"
    tasks: ["T-15", "T-17", "T-24"]
ec_coverage:
  - ec: "EC-01"
    tasks: ["T-06", "T-20"]
  - ec: "EC-02"
    tasks: ["T-06", "T-20"]
  - ec: "EC-03"
    tasks: ["T-06", "T-20"]
  - ec: "EC-04"
    tasks: ["T-06", "T-20"]
  - ec: "EC-05"
    tasks: ["T-06", "T-20"]
  - ec: "EC-06"
    tasks: ["T-05", "T-06", "T-20"]
  - ec: "EC-07"
    tasks: ["T-06", "T-20"]
  - ec: "EC-08"
    tasks: ["T-14", "T-26"]
  - ec: "EC-09"
    tasks: ["T-15", "T-27"]
  - ec: "EC-10"
    tasks: ["T-04", "T-27"]
  - ec: "EC-11"
    tasks: ["T-08", "T-20"]
  - ec: "EC-12"
    tasks: ["T-07", "T-20"]
  - ec: "EC-13"
    tasks: ["T-19", "T-20"]
  - ec: "EC-14"
    tasks: ["T-02", "T-29"]
  - ec: "EC-15"
    tasks: ["T-12", "T-29"]
  - ec: "EC-16"
    tasks: ["T-17", "T-23"]
  - ec: "EC-17"
    tasks: ["T-16", "T-28"]
nfr_coverage:
  - nfr: "NFR-01"
    tasks: ["T-15", "T-27"]
  - nfr: "NFR-02"
    tasks: ["T-14", "T-27"]
  - nfr: "NFR-03"
    tasks: ["T-04", "T-27"]
  - nfr: "NFR-04"
    tasks: ["T-14", "T-26"]
  - nfr: "NFR-05"
    tasks: ["T-28"]
  - nfr: "NFR-06"
    tasks: ["T-16", "T-28"]
  - nfr: "NFR-07"
    tasks: ["T-12", "T-29"]
  - nfr: "NFR-08"
    tasks: ["T-14", "T-26"]
  - nfr: "NFR-09"
    tasks: ["T-16", "T-24", "T-28"]
  - nfr: "NFR-10"
    tasks: ["T-17", "T-24"]
  - nfr: "NFR-11"
    tasks: ["T-04", "T-27"]
sources:
  - "GitHub-style checklist (Markdown task lists)"
  - "Fikua AC↔tasks traceability matrix"
created: "2026-08-01"
updated: "2026-08-01"
---

# Tasks — Llista de la compra ↔ rebost (auth stubbed)

> **Què és això.** El llistat d'implementació. Cada tasca és petita
> (≤ ~1 hora), autocontinguda i lligada explícitament a almenys un
> criteri d'acceptació. **Cap tasca sense un AC que la cobreixi; cap AC
> sense almenys una tasca.** Aquest fitxer és l'únic artefacte mutable
> durant `/code` — la resta estan bloquejats.

## 1. Task list

### Fase 0 — Fonaments (mòdul Go, migracions, esquelet)

- [x] **T-01** — Inicialitzar `app/go.mod` (Go 1.25) amb les dependències
  exactes de `design.md` §1/§2.3: `github.com/go-chi/chi/v5`,
  `modernc.org/sqlite`, `github.com/pressly/goose/v3`,
  `golang.org/x/text/unicode/norm` (obligatòria per NFC, ADR-02). Crear
  l'esquelet de directoris `cmd/niu/`, `internal/{config,store,items,auth,httpapi}`,
  `migrations/`, `web/`, `tests/`. · *covers:* NFR-11 (base)
- [x] **T-02** — Escriure les migracions goose `app/migrations/001_initial_schema.sql`
  i `app/migrations/002_seed_users.sql` amb el DDL exacte de `design.md`
  §6.2 (taules `users`, `sessions`, `items` amb columnes `name_normalized`
  i `updated_at`, `events`; índex únic `idx_items_name_normalized`; seed
  `Usuari A`/`Usuari B` amb hash placeholder). **DELETE dur, sense
  `deleted_at`** — decisió humana confirmada. · *covers:* EC-14
- [x] **T-03** — Implementar `internal/config`: parsing d'entorn
  (`NIU_PORT`, `NIU_DB_PATH`, `NIU_ENV`), validació fail-fast segons
  `PLAN.md` §6 (en NIU-1 pràcticament res és requerit — `NIU_SESSION_SECRET`/
  `NIU_USER_*_HASH` són "yes (NIU-4)", no aquí).
- [x] **T-04** — Implementar `internal/store`: obertura SQLite amb
  `modernc.org/sqlite`, DSN amb `_pragma=journal_mode(WAL)`,
  `_pragma=busy_timeout(5000)`, `_pragma=foreign_keys(on)` (design.md §7);
  wiring de goose embedded (`//go:embed migrations/*.sql`); **cap
  `SetMaxOpenConns` a la connexió d'escriptura en aquest ítem** — decisió
  humana confirmada, revisitar només si NFR-05 mostra contenció. Totes
  les consultes amb paràmetres vinculats (`?`), mai `fmt.Sprintf` cap a
  SQL. · *covers:* NFR-03, EC-10 (base)

### Fase 1 — Domini `internal/items`

- [x] **T-05** — Definir el domini `internal/items`: tipus `Item`, `User`
  (referència lleugera), interfícies `Repository` (`Create`, `Get`,
  `List`, `Update`, `Delete`, `ExistsByNormalizedName`) i `EventSink`
  (`Record(kind, payload)`), sense importar `net/http` ni
  `database/sql` (design.md §4). Implementar la normalització ADR-02:
  `norm.NFC.String(...)` → `strings.TrimSpace` → `strings.ToLower`, en
  aquest ordre exacte. · *covers:* AC-06 (base d'autoria), EC-06 (base)
- [x] **T-06** — Implementar `items.Service.Add(ctx, userID, rawName)`:
  retallar espais, validar longitud 1–200 (post-trim), rebutjar
  caràcters de control, normalitzar (T-05), comprovar
  `ExistsByNormalizedName` dins la mateixa transacció que l'`INSERT`,
  retornar `ErrDuplicate{ExistingLocation}` si escau; `position =
  MAX(position) WHERE location='shopping' + 1.0` (o `1.0` si buida);
  escriure event `item_added`. · *covers:* AC-01, EC-01, EC-02, EC-03,
  EC-04, EC-05, EC-06, EC-07
- [x] **T-07** — Implementar `items.Service.Move(ctx, userID, id,
  newLocation)`: transacció única `UPDATE items SET location=?,
  moved_by=?, moved_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP,
  position=? WHERE id=?` (ADR-01 — última escriptura per timestamp de
  servidor, sense `If-Match`/versió); retornar `ErrNotFound` si l'`id`
  no existeix; escriure event `item_moved`. · *covers:* AC-02, AC-03,
  AC-04, AC-09, EC-12
- [x] **T-08** — Implementar `items.Service.Delete(ctx, userID, id)`:
  idempotent — segona crida sobre un `id` ja eliminat també retorna èxit
  (`204` equivalent), sense error; escriure event `item_deleted` només
  quan la fila existia. · *covers:* AC-05, EC-11
- [x] **T-09** — Implementar `items.Service.CurrentUser` (delega a
  `auth.Authenticator`, vegeu T-10) i `items.Service.List(ctx)`: una
  sola consulta `SELECT` amb `JOIN` a `users` (per `added_by`/`moved_by`),
  `ORDER BY location, position`, sense N+1. · *covers:* AC-07 (base),
  NFR-05 (base)

### Fase 2 — Auth stub i seam d'interfície

- [x] **T-10** — Definir `internal/auth.Authenticator` (interfície:
  `CurrentUser(r *http.Request) (User, error)`) i
  `auth.StubAuthenticator{UserID: <seed A>}` que ignora la petició i
  retorna sempre el mateix usuari (ADR-03). Cap lògica inline als
  handlers — el seam ha de quedar llest per NIU-4 sense tocar
  `items_handlers.go`.

### Fase 3 — HTTP API

- [x] **T-11** — Muntar el router `chi/v5` a `internal/httpapi`:
  middleware `WithCurrentUser(authenticator)` que injecta l'usuari al
  context via `auth.FromContext`; middleware `chi/middleware.Recoverer`
  - wrapper propi que mai serialitza `err.Error()` intern — respon
  `500 {"error":{"code":"internal_error","message":"S'ha produït un
  error inesperat."}}` i registra el detall només al log del servidor
  (`log/slog` a stdout). · *covers:* NFR-08 (base d'errors uniformes)
- [x] **T-12** — Implementar el middleware `httpapi.SecurityHeaders`
  aplicat a **totes** les respostes (API i estàtics), abans de
  qualsevol altre middleware: `Strict-Transport-Security`,
  `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`, `Content-Security-Policy` exacta de
  `design.md` §9 (sense `'unsafe-inline'` enlloc). · *covers:* EC-15
  (base de resiliència, no directament — vegeu T-29 per al test real)
- [x] **T-13** — Implementar handlers `GET/POST /api/v1/items` i `PATCH/DELETE
  /api/v1/items/{id}`: deserialitzar/serialitzar JSON, delegar a
  `items.Service`, mapar `ErrDuplicate`→`409 duplicate_item`,
  `ErrNotFound`→`404 not_found`, error de validació→`400
  validation_failed`, envelope d'error uniforme
  `{"error":{"code","message"}}` (design.md §6.1). Confirmar que **cap
  ruta `GET` crida mai** `Add/Move/Delete`.
- [x] **T-14** — Implementar `GET /api/v1/me` (delega a
  `auth.Authenticator.CurrentUser`) i `GET /healthz` (`SELECT 1` contra
  SQLite; `200` si èxit, `503` en cas contrari, REL-03). Escriure el
  test d'integració que introspecciona la taula de rutes `chi` i
  assegura que cap handler `GET` coincideix amb un mètode mutador
  (EC-08). · *covers:* AC-07 (base), EC-08, NFR-04, NFR-08
- [x] **T-15** — Muntar `embed.FS` sobre `web/` a `cmd/niu/main.go`
  (`//go:embed web`), servir-lo amb `http.FileServer(http.FS(webFS))`
  arrelat a `/` per qualsevol ruta que no comenci per `/api/v1/` ni
  sigui `/healthz` (design.md §8). Cablejar el `main.go` complet:
  `config` → `store` → `items.Service` → `auth.StubAuthenticator` →
  `httpapi.NewRouter`.

### Fase 4 — Frontend (`app/web/`) — port del design-system

- [x] **T-16** — Copiar literalment els valors de `design-system/tokens.css`
  a `app/web/app.css` (mateixos noms de custom property, cap hex/px/ms
  reinventat). Autoallotjar els fitxers `.woff2` de Nunito
  (`Nunito-Regular.woff2`, `Nunito-Bold.woff2`) a `app/web/fonts/` —
  **mai** una crida a Google Fonts CDN (la CSP ho prohibeix). Aplicar
  l'escala tipogràfica i el `@font-face` amb el fallback stack de
  `proposal.md` §8.2.
- [x] **T-17** — Crear `app/web/index.html`: marcatge base, capçalera,
  dues caixes ("A comprar"/"Rebost") amb `role="region"`
  `aria-label`, regió `aria-live="polite" aria-atomic="true"`
  visualment oculta (tècnica `sr-only`) immediatament després de la
  capçalera, `<script type="module" src="js/main.js">`. Estructura DOM
  i classes idèntiques a `design-system/screen-desktop.html` /
  `screen-mobile.html` (`.item-row`, `.box`, `#list-shopping`,
  `#list-pantry`, `#live-region`).
- [x] **T-18** — Crear `app/web/js/flip.js`: port directe de
  `captureRects()`/`playFlip()` de `design-system/screen-desktop.html`
  (FLIP amb Web Animations API, `transform`, 250ms, `ease-out`;
  detecció de `prefers-reduced-motion` → cross-fade 150ms+150ms
  seqüencial, sense `transform`). Implementar el comportament exacte de
  FLIP mòbil a través del límit de pestanyes descrit a `design.md` §8
  (fila desapareix de la pestanya activa amb FLIP, apareix a la
  inactiva amb `.just-added` fade-in, sense `transform` cap a
  coordenades ocultes).
- [x] **T-19** — Crear `app/web/js/store.js`: estat en memòria (`items`
  array), `moveItemOptimistic(id, newLocation)` (actualitza abans de
  rebre resposta, `render({flipFromRects})`, `await api.moveItem(...)`,
  rollback + `render` invers + `toast.show` en cas d'error),
  `syncFromServer()` (diffing per `id`+`location`+`moved_at`). Guarda de
  confeti d'un sol tret: variable de mòdul `shoppingWasEmpty`, es
  dispara confeti només en la transició no-buit→buit causada per acció,
  mai en càrrega inicial (EC-13), es reinicia quan torna a haver-hi
  ítems.
- [x] **T-20** — Crear `app/web/js/api.js` (fetch wrappers: `getItems()`,
  `addItem()`, `moveItem()`, `deleteItem()`, `getMe()`) i
  `app/web/js/render.js` (`render()`, `renderRow()`,
  `renderEmptyState()`, `renderAvatars()` — port directe de la lògica
  ja provada a `design-system/screen-desktop.html`). **Zero `innerHTML`
  amb dades d'usuari** — tots els nodes amb `document.createElement` +
  `.textContent`. Implementar els missatges d'error exactes de
  `proposal.md` §8.4.3 (validació, duplicat amb nom de caixa).
- [x] **T-21** — Crear `app/web/js/a11y.js` (`announce()` amb el format
  exacte `"{Nom} mogut a {caixa}."` / `"{Nom} mogut a {caixa} per
  {usuari}."` per canvis remots) i implementar `renderAvatars()` amb la
  lògica d'un/dos avatars i separador `↩` de `proposal.md` §8.8.
  Gestió de `tabindex` mòbil per al botó "eliminar" (només tabulable
  quan la fila té el focus).
- [x] **T-22** — Crear `app/web/js/confetti.js` (`confetti()` amb guarda
  d'un sol tret, ~24–30 partícules, colors `moss`/`terracotta`/groc
  suau `#E8C468`, ~1200ms, `ease-in`, origen centre superior de "A
  comprar"; alternativa `prefers-reduced-motion` = destello estàtic
  `color.success-bg` 600ms) i `app/web/js/tabs.js` (`setActivePanel()`,
  breakpoint 768px, pestanya activa amb subratllat).
- [x] **T-23** — Crear `app/web/js/main.js`: punt d'entrada, wiring
  d'events (clic/tap i `Enter`/`Space` sobre `ItemRow`, formulari
  d'afegir, botó eliminar, pestanyes mòbil, `×`/`Escape` del toast),
  `syncFromServer()` immediat + `setInterval(syncFromServer, 10000)` +
  listener `window.addEventListener('focus', syncFromServer)`. Crida
  `GET /api/v1/me` en carregar.

### Fase 5 — Tests (small/medium/large, per la piràmide de `qa-engineer`)

- [x] **T-24** — Test unitari de validació de nom i normalització
  (`internal/items`): frontera 200/201 caràcters (EC-02/EC-03), nom
  buit/espais (EC-01), caràcters de control (EC-05), corpus Unicode
  complet accents/emoji/apòstrof (EC-04/NFR-11), i el cas NFC vs NFD
  documentat a ADR-02 (`"Àrab"` compost vs descompost han de coincidir
  després de normalitzar). · *covers:* AC-01 (unit), EC-01, EC-02,
  EC-03, EC-04, EC-05, EC-06 (normalització), NFR-11
- [x] **T-25** — Test d'integració CF-01/CF-07/CF-08/CF-09/AC-04:
  `POST /api/v1/items` → `GET` (persistència, CF-01), `PATCH`
  `location=pantry`/`location=shopping` → assert nova ubicació i camps
  d'autoria a la resposta i al `GET` posterior (CF-07/CF-08/CF-09).
  · *covers:* AC-01, AC-02, AC-03, AC-04
- [x] **T-26** — Test d'integració de duplicats i eliminació:
  CF-05 (6 combinacions retallat+minúscules a través de totes dues
  caixes), CF-06 (crear→eliminar→recrear mateix nom), CF-10/EC-11
  (`DELETE` + doble `DELETE` idempotent), EC-12 (moure ítem inexistent
  → error clar). · *covers:* AC-05, EC-06, EC-07, EC-11, EC-12
- [x] **T-27** — Test d'integració de dos usuaris i concurrència:
  CF-11 (dues sessions simulades, convergència via `GET`), CF-12/AC-09
  (dues `PATCH` concurrents via goroutines sobre el mateix ítem, assert
  cap 5xx i `GET` posterior amb estat únic coincidint amb l'`updated_at`
  més recent — ADR-01), CF-13 (atribució `added_by`/`moved_by` correcta
  amb identitats diferents). · *covers:* AC-06, AC-07, AC-08, AC-09
- [x] **T-28** — Test d'integració de seguretat S3a/S3b/S7/S8/EC-08/EC-09/EC-10:
  assert capçaleres de seguretat presents al 100% de respostes (S7,
  NFR-02); assert CSP sense `unsafe-inline` (S3b); `POST` amb nom
  `<img src=x onerror=alert(1)>` desat literalment i, en navegador real
  (Playwright), assert no-execució de script i text literal renderitzat
  (S3a/EC-09); `POST` amb `'; DROP TABLE items;--` desat literalment i
  taula intacta post-atac (S8/EC-10); assaig estàtic de rutes `GET`
  sense efecte mutador (EC-08, ja creat a T-14 — assert reforçat aquí
  amb grep de `fmt.Sprintf.*SELECT|INSERT|UPDATE|DELETE` a
  `internal/store/` com a comprovació estàtica recomanada). · *covers:*
  NFR-01, NFR-02, NFR-03, NFR-04, EC-08, EC-09, EC-10
- [x] **T-29** — Test E2E (Playwright) d'animació i interacció: CF-16
  (FLIP ~250ms, posició final correcta), CF-17
  (`prefers-reduced-motion` → cross-fade sense vol), CF-18 (confeti
  exactament un cop en buidar, no reapareix en renders posteriors),
  CF-19 (viewport mòbil 375×667, pestanyes, totes les accions),
  CF-20 (optimista + error de servidor mockejat → reversió + toast),
  CF-21 (navegació completa Tab/Enter/Space per afegir/moure/eliminar),
  CF-22/AC-16 (contingut exacte de `aria-live` en moviment propi i
  remot). · *covers:* AC-06, AC-10, AC-11, AC-12, AC-13, AC-14, AC-15,
  AC-16, EC-13 (estat buit sense confeti fals), EC-16
- [x] **T-30** — Test de persistència i rendiment: EC-14 (seed dades →
  reinici del procés/contenidor → `GET /api/v1/items` assert igualtat
  de conjunt); PERF-01/NFR-05 (seed de 500 ítems, mesura p95 de `GET
  /api/v1/items` < 200ms); PERF-02/NFR-06 (Lighthouse amb perfil 3G
  simulat, temps a interactiu < 1s). · *covers:* EC-14, NFR-05, NFR-06

### Fase 6 — Verificació manual obligatòria i procediment de tancament

- [x] **T-31** — Crear `app/tests/killtest/main.go` (ADR-04): programa
  Go independent que (1) arrenca el binari `niu` real com a procés
  fill contra una BD temporal, (2) llança una goroutine que envia
  `PATCH /api/v1/items/{id}` contínuament, (3) espera un interval
  aleatori (50–500ms) i envia `SIGKILL`, (4) reobre la BD i executa
  `PRAGMA integrity_check`, (5) arrenca el binari de nou i verifica
  `GET /healthz` = 200. Exposar com a target `killtest` al `Makefile`
  (`make killtest N=10`). Documentar a
  `app/tests/killtest/README.md` la comanda exacta i el resultat
  esperat. **Executar manualment 10 vegades i deixar constància del
  resultat (log/sortida) abans de tancar NIU-1** — procediment
  obligatori, no opcional (REL-01/NFR-07). · *covers:* EC-15, NFR-07
- [x] **T-32** — Crear `app/Makefile` amb els targets alineats amb
  `commands.*` del manifest: `test` (`go test ./...`), `lint`
  (`gofmt -l`, ha d'acceptar un path final), `typecheck` (`go vet
  ./...`), `build` (`go build ./...`), `bootstrap` (`go mod download`),
  `up`/`down` (docker compose — placeholder, NIU-2 el completa), i
  `killtest` (crida a T-31).
- [x] **T-33** — Auditoria manual d'accessibilitat i contrast: executar
  axe-core/Lighthouse sobre totes les combinacions text/fons de
  `proposal.md` §8.1.1 (EC-17/A11Y-02), i verificació puntual amb un
  lector de pantalla real de l'anunci `aria-live` (A11Y-03,
  complementari a T-29). Deixar constància del resultat. · *covers:*
  EC-17, NFR-09, NFR-10
- [x] **T-34** — Escaneig S11 de dades personals: revisar tots els
  fitxers versionats (codi, migracions, fixtures de test, documentació)
  i confirmar que només apareixen `Usuari A`/`Usuari B` i cap nom real,
  correu o detall domèstic identificable. · *covers:* (S11 — vegeu
  `test-plan.md`, no mapejat a cap AC/EC/NFR de `requirements.md`
  perquè és un requisit de seguretat de projecte transversal; es
  documenta aquí per traçabilitat amb el pla de proves)
- [x] **T-35** — Executar `commands.test`, `commands.lint`,
  `commands.typecheck` i `commands.build` del manifest de punta a punta
  i confirmar que **tots** els casos 🟢 NIU-1 de `docs/test-plan.md`
  (CF-01…CF-22, S1a, S3a, S3b, S7, S8, S11, PERF-01, PERF-02, REL-01,
  REL-03, A11Y-01, A11Y-02, A11Y-03) estan verds. Deixar constància
  explícita de l'execució manual de T-31 (10 repeticions) i T-33 abans
  de marcar la història com a llesta. · *covers:* verificació final de
  totes les AC/EC/NFR (tasca de tancament tècnic, no substitueix cap
  cobertura individual anterior)

### Closing (universal — all changes)

- [x] **C-01** — Append changelog entry (`docs.changelog` from manifest)
- [x] **C-02** — Transition backlog item to `In Progress` via the adapter (per `/code` phase contract; `/audit` will move it to `Reviewed`, `/commit` to `Human Review`)
- [ ] **C-03** — Propose semver bump (ASK USER — never apply unattended; deferred to `/commit`, not part of `/code`)

## 2. AC ↔ tasks traceability matrix

| AC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| AC-01 | Afegir un ítem nou a "A comprar" | T-06, T-07, T-20, T-21, T-24, T-25 |
| AC-02 | Moure "A comprar" → "Rebost" | T-07, T-13, T-20, T-25 |
| AC-03 | Moure "Rebost" → "A comprar" | T-07, T-13, T-20, T-25 |
| AC-04 | El moviment persisteix | T-07, T-20, T-25 |
| AC-05 | Eliminar un ítem | T-08, T-20, T-26 |
| AC-06 | Cada ítem mostra qui l'ha tocat | T-04, T-05, T-15, T-21, T-27, T-29 |
| AC-07 | Usuari actual identificat (auth stubbed) | T-09, T-14, T-16, T-21, T-27 |
| AC-08 | Dos usuaris veuen la mateixa llista | T-17, T-22, T-27 |
| AC-09 | Moviment concurrent convergeix sense error | T-07, T-22, T-27 |
| AC-10 | Animació de moviment (FLIP) | T-18, T-23, T-29 |
| AC-11 | Alternativa accessible (reduced motion) | T-18, T-23, T-29 |
| AC-12 | Actualització optimista amb èxit | T-19, T-23, T-29 |
| AC-13 | Actualització optimista amb fallada i reversió | T-19, T-23, T-29 |
| AC-14 | Confeti en buidar "A comprar" | T-19, T-23, T-29 |
| AC-15 | Navegació completa per teclat | T-15, T-24 (parcial), T-29 |
| AC-16 | Anunci per lectors de pantalla | T-15, T-17, T-21, T-24 (parcial), T-29 |

## 3. Edge cases ↔ tasks

| EC | Statement (short) | Covering tasks |
|----|--------------------|----------------|
| EC-01 | Nom buit o només espais | T-06, T-20, T-24 |
| EC-02 | Nom al límit (200 caràcters) | T-06, T-20, T-24 |
| EC-03 | Nom excedeix el límit (201) | T-06, T-20, T-24 |
| EC-04 | Nom amb Unicode complet | T-06, T-20, T-24 |
| EC-05 | Nom amb caràcters de control | T-06, T-20, T-24 |
| EC-06 | Ítem duplicat (qualsevol caixa) | T-05, T-06, T-20, T-24, T-26 |
| EC-07 | Duplicat exacte permès post-eliminació | T-06, T-20, T-26 |
| EC-08 | Intent de mutació via GET | T-14, T-28 |
| EC-09 | Injecció HTML/JS (XSS) | T-15, T-20, T-28 |
| EC-10 | Injecció SQL | T-04, T-28 |
| EC-11 | Eliminar ítem ja eliminat | T-08, T-20, T-26 |
| EC-12 | Moure ítem inexistent | T-07, T-20, T-26 |
| EC-13 | Llista buida en primer ús | T-19, T-20, T-29 |
| EC-14 | Reinici del contenidor | T-02, T-30 |
| EC-15 | Reinici a mig d'una escriptura | T-12, T-31 |
| EC-16 | Viewport mòbil | T-17, T-18, T-22, T-23, T-29 |
| EC-17 | Contrast de color insuficient | T-16, T-33 |

## 4. NFRs ↔ tasks

| NFR | Statement (short) | Covering tasks |
| --- | --- | --- |
| NFR-01 | Sense render HTML de dades d'usuari (XSS) | T-15, T-20, T-28 |
| NFR-02 | Capçaleres de seguretat obligatòries | T-12, T-28 |
| NFR-03 | Cap concatenació SQL | T-04, T-28 |
| NFR-04 | Cap mutació via GET | T-14, T-28 |
| NFR-05 | p95 < 200ms amb 500 ítems | T-09, T-30 |
| NFR-06 | Càrrega inicial < 1s en 3G | T-16, T-30 |
| NFR-07 | Sobreviu interrupció abrupta (10 repeticions) | T-12, T-31 |
| NFR-08 | `/healthz` reflecteix estat real de la BD | T-11, T-14 |
| NFR-09 | Contrast AA i operabilitat per teclat | T-16, T-24 (parcial), T-33 |
| NFR-10 | Canvis d'estat anunciats a lectors de pantalla | T-17, T-21 |
| NFR-11 | Alfabet català complet sense pèrdua | T-01 (dependència), T-05, T-24 |

## 5. Out of scope (mirrored from design)

> Fora d'abast d'aquest ítem — cap tasca de la llista anterior ha
> d'incloure res d'això.

- Autenticació real, pantalla de login, gestió de contrasenyes (NIU-4).
- Tokens de sessió, cookies `HttpOnly/Secure/SameSite`, doble-submit
  CSRF (NIU-4 — S1 complet, S2, S5, S6).
- Rate limiting per força bruta (S4 — NIU-4/NIU-2).
- Dockerfile, `compose.yaml` definitiu, workflows de CI/CD, DNS,
  Cloudflare Access, backup SQLite (NIU-2). El `Makefile` de T-32 crea
  només placeholders locals (`up`/`down` via `docker compose`
  existent al manifest), no la infraestructura de desplegament.
- Instrumentació OTEL / traces, `internal/observability` cablejat
  (NIU-3) — el paquet pot existir buit com a seam, però no es
  connecta.
- Quantitats numèriques per ítem — viu dins del text del nom.
- Notificacions en temps real (SSE o equivalent) — només sondeig i
  refetch al focus.
- Streaks, punts, classificacions (gamificació avançada).
- Multi-llar, rols, permisos, convidar usuaris.
- Optimistic locking amb `If-Match`/versió (ADR-01, rebutjat
  explícitament per aquest ítem).
- Reordenació manual dins d'una caixa (drag-to-reorder) — el patró
  `position REAL` es reserva per al futur però no s'exercita aquí.
- `deleted_at` / esborrat tou — DELETE dur confirmat pel titular humà.

## 6. Notes for the developer

- **Ordre estricte de dependència:** T-01→T-04 (fonaments) han
  d'estar fets abans de tocar `internal/items` (T-05...); el domini
  abans dels handlers HTTP (T-11...); els handlers abans del wiring
  frontend (T-16...). No saltar fases.
- **`golang.org/x/text/unicode/norm` no és opcional** (T-05, ADR-02):
  sense el pas NFC, EC-06 falla silenciosament per a noms catalans amb
  accents introduïts des de dispositius diferents (macOS NFD vs.
  teclats/navegadors NFC). Verificar-ho amb el test `"Àrab"` compost vs.
  descompost a T-24.
- **Fonts Nunito autoallotjades (T-16):** els fitxers `.woff2` reals
  s'han d'obtenir/generar (Google Fonts en permet la descàrrega per a
  autoallotjament, llicència OFL) — mai referenciar un CDN extern. Si
  els fitxers no existeixen encara al repositori, aquesta tasca inclou
  descarregar-los i col·locar-los a `app/web/fonts/`.
- **Port del design-system, no redisseny:** T-16 a T-23 són ports
  estructurals de `design-system/screen-desktop.html`,
  `screen-mobile.html` i els fitxers `component-*.html` — mateixos
  IDs/classes DOM, mateixos valors de `tokens.css`. Qualsevol desviació
  visual és un defecte, no una millora.
- **`make killtest` (T-31, ADR-04):** no corre a CI per defecte. És un
  procediment manual obligatori — cal deixar constància (log/sortida)
  de les 10 execucions abans de considerar NIU-1 tancat; `/audit`
  bloquejarà si no hi ha evidència.
- **Comandes del manifest:** `test: "cd app && go test ./..."`,
  `lint: "cd app && gofmt -l"` (accepta un path final per fitxer),
  `typecheck: "cd app && go vet ./..."`, `build: "cd app && go build
  ./..."`.

## 7. Preguntes obertes heretades de `design.md` §12

> `task-planner` no les resol — es reporten aquí perquè eren
> preguntes pendents de la porta humana abans de generar aquest
> fitxer. Ambdues han estat **confirmades pel propietari humà** just
> abans d'aquesta etapa (veure instruccions de la sessió):
>
> 1. **Abast de `busy_timeout`/`SetMaxOpenConns`** — confirmat: no
>    limitar la connexió d'escriptura en aquest ítem (T-04); revisitar
>    només si les proves de càrrega (T-30/NFR-05) mostren contenció.
> 2. **`deleted_at` no s'introdueix** — confirmat: DELETE dur (T-02);
>    `events` ja cobreix l'historial (T-06/T-07/T-08 escriuen
>    `item_added`/`item_moved`/`item_deleted`).
>
> Cap pregunta oberta bloquejant per a `fullstack-developer`.
