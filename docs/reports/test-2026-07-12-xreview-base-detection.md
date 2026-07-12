# Test report: xreview-base-detection

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-xreview-base-detection.md
- Tester: tester subagent
- Scope: changed-language (shell + golang); full regression via run-test.sh glob
- Evidence: `docs/evidence/test-2026-07-12-xreview-base-detection.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-ralph-cli-driver.sh` | 103 | 103 | 0 | 0 | ~5s |
| `tests/test-xreview-gate-regression.sh` | 21 | 21 | 0 | 0 | ~2s |
| `tests/test-xreview-prompt-render.sh` | 54 | 54 | 0 | 0 | ~2s |
| `./scripts/run-test.sh` (full regression glob) | all | all | 0 | 0 | ~60s |
| Go: `go test ./...` | 8 pkgs | 8 pkgs | 0 | 0 | cached |

## Test 14 gate-proof cases (required by plan AC2)

| Case | Description | Result |
| --- | --- | --- |
| 14a-i | OLD detection yields feature branch (not main) | PASS |
| 14a-ii | OLD gate sees EMPTY diff (cross-review would be skipped) | PASS |
| 14a-iii | `detect_base_branch` yields 'main' (not the feature branch) | PASS |
| 14a-iv | NEW gate sees NON-EMPTY diff (cross-review correctly fires) | PASS |
| 14b | `RALPH_XREVIEW_BASE=develop` wins over origin/HEAD=main | PASS |
| 14c | non-main origin/HEAD (`develop`) is honoured | PASS |
| 14c-edge | `origin/release/1.0` strips only leading `origin/` → `release/1.0` | PASS |
| 14d-i | no origin/HEAD + refs/heads/main exists → main | PASS |
| 14d-ii | no origin/HEAD + no main + refs/heads/master exists → master | PASS |
| 14e | `git worktree add` fixture resolves origin/HEAD through shared common dir | PASS |

## Coverage

- Statement: Shell — no instrumented coverage; all Test 14 fixture branches are exercised by the 10 sub-cases above. Go packages — `go test` cached green across 8 packages.
- Branch: detect_base_branch resolution order fully exercised: env override (14b) → symbolic-ref (14a, 14c, 14c-edge, 14e) → main fallback (14d-i) → master fallback (14d-ii).
- Function: `detect_base_branch`, `run_agent`, `pick_reviewer`, `count_triage_findings`, `resolve_phase_model`, `write_model_receipt` all covered.
- Notes: No skipped tests. Go tests ran from cache (no source changes to Go packages in this branch).

## Failure analysis

None. All tests passed.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `HEAD@{upstream}` base detection → empty diff → cross-review silently skipped on pushed feature branch | FIXED | Test 14a-ii (old gate: empty diff) vs 14a-iv (new gate: non-empty diff) |
| cross-review prompt renderer with slashed branch names | PASS | test-xreview-prompt-render.sh Case 2 (release/3.5) |
| gate regression suite (ACTION_REQUIRED, clean, --fix-all+WORTH_CONSIDERING, RENDER_FAILED) | PASS | test-xreview-gate-regression.sh 21/21 |

## Test gaps

None identified for this plan. The worktree fixture (14e) covers the shared-common-dir path. The `RALPH_XREVIEW_BASE` env override is tested end-to-end (14b). The `release/X.Y` slash-in-ref edge case is covered by 14c-edge.

## Verdict

PASS — 103/103 + 21/21 + 54/54 on targeted suites; full regression glob clean; 0 failures across all shell suites and Go packages. Proceed to /pr.

---

## Cycle 2 addendum (2026-07-12)

**Trigger:** docs + 2-line orchestrator preserve-form fix committed since Cycle 1 (commits c3d89f2, d3c39f8, 9e79bdf).

**Commits since Cycle 1:**
- `c3d89f2 fix: preserve operator-supplied RALPH_XREVIEW_BASE in orchestrator`
- `d3c39f8 docs: self-review cycle 2 addendum for xreview-base-detection`
- `9e79bdf docs: verify cycle 2 addendum for xreview-base-detection`

### Test execution (Cycle 2)

| Suite / Command | Tests | Passed | Failed | Notes |
| --- | --- | --- | --- | --- |
| `tests/test-ralph-cli-driver.sh` | 103 | 103 | 0 | All 10 Test-14 gate-proof cases pass |
| `tests/test-ralph-orchestrator-branch-names.sh` | 3 | 3 | 0 | orchestrator branch-name regression |
| `tests/test-ralph-orchestrator-pr-strategy.sh` | 24 | 24 | 0 | PR strategy + override mismatch warning |
| `./scripts/run-test.sh` (full regression glob) | all | all | 0 | 0 failures across all shell suites and Go packages |

### Go packages (Cycle 2)

All 8 packages pass: `internal/action`, `internal/cli`, `internal/config`, `internal/scaffold`, `internal/state`, `internal/ui`, `internal/ui/panes`, `internal/upgrade` — some from cache, `internal/scaffold` freshly run.

### Regression specific to the orchestrator fix (c3d89f2)

The `RALPH_XREVIEW_BASE` preserve-form fix is covered by Test 14b (`RALPH_XREVIEW_BASE=develop` override wins): PASS. The orchestrator-pr-strategy suite (24 tests) exercises the orchestrator code path that surrounds the fix, with no regressions.

### Verdict (Cycle 2)

PASS — 103 + 3 + 24 targeted tests clean; full regression glob clean; 0 failures. No new coverage gaps introduced by the docs/fix commits.
