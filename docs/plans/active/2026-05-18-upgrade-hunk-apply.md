# Upgrade Hunk Apply

- Status: Draft
- Owner: Codex
- Date: 2026-05-18
- Related request: Create a PR for issue #99.
- Related issue: 99
- Type: feat
- Branch: feat/99/upgrade-hunk-apply

## Objective

Implement the remaining `ralph upgrade` conflict workflow from #99: when a
baseline is available, review conflicts at hunk level using 3-way
`old template / local / new template` data, stage choices in memory, show a
pre-apply summary, and write target files/manifest/baseline only after explicit
confirmation.

## Scope

- Add a line-based 3-way merge planner in `internal/upgrade` that produces
  hunk-level choices from baseline, local disk, and new template content.
- Replace the baseline-backed file-level prompt with hunk-level
  `apply template hunk / keep local hunk / edit / skip file`.
- Add `$VISUAL` / `$EDITOR` manual edit support for the current hunk.
- Stage hunk decisions in memory and add a pre-apply summary prompt before
  writes when hunk review was used.
- Update manifest v2 state for partial results (`state = "partial"`,
  `disk_hash`, `template_hash`, baseline metadata).
- Preserve v1 / baseline-missing fallback behavior.
- Update tests and docs for the new hunk-level behavior.

## Non-goals

- Forcing partial hunk UI for v1 or baseline-missing entries.
- Adding `next` or `quit`.
- Rewriting the low-level `UnifiedDiff` renderer globally.
- Building a custom inline terminal editor beyond `$VISUAL` / `$EDITOR`.
- Solving transactional safety for every non-conflict auto-update path beyond
  the hunk-review staging required by #99.

## Assumptions

- The existing baseline cache introduced by #98 is the source of old-template
  text for hunk planning.
- A conservative line-based planner is sufficient for scaffold files; semantic
  language-aware merging is out of scope.
- If the editor is missing or fails, the hunk is not resolved and the prompt
  reappears.
- Pre-apply summary confirmation is required when hunk review decisions exist;
  legacy fallback keeps its existing interaction model.

## Affected areas

- `internal/upgrade/` merge planning helpers and tests.
- `internal/cli/upgrade.go` conflict resolution, staging, editor integration,
  summary confirmation, and manifest writes.
- `internal/cli/cli_test.go` integration coverage.
- `internal/scaffold/manifest.go` helper for partial/resolved baseline entries.
- `docs/specs/2026-04-16-ralph-cli-tool.md`.
- `docs/tech-debt/README.md`.
- `docs/reports/` evidence reports.

## Design decisions

- Standard flow. The issue is multi-file but cohesive and the core algorithm can
  be implemented and tested in one branch.
- Use `$VISUAL` / `$EDITOR` for manual edit. This matches the issue escape hatch
  without introducing a new interactive editor surface.
- Keep pre-apply summary scoped to hunk-review decisions in this PR; broader
  transactionality for all upgrade actions remains covered by existing tech debt.
- Critical forks: None. The main UX choices were settled in #99.

## Acceptance criteria

- [ ] Baseline-missing / v1 conflicts still use legacy `overwrite / skip / diff`.
- [ ] Baseline-available conflicts expose hunk-level
      `apply template hunk / keep local hunk / edit / skip file`.
- [ ] `keep local hunk` preserves only the current hunk and keeps the file
      managed/partial rather than making the whole file unmanaged.
- [ ] `skip file` leaves the local file unchanged and marks the file unmanaged.
- [ ] Prompt output does not include `next` or `quit`.
- [ ] Hunk choices do not write the target file until the pre-apply summary is
      confirmed.
- [ ] `N` / EOF at the summary leaves target file, manifest, and baseline cache
      unchanged for hunk-reviewed files.
- [ ] `y` writes resolved target content and updates manifest/baseline metadata.
- [ ] Partial results record `state = "partial"` and `disk_hash`; all-template
      results record `state = "managed"`.
- [ ] Normal upgrade diff UI continues to omit hunk headers and hash summaries.
- [ ] Missing editor / editor failure is safe and non-destructive.

## Implementation outline

1. Add `internal/upgrade/merge.go` with hunk planner and resolver primitives.
2. Add focused planner unit tests for template-only, local-only,
   non-overlap, overlap, and identical changes.
3. Refactor CLI conflict resolution to return staged conflict results instead
   of immediate file writes for baseline-backed conflicts.
4. Add hunk prompt loop with `apply`, `keep`, `edit`, and `skip file`.
5. Add pre-apply summary and confirmation for hunk-reviewed changes.
6. Add manifest helper for resolved baseline entries and wire partial state.
7. Update integration tests for hunk decisions, summary cancel/apply, editor
   success/failure, and fallback behavior.
8. Sync specs, tech debt, and reports.

## Verify plan

- Static analysis checks: `gofmt`, `go test ./...`, `./scripts/run-verify.sh`.
- Spec compliance criteria to confirm: all acceptance criteria above.
- Documentation drift to check: upgrade flow spec, tech debt row for #99, PR
  body.
- Evidence to capture: targeted Go tests, full Go suite, full verify log.

## Test plan

- Unit tests: merge planner hunk grouping and resolved output rendering.
- Integration tests: hunk apply/keep mix, all apply, skip file, summary no/yes,
  editor success and missing editor.
- Regression tests: v1 fallback, no `next`/`quit`, compact diff UI without hunk
  headers/hash summaries, non-interactive safety.
- Edge cases: identical local/template hunk, overlapping edits, missing baseline
  file, editor failure.
- Evidence to capture: `go test ./internal/upgrade ./internal/cli` and
  `./scripts/run-verify.sh`.

## Risks and mitigations

- Risk: line-based planner mishandles insertions or overlapping changes.
  Mitigation: planner unit tests cover zero-length insertion spans and overlap
  cases.
- Risk: pre-apply summary changes legacy non-interactive behavior too broadly.
  Mitigation: gate summary confirmation on hunk-reviewed conflicts only.
- Risk: editor integration can corrupt target files on failure. Mitigation:
  edit temp files only and apply to memory until final confirmation.
- Risk: manifest v1 compatibility regresses. Mitigation: keep v1 fields and
  explicit fallback tests.

## Rollout or rollback notes

- Rollout is backward-compatible for baseline-missing projects; partial UI
  appears only when baseline metadata and cache are readable.
- Rollback is a commit revert. Existing partial manifests still retain v1
  `hash` / `managed` fields, so older readers fail safe into conflict/skip.

## Open questions

- Whether future work should make all upgrade actions fully transactional, not
  only hunk-reviewed conflict writes. Existing tech debt already tracks broader
  init/upgrade transactional safety.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
