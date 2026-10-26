# Cross-review triage report: leader-spawn-pr-rules

- Date: 2026-10-26
- Plan: docs/plans/active/2026-10-26-leader-spawn-pr-rules.md
- Base branch: main (origin/main at 619cf5f8)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-26-leader-spawn-pr-rules.md. It fixes two findings of the org recheck that the user chose to fix before the release: RC2 (the leader template's seat spawns lack `--scope` and are refused under the default `autonomous` permission mode) and RC5 (the feature orgs' PRs did not follow the `/pr` skill's language rule). The approved digest is 7fb9a3d2e3cf. The branch changes:
  - `internal/org/feature.go`: `featureLeaderTask` writes the task's reserved paths line with the seats' `--scope` argument, quoted by `shellQuote` (efddd301)
  - `internal/org/prompts/leader.md`: item 2 of the mission and steps 3, 5, and 14 of the feature-org part, with tests in `internal/org/feature_test.go` and `internal/org/prompts_test.go` (efddd301)
  - the `/org` skill's leader cycle step 2 and org の終わり方 step 6, in all four copies (c654d596)
  - records: the RC2 and RC5 tech-debt rows marked RESOLVED, the spec's FR-4 and acceptance criterion 5 kept `[ ]` with marks dated 2026-10-26, and the spec evidence table (7d51236d, wording fixed in 0309acf8)
- Self-review report: docs/reports/self-review-2026-10-26-leader-spawn-pr-rules.md (43dc55e4). Merge: OK. LOW F1 (the records said "confirmed by tests" more strongly than the tests pin) and LOW F2 (RC12 listed among the open findings) were fixed in 0309acf8, within cycle 1 and before /verify, and the report notes it.
- Verify report: docs/reports/verify-2026-10-26-leader-spawn-pr-rules.md (2947c524). Verdict: pass. LOW V1 to V3 (three tech-debt rows whose triggers this branch hit, with no carry-over note) were handled by /sync-docs.
- Test report: docs/reports/test-2026-10-26-leader-spawn-pr-rules.md (1e2cf34f). Verdict: pass. run-test.sh, and go test of `./internal/org/... ./internal/cli/...` under go1.26.0 and go1.25.8, all exited 0 on 2947c524. All 12 mutants (six of the task line, six of the template sentences) failed the tests.
- Sync-docs report: docs/reports/sync-docs-2026-10-26-leader-spawn-pr-rules.md (8e6bec75). No doc drift; carry-over notes added to the RC4 row, the file-size row, and the org-feature-worktree row's (g).
- Implementation context summary:
  - ralph quotes the argument and the leader copies it unchanged, because `normalizeReservePath` lets `$`, backticks, and quotes through (design decision D6, from the Codex plan advisory).
  - No live run: whether a leader passes the argument as written and writes the PR in Japanese is unverified (plan R3), so FR-4 and acceptance criterion 5 stay `[ ]` until the next run.
- Reviewer run: `codex exec review --base main` with `sandbox_mode=read-only` (model gpt-6-astra, reasoning effort xhigh) exited 0 and wrote a non-empty `-o` file (231 bytes). Its verdict: no actionable regressions. It reported that formatting, whitespace, the four skill copies' equality, and a shell-quoting probe over the seven reservation fixtures passed. Its targeted `go test` could not start because the read-only sandbox blocked the build's temporary directory; the tests ran in /test instead.
- The main checkout and the task worktree stayed clean after the run (`git status --short` empty in both).

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
