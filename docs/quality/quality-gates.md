# Quality gates

## Inner loop: fast and local

Use these for rapid feedback:
- hook guardrails
- targeted linting
- targeted type checks
- targeted tests
- plan and report updates

## Outer loop: stricter and broader

Use these in CI or later-stage review:
- wider test suites
- integration and e2e checks (when implemented)
- architecture or structure checks
- secret scanning, dependency checks, and broader security scans (when implemented)
- deployment validation (when implemented)

## Suggested gate policy

### Must pass locally before "done"

- `./scripts/run-verify.sh` (all checks, backward-compatible)
- `./scripts/run-static-verify.sh` (static analysis only — wrapper for `HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh`, changed-language scope by default, used by `/verify`)
- `./scripts/run-test.sh` (tests only — wrapper for `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh`, changed-language scope by default, used by `/test`)
- project-specific local checks
- plan and docs sync if behavior changed

`run-static-verify.sh` and `run-test.sh` must stay non-overlapping:
- static mode runs format checks, linters, static analyzers, type checks,
  syntax checks, and drift checks; it must not run behavioral tests
- test mode runs behavioral unit, integration, and regression tests; it must not
  run format checks, linters, static analyzers, type checks, syntax-only gates,
  or drift checks
- `run-verify.sh` without `HARNESS_VERIFY_MODE` remains the backward-compatible
  aggregate and may run both static verification and tests
- `run-verify.sh` defaults to `RALPH_VERIFY_SCOPE=full`; the `/verify` and
  `/test` wrappers default to `RALPH_VERIFY_SCOPE=changed`. Changed scope runs
  only language packs affected by the current git diff while always running
  project-local gates. Shared or ambiguous changes fall back to full scope.
  The diff is taken from the merge-base of HEAD and `RALPH_VERIFY_BASE` when
  set, otherwise of HEAD and the default branch (origin's, else the current
  branch's tracked remote's, else local main/master), plus uncommitted and
  untracked files. With no remote default branch and no tracked remote (for
  example a repo with no remote), a HEAD on local main/master or detached at its
  tip is its own base, so only uncommitted and untracked files count and the
  result can be `no_changes`; set `RALPH_VERIFY_BASE` or
  `RALPH_VERIFY_SCOPE=full` to cover committed changes.

### Must pass in CI before merge

