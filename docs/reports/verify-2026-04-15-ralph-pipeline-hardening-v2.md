# Verify report: Ralph Pipeline Hardening v2 (post-Codex fix re-verify)

- Date: 2026-04-15
- Plan: Ralph Loop pipeline hardening (feat/ralph-pipeline-hardening)
- Verifier: verifier subagent (claude-opus-4-6)
- Scope: Re-verification after 2 Codex ACTION_REQUIRED fixes. All 8 changed files.
- Evidence: `docs/evidence/verify-2026-04-15-ralph-pipeline-hardening-v2.log`

## Spec compliance

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | Shared config module `scripts/ralph-config.sh` exists with 9 env vars and `validate_numeric` | PASS | File exists (57 lines). 9 `RALPH_*` env vars with `${VAR:-default}` pattern. `validate_numeric()` at line 34, `validate_all_numeric()` at line 50. `ralph_claude_flags()` dead code removed in self-review fix commit. |
| AC2 | No hardcoded `--model opus`, `--effort high`, `--effort max` in ralph-pipeline.sh, ralph-loop.sh | PASS | `grep --model opus --effort max --effort high scripts/` returns zero matches. All `claude -p` invocations (including both preflight probes) now use `"$RALPH_MODEL"`, `"$RALPH_EFFORT"`, `"$RALPH_PERMISSION_MODE"` variables. |
| AC3 | Signal handlers (`trap INT TERM EXIT`) in ralph-orchestrator.sh | PASS (with caveat) | Separate traps: `trap _on_signal INT TERM` at line 127, `trap cleanup_on_exit EXIT` at line 128. `_on_signal()` sets `_INTERRUPTED=1` then `exit 1`, triggering EXIT trap. `cleanup_on_exit()` kills tracked children, kills state-file PIDs, and writes "interrupted" status only when `_INTERRUPTED=1`. ralph-loop.sh has `trap _loop_interrupted INT TERM` at line 81. ralph-pipeline.sh has `trap _pipeline_cleanup EXIT` at line 80. **Caveat: This fix is NOT YET COMMITTED (exists only in working tree).** |
| AC4 | Race condition fixes: PID-suffixed temp files, whitespace trimming on `.status` reads | PASS | PID-suffixed temp files (`$$`): lines 121, 781, 946 in orchestrator. Whitespace trimming (`tr -d '[:space:]'`): lines 424, 432, 850, 856 in orchestrator. |
| AC5 | `gh` CLI preflight probe added | PASS | Probe 6 in `run_preflight()` at lines 345-353: checks `command -v gh`, logs warning if unavailable, records result in preflight JSON. Also guarded before PR creation at line 534 (orchestrator) and line 775 (pipeline). |
| AC6 | Slice timeout detection in orchestrator polling loop | PASS | `RALPH_SLICE_TIMEOUT` config var (default 1800s). Timeout check at lines 848-863: reads `slice-*.started` epoch, computes elapsed, kills PID and writes "timeout" status if exceeded. "timeout" recognized as terminal status in polling logic (line 775) and status counting (line 845). |
| AC7 | All tests pass (test-ralph-config.sh, test-ralph-signals.sh, test-ralph-status.sh) | LIKELY BUT UNVERIFIED | All three test files exist, are executable, and pass `sh -n` syntax check. `test_claude_flags` section removed from test-ralph-config.sh (matching removal of dead code). Test execution is /test responsibility. |
| AC8 | `validate_all_numeric` called at startup in all scripts | PASS | Called at: ralph-pipeline.sh:68, ralph-loop.sh:52, ralph-orchestrator.sh:57. The `scripts/ralph` CLI wrapper sources ralph-config.sh but delegates to orchestrator/pipeline which perform validation. This is the same known-acceptable pattern. |

## Codex fix verification

