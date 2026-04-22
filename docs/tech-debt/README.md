# Tech debt

Record debt that should not disappear into chat history.

Recommended fields:
- debt item
- impact
- why it was deferred
- trigger for paying it down
- related plan or report

## Entries

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Per-slice pipelines do NOT stop on CRITICAL self-review findings | Differs from standard `/work` flow behavior | Autonomous pipelines benefit from letting verify/test confirm true positives before halting | If false-negative CRITICAL findings slip through to merge | `.claude/rules/post-implementation-pipeline.md` |
| `ralph status --json` outputs plain text, not JSON | AC11 partially met; machine consumption broken | Phase 6a scope — table/JSON rendering deferred to Phase 6b Go native | Phase 6b implementation | `docs/plans/archive/2026-04-16-ralph-cli-tool.md` |
| `ralph init/upgrade` lack transactional safety (journal.toml, staging) | AC17/18/21 not met; interrupted init can leave inconsistent state | Development cost vs. shipping core CLI functionality first | Before v1.0 release or after first user report of interrupted init corruption | `docs/plans/archive/2026-04-16-ralph-cli-tool.md` |
| Phase 6b: Go native pipeline migration pending | Pipeline runs via shell wrapper (Phase 6a) | 3400 lines of shell → Go is a major effort requiring parity tests | Next PR; parity tests must pass before shell scripts are removed | `docs/plans/archive/2026-04-16-ralph-cli-tool.md` |
| `filePerm()` in upgrade.go duplicates permission logic from render.go with different strategy (name-based vs FS-metadata-based) | A new extensionless executable in templates needs updates in two places with different fix patterns | Scope of fix-template-distribution-gaps PR is distribution gaps, not refactoring permissions | Adding a second extensionless executable to templates | `docs/reports/self-review-2026-04-16-fix-template-distribution-gaps.md` |
| `.claude/hooks/check_mojibake.sh` + `mojibake-allowlist` + `tests/test-check-mojibake.sh` + `tests/fixtures/payloads/` + settings.json entry + AGENTS.md note + check-sync.sh exclusions | Every `Edit\|Write\|MultiEdit` in every session runs the scan; carrying a mitigation for a fixed upstream bug becomes dead weight | Temporary mitigation for upstream Claude Code Issue #43746 (U+FFFD injection at SSE chunk boundaries); permanent fix must land upstream | Upstream Issue #43746 closes in a released Claude Code version AND no local recurrences observed for 1 week | `docs/plans/archive/2026-04-17-mojibake-postedit-guard.md`, `docs/reports/self-review-mojibake-postedit-guard.md` |
| `ralph upgrade --resync <path>` / `--adopt` escape hatch for `Managed=false` entries | Once a user selects `skip` on a conflict, the entry is pinned unmanaged and silently skipped forever; there is no first-class way to re-adopt it back under ralph management short of hand-editing `.ralph/manifest.toml` or running `--force` on the whole tree | Out of scope for the "detect local edits + unified diff" PR — shipping the convergence contract first was the priority so users stop being prompt-stormed. The escape hatch needs UX design (per-path vs. glob, force-template-overwrite vs. keep-local-but-track) | First user report of being stuck on an unmanaged entry they want back under management, OR before v1.0 release | `docs/plans/active/2026-04-22-upgrade-detect-local-edits.md` (Open Questions), `docs/reports/self-review-2026-04-22-upgrade-detect-local-edits.md` |
