---
key: NIU-11
artefact: security-review
agent: security-engineer
verdict: APPROVED
blocking_findings: 0
date: 2026-08-07
---

# NIU-11 — Auditoria de seguretat

> Abast: diff no comitejat sobre `main` + fitxers nous no rastrejats.
> Estàndards: OWASP Top 10 (2021), OWASP Code Review Top 10 (2017), CWE.
> Read-only: cap fitxer de codi ha estat modificat.

## Veredicte

**APPROVED — 0 findings blocking, 0 major.**

El canvi és net. És una rèplica disciplinada del patró de NIU-6, no una
reinvenció: no afegeix cap client HTTP nou, cap camí d'egress nou, cap
esquema de validació duplicat a mà. Les tres superfícies que realment
podien trencar-se (SSRF, XSS via `href`, exposició de `description`)
estan cobertes per control de codi, no per accident ni només per la CSP.

He intentat explotar-lo activament (fuzzing de `ValidateURL` amb 18
payloads d'injecció d'esquema, traça completa input → DB → JSON → DOM,
anàlisi del sostre del pool). No he trobat cap camí d'atac funcional.
Només 3 observacions `minor`/`nit`, cap de les quals bloqueja.

---

## 1. Model d'amenaces del diff

| Element | Avaluació |
| --- | --- |
| **Frontera de confiança nova** | `POST /api/v1/projects` ara accepta una URL controlada per l'usuari que provoca una petició HTTP sortint del servidor. És el segon productor d'SSRF de l'aplicació. |
| **Dades de tercers noves** | `og:image` / `og:title` provinents d'una pàgina externa arbitrària, persistides i renderitzades al DOM. |
| **Classificació de dades** | Cap PII, cap credencial, cap dada de pagament. `description` es persisteix però no s'exposa. |
| **Transicions d'estat noves** | `preview_status` NULL → `pending` → `ready`/`partial`/`failed`. Cap sessió, token ni cua nova. |
| **Autenticació** | Cap superfície d'auth nova. La ruta hereta `WithCurrentUser` + `RequireCSRF`. |

L'atacant rellevant no és un anònim d'Internet: `POST /projects` exigeix
sessió i token CSRF (`internal/httpapi/router.go:128`), i el middleware
`WithCurrentUser` (`middleware.go:58-76`) retorna 401 abans que el
handler s'executi. El model d'amenaça real és per tant **(a)** un dels
dos usuaris legítims apuntant a una destinació interna, i **(b)** un
lloc de tercers hostil que serveix metadades Open Graph malicioses.

---

## 2. SSRF — el segon punt d'entrada (A10:2021, CWE-918)

**Verificat: sense bypass. `fetchsafe.FetchPreview` és l'única sortida.**

Verificacions concretes:

1. **Un sol client per procés.** `grep` de `fetchsafe.NewClient()` a tot
   el codi no de test retorna **exactament una** ocurrència
   (`cmd/niu/main.go:107`). `projectsFetch` (`main.go:124-132`) tanca
   sobre el **mateix** `fetchsafeClient` que `ideasFetch` — no en
   construeix un de segon. Compleix T-03h de NIU-6.

2. **Cap client HTTP paral·lel.** Els únics fitxers no de test que
   importen `net/http` sota `internal/` són `auth/`, `httpapi/` (servidor
   entrant) i `fetchsafe/`. `internal/projects` **no** importa
   `net/http` — la dependència passa pel seam `PreviewFetcher`
   (`projects.go:77-83`), igual que a `ideas`. La invariant de disseny
   "cap altre paquet fa fetch d'una URL d'usuari" es manté.

3. **Cap ruta que se salti la validació d'esquema.** `validateProjectURL`
   (`service.go:152-172`) delega a `ideas.ValidateURL` en lloc de
   reimplementar la regla http(s). Això és exactament el que demanava
   T-04 i és el detall que evita la classe d'error més típica en un
   segon caller: una validació bessona que divergeix amb el temps.

