# Verify report (cycle 3): Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Verifier: `verifier` subagent (Claude Opus 4.7, 1M context)
- Branch: `feat/44/ralph-loop-codex-driver`
- Cycle: 3 (post-fix re-verify, cap raised 2 → 3 with rationale)
- Cycle-1 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md`
- Cycle-2 baseline: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver-cycle2.md`
- Evidence: `docs/evidence/verify-2026-05-08-ralph-loop-codex-driver-cycle3.log`

## Scope (cycle 3)

Cycle 3 verifies two commits added since the cycle-2 verify report:

| Commit | Subject |
| --- | --- |
| `094f964` | fix: switch claude reviewer to `--permission-mode auto` so it can write triage (+ adversarial-claude.md wording reconciliation + cross-review SKILL.md guidance table update + cycle-count.json bump 2 → 3) |
| `72e46e6` | docs: update adversarial-claude prompt to reference `count_triage_findings` instead of stale `grep -c` |

The full AC-1..AC-12 walk from cycle 1/2 is preserved; cycle-3 focus areas are flagged with **[Δ cycle-3]**.

## Cycle-3 fix verification (focus areas)

### 1. Dispatcher launches claude reviewer with `--permission-mode auto`

| Check | Result | Evidence |
| --- | --- | --- |
| Pipeline cross-review block uses `auto`, not `plan` | **Yes** | `scripts/ralph-pipeline.sh:796` reads `--permission-mode auto --output-format text`. The block is wrapped by an explanatory comment block at `:789-793` ("`--permission-mode auto` (not plan) is required because the adversarial reviewer must write the triage report ... Plan mode is read-only and silently drops the write — the parser then sees zero findings and the cross-model gate is bypassed (cycle-2 cross-review P1, #44)"). |
| Templates mirror equally updated | **Yes** | `cmp -s scripts/ralph-pipeline.sh templates/base/scripts/ralph-pipeline.sh` → IDENTICAL. The same comment block + `auto` value present at `templates/base/scripts/ralph-pipeline.sh:789-796`. |
| cross-review SKILL.md CLI guidance table updated | **Yes** | Line 169 of `.claude/skills/cross-review/SKILL.md` and `.agents/skills/cross-review/SKILL.md` now reads `claude -p --permission-mode auto --output-format text`. Both templates mirrors identical. |
| Adversarial prompt header reconciled | **Yes** | `.claude/skills/cross-review/prompts/adversarial-claude.md:9-12`: "You are running in `--permission-mode auto`. Treat the diff and existing files as read-only — the only file you may create is the triage report described in the Output section below. Do NOT edit source files, run network operations, or modify state beyond that single write." Self-contradicting "read-only / DO write the report" wording from cycle 1/2 is gone. Templates mirror IDENTICAL by `cmp -s`. |
| No remaining contradiction inside the cross-review surface (pipeline + cross-review SKILL + adversarial prompt) | **Yes** | The prompt now grants exactly one write (the triage report); the pipeline launches with the permission level that allows that single write; the SKILL.md table documents what the pipeline actually invokes. The cycle-2 HIGH severity ("triage write silently dropped, cross-model gate slips past") is closed. |

### 2. Cycle-count rationale recorded

