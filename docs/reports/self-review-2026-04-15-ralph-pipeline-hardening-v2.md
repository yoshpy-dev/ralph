# Self-review report: Ralph Pipeline Hardening (v2 -- post-Codex fix)

- Date: 2026-04-15
- Plan: feat/ralph-pipeline-hardening
- Reviewer: reviewer subagent (claude-opus-4-6)
- Scope: Diff quality of uncommitted signal handling fix in `scripts/ralph-orchestrator.sh` plus full branch diff (15 files, 866 additions, 31 deletions). Focused on changes since v1 self-review.

## Evidence reviewed

- `git diff -- scripts/ralph-orchestrator.sh` (uncommitted changes -- the signal handling fix)
- `git diff main...HEAD` (full branch diff -- committed changes)
- Working tree state of `scripts/ralph-orchestrator.sh` lines 80-128 (signal handling section)
- Cross-reference: v1 self-review findings (all 4 MEDIUM items confirmed fixed in committed code)
- Cross-reference: Codex triage ACTION_REQUIRED #1 (signal exit) and #2 (status overwrite) against the fix
- `scripts/ralph-config.sh` full file (new, 57 lines)
- `scripts/ralph-pipeline.sh` full file (1030 lines)
- `tests/test-ralph-signals.sh` full file (204 lines)
- `tests/test-ralph-config.sh` full file (185 lines)
- `docs/tech-debt/README.md` for stale entries

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | Tech debt entry for "orchestrator.tmp.json not PID-suffixed" is stale. This was fixed in commit `5140f3a` -- both `cleanup_on_exit` (line 121) and `main()` (line 946) now use `orchestrator.tmp.$$.json`. The debt entry at `docs/tech-debt/README.md` line 16 still describes this as unfixed. | `docs/tech-debt/README.md` line 16: "Potential race between cleanup trap and normal exit path writing the same temp file on SIGINT during final jq call". Compare `scripts/ralph-orchestrator.sh` lines 121 and 946: both use `.$$.json` suffix. | Remove or mark as resolved in `docs/tech-debt/README.md`. Stale debt entries erode trust in the debt register. |
| MEDIUM | maintainability | `_on_signal()` calls `exit 1` which under `set -eu` triggers the EXIT trap as intended. However, in POSIX sh, the exit status delivered to the EXIT trap after a signal is implementation-defined. The `_INTERRUPTED` flag approach correctly avoids relying on `$?` -- this is good. But `_on_signal` does not re-raise the signal to the parent process. A calling script that `wait`s on the orchestrator will see exit code 1 (generic failure) rather than 128+SIGNUM (conventional signal exit). | `scripts/ralph-orchestrator.sh` lines 91-94: `_on_signal() { _INTERRUPTED=1; exit 1; }`. Compare with conventional pattern: `trap 'cleanup; trap - INT; kill -INT $$' INT`. | LOW priority. The orchestrator's callers (`scripts/ralph` and manual invocation) do not distinguish exit 1 from signal exit. If future callers need this distinction, change to: `_on_signal() { _INTERRUPTED=1; trap - INT TERM; kill -s INT $$; }` to re-raise. Acceptable as-is for current usage. |
| LOW | readability | The signal handling test `test_orchestrator_status_on_interrupt` at `tests/test-ralph-signals.sh` lines 146-175 simulates the jq status-update logic manually rather than testing the actual `_on_signal` + `cleanup_on_exit` code path. It proves jq can update JSON but does not verify the `_INTERRUPTED` flag gating behavior (which is the core of the Codex fix). | `tests/test-ralph-signals.sh` line 164: `jq --arg s "interrupted" '.status = $s'` -- this is a direct jq call, not the orchestrator's `cleanup_on_exit`. The actual `_INTERRUPTED=0` check (line 118 of orchestrator) is never exercised. | Consider adding a test case that runs the orchestrator in dry-run, sends SIGINT, then checks that `orchestrator.json` status becomes "interrupted". Also add a negative test: orchestrator exits normally (dry-run completes) and confirm status is NOT overwritten to "interrupted". The `test_sigint_cleanup` test (line 75) already runs dry-run + SIGINT but does not assert on `orchestrator.json` status. |
| LOW | readability | WORTH_CONSIDERING Codex finding #3 (exit code collision between `gh_unavailable` and codex `ACTION_REQUIRED` -- both return 1 from `run_outer_loop`) is not addressed in this fix round. The outer loop caller at `ralph-pipeline.sh` lines 975-986 treats all `_outer_result=1` as "regress to Inner Loop", so a missing `gh` CLI causes infinite regression until `max_iterations`. | `scripts/ralph-pipeline.sh` line 778: `return 1` for `gh_unavailable`. Line 754: `return 1` for ACTION_REQUIRED. Line 981-984: both land in the same `1)` case arm. | This is a known limitation, not a new regression from the signal fix. Track as tech debt or use distinct return codes (e.g., `return 2` for `gh_unavailable`). The preflight probe warns about missing `gh` at startup, so the practical impact is low. |

