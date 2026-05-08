# Cross-review triage report: Ralph Loop Codex driver (Phase 2)

- Date: 2026-05-08
- Plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md`
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context, post-/sync-docs)
- Self-review cross-ref: yes (`docs/reports/self-review-2026-05-08-ralph-loop-codex-driver.md`)
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-05-08-ralph-loop-codex-driver.md` (pinned via `.harness/state/standard-pipeline/active-plan.json`)
- Self-review report: `docs/reports/self-review-2026-05-08-ralph-loop-codex-driver.md` (verdict MERGE; 5 MEDIUM, 3 already fixed in 085bad7, 1 closed in 3351df2)
- Verify report: `docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md` (verdict pass)
- Test report: `docs/reports/test-2026-05-08-ralph-loop-codex-driver.md` (verdict PASS, 290/290)
- Implementation context summary: Phase 2 introduces a `claude→codex` reviewer-inversion path that (a) writes a triage report following the same template the pipeline parses, and (b) was deliberately scoped to keep the shell `./scripts/ralph` wrapper unchanged (Go CLI handles TOML). Codex's two findings probe exactly the boundary between these two design choices.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] `grep -c` over the entire triage report counts the literal `## ACTION_REQUIRED` / `## WORTH_CONSIDERING` / `## DISMISSED` headings (and the `After triage:` summary line) as findings. A clean report with zero findings still produces ≥2 false matches per category, forcing the outer loop to regress unnecessarily. | Real bug, and the new `adversarial-claude.md` prompt I added explicitly asks Claude to follow that template — so this PR makes the bug *visible on the codex-driver path*. Fix is minimal: parse the header summary line `After triage: ACTION_REQUIRED=N, ...` (canonical count) and fall back to "count `^|` table rows under each heading" if the line is missing. | `scripts/ralph-pipeline.sh` (parser block, currently lines 803-805) |
| 2 | [P2] `[loop] driver = "codex"` in `ralph.toml` is silently ignored when the user runs `./scripts/ralph run` (shell wrapper) or `./scripts/ralph-orchestrator.sh` directly — only the Go `ralph run` path propagates TOML. `docs/recipes/ralph-loop.md` shows the TOML config without naming this asymmetry. | Real doc/UX bug. The plan's Design decisions section deliberately scoped TOML→runtime to the Go CLI path (`./scripts/ralph` shell wrapper を直接叩かれた場合は env 未経由なら既定の claude にフォールバック), so the implementation is correct — but the recipe and SKILL.md documentation imply both surfaces honour TOML. Fix is a 1-paragraph doc clarification + an explicit env-var line in the TOML example. | `docs/recipes/ralph-loop.md`, `templates/base/docs/recipes/ralph-loop.md`, `.claude/skills/loop/SKILL.md` (and Codex mirror) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

(none — both findings are concrete and fixable now)

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

(none)

## Decision

Cycle 1/2. Both findings are ACTION_REQUIRED and the cap permits one fix-and-revalidate cycle. Recommended next step: apply both fixes in a single follow-up commit, then re-run the post-implementation pipeline (cycle 2).
