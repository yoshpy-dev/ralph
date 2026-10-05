---
name: plan
description: Create or refresh a scoped implementation plan before risky, ambiguous, long-running, or multi-file work. Accepts an optional GitHub issue number or URL for context pre-fill. Resolves high-leverage implementation forks with the user before finalizing. Ensures a clean-base task worktree before writing plan artifacts. Shows a visual review page and asks the user to approve the plan before /implement.
---
Create or update a plan in `docs/plans/active/` from an isolated task
worktree.

## Goals

- Turn a request into a versioned plan that survives context loss
- Define acceptance criteria and evidence before deep implementation
- Make later review and verification cheaper
- Let a human see and approve what will be built before implementation
  starts
- Keep plan artifacts out of the default checkout by ensuring a task worktree
  before the first file write

## Steps

1. Read `AGENTS.md`, `CLAUDE.md`, relevant `.claude/rules/`, and existing active plans.
2. Inspect only enough code and docs to understand the request and blast radius.
3. If a GitHub issue number or URL is provided:
   a. `gh issue view <number> --json title,body,labels,number`
   b. Pre-fill: Objective from title, Related request from body, Related issue: #N
   c. If no issue provided: set "Related issue: N/A"
4. Choose a branch type (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `ci`, `build`, `perf`, `release`, or `security`) and a stable slug from the request.
5. **Ensure the task worktree before creating the plan file**:
   - First run `./scripts/ralph-worktree.sh current`. If it returns a matching task state (for example from `/spec` handoff), adopt that worktree and do not create a second one.
   - Otherwise run `./scripts/ralph-worktree.sh ensure --id plan-<slug> --kind standard --branch <type>/<slug> --path .claude/worktrees/<slug> --canonical-ref "<issue URL, spec reference, or request summary>" --cleanup-policy pr-success`.
   - The helper must create the worktree from a clean default branch and store state under `$(git rev-parse --git-common-dir)/ralph/worktrees/`.
   - If a matching state record exists, resume it. If the branch/path exists with mismatched state, stop instead of overwriting.
   - All subsequent plan creation and edits happen inside the returned worktree path.
6. Create one active plan file inside the task worktree with `./scripts/new-feature-plan.sh --type <type> <slug> [issue-number]` or from [template.md](template.md). Set `Branch:` to the branch recorded in the worktree state rather than leaving it as `TBD`.
7. Fill in:
   - objective
   - scope and non-goals
   - assumptions
   - affected files or systems
   - visual review (filled in step 10)
   - acceptance criteria
   - implementation outline
   - verify plan (static analysis checks, spec compliance criteria, documentation drift checks, evidence to capture)
   - test plan (unit tests, integration tests, regression tests, edge cases, evidence to capture)
   - risk register
   - rollout or rollback notes
   - evidence targets
8. **Critical forks (convergent)**: After the initial draft is in place, scan the plan for "critical forks" — implementation decisions that meet **all three** of:
   - Two or more approaches differ materially in risk, cost, or rollback profile
   - The choice cannot be resolved from the codebase, existing `.claude/rules/`, docs, or a reasonable default
   - Reversing the decision mid-implementation would cost more than roughly one slice of rework

   For each critical fork identified:
   a. Use `AskUserQuestion` with one focused question and 2-4 concrete options. Each option must briefly state its pros/cons so the user can choose informedly.
   b. Record the chosen approach and its rationale in the plan's "Design decisions" section (see [template.md](template.md)).
   c. If the chosen option invalidates other plan sections (outline, risks, affected files), revise them before continuing.

   If no critical forks exist after scanning, write "Critical forks: None" in the Design decisions section and proceed.

   **Do NOT ask about**:
   - Stylistic or easily reversible choices (internal naming, helper placement inside an established module pattern)
   - Decisions already settled by `.claude/rules/`, `AGENTS.md`, or the upstream `/spec` output
   - Anything a reasonable default + explicit assumption would cover

   Purpose is **convergent** — narrow between enumerated options, not expand the design space. Divergent ideation belongs to `/spec`, not here.

