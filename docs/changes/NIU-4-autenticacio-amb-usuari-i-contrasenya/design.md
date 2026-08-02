---
artefact: design
key: "NIU-4"
title: "Autenticació amb usuari i contrasenya"
status: "approved"
owner: "software-architect"
requirements_path: "./requirements.md"
adr_count: 5
sources:
  - "arc42 (subset: §1 introduction, §4 solution strategy, §5 building blocks, §6 runtime, §8 cross-cutting, §11 risks)"
  - "ADR format (Michael Nygard, 2011)"
  - "C4 model — Levels 1 (context) and 2 (containers)"
created: "2026-08-02"
updated: "2026-08-02"
---

# Design — Autenticació amb usuari i contrasenya

> **Què és això.** La resposta tècnica als 14 AC / 12 EC / 9 NFR de
> `requirements.md`. Substitueix `auth.StubAuthenticator` (ADR-03 de
> [`../NIU-1-llista-de-la-compra-rebost-auth/design.md`](../NIU-1-llista-de-la-compra-rebost-auth/design.md))
> per una implementació real basada en contrasenya, sense tocar cap
> handler d'`items`. Refina `PLAN.md` §3 (Seguretat), §6 (Configuració) i
> §8 (abast NIU-4) sense contradir-los. Referència de projecte:
> [`../../architecture.md`](../../architecture.md). No hi ha Stage 1.5
> (visual) — decisió humana explícita per prioritzar el desplegament; la
> pantalla de login reutilitza `design-system/tokens.css` sense maqueta
> dedicada (§8 d'aquest document).

## 1. Introducció i restriccions (arc42 §1)

- **Objectiu d'aquest canvi:** substituir `StubAuthenticator` per
  `PasswordAuthenticator` (bcrypt + sessió opaca) darrere de la mateixa
  interfície `auth.Authenticator`, complint els 14 AC i 12 EC de
  `requirements.md` i les files S1/S2/S4/S5/S6/S9 de `PLAN.md` §3.
- **Restriccions (no negociables):**
  - Tècnica: `items_handlers.go` i `router.go` no canvien de forma —
    només la línia de wiring a `cmd/niu/main.go` que tria l'implementació
    d'`Authenticator` (ADR-03 NIU-1). Mateix binari únic, mateix SQLite,
    `modernc.org/sqlite`, `chi/v5`, `pressly/goose/v3` embedded. Nova
    dependència justificada: `golang.org/x/crypto/bcrypt` (no hi és
    encara a `go.mod`).
  - Organitzacional: repositori públic (S11) — cap hash real ni secret
    committejat; les credencials arriben per variables d'entorn a
    l'arrencada (`NIU_USER_A_HASH`, `NIU_USER_B_HASH`, `NIU_SESSION_SECRET`).
  - Temps/cost: NIU-4 bloqueja NIU-2 (`PLAN.md` §8) — sense Stage 1.5,
    sense maqueta dedicada de login; reutilitza tokens ja auditats a
    NIU-1.
  - Disseny: `design-system/tokens.css` és l'única font visual; la
    pantalla de login és un formulari mínim amb els mateixos tokens, no
    un component nou al sistema.
  - Seguretat: TTL de sessió = 30 dies (resolt amb l'humà,
    `requirements.md` §8). Els dos punts restants de §8 es resolen en
    aquest document (§2, ADR-02, ADR-04).

## 2. Estratègia de solució (arc42 §4)

1. **`PasswordAuthenticator` substitueix `StubAuthenticator` darrere de
   la mateixa interfície** (ADR-03 NIU-1, sense modificar-la) — llegeix
   la cookie de sessió, calcula `SHA-256(token)`, la busca a `sessions`,
   comprova `expires_at`, retorna `auth.User{ID}`. `httpapi` no canvia.
2. **Sessió = token opac de 256 bits (`crypto/rand`), mai JWT**
   (`PLAN.md` decision log) — es guarda `SHA-256(token)` a
   `sessions.token_hash`; el token en clar només existeix a la cookie del
   client i a la memòria del procés durant la petició de login (AC-08).
3. **Comparació de contrasenya sempre passa per bcrypt, també quan
   l'usuari no existeix** (ADR-02, resol S5/NFR-03) — s'evita la branca
   condicional que delataria per temporització si l'usuari existeix o no.
4. **CSRF de doble-submit sobre una cookie llegible per JS, separada de
   la cookie de sessió** (`HttpOnly`) — la sessió estructural ja depèn de
   `SameSite=Strict` + mateix origen (`PLAN.md` §2.1), el token CSRF és
   defensa en profunditat explícitament exigida per S1/AC-06/AC-07.
5. **Rate limiting en memòria del procés, no a SQLite** (ADR-01) —
   Niu és un sol procés sense escalat horitzontal (`PLAN.md` §2.1); un
   mapa protegit per mutex amb neteja periòdica evita escriptures SQLite
   addicionals en el camí calent de login i no sobreviu a un reinici,
   cosa acceptable per a l'amenaça que mitiga.
6. **Neteja de sessions expirades: goroutine en `ticker`, no neteja
   perezosa** (ADR-04) — un sol procés de llarga durada fa que un ticker
   sigui la solució més simple i predictible; la neteja perezosa
   obligaria cada lectura de sessió a pagar el cost d'un `DELETE`
   condicional.
7. **Validació d'entrada abans del rate limiter, rate limiter abans de
   bcrypt** (ADR-03, resol la segona pregunta oberta) — un payload sense
   usuari/contrasenya mai consumeix un intent del comptador ni toca
   bcrypt; només els intents amb tots dos camps presents compten.
8. **Seed de credencials via upsert a l'arrencada, no via nova migració**
   — `cmd/niu/main.go` llegeix `NIU_USER_A_HASH`/`NIU_USER_B_HASH` i fa
   `UPDATE users SET password_hash = ? WHERE name = ?` abans de servir
   trànsit; la migració 002 (placeholder) ja existeix i no es toca.
9. **`config.Load()` s'estén amb validació fail-fast dels tres secrets**
   (EC-12) — el procés es nega a arrencar si falta qualsevol dels quatre
   `NIU_USER_*_HASH`/`NIU_USER_*_NAME` o si `NIU_SESSION_SECRET` té menys
   de 32 bytes; mai arrenca a mig configurar.
10. **Frontend: una pantalla nova, cap canvi al `store.js`/`render.js`
    existents** — `login.html` separat d'`index.html`, `main.js` detecta
    401 a qualsevol crida i redirigeix a `/login.html?next=...`.

## 3. Decisions arquitectòniques (ADRs)

### ADR-01 — Rate limiting: en memòria del procés, llindars i backoff concrets

- **Status:** accepted
- **Context:** NFR-05/AC-10/EC-06/EC-07 exigeixen un mínim observable (≥10
  intents fallits/usuari en una finestra curta) però deixen el número
  exacte, la finestra, el llindar per IP i la forma del backoff com a
  decisió d'aquest document (`requirements.md` §8, pregunta 1). Niu és un
  sol procés sense escalat horitzontal (`PLAN.md` §2.1) — no cal
  coordinació entre instàncies.
- **Decision:** estructura en memòria `map[string]*bucket` protegida per
  `sync.Mutex`, amb **dues claus independents** que es consulten totes
  dues abans de tocar bcrypt:
  - **Per usuari:** clau = nom d'usuari normalitzat (mateix `TrimSpace` +
    `ToLower` que EC-06/ADR-02 de NIU-1, per no obrir una via de bypass
    amb variants de majúscules). Llindar: **10 intents fallits en una
    finestra mòbil de 5 minuts**. Compleix el mínim d'NFR-05 exactament.
  - **Per IP:** clau = `Cf-Connecting-Ip` si present (mateixa capçalera
    que `PLAN.md` §5.2 ja exigeix a Traefik per al rate limit
    d'infraestructura; sense ella es veurien IPs de vora de Cloudflare
    rotatòries), amb fallback a `r.RemoteAddr` en desenvolupament local.
    Llindar: **20 intents fallits en la mateixa finestra de 5 minuts**
    (més ample que el d'usuari perquè cobreix EC-06 — atac contra
    múltiples usuaris des d'una procedència — sense penalitzar
    prematurament un ús domèstic normal on dues persones comparteixen
    xarxa/NAT).
  - **Backoff fix, no exponencial:** un cop superat qualsevol dels dos
    llindars, els intents addicionals contra aquella clau es rebutgen
    amb `429` durant **la resta de la finestra de 5 minuts** (no
    backoff creixent per intent). Justificació: amb dues persones reals
    i sense atacants persistents esperats, un backoff exponencial afegeix
    complexitat d'implementació i de test sense guany observable — el
    llindar dur de 5 minuts ja fa inviable un atac de força bruta amb
    bcrypt cost 12 (10 intents cada 5 min ≈ 2.880 intents/dia, molt per
    sota del que cal per exhaurir un espai de contrasenyes raonable).
  - **Finestra mòbil implementada com a comptador amb `resetAt`:** cada
    `bucket` guarda `count int` i `windowStart time.Time`; si
    `time.Since(windowStart) > 5*time.Minute` en consultar, es reinicia
    `count=0, windowStart=now` abans de comptar l'intent actual (finestra
    fixa reiniciable, no un algorisme de finestra lliscant amb marques de
    temps per intent — suficient per al llindar exigit i molt més simple
    d'implementar i testejar).
  - **Neteja de buckets inactius:** el mateix ticker de neteja de
    sessions (ADR-04) esborra entrades del mapa amb
    `time.Since(windowStart) > 5*time.Minute` per evitar creixement
    il·limitat del mapa amb noms d'usuari/IPs que ja no ataquen.
