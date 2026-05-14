# Test: worktree-first standard flow

- Date: 2026-05-14
- Branch: `feat/worktree-first-standard-flow`
- Related issue: #77
- Verdict: Pass

## Behavioral Tests

| Test | Status | Evidence |
| --- | --- | --- |
| `tests/test-ralph-worktree.sh` | Pass | 17/17 assertions passed: state path, default branch, clean-base validation, ensure/resume/current, collision refusal, dirty-base refusal, cleanup removing worktree/local branch/state. |
| `./scripts/run-test.sh` | Pass | Full test wrapper passed, including shell suites and `go test ./...`. |

Evidence:

- `docs/evidence/verify-2026-05-14-102418.log`

## Coverage Notes

- The new helper is tested in a temporary Git repository so branch/path and cleanup behavior do not depend on this repo's current state.
- `cleanup --force-branch` is covered only after a state-backed worktree exists; arbitrary force deletion is not exposed by the test.
