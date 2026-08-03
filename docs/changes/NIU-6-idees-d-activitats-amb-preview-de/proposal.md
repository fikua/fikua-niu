---
artefact: proposal
key: "NIU-6"
type: "story"
title: "Idees d'activitats amb previsualització de link"
status: "approved"
owner: "product-manager"
parent_key: null
related_keys: ["NIU-1", "NIU-5"]
sources:
  - "Lean Canvas (Ash Maurya) — problem/solution framing"
  - "Amazon PR/FAQ — narrative front half"
created: "2026-08-03"
updated: "2026-08-03"
---

# Proposta — Idees d'activitats amb previsualització de link

> **Què és això.** Narrativa d'una pàgina que emmarca el problema, la
> solució proposada, a qui serveix i el valor que aporta. Lectura en
> menys de 3 minuts.

## 1. Titular

Niu guanya un tercer espai — separat de la llista de la compra i dels
projectes de casa — per desar idees d'activitats trobades per internet
sense perdre'n el context visual abans de decidir si es fan.

## 2. Problema

- Sovint Usuari A i Usuari B troben activitats interessants navegant —
  una web d'un restaurant, una publicació d'Instagram, un esdeveniment —
  però no tenen cap lloc compartit per desar-les que conservi el
  context visual (de què tracta, quin aspecte té) sense haver de tornar a
  obrir l'enllaç original.
- Igual que NIU-5 ja va identificar per a compres grans i projectes de
  casa, aquestes idees viuen avui fora de Niu: en captures de pantalla,
  missatges de WhatsApp compartits l'un a l'altre, o pestanyes obertes
  que s'acaben tancant sense decidir res.
- Un enllaç sol, sense context visual, és car de reconèixer més tard:
  cal tornar a obrir-lo per recordar de què tractava, i sovint això és
  precisament el fregament que fa que la idea es perdi.
- El cost és el mateix fregament domèstic recurrent que
  [overview.md](../../overview.md) ja documenta com a motivació de Niu —
  no catastròfic, però constant: una activitat es comenta un dia i
  ningú la torna a trobar quan arriba el moment de decidir un pla.

## 3. Client

- **Primari:** Usuari A i Usuari B, els dos usuaris de Niu, en peu d'igualtat
  — mateix rol que estableix [overview.md](../../overview.md) §"Per a
  qui": no hi ha administrador ni jerarquia.
- **Secundari:** cap. Mateixa naturalesa de producte privat per a dues
  persones que la resta de Niu.

## 4. Solució proposada