- **Consequences:** (+) zero escriptures SQLite noves al camí calent de
  login; (+) llindars simples de testejar determinísticament (bucle de
  N intents, assert N+1 rebutjat, mateix patró que `security_test.go`);
  (+) compleix el mínim d'NFR-05 amb marge. (−) el comptador es perd en
  un reinici del procés — acceptable: un reinici no és un vector d'atac
  practicable per algú extern, i el TTL de sessió de 30 dies (resolt)
  no depèn d'aquest estat.
- **Alternatives considered:** taula SQLite `login_attempts` (rebutjat:
  escriptura extra per cada intent fallit en el camí que precisament vol
  ser barat de descartar; sense guany de durabilitat rellevant per a
  l'amenaça); backoff exponencial per intent (rebutjat: complexitat i
  superfície de test no justificades per dues persones i un llindar dur
  ja suficient); Redis/cau externa (rebutjat: infraestructura nova per a
  un sol procés, contradiu `PLAN.md` §2.1).

### ADR-02 — Resistència a enumeració d'usuaris (S5): bcrypt sempre, contra un hash simulat si l'usuari no existeix

- **Status:** accepted
- **Context:** AC-11/S5/NFR-03 exigeixen cos de resposta byte-idèntic
  **i** temps de resposta no distingible entre "usuari inexistent" i
  "contrasenya incorrecta". Una implementació ingènua retorna l'error
  abans de cridar bcrypt quan l'usuari no existeix, cosa que la fa
  desenes de mil·lisegons més ràpida que el cas amb bcrypt (cost 12,
  >200ms per NFR-01) — una diferència de temporització trivialment
  mesurable des de fora.
