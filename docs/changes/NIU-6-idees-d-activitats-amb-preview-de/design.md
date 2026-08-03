---
artefact: design
key: "NIU-6"
title: "Idees d'activitats amb previsualització de link"
status: "approved"
owner: "software-architect"
requirements_path: "./requirements.md"
adr_count: 5
sources:
  - "arc42 (subset: §1 introduction, §4 solution strategy, §5 building blocks, §6 runtime, §8 cross-cutting, §11 risks)"
  - "ADR format (Michael Nygard, 2011)"
  - "C4 model — Levels 1 (context) and 2 (containers)"
  - "OWASP SSRF Prevention Cheat Sheet"
created: "2026-08-03"
updated: "2026-08-03"
security_review: "amended 2026-08-03 post security-engineer design review — F-01..F-07, see ADR-02/ADR-03/§8"
---

# Design — Idees d'activitats amb previsualització de link

> **Què és això.** La resposta tècnica als 11 AC / 18 EC / 9 NFR de
> `requirements.md`. Afegeix un domini nou (`internal/ideas`) al costat
> d'`internal/items` (NIU-1) i `internal/projects` (NIU-5), reutilitzant
> exactament els mateixos patrons de capes i el mateix middleware
> d'autenticació (NIU-4, sense cap canvi). **La part central d'aquest
> disseny és la mitigació de SSRF (NFR-05/06/07/08)** — es tracta amb el
> seu propi component, `internal/fetchsafe`, i el seu propi ADR (ADR-02),
> perquè cap altre punt del codi hagi de reimplementar-la. Referència de
> projecte: [`../../architecture.md`](../../architecture.md) §"Capes"/
> "Dades"/"Seguretat". Referència estructural: `design.md` d'NIU-5 (mateixa
> veu, mateix patró domini/store/httpapi).

## 1. Introducció i restriccions (arc42 §1)

- **Objectiu d'aquest canvi:** entregar un espai nou i independent —
  "Idees" — on desar idees d'activitats enganxant un enllaç; Niu recupera
  Open Graph (títol/imatge/descripció) del servidor, amb un mecanisme de
  recuperació de contingut extern que **mai** pot ser utilitzat per fer
  peticions arbitràries cap a xarxa interna o privada (SSRF), complint
  els 11 AC i 18 EC de `requirements.md` sense tocar `internal/items` ni
  `internal/projects`.
- **Restriccions (no negociables):**
  - Tècnica (`PLAN.md` §2/§3): mateix binari Go 1.26 + `chi/v5` +
    `modernc.org/sqlite` + `pressly/goose/v3` embedded; mateixa estructura
    `internal/<domini>` + `store` + `httpapi`; cap dependència externa
    nova per al parsing HTML/OG — únicament stdlib + `golang.org/x/net`
    (ADR-04).
  - Organitzacional: repositori **públic** — cap dada personal a fixtures
    ni tests (S11, ja establert). Cap credencial de Niu (cookie de
    sessió, token, secret) surt mai cap a un destí extern (NFR-08).
  - Seguretat (vinculant, `requirements.md` §0.3/NFR-05..08): esquema
    `http(s)` únicament; destí resolt mai en xarxa privada/loopback/
    link-local/pròpia instància, validat també a cada redirecció; timeout
    dur; límit de mida de resposta imposat mentre es descarrega, no
    després; client HTTP sortint sense credencials.
  - Funcional (ja tancat, no re-litigable): sense cicle de vida/estat,
    fallback no bloquejant (mai error dur per manca de previsualització),
    sense deduplicació d'enllaços.
  - Disseny: `proposal.md` §8 (Stage 1.5) ja fixa el token `color.mel`
    aprovat, la graella de targetes, i l'estat "Recuperant…" — aquest
    document dimensiona els contractes que aquell disseny visual assumeix
    (async, cache at save-time), no en redefineix cap detall visual.

## 2. Estratègia de solució (arc42 §4)

1. **Domini nou `internal/ideas`, no una extensió d'`items`/`projects`**
   (ADR-01) — model, cicle de vida (inexistent) i risc (SSRF) prou
   diferents per justificar un `Repository`/`Service` propis.
2. **Tot el risc SSRF es concentra en un component nou i únic,
   `internal/fetchsafe`** (ADR-02) — cap altre paquet fa peticions HTTP
   cap a destins introduïts per l'usuari. `fetchsafe` exposa una sola
   funció (`FetchPreview(ctx, rawURL) (Preview, error)`) que encapsula:
   validació d'esquema, resolució + validació d'IP abans de connectar,
   re-validació a cada salt de redirecció, timeout dur, límit de mida en
   streaming, i un `http.Client` dedicat sense cap capçalera de Niu.
3. **Scraping asíncron, no síncron dins la petició `POST`, via un worker
   pool acotat** (ADR-03, esmenat post-revisió de seguretat — F-05) —
   `POST /api/v1/ideas` desa la idea immediatament amb
   `preview_status = 'pending'` i respon `201` a l'instant; el treball
   s'encua a un pool de 4-8 workers en segon pla que executen
   `fetchsafe.FetchPreview` i fan un `UPDATE` posterior amb el resultat
   (èxit, parcial, o `failed`) — un `goroutine` sense límit per petició
   es va rebutjar per risc d'exhaurir la memòria del contenidor sota
   ràfegues simultànies. Confirma exactament el que `proposal.md` §8.2
   Estat D (targeta "Recuperant…") ja assumeix visualment.
4. **Cache at save-time: mai re-scraping en `GET`** (NFR-09) — el
   resultat es persisteix a `activity_ideas` un únic cop; llegir la
   llista és sempre una consulta SQLite, mai una petició sortint.
5. **Una idea és una fila que canvia `preview_status`, mai un
   delete+insert** (`PLAN.md` §2.4, mateix principi que `items.location`
   i `projects.state`) — el `UPDATE` de resolució del scraping toca
   només les columnes de previsualització, mai `id`/`url`/`added_by`.
6. **Sense unicitat d'enllaç** (EC-06, ja tancat) — cap índex únic sobre
   `url`; dues files independents amb el mateix enllaç són vàlides.
7. **Sense soft-delete** — `DELETE` dur, idèntic a `items`/`projects`
   (EC-15 idempotent).
8. **Reutilització completa de l'auth existent** (EC-14, ja tancat) —
   `internal/httpapi` munta les noves rutes dins del mateix grup
   `/api/v1` protegit per `WithCurrentUser` + `RequireCSRF` (NIU-4); zero
   codi d'autenticació nou.
9. **Sondeig + refetch-on-focus, mateix mecanisme d'NIU-1/NIU-5** —
   `GET /api/v1/ideas` s'afegeix al mateix cicle de `syncFromServer()`
   ja implementat a `web/js/store.js` (AC-06). Una idea en estat
   `pending` apareix als resultats del sondeig igual que qualsevol altra
   — el client no necessita cap mecanisme de sincronització nou per
   veure la resolució de l'altre usuari.
