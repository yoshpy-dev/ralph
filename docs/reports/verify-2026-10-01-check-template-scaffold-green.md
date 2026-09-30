# Verify report: check-template-scaffold-green

- Date: 2026-10-01
- Plan: docs/plans/active/2026-09-30-check-template-scaffold-green.md (issue #189)
- Verifier: verifier subagent (Claude Code), cycle 1
- Scope: `git diff main...HEAD` at HEAD `bfbbbf8` (`scripts/check-template.sh` + byte-identical `templates/base/scripts/check-template.sh`, `tests/test-check-template.sh`, `scripts/verify.local.sh`, `internal/scaffold/embed_test.go` comment, `docs/tech-debt/README.md` row removal, plan, self-review report, insight events). Spec compliance (AC-1..AC-7) + static analysis + doc drift. No behavioral test execution beyond the single confirmation runs this task explicitly allows and the self-review's own 9 findings cross-check (full behavioral coverage is `/test`'s job).

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `CI=true ./scripts/check-template.sh` (repo root) | PASS | `Template structure looks good.`, exit 0, no `FAIL:` line |
| `cmp scripts/check-template.sh templates/base/scripts/check-template.sh` | PASS | Byte-identical |
| entry count in `required_files` | PASS | 25 (extracted with awk), matches AC-4's "25 items" and `GOLDEN_ENTRIES` |
| `bash tests/test-check-template.sh` | PASS | `40 passed, 0 failed, 0 skipped` (this sandbox runs non-root, so the two root-skip cases E/F also ran for real rather than skipping) |
| `go test ./internal/scaffold/... -count=1` | PASS | `ok` |
| `shellcheck -S warning scripts/check-template.sh tests/test-check-template.sh` | PASS | Exit 0, no findings at warning level or above |
| `scripts/verify.local.sh:149` shellcheck target list | PASS | Contains `scripts/check-template.sh` (grep confirmed); no `templates/base/scripts/verify.local.sh` counterpart exists, so no second site to update |
| `grep -rn '#189' docs/tech-debt/README.md` | PASS | No match; the removed row's diff is a clean single-line deletion, no orphaned remnant, surrounding rows untouched |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | `check-sync.sh`: 159 IDENTICAL / 0 DRIFTED; `check-pipeline-sync.sh` OK; `check-skill-sync.sh` 13 in lock-step; `check-template-purity.sh` PASS; golang verifier: `gofmt: ok`, `0 issues`; `secret-scan-branch.sh`: clean (`d754bcd..bfbbbf8` against `origin/main`). Evidence: `docs/evidence/verify-2026-09-30-160046.log` |
| `./scripts/check-sync.sh` (standalone) | PASS | Same summary as above, `PASS: all files in sync.` |
| `./scripts/check-template-purity.sh` (standalone) | PASS | `PASS: no meta-repo-specific references found in templates.` |
| `grep -rn -i 'meta-repo\|issue #\|#189' scripts/check-template.sh templates/base/scripts/check-template.sh` | PASS | No match — shipped comments carry no issue number or meta-repo-only wording |
| `grep -rn 'check-template' docs/quality/ docs/architecture/repo-map.md README.md docs/recipes/` | PASS | Only generic CI-check mentions (quality-gates.md's "must pass in CI" list, repo-map.md's `scripts/` inventory); none describe the old fail-open behavior or claim README.md/docs/research/docs/roadmap are required |

## AC-by-AC

