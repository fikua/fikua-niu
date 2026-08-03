---
artefact: proposal
key: "NIU-5"
type: "story"
title: "Compres grans i projectes de casa"
status: "approved"
owner: "product-manager"
parent_key: null
related_keys: ["NIU-1"]
sources:
  - "Lean Canvas (Ash Maurya) — problem/solution framing"
  - "Amazon PR/FAQ — narrative front half"
created: "2026-08-02"
updated: "2026-08-02"
---

# Proposta — Compres grans i projectes de casa

> **Què és això.** Narrativa d'una pàgina que emmarca el problema, la
> solució proposada, a qui serveix i el valor que aporta. Lectura en
> menys de 3 minuts.

## 1. Titular

Niu guanya un segon espai, separat de la llista de la compra, per no
perdre idees i projectes de casa que avui es dissolen en converses i
notes soltes.

## 2. Problema

- La llista de la compra (NIU-1) resol el dia a dia — què falta i què ja
  hi ha a casa — però assumeix un cicle de vida de dos estats (*a
  comprar* ↔ *rebost*) que no serveix per a res que no es compri i
  consumeixi en un sol viatge al súper.
- Compres més grans (un moble, un televisor nou) i projectes de casa
  (pintar una habitació) tenen un cicle de vida diferent i més llarg:
  primer són una idea que algú comenta, després es decideix fer-la
  (potser amb un pressupost al cap), i finalment es fa realitat. No
  encaixen a cap caixa de NIU-1 sense forçar-ne el significat.
- Avui aquestes idees viuen igual que abans de Niu: en converses de
  WhatsApp que es perden entre altres temes, o en notes soltes sense cap
  lloc comú. No hi ha una única font de veritat sobre quines idees estan
  sobre la taula, quines ja s'han decidit, ni quines ja s'han fet.
- El cost és el mateix fregament domèstic que NIU-1 ja va identificar a
  [overview.md](../../overview.md) — no catastròfic, però constant: una
  idea es comenta un dia i es torna a proposar mesos després perquè cap
  dels dos recorda si ja se n'havia parlat o en quin punt es va quedar.

## 3. Client

- **Primari:** Usuari A i Usuari B, els dos usuaris de Niu, en peu d'igualtat
  — mateix rol que estableix [overview.md](../../overview.md) §"Per a
  qui": no hi ha administrador ni jerarquia.
- **Secundari:** cap. Mateixa naturalesa de producte privat per a dues
  persones que la resta de Niu.

## 4. Solució proposada

Un espai nou dins de Niu, separat de la llista de la compra, per
recollir compres grans i projectes de casa des que són només una idea
fins que es fan realitat. Cada element hi viu tot el seu recorregut —
des del moment en què algú l'anota com a idea, passant per quan es
decideix tirar-la endavant, fins que es dona per feta — sense necessitat
de tornar-lo a escriure enlloc ni de perdre'l en una conversa. Els dos
usuaris veuen el mateix estat de cada idea i qui l'ha proposat o
actualitzat per últim cop, seguint el mateix principi de font única de
veritat que ja aplica NIU-1 a la llista de la compra.

## 5. Valor i mesura d'èxit

- **Valor:** experiència d'usuari (UX) — elimina el mateix tipus de
  fregament domèstic que NIU-1 ja va resoldre per a la compra del dia a
  dia, ara aplicat a compres i projectes de cicle més llarg que avui no
  tenen cap lloc propi.
- **Mesura d'èxit:** com que Niu no té analítica de producte
  ([overview.md](../../overview.md) §"Com sabrem que funciona"), la
  mesura és qualitativa: durant almenys dues setmanes d'ús normal, tota
  idea de compra gran o projecte de casa que sorgeixi en una conversa es
  registra a Niu en lloc de quedar-se només parlada, i cap dels dos
  usuaris necessita preguntar a l'altre "en quin punt havíem deixat allò
  de...?".

## 6. Abast i fora d'abast

**En abast**

- Un espai dins de Niu, diferenciat de la llista de la compra, per a
  compres grans i projectes de casa.
- Cicle de vida propi per a cada element, diferent del de la llista de
  la compra: des d'idea inicial fins a fet/comprat, passant per un estat
  intermedi de decisió (l'estructura exacta d'aquest flux — llista
  simple o flux amb més d'un estat intermedi — és una pregunta oberta,
  §9).
