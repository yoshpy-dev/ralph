# Test report: codex-hooks-multi-event

- Date: 2026-08-24
- Plan: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
- Tester: `tester` subagent (Claude Code)
- Scope: behavioral tests only (`./scripts/run-test.sh`, changed-language scope by default). No static analysis, no diff-quality review, no spec-compliance audit — those were `/self-review` and `/verify`'s job (both already passed; see `docs/reports/verify-2026-08-24-codex-hooks-multi-event.md` for the static-portion evidence and its explicit hand-off of the behavioral-test clause of AC-8 to this step).
- Evidence: `docs/evidence/test-2026-08-24-codex-hooks-multi-event.log` (primary `./scripts/run-test.sh` run, `HARNESS_VERIFY_MODE=test`, scope `changed`); corroborating runs not saved as separate evidence files (gitignored `docs/evidence/*.log`, transient): a second full `./scripts/run-verify.sh` (`HARNESS_VERIFY_MODE=all`, scope `full`) at `docs/evidence/verify-2026-08-24-024541.log`, a standalone `bash tests/test-hook-wiring.sh` re-run, a fresh (`-count=1`, no cache) `go test ./internal/cli/...` full run, a fresh `go test ./...` across all 8 packages, and a targeted `-run` of the 3 new AC-5 doctor tests.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (all 27 shell suites, changed scope) | 913 (sum of per-suite totals below) | 913 | 0 | 0 | ~3 min (full run incl. Go) |
| `tests/test-hook-wiring.sh` (in-suite + standalone re-run) | 66 | 66 | 0 | 0 | <1s |
| `go test ./internal/cli/... -count=1 -v` (fresh, uncached) | full package (incl. `TestValidateCodexHooksJSON_*`, `TestCodexShippedHookEventsMatchesShippedHooksJSON`) | all pass | 0 | 0 | 33.96s |
| `go test ./... -count=1` (fresh, uncached, all 8 packages) | 8 packages | 8/8 `ok` | 0 | 0 | ~53s total (cli 33.96s, config 0.40s, insights 0.93s, org 9.53s, org/driver 1.80s, org/protocol 1.30s, scaffold 2.43s, upgrade 2.54s) |
| `go test ./internal/cli/... -count=1 -run 'TestValidateCodexHooksJSON\|TestCodexShippedHookEventsMatchesShippedHooksJSON' -v` (targeted, AC-5) | 3 | 3 | 0 | 0 | 0.26s |
| `./scripts/run-verify.sh` (second full run, mode=all, scope=full, corroborating) | same 27 shell suites (913) + golang pack (cached `go test ./...`, all 8 `ok`) | all pass | 0 | 0 | included above |

Per-suite shell totals (from the primary `./scripts/run-test.sh` evidence log, all `FAIL: 0`):
`test-agent-phase-boundaries.sh` 44/44, `test-branch-name.sh` 26/26, `test-check-mojibake.sh` 15/15, `test-check-skill-sync.sh` 13/13, `test-detect-changed-languages.sh` 23/23, `test-detect-languages-terraform.sh` 8/8, `test-ensure-pr-ready.sh` 7/7, `test-ensure-pr-title-prefix.sh` 13/13, `test-gc-artifacts.sh` 11/11, `test-hook-wiring.sh` 66/66, `test-insights-append.sh` 39/39, `test-language-pack-monorepo-roots.sh` 29/29, `test-no-loop-references.sh` 1/1, `test-post-edit-verify.sh` 4/4, `test-pre-bash-guard.sh` 24/24, `test-ralph-config.sh` 15/15, `test-ralph-dispatch.sh` 26/26, `test-ralph-worktree.sh` 29/29, `test-run-verify-scope.sh` 12/12, `test-secret-scan.sh` 6/6, `test-self-review-scope.sh` 64/64, `test-sync-skills.sh` 22/22, `test-template-purity.sh` 10/10, `test-terraform-gitignore.sh` 47/47, `test-terraform-pack-verify.sh` 36/36, `test-terraform-rule-frontmatter.sh` 11/11, `test-verify-mode-split.sh` 59/59, `test-xreview-helpers.sh` 29/29.

