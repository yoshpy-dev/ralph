# Verify evidence: org-director-wake

- Date: 2026-10-18 (UTC timestamps in the output read 2026-10-18T09:08; JST is UTC+9)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-wake, HEAD 111b65a6 (base 9e92d928)
- Redaction: the session scratch directory is `$SCRATCH`, and a probe's working directory under it is `$W`. The home directory would be `~`; none appears. The two `jq -e .` steps of the wrapper print the whole settings.json; those dumps are elided here. The full wrapper log is `docs/evidence/verify-2026-10-18-085853.log`, and the raw outputs of this file (with the probe scripts) are in `docs/evidence/verify-2026-10-18-org-director-wake.log` (both gitignored, not committed).
- Lines in `[brackets]` starting with `note:` were added when this file was written; everything else is command output. The `detach it with:` and `warning: director attach: ... no ralph org watch --all accepts work` lines of each successful attach are left out of section 3.

## 1. Static analysis

`./scripts/run-static-verify.sh` (default changed scope), rc 0. Summary lines:

```text
==> shellcheck hook + verify scripts
    OK
==> scripts/check-sync.sh
  IDENTICAL:      171
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5
PASS: all files in sync.
==> scripts/check-pipeline-sync.sh
    OK
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
==> scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
==> tech-debt README plan references
    OK
==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)
==> Language packs selected: golang
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned 9e92d928..111b65a6 against origin/main: clean
==> All verifiers passed.
```

The Go toolchains and the two sync scripts, run separately:

```text
== GOTOOLCHAIN=go1.25.8 go vet ./... ==
go version go1.25.8 darwin/arm64
rc=0
== GOTOOLCHAIN=go1.25.8 go build ./... ==
rc=0
== go vet ./... (local) ==
go version go1.26.0 darwin/arm64
rc=0
== check-skill-sync.sh ==
[ok] check-skill-sync: 14 skill(s) in lock-step
rc=0
== check-sync.sh ==
PASS: all files in sync.
rc=0
```

## 2. Mirrored files

```text
identical .claude/skills/director/SKILL.md .agents/skills/director/SKILL.md
identical .claude/skills/director/SKILL.md templates/base/.claude/skills/director/SKILL.md
identical .claude/skills/director/SKILL.md templates/base/.agents/skills/director/SKILL.md
identical .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md
identical .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md
identical .claude/skills/org/SKILL.md templates/base/.agents/skills/org/SKILL.md
identical docs/recipes/director-wake.md (2 faces)
identical docs/recipes/director-launch-cmd.md (2 faces)
identical scripts/ralph-config.sh (2 faces)
```

## 3. Probe: attach, the wake by wake_cmd, and inbox show (HEAD binary)

The binary was built with `GOTOOLCHAIN=go1.25.8 go build -o $SCRATCH/bin/ralph ./cmd/ralph` at HEAD 111b65a6. Every command ran under `env -i HOME=$W/home PATH=/usr/bin:/bin`, in `$W/other` (not a git repository), with `--state-dir $W/ledger` and `--config $W/cfg/ralph.toml`:

```toml
[org.watchdog]
interval_seconds = 5
[org.director]
wake = ["pull", "push"]
wake_cmd = ["./wake.sh", "{target}", "{text}"]
```

`$W/cfg/wake.sh` appends its working directory (`pwd -P`) and each argument in brackets to a mark file. No org runs, so the pulse calls nothing, and every inbox item was escalated in the same second, so no deadline passed and nothing reached osascript. herdr is not on that PATH.

