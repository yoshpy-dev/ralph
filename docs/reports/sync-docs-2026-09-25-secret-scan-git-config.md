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
