# codex-hook-single-source

- Status: Draft
- Owner: Codex
- Date: 2026-05-13
- Related request: codex-hook-single-source
- Related issue: 57
- Branch: issue-57-codex-hook-single-source

## Objective

Keep Codex project hooks represented in exactly one repo-local format:
`.codex/config.toml` inline hook entries. Detect any future reintroduction of
`.codex/hooks.json` beside inline hooks before it reaches users.

## Scope

- Root and template `.codex/README.md`
- Root and template `.codex/hooks/README.md`
- Root and template `.codex/config.toml` wording/shape if needed
- `scripts/verify.local.sh`
- Plan/review/verify/test/sync-docs artifacts

## Non-goals

- Changing Claude Code `.claude/settings.json` hook contracts.
- Moving unsupported Claude-only hook events into Codex.
- Using the existing dirty `.codex/hooks.json` as shipped output; it contains
  local absolute paths and contradicts the single-source objective.

## Assumptions

- Existing `[[hooks.PostToolUse]]` entries in `.codex/config.toml` count as
  the supported inline representation.
- `.codex/hooks/` remains reserved for future Codex-specific wrappers, not a
  duplicate script tree.

## Affected areas

- Codex project configuration docs
- Repo-local verification guard
- Root/template sync

## Design decisions

Critical forks: None. The issue already selects config TOML as source of truth.

## Acceptance criteria

- [ ] No tracked `.codex/hooks.json` is introduced.
- [ ] Hook docs describe `.codex/config.toml` inline hooks as the only shipped representation.
- [ ] `.codex/hooks/` is documented as README-only unless a wrapper is needed.
- [ ] Verification fails when `.codex/hooks.json` coexists with inline hooks.
- [ ] The inline hook detector handles valid TOML whitespace variants.
- [ ] Root `.codex/` and `templates/base/.codex/` stay synchronized.

## Implementation outline

1. Update Codex hook docs in root and template.
2. Add a TOML-aware duplicate hook representation guard to `scripts/verify.local.sh`.
3. Add focused smoke tests through shell snippets and run static verification.

## Verify plan

- Static analysis checks: `sh -n scripts/verify.local.sh`, TOML parse, `git diff --check`.
- Spec compliance criteria to confirm: each acceptance criterion above.
- Documentation drift to check: root/template `.codex` docs match.
- Evidence to capture: duplicate-representation smoke check and static verify logs.

## Test plan

- Unit tests: helper-level smoke tests via temporary TOML/hooks fixtures.
- Integration tests: `./scripts/run-static-verify.sh`.
- Regression tests: `scripts/check-sync.sh`.
- Edge cases: `[[ hooks.PostToolUse ]]`, `[ hooks.PostToolUse.match ]`, and no-hook config.
- Evidence to capture: verify/test reports with pass verdicts.

## Risks and mitigations

- Risk: false-positive duplicate detection. Mitigation: parse TOML with
  `tomllib` when available and keep a whitespace-tolerant fallback.
- Risk: Codex config docs diverge between root and template. Mitigation:
  update both and run sync check.

## Rollout or rollback notes

Rollback by reverting the verification/docs changes.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created: https://github.com/yoshpy-dev/ralph/pull/59
