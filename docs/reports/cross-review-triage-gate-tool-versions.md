# Cross-review triage report: gate-tool-versions

- Date: 2026-11-15
- Plan: docs/plans/active/2026-11-12-gate-tool-versions.md
- Base branch: main (7433ce84; the branch merged origin/main five times, last at 13250481, reviewed at d2fc5433)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/3 (third pipeline run; the cap was raised to 3 after the cycle 2 ACTION_REQUIRED, which does not increment the cycle; cycle 1 had 2 ACTION_REQUIRED fixed in 7c0bf678, cycle 2 had 1 fixed in dddd83bb)
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-gate-tool-versions.md (the Progress checklist records both cross-review fixes and the third pipeline run)
- Self-review report: docs/reports/self-review-2026-11-13-gate-tool-versions.md (Run 3, Merge: yes, LOW R3-1..R3-4, fixed by the third sync-docs pass 22080ee1)
- Verify report: docs/reports/verify-2026-11-13-gate-tool-versions.md (Run 3, pass, LOW V11, fixed by 22080ee1; ksh93 checked on the real /bin/ksh before and after dddd83bb)
- Test report: docs/reports/test-2026-11-13-gate-tool-versions.md (Run 3, pass)
- Reviewer: codex gpt-6-astra, reasoning effort xhigh, `-c sandbox_mode=read-only exec review --ignore-rules --base main`. Its output (the `-o` file, verbatim): "No actionable regressions were found in the toolchain probing, Node diagnostics, or gate/regate integration. Tests remain unverified because the read-only sandbox prevented Go from creating its temporary build directory." The reviewer's sandbox kept it from running the Go tests; `/test` ran them (pass), so nothing is missing from the pipeline's evidence.
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2 history

- Date: 2026-11-15. Base main (bbe2807d; merged at 34936129), reviewed at e45aa6ab. Cycle 2/2 (cap reached). Total reviewer findings: 1. After triage (cycle 2): 1 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as ca14df41.

### Cycle 2 triage context

