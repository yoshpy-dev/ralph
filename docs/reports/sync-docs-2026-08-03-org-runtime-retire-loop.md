# Sync-docs report: org-runtime-retire-loop

- Date: 2026-08-03
- Plan: `docs/plans/active/2026-08-03-org-runtime-retire-loop.md`
- Maintainer: `doc-maintainer` subagent (`/sync-docs`)
- Input: this pass runs after the plan's own implementation already rewrote
  `AGENTS.md` / `CLAUDE.md` / `README.md` / `.claude/rules/` /
  `docs/quality/` / spec FR-11 as part of Slice 4/5, with the verify report
  (`docs/reports/verify-2026-08-03-org-runtime-retire-loop.md`) recording
  AC-6 as Met. This is the residual sweep pass, scoped to what the plan's
  implementation slices did not already cover: `docs/recipes/`,
  `.claude/skills/*/SKILL.md` prose (beyond the guard test's literal grep
  tokens), `docs/quality/`, `docs/insights/README.md`, and `templates/base/`
  doc mirrors.

## Changes

1. **`docs/insights/README.md` (+ `templates/base/docs/insights/README.md`
   mirror, kept byte-identical)** — this was the one real semantic-drift
   finding. `tests/test-no-loop-references.sh` intentionally excludes
   `docs/insights/` from its literal-token grep (AC-6b: the directory
   legitimately documents historical schema vocabulary, `flow: loop` /
   `source: pipeline`), but the prose itself was still written in **present
   tense** as if the retired `ralph-pipeline.sh` were a live, currently
   running writer:
   - Title was "Ralph Pipeline Insight Events" and the intro said the
     directory "stores structured pipeline event data emitted by
     `ralph-pipeline.sh`" — reworded to describe the actual current writer
     (the post-implementation skills, via `scripts/insights-append.sh`) and
     added an explicit paragraph stating the pipeline writer is retired and
     `flow: loop` / `source: pipeline` are now historical-only, read for
     backward compatibility.
   - The `flow`, `driver`, `requested_model`, `effective_model`, `honored`,
     and `source` field-table rows said "required for `source:pipeline`" —
     reworded to "historical for `source:pipeline`" and added a note that
     current skill-emitted events omit the four model/driver fields
     entirely (confirmed by grepping every `insights-append.sh` call site
     in `.claude/skills/{self-review,verify,test,cross-review}/SKILL.md`:
     none pass `--driver`/`--requested-model`/`--effective-model`/`--honored`,
     all pass only `--flow standard --source skill`).
   - The single example JSON line used `"flow":"loop","source":"pipeline"`
     with driver/model fields — replaced with two examples: a current live
     event (`flow: standard`, `source: skill`, no driver/model fields,
     matching what `/self-review` actually emits) and the old historical
     example kept underneath, explicitly labeled as historical/still-readable.
   - "Why per-task files?" reasoned from "Ralph Loop slices commit on
     separate branches and merge sequentially" — reworded to a Loop-independent
     rationale (any task, standard `/work` or an org-runtime seat, works from
     its own worktree/branch; per-task files avoid merge conflicts either way).
   - "Appending events" said "The pipeline writes events automatically" —
     reworded to name the actual callers (the four post-implementation
     skills) and the invocation example switched from the old
     `--flow loop --source pipeline --driver claude ...` shape to the
     current `--flow standard --source skill` shape skills actually use.
   - Verified via `diff docs/insights/README.md
     templates/base/docs/insights/README.md` (empty — byte-identical) and
     `./scripts/check-sync.sh` (DRIFTED=0) after the edit.

2. **Plan (`docs/plans/active/2026-08-03-org-runtime-retire-loop.md`)** —
   ticked "Sync-docs artifact created" in the Progress checklist, with a
   one-line pointer to the `docs/insights/README.md` fix (the only content
   change this pass made) so a future reader does not need to open this
   report to see what changed.

## Reviewed, no drift found

