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

## Cycle 2 (pipeline cycle 2/2)

- Date: 2026-08-17
- Scope: `5283ee0..d21273b` — cross-review triage (`6e915f5`), cross-review AR fixes (`f80e60f`), cycle-2 self-review (`98d184f`), cycle-2 self-review fixes (`bd2d867`), cycle-2 verify/test artifacts (`54f97ac`, `d21273b`).
- Checked for residual drift:
  1. **C2-2 Codex-qualification wording** — `bd2d867` added the "Claude Code today; Codex's `.codex/config.toml` still calls hook scripts directly — Phase 3 tech debt" qualification to `AGENTS.md:106`, `docs/architecture/repo-map.md:20`, and `README.md:229`. Compared all three: `AGENTS.md` and `repo-map.md` use identical wording; `README.md`'s Hooks section uses a slightly longer variant ("...so `.ralph/local/hooks/<event>.d/` drop-ins do not run under Codex yet — Phase 3 tech debt") that says the same thing in prose-appropriate form for that section. No inconsistency — different sentence shapes for a repo-map bullet vs. a prose paragraph, same claim. No changes needed.
  2. **Tech-debt README rows** — verified the `.gitignore` row now reads `~~...~~ (RESOLVED 2026-08-17 in c2d501e)` with strikethrough on both the finding and its old owner note, and the new batched row (dispatcher `awk` duplication + `check-sync.sh` `ROOT_ONLY` label/counter) is present with full 6-column shape. Row count and pipe-count per data row confirmed consistent (`awk` column-count check: 6 fields per row across all data rows). No changes needed.
  3. **Stale references from the delta** — grepped for old symbol/path names touched by `f80e60f`/`bd2d867` (`ownerForScaffoldPath`, `OwnerCore`, `packRuleRelPath`, `filepath.Join` in dispatcher context) across `docs/`, `AGENTS.md`, `README.md`, `docs/architecture/repo-map.md`: no hits outside unrelated archived-plan history. No stale references introduced.
  4. **Spec FR-5 `#`-marker sentence** — re-read `docs/specs/2026-08-17-overlay-scaffold-v2.md` FR-5 against current `.gitignore` / `templates/base/.gitignore`: both still use `# BEGIN RALPH MANAGED (ralph:gitignore)` / `# END RALPH MANAGED` exactly as FR-5 describes. The C2-1 dispatcher signal-handling fix and the C2-3 free-area line addition (`# Project-specific ignores go below this line.`, appended *after* `# END RALPH MANAGED`) don't touch the marker format. Confirmed accurate, no change.
- Verification re-run: `./scripts/check-sync.sh` → `PASS: all files in sync.` (IDENTICAL 158, DRIFTED 0, KNOWN_DIFF 3); `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`.
- **Verdict: no drift.** No files changed in this cycle.

## Cycle 3 (pipeline cycle 3/3, final)

- Date: 2026-08-17
- Scope: `0be339a..HEAD` (HEAD `bb22a24`) — cross-review triage cycle 2 (`d5b478e`), cycle-2 AR fixes (`c289875`: `doctor` command-token parsing for dispatcher-wrapped hook commands, `init` symlinked-block-surface refusal), cycle-3 self-review (`3580c23`), cycle-3 self-review fixes (`2b1a855`: `RenderFS` `Stat`→`Lstat` dangling-symlink guard, tech-debt row for the two batched cycle-3 LOW findings), cycle-3 verify/test artifacts (`a7c857b`, `bb22a24`).
- Checked for residual drift (per team-lead handoff):
  1. **RenderFS existence semantics (`Stat`→`Lstat`)** — grepped `README.md`, `AGENTS.md`, and `docs/architecture/repo-map.md` for "existing files are skipped"-style claims and for any `symlink` mention. No hits. Neither doc describes the on-disk existence check at the `Stat`-vs-`Lstat` level of detail; both only state the higher-level contract (non-`--force` runs skip files that already exist), which the `Lstat` change does not invalidate — it only closes a dangling-symlink edge case within that same "already exists → skip" behavior. No doc references the symlink nuance at all, so nothing needed updating to match it. No changes needed.
  2. **Tech-debt README rows** — `2b1a855` appended a 5th plan-scoped row (cycle-3 batched LOW findings: mis-stamped `cycle` values in the insights event log, a `t.Fatalf` precondition-style message that reads as a broken fixture, and a stale triage-report cycle count). Verified column shape: all 5 plan-scoped rows (lines 96-105) are 6-field/5-column pipe rows, consistent with the table's header. No contradictions between rows — each documents a distinct, non-overlapping finding set. No changes needed.
  3. **Doctor docs (README/recipes) — hooks integrity** — grepped `README.md`, `docs/recipes/codex-setup.md`, and `AGENTS.md` for `doctor`. All three references stay at the generic capability level ("Check Claude Code, Codex, hooks, manifest drift, language packs"; "confirms Claude Code/Codex presence"; "warns if the project is unwritten / untrusted"). None describe `checkHooks`'s internal command-token-splitting logic (`internal/cli/doctor.go`, fixed in `c289875` to `strings.Fields(cmd)[0]` instead of stat-ing the whole dispatcher-wrapped command string), so the fix does not invalidate any doc claim. Still accurate. No changes needed.
  4. **Insights event log** — noted but out of scope: `docs/tech-debt/README.md`'s new cycle-3 row (item 2 above) already tracks the two mis-stamped `cycle` values in `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` as a deferred finding with its own trigger ("next change to this file"). Appending this cycle's own `sync_docs` event (below) is a new, correctly-stamped entry, not a correction of the prior ones — leaving the historical correction to whoever next actions that tech-debt row keeps this step's scope to sync-docs (doc/report accuracy), not test-data remediation.
