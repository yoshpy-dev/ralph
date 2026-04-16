# Codex triage report: ralph-cli-tool

- Date: 2026-04-16
- Plan: docs/plans/archive/2026-04-16-ralph-cli-tool.md
- Base branch: main
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Total Codex findings: 5
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=2

## Triage context

- Active plan: 2026-04-16-ralph-cli-tool (Phase 1-8, 6a, 7 complete; Phase 6b/9 deferred)
- Self-review report: docs/reports/self-review-2026-04-16-ralph-cli-tool.md
- Verify report: docs/reports/verify-2026-04-16-ralph-cli-tool.md
- Implementation context: CLI tool migration from template repo. Pack files in `templates/packs/<lang>/` contain `README.md` and `verify.sh`. Existing repo expects packs at `packs/languages/<lang>/`. `scripts/run-verify.sh` references `packs/languages/<lang>/verify.sh`.

## ACTION_REQUIRED

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 3 | [P1] Pack FS renders into repo root, not `packs/languages/<lang>/` | Real issue. `PackFS` is rooted at `templates/packs/<lang>`, so `RenderFS` writes `README.md` and `verify.sh` to the project root. Existing scripts (`run-verify.sh`, `check-coverage.sh`, `doctor`) expect `packs/languages/<lang>/verify.sh`. Multiple packs overwrite each other. | `internal/cli/init.go:137-146`, `internal/cli/upgrade.go:69-78` |
| 4 | [P1] Upgrade diffs all packs, not just installed ones | Real issue. Upgrade iterates every embedded pack, so projects with a single pack get files from all packs. `ComputeDiffsNoRemovals` skips removal detection but still adds new files from uninstalled packs. Manifest should record which packs were selected, and upgrade should filter by that list. | `internal/cli/upgrade.go:65-78` |

## WORTH_CONSIDERING

| # | Codex finding | Triage rationale | Affected file(s) |
|---|---------------|------------------|-------------------|
| 1 | [P1] First-time init with `Overwrite: true` can destroy existing files | Partially real. Fresh init does set `Overwrite: true`, which can overwrite existing `AGENTS.md` or `.claude/settings.json`. The re-init guard (line 115-118) only triggers when `.ralph/manifest.toml` exists. First-time adoption of an existing project is a valid use case. However, `Overwrite: true` is intentional for scaffold setup — a fresh init is expected to establish the canonical layout. Could be improved with `Overwrite: false` and a `--force` flag. | `internal/cli/init.go:128-131` |

## DISMISSED

| # | Codex finding | Dismissal reason | Category |
|---|---------------|------------------|----------|
| 2 | [P1] Missing `scripts/` tree in scaffolded projects | The `scripts/` directory is part of this repo's infrastructure, not the scaffolded output. Scaffolded projects get `.claude/hooks/`, `.claude/skills/`, `docs/`, `AGENTS.md`, `CLAUDE.md`, and `ralph.toml`. The hooks and skills in the scaffold reference `./scripts/run-verify.sh` which is this repo's own script, not the target project's. Target projects use `ralph doctor` and `ralph run` (which delegate to the installed CLI), not direct `scripts/` access. The scaffold templates under `templates/base/` do not include a `scripts/` directory because the CLI binary replaces the need for them. | context-aware-safe |
| 5 | [P3] `status --json` outputs plain text | Already tracked in `docs/tech-debt/README.md` as Phase 6b scope. Explicitly listed as non-goal for this PR. | out-of-scope |
