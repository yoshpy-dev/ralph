# Self-review report: upgrade-partial-hunks

- Plan: `docs/plans/active/2026-05-18-upgrade-partial-hunks.md`
- Related issue: #97
- Scope: diff quality only for the first Issue #97 implementation slice,
  including the follow-up prompt wording fix.
- Diff reviewed: manifest v2 metadata, baseline cache helpers, upgrade dry-run/pager flow, baseline-gated file-level prompt, tests, and docs.

## Findings

No blocking diff-quality findings after the prompt wording fix.

## Checks performed

- Reviewed changed code paths in `internal/scaffold/manifest.go`, `internal/scaffold/baseline.go`, `internal/cli/init.go`, `internal/cli/upgrade.go`, and `internal/upgrade/diff.go`.
- Checked for unrelated broad refactors: none found. Changes stay inside upgrade/init/manifest behavior plus matching docs/tests.
- Checked path handling in baseline cache: `WriteBaseline` rejects non-local template paths, and `ReadBaseline` requires `.ralph/baseline/` prefix before reading.
- Checked prompt wording: baseline-backed prompt is explicitly file-scoped
  (`apply template file / keep local file / edit`); no `skip`, `next`, or
  `quit` option appears.
- Checked diff display: upgrade UI omits hunk headers and template/local hash
  summaries; low-level diff rendering remains unchanged.
- Checked deferred behavior is explicit: full 3-way planner/manual edit/pre-apply summary is recorded in `docs/tech-debt/README.md`.

## Recommendation

Mergeable as a first slice for #97. It intentionally does not close the whole issue because true multi-hunk 3-way editing remains tracked follow-up work.
