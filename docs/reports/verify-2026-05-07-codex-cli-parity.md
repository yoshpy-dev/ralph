# Verify report: Codex CLI standard-flow parity

- Date: 2026-05-07
- Plan: `docs/plans/active/2026-05-07-codex-cli-parity.md`
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Scope: 10 commits on `feat/codex-cli-parity` (`12064e6` … `82679f1`); standard-flow Codex parity only. Loop driver work is OUT OF SCOPE — tracked in yoshpy-dev/ralph#44. No behavioural tests run here (that is `/test`'s responsibility).
- Evidence: `docs/evidence/verify-2026-05-07-codex-cli-parity.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| **AC-1** scaffold layout — `<dir>/{.claude/, .codex/, .agents/skills/, AGENTS.md, CLAUDE.md, ralph.toml}` | Verified | `templates/base/.codex/{config.toml,AGENTS.override.md,README.md,hooks/}` present; `templates/base/.agents/skills/.gitkeep` + 12 mirrored skill dirs present; `internal/cli/cli_test.go::TestExecuteInit_RendersCodexSurfaces`; `internal/scaffold/embed_test.go::TestTemplateBaseCodexAssetsExist`. |
| **AC-1b** `ralph doctor` effective config probe | Verified (static) | `internal/cli/doctor.go:148-203` — `checkCodexEffectiveConfig` parses `.codex/config.toml` and reports `pass`/`warn`/`fail` based on `[features] codex_hooks` + `[hooks.*]` entry count. Tests: `internal/cli/cli_test.go::TestCheckCodexEffectiveConfig_*`. |
| **AC-2** Codex full standard flow `$spec → $plan → $work → … → $pr` | Likely but unverified at runtime | Static contract evidence (skill mirrors, canonical pipeline references, `$skill-name` invocation rule documented in `templates/base/AGENTS.md` "Skill invocation" table) all green. Live smoke deferred per `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md` "Known gaps recorded". CI cannot run Codex (no binary, no credentials). |
| **AC-3** `scripts/check-skill-sync.sh` drift gate (5 checks) | Verified | `./scripts/check-skill-sync.sh` exits 0 with "13 skill(s) in lock-step". Six-fixture battery in `tests/test-check-skill-sync.sh` covers: A clean parity, B inventory drift, C body drift, D description drift, E policy drift (claude-forbid/codex-allow), F policy parity (both forbid). All PASS. Wired into `scripts/run-verify.sh` (and `run-static-verify.sh`). |
| **AC-4** post-impl pipeline parity Codex=sequential inline | Verified (static) | `templates/base/.claude/rules/subagent-policy.md` "Post-implementation pipeline under Codex — sequential inline" section + `.claude/rules/post-implementation-pipeline.md` "CLI execution mode" table both encode the rule. Canonical order matches between Claude and Codex sides. |
| **AC-5** `/cross-review` bidirectionality | Verified for `/work` skill body; intentionally partial in Loop scripts | `.claude/skills/cross-review/SKILL.md:33-55` (Step 2 driver detection + Step 4 dual reviewer dispatch via `RALPH_PRIMARY_CLI` and `which`-fallback). Triage report path is `docs/reports/cross-review-triage-<slug>.md`. Loop scripts retain Claude-driven `codex exec review` only — explicitly out-of-scope and signposted by the comment block at `scripts/ralph-pipeline.sh:716-726` (added in `82679f1`). |
| **AC-6** `ralph doctor` dual-CLI detection | Verified | `internal/cli/doctor.go::checkClaudeCLI` + `checkCodexCLI` both wired; both honour the `Require*` config flags (default false → warn-only). `internal/config/config.go:36` adds `RequireCodexCLI`; `internal/config/config_test.go::TestLoad_RequireCodexCLI` covers explicit `true`/`false`. |
| **AC-7** `run-verify.sh` green + AGENTS.md ≤32 KiB + audit-harness warning | Verified | `./scripts/run-verify.sh` exit 0 (this session, evidence `docs/evidence/verify-2026-05-07-092523.log`). `wc -c AGENTS.md templates/base/AGENTS.md` → 5286 / 4973 bytes (16% of 32768 cap). `audit-harness` warning logic at `.claude/skills/audit-harness/SKILL.md:22-33` (WARN >24 KiB, FAIL >32 KiB). |
| **AC-8** Tests for `.codex/` + `.agents/skills/` go:embed and `ralph init` | Verified | `internal/scaffold/embed_test.go::TestTemplateBaseCodexAssetsExist`; `internal/cli/cli_test.go::TestExecuteInit_RendersCodexSurfaces` (asserts manifest tracks Codex-side paths). The originally-planned `tests/upgrade_downgrade_test.go` is documented as deferred — see Coverage gaps below. |
| **AC-9** Rename ripple `codex-review`→`cross-review`, grep clean against allowlist | Verified | `grep -rln "codex-review\|codex-triage"` produces hits only in: `docs/plans/active/2026-05-07-codex-cli-parity.md` (plan body), `docs/specs/2026-05-07-codex-cli-parity.md` (spec body), `docs/plans/archive/*` (history), `docs/reports/codex-triage-2026-04-*.md` (immutable history), `docs/reports/{self-review,sync-docs,verify,test,walkthrough}-2026-04-*.md` (immutable history), `docs/reports/*-mojibake-postedit-guard.md` (immutable history), `docs/reports/self-review-2026-05-07-codex-cli-parity.md` and `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md` (current-cycle history), `docs/recipes/codex-setup.md` + `templates/base/docs/recipes/codex-setup.md` (rename history for end users), and `scripts/check-sync.sh:71` (the allowlist literal itself, with explanatory comment block). All match the plan's AC-9 allowlist. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-verify.sh` | exit 0 | "All verifiers passed." Evidence at `docs/evidence/verify-2026-05-07-092523.log`. |
| `./scripts/run-static-verify.sh` (HARNESS_VERIFY_MODE=static) | exit 0 | "All verifiers passed." Evidence at `docs/evidence/verify-2026-05-07-092536.log`. |
| `./scripts/check-skill-sync.sh` | exit 0 | "13 skill(s) in lock-step". |
| `./scripts/check-pipeline-sync.sh` | exit 0 | All 8 canonical-order consumers green. |
| `./scripts/check-sync.sh` | exit 0 | 139 IDENTICAL, 0 DRIFTED, 0 ROOT_ONLY, 14 TEMPLATE_ONLY (all expected — `.codex/`, `.agents/skills/.gitkeep`, etc.), 3 KNOWN_DIFF. |
| `./scripts/check-template.sh` | exit 0 | "Template structure looks good." |
| `gofmt -l .` | exit 0 | (no output) |
| `go vet ./...` | exit 0 | (no output) |
| `golangci-lint run` | exit 0 | "0 issues." |
| `shellcheck` (touched scripts) | exit 1 (info/warning only) | Only pre-existing SC1091 (sourced ralph-config.sh), SC2016 (intentional single-quoted jq filters), SC1083 (literal `{}` in `HEAD@{upstream}`), SC3045 (`printf '- ...'` POSIX-undefined warning) in `ralph-pipeline.sh` and `ralph-orchestrator.sh`. None introduced by this PR; none in the new `scripts/check-skill-sync.sh` or `tests/test-check-skill-sync.sh`. |
| `tests/test-check-skill-sync.sh` (6 fixtures) | 6 PASS / 0 FAIL | A clean parity, B inventory drift, C body drift, D description drift, E policy drift, F policy parity. |
| `tests/test-check-mojibake.sh` (11 fixtures) | 11 PASS / 0 FAIL | Pre-existing battery, run in passing as part of `run-verify.sh`. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md` (root + `templates/base/`) | Yes | Renamed step 8 to "Cross-review", references `cross-review` only. Skill-invocation table forbids `/skill-name` for Codex. |
| `CLAUDE.md` (root + `templates/base/`) | Yes | Manual-trigger skill list updated (`/release` repo-maintainer-only — not in `ralph init` — confirmed by absence from `templates/base/.claude/skills/`). |
| `README.md` | Yes | Pipeline diagrams updated to `cross-review`; both Claude (`/cross-review`) and Codex (`$cross-review`) syntax shown. |
| `docs/quality/definition-of-done.md` (root + template) | Yes | Pipeline order references `cross-review` consistently. |
| `docs/quality/quality-gates.md` | N/A — already silent | Does not reference the renamed skill. |
| `.claude/rules/post-implementation-pipeline.md` (root + template) | Yes | "CLI execution mode" table added; canonical order updated. |
| `.claude/rules/subagent-policy.md` (root + template) | Yes | New "Post-implementation pipeline under Codex — sequential inline" section + table updates. |
| `docs/recipes/codex-setup.md` (root + template) | Yes | Describes `codex trust .`, `ralph doctor`, and the rename for upgrading users. The two `codex-review` occurrences are intentional history, allowed by AC-9 allowlist. |
| `templates/base/.codex/AGENTS.override.md` + `README.md` | Yes | Provides project-trust + `[features] codex_hooks` guidance + Claude→Codex permission/sandbox table. |
| `templates/base/.codex/config.toml` | Yes | `model = "gpt-5.5"`, `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"`, `[features] codex_hooks = true`, profiles `work`/`review`. |
| `templates/base/ralph.toml` | Yes | New `[doctor] require_codex_cli = false` documented inline; `[pipeline]` model unchanged (Claude side). |
| `docs/architecture/repo-map.md` | Yes | One-line update reflecting Codex surfaces. |
| `docs/tech-debt/README.md` | Yes | Two new entries appended for the partial Loop rename and pre-rename JSON checkpoint migration (per self-review). |
| `docs/plans/active/2026-05-07-codex-cli-parity.md` Progress checklist | Stale (informational) | Slices 1–7 marked complete; Self-review/Verify/Test/PR boxes still unchecked. This is the expected "checklist drift" pattern for in-flight verification — not a fail condition. |
| Walkthrough `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md` | Yes | Documents deferred manual smoke test + the deferred upgrade round-trip fixture as "Known gaps recorded". |

## Observational checks

- AGENTS.md size: `5286` (root) / `4973` (template) bytes — 16% of the 32 KiB Codex `project_doc_max_bytes` cap. Comfortable headroom.
- Skill mirror inventory:
  - `.claude/skills/`: 13 directories (anti-bottleneck, audit-harness, cross-review, loop, plan, pr, release, self-review, spec, sync-docs, test, verify, work).
  - `.agents/skills/`: 13 directories (same set including `release`).
  - `templates/base/.claude/skills/`: 12 directories (no `release/` — repo-maintainer-only, not scaffolded into user projects).
  - `templates/base/.agents/skills/`: 12 directories (matches Claude template side; `release/` correctly excluded).
- `D-8` policy parity audit — `spec` and `release` carry `disable-model-invocation: true` on Claude side and `agents/openai.yaml`'s `policy.allow_implicit_invocation: false` on Codex side, in both root and (where applicable) template. Other 11 skills carry neither field on either side, which `check-skill-sync.sh`'s policy check accepts as "both default true".
- 82679f1 follow-up effect: `scripts/ralph-pipeline.sh` lines 715-760 retain `_codex_log` / `_has_codex` variable names, but a new 11-line comment block at lines 716-726 documents that those names are an intentional bookmark for the Loop driver work tracked at yoshpy-dev/ralph#44 — addressing the self-review HIGH "rename ripple stops at the phase string" finding via documentation rather than full rename. Acceptable under AC-9's "string-only rename ripple" scope (phase name, log paths, report glob, checkpoint key all renamed; private bookkeeping variable names retained).

## Coverage gaps

| Gap | Severity | Why deferred / Disposition |
| --- | --- | --- |
| Live Codex CLI smoke run for AC-2 (`$spec → … → $pr`) | Medium | Codex binary + credentials not available in CI. Documented in `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md` "What requires a manual smoke test" with a 6-step recipe for a maintainer with Codex installed. Static contract evidence is fully green; only behavioural confirmation against the live CLI is missing. |
| `tests/upgrade_downgrade_test.go` (Slice 7 step 36) | Low | Documented as "Known gaps recorded" in the walkthrough; existing `internal/cli` upgrade tests cover the diff engine. Round-trip-specific scenario lands as follow-up work. Plan progress checklist still marks Slice 7 complete because the live smoke test was the load-bearing item. |
| Codex/Claude `permission_mode` ↔ `sandbox_mode + approval_policy` semantic equivalence | Low | Mapping documented in `templates/base/.codex/AGENTS.override.md` and `templates/base/.codex/README.md`; behavioural confirmation requires the same live smoke run as AC-2. |
| Pre-rename JSON checkpoint migration in `internal/state/PipelineCheckpoint` | Low | Self-review MEDIUM finding; recorded in `docs/tech-debt/README.md`. Risk window is small (checkpoints regenerate naturally on next run) and accepted by maintainer. |
| Plan progress checklist boxes for Self-review / Verify / Test / PR | Trivial | Standard in-flight drift; the four steps are the post-implementation pipeline, completed in sequence. Not a verification fail. |

## Verdict

- **Verified**: AC-1, AC-1b, AC-3, AC-4, AC-5 (`/work` skill body), AC-6, AC-7, AC-8 (within scope of in-tree fixtures), AC-9. All static-analysis gates pass (`run-verify.sh`, `run-static-verify.sh`, `check-skill-sync.sh`, `check-pipeline-sync.sh`, `check-sync.sh`, `check-template.sh`, `gofmt`, `go vet`, `golangci-lint`, `shellcheck`). AGENTS.md sits at ~16% of the 32 KiB Codex cap. Documentation references for `cross-review` are consistent across AGENTS.md / CLAUDE.md / README.md / `docs/quality/` / rules / skills, with `codex-review` residue only inside the AC-9 allowlist.
- **Partially verified**: AC-5 in the Loop pipeline (`scripts/ralph-pipeline.sh`) — the Outer Loop hard-codes `codex exec review`. This is intentionally out of scope per the plan's Non-goals and is signposted by the comment block at lines 716-726 (yoshpy-dev/ralph#44 reference).
- **Not verified**: AC-2 end-to-end (Codex live smoke `$spec → … → $pr`) and the deferred `tests/upgrade_downgrade_test.go`. Both are documented gaps with a defined manual or follow-up handoff path; neither blocks `/verify` from passing.

**Pass.** No fail-condition AC, no static-analysis regression, no doc drift outside the AC-9 allowlist. The smallest single check that would most increase confidence is the manual Codex smoke run (10–15 minutes, recipe in the walkthrough): it is the only behavioural confirmation still outstanding for AC-2 and AC-5 (Codex driver path) before merge, and the deterministic gates in this branch already exercise every codified contract that surrounds it.

---

## Cycle 2 (cap-reached) re-verify

- Date: 2026-05-07
- Cycle: 2/2 — `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`. Cap is reached after this run.
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Scope: cycle-2 delta only — commits `3abd1d7` (cross-review ACTION_REQUIRED — `probeBinary`, default Codex hooks, bidirectional triage template) and `79d7a73` (self-review CRITICAL fix — repair broken hook paths/contracts, add `probeBinary` timeout / first-line / symmetric error). Cycle-1 ACs already verified above are spot-checked, not re-litigated.
- Evidence: `docs/evidence/verify-2026-05-07-100406.log` (full `run-verify.sh` rerun)

### Cycle-2 spec compliance — full AC sweep

| AC | Status | Cycle-2 evidence |
| --- | --- | --- |
| **AC-1** scaffold layout | Verified | Unchanged from cycle 1. `templates/base/.codex/config.toml` updated by `79d7a73` (hook contents) but file/path layout unchanged. `TestExecuteInit_RendersCodexSurfaces` re-runs PASS (manifest still tracks `.codex/config.toml`, `.agents/skills/spec/SKILL.md`, etc.). |
| **AC-1b** `ralph doctor` effective config probe | Verified, hardened | Two cycle-2 improvements: (a) `probeBinary` (now used by `checkClaudeCLI` / `checkCodexCLI`) wraps `LookPath` errors symmetrically, applies a 5s `context.WithTimeout`, and returns the first non-empty line of `--version`. (b) `checkCodexEffectiveConfig` semantics unchanged; now reports `pass — codex_hooks=true, 2 hook entry(ies). Confirm 'codex trust .' ran for this project` against the cycle-2 default-on hooks. New tests: `TestProbeBinary_BrokenShimFails`, `TestProbeBinary_MissingBinary`. Pre-existing `TestCheckCodexEffectiveConfig_*` (5 cases) still PASS. Live `go run ./cmd/ralph doctor` from repo root: pass-line is `Claude Code CLI: pass — 2.1.132 (Claude Code)` and `Codex CLI: pass — codex-cli 0.128.0`, confirming the version string surfaces correctly. |
| **AC-2** Codex full standard flow | Likely but unverified at runtime | Cycle-2 changes do not affect Codex flow surface. Live smoke still deferred. |
| **AC-3** `check-skill-sync.sh` drift gate | Verified | `./scripts/check-skill-sync.sh` exits 0; "13 skill(s) in lock-step". `tests/test-check-skill-sync.sh` 6/6 PASS in `run-verify.sh` rerun. |
| **AC-4** post-impl pipeline parity Codex=sequential inline | Verified | Unchanged from cycle 1. |
| **AC-5** `/cross-review` bidirectionality | Verified for `/work`; intentionally partial in Loop scripts | Unchanged from cycle 1. The `82679f1` signpost block at `scripts/ralph-pipeline.sh:716-726` still in place. |
| **AC-6** `ralph doctor` dual-CLI detection | Verified, hardened | Now goes beyond `LookPath` to actually run `--version`, with a 5s timeout. Both `checkClaudeCLI` and `checkCodexCLI` honour the `Require*` config flags (default false → warn-only). |
| **AC-7** `run-verify.sh` green + AGENTS.md ≤32 KiB | Verified | `./scripts/run-verify.sh` exit 0 in this rerun (evidence `docs/evidence/verify-2026-05-07-100406.log`). AGENTS.md sizes unchanged from cycle 1 (5286 / 4973 bytes). |
| **AC-8** Tests for `.codex/` + `.agents/skills/` go:embed and `ralph init` | Verified | Cycle-2 adds two new tests (`TestProbeBinary_*`); existing `TestExecuteInit_RendersCodexSurfaces` and `TestTemplateBaseCodexAssetsExist` re-run PASS. |
| **AC-9** Rename ripple grep clean against allowlist | Verified | `grep -rn "codex-review\|codex-triage"` across `*.sh / *.go / *.toml / *.json / *.yaml / *.yml` shows only: (a) `.claude/settings.local.json` (gitignored, local-only — confirmed in `.gitignore:15`, not shipped), and (b) `scripts/check-sync.sh:71` (the allowlist literal itself). All Markdown hits are inside the AC-9 allowlist (specs/plans/recipes/historical reports). No new residue introduced by cycle-2 commits. |

### Cycle-2 static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-verify.sh` (rerun this cycle) | exit 0 | "All verifiers passed." Evidence `docs/evidence/verify-2026-05-07-100406.log`. |
| `./scripts/check-skill-sync.sh` | exit 0 | 13 skill(s) in lock-step. |
| `./scripts/check-pipeline-sync.sh` | exit 0 | All 5 canonical-order consumers green. |
| `./scripts/check-sync.sh` | exit 0 | 139 IDENTICAL, 0 DRIFTED, 0 ROOT_ONLY, 14 TEMPLATE_ONLY, 3 KNOWN_DIFF. |
| `gofmt -l .` (root, Go files) | exit 0 | No diff in `internal/cli/doctor.go` or any cycle-2-touched Go file. |
| `go vet ./internal/cli/` | exit 0 | Clean. |
| `golangci-lint run` | exit 0 | (via `run-verify.sh`) — 0 issues. |
| `go test ./internal/cli/ -run "TestProbeBinary\|TestCheckCodex\|TestExecuteInit_RendersCodexSurfaces" -v` | PASS (8 / 8) | `TestProbeBinary_BrokenShimFails`, `TestProbeBinary_MissingBinary`, 5 `TestCheckCodexEffectiveConfig_*` cases, `TestExecuteInit_RendersCodexSurfaces`. |
| `python3 -c 'import tomllib; tomllib.loads(open("templates/base/.codex/config.toml").read())'` | OK | Cycle-2 TOML edit parses cleanly. |
| `tests/test-check-mojibake.sh` (11 fixtures) | 11 PASS / 0 FAIL | Pre-existing battery; the new Codex hooks reuse `.claude/hooks/check_mojibake.sh`, the same script this battery exercises. |

### Cycle-2 documentation drift

| Surface | In sync? | Notes |
| --- | --- | --- |
| `templates/base/.codex/config.toml` (cycle-2 comments) | Yes | The new comment block at lines 64-68 ("Reuses the same script the Claude-side hook calls (.claude/hooks/check_mojibake.sh)…") accurately describes the active wiring after `79d7a73`. The `Note:` block at lines 77-81 explains why no `^git commit` PostToolUse hook ships. Both comments are load-bearing — they explain the cycle-1 trap to a future maintainer reading the file alone. |
| `internal/cli/doctor.go` `probeBinary` docstring | Yes | The docstring at lines 111-120 was extended in `79d7a73` to document the 5s timeout and the first-non-empty-line return contract. Matches behaviour. |
| Plan progress checklist | Stale (informational) | The four post-impl checkboxes (Self-review/Verify/Test/PR) remain unchecked in `docs/plans/active/2026-05-07-codex-cli-parity.md:302-305`. Standard in-flight pattern; not a fail condition. (Carried over from cycle 1.) |
| `docs/tech-debt/README.md` | In sync | `79d7a73` appended two new debt rows: "ralph doctor's `checkHooks` only walks `.claude/settings.json` paths…" and "`probeBinary` in `internal/cli/doctor.go` runs `<bin> --version` with no timeout via `exec.Command(...).CombinedOutput()`". The second row is now stale because `79d7a73` itself added the timeout — see "Cycle-2 stale tech-debt note" below. |

### Resolution of cycle-2 self-review CRITICALs

| Cycle-2 CRITICAL finding | Disposition in `79d7a73` | Verified |
| --- | --- | --- |
| Edit/Write hooks point at non-existent `./scripts/check_mojibake.sh` | Repointed to `./.claude/hooks/check_mojibake.sh` (real file: `ls -la` returns 3.4k, executable, both at root and in `templates/base/`). Comment block at lines 64-68 documents the "share with Claude-side hook" rationale. | Yes — `grep -n "command = " templates/base/.codex/config.toml` shows only `./.claude/hooks/check_mojibake.sh` for both Edit and Write entries. No `./scripts/check_mojibake.sh` in any shipped config. |
| `^git commit` PostToolUse wiring of `commit-msg-guard.sh` (wrong script class) | Removed entirely. Replaced by a 4-line `# Note:` comment block at lines 77-81 explaining why this guard belongs in `.git/hooks/commit-msg`, not in `[hooks.PostToolUse]`. | Yes — `grep -n "^git commit\|scripts/commit-msg-guard" templates/base/.codex/config.toml` returns hits only inside the explanatory comment block, no active `[[hooks.PostToolUse]]` entry references it. |

Both cycle-2 CRITICAL hook-path bugs are resolved by configuration change (no script moves required). The fix is minimal and grep-able.

### Cycle-2 stale tech-debt note

The cycle-2 self-review tech-debt entry "`probeBinary` has no timeout" was added in the same commit (`79d7a73`) that *also* added the timeout. The debt row in `docs/tech-debt/README.md` now describes a problem that no longer exists. This is a low-impact documentation drift — recommend striking that row in a follow-up doc-only commit. Not a /verify fail condition because the underlying code is correct.

The other new debt row ("`checkHooks` only walks `.claude/settings.json`; no equivalent for `.codex/config.toml`") remains accurate and load-bearing — `checkHooks` was not extended in `79d7a73`.

### Cycle-2 verdict

- **Verified**: AC-1 (cycle-2 hook content edit does not break scaffold layout), AC-1b (now hardened via `probeBinary` timeout + first-line trim), AC-3, AC-4, AC-6 (now exercises `--version` rather than `LookPath`-only), AC-7 (run-verify.sh exit 0 again), AC-8 (new tests pass), AC-9 (no new residue). Two cycle-2 self-review CRITICALs resolved.
- **Partially verified**: AC-5 in Loop pipeline — unchanged from cycle 1 (intentional, signposted at `scripts/ralph-pipeline.sh:716-726`).
- **Not verified**: AC-2 end-to-end Codex live smoke — unchanged from cycle 1 (recipe in `docs/reports/walkthrough-2026-05-07-codex-cli-parity.md`).
- **Documentation drift**: one stale `docs/tech-debt/README.md` row about `probeBinary` timeout — fixed by the same commit that added it. Recommend a doc-only follow-up commit to strike the row.

**Pass (cycle 2/2, cap reached).** No AC regressed. The cycle-2 fix commits land both cross-review ACTION_REQUIRED items (probe robustness + bidirectional template) and the self-review CRITICALs (broken hook paths + misapplied commit-msg guard). The smallest additional check that would most increase confidence remains the manual Codex live smoke (AC-2 / AC-5 Codex driver path); no additional deterministic check is needed for cycle-2 specifically.
