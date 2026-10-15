# Verify evidence: org-director-start

- Date: 2026-10-15 (UTC timestamps in the output read 2026-10-15T08:xx; JST is UTC+9)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD dbd67a58 (base 9cbf7080)
- Redaction: the session scratch directory is `$SCRATCH`, the home directory `~`, the login name `$USER`, the host name `<host>`, and the codex session UUID `<session-uuid>`. The two `jq -e .` steps of the wrapper print the whole settings.json; those dumps are elided. The full wrapper log is `docs/evidence/verify-2026-10-15-084118.log`, and the raw outputs of this file are in `docs/evidence/verify-2026-10-15-org-director-start.log` (both gitignored, not committed).
- Lines in `[brackets]` starting with `note:` were added when this file was written; everything else is command output.

## 1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-15T08:41:18Z
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
[note: settings.json dump elided]
    OK
==> jq -e . templates/base/.claude/settings.json
[note: settings.json dump elided]
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
==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned 9cbf7080..dbd67a58 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-15-084118.log
rc=0
```

`0 issues.` is golangci-lint; staticcheck prints nothing on success.

## 2. Go checks (local go1.26.0, and go1.25.8 per go.mod)

```text
$ go version
go version go1.26.0 darwin/arm64
== gofmt -l (go1.26)
rc=0
== go vet ./... (go1.26)
rc=0
== GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
== GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
== GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
== go1.25.8 gofmt -l ./cmd ./internal  ($(GOTOOLCHAIN=go1.25.8 go env GOROOT)/bin/gofmt)
rc=0
== grep go directive
3:go 1.25.8
== git diff --check 9cbf7080...HEAD
rc=0
```

Every `-l` printed no file name.

## 3. Self-review fixes (da818b4f, dbd67a58)

```text
$ grep -rn 'directorRoleSpawnErr\|[^A-Za-z]modelAllowedForRole' --include='*.go' .
[note: no output]

