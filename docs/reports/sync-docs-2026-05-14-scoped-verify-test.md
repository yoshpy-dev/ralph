# Sync-docs report: scoped-verify-test

- Date: 2026-05-14
- Plan: `docs/plans/active/2026-05-14-scoped-verify-test.md`
- Scope: docs, rules, templates, and skill surfaces
- Verdict: pass

## Changes Checked

| Area | Status | Evidence |
| --- | --- | --- |
| Quality gates | updated | `docs/quality/quality-gates.md` documents changed local wrappers and full CI scope. |
| Definition of done | updated | `docs/quality/definition-of-done.md` documents changed-language `/verify` and `/test`, plus full Ralph Loop integration. |
| Pipeline rules | updated | `.claude/rules/post-implementation-pipeline.md` and `subagent-policy.md` describe changed local scope and full integration scope. |
| Skills and prompts | updated | Verify/test skills and loop prompts mention `RALPH_VERIFY_SCOPE`; `scripts/check-skill-sync.sh` passed. |
| Template mirrors | updated | `templates/base/**` mirrors root script/docs/skill changes; `scripts/check-sync.sh` passed. |
| Language-pack recipe | updated | `docs/recipes/adding-a-language-pack.md` now requires updating `detect-changed-languages.sh`. |

## Remaining Drift

- None found.
