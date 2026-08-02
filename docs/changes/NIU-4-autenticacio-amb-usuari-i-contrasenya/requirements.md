---
artefact: requirements
key: "NIU-4"
title: "Autenticació amb usuari i contrasenya"
status: "approved"
owner: "product-manager + qa-engineer"
proposal_path: "./proposal.md"
ac_count: 14
nfr_count: 9
sources:
  - "User Story format (Mike Cohn) — As a / I want / So that"
  - "INVEST — Independent, Negotiable, Valuable, Estimable, Small, Testable"
  - "Given/When/Then — Gherkin / Cucumber"
created: "2026-08-02"
updated: "2026-08-02"
---

# Requirements — Autenticació amb usuari i contrasenya

> **Què és això.** El contracte entre producte i enginyeria per a NIU-4.
> Cada criteri d'acceptació és observable des de fora del sistema i es
> traça a almenys una tasca de `tasks.md`. **Només comportament
> funcional — cap detall d'implementació.** Referència: [`./proposal.md`](./proposal.md).
> Font vinculant: `PLAN.md` §3 (Seguretat — taula S1–S11), §6
> (Configuració) i §8 (abast de NIU-4).

## 1. Historial d'usuari

- **Com a** membre de la llar (Usuari A o Usuari B)
- **Vull** iniciar sessió amb un nom d'usuari i una contrasenya, i poder
  tancar-la explícitament
- **Perquè** només les dues persones de la casa puguin veure i modificar
  la llista de la compra un cop l'aplicació es publiqui a internet, sense
  dependre de cap servei extern de tercers

## 2. Autoavaluació INVEST

