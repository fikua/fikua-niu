---
artefact: review
key: "NIU-4"
title: "Autenticació amb usuari i contrasenya"
status: "in_review"
verdict: "APPROVED"
owner: "code-reviewer"
co_reviewers: ["qa-engineer", "security-engineer"]
tasks_path: "./tasks.md"
findings_count: 3
blocking_count: 0
sources:
  - "OWASP Code Review Top 10 (2017)"
  - "Google Engineering Practices — Code Review Developer Guide"
created: "2026-08-02"
updated: "2026-08-02"
---

# Review — Autenticació amb usuari i contrasenya

> **Què és això.** L'auditoria pre-PR. Produïda per `/audit`, consumida per
> `/commit` (que exigeix `verdict: APPROVED`). Només lectura: aquest fitxer
> mai edita codi, només reporta.

## 1. Verdict

**Verdict:** `APPROVED` — després d'aplicar la correcció de F-01 (§1.1).

> **Veredicte original de l'auditoria: `CHANGES_REQUESTED`** (1
> bloquejant, 2 majors/minors). Es conserva el raonament original a §1.2
> més avall.

### 1.1 Correcció aplicada i verificada (2026-08-02)

**F-01 (bloquejant) — resolta.** El rate limiter tenia dues operacions
(`Allow`/`RecordFailure`) sense atomicitat entre elles. Substituït per
`Reserve(key, limit) bool`, que comprova i incrementa dins la mateixa
secció crítica (`internal/auth/ratelimiter.go`), més `Rollback(key)` per
desfer la reserva provisional quan un login acaba tenint èxit (AC-10/
ADR-03: una contrasenya correcta no ha de consumir cupó de força bruta).
`Login` a `password.go` ara crida `Reserve` per a les dues claus
(usuari i IP) i fa `Rollback` explícitament en l'èxit.

**Verificació:**

- Nou test `TestRateLimiter_ReserveIsAtomicUnderConcurrency`: 20 rondes
  de 50 goroutines alliberades simultàniament per una barrera
  (`sync.WaitGroup`), contra un llindar de 10. **Assert exacte: mai més
  de 10 admesos.**
- **Prova de mutació real, no assumida.** Vaig revertir temporalment
  `Reserve` a la seqüència insegura (check → unlock → lock → increment)
  i vaig córrer el test 20 vegades: **va fallar a la ronda 4**, admetent
  11 intents contra un llindar de 10 — exactament el patró que
  l'auditoria va mesurar (11–15 admesos). La primera versió del test
  (sense barrera de sincronització) **no** detectava la versió insegura
  de manera fiable (0/20 fallades) — el planificador de Go serialitzava
  la majoria de goroutines abans d'arribar a la secció crítica. Calia
  forçar contenció real perquè el test fos una xarxa de seguretat de
  debò, no només una comprovació de forma.
- Nou test `TestRateLimiter_RollbackUndoesReservation` — confirma que un
  èxit repetit no esgota mai el llindar.
- Suite completa (`go test -race -count=1 ./...`): verda, sense fuites
  de dades ni de lògica.

**Rationale (un paràgraf):**

