# niu

> App privada de llar compartida per a dues persones.

**niu** és el lloc on una parella construeix la seva llar. La v1 és una
llista de la compra compartida amb dues caixes — *A comprar* i *Rebost* —
on seleccionar un ítem el mou d'una a l'altra.

No té registre, ni recuperació de contrasenya, ni gestió d'usuaris. Dos
usuaris sembrats, i prou.

## Estat

En desenvolupament. Cap item completat encara.

| Item | Estat | Què és |
|---|---|---|
| [NIU-1](BACKLOG.md) | Capturat | Llista de la compra ↔ rebost |
| [NIU-2](BACKLOG.md) | Capturat | Desplegament i CI/CD |
| [NIU-3](BACKLOG.md) | Capturat | Observabilitat (OTEL) |
| [NIU-4](BACKLOG.md) | Capturat | Autenticació real |

## Com està fet

Un **binari únic de Go** que serveix l'API JSON *i* el frontend estàtic
(via `embed.FS`), amb SQLite. Un contenidor, un volum, un desplegament.

Frontend en HTML + CSS + JavaScript vanilla: sense framework, sense npm,
sense build step.

Aquesta tria no és per estalviar feina — és perquè el mateix origen fa
que el CSRF sigui **estructuralment impossible** en comptes de ser una
cosa que cal mitigar bé. Vegeu [PLAN.md](PLAN.md) §2.1.

```
┌─────────────────────┐  ┌─────────────────────┐
│  🛒 A comprar        │  │  🥫 Rebost          │
│  ─────────────       │  │  ─────────────      │
│  ○ Llet          🐦  │  │  ● Arròs      🦊 ↩  │
│  ○ Pa            🦊  │  │  ● Oli        🐦 ↩  │
│  [ + afegir…      ]  │  │                     │
└─────────────────────┘  └─────────────────────┘
```

## Documentació

| Document | Per a què |
|---|---|
| [PLAN.md](PLAN.md) | **El brief mestre.** Arquitectura, seguretat, desplegament, ordre d'execució. Congelat i aprovat |
| [BACKLOG.md](BACKLOG.md) | Els quatre items i el seu estat |
| `docs/test-plan.md` | Casos de prova en Gherkin — el mecanisme de control principal |
| `docs/changes/<KEY>-slug/` | Artefactes SDD per item |

## Desenvolupament

```bash
cd app
go mod download
go run ./cmd/niu        # http://localhost:8080
go test ./...
```

## Desplegament

`https://niu.fikua.com` — VPS OVH, Docker Compose, Traefik, darrere
Cloudflare. Desplegament automàtic en publicar una release de GitHub.

Detall a [PLAN.md](PLAN.md) §5.

---

Construït amb [Fikua SDD System](../fikua-sdd-system).