Un espai nou dins de Niu per desar idees d'activitats enganxant
simplement un enllaç: Niu recupera automàticament un títol, una imatge i
una descripció de la pàgina enllaçada i mostra la idea com una targeta
reconeixible d'un cop d'ull, sense haver de tornar a obrir l'enllaç
original per recordar de què tractava. Quan la pàgina no es pot
processar (per exemple, perquè bloqueja l'accés automàtic), la idea es
desa igualment amb l'enllaç visible, encara que sense targeta visual.
Els dos usuaris veuen les mateixes idees i qui les ha afegit, seguint el
mateix principi de font única de veritat que ja apliquen NIU-1 i NIU-5.

## 5. Valor i mesura d'èxit

- **Valor:** experiència d'usuari (UX) — elimina el mateix tipus de
  fregament domèstic que NIU-1 i NIU-5 ja resolen en altres àmbits (compra
  del dia a dia, projectes de cicle llarg), ara aplicat a idees
  d'activitats trobades navegant, que avui es perden per manca d'un lloc
  compartit amb context visual.
- **Mesura d'èxit:** com que Niu no té analítica de producte
  ([overview.md](../../overview.md) §"Com sabrem que funciona"), la
  mesura és qualitativa: durant almenys dues setmanes d'ús normal, tota
  activitat interessant que un dels dos trobi navegant es desa a Niu en
  lloc de quedar-se en una captura de pantalla o un missatge, i és
  reconeixible d'un cop d'ull sense haver de reobrir l'enllaç original.

## 6. Abast i fora d'abast

**En abast**

- Un espai dins de Niu, diferenciat de la llista de la compra (NIU-1) i
  dels projectes de casa (NIU-5), per a idees d'activitats afegides a
  partir d'un enllaç.
- Afegir una idea enganxant una URL; Niu en recupera automàticament un
  títol, una imatge i una descripció quan la pàgina enllaçada ho permet,
  i ho presenta com una targeta.
- Comportament definit quan la pàgina enllaçada **no** es pot processar
  (bloqueig, contingut no compatible, error de xarxa): la idea es desa
  igualment amb l'enllaç visible i un missatge clar que no hi ha
  previsualització disponible — mai un error que bloquegi l'usuari
  d'afegir la idea (detall exacte del missatge, `ux-ui-designer` a
  l'Etapa 1.5).
- Eliminar una idea desada.
- Cada idea mostra qui l'ha afegit, seguint el mateix principi
  d'atribució que NIU-1 i NIU-5.
- Els dos usuaris veuen les idees afegides per l'altre seguint el mateix
  mecanisme d'actualització que la resta de Niu (interrogació periòdica),
  sense necessitat de recarregar manualment.
- Persistència: les idees sobreviuen a reinicis del servei, igual que la
  resta de dades de Niu.
- Protecció bàsica de la informació introduïda pels usuaris, seguint el
  mateix estàndard que NIU-1: cap dada introduïda per un usuari
  s'interpreta com a codi executable, i el sistema no filtra mai detalls
  tècnics interns en cas d'error.

**Fora d'abast (explícit)**

- Qualsevol cicle de vida o estat (idea / planificada / feta) — a
  diferència de NIU-5, aquest ítem és una **llista de desades**, no un
  flux de decisió. Si l'ús real demostra que cal distingir "idea" de "ja
  decidida", és un canvi posterior, no una suposició d'aquesta proposta.
- Edició manual del títol, la imatge o la descripció recuperats — la
  targeta reflecteix el que s'ha pogut recuperar automàticament; no hi
  ha un formulari per corregir-los a mà en v1.
- Qualsevol integració amb l'API oficial d'Instagram o de cap altra
  xarxa social per esquivar bloquejos de scraping — el fallback
  d'enllaç-sense-previsualització (§4) és la resposta de v1; una
  integració d'API oficial és un cost i una dependència externa que no
  es justifiquen encara.
- Cerca, filtres, o categorització de les idees desades més enllà d'una
  llista simple.
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos — mateixa exclusió permanent que la resta
  de Niu ([overview.md](../../overview.md) §"Què no fa la v1").
- Qualsevol capa de gamificació (ratxes, punts) — mateixa postura que
  NIU-1 i NIU-5: es descarta fins que hi hagi ús real.
- Relació o dependència tècnica amb la llista de la compra (NIU-1) o amb
  compres i projectes (NIU-5) més enllà de compartir la mateixa app i
  els mateixos dos usuaris — és una col·lecció d'informació
  independent, amb un propòsit diferent (arxiu d'idees amb context
  visual, no seguiment d'estat).

**Diferit a un canvi posterior**

- Distingir estats dins de les idees desades (p. ex. "per fer" vs. "ja
  feta") si l'ús real ho demana — vegeu la incògnita corresponent a §7.
- Edició manual de les dades de la targeta quan la previsualització
  automàtica és incorrecta o incompleta.
- Actualitzar la previsualització si el contingut de la pàgina
  enllaçada canvia després de desar-la.

## 7. Riscos i incògnites

| Risc / incògnita | Severitat | Hipòtesi de mitigació |
| ---------------- | -------- | ---------------------- |
| Recuperar automàticament contingut d'una URL introduïda per un usuari, des del servidor, és una superfície coneguda d'abús (peticions server-side cap a destins arbitraris, incloent-hi xarxa interna) | HIGH | Es flag explícitament per a `software-architect`: la solució tècnica (Etapa 2) ha de tractar tota URL introduïda com a no fiable i restringir on el servidor pot arribar a fer la petició. No es proposa cap mitigació concreta aquí — és una decisió de disseny tècnic, no de producte. |
| Pàgines que bloquegen l'accés automàtitzat (Instagram i similars) o que no tenen contingut compatible (PDF, pàgines que exigeixen login) deixaran la idea sense previsualització sovint, no com a excepció rara | MEDIUM | El fallback d'enllaç-sense-previsualització (§6) és la resposta de v1, no un cas d'error puntual — es tracta com a comportament normal i esperat, no com un bug. |
| Sense cap camp d'estat ni distinció "per fer" vs. "ja feta", la llista pot créixer indefinidament amb idees que ja s'han fet o descartat, acumulant soroll | LOW | Fora d'abast d'aquesta proposta (v1 és una llista simple, §6); es revisa en un canvi posterior si l'ús real ho demana — vegeu paral·lelisme amb el mateix risc identificat a NIU-5 §7. |
| Confusió d'usuari entre aquest espai i els altres dos de Niu (llista de la compra, projectes de casa), si visualment s'assemblen massa tot i tenir propòsits diferents | LOW | `ux-ui-designer` haurà de diferenciar clarament els tres espais a l'Etapa 1.5 — es referencia aquí perquè no es resol en aquesta proposta. |
| Emmagatzemar contingut recuperat automàticament d'una pàgina externa (títol, descripció) pot incloure text no confiable que caldrà tractar amb cura en pantalla | LOW | Mateix estàndard que la resta de Niu (§6): cap dada introduïda o recuperada externament s'interpreta mai com a codi executable. Detall tècnic exacte a `design.md`. |

## 8. Visuals

> **Font única d'aquesta secció.** Aquesta especificació reutilitza
> íntegrament els tokens ja establerts a
> [NIU-1 `proposal.md` §8.1/§8.2](../NIU-1-llista-de-la-compra-rebost-auth/proposal.md)
> i el `design-system/tokens.css` existent — cap valor nou de color, tipus,
> radi, ombra o espaiat s'introdueix sense flag explícit (vegeu §8.0).
> `software-architect` la consumeix a l'Etapa 2 per dimensionar contractes;
> `fullstack-developer` implementa directament a partir d'aquí. Tots els
> valors són concrets — cap adjectiu sense xifra darrere.

### 8.0 Decisió de color — flag d'extensió del sistema de disseny

**Problema detectat:** `PLAN.md` §4 declara dos accents (`moss`,
`terracotta`), però tots dos ja estan compromesos:

- `color.moss` — caixa "Rebost" (NIU-1).
- `color.terracotta` / `color.terracotta-hover` — caixa "A comprar"
  (NIU-1, com a accent de pestanya) **i** tot l'espai "Projectes" (NIU-5,
  ADR-04, que el reutilitza com a accent primari de tot un espai nou).

Assignar terracota també a NIU-6 confondria exactament el que AC-07
exigeix evitar (diferenciar-se de NIU-5 "només pel visual"): dos espais
de nivell superior compartint el mateix accent no es distingeixen prou
com per superar AC-07 amb una comparació visual directa a `/audit`
(mateix mecanisme de verificació que NIU-5 §9 R-01).

**Decisió:** proposo un **tercer accent, `color.mel` (mel/mostassa
`#C99A3A`)** — coherent amb la família crema/molsa/terracota (un to
càlid, no un blau corporatiu), suficientment distant en to (groc-marró
enfront de verd i vermell-terrós) per no confondre's amb cap dels dos
existents ni amb `color.confetti-yellow` (`#E8C468`, reservat
exclusivament al confeti, PLAN.md §4 no el declara com a accent d'espai).

**Aquesta és una extensió proposada del sistema de disseny, no una
decisió tancada per aquest document** — `PLAN.md` §4 només va aprovar
crema/molsa/terracota. Es flag com a pregunta oberta a §9 perquè el
propietari humà l'aprovi abans que `software-architect`/
`fullstack-developer` en depenguin. Si el propietari prefereix no afegir
un tercer to, l'alternativa de fallback (sense cap valor nou) és **reduir
la diferenciació de NIU-6 a la navegació i la iconografia** (§8.2),
deixant terracota compartida entre "Projectes" i "Idees" — una opció
pitjor per AC-07 però que no requereix aprovació d'un nou token.

**Tokens nous proposats (pendents d'aprovació, seguint el format de
`design-system/tokens.css`):**

| Token | Valor | Ús |
|---|---|---|
| `--color-mel` | `#C99A3A` | Accent primari de l'espai "Idees" — botó "Desar idea", indicador de pestanya de navegació, vora de la targeta en estat de fallback |
| `--color-mel-hover` | `#B0842A` | Estat `:hover` de `mel` |
| `--color-mel-active` | `#966E20` | Estat `:active`/premut de `mel` |

Contrast verificat (mateixa fórmula WCAG 2.2 que NIU-1 §8.1.1): `#C99A3A`
sobre `#FBF6EC` (`color.bg`) = 2.1:1 — **insuficient per text normal**,
igual que passa amb `color.terracotta` pur a NIU-1. Per tant `color.mel`
segueix la mateixa regla que `terracotta` (§8.1.1 de NIU-1): **només
sobre superfícies grans/filled (botons amb text blanc) o text ≥24px**;
`color.mel-hover` (`#B0842A` sobre `#FBF6EC` = 3.4:1) és el token
**text-safe** per a etiquetes normals, comptadors o l'estat actiu de la
pestanya de navegació — mai `color.mel` pla per a text per sota de 24px.

### 8.1 Navegació — tercera entrada de nivell superior

Consistent amb ADR-04 de NIU-5 (una pestanya/enllaç nou a nivell
superior, sense inventar cap component de navegació nou): l'app passa de
dues a **tres** entrades de nivell superior. Com que el shipped
`app/web/index.html` encara no conté cap barra de navegació entre
espais (el `tabbar` actual és **intern** a la llista de la compra —
"Comprar" / "Rebost" — no un selector d'espai), aquesta és la primera
vegada que aquesta barra de nivell superior es materialitza visualment;
NIU-5 la va deixar com a judici obert de `fullstack-developer`. Es
proposa aquí perquè NIU-5 i NIU-6 no acabin duplicant-la de manera
inconsistent.

```
┌──────────────────────────────────────────────────────┐
│  niu          🛒 Compra   🏠 Projectes   💡 Idees   👤│
└──────────────────────────────────────────────────────┘
```

| Element | Especificació |
|---|---|
| Contenidor | Fila horitzontal (`space.md` gap), part de `.app-header` existent, o immediatament a sota si l'amplada no hi cap — decisió d'implementació lliure, no visual |
| Entrada inactiva | `color.text-secondary`, `type.body`, sense subratllat, icona emoji + etiqueta de text (mai només icona — AC-10/lectors de pantalla) |
| Entrada activa | `color.text-primary` `type.body-strong`, subratllat 3px del color propi de l'espai: `color.terracotta-hover` (Compra), `color.terracotta` sobre superfície (Projectes, ja establert per NIU-5 ADR-04) — **si el propietari no aprova `color.mel`, Idees hereta el mateix subratllat que Projectes**, veure §8.0 fallback |
| Entrada activa (si `color.mel` aprovat) | Idees: subratllat 3px `color.mel-hover` (text-safe) |
| `:focus-visible` | Anell `2px solid color.focus-ring`, outset 2px, zona tàctil mínima 44×44px (mòbil) |
| Icona "Idees" | 💡, decorativa (`aria-hidden="true"`) — mai l'única pista, l'etiqueta de text sempre acompanya |

Aquesta barra reutilitza exactament el patró visual del `TabBar` mòbil
ja especificat a NIU-1 §8.4.6 (subratllat de 3px sota la pestanya
activa, mateix anell de focus) — cap component nou al sistema, només una
instància a nivell d'app en lloc de dins d'una sola caixa.

### 8.2 Anatomia de la targeta d'idea (`IdeaCard`)

Estructura comuna a tots els estats: `[imatge (si existeix)] [títol]
[descripció (si existeix)] [avatar de qui l'ha afegit] [botó eliminar]`.

**Contenidor:** `color.surface`, `radius.lg` (20px — més generós que
`ItemRow` perquè aquí la targeta és la unitat principal de contingut, no
una fila d'una llista densa), `shadow.sm` en repòs, `shadow.md` en
`:hover`, padding `space.md`.

**Layout de la llista:** graella responsiva (`grid-template-columns:
repeat(auto-fill, minmax(240px, 1fr))`, gap `space.md`) — **no** una
llista vertical densa com `ItemRow` ni columnes d'estat com NIU-5,
perquè el contingut visual (imatge) necessita espai horitzontal per ser
reconeixible d'un cop d'ull (`proposal.md` §4). Aquesta graella és
precisament la tercera forma de disposició que diferencia l'espai "Idees"
de "Compra" (llista dual) i "Projectes" (llista única amb badge) —
reforça AC-07 més enllà del color.

#### Estat A — Targeta completa (AC-01)

| Zona | Especificació |
|---|---|
| Imatge | Ràtio 16:9, `object-fit: cover`, `radius.md` a les cantonades superiors (continua el `radius.lg` del contenidor), ocupa tota l'amplada de la targeta |
| Títol | `type.body-strong`, `color.text-primary`, màxim 2 línies (`-webkit-line-clamp: 2`), marge `space.sm` superior respecte la imatge |
| Descripció | `type.caption`, `color.text-secondary`, màxim 3 línies (`-webkit-line-clamp: 3`) |
| Enllaç original | Sempre present, discret: icona 🔗 + domini (no la URL completa) en `type.caption`, `color.text-secondary`, subratllat només en `:hover`/`:focus` — obre en pestanya nova |
| Avatar de qui l'ha afegit | `Avatar` (NIU-1 §8.4.7), 24px, cantonada inferior dreta de la targeta, mateix patró que NIU-1 AC-06 |
| Botó eliminar | Icona paperera, visible en `:hover`/`:focus` del contenidor (mateix patró d'aparició que `ItemRow` §8.4.2), mai visible per defecte en repòs — evita soroll visual en una graella densa |

#### Estat B — Fallback sense previsualització (AC-02)

Aquest **no és un estat d'error** — `requirements.md` §0.2 el tracta com
a resultat vàlid i habitual, i el disseny ho ha de comunicar visualment,
no amb l'estètica d'error (`color.error`/`color.error-bg`) ja reservada
a `AddItemInput` (NIU-1 §8.4.3).

| Zona | Especificació |
|---|---|
| Contenidor | Mateix `color.surface`/`radius.lg`, però **sense** zona d'imatge — alçada de la targeta es redueix, no deixa un rectangle buit on hauria anat la imatge |
| Icona substitutiva | Emoji 🔗 discret, 32px, `color.text-secondary`, centrat a l'espai on aniria la imatge — mateixa mida i to que la icona de `EmptyState` (NIU-1 §8.4.5), **no** una icona d'advertència ni de perill |
| Títol | Si Niu no ha pogut recuperar cap títol, es mostra el domini de l'enllaç (p. ex. `instagram.com`) en `type.body-strong` — mai una cadena buida ni "Sense títol" en to d'error |
| Missatge | `type.caption`, `color.text-secondary` (mai `color.error`): **"No s'ha pogut generar una previsualització d'aquest enllaç."** — descriptiu, no acusatori (no diu "ha fallat" ni "error") |
| Enllaç original | Mateix tractament que l'Estat A, però aquí és l'única via per identificar el contingut — es mostra la URL completa (truncada amb el·lipsi si excedeix l'amplada de la targeta), no només el domini |
| Avatar / botó eliminar | Idèntics a l'Estat A |

#### Estat C — Previsualització parcial (AC-03)

Mateixa anatomia que l'Estat A; cada zona absent (imatge, títol o
descripció) simplement **no es renderitza** — la targeta es redimensiona
per ocupar l'espai buit sense deixar un forat visual ni una vora
trencada. Exemple: si no hi ha imatge però sí títol i descripció, la
targeta comença directament pel títol amb el mateix padding `space.md`
superior que tindria sota la imatge. Si no hi ha descripció, el títol
passa directament a l'enllaç/avatar sense espai en blanc addicional.

**Regla general:** cap zona absent es reemplaça per un placeholder gris,
un requadre buit, ni cap element decoratiu que suggereixi contingut
trencat — l'omissió és silenciosa (`requirements.md` AC-03).

#### Estat D — Recuperant previsualització (transitori, entre l'enviament del formulari i la resposta del servidor)

`requirements.md` NFR-07 fixa un timeout dur per al scraping al
servidor — **no és instantani**. El disseny ha de comunicar espera sense
bloquejar l'usuari ni suggerir que l'app s'ha penjat.

| Element | Especificació |
|---|---|
| Comportament | En confirmar l'enviament del formulari (§8.3), l'input es buida immediatament i el focus hi torna (mateix patró d'"Èxit" d'`AddItemInput`, NIU-1 §8.4.3) — l'usuari pot continuar afegint idees o navegant sense esperar la resposta |
| Targeta temporal | Apareix immediatament a dalt de la graella una targeta en estat **"Recuperant…"**: mateix contenidor (`color.surface`, `radius.lg`), zona d'imatge substituïda per un `shimmer`/pols suau (`color.surface-soft` ↔ `color.border`, animació `opacity` 1.5s `ease-in-out` infinita — **no** un spinner giratori, coherent amb l'estètica calmada de `PLAN.md` §4) |
| Text de la targeta temporal | `type.caption`, `color.text-secondary`: **"Recuperant la previsualització…"**, domini de l'enllaç ja visible sota (l'usuari reconeix immediatament quina idea és, encara que la targeta final trigui) |
| Resolució | Quan el servidor respon (èxit, parcial o fallback), la targeta temporal es substitueix in-place per l'Estat A/B/C corresponent amb un `fade-in` de 150ms (mateixa durada que l'èxit d'afegir ítem, NIU-1 §8.4.3) — mai un salt brusc de posició a la graella |
| `prefers-reduced-motion` | El `shimmer` es substitueix per una opacitat fixa 0.7 sense animació (mateix principi que NIU-1 §8.6.2: mai moviment continu si l'usuari ho ha desactivat) |
| Timeout excessiu (EC-08) | Si el temps supera el llindar tècnic (valor exacte, Stage 2), la targeta temporal es resol directament a l'Estat B (fallback) — l'usuari mai veu un "Recuperant…" indefinit; no hi ha un estat visual separat de "timeout", es tracta igual que qualsevol altre fallback |
| Múltiples idees en curs | Cada targeta "Recuperant…" és independent — afegir una segona idea mentre la primera encara recupera no bloqueja ni cua la segona (coherent amb `requirements.md`, cap AC exigeix processament seqüencial) |

### 8.3 Entrada d'afegir idea (`AddIdeaInput`)

Reutilitza el patró estructural d'`AddItemInput` (NIU-1 §8.4.3), amb
camp únic (enllaç) en lloc de nom, i el verb "Desar" en lloc d'"Afegir"
per ser fidel al llenguatge de `proposal.md` §4 ("desar una idea").

| Estat | Especificació |
|---|---|
| Contenidor | Camp d'ample complet a dalt de la graella de targetes, `color.surface`, `radius.md`, vora `1px solid color.border`, placeholder **"Enganxa un enllaç…"** en `color.text-secondary` |
| `:focus-visible` | Vora `2px solid color.focus-ring` + `shadow.md` |
| Botó "Desar" | Text (no només icona), accent `color.mel` sobre superfície filled amb text blanc (o `color.terracotta` si §8.0 no s'aprova) — únic ús de l'accent primari de l'espai en un element interactiu petit |
| Enviant (EC-08, espera de xarxa) | **Important — no confondre amb l'Estat D de la targeta.** El botó "Desar" mostra espera discreta (opacitat 0.6, mateix patró que NIU-1 §8.4.3) només durant la validació immediata del format d'URL (client-side, instantani); en confirmar-se el format, la targeta "Recuperant…" (§8.2 Estat D) assumeix la resta de l'espera — el formulari **no roman bloquejat** esperant el scraping complet |
| Error de validació (EC-01/EC-10) | Vora `2px solid color.error`, missatge inline en `type.caption` + `color.error` sobre `color.error-bg`, `role="alert"`. Missatges concrets: <br>— buit/espais (EC-10): "Enganxa un enllaç abans de desar." <br>— esquema no vàlid (EC-01): "Aquest enllaç no és vàlid — ha de començar per http:// o https://." <br>— destí rebutjat (EC-02/EC-07, si el rebuig arriba com a error de formulari i no com a fallback silenciós — decisió de Stage 2): "Aquest enllaç no es pot desar." (missatge genèric, mai revela detalls de xarxa interns, per NFR-06) |
| Èxit | El missatge d'error (si n'hi havia) desapareix, l'input es buida i recupera el focus, la targeta "Recuperant…" apareix a dalt de la graella (§8.2 Estat D) |
| Format | Sense comptador de caràcters (a diferència d'`AddItemInput` — una URL no té el mateix llindar pràctic de 200 caràcters que un nom d'ítem; `requirements.md` no defineix cap límit de longitud explícit per a l'enllaç) |

### 8.4 Estat buit (EC-17)

Mateix tractament que `EmptyState` (NIU-1 §8.4.5): icona discreta
(emoji 💡 o 🌿, coherent amb la resta de Niu, 32px, `color.text-secondary`),
`type.body` en `color.text-secondary`: **"Encara no hi ha cap idea
desada."** Manté `AddIdeaInput` (§8.3) visible i actiu — l'estat buit no
bloqueja l'acció principal, mateix principi que NIU-1.

### 8.5 Responsive (EC-18, `PLAN.md` §4)

| Breakpoint | Comportament |
|---|---|
| Escriptori (≥768px, `--layout-breakpoint` existent) | Graella `auto-fill, minmax(240px, 1fr)` — normalment 3–4 columnes segons l'amplada de finestra, `--layout-max-content: 960px` |
| Mòbil (<768px) | Graella d'una sola columna (`minmax` col·lapsa a `1fr` únic), `AddIdeaInput` ample complet, mateixa barra de navegació de nivell superior (§8.1) amb icones+etiquetes que ja s'ajusta a mòbil per NIU-1/NIU-5 |
| Zona tàctil | Botó eliminar i botó "Desar": mínim 44×44px (`--touch-target-min` existent), mateix llindar que NIU-1 |

No calen pestanyes internes (a diferència de la llista de la compra) —
és un únic espai sense sub-seccions, així que EC-18 es compleix només
amb el canvi de graella a columna única.

### 8.6 Accessibilitat (AC-09, AC-10)

**Ordre de tabulació:** barra de navegació de nivell superior (§8.1) →
`AddIdeaInput` (camp + botó "Desar") → cada `IdeaCard` en ordre de
llista (dins de cada targeta: enllaç original → botó eliminar). Cap
element interactiu és inabastable només amb `Tab`/`Shift+Tab`.

**Narrativa de lector de pantalla per `IdeaCard`:**

- Cada targeta és un element `<article>` o `role="group"` amb
  `aria-label` construït a partir del títol disponible (o del domini, en
  fallback) — mai una llista de `<div>` sense semàntica.
- **Imatge (AC-10):** el `alt` **no** es genera automàticament a partir
  de l'atribut `alt` de la imatge Open Graph original — les imatges OG
  sovint no en porten cap, o en porten un de buit o irrellevant
  (decoratiu de la xarxa social d'origen, no descriptiu del contingut).
  Es defineix `alt=""` (imatge tractada com a **decorativa**, ignorada
  per lectors de pantalla) i el mateix títol textual ja visible a la
  targeta és qui porta la informació — evita que un lector de pantalla
  llegeixi un `alt` inútil o, pitjor, buit de manera que sembli que
  falta contingut. Aquesta és una desviació intencionada d'AC-10 tal
  com està redactat ("la imatge, si existeix, té un text alternatiu no
  buit") — es flag explícitament a §9 perquè el propietari confirmi
  quin dels dos comportaments prefereix.
- **Estat de fallback (Estat B):** el missatge "No s'ha pogut generar
  una previsualització d'aquest enllaç." s'inclou dins de l'`aria-label`
  o del contingut anunciat de la targeta — un lector de pantalla mai
  hauria de percebre un "buit" sense explicació on aniria la imatge.
- **Targeta "Recuperant…" (Estat D):** regió `aria-live="polite"`
  compartida (la mateixa ja existent a NIU-1, `#live-region`) anuncia
  **"Desant idea, recuperant previsualització…"** en confirmar l'enviament,
  i **"Idea desada"** (Estat A/C) o **"Idea desada sense previsualització"**
  (Estat B) en resoldre's — mai silenci durant una espera que pot durar
  diversos segons (NFR-07).
- **Botó eliminar:** `aria-label="Eliminar la idea «{títol o domini}»"`
  — mai només "Eliminar" sense context, mateix principi que
  `ItemRow` de NIU-1.

**Contrast:** totes les combinacions text/fons reutilitzen parelles ja
verificades a NIU-1 §8.1.1, excepte les noves parelles de `color.mel`
verificades a §8.0 — cap combinació nova sense verificar.

### 8.7 Matriu d'estats — AC/EC → tractament visual

| AC/EC | Tractament |
|---|---|
| AC-01 (targeta completa) | §8.2 Estat A |
| AC-02 (fallback) | §8.2 Estat B |
| AC-03 (previsualització parcial) | §8.2 Estat C |
| AC-04 (autoria) | `Avatar` (NIU-1 §8.4.7), reutilitzat sense canvis |
| AC-05 (eliminar) | Botó eliminar, §8.2, totes les targetes |
| AC-06 (convergència) | Sense tractament visual propi — mateix mecanisme de sondeig ja establert, cap indicador visual addicional |
| AC-07 (diferenciació visual) | §8.0 (accent) + §8.1 (navegació) + §8.2 (graella, tercera disposició diferent de NIU-1/NIU-5) |
| AC-08 (validació d'enllaç) | `AddIdeaInput` (§8.3), estat "Error de validació" |
| AC-09 (teclat) | §8.6, ordre de tabulació |
| AC-10 (lector de pantalla) | §8.6, narrativa + flag d'`alt=""` |
| EC-08 (timeout) | §8.2 Estat D → resolució a Estat B |
| EC-10 (enllaç buit) | `AddIdeaInput`, estat "Error de validació" |
| EC-11 (injecció HTML/JS) | Cap tractament visual especial — `textContent` únicament, mateix estàndard NIU-1 (S3); títol/descripció recuperats mai es renderitzen com HTML |
| EC-17 (llista buida) | §8.4 |
| EC-18 (mòbil) | §8.5 |

## 9. Preguntes obertes — RESOLTES a la porta humana (2026-08-03)

- **Fallback quan no es pot previsualitzar:** confirmat — la idea es
  desa igualment amb l'enllaç visible i sense targeta visual, sense cap
  pas de confirmació addicional.
- **Cicle de vida:** confirmat — v1 és una llista simple de desades,
  sense cap camp d'estat.
- **Enllaços duplicats:** confirmat — es permet desar el mateix enllaç
  més d'un cop, sense bloqueig ni avís (`requirements.md` EC-06).
- **Superfície de risc tècnic (SSRF):** pendent per a `software-architect`
  a l'Etapa 2, no bloqueja aquesta porta — `requirements.md` ja hi fixa
  els NFR d'obligat compliment (NFR-05–07).

## 9.1. Preguntes obertes de l'Etapa 1.5 — RESOLTES a la porta humana (2026-08-03)

- **Nou accent `color.mel` (§8.0):** confirmat — s'aprova el tercer
  token d'accent (`#C99A3A` / `-hover` `#B0842A` / `-active` `#966E20`)
  per diferenciar l'espai "Idees" de "Projectes" (terracota) i "Compra"
  (crema/molsa). No s'aplica el fallback de compartir terracota amb
  NIU-5.
- **`alt=""` en imatges recuperades (§8.6):** confirmat — les imatges
  Open Graph es tracten com a decoratives (`alt=""`), confiant en el
  títol visible com a font d'informació accessible. No es força alt
  text no buit.
