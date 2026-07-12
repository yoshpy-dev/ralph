# Verify report: unify-permission-mode

- Date: 2026-07-12
- Plan: `docs/plans/active/2026-07-12-unify-permission-mode.md`
- Verifier: verifier subagent (sonnet)
- Scope: changed-language scope (Go + shell; 14 files changed on fix/unify-permission-mode-default)
- Evidence: `docs/evidence/verify-2026-07-12-093108.log`
- Prior artifact: `docs/reports/self-review-2026-07-12-unify-permission-mode.md` (MERGE, 2 LOW)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: All three sources state `bypassPermissions` | PASS | `scripts/ralph-config.sh:20` — `RALPH_PERMISSION_MODE="${RALPH_PERMISSION_MODE:-bypassPermissions}"`; `templates/base/ralph.toml` diff line +7 — `permission_mode = "bypassPermissions"`; `internal/config/config.go` diff — `PermissionMode: "bypassPermissions"`. Load() backfill covered by `TestLoad_PermissionModeBackfill` (config_test.go). |
| AC2: env wins for RALPH_MODEL/EFFORT/PERMISSION_MODE | PASS (test existence confirmed) | `TestRunPipeline_EnvWinsOverTomlForModelEtc` in `internal/cli/run_env_test.go` sets toml to distinct values, pre-sets env, and asserts env values survive and that the count-per-key is exactly 1. Implementation: `appendEnvIfMissing` for all three vars (`run.go:62-64` in diff). |
| AC3: MAX_ITERATIONS/MAX_PARALLEL priority (flag > env > toml) | PASS (test existence confirmed) | `TestRunPipeline_MaxIterFlagBeatsEnv` (flag path: `maxIterChanged=true`, CLI=5 beats env=99) and `TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag` (env path: env=42 beats toml=7 when `maxIterChanged=false`). Implementation: `cmd.Flags().Changed("max-iterations"/"max-parallel")` (`run.go:34-35`) — Cobra flag-presence, not `!=0` heuristic (Codex advisory finding 4). |
| AC4: Recipe row unified; tech-debt row RESOLVED; README/.codex README no longer claim `auto`; rollback/override wording notes env-everywhere vs toml-only-via-Go | PASS | `docs/recipes/ralph-loop.md:141` — "Default is `bypassPermissions` across all entry points … shell wrappers do not parse `ralph.toml`". `docs/tech-debt/README.md` — HTML comment + strikethrough RESOLVED annotation. `README.md:276` and `.codex/README.md:58` — updated to `bypassPermissions` with conservative-override wording. Mirror byte-identity: `cmp docs/recipes/ralph-loop.md templates/base/docs/recipes/ralph-loop.md` → IDENTICAL; `cmp .codex/README.md templates/base/.codex/README.md` → IDENTICAL. |
| AC4b: `TestLoad_TemplateRalphToml` asserts template permission_mode == Default() | PASS | `config_test.go:378-381` — added assertion `cfg.Pipeline.PermissionMode != Default().Pipeline.PermissionMode`. Upgrade-path nuance documented in test comment at `config_test.go:356-359`. |
| AC5: Go tests / run-test.sh / run-verify.sh / check-sync / check-skill-sync pass | PARTIAL — static side PASS; behavioral side deferred to /test | `run-static-verify.sh < /dev/null` exits 0: shellcheck OK, jq OK, check-sync IDENTICAL=170/DRIFTED=0, check-pipeline-sync OK (9 files), check-skill-sync 13 skills in lock-step, gofmt OK, go vet 0 issues. Go behavioral tests and `run-test.sh` are /test responsibility. |

## Codex advisory adoption

