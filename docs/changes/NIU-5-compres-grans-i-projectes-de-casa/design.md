---
artefact: design
key: "NIU-5"
title: "Compres grans i projectes de casa"
status: "approved"
owner: "software-architect"
requirements_path: "./requirements.md"
adr_count: 4
sources:
  - "arc42 (subset: §1 introduction, §4 solution strategy, §5 building blocks, §6 runtime, §8 cross-cutting, §11 risks)"
  - "ADR format (Michael Nygard, 2011)"
  - "C4 model — Levels 1 (context) and 2 (containers)"
created: "2026-08-02"
updated: "2026-08-02"
---

# Design — Compres grans i projectes de casa

> **Què és això.** La resposta tècnica als 15 AC / 17 EC / 8 NFR de
> `requirements.md`. Afegeix un domini nou (`internal/projects`) al costat
> d'`internal/items` (NIU-1), reutilitzant exactament els mateixos patrons
> de capes, el mateix middleware d'autenticació (NIU-4, sense cap canvi) i
> el mateix mecanisme d'`events`. No toca `internal/items` ni cap handler
> existent. Referència de projecte: [`../../architecture.md`](../../architecture.md)
> §"Capes"/"Dades". No hi ha maqueta pixel-perfect de Stage 1.5 (omesa per
> decisió humana) — §7 d'aquest document fixa les decisions visuals
> mínimes necessàries perquè `fullstack-developer` no hagi d'improvisar.

## 1. Introducció i restriccions (arc42 §1)

- **Objectiu d'aquest canvi:** entregar un espai nou i independent —
  "Projectes" — amb un cicle de vida de tres estats (`idea` → `decidit` →
  `fet`, reversible en totes dues direccions), complint els 15 AC i 17 EC
  de `requirements.md`, sense tocar el domini `items` (NIU-1) ni el
  middleware d'auth (NIU-4).
- **Restriccions (no negociables, de `PLAN.md` §2/§3/§4, vinculants):**
  - Tècnica: mateix binari Go 1.25 + `chi/v5` + `modernc.org/sqlite` +
    `pressly/goose/v3` embedded; mateixa estructura `internal/<domini>` +
    `store` + `httpapi`; cap dependència nova (no cal `bcrypt`, `norm` ja
    disponible de NIU-1 per a la normalització de duplicats — mateix
    algorisme, EC-03).
  - Organitzacional: repositori **públic** — cap dada personal a fixtures
    ni migracions de seed (S11, ja establert).
  - Temps/cost: NIU-5 és independent d'NIU-2/NIU-3 en execució
    (`proposal.md` §6, "fora d'abast": cap relació tècnica amb `items`);
    reutilitza tot el que ja existeix d'NIU-1/NIU-4 sense re-obrir-ho.
  - Disseny: Stage 1.5 explícitament omesa — §7 d'aquest document és
    l'única font de decisions visuals per a aquest ítem; `PLAN.md` §4
    (paleta, tipografia, forma) continua sent vinculant.
  - Funcional (ja tancat, no re-litigable): 3 estats simples, pressupost
    en text lliure, sense estat "abandonat", eliminar és l'única baixa.

## 2. Estratègia de solució (arc42 §4)

1. **Domini nou `internal/projects`, no una extensió d'`internal/items`**
   (ADR-01) — cicle de vida, camps i regles de duplicat prou diferents
   per justificar un `Repository`/`Service` propis; `items` no es toca.
2. **Un projecte és una fila que canvia d'`state`, mai un parell
   delete+insert** (`PLAN.md` §2.4, mateix principi que `items.location`
   a NIU-1) — cada transició de `AC-02/AC-03/AC-09` és un `UPDATE` que
   també actualitza `last_updated_by`/`updated_at`, sense perdre mai
   `added_by`/`created_at`.
3. **`events` reutilitzada tal qual** — cap columna ni taula nova per a
   l'historial; cada transició escriu una fila `kind="project_state_changed"`
   amb el `payload` JSON (`from`, `to`), exactament el mateix mecanisme
   que `item_moved` a NIU-1 (NFR-01).
4. **Duplicats: mateix algorisme de normalització d'NIU-1
   (`items.NormalizeName`), reaplicat sobre `projects`, amb un abast més
   ampli** (ADR-02) — EC-03 exigeix bloquejar duplicats a través de
   **tots** els estats (`idea`/`decidit`/`fet`), no només entre dues
   caixes actives com NIU-1; l'índex únic parcial reflecteix aquest abast.
5. **Última escriptura guanya per timestamp del servidor, mateixa
   solució que ADR-01 d'NIU-1** — AC-07 (canvi d'estat concurrent) es
   resol amb `BEGIN IMMEDIATE` + `UPDATE` d'una sola transacció, sense
   optimistic locking, reproduint fil per fil el patró ja auditat a
   `store.ItemsRepository.Update`.
