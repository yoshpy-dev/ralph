# Verify report (cycle 2): Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Branch: `feat/44/ralph-loop-codex-driver`
- Cycle: 2 (post-fix re-verify)
- Cycle-1 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md` (verdict: pass with two soft-drift items)
- Evidence: `docs/evidence/verify-2026-05-08-ralph-loop-codex-driver-cycle2.log`

## Scope (cycle 2)

Cycle 2 verifies four commits added since the cycle-1 verify report:

| Commit | Subject |
| --- | --- |
| `4e964c0` | docs: sync subagent-policy + tech-debt for Phase 2 Loop driver |
| `3351df2` | fix: surface Loop driver in `ralph status` + `AGENTS.md` map (resolves cycle-1 AC-6 partial-met / AGENTS.md soft drift) |
| `91232dc` | fix: address Codex cross-review ACTION_REQUIRED findings (P1 `count_triage_findings` helper + P2 TOML/shell asymmetry doc clarification) |
| `0663f50` | fix: anchor triage summary regex + ASCII-fy `>=` in comment (cycle-2 self-review polish) |

The full AC-1..AC-12 walk is preserved below. Cycle-2 focus areas are flagged with **[Δ cycle-2]**.

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| **AC-1** `scripts/ralph-cli-driver.sh` exposes `run_agent` and dispatches on `RALPH_LOOP_DRIVER`; fake-CLI assertions cover argv/stdin/cwd/output | **Met** | Wrapper unchanged in shape (`run_agent` at `:74-93`, claude branch `:97-123`, codex branch `:128-175`). `tests/test-ralph-cli-driver.sh:80-172` continues to exercise both drivers via PATH-stubs. **[Δ cycle-2]** the new helper `count_triage_findings` (`:35-58`) is co-located in this file per the architecture rule "small, boring, well-named abstractions"; AC-1 surface is unchanged. |
| **AC-2** `RALPH_LOOP_DRIVER=codex --preflight --dry-run` green; codex Probe 5 checks `--output-last-message`/`-s`/`-c`; claude branch keeps `claude_md_readable`, codex branch uses `agents_md_readable` | **Met** | Live: `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` → exit 0, banner shows `driver: codex`, `agents_md_readable: pass`, `codex_exec_flags: skip_dry_run`. Claude side: exit 0, `claude_md_readable`/`json_output_format` selected. No drift since cycle 1. |
| **AC-3** `<log>.json` written for codex driver; sidecar contract unchanged | **Met** | `scripts/ralph-cli-driver.sh:167-172` synthesises `{result, session_id:null}` via `jq -Rs`. `tests/test-ralph-cli-driver.sh:122-133` (Test 2) asserts `<log>.json` shape. Sidecar names (`agent-signal`, `self-review-result`, `verify-result`, `test-result`, `pr-url`) appear at lines 437/438/501/518/521-522/576/589-593/631/640-641/665/673-674/868-869 of pipeline; cycle-2 commits did not touch these. |
| **AC-4** `internal/cli/run.go` propagates `cfg.Loop.*` to env only when env unset; env wins on conflict | **Met** | Unchanged from cycle 1 — `internal/cli/run.go:75-78`, `internal/cli/run_env_test.go:12-50`. `go build ./...` and `go vet ./...` clean. |
| **AC-5** cross-review dispatcher: `driver=claude → codex exec review`, `driver=codex → claude -p` adversarial; reviewer/driver fields recorded | **Met** | `scripts/ralph-pipeline.sh:755-797` dispatcher: `:778-781` invokes `codex exec review --base "$_base"` for `_reviewer=codex`; `:782-797` invokes `claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" --permission-mode plan --output-format text < adversarial-claude.md` for `_reviewer=claude`. `:823-824` writes `{driver, reviewer, action_required, worth_considering, dismissed}` to checkpoint JSON and `report_event "cross-review"`. **[Δ cycle-2]** the cycle-2 change is consumption-side only (`count_triage_findings`); the dispatcher branches at `:778-797` were not touched by `91232dc`/`0663f50` (`git diff 085bad7..HEAD -- scripts/ralph-pipeline.sh` shows only the parser block at `:798-817` changed). Adversarial prompt file present at `.claude/skills/cross-review/prompts/adversarial-claude.md` and templates mirror. |
| **AC-6** `ralph status` / `ralph doctor` show effective driver with source | **Met** | **[Δ cycle-2]** Cycle-1 marked this *partially met* because `ralph status` did not surface the driver. Commit `3351df2` resolves it: `internal/cli/status.go:24` reads `RALPH_LOOP_DRIVER` and `:107` prints `Loop driver: <effective> (source: %s)`. Shell wrapper: `scripts/ralph-status-helpers.sh:328` prints `Loop driver: %s` using `${RALPH_LOOP_DRIVER:-claude}`. Both surfaces now expose the effective driver. |
| **AC-7** `.claude/skills/{loop,cross-review}/SKILL.md` ↔ `.agents/skills/...` body parity; both mention Codex driver switch + reviewer inversion | **Met** | `cmp -s` IDENTICAL on all four pairings (root×{claude,agents}, templates×{claude,agents}, root↔templates). `./scripts/check-skill-sync.sh` reports `13 skill(s) in lock-step`. **[Δ cycle-2]** `91232dc` added a TOML/shell asymmetry callout at lines 144-153 of `loop/SKILL.md` (and three mirrors); all four files remain byte-identical. |
| **AC-8** `docs/recipes/ralph-loop.md` has Codex driver section with `codex trust .` → `RALPH_LOOP_DRIVER=codex ./scripts/ralph run` 3+ lines + sandbox/approval override | **Met** | **[Δ cycle-2]** `91232dc` expanded the recipe with explicit TOML/shell asymmetry. `docs/recipes/ralph-loop.md:182` shows env-var entrypoint, `:189-201` adds the Go-binary entrypoint that honours `[loop] driver = "codex"`, and `:208-215` spells out: "the shell wrappers (`./scripts/ralph`, `scripts/ralph-orchestrator.sh`) do **not** parse `ralph.toml` itself — `[loop] driver = "codex"` alone, paired with `./scripts/ralph run`, will silently fall back to claude". `templates/base/docs/recipes/ralph-loop.md` is byte-identical. The same callout appears in both `loop/SKILL.md` mirrors at `:146-153`. |
| **AC-9** `./scripts/run-verify.sh` green; `tests/test-ralph-cli-driver.sh` green | **Likely met (static portion verified)** | `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` → exit 0, all sub-verifiers pass. `tests/test-ralph-cli-driver.sh` is wired through `scripts/verify.local.sh`, syntax-checks clean (`bash -n` exit 0). Behavioural execution remains `/test`'s job. **[Δ cycle-2]** assertion count verified by inspection: 10+12+2+6+3+5+3+3+2+1+1 = **48** (Tests 1/2/3/4/5/6/7a/7b/7c/7e/7d). Test 7e (`prose-only NOT picked as summary`) added by `0663f50`. |
| **AC-10** `internal/config/config_test.go` covers `[loop] driver = "codex"`, invalid driver, sandbox + approval allowlists | **Met** | Unchanged from cycle 1 — `internal/config/config_test.go:132-...` (`TestLoad_LoopCodexDriver`), `:162-...` (`TestLoad_LoopRejectsInvalidDriver`); embed test at `internal/scaffold/embed_test.go:128-145`. `go test -run NotARealTest ./internal/config` shows compile clean. |
| **AC-11** Codex CLI walkthrough at `docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` (only required when codex CLI available) | **Deferred** | `codex` is available locally (`codex CLI: pass`), so per AC-11 the walkthrough should ship. The file does not exist. Same status as cycle 1 — this is operator/`/test` work, not a `/verify` blocker. |
| **AC-12** Backwards compat: env+TOML unset ⇒ driver=`claude`; existing claude users see no change | **Met** | `( . scripts/ralph-config.sh && echo $RALPH_LOOP_DRIVER )` → `claude` (default). `RALPH_LOOP_DRIVER=claude ./scripts/ralph-pipeline.sh --preflight --dry-run` exits 0 with the original probe set. `internal/config/config.go` `Default()` continues to set `claude`/`workspace-write`/`on-failure`. No behavioural delta against `main` for default users. |

