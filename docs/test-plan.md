# Niu — Pla de proves

> **Què és això.** Aquest document és el mecanisme de control principal
> del propietari humà del projecte. **El propietari no revisarà el
> codi.** Per tant, aquest document no és documentació — és **el
> contracte**.
>
> Regles (vinculants, `PLAN.md` §7):
>
> 1. Cada cas es descriu en Gherkin (Donat/Quan/Llavors).
> 2. **Cada cas es correspon amb un test automatitzat.** Un cas sense
>    test és una ficció — no s'hi escriuen casos aspiracionals.
> 3. El CI bloqueja en cas de fallada. Una suite en vermell vol dir que
>    la història no està feta.
> 4. Aquest document s'escriu durant `/define`, **abans** d'implementar,
>    i s'audita a `/audit`.
> 5. Els tests de seguretat **executen l'atac i n'afirmen el fracàs** —
>    no n'hi ha prou que la mitigació "existeixi".
>
> Document a nivell de projecte: cobreix totes les històries de
> `PLAN.md` §8. Cada cas indica a quin ítem del backlog pertany. Només
> **NIU-1** està en construcció ara; la resta (NIU-2, NIU-3, NIU-4)
> apareixen aquí perquè el contracte de seguretat i qualitat és
> d'abast de projecte, però **els seus tests encara no existeixen** —
> es marquen explícitament com a pendents de l'ítem corresponent.

## Llegenda d'estat

| Símbol | Significat |
|---|---|
| 🟢 NIU-1 | En abast ara; test ha d'existir i passar abans de tancar NIU-1 |
| ⚪ NIU-2/3/4 | Descrit aquí per traçabilitat de projecte; el test **no existeix encara** — es construirà quan s'aborda aquell ítem |

---

## 1. Casos funcionals

### 1.1 Llista de la compra — afegir ítems

#### CF-01 — Afegir un ítem vàlid 🟢 NIU-1

- **Donat** la caixa "A comprar" té zero o més ítems
- **Quan** l'usuari afegeix un ítem amb un nom vàlid
- **Llavors** l'ítem apareix a "A comprar" i, en recarregar la pàgina, continua sent-hi

**Test automatitzat:** integració — `POST /api/v1/items` seguit de `GET /api/v1/items`, assert presència de l'ítem.

#### CF-02 — Nom buit o només espais 🟢 NIU-1

- **Donat** el formulari d'afegir ítem
- **Quan** l'usuari envia un nom buit o compost només d'espais/tabulacions
- **Llavors** la petició es rebutja amb un missatge d'error clar i no es crea cap ítem

**Test automatitzat:** unitari (validació de nom) + integració (`POST` retorna error, `GET` no mostra cap ítem nou).

#### CF-03 — Nom de 200 caràcters acceptat, 201 rebutjat 🟢 NIU-1

- **Donat** el formulari d'afegir ítem
- **Quan** l'usuari envia un nom de exactament 200 caràcters (després de retallar espais)
- **Llavors** l'ítem s'accepta sencer
- **Donat** el mateix formulari
- **Quan** l'usuari envia un nom de 201 caràcters
- **Llavors** la petició es rebutja amb un missatge d'error clar

**Test automatitzat:** unitari (frontera 200/201) + integració (`POST` amb payload de 200 i 201 caràcters).

#### CF-04 — Nom amb accents, emoji i apòstrof 🟢 NIU-1

- **Donat** el formulari d'afegir ítem
- **Quan** l'usuari envia un nom com `Formatge d'ovella`, `Pastanagues 🥕`, `O'Neill`
- **Llavors** el nom es desa i es mostra exactament igual, sense corrupció

**Test automatitzat:** integració — cicle desar/llegir amb corpus de caràcters Unicode; assert igualtat byte a byte del camp retornat.

#### CF-05 — Ítem duplicat (retallat + insensible a majúscules, qualsevol caixa) 🟢 NIU-1

- **Donat** ja existeix un ítem `"Llet"` a "A comprar" (o a "Rebost")
- **Quan** l'usuari intenta afegir `"llet"`, `"Llet "` (amb espai final) o `"LLET"` a qualsevol de les dues caixes
- **Llavors** la petició es rebutja amb un missatge indicant que l'ítem ja existeix

**Test automatitzat:** integració — sembra un ítem, prova les tres variants contra totes dues caixes (6 combinacions), assert 100% rebutjades.

