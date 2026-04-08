---
name: codex-review
description: >
  Optional cross-model second opinion on the current diff using Codex.
  Invoked automatically after /test passes. If Codex CLI is unavailable,
  silently skips and proceeds to /pr. Findings are triaged by Claude Code
  using implementation context before presentation to the user.
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion, Write
---
Provide a cross-model second opinion on the current diff before PR creation.

## Goals

- Catch blind spots that single-model review may miss
- Leverage a different model's perspective for cross-validation
- Triage findings using implementation context to reduce noise
- Present findings as advisory — never auto-apply

## Steps

1. **Check Codex availability**:
   Run `./scripts/codex-check.sh` via Bash.
   If exit 1: note "Codex CLI not available — skipping to /pr" and invoke /pr.

2. **Invoke Codex review**:
   - Determine base branch via Bash: `BASE=$(git rev-parse --abbrev-ref HEAD@{upstream} 2>/dev/null | sed 's|origin/||' || echo main)`
   - Check the diff is non-empty: `git diff "$BASE"...HEAD --quiet` — if exit 0 (no diff), skip with a note and proceed to /pr.
   - Run native Codex reviewer via Bash:
     `codex exec review --base "$BASE"`
   - The native reviewer analyzes the full diff and returns structured findings with severity, affected files, and recommendations. It covers: correctness issues, security concerns, error handling gaps, logic errors, and missing test coverage.

3. **Triage findings** (new — noise reduction):
   Triage each Codex finding using implementation context. This step runs inline (main context) because triage value depends on knowing *why* the code was written that way.

   **Load triage context:**
   - Read the active plan from `docs/plans/active/`
   - Read the self-review report from `docs/reports/` (if available)
   - Read the verify report from `docs/reports/` (if available)
   - Consider implementation decisions made during the current session

   **If Codex returned non-structured output** (no clear severity/file/recommendation per finding): skip triage, fall back to Step 5-legacy (present all findings as-is, same as pre-triage behavior).

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

4. **Write triage report**:
   Write the triage report to `docs/reports/codex-triage-<plan-slug>.md` using the template at `docs/reports/templates/codex-triage-report.md`. Include:
   - All findings in their classified sections (ACTION_REQUIRED, WORTH_CONSIDERING, DISMISSED)
   - Triage rationale (1-2 sentences per finding to limit token cost)
   - Dismissal reasons with category for all DISMISSED findings
   - Summary counts in the header

5. **Present triaged findings**:
   Display findings grouped by classification:
   - **ACTION_REQUIRED**: Show full details (finding + triage rationale + affected files). Header: "要対応 (ACTION_REQUIRED)"
   - **WORTH_CONSIDERING**: Show full details. Header: "検討推奨 (WORTH_CONSIDERING)"
   - **DISMISSED**: Show count and note that details are in the triage report. Example: "除外: N 件（詳細は docs/reports/codex-triage-<slug>.md を参照）"

6. **User decision**:
   Branch based on triage results:

   **Case A — ACTION_REQUIRED findings exist**:
   Use AskUserQuestion:
   - Question: "Codex レビューで要対応の指摘があります。どう対応しますか？"
   - Options:
     1. 修正する — fix ACTION_REQUIRED issues, then re-run /self-review → /verify → /test → /codex-review
     2. WORTH_CONSIDERING も確認する — review both ACTION_REQUIRED and WORTH_CONSIDERING, then decide
     3. 指摘を確認済み、PR を作成する — proceed to /pr

   **Case B — No ACTION_REQUIRED, but WORTH_CONSIDERING exist**:
   Use AskUserQuestion:
   - Question: "Codex レビューで検討推奨の指摘があります（要対応はなし）。どう対応しますか？"
   - Options:
     1. 検討して修正する — review WORTH_CONSIDERING findings, fix as needed, then re-run pipeline
     2. PR を作成する — proceed to /pr

   **Case C — All findings DISMISSED (or no findings)**:
   Note "Codex: 全指摘トリアージ済み（要対応なし）— トリアージレポート: docs/reports/codex-triage-<slug>.md" and proceed to /pr.

7. **Proceed**:
   Based on user choice, either guide them back to fix-and-revalidate, or invoke /pr.

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
