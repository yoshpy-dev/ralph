# Codex triage report: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2 (standard-pipeline state files absent — plan introduces them; fallback mode)
- Total Codex findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Self-review report: `docs/reports/self-review-2026-04-23-pipeline-max-cycles-cap.md`
- Verify report: `docs/reports/verify-2026-04-23-pipeline-max-cycles-cap.md`
- Implementation context summary: Introduces a 2-run cap on the post-implementation pipeline. New state files `.harness/state/standard-pipeline/active-plan.json` and `cycle-count.json` persist plan identity and cycle counter across sessions. The cap-reached branch of `/codex-review` is supposed to keep user agency by dropping only the "fix" option and asking about raise-cap / PR / abort.

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | `/work` Step 0.5.d unconditionally rewrites `cycle-count.json` to `{"cycle": 1}` on every invocation. Cross-session resume erases the persisted count and gives the user two fresh pipeline runs, defeating the cap in exactly the scenario (context compaction) the plan's Assumption says file-based persistence is needed for. | **Real issue: yes.** Plan line 36 explicitly adopts file-based persistence because session memory is lost. Finding 1 directly contradicts that rationale. **Worth fixing: yes.** This is the main cross-session invariant the change was introduced to protect. Fix: only write `cycle=1` when the file is absent OR the recorded `plan_path` differs; otherwise keep the existing counter (or ask via AskUserQuestion). | `.claude/skills/work/SKILL.md`, `templates/base/.claude/skills/work/SKILL.md` |
| 2 | `/codex-review` Case B's `CAP_REACHED` branch says "Skip the re-run option and proceed directly to /pr", silently bypassing AskUserQuestion. Rule file line 40 and plan line 40 both require the cap-reached flow (raise / PR / abort) to apply to both Case A and Case B, with only the "fix" option dropped. | **Real issue: yes.** Documented contract mismatch between rule+plan vs skill. **Worth fixing: yes.** Removes user agency in the exact state the cap is meant to expose. Fix: mirror the Case A cap-reached AskUserQuestion for Case B (raise cap / PR / abort), wording adjusted for WORTH_CONSIDERING (no ACTION_REQUIRED remaining). | `.claude/skills/codex-review/SKILL.md`, `templates/base/.claude/skills/codex-review/SKILL.md` |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|

_(none)_

## DISMISSED

| # | Codex finding | Dismissal reason | Category |
|---|---------------|------------------|----------|

_(none)_

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
