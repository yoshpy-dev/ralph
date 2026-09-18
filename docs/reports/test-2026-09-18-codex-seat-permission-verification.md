# Test report: codex-seat-permission-verification

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-18-codex-seat-permission-verification.md`
- Related issue: #155
- Verdict: **PASS**

## Scope

Changed-language scope for this task: Go (`internal/org/permissions.go`
error-string edit, `internal/org/permissions_test.go` assertion updates,
comment-only edit in `internal/config/config.go`) plus markdown/shell
comments (`scripts/ralph-config.sh` + `templates/base/scripts/ralph-config.sh`
comment only). No logic changes to test; the plan's Non-goal is "no
`permissions.go` logic changes" and this run confirms that by inspection
(below) in addition to running the tests.

Per the handoff, this report covers only behavioral tests. No static
analysis, linting, or formatting was run here — that is `/verify`'s job and
is already covered by `docs/reports/verify-2026-09-18-codex-seat-permission-verification.md`.

## What ran

| # | Command | Result |
|---|---------|--------|
| 1 | `go test ./internal/org/ -run 'Permission' -count=1 -v` | PASS (13 test functions / subtests, 0 failures) |
| 2 | `go test ./internal/config/ -count=1` (`TestDefaultsLockStep` isolated first, then full package) | PASS |
| 3 | `./scripts/run-test.sh` (changed-language scope) | PASS — exit 0, "All verifiers passed." |
| 4 | `go test ./internal/... -count=1` (fresh, uncached) | PASS — 8/8 packages ok |
| 5 | `go test ./internal/org/ -cover -count=1` | PASS — 89.1% coverage |

## Per-test results

### 1. `go test ./internal/org/ -run 'Permission' -count=1 -v`

All permission-mode tests pass, 0.513s:

- `TestEnvelopeSummary_ListsModelPoolMaxSeatsAndPermissionDefault` — PASS
- `TestEnvelopeSummary_BlankPermissionDefault_FallsBackToPackageDefault` — PASS
- `TestResolvePermissionMode` (4 subtests: default/override/unknown-role/zero-value) — PASS
- `TestPermissionArgsForDriver_Claude` (3 subtests: autonomous/edits/guarded) — PASS
- `TestPermissionArgsForDriver_Codex_FailClosed` (4 subtests) — PASS
  - `guarded_is_allowed_with_no_flags`
  - `autonomous_is_rejected_fail-closed_when_codex_verified_is_false`
  - `edits_is_rejected_fail-closed_when_codex_verified_is_false`
  - `an_unknown_mode_is_reported_distinctly_from_the_fail-closed_case`
- `TestPermissionArgsForDriver_Codex_VerifiedUnlocksMapping` (3 subtests) — PASS
- `TestPermissionModeFromDetails_ExtractsFragmentOrEmpty` — PASS
- `TestOrgSpawn_PermissionMode_Claude_ArgvMapping` (3 subtests) — PASS

Confirmed the fail-closed tests exercise the new error substring: at
`internal/org/permissions_test.go:94` the autonomous/edits fail-closed
subtests assert `strings.Contains(err.Error(), "requires [org.permissions].codex_verified=true")`
(plus `"guarded"`). At `internal/org/permissions_test.go:113` the
unknown-mode distinctness subtest asserts the **opposite** —
`!strings.Contains(err.Error(), "requires [org.permissions].codex_verified=true")`
— which still discriminates the unknown-permission-mode error from the
fail-closed codex error after the string change, so the test's stated
intent (distinct error paths) still holds.

### 2. `go test ./internal/config/ -count=1`

`TestDefaultsLockStep` isolated with `-run 'TestDefaultsLockStep' -v` — PASS
(0.00s). Full package re-run — PASS, 0.659s (fresh sweep) / 0.236s (isolated
run). `scripts/ralph-config.sh` was touched only in a comment
(the `codex_verified` comment now points at the recipe instead of the
removed tech-debt line), so the lock-step defaults values are unchanged —
confirmed by the pass with no diff needed.

### 3. `./scripts/run-test.sh`

Mode: `test`, requested scope: `changed`. Ran 27 shell test suites (fully
enumerated header list, including `test-xreview-helpers.sh` at 29
assertions) — every suite reported `FAIL: 0` — followed by the Go verifier
fallback (`Language scope: full fallback` because `scripts/ralph-config.sh`
carries an unclassified extension for the language-scope detector; this
just means the Go pack ran too, which it needed to). Go verifier output:
all 8 packages `ok` (5 cached from step 1/2/4's runs, `internal/config` and
`internal/scaffold` re-ran fresh). Final line: `All verifiers passed.`
Evidence log: `docs/evidence/verify-2026-09-18-075849.log` (gitignored,
written by the wrapper itself).

Exit code captured directly (not through `tee`, per this repo's own
guidance against masking exit codes with `tee`): `EXIT:0`.

### 4. `go test ./internal/... -count=1` (fresh, uncached)

All 8 test-bearing packages `ok`:

| Package | Result | Time |
|---|---|---|
| `internal/cli` | ok | 39.564s |
| `internal/config` | ok | 0.659s |
| `internal/insights` | ok | 1.076s |
| `internal/org` | ok | 11.058s |
| `internal/org/driver` | ok | 2.123s |
| `internal/org/protocol` | ok | 1.585s |
| `internal/scaffold` | ok | 2.371s |
| `internal/upgrade` | ok | 3.162s |

`internal/cli` and `internal/org` ran in the same sweep as each other here
(both known occasional flakes under adjacent subprocess-heavy test load,
per prior test-agent memory) and both passed cleanly with no flake.

### 5. `go test ./internal/org/ -cover -count=1`

`coverage: 89.1% of statements`, 6.688s — matches the last recorded
`internal/org` baseline for this repo (org-implementer-seat-envelope cycle
2, 2026-09-17/18), confirming this plan's comment-only + error-string edit
did not move coverage.

## Inspection: permissions.go non-comment diff

Per the handoff, confirmed by inspection only (no edits made):

```
git diff main -- internal/org/permissions.go | grep -E '^[+-]' | grep -v '^+++\|^---' | grep -v '^[+-]//'
```

Output — exactly two changed lines, both `fmt.Errorf` call sites (autonomous
and edits branches of the codex fail-closed path), both changed to the same
new message text:

```
-…"org: codex seat permission mode %q not yet live-verified; only guarded is allowed (fail-closed; set [org.permissions].codex_verified=true after live-verifying your codex CLI's flags)"…
+…"org: codex seat permission mode %q requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first: docs/recipes/codex-seat-permissions.md)"…
```

(x2, identical replacement in both branches). No control-flow, condition, or
return-value-shape changes. This matches the plan's Deviation notes
("ロジック不変の文言変更として受け入れた") and AC-4's assertion. AC-5's
checkbox (`go test ./internal/org/... -count=1` and
`./scripts/run-verify.sh` green) predates this error-string change per the
verify report's own caveat — this test run supersedes that stale tick with
a live re-run after the string change, and it is green.

## Coverage gaps (known, pre-existing — not introduced by this change)

- The live E2E verification itself (5 codex seat spawns: autonomous pass,
  edits pass-with-caveats across 3 runs) is **integration evidence**, not
  unit-tested. It lives in `docs/evidence/codex-seat-permissions-2026-09-18.md`
  and is not re-run by this suite — re-running it would require a live
  codex CLI, herdr, and agmsg session, which the plan's Non-goals
  explicitly excludes from CI/automated re-verification.
- No new unit test exists asserting the exact new error string's presence
  of the recipe path (`docs/recipes/codex-seat-permissions.md`) — the
  existing assertions check for `"requires [org.permissions].codex_verified=true"`
  and `"guarded"` substrings only, not the trailing recipe pointer. This is
  a pre-existing test-precision gap (also true of the old string, which
  wasn't asserted in full either) and not something this plan's scope
  (comment/doc-positioning only) was expected to add.

## Flaky test notes

None observed this run. `internal/cli` and `internal/org` (previously
flagged as occasional subprocess-timeout flakes under adjacent
subprocess-heavy test load) both passed cleanly in the fresh full sweep
(step 4) run together in the same `go test ./internal/...` invocation.

## Verdict

**PASS.** All specified tests ran and passed: the full permission-mode
table in `internal/org`, the config lock-step test, the changed-language
`run-test.sh` wrapper (27/27 shell suites plus the Go verifier), and a
fresh full `go test ./internal/...` sweep (8/8 packages). The
`permissions.go` diff is confirmed comment/error-string only, matching the
plan's stated deviation. Safe to proceed to `/sync-docs` and beyond.
