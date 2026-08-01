---
artefact: design
key: "NIU-1"
title: "Llista de la compra ↔ rebost (auth stubbed)"
status: "draft"
owner: "software-architect"
requirements_path: "./requirements.md"
adr_count: 4
sources:
  - "arc42 (subset: §1 introduction, §4 solution strategy, §5 building blocks, §6 runtime, §8 cross-cutting, §11 risks)"
  - "ADR format (Michael Nygard, 2011)"
  - "C4 model — Levels 1 (context) and 2 (containers)"
created: "2026-08-01"
updated: "2026-08-01"
---

# Design — Llista de la compra ↔ rebost (auth stubbed)

> **Què és això.** La resposta tècnica als requeriments de `requirements.md`.
> Refina `PLAN.md` §2 (vinculant) sense contradir-lo. No repeteix
> comportament funcional — referencia AC/EC per ID. Referència de projecte:
> [`../../architecture.md`](../../architecture.md).

## 1. Introducció i restriccions (arc42 §1)

- **Objectiu d'aquest canvi:** entregar el backend Go + frontend vanilla
  de la llista de la compra ↔ rebost amb autenticació stubbed, complint
  els 16 AC i 17 EC de `requirements.md` i deixant el seam d'auth llest
  per a NIU-4.
- **Restriccions (no negociables, de `PLAN.md` §2/§3/§6, vinculants):**
  - Tècnica: un sol binari Go 1.25 servint `/api/v1/*` i `web/` via
    `embed.FS`; `modernc.org/sqlite` (pure Go, sense CGO); `chi/v5`;
    `pressly/goose/v3` embedded; `golang.org/x/text/unicode/norm`
    (normalització NFC obligatòria per a EC-06 — vegeu ADR-02); frontend
    sense build step, sense framework. `internal/` amb domini (`items`)
    desacoblat d'HTTP i SQL.
  - Organitzacional: repositori **públic** — cap dada personal (S11),
    identitats genèriques `Usuari A`/`Usuari B` fins que l'entorn les
    injecti.
  - Temps/cost: NIU-1 és l'únic ítem d'aquesta sessió
    (`single_item_per_session`); NIU-2/3/4 en depenen, no al revés.
  - Disseny: §8 de `proposal.md` és l'única font del sistema visual — cap
    component nou fora del seu inventari (§8.4).

## 2. Estratègia de solució (arc42 §4)

1. **Hexagonal lleuger via `internal/`.** `items` (domini) exposa
   interfícies (`Repository`, `Clock`) que `store` implementa amb SQLite;
   `httpapi` només coneix `items.Service`, mai `database/sql` ni `chi`
   directament dins del domini.
2. **Un ítem és una fila, mai un parell delete+insert** (PLAN.md §2.4) —
   `location` és l'única cosa que canvia en un moviment; això dona
   `moved_by`/`moved_at` "gratis" i un punt únic de contenció per fila.
3. **Última escriptura guanya per timestamp del servidor** (ADR-01) — la
   concurrència de AC-09/CF-12 es resol amb una regla determinista i
   observable, no amb un ordre d'arribada arbitrari de la xarxa.
4. **Duplicats: normalització a Go abans de comparar/inserir** (ADR-02) —
   `TRIM` + `strings.ToLower` amb `unicode` conscient d'accents, no
   `COLLATE NOCASE` de SQLite (ASCII-only), evitant fals-negatius amb
   noms catalans accentuats.
5. **`position REAL` amb reindexació perezosa** — inserir entre dues
   posicions és una sola `UPDATE`; només es renumera tota una caixa quan
   l'espai entre posicions veïnes cau per sota d'un llindar de precisió
   de `float64` (rar en ús de dues persones).
6. **Middleware d'auth com a seam d'interfície, no com a lògica inline**
   (ADR-03) — `httpapi` crida `auth.Authenticator.CurrentUser(r)`; NIU-1
   la implementa com un stub que sempre retorna l'usuari seed A; NIU-4
   substitueix la implementació sense tocar cap handler.
7. **Frontend com a mòduls ES sense estat compartit global mutable no
   controlat** — un únic mòdul `store.js` manté l'array `items` en
   memòria; `render()` és pura respecte a aquest estat, seguint
   exactament el patró ja provat a `design-system/screen-*.html`.
8. **Sondeig + refetch-on-focus** (ja congelat a PLAN.md §2.6) —
   implementat com un `setInterval(10000)` més un listener de `focus` a
   `window`, ambdós cridant la mateixa funció `syncFromServer()`.
9. **Test mid-write kill com a script Go dedicat, no com a test de
   CI ordinari** (ADR-04) — coherent amb qa-engineer, executat a demanda i
   documentat com a procediment repetible.
10. **Errors uniformes i genèrics cap al client** — un únic tipus
    `apiError{code, message}` a `httpapi`, mai un `error.Error()` intern
    filtrat cap enfora (S3/NFR-01 i "no leak" de PLAN.md §2.5).

## 3. Decisions arquitectòniques (ADRs)

### ADR-01 — Resolució de concurrència: última escriptura guanya per timestamp del servidor

- **Status:** accepted
- **Context:** AC-09/CF-12 exigeixen que dues `PATCH` gairebé simultànies
  sobre el mateix ítem no fallin mai amb 5xx i que ambdós clients
  convergeixin a un únic estat observable després d'un refetch.
  `qa-engineer` va assenyalar que calia un guanyador **determinista**, no
  només "sense error".
