# Cross-review triage report: repo-wide-drift-fixes

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-repo-wide-drift-fixes.md
- Base branch: main
- Driver: claude / Reviewer: codex
- Triager: Claude Code (main context)
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|------------------|------------------|------------------|
| 1 | [P2] pipeline-outer.md base detection uses `HEAD@{upstream}` which resolves to `origin/<same-branch>` on a pushed feature branch, making `git diff base...HEAD` empty and causing the agent to see no changes | Real regression: on any branch tracked by a remote (i.e. the common case after `git push -u`), the upstream ref resolves to the same branch tip — not the repo default branch — so the diff is always empty. Fixed in all 4 copies (`.claude/skills/loop/prompts/`, `.agents/skills/loop/prompts/`, and both `templates/base/` counterparts) by switching to `git symbolic-ref --short refs/remotes/origin/HEAD` which resolves `origin/HEAD` → the repo default branch regardless of which feature branch is currently checked out. | `.claude/skills/loop/prompts/pipeline-outer.md`, `.agents/skills/loop/prompts/pipeline-outer.md`, `templates/base/.claude/skills/loop/prompts/pipeline-outer.md`, `templates/base/.agents/skills/loop/prompts/pipeline-outer.md` |
| 2 | [P3] `check-skill-sync.sh` check 6 uses `-maxdepth 1` when enumerating `prompts/` files, so nested files like `prompts/sub/foo.md` are silently excluded from both the inventory and byte-identical checks — a file present only on one side would pass undetected | Real contract gap: the check comment says "every file under each skill's prompts/ directory" but `-maxdepth 1` only covers the top level. Fixed by removing `-maxdepth 1` from both `find` calls, using relative paths via `sed "s\|^$dir/\|\|"` so nested paths like `sub/x.md` sort and compare correctly. Mirrored byte-identically to `templates/base/scripts/check-skill-sync.sh`. Two new test cases added (L: nested file missing on mirror → fail; M: nested file byte-identical on both → pass). | `scripts/check-skill-sync.sh`, `templates/base/scripts/check-skill-sync.sh`, `tests/test-check-skill-sync.sh` |

## Result

Both findings are real bugs confirmed by code inspection. Fixes applied in this
commit as cycle-2 re-run of the cross-review fix pass. All verification gates
pass after fix: `test-check-skill-sync.sh` (13/13), `check-skill-sync.sh` (PASS),
`check-sync.sh` (PASS), `run-verify.sh` (PASS).

## Cycle 2 (2026-07-12)

- Driver: claude / Reviewer: codex / Cycle: 2/2 (cap reached)
- Cycle-1 fixes applied in 501d164 (+ 3706452 header list); re-run:
  self-review MERGE (HIGH recorded as tech-debt follow-up, not a diff
  defect), verify PASS, test PASS (13/13 gate cases + full regression 0
  failures).
- Reviewer result: no findings — "The changed scripts, prompt mirrors,
  help/comment updates, and documentation sync appear consistent with the
  intended drift fixes. I did not find an introduced bug."
- Known follow-up carried in docs/tech-debt/README.md: the same
  HEAD@{upstream} base-detection weakness exists at the sites that actually
  gate cross-review (scripts/ralph-pipeline.sh:807 + cross-review SKILL.md)
  — out of this plan's Non-goals (no behavior changes); separate fix PR
  recommended.
