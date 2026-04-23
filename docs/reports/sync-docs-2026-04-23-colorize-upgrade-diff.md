# Sync-docs report: colorize-upgrade-diff

- Date: 2026-04-23
- Plan: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md`
- Branch: `feat/colorize-upgrade-diff`
- Maintainer: `doc-maintainer` subagent (Claude Code)
- Upstream triggers:
  - Documentation drift flagged in `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/reports/verify-2026-04-23-colorize-upgrade-diff.md` §Documentation drift (LOW, non-blocking): stale diff sample at `docs/specs/2026-04-16-ralph-cli-tool.md:273-275` showing the pre-change `--- ralph template / +++ local / @@ -5,3 +5,5 @@` format.
  - Behavior-change ripple: TTY/`NO_COLOR` gating + line-number gutter + ANSI colorization were introduced in commit `cd5dd69` but no user-facing documentation describes the new UX or the opt-out.

## Scope

Align product documentation with the implemented behavior of `ralph upgrade`'s `[d]iff` view for:

- new line-numbered gutter (`<old> <new> │ <prefix><content>`, right-aligned, dynamic width with floor 2)
- new hunk-header format (`@@ 旧 L<start>–<end>  →  新 L<start>–<end> @@`, with `(空)` for empty sides)
- swapped label order (`--- local` / `+++ template (<version>)`)
- ANSI colorization gating (TTY-only, suppressed by any non-empty `NO_COLOR` per https://no-color.org)

No implementation, test, or template-mirrored files were touched.

## Files updated

| File | Change | Reason |
| --- | --- | --- |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/specs/2026-04-16-ralph-cli-tool.md` (§ upgrade フロー example, lines 273-282) | Replaced the stale 5-line diff snippet (`--- ralph template (0.6.0)` / `+++ local` / `@@ -5,3 +5,5 @@` / ` ...`) with a faithful 7-line capture of the current renderer: swapped label order (`--- local` / `+++ template (0.6.0)`), new hunk header (`@@ 旧 L5–7  →  新 L5–9 @@`), and four representative gutter rows showing equal/del/add/equal cases. Added the post-diff `template hash: ... local hash: ...` summary line that `showDiff` now emits. | Verifier flagged the stale snippet as LOW-severity drift; the spec is the canonical user-facing description of upgrade UX and was actively misleading after the format change. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/specs/2026-04-16-ralph-cli-tool.md` (§ 冪等性と自動修復, line 302 — Managed=false convergence bullet) | Extended the existing `[d]iff` description with: (a) the gutter-column contract (`<旧行番号> <新行番号> │ <prefix><内容>`, right-aligned, min 2 digits, dynamic widening), (b) the new hunk-header form including the `(空)` collapse for empty sides, (c) the ANSI color contract per line type (`---` bold red / `+++` bold green / `@@` cyan / `-` red / `+` green), (d) the TTY+`NO_COLOR` gating contract with explicit `https://no-color.org` reference, (e) the explicit suppression on pipe/redirect. | The convergence bullet is the contract-level description of the diff display. Pre-change wording mentioned only the line-direction (`--- local` / `+++ template`); without the gutter/color/`NO_COLOR` clauses, a reader reconstructing the contract from spec alone would not know the new acceptance criteria 1-4 (line numbers, color attributes, NO_COLOR suppression, non-TTY suppression) are first-class behavior. The wording matches the implementation anchors `internal/upgrade/unified_diff.go:51-72`, `internal/upgrade/colorize.go:7-15,61-87`, and `internal/cli/upgrade.go:85-97`. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/README.md` (Commands section, after `Run ralph help <command> for flags.`) | Added a new `### ralph upgrade interactive diff` subsection covering: the `[o]verwrite / [s]kip / [d]iff ?` prompt, the gutter format, the hunk-header format, the per-line-type colorization, and the `NO_COLOR` opt-out (with link to no-color.org) plus the automatic suppression on pipe/redirect. | README previously mentioned `ralph upgrade` only as a one-line command-table entry. The new colorize/gutter behavior is a user-visible UX change and the `NO_COLOR` opt-out is the documented escape hatch the plan's risk register pointed to (plan line 119: "CMD.exe is old environments may break → `NO_COLOR=1` で回避可能と README で案内"). Surfacing it in the README discharges that plan-level commitment. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md` (Progress checklist, lines 142-146) | Flipped `Review artifact created` from `[ ]` to `[x]` (already on disk at `docs/reports/self-review-2026-04-23-colorize-upgrade-diff.md`). Added a new `Sync-docs artifact created` line linking to this report. Verification and Test artifact lines were already checked in pass-1. | Bring plan progress in line with artifacts already on disk per `.claude/rules/planning.md` ("Keep progress checklists current while the task is in flight"). |

## Intentionally left alone