$ grep -rn 'confirmSeatPane(\|confirmOrgWorkspace(' internal --include='*.go'   (callers only)
internal/org/director_stop.go:313:		paneGone, err := o.confirmSeatPane(wsLabel, tabLabel, reg.PaneID, directorLabelWords)
internal/org/director_stop.go:321:		wsGone, err := o.confirmOrgWorkspace(wsLabel, reg.WorkspaceID, directorLabelWords)
internal/org/verbs.go:884:	switch wsGone, wsErr := o.confirmOrgWorkspace(workspaceLabel, workspaceID, words); {
internal/org/verbs.go:1138:	gone, err := o.confirmSeatPane(p.OrgID, p.Seat, paneID, seatLabelWords)
internal/org/verbs.go:1239:	switch gone, err := o.confirmSeatPane(orgID, seatID, paneID, seatLabelWords); {
internal/org/verbs.go:1325:	switch gone, err := o.confirmOrgWorkspace(orgID, workspaceID, seatLabelWords); {
internal/org/verbs.go:2135:	gone, closeErr := o.confirmOrgWorkspace(p.OrgID, workspaceID, seatLabelWords)

$ awk (comment lines over 82 columns in the files L6 named, plus director_start.go, director_stop.go, org_director.go, org_director_headless.go)
internal/org/spawn.go:53: 83
internal/org/spawn.go:93: 91
internal/org/spawn.go:1640: 90
internal/org/spawn.go:2193: 86
[note: git blame puts all four in eb30b172, aaaef61a, b133022c, c04186eb, none on this branch]

$ md5 -q of the four /director SKILL.md mirrors
6509f6c71f7e855db1c4b83f56794051
6509f6c71f7e855db1c4b83f56794051
6509f6c71f7e855db1c4b83f56794051
6509f6c71f7e855db1c4b83f56794051

$ awk -F'|' column count of every tech-debt row this branch changed (171, 189, 192, 212, 234-241, 245, 250-255)
[note: NF=7 (5 columns) for every one; the 10 rows with another count (22, 24, 32, 46, 57, 79, 90, 160, 163, 216) are the same at 9cbf7080]
```

## 4. Live check on an isolated herdr server (AC13, the plan's Evidence to capture)

### Setup

- herdr 0.7.5, codex-cli 0.160.1, Claude Code 2.1.291. HEAD binary built with `go build -o $SCRATCH/ds/bin/ralph ./cmd/ralph` from the clean worktree at dbd67a58.
- Before: `pgrep -fl 'herdr server'` printed nothing (exit 1): no herdr server of the user's was running. `~/.config/herdr/session.json` sha256 `004173699fb3c9320508a7442ed9d9f439d853d20a0732ec4cf8d0b4eb7811db`, `~/.config/herdr/herdr-server.log` last modified 1 Oct 14:35.
- Dedicated server, started from `/tmp/rdst` with nothing but these variables (`env -i`): `HOME=~ USER LOGNAME SHELL=/bin/bash LANG=en_US.UTF-8 TERM=xterm-256color TMPDIR=/tmp/rdst/tmp/ PATH=$SCRATCH/ds/bin:<codex bin>:<claude bin>:<node bin>:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin XDG_CONFIG_HOME=/tmp/rdst/cfg XDG_STATE_HOME=/tmp/rdst/state HERDR_SOCKET_PATH=/tmp/rdst/h.sock`. The pane shell is bash, so the zsh aliases of `codex` and `claude` in the user's `~/.zshrc` do not apply (bash reads only `~/.profile`, which adds `~/.local/bin` to PATH).
- `lsof -p <pid>` of the server: its only log is `/tmp/rdst/cfg/herdr/herdr-server.log`; stdout and stderr go to `$SCRATCH/ds/herdr-server.out`. Its startup lines: `api socket: /tmp/rdst/h.sock`, `logs: /tmp/rdst/cfg/herdr/herdr-server.log`.
- The ralph commands ran with `PATH=$SCRATCH/ds/hbin` (symlinks to herdr, git, and the HEAD ralph only, so no osascript could be found), `HERDR_SOCKET_PATH=/tmp/rdst/h.sock`, `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID` unset, the scratch ledger `--state-dir $SCRATCH/ds/state`, and the scratch config `--config $SCRATCH/ds/ralph.toml`:

```toml
[org]
driver_pool = ["claude", "codex"]
model_pool = [
  { driver = "claude", model = "haiku" },
  { driver = "codex", model = "gpt-6-astra" },
]

[org.permissions]
default = "autonomous"
codex_verified = false

[org.permissions.roles]
director = "guarded"

[org.director]
enabled = true
```

- codex model: `gpt-6-astra`, listed in `~/.codex/models_cache.json` (visibility `list`, no upgrade notice).

### L1. director start --driver codex

```text
$ ralph org director start --driver codex --model gpt-6-astra --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
ralph org director start: started the cross-org watch (pid 44058); log: $SCRATCH/ds/state/watch.log
started d1 (driver=codex, model=gpt-6-astra, permission=guarded, pane=w1:p2, workspace=w1)
talk to the director in herdr: pane w1:p2, in the tab director_9da3599cba71 of the workspace labelled ralph_director_9da3599cba71
stop it with: ralph org director stop --id d1 --state-dir '$SCRATCH/ds/state'
[exit 0]
```

`roles.director = "guarded"` won over `default = "autonomous"`, and the codex refusal of autonomous without `codex_verified` did not apply.

### L2. herdr's view, director.jsonl, and the status lines of d1

```text
$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w1:t1","agent_status":"working","focused":true,"label":"ralph_director_9da3599cba71","number":1,"pane_count":2,"tab_count":2,"workspace_id":"w1"}]}}
[exit 0]

$ herdr tab list --workspace w1
{"id":"cli:tab:list","result":{"tabs":[{"agent_status":"unknown","focused":true,"label":"1","number":1,"pane_count":1,"tab_id":"w1:t1","workspace_id":"w1"},{"agent_status":"working","focused":false,"label":"director_9da3599cba71","number":2,"pane_count":1,"tab_id":"w1:t2","workspace_id":"w1"}],"type":"tab_list"}}
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[{"agent":"codex","agent_status":"working","cwd":"$SCRATCH/ds/cwd","focused":false,"foreground_cwd":"$SCRATCH/ds/cwd","interactive_ready":true,"name":"director_9da3599cba71","pane_id":"w1:p2","revision":56,"state_change_seq":2,"tab_id":"w1:t2","terminal_id":"term_65ddd273d098f2","terminal_title":"⠸ 役割指示を読み込む | cwd","terminal_title_stripped":"役割指示を読み込む | cwd","workspace_id":"w1"}],"type":"agent_list"}}
[exit 0]
```

herdr returned the workspace label `ralph_director_9da3599cba71` (27 characters, two `_`) and the tab label and agent name `director_9da3599cba71` (21 characters) unchanged. The workspace also holds herdr's first tab, labelled `1` (pane w1:p1).

```text
## director.jsonl
{"ts":"2026-10-15T08:51:27Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"herdr","token":"9da3599cba71","model":"gpt-6-astra","permission_mode":"guarded","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:51:27Z","id":"d1","event":"workspace_created","workspace_id":"w1","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:51:27Z","id":"d1","event":"pane_created","pane_id":"w1:p2","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:51:30Z","id":"d1","event":"launched","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
## ledger entries
$SCRATCH/ds/state:
director.jsonl
director.lock
prompts
watch-alive.json
watch.lock
watch.log

$SCRATCH/ds/state/prompts:
director_9da3599cba71.md

$ ralph status --state-dir $SCRATCH/ds/state
no org runtime state found (state-dir=$SCRATCH/ds/state (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 44058, mark updated 2026-10-15T08:51:57Z)
director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane w1:p2, workspace w1, attached at 2026-10-15T08:51:27Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph org director status --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane w1:p2, workspace w1, attached at 2026-10-15T08:51:27Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: codex
wake: pull
attached_at: 2026-10-15T08:51:27Z
user: $USER
cwd: $SCRATCH/ds/cwd
herdr_pane: no
launch: herdr
state: running
model: gpt-6-astra
permission_mode: guarded
token: 9da3599cba71
workspace_id: w1
pane_id: w1:p2
[exit 0]

$ ralph org director status --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml --json
{
  "enabled": true,
  "state": "attached",
  "registration": {
    "id": "d1",
    "driver": "codex",
    "wake": [
      "pull"
    ],
    "pane_id": "w1:p2",
    "launch": "herdr",
    "state": "running",
    "token": "9da3599cba71",
    "model": "gpt-6-astra",
    "permission_mode": "guarded",
    "workspace_id": "w1",
    "attached_at": "2026-10-15T08:51:27Z",
    "user": "$USER",
    "cwd": "$SCRATCH/ds/cwd",
    "herdr_pane": false
  },
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]
```

No `manifest.jsonl` and no `model-receipts.jsonl` in the ledger.

### L3. What the codex director did (pane screen, and its codex session record)

Pane `w1:p2` just before the stop (`herdr pane read w1:p2 --lines 60`):

```text
› 役割指示を読み込んで従ってください: /private/tmp/c…
• Hook failed
  └ hook returned invalid session start JSON output

• 指定の役割指示を読み込み、内容に沿って進めます。

• Explored
  └ Read director_9da3599cba71.md
    + Show details

• Ran ralph org director status --state-dir '/private…
  └       --state-dir string   org manifest/receipts …

    Use "ralph org [command] --help" for more informa…
    + 28 lines (ctrl+t to expand)

• 役割指示を読み込みました


› Ask Codex to do anything

  GPT-6-Astra · max · Context 7% used
  ← for agents · ? for sho  ⚠ 8 warnings · f2 to view
```

The codex session record of this run (`~/.codex/sessions/2026/10/15/rollout-2026-10-15T17-51-35-<session-uuid>.jsonl`, read with jq):

```text
turn_context: {"model":"gpt-6-astra","approval_policy":"never","sandbox_policy":{"type":"danger-full-access"},"cwd":"$SCRATCH/ds/cwd"}
tool call 1: cat $SCRATCH/ds/state/prompts/director_9da3599cba71.md   (exit 0; the rendered prompt, `--state-dir '<ledger>'` quoted)
tool call 2: ralph org director status --state-dir '$SCRATCH/ds/state'   (exit 0)
  output begins: "ralph org drives the org-runtime mechanism layer: ... Available Commands: disband read report send spawn
  start (Spawn a headless lead seat (sugar over `ralph org spawn --role lead`)) status stop wait watch ..."
final assistant message:
  指定の状態確認コマンドが登録情報ではなく `ralph org` のヘルプを返したため、登録 ID が `d1` か確認できませんでした。役割指示に従い、以降の操作を進めず停止しています。
```

- The `ralph` the codex shell found was the Homebrew build (`/opt/homebrew/bin/ralph`, `ralph 5.1.0 (8c9f6f1 2026-08-24T09:44:24Z)`, whose `ralph org` has no `director` verb and still has the `lead` role), not the HEAD binary first on the herdr server's PATH: codex ran its commands through a shell that rebuilt PATH. The status output was the `ralph org` help, with exit 0.
- The director then did what the role prompt's first step says: it could not confirm that the current registration is `d1`, ran nothing else, and said so in its pane.
- `guarded` passed no permission flag, so codex ran with `~/.codex/config.toml`'s `approval_policy = "never"` and `sandbox_mode = "danger-full-access"`.
- `Hook failed: hook returned invalid session start JSON output` comes from the user's own codex hooks (the scratch cwd has no `.codex/`); it is not from this repo.

### L4. director stop --id d1

```text
$ ralph org director stop --id d1 --state-dir $SCRATCH/ds/state
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[]}}
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}
[exit 0]
## director.jsonl after stop (last two lines; the first four are those of L2)
{"ts":"2026-10-15T08:52:18Z","id":"d1","event":"detached","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:52:18Z","id":"d1","event":"stopped","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}

$ ralph status --state-dir $SCRATCH/ds/state
no org runtime state found (state-dir=$SCRATCH/ds/state (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 44058, mark updated 2026-10-15T08:51:57Z)
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph org director status --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
enabled: true
[exit 0]
```

### Restart of the dedicated server for claude

`~/.claude/settings.json` sets `permissions.defaultMode = "bypassPermissions"` (with `skipDangerousModePermissionPrompt = true`). A guarded claude director gets no `--permission-mode` flag, so with that settings file it would run in bypassPermissions mode. To keep the live director out of it, the dedicated server was stopped (`HERDR_SOCKET_PATH=/tmp/rdst/h.sock herdr server stop`, exit 0, pid 35480 gone, `pgrep` empty) and started again with the same variables plus `CLAUDE_CONFIG_DIR=/tmp/rdst/claude-cfg` (an empty directory): claude in the pane reads neither `~/.claude/settings.json` nor `~/.claude.json`. `ps eww` showed the variable in the new server's environment (pid 77399); `lsof` again showed only `/tmp/rdst/cfg/herdr/herdr-server.log`. The scratch config stayed `director = "guarded"`.

### L5. director start --driver claude

```text
$ ralph org director start --driver claude --model haiku --timeout-ms 30000 --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
started d2 (driver=claude, model=haiku, permission=guarded, pane=w1:p2, workspace=w1)
talk to the director in herdr: pane w1:p2, in the tab director_82130355444e of the workspace labelled ralph_director_82130355444e
stop it with: ralph org director stop --id d2 --state-dir '$SCRATCH/ds/state'
[exit 0]

$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w1:t1","agent_status":"idle","focused":true,"label":"ralph_director_82130355444e","number":1,"pane_count":2,"tab_count":2,"workspace_id":"w1"}]}}
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[{"agent":"claude","agent_status":"idle","cwd":"$SCRATCH/ds/cwd","focused":false,"foreground_cwd":"$SCRATCH/ds/cwd","interactive_ready":true,"name":"director_82130355444e","pane_id":"w1:p2","revision":0,"state_change_seq":1,"tab_id":"w1:t2","terminal_id":"term_65ddd3303751a2","workspace_id":"w1"}],"type":"agent_list"}}
[exit 0]
## director.jsonl (d2 lines)
{"ts":"2026-10-15T08:54:45Z","id":"d2","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"82130355444e","model":"haiku","permission_mode":"guarded","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:54:45Z","id":"d2","event":"workspace_created","workspace_id":"w1","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:54:45Z","id":"d2","event":"pane_created","pane_id":"w1:p2","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:54:48Z","id":"d2","event":"launched","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
```

The restarted herdr numbered its ids from `w1` again, so d2 got the same workspace and pane ids (`w1`, `w1:p2`) that the stopped d1 had recorded. The running watch (pid 44058) was reused: no second `started the cross-org watch` line.

### L6. The claude pane and the status lines of d2

```text
## L6. claude pane screen

The default interactive shell is now zsh.
To update your account to use zsh, please run `chsh -s
 /bin/zsh`.
For more details, please visit https://support.apple.c
om/kb/HT208050.
<host>:cwd $USER$ claude --model hai
ku '役割指示を読み込んで従ってください: $SCRATCH/ds/state/prompts/director_82130355444e.md'
[note: the pane wrapped this path over three lines; it is joined here for the redaction]
Welcome to Claude Code v2.1.291

 Let's get started.

 Choose the text style that looks best with your
 terminal
 To change this later, run /theme

     Auto (match terminal)
 ❯ ✔ Dark mode
     Light mode
     Dark mode (colorblind-friendly)
     Light mode (colorblind-friendly)
     Dark mode (ANSI colors only)
     Light mode (ANSI colors only)

 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
  1  function greet() {
  2 -  console.log("Hello, World!");
  2 +  console.log("Hello, Claude!");
  3  }
 ╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌
  Syntax theme: Monokai Extended (ctrl+t to disable)

$ herdr tab list --workspace w1
{"id":"cli:tab:list","result":{"tabs":[{"agent_status":"unknown","focused":true,"label":"1","number":1,"pane_count":1,"tab_id":"w1:t1","workspace_id":"w1"},{"agent_status":"idle","focused":false,"label":"director_82130355444e","number":2,"pane_count":1,"tab_id":"w1:t2","workspace_id":"w1"}],"type":"tab_list"}}
[exit 0]

$ ralph status --state-dir $SCRATCH/ds/state
no org runtime state found (state-dir=$SCRATCH/ds/state (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 44058, mark updated 2026-10-15T08:54:27Z)
director: attached d2 (driver=claude, headless running, model=haiku, pane w1:p2, workspace w1, attached at 2026-10-15T08:54:45Z, by $USER) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph org director status --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml --json
{
  "enabled": true,
  "state": "attached",
  "registration": {
    "id": "d2",
    "driver": "claude",
    "wake": [
      "pull"
    ],
    "pane_id": "w1:p2",
    "launch": "herdr",
    "state": "running",
    "token": "82130355444e",
    "model": "haiku",
    "permission_mode": "guarded",
    "workspace_id": "w1",
    "attached_at": "2026-10-15T08:54:45Z",
    "user": "$USER",
    "cwd": "$SCRATCH/ds/cwd",
    "herdr_pane": false
  },
  "corrupt_lines": 0,
  "ignored_events": 0
}
[exit 0]
```

herdr typed `claude --model haiku '<the prompt file's pointer>'` into the bash pane: no permission flag for guarded, `--model`, then the pointer. With the empty `CLAUDE_CONFIG_DIR`, Claude Code stopped at its first-run theme screen, which herdr already counted as `interactive_ready`, so start wrote `launched`. This claude director never read its prompt.

### L7. Refusals while d2 runs

The variable before each command (not echoed by the transcript) is given in a note.

```text
director.jsonl md5 before: ed929ae3f9b802709b02897e26b8c9e8

$ ralph org director start --driver codex --model gpt-6-astra --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
org: director: start: a director is already attached: d2 (driver claude, attached 2026-10-15T08:54:45Z, headless running): stop it with ralph org director stop --id d2 --state-dir '$SCRATCH/ds/state' from a terminal outside herdr, then run start again (start has no --replace)
[exit 1]

$ ralph org director attach --driver claude --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
org: director: attach: a director is already attached: d2 (driver claude, attached 2026-10-15T08:54:45Z, headless running): stop it with ralph org director stop --id d2 --state-dir '$SCRATCH/ds/state' from a terminal outside herdr, which closes its herdr pane, or replace it with ralph org director attach --driver claude --replace --state-dir '$SCRATCH/ds/state', which leaves that pane open until such a stop
[exit 1]

[note: HERDR_PANE_ID=w9:p9]
$ ralph org director start --driver claude --model haiku --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
org: director: start: HERDR_PANE_ID is set, so this runs inside a herdr pane: run ralph org director start from a terminal outside herdr
[exit 1]

[note: HERDR_PANE_ID=w1:p2]
$ ralph org director stop --id d2 --state-dir $SCRATCH/ds/state
org: director: stop d2: HERDR_PANE_ID is its pane "w1:p2", and closing that pane would end this command before it records the stop: run ralph org director stop from a terminal outside the director's pane
[exit 1]

[note: HERDR_WORKSPACE_ID=w1]
$ ralph org director stop --id d2 --state-dir $SCRATCH/ds/state
org: director: stop d2: HERDR_WORKSPACE_ID is its workspace "w1", labelled ralph_director_82130355444e, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
[exit 1]

[note: HERDR_WORKSPACE_ID=w1]
$ ralph org director stop --id d2 --force --state-dir $SCRATCH/ds/state
org: director: stop d2: HERDR_WORKSPACE_ID is its workspace "w1", labelled ralph_director_82130355444e, and closing that workspace would end this command before it records the stop: run ralph org director stop from a terminal outside the director's workspace
[exit 1]

$ ralph org director stop --id d1 --state-dir $SCRATCH/ds/state
org: director: stop d1: no open headless registration has the id: the headless registration was already stopped: d1 (driver codex, attached 2026-10-15T08:51:27Z, headless stopped)
[exit 1]

$ ralph org director stop --state-dir $SCRATCH/ds/state
org: director stop: give --id <id>, the id start printed (ralph org director status lists the open headless directors)
[exit 1]

$ ralph org spawn --org-id probe --id x --role director --driver claude --cwd $SCRATCH/ds/cwd --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
rejected: org: role "director" is not a seat role: the director is not an org seat; launch a headless director with ralph org director start, or promote an interactive session with ralph org director attach
org: role "director" is not a seat role: the director is not an org seat; launch a headless director with ralph org director start, or promote an interactive session with ralph org director attach
[exit 1]
director.jsonl md5 after: ed929ae3f9b802709b02897e26b8c9e8
director.jsonl
director.lock
prompts
watch-alive.json
watch.lock
watch.log
```

[note: an earlier spawn probe with the wrong flag `--seat-id` failed with `unknown flag: --seat-id` and one without `--cwd` with `org: --cwd is required`; both wrote nothing. The workspace refusal ran the label checks (herdr gets) and sent nothing: d2's agent was still listed afterwards (L8).]

### L8. stop --all --dry-run names the open headless director

```text
$ ralph org stop --all --dry-run --state-dir $SCRATCH/ds/state
no active seats
note: stop --all stops org seats only, so the headless director d2 (running) is still open: stop it with ralph org director stop --id d2 --state-dir '$SCRATCH/ds/state'
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[{"agent":"claude","agent_status":"idle","cwd":"$SCRATCH/ds/cwd","focused":false,"foreground_cwd":"$SCRATCH/ds/cwd","interactive_ready":true,"name":"director_82130355444e","pane_id":"w1:p2","revision":0,"state_change_seq":1,"tab_id":"w1:t2","terminal_id":"term_65ddd3303751a2","workspace_id":"w1"}],"type":"agent_list"}}
[exit 0]
```

### L9. Another ledger whose recorded ids herdr has given to d2 (AC5, AC14)

`$SCRATCH/ds/state2/director.jsonl` was written by hand: an open headless `d1` with token `aaaaaaaaaaaa`, workspace `w1`, pane `w1:p2` (the ids herdr now gives to the first ledger's d2), and `launched`.

```text
$ ralph org director status --state-dir $SCRATCH/ds/state2 --config $SCRATCH/ds/ralph.toml
director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane w1:p2, workspace w1, attached at 2026-10-15T08:00:00Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
[note: the field lines that follow (enabled to pane_id) are elided]
[exit 0]
B md5 before: 6b30cc6b5ef2fc56e77e79ab52ec2696

$ ralph org director stop --id d1 --state-dir $SCRATCH/ds/state2
stop d1: ctrl_c=skipped: not confirmed, pane=not closed: not confirmed, workspace=not closed: not confirmed
org: director: stop d1: pane "w1:p2" is not confirmed as d1's (a tab labelled director_aaaaaaaaaaaa in a workspace labelled ralph_director_aaaaaaaaaaaa), so nothing was sent to it: pane "w1:p2" is not in a tab labelled "director_aaaaaaaaaaaa": herdr has it in tab "w1:t2" labelled "director_82130355444e"; nothing was recorded; close it by hand: herdr workspace list, then herdr workspace close <the id of the workspace labelled ralph_director_aaaaaaaaaaaa>; afterwards record the stop with ralph org director stop --id d1 --force --state-dir '$SCRATCH/ds/state2'
[exit 1]
B md5 after the refused stop: 6b30cc6b5ef2fc56e77e79ab52ec2696

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[{"agent":"claude","agent_status":"idle","cwd":"$SCRATCH/ds/cwd","focused":false,"foreground_cwd":"$SCRATCH/ds/cwd","interactive_ready":true,"name":"director_82130355444e","pane_id":"w1:p2","revision":0,"state_change_seq":1,"tab_id":"w1:t2","terminal_id":"term_65ddd3303751a2","workspace_id":"w1"}],"type":"agent_list"}}
[exit 0]

$ ralph org director stop --id d1 --force --state-dir $SCRATCH/ds/state2
stopped d1 (forced; ctrl_c=skipped: not confirmed, pane=not closed: not confirmed, workspace=not closed: not confirmed)
the director is no longer attached
warning: director stop --force: d1 is recorded stopped, but its herdr pane or workspace may still be open (pane "w1:p2" is not confirmed as d1's (a tab labelled director_aaaaaaaaaaaa in a workspace labelled ralph_director_aaaaaaaaaaaa), so nothing was sent to it: pane "w1:p2" is not in a tab labelled "director_aaaaaaaaaaaa": herdr has it in tab "w1:t2" labelled "director_82130355444e"); find and close them by hand: herdr workspace list, then herdr workspace close <the id of the workspace labelled ralph_director_aaaaaaaaaaaa>
[exit 0]
## B after --force (last two lines)
{"ts":"2026-10-15T08:55:48Z","id":"d1","event":"detached","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:55:48Z","id":"d1","event":"stopped","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false,"force":true}

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[{"agent":"claude","agent_status":"idle","cwd":"$SCRATCH/ds/cwd","focused":false,"foreground_cwd":"$SCRATCH/ds/cwd","interactive_ready":true,"name":"director_82130355444e","pane_id":"w1:p2","revision":0,"state_change_seq":1,"tab_id":"w1:t2","terminal_id":"term_65ddd3303751a2","workspace_id":"w1"}],"type":"agent_list"}}
[exit 0]

$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[{"active_tab_id":"w1:t1","agent_status":"idle","focused":true,"label":"ralph_director_82130355444e","number":1,"pane_count":2,"tab_count":2,"workspace_id":"w1"}]}}
[exit 0]
A md5 (unchanged expected ed929ae3f9b802709b02897e26b8c9e8): ed929ae3f9b802709b02897e26b8c9e8
```

### L10. director stop --id d2

```text
$ ralph org director stop --id d2 --state-dir $SCRATCH/ds/state
stopped d2 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[]}}
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}
[exit 0]
## director.jsonl d2 lines after stop (last two)
{"ts":"2026-10-15T08:55:58Z","id":"d2","event":"detached","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:55:58Z","id":"d2","event":"stopped","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}

$ ralph status --state-dir $SCRATCH/ds/state
no org runtime state found (state-dir=$SCRATCH/ds/state (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 44058, mark updated 2026-10-15T08:55:57Z)
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
merge guarantee: unknown (not checked: run ralph doctor)
Run `ralph doctor` to check environment readiness.
[exit 0]

$ ralph org stop --all --dry-run --state-dir $SCRATCH/ds/state
no active seats
[exit 0]
```

### L11. A start whose agent start fails (AC3)

```text
$ ralph org director start --driver claude --model haiku --timeout-ms 200 --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
org: director: start d3: herdr agent start director_c7805b15e99b (agent_start_retries=0): herdr: exit status 1: {"error":{"code":"agent_pane_busy","message":"agent target pane w2:p2 is not an available shell"},"id":"cli:agent:start"} (herdr: agent_pane_busy: agent target pane w2:p2 is not an available shell); start closed what it had made in herdr, and d3 is recorded stopped
[exit 1]

$ herdr workspace list
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[]}}
[exit 0]

$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}
[exit 0]
## director.jsonl d3 lines
{"ts":"2026-10-15T08:56:09Z","id":"d3","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"c7805b15e99b","model":"haiku","permission_mode":"guarded","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:56:09Z","id":"d3","event":"workspace_created","workspace_id":"w2","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:56:09Z","id":"d3","event":"pane_created","pane_id":"w2:p2","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:56:09Z","id":"d3","event":"detached","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}
{"ts":"2026-10-15T08:56:09Z","id":"d3","event":"stopped","user":"$USER","cwd":"$SCRATCH/ds/cwd","herdr_pane":false}

$ ralph org director status --state-dir $SCRATCH/ds/state --config $SCRATCH/ds/ralph.toml
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
enabled: true
[exit 0]
```

The pane was not at its shell prompt yet 200 ms after the tab was made; the 200 ms context left no time for an `agent_pane_busy` retry.

### Cleanup

```text
## C1. the watch start launched
watch.log:
watching every org (interval=30s once=false state-dir=$SCRATCH/ds/state (source: flag))

$ ralph org watch --stop --state-dir $SCRATCH/ds/state
stopped ralph org watch --all (pid 44058)
[exit 0]
pid 44058 gone

## C2. the dedicated herdr server
{"id":"cli:workspace:list","result":{"type":"workspace_list","workspaces":[]}}
[exit 0]
[note: HERDR_SOCKET_PATH=/tmp/rdst/h.sock herdr server stop]
[exit 0]
pid 77399 gone
pgrep -fl 'herdr server' after:
[pgrep exit 1]
"/tmp/rdst": No such file or directory (os error 2)
~/.config/herdr/session.json sha256 after:
004173699fb3c9320508a7442ed9d9f439d853d20a0732ec4cf8d0b4eb7811db  ~/.config/herdr/session.json
.rw-r--r--@ 379k $USER  1 Oct 14:35 ~/.config/herdr/herdr-server.log
```

- `pgrep -fl 'scratchpad/ds'` and `pgrep -fl 'director_[0-9a-f]{12}'` printed nothing: no agent, watch, or ralph process of this run is left.
- The repo's ledger was not written: neither `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` nor the worktree's `.harness/state/org` exists.
- Left outside the scratch directory by the agents: the codex session record named in L3, and `~/.codex/config.toml`, whose modification time changed to 17:51 JST, the time the codex director started (it was 14 Oct 20:53 before). It holds no scratch path; what codex changed was not captured (no copy was taken before the run; likely the `[tui.model_availability_nux]` counter, which reads `gpt-6-astra = 4`, unconfirmed). It was not restored. claude ran with `CLAUDE_CONFIG_DIR` in `/tmp/rdst`: `~/.claude.json` has no project entry for the scratch cwd and `~/.claude/projects/` no directory for it.
- The scratch ledgers and the HEAD binary stay in the session scratch directory.

## Cycle 2 (HEAD a539a765)

- Date: 2026-10-15 (UTC timestamps in the output read 2026-10-15T12:xx; JST is UTC+9)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD a539a765. origin/main f21df66c is merged in at 1cdddbbb, so the branch diff is `git diff origin/main...HEAD` (merge base f21df66c)
- Redaction: as above. The probe directory `$SCRATCH/c2probe` is `$PROBE`. The probe ran every ralph command with `USER=probe`, so `probe` in the output is that value, not a login name. The full wrapper log is `docs/evidence/verify-2026-10-15-122004.log`, and the raw outputs of this section are appended to `docs/evidence/verify-2026-10-15-org-director-start.log` under `===== Cycle 2 (HEAD a539a765)` (both gitignored, not committed).

### C2-1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-15T12:20:04Z
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
[note: the settings.json dump of jq -e . is elided]
    OK
==> jq -e . templates/base/.claude/settings.json
[note: the settings.json dump of jq -e . is elided]
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
==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned f21df66c..a539a765 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-15-122004.log
rc=0
```

### C2-2. Go and sync checks

```text
$ gofmt -l ./cmd ./internal
rc=0
$ go vet ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
$ $(GOTOOLCHAIN=go1.25.8 go env GOROOT)/bin/gofmt -l ./cmd ./internal
[note: GOROOT is ~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.8.darwin-arm64]
rc=0
$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
rc=0
$ ./scripts/check-sync.sh | tail -6
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
$ git diff --check origin/main...HEAD
rc=0
$ git diff --check 54cab8a2..HEAD
rc=0
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-org-director-start.md
f65d7c59746c
$ md5 -q .claude/skills/director/SKILL.md .agents/skills/director/SKILL.md templates/base/.claude/skills/director/SKILL.md templates/base/.agents/skills/director/SKILL.md
dbe32a18b8b46c47b44657c4e2443cfe
dbe32a18b8b46c47b44657c4e2443cfe
dbe32a18b8b46c47b44657c4e2443cfe
dbe32a18b8b46c47b44657c4e2443cfe
$ git diff origin/main...HEAD -- '*.go' | grep '^+' | grep -i 'finding [0-9]\|cross-review cycle\|/test gap\|mutant [A-Z][0-9]\|gap [0-9]'
[grep exit 1]
$ grep -rn 'every ralph org command' internal/ | grep -v _test
[grep exit 1]
```

go1.26.0 is the local toolchain; go.mod says `go 1.25.8`. Every `-l` printed no file name. The two greps are the residue checks of the self-review's C2-L4 (report numbers in Go comments) and C2-L1 (the "every ralph org command" wording outside the tests).

### C2-3. HEAD binary probe: the prompt file with and without a relative --config

Setup:

- HEAD binary: `go build -o $PROBE/bin/ralph ./cmd/ralph` from the clean worktree at a539a765. `$PROBE/bin` holds only that binary, a symlink to git, and a stub `herdr` (a POSIX sh script that logs every call to `$PROBE/hstate/calls.log`, keeps the workspace and tab labels in files there, and answers in the JSON envelope shapes of `internal/org/driver/herdr.go`: `workspace create` gives `w<N>`, `tab create` gives `w<N>:p1` in tab `w<N>:t1`, `agent start` and `send-keys` answer ok, the gets answer the stored labels, and `pane close` and `workspace close` delete them; an unknown id answers a `*_not_found` envelope with exit 1). The stub never talks to a herdr server.
- Every ralph command ran under `env -i HOME=$PROBE/home USER=probe LOGNAME=probe PATH=$PROBE/bin TMPDIR=$PROBE/tmp/ LANG=en_US.UTF-8`, so no osascript, real herdr, agmsg, claude, or codex could be found, `HERDR_PANE_ID` and `HERDR_WORKSPACE_ID` were unset, and nothing read the user's home. The cwd was `$PROBE/work`, which is not in a git repo, so the ledger is not a main worktree's and the director runs where start ran.
- `$PROBE/work/cfg/ralph.toml` (the relative `--config cfg/ralph.toml`) and `$PROBE/work/ralph.toml` (read without `--config`) differ only in `[org.permissions.roles].director`, so the `permission=` field shows which file start read:

```toml
[org]
driver_pool = ["claude"]
model_pool = [
  { driver = "claude", model = "haiku" },
]

[org.permissions]
default = "autonomous"

[org.permissions.roles]
director = "guarded"   # $PROBE/work/ralph.toml: "edits"

[org.director]
enabled = true
```

- Before: `ls -A $PROBE/state` printed nothing (an empty scratch ledger). `pgrep -fl herdr` printed nothing (exit 1).
- The probe ran the prompt's first command and its stop command by taking the lines as written from the prompt file (`grep -m1 '^ralph org director status'` and the indented `ralph org director stop` line) and running them with `/bin/sh -c` in the prompt's `- cwd:` directory, as a pane shell would.

#### R1. director start --driver claude --model haiku with --config cfg/ralph.toml

````text
$ (cd $PROBE/work && ralph org director start --driver claude --model haiku --state-dir $PROBE/state --config cfg/ralph.toml)
ralph org director start: started the cross-org watch (pid 73368); log: $PROBE/state/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w1:p1, workspace=w1)
talk to the director in herdr: pane w1:p1, in the tab director_53b0807f6b30 of the workspace labelled ralph_director_53b0807f6b30
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state'
[exit 0]

$ ls $PROBE/state/prompts
director_53b0807f6b30.md

$ cat $PROBE/state/prompts/director_53b0807f6b30.md
# 役割: director(headless)

- id: d1 / driver: claude
- 台帳: '$PROBE/state'
- 設定: '$PROBE/work/cfg/ralph.toml'(start の --config)
- cwd: $PROBE/work

## 最初の手順

ほかの何よりも先に、次のコマンドを打つ。

```sh
ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'
```

今の登録の `id` が `d1` であることを確かめてから動く。`id` が `d1` と
違うとき、または登録がなく `not attached` と出たときは、ほかに何もしない
(受信箱の ack と resolve、leader への send、start をしない)。そのことを
この pane に書いて止まる。detach も打たない。

## ミッション

あなたは `ralph org director start` が立てた headless の director です。
登録(台帳の `director.jsonl` の `d1`)は start が済ませています。人と
機能ごとの org の間に立ち、手順の全体は `/director` skill に従ってください。

- skill の「昇格」の手順のうち、見張りの確かめ(`ralph status` の
  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
  済ませたので、打たない。
- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` で
  今の登録を読み直す手順は守る。`id` が `d1` でなくなっていたら、
  director として動くのをやめ、そのことをこの pane に書く。
- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` を付ける。
  `ralph status` には `--state-dir '$PROBE/state'` だけを付ける。cwd が
  どこでも、この台帳と、start が読んだ設定を使うため。終わりに人に頼む
  stop は ralph.toml を読まないので、`--state-dir '$PROBE/state'` だけを持つ。
- 人とはこの pane で話す。人は herdr でこの pane に入って話しかける。
- 実装しない。director がしてよいのは、分割計画を書くこと、repo を読む
  こと、ralph と gh の操作だけ。コードと文書の変更は、機能の org か標準
  フロー(`/plan` → `/implement`)に任せる。

## 終わり

- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
  detach は登録を外すだけで、この pane と workspace は開いたまま残る。
- 終えるときは、open か acked の件が残っていれば人に伝え、人に、herdr の外の
  端末から次のコマンドを打つよう頼む。stop がこの pane と workspace を閉じ、
  止めたことを台帳に記録する。

  ```sh
  ralph org director stop --id d1 --state-dir '$PROBE/state'
  ```

- stop は自分で打たない。この pane の中からは断られる(pane を閉じると、
  打ったセッションが終わって記録を書けないため)。
[end of $PROBE/state/prompts/director_53b0807f6b30.md]

$ grep -n 'ralph org' $PROBE/state/prompts/director_53b0807f6b30.md
13:ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'
23:あなたは `ralph org director start` が立てた headless の director です。
28:  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
30:- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` で
33:- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` を付ける。
44:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
51:  ralph org director stop --id d1 --state-dir '$PROBE/state'

[note: the prompt's first command, run as written from its cwd]
$ (cd $PROBE/work && ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml')
director: attached d1 (driver=claude, headless running, model=haiku, pane w1:p1, workspace w1, attached at 2026-10-15T12:23:48Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
driver: claude
wake: pull
attached_at: 2026-10-15T12:23:48Z
user: probe
cwd: $PROBE/work
herdr_pane: no
launch: herdr
state: running
model: haiku
permission_mode: guarded
token: 53b0807f6b30
workspace_id: w1
pane_id: w1:p1
[exit 0]

[note: the prompt's stop command, run as written]
$ (cd $PROBE/work && ralph org director stop --id d1 --state-dir '$PROBE/state')
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]
````

The relative `cfg/ralph.toml` reached the prompt as `'$PROBE/work/cfg/ralph.toml'`, absolute and one shell word. The two status commands the director runs carry `--state-dir` then `--config`; the stop line (line 51) and the `ralph status` instruction carry `--state-dir` only. `permission=guarded` is the `--config` file's value.

#### R2. director start --driver claude --model haiku without --config

````text
$ (cd $PROBE/work && ralph org director start --driver claude --model haiku --state-dir $PROBE/state)
started d2 (driver=claude, model=haiku, permission=edits, pane=w2:p1, workspace=w2)
talk to the director in herdr: pane w2:p1, in the tab director_dbac7302f192 of the workspace labelled ralph_director_dbac7302f192
stop it with: ralph org director stop --id d2 --state-dir '$PROBE/state'
[exit 0]

$ ls $PROBE/state/prompts
director_53b0807f6b30.md
director_dbac7302f192.md

$ cat $PROBE/state/prompts/director_dbac7302f192.md
# 役割: director(headless)

- id: d2 / driver: claude
- 台帳: '$PROBE/state'
- 設定: main worktree の ralph.toml(台帳が main worktree のものでなければ、cwd の ralph.toml)
- cwd: $PROBE/work

## 最初の手順

ほかの何よりも先に、次のコマンドを打つ。

```sh
ralph org director status --state-dir '$PROBE/state'
```

今の登録の `id` が `d2` であることを確かめてから動く。`id` が `d2` と
違うとき、または登録がなく `not attached` と出たときは、ほかに何もしない
(受信箱の ack と resolve、leader への send、start をしない)。そのことを
この pane に書いて止まる。detach も打たない。

## ミッション

あなたは `ralph org director start` が立てた headless の director です。
登録(台帳の `director.jsonl` の `d2`)は start が済ませています。人と
機能ごとの org の間に立ち、手順の全体は `/director` skill に従ってください。

- skill の「昇格」の手順のうち、見張りの確かめ(`ralph status` の
  `org watch:` の行と `ralph org watch --all --detach`)と attach は start が
  済ませたので、打たない。
- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state'` で
  今の登録を読み直す手順は守る。`id` が `d2` でなくなっていたら、
  director として動くのをやめ、そのことをこの pane に書く。
- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state'` を付ける。
  `ralph status` には `--state-dir '$PROBE/state'` だけを付ける。cwd が
  どこでも、この台帳と、start が読んだ設定を使うため。終わりに人に頼む
  stop は ralph.toml を読まないので、`--state-dir '$PROBE/state'` だけを持つ。
- 人とはこの pane で話す。人は herdr でこの pane に入って話しかける。
- 実装しない。director がしてよいのは、分割計画を書くこと、repo を読む
  こと、ralph と gh の操作だけ。コードと文書の変更は、機能の org か標準
  フロー(`/plan` → `/implement`)に任せる。

## 終わり

- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
  detach は登録を外すだけで、この pane と workspace は開いたまま残る。
- 終えるときは、open か acked の件が残っていれば人に伝え、人に、herdr の外の
  端末から次のコマンドを打つよう頼む。stop がこの pane と workspace を閉じ、
  止めたことを台帳に記録する。

  ```sh
  ralph org director stop --id d2 --state-dir '$PROBE/state'
  ```

- stop は自分で打たない。この pane の中からは断られる(pane を閉じると、
  打ったセッションが終わって記録を書けないため)。
[end of $PROBE/state/prompts/director_dbac7302f192.md]

$ grep -c -- --config $PROBE/state/prompts/director_dbac7302f192.md
0
[exit 1]

[note: the prompt's first command, run as written from its cwd]
$ (cd $PROBE/work && ralph org director status --state-dir '$PROBE/state')
director: attached d2 (driver=claude, headless running, model=haiku, pane w2:p1, workspace w2, attached at 2026-10-15T12:23:49Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d2
driver: claude
wake: pull
attached_at: 2026-10-15T12:23:49Z
user: probe
cwd: $PROBE/work
herdr_pane: no
launch: herdr
state: running
model: haiku
permission_mode: edits
token: dbac7302f192
workspace_id: w2
pane_id: w2:p1
[exit 0]

[note: the prompt's stop command, run as written]
$ (cd $PROBE/work && ralph org director stop --id d2 --state-dir '$PROBE/state')
stopped d2 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ diff <R1 prompt> <R2 prompt>
3c3
< - id: d1 / driver: claude
---
> - id: d2 / driver: claude
5c5
< - 設定: '$PROBE/work/cfg/ralph.toml'(start の --config)
---
> - 設定: main worktree の ralph.toml(台帳が main worktree のものでなければ、cwd の ralph.toml)
13c13
< ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'
---
> ralph org director status --state-dir '$PROBE/state'
16c16
< 今の登録の `id` が `d1` であることを確かめてから動く。`id` が `d1` と
---
> 今の登録の `id` が `d2` であることを確かめてから動く。`id` が `d2` と
24c24
< 登録(台帳の `director.jsonl` の `d1`)は start が済ませています。人と
---
> 登録(台帳の `director.jsonl` の `d2`)は start が済ませています。人と
30,31c30,31
< - 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` で
<   今の登録を読み直す手順は守る。`id` が `d1` でなくなっていたら、
---
> - 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state'` で
>   今の登録を読み直す手順は守る。`id` が `d2` でなくなっていたら、
33c33
< - あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state' --config '$PROBE/work/cfg/ralph.toml'` を付ける。
---
> - あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state'` を付ける。
51c51
<   ralph org director stop --id d1 --state-dir '$PROBE/state'
---
>   ralph org director stop --id d2 --state-dir '$PROBE/state'
[exit 1]
````

Without `--config` the prompt says `--config` nowhere (`grep -c` 0), and `permission=edits` is `./ralph.toml`'s value. The two prompts differ only in the id, the 設定 line, and the three lines that carry `--config` in R1; the stop line differs by the id alone.

#### director.jsonl and the stub herdr calls of R1 and R2

```text
$ cat $PROBE/state/director.jsonl
{"ts":"2026-10-15T12:23:48Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"53b0807f6b30","model":"haiku","permission_mode":"guarded","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d1","event":"workspace_created","workspace_id":"w1","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d1","event":"pane_created","pane_id":"w1:p1","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d1","event":"launched","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d2","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"dbac7302f192","model":"haiku","permission_mode":"edits","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d2","event":"workspace_created","workspace_id":"w2","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d2","event":"pane_created","pane_id":"w2:p1","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:49Z","id":"d2","event":"launched","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:50Z","id":"d2","event":"detached","user":"probe","cwd":"$PROBE/work","herdr_pane":false}
{"ts":"2026-10-15T12:23:50Z","id":"d2","event":"stopped","user":"probe","cwd":"$PROBE/work","herdr_pane":false}

$ cat $PROBE/hstate/calls.log
herdr [workspace] [create] [--cwd] [$PROBE/work] [--label] [ralph_director_53b0807f6b30]
herdr [tab] [create] [--workspace] [w1] [--cwd] [$PROBE/work] [--label] [director_53b0807f6b30]
herdr [agent] [start] [director_53b0807f6b30] [--kind] [claude] [--pane] [w1:p1] [--timeout] [60000] [--] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state/prompts/director_53b0807f6b30.md]
herdr [pane] [get] [w1:p1]
herdr [tab] [get] [w1:t1]
herdr [workspace] [get] [w1]
herdr [workspace] [get] [w1]
herdr [pane] [send-keys] [w1:p1] [C-c]
herdr [pane] [close] [w1:p1]
herdr [workspace] [close] [w1]
herdr [workspace] [create] [--cwd] [$PROBE/work] [--label] [ralph_director_dbac7302f192]
herdr [tab] [create] [--workspace] [w2] [--cwd] [$PROBE/work] [--label] [director_dbac7302f192]
herdr [agent] [start] [director_dbac7302f192] [--kind] [claude] [--pane] [w2:p1] [--timeout] [60000] [--] [--permission-mode] [acceptEdits] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state/prompts/director_dbac7302f192.md]
herdr [pane] [get] [w2:p1]
herdr [tab] [get] [w2:t1]
herdr [workspace] [get] [w2]
herdr [workspace] [get] [w2]
herdr [pane] [send-keys] [w2:p1] [C-c]
herdr [pane] [close] [w2:p1]
herdr [workspace] [close] [w2]

$ (cd $PROBE/work && ralph org watch --stop --state-dir $PROBE/state)
stopped ralph org watch --all (pid 73368)
[exit 0]
```

The guarded director got no permission flag and the edits one `--permission-mode acceptEdits`, both before `--model` and the pointer sentence. Neither ledger has a `manifest.jsonl` or `model-receipts.jsonl`.

### C2-4. HEAD vs pre-fix binary: the cross-review cycle-1 scenario (R3)

The pre-fix binary was built from `git archive 54cab8a2` (the commit before the fix 3243c499) into `$PROBE/oldbin`, with the same git symlink and stub herdr. `$PROBE/work/ralph.toml` was changed to `[org.director].enabled = false` (the rest as in C2-3); `cfg/ralph.toml` still says true. Each binary ran on its own empty ledger.

```text
## R3 (HEAD a539a765)
$ (cd $PROBE/work && ralph org director start --driver claude --model haiku --state-dir $PROBE/state3head --config cfg/ralph.toml)
ralph org director start: started the cross-org watch (pid 77080); log: $PROBE/state3head/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w3:p1, workspace=w3)
talk to the director in herdr: pane w3:p1, in the tab director_0abc3506eb3f of the workspace labelled ralph_director_0abc3506eb3f
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state3head'
[exit 0]

[note: lines 1-6 and the ralph org director lines of the prompt file]
# 役割: director(headless)

- id: d1 / driver: claude
- 台帳: '$PROBE/state3head'
- 設定: '$PROBE/work/cfg/ralph.toml'(start の --config)
- cwd: $PROBE/work
13:ralph org director status --state-dir '$PROBE/state3head' --config '$PROBE/work/cfg/ralph.toml'
23:あなたは `ralph org director start` が立てた headless の director です。
30:- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state3head' --config '$PROBE/work/cfg/ralph.toml'` で
44:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
51:  ralph org director stop --id d1 --state-dir '$PROBE/state3head'

[note: the prompt's first command, run as written from its cwd]
$ (cd $PROBE/work && ralph org director status --state-dir '$PROBE/state3head' --config '$PROBE/work/cfg/ralph.toml')
director: attached d1 (driver=claude, headless running, model=haiku, pane w3:p1, workspace w3, attached at 2026-10-15T12:24:38Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
[note: the registration fields that follow are elided; they match R1's with token 0abc3506eb3f, w3, w3:p1]
[exit 0]

$ (cd $PROBE/work && ralph org director stop --id d1 --state-dir $PROBE/state3head)
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ (cd $PROBE/work && ralph org watch --stop --state-dir $PROBE/state3head)
stopped ralph org watch --all (pid 77080)
[exit 0]

## R3 (pre-fix 54cab8a2)
$ (cd $PROBE/work && ralph org director start --driver claude --model haiku --state-dir $PROBE/state3old --config cfg/ralph.toml)
ralph org director start: started the cross-org watch (pid 77237); log: $PROBE/state3old/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w4:p1, workspace=w4)
talk to the director in herdr: pane w4:p1, in the tab director_724478a91769 of the workspace labelled ralph_director_724478a91769
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state3old'
[exit 0]

[note: lines 1-6 and the ralph org director lines of the prompt file]
# 役割: director(headless)

- id: d1 / driver: claude
- 台帳: '$PROBE/state3old'
- cwd: $PROBE/work

12:ralph org director status --state-dir '$PROBE/state3old'
22:あなたは `ralph org director start` が立てた headless の director です。
29:- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state3old'` で
42:- skill の「終わり」の手順のうち、`ralph org director detach` は打たない。
49:  ralph org director stop --id d1 --state-dir '$PROBE/state3old'

[note: the prompt's first command, run as written from its cwd]
$ (cd $PROBE/work && ralph org director status --state-dir '$PROBE/state3old')
director: disabled ([org.director].enabled = false; registration d1 is still recorded, headless running, pane w4:p1, workspace w4: stop it with ralph org director stop --id d1 --state-dir '$PROBE/state3old') -- ralph org escalate sends every item to the human at once
enabled: false
[note: the registration fields that follow are elided; token 724478a91769, w4, w4:p1]
[exit 0]

$ (cd $PROBE/work && ralph org director stop --id d1 --state-dir $PROBE/state3old)
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ (cd $PROBE/work && ralph org watch --stop --state-dir $PROBE/state3old)
stopped ralph org watch --all (pid 77237)
[exit 0]
```

With the pre-fix binary the start succeeded on the `--config` file, but the director's first command read `./ralph.toml` and answered `director: disabled`, which the prompt's first step treats as "not my registration" (stop and do nothing). With HEAD the same first command reads the `--config` file and answers `attached d1`, `enabled: true`. This is the cross-review cycle-1 finding reproduced on 54cab8a2 and gone on a539a765.

### C2-5. ralph status beside a director started with --config (R4)

Same files as C2-4 (`./ralph.toml` enabled = false, `cfg/ralph.toml` true), HEAD binary, ledger `$PROBE/state4`.

```text
$ (cd $PROBE/work && ralph org director start --driver claude --model haiku --state-dir $PROBE/state4 --config cfg/ralph.toml)
ralph org director start: started the cross-org watch (pid 80053); log: $PROBE/state4/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w5:p1, workspace=w5)
talk to the director in herdr: pane w5:p1, in the tab director_a2af88c354b0 of the workspace labelled ralph_director_a2af88c354b0
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state4'
[exit 0]

$ (cd $PROBE/work && ralph status --state-dir '$PROBE/state4' | /usr/bin/grep -E '^(director|org watch):')
org watch: running (pid 80053, mark updated 2026-10-15T12:25:20Z)
director: disabled ([org.director].enabled = false; registration d1 is still recorded, headless running, pane w5:p1, workspace w5: stop it with ralph org director stop --id d1 --state-dir '$PROBE/state4') -- ralph org escalate sends every item to the human at once
[exit 0]

$ (cd $PROBE/work && ralph org director status --state-dir '$PROBE/state4' --config '$PROBE/work/cfg/ralph.toml' | /usr/bin/sed -n 1,2p)
director: attached d1 (driver=claude, headless running, model=haiku, pane w5:p1, workspace w5, attached at 2026-10-15T12:25:20Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
[exit 0]

$ (cd $PROBE/work && ralph org director stop --id d1 --state-dir $PROBE/state4)
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]

$ (cd $PROBE/work && ralph org watch --stop --state-dir $PROBE/state4)
stopped ralph org watch --all (pid 80053)
[exit 0]
```

`ralph status` has no `--config` (`internal/cli/status.go` flags: `--state-dir`, `--org-id`, `--json`), so its `director:` line follows `./ralph.toml` while the director's own status follows the `--config` file. A leader's `ralph org escalate` without `--config` reads the repo's file too, so the `ralph status` line is the one that describes how escalate delivers here.

### C2-6. ralph org director start --help of the HEAD binary (the --config sentence)

```text
$ ralph org director start --help | sed -n '/^The director runs with/,/^$/p'
The director runs with the permission mode of
[org.permissions.roles].director, else [org.permissions].default (for
codex, autonomous and edits need [org.permissions].codex_verified), and
with --model, which [org.roles].director must allow; without --model,
with the first model_pool entry for --driver that it allows, and a
warning. Without --config, those settings and [org.director].enabled
are read from the main worktree's ralph.toml when the ledger is the main
worktree's, so a feature worktree's ralph.toml does not choose them. The
director runs in the ledger's main worktree root, else in the directory
start runs in. With --config, the ralph org commands the director's
prompt has it run carry --config with that file's absolute path, so the
director reads the settings start read (the prompt's stop command, for
the human, carries none: stop reads no ralph.toml).
```

### C2-7. Cleanup

- Every watch the starts launched was stopped with `ralph org watch --stop` (pids 73368, 77080, 77237, 80053). `pgrep -fl "$PROBE/"` afterwards printed nothing (exit 1).
- `$PROBE/hstate` holds only `calls.log` and the workspace counter: every stub workspace, tab, and pane the starts made was closed by a stop.
- `pgrep -fl 'herdr server'` and `pgrep -fl osascript` printed nothing (exit 1). No real herdr ran, and the user's herdr server and `~/.config/herdr` were not touched.
- The repo's ledger was not written: neither `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` nor the worktree's `.harness/state/org` exists. `$PROBE/home` and `$PROBE/tmp` are empty.
- The scratch ledgers, the two binaries, the stub, and the probe scripts stay in the session scratch directory.

## Cycle 3 (HEAD c5a6a596)

- Date: 2026-10-16 JST (UTC timestamps in the output read 2026-10-15T15:xx; JST is UTC+9)
- Worktree: ~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-start, HEAD c5a6a596, clean (`git status --porcelain` printed nothing). Merge base with origin/main is f21df66c. The delta since the cycle-2 verify (d3bf2066) holds the cross-review cycle-2 fixes D12 and D13 (plan 4b721b18, code caf1d025) and the cycle-3 self-review LOW fixes (plan 0bfbb054, code 83287d39)
- Redaction: as above. The probe directory `$SCRATCH/c3p` is `$PROBE`. The probe ran every ralph command with `USER=probe`. The full wrapper log is `docs/evidence/verify-2026-10-15-152534.log`, and the raw outputs of this section are appended to `docs/evidence/verify-2026-10-15-org-director-start.log` under `===== Cycle 3 (HEAD c5a6a596)` (both gitignored, not committed)

### C3-1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-15T15:25:34Z
- Mode: static
- Requested scope: changed

==> Running local verifier
==> shellcheck hook + verify scripts
    OK
[note: the 20 `sh -n` steps of .claude/hooks and templates/base/.claude/hooks and the two awk parses of the guard each printed OK; omitted here, in the wrapper log]
==> jq -e . .claude/settings.json
    [the JSON body jq printed is omitted here; it is in the wrapper log]
    OK
==> jq -e . templates/base/.claude/settings.json
    [the JSON body jq printed is omitted here; it is in the wrapper log]
    OK
==> Codex hook single-source guard
    OK
==> Codex inline hook detector smoke test
    OK
==> Codex PR provenance policy guard
    OK
==> scripts/check-sync.sh
[note: the TEMPLATE_ONLY and KNOWN_DIFF lines are omitted; the summary follows]
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
==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned f21df66c..c5a6a596 against origin/main: clean

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-10-15-152534.log
rc=0
```

### C3-2. Go and sync checks

```text
$ go version
go version go1.26.0 darwin/arm64
$ gofmt -l internal/ cmd/
rc=0
$ go vet ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...
rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...
rc=0
$ $(GOTOOLCHAIN=go1.25.8 go env GOROOT)/bin/gofmt -l ./cmd ./internal
rc=0
$ grep '^go ' go.mod
go 1.25.8
$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
rc=0
$ ./scripts/check-sync.sh | tail -8
=== Sync Summary ===
  IDENTICAL:      169
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5

PASS: all files in sync.
rc=0
$ git diff --check origin/main...HEAD
rc=0
$ git diff --check d3bf2066..HEAD
rc=0
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-org-director-start.md
96796452ceae
$ md5 -q {.claude,.agents,templates/base/.claude,templates/base/.agents}/skills/director/SKILL.md
7aaf58f617dcc749d5bb1f8a39c0dc8c
7aaf58f617dcc749d5bb1f8a39c0dc8c
7aaf58f617dcc749d5bb1f8a39c0dc8c
7aaf58f617dcc749d5bb1f8a39c0dc8c
$ md5 -q {.claude,.agents,templates/base/.claude,templates/base/.agents}/skills/org/SKILL.md
c1bb29c022619e5bf9b4572309ca7119
c1bb29c022619e5bf9b4572309ca7119
c1bb29c022619e5bf9b4572309ca7119
c1bb29c022619e5bf9b4572309ca7119
$ cmp docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md
[exit 0]
```

Every `-l` printed no file name. The plan digest equals the `- Approved:` line (96796452ceae).

### C3-3. Stub herdr probe: the D12 and D13 refusals, HEAD vs the pre-fix binary

Setup:

- HEAD binary: `go build -o $PROBE/bin/ralph ./cmd/ralph` from the clean worktree at c5a6a596. Pre-fix binary: `git archive 4b721b18` (the plan commit right before caf1d025) built into `$PROBE/prebin/ralph`. Each bin directory holds only ralph, a symlink to git, and the same stub `herdr`: a POSIX sh script that logs every call to `$PROBE/hstate/calls.log`, keeps the workspace and tab labels in files there, and answers in the JSON envelope shapes of `internal/org/driver/herdr.go` (as in C2-3). The stub never talks to a herdr server.
- Every ralph command ran under `env -i HOME=$PROBE/home USER=probe LOGNAME=probe PATH=<bin dir> TMPDIR=$PROBE/tmp/ LANG=en_US.UTF-8`, so no real herdr, codex, claude, agmsg, or osascript could be found, and `HERDR_PANE_ID` was unset. Each cwd is under `$PROBE/work`, outside any git repo, so the ledger (an explicit `--state-dir`) is not a main worktree's: the director runs where start runs, and the ledger is outside that cwd.
- After each command the probe printed how many lines the stub added to `calls.log`, and the lines.
- Configs (`ralph.toml` in the cwd, or `cfg/ralph.toml` given as the relative `--config cfg/ralph.toml`):
  - work/a: none (`config.Default()`: `[org.permissions].default = "autonomous"`, `codex_verified = false`)
  - work/b: `[org.permissions] default = "autonomous"`, `codex_verified = true`
  - work/c: `[org.permissions] default = "guarded"`, `codex_verified = true`; `[org.permissions.roles] director = "edits"`
  - work/d/cfg: `[org] agmsg_home = "~/agmsg-probe"`; `[org.permissions] default = "autonomous"`, `codex_verified = true`; `[org.permissions.roles] director = "guarded"`
  - work/e/cfg: `[org] agmsg_home = "agmsg/home"`; `[org.permissions.roles] director = "guarded"`
  - work/f/cfg: `[org] agmsg_home = "~"`
  - work/g (no `--config`): `[org] agmsg_home = "agmsg/home"`; `[org.permissions.roles] director = "guarded"`

```text
## Q1. codex director, no ralph.toml (config.Default: default autonomous, codex_verified false)
$ (cd work/a && ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-a)   [bin=bin]
org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, so the director could not start a feature org (ralph org start --plan runs git worktree add); set [org.permissions.roles].director = "guarded" and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, or start a claude director (--driver claude); a guarded codex director follows that config, so under sandbox_mode = "workspace-write" it meets the same wall
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-a
[no such directory]

## Q2. codex director, ralph.toml: default autonomous, codex_verified = true
$ (cd work/b && ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-b)   [bin=bin]
org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, so the director could not start a feature org (ralph org start --plan runs git worktree add); set [org.permissions.roles].director = "guarded" and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, or start a claude director (--driver claude); a guarded codex director follows that config, so under sandbox_mode = "workspace-write" it meets the same wall
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-b
[no such directory]

## Q3. codex director, ralph.toml: roles.director = edits, codex_verified = true
$ (cd work/c && ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-c)   [bin=bin]
org: director: start: permission mode "edits" of role director ([org.permissions.roles].director, else [org.permissions].default): a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, so the director could not start a feature org (ralph org start --plan runs git worktree add); set [org.permissions.roles].director = "guarded" and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, or start a claude director (--driver claude); a guarded codex director follows that config, so under sandbox_mode = "workspace-write" it meets the same wall
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-c
[no such directory]

## Q4. --config cfg/ralph.toml with agmsg_home = "agmsg/home" (claude director)
$ (cd work/e && ralph org director start --driver claude --model haiku --state-dir $PROBE/state-e --config cfg/ralph.toml)   [bin=bin]
org: director start: [org].agmsg_home "agmsg/home" in the --config file cfg/ralph.toml is a relative path: the director and the watch may resolve it from different directories and so use different agmsg installs (the director runs in the ledger's main worktree root when the ledger has one, while the watch resolves it from this directory), so start refuses it wherever it runs; give an absolute path (or one that starts with ~/)
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-e
[no such directory]

## Q5. --config cfg/ralph.toml with agmsg_home = "~"
$ (cd work/f && ralph org director start --driver claude --model haiku --state-dir $PROBE/state-f --config cfg/ralph.toml)   [bin=bin]
org: director start: [org].agmsg_home "~" in the --config file cfg/ralph.toml is a relative path: the director and the watch may resolve it from different directories and so use different agmsg installs (the director runs in the ledger's main worktree root when the ledger has one, while the watch resolves it from this directory), so start refuses it wherever it runs; give an absolute path (or one that starts with ~/)
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-f
[no such directory]

## Q6. positive control: guarded codex director with codex_verified, agmsg_home ~/..., ledger outside the cwd, RALPH_ORG_AGMSG_HOME set
$ (cd work/d && RALPH_ORG_AGMSG_HOME=/nonexistent/agmsg ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-d --config cfg/ralph.toml)   [bin=bin]
warning: director start: RALPH_ORG_AGMSG_HOME is set here, but the director's herdr pane has the herdr server's environment, so the director does not get it from here; [org].agmsg_home in the ralph.toml the director reads (the --config file, when given) reaches it
ralph org director start: started the cross-org watch (pid 79185); log: $PROBE/state-d/watch.log
started d1 (driver=codex, model=gpt-6-astra, permission=guarded, pane=w1:p1, workspace=w1)
talk to the director in herdr: pane w1:p1, in the tab director_232d1401dbed of the workspace labelled ralph_director_232d1401dbed
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state-d'
[exit 0]
[stub herdr calls during this command: 3]
herdr [workspace] [create] [--cwd] [$PROBE/work/d] [--label] [ralph_director_232d1401dbed]
herdr [tab] [create] [--workspace] [w1] [--cwd] [$PROBE/work/d] [--label] [director_232d1401dbed]
herdr [agent] [start] [director_232d1401dbed] [--kind] [codex] [--pane] [w1:p1] [--timeout] [60000] [--] [--model] [gpt-6-astra] [役割指示を読み込んで従ってください: $PROBE/state-d/prompts/director_232d1401dbed.md]
$ ls -A state-d
director.jsonl
director.lock
prompts
watch-alive.json
watch.lock
watch.log
[6 entries]
$ cat state-d/director.jsonl
{"ts":"2026-10-15T15:31:10Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"herdr","token":"232d1401dbed","model":"gpt-6-astra","permission_mode":"guarded","user":"probe","cwd":"$PROBE/work/d","herdr_pane":false}
{"ts":"2026-10-15T15:31:11Z","id":"d1","event":"workspace_created","workspace_id":"w1","user":"probe","cwd":"$PROBE/work/d","herdr_pane":false}
{"ts":"2026-10-15T15:31:11Z","id":"d1","event":"pane_created","pane_id":"w1:p1","user":"probe","cwd":"$PROBE/work/d","herdr_pane":false}
{"ts":"2026-10-15T15:31:11Z","id":"d1","event":"launched","user":"probe","cwd":"$PROBE/work/d","herdr_pane":false}
$ grep -n -- '--config\|設定' state-d/prompts/director_*.md | head -5
5:- 設定: '$PROBE/work/d/cfg/ralph.toml'(start の --config)
13:ralph org director status --state-dir '$PROBE/state-d' --config '$PROBE/work/d/cfg/ralph.toml'
30:- 会話の要約のあとに `ralph org director status --state-dir '$PROBE/state-d' --config '$PROBE/work/d/cfg/ralph.toml'` で
33:- あなたが打つ `ralph org` のコマンドには、どれも `--state-dir '$PROBE/state-d' --config '$PROBE/work/d/cfg/ralph.toml'` を付ける。
35:  どこでも、この台帳と、start が読んだ設定を使うため。終わりに人に頼む
$ (cd work/d && ralph org director stop --id d1 --state-dir $PROBE/state-d)   [bin=bin]
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]
[stub herdr calls during this command: 7]
herdr [pane] [get] [w1:p1]
herdr [tab] [get] [w1:t1]
herdr [workspace] [get] [w1]
herdr [workspace] [get] [w1]
herdr [pane] [send-keys] [w1:p1] [C-c]
herdr [pane] [close] [w1:p1]
herdr [workspace] [close] [w1]
$ (cd work/d && ralph org watch --stop --state-dir $PROBE/state-d)   [bin=bin]
stopped ralph org watch --all (pid 79185)
[exit 0]
[stub herdr calls during this command: 0]

## Q7. relative agmsg_home in ./ralph.toml without --config: not refused
$ (cd work/g && ralph org director start --driver claude --model haiku --state-dir $PROBE/state-g)   [bin=bin]
ralph org director start: started the cross-org watch (pid 79318); log: $PROBE/state-g/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w2:p1, workspace=w2)
talk to the director in herdr: pane w2:p1, in the tab director_f23b45caa595 of the workspace labelled ralph_director_f23b45caa595
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state-g'
[exit 0]
[stub herdr calls during this command: 3]
herdr [workspace] [create] [--cwd] [$PROBE/work/g] [--label] [ralph_director_f23b45caa595]
herdr [tab] [create] [--workspace] [w2] [--cwd] [$PROBE/work/g] [--label] [director_f23b45caa595]
herdr [agent] [start] [director_f23b45caa595] [--kind] [claude] [--pane] [w2:p1] [--timeout] [60000] [--] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state-g/prompts/director_f23b45caa595.md]
$ (cd work/g && ralph org director stop --id d1 --state-dir $PROBE/state-g)   [bin=bin]
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]
[stub herdr calls during this command: 7]
[note: the same 7 calls as Q6's stop, on w2 and w2:p1]
$ (cd work/g && ralph org watch --stop --state-dir $PROBE/state-g)   [bin=bin]
stopped ralph org watch --all (pid 79318)
[exit 0]
[stub herdr calls during this command: 0]

## Q8. pre-fix binary (4b721b18) on Q2's config
$ (cd work/b && ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-pre-b)   [bin=prebin]
ralph org director start: started the cross-org watch (pid 79512); log: $PROBE/state-pre-b/watch.log
started d1 (driver=codex, model=gpt-6-astra, permission=autonomous, pane=w3:p1, workspace=w3)
talk to the director in herdr: pane w3:p1, in the tab director_de3bf01610fa of the workspace labelled ralph_director_de3bf01610fa
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state-pre-b'
[exit 0]
[stub herdr calls during this command: 3]
herdr [workspace] [create] [--cwd] [$PROBE/work/b] [--label] [ralph_director_de3bf01610fa]
herdr [tab] [create] [--workspace] [w3] [--cwd] [$PROBE/work/b] [--label] [director_de3bf01610fa]
herdr [agent] [start] [director_de3bf01610fa] [--kind] [codex] [--pane] [w3:p1] [--timeout] [60000] [--] [--sandbox] [workspace-write] [--ask-for-approval] [never] [--add-dir] [$PROBE/state-pre-b] [--model] [gpt-6-astra] [役割指示を読み込んで従ってください: $PROBE/state-pre-b/prompts/director_de3bf01610fa.md]
[note: the stop (exit 0, the same 7 calls on w3) and `ralph org watch --stop` (pid 79512) followed]

## Q9. pre-fix binary (4b721b18) on Q4's config
$ (cd work/e && ralph org director start --driver claude --model haiku --state-dir $PROBE/state-pre-e --config cfg/ralph.toml)   [bin=prebin]
ralph org director start: started the cross-org watch (pid 79629); log: $PROBE/state-pre-e/watch.log
started d1 (driver=claude, model=haiku, permission=guarded, pane=w4:p1, workspace=w4)
talk to the director in herdr: pane w4:p1, in the tab director_3fa7add67038 of the workspace labelled ralph_director_3fa7add67038
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state-pre-e'
[exit 0]
[stub herdr calls during this command: 3]
herdr [workspace] [create] [--cwd] [$PROBE/work/e] [--label] [ralph_director_3fa7add67038]
herdr [tab] [create] [--workspace] [w4] [--cwd] [$PROBE/work/e] [--label] [director_3fa7add67038]
herdr [agent] [start] [director_3fa7add67038] [--kind] [claude] [--pane] [w4:p1] [--timeout] [60000] [--] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state-pre-e/prompts/director_3fa7add67038.md]
[note: the stop (exit 0, the same 7 calls on w4) and `ralph org watch --stop` (pid 79629) followed]

## Q10. pre-fix binary (4b721b18) on Q1 (no ralph.toml)
$ (cd work/a && ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-pre-a)   [bin=prebin]
org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): org: codex seat permission mode "autonomous" requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first: docs/recipes/codex-seat-permissions.md)
[exit 1]
[stub herdr calls during this command: 0]
$ ls -A state-pre-a
[no such directory]
```

The five refusals of the HEAD binary (Q1 to Q5) exit 1, call no herdr, launch no watch, and leave no state directory at all. Q8 and Q9 are the cross-review cycle-2 findings 1 and 2 reproduced on the pre-fix binary: with `codex_verified = true` the codex director launched under `--sandbox workspace-write --ask-for-approval never --add-dir <ledger>`, and the relative `agmsg_home` of a `--config` was accepted. Q10 shows that before the fix a default config already refused codex, but with the codex_verified message, which pointed at the recipe that unlocks the mode now refused.

Q11, run after Q1 to Q10 with the HEAD binary: the `RALPH_ORG_AGMSG_HOME` warning comes before the D12 refusal, so a refused start prints it too.

```text
$ (cd work/b && RALPH_ORG_AGMSG_HOME=/nonexistent/agmsg ralph org director start --driver codex --model gpt-6-astra --state-dir $PROBE/state-v5)   [bin=bin]
warning: director start: RALPH_ORG_AGMSG_HOME is set here, but the director's herdr pane has the herdr server's environment, so the director does not get it from here; [org].agmsg_home in the ralph.toml the director reads (the --config file, when given) reaches it
org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): a codex director runs only in permission mode guarded, whatever [org.permissions].codex_verified says: [note: the rest as in Q2]
[exit 1]
[stub herdr calls: 0]
$ ls -d state-v5
"$PROBE/state-v5": No such file or directory (os error 2)
```

### C3-4. ralph org director start --help of the HEAD binary (the permission and refusal paragraphs)

```text
$ ralph org director start --help | sed -n '/^The director runs with/,/^Usage:/p'
The director runs with the permission mode of
[org.permissions.roles].director, else [org.permissions].default, and
with --model, which [org.roles].director must allow; without --model,
with the first model_pool entry for --driver that it allows, and a
warning. Without --config, those settings and [org.director].enabled
are read from the main worktree's ralph.toml when the ledger is the main
worktree's, so a feature worktree's ralph.toml does not choose them. The
director runs in the ledger's main worktree root, else in the directory
start runs in. With --config, the ralph org commands the director's
prompt has it run carry --config with that file's absolute path, so the
director reads the settings start read (the prompt's stop command, for
the human, carries none: stop reads no ralph.toml). RALPH_ORG_AGMSG_HOME
set where start runs does not reach the director, whose herdr pane has
the herdr server's environment: start warns when it is set, and
[org].agmsg_home in ralph.toml is what reaches the director.

A codex director runs only in guarded: start refuses autonomous and
edits whatever [org.permissions].codex_verified says, because they run
codex under --sandbox workspace-write, which keeps .git read-only, so
the director could not start a feature org (ralph org start --plan runs
git worktree add). Set [org.permissions.roles].director = "guarded" and
let your codex config (~/.codex/config.toml) write .git and reach the
network, or use --driver claude; a guarded codex director follows that
config, so under sandbox_mode = "workspace-write" it meets the same
wall.

start is refused, and writes nothing and runs no herdr, when
[org.director].enabled is false, for a codex director in autonomous or
edits, when the --config file's [org].agmsg_home is a relative path
(neither absolute nor starting with ~/: the director runs in the
ledger's main worktree root when the ledger has one and the watch
resolves it from where start runs, so the two may resolve it from
different directories, and start refuses it wherever it runs; give an
absolute path), while a director is attached (one still starting
included: stop a headless one with `ralph org director stop --id <id>`,
detach a promoted session), and inside a herdr pane (HERDR_PANE_ID is
set): run start from a terminal outside herdr. start has no --replace.
[note: the rest of the paragraph (the watch, the cleanup) is unchanged since cycle 2]
```

The cycle-2 quote of this help in C2-6 ("for codex, autonomous and edits need [org.permissions].codex_verified") describes the tree before caf1d025; the HEAD help above replaces it.

### C3-5. Documentation drift greps

```text
$ grep -o '`--config` の `\[org\].agmsg_home` が相対パスなら[^。]*。' docs/specs/2026-10-07-org-multi-org-director.md
`--config` の `[org].agmsg_home` が相対パスなら、director と見張りが別の場所から解くので、start が断る(D13)。
$ grep -c 'may resolve it from different directories' README.md internal/cli/org_director_headless.go
README.md:1
internal/cli/org_director_headless.go:1
$ sed -n 187p .claude/skills/org/SKILL.md | grep -c 'guarded'
0
$ sed -n 187p .claude/skills/org/SKILL.md | grep -c 'agmsg_home'
0
$ sed -n 1152,1185p .claude/skills/org/SKILL.md | grep -c 'director start` が断る\|codex の director'
0
$ grep -n 'const notNeeded' internal/cli/doctor_codex_writable_root_test.go
982:	const notNeeded = "not needed: no role that can use a codex model resolves to edits or autonomous and <cfg> does not set sandbox_mode = \"workspace-write\""
$ grep -n '"a director in edits"' internal/cli/doctor_codex_writable_root_test.go
992:		{"a director in edits", true, "guarded", map[string]string{"director": "edits"}, "", "pass", notNeeded},
$ git grep -n -e 'add-dir' -- ':!docs/reports' ':!docs/evidence' ':!docs/plans/archive' ':!*_test.go' ':!docs/insights' | grep -i director
[note: 15 lines. The ones about the director are the plan (44, 91, 112, 124, 162: no --add-dir, D12), the recipe (both copies, 96 and 98: "The headless director ... gets no --add-dir"), tech-debt 259 (the D12 row), and director_start.go:459 ("no --add-dir: a codex director runs guarded only"). The rest match only "directory"/"directories" or the spec's file name (spec 182 and 236, tech-debt 169, tests/test-codex-exec-invocation.sh:124) and are about the leader seat or not about --add-dir for a director. None says a director gets --add-dir or needs codex_verified]
```

The `/org` skill's line 187 is its `director` row, and lines 1152 to 1185 are the `## permission 作法` items on the permission modes and the doctor's writable-root check.

### C3-6. Cleanup and the files outside the scratch directory

- Every watch the starts launched was stopped with `ralph org watch --stop` (pids 79185, 79318, 79512, 79629). Afterwards `pgrep -fl "$PROBE/"`, `pgrep -fl 'herdr server'`, and `pgrep -fl osascript` printed nothing (exit 1 each).
- `$PROBE/hstate` holds only `calls.log` and the workspace counter: every stub workspace, tab, and pane the starts made was closed by a stop. `$PROBE/home` and `$PROBE/tmp` are empty.
- `~/.codex/config.toml` before and after the probe: mtime `Oct 15 17:51:28 2026` (epoch 1792054288), size 33673, sha256 `ac1a4821574050d3630118510c7dd591bf1839b947f2bb35fe4106e41b61796f`, the same both times. No codex ran (none was on the probe's PATH).
- The repo's ledger was not written: neither `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` nor the worktree's `.harness/state/org` exists.
- The scratch ledgers, the two binaries, the stub, and the probe script stay in the session scratch directory.

## Cycle 4 (HEAD d0776dab)

Scope: the cross-review cycle-3 fixes D14 and D15 (plan 8a0356de, code b888d637) and the cycle-4 self-review LOW fixes (5097b67e), read against cycle 3's verify (3504ee30). The raw output of this section is appended to `docs/evidence/verify-2026-10-15-org-director-start.log`, and the wrapper's own log is `docs/evidence/verify-2026-10-15-183402.log` (both gitignored).

### C4-1. ./scripts/run-static-verify.sh (changed scope)

```text
# Verification run
- Timestamp: 2026-10-15T18:34:02Z
- Mode: static
- Requested scope: changed
==> Running local verifier
==> shellcheck hook + verify scripts                        OK
==> sh -n of the 10 hooks, root and templates/base          OK (20)
==> awk parse of the guard in .claude/hooks and the template OK (2)
==> jq -e . .claude/settings.json and the template           OK (2)
==> Codex hook single-source guard                           OK
==> Codex inline hook detector smoke test                    OK
==> Codex PR provenance policy guard                         OK
==> scripts/check-sync.sh                                    OK (IDENTICAL 169, DRIFTED 0, ROOT_ONLY 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5)
==> scripts/check-pipeline-sync.sh                           OK
==> scripts/check-skill-sync.sh                              OK ([ok] check-skill-sync: 14 skill(s) in lock-step)
==> scripts/check-template-purity.sh                         OK
==> tech-debt README plan references                         OK
==> Language scope: full fallback (unclassified:templates/base/ralph.toml)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned f21df66c..d0776dab against origin/main: clean
==> All verifiers passed.
Evidence saved to: docs/evidence/verify-2026-10-15-183402.log
rc=0
```

(The `==>` lines are folded one per check; the jq checks also print the two settings files, left out here. The full output is in the raw log.)

### C4-2. Go, sync, and frontmatter checks

```text
$ go version
go version go1.26.0 darwin/arm64
$ gofmt -l internal/ cmd/        -> no output, rc=0
$ go vet ./...                   -> no output, rc=0
$ GOTOOLCHAIN=go1.25.8 go version
go version go1.25.8 darwin/arm64
$ GOTOOLCHAIN=go1.25.8 go build ./...   -> rc=0
$ GOTOOLCHAIN=go1.25.8 go vet ./...     -> rc=0
$ "$(GOTOOLCHAIN=go1.25.8 go env GOROOT)/bin/gofmt" -l ./cmd ./internal   -> no output, rc=0
$ grep -n '^go ' go.mod
3:go 1.25.8
$ ./scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step          (rc=0)
$ ./scripts/check-sync.sh
IDENTICAL 169, DRIFTED 0, ROOT_ONLY 0, TEMPLATE_ONLY 11, KNOWN_DIFF 5; PASS: all files in sync.   (rc=0)
$ git diff --check origin/main...HEAD   -> no output, rc=0
$ git diff --check 3504ee30..HEAD       -> no output, rc=0
$ ./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-org-director-start.md
ace92dd0aaa7
$ md5 -q of the four /director SKILL.md copies
d6c990cdf5ef5d015af78867f59ec7be (all four)
$ md5 -q of the four /org SKILL.md copies
7439019e58b88ed23394c829dd00cbb7 (all four)

$ ruby -v
ruby 3.4.10 (2026-06-30 revision 2b0b7728dc) +PRISM [arm64-darwin25]
$ for f in $(git ls-files '*SKILL.md'); do ruby -ryaml -e 'YAML.safe_load(File.read(ARGV[0]).split(/^---$/)[1])' "$f"; done
ok=54 fail=0
$ git ls-files '*SKILL.md' | wc -l              -> 54
$ git ls-files '*SKILL.md' outside the four skillRoots of skill_frontmatter_test.go   -> none (grep rc=1)
$ find the four skillRoots -name SKILL.md | wc -l   -> 54

# the four /director copies at 8a0356de (before b888d637) and at HEAD
== .claude/skills/director/SKILL.md
old: Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 3 column 155
head: keys=["name", "description"] desc_equals_raw=true len=672
== .agents/skills/director/SKILL.md
old: Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 3 column 155
head: keys=["name", "description"] desc_equals_raw=true len=672
== templates/base/.claude/skills/director/SKILL.md
old: Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 3 column 155
head: keys=["name", "description"] desc_equals_raw=true len=672
== templates/base/.agents/skills/director/SKILL.md
old: Psych::SyntaxError: (<unknown>): mapping values are not allowed in this context at line 3 column 155
head: keys=["name", "description"] desc_equals_raw=true len=672
```

`desc_equals_raw=true` means the parsed description equals the raw text after `description: `, so no ` #` cut it. A separate awk scan of every tracked SKILL.md frontmatter for a top-level plain value (not starting with `"`, `'`, `>`, or `|`) holding a colon followed by a space or a tab, a colon at the end, or a space or a tab before `#` printed nothing: the shapes `TestSkillFrontmatter_UnquotedValuesHaveNoColonSpaceOrSpaceHash` looks for are absent at HEAD. The Go test itself was not run (that is /test).

### C4-3. Stub herdr probe: D14 with the HEAD binary and the pre-fix binary

Setup:

- HEAD binary: `go build -o $PROBE/bin/ralph ./cmd/ralph` from the clean worktree at d0776dab. Pre-fix binary: `git archive 8a0356de` (the plan commit right before b888d637) built into `$PROBE/prebin/ralph`. Each bin directory holds only ralph, a symlink to git, and the same stub `herdr`: a POSIX sh script that logs every call to `$PROBE/hstate/calls.log`, keeps the workspace and tab labels in files there, and answers in the envelope shapes of `internal/org/driver/herdr.go`. The stub never talks to a herdr server.
- Every ralph command ran under `env -i HOME=$PROBE/home USER=probe LOGNAME=probe PATH=<bin dir> TMPDIR=$PROBE/tmp/ LANG=en_US.UTF-8`, so no real herdr, codex, claude, agmsg, or osascript could be found. The cwd `$PROBE/work/p1` (or `p2`) is outside any git repo, with a `ralph.toml` of `[org.permissions.roles] director = "guarded"`; the ledger is an explicit `--state-dir`.
- The overlap is staged with markers, not with sleeps:
  - P1 and P3: a real start and a real stop. With `hold_tab`, the stub's `tab create` (start's step 3, after start recorded `workspace_created`) writes `tab_waiting` and waits for `release_tab`. The probe runs stop once `tab_waiting` exists. With `hold_close`, the stub's `workspace close` (stop's close) writes `release_tab` and waits until the ledger has a `launched` line (P1). With `hold_close_until_stopped` (one-shot) and `fail_agent`, start's agent start fails, and stop's close waits until the ledger has a `stopped` line (P3).
  - P2: the cross-review cycle-3 shape, where stop reads a registration with nothing recorded and so calls no herdr. The ledger holds one `attached` line written by hand in the format P1's start wrote. A perl script (`flock(2)` LOCK_EX on `director.lock`, the lock `internal/org/lockfile.go` polls with LOCK_EX|LOCK_NB) holds the lock; the probe starts stop in the background and, once `lsof -p <stop pid>` shows `director.lock` open (stop is in `withFileLock`, past its read), lets the holder append `workspace_created`, `pane_created`, and `launched` (the lines a start writes, in P1's format) and drop the lock. The start is emulated; the stop is the real binary.

```text
## P1 (HEAD d0776dab): stop reads the workspace only; start records the pane and launched while stop closes the workspace
$ (cd work/p1 && ralph org director start --driver claude --model haiku --state-dir $L) &
[start is waiting in herdr tab create; director.jsonl now:]
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"3dd3b2c9950f","model":"haiku","permission_mode":"guarded","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"workspace_created","workspace_id":"w1","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}

$ (cd work/p1 && ralph org director stop --id d1 --state-dir $L)
[stop exit 1]
[stop stdout]
[stop stderr]
stop d1: ctrl_c=skipped, pane=none recorded, workspace=closed
org: director: stop d1: record d1 stopped: a start in progress recorded a workspace or pane after stop read the registration: d1 had workspace "w1" and no pane when stop read it, and has workspace "w1" and pane "w1:p1" now; this stop closed only what it had read; nothing was recorded; run ralph org director stop --id d1 --state-dir '$PROBE/state-p1-bin' again to close what start recorded
[start exit 0]
[start stdout]
started d1 (driver=claude, model=haiku, permission=guarded, pane=w1:p1, workspace=w1)
talk to the director in herdr: pane w1:p1, in the tab director_3dd3b2c9950f of the workspace labelled ralph_director_3dd3b2c9950f
stop it with: ralph org director stop --id d1 --state-dir '$PROBE/state-p1-bin'
[start stderr]
ralph org director start: started the cross-org watch (pid 33033); log: $PROBE/state-p1-bin/watch.log
[stub herdr calls from start's launch to here: 5, stop's share starts at line 3]
herdr [workspace] [create] [--cwd] [$PROBE/work/p1] [--label] [ralph_director_3dd3b2c9950f]
herdr [tab] [create] [--workspace] [w1] [--cwd] [$PROBE/work/p1] [--label] [director_3dd3b2c9950f]
herdr [workspace] [get] [w1]
herdr [workspace] [close] [w1]
herdr [agent] [start] [director_3dd3b2c9950f] [--kind] [claude] [--pane] [w1:p1] [--timeout] [60000] [--] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state-p1-bin/prompts/director_3dd3b2c9950f.md]

$ cat director.jsonl
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"3dd3b2c9950f","model":"haiku","permission_mode":"guarded","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"workspace_created","workspace_id":"w1","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"pane_created","pane_id":"w1:p1","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"launched","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}

$ ralph org director status --state-dir $L
director: attached d1 (driver=claude, headless running, model=haiku, pane w1:p1, workspace w1, attached at 2026-10-15T18:39:52Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
(...)
state: running
token: 3dd3b2c9950f
workspace_id: w1
pane_id: w1:p1
[exit 0]

$ ralph org director stop --id d1 --state-dir $L   (run again)
stopped d1 (ctrl_c=skipped, pane=already closed, workspace=already closed)
the director is no longer attached
[exit 0]
[stub herdr calls: 2]
herdr [pane] [get] [w1:p1]
herdr [workspace] [get] [w1]
$ tail -n 2 director.jsonl
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:39:52Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
$ ralph org director status --state-dir $L
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
enabled: true
[exit 0]
$ ralph org watch --stop --state-dir $L
stopped ralph org watch --all (pid 33033)
[exit 0]

## P1 (pre-fix 8a0356de), the same staging
$ (cd work/p1 && ralph org director stop --id d1 --state-dir $L)
[stop exit 0]
[stop stdout]
stopped d1 (ctrl_c=skipped, pane=none recorded, workspace=closed)
the director is no longer attached
$ cat director.jsonl
(attached, workspace_created w2, pane_created w2:p1, launched, then:)
{"ts":"2026-10-15T18:40:04Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:40:04Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
[start exit 0: started d1 (... pane=w2:p1, workspace=w2)]

## P2 (HEAD d0776dab): stop reads nothing recorded; the emulated start records the workspace, the pane, and launched while stop waits on director.lock
$ cat director.jsonl   (before stop)
{"ts":"2026-10-16T00:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"launch":"herdr","token":"0123456789ab","model":"haiku","permission_mode":"guarded","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
[director.lock is held by the emulated start (pid 35560)]
[lsof: stop (pid 35595) has director.lock open: yes]
[lock holder exit 0; appended: yes]
[stop exit 1]
[stop stdout]
[stop stderr]
stop d1: ctrl_c=skipped, pane=none recorded, workspace=none recorded
org: director: stop d1: record d1 stopped: a start in progress recorded a workspace or pane after stop read the registration: d1 had no workspace and no pane when stop read it, and has workspace "w9" and pane "w9:p1" now; this stop closed only what it had read; nothing was recorded; run ralph org director stop --id d1 --state-dir '$PROBE/state-p2-bin' again to close what start recorded
[stub herdr calls during stop: 0]
$ cat director.jsonl
{"ts":"2026-10-16T00:00:00Z","id":"d1","event":"attached",...,"token":"0123456789ab",...}
{"ts":"2026-10-16T00:00:00Z","id":"d1","event":"workspace_created","workspace_id":"w9","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
{"ts":"2026-10-16T00:00:00Z","id":"d1","event":"pane_created","pane_id":"w9:p1","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
{"ts":"2026-10-16T00:00:00Z","id":"d1","event":"launched","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
$ ralph org director status --state-dir $L
director: attached d1 (driver=claude, headless running, model=haiku, pane w9:p1, workspace w9, attached at 2026-10-16T00:00:00Z, by probe) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
[exit 0]
$ ralph org director stop --id d1 --state-dir $L   (run again)
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
[exit 0]
[stub herdr calls: 7]
herdr [pane] [get] [w9:p1]
herdr [tab] [get] [w9:t1]
herdr [workspace] [get] [w9]
herdr [workspace] [get] [w9]
herdr [pane] [send-keys] [w9:p1] [C-c]
herdr [pane] [close] [w9:p1]
herdr [workspace] [close] [w9]
$ tail -n 2 director.jsonl
{"ts":"2026-10-15T18:40:33Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
{"ts":"2026-10-15T18:40:33Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
[stub state left for w9: 0]

## P2 (pre-fix 8a0356de), the same staging (token fedcba987654, w8)
[lsof: stop (pid 36167) has director.lock open: yes]
[lock holder exit 0; appended: yes]
[stop exit 0]
[stop stdout]
stopped d1 (ctrl_c=skipped, pane=none recorded, workspace=none recorded)
the director is no longer attached
no workspace id was recorded for d1: herdr may still have a workspace labelled ralph_director_fedcba987654 that start made before it could record it; look for it with herdr workspace list
[stop stderr]
[stub herdr calls during stop: 0]
$ cat director.jsonl
(attached, then the emulated start's workspace_created w8, pane_created w8:p1, launched, then:)
{"ts":"2026-10-15T18:40:40Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
{"ts":"2026-10-15T18:40:40Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work/p2","herdr_pane":false}
$ ralph org director status --state-dir $L
director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work
$ ralph org director stop --id d1 --state-dir $L   (run again)
org: director: stop d1: no open headless registration has the id: the headless registration was already stopped: d1 (driver claude, attached 2026-10-16T00:00:00Z, headless stopped)
[exit 1]
[stub herdr calls: 0]
[stub state left for w8: 3]   (the workspace, tab, and pane stay open in the stub, recorded stopped)

## P3 (HEAD d0776dab): as P1, but start's agent start fails, and start's cleanup records the stop while stop's workspace close is still in flight
$ (cd work/p1 && ralph org director stop --id d1 --state-dir $L)
[stop exit 1]
[stop stderr]
stop d1: ctrl_c=skipped, pane=none recorded, workspace=closed
org: director: stop d1: record d1 stopped: org: director: stop d1: no open headless registration has the id: the headless registration was already stopped: d1 (driver claude, attached 2026-10-15T18:42:29Z, headless stopped); nothing was recorded; record the stop with ralph org director stop --id d1 --state-dir '$PROBE/state-p3-bin'
[start exit 1]
[start stderr]
ralph org director start: started the cross-org watch (pid 42316); log: $PROBE/state-p3-bin/watch.log
org: director: start d1: herdr agent start director_3d8b252f5eb7 (agent_start_retries=0): herdr: exit status 1 (herdr: stub_failure: stub: agent start fails); start closed what it had made in herdr, and d1 is recorded stopped
[stub herdr calls from start's launch to here: 12]
herdr [workspace] [create] [--cwd] [$PROBE/work/p1] [--label] [ralph_director_3d8b252f5eb7]
herdr [tab] [create] [--workspace] [w3] [--cwd] [$PROBE/work/p1] [--label] [director_3d8b252f5eb7]
herdr [workspace] [get] [w3]
herdr [workspace] [close] [w3]
herdr [agent] [start] [director_3d8b252f5eb7] [--kind] [claude] [--pane] [w3:p1] [--timeout] [60000] [--] [--model] [haiku] [役割指示を読み込んで従ってください: $PROBE/state-p3-bin/prompts/director_3d8b252f5eb7.md]
herdr [pane] [get] [w3:p1]
herdr [tab] [get] [w3:t1]
herdr [workspace] [get] [w3]
herdr [workspace] [get] [w3]
herdr [pane] [send-keys] [w3:p1] [C-c]
herdr [pane] [close] [w3:p1]
herdr [workspace] [close] [w3]
$ cat director.jsonl
(attached, workspace_created w3, pane_created w3:p1, then:)
{"ts":"2026-10-15T18:42:30Z","id":"d1","event":"detached","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
{"ts":"2026-10-15T18:42:30Z","id":"d1","event":"stopped","user":"probe","cwd":"$PROBE/work/p1","herdr_pane":false}
$ ralph org director stop --id d1 --state-dir $L   (run again, as the first stop's error says)
org: director: stop d1: no open headless registration has the id: the headless registration was already stopped: d1 (driver claude, attached 2026-10-15T18:42:29Z, headless stopped)
[exit 1]
[stub herdr calls: 0]
$ ralph org watch --stop --state-dir $L
stopped ralph org watch --all (pid 42316)
[exit 0]
```

In P1 the stub lets start's agent start answer before stop's `workspace close` lands, so the HEAD record says `running` for a workspace stop has already closed until the second stop finds both gone. In P3 the first stop's error says "record the stop with ralph org director stop --id d1 ...", but start's cleanup had already recorded d1 stopped, and the stop it names is refused as already stopped (the report's C4-V3).

### C4-4. ralph org director stop --help of the HEAD binary (the D14 sentence)

```text
(...) --force, a human's cleanup, then
records the director stopped anyway, closing nothing more, and prints
what was left open and how to find it (herdr workspace list, the
workspace labelled ralph_director_<token>); when the labels match,
--force closes as usual. When a start that is still running records a
workspace or pane after stop read the registration, stop has not closed
it, so it records nothing and exits 1 with what it closed: run it again
to close what start recorded. stop reads no ralph.toml (nor the main
worktree's), so one that does not load does not block it.
```

### C4-5. Documentation drift greps

```text
$ sed -n 186,214p .claude/skills/director/SKILL.md   (### 終える, items 1-6)
item 5 names three refusals (a label that does not match, a label that cannot be checked, a close that fails) and --force; no item names the stop refused because a start recorded more meanwhile
$ grep -n 'director stop' .claude/skills/org/SKILL.md
59:  `director start` / `director stop`)は
187:| `director` | ... `director stop --id <id> [--force]` は、... label が合わない・確かめられない・閉じられないときは断られ、`--force` ならそのまま `stopped` を記録する(閉じていないものを出す)。director の pane か workspace の中からの stop は、`--force` でも断られる。... |
$ README.md:258 (the director paragraph)
"... is ended with `ralph org director stop --id <id>`, run from outside its workspace, which checks the labels, closes the pane and the workspace, and records it stopped (`--force` records a stop it cannot check or close, ...)"
$ docs/specs/2026-10-07-org-multi-org-director.md:85 (the 8a-2 note)
"stop は pane と workspace の label がその起動の token の名前 ... であることを確かめてから閉じ、`stopped` を書く。" (no --force, no refusal)
$ grep -c 'MarkStoppedIfUnchanged\|ACutOffWriteIsRestored\|D14\|D15' docs/tech-debt/README.md
0
$ grep -rn -i 'not available yet\|start はまだ\|headless の director はまだ' --include='*.md' --include='*.go' . (outside docs/reports, docs/evidence, docs/plans)
./internal/cli/org_director_test.go:816, 901, 902 (the test that checks the phrase is gone)
```

### C4-6. Cleanup and the files outside the scratch directory

- The three watches the starts launched were stopped with `ralph org watch --stop` (pids 33033, 33756, 42316). Afterwards `pgrep -fl "$PROBE/"`, `pgrep -fl 'herdr server'`, and `pgrep -fl osascript` printed nothing (exit 1 each).
- `$PROBE/hstate` holds `calls.log`, the workspace counter, the ledger pointer, and the stub's w8 workspace, tab, and pane: the pre-fix P2 left them open while it recorded d1 stopped (that is the finding). Every other stub workspace and pane was closed. `$PROBE/home` and `$PROBE/tmp` are empty.
- `~/.codex/config.toml` before and after the probe: mtime `Oct 15 17:51:28 2026` (epoch 1792054288), size 33673, sha256 `ac1a4821574050d3630118510c7dd591bf1839b947f2bb35fe4106e41b61796f`, the same both times. No codex ran (none was on the probe's PATH). The user's herdr server and `~/.config/herdr` were not touched (the stub is the only herdr on the probe's PATH).
- The repo's ledger was not written: neither `~/MyDev/github.com/yoshpy-dev/ralph/.harness/state/org` nor the worktree's `.harness/state/org` exists.
- The scratch ledgers, the two binaries, the stub, the lock holder, and the probe scripts stay in the session scratch directory.
