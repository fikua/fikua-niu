---
key: NIU-11
type: story
status: in_progress
path: /quick
owner: fullstack-developer
---

# NIU-11 — URL amb previsualització (miniatura) als projectes

> Generat per `/quick`. **No hi ha `proposal.md` / `requirements.md` /
> `design.md`** — aquest fitxer és l'únic artefacte de l'ítem. El
> context substitueix les referències a AC.

## Context

Quan es planifica una compra gran (un sofà, un moble, un televisor) la
referència sol ser una URL de botiga. Avui un projecte només té nom,
estat, pressupost i data — la URL es perd en converses.

La feina és **replicar el patró ja provat a `internal/ideas`** (NIU-6),
no inventar-ne cap de nou: `fetchsafe.FetchPreview` + `WorkerPool` +
`preview_status`. Cap codi nou de scraping, cap client HTTP nou.

### Decisions preses (no reobrir)

1. **Miniatura a la fila**, no targeta expandible. La fila de projecte es
   manté compacta i escanejable: miniatura ~48px + nom clicable cap a
   l'enllaç. **Sense descripció a la UI.**
2. La URL és **opcional** — un projecte sense URL es renderitza
   exactament com avui, sense forat visual ni placeholder.
3. **Fora d'abast:** editar la URL d'un projecte ja creat (avui
   `PATCH /projects/{id}` només accepta `state`); imatge gran i
   descripció.

### Restriccions heretades

- `description` es persisteix encara que la UI no la mostri: ve gratis
  del mateix `Preview` i evita una segona migració si algun dia es
  mostra. Persistir-la no és el mateix que exposar-la — vegeu T-05.
- **XSS (NFR-02):** `projects-render.js` és `textContent`-only. La
  miniatura afegeix el primer `src` amb dada d'usuari d'aquell fitxer:
  ha de ser `setAttribute('src', ...)` amb valor validat al backend,
  mai interpolació de cadena.
- La CSP ja permet `img-src 'self' https:` — no cal tocar-la. Si la
  imatge fos `http:` la CSP la bloquejaria: `fetchsafe` ja només accepta
  `og:image` amb esquema `http(s)`, i el navegador bloquejarà `http:`
  en una pàgina `https:` (mixed content). Acceptable: la targeta
  degrada a "sense miniatura".

## Tasques

### Dades

- [x] **T-01** — Migració `005_projects_url_preview.sql`: afegir a
  `projects` les columnes `url TEXT`, `title TEXT`, `image_url TEXT`,
  `description TEXT`, `preview_status TEXT` amb el mateix `CHECK
  (preview_status IN ('pending','ready','partial','failed'))` que
  `activity_ideas`. **`preview_status` ha de ser NULL-able aquí** (no
  `NOT NULL DEFAULT 'pending'` com a `activity_ideas`): les files
  existents i tot projecte sense URL no tenen cap preview pendent, i
  `pending` per a elles seria mentida — la UI hi mostraria un spinner
  etern. Incloure el `-- +goose Down` simètric.

### Backend — domini

- [x] **T-02** — `internal/projects/projects.go`: afegir a `Project` els
  camps `URL`, `Title`, `ImageURL`, `Description` (tots `*string`) i
  `PreviewStatus *string`. Declarar `PreviewFetcher` i el tipus
  `Preview` replicant `internal/ideas/service.go` (mateixa forma, el
  paquet segueix sense importar `net/http`).
- [x] **T-03** — `Repository`: estendre `Create` perquè accepti la `url`
  i afegir `UpdatePreview(ctx, id, title, imageURL, description *string,
  status string) error`, amb el mateix contracte que
  `ideas.Repository.UpdatePreview` (zero files afectades no és error —
  el projecte pot haver-se esborrat mentre el scrape volava).
- [x] **T-04** — `internal/projects/service.go`: `NewService` accepta
  `fetch PreviewFetcher` i `pool *ideas.WorkerPool`; `Add` valida la URL
  (només si se n'ha donat una) i, quan n'hi ha, encua `resolvePreview`
  al pool. Replicar `ideas.Service.resolvePreview` inclosa la regla
  "partial amb zero camps recuperats → `failed`".
  **Un projecte sense URL no encua res i desa `preview_status` NULL.**
  Reutilitzar la validació d'esquema — no duplicar-la a mà.

### Backend — HTTP

- [x] **T-05** — `internal/httpapi/projects_handlers.go`: acceptar `url`
  (opcional) al DTO de `POST`, i exposar `url`, `title`, `image_url`,
  `preview_status` a la resposta. **No exposar `description`**: es
  persisteix però la UI no la mostra (decisió 1), i el contracte HTTP no
  ha de publicar camps que ningú consumeix.
- [x] **T-06** — `cmd/niu/main.go`: wiring. Reutilitzar el
  `fetchsafeClient` **ja existent** (T-03h: un sol client per procés,
  mai un per servei). Decidir explícitament pool compartit vs segon
  pool i deixar-ho comentat al codi; per defecte **compartir el pool
  d'`ideas`** — 6 workers ja cobreixen la càrrega d'una app per a dues
  persones, i un segon pool duplicaria el sostre de memòria sense
  motiu. Si es comparteix, renombrar la variable a `previewPool` perquè
  el nom no menteixi sobre qui l'utilitza.

### Frontend

- [x] **T-07** — `web/index.html`: afegir l'input `url` opcional al
  formulari, dins de `.add-project-optional` al costat de pressupost i
  data. `type="url"`, `aria-label`, `maxlength`. (Sense `maxlength`
  explícit — es miralla exactament l'input `#add-idea-url` existent, que
  tampoc en té; el backend no imposa cap límit de longitud sobre la URL.)
- [x] **T-08** — `web/js/projects-api.js` + `projects-store.js`: passar
  la `url` al `POST`. Mantenir el patró optimista existent.
- [x] **T-09** — `web/js/projects-render.js`: si el projecte té
  `image_url`, afegir un `<img class="project-thumb">` (~48px) al
  començament de la fila; si té `url`, el nom passa a ser un `<a>` amb
  `target="_blank"` i **`rel="noopener noreferrer"`** (sense això, la
  pàgina de destí té accés a `window.opener`). `textContent` per al nom
  i `setAttribute` per a `src`/`href` — mai `innerHTML` (NFR-02).
  `alt=""` a la miniatura: és decorativa, el nom del projecte ja
  identifica la fila; un `alt` amb el títol duplicaria la lectura al
  lector de pantalla.
- [x] **T-10** — `web/app.css`: estil de `.project-thumb` (48px, cantons
  arrodonits, `object-fit: cover`) coherent amb el llenguatge càlid
  existent. La fila **no ha de créixer en alçada** — verificar-ho amb
  una fila sense miniatura al costat d'una amb miniatura.

### Verificació

- [x] **T-11** — Tests de `internal/projects`: `Add` amb URL encua i
  resol; `Add` sense URL no encua ni deixa `preview_status`; error del
  fetch → `failed`; partial sense camps → `failed`. Seguir
  `internal/ideas/service_test.go` amb el mateix fake repo.
- [x] **T-12** — Test d'integració: `POST /api/v1/projects` amb `url`
  retorna 201 sense esperar el scrape, i la resposta exposa
  `preview_status`. Verificar que `description` **no** apareix al JSON.
- [x] **T-13** — `cd app && go build ./... && go test ./...` i
  `gofmt -l` net.

## Fet quan

- La migració aplica i reverteix net.
- Afegir un projecte amb la URL d'una botiga mostra la miniatura i el
  nom clicable; sense URL la fila es veu exactament com abans.
- Suite verda i `gofmt -l` sense sortida.