## Cycle-2 fix verification (focus areas)

### 1. Parser robustness — `count_triage_findings`

| Check | Result | Evidence |
| --- | --- | --- |
| Helper lives in `scripts/ralph-cli-driver.sh` (not pipeline) | **Yes** | `scripts/ralph-cli-driver.sh:25-58`. Co-located with `pick_reviewer` per architecture rule "small, boring, well-named abstractions… testable in isolation". |
| `ralph-pipeline.sh` calls helper, no inline `grep -c '<CATEGORY>'` | **Yes** | `scripts/ralph-pipeline.sh:810-812` calls `count_triage_findings ... ACTION_REQUIRED|WORTH_CONSIDERING|DISMISSED`. Old `_action_required="$(grep -c 'ACTION_REQUIRED' ...)"` lines fully removed from both root and `templates/base` copies. |
| Anchored summary regex (cycle-2 polish) | **Yes** | `scripts/ralph-cli-driver.sh:46`: `grep -m1 -E '^[- ]*After triage: ACTION_REQUIRED=[0-9]+'`. Matches the canonical bullet from `docs/reports/templates/cross-review-triage-report.md:12` (`- After triage: ACTION_REQUIRED=X, WORTH_CONSIDERING=Y, DISMISSED=Z`) but *not* a reviewer's prose body that happens to contain `ACTION_REQUIRED=2`. Test 7e at `tests/test-ralph-cli-driver.sh:312-326` locks this in. |
| Unicode `≥` removed from comment | **Yes** | `LC_ALL=C grep -nP '[\x80-\xFF]'` against root + template copies of `ralph-cli-driver.sh` and `ralph-pipeline.sh` returns **no matches**. The cycle-2 comment now reads `>=2 matches` (ASCII) at `scripts/ralph-pipeline.sh:803`. |
| Mirror parity (root ↔ templates/base) | **Yes** | `cmp -s` IDENTICAL for all four edited shell scripts. |

