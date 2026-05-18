# Test: base-language-rules

- Verdict: PASS
- Related issue: #104
- Evidence: `docs/evidence/verify-2026-05-18-090944.log`

## Commands

- `./scripts/run-test.sh` PASS
- `tests/test-terraform-rule-frontmatter.sh` PASS
- `go test ./...` PASS through `./scripts/run-test.sh`

## Regression Coverage

- `TestExecuteInit_NewProject` verifies selected packs create
  `.claude/rules/<pack>.md`, do not emit `packs/languages/<pack>/rule.md`, and
  do not create unselected pack rules.
- `TestRunUpgrade_DoesNotAddUnselectedPackRules` covers empty-pack upgrade.
- `TestRunUpgrade_MigratesInstalledPackRuleFromBaseManifest` covers the legacy
  base-managed rule migration path for installed packs.
- `TestRunUpgrade_DropsLegacyUninstalledBaseRule` covers the uninstalled legacy
  base-rule removal notice and manifest drop.
- `TestRenderFS_SkipsConfiguredPaths` covers renderer skip-path behavior.
