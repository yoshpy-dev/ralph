# sync-docs report: secret-scan-git-config

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-25-secret-scan-git-config.md` (issue #176)
- Prior reports: `docs/reports/self-review-2026-09-25-secret-scan-git-config.md`
  (Pass, MEDIUM 1 / LOW 6, fixed in Slice D),
  `docs/reports/verify-2026-09-25-secret-scan-git-config.md` (Pass),
  `docs/reports/test-2026-09-25-secret-scan-git-config.md` (Pass)

## Summary

`scripts/secret-scan.sh` (and its `templates/base/scripts/` copy) now pins
the git settings its range and staged scans read under, so a local run
scans the same added lines as CI regardless of local git config, and exits
3 ("could not scan") instead of a false clean whenever git itself fails or
a staged/range blob cannot be read. `scripts/secret-scan-branch.sh` reads a
symlinked `.gitallowed` target byte for byte and propagates the scanner's
own exit 3 in both `--strict` and default mode. This pass fixed the three
documentation-drift items `/verify` flagged, added the two tech-debt rows
the plan's Non-goals call for plus a third for a test-coverage gap `/test`
surfaced, and closed out the plan's own Deviation notes and checklist.

## Documentation drift checked and fixed

- **`.claude/skills/pr/SKILL.md` Step 3** (and its three mirrors:
  `.agents/skills/pr/SKILL.md`, `templates/base/.claude/skills/pr/SKILL.md`,
  `templates/base/.agents/skills/pr/SKILL.md`) — the exit-3 explanation
  covered only `secret-scan-branch.sh`'s own "could not scan" / "nothing to
  scan" cases. Added the third case this PR makes reachable: the scanner
  itself failing to read the range, printed as `scanner failed with exit 3`
  (for example a git error), with the same do-not-push/fix-and-re-run
  instruction. Edited `.claude/skills/pr/SKILL.md`, regenerated
  `.agents/skills/pr/SKILL.md` with `./scripts/sync-skills.sh`, then copied
  both bodies over the two `templates/base/` mirrors — all four are
  `diff`-identical.
- **`docs/quality/quality-gates.md`** (root and `templates/base/` copies) —
  added one clause to the existing `secret-scan.sh --range` bullet: the
  range scan reads the same added lines as CI regardless of local git
  config, and exits 3 (not clean) when git cannot read the range. Kept the
  two copies' pre-existing difference as-is (the template copy folds
  `check-template.sh`/`check-sync.sh` into one line with no issue-number
  history; the root copy keeps them as two lines).
- **`docs/architecture/repo-map.md`** — the scripts bullet's "secret and
  commit safety" group listed `secret-scan.sh` but not
  `secret-scan-branch.sh` (a gap left by #169, flagged by `/verify` as
  pre-existing and out of scope for that PR). Added it in the bullet's
  existing style. No template copy of this file exists
  (`find . -name repo-map.md` returns one hit), so nothing to mirror.

## Files changed in this pass

- **`docs/tech-debt/README.md`** — three new rows:
  - CI-wide gaps the scanner shares with CI (both miss them): a committed
    `-diff`/`binary` attribute or a real binary file (`git log -p` prints
    "Binary files differ", content never scanned); merge commits (`git log
    -p` prints no diff for a merge by default, so a secret introduced while
    resolving a conflict is never scanned); an added line whose content
    starts with `++ ` (the `+++ ` header-skip rule drops it; after this
    PR's ANSI strip, also a colored line starting with an escape sequence
    then `++ `). From the plan's Non-goals.
  - Local-only gaps the scanner cannot pin with `-c` or a flag: a
    `-diff`/`binary` attribute in `.git/info/attributes` (not a git config
    key); a local `diff.<driver>.binary=true` for a driver the committed
    `.gitattributes` names; uncommitted edits to the working tree's own
    `.gitattributes` (plausible, unverified); `secret-scan-branch.sh`'s own
    git calls (`merge-base`, `rev-list`, the `.gitallowed` read) still
    following `refs/replace/*`. From the work-slice Deviation notes'
    out-of-scope findings.
  - Test gaps from `/test`'s report: `HUP`/`INT` signal-trap exit codes are
    unexercised (only `TERM` is tested); `-c log.showSignature=false` and
    `--no-ext-diff` have no discriminating test (the latter is a structural
    no-op for `scan_range`'s exact invocation, not a real gap). Kept as one
    row per the task's own fallback option, since it is a single coherent
    "test coverage" theme distinct from the two behavioral-gap rows above.
  - Did not add a row for `diff.renameLimit` — it was reproduced and fixed
    in Slice D (commit `aecee05`), not deferred.
- **`docs/plans/active/2026-09-25-secret-scan-git-config.md`**
  - Added Deviation-notes bullets for `/verify` (`b76d9da`, PASS, doc-drift
    items recorded), `/test` (`237dc14`, PASS; 21 mutations, 18
    discriminating, the 3 non-discriminating explained; CI parity on 4
    ranges including one byte-identical OLD/NEW diff; a 9-scenario live
    demonstration of the plan's own reproduction table), and this
    `/sync-docs` pass, dated 2026-09-25, matching the style of the existing
    bullets.
  - Annotated the earlier work-slice bullet's `diff.renameLimit`(未確認)
    item with "(その後 self-review の MEDIUM で再現し、Slice D で固定)" so
    the plan no longer implies it is still open.
  - Ticked `Verification artifact created` and `Test artifact created` in
    the Progress checklist. No AC checkbox touched. `grep -c '^# '` = 1
    before and after.
- **`docs/insights/events/2026-09-25-secret-scan-git-config.jsonl`** —
  appended via `./scripts/insights-append.sh` (sync_docs, cycle 1, complete,
  0/0/0/0).

## Mirror/sync verification

```
$ ./scripts/check-skill-sync.sh      # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-sync.sh            # PASS: all files in sync. IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5
$ ./scripts/check-template-purity.sh # PASS: no meta-repo-specific references found in templates.
```

`./scripts/sync-skills.sh` was re-run after editing `pr/SKILL.md`
(`[sync-skills] done: 13 skill(s) mirrored to .agents/skills`); the three
gates above were re-run directly against the current worktree state, not
taken from a prior report.

## Walkthrough

Not applicable to this pass — `git diff main...HEAD --stat -- scripts
tests`: 4 files, 762 insertions(+), 47 deletions(-)
(`scripts/secret-scan.sh`, `scripts/secret-scan-branch.sh`,
`tests/test-secret-scan.sh`, `tests/test-secret-scan-branch.sh`); a
production-code walkthrough decision is `/pr`'s, based on the full
`main...HEAD` diff, not `/sync-docs`'s own doc-only changes.

## Known gaps

- Did not re-run `./scripts/run-verify.sh`, `./scripts/run-test.sh`, or the
  secret-scan test suites; those are `/verify`'s and `/test`'s completed
  job (both cited above, Pass).
- `./scripts/secret-scan-branch.sh --strict` was run after staging this
  pass's own changes, per the handoff's step 9 — see the commit-boundary
  evidence reported back to the orchestrator, not reproduced here.

## Cycle 2

- Date: 2026-09-25
- Plan: `docs/plans/active/2026-09-25-secret-scan-git-config.md` (issue #176)
- Prior reports since cycle 1 (`f6d1b87`): cross-review triage
  (`docs/reports/cross-review-triage-secret-scan-git-config.md`, AR-1,
  ACTION_REQUIRED, user chose fix-and-re-run), self-review cycle 2 (same
  report file, `## Cycle 2`, `750d247`, LOW 4), verify cycle 2 (same
  report file, `## Cycle 2`, `092c5e3`, PASS), test cycle 2 (same report
  file, `## Cycle 2`, `b403b9f`, PASS)

### Correction to the cycle-1 section above

**Do not re-read the cycle-1 section's quality-gates claim as current.**
Cycle-1's `## Files changed in this pass` said the
`docs/quality/quality-gates.md` edit added "the range scan reads the same
lines as CI regardless of local git config." The cycle-2 self-review
(finding 1) found this an overclaim: `diff.<driver>.binary=true` is
itself a local git config setting the scanner cannot pin, so "regardless
of local git config" was not true even at cycle-1 HEAD. Fixed in `443ffb7`
(a code/doc commit, not a `/sync-docs` commit) to "except for a few
local-only attribute sources it cannot pin (listed in the
`scripts/secret-scan.sh` header ...)". The cycle-1 section of this report
is left as written (an accurate record of what that pass did), not edited
to match; this note is the correction.

Two other cycle-1 items were also superseded by code changes after
cycle-1's `/sync-docs` pass, not by a documentation error: `ca8a212`
(cross-review's AR-1 fix) removed the "++ " item from the CI-shared
tech-debt row cycle 1 added, since the state-aware header rule now
detects that case; `443ffb7` (Slice F) rewrote the local-only tech-debt
row cycle 1 added, since `GIT_ATTR_SOURCE=HEAD` newly pins the
uncommitted-`.gitattributes` gap that row had listed as unpinnable.

### Documentation drift checked this cycle

Re-checked every surface cycle 1 touched, plus the doc-facing findings
from self-review/verify/test cycle 2, against the code at `b403b9f`:

- **`.claude/skills/pr/SKILL.md`** (+ 3 mirrors) — no cycle-2 change
  touched `secret-scan-branch.sh`'s exit-code contract; still `diff`-
  identical across all four copies. No edit needed.
- **`docs/quality/quality-gates.md`** (root + `templates/base/`) —
  already corrected by `443ffb7` before this pass started (see above);
  verify cycle 2 confirmed no open drift here. No edit needed.
- **`docs/architecture/repo-map.md`** — unaffected by any cycle-2 code
  change; verify cycle 2 confirmed `secret-scan-branch.sh` is still
  listed. No edit needed.
- **`scripts/secret-scan.sh` header / `templates/base/scripts/
  secret-scan.sh` header** — `443ffb7` already updated the exit-code
  list, the local-only-gaps list, the `--diff` usage line, and the
  `--no-abbrev` comment per self-review cycle 2's findings 2–4; both
  copies are still `cmp`-identical. No edit needed.
- **`docs/tech-debt/README.md`** — the one item that *was* stale: the
  test-gaps row cycle 1 added (rewritten once already by `443ffb7`
  after self-review cycle 2) characterized the SGR-strip gap narrowly
  ("the `while` loop... a single `sub` passes every test"). Test cycle
  2's mutation 3 (replacing the whole anchored strip with the original
  unanchored `gsub`) showed the gap is broader: the `in_hunk` state
  machine, not the strip's shape, is what now prevents header
  misdetection, so no version of the strip is currently discriminated.
  Rewrote the row's SGR clause to say so, and added one sentence noting
  the header's "git 2.41+" claim for `GIT_ATTR_SOURCE` is unverified
  against the git changelog (test cycle 2's gap 4) — worded to state
  the header's claim as fact and flag only the *verification* as open,
  matching the header text itself rather than contradicting it.

### Files changed this cycle

- **`docs/tech-debt/README.md`** — rewrote the test-gaps row's SGR-strip
  clause (see above); no other row touched.
- **`docs/plans/active/2026-09-25-secret-scan-git-config.md`** — added
  Deviation-notes bullets for verify cycle 2 (`092c5e3`, PASS), test
  cycle 2 (`b403b9f`, PASS: 74/108/32, 3 of 4 requested mutations
  discriminating exactly, whole-history added-line-set parity at
  213,915 lines byte-identical, gaps updated), and this sync-docs cycle
  2 pass, dated 2026-09-25, matching the style of the existing bullets.
  No AC or checklist box changed this cycle (both were already ticked
  in cycle 1). `grep -c '^# '` = 1 before and after.
- **`docs/insights/events/2026-09-25-secret-scan-git-config.jsonl`** —
  appended via `./scripts/insights-append.sh` (sync_docs, cycle 2,
  complete, 0/0/0/0).
- **This report** — this `## Cycle 2` section.

### Mirror/sync verification (cycle 2)

```
$ ./scripts/check-skill-sync.sh      # [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-sync.sh            # PASS: all files in sync
$ ./scripts/check-template-purity.sh # PASS: no meta-repo-specific references found in templates.
```

No skill body or `templates/base` mirror was touched this cycle, so
`sync-skills.sh` was not re-run.

### Walkthrough (cycle 2)

`git diff main...HEAD --stat -- scripts tests`: 4 files, 866
insertions(+), 49 deletions(-) (`scripts/secret-scan.sh`,
`scripts/secret-scan-branch.sh`, `tests/test-secret-scan.sh`,
`tests/test-secret-scan-branch.sh`) — up from cycle 1's 762(+)/47(-) by
the AR-1 fix and Slice F. Walkthrough-or-not is `/pr`'s decision, based
on the full `main...HEAD` diff.

### Known gaps (cycle 2)

- Did not re-run `./scripts/run-verify.sh`, `./scripts/run-test.sh`, or
  the secret-scan test suites; `/verify` and `/test` cycle 2 already did
  (both cited above, Pass).
- Did not chase down which git release introduced `GIT_ATTR_SOURCE`; the
  header's "2.41+" claim is carried forward as-is (test cycle 2's own
  gap 4), now also flagged in the tech-debt test-gaps row.
- `./scripts/secret-scan-branch.sh --strict` was run after staging this
  cycle's own changes, per the handoff's step 5 — see the commit-
  boundary evidence reported back to the orchestrator, not reproduced
  here.
