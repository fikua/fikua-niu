---
artefact: tasks
key: "NIU-5"
title: "Compres grans i projectes de casa"
status: "approved"
owner: "task-planner"
design_path: "./design.md"
requirements_path: "./requirements.md"
task_count: 33
ac_coverage:
  - ac: "AC-01"
    tasks: ["T-03", "T-04", "T-08", "T-14", "T-18", "T-23", "T-25", "T-33"]
  - ac: "AC-02"
    tasks: ["T-05", "T-09", "T-15", "T-19", "T-24", "T-33"]
  - ac: "AC-03"
    tasks: ["T-05", "T-09", "T-15", "T-19", "T-24", "T-33"]
  - ac: "AC-04"
    tasks: ["T-11", "T-16", "T-19", "T-24", "T-33"]
  - ac: "AC-05"
    tasks: ["T-06", "T-10", "T-17", "T-19", "T-25", "T-33"]
  - ac: "AC-06"
    tasks: ["T-07", "T-20", "T-27", "T-33"]
  - ac: "AC-07"
    tasks: ["T-05", "T-19", "T-28", "T-33"]
  - ac: "AC-08"
    tasks: ["T-13", "T-18", "T-19", "T-21", "T-33"]
  - ac: "AC-09"
    tasks: ["T-05", "T-09", "T-15", "T-19", "T-24", "T-33"]
  - ac: "AC-10"
    tasks: ["T-02", "T-04", "T-22", "T-33"]
  - ac: "AC-11"
    tasks: ["T-19", "T-30", "T-33"]
  - ac: "AC-12"
    tasks: ["T-16", "T-19", "T-30"]
  - ac: "AC-13"
    tasks: ["T-32"]
  - ac: "AC-14"
    tasks: ["T-02", "T-04", "T-16", "T-19", "T-31"]
  - ac: "AC-15"
    tasks: ["T-02", "T-04", "T-16", "T-19", "T-31"]
ec_coverage:
  - ec: "EC-01"
    tasks: ["T-04", "T-22"]
  - ec: "EC-02"
    tasks: ["T-04", "T-22"]
  - ec: "EC-03"
    tasks: ["T-03", "T-04", "T-08", "T-23"]
  - ec: "EC-04"
    tasks: ["T-08", "T-25"]
  - ec: "EC-05"
    tasks: ["T-06", "T-29"]
  - ec: "EC-06"
    tasks: ["T-19", "T-29"]
  - ec: "EC-07"
    tasks: ["T-04"]
  - ec: "EC-08"
    tasks: ["T-16", "T-26"]
  - ec: "EC-09"
    tasks: ["T-01", "T-26"]
  - ec: "EC-10"
    tasks: ["T-12", "T-26"]
  - ec: "EC-11"
    tasks: ["T-12", "T-26"]
  - ec: "EC-12"
    tasks: ["T-06", "T-09", "T-24"]
  - ec: "EC-13"
    tasks: ["T-06", "T-10", "T-24"]
  - ec: "EC-14"
    tasks: ["T-17", "T-30"]
  - ec: "EC-15"
    tasks: ["T-17", "T-30"]
  - ec: "EC-16"
    tasks: ["T-04", "T-22"]
  - ec: "EC-17"
    tasks: ["T-04", "T-22"]
nfr_coverage:
  - nfr: "NFR-01"
    tasks: ["T-06", "T-27"]
  - nfr: "NFR-02"
    tasks: ["T-16", "T-26"]
  - nfr: "NFR-03"
    tasks: ["T-01", "T-26"]
  - nfr: "NFR-04"
    tasks: ["T-12", "T-26"]
  - nfr: "NFR-05"
    tasks: ["T-12", "T-26"]
  - nfr: "NFR-06"
    tasks: ["T-19", "T-30"]
  - nfr: "NFR-07"
    tasks: ["T-19", "T-30"]
  - nfr: "NFR-08"
    tasks: ["T-13", "T-19"]
