# AGENTS.md

This repository hosts `ralph`, a CLI for harness engineering. Run `ralph init` to scaffold a new project from this source.

Treat this file as a **map**:
- short
- stable
- cross-vendor
- easy to verify against the repo

<!-- BEGIN RALPH MANAGED (ralph:agents-md) -->
## Mission

Build coding-agent workflows that are:
- reliable
- inspectable
- evidence-backed
- easy to extend
- cheap by default, richer only when needed

## Primary loop

This project ships two independent execution surfaces:

- **development harness** — the interactive standard flow below, used for all spec/plan/implement/review changes.
- **org runtime** — autonomous multi-seat execution (`ralph org spawn/send/wait/...`) for tasks that need a coordinating `leader` plus role seats running outside a single interactive session.

The development harness:

1. Spec (auto, optional — refines vague ideas into detailed specifications via decision-tree questioning, codebase exploration, web research, and user clarification)
2. Plan (auto — ensures a clean-base task worktree, creates plan, shows a visual review page and asks for approval: Approve / Needs changes) [+ other-model plan advisory when `[standard].plan_advisory` is on and the other CLI is available; a skip is recorded]
3. Implement (auto — resumes task worktree, checks the plan's approval digest, interactive implementation)
4. Self-review (auto — via `reviewer` subagent, or pipeline-internal)
5. Verify (auto — via `verifier` subagent, or pipeline-internal)
6. Test (auto — via `tester` subagent, or pipeline-internal)
7. Sync-docs (auto — via `doc-maintainer` subagent, or pipeline-internal)
8. Cross-review (auto — cross-model second opinion via the other agent: Claude → Codex; Codex → Claude; calls it only when `[standard].cross_review` is on and that CLI is available, otherwise records the skip)
9. PR (auto — includes hand-off)
10. CI verify + human merge

All repo writes in spec/plan/implement flows must happen inside a task worktree
created from a clean default branch.

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Verification & test contracts

Key rule: never say "done" without saying what was verified and what remains unverified. Tests must pass before PR creation.

## Hard rules

- Keep this file short
- Keep `CLAUDE.md` short
- Move detailed topic guidance into `.claude/rules/ralph/`
- Move step-by-step workflows into `.claude/skills/`
- Promote repeated mistakes into hooks, tests, CI, or scripts
- Do not expand plans into brittle low-level instructions unless the task truly needs it
- Keep names grep-able and boundaries explicit
- Update docs when behavior, contracts, or workflows change

Detailed topic guidance lives in `.claude/rules/ralph/`; step-by-step
workflows live in `.claude/skills/`.
<!-- END RALPH MANAGED -->

## Org runtime pointers (meta-repo specific)

The self-review, verify, test, and sync-docs steps run through
phase-specific subagents for both Claude Code and Codex; `/cross-review`
and `/pr` remain inline. Local task state for spec/plan/implement lives under
`$(git rev-parse --git-common-dir)/ralph/worktrees/`; PR success cleans up
the task worktree and local branch while leaving the remote PR branch
intact.

See `internal/org/`, `docs/specs/2026-08-01-org-runtime.md`, and
`.claude/rules/ralph/agent-messaging.md` for the org runtime's implementation
and protocol contract.

## Repo map

- `cmd/ralph/` — Go entrypoint for the ralph CLI (cobra root, ldflags injection, go:embed wiring)
- `internal/cli/` — cobra subcommands (init, upgrade, eject, adopt, status, doctor, pack, insights, org, config, version)
- `internal/scaffold/` — go:embed template system, manifest TOML (v3 ownership fields: `layout`/`owner` core|fork|seed|block/`forked_from_version`, opt-in write API, legacy read compat), file render with SHA256 hashes
- `internal/upgrade/` — overlay engine wired into `ralph upgrade` for v2 layouts: core replace planner with ordered ops + commit barrier, managed block engine, `.claude/settings.json` 3-way merge against the `.ralph/core/settings.ralph.json` snapshot, advisory diff, upgrade report writer (`docs/reports/upgrade-*.md`); legacy layouts migrate via a confirmed one-time flow (internal/cli/migrate.go) before chaining into this engine (spec: docs/specs/2026-08-17-overlay-scaffold-v2.md)
- `internal/config/` — ralph.toml parser with defaults; `role_models.go` holds `[org.role_models]` (each org role's driver and model) and its two built-in default sets; `local.go` applies the git-ignored personal `ralph.local.toml` from the main worktree's root key by key, and holds `LocalOnlyKeys` (keys only that file may set: `[org].state_dir` and five script keys in `[plan_visual]`, `[standard]`, and `[verify]`) and `LocalOrgStateDirAt` (the org ledger's place for `ResolveOrgStateDir`)
- `internal/gitroot/` — git helpers for the main worktree's root and a working tree's toplevel, shared by config (`ralph.local.toml`) and org (the ledger)
- `internal/org/` — org runtime mechanism layer: envelope validation (per-org `max_seats`; org-wide `max_orgs` / `max_total_seats` in `envelope.go`, their counts and scope reservations in `reserve.go`), the driver and model of a role from `[org.role_models]` (`role_models.go`: `ResolveRoleModel`, and the refusal of a config that still sets `[org.roles]`), split plans and their approval digest (`split.go`), the per-feature `ralph org start --plan` (worktree + bound leader with the recorded `[org.approval].mode`, `feature.go`), feature plan approval (`approve.go`: `ralph org approve` records a human's approval of a feature org's plan from its inbox QUESTION; `approval.go`: in a per-plan org Send and Spawn refuse an implementer TASK or spawn with no approved plan), the feature PR gate (`gate.go` with the script running in `gate_run.go` and the gate record that gate and regate write and merge-order reads in `gate_record.go`: `ralph org gate` runs `run-static-verify.sh` and `run-test.sh` on a feature org's rebased head and records `gate_passed`/`gate_failed` with the head and base SHAs; the gates of a ledger run their scripts one at a time under `gate.lock` (`gate_lock.go`: a wait with no time limit whose stderr lines name the holder, Gate's re-check of the org and worktree after the wait, and the lock's file passed to the scripts so the lock outlives a killed ralph); a gate records the load average around the scripts and waits for `[org.gate].max_load` before it starts (`gate_load.go`); `ralph org gate accept` (`gate_accept.go`) records a human's acceptance of a branch's latest gate failure as `gate_accepted`, bound to that failure's log, which merge-order lists as `accepted` and merge merges as it merges `ready`), the merge order (`mergeorder.go`: `ralph org merge-order` reads the gate records and origin's branches with one `git ls-remote` and lists which feature PRs may be merged now), the re-gate after a merge (`regate.go` and `regate_merge.go`, with the fetches, the check of HEAD against origin, and the bounded git runner in `regate_git.go` and, before the merge, the check that nothing ignored is on disk where it writes in `regate_inway.go`: `ralph org regate --branch` merges origin's default branch into a disbanded feature org's branch with a merge commit, runs the gate scripts, records `via=regate`, and pushes without force; under `[org.gate].regate_checks = "ci"`, `regate_ci.go` pushes first and records `gate_ci_passed`/`gate_ci_failed` with `via=regate_ci` from the PR's checks, read with `prchecks.go`, which merge shares; `regate.lock` keeps a spawn and a start of that org out while it runs, and a regate that runs the gate scripts takes `gate.lock` before it), the merge (`merge.go` with the checks, `merge_run.go` with the merge, the read-back, and `pr_merged`, and `merge_wait.go` with the wait of `--wait-checks` for checks still pending and for the branch's org to be disbanded, with the lock released: `ralph org merge --branch` merges with gh the PR of a branch merge-order lists as ready or accepted once it has checked the PR, its checks, and the default branch's protection live (`CheckMergeEnforcement` in `mergeenforce.go`, which shares the readers of `mergeguarantee.go`), under `regate.lock`), the cleanup of a merged feature org's worktree (`cleanup.go`, with its refusal text and the commands it names in `cleanup_text.go`: `ralph org merge` runs it once it records `pr_merged`, and `ralph org cleanup` runs it for a PR merged on GitHub, reading the PRs in origin's repository only, as merge does, with `ReadMergeOrigin` in `mergeorigin.go`; once its checks pass it removes the feature worktree, its local branch, and its `ralph-worktree.sh` record with `--force-branch`, under `regate.lock`, and records `worktree_cleaned`), the words of the commands that merge and cleanup print (`shellword.go`: `ShellWord`, the one function that quotes a word only when needed, and `VerbCommand`, which builds a `ralph org` command with `--state-dir` only when it was passed), the merge guarantee (`mergeguarantee.go`: `ralph doctor` reads the protection of origin's default branch with gh and caches the kind in `merge-guarantee.json`, which `ralph status` and `ralph org merge-order` read without gh), the org inbox shared by every org (`inbox.go`: append-only `inbox.jsonl`, open → acked → resolved) and `ralph org escalate` (`escalate.go`: with `[org.director].enabled` and a cross-org watch accepting work it only records the item, for the director and the ack deadline; otherwise it also sends it to the human at once), the follow-up record (`followup.go`: `ralph org followup add`/`list`/`done` keep the issues the director asks the human about once another branch's PR is merged, in append-only `followups.jsonl` under `followups.lock`, open → done, and `FollowupRecordedMerge` tells `ralph status` which are ready from the ledger's `pr_merged` and `worktree_cleaned`; `mergeorder_followup.go`: `ralph org merge-order` tells it from the after-branch's PRs with gh; both count only a merge after the follow-up was recorded, except that status counts the cleanup of a PR merged on GitHub before it, so merge-order's answer is the one to use when one says ready and the other waiting, and status's, with merge-order's reason, when merge-order says unknown), seat saga manifest (flock-serialized), receipts, permission-mode envelopes, herdr/agmsg driver adapters, role prompt templates (go:embed), typed message protocol, org report generation, the cross-org watch `ralph org watch --all` (`RunWatchAll` in `watch.go`: every running org's pulse, under the single-instance `watch.lock` and the liveness mark `watch-alive.json` of `watchalive.go`; the pulse's own escalations (`escalateAlert`: a deadman timeout, or an ALERT about the leader) are `ALERT` items of the same inbox, delivered by `escalate`'s rule, `inboxDelivery` in `escalate.go`; an alert that fires again while its item is not resolved is a `recurred` event on that item, in `watch_recurrence.go`, and `watch_scope.go` raises scope_change in an org with a reservation only for paths outside it) with the inbox ack/resolve deadlines (`deadline.go`) and the on-demand watcher, the watch gate that refuses spawn, start (and `director start`), and TASK sends while no long-running watch accepts work (`watchgate.go`), the herdr server gate that `Spawn` (not a dry run), `StartFeature`, and `StartDirector` (in herdr) call before they make anything in herdr and that the CLI sets to start a stopped `herdr server` (`herdrgate.go`), and the director registration (`director.go`: `ralph org director attach`/`detach`/`status` keep one registration per ledger in `director.jsonl` under `director.lock`; `director_start.go` and `director_stop.go`: `ralph org director start`/`stop` start and stop a headless director in a herdr workspace of its own, or with `[org.director].launch_cmd` in a terminal that command opens (`director_cmd.go`: the launch script checks in with the hidden `ralph org director launched`), recorded in `director.jsonl`), the director check (`director_check.go`: `ralph org watch --all` checks the current headless director each cycle before the wake and writes `director-alive.json`; `ralph org director start` cleans up an ended one first, in `director_start_cleanup.go`), the director's wait (`director_wait.go`: `ralph org wait --inbox --director <id>` keeps `director-wait.json` under `director-wait.lock` (when it started, last wrote, the watch it saw, how it ended), which `ralph status` and `ralph org director status` show, and ends with exit 4 when a watch that accepted work no longer does and 5 when its registration is not current), the push wake (`director_wake.go`: `ralph org watch --all` wakes a director registered with push through `[org.director].wake_cmd` or input to its herdr pane, and records `woken` in the inbox), the resume of seats stopped on an API error (`resume.go`: `ralph org resume` reads the herdr state and the end of the pane of each running seat and of each headless director in herdr, types `ResumeText` only into a pane stopped on a known API error and presses Enter alone where the CLI asks for it after a usage limit reset, in the order other seats, leaders, directors, and records `resumed` for a seat it typed into; the screen rules and the sentence list of each driver are in `resume_screens.go`), the screen fallback of send (`send_fallback.go`, with the screen rules in `send_screen.go` on the screen model of `resume_screens.go`: `ClassifySendScreen` for the wait and `sendInputHolds` for the check before Enter; when herdr's idle/done wait for a claude or codex seat runs out of all but a reserve of `--timeout-ms`, `ralph org send` reads the pane twice and types only into a screen waiting for input, checks the submit on the screen, and records `via=screen` on `sent`; `--no-screen-fallback` turns it off), `ralph org status --live` (`status_live.go`: herdr's state and the screen of each active seat, with a `mismatch:` line for a seat herdr reports working while its screen waits for input; it starts no herdr server), and the pane lock (`pane_lock.go`: `ralph org send`, resume, and the push wake (`director_wake.go`: the line it types into a herdr director's pane, and a configured `wake_cmd` when that pane is recorded) take one lock per herdr pane while they type, so two of them never type into a pane at once; `send` waits for the seat to be idle or done once more after it takes the lock (on the screen path it reads the screen again instead) and, for a TASK to an implementer in a per-plan org, checks the plan's approval a last time under it) (spec: docs/specs/2026-08-01-org-runtime.md; protocol rule: .claude/rules/ralph/agent-messaging.md)
- `internal/insights/` — insight event/receipt readers, aggregation, and report backfill for `ralph insights`
- `templates/` — go:embed source: base scaffold, language packs
- `docs/specs/` — spec files produced by `/spec` (`<date>-<slug>.md`)
- `docs/plans/active/` — current plans (`<date>-<slug>.md` single files)
- `docs/plans/archive/` — completed plans
- `docs/plans/templates/` — plan templates (`feature-plan.md`)
- `docs/reports/` — self-review, verify, test, sync-docs, cross-review triage, walkthrough artifacts
- `docs/insights/` — committed insight events (`events/<date>-<slug>.jsonl`); schema in `docs/insights/README.md`; consumed by `ralph insights`
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/ralph/` — path-scoped ralph guidance (read by both agents); language pack rules also render here as `<lang>.md`
- `.claude/skills/` — Claude-side on-demand workflows
- `.claude/agents/` — Claude Code subagent definitions
- `.claude/hooks/` — deterministic runtime controls; `settings.json` points each event at `ralph-dispatch.sh <event>`, which fans out to `.claude/hooks/<event>.d/` (core), `.ralph/local/hooks/<event>.d/` (downstream local, committed), then `.claude/hooks/local/<event>.d/` (downstream local, gitignored)
  - `check_mojibake.sh` + `mojibake-allowlist` — temporary U+FFFD detection guard for Claude Code SSE mojibake (remove once upstream Issue #43746 ships)
- `.ralph/core/` — generation sources consumed by `ralph init` (e.g. `AGENTS.core.md`, the managed-block content for `AGENTS.md`; `settings.ralph.json`, the settings 3-way-merge snapshot consumed by `ralph upgrade`)
- `.ralph/local/` — downstream extension points (`hooks/<event>.d/`, `verify.d/`, `test.d/`) that `scripts/run-verify.sh` and the hooks dispatcher execute after core processing (`hooks/<event>.d/` wiring runs under both Claude Code and Codex — both route through `ralph-dispatch.sh`)
- `.agents/skills/` — Codex-side skill bodies (mirrors `.claude/skills/`; regenerated by `scripts/sync-skills.sh`, drift-checked by `scripts/check-skill-sync.sh`)
- `.codex/` — Codex project config for this meta-repo (`config.toml`, `hooks.json`, `agents/`, `hooks/`, `AGENTS.override.md`, `README.md`); `hooks.json` is the hook wiring source of truth (routes `PostToolUse`, `PreToolUse`, `SessionStart`, and `UserPromptSubmit` through `ralph-dispatch.sh`; `config.toml` keeps only a reference comment); `agents/` contains Codex custom agent definitions; same shape as `templates/base/.codex/` so ralph dogfoods the parity it ships
- `templates/base/.codex/` — `ralph init` source for the same surface; root `.codex/` and template `.codex/` are kept identical via `scripts/check-sync.sh` (no KNOWN_DIFFS today)
- `packs/languages/` — language-specific depth (also copied to `templates/packs/` for embedding)
- `scripts/` — reusable verification and bootstrap scripts (includes `ralph-common.sh` (shared shell helpers: ts/log/default_branch/detect_active_plan_dir); `ralph-config.sh`, `ralph-worktree.sh`, `xreview-helpers.sh` (cross-review driver helpers: `detect_base_branch` / `pick_reviewer` / `count_triage_findings`), `install.sh`, skills-mirror generator `sync-skills.sh`, drift gate `check-skill-sync.sh`, artifact retention GC `gc-artifacts.sh`, insight-event appender `insights-append.sh`, Codex availability probe `codex-check.sh`, branch-history secret scan `secret-scan-branch.sh`, `/plan` visual review helper `plan-visual.sh` (`open` / `shot` / `lint` / `digest`))
- `docs/recipes/` — hands-on recipes (Codex setup, language packs, worktrees)
- `.harness/state/` — runtime state, not canonical truth

## Planning contract

Every non-trivial task should have:
- objective
- scope and non-goals
- affected files or modules
- acceptance criteria
- verify plan (static analysis, spec compliance, doc drift)
- test plan (unit, integration, regression, edge cases)
- risk register
- rollout or rollback note
- evidence targets

## Review contract

Reviews should produce artifacts, not only chat output:
- findings with severity (diff quality only)
- evidence
- merge or no-merge recommendation
- follow-ups
- known gaps

## Verification & test contracts (meta-repo addition)

See `docs/quality/definition-of-done.md` for full checklists.

## Human escalation boundaries

Escalate to a human only for:
- irreversible destructive actions
- secrets or credentials you cannot access
- product or design judgment that cannot be verified from repo context
- external approvals that are genuinely required

Everything else should first attempt self-verification.
