# Test report: add-terraform-language-pack

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-add-terraform-language-pack.md` (issue #52)
- Tester: tester subagent (Claude Code)
- Scope: behavioral tests for the new Terraform/OpenTofu language pack on branch `feat/52/add-terraform-language-pack` (5 implementation commits vs `main`). Covers `scripts/detect-languages.sh` terraform branch, `packs/languages/terraform/verify.sh` mode dispatch + fail-open guard, `internal/scaffold.PackFS("terraform")` resolution via `ralph pack list`, and `.claude/rules/terraform.md` frontmatter contract.
- Evidence: `docs/evidence/test-2026-05-13-add-terraform-language-pack.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-check-mojibake.sh` (pre-existing) | 11 | 11 | 0 | 0 | <1s |
| `tests/test-check-skill-sync.sh` (pre-existing) | 6 | 6 | 0 | 0 | <1s |
| `tests/test-ralph-cli-driver.sh` (pre-existing) | 48 | 48 | 0 | 0 | ~6s |
| `tests/test-detect-languages-terraform.sh` (NEW) | 8 | 8 | 0 | 0 | <1s |
| `tests/test-terraform-pack-verify.sh` (NEW) | 32 | 32 | 0 | 0 | <1s |
| `tests/test-terraform-rule-frontmatter.sh` (NEW) | 9 | 9 | 0 | 0 | <1s |
| `go test ./... -count=1` | 9 pkgs | 9 | 0 | 0 SKIP at suite-level (1 pre-existing `TestAvailablePacks_WithMockFS` t.Skip inside `internal/scaffold`, unchanged) | ~26s |
| `go run ./cmd/ralph pack list` (integration) | 1 | 1 | 0 | 0 | <1s |
| **Total** | **114** | **114** | **0** | **0** | **~35s** |

`./scripts/run-test.sh` exit code: **0**. Full evidence at `docs/evidence/test-2026-05-13-add-terraform-language-pack.log`.

### Plan test plan ↔ executed coverage

| Plan item | Executed in | Result |
| --- | --- | --- |
| Unit: `internal/scaffold` recognises `terraform` (`AvailablePacks` / `PackFS`) | `go test ./internal/scaffold/...` (pre-existing) + `go run ./cmd/ralph pack list` live check | PASS — output lists `terraform` |
| Integration: temp dir + `.tf` → `detect-languages.sh` emits `terraform` | `test-detect-languages-terraform.sh` Case 1 | PASS |
| Integration: no markers → warning + exit 0 | `test-terraform-pack-verify.sh` Case A | PASS |
| Integration: `.tf` + no `terraform`/`tofu` on PATH → error + exit 1 (fail-open regression guard) | `test-terraform-pack-verify.sh` Case B | PASS |
| Integration: `.tf` + CLI + no `.terraform/` → `validate` skipped | `test-terraform-pack-verify.sh` Case C | PASS |
| Integration: `static` / `test` / `all` mode dispatch | `test-terraform-pack-verify.sh` Cases C, E, F, G + bonus H (unknown mode → exit 2) | PASS |
| Integration: `ralph pack list` includes `terraform` (PackFS auto-discovery) | `go run ./cmd/ralph pack list` | PASS |
| Regression: `go test ./...` whole-suite green | `go test ./... -count=1` (fresh, no cache) | PASS (9 packages, 0 fail) |
| Edge case: `.tofu`-only OpenTofu repo → emit `terraform` | `test-detect-languages-terraform.sh` Case 2 | PASS |
| Edge case: `.terraform.lock.hcl` only (no `.tf`) → emit | `test-detect-languages-terraform.sh` Case 3 | PASS |
| Edge case: `*.tftest.hcl` absent → test mode skips with exit 0 | `test-terraform-pack-verify.sh` Case E | PASS |
| Edge case: `.terraform/` absent → `validate` skipped, overall pass | `test-terraform-pack-verify.sh` Case C | PASS |
| Verifier-recommended: deterministic check on `.claude/rules/terraform.md` frontmatter `paths:` entries | `test-terraform-rule-frontmatter.sh` | PASS (4 globs + key + fence + mirror byte-identity) |

### Additional behavioral coverage beyond the plan

| Bonus assertion | Where | Why it matters |
| --- | --- | --- |
| `.terraform/` directory containing a leftover `.tf` is pruned and does NOT trigger emit | `test-detect-languages-terraform.sh` Case 4 | Guards against false positives on init-leftover repos with no real sources. |
| Empty dir → no `terraform` emit | `test-detect-languages-terraform.sh` Case 5 | Negative-space sanity. |
| Mixed Go + Terraform repo emits both packs | `test-detect-languages-terraform.sh` Case 6 | Confirms terraform branch does not short-circuit other detectors. |
| Multiple markers → `terraform` emitted exactly once | `test-detect-languages-terraform.sh` Case 7 | Validates the existing `seen` dedup loop covers terraform. |
| `tofu` preferred over `terraform` when both on PATH (OpenTofu support) | `test-terraform-pack-verify.sh` Case D | Validates design decision #1 ([plan §Design decisions](../plans/archive/2026-05-13-add-terraform-language-pack.md)) at runtime. |
| `tflint` / `tfsec` / `trivy config` absent → "Skipping …" + still exit 0 | `test-terraform-pack-verify.sh` Case C | Optional-linter contract: missing optional tools must not fail the gate. |
| Unknown `HARNESS_VERIFY_MODE` → exit 2 with explicit error naming the bad mode | `test-terraform-pack-verify.sh` Case H | Documents the explicit-error contract instead of silent fallback. |
| `.terraform/` present → `validate` IS invoked (inverse of Case C) | `test-terraform-pack-verify.sh` Case I | Pins the positive branch so a future refactor cannot regress it to "always skip". |
| Lockfile-only repo (no `.tf`/`.tofu` source) → verifier still runs end-to-end | `test-terraform-pack-verify.sh` Case J | Catches a regression where `has_markers()` might only inspect source extensions. |
| `mode=test` must NOT invoke `fmt` / `validate` | `test-terraform-pack-verify.sh` Cases E, F | Cross-cuts the dispatch table. |

### Test methodology notes

- The `verify.sh` integration tests never invoke a real `terraform` or `tofu` binary. They build a sandboxed `PATH=<stubsdir>:/usr/bin:/bin` with shell-script stubs that record their argv to `calls.log` and exit 0. This isolates the verifier's branch logic from CLI presence on the host, makes the tests deterministic on machines without HashiCorp tooling, and lets us assert which subcommands were invoked (`fmt -check -recursive`, `validate`, `test`) by grepping the calls log.
- All tests use POSIX `sh` to match the existing `tests/` suite and the production `verify.sh`. No `bash`-isms.
- Each test creates its own `$(mktemp -d)` workspace and cleans up via `trap cleanup EXIT`.

## Coverage

- Statement: not measurable for shell — the repo has no instrumented coverage tool for shell scripts. Coverage tracked by case scope instead. Of `packs/languages/terraform/verify.sh`'s 93 lines, every code branch is exercised by at least one of the 32 `test-terraform-pack-verify.sh` cases:
  - `has_markers()` true / false / lockfile-only — Cases C/D/I/J (true) + A (false) + J (lockfile-only true)
  - `IAC_CLI` selection (tofu / terraform / neither) — Cases D (tofu) + C/E/F/G/H/I/J (terraform) + B (neither)
  - `run_static`: fmt always, `validate` gated on `.terraform/` (C skip + I run), tflint/tfsec/trivy missing → skip (C)
  - `run_tests`: `*.tftest.hcl` gate (E skip + F run)
  - `case "$mode"`: static/test/all (C+E+F+G) + unknown (H)
- Branch: every `if`/`elif`/`case` arm exercised. The only un-stubbed arms are the `tflint` / `tfsec` / `trivy` *invocation* paths (we only test the "command missing" arm); these are optional linters and out of scope per the plan's design decision #5.
- Function: `has_markers()`, `run_static()`, `run_tests()` all exercised.
- Notes:
  - `scripts/detect-languages.sh` terraform branch (line 41) is exercised by all 7 cases in `test-detect-languages-terraform.sh`.
  - `.claude/rules/terraform.md` frontmatter is pinned at 9 assertions (4 globs + paths-key + fence + mirror byte-identity + 2 existence checks).
  - Go-side coverage relies on the pre-existing `internal/scaffold` test suite. No new Go tests were added because `AvailablePacks()` reads from the embedded FS by directory listing, and the verifier subagent already confirmed `ralph pack list` enumerates `terraform`. The same check runs as an integration smoke test in this report.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| _none_ | — | — | — |

No failures. All 114 assertions pass.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `go test ./...` whole-suite (pre-existing baseline must remain green after pack addition) | PASS | `go test ./... -count=1` returned 0; all 9 packages with tests reported `ok`. |
| `./scripts/check-sync.sh` byte-identity between `packs/languages/<lang>/` and `templates/packs/<lang>/` for all existing languages (golang/dart/python/rust/typescript) | PASS | Invoked indirectly via `verify.local.sh` → `scripts/check-sync.sh` during the `./scripts/run-test.sh` static phase. Summary: `IDENTICAL: 148, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. |
| `tests/test-check-mojibake.sh` (hook smoke regressions) | PASS | 11/11. |
| `tests/test-check-skill-sync.sh` (skill drift) | PASS | 6/6. |
| `tests/test-ralph-cli-driver.sh` (driver dispatch + cross-review inversion + triage parser) | PASS | 48/48. |
| `./scripts/check-skill-sync.sh` (Claude ↔ Codex skill body lock-step) | PASS | Reported earlier in the verify step; unchanged by this PR. |
| Fail-open avoidance: markers present + no IaC CLI must exit 1 (Codex HIGH#2 reflected in plan design decision #2) | PASS — new regression guard | `test-terraform-pack-verify.sh` Case B. This is the load-bearing regression test for the fail-open behavior the plan calls out as Codex HIGH#2. |

## Test gaps

The following remain explicitly outside this report's scope:

1. **`ralph pack add terraform` end-to-end placement** — not tested. Per the plan's Non-goals and Risk R6, `internal/cli/pack.go:64-67` has a pre-existing placement bug (`addPack` writes to project root rather than `packs/languages/<lang>/`). The plan deliberately scopes acceptance to "`PackFS("terraform")` resolves", which the `ralph pack list` integration check satisfies. A follow-up issue is recommended at `/pr` time.
2. **Live `terraform` / `tofu` invocation against a real fixture** — not tested. All CLI invocations are stubbed. A future hermetic test (e.g. Docker + pinned `tofu` binary) could exercise the real `fmt -check -recursive` / `validate` / `test` exit codes, but this is out of scope for a unit/integration suite; correctness of the dispatch is fully covered by the stub-based assertions.
3. **`tflint` / `tfsec` / `trivy config` PASS paths** — only the "command missing → skip" arms are tested. Per plan design decision #5, these are optional linters; exercising their PASS paths would require pinning real linter binaries.
4. **Hidden-file glob (`**/.terraform.lock.hcl`) runtime path matching by Claude Code editor** — the verifier subagent's report already noted this is not deterministically verifiable from inside the repo (the consumer is the Claude Code editor's path matcher, external to this codebase). The frontmatter test pins the *contract*; runtime correctness is observational.
5. **Codex parity for terraform pack** — `.codex/AGENTS.override.md` and `templates/base/.codex/AGENTS.override.md` were not modified by this PR (the pack rule is consumed from `.claude/rules/terraform.md` mirror, which Codex reads via the shared `.claude/rules/` policy). No Codex-specific test is required.

None of the above gaps block PR creation; they are documented for follow-up planning.

## Verdict

- **Pass.** All 114 assertions across the new and pre-existing suites passed. `./scripts/run-test.sh` exit 0. `go test ./... -count=1` exit 0. The plan's test plan is fully covered, plus 8 additional behavioral assertions and a deterministic frontmatter contract test added per the verifier's recommendation.
- **Fail:** none.
- **Blocked:** none.

Proceed to `/sync-docs`, then `/cross-review`, then `/pr`. New test files are ready to be committed as a `test:` slice before the pipeline moves on:

- `tests/test-detect-languages-terraform.sh` (NEW)
- `tests/test-terraform-pack-verify.sh` (NEW)
- `tests/test-terraform-rule-frontmatter.sh` (NEW)
- `scripts/verify.local.sh` (wire-up: 3 new entries in `run_hook_tests`)