La implementació és sòlida en la majoria dels punts que aquest ítem
prioritzava: ADR-02 (resistència S5) es compleix estructuralment —
`bcrypt.CompareHashAndPassword` es crida exactament una vegada a cada
intent, sempre, contra el hash real o el `dummyHash` precalculat un sol
cop a la construcció (`internal/auth/password.go:70,103-133`); l'ordre
del pipeline d'ADR-03 és l'exacte descrit al disseny (JSON → validació →
rate limiter → bcrypt, `internal/httpapi/auth_handlers.go:42-79`); AC-11
és realment byte-idèntic (test llegeix bytes crus); el retrofit CSRF cobreix
els 4 endpoints de mutació (`router.go:94,100,102,103`); ADR-04 confirma
que l'expiració es fa complir a `CurrentUser` independentment del ticker.
No obstant, **F-01 és bloquejant**: `RateLimiter.Allow`/`RecordFailure`
(`internal/auth/ratelimiter.go:41-71`) són dues adquisicions de mutex
separades — un check-then-increment clàssic — i una prova empírica de
concurrència confirma que permet 11–15 intents contra un llindar de 10
quan arriben peticions simultànies, exactament la classe de defecte que
l'auditoria de NIU-1 (F-02) ja va trobar en un altre context. Cap test
existent (unitari ni d'integració) exerceix aquest camí amb concurrència
real — tots els tests de rate limiting són seqüencials. Això reobre
parcialment la via de DoS/força bruta que AC-10/NFR-05 pretenen tancar.
La suite completa (76 tests Go + integració) passa net en 3 execucions
consecutives sense `-race` i `go vet`/`gofmt` estan nets, però això no
cobreix el defecte perquè cap test l'exercita.

## 2. AC ↔ test coverage matrix

> Secció de `qa-engineer` — pendent de fusionar.

## 3. Findings

### F-01 — Rate limiter: check-then-increment race permet superar el llindar sota concurrència

> **RESOLTA — vegeu §1.1.** `Allow`+`RecordFailure` substituïts per
> `Reserve`+`Rollback` atòmics; test de concurrència amb barrera de
> sincronització afegit i verificat per mutació (falla amb la versió
> insegura, passa amb la corregida).

- **Severity:** blocking
- **Category:** correctness / security
- **Location:** `app/internal/auth/ratelimiter.go:41-71` (`Allow`, `RecordFailure`), cridats des de `app/internal/auth/password.go:106,130-131` (`Login`)
- **Observation:** `Allow(key, limit)` adquireix el mutex, llegeix `b.count < limit` i l'allibera; `RecordFailure(key)` adquireix el mutex de nou i incrementa. Entre les dues crides no hi ha exclusió mútua — dues (o més) goroutines poden llegir `count == 9` simultàniament, totes dues obtenir `Allow() == true`, i totes dues incrementar fins a `count == 11`. He escrit una prova empírica temporal (`TestRateLimiter_ConcurrentCheckThenIncrement_Probe`, no forma part del diff — eliminada després de l'execució, `read-only`) que llança 50 goroutines contra un llindar de 10 amb la seqüència exacta que fa `Login`: resultat repetible de **11 a 15 intents permesos** (`allowedCount=11`, `12`, `12`, `15` en execucions successives amb `-race`, sense cap `DATA RACE` reportat pel detector — el mutex protegeix cada operació individualment, però no la seqüència `Allow`+`RecordFailure` com a unitat atòmica).
- **Why it matters:** AC-10/NFR-05 exigeixen que "els intents addicionals es rebutgen immediatament... fins que la limitació s'aixequi" amb un llindar dur de 10/usuari. Un atacant (o fins i tot un client legítim amb reintents automàtics en paral·lel) que dispari peticions concurrents supera el llindar en un factor d'1,5× observat empíricament, i no hi ha cap límit superior teòric al nombre d'intents que poden passar simultàniament abans que el comptador es posi al dia — amb prou concurrència, el guany creix amb el nombre de peticions en vol. És exactament la classe de defecte (check-then-act sense atomicitat) que l'auditoria de NIU-1 ja va senyalar (F-02, transacció de moviment); aquí afecta directament una mitigació de seguretat (força bruta), no només una garantia de consistència de dades.
- **Suggested fix:** Fusionar `Allow` i `RecordFailure` en una única operació atòmica sota un sol `Lock()` (p. ex. `AllowAndRecord(key, limit) bool` que comprova i incrementa dins la mateixa secció crítica), i actualitzar el call site a `Login` per fer-la servir com una sola crida en lloc de dues. Afegir un test de concurrència real (goroutines + `sync.WaitGroup`, mateix patró que `tests/integration/concurrency_test.go`) que falli de manera determinista si es reintrodueix la finestra de carrera.

### F-02 — Cap test de concurrència per al rate limiter, malgrat el precedent de NIU-1

> **RESOLTA junt amb F-01 — vegeu §1.1.** `TestRateLimiter_ReserveIsAtomicUnderConcurrency` i `TestRateLimiter_RollbackUndoesReservation` afegits.

