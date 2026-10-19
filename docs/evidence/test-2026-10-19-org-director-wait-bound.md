# Test evidence: org-director-wait-bound

- HEAD 5c7fa9df (base 3a803a87), macOS arm64 (Darwin 25.5.0), go1.26.0 and go1.25.8 (`GOTOOLCHAIN`)
- 2026-10-19 22:37 to 23:03 JST. Load average 8 to 13 during the runs
- `<scratch>` is the session scratchpad outside the worktree
- No tracked file was changed. Mutants ran in a `git archive HEAD` copy under `<scratch>` (details in `test-2026-10-19-org-director-wait-bound-mutation.txt`). `git status --porcelain` was empty before and after; `git status --porcelain --ignored` (without `.harness/cache/`) differed only by the run-test.sh log `docs/evidence/verify-2026-10-19-134151.log`
- Go runs never overlapped: every command below ran alone, one after another

## 1. Targeted tests, go1.26.0

`go test ./internal/org/ -run 'TestDirectorSkill' -count=1 -v -timeout 30m`, rc 0, 1 s.
4 top-level tests and 21 subtests, 25 `--- PASS` lines, no FAIL or SKIP:

```
--- PASS: TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase (0.02s)
--- PASS: TestDirectorSkill_PinsTheAutoMergeSteps (0.00s)
--- PASS: TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap (0.00s)
--- PASS: TestDirectorSkillWaitBoundProblems (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_bounded_wait_below_the_Monitor's_timeout_ms (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_old_bullet:_--timeout-ms_0_and_no_Monitor_timeout_ms (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/N_equal_to_M (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/N_greater_than_M (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/M_over_the_Monitor's_maximum (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_paragraph's_timeout_ms_differs_from_the_bullet's_M (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/two_timeout_ms_values_in_the_bullet (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_minutes_the_bullet_names:_N's_25,_the_Monitor's_default_5_and_maximum_30 (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/a_stale_25_分_after_the_list_once_N_is_20_minutes (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/N_not_a_whole_number_of_minutes (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/minutes_and_timeout_ms_outside_the_subsection (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/M_missing_from_the_bullet,_though_the_paragraph_after_the_list_has_one (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/N_missing_from_the_bullet,_though_the_Codex_bullet_has_one (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/two_--timeout-ms_values_in_the_bullet (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/--timeout-ms_0_elsewhere_in_the_text,_with_a_correct_bullet (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/--timeout-ms_0_at_the_very_end_of_the_text (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/--timeout-ms_0_wrapped_over_two_lines_elsewhere (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_bullet_wrapped_over_lines (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/the_bullet_at_the_end_of_the_text (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/no_Claude_Code_bullet (0.00s)
    --- PASS: TestDirectorSkillWaitBoundProblems/a_Claude_Code_bullet_only_in_another_subsection (0.00s)
PASS
ok  	github.com/yoshpy-dev/ralph/internal/org	0.553s
```

## 2. Targeted tests, go1.25.8

`GOTOOLCHAIN=go1.25.8 go test ./internal/org/ -run 'TestDirectorSkill' -count=1 -v -timeout 30m`, rc 0.
`GOTOOLCHAIN=go1.25.8 go env GOVERSION` prints `go1.25.8`, and `go test -n` resolves the compiler under `golang.org/toolchain@v0.0.1-go1.25.8.darwin-arm64`.
The verbose output is the same 25 `--- PASS` lines as section 1; `diff` of the two outputs differs only in the last line:

```
52c52
< ok  	github.com/yoshpy-dev/ralph/internal/org	0.553s
---
> ok  	github.com/yoshpy-dev/ralph/internal/org	0.356s
```

The first go1.25.8 run happened to end with the same `0.553s` as the go1.26.0 run. Two more runs of each toolchain without `-v` ended at 0.357 to 0.388 s, and the run recorded here is the one rerun with `-v` after them.

## 3. `GOFLAGS=-timeout=30m ./scripts/run-test.sh` (go1.26.0)