- **Decision:** cada `PATCH /api/v1/items/{id}` és una transacció SQLite
  que llegeix la fila, aplica els canvis sol·licitats i escriu
  `updated_at = CURRENT_TIMESTAMP` (nova columna, ADR implica migració —
  vegeu §6) dins la mateixa transacció, protegida per `busy_timeout(5000)`
  i el bloqueig d'escriptor únic que SQLite ja imposa (una sola escriptura
  physical a la vegada, `WAL` permet lectors concurrents). **No hi ha
  optimistic locking amb `If-Match`/versió**: la segona petició que arriba
  al servidor (per ordre real d'execució de la transacció SQLite, que és
  seqüencial per disseny) simplement sobreescriu el resultat de la
  primera. El "guanyador" és, doncs, **la petició l'escriptura de la qual
  es confirma en darrer lloc a la base de dades** — equivalent en la
  pràctica a l'ordre d'arribada real processat per SQLite, mai a un
  timestamp de client (no fiable, rellotges no sincronitzats).
- **Comportament observable que el test pot afirmar:** després que
  ambdues `PATCH` retornin (cap 5xx), un `GET /api/v1/items/{id}}`
  (o el `GET /api/v1/items` complet) retorna **una única fila** l'estat
  de la qual (`location`, `moved_by`, `moved_at`) coincideix exactament
  amb el cos de resposta de la `PATCH` l'escriptura de la qual es va
  confirmar en darrer lloc a SQLite — el test d'integració ho verifica
  inspeccionant quina de les dues respostes `PATCH` té el `updated_at`
  més recent i assegurant que el `GET` posterior hi coincideix camp a
  camp.
- **Consequences:** (+) zero complexitat addicional (sense capçaleres
  `If-Match`, sense columna de versió); (+) compleix AC-09 amb un test
  determinista. (−) si en el futur (post-v1) es necessita detectar
  col·lisions per avisar l'usuari ("algú altre ho ha canviat"), caldrà
  afegir versionat — explícitament fora d'abast aquí.
- **Alternatives considered:** timestamp generat pel client (rebutjat:
  rellotges no sincronitzats, vector d'atac trivial); optimistic locking
  amb `version` incremental i 409 en conflicte (rebutjat: AC-09 exigeix
  "cap error", un 409 seria una regressió funcional); CRDT/merge
  (sobreenginyeria per dues persones i un camp `location` enumerat).

### ADR-02 — Duplicats: normalització Unicode a Go, no `COLLATE NOCASE`

- **Status:** accepted
- **Context:** EC-06 exigeix bloquejar duplicats retallats i
  insensibles a majúscules a través de totes dues caixes, amb noms en
  català que poden portar accents (`"Llet"` vs `"LLET"` és trivial en
  ASCII, però `"Pastanaga"` vs `"PASTANAGA"` i futurs noms com `"Àrab"`
  no ho són — `COLLATE NOCASE` de SQLite només plega ASCII A-Z/a-z).
- **Decision:** la normalització passa **sempre** per Go abans de tocar
  SQLite, en **tres** passos i en aquest ordre:
  `norm.NFC.String(...)` → `strings.TrimSpace` → `strings.ToLower`.

  > ⚠️ **El pas NFC no és opcional.** `ToLower` sol NO detecta duplicats
  > quan el mateix text arriba amb bytes diferents. Verificat
  > empíricament: `"Àrab"` en forma composta (`À` = 1 punt de codi, 5
  > bytes) i en forma descomposta (`A` + accent combinant, 6 bytes) es
  > veuen **idèntics a la pantalla** però `lower(NFC) == lower(NFD)` és
  > `false` → l'ítem duplicat **passaria el filtre i entraria a la
  > llista**, incomplint EC-06. Després de normalitzar tots dos a NFC, la
  > comparació és `true`.
  >
  > Això no és teòric: macOS històricament produeix NFD i la majoria de
  > teclats i navegadors produeixen NFC, així que dos ítems amb accent
  > afegits des de dispositius diferents poden xocar. Amb noms en català
  > (`Àrab`, `Maçã`, `Pernil dolç`) el cas és plausible en ús real.
  >
  > Requereix `golang.org/x/text/unicode/norm` — una dependència
  > justificada, no una comoditat.

  `strings.ToLower` sí que és Unicode-aware per al plegat de majúscules
  (verificat: `Llet`/`LLET`, `Pastanaga`/`PASTANAGA`, `Àrab`/`ÀRAB`,
  `Maçã`/`MAÇÃ` coincideixen tots un cop les formes estan normalitzades).
  El valor normalitzat
  es desa en una columna generada `name_normalized TEXT NOT NULL` amb un
  índex `UNIQUE` **parcial** (`WHERE deleted_at IS NULL`, vegeu §6) sobre
  aquesta columna — la unicitat és una restricció de base de dades, no
  només una comprovació prèvia a l'aplicació (evita una finestra de
  carrera entre "comprovar" i "inserir" si mai hi hagués dues escriptures
  concurrents de creació amb el mateix nom). `items.name` (el valor
  mostrat) mai es normalitza — es desa exactament com l'usuari l'ha
  escrit (EC-04).
- **Consequences:** (+) unicitat correcta per a qualsevol alfabet
  Unicode, no només ASCII; (+) la restricció a nivell de BD elimina la
  race condition de "check-then-insert"; (+) el missatge d'error (8.4.3
  proposal.md) pot incloure la caixa on ja existeix perquè Go ja sap
  quina fila ha xocat abans de decidir el missatge. (−) una columna
  addicional a mantenir sincronitzada amb `name` a cada `INSERT`
  (mitigat: només s'escriu a `INSERT`, `name` mai es actualitza via
  `PATCH` en aquest v1 — no hi ha "editar nom").
