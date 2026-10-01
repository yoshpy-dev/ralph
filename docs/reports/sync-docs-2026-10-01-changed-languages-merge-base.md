# sync-docs report: changed-languages-merge-base

## Cycle 1

- Date: 2026-10-02 (the file keeps the plan's 2026-10-01 prefix; the run crossed local midnight)
- Plan: `docs/plans/active/2026-10-01-changed-languages-merge-base.md` (issue #190)
- Pipeline cycle: 1 of 2. Delta: `main` `a3adc13b` to branch HEAD `2839d16e`.
- Prior reports (the commits that added them): `docs/reports/self-review-2026-10-01-changed-languages-merge-base.md`
  (`fff03042`, addendum `6a554009`; Pass, recommend merge),
  `docs/reports/verify-2026-10-01-changed-languages-merge-base.md` (`2da7a274`; Pass, LOW 1 = V-1),
  `docs/reports/test-2026-10-01-changed-languages-merge-base.md` (`b77fbd81`, addendum `52e0142b`; Pass)

### Summary

The shipped behavior is documented correctly in `docs/quality/quality-gates.md`
and the detector header; no other surface describes how the changed scope
picks its diff base. The one gap was V-1 from the verify report: the cases
where the base resolves to HEAD itself were not written down anywhere. One
sentence was added to the changed-scope paragraph (root and template) and one
to the detector header comment (root and template). No code changed.

### Delta reviewed

`git diff main...HEAD`, documentation-relevant parts:

- `scripts/detect-changed-languages.sh` and the byte-identical
  `templates/base/scripts/detect-changed-languages.sh`: with `RALPH_VERIFY_BASE`
  empty, the diff base is the first existing full ref among origin's default
  branch (`refs/remotes/origin/HEAD` target under `refs/remotes/origin/`, then
  `origin/main`, `origin/master`), the same three for the current branch's
  tracked remote (a configured remote name only), then local `main`, `master`.
  `@{upstream}` is never consulted. When `branch.<b>.remote` is set (a remote
  name, `.`, or a URL), no default ref was found, and the local fallback is the
  current branch itself, the result is `scope=full reason=no_remote_default:<remote|.|url>`
  (a URL is never printed).
- `docs/quality/quality-gates.md` and `templates/base/docs/quality/quality-gates.md`:
  the changed-scope paragraph already names the merge-base and the base order
  (added in Slice A, reworded in Slice B).
- `tests/test-detect-changed-languages.sh`, `tests/test-run-verify-scope.sh`:
  test-only; not touched in this pass.

### Changes made

| File | Lines | Change |
|------|-------|--------|
| `docs/quality/quality-gates.md` | 46-50 | One sentence appended to the changed-scope bullet: with no remote default branch and no tracked remote, HEAD on local main/master or detached at its tip is its own base, so only uncommitted and untracked files count and the result can be `no_changes`; `RALPH_VERIFY_BASE` or `RALPH_VERIFY_SCOPE=full` covers committed changes. |
| `templates/base/docs/quality/quality-gates.md` | 46-50 | Identical edit. Lines 39-50 of the two files are identical (`diff` on the paragraph); the files as a whole keep the existing `check-sync.sh` KNOWN_DIFF. |
| `scripts/detect-changed-languages.sh` | 32-37 | Comment only: same limitation as one sentence at the end of the base-selection header. |
| `templates/base/scripts/detect-changed-languages.sh` | 32-37 | Copied from the root file with `cp -p`; `cmp` reports identical, mode stays 755. |

The wording was checked against the code before it was written, with scratch
probes in a hermetic repo (HOME and global git config pinned):

| Fixture | Result |
|---------|--------|
| no remote, on `main`, committed `go.mod`, clean tree | `reason=no_changes languages=` |
| same repo, HEAD detached at the `main` tip | `reason=no_changes languages=` |
| same detached HEAD with `RALPH_VERIFY_BASE=HEAD~1` | `reason=changed_languages languages=golang` |
| on `main`, committed `go.mod` plus an untracked `s.py` | `languages=python` (only the uncommitted file counts) |
| feature branch off local `main`, no remote, committed `go.mod` | `languages=golang` (local `main` is a different commit, so it stays a real base) |

Conditions the sentence deliberately keeps: "no tracked remote" excludes the
case where `branch.<b>.remote` is set, because that case already returns
`full` (`no_remote_default:*`) and is documented in the header's earlier
sentence; the quality-gates sentence does not claim `no_changes` for it.

### Surfaces checked for drift (none found)

Grepped for `RALPH_VERIFY_BASE`, `@{upstream}`, `merge-base`, `upstream`,
`changed scope`, `changed-language`, `detect-changed-languages`,
`RALPH_VERIFY_SCOPE`, `no_changes`, `docs_only` across `README.md`,
`AGENTS.md`, `docs/architecture/`, `docs/recipes/`, `docs/specs/`,
`docs/quality/`, `docs/tech-debt/README.md`, `.claude/agents/`,
`.claude/skills/{verify,test}`, `.agents/skills/{verify,test}`,
`.codex/agents/`, and the `templates/base/` copies of each.

- `README.md`, `AGENTS.md`, `docs/specs/`: no mention of the diff base,
  `RALPH_VERIFY_BASE`, or changed-scope selection.
- `docs/architecture/repo-map.md:61`: lists `detect-changed-languages.sh` in
  the `scripts/` inventory with no description of internals. Still accurate.
- `docs/recipes/adding-a-language-pack.md:39,50,79` (and the template at
  `:53`): how to add detector patterns and keep the pair in sync. Does not
  mention the base. Still accurate.
- `.claude/agents/verifier.md:21`, `.claude/agents/tester.md:19-20`,
  `.claude/skills/verify/SKILL.md:13`, `.claude/skills/test/SKILL.md:11`
  (the `.agents/skills/`, `.codex/agents/`, and `templates/base/` copies
  carry the same lines): "changed-language scope by default" and when to use
  `RALPH_VERIFY_SCOPE=full`. They never state the base. Still accurate.
- `docs/quality/definition-of-done.md:12-13` (and template): "changed-language
  scope by default". Still accurate.
- `docs/quality/quality-gates.md:26-27,39-46`: lines 26-27 and 39-42 are
  unchanged and accurate; 43-46 already matched the code (verify report,
  "文書のずれ" table); 46-50 is this pass's addition.
- `docs/tech-debt/README.md:56-57`: a resolved row about `detect_base_branch`
  in the cross-review helper (`HEAD@{upstream}` at the cross-review sites).
  A different code path from this detector. No change.
- `scripts/run-verify.sh:63-98` reads `reason` for display only; the new
  `no_remote_default:*` values need no handling there (verify report).

### Tech-debt decision

No row added. V-1 is documented behavior, unchanged from before this branch
(old and new detector both return `no_changes` for the remote-less `main`
case), and it is now written down in the two places where an operator would
look: the changed-scope bullet in `quality-gates.md` and the detector header.
The tester's remaining gaps G-5 and G-7 were closed by Slice E (`e35840d5`,
cases 34 and 35 in the detector suite); G-6 is cosmetic (reason label only)
and the test report does not count it. `docs/tech-debt/README.md` has no
`detect-changed-languages` row to update.

### Files changed in this pass

- `docs/quality/quality-gates.md`
- `templates/base/docs/quality/quality-gates.md`
- `scripts/detect-changed-languages.sh`
- `templates/base/scripts/detect-changed-languages.sh`
- `docs/reports/sync-docs-2026-10-01-changed-languages-merge-base.md` (this report)
- `docs/insights/events/2026-10-01-changed-languages-merge-base.jsonl`
  (sync_docs, cycle 1, verdict pass, via `./scripts/insights-append.sh`)

The plan and the other pipeline reports were not edited (the orchestrator
owns the plan's Deviation notes). Tests, `.github/workflows`, `.gitallowed`,
scanner patterns, and `.codex/config.toml` were not touched.

### Checks run

```
$ sh -n scripts/detect-changed-languages.sh            # ok
$ dash -n scripts/detect-changed-languages.sh          # ok
$ shellcheck -S warning scripts/detect-changed-languages.sh   # ok
$ cmp scripts/detect-changed-languages.sh templates/base/scripts/detect-changed-languages.sh   # identical
$ sh tests/test-detect-changed-languages.sh            # PASS 83 / 83
$ sh tests/test-run-verify-scope.sh                    # PASS 19 / 19
$ dash tests/test-detect-changed-languages.sh          # PASS 83 / 83
$ dash tests/test-run-verify-scope.sh                  # PASS 19 / 19
$ ./scripts/check-sync.sh              # rc 0; IDENTICAL: 159, DRIFTED: 0, ROOT_ONLY: 0, TEMPLATE_ONLY: 11, KNOWN_DIFF: 5
$ ./scripts/check-skill-sync.sh        # rc 0; [ok] check-skill-sync: 13 skill(s) in lock-step
$ ./scripts/check-template-purity.sh   # rc 0; PASS: no meta-repo-specific references found in templates.
```

The detector's comment edit is ASCII only. `sync-skills.sh` was not run: no
skill body changed, so `.agents/skills/` and `templates/base/` skill mirrors
are untouched.

### Walkthrough size

`git diff main --shortstat` (working tree plus branch commits, which equals
`main...HEAD` after this pass's commit) excluding `docs/plans/`,
`docs/reports/`, `docs/insights/`: 6 files, 751 insertions(+), 22
deletions(-), 773 changed lines. `tests/test-detect-changed-languages.sh`
(470) and `tests/test-run-verify-scope.sh` (93) account for 563 of the
insertions; the two detector copies are 86 each. This is over the 500-line
rule of thumb; no walkthrough written here, `/pr` decides.

### Known gaps

- `./scripts/run-verify.sh` and `./scripts/run-test.sh` were not re-run in
  this pass; that is `/verify` and `/test` territory (both Pass; the suites
  were re-run above after the comment edit only).
- Older git (2.8-era) and Linux dash are not exercised; macOS git and
  `/bin/sh` plus `dash` only. Unchanged from the verify report.
- The quality-gates sentence describes the remote-less `main` and detached
  cases; it does not enumerate the `no_remote_default:*` full-scope reasons,
  which live in the detector header.
