---
name: cross-review
description: >
  Cross-model second opinion on the current diff, run when the
  [standard].cross_review switch is on and the reviewer CLI is available.
  Calls the CLI other than the one currently driving (Claude → Codex; Codex →
  Claude). Runs inline in the main context (not delegated as subagent) after
  /sync-docs completes. When the switch is off, the reviewer CLI is
  unavailable, or the diff is empty, records the skip in the triage report
  and proceeds to /pr.
  Findings are triaged by the driving CLI using implementation context before
  presentation to the user.
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion, Write
---
Provide a cross-model second opinion on the current diff before PR creation.
The review runs when `[standard].cross_review` is on (step 0), the reviewer
CLI is available (step 3), and the diff against the base branch is not empty
(step 4). Otherwise the skill writes a triage report that records the skip
and why, and hands off to `/pr`.

## Goals

- Catch blind spots that single-model review may miss
- Leverage a different model's perspective for cross-validation
- Triage findings using implementation context to reduce noise
- Present findings as advisory — never auto-apply

## Steps

0. **Read the switch (`[standard].cross_review`)**: from the repository root (the task worktree root, so `ralph` reads its `./ralph.toml`), run `ralph config get standard.cross_review` and read its exit code, stdout, and stderr.
   - Exit 0 and `true`: continue. Exit 0 and `false`: skip, with the reason `off: [standard].cross_review`.
   - Exit 1 with `unknown command "config"` on stderr (a ralph that predates the verb): skip, with the reason `ralph without config get; treated as off`.
   - Exit 127 (`ralph` is not on PATH): skip, with the reason `ralph not found; treated as off`.
   - Anything else (exit 2, 3, any other 1, or exit 0 with neither `true` nor `false`): stop and report the error to the user. Do not treat it as off.

   A skip does not call the reviewer: run steps 1 and 2 (the report header needs the cycle, the driver, and the reviewer), leave out steps 3 to 5, and write the skipped report in step 6.

1. **Resolve active plan identity and read cycle counter** (standard flow cap enforcement):
   a. Read `.harness/state/standard-pipeline/active-plan.json` to get the pinned plan path.
      - **If present**: proceed to step 1.b (persisted-identity mode).
      - **If missing**: warn the user and continue in **fallback mode** — no persisted identity. In fallback mode: skip step 1.b entirely (do NOT read or create `cycle-count.json`, to avoid reusing stale counters from other plans or leaking orphan state) and set `cycle=1`, `cap=∞` for step 7 (cap cannot be enforced).
   b. (Persisted-identity mode only) Confirm `active-plan.json` includes the task worktree metadata written by `/implement`: `plan_path`, `worktree_path`, `branch`, and `worktree_state_id`. If `plan_path` no longer exists, try `./scripts/ralph-worktree.sh resume --id <worktree_state_id>` and recover the plan from the recorded canonical reference before falling back. Do not rescan `docs/plans/active/` while valid pinned state exists.
   c. (Persisted-identity mode only) Read `.harness/state/standard-pipeline/cycle-count.json`. If its `plan_path` matches `active-plan.json`, use its `cycle`. If missing, initialize `{"plan_path": "<path>", "cycle": 1}` (first /cross-review run of this plan). If its `plan_path` does **not** match, warn and treat as fallback mode for this run (do not overwrite — `/implement` is responsible for resolving mismatched state).
   d. Read `RALPH_STANDARD_MAX_PIPELINE_CYCLES` by sourcing `./scripts/ralph-config.sh` in a subshell (default `2`).
   e. Record the current cycle number and the cap for use in Step 7.

   **Hard prohibition**: Do NOT rediscover the plan by rescanning `docs/plans/active/` once `active-plan.json` exists. Always consume the persisted path and worktree state. This prevents cross-plan counter leakage when multiple plans coexist.

