# Self-review report: ralph-insights

- Date: 2026-07-13
- Plan: docs/plans/active/2026-07-13-ralph-insights.md
- Reviewer: reviewer subagent (self-review, diff quality only)
- Scope: `git diff origin/develop..HEAD` (8 commits: schema+appender, pipeline wiring, Go insights package, CLI command, backfill, skill/doc wiring)

## Evidence reviewed

- `scripts/insights-append.sh` — schema-v1 JSONL appender (+ mirror `templates/base/scripts/insights-append.sh`)
- `scripts/ralph-pipeline.sh` — 7 insight-emit call sites + `emit_insight_event` wrapper + run-id/slug init (+ mirror)
- `internal/insights/{event,reader,receipts,aggregate,backfill}.go` — Go package
- `internal/cli/insights.go`, `internal/cli/root.go` — CLI command wiring
- `.claude/skills/{self-review,verify,test,cross-review}/SKILL.md` + `.agents/` + `templates/base/` mirrors
- `docs/insights/README.md`, `.claude/rules/model-routing.md`, `AGENTS.md`
- Mirror parity: `cmp` on both mirrored scripts → IDENTICAL; git mode `100755` on both sides

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `emit_insight_event` builds `_eie_args` as a single space-joined string then relies on unquoted word-splitting (`SC2086` disabled). `PIPELINE_SLUG` is derived from an arbitrary git branch name (`${_pii_branch##*/}`). A branch containing shell-glob metacharacters or whitespace would corrupt the arg vector or trigger globbing. Not a secret-leak (no command substitution), but a correctness/robustness hazard. | `scripts/ralph-pipeline.sh:166-176` (build + `bash ... ${_eie_args}`), slug at `:1218-1227` | Prefer a positional array (`set --` / `_eie_args=("--slug" "$PIPELINE_SLUG" ...)`) so each field stays a distinct, quoted argument. If keeping the string form, sanitize the slug to `[a-zA-Z0-9._-]`. |
| LOW | debug-code | `log_warn() { log "Warning: $*"; }` is defined in `ralph-pipeline.sh` and referenced only in the adjacent comment; every actual guard uses `log "Warning: ..."` directly, so the function is dead code. (In `ralph-orchestrator.sh` `log_warn` is genuinely used — this is only redundant in the pipeline file.) | `scripts/ralph-pipeline.sh:179-180` (def); guards at `:172` use `log` directly | Remove the unused `log_warn` definition from the pipeline (and its mirror), or actually call it in the guards for consistency. |
| LOW | readability | Slug-init comment says "If branch has issue component (type/NNN/slug), strip two prefixes" but the code is a single `${_pii_branch##*/}` (last-segment strip). Behavior is correct for `type/NNN/slug` → `slug`, but the comment implies conditional two-prefix logic that does not exist. | `scripts/ralph-pipeline.sh:1220-1224` | Reword the comment to describe the actual last-segment strip. |
| LOW | maintainability | `HonoredRate float64` uses `-1` as a "no routing data" sentinel and is serialized verbatim in `--json` output (no `omitempty`, no custom marshaler). Machine consumers of `ralph insights --json` will see `"honored_rate": -1`, a magic value that must be documented to be interpreted. | `internal/insights/aggregate.go:10-12,102-105`; emitted via `enc.Encode(agg)` in `internal/cli/insights.go:79-83` | Document the `-1` sentinel in `docs/insights/README.md` JSON schema, or serialize absence as `null` (pointer / `*float64`). Non-blocking; internal contract. |
| LOW | maintainability | Duplicated codex-vs-claude routing block (`_x_effective_model=...; honored=false/true`) is copy-pasted at all 5 non-cross-review emit sites (implement, self_review, verify, test, sync_docs, pr). Copy-paste is currently correct but drift-prone. | `scripts/ralph-pipeline.sh` implement/self_review/verify/test/sync_docs/pr emit blocks | Optional: extract a small helper that echoes `effective_model honored` from `RALPH_LOOP_DRIVER` + requested model. Not blocking. |
| LOW | maintainability | `detectFlow` (backfill) infers `"loop"` when any of the first 20 lines contains the substring `"loop"` — a report mentioning "Ralph Loop" in prose or a slug like `worktree-gc-loop` would be classified `loop` regardless of actual flow. Flow is explicitly best-effort/omittable, so impact is low. | `internal/insights/backfill.go:161-179` | Acceptable as documented best-effort; consider tightening the token match (`ralph-pipeline` alone) if flow accuracy matters later. |

## Positive notes

- The appender builds JSON exclusively via `jq -cn --arg`/`--argjson` (never string interpolation), which is the correct injection-safe pattern for user-supplied slug/model values — noted explicitly in a comment. Boolean `honored` is passed through `--argjson` so it stays a real JSON bool.
- Input validation in `insights-append.sh` is thorough: required-field checks, enum validation for flow/phase/verdict/source/driver/honored, and non-negative-integer validation for all count fields before they reach `jq`.
- Go error handling is consistent and explicit: `ReadEvents`/`ReadReceipts` treat missing dirs/files as graceful degradation (empty + nil error), corrupt JSONL lines are counted in `SkippedLines` and skipped rather than aborting, and `scanner.Err()` is checked. Deferred `Close()` errors are intentionally ignored via `_ =`.
- Backfill is idempotent via a `source_report_path:phase:cycle` dedup key, with in-run marking so a single `--apply` pass does not double-write.
- Both mirrored scripts are byte-identical (`cmp` exit 0) with matching `100755` git mode; skill wiring landed in `.claude/`, `.agents/`, and `templates/base/` copies together.
- No secrets, no leftover TODO/FIXME/debug prints, no commented-out code in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| _(none — findings above are all fix-in-place, no deferred work introduced)_ | | | | |

## Recommendation

- Merge: yes, with follow-ups. No CRITICAL findings. The MEDIUM arg-splitting robustness item is worth addressing before merge (or accepting consciously, since branch names are typically slug-safe); the LOW items are non-blocking cleanups.
- Follow-ups:
  1. Convert `emit_insight_event` arg assembly to a quoted array or sanitize `PIPELINE_SLUG` (MEDIUM).
  2. Remove the dead `log_warn` definition from `ralph-pipeline.sh` + mirror (LOW).
  3. Fix the misleading two-prefix slug comment (LOW).
  4. Document or `null`-encode the `honored_rate: -1` sentinel in the JSON schema (LOW).
