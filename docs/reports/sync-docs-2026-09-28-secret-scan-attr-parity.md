# Sync-docs report: secret-scan-attr-parity

- Date: 2026-09-28
- Plan: `docs/plans/active/2026-09-27-secret-scan-attr-parity.md` (issue #181)
- Agent: `doc-maintainer` subagent (cycle 1)
- Branch: `fix/secret-scan-attr-parity`, base HEAD `b8e46fc` (base `main` at `03f3e8a`)

## Checked

- `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` header comments
  against the shipped `scan_range`, `check_info_attributes`, and
  `use_merge_attributes` behavior.
- `docs/quality/quality-gates.md` and its `templates/base/` copy — the
  secret-scan bullet's description of `RALPH_SECRET_SCAN_ATTR_SOURCE`,
  `.git/info/attributes` refusal location, and the merge-attributes fallback.
- `.claude/skills/pr/SKILL.md` and its three mirrors
  (`.agents/skills/pr/SKILL.md`, `templates/base/.claude/skills/pr/SKILL.md`,
  `templates/base/.agents/skills/pr/SKILL.md`) — Step 3's exit-3 causes,
  four-way `diff`.
- `docs/tech-debt/README.md` — every row mentioning `secret-scan`,
  `.gitattributes`, `GIT_ATTR`, or a merge guard, plus every
  `docs/plans/active/*.md` path referenced anywhere in the file, checked
  against each plan's actual archive status.
- `README.md`, `docs/quality/definition-of-done.md`, `docs/recipes/`,
  `AGENTS.md`, `docs/architecture/repo-map.md` — grepped for
  `secret-scan`/`gitattributes`/`GIT_ATTR`; the only hits are inventory
  bullets in `AGENTS.md` and `docs/architecture/repo-map.md` that just name
  the two scripts, with no behavior claim to drift.
- `docs/plans/active/2026-09-27-secret-scan-attr-parity.md` — Progress
  checklist and Deviation notes against the actual commit history and the
  test report's own findings.

## Stale

- `docs/quality/quality-gates.md` (root and template copy) already matched
  the shipped `f2ceeca` behavior exactly — no change needed.
- All four `/pr` `SKILL.md` copies already matched each other and the
  shipped exit-3 semantics (byte-identical `diff`, confirmed).
- `docs/tech-debt/README.md:137`'s trigger-column citation
  `docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md` was stale:
  that plan was archived (PR #167 merged) and now lives at
  `docs/plans/archive/2026-09-25-doctor-shell-alias-rc-types.md`.
- `docs/tech-debt/README.md` had no row for this cycle's `/test` findings
  (the branch test file's missing real-git-version `SKIP` gate and four
  untested error branches) — a missing entry, not a wording drift.
- The plan's Deviation notes had no `test` or `sync-docs` bullet yet, and
  `Test artifact created` was unchecked even though the test report already
  exists at `b8e46fc`.

Other `docs/plans/active/*.md` paths already referenced in
`docs/tech-debt/README.md` were spot-checked against their current archive
status; several are also stale (plans since archived but still cited under
`active/`), but only the one row named for this task was in scope to fix —
left as-is, see Left.

## Changed

- `docs/tech-debt/README.md`:
  - Fixed the stale path on the `checkShellAliases`-adjacent
    `tests/test-ralph-dispatch.sh` row: `docs/plans/active/2026-09-25-doctor-shell-alias-rc-types.md`
    → `docs/plans/archive/2026-09-25-doctor-shell-alias-rc-types.md`.
  - Appended one new row (existing five-column format) for the `/test`
    cycle-1 gaps: `tests/test-secret-scan-branch.sh` has no real-git-version
    `SKIP` gate (19 assertions fail under a real git 2.40.4 with a
    mismatched-but-production-correct message, unlike its sibling
    `test-secret-scan.sh`, which has one), plus four untested error branches
    in `scripts/secret-scan-branch.sh`: `use_merge_attributes`'s
    `drivers_rc`=other, the outer `attr_diff_rc`=other,
    `check_info_attributes`'s `grep`-exits-other-than-0/1, and
    `git_version_at_least`'s malformed/empty-version-string arm.
- `docs/plans/active/2026-09-27-secret-scan-attr-parity.md`:
  - Added a Deviation notes bullet for test cycle 1 (`b8e46fc`, pass; 10
    mutations all discriminating; live comparison against `main`'s pre-fix
    scanner in 10 scenarios/15 invocations; whole-history parity 394,454 raw
    lines identical; Docker git 2.40.4/2.43.7; the gaps above, code
    unchanged).
  - Added a Deviation notes bullet for this sync-docs pass (drift-check
    result, the tech-debt path fix, the new tech-debt row).
  - Ticked `Test artifact created` in the Progress checklist (the report
    already exists); AC checkboxes were left untouched.

## Left

- Other stale `docs/plans/active/*.md` citations in `docs/tech-debt/README.md`
  (several rows point at plans now under `docs/plans/archive/`) — out of
  this task's scope, which named only the
  `2026-09-25-doctor-shell-alias-rc-types.md` row; a repo-wide sweep would
  materially widen this plan's diff.
- `docs/tech-debt/README.md`'s pre-existing rows for #176/#181 (lines
  referencing blind spots, replace-ref gaps, and CI merge-tree parity) —
  accurate as written for the shipped behavior; the new row is additive, not
  a rewrite.
- No README, spec, or definition-of-done changes were needed; the grep in
  Checked found nothing describing the scanner's attribute behavior outside
  the files already named in the plan's own doc list.

## Diff size (scripts, tests)

```
$ git diff main...HEAD --stat -- scripts tests
 scripts/secret-scan-branch.sh    | 180 +++++++++++++++++-
 scripts/secret-scan.sh           | 129 ++++++++++---
 tests/test-secret-scan-branch.sh | 397 ++++++++++++++++++++++++++++++++++++++-
 tests/test-secret-scan.sh        | 217 ++++++++++++++++++++-
 4 files changed, 888 insertions(+), 35 deletions(-)
```

## Gates

- `./scripts/check-skill-sync.sh` — PASS
- `./scripts/check-sync.sh` — PASS
- `./scripts/check-template-purity.sh` — PASS
- `./scripts/secret-scan.sh --staged` — clean (run after staging)
- `./scripts/secret-scan-branch.sh --strict` — clean

## Verdict

Documentation is in sync with the shipped code. One stale archived-plan
path fixed, one tech-debt row added for the test cycle's coverage gaps, and
the plan's checklist and deviation notes brought current. No behavior or
wording changes were needed in `scripts/secret-scan.sh`'s header,
`quality-gates.md`, or the four `/pr` skill mirrors — they already matched
the shipped code.

## Cycle 2

- Date: 2026-09-28
- Base HEAD for this cycle: `f0463af` (cycle-1 sync-docs commit was
  `af29652`; commits since: cross-review triage/decision, Slice E
  (`fe4f383`, merge-tree isolation + real-git-version test gate),
  self-review cycle 2 (`e8a3070`), Slice F (`8443a03`, C2-M1..C2-L5
  fixes), verify cycle 2, test cycle 2).

### Checked

- `scripts/secret-scan-branch.sh` header comment (the merge-tree
  paragraph, `attributes_unguaranteed`'s doc comment, and
  `use_merge_attributes`'s doc comment) against Slice E/F's shipped
  behavior: the 8 merge-tree isolation pins (`GIT_ATTR_SOURCE` set to
  HEAD's commit, `GIT_ATTR_NOSYSTEM=1`, `core.attributesFile=/dev/null`,
  `attr.tree=`, `merge.renormalize=false`, `merge.renames=true`,
  `merge.directoryRenames=conflict`, `merge.renameLimit=7000`) and the
  scoped remedy (only passed for the local-merge-driver, old-git,
  no-`--write-tree`, and merge-tree-failure causes — confirmed by reading
  every `attributes_unguaranteed` call site at
  `scripts/secret-scan-branch.sh:453,458,464,469,473,496,504,508,524`).
- `docs/quality/quality-gates.md` and its `templates/base/` copy —
  four-way comparison of the secret-scan bullet against each other and
  against C2-L5's "on git 2.41 or later" fix.
- `.claude/skills/pr/SKILL.md` and its three mirrors — byte-identical
  `diff` (all three empty) plus `./scripts/check-skill-sync.sh`.
- `docs/tech-debt/README.md`'s two rows Slice F touched: the "Local-only
  ways" row (line 139, the C2-L1 early-warning-loss sentence and its
  cited AC-3 decision) and the "CI's `pull_request` checkout" row (line
  140, the cycle-1 cross-review pointer pinned to `reviewed HEAD af29652`
  / `read it at d884024`, and the new GitHub-server-side-merge clause);
  every file path in both rows' evidence cells checked for existence.
- Whether the test-gap row Slice E removed (the one this report's Cycle 1
  section added, about the missing real-git-version `SKIP` gate and four
  untested branches) is still referenced anywhere as if it existed:
  grepped the repo for its distinguishing phrases.
- `docs/recipes/` and `README.md` — grepped for
  `secret-scan`/`gitattributes`/`GIT_ATTR`/`merge-tree`; no hits.
- `AGENTS.md`'s repo-map bullet — confirmed `secret-scan-branch.sh`'s
  listed role (`branch-history secret scan`) is still accurate; no script
  was added, removed, or repurposed this cycle.
- Repo-wide grep for two stale-phrasing patterns named in the task: any
  doc still saying merge-tree runs with local merge config, or that the
  remedy is unconditionally/always appended to every `attributes_unguaranteed`
  reason.

### Drift found

None. Every location above already reflects Slice E/F's shipped behavior:

- The header's merge-tree paragraph already states isolation ("no local
  attributes file, renormalization, or rename setting changes how
  `.gitattributes` merges") and the "Either way the reason says how to
  get past it" sentence is already scoped to exactly the four
  merge-cannot-be-computed causes that pass `$merge_remedy` in code — it
  does not claim the temp-file, HEAD-unresolvable, or outer-`git diff`
  failure paths get a remedy, matching `attributes_unguaranteed`'s own
  updated doc comment ("A missing temporary file ..., an unresolvable
  HEAD, and a failed `.gitattributes` comparison give the reason only").
- Both `quality-gates.md` copies already carry "on git 2.41 or later" and
  remain in sync with each other (their one difference is the
  pre-existing, intentional `docs/tech-debt/README.md` mention that only
  the root copy carries, unrelated to this plan).
- All four `/pr` `SKILL.md` copies remain byte-identical; Step 3's prose
  is high-level enough (names the four merge-cannot-be-computed causes,
  phrases the remedy as one example fix rather than a universal
  guarantee) that Slice E/F required no edit there.
- `docs/tech-debt/README.md:139` already carries the C2-L1 sentence
  verbatim ("`secret-scan-branch.sh`'s default mode (`run-verify.sh`'s
  early warning) prints `cannot scan` and exits 0 without scanning, where
  main's script scanned with that file ... `#181's AC-3 decision`").
  `docs/tech-debt/README.md:140` already carries the C2-L5 fix: the
  cross-review pointer is pinned to a specific reviewed HEAD and commit
  rather than citing the rewritable per-slug file bare, and a new clause
  documents that GitHub's server-side merge attributes are unspecified.
- The removed test-gap row is not referenced anywhere as if it still
  existed — its only remaining mentions are historical, past-tense
  records (this report's own Cycle 1 section, the plan's cycle-1
  Deviation notes bullet, and the cross-review triage row that
  originally asked for it to be dropped once closed).
- One evidence-cell path does not yet resolve:
  `docs/tech-debt/README.md`'s rows 139/140 cite
  `docs/plans/archive/2026-09-27-secret-scan-attr-parity.md`, but the
  plan is still at `docs/plans/active/2026-09-27-secret-scan-attr-parity.md`
  (confirmed with `ls`) since `/pr` has not archived it yet. This is a
  pre-existing forward-reference (present since cycle 1's own audit, not
  introduced this cycle) that becomes accurate once `/pr` archives the
  plan; the same pattern already appears in other rows of this file for
  earlier plans. Not fixed, since `active/` would be equally wrong for
  the few hours until archival and `archive/` is what a post-merge reader
  will see.
- `AGENTS.md` and `docs/architecture/repo-map.md`'s inventory bullets are
  unchanged and still accurate (name-only, no behavior claim).

### Changed

None — no doc edits this cycle.

### Gates

- `./scripts/check-skill-sync.sh` — PASS (13 skills in lock-step)
- `./scripts/check-sync.sh` — PASS (0 DRIFTED)
- `./scripts/check-template-purity.sh` — PASS
- `./scripts/secret-scan.sh --staged` — clean (this report + insight
  event only; no other files staged)
- `./scripts/secret-scan-branch.sh --strict` — clean (see line below)

### Diff size (scripts, tests) at this cycle's HEAD

```
$ git diff main...HEAD --stat -- scripts tests
 scripts/secret-scan-branch.sh    | 233 +++++++++++-
 scripts/secret-scan.sh           | 129 +++++--
 tests/test-secret-scan-branch.sh | 785 ++++++++++++++++++++++++++++++++++++++-
 tests/test-secret-scan.sh        | 217 ++++++++++-
 4 files changed, 1328 insertions(+), 36 deletions(-)
```

### Verdict

No drift. Every doc surface the task named already reflects Slice E/F's
merge-tree isolation, scoped remedy, and the cycle-2 self-review's five
LOW/one MEDIUM fixes. No edits made this cycle.
