# Verify evidence: org-director-attach

- Date: 2026-10-15 (UTC timestamps in the output read 2026-10-14T18:xx)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-attach, HEAD 676e7f98
- Redaction: the session scratch directory is `$SCRATCH`, the home directory `~`, and the login name `$USER`. The two `jq -e .` steps of the wrapper print the whole settings.json; those dumps are elided. The full wrapper log is `docs/evidence/verify-2026-10-14-184901.log` (gitignored, not committed).

## 1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-14T18:49:01Z
- Mode: static
- Requested scope: changed

==> Running local verifier
==> shellcheck hook + verify scripts
    OK
==> sh -n .claude/hooks/check_mojibake.sh
    OK
==> sh -n .claude/hooks/lib_json.sh
    OK
==> sh -n .claude/hooks/post_edit_verify.sh
    OK
==> sh -n .claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n .claude/hooks/pre_bash_guard.sh
    OK
==> sh -n .claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n .claude/hooks/prompt_gate.sh
    OK
==> sh -n .claude/hooks/ralph-dispatch.sh
    OK
==> sh -n .claude/hooks/session_end_summary.sh
    OK
==> sh -n .claude/hooks/session_start_context.sh
    OK
==> sh -n templates/base/.claude/hooks/check_mojibake.sh
    OK
==> sh -n templates/base/.claude/hooks/lib_json.sh
    OK
==> sh -n templates/base/.claude/hooks/post_edit_verify.sh
    OK
==> sh -n templates/base/.claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n templates/base/.claude/hooks/pre_bash_guard.sh
    OK
==> sh -n templates/base/.claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n templates/base/.claude/hooks/prompt_gate.sh
    OK
==> sh -n templates/base/.claude/hooks/ralph-dispatch.sh
    OK
==> sh -n templates/base/.claude/hooks/session_end_summary.sh
    OK
==> sh -n templates/base/.claude/hooks/session_start_context.sh
    OK
==> awk parse of the guard in .claude/hooks
    OK
==> awk parse of the guard in templates/base/.claude/hooks
    OK
==> jq -e . .claude/settings.json
    (jq output of the file elided)
    OK
==> jq -e . templates/base/.claude/settings.json
    (jq output of the file elided)
    OK
==> Codex hook single-source guard
    OK
==> Codex inline hook detector smoke test
    OK
==> Codex PR provenance policy guard
    OK
==> scripts/check-sync.sh
=== Checking templates/base/ <-> root ===
  TEMPLATE_ONLY  .agents/skills/.gitkeep
  KNOWN_DIFF     .claude/rules/ralph/model-routing.md
  KNOWN_DIFF     .github/workflows/verify.yml
  TEMPLATE_ONLY  .ralph/core/settings.ralph.json
  KNOWN_DIFF     CLAUDE.md
  TEMPLATE_ONLY  docs/architecture/.gitkeep
  TEMPLATE_ONLY  docs/evidence/.gitkeep
  KNOWN_DIFF     docs/quality/quality-gates.md
  TEMPLATE_ONLY  docs/recipes/.gitkeep
  KNOWN_DIFF     docs/recipes/adding-a-language-pack.md
  TEMPLATE_ONLY  docs/references/.gitkeep
  TEMPLATE_ONLY  docs/reports/.gitkeep
  TEMPLATE_ONLY  docs/research/.gitkeep
  TEMPLATE_ONLY  docs/roadmap/.gitkeep
  TEMPLATE_ONLY  docs/tech-debt/.gitkeep
  TEMPLATE_ONLY  ralph.toml

=== Checking AGENTS.md managed block ===

=== Checking templates/packs/ <-> packs/languages/ ===

=== Checking for root-only files not in templates/base/ ===

=== Checking for packs/languages/ files not in templates/packs/ ===

=== Sync Summary ===
  IDENTICAL:      169
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
    OK
