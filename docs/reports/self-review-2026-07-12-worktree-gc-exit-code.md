# Self-review report: worktree-gc-exit-code

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-worktree-gc-exit-code.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff main...HEAD` — `scripts/ralph-worktree.sh`, `templates/base/scripts/ralph-worktree.sh`, `tests/test-ralph-worktree.sh`, plus the plan doc. Diff quality only; no spec/static/test verdicts.

## Evidence reviewed

- **Code change** (`scripts/ralph-worktree.sh:333-336`): the trailing short-circuit `[ "$count" -eq 0 ] && printf 'No stale...'` (whose truth value became the function return, yielding `return 1` whenever a stale entry was found/pruned) is replaced by an explicit `if [ "$count" -eq 0 ]; then printf ...; fi` followed by `return 0`. Semantically identical listing/prune behavior; only the exit code changes to a guaranteed 0. The `if ...; then ... fi` form matches the pre-existing style in the same function (lines 327-331), so it is style-consistent.
- **Mirror byte-identity**: `cmp scripts/ralph-worktree.sh templates/base/scripts/ralph-worktree.sh` → identical. `git ls-files --stage` shows both at mode `100755`, same blob hash `5810372`. Mirror discipline satisfied.
- **Tests fail against old code**: swapped in `git show main:scripts/ralph-worktree.sh` and ran the new test file → runner exits **1** (output halts after case (a)). Restored new code → **29 passed, 0 failed**. So the added cases genuinely gate the fix.
- **Spot-check case (b)/(c) assert rc=0 where old code returned 1**: confirmed. Direct probe of the old script with one stale entry: `gc` exits 1, `gc --prune` exits 1. New assertions `assert_eq "gc with stale entry exits 0" 0 "$rc"` and `assert_eq "gc --prune exits 0" 0 "$rc"` target exactly that regression.
- **No stray content**: diff is confined to the two mirrored scripts, the test file, and the plan. No debug code, secrets, TODO markers, or unrelated formatting churn.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | Under the test file's `set -eu`, the capture idiom `_gc_out="$(... gc ...)"; rc=$?` cannot cleanly record a non-zero rc: if `gc` returns non-zero, the assignment line itself aborts the whole script before `rc=$?` runs, so the intended `assert_eq ... exits 0` never prints a legible FAIL. Verified: `sh -eu -c '_out="$(false-returning-fn)"; rc=$?; echo reached'` never prints "reached". Consequence against the *old* code — cases (b)(c)(d) do not reach their assertions; the suite still fails (runner exit 1) but via a silent mid-script abort at the first stale case, not a labeled assertion failure. Harmless for the fixed code (gc always returns 0 now), but a future re-regression would surface as an opaque abort rather than a named FAIL. | New test lines using `rc=$?` after `"$RALPH_WORKTREE" gc ...`; shell probe above | Optional follow-up: capture rc without tripping `set -e`, e.g. `set +e; _gc_out="$(... gc ...)"; rc=$?; set -e` (matching the existing `assert_exit` helper's own `set +e`/`set -e` bracket), or route exit-code checks through `assert_exit`. Not blocking. |

## Positive notes

- Minimal, targeted 3-line fix; the `return 0` makes the success contract explicit rather than implicit in the last expression's truth value — a readability win.
- Test coverage is thorough for a small fix: exit code, message text, STALE listing format, prune-vs-no-prune deletion, and the non-stale negative case are all asserted, and the `_live_json` fixture is cleaned up.
- Mirror and file mode were kept in lock-step, which is the most common miss in this repo's dual-tree layout.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| — | — | — | — | — |

_(No tech-debt entries added; the LOW finding is an optional test-robustness nicety, not deferred work.)_

## Recommendation

- Merge: yes. No CRITICAL or HIGH findings. The fix is correct, style-consistent, mirrored byte-for-byte, and its tests fail against the pre-fix code and pass (29/29) against the fix.
- Follow-ups: optionally harden the new gc test's exit-code capture to survive `set -e` so a future re-regression fails as a named assertion rather than a silent abort (LOW).
