---
name: pr
description: Create a pull request after self-review, verify, and test pass. Handles branch push, PR creation, plan archiving, and hand-off. Invoke automatically after /codex-review completes (or is skipped), when self-review, verify, and test reports all exist with passing verdicts.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Create a PR to hand off completed work for human review and merge.

## Pre-checks

Before creating a PR, confirm ALL of the following:

1. A self-review report exists in `docs/reports/` with no CRITICAL findings.
2. A verify report exists in `docs/reports/` with pass or partial-pass verdict.
3. A test report exists in `docs/reports/` with pass verdict. **If tests failed, do NOT create the PR.**
4. Raw evidence is saved in `docs/evidence/`.
5. Branch name follows `<type>/<issue>/<slug>` or `<type>/<slug>` format.
6. You are NOT on main or master.
7. `gh` CLI is available (if not, provide manual commands instead).

If any pre-check fails, stop and explain what is missing.

## Steps

1. Stage changes with `git add` (prefer specific files over `-A`).
2. Create a conventional commit: `<type>: <description>`. If a GitHub issue is linked, append `Refs #<number>` to the commit body.
3. Push the branch: `git push -u origin HEAD`.
4. Create the PR with `gh pr create` using [template.md](template.md) for the body structure.
5. For large diffs (>500 changed lines), create a walkthrough in `docs/reports/walkthrough-<date>-<slug>.md`.
6. Archive the plan: `./scripts/archive-plan.sh <slug>`.

## Completion gate

Do NOT present the PR as complete unless ALL of the following are true:

- [ ] PR created and URL displayed to the user
- [ ] Plan archived from `docs/plans/active/` to `docs/plans/archive/`
- [ ] For large diffs: walkthrough report exists
- [ ] Commit follows conventional commit format
