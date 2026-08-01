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

> Pendent — `ux-ui-designer` completarà aquesta secció a l'Etapa 1.5 amb
> les especificacions visuals, fluxos de pantalla i mapatge al sistema
> de disseny.

## 9. Preguntes obertes per a la porta humana

- Cap pregunta pendent de decisió de producte: els dos punts que estaven
  oberts al backlog (duplicats i model de sincronització) ja s'han
  resolt amb l'usuari humà i es reflecteixen com a decisions a §6 i §7
  d'aquesta proposta.
- Confirmar que la mesura d'èxit qualitativa descrita a §5 (dues
  setmanes d'ús sense recórrer a WhatsApp) és acceptable com a criteri,
  donat que Niu no tindrà analítica de producte.
