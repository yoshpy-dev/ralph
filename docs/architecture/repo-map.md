# Repo map

## Core files

- `AGENTS.md`: vendor-neutral map
- `CLAUDE.md`: Claude-specific always-on additions
- `README.md`: human entry point

## Claude control plane

- `.claude/rules/`: path-scoped or topic-scoped guidance
- `.claude/skills/`: on-demand workflows
- `.claude/agents/`: specialized subagents
- `.claude/hooks/`: deterministic hook scripts
- `.claude/settings.json`: hook and permission configuration

## Process artifacts

- `docs/specs/`: spec files produced by `/spec`
- `docs/plans/active/`: in-flight plans
- `docs/plans/archive/`: completed plans
- `docs/reports/`: self-review, verify, test, and walkthrough reports
- `docs/quality/`: definition of done and gates
- `docs/tech-debt/`: known debt and follow-ups
- `docs/evidence/`: what counts as evidence

## Skills

- `.claude/skills/spec/`: refine vague ideas into detailed specifications (manual trigger)
- `.claude/skills/plan/`: create plans, select execution flow (auto)
- `.claude/skills/work/`: create branch and execute plans interactively (auto)
- `.claude/skills/loop/`: create worktree and set up Ralph Loop autonomous iteration (auto)
- `.claude/skills/self-review/`: self-review diff quality (auto)
- `.claude/skills/verify/`: spec compliance and static analysis (auto)
- `.claude/skills/test/`: behavioral tests (auto)
- `.claude/skills/codex-review/`: cross-model second opinion via Codex (auto, optional)
- `.claude/skills/pr/`: create PRs, archive plans, hand off (auto)
- `.claude/skills/sync-docs/`: documentation sync (auto)
- `.claude/skills/audit-harness/`: harness consistency audit (auto)
- `.claude/skills/anti-bottleneck/`: reduce unnecessary human interruptions (internal)
- `.claude/skills/release/`: cut a Homebrew release tag (manual trigger; repo-only, not distributed via template)

## Extensions

- `packs/languages/`: stack-specific rules and verification
- `scripts/`: init (`init-project.sh`), bootstrap (`bootstrap.sh`), install one-liner (`install.sh`), plan creation (`new-feature-plan.sh`, `new-ralph-plan.sh`), plan archival (`archive-plan.sh`), verification (`run-verify.sh`, `run-static-verify.sh`, `run-test.sh`), CI checks (`check-coverage.sh`, `check-pipeline-sync.sh`, `check-sync.sh`, `check-template.sh`), commit safety (`commit-msg-guard.sh`), language detection (`detect-languages.sh`), language pack creation (`new-language-pack.sh`), Ralph Loop orchestration (`ralph-loop.sh`, `ralph-loop-init.sh`, `ralph-status-helpers.sh`), pipeline orchestration (`ralph-pipeline.sh`, `ralph-orchestrator.sh`, `ralph-config.sh`, `ralph` CLI), TUI build (`build-tui.sh`), Codex availability check (`codex-check.sh`)
- `.github/workflows/`: CI checks (verify.yml, check-template.yml) and release automation (release.yml for goreleaser)

## Runtime state

- `.harness/state/`: transient markers and summaries
- `.harness/state/loop/`: Ralph Loop state (PROMPT.md, progress.log, iteration logs)
- `.harness/state/loop-archive/`: archived loop sessions
- `.harness/state/pipeline/`: pipeline mode state (checkpoint.json, phase logs, execution events, `.agent-signal` sidecar, `.pr-url` sidecar)
- `.harness/state/orchestrator/`: multi-worktree orchestrator state (slice status, PIDs)
- `.harness/logs/`: local logs
