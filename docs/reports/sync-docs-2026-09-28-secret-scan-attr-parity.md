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
