# Cross-review triage report: org-implementer-seat-envelope

- Date: 2026-09-17
- Plan: docs/plans/active/2026-09-16-org-implementer-seat-envelope.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-09-16-org-implementer-seat-envelope.md (D3: default model_pool now includes 5 codex entries)
- Self-review report: docs/reports/self-review-2026-09-16-org-implementer-seat-envelope.md (did not cover the partial-override load path)
- Verify report: docs/reports/verify-2026-09-16-org-implementer-seat-envelope.md (AC-6 tested the retired `[org.budget]` table being ignored, not a driver_pool-only override)
- Implementation context summary: `config.Load()` starts from `Default()` and `toml.Unmarshal` only overwrites keys present in the document. Before this branch the default pool was claude-only, so `driver_pool = ["claude"]` alone was a valid override. With codex entries in the default pool, the inherited codex entries now fail the "model_pool entry driver must be in driver_pool" validation. Reproduced in-session: `Load()` on a ralph.toml containing only `[org] driver_pool = ["claude"]` returns `[org].model_pool entry driver "codex" not present in [org].driver_pool [claude]`.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Preserve configs that override only `driver_pool`: with the new default codex `model_pool` entries, a ralph.toml that sets only `[org].driver_pool = ["claude"]` fails `Load()` because inherited default codex entries are rejected by the driver_pool membership check. | Real issue (reproduced) and worth fixing now: it is a backward-incompatible regression for downstream projects that disabled codex via `driver_pool` alone, and `ralph upgrade` ships the new defaults to them. Fix: when the document does not define `model_pool`, filter the inherited default pool to drivers present in the effective `driver_pool` before validation; keep explicit `model_pool` documents strictly validated. Add a regression test for the driver_pool-only override. | internal/config/config.go:146-150 (defaults), internal/config/config.go:196-226 (Load validation) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
