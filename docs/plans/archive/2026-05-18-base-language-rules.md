# base-language-rules

- Status: In progress
- Owner: Codex
- Date: 2026-05-18
- Related request: Fix issue #104: base upgrade distributes language-specific rules without selected packs
- Related issue: #104
- Type: fix
- Branch: fix/104-base-language-rules

## Objective

Stop `ralph upgrade` from distributing language-specific `.claude/rules/<lang>.md`
files through the always-on base template. Language-specific rules should be
installed only for selected language packs, matching the existing
`packs/languages/<lang>/` opt-in boundary.

## Scope

- Move distributed language rule files out of `templates/base/.claude/rules/`.
- Teach init/upgrade pack rendering to install a pack's `rule.md` as
  `.claude/rules/<pack>.md` while keeping ordinary pack files under
  `packs/languages/<pack>/`.
- Track the emitted rule files in the manifest so future upgrades can update,
  skip, or conflict-resolve them like other managed files.
- Add regression coverage for no-pack and selected-pack upgrade behavior.
- Update docs/recipes that currently instruct pack authors to mirror language
  rules into the base template.

## Non-goals

- Changing interactive `ralph init` defaults. Accepting the all-pack default is
  a separate product/documentation decision.
- Changing language detection scripts or `run-verify.sh` dispatch behavior.
- Removing stack-agnostic shared rules from the base template.

## Assumptions

- `templates/packs/<lang>/rule.md` is already the intended source for
  language-specific rule prose.
- Pack `rule.md` should not be copied into `packs/languages/<lang>/`; it is a
  scaffold control file whose destination is `.claude/rules/<lang>.md`.
- Existing users who already have base-managed language rules should get a
  deterministic migration path, not repeated "new file" churn.

## Affected areas

- `internal/cli/init.go`
- `internal/cli/upgrade.go`
- `internal/scaffold/render.go` or adjacent render helpers
- `internal/cli/cli_test.go`
- `templates/base/.claude/rules/`
- `templates/packs/*/rule.md`
- `docs/recipes/adding-a-language-pack.md`
- `templates/base/docs/recipes/adding-a-language-pack.md`

## Design decisions

Critical forks: None. The issue body already identifies the desired boundary:
base remains stack-agnostic; language-specific rule content moves to the pack
path and is rendered to `.claude/rules/<lang>.md` only for installed packs.

## Acceptance criteria

- [ ] A project with `meta.packs = []` does not receive
      `.claude/rules/rust.md` or `.claude/rules/typescript.md` from
      `ralph upgrade`.
- [ ] A project with one selected pack receives that pack's
      `.claude/rules/<pack>.md` plus its `packs/languages/<pack>/...` files.
- [ ] Pack control file `rule.md` is not emitted into
      `packs/languages/<pack>/`.
- [ ] Existing base-managed language rules have a deterministic migration path
      when the corresponding pack is or is not installed.
- [ ] Tests cover empty-pack upgrade, selected-pack init/upgrade, and rule
      migration behavior.
- [ ] Documentation no longer instructs pack authors to mirror language rules
      into `templates/base/.claude/rules/`.

## Implementation outline

1. Add a pack rendering helper that treats `rule.md` as a special source file:
   hash/write it to `.claude/rules/<pack>.md`; render all other pack files under
   `packs/languages/<pack>/`.
2. Use that helper from init and upgrade, including baseline writes and manifest
   key mapping.
3. Remove distributed language rule files from `templates/base/.claude/rules/`
   and ensure each pack has the corresponding `templates/packs/<pack>/rule.md`.
4. Add tests around pack rule destination, empty-pack upgrades, and migration
   of old base-managed language rule entries.
5. Update language-pack authoring docs and run sync checks.

## Verify plan

- Static analysis checks: `gofmt`; `go vet` if available through project
  wrapper; `./scripts/check-sync.sh`; `./scripts/check-skill-sync.sh`.
- Spec compliance criteria to confirm: issue #104 acceptance criteria, manifest
  namespace boundaries, no base language rule files.
- Documentation drift to check: language pack recipe, README opt-in wording, repo
  map references.
- Evidence to capture: command output in `docs/reports/verify-...md`.

## Test plan

- Unit tests: helper behavior for pack render planning if factored separately.
- Integration tests: `internal/cli` init/upgrade tempdir tests.
- Regression tests: no-pack upgrade must not add `.claude/rules/rust.md` or
  `.claude/rules/typescript.md`; selected pack must add only its rule.
- Edge cases: legacy manifest tracks `.claude/rules/rust.md` as base but the
  current project has no rust pack; legacy manifest with rust pack installed.
- Evidence to capture: targeted `go test ./internal/cli ./internal/scaffold
  ./internal/upgrade` and full project wrapper if time permits.

## Risks and mitigations

- Risk: moving rule files can look like template deletion for existing users.
  Mitigation: add migration logic and regression tests for legacy manifest
  entries.
- Risk: pack `rule.md` accidentally lands in `packs/languages/<pack>/`.
  Mitigation: special-case it in the pack render path and assert absence.
- Risk: docs/check-sync expects root `.claude/rules/<lang>.md` mirrors.
  Mitigation: update the recipe and run sync checks; adjust tests only if the
  current sync contract encodes the old bug.

## Rollout or rollback notes

Rollout is a normal `ralph upgrade` behavior change. Rollback restores language
rule files to `templates/base/.claude/rules/` and removes the pack-rule
destination mapping.

## Open questions

None.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