#### CF-06 — Duplicat exacte permès després d'eliminar l'original 🟢 NIU-1

- **Donat** un ítem `"Pa"` ha existit i ha estat eliminat
- **Quan** l'usuari afegeix `"Pa"` de nou
- **Llavors** s'accepta com un ítem nou

**Test automatitzat:** integració — crear, eliminar, tornar a crear el mateix nom, assert èxit.

### 1.2 Moviment entre caixes

#### CF-07 — Moure de "A comprar" a "Rebost" 🟢 NIU-1

- **Donat** un ítem existeix a "A comprar"
- **Quan** l'usuari el selecciona per moure'l
- **Llavors** l'ítem passa a "Rebost" en una sola operació, amb autor i moment del moviment actualitzats

**Test automatitzat:** integració — `PATCH /api/v1/items/{id}` amb `location=pantry`; assert nova ubicació i camps d'autoria a la resposta i al `GET` posterior.

#### CF-08 — Moure de "Rebost" a "A comprar" 🟢 NIU-1

- **Donat** un ítem existeix a "Rebost"
- **Quan** l'usuari el selecciona per moure'l
- **Llavors** l'ítem torna a "A comprar" amb autor i moment actualitzats

**Test automatitzat:** integració, simètric al CF-07.

#### CF-09 — El moviment persisteix a través d'una recàrrega 🟢 NIU-1

- **Donat** un ítem s'ha mogut d'una caixa a l'altra
- **Quan** es recarrega la pàgina (nova petició `GET`)
- **Llavors** l'ítem apareix a la caixa de destí, no a l'original

**Test automatitzat:** integració — `PATCH` seguit de `GET` independent, assert ubicació.

#### CF-10 — Eliminar un ítem 🟢 NIU-1

- **Donat** un ítem existeix (a qualsevol caixa)
- **Quan** l'usuari l'elimina
- **Llavors** desapareix de totes dues caixes i no reapareix en recarregar

**Test automatitzat:** integració — `DELETE`, després `GET` confirma absència.

### 1.3 Dos usuaris

#### CF-11 — Convergència entre usuàries A i B 🟢 NIU-1

- **Donat** la usuària A afegeix un ítem
- **Quan** la usuària B recarrega, torna el focus a la finestra, o espera l'interval de sondeig (~10s)
- **Llavors** B veu l'ítem afegit per A

**Test automatitzat:** integració — dues sessions HTTP simulades, `POST` des d'una, `GET` des de l'altra, assert visibilitat.

#### CF-12 — Moviment concurrent del mateix ítem convergeix sense error 🟢 NIU-1

- **Donat** el mateix ítem és visible per A i B
- **Quan** totes dues el mouen gairebé simultàniament (potser a destins diferents)
- **Llavors** cap petició falla amb error de servidor, i després d'un refresc ambdós clients veuen el mateix estat final (últim escriptor guanya)

**Test automatitzat:** integració — dues peticions `PATCH` concurrents (goroutines/threads paral·lels) contra el mateix ítem, assert que ambdues retornen 2xx o un error controlat (mai 5xx), i que un `GET` posterior mostra un estat consistent i únic.

#### CF-13 — Accions atribuïdes a l'avatar correcte 🟢 NIU-1

- **Donat** un ítem és afegit per A i mogut per B
- **Quan** es consulta l'ítem
- **Llavors** la resposta identifica A com a creador i B com a darrer autor del moviment

**Test automatitzat:** integració — assert camps d'autoria després d'accions fetes amb identitats d'usuari diferents.

### 1.4 Persistència

#### CF-14 — Reinici del contenidor conserva les dades 🟢 NIU-1

- **Donat** hi ha ítems desats
- **Quan** el contenidor es reinicia
- **Llavors** en tornar a arrencar, tots els ítems i el seu estat romanen intactes

**Test automatitzat:** integració/script — sembrar dades, reiniciar el procés/contenidor, `GET /api/v1/items` assert igualtat de conjunt.

#### CF-15 — Reinici a mig d'una escriptura no corromp la base de dades 🟢 NIU-1

- **Donat** una escriptura està en curs
- **Quan** el procés es talla abruptament (`docker kill` o equivalent)
- **Llavors** la base de dades no queda corrupta i l'aplicació arrenca correctament

