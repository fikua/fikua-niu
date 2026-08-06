---
key: NIU-11
artefact: status
status: done
shipped: 2026-08-07
path: /quick
commit: 7bfd197
---

# NIU-11 — STATUS

Enviat via `/quick` (sense PR, push directe a `main`). L'auditoria
`APPROVED` és la porta de revisió d'aquest camí.

## Abast entregat

URL opcional amb previsualització (miniatura) als projectes, replicant
el patró d'`internal/ideas` (NIU-6): `fetchsafe.FetchPreview` +
worker pool compartit + `preview_status`. Cap codi nou de scraping,
cap segon client HTTP.

## Desviacions respecte del pla

- `preview_status` és NULL-able (a `activity_ideas` és `NOT NULL DEFAULT
  'pending'`). Un projecte sense URL no té cap previsualització pendent.
- Miniatura de **32px, no 48px** com deia T-10. Els 48px feien créixer la
  fila 16px, contradient el criteri d'acceptació que la mateixa tasca
  fixava. Detectat a `/audit`.
- `ideas.validateURL` → `ideas.ValidateURL` (exportada). T-04 exigia
  reutilitzar la validació en lloc de duplicar-la, i no era exportada.

## Auditoria

Ronda 1 `CHANGES_REQUESTED`, ronda 2 `APPROVED`. Dos bloquejants:

1. **Cursa de dades** al doble de test de `projects` — havia perdut el
   mutex que la seva contrapartida d'`ideas` sí que porta. Passava CI en
   silenci perquè `commands.test` del manifest no porta `-race`.
2. **Cap test cobria la migració contra una fila preexistent** — just
   l'escenari per al qual existeix la decisió de fer `preview_status`
   NULL-able.

Seguretat: `APPROVED`, 0 troballes bloquejants ni majors.

## Seguiment obert

- **`perf-3g.spec.js` falla** (~1035ms contra un llindar de 1000ms).
  **Preexistent, no causat per NIU-11**: verificat apilant tots els
  canvis i executant-lo sobre `main` net, on falla pitjor (1043ms).
  Mereix un ítem propi.
- **`docs/test-plan.md` no té entrada per a NIU-11.** Conseqüència
  estructural de `/quick`, que se salta `/define`. `PLAN.md §7` declara
  aquest document com a contracte vinculant — cal decidir si `/quick` hi
  ha d'escriure o si els ítems `/quick` en queden exempts.
- **F-04** (cap test d'integració exercita el wiring del pool compartit
  que `main.go` porta a producció) — no bloquejant per acord mutu.
- Editar la URL d'un projecte ja creat va quedar fora d'abast:
  `PATCH /projects/{id}` només accepta `state`.

## Desplegament

Pendent. La imatge s'ha de construir i desplegar com de costum — vegeu
la nota del workflow al fil de la sessió: `Build & Publish Docker Image`
no sempre dispara sol amb un push a `app/**`.