2. **Resolve driver and reviewer CLIs**:
   The skill must call the CLI other than the one currently driving the work.

   a. **Determine driver**: the driver is the CLI running this skill (Claude Code → `claude`, Codex → `codex`). `RALPH_PRIMARY_CLI` set to `claude` or `codex` (case-insensitive) overrides it; any other value is ignored. Do not infer the driver from `which`.
   b. **Determine reviewer**: source `scripts/xreview-helpers.sh` and call `pick_reviewer` with the driver from step 2.a to get the opposite CLI. Work out the driver and the reviewer in one Bash call, with `<claude|codex>` replaced by the CLI running this skill:
      ```
      . scripts/xreview-helpers.sh; DRIVER=<claude|codex>; override=$(printf '%s' "${RALPH_PRIMARY_CLI:-}" | tr '[:upper:]' '[:lower:]'); case "$override" in claude|codex) DRIVER=$override ;; esac; REVIEWER=$(pick_reviewer "$DRIVER"); echo "driver=$DRIVER reviewer=$REVIEWER"
      ```
      A Bash tool keeps no shell state between calls, so `$DRIVER` and `$REVIEWER` are gone once this call ends; the later steps use the two printed values, not the variables. `pick_reviewer` keeps the "reviewer is always the driver's opposite" mapping in one grep-able, unit-tested place (`tests/test-xreview-helpers.sh`) instead of duplicating it inline.

   Record the resolved driver and reviewer for the rest of the skill. Both must be reported in the triage report header.

3. **Check reviewer availability**: check that the reviewer's CLI exists: `./scripts/codex-check.sh` exits 0 for `codex`, `command -v claude` succeeds for `claude`. If it does not, skip, with the reason `<reviewer> not available` (for example `codex not available`): leave out steps 4 and 5 and write the skipped report in step 6.

4. **Invoke reviewer**:
   - Determine base branch via Bash: `. scripts/xreview-helpers.sh; BASE=$(detect_base_branch)` — resolution order: (1) `$RALPH_XREVIEW_BASE` if set and non-empty (explicit override); (2) `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` with leading `origin/` stripped (repo default branch); (3) `main` if `refs/heads/main` exists, else `master`.
   - Check the diff is non-empty, in the same Bash call as the base branch above (`$BASE` does not outlive a call, and with an empty `$BASE` the check always sees no diff): `. scripts/xreview-helpers.sh; BASE=$(detect_base_branch); git diff "$BASE"...HEAD --quiet; echo "base=$BASE rc=$?"`. Record the printed base: later calls (the reviewer prompt below) use that value, not a shell variable. `rc=0` means no diff: skip, with the reason `empty diff`, leave out the rest of this step and step 5, and write the skipped report in step 6.
   - **Save the git state** the reviewer must not change. Define `<scratch>` once: the agent's scratchpad directory when the tool provides one, else `$(mktemp -d)`. From the repository root, run the command of `/plan` step 11.c with this run's file name: `{ git rev-parse HEAD; git status --porcelain=v1 --untracked-files=all; git diff --cached; git diff; git ls-files -z --others --exclude-standard | xargs -0 git hash-object --; } > <scratch>/cross-review-<slug>-c<cycle>-git-before.txt` (HEAD, the index, and the work tree with its untracked files).
   - **reviewer = `codex`**: The native reviewer analyzes the full diff and returns structured findings with severity, affected files, and recommendations. Source `./scripts/ralph-config.sh` and re-derive `$BASE` in the same Bash call as the launch below (a Bash tool keeps no shell state between calls), then run it in the background (Claude Code: Bash `run_in_background`). A watchdog in the same background shell enforces the 20-minute bound directly — the same `sleep 1200` shape as `/plan` step 11.c — instead of relying on a completion notification a hung codex would never send:
     ```
     . ./scripts/ralph-config.sh; . scripts/xreview-helpers.sh; BASE=$(detect_base_branch)
     rm -f <scratch>/cross-review-<slug>-c<cycle>-last.md; command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-xhigh}" -c sandbox_mode=read-only exec review --ignore-rules --base "$BASE" -o <scratch>/cross-review-<slug>-c<cycle>-last.md </dev/null > <scratch>/cross-review-<slug>-c<cycle>.log 2>&1 & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid" 2>/dev/null; do sleep 1; done; kill "$cpid" 2>/dev/null ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid" 2>/dev/null; kill "$wpid" 2>/dev/null; echo "codex rc=$rc"
     ```
     (`</dev/null` / `command` reasons: see `/plan` step 11.c.) `-c sandbox_mode=read-only` runs the commands the reviewer executes in a sandbox that blocks file writes and, by default, network access: the project's `.codex/config.toml` sets `sandbox_mode = "danger-full-access"` and `codex exec` never asks for approval, so without the override an instruction hidden in the diff under review could run any command (`exec review` has no `--sandbox` flag). Keep every `-c` before `exec`: on codex-cli 0.154.0 a `-c` placed after `exec` makes codex discard all the `-c` options given before it, this one and the reasoning effort included. `--ignore-rules` stops an `allow` rule in a user or project execpolicy `.rules` file from running a matching command outside that sandbox. Neither covers MCP servers from the user's config, which still start and whose tools run outside the sandbox, and the reviewer can still read files outside the repository. `rm -f` before launch means a leftover file from an earlier cycle cannot be mistaken for this run's output. (Watchdog shape rationale: see `/plan` step 11.c — `wait` only works on the calling shell's own direct children, so `sleep`, `codex`, and the poll loop are all backgrounded as direct siblings instead of nesting `sleep`+`kill` inside one subshell.)

     **Completion contract**: wait for the task's completion notification, then read `codex rc=` from its output; read the findings from the `-o` file, not the log. Treat the reviewer as complete only when `rc=0` and the `-o` file is non-empty. Otherwise (non-zero `rc`, or the `-o` file missing/empty): write `Reviewer status: incomplete (<reason>)` in the triage report header (instead of `Reviewer status: complete`), skip Step 5 triage, and in Step 8 do not use Cases A–C — instead ask (`AskUserQuestion` under Claude, numbered options under Codex) with (1) re-run the reviewer, (2) proceed to `/pr` recording "cross-review incomplete" as a known gap, (3) abort. A `sleep 1200` timeout kill commonly still reports `rc=0` (codex shuts down gracefully on the signal), so the `-o` file check — not `rc` — is what actually catches a timeout; do not treat `rc=0` alone as sufficient.
   - **reviewer = `claude`**: `RALPH_WIP_CHECKPOINT=off claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --permission-mode auto --output-format json` with a prompt that instructs Claude to act as an adversarial diff reviewer (see prompt template at the end of this file). (the variable is read from the environment — set it directly or source `scripts/ralph-config.sh`, which exports it; unset falls back to `opus`). The reviewer may run `git diff` to read the diff, so this call keeps the tools that `/plan` step 11.c takes away, and the project's hooks load with it: `RALPH_WIP_CHECKPOINT=off` keeps ralph's SessionEnd hook from committing the branch's uncommitted changes as a `wip:` commit, and the git-state check below catches any other change. Complete only when `claude -p` exits 0 and returns a non-empty result; otherwise the same incomplete path above applies (status line, no triage, the Step 8 dialog).
   - **Check the git state**: once the reviewer's run has ended, complete or not, write the git state again with the same command to `<scratch>/cross-review-<slug>-c<cycle>-git-after.txt` and `cmp` it with `-git-before.txt`. If they differ, stop: show the user the `diff` of the two files and do not go on to step 5, since something changed the repository while the reviewer ran.

   Both paths must produce findings with: severity (HIGH/MEDIUM/LOW), affected file/line refs, what-can-go-wrong, recommended fix.

