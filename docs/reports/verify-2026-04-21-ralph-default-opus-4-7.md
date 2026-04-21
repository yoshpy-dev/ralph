# Verify report: ralph default → claude-opus-4-7 / xhigh

- Date: 2026-04-21
- Plan: (none — defaults-only update, no `docs/plans/active/*` entry; task intent comes from user context)
- Verifier: verifier subagent (verify skill)
- Scope: spec compliance + static analysis + documentation drift for branch `chore/ralph-default-opus-4-7` @ `8fbe203`
- Evidence: `docs/evidence/verify-2026-04-21-ralph-default-opus-4-7.log`

## Spec compliance

User-stated intent: `ralph init` should scaffold `claude-opus-4-7` / `xhigh` instead of `claude-sonnet-4-20250514` / `high`. Shell-entry fallbacks should agree.

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| Go `Default()` pipeline model is `claude-opus-4-7` | Verified | `internal/config/config.go:43` |
| Go `Default()` pipeline effort is `xhigh` | Verified | `internal/config/config.go:44` |
| Scaffold TOML (`templates/base/ralph.toml`) matches Go defaults | Verified | `templates/base/ralph.toml:4-5` (`model = "claude-opus-4-7"`, `effort = "xhigh"`) |
| `internal/config` unit-test expectations pinned to new defaults | Verified | `internal/config/config_test.go:11,14-16,78,101,68` (TestDefault asserts both; TestLoad_PartialConfig asserts effort `xhigh`; TestLoad_FullRoundTrip uses new model) |
| `scripts/ralph-config.sh` shell fallback `RALPH_MODEL:-claude-opus-4-7` | Verified | `scripts/ralph-config.sh:18` |
| `scripts/ralph-config.sh` shell fallback `RALPH_EFFORT:-xhigh` | Verified | `scripts/ralph-config.sh:19` |
| `templates/base/scripts/ralph-config.sh` mirrored identically to root | Verified | `cmp` → IDENTICAL; both files show new defaults on lines 18-19 |
| Ralph Loop recipe table (root + template mirror) reflects new defaults | Verified | `docs/recipes/ralph-loop.md:139-140` and mirrored `templates/base/docs/recipes/ralph-loop.md:139-140` |
| Spec doc TOML example (historical design doc) updated | Verified | `docs/specs/2026-04-16-ralph-cli-tool.md:238-239` |
| Shell test `tests/test-ralph-config.sh` defaults block asserts new values | Verified | `tests/test-ralph-config.sh:54,57` |
| Self-review HIGH finding (shell/Go default drift) addressed in-branch by commit `8fbe203` | Verified | self-review H-1 called out drift at `scripts/ralph-config.sh:18-19`; `git diff` shows both lines now hold `claude-opus-4-7` / `xhigh`; `tests/test-ralph-config.sh` expectations updated accordingly; root/template still mirrored |
| Runtime wiring: Go CLI exports `Model`/`Effort` into the shell env before invoking orchestrator | Likely but unverified (static) | `internal/cli/run.go:56-57` passes `RALPH_MODEL=...` and `RALPH_EFFORT=...`; actual exec behavior is tester scope |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-verify.sh` (fresh) | PASS | All stages OK; evidence log `docs/evidence/verify-2026-04-21-021403.log`; branch-specific evidence in `verify-2026-04-21-ralph-default-opus-4-7.log` |
| `gofmt` (via run-verify) | ok | 0 issues |
| `go vet` (via run-verify) | ok | 0 issues |
| `go test ./...` (via run-verify) | ok | All packages pass (cached); this is static-verifier scope only — behavioral confirmation belongs to `/test` |
| `scripts/check-sync.sh` | PASS | IDENTICAL:107, DRIFTED:0, ROOT_ONLY:0, KNOWN_DIFF:3 |
| `cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh` | IDENTICAL | Mirror invariant holds |
| `cmp docs/recipes/ralph-loop.md templates/base/docs/recipes/ralph-loop.md` | IDENTICAL | Mirror invariant holds |
| `grep -rn "claude-sonnet-4-20250514"` | 0 matches outside self-review narrative | Old model ID fully removed from active code and docs |
| `grep -rn "RALPH_MODEL:-opus"` | 0 matches outside self-review narrative | Old shell default fully removed |
| `grep -rn "RALPH_EFFORT:-high"` | 0 matches outside self-review narrative | Old shell default fully removed |
| `sh -n` on all 18 hook scripts | OK | Via run-verify |
| `jq -e .` on both `.claude/settings.json` files | OK | Via run-verify |
| `tests/test-check-mojibake.sh` | PASS (11/11) | Via run-verify |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/recipes/ralph-loop.md` defaults table | Yes | Row for `RALPH_MODEL` and `RALPH_EFFORT` updated to `claude-opus-4-7` / `xhigh`; override example `RALPH_MODEL=sonnet` on line 153 is still an override demo, not a default claim — consistent |
| `templates/base/docs/recipes/ralph-loop.md` | Yes | Byte-identical mirror of root (`cmp` OK) |
| `docs/specs/2026-04-16-ralph-cli-tool.md` | Yes | Historical design doc's shipping-defaults example now matches `templates/base/ralph.toml` |
| `templates/base/ralph.toml` | Yes | Shipping scaffold matches Go `Default()` byte-for-byte on changed lines |
| Go tests in `internal/config/config_test.go` | Yes | TestDefault, TestLoad_PartialConfig, TestLoad_FullRoundTrip expectations all updated |
| Shell tests in `tests/test-ralph-config.sh` | Yes | Defaults assertions updated to new values; override assertions untouched (correct — overrides are independent of defaults) |
| `AGENTS.md`, `CLAUDE.md`, `README.md` | N/A (no references) | No references to either old or new default model/effort in these files — nothing to drift |
| Tech-debt register (`docs/tech-debt/README.md`) | Yes | Self-review H-1 was the only drift item; it was fixed in-branch (commit 8fbe203) rather than deferred, so no tech-debt entry is required per the self-review's stated deletion rule |
| Stale references to `TestLoad_PartialConfig` using dated-suffix model ID `claude-opus-4-20250514` | Informational | Still present at `internal/config/config_test.go:44` as a sample override; self-review L-1 notes this is a non-blocking readability wart. Not drift — documented as accepted and correct (override path takes any string) |
| Self-review report's own narrative (`docs/reports/self-review-2026-04-21-ralph-default-opus-4-7.md`) | Stale but expected | Report still cites `scripts/ralph-config.sh:19 RALPH_EFFORT="${RALPH_EFFORT:-high}"` and recommends the H-1 fix; since fix landed in the follow-up commit `8fbe203`, the report is a point-in-time snapshot of review state, not a live claim about current code. This matches the project's recurring-blind-spots rule: self-review reports are not updated post-fix |