sources:
  - "GitHub-style checklist (Markdown task lists)"
  - "Fikua AC↔tasks traceability matrix"
created: "2026-08-02"
updated: "2026-08-02"
---

# Tasks — Compres grans i projectes de casa

> **Què és això.** El full de ruta d'implementació per a NIU-5. Cada
> tasca és petita (≤ ~1h), autocontinguda, i traçada a almenys un
> criteri d'acceptació de `requirements.md`. **Cap tasca sense un AC/EC/
> NFR que la cobreixi; cap AC/EC/NFR sense almenys una tasca.** Aquest
> fitxer és l'únic artefacte mutable durant `/code` — la resta són
> bloquejats (`design.md`/`requirements.md` aprovats).
>
> Traducció mecànica de `design.md` (4 ADRs aprovats) — cap decisió de
> disseny nova. `internal/items`, `internal/auth`, `internal/config` no
> es toquen excepte on s'indica explícitament (T-02, reutilització d'una
> sola funció exportada).

## 1. Task list

### Foundations

- [x] **T-01** — Escriure la migració goose
  `app/migrations/003_projects.sql` amb el DDL exacte de `design.md`
  §6.2: taula `projects` (`id`, `name`, `name_normalized`, `state` amb
  `CHECK (state IN ('idea','decidit','fet')) DEFAULT 'idea'`, `budget
  TEXT NULL`, `target_date TEXT NULL`, `added_by`, `last_updated_by`,
  `created_at`, `updated_at`), índex únic
  `idx_projects_name_normalized ON projects(name_normalized)` **sense**
  cap clàusula `WHERE` (ADR-02 — abast a través de tots els estats).
  Totes les consultes que la toquin usaran paràmetres vinculats, mai
  `fmt.Sprintf` cap a SQL. · *covers:* NFR-03 (base), EC-09 (base)
- [x] **T-02** — Definir el domini `internal/projects`: tipus `Project`
  (camps `ID`, `Name`, `State`, `Budget *string`, `TargetDate *string`,
  `AddedBy`/`LastUpdatedBy items.User`, `CreatedAt`/`UpdatedAt`),
  interfícies `Repository` (`Create`, `Get`, `List`, `UpdateState`,
  `Delete`, `ExistsByNormalizedName`) i `EventSink`
  (`Record(ctx, userID, kind, payload)`), sense importar `net/http` ni
  `database/sql` (design.md §4). Importar `internal/items` **només**
  per `items.NormalizeName` (ADR-02) — cap altre símbol. · *covers:*
  AC-10 (base), AC-14 (base), AC-15 (base)

### Implementation

- [x] **T-03** — Implementar `projects.Service.Add(ctx, userID, rawName,
  rawBudget, rawTargetDate)`: retallar i validar `name` (1–200 post-trim,
  mateixes regles de caràcters de control que `items`), normalitzar amb
  `items.NormalizeName` (ADR-02), comprovar `ExistsByNormalizedName`
  dins la mateixa transacció que l'`INSERT` (check-then-insert protegit
  per l'índex únic, mateix patró que `ItemsRepository.Create`); si
  existeix **en qualsevol estat** retornar `ErrDuplicate{}` (EC-03). ·
  *covers:* AC-01 (base)
- [x] **T-04** — Estendre `projects.Service.Add` amb la validació de
  `budget` (opcional, 1–200 caràcters post-trim si present, mateix
  llindar que el nom — EC-16) i `target_date` (opcional, format ISO-8601
  `YYYY-MM-DD` vàlid si present, **sense** cap comprovació de "no
  passat" — EC-17, resolent EC-07 amb la decisió de text lliure ja
  tancada). Si tot és vàlid: `INSERT` amb `state='idea'`,
  `added_by=userID`, `last_updated_by=userID`, escriure event
  `project_added`. · *covers:* AC-01, AC-10, AC-14, AC-15, EC-01, EC-02,
  EC-07, EC-16, EC-17
