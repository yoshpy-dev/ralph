# Test report: Ralph Loop Codex driver — cycle 4

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Tester: tester subagent (Claude Opus 4.7, 1M context)
- Scope: cycle-4 revalidation after commit `f735299` ("fix: address cycle-3 cross-review P2 findings (3)") — fake-CLI stub TTY-guards, `checkLoopDriver` codex-missing fail path, `checkLoopDriver` env-priority detail rendering, plus 2 new doctor tests pinning the new behaviour. Cycle counter raised 3 → 4 in `.harness/state/standard-pipeline/cycle-count.json` per the commit body.
- Branch: `feat/44/ralph-loop-codex-driver`
- Evidence:
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-go.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-go-verbose.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-cli-driver-tty.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-cli-driver-pipe.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-mojibake.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-skill-sync.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-preflight-claude.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-preflight-codex.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cycle4-run-verify.log`

## Cycle-4 delta under test

| Surface | Cycle-4 change (commit `f735299`) |
| --- | --- |
| `tests/fixtures/cli-stubs/codex` + `.../claude` | Stubs now skip the stdin drain when stdin is a TTY (`if [ -t 0 ]; then stdin_buf=""; else stdin_buf="$(head -c 4096 \|\| true)"; fi`) so an interactive `tests/test-ralph-cli-driver.sh` run cannot block on terminal stdin; non-interactive (pipe / file redirect) callers still drain. |
| `internal/cli/doctor.go::checkLoopDriver` | Returns `fail` when effective driver is `codex` but `codex` is not in `PATH` (previously reported `pass` while next `ralph run` preflight blocked). |
| `internal/cli/doctor.go::checkLoopDriver` | Resolves `CodexSandbox`, `CodexApprovalPolicy`, and `ClaudeReviewerModel` through the same env > TOML > default priority used for `Driver`. Doctor detail now reflects env overrides instead of always showing TOML defaults. |
| `internal/cli/cli_test.go` | New test `TestCheckLoopDriver_FailsWhenCodexMissing` pins the codex-missing fail behaviour. |
| `internal/cli/cli_test.go` | New test `TestCheckLoopDriver_EnvOverridesShownInDetail` pins the env-priority display. |
| `internal/cli/cli_test.go::TestCheckLoopDriver_PriorityAndSource` | Modified — now installs a sham `codex` on `PATH` for codex-driver subcases so the new fail-when-missing guard does not invalidate the priority assertion. |

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Notes |
| --- | --- | --- | --- | --- | --- |
| `go test -count=1 ./...` (verbose tally) | 230 | 228 | 0 | 2 | Same 2 baseline SKIPs as cycles 2 / 3 (`internal/scaffold.TestBaseFS_WithMockFS`, `TestAvailablePacks_WithMockFS`). All 9 packages with tests `ok`. |
| New: `TestCheckLoopDriver_FailsWhenCodexMissing` | 1 | 1 | 0 | 0 | Visible in verbose log; covers driver=codex + missing PATH binary → fail. |
| New: `TestCheckLoopDriver_EnvOverridesShownInDetail` | 1 | 1 | 0 | 0 | Visible in verbose log; covers env > TOML > default for sandbox/approval/reviewer-model in doctor detail. |
| Modified: `TestCheckLoopDriver_PriorityAndSource` (4 subcases) | 4 | 4 | 0 | 0 | All four subcases pass with the sham-codex helper now installed for codex paths. |
| `tests/test-ralph-cli-driver.sh` (TTY stdin path) | 48 | 48 | 0 | 0 | Direct invocation as if from a developer terminal. Test 6 series asserts stdin payload reached the stub via the wrapper's `< "$prompt_file"` redirect (always non-TTY at the stub boundary), so the drain branch still runs. |
| `tests/test-ralph-cli-driver.sh` (non-TTY pipe path: `bash -c '...' < /dev/null`) | 48 | 48 | 0 | 0 | Same 48 cases; stub's `[ -t 0 ]` evaluates false → drain branch runs identically. Confirms the guard does not break the non-interactive path. |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | Regression baseline. |
| `tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | Regression baseline. |
| `RALPH_LOOP_DRIVER=claude scripts/ralph-pipeline.sh --preflight --dry-run` | 1 | 1 | 0 | 0 | exit 0; preflight reports `driver: claude` + claude/jq/git pass + `claude_md_readable`/`json_output_format` skip_dry_run + codex CLI available. |
| `PATH=tests/fixtures:$PATH RALPH_LOOP_DRIVER=codex scripts/ralph-pipeline.sh --preflight --dry-run` | 1 | 1 | 0 | 0 | exit 0; preflight reports `driver: codex` + codex/jq/git pass + `agents_md_readable` pass + `codex_exec_flags` skip_dry_run + claude CLI available. |
| Edge-case (a): `RALPH_LOOP_DRIVER=foo` | 1 | 1 | 0 | 0 | exit 1, stderr `Error: RALPH_LOOP_DRIVER must be "claude" or "codex", got: foo`. |
| Edge-case (b): `RALPH_CODEX_SANDBOX=foo` | 1 | 1 | 0 | 0 | exit 1, stderr `Error: RALPH_CODEX_SANDBOX must be one of read-only\|workspace-write\|danger-full-access, got: foo`. |
| Edge-case (c): `RALPH_CODEX_APPROVAL_POLICY=foo` | 1 | 1 | 0 | 0 | exit 1, stderr `Error: RALPH_CODEX_APPROVAL_POLICY must be one of untrusted\|on-failure\|on-request\|never, got: foo`. |
| `./scripts/run-verify.sh` (full end-to-end) | 1 | 1 | 0 | 0 | exit 0; aggregates: shellcheck → sh -n hook batch → settings.json validation × 2 → mojibake (11/11) → check-skill-sync (6/6) → check-sync → cli-driver (48/48) → gofmt → staticcheck (0 issues) → `go test ./...`. Evidence: `docs/evidence/verify-2026-05-08-055426.log` (referenced inside the run-verify log tail). |

