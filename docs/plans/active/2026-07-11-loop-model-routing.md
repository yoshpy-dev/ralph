# loop-model-routing

- Status: Approved (autonomous goal session; decisions self-selected and recorded below)
- Owner: Claude Code
- Date: 2026-07-11
- Related request: Use the session model (Fable 5) as the orchestrator "brain" and delegate execution to cheaper models without losing quality (PR1 of 2)
- Related issue: N/A
- Type: feat
- Branch: feat/loop-model-routing

## Objective

Introduce per-phase model routing into the Ralph Loop pipeline so that
token-heavy execution phases run on cheaper models (`sonnet`, `haiku`) while
judgment phases stay on `opus`, with a deterministic cycle-based escalation
path and an auditable model-receipt trail. This implements the
orchestrator–worker / plan-big-execute-small pattern (Anthropic-validated:
~96% quality at ~46% cost for orchestrator+worker; ~92% at ~63% for
executor+advisor) inside ralph's existing deterministic-script harness.

## Scope

1. **Per-phase model variables** (shell + toml + Go, lock-step):
   - `RALPH_IMPLEMENT_MODEL` (default `sonnet`) — Inner Loop implement phase
   - `RALPH_SELF_REVIEW_MODEL` (default `opus`) — judgment seat
   - `RALPH_VERIFY_MODEL` (default `sonnet`)
   - `RALPH_TEST_MODEL` (default `sonnet`)
   - `RALPH_SYNC_DOCS_MODEL` (default `sonnet`)
   - `RALPH_PR_MODEL` (default `sonnet`) — PR-creation agent turn
   - `RALPH_PROBE_MODEL` (default `haiku`) — CLI capability probes
   - `RALPH_ESCALATION_MODEL` (default `opus`) — see escalation below
   - `RALPH_FORCE_MODEL` (default unset) — when set, overrides **all** phase
     models (single-knob rollback / "run everything on X"). Precedence:
     `RALPH_FORCE_MODEL` > `RALPH_<PHASE>_MODEL` > phase default.
   - `RALPH_MODEL` remains the global fallback for any agent turn without a
     dedicated phase variable.
2. **`run_agent` model override + pure resolver**: optional 4th positional
   argument `run_agent <prompt> <log> [extra_args] [model]`; empty/omitted
   falls back to `$RALPH_MODEL`. Claude driver passes it to
   `claude -p --model`. Codex driver ignores it (documented limitation; Codex
   model lives in `.codex/config.toml`). Model selection itself lives in a
   pure helper `resolve_phase_model <phase> <cycle>` (prints the routed model,
   applying force-knob and escalation rules) so routing logic is unit-testable
   without running the pipeline or any CLI.
3. **Cycle-based escalation (quality-floor gate)**: when the Outer Loop enters
   a fix-and-revalidate cycle (outer cycle ≥ 2), the implement phase runs on
   `RALPH_ESCALATION_MODEL` instead of `RALPH_IMPLEMENT_MODEL`. Deterministic
   trigger — verify/test/cross-review failures are the quality floor; no LLM
   router.
4. **Model receipts**: helper
   `write_model_receipt <phase> <cycle> <requested_model> <reason>` appends
   JSONL to `.harness/state/pipeline/model-receipts.jsonl` (fields: `ts`,
   `phase`, `cycle`, `driver`, `requested_model`, `effective_model`,
   `honored`, `effort`, `reason`). For the codex driver
   `effective_model="codex-default"` and `honored=false` — receipts must never
   claim a model the driver did not apply. Written before every routed
   `run_agent` call **and** at the direct cross-review call sites
   (`claude -p` reviewer / `codex exec review`), including in `DRY_RUN=1`.
5. **ralph.toml `[pipeline.phases]`** section + `internal/config/config.go`
   defaults + `internal/cli/run.go` env export (`appendEnvIfMissing`, so
   env > toml > default priority holds).
6. **Docs**: `.claude/rules/model-routing.md` (per-phase table + escalation +
   receipts), `.claude/rules/subagent-policy.md` (loop section pointer),
   Ralph Loop recipe under `docs/recipes/` if it documents model config.