- [x] **Independent** — ✅ Depèn únicament de l'esquema `users`/`sessions` ja creat a NIU-1 (migració 1) i del seam `auth.Authenticator` (ADR-03); no bloqueja ni és bloquejat per NIU-2/NIU-3.
- [x] **Negotiable** — ✅ El mecanisme (cookie de sessió opaca + bcrypt) està fixat a `PLAN.md` §3, però el detall d'UI del formulari de login és obert a `ux-ui-designer`.
- [x] **Valuable** — ✅ És el requisit bloquejant per al desplegament públic (NIU-2): sense NIU-4 no hi ha manera segura d'exposar l'app a internet.
- [x] **Estimable** — ✅ Superfície acotada: dos endpoints, un middleware, una pantalla.
- [x] **Small** — ✅ Cap a un sol ítem de backlog, sense registre ni gestió d'usuaris.
- [x] **Testable** — ✅ Tots els AC són observables via HTTP (codis d'estat, capçaleres, cossos de resposta) o via inspecció de la base de dades des de fora del procés de l'app, seguint el mateix patró que `security_test.go` de NIU-1.

## 3. Criteris d'acceptació

### AC-01 — Login amb credencials correctes obre sessió

- **Given** un usuari sembrat (`Usuari A` o `Usuari B`) amb una contrasenya coneguda
- **When** envia el seu nom d'usuari i la contrasenya correcta a l'endpoint de login
- **Then** rep una resposta d'èxit, se li estableix una cookie de sessió amb els atributs `HttpOnly; Secure; Path=/; SameSite=Strict`, i les crides posteriors a endpoints protegits amb aquesta cookie es tracten com a autenticades

### AC-02 — Login amb contrasenya incorrecta és rebutjat

- **Given** un usuari sembrat existent
- **When** envia el seu nom d'usuari amb una contrasenya incorrecta
- **Then** rep un error d'autenticació, no se li estableix cap cookie de sessió, i cap crida posterior sense credencials noves es tracta com a autenticada

### AC-03 — Login amb usuari inexistent és rebutjat

- **Given** un nom d'usuari que no existeix a l'aplicació
- **When** s'intenta iniciar sessió amb qualsevol contrasenya
- **Then** rep un error d'autenticació equivalent al d'AC-02 (vegeu AC-11/S5 per als detalls d'equivalència exigits)

### AC-04 — Logout tanca la sessió activa

- **Given** un usuari amb una sessió activa (cookie vàlida)
- **When** crida l'endpoint de logout
- **Then** la sessió es marca com a invàlida al servidor, i qualsevol crida posterior a un endpoint protegit amb aquella mateixa cookie es rebutja com a no autenticada

### AC-05 — Accés a un endpoint protegit sense sessió es rebutja

- **Given** cap cookie de sessió present a la petició
- **When** es crida qualsevol endpoint que requereixi autenticació (p. ex. `/api/v1/items`, `/api/v1/me`)
- **Then** la petició es rebutja amb un error d'autenticació, sense exposar cap dada protegida al cos de la resposta

### AC-06 — Mutacions requereixen token CSRF de doble-submit

- **Given** un usuari amb sessió activa
- **When** envia una petició de mutació (`POST`, `PATCH`, `DELETE`) incloent el token CSRF esperat segons el patró de doble-submit
- **Then** la petició es processa amb normalitat (subjecta a la resta de validacions de negoci)

### AC-07 — Mutació sense token CSRF es rebutja

- **Given** un usuari amb sessió activa (cookie vàlida)
- **When** envia una petició de mutació sense el token CSRF, o amb un token que no coincideix
- **Then** la petició es rebutja amb un error d'autorització i no produeix cap efecte al sistema

### AC-08 — El token de sessió mai és recuperable en clar des de la base de dades

- **Given** una sessió activa creada per un login recent
- **When** s'inspecciona l'emmagatzematge de sessions de l'aplicació
- **Then** no hi ha cap valor emmagatzemat igual al token que el client té a la cookie — només un valor derivat (hash) que no permet reconstruir el token original

### AC-09 — Cada login emet un token nou; el logout l'invalida al servidor

- **Given** un mateix usuari que inicia sessió dues vegades (per exemple, després d'un logout previ, o des de dos clients)
- **When** es comparen els tokens de sessió emesos en cada login
- **Then** són diferents entre ells; i un cop es fa logout amb un d'ells, reutilitzar aquell token en una petició posterior es rebutja com a no autenticat

### AC-10 — Força bruta contra el login es limita

- **Given** intents repetits de login fallits contra el mateix usuari o des de la mateixa procedència
- **When** el nombre d'intents supera el llindar configurat
- **Then** els intents addicionals es rebutgen immediatament (sense ni tan sols verificar la contrasenya) fins que la limitació s'aixequi

### AC-11 — Resposta d'error de login no distingeix usuari inexistent de contrasenya incorrecta

- **Given** dos intents de login fallits: un contra un usuari que no existeix, un altre contra un usuari que existeix amb la contrasenya incorrecta
- **When** es comparen els cossos de resposta d'error de tots dos intents
- **Then** són idèntics byte a byte (mateix codi, mateix missatge, mateixa estructura)

### AC-12 — Sessió expira i deixa de ser vàlida

- **Given** una sessió creada amb un moment d'expiració
- **When** es fa una petició autenticada després que aquell moment d'expiració hagi passat
- **Then** la petició es rebutja com a no autenticada, encara que la cookie encara existeixi al client

### AC-13 — Credencials es sembren des de configuració, no des de codi ni base de dades committejada

- **Given** les variables d'entorn `NIU_USER_A_HASH` i `NIU_USER_B_HASH` (i homòlegs de nom/visualització) configurades a l'arrencada
- **When** l'aplicació arrenca
- **Then** els dos usuaris poden iniciar sessió amb les credencials corresponents a aquells hashos, sense que cap contrasenya en clar ni hash real aparegui en cap fitxer del repositori

### AC-14 — Cicle complet login → ús → logout

- **Given** una persona amb credencials vàlides i cap sessió activa
- **When** inicia sessió, realitza almenys una acció protegida (p. ex. llistar ítems), i tanca la sessió
- **Then** cada pas produeix el resultat esperat (AC-01, acció exitosa, AC-04) sense cap error inesperat, completant el flux end-to-end exigit per `PLAN.md` §8 com a condició de "fet"

## 4. Edge cases i escenaris negatius

### EC-01 — Cookie de sessió manipulada es rebutja (S2)

- **Given** una cookie de sessió amb el valor alterat respecte a l'emès pel servidor
- **When** es crida un endpoint protegit amb aquesta cookie
- **Then** la petició es rebutja com a no autenticada (mateix tractament que EC-02, sense diferència observable de temps o missatge que reveli si el token era "gairebé vàlid")

### EC-02 — Petició sense cap cookie es rebutja (S2)

- **Given** cap capçalera `Cookie` present a la petició
- **When** es crida un endpoint protegit
- **Then** la petició es rebutja com a no autenticada

### EC-03 — Falsificació del token CSRF a partir d'un valor conegut es rebutja (S1)

- **Given** un atacant que coneix o endevina un valor de token CSRF plausible però no el que el servidor ha emès per a la sessió de la víctima
- **When** l'intenta servir-lo en una mutació contra la sessió de la víctima
- **Then** la petició es rebutja — el token CSRF ha d'estar lligat a la sessió concreta, no ser un secret global reutilitzable

### EC-04 — Reutilització d'un token de sessió previ a un login (fixació de sessió) (S6)

- **Given** un atacant que ha obtingut o predit un token de sessió abans que la víctima iniciï sessió
- **When** la víctima completa el login
- **Then** el token vàlid després del login és diferent del que l'atacant coneixia — el token pre-login queda inservible

### EC-05 — Reutilització del token després de logout (S6)

- **Given** una sessió tancada explícitament (logout)
- **When** s'intenta una petició protegida amb el token de la sessió tancada
- **Then** es rebutja, encara que el moment d'expiració original encara no hagi arribat

### EC-06 — Ratxa d'intents fallits contra usuaris diferents des de la mateixa procedència (S4, límit per IP)

- **Given** intents fallits repetits contra diversos noms d'usuari (existents o no) originats des de la mateixa procedència
- **When** el nombre d'intents supera el llindar per procedència
- **Then** els intents addicionals des d'aquella procedència es rebutgen, independentment de contra quin nom d'usuari s'addrecin

### EC-07 — Ratxa d'intents fallits contra el mateix usuari des de procedències diferents (S4, límit per usuari)

- **Given** intents fallits repetits contra el mateix nom d'usuari, originats des de múltiples procedències diferents (simulant una xarxa de bots)
- **When** el nombre d'intents contra aquell usuari supera el llindar per usuari
- **Then** els intents addicionals contra aquell usuari es rebutgen encara que cada procedència individual estigui per sota del seu propi límit

### EC-08 — Login amb el camp de contrasenya buit o absent (validació d'entrada)

- **Given** el formulari o payload de login
- **When** s'envia sense contrasenya, o amb una cadena buida
- **Then** es rebutja amb un error de validació, sense arribar a comparar contra cap hash ni consumir un intent del comptador de força bruta de manera indistingible d'un intent real (vegeu Q-01 a §8)

### EC-09 — Login amb nom d'usuari buit o absent (validació d'entrada)

- **Given** el formulari o payload de login
- **When** s'envia sense nom d'usuari
- **Then** es rebutja amb un error de validació equivalent a AC-11 pel que fa a no revelar si el camp buit "gairebé" coincideix amb algun usuari

### EC-10 — Sessions expirades s'esborren i no s'acumulen indefinidament

- **Given** múltiples sessions creades i ja expirades amb el pas del temps
- **When** passa prou temps o s'executa el procés de neteja
- **Then** les sessions expirades deixen d'ocupar espai actiu a l'emmagatzematge de sessions vàlides (no s'exigeix un termini concret aquí — vegeu NFR-08)

### EC-11 — Petició de mutació amb cookie vàlida però sense sessió corresponent (sessió esborrada entremig) (S2/S6)

- **Given** una cookie que en algun moment va ser vàlida però la sessió corresponent ja no existeix al servidor (per logout en un altre dispositiu, expiració o neteja)
- **When** s'intenta una petició protegida
- **Then** es rebutja com a no autenticada, amb el mateix tractament que EC-01/EC-02 (sense distingir "token mai vàlid" de "token ja invalidat")

### EC-12 — Arrencada sense les variables d'entorn de credencials requerides (S9/config)

- **Given** falten `NIU_USER_A_HASH`, `NIU_USER_B_HASH` o `NIU_SESSION_SECRET` a l'entorn, o `NIU_SESSION_SECRET` té menys de 32 bytes
- **When** el procés arrenca
- **Then** el procés es nega a arrencar i ho indica clarament (fail-fast, `PLAN.md` §6) — mai arrenca en un estat parcialment configurat que accepti peticions

## 5. Requisits no funcionals (NFR)

| ID | Categoria | Enunciat | Objectiu / llindar |
| --- | --- | --- | --- |
| NFR-01 | sec | Els hashos de contrasenya es calculen amb bcrypt a un cost computacional que fa inviable un atac de força bruta offline a escala domèstica | Cost bcrypt = 12 (mesurat: el temps de verificació d'una contrasenya és > 200ms en maquinari de desplegament habitual, verificable per temporització directa de l'operació) |
| NFR-02 | sec | Els tokens de sessió tenen prou entropia per fer inviable l'endevinament | 256 bits generats amb un generador criptogràficament seguir; verificable per inspecció de la longitud/font declarada del token, no per assaig estadístic |
| NFR-03 | sec | La comparació de credencials i de tokens és resistent a atacs de temporització | La verificació de contrasenya i la validació de token utilitzen comparació de temps constant; verificable per revisió de codi a `/audit` i, com a senyal complementari, per una mesura de variància de temps de resposta entre "usuari inexistent" i "contrasenya incorrecta" (AC-11 ja cobreix la igualtat del cos de resposta) |
| NFR-04 | sec | Cap secret (hash de contrasenya, `NIU_SESSION_SECRET`, token de sessió) apareix mai en un fitxer committejat al repositori ni a la imatge de contenidor publicada | Escaneig de repositori i d'imatge sense cap coincidència (S9) — mateix mètode que S9/S11 ja establert a NIU-1 |
| NFR-05 | sec | Limitació de força bruta amb backoff — el llindar exacte és una decisió d'implementació, no un requisit de producte fix | Com a mínim: 10 intents fallits consecutius contra el mateix usuari en un període curt (minuts) desencadenen limitació; el llindar per IP pot ser més ampli. Valor numèric exacte i finestra temporal es confirmen a `design.md` (Stage 2) — vegeu pregunta oberta §8 |
| NFR-06 | perf | El cost de bcrypt (NFR-01) no converteix el login en perceptiblement lent per a l'ús real de dues persones | Temps de resposta del login (camí feliç) < 1s p95, mesurat en maquinari de desplegament habitual |
| NFR-07 | obs | Els intents de login fallits (incloent-hi els bloquejats per limitació de força bruta) queden registrats per a diagnòstic, sense registrar mai la contrasenya en clar | Log del servidor conté nom d'usuari intentat + resultat (èxit/fallada/limitat) + marca temporal; mai el valor de contrasenya ni el token de sessió en clar |
| NFR-08 | reliab | Les sessions expirades no s'acumulen indefinidament a l'emmagatzematge actiu | Existeix un mecanisme (programat o a demanda) que retira sessions expirades en un termini raonable (hores-dies, no mesos); verificat per EC-10 |
| NFR-09 | compliance | Cap dada personal real (noms, contrasenyes reals) apareix en cap fitxer committejat, inclosos fixtures de test i captures de pantalla del formulari de login | Escaneig de repositori sense coincidències — mateix mètode que S11 (NIU-1), estès al formulari de login i als seus tests |

## 6. Estratègia de proves (redactada per `qa-engineer`)

- **Unitat:** validació de format d'entrada de login (EC-08/EC-09 — camps buits/absents), lògica de comparació de temps constant i de generació/hashing de token com a funcions aïllades (NFR-02/NFR-03), càlcul del llindar de rate limiting per IP/usuari.
- **Integració (nucli de la cobertura):** AC-01 a AC-14 i EC-01 a EC-12 es verifiquen majoritàriament a nivell d'integració HTTP contra un servidor de test real amb SQLite temporal, seguint exactament el patró ja establert a `app/tests/integration/security_test.go` de NIU-1 (`newTestServer`, `doJSON`, assercions sobre codi d'estat, capçaleres `Set-Cookie` i cos de resposta). Els casos que exigeixen inspeccionar l'emmagatzematge de sessions directament (AC-08, S2c) reutilitzen el patró ja usat a `TestSQLInjectionPayload_StoredLiterally_TableSurvives` d'obrir `srv.Store.DB` i consultar-hi directament.
- **E2E:** el cicle complet AC-14 (login → ús → logout via UI real, no només l'API) es cobreix amb un test de navegador (Playwright, mateix framework ja en ús a NIU-1 per a CF-16–CF-22), perquè exercita el formulari de login i la persistència real de la cookie al navegador, cosa que un test d'integració HTTP pur no pot demostrar per si sol.
- **Manual / exploratori:** cap — cada AC/EC té una via d'automatització realista amb l'eina ja disponible al projecte (Go `httptest` + Playwright). No es preveu cap cas que quedi fora de CI per a NIU-4 (a diferència de REL-01/NIU-1, que exigia un script dedicat per raons de fiabilitat de CI, no aplicable aquí).
- **Validació d'NFR:**
  - NFR-01/NFR-06 (cost bcrypt i temps de login): mesura directa del temps d'execució de l'operació de verificació en un test d'integració (assert que és superior a un llindar mínim per NFR-01 i inferior a un llindar màxim per NFR-06).
  - NFR-02: revisió de codi a `/audit` (font d'aleatorietat i longitud del token) — no assajable estadísticament amb un test unitari raonable.
  - NFR-03: revisió de codi a `/audit` per confirmar l'ús d'una funció de comparació de temps constant; complementat per AC-11 (igualtat de resposta) com a senyal observable des de fora.
  - NFR-04/NFR-09: escaneig automatitzat de repositori i d'imatge (mateix mecanisme que S9/S11 de NIU-1), a CI o a `/audit`.
  - NFR-05: verificat per AC-10/EC-06/EC-07 amb el llindar concret que es fixi a `design.md`.
  - NFR-07: integració — assert que un intent fallit produeix una entrada de log amb els camps esperats i sense el valor de la contrasenya (per captura del `log/slog` output del test, mateix mecanisme ja disponible al projecte).
  - NFR-08: integració — EC-10, sembrant una sessió amb `expires_at` en el passat i confirmant que el procés de neteja la retira.

## 7. Fora d'abast (explícit)

- Registre de nous usuaris — `PLAN.md` §1: "mai tindrà registre d'usuaris".
- Recuperació o restabliment de contrasenya.
- Gestió d'usuaris (afegir, editar, desactivar comptes) — són exactament dos usuaris sembrats per sempre.
- Rols o permisos diferenciats entre Usuari A i Usuari B — tots dos tenen accés idèntic.
- Autenticació multifactor.
- Autenticació mòbil amb `Authorization: Bearer` — reservada explícitament per a un futur ítem si mai arriba una app mòbil (`PLAN.md` §9); NIU-4 només cobreix l'autenticació basada en cookie.
- Cloudflare Access com a mecanisme d'autenticació o de protecció addicional — decidit explícitament que no fa falta perquè NIU-4 es desplega abans de l'exposició pública (`PLAN.md` §3, S10 fora de la taula de NIU-4).
- Rotació automàtica de `NIU_SESSION_SECRET` en calent — la rotació és un canvi manual de configuració i redeploy, fora d'abast d'aquest ítem.

## 8. Preguntes obertes

- [ ] **Llindar numèric exacte i finestra temporal del rate limiting (NFR-05/AC-10/EC-06/EC-07).** `PLAN.md` §3 exigeix "rate limit per IP i per usuari amb backoff" sense fixar un número. Aquest document proposa un mínim observable (10 intents fallits consecutius per usuari en una finestra de minuts) perquè AC-10 sigui testable ja en aquesta etapa, però el valor exacte (llindar per IP, durada del backoff, si és fix o exponencial) es pot deixar com a decisió de `software-architect` a Stage 2 **sempre que compleixi el mínim observable aquí declarat** — owner: `software-architect` (Stage 2), confirmació humana no bloquejant per a aquesta porta.
- [ ] **EC-08 (contrasenya buida) ha de consumir un intent del comptador de força bruta?** Proposta: un payload sense contrasenya es rebutja per validació d'entrada abans d'arribar a la lògica d'intents, per no obrir una via de denegació de servei trivial (enviar payloads buits en bucle sense contrasenya). A confirmar a Stage 2 — owner: `software-architect`.
- [x] **Durada de vida de la sessió (TTL) abans d'expirar (AC-12/EC-10).** **Resolt amb el propietari humà: 30 dies.** Per a un ús domèstic de dues persones de confiança, la molèstia de re-loguejar-se sovint no aporta cap guany real de seguretat — el que protegeix la sessió és `HttpOnly`/`Secure`/`SameSite=Strict`, no la durada. `software-architect` fixa `NIU_SESSION_TTL` (o equivalent) a 30 dies a `design.md`.

