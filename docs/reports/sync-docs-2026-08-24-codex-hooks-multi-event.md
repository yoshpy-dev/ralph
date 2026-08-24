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

---

# Cycle 2

- **Date:** 2026-08-24
- **Trigger:** fix commits `381b938` (require event-matched dispatcher
  argument in the doctor hooks check) and `ed2a2d0` (harden the shell
  gate to the same event-argument check, reword the doctor finding string,
  accept quoted event arguments) landed after cycle-1 sync-docs.
- **Scope:** drift these two fix commits introduced, per the cycle-2 verify
  report's "Documentation drift (cycle 2)" table — not a full repo-wide
  re-sweep.

## What changed

| File | Change |
|------|--------|
| `.codex/README.md` (doctor-check summary sentence, was line 168) | "validates `hooks.json` itself (present, valid JSON, schema-conformant, routed through the dispatcher)" → "...each shipped event routed through the dispatcher with that event's own name as the dispatcher argument — not just referenced by basename" |
| `templates/base/.codex/README.md` (twin) | same wording fix |

The old sentence did not literally contradict the strengthened check (it was
still true, just less precise — the check *does* require routing through the
dispatcher, it just now also requires the matching event argument). The
cycle-2 verify report flagged this as an optional, non-blocking extension
(not a required fix); applying it now closes the gap between the doc and the
actual `validateCodexHooksJSON` semantics added in `381b938`/`ed2a2d0`.

## What was checked and found already in sync (no change)