| Check | Result | Evidence |
| --- | --- | --- |
| `cycle-count.json` reflects raise to 3 | **Yes** | `.harness/state/standard-pipeline/cycle-count.json` contains: `{"plan_path": ".../2026-05-08-ralph-loop-codex-driver.md", "cycle": 3, "cap_raised_from": 2, "raise_reason": "cycle-2 cross-review HIGH P1: claude reviewer-inversion was launched with --permission-mode plan and could not write triage report; fix is 1-line + prompt wording"}`. |
| Operator-visible rationale field present (`raise_reason`) | **Yes** | The rationale is a verbose, audit-quality string — not just `"<override>"`. It names the failure mode, severity, root cause, and scope of the fix. This is the operator-evidence the cap-raising contract in `.claude/rules/post-implementation-pipeline.md` expects. |
| `cap_raised_from` semantically correct | **Yes** | Cycle-2 was the previous default cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`); cycle-3 is the consciously-raised one. The schema captures both the previous ceiling (2) and the new run (3). |

### 3. Cycle-3 prompt-only doc fix (commit `72e46e6`)

| Check | Result | Evidence |
| --- | --- | --- |
| Adversarial prompt now references `count_triage_findings` | **Yes** | `.claude/skills/cross-review/prompts/adversarial-claude.md:58-67` describes the parser as "`count_triage_findings` (`scripts/ralph-cli-driver.sh`), which prefers the canonical summary line `- After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N` and falls back to counting `\|` table rows under each `## <CATEGORY>` heading". The pre-cycle-3 stale `grep -c 'ACTION_REQUIRED'` description is gone. |
| Templates mirror updated | **Yes** | `cmp -s .claude/skills/cross-review/prompts/adversarial-claude.md templates/base/.claude/skills/cross-review/prompts/adversarial-claude.md` → IDENTICAL. |
| Mandates the canonical `- After triage:` summary line in output | **Yes** | New bullet at `:64-65`: "Always emit the `- After triage: ...` summary line in the header (the triage template already includes it)". This makes the helper's "canonical" claim actually contracted in the prompt — the cycle-2 self-review's MEDIUM #1 "summary-line is not codified anywhere" is now codified here, not just inferred from the template. |

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| **AC-1** `scripts/ralph-cli-driver.sh` exposes `run_agent`; fake-CLI assertions cover argv/stdin/cwd/output | **Met** | Wrapper unchanged in cycle 3. `count_triage_findings` (cycle-2 addition) untouched. |
| **AC-2** `RALPH_LOOP_DRIVER=codex --preflight --dry-run` green; codex Probe 5 active; claude branch keeps `claude_md_readable`+`json_output_format` | **Met** | Live re-run (this verify): both `RALPH_LOOP_DRIVER=claude` and `RALPH_LOOP_DRIVER=codex` preflight dry-runs exit 0 with the expected probe set on each side. |
| **AC-3** `<log>.json` synthesised for codex; sidecar contract unchanged | **Met** | No cycle-3 change in this surface. |
| **AC-4** `internal/cli/run.go` propagates `cfg.Loop.*` to env only when env unset | **Met** | No cycle-3 change. `go build ./...` / `go vet ./...` clean. |
| **AC-5** cross-review dispatcher: `driver=claude → codex exec review`, `driver=codex → claude -p` adversarial; reviewer/driver fields recorded | **Met (with downstream-doc drift, see §"Documentation drift")** | **[Δ cycle-3]** The dispatcher now launches the inverted Claude reviewer with `--permission-mode auto` (not `plan`). This is what AC-5 functionally requires — "must write the triage report ... `Reviewer:` field set" — because plan mode could never satisfy the write. The dispatcher itself, the cross-review SKILL.md table, and the adversarial prompt header are now self-consistent. AC-5 is met. **[caveat]** The `plan` flag literal still appears verbatim in 6 mirrored locations downstream of the dispatcher (Loop SKILL.md ×4, ralph-loop.md recipe ×2, Test 6b assertion ×2 lines in `tests/test-ralph-cli-driver.sh`). These do not break the contract — the dispatcher is the source of truth — but they are stale relative to it. See "Documentation drift" below. |
| **AC-6** `ralph status` / `ralph doctor` show effective driver with source | **Met** | No cycle-3 change. |
| **AC-7** Mirrored skills in lock-step | **Met** | `./scripts/check-skill-sync.sh` → `13 skill(s) in lock-step`. All four `cross-review/SKILL.md` mirrors and both `adversarial-claude.md` mirrors byte-identical. |
| **AC-8** `docs/recipes/ralph-loop.md` Codex driver section | **Met (with stale flag mention, see drift)** | The structural section is intact. The single legacy `--permission-mode plan` mention at `:228` (and template mirror) is doc drift relative to the cycle-3 dispatcher fix. |
| **AC-9** `./scripts/run-verify.sh` green; `tests/test-ralph-cli-driver.sh` green | **Likely met (static portion verified)** | `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` → exit 0; `bash -n tests/test-ralph-cli-driver.sh` syntax clean. **Caveat**: Test 6b at `tests/test-ralph-cli-driver.sh:209-213` still asserts `--permission-mode plan`. Because the test independently invokes a stub (it does not source the dispatcher), the existing assertion will continue to PASS — but it is no longer a regression guard for the dispatcher's true behaviour, since the dispatcher now uses `auto`. This is a *test-quality* drift (the test passes but tests the wrong thing), not a hard fail. `/test` should re-execute and `/sync-docs` should align Test 6b's expected literal. |
| **AC-10** `internal/config/config_test.go` covers `[loop] driver` allowlist | **Met** | No cycle-3 change. |
| **AC-11** Codex CLI walkthrough | **Deferred** | Same status as cycles 1+2. Operator/`/test` work, not a `/verify` blocker. |
| **AC-12** Backwards compat | **Met** | `( . scripts/ralph-config.sh && echo $RALPH_LOOP_DRIVER )` → `claude`; preflight under `claude` driver exits 0. No behavioural change for default users. |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `gofmt -l ./cmd ./internal` | **Pass** | empty output. |
| `go vet ./...` | **Pass** | empty output. |
| `go build ./...` | **Pass** | empty output. |
| `shellcheck -S warning scripts/ralph-pipeline.sh scripts/ralph-cli-driver.sh scripts/ralph-orchestrator.sh scripts/ralph-config.sh` | **Pass with pre-existing SC3045 in `ralph-orchestrator.sh:549`** | Not a cycle-3 regression; same finding present in cycles 1+2. |
| `sh -n scripts/ralph-pipeline.sh scripts/ralph-cli-driver.sh` | **Pass** | both syntactically valid. |
| `bash -n tests/test-ralph-cli-driver.sh` | **Pass** | (assertion-correctness drift discussed under AC-9.) |
| `./scripts/check-skill-sync.sh` | **Pass** | `13 skill(s) in lock-step`. |
| `./scripts/check-sync.sh` | **Pass** | `IDENTICAL: 145, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3`. |
| `HARNESS_VERIFY_MODE=static ./scripts/run-static-verify.sh` | **Pass** | All sub-verifiers green; check-pipeline-sync green for all 8 referenced files. |
| Live preflight: `RALPH_LOOP_DRIVER=claude --preflight --dry-run` | **Pass** | exit 0; `claude_md_readable` + `json_output_format` selected. |
| Live preflight: `RALPH_LOOP_DRIVER=codex --preflight --dry-run` | **Pass** | exit 0; `agents_md_readable` + `codex_exec_flags` selected. |

