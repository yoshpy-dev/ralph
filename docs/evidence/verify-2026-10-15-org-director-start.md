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
