---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Accepts an optional GitHub issue number or URL for context pre-fill. Creates a feature branch after plan approval. Manual trigger only.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Write, Edit, Bash, AskUserQuestion
---
Create or update a plan in `docs/plans/active/`.

## Goals

- Turn a request into a versioned plan that survives context loss
- Define acceptance criteria and evidence before deep implementation
- Make later review and verification cheaper

## Steps

1. Read `AGENTS.md`, `CLAUDE.md`, relevant `.claude/rules/`, and existing active plans.
2. Inspect only enough code and docs to understand the request and blast radius.
2.5. If a GitHub issue number or URL is provided:
     a. `gh issue view <number> --json title,body,labels,number`
     b. Pre-fill: Objective from title, Related request from body, Related issue: #N
     c. If no issue provided: set "Related issue: N/A"
3. Choose one active plan file. If none exists, create one with `./scripts/new-feature-plan.sh <slug> [issue-number]` or from [template.md](template.md).
4. Fill in:
   - objective
   - scope and non-goals
   - assumptions
   - affected files or systems
   - acceptance criteria
   - implementation outline
   - verify plan (static analysis checks, spec compliance criteria, documentation drift checks, evidence to capture)
   - test plan (unit tests, integration tests, regression tests, edge cases, evidence to capture)
   - risk register
   - rollout or rollback notes
   - evidence targets
5. Keep the plan high-level enough to avoid cascading low-level mistakes.
6. End with a short readiness checklist.
7. Create feature branch:
   a. Determine type from context (feat/fix/docs/refactor/chore).
   b. With issue: `git checkout -b <type>/<issue>/<slug>`
   c. Without issue: `git checkout -b <type>/<slug>`
   d. Slug = plan filename slug. Do NOT push yet.

8. **Flow selection**: Use **AskUserQuestion** to ask the user which execution flow to use.
   - Question: "どちらの開発フローで進めますか？"
   - Options:
     1. **標準フロー (/work)** — Claude Code 内で対話的に実装を進める（短〜中規模タスク向け）
     2. **Ralph Loop (/loop)** — ターミナルで自律反復実行する（大規模・持続的自律作業向け）
   - If the plan mentions large-scale refactoring, migration, test-coverage campaigns, or multi-file autonomous work, recommend Ralph Loop.
   - Otherwise, recommend 標準フロー.
   - After the user chooses, state which flow will be used next and proceed accordingly.

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Feature branch created and checked out
- Chosen execution flow (standard /work or Ralph Loop /loop)

## Anti-bottleneck

Before asking the user for confirmation or choices during planning, first check whether the answer is available from the codebase, existing plans, docs, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

- [template.md](template.md)
