# docs/insights — Post-Implementation Insight Events

This directory stores structured insight event data emitted by post-implementation
skills (`/self-review`, `/verify`, `/test`, `/cross-review`) via
`scripts/insights-append.sh`, and derived by `ralph insights backfill` from existing
Markdown reports. The events are the primary source for `ralph insights`.

Historical events also include those written by the now-retired Ralph Loop
autonomous pipeline (`flow: loop`, `source: pipeline`). `ralph insights` continues
to read these for backward compatibility, but no live writer emits them anymore —
every currently-written event uses `flow: standard`.

## Schema v1

Each event is one JSON line appended to a per-task file under `events/`.

### Fields

| Field | Type | Required | Values / Notes |
|-------|------|----------|----------------|
| `schema` | integer | yes | Always `1` for schema v1. Bumps on breaking change. |
| `ts` | string | yes | ISO8601 UTC timestamp (e.g. `2026-07-13T01:23:45Z`). |
| `run_id` | string | historical for `source:pipeline`; optional for `source:skill\|backfill` | Per-pipeline-invocation identifier (`<ts>-<pid>`). Historically constant across all events in one `ralph-pipeline.sh` run (that writer is retired). Omitted when written by a skill agent or by `ralph insights backfill` (no single run context). |
| `slug` | string | yes | Task slug (matches plan file basename, e.g. `ralph-insights`). |
| `flow` | string | yes | `standard` or `loop`. `loop` is historical only (written by the retired Ralph Loop pipeline). All current skill-emitted events use `standard`. |
| `phase` | string | yes | Phase name: `implement`, `self_review`, `verify`, `test`, `sync_docs`, `cross_review`, `pr`. |
| `cycle` | integer | yes | 1-based outer cycle number. Default: `1` when `--cycle` is omitted from `insights-append.sh`. |
| `verdict` | string | yes | `pass`, `fail`, `complete`, `action_required`, or `n/a`. |
| `findings` | object | yes | `{"critical": N, "high": N, "medium": N, "low": N}`. Use `0` for phases where findings are not applicable. |
| `triage` | object | yes | `{"action_required": N, "worth_considering": N, "dismissed": N}`. Use `0` for non-cross-review phases. |
| `driver` | string | historical for `source:pipeline`; optional for `source:skill\|backfill` | `claude` or `codex`. Only ever emitted by the retired pipeline writer; current skill-emitted events omit it. |
| `requested_model` | string | historical for `source:pipeline`; optional for `source:skill\|backfill` | Model requested for this phase (e.g. `sonnet`, `opus`). Historical field, see `driver`. |
| `effective_model` | string | historical for `source:pipeline`; optional for `source:skill\|backfill` | Model actually used. Equals `requested_model` for the Claude driver; `codex-default` for the Codex driver. Historical field, see `driver`. |
| `honored` | boolean | historical for `source:pipeline`; optional for `source:skill\|backfill` | `true` if `effective_model == requested_model`. `false` for Codex driver (known gap). Historical field, see `driver`. |
| `source` | string | yes | `pipeline` (historical only — written by the now-retired `ralph-pipeline.sh`), `skill` (written by a post-implementation skill agent — the only live writer today), or `backfill` (written by `ralph insights backfill`). |

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

Cross-review triage reports may contain multiple pipeline cycles (each introduced by a `## Cycle N` section heading); `ralph insights backfill` emits one event per cycle found, so a single report file can produce multiple events with different `cycle` values.

### Example line

Current live event (written by the `/self-review` skill):

```json
{"schema":1,"ts":"2026-08-03T01:23:45Z","slug":"ralph-insights","flow":"standard","phase":"self_review","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```

Historical event (written by the now-retired `ralph-pipeline.sh`; still readable):

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

Each task (standard `/work` flow or an org-runtime seat) typically runs from
its own worktree and branch. If all events were appended to a single global
JSONL file, concurrent tasks would produce conflicting appends on merge.
Per-task files make merges trivially clean: each task owns exactly one file,
and no two branches touch the same file.

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

### JSON output sentinel: `honored_rate: -1`

When `ralph insights --json` is used, the aggregate object includes
`honored_rate` in the routing section. A value of `-1` means no routing
data is available (no events with `honored` field present, or zero events
total). Machine consumers must treat `-1` as "no data" and not as a real
honor rate. Example: a fresh repo with only backfill events derived from
Markdown reports will see `"honored_rate": -1` because backfill events
do not carry routing fields.

## Appending events

Events are written by `scripts/insights-append.sh`, called directly by the
post-implementation skills (`/self-review`, `/verify`, `/test`,
`/cross-review`) after each writes its report. `ralph insights backfill`
derives events from existing Markdown reports when live events are missing.

```sh
scripts/insights-append.sh \
  --slug my-task \
  --flow standard \
  --phase self_review \
  --verdict pass \
  --source skill \
  --cycle 1 \
  --critical 0 --high 0 --medium 0 --low 0
```

Required flags: `--slug`, `--flow`, `--phase`, `--verdict`, `--source`.
All others are optional (defaults to 0 for counts, omitted for routing fields).

See `scripts/insights-append.sh --help` for the full interface.

## Org runtime receipts(参照)

`ralph insights` はイベント集計に加えて、org runtime のモデル受領証
(`<state-dir>/model-receipts.jsonl`)を既定で読み、`org_id` × `seat_id` で
tri-state `honored`(true/false/unknown)を集計する Receipts セクションを表示
する。受領証のスキーマと集計契約の正本は `ralph insights --help` と
`.claude/rules/ralph/model-routing.md` の「Org runtime model receipts」節を参照。
このファイルが定義するのはイベント(`events/*.jsonl`)スキーマのみで、受領証
はコミットされないローカル診断データである点に注意。
