---
name: cross-review
description: >
  Optional cross-model second opinion on the current diff. Calls the CLI other
  than the one currently driving (Claude → Codex; Codex → Claude). Runs inline
  in the main context (not delegated as subagent) after /sync-docs completes.
  If the reviewer CLI is unavailable, silently skips and proceeds to /pr.
  Findings are triaged by the driving CLI using implementation context before
  presentation to the user.
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion, Write
---
Provide a cross-model second opinion on the current diff before PR creation.

## Goals

- Catch blind spots that single-model review may miss
- Leverage a different model's perspective for cross-validation
- Triage findings using implementation context to reduce noise
- Present findings as advisory — never auto-apply

## Steps

1. **Resolve active plan identity and read cycle counter** (standard flow cap enforcement):
   a. Read `.harness/state/standard-pipeline/active-plan.json` to get the pinned plan path.
      - **If present**: proceed to step 1.b (persisted-identity mode).
      - **If missing**: warn the user and continue in **fallback mode** — no persisted identity. In fallback mode: skip step 1.b entirely (do NOT read or create `cycle-count.json`, to avoid reusing stale counters from other plans or leaking orphan state) and set `cycle=1`, `cap=∞` for step 7 (cap cannot be enforced).
   b. (Persisted-identity mode only) Confirm `active-plan.json` includes the task worktree metadata written by `/work`: `plan_path`, `worktree_path`, `branch`, and `worktree_state_id`. If `plan_path` no longer exists, try `./scripts/ralph-worktree.sh resume --id <worktree_state_id>` and recover the plan from the recorded canonical reference before falling back. Do not rescan `docs/plans/active/` while valid pinned state exists.
   c. (Persisted-identity mode only) Read `.harness/state/standard-pipeline/cycle-count.json`. If its `plan_path` matches `active-plan.json`, use its `cycle`. If missing, initialize `{"plan_path": "<path>", "cycle": 1}` (first /cross-review run of this plan). If its `plan_path` does **not** match, warn and treat as fallback mode for this run (do not overwrite — `/work` is responsible for resolving mismatched state).
   d. Read `RALPH_STANDARD_MAX_PIPELINE_CYCLES` by sourcing `./scripts/ralph-config.sh` in a subshell (default `2`).
   e. Record the current cycle number and the cap for use in Step 7.

   **Hard prohibition**: Do NOT rediscover the plan by rescanning `docs/plans/active/` once `active-plan.json` exists. Always consume the persisted path and worktree state. This prevents cross-plan counter leakage when multiple plans coexist.

2. **Resolve driver and reviewer CLIs**:
   The skill must call the CLI other than the one currently driving the work.

   a. **Determine driver**:
      - **Explicit override**: read env `RALPH_PRIMARY_CLI`. Accepts `claude` or `codex` (case-insensitive). Anything else falls through to auto-detect.
      - **Auto-detect** (when no override):
        - If `which codex` succeeds AND `which claude` does not → driver = `codex`.
        - If `which claude` succeeds AND `which codex` does not → driver = `claude`.
        - If both succeed → driver = `claude` (back-compat default; the skill body originally only ran in Claude).
        - If neither succeeds → skip with note "no review CLI available — proceeding to /pr".
   b. **Determine reviewer**: source `scripts/xreview-helpers.sh` and call `pick_reviewer` with the driver from step 2.a to get the opposite CLI: `. scripts/xreview-helpers.sh; REVIEWER=$(pick_reviewer "$DRIVER")`. This keeps the "reviewer is always the driver's opposite" mapping in one grep-able, unit-tested place (`tests/test-xreview-helpers.sh`) instead of duplicating it inline.

   Record the resolved driver and reviewer for the rest of the skill. Both must be reported in the triage report header.

3. **Check reviewer availability**:
   - reviewer = `codex`: run `./scripts/codex-check.sh`. If exit 1: note "Codex not available — skipping to /pr" and invoke /pr.
   - reviewer = `claude`: run `command -v claude` via Bash. If not found: note "Claude CLI not available — skipping to /pr" and invoke /pr.

