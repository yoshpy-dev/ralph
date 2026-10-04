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

- `./scripts/secret-scan.sh --range <merge-base>..HEAD` — pull request secret leak scan (`.github/workflows/verify.yml`); the range scan pins the local git settings and attribute files that change which added lines `git log -p` prints (a local `diff.<driver>.binary` goes back to git's own content check, and the user-level and system-wide attributes files are dropped), so on git 2.41 or later a local run scans the same added lines as CI, apart from `.git/info/attributes`: git reads that file under every setting, and CI's clone has none. It exits 3 (not clean) when git cannot read the range. It reads HEAD's attributes, while CI checks out the PR merge commit; the local mirror below reads the merge's attributes when the base changed `.gitattributes`, and does not scan while `.git/info/attributes` holds a rule (details in the `scripts/secret-scan.sh` and `scripts/secret-scan-branch.sh` headers)
  - Local mirror of this same range scan: `./scripts/secret-scan-branch.sh` runs automatically from `./scripts/run-verify.sh` (static and all modes, default mode — an unscannable state is a one-line notice, not a failure) and again with `--strict` from `/pr` right before push. While `.git/info/attributes` holds a rule it does not scan ("cannot scan", so `--strict` exits 3); the range scan itself reads that file as git does, so the merge guard hook is not blocked by an unrelated rule there. When the base changed `.gitattributes` after the branch forked, it reads the attributes of `git merge-tree --write-tree <base> HEAD`, as CI's merge commit has them; a conflicting merge reads HEAD's, with a notice. `--strict` exits 3 when that merge cannot be computed the way CI's is (a local merge driver, a git older than 2.41, merge-tree failing), and merging or rebasing the base into the branch clears it
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` (`.github/workflows/verify.yml`)
- `./scripts/check-template.sh` — required-file / structure check (`.github/workflows/verify.yml`)
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
| Envelope validation | model pool / role pool / `max_seats` checked by `ralph org spawn` | Spawn rejected, recorded in manifest |
| Watchdog pulse layer | stall / liveness / scope-change ALERT and deadman human escalation (`ralph org watch`) | ALERT to leader; unanswered alerts escalate to a human |
| Quality pipeline gate | impl exit checks → reviewer (re-runs `run-static-verify.sh` / `run-test.sh` first, then reviews the diff) → leader arbitration | Gate fail: the reviewer returns BLOCKED (`GATE: fail`) without reviewing and the leader routes it back to impl; a gate that cannot run (`GATE: unrunnable`) goes to the leader, not impl. This row is driven by the role prompt templates, not enforced mechanically |

See `.claude/rules/ralph/agent-messaging.md` for the org runtime protocol and
`.harness/state/org/manifest.jsonl` for the append-only audit trail.

## Important

If a rule truly matters, it should eventually live in a deterministic gate.