### 2. AC-5 dispatch unchanged (consumption-side only)

`git diff 085bad7..HEAD -- scripts/ralph-pipeline.sh` shows only:
- the parser block (`:798-817`) where the helper is called and a clarifying comment was added
- (cycle-2 polish) one comment-line ASCII-fy

The driver-aware case statement at `scripts/ralph-pipeline.sh:778-797` (which is the AC-5 contract) was **not** touched. `pick_reviewer()` and the dispatch invariants are preserved. Test 6 (`tests/test-ralph-cli-driver.sh:187-214`) continues to assert that `driver=claude` invokes `codex exec review` and `driver=codex` invokes `claude -p --permission-mode plan` against the adversarial prompt.

### 3. AC-8 TOML / shell-wrapper asymmetry callout

The asymmetry is now spelled out in **four** mirrored locations (all byte-identical):
- `docs/recipes/ralph-loop.md:189-215` — full prose explanation with both entrypoints
- `templates/base/docs/recipes/ralph-loop.md` — IDENTICAL to root via `cmp -s`
- `.claude/skills/loop/SKILL.md:144-153` and `.agents/skills/loop/SKILL.md:144-153` — short-form callout: "ralph.toml の `[loop] driver = ...` は **Go バイナリ `ralph run` 経由でのみ有効**"
- `templates/base/.claude/skills/loop/SKILL.md` and `templates/base/.agents/skills/loop/SKILL.md` — IDENTICAL via `cmp -s`

The contract is correctly summarised: TOML alone requires `./scripts/ralph run` (Go binary), because the shell wrappers do not parse `ralph.toml`.

### 4. Test 7e and 48-assertion count

