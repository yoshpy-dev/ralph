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
