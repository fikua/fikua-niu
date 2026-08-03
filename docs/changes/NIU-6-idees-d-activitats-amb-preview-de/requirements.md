---
artefact: requirements
key: "NIU-6"
title: "Idees d'activitats amb previsualització de link"
status: "approved"
owner: "product-manager + qa-engineer"
proposal_path: "./proposal.md"
ac_count: 11
nfr_count: 9
sources:
  - "User Story format (Mike Cohn) — As a / I want / So that"
  - "INVEST — Independent, Negotiable, Valuable, Estimable, Small, Testable"
  - "Given/When/Then — Gherkin / Cucumber"
created: "2026-08-03"
updated: "2026-08-03"
---

# Requirements — Idees d'activitats amb previsualització de link

> **Què és això.** El contracte entre producte i enginyeria per a NIU-6.
> Cada criteri d'acceptació és observable des de fora del sistema i es
> traça a almenys una tasca de `tasks.md`. **Només comportament
> funcional — cap detall d'implementació.** Referència:
> [`./proposal.md`](./proposal.md).
> Font vinculant: `PLAN.md` §2.4 (filosofia de model de dades), §3
> (Seguretat — base per als NFR de seguretat d'aquest ítem, que
> l'estenen amb una superfície nova: recuperació server-side de
> contingut extern).

## 0. Decisions recomanades (per a la porta humana)

> `proposal.md` §9 deixa tres preguntes explícitament obertes.
> `qa-engineer` i `product-manager` proposem una resposta concreta a
> cadascuna perquè la porta humana tingui alguna cosa a confirmar o
> corregir, en lloc d'una pregunta buida. **Són recomanacions, no
> decisions preses.**

### 0.1 Cicle de vida — recomanació: cap camp d'estat en v1

- **Recomanació:** v1 és una **llista simple de desades**: afegir i
  eliminar, sense cap camp d'estat (ni "feta", ni "planificada").
- **Per què:** consistent amb el biaix d'aquest projecte cap a un abast
  mínim de v1 (`PLAN.md` §1), i amb el propòsit declarat d'aquest espai
  (`proposal.md` §4/§6): és un arxiu d'idees amb context visual, no un
  flux de decisió com NIU-5. Introduir un camp d'estat abans de tenir
  ús real seria exactament el mateix error que NIU-5 §0 ja va evitar
  conscientment en l'altre sentit (allà, un flux de 3 estats calia
  perquè el propòsit *és* seguir una decisió; aquí, el propòsit
  declarat és col·leccionar, no decidir).
- **Conseqüència pràctica:** si l'ús real demostra que cal distingir
  "per fer" de "ja feta" per reduir soroll a la llista, és un canvi
  additiu posterior (afegir un camp), no una migració destructiva.
- **Es confirma a la porta humana d'aquest document** (§8, primera
  pregunta oberta) — si el propietari prefereix, com a mínim, poder
  marcar una idea com a "feta" des del dia 1, es reescriu l'abast
  d'AC-05/AC-06 abans d'aprovar Stage 1.

### 0.2 Fallback quan la previsualització no es pot generar — recomanació: desar igualment, sense bloquejar

- **Recomanació:** quan la recuperació automàtica de títol/imatge/
  descripció falla, fa timeout, o la pàgina no és compatible, la idea
  **es desa igualment** amb l'enllaç visible i sense targeta visual —
  mai un error que impedeixi desar la idea.
- **Per què:** el propòsit de l'espai (`proposal.md` §2) és no perdre
  la idea; bloquejar l'usuari perquè un tercer (Instagram, un web amb
  bloqueig anti-bot) no permet l'accés automàtic seria traslladar un
  problema tècnic aliè a fregament d'usuari — exactament el fregament
  que aquest ítem vol eliminar. `proposal.md` §7 ja documenta que
  aquest fallback serà el cas **habitual**, no l'excepció, per a webs
  com Instagram.
- **Conseqüència pràctica:** la idea sense previsualització és un
  resultat vàlid i esperat, no un estat d'error transitori — no hi ha
  reintent automàtic ni cua de reprocessament en v1 (fora d'abast, §7).
- **Es confirma a la porta humana d'aquest document** (§8, segona
  pregunta oberta).

### 0.3 SSRF — no es proposa mitigació aquí, però es fixen les restriccions mínimes com a NFR vinculant

