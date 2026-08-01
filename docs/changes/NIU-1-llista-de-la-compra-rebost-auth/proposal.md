---
artefact: proposal
key: "NIU-1"
type: "story"
title: "Llista de la compra ↔ rebost (auth stubbed)"
status: "draft"
owner: "product-manager"
parent_key: null
related_keys: ["NIU-2", "NIU-3", "NIU-4"]
sources:
  - "Lean Canvas (Ash Maurya) — problem/solution framing"
  - "Amazon PR/FAQ — narrative front half"
created: "2026-08-01"
updated: "2026-08-01"
---

# Proposta — Llista de la compra ↔ rebost (auth stubbed)

> **Què és això.** Narrativa d'una pàgina que emmarca el problema, la
> solució proposada, a qui serveix i el valor que aporta. Lectura en
> menys de 3 minuts.

## 1. Titular

Niu substitueix el WhatsApp i la memòria per una llista de la compra
compartida que sempre diu la veritat sobre què falta i què ja hi ha a
casa.

## 2. Problema

- Una parella gestiona avui què falta comprar i què ja hi ha al rebost a
  través de mitjans que no estan pensats per a això: notes mentals,
  paperets solts, fils de WhatsApp que s'esborren o es perden entre
  altres converses.
- No hi ha una única font de veritat: cadascú pot tenir una versió
  diferent de "què falta", i la única manera de resoldre el dubte és
  obrir físicament l'armari o la nevera.
- El cost no és catastròfic però és constant: es compren duplicats, es
  descobreix al súper que no es recorda si falta oli, o es torna a casa
  sense allò que sí que faltava.
- El problema no es resoldrà sol perquè cap de les eines actuals
  (WhatsApp, notes) està dissenyada per representar un estat compartit
  de "què tenim" — només per comunicar, no per registrar.

## 3. Client

- **Primari:** Usuari A i Usuari B — les dues úniques persones que
  utilitzaran Niu, en peu d'igualtat. No hi ha rol d'administrador ni
  jerarquia entre elles.
- **Secundari:** cap. Niu és una app privada per a exactament dues
  persones (vegeu [overview.md](../../overview.md) §"Per a qui") i no té
  ambició de créixer a més usuaris.

## 4. Solució proposada

Una llista de la compra compartida amb dues caixes visibles alhora, **A
comprar** i **Rebost**: seleccionar un ítem el mou d'una caixa a l'altra
amb una animació immediata i sense fricció. Els dos usuaris veuen la
mateixa llista actualitzada en qüestió de segons i cada ítem mostra qui
l'ha afegit o mogut per últim cop. Per fer possible aquesta primera
versió sense construir encara l'autenticació real (NIU-4), l'usuari
actual és fix (stubbed): l'app sap qui ets sense necessitat de fer login,
i el mecanisme real de login es podrà endollar més endavant sense canviar
com es comporta la llista.

## 5. Valor i mesura d'èxit

- **Valor:** experiència d'usuari (UX) — elimina un fregament domèstic
  petit però diari, substituint un canal de comunicació genèric
  (WhatsApp) per una eina pensada exactament per a aquest estat
  compartit.
- **Mesura d'èxit:** ús real i sostingut al súper sense necessitat de
  tornar a WhatsApp per confirmar dubtes sobre la llista. Com que Niu no
  té analítica de producte ([overview.md](../../overview.md)
  §"Com sabrem que funciona"), l'única mesura vàlida és qualitativa: els
  dos usuaris fan servir Niu com a única font de veritat durant almenys
  dues setmanes consecutives d'ús normal, sense recórrer a WhatsApp per a
  aquest propòsit.

## 6. Abast i fora d'abast

**En abast**

- Dues caixes — **A comprar** i **Rebost** — amb moviment d'ítems entre
  elles en una sola acció.
- Afegir, moure i eliminar ítems; cada ítem mostra qui l'ha afegit i qui
  l'ha mogut per últim cop.
- Els dos usuaris veuen canvis fets per l'altre en un termini curt sense
  necessitat de recarregar manualment.
- Usuari actual determinat de forma fixa (stubbed), sense pantalla de
  login. Tot el disseny d'aquesta llista ha de ser independent del
  mecanisme d'autenticació que arribi després.
- Experiència visual completa i definitiva (no un esborrany): l'estètica
  càlida descrita a [PLAN.md](../../../PLAN.md) §4, animació de moviment
  entre caixes, actualització immediata en tocar un ítem.
- Accessibilitat de primer nivell: navegació completa per teclat,
  contrast suficient per llegir còmodament, anunci dels canvis per a
  lectors de pantalla, i respecte per la preferència de moviment reduït.
- Confirmació visual quan la llista de la compra queda buida (una
  petita celebració).
- Protecció bàsica de la informació introduïda pels usuaris: cap dada
  introduïda per un usuari s'ha d'interpretar mai com a codi executable,
  i el sistema no ha de filtrar mai detalls tècnics interns en cas
  d'error.
- Persistència: la llista sobreviu a reinicis del servei sense pèrdua de
  dades.