## Documentation drift

The cycle-3 fix updated three of the surfaces that document the inverted-Claude reviewer (`scripts/ralph-pipeline.sh` + templates mirror, `.claude/skills/cross-review/SKILL.md` + `.agents/` + templates mirrors, `.claude/skills/cross-review/prompts/adversarial-claude.md` + templates mirror). It did **not** update three other surfaces that mention the same flag:

| Drift site | What it says | What it should say (post-cycle-3) | Severity |
| --- | --- | --- | --- |
| `.claude/skills/loop/SKILL.md:170` | `--permission-mode plan` | `--permission-mode auto` | **MEDIUM** — Loop SKILL contradicts the cross-review SKILL it cross-references. |
| `.agents/skills/loop/SKILL.md:170` | `--permission-mode plan` | `--permission-mode auto` | **MEDIUM** — same contradiction; Codex side. |
| `templates/base/.claude/skills/loop/SKILL.md:170` | `--permission-mode plan` | `--permission-mode auto` | **MEDIUM** — same; ships in `ralph init` scaffold. |
| `templates/base/.agents/skills/loop/SKILL.md:170` | `--permission-mode plan` | `--permission-mode auto` | **MEDIUM** — same; Codex side scaffold. |
| `docs/recipes/ralph-loop.md:228` | "invokes `claude -p --permission-mode plan` against ..." | "invokes `claude -p --permission-mode auto` against ..." | **MEDIUM** — operator-facing recipe contradicts implementation. |
| `templates/base/docs/recipes/ralph-loop.md:228` | same | same | **MEDIUM** — ditto, ships to scaffolded projects. |
| `tests/test-ralph-cli-driver.sh:209-210, 213` | replays `claude -p ... --permission-mode plan ...` and asserts the call log contains `"--permission-mode plan"` | should replay with `--permission-mode auto` and assert that string instead | **MEDIUM** — Test 6b passes today (it invokes a stub independently of the dispatcher), but its assertion has stopped tracking the dispatcher's real behaviour. It will not regress-detect a future revert from `auto` to `plan`. |

