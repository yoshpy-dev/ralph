# Self-review report: upgrade-partial-hunks

- Plan: `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`
- Related issue: #97
- Scope: diff quality only for the first Issue #97 implementation slice.
- Diff reviewed: manifest v2 metadata, baseline cache helpers, upgrade dry-run/pager flow, baseline-gated hunk prompt, tests, and docs.

## Findings

No blocking diff-quality findings.

## Checks performed

- Reviewed changed code paths in `internal/scaffold/manifest.go`, `internal/scaffold/baseline.go`, `internal/cli/init.go`, `internal/cli/upgrade.go`, and `internal/upgrade/diff.go`.
- Checked for unrelated broad refactors: none found. Changes stay inside upgrade/init/manifest behavior plus matching docs/tests.
- Checked path handling in baseline cache: `WriteBaseline` rejects non-local template paths, and `ReadBaseline` requires `.ralph/baseline/` prefix before reading.
- Checked prompt wording: hunk prompt contains only `apply / keep / edit / skip file`; no `next` or `quit` option appears.
- Checked deferred behavior is explicit: full 3-way planner/manual edit/final summary is recorded in `docs/tech-debt/README.md`.

## Recommendation

Mergeable as a first slice for #97. It intentionally does not close the whole issue because true multi-hunk 3-way editing remains tracked follow-up work.