- Registre intern de cada acció (qui ha afegit, mogut o eliminat un
  ítem) encara que aquesta v1 no en mostri cap ús directe més enllà de
  l'atribució per avatar — és la base per a possibles funcionalitats
  futures ([overview.md](../../overview.md) §"Futur possible").

**Fora d'abast (explícit)**

- Autenticació real (pantalla de login, contrasenyes, sessions
  revocables) — és NIU-4.
- Registre d'usuaris nous, recuperació de contrasenya, gestió de rols o
  permisos — explícitament exclòs de tot Niu, no només d'aquesta versió
  ([overview.md](../../overview.md) §"Què no fa la v1").
- Notificacions push.
- Camp numèric de quantitat: si cal indicar quantitat, s'escriu dins del
  mateix nom de l'ítem (p. ex. "2 llets"). Decisió deliberada per
  mantenir la llista simple.
- Desplegament, infraestructura, CI/CD i observabilitat — són NIU-2 i
  NIU-3.
- Actualització en temps real via connexió persistent (p. ex. events en
  viu) — es descarta explícitament per a la v1 a favor d'un mecanisme
  més senzill (§7).
- Tasques de casa, planificació de menús, despeses compartides,
  gamificació amb ratxes o punts — futur no compromès.

**Diferit a un canvi posterior**

- Substitució del mecanisme d'usuari fix per autenticació real amb
  login/logout (NIU-4), reutilitzant el mateix contracte de "qui sóc jo
  ara" que aquesta història estableix.
- Qualsevol capa de gamificació més enllà de l'atribució per avatar i la
  celebració en buidar la llista.

## 7. Riscos i incògnites

| Risc / incògnita | Severitat | Hipòtesi de mitigació |
| ---------------- | -------- | ---------------------- |
| Ítems duplicats (mateix nom en majúscules/minúscules o espais diferents) creant confusió sobre si cal comprar-lo o ja hi és | MEDIUM | **Resolt en aquesta proposta:** afegir un ítem el nom del qual ja existeix (sense distingir majúscules ni espais als extrems) a qualsevol de les dues caixes es rebutja amb un missatge clar. Decisió deliberada: preferim una llista neta a permetre entrades redundants. |
| Dos usuaris movent el mateix ítem gairebé alhora poden generar una percepció d'inconsistència momentània | LOW | Cada moviment estableix un estat absolut (no alterna), de manera que encara que hi hagi una col·lisió, els dos clients acaben convergint al mateix estat en la següent actualització. |
| Actualització basada en interrogació periòdica (no instantània) pot fer que un usuari vegi un canvi de l'altre amb uns segons de retard | LOW | **Resolt en aquesta proposta:** interrogació cada ~10 segons més actualització en tornar el focus a la finestra és suficient per a dues persones i evita la complexitat d'un mecanisme de notificació en viu. Es revisarà només si l'ús real ho fa sentir lent. |
| L'estètica càlida i l'animació de moviment podrien alentir la percepció de rapidesa si no s'implementen amb cura | LOW | Actualització optimista (el moviment es veu immediatament, abans de confirmar amb el servidor) i animació curta (~250ms). |
| Absència d'autenticació real deixa l'app potencialment exposada si es desplega públicament abans de NIU-4 | MEDIUM | Fora de l'abast funcional d'aquesta història, però mitigat operativament a NIU-2 amb Cloudflare Access ([PLAN.md](../../../PLAN.md) §3, fila S10). Aquesta proposta no depèn de la mitigació, però la referencia perquè l'ordre d'execució (§8) en depèn. |

## 8. Visuals

> **Font única d'aquesta secció.** No existeix cap sistema de disseny previ
> ni cap Figma per a Niu — aquesta secció **és** el sistema de disseny de
> Niu v1, en format ASCII/Markdown (regla de l'agent: una sola font per
> pantalla, mai duplicada). `fullstack-developer` implementa directament a
> partir d'aquí; `software-architect` la consumeix a l'Etapa 2 per
> dimensionar components i contractes. Tots els valors són concrets — cap
> adjectiu sense xifra darrere.

### 8.1 Design tokens

Totes les combinacions text/fons per sota s'han comprovat aritmèticament
(fórmula de luminància relativa WCAG 2.2, algorisme sRGB→lineal estàndard).
El llindar AA és **4.5:1** per a text normal i **3:1** per a text gran
(≥24px, o ≥19px en negreta) i per a components d'interfície/estat de
focus.

**Colors base**

