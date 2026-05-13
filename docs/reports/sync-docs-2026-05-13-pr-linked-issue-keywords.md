# Sync-docs report: pr-linked-issue-keywords

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-pr-linked-issue-keywords.md`
- Maintainer: Codex
- Scope: Documentation and template drift for `/pr` linked issue handling.

## Documentation Updates

| File | Status | Notes |
| --- | --- | --- |
| `.claude/skills/pr/SKILL.md` | Updated | Documents `Closes` vs `Refs` selection. |
| `.agents/skills/pr/SKILL.md` | Updated | Mirrors Claude skill body. |
| `.claude/skills/pr/template.md` | Updated | Uses `Closes #__ISSUE__` and guidance comment. |
| `.agents/skills/pr/template.md` | Updated | Mirrors Claude template. |
| `templates/base/.../pr/*` | Updated | Keeps scaffolded projects aligned. |

## Drift Checks

- `scripts/check-sync.sh`: PASS.
- `./scripts/check-skill-sync.sh`: PASS.

## Verdict

Pass.
