# Self-review report: Ralph Pipeline Hardening

- Date: 2026-04-15
- Plan: feat/ralph-pipeline-hardening
- Reviewer: reviewer subagent (claude-opus-4-6)
- Scope: Diff quality of 8 changed files (scripts/ralph-config.sh, ralph-pipeline.sh, ralph-orchestrator.sh, ralph-loop.sh, ralph, tests/test-ralph-config.sh, tests/test-ralph-signals.sh, tests/test-ralph-status.sh) -- 653 additions, 27 deletions

## Evidence reviewed

- Full `git diff main...HEAD` (all 8 files)
- Current state of all modified scripts (ralph-config.sh, ralph-pipeline.sh, ralph-orchestrator.sh, ralph-loop.sh, scripts/ralph)
- Cross-reference of `ralph_claude_flags()` and `validate_all_numeric()` callsites
- Cross-reference of all `claude -p` invocation sites for flag consistency
- `tr -d '[:space:]'` behavior on status strings and PIDs
- Temp file naming (`orchestrator.tmp.json`) for race potential

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `ralph_claude_flags()` defined in `ralph-config.sh` (line 61) but never called by any production script. Only referenced in `tests/test-ralph-config.sh`. All 4 consumer scripts inline the flags directly (`--model "$RALPH_MODEL" --effort "$RALPH_EFFORT" --permission-mode "$RALPH_PERMISSION_MODE"`). | `grep -r ralph_claude_flags` returns only config definition + test file. `ralph-pipeline.sh:143,156`, `ralph-loop.sh:112`, and preflight probes at `ralph-pipeline.sh:294,329` all inline the flags. | Either use `ralph_claude_flags` in the consumer scripts (replacing 6+ inline expansions) or remove it to avoid dead code. If kept as future API, add a comment stating it is intentionally unused in current scripts. |
| MEDIUM | maintainability | `validate_all_numeric()` defined in `ralph-config.sh` (line 50) but never called by any production script. Each script validates its own CLI args individually via `validate_numeric` at parse time. Only called in `tests/test-ralph-config.sh`. | `grep -r validate_all_numeric` returns only config definition + test file. | Same as above: either wire it into script startup (e.g., after sourcing config, call `validate_all_numeric` to catch bad env vars early) or remove. Currently, a user who sets `RALPH_MAX_ITERATIONS=abc` via environment (without CLI override) will not see a validation error in any script. |
| MEDIUM | null-safety | Preflight probe `claude -p` calls at `ralph-pipeline.sh:294` and `ralph-pipeline.sh:329` omit `--permission-mode "$RALPH_PERMISSION_MODE"` while all `run_claude()` invocations include it. This inconsistency means preflight probes run with a different permission mode than actual pipeline execution, potentially giving misleading probe results. | `ralph-pipeline.sh:294`: `claude -p --model "$RALPH_MODEL" --effort "$RALPH_EFFORT" --output-format text` (no `--permission-mode`). Compare with `ralph-pipeline.sh:143`: full flags including `--permission-mode "$RALPH_PERMISSION_MODE"`. | Add `--permission-mode "$RALPH_PERMISSION_MODE"` to both preflight probe invocations for consistency, or document the intentional omission. |
| MEDIUM | maintainability | `orchestrator.tmp.json` temp file is not PID-suffixed, creating a potential race if the cleanup trap (`cleanup_on_exit` at line 110) and the normal status update at the end of `main()` (line 935) both write to `${ORCH_STATE}/orchestrator.tmp.json` concurrently. The `.deps_check` temp file was correctly changed to use `$$` suffix (line 770), but the `orchestrator.tmp.json` pattern was not updated. | `ralph-orchestrator.sh:111` and `ralph-orchestrator.sh:935` both write to `${ORCH_STATE}/orchestrator.tmp.json`. | Suffix with `$$` or use `mktemp` for both temp JSON files, consistent with the `.deps_check.$$.tmp` pattern already applied in this diff. |
| LOW | readability | `tr -d '[:space:]'` is a blunt instrument -- it removes ALL whitespace characters including internal spaces, tabs, and newlines. While safe for the current use cases (status strings like "running"/"complete" and numeric PIDs never contain internal spaces), a reader might wonder whether multi-word statuses could appear in the future. | `ralph-orchestrator.sh:413,421,839,845` all use `tr -d '[:space:]'`. | Consider using `tr -d ' \n\r\t'` (explicit characters) or `sed 's/^[[:space:]]*//;s/[[:space:]]*$//'` (trim only leading/trailing) to make intent clearer. This is stylistic -- current behavior is correct. |
| LOW | readability | The timeout check block (orchestrator lines 836-852) is deeply nested: while-loop > case > running branch > if started_file > if elapsed >= timeout > if pid_file. Six levels of nesting make this section harder to follow. | `ralph-orchestrator.sh:835-853` | Consider extracting the timeout check into a `check_slice_timeout()` function to flatten nesting and improve readability. |
| LOW | naming | `_vn_name` and `_vn_value` in `validate_numeric()` use a `_vn_` prefix for namespace safety, which is good practice. However, other helper functions in the same file do not follow this convention (e.g., `ralph_claude_flags` uses no prefix for locals). | `ralph-config.sh:34-46` vs `ralph-config.sh:61-62` | Minor inconsistency. `ralph_claude_flags` has no local variables that could collide, so this is acceptable. |

