# Verify report: Ralph Pipeline Hardening

- Date: 2026-04-15
- Plan: Ralph Loop pipeline hardening (feat/ralph-pipeline-hardening)
- Verifier: verifier subagent (static analysis + spec compliance)
- Scope: scripts/ralph-config.sh, scripts/ralph-pipeline.sh, scripts/ralph-orchestrator.sh, scripts/ralph-loop.sh, scripts/ralph, tests/test-ralph-config.sh, tests/test-ralph-signals.sh, tests/test-ralph-status.sh
- Evidence: `docs/evidence/verify-2026-04-15-ralph-pipeline-hardening.log`

## Spec compliance

| # | Acceptance criterion | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | Shared config module `scripts/ralph-config.sh` exists with 9 env vars and `validate_numeric` | PASS | File exists (57 lines). 9 `RALPH_*` env vars with `${VAR:-default}` pattern. `validate_numeric()` at line 34, `validate_all_numeric()` at line 50. |
| AC2 | No hardcoded `--model opus`, `--effort high`, `--effort max` in ralph-pipeline.sh, ralph-loop.sh | PASS | `grep -rn` across `scripts/` returns zero matches. All 5 `claude -p` invocations use `"$RALPH_MODEL"`, `"$RALPH_EFFORT"`, `"$RALPH_PERMISSION_MODE"` variables. |
| AC3 | Signal handlers (`trap INT TERM EXIT`) in ralph-orchestrator.sh | PASS | `trap cleanup_on_exit INT TERM EXIT` at line 119. `cleanup_on_exit()` kills tracked child PIDs, kills PIDs from state files, updates orchestrator.json to "interrupted". ralph-loop.sh also has `trap _loop_interrupted INT TERM` at line 81. ralph-pipeline.sh has `trap _pipeline_cleanup EXIT` at line 80. |
| AC4 | Race condition fixes: PID-suffixed temp files, whitespace trimming on `.status` reads | PASS | PID-suffixed temp files (`$$`): lines 113, 772, 937-938 in orchestrator. Whitespace trimming (`tr -d '[:space:]'`): lines 415, 423, 841, 847 in orchestrator. |
| AC5 | `gh` CLI preflight probe added | PASS | Probe 6 in `run_preflight()` at lines 345-353: checks `command -v gh`, logs warning if unavailable, records result in preflight JSON. Also checked before PR creation at line 775. |
| AC6 | Slice timeout detection in orchestrator polling loop | PASS | `RALPH_SLICE_TIMEOUT` config var (default 1800s). Timeout check at lines 838-854: reads `slice-*.started` epoch, computes elapsed, kills PID and writes "timeout" status if exceeded. "timeout" recognized as terminal status in polling logic (lines 766, 836). |
| AC7 | All tests pass (test-ralph-config.sh, test-ralph-signals.sh, test-ralph-status.sh) | LIKELY BUT UNVERIFIED | All three test files exist, are executable, and pass `sh -n` syntax check. Test execution is /test responsibility. |
| AC8 | `validate_all_numeric` called at startup in all scripts | PASS | Called at: ralph-pipeline.sh:68, ralph-loop.sh:52, ralph-orchestrator.sh:57. The `scripts/ralph` CLI wrapper sources ralph-config.sh but delegates to orchestrator/pipeline which perform validation. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n` on all 8 changed files | PASS (8/8) | All shell scripts pass POSIX syntax check |
| `shellcheck` | NOT AVAILABLE | Not installed on this machine. Would strengthen confidence if run in CI. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | PASS (exit 0) | "docs or scaffold-level work only" -- expected for shell scripts |
| File permissions check | PASS | All scripts and tests are executable |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `CLAUDE.md` | Yes | No behavioral changes to skill definitions or pipeline order |
| `AGENTS.md` | Yes | No changes to repo map or pipeline contracts |
| `.claude/rules/post-implementation-pipeline.md` | Yes | Pipeline order unchanged |
| `.claude/rules/subagent-policy.md` | Yes | References to `claude -p` invocations remain valid (now via config vars) |
| `scripts/ralph-config.sh` header comments | Yes | Usage examples match actual sourcing pattern |
| Self-review report (`docs/reports/self-review-2026-04-15-ralph-pipeline-hardening.md`) | Exists | References the new config module correctly |

## Observational checks

1. **Config sourcing chain verified**: All 4 consumer scripts (`ralph`, `ralph-pipeline.sh`, `ralph-orchestrator.sh`, `ralph-loop.sh`) source `ralph-config.sh` at startup via `. "${SCRIPT_DIR}/ralph-config.sh"`.

2. **Signal handler coverage**: Three different trap strategies appropriate to each script's role:
   - orchestrator: full cleanup (kill children, update JSON) on INT/TERM/EXIT
   - loop: status update to "interrupted" on INT/TERM
   - pipeline: temp file cleanup on EXIT

3. **PID-suffixed temp files prevent clobbering**: Uses `$$` (current shell PID) to suffix temporary JSON files during atomic write-and-rename operations, preventing parallel orchestrator instances from corrupting each other's state.

4. **Whitespace trimming is defensive**: All `.status` and `.pid` file reads in the orchestrator polling loop use `tr -d '[:space:]'`, preventing trailing newlines or spaces from causing string comparison failures.

5. **Test coverage for new features**: `test-ralph-config.sh` covers defaults (9 vars), env override (4 vars), numeric validation (positive/negative/edge), and `validate_all_numeric`. `test-ralph-signals.sh` covers SIGINT cleanup and orchestrator status update. `test-ralph-status.sh` adds whitespace trimming tests.

## Coverage gaps

1. **shellcheck**: Not available on this machine. Recommend adding to CI for comprehensive static analysis of all shell scripts.

2. **Runtime behavior of timeout kill**: The timeout logic in the polling loop looks correct structurally, but verifying that `kill "$_spid"` reliably terminates the subshell tree requires runtime testing (process group semantics).

3. **`scripts/ralph` does not call `validate_all_numeric` directly**: It sources ralph-config.sh but does not validate at the wrapper level. Validation happens in the delegated scripts (orchestrator, pipeline). This is acceptable design but means invalid env vars would not be caught until the inner script starts.

4. **Codex advisory**: Not available on this machine.

## Verdict

- **Verified (static + spec compliance)**: AC1, AC2, AC3, AC4, AC5, AC6, AC8
- **Likely but unverified (requires /test)**: AC7 (test execution), runtime timeout kill behavior, process group cleanup under SIGTERM
- **Not verified**: shellcheck analysis (tool not installed), codex review (tool not available)