| Token | Valor | Ús |
|---|---|---|
| `color.bg` | `#FBF6EC` | Fons general de la pàgina (crema) |
| `color.surface` | `#FFFFFF` | Fons de les caixes ("A comprar", "Rebost"), targetes, toast |
| `color.surface-soft` | `#F3ECDD` | Fons de la fila d'ítem en repòs (lleugerament diferenciada de `surface`) |
| `color.border` | `#DCD2BC` | Vores de caixes, inputs, separadors |
| `color.text-primary` | `#2E2A22` | Text principal (noms d'ítem, títols de caixa) |
| `color.text-secondary` | `#5C5546` | Text secundari (marques de temps, ajuda, placeholder) |
| `color.moss` | `#3F6B4A` | Verd molsa — acció primària (caixa "Rebost", botó afegir, indicador de moviment cap a rebost) |
| `color.moss-hover` | `#355C3F` | Estat `:hover` de `moss` |
| `color.moss-active` | `#2C4C34` | Estat `:active`/premut de `moss` |
| `color.terracotta` | `#C1552C` | Terracota — accent (caixa "A comprar", indicador de moviment cap a comprar). **Només sobre superfícies grans o text ≥24px** (vegeu 8.1.1) |
| `color.terracotta-hover` | `#A6431F` | Estat `:hover` de `terracotta`. També és el token de **text** terracota sobre `color.bg`/`color.surface` (vegeu 8.1.1) |
| `color.terracotta-active` | `#8C3719` | Estat `:active`/premut de `terracotta` |
| `color.focus-ring` | `#1F5FA8` | Anell de focus visible (blau, deliberadament fora de la paleta càlida — necessita destacar sobre crema, verd i terracota alhora) |
| `color.error` | `#A6341C` | Text i icona d'error (validació, toast de fallada) |
| `color.error-bg` | `#F6DFD7` | Fons subtil per a missatges d'error inline |
| `color.success-bg` | `#E4EFE3` | Fons subtil per a confirmacions (no usat en v1 més enllà del toast de reversió) |

**8.1.1 Taula de contrast (AA, calculada)**

| Parell | Ràtio | Llindar aplicable | Resultat |
|---|---|---|---|
| `text-primary` sobre `bg` | 13.26:1 | 4.5:1 | ✅ AAA |
| `text-primary` sobre `surface` | 14.28:1 | 4.5:1 | ✅ AAA |
| `text-secondary` sobre `bg` | 6.86:1 | 4.5:1 | ✅ AA |
| `text-secondary` sobre `surface` | 7.39:1 | 4.5:1 | ✅ AA |
| `moss` sobre `bg` (text/icona) | 5.71:1 | 4.5:1 | ✅ AA |
| Blanc sobre `moss` (botó ple) | 6.15:1 | 4.5:1 | ✅ AA |
| Blanc sobre `moss-hover` | 7.63:1 | 4.5:1 | ✅ AA |
| Blanc sobre `moss-active` | 9.59:1 | 4.5:1 | ✅ AAA |
| `terracotta` sobre `bg` (**només text gran ≥24px o component UI**) | 4.24:1 | 3:1 | ✅ AA (gran/UI). ❌ si s'usa en text normal <24px — **prohibit** |
| `terracotta-hover` sobre `bg` (text normal, qualsevol mida) | 5.66:1 | 4.5:1 | ✅ AA |
| Blanc sobre `terracotta` (botó ple) | 4.56:1 | 4.5:1 | ✅ AA (marge mínim: mantenir el to exacte, no enfosquir accidentalment el blanc) |
| Blanc sobre `terracotta-hover` | 6.10:1 | 4.5:1 | ✅ AA |
| Blanc sobre `terracotta-active` | 7.85:1 | 4.5:1 | ✅ AAA |
| Blanc sobre `error` (toast) | 6.71:1 | 4.5:1 | ✅ AA |
| `focus-ring` sobre `bg` | 5.98:1 | 3:1 (component no-text) | ✅ AA |
| `border` sobre `bg` | 1.39:1 | — (decoratiu, no porta informació sola) | N/A — mai usar sol per transmetre estat |

**Regla d'implementació derivada (EC-17 / A11Y-02):** `color.terracotta`
(`#C1552C`) només s'aplica a: (a) fons de components grans (pestanya
activa, botó ple amb text blanc — verificat més amunt), o (b) text ≥24px.
Per a qualsevol etiqueta, missatge o número en terracota a mida de text
normal (p. ex. un compte petit, un enllaç), s'usa `color.terracotta-hover`
(`#A6431F`) com a color de **text**, mai `#C1552C`. El nom del token es
manté (`-hover` també és el rol de "text terracota") per no introduir un
tercer to sense necessitat — un únic parell moss/terracotta amb 3 variants
cadascun és suficient per a v1.

**Escala de radi (PLAN.md §4: 16–20px)**

| Token | Valor | Ús |
|---|---|---|
| `radius.sm` | 8px | Badges d'avatar, inputs petits |
| `radius.md` | 16px | Files d'ítem, botons, toast |
| `radius.lg` | 20px | Caixes principals ("A comprar", "Rebost"), targetes |

**Escala d'ombra (soft shadows, PLAN.md §4)**

| Token | Valor | Ús |
|---|---|---|
| `shadow.sm` | `0 1px 2px rgba(46,42,34,0.06)` | Fila d'ítem en repòs |
| `shadow.md` | `0 4px 12px rgba(46,42,34,0.10)` | Caixa principal, input amb focus |
| `shadow.lg` | `0 8px 24px rgba(46,42,34,0.16)` | Toast, ítem durant l'animació de moviment (elevació temporal) |

**Escala d'espaiat** (base 4px, per mantenir alineació amb el radi i la
tipografia)

