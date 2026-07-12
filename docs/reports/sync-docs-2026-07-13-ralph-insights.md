# Sync-docs report: ralph-insights

- Date: 2026-07-13
- Plan: `docs/plans/active/2026-07-13-ralph-insights.md`
- Branch: `feat/ralph-insights`
- Author: `doc-maintainer` subagent
- Verify evidence: `docs/evidence/verify-2026-07-12-181637.log`

## Scope of this diff

The ralph-insights feature adds `ralph insights` command, insight event schema v1,
`scripts/insights-append.sh`, pipeline wiring, Go aggregation package, backfill
parser, skill instruction lines, and `docs/insights/README.md` schema doc.

This sync-docs pass addresses three known drift items from the verify report
(`docs/reports/verify-2026-07-13-ralph-insights.md`) plus a full drift sweep.

## Drift items fixed

### 1. `docs/insights/README.md`: `run_id` required-vs-optional clarification

**Was:** `run_id` marked "Required: yes" with no qualification.

**Now:** "Required: required for `source:pipeline`; optional for
`source:skill|backfill`" with an explanatory note that skill agents and
`ralph insights backfill` have no single run context and therefore omit the
field.

**Why:** The appender treats `--run-id` as optional; `internal/insights/event.go`
has `RunID string` (zero-value when absent). The schema doc must reflect the
actual contract.

### 2. `docs/insights/README.md`: `honored_rate: -1` sentinel documented

**Added:** New "JSON output sentinel: `honored_rate: -1`" subsection under
"Consuming events" explaining that `-1` means no routing data is available
(no events carry the `honored` field, or zero events total). Machine consumers
must treat `-1` as "no data" rather than a real honor rate. Example given:
fresh repos with only backfill events will see `-1`.

**Why:** `internal/insights/aggregate.go` emits `HonoredRate: -1` as a sentinel
for the no-routing-data case. Without documentation, machine consumers of
`ralph insights --json` have no contract to program against.

### 3. `scripts/ralph-pipeline.sh` line 1218: comment corrected

**Was:** "Strip type prefix (everything up to and including the first '/'). / If
branch has issue component (type/NNN/slug), strip two prefixes."

**Now:** "Strip all path components except the last one (##*/ strips everything /
up to and including the last '/'), giving the final slug segment."

**Why:** `##*/` is a greedy strip of everything up to and including the *last*
`/` — it strips one path operation, not two prefixes. The old comment described
the wrong semantics and would confuse readers trying to understand slug
derivation.

No code change — comment only.

## Files updated in this sync pass

| File | Change | Why |
| --- | --- | --- |
| `docs/insights/README.md` | `run_id` Required column clarified; `honored_rate: -1` sentinel section added | Fix verify drift items 1 and 2 |
| `templates/base/docs/insights/README.md` | Byte-identical mirror of `docs/insights/README.md` | Mirror rule: template twin must stay in sync |
| `scripts/ralph-pipeline.sh` | Line 1218 comment corrected ("strip two prefixes" → accurate description of `##*/`) | Fix verify drift item 3 |
| `templates/base/scripts/ralph-pipeline.sh` | Byte-identical mirror of `scripts/ralph-pipeline.sh` | Mirror rule: template twin must stay in sync |
| `README.md` | Two rows added to the Commands table: `ralph insights [--json]` and `ralph insights backfill [--apply]` | README enumerates all CLI commands; `ralph insights` is a new sibling command alongside `ralph doctor`, `ralph status`, etc. |
| `scripts/check-sync.sh` | `docs/insights/events/` prefix added to `ROOT_ONLY_EXCLUSIONS` | Per-task event files (e.g. `2026-07-12-ralph-insights.jsonl`) are runtime artifacts, not template content; only `.gitkeep` belongs in `templates/base/docs/insights/events/` |
| `docs/plans/active/2026-07-13-ralph-insights.md` | Progress checklist: Review/Verification/Test artifact boxes ticked with report paths | Self-review, verify, and test reports all exist and passed |

## Files checked and left unchanged

| Doc / contract | Result | Evidence |
| --- | --- | --- |
| `AGENTS.md` — `docs/insights/` map entry | Already in sync | Line 70 added by implementation (Slice 6): "`docs/insights/` — committed insight events". Template base identical (check-sync IDENTICAL). |
| `AGENTS.md` — `internal/cli/` subcommand list | Already in sync | Verified: `internal/cli/` entry covers "cobra subcommands (init, upgrade, run, status, retry, abort, doctor, pack, version)" — this is a repo-map entry for the package, not a per-command enumeration; no change needed here. |
| `CLAUDE.md` | No change needed | Scoped to skill orchestration; does not enumerate individual commands. |
| `.claude/rules/model-routing.md` | Already in sync | Receipts paragraph already updated in Slice 6 to point to `ralph insights` for routing honor-rate aggregation. |
| `docs/quality/definition-of-done.md` | No change needed | DoD is pipeline-shaped; `ralph insights` is a new CLI command, not a pipeline step or quality gate. |
| `docs/recipes/` | No change needed | No existing recipe references are now wrong. A future `docs/recipes/insights.md` recipe is a follow-up candidate, not required for v1. |
| `.claude/skills/{self-review,verify,test,cross-review}/SKILL.md` | Already in sync | All four skill bodies have `insights-append.sh` instruction lines (Slice 6); check-skill-sync exits 0 (13 skills in lock-step). |
| `templates/base/AGENTS.md` | Already identical | check-sync IDENTICAL: 177. |

## Template mirror status

| File | Status |
| --- | --- |
| `docs/insights/README.md` ↔ `templates/base/docs/insights/README.md` | IDENTICAL (cp applied) |
| `scripts/ralph-pipeline.sh` ↔ `templates/base/scripts/ralph-pipeline.sh` | IDENTICAL (cp applied) |
| `README.md` | ROOT_ONLY_EXCLUSIONS — not mirrored by design |
| `scripts/check-sync.sh` | ROOT_ONLY_EXCLUSIONS — not mirrored by design |

## Verification results

| Check | Result |
| --- | --- |
| `./scripts/check-sync.sh` | PASS — IDENTICAL: 177, DRIFTED: 0, ROOT_ONLY: 0, KNOWN_DIFF: 3 |
| `./scripts/check-skill-sync.sh` | PASS — 13 skills in lock-step |
| `./scripts/run-verify.sh` | PASS — all verifiers passed (gofmt ok, go vet clean, go test all ok, shellcheck clean) |

Evidence: `docs/evidence/verify-2026-07-12-181637.log`