```
# Verification run
- Timestamp: 2026-10-19T13:41:51Z
- Mode: test
- Requested scope: changed
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
ok  	github.com/yoshpy-dev/ralph/internal/cli	333.027s
ok  	github.com/yoshpy-dev/ralph/internal/config	0.514s
ok  	github.com/yoshpy-dev/ralph/internal/insights	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org	428.686s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	(cached)
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	(cached)

==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-10-19-134151.log
rc=0 elapsed=996s
```

Shell suites (local verifier), 41, each ending `OK`. No summary line has a non-zero failure count. One skip, in `tests/test-secret-scan-branch.sh`: `the real git older than 2.41 case (git 2.49.0 is 2.41 or later; the stubbed old git covers it)`.

```
tests/test-agent-models.sh tests/test-agent-phase-boundaries.sh tests/test-archive-plan.sh
tests/test-branch-name.sh tests/test-check-mojibake.sh tests/test-check-skill-sync.sh
tests/test-check-template.sh tests/test-codex-exec-invocation.sh
tests/test-detect-changed-languages.sh tests/test-detect-languages-terraform.sh
tests/test-ensure-pr-ready.sh tests/test-ensure-pr-title-prefix.sh tests/test-gc-artifacts.sh
tests/test-hook-wiring.sh tests/test-insights-append.sh tests/test-language-pack-monorepo-roots.sh
tests/test-lib-json.sh tests/test-new-feature-plan.sh tests/test-no-loop-references.sh
tests/test-plan-visual.sh tests/test-post-edit-verify.sh tests/test-pr-owner-lookup.sh
tests/test-pre-bash-guard.sh tests/test-ralph-config.sh tests/test-ralph-dispatch.sh
tests/test-ralph-worktree.sh tests/test-run-verify-branch-secret-scan.sh
tests/test-run-verify-concurrency.sh tests/test-run-verify-scope.sh
tests/test-secret-scan-branch.sh tests/test-secret-scan.sh tests/test-self-review-scope.sh
tests/test-skill-insight-cycle.sh tests/test-sync-skills.sh tests/test-template-purity.sh
tests/test-terraform-gitignore.sh tests/test-terraform-pack-verify.sh
tests/test-terraform-rule-frontmatter.sh tests/test-verify-local-hook-tests.sh
tests/test-verify-mode-split.sh tests/test-xreview-helpers.sh
```

The five `(cached)` packages do not import `internal/org`'s test files: the only Go change is `internal/org/director_skill_sync_test.go`, a `_test.go` file of package `org`, which is not part of what importers compile. `internal/org` and `internal/cli` were run again.

## 4. Mutants

22 runs on go1.26.0 and the same 22 on go1.25.8, in the `git archive HEAD` copy. Verdicts and problem lines were the same on both toolchains (only the timings differed). Full table: `test-2026-10-19-org-director-wait-bound-mutation.txt`.

| Group | Runs | Result |
| --- | --- | --- |
| control (HEAD files) | 1 | pass |
| lead's mutants (1) to (6), with (1) on each of the four copies, (3) as the old bullet and as the whole 3a803a87 skill, (5) on each of the three "25 分" | 12 | 12 red: (1) to (5) by the real-copies test, (6) by the table test only (the real-copies test stays green on the HEAD skill, whose N < M) |
| (6) companions: an N == M skill with the minutes in step, under the HEAD test and under the (6) test | 2 | 2 red (HEAD: the real-copies test; (6): only the table test) |
| extra edge cases from the plan (M missing from the bullet, `--timeout-ms 0` in the Codex bullet) | 2 | 2 red |
| gap probes (prose of AC2 and AC3 deleted from all four copies, and the AC3 rules reverted to their 3a803a87 wording) | 5 | 5 green, recorded as a test gap |

## 5. Worktree state

```
git status --porcelain                      # before: empty, after: empty
git status --porcelain --ignored            # without .harness/cache/: 13 lines before, 14 after
> !! docs/evidence/verify-2026-10-19-134151.log
```
