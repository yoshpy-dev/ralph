# Cross-review triage report: org-narrow-verification

- Date: 2026-10-28
- Plan: docs/plans/active/2026-10-27-org-narrow-verification.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-27-org-narrow-verification.md
- Self-review report: docs/reports/self-review-2026-10-27-org-narrow-verification.md (cycle 2, run 2: LOW 5; R2-1..R2-4 fixed in F8 28d83b96, R2-5 recorded in tech-debt by sync-docs 6a5bed1e)
- Verify report: docs/reports/verify-2026-10-27-org-narrow-verification.md (cycle 2, pass)
- Test report: docs/reports/test-2026-10-27-org-narrow-verification.md (cycle 2, pass; `run-test.sh` rc 1 from the known flaky `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` only, `go test ./internal/org/ -count=1` re-run ok in 943.4 s)
- Reviewed HEAD: 6a5bed1e. `codex rc=0`, `-o` file 1,732 bytes, so the review is complete. The reviewer ran diff and formatting checks; its Go tests could not run in the read-only sandbox (no build directory).
- Cycle 1 (triage 8438af88): ACTION_REQUIRED 1, the implementer's secret cleanup could touch pre-existing out-of-scope changes. Fixed in F4 ffafc9d2; the cycle-2 reviewer did not raise it again.
- Implementation context summary: S1 and S3 made the narrowest check the default for a slice. For Go the templates spell it as `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && go test ./<dir>/`, run from the worktree root. Before this branch the slice ran `./scripts/run-verify.sh`, whose Go pack finds every `go.mod` and runs inside each module root, and whose `run_tests` puts `-timeout=30m` first in `GOFLAGS` (`packs/languages/golang/verify.sh:108-116`). The narrowed command dropped both.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] The bare `go test ./<dir>/` falls back to Go's 10-minute timeout instead of the Go pack's 30-minute default, so a package over ten minutes fails a valid slice verification and blocks its commit. Keep the pack's timeout and its caller-override order in the implementer fallback and in the commands the leader writes. | Real for this repository: the cycle-2 test report records `go test ./internal/org/ -count=1` passing in 943.4 s, past the 600 s default, and `internal/org` is the package most org slices here touch. The old slice command (`run-verify.sh`) had the 30-minute default (`packs/languages/golang/verify.sh:108-116`), so this is a regression the branch introduced. Worth fixing now: the fix is the command text in the templates, the skill copies, and the tests that pin it. | `internal/org/prompts/implementer.md` (`:37-40`, example `:118`), `internal/org/prompts/leader.md` (`:32-35`, example `:494`), `.claude/skills/org/SKILL.md` and its three copies (`:1330`), `internal/org/prompts_verification_test.go` (`implementerGoSliceCheck` and the example pins) |
| 2 | [P2] Run from the worktree root, `go vet ./service/` fails for a repository with `service/go.mod` and no root `go.work`, because Go does not enter a nested module. Resolve the module that contains the package and run the narrowed checks there, in the leader's commands and the implementer fallback. | Real: `tests/test-run-verify-scope.sh` covers the nested-module layout, and the Go pack runs inside each module root (`packs/languages/golang/verify.sh:20-35`, `verify_root`). The templates ship in the `ralph` binary and serve every scaffolded project, so a downstream repository with nested modules would see every Go slice fail verification. Worth fixing now: the same template sentences as finding 1, fixed together. | Same as finding 1 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