10. **Parsing Open Graph amb stdlib únicament, sense llibreria externa**
    (ADR-04) — `golang.org/x/net/html` (tokenizer stdlib-adjacent, ja
    l'ecosistema `golang.org/x/*` establert al `go.mod`) per llegir
    `<meta property="og:*">` del `<head>`, sense necessitat de parsejar
    el document sencer ni d'una dependència de scraping de tercers.
11. **Nova pestanya de navegació + `color.mel`** (ADR-05, resol AC-07,
    confirma `proposal.md` §8.0/§8.1) — tercera entrada de nivell
    superior amb el nou token de disseny aprovat a Stage 1.5.

## 3. Decisions arquitectòniques (ADRs)

### ADR-01 — Domini nou `internal/ideas`, no una extensió d'`items`/`projects`

- **Status:** accepted
- **Context:** `proposal.md` §6 exclou explícitament qualsevol relació
  de model de dades amb `items` (NIU-1) o `projects` (NIU-5) més enllà de
  compartir app i usuaris — és una col·lecció d'informació independent
  amb un propòsit diferent (arxiu d'idees amb context visual, no
  seguiment d'estat ni llista de compra).
- **Decision:** `internal/ideas` és un paquet nou, estructuralment
  paral·lel a `internal/items`/`internal/projects` (mateix trio
  `Repository`/`Service`/tipus de domini), amb taula pròpia
  `activity_ideas`. Cap paquet existent es modifica.
- **Consequences:** (+) zero risc de regressió sobre els tests ja verds
  d'NIU-1/NIU-4/NIU-5; (+) el camp `preview_status` i les columnes
  nul·lables de previsualització no contaminen cap altre domini; (+)
  coherent amb l'"Independent" d'INVEST (`requirements.md` §2). (−)
  duplicació estructural (un tercer `Repository` amb forma similar) —
  acceptat pel mateix motiu que ADR-01 d'NIU-5: més barat que acoblar
  dominis amb propòsits diferents.
- **Alternatives considered:** afegir una columna `kind` a `items` o
  `projects` (rebutjat: `proposal.md` ho prohibeix explícitament); una
  taula compartida "contingut desat" amb un `CHECK` de tipus (rebutjat:
  barreja tres cicles de vida diferents en un sol esquema).

### ADR-02 — Mitigació de SSRF: validació d'IP pre-connexió + re-validació a cada redirecció, en un component dedicat

- **Status:** accepted
- **Context:** `proposal.md` §7 flag SSRF com a risc `HIGH` i el deriva
  explícitament a aquesta etapa. `requirements.md` NFR-05/06/07/08 fixen
  llindars observables vinculants: rebuig d'esquemes no `http(s)`, rebuig
  de destins privats/loopback/pròpia instància **incloent-hi via DNS i
  via redirecció** (EC-04/EC-07 — el clàssic bypass SSRF-per-redirecció:
  un servidor hostil pot respondre `200` a la primera validació i després
  `30x` cap a `169.254.169.254` o cap al VPS mateix), timeout dur, límit
  de mida en streaming, i cap credencial de Niu filtrada (NFR-08).
  Validar només la URL d'entrada (esquema + host literal) **no** és
  suficient: (a) el nom de domini pot resoldre via DNS a una IP privada
  (EC-07, DNS rebinding), i (b) fins i tot si la primera resolució és
  pública, una redirecció posterior pot apuntar a un destí privat sense
  que Go la torni a validar — `http.Client` per defecte segueix
  redireccions sense cap hook de validació entre salts.
- **Decision:** tota la lògica viu en un paquet nou i únic,
  `internal/fetchsafe`, amb una sola funció pública `FetchPreview(ctx
  context.Context, rawURL string) (Preview, error)`. Cap altre paquet fa
  `http.Get`/`http.Client.Do` cap a una URL controlada per l'usuari.
  Mecanisme concret:
  1. **Validació d'esquema (NFR-05):** `url.Parse` + comprovació
     `scheme == "http" || scheme == "https"` **abans** de qualsevol
     activitat de xarxa. Qualsevol altre esquema (`file://`,
     `javascript:`, `ftp://`, `data:`, buit) es rebutja immediatament
     (`ErrSchemeRejected`), zero peticions.
  2. **`net.Dialer.ControlContext` com a punt únic de validació d'IP
     (NFR-06/EC-02/EC-07), amb `DisableKeepAlives: true` (F-01/F-06,
     esmenat post-revisió de seguretat):**
     - Es prefereix explícitament `ControlContext func(ctx
       context.Context, network, address string, c syscall.RawConn)
       error` per sobre de `Control` — Go **ignora** `Control` quan
       `ControlContext` és present al mateix `net.Dialer`, i
       `ControlContext` dona accés al `context` amb el timeout de 5s
       (pas 4) dins del mateix punt de validació.
     - **S'elimina la crida separada a `net.DefaultResolver.LookupIPAddr`
       que la versió anterior d'aquest ADR descrivia com a pas previ.**
       Aquella crida i la validació dins de `Control`/`ControlContext`
       formaven un parell TOCTOU (Time-Of-Check-Time-Of-Use): les
       dues resolucions DNS podien discrepar (un servidor DNS hostil pot
       respondre diferent a mil·lisegons de diferència), reobrint
       exactament el forat de DNS rebinding que EC-07 pretenia tancar.
       `ControlContext` ja rep l'adreça **totalment resolta** en el
       moment del `dial` — validar únicament aquí és suficient i evita
       la duplicació.
     - Per **cada** IP que `ControlContext` rep (la resolta pel dialer
       intern de Go, no una resolta prèviament pel nostre codi), es
       normalitza i es classifica (vegeu pas 7, ara amb `Unmap()` i
       criteri d'allowlist) abans de deixar continuar la connexió TCP.
       Si `ControlContext` retorna un error, `net.Dialer.DialContext`
       avorta abans de fer el `connect()` — **cap byte surt mai cap al
       destí prohibit**.
     - **`DisableKeepAlives: true` al `http.Transport` dedicat.** Sense
       aquesta opció, `http.Transport` reutilitza una connexió TCP ja
       oberta cap al mateix `host:port` per a salts de redirecció que
       apunten al mateix host — i una connexió reutilitzada **no torna a
       cridar `DialContext`/`ControlContext`**, saltant-se la validació
       per complet en aquest cas. Verificat empíricament per
       `security-engineer` (F-01): una cadena de 4 redireccions cap al
       mateix host va disparar `Control` una sola vegada. Per a un fetch
       de previsualització d'un sol tret i 5s de timeout, no hi ha cap
       cost de rendiment a pagar per desactivar keep-alives — no és un
       client d'ús repetit cap al mateix host.
     - Aquest mecanisme cobreix EC-07 (DNS rebinding) de franc: com que
       valida l'IP **resolta**, no el text de la URL, un domini que
       resol a `10.x.x.x` es rebutja igual que una IP literal
       `10.x.x.x` — no calen dues rutes de codi diferents.
  3. **Re-validació a cada salt de redirecció, defensa en profunditat
     (NFR-06/EC-04, esmenat — F-01):** `http.Client.CheckRedirect`
     s'estableix a una funció que fa **dues** coses, no una:
     - (a) Limita la cadena a **5 salts** (error propi passats els 5).
     - (b) **Re-valida explícitament dins de `CheckRedirect` mateix:**
       re-parseja `req.URL` (la següent petició que el client està a
       punt de seguir) i torna a comprovar que l'esquema és `http`/
       `https` (rebutja qualsevol `30x` cap a `file://`, `javascript:`,
       etc., que un servidor hostil podria intentar). Aquesta
       comprovació **no substitueix** la validació d'IP de
       `ControlContext` — és una capa addicional i barata, perquè no es
       pot assumir que `DisableKeepAlives` + `ControlContext` sigui
       l'única barrera; si algú en el futur modifica el `Transport`
       (p. ex. reactiva keep-alives per error), `CheckRedirect` encara
       talla els esquemes perillosos abans que arribin a intentar
       connectar.
     - **Nota de disseny (esmenada):** la versió anterior d'aquest ADR
       afirmava que la re-validació a cada redirecció era "estructural"
       únicament gràcies a `Control`/`Dialer`, sense cap lògica pròpia a
       `CheckRedirect`. `security-engineer` ho ha verificat empíricament
       com a **fals** sota connexions reutilitzades (F-01, vegeu més
       amunt) — es corregeix aquí amb (1) `DisableKeepAlives: true` que
       restaura la garantia "cada salt truca `ControlContext`", i (2) la
       validació explícita a `CheckRedirect` com a xarxa de seguretat
       independent, no com una redundància cosmètica.
  4. **Timeout dur (NFR-07/EC-08): 5 segons totals**, aplicat via
     `context.WithTimeout(ctx, 5*time.Second)` que embolcalla tota la
     crida (DNS + connexió + TLS + capçaleres + cos) — no només un
     `http.Client.Timeout` (que ja s'estableix igualment com a xarxa de
     seguretat de segon nivell), sinó el `context` que `FetchPreview`
     passa a `http.NewRequestWithContext`, perquè un servidor que envia
     capçaleres de seguida però allarga el cos indefinidament (EC-03/
     EC-08 combinats) també quedi tallat.
  5. **Límit de mida en streaming (NFR-07/EC-03): 2 MiB**, imposat amb
     `io.LimitReader(resp.Body, 2<<20)` **abans** de passar el lector al
     parser HTML — mai `io.ReadAll(resp.Body)` sense límit. Si el
     `LimitReader` talla el contingut (el `head` no s'ha acabat de
     llegir), es tracta com a `EC-03`/timeout de contingut: la
     previsualització es resol a fallback, no es tracta com un error
     fatal del procés.
  6. **Client HTTP dedicat, sense credencials (NFR-08):** `fetchsafe`
     construeix el seu propi `http.Client{Transport: ...}` a l'arrencada
     de l'app (una sola instància, reutilitzada — no un client nou per
     petició). **No** comparteix cap `http.Client`/`http.Transport` amb
     cap altra part de Niu (que avui no en té cap altre, però la
     separació és explícita perquè cap ús futur el reutilitzi per
     accident). No s'hi adjunta cap `Cookie`, cap capçalera
     `Authorization`, cap `NIU_SESSION_SECRET`. L'única capçalera
     sortint pròpia és un `User-Agent` identificable (`Niu-LinkPreview/1.0
     (+https://niu.fikua.com)`), perquè els servidors remots puguin
     identificar el trànsit sense necessitat de cap capçalera d'auth.
  7. **Classificació d'IP: allowlist, no enumeració de denylist (aplicat
     a `ControlContext`, IPv4 i IPv6; esmenat — F-02/F-07):**
     - **Pas 0, obligatori abans de qualsevol classificació:**
       `addr.Unmap()` sobre el `netip.Addr` rebut. Verificat
       empíricament per `security-engineer` (F-02): una adreça IPv4
       mapejada a IPv6 (`::ffff:127.0.0.1`, `::ffff:169.254.169.254`)
       retorna `false` a `IsLoopback()`/`IsPrivate()`/
       `IsLinkLocalUnicast()` si **no** es fa `Unmap()` primer — les
       comprovacions de rang es salten silenciosament sobre la forma
       mapejada. Cap crida a `Is*()` es fa mai sobre l'adreça crua sense
       passar abans per `Unmap()`.
     - **Criteri d'acceptació (allowlist, no denylist enumerada):**
       després d'`Unmap()`, l'IP només es considera vàlida per connectar
       si `addr.IsGlobalUnicast() && !addr.IsPrivate()`. Aquest criteri
       reemplaça l'enumeració de rangs prohibits de la versió anterior
       d'aquest ADR — una llista tancada de rangs (RFC 1918, link-local,
       loopback…) és necessàriament incompleta (`security-engineer` ha
       identificat blocs especials no coberts per la llista original:
       `::` no especificada, `255.255.255.255`, `240.0.0.0/4` reservat,
       `198.18.0.0/15` de benchmarking); un allowlist basat en
       `IsGlobalUnicast()` els exclou tots per construcció, sense haver
       d'enumerar-los un a un.
     - **Rebuig explícit de NAT64/6to4 (F-07):** `64:ff9b::/96` (NAT64) i
       `2002::/16` (6to4) poden re-codificar una IPv4 privada/loopback
       (p. ex. `127.0.0.1`) dins d'una adreça IPv6 que, un cop
       "desempaquetada" per l'SO/resolver, torna a resoldre's a
       `127.0.0.1`. Es rebutgen explícitament aquests dos prefixos abans
       del criteri d'allowlist general, ja que `IsGlobalUnicast()` per
       si sol no garanteix que el correspongui IPv4 encapsulat sigui
       també global.
     - **Multicast** (`IsMulticast()`) es rebutja explícitament encara
       que no sigui un vector SSRF típic — `IsGlobalUnicast()` ja el
       n'exclou, es documenta per claredat.
  8. **Denylist de noms d'amfitrió, mecanisme separat de la validació
     d'IP (NFR-06, nou pas — F-03/F-04):** la detecció de "mateixa
     instància de Niu" **no** es fa (només) via IP — es verifica
     empíricament que `niu.fikua.com` resol a una IP **pública** de
     l'edge de Cloudflare (topologia real: `app/compose.yaml` — DNS
     Cloudflare-proxied → Traefik → contenidor), que passa qualsevol
     comprovació de rang privat i permetria un SSRF en bucle cap a la
     mateixa app a través de l'edge. Per això:
     - S'afegeix una **denylist de noms d'amfitrió**, comprovada
       **abans i independentment** de la resolució/validació d'IP (no
       la substitueix — són dues capes, no una alternativa a l'altra):
       `niu.fikua.com` (hardcoded) més qualsevol valor de
       `NIU_PUBLIC_HOST` (configurable per entorn, per si el domini
       canvia o hi ha un entorn de staging amb un altre nom).
     - La mateixa denylist s'estén per cobrir els **noms de servei de la
       xarxa Docker `traefik-public`** coneguts al desplegament actual
       (`otel-collector`, `dozzle`, `openobserve`, el dashboard de
       Traefik) — són resolubles per DNS embegut de Docker des de dins
       del contenidor de Niu i no es pot assumir que el pont Docker
       visqui sempre dins d'un rang RFC1918 (assumpció no verificada de
       la versió anterior d'aquest ADR). S'adopta la solució explícita
       (llista de noms coneguts) en lloc de tractar tota resolució via
       el resolver DNS embegut de Docker com a sospitosa per defecte,
       perquè és més senzilla d'implementar i prou per a la topologia
       actual d'un sol VPS; si el nombre de serveis a `traefik-public`
       creix significativament, revisar aquesta decisió (nota per a
       `platform-engineer`).
     - Comprovació case-insensitive sobre el host de la URL **abans**
       de cap crida DNS — si coincideix, `ErrDestinationForbidden`
       immediat, zero peticions de xarxa.