- `proposal.md` §7 flag explícitament que recuperar contingut
  server-side d'una URL introduïda per l'usuari és una superfície
  d'abús coneguda i demana que `software-architect` la resolgui a
  l'Etapa 2. **Aquest document no dissenya la mitigació** — però, com
  a requeriment no funcional vinculant, exigeix que qualsevol solució
  tècnica compleixi un mínim observable (NFR-06/07/08 més avall):
  timeout dur, límit de mida de resposta, rebuig d'esquemes no
  `http(s)` i rebuig de destins que resolen a xarxa privada/loopback
  (incloent-hi la mateixa instància de Niu i el VPS on viu). Aquests
  llindars són el que `qa-engineer` pot verificar des de fora del
  sistema sense conèixer la implementació — la implementació concreta
  (allowlist, resolució DNS controlada, etc.) és decisió de
  `software-architect`.
- **No es confirma a la porta humana d'aquest document** — és un
  requisit no negociable, no una pregunta oberta.

## 1. Historial d'usuari

- **Com a** membre de la parella que utilitza Niu
- **Vull** desar una idea d'activitat enganxant només l'enllaç trobat
  navegant, i veure-la representada amb una targeta reconeixible
  (títol, imatge, descripció) sense haver de tornar a obrir l'enllaç
- **Perquè** cap activitat interessant es perdi en una captura de
  pantalla o un missatge que ningú torna a mirar quan arriba el moment
  de decidir un pla.

## 2. Autoavaluació INVEST

- [x] **Independent** — ✅ No depèn de cap altre ítem en curs; comparteix
  només l'app i els dos usuaris amb NIU-1/NIU-5, no el seu model de
  dades (`proposal.md` §6, "fora d'abast").
- [x] **Negotiable** — ✅ El cicle de vida (§0.1) i el comportament del
  fallback (§0.2) són negociables a la porta humana; el contracte
  extern (desar un enllaç, veure'l com a targeta o com a enllaç
  simple, eliminar-lo) és fix. El mecanisme tècnic de mitigació de
  SSRF (§0.3) és negociable a Stage 2, els seus llindars mínims no.
- [x] **Valuable** — ✅ Elimina un fregament domèstic real i recurrent
  (`proposal.md` §2), amb una mesura d'èxit qualitativa clara
  (`proposal.md` §5).
- [x] **Estimable** — ✅ Abast tancat un cop es resolguin les preguntes
  de §8: desar enllaç + scraping OG amb fallback + eliminar.
- [x] **Small** — ✅ Encaixa en un sol cicle de `/define` → `/code` →
  `/audit`, assumint la llista sense estat recomanada a §0.1. La
  complexitat principal (SSRF) és de disseny tècnic, no d'abast
  funcional.
- [x] **Testable** — ✅ Cada AC és observable via l'API HTTP o el DOM
  renderitzat; cap AC depèn d'inspeccionar estat intern no exposat.
  Els NFR de seguretat (§0.3) són observables externament: es poden
  verificar enviant una URL que apunti a un destí prohibit i
  comprovant que la petició es rebutja, sense conèixer com s'implementa
  el rebuig.

## 3. Criteris d'acceptació

### AC-01 — Afegir una idea a partir d'un enllaç vàlid amb previsualització

- **Given** l'usuari té un enllaç a una pàgina pública que exposa
  metadades Open Graph (títol, imatge, descripció)
- **When** l'usuari enganxa l'enllaç i confirma que vol desar la idea
- **Then** la idea apareix a la llista com una targeta amb títol,
  imatge i descripció recuperats automàticament, persisteix després de
  recarregar la pàgina, i mostra qui l'ha afegit

### AC-02 — Afegir una idea quan la previsualització no es pot generar

- **Given** l'usuari té un enllaç a una pàgina que bloqueja l'accés
  automàtic, no té metadades compatibles, o no respon a temps
- **When** l'usuari enganxa l'enllaç i confirma que vol desar la idea
- **Then** la idea es desa igualment, amb l'enllaç visible i un
  missatge clar que no hi ha previsualització disponible — **la idea
  mai deixa de desar-se** per aquest motiu, i l'usuari no rep cap error
  bloquejant

### AC-03 — Previsualització amb dades parcials

- **Given** l'usuari té un enllaç a una pàgina que exposa només algunes
  metadades Open Graph (p. ex. títol però no imatge, o imatge però no
  descripció)
