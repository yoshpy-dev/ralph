# Sync-docs report: Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Branch: `feat/44/ralph-loop-codex-driver`
- Maintainer: `doc-maintainer` subagent (Claude Opus 4.7, 1M context)
- Inputs read: self-review (`docs/reports/self-review-2026-05-08-ralph-loop-codex-driver.md`), verify (`docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md`), test (`docs/reports/test-2026-05-08-ralph-loop-codex-driver.md`)

## Files touched

| Path | Reason |
| --- | --- |
| `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md` | Add a progress-checklist tick reflecting the AC-6 follow-up (commit `3351df2` — `ralph status` driver line + AGENTS.md primary-loop paragraph) so the checklist matches the landed implementation. |
| `.claude/rules/subagent-policy.md` | Tighten the Codex-detection paragraph to say "in the standard flow" (it had been silent on the Loop side), and rewrite the "Post-implementation pipeline for /loop" paragraph so the rule explicitly covers the driver-aware `run_agent` wrapper, the cross-review reviewer inversion, and the env > TOML > default precedence for `RALPH_LOOP_DRIVER`. The rule now describes both `RALPH_PRIMARY_CLI` (standard flow) and `RALPH_LOOP_DRIVER` (Loop) symmetrically. |
| `templates/base/.claude/rules/subagent-policy.md` | Same edits mirrored to the scaffold copy so `scripts/check-sync.sh` and `scripts/check-skill-sync.sh` stay green. Verified via `run-verify.sh`. |
| `docs/tech-debt/README.md` | (a) Strike through and annotate the row "`ralph-pipeline.sh` Outer Loop still hard-codes `codex exec review`" — this PR is exactly the work that resolves it (dispatcher routes through `pick_reviewer`, variables renamed `_xreview_log`/`_reviewer`, log line updated). Row is preserved with `~~strikethrough~~` + a `(RESOLVED 2026-05-08 in feat/44/ralph-loop-codex-driver: …)` note for traceability, matching the pattern used for the resolved `probeBinary` entry. (b) Add two new rows for the MEDIUM self-review findings the implementation deliberately deferred: Probe 1 / Probe 7 dropping the `--version` round-trip, and the unreachable `pick_reviewer` third-arm fallback. Both rows cite `docs/reports/self-review-2026-05-08-ralph-loop-codex-driver.md` as the source. |

## Files deliberately not touched

