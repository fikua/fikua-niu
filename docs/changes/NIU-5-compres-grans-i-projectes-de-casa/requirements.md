---
artefact: requirements
key: "NIU-5"
title: "Compres grans i projectes de casa"
status: "approved"
owner: "product-manager + qa-engineer"
proposal_path: "./proposal.md"
ac_count: 15
nfr_count: 8
sources:
  - "User Story format (Mike Cohn) — As a / I want / So that"
  - "INVEST — Independent, Negotiable, Valuable, Estimable, Small, Testable"
  - "Given/When/Then — Gherkin / Cucumber"
created: "2026-08-02"
updated: "2026-08-02"
---

# Requirements — Compres grans i projectes de casa

> **Què és això.** El contracte entre producte i enginyeria per a NIU-5.
> Cada criteri d'acceptació és observable des de fora del sistema i es
> traça a almenys una tasca de `tasks.md`. **Només comportament
> funcional — cap detall d'implementació.** Referència:
> [`./proposal.md`](./proposal.md).
> Font vinculant: `PLAN.md` §2.4 (filosofia de model de dades — UPDATE,
> no delete+insert; taula `events`), §3 (Seguretat), §4 (Look & feel).

## 0. Decisió recomanada sobre el model d'estats (per a la porta humana)

> `proposal.md` §9 deixa aquesta pregunta explícitament oberta.
> `qa-engineer` i `product-manager` proposem una resposta concreta aquí
> perquè la porta humana tingui alguna cosa a confirmar o corregir, en
> lloc d'una pregunta buida. **Això és una recomanació, no una decisió
> presa.**

- **Recomanació:** un cicle de vida **simple de tres estats** — `idea` →
  `decidit` → `fet` — sense graons intermedis (p. ex. sense separar
  "decidit" de "pressupostat" com a estats diferents).
- **Per què:** consistent amb el biaix d'aquest projecte cap a un abast
  mínim de v1 (`PLAN.md` §1, aplicat a NIU-1 amb només dues caixes en
  lloc d'un flux Kanban complet). Un flux amb més graons intermedis
  (p. ex. `idea` → `pressupostat` → `decidit` → `comprant` → `fet`)
  assumeix un volum i una disciplina de manteniment d'estat que dues
  persones gestionant projectes domèstics probablement no sostindran —
  el risc real (§7 de `proposal.md`) és que el model quedi
  *sobredimensionat* per a l'ús real, no pas que en falti un graó.
- **Conseqüència pràctica:** si l'ús real demostra que "decidit" amaga
  dues fases prou diferents (p. ex. "hem parlat de pressupost" vs "ja
  ho hem encarregat"), es pot afegir un quart estat en un canvi
  posterior sense trencar res existent — la taula d'`events` (§2.4
  `PLAN.md`) ja captura cada transició, així que dividir un estat en dos
  més endavant és un canvi additiu, no una migració destructiva.
- **Es confirma a la porta humana d'aquest document** (veure §8, primera
  pregunta oberta) — si el propietari prefereix un quart estat des del
  dia 1, es reescriu AC-02/AC-03 abans d'aprovar Stage 1.

## 1. Historial d'usuari

- **Com a** membre de la parella que utilitza Niu
- **Vull** anotar una idea de compra gran o projecte de casa, seguir-ne
  l'estat fins que es decideixi i finalment es faci, i veure sempre en
  quin punt està i qui l'ha actualitzat per últim cop
- **Perquè** cap idea es perdi en una conversa ni s'hagi de repetir
  perquè cap dels dos recorda si ja se n'havia parlat.

## 2. Autoavaluació INVEST

- [x] **Independent** — ✅ No depèn de cap altre ítem en curs; comparteix
  només l'app i els dos usuaris amb NIU-1, no el seu model de dades ni
  el seu cicle de vida (`proposal.md` §6, "fora d'abast").
- [x] **Negotiable** — ✅ El model d'estats exacte (§0) i els camps
  addicionals (§0 de `proposal.md`, pregunta oberta) són negociables a
  la porta humana d'aquest document; el contracte extern (llista
  d'elements amb estat i autoria) és fix.
- [x] **Valuable** — ✅ Elimina un fregament domèstic real i recurrent
  (`proposal.md` §2), amb una mesura d'èxit qualitativa clara.
