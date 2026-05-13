# Sync-docs report — fix-xreview-placeholder-substitution (issue #50)

- Plan: `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md`
- Branch: `fix/50/xreview-placeholder-substitution`
- Date: 2026-05-13
- Commits in scope: `0304686`, `4f15681`, `d2dd875`, `f3363b6`, `fd3e958`, `12a1984`

## Behavior change recap

The cross-review gate under `RALPH_LOOP_DRIVER=codex` now:

1. Pre-renders `.claude/skills/cross-review/prompts/adversarial-claude.md`
   into `${PIPELINE_DIR}/outer-N-adversarial-claude.md` with
   `${BASE_BRANCH}` / `${REPORTS_DIR}` substituted (awk `index`/`substr`
   literal replacement — safe against `#`, `&`, `\`, `/` in refs or
   configurable report paths).
2. Fails the gate **closed** on any render failure
   (`render_failed_awk` from a non-zero awk exit, or
   `render_failed_unresolved_placeholders` from the allowlist guard).
3. Records the `_render_failed` flag in both the checkpoint JSON
   (`cross_review_triage.render_failed`) and the `report_event "cross-review"`
   JSONL line, so post-mortem telemetry can distinguish "reviewer ran clean"
   from "renderer broke".

Prior behavior: literal `${BASE_BRANCH}` / `${REPORTS_DIR}` strings were
piped to `claude -p`, the reviewer could not find or write the triage
report at the expected path, the `find … -newer checkpoint.json` parser
returned empty, and the gate silently passed (ACTION_REQUIRED defaulted to
0). Regression history: issue #50.

## Drift audit

| # | Surface | Verdict | Action |
|---|---------|---------|--------|
| 1 | `AGENTS.md` | No drift — speaks at contract level (primary loop, repo map) and never names placeholders or the renderer. | No edit. |
| 2 | `CLAUDE.md` | No drift — contract-level only. | No edit. |
| 3 | `README.md` | No drift — operating-loop section already says cross-review "calls `claude -p` with adversarial reviewer prompt" without quoting the implementation. | No edit. |
| 4 | `docs/quality/definition-of-done.md` | No drift — only mentions the canonical pipeline order. | No edit. |
| 5 | `docs/recipes/ralph-loop.md` | Lines 227-231 describe the inversion at contract level (names `adversarial-claude.md` by path but not its placeholder shape). The new rendering contract is now documented in `.claude/skills/cross-review/SKILL.md`, which is the canonical place. Adding a duplicate sentence to the recipe risks drift between the two. | No edit. |
| 6 | `docs/recipes/codex-setup.md` | No drift — only refers to cross-review at the `RALPH_PRIMARY_CLI` / triage-report level. | No edit. |
| 7 | `.claude/skills/cross-review/SKILL.md` | Already updated in `0304686` / `12a1984`. Spot-check confirmed the "Prompt rendering contract (claude reviewer path)" paragraph names the placeholder set, the per-cycle render path, the allowlist guard, the fail-closed contract, and the regression tests. | No edit. |
| 8 | `.agents/skills/cross-review/SKILL.md` | Byte-identical to the Claude side (`diff -q` reports no difference). `scripts/check-skill-sync.sh` reports 13 skills in lock-step (per plan progress). | No edit. |
| 9 | `.claude/skills/cross-review/prompts/adversarial-claude.md` | Already carries the rendering-contract HTML comment (lines 1-15) explaining that `${BASE_BRANCH}` / `${REPORTS_DIR}` are pre-rendered and that adding a new placeholder requires updating the renderer + allowlist. | No edit. |
| 10 | `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md` | Progress checklist had three open items (Review / Verification / Test artifact) that are now satisfied — corresponding files already exist under `docs/reports/`. | **Ticked** Review / Verification / Test boxes with explicit report paths. Left PR box open (PR not yet created). |
| 11 | `docs/tech-debt/README.md` | Already records the MEDIUM-#1 renderer-duplication entry (commits `0304686` / `12a1984`); no other drift detected (the unrelated cycle-2 LOW about telemetry-noise on render-failure paths is intentionally left out of tech-debt per cycle-2 self-review). | No edit. |
| 12 | `internal/state/types.go` (`CrossReviewTriage`) | **Drift considered, no edit.** The struct has fields for `action_required` / `worth_considering` / `dismissed` but no `RenderFailed`. The user's note is the authoritative call: `encoding/json` silently drops unknown keys on read and never writes a field that isn't on the struct, so checkpoints with `render_failed` round-trip correctly via the Go reader. Grep confirms no consumer in `internal/cli/` or `internal/state/` (including `reader_test.go`) reads `render_failed`. Adding the field would be inert and would invite a JSON-tag fallback discussion (similar to the tech-debt entry about `codex_review_triage` → `cross_review_triage` migration). Deferred. | No edit. |
| 13 | `internal/state/testdata/checkpoint.json`, `checkpoint-complete.json` | Fixtures predate `render_failed`. They still round-trip correctly because Go drops unknown fields on read and never writes the missing one. Adding a `render_failed: 0` line is cosmetic and would not change `reader_test.go` outcomes. | No edit (consistent with item 12). |
| 14 | `scripts/ralph-pipeline.sh:1037` initial-checkpoint template | The hard-coded JSON skeleton at line 1037 omits `render_failed`. This is benign — the field is added at `ckpt_update` time (line 884) on the first cross-review pass and is absent only during the brief window before the first cross-review fires. No consumer reads the absent value. | No edit (intentional). |

## Out-of-scope drift noted but not fixed

- **Go-side `RenderFailed` field**: explicitly deferred per the user's
  instruction in this scope ("prefer NOT to add it unless something actually
  reads it"). If a future TUI / status surface decides to display render
  failures distinct from ACTION_REQUIRED counts, the field can land then
  along with a JSON-tag fallback if needed. Tracked here for traceability
  rather than in `docs/tech-debt/README.md` because there is no current
  consumer to be hampered.
- **Initial-checkpoint skeleton at `scripts/ralph-pipeline.sh:1037`**: omits
  `render_failed`. Not a defect today; would be tightened naturally the next
  time the skeleton is touched.

## Evidence

- `diff -q .claude/skills/cross-review/SKILL.md .agents/skills/cross-review/SKILL.md` → no output (identical).
- `grep -rn "cross_review_triage\|CrossReviewTriage\|render_failed" internal/ cmd/` → only `internal/state/types.go`, `internal/state/reader_test.go`, and two `testdata/checkpoint*.json` fixtures. None of them read `render_failed`.
- Plan checklist now reflects the three completed report artifacts under `docs/reports/*-2026-05-13-fix-xreview-placeholder-substitution*.md`.
- `docs/tech-debt/README.md` already carries the MEDIUM-#1 renderer-duplication entry (last row), so the self-review follow-up is in lock-step with the code.

## Files touched by /sync-docs

- `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md` — progress checklist (3 boxes ticked with report paths).
- `docs/reports/sync-docs-2026-05-13-fix-xreview-placeholder-substitution.md` — this report.

No source code or skill body was modified by /sync-docs.
