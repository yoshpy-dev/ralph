# Codex triage report: upgrade-detect-local-edits

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-upgrade-detect-local-edits.md`
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes (`docs/reports/self-review-2026-04-22-upgrade-detect-local-edits.md`)
- Total Codex findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-04-22-upgrade-detect-local-edits.md`
- Self-review report: `docs/reports/self-review-2026-04-22-upgrade-detect-local-edits.md`
- Verify report: `docs/reports/verify-2026-04-22-upgrade-detect-local-edits.md` (called out the `--force` / unmanaged interaction as a coverage gap — Codex upgrades that to a functional contract violation)
- Implementation context summary: The feature introduces `Managed=false` to mean "user-owned, silent-skip." Two boundary conditions were not exercised by current tests: (a) `--force` interaction with already-unmanaged entries, (b) template-side removal of a file that the user previously chose to own.

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | `--force` does not re-adopt files previously skipped to `Managed=false`. The `ComputeDiffsWithManifest` early-return emits `ActionSkip` for all unmanaged entries before `runUpgrade` ever checks the `force` flag, so `ralph upgrade --force` silently leaves them untouched. | Real issue: contradicts both the cobra flag help (`overwrite all files without prompting`) and the spec written during `/sync-docs` (`--force` → `{Hash: newHash, Managed: true}`). Verify report's "Coverage gap #1" foreshadowed this — Codex correctly promotes it to a functional regression. Worth fixing: the spec commit (`5465679`) is on the same branch, so fixing here keeps spec and code in lockstep. | `internal/upgrade/diff.go:84-97`, `internal/cli/upgrade.go` force branch |
| 2 | Template-side removal of a user-owned path silently un-tracks the manifest entry. Removal detection does not check `Managed`, so a `Managed=false` entry that disappears from the template becomes `ActionRemove`; `runUpgrade` drops the entry, and a later reintroduction of that path is no longer silent-skipped. | Real issue: breaks the new "user owns this forever until `--resync`" contract. Worth fixing: the whole point of `Managed=false` is convergence across arbitrary template changes, not just across same-version reruns. | `internal/upgrade/diff.go` removal loop, `internal/cli/upgrade.go` `ActionRemove` branch |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|

(none)

## DISMISSED

| # | Codex finding | Dismissal reason | Category |
|---|---------------|------------------|----------|

(none)

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