**Test automatitzat:** script dedicat repetit 10 vegades — llançar escriptures contínues, matar el procés en un instant aleatori, verificar arrencada neta i integritat (`PRAGMA integrity_check`).

### 1.5 Interfície

#### CF-16 — Animació de moviment correcta 🟢 NIU-1

- **Donat** un ítem es mou
- **Quan** `prefers-reduced-motion` no està actiu
- **Llavors** l'animació de vol (FLIP) s'executa en ~250ms i l'ítem acaba posicionat correctament a la caixa de destí

**Test automatitzat:** E2E (navegador real) — mesurar durada de la transició i posició final del DOM.

#### CF-17 — `prefers-reduced-motion` desactiva el vol 🟢 NIU-1

- **Donat** `prefers-reduced-motion` està actiu
- **Quan** un ítem es mou
- **Llavors** es mostra un esvaïment (cross-fade), no un desplaçament volador

**Test automatitzat:** E2E amb l'emulació de `prefers-reduced-motion` activada al navegador de test.

#### CF-18 — Confeti en buidar "A comprar", un sol cop 🟢 NIU-1

- **Donat** "A comprar" té un únic ítem
- **Quan** aquest ítem es mou o s'elimina i la caixa queda buida
- **Llavors** s'executa una animació de confeti exactament un cop, i no torna a dispara's en renderitzats posteriors mentre la llista continua buida

**Test automatitzat:** E2E — assert que l'element/esdeveniment de confeti apareix exactament una vegada després de buidar, i no reapareix en accions posteriors sense afegir-hi ítems.

#### CF-19 — Viewport mòbil apila les caixes 🟢 NIU-1

- **Donat** l'usuari obre l'aplicació en una amplada de pantalla mòbil
- **Quan** interactua amb la interfície
- **Llavors** les caixes es presenten apilades amb pestanyes, i totes les accions (afegir, moure, eliminar) continuen funcionant

**Test automatitzat:** E2E amb viewport emulat (p. ex. 375×667).

#### CF-20 — Actualització optimista amb error de servidor reverteix i avisa 🟢 NIU-1

- **Donat** un usuari mou un ítem i la interfície ja l'ha mogut de manera optimista
- **Quan** la petició al servidor falla
- **Llavors** l'ítem torna animadament a la posició original i es mostra un avís (toast) no bloquejant

**Test automatitzat:** E2E — simular resposta d'error del servidor (mock/intercept de xarxa), assert reversió visual i presència del toast.

### 1.6 Accessibilitat funcional

#### CF-21 — Navegació completa per teclat 🟢 NIU-1

- **Donat** l'usuari no fa servir ratolí ni pantalla tàctil
- **Quan** navega únicament amb teclat
- **Llavors** pot afegir, moure i eliminar ítems sense cap altre dispositiu d'entrada

**Test automatitzat:** E2E — simular seqüències de `Tab`/`Enter`/`Space`, assert que cada acció es pot completar.

#### CF-22 — Anunci `aria-live` en moure un ítem 🟢 NIU-1

- **Donat** un lector de pantalla actiu
- **Quan** un ítem es mou (per acció pròpia o reflectint un canvi remot detectat pel sondeig)
- **Llavors** una regió `aria-live` anuncia el nom de l'ítem i la caixa de destí

**Test automatitzat:** E2E — assert contingut de la regió `aria-live` després del moviment.

---

## 2. Casos no funcionals

### 2.1 Seguretat

**Regla:** cada test executa l'atac real i afirma que **falla** — no
n'hi ha prou que la mitigació existeixi en el codi.

