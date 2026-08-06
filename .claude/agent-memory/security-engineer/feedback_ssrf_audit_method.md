---
name: ssrf-audit-mutation-testing
description: When auditing a security control that already has "regression tests", mutation-test the control instead of trusting the suite — Niu NIU-6 proved a blocking-gate test could not detect the regression it was named for.
metadata:
  type: feedback
---

When auditing a security mitigation that ships with a named regression test,
**revert the mitigation in the production code and re-run the test**. Do not
accept "the regression test passes" as evidence the regression is covered.

**Why:** during the NIU-6 `/audit` (2026-08-03), the SSRF suite's
`TestFetchPreview_RedirectToSameHost_EachHopReDialed_F01Regression` was
declared a **blocking** NFR-06 gate test for the F-01 keep-alive bypass.
Flipping `DisableKeepAlives` to `false` in `internal/fetchsafe/client.go`
left the integration test **passing unchanged** — its destination was
loopback, so the redirect chain never progressed past hop 0 and the
keep-alive path was never exercised at all. Only a literal-code unit test
(`TestNewTransport_DisableKeepAlivesTrue`) caught it. The test's own comment
was honest about the limitation but drew a conclusion that did not follow.

**How to apply:**
- Mutation-test each mitigation before writing "no regression" in a finding.
  `cp` the file, `sed` the flag, run the suite, restore. Cheap and decisive.
- Watch for the specific anti-pattern: **a test whose subject is blocked by
  an earlier layer never exercises the later layer it claims to verify.**
  In SSRF suites this is systemic — any local listener binds to loopback,
  and loopback is itself a forbidden destination, so behavioural assertions
  about redirect/connection-reuse mechanics are structurally unreachable
  unless the IP gate is deliberately stubbed for that test.
- To prove per-hop re-dial behaviour for real, inject an instrumented
  `ControlContext` that counts invocations and permits loopback during the
  test, then assert `dials == hops`.
- Clean up every probe file before finishing; leave `git status` untouched.

Related: [[niu-fetchsafe-ssrf-model]]
