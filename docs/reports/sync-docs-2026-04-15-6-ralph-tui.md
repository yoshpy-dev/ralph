# Sync-docs report: ralph-tui (slice-6)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md (slice-6)
- Agent: pipeline-sync-docs (autonomous)
- Scope: documentation sync after implementation

## Product-level sync

| Document | Updated? | Details |
| --- | --- | --- |
| Active plan progress | no | Manifest progress checklist is for orchestrator to mark after all slices merge |
| README.md | **yes** | Added TUI description, `--no-tui`/`--json` flags, `retry`/`abort --slice` subcommands, `build-tui.sh` to Ralph Loop section; added `cmd/` and `internal/` to directory tree |
| AGENTS.md | **yes** | Added `cmd/ralph-tui/`, `internal/state/`, `internal/watcher/`, `internal/ui/`, `internal/action/` to repo map; added `build-tui.sh` to scripts description |
| CLAUDE.md | no | No TUI mentions needed (confirmed by plan: スコープ外) |
| `.claude/rules/` | no | No rules affected by TUI changes |
| `docs/quality/` | no | No quality gate changes needed |

## Harness-internal sync (7 categories)

| Category | Status | Notes |
| --- | --- | --- |
| Skills added/removed/renamed | in sync | No skills changed |
| Hooks added/removed | in sync | No hooks changed |
| Rules added/removed | in sync | No rules changed |
| Language packs | in sync | No packs changed |
| Scripts added/removed/renamed | **updated** | `build-tui.sh` added to AGENTS.md repo map and README.md |
| Quality gates | in sync | No gate behavior changes |
| PR skill | in sync | No PR workflow changes |

## Files updated

1. `README.md` — Ralph Loop section: added TUI launch description, new subcommands (`retry`, `abort --slice`), new flags (`--no-tui`, `--json`), `build-tui.sh` usage; directory tree: added `cmd/` and `internal/` packages
2. `AGENTS.md` — Repo map: added 5 Go package entries (`cmd/ralph-tui/`, `internal/state/`, `internal/watcher/`, `internal/ui/`, `internal/action/`); updated scripts description to include `build-tui.sh`

## Documentation drift detected

None. All documentation is now in sync with the implementation.

## Notes

- The plan explicitly states TUI mentions are not needed in CLAUDE.md (`スコープ外のため`). This is correct because the TUI is an optional enhancement that does not change Claude Code's operational behavior.
- `scripts/ralph` help text already includes `retry`, `abort --slice`, and `--no-tui` documentation — no additional updates needed there.
