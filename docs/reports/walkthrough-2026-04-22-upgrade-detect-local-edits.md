# Walkthrough: ralph upgrade local-edit detection

- Date: 2026-04-22
- Plan: `docs/plans/archive/2026-04-22-upgrade-detect-local-edits.md`
- Branch: `feat/upgrade-detect-local-edits`
- Diff size: ~2,200 lines (code + tests + docs + evidence)

## Why this PR is large

Net production-code change is compact (~300 lines across 4 files). The rest is:
- ~300 lines of unit + integration tests
- ~800 lines of reports (self-review / verify / test / sync-docs / codex-triage, pass 1 + pass 2)
- ~500 lines of evidence logs
- ~250 lines of plan + spec + tech-debt updates

Reading order for reviewers who want the shortest path:

1. **Plan** — `docs/plans/archive/2026-04-22-upgrade-detect-local-edits.md` (what + why, acceptance criteria)
2. **Implementation**
   - `internal/upgrade/diff.go` — three small behavior changes
   - `internal/upgrade/unified_diff.go` — new file, pure LCS implementation
   - `internal/cli/upgrade.go` — DI refactor + resolveConflict rewrite
   - `internal/scaffold/manifest.go` — one new helper
3. **Tests**
   - `internal/upgrade/diff_test.go` — new assertions for all branch changes
   - `internal/upgrade/unified_diff_test.go` — new file
   - `internal/cli/cli_test.go` — interactive-path integration tests
4. **Codex triage** — `docs/reports/codex-triage-2026-04-22-upgrade-detect-local-edits.md` explains the two ACTION_REQUIRED findings and their pass-2 fixes.

## Core behavior changes in 3 rules

### 1. Local edit detection (template unchanged)

`internal/upgrade/diff.go` — the "template hash == manifest hash" branch now splits on disk state:
- disk hash == manifest hash → `ActionSkip` (existing)
- disk hash ≠ manifest hash → `ActionConflict` (new; surfaces user edits)

### 2. `Managed=false` as "user owns this"

New `Managed=false` semantics make `skip` resolution convergent:
- `skip` resolution writes `{Hash: diskHash, Managed: false}` — stops auto-management
- `ComputeDiffsWithManifest` returns `ActionSkip` for `Managed=false` entries (no prompt, no auto-update)
- Removal-detection loop preserves `Managed=false` entries as `ActionSkip` (not `ActionRemove`) across template deletions
- `--force` re-adopts `Managed=false` entries: writes template content, flips `Managed=true`

### 3. Unified diff viewer

`[d]iff` in the prompt now shows a real unified diff (`internal/upgrade/unified_diff.go`, LCS-based) instead of a hash-only summary. Labels: `--- local` / `+++ template (version)`. Fallback to hash summary on disk-read failure.

## I/O DI refactor

`runUpgrade` is now a thin shim over `runUpgradeIO(targetDir, force, in, out, errOut)`. All progress text goes through a `writef` helper. This is what makes `TestRunUpgrade_Interactive*` tests possible — they script stdin and assert on captured stdout without touching the real process streams.

## Codex review cycle

- **Pass 1**: plan reviewed adversarially — HIGH "prompt storm" + MEDIUM "untested interactive path" both incorporated into the plan before implementation.
- **Pass 2**: diff reviewed — two ACTION_REQUIRED findings:
  - `--force` ignored `Managed=false` entries (contract violation)
  - `Managed=false` entries dropped across template removals (convergence violation)
- Both addressed in commit `a920352` with dedicated tests. Pass 2 pipeline (self-review / verify / test / sync-docs) all green.

## Risks

- Existing users with an unmanaged state (heal-path empty hash + local edit) will now get one conflict prompt on next upgrade — addressed by the convergence contract (skip once, silent thereafter).
- No `--resync <path>` yet: tech debt logged in `docs/tech-debt/README.md`. `--force` is the current tree-wide escape hatch.

## Rollback

Revert the merge commit. Manifest schema is unchanged — `Managed` was always present; this PR just gave `Managed=false` meaning in the upgrade flow. Old code treats all entries as if `Managed=true`, which for existing users is the status quo.
