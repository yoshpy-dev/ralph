# Self-Review: loop-model-routing

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-loop-model-routing.md
- Branch: feat/loop-model-routing (base main at 45e9060)
- Commits: 77f8f74 (shell config+driver), 18ead46 (pipeline routing), 20c8a80 (Go config+toml), 709bafe (docs)
- Scope: diff quality only (no test execution, no spec compliance, no doc-drift, no static analysis)

## Verdict

MERGE. No CRITICAL or HIGH findings. The per-phase routing is implemented as a
pure, unit-testable resolver; precedence (`RALPH_FORCE_MODEL` > escalation >
per-phase var > fallback) is coherent across shell, TOML, and Go; the codex
driver's ignored-model limitation is honestly reflected in receipts
(`honored=false`); the three synced scripts are byte-identical to their
`templates/base/` mirrors; and no secrets, debug code, or swallowed errors were
introduced. Three LOW findings are documented for awareness — none block merge.

## Finding counts by severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 0 |
| LOW      | 3 |

## What was reviewed

Code files in scope (`git diff 45e9060..HEAD`):

- `scripts/ralph-cli-driver.sh` — new `resolve_phase_model`, `write_model_receipt`;
  `run_agent` / `_run_agent_claude` gain an optional 4th model arg.
- `scripts/ralph-config.sh` — 8 new per-phase defaults + `RALPH_FORCE_MODEL`,
  all exported.
- `scripts/ralph-pipeline.sh` — 6 phase call sites + 2 probes routed; escalation
  wiring; receipt calls; `run_inner_loop` gains a 3rd `_outer_cycle_num` arg.
- `internal/config/config.go` — `PhaseModelConfig` struct, `Default()`, `Load()`
  backfill.
- `internal/cli/run.go` — env export via `appendEnvIfMissing`.
- `templates/base/*` mirrors, `ralph.toml`, docs, and three test files.

Docs/plan/evidence artifacts skimmed but not nitpicked per task brief.

## Verification performed during review

- **Resolver behavior** (sourced `ralph-cli-driver.sh`, exercised directly):
  `implement` cycle 1 → `sonnet`, cycle 2 → `opus` (escalation), cycle 0/empty →
  `sonnet` (no escalation), `self_review` → `opus`, `probe` → `haiku`, unknown
  phase → `$RALPH_MODEL` fallback, `RALPH_FORCE_MODEL=haiku` forces every
  phase/cycle. All correct.
- **jq boolean encoding**: `write_model_receipt` uses
  `--argjson honored "$_wmr_honored"` where `_wmr_honored` is the string
  `true`/`false`. Confirmed `jq -cn --argjson honored true` emits a real JSON
  boolean (not a quoted string), so receipts type `honored` correctly.
- **Mirror byte-identity**: `cmp -s` confirms `scripts/ralph-config.sh`,
  `ralph-cli-driver.sh`, and `ralph-pipeline.sh` are identical to their
  `templates/base/` copies.
- **Cycle-var scoping**: `_outer_cycle_num` (3rd arg to `run_inner_loop`,
  default `0`) is in scope at all three inner-phase call sites (implement L511,
  self_review L601, verify L658, test L694). `sync_docs` (L776) and `pr` (L982)
  live in `run_outer_loop` where `_cycle="$1"` IS the outer cycle (called as
  `run_outer_loop "$_outer_cycle"`), so passing `$_cycle` there is correct.
- **`appendEnvIfMissing` + Force guard**: the non-Force phases are backfilled to
  non-empty defaults in `Load()`, so `appendEnvIfMissing` never writes a blank
  that would mask a user env var; Force is explicitly guarded with
  `if cfg.Pipeline.Phases.Force != ""`. Reasoning in the run.go comment matches
  the actual `appendEnvIfMissing` early-return semantics.
- **Secrets/debug scan**: no TODO/FIXME/console/password/token/api-key strings
  introduced; the only `mktemp` hit is a legitimate test workdir.

## Findings

### LOW-1: `write_model_receipt` can record `reason=escalation` with a non-escalated model under `RALPH_FORCE_MODEL`

In `run_inner_loop` (ralph-pipeline.sh:511-517) the receipt `reason` is computed
from the outer-cycle number alone:

