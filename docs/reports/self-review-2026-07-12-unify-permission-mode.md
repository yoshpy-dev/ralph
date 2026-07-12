# Self-Review: unify-permission-mode

- Date: 2026-07-12
- Plan: `docs/plans/active/2026-07-12-unify-permission-mode.md`
- Branch: `fix/unify-permission-mode-default` (base `main`)
- Diff reviewed: `git diff main...HEAD` — 13 files, +477/-29 (of which the plan file is +184; implementation commit `1aead8f` is the substantive change)
- Scope: **diff quality only** (naming, readability, correctness of the priority logic, test quality, doc-claim accuracy, mirror byte-identity, secrets, exception handling). No spec-compliance, no test-coverage judgment, no doc-drift audit — those belong to `/verify` and `/test`.

## Verdict

**MERGE** — no CRITICAL or HIGH findings. The diff is clean, well-commented, reuses the existing `appendEnvIfMissing` helper rather than inventing new machinery, and the tests genuinely pin the env>toml>default contract end-to-end (verified by executing them). Findings are 2 LOW / informational only.

## Counts

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH | 0 |
| MEDIUM | 0 |
| LOW | 2 |

CRITICAL findings: **No.**

## What was checked (evidence)

### 1. run.go priority logic — CORRECT

- `newRunCmd` now threads `cmd.Flags().Changed("max-iterations")` / `("max-parallel")` into `runPipeline` as two booleans (`internal/cli/run.go:34-36`). This is the Cobra flag-presence check the plan mandates (advisory finding 4), not the old `!= 0` heuristic — so `--max-iterations 0` is now distinguishable from "flag absent".
- MODEL/EFFORT/PERMISSION_MODE changed from unconditional `append` to `appendEnvIfMissing` (`run.go:62-64`), so a pre-set env var wins over the TOML value. This makes them consistent with the loop/phase vars that already used `appendEnvIfMissing` (lines 86-109). Before this change these three were the only exports that let TOML beat env — the fix removes that inconsistency.
- MAX_* honour CLI > env > TOML: flag present → unconditional `append` (CLI wins); flag absent → `appendEnvIfMissing` from TOML (env wins) (`run.go:73-80`). Correct.
- `appendEnvIfMissing` (`run.go:248-256`) treats present-but-empty (`KEY=`) as present via exact `prefix` match — the "non-empty env wins" contract. Unchanged helper, correctly reused.
- **No collateral damage to untouched exports**: `RALPH_LOOP_DRIVER`, `RALPH_CODEX_*`, `RALPH_CLAUDE_REVIEWER_MODEL`, all 8 phase-model vars, and `RALPH_FORCE_MODEL` still use `appendEnvIfMissing` exactly as before (`run.go:86-109`). No `RALPH_PROMPTS` export exists in this file and none was added/removed. Diff is surgical.

### 2. Test quality — tests pin the contract, not the implementation

I executed the tests rather than reading them only.

- `TestRunPipeline_EnvWinsOverTomlForModelEtc`: sets TOML to distinct values, pre-sets env to different values, asserts the env values survive AND asserts exactly one entry per key. Reads the orchestrator stub's dumped `env.txt` (the resolved child env), not the raw Go slice — the right layer.
- `TestRunPipeline_MaxIterFlagBeatsEnv`: I verified independently that Go's `exec.Cmd.Env` with duplicate keys collapses to **last-wins** in the child's `env` output (`FOO=99` then `FOO=5` → child sees `FOO=5`). So reading `env.txt` genuinely validates ordering: if the implementation appended the CLI value *before* the env var, the test would read `99` and fail. The test is sound.
- `TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag` and `TestRunPipeline_EmptyPermissionModeEnv` (Go side of the empty-env contract) both assert the negative (toml value must NOT appear) in addition to the positive — good discrimination.
- `tests/test-ralph-config.sh::test_empty_env`: sources `scripts/ralph-config.sh` directly and asserts `${RALPH_PERMISSION_MODE:-bypassPermissions}` resolves empty → `bypassPermissions` and `auto` → `auto`. This is the shell-layer half of the end-to-end contract the plan requires. **Ran it: 43/43 pass**, including both new assertions.
- `internal/cli` + `internal/config` Go tests: **pass**.
- `TestLoad_TemplateRalphToml` now also asserts `template permission_mode == Default().PermissionMode` (advisory finding 5), so template/Go/shell cannot silently drift again.

