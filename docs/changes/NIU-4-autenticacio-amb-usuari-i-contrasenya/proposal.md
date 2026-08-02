---
artefact: proposal
key: "NIU-4"
type: "story"
title: "Autenticació amb usuari i contrasenya"
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

# Proposal — Autenticació amb usuari i contrasenya

> **What this is.** A one-page narrative that frames the problem, the
> proposed solution, the users it serves, and the value it delivers.
> Inspired by Amazon's PR/FAQ (front half only — no FAQ section) and the
> Lean Canvas blocks for problem/customer/value/solution. **Read top to
> bottom in under 3 minutes.**

## 1. Headline

Niu demana usuari i contrasenya abans de deixar veure o tocar la llista, tancant la finestra d'exposició pública que fins ara cobria el login fictici.

## 2. Problema

- NIU-1 va deixar l'autenticació **stubbed**: qualsevol persona que
  arribi a l'aplicació es tracta automàticament com a `Usuari A`, sense
  cap credencial. Això és correcte mentre l'app només corre en local,
  però NIU-2 (desplegament) l'exposarà a `https://niu.fikua.com`, a
  internet obert.
- Sense una barrera real d'entrada, qualsevol persona amb l'URL podria
  llegir i modificar la llista de la compra i el rebost d'una llar real
  — no és informació sensible en el sentit legal, però és privada i el
  propietari del producte ho ha marcat com a requisit explícit ("segura
  tot i ser simple", `overview.md` §Principis).
- L'ordre d'execució del projecte es va invertir precisament per
  aquesta raó (`PLAN.md` §8, decisió del 2026-08-02): desplegar sense
  autenticació real i tapar el forat amb un pedaç d'infraestructura
  (Cloudflare Access) obligava a inventar un patró de desplegament sense
  precedent a la plataforma. Enviar l'autenticació abans que el
  desplegament elimina el problema en origen.
- Aquest article ja té el seient tècnic preparat: NIU-1 va introduir la
  interfície `auth.Authenticator` (ADR-03, `design.md` de NIU-1)
  precisament perquè aquest canvi fos una substitució d'implementació,
  no una obra nova.

## 3. Client

- **Primari:** els dos membres de la llar que fan servir Niu dia a dia
  (`Usuari A` i `Usuari B` — noms genèrics perquè el repositori és
  públic, S11).
- **Secundari:** l'operador de la plataforma (la persona que desplega i
  manté Niu), que és qui provisiona les credencials via variables
  d'entorn en el moment del desplegament — no hi ha cap tercer rol
  d'administració dins de l'aplicació.

## 4. Solució proposada

Afegir una pantalla de login (mateix llenguatge visual càlid que la
resta de Niu) que demana usuari i contrasenya, i que en validar-los
obre una sessió identificada per una cookie. Un cop dins, l'ús de l'app
és exactament el que ja existeix (NIU-1); en tancar sessió o deixar-la
caducar, cal tornar a introduir les credencials. No hi ha registre ni
recuperació de contrasenya: només existeixen dues persones i les seves
credencials les fixa l'operador abans de desplegar, no l'aplicació.
Tècnicament, això substitueix la implementació fictícia del seam
d'autenticació que ja existeix (`auth.Authenticator`) per una de real
basada en contrasenya — la resta de l'aplicació no n'hauria de notar
la diferència.

## 5. Valor i mesura d'èxit

- **Valor:** risc — elimina l'exposició pública d'una app privada abans
  que existeixi cap desplegament públic (NIU-2 en depèn explícitament).
  També és el requisit que desbloqueja NIU-2: sense aquest ítem, el
  desplegament no pot començar (`BACKLOG.md`, `PLAN.md` §8).
- **Mesura d'èxit:** cap acció de lectura o escriptura sobre la llista
  és accessible sense una sessió vàlida (comprovat per la bateria de
  seguretat de §7.2 de `PLAN.md`, ítems S1/S2/S4/S5/S6/S9), i una
  persona amb les credencials correctes pot completar el cicle complet
  login → ús → logout sense fricció perceptible.

## 6. Abast i fora d'abast

**In scope**