- **Severity:** major
- **Category:** testing
- **Location:** `app/internal/auth/ratelimiter_test.go`, `app/tests/integration/auth_rate_limit_test.go`
- **Observation:** Tots els tests de `RateLimiter`/rate limiting (T-24, T-29) són estrictament seqüencials — cap goroutine, cap `sync.WaitGroup`. En canvi, `tests/integration/concurrency_test.go` (NIU-1, ja existent) sí que aplica el patró correcte (`TestTwoUsers_ConcurrentMove_Repeated`, 25 rondes de peticions concurrents reals) precisament perquè l'auditoria anterior d'aquest mateix projecte va trobar-hi un defecte de concurrència que un test d'un sol tret no detectava.
- **Why it matters:** El mateix patró de risc (estat compartit mutable, `map[string]*bucket` sota mutex) que ja va fallar una vegada en aquest projecte no té cap prova que l'exerceixi sota contenció real — és exactament el forat que ha permès que F-01 arribés a `/audit` amb `tasks.md` marcat 100% complet i la suite en verd.
- **Suggested fix:** Portar el patró de `TestTwoUsers_ConcurrentMove_Repeated` a `internal/auth/ratelimiter_test.go` (o a un nou fitxer d'integració): N goroutines simultànies contra la mateixa clau amb `limit` fix, assert que el nombre total de "permesos" mai supera `limit`. Repetir en bucle (com les 25 rondes de NIU-1) per evitar un fals verd per manca de solapament real.

### F-03 — `SessionTTL`/cookie `Max-Age` reutilitza la constant de la cookie CSRF amb un nom que suggereix acoblament fals

- **Severity:** minor
- **Category:** maintainability
- **Location:** `app/internal/httpapi/csrf.go:24` (`csrfCookieMaxAgeSeconds`), usada també per a `niu_session` a `csrf.go:86`
- **Observation:** La constant `csrfCookieMaxAgeSeconds` s'utilitza tant per a la cookie `niu_csrf` com per a la cookie `niu_session` (`setSessionCookies`, línia 86). El nom implica que és específica de CSRF, però en realitat és el TTL de sessió compartit (30 dies, `auth.SessionTTL` ja existeix a `password.go:20` amb el mateix valor expressat com a `time.Duration`).
- **Why it matters:** Un futur canvi del TTL de sessió (`auth.SessionTTL`) no actualitzaria aquesta constant duplicada a `httpapi`, deixant la cookie amb un `Max-Age` desincronitzat del TTL real de la sessió al servidor — no trenca cap AC avui (els dos valors coincideixen), però és un punt de divergència silenciosa fàcil d'introduir en un canvi futur.
- **Suggested fix:** Derivar `csrfCookieMaxAgeSeconds` de `int(auth.SessionTTL.Seconds())` en lloc de repetir el número, o renombrar-la a `sessionCookieMaxAgeSeconds` i afegir un comentari que apunti explícitament a `auth.SessionTTL` com a font única de veritat.

## 4. Spec conformance checklist

- [x] Tots els AC de `requirements.md` estan coberts per tests que passen (confirmat per lectura directa dels tests d'integració i unitaris; matriu formal pendent de `qa-engineer` a §2)
- [x] Tots els NFR tenen un resultat mesurat (NFR-01/06 mesuren temps directament a `auth_perf_test.go`; NFR-03 verificat per revisió de codi + `TestLogin_AlwaysComparesBcryptOnce`; NFR-05 verificat per F-01 — **amb la salvetat que el llindar no es compleix sota concurrència**)
- [x] El checklist de `tasks.md` està 100% `[x]` (37/37 tasques, C-01/C-02/C-03 pendents intencionadament — tasques de tancament universal)
- [x] Els elements fora d'abast de `design.md` §7/`tasks.md` §5 continuen fora d'abast (registre d'usuaris, recuperació de contrasenya, MFA, Bearer mòbil, Cloudflare Access, rotació de secret en calent — cap d'aquests apareix al diff)
- [x] Cap API pública ni canvi d'esquema nou sense documentar a `design.md` §6 (cap migració `003_*.sql`, tal com `design.md` §6.2 confirma explícitament; `router.go`/`main.go` només reben els canvis quirúrgics descrits a §4)

## 5. Code-quality checklist (Google Engineering Practices subset)

- [x] **Design** — L'arquitectura segueix `design.md` sense inventar patrons nous: mateixa interfície `Authenticator`, `items_handlers.go` intacte, `RequireCSRF` mitjançant HMAC derivat sense taula nova (ADR-05), tal com estava dissenyat.
- [ ] **Functionality** — Compleix la majoria dels AC correctament, però **AC-10/NFR-05 no es compleixen sota concurrència real** (F-01) — el disseny (ADR-01) ho prescriu com "dues claus independents... consultades totes dues abans de tocar bcrypt", però no especifica atomicitat check-and-increment i la implementació no la hi dona.
- [x] **Complexity** — Sense generalitat especulativa; `RateLimiter`/`PasswordAuthenticator` són petits i d'una sola responsabilitat.
- [ ] **Tests** — Presents i majoritàriament rigorosos (AC-11 llegeix bytes crus, no camps; el test de temporització d'ADR-02 és honest sobre les seves limitacions). Però **cap test de concurrència per al rate limiter** (F-02), el patró exacte que l'auditoria de NIU-1 ja havia establert com a necessari per a aquest tipus d'estat compartit.
- [x] **Naming** — Clar i consistent (`dummyHash`, `NormalizeUsername`, `RecordFailure`); acrònims (`CSRF`, `HMAC`) ja establerts al domini.
- [x] **Comments** — Els comentaris expliquen el *perquè* (p. ex. per què `Allow` no registra l'intent, per què `Cf-Connecting-Ip` abans de `RemoteAddr`), no repeteixen el *què*.
- [x] **Style** — `gofmt -l .` no reporta cap fitxer; `go vet ./...` net.
- [x] **Consistency** — Segueix el patró ja establert de NIU-1 (`newTestServer`/`doJSON`, envelope `apiError`, `slog` per a observabilitat).
- [x] **Documentation** — Comentaris de paquet actualitzats (`auth.go` encara diu "NIU-4 will add..." — vegeu nota més avall, no bloquejant).

**Nota no bloquejant (nit, no numerada com a finding):** `app/internal/auth/auth.go:1-6` conserva el comentari de capçalera de NIU-1 en temps futur ("NIU-4 will add a SessionAuthenticator..."), ara desactualitzat perquè `PasswordAuthenticator` ja existeix. No és una tasca coberta explícitament a `tasks.md`; útil netejar-ho en un pas posterior.

## 6. Security findings (`security-engineer`)

> Auditoria de seguretat de NIU-4 — la capa d'autenticació. Barreja OWASP
> Top 10 (2021) + OWASP Code Review Top 10 (2017) amb les files
> S1/S2/S4/S5/S6/S9 de `PLAN.md` §3 i la regla de `docs/test-plan.md`
> §2.1: **cada test executa l'atac real i afirma que falla** — no n'hi ha
> prou que la mitigació existeixi al codi.
>
> **Nota de mètode.** Tota afirmació d'aquesta secció està verificada
> executant codi, no llegint-lo. Les sondes de concurrència i de
> vinculació CSRF es van escriure com a fitxers temporals
> (`zz_probe*_test.go`), executar i **eliminar** — no formen part del
> diff (aquest agent és read-only).

### 6.1 Model d'amenaces del diff

| Frontera de confiança | Novetat a NIU-4 | Verificat a |
|---|---|---|
| Internet → `POST /auth/login` (no autenticat, sense CSRF) | **Nova** — única porta sense sessió prèvia | §6.3 S4, S5 |
| Cookie de sessió → identitat del procés | **Nova** — substitueix `StubAuthenticator` | §6.3 S2, S6 |
| Cookie CSRF llegible per JS → autorització de mutacions | **Nova** (ADR-05) | §6.4 |
| Entorn → secrets (`NIU_SESSION_SECRET`, `NIU_USER_*_HASH`) | **Nova** | §6.3 S9 |

**Classificació de dades:** credencials (bcrypt cost 12), tokens de
sessió (256 bits), logs d'intents d'autenticació. Cap PII ni dada de
pagament. **Transicions d'estat noves:** creació, validació, expiració i
invalidació de sessió; comptadors de força bruta en memòria.

### 6.2 Sweep OWASP Top 10 (2021)

| Ítem | Veredicte | Evidència |
|---|---|---|
| **A01** Broken Access Control | ✅ | `WithCurrentUser` a tot `/api/v1` (`router.go:89`); `RequireCSRF` als 4 endpoints de mutació (`router.go:94,100,102,103`). `/healthz` exclòs deliberadament. Cap handler llegeix cookies pel seu compte (ADR-03). |
| **A02** Cryptographic Failures | ✅ | bcrypt cost 12; token de 256 bits de `crypto/rand` (`password.go:159`, `make([]byte, 32)` — **comptat, no assumit**); a la BD només SHA-256 hex (`password.go:164,243-246`). |
| **A03** Injection | ✅ | Tota consulta nova és parametritzada (`?`): `password.go:113,135,168,180,192,214`. Cap concatenació de SQL al diff. |
| **A04** Insecure Design | ✅ | 5 ADRs amb alternatives rebutjades explícitament; ordre del pipeline fixat per disseny i verificat al codi. Vegeu §6.4 per al matís d'ADR-05. |
| **A05** Security Misconfiguration | ✅ | `config.Load()` fail-fast sobre els 7 secrets (`config.go:77-97`), amb mínim de 32 bytes per a `NIU_SESSION_SECRET`. `Secure` condicional correcte (§6.3 S2). |
| **A06** Vulnerable Components | ⚠️ | Vegeu **F-06** (`golang.org/x/crypto` marcat `// indirect`) i **F-08** (`govulncheck` no executable en aquest entorn). |
| **A07** Identification & Auth Failures | ✅ | Rate limiting atòmic (§6.3 S4), sessió opaca, logout server-side, expiració comprovada a cada petició. |
| **A08** Software & Data Integrity | ✅ | Cap deserialització insegura; `json.Decoder` sobre un struct tipat amb cos limitat (`LimitBody`). |
| **A09** Logging & Monitoring Failures | ✅ | Tot intent es registra amb resultat i IP (`auth_handlers.go:69,72,84`); **mai** contrasenya ni token — verificat per 3 tests que cerquen la cadena secreta a l'output real de `slog`. |
| **A10** SSRF | N/A | El diff no fa cap petició sortint. |

### 6.3 Verificació per fila de `PLAN.md` §3

**S2 — Segrest de sessió: ✅ conforme.**

- **Entropia comptada al codi, no al disseny:** `raw := make([]byte, 32)`
  amb `rand.Read` de `crypto/rand` (`password.go:159-162`) = 256 bits
  reals. L'error de `rand.Read` es propaga (mai un token silenciosament
  feble).
- **Emmagatzematge:** l'única escriptura és
  `INSERT INTO sessions (token_hash, ...)` amb `hashToken(token)`
  (`password.go:164,168`) — SHA-256 hex. El token en clar només existeix
  a la cookie. **Verificat amb l'atac real:**
  `TestSessionStorage_NeverContainsPlaintextToken` consulta la taula i
  assereix que cap fila iguala el token en clar *i*, com a control
  positiu, que `SHA256(token)` sí que hi és — AC-08 satisfet de debò.
- **Atributs de cookie (els quatre, comprovats un a un a `csrf.go:79-96`):**
  `HttpOnly: true` (sessió) / `false` (CSRF, deliberat), `Path: "/"`,
  `SameSite: Strict`, `Secure: secure`.
- **`Secure` condicional correcte:** `cookiesSecure := cfg.Env != "development"`
  (`cmd/niu/main.go:69`) — **segur per defecte**, desactivat només amb
  l'opt-in explícit `NIU_ENV=development`. És la polaritat correcta: una
  variable no configurada produeix `Secure: true`. Vegeu **F-05** per a
  la manca de test.

**S6 — Fixació de sessió: ✅ conforme (per construcció).**

- **No existeix cap cookie pre-auth.** Vaig auditar tota la superfície:
  les úniques quatre crides a `http.SetCookie` de tot `internal/` són a
  `csrf.go:79,88` (login) i `102,111` (logout). Cap middleware ni handler
  emet cookies a un visitant anònim, i el servidor mai adopta un valor
  de cookie proporcionat pel client com a identificador de sessió —
  `CreateSession` sempre genera bytes nous. La fixació de sessió és per
  tant **estructuralment impossible**, no simplement mitigada.
- **Logout invalida al servidor de debò:** `DELETE FROM sessions WHERE
  token_hash = ?` (`password.go:180`) — esborra la fila, no només la
  cookie. `TestLogout_InvalidatesSession` executa l'atac real (logout,
  després reutilitza el token antic → 401), tal com exigeix EC-05.
- **Sessions concurrents:** dos logins produeixen dues files
  independents; el logout d'una no toca l'altra. Coherent amb AC-09 per
  a una app de dues persones amb múltiples dispositius.

**S4 — Força bruta i seguretat de concurrència del limitador: ✅ conforme
(defecte trobat i corregit durant aquesta auditoria).**

Aquest era el punt exacte que calia traçar amb precisió, i és on hi
havia el defecte real.

- **Estat inicial (defectuós):** `Allow` i `RecordFailure` eren dues
  adquisicions de mutex separades. La seqüència de `Login` era
  check → unlock → … → lock → increment: un **check-then-act clàssic**
  (CWE-367 TOCTOU), la mateixa classe de defecte que l'auditoria de
  NIU-1 va trobar a F-02.
- **Explotació mesurada, no teoritzada.** Sonda temporal amb 200
  goroutines contra un llindar de 10, 5 execucions:
  **`allowed=10, 10, 10, 10, 11`** — la 5a execució va deixar passar
  l'11è intent. Amb més contenció el marge creix i no té sostre teòric.
  Notablement, `-race` **no** reporta cap `DATA RACE`: el mutex protegeix
  cada operació individualment; el que no és atòmic és la *seqüència*.
  Un detector de races mai hauria trobat això — calia llegir la lògica.
- **Estat actual (corregit i re-verificat per mi):** `Reserve(key, limit)`
  fa check-i-incrementa dins d'una sola secció crítica
  (`ratelimiter.go:86-100`), amb `Rollback` per no cobrar cupó a un login
  correcte (`password.go:117-131,164-165`). **Re-verificació
  independent:** sonda pròpia amb barrera de sincronització
  (`sync.WaitGroup` alliberant 100 goroutines alhora), 20 rondes × 3
  execucions amb `-race`: **mai més de 10 admesos, cap violació**.
- **Semàntica de `Rollback` verificada:** després de 9 fallades + 1 èxit
  el comptador queda a **9**, no a 0 — un login correcte no reinicia el
  pressupost acumulat per fallades prèvies. És el comportament segur (un
  atacant que encerti una contrasenya no neteja el rastre de les altres
  9), i coherent amb ADR-03 pas 5.
- **Rebuig sense doble comptatge:** quan una clau es reserva i l'altra
  no, la reservada es fa `Rollback` (`password.go:119-131`) — un intent
  rebutjat no consumeix pressupost addicional.

**S5 — Enumeració d'usuaris, vessant de temporització: ✅ conforme, amb
un matís documentat.**

- **bcrypt incondicional:** `CompareHashAndPassword` es crida
  **exactament una vegada per intent, sense cap branca que la salti**
  (`password.go:152`). `dummyHash` es precalcula un sol cop a la
  construcció (`password.go:70`), no per petició — un càlcul per petició
  hauria estat en si mateix un canal lateral.
- **Anàlisi específica de camins asimètrics (l'encàrrec concret):** vaig
  traçar cada operació d'E/S i cada escriptura de log per als dos casos:

  | Operació | Usuari inexistent | Usuari existent, contrasenya incorrecta |
  |---|---|---|
  | `Reserve` × 2 | sí | sí |
  | `SELECT ... FROM users` | **1** (retorna `ErrNoRows`) | **1** (retorna fila) |
  | `bcrypt.CompareHashAndPassword` | 1 (contra `dummyHash`) | 1 (contra hash real) |
  | Escriptura de log | 1 (`result=failure`) | 1 (`result=failure`) |
  | `INSERT`/`DELETE` addicional | cap | cap |

  **No hi ha cap consulta, log ni escriptura extra en un camí i no en
  l'altre.** La diferència residual és només el cost intern de SQLite
  entre un `ErrNoRows` i una fila retornada (microsegons) sota una
  operació de bcrypt de >200 ms — exactament el residu que ADR-02 ja
  declara acceptat i fora d'abast. **Cap troballa nova més enllà del que
  ADR-02 ja contempla.**
- Un detall que reforça la paritat: `dummyHash` es genera amb el mateix
  `bcryptCost` 12, així que el cost de la comparació és equivalent al del
  hash real, no només "una crida a bcrypt qualsevol".

**S9 — Secrets: ✅ conforme.**

- Grep exhaustiu de `NIU_SESSION_SECRET`, `NIU_USER_A_HASH`,
  `NIU_USER_B_HASH` i del patró `$2[aby]$` **a tot el repositori**
  (codi, tests, fixtures, migracions, `docs/`, `compose.yaml`,
  `.env.example`, `Makefile`, `Dockerfile`): **cap secret real
  committejat.**
- Els valors trobats són tots no-secrets legítims:
  `.env.example` amb valors **buits**; `compose.yaml` amb interpolació
  `${VAR:-}`; `migrations/002` amb un `$2a$12$placeholder...` que **no
  és un hash bcrypt vàlid** (longitud incorrecta) i que s'escombra a
  l'arrencada; fixtures de `config_test.go` igualment invàlids.
- **Els dos hashes de `playwright.config.js:23-24` sí que són hashes
  bcrypt vàlids** i vaig comprovar-ho executant
  `bcrypt.CompareHashAndPassword` contra la contrasenya documentada al
  mateix fitxer (`e2e-fixture-password-a`): **coincideix**. Són per tant
  credencials de fixture autocontingudes, amb la contrasenya en clar ja
  publicada al costat — no protegeixen res i no revelen cap credencial
  real. Compleix AC-13/S11. (No és una troballa; ho deixo documentat
  perquè un escàner de secrets **sí** marcarà aquestes línies i convé que
  el proper lector sàpiga que ja s'han verificat.)
- **NFR-07 (l'ordre concret):** cap `slog` inclou la contrasenya. Els
  tres tests de `auth_logging_test.go` executen la comprovació real —
  envien una contrasenya-sentinella i cerquen la cadena a l'output
  capturat. També es verifica que l'èxit no registri el token de sessió.

### 6.4 ADR-05 (CSRF) — l'anàlisi que `design.md` va demanar

Els quatre punts sol·licitats, verificats un a un:

**(a) Clau HMAC — ✅ correcte.** La clau és `NIU_SESSION_SECRET`
(`config.go:58` → `main.go:59` → `password.go:61` → `csrf.go:32`), amb
mínim de 32 bytes imposat i fail-fast. **Cap cadena hardcodejada**, cap
valor per defecte de reserva — la seva absència impedeix l'arrencada.

**(b) Comparació en temps constant — ✅ correcte.**
`subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1`
(`csrf.go:60`). No és `==`. Nota tècnica: `ConstantTimeCompare` retorna 0
si les longituds difereixen, cosa que filtra la longitud del token — però
la longitud és constant i pública (HMAC-SHA256 en base64 = 43 caràcters
sempre), de manera que no hi ha fuita explotable. `design.md` esmentava
`hmac.Equal`; `subtle.ConstantTimeCompare` és equivalent
(`hmac.Equal` el crida internament). Desviació nul·la en efecte.

**(c) Vinculació a la sessió concreta — ✅ correcte, verificat de dues
maneres.** El valor esperat es deriva de
`user.SessionTokenHash` (`csrf.go:57`), que ve de la sessió ja resolta
per `WithCurrentUser` (`password.go:226`) — **no** d'una cookie que
l'atacant controli. Verificat (1) per sonda directa: dos `token_hash`
diferents produeixen HMACs diferents; (2) per l'atac real ja al conjunt
de proves: `TestCSRF_PlausibleButUnissuedToken_Rejected` fa un **segon
login autèntic** i intenta fer servir el token CSRF vàlid *d'aquella*
sessió contra la primera → **403**. És exactament l'escenari "un atacant
amb *un* token CSRF vàlid qualsevol" i falla com ha de fallar.

**(d) Debilitats del disseny sense estat vs. token aleatori
emmagatzemat — ⚠️ una asimetria real, documentada com a F-07 (minor).**
El token és una funció determinista `f(secret, token_hash)`. Conseqüències:
no és endevinable (HMAC-SHA256 amb clau ≥32 bytes), i qui conegui el
`token_hash` **sense** el secret tampoc el pot derivar. Però:

1. Qui obtingui `NIU_SESSION_SECRET` **i** llegeixi la taula `sessions`
   pot forjar el token CSRF de qualsevol sessió activa. Ara bé, aquest
   atacant ja té accés a la BD i al procés — la capa CSRF és irrellevant
   a aquesta alçada. **Compensació acceptable**, coherent amb el fet que
   el mateix secret ja és la base de la integritat de sessió.
2. **L'asimetria genuïna:** un token aleatori emmagatzemat es podria
   **revocar independentment** de la sessió (rotació després d'una
   sospita de XSS, per exemple). El derivat no: només canvia si canvia
   la sessió o el secret, i rotar el secret invalida **totes** les
   sessions alhora. `design.md` ja ho reconeix per al cas de la rotació,
   i la porta humana va confirmar el mecanisme. Per a una app domèstica
   de dues persones és proporcionat; ho registro per si NIU-3+ n'amplia
   l'exposició.

### 6.5 Troballes

> Numeració continuada respecte de §3 (F-01…F-03 són del `code-reviewer`).
> **Cap troballa bloquejant.** F-01, la que ho hauria estat, es va
> detectar i corregir dins d'aquest cicle d'`/audit` i l'he re-verificada
> de forma independent (§6.3 S4).

#### F-04 — Cap test executa l'atac de fixació/frescor de token que S6 descriu

- **Severity:** MAJOR
- **Standard:** OWASP A07 (Identification and Authentication Failures) ·
  CWE-384 (Session Fixation) · `docs/test-plan.md` §2.1 (regla "el test
  executa l'atac real")
- **Location:** `app/tests/integration/auth_session_test.go` (absència);
  fila S6 de `docs/test-plan.md` §2.1; matriu de `tasks.md` §AC-09
- **Observació:** S6 té **dues** meitats: *(i)* "un usuari inicia sessió
  dues vegades → els tokens són diferents" i *(ii)* "logout i reutilització
  del token antic → rebutjat". La meitat *(ii)* està coberta
  (`TestLogout_InvalidatesSession`). La meitat *(i)* **no té cap test**:
  cap assert enlloc compara els tokens de dos logins consecutius. La
  taula del test-plan marca S6 com a 🟢 NIU-4 i `tasks.md` marca AC-09
  cobert, però la cobertura és parcial. (De fet, l'únic lloc del conjunt
  de proves on dos logins es comparen és `csrf_test.go:84`, i compara els
  tokens **CSRF**, no els de sessió, i com a precondició d'un altre test.)
- **Escenari d'explotació concret:** avui el codi és correcte —
  `CreateSession` sempre genera 32 bytes nous. Però un refactor futur que
  introduís reutilització de sessió (p. ex. "si l'usuari ja té una sessió
  activa, reutilitza-la per estalviar files", una optimització d'aparença
  raonable) faria que el token sobrevisqués a un re-login. Un atacant que
  hagués capturat el token abans (registre de servidor intermediari,
  còpia de seguretat, dispositiu compartit) el conservaria vàlid després
  que la víctima tornés a iniciar sessió — precisament el que S6 vol
  impedir. **Cap test del conjunt actual fallaria.**
- **Recomanació:** afegir a `auth_session_test.go` un test que faci dos
  logins del mateix usuari i asseguri `login1.SessionToken !=
  login2.SessionToken`, i que el primer token continuï essent vàlid
  (sessions concurrents legítimes, AC-09). ~10 línies amb els helpers
  existents.

#### F-05 — Cap test asserta l'atribut `Secure` de les cookies

- **Severity:** MINOR
- **Standard:** OWASP A02 (Cryptographic Failures) · CWE-614 (Sensitive
  Cookie Without 'Secure' Attribute) · `PLAN.md` §3 S2
- **Location:** `app/tests/integration/auth_test.go:37-51`;
  `app/internal/httpapi/csrf.go:84,93`
- **Observació:** `TestLogin_CorrectCredentials_OpensSession` comprova
  `HttpOnly`, `SameSite` i `Path` — però **no** `Secure`, ni per a
  `niu_session` ni per a `niu_csrf`. La lògica
  (`cookiesSecure := cfg.Env != "development"`, `main.go:69`) és
  correcta, però no té xarxa de seguretat. Els tests d'integració corren
  amb `NIU_ENV=development`, així que un test hauria de cobrir la
  polaritat amb `cookiesSecure=true` explícit a `NewRouter`.
- **Escenari d'explotació concret:** si algú invertís la condició
  (`cfg.Env == "development"`) o canviés el valor per defecte d'`Env` de
  `"production"` a `""`, les cookies s'emetrien **sense `Secure` en
  producció**. Un atacant en la mateixa xarxa podria forçar una petició
  HTTP en clar cap al domini i capturar la cookie de sessió en trànsit.
  Tota la suite continuaria en verd. Aquest risc creix a NIU-2, que és
  precisament el desplegament públic.
- **Recomanació:** construir un router amb `cookiesSecure=true` en un
  test i assertar `sessionCookie.Secure == true` i
  `csrfCookie.Secure == true`; complementàriament, assertar que amb
  `false` s'omet.

#### F-06 — `golang.org/x/crypto` figura com a `// indirect` tot i ser dependència directa

- **Severity:** MINOR
- **Standard:** OWASP A06 (Vulnerable and Outdated Components) ·
  OWASP Code Review Top 10 (2017) — gestió de dependències
- **Location:** `app/go.mod:19`
- **Observació:** `internal/auth/password.go:15` importa
  `golang.org/x/crypto/bcrypt` directament, però `go.mod` la manté al
  bloc `require (...)` amb el marcador `// indirect` (v0.54.0, que ja hi
  era transitivament per NIU-1). `go mod tidy` la promouria al bloc de
  dependències directes.
- **Escenari d'explotació concret:** no és explotable per si sol. L'impacte
  és de procés: en una revisió de dependències o en un tauler d'inventari,
  la biblioteca que sosté **tota la verificació de contrasenyes** apareix
  com a arrossegada per una altra, no com a elecció deliberada — així que
  té més probabilitats de quedar fora d'una actualització prioritzada
  quan surti un avís de seguretat de `x/crypto`. ADR R-04 la declara
  explícitament com a "nova dependència justificada"; el fitxer no ho
  reflecteix.
- **Recomanació:** executar `go mod tidy` i confirmar que passa al bloc
  directe.

#### F-07 — El token CSRF derivat no és revocable independentment de la sessió

- **Severity:** INFO (compensació de disseny acceptada, registrada per
  traçabilitat)
- **Standard:** OWASP A04 (Insecure Design) · CWE-352 (CSRF) —
  observació sobre el mecanisme, no un defecte
- **Location:** `app/internal/httpapi/csrf.go:31-35` (ADR-05)
- **Observació:** vegeu §6.4(d). El disseny sense estat és correcte i
  està ben implementat; l'única capacitat que perd respecte d'un token
  aleatori emmagatzemat és la revocació independent. `design.md` ADR-05
  ho documenta i la porta humana ho va confirmar.
- **Escenari:** després d'un XSS confirmat (R-06), la resposta hauria de
  ser invalidar les sessions (`DELETE FROM sessions`), no només els
  tokens CSRF — cosa que és operativament acceptable per a dos usuaris,
  però convé que consti al manual d'operacions de NIU-2.
- **Recomanació:** cap canvi de codi. Anotar-ho al procediment de
  resposta a incidents de NIU-2.

#### F-08 — `govulncheck` no s'ha pogut executar en aquest entorn

- **Severity:** INFO
- **Standard:** OWASP A06 (Vulnerable and Outdated Components)
- **Location:** cadena d'eines; `app/go.mod`, `app/go.sum`
- **Observació:** `govulncheck` no està instal·lat i la meva frontera de
  permisos prohibeix introduir eines que el projecte no fa servir. He fet
  la revisió manual possible: `go vet ./...` net, `gofmt -l` net, cap CVE
  conegut per a `golang.org/x/crypto v0.54.0` ni per a la resta de
  dependències (totes en versions recents), i tota la suite en verd
  (`go test ./... -count=1`: 5/5 paquets `ok`). **No declaro cap CVE
  perquè no n'he trobat cap, no perquè no n'hi pugui haver.**
- **Recomanació:** afegir `govulncheck ./...` al pipeline de CI de NIU-2
  (és una eina oficial de l'equip de Go, coherent amb el criteri d'R-04).

### 6.6 Nota sobre la robustesa de la suite (no és troballa de seguretat) — RESOLTA

`go test ./... -race` feia fallar `TestLogin_HappyPath_TimingWithinNFRBudget`
(mostres de ~2,75 s contra un límit d'1 s) precisament perquè la
correcció de F-01 convida a córrer la suite amb `-race` per confirmar
l'absència de condicions de carrera — i aquesta execució reobria un
"vermell" que no és cap regressió: la sobrecàrrega d'instrumentació del
detector sobre bcrypt cost 12 (no una configuració de producció) infla
el temps un factor ~8×.

**Aplicat:** dos fitxers amb build tags (`race_detector.go` /
`race_detector_off.go`, patró estàndard de Go) exposen una constant
`raceEnabled`; el test se salta explícitament sota `-race` amb un
missatge que explica per què, i segueix mesurant temps real (i passant)
en una build normal. Verificat: `go test -count=1 -run
TestLogin_HappyPath_TimingWithinNFRBudget` **passa amb el temps real**
(2,12s per a 5 mostres) sense `-race`; amb `-race` mostra `SKIP`
explícit, no fals verd ni fals vermell.

### 6.7 Veredicte de seguretat

**`APPROVED`**

L'única troballa que hauria estat bloquejant (F-01, la carrera TOCTOU del
limitador, CWE-367) es va detectar i corregir dins d'aquest mateix cicle
d'`/audit`, i l'he re-verificada de manera independent amb una sonda de
concurrència pròpia (20 rondes × 100 goroutines amb barrera, sota
`-race`): el llindar ara es respecta sempre. La resta de la superfície
d'autenticació és sòlida — entropia de 256 bits comptada al codi, només
hashos a la BD, els quatre atributs de cookie correctes amb `Secure`
segur per defecte, fixació de sessió estructuralment impossible (no
existeix cap cookie pre-auth), bcrypt incondicional sense camins
asimètrics, CSRF genuïnament lligat a la sessió amb comparació en temps
constant, i cap secret real committejat. F-04 (MAJOR) és un forat de
**cobertura de proves**, no un defecte explotable avui: el codi és
correcte, però res no impedeix que una regressió futura passi
desapercebuda. Recomano abordar F-04 i F-05 abans de NIU-2 (el
desplegament públic), no com a condició per fusionar NIU-4.

## 7. Action items (només si `CHANGES_REQUESTED`)

1. Convertir `RateLimiter.Allow`+`RecordFailure` en una operació atòmica única sota un sol `Lock()` i actualitzar `PasswordAuthenticator.Login` per fer-la servir — owner: `fullstack-developer` — fixes: F-01
2. Afegir un test de concurrència real (goroutines concurrents contra la mateixa clau, mateix patró que `TestTwoUsers_ConcurrentMove_Repeated`) que falli de manera determinista si la finestra de carrera es reintrodueix — owner: `fullstack-developer` — fixes: F-01, F-02
3. (Opcional, no bloquejant) Derivar `csrfCookieMaxAgeSeconds` d'`auth.SessionTTL` en lloc de duplicar el valor — owner: `fullstack-developer` — fixes: F-03

## 8. Sign-off

> Pendent — no s'omple fins que el verdict sigui `APPROVED`.