==> scripts/check-pipeline-sync.sh
[ok] Canonical source valid
[ok] .claude/skills/implement/SKILL.md: all pipeline steps referenced
[ok] .claude/skills/cross-review/SKILL.md: all pipeline steps referenced
[ok] .claude/rules/ralph/subagent-policy.md: all pipeline steps referenced
[ok] docs/quality/definition-of-done.md: all pipeline steps referenced
[ok] README.md: all pipeline steps referenced
[ok] AGENTS.md: all pipeline steps referenced
    OK
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
    OK
==> scripts/check-template-purity.sh
=== Checking templates for meta-repo-specific references ===

PASS: no meta-repo-specific references found in templates.
    OK
==> tech-debt README plan references
    OK
==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned ef575748..676e7f98 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-14-184901.log
rc=0
```

## 2. Go and sync checks

```text
$ go version
go version go1.26.0 darwin/arm64
$ gofmt -l .
rc=0
$ go vet ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
$ scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
rc=0
$ scripts/check-sync.sh
=== Checking for root-only files not in templates/base/ ===

=== Checking for packs/languages/ files not in templates/packs/ ===

=== Sync Summary ===
  IDENTICAL:      169
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
rc=0
$ git diff --check ef575748...HEAD
rc=0
```

## 3. HEAD binary probe (temp ledger; PATH holds only git, so no osascript, herdr, or agmsg runs)

Script: three phases. (1) no watch: attach, a refused second attach, escalate, the ledger files, `ralph status`. (2) a foreground `ralph org watch --all` on the same temp ledger: escalate, inbox list and show, `--replace`, the replaced id's detach, status, detach, `watch --stop`. (3) `--config` with `enabled = false`: attach, escalate, status, and `--org-id` refused.

```text
## 1. no watch: attach warns, escalate sends at once (inbox_no_watch)