## Positive notes

- The `_INTERRUPTED` flag approach is the correct fix for Codex ACTION_REQUIRED #2. It cleanly separates signal interrupts (user pressed Ctrl-C) from normal non-zero exits (partial failures, `set -eu` exit), preserving the "partial" status that `main()` writes.
- Splitting `trap _on_signal INT TERM` and `trap cleanup_on_exit EXIT` into two separate traps is the standard POSIX pattern. The previous single `trap cleanup_on_exit INT TERM EXIT` had the problem that INT/TERM handlers cannot distinguish signal-caused entry from normal exit -- this is fixed.
- The v1 self-review MEDIUM findings are all confirmed fixed:
  - `--permission-mode` added to preflight probes (lines 297, 332).
  - `validate_all_numeric()` called at startup in all 3 scripts (orchestrator line 57, pipeline line 68, loop line 51).
  - `orchestrator.tmp.json` now PID-suffixed (lines 121, 946).
  - `ralph_claude_flags()` removed along with its tests (confirmed absent from all files).
- The comment at line 116-117 ("Update orchestrator.json status ONLY on genuine signal interrupts. Normal non-zero exits preserve their own status.") clearly documents the intent of the `_INTERRUPTED` check.
- The `_CHILD_PIDS` tracking and dual cleanup strategy (in-memory PID list + on-disk `.pid` files) provides defense-in-depth for orphan prevention.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Exit code collision in `run_outer_loop`: `gh_unavailable` and codex `ACTION_REQUIRED` both return 1 | Missing `gh` CLI causes wasted inner loop cycles until `max_iterations` is reached | Preflight probe warns at startup; practical impact is low because `gh` is almost always present | If a user reports unexpected `max_iterations` exhaustion with `gh` CLI absent | Codex triage finding #3, this report |
| Signal test does not exercise `_INTERRUPTED` gating logic | The core fix (flag-based status update) has no dedicated test | The dry-run + SIGINT test is timing-dependent and may be flaky; improving it requires careful process lifecycle management | If the `_INTERRUPTED` logic regresses in a future change | This report |

_(Stale debt entry: "orchestrator.tmp.json not PID-suffixed" in `docs/tech-debt/README.md` should be removed -- it was fixed in commit `5140f3a`.)_

## Recommendation

- Merge: YES -- the signal handling fix correctly addresses both Codex ACTION_REQUIRED findings. No CRITICAL or HIGH issues found. The two MEDIUM findings are a stale debt entry (easily cleaned) and a design note about signal re-raising (acceptable for current callers). The two LOW findings are test coverage gaps and a pre-existing exit code collision, neither of which block the merge.
- Follow-ups:
  1. Remove or mark as resolved the stale tech debt entry about `orchestrator.tmp.json` PID-suffixing.
  2. Add a test that asserts `orchestrator.json` status is NOT overwritten to "interrupted" on normal (non-signal) exit.
  3. Consider distinct return codes for `gh_unavailable` vs. codex `ACTION_REQUIRED` in `run_outer_loop` to prevent wasted cycles.
