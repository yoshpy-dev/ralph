# Codex triage report: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached — fallback mode, no state files on disk)
- Total Codex findings (cumulative): 4 (cycle 1: 2, cycle 2: 2)
- After triage (cycle 2): ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Self-review report: `docs/reports/self-review-2026-04-23-pipeline-max-cycles-cap.md` (cycle 1 + cycle 2 addendum)
- Verify report: `docs/reports/verify-2026-04-23-pipeline-max-cycles-cap.md` (cycle 1 PASS + cycle 2 PASS)
- Cycle 1 findings: both ACTION_REQUIRED, fixed in commit `e27102a`.

## ACTION_REQUIRED (cycle 2)

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 3 | `/work` Step 0 (a/b/e) reads "the active plan" and rewrites `Branch:` in plan file BEFORE Step 0.5 asks the user to pick among multiple active plans. With >1 plan, branch naming or plan-file writes can land on the wrong plan. | **Real issue: yes.** Plan-identity guarantee is voided when Step 0 runs with an ambiguous "active plan" selection. **Worth fixing: yes.** P1 from Codex; central invariant of this PR. Fix: move plan selection to run before Step 0 operations (renumber), or fold selection into Step 0.a itself. | `.claude/skills/work/SKILL.md`, `templates/base/.claude/skills/work/SKILL.md` |
| 4 | Cap-override: if cycle=cap and user raises cap by +1, Step 7 increments cycle, tripping the new cap immediately. User must raise cap by at least +2 to actually receive one extra pass, but prompt does not say so. | **Real issue: yes.** Off-by-one in the cap-override UX — the documented "raise the cap and re-run" does not actually deliver a rerun. **Worth fixing: yes.** P2 from Codex. Fix: either defer increment until after the rerun, or strengthen the prompt to tell the user to set `RALPH_STANDARD_MAX_PIPELINE_CYCLES > current cycle` (e.g. if cycle=2, set to 3 AND the flow will increment to 3 before rerun → still tripped; safer fix is "increment AFTER the rerun's /codex-review starts"). | `.claude/skills/codex-review/SKILL.md`, `templates/base/.claude/skills/codex-review/SKILL.md` |

## WORTH_CONSIDERING

_(none)_

## DISMISSED

_(none)_

## Cycle 1 (historical)

Findings 1 and 2 were both ACTION_REQUIRED, fixed in commit `e27102a`:
- #1 cycle-counter reset on resume (work SKILL Step 0.5.d)
- #2 Case B cap-reached skipped AskUserQuestion (codex-review SKILL)

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
