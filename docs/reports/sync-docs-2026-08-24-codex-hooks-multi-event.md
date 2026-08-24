# Sync-docs report — codex-hooks-multi-event

- **Date:** 2026-08-24
- **Plan:** `docs/plans/active/2026-08-24-codex-hooks-multi-event.md`
- **Scope:** documentation drift from `git diff main...HEAD` — the four-event
  Codex hook wiring (`PostToolUse` + `PreToolUse[matcher Bash]` +
  `SessionStart` + `UserPromptSubmit`) shipped in `.codex/hooks.json` /
  `templates/base/.codex/hooks.json`.

## What changed

Fixed three stale "routes `PostToolUse` through `ralph-dispatch.sh`"
statements that pre-dated this diff and were left factually wrong by the new
event set:

| File | Change |
|------|--------|
| `AGENTS.md` (repo map, `.codex/` bullet) | "routes `PostToolUse`" → "routes `PostToolUse`, `PreToolUse`, `SessionStart`, and `UserPromptSubmit`" |
| `docs/architecture/repo-map.md` (`.codex/` bullet) | same wording fix |
| `README.md` (repo tree comment on `hooks.json`) | "hook wiring: routes PostToolUse through ralph-dispatch.sh" → adds the three new events |

All three sit **outside** the `AGENTS.md` managed block (`<!-- BEGIN RALPH
MANAGED -->` spans lines 11–68; the stale bullet was at line 108), so no
change to `.ralph/core/AGENTS.core.md` was needed — this is meta-repo-only
repo-map content, not part of what `ralph init` scaffolds. Confirmed
`templates/base/AGENTS.md` has no `.codex`/`PostToolUse` bullet to match, and
neither `docs/architecture/repo-map.md` nor `README.md` has a
`templates/base/` twin.

## What was already in sync (checked, no change)

- `.codex/README.md` and `templates/base/.codex/README.md` — already updated
  in-diff to describe all four events (`.codex/README.md:94`, `:110`, `:125`,
  `:128-129`).
- `.codex/hooks/README.md` and `templates/base/.codex/hooks/README.md` —
  already updated in-diff (line 33 lists the full wired event set).
- `.claude/rules/ralph/ralph-workflow.md` (lines 49–50) — says "Codex
  equivalents in `.codex/hooks.json`" generically, no event-specific claim;
  no drift.
- `docs/recipes/codex-setup.md` — mentions `.codex/hooks.json` only in the
  context of the `[features] hooks` toggle and hook-trust approval; no
  event-specific content to go stale.
- `AGENTS.md` primary-loop / pipeline-order text — no Codex-hook-event
  mentions.
- `docs/evidence/codex-hooks-livefire-*.md`, `docs/specs/*.md`,
  `docs/tech-debt/README.md` — historical/evidence records referencing the
  old single-`PostToolUse` state; intentionally left as-is (they document a
  prior state, not current behavior, and are not in the drift-check scope of
  this diff).

## Verification run

- `./scripts/check-sync.sh` — PASS (157 identical, 0 drifted, 5 known-diff, 11 template-only)
- `bash tests/test-hook-wiring.sh` — PASS: 66, FAIL: 0
- `./scripts/run-static-verify.sh` — all checks OK, including
  `scripts/check-sync.sh`, `scripts/check-pipeline-sync.sh`,
  `scripts/check-skill-sync.sh`, `scripts/check-template-purity.sh` (PASS: no
  meta-repo-specific references found in templates), and the Go verifier
  (`gofmt: ok`, `0 issues`)

## Files touched by this sync-docs pass

- `AGENTS.md`
- `README.md`
- `docs/architecture/repo-map.md`
- `docs/reports/sync-docs-2026-08-24-codex-hooks-multi-event.md` (this report)