| Token | Valor |
|---|---|
| `space.xs` | 4px |
| `space.sm` | 8px |
| `space.md` | 16px |
| `space.lg` | 24px |
| `space.xl` | 32px |
| `space.2xl` | 48px |

### 8.2 Tipografia

- **Família:** Nunito (triada entre les dues opcions de PLAN.md §4 —
  Nunito té millor suport de pesos variables amb un sol fitxer i una `x-
  height` una mica més gran, que ajuda la llegibilitat dels noms d'ítem
  curts en majúscules/emoji mesclats).
- **Autoallotjament obligatori:** `.woff2` variable o estàtic a
  `app/web/fonts/`, servit pel mateix binari Go. **Cap crida a Google
  Fonts ni cap altre host extern** — la CSP no permet `font-src` cap a
  dominis externs i evita filtrar la IP de l'usuari.
- **Pesos necessaris (només 2, per PERF-02 <1s en 3G):**
  - `Nunito-Regular` (400) — cos de text, noms d'ítem, placeholder.
  - `Nunito-Bold` (700) — títols de caixa, botó primari, comptador, toast.
  - Cap `Italic`, cap pes intermedi. Si el fitxer variable no permet triar
    subconjunt de pesos, usar dos fitxers estàtics en lloc del variable
    complet — prioritzar mida sobre flexibilitat de pes.
- **Fallback stack:** `"Nunito", "Segoe UI", system-ui, -apple-system,
  sans-serif` (rounded sans del sistema si Nunito no carrega a temps —
  evita FOIT llarg en 3G).
- **Escala tipogràfica:**

| Token | Mida | Pes | Ús |
|---|---|---|---|
| `type.title` | 24px / 1.25 | Bold | Títol de caixa ("🛒 A comprar", "🥫 Rebost") |
| `type.body` | 16px / 1.4 | Regular | Nom d'ítem, text de formulari |
| `type.body-strong` | 16px / 1.4 | Bold | Nom d'ítem quan s'anuncia una acció (moment breu de confirmació visual, opcional) |
| `type.caption` | 13px / 1.3 | Regular | Marca de temps de moviment, ajuda del formulari, comptador de caràcters |
| `type.button` | 16px / 1.4 | Bold | Text de botons |
| `type.toast` | 14px / 1.4 | Regular | Missatge de toast |

### 8.3 Layout de pantalla

**Breakpoint:** `768px` és el punt de tall (per sota → mòbil apilat amb
pestanyes, EC-16; a partir de → escriptori dues caixes costat a costat).
Aquest valor és una decisió de disseny d'aquesta secció (no ve fixat per
PLAN.md), triat perquè és el llindar habitual tauleta-vertical/mòbil i
dona marge suficient perquè cada caixa d'escriptori tingui almenys 320px
d'ample útil abans de sentir-se comprimida.

**8.3.1 Escriptori (≥768px)**

```
┌──────────────────────────────────────────────────────────────────────┐
│  niu                                                    [Usuari A 🐦] │  ← capçalera, space.lg vertical
├──────────────────────────────────────────────────────────────────────┤
│                                                                        │
│  ┌────────────────────────────┐    ┌────────────────────────────┐   │
│  │ 🛒 A comprar            (2) │    │ 🥫 Rebost               (2) │   │  ← type.title, radius.lg,
│  │ ──────────────────────────  │    │ ──────────────────────────  │   │    shadow.md, space.lg padding
│  │                              │    │                              │   │
│  │ ┌──────────────────────────┐│    │ ┌──────────────────────────┐│   │
│  │ │ ○  Llet              🐦  ││    │ │ ●  Arròs         🦊 ↩    ││   │  ← fila d'ítem, radius.md,
│  │ └──────────────────────────┘│    │ └──────────────────────────┘│   │    shadow.sm, space.sm gap
│  │ ┌──────────────────────────┐│    │ ┌──────────────────────────┐│   │
│  │ │ ○  Pa                🦊  ││    │ │ ●  Oli           🐦 ↩    ││   │
│  │ └──────────────────────────┘│    │ └──────────────────────────┘│   │
│  │                              │    │                              │   │
│  │ ┌──────────────────────────┐│    │                              │   │
│  │ │ + afegir un ítem…        ││    │                              │   │
│  │ └──────────────────────────┘│    │                              │   │
│  └────────────────────────────┘    └────────────────────────────┘   │
│                                                                        │
└──────────────────────────────────────────────────────────────────────┘
   [regió aria-live, visualment oculta, ampla completa]
```

- Les dues caixes tenen la mateixa amplada (`1fr 1fr`), `space.lg` (24px)
  de separació entre elles, `space.xl` (32px) de marge respecte als
  costats de la pàgina en viewports amplis (amplada màxima de contingut:
  960px, centrada).
- Cada caixa és `color.surface` sobre `color.bg`, `radius.lg`,
  `shadow.md`, `space.lg` de padding intern.
