# Repo map

## Core files

- `AGENTS.md`: vendor-neutral map; user-owned skeleton + a managed block whose content is sourced from `.ralph/core/AGENTS.core.md`
- `CLAUDE.md`: minimal seed (imports `AGENTS.md`); ralph's always-on guidance auto-loads as a project rule from `.claude/rules/ralph/ralph-workflow.md`
- `README.md`: human entry point

## Claude control plane

- `.claude/rules/ralph/`: shipped ralph guidance, path-scoped or topic-scoped (read by both agents); language pack rules also render here as `<lang>.md`
- `.claude/skills/`: Claude-side skill bodies (mirrored to `.agents/skills/`)
- `.claude/agents/`: Claude Code subagent definitions
- `.claude/hooks/`: hook implementations plus `<event>.d/` dispatch entries; `.claude/settings.json` points every event at the single `ralph-dispatch.sh <event>` entry point, which fans out in order through `.claude/hooks/<event>.d/` (core) → `.ralph/local/hooks/<event>.d/` (downstream local, committed) → `.claude/hooks/local/<event>.d/` (downstream local, gitignored)
- `.claude/settings.json`: hook and permission configuration

## Overlay layout (`.ralph/`)

- `.ralph/core/`: generation sources `ralph init` consumes (e.g. `AGENTS.core.md`, the managed-block content for `AGENTS.md`)
- `.ralph/local/`: downstream extension points (`hooks/<event>.d/`, `verify.d/`, `test.d/`) that the hooks dispatcher and `scripts/run-verify.sh` execute after core processing (`hooks/<event>.d/` wiring is Claude Code today; Codex's `.codex/config.toml` still calls hook scripts directly — Phase 3 tech debt)

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