- **Alternatives considered:** `COLLATE NOCASE` (rebutjat: ASCII-only,
  fals-negatiu amb accents); normalització amb `golang.org/x/text/unicode/norm`
  NFC + case-fold complet (considerat però innecessari per al corpus
  esperat — noms curts catalans/emoji, no texts multilingües amb
  ortografies especials; es documenta com a millora futura si mai calgués
  suport per a alfabets amb regles de plegat més complexes).

### ADR-03 — Seam d'autenticació: interfície `auth.Authenticator`, stub en NIU-1

- **Status:** accepted
- **Context:** PLAN.md §8/NIU-1 exigeix que `httpapi` ja passi per un
  middleware d'auth encara que la implementació sigui fixa, perquè
  NIU-4 substitueixi la implementació "no la forma".
- **Decision:** `internal/auth` defineix `type Authenticator interface {
  CurrentUser(r *http.Request) (User, error) }`. `httpapi` crida
  sempre `auth.FromContext(ctx)` (injectat per un middleware
  `WithCurrentUser(authenticator)`) — mai llegeix cookies/headers
  directament als handlers. NIU-1 registra `auth.StubAuthenticator{
  UserID: <seed A>}` que ignora la petició i retorna sempre el mateix
  usuari. `GET /api/v1/me` delega a la mateixa interfície. NIU-4
  substituirà `StubAuthenticator` per `SessionAuthenticator` (llegeix
  cookie, valida hash contra `sessions`) sense canviar cap handler
  d'`items`.
- **Consequences:** (+) zero canvis a `httpapi/items_handlers.go` quan
  arribi NIU-4; (+) `items.Service` rep sempre un `userID` tipat, mai un
  `*http.Request`. (−) cap, és el cost mínim per mantenir el contracte
  net (una interfície de dos mètodes).
- **Alternatives considered:** hardcodejar `userID` constant dins dels
  handlers (rebutjat: NIU-4 hauria de tocar cada handler, violant
  l'exigència explícita de PLAN.md/§8 proposal.md); flag de
  configuració que canviï de mode dins del mateix `Authenticator`
  (rebutjat: barreja dues responsabilitats en una implementació,
  complica el test de cadascuna per separat).

### ADR-04 — Test EC-15/REL-01 (mid-write kill): script Go dedicat fora del pipeline normal de CI

- **Status:** accepted
- **Context:** REL-01/EC-15 exigeixen que un `SIGKILL` a mig `UPDATE` no
  corrompi SQLite. Matar el procés real de manera fiable i repetible dins
  d'un `go test` normal és fràgil en CI (temporització no determinista,
  contenidors compartits, diferències d'I/O entre runners).
- **Decision:** s'implementa `app/tests/killtest/main.go`, un programa Go
  independent (no `go test`) que: (1) arrenca el binari `niu` real com a
  procés fill apuntant a una BD temporal; (2) llança una goroutine que
  envia contínuament `PATCH /api/v1/items/{id}` a ritme alt; (3) espera un
  interval aleatori (`rand.Intn` entre 50–500ms) i envia `SIGKILL` al
  procés fill; (4) reobre la mateixa BD amb `modernc.org/sqlite` i executa
  `PRAGMA integrity_check`; (5) arrenca el binari de nou i verifica
  `GET /healthz` = 200. Aquest script s'exposa com `make killtest N=10`
  (`internal Makefile target`) que l'itera 10 vegades consecutives
  (NFR-07). **No corre a CI per defecte** — corre com a **procediment
  manual documentat i obligatori** abans de tancar NIU-1 (coherent amb
  `test-plan.md` REL-01/CF-15, que ja el marca com a manual/script
  dedicat). Es documenta a `app/tests/killtest/README.md` amb la comanda
  exacta i el resultat esperat, perquè sigui repetible per qualsevol
  humà o agent sense inventar-se el mecanisme.
- **Consequences:** (+) test real (SIGKILL real, no simulat), reflecteix
  fidelment `docker kill`; (+) no introdueix flakiness al pipeline
  bloquejant de PR. (−) no s'executa automàticament a cada push — mitigat
  perquè `test-plan.md` ja documenta REL-01 com a obligatori-però-manual,
  i `/audit` ha de verificar que s'ha executat abans d'aprovar.
- **Alternatives considered:** `go test` amb `t.Skip` condicional en CI
  (rebutjat: amaga el requisit, no queda com "obligatori" a la vista);
  contenidor Docker real amb `docker kill` orquestrat des de GitHub
  Actions (considerat per a NIU-2, no per NIU-1 — afegiria una
  dependència de Docker-in-Docker al pipeline d'aquest ítem sense
  necessitat, ja que el fill·fill de procés natiu Go aconsegueix el
  mateix efecte sobre el fitxer SQLite sense necessitar contenidors).

## 4. Building blocks (arc42 §5 + C4 Nivell 2)

```text
┌───────────────────────────────────────────────────────────────────┐
│                        cmd/niu/main.go                             │
│         (wiring: config → store → items.Service → httpapi)         │
└───────────────────────────────────────────────────────────────────┘
        │                │                │                │
        ▼                ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ internal/     │  │ internal/     │  │ internal/     │  │ internal/     │
│ config        │  │ store         │  │ items         │  │ httpapi       │
│ env→struct,   │  │ SQLite,       │  │ DOMINI:       │  │ handlers,     │
│ fail-fast     │  │ migracions,   │  │ Service,      │  │ middleware,   │
│ validation    │  │ implementa    │  │ Repository    │  │ routing,      │
│               │  │ items.Repo-   │  │ (interfície), │  │ headers de    │
│               │  │ sitory        │  │ validació,    │  │ seguretat     │
│               │  │               │  │ normalització │  │               │
└──────────────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
                          │                 │  ▲              │
                          │                 │  │ implementa   │
                          │                 │  └──────────────┘
                          ▼                 │  interfície     │
                   ┌─────────────┐          │  auth.Authenticator
                   │ SQLite WAL  │          │              │
                   │ /data/niu.db│          ▼              ▼
                   └─────────────┘   ┌──────────────┐ ┌──────────────┐
                                     │ internal/auth│ │ web/ (embed) │
                                     │ StubAuth-     │ │ HTML/CSS/JS  │
                                     │ enticator +   │ │ servit per   │
                                     │ interfície    │ │ httpapi via  │
                                     │ Authenticator │ │ embed.FS     │
                                     └──────────────┘ └──────────────┘
```

- **`internal/items` (domini)** — responsabilitat: regles de negoci
  (validació de nom, normalització de duplicats, transicions de
  `location`, esdeveniment `events` a emetre). Defineix `Repository`
  (interfície: `Create`, `Get`, `List`, `Update`, `Delete`,
  `ExistsByNormalizedName`) i `EventSink` (interfície: `Record(kind,
  payload)`). **No importa `net/http` ni `database/sql`.** `store`
  implementa ambdues interfícies.
- **`internal/store`** — SQLite via `modernc.org/sqlite`, obertura amb
  els tres `_pragma` (WAL, busy_timeout, foreign_keys), goose embedded
  (`//go:embed migrations/*.sql`), implementació concreta de
  `items.Repository` i `items.EventSink`.
- **`internal/httpapi`** — routing `chi/v5`, handlers prims que
  deserialitzen/serialitzen JSON i deleguen a `items.Service`,
  middleware de capçaleres de seguretat (S7), middleware
  `WithCurrentUser`, servidor d'estàtics des d'`embed.FS` muntat a `/`
  (tot el que no comenci per `/api/v1/` o `/healthz`).
- **`internal/auth`** — interfície `Authenticator` + `StubAuthenticator`
  (NIU-1). `sessions`/`SessionAuthenticator` reals: NIU-4.
- **`internal/config`** — parsing d'entorn, validació fail-fast (rebutja
  arrencar si falta configuració requerida — en NIU-1 pràcticament res és
  requerit encara, ja que `NIU_SESSION_SECRET`/`NIU_USER_*_HASH` són
  "yes (NIU-4)" segons PLAN.md §6; NIU-1 seedeja usuaris placeholder
  directament a la migració, no via env).
