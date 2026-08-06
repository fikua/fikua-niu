# Security Engineer — Memory Index

- [Niu auth model](project_niu_auth_model.md) — shell is public by design, only `/api/v1/*` is server-gated; read before any "leaks when unauthenticated" review
- [Niu pre-existing issues](project_niu_preexisting.md) — open redirect via `next`, static dir listings; verify origin before filing against a branch
- [Niu fetchsafe SSRF model](project_niu_fetchsafe.md) — validates the fetch destination, NOT the recovered OG values; treat image_url/title/description as attacker-controlled
- [SSRF audit method](feedback_ssrf_audit_method.md) — mutation-test security controls; a "regression test" blocked by an earlier layer proves nothing