- **When** l'usuari enganxa l'enllaç i confirma que vol desar la idea
- **Then** la targeta mostra els camps recuperats i omet visiblement
  els que no existeixen, sense mostrar un error ni un buit trencat en
  el lloc de la dada absent

### AC-04 — Cada idea mostra qui l'ha afegit

- **Given** una idea ha estat desada
- **When** es mostra a la llista
- **Then** és visible quin dels dos usuaris l'ha afegit

### AC-05 — Eliminar una idea desada

- **Given** una idea existeix a la llista (amb o sense previsualització)
- **When** l'usuari l'elimina explícitament
- **Then** la idea desapareix de la llista i no reapareix en recarregar

### AC-06 — Dos usuaris veuen les mateixes idees (convergència eventual)

- **Given** un usuari afegeix o elimina una idea
- **When** l'altre usuari recarrega la pàgina, torna el focus a la
  finestra, o espera l'interval de sondeig ja establert per NIU-1
  (~10s)
- **Then** veu la llista actualitzada, incloent-hi les targetes amb
  previsualització i les idees en fallback sense-previsualització

### AC-07 — Espai visualment diferenciat de la llista de la compra i de compres/projectes

- **Given** l'usuari navega per Niu
- **When** entra a l'espai d'idees d'activitats
- **Then** distingeix clarament, només pel visual, que no és ni la
  llista de la compra (NIU-1) ni l'espai de compres grans / projectes
  de casa (NIU-5) — mantenint la mateixa estètica càlida (`PLAN.md`
  §4)

### AC-08 — Enllaç obligatori i amb format vàlid

- **Given** el formulari d'afegir idea
- **When** l'usuari hi introdueix un text que és una URL `http(s)`
  sintàcticament vàlida
- **Then** la petició es processa (amb o sense previsualització segons
  AC-01/AC-02)

### AC-09 — Navegació completa per teclat