4. **La validació d'entrada no és el control de seguretat real.**
   Important: `ValidateURL` és només un filtre barat previ.
   `FetchPreview` torna a validar l'esquema, aplica el denylist de
   hostnames *abans* de qualsevol DNS, i valida cada IP resolta a
   `ControlContext` amb `DisableKeepAlives` perquè cada salt de redirecció
   re-dialí. Un projecte creat amb `http://192.168.1.1/` passa
   `ValidateURL` (com ha de ser: l'esquema és vàlid) i és **rebutjat al
   dial** amb `ErrDestinationForbidden`. La defensa és a la capa
   correcta.

5. **El worker usa el context de fons, no el de la petició**
   (`service.go:236`), com a `ideas` — cap cancel·lació prematura ni
   fuita de context de petició.

**Conclusió:** afegir un segon caller no ha debilitat cap mitigació. El
disseny de NIU-6 (una única funció d'egress) és precisament el que fa
que aquesta ampliació sigui segura per construcció.

---

## 3. XSS — `href` i `src` (A03:2021, CWE-79/CWE-83)

Aquest era el punt de risc més alt del canvi: `projects-render.js` passa
de ser estrictament `textContent`-only a escriure dos atributs amb dades
d'origen no confiable. **Són dues rutes de confiança diferents** i les he
traçat per separat, com demanava l'encàrrec.

### 3.1 `src` ← `og:image` (dada de tercers)

Camí: pàgina externa → `parseOpenGraph` → `applyMetaToken` → DB →
`projectDTO.ImageURL` → `img.setAttribute('src', ...)`.

Control server-side: `ogparse.go:162` descarta qualsevol `og:image` que
no passi `isHTTPOrHTTPSURL`. Rebutja `javascript:`, `data:`, `file:` i
els scheme-relative `//host/x.png`. Es descarta el camp silenciosament,
no es trenca tota la preview. **Correcte i és control de codi**, no
dependent de la CSP.

Val a dir que `<img src="javascript:...">` no és executable als
navegadors moderns de tota manera, però la validació és la barrera
correcta i no depèn d'aquesta assumpció.

### 3.2 `href` ← URL de l'usuari (ruta de confiança DIFERENT)

Aquesta és la que calia comprovar de veritat: `renderProjectName`
(`projects-render.js:143`) fa `link.setAttribute('href', project.url)`
amb la cadena que **l'usuari** ha enviat — que **no** passa mai per
`isHTTPOrHTTPSURL` (aquesta funció només s'aplica a `og:image`).

La barrera única per a aquest camí és `ideas.ValidateURL`
(`ideas/service.go:61-85`). L'he fuzzejat amb 18 payloads d'evasió
d'esquema executant la lògica verbatim:

```
"javascript:alert(1)"                    rebutjat
"JavaScript:alert(1)"                    rebutjat
"jAvAsCrIpT:alert(1)"                    rebutjat
"java\tscript:alert(1)"                  rebutjat
"\x00javascript:alert(1)"                rebutjat
" javascript:alert(1)"                   rebutjat
"javascript://example.com/%0aalert(1)"   rebutjat
"data:text/html,<script>alert(1)</script>" rebutjat
"vbscript:msgbox(1)"                     rebutjat
"//evil.com/x"                           rebutjat
"http:example.com"                       rebutjat   (Host buit)
```

Cap variant de `javascript:` sobreviu. La raó tècnica és que
`url.Parse` de Go **normalitza l'esquema a minúscules** abans de la
comparació, i rebutja tabs/nulls/espais dins l'esquema com a URL
invàlida. La comprovació addicional `parsed.Host == ""` tanca
`http:example.com` i `javascript://...` (que parseja amb esquema
`javascript`, rebutjat abans). **No hi ha XSS explotable via `href`.**

### 3.3 Metacaràcters HTML dins una URL vàlida

Aquests **sí** s'accepten i es persisteixen crus:

```
"http://example.com/\"><img src=x onerror=alert(1)>"   acceptat
"https://example.com/?q=</a><script>alert(1)</script>" acceptat
```

**No són explotables**: arriben al DOM exclusivament via
`setAttribute()` i `textContent`, mai per `innerHTML` ni interpolació de
cadena. `setAttribute` no invoca el parser HTML — el valor es tracta com
a dada literal de l'atribut. He verificat que `projects-render.js` no
conté cap `innerHTML` en tot el fitxer. Correcte per NFR-02.

### 3.4 Paper real de la CSP

`Content-Security-Policy: default-src 'self'; script-src 'self'; ...
img-src 'self' https:; object-src 'none'; base-uri 'none'`
(`middleware.go:47-50`).

La CSP aquí és **defensa en profunditat, no el control primari** — i
això és el correcte. `script-src 'self'` (sense `unsafe-inline`) faria
inert un hipotètic `javascript:` a `href` en navegadors que apliquin CSP
als navigation-schemes, però la barrera efectiva i intencionada és la
validació server-side. `object-src 'none'` i `base-uri 'none'` són
presents. Cap canvi de CSP necessari, com deia la spec.

---

## 4. `rel="noopener noreferrer"` — correcte

`projects-render.js:145-146`:

```js
link.target = '_blank';
link.rel = 'noopener noreferrer';
```

Present, ben escrit i aplicat al mateix element que porta
`target="_blank"`. Sense `noopener`, la pàgina de destí obtindria
`window.opener` i podria fer reverse tabnabbing (redirigir la pestanya
d'origen a un phishing de Niu). Coherent amb `ideas-render.js:61-62`.

Reforç addicional que ja hi era: `Referrer-Policy: no-referrer` global
(`middleware.go:46`), de manera que ni la navegació ni la càrrega de la
miniatura filtren la URL de Niu al tercer.

---

## 5. Esgotament de recursos amb el pool compartit (A04:2021, CWE-770)

**Avaluació: acceptable per al context de desplegament. Sense finding.**

El pool passa de tenir un productor (ideas) a dos (ideas + projects),
amb `workerPoolSize = 6` i cua de `6*4 = 24` (`workerpool.go:25,53`).

**El sostre de memòria no canvia.** És el punt clau i està ben raonat a
`main.go:98-106`: el límit de 2MiB per scrape s'aplica *per treballador
en vol*, no per productor. Amb 6 treballadors el pic segueix sent
~12MiB, exactament el mateix que abans de NIU-11. Un segon pool
**duplicaria** el sostre a ~24MiB sense necessitat. La decisió de
compartir és la més conservadora de les dues en memòria, dins del límit
de 128M del contenidor (`compose.yaml`).

**Starvation entre espais:** teòricament sí — la cua és FIFO i sense
quota per espai, de manera que 24 idees encuades de cop retarden la
resolució de la preview d'un projecte. L'impacte real és que la
miniatura triga uns segons més a aparèixer; la fila del projecte ja s'ha
creat i renderitzat (el 201 mai espera l'scrape, ADR-03). Amb 5s de
timeout dur per scrape i 6 treballadors, drenar una cua plena és de
l'ordre de ~20s en el pitjor cas absolut. Per a una app de dues
persones autenticades això no és una condició de denegació de servei.

**`Submit` no bloqueja el handler HTTP** més enllà de la capacitat del
buffer, i si la cua és plena l'`select` amb `ctx.Done()` evita la fuita
d'una goroutine bloquejada. No hi ha creixement il·limitat de goroutines.

Nota de context: si mai s'obre el registre a usuaris no confiats,
aquesta anàlisi s'ha de refer — allà sí caldria quota per usuari. Avui
la superfície és autenticada i de dos usuaris.

---

## 6. Exposició de `description` (A01:2021, CWE-213)

**Verificat: no és assolible per cap camí.**

- `projectDTO` (`dto.go:71-84`) **no té** camp `Description`. Com que Go
  serialitza per struct explícita, un camp absent de l'struct no pot
  aparèixer al JSON.
- `toProjectDTO` (`dto.go:88-105`) no el copia.
- Els **tres** punts de serialització de projectes
  (`projects_handlers.go:30`, `:59`, `:98` — list, create, patch) passen
  tots per `toProjectDTO`/`toProjectDTOs`. Cap fa `writeJSON` d'un
  `projects.Project` cru.
- `grep` de `Description` a `internal/projects`, `internal/httpapi` i
  `internal/store`: l'única aparició a `dto.go` és la de **`ideaDTO`**
  (línia 125), que és una altra entitat i que sí l'exposa
  deliberadament.
- El test d'integració `TestProjects_Add_WithURL_DescriptionNeverExposed`
  (`projects_test.go:563-597`) ho verifica **contra els bytes crus** de
  la resposta (`strings.Contains(body, "\"description\"")`) tant al POST
  com al GET, no contra un struct que ignoraria silenciosament una clau
  desconeguda. És la manera correcta d'escriure aquesta asserció.

---

## 7. Escombrada OWASP Top 10 (2021)

| Risc | Estat | Nota |
| --- | --- | --- |
| **A01 Broken Access Control** | OK | Ruta sota `WithCurrentUser` (401 abans del handler) + `RequireCSRF`. `description` no exposada (§6). |
| **A02 Cryptographic Failures** | N/A | El canvi no toca cripto, secrets ni emmagatzematge sensible. |
| **A03 Injection** | OK | SQL 100% parametritzat (`store/projects.go` Create/UpdatePreview). XSS analitzat a §3 — sense camí explotable. |
| **A04 Insecure Design** | OK | Reutilitza un patró ja auditat en lloc d'inventar-ne un. Límits de recursos heretats i intactes (§5). |
| **A05 Security Misconfiguration** | OK | CSP, `nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer` presents i sense canvis. Migració amb `CHECK` constraint i `Down` simètric. |
| **A06 Vulnerable Components** | OK | `go.mod` **sense canvis** — cap dependència nova. |
| **A07 Auth Failures** | OK | Cap superfície d'auth nova. Rate limiting de login intacte. |
| **A08 Data Integrity Failures** | OK | Cap deserialització insegura: `json.Decoder` cap a un struct tipat tancat. |
| **A09 Logging & Monitoring** | OK amb nota | Vegeu N-01. |
| **A10 SSRF** | OK | §2 — sense bypass. |

**Secrets:** `grep` del diff per patrons de clau/token/`.env`/clau
privada — **cap**. El User-Agent sortint és credential-free per disseny
(`client.go:19`): cap Cookie ni Authorization viatja al tercer.

---

## 8. Observacions no bloquejants

### N-01 (`minor`) — La URL de l'usuari s'escriu als logs

`service.go:262`:

```go
slog.Debug("fetchsafe: project preview resolution failed",
    "project_id", projectID, "url", rawURL, "error", err)
```

La URL controlada per l'usuari acaba al log. És a nivell `Debug` (fora
de producció normalment), replica exactament el que ja fa `ideas`, i les
URLs de botigues no són PII sensible en aquest context. Però una URL pot
portar tokens a la query string, i el sistema de logs (openobserve) té
una audiència diferent de la de l'app. CWE-532, exposició baixa.

*Recomanació (no bloquejant):* considerar loguejar només l'`host` en
lloc de la URL completa. Aplica igualment a `ideas` — és deute heretat,
no introduït aquí.

### N-02 (`nit`) — Sense límit de longitud a la URL d'entrada

`validateProjectURL` no imposa cap límit de longitud (la spec ho
reconeix explícitament a T-07). Els camps *recuperats* sí que estan
limitats (`maxImageURLLen = 2048`, etc.), però la URL enviada per
l'usuari no. Un usuari autenticat podria desar una URL de diversos MB
i inflar la BD de SQLite.

Explotabilitat real: molt baixa (requereix sessió vàlida + CSRF, i són
dos usuaris de confiança). Consistent amb `ideas`. Ho deixo com a nit
per coherència futura, no com a acció.

### N-03 (`nit`) — Confiança de contingut mixt a la miniatura

Si un `og:image` és `http:` en una pàgina servida per `https:`, el
navegador el bloquejarà per mixed content i la fila degradarà a "sense
miniatura". Ja està documentat i acceptat a la spec. Ho anoto només
perquè es manifestarà com un "bug" visual ocasional que **no** ho és.

---

## 9. Conclusió

El canvi **no introdueix cap vulnerabilitat explotable**. Les tres
decisions que el fan segur són deliberades i visibles al codi:
reutilitzar `ideas.ValidateURL` en lloc de duplicar-la, reutilitzar el
`fetchsafeClient` únic en lloc de crear-ne un de segon, i mantenir
`description` fora de l'struct del DTO en lloc de confiar que ningú la
llegeixi.

Aquest és el resultat directe de com es va construir `fetchsafe` a
NIU-6: quan l'única sortida de xarxa és una sola funció, afegir un segon
consumidor és una operació segura per construcció. Val la pena
mantenir aquesta invariant explícitament a qualsevol futur canvi.

`go build`, `gofmt -l`, `go vet` i `go test ./...` tots nets.

**Veredicte: APPROVED.**
