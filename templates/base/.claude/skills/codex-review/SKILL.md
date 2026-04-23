---
name: codex-review
description: >
  Optional cross-model second opinion on the current diff using Codex.
  Runs inline in the main context (not delegated as subagent) after /sync-docs
  completes. If Codex CLI is unavailable, silently skips and proceeds to /pr.
  Findings are triaged by Claude Code using implementation context before
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
   b. (Persisted-identity mode only) Read `.harness/state/standard-pipeline/cycle-count.json`. If its `plan_path` matches `active-plan.json`, use its `cycle`. If missing, initialize `{"plan_path": "<path>", "cycle": 1}` (first /codex-review run of this plan). If its `plan_path` does **not** match, warn and treat as fallback mode for this run (do not overwrite — `/work` is responsible for resolving mismatched state).
   c. Read `RALPH_STANDARD_MAX_PIPELINE_CYCLES` by sourcing `./scripts/ralph-config.sh` in a subshell (default `2`).
   d. Record the current cycle number and the cap for use in Step 7.

   **Hard prohibition**: Do NOT rediscover the plan by rescanning `docs/plans/active/` once `active-plan.json` exists. Always consume the persisted path. This prevents cross-plan counter leakage when multiple plans coexist.

2. **Check Codex availability**:
   Run `./scripts/codex-check.sh` via Bash.
   If exit 1: note "Codex CLI not available — skipping to /pr" and invoke /pr.

3. **Invoke Codex review**:
   - Determine base branch via Bash: `BASE=$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||' || echo main)`
   - Check the diff is non-empty: `git diff "$BASE"...HEAD --quiet` — if exit 0 (no diff), skip with a note and proceed to /pr.
   - Run native Codex reviewer via Bash:
     `codex exec review --base "$BASE"`
   - The native reviewer analyzes the full diff and returns structured findings with severity, affected files, and recommendations. It covers: correctness issues, security concerns, error handling gaps, logic errors, and missing test coverage.

4. **Triage findings** (new — noise reduction):
   Triage each Codex finding using implementation context. This step runs inline (main context) because triage value depends on knowing *why* the code was written that way.

   **Load triage context:**
   - Read the active plan using the path recorded in `active-plan.json` from Step 1 — do not rescan `docs/plans/active/`. If `active-plan.json` is absent (fallback mode), use the path resolved in Step 1's fallback.
   - Read the self-review report from `docs/reports/` (if available)
   - Read the verify report from `docs/reports/` (if available)
   - Consider implementation decisions made during the current session

   **If Codex returned non-structured output** (no clear severity/file/recommendation per finding): skip triage, fall back to Step 6 legacy behavior (present all findings as-is, same as pre-triage behavior).

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

5. **Write triage report**:
   Write the triage report to `docs/reports/codex-triage-<plan-slug>.md` using the template at `docs/reports/templates/codex-triage-report.md`. Include:
   - All findings in their classified sections (ACTION_REQUIRED, WORTH_CONSIDERING, DISMISSED)
   - Triage rationale (1-2 sentences per finding to limit token cost)
   - Dismissal reasons with category for all DISMISSED findings
   - Summary counts in the header
   - Current cycle and cap (from Step 1) in the header, e.g. `Cycle: 2/2 (cap reached)`

6. **Present triaged findings**:
   Display findings grouped by classification:
   - **ACTION_REQUIRED**: Show full details (finding + triage rationale + affected files). Header: "要対応 (ACTION_REQUIRED)"
   - **WORTH_CONSIDERING**: Show full details. Header: "検討推奨 (WORTH_CONSIDERING)"
   - **DISMISSED**: Show count and note that details are in the triage report. Example: "除外: N 件（詳細は docs/reports/codex-triage-<slug>.md を参照）"