## Positive notes

- Extracting hardcoded CLI flags into `ralph-config.sh` with `${VAR:-default}` pattern is clean and follows POSIX sh idioms correctly.
- PID-suffixed temp files (`$$.tmp`) for `.deps_check` properly address the documented race condition.
- Signal handlers (`cleanup_on_exit` in orchestrator, `_loop_interrupted` in loop) correctly update state files before exit, enabling clean resume.
- The `gh` CLI preflight probe (pipeline probe 6) and the guard before PR creation are well-placed -- fail-fast at detection time rather than deep in the PR creation flow.
- `schema_version: 1` addition to `orchestrator.json` is forward-thinking for schema evolution.
- Ambiguous/missing dependency warnings (orchestrator lines 785-789) are a real improvement over silent prefix-match behavior.
- Test files follow the existing test harness conventions (`assert_eq`, `assert_exits_nonzero`, `_pass/_fail/_total` counters) consistently.
- The cleanup trap in `ralph-pipeline.sh` correctly targets only dotfile temp prompts, avoiding accidental deletion of log artifacts.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `ralph_claude_flags()` and `validate_all_numeric()` are dead code in production | Unused code increases maintenance surface. Readers must trace callsites to confirm they are intentional. | They serve as API for potential future use and are tested. | Next change to ralph-config.sh or consumer scripts. | This report |
| Preflight probes omit `--permission-mode` flag | Probe results may not reflect actual pipeline execution environment. | Preflight probes are lightweight capability checks; permission mode is unlikely to affect probe success. | If permission-mode-related failures are observed in pipeline but not in preflight. | This report |
| `orchestrator.tmp.json` not PID-suffixed | Potential race between cleanup trap and normal exit path writing the same temp file. | In practice the trap and normal path are unlikely to race because the trap fires after main() returns, but on SIGINT during the final jq call it could corrupt. | If orchestrator.json corruption is observed after interrupt. | This report |
| Env-only config not validated at startup | A user setting `RALPH_MAX_ITERATIONS=abc` via env (without CLI override) gets no error until arithmetic evaluation fails later with a cryptic message. | `validate_all_numeric` exists but is not wired in. | Wire `validate_all_numeric` into each script's startup, after sourcing config. | This report |

## Recommendation

- Merge: YES -- no CRITICAL or HIGH findings. All MEDIUM findings are maintainability/consistency issues that do not affect correctness in the current usage patterns.
- Follow-ups:
  1. Wire `validate_all_numeric()` into each script startup or remove it (addresses env-only validation gap).
  2. Add `--permission-mode` to preflight probe `claude -p` calls for flag consistency.
  3. PID-suffix `orchestrator.tmp.json` to match the `.deps_check.$$.tmp` pattern.
  4. Decide whether `ralph_claude_flags()` should be used by consumer scripts or removed.
