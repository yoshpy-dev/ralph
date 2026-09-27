# Verify report: doctor-agmsg-store-in-project

- Date: 2026-09-27
- Plan: docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md
- Verifier: verifier subagent (cycle 1)
- HEAD: 7af3ff229e4594cf29015e6995ec087900bde392 (branch fix/doctor-agmsg-store-in-project)
- Worktree: /Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/doctor-agmsg-store-in-project
- `git status --porcelain` empty before this run

## Static analysis

- `./scripts/run-static-verify.sh` (changed-language scope): PASS — gofmt ok, `go vet`/golangci-equivalent 0 issues, `scripts/check-sync.sh` 0 DRIFTED (159 IDENTICAL, 11 TEMPLATE_ONLY, 5 KNOWN_DIFF — all pre-existing), `scripts/check-skill-sync.sh` 13 skills in lock-step, `scripts/check-template-purity.sh` clean, branch secret scan (`88e7429..7af3ff2` vs `origin/main`) clean. Evidence: `docs/evidence/verify-2026-09-27-045139.log`.
- `gofmt -l internal/cli` — no output (clean).
- `go vet ./internal/cli/...` — no output (clean).
- `scripts/check-skill-sync.sh`, `scripts/check-sync.sh`, `scripts/check-template-purity.sh` — re-run directly (not just inside the wrapper): all PASS, same result as above.
- gopls modernize hints: `gopls check internal/cli/doctor_codex_writable_root.go internal/cli/doctor.go internal/cli/doctor_codex_writable_root_test.go internal/cli/doctor_codex_writable_root_unix_test.go` — no output on any of the four touched files. Nothing lands on lines this branch added.

## Spec compliance (AC-1 .. AC-8)