- **Consequences:** (+) un únic punt d'entrada auditable
  (`fetchsafe.FetchPreview`) — `qa-engineer` pot testejar tots els EC de
  SSRF (EC-01/02/04/07) contra aquesta única funció, sense haver de
  perseguir cada lloc del codi que fa una petició externa; (+) tancar la
  validació a nivell de `Dialer`/`ControlContext` **combinat amb**
  `DisableKeepAlives: true` i la re-validació explícita a
  `CheckRedirect` fa el bypass per redirecció cobert per dues capes
  independents, no una sola assumpció (esmenat — F-01: la versió
  anterior confiava únicament en el `Dialer` i era, de fet, saltable amb
  connexions reutilitzades); (+) l'allowlist (`IsGlobalUnicast() &&
  !addr.IsPrivate()` post-`Unmap()`, F-02/F-07) és més robust que
  l'enumeració de rangs que substitueix — nous blocs de reserva
  IANA futurs queden coberts sense haver de tocar aquest ADR; (+) la
  denylist de noms d'amfitrió (F-03/F-04) tanca un vector que cap
  validació d'IP pot cobrir (edge públic que fa proxy cap a la mateixa
  app); (+) el límit de mida en streaming evita l'exhauriment de memòria
  fins i tot abans que el timeout dispari. (−) `net.Dialer.ControlContext`
  amb `syscall.RawConn` és una API de baix nivell, més delicada de
  testejar unitàriament que una simple comprovació de string — mitigat
  amb un test d'integració dedicat contra un servidor de test
  controlable, **incloent-hi explícitament un cas de redirecció al
  mateix host i un cas d'adreça `::ffff:`-mapejada** (vegeu nota de
  cobertura més avall i forward-note per a `task-planner`/`qa-engineer`
  a §7) en lloc de només tests unitaris. (−) la denylist de noms
  d'amfitrió (F-03/F-04) és una llista mantinguda a mà — s'accepta
  perquè la topologia d'un sol VPS canvia poc i el cost d'oblidar
  afegir-hi un nom nou és baix comparat amb l'alternativa (tractar tota
  resolució Docker DNS com a sospitosa, molt més intrusiu).
- **Alternatives considered:** validar només la URL d'entrada (host
  literal/IP en el text) sense validar la resolució DNS (rebutjat:
  EC-07 exigeix explícitament cobrir el cas de resolució, no només
  aparença textual); usar `CheckRedirect` com a **únic** punt de
  validació, re-parsejant manualment cada `Location` i resolent DNS
  manualment allà (rebutjat: duplicaria la resolució que
  `ControlContext` ja fa en el flux normal de connexió, reobrint el
  parell TOCTOU que F-06 identifica — `CheckRedirect` es manté només
  com a capa addicional barata per a esquema, no com a substitut de la
  validació d'IP); mantenir la crida separada a
  `net.DefaultResolver.LookupIPAddr` en paral·lel a `Control`/
  `ControlContext` (rebutjat explícitament — F-06: exactament el parell
  TOCTOU que permet discrepància entre les dues resolucions DNS);
  enumeració de rangs prohibits en lloc d'allowlist (rebutjat després de
  F-07 — necessàriament incompleta davant blocs especials no anticipats,
  mentre que `IsGlobalUnicast()` els exclou tots per construcció); tractar
  tota resolució via el resolver DNS embegut de Docker com a sospitosa
  per defecte en lloc d'una denylist de noms explícita (rebutjat per a
  F-04 — més intrusiu i innecessari per a la topologia actual d'un sol
  VPS; documentat com a revisió futura si `traefik-public` creix); proxy
  de sortida dedicat / servei extern de "URL unfurling" (rebutjat: infra
  nova per a un VPS únic, cost desproporcionat per a un ítem d'un sol
  ítem de backlog); allowlist de dominis coneguts per a l'URL d'entrada
  de l'usuari (rebutjat: contradiu el propòsit del producte — l'usuari
  ha de poder enganxar qualsevol enllaç, no només d'una llista tancada;
  no confondre amb l'allowlist de **classificació d'IP** de F-07, que
  és un mecanisme diferent i sí adoptat).

### ADR-03 — Scraping asíncron: la petició `POST` respon abans que el scraping s'acabi

- **Status:** accepted
- **Context:** `requirements.md` NFR-07 exigeix un timeout dur (5s, ADR-02)
  per al scraping, i `proposal.md`/`requirements.md` ja tracten el
  fallback com a resultat normal i freqüent (Instagram i similars). Calia
  decidir si `POST /api/v1/ideas` espera el resultat del scraping abans
  de respondre (síncron, més simple, però l'usuari espera fins a 5s per
  petició, potencialment percebut com a "penjat") o si respon
  immediatament amb un estat pendent i resol la previsualització en
  segon pla (async, l'usuari continua usant l'app de seguida).
  `proposal.md` §8.2 Estat D ja especifica visualment una targeta
  "Recuperant…" que apareix immediatament després d'enviar el formulari,
  amb l'input buidant-se a l'instant i el focus tornant-hi — aquest
  comportament **només és cert si el servidor ja ha respost** abans que
  el scraping s'acabi. Un disseny síncron faria mentir aquesta
  especificació visual (l'input no es podria buidar fins a 5s després).
- **Decision:** **asíncron, amb un límit explícit de concurrència
  (esmenat — F-05).** `POST /api/v1/ideas` valida l'URL (format +
  esquema, NFR-05 — validació barata, sense xarxa), insereix la fila amb
  `preview_status = 'pending'` i respon `201` immediatament amb la idea
  en aquest estat. El `Service` **no** llança un `goroutine` sense
  restricció per cada `POST` — encua el treball de scraping en un
  **worker pool acotat** (semàfor amb buffer, `cap = 4` a `8` goroutines
  concurrents de scraping, valor exacte a confirmar per
  `fullstack-developer`/`platform-engineer` en funció de la memòria real
  observada per scrape). Cada worker crida `fetchsafe.FetchPreview` (amb
  el seu propi `context.WithTimeout` de 5s, independent del context de
  la petició HTTP, que ja s'haurà tancat en respondre) i, en resoldre's
  (èxit, parcial, o fallada), fa un `UPDATE` de `activity_ideas` amb el
  resultat i `preview_status` final (`ready` | `partial` | `failed`). Si
  el pool ja té totes les places ocupades, la nova idea espera en cua
  fins que se n'alliberi una — mai es rebutja el `POST` per aquest motiu
  (la fila ja s'ha inserit `pending` abans que el treball s'encoli; un
  `pending` que triga una mica més a resoldre's és un resultat visual ja
  cobert per l'Estat D, no un error nou). El client descobreix la
  resolució al següent `GET /api/v1/ideas` (sondeig existent, ~10s, o
  refetch-on-focus) — **no** WebSocket ni SSE nou (coherent amb
  `PLAN.md` §2.6, ja rebutjat per a la resta de Niu).
- **Consequences:** (+) coherent amb l'estat visual "Recuperant…" ja
  aprovat a Stage 1.5 — cap contradicció entre `design.md` i
  `proposal.md`; (+) l'usuari no espera mai el servidor per continuar
  interactuant amb l'app (EC-16, múltiples idees en curs simultàniament,
  ja anticipat a `proposal.md` §8.2 Estat D); (+) un servidor extern lent
  o hostil mai bloqueja el fil de la petició HTTP entrant — els workers
  de scraping són independents del cicle de vida de la petició; (+) el
  límit de concurrència (F-05) acota l'ús de memòria pic del contenidor
  — amb el límit de 128M/0.5CPU declarat a `compose.yaml` i ~2MiB
  retinguts per scrape en curs (ADR-02, límit de 2MiB en streaming), un
  `goroutine`-per-`POST` sense cap era una via directa a OOM sota
  ràfegues d'afegits simultanis; un pool de 4–8 workers acota aquest
  cost a un múltiple petit i previsible de 2MiB, independentment de
  quantes idees s'afegeixin de cop. (−) el pool sobreviu la petició HTTP
  original — cal que `main.go` proporcioni al `Service` un `context` de
  fons propi (no el de la petició, que es cancel·la en respondre) i que
  el pool (o el semàfor que l'implementa) es tanqui ordenadament a
  l'aturada de l'app; documentat aquí perquè `fullstack-developer` no ho
  passi per alt. (−) si el procés es reinicia mentre una idea és
  `pending` (EC vora), la idea queda `pending` per sempre — mitigat a
  §8 (Resiliència) i R-02: acceptable en v1 perquè el propietari pot
  eliminar i tornar a afegir; no es construeix cap cua de reprocessament
  (`requirements.md` §7, ja fora d'abast explícitament).
- **Alternatives considered:** síncron amb timeout de 5s dins de la
  petició `POST` (rebutjat: contradiu directament l'estat "Recuperant…"
  de `proposal.md` §8.2 Estat D, que assumeix que l'input es buida a
  l'instant; també deixaria el fil HTTP ocupat fins a 5s per petició,
  amb dues persones podent afegir idees alhora); cua de tasques
  persistent (p. ex. una taula `scrape_jobs` amb un worker separat)
  (rebutjat: sobre-enginyeria per a un sol usuari afegint idees
  esporàdicament — un pool de workers en memòria n'hi ha prou, i si el
  procés cau a mig scraping, l'única conseqüència és una idea que es
  queda `pending`, ja acceptat a R-02); `goroutine`-per-`POST` sense cap
  de concurrència (rebutjat — F-05: sota el límit de 128M/0.5CPU del
  contenidor, una ràfega d'afegits simultanis sense límit de concurrència
  podria exhaurir la memòria disponible; el cost d'un worker pool acotat
  és mínim i elimina aquest risc per construcció).

### ADR-04 — Parsing Open Graph amb `golang.org/x/net/html`, sense llibreria de scraping de tercers

- **Status:** accepted
- **Context:** calia triar com parsejar les etiquetes `<meta
  property="og:title|og:image|og:description">` del HTML recuperat.
  Opcions: una llibreria de tercers especialitzada en Open Graph (n'hi ha
  diverses a l'ecosistema Go), un parser HTML complet + selecció manual
  de nodes, o una cerca de patró (regex) sobre el text cru.
- **Decision:** `golang.org/x/net/html` (el tokenizer/parser HTML oficial
  de l'ecosistema `golang.org/x`, no una dependència de tercers no
  mantinguda pel mateix grup que ja proveeix `golang.org/x/text` al
  `go.mod`). `fetchsafe` (o un sub-paquet `fetchsafe/ogparse` si convé
  separar-ho) tokenitza el flux HTML **limitat pel `LimitReader` de 2MiB
  (ADR-02)**, s'atura en trobar el tancament de `<head>` (les etiquetes
  OG sempre hi viuen — no cal continuar llegint `<body>`, que és on
  viuria la major part del pes d'una pàgina real), i extreu els valors
  dels `<meta>` amb `property` començant per `og:`.
- **Consequences:** (+) cap dependència nova de tercers a auditar/
  mantenir — `golang.org/x/net` és del mateix paraigua que
  `golang.org/x/text`, ja present i confiat al `go.mod`; (+) aturar-se a
  `</head>` evita llegir/parsejar el `<body>` sencer d'una pàgina real
  (normalment molt més pesat que el `<head>`), reduint el cost de CPU
  independentment del límit de mida en bytes; (+) un tokenizer HTML
  real (enfront de regex) no es trenca amb HTML malformat o amb atributs
  en ordre inesperat (EC-05, metadades absents/malformades es tracten
  com a "no trobades", no com un crash de parsing). (−) un pèl més de
  codi que una regex ingènua — acceptat: una regex sobre HTML arbitrari
  d'internet és fràgil i, empíricament, una font recurrent de bugs de
  parsing.
- **Alternatives considered:** llibreria de tercers d'Open Graph
  (rebutjat: dependència externa no necessària per un parsing tan
  acotat — `proposal.md` no exigeix cap format més enllà d'OG bàsic);
  regex sobre el text cru (rebutjat: fràgil davant HTML real,
  especialment amb atributs en ordre variable o comentaris intercalats);
  parsejar el document HTML sencer sense aturar-se a `</head>`
  (rebutjat: cost de CPU innecessari sobre pàgines reals, sense cap
  benefici — les OG tags mai viuen a `<body>`).

### ADR-05 — Diferenciació visual: tercera pestanya de navegació + token `color.mel`

- **Status:** accepted
- **Context:** AC-07 exigeix que l'espai es distingeixi clarament,
  només pel visual, de "Compra" (NIU-1) i "Projectes" (NIU-5).
  `proposal.md` §8.0/§9.1 ja resol aquesta decisió a la porta humana de
  Stage 1.5: nou token `--color-mel` (`#C99A3A`) aprovat, graella de
  targetes (enfront de llista dual o llista amb badge) com a tercera
  disposició, i la barra de navegació de tres entrades materialitzada
  per primer cop (NIU-5 l'havia deixat com a judici obert).
- **Decision:** aquest `design.md` **no redefineix** cap valor visual —
  confirma que `internal/httpapi`/`web/` implementen exactament
  `proposal.md` §8.1 (barra de navegació de tres entrades, subratllat
  `color.mel-hover` per a "Idees" actiu) i §8.2 (graella
  `auto-fill, minmax(240px, 1fr)`, `IdeaCard` amb els quatre estats A–D).
  L'única decisió tècnica pròpia d'aquest document és **on** viu la
  ruta/vista nova: com a tercera secció dins del mateix `index.html`
  (mateix patró que NIU-5 §7 va deixar obert i que aquest ítem
  materialitza), reutilitzant `web/js/store.js`/`api.js`/`a11y.js`
  existents amb wrappers nous (`getIdeas`, `addIdea`, `deleteIdea`), cap
  component nou al sistema de disseny.
- **Consequences:** (+) cap ambigüitat visual pendent — Stage 1.5 ja ho
  va resoldre amb detall pixel-level, aquest document només en confirma
  el contracte tècnic (API, estats de dades) que ho fa possible; (+)
  reutilització total de la infraestructura de sondeig/accessibilitat ja
  existent. (−) cap — decisió íntegrament heretada, sense marge
  d'interpretació pendent (a diferència d'NIU-5 ADR-04, que sí que en
  deixava).
