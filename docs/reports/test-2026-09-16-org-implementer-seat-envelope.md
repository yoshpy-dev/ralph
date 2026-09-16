# Test report: org-implementer-seat-envelope

- Date: 2026-09-16
- Plan: `docs/plans/active/2026-09-16-org-implementer-seat-envelope.md`
- Tester: `tester` subagent (Claude Code, sonnet)
- Scope: `git diff main...HEAD` (HEAD `463e943`). Behavioral tests only (`./scripts/run-test.sh`, changed-language scope which fell back to full Go scope because `scripts/ralph-config.sh` is language-unclassified), plus `go test ./... -count=1` and per-package `go test -cover`. No static analysis (verifier's scope, already PASS per `docs/reports/verify-2026-09-16-org-implementer-seat-envelope.md`).
- Evidence: `docs/evidence/test-2026-09-16-org-implementer-seat-envelope.log` (`./scripts/run-test.sh` raw output, exit 0)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (full shell suite, 23 files under `tests/`) | 835 assertions | 835 | 0 | 0 | ~110s (dominated by internal/cli Go tests) |
| `go test ./... -count=1` (fresh, uncached, run directly) | 8/8 packages | 8 | 0 | 0 | ~55s wall |
| `go test ./... -count=1` (re-run inside `run-test.sh`'s golang verifier, full-scope fallback) | 8/8 packages | 8 | 0 | 0 | ~47s wall (cli/org uncached, rest cached) |
| `tests/test-ralph-config.sh` | 15 | 15 | 0 | 0 | <1s |
| `tests/test-no-loop-references.sh` | 1 | 1 | 0 | 0 | <1s |
| `internal/config` targeted (`TestDefault*`, `TestLoad_IgnoresRetiredOrgBudgetTable`, `TestDefaultsLockStep`) | 6 | 6 | 0 | 0 | 0.25s |
| `internal/org` targeted (`DefaultModelForDriverAndRole`, prompts, `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation`) | 8 | 8 | 0 | 0 | 0.44s |
| `internal/org` all `TestWatch*` | 24 | 24 | 0 | 0 | included above |
| `internal/cli` `TestCheckCodexModelSlugs_*` (doctor codex-slug check, 5 paths) | 5 | 5 | 0 | 0 | 0.30s |
| `internal/cli` `TestOrgSpawn_ModelFlagOmitted_*` / `TestOrgStart_ModelFlagOmitted_*` | 4 | 4 | 0 | 0 | 0.59s |

Shell total: 835 assertions across 23 files in `tests/` (710 counted via the `PASS:`/`FAIL: 0` summary-block style + 125 via the `N passed, 0 failed` style used by `test-branch-name.sh`, `test-ensure-pr-ready.sh`, `test-ensure-pr-title-prefix.sh`, `test-gc-artifacts.sh`, `test-insights-append.sh`, `test-ralph-worktree.sh`). `./scripts/run-test.sh` exited 0.

Note on scope: `RALPH_VERIFY_SCOPE` defaults to `changed`, but `detect-languages.sh` classified `scripts/ralph-config.sh` (touched by this plan's model-pool/budget-removal slices) as language-unclassified, which triggers the documented "unclassified → full fallback" rule (`tests/test-detect-changed-languages.sh` covers this exact behavior). The run therefore executed the **full** `go test ./...` across all 8 packages rather than a changed-package subset — a stronger check than the default changed-scope would have given, and appropriate given this plan touches `internal/config`, `internal/org`, and `internal/cli` together.

## Coverage

- `internal/config`: 93.5% (informational; plan's Assumptions/AC-3/AC-6 surface area — model pool, `[org.budget]` retirement)
- `internal/org`: 89.1% (AC-1, AC-2, AC-7, AC-7b surface — prompts, watch)
- `internal/org/driver`: 92.0% (unaffected by this plan; unchanged)
- `internal/org/protocol`: 97.9% (unaffected by this plan; unchanged)
- `internal/cli`: 80.9% (AC-4, AC-5 surface — doctor codex-slug check, spawn/start `--model` fallback); above the 80% guideline in `.claude/rules/ralph/code-review.md`'s review checklist
- `internal/insights`: 86.1%, `internal/scaffold`: 75.7%, `internal/upgrade`: 91.2% (unaffected by this plan; recorded for baseline continuity per tester memory)
- Branch/function breakdowns: not separately instrumented; `go test -cover` reports statement coverage only, consistent with prior test reports for this repo.

## Failure analysis

None. No failing tests in any suite.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `watch_test.go`'s budget-era tests removed without leaving `Cutoff`/`OrgStartTS` dependents broken (plan Risk register item 1) | Confirmed fixed | All 24 `TestWatch*` tests pass; `grep -n "Cutoff\|OrgStartTS" internal/org/watch.go internal/org/watch_test.go` (re-run here) → zero hits. |
| Legacy `[org.budget]` `ralph.toml` must not error on `Load()` (AC-6 edge case) | Confirmed fixed | `TestLoad_IgnoresRetiredOrgBudgetTable` passes; TOML unknown-section decoding silently ignores the retired table. |
| Legacy watch-status fixture with budget `Conditions`/`PendingAlerts`/`Escalated` must not misfire deadman escalation (AC-7b, the plan's primary regression risk) | Confirmed fixed | `TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation` passes; hand-written legacy fixture (including the retired `"cutoff"` JSON field) is pruned before `checkDeadman` runs, no escalation fires. |
| `spawn`'s prior behavior (hard-required `--model`, `org.go:182`) must become a warn-and-fallback, not silently break existing scripts that already pass `--model` | Confirmed fixed, non-regressive | `TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry` (omitted case) and a live `--model fable` run (explicit case, see Integration checks) both succeed identically in shape; explicit `--model` still short-circuits the fallback (no warning printed). |
| `doctor`'s existing `--probe-models` path (`checkOrgModelProbes`) must stay unchanged alongside the new codex-slug Check | Confirmed unaffected | `git diff main...HEAD -- internal/cli/doctor.go` shows only the new Check registration (11 lines), not a rewrite of the probe path; `go test ./internal/cli/...` (full package, 80.9% cover) shows no probe-related test failures. |

## Integration checks (plan's Test plan section)

- `ralph org spawn --dry-run --role implementer --driver claude --model fable --org-id test-org --id implementer-1 --cwd <worktree> --allow-unscoped --state-dir <scratch>` → `spawned seat "implementer-1" (... dry_run=true)`, exit 0.
- Same command **without** `--model` → stderr: `org: --model omitted; falling back to first [org].model_pool entry permitted for role implementer on claude: fable (pass --model explicitly)`, followed by the same successful `spawned seat "implementer-2" ...` line, exit 0. Confirms AC-5's fail-open fallback (not fail-closed) end-to-end, not just via the CLI unit test.
- `ralph org spawn` for `--driver codex` without `--model` (with `--scope`, since codex's default permission mode requires it) printed the analogous fallback line — `org: --model omitted; falling back to first [org].model_pool entry permitted for role implementer on codex: gpt-6-astra (pass --model explicitly)` — confirming the plan's edge case "codex driver without `--model` resolves `gpt-6-astra` (dry-run)". The command then exits 1 on an **unrelated**, pre-existing, fail-closed guard (`codex seat permission mode "autonomous" not yet live-verified; only guarded is allowed`) that requires `[org.permissions].codex_verified=true` after live-verifying the codex CLI — explicitly out of this plan's scope (Non-goals: "codex 座席の autonomous / edits permission mode の実機検証(`codex_verified` は据え置き)"). The model-resolution behavior under test (the warning text and `gpt-6-astra` value) is confirmed printed before that unrelated gate runs, which is what the plan's edge case asks to verify; full end-to-end codex spawn success is intentionally out of scope and also covered statically by `TestDefaultModelForDriver_DefaultPoolHeads` (pins `gpt-6-astra` as codex's pool head).
- `ralph doctor` output contains the `Org codex model slugs` line. On this machine (real `~/.codex/models_cache.json`, no `CODEX_HOME` override), it currently reports **warn** — `1 codex model_pool slug(s) not found ...: gpt-6-astra` — rather than the `pass` recorded in yesterday's verify report. This is not a code regression: it reflects the real codex model cache on this machine having changed between 2026-09-16 (verify pass) and 2026-09-17 (this test run), which is exactly the drift scenario the plan's Risk register anticipates ("codex スラッグはモデル更新で陳腐化する... doctor Check で早期検知"). The Check is working as designed by surfacing it as `warn`, not silently passing or hard-failing.
- `CODEX_HOME` override honored: pointing `CODEX_HOME` at a scratch directory with no `models_cache.json` produced `ℹ Org codex model slugs: info — codex models cache not found at <scratch>/models_cache.json — skipping`, confirming the env override is read (`doctor_codex_models.go:22`) and the missing-cache path degrades to `info`, not `warn`/`fail`.
- Old `ralph.toml` with `[org.budget]` loads: confirmed via `TestLoad_IgnoresRetiredOrgBudgetTable` (unit-level, table-driven with the exact legacy section text).

## Test gaps

- Live, non-dry-run `ralph org spawn` with a real herdr/agmsg driver was not exercised — this dev environment lacks agmsg/herdr tooling (same limitation noted in the verify report); `--dry-run` is the practical ceiling here, consistent with existing CLI test conventions in this repo.
- Full end-to-end codex-driver spawn success (post `codex_verified=true`) was not exercised, matching the plan's explicit Non-goal.
- Branch/function coverage percentages are not available (`go test -cover` in this repo reports statement coverage only); this is a pre-existing tooling gap, not new to this plan.
- The `ralph doctor` warn-vs-pass result for the live machine check is environment-dependent (depends on the current state of `~/.codex/models_cache.json`) and will vary run-to-run as codex updates its cache; the fixture-based unit tests (`TestCheckCodexModelSlugs_*`) are the durable, deterministic source of truth for this Check's 3 (+2 bonus) code paths.

## Verdict

- Pass: all 835 shell assertions across 23 `tests/*.sh` files, all 8 Go packages (`go test ./... -count=1`, both a fresh uncached run and the cached run-test.sh re-run), all plan-named unit tests (config defaults/budget-ignore/model-pool-lockstep, org prompts implementer/fan-out/lead-delegation, org watch AC-7/AC-7b including the budget-prune regression test, cli doctor codex-slug 3(+2) paths, cli spawn/start `--model` omission fallback), and all integration/edge-case checks from the plan's Test plan section (dry-run spawn with/without `--model` for both claude and codex drivers, `ralph doctor`'s codex-slug line, `CODEX_HOME` override, legacy `[org.budget]` toml load).
- Fail: none.
- Blocked: none.

Behavioral test execution is complete for this plan. Proceeding to `/pr` is unblocked from `/test`'s perspective (deferred items above are pre-existing environment limitations, not plan-introduced gaps).