5. **Triage findings** (noise reduction):
   Triage each finding using implementation context. This step runs inline (main context) because triage value depends on knowing *why* the code was written that way.

   **Load triage context:**
   - Read the active plan using the path recorded in `active-plan.json` from Step 1 — do not rescan `docs/plans/active/`. If `active-plan.json` is absent (fallback mode), use the path resolved in Step 1's fallback.
   - Read the self-review report from `docs/reports/` (if available)
   - Read the verify report from `docs/reports/` (if available)
   - Consider implementation decisions made during the current session

   **If the reviewer returned non-structured output** (no clear severity/file/recommendation per finding): skip triage, fall back to Step 7 legacy behavior (present all findings as-is).

   **2-axis evaluation** (Semgrep pattern):
   For each finding, evaluate on two independent axes:
   - **Axis 1 — Real issue?**: Is this a genuine problem affecting correctness, security, reliability, or data integrity? Or is it a style preference, hypothetical concern, or false positive?
   - **Axis 2 — Worth fixing?**: Given the plan's scope, non-goals, existing mitigations (from self-review), and cost-benefit, should this be addressed now?

   **Classification rules:**
   | Axis 1: Real issue | Axis 2: Worth fixing | Classification |
   |---------------------|----------------------|----------------|
   | Yes | Yes | `ACTION_REQUIRED` |
   | Yes | Debatable | `WORTH_CONSIDERING` |
   | Debatable | Yes | `WORTH_CONSIDERING` |
   | Debatable | Debatable | `WORTH_CONSIDERING` |
   | No | — | `DISMISSED` |
   | — | No (out of scope, already addressed) | `DISMISSED` |

   **Conservative principle**: When uncertain, classify upward: DISMISSED → WORTH_CONSIDERING → ACTION_REQUIRED. Never silently drop findings.

   **DISMISSED categories** (each dismissed finding must have one):
   - `false-positive` — the finding is factually incorrect given the actual code
   - `already-addressed` — the issue was already fixed (cross-ref self-review or verify report)
   - `style-preference` — subjective style choice, not a defect
   - `out-of-scope` — valid concern but outside the plan's scope/non-goals
   - `context-aware-safe` — appears risky in isolation but is safe given the implementation context