- **Alternatives considered:** cap — `proposal.md` §9.1 ja va tancar
  aquesta decisió a la porta humana abans d'arribar a Stage 2.

## 4. Building blocks (arc42 §5 + C4 Nivell 2)

> Només els components que aquest canvi toca. `internal/items`,
> `internal/projects`, `internal/auth`, `internal/config` i
> `internal/store` (parts existents) no canvien de forma i no es
> repeteixen aquí.

```text
┌───────────────────────────────────────────────────────────────────┐
│                          cmd/niu/main.go                           │
│  afegeix wiring d'ideas.Service + IdeasRepository + fetchsafe      │
│  .Client; proporciona un context.Background() propi i el worker    │
│  pool acotat (4-8, ADR-03 esmenat F-05) per al scraping; cap       │
│  canvi al cablejat existent                                        │
└───────────────────────────────────────────────────────────────────┘
     │                    │                    │              │
     ▼                    ▼                    ▼              ▼
┌────────────┐     ┌─────────────┐     ┌───────────────┐  ┌───────────┐
│internal/   │     │internal/    │     │internal/       │  │internal/  │
│ideas (NOU) │────▶│fetchsafe    │     │httpapi         │  │store      │
│Service,    │     │(NOU)        │     │(afegit, no     │  │(afegit,   │
│Repository, │     │única porta  │     │modificat)      │  │no modif.  │
│valida URL, │     │d'entrada a   │     │ideas_handlers  │  │per a      │
│encua al    │     │peticions     │     │.go (nou),      │  │items/     │
│worker pool │     │sortides.     │     │rutes noves     │  │projects)  │
│acotat      │     │Detall complet│     │sota grup ja    │  │Ideas      │
│(ADR-03),   │     │a ADR-02:     │     │existent         │  │Repository │
│og-parse    │     │esquema, host │     │WithCurrentUser  │  │implementa │
│via         │     │denylist      │     │+ RequireCSRF    │  │ideas.     │
│x/net/html  │     │(F-03/F-04),  │     │(NIU-4)          │  │Repository │
│(ADR-04)    │     │IP allowlist  │     └────────────────┘  │+ EventSink│
└─────┬──────┘     │post-Unmap    │                          └─────┬─────┘
      │            │(F-02/F-07),  │                                │
      │            │CheckRedirect │                                │
      │            │+ no keep-    │                                │
      │            │alive (F-01), │                                │
      │            │timeout 5s,   │                                │
      │            │cap 2MiB,     │                                │
      │            │0 credencials │                                │
      │            └──────┬───────┘                                ▼
      │                   │                                 ┌─────────────┐
      │                   ▼                                 │ SQLite:      │
      │            ┌──────────────┐                         │ activity_    │
      │            │  Internet    │                         │ ideas +      │
      │            │  (destí de   │                         │ events       │
      │            │  l'usuari)   │                         │ (esquema nou │
      │            └──────────────┘                         │  + existent) │
      └───────────────────────────────────────────────────────▶┘
```

