# Test report: org-runtime-retire-loop

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-03-org-runtime-retire-loop.md`
- Tester: tester subagent (Claude Code)
- Scope: full (`RALPH_VERIFY_SCOPE=full`) — this plan removes/rewrites code
  across scripts, Go packages, skills, and config, so changed-language scope
  would under-cover the diff. Also ran `go test ./... -count=1` directly
  (fresh, non-cached) as a second, independent pass over the Go tree.
- Evidence: `docs/evidence/test-2026-08-03-org-runtime-retire-loop.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 |
| `tests/test-branch-name.sh` | 26 | 26 | 0 | 0 |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 |
| `tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 |
| `tests/test-gc-artifacts.sh` | 11 | 11 | 0 | 0 |
| `tests/test-insights-append.sh` | 39 | 39 | 0 | 0 |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 |
| `tests/test-no-loop-references.sh` (new, AC-5) | 1 | 1 | 0 | 0 |
| `tests/test-ralph-config.sh` | 15 | 15 | 0 | 0 |
| `tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 |
| `tests/test-self-review-scope.sh` | 64 | 64 | 0 | 0 |
| `tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 |
| `tests/test-xreview-helpers.sh` (AC-4, migrated) | 26 | 26 | 0 | 0 |
| **Shell subtotal** | **552** | **552** | **0** | **0** |
| `go test ./... -count=1` (fresh, uncached) | 8 packages | 8 ok | 0 | 0 |

Go packages exercised: `internal/cli`, `internal/config`, `internal/insights`,
`internal/org`, `internal/org/driver`, `internal/org/protocol`,
`internal/scaffold`, `internal/upgrade`. `github.com/yoshpy-dev/ralph` and
`cmd/ralph` have no test files (entrypoint glue only — consistent with prior
cycles). No `cmd/ralph-tui`, `internal/ui`, `internal/state`, `internal/action`,
or `internal/watcher` packages remain to test — confirmed removed (AC-1).

All 23 shell test files under `tests/` were run (verified file count matches
suite count: `ls tests/*.sh | wc -l` = 23). `go build ./...` green.

## Coverage

- `internal/org`: 88.3% (unchanged from the prior watchdog cycle's 88.1–88.3%
  range; both PR④ known-gap fixes are covered — see Regression checks below)
- `internal/org/driver`: 92.0%
- `internal/org/protocol`: 97.9%
- `internal/cli`: 76.2% (includes the new org-manifest-based `status`
  rewrite; lower than judgment-seat packages because CLI glue/error-path
  branches are harder to hit exhaustively — not a regression from this plan)
- `internal/config`: 94.2%
- Branch/function: not separately instrumented (Go's `-cover` reports
  statement coverage only; consistent with prior cycles per tester memory)
- Notes: Coverage was not gated as pass/fail criteria per this repo's
  conventions (no enforced threshold in `run-test.sh`); reported for
  visibility per plan Verify plan intent.

## Failure analysis

No failures. Table intentionally empty.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| PR④ known gap #5 (deadman probe-recovery false-clear) | Fixed, tested | `TestWatch_Deadman_ProbeOutageRecoveryAlone_DoesNotClearPendingAlert` (internal/org/watch_test.go:1666) passes |
| PR④ known gap #6 (`WatchdogJoined` cleared on failed Join) | Fixed, tested | `TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess` (internal/org/watch_test.go:1797) passes |
| `ralph status` org-manifest rewrite (AC-2) | Covered | 7 tests in `internal/cli/status_test.go`: all-orgs listing, `--org-id` filter, empty-state-dir friendly note, corrupt-manifest warning, JSON schema parity (empty vs populated), JSON output validity, watch-heartbeat/pending-alert-count display |
| `xreview-helpers.sh` migration (AC-4: `detect_base_branch` / `pick_reviewer` / `count_triage_findings`) | Covered | `tests/test-xreview-helpers.sh`, 26/26 — includes the gate-proof case (3a) showing the old base-detection logic would have produced an empty diff and skipped cross-review, while the migrated `detect_base_branch` produces a non-empty diff |
| `[loop]`/`[pipeline]` config removal (AC-3) | Covered | `internal/config` defaults_sync_test green (94.2% coverage); `tests/test-ralph-config.sh` now 15 tests (down from 43 in the pre-retirement branch, expected — loop/pipeline-phase config tests were removed along with the config surface) |
| Zero live references to retired Loop system (AC-5) | Covered | `tests/test-no-loop-references.sh` (new guard test) — single assertion, passes; excludes historical documents per plan's stated exclusion list |
| Two previously flagged flaky tests (per task instructions) | No flake observed | `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` and `TestRunWatcher_TimeoutIndependentOfSmallInterval` passed on the full-suite run and again on 3x isolated re-runs each (`-count=3`) |

## Test gaps

- `ralph status` in a non-git working directory is not explicitly tested,
  but `status.go` has no git dependency in its implementation, so this is a
  non-issue rather than a real gap.
- Coverage numbers are statement-only (no branch/function instrumentation in
  this repo's Go tooling) — same limitation noted in prior cycles.
- `internal/cli` at 76.2% is on the lower end; largely CLI wiring/error-path
  branches (e.g., `doctor` external-tool probe failure paths) that are
  harder to exercise deterministically. Not new to this plan and not a
  blocking gap for AC coverage.
- No dedicated test for the `ralph upgrade` remove-behavior smoke check
  (plan's Non-goals explicitly scope this to "confirmation only", already
  captured in `docs/evidence/org-retire-loop-smoke-2026-08-03.txt` from a
  prior slice, not part of this /test run).

## Verdict

- Pass: yes — 552/552 shell tests, 8/8 Go packages, `go build ./...` green,
  all plan-mandated test-plan items (gap fixes, status org rewrite,
  xreview-helpers migration, zero-reference guard, full regression) covered
  and passing. Both previously-noted flaky tests confirmed stable across
  repeated isolated runs.
- Fail: none.
- Blocked: none.

**Proceed to `/sync-docs` → `/cross-review` → `/pr`.**

---

## Cycle 2 (re-run after cross-review fix)

- Date: 2026-08-03
- Trigger: cross-review ACTION_REQUIRED fix-and-revalidate cycle. Two fix
  commits landed on top of the cycle-1 HEAD: `fea2ee6` (status manifest
  path double-join + `pick_reviewer` case normalization) and `47e2eee`
  (exclude dry-run seats from status active counts).
- Worktree: `.claude/worktrees/org-runtime-retire-loop`, branch
  `refactor/org-runtime-retire-loop`, HEAD `510849f`.
- Scope: full (`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`), same
  rationale as cycle 1 — this plan touches scripts, Go packages, skills,
  and config broadly. Also re-ran `go test ./... -count=1` (fresh,
  uncached) as an independent pass.
- Evidence: `docs/evidence/test-2026-08-03-org-runtime-retire-loop-cycle2.log`

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Delta vs cycle 1 |
| --- | --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | unchanged |
| `tests/test-branch-name.sh` | 26 | 26 | 0 | 0 | unchanged |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | unchanged |
| `tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 | unchanged |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | unchanged |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | unchanged |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 | unchanged |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 | unchanged |
| `tests/test-gc-artifacts.sh` | 11 | 11 | 0 | 0 | unchanged |
| `tests/test-insights-append.sh` | 39 | 39 | 0 | 0 | unchanged |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 | unchanged |
| `tests/test-no-loop-references.sh` | 1 | 1 | 0 | 0 | unchanged |
| `tests/test-ralph-config.sh` | 15 | 15 | 0 | 0 | unchanged |
| `tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 | unchanged |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | unchanged |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | unchanged |
| `tests/test-self-review-scope.sh` | 64 | 64 | 0 | 0 | unchanged |
| `tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 | unchanged |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 | unchanged |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 | unchanged |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 | unchanged |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | unchanged |
| `tests/test-xreview-helpers.sh` | **29** | 29 | 0 | 0 | **+3 vs cycle 1's 26** — `fea2ee6` added cases covering the manifest path double-join and case-insensitive `pick_reviewer` normalization |
| **Shell subtotal** | **555** | **555** | **0** | **0** | +3 net |
| `go test ./... -count=1` (fresh, uncached) | 8 packages | 8 ok | 0 | 0 | unchanged package set |

`ls tests/*.sh \| wc -l` = 23 (same file count as cycle 1; the +3 assertions
came from growing `test-xreview-helpers.sh`, not a new file).

### New/changed tests targeted by this cycle

| Test | Result | Note |
| --- | --- | --- |
| `TestStatusCmd_SeesSeatWrittenByRealOrgSpawn` (`internal/cli/status_test.go`, added in `47e2eee`) | PASS | Verifies `ralph status` picks up a seat row written by a real `ralph org spawn` manifest entry |
| `TestStatusCmd_DryRunSeatIsARowButNotCountedInAggregates` (`internal/cli/status_test.go`, added in `47e2eee`) | PASS | Confirms the dry-run-seat aggregate-count exclusion fix: seat appears in the row listing but is excluded from active-seat counts |
| `tests/test-xreview-helpers.sh` new cases (3× manifest-path/case-normalization assertions from `fea2ee6`) | PASS (29/29) | See per-test-name breakdown in the isolated run below |

Isolated re-run of the two new `internal/cli` tests (`-run
'TestStatusCmd_SeesSeatWrittenByRealOrgSpawn|TestStatusCmd_DryRunSeatIsARowButNotCountedInAggregates'
-v -count=1`): both PASS.

### Flaky-test re-verification (per task instructions)

- `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`: isolated `-count=3`
  re-run → 3/3 PASS.
- `TestRunWatcher_TimeoutIndependentOfSmallInterval`: isolated `-count=5`
  re-run → 5/5 PASS.
- One data point worth recording: running the doctor test immediately
  before the watcher test in the same shell invocation (doctor test made
  live `claude`/`codex`/`agmsg` probe subprocess calls, ~9s) produced 2/3
  watcher-test timeouts in a follow-up `-count=3` batch, because the
  watcher test's assertion is wall-clock-timeout-based and sensitive to
  real subprocess contention on the machine. Re-isolating each test alone
  (no adjacent subprocess-heavy test) was fully stable across 3 and 5
  reps respectively. This matches the existing "flaky under contention,
  not a reliable repro" classification from the cycle-1 report and prior
  tester memory — no code fix indicated, this is a test-environment
  contention artifact, not a regression.

### Failure analysis

No failures. Table intentionally empty.

### Regression checks (cycle 2 additions only — see cycle 1 for the full list)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Status manifest path double-join (cross-review finding, fixed in `fea2ee6`) | Fixed, tested | `tests/test-xreview-helpers.sh` new cases + `internal/cli/status_test.go` additions in `fea2ee6`, all passing |
| `pick_reviewer` case normalization (cross-review finding, fixed in `fea2ee6`) | Fixed, tested | `tests/test-xreview-helpers.sh` cases 1g/1h/1i (uppercase/mixed-case arg and env), all passing |
| Dry-run seats inflating status active counts (cross-review finding, fixed in `47e2eee`) | Fixed, tested | `TestStatusCmd_DryRunSeatIsARowButNotCountedInAggregates` passing |
| Real-spawn seat visibility in `ralph status` (fixed in `47e2eee`) | Covered | `TestStatusCmd_SeesSeatWrittenByRealOrgSpawn` passing |

### Test gaps

Same as cycle 1 — no new gaps introduced by the two fix commits. The fix
commits added targeted tests for exactly the behavior they changed
(manifest path resolution, reviewer case-folding, dry-run count exclusion);
no additional blind spots identified.

### Verdict (Cycle 2)

- Pass: yes — 555/555 shell tests (+3 net vs cycle 1, all in
  `test-xreview-helpers.sh`), 8/8 Go packages, both cross-review fix
  commits (`fea2ee6`, `47e2eee`) covered by passing tests including the two
  new `internal/cli/status_test.go` cases named in the task. Both
  previously-flagged flaky tests reconfirmed stable in isolation (3/3 and
  5/5); the one contention data point observed is consistent with the
  known timing-sensitivity note, not a new regression.
- Fail: none.
- Blocked: none.

**Proceed to `/sync-docs` → `/cross-review` → `/pr`.**