- [x] **T-05** — Implementar `projects.Service.ChangeState(ctx, userID,
  id, newState)`: validar que `newState` sigui un dels tres valors
  coneguts (`idea`/`decidit`/`fet`) — sense màquina d'estats amb
  transicions prohibides, qualsevol dels tres és sempre un moviment
  vàlid des de qualsevol altre (AC-09). `Repository.UpdateState` obre
  `BEGIN IMMEDIATE` (mateix patró que `ItemsRepository.Update`, ADR-01
  d'NIU-1) i executa un sol `UPDATE projects SET state=?,
  last_updated_by=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`. ·
  *covers:* AC-02, AC-03, AC-09, AC-07 (base)
- [x] **T-06** — Completar `ChangeState`: si `RowsAffected == 0` (`id`
  no existeix, EC-12) retornar `ErrNotFound`; en èxit, `COMMIT` i
  escriure event `project_state_changed` amb `payload {"from": "...",
  "to": "..."}` (NFR-01). Confirmar que un element estancat setmanes a
  `decidit` no pateix cap canvi automàtic (EC-05 — comportament,
  no codi addicional). · *covers:* AC-05 (base concurrència), EC-05,
  EC-12, EC-13, NFR-01
- [x] **T-07** — Implementar `projects.Service.List(ctx)`: una sola
  consulta `SELECT` amb `JOIN` a `users` (per `added_by` i
  `last_updated_by`), sense N+1, mateix patró que
  `ItemsRepository.List`. · *covers:* AC-06 (base — llista consultada
  per `GET`/sondeig)
- [x] **T-08** — Implementar `projects.Service.Delete(ctx, userID, id)`:
  `DELETE FROM projects WHERE id = ?` idempotent (`existed bool`, mateix
  patró que `items.Delete`); escriure event `project_deleted` amb
  `payload {"project_id": id}` només quan la fila existia. Un projecte
  eliminat deixa de comptar per a `ExistsByNormalizedName` (EC-04). ·
  *covers:* AC-01 (base — EC-04 duplicat post-eliminació), EC-03 (base
  d'exclusió d'eliminats), EC-04
- [x] **T-09** — Estendre `internal/store` amb `ProjectsRepository`,
  implementant `projects.Repository` i `projects.EventSink` (delegant
  `Record` a la mateixa taula `events` ja existent, cap columna ni tipus
  nou). Implementar `Create`, `Get`, `List`, `UpdateState`, `Delete`,
  `ExistsByNormalizedName` seguint exactament els patrons de
  `store.ItemsRepository`. · *covers:* AC-02, AC-03, AC-09, EC-12
- [x] **T-10** — Implementar handlers `GET/POST /api/v1/projects` i
  `PATCH/DELETE /api/v1/projects/{id}` a nou fitxer
  `internal/httpapi/projects_handlers.go`: deserialitzar/serialitzar
  JSON, delegar a `projects.Service`, mapar `ErrDuplicate`→`409
  duplicate_project`, `ErrNotFound`→`404 not_found`, error de
  validació→`400 validation_failed`, envelope d'error uniforme
  (`design.md` §6.1, reutilitza `apiError` existent). · *covers:* AC-05,
  EC-13 (base idempotència `DELETE`)
- [x] **T-11** — Estendre `internal/httpapi/dto.go` amb `projectDTO`
  (mapeig `Project`→JSON de resposta amb la forma exacta de `design.md`
  §6.1: `id`, `name`, `state`, `budget`, `target_date`, `added_by`,
  `last_updated_by` sempre no-nul, `created_at`, `updated_at`). ·
  *covers:* AC-04
- [x] **T-12** — **Modificar `router.go`** (canvi quirúrgic, no
  reescriptura): registrar el nou grup de rutes `/api/v1/projects`
  dins del `r.Route("/api/v1", ...)` ja existent, reutilitzant
  `WithCurrentUser` (ja muntat al grup) i
  `RequireCSRF(s.authenticator.SessionSecret())` per a
  `POST`/`PATCH`/`DELETE`, mai per a `GET`. Confirmar que cap ruta `GET`
  crida `Add`/`ChangeState`/`Delete`. **`items_handlers.go`,
  `auth_handlers.go` i `csrf.go` no es toquen.** · *covers:* EC-10,
  EC-11, NFR-04, NFR-05
- [x] **T-13** — Afegir a `cmd/niu/main.go` el wiring de
  `projects.Service` + `store.ProjectsRepository`, cablejat amb el
  mateix `*Store`/`config`/`authenticator` ja construïts — cap canvi a
  l'autenticador ni al cablejat existent d'`items`. Confirmar (sense
  codi addicional) que no s'ha de cablejar cap gestió de
  `prefers-reduced-motion` per a aquest espai, ja que ADR-03 el
  documenta com a no aplicable. · *covers:* AC-08 (base disponibilitat
  de dades), NFR-08 (documentat com a no aplicable)
- [x] **T-14** — Crear `app/web/js/projects-api.js` (o estendre
  `api.js`): fetch wrappers `getProjects()`, `addProject()`,
  `patchProjectState()`, `deleteProject()`, reutilitzant exactament el
  mateix patró (capçalera CSRF, `handleUnauthenticated()`) que els
  wrappers d'`items` ja existents a `api.js`. · *covers:* AC-01 (base
  frontend)
- [x] **T-15** — Crear `app/web/js/projects-store.js` (o estendre
  `store.js`): estat en memòria (`projects` array), `changeProjectState
  (id, newState)` (crida `patchProjectState`, actualitza l'estat local
  amb la resposta), `syncProjectsFromServer()` seguint exactament el
  mateix cicle de `syncFromServer()` ja implementat per `items` (AC-06,
  sondeig + refetch-on-focus, sense inventar un segon mecanisme). ·
  *covers:* AC-02, AC-03, AC-09
- [x] **T-16** — Crear `app/web/js/projects-render.js` (o estendre
  `render.js`): `renderProjectRow()` amb nom, badge d'estat (text
  llegible, no només color), avatar de qui l'ha afegit, avatar + data
  de qui l'ha actualitzat per últim cop, `budget`/`target_date` només
  si informats (§7 de `design.md`). **Zero `innerHTML` amb dades
  d'usuari** — tots els nodes amb `document.createElement` +
  `.textContent` (EC-08/NFR-02). Selector/menú de tres opcions per
  canviar d'estat en qualsevol direcció (mai només "següent estat"). ·
  *covers:* AC-04, AC-12 (base marcatge `aria-live`), AC-14, AC-15,
  EC-08, NFR-02
- [x] **T-17** — Estendre `projects-render.js` amb
  `renderEmptyProjectsState()`: missatge clar de "cap projecte encara"
  (EC-14, mai un error ni una taula buida sense context). Confirmar que
  el disseny responsive reutilitza el mateix patró CSS que `items`
  sense necessitar el mecanisme de pestanyes (EC-15, llista vertical
  simple). · *covers:* AC-05 (base UI eliminar), EC-14, EC-15
- [x] **T-18** — Crear la nova entrada de navegació ("🏠 Projectes" o
  equivalent), visible des de qualsevol punt de l'app, costat a costat
  amb l'accés a la llista de la compra, servida des del mateix
  `index.html`/`embed.FS` o com a document separat
  (`web/projects.html`) — decisió lliure d'implementació (ADR-04, §7 de
  `design.md`), sempre que naveguin entre seccions sense recarregar tot
  l'estat d'autenticació. · *covers:* AC-01 (base UI d'entrada), AC-08
  (base navegació)
- [x] **T-19** — Aplicar l'accent terracota (ja definit a `PLAN.md` §4)
  com a color primari de l'espai de projectes — badges d'estat, botó
  "afegir" — en lloc del verd molsa ja usat per la llista de la compra
  (ADR-04). Implementar el canvi d'estat com una actualització de
  text/badge, sense animació de desplaçament (ADR-03) — cap gestió
  especial de `prefers-reduced-motion` en aquest espai perquè no hi ha
  moviment. Implementar navegació completa per teclat (afegir, canviar
  estat en qualsevol direcció, eliminar sense ratolí — AC-11) i la
  regió `aria-live="polite"` amb el format exacte "{nom} ara està
  {estat}" en cada canvi (propi o remot via sondeig). · *covers:* AC-02,
  AC-03, AC-04, AC-05, AC-07 (base UI, sense error visible), AC-08,
  AC-09, AC-11, AC-12, AC-14, AC-15, EC-06, NFR-06, NFR-07, NFR-08
- [x] **T-20** — Cablejar `syncProjectsFromServer()` (T-15) al mateix
  cicle ja implementat a `web/js/main.js`:
  `setInterval(syncFromServer, 10000)` + listener de focus de finestra,
  sense inventar un segon mecanisme de sincronització. · *covers:* AC-06

### Verification

- [x] **T-21** — Afegir test E2E (Playwright) de diferenciació visual
  (AC-08): navegar a l'espai de projectes i confirmar, per comparació
  amb la llista de la compra (NIU-1), que la pestanya/entrada de
  navegació i l'accent de color (terracota vs. verd molsa) són
  clarament diferents; complementar amb revisió visual puntual
  (`code-reviewer`/`ux-ui-designer`) documentada com a evidència. ·
  *covers:* AC-08
- [x] **T-22** — Afegir tests unitaris a `internal/projects` per a la
  validació de nom, `budget` i `target_date`: frontera 200/201
  caràcters per al nom (EC-01/EC-02) i per al pressupost (EC-16),
  format vàlid/invàlid de `target_date`, i confirmació explícita que
  una data passada s'accepta sense error (EC-17). · *covers:* AC-10,
  EC-01, EC-02, EC-16, EC-17
- [x] **T-23** — Afegir tests unitaris/d'integració a `internal/projects`
  per a la normalització de duplicats (ADR-02): reutilització de
  `items.NormalizeName` confirmada per trim+minúscules+NFC/NFD, i que
  un duplicat es rebutja **independentment de l'estat** de l'element
  existent (`idea`, `decidit` o `fet` — EC-03), amb les mateixes
  combinacions retallat+majúscules/minúscules ja provades a NIU-1. ·
  *covers:* AC-01, EC-03
- [x] **T-24** — Afegir tests d'integració per a AC-02/AC-03/AC-09 i
  EC-12/EC-13 (`tests/integration/projects_test.go`, mateix patró
  `newTestServer`/`doJSON` ja establert): `PATCH` en les tres direccions
  possibles (`idea`→`decidit`, `decidit`→`fet`, i reversions
  `fet`→`decidit`, `decidit`→`idea`) assertant autoria i moment de
  canvi actualitzats; `PATCH`/`DELETE` sobre un `id` inexistent → `404
  not_found` sense afectar altres elements (EC-12); doble `DELETE`
  idempotent sense 5xx (EC-13). · *covers:* AC-02, AC-03, AC-04, AC-09,
  EC-12, EC-13
- [x] **T-25** — Afegir tests d'integració per a AC-05 i EC-04
  (`projects_test.go`): `POST` → `GET` (persistència, AC-01);
  `DELETE` elimina l'element de la llista i no reapareix en un `GET`
  posterior; crear→eliminar→recrear el mateix nom s'accepta com a
  element nou (la comprovació de duplicat només mira elements
  actius). · *covers:* AC-05, EC-04
- [x] **T-26** — Afegir tests d'integració de seguretat a
  `projects_test.go`, reutilitzant exactament els mateixos patrons de
  test ja escrits per NIU-1/NIU-4 (`security_test.go`,
  `sql_static_test.go`) aplicats a les noves rutes: `POST` amb nom
  `<img src=x onerror=alert(1)>` desat literalment i, en navegador real
  (Playwright), assert de no-execució de script (EC-08/NFR-02); `POST`
  amb `'; DROP TABLE items;--` (o equivalent contra `projects`) desat
  literalment amb la resta de dades intactes (EC-09/NFR-03); assaig
  estàtic de la taula de rutes `chi` estès per confirmar que cap ruta
  `GET` sota `/api/v1/projects` té efecte mutador (EC-10/NFR-04);
  petició sense cookie de sessió vàlida contra qualsevol endpoint
  d'aquest espai rebutjada com a no autenticada (EC-11/NFR-05). ·
  *covers:* EC-08, EC-09, EC-10, EC-11, NFR-02, NFR-03, NFR-04, NFR-05
- [x] **T-27** — Afegir test d'integració per a NFR-01
  (`projects_test.go`): per cada transició d'AC-02/AC-03/AC-09, assert
  d'una fila `events` amb `kind="project_state_changed"` i el `payload`
  `{"from", "to"}` correcte, per inspecció directa post-transacció; i
  que dos clients simulats convergeixen al mateix estat via `GET`
  posterior (AC-06). · *covers:* AC-06, NFR-01
- [x] **T-28** — Afegir test d'integració de concurrència per a AC-07
  (`projects_test.go`, mateix patró que `concurrency_test.go` d'NIU-1):
  dues `PATCH` gairebé simultànies sobre el mateix element (potser a
  estats diferents) via goroutines, assert que cap falla amb 5xx i que,
  després d'un `GET` posterior, ambdós clients recuperarien el mateix
  estat final (l'última escriptura confirmada a SQLite). · *covers:*
  AC-07
- [x] **T-29** — Afegir test unitari/integració per a EC-05 i EC-06
  (`internal/projects` i/o `projects_test.go`): un element sembrat amb
  `updated_at` simulat de setmanes/mesos enrere continua visible amb el
  seu estat real sense cap caducitat, arxivament ni ocultació
  automàtica (EC-05); confirmar que no existeix cap estat
  "abandonat/descartat" al `CHECK` de `state` ni a l'API — l'única
  acció disponible per a un element no desitjat és `DELETE` (EC-06). ·
  *covers:* EC-05, EC-06
- [x] **T-30** — Afegir test E2E (Playwright) d'accessibilitat i estat
  buit per a AC-11/AC-12/EC-14/EC-15 (`tests/e2e/specs/projects.spec.js`):
  navegació completa per Tab/Enter per afegir, canviar estat (en
  qualsevol direcció) i eliminar sense ratolí (AC-11); contingut exacte
  de la regió `aria-live` en un canvi propi i en un canvi reflectit per
  sondeig (AC-12); espai buit en primer ús mostra l'estat visual clar
  sense error (EC-14); viewport mòbil 375×667 manté totes les
  funcionalitats (EC-15). · *covers:* AC-11, AC-12, EC-14, EC-15,
  NFR-06, NFR-07
- [x] **T-31** — Afegir test d'integració per a AC-14/AC-15
  (`projects_test.go`): crear un projecte amb `budget`/`target_date`
  informats i un altre sense — assert que els valors es desen i es
  mostren sencers quan són presents, i que són `null` (sense mostrar
  cap camp) quan es deixen buits. · *covers:* AC-14, AC-15
- [x] **T-32** — Actualitzar `docs/overview.md` per esmentar l'espai de
  compres grans / projectes de casa com a funcionalitat existent de
  Niu (cicle de vida de tres estats, autoria, pressupost i data
  objectiu opcionals), mantenint-lo com a font única de veritat del que
  fa l'app. · *covers:* AC-13
- [x] **T-33** — Executar `commands.test` (`cd app && go test ./...`),
  `commands.lint` (`gofmt -l`) i `commands.typecheck` (`cd app && go vet
  ./...`) del manifest de punta a punta; confirmar que tots els casos
  🟢 de NIU-5 (15 AC + 17 EC) estan verds sense regressions sobre la
  suite existent d'NIU-1/NIU-4. · *covers:* verificació final de totes
  les AC/EC/NFR (tasca de tancament tècnic, no substitueix cap
  cobertura individual anterior)

### Closing (universal — all changes)

- [ ] **C-01** — Append changelog entry (`docs.changelog` from manifest)
- [ ] **C-02** — Transition backlog item to `Human Review` via the adapter
- [ ] **C-03** — Propose semver bump (ASK USER — never apply unattended)

## 2. AC ↔ tasks traceability matrix

| AC | Statement (short) | Covering tasks |
| --- | --- | --- |
| AC-01 | Afegir una idea nova | T-03, T-04, T-08, T-14, T-18, T-23, T-25, T-33 |
| AC-02 | Marcar una idea com a decidida | T-05, T-09, T-15, T-19, T-24, T-33 |
| AC-03 | Marcar un element decidit com a fet | T-05, T-09, T-15, T-19, T-24, T-33 |
| AC-04 | Cada element mostra qui l'ha tocat i quan | T-11, T-16, T-19, T-24, T-33 |
| AC-05 | Eliminar un element | T-06, T-10, T-17, T-19, T-25, T-33 |
| AC-06 | Dos usuaris veuen el mateix estat | T-07, T-20, T-27, T-33 |
| AC-07 | Canvi d'estat concurrent convergeix sense error | T-05, T-19, T-28, T-33 |
| AC-08 | Espai visualment diferenciat | T-13, T-18, T-19, T-21, T-33 |
| AC-09 | Retrocedir un estat és possible | T-05, T-09, T-15, T-19, T-24, T-33 |
| AC-10 | Nom d'element obligatori i acotat | T-02, T-04, T-22, T-33 |
| AC-11 | Navegació completa per teclat | T-19, T-30, T-33 |
| AC-12 | Anunci per lectors de pantalla en canviar d'estat | T-16, T-19, T-30, T-33 |
| AC-13 | `overview.md` reflecteix el nou espai | T-32, T-33 |
| AC-14 | Afegir pressupost opcional (text lliure) | T-02, T-04, T-16, T-19, T-31, T-33 |
| AC-15 | Afegir data objectiu opcional | T-02, T-04, T-16, T-19, T-31, T-33 |

## 3. Edge cases ↔ tasks

| EC | Statement (short) | Covering tasks |
| --- | --- | --- |
| EC-01 | Nom buit o només espais | T-04, T-22 |
| EC-02 | Nom al límit de longitud (200/201) | T-04, T-22 |
| EC-03 | Nom d'idea duplicat (qualsevol estat) | T-03, T-04, T-08, T-23 |
| EC-04 | Duplicat exacte permès post-eliminació | T-08, T-25 |
| EC-05 | Element estancat indefinidament a `decidit` | T-06, T-29 |
| EC-06 | Eliminar vs. marcar abandonada (no existeix estat) | T-19, T-29 |
| EC-07 | Format del camp de pressupost (text lliure, resolt) | T-04 |
| EC-08 | Injecció HTML/JS al nom (XSS) | T-16, T-26 |
| EC-09 | Injecció SQL al nom | T-01, T-26 |
| EC-10 | Intent de mutació via `GET` | T-12, T-26 |
| EC-11 | Accés sense sessió autenticada | T-12, T-26 |
| EC-12 | Canviar l'estat d'un element ja eliminat | T-06, T-09, T-24 |
| EC-13 | Eliminar un element ja eliminat (idempotent) | T-06, T-10, T-24 |
| EC-14 | Llista buida en primer ús | T-17, T-30 |
| EC-15 | Viewport mòbil | T-17, T-30 |
| EC-16 | Pressupost al límit de longitud (200/201) | T-04, T-22 |
| EC-17 | Data objectiu en el passat | T-04, T-22 |

## 4. NFRs ↔ tasks

| NFR | Statement (short) | Covering tasks |
| --- | --- | --- |
| NFR-01 | Cap canvi d'estat esborra l'historial d'autoria (`events`) | T-06, T-27 |
| NFR-02 | Cap nom es renderitza com a HTML (XSS) | T-16, T-26 |
| NFR-03 | Cap valor concatenat a SQL | T-01, T-26 |
| NFR-04 | Cap mutació via `GET` | T-12, T-26 |
| NFR-05 | Tots els endpoints requereixen sessió vàlida | T-12, T-26 |
| NFR-06 | Contrast AA i operabilitat per teclat | T-19, T-30 |
| NFR-07 | Canvis d'estat anunciats a tecnologies d'assistència | T-19, T-30 |
| NFR-08 | `prefers-reduced-motion` (no aplicable — sense animació) | T-13, T-19 |

## 5. Out of scope (mirrored from design)

> Fora d'abast d'aquest ítem — cap tasca de la llista anterior ha
> d'incloure res d'això.

- Integració amb comerços, cercadors de preus o enllaços a productes
  concrets.
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos.
- Gamificació (ratxes, punts) sobre aquest espai.
- Qualsevol relació tècnica o de model de dades amb `internal/items`
  (NIU-1) més enllà de reutilitzar `items.NormalizeName` (ADR-02) —
  col·leccions independents, cap taula ni tipus compartit més enllà
  d'aquesta funció pura.
- Vista d'historial o d'anàlisi sobre projectes passats més enllà de
  l'estat actual de cada element.
- Caducitat automàtica, arxivament o ocultació d'elements estancats a
  `decidit` (EC-05) — comportament esperat, no llacuna.
- Estat diferenciat d'"abandonat/descartat" (EC-06) — eliminar és
  l'única acció disponible en v1.
- Camp de notes lliures — només pressupost (text lliure) i data
  objectiu s'inclouen en v1 (AC-14/AC-15).
- Maqueta pixel-perfect de Stage 1.5 — omesa per decisió humana; §7 de
  `design.md` és l'única font de decisions visuals per a aquest ítem.
- Tres columnes per estat o drag-and-drop entre zones — disposició
  confirmada com una sola llista amb badge d'estat (design.md §10).
- Animació de transició d'estat (vol, FLIP, desplaçament) — ADR-03,
  canvi d'estat és només actualització de text/badge.
- `deleted_at` / esborrat tou — `DELETE` dur, mateixa postura que
  `internal/items`.

## 6. Notes for the developer

- **Ordre estricte de dependència:** T-01/T-02 (migració + domini) abans
  de tocar el servei (T-03...); el domini i el repositori (T-03–T-09)
  abans dels handlers HTTP (T-10–T-13); backend abans del wiring
  frontend (T-14...). No saltar fases.
- **`internal/items` no es toca** excepte per importar la funció
  exportada `NormalizeName` des d'`internal/projects` (ADR-02) — cap
  altre símbol, cap canvi de fitxer dins d'`internal/items`.
- **`items_handlers.go`, `auth_handlers.go`, `csrf.go` no es toquen.**
  Només `router.go` rep un canvi quirúrgic (registrar el nou grup de
  rutes) i `main.go` rep el wiring nou (T-12/T-13).
- **Migració número 003:** la següent disponible després de
  `001_initial_schema.sql` i `002_seed_users.sql` ja existents. Sense
  seed de dades — taula nova sense files prèvies.
- **Reutilització de patrons de test:** `newTestServer`/`doJSON` de
  `tests/integration/helpers_test.go`, i els patrons ja escrits a
  `security_test.go`/`sql_static_test.go`/`concurrency_test.go` es
  reutilitzen sense modificació per a T-24 a T-29 — no inventar un
  altre harness.
- **NFR-08 (animació):** confirmat com a no aplicable a `design.md`
  ADR-03 — no cal cap test de `prefers-reduced-motion` en aquest espai
  perquè no hi ha cap transició de moviment que respectar. T-19 només
  ha de confirmar que el canvi d'estat és una actualització de
  text/badge, no implementar cap gestió addicional.
- **Comandes locals:** `cd app && go test ./...`, `gofmt -l .` (accepta
  un path final), `cd app && go vet ./...`, `cd app && go build ./...`;
  Playwright: `cd app/tests/e2e && npx playwright test` (T-21/T-30 s'hi
  afegeixen).