- **`internal/ideas` (domini, nou)** — responsabilitat: validació de
  format d'URL (EC-01/EC-10), orquestració del cicle
  desar-immediatament + resoldre-en-segon-pla via el worker pool acotat
  (ADR-03, esmenat F-05), invocació de `fetchsafe.FetchPreview`,
  invocació del parser Open Graph (ADR-04). Defineix `Repository`
  (`Create`, `Get`, `List`, `UpdatePreview`, `Delete`) i `EventSink`
  (mateixa forma mínima que `items`/`projects`).
  **No** importa `net/http` directament per a peticions sortints — només
  crida `fetchsafe.FetchPreview`, que sí que ho fa. No reutilitza
  `items.NormalizeName` (a diferència d'NIU-5) — EC-06 ja confirma que
  no hi ha deduplicació en aquest espai.
- **`internal/fetchsafe` (nou, aïllat)** — **única porta d'entrada** a
  qualsevol petició HTTP sortint cap a un destí controlat per l'usuari
  (ADR-02). No coneix res del domini `ideas` (no sap què és una "idea",
  no importa `internal/ideas`) — rep una URL, retorna una `Preview{
  Title, ImageURL, Description string; Partial bool}` o un error
  tipat (`ErrSchemeRejected`, `ErrDestinationForbidden`, `ErrTimeout`,
  `ErrResponseTooLarge`, `ErrUnsupportedContentType`). Aquest aïllament
  és intencionat: cap altre paquet pot "oblidar-se" d'aplicar la
  mitigació perquè cap altre paquet fa la petició.
- **`internal/store`** — s'estén amb `IdeasRepository`, paral·lel a
  `ItemsRepository`/`ProjectsRepository`: `Create` insereix amb
  `preview_status='pending'`, `UpdatePreview` és l'únic `UPDATE`
  permès un cop creada la fila (mai toca `url`/`added_by`/`created_at`).
- **`internal/httpapi`** — afegeix `ideas_handlers.go`
  (`handleListIdeas`, `handleCreateIdea`, `handleDeleteIdea`) i `dto.go`
  s'estén amb `ideaDTO`. **Modifica `router.go`** només per registrar
  `/api/v1/ideas` dins del grup `/api/v1` ja existent. Cap altre handler
  es toca.
- **`web/` (nous fitxers)** — tercera secció de navegació (ADR-05),
  `IdeaCard` amb els 4 estats de `proposal.md` §8.2, `AddIdeaInput`
  (§8.3), reutilitzant `api.js`/`a11y.js`/`store.js` existents amb
  wrappers nous.

## 5. Vista d'execució (arc42 §6)

**Flux 1 — Afegir una idea, scraping asíncron (AC-01/AC-02/AC-03,
EC-01–EC-05, EC-08–EC-10, ADR-03):**

1. Client envia `POST /api/v1/ideas {"url": "https://..."}`.
2. `httpapi` decodifica el body, crida `ideas.Service.Add(ctx, userID,
   rawURL)`.
3. `Service` valida format (URL sintàcticament vàlida, EC-10 buit
   rebutjat) i **esquema únicament** (`http`/`https`, NFR-05/EC-01) —
   **sense** cap petició de xarxa encara. Si l'esquema no és vàlid:
   `ErrSchemeRejected`, `httpapi` respon `400 validation_failed`, **cap
   fila es crea**.
4. Si el format és vàlid: `Repository.Create` insereix la fila amb
   `preview_status = 'pending'`, `title/image_url/description = NULL`.
   `Service` escriu un event `idea_added`. `httpapi` respon
   **immediatament** `201` amb la idea en estat `pending`.
5. `Service.Add` encua el treball al worker pool acotat (4-8 workers,
   ADR-03 esmenat F-05; amb el `context.Background()` de fons
   proporcionat per `main.go`, no el de la petició HTTP, ja tancada).
   Quan un worker el recull, crida `fetchsafe.FetchPreview(bgCtx,
   rawURL)`.
6. **Dins de `fetchsafe.FetchPreview`** (ADR-02, detall complet allà):
   comprova primer la denylist de noms d'amfitrió (F-03/F-04, sense cap
   petició de xarxa si coincideix), després resol l'IP i la valida amb
   el criteri d'allowlist post-`Unmap()` (F-02/F-07), connecta amb
   timeout de 5s (keep-alives desactivats, F-01), llegeix com a màxim
   2MiB en streaming, comprova `Content-Type` (EC-09: si no és
   `text/html`-compatible, es tracta com a fallback sense ni intentar
   el parsing OG).
