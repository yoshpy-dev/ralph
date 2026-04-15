# Codex triage report: Ralph Pipeline Hardening

- Date: 2026-04-15
- Plan: feat/ralph-pipeline-hardening
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Codex model: gpt-5.4 (v0.120.0)
- Rounds: 2

## Round 1 — Total findings: 3, After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=0

### Triage context

- Self-review report: docs/reports/self-review-2026-04-15-ralph-pipeline-hardening.md
- Verify report: docs/reports/verify-2026-04-15-ralph-pipeline-hardening.md
- Implementation context: Shared config module, hardcoded value elimination, signal handler addition, race condition fixes, numeric validation.

### ACTION_REQUIRED (fixed)

| # | Codex finding | Triage rationale | Fix |
|---|---------------|------------------|-----|
| 1 | [P1] INT/TERM trap does not call `exit` — orchestrator continues after cleanup | Real bug: `cleanup_on_exit` never exits. Signal handling is in scope (AC3). | Added `_on_signal()` with `_INTERRUPTED` flag and `exit 1` |
| 2 | [P1] EXIT trap overwrites "partial" status with "interrupted" on non-zero exit | Real bug: `create_unified_pr()` writes "partial" then `return 1`, EXIT trap overwrites to "interrupted". | EXIT trap now guards with `_INTERRUPTED == 1` |

### WORTH_CONSIDERING (upgraded to ACTION_REQUIRED in Round 2)

| # | Codex finding | Triage rationale |
|---|---------------|------------------|
| 3 | [P2] Missing `gh` returns exit code 1, misinterpreted as "codex ACTION_REQUIRED" | Exit code collision. Upgraded after Round 2 confirmed orchestrator-side impact. |

## Round 2 — Total findings: 2, After triage: ACTION_REQUIRED=2, DISMISSED=0

### Triage context

- Self-review v2: docs/reports/self-review-2026-04-15-ralph-pipeline-hardening-v2.md
- Verify v2: docs/reports/verify-2026-04-15-ralph-pipeline-hardening-v2.md
- Implementation context: Round 1 fixes applied. Signal handling refactored with `_INTERRUPTED` flag.

### ACTION_REQUIRED (fixed)

| # | Codex finding | Triage rationale | Fix |
|---|---------------|------------------|-----|
| 4 | [P1] `gh_unavailable` not in orchestrator terminal status patterns — slice relaunched infinitely | Real bug: orchestrator skip-launch and failure-count patterns missing `gh_unavailable`. | Added `gh_unavailable` to both patterns in orchestrator and status helpers |
| 5 | [P2] `run_outer_loop` returns 1 for both `gh_unavailable` and ACTION_REQUIRED | Exit code collision causes futile Inner Loop retries. | Changed to `return 2` with new case in caller: `_finalize "gh_unavailable"` |

## DISMISSED

(none across both rounds)

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
