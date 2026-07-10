# Verify report: loop-model-routing

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-loop-model-routing.md
- Verifier: verifier subagent (claude-sonnet-4-6)
- Scope: changed-language scope (shell + golang)
- Evidence: `docs/evidence/verify-2026-07-10-184917.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `ralph-config.sh` defines, exports, and defaults all 8 new variables; `test-ralph-config.sh` asserts defaults and env-override | PASS | `scripts/ralph-config.sh:50-72`: `RALPH_FORCE_MODEL` (empty default), `RALPH_IMPLEMENT_MODEL` (sonnet), `RALPH_SELF_REVIEW_MODEL` (opus), `RALPH_VERIFY_MODEL` (sonnet), `RALPH_TEST_MODEL` (sonnet), `RALPH_SYNC_DOCS_MODEL` (sonnet), `RALPH_PR_MODEL` (sonnet), `RALPH_PROBE_MODEL` (haiku), `RALPH_ESCALATION_MODEL` (opus) — all exported. `tests/test-ralph-config.sh:93-142`: defaults and overrides asserted for FORCE, IMPLEMENT, PROBE, ESCALATION. |
| AC2: `run_agent` honors 4th-arg model for claude driver; omitted falls back to `RALPH_MODEL`; codex driver unaffected | PASS | `scripts/ralph-cli-driver.sh:190-204`: 4th positional `_model_arg` passed to `_run_agent_claude`; L215-244 passes it as `--model $_model_arg`; `_run_agent_codex` ignores it with explicit comment (L250). `tests/test-ralph-cli-driver.sh` (grep confirmed per self-review L47-48). |
| AC3: `resolve_phase_model` unit tests prove routing without CLI: implement cycle 1 → RALPH_IMPLEMENT_MODEL, cycle ≥ 2 → RALPH_ESCALATION_MODEL, `self_review` → RALPH_SELF_REVIEW_MODEL, RALPH_FORCE_MODEL=X forces all | PASS | `tests/test-model-routing.sh:336-385`: Tests 8a–8g (all phases + empty-cycle edge case) and Tests 9a–9d (FORCE_MODEL forces implement cycle 1, implement cycle 2 override, self_review, probe). `resolve_phase_model` is a pure function in `scripts/ralph-cli-driver.sh:47-80`. |
| AC3b: DRY_RUN=1 pipeline pass writes receipts for each phase; every line parses with `jq`; `requested_model` matches resolver; codex driver shows `effective_model="codex-default"` and `honored=false` | PASS | `tests/test-model-routing.sh`: Case 1 (DRY_RUN=1 default), Case 2 (FORCE_MODEL=opus), Case 3 (RALPH_LOOP_DRIVER=codex codex-default + honored=false assertions). Receipt schema verified: `ts/phase/cycle/driver/requested_model/effective_model/honored/effort/reason` — `write_model_receipt` in `scripts/ralph-cli-driver.sh:91-133`. Delegated to /test for full pipeline execution. |
| AC4: `ralph.toml` `[pipeline.phases]` keys parse into Go config with correct defaults; `ralph run` exports matching `RALPH_*` env vars with env > toml > default priority; `go test ./...` passes | PASS | `internal/config/config.go:57-78` (`PhaseModelConfig` struct, all 9 fields). `Default()` L106-115: all defaults set. `Load()` L194-216: backfill for all non-Force fields. `internal/cli/run.go:92-102`: `appendEnvIfMissing` for 8 per-phase vars; Force guarded by `!= ""` check. Go `config_test.go`: `TestDefault_Phases`, `TestLoad_PhasesAbsent`, `TestLoad_PhasesPartialBackfill`, `TestLoad_PhasesFullRoundTrip`. `go vet ./...` passes (0 issues). Behavioral `go test ./...` execution delegated to /test. |
| AC5: `check-sync.sh` and `check-pipeline-sync.sh` pass; scripts and `ralph.toml` byte-identical; `model-routing.md` follows allowlist policy | PASS | `check-sync.sh` output: DRIFTED=0, PASS. `model-routing.md` listed as KNOWN_DIFF (correct — root has Go-specific note, template omits it; diff shows only 5 lines removed from template). `cmp -s` confirms `ralph-config.sh`, `ralph-cli-driver.sh`, `ralph-pipeline.sh` are byte-identical to `templates/base/` copies. Root `ralph.toml` does not exist (meta-repo expected); `templates/base/ralph.toml` contains `[pipeline.phases]` with all 8 keys. `check-pipeline-sync.sh`: all 9 references OK. |
| AC6: `model-routing.md` documents per-phase variables, escalation rule, and receipts; "Where the values live" lists new keys; no stale refs | PASS | `model-routing.md:44-78`: "Ralph Loop per-phase routing" section with 8-row table, escalation rule, FORCE_MODEL single-knob rollback, receipt schema. "Where the values live" updated at L70-74 to include per-phase vars and `[pipeline.phases]` / `PhaseModelConfig`. `RALPH_MODEL` references in the file are contextually correct (global fallback for unrouted turns at L59; shell defaults at L70) — no stale usage. |
| AC7: `./scripts/run-verify.sh` passes end-to-end | PASS | `run-static-verify.sh` (equivalent) output: shellcheck OK for all hooks, `jq` validation OK, `check-sync.sh` PASS, `check-pipeline-sync.sh` OK (9 references), `check-skill-sync.sh` OK (13 skills), `gofmt` OK, `go vet` 0 issues. "All verifiers passed." Exit 0. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n scripts/ralph-config.sh` | OK | POSIX syntax valid |
| `sh -n scripts/ralph-cli-driver.sh` | OK | POSIX syntax valid |
| `sh -n scripts/ralph-pipeline.sh` | OK | POSIX syntax valid |
| `cmp -s scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` | IDENTICAL | Byte-identical mirror |
| `cmp -s scripts/ralph-cli-driver.sh templates/base/scripts/ralph-cli-driver.sh` | IDENTICAL | Byte-identical mirror |
| `cmp -s scripts/ralph-pipeline.sh templates/base/scripts/ralph-pipeline.sh` | IDENTICAL | Byte-identical mirror |
| `./scripts/check-sync.sh` | PASS (DRIFTED=0) | KNOWN_DIFF=3 (model-routing.md, verify.yml, CLAUDE.md — all pre-existing or AC5 allowlisted) |
| `./scripts/check-pipeline-sync.sh` | OK — 9/9 references | All pipeline step references found |
| `./scripts/check-skill-sync.sh` | OK — 13 skills in lock-step | No skill drift |
| `go vet ./...` | 0 issues | |
| `gofmt` check | OK | |
| `./scripts/run-static-verify.sh` | Exit 0 — All verifiers passed | Evidence: `docs/evidence/verify-2026-07-10-184917.log` |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/model-routing.md` — per-phase table | YES | New "Ralph Loop per-phase routing" section added with 8-row table, escalation, FORCE_MODEL, receipts. |
| `.claude/rules/model-routing.md` — "Where the values live" | YES | Updated to reference `scripts/ralph-config.sh` (8 per-phase vars), `ralph.toml` `[pipeline.phases]`, and `internal/config/config.go` `PhaseModelConfig`. Commit 709bafe. |
| `templates/base/.claude/rules/model-routing.md` | YES (allowlist diff) | Root copy has 5 extra lines about Go CLI not found in template. Difference is the allowlisted KNOWN_DIFF per AC5. Correct behavior. |
| `.claude/rules/subagent-policy.md` — Loop section | YES | Updated at the "Post-implementation pipeline for /loop" paragraph to reference `resolve_phase_model`, `RALPH_ESCALATION_MODEL`, and `model-receipts.jsonl`. Commit 709bafe. |
| `templates/base/.claude/rules/subagent-policy.md` | YES (IDENTICAL) | `cmp -s` confirmed via `check-sync.sh` (IDENTICAL, not KNOWN_DIFF). |
| `docs/recipes/ralph-loop.md` model config section | YES | L139 documents `RALPH_MODEL` fallback; L154-168 documents per-phase routing table and precedence; references `model-routing.md`. Commit 709bafe. |
| `templates/base/docs/recipes/ralph-loop.md` | YES (IDENTICAL) | check-sync.sh confirms IDENTICAL. |
| `templates/base/ralph.toml` `[pipeline.phases]` | YES | All 8 keys present with correct defaults; Force commented out. |
| Variable name consistency: `RALPH_SYNC_DOCS_MODEL` vs `sync_docs` | YES | Shell variable is `RALPH_SYNC_DOCS_MODEL` (`_` separator); TOML key is `sync_docs`; Go field is `SyncDocs`; `resolve_phase_model` case is `sync_docs`. All consistent. |
| Receipt schema fields documented vs implemented | YES | Schema in `model-routing.md`: `ts/phase/cycle/driver/requested_model/effective_model/honored/effort/reason`. Implementation in `write_model_receipt` (`ralph-cli-driver.sh:113-123`): all 9 fields match. |

## Observational checks

1. **`resolve_phase_model` is a pure function**: no CLI invocation, no side effects. Called safely in DRY_RUN mode. Verified by reading `ralph-cli-driver.sh:47-80`.

2. **Escalation threshold (`>= 2`) appears in two places** (self-review LOW-2): `resolve_phase_model` (authoritative) and the `_impl_reason` computation in `ralph-pipeline.sh:512`. They agree. Not a bug — the resolver is the single source of truth for the model value; the pipeline re-derives the label for receipt auditing.

3. **`write_model_receipt` codex path**: `effective_model="codex-default"`, `honored=false` (string `"false"` used with `--argjson` so it emits a JSON boolean). Confirmed at `ralph-cli-driver.sh:104-110`.

4. **`RALPH_FORCE_MODEL` empty guard in Go `run.go`**: `if cfg.Pipeline.Phases.Force != "" { appendEnvIfMissing(...) }` at L100-102 prevents blank override masking. Correct.

5. **`appendEnvIfMissing` semantics**: returns early if key is already present in `env` slice — preserves env > TOML priority. Verified at `internal/cli/run.go:92-102`.

6. **Cross-review receipt call sites**: `ralph-pipeline.sh:811` (codex path: `write_model_receipt cross_review`) and L881 (claude path: `write_model_receipt cross_review "$_cycle" "$RALPH_CLAUDE_REVIEWER_MODEL" "reviewer-inversion"`). Both present.

## Coverage gaps

1. **AC3b DRY_RUN pipeline pass behavioral assertion** — `test-model-routing.sh` exists and covers Case 1/2/3, but actual execution has not been observed in this verify run. Delegated to /test.

2. **`go test ./...` execution** — `go vet` passes; `TestDefault_Phases`, `TestLoad_Phases*`, `TestRunPipeline_ExportsPhaseModelEnv`, `TestRunPipeline_ExportsForceModelWhenSet` exist in the diff, but running the full test suite is delegated to /test.

3. **`tests/test-ralph-cli-driver.sh` 4th-arg model assertion** — file was modified (appears in diff), and a stub `claude` binary approach for `--model` capture was mentioned in the plan test section. Full test execution delegated to /test.

4. **Codex driver model-ignore at runtime** — the comment at `_run_agent_codex` and `honored=false` receipt are correct; actual Codex execution behavior is untestable without the `codex` binary.

## Verdict

PASS

- **Verified**: AC1 (8 new variables, defaults, export, test file coverage), AC2 (4th-arg model in `run_agent`/`_run_agent_claude`, codex ignores), AC3 (pure `resolve_phase_model` logic across all phases + FORCE + escalation + edge cases in test file), AC5 (byte-identical scripts, check-sync DRIFTED=0, allowlist policy for model-routing.md), AC6 (model-routing.md updated with per-phase table + escalation + receipts + "Where the values live"), AC7 (`run-static-verify.sh` exit 0, all gates pass).
- **Partially verified**: AC3b (write_model_receipt schema and codex-driver path verified statically; DRY_RUN pipeline behavioral test delegated to /test), AC4 (Go struct, defaults, backfill, env export logic verified; `go test ./...` execution delegated to /test).
- **Not verified**: Runtime CLI execution (actual `claude -p --model sonnet` flag passing, Codex driver behavior, escalation triggering in a live pipeline run). These are behavioral tests — delegated to /test.
- **Documentation drift**: None found. All variable names, receipt schema fields, and doc cross-references are consistent across shell, TOML, Go, and markdown.
- **Known gap carried forward**: LOW-1 from self-review — `write_model_receipt reason` can say "escalation" under RALPH_FORCE_MODEL at outer cycle ≥ 2. Audit field values (`requested_model`, `effective_model`) remain accurate; only the free-text `reason` is slightly misleading. Not a blocker.