7. **Resolució:**
   - Èxit complet (AC-01): `title`, `image_url`, `description` tots
     presents → `Repository.UpdatePreview` amb
     `preview_status='ready'`.
   - Parcial (AC-03): alguns camps `NULL` (EC-05, metadades absents) →
     `preview_status='partial'`, els camps absents queden `NULL`.
   - Qualsevol error de `fetchsafe` (esquema ja descartat al pas 3;
     destí prohibit EC-02/EC-04/EC-07; timeout EC-08; resposta massiva
     EC-03; contingut no-HTML EC-09; sense metadades EC-05) →
     `preview_status='failed'`, tots els camps de previsualització
     `NULL`. **Mai** es reintenta automàticament (fora d'abast,
     `requirements.md` §7).
8. `Service` escriu un event `idea_preview_resolved` amb
   `{"idea_id": ..., "status": "ready"|"partial"|"failed"}`.
9. El client descobreix la resolució al proper `GET /api/v1/ideas`
   (sondeig ~10s o refetch-on-focus, AC-06) — la targeta "Recuperant…"
   (`proposal.md` §8.2 Estat D) es substitueix per l'Estat A/B/C
   corresponent.

**Flux 2 — Llegir la llista, mai re-scraping (AC-04/AC-06, NFR-09):**

1. Client envia `GET /api/v1/ideas`.
2. `Service.List` crida `Repository.List` — una única consulta SQLite
   amb `JOIN` a `users` (`added_by`), **sense cap petició de xarxa
   sortint**, independentment de quantes vegades es cridi (NFR-09).
3. Respon `200` amb totes les idees, cadascuna amb el seu
   `preview_status` actual (`pending`/`ready`/`partial`/`failed`) i els
   camps de previsualització tal com estan persistits — mai recalculats.

**Flux 3 — Eliminar una idea (AC-05, EC-15):**

1. Client envia `DELETE /api/v1/ideas/{id}`.
2. `Service.Delete` crida `Repository.Delete` (`DELETE FROM
   activity_ideas WHERE id = ?`, idempotent: `existed bool`, mateix
   patró que `items.Delete`/`projects.Delete`).
3. Si `existed`: escriu event `idea_deleted`.
4. Respon `204` sempre (EC-15, idempotent). **Nota:** si el worker del
   pool (ADR-03) que processa el scraping del Flux 1 encara està en
   curs quan arriba el `DELETE`, es deixa completar igualment (no hi ha
   cancel·lació explícita — simplement el seu `UpdatePreview` final
   trobarà `0` files afectades perquè l'`id` ja no existeix, i s'ignora
   silenciosament; documentat a
   §8 Resiliència).

## 6. Contractes i model de dades

### 6.1 API

Tots els endpoints sota `/api/v1/ideas`, dins del mateix grup `/api/v1`
ja protegit per `WithCurrentUser` (NIU-4); les mutacions reutilitzen
`RequireCSRF` exactament com `/api/v1/items` i `/api/v1/projects`.

| Endpoint | Mètode | Petició (alt nivell) | Resposta (alt nivell) |
| --- | --- | --- | --- |
| `/api/v1/ideas` | `GET` | — | `200 { "ideas": [Idea...] }` |
| `/api/v1/ideas` | `POST` | `{ "url": string }` | `201 Idea` (estat `pending`) \| `400 validation_failed` |
| `/api/v1/ideas/{id}` | `DELETE` | — | `204` (idempotent, EC-15) |

**`Idea` (forma de resposta):**

```json
{
  "id": 12,
  "url": "https://example.com/activitat",
  "title": "Un restaurant fantàstic",
  "image_url": "https://example.com/img.jpg",
  "description": "La millor paella de la ciutat.",
  "preview_status": "ready",
  "added_by": { "id": 1, "display_name": "Usuari A", "avatar_emoji": "🐦" },
  "created_at": "2026-08-03T10:00:00Z"
}
```

