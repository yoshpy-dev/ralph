# Sync-docs report: check-template-required-files-test

- Date: 2026-09-29
- Plan: `docs/plans/active/2026-09-29-check-template-required-files-test.md` (issue #183)
- Agent: `doc-maintainer` subagent (cycle 1)
- Branch: `test/check-template-required-files-test`, HEAD `ce51738`

## Checked

- `docs/tech-debt/README.md:135`: the #189 row (which replaced the old
  "`required_files` has no regression coverage" row in Slice A/B). Verified:
  the row correctly names `templates/base/.github/workflows/verify.yml`'s
  "Check template structure" step as the scaffold-side caller (not
  `bootstrap.sh`, which the self-review's MEDIUM finding caught and Slice B
  fixed); the code-span pipe `` `\| while read` `` is escaped (`grep -o`
  count: 7 total `|`, 1 escaped, leaving 6 unescaped column separators →
  5 cells, matching the table's 5-column header); the Related-plan/report
  cell cites "Issue #183, issue #189" rather than a `docs/plans/active/...`
  path that would break once this plan archives. No other row in the file
  still describes the `required_files` coverage gap as open (grepped the
  whole file for `check-template` and `189`).
- `scripts/check-template.sh` header/comment (lines 1–40, the
  `# --- Required files ---` block and its `required_files="..."`
  assignment): no count claim (e.g. "N required files") appears anywhere in
  the comment for the new-code-to-check against — there is nothing to drift.
  Confirmed 28 entries present and root/template copies byte-identical
  (`cmp`, matching the verify report's own check).
- `docs/quality/quality-gates.md` (lines 49–50) and
  `docs/architecture/repo-map.md` (lines 61–62): grepped for
  `check-template`, `required_files`, `TestTemplateBase`. Both only name
  `check-template.sh` generically as a CI check / script inventory entry
  (quality-gates.md: "which CI job runs it"; repo-map.md: "which directory
  it lives in, alongside other CI/drift-check scripts"). Neither makes any
  claim about the required-files list's contents, count, or test coverage,
  so neither is affected by this diff.
- `docs/quality/definition-of-done.md`, `AGENTS.md`, `README.md`,
  `docs/recipes/`: grepped for `check-template`, `required_files`,
  `TestTemplateBase`. No hits in any of the four locations — nothing there
  describes this script's contract.
- `tests/test-check-template.sh:1–49` header comment and
  `internal/scaffold/embed_test.go` comment above `requiredTemplateScripts`
  (Slice B's self-review-fix target): both name all four sites that must be
  updated together (`scripts/check-template.sh`,
  `templates/base/scripts/check-template.sh`, `requiredTemplateScripts`,
  `GOLDEN_ENTRIES`), state the ordering match is a "readability convention
  only" (not enforced), name the Go test
  (`TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles`) explicitly, and
  cite "issue #183" rather than a plan path. Matches the verify report's own
  confirmation of these same self-review fixes.
- Plan's Deviation notes and 調査で確認したこと section, cross-checked
  against current file state (not edited — out of this cycle's scope per
  the task's constraints): the 28-entry count (6 non-script + 22
  `scripts/`), the byte-identical copies, and the #189 hand-off are all
  consistent with what's actually in the tree today.
- Gates: `./scripts/check-skill-sync.sh` (13 skills in lock-step),
  `./scripts/check-sync.sh` (159 IDENTICAL / 0 DRIFTED / 11 TEMPLATE_ONLY /
  5 KNOWN_DIFF, PASS), `./scripts/check-template-purity.sh` (PASS, no
  meta-repo-specific references in templates). All pass; expected to be
  unaffected since this diff touches only `tests/test-check-template.sh`,
  `internal/scaffold/embed_test.go`, both `check-template.sh` copies
  (already byte-identical), and `docs/tech-debt/README.md`, none of which
  these three gates compare against each other in a way this diff could
  break.

## Stale

None found. Every doc location checked already matches the shipped
`fbcfb38`/`ce51738` behavior, including the two self-review MEDIUM fixes
(caller name, pipe escaping) that were the only doc-accuracy findings this
plan's pipeline surfaced.

## Changed

None. No edits made in this cycle.

## Left

None outstanding from this cycle's scope. Pre-existing, out-of-scope
observations noted for context only (not acted on, per the plan's
Non-goals and this task's constraints):

- The plan's Open questions section (which 6 scripts to add to either list
  next) and #189 (hook-reference false-positive/fail-open, meta-repo-only
  required-file entries) remain open by design — both are explicitly
  out of #183's scope and already tracked (the latter by issue number in
  the tech-debt row itself).

## Verdict

No drift found. No files changed in this cycle.
