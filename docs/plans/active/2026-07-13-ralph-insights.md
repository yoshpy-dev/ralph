# ralph-insights

- Status: Approved
- Owner: Claude Code
- Date: 2026-07-13
- Related request: ralph insights — aggregate evidence data (model receipts, pipeline reports) into actionable routing/pipeline insights
- Related issue: N/A
- Type: feat
- Branch: feat/ralph-insights
- Base: develop

## Objective

Make ralph's accumulated evidence queryable: a `ralph insights` command that
answers "which pipeline phase catches which kind of issue", "did escalation
(outer cycle ≥ 2) actually improve outcomes", and "were model routing requests
honored" — so model routing and pipeline configuration can be tuned from data
instead of intuition.

## Scope

1. **Insight event contract (sidecar)** — a small, versioned JSONL schema that
   pipeline phases append deterministically at run time. One file per task
   (`docs/insights/events/<date>-<slug>.jsonl`) to avoid cross-branch merge
   conflicts from parallel Ralph Loop slices. Committed to the repo (decision
   below).
2. **Deterministic appender** — `scripts/insights-append.sh` (arg-validated,
   jq-built JSON, shellcheck-clean, mirrored to templates). Called by:
   - `ralph-pipeline.sh` after each phase completes (loop flow, automatic)
   - post-implementation skills (`self-review`, `verify`, `test`,
     `cross-review`) via one instruction line (standard flow, agent-invoked;
     backfill is the safety net when an agent forgets)
3. **Go aggregation** — new `internal/insights` package:
   - event reader (glob `docs/insights/events/*.jsonl`, schema-tolerant)
   - receipts reader (`.harness/state/pipeline/model-receipts.jsonl`, optional)
   - backfill parser for existing `docs/reports/*.md` (best-effort: self-review
     severity counts, verify/test verdicts, cross-review triage counts)
   - aggregator (per-phase catch profile, escalation outcomes, honored-rate)
4. **CLI** — `ralph insights` (human table) and `ralph insights --json`;
   `ralph insights backfill [--apply]` (dry-run by default, idempotent via
   slug+phase dedupe key, emits `source:"backfill"` events).
5. Tests (Go unit + shell), docs (`docs/insights/README.md` schema doc,
   AGENTS.md map line, model-routing.md pointer), template mirroring.

## Non-goals

- No automatic tuning (insights inform humans; no LLM router, no auto-change
  of `RALPH_*_MODEL` values).
- No TUI pane; no charts. Plain table + JSON only in v1.
- No re-parsing of `docs/evidence/*.log` (raw logs stay out of scope).
- No cross-repo aggregation.
- No changes to what reports contain — the sidecar adds data, never replaces
  report artifacts.

## Assumptions

- `model-receipts.jsonl` may be absent (fresh clone, no loop run yet) — every
  aggregation must degrade gracefully to events-only or backfill-only views.
- Reports older than the 30-day retention may already be GC'd; backfill only
  covers what exists on disk.
- Standard-flow skill compliance is LLM-dependent; the deterministic writers
  are the pipeline hooks, and backfill covers gaps.

## Affected areas

- `internal/insights/` (new), `internal/cli/insights.go` (new),
  `internal/cli/root wiring`
- `scripts/insights-append.sh` (new), `scripts/ralph-pipeline.sh` (phase-end
  hook calls)
- `.claude/skills/{self-review,verify,test,cross-review}/SKILL.md` (+ mirror
  via `scripts/sync-skills.sh`)
- `docs/insights/` (new, committed), `AGENTS.md`,
  `.claude/rules/model-routing.md` (pointer), `templates/base/` mirrors
- `tests/test-insights-append.sh` (new), Go tests in `internal/insights` and
  `internal/cli`

## Design decisions

- **Flow: standard (/work)** — cohesive medium feature; parallel slicing adds
  integration cost without payoff. (User-confirmed.)
- **Data contract: structured sidecar + backfill** — machine-readable JSONL
  events written at pipeline time are the primary source; existing free-form
  Markdown reports are backfilled best-effort. Rationale: report parsing alone
  is permanently brittle; sidecar alone discards 3 months of history.
  (User-confirmed.)
- **Storage: committed `docs/insights/`** — insights survive clones and can be
  shared/reviewed; growth is bounded by the same GC policy family as reports.
  (User-confirmed.)
- **Event file per task, not one global JSONL** — parallel Ralph Loop slices
  commit on separate branches; appends to a single file would conflict on
  every merge. Per-task files (`events/<date>-<slug>.jsonl`) make merges
  trivially clean. (Derived; recorded here, not asked.)
- **Schema v1 (frozen fields)**: `schema`, `ts`, `slug`, `flow`
  (`standard|loop`), `phase`, `cycle`, `verdict` (`pass|fail|complete|
  action_required|n/a`), `findings` (`{critical,high,medium,low}`), `triage`
  (`{action_required,worth_considering,dismissed}`), `driver`, `model`,
  `source` (`pipeline|skill|backfill`). Unknown extra fields tolerated on
  read; `schema` bumps on breaking change.