- **Finding-string / basename-only-check sweep:** `grep -rln "routed through
  ralph-dispatch.sh"` across `*.md`/`*.go`/`*.sh` — only hits are
  `tests/test-hook-wiring.sh`, `internal/cli/doctor_hooks_test.go`,
  `internal/cli/doctor.go` (code/test files, not doc drift; the test file's
  one `strings.Contains` match against the old bare finding string is an
  intentional substring check inside a still-correct positive-test assertion,
  already confirmed by the cycle-2 verify report's finding-string sweep).
  No doc site quotes the old finding string or describes the check as
  basename-only.
- `docs/quality/definition-of-done.md` — no `doctor`/`dispatch` mentions;
  nothing to go stale.
- `docs/recipes/codex-setup.md:28` ("not routed through the dispatcher") —
  generic phrasing, does not claim basename-only matching; no contradiction
  with the strengthened check.
- `docs/reports/cross-review-triage-codex-hooks-multi-event.md` — the
  cycle-2 verify report flagged this row's rationale sentence as stale
  ("修正対象は doctor 側のみ", pre-dating `ed2a2d0`'s shell-gate fix). Checked
  the current file: it was already corrected in commit `ef03408` (title:
  "docs: cycle-2 verify report and triage rationale correction"), which
  landed after the verify report's cycle-2 section was written. The verify
  report's own note is now itself a stale snapshot of a since-fixed doc, not
  a live gap — no action needed here.
- `docs/plans/active/2026-08-24-codex-hooks-multi-event.md` (AC-5 prose /
  progress checklist) — cycle-2 verify report noted this as a known,
  non-blocking gap expected to resolve at `/pr` (checklist tick + archival),
  consistent with cycle-1's assessment. Left as-is per that convention.

## Verification run

- `diff .codex/README.md templates/base/.codex/README.md` — identical.
- `./scripts/check-sync.sh` — PASS (157 identical, 0 drifted, 5 known-diff, 11 template-only)
- `./scripts/check-template-purity.sh` — PASS (no meta-repo-specific references found in templates)

## Files touched by this cycle-2 sync-docs pass

- `.codex/README.md`
- `templates/base/.codex/README.md`
- `docs/reports/sync-docs-2026-08-24-codex-hooks-multi-event.md` (this report)

---

# Cycle 3

- **Date:** 2026-08-24
- **Trigger:** pipeline cap raised 2→3 (operator approval) after cycle-2
  cross-review AR#1. Cycle-3 fix commits: `6c41189` (insight jsonl data fix —
  cycle-2 `sync_docs` event was mis-stamped `cycle:1`; also appended the
  `cross_review` cycle-2 event and rewrote the triage report), `b7cec3c`
  (shell hook-wiring gate now accepts double- and single-quoted event
  arguments, closing the quote-parity asymmetry with `dispatchEventArgRes` in
  `internal/cli/doctor.go` that cycle-3 self-review flagged as C3-5), `70bae95`
  (plan AC-5 prose + Deviations section + one new tech-debt row for the
  insight-cycle-stamping defect class).
- **Scope:** drift these three commits introduce, per this prompt's checklist
  and the cycle-3 verify report's "Documentation drift (cycle 3)" table — not
  a full repo-wide re-sweep.

## What was checked and found already in sync (no change)

- **Shell gate / doctor quoting behavior vs. README twins:** `6c41189` is a
  pure data fix (one jsonl field + a triage-report rewrite, no code/doc
  claims) and `b7cec3c` only changes what `tests/test-hook-wiring.sh`'s CI
  gate *accepts* — it does not change or contradict any documented behavior.
  `.codex/README.md:167-170` and its `templates/base/` twin already say (from
  cycle 2) "each shipped event routed through the dispatcher with that
  event's own name as the dispatcher argument — not just referenced by
  basename." That sentence describes `doctor`'s check, doesn't claim quoted
  forms are rejected, and doesn't mention the shell CI gate at all — so
  `b7cec3c` closing a quote-acceptance gap in the shell gate leaves it
  accurate. `grep -rln test-hook-wiring --include="*.md" .` confirms no
  doc site (README twins, `AGENTS.md`, `docs/architecture/repo-map.md`)
  describes `tests/test-hook-wiring.sh`'s quoting behavior at all — nothing
  to contradict.
- **`docs/insights/README.md`:** no change made. Confirmed no schema change
  — the cycle-2 `sync_docs` cycle-stamp fix in `6c41189` was a one-row data
  correction in a committed events file, not a schema or cycle-semantics
  change. `docs/insights/README.md:27` ("Default: `1` when `--cycle` is
  omitted from `insights-append.sh`") already accurately documents the root
  cause `70bae95`'s new tech-debt row points at — still true, still correct.
- **Old 2-cycle-cap references:** `.claude/rules/ralph/post-implementation-pipeline.md`
  and `docs/quality/definition-of-done.md` describe `RALPH_STANDARD_MAX_PIPELINE_CYCLES`'s
  **default** (2) as a property of the harness, not of this task. This task's
  cap raise to 3 is task-local state (`.harness/state/standard-pipeline/cycle-count.json`)
  and is correctly recorded as task-local in the plan's Deviations section
  (`70bae95`), not asserted as a global default change. No doc describes "2"
  as an immutable ceiling — both docs already state it's controlled by the
  env var and raisable per the cap-reached UX. No drift.
- **Cross-review triage report header** ("Cycle: 2/2 (cap reached)"): flagged
  by the cycle-3 verify report as known, non-blocking drift, but explicitly
  out of `/sync-docs` scope — that file is rewritten by `/cross-review` each
  cycle and will self-correct when cycle-3 `/cross-review` runs. Not touched
  here.

## Verification run

- `grep -rln "test-hook-wiring" --include="*.md" .` — no doc site describes
  the shell gate's quoting behavior; confirmed no contradiction.
- `grep -n "quote\|quoted\|basename" .codex/README.md .codex/hooks/README.md
  templates/base/.codex/README.md templates/base/.codex/hooks/README.md
  docs/architecture/repo-map.md AGENTS.md README.md` — only pre-existing,
  unrelated hits (`config.toml` quoted-`"false"` example, the
  basename-vs-event-argument sentence already fixed in cycle 2).
- `./scripts/check-sync.sh` — PASS (157 identical, 0 drifted, 5 known-diff,
  11 template-only)

## Files touched by this cycle-3 sync-docs pass

- `docs/reports/sync-docs-2026-08-24-codex-hooks-multi-event.md` (this
  report only — no other doc required a fix)