```text
## attach refusals (config has wake_cmd with {target})
$ ralph org director attach --driver claude --state-dir $W/ledger --config $W/cfg/ralph.toml
org: director attach: [org.director].wake_cmd has {target}, but no --wake-target was given (wake pull,push): pass --wake-target with where [org.director].wake_cmd sends the wake line; [org.director].wake and wake_cmd are read from the --config file $W/cfg/ralph.toml
rc=1
$ ralph org director attach --driver claude --wake pull --wake-target t1 --state-dir $W/ledger --config $W/cfg/ralph.toml
org: director attach: --wake-target is given, but the wake has no push (wake pull): the watch sends to the wake target only to push; drop --wake-target, or attach with --wake push or --wake pull,push; [org.director].wake_cmd is read from the --config file $W/cfg/ralph.toml
rc=1
$ ralph org director attach --driver claude --wake push --wake-target -x --state-dir $W/ledger --config $W/cfg/ralph.toml
org: director attach: --wake-target: the wake target is not usable: "-x" starts with "-", which the command [org.director].wake_cmd runs could read as an option
rc=1
$ ralph org director attach --driver claude --wake push,push --state-dir $W/ledger --config $W/cfg/ralph.toml
org: director attach: --wake: element 1 "push" repeats element 0; list each value once
rc=1
ledger after refusals:
[note: nothing listed: the ledger directory is still empty]
## attach that records push
$ ralph org director attach --driver claude --wake-target term;1 --state-dir $W/ledger --config $W/cfg/ralph.toml
attached d1 (driver=claude, wake=pull,push, wake_target=term;1)
rc=0
{"ts":"2026-10-18T09:08:42Z","id":"d1","event":"attached","driver":"claude","wake":["pull","push"],"wake_target":"term;1","user":"","cwd":"$W/other","herdr_pane":false}
$ ralph org director status --state-dir $W/ledger --config $W/cfg/ralph.toml
director: attached d1 (driver=claude, attached at 2026-10-18T09:08:42Z, wake=pull,push to term;1) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
enabled: true
auto_merge: false
id: d1
driver: claude
wake: pull,push
wake_target: term;1
attached_at: 2026-10-18T09:08:42Z
user: -
cwd: $W/other
herdr_pane: no
rc=0
## two open items, one acked item
[note: inbox.jsonl was written by the probe: e1 (delivery deadline, body "MARKER-body; rm -rf x"), e2 (delivery now), e3 escalated and acked]
## watch --all --once from another directory with --config
$ ralph org watch --all --once --state-dir $W/ledger --config $W/cfg/ralph.toml
watching every org (interval=5s once=true state-dir=$W/ledger (source: flag))
rc=0
wake_cmd runs:
$W/cfg|[term;1][New org inbox items for the director: e1 e2. Read them with ralph org inbox and handle each as the director skill says.]
inbox woken lines:
{"ts":"2026-10-18T09:08:42Z","id":"e1","event":"woken","director_id":"d1","via":"wake_cmd"}
{"ts":"2026-10-18T09:08:42Z","id":"e2","event":"woken","director_id":"d1","via":"wake_cmd"}
## second cycle: no new run
$ ralph org watch --all --once --state-dir $W/ledger --config $W/cfg/ralph.toml
watching every org (interval=5s once=true state-dir=$W/ledger (source: flag))
rc=0
wake_cmd run count:        1
$ ralph org inbox show e1 --state-dir $W/ledger --config $W/cfg/ralph.toml
id: e1
state: open
org: o1
type: QUESTION
escalated_at: 2026-10-18T09:08:42Z
notified: deferred (left to the director, or to the inbox deadlines of ralph org watch --all; to send it to the human now: ralph org inbox notify e1 --state-dir '$W/ledger')
message:
  TYPE: QUESTION

  MARKER-body; rm -rf x
events:
  2026-10-18T09:08:42Z escalated
  2026-10-18T09:08:42Z woken director_id=d1 via=wake_cmd
rc=0
## attach --replace --wake pull, then a new item e4
$ ralph org director attach --driver codex --replace --wake pull --state-dir $W/ledger --config $W/cfg/ralph.toml
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-18T09:08:42Z, wake=pull,push, wake_target=term;1)
rc=0
$ ralph org watch --all --once --state-dir $W/ledger --config $W/cfg/ralph.toml
watching every org (interval=5s once=true state-dir=$W/ledger (source: flag))
rc=0
wake_cmd run count:        1
```

`[note: what this shows: the four attach refusals exit 1 and write nothing; wake_cmd runs once, in the directory of the --config file (not the working directory $W/other), with the argv replaced and no shell (the target "term;1" arrives as one argument); the line names e1 and e2 (open, either delivery) and not e3 (acked), and holds nothing of the body; one woken event per item with director_id and via; no second run for woken items; inbox show lists the woken event; a pull-only registration is not woken for a new item.]`

## 4. Probe: a feature worktree does not choose wake_cmd (AC4)

A scratch repository `$W/main` (branch main, one empty commit) with a linked worktree `$W/feature` (branch feat). Both have a `wake.sh` that appends `pwd -P` and its arguments to its own mark file, and a `ralph.toml` with `wake = ["push"]` and `wake_cmd = ["./wake.sh", "<side>", "{text}"]` (`<side>` is `main` or `feature`). Every command ran in `$W/feature` with no `--config` and no `--state-dir`, under `env -i` with `GIT_CONFIG_GLOBAL=/dev/null` and `GIT_CONFIG_NOSYSTEM=1`. The item e1 was appended to `$W/main/.harness/state/org/inbox.jsonl` by the probe.

```text
## attach from the feature worktree, no --config, no --state-dir (wake from main: push)
$ ralph org director attach --driver claude
attached d1 (driver=claude, wake=push)
rc=0
## escalated item, then watch --all --once from the feature worktree
$ ralph org watch --all --once
watching every org (interval=30s once=true state-dir=$SCRATCH/probe/wt/main/.harness/state/org (source: git-main-worktree))
rc=0
main wake.sh runs: $SCRATCH/probe/wt/main|main New org inbox items for the director: e1. Read them with ralph org inbox and handle each as the director skill says.
feature wake.sh runs: none
main root: $SCRATCH/probe/wt/main
```

## 5. Probe: start with launch_cmd, push, and no wake_cmd (AC3)

`ralph.toml`: `[org.director]` with `wake = ["pull", "push"]` and `launch_cmd = ["/usr/bin/touch", "$W/launch_ran", "{script_sh}"]`, no wake_cmd. Run in `$W` under `env -i HOME=$W/home PATH=/usr/bin:/bin`.

```text
$ ralph org director start --driver claude --model opus --state-dir $W/ledger --config $W/ralph.toml
org: director: start: the wake has push, but [org.director].wake_cmd is empty (wake pull,push), and [org.director].launch_cmd is set: launch_cmd opens a terminal ralph cannot type into, so ralph org watch --all could not wake the director; set [org.director].wake_cmd (its {target} is director_<token>, the name launch_cmd gives the terminal) or use wake = ["pull"]
rc=1
ledger: [note: empty]
launch_cmd not run
```
