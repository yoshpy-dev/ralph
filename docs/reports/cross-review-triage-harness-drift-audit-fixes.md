# Cross-review triage report: harness-drift-audit-fixes

- Date: 2026-07-12
- Plan: N/A (docs-only audit-fix slice; spec = audit findings)
- Base branch: main (4d80723)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] "via `ralph run`" is ambiguous: the recipe's own examples use the SHELL wrapper `./scripts/ralph run`, which sources ralph-config.sh (bypassPermissions) and never reads TOML; only the Go binary `ralph run` exports the `auto` toml default | Correct — the wrapper/binary distinction is exactly the confusion the fix was meant to remove. Reworded to name shell entry points (direct + `./scripts/ralph` wrapper → `bypassPermissions`) vs Go binary `ralph run` (→ `auto`) in both recipe copies | docs/recipes/ralph-loop.md:141 + template mirror |

## Result

Fix applied inline (single-sentence wording, both copies, byte-identical);
check-sync re-passed. No re-review cycle needed beyond deterministic gates —
docs-only wording within the same finding scope.