| ID | Cas | Donat / Quan / Llavors | Estat |
|---|---|---|---|
| S1a | Cap mutació via `GET` | **Donat** la taula de rutes de l'API — **Quan** s'inspeccionen totes les rutes `GET` — **Llavors** cap d'elles produeix un efecte d'escriptura (crear/moure/eliminar) | 🟢 NIU-1 |
| S1b | Token CSRF de doble-submit rebutja mutacions sense token | **Donat** una sessió autenticada — **Quan** s'envia `POST/PATCH/DELETE` sense el token CSRF esperat — **Llavors** la petició es rebutja amb 403 | ⚪ NIU-4 |
| S2a | Petició sense cookie és rebutjada | **Donat** cap cookie de sessió — **Quan** es crida un endpoint protegit — **Llavors** respon 401 | ⚪ NIU-4 |
| S2b | Cookie alterada és rebutjada | **Donat** una cookie de sessió amb el valor manipulat — **Quan** es crida un endpoint protegit — **Llavors** respon 401 | ⚪ NIU-4 |
| S2c | La base de dades mai conté el token en clar | **Donat** una sessió activa amb token conegut pel test — **Quan** s'inspecciona la taula `sessions` — **Llavors** cap valor emmagatzemat coincideix amb el token en clar (només el seu hash) | ⚪ NIU-4 |
| S3a | XSS en nom d'ítem es renderitza com a text literal | **Donat** un ítem amb nom `<img src=x onerror=alert(1)>` — **Quan** es renderitza en un navegador real — **Llavors** no s'executa cap script i el text apareix literalment | 🟢 NIU-1 |
| S3b | CSP sense `unsafe-inline` | **Donat** qualsevol resposta HTML — **Quan** s'inspecciona la capçalera `Content-Security-Policy` — **Llavors** no conté `unsafe-inline` | 🟢 NIU-1 |
| S4 | 10 intents de login fallits activen limitació | **Donat** un usuari desconegut o contrasenya incorrecta — **Quan** es fan 10 intents consecutius — **Llavors** l'intent següent es rebutja per límit de freqüència | ⚪ NIU-4 |
| S5 | Usuari inexistent i contrasenya incorrecta donen error idèntic | **Donat** un nom d'usuari que no existeix i un altre que existeix amb contrasenya incorrecta — **Quan** s'intenta iniciar sessió amb cadascun — **Llavors** el cos de la resposta d'error és byte a byte idèntic en tots dos casos | ⚪ NIU-4 |
| S6 | Token canvia en cada login; logout l'invalida | **Donat** un usuari que inicia sessió dues vegades — **Quan** es comparen els tokens emesos — **Llavors** són diferents; **Quan** es fa logout i es reutilitza el token antic — **Llavors** es rebutja | ⚪ NIU-4 |
| S7 | Capçaleres de seguretat presents a totes les respostes | **Donat** qualsevol resposta de l'API o de fitxers estàtics — **Quan** s'inspeccionen les capçaleres — **Llavors** hi són totes: `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Content-Security-Policy` | 🟢 NIU-1 |
| S8 | Injecció SQL al nom d'ítem no afecta la base de dades | **Donat** un nom d'ítem `'; DROP TABLE items;--` — **Quan** es crea l'ítem i després es consulta l'estat de la taula `items` — **Llavors** el nom es desa literalment com a text i la taula continua existint amb totes les seves dades | 🟢 NIU-1 |
| S9 | Cap secret a la imatge ni al repositori | **Donat** la imatge Docker publicada i el repositori git — **Quan** s'escaneja la imatge (`docker history` / eina d'escaneig) i es cerca al repo — **Llavors** no apareix cap hash de contrasenya ni secret en clar | ⚪ NIU-2 |
| S10 | Política de Cloudflare Access activa abans del primer desplegament públic | **Donat** el domini `niu.fikua.com` — **Quan** s'intenta accedir sense passar per l'autenticació de Cloudflare Access — **Llavors** l'accés es bloqueja | ⚪ NIU-2 (verificació manual documentada, no test de CI) |
| S11 | Cap dada personal al repositori públic | **Donat** que `fikua/fikua-niu` és un repositori **públic** — **Quan** s'escanegen tots els fitxers versionats (codi, comentaris, migracions, dades de seed, fixtures de test, documentació) — **Llavors** no hi apareix cap nom real, correu electrònic ni detall domèstic identificable; els documents només fan servir `Usuari A` / `Usuari B` i les identitats reals s'injecten per variable d'entorn | 🟢 NIU-1 |

### 2.2 Rendiment

| ID | Cas | Donat / Quan / Llavors | Estat |
|---|---|---|---|
| PERF-01 | Latència de llistat amb 500 ítems | **Donat** la base de dades conté 500 ítems — **Quan** es fan repetides peticions `GET /api/v1/items` — **Llavors** el p95 de latència és < 200ms | 🟢 NIU-1 |
| PERF-02 | Càrrega inicial ràpida en xarxa lenta | **Donat** un perfil de xarxa 3G simulada — **Quan** es carrega la pàgina per primer cop — **Llavors** el temps fins a interactiu és < 1s | 🟢 NIU-1 |
| PERF-03 | Mida de la imatge de contenidor | **Donat** la imatge Docker publicada — **Quan** se n'inspecciona la mida — **Llavors** és < 30MB | ⚪ NIU-2 |

