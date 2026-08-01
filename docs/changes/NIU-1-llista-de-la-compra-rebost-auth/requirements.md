---
artefact: requirements
key: "NIU-1"
title: "Llista de la compra ↔ rebost (auth stubbed)"
status: "approved"
owner: "product-manager + qa-engineer"
proposal_path: "./proposal.md"
ac_count: 16
nfr_count: 11
sources:
  - "User Story format (Mike Cohn) — As a / I want / So that"
  - "INVEST — Independent, Negotiable, Valuable, Estimable, Small, Testable"
  - "Given/When/Then — Gherkin / Cucumber"
created: "2026-08-01"
updated: "2026-08-01"
---

# Requirements — Llista de la compra ↔ rebost (auth stubbed)

> **Què és això.** El contracte entre producte i enginyeria. Cada
> criteri d'acceptació és verificable i es traçarà amb almenys una tasca a
> `tasks.md`. **Només comportament funcional — cap detall d'implementació.**
>
> Font: `PLAN.md` §2 (Arquitectura), §3 (Seguretat), §4 (Look & feel),
> §7.1 (Casos funcionals), §8 (NIU-1). Vegeu també `proposal.md`.

## 1. User story

- **Com a** membre de la parella que utilitza Niu
- **Vull** afegir, moure entre "A comprar" i "Rebost", i eliminar ítems
  d'una llista de la compra compartida
- **Perquè** l'altra persona vegi sempre l'estat real de què falta i què
  ja hi ha a casa, sense dependre de WhatsApp o memòria.

## 2. INVEST self-check

