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
   - implementation outline: one item per slice, each ending with that slice's verification command, the narrowest check that covers the slice's files
     (see the comment in the template's `## Implementation outline`)
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
    a. Decide whether to draw, using [diagrams.md](diagrams.md) "When to draw, when to skip". If skipping, write `None (<reason>)` in the plan's `## Visual review` section (add the section as in 10.d when missing) and go to step 11 (the approval gate in step 12 still applies).
    b. Copy [visual-template.html](visual-template.html) to `.harness/state/plan-visual/<slug>.html` inside the task worktree and fill it in following [diagrams.md](diagrams.md): 1 to 3 easy figures first (the first is `id="overview"`, the figure `/pr` attaches to the PR), then the "Before you approve" box, then the closed fold with the detail overview (`id="detail-overview"`) and one detail figure per question about the mechanism. Each detail figure carries a "What to check:" caption, and every node in it is grounded in an existing path or a path the plan names.
    c. Check the page per [diagrams.md](diagrams.md) "Self-check before showing". Lint: run `./scripts/plan-visual.sh lint .harness/state/plan-visual/<slug>.html` and fix the page until it exits 0. If `./scripts/plan-visual.sh --help` prints no `scripts/plan-visual.sh lint <html>` line (a worktree made before `lint` existed), run the main checkout's helper on the page's absolute path as [diagrams.md](diagrams.md) shows; if that helper's `--help` has no such line either, skip the lint and say why. Self-check: shoot the full page, the overview (`--fragment overview`), and every other figure one by one (`--fragment <id>`) with `./scripts/plan-visual.sh shot`, read every PNG, fix what is wrong, and repeat until every PNG is clean. If `shot` exits 2 (no Chrome / Chromium found), or the agent cannot read images, skip the self-check (not the lint) and say so.
    d. Record in the plan's `## Visual review` section: the page path, the lint result (or why the lint was skipped), and the self-check result (or why the self-check was skipped). A plan made from an older `docs/plans/templates/feature-plan.md` (a seed file that `ralph upgrade` does not overwrite) may lack the section; add it after `## Affected areas`.
11. **Other-model plan advisory (`[standard].plan_advisory`)**: the other CLI's model reviews the plan adversarially (Claude Code asks codex, Codex asks claude). Define `<scratch>` once: the agent's scratchpad directory when the tool provides one, else `$(mktemp -d)`. Record the outcome on the `Plan reviewed` item of the plan's `## Progress checklist` (the digest skips that section) as `plan advisory: <outcome>`, where the outcome is `skipped (<reason>)`, `incomplete (<reason>)`, `no findings (<reviewer>)`, or `ran with <reviewer>` followed by the findings and what happened to them. When you record it, also tick that item's box (`- [x] Plan reviewed`), whatever the outcome, a skip included: the box says this step is done, not that a review ran.
    a. **Read the switch**: from the repository root (the task worktree root, so `ralph` reads its `./ralph.toml`), run `ralph config get standard.plan_advisory` and read its exit code, stdout, and stderr. After a skip, go to step 12.
       - Exit 0 and `true`: continue. Exit 0 and `false`: skip, recording `plan advisory: skipped (off: [standard].plan_advisory)`.
       - Exit 1 with `unknown command "config"` on stderr (a ralph that predates the verb): skip, recording `plan advisory: skipped (ralph without config get; treated as off)`.
       - Exit 127 (`ralph` is not on PATH): skip, recording `plan advisory: skipped (ralph not found; treated as off)`.
       - Anything else (exit 2, 3, any other 1, or exit 0 with neither `true` nor `false`): stop and report the error to the user. Do not treat it as off.
    b. **Driver and reviewer**: the driver is the CLI running this skill (Claude Code → `claude`, Codex → `codex`). `RALPH_PRIMARY_CLI` set to `claude` or `codex` (case-insensitive) overrides it; any other value is ignored. Do not infer the driver from `which`. The reviewer is the other one. Work out the driver and the reviewer in one Bash call, with `<claude|codex>` replaced by the CLI running this skill:
       ```
       . scripts/xreview-helpers.sh; DRIVER=<claude|codex>; override=$(printf '%s' "${RALPH_PRIMARY_CLI:-}" | tr '[:upper:]' '[:lower:]'); case "$override" in claude|codex) DRIVER=$override ;; esac; REVIEWER=$(pick_reviewer "$DRIVER"); echo "driver=$DRIVER reviewer=$REVIEWER"
       ```
       A Bash tool keeps no shell state between calls, so `$DRIVER` and `$REVIEWER` are gone once this call ends; the later steps use the two printed values, not the variables. Then check that the reviewer's CLI exists: `./scripts/codex-check.sh` exits 0 for `codex`, `command -v claude` succeeds for `claude`. If it does not, skip, record `plan advisory: skipped (<reviewer> not available)`, and go to step 12.
    c. **Launch**: first, from the repository root, save the git state the reviewer must not change: `{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > <scratch>/plan-advisory-<slug>-git-before.txt` (HEAD, the index, and the work tree with its untracked files; the last two commands also catch an edit to a file that was already changed or untracked, such as the plan itself). Then run the launch below in the background (Claude Code: Bash `run_in_background`; Codex driver: its own background run). Its first line sources `./scripts/ralph-config.sh` and reads the reviewer's model and effort with `ralph_setting`, in the same Bash call as the launch (a Bash tool keeps no shell state between calls). `ralph_setting <key>` runs `ralph config get <key>` from the repository root, so the values come from the `[reviewer]` table of `ralph.toml`, and a `ralph.local.toml` in the main worktree's root overrides them on this machine only; no environment variable changes them. It falls back to the default in its table silently only when `ralph` is not on PATH or predates `config get`. On any other failure (exit 2 for a key that `ralph` does not know, exit 3 for a `ralph.toml` or `ralph.local.toml` that cannot be read or fails validation) it writes one line on stderr naming the key, the exit code, and the default, and the launch goes on with the default; pass that line on to the user. A watchdog in the same background shell enforces the 20-minute bound directly — the `sleep 1200` below — instead of relying on a completion notification a hung reviewer would never send.

       **Reviewer `codex`**:
       ```
       . ./scripts/ralph-config.sh; model=$(ralph_setting reviewer.codex_model); effort=$(ralph_setting reviewer.codex_effort)
       rm -f <scratch>/plan-advisory-<slug>-last.md; command codex -m "$model" -c "model_reasoning_effort=$effort" exec --sandbox read-only --ignore-rules -o <scratch>/plan-advisory-<slug>-last.md "You are an adversarial plan reviewer. Your job is to break confidence in this plan, not to validate it. Default to skepticism — assume the plan can fail in subtle, high-cost ways until evidence says otherwise. Review for: (1) blind spots and missing risks — what failure modes are not addressed? (2) scope concerns — too broad, too narrow, or poorly bounded? (3) acceptance criteria gaps — can each criterion be verified deterministically? (4) design decision weaknesses — are there simpler or safer alternatives? (5) rollback and partial-failure scenarios — what happens if implementation stalls halfway? Report only material findings. Each finding must answer: What can go wrong? Why is this plan vulnerable? What is the likely impact? What concrete change would reduce the risk? Number each finding with severity [HIGH/MEDIUM/LOW]. Prefer one strong finding over several weak ones. If the plan looks solid, say so directly with no findings. Here is the plan file to review: docs/plans/active/<plan-file>" </dev/null > <scratch>/plan-advisory-<slug>.log 2>&1 & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"
       ```
       `</dev/null` closes stdin so codex does not wait for "additional input from stdin"; `command` bypasses a shell alias that adds its own `-m` (codex rejects a duplicated flag). `--sandbox read-only` blocks file writes and, by default, network access for the commands the advisory executes, and `--ignore-rules` stops an `allow` rule in a user or project execpolicy `.rules` file from running a matching command outside that sandbox. Neither covers MCP servers from the user's config or file reads outside the repository (see `/cross-review` Step 4). `rm -f` before launch means a leftover file from an earlier run cannot be mistaken for this run's output. The watchdog backgrounds `sleep 1200` and `codex` as direct siblings (not nested inside a further subshell) and polls with `kill -0` rather than `wait`, because `wait` only works on the calling shell's own direct children — a nested `( sleep 1200; kill "$cpid" ) &` form leaves an orphaned `sleep` behind when codex finishes early, since killing that wrapper subshell does not kill its own child.

       **Reviewer `claude`** (`<prompt>` is the codex prompt above, word for word, ending with the plan path):
       ```
       . ./scripts/ralph-config.sh; model=$(ralph_setting reviewer.claude_model); effort=$(ralph_setting reviewer.claude_effort)
       rm -f <scratch>/plan-advisory-<slug>-last.md; RALPH_WIP_CHECKPOINT=off command claude -p --model "$model" --effort "$effort" --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text "<prompt>" </dev/null > <scratch>/plan-advisory-<slug>-last.md 2> <scratch>/plan-advisory-<slug>.log & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "claude rc=$rc"
       ```
       `--restricted` removes the built-in tools that run commands and ignores the user, project, and local settings files, so the hooks they define do not load (ralph's SessionEnd hook commits a feature branch's uncommitted changes with `git add -A`); it also confines the file tools to the working directory. `--tools Read,Grep,Glob` leaves only tools that read, and `--permission-mode dontAsk` refuses anything that would ask for permission. `--tools` does not cover MCP tools, so `--strict-mcp-config` loads no MCP server from any config, and `--disallowedTools "mcp__*"` removes the tools of a server that starts anyway. `RALPH_WIP_CHECKPOINT=off` keeps a hook that runs anyway from making a `wip:` commit. `--effort` sets the reasoning effort; claude ignores a value it does not know, with a warning, and uses its own default, so `ralph config get` refuses (exit 3) a `[reviewer].claude_effort` other than low, medium, high, xhigh, or max. Keep the prompt right after `--output-format text`: `--tools` and `--disallowedTools` take a list and would read a prompt placed after them as one more tool name. On claude 2.1.304 this left the session with Glob, Grep, and Read and no MCP server, and no project hook ran. `</dev/null`, `command`, `rm -f`, and the watchdog are there for the same reasons as in the codex form.
    d. **Completion contract**: wait for the task's completion notification, then read `codex rc=` or `claude rc=` from its output. Write the git state again with the same command, to `<scratch>/plan-advisory-<slug>-git-after.txt`, and `cmp` it with `-git-before.txt`. If they differ, stop: show the user the `diff` of the two files and do not go on to step 12, since something changed the repository while the reviewer ran. Read the findings from `<scratch>/plan-advisory-<slug>-last.md`, not the log. Treat the advisory as complete only when `rc=0` and that file is non-empty. Otherwise (non-zero `rc`, or the file missing/empty): record `plan advisory: incomplete (<reason>)` and proceed to step 12 — never report it as "no findings". A `sleep 1200` timeout kill commonly still reports `rc=0` for codex (codex shuts down gracefully on the signal), so the file check — not `rc` — is what actually catches a timeout; do not treat `rc=0` alone as sufficient.
    e. Present the reviewer's findings to the user as a numbered list.
    f. If the reviewer returned no actionable findings: record `plan advisory: no findings (<reviewer>)` and proceed to step 12.
    g. If findings exist, use AskUserQuestion:
       - Question: "The plan advisory returned findings on the plan. How do you want to proceed?"
       - Options:
         1. Update plan — edit plan per relevant findings, then re-display; redo step 10 for every figure the edit affects before step 12
         2. Acknowledge findings, continue — proceed without changes
    h. After the user decision, record `plan advisory: ran with <reviewer>`, the findings, and what happened to them, then go to step 12.
12. **Approval gate**: the user approves the plan before `/implement`.
    a. If step 10 produced a page, run `./scripts/plan-visual.sh open .harness/state/plan-visual/<slug>.html` and give the user the absolute path it prints. It prints the path even when nothing could open the page (no opener, headless environment), so always pass it on. Some Linux openers (`xdg-open` with certain desktop backends) may not return until the browser exits, so on Linux run the command in the background (Claude Code: Bash `run_in_background`; Codex driver: its own background run).
    b. Summarize in chat: the scope in one paragraph, the slices, what remains unknown, whether the lint or the self-check was skipped, and what happened to the plan advisory's findings (none, applied, acknowledged, incomplete, or skipped and why).
    c. Ask with `AskUserQuestion` (Codex: numbered options):
       - Question: "Approve this plan?"
       - Options:
         1. Approve — record the approval and hand off to `/implement`
         2. Needs changes — describe what should change
    d. **Needs changes**: apply the feedback to the plan, redo step 10 for the affected figures, and return to 12.a. Do not re-run the plan advisory unless the user asks.
    e. **Approve**: after the last edit to the plan body, run `./scripts/plan-visual.sh digest <plan-path>`. Then set `- Status: Approved`, set `- Approved: <YYYY-MM-DD> sha256:<digest>` (a plan made from an older `docs/plans/templates/feature-plan.md` may lack this line; add it right after `- Status:`), and tick `Plan approved` in the progress checklist. The digest skips the `- Status:`, `- Approved:`, and `- Branch:` lines and the `## Progress checklist` section and reads a `- [x]` / `- [X]` box at the start of a list item as `- [ ]`, so writing them keeps it valid; re-run `digest` and confirm it still matches.
    f. State that `/implement` is the next skill to invoke. Ticking checkboxes (acceptance criteria, checklists) does not change the digest, so marking acceptance criteria done is fine. Any other later edit to the plan outside `- Status:`, `- Approved:`, `- Branch:`, and `## Progress checklist` changes the digest, and `/implement` then asks whether to redo the visual review and approval. Record implementation notes under `## Progress checklist`.

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
| Plan advisory (step 11) | Driver `claude`, so the reviewer is `codex`; the switch is read with `ralph config get standard.plan_advisory` | Driver `codex`, so the reviewer is `claude`; the switch is read the same way |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
