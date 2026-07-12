# docs/insights — Ralph Pipeline Insight Events

This directory stores structured pipeline event data emitted by `ralph-pipeline.sh`
and post-implementation skills. The events are the primary source for `ralph insights`.

## Schema v1

Each event is one JSON line appended to a per-task file under `events/`.

### Fields

| Field | Type | Required | Values / Notes |
|-------|------|----------|----------------|
| `schema` | integer | yes | Always `1` for schema v1. Bumps on breaking change. |
| `ts` | string | yes | ISO8601 UTC timestamp (e.g. `2026-07-13T01:23:45Z`). |
| `run_id` | string | yes | Per-pipeline-invocation identifier (`<ts>-<pid>`). Constant across all events in one `ralph-pipeline.sh` run. |
| `slug` | string | yes | Task slug (matches plan file basename, e.g. `ralph-insights`). |
| `flow` | string | yes | `standard` or `loop`. Pipeline-emitted events are always `loop`. Skill-emitted events use `standard`. |
| `phase` | string | yes | Phase name: `implement`, `self_review`, `verify`, `test`, `sync_docs`, `cross_review`, `pr`. |
| `cycle` | integer | yes | 1-based outer cycle number. |
| `verdict` | string | yes | `pass`, `fail`, `complete`, `action_required`, or `n/a`. |
| `findings` | object | yes | `{"critical": N, "high": N, "medium": N, "low": N}`. Use `0` for phases where findings are not applicable. |
| `triage` | object | yes | `{"action_required": N, "worth_considering": N, "dismissed": N}`. Use `0` for non-cross-review phases. |
| `driver` | string | yes | `claude` or `codex`. |
| `requested_model` | string | yes | Model requested for this phase (e.g. `sonnet`, `opus`). |
| `effective_model` | string | yes | Model actually used. Equals `requested_model` for the Claude driver; `codex-default` for the Codex driver. |
| `honored` | boolean | yes | `true` if `effective_model == requested_model`. `false` for Codex driver (known gap). |
| `source` | string | yes | `pipeline` (written by `ralph-pipeline.sh`), `skill` (written by a post-implementation skill agent), or `backfill` (written by `ralph insights backfill`). |

### Optional fields

Optional fields that are absent are omitted entirely (not emitted as `null`).
Go readers must tolerate absent optional fields and unknown extra fields
(forward-compatible with future schema versions).

### Backfill-only fields

Events written by `ralph insights backfill` carry one additional field:

| Field | Type | Notes |
|-------|------|-------|
| `source_report_path` | string | Absolute path to the source report file used for deduplication. Present only when `source == "backfill"`. Used as part of the dedup key (`source_report_path + ":" + phase + ":" + cycle`). |

The `source_report_path` field is not emitted by pipeline or skill writers.
Readers that do not use backfill can safely ignore it (unknown extra fields are tolerated).

### Example line

```json
{"schema":1,"ts":"2026-07-13T01:23:45Z","run_id":"20260713T012345Z-12345","slug":"ralph-insights","flow":"loop","phase":"self_review","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"opus","effective_model":"opus","honored":true,"source":"pipeline"}
```

## File layout

Events for each task are stored in a single file:

```
docs/insights/events/<UTC-date>-<slug>.jsonl
```

Example: `docs/insights/events/2026-07-13-ralph-insights.jsonl`

### Why per-task files?

Ralph Loop slices commit on separate branches and merge sequentially.
If all events were appended to a single global JSONL file, every parallel
slice would produce a conflicting append on merge. Per-task files make
merges trivially clean: each task owns exactly one file, and no two branches
touch the same file.

## Retention

Event files follow the same retention policy family as `docs/reports/`.
The 30-day GC window that applies to reports is a candidate for extension
to cover `docs/insights/events/` as well. Extension of `scripts/gc-artifacts.sh`
to cover this directory is a documented follow-up (not in scope for v1).

## Consuming events

Use the CLI:

```
ralph insights
ralph insights --json
ralph insights backfill [--apply]
```

## Appending events

Events are written by `scripts/insights-append.sh`. The pipeline writes events
automatically. Post-implementation skills may call the script directly.

```sh
scripts/insights-append.sh \
  --slug my-task \
  --flow loop \
  --phase self_review \
  --verdict pass \
  --source pipeline \
  --cycle 1 \
  --critical 0 --high 0 --medium 0 --low 0 \
  --driver claude \
  --requested-model opus \
  --effective-model opus \
  --honored true
```

Required flags: `--slug`, `--flow`, `--phase`, `--verdict`, `--source`.
All others are optional (defaults to 0 for counts, omitted for routing fields).

See `scripts/insights-append.sh --help` for the full interface.
