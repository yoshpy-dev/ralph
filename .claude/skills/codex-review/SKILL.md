---
name: codex-review
description: >
  Optional cross-model second opinion on the current diff using Codex.
  Invoked automatically after /test passes. If Codex CLI is unavailable,
  silently skips and proceeds to /pr. Findings are advisory only —
  the user decides whether to act on them.
allowed-tools: Read, Grep, Glob, Bash, AskUserQuestion, Write
---
Provide a cross-model second opinion on the current diff before PR creation.

## Goals

- Catch blind spots that single-model review may miss
- Leverage a different model's perspective for cross-validation
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

3. **Present findings**:
   Display Codex findings as a numbered list with severity levels.

4. **User decision**:
   - If Codex returned no actionable findings: note "Codex: 指摘なし" and proceed to /pr.
   - If findings exist, use AskUserQuestion:
     - Question: "Codex から diff への指摘があります。どう対応しますか？"
     - Options:
       1. 修正する — fix the identified issues, then re-run /self-review → /verify → /test → /codex-review
       2. 指摘を確認済み、PR を作成する — proceed to /pr

5. **Proceed**:
   Based on user choice, either guide them back to fix-and-revalidate, or invoke /pr.

## What /codex-review does NOT do

- **Auto-fix**: Findings are advisory only. No code changes.
- **Block the flow**: If Codex is unavailable, flow continues silently.
- **Replace /self-review**: Self-review (/self-review) and Codex review are complementary.

## Anti-patterns to avoid

- Do NOT auto-apply all Codex suggestions (causes churn)
- Do NOT loop more than once without user confirmation
- Do NOT use Review Gate / Stop Hook automation
