# Repo map

## Core files

- `AGENTS.md`: vendor-neutral map
- `CLAUDE.md`: Claude-specific always-on additions
- `README.md`: human entry point

## Claude control plane

- `.claude/rules/`: path-scoped or topic-scoped guidance (read by both agents)
- `.claude/skills/`: Claude-side skill bodies (mirrored to `.agents/skills/`)
- `.claude/agents/`: Claude Code subagent definitions
- `.claude/hooks/`: deterministic hook scripts
- `.claude/settings.json`: hook and permission configuration

## Codex control plane

- `.agents/skills/`: Codex-side skill bodies (kept in lock-step with `.claude/skills/` via `scripts/check-skill-sync.sh`)
- `.codex/`: project-level Codex config for the meta-repo itself (`config.toml`, `agents/`, `hooks/`, `AGENTS.override.md`, `README.md`); `agents/` contains Codex custom agent definitions; same shape as `templates/base/.codex/` so ralph dogfoods the parity it ships
- `templates/base/.codex/`: `ralph init` source for the same surface; root `.codex/` and template `.codex/` are kept byte-identical, validated by `scripts/check-sync.sh` (no KNOWN_DIFFS today)

## Process artifacts

- `docs/specs/`: spec files produced by `/spec`
- `docs/plans/active/`: in-flight plans
- `docs/plans/archive/`: completed plans
- `docs/reports/`: self-review, verify, test, sync-docs, cross-review triage, and walkthrough reports
- `docs/quality/`: definition of done and gates
- `docs/tech-debt/`: known debt and follow-ups
- `docs/evidence/`: what counts as evidence
- `docs/recipes/`: hands-on recipes (Codex setup, agent teams, language packs, worktrees)
- `docs/roadmap/`: maturity-model and future direction documents
- `docs/research/`: approach comparisons and investigation notes
- `docs/references/`: source notes and external reference links

## Skills

- `.claude/skills/spec/`: refine vague ideas into detailed specifications (auto-invoked when a request is too vague for /plan)
- `.claude/skills/plan/`: create plans (auto)
- `.claude/skills/work/`: create branch and execute plans interactively (auto)
- `.claude/skills/org/`: autonomous multi-seat execution via `ralph org` verbs (auto)
- `.claude/skills/self-review/`: self-review diff quality (auto)
- `.claude/skills/verify/`: spec compliance and static analysis (auto)
- `.claude/skills/test/`: behavioral tests (auto)
- `.claude/skills/cross-review/`: cross-model second opinion via the other agent (Claude → Codex; Codex → Claude) (auto, optional)
- `.claude/skills/pr/`: create PRs, archive plans, hand off (auto)
- `.claude/skills/sync-docs/`: documentation sync (auto)
- `.claude/skills/audit-harness/`: harness consistency audit (auto)
- `.claude/skills/anti-bottleneck/`: reduce unnecessary human interruptions (internal)
- `.claude/skills/release/`: cut a Homebrew release tag (manual trigger; repo-only, not distributed via template)

## Extensions

- `packs/languages/`: stack-specific rules and verification
- `scripts/`: init/bootstrap/install (`init-project.sh`, `bootstrap.sh`, `install.sh`), plan creation and archival (`new-feature-plan.sh`, `archive-plan.sh`), branch/worktree/PR guards (`branch-name.sh`, `ralph-worktree.sh`, `ensure-pr-ready.sh`, `ensure-pr-title-prefix.sh`), verification (`run-verify.sh`, `run-static-verify.sh`, `run-test.sh`, `verify.local.sh`), CI and drift checks (`check-coverage.sh`, `check-pipeline-sync.sh`, `check-skill-sync.sh`, `check-sync.sh`, `check-template.sh`), secret and commit safety (`secret-scan.sh`, `pre-commit-secret-guard.sh`, `commit-msg-guard.sh`, `prepare-commit-msg-secret-guard.sh`, `pre-merge-commit-secret-guard.sh`), language detection (`detect-languages.sh`, `detect-changed-languages.sh`), language pack creation (`new-language-pack.sh`), standard-flow shared config and cross-review helpers (`ralph-config.sh`, `xreview-helpers.sh`, `ralph-common.sh`), skills-mirror generation (`sync-skills.sh`), artifact retention (`gc-artifacts.sh`), insight events (`insights-append.sh`), Codex availability check (`codex-check.sh`)
- `.github/workflows/`: CI checks (verify.yml, check-template.yml) and release automation (release.yml for goreleaser)

## Tests

- `tests/`: shell test suites (`tests/test-*.sh`) and fixtures (`tests/fixtures/`, incl. `cli-stubs/`) covering scripts, hooks, and pipeline behavior

## Runtime state

- `.harness/state/`: transient markers and summaries
- `.harness/state/standard-pipeline/`: standard-flow post-implementation pipeline cycle-cap state (`active-plan.json`, `cycle-count.json`)
- `.harness/state/org/`: org runtime manifest, saga records, receipts, watchdog state
- `.harness/logs/`: local logs
