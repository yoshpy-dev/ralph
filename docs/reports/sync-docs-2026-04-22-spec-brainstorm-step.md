# Sync-docs report: spec brainstorm step

- Date: 2026-04-22
- Plan: (none — docs-only change; acceptance criteria supplied in-channel and echoed in the verify report)
- Branch: `docs/spec-brainstorm-step`
- Maintainer: `doc-maintainer` subagent (Claude Code)
- Upstream triggers:
  - `docs/reports/self-review-2026-04-22-spec-brainstorm-step.md` — 4 LOW findings, no blockers, merge recommended.
  - `docs/reports/verify-2026-04-22-spec-brainstorm-step.md` — PASS; AC1–AC5 all verified; flagged `docs/architecture/repo-map.md:29` as optional polish.
  - `docs/reports/test-2026-04-22-spec-brainstorm-step.md` — PASS; docs-only, no behavioral tests required.

## Scope

Confirm that product and harness-internal documentation reflect the `/spec` skill's new Step 2 (Brainstorm / 壁打ち), the renamed Step 5 (Clarify residual requirements), and the shift from 8 to 9 steps. The source-of-truth update (`.claude/skills/spec/SKILL.md` plus its `templates/base/` mirror) and the surface-level description updates (`CLAUDE.md`, `AGENTS.md`, `README.md`, and their `templates/base/` mirrors) were already made on this branch and are out of scope for this report.

This pass focuses on drift sweep only: anywhere the repo still describes `/spec`'s mechanism, step count, or interaction model in a way that pre-dates brainstorming.

Scan methodology:

- `grep /spec` repo-wide (25 matching files).
- `grep 'Clarify requirements' | 'step 8.*spec' | '8 step' | '8-step' | 'eight step'` — no matches outside this branch's own reports.
- `grep 'Understand the request'` — only the new Step 1 heading and the reports that cite it.
- `grep 'codebase exploration' | 'interactive clarification' | 'brainstorm' | '壁打ち'` — all surviving hits now carry the new "iterative brainstorming" phrasing.

## Files updated

| File | Change | Reason |
| --- | --- | --- |

_(No product or harness files needed updates in this pass. See "Intentionally left alone" below for the full set of candidates considered.)_

## Intentionally left alone

| Area | Why not changed |
| --- | --- |
| `.claude/skills/spec/SKILL.md` and `templates/base/.claude/skills/spec/SKILL.md` | Source of truth for `/spec`; already updated on this branch and byte-identical. Skill source is explicitly out of `/sync-docs` scope per the task brief. |
| `CLAUDE.md`, `AGENTS.md`, `README.md`, `templates/base/CLAUDE.md`, `templates/base/AGENTS.md` | Already updated on this branch with "iterative brainstorming (壁打ち)" phrasing (verify AC5 verified). No further drift. |
| `docs/architecture/repo-map.md:29` — `.claude/skills/spec/: refine vague ideas into detailed specifications (manual trigger)` | **Decided: do not update.** The entry follows the same one-line purpose-only pattern as every other skill on L29–L41 (plan, work, loop, self-review, verify, test, codex-review, pr, sync-docs, audit-harness, anti-bottleneck, release) — none mention *how* they work. Appending "expands sparse inputs via iterative brainstorming (壁打ち)" would (a) break the map's consistent terseness, (b) make `/spec` the only skill with a mechanism hint, and (c) violate `.claude/rules/documentation.md` ("Keep `AGENTS.md` as a map, not an encyclopedia" — applies by analogy to `repo-map.md`, which is explicitly a "Repo map"). The verify report already classified this as optional polish consistent with the "map, not encyclopedia" rule; this pass confirms that judgment. |
| `.claude/rules/subagent-policy.md:40` + `templates/base/.claude/rules/subagent-policy.md:40` | Says `/spec` "relies heavily on `AskUserQuestion` for requirement clarification (active back-and-forth with the user) and on `AskUserQuestion` for output selection". This is still accurate post-change — it captures the policy reason (why `/spec` stays inline and is not delegated to a subagent) and holds for both Step 2 (brainstorming) and Step 5 (clarification), which are the two `AskUserQuestion`-heavy steps. No edit needed; the rule is mechanism-agnostic and the claim "relies heavily on `AskUserQuestion`" is now *more* true, not less. |
| `.claude/skills/spec/template.md` | Describes the structure of the **output spec document** (概要 / 背景 / 要件 / 受け入れ基準 / etc.), not the `/spec` workflow. Step-number-agnostic by construction; no update needed. |
| `docs/quality/definition-of-done.md` | Does not describe `/spec` mechanism; only references "spec compliance" in a general verify context. Unchanged. |
| `docs/tech-debt/README.md` | The only `/spec`-adjacent hit (line 22) is a file-path reference (`docs/specs/2026-04-16-ralph-cli-tool.md`), unrelated to `/spec` workflow mechanics. |
| `scripts/check-sync.sh` | Lists `docs/specs/` as an intentional root-only path exclusion. No workflow description. |
| `README.md:117` (slash-command one-liner `/spec (optional) → /plan → …`) | Sequence indicator only; does not describe `/spec`'s mechanism. Updating here would be over-description (the Operating Loop §1 at L129–133 already carries the full mechanism description). |
| `CLAUDE.md:9` (parenthetical "refine vague ideas") | Process-role label identifying which manual-trigger skills exist, not a mechanism description. The full mechanism description is on the very next line (L10); duplicating "brainstorming" here would be over-description. Same judgment applied and documented in the verify report. |
| Archived plans (`docs/plans/archive/*`) and archived reports | Frozen by convention; historical content is intentionally preserved. |
| Other `/spec`-mentioning reports (`docs/reports/*-2026-04-21-*.md`, `docs/reports/*-2026-04-22-upgrade-detect-local-edits*.md`) | These reference `docs/specs/<file>` as a path and predate this change. Reports are historical records; they describe the repo state at a past point in time and are not rewritten on every contract change. |
| Implementation and test files | Explicitly out of scope for `/sync-docs` (docs-only branch; no code changed). |