- El camp "+ afegir un ítem…" només apareix a la caixa "A comprar" — no
  té sentit afegir directament a "Rebost" (l'abast només defineix "afegir
  a A comprar", §6 proposal.md).
- El comptador `(2)` al costat del títol és el nombre d'ítems a la caixa;
  ajuda a copsar l'estat sense llegir cada fila — reforça, no substitueix,
  el contingut.

**8.3.2 Mòbil (<768px, EC-16)**

```
┌───────────────────────────────┐
│  niu              [🐦]         │  ← capçalera compacta
├───────────────────────────────┤
│ ┌───────────┬───────────────┐ │
│ │ 🛒 Comprar│  🥫 Rebost    │ │  ← tab bar, space.sm padding,
│ │   (2)     │     (2)       │ │    pestanya activa amb subratllat
│ └───────────┴───────────────┘ │    moss/terracotta 3px + negreta
├───────────────────────────────┤
│                                │
│ ┌────────────────────────────┐│
│ │ ○  Llet               🐦   ││  ← fila d'ítem, amplada completa
│ └────────────────────────────┘│    menys space.md de marge
│ ┌────────────────────────────┐│
│ │ ○  Pa                 🦊   ││
│ └────────────────────────────┘│
│                                │
│ ┌────────────────────────────┐│
│ │ + afegir un ítem…          ││
│ └────────────────────────────┘│
│                                │
└───────────────────────────────┘
   [regió aria-live, oculta visualment]
```

- Una sola caixa visible alhora; l'altra és accessible via la pestanya.
  Contingut de totes dues caixes es manté al DOM (no es descarrega en
  canviar de pestanya) — necessari perquè l'animació FLIP (AC-10) pugui
  travessar el canvi de pestanya si un ítem es mou i cal ensenyar la
  caixa de destí (vegeu 8.6).
- Mida mínima de zona tàctil: **44×44px** (WCAG 2.5.8 AA, encara que
  formalment és un criteri AAA a 2.2 — s'aplica igualment com a mínim de
  producte perquè és mòbil real, EC-16). Això fixa l'alçada mínima de
  cada fila d'ítem i de cada pestanya.
- El camp "+ afegir un ítem…" es manté fixat a la part inferior de la
  caixa "A comprar" quan aquesta pestanya és l'activa (no sticky al
  viewport — simplement l'últim element de la llista, per no tapar
  contingut en pantalles curtes).

### 8.4 Inventari de components

Per a cada component: `default`, `:hover`, `:focus-visible`,
`:active`/premut, `disabled` (on apliqui). Cap component nou fora
d'aquesta llista — si `software-architect` o `fullstack-developer`
necessiten un component no descrit aquí, cal tornar a aquesta etapa.

**8.4.1 Caixa/panell (`Box`)**

| Estat | Especificació |
|---|---|
| Default | `color.surface`, `radius.lg`, `shadow.md`, vora `1px solid color.border` |
| Buida (EC-13) | Vegeu 8.4.5 "Estat buit" — substitueix la llista de files |
| — | No té `:hover`/`:focus` propis — no és interactiu com a unitat |

**8.4.2 Fila d'ítem (`ItemRow`)**

Estructura: `[indicador de caixa] [nom] [avatar(s)] [botó eliminar (en focus/hover)]`

| Estat | Especificació |
|---|---|
| Default | `color.surface-soft`, `radius.md`, `shadow.sm`, padding `space.sm` vertical / `space.md` horitzontal, `type.body` |
| `:hover` | Fons puja a `color.surface` + `shadow.md`; apareix el botó "eliminar" (icona paperera) a la dreta, abans ocult |
| `:focus-visible` | Anell de focus `2px solid color.focus-ring`, `outset` 2px (separat de la vora, mai només un canvi de color de fons) |
| `:active` / seleccionant per moure | `scale(0.98)` durant 100ms abans d'iniciar el FLIP (feedback tàctil immediat, AC-10) |
| Pendent (optimista, abans de confirmació del servidor) | Opacitat 0.85 + `shadow.lg` (elevació temporal) fins que el servidor confirma o falla (AC-12/AC-13) |
| Rebutjat pel servidor (rollback, AC-13) | Torna a la posició/caixa original amb la mateixa animació de FLIP invertida; simultàniament apareix el toast (8.4.4) |
| Marcat com a duplicat en intentar afegir (EC-06) | No aplica a `ItemRow` — el rebuig succeeix al formulari d'afegir (8.4.3), l'ítem existent **no** canvia d'aparença |

