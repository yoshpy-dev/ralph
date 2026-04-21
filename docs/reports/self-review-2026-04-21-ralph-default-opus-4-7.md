# Self-review report: ralph default model → claude-opus-4-7 / effort xhigh

- Date: 2026-04-21
- Plan: (none — small well-scoped defaults update, no `docs/plans/active/*` entry)
- Reviewer: reviewer subagent (self-review skill)
- Scope: diff quality only — commit `4f5d5b5` on branch `chore/ralph-default-opus-4-7`
  - `templates/base/ralph.toml` — scaffold defaults
  - `internal/config/config.go` — Go `Default()` values
  - `internal/config/config_test.go` — TestDefault / TestLoad_PartialConfig / TestLoad_FullRoundTrip expectations
  - `docs/specs/2026-04-16-ralph-cli-tool.md` — TOML example in historical spec

Spec compliance, test coverage, and documentation drift are intentionally **out of scope** per skill contract (see `.claude/rules/post-implementation-pipeline.md`).

## Evidence reviewed

- `git show 4f5d5b5` — full diff (4 files, +15/−12)
- `git log --oneline -10 -- scripts/ralph-config.sh` — last touched in `5140f3a` (unrelated MEDIUM fix) and `9f42cca` (shared-config introduction); never set to `claude-sonnet-4-20250514`
- `git log --oneline -10 -- internal/config/config.go` — previous touch `d82a404`, before that `6c68acb` (parser introduction)
- `Grep "claude-sonnet-4-20250514"` → 0 matches (all old references removed)
- `Grep "claude-opus-4-7"` → 6 matches across the 4 changed files only (no orphaned references elsewhere)
- `Grep "xhigh"` → 7 matches across the 4 changed files only
- `Grep -i "effort"` → surfaced `scripts/ralph-config.sh:19 RALPH_EFFORT="${RALPH_EFFORT:-high}"` and `docs/recipes/ralph-loop.md:140 | RALPH_EFFORT | high | …` — unchanged (see Finding H-1)
- `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` → IDENTICAL (root/template mirror still consistent)
- `cmp docs/recipes/ralph-loop.md templates/base/docs/recipes/ralph-loop.md` → IDENTICAL
- `internal/cli/run.go:56-57` — `cfg.Pipeline.Model` / `cfg.Pipeline.Effort` are exported as `RALPH_MODEL` / `RALPH_EFFORT` into the shell environment before invoking the orchestrator
- `./scripts/run-verify.sh` passes — `docs/evidence/verify-2026-04-21-015245-ralph-default-model.log` (verifier will confirm)

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| HIGH | maintainability | **Defaults drift between Go config and shell config.** `internal/config/config.go:43-44` now sets `Model: "claude-opus-4-7"` / `Effort: "xhigh"`, but `scripts/ralph-config.sh:18-19` still holds `RALPH_MODEL="${RALPH_MODEL:-opus}"` / `RALPH_EFFORT="${RALPH_EFFORT:-high}"`. When users invoke `ralph run` via the Go CLI, `internal/cli/run.go:56-57` overrides the shell defaults (correct path). But direct invocations such as `./scripts/ralph-loop.sh`, `./scripts/ralph run` (pre-Go-CLI wrapper), or the test in `tests/test-ralph-config.sh:53-57` that pins `default RALPH_EFFORT = high` will continue to use `high`, not `xhigh`. The two source-of-truth values now disagree for the same env var. | `scripts/ralph-config.sh:18-19`, `internal/cli/run.go:56-57`, `tests/test-ralph-config.sh:53-57`, `docs/recipes/ralph-loop.md:139-140` | Either (a) sync `ralph-config.sh` + `tests/test-ralph-config.sh` + `docs/recipes/ralph-loop.md` to `xhigh` in the same PR, or (b) explicitly document in the commit body / tech-debt that shell-entry defaults are intentionally left behind and give the reason. Today neither is done. |
| MEDIUM | unnecessary-change | **`TestDefault` gains an assertion that was absent before.** The diff adds `if cfg.Pipeline.Effort != "xhigh" { … }` (`internal/config/config_test.go:14-16`). There is no corresponding assertion for `PermissionMode`, `MaxParallel`, `SliceTimeout`, etc. — so this test is now lopsided (the only string field it pins in addition to `Model` is `Effort`). It is correct and harmless, but the asymmetry is a small readability wart: a future reader may wonder why `Effort` is singled out. | `internal/config/config_test.go:14-16` vs the rest of `TestDefault` | Low-priority: either leave as-is (ties the new default to a regression test — defensible) or extend `TestDefault` to assert the full `Default()` struct with a table/compare. Not a merge blocker. |
| LOW | consistency | **Mixed model-ID formats in the same test file.** The new default uses the dated-suffix-less `claude-opus-4-7` (per task context, the 4.6+ convention), but `TestLoad_PartialConfig` still uses the legacy dated form `claude-opus-4-20250514` as its sample override value (`config_test.go:44, 57-58`). Functionally fine — any string is valid as an override — but a reader skimming the test file sees both conventions side by side with no explanation. | `internal/config/config_test.go:44, 57-58` | Optional: change the sample override to another dated-less ID (e.g. `claude-sonnet-4-6`) OR add a one-line comment noting the override accepts any string. Not a merge blocker; leaving avoids scope creep. |

