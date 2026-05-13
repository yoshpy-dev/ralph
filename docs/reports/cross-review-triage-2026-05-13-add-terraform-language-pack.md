# Cross-review triage report: add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 4 (3 LOW in inline table + 1 P2 summary)
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=3

## Triage context

- Active plan: `docs/plans/active/2026-05-13-add-terraform-language-pack.md`
- Self-review report: `docs/reports/self-review-2026-05-13-add-terraform-language-pack.md`
- Verify report: `docs/reports/verify-2026-05-13-add-terraform-language-pack.md`
- Implementation context summary: Issue #52 adds a Terraform/OpenTofu language pack. The pack body, rule file, detect-languages.sh wiring, and recipe doc are all complete and verified. The plan explicitly defers the `ralph pack add <lang>` pathing bug (Codex /plan HIGH#1) as out-of-scope. Tests live in `tests/test-*-terraform-*.sh` and exercise hermetic stubs via PATH narrowing.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | **[P2] Terraform verifier tests are non-hermetic on hosts with system-installed IaC tooling.** `clean_path="/usr/bin:/bin"` (`tests/test-terraform-pack-verify.sh:86`) still exposes `/usr/bin/terraform`, `/usr/bin/tofu`, `/usr/bin/tflint`, `/usr/bin/tfsec`, or `/usr/bin/trivy` if any of them are apt-installed (common in CI). Tests that intend to exercise the "no CLI" or "optional tool absent" branches will silently call real binaries instead, producing false PASS or random failure depending on what the host has. | Real issue (Axis 1 YES) — test hermeticity is a foundational property of behavioral regression suites. Worth fixing (Axis 2 YES) — the tester's report cites 8 stub-CLI scenarios that depend on this guarantee; if the guarantee is leaky on CI hosts, those 8 assertions can't catch regressions in the fail-open code path (which itself is a Codex /plan HIGH#2 mitigation we explicitly committed to). Fix is small and contained to the test file. | `tests/test-terraform-pack-verify.sh:83-86`, `tests/test-terraform-pack-verify.sh` stub-CLI scenarios that rely on `clean_path` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

(none)

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|
| 1 | [LOW] `verify.sh` validates `HARNESS_VERIFY_MODE` only after marker detection — `HARNESS_VERIFY_MODE=foo` in an empty dir gets silent exit 0 instead of a config error. | Codex itself notes this "matches no other pack's behavior" — peer packs (`golang`, `python`, `rust`) early-exit on missing markers before mode handling. Diverging from peers for stricter feedback would be inconsistent. | style-preference |
| 2 | [LOW] Marker-detection `find` expression duplicated between `verify.sh:11-12`, `verify.sh:67`, and `scripts/detect-languages.sh:41`. | Codex itself says "acceptable for a small pack; if a fourth copy appears, extract `has_iac_sources`." Already triaged in self-review LOW #2 (same content). Refactoring across pack/script boundaries is out of scope for this PR. | out-of-scope |
| 3 | [LOW] `.claude/rules/terraform.md` declares `paths: - "**/.terraform.lock.hcl"`; hidden-file glob handling by the editor matcher is not exercised by any existing rule in the repo. | Already triaged in self-review LOW #3 and verify report (verifier deferred to `/test`, and tester noted runtime hidden-glob behavior is a Claude Code editor concern, not a deterministic test target). Codex itself classifies this as "not a diff-quality blocker." | already-addressed |

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