### 2.3 Fiabilitat

| ID | Cas | Donat / Quan / Llavors | Estat |
|---|---|---|---|
| REL-01 | `docker kill` a mig `UPDATE` no corromp la base de dades | **Donat** una escriptura en curs — **Quan** el contenidor es mata abruptament — **Llavors** la base de dades passa `PRAGMA integrity_check` i l'app arrenca neta | 🟢 NIU-1 |
| REL-02 | Restauració d'una còpia de seguretat és fidel | **Donat** una còpia de seguretat presa amb `sqlite3 .backup` — **Quan** es restaura en una base de dades buida — **Llavors** el conjunt d'ítems resultant és idèntic a l'original | ⚪ NIU-2 |
| REL-03 | `/healthz` reflecteix l'estat real de la base de dades | **Donat** la base de dades és accessible — **Quan** es crida `GET /healthz` — **Llavors** respon 200; **Donat** la base de dades no és accessible — **Quan** es crida `GET /healthz` — **Llavors** respon amb un codi d'error | 🟢 NIU-1 |

### 2.4 Accessibilitat

| ID | Cas | Donat / Quan / Llavors | Estat |
|---|---|---|---|
| A11Y-01 | Totes les accions operables per teclat | **Donat** cap dispositiu de punter — **Quan** es navega només amb teclat — **Llavors** afegir, moure i eliminar es poden completar | 🟢 NIU-1 |
| A11Y-02 | Contrast AA en tot el text | **Donat** la paleta cromàtica definida — **Quan** s'audita cada combinació text/fons — **Llavors** compleix el llindar AA de WCAG 2.2 | 🟢 NIU-1 |
| A11Y-03 | `aria-live` anuncia els moviments | **Donat** un lector de pantalla — **Quan** un ítem es mou — **Llavors** la regió `aria-live` anuncia el canvi de manera comprensible | 🟢 NIU-1 |

### 2.5 Observabilitat

| ID | Cas | Donat / Quan / Llavors | Estat |
|---|---|---|---|
| OBS-01 | Una petició genera una traça completa | **Donat** l'aplicació instrumentada amb OTEL — **Quan** es fa una petició HTTP — **Llavors** apareix una traça completa a OpenObserve amb `service.name = niu` | ⚪ NIU-3 |
| OBS-02 | La traça inclou span HTTP i span de base de dades | **Donat** la mateixa petició de OBS-01 — **Quan** s'inspecciona la traça — **Llavors** conté un span HTTP i almenys un span de l'operació de base de dades corresponent | ⚪ NIU-3 |
| OBS-03 | El col·lector no descarta dades de Niu | **Donat** la configuració actualitzada del col·lector (`config.yaml` amb `service.name == "niu"`) — **Quan** es genera trànsit des de Niu — **Llavors** les dades apareixen a OpenObserve, no es perden silenciosament | ⚪ NIU-3 |

---

## 3. Resum de cobertura per ítem del backlog

| Ítem | Casos funcionals | Casos no funcionals | Total |
|---|---|---|---|
| NIU-1 | CF-01 a CF-22 (22) | S1a, S3a, S3b, S7, S8, S11, PERF-01, PERF-02, REL-01, REL-03, A11Y-01, A11Y-02, A11Y-03 (13) | 35 |
| NIU-2 | — | S9, S10, PERF-03, REL-02 (4) | 4 |
| NIU-3 | — | OBS-01, OBS-02, OBS-03 (3) | 3 |
| NIU-4 | — | S1b, S2a, S2b, S2c, S4, S5, S6 (7) | 7 |

**Regla de tancament:** cap ítem del backlog es dona per fet (`Done`)
mentre algun dels seus casos marcats amb el seu propi símbol d'estat no
tingui un test automatitzat verd a CI (excepte S10, verificació manual
documentada per naturalesa, i REL-01/EC de reinici abrupte si l'entorn
de CI no ho permet de manera fiable — en aquest darrer cas, procediment
manual repetible i obligatori, no opcional).
