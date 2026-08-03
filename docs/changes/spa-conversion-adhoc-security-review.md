# Revisió de seguretat ad-hoc — SPA fallback (`niu-spa-conversion`)

- **Branca:** `niu-spa-conversion` (5 commits per davant de `main`, sense fusionar)
- **Data:** 2026-08-03
- **Abast:** revisió focalitzada, **no** un escombrat OWASP complet de l'aplicació
- **Fitxer principal:** `app/internal/httpapi/router.go` (`spaFallback` / `serveIndex`)
- **Rol:** security-engineer (read-only — cap edició de codi)
- **Veredicte:** **NET** — cap troballa *blocking* ni *major*

---

## 1. Resum executiu

El canvi afegeix un *fallback* SPA al costat servidor: qualsevol GET/HEAD amb
un camí que no resol a un fitxer real dins `webFS` rep el contingut d'
`index.html` en lloc d'un 404.

He verificat empíricament els cinc punts sol·licitats muntant el router real
amb `httptest` (fakes autenticat i **no** autenticat) i comparant el
comportament contra `main` mitjançant un `git worktree`. Els fitxers de sonda
s'han eliminat en acabar; l'arbre de treball queda net.

**Conclusió:** el *fallback* està correctament dissenyat. No introdueix cap
bypass d'autenticació, cap *path traversal*, i no altera capçaleres de
seguretat ni el patró de peticions rellevant per al *rate limiting*. Les dues
observacions registrades són **minor**/**nit** i cap bloqueja la fusió.

| Punt revisat | Resultat |
|---|---|
| 1. Bypass d'auth via fallback | Net — sense canvi de postura |
| 2. Path traversal / confusió d'assets | Net — sandbox d'`embed.FS` intacte |
| 3. Capçaleres CSP i de seguretat | Net — middleware més extern, s'apliquen |
| 4. Gestió de mètodes | Net — correcte; API 404/401 no s'empassen |
| 5. Interacció amb el rate limiting | Millora — redueix peticions |

---

## 2. Model d'amenaces del diff

**Fronteres de confiança travessades:** una de sola — el límit
públic/no-autenticat davant del servidor de fitxers estàtics encastat. El
canvi **no** toca cap codi d'autenticació, autorització, CSRF, sessions ni
accés a dades. `WithCurrentUser`, `RequireCSRF` i el rate limiter d'auth
queden literalment intactes al diff.

**Classificació de dades tocada:** cap. El *fallback* només serveix bytes
estàtics d'`index.html` — el mateix fitxer que ja era públic.

**Transicions d'estat afegides:** cap al servidor. El router client
(`history.pushState`) és purament de presentació.

---

## 3. Troballes per punt

### Punt 1 — Bypass d'autenticació via el fallback (A01, A07) — **NET**

**Ordre del middleware.** `r.NotFound(spaFallback(...))` es registra a
l'arrel del router. Les cadenes `r.Use(...)` (SecurityHeaders, LimitBody,
Recoverer, Logger) **sí** l'envolten; `WithCurrentUser` **no**, perquè està
muntat només dins de `r.Route("/api/v1", ...)`. Això és correcte i
intencionat: el shell estàtic mai no ha estat protegit per auth al servidor.

**Comparació verificada contra `main`:**

| Camí | `main` | Branca | Delta |
|---|---|---|---|
| `GET /` | **200**, 4841 B (shell complet) | 200, 7375 B (shell fusionat) | ja era públic |
| `GET /projects.html` | **200**, 3420 B | 200 (shell) | ja era públic |
| `GET /projects` | 404 | 200 (shell) | **nou** |
| `GET /nonexistent` | 404 | 200 (shell) | **nou** |

El punt clau: **`GET /` ja servia el shell sencer a usuaris no autenticats a
`main`.** El model d'auth d'aquesta aplicació és *client-side gating* — el
shell es carrega sempre, i `js/main.js` crida `GET /api/v1/me`; en rebre un
401, `api.js` redirigeix a `/login.html?next=...`. Les dades reals viuen
exclusivament rere `/api/v1/*`, que **sí** està protegit per
`WithCurrentUser`.

Per tant el *fallback* **no canvia la postura d'auth**: amplia el conjunt de
camins que arriben a un shell que ja era públic, sense exposar cap superfície
nova.

**Què conté el shell quan no estàs autenticat?** Revisat `web/index.html`
(154 línies): és HTML 100 % estàtic. Conté l'estructura de navegació
("🛒 Compra", "🏠 Projectes"), etiquetes de secció i contenidors buits
(`<ul id="list-shopping">`, `<ul id="projects-list">`). **No** conté cap
token CSRF, ni identificador de sessió, ni dades d'usuari, ni cap secret
(verificat per grep). Els noms i avatars s'omplen només via JS després d'un
`/api/v1/me` reeixit.

L'única fuita és **estructural**: un anònim aprèn que Niu té una secció de
compra i una de projectes. Això ja era observable a `main` via `GET /` i
`GET /projects.html`, i és informació de disseny d'una app domèstica per a
dues persones — no és un secret. Cap acció requerida.

> **Nota (fora d'abast, preexistent).** `web/js/auth.js` fa
> `window.location.href = params.get('next') || '/'` sense validar que
> `next` sigui una ruta relativa (CWE-601, *Open Redirect*). Un valor com
> `//evil.example` produiria una redirecció externa després del login.
> **No l'introdueix aquesta branca** (`git diff main...HEAD -- app/web/js/auth.js`
> és buit; prové de NIU-4). L'impacte és baix (cal enganyar l'usuari perquè
> obri una URL de login manipulada), però val la pena capturar-ho com a ítem
> de backlog independent. Mitigació habitual: rebutjar qualsevol `next` que
> no comenci per `/` seguit d'un caràcter diferent de `/` o `\`.

### Punt 2 — Path traversal / confusió d'assets (A01, A03, CWE-22) — **NET**

La lògica de camins és:

```go
cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
if cleanPath == "" || cleanPath == "." {
    cleanPath = "index.html"
}
if _, err := fs.Stat(webFS, cleanPath); err != nil { serveIndex(...); return }
fileServer.ServeHTTP(w, r)
```

Tres capes independents fan que això sigui segur:

1. **`net/http` neteja abans d'arribar-hi.** El `ServeMux`/servidor
   normalitza i emet un 301 per als camins amb `..` no normalitzats
   (verificat: `GET /../index.html` → 301, `GET /js/../index.html` → 301).
2. **`path.Clean`** resol qualsevol `.`/`..` restant abans del `Stat`.
3. **`fs.Stat` sobre un `fs.FS`** rebutja per construcció qualsevol camí no
   *slash-separated*, absolut o amb elements `..` (regla `fs.ValidPath`).
   A més, `main.go` fa `fs.Sub(niu.WebFS, "web")`, així que l'arrel ja està
   confinada al subarbre `web/`.

Punt important de disseny: el `Stat` es fa **només per decidir la
ramificació**. El servei real el fa `fileServer` amb la `r.URL.Path`
**original** (no la manipulada), i la branca de *fallback* obre una constant
literal `"index.html"` — mai un camí derivat de l'entrada de l'usuari. Per
tant no hi ha cap camí de codi on una cadena controlada per l'atacant
arribi a un `Open()`.

**Verificat empíricament** — cap variant escapa del sandbox:

| Petició | Resultat |
|---|---|
| `/../../../../etc/passwd` | 200 shell (no el fitxer) |
| `/%2e%2e/%2e%2e/etc/passwd` | 200 shell |
| `/js/../index.html` | 301 (normalització) |
| `//login.html` | 200 `login.html` (correcte) |

*Confusió d'assets:* un fitxer que existeix se serveix sempre com abans; un
que no existeix cau al shell amb `Content-Type: text/html`. Com que
`X-Content-Type-Options: nosniff` està actiu, un `GET /app.js` inexistent
retorna HTML que el navegador **no** interpretarà com a JavaScript — falla
net, sense risc d'execució.

### Punt 3 — Capçaleres CSP i de seguretat (A05) — **NET**

`SecurityHeaders` és el **primer** `r.Use()` del router arrel, i per tant
embolcalla tot, inclòs el handler `NotFound`. No hi ha capçaleres per
handler que el *fallback* pugui saltar-se.

Verificat que la resposta de *fallback* porta el joc complet, idèntic al
d'una ruta normal:

```
GET /projects  -> 200
  Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self';
                           font-src 'self'; img-src 'self'; connect-src 'self';
                           object-src 'none'; base-uri 'none'
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  Referrer-Policy: no-referrer
  Strict-Transport-Security: max-age=63072000; includeSubDomains
```

Idèntic per a `/`, `/app.css` i `/login.html`. La CSP és estricta (sense
`unsafe-inline`), cosa que encaixa amb el shell: tot el JS és en mòduls
externs, no hi ha scripts en línia.

`serveIndex` fa `w.Header().Set("Content-Type", ...)` abans de
`http.ServeContent`, cosa que només afegeix una capçalera pròpia i no
n'esborra cap de les del middleware. Correcte.

### Punt 4 — Gestió de mètodes (A05) — **NET**

El *fallback* es restringeix correctament: mètodes diferents de GET/HEAD es
deleguen a `fileServer`, que respon 404 per a camins inexistents.

**La preocupació concreta de l'informe — que `/api/v1/itms` s'empassi un 404
i retorni HTML — no es materialitza.** El motiu és que chi munta
`r.Route("/api/v1", ...)` amb un *wildcard* de subarbre, de manera que
qualsevol camí sota `/api/v1/` és capturat pel subrouter (i pel seu
`WithCurrentUser`) **abans** d'arribar al `NotFound` de l'arrel:

| Petició | Resultat |
|---|---|
| `GET /api/v1/itms` (no auth) | **401 JSON** `{"error":{"code":"unauthenticated"...}}` |
| `POST/PUT/DELETE/PATCH /api/v1/itms` | **401 JSON** |
| `PUT /api/v1/items/` (ruta existent, mètode incorrecte) | **405** |
| `POST/PUT/DELETE/PATCH/OPTIONS /projects` | **404** `text/plain` |
| `HEAD /projects` | 200, cos buit, `text/html` (correcte per HEAD) |

Els camins amb forma d'API mai no reben HTML. Amb un client autenticat,
`/api/v1/itms` retorna un 404 JSON net del subrouter. La separació és neta.

> **Nit.** Amb sessió activa, `GET /api/v1/itms` retorna el 404 del
> subrouter, però `GET /api/v1` (sense barra final) també cau al mateix
> tractament. No és un problema de seguretat, només consistència d'API.

### Punt 5 — Interacció amb el rate limiting (A05 / disponibilitat) — **MILLORA**

Context: l'incident real de 429 es va produir perquè NIU-5 va afegir una
segona pàgina amb *polling*, i navegar entre `index.html` i `projects.html`
era una **navegació de pàgina completa** que retornava a demanar
`manifest.json` + `/api/v1/me` + la llista sencera cada cop.

Aquest canvi **redueix** la pressió sobre el rate limiter, no l'augmenta:

- Navegar entre "Compra" i "Projectes" ara és un *toggle* del DOM
  (`history.pushState` + alternar `hidden`) — **zero peticions de xarxa**,
  on abans n'hi havia una tanda sencera.
- La identitat es resol **una sola vegada** per càrrega (`GET /api/v1/me`),
  no una per pàgina.
- El *fallback* **no genera cap petició nova**. Cada navegació directa a
  `/projects` és exactament una petició, com abans ho era `/projects.html`.

**Sobre "cerques d'assets fallides durant el bootstrap":** revisat
`index.html` i `main.js` — tots els `modulepreload` i l'`import()` dinàmic
de `projects-view.js` apunten a fitxers que **existeixen** a `webFS`. Cap es
resol via el *fallback*, així que no s'introdueix cap petició addicional que
compti contra el límit.

L'únic efecte teòric contrari: com que camins inexistents ara retornen 200
en lloc de 404, un escaneig automatitzat consumiria quota igual que abans
(el rate limiter de Traefik compta peticions, no codis d'estat) però rebria
respostes de 7 KB en lloc de 19 B. Per a una app domèstica rere Cloudflare
amb `sourcecriterion` per `Cf-Connecting-Ip`, això és irrellevant.

> **Observació de procés (no és una troballa de codi).** La pujada del rate
> limit a `app/compose.yaml` (`average` 10 → 60, `burst` 20 → 100) està
> **només a l'arbre de treball, sense commit** — no forma part dels 5
> commits de la branca (`git diff main...HEAD -- app/compose.yaml` és buit).
> Convé fer-ne commit conscientment amb la resta del canvi; altrament la
> mitigació de l'incident es podria perdre en un `checkout` i el 429
> reapareixeria en desplegar. Val la pena notar que la conversió a SPA per
> si sola ja redueix molt les peticions, així que potser el límit no cal
> pujar-lo tant com 60/100 — es podria reavaluar amb dades reals.

---

## 4. Cobertura de proves — **minor**

`spaFallback` i `serveIndex` **no tenen cap prova** (`router_test.go` i les
proves d'integració no cobreixen el `NotFound`). Comportaments que ara
depenen de codi sense xarxa de seguretat:

- els camins `/api/v1/*` inexistents han de retornar JSON, mai HTML;
- POST/PUT/DELETE a camins inexistents han de fer 404, no servir el shell;
- els assets existents no han de caure mai al *fallback*.

Cap d'aquests és exercit avui. Són exactament les invariants que una
refactorització futura del router podria trencar en silenci. Recomanació
(no bloquejant): afegir un test de taula a `router_test.go` — les sondes
que he fet servir són trivials de convertir-hi.

*(Sense relació amb aquesta branca: `GET /js/`, `/assets/` i `/fonts/`
retornen llistats de directori del `http.FileServer`. És comportament
preexistent a `main` i només exposa noms de fitxers estàtics ja públics —
ho anoto per completesa, no com a troballa d'aquest diff.)*

---

## 5. Resum de troballes

| ID | Severitat | Punt | Descripció | Referència |
|---|---|---|---|---|
| S-01 | *minor* | Proves | `spaFallback`/`serveIndex` sense cobertura; les invariants API-vs-shell no estan blindades | OWASP Code Review Top 10 (C1) |
| S-02 | *nit* | Procés | Pujada del rate limit a `compose.yaml` sense commit; risc de perdre la mitigació de l'incident 429 | A05 |
| — | *informatiu* | Preexistent | Open redirect no validat via `next` a `auth.js` (de NIU-4, fora d'aquest diff) | CWE-601 |

**Cap troballa *blocking* ni *major*.**

---

## 6. Veredicte

**APROVAT** des de la perspectiva de seguretat.

El *fallback* SPA està implementat amb cura: delega el servei real al
`fileServer` amb el camí original, obre una constant literal a la branca de
*fallback*, es restringeix a GET/HEAD, i queda per sota del middleware de
capçaleres de seguretat. La preocupació més plausible — que camins amb forma
d'API s'empassessin els 404 i retornessin HTML — no es dona, perquè el
subrouter `/api/v1` captura aquest espai de noms abans del `NotFound`.

Les dues accions recomanades (proves per a `spaFallback`, commit del canvi de
`compose.yaml`) són de seguiment i no han de bloquejar la fusió.

---

### Verificació feta

- Lectura completa de `router.go`, `middleware.go`, `index.html`, `main.js`,
  `auth.js`, `api.js`, `web.go`, `cmd/niu/main.go`.
- `git diff main...HEAD` complet del canvi.
- Sondes `httptest` amb autenticador fals **autenticat** i **no autenticat**,
  sobre `fstest.MapFS` i sobre l'arbre `web/` real.
- Comparació de comportament contra `main` via `git worktree`.
- `go vet ./...` net; `go test ./...` tot en verd.
- Els fitxers de sonda temporals s'han eliminat; l'arbre de treball queda amb
  l'única modificació preexistent de `compose.yaml`.