| AC | Verdict | Evidence |
|----|---------|----------|
| AC-1 (in-project warn narrows both config-absent/exists Details; no unconditional "cannot send" or "add" survives) | PASS | `internal/cli/doctor_codex_writable_root.go:817-846` (`codexWritableRootDetail`'s two `if containingProject != ""` branches, lines 822-829 and 835-840); wiring `internal/cli/doctor.go:150-154` (`projectDir` = `filepath.Abs(targetDir)`, `""` on error); test `TestCheckCodexAgmsgWritableRoot_StoreInsideProject_Warn` (`internal/cli/doctor_codex_writable_root_test.go:2026-2073`) asserts both the "config absent" and "config exists" subtests contain the in-project phrasing and asserts absence of `codexWorkspaceWriteUnconditionalPhrase` (the "cannot send RESULT" half, `:1919`) **and** a formatted `codexAddWritableRootUnconditionalPhrase` (the "add …" half, `:1928`) — closing the self-review's coverage gap that only the "cannot send" half was negatively asserted. Wiring itself is pinned end-to-end (not just at the check-function level) by `TestRunDoctorOpts_CodexSandboxCheck_StoreInsideProjectThroughTheSeam` (`:1814-1878`), which runs through `runDoctorOpts` and would fail if `doctor.go:150-154` reverted to passing `""`. |
| AC-2 (store under project's `.agents`/`.git`/`.codex` gets no supplement) | PASS | `internal/cli/doctor_codex_writable_root.go:1003-1009` (`containingProject` only set when `codexRootCoverage(projectDir, store, ...)` returns `covers && !blocked`, reusing the existing protected-directory rule, no new logic); `TestCheckCodexAgmsgWritableRoot_StoreUnderProjectProtectedDir_NoSupplement` (`:2074-2109`, table-driven over `.agents`/`.git`/`.codex`) asserts byte-identical Detail between `projectDir=""` and `projectDir=project`; `TestCheckCodexAgmsgWritableRoot_ProjectDotAgentsSymlinked_NoSupplement` (`:2110-2130`) covers the symlinked-`.agents` variant of the same rule. |
| AC-3 (store outside project: Detail unchanged, existing tests pass) | PASS | Diffed `codexWritableRootDetail`'s `containingProject == ""` branches against `git show main:internal/cli/doctor_codex_writable_root.go`'s pre-change function: the two `!exists`/`exists` format strings and argument lists are byte-identical to main (confirmed by direct `diff` of the two function bodies with only the two new `if containingProject != ""` blocks as the delta). `TestCheckCodexAgmsgWritableRoot_StoreOutsideProject_NoInProjectPhrase` (`:2131-2152`) asserts the in-project phrase is absent and the original wording remains. All 51+2 pre-existing call sites across `doctor_codex_writable_root_test.go` and `doctor_codex_writable_root_unix_test.go` were mechanically updated to pass `""` for the new `projectDir`/`containingProject` params (confirmed by reading the unix-test diff: only the added `""` argument, no other change), which reproduces the prior behavior by construction. |
| AC-4 (pass and info Details unchanged) | PASS | The only caller of `codexWritableRootDetail` on the pass path (`:988`, explicit root) passes `containingProject=""` unconditionally, so that Detail is structurally unaffected. `codexImplicitRootDetail` (implicit-root pass) and all `info`-path `Detail` assignments (`:925-966`) take no new parameter and are untouched by this diff — confirmed by `git diff main...HEAD -- internal/cli/doctor_codex_writable_root.go` showing no hunk inside those blocks. |
| AC-5 (symlinked project directory, e.g. macOS `/var` vs `/private/var`, gives the same AC-1/AC-2 verdict) | PASS | `TestCheckCodexAgmsgWritableRoot_ProjectDotAgentsSymlinked_NoSupplement` (`:2110-2130`, AC-2 via a symlinked `.agents`) and `TestCheckCodexAgmsgWritableRoot_ProjectDirSymlinked_SameAsResolved` (`:2153-2178`, AC-1 via a symlinked project directory itself) both compare the direct and symlinked forms and assert the same wording/status. Reuses `codexRootCoverage`'s existing resolved+configured pairing logic (`:1005`), which already handles symlink resolution for explicit `writable_roots` entries — no new symlink-handling code was written for this plan. |
| AC-6 (`AGMSG_STORAGE_PATH` inside the project also gets AC-1) | PASS | `TestCheckCodexAgmsgWritableRoot_StorageOverrideInsideProject_Warn` (`:2184-2209`): `agmsgHome` is unrelated to the project, `AGMSG_STORAGE_PATH` override lands inside it, and the in-project wording still appears. `TestCheckCodexAgmsgWritableRoot_StoreEqualsProjectDir_Warn` (`:2210-2229`, the plan's explicit edge case) covers store == project directory itself. |
| AC-7 (recipe x2, `/org` skill x4, `check-skill-sync.sh`/`check-sync.sh` pass, tech-debt row updated) | PASS | `docs/recipes/codex-seat-permissions.md` and `templates/base/docs/recipes/codex-seat-permissions.md` are byte-identical (`cmp` clean) and both add the same sentence, including the `.git`/`.agents`/`.codex` exception. `.claude/skills/org/SKILL.md`, `.agents/skills/org/SKILL.md`, `templates/base/.claude/skills/org/SKILL.md`, `templates/base/.agents/skills/org/SKILL.md` are all byte-identical (`cmp` clean, four-way) with the matching Japanese sentence and the same exception. `scripts/check-skill-sync.sh` and `scripts/check-sync.sh` both PASS (0 DRIFTED). `docs/tech-debt/README.md`'s `checkCodexAgmsgWritableRoot` row (line ~131) has a new sentence: "Since #170, a store inside the project narrows the warn's Detail … but does not flip the outcome to `pass` … a seat may run in a task worktree (e.g. `.claude/worktrees/<slug>`), whose working directory does not contain a store that lives elsewhere in the project" — matches the self-review's L4 fix (drops the unsupported "often" claim and the reversed containment direction). |
| AC-8 (`run-verify.sh` green, `go test ./internal/cli/...` ok, `secret-scan-branch.sh --strict` clean) | PARTIAL — static half PASS, behavioral half deferred to `/test` | `./scripts/secret-scan-branch.sh --strict` run directly: `scanned 88e7429..7af3ff2 against origin/main: clean`. `go test ./internal/cli/...` was **not** run here (behavioral tests are the tester's scope per the task assignment); `go vet ./internal/cli/...` (a strict compile-and-static-check superset) is clean, which at minimum confirms the package and its tests compile. |

## Non-goals check

- Status stays `warn`, never `pass`, for an in-project store: `checkCodexAgmsgWritableRoot`'s only path that computes `containingProject` (`:1003-1009`) runs after both the explicit-root pass (`:985-990`) and implicit-root pass (`:992-998`) checks have already returned; the function unconditionally sets `r.Status = "warn"` at `:1011` right after computing `containingProject`, so there is no code path where an in-project store produces `pass`.
- No attempt to read a seat's actual `--cwd`: `projectDir` is only ever `runDoctorFull`'s `targetDir` made absolute (`doctor.go:150`) or `""`; no new code in this diff reads `internal/org` seat state, `ralph.toml`'s `[org]` spawn history, or any process/pane inspection.
- No other check changed and status logic for this check is unchanged apart from the new narrowing: `git diff main...HEAD --stat` (below) touches only this check's file, its two test files, `doctor.go`'s call site, docs, and reports/insights — no other `internal/cli/doctor_*.go` file is in the diff.

## Documentation drift

- Recipe (`docs/recipes/codex-seat-permissions.md` + `templates/base/` copy) and `/org` skill (4 mirrors) — checked above under AC-7, no drift, byte-identical copies confirmed with `cmp`.
- `docs/tech-debt/README.md` — checked above under AC-7, row updated and consistent with the shipped behavior.
- Full-repo grep for `"agmsg writable root"` and `"Codex sandbox (agmsg"` across `docs/`, `.claude/`, `.agents/`, `templates/`, `README.md`: no hits outside the recipe, the four skill mirrors, the tech-debt row, this plan, its self-review, and historical archive/report files for prior plans (#164, #170's own artifacts) — no other current-state doc describes this check that would need updating.
- No spec file references `checkCodexAgmsgWritableRoot`, `containingProject`, or the "Codex sandbox (agmsg writable root)" check name in a way this change would invalidate.

## Diff hygiene

`git diff main...HEAD --stat`: 14 files — `internal/cli/doctor.go`, `internal/cli/doctor_codex_writable_root.go`, `internal/cli/doctor_codex_writable_root_test.go`, `internal/cli/doctor_codex_writable_root_unix_test.go` (the plan's affected code), `docs/recipes/codex-seat-permissions.md` + `templates/base/` copy, `.claude/skills/org/SKILL.md` + `.agents/skills/org/SKILL.md` + both `templates/base/` mirrors, `docs/tech-debt/README.md` (the plan's affected docs), plus `docs/plans/active/2026-09-26-doctor-agmsg-store-in-project.md`, `docs/reports/self-review-2026-09-27-doctor-agmsg-store-in-project.md`, `docs/insights/events/2026-09-27-doctor-agmsg-store-in-project.jsonl`. No unrelated files touched.

## Verdict

**PASS.**

## Follow-ups

- `/test`: run `TMPDIR=/tmp go test ./internal/cli/... -count=1` (behavioral verdict is out of this report's scope by task assignment); the self-review's identified coverage gaps (unpinned `doctor.go` wiring, missing negative "add" assertion) are already closed by `TestRunDoctorOpts_CodexSandboxCheck_StoreInsideProjectThroughTheSeam` and the `codexAddWritableRootUnconditionalPhrase` assertion in `TestCheckCodexAgmsgWritableRoot_StoreInsideProject_Warn` — the tester should confirm both pass rather than re-adding them.
- `/sync-docs`: no drift found; nothing to sync beyond what already shipped in this diff.

## Known gaps

- `go test ./internal/cli/...` itself was not executed in this verify pass (tester's scope); only compile-level confidence (`go vet`) was obtained here.