9. Keep the plan high-level enough to avoid cascading low-level mistakes, and end it with a short readiness checklist.
10. **Visual review page**: build a page that shows the approver what will be built.
    a. Decide whether to draw, using [diagrams.md](diagrams.md) "When to draw, when to skip". If skipping, write `None (<reason>)` in the plan's `## Visual review` section and go to step 11 (the approval gate in step 12 still applies).
    b. Copy [visual-template.html](visual-template.html) to `.harness/state/plan-visual/<slug>.html` inside the task worktree and fill it in following [diagrams.md](diagrams.md): the overview figure first, one detail figure per question the approver needs answered, a "What to check:" caption on every figure, and every node grounded in an existing path or a path the plan names.
    c. Self-check per [diagrams.md](diagrams.md) "Self-check before showing": shoot the full page and the overview (`--fragment overview`) with `./scripts/plan-visual.sh shot`, read both PNGs, fix what is wrong, and repeat until both are clean. If `shot` exits 2 (no Chrome / Chromium found), or the agent cannot read images, skip the self-check and say so.
    d. Record in the plan's `## Visual review` section: the page path and the self-check result (or why the self-check was skipped).
11. **Codex plan advisory (optional)**:
    a. Run `./scripts/codex-check.sh` via Bash.
    b. If exit 1 (not available): note "Codex not available — skipping plan advisory" and proceed to step 12.
    c. If exit 0 (available): invoke Codex to adversarially review the plan via Bash. Define `<scratch>` once: the agent's scratchpad directory when the tool provides one, else `$(mktemp -d)`. Source `./scripts/ralph-config.sh` in the same Bash call as the launch below (a Bash tool keeps no shell state between calls), then run it in the background (Claude Code: Bash `run_in_background`; Codex driver: its own background run). A watchdog in the same background shell enforces the 20-minute bound directly — the `sleep 1200` below — instead of relying on a completion notification a hung codex would never send:
       ```
       . ./scripts/ralph-config.sh
       rm -f <scratch>/plan-advisory-<slug>-last.md; command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-xhigh}" exec --sandbox read-only --ignore-rules -o <scratch>/plan-advisory-<slug>-last.md "You are an adversarial plan reviewer. Your job is to break confidence in this plan, not to validate it. Default to skepticism — assume the plan can fail in subtle, high-cost ways until evidence says otherwise. Review for: (1) blind spots and missing risks — what failure modes are not addressed? (2) scope concerns — too broad, too narrow, or poorly bounded? (3) acceptance criteria gaps — can each criterion be verified deterministically? (4) design decision weaknesses — are there simpler or safer alternatives? (5) rollback and partial-failure scenarios — what happens if implementation stalls halfway? Report only material findings. Each finding must answer: What can go wrong? Why is this plan vulnerable? What is the likely impact? What concrete change would reduce the risk? Number each finding with severity [HIGH/MEDIUM/LOW]. Prefer one strong finding over several weak ones. If the plan looks solid, say so directly with no findings. Here is the plan file to review: docs/plans/active/<plan-file>" </dev/null > <scratch>/plan-advisory-<slug>.log 2>&1 & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"
       ```
       `</dev/null` closes stdin so codex does not wait for "additional input from stdin"; `command` bypasses a shell alias that adds its own `-m` (codex rejects a duplicated flag). `--sandbox read-only` blocks file writes and, by default, network access for the commands the advisory executes, and `--ignore-rules` stops an `allow` rule in a user or project execpolicy `.rules` file from running a matching command outside that sandbox. Neither covers MCP servers from the user's config or file reads outside the repository (see `/cross-review` Step 4). `rm -f` before launch means a leftover file from an earlier run cannot be mistaken for this run's output. The watchdog backgrounds `sleep 1200` and `codex` as direct siblings (not nested inside a further subshell) and polls with `kill -0` rather than `wait`, because `wait` only works on the calling shell's own direct children — a nested `( sleep 1200; kill "$cpid" ) &` form leaves an orphaned `sleep` behind when codex finishes early, since killing that wrapper subshell does not kill its own child.
    d. **Completion contract**: wait for the task's completion notification, then read `codex rc=` from its output; read the findings from the `-o` file, not the log. Treat the advisory as complete only when `rc=0` and the `-o` file is non-empty. Otherwise (non-zero `rc`, or the `-o` file missing/empty): note "Codex advisory: incomplete (<reason>)" and proceed to step 12 — never report it as "Codex: no findings". A `sleep 1200` timeout kill commonly still reports `rc=0` (codex shuts down gracefully on the signal), so the `-o` file check — not `rc` — is what actually catches a timeout; do not treat `rc=0` alone as sufficient.
    e. Present Codex findings to the user as a numbered list.
    f. If Codex returned no actionable findings: note "Codex: no findings" and proceed to step 12.
    g. If findings exist, use AskUserQuestion:
       - Question: "Codex returned findings on the plan. How do you want to proceed?"
       - Options:
         1. Update plan — edit plan per relevant findings, then re-display; redo step 10 for every figure the edit affects before step 12
         2. Acknowledge findings, continue — proceed without changes
    h. After the user decision, go to step 12.