- **Decision:** `PasswordAuthenticator` precalcula a l'arrencada (un sol
  cop, no per petició) un **hash bcrypt fix d'una contrasenya dummy**
  (`dummyHash`, generat amb el mateix cost 12, embegut com a constant
  derivada — no és un secret, no protegeix res, només serveix per fer
  córrer `bcrypt.CompareHashAndPassword`). En cada intent de login:
  1. Es busca l'usuari per nom normalitzat.
  2. **Independentment del resultat de la cerca**, es crida
     `bcrypt.CompareHashAndPassword(hash, []byte(password))` exactament
     una vegada: `hash` és `user.PasswordHash` si l'usuari existeix, o
     `dummyHash` si no. El resultat de la comparació contra `dummyHash`
     **sempre és fals** (la contrasenya dummy mai coincideix amb res que
     un client enviï), però el cost computacional de bcrypt es paga
     igual en tots dos casos.
  3. Si l'usuari no existia **o** la comparació falla, es retorna
     exactament el mateix error (`AC-11`): mateix codi HTTP (401), mateix
     `code` (`"invalid_credentials"`), mateix `message`, sense cap camp
     addicional que reveli quin dels dos casos ha passat.
- **Consequences:** (+) el cost de CPU de la ruta és idèntic
  estructuralment (una crida a bcrypt sempre, mai zero); (+) cap branca
  condicional que salti bcrypt; (+) test S5 (temps + cos) verificable amb
  el patró ja existent. (−) un atac que mesuri temps a escala de
  microsegons amb milers de mostres **podria** en teoria distingir petites
  diferències (cerca SQL de l'usuari abans de la comparació); es considera
  fora d'abast per a una app domèstica sense exposició a aquest nivell
  d'atac (`requirements.md` NFR-03 ja demana "marge ampli, per evitar
  flakiness en CI", no resistència a atacs de canal lateral d'alta
  precisió).
- **Alternatives considered:** retornar l'error abans de bcrypt quan
  l'usuari no existeix (rebutjat: és exactament el forat de temporització
  que S5 prohibeix); `time.Sleep` compensatori fins a un llindar fix
  (rebutjat: afegeix un component més a mantenir sincronitzat amb el cost
  real de bcrypt, i és menys robust que fer-lo estructuralment idèntic).

### ADR-03 — Ordre del pipeline de login: validació d'entrada abans del rate limiter, rate limiter abans de bcrypt

- **Status:** accepted
- **Context:** EC-08 (contrasenya buida/absent) i EC-09 (usuari buit/absent)
  pregunten si consumeixen un intent del comptador de força bruta.
  `requirements.md` proposa que no, per evitar una via de denegació de
  servei trivial (bombardejar el comptador amb payloads buits en bucle);
  aquest document ho confirma i el fixa amb precisió d'implementació
  (`requirements.md` §8, pregunta 2).