Indicador de caixa: cercle buit `○` (`color.text-secondary`, 8px) a "A
comprar"; cercle ple `●` (`color.moss`, 8px) a "Rebost". És decoratiu-
redundant amb la posició (mai l'única pista), però ajuda l'escaneig
visual ràpid.

**8.4.3 Input d'afegir ítem (`AddItemInput`)**

| Estat | Especificació |
|---|---|
| Default | `color.surface`, `radius.md`, vora `1px solid color.border`, placeholder "+ afegir un ítem…" en `color.text-secondary` |
| `:focus-visible` | Vora `2px solid color.focus-ring` + `shadow.md`; el placeholder desapareix en escriure |
| Enviant (petita espera de xarxa) | Botó "Afegir" (icona `+` o text) mostra un espera discret (opacitat 0.6), input `disabled` |
| Error de validació (EC-01/EC-02/EC-03/EC-05) | Vora `2px solid color.error`, missatge inline sota l'input en `type.caption` + `color.error` sobre `color.error-bg`, `role="alert"`. Missatges concrets: <br>— buit/espais: "Escriu un nom abans d'afegir." <br>— >200 car.: "Massa llarg — màxim 200 caràcters (portes {n}/200)." <br>— caràcters de control: "Aquest nom conté caràcters no vàlids." |
| Duplicat rebutjat (EC-06) | Mateix tractament visual d'error, però missatge específic i **accionable**: "«{nom}» ja hi és a {caixa on ja existeix}." (p. ex. "«Llet» ja hi és a Rebost.") — indica explícitament la caixa on ja existeix, no només "ja existeix", perquè l'usuari entengui que cal moure'l, no afegir-lo. El focus roman a l'input, el text escrit **no** s'esborra (l'usuari pot corregir) |
| Èxit | El missatge d'error (si n'hi havia) desapareix, l'input es buida i recupera el focus, la nova fila apareix a dalt de "A comprar" amb un `fade-in` breu (150ms) |
| Comptador de caràcters | `type.caption`, `color.text-secondary`, format "{n}/200", visible sempre (no només prop del límit) — referma el límit sense sorpresa a EC-02/EC-03 |

**8.4.4 Toast**

| Estat | Especificació |
|---|---|
| Aparició (AC-13) | Ancorat a la part inferior central (escriptori) o inferior amplada completa amb marge `space.md` (mòbil); `color.surface`, `radius.md`, `shadow.lg`, icona d'error + `type.toast` |
| Contingut | "No s'ha pogut moure «{nom}». Torna-ho a provar." — sempre inclou el nom de l'ítem, mai un missatge genèric sol |
| Comportament | No bloquejant (no intercepta clics fora seu), auto-descartat als 5s, també descartable amb una `×` o tecla `Escape`; `role="status"` (no `alert`, perquè no interromp — és informatiu i ja acompanya un canvi visual reversible) |
| `:focus-visible` (sobre el botó `×`) | Anell `2px solid color.focus-ring` |

**8.4.5 Estat buit (`EmptyState`, EC-13)**

| Estat | Especificació |
|---|---|
| "A comprar" buida, primer ús o després de comprar-ho tot | Icona/il·lustració discreta (emoji `🌿` o similar, mida 32px), `type.body` en `color.text-secondary`: "Res per comprar ara mateix." Manté l'input d'afegir visible i actiu — l'estat buit no bloqueja l'acció principal |
| "Rebost" buit, primer ús (EC-13) | Mateix tractament visual, text: "El rebost encara està buit." **Sense confeti** — el confeti (AC-14) només dispara quan "A comprar" passa de tenir ítems a buida per acció de l'usuari, mai en la càrrega inicial ni a "Rebost" |
| Ambdues buides simultàniament (primer arrencada, EC-13) | Cada caixa mostra el seu propi estat buit de manera independent; cap animació es dispara en aquest cas (vegeu 8.6 per la distinció exacta) |

**8.4.6 Barra de pestanyes mòbil (`TabBar`, EC-16)**

| Estat | Especificació |
|---|---|
| Pestanya inactiva | `color.text-secondary`, `type.body`, sense subratllat |
| Pestanya activa | `color.text-primary` `type.body-strong`, subratllat 3px del color de la caixa corresponent (`color.moss` per Rebost, `color.terracotta-hover` per A comprar — text-safe, vegeu 8.1.1) |
| `:focus-visible` | Anell `2px solid color.focus-ring` al voltant de tota la zona tàctil de la pestanya (mínim 44×44px) |
| `:active`/premut | Fons `color.surface-soft` breu (100ms) com a feedback tàctil |

**8.4.7 Avatar (`Avatar`)**

| Estat | Especificació |
|---|---|
| Default | Cercle 28px (escriptori) / 32px (mòbil, per llegibilitat tàctil), fons `color.surface-soft`, emoji centrat a mida 16px/18px |
| Dos avatars al mateix ítem | Es col·loquen en línia amb un separador `↩` petit entre ells (vegeu 8.8) — mai superposats en cercle, perquè la superposició amagaria un dels dos emojis i trencaria la llegibilitat |

### 8.5 Matriu d'estats — AC/EC → tractament visual