- **`docs/recipes/`** (`worktrees.md`, `agent-teams.md`, `codex-setup.md`,
  `adding-a-language-pack.md`) — no Ralph Loop references of any kind
  (literal or prose). `worktrees.md`'s Org runtime section and
  `agent-teams.md`'s "single-session loop" language are both accurate to the
  post-retirement state. (`docs/recipes/ralph-loop*.md` was already deleted
  by this plan's Slice 4, per the plan's own Scope section.)
- **`.claude/skills/*/SKILL.md` bodies** — grepped every skill for
  `loop|pipeline|orchestrat|directory plan|manifest.md|slice` prose (not
  just the guard test's literal tokens). All "pipeline" mentions
  (`cross-review`, `pr`, `work`, `sync-docs` SKILL.md bodies) refer to the
  surviving standard-flow post-implementation pipeline
  (`self-review → verify → test → sync-docs → cross-review → pr`), not the
  retired Ralph Loop autonomous pipeline — correct terminology, no change
  needed. `work/SKILL.md`'s directory-plan guard (Step 1) already correctly
  describes Ralph Loop's autonomous execution system as retired, in past
  tense, and redirects to `/plan` or `/org` — this is intentional prose
  documented in the plan's own Implementation notes, not drift.
- **`docs/quality/definition-of-done.md`** and **`docs/quality/quality-gates.md`**
  — both already reflect the two-surface structure (standard `/work` DoD vs.
  a separate "For org runtime tasks" section; "Org runtime gates" table).
  "Inner loop" / "Outer loop" in `quality-gates.md` is generic fast-vs-broad
  CI terminology, unrelated to Ralph Loop — no change needed.
- **`docs/architecture/repo-map.md`** — already rewritten by this plan's
  Slice 5 (per the plan's own Implementation notes); reconfirmed coherent
  with the current skill/script/directory set, no additional drift found.
- **`README.md`** — Quick start, Operating loop, Commands, and Repository
  layout sections all already describe the two-surface (`/work` harness +
  org runtime) structure with no Ralph Loop residue; `ralph insights`
  command description ("Aggregate pipeline insight events") is generic
  enough to remain accurate after the `docs/insights/README.md` rewording.
- **`docs/tech-debt/README.md`** — contains many literal `ralph-pipeline`/
  `ralph-loop` mentions, but all are inside `RESOLVED 2026-08-03` rows/HTML
  comments explicitly documenting what was retired and when — historical
  record, not live guidance. Excluded from the guard test by design; left
  as-is.

## Drift checks

- `./scripts/check-sync.sh` → `IDENTICAL: 143, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 10, KNOWN_DIFF: 3` — PASS, confirms `docs/insights/README.md` and its template mirror stayed byte-identical after the edit.
- `sh tests/test-no-loop-references.sh` → `PASS: no live references to the retired Ralph Loop execution system outside historical documents` — unaffected by this pass's docs/insights/ edit (already excluded by design) and unaffected by the plan-checklist edit (docs/plans/active/ already excluded).
- `./scripts/run-verify.sh` (full scope) → `All verifiers passed.` (`docs/evidence/verify-2026-08-02-210116.log`); includes `check-sync.sh`, `check-skill-sync.sh` (13/13), `check-pipeline-sync.sh`, `tests/test-xreview-helpers.sh` (26/26), gofmt/go vet/staticcheck/golangci-lint clean, `go test ./...` for the changed Go packages all green. No skill bodies were touched this pass, so `sync-skills.sh` / `check-skill-sync.sh` re-generation was not required.

## Not changed (left alone per instruction)

- The two cosmetic LOW findings carried over from the verify report
  (`scripts/xreview-helpers.sh`'s bare `_file`/`_category`/`_summary`/`_n`
  locals lacking the sibling functions' `_ctf_` prefix; `README.md`'s
  directory-tree column misalignment for `config/`/`insights/`) are code/format
  issues, not documentation drift — left untouched per this task's scope.

## Drift clean

Yes. After this pass: `check-sync.sh` is green (docs/insights mirror stayed
in lock-step), `test-no-loop-references.sh` is green, and
`docs/insights/README.md` no longer describes the retired
`ralph-pipeline.sh` as a live writer. No other doc surface (recipes, skill
bodies, quality docs, repo-map, README) had residual drift beyond what the
plan's own implementation slices already fixed.

## Next step

Proceed to `/cross-review` (optional) then `/pr`.

## Cycle 2

- Date: 2026-08-03
- Scope: doc-impact check for the two cross-review fix commits made after
  cycle-1 sync-docs (`a868fca`): `fea2ee6` (status manifest path double-join
  fix via `orgManifestPath`; `pick_reviewer` case-insensitive normalization)
  and `47e2eee` (excludes dry-run seats from `ralph status`'s active/total
  counts; refreshes `IncludeDryRun` doc comments in `status.go` /
  `manifest.go`). No new commits authored this pass.

### Checked, no drift found

- **`ralph status` dry-run semantics** — grepped `README.md`, `AGENTS.md`,
  `CLAUDE.md`, `.claude/rules/*.md`, `docs/quality/*.md`,
  `docs/architecture/repo-map.md`, `docs/recipes/*.md`, and
  `.claude/skills/org/SKILL.md` for any claim about whether dry-run seats
  count toward `ralph status`'s active/total numbers. None of them assert
  anything about it either way (README's one-line command table entry and
  `status.go`'s own `Long` cobra help text describe roster display and
  active counts in general terms, not the dry-run-inclusion rule), so
  `47e2eee`'s behavior change (dry-run seats now render as rows via
  `IncludeDryRun: true` but are excluded from `ActiveCount`/`TotalCount` via
  the new `buildRealSeatCounts`) has no doc surface to contradict. Confirmed
  `ralph org status --all`'s own dry-run-inclusion behavior
  (`newOrgStatusCmd` in `internal/cli/org.go`) is a separate code path
  untouched by either fix commit, so `.claude/skills/org/SKILL.md`'s
  "`--all` で dry-run 座席も含める" line (line 47) still describes the right
  command correctly.
- **Cross-review SKILL.md accuracy re: case-insensitive `RALPH_PRIMARY_CLI` +
  `pick_reviewer`** — `.claude/skills/cross-review/SKILL.md` (and its
  `.agents/skills/` mirror) already stated, before `fea2ee6`, that
  `RALPH_PRIMARY_CLI` "Accepts `claude` or `codex` (case-insensitive)"
  (line 38) and that Step 2.b calls `pick_reviewer` with that driver value
  (line 44). `fea2ee6` was a code fix bringing `pick_reviewer`'s
  implementation in line with prose the skill already documented (per the
  cycle-2 verify report's AR-2 entry) — not a doc change, and no stale text
  remains: both mirrors still read exactly as they did in cycle 1, and
  `diff` between the two confirms they stay byte-identical.
- **`docs/recipes/`** — re-grepped `worktrees.md`, `agent-teams.md`,
  `codex-setup.md`, `adding-a-language-pack.md` for `status`/`dry-run`/
  `manifest` mentions; none reference the two fixed code paths.
- **Org docs** (`docs/specs/2026-08-01-org-runtime.md`,
  `.claude/rules/agent-messaging.md`, `docs/quality/quality-gates.md`,
  `.claude/rules/git-commit-strategy.md`) — all reference
  `.harness/state/org/manifest.jsonl` generically (audit trail, saga
  record); none describe the manifest *path construction* that
  `orgManifestPath` fixed, so no drift.

### Drift checks

- `./scripts/check-sync.sh` → `IDENTICAL: 143, DRIFTED: 0` (unchanged from
  cycle 1) — no doc mirrors touched this pass.
- `sh tests/test-no-loop-references.sh` → PASS, unaffected (no loop
  references introduced or removed by the two fix commits).

### Drift clean

Yes. Both fix commits (`fea2ee6`, `47e2eee`) are internal implementation
corrections that bring code behavior in line with what the docs already
described (case-insensitive `RALPH_PRIMARY_CLI`) or fix a bug in an
undocumented-either-way area (`ralph status` dry-run row-vs-count split, and
the manifest path double-join). No committed doc, rule, skill, or recipe
made a claim these commits contradict. No file changes required this cycle.

### Next step

Proceed to `/cross-review` (optional) then `/pr`.
