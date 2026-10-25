# Review: hooks-matcher (reviewer seat, TASK t-2)

- Reviewed commit: 1c31dcf959925b7c61784711862678cb387ebf26
- Base: ffb3e271b92c54aa7f21bc3548ac3530882c79f4 (merge-base HEAD origin/main)
- Plan: docs/plans/active/2026-10-25-hooks-matcher.md
- Files: internal/cli/doctor.go, internal/cli/doctor_hooks_test.go

## Gate

GATE: pass

| Script | Result | Evidence |
|--------|--------|----------|
| `./scripts/run-static-verify.sh` | rc=0, all verifiers passed (gofmt ok, 0 issues, branch secret scan clean; staticcheck not installed, skipped) | docs/evidence/verify-2026-10-25-024320.log |
| `./scripts/run-test.sh` | rc=0, every Go package ok (internal/cli 353s, internal/org 435s) | docs/evidence/verify-2026-10-25-024340.log |

The two new test functions ran on their own: 13 subtests passed.

## Acceptance criteria

| # | Criterion | Verdict | Evidence |
|---|-----------|---------|----------|
| 1 | (2) A non-string `matcher` gives a finding; an absent or string `matcher` gives none | Met | doctor.go:501-506 (`present` check, then string assertion); TestValidateCodexHooksJSON_MatcherType covers absent, string, empty string (no finding) and number, array, object, bool, null (finding) |
| 2 | (1) The config.toml coexistence warning is skipped when the only `[hooks]` child key is `state`, and kept for any other key | Met | doctor.go:376-380 and `hasNonStateHooksKey` at doctor.go:404-413; TestCheckCodexEffectiveConfig_HooksTableCoexistence covers state only (no warning), state plus another key and another key only (warning) |
| 3 | Both the firing and the silent case are tested | Met | the two tests above; the matcher test uses a unique finding string, and the coexistence test also checks `Status == "pass"` for the silent cases |

Doc drift: a grep for both finding strings found no document that quotes them outside the plan and an old self-review report.

## Findings

| Severity | Finding | Evidence |
|----------|---------|----------|
| LOW | An empty `[hooks]` table used to warn and is now silent, because the loop has no non-state key. The plan lists an empty table as an edge case but states no expected result, and the commit has a test for it. Silence is reasonable (nothing duplicates hooks.json), but the plan does not say so. | doctor.go:404-413; doctor_hooks_test.go case "empty hooks table" |
| LOW | The "empty hooks table" test case sets `wantWarn: false`, which is already the zero value. The explicit field is harmless and states intent. | doctor_hooks_test.go (coexistence cases) |

No CRITICAL, HIGH or MEDIUM findings. Code reads cleanly: names are grep-able, the helper is small and commented, error text follows the existing `hooks.json %s[%d]...` format, and the change stays inside the two files in the plan's Affected areas.

Counts: CRITICAL 0, HIGH 0, MEDIUM 0, LOW 2.

## Seat rules

The review edited no code and no plan. `docs/tech-debt/README.md` is untouched. The tests of the commit do not call herdr, agmsg, gh, osascript or the network; the gate scripts ran as shipped.

Merge recommendation: no blocker from this review.
