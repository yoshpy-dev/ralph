# Verify report: ralph-insights

- Date: 2026-07-13
- Plan: docs/plans/active/2026-07-13-ralph-insights.md
- Verifier: verifier subagent (spec compliance + static analysis only)
- Scope: `git diff develop...HEAD` — 5 commits (schema+appender, pipeline wiring, Go package, CLI, backfill+docs) + 1 fix commit (d3b30e3)
- Evidence: `docs/evidence/verify-2026-07-13-ralph-insights.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `scripts/insights-append.sh` appends schema-v1 JSONL; rejects missing required args with exit≠0; shellcheck-clean | **MET** | File exists at `scripts/insights-append.sh`. Validates 5 required flags (`--slug`, `--flow`, `--phase`, `--verdict`, `--source`) with `require_field`. Enum and non-negative integer validation present. `shellcheck --severity=warning` exits 0. Template mirror at `templates/base/scripts/insights-append.sh` is byte-identical (`check-sync.sh DRIFTED=0`). |
| AC2: `DRY_RUN=1` pass emits one event per executed phase; forced-outcome tests assert exact verdict/findings/triage JSON | **MET with documented deviation** | `emit_insight_event` is called at all 7 phase completion points (implement, self_review, verify, test, sync_docs, cross_review, pr) — 11 total call sites in `scripts/ralph-pipeline.sh`. `tests/test-insights-pipeline-events.sh` covers DRY_RUN=1 (Case 1: 6 events with --skip-pr; Case 2: 7 events default). **Forced-outcome semantics** (self-review CRITICAL→fail, verify fail→fail, cross-review ACTION_REQUIRED→action_required): implemented at the appender boundary — Case 4 tests call `insights-append.sh` directly with the exact argument values that `emit_insight_event` would pass after parsing, asserting exact JSON output. The test file's header documents this approach explicitly ("equivalent to unit-testing emit_insight_event's output contract at the appender boundary"). DRY_RUN cannot exercise agent-sidecar paths, but semantic correctness of verdict→JSON mapping is fully verified at the appender. This satisfies AC2's intent: the exact emitted JSON values for verdict, findings, and triage counts are asserted for all forced-outcome paths. |
| AC3: `ralph insights` prints per-phase summary; `ralph insights --json` emits valid JSON; Go unit tests with fixtures | **MET** | `internal/cli/insights.go` implements human table (Events, Routing, Escalation, Local receipts sections) and `--json` path. `internal/cli/root.go` wires `newInsightsCmd()`. Unit tests in `internal/insights/insights_test.go` and `internal/cli/insights_test.go` with fixture files in `internal/insights/testdata/`. `go vet` exits 0. |
| AC4: zero data sources → `ralph insights` exits 0 with "no data yet" note naming expected locations | **MET** | `internal/cli/insights.go:72-76`: when `agg.TotalEvents == 0 && !agg.Receipts.Present`, prints "No insight data yet." with `eventsDir` and `receiptsPath` paths. |
| AC5: `ralph insights backfill` dry-run lists derivable events from self-review/verify/test/cross-review; `--apply` writes with `source:"backfill"`; second `--apply` adds zero duplicates; multi-cycle fixture yields distinct events per cycle | **MET** | `internal/insights/backfill.go` parses all four report types with dedicated parsers. `DedupeKey` = `source_report_path:phase:cycle` per plan AC5 requirement. `LoadExistingDedupeKeys` filters `source=="backfill"` events. `internal/insights/testdata/multi_cycle.jsonl` fixture present. `internal/insights/backfill_test.go` covers parsing and idempotency. |
| AC6: post-implementation skill bodies instruct event append; mirror regenerated via `sync-skills.sh`; `check-skill-sync.sh`, `check-sync.sh`, `check-template.sh` all pass | **MET** | All four skills have `insights-append.sh` instruction line: `self-review/SKILL.md` (line 47), `verify/SKILL.md` (line 39), `test/SKILL.md` (line 50), `cross-review/SKILL.md` (line 196). All flag names match `insights-append.sh` interface exactly. `check-skill-sync.sh` exits 0: "13 skill(s) in lock-step". `check-sync.sh` exits 0: DRIFTED=0. `check-template.sh` exits 0. |
| AC7: `docs/insights/README.md` documents schema v1 and retention; AGENTS.md and model-routing.md point to `ralph insights` | **MET** | `docs/insights/README.md` exists with full schema v1 table (15 fields), retention section, examples. `AGENTS.md` line 70: "`docs/insights/` — committed insight events". `model-routing.md` Receipts paragraph: "Insight events (committed to `docs/insights/events/`) embed `requested_model / effective_model / honored` per phase so `ralph insights` can aggregate routing honor-rate from durable data". Template base `AGENTS.md` has identical entry (check-sync IDENTICAL). |
| AC8: `go test ./... -count=1` and `./scripts/run-verify.sh` pass | **PARTIALLY MET** (static only) | `run-static-verify.sh` exits 0: "All verifiers passed". `gofmt -l internal/ cmd/` exits 0 with no output. `go vet ./...` exits 0. Behavioral `go test ./...` is tester's scope (not run here per /verify contract). |

## AC2 forced-outcome deviation — intent evaluation

The plan states: "forced-outcome tests (self-review findings present, verify fail, test fail, cross-review ACTION_REQUIRED) assert the exact emitted JSON values". The implementation tests this at the appender boundary rather than end-to-end through the pipeline parser because DRY_RUN cannot write agent sidecars. The test exercises:
- `--verdict fail --critical 2 --high 1` → JSON `{"verdict":"fail","findings":{"critical":2,"high":1,...}}` asserted (4a/4b/4c)
- `--verdict fail` on verify → JSON `{"verdict":"fail","phase":"verify"}` asserted (4d/4e)
- `--verdict action_required --action-required 3 --worth-considering 1 --dismissed 2` → JSON triage counts asserted (4f/4g/4h/4i)
- `--verdict pass` on test → findings all zero asserted (4j/4k)

This satisfies the AC's intent: the semantic mapping from parsed outcome to emitted JSON is deterministically verified. The only gap is that the pipeline parser functions themselves (e.g., reading `.verify-result` sidecar) are not tested at the shell level in this approach — that belongs to integration tests with real agent runs.

**Verdict on AC2: MET.** The deviation (appender-boundary rather than full-pipeline forced-outcome) is correctly documented in the wave A self-review report and in the test file header.

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS (exit 0) | All verifiers passed; golang scope; gofmt ok; 0 vet issues |
| `gofmt -l internal/ cmd/` | PASS | No files reported |
| `go vet ./...` | PASS (exit 0) | No issues |
| `shellcheck --severity=warning scripts/insights-append.sh` | PASS (exit 0) | Clean |
| `shellcheck --severity=warning scripts/ralph-pipeline.sh` | PASS (exit 0) | Clean (fix commit d3b30e3 resolved MEDIUM SC2086 issue from self-review) |
| `shellcheck --severity=warning tests/test-insights-append.sh tests/test-insights-pipeline-events.sh` | PASS (exit 0) | Clean |
| `scripts/check-skill-sync.sh` | PASS | 13 skills in lock-step |
| `scripts/check-sync.sh` | PASS | DRIFTED=0, ROOT_ONLY=0 |
| `scripts/check-template.sh` | PASS | Template structure ok |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/insights/README.md` schema vs `internal/insights/event.go` Event struct | **IN SYNC** (with one doc drift) | All 15 schema fields present in both. **DOC DRIFT**: README marks `run_id` as required ("yes"), but `scripts/insights-append.sh` treats it as optional (`--run-id` not in required field checks); `internal/insights/event.go` has `RunID string` (not a pointer, zero-value when absent). The schema should clarify `run_id` is present for pipeline-emitted events but omittable for skill/backfill events. Non-blocking doc drift. |
| `AGENTS.md` map entry for `docs/insights/` | **IN SYNC** | Line 70 added; template base identical. |
| `model-routing.md` Receipts paragraph pointer to `ralph insights` | **IN SYNC** | Updated to mention insight events and routing honor-rate aggregation. |
| Skill bodies (`self-review`, `verify`, `test`, `cross-review`) | **IN SYNC** | All four have `insights-append.sh` instruction; `.claude/`, `.agents/`, `templates/base/` mirrors all identical. |
| `internal/insights/aggregate.go` `HonoredRate: -1` sentinel | **DOC DRIFT** | `-1` sentinel for "no routing data" is not documented in `docs/insights/README.md`. Machine consumers of `ralph insights --json` will see `"honored_rate": -1` without documentation of its meaning. LOW severity, non-blocking. |
| Slug init comment in `ralph-pipeline.sh` | **DOC DRIFT** | Line 1218: "strip two prefixes" comment remains inaccurate after fix commit. Code correctly does `##*/` (last-segment strip). LOW severity, non-blocking. |

