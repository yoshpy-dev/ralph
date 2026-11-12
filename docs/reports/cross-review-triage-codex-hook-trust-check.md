# Cross-review triage report: codex-hook-trust-check

- Date: 2026-11-12
- Plan: docs/plans/active/2026-11-12-codex-hook-trust-check.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-codex-hook-trust-check.md (D1–D6, AC1–AC9; Progress checklist records S1–S6 and the pipeline results)
- Self-review report: docs/reports/self-review-2026-11-12-codex-hook-trust-check.md (Merge: yes, LOW 9)
- Verify report: docs/reports/verify-2026-11-12-codex-hook-trust-check.md (pass)
- Test report: docs/reports/test-2026-11-12-codex-hook-trust-check.md (pass; the first attempt failed on ENOSPC)
- Implementation context summary: the branch adds `ralph doctor`'s Codex hook trust check and a stderr note at codex launches, keyed the way codex 0.166.0 keys hook approvals (the root is resolved like codex's `resolve_root`, both spellings of a symlinked root are checked). The rule for this series: a gap this change introduces that makes a notice disappear (a false pass) is fixed; older patterns and over-notifying differences are recorded. All three findings below make the new check report "approved" (or "nothing to approve") while codex runs the hooks unapproved or not at all, so they are false passes the change introduced.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Nested project hook files are not checked: codex loads `.codex/hooks.json` of every project layer between the checkout root and the cwd (e.g. `/repo/sub/.codex/hooks.json` for a seat in `/repo/sub`), but the check reads only the root's, so approved root hooks hide unapproved nested ones; map linked-worktree layers to the main checkout keeping the relative directory. | Real: codex's `root_checkout_hooks_folder_for_dir` maps each layer dir as `<repo root>/<relative dir>/.codex`, and project layers are discovered between the root and the cwd. ralph's scaffold has only the root layer and seats usually run at the worktree root, but a `--cwd` in a subdirectory with its own `.codex/hooks.json`, or doctor run from such a subdirectory, gets a pass that hides unapproved hooks. Fix: enumerate the layers from the root down to the dir (lexically), read each existing `.codex/hooks.json`, key it under the mapped path. | internal/cli/codex_hook_trust.go |
| 2 | [P2] Null hook structures are accepted: `{"hooks": null}` (or null groups/handlers) in the main checkout's hooks.json parses as zero slots and the check passes with "no hook needs approval", while codex rejects the structure and loads no hooks; Check 3 reads only the worktree's own file, so nothing reports it from a linked worktree. | Real and cheap: a false pass of the "nothing to approve" kind while the hooks codex should run are absent. Fix: treat null or non-object/array structures and handlers without the fields codex requires as invalid (info when it is Check 3's file, warn otherwise). | internal/cli/codex_hook_trust.go |
| 3 | [P2] The inferred main checkout is not verified to own the common git directory: for a linked worktree of a bare repository at `/outer/cache.git` with an unrelated `/outer/.git`, ralph picks `/outer` as the main checkout; codex verifies ownership and falls back to the worktree's own hooks. | Real: the plan mirrored codex's `resolve_root` but skipped its ownership check (noted in a doc comment as satisfied by worktrees git creates — not true for this layout). The result is checking unrelated approved hooks and missing the worktree's own. Fix: accept `<main>` only when its `.git` (a directory, or a gitdir file) resolves, canonically, to the worktree's common directory; otherwise the root is the worktree itself. | internal/cli/codex_hook_trust.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