`preview_status` ∈ `pending | ready | partial | failed`. `title`,
`image_url`, `description` són `null` si `preview_status` és `pending` o
`failed`, o si el camp concret no es va poder recuperar sota `partial`
(AC-03). `url` és **sempre** present, independentment de l'estat
(AC-02: l'enllaç és l'única via d'identificar la idea en fallback).

**Envelope d'error (reutilitza `apiError` existent, cap tipus nou):**

```json
{ "error": { "code": "validation_failed", "message": "Aquest enllaç no és vàlid — ha de començar per http:// o https://." } }
```

Cap codi d'error nou distingeix "destí prohibit" (EC-02) de qualsevol
altre motiu de fallback — coherent amb NFR-06 ("missatge clar que no
revela detalls interns de xarxa"): un destí de xarxa privada mai arriba
a generar un error `400` diferenciat perquè, per disseny (ADR-03), la
petició `POST` **ja ha respost `201`** abans que `fetchsafe` avaluï el
destí — el rebuig de xarxa es manifesta únicament com
`preview_status='failed'`, indistingible d'un timeout o d'una pàgina
sense metadades. Això és una conseqüència directa i desitjada de l'ADR-03
asíncron: no hi ha manera d'informar l'usuari en temps de petició que el
destí era prohibit sense revelar-li per què (NFR-06), i el disseny visual
ja tracta "sense previsualització" com un resultat únic i normal
(`proposal.md` §8.2 Estat B), no com un ventall de motius d'error.

**Mai `GET` amb efecte (EC-13/NFR-03):** mateixa comprovació estàtica que
NIU-1/NIU-5, estesa per cobrir també `/api/v1/ideas/*`.

**Autenticació (EC-14/NFR-04):** cap ruta nova és pública — totes viuen
dins del `r.Route("/api/v1", ...)` que ja aplica `WithCurrentUser`.

### 6.2 Model de dades (deltes sobre l'esquema existent)

| Entitat | Canvi | Risc de migració |
| --- | --- | --- |
| `activity_ideas` | **Taula nova.** Vegeu migració més avall. | LOW — taula nova, sense dades prèvies |
| `events` | **Cap canvi d'esquema.** S'escriu des del dia 1 (`idea_added`, `idea_preview_resolved`, `idea_deleted`). | LOW |
| `items`, `projects`, `users`, `sessions` | **Cap canvi.** | — |

**Migració goose (`app/migrations/00X_activity_ideas.sql`, número
següent al de la migració `projects` d'NIU-5):**

```sql
-- +goose Up
CREATE TABLE activity_ideas (
  id              INTEGER PRIMARY KEY,
  url             TEXT NOT NULL,
  title           TEXT,
  image_url       TEXT,
  description     TEXT,
  preview_status  TEXT NOT NULL CHECK (preview_status IN ('pending','ready','partial','failed')) DEFAULT 'pending',
  added_by        INTEGER REFERENCES users(id),
  created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- EC-06: cap índex únic sobre url — deduplicació explícitament fora
-- d'abast; dues files amb el mateix enllaç són vàlides.

-- +goose Down
DROP TABLE activity_ideas;
```

**Notes de la migració:**

- `preview_status` amb `CHECK` + `DEFAULT 'pending'` reflecteix ADR-03
  (tota fila neix `pending`) i tanca la porta a un cinquè valor
  accidental.
- `title`/`image_url`/`description` són `TEXT NULL` sense `CHECK` de
  longitud — són contingut extern no fiable (EC-11/EC-12), es tracten
  íntegrament com a text opac a la capa d'aplicació (`textContent` al
  frontend, paràmetres vinculats a SQL, mai concatenació).
- `url` **no** té índex únic (EC-06, confirmat sense deduplicació) ni
  `CHECK` de longitud — `requirements.md` no en defineix cap límit
  explícit (`proposal.md` §8.3, "Format").
- Cap columna de "qui ha resolt la previsualització" ni "quan" — no és
  necessària per cap AC; `updated_at` no existeix en aquesta taula
  (a diferència de `projects`) perquè no hi ha cap AC que en depengui.
- No cal cap canvi a cap migració anterior.

## 7. Concerns transversals (arc42 §8)

- **Seguretat:** aquest és el canvi de seguretat més profund de Niu fins
  ara — primera vegada que el servidor inicia peticions de xarxa
  sortints cap a destins controlats per l'usuari. Mitigat íntegrament
  per `internal/fetchsafe` (ADR-02, **esmenat després de revisió de
  seguretat de disseny — vegeu nota de cobertura de tests més avall**):
  esquema `http(s)` únicament, denylist de noms d'amfitrió independent
  de la validació d'IP (F-03/F-04), validació d'IP amb criteri
  d'allowlist post-`Unmap()` (F-02/F-07) pre-connexió i a cada
  redirecció amb `DisableKeepAlives` + revalidació explícita a
  `CheckRedirect` (F-01), timeout 5s, cap 2MiB en streaming, client
  sense credencials. `textContent`/CSP sense `unsafe-inline` per a
  `title`/`description`/`url` recuperats (EC-11/NFR-01), paràmetres
  vinculats a SQL (EC-12/NFR-02), cap ruta `GET` amb efecte
  (EC-13/NFR-03), cap endpoint fora del middleware d'auth
  (EC-14/NFR-04).
  **Nota de cobertura de tests (forward-note per a `task-planner` i
  `qa-engineer` — no redefineix `requirements.md`, l'estén a nivell de
  disseny):** el pla de tests actual per a EC-04 (redirecció cap a un
  destí prohibit) ha de garantir explícitament, com a mínim, dos casos
  que la revisió de seguretat ha identificat com a no coberts per un
  simple test de redirecció cross-host: (1) **redirecció al mateix
  host** (`Location:` apuntant al mateix `host:port` que la petició
  original, amb el servidor de test responent des d'una IP prohibida en
  el segon salt) — necessari per detectar una regressió de F-01 (connexió
  reutilitzada saltant-se `ControlContext`); (2) **destí en forma
  IPv4-mapejada-a-IPv6** (`::ffff:127.0.0.1` o `::ffff:169.254.169.254`)
  — necessari per detectar una regressió de F-02 (`Unmap()` omès o
  eliminat per accident). Sense aquests dos casos, la matriu EC-04
  existent pot passar en verd contra una implementació que reintrodueixi
  qualsevol dels dos forats.
- **Observabilitat:** fora d'abast explícit (NIU-3 encara no desplegat).
  `internal/fetchsafe` registra via `log/slog` cada rebuig (esquema no
  vàlid, destí prohibit, timeout, mida excedida) amb l'`idea_id`
  associat, **mai** l'IP resolta concreta a nivell `info` (evitar soroll
  de log amb intents repetits) però sí a nivell `debug` per a
  diagnòstic. Cap dada de xarxa interna de Niu (topologia del VPS) es
  filtra mai al client (NFR-06).
- **Rendiment:** `GET /api/v1/ideas` és una única consulta amb `JOIN`,
  sense N+1, mateix patró que `ItemsRepository.List`/
  `ProjectsRepository.List`. El scraping asíncron (ADR-03) garanteix que
  cap petició `POST` triga més que el temps de l'`INSERT` — el cost del
  scraping (fins a 5s) mai es reflecteix en la latència percebuda per
  l'usuari que l'ha iniciat. El worker pool acotat (4-8, ADR-03 esmenat
  F-05) posa un límit explícit al consum de memòria pic sota ràfegues
  d'afegits simultanis, coherent amb el límit de 128M/0.5CPU del
  contenidor (`compose.yaml`).
- **Resiliència:** el scraping (ADR-03) és "fire-and-forget" respecte al
  cicle de vida de la petició HTTP, executat per un worker pool acotat
  en lloc d'un `goroutine` sense límit per `POST` (esmenat — F-05) —
  documentat com a decisió acceptada, no com un descuit (R-02). Si el
  procés es reinicia amb idees `pending`, es queden `pending`
  indefinidament (sense cua de reprocessament, fora d'abast); l'usuari
  pot eliminar i tornar a afegir com a solució manual. `DELETE`
  concurrent amb una resolució de scraping en curs (Flux 3, pas 4) és
  segur: el `WHERE id = ?` de l'`UPDATE` final simplement no troba la
  fila i no fa res.
- **Compliance i privacitat:** S11 — cap dada personal a fixtures/
  migracions. Contingut recuperat d'una pàgina externa (títol,
  descripció, URL de la imatge) es tracta com a dada no fiable, mai com
  a codi executable, i mai s'envia cap credencial de Niu a l'exterior
  (NFR-08) — rellevant també des del punt de vista de privadesa: la
  identitat de Niu (sessió, usuari) mai és visible per al servidor de
  destí de la previsualització.
- **Accessibilitat:** WCAG 2.2 AA, coherent amb NIU-1/NIU-5. Detall
  complet ja fixat a `proposal.md` §8.6 (Stage 1.5) — `alt=""` a les
  imatges OG (decoratives, confirmat a la porta humana §9.1),
  `aria-live="polite"` anunciant "Desant idea, recuperant
  previsualització…" i la resolució final. Aquest document no en
  redefineix res, només confirma que el contracte de dades
  (`preview_status`) proporciona prou informació perquè el frontend
  generi aquests anuncis sense inventar-se estats.
- **i18n/l10n:** cap canvi — UI en català fix, mateix que la resta de
  Niu.

## 8. Riscos (arc42 §11)

| ID | Risc | Severitat | Mitigació | Owner |
| --- | --- | --- | --- | --- |
| R-01 | Una implementació futura d'un client HTTP en un altre punt del codi (p. ex. per a un ús no relacionat) podria oblidar aplicar la mitigació de `fetchsafe` si algú fa una petició externa sense passar per aquest paquet | MED | `internal/fetchsafe` és l'únic punt permès de peticions HTTP cap a destins d'usuari (ADR-02); `code-reviewer`/`security-engineer` verifiquen a `/audit` que cap altra crida `http.Get`/`http.Client.Do` amb una URL d'usuari existeixi fora d'aquest paquet | `code-reviewer` |
| R-02 | El scraping asíncron (ADR-03) deixa idees `pending` per sempre si el procés es reinicia a mig scraping | LOW | Acceptat explícitament — sense cua de reprocessament en v1 (`requirements.md` §7, fora d'abast); l'usuari pot eliminar i re-afegir. Documentat perquè no se sobreentengui com un bug pendent de resoldre | `software-architect` (acceptat) |
| R-03 | **RESOLT (2026-08-03, revisió de seguretat).** La versió original d'aquest disseny confiava únicament en `net.InterfaceAddrs()` + rangs RFC1918 per detectar "la mateixa instància de Niu", i no cobria `niu.fikua.com` (resol a una IP **pública** de l'edge de Cloudflare — verificat contra `app/compose.yaml`) ni els veïns de la xarxa Docker `traefik-public` (F-03/F-04) | ~~MED~~ **LOW residual** | ADR-02 (esmenat) afegeix una denylist explícita de noms d'amfitrió (`niu.fikua.com` + `NIU_PUBLIC_HOST` + noms de servei coneguts de `traefik-public`), independent i anterior a la validació d'IP — mecanisme separat perquè cap validació de rang pot cobrir un edge públic que fa proxy cap a la mateixa app. Risc residual LOW: la denylist és una llista mantinguda a mà, revisar-la si `traefik-public` creix (nota a ADR-02 Consequences) | `software-architect` (mitigat en aquest document); `platform-engineer` (mantenir la llista en créixer l'stack) |
| R-04 | `golang.org/x/net/html` (ADR-04) és una dependència nova, encara que del mateix paraigua `golang.org/x`; introdueix superfície de parsing sobre contingut no fiable (HTML d'internet) | LOW | Tokenizer stdlib-adjacent, àmpliament usat i auditat a l'ecosistema Go; el `LimitReader` de 2MiB (ADR-02) limita l'entrada que arriba mai al parser, reduint la superfície d'un possible bug de parsing a un input acotat | `software-architect` (acceptat) |
| R-05 | Sense maqueta pixel-perfect pròpia d'aquest `design.md` (ja coberta per `proposal.md` §8, Stage 1.5 no omesa per a NIU-6) — risc mínim de desviació entre aquest contracte de dades i el disseny visual si `preview_status` no cobreix algun estat visual assumit | LOW | Verificat explícitament a §6.1/§7: els 4 estats visuals de `proposal.md` §8.2 (A/B/C/D) mapegen exactament als 4 valors de `preview_status` (`ready`→A, `failed`→B, `partial`→C, `pending`→D) — cap estat visual orfe de dades | `software-architect` (verificat en aquest document) |
| R-06 | **NOU (revisió de seguretat, F-01).** El disseny original d'ADR-02 afirmava que la re-validació a cada redirecció era "estructural" via `Dialer`/`Control` únicament; `security-engineer` ha verificat empíricament que això és fals sota connexions reutilitzades (`http.Transport` amb keep-alives actius salta `Control` en una cadena de redireccions al mateix host — 4 salts → 1 sola invocació de `Control`) | HIGH (abans de la mitigació) → **RESOLT** | `DisableKeepAlives: true` al `Transport` dedicat de `fetchsafe` + revalidació explícita d'esquema dins de `CheckRedirect` com a defensa en profunditat (ADR-02, esmenat). Requereix un test d'integració amb redirecció al mateix host per evitar regressió (vegeu forward-note a §7) | `software-architect` (mitigat en aquest document); `qa-engineer`/`task-planner` (cobertura de test obligatòria) |
| R-07 | **NOU (revisió de seguretat, F-02).** Adreces IPv4 mapejades a IPv6 (`::ffff:127.0.0.1`, `::ffff:169.254.169.254`) passaven totes les comprovacions de classificació d'IP sense un `Unmap()` previ | HIGH (abans de la mitigació) → **RESOLT** | `Unmap()` obligatori abans de qualsevol crida `Is*()` (ADR-02, esmenat), combinat amb el nou criteri d'allowlist (F-07). Requereix un test d'integració amb destí en forma `::ffff:` per evitar regressió (vegeu forward-note a §7) | `software-architect` (mitigat en aquest document); `qa-engineer`/`task-planner` (cobertura de test obligatòria) |
| R-08 | **NOU (revisió de seguretat, F-05).** Un `goroutine`-per-`POST` sense límit de concurrència podria exhaurir la memòria del contenidor (128M/0.5CPU, `compose.yaml`) sota una ràfega d'afegits simultanis, a ~2MiB retinguts per scrape en curs | MED (abans de la mitigació) → **LOW residual** | Worker pool acotat (4-8 workers, ADR-03 esmenat). Risc residual LOW: el valor exacte del cap (4 vs. 8) es confirma empíricament durant `/code`, no és una decisió tancada en aquest document | `fullstack-developer` (confirmar el valor del cap durant la implementació) |

## 9. Preguntes obertes — RESOLTES a la porta humana (2026-08-03)

- [x] **Llindars concrets de `fetchsafe`:** confirmats — timeout 5s,
  cap de resposta 2MiB, màxim 5 redireccions. Sense canvis a ADR-02.
- [x] **Nom del paquet:** confirmat — `internal/fetchsafe`.
- [x] **R-03 (rang de xarxa Docker del VPS real):** RESOLT per aquesta
  esmena — l'enfoc original (confiar en rangs RFC1918 per detectar "la
  mateixa instància") ha estat substituït per una denylist explícita de
  noms d'amfitrió (ADR-02, F-03/F-04), que no depèn de cap assumpció
  sobre el rang de xarxa Docker. Ja no bloqueja `/audit` per aquest
  motiu; el residual (mantenir la llista de noms si `traefik-public`
  creix) es tracta a R-03 §8, no com a pregunta oberta.
- [ ] **Cap final de concurrència del worker pool (F-05, ADR-03):**
  aquest document fixa el rang 4-8 però no un valor únic — `tasks.md`
  ha d'incloure una tasca explícita perquè `fullstack-developer`
  confirmi el valor concret durant `/code` (R-08 §8), no bloquejant per
  a `/audit` però ha de quedar reflectit al codi i no deixar-se com un
  `TODO` sense resoldre.
- [ ] **Cobertura de tests EC-04 (redirecció mateix host + `::ffff:`
  mapejat):** aquest document deixa una forward-note explícita a §7 —
  `task-planner`/`qa-engineer` han d'assegurar que la matriu de tests
  d'EC-04 inclou aquests dos casos abans de considerar la mitigació de
  SSRF verificada a `/audit`. Bloquejant per a l'aprovació de
  `tasks.md`, no per a l'aprovació d'aquest `design.md`.

---

**Nota d'esmena (2026-08-03):** aquest `design.md` s'ha esmenat en
aquesta mateixa etapa de Stage 2 (encara no s'ha passat la porta humana
sobre aquest contingut) arran d'una revisió de seguretat feta pel
`security-engineer` **sobre el disseny, abans d'escriure cap codi**.
Els set findings (F-01 a F-07) es tanquen o s'accepten explícitament a
ADR-02, ADR-03 i §8 — cap requereix reobrir `proposal.md`/
`requirements.md`. Vegeu el front-matter (`security_review`) per a la
data d'esmena.