6. **Write triage report**:
   Write the triage report to `docs/reports/cross-review-triage-<plan-slug>.md` using the template at `docs/reports/templates/cross-review-triage-report.md`. Include:
   - Header line `Driver: <claude|codex>  Reviewer: <claude|codex>` so the report is self-describing.
   - Header line `Reviewer status: complete`, or `incomplete (<reason>)` per Step 4's completion contract (for either reviewer, when the call did not finish cleanly), or `skipped (<reason>)` when step 0, step 3, or step 4 (an empty diff) skipped the review, with the reason that step gives (for example `Reviewer status: skipped (off: [standard].cross_review)` or `Reviewer status: skipped (empty diff)`).
   - For an incomplete or skipped run: still write the canonical `- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0` line (so `count_triage_findings` and the insights backfill can still parse the file), and state in the Triage context section that the review did not run and why. A skipped run also writes `Total reviewer findings: 0` and the cycle as usual.
   - All findings in their classified sections (ACTION_REQUIRED, WORTH_CONSIDERING, DISMISSED)
   - Triage rationale (1-2 sentences per finding to limit token cost)
   - Dismissal reasons with category for all DISMISSED findings
   - Summary counts in the header, as a single canonical line: `- After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N`
   - Current cycle and cap (from Step 1) in the header, e.g. `Cycle: 2/2 (cap reached)`

   **Verify the counts**: after writing the report, source `scripts/xreview-helpers.sh` and re-derive each category's count from the file just written — `. scripts/xreview-helpers.sh; count_triage_findings docs/reports/cross-review-triage-<slug>.md ACTION_REQUIRED` (repeat for `WORTH_CONSIDERING` and `DISMISSED`) — and confirm the three numbers match the summary line above before Step 7/8 branch on them. This reuses the same parser `tests/test-xreview-helpers.sh` pins, so a mismatch means the header line or table formatting drifted from `docs/reports/templates/cross-review-triage-report.md`.

   **Skipped run** (step 0, step 3, or step 4's empty diff): after the count check, append the insight event with `--verdict n/a` (see "Insight event" below) and invoke `/pr`. Steps 7 and 8 do not apply, and `cycle-count.json` is not incremented.

7. **Present triaged findings**:
   Display findings grouped by classification:
   - **ACTION_REQUIRED**: Show full details (finding + triage rationale + affected files). Header: "Action required (ACTION_REQUIRED)"
   - **WORTH_CONSIDERING**: Show full details. Header: "Worth considering (WORTH_CONSIDERING)"
   - **DISMISSED**: Show count and note that details are in the triage report. Example: "Dismissed: N items (see docs/reports/cross-review-triage-<slug>.md for details)"

8. **User decision**:
   Branch based on triage results **and** on whether the pipeline cycle cap has been reached (see Step 1 — cycle vs `RALPH_STANDARD_MAX_PIPELINE_CYCLES`).

   Let `CAP_REACHED = (cycle >= RALPH_STANDARD_MAX_PIPELINE_CYCLES)`. At the default cap of 2, `CAP_REACHED` is true during the second (and final) `/cross-review` run.

   **Records-only option**: offer "Fix records only" in Case A and Case B only when every finding to be fixed (the ACTION_REQUIRED findings in Case A, the WORTH_CONSIDERING findings in Case B) is confined to this task's pipeline records as defined in `.claude/rules/ralph/post-implementation-pipeline.md` ("Exception: fixes confined to this task's pipeline records"). Otherwise leave it out of the list. It is offered whether or not `CAP_REACHED`, because it does not start a new run.

   **Case A — ACTION_REQUIRED findings exist**:
   - If NOT `CAP_REACHED`: Use AskUserQuestion (Claude) or numbered stdin prompt (Codex):
     - Question: "Cross-review reported ACTION_REQUIRED findings. How do you want to proceed?"
     - Options:
       1. Fix — fix ACTION_REQUIRED issues, then re-run the full post-implementation pipeline: /self-review → /verify → /test → /sync-docs → /cross-review
       2. Fix records only — fix the records, check the exception's conditions, record the check in the triage report, then proceed to /pr (see the records-only note above)
       3. Also review WORTH_CONSIDERING — review both ACTION_REQUIRED and WORTH_CONSIDERING, then decide
       4. Acknowledge and create PR — proceed to /pr
   - If `CAP_REACHED` (cap-reached flow):
     - Question: "Pipeline re-run cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`) reached, but ACTION_REQUIRED findings remain. What do you want to do?"
     - Options:
       1. Raise the cap temporarily and re-run — have the user set a higher `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (e.g. export it) and re-run the pipeline
       2. Fix records only — same as in the non-cap list; the cap does not need raising
       3. Record findings and create PR — add unresolved ACTION_REQUIRED findings to the PR body's Known gaps section, then proceed to /pr
       4. Abort — stop without creating a PR; the user will resume manually

   **Case B — No ACTION_REQUIRED, but WORTH_CONSIDERING exist**:
   - If NOT `CAP_REACHED`:
     - Question: "Cross-review reported WORTH_CONSIDERING findings (no ACTION_REQUIRED). How do you want to proceed?"
     - Options:
       1. Review and fix — review WORTH_CONSIDERING findings, fix as needed, then re-run the full post-implementation pipeline
       2. Fix records only — fix the records, check the exception's conditions, record the check in the triage report, then proceed to /pr (see the records-only note above)
       3. Create PR — proceed to /pr
   - If `CAP_REACHED`:
     - Question: "Pipeline re-run cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`) reached, but WORTH_CONSIDERING findings remain. What do you want to do?"
     - Options:
       1. Raise the cap temporarily and re-run
       2. Fix records only — same as in the non-cap list; the cap does not need raising
       3. Create PR — add unresolved WORTH_CONSIDERING findings to the PR body's Known gaps section, then proceed to /pr
       4. Abort

   **Case C — All findings DISMISSED (or no findings)**:
   Note "Cross-review: all findings triaged (no ACTION_REQUIRED) — triage report: docs/reports/cross-review-triage-<slug>.md" and proceed to /pr.