| Fix | Status | Evidence |
| --- | --- | --- |
| INT/TERM trap calls `_on_signal()` which sets `_INTERRUPTED=1` and `exit 1` | PASS (uncommitted) | `_on_signal()` at line 91-94: sets flag, then `exit 1`. `trap _on_signal INT TERM` at line 127. The `exit 1` triggers the EXIT trap, which performs cleanup. This correctly ensures the orchestrator terminates on signals rather than continuing execution. |
| EXIT trap only writes "interrupted" status when `_INTERRUPTED=1` | PASS (uncommitted) | `cleanup_on_exit()` at line 118: `if [ "$_INTERRUPTED" -eq 1 ]`. On normal non-zero exits (e.g., `main()` returns 1 for partial failure), `_INTERRUPTED` remains 0, so the EXIT trap does NOT overwrite the "partial" status written by `main()` at line 943. This correctly preserves the distinction between signal-interrupted and normal-failure exits. |
| "partial" status preserved for normal failures | PASS (uncommitted) | `main()` writes `"partial"` at line 943 when `_failed > 0`. Since `cleanup_on_exit` only overwrites status when `_INTERRUPTED=1`, and `_INTERRUPTED` stays 0 on normal `return 1`, the "partial" status survives the EXIT trap. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n` on all 8 changed files | PASS (8/8) | All shell scripts pass POSIX syntax check (including uncommitted orchestrator changes) |
| `shellcheck` | NOT AVAILABLE | Not installed on this macOS environment. Would strengthen confidence if run in CI. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | EXIT 2 | "No verifier ran for code-like changes." -- no shell verifier is configured. Not a failure of the code under review. |
| File permissions check | PASS | All 8 scripts and tests have executable bit set (rwxr-xr-x) |
| Hardcoded flag grep | PASS | Zero matches for `--model opus`, `--effort max`, `--effort high` across `scripts/` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | Yes | No behavioral changes to skill definitions or pipeline order |
| `AGENTS.md` | Yes | Minor update (+2 lines), no conflict with pipeline contracts |
| `.claude/rules/post-implementation-pipeline.md` | Yes | Pipeline order unchanged |
| `.claude/rules/subagent-policy.md` | Yes | References to `claude -p` invocations remain valid (now via config vars) |
| `docs/recipes/ralph-loop.md` | Yes | Configuration table lists all 9 RALPH_* vars. Safety rails table includes signal handling, slice timeout, and numeric validation. |
| `docs/tech-debt/README.md` | STALE | Entry 1 says "`orchestrator.tmp.json` in `cleanup_on_exit` not PID-suffixed" but this is now fixed: line 121 uses `orchestrator.tmp.$$.json`. The entry should be removed or marked resolved. |
| `docs/reports/self-review-2026-04-15-ralph-pipeline-hardening.md` | STALE | References MEDIUM findings that have been addressed: (1) `ralph_claude_flags()` removed, (2) `validate_all_numeric()` wired in, (3) preflight `--permission-mode` added, (4) `orchestrator.tmp.json` PID-suffixed. Tech debt table at bottom also references resolved items. Acceptable: self-review reports are point-in-time snapshots, not living docs. |

## Observational checks

1. **Signal handling split is correct.** The two-trap pattern (`_on_signal` for INT/TERM, `cleanup_on_exit` for EXIT) is the standard POSIX approach for separating signal-interrupted cleanup from normal cleanup. The `_INTERRUPTED` flag bridges them safely because shell traps are synchronous -- `_on_signal` runs to completion before `exit 1` triggers the EXIT trap. There is no race condition between setting the flag and reading it.

2. **"partial" status flow is correct.** When some slices fail but others complete, `main()` at line 943 writes `"partial"` to `orchestrator.json`. Then `main` calls `return 1` (line 952), which triggers `exit 1` at the script level, which triggers the EXIT trap. Since `_INTERRUPTED=0` (no signal was received), `cleanup_on_exit` skips the status overwrite. The "partial" status survives.

3. **Previous `_exit_code=$?` bug confirmed.** In the committed code (before the Codex fix), `cleanup_on_exit` captures `_exit_code=$?` and overwrites status to "interrupted" whenever `_exit_code != 0`. This means normal `partial` failures would be incorrectly overwritten to "interrupted". The Codex fix correctly separates these cases.

4. **Config sourcing chain intact.** All 4 consumer scripts source `ralph-config.sh` at startup. `validate_all_numeric()` is called in 3 of 4 (pipeline, loop, orchestrator). The CLI wrapper delegates, so validation happens in the delegated script. No regression from the Codex fix.

5. **PID-suffixed temp files now consistent.** Both the cleanup trap (line 121) and normal exit path (line 946) use `orchestrator.tmp.$$.json`. The `.deps_check.$$.tmp` pattern (line 781) also uses `$$`. All three temp file sites are consistent.

6. **Test files unchanged by Codex fix.** The Codex fix only modifies `scripts/ralph-orchestrator.sh`. Test files `test-ralph-signals.sh`, `test-ralph-config.sh`, and `test-ralph-status.sh` remain as committed. `test-ralph-signals.sh` tests signal behavior via dry-run mode, which exercises the trap setup. The test at line 163-174 simulates what `cleanup_on_exit` does, confirming the "interrupted" status write, but does not directly test the `_INTERRUPTED` flag gating. This is a minor test gap.

## Coverage gaps

1. **Codex fix is NOT committed.** The `_on_signal` / `_INTERRUPTED` changes to `scripts/ralph-orchestrator.sh` exist only in the working tree. They must be committed before the PR is created.

2. **shellcheck not available.** Not installed on this machine. Would strengthen confidence if run in CI.

3. **`test-ralph-signals.sh` does not directly test the `_INTERRUPTED` flag gating.** The test at line 163-174 simulates `cleanup_on_exit` behavior but uses `jq` directly, not the actual function with the `_INTERRUPTED` guard. A test that verifies "normal non-zero exit does NOT write 'interrupted'" would increase confidence in the Codex fix. This is a /test responsibility.

4. **Tech debt entry is stale.** `docs/tech-debt/README.md` entry 1 about `orchestrator.tmp.json` not being PID-suffixed is now resolved but not updated.

5. **Runtime behavior of `_on_signal` + `exit 1` triggering EXIT trap.** The POSIX shell spec guarantees that `exit` from within a trap handler triggers the EXIT trap. This is well-defined behavior, but has not been runtime-tested in this verification.

## Verdict

- **Verified (static + spec compliance)**: AC1, AC2, AC4, AC5, AC6, AC8. Both Codex fixes are structurally correct and the logic is sound.
- **Verified with caveat (uncommitted)**: AC3 and both Codex fixes. The signal handler changes pass `sh -n` and are logically correct, but exist only in the working tree. **Must be committed before PR.**
- **Likely but unverified (requires /test)**: AC7 (test execution), runtime validation of `_on_signal` → EXIT trap chain, test for "partial status not overwritten on normal failure".
- **Not verified**: shellcheck analysis (tool not installed).

### Action required

1. **COMMIT the Codex fix.** `scripts/ralph-orchestrator.sh` has uncommitted changes containing the `_on_signal` / `_INTERRUPTED` pattern. This must be staged and committed before proceeding to PR.
2. **Update or remove the stale tech debt entry** in `docs/tech-debt/README.md` (entry 1 about `orchestrator.tmp.json`).