## Cross-reference checks performed

- `grep '/spec' -r` → 25 files; each manually classified (above) as either (a) already updated by this branch, (b) mechanism-agnostic reference, (c) historical report/archive, or (d) path reference to `docs/specs/`.
- `grep -i 'Clarify requirements'` — only within the self-review/verify reports on this branch that cite the absence of stale hits. Confirmed no stale references to the old Step 5 name.
- `grep -i '8 step|8-step|eight step'` — zero hits outside the branch's own self-review/verify reports (which quote the absence). No stale step-count references.
- `grep 'Understand the request'` — exactly 4 hits, all legitimate: the new Step 1 heading in `.claude/skills/spec/SKILL.md:31`, its mirror in `templates/base/…:31`, and two report citations on this branch.
- `grep 'codebase exploration|interactive clarification'` — all 4 active-content hits (CLAUDE.md, AGENTS.md, and their two template mirrors) now carry the "iterative brainstorming" prefix. The remaining hits are in this branch's own reports.
- `grep 'brainstorm|壁打ち'` — new term correctly localized to the 7 files this branch touched plus this branch's reports. Not leaking into unrelated surfaces.
- `scripts/check-sync.sh` run transitively by `/verify`: `IDENTICAL=107 / DRIFTED=0 / ROOT_ONLY=0 / TEMPLATE_ONLY=9 / KNOWN_DIFF=3`. No mirror drift.
- Mirror parity spot-check: `diff .claude/skills/spec/SKILL.md templates/base/.claude/skills/spec/SKILL.md` → byte-identical (confirmed in verify AC4).

## Harness-internal sync checklist

Walked through the sync-docs skill checklist explicitly:

- **Skills added/removed/renamed**: none. `/spec` still exists with the same name, same manual-trigger status, same frontmatter shape. `AGENTS.md` Repo map and `README.md` Operating Loop both still list it correctly.
- **Hooks added/removed**: none.
- **Rules added/removed**: none. `.claude/rules/subagent-policy.md` still correct (see "Intentionally left alone" above).
- **Language packs added/removed**: none.
- **Scripts added/removed**: none. `README.md` Quick Start's `./scripts/run-verify.sh` reference is still valid.
- **Quality gates changed**: none. Pipeline order unchanged; `docs/quality/definition-of-done.md` still accurate.
- **PR skill consistency**: unchanged — `/spec` sits before `/plan` in the loop and does not interact with `/pr` contracts.

All checklist items pass without edits.

## Evidence

- Self-review: `docs/reports/self-review-2026-04-22-spec-brainstorm-step.md`
- Verify: `docs/reports/verify-2026-04-22-spec-brainstorm-step.md`
- Test: `docs/reports/test-2026-04-22-spec-brainstorm-step.md`
- Source-of-truth files (already updated in this branch, untouched by this report):
  - `.claude/skills/spec/SKILL.md`
  - `templates/base/.claude/skills/spec/SKILL.md`
  - `CLAUDE.md:10`, `templates/base/CLAUDE.md:10`
  - `AGENTS.md:22`, `templates/base/AGENTS.md:19`
  - `README.md:131`

## Verdict

Documentation is in sync with the source-of-truth `/spec` SKILL.md. No drift found outside the files already updated on this branch. The one item flagged by `/verify` (`docs/architecture/repo-map.md:29`) is intentionally left alone: the project's own documentation rule (`.claude/rules/documentation.md` — "Keep `AGENTS.md` as a map, not an encyclopedia") argues against expanding one-line map entries with mechanism detail, and the entry's purpose is still accurate as stated.

Ready for `/codex-review` (optional) and `/pr`.
