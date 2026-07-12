# Sync-docs report — repo-wide drift sweep (post #124–#131 series)

- Date: 2026-07-13
- Scope: user-invoked `/sync-docs` full-repo audit after the six-PR series
  (#124–#129), the develop→main integration (#130), and the audit-fix PR
  (#131). Two parallel read-only audits (product-level docs, harness-internal
  surfaces) followed by a single fix pass.

## Drift found and fixed (10 items)

| # | File | Drift | Severity | Fix |
|---|------|-------|----------|-----|
| 1 | `AGENTS.md` (internal/cli bullet) | subcommand list missing `insights` | HIGH | added |
| 2 | `AGENTS.md` (repo map) | `internal/insights/` package missing | HIGH | bullet added |
| 3 | `AGENTS.md` (.agents/skills bullet) | no mention of the `sync-skills.sh` generator (CLAUDE.md already had it) | MEDIUM | reworded to generator + drift gate |
| 4 | `AGENTS.md` (scripts bullet) | `gc-artifacts.sh` / `sync-skills.sh` / `insights-append.sh` unlisted | MEDIUM | added |
| 5 | `README.md` (source tree) | `internal/insights/` missing | MEDIUM | added |
| 6 | `docs/quality/quality-gates.md` (CI list) | `check-skill-sync.sh` now CI-enforced but unlisted | MEDIUM | added |
| 7 | `docs/tech-debt/README.md` (Phase 6b) | `PR #TODO` placeholder | LOW | → `PR #128` |
| 8 | `docs/tech-debt/README.md` (shell CLI retirement) | referenced the dual-cli plan under `docs/plans/active/` (archived) | MEDIUM | → archive path |
| 9 | `docs/recipes/ralph-loop.md` (RALPH_XREVIEW_BASE) | cited the abolished `develop` branch as the example | LOW | generic non-default-branch wording |
| 10 | `docs/architecture/repo-map.md` (scripts) | `ralph-common.sh` + the three new scripts unlisted | MEDIUM | added |

Template mirrors updated for the three files with twins: `AGENTS.md`,
`docs/quality/quality-gates.md`, `docs/recipes/ralph-loop.md`
(root `README.md` and `docs/architecture/repo-map.md` have no twins).

## Verified clean (no drift)

- README command table vs `internal/cli` registrations; Quick Start; Ralph
  Loop flags; safety rails; no develop references
- CLAUDE.md skill lists and directory notes; all 13 skill frontmatters
- `post-implementation-pipeline.md` 8-file reference list (check-pipeline-sync
  green); work/SKILL.md Step 13; pr/SKILL.md pre-checks vs report naming
- `model-routing.md` per-phase vars vs `ralph-config.sh`; receipts paragraph
  vs `ralph-cli-driver.sh`; `defaults_sync_test.go` tripwire green
- `docs/insights/README.md` schema vs appender flags vs `event.go` json tags
- hooks ↔ `.claude/settings.json` ↔ `.codex/config.toml` (byte-identical) ↔
  templates twin; KNOWN_DIFFS still exactly 3 and justified
- language packs (6) with real verifiers, rules, detection, template parity
- insight-event skill snippets vs appender enums; `.agents` mirrors in
  lock-step (13 skills)
- recipes: worktrees, codex-setup; quality definition-of-done

## Gates

- `check-sync.sh` PASS / `check-skill-sync.sh` 13 in lock-step /
  `check-pipeline-sync.sh` ok / `check-template.sh` ok /
  `run-verify.sh` all verifiers passed (see PR checks)
