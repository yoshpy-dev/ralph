# Verify report: Ralph Pipeline Hardening v3 (post-Codex Round 3 re-verify)

- Date: 2026-04-15
- Plan: Ralph Loop pipeline hardening (feat/ralph-pipeline-hardening)
- Verifier: verifier subagent (claude-opus-4-6)
- Scope: Re-verification after 3 rounds of Codex fixes. All 8 changed scripts + 3 test files. Delta since v2: commits b951bd0, 988bf57, 9b0ebb8.
- Evidence: `docs/evidence/verify-2026-04-15-ralph-pipeline-hardening-v3.log`

## Spec compliance

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | Shared config module `scripts/ralph-config.sh` exists with 9 env vars and `validate_numeric` | PASS | File exists (57 lines). 9 `RALPH_*` env vars with `${VAR:-default}` pattern. `validate_numeric()` at line 34, `validate_all_numeric()` at line 50. No dead code remains (ralph_claude_flags removed in earlier commit). |
| AC2 | No hardcoded `--model opus`, `--effort high`, `--effort max` in scripts/ | PASS | `grep --model opus --effort max --effort high scripts/` returns zero matches. All `claude -p` invocations use `"$RALPH_MODEL"`, `"$RALPH_EFFORT"`, `"$RALPH_PERMISSION_MODE"` variables sourced from ralph-config.sh. |
| AC3 | Signal handlers (`trap INT TERM EXIT`) in ralph-orchestrator.sh | PASS | Now fully committed. Separate traps: `trap _on_signal INT TERM` at line 115, `trap cleanup_on_exit EXIT` at line 116. `_on_signal()` sets `_INTERRUPTED=1` then `exit 1` (lines 83-86). `cleanup_on_exit()` kills .pid file PIDs and only writes "interrupted" status when `_INTERRUPTED=1` (line 106). ralph-loop.sh has `trap _loop_interrupted INT TERM` at line 81. ralph-pipeline.sh has `trap _pipeline_cleanup EXIT` at line 80. v2 "uncommitted" caveat is resolved. |
| AC4 | Race condition fixes: PID-suffixed temp files, whitespace trimming on `.status` reads | PASS | PID-suffixed temp files (`$$`): lines 109, 768, 933 in orchestrator. Whitespace trimming (`tr -d '[:space:]'`): lines 411, 419, 837, 843 in orchestrator. |
| AC5 | `gh` CLI preflight probe added | PASS | Probe 6 in `run_preflight()` at lines 345-353: checks `command -v gh`, logs warning if unavailable, records result in preflight JSON. Also guarded before PR creation at line 521 (orchestrator) and line 775 (pipeline). `gh_unavailable` now returns exit code 2, handled as terminal status in `main()` (line 986-989). |
| AC6 | Slice timeout detection in orchestrator polling loop | PASS | `RALPH_SLICE_TIMEOUT` config var (default 1800s). Timeout check at lines 835-850: reads `slice-*.started` epoch, computes elapsed, kills PID and writes "timeout" status if exceeded. "timeout" recognized as terminal status in both skip-launch (line 762) and failure-count (line 832) patterns. "timeout" now also displays correctly in status helpers (lines 113, 168, 408). |
| AC7 | All tests pass (test-ralph-config.sh, test-ralph-signals.sh, test-ralph-status.sh) | LIKELY BUT UNVERIFIED | All three test files exist, are executable, and pass `sh -n` syntax check. Test execution is /test responsibility. |
| AC8 | `validate_all_numeric` called at startup in all scripts | PASS | Called at: ralph-pipeline.sh:68, ralph-loop.sh:52, ralph-orchestrator.sh:54. The `scripts/ralph` CLI wrapper sources ralph-config.sh but delegates to orchestrator/pipeline which perform validation. Known-acceptable pattern. |

## v3-specific fix verification