- Pantalla de login amb el mateix sistema visual que la resta de Niu.
- Inici de sessió (`POST /api/v1/auth/login`) i tancament de sessió
  (`POST /api/v1/auth/logout`).
- Sessió representada per un testimoni opac (mai un JWT), no llegible
  ni desxifrable pel navegador, guardat en una cookie que el JavaScript
  no pot llegir.
- Protecció contra l'ús de la sessió d'una altra persona sense el seu
  consentiment (falsificació de peticions entre llocs — CSRF).
- Límit d'intents d'inici de sessió per evitar l'endevinalla per força
  bruta de contrasenyes.
- Missatge d'error que no permet distingir "aquest usuari no existeix"
  de "la contrasenya és incorrecta".
- Caducitat de la sessió amb el pas del temps i neteja de sessions
  vençudes.
- Substitució de l'autenticació fictícia (NIU-1) per la real, sense
  requerir canvis a la resta de funcionalitats ja existents (llista de
  la compra i rebost).
- Provisió de les dues úniques credencials vàlides per part de
  l'operador en el moment del desplegament (no des de dins de l'app).

**Out of scope (explicit)**

- Registre de nous usuaris.
- Recuperació o canvi de contrasenya des de l'aplicació.
- Gestió d'usuaris, rols o permisos.
- Qualsevol mecanisme d'autenticació alternatiu (OAuth de tercers,
  inici de sessió social, codis d'un sol ús). Es va provar Google OAuth
  el mateix dia que es va escriure aquest pla i es va revertir perquè
  introduïa una dependència externa (configuració a Google Cloud
  Console) que bloquejava el progrés — decisió tancada, no es reobre
  (`PLAN.md` §0, §8).
- Notificacions de nou inici de sessió, historial de sessions actives,
  o tancament remot de sessions.

**Deferred to a later change**

- Suport per a un tercer tipus de credencial (p. ex. token portador per
  a una futura app mòbil) — el disseny ja ho deixa obert (`PLAN.md`
  §9), però no s'implementa aquí.

## 7. Riscos i incògnites

| Risc / incògnita | Severitat | Hipòtesi de mitigació |
| --- | --- | --- |
| Un atacant intenta endevinar la contrasenya per força bruta abans que el límit d'intents s'apliqui prou d'hora | MEDIUM | Límit d'intents per IP i per usuari amb espera creixent, ja fixat com a requisit vinculant a `PLAN.md` §3 (S4) |
| Un missatge d'error diferent per "usuari no existeix" vs "contrasenya incorrecta" permet a algú saber quins noms d'usuari són vàlids | MEDIUM | Missatge idèntic i temps de resposta pràcticament constant en ambdós casos (S5), verificat amb un test dedicat |
| Les dues úniques contrasenyes vàlides queden fora de sincronia entre el que l'operador creu haver configurat i el que l'app espera, deixant l'app inaccessible després d'un desplegament | LOW | L'app ha de rebutjar arrencar si falta alguna de les variables d'entorn requerides (coherent amb la validació fail-fast ja establerta a `PLAN.md` §6) |
| La sessió d'una persona queda oberta en un dispositiu compartit o robat més temps del necessari | LOW | Caducitat de sessió i tancament de sessió explícit, sempre invalidat també al servidor, no només al navegador |
| Aquest ítem bloqueja NIU-2 (desplegament); qualsevol ambigüitat aquí retarda tota la resta del pla | MEDIUM | Cap pregunta funcional pendent detectada en aquesta proposta (§9); si `requirements.md` en revela alguna, es resol abans de la porta humana |

## 8. Visuals (optional — populated by `ux-ui-designer` in Stage 1.5)

Pendent — `ux-ui-designer` afegirà aquí la pantalla de login (mateixos
tokens de disseny que NIU-1: crema/verd molsa/terracota, tipografia
arrodonida) durant l'Etapa 1.5, un cop aquesta proposta i els
requeriments quedin aprovats.

## 9. Open questions for the human gate

- Cap pregunta funcional pendent. L'abast, els dos rols d'usuari i el
  mecanisme (usuari/contrasenya) ja estan fixats i tancats a `PLAN.md`
  §0 i §8 — no es reobren aquí.
