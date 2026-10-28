# Cross-review triage report: drop-org-shell-defaults

- Date: 2026-10-28
- Plan: docs/plans/active/2026-10-28-drop-org-shell-defaults.md
- Base branch: main (local main at 4389396f; the branch's merge base is 15e301c8)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-28-drop-org-shell-defaults.md (issue #273). It removes the copies of the `[org]` defaults (18 `RALPH_ORG_*` variables and 3 export lines) from `scripts/ralph-config.sh` and its template copy, keeping only `RALPH_ORG_AGMSG_HOME` (unexported), and makes `internal/config/defaults_sync_test.go` check `config.Default()` against `templates/base/ralph.toml` for 19 `[org]` keys. The approved digest is 2ec5763f7abf.
- Self-review report: docs/reports/self-review-2026-10-28-drop-org-shell-defaults.md. Merge: yes. L1, L2, L3, and L5 were fixed in 7d59e278. L4 (the count of 19 keys lives only in comments) is left for the PR's Known gaps.
- Verify report: docs/reports/verify-2026-10-28-drop-org-shell-defaults.md (ecf31b75). Verdict: pass. INFO D-1 and D-2 were fixed in the sync-docs commit 097c7f35. D-3 (the spec's note (c)) is still accurate and stays. D-4 (the plan's "2 export lines" against the 3 removed) is recorded under the plan's Progress checklist.
- Test report: docs/reports/test-2026-10-28-drop-org-shell-defaults.md (339e465f). Verdict: pass. The first `./scripts/run-test.sh` failed only on the known flaky `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` in `internal/org`, which this branch does not touch. The re-run passed. Its Test gaps section records that a swap of two keys with the same default value, or the removal of one comparison line, passes the committed tests. The tester's throwaway 19-row probe passed at HEAD, so the wiring of all 19 keys is correct today. The table test goes into #274, which changes `Default()`.
- Sync-docs report: docs/reports/sync-docs-2026-10-28-drop-org-shell-defaults.md, commit 097c7f35.
- Implementation context summary:
  - `orgTemplateDrift` reads the 19 keys through a probe struct of pointer fields. go-toml/v2 leaves a field nil when its key is absent, so a key removed from the template is reported as missing even though `Load` starts from `Default()`.
  - `TestDefaultsLockStep` has two subtests. `org` compares the template with `Default()`. `shell_and_skills` compares `RALPH_ORG_AGMSG_HOME` in `scripts/ralph-config.sh` with `Default()` and keeps the SKILL.md fallback checks.
  - No Go code outside tests reads the removed variables. The ralph binary reads `RALPH_ORG_AGMSG_HOME` from the environment (`driver.ResolveAgmsgHome`), which is why that line stays.
- Reviewer run: `codex exec review --base main` with `sandbox_mode=read-only` exited 0 and wrote a non-empty `-o` file. Its verdict, quoted: "No actionable regressions were found. The removed variables have no remaining runtime consumers, and the replacement checks preserve coverage while detecting missing template keys. All 19 shell tests and static checks passed; targeted Go tests were blocked by the read-only sandbox." The Go tests ran in /test instead.
- The main checkout stayed clean after the run (`git status --short` empty).

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