| Fix (Codex Round) | Status | Evidence |
| --- | --- | --- |
| `gh_unavailable` added to terminal status patterns (Round 2, commit b951bd0) | PASS | Present in: orchestrator skip-launch (line 762), orchestrator failure-count (line 832), status-helpers resolve_display_phase (line 113), status-helpers status_icon (line 168). Without this fix, a `gh_unavailable` slice would be relaunched indefinitely. |
| `run_outer_loop` returns exit code 2 for `gh_unavailable` (Round 2, commit b951bd0) | PASS | Line 778: `return 2`. `main()` case handler at lines 986-989 calls `_finalize "gh_unavailable"` and `return 0`. This prevents futile Inner Loop regression that occurred when both `gh_unavailable` and `ACTION_REQUIRED` returned 1. |
| Dead `_CHILD_PIDS`/`register_child` removed (Round 3, commit 9b0ebb8) | PASS | `grep '_CHILD_PIDS\|register_child' scripts/` returns only one hit: a comment at line 90 referencing `_CHILD_PIDS` in a docstring. No variable declaration, no function definition, no call sites remain. The v3 self-review MEDIUM finding has been addressed. |
| `timeout` added to status display patterns (Round 3, commit 988bf57) | PASS | Present in: `resolve_display_phase` (line 113), `status_icon` (line 168), `_render_table` detail column (line 408). Without this fix, timed-out slices would display as unknown/blank instead of red failures. |
| `cleanup_on_exit` PID-reuse risk resolved (Round 3, commit 988bf57) | PASS | `cleanup_on_exit` no longer iterates `_CHILD_PIDS`. Uses .pid files exclusively (lines 94-103). .pid files are deleted when slices complete, eliminating the PID-reuse risk on long-running sessions. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n` on all 9 files | PASS (9/9) | All shell scripts and tests pass POSIX syntax check |
| `shellcheck` | NOT AVAILABLE | Not installed on this macOS environment |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | EXIT 0 | "docs or scaffold-level work only" -- expected for shell-script changes when no shell verifier is configured |
| File permissions check | PASS | All 9 scripts and tests have executable bit set (rwxr-xr-x) |
| Hardcoded flag grep | PASS | Zero matches for `--model opus`, `--effort max`, `--effort high` across `scripts/` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | Yes | No behavioral changes to skill definitions or pipeline order |
| `AGENTS.md` | Yes | Minor update: added `ralph-config.sh` to repo map line. Accurate. |
| `README.md` | Yes | Safety rails description updated to include slice timeout, signal handlers, and configurable settings. Accurate. |
| `.claude/rules/post-implementation-pipeline.md` | Yes | Pipeline order unchanged |
| `.claude/rules/subagent-policy.md` | Yes | References to `claude -p` invocations remain valid (now via config vars) |
| `docs/recipes/ralph-loop.md` | Yes | Configuration table lists all 9 RALPH_* vars. Safety rails table includes signal handling, slice timeout, and numeric validation. All entries match ralph-config.sh defaults. |
| `docs/tech-debt/README.md` | STALE | Entry 1 says `gh_unavailable` and `ACTION_REQUIRED` "both return 1" -- this was fixed in commit b951bd0. `run_outer_loop` now returns 2 for `gh_unavailable` and 1 for `ACTION_REQUIRED`, with separate handlers in `main()`. Entry should be removed or marked resolved. |
| `scripts/ralph-orchestrator.sh` lines 90-91 | MINOR DRIFT | Comment references `_CHILD_PIDS` which was removed in commit 9b0ebb8. The comment explains why `_CHILD_PIDS` is not used, but the construct no longer exists. Harmless but mildly misleading to future readers. |
| `docs/reports/self-review-2026-04-15-ralph-pipeline-hardening-v2.md` | STALE (expected) | References uncommitted signal handling code that is now committed. This is a point-in-time snapshot, not a living doc. Acceptable. |

## Observational checks

1. **Signal handling is correctly committed and structured.** The `_on_signal`/`_INTERRUPTED`/`cleanup_on_exit` pattern at lines 80-116 is the standard POSIX two-trap pattern. `_on_signal` sets the flag and calls `exit 1`, which triggers the EXIT trap. `cleanup_on_exit` only overwrites status to "interrupted" when `_INTERRUPTED=1`. The v2 "uncommitted" caveat is fully resolved -- all signal handling code is committed (commit 23a9e8a).

2. **Exit code collision is resolved.** `run_outer_loop` now returns 2 for `gh_unavailable` (line 778) and 1 for `ACTION_REQUIRED` (line 754). `main()` handles case 1 (lines 981-984) by regressing to Inner Loop and case 2 (lines 986-989) by finalizing as `gh_unavailable`. This eliminates the wasted Inner Loop cycles that occurred when both returned 1.

3. **Dead code fully removed.** The `_CHILD_PIDS` variable, `register_child()` function, and its call site are gone (commit 9b0ebb8). Only a comment reference at line 90-91 remains, which is a minor documentation cleanup opportunity.

4. **Terminal status consistency across all pattern sites.** The full terminal status set (`failed|stuck|repair_limit|aborted|config_error|gh_unavailable|timeout|max_iterations|max_inner_cycles|max_outer_cycles`) appears consistently in:
   - Orchestrator skip-launch (line 762)
   - Orchestrator failure-count (line 832)
   - Status helpers `resolve_display_phase` (line 113)
   - Status helpers `status_icon` (line 168, uses `max_*` glob)
   - Pipeline `_finalize` covers: complete, aborted, stuck, repair_limit, config_error, gh_unavailable, max_iterations, max_inner_cycles, max_outer_cycles

5. **Config sourcing chain intact.** All 4 consumer scripts source `ralph-config.sh` at startup. `validate_all_numeric()` called in 3 of 4 (pipeline, loop, orchestrator). The CLI wrapper delegates, so validation happens in the delegated script.

## Coverage gaps

1. **shellcheck not available.** Not installed on this macOS environment. Would strengthen confidence if run in CI.

2. **`test-ralph-signals.sh` does not directly test the `_INTERRUPTED` flag gating.** Same gap as v2. The test at lines 163-174 simulates `cleanup_on_exit` behavior but uses `jq` directly, not the actual function with the `_INTERRUPTED` guard. Tracked in `docs/tech-debt/README.md` entry 2.

3. **`timeout` status has no mock in `test-ralph-status.sh`.** The status display test covers `complete`, `running`, `pending`, `failed` but not `timeout`. Identified in v3 self-review as LOW severity.

4. **Tech debt entry 1 is stale.** The exit code collision is resolved but `docs/tech-debt/README.md` still describes the old behavior. Should be removed or marked resolved.

5. **Stale comment in orchestrator.** Lines 90-91 reference `_CHILD_PIDS` which was removed. Harmless but should be cleaned up.

6. **Runtime behavior of `_on_signal` + `exit 1` triggering EXIT trap.** Well-defined POSIX behavior, but has not been runtime-tested in this verification.

## Verdict

- **Verified (static + spec compliance)**: AC1, AC2, AC3, AC4, AC5, AC6, AC8. All v3 Codex fixes (gh_unavailable terminal status, exit code 2 handler, dead code removal, timeout display, PID-reuse fix) are correct and committed.
- **Likely but unverified (requires /test)**: AC7 (test execution), runtime validation of signal handler chain, timeout status mock coverage.
- **Not verified**: shellcheck analysis (tool not installed).

### Remaining action items

1. **Update or remove stale tech debt entry 1** in `docs/tech-debt/README.md`. The exit code collision is resolved -- `gh_unavailable` now returns 2, not 1.
2. **Clean up stale comment** at `scripts/ralph-orchestrator.sh` lines 90-91 that references removed `_CHILD_PIDS`.

Neither item blocks merge. Both are minor cleanup tasks.

### Confidence assessment

High confidence for merge. All 8 acceptance criteria are either PASS (7) or LIKELY BUT UNVERIFIED (1, test execution). All 5 v3-specific fixes are verified correct. No CRITICAL or HIGH issues. Two minor documentation drift items identified (stale tech debt entry, stale comment).
