---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Accepts an optional GitHub issue number or URL for context pre-fill. Resolves high-leverage implementation forks with the user before finalizing. Does not create a branch — branch/worktree creation is deferred to the chosen flow skill.
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
2.7. **Flow selection**: Use **AskUserQuestion** to ask the user which execution flow to use.
   - Question: "どちらの開発フローで進めますか？"
   - Options:
     1. **標準フロー (/work)** — Claude Code 内で対話的に実装を進める（短〜中規模タスク向け）
     2. **Ralph Loop (/loop)** — ディレクトリベースプランで並列スライス自律実行する（大規模・分割可能タスク向け）
   - If the plan mentions large-scale refactoring, migration, test-coverage campaigns, or multi-file autonomous work, recommend Ralph Loop.
   - After the user chooses, proceed to step 3.
3. Choose one active plan file based on the flow selected in 2.7:
   - **標準フロー**: Create with `./scripts/new-feature-plan.sh <slug> [issue-number]` or from [template.md](template.md).
   - **Ralph Loop**: Create with `./scripts/new-ralph-plan.sh <slug> [issue-number] [slice-count]` to generate a directory-based plan structure under `docs/plans/active/<date>-<slug>/`.
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
4.5. **Critical forks (convergent)**: After the initial draft is in place, scan the plan for "critical forks" — implementation decisions that meet **all three** of:
   - Two or more approaches differ materially in risk, cost, or rollback profile
   - The choice cannot be resolved from the codebase, existing `.claude/rules/`, docs, or a reasonable default
   - Reversing the decision mid-implementation would cost more than roughly one slice of rework

   For each critical fork identified:
   a. Use `AskUserQuestion` with one focused question and 2-4 concrete options. Each option must briefly state its pros/cons so the user can choose informedly.
   b. Record the chosen approach and its rationale in the plan's "Design decisions" section (see [template.md](template.md)).
   c. If the chosen option invalidates other plan sections (outline, risks, affected files), revise them before continuing.

   If no critical forks exist after scanning, write "Critical forks: なし" in the Design decisions section and proceed.

   **Do NOT ask about**:
   - Stylistic or easily reversible choices (internal naming, helper placement inside an established module pattern)
   - Decisions already settled by `.claude/rules/`, `AGENTS.md`, or the upstream `/spec` output
   - The flow-level choice already made in step 2.7
   - Anything a reasonable default + explicit assumption would cover

   Purpose is **convergent** — narrow between enumerated options, not expand the design space. Divergent ideation belongs to `/spec`, not here.

5. Keep the plan high-level enough to avoid cascading low-level mistakes.
6. End with a short readiness checklist.
6.5. **Codex plan advisory (optional)**:
   a. Run `./scripts/codex-check.sh` via Bash.
   b. If exit 1 (not available): note "Codex CLI not available — skipping plan advisory" and proceed to step 7.
   c. If exit 0 (available): invoke Codex to adversarially review the plan via Bash:
      `codex exec --sandbox read-only "You are an adversarial plan reviewer. Your job is to break confidence in this plan, not to validate it. Default to skepticism — assume the plan can fail in subtle, high-cost ways until evidence says otherwise. Review for: (1) blind spots and missing risks — what failure modes are not addressed? (2) scope concerns — too broad, too narrow, or poorly bounded? (3) acceptance criteria gaps — can each criterion be verified deterministically? (4) design decision weaknesses — are there simpler or safer alternatives? (5) rollback and partial-failure scenarios — what happens if implementation stalls halfway? Report only material findings. Each finding must answer: What can go wrong? Why is this plan vulnerable? What is the likely impact? What concrete change would reduce the risk? Number each finding with severity [HIGH/MEDIUM/LOW]. Prefer one strong finding over several weak ones. If the plan looks solid, say so directly with no findings. Here is the plan file to review: docs/plans/active/<plan-file>"`
   d. Present Codex findings to the user as a numbered list.
   e. If Codex returned no actionable findings: note "Codex: 指摘なし" and proceed to step 7.
   f. If findings exist, use AskUserQuestion:
      - Question: "Codex からプランへの指摘があります。どう対応しますか？"
      - Options:
        1. プランを修正する — edit plan per relevant findings, then re-display
        2. 指摘を確認済み、このまま進む — proceed without changes
   g. After user decision, proceed to step 7.
7. **Flow confirmation**: Confirm the flow selected in step 2.7 and state which skill to invoke next:
   - 標準フロー → `/work`
   - Ralph Loop → `/loop`

## Output

- Updated or newly created plan file
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Branch/Worktree creation is deferred to the chosen flow skill
- Chosen execution flow (standard /work or Ralph Loop /loop)

## Anti-bottleneck

Before asking the user for confirmation or choices during planning, first check whether the answer is available from the codebase, existing plans, docs, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

## Additional resources

- [template.md](template.md)