| AC/EC | Estat visual que el satisfà |
|---|---|
| EC-13 (buit primer ús) | 8.4.5 `EmptyState`, ambdues caixes independentment, sense confeti |
| AC-10 (FLIP) | 8.6.1 — vol de `space.md` d'elevació, ~250ms, `ease-out` |
| AC-11 (reduced motion) | 8.6.2 — cross-fade, mateixa durada, sense desplaçament de posició |
| AC-12 (optimista, èxit) | `ItemRow` estat "Pendent" (8.4.2) que es resol silenciosament a estat normal en confirmar-se; **cap parpelleig** — la fila mai torna a l'estat previ abans d'assentar-se al nou |
| AC-13 (optimista, fallada + rollback) | `ItemRow` "Rebutjat pel servidor" (8.4.2) + `Toast` (8.4.4) simultanis |
| AC-14 (confeti, un sol cop) | 8.6.3 — confeti + `EmptyState` de "A comprar" apareixen junts, disparat només per una transició d'estat (no-buit → buit) detectada al client, mai per un render en què ja estava buida |
| EC-01/EC-02/EC-03 (validació) | `AddItemInput` estat "Error de validació" (8.4.3), missatge específic per cas |
| EC-06 (duplicat) | `AddItemInput` estat "Duplicat rebutjat" (8.4.3), missatge que **anomena la caixa** on ja existeix |
| AC-06 (autoria visible) | `Avatar` (8.4.7), un o dos segons si afegit i mogut coincideixen (8.8) |
| AC-15 (teclat complet) | 8.7 — ordre de tabulació i activació amb `Enter`/`Space` |
| AC-16 (aria-live) | 8.7 — regió dedicada, format d'anunci exacte especificat |

### 8.6 Especificació de moviment (motion spec)

**8.6.1 FLIP (AC-10, ~250ms)**

- Tècnica FLIP (First-Last-Invert-Play) amb Web Animations API (PLAN.md
  §2.3).
- Propietat animada: `transform` (translate X/Y calculat entre posició
  origen i destí) — **mai** `top`/`left` (cost de layout).
- Durada: **250ms** exactes.
- Corba: `ease-out` (`cubic-bezier(0, 0, 0.2, 1)`) — sortida ràpida,
  arribada suau; coherent amb un moviment "cap a un lloc de repòs".
- Elevació temporal durant el vol: `shadow.lg` + `scale(1.03)` al pic del
  moviment, tornant a `scale(1)` en aterrar — reforça la sensació física
  sense afegir una segona animació independent.
- En aterrar: la fila queda exactament a la posició final de la llista
  ordenada (mai sobreposada a una altra fila ni desplaçada).

**8.6.2 Alternativa `prefers-reduced-motion` (AC-11)**

- Detectat via `matchMedia('(prefers-reduced-motion: reduce)')`.
- Substitueix el vol per un **cross-fade**: la fila desapareix de la
  caixa d'origen amb `opacity 1→0` en 150ms, i apareix a la caixa de
  destí amb `opacity 0→1` en 150ms (seqüencial, no simultani, perquè mai
  hi hagi dues còpies visibles alhora).
- Cap `transform`/desplaçament de posició en aquest mode.
- Mateixa durada total percebuda (~300ms) per no introduir una diferència
  de ritme notable entre modes.

**8.6.3 Confeti (AC-14)**

- Dispara **només** en la transició detectada al client de "A comprar
  amb ≥1 ítem" → "A comprar amb 0 ítems" causada per una acció (moure
  l'últim ítem o eliminar-lo) — mai en la càrrega/render inicial encara
  que la caixa ja estigui buida (EC-13), i mai en un render posterior
  mentre segueix buida.
