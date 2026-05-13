# Verify report: pr-linked-issue-keywords

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-pr-linked-issue-keywords.md`
- Verifier: Codex
- Scope: Acceptance criteria, skill parity, and static checks.

## Results

| Check | Result | Evidence |
| --- | --- | --- |
| `./scripts/check-skill-sync.sh` | PASS | 13 skills in lock-step. |
| `scripts/check-sync.sh` | PASS | Root/template mirrors in sync. |
| `git diff --check` | PASS | No whitespace errors. |
| `./scripts/run-static-verify.sh` | PASS | `docs/evidence/verify-2026-05-13-102600.log` |

## Acceptance Criteria

- PR body template defaults related issue output to `Closes #__ISSUE__`: PASS.
- `/pr` skill documents when to keep `Closes` and when to switch to `Refs`: PASS.
- GitHub closing keyword caveats are documented in normal guidance: PASS.
- `.claude/skills/pr/`, `.agents/skills/pr/`, and template mirrors stay in sync: PASS.

## Coverage Gaps

- No live GitHub merge auto-close was exercised; this is a template/skill contract change and the behavior is provided by GitHub.

## Verdict

Pass.