| Finding | Adopted? | Evidence |
| --- | --- | --- |
| 1 HIGH — hook-protection wording corrected | PASS | `templates/base/ralph.toml:14` — "interactive hooks do NOT fire under `claude -p`" |
| 2 HIGH — rollback wording: env everywhere, toml only via Go path | PASS | `docs/recipes/ralph-loop.md:141`, `README.md:276`, `.codex/README.md:58` all state the distinction |
| 3 MEDIUM — empty-env contract defined and tested end-to-end | PASS | `TestRunPipeline_EmptyPermissionModeEnv` (Go side) + `test_empty_env` in `tests/test-ralph-config.sh` (shell side) |
| 4 MEDIUM — `Flags().Changed()` instead of `!=0` | PASS | `run.go:34-35` — `cmd.Flags().Changed("max-iterations")` / `cmd.Flags().Changed("max-parallel")` |
| 5 MEDIUM — template-toml permission assertion + upgrade-path nuance in PR body | PASS | `config_test.go:378-381` assertion; upgrade note at `config_test.go:356-359` |
| 6 MEDIUM — README/.codex README doc scope added | PASS | Both updated to `bypassPermissions` default with override note |
| 7 LOW — optional live smoke | NOT APPLIED (by design) | Plan states "optional, skip with a note when claude binary absent — do NOT add to CI-critical path". No CI regression from omission. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh < /dev/null` | PASS (exit 0) | All gates passed; see evidence log |
| shellcheck (all hooks) | PASS | 18 hook files, all OK |
| `jq -e . .claude/settings.json` | PASS | Valid JSON |
| `jq -e . templates/base/.claude/settings.json` | PASS | Valid JSON |
| `scripts/check-sync.sh` | PASS | IDENTICAL=170, DRIFTED=0, ROOT_ONLY=0 |
| `scripts/check-pipeline-sync.sh` | PASS | 9 target files, all pipeline steps referenced |
| `scripts/check-skill-sync.sh` | PASS | 13 skills in lock-step |
| `gofmt` | PASS | No formatting issues |
| `go vet` | PASS | 0 issues |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/ralph-loop.md` RALPH_PERMISSION_MODE row | IN SYNC | Unified to `bypassPermissions`; conservative override wording added |
| `templates/base/docs/recipes/ralph-loop.md` | IN SYNC | Byte-identical mirror confirmed via `cmp` |
| `docs/tech-debt/README.md` permission-mode row | IN SYNC | RESOLVED annotation applied (HTML comment + strikethrough) |
| `README.md` permission policy column | IN SYNC | Updated from `auto` to `bypassPermissions` with override note |
| `.codex/README.md` permission policy column | IN SYNC | Updated; mirror of `templates/base/.codex/README.md` |
| `templates/base/.codex/README.md` | IN SYNC | Byte-identical mirror of `.codex/README.md` |
| `templates/base/ralph.toml` toml comment (guardrails wording) | IN SYNC | Correctly states hooks do NOT fire under `claude -p`; no overclaiming |
| `.claude/rules/model-routing.md` | IN SYNC / UNTOUCHED | No model-semantics change; model-routing.md correctly untouched per plan Non-goals |
| `docs/quality/definition-of-done.md` | IN SYNC / UNTOUCHED | No pipeline-order change; no drift |
| `CLAUDE.md` / `AGENTS.md` | IN SYNC / UNTOUCHED | No workflow-order or contract change; no drift |

### Residue check

`grep -rn 'permission_mode.*=.*auto|permission_mode = auto|permission_mode="auto"' ... --include="*.md" --include="*.toml" ... | grep -v docs/tech-debt` → 0 hits. No operational file still claims `auto` as the effective default.

## Observational checks

- `appendEnvIfMissing` is an existing tested helper (`run.go:248-256`); this PR reuses it for 3 previously unconditional `append` calls. No new machinery introduced.
- `cmd.Flags().Changed()` correctly distinguishes `--max-iterations 0` (flag present) from flag absent, which the old `!=0` heuristic could not.
- Both mirror pairs confirmed byte-identical via `cmp` — check-sync would also catch future drift via CI.
- The Codex advisory finding 7 (optional live smoke) is intentionally not applied; plan explicitly states it must not be added to the CI-critical path. No gap for /verify.

## Coverage gaps

- AC5 behavioral portion (Go test execution, `run-test.sh` shell glob) is deferred to /test. Test names exist at code-reading level: `TestRunPipeline_EnvWinsOverTomlForModelEtc`, `TestRunPipeline_MaxIterFlagBeatsEnv`, `TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag`, `TestRunPipeline_EmptyPermissionModeEnv` in `internal/cli/run_env_test.go`; `TestDefault_PermissionMode`, `TestLoad_PermissionModeBackfill`, `TestLoad_TemplateRalphToml` assertion in `internal/config/config_test.go`; `test_empty_env` in `tests/test-ralph-config.sh`.
- Codex advisory finding 7 (live `claude -p --permission-mode bypassPermissions` smoke) is optional and excluded from CI — not a gap for /verify.

## Verdict

**PASS**

- Verified: AC1 (three-source grep), AC2 (test existence + implementation), AC3 (test existence + Cobra Flags().Changed implementation), AC4 (recipe row, tech-debt RESOLVED, README drift eliminated, mirror byte-identity), AC4b (TestLoad_TemplateRalphToml assertion + upgrade note), all 7 Codex advisory adoptions, static analysis (run-static-verify.sh exit 0, check-sync DRIFTED=0, check-skill-sync 13 skills, gofmt, go vet), documentation drift (no `auto` residue, model-routing.md correctly untouched).
- Partially verified: AC5 — static side PASS; behavioral test execution deferred to /test.
- Not verified: Live runtime smoke (advisory finding 7, intentionally excluded from CI-critical path).
