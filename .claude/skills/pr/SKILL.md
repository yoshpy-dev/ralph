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

0. **Resolve pinned plan identity** (standard flow):
   Read `.harness/state/standard-pipeline/active-plan.json` to obtain the exact plan path persisted by `/work`. Use this path for archival in Step 5 instead of rescanning `docs/plans/active/`. If the file is absent (e.g. Ralph Loop or legacy session), fall back to the single file under `docs/plans/active/` or ask the user which plan to archive.
1. Check for uncommitted changes with `git status --porcelain`.
   - **If uncommitted changes exist**: Stage with `git add` (prefer specific files over `-A`) and create a conventional commit: `<type>: <description>`. If a GitHub issue is linked, append `Refs #<number>` to the commit body.
   - **If working tree is clean** (intermediate commits already exist): Skip staging and committing — proceed directly to push.
2. Push the branch: `git push -u origin HEAD`.
3. Create the PR with `gh pr create` using [template.md](template.md) for the body structure. **PR title and body must be written in Japanese.**
4. For large diffs (>500 changed lines), create a walkthrough in `docs/reports/walkthrough-<date>-<slug>.md`.
5. Archive the plan using the path resolved in Step 0: `./scripts/archive-plan.sh <absolute-plan-path>`.
6. **Clear standard-pipeline state** (on successful PR creation):
   `rm -f .harness/state/standard-pipeline/active-plan.json .harness/state/standard-pipeline/cycle-count.json`.
   If PR creation fails, leave the state files in place so the user can resume.

## Completion gate

Do NOT present the PR as complete unless ALL of the following are true:

- [ ] PR created and URL displayed to the user
- [ ] Plan archived from `docs/plans/active/` to `docs/plans/archive/`
- [ ] For large diffs: walkthrough report exists
- [ ] Commit follows conventional commit format
- [ ] `.harness/state/standard-pipeline/` state files removed on success