- `./scripts/secret-scan.sh --range <merge-base>..HEAD` — pull request secret leak scan (`.github/workflows/verify.yml`); the range scan pins the local git settings and attribute files that change which added lines `git log -p` prints (a local `diff.<driver>.binary` goes back to git's own content check, and the user-level and system-wide attributes files are dropped), so on git 2.41 or later a local run scans the same added lines as CI, apart from `.git/info/attributes`: git reads that file under every setting, and CI's clone has none. It exits 3 (not clean) when git cannot read the range. It reads HEAD's attributes, while CI checks out the PR merge commit; the local mirror below reads the merge's attributes when the base changed `.gitattributes`, and does not scan while `.git/info/attributes` holds a rule (details in the `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` headers and `docs/tech-debt/README.md`)
  - Local mirror of this same range scan: `./scripts/secret-scan-branch.sh` runs automatically from `./scripts/run-verify.sh` (static and all modes, default mode — an unscannable state is a one-line notice, not a failure) and again with `--strict` from `/pr` right before push. While `.git/info/attributes` holds a rule it does not scan ("cannot scan", so `--strict` exits 3); the range scan itself reads that file as git does, so the merge guard hook is not blocked by an unrelated rule there. When the base changed `.gitattributes` after the branch forked, it reads the attributes of `git merge-tree --write-tree <base> HEAD`, as CI's merge commit has them; a conflicting merge reads HEAD's, with a notice. `--strict` exits 3 when that merge cannot be computed the way CI's is (a local merge driver, a git older than 2.41, merge-tree failing), and merging or rebasing the base into the branch clears it
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (`.github/workflows/verify.yml`)
- `./scripts/check-template.sh` (`.github/workflows/check-template.yml`)
- `./scripts/check-sync.sh` — templates/root parity (`.github/workflows/check-template.yml`)
- `./scripts/check-coverage.sh` — language-pack coverage gate (graceful skip if no packs) (`.github/workflows/verify.yml`)
- `./scripts/check-pipeline-sync.sh` — pipeline order consistency across 6 reference files (`.github/workflows/verify.yml`)
- `./scripts/check-skill-sync.sh` — `.claude/skills/` ↔ `.agents/skills/` mirror parity (`.github/workflows/verify.yml`)

### Not yet implemented in CI

The following are aspirational gates listed for future adoption:
- broader test coverage (unit, integration, e2e)
- dependency vulnerability scans and broader security scans beyond secret leak detection
- org or repo-specific policy checks

## Org runtime gates

Autonomous multi-seat execution (org runtime, `ralph org` verbs) enforces its
own gates deterministically, independent of any LLM judgment:

| Gate | Mechanism | On failure |
|------|-----------|------------|
| Envelope validation | model pool / role pool / `max_seats` / `max_orgs` / `max_total_seats` / `--reserve` path overlap with another running org, checked by `ralph org spawn` | Spawn rejected, recorded in manifest; a retry of a seat whose earlier spawn is unfinished (last state event `spawn_started` or `spawn_step`) that fails the pool, role, permission, or `--scope` check is refused with nothing recorded, so the pane on record stays for `ralph org stop` |
| Split plan approval | `ralph org start --plan` reads the split plan from `<state dir>/splits/<id>.md`, compares its body digest (the rule of `scripts/plan-visual.sh digest`) with the `- Approved:` line, runs the spawn checks above before any side effect, and reuses a feature worktree only when its `ralph-worktree.sh` record and checkout match | Start refused with the reason, no ledger record, worktree, or herdr call; a refusal after the worktree exists leaves it and the message says how to reuse or remove it |
| Watchdog pulse layer | `ralph org watch --all`, one process per ledger under `watch.lock`: every running org's stall / liveness / scope-change ALERT, and an ALERT the leader did not act on within `[org].deadman_minutes` (or one about the leader) recorded as an `ALERT` item of the inbox that reaches the human by the inbox's rules (`escalations.jsonl` directly when the inbox cannot take it), plus the inbox deadlines (`[org.director]` `ack_minutes` for an open item never notified, `resolve_minutes` for an item not resolved). With `[org.director].enabled` (default true) and such a watch accepting work, `ralph org escalate` only records an item (delivery `deadline`), so the ack deadline is what brings an item no director acked to the human; with no watch accepting work, or `enabled = false`, escalate sends it at once (`inbox_no_watch`, `inbox_director_disabled`) | ALERT to leader; unanswered alerts become inbox `ALERT` items for the director or the human; an item past a deadline is sent to the human (`escalations.jsonl`, the watch's stderr, desktop notification) and recorded as `notified` |
| Cross-org watch gate | `ralph org spawn`, both forms of `ralph org start`, `ralph org director start`, and a `TASK` sent with `ralph org send` (typed, or `--raw` with `TYPE: TASK` on its first line) need a watch on the ledger that accepts work: `ralph org watch --all` running, not stopping, not a `--once` run (judged from `watch.lock` and `watch-alive.json`); `start` and `director start` launch one when none runs | Refused with exit 1 and nothing recorded (no manifest event, no receipt, no herdr or agmsg call), `--dry-run` included; the error says how to start the watch, stop a stale one, or wait for one that is ending. `stop`, `disband`, `escalate`, `approve`, `gate`, `merge-order`, `regate`, `merge`, the `director` verbs other than `start`, and the inbox verbs work with no watch (`director attach` registers anyway and warns on stderr that escalate then sends items at once) |
| Feature plan approval | In an org `ralph org start --plan` started in approval mode `per-plan` (`[org.approval].mode`, the default, recorded in the org's feature binding at start), `ralph org spawn` of a seat whose role is `implementer`, and a `TASK` to such a seat (typed or `--raw`), need a plan a human approved with `ralph org approve`: a `plan_approved` record in the org's current life (after its last `disbanded`) for the path and digest of the TASK's one `PLAN: <path> sha256:<digest>` header (for a spawn, any approval of the org), the file's digest and whole-content hash unchanged since, and the plan's `- Status: Approved` line and an `- Approved:` line that is exactly `<YYYY-MM-DD> sha256:<digest>` with that digest. In every org `start --plan` started, any seat but the leader is also refused while the org's leader seat is not running | A `TASK` is refused with exit 1 and nothing recorded, `--dry-run` included; a spawn is refused with a `rejected` event and a receipt, as the envelope checks are. The error says what the check found, then `refused by the approval check` and the one next step for that reason (wait for the human's answer, or escalate a BLOCKED when an approving `DECISION` came without the record; write and commit the plan's two header lines; undo the change or ask for a new approval; for the `TASK`'s `PLAN:` header, put one right after `TASK_ID:` when there is none, two, or a malformed one, or change its digest to the current one when it names an old or mistyped digest of a plan whose current version is its latest approval and passes the check, never to an earlier approved version; escalate a BLOCKED for anything else), and tells the leader never to run `ralph org approve` itself and, for a `TASK`, never to send it with `--raw`. approve's check of the caller's herdr pane (`HERDR_PANE_ID`) is a tripwire, not a boundary; the record keeps who ran it, for audit. approve only records: it messages no leader and resolves no inbox item, and prints the `ralph org send` `DECISION` and then the `ralph org inbox resolve` to run once that send exited 0, so an item whose `DECISION` was never sent stays under the inbox deadlines; a second approve of an approved item is refused and prints both again |
| Quality pipeline gate | impl exit checks → reviewer (re-runs `run-static-verify.sh` / `run-test.sh` first, then reviews the diff) → leader arbitration | Gate fail: the reviewer returns BLOCKED (`GATE: fail`) without reviewing and the leader routes it back to impl; a gate that cannot run (`GATE: unrunnable`) goes to the leader, not impl. This row is driven by the role prompt templates, not enforced mechanically (see docs/tech-debt). For a feature org (`ralph org start --plan`), the final gate is recorded mechanically: `ralph org gate` runs `run-static-verify.sh` and `run-test.sh` on the rebased head and records `gate_passed` or `gate_failed` with the head and base SHAs, and it refuses before running, recording nothing, when the org is not a running feature org with a complete binding, the directory is outside the feature worktree or off its branch (a detached HEAD and a rebase in progress included), the tree is dirty, no default branch is found, the head does not contain the default branch's tip, or `scripts/run-static-verify.sh` or `scripts/run-test.sh` is missing or not executable. Running it before the PR is still the leader template's instruction. Once the org is disbanded, `ralph org regate --branch <branch>` takes its place after another PR is merged: it merges origin's default branch into the branch (a merge commit, no rebase, no force push), runs the same two scripts, records the same events with `via=regate`, and pushes a pass |

See `.claude/rules/ralph/agent-messaging.md` for the org runtime protocol and
`.harness/state/org/manifest.jsonl` for the append-only audit trail.

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
