# Test report: Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: docs/plans/active/2026-05-08-ralph-loop-codex-driver.md
- Branch: feat/44/ralph-loop-codex-driver
- Tester: Claude Code (tester subagent)
- Scope: behavioural test execution per the plan's "Test plan" section — Go unit tests, fake-CLI driver shell tests, preflight smokes for both drivers, three edge-case bashes, and the full `./scripts/run-verify.sh` aggregator.
- Evidence:
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-go.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-cli.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-preflight-claude.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-preflight-codex.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-edge-driver.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-edge-sandbox.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-edge-approval.log`
  - `docs/evidence/test-2026-05-08-ralph-loop-codex-driver-run-verify.log`
  - `docs/evidence/verify-2026-05-08-033738.log` (run-verify.sh native evidence)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test ./... -count=1` (all 12 packages, of which 9 carry tests) | 230 | 228 | 0 | 2 | ~25 s wall |
| `tests/test-ralph-cli-driver.sh` (fake-CLI integration; 6 sections covering AC-1, AC-3, AC-5) | 38 | 38 | 0 | 0 | <2 s |
| `tests/test-check-mojibake.sh` (regression) | 11 | 11 | 0 | 0 | <1 s |
| `tests/test-check-skill-sync.sh` (regression) | 6 | 6 | 0 | 0 | <1 s |
| `RALPH_LOOP_DRIVER=claude ./scripts/ralph-pipeline.sh --preflight --dry-run` | 1 (smoke) | 1 | 0 | 0 | <1 s |
| `RALPH_LOOP_DRIVER=codex  ./scripts/ralph-pipeline.sh --preflight --dry-run` | 1 (smoke) | 1 | 0 | 0 | <1 s |
| Edge: `RALPH_LOOP_DRIVER=foo` early-exit | 1 | 1 | 0 | 0 | <1 s |
| Edge: `RALPH_CODEX_SANDBOX=bogus` early-exit | 1 | 1 | 0 | 0 | <1 s |
| Edge: `RALPH_CODEX_APPROVAL_POLICY=asap` early-exit | 1 | 1 | 0 | 0 | <1 s |
| `./scripts/run-verify.sh` (full aggregator: shellcheck, sh -n, jq, sync gates, every shell test, gofmt, staticcheck, go test) | 1 (aggregate) | 1 | 0 | 0 | ~30 s wall |

Totals (excluding the aggregate row, which re-runs members above): **290 distinct test/assertion executions, 290 PASS, 0 FAIL, 2 SKIP.**

### Per-package Go breakdown

| Package | Result |
| --- | --- |
| `github.com/yoshpy-dev/ralph` | no test files |
| `cmd/ralph` | no test files |
| `cmd/ralph-tui` | no test files |
| `internal/action` | ok |
| `internal/cli` | ok (includes `TestCheckLoopDriver_PriorityAndSource` 4 subtests, `TestAppendEnvIfMissing_*` 3 cases) |
| `internal/config` | ok (includes `TestDefault_Loop`, `TestLoad_LoopCodexDriver`, `TestLoad_LoopRejectsInvalidDriver` 3 subtests) |
| `internal/scaffold` | ok (includes `TestTemplateBaseRalphTomlHasLoopSection`; 2 historical mock-FS skips below) |
| `internal/state` | ok |
| `internal/ui` | ok |
| `internal/ui/panes` | ok |
| `internal/upgrade` | ok |
| `internal/watcher` | ok |

### Plan-required tests landed and green

| Test asset | Status |
| --- | --- |
| `tests/test-ralph-cli-driver.sh` (38 assertions, AC-1 + AC-3 + AC-5) | PASS |
| `internal/config/config_test.go::TestDefault_Loop` | PASS |
| `internal/config/config_test.go::TestLoad_LoopCodexDriver` | PASS |
| `internal/config/config_test.go::TestLoad_LoopRejectsInvalidDriver` (3 cases: unknown driver / sandbox / approval) | PASS |
| `internal/cli/run_env_test.go::TestAppendEnvIfMissing_*` (3 cases: TomlFillsWhenEnvAbsent / EnvWinsOverToml / DoesNotMatchPrefix) | PASS |
| `internal/cli/doctor_loop_test.go::TestCheckLoopDriver_PriorityAndSource` (4 subtests: default / env wins / TOML-only / empty TOML) | PASS |
| `internal/scaffold/embed_test.go::TestTemplateBaseRalphTomlHasLoopSection` | PASS |
| Preflight smoke `driver=claude --dry-run` | PASS (probes `claude_md_readable`, `json_output_format` — both `skip_dry_run` as designed) |
| Preflight smoke `driver=codex --dry-run` | PASS (probes flip to `agents_md_readable: pass` and `codex_exec_flags: skip_dry_run` — driver-aware probe selection works) |

## Coverage