- [x] **Estimable** — ✅ Abast tancat un cop es resolguin les preguntes
  de §8: CRUD + transicions d'estat + UI diferenciada de NIU-1.
- [x] **Small** — ✅ Encaixa en un sol cicle de `/define` → `/code` →
  `/audit`, assumint el model de 3 estats recomanat a §0.
- [x] **Testable** — ✅ Cada AC és observable via l'API HTTP o el DOM
  renderitzat; cap AC depèn d'inspeccionar estat intern no exposat.

## 3. Criteris d'acceptació

### AC-01 — Afegir una idea nova

- **Given** l'espai de compres grans / projectes té zero o més elements
- **When** l'usuari afegeix un element nou amb un nom vàlid
- **Then** l'element apareix amb estat inicial `idea`, persisteix després
  de recarregar la pàgina, i mostra qui l'ha afegit

### AC-02 — Marcar una idea com a decidida

- **Given** un element existeix amb estat `idea`
- **When** l'usuari el marca com a `decidit`
- **Then** l'element passa a l'estat `decidit`, amb l'autor i el moment
  del canvi actualitzats, i continua sent visible i editable

### AC-03 — Marcar un element decidit com a fet

- **Given** un element existeix amb estat `decidit`
- **When** l'usuari el marca com a `fet`
- **Then** l'element passa a l'estat `fet`, amb l'autor i el moment del
  canvi actualitzats

### AC-04 — Cada element mostra qui l'ha tocat i quan

- **Given** un element ha estat afegit o ha canviat d'estat
- **When** es mostra l'element a la interfície
- **Then** és visible qui l'ha afegit i, si ha canviat d'estat, qui ha
  fet el darrer canvi i quan

### AC-05 — Eliminar un element

- **Given** un element existeix en qualsevol estat
- **When** l'usuari l'elimina explícitament
- **Then** l'element desapareix de la llista i no reapareix en
  recarregar

### AC-06 — Dos usuaris veuen el mateix estat (convergència eventual)

- **Given** un usuari afegeix un element o li canvia l'estat
- **When** l'altre usuari recarrega la pàgina, torna el focus a la
  finestra, o espera l'interval de sondeig ja establert per NIU-1 (~10s)
- **Then** veu l'element i el seu estat actualitzat

### AC-07 — Canvi d'estat concurrent del mateix element convergeix sense error

- **Given** els dos usuaris tenen el mateix element visible
- **When** tots dos li canvien l'estat gairebé simultàniament (potser a
  estats diferents)
- **Then** cap de les dues peticions falla amb error de servidor, i
  després del següent refresc/sondeig ambdós clients mostren el mateix
  estat final (l'última escriptura acceptada pel servidor)

### AC-08 — Espai visualment diferenciat de la llista de la compra

- **Given** l'usuari navega per Niu
- **When** entra a l'espai de compres grans / projectes de casa
- **Then** distingeix clarament, només pel visual, que no és la llista
  de la compra (NIU-1) — mantenint la mateixa estètica càlida
  (`PLAN.md` §4)

### AC-09 — Retrocedir un estat és possible

- **Given** un element té estat `decidit` o `fet`
- **When** l'usuari el torna a un estat anterior (p. ex. de `decidit` a
  `idea`, o de `fet` a `decidit`)
- **Then** l'element torna a l'estat anterior, amb l'autor i el moment
  del canvi actualitzats, sense perdre l'historial de qui el va afegir
  originalment

### AC-10 — Nom d'element obligatori i acotat

- **Given** el formulari d'afegir element
- **When** l'usuari hi introdueix un nom vàlid (1–200 caràcters després
  de retallar espais, seguint el mateix llindar que NIU-1)
- **Then** l'element s'accepta i es desa sencer

### AC-11 — Navegació completa per teclat

- **Given** l'usuari no utilitza ratolí ni pantalla tàctil
- **When** navega l'espai únicament amb teclat
- **Then** pot afegir un element, canviar-ne l'estat (en qualsevol
  direcció) i eliminar-lo sense necessitar cap altre dispositiu d'entrada

### AC-12 — Anunci per lectors de pantalla en canviar d'estat

- **Given** l'usuari utilitza un lector de pantalla
- **When** un element canvia d'estat (per acció pròpia o per sondeig que
  reflecteix un canvi remot)
