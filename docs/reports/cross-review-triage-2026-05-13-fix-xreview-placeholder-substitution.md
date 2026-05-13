# Cross-review triage report: fix-xreview-placeholder-substitution

- Date: 2026-05-13
- Plan: docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md`
- Self-review report: `docs/reports/self-review-2026-05-13-fix-xreview-placeholder-substitution.md` (cycle 1) and `…-cycle2.md` (cycle 2 — verdict: merge, 0 CRITICAL / 0 HIGH / 0 MEDIUM)
- Verify report: `docs/reports/verify-2026-05-13-fix-xreview-placeholder-substitution.md` (all 9 ACs verified, static analysis clean)
- Implementation context summary: Six commits land. Renderer + allowlist guard (`0304686`); renderer unit test 54/54 (`4f15681`); end-to-end gate-regression test 21/21 (`d2dd875`); `templates/base/` mirror (`f3363b6`); plan progress (`fd3e958`); cycle-2 fix wiring `_render_failed` into the gate decision and absorbing self-review LOWs (`12a1984`); sync-docs (`7f38512`). All verifiers green.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

(none)

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | **[P2] Allowlist guard scans the rendered output, so a `BASE_BRANCH` or `REPORTS_DIR` value that legitimately contains literal `${SOMETHING}` text (e.g. a git ref like `release-${YEAR}`) is misclassified as an unresolved placeholder. `_render_failed=1` regresses the gate every outer-loop cycle without invoking the reviewer.** Suggested fix: scan the **template** (input file) for unsupported placeholders *before* injecting replacement values, or exclude occurrences originating from the substituted values. | **Real issue, low probability, not blocking.** Git refs may contain `$`, `{`, `}` (the forbidden set is `~^:?*[\<space><control>` plus `.lock` suffix), so `release-${YEAR}` as a branch name is technically valid. In practice such refs are extraordinarily rare — common conventions use `release/3.5`, dates, or hashes. Even if encountered, the failure mode is loud (clear error in the log, gate fails closed with a named status) and the manual workaround is "rename the branch" — orders of magnitude less severe than the silent-bypass bug from issue #50 this PR closes. The canonical fix (scan template pre-render) is also a small, clean refactor — better done deliberately as a follow-up than rushed into the second pipeline cycle. Classified WORTH_CONSIDERING per the conservative principle (real issue, debatable timing). | `scripts/ralph-pipeline.sh:836-841`, `templates/base/scripts/ralph-pipeline.sh:836-841` |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

(none)

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Follow-up note

If WORTH_CONSIDERING #1 is acted on later, the canonical fix is:

1. Move the `grep -oE '\$\{[A-Z_][A-Z0-9_]*\}'` allowlist scan **before** the awk substitution, against `$_adv_prompt` (the template), not `$_rendered_prompt`.
2. Filter the result against the supported placeholder set `{BASE_BRANCH, REPORTS_DIR}` and fail closed only on unsupported tokens.
3. Drop the post-render scan, or keep it as a paranoid post-substitution sanity check that explicitly subtracts any token that also appeared in `BASE_BRANCH` / `REPORTS_DIR` raw values.

This decouples "the template uses only known placeholders" (the actual contract) from "the rendered output contains no shell-style text" (a false proxy for that contract).