7. **Template mirrors**: `templates/base/scripts/{ralph-config.sh,
   ralph-cli-driver.sh,ralph-pipeline.sh}`, `templates/base/ralph.toml`,
   `templates/base/.claude/rules/model-routing.md` (+ subagent-policy if
   touched) kept identical per `check-sync.sh` / `check-pipeline-sync.sh`.
8. **Tests**: extend `tests/test-ralph-config.sh`,
   `tests/test-ralph-cli-driver.sh`; add escalation/receipt coverage
   (new `tests/test-model-routing.sh` or extension); Go table tests in
   `internal/config` for `[pipeline.phases]` parsing and defaults.

## Non-goals

- Standard-flow (/work) orchestrator discipline — the `implementer` subagent
  (sonnet) with structured handoff, cross-review SKILL.md `--model opus`
  hardcode fix, and Fable-5-as-orchestrator rules land in **PR2**
  (separate plan: `standard-flow-orchestrator`).
- Per-phase `effort` tuning (`RALPH_EFFORT` stays global).
- LLM-based dynamic routing/cascading (per-decision router overhead does not
  scale for agent workloads; ralph prefers deterministic triggers).
- Codex-side per-phase model control (`codex exec -m`) — recorded as known gap.
- Changing `.claude/agents/*.md` frontmatter tiers (already correct).

## Assumptions

- `claude -p --model sonnet|haiku|opus` accepts stable aliases (already used
  by `RALPH_MODEL`/`RALPH_CLAUDE_REVIEWER_MODEL`).
- Quality is preserved by plan artifacts + unchanged verify/test/cross-review
  gates, not by the implement model tier (model-routing.md doctrine, backed by
  Anthropic benchmark evidence).
- `templates/base/scripts/` must stay byte-identical to `scripts/` for the
  synced files (enforced by existing sync gates).

## Affected areas

- `scripts/ralph-config.sh` — new defaults + export + validation notes
- `scripts/ralph-cli-driver.sh` — `run_agent` 4th arg, `_run_agent_claude`
  model resolution, `write_model_receipt`
- `scripts/ralph-pipeline.sh` — call sites: implement (~L509), self-review
  (~L592), verify (~L647), test (~L681), sync-docs (~L764), PR (~L965),
  probes (~L301, ~L349); escalation wiring in the Outer Loop; receipt calls.
  Audit any other `claude -p` / `run_agent` call sites in `scripts/`
  (incl. `ralph-orchestrator.sh`) and route or explicitly leave them on
  `RALPH_MODEL` with a comment.
- `internal/config/config.go`, `internal/config/config_test.go` (or new),
  `internal/cli/run.go`
- `templates/base/ralph.toml`, `templates/base/scripts/*` mirrors,
  `templates/base/.claude/rules/model-routing.md`
- `.claude/rules/model-routing.md`, `.claude/rules/subagent-policy.md`
- `tests/test-ralph-config.sh`, `tests/test-ralph-cli-driver.sh`,
  `tests/test-model-routing.sh` (new)
- `docs/recipes/` Ralph Loop recipe (drift check)

## Design decisions

Critical forks were resolved autonomously (goal session; user pre-authorized
self-selection of recommended options). Recorded for review:

1. **Flow: standard /work, not Ralph Loop.** Editing `ralph-pipeline.sh`
   while Ralph Loop executes it is self-modifying and risky; the change set
   is a tightly coupled shell+Go+docs unit better done in verified slices.
2. **Default implement model drops to `sonnet` (behavior change).** Chosen
   over "opt-in only" because ralph's mission is "cheap by default, richer
   only when needed", `model-routing.md` already assigns scoped
   implementation to the sonnet tier, and Anthropic's published
   orchestrator/worker benchmarks support near-parity quality. Rollback is a
   single env var (`RALPH_IMPLEMENT_MODEL=opus`) or toml key.
3. **Model override as `run_agent` 4th positional arg**, not `VAR=x func`
   prefix (assignment persistence is unspecified/persistent in some POSIX
   shells, e.g. dash) and not per-phase resolution inside the driver (call
   sites in the pipeline know phase + cycle context; driver stays a dumb
   dispatcher).