- **`web/`** — HTML + CSS + ES modules, `embed.FS`, consumeix
  `design-system/tokens.css` (còpia literal, no reinventar valors) i
  reimplementa l'estructura DOM ja provada a
  `design-system/screen-desktop.html` / `screen-mobile.html`.

## 5. Vista d'execució (arc42 §6)

**Flux 1 — Afegir un ítem (AC-01, EC-01–EC-06):**

1. `AddItemInput` envia `POST /api/v1/items {"name": "..."}`.
2. `httpapi` decodifica el body, crida `items.Service.Add(ctx, userID,
   rawName)`.
3. `items.Service` retalla espais, valida longitud (1–200), rebutja
   caràcters de control, normalitza (ADR-02) i crida
   `Repository.ExistsByNormalizedName` (dins la mateixa transacció que
   l'`INSERT` per evitar la carrera de creació).
4. Si existeix: retorna `ErrDuplicate{ExistingLocation: "pantry"}`;
   `httpapi` respon `409` amb l'envelope d'error, `httpapi` afegeix el
   nom de la caixa al missatge (8.4.3 proposal.md).
5. Si no existeix: `INSERT` amb `position` = `MAX(position)+1` de
   `shopping`, `added_by = userID`, escriu un event `item_added` a
   `events`. Respon `201` amb l'ítem complet.
6. Frontend afegeix la fila amb `fade-in` 150ms (èxit, 8.4.3).

**Flux 2 — Moure un ítem, optimista amb rollback (AC-02/AC-03/AC-09/AC-12/AC-13):**

1. L'usuari activa una fila → JS captura `getBoundingClientRect()` de
   totes les files (`captureRects()`, ja provat a
   `design-system/screen-*.html`), actualitza l'estat local
   (`items` en memòria) **abans** de rebre resposta, crida `render({
   flipFromRects })` → FLIP de 250ms (o cross-fade si
   `prefers-reduced-motion`).
2. En paral·lel, `fetch PATCH /api/v1/items/{id} {"location": "pantry"}`.
3. `httpapi` → `items.Service.Move(ctx, userID, id, newLocation)` → una
   sola transacció: `UPDATE items SET location=?, moved_by=?,
   moved_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP,
   position=? WHERE id=?`. Si `id` no existeix (EC-12, ítem ja
   eliminat): `404` amb error `not_found`.
4. **Èxit (AC-12):** resposta `200` amb l'ítem actualitzat; el client ja
   mostrava aquest estat — no fa res més (cap parpelleig).
5. **Fallada de xarxa o error del servidor (AC-13):** el client reverteix
   l'estat local a la posició prèvia, reprodueix el FLIP invers, mostra
   el `Toast` (8.4.4) amb el nom de l'ítem.
6. **Concurrència (AC-09/CF-12, ADR-01):** si B també mou el mateix ítem
   gairebé alhora, cada `PATCH` és una transacció independent; SQLite
   les serialitza (un sol escriptor). Ambdues responen `2xx`. En el
   proper `syncFromServer()` (sondeig o focus), tots dos clients
   recuperen l'estat final únic via `GET /api/v1/items` i
   re-renderitzen sense error.

**Flux 3 — Sondeig i refetch en focus (AC-08, AC-16 per canvis remots):**

1. Al carregar la pàgina: `syncFromServer()` immediat, després
   `setInterval(syncFromServer, 10000)`.
2. `window.addEventListener('focus', syncFromServer)`.
3. `syncFromServer()` crida `GET /api/v1/items`, compara amb l'estat
   local (per `id` + `location` + `moved_at`), i per a cada ítem el
   `location` del qual ha canviat **des de l'última vegada que
   aquest client el va veure** (canvi remot, no originat per aquest
   client): re-renderitza amb FLIP (si visible a totes dues caixes al
   DOM) i emet l'anunci `aria-live` amb el format "per {usuari}" (8.7
   proposal.md). Ítems purament nous es fan aparèixer amb `fade-in`, no
   FLIP (no hi ha posició d'origen coneguda pel client).

## 6. Contractes i model de dades

### 6.1 API

| Endpoint | Mètode | Petició (alt nivell) | Resposta (alt nivell) |
| --- | --- | --- | --- |
| `/api/v1/items` | `GET` | — | `200` `{ "items": [Item...] }` |
| `/api/v1/items` | `POST` | `{ "name": string }` | `201` `Item` \| `400 validation_failed` \| `409 duplicate_item` |
| `/api/v1/items/{id}` | `PATCH` | `{ "location": "shopping"\|"pantry" }` (idempotent, mai un toggle) | `200` `Item` \| `404 not_found` \| `400 validation_failed` |
| `/api/v1/items/{id}` | `DELETE` | — | `204` (idempotent — 2a crida també `204`, EC-11) |
| `/api/v1/me` | `GET` | — | `200` `{ "id", "name", "display_name", "avatar_emoji" }` (stub, AC-07) |
| `/healthz` | `GET` | — | `200` si `SELECT 1` contra SQLite té èxit; `503` en cas contrari (REL-03/NFR-08) |

**`Item` (forma de resposta):**

```json
{
  "id": 12,
  "name": "Llet",
  "location": "shopping",
  "position": 1.5,
  "added_by": { "id": 1, "display_name": "Usuari A", "avatar_emoji": "🐦" },
  "moved_by": { "id": 2, "display_name": "Usuari B", "avatar_emoji": "🦊" },
  "moved_at": "2026-08-01T10:15:00Z",
  "created_at": "2026-08-01T09:00:00Z"
}
```

`moved_by`/`moved_at` són `null` si l'ítem no s'ha mogut mai des de la
creació (8.4.7 proposal.md: mostra un sol avatar en aquest cas).

**Envelope d'error uniforme (binding, PLAN.md §2.5):**

```json
{ "error": { "code": "duplicate_item", "message": "«Llet» ja hi és a Rebost." } }
```

Codis usats a NIU-1: `validation_failed` (EC-01/02/03/05), `duplicate_item`
(EC-06), `not_found` (EC-12, PATCH/DELETE sobre id inexistent),
`internal_error` (genèric, mai detall intern — S3 secció error).

**Idempotència (EC-11):** `DELETE` sobre un `id` ja eliminat retorna
`204` igualment (no `404`) — la consulta prèvia comprova existència
només per decidir si cal fer `DELETE FROM`/emetre event; l'absència
posterior és l'èxit, no un error.

**Mai `GET` amb efecte (EC-08/NFR-04/S1a):** cap ruta `GET` crida mai
`items.Service.Add/Move/Delete`. Verificat per un test d'integració
que introspecciona la taula de rutes `chi` i assegura que cap handler
`GET` registrat coincideix amb un dels tres mètodes mutadors.

### 6.2 Model de dades (deltes sobre PLAN.md §2.4)

| Entitat | Canvi | Risc de migració |
|---|---|---|
| `items` | + columna `name_normalized TEXT NOT NULL`, + columna `updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`, + columna `deleted_at TIMESTAMP NULL` (esborrat tou per mantenir `events`/`moved_by` consistents i permetre EC-07 sense violar l'índex únic), + índex únic parcial | LOW — taula nova en aquest ítem, sense dades prèvies |
| `users` | Sense canvis d'esquema respecte PLAN.md §2.4; NIU-1 sembra 2 files placeholder (`Usuari A`/`Usuari B`, hash `bcrypt` d'una contrasenya que mai s'usarà fins NIU-4) | LOW |
| `sessions` | Sense canvis; taula creada però sense ús a NIU-1 | LOW |
| `events` | Sense canvis d'esquema; s'escriu des del dia 1 (`item_added`, `item_moved`, `item_deleted`) | LOW |

**Nota sobre `deleted_at` (esborrat tou):** PLAN.md §2.4 no l'esmenta
explícitament, però EC-07 (duplicat exacte permès després d'eliminar)
exigeix que la comprovació d'unicitat "només miri ítems actius, no
l'historial". Amb un `DELETE FROM items` dur, `EC-07` funcionaria igual
de bé (la fila ja no existeix). Aquest disseny opta per **DELETE dur**,
no esborrat tou — més simple i suficient, ja que `events` ja conserva
l'historial (`item_deleted` amb el `payload` complet). `deleted_at` **no
s'afegeix**; es retira aquesta idea de l'esborrany inicial per no
introduir una columna sense ús real. L'índex únic parcial es manté per
robustesa futura però la clàusula `WHERE` es simplifica (vegeu SQL
següent) ja que no hi ha esborrat tou.

**Migracions goose (`app/migrations/`):**

```sql
-- 001_initial_schema.sql
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
```

```sql
-- 002_seed_users.sql
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
```

**`position` — estratègia de fraccionament:** nou ítem = `MAX(position)
WHERE location = ? ) + 1.0` (o `1.0` si la caixa és buida). No hi ha
reordenació manual dins d'una caixa en v1 (fora d'abast — no hi ha AC
que ho demani), així que la inserció entre dues posicions no s'exercita
avui; el disseny reserva el patró (`REAL`, mai enter seqüencial) perquè
un futur "arrossegar per reordenar" no requereixi una migració. Si mai
`(pos_a + pos_b) / 2` convergeix per sota de la precisió útil de
`float64` (~15-17 dígits significatius — pràcticament mai amb desenes
d'ítems), `store` reindexa tota la caixa a enters espaiats de `1.0` dins
d'una transacció.

## 7. Especificitats de SQLite

- **Obertura (binding, PLAN.md §2.3):** DSN
  `file:/data/niu.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)`.
- **Serialització d'escriptures entre els dos usuaris:** SQLite en mode
  WAL permet lectors concurrents il·limitats però **un sol escriptor a
  la vegada**; `busy_timeout(5000)` fa que una segona transacció
  d'escriptura que arribi mentre una altra és en curs esperi fins a 5s
  abans de fallar amb `SQLITE_BUSY`, en lloc de fallar immediatament.
  Amb només dues persones i operacions d'escriptura breus (un `UPDATE`
  d'una fila), la probabilitat real de tocar aquest límit és molt baixa;
  si es toqués, `store` retorna `internal_error` (503-equivalent a nivell
  d'aplicació) i el client optimista fa rollback (Flux 2, pas 5).
  `database/sql`'s `db.SetMaxOpenConns(1)` **no** s'aplica als lectors
  (limitaria WAL innecessàriament); només cal per a la connexió que fa
  `INSERT`/`UPDATE`/`DELETE` si el driver `modernc.org/sqlite` mostra
  contenció observable en proves de càrrega — decisió deferida a
  implementació amb un comentari explícit al codi si s'activa.
- **`foreign_keys(on)`:** aplicat a cada connexió oberta (no és un
  paràmetre de fitxer persistent a SQLite, cal repetir-lo per connexió —
  `modernc.org/sqlite` ho gestiona via la cadena `_pragma` al DSN,
  aplicada automàticament en cada nova connexió del pool).
- **Backup en calent (referència per a NIU-2, no d'abast aquí):** mai
  `cp` del fitxer; `sqlite3 .backup` és responsabilitat de NIU-2/§5.5
  PLAN.md, no d'aquest ítem.

## 8. Arquitectura de frontend (`app/web/`)

```text
app/web/
├── index.html          — un sol document, marcatge base + `<script type="module">`
├── app.css             — importa els mateixos valors que design-system/tokens.css
│                          (valors idèntics, no reinventats — copiar-los, no referenciar
│                          el directori design-system/ en producció)
├── fonts/
│   ├── Nunito-Regular.woff2
│   └── Nunito-Bold.woff2
└── js/
    ├── api.js           — fetch wrappers: getItems(), addItem(), moveItem(), deleteItem(), getMe()
    ├── store.js         — estat en memòria (`items` array), syncFromServer(), diffing per FLIP
    ├── render.js         — render(), renderRow(), renderEmptyState(), renderAvatars() (port directe
    │                          de la lògica ja provada a design-system/screen-desktop.html)
    ├── flip.js           — captureRects(), playFlip() (idèntic a design-system, extret a mòdul propi)
    ├── a11y.js           — announce() (aria-live), gestió de tabindex mòbil
    ├── confetti.js        — confetti() amb guarda d'un sol tret (§8.6.3, vegeu més avall)
    ├── tabs.js            — setActivePanel() (mòbil, breakpoint 768px)
    └── main.js            — punt d'entrada: wiring d'events, `setInterval`, listener de `focus`
```

- **`embed.FS`:** `cmd/niu/main.go` declara `//go:embed web` sobre un
  `var webFS embed.FS` a `internal/httpapi` (o passat com a paràmetre de
  wiring); `httpapi.NewRouter` munta un `http.FileServer` sobre
  `http.FS(webFS)` arrelat a `web/` per a qualsevol ruta que no
  comenci per `/api/v1/` ni sigui `/healthz`. Sense server-side
  templating (PLAN.md §2.1) — `index.html` és estàtic, `js/main.js`
  crida `GET /api/v1/me` en carregar per saber qui és l'usuari actual.
- **Mapatge de tokens:** `app.css` és una còpia dels valors de
  `design-system/tokens.css` (mateixos noms de custom property) —
  `fullstack-developer` no reinventa cap hex/px/ms; els components
  (`ItemRow`, `AddItemInput`, `Toast`, `EmptyState`, `TabBar`, `Avatar`)
  es porten estructuralment (mateixos IDs/classes DOM: `.item-row`,
  `.box`, `#list-shopping`, `#list-pantry`, `#live-region`, etc.) tal
  com ja existeixen a `screen-desktop.html`/`screen-mobile.html`, però
  substituint les dades en memòria estàtiques d'aquell preview per
  crides reals a `api.js`.
- **Sondeig + focus (Flux 3 §5):** implementat a `main.js`, crida
  `store.syncFromServer()`.
- **Optimistic update + rollback (AC-12/AC-13):** `store.js` exposa
  `moveItemOptimistic(id, newLocation)` que (1) actualitza l'array local
  i retorna les `rects` prèvies, (2) crida `render({flipFromRects})`,
  (3) awaita `api.moveItem(id, newLocation)`, (4) en cas d'error,
  restaura l'estat anterior i torna a cridar `render({flipFromRects:
  <rects post-flip>})` per revertir visualment, més
  `toast.show(itemName)`.
- **Guarda de confeti d'un sol tret (AC-14):** variable de mòdul
  `let shoppingWasEmpty = false;` (o equivalent persistit a `sessionStorage`
  si cal sobreviure a un refresc de pàgina — no especificat com a
  requerit per AC-14, que parla de "renderitzats posteriors" dins de la
  mateixa sessió; s'opta per variable de mòdul en memòria, més simple, ja
  que un reload complet és un nou "primer render" i EC-13 ja exigeix no
  disparar confeti en càrrega inicial encara que la caixa ja estigui
  buida). A cada `render()`: si `shoppingItems.length === 0 &&
  !shoppingWasEmpty && wasNonEmptyBefore` → `confetti(); shoppingWasEmpty
  = true`. Es reinicia (`shoppingWasEmpty = false`) tan bon punt
  `shoppingItems.length > 0` en qualsevol render posterior.
- **Mòbil FLIP a través del límit de pestanyes (resolt, vegeu ADR
  implícit a §8.3.2 — comportament, no decisió arquitectònica nova):**
  el DOM manté totes dues llistes (`#list-shopping`, `#list-pantry`)
  sempre renderitzades; només `.panel.is-visible`/`display:none` decideix
  què es veu (com ja implementa `design-system/screen-mobile.html`).
  Comportament exacte que `fullstack-developer` ha de reproduir
  fil per fil (no hi ha marge d'interpretació):
  1. Si l'ítem mogut **roman visible** (l'usuari mou un ítem cap a la
     pestanya que ja és l'activa — no aplicable en aquest domini, ja
     que moure sempre canvia de caixa, per tant sempre implica que
     l'origen és visible i la destinació **no** ho és) — cas real
     únic: **l'ítem sempre desapareix de la pestanya activa i apareix a
     la inactiva.**
  2. La fila es reprodueix amb un FLIP **només dins de la pestanya
     d'origen** (encara visible): un moviment breu cap avall/amunt
     seguit d'una esvaïda (`opacity 1→0`) mentre la resta de files de la
     mateixa caixa es reacomoden amb el seu propi FLIP curt cap a les
     seves noves posicions — **mai** s'anima un desplaçament amb
     `transform` cap a coordenades de la pestanya oculta (aquestes
     coordenades no existeixen visualment, `display:none` no té
     `getBoundingClientRect()` útil).
  3. En arribar el `render()`, la fila apareix a la caixa de destí
     (actualment oculta) amb la classe `.just-added`
     (`opacity 0→1`, 150ms, sense `transform`) — **només visible
     quan l'usuari canvia de pestanya**, moment en el qual ja no hi ha
     animació pendent (la transició ja s'ha completat abans que
     l'usuari pugui veure-la).
  4. Amb `prefers-reduced-motion`: el mateix cross-fade descrit a §8.6.2
     de `proposal.md`, sense el pas intermedi de desplaçament.
  5. L'anunci `aria-live` (AC-16) es dispara **sempre**, independentment
     de si la caixa de destí és visible o no — l'anunci és auditiu/de
     text, no depèn de visibilitat DOM.
  Aquest comportament substitueix qualsevol lectura de "FLIP travessant
  el canvi de pestanya" com un vol físic entre panells (impossible amb
  un panell en `display:none`); el que sí que "travessa" és la
  continuïtat de dades (el mateix objecte `Item` es re-renderitza a
  l'altra llista) i l'anunci `aria-live`, no un `transform` visual entre
  ubicacions ocultes.

## 9. Implementació de seguretat

- **S3 (XSS, NFR-01):** `render.js` crea tots els nodes amb
  `document.createElement` + `.textContent` per a qualsevol dada
  d'usuari (`item.name`, noms d'usuari) — **zero** ús d'`innerHTML` amb
  dades d'usuari a tot `app/web/js/`. La CSP (següent punt) prohibeix
  `'unsafe-inline'`, de manera que un `<script>` injectat via nom d'ítem
  mai s'executaria encara que hi hagués un `innerHTML` erroni — defensa
  en profunditat, no substitut de la disciplina `textContent`.
- **S7 (capçaleres, NFR-02):** middleware `httpapi.SecurityHeaders`
  aplicat a **totes** les respostes (API i estàtics) via `chi.Use(...)`
  al router arrel, abans de qualsevol altre middleware:
  `Strict-Transport-Security: max-age=63072000; includeSubDomains`,
  `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
  `Referrer-Policy: no-referrer`,
  `Content-Security-Policy: default-src 'self'; script-src 'self';
  style-src 'self'; font-src 'self'; img-src 'self'; connect-src 'self';
  object-src 'none'; base-uri 'none'` (sense `'unsafe-inline'` enlloc).
- **S8 (SQL injection, NFR-03):** `internal/store` usa exclusivament
  `database/sql` amb paràmetres vinculats (`?` placeholders, mai
  `fmt.Sprintf` cap a SQL). Un `code-reviewer`/CI grep de
  `fmt.Sprintf.*SELECT|INSERT|UPDATE|DELETE` dins `internal/store/` és
  una comprovació estàtica recomanada a `tasks.md` (no bloquejant per
  disseny, però senzilla d'afegir).
- **Seam d'auth (ADR-03):** vegeu §3. `httpapi.WithCurrentUser` és **on**
  NIU-4 canvia una línia de wiring (`auth.NewSessionAuthenticator(...)`
  en lloc de `auth.NewStubAuthenticator(...)`), mai el router ni els
  handlers d'`items`.
- **Errors sense fuita (PLAN.md §2.5):** `httpapi` mai serialitza
  `err.Error()` d'un error intern (SQL, panic recuperat) cap al client;
  un middleware de recuperació (`chi/middleware.Recoverer` + wrapper propi)
  captura panics i respon `500 {"error":{"code":"internal_error",
  "message":"S'ha produït un error inesperat."}}`, registrant el detall
  només al log del servidor (stdout, capturat per Docker — OTEL a NIU-3).

## 10. Concerns transversals (arc42 §8)

- **Seguretat:** vegeu §9. Sense canvis d'autenticació real en aquest
  ítem (stub, ADR-03); NIU-1 cobreix S3/S7/S8/S11 (S1 parcial via
  EC-08/NFR-04).
- **Observabilitat:** fora d'abast (NIU-3) — `internal/observability`
  existeix com a paquet buit/no cablejat en aquest ítem; els handlers
  registren logs bàsics via `log/slog` a stdout (nivell mínim per
  depurar en local), sense OTEL SDK encara.
- **Rendiment:** NFR-05 (p95 <200ms amb 500 ítems) i NFR-06 (<1s en 3G
  simulada). `GET /api/v1/items` és una única consulta `SELECT` amb
  `JOIN` a `users` (dues vegades, per `added_by`/`moved_by`) i un
  `ORDER BY location, position` — sense N+1 (una sola query, mapatge en
  memòria). NFR-06 es beneficia de zero framework/build i fonts
  autoallotjades amb només 2 pesos (§8.2 proposal.md).
- **Resiliència:** EC-11/EC-12 (idempotència), EC-15/REL-01 (mid-write
  kill, ADR-04), `busy_timeout` com a mitigació de contenció (§7).
  `/healthz` (REL-03) verifica una consulta trivial real, no només que
  el procés respongui.
- **Compliance i privacitat:** S11 — cap dada personal a cap fitxer
  committejat (codi, migracions de seed, fixtures de test); noms
  genèrics `Usuari A`/`Usuari B` arreu, tal com ja fa aquest mateix
  document.
- **Accessibilitat:** WCAG 2.2 AA (NFR-09), navegació completa per
  teclat (AC-15), regió `aria-live="polite" aria-atomic="true"` (AC-16,
  NFR-10) amb el format exacte de §8.7 proposal.md, zones tàctils
  ≥44×44px (EC-16). Implementació d'estructura DOM/aria idèntica a la ja
  validada a `design-system/`.
- **i18n/l10n:** NFR-11 — corpus Unicode complet (EC-04) desat i
  retornat verbatim; UI en català fix (sense selector d'idioma, fora
  d'abast).

## 11. Riscos (arc42 §11)

| ID | Risc | Severitat | Mitigació | Owner |
| --- | --- | --- | --- | --- |
| R-01 | `PRAGMA integrity_check` triga significativament més en una BD gran, alentint el test EC-15/REL-01 en repeticions | LOW | Test executat contra una BD petita representativa (desenes d'ítems), no la mida de producció; documentat a ADR-04 | `fullstack-developer` |
| R-02 | Índex únic parcial sobre `name_normalized` pot xocar amb una futura funcionalitat d'edició de nom (fora d'abast v1) si no es revisita | LOW | Documentat a §6.2; qualsevol futur "editar nom" haurà de repetir la comprovació de duplicat abans de l'`UPDATE`, mateix patró que `Add` | `software-architect` (revisitar a un futur `/define`) |
| R-03 | `busy_timeout(5000)` insuficient sota càrrega inesperada (molt improbable amb 2 usuaris) causant `SQLITE_BUSY` visible a l'usuari | LOW | Rollback optimista + toast ja cobreix aquest cas (Flux 2); revisar si l'ús real ho contradiu | `fullstack-developer` |
| R-04 | Reproduir manualment el port de `design-system/*.html` a `app/web/js/*.js` introdueix drift respecte a l'especificació visual si no es fa amb disciplina | MEDIUM | `code-reviewer`/`ux-ui-designer` verifiquen a `/audit` que el DOM/CSS shipat coincideix estructuralment amb els fitxers de referència | `code-reviewer` |
| R-05 | El test dedicat de mid-write kill (ADR-04) no s'executa perquè no és part del pipeline automàtic de CI | MEDIUM | `/audit` bloqueja explícitament si no hi ha evidència (log/sortida) de les 10 execucions manuals abans d'aprovar NIU-1 | `qa-engineer` |

## 12. Preguntes obertes per a la porta humana

- Cap pregunta funcional pendent (heretat de `requirements.md` §8).
- **Confirmar l'abast de `busy_timeout`/`SetMaxOpenConns` per a la
  connexió d'escriptura** (§7): es proposa deixar-ho sense limitar
  explícitament en aquest ítem i només actuar-hi si les proves de
  càrrega (NFR-05) mostren contenció — demana conformitat abans de
  `task-planner`.
- **Confirmar que `deleted_at` no s'introdueix** (§6.2): s'ha optat per
  DELETE dur en lloc de soft-delete perquè `events` ja cobreix
  l'historial; si el propietari humà prefereix soft-delete per a
  auditoria futura, cal dir-ho abans de generar `tasks.md` (canviaria
  l'esquema de la migració 001).