- [x] **Independent** — ✅ No depèn de cap altre ítem en curs (NIU-2/3/4 en depenen d'aquest, no al revés).
- [x] **Negotiable** — ✅ El disseny intern (handlers, esquema exacte de taules) és obert; el contracte extern (API, comportament) és fix per PLAN.md.
- [x] **Valuable** — ✅ És l'única funcionalitat visible per a l'usuari final en v1.
- [x] **Estimable** — ✅ Abast tancat: CRUD + UI + seguretat bàsica, sense auth real.
- [x] **Small** — ✅ Encaixa en un sol cicle de `/define` → `/code` → `/audit`.
- [x] **Testable** — ✅ Cada AC és observable via l'API HTTP o el DOM renderitzat; cap AC depèn d'inspeccionar estat intern no exposat.

## 3. Acceptance criteria

### AC-01 — Afegir un ítem nou a "A comprar"

- **Given** la llista "A comprar" té zero o més ítems
- **When** l'usuari afegeix un ítem amb un nom vàlid (1–200 caràcters després de retallar espais)
- **Then** l'ítem apareix a "A comprar" immediatament i persisteix després de recarregar la pàgina

### AC-02 — Moure un ítem de "A comprar" a "Rebost"

- **Given** un ítem existeix a "A comprar"
- **When** l'usuari el selecciona per moure'l
- **Then** l'ítem desapareix de "A comprar" i apareix a "Rebost" en una sola operació, amb l'autor del moviment i la marca de temps actualitzats

### AC-03 — Moure un ítem de "Rebost" a "A comprar"

- **Given** un ítem existeix a "Rebost"
- **When** l'usuari el selecciona per moure'l
- **Then** l'ítem torna a "A comprar", amb l'autor del moviment i la marca de temps actualitzats

### AC-04 — El moviment persisteix

- **Given** un ítem s'ha mogut d'una caixa a l'altra
- **When** l'usuari recarrega la pàgina
- **Then** l'ítem apareix a la caixa de destí, no a l'original

### AC-05 — Eliminar un ítem

- **Given** un ítem existeix a "A comprar" o a "Rebost"
- **When** l'usuari l'elimina
- **Then** l'ítem desapareix de totes dues caixes i no reapareix en recarregar

### AC-06 — Cada ítem mostra qui l'ha tocat

- **Given** un ítem ha estat afegit o mogut per un usuari concret
- **When** es mostra l'ítem a la interfície
- **Then** l'avatar (emoji) de l'usuari que l'ha afegit i, si escau, de qui l'ha mogut per darrer cop, és visible sobre l'ítem

### AC-07 — Usuari actual identificat (auth stubbed)

- **Given** l'aplicació funciona amb un usuari fix (sense pantalla de login)
- **When** el client consulta l'usuari actual
- **Then** rep les dades d'aquest usuari (nom, nom visible, avatar) de manera consistent en totes les peticions de la mateixa sessió de navegador

### AC-08 — Dos usuaris veuen la mateixa llista (convergència eventual)

- **Given** la usuària A afegeix un ítem
- **When** l'usuari B recarrega la pàgina, torna el focus a la finestra, o espera l'interval de sondeig (~10s)
- **Then** l'usuari B veu l'ítem afegit per A

### AC-09 — Moviment concurrent del mateix ítem convergeix sense error

- **Given** dos usuaris tenen el mateix ítem visible a la seva pantalla
- **When** tots dos el mouen (a destins potencialment diferents) gairebé simultàniament
- **Then** cap de les dues peticions falla amb error, i després del següent refresc/sondeig ambdós clients mostren el mateix estat final (el de l'última escriptura acceptada pel servidor)

### AC-10 — Animació de moviment (FLIP)

- **Given** l'usuari selecciona un ítem per moure'l
- **When** el sistema no té `prefers-reduced-motion` activat
- **Then** l'ítem es desplaça visualment cap a la caixa de destí en ~250ms i queda posicionat correctament a la caixa de destí en acabar l'animació

### AC-11 — Alternativa accessible a l'animació

- **Given** l'usuari té `prefers-reduced-motion` activat al sistema/navegador
- **When** mou un ítem
- **Then** el canvi es mostra amb una transició d'esvaïment (cross-fade), sense desplaçament volador

### AC-12 — Actualització optimista amb èxit

- **Given** l'usuari mou un ítem
- **When** la petició al servidor té èxit
- **Then** la interfície ja mostrava el nou estat abans de rebre la resposta, i cap parpelleig ni reversió es produeix

### AC-13 — Actualització optimista amb fallada i reversió

- **Given** l'usuari mou un ítem i la interfície l'ha mogut de manera optimista
- **When** la petició al servidor falla (error de xarxa o resposta d'error)
- **Then** l'ítem torna animadament a la seva posició original i es mostra un avís no bloquejant (toast) indicant l'error

### AC-14 — Confeti en buidar "A comprar"

- **Given** la caixa "A comprar" té almenys un ítem
- **When** l'últim ítem es mou o s'elimina i la caixa queda buida
- **Then** s'executa una animació de confeti discreta exactament una vegada (no es repeteix en renderitzats posteriors mentre la llista continua buida)

### AC-15 — Navegació completa per teclat

- **Given** l'usuari no utilitza ratolí ni pantalla tàctil
- **When** navega l'aplicació únicament amb teclat
- **Then** pot afegir un ítem, moure'l entre caixes i eliminar-lo sense necessitar cap altre dispositiu d'entrada

### AC-16 — Anunci per lectors de pantalla en moure un ítem

- **Given** l'usuari utilitza un lector de pantalla
- **When** un ítem es mou d'una caixa a l'altra (per acció pròpia o per sondeig que reflecteix un canvi remot)
- **Then** una regió `aria-live` anuncia el canvi de manera comprensible (nom de l'ítem i caixa de destí)

## 4. Edge cases and negative scenarios

### EC-01 — Nom buit o només espais en blanc

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom buit o compost únicament per espais/tabulacions
- **Then** la petició és rebutjada amb un missatge d'error clar i cap ítem es crea

### EC-02 — Nom al límit de longitud (200 caràcters)

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom de exactament 200 caràcters després de retallar espais
- **Then** l'ítem s'accepta i es desa sencer

### EC-03 — Nom que excedeix el límit (201 caràcters)

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom de 201 caràcters o més després de retallar espais
- **Then** la petició és rebutjada amb un missatge d'error clar

### EC-04 — Nom amb Unicode complet (accents, emoji, apòstrof)

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom com `Pastanagues 🥕`, `O'Neill`, o `Formatge d'ovella`
- **Then** el nom es desa i es mostra exactament igual, sense corrupció ni escapament visible

### EC-05 — Nom amb caràcters de control

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom que conté caràcters de control no imprimibles (p. ex. bytes de control ASCII)
- **Then** el sistema els rebutja o els neutralitza abans de desar — mai els emmagatzema tal qual ni permet que trenquin el renderitzat

### EC-06 — Ítem duplicat (mateix nom, en qualsevol de les dues caixes)

- **Given** ja existeix un ítem amb un nom que, retallat i comparat sense distingir majúscules/minúscules, coincideix amb `"llet"` (p. ex. ja hi ha `"Llet"` a "A comprar" o `"LLET "` a "Rebost")
- **When** l'usuari intenta afegir `"llet"`, `"Llet "`, o `"LLET"` a qualsevol de les dues caixes
- **Then** la petició és **rebutjada** amb un missatge clar indicant que l'ítem ja existeix, independentment de quina caixa el conté

### EC-07 — Duplicat exacte permès després d'eliminar l'original

- **Given** un ítem `"Pa"` va existir i ha estat eliminat
- **When** l'usuari afegeix `"Pa"` de nou
- **Then** s'accepta com un ítem nou (la comprovació de duplicats només mira ítems actius, no l'historial)

### EC-08 — Intent de mutació via `GET`

- **Given** l'API exposa `/api/v1/items`
- **When** un client intenta provocar una mutació (crear, moure, eliminar) mitjançant una petició `GET`
- **Then** no existeix cap ruta `GET` que produeixi efectes secundaris; qualsevol intent d'aquest tipus és, per disseny de rutes, tractat com a simple lectura o com a ruta inexistent (404/405)

### EC-09 — Injecció HTML/JS al nom de l'ítem (XSS)

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom com `<img src=x onerror=alert(1)>` o `<script>alert(1)</script>`
- **Then** el nom es desa i es mostra com a **text literal** a la pantalla (no s'executa cap script, no es renderitza cap element HTML)

### EC-10 — Injecció SQL al nom de l'ítem

- **Given** el formulari d'afegir ítem
- **When** l'usuari envia un nom com `'; DROP TABLE items;--`
- **Then** el nom es desa literalment com a text, la resta de dades de l'aplicació roman intacta, i cap taula desapareix

### EC-11 — Eliminar un ítem que ja no existeix (doble clic / re-enviament)

- **Given** un ítem ha estat eliminat (per aquest usuari o per l'altre, ja convergit)
- **When** una segona petició d'eliminació pel mateix ítem arriba al servidor
- **Then** la petició no provoca un error 5xx ni corromp l'estat; l'ítem continua absent (operació idempotent)

### EC-12 — Moure un ítem inexistent o ja eliminat

- **Given** un ítem ha estat eliminat
- **When** un client (amb estat desactualitzat) intenta moure'l
- **Then** la petició respon amb un error clar sense afectar altres ítems, i el client pot refrescar per recuperar l'estat real

### EC-13 — Llista buida en primer ús

- **Given** l'aplicació s'inicia per primer cop (seed inicial, sense ítems)
- **When** l'usuari obre la interfície
- **Then** ambdues caixes es mostren buides sense error, amb un estat visual clar de "res per mostrar" (sense disparar confeti erròniament a "Rebost")

### EC-14 — Reinici del contenidor amb dades existents

- **Given** hi ha ítems desats
- **When** el contenidor es reinicia
- **Then** en tornar a arrencar, tots els ítems i el seu estat (caixa, autor, moment del moviment) romanen intactes

### EC-15 — Reinici a mig d'una escriptura

- **Given** una escriptura a la base de dades està en curs
- **When** el procés es talla abruptament (equivalent a `docker kill`)
- **Then** la base de dades no queda corrupta i l'aplicació arrenca correctament al reinici següent

### EC-16 — Viewport mòbil

- **Given** l'usuari obre l'aplicació en una pantalla estreta (mòbil)
- **When** interactua amb la interfície
- **Then** les dues caixes es presenten apilades amb pestanyes navegables, mantenint totes les funcionalitats (afegir, moure, eliminar)

### EC-17 — Contrast de color insuficient (regressió visual)

- **Given** la paleta de colors definida (crema, verd molsa, terracota)
- **When** es renderitza qualsevol text sobre el seu fons
- **Then** la relació de contrast compleix el llindar AA (WCAG 2.2) per a text normal i text gran

## 5. Non-functional requirements (NFRs)

| ID | Category | Statement | Target / threshold |
|----|----------|-----------|---------------------|
| NFR-01 | sec (S3 — XSS) | Cap nom d'ítem ni cap altra dada d'usuari es renderitza mai com a HTML; tot text es insereix com a text pla | Zero ocurrències d'`innerHTML` amb dades d'usuari al codi client; CSP sense `unsafe-inline` present a totes les respostes HTML |
| NFR-02 | sec (S7 — capçaleres) | Totes les respostes HTTP inclouen les capçaleres de seguretat obligatòries | `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer` i `Content-Security-Policy` presents al 100% de les respostes verificades |
| NFR-03 | sec (S8 — injecció SQL) | Cap valor introduït per l'usuari es concatena mai a una sentència SQL | 100% de les consultes que incorporen entrada d'usuari utilitzen paràmetres vinculats; test d'injecció (EC-10) passa en cada execució de CI |
| NFR-04 | sec (S1 parcial — sense CSRF complet fins NIU-4) | Cap mutació és accessible via `GET` | Assaig de la taula de rutes: 0 rutes `GET` amb efecte d'escriptura |
| NFR-05 | perf | Temps de resposta de llistat d'ítems es manté baix amb una llista realista | p95 de `GET /api/v1/items` < 200 ms amb 500 ítems a la base de dades |
| NFR-06 | perf | Càrrega inicial de la interfície és ràpida en connexions lentes | Temps de primera pintura interactiva < 1 s simulant una xarxa 3G |
| NFR-07 | reliability | La base de dades sobreviu una interrupció abrupta durant una escriptura | 0 casos de corrupció detectada en 10 repeticions de `docker kill` durant una escriptura activa (EC-15) |
| NFR-08 | reliability | El servei de salut reflecteix l'estat real de la dependència de dades | `GET /healthz` retorna 200 únicament quan la base de dades respon a una consulta trivial; retorna un codi d'error en cas contrari |
| NFR-09 | a11y | Totes les pantalles compleixen contrast AA i són operables per teclat | WCAG 2.2 AA verificat per a totes les combinacions text/fons definides a §4 del `PLAN.md`; 100% de les accions (afegir, moure, eliminar) accessibles sense ratolí |
| NFR-10 | a11y | Els canvis d'estat rellevants s'anuncien a tecnologies d'assistència | Regió `aria-live="polite"` (o equivalent) actualitzada en cada moviment d'ítem, verificat amb un lector de pantalla real o eina d'auditoria |
| NFR-11 | i18n/l10n | Els noms d'ítem admeten l'alfabet català complet i símbols habituals sense pèrdua | 0 casos de corrupció o normalització incorrecta per als caràcters de EC-04 (accents, emoji, apòstrof) en un cicle desar→llegir |

## 6. Testing strategy (drafted by `qa-engineer`)

> Pirámide Google Testing Blog: **small** (unit) per lògica pura de
> validació; **medium** (integració) per l'API contra SQLite real;
> **large** (E2E amb navegador real, p. ex. Playwright) només per
> animació, accessibilitat i XSS renderitzat — allò que exigeix un DOM
> real. Cap AC de seguretat es dona per bo només perquè "existeix una
> mitigació": cada test de seguretat executa l'atac i n'afirma el
> fracàs.

| Identificador | Unit | Integració | E2E | Manual | Validació NFR |
|---|---|---|---|---|---|
| AC-01 | ✅ (validació nom) | ✅ (persistència real) | — | — | — |
| AC-02 | — | ✅ | ✅ (visual/animació) | — | — |
| AC-03 | — | ✅ | ✅ | — | — |
| AC-04 | — | ✅ | — | — | — |
| AC-05 | — | ✅ | — | — | — |
| AC-06 | — | ✅ (autoria a la resposta) | ✅ (avatar visible) | — | — |
| AC-07 | ✅ | ✅ | — | — | — |
| AC-08 | — | ✅ (dos clients simulats) | ⚠️ manual amb dos navegadors si cal explorar | ✅ | — |
| AC-09 | — | ✅ (peticions concurrents simulades) | — | — | — |
| AC-10 | — | — | ✅ (navegador real, mesura de durada ~250ms) | — | — |
| AC-11 | — | — | ✅ (`prefers-reduced-motion` emulat) | — | — |
| AC-12 | — | ✅ | ✅ | — | — |
| AC-13 | — | ✅ (simulant error de servidor) | ✅ (reversió + toast) | — | — |
| AC-14 | — | — | ✅ (confeti dispara un cop) | — | — |
| AC-15 | — | — | ✅ (navegació per Tab/Enter) | ⚠️ exploratori amb usuari real | — |
| AC-16 | — | — | ✅ (assert contingut `aria-live`) | ⚠️ verificació puntual amb lector de pantalla real | — |
| EC-01 | ✅ | ✅ | — | — | — |
| EC-02 | ✅ | ✅ | — | — | — |
| EC-03 | ✅ | ✅ | — | — | — |
| EC-04 | ✅ | ✅ | ✅ (renderitzat verbatim) | — | — |
| EC-05 | ✅ | ✅ | — | — | — |
| EC-06 | ✅ (normalització trim+lowercase) | ✅ (contra dades reals a totes dues caixes) | — | — | — |
| EC-07 | — | ✅ | — | — | — |
| EC-08 | — | ✅ (assaig de la taula de rutes) | — | — | — |
| EC-09 | — | — | ✅ (assert absència d'execució de script en navegador real) | — | — |
| EC-10 | — | ✅ (verifica taula intacta post-atac) | — | — | — |
| EC-11 | — | ✅ (doble DELETE) | — | — | — |
| EC-12 | — | ✅ | — | — | — |
| EC-13 | — | ✅ | ✅ (estat buit sense confeti fals) | — | — |
| EC-14 | — | ✅ (reinici de contenidor en CI/local) | — | — | — |
| EC-15 | — | — | — | ⚠️ requereix orquestració de `docker kill`, documentat com a procediment manual/script dedicat | ✅ repetit 10 cops |
| EC-16 | — | — | ✅ (viewport emulat) | — | — |
| EC-17 | — | — | — | ✅ (auditoria de contrast, eina automatitzada tipus axe/Lighthouse) | ✅ |
| NFR-01 | ✅ (grep de codi client) | — | ✅ (EC-09 en navegador real) | — | Revisió de codi + test E2E |
| NFR-02 | — | ✅ (assert capçaleres a cada resposta) | — | — | Test d'integració executat en CI |
| NFR-03 | — | ✅ (EC-10) | — | — | Test d'integració amb payload d'injecció |
| NFR-04 | — | ✅ (EC-08) | — | — | Assaig estàtic de rutes en CI |
| NFR-05 | — | ✅ (seed de 500 ítems + mesura de latència) | — | — | Test de càrrega lleuger en CI o script dedicat |
| NFR-06 | — | — | ✅ (Lighthouse amb perfil de xarxa 3G simulada) | — | Auditoria automatitzada en CI o pre-release |
| NFR-07 | — | ✅ (EC-15 automatitzat si l'entorn ho permet) | — | ⚠️ | Script de repetició (`docker kill` × 10) |
| NFR-08 | ✅ | ✅ (`/healthz` amb DB caiguda simulada) | — | — | Test d'integració |
| NFR-09 | — | — | — | ✅ | Auditoria automatitzada (axe-core o equivalent) + revisió manual puntual |
| NFR-10 | — | — | ✅ (EC-16, AC-16) | — | Test E2E que llegeix el contingut d'`aria-live` |
| NFR-11 | ✅ | ✅ (EC-04 cicle desar/llegir) | — | — | Test d'integració amb corpus de caràcters |

**Notes de cobertura:**

- Els casos S4, S5, S6, S1 (CSRF complet) i S2 (segrest de sessió) del
  `PLAN.md` §3/§7.2 **no** formen part d'aquest AC/EC set — pertanyen a
  NIU-4 (autenticació real) i es testejaran allà. Aquest document només
  cobreix S3, S7, S8 i la restricció "cap mutació via GET" (part de S1).
- EC-15 (reinici a mig escriptura) i NFR-07 depenen de poder matar el
  procés en marxa de manera controlada; si l'entorn de CI no ho permet
  de manera fiable, es documenta com a procediment manual repetible en
  lloc d'un test automatitzat de CI — però **segueix sent obligatori**
  abans de donar la història per feta, no opcional.
- La convergència eventual (AC-08, AC-09) es testeja simulant dos
  clients (dues sessions HTTP) contra el mateix servidor, no amb
  navegadors reals, excepte per a exploració puntual.

## 7. Out of scope (explicit)

- Autenticació real, pantalla de login, gestió de contrasenyes (NIU-4).
- Tokens de sessió, cookies `HttpOnly/Secure/SameSite`, doble-submit CSRF (NIU-4 — S1 complet, S2, S5, S6).
- Rate limiting per força bruta (S4 — NIU-4/NIU-2).
- Desplegament, CI/CD, DNS, Cloudflare Access, backup (NIU-2).
- Instrumentació OTEL / traces (NIU-3).
- Quantitats numèriques per ítem — la quantitat viu dins del text del nom (p. ex. "Llet x2").
- Notificacions en temps real (SSE o equivalent) — només sondeig (~10s) i refetch al focus.
- Streaks, punts, classificacions (gamificació avançada) — només avatar i confeti bàsic en aquesta versió.
- Multi-llar, rols, permisos, convidar usuaris.

## 8. Open questions

Cap pregunta oberta bloquejant Stage 2: les decisions prèviament
`[OBERT]` a `PLAN.md` (duplicats i mecanisme de sincronització) han estat
resoltes pel titular humà i s'han incorporat com AC-08, AC-09, EC-06 i
EC-07 en aquest document.

- [ ] Cap.