- **Given** l'usuari no utilitza ratolí ni pantalla tàctil
- **When** navega l'espai únicament amb teclat
- **Then** pot afegir una idea (enganxar l'enllaç i confirmar) i
  eliminar-ne sense necessitar cap altre dispositiu d'entrada

### AC-10 — Targeta accessible per lectors de pantalla

- **Given** l'usuari utilitza un lector de pantalla
- **When** es mostra una targeta (amb o sense previsualització)
- **Then** el títol, la descripció (si existeix) i l'enllaç subjacent
  són anunciats de manera comprensible; la imatge (si existeix) té un
  text alternatiu no buit

### AC-11 — `overview.md` reflecteix el nou espai

- **Given** aquest ítem queda aprovat i implementat
- **Then** `docs/overview.md` s'actualitza per esmentar l'espai
  d'idees d'activitats com a funcionalitat existent de Niu, mantenint-lo
  com a font única de veritat del que fa l'app

> AC-11 no té un "When" d'interacció d'usuari — és una condició de
> tancament documental, no una interacció observable des de la UI/API.
> Es manté com a AC pel mateix motiu que NIU-5 AC-13: `proposal.md` no
> l'esmenta explícitament com a pregunta oberta d'aquesta vegada, però
> es manté la mateixa disciplina documental que la resta de Niu.

## 4. Casos límit i escenaris negatius

### EC-01 — Esquema d'URL no `http(s)`

- **Given** el formulari d'afegir idea
- **When** l'usuari envia una URL amb un esquema diferent de `http`/
  `https` (p. ex. `file://`, `javascript:`, `ftp://`, `data:`)
- **Then** la petició es rebutja amb un missatge d'error clar i cap
  idea es crea — **sense** intentar cap recuperació de contingut

### EC-02 — URL que apunta a xarxa privada, loopback, o a la mateixa instància de Niu

- **Given** el formulari d'afegir idea
- **When** l'usuari envia una URL que resol a `localhost`, `127.0.0.1`,
  una adreça de xarxa privada (`10.0.0.0/8`, `172.16.0.0/12`,
  `192.168.0.0/16`), una adreça link-local, o el mateix domini/adreça
  IP on corre la instància de Niu (incloent-hi el VPS que l'allotja)
- **Then** la petició es rebutja com a destí no permès, amb un
  missatge d'error clar que no revela detalls interns de xarxa, i **la
  idea es pot desar igualment sense previsualització** (mateix
  fallback d'AC-02) o bé es rebutja tota la petició — comportament
  exacte a decidir a Stage 2, però en cap cas el servidor arriba a fer
  la petició cap a aquest destí

> Aquest EC és la concreció observable del risc SSRF flag a
> `proposal.md` §7 i formalitzat com a NFR-06/07/08 (§5). No prescriu
> la implementació (resolució DNS, allowlist, etc.) — només exigeix que
> el resultat extern sigui "el servidor mai contacta aquest destí".

### EC-03 — Resposta extremadament gran

- **Given** l'URL enllaçada respon amb un cos de mida molt superior a
  l'esperat per a una pàgina HTML normal (p. ex. desenes o centenars de
  MB, intencionadament o per error del servidor remot)
- **When** Niu intenta recuperar-ne la previsualització
- **Then** la recuperació s'atura abans de completar la descàrrega
  sencera, la idea es desa en mode fallback (AC-02), i el procés no
  exhaureix memòria ni bloqueja altres peticions al servidor

### EC-04 — Cadena de redireccions

- **Given** l'URL enllaçada respon amb una o més redireccions HTTP
  abans d'arribar al contingut final
- **When** Niu intenta recuperar-ne la previsualització
- **Then** o bé se segueix la cadena fins a un límit raonable de salts
  i cada salt intermedi es valida amb les mateixes restriccions que la
  URL original (EC-01/EC-02 — una redirecció no és una via per
  esquivar-les), o bé se supera el límit i es desa en mode fallback
  (AC-02) — en cap cas una redirecció acaba fent que el servidor
  contacti un destí que hauria estat rebutjat si s'hagués introduït
  directament

### EC-05 — Metadades Open Graph absents o malformades

- **Given** l'URL enllaçada respon correctament però no conté cap
  etiqueta Open Graph reconeguda, o el contingut és malformat
- **When** Niu intenta recuperar-ne la previsualització
- **Then** la idea es desa en mode fallback (AC-02), sense error visible
  a l'usuari més enllà del missatge esperat de "sense previsualització"

### EC-06 — Idea duplicada (mateix enllaç desat dues vegades)

- **Given** ja existeix una idea desada amb un enllaç
- **When** l'usuari (el mateix o l'altre) intenta desar el mateix
  enllaç una altra vegada
- **Then** la petició **no es bloqueja** — a diferència de NIU-1/NIU-5,
  aquest espai no imposa unicitat d'enllaç en v1 (és una col·lecció
  d'idees, no un catàleg amb identitat única); la segona idea es desa
  com una entrada independent

> **Nota de disseny (no d'implementació):** aquesta decisió és
> deliberadament diferent de la regla de duplicats de NIU-1/NIU-5.
> Allà, el duplicat representa la mateixa entitat de domini (el mateix
> ítem de compra, el mateix projecte) i bloquejar-lo evita soroll. Aquí,
> dues persones poden trobar i desar el mateix enllaç independentment
> sense saber-ho, i no hi ha un cost clar a permetre-ho — imposar
> unicitat exigiria decidir normalització d'URL (paràmetres de
> tracking, `www.` vs no, `http` vs `https`) sense cap benefici
> evident. Si l'ús real demostra fregament per duplicats visibles, és
> una petita extensió per a un canvi posterior. Es marca com a
> pregunta oberta a §8 perquè el propietari confirmi.

### EC-07 — URL que apunta a localhost/xarxa privada arriba embolcallada (no literal)

- **Given** el formulari d'afegir idea
- **When** l'usuari envia una URL amb un nom de domini que **resol**
  (via DNS) a una adreça de xarxa privada o loopback, encara que el
  text de la URL en si no sigui una IP literal
- **Then** el mateix rebuig d'EC-02 aplica — la restricció es basa en
  el destí real de la petició, no en l'aparença textual de la URL

### EC-08 — Objectiu respon molt lentament (timeout)

- **Given** l'URL enllaçada no respon (o respon parcialment) dins d'un
  temps raonable
- **When** Niu intenta recuperar-ne la previsualització
- **Then** la recuperació s'atura en arribar al límit de temps, la
  idea es desa en mode fallback (AC-02), i l'usuari no espera
  indefinidament una resposta de la interfície

### EC-09 — Contingut no HTML (PDF, imatge directa, etc.)

- **Given** l'URL enllaçada respon amb un tipus de contingut que no és
  una pàgina HTML (p. ex. un PDF, una imatge servida directament, un
  vídeo)
- **When** Niu intenta recuperar-ne la previsualització
- **Then** la idea es desa en mode fallback (AC-02) — no s'intenta
  interpretar el contingut no-HTML com si tingués metadades Open Graph

### EC-10 — Enllaç buit o només espais

- **Given** el formulari d'afegir idea
- **When** l'usuari envia un camp d'enllaç buit o compost únicament
  per espais/tabulacions
- **Then** la petició es rebutja amb un missatge d'error clar i cap
  idea es crea

### EC-11 — Injecció HTML/JS al títol o descripció recuperats

- **Given** una pàgina externa exposa metadades Open Graph que
  contenen marcatge HTML o script (p. ex. un títol
  `<img src=x onerror=alert(1)>`)
- **When** Niu recupera i mostra la targeta
- **Then** el contingut es mostra com a **text literal** (no s'executa
  cap script, no es renderitza cap element HTML) — mateix estàndard
  que NIU-1 (S3)

### EC-12 — Injecció SQL a l'enllaç o a les metadades recuperades

- **Given** el formulari d'afegir idea o el contingut recuperat d'una
  pàgina externa
- **When** l'enllaç o qualsevol metadada recuperada conté un fragment
  com `'; DROP TABLE ideas;--`
- **Then** el valor es desa literalment com a text, la resta de dades
  de l'aplicació roman intacta, i cap taula desapareix — mateix
  estàndard que NIU-1 (S8)

### EC-13 — Intent de mutació via `GET`

- **Given** l'API exposa endpoints per a aquest espai
- **When** un client intenta provocar una mutació (crear, eliminar)
  mitjançant una petició `GET`
- **Then** no existeix cap ruta `GET` amb efectes secundaris — mateix
  estàndard que NIU-1 (S1a)

### EC-14 — Accés sense sessió autenticada

- **Given** cap cookie de sessió vàlida present a la petició
- **When** es crida qualsevol endpoint d'aquest espai
- **Then** la petició es rebutja com a no autenticada, exactament amb
  el mateix mecanisme ja establert per NIU-4 — aquest ítem no
  introdueix cap excepció ni ruta pública nova

### EC-15 — Eliminar una idea ja eliminada (doble clic / reenviament)

- **Given** una idea ja ha estat eliminada
- **When** una segona petició d'eliminació per la mateixa idea arriba
  al servidor
- **Then** la petició no provoca un error 5xx ni corromp l'estat; la
  idea continua absent (operació idempotent)

### EC-16 — Reenviament del mateix formulari d'afegir (doble clic)

- **Given** l'usuari envia el formulari d'afegir idea
- **When** la mateixa petició d'afegir s'envia dues vegades seguides
  (doble clic, doble tap, reintent de xarxa del client)
- **Then** el comportament segueix EC-06 (no hi ha unicitat d'enllaç en
  v1): es creen dues idees independents, sense error 5xx ni estat
  corrupte — es documenta explícitament perquè no se sobreentengui
  idempotència on no n'hi ha (a diferència d'EC-15, que sí que ho és)

