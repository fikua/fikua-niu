---
artefact: status
key: "NIU-5"
title: "Compres grans i projectes de casa"
status: "archived"
outcome: "shipped"
owner: "Fikua Studio"
shipped_on: "2026-08-03"
pr_links: ["https://github.com/fikua/fikua-niu/pull/1"]
sources:
  - "Google SRE — Postmortem template (no-blame, lightweight)"
created: "2026-08-03"
updated: "2026-08-03"
---

# STATUS — Compres grans i projectes de casa

> **What this is.** A short, no-blame closing note appended when `/archive`
> runs. Subset of the Google SRE postmortem template — adapted for
> *change closure*, not incident response. **Not a retrospective.**
> Goal: future-you finds what happened, what changed, and what to watch.

## 1. Summary (TL;DR)

Niu guanya un segon espai, separat de la llista de la compra, per fer
seguiment de compres grans i projectes de casa amb un cicle de vida propi
(`idea` → `decidit` → `fet`, en qualsevol direcció). Va passar per les 15
AC / 17 EC / 8 NFR de `requirements.md` sense retallar abast. La primera
passada de `/audit` va tornar `CHANGES_REQUESTED` per dos findings
`major` (una race condition real i noms reals filtrats a un repo públic);
tots dos es van corregir i verificar abans d'una segona auditoria que va
donar `APPROVED`. Fusionat el 2026-08-03.

## 2. What shipped

- **Proposal:** `./proposal.md`
- **Requirements:** `./requirements.md`
- **Design:** `./design.md`
- **Tasks:** `./tasks.md`
- **Review:** `./review.md`
- **PR(s):** <https://github.com/fikua/fikua-niu/pull/1>

Nou domini `internal/projects` (repository/service/handler), migració
`003_projects.sql`, endpoints REST sota `/api/v1/projects`, i una nova
pantalla (`projects.html`) amb llista única i accent terracota,
consistent amb l'estètica càlida establerta a `PLAN.md` §4.

## 3. Deviations from the plan

**Intentional changes during build**

- Stage 1.5 (UX) es va ometre deliberadament per a aquest ítem — les
  decisions visuals (llista única, sense drag-and-drop, accent terracota
  reutilitzat) es van resoldre directament a `design.md` §7/ADR-04 amb
  confirmació a la porta humana de Stage 2, en lloc d'una etapa de disseny
  visual separada.
- Reutilització d'`items.NormalizeName` des d'`internal/projects` (ADR-02
  de `design.md`) en lloc de duplicar la lògica de normalització — decisió
  conscient, confirmada a la porta humana.

**Unintentional gaps discovered**

- **Race condition (F-23, major):** `Service.ChangeState` llegia l'estat
  previ amb un `repo.Get()` no transaccional abans de l'`UPDATE` dins
  `BEGIN IMMEDIATE`. Sota canvis d'estat concurrents, l'event
  `project_state_changed` podia registrar un `"from"` incorrecte,
  corrompent silenciosament l'historial d'auditoria que NFR-01 exigeix.
  Detectat a la primera passada de `/audit` (verificat empíricament amb
  una prova de 200 rondes concurrents), corregit movent la lectura dins
  la mateixa transacció. Verificat amb `go test -race` complet (3
  execucions netes) i un test de regressió nou
  (`TestProjects_ConcurrentStateChange_Repeated` +
  `assertStateChangedEventChainIsConsistent`). **Aquesta és la tercera
  vegada que aquest patró de check-then-act apareix al projecte** (ja
  havia passat a NIU-1 i al rate-limiter de NIU-4) — vegeu §5.
- **Noms reals en un repo públic (F-20, major):** el procés de `/define`
  d'aquesta sessió va introduir "Oriol"/"Joana" a `BACKLOG.md` i als
  `proposal.md` de NIU-5 i NIU-6, violant `PLAN.md` §3 S11 (el repo
  `fikua/fikua-niu` és públic). Corregit substituint per "Usuari A"/
  "Usuari B" a tots els fitxers afectats. Detectat per `security-engineer`
  a `/audit`, no per cap revisió humana prèvia — cap procés automàtic
  comprovava això abans.
- **Cobertura de test parcial (no bloquejant):** NFR-02 (zero `innerHTML`
  amb dades d'usuari) no té cap comprovació estàtica automatitzada en CI
  — només es va verificar manualment llegint el codi. NFR-07 (anunci
  `aria-live` en canvi d'estat) només està testejat per al canvi fet per
  l'actor local, no pel cas de canvi reflectit per sondeig remot que
  AC-12 exigeix explícitament.

## 4. Follow-ups

- [ ] NIU-2 — Desplegament (ja capturat, pendent — desplegarà aquest canvi a producció)
- [ ] NIU-8 — Comprovació estàtica en CI que bloquegi `innerHTML` amb dades d'usuari (gap NFR-02 identificat a `/audit`)
- [ ] NIU-9 — Test d'`aria-live` per a canvi d'estat reflectit per sondeig remot (gap NFR-07 identificat a `/audit`)
- [ ] NIU-10 — Revisar el patró recurrent de check-then-act (F-23, ja la tercera vegada al projecte — NIU-1, NIU-4, ara NIU-5)

## 5. Lessons (optional, no-blame)

- **What worked well:** la revisió de seguretat anticipada (abans d'aprovar Stage 2 de NIU-6) va evitar que un forat SSRF real arribés a `tasks.md`; el mateix patró de "revisar el disseny abans d'implementar" no es va aplicar a NIU-5 perquè el risc no semblava tan evident — però el resultat de `/audit` mostra que hauria estat útil igualment per a la race condition.
- **What we'd do differently:** el mateix defecte de check-then-act ja s'havia vist a NIU-1 i NIU-4. Si aquest patró s'hagués documentat com a convenció explícita del projecte després de la primera vegada, probablement `fullstack-developer` l'hauria evitat proactivament a NIU-5 en lloc de repetir-lo una tercera vegada.
- **What surprised us:** la filtració de noms reals no la va detectar cap pas humà del procés — només `security-engineer` a `/audit`, perquè el repo és públic i `PLAN.md` §3 S11 ja ho prohibia explícitament. Cap hook ni validador automàtic ho comprova encara.

## 6. References

- `PLAN.md` §2.4 (filosofia de model de dades), §3 (seguretat, incloent S11)
- `docs/changes/NIU-1-*/review.md` i `docs/changes/NIU-4-*/review.md` — instàncies prèvies del mateix defecte de check-then-act
