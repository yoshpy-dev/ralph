# Sync-docs report: overlay-scaffold-v2 Phase 2

- Date: 2026-08-17
- Plan: `docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Doc maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Scope: `git diff main...HEAD` on `feat/overlay-scaffold-v2-p2` (base `main`, HEAD `0132d9f`, 124+ files). Prior artifacts: self-review (fixed), verify (PASS), test (PASS, 593/593 shell + 8/8 Go packages).

## What was synced

1. **`README.md`** — the scaffold tree (`## What ralph init scaffolds`) and the `## Hooks` section still described the pre-v2 layout: no `.ralph/core/`/`.ralph/local/`, `.claude/rules/` without the `ralph/` subdirectory, `.claude/hooks/` described only as "deterministic runtime guardrails" with no mention of the dispatcher, `settings.json` described as holding hooks directly, and `CLAUDE.md` described as "Claude Code specific guidance" instead of the new minimal seed. Updated the scaffold tree to show `.ralph/core/` and `.ralph/local/` as top-level entries, `.claude/rules/ralph/`, and `.claude/hooks/` as "hook implementations + `<event>.d/` dispatch entries"; updated the `AGENTS.md`/`CLAUDE.md` tree-row comments to describe the managed-block/seed model; rewrote the `## Hooks` section to describe the `ralph-dispatch.sh <event>` fan-out (core → `.ralph/local/hooks/<event>.d/` → `.claude/hooks/local/<event>.d/`) and how to add a hook via drop-in instead of editing `settings.json`.
   - Everything else the team-lead handoff flagged for README (init/upgrade description, Quick start, Operating loop, Portability tables) was already accurate — those don't describe layout internals and weren't invalidated by this diff.

2. **`docs/architecture/repo-map.md`** — not explicitly named in the handoff but directly invalidated by the diff (same drift class as README's scaffold tree, just a different doc). `## Core files` described `CLAUDE.md` as "Claude-specific always-on additions" (now a minimal seed) and didn't mention `AGENTS.md`'s managed-block sourcing. `## Claude control plane`'s `.claude/rules/` bullet didn't mention the `ralph/` subdirectory, and `.claude/hooks/` didn't mention the dispatcher. There was no section for `.ralph/` at all. Updated the `Core files` and `Claude control plane` bullets to match, and added a new `## Overlay layout (.ralph/)` section (mirroring `AGENTS.md`'s own repo-map wording for `.ralph/core/` and `.ralph/local/`).

3. **`docs/specs/2026-08-17-overlay-scaffold-v2.md` FR-5** — per the handoff: FR-5 defined only the HTML-comment marker style; the plan's Deviations record adds the `#`-style marker for `.gitignore` (Phase 2, Slice 3). Appended one sentence to FR-5 stating that file types which don't tolerate HTML comments use a line-comment marker (`# BEGIN RALPH MANAGED (ralph:<surface>)` / `# END RALPH MANAGED`) with the same surface key, so the requirement now matches the implemented `.gitignore` behavior.

4. **`docs/tech-debt/README.md`** — added the row the team lead requested for the test gap `/test` flagged: `scripts/check-sync.sh`'s block-aware `AGENTS.md` `DRIFTED` branch (`extract_agents_md_block` + the `diff -q` comparison against `.ralph/core/AGENTS.core.md`) has no isolated fixture exercising the drift-detected path, only the happy path via the real repo tree. Trigger: next change to that comparison. Evidence: `docs/reports/test-2026-08-17-overlay-scaffold-v2-p2.md` (Test gaps), `scripts/check-sync.sh`, plan AC-7.

5. **Plan progress checklist** (`docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`) — added and checked "Sync-docs artifact created" (this report).

## Checked, no changes needed

- `docs/recipes/adding-a-language-pack.md`, `docs/recipes/worktrees.md` — already reference `.claude/rules/ralph/<lang>.md` and `.claude/rules/ralph/agent-messaging.md`; slice 6's sweep already updated these.
- `docs/recipes/codex-setup.md`, `docs/recipes/agent-teams.md` — no rule-path, hook-path, or CLAUDE.md-structure references invalidated by this diff.
- `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md` — already reference `.claude/rules/ralph/git-commit-strategy.md` and `.claude/rules/ralph/post-implementation-pipeline.md`; no other content invalidated by hooks/rules re-layout.
- `.ralph/core/AGENTS.core.md` (root) vs. `templates/base/.ralph/core/AGENTS.core.md` — confirmed byte-identical (`diff` clean); both reference `.claude/rules/` and `.claude/skills/` generically, which stays correct after the re-layout.
- AGENTS.md (root and template) — not touched by this step; slice 5 already migrated root's managed block and slice 6 already swept the repo-map's rule-path references. Re-confirmed `check-sync.sh`'s block-aware comparison reports 0 drift.
- `README.md` — init/upgrade CLI description, Quick start, Operating loop diagram, Commands table, Portability section, Known differences table: all accurate, none invalidated (this plan's Non-goals explicitly exclude `ralph upgrade` wiring changes and `.codex/` re-layout).
- No skills, hooks (as a *set* — the dispatcher is a re-plumbing of existing hooks, not an add/remove), or language packs were added/removed in this diff, so the harness-internal "Skills added/removed", "Language packs added/removed" checklist items don't apply. Hook *implementation location* did change (see README/repo-map edits above).
- `docs/tech-debt/README.md` already carries rows for the other drift items this plan surfaced (KNOWN_DIFFS resolution, `pre_bash_guard.sh` payload-parsing gap, pack-rule rename detection, Codex dispatcher parity gap, `.gitignore` block extent) from slices 5/6's self-review; only the `/test`-flagged AC-7 fixture gap was missing.

## Verification after doc edits

```
$ ./scripts/check-sync.sh
=== Sync Summary ===
  IDENTICAL:      158
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  10
  KNOWN_DIFF:     3
PASS: all files in sync.

$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
```

No template-side edits were required: `README.md`, `docs/architecture/repo-map.md`, `docs/specs/`, `docs/tech-debt/README.md`, and `docs/plans/active/` are all meta-repo-only paths (confirmed each has no `templates/base/` twin, or only a `.gitkeep` placeholder).

## Files changed

- `README.md`
- `docs/architecture/repo-map.md`
- `docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`
- `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- `docs/tech-debt/README.md`

## Known gaps

- None identified in this step beyond the tech-debt row recorded above, which is a test-coverage gap (already tracked), not a documentation gap.