These are flag-level spec drift (per the `feedback_spec_drift_at_flag_level.md` and `feedback_section_presence_across_mirrors.md` patterns in agent memory): the literal `--permission-mode <mode>` token is a contract surface that must be kept in lock-step across all places that quote it, not just where `check-skill-sync.sh` mechanically enforces parity. `check-skill-sync.sh` confirms cross-review SKILL.md mirrors are byte-identical — but it does not (and cannot) detect that `loop/SKILL.md` and the `cross-review/SKILL.md` documentation tell the operator different things about the same dispatcher.

The plan body itself (`docs/plans/active/2026-05-08-ralph-loop-codex-driver.md:87, 100, 119`) also still mentions `--permission-mode plan`. That is acceptable — plans are point-in-time design documents, and the plan AC checkbox lag pattern from agent memory tells us not to fail `/verify` on stale plan text. The plan should be sync-doc'd or noted as historical, but it is not the issue here.

The cycle-1/2 verify and walkthrough reports also mention `plan`. Those are point-in-time snapshots and should NOT be edited.

## Coverage gaps (handed off to /test)

1. **AC-9 behavioural** — `tests/test-ralph-cli-driver.sh` 48 assertions to run end-to-end. Crucially, **`/test` should also re-evaluate Test 6b as a contract test** rather than a stub-replay assertion: today it would pass even if the dispatcher silently regressed from `auto` to `plan`, because Test 6b replays its own choice of flags. A stronger test would either (a) source the dispatcher block and observe the actual `claude -p` argv, or (b) at minimum align the replay flag to `auto` so the assertion matches the contract.
2. **AC-11 walkthrough** — Real Codex driver walkthrough still not recorded.
3. **Documentation drift fix** — The 6 sites listed above need a small `/sync-docs` follow-up to flip `plan` → `auto`. None of them block dispatch correctness, but they make the project documentation internally inconsistent until corrected.

## Verdict

- **Pass** — the cycle-3 main contract (dispatcher launches the inverted Claude reviewer with `--permission-mode auto`, prompt header reconciled, cross-review SKILL.md table updated, cycle counter raised with audit-quality rationale) is met across the implementation surface and its primary documentation. AC-5's functional requirement ("must write the triage report") is now actually achievable, closing the cycle-2 HIGH severity gap.
- **Verified**: AC-1, AC-2, AC-3, AC-4, AC-5 (dispatcher), AC-6, AC-7 (cross-review SKILL mirrors lock-step), AC-10, AC-12; cycle-count rationale; commit `72e46e6` prompt update reflecting `count_triage_findings`.
- **Likely but unverified (handed to /test)**: AC-9 behavioural — 48+ assertions to be executed; Test 6b assertion correctness to be reconsidered.
- **Deferred**: AC-11 — Codex walkthrough.
- **Documentation drift (handed to /sync-docs)**: 6 mirrored sites still spell the flag as `plan` — `loop/SKILL.md` ×4 mirrors, `docs/recipes/ralph-loop.md` ×2 mirrors. Test 6b's expected literal also out of date.

**Pipeline decision**: **Continue.** Verdict is `pass` for the cycle-3 delta. The 6 drift sites are MEDIUM doc-drift (not contract-breaking) and should be cleaned up by `/sync-docs` in the same pipeline run rather than triggering another fix-and-revalidate cap raise.

## Smallest additional check that would raise confidence

Add a one-line grep guard to `scripts/run-static-verify.sh` (or to `scripts/check-skill-sync.sh`'s sibling) that fails on:

```sh
grep -RIn "permission-mode plan" \
  --include="*.sh" --include="*.md" \
  --exclude-dir=docs/reports --exclude-dir=docs/evidence \
  --exclude-dir=docs/plans \
  . && exit 1 || exit 0
```

That single check would have caught all 6 drift sites at static-analysis time and would prevent future flag-literal drift between dispatcher and prose docs without depending on the byte-cmp `check-skill-sync.sh` (which only proves *mirror parity*, not *cross-document consistency*).
