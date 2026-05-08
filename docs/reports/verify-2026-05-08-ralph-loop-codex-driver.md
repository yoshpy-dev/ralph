# Verify report: Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Scope: Spec compliance against AC-1..AC-12 and static analysis on `feat/44/ralph-loop-codex-driver` (12 commits, 73a67e5..085bad7). Behavioural test execution is out of scope (handled by `/test`).
- Evidence: `docs/evidence/verify-2026-05-08-ralph-loop-codex-driver.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| **AC-1** `scripts/ralph-cli-driver.sh` exposes `run_agent` and dispatches on `RALPH_LOOP_DRIVER`; fake-CLI assertions cover argv/stdin/cwd/output | **Met** | `scripts/ralph-cli-driver.sh:39-58` (dispatch), `:62-88` (claude branch), `:93-140` (codex branch). `tests/test-ralph-cli-driver.sh:80-133` exercises both drivers via PATH-stubs at `tests/fixtures/cli-stubs/{claude,codex}` and asserts `.bin`/`.argv`/`.stdin`/`.session_id`/`.result`. Test compiles (`bash -n` OK). |
| **AC-2** `RALPH_LOOP_DRIVER=codex --preflight --dry-run` green; codex Probe 5 checks `--output-last-message`/`-s`/`-c`; claude branch keeps `claude_md_readable`, codex branch uses `agents_md_readable` | **Met** | Live run: `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` → exit 0, banner shows `driver: codex`, `agents_md_readable: pass`, `codex_exec_flags: skip_dry_run`. `scripts/ralph-pipeline.sh:280-309` (probe 3 driver-aware), `:323-379` (probe 5 driver-aware). |
| **AC-3** `<log>.json` written for codex driver; sidecar contract unchanged | **Met** | `scripts/ralph-cli-driver.sh:131-137` synthesises `{result, session_id:null}` via jq. `tests/test-ralph-cli-driver.sh:122-133` (Test 2) asserts `<log>.json` shape. Sidecar names (`agent-signal`, `self-review-result`, `verify-result`, `test-result`, `pr-url`) appear in 5 lines of `.claude/skills/loop/prompts/`; pipeline reads them unchanged. |
| **AC-4** `internal/cli/run.go` propagates `cfg.Loop.*` to env only when env unset; env wins on conflict | **Met** | `internal/cli/run.go:75-78` (4 `appendEnvIfMissing` calls). `internal/cli/run_env_test.go:12-50` covers 3 priority cases (`TomlFillsWhenEnvAbsent`, `EnvWinsOverToml`, `DoesNotMatchPrefix`). Compile-clean via `go vet ./...`. |
| **AC-5** cross-review dispatcher: `driver=claude → codex exec review`, `driver=codex → claude -p` adversarial; reviewer/driver fields recorded | **Met** | `scripts/ralph-pipeline.sh:755-817` is the dispatcher. `:778-797` runs the case statement; `:816-817` writes `driver`/`reviewer` into checkpoint and `report_event "cross-review"` JSONL. `tests/test-ralph-cli-driver.sh:174-214` asserts both branches (Test 5 + Test 6). Triage report header (`Driver:`, `Reviewer:`) is required by `.claude/skills/cross-review/prompts/adversarial-claude.md:53-54`. |
| **AC-6** `ralph status` / `ralph doctor` show effective driver with source | **Partially met** | `ralph doctor` (`internal/cli/doctor.go:433-465`, `checkLoopDriver`) prints `Loop driver: <effective> (source: env|toml|default)` plus sandbox/approval/reviewer when codex. `internal/cli/doctor_loop_test.go:14-...` covers env-only, TOML-only, both, neither. **However** `ralph status` (`scripts/ralph` cmd_status, `internal/cli/status.go`, `scripts/ralph-status-helpers.sh`) does **not** surface the driver — no driver/loop fields found. The orchestrator banner does log it (`scripts/ralph-orchestrator.sh:660-664`), so the runtime view is covered, but `ralph status` is the named user-facing command. Plan implementation step 8 only listed `internal/cli/doctor.go`, so this is plan/implementation drift. |
| **AC-7** `.claude/skills/{loop,cross-review}/SKILL.md` ↔ `.agents/skills/...` body parity; both mention Codex driver switch + reviewer inversion | **Met** | `cmp` byte-identical for both pairs. `./scripts/check-skill-sync.sh` reports "13 skill(s) in lock-step". `loop/SKILL.md:147-164` documents the switch (`RALPH_LOOP_DRIVER=codex ./scripts/ralph run`) and the inversion. `cross-review/SKILL.md:160-169` describes the Loop reviewer-inversion table. |
| **AC-8** `docs/recipes/ralph-loop.md` has Codex driver section with `codex trust .` → `RALPH_LOOP_DRIVER=codex ./scripts/ralph run` 3+ lines + sandbox/approval override | **Met** | `docs/recipes/ralph-loop.md:164` heading "Running Loop under the Codex driver"; `:170` mentions `codex trust .`; `:180` has `RALPH_LOOP_DRIVER=codex ./scripts/ralph run`; `:189-190` shows `codex_sandbox`/`codex_approval_policy` overrides. |
| **AC-9** `./scripts/run-verify.sh` green; `tests/test-ralph-cli-driver.sh` green | **Likely met (static portion)** | `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` → exit 0, all sub-verifiers pass (check-sync, check-pipeline-sync, check-skill-sync, golang verify). `tests/test-ralph-cli-driver.sh` is wired into `scripts/verify.local.sh:104-105`, syntax-checks clean. Behavioural execution of the test is `/test`'s job. |
| **AC-10** `internal/config/config_test.go` covers `[loop] driver = "codex"`, invalid driver, sandbox + approval allowlists | **Met** | `internal/config/config_test.go:132-...` `TestLoad_LoopCodexDriver` exercises the codex setting. `:162-...` `TestLoad_LoopRejectsInvalidDriver` tests three rejection paths in one table (unknown driver, sandbox, approval). `internal/scaffold/embed_test.go:128-145` `TestTemplateBaseRalphTomlHasLoopSection` asserts the embed payload. `go test ./internal/config -run NotARealTest` shows the package compiles. |
| **AC-11** Codex CLI walkthrough captured at `docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` (only required when codex CLI available) | **Deferred** | `codex` IS available on this host (preflight Probe 7: `codex CLI: available`), so per AC-11 the walkthrough should be produced. The file does not exist. This is a behavioural run that belongs to `/test` or a manual operator step. Not blocking for `/verify`, but `/test` and `/cross-review` should pick it up. |
| **AC-12** Backwards compat: env+TOML unset ⇒ driver=`claude`; existing claude users see no change | **Met** | `( . scripts/ralph-config.sh && echo $RALPH_LOOP_DRIVER )` → `claude`. `RALPH_LOOP_DRIVER=claude ./scripts/ralph-pipeline.sh --preflight --dry-run` → exit 0 with the original probe set (`claude_md_readable`, `json_output_format`). `internal/config/config.go:70-75` sets defaults in `Default()`. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `gofmt -l ./cmd ./internal` | **Pass** | empty output. |
| `go vet ./...` | **Pass** | empty output. |
| `go build ./...` | **Pass** | empty output. |
| `go test -run NotARealTest ./internal/cli ./internal/config ./internal/scaffold` | **Pass (compile-only)** | Used solely to confirm tests compile after the new fields/cases were added. |
| `shellcheck scripts/ralph-cli-driver.sh scripts/ralph-pipeline.sh scripts/ralph-orchestrator.sh scripts/ralph-config.sh` | **Pass with pre-existing info-level findings** | New `ralph-cli-driver.sh` is clean. Existing pipeline/orchestrator info findings (SC1091 `Not following: ralph-config.sh` x3, SC2016 `single quotes` x3, SC3045 `printf -` in orchestrator:549) are NOT introduced by this PR — `git diff main...HEAD -- scripts/ralph-orchestrator.sh` shows the SC3045 line was not touched in the diff. Same shape memory documents these as pre-existing. |
| `sh -n` syntax check on the four edited scripts | **Pass** | All four files parse. |
| `bash -n tests/test-ralph-cli-driver.sh` | **Pass** | Test script is syntactically valid. |
| `./scripts/check-skill-sync.sh` | **Pass** | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | **Pass** | `IDENTICAL: 145, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | **Pass** | All sub-verifiers (check-sync, check-pipeline-sync, check-skill-sync, golang verify, gofmt, go vet, go test) green. |
| Live preflight smoke: `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` | **Pass** | `=== Preflight probe PASSED ===`, codex-specific probes selected. |
| Live preflight smoke: `RALPH_LOOP_DRIVER=claude ./scripts/ralph-pipeline.sh --preflight --dry-run` | **Pass** | `=== Preflight probe PASSED ===`, claude-specific probes selected. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/loop/SKILL.md` ↔ `.agents/skills/loop/SKILL.md` body | **Yes** (`cmp` byte-identical, 13 skills in lock-step) | Both reference `RALPH_LOOP_DRIVER=codex ./scripts/ralph run` and the inverted reviewer. |
| `.claude/skills/cross-review/SKILL.md` ↔ `.agents/skills/cross-review/SKILL.md` body | **Yes** (`cmp` byte-identical) | Both describe the Loop-reviewer-inversion table. |
| `.claude/skills/cross-review/prompts/adversarial-claude.md` (Codex side) | **Not mirrored** to `.agents/skills/...` | The prompt file only exists in `.claude/` and `templates/base/.claude/`. **Acceptable** — Codex never invokes this prompt because driver=codex inverts to claude reviewer (handled on the Claude side). `check-skill-sync.sh` only enforces SKILL.md body parity, not arbitrary subdirs. Templates side (`templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md`) IS present. |
| Mirrored shell scripts (`templates/base/scripts/ralph-{cli-driver,pipeline,config,orchestrator}.sh`) | **Identical** to root copies | Verified via `cmp -s`. `check-sync.sh` confirms 145 identical files, zero drifted. |
| `docs/recipes/ralph-loop.md` | **In sync** | Adds full "Running Loop under the Codex driver" section + sandbox/approval override examples + env-vs-TOML priority guidance. |
| `docs/quality/definition-of-done.md` | **In sync** | `:45-49` adds the Phase-2 driver-selection note explaining env > TOML > default and the cross-review reviewer inversion. |
| `README.md` | **In sync** | `:217-222` documents `RALPH_LOOP_DRIVER=claude\|codex` and links to the recipe. |
| `AGENTS.md` | **Partial — see below** | Repo map (`AGENTS.md:72`) only mentions "Codex setup" recipe; the AGENTS.md primary loop and Codex setup checklist do NOT explicitly mention Loop driver selection. Plan said "AGENTS.md / Codex setup checklist references Loop driver where appropriate". This is a planning-vs-implementation soft drift. README + recipe + DoD already cover it. |
| `templates/base/AGENTS.md` / `templates/base/CLAUDE.md` | **In sync** with their root counterparts (per `check-sync.sh` KNOWN_DIFF policy) | These are scaffolded variants by design. |
| `internal/cli/doctor.go` (effective driver display) | **In sync** | `checkLoopDriver` is reachable from the doctor command, with priority-source logic mirroring `appendEnvIfMissing`. |
| `internal/cli/status.go` / `scripts/ralph-status-helpers.sh` | **Soft drift** | Neither references the Loop driver. AC-6 named both `ralph status` and `ralph doctor`; only doctor was implemented. Orchestrator startup banner does log driver, so runtime visibility is covered, but the static `ralph status` view is not. |

## Observational checks

- Confirmed via `cmp` that mirrored shell scripts and SKILL.md files are byte-identical between root, `templates/base/`, and `.agents/skills/`.
- Confirmed pipeline cycle counter, orchestrator validation, and TOML→env propagation all converge on the same allowlist (`claude|codex` × `read-only|workspace-write|danger-full-access` × `untrusted|on-failure|on-request|never`).
- Live preflight passes for both drivers; codex side correctly substitutes `agents_md_readable` for `claude_md_readable` and `codex_exec_flags` for `json_output_format`.
- `pick_reviewer()` is exposed at the top of `ralph-cli-driver.sh` and exercised by Test 5 of `test-ralph-cli-driver.sh` for all three branches (driver=claude, driver=codex, unset).

## Coverage gaps (handed off to /test and /cross-review)

1. **AC-9 (behavioural)** — `tests/test-ralph-cli-driver.sh` is wired and syntactically valid, but `/verify` does not run it. `/test` must execute it and confirm all 38 assertions pass.
2. **AC-11 (walkthrough)** — Real `RALPH_LOOP_DRIVER=codex ./scripts/ralph run ...` walkthrough not yet recorded. Codex CLI is available locally, so per the plan AC-11 *should* produce `docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md`. This is operator/`/test` work.
3. **AC-6 (status drift)** — `ralph status` does not surface the effective Loop driver (only `ralph doctor` does). Either tighten the AC during `/sync-docs`/`/cross-review`, or add a one-liner driver field to `internal/cli/status.go` + the shell `ralph-status-helpers.sh`. Self-review did not flag this; flagging here so a follow-up commit can decide.
4. **AGENTS.md soft drift** — Plan called for AGENTS.md/Codex setup checklist to reference the Loop driver. README, recipe, DoD all do; AGENTS.md does not (only the repo-map line). Optional follow-up.
5. **Pre-existing shellcheck noise** — Not regressions, but worth recording: `ralph-orchestrator.sh:549` (SC3045 `printf -`), three SC2016 single-quote info findings in `ralph-pipeline.sh`, SC1091 sourcing-not-followed info findings. None block this PR.

## Verdict

- **Pass** — all CRITICAL and HIGH ACs are met; spec compliance is solid; static analysis is clean; mirror parity holds.
- **Verified**: AC-1 (driver dispatch + fake-CLI test wiring), AC-2 (preflight green for both drivers, live), AC-3 (`<log>.json` synthesis + sidecar contract intact), AC-4 (env > TOML > default test triad), AC-5 (cross-review dispatcher + driver/reviewer JSONL recording), AC-7 (skill mirror parity), AC-8 (recipe Codex section), AC-10 (config tests for invalid driver/sandbox/approval), AC-12 (backwards compat).
- **Partially verified**: AC-6 (`ralph doctor` covered with full source-priority test; `ralph status` not surfaced — plan/implementation soft drift).
- **Likely but unverified (/test)**: AC-9 behavioural (`tests/test-ralph-cli-driver.sh` 38 assertions execution) — wiring/syntax confirmed, runtime green is `/test`'s job.
- **Deferred (Codex CLI walkthrough)**: AC-11 — codex is available; walkthrough report not produced. Operator/`/test` step.
- **Documentation soft drift (non-blocking)**: AGENTS.md primary-loop / Codex-setup-checklist does not explicitly mention Loop driver selection; only the repo-map line does. README + recipe + DoD compensate. `/sync-docs` may decide to tighten.

**Pipeline decision**: Continue. `/test` should run AC-9 + ideally produce the AC-11 walkthrough; `/cross-review` should weigh whether the AC-6 status gap and AGENTS.md soft drift are worth a follow-up commit on this branch.