```sh
_impl_model="$(resolve_phase_model implement "$_outer_cycle_num")"
if [ "$_outer_cycle_num" -ge 2 ] 2>/dev/null; then
  _impl_reason="escalation"
else
  _impl_reason="phase-default"
fi
```

But `resolve_phase_model` returns `RALPH_FORCE_MODEL` first (before the
escalation branch). So with `RALPH_FORCE_MODEL=sonnet` at outer cycle 2, the
receipt records `requested_model=sonnet, reason=escalation` — the reason claims
escalation even though FORCE overrode it and no escalation actually occurred.
Verified empirically. The `requested_model`/`effective_model` fields remain
accurate; only the free-text `reason` is slightly misleading. Low severity
because FORCE is a deliberate operator rollback knob and the audited model
values are still truthful. If tightened, gate the reason on
`[ -z "${RALPH_FORCE_MODEL:-}" ] && [ "$_outer_cycle_num" -ge 2 ]`.

Evidence: ralph-pipeline.sh:511-517; resolve_phase_model FORCE branch is
ralph-cli-driver.sh (RALPH_FORCE_MODEL wins over everything when set).

### LOW-2: escalation threshold (`>= 2`) is expressed in two places

The `outer cycle >= 2 → escalate` rule lives both inside `resolve_phase_model`
(the `implement` branch in ralph-cli-driver.sh) and inline in `run_inner_loop`
(the `_impl_reason` computation, ralph-pipeline.sh:512). They agree today, but a
future change to the threshold must touch both sites or the receipt `reason`
will silently disagree with the resolved model. Not a bug — a maintainability
snag. The resolver is the single source of truth for the *model*; the pipeline
only re-derives the *reason label*. Documenting for awareness; no action
required for merge.

### LOW-3: new Go test loop repeats the `for _, e := range` membership pattern (slices.Contains hint)

`internal/cli/run_env_test.go` gains `TestRunPipeline_ExportsPhaseModelEnv` /
`TestRunPipeline_ExportsForceModelWhenSet`, one of which uses a
`for _, e := range envLines` linear membership scan — the third of the three
"Loop can be simplified using slices.Contains" hints noted in the task brief.
The other two hints (~L56/L157) pre-exist on main (confirmed via
`git show 45e9060:internal/cli/run_env_test.go`); only this one is introduced by
the diff. It is stylistically consistent with the surrounding file, so this is a
cosmetic note, not a defect.

Evidence: `git diff 45e9060..HEAD -- internal/cli/run_env_test.go` shows the new
membership loop; base file already contains equivalent loops.

## Notable good practices in this diff

- **Codex honesty**: `_run_agent_codex` explicitly ignores the model arg with a
  comment, and `write_model_receipt` records `effective_model="codex-default"`,
  `honored=false` so receipts cannot claim a model the driver never applied.
- **jq-missing fallback**: `write_model_receipt` mirrors the existing
  `_run_agent_codex` printf fallback, so receipt writing degrades gracefully
  without jq.
- **Backward compatibility**: `run_agent` documents that 2- and 3-arg call sites
  are unchanged; the 4th arg is optional with a `$RALPH_MODEL` fallback.
- **Test cleanup discipline**: `tests/test-model-routing.sh` writes receipts only
  inside `cd "$TMP1"` (mktemp'd) and traps cleanup on those dirs — it does not
  touch the real repo's `.harness/state/`.
- **Force semantics documented consistently** across shell config comment,
  `ralph.toml`, Go struct doc comment, `model-routing.md`, and the recipe.

## Tech debt

None warranting a `docs/tech-debt/` entry. LOW-1 and LOW-2 are minor
receipt-accuracy/maintainability notes local to two adjacent lines; the codex
`honored=false` gap is already recorded as a known limitation in the plan and
`model-routing.md`.

## Evidence summary

- Resolver outputs verified by sourcing the driver and calling
  `resolve_phase_model` across phases/cycles/FORCE.
- `--argjson honored true|false` emits real JSON booleans (verified).
- Three synced scripts byte-identical to `templates/base/` (`cmp -s`).
- `_outer_cycle_num`/`_cycle` in scope at every routed call site.
- `appendEnvIfMissing` early-return + Force `!= ""` guard prevent blank-override
  masking.
- No secrets/debug/swallowed-error patterns introduced.
