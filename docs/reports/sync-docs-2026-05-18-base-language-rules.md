# Sync Docs: base-language-rules

- Verdict: PASS
- Related issue: #104

## Updated Documentation

- `docs/recipes/adding-a-language-pack.md`
- `templates/base/docs/recipes/adding-a-language-pack.md`

## Checks

- `./scripts/check-sync.sh` PASS
- `./scripts/check-skill-sync.sh` PASS

## Notes

The docs now describe language pack `rule.md` as the source rendered to
`.claude/rules/<lang>.md` only when the pack is selected. The old instruction to
mirror language rules into `templates/base/.claude/rules/` was removed.
