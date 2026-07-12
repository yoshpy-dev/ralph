# repo-wide-drift-fixes

- Status: Approved (autonomous /sync-docs follow-up; findings from 3-agent repo-wide audit)
- Owner: Claude Code
- Date: 2026-07-12
- Related request: /sync-docs repo-wide drift investigation -> fix confirmed findings
- Related issue: N/A
- Type: docs
- Branch: docs/repo-wide-drift-fixes

## Objective

Fix all confirmed findings from the repo-wide drift audit (3 parallel
read-only auditors: skills-vs-implementation, docs-vs-implementation,
code-adjacent docs). One dismissed finding is recorded for transparency.

## Scope (10 fixes)

1. **Skills prompts mirror gap (audit A, M-1)**: `.agents/skills/cross-review/prompts/adversarial-claude.md`
   and `templates/base/.agents/skills/cross-review/prompts/adversarial-claude.md`
   are missing (present under `.claude/skills/...` in both trees). Copy the
   files byte-identically AND extend `scripts/check-skill-sync.sh` to compare
   every file under each skill's `prompts/` directory (byte parity, both
   directions: missing-in-mirror and extra-in-mirror are failures). Extend
   `tests/test-check-skill-sync.sh` with cases for a missing and a differing
   prompts file. Loop prompts (`loop/prompts/pipeline-*.md`) are already
   mirrored - the new gate must pass on them unchanged.
2. **pipeline-outer prompt hardcodes main (audit A, M-2)**: in all copies of
   `skills/loop/prompts/pipeline-outer.md` (.claude + .agents, root +
   templates/base), replace the hardcoded `git diff main...HEAD` instruction
   with dynamic base detection mirroring ralph-pipeline.sh (upstream ref via
   `git rev-parse --abbrev-ref 'HEAD@{upstream}'`, strip `origin/`, fall back
   to `main`).
3. **check-skill-sync intent note (audit A, M-3)**: one header comment in
   `scripts/check-skill-sync.sh`: `allowed-tools` / `disable-model-invocation`
   frontmatter is Claude-Code-specific and intentionally NOT mirrored (Codex
   equivalents live in `agents/openai.yaml` policy metadata); body parity is
   the contract.
4. **tech-debt stale plan paths (audit B)**: in `docs/tech-debt/README.md`,
   four `docs/plans/active/...` references whose plans are archived ->
   `docs/plans/archive/...` (2026-04-22-upgrade-detect-local-edits,
   2026-04-23-colorize-upgrade-diff, 2026-05-13-add-terraform-language-pack,
   2026-07-12-cli-stub-stdin-hang).
5. **tech-debt stale line refs (audit B)**: `ralph-cli-driver.sh:30-35`
   (pick_reviewer) and `:52-57` (count_triage_findings) -> current actual
   locations (verify with grep -n before writing; approx 184-189 / 155-180).
6. **quality-gates workflow annotation (audit B)**: verify which workflow
   runs `check-coverage.sh` and `check-pipeline-sync.sh` (grep
   .github/workflows/), then annotate those two lines in
   `docs/quality/quality-gates.md` (+ templates/base mirror, byte-identical
   pair) with the correct workflow file.
7. **ralph-loop.md legacy note (audit B)**: at the top of the "How it works"
   flow section describing `ralph-loop.sh`, add a short note: this describes
   the LEGACY standalone single-slice runner; `ralph run` / orchestrator-based
   multi-slice execution uses `ralph-orchestrator.sh` -> `ralph-pipeline.sh`
   (see "Integration with the operating loop"). Both copies (root +
   templates/base, byte-identical pair).
8. **repo-map missing dirs (audit B)**: add `docs/roadmap/`,
   `docs/research/`, `docs/references/` entries to
   `docs/architecture/repo-map.md` (confirm each dir exists first; match the
   file's list style).
9. **run.go flag help env mention (audit C)**: `--max-iterations` /
   `--max-parallel` help strings must name the env override, e.g.
   "total iteration cap (default from RALPH_MAX_ITERATIONS env or ralph.toml)".
10. **OrchestratorState doc comment + dormant KNOWN_DIFFS entry (audit C)**:
    `internal/state/types.go` OrchestratorState comment -> "represents a
    subset of orchestrator.json (fields needed by the TUI/status; unknown
    fields are ignored)". In `scripts/check-sync.sh`, AGENTS.md is listed in
    KNOWN_DIFFS but root/template copies are byte-identical today (confirmed
    via cmp) - remove the AGENTS.md entry so future accidental divergence
    fails the gate (re-verify identity first; if they differ after all, leave
    the entry and correct its comment instead).

## Dismissed finding (recorded)

Audit A H-1 (cross-review SKILL output-format json vs pipeline text) -
DISMISSED as context-aware-safe: json is the standard-flow inline reviewer
contract (the driving CLI parses the JSON itself); text is the Ralph Loop
internal dispatch, separately and correctly documented in the same SKILL's
"Reviewer inversion inside Ralph Loop" table. No change.

## Non-goals

- No behavior changes beyond help-string/comment text and the sync-gate
  coverage extension.
- No model-receipts/escalation additions to quality docs (deliberate
  division: model-routing.md owns that detail).
- Codex plan advisory intentionally skipped for this batch (docs plus a
  small deterministic-gate extension already covered by an existing test
  suite); cross-review still runs before PR.

## Acceptance criteria

- [ ] AC1: extended check-skill-sync fails on a missing/differing prompts
  file in a mirror (new test cases prove it) and passes on the repaired
  tree; both adversarial-claude.md mirror copies exist byte-identical.
- [ ] AC2: no `git diff main...HEAD` literal remains in any
  pipeline-outer.md copy; all four copies stay in lock-step.
- [ ] AC3: zero `docs/plans/active/` references in docs/tech-debt/README.md;
  the two line refs point at the real function locations.
- [ ] AC4: quality-gates annotation matches the actual workflow file(s);
  root/template pair byte-identical.
- [ ] AC5: ralph-loop.md legacy note in both copies; repo-map lists the
  three doc dirs; run.go help names env overrides; OrchestratorState comment
  says subset; AGENTS.md KNOWN_DIFFS entry removed (or comment corrected).
- [ ] AC6: full gates pass (check-sync, check-skill-sync incl. new cases,
  run-verify, run-test).

## Implementation outline

Single slice; verify every factual claim (line numbers, workflow ownership,
byte-identity) before editing.

## Risks and mitigations

- New prompts-parity gate could false-fail on legitimate one-sided files ->
  only cross-review (repaired here) and loop (already mirrored) have prompts
  dirs; tests cover both directions.
- Line-number refs will rot again -> accepted; only grossly wrong refs fixed.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (docs/repo-wide-drift-fixes)
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
