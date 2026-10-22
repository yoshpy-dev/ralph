# Cross-review triage report: split-preamble

- Date: 2026-10-23
- Plan: docs/plans/active/2026-10-22-split-preamble.md
- Base branch: main (local main and origin/main both at a7bdc884 before the run)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-22-split-preamble.md (the fifth PR of the trial-fix series: F5, the split plan's preamble that `ralph org start --plan` did not hand to the leader, plus the records that turn `fix/seat-wip-checkpoint` into #250; approved digest f0735c9a1ac6, unchanged through the pipeline)
- Self-review report: docs/reports/self-review-2026-10-22-split-preamble.md (Merge OK; LOW L1 to L4 fixed in b09b7c12 and c0ff8398)
- Verify report: docs/reports/verify-2026-10-22-split-preamble.md (Verdict: pass; INFO I1 and I3 fixed in 3b21b51b, I2 recorded in the plan)
- Test report: docs/reports/test-2026-10-22-split-preamble.md (Verdict: pass on go1.26 and go1.25.8; 38 of 41 mutants red; G1 and G2, a feature with `- Depends on:` and trailing spaces on the preamble's last line, filled in 7a955d2c, which made the 3 surviving mutants red; the internal/cli test that failed once under load is a race that predates this branch, recorded in tech-debt in 33e51acc)
- Sync-docs report: docs/reports/sync-docs-2026-10-22-split-preamble.md (30068218; no drift found, `check-skill-sync.sh` and `check-sync.sh` rc 0)
- Implementation context summary: `parseSplitPlan` (internal/org/split.go) reads the `## ` sections between the header and `## Features` into `SplitPlan.Preamble`, with fences kept whole and blank lines around it trimmed; the header's free text, the lines of `## Features` before the first `### <slug>`, and the sections after `## Features` still go to no leader and are not refused (tech-debt row). `featureLeaderTask` (internal/org/feature.go) puts the lead line 全機能の座席に適用する決まり and the preamble in a backtick fence one longer than its longest run (at least 3) between the task's lines and the feature's body, and adds nothing for a blank preamble. `RenderRolePrompt` substitutes in one `strings.NewReplacer` pass, so a `{{ORG_ID}}` inside the preamble stays literal. Steps 1, 3, and 5 of the 機能ごとの org part of `internal/org/prompts/leader.md` have the leader copy the block into `## 全機能の座席に適用する決まり` of its feature plan and spawn with the driver and model it names; nothing in ralph checks that (tech-debt row). The /org and /director skills (4 copies each), spec FR-4, the README, tech-debt, the evidence table, and the spec's PR list (33) were updated. The reviewer ran `codex exec review --base main` with `sandbox_mode=read-only`, exited 0, and wrote a non-empty `-o` file: no actionable regressions in parsing, prompt construction, or the leader's guidance; its whitespace, Go formatting, and skill mirror checks passed, and its focused Go tests could not run because the read-only sandbox blocked the build directory. The tester ran `run-test.sh` (rc 0 on the second run) and the org and cli packages on go1.25.8, the implementer of 7a955d2c ran `internal/org` in full, and `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` runs at e9824d39 alongside this review; the commits after 7a955d2c change reports, the plan's checklist, the tech-debt row, and the insight events only.

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
