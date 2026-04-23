# Self-review report: colorize-upgrade-diff

- Date: 2026-04-23
- Plan: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md`
- Reviewer: reviewer subagent (self-review)
- Scope: diff quality only for commit `cd5dd69` on branch `feat/colorize-upgrade-diff`. Spec compliance, test coverage, and doc drift are explicitly out of scope (handled by `/verify` and `/test`).

## Evidence reviewed

- `git diff main...HEAD --stat` — 9 files, +696 / -64.
- Full per-file diff for:
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff.go`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/colorize.go` (new)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/colorize_test.go` (new)
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/upgrade/unified_diff_test.go`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/upgrade.go`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/internal/cli/cli_test.go`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/go.mod`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md`
  - `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt` (new)
- Cross-checks performed:
  - `grep -rn "@@ -" --include="*.go" --include="*.sh" --include="*.md"` to confirm no downstream code parses the old hunk-header format. Only the doc-comment in `unified_diff.go` references it (now historical). Confirms the plan’s “display artifact, not parsed” claim.
  - Verified `term.IsTerminal` signature in `/Users/hiroki.yoshioka/go/pkg/mod/github.com/charmbracelet/x/term@v0.2.2/term.go` (`func IsTerminal(fd uintptr) bool`) — matches `*os.File.Fd()`’s `uintptr` return.
  - Byte-level audit of test fixtures (`internal/upgrade/unified_diff_test.go:149` and `docs/evidence/...:8`) to confirm the gutter math (`gutterWidth=3` produces `"100"+" "+"   "+" │ "+"-X"`) — matches both files.
  - Confirmed nothing in `templates/base/` changed (no mirror-discipline obligations triggered).
  - `go.mod` change is a pure indirect → direct promotion of `github.com/charmbracelet/x/term v0.2.2`, no version bump. No new transitive dependencies.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `runUpgradeIO`’s signature now carries six positional parameters (`targetDir, force, in, out, errOut, colorize`). Adding a seventh — likely once the next display-mode flag (e.g. `--no-pager`, theme) lands — will become an ergonomic and review hazard. The trailing `bool` is also a known anti-pattern at call sites (`runUpgradeIO(dir, false, ..., true)` vs `..., false)`). | `internal/cli/upgrade.go:99-102` and the eight call sites updated in `internal/cli/cli_test.go:503,547,592,668,700,754,772,812,822,862`. | Optional now; convert to a small `runUpgradeIO(targetDir string, opts upgradeIOOpts)` (or similar) the next time another display-flag is added. Track in tech-debt below. |
| LOW | naming | `ansiDimDefault` reads as “dim AND default”, which is opaque. The constant is just SGR `2` (faint). Other constants in the block (`ansiCyan`, `ansiRed`) follow `ansi<Color>`/`ansi<Attr><Color>` — `ansiDim` would be consistent and grep-friendlier. | `internal/upgrade/colorize.go:14`. | Rename to `ansiDim`. Single-file, no public surface. |
| LOW | maintainability | The hunk header uses a U+2013 en-dash inside `formatRange` (`L1–4`). It is visually correct but reduces grep-ability (`grep "L1-4"` will miss it) and forces every test to copy-paste the exact rune. Tests already do this, which compounds the friction if the format ever needs to change. | `internal/upgrade/unified_diff.go:92` and tests at `internal/upgrade/unified_diff_test.go:64,66`. | Acceptable as a display choice; if maintenance of the test fixtures becomes painful, consider an ASCII `-` or extracting the rune to a named constant alongside `diffSeparator`. |
| LOW | maintainability | `colorize` is plumbed through `runUpgradeIO → resolveConflict → showDiff` even though it is consumed only by `showDiff`. When `--force` is set, the value is passed along but never read (the conflict path is bypassed). Three function signatures changed for one read site. | `internal/cli/upgrade.go:102, 216, 336, 362`. | Acceptable for now. If a future Options struct lands (see first LOW), this collapses naturally. |
| LOW | readability | `gutterWidth`’s `maxLine = oldStart + oldCount` (and the new-side equivalent) is correct but takes a second to verify because `oldStart` is 0-based and the displayed value is 1-based. The off-by-one safety relies on the “0-based start + count = last 1-based number” identity. A short comment would prevent future drift. | `internal/upgrade/unified_diff.go:99-106`. | Add a one-line comment: `// oldStart is 0-based; oldStart+oldCount equals the last displayed (1-based) line number.` |
| LOW | unnecessary-change | `internal/upgrade/unified_diff.go:23-25` keeps the original “When either side lacks a trailing newline…” paragraph next to the new “Each emitted change line carries two gutter columns…” paragraph. The two are stacked, but a single coherent doc-comment would read better. Pure prose nit. | `internal/upgrade/unified_diff.go:13-26`. | Optional copy-edit. |