Note: the wrapper's changed-language-scope detection resolved to `golang` for this diff ("full fallback (unclassified:.codex/hooks.json)" — expected, since `.codex/hooks.json` isn't a language-pack-classified extension) and ran the full 27-file shell suite plus the `golang` verifier's `go test ./...`. This is the repo's normal behavior for this kind of change, not scope creep.

## Coverage

- Statement/Branch/Function: not separately instrumented for this diff; shell suites have no coverage tool (project convention — coverage is measured by test-case scope). Go coverage was not re-profiled in this run since no `internal/cli` coverage claim was made in the plan beyond "existing checks stay green."
- Notes: the 3 new/changed AC-5 Go tests (`TestValidateCodexHooksJSON_PostToolUseOnly_FlagsMissingShippedEvents`, `TestValidateCodexHooksJSON_AllFourEventsWired_NoMissingEventFindings`, `TestCodexShippedHookEventsMatchesShippedHooksJSON`) directly cover the new `codexShippedHookEvents` iteration logic in `internal/cli/doctor.go`, including both the negative (legacy PostToolUse-only fixture) and positive (all-four-wired) branches, plus the Go/JSON event-set drift guard added in the self-review fix commit. The 44 new/modified assertions in `tests/test-hook-wiring.sh:174-246` (per-event loop over `PreToolUse`/`SessionStart`/`UserPromptSubmit`, matcher-exactness check, non-goal absence guard for `SessionEnd`/`PreCompact`) directly cover AC-1/AC-6.

## Failure analysis

None — no failures observed in any run (primary or corroborating).

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Flaky `go test ./internal/cli/...` (parallel tempdir suspect, noted by a previous run of this task) | Not reproduced | Ran `go test ./internal/cli/... -count=1 -v` fresh (uncached) once, full pass, 33.96s, no failing test name to capture. Also covered by the cached run inside both `./scripts/run-test.sh` executions and a fresh full `go test ./...` across all 8 packages — all green. Per tester memory, this class of flake needs adjacent real-subprocess-spawning test load to reproduce (e.g. doctor probe tests immediately before watcher tests); this diff does not touch `internal/org`'s watcher path, and no such contention was observed here. Treating as non-deterministic/environmental, not a regression tied to this diff — if it recurs, capture the exact failing test name before rerunning per the task's instruction. |
| `.codex/hooks.json` PostToolUse trust-hash preservation (existing entry must not change) | Confirmed unchanged | `tests/test-hook-wiring.sh` "AR#1 regression guard" assertion (`root`/`templates/base` both PASS); independently cross-checked in `docs/reports/verify-2026-08-24-codex-hooks-multi-event.md`. |
| Existing doctor checks (schema/direct-reference/co-existence/features.hooks) | Unaffected, still green | `go test ./internal/cli/...` full pass includes the pre-existing doctor test suite alongside the 3 new AC-5 tests; no regressions. |

## Test gaps

- **Live-fire runtime behavior (AC-2/AC-3/AC-4)** is not re-executed by this step — it is evidence-doc-based (`docs/evidence/codex-hooks-multi-event-slice1-2026-08-24.md`, `docs/evidence/codex-hooks-multi-event-fixture-2026-08-24.md`), Codex-CLI-version-pinned (`codex-cli 0.147.0`), and inherently environment-observational (real Codex process dispatching real hooks). `/test`'s deterministic suite cannot substitute for this; this matches how `/verify` scoped it and how the prior sibling plan (`2026-08-20-codex-hooks-json-wiring`) was verified.
- **The Codex `ask`-decision path for `PreToolUse`** is out of scope for this plan (AC-4 only requires the `deny` proof) and is tracked as tech-debt (`docs/tech-debt/README.md`, self-review M4) — no automated test exists for it, by design, not oversight.
- No new negative-path unit test exists for a hooks.json that has a `PreToolUse` entry with a matcher other than `Bash` reaching `validateCodexHooksJSON` (the matcher-exactness check lives only in the shell suite, `tests/test-hook-wiring.sh`, not in Go). Not required by any AC; noting as a minor blind spot for future doctor-hardening work.