6. **Sense estat "abandonat": `DELETE` dur, sense soft-delete** (EC-06,
   ja tancat) — el mateix `DELETE FROM projects WHERE id = ?` idempotent
   que `items.Delete`, amb `events` conservant l'historial complet
   (`project_deleted`).
7. **Reutilització completa de l'auth existent** (EC-11, ja tancat) —
   `internal/httpapi` munta les noves rutes dins del mateix grup
   `/api/v1` protegit per `WithCurrentUser` + `RequireCSRF`
   (NIU-4, ADR-03/ADR-05 del seu `design.md`); zero codi d'autenticació
   nou.
8. **Sondeig + refetch-on-focus, mateix mecanisme d'NIU-1** — `GET
   /api/v1/projects` s'afegeix al mateix cicle de `syncFromServer()` ja
   implementat a `web/js/store.js`, sense inventar un segon mecanisme de
   sincronització (AC-06).
9. **Sense animació de transició d'estat** (ADR-03, resol NFR-08) — el
   canvi d'estat es representa com una actualització visual d'una
   etiqueta/badge, no un vol FLIP entre contenidors; `prefers-reduced-motion`
   no aplica perquè no hi ha moviment a reduir.
10. **Espai visualment diferenciat via nova ruta i accent de color, no un
    component nou al sistema** (ADR-04, resol AC-08) — nova pestanya de
    navegació + accent terracota (enfront del verd molsa ja usat per la
    llista de compra), reutilitzant els mateixos tokens de `PLAN.md` §4.
11. **Pressupost i data objectiu: columnes opcionals de tipus simple**
    — `budget TEXT NULL` (mateix llindar 1–200 que el nom, EC-16) i
    `target_date TEXT NULL` (data ISO-8601 `YYYY-MM-DD`, sense CHECK de
    rang — EC-17 exigeix acceptar dates passades).

## 3. Decisions arquitectòniques (ADRs)

### ADR-01 — Domini nou `internal/projects`, no una extensió d'`internal/items`

- **Status:** accepted
- **Context:** `proposal.md` §6 ("fora d'abast") ja descarta explícitament
  qualsevol relació de model de dades amb `items` — són "dues col·leccions
  d'informació independents amb cicles de vida diferents". Calia decidir
  si això es tradueix en un paquet Go/taula nous o en estendre `items`
  amb un camp de "tipus" que bifurqui comportament.
- **Decision:** `internal/projects` és un paquet nou, estructuralment
  idèntic a `internal/items` (mateix trio `Repository`/`Service`/tipus de
  domini), amb una taula pròpia `projects`. `internal/items` **no es
  toca** — ni el seu esquema, ni el seu codi, ni els seus tests.
- **Consequences:** (+) zero risc de regressió sobre els 31+18 tests ja
  verds d'NIU-1/NIU-4; (+) el cicle de vida de 3 estats i els camps
  opcionals (`budget`, `target_date`) no contaminen el model d'`items`
  amb camps `NULL` que mai usaria; (+) coherent amb l'"Independent"
  d'INVEST ja marcat a `requirements.md` §2. (−) alguna duplicació
  estructural (dues taules `Repository` amb formes semblants) — acceptat:
  la duplicació és més barata que l'acoblament fals d'una taula
  `items` amb un `CHECK` de tipus i camps opcionals segons el tipus.
- **Alternatives considered:** columna `kind` a `items` que distingeixi
  "compra" de "projecte" (rebutjat: `proposal.md` ho prohibeix
  explícitament — no és una extensió de la mateixa entitat); una taula
  compartida amb un esquema més ampli i columnes `NULL` segons el tipus
  (rebutjat: barreja dues validacions i dos cicles de vida en un sol
  `CHECK`, complicant cada test futur d'`items`).

### ADR-02 — Duplicats: mateix algorisme de normalització, abast ampliat a tots els estats

- **Status:** accepted
- **Context:** EC-03 exigeix bloquejar duplicats retallats i insensibles a
  majúscules **a través de qualsevol estat** (`idea`/`decidit`/`fet`), a
  diferència d'NIU-1 (`items`, EC-06 del seu propi `requirements.md`) que
  només compara entre dues caixes actives. La normalització Unicode
  (NFC + trim + lower) ja és correcta i auditada a `items.NormalizeName`.
- **Decision:** `internal/projects` reutilitza `items.NormalizeName` tal
  qual (funció exportada, cap duplicació de l'algorisme) per calcular
  `name_normalized` abans d'inserir. La diferència respecte a NIU-1 és
  només l'**abast** de l'índex: `CREATE UNIQUE INDEX
  idx_projects_name_normalized ON projects(name_normalized)` **sense** cap
  clàusula `WHERE state != ...` — un duplicat es rebutja
  independentment de l'estat de la fila existent (`idea`, `decidit` o
  `fet`), exactament com exigeix EC-03. Com que aquest ítem no introdueix
  soft-delete (ADR de NIU-1 §6.2 ja ho havia descartat per `items`, i
  aquí es manté la mateixa postura), un projecte **eliminat** (`DELETE`
  dur) deixa de comptar per a la comprovació de duplicat — coherent amb
  EC-04.
- **Consequences:** (+) zero algorisme nou a mantenir/testejar — el mateix
  corpus de tests Unicode ja validat per NIU-1 (accents catalans, NFC/NFD)
  cobreix aquest cas per transitivitat; (+) la regla "a través de
  qualsevol estat" es tradueix en un índex més simple que el d'`items`
  (sense clàusula `WHERE`). (−) `internal/projects` importa
  `internal/items` només per aquesta funció — un acoblament petit i
  intencionat (una funció pura, sense estat, sense importar `net/http`
  ni `database/sql`), documentat aquí perquè no sembli un descuit de
  l'ADR-01.
- **Alternatives considered:** duplicar `NormalizeName` dins de
  `internal/projects` (rebutjat: divergiria en silenci si algú corregeix
  un cas Unicode a `items` sense recordar-se de l'altra còpia — el mateix
  algorisme ha de viure en un sol lloc); `COLLATE NOCASE` de SQLite
  (rebutjat pels mateixos motius que ADR-02 d'NIU-1: ASCII-only).

### ADR-03 — Sense animació de transició d'estat (NFR-08 no aplicable)

- **Status:** accepted
- **Context:** `requirements.md` NFR-08 condiciona l'aplicabilitat
  d'aquest NFR a una decisió visual de Stage 1.5 — explícitament omesa
  per decisió humana. Calia decidir, en absència d'una maqueta, si el
  canvi d'estat es representa amb algun tipus de moviment/transició (que
  activaria l'exigència de `prefers-reduced-motion`) o com una simple
  actualització de contingut.
- **Decision:** el canvi d'estat es representa com l'actualització d'una
  etiqueta/badge de text dins de la mateixa fila (p. ex. `idea` → un
  `<span class="badge">` que canvia de text i de color), sense cap
  desplaçament (`transform`), sense FLIP, sense reordenació animada de la
  llista. Un canvi d'estat pot, com a màxim, disparar un `fade`
  d'opacitat curt (150ms) sobre el badge mateix — **mai** un vol o un
  desplaçament de fila com el d'NIU-1. **Conseqüència directa:** NFR-08
  es marca **no aplicable** en aquest disseny — no hi ha cap animació de
  moviment que respectar amb `prefers-reduced-motion`, tal com
  `requirements.md` §6 ja anticipava com a desenllaç possible.
- **Consequences:** (+) zero dependència del mòdul `flip.js` d'NIU-1 per
  a aquest espai — més simple d'implementar i de testejar; (+) tanca
  NFR-08 sense deixar-lo com una ambigüitat silenciosa (documentat aquí,
  no només "descartat" sense explicació). (−) menys "polish" visual que
  la llista de la compra — acceptat conscientment: aquest espai té un
  ritme d'ús molt més baix (canvis d'estat esporàdics, no diverses vegades
  al dia) i no es beneficia del mateix nivell d'inversió en micro-interacció.
- **Alternatives considered:** reutilitzar el mateix FLIP d'NIU-1 entre
  seccions "Idees"/"Decidit"/"Fet" si la UI s'organitza en tres columnes
  (considerat però no obligatori: `requirements.md` no exigeix cap
  disposició en columnes, només que l'estat sigui visible i canviable;
  imposar tres columnes seria una decisió visual que ni tan sols Stage
  1.5 va arribar a prendre — es deixa com a opció vàlida per a
  `fullstack-developer`, vegeu §7, sense exigir-hi animació).

### ADR-04 — Diferenciació visual: nova ruta + accent de color, no un component nou

- **Status:** accepted
- **Context:** AC-08 exigeix que l'espai es distingeixi clarament,
  "només pel visual", de la llista de la compra, mantenint la mateixa
  estètica càlida de `PLAN.md` §4. Sense maqueta de Stage 1.5, calia
  fixar quines decisions de navegació/color són vinculants perquè
  `fullstack-developer` no hagi d'inventar-se l'estructura de zero.
- **Decision:** dues decisions mínimes, cap altra:
  1. **Navegació:** una pestanya/enllaç nou al nivell superior de l'app
     (p. ex. una barra de navegació simple amb dues entrades: "🛒 Compra"
     i "🏠 Projectes"), servit des del mateix `index.html`/embed.FS o com
     a document separat (`web/projects.html`) — **decisió d'implementació
     lliure per a `fullstack-developer`**, sempre que naveguin entre
     seccions sense recarregar tot l'estat d'autenticació (AC-08 no exigeix
     una SPA amb router, només accessibilitat i claredat visual).
  2. **Accent de color:** l'espai de projectes usa **terracota** com a
     color d'accent primari (badges d'estat, botó d'afegir), en lloc del
     verd molsa ja emprat per la llista de la compra — ambdós ja definits
     als tokens de `PLAN.md` §4 ("moss green and terracotta accents"), de
     manera que no calen valors nous, només una assignació diferent dels
     ja existents.
  Fora d'aquestes dues decisions, la resta (disposició en llista simple
  vs. columnes per estat, iconografia exacta dels tres estats) queda
  explícitament oberta per a `fullstack-developer` — vegeu §7 i §9.
- **Consequences:** (+) AC-08 queda satisfet amb el mínim canvi
  estructural (cap component nou al sistema, cap maqueta a esperar);
  (+) coherent amb l'estètica ja aprovada, sense introduir cap valor de
  disseny nou. (−) sense maqueta, hi ha marge d'interpretació en la
  disposició exacta — mitigat explícitament a §7/§9 com a judici obert
  per a implementació, no com una ambigüitat de requisit.
- **Alternatives considered:** esperar una maqueta dedicada abans de
  dissenyar (rebutjat: la porta humana ja va confirmar ometre Stage 1.5
  per aquest ítem — esperar-la contradiria aquesta decisió); reutilitzar
  el mateix verd molsa amb només un títol de secció diferent (rebutjat:
  no compliria "clarament" d'AC-08 — dues seccions amb idèntica paleta i
  sense cap altre senyal visual són fàcils de confondre en un cop d'ull).

## 4. Building blocks (arc42 §5 + C4 Nivell 2)

> Només els components que aquest canvi toca. `internal/items`,
> `internal/auth`, `internal/config`, `internal/store` (part d'`items`) i
> tot `web/js/*` existent no canvien de forma i no es repeteixen aquí.

```text
┌────────────────────────────────────────────────────────────────┐
│                     cmd/niu/main.go                              │
│  (afegeix: wiring de projects.Service + ProjectsRepository,       │
│   cap canvi a l'autenticador ni al cablejat existent d'items)     │
└────────────────────────────────────────────────────────────────┘
        │                          │                     │
        ▼                          ▼                     ▼
┌──────────────────┐      ┌──────────────────┐   ┌──────────────────┐
│ internal/projects │      │ internal/httpapi  │   │ internal/store    │
│ (NOU)             │      │ (afegit, no       │   │ (afegit, no       │
│ DOMINI: Service,  │◀─────│  modificat)       │──▶│  modificat per a  │
│ Repository        │ crida│ projects_handlers │   │  items)           │
│ (interfície),      │      │.go (nou), rutes  │   │ ProjectsRepository│
│ validació, reutil-│      │ noves sota grup   │   │ implementa        │
│ itza              │      │ existent          │   │ projects.Repo-    │
│ items.Normalize-  │      │ WithCurrentUser + │   │ sitory i          │
│ Name (ADR-02)      │      │ RequireCSRF ja    │   │ projects.EventSink│
└──────────┬────────┘      │ existents (NIU-4) │   │ (Record delegat a │
           │               └──────────────────┘   │  la mateixa taula │
           │                                       │  events)          │
           ▼                                       └──────┬───────────┘
    ┌──────────────┐                                       ▼
    │ internal/items│ (només la funció NormalizeName,       ┌─────────────┐
    │ (NO modificat)│  no cap altre símbol)                 │ SQLite:     │
    └──────────────┘                                        │ projects+   │
                                                              │ events      │
                                                              │ (esquema    │
                                                              │  existent)  │
                                                              └─────────────┘
```

- **`internal/projects` (domini, nou)** — responsabilitat: validació de
  nom (mateix llindar 1–200 que `items`), validació de `budget`
  (1–200 opcional), validació de `target_date` (data vàlida opcional,
  sense restricció de passat), transicions d'estat (`idea`↔`decidit`↔`fet`),
  esdeveniment a emetre a `events`. Defineix `Repository` (interfície:
  `Create`, `Get`, `List`, `UpdateState`, `Delete`,
  `ExistsByNormalizedName`) i reutilitza el tipus `items.User` per a
  `added_by`/`last_updated_by` (mateixa forma, cap duplicació). **No**
  importa `net/http` ni `database/sql`. Importa `internal/items` només
  per `items.NormalizeName` (ADR-02).
- **`internal/store`** — s'estén (no es reescriu) amb
  `ProjectsRepository`, estructuralment paral·lel a `ItemsRepository`:
  mateix patró `BEGIN IMMEDIATE` per a `UpdateState` (ADR concurrència,
  §5 Flux 2), mateixa forma de `Record` reutilitzant la taula `events`
  existent (cap canvi a l'`EventSink` — `projects.EventSink` és la
  mateixa interfície mínima `Record(ctx, userID, kind, payload)` que
  `items.EventSink`, implementada pel mateix `*Store`/`ItemsRepository`
  o per un tipus germà trivial que delega a la mateixa taula).
- **`internal/httpapi`** — afegeix `projects_handlers.go`
  (`handleListProjects`, `handleCreateProject`, `handlePatchProjectState`,
  `handleDeleteProject`) i `dto.go` s'estén amb `projectDTO`. **Modifica
  `router.go`** només per registrar el nou grup de rutes
  `/api/v1/projects` dins del `r.Route("/api/v1", ...)` ja existent,
  reutilitzant `WithCurrentUser` (ja muntat al grup) i
  `RequireCSRF(s.authenticator.SessionSecret())` (mateix patró que
  `/items`) per a les mutacions. **`items_handlers.go`, `auth_handlers.go`
  i `csrf.go` no es toquen.**
- **`web/` (nous fitxers, no llistats fil per fil — vegeu §7)** — una
  nova secció/pestanya de navegació i els mòduls JS mínims per llistar,
  afegir, canviar d'estat i eliminar projectes, reutilitzant
  `api.js`/`a11y.js` existents (estenent-los amb els wrappers
  `getProjects`/`addProject`/`patchProjectState`/`deleteProject`, mateix
  patró que els wrappers d'`items`) i `app.css`/tokens ja existents.

## 5. Vista d'execució (arc42 §6)

**Flux 1 — Afegir una idea nova (AC-01, EC-01–EC-03, EC-16, EC-17):**

1. Client envia `POST /api/v1/projects {"name": "...", "budget":
   "..."|null, "target_date": "YYYY-MM-DD"|null}`.
2. `httpapi` decodifica el body, crida
   `projects.Service.Add(ctx, userID, rawName, rawBudget, rawTargetDate)`.
3. `Service` retalla i valida `name` (1–200, mateixes regles de caràcters
   de control que `items`), valida `budget` si present (1–200 després de
   retallar), valida `target_date` si present (format de data vàlid,
   **sense** cap comprovació de "no passat" — EC-17), normalitza el nom
   (`items.NormalizeName`, ADR-02) i crida
   `Repository.ExistsByNormalizedName` dins la mateixa transacció que
   l'`INSERT` (mateix patró check-then-insert protegit per l'índex únic
   que `ItemsRepository.Create`).
4. Si existeix (en **qualsevol** estat, EC-03): retorna
   `ErrDuplicate{}`; `httpapi` respon `409 duplicate_project`.
5. Si no existeix: `INSERT` amb `state = 'idea'`, `added_by = userID`,
   `last_updated_by = userID`, escriu un event
   `project_added`. Respon `201` amb el projecte complet.

**Flux 2 — Canviar d'estat, en qualsevol direcció (AC-02/AC-03/AC-09,
EC-12/EC-13, concurrència AC-07):**

1. Client envia `PATCH /api/v1/projects/{id} {"state": "idea"|"decidit"|"fet"}`
   (idempotent — sempre un valor absolut, mai un "següent estat"
   implícit, mateix principi que `PATCH /items/{id}` d'NIU-1).
2. `httpapi` → `projects.Service.ChangeState(ctx, userID, id, newState)`.
3. **Validació de transició:** com que les tres direccions són vàlides
   (AC-09: `idea`↔`decidit`↔`fet` en ambdós sentits), `Service` només
   verifica que `newState` sigui un dels tres valors coneguts — **no**
   hi ha una màquina d'estats amb transicions prohibides; qualsevol dels
   tres valors és sempre un moviment vàlid des de qualsevol dels altres
   dos (`requirements.md` AC-09 ho confirma explícitament: "en qualsevol
   direcció").
4. `Repository.UpdateState` obre `BEGIN IMMEDIATE` (mateix patró que
   `ItemsRepository.Update`, ADR-01 d'NIU-1) i executa un sol `UPDATE
   projects SET state=?, last_updated_by=?, updated_at=CURRENT_TIMESTAMP
   WHERE id=?`. Si `id` no existeix (EC-12, projecte ja eliminat):
   `RowsAffected == 0` → `ErrNotFound` → `httpapi` respon `404 not_found`.
5. **Èxit:** `COMMIT`, escriu un event `project_state_changed` amb
   `payload {"from": "...", "to": "..."}`, respon `200` amb el projecte
   actualitzat.
6. **Concurrència (AC-07):** si A i B canvien l'estat del mateix projecte
   gairebé alhora, cada `PATCH` és una transacció independent protegida
   per `BEGIN IMMEDIATE`; SQLite les serialitza (un sol escriptor, la
   segona espera fins a `busy_timeout(5000)` i després s'executa). Cap de
   les dues falla amb 5xx; ambdues responen `2xx`. En el proper
   `syncFromServer()` (sondeig o focus), tots dos clients recuperen
   l'estat final únic via `GET /api/v1/projects` i convergeixen sense
   error — **la darrera escriptura confirmada a SQLite guanya**,
   exactament el mateix comportament observable que ADR-01 d'NIU-1
   (§3 d'aquell `design.md`), reaplicat aquí sense cap variació.

**Flux 3 — Eliminar un projecte (AC-05, EC-13):**

1. Client envia `DELETE /api/v1/projects/{id}`.
2. `Service.Delete` crida `Repository.Delete` (`DELETE FROM projects
   WHERE id = ?`, idempotent: `existed bool` igual que `items.Delete`).
3. Si `existed`: escriu event `project_deleted` amb el `payload`
   `{"project_id": id}`.
4. Respon `204` sempre (existís o no prèviament) — una segona `DELETE`
   sobre el mateix `id` també `204` (EC-13, mateix patró EC-11 d'NIU-1).

## 6. Contractes i model de dades

### 6.1 API

Tots els endpoints sota `/api/v1/projects`, dins del mateix grup
`/api/v1` ja protegit per `WithCurrentUser` (NIU-4); les mutacions
reutilitzen `RequireCSRF` exactament com `/api/v1/items`.

| Endpoint | Mètode | Petició (alt nivell) | Resposta (alt nivell) |
| --- | --- | --- | --- |
| `/api/v1/projects` | `GET` | — | `200 { "projects": [Project...] }` |
| `/api/v1/projects` | `POST` | `{ "name": string, "budget": string\|null, "target_date": "YYYY-MM-DD"\|null }` | `201 Project` \| `400 validation_failed` \| `409 duplicate_project` |
| `/api/v1/projects/{id}` | `PATCH` | `{ "state": "idea"\|"decidit"\|"fet" }` (idempotent, absolut) | `200 Project` \| `404 not_found` \| `400 validation_failed` |
| `/api/v1/projects/{id}` | `DELETE` | — | `204` (idempotent, EC-13) |

**`Project` (forma de resposta):**

```json
{
  "id": 7,
  "name": "Televisor nou",
  "state": "decidit",
  "budget": "uns 600€, potser més",
  "target_date": "2026-12-01",
  "added_by": { "id": 1, "display_name": "Usuari A", "avatar_emoji": "🐦" },
  "last_updated_by": { "id": 2, "display_name": "Usuari B", "avatar_emoji": "🦊" },
  "created_at": "2026-08-01T09:00:00Z",
  "updated_at": "2026-08-02T18:30:00Z"
}
```

`budget`/`target_date` són `null` si no es van informar (AC-14/AC-15).
`last_updated_by` és **sempre no-nul** (a diferència de `moved_by` a
`items`): a la creació ja s'assigna `last_updated_by = added_by`, ja que
AC-04 exigeix que "si ha canviat d'estat" es mostri qui ho ha fet, però
no diu res sobre l'estat inicial — assignar-lo des del principi evita un
`null` innecessari i simplifica el frontend (sempre hi ha un avatar a
mostrar, mai cal la lògica condicional "un sol avatar si mai s'ha mogut"
que sí calia a `items`).

**Envelope d'error (reutilitza `apiError` existent, cap tipus nou):**

```json
{ "error": { "code": "duplicate_project", "message": "«Televisor nou» ja existeix." } }
```

Codis nous a NIU-5: `duplicate_project` (EC-03), reutilitza
`validation_failed`/`not_found`/`internal_error`/`unauthenticated`/`csrf_failed`
ja existents (NIU-1/NIU-4) sense cap variació de contracte.

**Mai `GET` amb efecte (EC-10/NFR-04):** mateixa comprovació estàtica que
NIU-1 (assaig de la taula de rutes `chi`), estesa per cobrir també
`/api/v1/projects/*`.

**Autenticació (EC-11/NFR-05):** cap ruta nova és pública — totes viuen
dins del `r.Route("/api/v1", ...)` que ja aplica `WithCurrentUser`; zero
excepcions, zero codi d'auth nou.

### 6.2 Model de dades (deltes sobre l'esquema existent)

| Entitat | Canvi | Risc de migració |
| --- | --- | --- |
| `projects` | **Taula nova.** Vegeu migració 003 més avall. | LOW — taula nova, sense dades prèvies |
| `events` | **Cap canvi d'esquema.** S'escriu des del dia 1 (`project_added`, `project_state_changed`, `project_deleted`), mateix mecanisme que `items` (NFR-01). | LOW |
| `items`, `users`, `sessions` | **Cap canvi.** | — |

**Migració goose (`app/migrations/003_projects.sql`):**

```sql
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
```

**Notes de la migració:**

- `state` amb `CHECK` + `DEFAULT 'idea'` reflecteix AC-01 (estat inicial
  sempre `idea`) i tanca la porta a un quart valor accidental — coherent
  amb la decisió tancada de 3 estats simples (`requirements.md` §0/§8).
  Afegir un quart estat en el futur (marcat com a extensió additiva a
  `requirements.md` §0) només requeriria estendre aquest `CHECK` en una
  migració posterior, sense tocar cap fila existent.
- `budget`/`target_date` són `TEXT` (no `NUMERIC`/`DATE`), coherent amb
  la decisió tancada de text lliure per a `budget` (EC-07 resolt) i amb
  SQLite no tenint un tipus de data natiu — `target_date` es desa com a
  cadena `YYYY-MM-DD` i es valida a Go (format, no rang: EC-17 exigeix
  acceptar dates passades sense restricció).
- `last_updated_by` s'assigna igual a `added_by` a la creació (§6.1) —
  cap `NULL` transitori que calgui gestionar al frontend.
- No cal cap canvi a `002_seed_users.sql` ni a l'esquema d'`users`.

## 7. Decisions visuals mínimes per a `fullstack-developer` (arc42 §5, extensió — Stage 1.5 omesa)

> Aquesta secció existeix perquè `proposal.md` §8 va deixar l'especificació
> visual explícitament pendent i la porta humana va confirmar ometre
> l'Etapa 1.5 per aquest ítem. El que segueix **no és una maqueta** —
> és el conjunt mínim de decisions que un desenvolupador necessita per no
> haver d'improvisar-ne l'estructura, seguint ADR-03/ADR-04.

**Vinculant (no marge d'interpretació):**

- Nova entrada de navegació ("🏠 Projectes" o equivalent) visible des de
  qualsevol punt de l'app, costat a costat amb l'accés a la llista de la
  compra (ADR-04).
- Accent terracota (ja definit a `PLAN.md` §4) com a color primari
  d'aquest espai — badges d'estat, botó "afegir".
- Cada element mostra, com a mínim: nom, badge d'estat (amb text llegible,
  no només color — AC-11/contrast), avatar de qui l'ha afegit, avatar +
  data de qui l'ha actualitzat per últim cop (AC-04), `budget`/`target_date`
  només si estan informats (AC-14/AC-15).
- Cada element permet canviar l'estat en **qualsevol** direcció (AC-09) —
  p. ex. un selector/menú de tres opcions, mai només "següent estat".
- Canvi d'estat: actualització de text/badge, sense animació de
  desplaçament (ADR-03) — `prefers-reduced-motion` no requereix cap
  gestió especial aquí perquè no hi ha moviment.
- Estat buit (EC-14): missatge clar de "cap projecte encara", mai un
  error ni una taula buida sense context.
- Accessibilitat: mateix nivell WCAG 2.2 AA que NIU-1 (AC-11/AC-12/NFR-06/NFR-07)
  — navegació completa per teclat, regió `aria-live="polite"` que anunciï
  "{nom} ara està {estat}" en cada canvi (propi o remot, via sondeig).
- Responsive (EC-15): el contingut s'adapta a mòbil sense perdre cap
  funcionalitat, mateix patró que NIU-1 (no necessàriament el mateix
  disseny de pestanyes — una llista vertical simple ja compleix
  "s'adapta" sense necessitar el mecanisme de tabs d'`items`).

**Confirmat a la porta humana (§10) — ja no és un judici obert:**

- Disposició: **una sola llista** amb badge d'estat visible a cada fila.
  No es construeixen tres columnes ni cap mecanisme de drag-and-drop
  entre zones.

**Judici obert per a `fullstack-developer` (documentat com a decisió
d'implementació, no com una ambigüitat de requisit):**

- Iconografia exacta per a cada estat (emoji, icona SVG, o només text) —
  ha de mantenir-se llegible i coherent amb el to càlid de `PLAN.md` §4,
  sense inventar cap component nou al sistema de disseny.
- Si es mostra com a document HTML separat (`web/projects.html`) o com a
  vista dins del mateix `index.html` amb un router client mínim — ambdues
  opcions són vàlides sota el mateix `embed.FS`/`http.FileServer` ja
  existent (§4).

## 8. Concerns transversals (arc42 §8)

- **Seguretat:** cap superfície nova d'autenticació — reutilització
  completa d'`auth.PasswordAuthenticator` i `RequireCSRF` (NIU-4). Mateixa
  disciplina `textContent`/CSP sense `unsafe-inline` per a `name`/`budget`
  (EC-08/NFR-02), mateixos paràmetres vinculats a SQL (EC-09/NFR-03), cap
  ruta `GET` amb efecte (EC-10/NFR-04), cap endpoint fora del middleware
  d'auth (EC-11/NFR-05).
- **Observabilitat:** fora d'abast explícit d'aquest ítem (NIU-3, no
  encara desplegat). `internal/projects`/`internal/httpapi` registren
  logs bàsics via `log/slog` seguint el mateix nivell mínim que la resta
  de l'app.
- **Rendiment:** sense NFR de rendiment específic per a aquest ítem
  (`requirements.md` §5 no en defineix cap); `GET /api/v1/projects` és una
  única consulta amb `JOIN` a `users` (dues vegades, `added_by`/
  `last_updated_by`), sense N+1, mateix patró que `ItemsRepository.List`.
- **Resiliència:** EC-12/EC-13 (idempotència de `PATCH`/`DELETE` sobre
  elements ja eliminats), `busy_timeout(5000)` ja establert (NIU-1)
  cobreix contenció d'escriptura entre els dos usuaris.
- **Compliance i privacitat:** S11 — cap dada personal committejada a
  fixtures ni migracions d'aquest ítem; noms genèrics `Usuari A`/`Usuari
  B` reutilitzats, cap canvi respecte al que ja existeix.
- **Accessibilitat:** WCAG 2.2 AA (NFR-06), navegació completa per teclat
  (AC-11), regió `aria-live="polite"` (AC-12/NFR-07) amb el format "{nom}
  ara està {estat}" — vegeu §7.
- **i18n/l10n:** cap canvi — UI en català fix, mateix que la resta de Niu.
- **Animació/moviment (NFR-08):** **no aplicable** — ADR-03 documenta
  explícitament que aquest disseny no introdueix cap transició animada
  entre estats; per tant no hi ha res que `prefers-reduced-motion` hagi
  de desactivar en aquest espai concret.

## 9. Riscos (arc42 §11)

| ID | Risc | Severitat | Mitigació | Owner |
| --- | --- | --- | --- | --- |
| R-01 | Sense maqueta de Stage 1.5, la disposició visual final podria no diferenciar-se prou clarament de la llista de la compra malgrat ADR-04 | LOW | §7 fixa navegació + accent de color com a vinculants; `ux-ui-designer`/`code-reviewer` verifiquen AC-08 a `/audit` amb una comparació visual directa, mateix mecanisme que R-04 del `design.md` d'NIU-1 | `code-reviewer` |
| R-02 | Un quart estat futur (p. ex. separar "pressupostat" de "decidit") exigiria estendre el `CHECK` de `state` i revisar tota transició assumida com "sempre vàlida" (§5 Flux 2) | LOW | Documentat a §6.2 com a canvi additiu; `events` ja captura cada transició individual, així que dividir un estat és additiu, no destructiu, tal com ja anticipa `requirements.md` §0 | `software-architect` (revisitar a un futur `/define`) |
| R-03 | `target_date` com a `TEXT` sense `CHECK` de format permet, en teoria, una cadena no-data si la validació de Go es bypassa (bug, no atac) | LOW | La validació de format viu a `internal/projects` (mateixa disciplina que la resta del domini — mai confiar només en el tipus de columna SQLite, que no valida `TEXT`); test unitari cobreix format invàlid | `fullstack-developer` |
| R-04 | Reutilitzar `items.NormalizeName` crea un acoblament (petit) entre dos dominis que `proposal.md` volia mantenir independents | LOW | ADR-02 documenta explícitament per què és preferible a duplicar l'algorisme; l'acoblament és d'una sola funció pura, no d'estat ni de taules — no trenca la independència funcional que `proposal.md` §6 exigeix | `software-architect` (acceptat) |

## 10. Preguntes obertes — RESOLTES a la porta humana (2026-08-02)

- [x] **Reutilització d'`items.NormalizeName` des d'`internal/projects`
  (ADR-02): confirmada.** `internal/projects` importa directament
  `items.NormalizeName` — no es mou a cap paquet compartit nou. Decisió
  tancada.
- [x] **Disposició visual: llista única amb badge d'estat, no tres
  columnes.** Confirmat — l'estat es mostra com una etiqueta/badge de
  color sobre una llista única, sense zones separades per estat ni
  drag-and-drop. `task-planner` genera les tasques de UI amb aquesta
  disposició, no com a judici obert per `fullstack-developer`.