- Afegir, actualitzar l'estat i eliminar un element; cada element mostra
  qui l'ha afegit i qui l'ha actualitzat per últim cop, seguint el mateix
  principi d'atribució que NIU-1 (§6 de la seva proposta).
- Els dos usuaris veuen els canvis fets per l'altre seguint el mateix
  mecanisme d'actualització que ja utilitza la llista de la compra
  (interrogació periòdica), sense necessitat de recarregar manualment.
- Persistència: els elements sobreviuen a reinicis del servei, igual que
  la llista de la compra.
- Protecció bàsica de la informació introduïda pels usuaris, seguint el
  mateix estàndard que NIU-1: cap dada introduïda per un usuari
  s'interpreta com a codi executable, i el sistema no filtra mai detalls
  tècnics interns en cas d'error.

**Fora d'abast (explícit)**

- Qualsevol integració amb comerços, cercadors de preus, o enllaços a
  productes concrets — aquest ítem és seguiment d'estat, no un
  comparador de compres.
- Notificacions push o recordatoris programats.
- Multi-llar, rols o permisos — mateixa exclusió permanent que la resta
  de Niu ([overview.md](../../overview.md) §"Què no fa la v1").
- Qualsevol capa de gamificació (ratxes, punts) — mateixa postura que
  NIU-1: es descarta fins que hi hagi ús real.
- Relació o dependència tècnica amb la llista de la compra (NIU-1) més
  enllà de compartir la mateixa app i els mateixos dos usuaris — són
  dues col·leccions d'informació independents amb cicles de vida
  diferents, no una extensió de la mateixa entitat.

**Diferit a un canvi posterior**

- Camp de notes lliures — descartat a la porta humana; `requirements.md`
  §8 confirma que v1 inclou pressupost (text lliure) i data objectiu,
  però no notes.
- Qualsevol vista d'historial o d'anàlisi sobre projectes passats més
  enllà de veure l'estat actual de cada element.

## 7. Riscos i incògnites

| Risc / incògnita | Severitat | Hipòtesi de mitigació |
| ---------------- | -------- | ---------------------- |
| El model d'estats (llista simple de 3 estats vs. flux amb graons intermedis) queda subdimensionat o sobredimensionat per a l'ús real de dues persones | MEDIUM | Decisió pendent explícita per a `requirements.md` (§9) — es resol amb el propietari abans de tancar la porta humana d'aquesta proposta, no es fa una suposició de disseny aquí. |
| Sense camp de pressupost ni data objectiu, l'espai es queda curt per a projectes que sí que els necessiten (p. ex. "pintar l'habitació" amb un cost estimat) | LOW | Decisió pendent explícita per a `requirements.md` (§9); si es difereix, els elements segueixen sent útils només com a seguiment d'estat i es pot afegir un camp més endavant sense trencar res existent. |
| Confusió d'usuari entre aquest espai nou i la llista de la compra (NIU-1), si visualment s'assemblen massa tot i tenir semàntiques diferents | LOW | `ux-ui-designer` haurà de diferenciar clarament els dos espais a l'Etapa 1.5 — es referencia aquí perquè no es resol en aquesta proposta. |
| Elements que queden "aparcats" indefinidament a l'estat d'idea sense que ningú els decideixi ni els descarti, acumulant soroll | LOW | Fora d'abast d'aquesta proposta (no s'introdueix caducitat automàtica); es pot revisar en un canvi posterior si l'ús real ho demana. |

## 8. Visuals

Pendent — `ux-ui-designer` hi afegirà l'especificació visual a l'Etapa
1.5, un cop aquesta proposta i els requeriments quedin aprovats. Haurà
de decidir, en particular, com es diferencia visualment aquest espai de
la llista de la compra (NIU-1) tot mantenint la mateixa estètica càlida
de [PLAN.md](../../../PLAN.md) §4.

## 9. Preguntes obertes — RESOLTES a la porta humana (2026-08-02)

- **Model d'estats:** confirmat — llista simple de tres estats (`idea` →
  `decidit` → `fet`), sense graons intermedis. Veure `requirements.md`
  §0/§8.
- **Camps addicionals:** confirmat — pressupost (text lliure) i data
  objectiu s'inclouen a v1; notes lliures queden descartades. Veure
  `requirements.md` AC-14/AC-15.
- **Relació amb `overview.md`:** confirmat — s'actualitza en aprovar
  aquest ítem (AC-13).