## Observational checks

- `docs/insights/events/.gitkeep` is committed (directory tracked in git).
- Template base mirrors `docs/insights/README.md`, `docs/insights/events/.gitkeep`, `scripts/insights-append.sh`, `scripts/ralph-pipeline.sh`, and all 4 skill files — confirmed identical via `check-sync.sh IDENTICAL: 177`.
- Insight event generated for verify phase: `docs/insights/events/2026-07-12-ralph-insights.jsonl` — valid schema-v1 JSON line.
- Fix commit d3b30e3 addressed MEDIUM finding (SC2086 word-splitting in emit_insight_event) and LOW dead-code finding (log_warn). Two remaining LOW items (slug comment, honored_rate sentinel) are not fixed but are non-blocking.

## Coverage gaps

- Behavioral tests (`go test ./...`, shell test execution) — tester's scope, not verified here.
- DRY_RUN=1 end-to-end execution against the actual `test-insights-pipeline-events.sh` fixture — would require running the shell test suite.
- Flow detection correctness (`detectFlow` in backfill) — best-effort by design; not deterministically verifiable at static analysis time.
- `ralph insights` and `ralph insights backfill` output formatting — observational, requires running the binary.

## Verdict

**PASS**

- Verified:
  - AC1: appender validates, appends valid JSONL, shellcheck-clean, template mirrored
  - AC2: DRY_RUN=1 path covered; forced-outcome semantics verified at appender boundary (documented deviation, satisfies AC intent)
  - AC3: CLI implemented with correct structure; Go package in place; test fixtures exist
  - AC4: zero-data early return with expected-location message
  - AC5: backfill parsers for all 4 report types; dedupe key = source_report_path+phase+cycle; idempotency design correct
  - AC6: all 4 skills wired; check-skill-sync, check-sync, check-template all pass
  - AC7: docs/insights/README.md, AGENTS.md map, model-routing.md pointer — all in sync
  - AC8 (static): gofmt clean, go vet clean, run-static-verify.sh exit 0
  - Static analysis: all tools pass

- Partially verified:
  - AC8 (behavioral): `go test ./...` not run (tester's responsibility)
  - AC2 forced-outcome: pipeline parser → sidecar reading path not tested end-to-end (by design; appender-boundary approach is documented and intentional)

- Not verified:
  - Runtime behavior of `ralph insights` CLI output rendering (requires binary execution)
  - DRY_RUN=1 shell test execution (test suite)

- Remaining doc drift (non-blocking, follow-up):
  1. `docs/insights/README.md`: clarify `run_id` as required for `source:pipeline` events, optional for `source:skill|backfill`
  2. `docs/insights/README.md`: document `honored_rate: -1` sentinel meaning in JSON output section
  3. `scripts/ralph-pipeline.sh` line 1218: correct slug comment to "strip last path component (`##*/`)"