- **Then** una regió `aria-live` anuncia el canvi de manera comprensible
  (nom de l'element i nou estat)

### AC-13 — `overview.md` reflecteix el nou espai

- **Given** aquest ítem queda aprovat i implementat
- **Then** `docs/overview.md` s'actualitza per esmentar l'espai de
  compres grans / projectes de casa com a funcionalitat existent de Niu,
  mantenint-lo com a font única de veritat del que fa l'app
  (`proposal.md` §9)

> AC-13 no té un "When" d'interacció d'usuari — és una condició de
> tancament documental, no una interacció observable des de la UI/API.
> Es manté com a AC perquè `proposal.md` §9 la planteja explícitament
> com a acció a confirmar, no com un simple detall d'implementació.

## 4. Casos límit i escenaris negatius

### EC-01 — Nom buit o només espais en blanc

- **Given** el formulari d'afegir element
- **When** l'usuari envia un nom buit o compost únicament per
  espais/tabulacions
- **Then** la petició es rebutja amb un missatge d'error clar i cap
  element es crea

### EC-02 — Nom al límit de longitud (200 / 201 caràcters)

- **Given** el formulari d'afegir element
- **When** l'usuari envia un nom d'exactament 200 caràcters després de
  retallar espais
- **Then** s'accepta sencer
- **Given** el mateix formulari
- **When** l'usuari envia un nom de 201 caràcters
- **Then** es rebutja amb un missatge d'error clar

### EC-03 — Nom d'idea duplicat

- **Given** ja existeix un element actiu (no eliminat) amb un nom que,
  retallat i comparat sense distingir majúscules/minúscules, coincideix
  amb el nom introduït (p. ex. ja existeix `"Televisor nou"` en
  qualsevol estat)
- **When** l'usuari intenta afegir `"televisor nou"`, `"Televisor nou "`,
  o `"TELEVISOR NOU"` com a idea nova
- **Then** la petició es rebutja amb un missatge clar indicant que
  l'element ja existeix, **independentment de l'estat en què es trobi**
  (una idea ja `decidida` o ja `feta` també compta com a duplicat —
  reobrir-la és canviar-li l'estat, no crear-ne una de nova)

> **Nota de disseny (no d'implementació):** aquesta regla és
> deliberadament més àmplia que la de NIU-1 (que només compara entre
> dues caixes actives). Aquí, un element `fet` continua bloquejant un
> duplicat exacte perquè el cas d'ús típic ("ja hem comprat el
> televisor, algú se n'oblida i el torna a proposar") és exactament el
> fregament que aquest ítem vol eliminar (`proposal.md` §2). Si
> l'usuari vol tornar a plantejar un projecte ja fet (p. ex. "repintar"
> anys després), utilitza AC-09 per reobrir-lo o bé tria un nom
> lleugerament diferent — aquest document no resol automàticament
> aquesta ambigüitat semàntica.

### EC-04 — Duplicat exacte permès després d'eliminar l'original

- **Given** un element amb un nom concret ha estat eliminat (AC-05)
- **When** l'usuari afegeix un element amb el mateix nom
- **Then** s'accepta com un element nou (la comprovació de duplicats
  (EC-03) només mira elements actius, no elements eliminats)

### EC-05 — Element estancat indefinidament a `decidit`

- **Given** un element porta setmanes o mesos a l'estat `decidit` sense
  passar a `fet` ni tornar a `idea`
- **When** qualsevol dels dos usuaris el consulta
- **Then** l'element continua sent visible amb el seu estat i la data
  del darrer canvi, **sense** cap caducitat automàtica, arxivament ni
  ocultació silenciosa — la visibilitat continuada és precisament el
  mecanisme que evita l'oblit (`proposal.md` §2); la decisió de
  reactivar-lo, avançar-lo o eliminar-lo és sempre humana i explícita

> Aquest EC confirma per escrit el que `proposal.md` §7 ja marca com a
> risc de severitat baixa i explícitament fora d'abast (sense
> caducitat automàtica). Es documenta aquí com a comportament
> **esperat**, no com una llacuna: un element estancat és, per
> disseny, responsabilitat dels usuaris, no del sistema.

### EC-06 — Eliminar una idea vs. marcar-la abandonada

- **Given** un element que ja no interessa tirar endavant (p. ex. una
  idea rebutjada, no un projecte fet)