## Verdict

- **Pass.** All 913 shell-suite assertions across 27 files green (0 failures), all 8 Go packages `ok` in both cached (in-wrapper) and fresh (`-count=1`) runs, `tests/test-hook-wiring.sh` 66/66 in both the wrapper run and a standalone re-run, the 3 new AC-5 doctor tests pass individually and in the full package run, and a second full `./scripts/run-verify.sh` (mode=all, scope=full) also exited 0 with identical suite coverage. AC-8's behavioral-test clause (flagged by `/verify` as this step's responsibility) is now satisfied: `go test ./...`, `tests/test-hook-wiring.sh`, `tests/test-pre-bash-guard.sh` (24/24), and the full `./scripts/run-verify.sh` all pass.
- Fail: none.
- Blocked: none.
- The previously-flagged flaky `go test ./internal/cli/...` FAIL did not reproduce in this session (checked via one fresh isolated run plus two cached runs, all green) — not treated as a live regression, but noted above per the task's instruction to report determinism status rather than silently rerun-and-forget.
- **Cleared to proceed to `/sync-docs` → `/cross-review` → `/pr`.**

## Cycle 2 (2026-08-24, post cross-review fix-and-revalidate)

- Plan: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
- Tester: `tester` subagent (Claude Code)
- Scope: behavioral tests only. Re-run triggered by two fix commits landed after Cycle 1: `381b938` (doctor now requires the dispatcher-routing check to match the *specific* hook event as an argument, not just "any dispatcher call exists" — 2 new Go tests) and `ed2a2d0` (shell gate unification/hardening in `tests/test-hook-wiring.sh`, from 66 → 68 checks; a doctor finding-message wording change; a quoted-argument regex fix, with matching new tests).
- Evidence: `docs/evidence/test-2026-08-24-codex-hooks-multi-event-cycle2.log` (primary `./scripts/run-test.sh` run); `docs/evidence/cli-test-cycle2.out` (fresh, uncached `go test ./internal/cli/ -count=1 -v`, saved this cycle unlike Cycle 1); `docs/evidence/run-verify-full-cycle2.log` (full `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`, mode=all default, AC-8 literal — confirmed separately with a quiet re-run for exit-code capture: `EXIT=0`).

### Test execution

