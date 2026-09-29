# Review report: check-template-required-files-test (self-review, cycle 1)

- Date: 2026-09-29
- Plan: docs/plans/active/2026-09-29-check-template-required-files-test.md (issue #183)
- Reviewer: reviewer subagent (Claude Code)
- Scope: diff quality only for `git diff main...HEAD -- tests/ internal/ scripts/ templates/ docs/tech-debt/` at 6814e0e (code commit 26c247f: new `tests/test-check-template.sh`, `internal/scaffold/embed_test.go`, both `check-template.sh` copies, one tech-debt row replaced). Read in full: `tests/test-check-template.sh`, `scripts/check-template.sh`, the `embed_test.go` hunks. No spec-compliance verdict, no test verdict, no documentation-drift audit.

## Evidence reviewed

- HEAD 6814e0e and `git status --porcelain` empty at review start. `git log main..HEAD`: 12c50fa, 82efaad (plan), 26c247f (code), 6814e0e (plan record).
- `cmp scripts/check-template.sh templates/base/scripts/check-template.sh`: identical. In each copy the only change is the `required_files` block: 10 added entries, and `scripts/xreview-helpers.sh` moved up so the `scripts/` entries follow the Go list's order.
- `git ls-files -s tests/test-check-template.sh`: mode 100755.
- `go.mod` declares `go 1.25.8`, so `strings.CutPrefix` (Go 1.20+) is available.
- One timed run of the new test, to answer the speed question: `32 passed, 0 failed`, real 2.23 s. Entries named `check-template-fixture.*` under `$TMPDIR`: 0 before the run, 0 after.
- Root claim of the new tech-debt row: `CI=true ./scripts/check-template.sh` at the worktree root exits 0. It prints 7 `FAIL:` lines, one per `ralph-dispatch.sh <Event>` command in `.claude/settings.json`, and then `Template structure looks good.` This part of the row is accurate.
- Downstream caller of `check-template.sh`: `templates/base/scripts/bootstrap.sh` does not exist. A grep for `check-template` under `templates/base` finds only `templates/base/.github/workflows/verify.yml:16` (`run: ./scripts/check-template.sh`, workflow triggered on `pull_request`) and `templates/base/docs/quality/quality-gates.md:49`.
- Fresh-scaffold probe: a binary built from this worktree into the session scratchpad, then `ralph init --yes` into the scratchpad with a scratch `HOME`. The scaffold has 30 scripts, no `scripts/bootstrap.sh`, and `.github/workflows/verify.yml`. There, `CI=true sh ./scripts/check-template.sh` exits 1 with three `Missing required file` lines (`README.md` and the two docs files) plus the 7 hook lines.
- Table shape of `docs/tech-debt/README.md:135`: the row has 7 unescaped pipes, and the header row has 6. The extra one is the pipe inside the code span `` `\| while read` `` (shown escaped here). The register already escapes a pipe inside code with a backslash (the mojibake row's `Edit`/`Write`/`MultiEdit` matcher; 6 occurrences in the file).
- `bash` 3.2.57 (`/bin/bash`, and the `bash` on PATH): `out="$(cd "" && pwd)"` returns 0 and prints the caller's cwd.
- Convention in `tests/*.sh`: 26 files call `mktemp -d`, and 24 of them register an `EXIT` trap.
- `docs/plans/active/` paths in the register on main: 16 distinct paths, all missing (plans archived, references never rewritten). `scripts/archive-plan.sh` moves the plan file and does not rewrite references.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | docs (table shape) | The new #189 row renders broken on GitHub. Its code span `` `\| while read` `` holds an unescaped pipe. GFM splits a table row on unescaped pipes before it parses code spans, so the row has 6 cells under a 5-column header. The Debt item cell ends at "the check's", every later cell moves one column right, "#189." lands under Related plan/report, and the real Related cell (the plan path and issue #189) is dropped as an excess cell. | `docs/tech-debt/README.md:135`; pipe count above; the register's backslash-escape convention | Escape that pipe with a backslash, as the mojibake row does. |
| MEDIUM | docs (accuracy) | The new row names the wrong downstream caller and understates the impact. It says "`bootstrap.sh`'s invocation of `check-template.sh` is red from the very first scaffold". `bootstrap.sh` exists only in the meta-repo: the template has no copy, and a fresh scaffold ships none. The scaffold's real caller is its own CI, the "Check template structure" step of `.github/workflows/verify.yml`, which runs on every pull request. There `check-template.sh` exits 1 (probe above), so every scaffolded project's verify job fails on every PR until someone adds the three meta-repo files. The Impact cell ("training users to ignore its output") describes a warning, not a failing CI job. Smaller: "(only `.gitkeep` placeholders)" holds for the two docs directories, not for `README.md`, which has no placeholder. #189's body and the plan's 調査 section carry the same misattribution (`bootstrap.sh` continues with `[warn]`). | `docs/tech-debt/README.md:135`; `templates/base/.github/workflows/verify.yml:15-16`; fresh-scaffold probe | Name the scaffold's `verify.yml` step as the downstream caller and say that it fails the scaffold's PR CI. Correct #189's body the same way (an external write, so the lead's call). |
| LOW | robustness (temp dir) | Case B neither checks `mktemp -d` nor registers an `EXIT` trap. Cleanup is only the final `rm -rf`, so an interrupt or a runner timeout during the 29 `check-template.sh` runs leaves `check-template-fixture.*` in the host `$TMPDIR`. If `mktemp -d` fails, `fixture` is empty. `build_fixture ""` then tries to create `/README.md`, `/scripts/*.sh` and so on (permission denied for a normal user, written for root), and `cd ""` succeeds, so `check-template.sh` runs against the caller's cwd. 24 of the 26 tests in `tests/` that call `mktemp -d` use an `EXIT` trap. | `tests/test-check-template.sh:140`, `:141`, `:147`, `:166`; the bash 3.2 `cd ""` probe | Stop case B with a `fail` when `mktemp -d` fails, and register an `EXIT` trap that removes the fixture. |
| LOW | comments | Maintenance pointers are incomplete or imprecise. (1) The golden comment says to update the golden together with the two `check-template.sh` copies. It does not name `requiredTemplateScripts`, the fourth place a `scripts/` entry has to change, and the Go var's comment does not name the shell golden. A contributor finds the fourth site only through a red test. (2) The same comment says the `scripts/` entries are "in the same order as ... requiredTemplateScripts". Nothing enforces that (the Go test compares sets), so a reorder of the Go list silently falsifies it. (3) "caught by one of the two tests" leaves the two tests unnamed. A drop from either list is caught by `TestTemplateBaseScriptsMatchCheckTemplateRequiredFiles`, and `TestTemplateBaseScriptsExist` does not detect a drop. (4) "heredoc-style block" misnames the construct, which is a multi-line double-quoted assignment. | `tests/test-check-template.sh:43-47`; `internal/scaffold/embed_test.go:51-55`, `:109` | Name all four sites in one of the two comments and point the other at it. Drop the order claim, or call it a convention. Name the Match test. Say "multi-line `required_files=\"...\"` assignment". |
| LOW | comments (stale path at merge) | The test header, the Go var comment and the new row's Related cell cite `docs/plans/active/2026-09-29-check-template-required-files-test.md`. This PR's own `/pr` step moves the plan to `docs/plans/archive/`, and nothing rewrites references, so all three dangle at merge. The register already carries 16 dangling `active/` paths. | `tests/test-check-template.sh:16`; `internal/scaffold/embed_test.go:55`; `docs/tech-debt/README.md:135` | Cite issue #183, which survives archival, or the `docs/plans/archive/` path. |
| LOW | readability (failure output) | When case A fails, it prints both 28-line lists in full. For one dropped or renamed entry, the reader has to compare two blocks by eye. The Go test prints only the names that differ. | `tests/test-check-template.sh:128-132` | Print a `diff` of the two lists, or list the entries found on only one side, as the Go test does. |

No CRITICAL or HIGH findings.

## Positive notes

- Both extractions fail closed when the block's format drifts. A missing block fails case A's "non-empty" check and hits the Go test's `Fatalf`, which names the expected format. A closing quote moved onto the last entry makes case A differ and gives the Go test an only-in name with a trailing quote. A CRLF file matches no `required_files="` line in Go, so it also ends in the 0-entry `Fatalf`.
- Each removal assertion in case B needs a non-zero exit and the specific `Missing required file: <entry>` line. The assertion is tied to the entry it removed, not to any failure of the fixture.
- The `.bak` names sit outside every path `check-template.sh` scans (`-name '*.sh'` and the exact `.claude/settings.json`). `CI=true` and the absent `.git` each skip the git-hook check. The `cd` stays inside the command substitution.
- The Go comparison is a set difference with sorted only-in lists, so its failure output is deterministic and names the differing entries.
- `set -u` is safe on bash 3.2: the array is a non-empty literal, and every function variable is declared `local`.
- The suite adds about 2 s and has no timing, network or shared state beyond its own `mktemp` directory, so it is not a flake source.
- Both `check-template.sh` copies are identical, and the change is confined to the list.

## Coverage gaps

- No `go test`, `shellcheck` or mutation re-run beyond the one timed run above. Those belong to /verify and /test.
- Case A compares in order, so reordering `required_files` fails it with no behavioral change. The comment states this, and I did not count it as a finding.
- `check-template.sh`'s other checks were read only to judge the fixture and the #189 row's accuracy. Their defects are #189's scope.
- The 6 scripts in neither list stay an open question in the plan. The new tests guard only names that are already in one of the lists.

## Recommendation

- Merge: yes, once the two MEDIUM findings are fixed. Both sit in the single new row at `docs/tech-debt/README.md:135`, and each is a one-line edit. No CRITICAL or HIGH.
- Follow-ups: correct #189's body so it names the scaffold's `verify.yml` step as the downstream caller and says it fails the scaffold's PR CI (external write, the lead's call). Fix the four LOWs in this cycle, or batch the unfixed ones into one tech-debt row.
