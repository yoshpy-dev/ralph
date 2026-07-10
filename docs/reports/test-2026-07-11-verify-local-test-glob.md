# Test Report: verify-local-test-glob

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-verify-local-test-glob.md
- Branch: fix/verify-local-test-glob (base main, commit e89acfe)
- Runner: `./scripts/run-test.sh` (test mode; local verifier + golang pack)
- Evidence: docs/evidence/test-2026-07-11-verify-local-test-glob.log
- Prior: self-review (MERGE), verify (PASS)

## Verdict: PASS

`./scripts/run-test.sh` exited **0**. All 28 `tests/test-*.sh` suites executed
(was 23 before this fix), all Go packages `ok`, and the edge case confirms a
failing test still regresses the gate (exit 1).

## Behavioral gate result

- `./scripts/run-test.sh` → EXIT 0
- 28 `tests/test-*.sh` suites discovered (all executable), all 28 executed via
  the new glob loop in `run_hook_tests()`
- 714 shell-test PASS assertions total, 0 FAIL
- No `--- FAIL` / `not ok` / hard-failure markers anywhere in the log
  (the `FAIL: 0` lines are per-suite zero-failure tallies; `PASS: ... fails
  closed` lines are negative-path assertions that passed)

## The 5 formerly-skipped suites (the fix's core value)

Before this change these 5 were absent from the hand-maintained `if [ -x ... ]`
list and ran in **no** mode. They now execute via the glob and all pass:

| Suite | PASS assertions |
|-------|-----------------|
| tests/test-ralph-config.sh | 27 |
| tests/test-ralph-signals.sh | 3 |
| tests/test-ralph-status.sh | 51 |
| tests/test-xreview-gate-regression.sh | 21 |
| tests/test-xreview-prompt-render.sh | 54 |

Subtotal: 156 assertions, 0 failures — all newly gated by this fix.

## All 28 suites executed (pass counts)

| Suite | PASS |
|-------|------|
| test-agent-phase-boundaries.sh | 45 |
| test-branch-name.sh | 29 |
| test-check-mojibake.sh | 12 |
| test-check-skill-sync.sh | 8 |
| test-detect-changed-languages.sh | 24 |
| test-detect-languages-terraform.sh | 9 |
| test-ensure-pr-ready.sh | 7 |
| test-ensure-pr-title-prefix.sh | 13 |
| test-language-pack-monorepo-roots.sh | 30 |
| test-ralph-cli-driver.sh | 49 |
| **test-ralph-config.sh** (new) | 27 |
| test-ralph-dry-run-side-effects.sh | 5 |
| test-ralph-orchestrator-branch-names.sh | 3 |
| test-ralph-orchestrator-pr-strategy.sh | 24 |
| test-ralph-run-options.sh | 5 |
| **test-ralph-signals.sh** (new) | 3 |
| test-ralph-slice-skip-pr.sh | 4 |
| **test-ralph-status.sh** (new) | 51 |
| test-ralph-worktree.sh | 17 |
| test-run-verify-scope.sh | 13 |
| test-secret-scan.sh | 7 |
| test-self-review-scope.sh | 97 |
| test-terraform-gitignore.sh | 48 |
| test-terraform-pack-verify.sh | 37 |
| test-terraform-rule-frontmatter.sh | 12 |
| test-verify-mode-split.sh | 60 |
| **test-xreview-gate-regression.sh** (new) | 21 |
| **test-xreview-prompt-render.sh** (new) | 54 |

Total: 28 suites, 714 PASS assertions, 0 FAIL. (**new** = formerly-skipped.)

## Go test packages (regression guard — unchanged by this diff)

`go test ./...` via the golang pack: all packages `ok`, no `--- FAIL`.

```
?   github.com/yoshpy-dev/ralph                 [no test files]
?   github.com/yoshpy-dev/ralph/cmd/ralph       [no test files]
?   github.com/yoshpy-dev/ralph/cmd/ralph-tui   [no test files]
ok  github.com/yoshpy-dev/ralph/internal/action    (cached)
ok  github.com/yoshpy-dev/ralph/internal/cli       (cached)
ok  github.com/yoshpy-dev/ralph/internal/config    (cached)
ok  github.com/yoshpy-dev/ralph/internal/scaffold  (cached)
ok  github.com/yoshpy-dev/ralph/internal/state     (cached)
ok  github.com/yoshpy-dev/ralph/internal/ui        (cached)
ok  github.com/yoshpy-dev/ralph/internal/ui/panes  (cached)
ok  github.com/yoshpy-dev/ralph/internal/upgrade   (cached)
ok  github.com/yoshpy-dev/ralph/internal/watcher   (cached)
```

The diff touches only `scripts/verify.local.sh` (glob) and 7 shell test files
(shellcheck fixes / `cd` guards). No Go source changed; Go results are a clean
regression guard. `==> All verifiers passed.` closes the run.

## Edge case: a failing test still fails the gate

Fail-closed proof that the glob loop actually enforces (not just discovers):

1. Created throwaway executable `tests/test-zz-tmp-fail.sh` = `#!/bin/sh` + `exit 1`.
2. Ran `HARNESS_VERIFY_MODE=test ./scripts/verify.local.sh` → **overall exit 1**.
3. The throwaway was picked up by the glob (`==> tests/test-zz-tmp-fail.sh` →
   `FAIL`), and **all 28 real suites still ran** (29 `==> tests/` headers total).
   Because `zz` sorts last, every other suite executed before it and reported
   its normal pass counts — the glob does not abort on first failure discovery,
   it runs all then propagates the failure.
4. Deleted the throwaway; `git status --porcelain` shows only the two
   pre-existing untracked report artifacts (self-review, verify) — tracked tree
   clean, no leftover from the edge test.

This confirms the fix does not weaken the gate: newly-discovered suites are
enforced, not merely listed.

## Test gaps / notes

- No automated test asserts "glob discovers a newly-added test file" — the edge
  case above exercises this manually, but there is no permanent regression test
  for the enumeration-drift class this PR fixes. Optional follow-up: a
  meta-test that drops a temp `tests/test-*.sh` and asserts it appears in the
  test-mode run. Not blocking; the structural glob change makes future drift
  unlikely by construction.
- Go packages ran from cache (`(cached)`); results are valid but not a cold
  rebuild. This diff changes no Go source, so cache reuse is appropriate.

## Do NOT block PR

Behavioral gate is green. Static analysis / shellcheck (AC3, AC5) are the
verifier's scope and are out of this report; the verify report already recorded
PASS.