- **Decision:** `handleLogin` executa, en aquest ordre estricte, cada pas
  només si el previ passa:
  1. **Decodificació JSON.** Si falla: `400 validation_failed`. No toca
     el rate limiter ni compta res.
  2. **Validació d'entrada pura** (sense tocar BD ni rate limiter):
     `username != ""` (després de `TrimSpace`) i `password != ""` (sense
     trim — un espai literal és una contrasenya vàlida en teoria, encara
     que cap de les dues seedejades ho sigui). Si falla qualsevol: `400
     validation_failed`, mateix missatge genèric per a tots dos camps
     absents (no cal diferenciar quin camp faltava — no és informació
     sensible, però simplifica el contracte). **Aquest pas és previ al
     rate limiter i mai el toca** — respon EC-08/EC-09 explícitament: un
     payload buit no consumeix cap intent, perquè mai arriba al pas 3.
  3. **Consulta del rate limiter** (ADR-01), per usuari normalitzat i per
     IP. Si qualsevol dels dos llindars ja està superat: `429
     rate_limited`, **sense** cridar bcrypt ni consultar `users` (AC-10:
     "es rebutgen immediatament, sense ni tan sols verificar la
     contrasenya").
  4. **Comparació de credencials** (ADR-02): cerca d'usuari + bcrypt
     sempre. Si falla: **es registra l'intent fallit al rate limiter**
     (incrementa el comptador d'usuari i d'IP), es retorna `401
     invalid_credentials` (cos AC-11).
  5. **Èxit:** es genera token nou (ADR de sessió, §5), **no** s'incrementa
     el comptador de força bruta (un èxit no és un intent fallit; si
     l'usuari havia acumulat intents fallits previs per sota del
     llindar, el comptador no es reinicia explícitament — expira sol amb
     la finestra de 5 minuts, sense necessitat de lògica addicional).
- **Consequences:** (+) EC-08/EC-09 resolts sense ambigüitat:
  un atacant que bombardegi amb payloads buits no exhaureix mai el
  llindar de força bruta legítim contra un usuari real (però sí que se'l
  pot rate-limitar a nivell de xarxa/Traefik si cal — fora d'abast
  d'aquest document, ja cobert per `PLAN.md` §5.2); (+) ordre explícit,
  sense marge d'interpretació per `fullstack-developer`. (−) un atacant
  podria enviar payloads buits il·limitadament sense ser bloquejat per
  aquest rate limiter (mitigat: la validació és O(1) i no toca bcrypt ni
  BD, el cost real per petició és menyspreable; el rate limit
  d'infraestructura de Traefik, `PLAN.md` §5.2, ja cobreix volum brut de
  peticions independentment del contingut).
- **Alternatives considered:** comptar el payload buit com a intent fallit
  (rebutjat explícitament: obre la via de DoS trivial que
  `requirements.md` ja identifica); validar dins del rate limiter mateix
  (rebutjat: barreja dues responsabilitats, fa més difícil testejar
  cadascuna per separat).

### ADR-04 — Neteja de sessions expirades: goroutine amb `ticker`

- **Status:** accepted
- **Context:** EC-10/NFR-08 exigeixen que les sessions expirades no
  s'acumulin indefinidament, sense exigir un termini concret. Niu és un
  procés únic de llarga durada (no serverless, no multi-instància).
- **Decision:** `cmd/niu/main.go` arrenca una goroutine de fons amb
  `time.NewTicker(1 * time.Hour)` que executa
  `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP` a cada tic,
  més una primera passada immediata a l'arrencada (abans del primer tic,
  per netejar sessions expirades durant una parada llarga del procés). La
  goroutine s'atura netament quan el `context.Context` arrel es cancel·la
  (mateix patró de shutdown que la resta de `main.go`). La mateixa
  goroutine neteja els `bucket` de rate limiting inactius (ADR-01) en cada
  tic, reutilitzant el mateix mutex/estructura sense una segona goroutine.
- **Consequences:** (+) simple, predictible, cap dependència externa
  (`cron`, `pg_cron` no aplica a SQLite); (+) una hora és un termini molt
  per sota de "hores-dies" que exigeix NFR-08, i molt per sota de l'escala
  del TTL de 30 dies, així que mai hi ha una acumulació significativa;
  (+) reaprofita l'única connexió `*sql.DB` ja oberta, sense pool nou.
  (−) fins a 1h de sessions expirades poden restar visibles a una
  inspecció directa de la taula (acceptable: EC-10 exigeix que "no
  s'acumulin indefinidament", no un termini immediat; el test EC-10
  sembra una sessió amb `expires_at` en el passat i **força** l'execució
  del procés de neteja directament, no espera el ticker real — vegeu §6).
- **Alternatives considered:** neteja perezosa a cada lectura de sessió
  (rebutjat: cada peticiódel camí calent pagaria potencialment un `DELETE`
  addicional; barreja responsabilitats de lectura i manteniment); neteja
  només a l'arrencada (rebutjat: un procés de llarga durada acumularia
  sessions expirades durant setmanes entre reinicis, incomplint
  l'esperit d'NFR-08); `cron` extern al contenidor (rebutjat: Niu és
  un sol binari sense procés supervisor extra, `PLAN.md` §2.1).

### ADR-05 — CSRF de doble-submit: cookie separada llegible per JS, verificada contra capçalera

- **Status:** accepted
- **Context:** AC-06/AC-07/EC-03/S1 exigeixen un token CSRF lligat a la
  sessió concreta (no un secret global reutilitzable), amb un mecanisme
  concret de lliurament al client que `proposal.md`/`requirements.md`
  deixen obert a aquest document.
- **Decision:** en cada login amb èxit, a més de la cookie de sessió
  (`HttpOnly`), el servidor genera un **segon valor aleatori** de 128
  bits (`crypto/rand`) i el desa en una **cookie separada, NO
  `HttpOnly`**: `niu_csrf`, atributs `Secure; Path=/; SameSite=Strict`
  (sense `HttpOnly` — el JavaScript del client la llegeix explícitament).
  Aquest valor **no** es persisteix a `sessions` ni a cap altra taula: es
  deriva determinísticament a partir del `token_hash` de la sessió
  (`HMAC-SHA256(NIU_SESSION_SECRET, token_hash)`, codificat en base64
  URL-safe) — així el servidor el pot **recalcular i verificar** sense
  cap emmagatzematge addicional, i queda lligat criptogràficament a la
  sessió concreta (no és un secret global: canvia amb cada sessió,
  compleix EC-03). `main.js` llegeix `niu_csrf` de `document.cookie` en
  cada petició de mutació i l'envia a la capçalera `X-CSRF-Token`. Un
  middleware `RequireCSRF` (aplicat només a `POST`/`PATCH`/`DELETE` sota
  `/api/v1/`, mai a `GET`) recalcula l'HMAC esperat a partir del
  `token_hash` de la sessió ja resolta pel middleware d'auth i el compara
  en temps constant (`hmac.Equal`) contra la capçalera rebuda; qualsevol
  discrepància (absent, buida, no coincident) és `403 csrf_failed`.
