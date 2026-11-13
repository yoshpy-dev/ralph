# Cross-review triage report: gate-tool-versions

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-gate-tool-versions.md
- Base branch: main (bb416c48; the branch merged origin/main at 349f5763, reviewed at 6b6fd909)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-gate-tool-versions.md (D1–D8, AC1–AC10; D7 says measuring changes nothing, D6 says doctor measures in the seat's non-interactive shell, `$SHELL -lc`)
- Self-review report: docs/reports/self-review-2026-11-13-gate-tool-versions.md (second run, LOW 5; neither finding below is among them)
- Verify report: docs/reports/verify-2026-11-13-gate-tool-versions.md (pass; V1, pnpm fetching its packageManager version, fixed in e7274fec by the eighth and ninth variables)
- Test report: docs/reports/test-2026-11-13-gate-tool-versions.md (pass, second run with a 60-minute timeout)
- Implementation context summary: the gate (`/bin/sh -c`, no startup files) and doctor (`$SHELL -lc`, a login shell that reads its startup files) run the same probe script, `toolchain.BuildScript`, with `toolchain.ProbeEnv`'s environment, which sets nine variables that stop auto-install, auto-pin, downloads, and pnpm's version switching. The script's `rtc_tool` records what `command -v` prints and then runs `"$rtc_t" "$@"`.
- Reviewer: codex gpt-6-astra, reasoning effort xhigh, `-c sandbox_mode=read-only exec review --ignore-rules --base main`. Its summary line: "Two reproduced shell-integration defects can produce incorrect doctor comparisons or re-enable probe side effects. Diff and targeted formatting checks passed; Go test suites were not rerun in the read-only environment."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Reapply probe safety variables after shell startup: a login startup file that exports `COREPACK_ENABLE_AUTO_PIN=1` or turns pnpm's version management on overrides the environment doctor passes, and the script never sets them again; codex reproduced it with bash's startup files. Doctor can then download a package manager or write `packageManager` into package.json. | Real: the environment is set before the login shell reads its startup files, and `BuildScript` has no line that sets the nine variables again (probe.go `probeFunc`, doctor_node.go:152-158). It breaks D7 and AC10, which this branch introduced, so it is a hole of this change, not an older type. Worth fixing: the script can unset and export the nine itself at its top, a few lines shared by the gate and doctor. | internal/toolchain/probe.go, internal/cli/doctor_node.go |
| 2 | [P2] Honor shell aliases when probing doctor tool versions: when the login shell defines an alias for a tool, `command -v` prints the alias, but `"$rtc_t" "$@"` bypasses alias expansion and runs the PATH executable. With a Node 22 alias and Node 24 on PATH, zsh's `node -v` printed v22 and the probe v24 next to the alias. Doctor compares the wrong version: a false mismatch or a false pass. | Real: a quoted parameter expansion is not alias-expanded in zsh or bash, so the line the record names and the command that ran differ (probe.go:117-119). A false pass hides the mismatch the check exists to report, which is the direction the series fixes ("知らせが消える向きの穴は直す"), and the code is new in this branch. Worth fixing: the record and the run must name the same command. | internal/toolchain/probe.go, internal/cli/doctor_node.go, internal/org/gate_tools.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
