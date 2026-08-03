---
artefact: tasks
key: "NIU-6"
title: "Idees d'activitats amb previsualització de link"
status: "approved"
owner: "task-planner"
design_path: "./design.md"
requirements_path: "./requirements.md"
task_count: 44
ac_coverage:
  - ac: "AC-01"
    tasks: ["T-03", "T-04", "T-05", "T-06", "T-11", "T-16", "T-19", "T-23", "T-28"]
  - ac: "AC-02"
    tasks: ["T-03", "T-04", "T-06", "T-07", "T-16", "T-19", "T-24", "T-28"]
  - ac: "AC-03"
    tasks: ["T-06", "T-07", "T-16", "T-19", "T-24", "T-28"]
  - ac: "AC-04"
    tasks: ["T-10", "T-15", "T-19", "T-23", "T-28"]
  - ac: "AC-05"
    tasks: ["T-09", "T-13", "T-17", "T-19", "T-25", "T-28"]
  - ac: "AC-06"
    tasks: ["T-08", "T-17", "T-20", "T-26", "T-28"]
  - ac: "AC-07"
    tasks: ["T-18", "T-19", "T-28"]
  - ac: "AC-08"
    tasks: ["T-02", "T-06", "T-23", "T-28"]
  - ac: "AC-09"
    tasks: ["T-17", "T-19", "T-25", "T-28"]
  - ac: "AC-10"
    tasks: ["T-17", "T-19", "T-27", "T-28"]
  - ac: "AC-11"
    tasks: ["T-16", "T-28"]
sources:
  - "GitHub-style checklist (Markdown task lists)"
  - "Fikua AC↔tasks traceability matrix"
created: "2026-08-03"
updated: "2026-08-03"
---

# Tasks — Idees d'activitats amb previsualització de link