- **Consequences:** (+) zero taula nova, zero escriptura addicional —
  el valor es deriva, no es guarda; (+) lligat a la sessió (EC-03: un
  atacant que endevini/conegui un token plausible d'una altra sessió mai
  coincideix, perquè l'HMAC depèn del `token_hash` real); (+) patró
  estàndard de doble-submit, fàcil de testejar (`security_test.go`
  pattern). (−) si `NIU_SESSION_SECRET` roté en calent, tots els tokens
  CSRF derivats de sessions existents deixarien de validar-se
  (acceptable: la rotació en calent és explícitament fora d'abast,
  `requirements.md` §7 / `proposal.md` §6).
- **Alternatives considered:** token CSRF aleatori independent persistit
  a `sessions` (rebutjat: columna nova + escriptura extra per a un valor
  que un HMAC derivat aconsegueix sense estat); secret CSRF global
  compartit per totes les sessions (rebutjat explícitament per EC-03: no
  lligat a la sessió, exactament l'anti-patró que el requisit prohibeix).

## 4. Building blocks (arc42 §5 + C4 Nivell 2)

> Només els components que aquest canvi toca — la resta d'NIU-1
> (`internal/items`, `internal/store` per a ítems, `web/js/store.js` i
> derivats) no canvia de forma i no es repeteix aquí.

```text
┌────────────────────────────────────────────────────────────────┐
│                     cmd/niu/main.go                              │
│  (afegeix: seed de credencials via env, goroutine de neteja,     │
│   tria PasswordAuthenticator en lloc de StubAuthenticator)       │
└────────────────────────────────────────────────────────────────┘
        │                          │                     │
        ▼                          ▼                     ▼
┌──────────────────┐      ┌──────────────────┐   ┌──────────────────┐
│ internal/auth     │      │ internal/httpapi  │   │ internal/config   │
│ (NOU en NIU-4)    │      │ (afegit, no       │   │ (estès: valida    │
│ PasswordAuthentic-│◀─────│  modificat)       │   │ els 5 secrets     │
│ ator implementa   │ crida│ auth_handlers.go  │   │ nous, fail-fast)  │
│ Authenticator     │      │ (login/logout),   │   └──────────────────┘
│ (interfície ja    │      │ RequireCSRF mw,   │
│ existent, ADR-03  │      │ WithCurrentUser   │
│ NIU-1)            │      │ ja resol via la   │
│                   │      │ nova implementació│
│ RateLimiter       │      └──────────────────┘
│ (mapa en memòria) │               │
│ SessionStore      │               ▼
│ (crea/valida/neteja)      ┌──────────────────┐
└──────────┬────────┘       │ internal/items    │
           │                │ handlers — ZERO   │
           ▼                │ canvis (ADR-03)   │
    ┌─────────────┐         └──────────────────┘
    │ SQLite: users│
    │  + sessions  │
    │ (esquema ja  │
    │  existent,   │
    │  NIU-1 mig.1)│
    └─────────────┘
```

- **`internal/auth` (nou codi, interfície existent)** — afegeix
  `PasswordAuthenticator` (implementa `Authenticator.CurrentUser` llegint
  la cookie de sessió i validant-la contra `sessions`), `RateLimiter`
  (mapa en memòria, mètodes `Allow(key) bool` i `RecordFailure(key)`),
  i les funcions de gestió de sessió: `CreateSession(userID) (token
  string, err error)`, `Login(username, password string) (token string,
  err error)` (orquestra ADR-02/ADR-03), `Logout(token string) error`,
  `CleanupExpired(ctx) error` (ADR-04). **No** importa `net/http` per a
  la lògica de negoci — rep i retorna tipus propis; `httpapi` és qui
  llegeix/escriu la cookie.
- **`internal/httpapi`** — afegeix `auth_handlers.go`
  (`handleLogin`, `handleLogout`) i `RequireCSRF` (nou middleware,
  ADR-05), muntat només a les rutes de mutació sota `/api/v1/` (no a
  `/api/v1/auth/login` — encara no hi ha sessió; no a `GET`). **Modifica
  `router.go`** només per registrar les dues rutes noves
  (`POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, totes dues
  **fora** de `WithCurrentUser` per a login —abans no hi ha sessió— però
  `logout` sí que hi passa per resoldre quina sessió invalidar) i aplicar
  `RequireCSRF` al grup de rutes de mutació existent. **`items_handlers.go`
  no es toca** (ADR-03 NIU-1 confirmat).
- **`internal/config`** — s'estén (no es reescriu) amb els cinc camps
  nous i la seva validació fail-fast: `SessionSecret`,
  `UserAName/UserADisplay/UserAHash`, `UserBName/UserBDisplay/UserBHash`.
- **`cmd/niu/main.go`** — afegeix: (1) upsert de credencials a
  l'arrencada des de `config`, (2) construcció de
  `auth.NewPasswordAuthenticator(store.DB, cfg.SessionSecret)` en lloc
  d'`auth.StubAuthenticator{}`, (3) llançament de la goroutine de neteja
  (ADR-04) amb el `context.Context` de shutdown ja existent al procés.
- **`web/login.html` + `web/js/auth.js` (nous)** — pantalla de login
  separada, reutilitza `app.css`/`tokens.css` sense maqueta dedicada
  (§8).

## 5. Vista d'execució (arc42 §6)

**Flux 1 — `POST /api/v1/auth/login`, camí d'èxit (AC-01):**

1. Client envia `{"username": "usuari_a", "password": "..."}` sense
   cookies prèvies necessàries.
2. `handleLogin` decodifica el JSON (pas 1, ADR-03). Si falla: `400`.
3. Valida `username`/`password` no buits (pas 2, ADR-03). Si falla:
   `400 validation_failed`, **rate limiter no tocat**.
4. Consulta `RateLimiter.Allow(username_normalitzat)` i
   `RateLimiter.Allow(ip)` (pas 3). Si algun ja excedeix el llindar:
   `429 rate_limited`, **bcrypt mai cridat**.
5. `auth.Login(username, password)`: cerca l'usuari per nom normalitzat;
   crida bcrypt sempre, contra `user.PasswordHash` o `dummyHash` (ADR-02).
6. **Contrasenya incorrecta o usuari inexistent:**
   `RateLimiter.RecordFailure(username_normalitzat)` +
   `RateLimiter.RecordFailure(ip)`; resposta `401
   {"error":{"code":"invalid_credentials","message":"Usuari o contrasenya
   incorrectes."}}` — cos idèntic en tots dos casos (AC-11).
7. **Èxit:** genera 256 bits amb `crypto/rand`, calcula
   `SHA-256(token)`, `INSERT INTO sessions (token_hash, user_id,
   expires_at) VALUES (?, ?, CURRENT_TIMESTAMP + 30 days)`. Calcula el
   token CSRF derivat (ADR-05). Respon `200
   {"user": {"id","display_name","avatar_emoji"}}` amb dues cookies
   `Set-Cookie`: `niu_session=<token>; HttpOnly; Secure; Path=/;
   SameSite=Strict; Max-Age=2592000` i `niu_csrf=<hmac>; Secure; Path=/;
   SameSite=Strict; Max-Age=2592000` (sense `HttpOnly`).

**Flux 2 — `POST /api/v1/auth/login`, camins de fallada i força bruta
(AC-02/AC-03/AC-10/EC-06/EC-07/EC-08/EC-09):**

- Usuari inexistent (AC-03) i contrasenya incorrecta (AC-02) segueixen
  exactament els passos 1–6 del Flux 1 i produeixen el mateix cos `401`
  (AC-11) — no hi ha camí de codi separat entre els dos casos més enllà
  de quin `hash` s'usa a bcrypt (ADR-02).
- Onzè intent fallit contra el mateix usuari en <5 min (AC-10): pas 4
  rebutja abans d'arribar al pas 5 — `429`, bcrypt no cridat, verificable
  perquè el temps de resposta d'un `429` és molt inferior al d'un `401`
  amb bcrypt.
- Ratxa contra usuaris diferents des de la mateixa IP (EC-06): el
  llindar per IP (20/5min) actua encara que cada usuari individual
  estigui per sota del seu propi llindar de 10.
- Ratxa contra el mateix usuari des d'IPs diferents (EC-07): el llindar
  per usuari (10/5min) actua encara que cada IP estigui per sota del
  seu propi llindar de 20.
- Contrasenya buida/absent (EC-08) o usuari buit/absent (EC-09): pas 2
  rebutja amb `400` abans del pas 3 — el rate limiter mai veu l'intent.

**Flux 3 — `POST /api/v1/auth/logout` (AC-04/AC-09/EC-05):**

1. Petició inclou la cookie `niu_session` (i, si `RequireCSRF` s'aplica
   també a logout —decisió: **sí**, és una mutació— la cookie/capçalera
   CSRF).
2. `WithCurrentUser` (ja existent) resol la sessió activa via
   `PasswordAuthenticator.CurrentUser` — si no hi ha sessió vàlida,
   `401` abans d'arribar al handler (mateix camí que qualsevol altre
   endpoint protegit).
3. `handleLogout` crida `auth.Logout(token)`:
   `DELETE FROM sessions WHERE token_hash = ?`.
4. Respon `204`. El client esborra el seu estat local i redirigeix a
   `/login.html`. Qualsevol petició posterior amb la cookie antiga
   (encara present al navegador fins que expiri) es rebutja amb `401`
   perquè la fila ja no existeix a `sessions` (EC-05/EC-11).

**Flux 4 — CSRF de doble-submit en una mutació (AC-06/AC-07/EC-03):**

1. Client ja té `niu_session` (HttpOnly) i `niu_csrf` (llegible) d'un
   login previ.
2. `main.js`/`api.js` llegeix `niu_csrf` de `document.cookie` abans de
   qualsevol `POST`/`PATCH`/`DELETE` i l'envia com a capçalera
   `X-CSRF-Token`.
3. `WithCurrentUser` resol la sessió (si no n'hi ha: `401`, `RequireCSRF`
   ni s'executa).
4. `RequireCSRF` recalcula `HMAC-SHA256(NIU_SESSION_SECRET,
   token_hash_de_la_sessió_resolta)` i el compara amb `hmac.Equal`
   contra `X-CSRF-Token`. Coincident → continua cap al handler
   d'`items` (AC-06). No coincident/absent → `403 csrf_failed`, handler
   d'`items` mai s'executa (AC-07) — verificable perquè un `GET`
   posterior no mostra cap efecte.
5. Un atacant que only conegui/endevini un valor CSRF plausible d'una
   altra sessió (EC-03) falla perquè l'HMAC és funció del `token_hash`
   real de la sessió de la víctima, no d'un secret global.

## 6. Contractes i model de dades

### 6.1 API

| Endpoint | Mètode | Petició | Resposta |
|---|---|---|---|
| `/api/v1/auth/login` | `POST` | `{ "username": string, "password": string }` | `200 { "user": { "id", "display_name", "avatar_emoji" } }` + `Set-Cookie` ×2 (`niu_session`, `niu_csrf`) \| `400 validation_failed` \| `401 invalid_credentials` \| `429 rate_limited` |
| `/api/v1/auth/logout` | `POST` | — (cookies + `X-CSRF-Token`) | `204` \| `401` (sense sessió vàlida) |
| `/api/v1/items`, `/api/v1/me`, etc. (existents) | — | — | Ara reben `401` si no hi ha sessió vàlida (abans sempre autenticaven via stub); `403 csrf_failed` a mutacions sense token vàlid |

**Cossos d'error nous (mateix envelope que NIU-1, `apiError`):**

```json
{ "error": { "code": "invalid_credentials", "message": "Usuari o contrasenya incorrectes." } }
{ "error": { "code": "rate_limited", "message": "Massa intents. Torna-ho a provar més tard." } }
{ "error": { "code": "csrf_failed", "message": "Petició no vàlida." } }
{ "error": { "code": "validation_failed", "message": "Cal indicar usuari i contrasenya." } }
```

`401` sense sessió (endpoints protegits existents, AC-05) reutilitza
`code: "unauthenticated"`, ja que abans mai calia distingir-ho (stub
sempre autenticava). Cos mínim, mai inclou dades protegides (AC-05).

**Cookies (binding, S2/S6):**

| Cookie | Atributs | Contingut | Llegible per JS |
|---|---|---|---|
| `niu_session` | `HttpOnly; Secure; Path=/; SameSite=Strict; Max-Age=2592000` | token opac 256 bits, base64 URL-safe | No |
| `niu_csrf` | `Secure; Path=/; SameSite=Strict; Max-Age=2592000` | HMAC derivat, base64 URL-safe | Sí (expressament) |

`Max-Age=2592000` = 30 dies (TTL resolt, `requirements.md` §8). En
`NIU_ENV=development`, `Secure` s'omet (ja establert com a comportament
existent per a localhost sense TLS, `PLAN.md` §6) — la resta d'atributs
es mantenen.

### 6.2 Model de dades (deltes sobre l'esquema existent, NIU-1 migració 1)

| Entitat | Canvi | Risc de migració |
|---|---|---|
| `sessions` | **Cap canvi d'esquema.** `token_hash`/`user_id`/`created_at`/`expires_at` ja existeixen (NIU-1 migració 1) i cobreixen exactament el que calia. | — |
| `users` | **Cap canvi d'esquema.** `password_hash` ja existeix (NIU-1 migració 1, placeholder). Es reemplaça el **valor** de les dues files via `UPDATE` a l'arrencada, no via migració nova. | LOW — `UPDATE` idempotent sobre `name` conegut |
| Força bruta | **Cap taula nova.** En memòria (ADR-01), no persistit. | — |

**No cal cap migració `003_*.sql` per a NIU-4.** L'esquema de NIU-1 ja
va anticipar `sessions`/`users.password_hash` exactament per a aquest
moment (`PLAN.md` §2.4: "`users.password_hash` exists from the first
migration even though NIU-4 is last"). L'única escriptura nova a
l'arrencada és l'upsert de credencials:

```sql
UPDATE users SET password_hash = ? WHERE name = ?;
```

executat dues vegades (Usuari A, Usuari B) dins d'una transacció, amb
`fullstack-developer` verificant `RowsAffected == 1` per cada `UPDATE` —
si un nom no coincideix amb cap fila existent, el procés falla
l'arrencada (coherent amb el fail-fast d'EC-12: la migració 002 ja va
sembrar `usuari_a`/`usuari_b`, així que un mismatch de nom és un error
de configuració real, no un cas esperat).

## 7. Arquitectura de frontend (nous fitxers a `app/web/`)

- **`web/login.html`** — document separat d'`index.html`, mateix
  `<head>` (fonts self-hosted, `app.css`), formulari mínim: camp
  `username`, camp `password` (`type="password"`), botó de submit. Sense
  maqueta dedicada (Stage 1.5 omesa) — s'usen directament les classes de
  `component-add-input.html` del design-system per al camp de text i un
  botó primari existent, sense inventar cap component visual nou.
- **`web/js/auth.js` (nou)** — `login(username, password)` crida
  `fetch('/api/v1/auth/login', {method:'POST', credentials:'same-origin',
  body: JSON.stringify({username,password})})`; en `200`, llegeix el
  paràmetre `?next=` de la URL (per defecte `/`) i hi redirigeix
  (`window.location.href`); en `401`/`429`/`400`, mostra el `message` del
  cos d'error sota el formulari (mateix patró de `Toast`/missatge
  d'error inline ja existent al design-system, sense component nou).
  `logout()` crida `POST /api/v1/auth/logout` amb la capçalera CSRF
  (llegida de `document.cookie`) i redirigeix a `/login.html`.
- **`web/js/api.js` (existent, s'estén)** — cada wrapper de mutació
  (`addItem`, `moveItem`, `deleteItem`) ha d'incloure la capçalera
  `X-CSRF-Token` llegida de la cookie `niu_csrf` (funció nova
  `getCsrfToken()` que fa `document.cookie.split(...)`, sense llibreria
  externa). Cada wrapper (mutació o lectura) que rebi `401` crida un nou
  gestor centralitzat `handleUnauthenticated()` que redirigeix a
  `/login.html?next=<ruta actual>` — així **qualsevol** crida existent
  (`getItems`, `getMe`, `addItem`, ...) queda coberta sense duplicar la
  lògica de redirecció a cada punt de crida.
- **`web/js/main.js` (existent, s'estén)** — a l'arrencada, abans de
  muntar la UI de la llista, crida `getMe()`; si respon `401`,
  redirigeix immediatament a `/login.html?next=/` sense arribar a
  renderitzar cap fila (evita el "parpelleig" d'una llista buida abans
  del redirect).
- **`router.go` (`internal/httpapi`)** — `web/login.html` es serveix pel
  mateix `http.FileServer` sobre `embed.FS` ja existent (cap canvi de
  mecanisme, és un fitxer estàtic més sota `web/`).

## 8. Concerns transversals (arc42 §8)

- **Seguretat:** vegeu §3 (ADR-01 a ADR-05) i §9 addicional més avall.
  Cobreix S1/S2/S4/S5/S6/S9 de `PLAN.md` §3.
- **Observabilitat (NFR-07):** cada intent de login (èxit, fallada,
  limitat) es registra amb `log/slog` des de `handleLogin`:
  `slog.Info("login attempt", "username", usernameNormalitzat, "result",
  "success"|"failure"|"rate_limited", "ip", ip)`. **Mai** el valor de
  `password` ni el token de sessió en clar — verificat capturant
  l'output de `slog` als tests, mateix mecanisme que NIU-1. OTEL
  (spans/traces) és fora d'abast (NIU-3); aquest logging bàsic n'és el
  precedent que NIU-3 instrumentarà.
- **Rendiment (NFR-06):** el cost dominant és bcrypt (>200ms per
  disseny, NFR-01) — s'accepta explícitament perquè NFR-06 fixa el
  llindar a <1s p95, amb marge ampli. Cap consulta N+1: el login és una
  sola `SELECT` d'usuari + una comparació bcrypt + un `INSERT` de sessió.
- **Resiliència:** una fallada de `RateLimiter` (mai — és en memòria,
  no pot fallar per xarxa/BD) no aplica; una fallada de `sessions`
  `INSERT` (BD ocupada, `busy_timeout` de NIU-1 ja cobreix fins a 5s)
  cau al mateix `internal_error` genèric ja establert. Reinici del
  procés esborra el rate limiter (acceptable, ADR-01) però no les
  sessions (persistents a SQLite) ni el TTL de 30 dies.
- **Compliance i privacitat:** NFR-09/S11 — cap credencial real, hash
  real ni nom real en cap fitxer committejat, incloent-hi fixtures de
  test (que usen contrasenyes/hash inventats per al test, mai reals) i
  el mateix `login.html` (sense noms reals al placeholder del camp
  `username`).
- **Accessibilitat (si UI):** el formulari de login segueix el mateix
  nivell WCAG 2.2 AA ja exigit a NIU-1 — etiquetes `<label>` associades
  (`for`/`id`), navegació completa per teclat (`Tab` entre camps i
  submit, `Enter` envia el formulari), missatge d'error associat amb
  `aria-describedby` al camp corresponent, contrast AA amb els mateixos
  tokens ja auditats. Sense maqueta dedicada, però **no** sense
  accessibilitat — són requisits transversals, no visuals.
- **i18n/l10n:** cap canvi — UI en català fix, mateix que NIU-1.

## 9. Riscos (arc42 §11)

| ID | Risc | Severitat | Mitigació | Owner |
|---|---|---|---|---|
| R-01 | Un atacant intenta força bruta abans que el rate limit s'apliqui prou d'hora | MEDIUM (heretat de `proposal.md` §7) | ADR-01: llindars durs (10/usuari, 20/IP en 5 min) + bcrypt cost 12; verificat per S4/EC-06/EC-07 | `fullstack-developer` |
| R-02 | Diferència de temps observable entre usuari inexistent i contrasenya incorrecta | MEDIUM (heretat) | ADR-02: bcrypt sempre, contra hash dummy si cal; verificat per S5 (marge ampli) | `fullstack-developer` |
| R-03 | Rate limiter en memòria es perd en cada reinici del procés (deploy, crash) | LOW | Acceptat explícitament a ADR-01 — l'amenaça mitigada (força bruta sostinguda) no es beneficia d'un reinici puntual; TTL de sessió (30 dies) no en depèn | `software-architect` (acceptat) |
| R-04 | Nova dependència `golang.org/x/crypto/bcrypt` introdueix una superfície nova a `go.sum` | LOW | Llibreria oficial de l'ecosistema Go (`golang.org/x/...`), mateix nivell de confiança que `golang.org/x/text` ja usat a NIU-1 | `fullstack-developer` |
| R-05 | Upsert de credencials a l'arrencada falla silenciosament si el nom d'usuari a l'env no coincideix amb el seedejat | MEDIUM | §6.2: `fullstack-developer` verifica `RowsAffected == 1` per `UPDATE` i falla l'arrencada (fail-fast) si no — cobreix EC-12 | `fullstack-developer` |
| R-06 | Cookie `niu_csrf` llegible per JS és, per disseny, accessible a qualsevol script que corri en el mateix origen (p. ex. via un XSS no detectat) | LOW | Defensa en profunditat ja existent: CSP sense `unsafe-inline` + disciplina `textContent` (S3, NIU-1) fa un XSS ja molt improbable; el doble-submit no pretén mitigar XSS, només CSRF — són amenaces diferents | `security-engineer` (a `/audit`) |
| R-07 | `RequireCSRF` aplicat incorrectament a `POST /api/v1/auth/login` bloquejaria el primer login (encara no hi ha token CSRF) | LOW | §4/§5 ho fixen explícitament: `RequireCSRF` només es munta a mutacions d'`/api/v1/items` i a `/api/v1/auth/logout`, mai a `/api/v1/auth/login` | `fullstack-developer` |

## 10. Preguntes obertes per a la porta humana

- Cap pregunta funcional pendent (heretat de `requirements.md` §8).
- Les dues preguntes escalades a Stage 2 (llindars de rate limiting i
  tractament d'EC-08) queden **resoltes** a ADR-01 i ADR-03
  respectivament — es couen sense conflicte amb el mínim observable
  d'`requirements.md`. Confirmació humana no bloquejant, tal com ja
  indicava `requirements.md` §8.
- **Confirmar el mecanisme CSRF triat (ADR-05, HMAC derivat sense
  emmagatzematge).** És una decisió d'implementació sense impacte
  funcional observable (AC-06/AC-07 es compleixen igual), però si el
  propietari humà prefereix un token CSRF aleatori persistit en lloc de
  derivat per HMAC (per exemple, per poder revocar-lo independentment de
  la sessió), cal dir-ho abans de `task-planner` — canviaria l'esquema
  (columna nova a `sessions` o taula pròpia).