## Observational checks

- Runtime entry-point consistency: `internal/cli/run.go:56-57` exports `RALPH_MODEL` / `RALPH_EFFORT` from `cfg.Pipeline.*` into the child shell environment before invoking `scripts/ralph` / `ralph-pipeline.sh`. Because (a) Go `Default()` now returns `claude-opus-4-7`/`xhigh` and (b) the shell `:-` fallbacks also resolve to `claude-opus-4-7`/`xhigh`, both entry paths converge on the same values:
  - Go CLI entry: `Default()` → env export → `claude -p --model claude-opus-4-7 --effort xhigh ...`
  - Direct shell entry (`./scripts/ralph-pipeline.sh` without Go wrapper): shell fallback → `claude -p --model claude-opus-4-7 --effort xhigh ...`
- No hard-coded `--model` or `--effort` string literals were introduced; all downstream invocations in `scripts/ralph-pipeline.sh:146,159,300,335` and `scripts/ralph-loop.sh:114` consume the env vars via `"$RALPH_MODEL"` / `"$RALPH_EFFORT"` — static analysis confirms no bypass path.

## Coverage gaps

- Model ID `claude-opus-4-7` validity against the Anthropic Models API is asserted by the user context (dateless 4.6-era convention) but not verified against a live endpoint. Verifying the ID resolves at runtime is a `/test` concern, not `/verify`.
- The `claude -p --effort xhigh` invocation is exercised only indirectly: the shell tests check the env var's value, not whether the `claude` CLI accepts `xhigh`. This is expected static-analysis coverage; behavioral confirmation belongs to tester/CI.
- `TestLoad_PartialConfig`'s continued use of `claude-opus-4-20250514` as a sample override (self-review L-1) is informational; it is not a drift since the test asserts override behavior, not default behavior. No action required in this PR.

## Verdict

- Verified:
  - Go `Default()` values match user intent.
  - Scaffold (`templates/base/ralph.toml`) and Go defaults are consistent.
  - Shell fallbacks (root + template mirror) match Go defaults.
  - Unit/shell tests pinned to new values.
  - Documentation (recipes + spec example, root + mirror) updated.
  - Self-review H-1 drift fully resolved in-branch by `8fbe203`.
  - Static verifier (`./scripts/run-verify.sh`) passes end-to-end on fresh run.
  - No orphan references to old values in active code or docs.
  - Mirror invariants (`cmp` equality) preserved.
- Partially verified:
  - Runtime wiring in `internal/cli/run.go:56-57` is statically correct but execution-path validation is tester scope.
- Not verified (out of scope for `/verify`):
  - Whether Anthropic's API accepts `claude-opus-4-7` at runtime (tester + upstream reality).
  - Whether `claude -p --effort xhigh` is accepted by the installed `claude` CLI (tester).
  - Behavioral integration (Go → shell env export → pipeline) end-to-end (tester).

### Verdict: **PASS**

All acceptance criteria implied by the user's stated intent are met. Static analysis is green. Documentation is in sync (root + template mirrors, recipe tables, spec example, tests). The one HIGH finding from self-review (Go↔shell default drift) was resolved in the follow-up commit on the same branch, and the tech-debt entry the self-review would have added is correctly omitted because the debt was paid before merge.

No blockers. Recommend proceeding to `/test`.

## Recommended minimal next check (highest-confidence delta)

Single highest-value behavioral assertion for `/test`: run `go test ./internal/config/... -run 'TestDefault|TestLoad_PartialConfig|TestLoad_FullRoundTrip' -count=1` and `./tests/test-ralph-config.sh` (uncached, in that order). These directly exercise the three default-path invariants (Go Default, partial load with new effort, full round-trip with new model) plus the shell fallback chain in one cheap pass. If both pass, the defaults-change is behaviorally complete.