4. **Invoke reviewer**:
   - Determine base branch via Bash: `. scripts/xreview-helpers.sh; BASE=$(detect_base_branch)` — resolution order: (1) `$RALPH_XREVIEW_BASE` if set and non-empty (explicit override); (2) `git symbolic-ref --quiet --short refs/remotes/origin/HEAD` with leading `origin/` stripped (repo default branch); (3) `main` if `refs/heads/main` exists, else `master`.
   - Check the diff is non-empty: `git diff "$BASE"...HEAD --quiet` — if exit 0 (no diff), skip with a note and proceed to /pr.
   - **reviewer = `codex`**: `codex exec review --base "$BASE"`
     The native reviewer analyzes the full diff and returns structured findings with severity, affected files, and recommendations.
   - **reviewer = `claude`**: `claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --permission-mode auto --output-format json` with a prompt that instructs Claude to act as an adversarial diff reviewer (see prompt template at the end of this file). (the variable is read from the environment — set it directly or source `scripts/ralph-config.sh`, which exports it; unset falls back to `opus`)

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
   - All findings in their classified sections (ACTION_REQUIRED, WORTH_CONSIDERING, DISMISSED)
   - Triage rationale (1-2 sentences per finding to limit token cost)
   - Dismissal reasons with category for all DISMISSED findings
   - Summary counts in the header, as a single canonical line: `- After triage: ACTION_REQUIRED=N, WORTH_CONSIDERING=N, DISMISSED=N`
   - Current cycle and cap (from Step 1) in the header, e.g. `Cycle: 2/2 (cap reached)`

   **Verify the counts**: after writing the report, source `scripts/xreview-helpers.sh` and re-derive each category's count from the file just written — `. scripts/xreview-helpers.sh; count_triage_findings docs/reports/cross-review-triage-<slug>.md ACTION_REQUIRED` (repeat for `WORTH_CONSIDERING` and `DISMISSED`) — and confirm the three numbers match the summary line above before Step 7/8 branch on them. This reuses the same parser `tests/test-xreview-helpers.sh` pins, so a mismatch means the header line or table formatting drifted from `docs/reports/templates/cross-review-triage-report.md`.

7. **Present triaged findings**:
   Display findings grouped by classification:
   - **ACTION_REQUIRED**: Show full details (finding + triage rationale + affected files). Header: "Action required (ACTION_REQUIRED)"
   - **WORTH_CONSIDERING**: Show full details. Header: "Worth considering (WORTH_CONSIDERING)"
   - **DISMISSED**: Show count and note that details are in the triage report. Example: "Dismissed: N items (see docs/reports/cross-review-triage-<slug>.md for details)"