- Test 7e (`tests/test-ralph-cli-driver.sh:312-326`) feeds a triage report whose prose body contains `ACTION_REQUIRED=2` but no `After triage:` header line, and asserts `count_triage_findings ... ACTION_REQUIRED == 0`. This is the regression test for the anchored-regex fix in `0663f50`.
- Per-test assertion tally: 1a-1j(10) + 2a-2l(12) + 3a-3b(2) + 4a-4f(6) + 5a-5c(3) + 6a-i,ii + 6b-i,ii,iii(5) + 7a-i,ii,iii(3) + 7b-i,ii,iii(3) + 7c-i,ii(2) + 7e(1) + 7d(1) = **48**. Matches the expected total.
- `bash -n tests/test-ralph-cli-driver.sh` exits 0 (syntax clean).

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `gofmt -l ./cmd ./internal` | **Pass** | empty output. |
| `go vet ./...` | **Pass** | empty output. |
| `go build ./...` | **Pass** | empty output. |
| `shellcheck scripts/ralph-cli-driver.sh scripts/ralph-pipeline.sh scripts/ralph-orchestrator.sh scripts/ralph-config.sh` | **Pass with pre-existing info-level findings** | New helper in `ralph-cli-driver.sh` is clean. Findings: SC1091 (sourcing) x3, SC2016 (single-quoted jq filters in `ckpt_update`) x3, SC3045 (`printf -` in orchestrator:549). All present pre-Phase-2; cycle-2 introduces none. |
| `sh -n` on the four edited scripts | **Pass** | All four parse. |
| `bash -n tests/test-ralph-cli-driver.sh` | **Pass** | Test script syntactically valid (338 lines). |
| `./scripts/check-skill-sync.sh` | **Pass** | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | **Pass** | `IDENTICAL: 145, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | **Pass** | All sub-verifiers green. |
| Live preflight: `RALPH_LOOP_DRIVER=codex --preflight --dry-run` | **Pass** | exit 0, codex-specific probes selected. |
| Live preflight: `RALPH_LOOP_DRIVER=claude --preflight --dry-run` | **Pass** | exit 0, claude-specific probes selected. |
| Non-ASCII byte scan (Unicode `≥` removal) | **Pass** | `LC_ALL=C grep -nP '[\x80-\xFF]'` over root + template copies of the two edited scripts → no matches. |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/loop/SKILL.md` ↔ `.agents/skills/loop/SKILL.md` (root + templates) | **Yes** | `cmp -s` IDENTICAL across all four. |
| `.claude/skills/cross-review/SKILL.md` ↔ `.agents/skills/cross-review/SKILL.md` | **Yes** | `check-skill-sync.sh` covers it (13 skills lock-step). |
| `.claude/skills/cross-review/prompts/adversarial-claude.md` (referenced by pipeline) | **Present in `.claude/` and `templates/base/.claude/`** | Pipeline only reads from `.claude/skills/...` (the dispatch-side prompt, not Codex-side), so the absence of an `.agents/` mirror is acceptable — `check-skill-sync.sh` only enforces SKILL.md body parity. |
| Mirrored shell scripts (`templates/base/scripts/ralph-{cli-driver,pipeline,config,orchestrator}.sh`) | **IDENTICAL** to root | Verified via `cmp -s` (all four). |
| `docs/recipes/ralph-loop.md` ↔ `templates/base/docs/recipes/ralph-loop.md` | **IDENTICAL** | `cmp -s`. **[Δ cycle-2]** TOML/shell asymmetry section added at `:189-215`. |
| `docs/quality/definition-of-done.md` | **In sync** | `:46` references `RALPH_LOOP_DRIVER` and source priority. |
| `README.md` | **In sync** | `:217` documents `RALPH_LOOP_DRIVER=claude\|codex`. |
| `AGENTS.md` | **In sync** | **[Δ cycle-2]** `3351df2` added line 36 explicitly summarising Loop driver selection + cross-review reviewer inversion + `ralph status`/`ralph doctor` exposure. Cycle-1 soft drift resolved. |
| `templates/base/AGENTS.md` / `templates/base/CLAUDE.md` | **In sync** | per `check-sync.sh` allowed deltas. |
| `internal/cli/doctor.go` (effective driver display) | **In sync** | `checkLoopDriver` unchanged, exercised by `internal/cli/doctor_loop_test.go`. |
| `internal/cli/status.go` / `scripts/ralph-status-helpers.sh` | **In sync** | **[Δ cycle-2]** Cycle-1 flagged this as soft drift (status did not surface driver). `3351df2` added `internal/cli/status.go:24,107` and `scripts/ralph-status-helpers.sh:328`. AC-6 now fully met. |

## Coverage gaps (handed off to /test)

1. **AC-9 behavioural** — `tests/test-ralph-cli-driver.sh` is wired and syntax-clean, but `/verify` does not execute it. `/test` must run it and confirm all 48 assertions pass.
2. **AC-11 walkthrough** — Real `RALPH_LOOP_DRIVER=codex ./scripts/ralph run ...` walkthrough still not recorded. Codex CLI is available locally; per the plan this *should* be produced. Operator/`/test` work.
3. **Pre-existing shellcheck noise** — Not regressions: SC1091 sourcing-not-followed (info), SC2016 single-quoted jq filters x3 (info), SC3045 `printf -` in `ralph-orchestrator.sh:549` (warning). Cycle-2 introduces none.

## Verdict

- **Pass** — all CRITICAL and HIGH ACs are met; the two cycle-1 soft-drift items (AC-6 status surfacing, AGENTS.md primary-loop note) are now resolved by `3351df2`; the cycle-2 fix targets (parser regression P1, anchored-regex MEDIUM, Unicode `≥` LOW) are verified in code, mirrored to templates, and locked in by Test 7 + Test 7e.
- **Verified**: AC-1, AC-2, AC-3, AC-4, AC-5 (dispatch unchanged; consumption-side parser swap proven), AC-6 (cycle-1 partial → cycle-2 met), AC-7, AC-8 (cycle-2 callout added in 4 mirrored files), AC-10, AC-12.
- **Likely but unverified (handed to /test)**: AC-9 behavioural — 48 assertions to be executed.
- **Deferred**: AC-11 — Codex walkthrough report not produced; codex CLI is available, so operator should run it during `/test` or as a follow-up.
- **Documentation drift**: none. AGENTS.md, README, recipe, DoD, SKILL.md mirrors all synchronised.

**Pipeline decision**: Continue. `/test` should:
1. Execute `tests/test-ralph-cli-driver.sh` and confirm 48/48 pass (with focus on Test 7 and Test 7e).
2. Optionally produce the `docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` if the operator wants to close AC-11 on this branch.

## Smallest additional check that would raise confidence

Run only `tests/test-ralph-cli-driver.sh` under `bash -e` (the file currently uses `set -uo pipefail`; an explicit `-e` invocation by the runner would short-circuit on the first failed assertion and surface the exact regression site). This is a one-line add to `scripts/run-verify.sh`'s test invocation, not a code change.