Critical forks beyond the three above: None.

## Acceptance criteria

- [ ] AC1: `scripts/insights-append.sh` appends a schema-v1-valid JSON line to
      `docs/insights/events/<date>-<slug>.jsonl`; rejects missing required
      args with exit ≠ 0 and a usage message; shellcheck-clean.
- [ ] AC2: a `DRY_RUN=1` `ralph-pipeline.sh` pass emits one event per executed
      phase (asserted by a shell test, same fixture style as
      `tests/test-model-routing.sh`).
- [ ] AC3: `ralph insights` prints a per-phase summary (events found,
      verdicts, findings by severity, escalation outcomes, receipts
      honored-rate when receipts exist) and `ralph insights --json` emits the
      same aggregate as valid JSON; covered by Go unit tests with fixtures.
- [ ] AC4: with zero data sources present, `ralph insights` exits 0 with an
      explicit "no data yet" note naming the expected locations.
- [ ] AC5: `ralph insights backfill` (dry-run) lists derivable events from at
      least self-review, verify, test, and cross-review-triage report types;
      `--apply` writes them with `source:"backfill"`; a second `--apply` run
      adds zero duplicates (idempotency test).
- [ ] AC6: post-implementation skill bodies instruct the event append; mirror
      regenerated via `scripts/sync-skills.sh`; `check-skill-sync.sh`,
      `check-sync.sh`, `check-template.sh` all pass.
- [ ] AC7: `docs/insights/README.md` documents schema v1 and retention;
      AGENTS.md map and model-routing.md receipts section point to
      `ralph insights`.
- [ ] AC8: `go test ./... -count=1` and `./scripts/run-verify.sh` pass.

## Implementation outline

1. Slice 1 — contract + appender: `docs/insights/README.md` (schema v1),
   `scripts/insights-append.sh`, `tests/test-insights-append.sh`, template
   mirror.
2. Slice 2 — pipeline wiring: phase-end hook calls in `ralph-pipeline.sh`
   (incl. DRY_RUN), shell test, template mirror.
3. Slice 3 — Go package: `internal/insights` (event/receipt readers,
   aggregator) + unit tests with fixtures.
4. Slice 4 — CLI: `ralph insights` + `--json`; graceful empty-state; wire into
   root command; CLI tests.
5. Slice 5 — backfill: report parsers + `ralph insights backfill` +
   idempotency tests (fixtures copied from real reports).
6. Slice 6 — skills + docs: skill instruction lines, `sync-skills.sh` regen,
   AGENTS.md/model-routing.md touches, final sync gates.

## Verify plan

- Static analysis: `gofmt -l`, `go vet ./...`, shellcheck on new/changed
  scripts (already in CI scope after core-hardening).
- Spec compliance: walk AC1–AC8 with evidence.
- Documentation drift: AGENTS.md map, model-routing.md receipts pointer,
  docs/insights/README.md vs implemented schema; skill mirror parity.
- Evidence: `docs/evidence/verify-<date>-ralph-insights.log`.

## Test plan

- Unit: event reader (valid/corrupt/missing lines), aggregator (phase
  grouping, escalation bucketing, honored-rate), backfill parsers (fixture
  reports incl. malformed ones → skipped-with-count, never panic).
- Integration: DRY_RUN pipeline emits events end-to-end; `ralph insights`
  against a fixture tree; backfill idempotency double-run.
- Regression: existing pipeline tests (`test-model-routing.sh`,
  `test-ralph-*.sh`) unchanged and passing — the hook must not alter phase
  semantics or exit codes.
- Edge cases: empty events dir; receipts without events; events without
  receipts; slug with unusual characters; schema field missing (older event).
- Evidence: test report + `docs/evidence/` log.

## Risks and mitigations

- R1: JSONL merge conflicts from parallel slices → per-task event files
  (design decision above).
- R2: standard-flow agents forget to append events → backfill parser as
  safety net; skill instruction kept to one copy-pasteable command.
- R3: backfill parser breaks on free-form reports → best-effort with
  parse-miss counters surfaced in output; never a hard failure.
- R4: `docs/insights/` grows unbounded → same retention family as reports;
  note in README; gc-artifacts.sh extension deferred (documented follow-up).
- R5: pipeline hook failure must not fail the pipeline → appender invoked
  fire-and-forget (`|| log_warn`), asserted by test.

## Rollout or rollback notes

- Single PR to `develop`. Additive feature: no existing behavior changes
  except one hook call per pipeline phase (guarded, non-fatal).
- Rollback = revert the PR; event files already committed are inert data.

## Open questions

- None blocking. Follow-up candidates: gc-artifacts.sh coverage for
  `docs/insights/events/`, TUI pane, cross-repo aggregation.

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/ralph-insights, worktree .claude/worktrees/ralph-insights)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