9. **Proceed**:
   - **Incomplete review** (Step 4's incomplete path, chosen via Step 8's alternate dialog): "re-run the reviewer" returns to Step 4 and leaves `cycle-count.json` unchanged — no review happened, so no cycle was consumed. "Proceed to `/pr`" and "abort" follow the same branches as below.
   - **Skipped review** (step 0, step 3, or step 4's empty diff): invoke /pr as Step 6 says, with no Step 8 dialog; `cycle-count.json` is not incremented, since no review happened.
   - **Non-cap re-run** (Case A / Case B, `CAP_REACHED = false`): If `active-plan.json` exists, increment `cycle-count.json` (`cycle += 1`), then guide the user back to `/self-review`. The incremented cycle represents "the pass the user is about to enter".
   - **Cap-reached Option 1** ("Raise the cap temporarily and re-run"): Do **NOT** increment `cycle-count.json`. Instruct the user to `export RALPH_STANDARD_MAX_PIPELINE_CYCLES=<current cycle + 1>` (or higher) before re-running, so the unchanged `cycle` falls below the new cap. Then guide them back to `/self-review`.
   - **Fix records only** (Case A / Case B, with or without `CAP_REACHED`): make the fix commit, check it, and record the result exactly as `.claude/rules/ralph/post-implementation-pipeline.md` ("Exception: fixes confined to this task's pipeline records") says: the fix commit leaves the triage report alone, and the result goes into the triage report in a separate append-only commit. If every check there passes, do **NOT** increment `cycle-count.json`, and invoke /pr, stating in the PR body that the exception was used. If any fails, the fix needs the full pipeline: take the non-cap re-run path above, or, when `CAP_REACHED`, the cap-reached Option 1 path.
   - If the user chooses `/pr`: invoke /pr (which is responsible for deleting `active-plan.json` and `cycle-count.json` on success).
   - If the user chooses Abort: stop without invoking /pr; leave state files in place so the next `/implement` can resume.

## CLI execution modes

| Aspect | Claude Code (driver = claude) | Codex (driver = codex) |
|--------|-------------------------------|------------------------|
| Reviewer invocation | `command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-xhigh}" -c sandbox_mode=read-only exec review --ignore-rules --base "$BASE" -o <scratch>/cross-review-<slug>-c<cycle>-last.md </dev/null` (background with a 20-minute watchdog and the git-state check; see Step 4) | `RALPH_WIP_CHECKPOINT=off claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --permission-mode auto --output-format json` (adversarial reviewer prompt; complete only on exit 0 with a non-empty result; the git-state check of Step 4) |
| Step 8 user dialog | Structured choices via `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit 1–4 |
| Triage execution | inline (main context) | inline — chained within a single agent |
| Output file | `docs/reports/cross-review-triage-<slug>.md` | Same |

Driver / reviewer detection follows Step 2, the same rule as `/plan` step 11.b: the driver is the CLI running the skill, `RALPH_PRIMARY_CLI` overrides it, and `which` is not used to infer it; `pick_reviewer` from `scripts/xreview-helpers.sh` then computes the reviewer from the driver. `.codex/AGENTS.override.md` has Codex set `RALPH_PRIMARY_CLI=codex`. Step 6's count verification sources the same file's `count_triage_findings` (fake-CLI-free regression coverage for both helpers lives in `tests/test-xreview-helpers.sh`).

## Insight event (best-effort)

After writing the triage report (Step 6), append one insight event (errors are non-fatal):
```
./scripts/insights-append.sh --slug <slug> --flow standard --phase cross_review \
  --verdict <pass|action_required|n/a> --action-required <N> --worth-considering <N> \
  --dismissed <N> --cycle auto --source skill || true
```
Use `--verdict action_required` when ACTION_REQUIRED findings exist; `--verdict n/a` when the reviewer was incomplete (Step 4) or skipped (step 0, step 3, or step 4's empty diff) — the review did not run, so it is neither a pass nor a fail; `pass` otherwise. `--cycle auto` reads the pipeline cycle from `.harness/state/standard-pipeline/` and falls back to 1, the same rule as Step 1.

## What /cross-review does NOT do

- **Auto-fix**: Findings are advisory only. No code changes.
- **Block the flow on a skip**: If the switch is off, the reviewer CLI is unavailable, or the diff is empty, the flow continues to /pr, and the triage report records the skip (`Reviewer status: skipped (<reason>)`). Only an error from `ralph config get` (step 0) or a repository change during the review (step 4) stops it.
- **Replace /self-review**: Self-review (/self-review) and cross-review are complementary.
- **Suppress findings**: All findings (including DISMISSED) are recorded in the triage report for transparency.

## Anti-patterns to avoid

- Do NOT auto-apply all reviewer suggestions (causes churn)
- Do NOT loop more than once without user confirmation
- Do NOT use Review Gate / Stop Hook automation
- Do NOT dismiss findings without a documented reason and category
- Do NOT classify uncertain findings as DISMISSED — use WORTH_CONSIDERING instead
- Do NOT call the same CLI as the one driving — that defeats the cross-model purpose

## Claude reviewer prompt template

When `reviewer = claude`, run `claude -p` with a prompt similar to the one
below, with `<base>` replaced by the base step 4 printed (`base=<base>`):

```
You are an adversarial diff reviewer. Default to skepticism — assume the diff
can fail in subtle, high-cost ways until evidence says otherwise. Review the
diff between <base> and HEAD for: (1) correctness issues, (2) security
concerns, (3) error handling gaps, (4) logic errors, (5) missing test coverage,
(6) blind spots specific to the change. Report each finding with severity
[HIGH/MEDIUM/LOW], affected file:line, what can go wrong, why it is vulnerable,
likely impact, and concrete change to reduce the risk. Prefer one strong
finding over several weak ones. If the diff looks solid, say so directly.
```

Pipe the diff via stdin or fetch it inside the prompt with `git diff <base>...HEAD`.
