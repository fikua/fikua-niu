# Changelog

Tots els canvis destacables d'aquest projecte es documenten en aquest fitxer.

El format segueix [Keep a Changelog](https://keepachangelog.com/ca/1.1.0/),
i el projecte s'adhereix a [Semantic Versioning](https://semver.org/lang/ca/).

## [Unreleased]

### Added

- Brief mestre (`PLAN.md`) amb arquitectura, seguretat, desplegament i pla de proves.
- Manifest `fikua.yml` per al framework Fikua SDD.
- Backlog inicial amb quatre items (NIU-1 … NIU-4).
- **NIU-1 — Llista de la compra ↔ rebost (auth stubbed):** implementació
  completa del backend Go (`internal/config`, `internal/store`,
  `internal/items`, `internal/auth`, `internal/httpapi`) i del frontend
  vanilla (`app/web/`) amb FLIP, actualització optimista, confeti d'un
  sol tret, pestanyes mòbil i anuncis `aria-live`. Migracions goose
  embedded, SQLite pur Go (`modernc.org/sqlite`), normalització NFC de
  noms d'ítem (ADR-02), seguretat S3/S7/S8, i seam d'autenticació
  (`auth.Authenticator`) llest per NIU-4. Suite de tests: unitaris i
  d'integració en Go (26 tests) + E2E Playwright (11 tests, inclou
  auditoria axe-core WCAG 2.2 AA) + procediment manual `killtest`
  (10 execucions de `SIGKILL` sense corrupció, ADR-04/REL-01).
