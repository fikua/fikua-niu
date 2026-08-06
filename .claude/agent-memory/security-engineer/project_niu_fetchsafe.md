---
name: niu-fetchsafe-ssrf-model
description: Niu's internal/fetchsafe validates the fetch DESTINATION but not the RECOVERED OG values — the boundary that matters when reviewing anything downstream of link preview.
metadata:
  type: project
---

`app/internal/fetchsafe` is the single audited egress point for user-supplied
URLs in Niu (NIU-6 link preview). Its trust boundary is narrower than it
first appears.

**What it validates:** the *destination of the fetch* — scheme, hostname
denylist, resolved IP at `ControlContext` (the only IP-classification point,
paired with `DisableKeepAlives: true` so every redirect hop re-dials),
per-hop scheme re-validation, 5-hop cap, 5s context timeout, 2MiB
`io.LimitReader`, credential-free client. Audited 2026-08-03 against 18
independent bypass vectors (int/octal/hex-encoded IPv4, IPv6 zone IDs,
NAT64/6to4, mapped IPv4, userinfo/fragment confusion, punycode, happy-eyeballs
multi-address, Content-Type confusion) — **all correctly rejected**.

**What it validates about recovered values (added 2026-08-03, commit
c1f7274, re-audited and probe-verified):** `og:image` must parse as an
absolute http(s) URL with a non-empty host (`isHTTPOrHTTPSURL` in
`ogparse.go`), else it is silently discarded. Title/description/image are
rune-truncated to 300/1000/2048. The denylist now strips ONE trailing dot
(commit 878692a), so `niu.fikua.com.` is denied; `niu.fikua.com..` still
isn't, but has an empty DNS label and does not resolve — not exploitable.

**What it still does NOT validate:** the *destination* of the recovered
`og:image` — only its scheme. `http://127.0.0.1/x.png` is happily stored.
That is fine today because the **browser**, not the server, fetches it and
the CSP (`img-src 'self' https:`) blocks it.

**Why:** the design (`docs/changes/NIU-6-.../design.md` ADR-02) framed the
threat model entirely as "where does the request go", never "what comes
back". That framing is correct for SSRF proper and the mitigation is genuinely
strong — the original gap was completeness, not concept.

**How to apply:** `image_url`/`title`/`description` from `activity_ideas`
are now scheme-safe and length-bounded, but still **attacker-chosen
content**. The dangerous move is a new **server-side** sink (image proxy,
thumbnailer, email embed, outbound API call): scheme validation says nothing
about destination, so any such sink must route `image_url` back through
`fetchsafe` as a fetch destination. Client-side `img.src` is already covered.

Related: [[niu-auth-model]], [[ssrf-audit-mutation-testing]]