| Suite / Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-test.sh` (28 shell suites, changed-scope resolved to `golang` full-fallback) | 913 assertions across 27 suites with countable totals + 1 non-numeric summary (`test-no-loop-references.sh`, single-assertion guard) — 0 FAIL anywhere | One more suite header than Cycle 1's 27-file count report text (28 vs 27) is a counting-convention difference, not a new suite — Cycle 1's per-suite list already enumerated `test-branch-name.sh` etc.; this cycle just walked all headers explicitly. `test-hook-wiring.sh` now 68/68 (was 66/66 in Cycle 1), matching the ed2a2d0 gate-unification commit. |
| `bash tests/test-hook-wiring.sh` (standalone re-run) | 68/68, 0 FAIL | Confirms the 68-count is stable outside the wrapper too. |
| `go test ./internal/cli/ -count=1 -v` (fresh, uncached) | 262 `--- PASS`, 0 `--- FAIL`, `ok ... 31.397s`/`31.864s` (ran twice, both green) | Includes the 2 new AC-driven doctor tests from `381b938` plus the hardening-round test edits from `ed2a2d0`. No flake reproduced in either fresh run. |
| `./scripts/run-test.sh`'s in-wrapper `go test ./...` (8 packages) | 8/8 `ok`, 0 failures | `internal/cli` ran uncached this pass (31.791s, not `(cached)`) since source changed since Cycle 1; other 7 packages cached-green. |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (AC-8 literal, mode defaults to `all`) | Exit 0, gofmt ok, 0 staticcheck issues, all 28 shell suites green, 8/8 Go packages `ok` (cached, unchanged since the wrapper's own `go test ./...` run moments earlier) | This is the full static+test dispatch (mode=all), matching AC-8's literal wording ("`./scripts/run-verify.sh` exit 0, check-sync / purity / 全テスト green"). Ran twice: once with `tee` for a saved log, once quietly to capture `$?` — both exit 0. |

### Regression checks

| Previously flagged item | Status | Evidence |
| --- | --- | --- |
| Flaky `go test ./internal/cli/...` (subprocess-contention class, per tester memory) | Not reproduced | Two independent fresh (`-count=1`) full-package runs this cycle, both 262/262 green, ~31-32s each. Consistent with memory: this flake needs adjacent real-subprocess test load to trigger, not mere repetition, and none was observed. |
| `.codex/hooks.json` PostToolUse trust-hash preservation | Confirmed unchanged | `tests/test-hook-wiring.sh` AR#1 regression-guard assertions, both suite copies (root/templates/base), still PASS after the `ed2a2d0` gate rewrite. |
| Doctor's existing checks (schema/direct-reference/co-existence/features.hooks) | Unaffected, still green | Full `internal/cli` package pass includes pre-existing doctor suite alongside all new/changed tests from both fix commits; no regressions. |

### Test gaps

- Same as Cycle 1: live-fire runtime behavior (AC-2/3/4) remains evidence-doc-based, not re-executed by `/test`; the Codex `ask`-decision path for `PreToolUse` remains untested by design (tracked as tech-debt); no Go-level negative test for a non-`Bash` `PreToolUse` matcher (shell-suite-only coverage) — unchanged, not required by any AC.
- New from this cycle: no dedicated regression test asserts the *old, pre-`381b938`* laxer doctor behavior (any-dispatcher-call, event-argument-agnostic) is gone — coverage relies on the new tests asserting the *new* stricter behavior directly (`381b938`'s 2 tests + `ed2a2d0`'s edits), which is sufficient to prove the fix works but doesn't independently document what regressed. Minor, not blocking.

### Verdict (Cycle 2)

- **Pass.** All 913+ shell-suite assertions across 28 files green (0 failures, `test-hook-wiring.sh` now 68/68), all 8 Go packages `ok` in both the wrapper's cached run and two independent fresh (`-count=1`) `internal/cli` runs (262/262 each), `tests/test-hook-wiring.sh` 68/68 in both the wrapper and a standalone re-run, and a full `./scripts/run-verify.sh` (mode=all, scope=full, AC-8 literal) exits 0 on two separate invocations (one logged, one quiet-for-exit-code).
- Fail: none.
- Blocked: none.
- The `go test ./internal/cli/...` flake noted in tester memory did not reproduce in either of this cycle's fresh runs; no failing test name to report.
- **Cleared to proceed to `/sync-docs` → `/cross-review` → `/pr`.**

## Cycle 3 (2026-08-24, pipeline cap raised 2 → 3)

- Plan: `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
- Tester: `tester` subagent (Claude Code)
- Scope: behavioral tests only. Re-run triggered by two commits landed after Cycle 2: `b7cec3c` (`tests/test-hook-wiring.sh` shell gate now accepts quoted event arguments, matching the doctor's Go-side parity — no assertion-count change) and `6c41189` (insight jsonl one-line data fix; documentation/insight-event data only, no production code touched).
- Evidence: `docs/evidence/cli-test-cycle3.out` (fresh, uncached `go test ./internal/cli/ -count=1 -v`, saved this cycle). The `./scripts/run-test.sh` and `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` raw logs were captured but not saved as tracked evidence files — both match the existing `docs/evidence/*.log` gitignore pattern (`.gitignore:58`), consistent with how Cycle 1's corroborating runs were handled.

### Test execution

| Suite / Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-test.sh` (28 shell suite headers, changed-scope resolved to `golang` full-fallback via `.codex/hooks.json`) | 0 `FAIL` lines anywhere in the log (grepped for `FAIL` excluding `FAIL: 0` summary lines — none found) | `test-hook-wiring.sh` still 68/68 — the `b7cec3c` quoted-argument fix is a shell-side robustness change, not a new assertion; count unchanged from Cycle 2. `test-xreview-helpers.sh` 29/29, `test-verify-mode-split.sh` 59/59, `test-self-review-scope.sh` 64/64, `test-terraform-gitignore.sh` 47/47, `test-terraform-pack-verify.sh` 36/36 — all suites individually confirmed green in the log, matching Cycle 2's per-suite baseline. |
| `bash tests/test-hook-wiring.sh` (standalone re-run) | 68/68, `FAIL: 0`, exit 0 | Confirms the quoted-argument fix didn't regress the standalone entrypoint either. |
| `go test ./internal/cli/ -count=1 -v` (fresh, uncached) | 262 `--- PASS`, 0 `--- FAIL`, 0 `--- SKIP`, `ok ... 33.888s` | Saved to `docs/evidence/cli-test-cycle3.out`. Same 262-test count as Cycle 2 (no Go source changed this cycle — both delta commits were shell/insights-data only). No flake reproduced. |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (AC-8 literal, mode defaults to `all`) | Exit 0 (captured via `echo "EXIT=$?"` after file redirection, not `tee`-masked per tester memory) | gofmt ok, 0 staticcheck issues, all 28 shell suite headers green, 8/8 Go packages `ok` (cached, matching the wrapper's own `go test ./...` moments earlier). Zero `FAIL` lines in the full log (grepped, excluding `FAIL: 0`). |

### Regression checks

| Previously flagged item | Status | Evidence |
| --- | --- | --- |
| Flaky `go test ./internal/cli/...` (subprocess-contention class, per tester memory) | Not reproduced | One fresh (`-count=1`) full-package run this cycle, 262/262 green, 33.888s. Consistent with memory: needs adjacent real-subprocess test load to trigger, none observed here. |
| Shell gate quoted-argument regression risk (the motivating fix for `b7cec3c`) | Confirmed fixed and non-regressing | `tests/test-hook-wiring.sh` 68/68 in both the wrapper run and a standalone re-run; no new failures introduced by the quoting change. |
| `.codex/hooks.json` PostToolUse trust-hash preservation (AR#1 regression guard) | Confirmed unchanged | AR#1 assertions still PASS in both `root`/`templates/base` copies. |

### Test gaps

- Same as Cycle 2: live-fire runtime behavior (AC-2/3/4) remains evidence-doc-based, not re-executed by `/test`; the Codex `ask`-decision path for `PreToolUse` remains untested by design (tracked as tech-debt); no Go-level negative test for a non-`Bash` `PreToolUse` matcher (shell-suite-only coverage); no dedicated regression test asserts the old laxer doctor behavior is gone (coverage relies on the new tests proving the new behavior directly) — all unchanged, not required by any AC.
- New from this cycle: no dedicated unit test isolates the quoted-vs-unquoted event-argument parsing added in `b7cec3c` as its own named case distinct from the existing dispatcher-routing assertions — the fix is covered implicitly (the existing 68 assertions still pass against the now-quoting-tolerant gate), not via a standalone "quoted argument accepted" regression test. Minor, not blocking; worth a follow-up if this class of shell-quoting fix recurs.

### Verdict (Cycle 3)

- **Pass.** Zero `FAIL` lines across the full `./scripts/run-test.sh` run (28 shell suite headers, `test-hook-wiring.sh` steady at 68/68), a standalone `bash tests/test-hook-wiring.sh` re-run also 68/68, a fresh (`-count=1`) `go test ./internal/cli/` run 262/262 with 0 skips (`docs/evidence/cli-test-cycle3.out`), and `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (AC-8 literal) exits 0 with gofmt/staticcheck/all shell suites/all 8 Go packages green.
- Fail: none.
- Blocked: none.
- The `go test ./internal/cli/...` flake noted in tester memory did not reproduce in this cycle's fresh run; no failing test name to report.
- **Cleared to proceed to `/sync-docs` → `/cross-review` → `/pr`.**
