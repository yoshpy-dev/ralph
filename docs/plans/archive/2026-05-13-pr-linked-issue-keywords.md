# pr-linked-issue-keywords

- Status: Draft
- Owner: Codex
- Date: 2026-05-13
- Related request: pr-linked-issue-keywords
- Related issue: 58
- Branch: issue-58-pr-linked-issue-keywords

## Objective

Update the `/pr` skill contract and PR body template so completed issue work
uses GitHub closing keywords by default, while partial or related work remains
able to use `Refs`.

## Scope

- `.claude/skills/pr/SKILL.md`
- `.agents/skills/pr/SKILL.md`
- `.claude/skills/pr/template.md`
- `.agents/skills/pr/template.md`
- Plan, review, verify, test, and sync-docs artifacts for this PR

## Non-goals

- Retrospective edits to existing merged PRs or archived plans.
- Commit message issue-link behavior.
- Automatic semantic detection of whether an issue is fully closed.

## Assumptions

- `/pr` still resolves issue numbers from branch names or plan metadata.
- Human/agent judgment decides whether a PR fully closes an issue.

## Affected areas

- PR creation workflow documentation
- Claude/Codex skill parity checks

## Design decisions

Critical forks: None. The issue scope is documentation/template guidance only.

## Acceptance criteria

- [ ] PR body template defaults related issue output to `Closes #__ISSUE__`.
- [ ] `/pr` skill documents when to keep `Closes` and when to switch to `Refs`.
- [ ] GitHub closing keyword caveats are documented in plain text guidance.
- [ ] `.claude/skills/pr/` and `.agents/skills/pr/` stay in sync.

## Implementation outline

1. Update the PR templates on both Claude and Codex sides.
2. Add linked-issue handling guidance to both `/pr` skill bodies.
3. Run skill sync/static verification and write reports.

## Verify plan

- Static analysis checks: `./scripts/check-skill-sync.sh`, `git diff --check`.
- Spec compliance criteria to confirm: each acceptance criterion above.
- Documentation drift to check: no unrelated workflow docs need updates.
- Evidence to capture: command outputs in `docs/evidence/`.

## Test plan

- Unit tests: none; skill/template docs only.
- Integration tests: `./scripts/run-static-verify.sh`.
- Regression tests: `./scripts/check-skill-sync.sh`.
- Edge cases: partial issue work uses `Refs`; keyword outside comments/code/quotes.
- Evidence to capture: verify/test reports with pass verdicts.

## Risks and mitigations

- Risk: accidentally closing an issue on partial work. Mitigation: explicit
  `Closes` vs `Refs` rule in skill guidance and template comments.
- Risk: Claude/Codex skill drift. Mitigation: run skill sync check.

## Rollout or rollback notes

Rollback by reverting this docs/template PR.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created: https://github.com/yoshpy-dev/ralph/pull/61
