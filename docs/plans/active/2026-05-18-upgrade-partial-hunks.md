# Upgrade Partial Hunk Adoption

- Status: In Progress
- Owner: Codex
- Date: 2026-05-18
- Related request: Create a PR for issue #97.
- Related issue: #97
- Type: feat
- Branch: feat/upgrade-partial-hunks

## Objective

Add the first implementation slice for safer `ralph upgrade` conflict review:
manifest v2-compatible baseline metadata, baseline cache writes for accepted
template files, a dry-run diff preview, pager support for large diffs, and a
baseline-gated hunk review path that uses `apply / keep / edit / skip file`
without changing existing v1-project behavior destructively.

## Scope

- Extend manifest file metadata with `state`, `template_hash`, `disk_hash`,
  `baseline_status`, and `baseline_path` while keeping v1 `hash` and `managed`
  readable/writable for compatibility.
- Write baseline cache files under `.ralph/baseline/` when templates are
  initialized, auto-updated, force-overwritten, manually overwritten, added, or
  force re-adopted.
- Preserve existing behavior for entries whose baseline is missing.
- Add `ralph upgrade --dry-run --diff` to preview conflicts without writing
  target files or manifest changes.
- Add pager support for diff display with safe fallback to stdout.
- Add the baseline-gated hunk prompt shape only when baseline content is
  available: `[a]pply template hunk / [k]eep local hunk / [e]dit / [s]kip file`.
  Full manual edit and multi-hunk staged merge semantics are tracked as
  follow-up debt before #97 can close.

## Non-goals

- Full semantic 3-way merge for every possible overlapping edit shape.
- Enabling partial hunk resolution for v1 entries that lack baseline content.
- Adding `[n]ext` or `[q]uit`.
- Replacing the existing line-based diff algorithm globally.
- Removing v1 manifest fields.

## Assumptions

- Baseline cache can mirror template paths under `.ralph/baseline/` because
  template paths are already checked against target-root escape in scaffold
  rendering and diff walks.
- The first PR may leave deeper merge refinements as follow-up as long as
  baseline-missing entries remain on the current safe flow.
- `$EDITOR`/`VISUAL` may be absent; hunk edit mode can return a clear error
  without writing partial changes.

## Affected areas

- `internal/scaffold/manifest.go`
- `internal/cli/init.go`
- `internal/cli/upgrade.go`
- `internal/upgrade/`
- `internal/cli/cli_test.go`
- `internal/scaffold/manifest_test.go`
- `docs/specs/2026-04-16-ralph-cli-tool.md`
- `docs/tech-debt/README.md` if follow-up scope remains

## Design decisions

- Standard flow chosen by default because the user requested a PR directly,
  and the first implementation slice is bounded enough for one branch.
- `baseline_status` is the manifest field name, per the issue discussion.
- Baseline-missing entries stay on the existing overwrite/skip/diff path; this
  is the central non-destructive migration rule.
- Hunk decisions are staged in memory and written only after final file
  confirmation.
- Critical forks: None remaining. The high-risk product choices were already
  settled in issue #97.

## Acceptance criteria

- [x] Existing v1 manifests still read successfully and do not force partial
      hunk UI when baseline content is missing.
- [x] Newly initialized and accepted template files record
      `baseline_status = "available"` and `baseline_path`.
- [x] `Managed=false` entries remain unmanaged across v2 metadata writes and
      baseline creation.
- [x] `ralph upgrade --dry-run --diff` previews changes without writing target
      files or `.ralph/manifest.toml`.
- [x] Conflict diff display can use a pager when requested and falls back to
      stdout if the pager cannot start.
- [x] Baseline-available conflicts expose only
      `apply / keep / edit / skip file`; no `next` or `quit` prompt text is
      introduced.
- [x] Full hunk edit/final-summary semantics are explicitly tracked as
      follow-up debt instead of silently implied by this first slice.
- [x] Tests cover manifest compatibility, baseline writing, dry-run behavior,
      and hunk prompt option shape.

## Implementation outline

1. Extend `ManifestFile` with v2 fields and helper methods that preserve v1
   compatibility.
2. Add baseline cache helpers for safe relative path validation, write, and
   read.
3. Wire baseline writes through init and all upgrade paths that accept template
   content.
4. Add upgrade options: `--dry-run`, `--diff`, and `--pager`.
5. Render dry-run summaries/diffs from computed `FileDiff` values without
   mutating disk.
6. Introduce a baseline-gated hunk prompt path while deferring the full merge
   planner/editor to tech debt.
7. Add focused tests and update docs/spec text.

## Verify plan

- Static analysis checks: `gofmt`, `go vet`, `go build ./...`, repo verify
  script.
- Spec compliance criteria to confirm: every acceptance criterion above.
- Documentation drift to check: upgrade flow spec, tech debt notes, issue #97
  wording.
- Evidence to capture: command output for targeted Go tests and full verify.

## Test plan

- Unit tests: manifest v1/v2 round trip; baseline path validation; hunk
  decision rendering.
- Integration tests: init baseline writes; upgrade accepted paths update
  baseline metadata; dry-run diff does not mutate files/manifest.
- Regression tests: unmanaged skip remains unmanaged; non-interactive conflict
  still safe-skips when baseline is missing.
- Edge cases: missing baseline file despite manifest metadata; pager command
  failure; baseline path escaping attempt.
- Evidence to capture: `go test ./internal/scaffold ./internal/upgrade ./internal/cli`.

## Risks and mitigations

- Risk: manifest v2 fields change behavior for existing repos. Mitigation:
  keep `hash`/`managed`, gate partial UI on baseline availability, add v1 tests.
- Risk: baseline cache path escapes `.ralph/baseline`. Mitigation: validate
  cleaned relative paths before write/read.
- Risk: hunk resolver writes partial files before all choices are known.
  Mitigation: stage resolved content in memory and write only after final
  confirmation.
- Risk: pager invocation blocks or fails. Mitigation: only use explicit or TTY
  mode and fall back to stdout on error.

## Rollout or rollback notes

- Rollout is backward-compatible: v1 manifests are still accepted and baseline
  missing entries keep current behavior.
- Rollback is a commit revert. Existing projects may have extra
  `.ralph/baseline/` files and v2 metadata, but v1 fields remain present.

## Open questions

- Whether future PRs should compress or hash baseline cache paths for very
  long filenames.
- Whether hunk edit mode should require `$EDITOR` or support inline editing as
  a fallback.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