7. **User decision**:
   Branch based on triage results **and** on whether the pipeline cycle cap has been reached (see Step 1 — cycle vs `RALPH_STANDARD_MAX_PIPELINE_CYCLES`).

   Let `CAP_REACHED = (cycle >= RALPH_STANDARD_MAX_PIPELINE_CYCLES)`. At the default cap of 2, `CAP_REACHED` is true during the second (and final) `/codex-review` run.

   **Case A — ACTION_REQUIRED findings exist**:
   - If NOT `CAP_REACHED`: Use AskUserQuestion:
     - Question: "Codex レビューで要対応の指摘があります。どう対応しますか？"
     - Options:
       1. 修正する — fix ACTION_REQUIRED issues, then re-run the full post-implementation pipeline: /self-review → /verify → /test → /sync-docs → /codex-review
       2. WORTH_CONSIDERING も確認する — review both ACTION_REQUIRED and WORTH_CONSIDERING, then decide
       3. 指摘を確認済み、PR を作成する — proceed to /pr
   - If `CAP_REACHED` (cap-reached flow): Use AskUserQuestion:
     - Question: "パイプライン再実行の上限（`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`）に到達しました。要対応の指摘が残っていますが、どうしますか？"
     - Options:
       1. 上限を一時的に引き上げて再実行 — have the user set a higher `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (e.g. export it) and re-run the pipeline
       2. 指摘は記録し PR を作成する — add unresolved ACTION_REQUIRED findings to the PR body's Known gaps section, then proceed to /pr
       3. 中止 — stop without creating a PR; the user will resume manually

   **Case B — No ACTION_REQUIRED, but WORTH_CONSIDERING exist**:
   - If NOT `CAP_REACHED`: Use AskUserQuestion:
     - Question: "Codex レビューで検討推奨の指摘があります（要対応はなし）。どう対応しますか？"
     - Options:
       1. 検討して修正する — review WORTH_CONSIDERING findings, fix as needed, then re-run the full post-implementation pipeline: /self-review → /verify → /test → /sync-docs → /codex-review
       2. PR を作成する — proceed to /pr
   - If `CAP_REACHED` (cap-reached flow, Case B variant): Use AskUserQuestion:
     - Question: "パイプライン再実行の上限（`RALPH_STANDARD_MAX_PIPELINE_CYCLES=<cap>`）に到達しました。検討推奨の指摘が残っていますが、どうしますか？"
     - Options:
       1. 上限を一時的に引き上げて再実行 — have the user export a higher `RALPH_STANDARD_MAX_PIPELINE_CYCLES` and re-run the pipeline
       2. PR を作成する — add unresolved WORTH_CONSIDERING findings to the PR body's Known gaps section, then proceed to /pr
       3. 中止 — stop without creating a PR; the user will resume manually

   **Case C — All findings DISMISSED (or no findings)**:
   Note "Codex: 全指摘トリアージ済み（要対応なし）— トリアージレポート: docs/reports/codex-triage-<slug>.md" and proceed to /pr.

8. **Proceed**:
   - **Non-cap re-run** (Case A / Case B, `CAP_REACHED = false`): If `active-plan.json` exists, increment `cycle-count.json` (`cycle += 1`), then guide the user back to `/self-review`. The incremented cycle represents "the pass the user is about to enter".
   - **Cap-reached Option 1** ("上限を一時的に引き上げて再実行"): Do **NOT** increment `cycle-count.json`. Instruct the user to `export RALPH_STANDARD_MAX_PIPELINE_CYCLES=<current cycle + 1>` (or higher) before re-running, so the unchanged `cycle` falls below the new cap. Then guide them back to `/self-review`. Rationale: incrementing here would immediately re-trip the raised cap on the next `/codex-review` entry, leaving the user with zero extra passes.
   - If the user chooses `/pr`: invoke /pr (which is responsible for deleting `active-plan.json` and `cycle-count.json` on success).
   - If the user chooses 中止: stop without invoking /pr; leave state files in place so the next `/work` can resume.

## What /codex-review does NOT do

- **Auto-fix**: Findings are advisory only. No code changes.
- **Block the flow**: If Codex is unavailable, flow continues silently.
- **Replace /self-review**: Self-review (/self-review) and Codex review are complementary.
- **Suppress findings**: All findings (including DISMISSED) are recorded in the triage report for transparency.

## Anti-patterns to avoid

- Do NOT auto-apply all Codex suggestions (causes churn)
- Do NOT loop more than once without user confirmation
- Do NOT use Review Gate / Stop Hook automation
- Do NOT dismiss findings without a documented reason and category
- Do NOT classify uncertain findings as DISMISSED — use WORTH_CONSIDERING instead
