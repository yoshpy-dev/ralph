# Verify: base-language-rules

- Verdict: PASS
- Related issue: #104
- Evidence: `docs/evidence/verify-2026-05-18-091042.log`

## Commands

- `./scripts/run-static-verify.sh` PASS
- `./scripts/check-sync.sh` PASS
- `./scripts/check-skill-sync.sh` PASS
- `./scripts/branch-name.sh validate "$(git branch --show-current)"` PASS

## Spec Compliance

- `templates/base/.claude/rules/` no longer distributes language-specific
  `dart`, `golang`, `python`, `rust`, `terraform`, or `typescript` rules.
- Pack `rule.md` sources are mirrored under both `packs/languages/<lang>/` and
  `templates/packs/<lang>/`, and sync checking compares root dogfood rules to
  those pack sources.
- Upgrade handling preserves selected-pack rule state and removes only the
  base-managed legacy path for unselected packs.

## Documentation Drift

- Updated the language-pack recipe in both root docs and `templates/base/docs/`
  to document `rule.md` as a pack control file rendered to
  `.claude/rules/<lang>.md` only when selected.