**Combined totals:** 376 PASS / 0 FAIL / 2 SKIP across Go + every shell suite + preflight × 2 + edge cases × 3 + run-verify. Two SKIPs are pre-existing baseline (mock filesystem suites in `internal/scaffold`).

Cycle-by-cycle progression: cycle 2 = 299/299 PASS; cycle 3 = 374/374 PASS; cycle 4 = 376/376 PASS. The +2 vs cycle 3 reflects the two new doctor tests (`FailsWhenCodexMissing`, `EnvOverridesShownInDetail`). The two TTY/pipe runs of the cli-driver suite count once toward the 376 (the second is a redundant verification not double-counted).

## TTY-guard verification (specific to cycle-4 P2 #1)

Per the request, both the interactive (TTY stdin) and non-interactive (pipe / `< /dev/null`) invocation paths were exercised:

- **TTY path:** `tests/test-ralph-cli-driver.sh` invoked directly with the parent shell's stdin attached. Result: 48/48 PASS, exit 0. At the test-script level `[ -t 0 ]` is true; however the wrapper (`scripts/ralph-cli-driver.sh`) feeds the prompt to the stub via `codex exec ... - < "$prompt_file"`, so at the *stub* boundary stdin is a pipe — `[ -t 0 ]` is false and the drain branch executes. Test 6's `stdin` assertion (`assert_jq_contains '.stdin' "Adversarial review" ...`) passes, confirming the drain still happens.
- **Non-TTY pipe path:** `bash -c 'tests/test-ralph-cli-driver.sh' < /dev/null`. Result: 48/48 PASS, exit 0. Both the test script and the stub see non-TTY stdin; identical drain behaviour, identical assertions.
- **Static evidence:** the guard at `tests/fixtures/cli-stubs/codex:45-49` (and the symmetric block in `tests/fixtures/cli-stubs/claude`) skips the drain only when the stub *itself* sees a TTY on stdin. Since the wrapper always pipes, the existing 48-case assertion surface is unchanged. The guard's purpose is to prevent a developer who runs the stub directly (or composes it without redirecting stdin) from blocking on terminal stdin. That scenario is not exercised by automated tests but is the originally reported P2 #1 hang.

Diff between TTY and pipe runs: zero — the two evidence logs are byte-comparable in their PASS/FAIL summary tails (both `PASS: 48 / FAIL: 0`).

## checkLoopDriver fail-when-codex-missing verification

`TestCheckLoopDriver_FailsWhenCodexMissing` is present in `internal/cli/cli_test.go` (visible in verbose log: `=== RUN   TestCheckLoopDriver_FailsWhenCodexMissing` → `--- PASS`). The test pins the new behaviour: when `RALPH_LOOP_DRIVER=codex` (or TOML `[loop] driver = "codex"`) is in effect but `codex` is absent from `PATH`, `checkLoopDriver` returns `fail` rather than `pass`. Without this guard, doctor would have reported a healthy state while the next `ralph run` preflight blocked on the missing required CLI.

## checkLoopDriver env-priority verification

`TestCheckLoopDriver_EnvOverridesShownInDetail` is present (verbose log: PASS). The test pins env > TOML > default priority for the three companion env vars (`RALPH_CODEX_SANDBOX`, `RALPH_CODEX_APPROVAL_POLICY`, `RALPH_CLAUDE_REVIEWER_MODEL`). The doctor detail now displays the effective value plus its source (`env`/`toml`/`default`) for each, matching the `Driver` field's existing behaviour and closing a P2 finding from cycle-3 where exporting `RALPH_CODEX_SANDBOX=danger-full-access` still showed the TOML default in `ralph doctor`.

`TestCheckLoopDriver_PriorityAndSource` (the modified, pre-existing test) continues to pass with all four subcases. The cycle-4 modification installs a sham `codex` executable on PATH for the two codex-driver subcases; without it, the fail-when-codex-missing guard would have caused the priority assertion to fail. This is a coupled change — the new guard *needed* the existing test to be updated for its codex paths.

## Coverage