- Insight event appended: `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` — `phase: sync_docs, cycle: 3, verdict: complete`.
- Verification re-run: `./scripts/check-sync.sh` → `PASS: all files in sync.` (IDENTICAL 158, DRIFTED 0, KNOWN_DIFF 3); `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`.
- **Verdict: no drift.** No doc files changed in this cycle beyond this report and the insight event.

## Cycle 4 (pipeline cycle 4/4, final)

- Date: 2026-08-18
- Scope: `c5771af..HEAD` (HEAD `6568a88`) — cross-review triage cycle 3 (`756aac8`), cycle-3 ACTION_REQUIRED fixes (`28842a0`: dispatcher `kill_child` on signal, `renderMappedFile` `Stat`→`Lstat` dangling-symlink guard, insight-log cycle-stamp corrections), a one-line insight-stamp follow-up fix (`88ba334`), cycle-4 self-review (`ef9e8e2`), a tech-debt register edit (`1e03cd5`), cycle-4 verify/test artifacts (`34120b3`, `6568a88` — the latter also ticks all 11 AC checkboxes in the plan).
- Checked for residual drift (per team-lead handoff):
  1. **Dispatcher signal handling / pack-rule rendering vs. docs** — grepped `README.md`, `AGENTS.md`, and `docs/architecture/repo-map.md` for `signal`, `TERM`/`SIGTERM`, `kill_child`, `child process`, `packRule`, `renderMappedFile`, and "pack rule": no hits describing dispatcher child-process signal semantics or `renderMappedFile`'s existence-check mechanics. All three docs stay at the higher contract level ("the hooks dispatcher fans out to `<event>.d/`", "language pack rules render at `.claude/rules/ralph/<lang>.md`") that `28842a0`'s two fixes do not invalidate — same pattern the cycle-3 sync-docs already confirmed for the sibling `RenderFS` `Lstat` fix. No changes needed.
  2. **Tech-debt README rows after `1e03cd5`'s strikethroughs** — read the row `1e03cd5` edited: it strikes clauses (a)/(b) of the cycle-3 batched deferral (the mis-stamped insight-log cycle values, now resolved by `28842a0`/`88ba334`; the stale triage-report pointer/header, superseded by `756aac8`'s full triage rewrite) and leaves the one still-open clause (the `cli_test.go` `t.Fatalf("precondition failed: ...")` wording) live, with the file pointer switched from a decayed line number to the test name `TestAddPack_LegacyManifest_StaysOwnerless`. Verified against the tree: `grep -n 'precondition failed: addPack' internal/cli/cli_test.go` still finds it (now at line 2481, confirming the row's line-number pointer was correctly dropped rather than just moved), and `docs/reports/cross-review-triage-overlay-scaffold-v2-p2.md` no longer has an `init.go` reference (confirming clause (b) is genuinely resolved, not just asserted resolved). Row is internally consistent with the current tree. No further changes needed.
  3. **Plan checkboxes** — `6568a88` ticked all 11 AC checkboxes (`AC-1`..`AC-11`) to `[x]`, matching `/test`'s cycle-4 verdict (599/599 shell + 8/8 Go, all ACs re-verified across cycles). Progress checklist: `Plan reviewed` / `Branch created` / `Implementation started` / `Review artifact created` / `Verification artifact created` / `Test artifact created` / `Sync-docs artifact created` are all `[x]`; only `PR created` remains `[ ]`, which is correct — `/pr` has not run yet. Coherent.
  4. **Insight event log** — cross-checked `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` against the pipeline history: cycle 4 had `verify` (16:06:24Z) and `test` (16:11:36Z) events but no `self_review` event, unlike cycles 1–3 which each carry one (the cycle-2 gap was itself caught and backfilled by `28842a0`). This is the same drift class as the cycle-3 finding C3-4 that `28842a0` already fixed once. Appended the missing `self_review` `cycle:4` event in timestamp order (`16:01:30Z`, matching `ef9e8e2`'s commit time, ahead of the `16:06:24Z` verify event) with `medium:1, low:0` findings, matching the self-review report's Cycle 4 section (1 MEDIUM — C4-1 — no CRITICAL/HIGH/LOW). Also appended this cycle's own `sync_docs` event.
- Insight events appended: `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` — backfilled `phase: self_review, cycle: 4, verdict: pass, medium: 1` (missing entry, item 4 above); `phase: sync_docs, cycle: 4, verdict: complete` (this cycle).
- Verification re-run: `./scripts/check-sync.sh` → `PASS: all files in sync.` (IDENTICAL 158, DRIFTED 0, KNOWN_DIFF 3); `./scripts/check-skill-sync.sh` → `[ok] check-skill-sync: 13 skill(s) in lock-step`.
- **Verdict: no drift** in README/AGENTS.md/repo-map/tech-debt register (items 1–3); one process gap found and fixed directly (item 4, missing insight event — a records-completeness gap, not a documentation-drift gap). No `.md` doc files changed in this cycle beyond this report; the insight-events JSONL and this report are the only files touched.