$ ralph org director attach --driver claude --state-dir $SCRATCH/probe/ledger
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1 --state-dir '$SCRATCH/probe/ledger'
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/probe/ledger (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/probe/ledger'
[exit 0]

$ ralph org director attach --driver codex --state-dir $SCRATCH/probe/ledger
org: director: attach: a director is already attached: d1 (driver claude, attached 2026-10-14T18:56:36Z): end it with ralph org director detach --id d1 --state-dir '$SCRATCH/probe/ledger' from the session that attached it (or with --force in place of --id, when that session is gone), or replace it with ralph org director attach --driver codex --replace --state-dir '$SCRATCH/probe/ledger'
[exit 1]

$ ralph org escalate --org-id org-a --text TYPE: QUESTION

plan A or plan B? --state-dir $SCRATCH/probe/ledger
ORG ESCALATION: org=org-a item=e1 type=QUESTION -- recorded in $SCRATCH/probe/ledger/inbox.jsonl; read it with: ralph org inbox show e1 --state-dir '$SCRATCH/probe/ledger'
escalated e1 (org=org-a type=QUESTION)
[exit 0]

--- $SCRATCH/probe/ledger/director.jsonl
{"ts":"2026-10-14T18:56:36Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/cwd","herdr_pane":false}

--- $SCRATCH/probe/ledger/inbox.jsonl
{"ts":"2026-10-14T18:56:36Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nplan A or plan B?","delivery":"now"}
{"ts":"2026-10-14T18:56:36Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"failed: exec: \"osascript\": executable file not found in $PATH"}

--- $SCRATCH/probe/ledger/escalations.jsonl
{"ts":"2026-10-14T18:56:36Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}

$ ralph status --state-dir $SCRATCH/probe/ledger
no org runtime state found (state-dir=$SCRATCH/probe/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running
director: attached d1 (driver=claude, attached at 2026-10-14T18:56:36Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

## 2. a long-running watch accepts work: escalate only records (delivery deadline)

$ ralph status --state-dir $SCRATCH/probe/ledger
no org runtime state found (state-dir=$SCRATCH/probe/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 24695, mark updated 2026-10-14T18:56:36Z)
director: attached d1 (driver=claude, attached at 2026-10-14T18:56:36Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph org escalate --org-id org-a --text TYPE: BLOCKED
TASK_ID: org-a

stuck --state-dir $SCRATCH/probe/ledger
ORG ESCALATION RECORDED: org=org-a item=e2 type=BLOCKED -- left in $SCRATCH/probe/ledger/inbox.jsonl for the director; if it is still open after the ack deadline of ralph org watch --all ([org.director].ack_minutes), the watch sends it to the human; read it with: ralph org inbox show e2 --state-dir '$SCRATCH/probe/ledger'
escalated e2 (org=org-a type=BLOCKED)
[exit 0]

$ ralph org inbox --state-dir $SCRATCH/probe/ledger
ID	STATE	ORG	TYPE	TASK_ID	ESCALATED_AT	NOTIFIED	SUMMARY
e1	open	org-a	QUESTION	-	2026-10-14T18:56:36Z	yes	plan A or plan B?
e2	open	org-a	BLOCKED	org-a	2026-10-14T18:56:36Z	deferred	stuck
[exit 0]

$ ralph org inbox show e2 --state-dir $SCRATCH/probe/ledger
id: e2
state: open
org: org-a
type: BLOCKED
task_id: org-a
escalated_at: 2026-10-14T18:56:36Z
notified: deferred (left to the director, or to the inbox deadlines of ralph org watch --all; to send it to the human now: ralph org inbox notify e2 --state-dir '$SCRATCH/probe/ledger')
message:
  TYPE: BLOCKED
  TASK_ID: org-a

  stuck
events:
  2026-10-14T18:56:36Z escalated
[exit 0]

--- $SCRATCH/probe/ledger/escalations.jsonl
{"ts":"2026-10-14T18:56:36Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}

$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/probe/ledger
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-14T18:56:36Z)
detach it with: ralph org director detach --id d2 --state-dir '$SCRATCH/probe/ledger'
[exit 0]

$ ralph org director detach --id d1 --state-dir $SCRATCH/probe/ledger
org: director: detach d1: the id is not the current registration's: the current registration is d2 (driver codex, attached 2026-10-14T18:56:36Z), so the session that attached d1 was replaced (or d1 has already been detached) and does not need to detach
[exit 1]

$ ralph org director status --state-dir $SCRATCH/probe/ledger
director: attached d2 (driver=codex, attached at 2026-10-14T18:56:36Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d2
driver: codex
wake: pull
attached_at: 2026-10-14T18:56:36Z
user: $USER
cwd: $SCRATCH/probe/cwd
herdr_pane: no
[exit 0]

$ ralph org director status --json --state-dir $SCRATCH/probe/ledger
{
  "enabled": true,
  "state": "attached",
  "registration": {
    "id": "d2",
    "driver": "codex",
    "wake": [
      "pull"
    ],
    "attached_at": "2026-10-14T18:56:36Z",
    "user": "$USER",
    "cwd": "$SCRATCH/probe/cwd",
    "herdr_pane": false
  },
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]

$ ralph org director detach --id d2 --state-dir $SCRATCH/probe/ledger
detached d2
[exit 0]

$ ralph org watch --stop --state-dir $SCRATCH/probe/ledger
stopped ralph org watch --all (pid 24695)
[exit 0]

--- $SCRATCH/probe/watch.log
watching every org (interval=30s once=false state-dir=$SCRATCH/probe/ledger (source: flag))

## 3. enabled = false: attach refused, escalate sends at once (inbox_director_disabled)

$ ralph org director attach --driver claude --state-dir $SCRATCH/probe/ledger --config $SCRATCH/probe/disabled.toml
org: director attach: [org.director].enabled is false in the --config file $SCRATCH/probe/disabled.toml, so ralph org escalate already sends every item to the human at once and leaves nothing to a director; to use one, set [org.director].enabled = true there
[exit 1]

$ ralph org escalate --org-id org-b --text TYPE: QUESTION

which? --state-dir $SCRATCH/probe/ledger --config $SCRATCH/probe/disabled.toml
ORG ESCALATION: org=org-b item=e3 type=QUESTION -- recorded in $SCRATCH/probe/ledger/inbox.jsonl; read it with: ralph org inbox show e3 --state-dir '$SCRATCH/probe/ledger'
escalated e3 (org=org-b type=QUESTION)
[exit 0]

--- $SCRATCH/probe/ledger/escalations.jsonl
{"ts":"2026-10-14T18:56:36Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_watch"}
{"ts":"2026-10-14T18:56:37Z","org_id":"org-b","inbox_id":"e3","subject":"leader","reason":"inbox_director_disabled"}

$ ralph org director status --state-dir $SCRATCH/probe/ledger --config $SCRATCH/probe/disabled.toml
director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once
enabled: false
[exit 0]

$ ralph org director status --org-id org-a --state-dir $SCRATCH/probe/ledger
org: director status: --org-id does not apply: the director registration belongs to the whole ledger, not to one org
[exit 1]

--- $SCRATCH/probe/ledger/director.jsonl
{"ts":"2026-10-14T18:56:36Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/cwd","herdr_pane":false}
{"ts":"2026-10-14T18:56:36Z","id":"d1","event":"detached","user":"$USER","cwd":"$SCRATCH/probe/cwd","herdr_pane":false}
{"ts":"2026-10-14T18:56:36Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"user":"$USER","cwd":"$SCRATCH/probe/cwd","herdr_pane":false}
{"ts":"2026-10-14T18:56:36Z","id":"d2","event":"detached","user":"$USER","cwd":"$SCRATCH/probe/cwd","herdr_pane":false}
```

## 4. Help texts of the HEAD binary (director verbs; inherited flags dropped)

```text
### ralph org director --help
ralph org director keeps the director registration of the ledger in
<state-dir>/director.jsonl. A director is an interactive session the
human talks to, promoted with the /director skill: it drafts split
plans, starts feature orgs, and handles the org inbox. One director is
registered at a time for every org in the ledger, so these verbs take
no --org-id. attach records the session and prints the registration's
id (d1, d2, ...); it is refused while another director is attached,
unless --replace is given. detach ends the registration and needs that
id, so a session that was replaced is refused and need not detach;
--force ends the current registration whatever its id, for a human
cleaning up. status shows the registration and [org.director].enabled.

The registration does not change what `ralph org escalate` does:
[org.director].enabled and the cross-org watch decide that (see
ralph org escalate --help). Nor does it mean the session is still
running: an item the director does not ack reaches the human at the
ack deadline of `ralph org watch --all`. A director that ralph starts
by itself (ralph org director start) is not available yet.

Usage:
  ralph org director [flags]
  ralph org director [command]

Available Commands:
  attach      Register this session as the director of the ledger
  detach      End the director registration of the ledger
  status      Show the director registration of the ledger and [org.director].enabled

Flags:
  -h, --help   help for director

Global Flags:

Use "ralph org director [command] --help" for more information about a command.
### ralph org director attach --help
ralph org director attach registers the session that runs it as the
director of the ledger (an attached event in <state-dir>/director.jsonl)
and prints the registration's id: attached d1 (driver=claude). Keep the
id: `ralph org director detach --id <id>` needs it, and
`ralph org director status` shows the current one, so a session that
finds another id there was replaced. --driver is the session's agent,
claude or codex. The event also records who ran attach: USER, the
current directory, and whether HERDR_PANE_ID was set (not its value).

One director is attached at a time: while one is, attach is refused
unless --replace is given, which ends the current registration and
records the new one in one write, under one lock. The replaced
session's detach is then refused, and it need not detach.

attach is refused, and writes nothing, when [org.director].enabled is
false: escalate then sends every item to the human at once. Without
--config, enabled is read from the main worktree's ralph.toml when the
ledger is the main worktree's, as `ralph org escalate` reads it.

When no `ralph org watch --all` accepts work on the ledger, attach still
records the registration and warns on stderr: until one does, escalate
sends items to the human at once instead of leaving them to the
director. Start one with `ralph org watch --all --detach`.

Usage:
  ralph org director attach [flags]

Flags:
      --driver string   the session's agent: claude or codex (required)
  -h, --help            help for attach
      --replace         end the current registration and record this one instead of refusing

Global Flags:
### ralph org director detach --help
ralph org director detach ends the current director registration of the
ledger (a detached event in <state-dir>/director.jsonl). Give --id, the
id attach printed. When it is not the current registration's, detach is
refused and writes nothing: that session was replaced with
`ralph org director attach --replace` (or its registration has already
ended), and it need not detach. --force ends the current registration
whatever its id, for a human cleaning up after a session that is gone,
and is recorded as forced; it cannot be combined with --id. With no
registration, detach exits 1. detach reads no ralph.toml (nor the main
worktree's), so one that does not load does not block it.

Usage:
  ralph org director detach [flags]

Flags:
      --force       end the current registration whatever its id (a human's cleanup)
  -h, --help        help for detach
      --id string   the id of the registration to end (d1, d2, ...), as attach printed it

Global Flags:
### ralph org director status --help
ralph org director status shows [org.director].enabled, read as
`ralph org escalate` reads it, and the current director registration of
the ledger: the director: line of `ralph status` (attached, not
attached, disabled, or unreadable), then enabled and, when a director is
attached, the registration's fields, or all of it as JSON with --json.
The registration does not tell whether its session is still running.

Usage:
  ralph org director status [flags]

Flags:
  -h, --help   help for status
      --json   machine-readable JSON output

Global Flags:
```

## Cycle 2 (HEAD 35a453aa)

- Date: 2026-10-15 (UTC timestamps in the output read 2026-10-14T20:xx)
- Scope: `git diff 7cd73066..HEAD` (the restore of 634b019a, the self-review fixes of f4c1a394, and the sync-docs of 624f603b), read against `git diff ef575748...HEAD`
- Redaction as in cycle 1: the session scratch directory is `$SCRATCH`, the home directory `~`, and the login name `$USER`. The two `jq -e .` dumps of settings.json are elided. The full wrapper log is `docs/evidence/verify-2026-10-14-204359.log` (gitignored, not committed).

### C2-1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-14T20:43:59Z
- Mode: static
- Requested scope: changed

==> Running local verifier
==> shellcheck hook + verify scripts
    OK
==> sh -n .claude/hooks/check_mojibake.sh
    OK
==> sh -n .claude/hooks/lib_json.sh
    OK
==> sh -n .claude/hooks/post_edit_verify.sh
    OK
==> sh -n .claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n .claude/hooks/pre_bash_guard.sh
    OK
==> sh -n .claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n .claude/hooks/prompt_gate.sh
    OK
==> sh -n .claude/hooks/ralph-dispatch.sh
    OK
==> sh -n .claude/hooks/session_end_summary.sh
    OK
==> sh -n .claude/hooks/session_start_context.sh
    OK
==> sh -n templates/base/.claude/hooks/check_mojibake.sh
    OK
==> sh -n templates/base/.claude/hooks/lib_json.sh
    OK
==> sh -n templates/base/.claude/hooks/post_edit_verify.sh
    OK
==> sh -n templates/base/.claude/hooks/post_tool_failure_feedback.sh
    OK
==> sh -n templates/base/.claude/hooks/pre_bash_guard.sh
    OK
==> sh -n templates/base/.claude/hooks/precompact_checkpoint.sh
    OK
==> sh -n templates/base/.claude/hooks/prompt_gate.sh
    OK
==> sh -n templates/base/.claude/hooks/ralph-dispatch.sh
    OK
==> sh -n templates/base/.claude/hooks/session_end_summary.sh
    OK
==> sh -n templates/base/.claude/hooks/session_start_context.sh
    OK
==> awk parse of the guard in .claude/hooks
    OK
==> awk parse of the guard in templates/base/.claude/hooks
    OK
==> jq -e . .claude/settings.json
    (settings.json dump elided)
    OK
==> jq -e . templates/base/.claude/settings.json
    (settings.json dump elided)
    OK
==> Codex hook single-source guard
    OK
==> Codex inline hook detector smoke test
    OK
==> Codex PR provenance policy guard
    OK
==> scripts/check-sync.sh
=== Checking templates/base/ <-> root ===
  TEMPLATE_ONLY  .agents/skills/.gitkeep
  KNOWN_DIFF     .claude/rules/ralph/model-routing.md
  KNOWN_DIFF     .github/workflows/verify.yml
  TEMPLATE_ONLY  .ralph/core/settings.ralph.json
  KNOWN_DIFF     CLAUDE.md
  TEMPLATE_ONLY  docs/architecture/.gitkeep
  TEMPLATE_ONLY  docs/evidence/.gitkeep
  KNOWN_DIFF     docs/quality/quality-gates.md
  TEMPLATE_ONLY  docs/recipes/.gitkeep
  KNOWN_DIFF     docs/recipes/adding-a-language-pack.md
  TEMPLATE_ONLY  docs/references/.gitkeep
  TEMPLATE_ONLY  docs/reports/.gitkeep
  TEMPLATE_ONLY  docs/research/.gitkeep
  TEMPLATE_ONLY  docs/roadmap/.gitkeep
  TEMPLATE_ONLY  docs/tech-debt/.gitkeep
  TEMPLATE_ONLY  ralph.toml

=== Checking AGENTS.md managed block ===

=== Checking templates/packs/ <-> packs/languages/ ===

=== Checking for root-only files not in templates/base/ ===

=== Checking for packs/languages/ files not in templates/packs/ ===

=== Sync Summary ===
  IDENTICAL:      169
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
    OK
==> scripts/check-pipeline-sync.sh
[ok] Canonical source valid
[ok] .claude/skills/implement/SKILL.md: all pipeline steps referenced
[ok] .claude/skills/cross-review/SKILL.md: all pipeline steps referenced
[ok] .claude/rules/ralph/subagent-policy.md: all pipeline steps referenced
[ok] docs/quality/definition-of-done.md: all pipeline steps referenced
[ok] README.md: all pipeline steps referenced
[ok] AGENTS.md: all pipeline steps referenced
    OK
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
    OK
==> scripts/check-template-purity.sh
=== Checking templates for meta-repo-specific references ===

PASS: no meta-repo-specific references found in templates.
    OK
==> tech-debt README plan references
    OK
==> Language scope: full fallback (unclassified:scripts/ralph-config.sh)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned ef575748..35a453aa against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-14-204359.log
rc=0
```

### C2-2. Go and sync checks

```text
$ go version
go version go1.26.0 darwin/arm64
$ gofmt -l .
rc=0
$ go vet ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
rc=0
$ ./scripts/check-sync.sh | tail -9

=== Sync Summary ===
  IDENTICAL:      169
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
rc=0
$ git diff --check ef575748...HEAD
rc=0
$ git diff --check 7cd73066..HEAD
rc=0
```

### C2-3. HEAD binary probe: AC2 refusals and failed writes leave director.jsonl as it was

Script `ac2-probe.sh` (scratch): PATH holds only git, the ledger and the cwd are scratch dirs. Each `cmp` line compares director.jsonl with a copy taken just before the command. The failed writes make director.jsonl read-only (`chmod 444`), so the open fails; a write that fails partway is not reproduced here (C2-4).

```text
## refusals before any registration
$ ralph org director detach --id d1 --state-dir $SCRATCH/c2/ac2/ledger
org: director: detach d1: no director is attached
[exit 1]
cmp (detach, none attached): director.jsonl still missing

$ ralph org director attach --driver gpt --state-dir $SCRATCH/c2/ac2/ledger
org: director attach: --driver "gpt" is not claude or codex
[exit 1]
cmp (bad driver): director.jsonl still missing

$ ralph org director attach --driver claude --org-id org-a --state-dir $SCRATCH/c2/ac2/ledger
org: director attach: --org-id does not apply: the director registration belongs to the whole ledger, not to one org
[exit 1]
cmp (--org-id): director.jsonl still missing

$ ralph org director attach --driver claude --config $SCRATCH/c2/ac2/disabled.toml --state-dir $SCRATCH/c2/ac2/ledger
org: director attach: [org.director].enabled is false in the --config file $SCRATCH/c2/ac2/disabled.toml, so ralph org escalate already sends every item to the human at once and leaves nothing to a director; to use one, set [org.director].enabled = true there
[exit 1]
cmp (enabled = false): director.jsonl still missing

## attach d1, then refusals
$ ralph org director attach --driver claude --state-dir $SCRATCH/c2/ac2/ledger
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1 --state-dir '$SCRATCH/c2/ac2/ledger'
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/c2/ac2/ledger (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/c2/ac2/ledger'
[exit 0]

$ ralph org director attach --driver codex --state-dir $SCRATCH/c2/ac2/ledger
org: director: attach: a director is already attached: d1 (driver claude, attached 2026-10-14T20:50:00Z): end it with ralph org director detach --id d1 --state-dir '$SCRATCH/c2/ac2/ledger' from the session that attached it (or with --force in place of --id, when that session is gone), or replace it with ralph org director attach --driver codex --replace --state-dir '$SCRATCH/c2/ac2/ledger'
[exit 1]
cmp (second attach): director.jsonl byte-identical

$ ralph org director detach --state-dir $SCRATCH/c2/ac2/ledger
org: director detach: give --id <id>, the id attach printed, or --force
[exit 1]
cmp (detach, no --id): director.jsonl byte-identical

$ ralph org director detach --id d1 --force --state-dir $SCRATCH/c2/ac2/ledger
org: director detach: --id and --force cannot be used together
[exit 1]
cmp (detach, --id and --force): director.jsonl byte-identical

$ ralph org director detach --id x1 --state-dir $SCRATCH/c2/ac2/ledger
org: director: detach: "x1" is not a registration id (d<N>, such as d1)
[exit 1]
cmp (detach, bad id): director.jsonl byte-identical

$ ralph org director detach --id d7 --state-dir $SCRATCH/c2/ac2/ledger
org: director: detach d7: the id is not the current registration's: the current registration is d1 (driver claude, attached 2026-10-14T20:50:00Z), so the session that attached d7 was replaced (or d7 has already been detached) and does not need to detach
[exit 1]
cmp (detach, other id): director.jsonl byte-identical

## failed writes: director.jsonl read-only (open fails)
$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/c2/ac2/ledger
org: director: attach: nothing was recorded, so d1 (driver claude, attached 2026-10-14T20:50:00Z) is still the current registration: open $SCRATCH/c2/ac2/ledger/director.jsonl: open $SCRATCH/c2/ac2/ledger/director.jsonl: permission denied
[exit 1]
cmp (replace, read-only): director.jsonl byte-identical

$ ralph org director detach --id d1 --state-dir $SCRATCH/c2/ac2/ledger
org: director: detach d1: nothing was recorded, so d1 (driver claude, attached 2026-10-14T20:50:00Z) is still the current registration: open $SCRATCH/c2/ac2/ledger/director.jsonl: open $SCRATCH/c2/ac2/ledger/director.jsonl: permission denied
[exit 1]
cmp (detach, read-only): director.jsonl byte-identical

$ ralph org director detach --force --state-dir $SCRATCH/c2/ac2/ledger
org: director: force detach: nothing was recorded, so d1 (driver claude, attached 2026-10-14T20:50:00Z) is still the current registration: open $SCRATCH/c2/ac2/ledger/director.jsonl: open $SCRATCH/c2/ac2/ledger/director.jsonl: permission denied
[exit 1]
cmp (force detach, read-only): director.jsonl byte-identical

## replace d1 with d2, then the replaced session's detach
$ ralph org director attach --driver codex --replace --state-dir $SCRATCH/c2/ac2/ledger
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-14T20:50:00Z)
detach it with: ralph org director detach --id d2 --state-dir '$SCRATCH/c2/ac2/ledger'
warning: director attach: d2 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/c2/ac2/ledger (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/c2/ac2/ledger'
[exit 0]

$ ralph org director detach --id d1 --state-dir $SCRATCH/c2/ac2/ledger
org: director: detach d1: the id is not the current registration's: the current registration is d2 (driver codex, attached 2026-10-14T20:50:01Z), so the session that attached d1 was replaced (or d1 has already been detached) and does not need to detach
[exit 1]
cmp (replaced id detach): director.jsonl byte-identical

$ ralph org director status --state-dir $SCRATCH/c2/ac2/ledger
director: attached d2 (driver=codex, attached at 2026-10-14T20:50:01Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d2
driver: codex
wake: pull
attached_at: 2026-10-14T20:50:01Z
user: $USER
cwd: $SCRATCH/c2/ac2/cwd
herdr_pane: no
[exit 0]

$ ralph org director detach --id d2 --state-dir $SCRATCH/c2/ac2/ledger
detached d2
[exit 0]

--- director.jsonl (event and id per line)
attached d1
detached d1
attached d2
detached d2
```

### C2-4. Attempt at a real partial write with RLIMIT_FSIZE (not reproduced)

Scripts `fsize-try.sh` and `fsize-over.sh` (scratch, `/bin/bash`, `ulimit -f 1` in a subshell around one `attach --replace`). In the first, the cwd is padded with 150 characters so that the old line and the detached fit under the limit and the attached does not; in the second, director.jsonl is already past the limit before the write. Both writes went through whole, so on this macOS the limit does not stop this append, and the cut back after a write that fails partway is covered only by the write seam of the Go tests.

```text
ulimit -f 1; size before:      435
subshell rc=0
[exit 0]
--- stdout
attached d2 (driver=codex), replacing d1 (driver=claude, attached at 2026-10-14T20:52:12Z)
detach it with: ralph org director detach --id d2 --state-dir '$SCRATCH/c2/try/ledger'
--- stderr
warning: director attach: d2 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/c2/try/ledger (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/c2/try/ledger'
size after:     1270
--- director.jsonl (per line: bytes, last char)
435 bytes: {"ts":"2026-10-14T20:52:12Z","id":"d1","event":"attached","d
401 bytes: {"ts":"2026-10-14T20:52:12Z","id":"d1","event":"detached","u
434 bytes: {"ts":"2026-10-14T20:52:12Z","id":"d2","event":"attached","d
0000000   \n

size before:     1898 (ulimit -f 1 = 1024 bytes in bash)
subshell rc=0
[exit 0]
--- stdout
attached d5 (driver=codex), replacing d4 (driver=codex, attached at 2026-10-14T20:52:12Z)
detach it with: ralph org director detach --id d5 --state-dir '$SCRATCH/c2/over/ledger'
--- stderr
warning: director attach: d5 is recorded, but no ralph org watch --all accepts work on the ledger $SCRATCH/c2/over/ledger (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '$SCRATCH/c2/over/ledger'
size after:     2435
cmp: director.jsonl changed
director: attached d5 (driver=codex, attached at 2026-10-14T20:52:12Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
```

### C2-5. ralph org director status --help of the HEAD binary (cycle-1 V4; the description only, usage and flags dropped)

```text
ralph org director status shows [org.director].enabled, read as
`ralph org escalate` reads it, and the current director registration of
the ledger: the director: line of `ralph status` (attached, not
attached, disabled, or unreadable), then enabled and, when a director is
attached, the registration's fields, or all of it as JSON with --json.
With enabled false the state is disabled whether or not a registration is
recorded; one that is still recorded is named and its fields are shown.
The registration does not tell whether its session is still running.
```