12. **Approval gate**: the user approves the plan before `/implement`.
    a. If step 10 produced a page, run `./scripts/plan-visual.sh open .harness/state/plan-visual/<slug>.html` and give the user the absolute path it prints. It prints the path even when nothing could open the page (no opener, headless environment), so always pass it on. Some Linux openers (`xdg-open` with certain desktop backends) may not return until the browser exits, so on Linux run the command in the background (Claude Code: Bash `run_in_background`; Codex driver: its own background run).
    b. Summarize in chat: the scope in one paragraph, the slices, what remains unknown, whether the self-check was skipped, and what happened to the Codex findings (none, applied, acknowledged, skipped, or incomplete).
    c. Ask with `AskUserQuestion` (Codex: numbered options):
       - Question: "Approve this plan?"
       - Options:
         1. Approve — record the approval and hand off to `/implement`
         2. Needs changes — describe what should change
    d. **Needs changes**: apply the feedback to the plan, redo step 10 for the affected figures, and return to 12.a. Do not re-run the Codex advisory unless the user asks.
    e. **Approve**: after the last edit to the plan body, run `./scripts/plan-visual.sh digest <plan-path>`. Then set `- Status: Approved`, set `- Approved: <YYYY-MM-DD> sha256:<digest>`, and tick `Plan approved` in the progress checklist. The digest skips the `- Status:`, `- Approved:`, and `- Branch:` lines and the `## Progress checklist` section, so writing them keeps it valid; re-run `digest` and confirm it still matches.
    f. State that `/implement` is the next skill to invoke. Any later edit to the plan outside `- Status:`, `- Approved:`, `- Branch:`, and `## Progress checklist` changes the digest, and `/implement` then asks whether to redo the visual review and approval. Record implementation notes under `## Progress checklist`.

## Output

- Updated or newly created plan file
- `Status: Approved` with its digest on the `- Approved:` line
- Visual review page path (`.harness/state/plan-visual/<slug>.html`), or `None (<reason>)`
- One paragraph summary of what is in scope
- Explicit statement of what remains unknown
- Task worktree path and branch recorded by `ralph-worktree.sh`
- Next step: invoke `/implement`

## Anti-bottleneck

Before asking the user for confirmation or choices during planning, first check whether the answer is available from the codebase, existing plans, docs, or reasonable defaults. See the `anti-bottleneck` skill for the full checklist.

The step 12 approval is a required human decision: do not skip it on anti-bottleneck grounds.

## Additional resources

- [template.md](template.md)
- [diagrams.md](diagrams.md)
- [visual-template.html](visual-template.html)

## CLI execution modes

This skill runs under both Claude Code and Codex. The execution mode follows
the conventions in AGENTS.md and `.codex/AGENTS.override.md`.

| Aspect | Claude Code | Codex |
|--------|-------------|-------|
| Skill invocation | `/skill-name` slash command | `$skill-name` mention or the `/skills` menu (avoid the `/skill-name` form — it collides with built-ins) |
| Skill body path | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Subagent mechanism | `Task(subagent_type=...)` when a policy delegates | `.codex/agents/` custom agents when a policy delegates |
| Structured prompts | `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit reply |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
