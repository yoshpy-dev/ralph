# Walkthrough: base-language-rules

- Related issue: #104
- Branch: `fix/104-base-language-rules`

## What Changed

1. Language pack `rule.md` is now a control file. It is skipped during normal
   pack rendering and installed as `.claude/rules/<pack>.md` only for selected
   packs.
2. Upgrade diffing accepts skip paths and a remapped single-file diff, so pack
   rules can use the existing conflict, baseline, and manifest machinery.
3. Legacy base-managed language rules migrate deterministically:
   selected packs adopt the pack rule, while unselected packs get the existing
   removed-from-template notice.
4. Language-specific rule files were removed from `templates/base/.claude/rules/`
   and moved into pack sources under `packs/languages/<lang>/rule.md` and
   `templates/packs/<lang>/rule.md`.
5. Sync checks and language-pack authoring docs were updated so root dogfood
   rules mirror pack rule sources instead of the base template.

## Reviewer Focus

- `internal/cli/language_pack.go`: pack rule destination mapping and boundary
  check.
- `internal/cli/upgrade.go`: legacy manifest split and pack rule diff handling.
- `internal/upgrade/diff.go`: refactor preserving existing file-diff behavior
  while adding skip-path options.
- `internal/cli/cli_test.go`: no-pack, selected-pack, and legacy migration
  regression coverage.