No CRITICAL or HIGH findings. All findings are LOW. No blocking issues.

Items explicitly checked and clean:

- **Secrets / credentials**: none introduced; ANSI literals only.
- **Debug code**: no `fmt.Println`, no `// TODO`, no commented-out code in the diff.
- **Exception handling**: no errors swallowed. `shouldColorize` correctly defaults to `false` on `nil` `*os.File`. `Colorize("")` short-circuits. `term.IsTerminal` returns a `bool`, no error path to swallow.
- **Null safety / boundaries**: `Colorize` handles empty input, trailing-newline preservation, and unrecognized prefixes (passthrough). `ansiForLine` byte-indexes after `diffSeparator` with an `after < len(line)` guard, so a malformed gutter row cannot panic. `shouldColorize` guards `out == nil` before `Fd()`.
- **Security**: no untrusted-input parsing, no shell exec, no path construction. ANSI escapes are emitted only when both `term.IsTerminal` and `NO_COLOR==""` agree, so log-capture / CI pipelines stay clean. The `\x1b` byte appears only in fixed `const` strings — no user-controlled string ever flows into an SGR code, so terminal-injection (e.g. embedded title-set escapes) is not in play.
- **Typos / copy-paste**: hunk-header text (`旧`, `新`, `(空)`) matches between implementation, tests, and the captured evidence file (byte-verified at `unified_diff_test.go:149` vs `docs/evidence/...:8`).
- **Gutter math**: implementation output `"100"+" "+"   "+" │ "+"-X"` exactly matches the asserted bytes in `unified_diff_test.go:149` (`"100     │ -X\n"`, 5 spaces between `100` and `│`).
- **Mirror discipline** (`.claude/rules/...`): no files under `templates/base/`, `.claude/hooks/`, `.claude/rules/`, or `.claude/skills/` were touched, so the root↔templates mirror invariant is not engaged.
- **Generated artifacts**: no `coverage.out`, `bin/`, `*.prof`, or `.harness/state/` files in the diff. The single new evidence file (`docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt`) is a deliberate, plan-referenced capture.
- **`go.mod` change**: pure indirect→direct promotion of `github.com/charmbracelet/x/term v0.2.2`. No new dependency, no version bump, no `replace` directive.

## Positive notes

- Clean separation between rendering (`UnifiedDiff` returns plain text) and presentation (`Colorize` is a pure post-processor). This made the `colorize=false` non-TTY assertion and the new `TestRunUpgrade_InteractiveDiff_ColorizesWhenEnabled` integration test easy to express.
- `Colorize` degrades gracefully on unknown line shapes (passthrough) instead of corrupting output — explicitly documented in the doc-comment and covered by `TestColorize_UnknownLinePassthrough`.
- `ansiForLine` correctly accounts for the multi-byte `│` separator with a written-down rationale at `colorize.go:73-74`. The `diffSeparator` constant is shared with `unified_diff.go` (single source of truth) and called out as multi-byte.
- `shouldColorize` honors the de-facto `NO_COLOR` standard with a link to the spec. The `nil`-out guard keeps the function safe under future refactors.
- Test renaming from `-d` / `+d` substring matches to `│ -d` / `│ +d` actually strengthens the assertions: it now requires the gutter to be present, not just the prefix.
- The captured `docs/evidence/colorize-upgrade-diff-2026-04-23-nocolor.txt` file is a real interactive-session transcript (with the literal `[o]verwrite / [s]kip / [d]iff ?` prompt mid-line on line 5) and gives any future reviewer an immediate reference for the rendered output.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `runUpgradeIO` positional-parameter creep (now 6 params; trailing `bool colorize`) | Low — callers must remember argument order; risk grows linearly with each new display flag. | Out of scope for this UX change; refactor would touch every test in `internal/cli/cli_test.go` for no current behavior gain. | Next time another display/IO knob lands (theme, paging, width override). | This report; `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md` |

(Single new entry. Will be appended to `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/tech-debt/README.md`.)

## Recommendation

- Merge: **YES** — proceed to `/verify`. No CRITICAL or HIGH findings; all LOW items are deferrable.
- Follow-ups (none blocking):
  1. Rename `ansiDimDefault` → `ansiDim` (single-file, no API surface).
  2. Add the one-line comment to `gutterWidth` clarifying the 0-based / 1-based identity.
  3. Track the `runUpgradeIO` parameter-creep in tech-debt (entry above).