- Un cop disparat, el component estableix un indicador local (p. ex. "ja
  celebrat per a aquest buidatge") que només es reinicia quan "A comprar"
  torna a tenir almenys un ítem — així un `refetch` de sondeig (~10s) amb
  la caixa encara buida mai el torna a disparar.
- Partícules: ~24–30 (discret, no una pantalla plena — PLAN.md §4 diu
  "discreta"), colors limitats a `color.moss`, `color.terracotta`, i un
  groc suau (`#E8C468`, no usat enlloc més — únic ús decoratiu puntual).
- Durada: ~1200ms, caiguda amb `ease-in`, origen des del centre superior
  de la caixa "A comprar".
- Si `prefers-reduced-motion` és actiu: el confeti se substitueix per un
  únic destello estàtic breu (canvi de fons de `EmptyState` a
  `color.success-bg` durant 600ms i tornada a `color.bg`) — la
  celebració es manté com a esdeveniment (no es descarta del tot), però
  sense partícules en moviment.

### 8.7 Narrativa d'accessibilitat

**Ordre de tabulació (escriptori):** capçalera (nom d'usuari, si
interactiu) → input "afegir ítem" → botó "Afegir" → files de "A comprar"
(d'amunt a avall, cadascuna com un element enfocable únic que representa
tota la fila) → files de "Rebost" (d'amunt a avall). Cada fila enfocada
mostra també, en focus, el botó "eliminar" (que passa a formar part de
l'ordre de tabulació **només** quan la fila que el conté té el focus —
`tabindex` gestionat, no un botó sempre tabulable per cada fila).

**Ordre de tabulació (mòbil):** capçalera → pestanyes (Tab entre "A
comprar"/"Rebost", `Enter`/`Space` per activar) → input "afegir ítem"
(si la pestanya activa és "A comprar") → files de la caixa activa
únicament (la caixa inactiva no forma part de l'ordre de tabulació
mentre no és visible — evita saltar a contingut ocult).

**Activació:**

- Una fila d'ítem enfocada es mou amb `Enter` o `Space` (AC-15) —
  equivalent exacte al clic/tap.
- Eliminar requereix enfocar el botó "eliminar" específic (mai la
  mateixa tecla que mou, per evitar esborrats accidentals) i activar-lo
  amb `Enter`/`Space`.
- `Escape` descarta el toast si n'hi ha un de visible.

**Focus visible:** tots els components interactius (`ItemRow`,
`AddItemInput`, botó "Afegir", botó "eliminar", pestanyes, botó `×` del
toast) mostren l'anell `2px solid color.focus-ring`, sempre `outset`
(mai només un canvi de fons/vora que es podria confondre amb `:hover`).
Contrast de l'anell verificat a 8.1.1 (5.98:1 sobre `color.bg`).

**Regió `aria-live` (AC-16):** una única regió `aria-live="polite"
aria-atomic="true"`, visualment oculta (tècnica `sr-only`, mai
`display:none` ni `visibility:hidden` — ha de romandre en el DOM
accessible), situada immediatament després de la capçalera i abans de
les dues caixes, compartida per ambdues.

**Format exacte de l'anunci en moure un ítem:**

```
"{Nom de l'ítem} mogut a {nom de la caixa de destí}."
```

Exemples literals:
- `"Llet mogut a Rebost."`
- `"Arròs mogut a A comprar."`

Aquest mateix format s'usa tant si el moviment l'origina l'usuari local
com si arriba reflectit per sondeig (un canvi fet per l'altre usuari) —
en aquest segon cas, cal afegir qui l'ha mogut perquè no soni com una
acció pròpia: `"{Nom de l'ítem} mogut a {caixa} per {nom visible de
l'usuari}."` (p. ex. `"Oli mogut a Rebost per Usuari B."`).

**Rols i etiquetes:**

- Cada caixa és `role="region"` amb `aria-label` = el seu títol exacte
  ("A comprar" / "Rebost"), no dependent de l'emoji decoratiu de la
  capçalera (`aria-hidden="true"` a l'emoji).
- Cada `ItemRow` és `role="button"` (o element natiu `<button>`) amb
  `aria-label` que inclou el nom i l'acció: `aria-label="Moure {nom} a
  {caixa contrària}"`.
- Els avatars són `aria-hidden="true"` visualment i el seu significat
  (qui ha afegit/mogut) es porta com a part del `aria-label` de la fila
  quan sigui rellevant, no com a informació només-visual.
- El comptador de caràcters de l'input és `aria-live="polite"` **només**
  quan queden ≤20 caràcters (evita soroll constant mentre s'escriu un nom
  curt).

**Mida mínima de zona tàctil (mòbil, EC-16):** 44×44px per a qualsevol
element interactiu (fila, pestanya, botó eliminar, botó `×` del toast) —
aplicat com a `min-height`/`min-width` encara que el contingut visual
sigui més petit (el `padding` compensa).

### 8.8 Superfície de gamificació (només v1)

Per PLAN.md §4, v1 conté **exclusivament** dues peces de gamificació —
res més s'afegeix sense tornar a aquesta etapa:

1. **Avatar emoji per usuari a cada ítem (AC-06).**
   - Mida: 28px escriptori / 32px mòbil (8.4.7).
   - Un sol avatar visible quan `added_by` i `moved_by` coincideixen (o
     `moved_by` és nul — l'ítem no s'ha mogut mai des que es va crear):
     mostra únicament l'avatar de qui l'ha afegit.
   - **Dos avatars** quan `added_by` ≠ `moved_by` (algú diferent de qui
     el va afegir l'ha mogut per últim cop): `[avatar afegit] ↩ [avatar
     mogut]`, en aquest ordre (creació → últim moviment), amb la fletxa
     `↩` de 12px en `color.text-secondary` entre ells. Aquesta lectura
     ("qui el va posar a la llista, i qui ha fet l'últim moviment") és la
     que apareix a l'exemple de PLAN.md §4 (`🐦 ↩` a la caixa "Rebost").
   - En passar el ratolí/enfocar un avatar (escriptori): `title`/tooltip
     natiu amb el nom visible complet (p. ex. "Afegit per Usuari A"),
     només com a reforç — la informació essencial ja és a `aria-label`
     de la fila (8.7), no depèn del tooltip.

2. **Confeti discret en buidar "A comprar" (AC-14).**
   - Especificat íntegrament a 8.6.3.

**Explícitament NO en aquesta versió** (PLAN.md §4, "❌ Streaks, punts,
leaderboards"): cap comptador de ratxa, cap puntuació, cap classificació,
cap indicador de "qui ha afegit més aquesta setmana". La taula `events`
recull les dades des del primer dia (§2.4 PLAN.md) però **cap element
visual d'aquesta especificació les mostra encara** — construir-ho ara
seria disseny especulatiu sobre un hàbit d'ús que encara no existeix.

## 9. Preguntes obertes per a la porta humana

- Cap pregunta pendent de decisió de producte: els dos punts que estaven
  oberts al backlog (duplicats i model de sincronització) ja s'han
  resolt amb l'usuari humà i es reflecteixen com a decisions a §6 i §7
  d'aquesta proposta.
- Confirmar que la mesura d'èxit qualitativa descrita a §5 (dues
  setmanes d'ús sense recórrer a WhatsApp) és acceptable com a criteri,
  donat que Niu no tindrà analítica de producte.