| Path | Why no edit was needed |
| --- | --- |
| `AGENTS.md` (root) | Already updated in commit `3351df2` (line 36 carries the new "Ralph Loop runs under whichever CLI is selected by `RALPH_LOOP_DRIVER`…" paragraph). The wording is concise (one paragraph), still under the documentation.md "map, not encyclopedia" guidance, and aligned with `templates/base/AGENTS.md`. |
| `templates/base/AGENTS.md` | Already updated in `3351df2` (lines 39-42). Wording matches the root file's intent; structural differences with the root are pre-existing template-vs-meta-repo split. |
| `CLAUDE.md` (root + templates/base) | Phase 2 introduces no Claude-specific behaviour change — the driver knob is a Loop-side / orchestrator-side concern. CLAUDE.md is already smaller than AGENTS.md per the documentation rule. |
| `README.md` | Already updated in `6b2dd53` (lines 217-222). Anchor link `docs/recipes/ralph-loop.md#running-loop-under-the-codex-driver` matches the recipe heading "### Running Loop under the Codex driver" (Markdown converts to lowercase + hyphens). Verified by `grep`. |
| `docs/recipes/ralph-loop.md` | Already comprehensive: lines 164-209 cover `codex trust .`, env-var invocation, `[loop]` TOML override, env > TOML priority, what changes inside the pipeline, preflight Probe 5, and the cross-review reviewer inversion. Mirror in `templates/base/docs/recipes/ralph-loop.md` is identical (verified live in `/verify`). |
| `docs/quality/definition-of-done.md` | Already updated in `6b2dd53` (lines 45-49) with "Driver selection (Phase 2 / issue #44)" callout that names env > `[loop] driver` > default precedence and the cross-review reviewer inversion. |
| `.claude/rules/post-implementation-pipeline.md` (+ templates/base mirror) | Pipeline order did not change in Phase 2; only the wrapper underneath the per-step `claude -p` calls became driver-aware. The rule still names the canonical order correctly and the "Pipeline cycle cap" / "Integration pipeline" sections still apply unmodified. Adding a driver paragraph here would duplicate `subagent-policy.md`'s newly tightened text. |
| `.claude/skills/loop/SKILL.md` ↔ `.agents/skills/loop/SKILL.md` | Both already document `RALPH_LOOP_DRIVER=codex ./scripts/ralph run` and the reviewer inversion; `cmp` byte-identical with `templates/base/` mirrors. `check-skill-sync.sh` reports 13 skills in lock-step. |
| `.claude/skills/cross-review/SKILL.md` ↔ `.agents/skills/cross-review/SKILL.md` | Both describe the Loop reviewer-inversion table; verified byte-identical in `/verify`. |
| `.claude/skills/cross-review/prompts/adversarial-claude.md` | Mirrored to `templates/base/.claude/skills/cross-review/prompts/`. Not mirrored to `.agents/skills/cross-review/prompts/` deliberately — Codex driver inverts to a Claude reviewer, so Codex never invokes this prompt. The verifier explicitly accepted this asymmetry; a mirror would be dead weight. `check-skill-sync.sh` enforces SKILL.md body parity, not arbitrary subdirs. |
| `templates/base/ralph.toml` | The `[loop]` section is already present (verified by `internal/scaffold/embed_test.go::TestTemplateBaseRalphTomlHasLoopSection`). The self-review's LOW finding about tightening the priority comment ("env wins / Go side propagates only when env is unset") is a quality-of-comment improvement — declined here to keep the sync-docs surface minimal; folded into the self-review LOW backlog if a future commit revisits the file. |
| `scripts/check-skill-sync.sh` enforcement coverage | The new `adversarial-claude.md` prompt is mirrored only to `templates/base/.claude/`. Verifier accepted the asymmetry (Codex never calls it); leaving enforcement coverage as-is. `run-verify.sh` confirms `check-skill-sync.sh` reports 13 skills in lock-step. |

## Cross-references re-checked

- `docs/recipes/ralph-loop.md#running-loop-under-the-codex-driver` resolves: README.md:221 link target matches the recipe's H3 heading on line 164.
- `cmp -s scripts/ralph-cli-driver.sh templates/base/scripts/ralph-cli-driver.sh` and the same for `ralph-pipeline.sh`, `ralph-config.sh`, `ralph-orchestrator.sh`, `.claude/skills/loop/SKILL.md`, `.agents/skills/loop/SKILL.md`, `.claude/skills/cross-review/SKILL.md`, `.agents/skills/cross-review/SKILL.md`, `adversarial-claude.md` — all byte-identical (verified earlier in `/verify` and re-confirmed by `check-sync.sh` running inside `run-verify.sh`).
- The strike-through resolution of the Outer-Loop hardcoding tech-debt row is corroborated by `scripts/ralph-pipeline.sh:761,774,780,792` (variables now `_xreview_log` / `_reviewer`, log line "Running cross-review (driver=…, reviewer=…)…", dispatcher calls `pick_reviewer`).
- Open Questions on the plan still says "なし" (line 196 of the plan); no edit needed there.

## Final verify verdict

`./scripts/run-verify.sh` → **PASS** (exit 0). Evidence at `docs/evidence/verify-2026-05-08-034528.log`.

- check-sync: IDENTICAL 145, DRIFTED 0
- check-pipeline-sync: PASS
- check-skill-sync: 13 skill(s) in lock-step
- shellcheck/sh -n/jq/bash -n: all PASS (pre-existing info-level findings unchanged)
- mojibake guard: 11/11 cases PASS
- ralph-cli-driver fake-CLI test: 38/38 assertions PASS
- gofmt / go vet / go test: PASS across all 12 packages
- preflight smokes: both `RALPH_LOOP_DRIVER=claude` and `RALPH_LOOP_DRIVER=codex` PASS

Doc sync did not introduce drift in any tracked surface. Pipeline can proceed to `/cross-review` and then `/pr`.