8. **User decision**:
   Branch based on triage results **and** on whether the pipeline cycle cap has been reached (see Step 1 — cycle vs `RALPH_STANDARD_MAX_PIPELINE_CYCLES`).

   Let `CAP_REACHED = (cycle >= RALPH_STANDARD_MAX_PIPELINE_CYCLES)`. At the default cap of 2, `CAP_REACHED` is true during the second (and final) `/cross-review` run.

   **Case A — ACTION_REQUIRED findings exist**:
   - If NOT `CAP_REACHED`: Use AskUserQuestion (Claude) or numbered stdin prompt (Codex):
     - Question: "Cross-review reported ACTION_REQUIRED findings. How do you want to proceed?"
     - Options:
       1. Fix — fix ACTION_REQUIRED issues, then re-run the full post-implementation pipeline: /self-review → /verify → /test → /sync-docs → /cross-review
       2. Also review WORTH_CONSIDERING — review both ACTION_REQUIRED and WORTH_CONSIDERING, then decide
       3. Acknowledge and create PR — proceed to /pr
   - If `CAP_REACHED` (cap-reached flow):
     - Question: "Pipeline re-run cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`) reached, but ACTION_REQUIRED findings remain. What do you want to do?"
     - Options:
       1. Raise the cap temporarily and re-run — have the user set a higher `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (e.g. export it) and re-run the pipeline
       2. Record findings and create PR — add unresolved ACTION_REQUIRED findings to the PR body's Known gaps section, then proceed to /pr
       3. Abort — stop without creating a PR; the user will resume manually

   **Case B — No ACTION_REQUIRED, but WORTH_CONSIDERING exist**:
   - If NOT `CAP_REACHED`:
     - Question: "Cross-review reported WORTH_CONSIDERING findings (no ACTION_REQUIRED). How do you want to proceed?"
     - Options:
       1. Review and fix — review WORTH_CONSIDERING findings, fix as needed, then re-run the full post-implementation pipeline
       2. Create PR — proceed to /pr
   - If `CAP_REACHED`:
     - Question: "Pipeline re-run cap (`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`) reached, but WORTH_CONSIDERING findings remain. What do you want to do?"
     - Options:
       1. Raise the cap temporarily and re-run
       2. Create PR — add unresolved WORTH_CONSIDERING findings to the PR body's Known gaps section, then proceed to /pr
       3. Abort

   **Case C — All findings DISMISSED (or no findings)**:
   Note "Cross-review: all findings triaged (no ACTION_REQUIRED) — triage report: docs/reports/cross-review-triage-<slug>.md" and proceed to /pr.

9. **Proceed**:
   - **Non-cap re-run** (Case A / Case B, `CAP_REACHED = false`): If `active-plan.json` exists, increment `cycle-count.json` (`cycle += 1`), then guide the user back to `/self-review`. The incremented cycle represents "the pass the user is about to enter".
   - **Cap-reached Option 1** ("Raise the cap temporarily and re-run"): Do **NOT** increment `cycle-count.json`. Instruct the user to `export RALPH_STANDARD_MAX_PIPELINE_CYCLES=<current cycle + 1>` (or higher) before re-running, so the unchanged `cycle` falls below the new cap. Then guide them back to `/self-review`.
   - If the user chooses `/pr`: invoke /pr (which is responsible for deleting `active-plan.json` and `cycle-count.json` on success).
   - If the user chooses Abort: stop without invoking /pr; leave state files in place so the next `/work` can resume.

## CLI execution modes

| Aspect | Claude Code (driver = claude) | Codex (driver = codex) |
|--------|-------------------------------|------------------------|
| Reviewer invocation | `codex exec review --base "$BASE"` | `claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --permission-mode auto --output-format json` (adversarial reviewer prompt) |
| Step 8 user dialog | Structured choices via `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit 1–3 |
| Triage execution | inline (main context) | inline — chained within a single agent |
| Output file | `docs/reports/cross-review-triage-<slug>.md` | Same |

Driver / reviewer detection reuses the Step 2 logic, which sources `scripts/xreview-helpers.sh` and calls `pick_reviewer` to compute the reviewer from the resolved driver. Exporting `RALPH_PRIMARY_CLI` makes the choice unambiguous. Step 6's count verification sources the same file's `count_triage_findings` (fake-CLI-free regression coverage for both helpers lives in `tests/test-xreview-helpers.sh`).

## Insight event (best-effort)

After writing the triage report (Step 6), append one insight event (errors are non-fatal):
```
./scripts/insights-append.sh --slug <slug> --flow standard --phase cross_review \
  --verdict <pass|action_required> --action-required <N> --worth-considering <N> \
  --dismissed <N> --source skill || true
```
Use `--verdict action_required` when ACTION_REQUIRED findings exist; `pass` otherwise.

## What /cross-review does NOT do

- **Auto-fix**: Findings are advisory only. No code changes.
- **Block the flow**: If the reviewer CLI is unavailable, flow continues silently.
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

When `reviewer = claude`, run `claude -p` with a prompt similar to:

```
You are an adversarial diff reviewer. Default to skepticism — assume the diff
can fail in subtle, high-cost ways until evidence says otherwise. Review the
diff between origin/$BASE and HEAD for: (1) correctness issues, (2) security
concerns, (3) error handling gaps, (4) logic errors, (5) missing test coverage,
(6) blind spots specific to the change. Report each finding with severity
[HIGH/MEDIUM/LOW], affected file:line, what can go wrong, why it is vulnerable,
likely impact, and concrete change to reduce the risk. Prefer one strong
finding over several weak ones. If the diff looks solid, say so directly.
```

Pipe the diff via stdin or fetch it inside the prompt with `git diff $BASE...HEAD`.
