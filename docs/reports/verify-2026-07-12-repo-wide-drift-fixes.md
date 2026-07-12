# Verify report: repo-wide-drift-fixes

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-repo-wide-drift-fixes.md
- Verifier: verifier subagent (spec compliance + static analysis)
- Scope: `git diff main...HEAD` — commits 849f60f (plan), 9e39175 (impl), d6cc77b (self-review), 89658bd (header fix); 19 changed files
- Evidence: `docs/evidence/verify-2026-07-12-104725.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: check-skill-sync extended for prompts/ parity; new test cases H–K prove it; adversarial-claude.md in all 4 locations, byte-identical | PASS | `scripts/check-skill-sync.sh` lines 235–268 implement check 6; `tests/test-check-skill-sync.sh` lines 139–172 add cases H–K (missing-in-mirror, differing content, identical, real-tree no-regression); `cmp` confirms all 4 copies of `adversarial-claude.md` are byte-identical (root .claude vs .agents IDENTICAL; templates/base .claude vs .agents IDENTICAL; root vs templates/base IDENTICAL); check-skill-sync reports "13 skill(s) in lock-step" |
| AC2: no `git diff main...HEAD` literal in any pipeline-outer.md copy; all 4 copies in lock-step | PASS | `grep -r "git diff main\.\.\.HEAD"` returns no matches across all 4 copies; dynamic base detection snippet present at `.claude/skills/loop/prompts/pipeline-outer.md:17-19`; `cmp` confirms all 4 copies byte-identical (.claude vs .agents IDENTICAL; templates/base .claude vs .agents IDENTICAL; root vs templates/base IDENTICAL) |
| AC3: zero `docs/plans/active/` refs in docs/tech-debt/README.md; two line refs point at real function locations | PASS | `grep "docs/plans/active/"` returns no matches; `pick_reviewer` confirmed at `scripts/ralph-cli-driver.sh:184` and `count_triage_findings` at `scripts/ralph-cli-driver.sh:155`; tech-debt README references `:184-189` and `:155-178` which match actual function start lines |
| AC4: quality-gates annotation matches actual workflow file(s); root/template pair byte-identical | PASS | Both `check-coverage.sh` and `check-pipeline-sync.sh` lines annotated with `(.github/workflows/verify.yml)`; `grep -rln` in self-review confirmed these scripts run only in `verify.yml`; `cmp docs/quality/quality-gates.md templates/base/docs/quality/quality-gates.md` → IDENTICAL |
| AC5: ralph-loop.md legacy note in both copies; repo-map lists 3 doc dirs; run.go help names env overrides; OrchestratorState comment says subset; AGENTS.md KNOWN_DIFFS entry removed | PASS | Legacy note at `docs/recipes/ralph-loop.md:88-95` ("Note (legacy flow): …legacy standalone runner…"); root/template `cmp` → IDENTICAL; repo-map lists `docs/roadmap/`, `docs/research/`, `docs/references/` at lines 34-36; `run.go:41-42` help strings mention `RALPH_MAX_ITERATIONS` and `RALPH_MAX_PARALLEL` env; `internal/state/types.go:22` OrchestratorState comment says "represents a subset of orchestrator.json (fields needed by the TUI/status; unknown fields are ignored)"; `scripts/check-sync.sh` KNOWN_DIFFS no longer contains `"AGENTS.md"` (diff confirms removal); `cmp AGENTS.md templates/base/AGENTS.md` → IDENTICAL so removal is safe |
| AC6: full gates pass (check-sync, check-skill-sync incl. new cases, run-verify, run-test) | PARTIAL — static gates PASS; run-test is tester's responsibility | `scripts/check-sync.sh`: DRIFTED=0, ROOT_ONLY=0; `scripts/check-skill-sync.sh`: 13 skills in lock-step; `scripts/run-static-verify.sh`: exit 0 "All verifiers passed"; gofmt OK, go vet 0 issues; `run-test.sh` out of scope for /verify |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh < /dev/null` | PASS (exit 0) | All verifiers passed; evidence at `docs/evidence/verify-2026-07-12-104725.log` |
| `scripts/check-sync.sh` | PASS | IDENTICAL:171, DRIFTED:0, ROOT_ONLY:0, TEMPLATE_ONLY:10, KNOWN_DIFF:3 |
| `scripts/check-pipeline-sync.sh` | PASS | All 8 pipeline reference files OK |
| `scripts/check-skill-sync.sh` | PASS | 13 skills in lock-step |
| `shellcheck scripts/check-skill-sync.sh` | PASS | Self-review confirmed clean (0 warnings) |
| Go verifier (gofmt + go vet) | PASS | gofmt: ok; 0 issues |
| `sh -n` on all hooks | PASS | All 18 hook files OK |
| `jq -e .` on settings.json (root + template) | PASS | Both files valid JSON |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md` plan path references | YES | All four `docs/plans/active/` refs replaced with `docs/plans/archive/` paths |
| `docs/tech-debt/README.md` line refs | YES | `pick_reviewer:184-189` and `count_triage_findings:155-178` match actual function start lines in `scripts/ralph-cli-driver.sh` |
| `docs/quality/quality-gates.md` workflow annotation | YES | Both check scripts annotated with `(.github/workflows/verify.yml)`; root/template byte-identical |
| `docs/recipes/ralph-loop.md` legacy note | YES | Legacy note added in "How it works" flow section; both copies (root + templates/base) byte-identical |
| `docs/architecture/repo-map.md` | YES | Three new doc dir entries match actual directory presence |
| `internal/cli/run.go` help strings | YES | Both `--max-iterations` and `--max-parallel` name env overrides in help text |
| `internal/state/types.go` OrchestratorState comment | YES | Comment now states "subset … unknown fields are ignored" |
| `scripts/check-sync.sh` KNOWN_DIFFS | YES | AGENTS.md entry removed; root/template are byte-identical (safe removal) |
| `scripts/check-skill-sync.sh` intent note | YES | Lines 26-29 document Claude Code-specific frontmatter scope explicitly |
| Self-review LOW-1 (sed anchor divergence) | ACCEPTED | Documented as benign deviation; no action taken per self-review recommendation |
| Self-review LOW-2 (test header comment "five" → "six") | FIXED (commit 89658bd) | Header now reads "six drift modes" per self-review recommendation |
| New legacy note / annotations vs other docs | NO CONFLICT | Legacy note accurately describes current architecture; AGENTS.md and CLAUDE.md are consistent with the note content |

## Observational checks

- Byte-identity verified via `cmp` for all 4 mirror pairs that required
  lock-step updates: `adversarial-claude.md` (×4), `pipeline-outer.md` (×4),
  `quality-gates.md` (×2), `ralph-loop.md` (×2).
- `AGENTS.md` root vs template: `cmp` confirms IDENTICAL after KNOWN_DIFFS
  removal; `check-sync.sh` run confirms DRIFTED=0 (the removed entry no longer
  masks future divergence).
- Test cases H–K in `tests/test-check-skill-sync.sh` exercise all failure
  modes of check 6 (one-sided prompts/, differing content) and the
  no-regression path (identical content, real repo tree). All 11 cases
  confirmed passing by self-review (`bash tests/test-check-skill-sync.sh`
  output: 11/11 PASS).
- `run.go:41-42` help strings confirmed via grep; env variable names match
  `RALPH_MAX_ITERATIONS` and `RALPH_MAX_PARALLEL` exported by `run.go:66-83`.

## Coverage gaps

- `run-test.sh` (behavioral tests) not run here — tester's responsibility per
  pipeline contract. AC6 partial pending test run.
- Runtime resolution of `run_agent`, `pick_reviewer`, `count_triage_findings`
  and the prompts/ parity gate under live execution cannot be verified
  statically; classified as "likely but unverified" pending `/test`.
- Dismissed finding (Audit A H-1: cross-review SKILL output-format json vs
  pipeline text) remains dismissed; no static evidence contradicts the
  dismissal rationale.

## Verdict

- Verified: AC1 (fully), AC2 (fully), AC3 (fully), AC4 (fully), AC5 (fully)
- Partially verified: AC6 — static gates pass; behavioral tests pending
- Not verified: none (all claims have static evidence)

**Overall: PASS (static gates)**. No blocking issues. Proceed to `/test`.