| Area | Why not changed |
| --- | --- |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/AGENTS.md` | Repo map describes `internal/upgrade/` as "hash-based diff engine, conflict resolution (auto-update, conflict, add, remove)". Colorization is a presentation refinement of the existing `conflict` action, not a new top-level action or contract. Keeping the map short per `.claude/rules/documentation.md`. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/CLAUDE.md` | No default-behavior or pipeline-level change. Colorize is opt-out, not opt-in. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/templates/base/` | `scripts/check-sync.sh` reports `IDENTICAL: 116, DRIFTED: 0` for the colorize commit. `README.md` is in `ROOT_ONLY_EXCLUSIONS` (line 73) by design — not mirrored — so the README addition introduces no mirror obligation. No file under `templates/base/` mentions `ralph upgrade` UX, the diff format, or `NO_COLOR` (verified via `grep -n "ralph upgrade\|@@\|NO_COLOR" templates/`). |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/rules/*` | No rule file references `ralph upgrade` conflict UI or diff display. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/recipes/*` | Grep confirmed no recipe documents `ralph upgrade` interactive prompt or diff output. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/architecture/repo-map.md` | Lists `internal/upgrade/` without behavioral contract. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/quality/*` | DoD / quality gates unchanged — colorize adds no new gate. |
| `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/tech-debt/README.md` | Already updated in working-tree to include the `runUpgradeIO` positional-parameter creep entry (line 23) — committed by /verify pass. No additional drift. |
| Implementation, test, and evidence files | Explicitly out of scope for `/sync-docs`. |

## Cross-reference checks performed

- `grep -l "ralph upgrade" docs/**/*` — outside of archived plans, the active plan, this branch's reports, and `docs/specs/2026-04-16-ralph-cli-tool.md`, no other doc references the command. Archived plans are frozen by convention; no edits.
- `grep -rn "@@ -\d" docs/ README.md AGENTS.md CLAUDE.md .claude/` — only the spec sample at line 275 matched (now fixed). No other surface still shows the pre-change hunk-header form.
- `grep -rn "NO_COLOR" docs/ README.md` — pre-edit: zero user-facing matches. Post-edit: spec line 302 + README new subsection. The opt-out is now discoverable from both the user manual (README) and the contract (spec).
- `scripts/check-sync.sh` — `IDENTICAL: 116, DRIFTED: 0, ROOT_ONLY: 1, TEMPLATE_ONLY: 0, KNOWN_DIFF: 3`. The single ROOT_ONLY item (`.claude/scheduled_tasks.lock`) is unrelated working-tree noise predating this branch and is not introduced by colorize. No DRIFTED files.
- Spec internal consistency: the new gutter/color clauses on line 302 (Managed=false bullet) match the new sample on lines 273-282 (upgrade フロー example) — both use the `--- local` / `+++ template (version)` order, both reference the `@@ 旧 L… → 新 L… @@` form, and both are consistent with `internal/upgrade/unified_diff.go` and `internal/cli/upgrade.go:374-379`.
- README/spec consistency: the README subsection's claims (gutter format, hunk header, color mapping, `NO_COLOR` suppression, pipe/redirect suppression) are a strict subset of the spec's contract wording — no drift between user-facing summary and contract spec.

## Evidence

- Verify drift recommendation: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/reports/verify-2026-04-23-colorize-upgrade-diff.md` §Documentation drift (LOW row for `docs/specs/2026-04-16-ralph-cli-tool.md:273-275`).
- Visual confirmation transcript referenced by the new spec sample: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt` (lines 5-13 show the byte-exact gutter format the spec sample now mirrors).
- Implementation anchors cited in the new spec wording:
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff.go:11` — `diffSeparator` constant (` │ `, multi-byte)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff.go:51-72` — gutter rendering loop
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff.go:85-93` — `formatRange` (`(空)` and `L<start>–<end>` format)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff.go:98-116` — `gutterWidth` (floor 2, dynamic widening)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/colorize.go:7-15,61-87` — ANSI constants + per-line gating
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/upgrade.go:85-97` — `shouldColorize` (NO_COLOR + TTY check)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/upgrade.go:374-379` — `--- local` / `+++ template (<version>)` label call site
- Test anchors locking in the documented contract:
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff_test.go:154-184` — `TestUnifiedDiff_GutterWidth_FiveDigitLineNumbers`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/cli_test.go:597-606` — `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/cli_test.go:615-645` — `TestRunUpgrade_InteractiveDiff_ColorizesWhenEnabled`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/cli_test.go:921-943` — `TestShouldColorize_HonorsNoColorAndTTY`

## Verdict

Documentation now matches implementation for all 6 acceptance criteria of the colorize-upgrade-diff plan. Spec, README, and plan are aligned with commit `cd5dd69` and the working-tree fixups. Mirror parity (`scripts/check-sync.sh`) holds. Ready for `/codex-review` (optional) and `/pr`.