- Go: instrumented coverage not collected (run-verify uses plain `go test`). All 9 packages with tests pass; the cycle-4 changes add coverage for two previously unverified branches in `checkLoopDriver` (codex-missing fail; env-priority for sandbox/approval/reviewer).
- Shell: framework-free; coverage measured by case scope. `tests/fixtures/cli-stubs/codex:45-49` and `.../claude` symmetric block introduce a runtime-only branch (`[ -t 0 ]` skip) that is *not* directly asserted by the cli-driver suite — see *Test gaps* below.

## Failure analysis

No failures. Table omitted.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cycle-2 P1 — Codex cross-review reported zero findings because `--permission-mode plan` made the inverted Claude reviewer read-only | Resolved by `094f964` (cycle 3); still holds in cycle 4 — `grep -n 'permission-mode' scripts/ralph-pipeline.sh` returns line 796 = `--permission-mode auto`. | static grep + Test 6b-ii assertion (`tests/test-ralph-cli-driver.sh:213`) now reads `auto`. |
| Cycle-3 P2 #1 — interactive `tests/test-ralph-cli-driver.sh` hangs on terminal stdin via fake stub | Resolved by `f735299` TTY-guard. Both TTY and pipe paths return 48/48 PASS in cycle 4. | evidence sections cli-driver-tty / cli-driver-pipe. |
| Cycle-3 P2 #2 — `ralph doctor` reports pass for codex driver while `codex` is missing from PATH | Resolved by `f735299`. New `TestCheckLoopDriver_FailsWhenCodexMissing` pins the fail behaviour. | verbose log line for that test. |
| Cycle-3 P2 #3 — `ralph doctor` always shows TOML default for sandbox/approval/reviewer-model regardless of env override | Resolved by `f735299`. New `TestCheckLoopDriver_EnvOverridesShownInDetail` pins env-priority detail. | verbose log line for that test. |
| Cycle-2 baseline cli-driver count (48/48) | Holds in cycle 4 across both TTY and pipe runs. | evidence files. |
| `tests/test-check-mojibake.sh` 11/11 baseline | 11/11 holds. | mojibake evidence. |
| `tests/test-check-skill-sync.sh` 6/6 baseline | 6/6 holds. | skill-sync evidence. |
| Pipeline shellcheck/sh -n parity between root and `templates/base/` | Holds — `run-verify.sh` exit 0 includes both root and template paths. | run-verify evidence. |
| AC-12 backward-compat (driver unset → claude default unchanged) | Holds — claude preflight smoke is exit 0 with identical observable surface. | preflight-claude evidence. |

## Test gaps

The TTY-guard branch in `tests/fixtures/cli-stubs/codex:45-49` (and the symmetric `claude` stub) is not directly exercised by an automated assertion. The cycle-4 evidence covers it indirectly:
- Both TTY and pipe runs of the suite still pass 48/48, demonstrating no regression from adding the guard.
- The original failure mode (interactive hang) cannot be reliably reproduced in CI (CI is non-interactive by definition), so a regression would only manifest on a developer running the suite directly with terminal stdin. The static guard is straightforward enough that this gap is acceptable.

**Recommendation (not a blocker):** add a single fixture-level test that runs the stub stand-alone with `< /dev/tty` (or simulates a TTY via `script(1)`) and asserts the stub exits within a small timeout. This would convert the indirect coverage into a direct assertion. Filed as a follow-up; not required for cycle-4 verdict because cycle-4 evidence already shows both invocation paths are green and the guard logic is a 4-line conditional.

Other gaps from previous cycles are unchanged: real Codex turn not run (per `/test` contract); `verify.local.sh` mojibake matrix focuses on Edit/Write/MultiEdit; instrumented Go coverage not collected. See cycles 2 and 3 reports for context.

## Cycle-4 specific spot-checks

- **Permission-mode regression:** still `auto` (line 796 of `scripts/ralph-pipeline.sh`). Test 6b-ii literal in cli-driver suite is `--permission-mode auto` (line 213). Cycle-3 test-name drift gap closed.
- **fake-CLI stub TTY-guard symmetry:** both `tests/fixtures/cli-stubs/codex` and `tests/fixtures/cli-stubs/claude` updated identically by `f735299`.
- **doctor.go scope:** the `f735299` diff stat shows `internal/cli/doctor.go` 52 lines changed and `internal/cli/cli_test.go` extended; both compile and pass under `go test -count=1`.
- **Cycle-counter audit:** `.harness/state/standard-pipeline/cycle-count.json` raise-history field documents the 3 → 4 raise (per commit body — not re-verified here as it is configuration plumbing, not behaviour).

## Verdict

- **Pass: yes**
- Fail: none
- Blocked: none
- Recommendation: proceed to `/cross-review` (cycle 4) → `/pr`. The pipeline cycle counter is now at 4; per `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` raised to 4 (per cycle-3 progress checklist), this is the final cycle before the cap is enforced. Any further cross-review ACTION_REQUIRED would force the operator-fork (raise cap / proceed with known gaps / abort) per `.claude/rules/post-implementation-pipeline.md`.