- **When** l'usuari decideix què fer-ne
- **Then** l'única acció disponible en aquesta versió és l'eliminació
  (AC-05) — **no existeix un estat "abandonat/descartat" diferenciat en
  v1**; eliminar un element el treu completament de la llista visible

> Aquesta és una decisió d'abast explícita, no un oblit: `proposal.md`
> no menciona un estat d'abandonament, i afegir-ne un quart trencaria
> la simplicitat recomanada a §0. Si l'ús real demostra que "vam
> decidir no fer-ho, però volem recordar que hi vam pensar" és un cas
> prou freqüent, és una petita extensió per a un canvi posterior (un
> quart estat `descartat`, additiu sobre l'`events` existent) — no
> s'introdueix aquí. Es marca com a pregunta oberta a §8 perquè el
> propietari confirmi que eliminar n'hi ha prou per a v1.

### EC-07 — Format del camp de pressupost, si s'inclou

- **Given** la pregunta oberta de `proposal.md` §9 sobre si calen camps
  addicionals (pressupost, notes, data objectiu)
- **When** es decideix incloure un camp de pressupost en aquesta versió
- **Then** ha de quedar explícitament decidit **abans de Stage 2** si es
  tracta d'un **camp de text lliure** (p. ex. "uns 300€, potser més si
  triem el moble bo") o d'un **camp numèric estructurat** (una xifra amb
  moneda implícita) — les dues opcions tenen implicacions de validació i
  de test diferents (un camp numèric necessita EC de format i de
  frontera; un camp de text lliure només necessita el mateix llindar de
  longitud que el nom)

