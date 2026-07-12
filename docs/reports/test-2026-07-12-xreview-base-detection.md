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