| AC | Verdict | Evidence |
| --- | --- | --- |
| AC-1 (root `CI=true` run, no FAIL, exit 0) | PASS | Live run above: `Template structure looks good.`, exit 0 |
| AC-2 (missing hook → exit 1 with bare path; existing hook with argument → no FAIL; shared dispatcher → exactly one FAIL line) | PASS | `tests/test-check-template.sh` case E, all four sub-cases green (line-level: "an existing hook referenced with an argument passes with no FAIL", "a missing hook fails with the path only (no argument) and exit 1", "a dispatcher missing across several events produces exactly one FAIL line", "an unreadable settings.json is reported instead of silently skipped"). Independently reproduced outside the test harness in a scratch fixture (`/private/tmp/.../scratchpad/verify189/fixture`): the fixed script FAILs with `./.claude/hooks/missing.sh` (no argument) for a missing hook, and passes with no FAIL for an existing hook referenced as `./.claude/hooks/ralph-dispatch.sh PreToolUse` |
| AC-3 (fresh scaffold, `go run ./cmd/ralph init --yes <tmp>`, no FAIL/exit 0; SKIP not PASS when `go` unavailable) | PASS | Test case G green in the same run ("a fresh ralph init scaffold passes check-template.sh with no FAIL line"); `tests/test-check-template.sh:432` uses `skip` (not `pass`) when `go` is unavailable — confirmed by reading the code (finding 7's fix) |
| AC-4 (25 entries = GOLDEN_ENTRIES; Go test green; byte-identical copies) | PASS | `required_files` has 25 entries (counted). `go test ./internal/scaffold/... -count=1` → `ok`. `cmp` → byte-identical. Case A ("required_files matches the golden list") green in the test run |
| AC-5 (mutation: subshell revert breaks missing-hook detection; full-path-as-key breaks the argument case) | PASS | Independently reproduced both mutations in scratch copies of `scripts/check-template.sh` (not just trusting the plan's Deviation notes): (1) reverting the hook loop to `tr -d '"' \| cut ... \| sort -u \| while read ...` (pipeline/subshell form) against a fixture with a missing hook reintroduces the fail-open bug exactly as described — prints the `FAIL:` line, then still prints `Template structure looks good.` and exits 0; (2) dropping the `cut -d ' ' -f 1` step (using the full command string as the lookup key) against a fixture with an existing hook referenced with an argument turns the previously-passing case into a `FAIL: ... references missing hook: ./.claude/hooks/ralph-dispatch.sh PreToolUse` (argument now baked into the compared path), exit 1 |
| AC-7 (3 find-driven loops propagate failure; space-in-path handled; mutation: `find \| while` subshell drops the failure) | PASS | Test cases F all green: non-executable script detected, skill dir without SKILL.md detected, agent file missing `tools:` detected, executable script path containing a space not flagged (line 393's judgment now uses `! grep -q '^FAIL:'`, matching finding 8's fix — confirmed by code read, not just trusting the plan), unreadable subtree reported instead of silently skipped. The AC's own mutation claim (reverting a loop to `find ... \| while read`) is recorded in the plan's Deviation notes and not independently re-mutated here (three near-identical structural mutations beyond the two already reproduced for AC-5 was judged lower marginal value than the doc-drift and self-review cross-checks below); treated as documentary evidence consistent with the current code's shared temp-file-then-`while`-read pattern across all three loops |
| AC-6 (tech-debt row removed; shellcheck clean; verify.local.sh target list updated; full-scope static verify + check-sync + check-template-purity green) | PASS | All five sub-checks confirmed live above (tech-debt grep, shellcheck, verify.local.sh grep, full-scope run-static-verify.sh, check-sync.sh, check-template-purity.sh) |

## Self-review findings cross-check (9 findings, all claimed fixed in Slice B `8fd5113`)

Read each against the current code rather than trusting the plan's summary:

| # | Severity | Status | Evidence |
| --- | --- | --- | --- |
| 1 | MEDIUM | Fixed | `scripts/check-template.sh:25-27` / template copy: comment now reads "Files a `ralph init` scaffold does not receive are deliberately not listed here (for example, the ralph repository's own README and its docs/research/ and docs/roadmap/ notes)" — no issue number, no "meta-repo" wording. Confirmed no shipped file contains `meta-repo`, `issue #`, or `#189` (grep above) |
| 2 | LOW | Fixed | `2>/dev/null` removed from the `find` calls; only existing search roots are passed (`[ -d "$root" ] && set -- "$@" "$root"`); an unreadable subtree now makes `find` exit non-zero, reported via `fail "could not list ... find exited with $rc"` (code read at `scripts/check-template.sh:65-101`). Test case F's "unreadable subtree" sub-case is green |
| 3 | LOW | Fixed | grep's own exit code is captured (`rc=$?`) and `fail`s when `rc -gt 1`, separate from the `tr`/`cut` pipeline (`scripts/check-template.sh:118-121`). Test case E's "unreadable settings.json" sub-case is green |
| 4 | LOW | Fixed | Hook paths are deduplicated via `cut -d ' ' -f 1 \| sort -u` before the loop (`scripts/check-template.sh:127`), so a dispatcher shared by 7 events now produces one FAIL line. Test case E's "dispatcher missing across several events produces exactly one FAIL line" is green |
| 5 | LOW | Fixed | Top comment now scopes correctly: "The find-driven loops below ... read from a temp file instead of looping over `$(find ...)` directly ... The settings hook-reference check below reads from a temp file instead of piping into `while`, because a pipeline runs the loop body in a subshell" (`scripts/check-template.sh:11-14`) — subshell issue now attributed to the hook check, not the find loops. Test header (`tests/test-check-template.sh:1-35`) now names the hook-reference check and fresh scaffold explicitly |
| 6 | LOW | Fixed | `internal/scaffold/embed_test.go:171` comment now reads `// non-script entry (AGENTS.md, CLAUDE.md, .claude/settings.json)`, matching the current 3-item non-script set |
| 7 | LOW | Fixed | `tests/test-check-template.sh:432` uses `skip` (not `pass`) when `go` is unavailable; a dedicated `skip()` helper (line 69) exists and is excluded from the pass count |
| 8 | LOW | Fixed | `tests/test-check-template.sh:393` now asserts `! grep -q '^FAIL:'` instead of the old `! grep -qF 'with space'`, which never matched the fixed script's actual two-line FAIL output |
| 9 | LOW | Fixed | `trap 'exit 130' INT`, `trap 'exit 143' TERM`, `trap 'exit 129' HUP` added alongside the existing `trap cleanup EXIT` (`scripts/check-template.sh:18-20`). Independently verified (not just code-read) with a minimal script using the identical trap pattern: `SIGTERM` sent to `/bin/sh`, `/bin/bash`, and `/bin/dash` all ran `cleanup` and exited 143 — dash previously left its temp dir behind per the self-review's own probe, now fixed |

All 9 findings confirmed fixed against the code at HEAD `bfbbbf8`, not only against the plan's summary.

## Coverage gaps

- AC-7's own mutation claim (reverting one of the three find-driven loops to `find ... | while read`) was not independently re-mutated here; only the two AC-5 hook-check mutations were independently reproduced. The plan's Deviation notes record this mutation as already run in Slice A/B; behavioral re-confirmation, if wanted, is `/test`'s job.
- The fresh-scaffold case (AC-3/G) was confirmed via the test suite's own `go run ./cmd/ralph init --yes <tmp>` invocation inside `bash tests/test-check-template.sh`, not via a second standalone build+run outside the harness.
- No isolated multi-run stability check (e.g., repeated `go test -run TestTemplateBase` or repeated `bash tests/test-check-template.sh` for flake detection) was performed; a single green run of each was treated as sufficient for spec-compliance verification, consistent with this task's scope (behavioral coverage depth is `/test`'s responsibility).
- Did not re-verify `docs/quality/quality-gates.md`'s and `docs/architecture/repo-map.md`'s full text beyond the targeted `check-template` grep; no other changed behavior in this diff (only `check-template.sh`'s internals and the `required_files` list) touches content elsewhere in those files.

## Verdict

- Verified: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7; all 9 self-review findings fixed in the code; no doc drift found in `docs/tech-debt/README.md`, `docs/quality/quality-gates.md`, `docs/architecture/repo-map.md`, `README.md`, `docs/recipes/`; shipped comments contain no meta-repo-only or issue-number wording.
- Partially verified: AC-7's loop-subshell mutation (relied on the plan's recorded evidence rather than an independent re-mutation in this pass).
- Not verified: none.
- **Overall verdict: PASS.**
