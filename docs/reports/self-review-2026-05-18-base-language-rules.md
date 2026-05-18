# Self Review: base-language-rules

- Verdict: PASS
- Scope: diff quality, maintainability, unnecessary changes, obvious correctness risks
- Related issue: #104

## Findings

No blocking or critical findings.

## Reviewed Areas

- Pack rendering now treats `rule.md` as a control file and writes it to
  `.claude/rules/<pack>.md`, while ordinary pack files remain under
  `packs/languages/<pack>/`.
- Upgrade diffing now supports skipped template paths and remapped single-file
  destinations without changing the existing per-file conflict semantics.
- Legacy base-managed language rules are split out of base removal detection
  only when the corresponding pack is installed, giving installed packs a
  migration path and uninstalled packs a removal notice.
- Sync and Terraform rule tests were updated to match the new pack-rule source
  of truth.

## Residual Risk

- Users with uninstalled legacy language rules still receive the standard
  "removed from template" notice and must delete those local files manually.
  This matches the existing removal behavior and avoids deleting user files
  automatically.
