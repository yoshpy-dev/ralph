# Sync-docs report: ralph default → claude-opus-4-7 / xhigh

- Date: 2026-04-21
- Branch: `chore/ralph-default-opus-4-7` @ `8fbe203`
- Scope: documentation drift sweep after ralph defaults change (model `claude-sonnet-4-20250514` → `claude-opus-4-7`; shell-side `opus` / `high` and Go `high` → `xhigh`)
- Verdict: **No drift found.** All documentation referencing the changed defaults was already updated in the two in-branch commits (`4f5d5b5`, `8fbe203`).

## Files scanned for drift

Top-level docs that could reasonably mention defaults or model IDs:

- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.claude/rules/` (all files)
- `.claude/skills/` (all files, prompts included)
- `docs/quality/definition-of-done.md`
- `docs/quality/` (full tree)
- `docs/tech-debt/README.md`
- `docs/plans/templates/` (all files)
- `docs/plans/active/` (empty)
- `docs/specs/2026-04-16-ralph-cli-tool.md`
- `docs/recipes/ralph-loop.md`
- `templates/base/README.md` (does not exist)
- `templates/base/AGENTS.md`
- `templates/base/CLAUDE.md`
- `templates/base/ralph.toml`
- `templates/base/scripts/ralph-config.sh`
- `templates/base/scripts/ralph-pipeline.sh`
- `templates/base/scripts/ralph-loop.sh`
- `templates/base/.claude/skills/loop/SKILL.md`
- `templates/base/docs/recipes/ralph-loop.md`

Patterns searched:
- `claude-sonnet-4` / `claude-sonnet-4-20250514`
- `claude-opus-4-7`
- `RALPH_MODEL` / `RALPH_EFFORT`
- `xhigh`, `=high`, `"high"`, `'high'`, bare `opus` / `sonnet` / `effort` / `model`
- `--model` / `--effort` in skills/prompts

## Files updated

None. All drift was fixed in-branch by commits `4f5d5b5` and `8fbe203`:

| File | Committed update |
|------|------------------|
| `internal/config/config.go` | Go `Default()` now returns `claude-opus-4-7` / `xhigh` (`4f5d5b5`) |
| `internal/config/config_test.go` | Asserts new defaults (`4f5d5b5`) |
| `templates/base/ralph.toml` | Scaffold TOML matches Go defaults (`4f5d5b5`) |
| `docs/specs/2026-04-16-ralph-cli-tool.md` | Spec example TOML (`4f5d5b5`) |
| `scripts/ralph-config.sh` + `templates/base/` mirror | Shell fallback pins (`8fbe203`) |
| `tests/test-ralph-config.sh` | Shell test expectations (`8fbe203`) |
| `docs/recipes/ralph-loop.md` + `templates/base/` mirror | Defaults table in "Configuration via environment variables" section (`8fbe203`) |

No post-commit doc edits were required.

## Files intentionally not updated

| File / area | Reason |
|-------------|--------|
| `docs/reports/self-review-2026-04-21-ralph-default-opus-4-7.md` | Historical pipeline artifact — task instruction explicitly says do not touch. The report's mention of the pre-fix state (`RALPH_EFFORT:-high`) is a point-in-time snapshot of the self-review moment; the follow-up commit `8fbe203` landed the fix. This matches the project's recurring-blind-spots rule: self-review reports are not updated post-fix. |
| `docs/reports/verify-2026-04-21-ralph-default-opus-4-7.md` | Same — historical artifact. |
| `docs/reports/test-2026-04-21-ralph-default-opus-4-7.md` | Same — historical artifact. |
| `docs/evidence/verify-*.log`, `docs/evidence/test-*.log` | Immutable evidence; never rewritten. |
| `docs/tech-debt/README.md` | Self-review H-1 (Go/shell default drift) was fixed in-branch by `8fbe203`, not deferred. No tech-debt row to add. Existing rows (Go pipeline migration) are unrelated to this change. |
| `README.md` | The Ralph loop section mentions "model, effort, permission mode, iteration caps, timeouts" abstractly as env-var-configurable (line 250). It does not cite specific default values and correctly points to `scripts/ralph-config.sh` as source of truth, so it stays valid with the new defaults. |
| `AGENTS.md`, `CLAUDE.md` | Neither file references specific model IDs or effort levels. Only hit is `cross-model second opinion` (about Codex, unrelated). |
| `.claude/rules/*` | No references to specific model IDs or effort levels anywhere. |
| `.claude/skills/*` | No hard-coded defaults; the loop skill prompts use env-var expansion at runtime. |
| `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md` | Do not mention defaults or model IDs. |
| `docs/plans/templates/*` | Do not mention defaults or model IDs. |
| `templates/base/AGENTS.md`, `templates/base/CLAUDE.md` | Match upstream — no specific model IDs. |
| `templates/base/scripts/ralph-pipeline.sh`, `templates/base/scripts/ralph-loop.sh` | Consume `$RALPH_MODEL` / `$RALPH_EFFORT` from env; no hard-coded defaults. |
| `docs/plans/active/` | Empty (task was ad-hoc). No active plan progress to update. |

## Verification

No files edited → `./scripts/run-verify.sh` not re-run in this step. The task's preceding pipeline step produced fresh evidence at `docs/evidence/verify-2026-04-21-ralph-default-opus-4-7.log` (all stages PASS) which covers the committed state this sync-docs pass operated against.

## Summary

Documentation is aligned with the new defaults. No additional sync edits were necessary because the behavior change and its doc references landed together in the two commits.