Total: **0 CRITICAL**, **1 HIGH**, **1 MEDIUM**, **1 LOW**.

### Notes on items deliberately NOT flagged

- **Spec-doc update (`docs/specs/2026-04-16-ralph-cli-tool.md:238-239`).** The task context states the spec is treated as a historical design doc whose TOML example shows shipping defaults. Keeping the example in sync with the ship values is consistent with that framing, and the change is a clean 2-line replacement with no semantic drift. No finding.
- **Commit message format.** `chore: set ralph default model to claude-opus-4-7 and effort xhigh` follows the Conventional Commits format defined in `.claude/rules/git-commit-strategy.md`. Body cites evidence file, which matches house style. No finding.
- **Secrets / debug code / exception handling.** N/A — diff is 4 string-literal replacements plus one new test assertion. No secrets, no debug prints, no error paths touched.
- **Null safety.** N/A — `Default()` returns a value-type struct; the changed fields are non-pointer strings.
- **Naming / readability of the new values themselves.** `claude-opus-4-7` matches the existing grep-able convention (`claude-opus-4-6`, `claude-sonnet-4-6` format). `xhigh` is consistent with the lowercase, no-separator effort levels accepted by `claude -p` (`low`/`medium`/`high`/`xhigh`).

## Positive notes

- Scope is tight: 4 files, +15/−12, all changes are semantically tied to the single decision (switch defaults).
- The new `Effort: "xhigh"` assertion in `TestDefault` is the right place to lock in the new default — it catches silent regression in one line.
- `templates/base/ralph.toml` and `internal/config/config.go` are kept byte-for-byte consistent on the `[pipeline]` header/ordering, preserving the "shipping scaffold == Go defaults" invariant.
- Verify evidence (`docs/evidence/verify-2026-04-21-015245-ralph-default-model.log`) is attached in the commit body — matches the project's evidence-over-confidence rule.
- No collateral whitespace / reformatting noise in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Two independent default tables for `RALPH_MODEL` / `RALPH_EFFORT` (Go `config.Default()` vs shell `ralph-config.sh`). | Users entering through the shell path silently get `opus` / `high`; users entering through Go CLI get `claude-opus-4-7` / `xhigh`. Same env var, two answers depending on entry point. | This PR was scoped to "defaults update" and only touched the Go side. Shell side is the fallback for standalone `./scripts/ralph-loop.sh` usage. | Next time any ralph default changes, OR when `tests/test-ralph-config.sh` starts failing against production usage, OR before cutting a release that advertises the new defaults. | This report (HIGH finding H-1). If addressed separately, cross-link here. |

_(If the H-1 finding is addressed in this same PR, delete this tech-debt row. If deferred, also append to `docs/tech-debt/README.md`.)_

## Recommendation

- **Merge:** YES (no CRITICAL), **with caveat** — H-1 is a real drift and should be either (a) fixed in this PR by also bumping `scripts/ralph-config.sh` + `tests/test-ralph-config.sh` expectations + `docs/recipes/ralph-loop.md` table, or (b) tracked in `docs/tech-debt/README.md` with an explicit "shell defaults intentionally left at `high`" note before merge.
- **Stop condition (CRITICAL findings):** not triggered — pipeline may proceed to `/verify`.
- **Follow-ups:**
  1. Decide on H-1 resolution (sync shell defaults OR document deferral).
  2. Optional: M-1 symmetry in `TestDefault` (future cleanup, not this PR).
  3. Optional: L-1 model-ID format mix in `TestLoad_PartialConfig` (future cleanup, not this PR).