> **Què és això.** El full de ruta d'implementació per a NIU-6. Cada
> tasca és petita (≤ ~1h), autocontinguda, i traçada a almenys un
> criteri d'acceptació de `requirements.md`. **Cap tasca sense un
> AC/EC/NFR que la cobreixi; cap AC/EC/NFR sense almenys una tasca.**
> Aquest fitxer és l'únic artefacte mutable durant `/code` — la resta
> són bloquejats (`design.md`/`requirements.md` aprovats, `design.md`
> ja esmenat post-revisió de seguretat abans d'arribar aquí).
>
> Traducció mecànica de `design.md` (5 ADRs aprovats) — cap decisió de
> disseny nova. `internal/items`, `internal/projects`, `internal/auth`,
> `internal/config` no es toquen. **`internal/fetchsafe` és el component
> de risc més alt d'aquest ítem** (ADR-02, esmenat arran de F-01 a F-07)
> — les tasques T-03 a T-03h el desglossen explícitament mecanisme a
> mecanisme, en lloc d'una sola tasca "implementar fetchsafe", perquè
> cap sub-mecanisme de mitigació de SSRF quedi implícit.

## 1. Task list

### Foundations

- [x] **T-01** — Escriure la migració goose
  `app/migrations/00X_activity_ideas.sql` (número següent al de la
  migració `projects` d'NIU-5) amb el DDL exacte de `design.md` §6.2:
  taula `activity_ideas` (`id`, `url NOT NULL`, `title`, `image_url`,
  `description` — tots `TEXT NULL` sense `CHECK` de longitud —,
  `preview_status TEXT NOT NULL CHECK (preview_status IN
  ('pending','ready','partial','failed')) DEFAULT 'pending'`,
  `added_by REFERENCES users(id)`, `created_at`). **Sense** índex únic
  sobre `url` (EC-06, deduplicació fora d'abast). Totes les consultes
  futures sobre aquesta taula usaran paràmetres vinculats, mai
  `fmt.Sprintf` cap a SQL. · *covers:* NFR-02 (base)
- [x] **T-02** — Definir el domini `internal/ideas`: tipus `Idea`
  (`ID`, `URL`, `Title *string`, `ImageURL *string`, `Description
  *string`, `PreviewStatus`, `AddedBy items.User`, `CreatedAt`),
  interfícies `Repository` (`Create`, `Get`, `List`, `UpdatePreview`,
  `Delete`) i `EventSink` (mateixa forma mínima que `items`/`projects`).
  **No** importa `net/http` directament — només crida
  `fetchsafe.FetchPreview` (design.md §4/ADR-02). No importa
  `items.NormalizeName` (EC-06, sense deduplicació). · *covers:*
  AC-07 (base — espai independent)

### Implementation — `internal/fetchsafe` (component crític, ADR-02 esmenat)

> Cada mecanisme de mitigació de SSRF és una tasca pròpia i verificable
> per separat — `code-reviewer`/`security-engineer` han de poder marcar
> cadascuna com present al codi sense haver-la de deduir d'una tasca
> genèrica "implementar fetchsafe".

- [x] **T-03** — Crear `internal/fetchsafe` amb la signatura pública
  única `FetchPreview(ctx context.Context, rawURL string) (Preview,
  error)` i els errors tipats (`ErrSchemeRejected`,
  `ErrDestinationForbidden`, `ErrTimeout`, `ErrResponseTooLarge`,
  `ErrUnsupportedContentType`). Cap altre paquet de l'aplicació pot
  importar `net/http` per fer una petició cap a una URL controlada per
  l'usuari (design.md §4, R-01). · *covers:* AC-01 (base), AC-02 (base)
- [x] **T-03a** — Implementar la validació d'esquema (NFR-05/EC-01):
  `url.Parse` + comprovació `scheme == "http" || scheme == "https"`
  **abans** de qualsevol activitat de xarxa; rebutjar `file://`,
  `javascript:`, `ftp://`, `data:`, esquema buit amb
  `ErrSchemeRejected`, zero peticions de xarxa. · *covers:* NFR-05
- [x] **T-03b** — Implementar la denylist de noms d'amfitrió
  (NFR-06 — F-03/F-04), comprovada **abans i independentment** de
  qualsevol resolució DNS: `niu.fikua.com` (hardcoded) + valor de
  `NIU_PUBLIC_HOST` (variable d'entorn) + noms de servei coneguts de la
  xarxa Docker `traefik-public` (`otel-collector`, `dozzle`,
  `openobserve`, dashboard de Traefik). Comprovació case-insensitive
  sobre el host de la URL; si coincideix, `ErrDestinationForbidden`
  immediat sense cap crida DNS. · *covers:* NFR-06 (base — vector edge
  públic)
- [x] **T-03c** — Construir el `net.Dialer` amb `ControlContext` (no
  `Control`) com a **únic** punt de validació d'IP (NFR-06/EC-02/EC-07)
  — **no** afegir cap crida separada a
  `net.DefaultResolver.LookupIPAddr` abans o en paral·lel (rebutjat
  explícitament a ADR-02 per crear el parell TOCTOU de F-06). Dins de
  `ControlContext`, per cada IP resolta pel dialer intern de Go: cridar
  `Unmap()` **abans** de qualsevol `Is*()` (F-02), rebutjar
  explícitament els prefixos NAT64 `64:ff9b::/96` i 6to4 `2002::/16`
  (F-07), i acceptar la connexió únicament si
  `addr.IsGlobalUnicast() && !addr.IsPrivate()` post-`Unmap()` (criteri
  d'allowlist, no enumeració de rangs — F-07); rebutjar també
  `IsMulticast()` per claredat encara que ja quedi exclòs per
  `IsGlobalUnicast()`. Si `ControlContext` retorna error, `DialContext`
  avorta abans del `connect()`. · *covers:* NFR-06 (mecanisme central —
  EC-02/EC-07)
- [x] **T-03d** — Configurar `http.Transport` dedicat amb
  `DisableKeepAlives: true` (F-01/F-06) — sense aquesta opció, una
  cadena de redireccions cap al mateix host reutilitza la connexió TCP
  ja oberta i mai torna a cridar `DialContext`/`ControlContext`,
  saltant-se la validació d'IP per complet en aquest cas (verificat
  empíricament per `security-engineer`: 4 salts → 1 sola invocació de
  `Control` sense aquesta opció). · *covers:* NFR-06 (defensa —
  connexions reutilitzades)
- [x] **T-03e** — Implementar `http.Client.CheckRedirect` amb **dues**
  responsabilitats explícites (NFR-06/EC-04, esmenat F-01): (a) limitar
  la cadena a **5 salts** (error propi passats els 5); (b) re-validar
  explícitament dins de `CheckRedirect` que l'esquema de la següent URL
  a seguir és `http`/`https` — rebutjar qualsevol `30x` cap a
  `file://`/`javascript:`/altres esquemes. Aquesta comprovació és una
  capa addicional, no substitueix la validació d'IP de T-03c. ·
  *covers:* NFR-06 (defensa en profunditat — redireccions), NFR-07
  (límit de salts)
- [x] **T-03f** — Configurar el `context.WithTimeout(ctx, 5*time.Second)`
  que embolcalla tota la crida `FetchPreview` (DNS + connexió + TLS +
  capçaleres + cos), no només `http.Client.Timeout` (que es manté com a
  xarxa de seguretat de segon nivell) — un servidor que respon
  capçaleres de seguida però allarga el cos indefinidament també queda
  tallat als 5s. · *covers:* NFR-07 (timeout, EC-08)
- [x] **T-03g** — Implementar el límit de mida en streaming: 2 MiB via
  `io.LimitReader(resp.Body, 2<<20)` aplicat **abans** de passar el
  lector al parser HTML, mai `io.ReadAll(resp.Body)` sense límit. Si el
  `LimitReader` talla el contingut abans d'acabar el `<head>`, tractar
  com a fallback (no com a error fatal del procés). · *covers:* NFR-07
  (límit de mida, EC-03)
- [x] **T-03h** — Construir el `http.Client` dedicat de `fetchsafe` a
  l'arrencada de l'app (una sola instància reutilitzada, no un client
  nou per petició), sense cap `Cookie`/`Authorization`/secret de Niu
  adjunt; l'única capçalera pròpia és un `User-Agent` identificable
  (`Niu-LinkPreview/1.0 (+https://niu.fikua.com)`). No comparteix
  `Transport` amb cap altra part de Niu. · *covers:* NFR-08

### Implementation — parsing Open Graph (ADR-04)

- [x] **T-04** — Implementar el parsing Open Graph amb
  `golang.org/x/net/html` (afegir dependència al `go.mod`): tokenitzar
  el flux HTML ja limitat pel `LimitReader` de T-03g, aturar-se en
  trobar el tancament de `<head>`, extreure `og:title`, `og:image`,
  `og:description` dels `<meta property="og:*">`. Comprovar
  `Content-Type` de la resposta abans de parsejar: si no és compatible
  amb HTML (EC-09 — PDF, imatge, vídeo), no intentar el parsing i
  tractar com a fallback directament. · *covers:* AC-01, EC-09

### Implementation — `internal/ideas` (servei + repositori)

- [x] **T-05** — Implementar `ideas.Service.Add(ctx, userID, rawURL)`:
  validar format sintàctic de URL i rebutjar buit/només-espais
  (EC-10); validar **només l'esquema** (`http`/`https`, NFR-05/EC-01,
  reutilitzant la mateixa comprovació barata que T-03a exposa, sense
  cap petició de xarxa encara). Si l'esquema no és vàlid, retornar
  `ErrSchemeRejected` sense crear cap fila. · *covers:* AC-01 (base),
  EC-01, EC-10
- [x] **T-06** — Completar `ideas.Service.Add`: si el format és vàlid,
  `Repository.Create` insereix la fila amb `preview_status='pending'`,
  camps de previsualització `NULL`; escriure event `idea_added`;
  retornar la idea `pending` immediatament (ADR-03 — la resposta
  `201` no espera el scraping). · *covers:* AC-01, AC-02 (base creació
  immediata)
- [x] **T-07** — Implementar el worker pool acotat de scraping (ADR-03
  esmenat F-05): un semàfor amb buffer de capacitat configurable
  (interval 4–8, vegeu **T-07a** per al valor concret), alimentat per
  `Service.Add` després d'inserir la fila `pending`; cada worker crida
  `fetchsafe.FetchPreview` amb el `context.Background()` de fons
  proporcionat per `main.go` (no el de la petició HTTP, ja tancada en
  respondre). En resoldre's (èxit/parcial/fallada), fa `UPDATE` via
  `Repository.UpdatePreview` amb el resultat i el `preview_status`
  final (`ready`/`partial`/`failed`); **mai reintent automàtic**
  (fora d'abast, `requirements.md` §7). · *covers:* AC-01, AC-02,
  AC-03
- [x] **T-07a** — **Confirmar i documentar el valor exacte de
  concurrència del worker pool dins del rang 4–8 fixat per ADR-03/R-08.**
  Triar un valor concret (p. ex. `6`) segons el consum de memòria
  observat per scrape en curs (~2 MiB retinguts per worker actiu, límit
  del contenidor 128M/0.5CPU a `compose.yaml`) i deixar-lo com a
  constant nomenada al codi (no com a `TODO` sense resoldre) amb un
  comentari d'una línia que en justifiqui el valor (p. ex. "6 workers ×
  2MiB ≈ 12MiB de pic, marge ampli sota el límit de 128M del
  contenidor"). **No bloquejant per a `/audit`** (design.md §9), però
  ha de quedar reflectit al codi abans de tancar aquest ítem. · *covers:*
  NFR-07 (base — límit de concurrència/memòria)
- [x] **T-08** — Cablejar a `main.go` un `context.Background()` propi
  per al worker pool (independent del context de cada petició HTTP) i
  assegurar un tancament ordenat del pool a l'aturada de l'app (ADR-03
  — el pool sobreviu la petició original que l'ha encuat). · *covers:*
  AC-06 (base — resolució en segon pla visible per sondeig)
- [x] **T-09** — Implementar `ideas.Service.Delete(ctx, userID, id)`:
  `DELETE FROM activity_ideas WHERE id = ?` idempotent (`existed
  bool`, mateix patró que `items.Delete`/`projects.Delete`); escriure
  event `idea_deleted` només quan la fila existia; confirmar que un
  `UPDATE` de resolució de scraping en curs sobre un `id` ja eliminat
  no troba files i s'ignora silenciosament (Flux 3, sense cancel·lació
  explícita). · *covers:* AC-05
- [x] **T-10** — Estendre `internal/store` amb `IdeasRepository`,
  implementant `ideas.Repository` i `ideas.EventSink` (delegant
  `Record` a la taula `events` ja existent, cap columna ni tipus nou).
  Implementar `Create` (insereix `pending`), `Get`, `List` (consulta
  única amb `JOIN` a `users` per `added_by`, sense N+1), `UpdatePreview`
  (**únic** `UPDATE` permès un cop creada la fila, mai toca
  `id`/`url`/`added_by`), `Delete`. · *covers:* AC-04

### Implementation — HTTP layer

- [x] **T-11** — Implementar handlers `GET/POST /api/v1/ideas` i
  `DELETE /api/v1/ideas/{id}` a nou fitxer
  `internal/httpapi/ideas_handlers.go`: deserialitzar/serialitzar JSON,
  delegar a `ideas.Service`, mapar `ErrSchemeRejected`/error de format
  → `400 validation_failed`, envelope d'error uniforme (`design.md`
  §6.1, reutilitza `apiError` existent, cap codi nou per a "destí
  prohibit" — coherent amb NFR-06 "missatge clar sense revelar detalls
  interns"). `DELETE` respon sempre `204` (EC-15, idempotent). ·
  *covers:* AC-04
- [x] **T-12** — Estendre `internal/httpapi/dto.go` amb `ideaDTO`
  (mapeig `Idea`→JSON amb la forma exacta de `design.md` §6.1: `id`,
  `url` sempre present, `title`/`image_url`/`description` (`null` si
  `pending`/`failed`, o el camp concret si falta sota `partial`),
  `preview_status`, `added_by`, `created_at`). · *covers:* AC-01, AC-03
- [x] **T-13** — **Modificar `router.go`** (canvi quirúrgic, no
  reescriptura): registrar el nou grup de rutes `/api/v1/ideas` dins
  del `r.Route("/api/v1", ...)` ja existent, reutilitzant
  `WithCurrentUser` (ja muntat al grup) i `RequireCSRF` per a
  `POST`/`DELETE`, mai per a `GET`. Confirmar que cap ruta `GET` crida
  `Add`/`Delete`. **`items_handlers.go`, `projects_handlers.go`,
  `auth_handlers.go`, `csrf.go` no es toquen.** · *covers:* AC-05 (base
  eliminar via API)
- [x] **T-14** — Afegir a `cmd/niu/main.go` el wiring de
  `ideas.Service` + `store.IdeasRepository` + el client HTTP dedicat de
  `fetchsafe` (T-03h) + el worker pool acotat (T-07/T-08), cablejat amb
  el mateix `*Store`/`config`/`authenticator` ja construïts — cap canvi
  al cablejat existent d'`items`/`projects`. · *covers:* AC-01 (base
  disponibilitat end-to-end)

### Implementation — frontend

- [x] **T-15** — Crear `app/web/js/ideas-api.js` (o estendre `api.js`):
  fetch wrappers `getIdeas()`, `addIdea(url)`, `deleteIdea(id)`,
  reutilitzant exactament el mateix patró (capçalera CSRF,
  `handleUnauthenticated()`) que els wrappers d'`items`/`projects` ja
  existents. · *covers:* AC-04 (base transport)
- [x] **T-16** — Crear `app/web/js/ideas-store.js` (o estendre
  `store.js`): estat en memòria (`ideas` array),
  `syncIdeasFromServer()` seguint exactament el mateix cicle de
  `syncFromServer()` ja implementat per `items`/`projects` (sondeig
  ~10s + refetch-on-focus, sense inventar un segon mecanisme). ·
  *covers:* AC-01, AC-02, AC-03, AC-11 (base — dades disponibles per a
  l'anunci `aria-live`)
- [x] **T-17** — Crear `app/web/js/ideas-render.js` (o estendre
  `render.js`): implementar `AddLinkInput` (§8.3 de `proposal.md`) i
  `IdeaCard` amb els **quatre estats** A (`ready`) / B (`failed`) / C
  (`partial`) / D (`pending`, "Recuperant…") mapejats exactament als
  quatre valors de `preview_status`. **Zero `innerHTML` amb dades
  externes o d'usuari** — títol, descripció i URL sempre via
  `document.createElement` + `.textContent` (EC-11/NFR-01); imatge amb
  `alt=""` (decorativa, confirmat a la porta humana de Stage 1.5).
  Camps absents sota `partial` s'ometen visiblement, sense buit
  trencat. Botó d'eliminar per targeta. · *covers:* AC-02, AC-03,
  AC-05, AC-09 (base marcatge estructural per a teclat)
- [x] **T-18** — Crear la nova entrada de navegació de tercer nivell
  ("Idees") aplicant el token `--color-mel` (`#C99A3A`, ja aprovat a
  Stage 1.5) com a accent primari (subratllat `color.mel-hover` per a
  l'entrada activa), diferenciada visualment de l'accent verd molsa
  (NIU-1) i terracota (NIU-5); disposició en graella `auto-fill,
  minmax(240px, 1fr)` per a les targetes (`proposal.md` §8.2). ·
  *covers:* AC-07
- [x] **T-19** — Implementar la regió `aria-live="polite"` anunciant
  "Desant idea, recuperant previsualització…" en enviar el formulari i
  la resolució final (èxit/parcial/fallada) en arribar per sondeig;
  navegació completa per teclat per afegir (enganxar+confirmar) i
  eliminar cada targeta sense ratolí (AC-09); text alternatiu no buit
  o `alt=""` justificat i títol/descripció/enllaç anunciats de manera
  comprensible per lector de pantalla (AC-10). · *covers:* AC-01,
  AC-02, AC-03, AC-09, AC-10, AC-11 (base contracte de dades)

### Verification

- [x] **T-20** — Afegir tests unitaris a `internal/ideas`/`fetchsafe`
  per a la validació de format i esquema d'URL (AC-08/EC-01/EC-10):
  URL buida o només espais rebutjada (EC-10); esquemes `file://`,
  `javascript:`, `ftp://`, `data:` rebutjats sense cap petició de
  xarxa (EC-01/NFR-05). · *covers:* AC-08, EC-01, EC-10, NFR-05
- [x] **T-21** — Afegir test unitari de parsing Open Graph (AC-01/
  AC-03/EC-05): HTML amb totes les etiquetes OG (èxit complet), HTML
  amb només algunes etiquetes (parcial, EC-05 camps absents), HTML
  sense cap etiqueta OG reconeguda o malformat (EC-05, tractat com a
  "no trobat", no com a crash de parsing). · *covers:* AC-01, AC-03,
  EC-05
- [ ] **T-22** — Afegir tests d'integració contra un servidor HTTP de
  test controlat (mock local, mai internet real — imprescindible per a
  tots els casos SSRF) per a EC-03/EC-08/EC-09: resposta de mida molt
  superior al límit de 2 MiB (EC-03, assert que la descàrrega s'atura i
  la idea es desa en fallback, sense exhaurir memòria); latència
  superior al timeout de 5s (EC-08, assert fallback sense espera
  indefinida a la UI); `Content-Type` no HTML — PDF/imatge/vídeo
  (EC-09, assert que no s'intenta parsing OG). · *covers:* AC-02,
  EC-03, EC-08, EC-09, NFR-07
- [ ] **T-23** — Afegir test d'integració per a AC-01/AC-04/AC-06 amb
  el servidor de test simulant OG complet: `POST` amb enllaç vàlid →
  `GET` posterior mostra la idea com a targeta amb títol/imatge/
  descripció, autoria visible, i persisteix després de "recarregar"
  (segona consulta `GET`). · *covers:* AC-01, AC-04, AC-08 (base
  contrast amb NIU-1/NIU-5)
- [ ] **T-24** — Afegir test d'integració per a AC-02/AC-03 amb el
  servidor de test simulant un bloqueig d'accés (403), un timeout, i
  metadades OG parcials: assert que la idea es desa igualment en tots
  tres casos (mai error bloquejant), amb `preview_status` `failed` o
  `partial` segons el cas, i que el `POST` original no ha esperat el
  resultat del scraping (ADR-03, resposta `201` immediata). · *covers:*
  AC-02, AC-03
- [ ] **T-25** — Afegir test d'integració per a AC-05/AC-09/EC-15/EC-16
  (`ideas_test.go`, mateix patró `newTestServer`/`doJSON` establert):
  `DELETE` elimina la idea de la llista i no reapareix en un `GET`
  posterior; doble `DELETE` sobre el mateix `id` idempotent sense 5xx
  (EC-15); doble `POST` del mateix formulari (doble clic) crea dues
  idees independents sense 5xx ni estat corrupte (EC-16, documentat
  explícitament com a **no** idempotent, a diferència d'EC-15). ·
  *covers:* AC-05, AC-09, EC-15, EC-16
- [ ] **T-26** — Afegir test d'integració amb dos clients simulats per
  a AC-06/EC-06/EC-17: un client afegeix i un altre elimina una idea,
  el segon client veu la llista actualitzada (incloent-hi targetes amb
  previsualització i idees en fallback) en el següent `GET` (AC-06);
  el mateix enllaç desat dues vegades no es bloqueja i genera dues
  entrades independents (EC-06); llista buida en primer ús mostra un
  estat visual clar sense error (EC-17, E2E complementari). · *covers:*
  AC-06, EC-06, EC-17
- [ ] **T-27** — Afegir tests unitaris/integració de seguretat
  reutilitzant exactament els patrons ja escrits per NIU-1/NIU-4/NIU-5
  (`security_test.go`, `sql_static_test.go`) aplicats a
  `/api/v1/ideas`: enllaç o metadada recuperada amb marcatge HTML/script
  (`<img src=x onerror=alert(1)>`) desat literalment i, en navegador
  real (Playwright), assert de no-execució de script (EC-11/NFR-01);
  enllaç o metadada amb `'; DROP TABLE ideas;--` desat literalment amb
  la resta de dades intactes (EC-12/NFR-02); assaig estàtic de la
  taula de rutes `chi` confirmant que cap ruta `GET` sota
  `/api/v1/ideas` té efecte mutador (EC-13/NFR-03); petició sense
  cookie de sessió vàlida contra qualsevol endpoint d'aquest espai
  rebutjada com a no autenticada (EC-14/NFR-04); inspecció de
  capçaleres de la petició sortint del servidor de test confirmant 0
  capçaleres d'autenticació de Niu presents (NFR-08). · *covers:*
  AC-10, EC-11, EC-12, EC-13, EC-14, NFR-01, NFR-02, NFR-03, NFR-04,
  NFR-08
- [ ] **T-27a** — Afegir test d'integració dedicat per a EC-02/EC-07
  (destí prohibit per IP literal o per resolució DNS): URL amb IP
  literal `127.0.0.1`/`10.x.x.x`/`169.254.x.x` rebutjada sense
  completar la petició (EC-02); domini que **resol** (via un
  doble/mock de resolució DNS controlat pel test) a una IP de xarxa
  privada o loopback rebutjat de la mateixa manera, encara que el text
  de la URL no sigui una IP literal (EC-07, DNS rebinding). Assert
  explícit de **zero connexió TCP establerta** cap al destí en tots dos
  casos, no només un error de resposta. · *covers:* EC-02, EC-07,
  NFR-06
- [ ] **T-27b** — Afegir test d'integració dedicat per a la denylist de
  noms d'amfitrió (T-03b, F-03/F-04): URL apuntant a `niu.fikua.com` i
  a un nom configurat via `NIU_PUBLIC_HOST` de test, ambdues
  rebutjades **abans** de qualsevol resolució DNS (assert 0 peticions
  DNS/TCP sortints), diferenciant aquest mecanisme del de validació
  d'IP (T-03c) perquè una implementació futura no el pugui ballar per
  descuit. · *covers:* NFR-06
- [ ] **T-27c** — **[Regressió F-01 — obligatori, no genèric] Afegir
  test d'integració de redirecció al MATEIX host cap a un destí
  prohibit (EC-04).** Muntar un servidor de test que respongui `200` a
  la primera validació i, en un segon salt, respongui des del **mateix
  `host:port`** que la petició original però simulant/apuntant una IP
  de xarxa prohibida per a aquest salt concret; assert que la petició
  final es rebutja i que `ControlContext` s'invoca a **cada** salt (no
  només al primer) — aquest és l'únic test que detecta una regressió
  del bypass de connexió-reutilitzada que `security-engineer` va trobar
  explotable (F-01) si `DisableKeepAlives` (T-03d) es desactivés o
  s'oblidés en un canvi futur. **No confondre amb, ni substituir per,**
  un test de redirecció cross-host genèric — aquest últim NO detecta
  F-01. · *covers:* EC-04, NFR-06
- [ ] **T-27d** — **[Regressió F-02 — obligatori, no genèric] Afegir
  test d'integració amb destí en forma IPv4-mapejada-a-IPv6
  (`::ffff:127.0.0.1` i/o `::ffff:169.254.169.254`).** Petició directa
  (o via redirecció) cap a una d'aquestes adreces; assert que es
  rebutja igual que la forma IPv4 literal equivalent — aquest és
  l'únic test que detecta una regressió d'`Unmap()` (T-03c) omès o
  eliminat per accident en un canvi futur, ja que `IsLoopback()`/
  `IsPrivate()`/`IsLinkLocalUnicast()` retornen `false` sobre la forma
  mapejada sense `Unmap()` previ. **No confondre amb, ni substituir
  per,** el test genèric d'IP literal `127.0.0.1` d'T-27a — aquest
  últim NO detecta F-02. · *covers:* EC-04, NFR-06
- [ ] **T-27e** — Afegir test d'integració per a NFR-09 (cache
  at-save-time): desar una idea contra el servidor de test, fer
  diverses lectures consecutives de `GET /api/v1/ideas`, assert que el
  comptador de peticions sortints del servidor de test roman en **1**
  independentment de quants `GET` es facin a la llista. · *covers:*
  NFR-09
- [ ] **T-28** — Afegir test E2E (Playwright) per a AC-07/AC-09/AC-10/
  AC-11 (`tests/e2e/specs/ideas.spec.js`): comparació visual amb NIU-1/
  NIU-5 confirmant diferenciació clara (AC-07); navegació completa per
  Tab/Enter per afegir un enllaç i eliminar una targeta sense ratolí
  (AC-09); assert de text alternatiu no buit i contingut anunciat de
  manera comprensible per lector de pantalla en una targeta amb i
  sense previsualització (AC-10); viewport mòbil 375×667 mantenint
  totes les funcionalitats (EC-18). Documentar `overview.md` com a
  revisió manual d'AC-11 (revisió documental, no automatitzable). ·
  *covers:* AC-07, AC-09, AC-10, EC-18
- [ ] **T-29** — Actualitzar `docs/overview.md` per esmentar l'espai
  d'idees d'activitats amb previsualització de link com a funcionalitat
  existent de Niu (llista simple sense estat, fallback no bloquejant,
  sense deduplicació d'enllaços), mantenint-lo com a font única de
  veritat del que fa l'app. · *covers:* AC-11
- [ ] **T-30** — Executar `commands.test` (`cd app && go test ./...`),
  `commands.lint` (`gofmt -l`) i `commands.typecheck` (`cd app && go
  vet ./...`) del manifest de punta a punta; confirmar explícitament
  que els tres tests de regressió SSRF (T-27c, T-27d) i el test de
  denylist de noms (T-27b) estan verds, sense regressions sobre la
  suite existent d'NIU-1/NIU-4/NIU-5. **Bloquejant per a `/audit`**
  (design.md §7, NFR-06 és el punt de bloqueig d'aquest ítem). ·
  *covers:* verificació final de totes les AC/EC/NFR (tasca de
  tancament tècnic, no substitueix cap cobertura individual anterior)

### Closing (universal — all changes)

- [ ] **C-01** — Append changelog entry (`docs.changelog` from manifest)
- [ ] **C-02** — Transition backlog item to `Human Review` via the adapter
- [ ] **C-03** — Propose semver bump (ASK USER — never apply unattended)

## 2. AC ↔ tasks traceability matrix

| AC | Statement (short) | Covering tasks |
| --- | --- | --- |
| AC-01 | Afegir una idea amb previsualització completa | T-03, T-04, T-05, T-06, T-11, T-12, T-14, T-16, T-19, T-23, T-30 |
| AC-02 | Afegir una idea quan la previsualització falla | T-03, T-04, T-06, T-07, T-16, T-19, T-22, T-24, T-30 |
| AC-03 | Previsualització amb dades parcials | T-06, T-07, T-12, T-16, T-19, T-21, T-24, T-30 |
| AC-04 | Cada idea mostra qui l'ha afegit | T-10, T-11, T-15, T-19, T-23, T-30 |
| AC-05 | Eliminar una idea desada | T-09, T-13, T-17, T-19, T-25, T-30 |
| AC-06 | Dos usuaris veuen les mateixes idees | T-08, T-17, T-20, T-26, T-30 |
| AC-07 | Espai visualment diferenciat | T-02, T-18, T-19, T-23, T-28, T-30 |
| AC-08 | Enllaç obligatori i amb format vàlid | T-05, T-20, T-30 |
| AC-09 | Navegació completa per teclat | T-17, T-19, T-25, T-28, T-30 |
| AC-10 | Targeta accessible per lectors de pantalla | T-17, T-19, T-27, T-28, T-30 |
| AC-11 | `overview.md` reflecteix el nou espai | T-16, T-19, T-28, T-29, T-30 |

## 3. Edge cases ↔ tasks

| EC | Statement (short) | Covering tasks |
| --- | --- | --- |
| EC-01 | Esquema d'URL no `http(s)` | T-03a, T-05, T-20 |
| EC-02 | Destí xarxa privada/loopback/pròpia instància | T-03b, T-03c, T-27a, T-30 |
| EC-03 | Resposta extremadament gran | T-03g, T-22, T-30 |
| EC-04 | Cadena de redireccions (incloent-hi els dos casos de regressió) | T-03e, T-27c, T-27d, T-30 |
| EC-05 | Metadades Open Graph absents o malformades | T-04, T-21 |
| EC-06 | Idea duplicada (mateix enllaç dues vegades) | T-26 |
| EC-07 | URL que resol a xarxa privada via DNS | T-03c, T-27a, T-30 |
| EC-08 | Timeout de resposta | T-03f, T-22, T-30 |
| EC-09 | Contingut no HTML | T-04, T-22 |
| EC-10 | Enllaç buit o només espais | T-05, T-20 |
| EC-11 | Injecció HTML/JS al títol o descripció | T-17, T-27 |
| EC-12 | Injecció SQL a l'enllaç o metadades | T-01, T-27 |
| EC-13 | Intent de mutació via `GET` | T-13, T-27 |
| EC-14 | Accés sense sessió autenticada | T-13, T-27 |
| EC-15 | Eliminar una idea ja eliminada (idempotent) | T-09, T-25 |
| EC-16 | Reenviament del mateix formulari (doble clic) | T-25 |
| EC-17 | Llista buida en primer ús | T-17, T-26 |
| EC-18 | Viewport mòbil | T-28 |

## 4. NFRs ↔ tasks

| NFR | Statement (short) | Covering tasks |
| --- | --- | --- |
| NFR-01 | Cap dada externa/d'usuari es renderitza com a HTML (XSS) | T-17, T-27 |
| NFR-02 | Cap valor concatenat a SQL (injecció) | T-01, T-27 |
| NFR-03 | Cap mutació via `GET` | T-13, T-27 |
| NFR-04 | Tots els endpoints requereixen sessió vàlida | T-13, T-27 |
| NFR-05 | SSRF — rebuig d'esquema no `http(s)` | T-03a, T-20 |
| NFR-06 | SSRF — destins prohibits (denylist + IP allowlist + redireccions) | T-03b, T-03c, T-03d, T-03e, T-27a, T-27b, T-27c, T-27d, T-30 |
| NFR-07 | SSRF — límits de recurs (timeout + mida + concurrència) | T-03f, T-03g, T-07a, T-22 |
| NFR-08 | Cap credencial de Niu surt cap a destí extern | T-03h, T-27 |
| NFR-09 | Cache at-save-time — mai re-scraping en `GET` | T-06, T-10, T-27e |

## 5. Out of scope (mirrored from design)

> Fora d'abast d'aquest ítem — cap tasca de la llista anterior ha
> d'incloure res d'això.

- Qualsevol cicle de vida o estat (idea / planificada / feta) en v1 —
  llista simple de desar/eliminar únicament (§0.1 de `requirements.md`).
- Edició manual del títol, la imatge o la descripció recuperats.
- Actualitzar la previsualització si el contingut de la pàgina
  enllaçada canvia després de desar-la.
- Reintent automàtic o cua de reprocessament quan la previsualització
  falla en el moment de desar — el fallback és permanent fins que
  s'esborri i es torni a afegir (R-02, acceptat explícitament).
- Integració amb l'API oficial d'Instagram o de cap altra xarxa social
  per esquivar bloquejos de scraping.
- Cerca, filtres, o categorització de les idees desades més enllà d'una
  llista simple.
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos.
- Gamificació (ratxes, punts) sobre aquest espai.
- Unicitat o deduplicació d'enllaços (EC-06, confirmat).
- Qualsevol relació tècnica o de model de dades amb `internal/items`
  (NIU-1) o `internal/projects` (NIU-5) — domini completament
  independent (ADR-01).
- WebSocket o SSE per notificar la resolució del scraping — es
  descobreix via el mateix sondeig/refetch-on-focus ja existent
  (ADR-03).
- Cua de tasques persistent (`scrape_jobs` en taula) per al worker pool
  — un pool en memòria n'hi ha prou (ADR-03, alternativa rebutjada).
- Proxy de sortida dedicat o servei extern de "URL unfurling" (ADR-02,
  alternativa rebutjada).
- Tractar tota resolució via el resolver DNS embegut de Docker com a
  sospitosa per defecte — es fa servir denylist de noms explícita
  (ADR-02, F-03/F-04).

## 6. Notes for the developer

- **Ordre estricte de dependència:** T-01/T-02 (migració + domini)
  abans de tocar `fetchsafe` (T-03...) i el servei (T-05...);
  `fetchsafe` complet (T-03 a T-03h) abans de cablejar-lo des
  d'`ideas.Service` (T-05/T-06/T-07); domini/repositori (T-05–T-10)
  abans dels handlers HTTP (T-11–T-14); backend abans del wiring
  frontend (T-15...). No saltar fases — en particular, **no
  implementar cap crida HTTP sortint fora de `fetchsafe`**.
- **`internal/items`, `internal/projects`, `internal/auth`,
  `internal/config` no es toquen.** `items_handlers.go`,
  `projects_handlers.go`, `auth_handlers.go`, `csrf.go` no es toquen.
  Només `router.go` rep un canvi quirúrgic (registrar el nou grup de
  rutes) i `main.go` rep el wiring nou (T-13/T-14).
- **`internal/fetchsafe` és el component d'auditoria prioritària.**
  `code-reviewer`/`security-engineer` a `/audit` han de confirmar,
  independentment els uns dels altres, que cada mecanisme de T-03a a
  T-03h existeix literalment al codi (no com a comentari o intenció) —
  vegeu R-01 a `design.md` §8: cap altre punt del codi pot fer una
  petició HTTP cap a una URL d'usuari.
- **NFR-06 és blocant per a `/audit`** (design.md §7/§9): sense els
  tests T-27a, T-27b, T-27c i T-27d verds contra el servidor de test
  simulat, cap implementació d'aquest ítem es pot considerar conforme,
  independentment de com de correcte sembli el codi a simple vista.
- **T-27c i T-27d no són "més tests d'EC-04" genèrics** — són les
  regressions específiques dels findings F-01 (connexió reutilitzada
  saltant-se `ControlContext` en redirecció al mateix host) i F-02
  (`Unmap()` omès sobre adreces `::ffff:`-mapejades) que
  `security-engineer` va trobar explotables contra el disseny original.
  Un test de redirecció cross-host per si sol **no** detecta cap dels
  dos.
- **T-07a (valor del cap de concurrència) no bloqueja `/audit`** però
  ha d'estar resolt i documentat al codi abans de tancar l'ítem —
  `design.md` §9 ho marca explícitament com a decisió delegada a
  `/code`, no com un forat obert indefinidament.
- **Tot test contra un "servidor extern" ha d'apuntar a un doble/mock
  HTTP local controlat pel propi entorn de test, mai a internet real
  ni a xarxa privada real del CI** — imprescindible per a T-22, T-27a,
  T-27b, T-27c, T-27d, T-27e.
- **Reutilització de patrons de test:** `newTestServer`/`doJSON` de
  `tests/integration/helpers_test.go`, i els patrons ja escrits a
  `security_test.go`/`sql_static_test.go` es reutilitzen sense
  modificació per a T-27 — no inventar un altre harness per a la part
  no-SSRF. Els casos SSRF (T-22, T-27a–T-27e) sí que requereixen un
  servidor HTTP de test nou i controlable (respostes lentes, grans,
  redireccions configurables, tipus de contingut variables, captura de
  capçaleres) — construir-lo un sol cop i reutilitzar-lo entre totes
  aquestes tasques, no duplicar-lo per cas.
- **Migració número:** la següent disponible després de la migració
  `projects` d'NIU-5. Sense seed de dades — taula nova sense files
  prèvies.
- **Comandes locals:** `cd app && go test ./...`, `gofmt -l .` (accepta
  un path final), `cd app && go vet ./...`, `cd app && go build
  ./...`; Playwright: `cd app/tests/e2e && npx playwright test`
  (T-28 s'hi afegeix).