4. **Escalation trigger is deterministic (outer cycle ≥ 2)**, not an LLM
   quality classifier: the existing verify/test/cross-review gates already
   act as the quality floor, and cascade routers add per-decision overhead
   that research shows does not scale for agent pipelines. Escalation applies
   to the implement/fix seat only; review seat is already `opus`.
5. **Receipts are JSONL under `.harness/state/pipeline/`** (runtime state,
   not canonical truth), mirroring the Route Receipts pattern; they make
   routing auditable and give tests a deterministic observation point in
   `DRY_RUN` mode. Receipts record `requested_model` vs `effective_model`
   separately so a driver that ignores the request (codex) cannot produce
   false audit evidence (Codex advisory finding 3).
6. **Probes move to `haiku`** — they only test CLI output-format capability,
   not intelligence; cheapest possible seat.
7. **Verify/test stay on `sonnet` despite Codex advisory finding 1's
   suggestion to keep them on `opus`**: parity argument — the standard /work
   flow already runs `verifier` and `tester` subagents on `sonnet` (agent
   frontmatter, merged in PR #113), and both phases mostly interpret the
   output of deterministic scripts (`run-static-verify.sh`, `run-test.sh`).
   The self-review judgment seat stays `opus`. The rest of finding 1 is
   adopted: escalation logic is proven by unit-testing the pure
   `resolve_phase_model` helper (reachable without a full pipeline run, which
   `DRY_RUN` short-circuits to COMPLETE).
8. **`RALPH_FORCE_MODEL` single-knob override** (Codex advisory finding 2):
   one documented rule for "make everything run on X" and for full rollback
   to pre-change behavior (`RALPH_FORCE_MODEL=opus`), instead of asking
   operators to reason about eight per-phase variables.

## Acceptance criteria

- [ ] AC1: `ralph-config.sh` defines, exports, and defaults all 8 new
  variables exactly as scoped; `tests/test-ralph-config.sh` asserts defaults
  and env-override behavior.
- [ ] AC2: `run_agent` honors the 4th-arg model for the claude driver
  (asserted via stub `claude` binary capturing `--model`); omitted arg falls
  back to `RALPH_MODEL`; codex driver is unaffected.
- [ ] AC3: `resolve_phase_model` unit tests prove routing without any CLI:
  `implement` cycle 1 → `RALPH_IMPLEMENT_MODEL`, `implement` cycle ≥ 2 →
  `RALPH_ESCALATION_MODEL`, `self_review` → `RALPH_SELF_REVIEW_MODEL`, and
  `RALPH_FORCE_MODEL=X` forces every phase/cycle combination to `X`.
- [ ] AC3b: A `DRY_RUN=1` pipeline pass writes receipts for each routed phase;
  every line parses with `jq`; `requested_model` matches the resolver output;
  under `RALPH_LOOP_DRIVER=codex` receipts show
  `effective_model="codex-default"` and `honored=false`.
- [ ] AC4: `ralph.toml` `[pipeline.phases]` keys parse into Go config with
  correct defaults when absent; `ralph run` exports the matching `RALPH_*`
  env vars with env > toml > default priority; `go test ./...` passes.
- [ ] AC5: `./scripts/check-sync.sh` and `./scripts/check-pipeline-sync.sh`
  pass. Synced scripts and `ralph.toml` are byte-identical between root and
  `templates/base/`; `.claude/rules/model-routing.md` follows the existing
  allowlisted-diff policy (template copy stays free of repo-only Go notes)
  rather than forced byte-identity.
- [ ] AC6: `.claude/rules/model-routing.md` documents per-phase variables,
  escalation rule, and receipts; "Where the values live" lists the new keys;
  no stale references left (grep for `RALPH_MODEL` docs mentions).
- [ ] AC7: `./scripts/run-verify.sh` passes end-to-end (shellcheck, tests,
  drift gates).

## Implementation outline

1. **Slice 1 — shell config + driver** (`ralph-config.sh`,
   `ralph-cli-driver.sh` + template mirrors + tests): new variables incl.
   `RALPH_FORCE_MODEL`, `resolve_phase_model`, `run_agent` 4th arg,
   `write_model_receipt` (requested/effective/honored), test coverage.
2. **Slice 2 — pipeline routing** (`ralph-pipeline.sh` + mirror + tests):
   route the 6 phase call sites + 2 probes, escalation in the Outer Loop,
   receipt calls, `DRY_RUN` escalation test.
3. **Slice 3 — Go config + toml** (`config.go`, `run.go`, tests,
   `templates/base/ralph.toml`): `[pipeline.phases]`, defaults, env export.
4. **Slice 4 — docs sync** (`model-routing.md`, `subagent-policy.md`,
   recipes + template mirrors).
   Each slice: implement → `./scripts/run-verify.sh` → commit (validation
   gate per `git-commit-strategy.md`).

## Verify plan

- Static analysis: `./scripts/run-verify.sh` (shellcheck for sh, `go vet`
  via language pack for Go).
- Spec compliance: acceptance criteria AC1–AC7 checked one-by-one against
  the diff by the `verifier` subagent.
- Documentation drift: model-routing.md/subagent-policy.md/recipe vs actual
  variable names; template mirrors via sync gates.
- Evidence: verify report in `docs/reports/verify-2026-07-11-loop-model-routing.md`.

## Test plan

- Unit: `tests/test-ralph-config.sh` (defaults/overrides),
  `tests/test-ralph-cli-driver.sh` (model arg → `--model` flag via stub CLI).
- Integration: `tests/test-model-routing.sh` — `DRY_RUN=1` pipeline pass
  asserting receipt sequence incl. escalation on cycle 2.
- Regression: full `tests/test-*.sh` glob via `./scripts/run-test.sh`;
  `go test ./...`.
- Edge cases: empty 4th arg ≡ omitted; unset phase var falls back to default;
  receipts dir missing (created); jq missing (receipt helper degrades
  gracefully like existing driver code); codex driver ignores model arg
  without error.
- Evidence: test report in `docs/reports/test-2026-07-11-loop-model-routing.md`.

## Risks and mitigations

- **Quality regression from sonnet implement seat** → unchanged opus
  self-review + verify/test gates + cycle-2 escalation; single-var rollback.
- **POSIX sh portability** (no `local`, arg handling) → follow existing
  `_prefixed` variable style; shellcheck gate.
- **Silent divergence root vs templates** → sync gates in run-verify (AC5).
- **Existing users surprised by cheaper default** → documented in
  model-routing.md + recipe; `RALPH_IMPLEMENT_MODEL=opus` restores prior
  behavior; receipts make actual routing visible in `ralph status` artifacts.
- **Codex driver semantics drift** (model arg ignored) → explicit comment +
  docs known-gap note.

## Rollout or rollback notes

- Rollout: merge → scaffolded projects pick the change up via `ralph upgrade`
  (hash-based diff engine handles the synced scripts).
- Rollback: revert PR, or set `RALPH_FORCE_MODEL=opus` (single knob restoring
  pre-change behavior for every phase) — finer-grained:
  `RALPH_IMPLEMENT_MODEL=opus` for the implement seat only.

## Codex plan advisory (evidence)

Codex (codex-cli 0.139.0, read-only) returned 4 findings; all adopted or
addressed with recorded rationale:

1. [HIGH] Escalation not deterministically provable under `DRY_RUN` →
   adopted: pure `resolve_phase_model` helper + AC3/AC3b split. Verify/test
   tier suggestion declined with parity rationale (Design decision 7).
2. [HIGH] Global override/rollback semantics underspecified → adopted:
   `RALPH_FORCE_MODEL` (Design decision 8), AC3 force coverage, rollback note.
3. [HIGH] Receipts as false audit evidence → adopted: `requested_model` /
   `effective_model` / `honored` schema + codex-driver and cross-review
   receipt coverage (Scope 4, AC3b).
4. [MEDIUM] AC5 conflicted with the existing `check-sync.sh` allowlist for
   `model-routing.md` → adopted: AC5 rewritten to byte-identity for scripts +
   toml, allowlist policy for the rule file.

## Open questions

- None blocking. Follow-up (PR2, separate plan): standard-flow orchestrator
  discipline — `implementer` subagent (sonnet) + structured handoff in
  `/work`, cross-review SKILL.md `--model opus` → `RALPH_CLAUDE_REVIEWER_MODEL`,
  haiku guidance for Explore fan-outs, model-routing sync-target list fix.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/loop-model-routing)
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
