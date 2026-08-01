# Niu — Arquitectura

> El *com*. El *què* i el *per què* viuen a [overview.md](overview.md).
> El detall vinculant i les decisions congelades són a [PLAN.md](../PLAN.md).

## Vista general

Un sol binari de Go, un sol contenidor, un sol volum.

```
        Internet
           │
    ┌──────▼──────┐
    │ Cloudflare  │  DNS proxied (registre A) + WAF + TLS
    │   + Access  │  Access limita l'accés als dos correus
    └──────┬──────┘
           │ 443
    ┌──────▼──────┐
    │  Traefik    │  routing per labels, rate limit
    └──────┬──────┘
           │ 8080
    ┌──────▼───────────────────────────────┐
    │  contenidor niu                       │
    │  ┌─────────────┐  ┌────────────────┐  │
    │  │ API JSON    │  │ embed.FS       │  │
    │  │ /api/v1/*   │  │ web/ (HTML/JS) │  │
    │  └──────┬──────┘  └────────────────┘  │
    │         │                             │
    │  ┌──────▼──────┐                      │
    │  │ SQLite WAL  │  volum niu-data      │
    │  │ /data/niu.db│                      │
    │  └─────────────┘                      │
    └──────────┬────────────────────────────┘
               │ OTLP/HTTP :4318
        ┌──────▼───────┐
        │ otel-collector│ → OpenObserve
        └──────────────┘
```

## La decisió central: binari únic

El binari serveix l'API **i** els fitxers estàtics. Això és una decisió de
**desplegament**, no d'acoblament.

La conseqüència important és de seguretat: mateix origen → cookies
`SameSite=Strict` → **el CSRF deixa de ser possible per construcció**, no
per una mitigació que puguem implementar malament.

L'alternativa (front a Cloudflare Pages) hauria obligat a `SameSite=None`,
CORS amb credencials i defensa CSRF activa i permanent, a canvi de cap
avantatge per a una app de dues persones.

Regla que els agents han de respectar: **l'API i el servidor d'estàtics
són independents.** L'API no renderitza HTML mai. El frontend parla amb
`/api/v1/*` per `fetch`, exactament com ho faria qualsevol altre client.
Res de templating al servidor.

## Capes

```
app/
├── cmd/niu/           punt d'entrada, cablejat
└── internal/
    ├── config/        env → struct, validació fail-fast
    ├── store/         SQLite, migracions
    ├── items/         domini: llista i rebost
    ├── auth/          sessions (NIU-4)
    ├── httpapi/       handlers, middleware, routing
    └── observability/ OTEL (NIU-3)
```

`internal/` impedeix per compilador que ningú importi aquestes peces des
de fora. El domini (`items`) no coneix HTTP ni SQL: rep interfícies.

## Dades

Quatre taules: `users`, `sessions`, `items`, `events`. Esquema complet a
[PLAN.md §2.4](../PLAN.md).

Dues decisions que val la pena entendre:

**Un ítem és una fila que canvia de `location`.** Moure de compra a rebost
és un `UPDATE`, no un esborrat més una inserció. Això conserva qui l'ha
mogut i quan, i deixa la porta oberta a la gamificació sense refactor.

**`position` és `REAL`.** Inserir entre `1.0` i `2.0` és posar `1.5` —
una fila tocada, no renumerar la llista sencera.

`events` és append-only i ningú la llegeix a la v1. Costa vint línies ara
i estalvia un backfill dolorós després.

## SQLite

Obert amb `journal_mode(WAL)`, `busy_timeout(5000)` i `foreign_keys(on)`.
WAL importa perquè els dos usuaris poden escriure alhora.

Driver `modernc.org/sqlite` — Go pur, sense CGO. És el que permet un
binari estàtic i una imatge distroless. **No fer servir
`mattn/go-sqlite3`.**

Per a dues persones i llistes de la compra, SQLite no és una limitació:
és la mida correcta. Si algun dia calgués Postgres, el canvi és a
`internal/store/` i prou.

## Seguretat

Deu amenaces amb la seva mitigació, cadascuna amb un test que **executa
l'atac** i verifica que falla. Taula completa a [PLAN.md §3](../PLAN.md),
tests a `test-plan.md`.

El punt més delicat és temporal: com que l'autenticació real és l'últim
item, entre el desplegament i NIU-4 l'app estarà pública sense login. Ho
tanca **Cloudflare Access** amb els dos correus, configurat al mateix
desplegament.

## Observabilitat

Traces OTLP cap al col·lector compartit (`otel-collector:4318`), que les
reenvia a OpenObserve. L'app no porta credencials: les té el col·lector.

Niu serà la primera instrumentació Go amb SDK d'aquesta plataforma. Dos
paranys documentats a [PLAN.md §5.4](../PLAN.md) — el port i el protocol —
que causen pèrdua silenciosa de dades si s'ignoren.

## Preparat per a mòbil (sense construir-ho)

L'API és JSON i està separada de la presentació; els tokens de sessió són
opacs i viuen al servidor; el prefix `/api/v1/` ja hi és.

Si algun dia hi ha app Flutter, el backend només necessita llegir el token
d'una capçalera `Authorization: Bearer` com a alternativa a la cookie —
unes deu línies en un middleware, reaprofitant tota la generació,
hashing, expiració i revocació.

**No s'implementa res d'això ara.** Vegeu [PLAN.md §9](../PLAN.md).