> Aquest EC no prescriu una resposta: és el marcador exacte que
> `qa-engineer` necessita per saber quins casos de prova addicionals
> caldria escriure si aquest camp s'aprova. Recomanació pròpia (no
> vinculant): si s'inclou, un **camp de text lliure opcional** és
> coherent amb el biaix de mínim abast de §0 — evita decidir moneda,
> validació numèrica i arrodoniment per a un ús que, amb tota
> probabilitat, prefereix escriure "unes 300, a veure" tal com ja fa
> NIU-1 amb la quantitat dins del nom (`PLAN.md` §2.4, "no quantity
> column"). Vegeu pregunta oberta a §8.

### EC-08 — Injecció HTML/JS al nom de l'element (XSS)

- **Given** el formulari d'afegir element
- **When** l'usuari envia un nom com `<img src=x onerror=alert(1)>`
- **Then** el nom es desa i es mostra com a **text literal** (no
  s'executa cap script, no es renderitza cap element HTML) — mateix
  estàndard que NIU-1 (S3)

### EC-09 — Injecció SQL al nom de l'element

- **Given** el formulari d'afegir element
- **When** l'usuari envia un nom com `'; DROP TABLE items;--` (o
  equivalent contra la taula pròpia d'aquest ítem)
- **Then** el nom es desa literalment com a text, la resta de dades de
  l'aplicació roman intacta, i cap taula desapareix — mateix estàndard
  que NIU-1 (S8)

### EC-10 — Intent de mutació via `GET`

- **Given** l'API exposa endpoints per a aquest espai
- **When** un client intenta provocar una mutació (crear, canviar
  estat, eliminar) mitjançant una petició `GET`
- **Then** no existeix cap ruta `GET` amb efectes secundaris — mateix
  estàndard que NIU-1 (S1a)

### EC-11 — Accés sense sessió autenticada

- **Given** cap cookie de sessió vàlida present a la petició
- **When** es crida qualsevol endpoint d'aquest espai
- **Then** la petició es rebutja com a no autenticada, exactament amb
  el mateix mecanisme ja establert per NIU-4 — aquest ítem no
  introdueix cap excepció ni ruta pública nova

### EC-12 — Canviar l'estat d'un element ja eliminat

- **Given** un element ha estat eliminat (per aquest usuari o per
  l'altre, ja convergit)
- **When** un client amb estat desactualitzat intenta canviar-li l'estat
- **Then** la petició respon amb un error clar sense afectar altres
  elements, i el client pot refrescar per recuperar l'estat real

### EC-13 — Eliminar un element ja eliminat (doble clic / reenviament)

- **Given** un element ja ha estat eliminat
- **When** una segona petició d'eliminació pel mateix element arriba al
  servidor
- **Then** la petició no provoca un error 5xx ni corromp l'estat;
  l'element continua absent (operació idempotent)

### EC-14 — Llista buida en primer ús

- **Given** l'espai s'obre per primer cop sense cap element
- **When** l'usuari hi entra
- **Then** es mostra un estat visual clar de "res per mostrar", sense
  error i sense confondre's amb un error de càrrega

### EC-15 — Viewport mòbil

- **Given** l'usuari obre aquest espai en una pantalla estreta (mòbil)
- **When** interactua amb la interfície
- **Then** el contingut s'adapta mantenint totes les funcionalitats
  (afegir, canviar estat, eliminar), seguint el mateix patró responsive
  que NIU-1

## 5. Requisits no funcionals (NFR)

| ID | Categoria | Enunciat | Objectiu / llindar |
| --- | --- | --- | --- |
| NFR-01 | integritat de dades | Cap canvi d'estat esborra ni sobreescriu la història d'autoria d'un element — segueix la filosofia UPDATE-no-delete+insert de `PLAN.md` §2.4 | 100% dels canvis d'estat queden reflectits com a esdeveniment a la taula d'`events` (o equivalent), verificable per inspecció directa després de cada transició d'AC-02/AC-03/AC-09 |
| NFR-02 | sec (XSS) | Cap nom d'element ni cap altra dada d'usuari es renderitza mai com a HTML | Zero ocurrències d'`innerHTML` amb dades d'usuari al codi client; EC-08 passa en cada execució de CI |
| NFR-03 | sec (injecció SQL) | Cap valor introduït per l'usuari es concatena mai a una sentència SQL | 100% de les consultes amb entrada d'usuari utilitzen paràmetres vinculats; EC-09 passa en cada execució de CI |
| NFR-04 | sec (mutació via GET) | Cap mutació és accessible via `GET` | Assaig de la taula de rutes: 0 rutes `GET` amb efecte d'escriptura (EC-10) |
| NFR-05 | sec (autenticació) | Tots els endpoints d'aquest espai requereixen sessió vàlida, reutilitzant el mecanisme de NIU-4 sense excepcions | EC-11 passa en cada execució de CI; 0 endpoints d'aquest ítem exempts del middleware d'autenticació |
| NFR-06 | a11y | Totes les pantalles d'aquest espai compleixen contrast AA i són operables per teclat, consistent amb el llistó ja fixat per NIU-1 | WCAG 2.2 AA verificat per a totes les combinacions text/fons definides a `PLAN.md` §4; 100% de les accions (afegir, canviar estat, eliminar) accessibles sense ratolí (AC-11) |
| NFR-07 | a11y | Els canvis d'estat rellevants s'anuncien a tecnologies d'assistència | Regió `aria-live="polite"` (o equivalent) actualitzada en cada canvi d'estat, verificat amb lector de pantalla real o eina d'auditoria (AC-12) |
| NFR-08 | a11y (moviment) | Si aquest espai incorpora qualsevol transició animada entre estats, respecta `prefers-reduced-motion` igual que NIU-1 | 0 animacions de moviment/vol quan `prefers-reduced-motion` està actiu — es verifica només si es dissenya alguna transició animada a Stage 1.5; si l'espai es resol sense animació de transició (llista simple), aquest NFR es documenta com a no aplicable a `design.md` |

## 6. Estratègia de proves (redactada per `qa-engineer`)

> Piràmide Google Testing Blog: **small** (unit) per validació de nom i
> normalització de duplicats; **medium** (integració) per l'API contra
> SQLite real, seguint exactament el patró ja establert a
> `app/tests/integration/` de NIU-1/NIU-4; **large** (E2E) només per
> allò que exigeix un DOM real — accessibilitat, diferenciació visual,
> XSS renderitzat. Cap AC de seguretat es dona per bo només perquè
> "existeix una mitigació": cada test de seguretat executa l'atac i
> n'afirma el fracàs, mateix principi que `docs/test-plan.md` §2.1.

| Identificador | Unit | Integració | E2E | Manual | Validació NFR |
| --- | --- | --- | --- | --- | --- |
| AC-01 | ✅ (validació de nom) | ✅ (persistència real) | — | — | — |
| AC-02 | — | ✅ | — | — | — |
| AC-03 | — | ✅ | — | — | — |
| AC-04 | — | ✅ (autoria a la resposta) | ✅ (visible a la UI) | — | — |
| AC-05 | — | ✅ | — | — | — |
| AC-06 | — | ✅ (dos clients simulats) | ⚠️ manual amb dos navegadors si cal explorar | ✅ | — |
| AC-07 | — | ✅ (peticions concurrents simulades) | — | — | — |
| AC-08 | — | — | ✅ (comparació visual amb NIU-1) | ⚠️ revisió visual puntual (`ux-ui-designer`) | — |
| AC-09 | — | ✅ (totes les transicions vàlides, ambdues direccions) | — | — | — |
| AC-10 | ✅ | ✅ | — | — | — |
| AC-11 | — | — | ✅ (navegació per Tab/Enter) | ⚠️ exploratori amb usuari real | — |
| AC-12 | — | — | ✅ (assert contingut `aria-live`) | ⚠️ verificació puntual amb lector de pantalla real | — |
| AC-13 | — | — | — | ✅ (revisió documental — `overview.md` actualitzat, no automatitzable) | — |
| AC-14 | ✅ (validació de longitud) | ✅ (persistència, camp opcional) | — | — | — |
| AC-15 | — | ✅ (persistència, camp opcional) | — | — | — |
| EC-01 | ✅ | ✅ | — | — | — |
| EC-02 | ✅ | ✅ | — | — | — |
| EC-03 | ✅ (normalització trim+lowercase) | ✅ (contra elements reals en qualsevol estat) | — | — | — |
| EC-04 | — | ✅ | — | — | — |
| EC-05 | — | ✅ (verifica absència de canvi automàtic passat un llarg període simulat) | — | ⚠️ (confirmació que no hi ha job/cron ocult) | — |
| EC-06 | — | ✅ (confirma que no existeix estat `abandonat`, només `DELETE`) | — | — | — |
| EC-07 | ⚠️ pendent de decisió — cas real d'unit/integració es defineix a Stage 2 un cop es resolgui format (§8) | — | — | — | — |
| EC-08 | — | — | ✅ (assert absència d'execució de script en navegador real) | — | — |
| EC-09 | — | ✅ (verifica taula intacta post-atac) | — | — | — |
| EC-10 | — | ✅ (assaig de la taula de rutes) | — | — | — |
| EC-11 | — | ✅ (petició sense cookie) | — | — | — |
| EC-12 | — | ✅ | — | — | — |
| EC-13 | — | ✅ (doble DELETE) | — | — | — |
| EC-14 | — | ✅ | ✅ (estat buit sense error) | — | — |
| EC-15 | — | — | ✅ (viewport emulat) | — | — |
| EC-16 | ✅ | ✅ | — | — | — |
| EC-17 | — | ✅ | — | — | — |
| NFR-01 | — | ✅ (assert fila d'`events` per cada transició) | — | — | Test d'integració a cada AC de canvi d'estat |
| NFR-02 | ✅ (grep de codi client) | — | ✅ (EC-08 en navegador real) | — | Revisió de codi + test E2E |
| NFR-03 | — | ✅ (EC-09) | — | — | Test d'integració amb payload d'injecció |
| NFR-04 | — | ✅ (EC-10) | — | — | Assaig estàtic de rutes en CI |
| NFR-05 | — | ✅ (EC-11) | — | — | Test d'integració executat en CI |
| NFR-06 | — | — | — | ✅ | Auditoria automatitzada (axe-core o equivalent) + revisió manual puntual, mateix mecanisme que NIU-1 |
| NFR-07 | — | — | ✅ (AC-12) | — | Test E2E que llegeix el contingut d'`aria-live` |
| NFR-08 | — | — | ⚠️ condicionat al disseny visual de Stage 1.5 | — | Documentat com a no aplicable a `design.md` si no hi ha animació de transició |

**Notes de cobertura:**

- EC-07 (format del camp de pressupost) queda resolt: text lliure,
  mateix llindar de longitud que el nom (veure EC-16). `task-planner` ja
  pot generar tasques per a AC-14/AC-15/EC-16/EC-17.
- NFR-08 depèn d'una decisió visual (`ux-ui-designer`, Stage 1.5): si el
  canvi d'estat es dissenya com una simple actualització de text/badge
  sense animació de moviment, aquest NFR es marca com a no aplicable
  sense necessitat de cap test — es documenta explícitament a
  `design.md` per no deixar-lo com una ambigüitat silenciosa.
- Els mecanismes de seguretat (EC-08, EC-09, EC-10, EC-11) reutilitzen
  exactament els mateixos patrons de test ja escrits per NIU-1/NIU-4
  (`app/tests/integration/security_test.go`), aplicats a les noves rutes
  d'aquest ítem — no s'inventa cap infraestructura de test nova.

## 7. Fora d'abast (explícit)

- Integració amb comerços, cercadors de preus o enllaços a productes
  concrets (`proposal.md` §6).
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos.
- Gamificació (ratxes, punts) sobre aquest espai.
- Qualsevol relació tècnica o de model de dades amb la llista de la
  compra (NIU-1) més enllà de compartir app i usuaris — són col·leccions
  independents.
- Vista d'historial o d'anàlisi sobre projectes passats més enllà de
  l'estat actual de cada element (`proposal.md` §6, "Diferit").
- Caducitat automàtica, arxivament o ocultació d'elements estancats a
  `decidit` (EC-05) — comportament esperat, no llacuna.
- Estat diferenciat d'"abandonat/descartat" — eliminar és l'única acció
  disponible en v1 (EC-06), confirmat a §8.
- Camp de notes lliures — descartat a la porta humana; només pressupost
  (text lliure) i data objectiu s'inclouen en v1 (AC-14, AC-15).

## 8. Preguntes obertes — RESOLTES a la porta humana (2026-08-02)

- [x] **Model d'estats: 3 estats simples.** Confirmat: `idea` → `decidit`
  → `fet`, sense graons intermedis, tal com recomanava §0. AC-02/AC-03
  es mantenen sense canvis.
- [x] **Camps addicionals: pressupost + data objectiu, sí.** El
  propietari ha confirmat que aquesta primera versió inclou **totes
  dues**: un camp de pressupost i un camp de data objectiu (a banda de
  nom i estat). Format del pressupost: **text lliure** (recomanació de
  §0/EC-07 confirmada), coherent amb la decisió ja presa a NIU-1 de no
  crear una columna numèrica de quantitat (`PLAN.md` §2.4). La data
  objectiu és una data simple (sense hora), opcional, sense validació
  més enllà de ser una data vàlida. Veure AC-14/AC-15 i EC-16/EC-17 nous
  més avall.
- [x] **Estat "abandonat/descartat": no, només eliminar.** Confirmat tal
  com assumia EC-06 — v1 no té estat diferenciat d'abandonament;
  eliminar és l'única acció.
- [x] **Actualització d'`overview.md` (AC-13).** Resolt a `proposal.md`
  §9: es confirma en aprovar aquesta proposta, no és un bloqueig. Es
  manté com AC-13 perquè és una condició de tancament explícita de
  l'ítem, no un detall implícit.

### AC-14 — Afegir pressupost opcional (text lliure)

- **Given** el formulari d'afegir o editar un element
- **When** l'usuari hi introdueix un text de pressupost (p. ex. "uns
  300€, potser més si triem el moble bo"), amb el mateix llindar de
  longitud que el nom (1–200 caràcters) o el deixa buit
- **Then** el valor es desa i es mostra sencer; si es deixa buit, no es
  mostra cap camp de pressupost a l'element

### AC-15 — Afegir data objectiu opcional

- **Given** el formulari d'afegir o editar un element
- **When** l'usuari hi selecciona una data vàlida o el deixa buit
- **Then** la data es desa i es mostra en el format de data ja establert
  a la resta de Niu; si es deixa buida, no es mostra cap data a l'element

### EC-16 — Pressupost al límit de longitud (200 / 201 caràcters)

- **Given** el formulari amb el camp de pressupost
- **When** l'usuari envia un text de pressupost de 201 caràcters
- **Then** es rebutja amb un missatge d'error clar (mateix llindar que
  EC-02 aplicat al nom)

### EC-17 — Data objectiu en el passat

- **Given** el formulari amb el camp de data objectiu
- **When** l'usuari selecciona una data anterior a avui
- **Then** s'accepta sense error — una data objectiu passada és
  informació vàlida (p. ex. un projecte endarrerit), no una entrada
  invàlida; el sistema no imposa cap restricció temporal