- Statement / branch / function: not measured for this run (no `-coverprofile` requested by the plan; shell suites have no instrumented coverage tool).
- Notes:
  - `tests/test-ralph-cli-driver.sh` covers all six driver-wrapper paths called out in the plan: claude/JSON, codex/sidecar synth, codex `.last` missing fallback, DRY_RUN short-circuit on both drivers, `pick_reviewer` inversion (3 cases), and the cross-review dispatcher (driver=claude → codex review subcommand; driver=codex → claude `-p --permission-mode plan`).
  - Go-side TOML→env priority (AC-4) is exercised by the 4-subtest `TestCheckLoopDriver_PriorityAndSource` and the 3-case `TestAppendEnvIfMissing_*` in tandem.
  - Bash-side input validation (AC-12 implicit, plus the three plan edge cases) is exercised live against `ralph-pipeline.sh`, not just `ralph-config.sh` — the rejection happens before the preflight probe loop, which is the right surface.
  - Skipped: `internal/scaffold::TestBaseFS_WithMockFS` and `TestAvailablePacks_WithMockFS` — pre-existing skips unrelated to this plan (mock-FS variants of pack discovery; the real-embed counterparts run and pass). Documented in agent memory as long-standing.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |

No failures.

## Edge-case verdicts

Each edge case was run with `set +e` and `>/dev/null 2>&1; echo $?` to capture the real process exit (the initial `tee`-piped runs masked them as `0` because `tee` succeeds even when its stdin source dies; switched to `PIPESTATUS`-safe redirection).

| Edge case | Command | Exit code | Error message captured | Verdict |
| --- | --- | --- | --- | --- |
| Invalid driver | `RALPH_LOOP_DRIVER=foo ./scripts/ralph-pipeline.sh --preflight --dry-run` | **1** | `Error: RALPH_LOOP_DRIVER must be "claude" or "codex", got: foo` | PASS — fails fast, useful message |
| Invalid sandbox | `RALPH_CODEX_SANDBOX=bogus RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` | **1** | `Error: RALPH_CODEX_SANDBOX must be one of read-only|workspace-write|danger-full-access, got: bogus` | PASS |
| Invalid approval policy | `RALPH_CODEX_APPROVAL_POLICY=asap RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` | **1** | `Error: RALPH_CODEX_APPROVAL_POLICY must be one of untrusted|on-failure|on-request|never, got: asap` | PASS |

All three rejected with exit 1 and a self-explaining message that names the offending variable, the allowed value set, and the actual value.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Claude-driver pipeline behaviour (AC-12 backward-compat) | PASS | preflight-claude.log shows `driver: claude`, `claude_md_readable: skip_dry_run`, `json_output_format: skip_dry_run`, `Preflight probe PASSED`. Identical probe shape as pre-Phase-2 main aside from the added `driver:` line. |
| Skill-sync drift gate (`tests/test-check-skill-sync.sh`) | PASS | 6/6 cases green inside `run-verify.sh`. Both `loop` and `cross-review` skills synced across `.claude/skills/` and `.agents/skills/`. |
| Mojibake hook (`tests/test-check-mojibake.sh`) | PASS | 11/11 inside run-verify.sh. |
| `templates/base/ralph.toml` round-trip embedding | PASS | `TestTemplateBaseRalphTomlHasLoopSection` confirms `[loop]` section is shipped to scaffolded projects. |
| Go suites (action / cli / config / scaffold / state / ui / ui/panes / upgrade / watcher) all green | PASS | run-verify.sh "Running golang verifier" stanza shows `ok` for every package; uncached re-run in `go-test.log` confirms 228 PASS / 0 FAIL / 2 SKIP. |

## Test gaps

- AC-11 (real Codex walkthrough) is intentionally fake-codex-only on this run; the plan permits substitution when Codex CLI is available only in advisory mode. Walkthrough artifact at `docs/reports/walkthrough-*.md` is still pending and is tracked under the Progress checklist, not as a test failure.
- No instrumented Go coverage was captured. The plan's test plan does not require coverage thresholds, but adding `-coverprofile` to `run-verify.sh`'s Go stanza would surface drift on future Loop changes — flagged as a future hardening item, not blocking.
- The cross-review reviewer-inversion is exercised at the `pick_reviewer` and dispatcher layer (Test 5 + Test 6 in `test-ralph-cli-driver.sh`). End-to-end emission of `report_event "cross-review"` JSONL with `{"driver":"...","reviewer":"..."}` is not asserted in tests yet — the verifier already inspected the code path and confirmed it under AC-5; the test could be tightened with a JSONL grep follow-up. Not blocking for this PR.

## Flaky behaviour or environment-conditional skips

- 2 long-standing SKIPs in `internal/scaffold` (`TestBaseFS_WithMockFS`, `TestAvailablePacks_WithMockFS`) — unrelated to this plan, present on `main`, recorded in agent memory.
- No flaky tests observed in this run. Both preflight smokes and the three edge-case bashes were deterministic across re-runs.
- One operator-side caveat: piping `./scripts/ralph-pipeline.sh ... | tee` masks non-zero exits because `tee` returns 0. When asserting failure exit codes, use `>file 2>&1; echo $?` or `${PIPESTATUS[0]}`. Captured in agent memory under tester patterns for future runs.

## Verdict

- Pass: **YES**
- Fail: 0
- Blocked: 0

All test plan items (unit, integration, regression, edge cases) executed and green. The pipeline may proceed to `/sync-docs` and onward.