- Active plan: docs/plans/active/2026-11-12-gate-tool-versions.md (D1, D6, D7, AC6, AC10; the Progress checklist records the cycle 1 fix and the cycle 2 pipeline)
- Self-review report: docs/reports/self-review-2026-11-13-gate-tool-versions.md (cycle 2, Merge: yes, LOW 3; its LOW 1 already named the same remedy, running the checked `"$rtc_p"` instead of `"$rtc_t"`, and noted that macOS /bin/ksh prints an alias's value alone)
- Verify report: docs/reports/verify-2026-11-13-gate-tool-versions.md (cycle 2, pass, LOW V7–V10)
- Test report: docs/reports/test-2026-11-13-gate-tool-versions.md (cycle 2, pass)
- Implementation context summary: since 7c0bf678 the probe script runs a tool only when `command -v` printed something that starts with `/` (`rtc_tool`'s `case $rtc_p in /*`), and runs it by name (`"$rtc_t" "$@"`). Doctor measures in `$SHELL -lc` when `$SHELL`'s base name is one of `doctorNodeShells` (sh, bash, zsh, dash, ksh, mksh, yash), else in `/bin/sh -lc`. The gate runs `/bin/sh -c`, which reads no startup file.
- Reviewer: codex gpt-6-astra, reasoning effort xhigh, `-c sandbox_mode=read-only exec review --ignore-rules --base main`. Its summary line: "The doctor check can report the wrong Node version for an explicitly accepted shell. The shell-resolution defect was reproduced; focused Go tests could not start because the read-only sandbox blocked creation of Go's temporary build directory."
- The first cycle 2 run (2026-11-15 18:54) was cut off when the session restarted, before it wrote its `-o` file; the git state was unchanged, and the reviewer was run again at 19:02 without counting a cycle. The git state before and after the second run was identical (`cmp` of the two state files).
- Checked by the triager on this machine (a fake `node` on PATH printing v24, an alias target `node22` printing v22): macOS `/bin/ksh` (ksh93) answers `command -v node` with the alias's value `/…/node22`; running that path prints v22, running the name prints v24. For an alias with arguments it prints the value in quotes (`'/…/node22 --x'`), which the current check already treats as not a file. `/bin/sh` (bash in POSIX mode) and dash print `alias node='…'`, zsh `alias node=…`; plain `bash -c` prints the PATH file. mksh and yash are not installed here.

### Cycle 2 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Exclude ksh until its alias resolution is handled correctly: with `$SHELL=/bin/ksh` and a login profile that aliases `node` to another installation, ksh93's `command -v` returns the alias's absolute target, which passes the probe's guard, but `"$rtc_t"` runs the unaliased PATH executable; with a Node 22 alias and Node 24 on PATH the probe records the Node 22 path with `v24.15.0`, and doctor can report a false match. | Real: reproduced above. The record names one file and the version comes from another, the same mismatch the cycle 1 finding 2 was about, left open for a shell `doctorNodeShells` accepts. A false match hides the mismatch the check exists to report, the direction the series fixes, and the code is new in this branch. Worth fixing: running the path that passed the check (`"$rtc_p"`) makes the record and the run the same string in every shell, and in ksh93 measures what `node` runs there. | internal/toolchain/probe.go, internal/cli/doctor_node.go |

Fixed in dddd83bb: `rtc_tool` runs the path `command -v` printed (`"$rtc_p"`), so the record and the run name the same file in every shell (consult consult-plan-kshalias: run the checked path rather than sending ksh to the `/bin/sh` fallback). The cap was raised to 3 and the whole pipeline re-ran.

### Cycle 2 WORTH_CONSIDERING

None.

### Cycle 2 DISMISSED

None.

## Cycle 1 history

- Date: 2026-11-13. Base main (bb416c48; merged at 349f5763), reviewed at 6b6fd909. Cycle 1/2. Total reviewer findings: 2. After triage (cycle 1): 2 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 0abfb274; both fixed in 7c0bf678.

### Cycle 1 triage context

- Self-review report (cycle 1, second run, LOW 5), verify report (cycle 1, pass; V1 fixed in e7274fec), test report (cycle 1, pass with a 60-minute timeout).
- Reviewer summary line: "Two reproduced shell-integration defects can produce incorrect doctor comparisons or re-enable probe side effects. Diff and targeted formatting checks passed; Go test suites were not rerun in the read-only environment."
- The git state before and after the reviewer's run was identical.

### Cycle 1 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Reapply probe safety variables after shell startup: a login startup file that exports `COREPACK_ENABLE_AUTO_PIN=1` or turns pnpm's version management on overrides the environment doctor passes, and the script never sets them again; codex reproduced it with bash's startup files. Doctor can then download a package manager or write `packageManager` into package.json. | Real: the environment is set before the login shell reads its startup files, and `BuildScript` had no line that set the nine variables again. It broke D7 and AC10, which this branch introduced. Fixed in 7c0bf678 (`probeEnvLines`). | internal/toolchain/probe.go, internal/cli/doctor_node.go |
| 2 | [P2] Honor shell aliases when probing doctor tool versions: when the login shell defines an alias for a tool, `command -v` prints the alias, but `"$rtc_t" "$@"` bypasses alias expansion and runs the PATH executable. With a Node 22 alias and Node 24 on PATH, zsh's `node -v` printed v22 and the probe v24 next to the alias. | Real: a quoted parameter expansion is not alias-expanded, so the record and the run differed; a false pass hides the mismatch. Fixed in 7c0bf678 by not running what `command -v` did not print as an absolute path (consult consult-plan-xreviewfix: not running and reporting, rather than running the alias with eval, which could run `mise exec` and install a version). | internal/toolchain/probe.go, internal/cli/doctor_node.go, internal/org/gate_tools.go |

### Cycle 1 WORTH_CONSIDERING

None.

### Cycle 1 DISMISSED

None.
