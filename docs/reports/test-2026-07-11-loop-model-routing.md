# Test report: loop-model-routing

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-loop-model-routing.md
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: changed-language scope (shell + golang) — full fallback triggered by unclassified scripts/ralph-cli-driver.sh; golang pack executed
- Evidence: `docs/evidence/test-2026-07-11-loop-model-routing.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | — |
| `tests/test-branch-name.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-check-skill-sync.sh` | 7 | 7 | 0 | 0 | — |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | — |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | — |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 | — |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 | — |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-model-routing.sh` **(new)** | 19 | 19 | 0 | 0 | — |
| `tests/test-ralph-cli-driver.sh` **(extended)** | 77 | 77 | 0 | 0 | — |
| `tests/test-ralph-config.sh` **(extended)** | 41 | 41 | 0 | 0 | — |
| `tests/test-ralph-dry-run-side-effects.sh` | 5 | 5 | 0 | 0 | — |
| `tests/test-ralph-orchestrator-branch-names.sh` | 3 | 3 | 0 | 0 | — |
| `tests/test-ralph-orchestrator-pr-strategy.sh` | 24 | 24 | 0 | 0 | — |
| `tests/test-ralph-run-options.sh` | 5 | 5 | 0 | 0 | — |
| `tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | — |
| `tests/test-ralph-slice-skip-pr.sh` | 4 | 4 | 0 | 0 | — |
| `tests/test-ralph-status.sh` | 51 | 51 | 0 | 0 | — |
| `tests/test-ralph-worktree.sh` | 17 | 17 | 0 | 0 | — |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | — |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | — |
| `tests/test-self-review-scope.sh` | 96 | 96 | 0 | 0 | — |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 | — |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 | — |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | — |
| `tests/test-xreview-gate-regression.sh` | 21 | 21 | 0 | 0 | — |
| `tests/test-xreview-prompt-render.sh` | 54 | 54 | 0 | 0 | — |
| **Shell subtotal** | **762** | **762** | **0** | **0** | — |
| `go test ./internal/config/...` | 5 | 5 | 0 | 0 | 0.47s |
| `go test ./internal/cli/...` | 4 | 4 | 0 | 0 | 7.98s |
| `go test ./internal/action/...` | — | ok | 0 | 0 | 2.33s |
| `go test ./internal/scaffold/...` | — | ok | 0 | 0 | 0.84s |
| `go test ./internal/state/...` | — | ok | 0 | 0 | 1.19s |
| `go test ./internal/ui/...` | — | ok | 0 | 0 | 0.31s |
| `go test ./internal/ui/panes/...` | — | ok | 0 | 0 | 1.82s |
| `go test ./internal/upgrade/...` | — | ok | 0 | 0 | 1.44s |
| `go test ./internal/watcher/...` | — | ok | 0 | 0 | 2.97s |
| `./scripts/run-test.sh` (aggregate) | — | EXIT 0 | — | — | — |

## Plan test-plan item coverage

### AC1 — per-phase model defaults and env overrides in `ralph-config.sh`

Covered by `tests/test-ralph-config.sh` (41 tests total, 41 pass):
- "Per-phase model variable default tests" section: `RALPH_FORCE_MODEL` (empty), `RALPH_IMPLEMENT_MODEL` (sonnet), `RALPH_SELF_REVIEW_MODEL` (opus), `RALPH_VERIFY_MODEL` (sonnet), `RALPH_TEST_MODEL` (sonnet), `RALPH_SYNC_DOCS_MODEL` (sonnet), `RALPH_PR_MODEL` (sonnet), `RALPH_PROBE_MODEL` (haiku), `RALPH_ESCALATION_MODEL` (opus) — all 9 defaults confirmed.
- "Per-phase model variable env override tests": `RALPH_IMPLEMENT_MODEL`, `RALPH_SELF_REVIEW_MODEL`, `RALPH_PROBE_MODEL`, `RALPH_FORCE_MODEL`, `RALPH_ESCALATION_MODEL` — 5 env-override assertions confirmed.

### AC2 — `run_agent` 4th-arg model passed as `--model` to claude driver

Covered by `tests/test-ralph-cli-driver.sh` Test 10 (3 tests):
- 10a: 4th arg `sonnet` → `--model sonnet` PASS
- 10b: omitted 4th arg → `--model` falls back to `RALPH_MODEL` (opus) PASS
- 10c: empty 4th arg → `--model` falls back to `RALPH_MODEL` (opus) PASS

Codex driver unaffected: no `--model` arg injected (Test 2 confirms only sandbox/approval flags).

### AC3 — `resolve_phase_model` unit tests (pure function, no CLI)

Covered by `tests/test-ralph-cli-driver.sh` Tests 8 and 9 (12 tests):
- 8a: implement cycle 1 → sonnet PASS
- 8b: implement cycle 2 → opus (escalation) PASS
- 8c: implement empty cycle → sonnet (no escalation) PASS
- 8d: self_review → opus PASS
- 8e: probe → haiku PASS
- 8f: unknown phase → RALPH_MODEL fallback PASS
- 8g: verify → sonnet PASS
- 8h: test → sonnet PASS
- 9a: FORCE_MODEL=haiku forces implement cycle 1 → haiku PASS
- 9b: FORCE_MODEL=haiku forces implement cycle 2 → haiku (not opus) PASS
- 9c: FORCE_MODEL=haiku forces self_review → haiku PASS
- 9d: FORCE_MODEL=haiku forces probe → haiku PASS

### AC3b — DRY_RUN=1 pipeline receipt sequence and codex-driver semantics

Covered by `tests/test-model-routing.sh` (19 tests across 3 cases):

**Case 1: DRY_RUN=1 default routing (9 tests)**
- 1a: model-receipts.jsonl created PASS
- 1b: all receipt lines parse as valid JSON PASS
- 1c–1g: implement→sonnet, self_review→opus, verify→sonnet, test→sonnet, sync_docs→sonnet PASS
- 1h: honored=true (claude driver) PASS
- 1i: driver=claude PASS

**Case 2: RALPH_FORCE_MODEL=opus (6 tests)**
- 2a: receipt file created PASS
- 2b–2f: all phases (implement, self_review, verify, test, sync_docs) → opus PASS

**Case 3: RALPH_LOOP_DRIVER=codex (4 tests)**
- 3a: receipt file created PASS
- 3b: effective_model=codex-default PASS
- 3c: honored=false PASS
- 3d: driver=codex PASS

Receipt schema verified by `tests/test-ralph-cli-driver.sh` Test 11 (14 tests):
- All 9 schema fields (ts, phase, cycle, driver, requested_model, effective_model, honored, effort, reason) asserted on both claude and codex paths.

### AC4 — Go config `[pipeline.phases]` and `ralph run` env export

Covered by `go test ./internal/config/...` (5 tests, all PASS; run cold — no cache):
- `TestDefault_Phases`: all 9 fields have correct default values PASS
- `TestLoad_PhasesAbsent`: missing `[pipeline.phases]` section backfills all defaults PASS
- `TestLoad_PhasesPartialBackfill`: partial toml section backfills missing fields PASS
- `TestLoad_PhasesFullRoundTrip`: all 9 toml keys round-trip correctly PASS
- `TestLoad_TemplateRalphToml`: `templates/base/ralph.toml` loads without error and has correct implement default PASS

Covered by `go test ./internal/cli/...` (2 specific tests, all PASS):
- `TestRunPipeline_ExportsPhaseModelEnv`: `ralph run` passes 8 per-phase env vars to pipeline PASS
- `TestRunPipeline_ExportsForceModelWhenSet`: `RALPH_FORCE_MODEL=opus` is exported when non-empty; not exported when empty PASS

## Edge cases confirmed

| Edge case | Test | Status |
| --- | --- | --- |
| Empty 4th arg to `run_agent` falls back to `RALPH_MODEL` | test-ralph-cli-driver.sh 10c | PASS |
| Omitted 4th arg falls back to `RALPH_MODEL` | test-ralph-cli-driver.sh 10b | PASS |
| Unset phase var falls back to default (unknown phase → RALPH_MODEL) | test-ralph-cli-driver.sh 8f | PASS |
| Empty cycle arg treated as cycle 1 (no escalation) | test-ralph-cli-driver.sh 8c | PASS |
| RALPH_FORCE_MODEL overrides escalation (cycle 2 still → force model) | test-ralph-cli-driver.sh 9b | PASS |
| Codex driver: receipts created but effective_model=codex-default | test-model-routing.sh Case 3 | PASS |
| RALPH_FORCE_MODEL='' (empty) does not mask phase variables | test-model-routing.sh Case 1 | PASS |
| Receipts dir auto-created (no pre-existing `.harness/state/pipeline/`) | test-model-routing.sh Case 1–3 | PASS |
| All receipt lines parse as valid JSON | test-model-routing.sh 1b, test-ralph-cli-driver.sh 11c | PASS |
| FORCE_MODEL unset in Go `run.go` when empty (no blank env override) | TestRunPipeline_ExportsForceModelWhenSet | PASS |
| Partial `[pipeline.phases]` toml backfills missing fields to default | TestLoad_PhasesPartialBackfill | PASS |

## Coverage

- Statement: Shell scripts — no instrumented coverage; all branches exercise by test case scope. Go packages: `go test -cover` not explicitly measured; all named test targets pass.
- Branch: Per-phase routing branches (force / escalation / phase-default / fallback) confirmed by Tests 8, 9, and Cases 1–3. FORCE_MODEL empty-guard branch in `run.go` confirmed by `TestRunPipeline_ExportsForceModelWhenSet`.
- Function: `resolve_phase_model`, `write_model_receipt`, `_run_agent_claude` (model arg path), `_run_agent_codex` (model-ignored path), `appendEnvIfMissing`, `Default().Pipeline.Phases`, `Load()` backfill all exercised.
- Notes: Codex actual runtime model behavior (whether `codex exec` truly ignores `--model`) is untestable without the `codex` binary. This is a documented known gap (non-fixable without Codex API support), not a test suite gap.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Shell test glob discovers all `tests/test-*.sh` (fix/verify-local-test-glob) | OK | All 29 suites executed via glob; no hand-maintained list |
| `count_triage_findings` anchored regex (pick_reviewer/cross-review) | OK | test-ralph-cli-driver.sh Tests 7a–7e PASS |
| xreview placeholder substitution and gate regression | OK | test-xreview-prompt-render.sh 54 PASS, test-xreview-gate-regression.sh 21 PASS |
| Terraform language detection and gitignore | OK | All 8 + 47 PASS |

## Test gaps

- `RALPH_PROBE_MODEL` and `RALPH_VERIFY_MODEL` / `RALPH_TEST_MODEL` / `RALPH_SYNC_DOCS_MODEL` / `RALPH_PR_MODEL` env-override assertions not in `test-ralph-config.sh` — only 5 of 8 non-Force per-phase vars have override tests. Not a blocker: defaults are asserted for all 9, and `resolve_phase_model` confirms routing via its own tests. Optional follow-up: add override tests for the remaining 4 vars.
- `RALPH_FORCE_MODEL` env override does appear in `test-ralph-config.sh`, confirming it is non-empty (`haiku`) — but `RALPH_VERIFY_MODEL` / `RALPH_TEST_MODEL` / `RALPH_SYNC_DOCS_MODEL` / `RALPH_PR_MODEL` overrides are not separately tested in that suite. The `resolve_phase_model` tests (8g, 8h) confirm routing logic for those phases.
- Codex runtime model behavior is untestable without the `codex` binary. Known gap; receipt schema confirms `honored=false` and `effective_model=codex-default` for the auditable record.
- `TestLoad_TemplateRalphToml` tests that the template TOML loads without error and the implement default is correct, but does not assert all 9 fields independently (covered by `TestLoad_PhasesFullRoundTrip` which uses an in-memory TOML string). Minor redundancy opportunity but not a gap.

## Verdict

- Pass: YES
- Fail: 0
- Blocked: No

`./scripts/run-test.sh` exited 0. All 29 shell test suites (762 assertions) and all 9 Go packages passed. The 5 plan-specific test targets named by `/verify` (TestDefault_Phases, TestLoad_PhasesAbsent/PartialBackfill/FullRoundTrip, TestLoad_TemplateRalphToml, TestRunPipeline_ExportsPhaseModelEnv, TestRunPipeline_ExportsForceModelWhenSet) were confirmed individually against a cold test cache. AC1 through AC3b and AC4 behavioral execution are all confirmed PASS. No regressions detected in existing suites.
