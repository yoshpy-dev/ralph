# sync-docs report: codex-effective-model-receipt

- Date: 2026-09-20
- Plan: `docs/plans/active/2026-09-20-codex-effective-model-receipt.md` (issue #165)
- Prior reports: `docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md`
  (MERGE after cycle-1 fixes, all 11 findings resolved),
  `docs/reports/verify-2026-09-20-codex-effective-model-receipt.md` (Pass, no
  documentation drift found), `docs/reports/test-2026-09-20-codex-effective-model-receipt.md`
  (Pass, all commands green; named coverage/test gaps, none blocking)
- Scope: `git diff main...HEAD` on `feat/codex-effective-model-receipt`, HEAD
  `771e446`

## Summary

Read `internal/org/codex_session.go`, `internal/org/spawn.go` (the
`observeCodexSpawnReceipt`/`codexFoundReceipt`/`codexUnknownReceipt`
section), `internal/org/verbs.go` (`Stop`, `codexSpawnCorrelation`,
`hasObservedCodexReceiptSince`, `observeStopModelReceipt`),
`internal/cli/org.go` (`printCodexModelMismatchWarning` and its two call
sites, spawn/start and stop), and `internal/cli/doctor_codex_models.go`
(`checkCodexModelSlugs`, `codexRetirementClause`) against the doc passages
this task's hand-off listed as already updated. All of them (rule, spec
FR-9, recipe + template, `/org` skill's new "codex 座席の実効モデルは
receipts に記録される" bullet across all 4 mirrors, evidence file) match
the code at HEAD, including the self-review cycle-1 details (three-directory
window, pointer-sentence match, three distinct `unknown` reasons, stop
appends only when it finds the record, `model_observed=` token). No
inaccuracy found in any of them; nothing there was edited.

Two real gaps were found and fixed while checking the "not yet touched"
surfaces the hand-off named:

1. The `/org` skill's own "既定の model_pool" section — the detailed
   reference for the "Org codex model slugs" doctor Check the front-matter
   bullet already promises shows retirement info — described only the
   Check's `warn` (slug missing from cache) outcome. It never mentioned the
   `info` (slug present but scheduled to retire) outcome Slice C of this
   same plan added, even though the live default pool's `gpt-5.5` triggers
   exactly that outcome today (evidence file, "`ralph doctor` の表示"
   section).
2. `docs/tech-debt/README.md` had no row for the observer's dependency on
   codex's own undocumented session-record shape, measured against only
   codex-cli 0.154.0 and five real records — a format change in a future
   codex-cli release degrades every codex-seat receipt to `honored=unknown`
   silently (no error, no failing test), which is exactly the shape of risk
   this file tracks elsewhere.

## Files changed in this pass

- `.claude/skills/org/SKILL.md` — "既定の model_pool" section: added one
  sentence noting that a cached-but-retiring slug gets an `info` result
  naming the replacement and retirement date, and that a missing-slug
  `warn` takes priority over it. Placed right after the existing `warn`
  sentence, before the disappearance-handling paragraph that follows.
- `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`,
  `templates/base/.agents/skills/org/SKILL.md` — regenerated via
  `./scripts/sync-skills.sh` (root → `.agents/`) plus a manual copy to the
  two `templates/base/` paths (the sync script only mirrors `.claude/` →
  `.agents/`, not the template copies — see agent-memory
  `feedback_sync_skills_leaves_templates_stale.md`); all four are
  byte-identical again after the copy.
- `docs/tech-debt/README.md` — appended one row: `ObserveCodexEffectiveModel`
  parses codex's own undocumented JSONL session-record shape, verified only
  on codex-cli 0.154.0 against five real records; every parse failure
  already degrades silently to `unknown` by design, so a future codex-cli
  format change would silently stop matching with no operator-visible
  signal. Points at the evidence file's "確認していないこと" (c) and the
  plan's own Risks and mitigations row for the same risk. 6-pipe row shape
  and even backtick count checked against the file's existing 5-column
  convention (see Gate results below).
- `docs/plans/active/2026-09-20-codex-effective-model-receipt.md` —
  checked off "Verification artifact created" and "Test artifact created"
  in the Progress checklist (both reports already exist and are Pass; the
  checklist was simply stale, not a Scope/AC edit). Added one Deviation
  notes bullet ("2026-09-20 sync-docs cycle 1") summarizing this pass's two
  edits, the two follow-up candidates considered and deliberately not
  turned into a second tech-debt row (to avoid duplicating the test
  report's Test gaps and the plan's own Open questions), and the surfaces
  confirmed unchanged.

## Surfaces checked — no change needed

- **`README.md`** — no description of individual `ralph doctor` checks or
  org model receipts beyond the top-level feature table and the `ralph org`
  verb list; nothing there claims anything about codex model observation
  that could now be stale.
- **`AGENTS.md`** repo map, `internal/org/` line — already lists "receipts"
  among the package's responsibilities; kept as-is. `codex_session.go` is
  new implementation supporting that same "receipts" concern, and AGENTS.md
  is deliberately a short map (`.claude/rules/ralph/documentation.md`), not
  a file-by-file index — no other line in this same list names an
  individual source file either.
- **`docs/insights/README.md`**, "Org runtime receipts" section — only
  points at `.claude/rules/ralph/model-routing.md`'s "Org runtime model
  receipts" section for the schema/aggregation contract, which is accurate
  (that section already documents the codex observation, and the event
  schema `docs/insights/README.md` itself owns is untouched by this plan).
- **`internal/org/prompts/lead.md`** — deliberately left unchanged. Lead
  runs `ralph org spawn`/`stop` itself and sees the CLI's own stderr output
  directly in the same turn; `printCodexModelMismatchWarning`'s text is
  already self-describing (names both the commanded and reported model,
  the two likely causes, and `ralph doctor` as the next step), so there is
  nothing to prime in advance. `lead.md` already defers all verb detail to
  the `/org` skill ("動詞の詳しい使い方...は `/org` skill...を...参照して
  ください"), which now documents this feature; duplicating that reference
  chain into an already-tight role prompt was not worth it.
- **`.claude/skills/org/SKILL.md`** verb-reference table, `spawn`/`stop`
  rows — left terse (`spawn`'s row lists flags, `stop`'s is one clause,
  unchanged from before this plan). No other Details token in the file
  (e.g. `stopped`'s existing "pane=ok leave=ok" shape) is named at that
  same table's level of detail either, so adding `model_observed=` there
  alone would be an inconsistent level of documentation; the "前提" section
  bullet already explains the up-to-8-s spawn poll and the single stop-time
  check in prose.
- **`docs/specs/2026-08-01-org-runtime.md`**, "運用ノート: 既定 codex
  スラッグの更新手順" (issue #156) — describes a different failure mode
  (a slug disappearing from `models_cache.json` entirely, requiring a
  multi-week manual observation) than the `info`/retiring signal this plan
  added (a slug present in the cache with a scheduled retirement date).
  Makes no claim the new signal would contradict; left unedited. AC line 72
  ("receipts の `effective_model` が一致する") predates this branch and
  uses looser terminology than the actual field name
  (`reported_effective_model`, FR-9) — pre-existing, outside this plan's
  touched surfaces, not fixed here.
- **`docs/evidence/codex-seat-permissions-2026-09-18.md`** — already
  carries the one added line under P5 (per the hand-off); still accurate,
  no further edit.
- **`docs/evidence/codex-effective-model-receipt-2026-09-20.md`** — the
  plan's own real-record evidence file; already complete (model/status
  only, no session content), no change owed by a docs-only pass.

## Gate results

| Check | Result |
| --- | --- |
| `./scripts/check-skill-sync.sh` | PASS — 13 skill(s) in lock-step |
| `./scripts/check-sync.sh` | PASS — `DRIFTED: 0` (158 IDENTICAL, 11 TEMPLATE_ONLY, 5 KNOWN_DIFF, all pre-existing and unrelated to this change; `.claude/rules/ralph/model-routing.md`'s root-vs-template divergence is an existing, documented `KNOWN_DIFF`) |
| `./scripts/check-template-purity.sh` | PASS — no meta-repo-specific references found in templates |
| `go test ./internal/org/ -run TestSendDefaults -count=1 -v` | PASS — `TestSendDefaults_DocsMatchConstant` |
| Markdown table well-formedness (new tech-debt row) | Manually checked: 6 pipes (matches every other row in the file's 5-column table) and an even backtick count (26, balanced) on the new line |
| Credential-shaped string scan | `git diff -- docs/tech-debt/README.md docs/plans/active/2026-09-20-codex-effective-model-receipt.md \| grep -niE "key\s*=\|token\s*=\|secret\s*=\|password\s*="` — no match. No `.gitallowed` edit was made. |

## Known gaps

- Did not re-run `go test ./internal/org/... ./internal/cli/...`, `-race`,
  or `TMPDIR=/tmp` — those are `/test`'s completed job (Pass, cited above);
  this pass only ran the one gate the hand-off named that reads a doc
  surface this pass touched.
- Did not re-run the range secret scan (`./scripts/secret-scan.sh --range
  ...`) — that is a `/pr`-time pre-check per this task's own constraints;
  manually spot-checked the diff for credential-shaped `key=value` strings
  (none, see Gate results) and confirmed no `.gitallowed` edit was made.
- Did not add a second tech-debt row for the other two candidates the
  hand-off named ((a) no live end-to-end spawn-time observation on a real
  seat, (b) whether codex creates a session record while the
  model-retirement dialog is open): (a) is already named in the test
  report's own Test gaps section under the same evidence-file citation;
  (b) is already the plan's own Open Questions entry, with a working
  fallback already implemented (stop-time re-observation) rather than a
  deferred fix. Recording either again in `docs/tech-debt/README.md` would
  duplicate an existing, still-current record rather than surface a new one.