### 3. Doc-claim accuracy — VERIFIED against actual behavior

- toml comment (`templates/base/ralph.toml:9-19`) correctly states interactive hooks do **NOT** fire under `claude -p` and are **not** part of the safety story — this is the corrected guardrail wording the plan flagged as load-bearing (advisory finding 1). The claim is accurate; it does not overclaim hook protection.
- Recipe row (`docs/recipes/ralph-loop.md:141`) claims `bypassPermissions` is the default "across all entry points (shell scripts, `./scripts/ralph run`, and the Go `ralph run` binary)". I verified `scripts/ralph` sources `ralph-config.sh` at line 14, so `./scripts/ralph run` gets the shell default `bypassPermissions` — claim accurate. The env>toml>Go-path nuance ("shell wrappers do not parse ralph.toml") matches reality.
- README.md / .codex/README.md permission-policy rows updated from `auto` to `bypassPermissions` with the conservative-override note (advisory finding 6). Accurate.

### 4. Mirror byte-identity — PASS

- `cmp .codex/README.md templates/base/.codex/README.md` → identical.
- `cmp docs/recipes/ralph-loop.md templates/base/docs/recipes/ralph-loop.md` → identical.

### 5. Secrets / exception handling / debug code

- No hardcoded secrets, tokens, or credentials.
- `config.Load` error is handled with a warning + defaults fallback (`run.go:52-54`) — unchanged, appropriate for this CLI (defaults are safe).
- No leftover debug prints, TODOs, or commented-out code in the implementation.
- Commit message uses conventional format; no shell-substitution risk.

## Findings

### LOW-1 — `containsKV` is an "any occurrence" check, weaker than the child-env resolution it stands in for

`containsKV` (`internal/cli/run_env_test.go:54-62`) returns true if *any* slice entry matches. In the MAX_* flag tests the env slice legitimately contains two entries for the same key (inherited env + appended CLI value). The tests are still correct **only because** they read from the stub's `env.txt` (child-resolved, last-wins) rather than passing the raw Go slice to `containsKV`. This is a latent trap: a future test that calls `containsKV` on the pre-exec Go slice for a duplicated key would get a false PASS. The current tests do not do this, so no bug today.

- Evidence: `run_env_test.go:54-62`, `TestRunPipeline_MaxIterFlagBeatsEnv` reads `readEnvLines()` (from `env.txt`), so it is safe.
- Suggested follow-up (optional): add a one-line doc comment on `containsKV` noting it must be used against child-resolved env (`env.txt`), not the raw `os.Environ()`-derived slice.

### LOW-2 — `setupEnvStub` cleanup restores cwd but not a leaked stub dir handle (cosmetic)

`setupEnvStub` (`run_env_test.go`) uses `t.TempDir()` (auto-cleaned by the test framework) and restores cwd in the deferred closure. This is fine. Minor: the helper `os.Chdir`es the whole test process, so these tests cannot run in parallel with each other — none call `t.Parallel()`, so no actual issue, but worth a comment if parallelism is ever added. Informational only.

## Tech-debt register

The permission-mode divergence row in `docs/tech-debt/README.md:40-41` is correctly marked RESOLVED using the established double-annotation convention (HTML comment + inline `~~strikethrough~~ (RESOLVED …)`), matching rows 31-32 and 42-43. Row preserved for traceability per convention. No new tech-debt introduced by this diff.

## Follow-ups (non-blocking)

1. (LOW-1) Optionally document `containsKV`'s "any occurrence" semantics so it is not misused on a pre-exec slice with duplicate keys.
2. (LOW-2) Optionally note the `os.Chdir` non-parallel constraint on `setupEnvStub`.

Neither blocks merge.