### EC-17 — Llista buida en primer ús

- **Given** l'espai s'obre per primer cop sense cap idea desada
- **When** l'usuari hi entra
- **Then** es mostra un estat visual clar de "res per mostrar", sense
  error i sense confondre's amb un error de càrrega

### EC-18 — Viewport mòbil

- **Given** l'usuari obre aquest espai en una pantalla estreta (mòbil)
- **When** interactua amb la interfície
- **Then** el contingut s'adapta mantenint totes les funcionalitats
  (afegir, veure targetes, eliminar), seguint el mateix patró
  responsive que NIU-1/NIU-5

## 5. Requisits no funcionals (NFR)

| ID | Categoria | Enunciat | Objectiu / llindar |
| --- | --- | --- | --- |
| NFR-01 | sec (XSS) | Cap dada recuperada d'una pàgina externa (títol, descripció) ni cap enllaç introduït per l'usuari es renderitza mai com a HTML | Zero ocurrències d'`innerHTML` amb dades externes o d'usuari al codi client; EC-11 passa en cada execució de CI |
| NFR-02 | sec (injecció SQL) | Cap valor introduït per l'usuari ni recuperat d'una pàgina externa es concatena mai a una sentència SQL | 100% de les consultes amb entrada d'usuari o dada externa utilitzen paràmetres vinculats; EC-12 passa en cada execució de CI |
| NFR-03 | sec (mutació via GET) | Cap mutació és accessible via `GET` | Assaig de la taula de rutes: 0 rutes `GET` amb efecte d'escriptura (EC-13) |
| NFR-04 | sec (autenticació) | Tots els endpoints d'aquest espai requereixen sessió vàlida, reutilitzant el mecanisme de NIU-4 sense excepcions | EC-14 passa en cada execució de CI; 0 endpoints d'aquest ítem exempts del middleware d'autenticació |
| NFR-05 | sec (SSRF — restricció d'esquema) | El mecanisme de recuperació de previsualització **rebutja qualsevol esquema d'URL diferent de `http`/`https`** abans de fer cap petició de xarxa | 100% de les URL amb esquema no `http(s)` rebutjades sense petició de xarxa sortint; EC-01 passa en cada execució de CI |
| NFR-06 | sec (SSRF — destins prohibits) | El mecanisme de recuperació de previsualització **mai fa una petició** cap a `localhost`, adreces loopback, rangs de xarxa privada, adreces link-local, o el domini/IP de la mateixa instància de Niu — incloent-hi quan el destí s'hi arriba per resolució DNS o per redirecció, no només per IP literal | 100% de les URL/redireccions cap a aquests destins rebutjades sense completar la petició; EC-02/EC-04/EC-07 passen en cada execució de CI. **Vinculant per a Stage 2** (`proposal.md` §7, risc HIGH) — la mitigació concreta és decisió de `software-architect`, aquest llindar no |
| NFR-07 | sec (SSRF — límits de recurs) | La recuperació de previsualització té un **temps màxim d'espera** i un **límit de mida de resposta**, per tant no pot penjar-se indefinidament ni exhaurir memòria del servidor | Timeout dur i límit de mida definits a Stage 2 (valors concrets, decisió tècnica); EC-03/EC-08 passen en cada execució de CI simulant un servidor lent i un servidor amb resposta excessivament gran |
| NFR-08 | sec / privadesa (contingut extern) | Cap capçalera d'autenticació, cookie de sessió, o credencial de Niu s'envia mai en la petició de recuperació de previsualització cap al destí extern | Inspecció de la petició sortint en test d'integració: 0 capçaleres d'autenticació de Niu presents cap a cap destí extern |
| NFR-09 | perf / cost (caching) | El resultat de la previsualització (títol, imatge, descripció, o l'estat de fallback) **no es torna a recuperar de la pàgina externa cada cop que es mostra la llista** — es recupera un cop en desar la idea i es reutilitza en visualitzacions posteriors | 0 peticions de xarxa sortints cap al domini extern en `GET` repetits de la llista per a una mateixa idea ja desada; verificable per compte de peticions sortints en test d'integració amb un servidor extern simulat |

> Els NFR-05 a NFR-08 concreten, en forma de llindar observable
> externament, el risc SSRF que `proposal.md` §7 flag com a `HIGH` i
> deriva explícitament a `software-architect` per a Stage 2. Aquest
> document **no** proposa mecanisme (allowlist, resolució DNS
> controlada, proxy de sortida, etc.) — només fixa què ha de ser
> observablement cert un cop la mitigació existeixi.

## 6. Estratègia de proves (redactada per `qa-engineer`)

> Piràmide Google Testing Blog: **small** (unit) per a validació de
> format d'URL i parsing de metadades Open Graph; **medium**
> (integració) per a l'API contra SQLite real i contra un servidor HTTP
> extern simulat (control total de respostes, redireccions, mides,
> latència i tipus de contingut — imprescindible per als casos SSRF,
> que no es poden ni s'han de provar contra internet real); **large**
> (E2E) només per allò que exigeix un DOM real — accessibilitat,
> diferenciació visual entre els tres espais, XSS renderitzat. Cap AC
> de seguretat es dona per bo només perquè "existeix una mitigació":
> cada test de seguretat executa l'atac i n'afirma el fracàs, mateix
> principi que `docs/test-plan.md` §2.1 i que NIU-5 §6.

| Identificador | Unit | Integració | E2E | Manual | Validació NFR |
| --- | --- | --- | --- | --- | --- |
| AC-01 | ✅ (parsing OG) | ✅ (servidor extern simulat amb OG complet) | ✅ (targeta visible al DOM) | — | — |
| AC-02 | ✅ (detecció de fallada/timeout) | ✅ (servidor extern simulat que falla/bloqueja) | ✅ (estat visual de fallback) | ⚠️ revisió puntual del missatge (`ux-ui-designer`) | — |
| AC-03 | ✅ (parsing OG parcial) | ✅ (servidor extern simulat amb OG incomplet) | ✅ (camps absents no trencats visualment) | — | — |
| AC-04 | — | ✅ (autoria a la resposta) | ✅ (visible a la UI) | — | — |
| AC-05 | — | ✅ | — | — | — |
| AC-06 | — | ✅ (dos clients simulats) | ⚠️ manual amb dos navegadors si cal explorar | ✅ | — |
| AC-07 | — | — | ✅ (comparació visual amb NIU-1/NIU-5) | ⚠️ revisió visual puntual (`ux-ui-designer`) | — |
| AC-08 | ✅ (validació de format d'URL) | ✅ | — | — | — |
| AC-09 | — | — | ✅ (navegació per Tab/Enter) | ⚠️ exploratori amb usuari real | — |
| AC-10 | — | — | ✅ (assert text alternatiu i contingut anunciat) | ⚠️ verificació puntual amb lector de pantalla real | — |
| AC-11 | — | — | — | ✅ (revisió documental — `overview.md` actualitzat, no automatitzable) | — |
| EC-01 | ✅ | ✅ (assert 0 petició sortint) | — | — | NFR-05 |
| EC-02 | ✅ (si la validació és resoluble sense xarxa) | ✅ (servidor DNS/xarxa simulat) | — | — | NFR-06 |
| EC-03 | — | ✅ (servidor extern simulat amb resposta massiva) | — | — | NFR-07 |
| EC-04 | — | ✅ (servidor extern simulat amb cadena de redireccions, incloent-hi una cap a destí prohibit) | — | — | NFR-06 |
| EC-05 | ✅ (parsing sense etiquetes OG) | ✅ | — | — | — |
| EC-06 | — | ✅ (mateix enllaç desat dues vegades, assert dues entrades) | — | — | — |
| EC-07 | — | ✅ (domini que resol a IP privada simulada) | — | — | NFR-06 |
| EC-08 | — | ✅ (servidor extern simulat amb latència superior al llindar) | — | — | NFR-07 |
| EC-09 | ✅ (detecció de content-type no HTML) | ✅ (servidor extern simulat amb PDF/imatge) | — | — | — |
| EC-10 | ✅ | ✅ | — | — | — |
| EC-11 | — | — | ✅ (assert absència d'execució de script en navegador real) | — | NFR-01 |
| EC-12 | — | ✅ (verifica taula intacta post-atac, incloent-hi via metadada recuperada) | — | — | NFR-02 |
| EC-13 | — | ✅ (assaig de la taula de rutes) | — | — | NFR-03 |
| EC-14 | — | ✅ (petició sense cookie) | — | — | NFR-04 |
| EC-15 | — | ✅ (doble DELETE) | — | — | — |
| EC-16 | — | ✅ (doble POST, assert dues entrades sense 5xx) | — | — | — |
| EC-17 | — | ✅ | ✅ (estat buit sense error) | — | — |
| EC-18 | — | — | ✅ (viewport emulat) | — | — |
| NFR-01 | ✅ (grep de codi client) | — | ✅ (EC-11 en navegador real) | — | Revisió de codi + test E2E |
| NFR-02 | — | ✅ (EC-12) | — | — | Test d'integració amb payload d'injecció, incloent-hi via metadada externa |
| NFR-03 | — | ✅ (EC-13) | — | — | Assaig estàtic de rutes en CI |
| NFR-04 | — | ✅ (EC-14) | — | — | Test d'integració executat en CI |
| NFR-05 | ✅ | ✅ (EC-01) | — | — | Test d'integració amb esquemes prohibits (`file://`, `javascript:`, `ftp://`, `data:`) |
| NFR-06 | ⚠️ pendent del mecanisme concret (Stage 2) | ✅ (EC-02, EC-04, EC-07 contra servidor/DNS simulat) | — | — | **Blocant per a `/audit`** — cap mitigació es dona per bona sense aquests tests |
| NFR-07 | — | ✅ (EC-03, EC-08 amb valors límit definits a Stage 2) | — | — | Test d'integració amb servidor lent/voluminós simulat |
| NFR-08 | — | ✅ (inspecció de capçaleres de la petició sortint) | — | — | Test d'integració amb servidor extern simulat que registra capçaleres rebudes |
| NFR-09 | — | ✅ (compte de peticions sortints en `GET` repetits) | — | — | Test d'integració: desar una idea, fer diverses lectures de la llista, assert 1 sola petició sortint al domini extern |

**Notes de cobertura:**

- **Tot test contra un "servidor extern" ha d'apuntar a un doble de
  test controlat pel propi entorn de CI (mock HTTP local), mai a
  internet real.** Això és imprescindible per als casos SSRF (EC-02,
  EC-04, EC-07): no es pot ni s'ha de provar contra xarxa privada real
  del CI ni contra webs de tercers — el doble simula el destí prohibit
  perquè el test sigui determinista i no depengui de xarxa externa.
- **NFR-06 és el punt de bloqueig d'aquest ítem a `/audit`.** Sense els
  tres tests d'integració (EC-02/EC-04/EC-07) passant contra el doble
  de xarxa, cap implementació de Stage 2 es pot considerar conforme,
  independentment de quina mitigació concreta triï `software-architect`.
- Els mecanismes de seguretat que reutilitzen patrons ja existents
  (EC-11, EC-12, EC-13, EC-14) reutilitzen exactament els mateixos
  patrons de test ja escrits per NIU-1/NIU-4/NIU-5
  (`app/tests/integration/security_test.go`) — no s'inventa cap
  infraestructura de test nova per a aquesta part.
- Els tests de SSRF (NFR-05/06/07/08) sí que requereixen infraestructura
  de test nova: un servidor HTTP de test controlable (respostes lentes,
  respostes grans, redireccions, tipus de contingut variables,
  captura de capçaleres rebudes). Es marca a `tasks.md` (Stage 3) com
  a prerequisit compartit de diversos casos, no com a feina duplicada
  per cada EC.

## 7. Fora d'abast (explícit)

- Qualsevol cicle de vida o estat (idea / planificada / feta) en v1 —
  vegeu recomanació §0.1; llista simple de desar/eliminar únicament.
- Edició manual del títol, la imatge o la descripció recuperats
  (`proposal.md` §6).
- Actualitzar la previsualització si el contingut de la pàgina
  enllaçada canvia després de desar-la (`proposal.md` §6, "Diferit").
- Reintent automàtic o cua de reprocessament quan la previsualització
  falla en el moment de desar — el fallback (§0.2/AC-02) és permanent
  per a aquella idea fins que s'esborri i es torni a afegir.
- Integració amb l'API oficial d'Instagram o de cap altra xarxa social
  per esquivar bloquejos de scraping (`proposal.md` §6).
- Cerca, filtres, o categorització de les idees desades més enllà
  d'una llista simple (`proposal.md` §6).
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos.
- Gamificació (ratxes, punts) sobre aquest espai.
- Unicitat o deduplicació d'enllaços — vegeu EC-06, confirmat com a
  pregunta oberta a §8.
- Qualsevol relació tècnica o de model de dades amb la llista de la
  compra (NIU-1) o amb compres/projectes (NIU-5) més enllà de compartir
  app i usuaris.

## 8. Preguntes obertes — RESOLTES a la porta humana (2026-08-03)

- [x] **Cicle de vida: confirmat sense estat (§0.1).** v1 és una llista
  simple — només afegir i eliminar, sense camp d'estat. AC-05 es manté
  sense canvis.
- [x] **Fallback sense previsualització: confirmat comportament no
  bloquejant (§0.2).** Desar sempre la idea, amb o sense targeta, sense
  avís de confirmació addicional. AC-02 es manté sense canvis.
- [x] **Unicitat d'enllaços: confirmat que no es dedupliquen (EC-06).**
  Es permet desar el mateix enllaç més d'un cop, sense bloqueig ni avís.
- [ ] **Llindars concrets de SSRF (NFR-06/07) i de mida/timeout
  (NFR-07): no bloquegen aquesta porta.** Són decisió de
  `software-architect` a Stage 2; aquest document només en fixa
  l'existència i l'observabilitat externa. — owner: `software-architect`
  (Stage 2, no Stage 1)
