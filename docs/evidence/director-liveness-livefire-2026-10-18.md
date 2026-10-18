# Live check: the watch's check of a headless codex director and start's cleanup (stage 8b-2a, S6)

Plan: `docs/plans/active/2026-10-18-org-director-liveness.md` (D9, AC9).
Two runs against an isolated herdr server and an isolated ledger: (a) a
codex director ended with `/quit`, then checked by the watch and cleaned up
by the next `ralph org director start`; (b) a `ralph org director start`
killed while its agent was starting, then checked by the watch and cleaned up
by the next start. One more start was then made against a live director, to
see the refusal.

Values left out: the launch tokens are `<token>`, the herdr panes are
`<pane-d1>` … `<pane-d4>` (or `<pane>`), the pids of the start processes are
`<pid>`, and `$HOME` is `~`.

## Setup

- ralph: built from branch `feat/org-director-liveness` at `0b9b7ed3` (S1–S4):
  `go build -o /tmp/rlv/bin/ralph ./cmd/ralph`
- herdr 0.7.5 (`/opt/homebrew/bin/herdr`) and codex-cli 0.160.1 (its native
  binary under `~/.local/share/mise/installs/npm-openai-codex/0.160.1/`),
  symlinked into `/tmp/rlv/bin` with ralph and a stub `osascript` that only
  appends its arguments to `/tmp/rlv/osascript.log`
- The herdr server was started with `env -i HOME=/tmp/rlv/home
  PATH=/tmp/rlv/bin:/usr/bin:/bin SHELL=/bin/zsh XDG_CONFIG_HOME=/tmp/rlv/cfg
  XDG_STATE_HOME=/tmp/rlv/state HERDR_SOCKET_PATH=/tmp/rlv/h.sock herdr server`
  (it printed `api socket: /tmp/rlv/h.sock`). Every ralph and herdr command
  ran under the same `env -i`, with `--state-dir /tmp/rlv/ledger`, and the
  org verbs also with `--config /tmp/rlv/proj/ralph.toml`. No herdr server of
  the user ran before or after
- The fake HOME was set up as in `docs/evidence/director-wake-livefire-2026-10-18.md`
  (zsh startup files that only put `/tmp/rlv/bin` first on `PATH`; a
  `.codex/config.toml` with `model = "gpt-6-astra"`, `model_reasoning_effort
  = "low"`, `sandbox_mode = "workspace-write"`, `approval_policy =
  "on-request"`, writable root `/tmp/rlv/ledger`; a 0600 copy of
  `auth.json`, removed with `/tmp/rlv`)
- `/tmp/rlv/proj`: `git init` with only `ralph.toml` and `.agents/skills/` of
  the branch (`git archive HEAD .agents/skills`)

```toml
[org]
model_pool = [{ driver = "codex", model = "gpt-6-astra" }]

[org.watchdog]
interval_seconds = 10

[org.permissions]
default = "guarded"

[org.permissions.roles]
director = "guarded"

[org.director]
enabled = true
wake = ["pull"]
```

```
$ ralph org watch --all --detach …
started ralph org watch --all (pid 99467); log: /tmp/rlv/ledger/watch.log
rc=0
```

## Run (a): the agent ends with `/quit`

```
$ ralph org director start --driver codex --model gpt-6-astra …
started d1 (driver=codex, model=gpt-6-astra, permission=guarded, pane=<pane-d1>, workspace=w1)
talk to the director in herdr: pane <pane-d1>, in the tab director_<token> of the workspace labelled ralph_director_<token>
stop it with: ralph org director stop --id d1 --state-dir '/tmp/rlv/ledger'
rc=0
```

The `attached` line carried the start process: `"starter_pid":<pid>,
"starter_start_time":"Sun Oct 18 18:04:58 2026"` (ps's `lstart`, read with
`LC_ALL=C` and `TZ=UTC`).

codex first showed its folder trust dialog: the trusted path in the fake
config was `/tmp/rlv/proj`, and codex ran in `/private/tmp/rlv/proj`. Enter
was sent to the pane (`herdr pane send-keys <pane-d1> Enter`) to accept it.
The watch's next cycle, while codex was in that dialog and then in its first
turn:

```
$ ralph status --state-dir /tmp/rlv/ledger
…
director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane <pane-d1>, workspace w1, attached at 2026-10-18T18:04:59Z) -- checked by the watch at 2026-10-18T18:05:25Z: alive (agent_found); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
$ cat /tmp/rlv/ledger/director-alive.json
{
  "id": "d1",
  "attached_at": "2026-10-18T18:04:59Z",
  "state": "alive",
  "how": "agent_found",
  "checked_at": "2026-10-18T18:05:25Z",
  "detail": ""
}
```

codex's first turn read its role prompt and the director skill, ran
`ralph org inbox`, and left a background terminal running (`1 background
terminal running`, the `ralph org wait --inbox` of pull; see Findings). At
18:06:20 `/quit` was typed into the pane (`herdr pane send-text`, then Enter).
codex printed its `codex resume …` line and the pane fell back to its zsh
prompt. herdr then said:

```
$ herdr agent list
{"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}
$ herdr pane get <pane-d1>
{"id":"cli:pane:get","result":{"pane":{"agent_status":"unknown",…,"pane_id":"<pane-d1>",…
```

The watch's next cycle:

```
$ ralph status --state-dir /tmp/rlv/ledger
…
director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane <pane-d1>, workspace w1, attached at 2026-10-18T18:04:59Z) -- checked by the watch at 2026-10-18T18:06:46Z: ended (agent_ended); as it has ended, ralph org director start --driver <claude|codex> --state-dir '/tmp/rlv/ledger' cleans it up and then starts a new headless director, or ralph org director stop --id d1 --state-dir '/tmp/rlv/ledger' stops it; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
```

`ralph org director status` printed the same `director:` line, then the
registration's fields (`state: running`, `launch: herdr`, …). The mark read
`"state": "ended", "how": "agent_ended", "checked_at":
"2026-10-18T18:06:46Z"`.

An attach without `--replace` was refused with the watch's check in the text
(D7), and wrote nothing:

```
$ ralph org director attach --driver codex …
org: director: attach: a director is already attached: d1 (driver codex, attached 2026-10-18T18:04:59Z, headless running): stop it with ralph org director stop --id d1 --state-dir '/tmp/rlv/ledger' from a terminal outside herdr, which closes its herdr pane, or replace it with ralph org director attach --driver codex --replace --state-dir '/tmp/rlv/ledger', which leaves that pane open until such a stop; the watch checked d1 at 2026-10-18T18:06:56Z: ended (agent_ended); as it has ended, ralph org director start --driver <claude|codex> --state-dir '/tmp/rlv/ledger' cleans it up and then starts a new headless director (it checks d1 again first)
rc=1
```

The next start cleaned d1 up and started d2:

```
$ ralph org director start --driver codex --model gpt-6-astra …
cleaned up d1, which had ended (agent_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped
started d2 (driver=codex, model=gpt-6-astra, permission=guarded, pane=<pane-d2>, workspace=w2)
talk to the director in herdr: pane <pane-d2>, in the tab director_<token> of the workspace labelled ralph_director_<token>
stop it with: ralph org director stop --id d2 --state-dir '/tmp/rlv/ledger'
rc=0
$ herdr workspace list
… one workspace, w2 (labelled ralph_director_<token>): w1 was closed
```

d2 was then stopped by hand for run (b): `ralph org director stop --id d2`
rc 0 (`stopped d2 (ctrl_c=ok, pane=closed, workspace=closed)`).

## Run (b): start killed while its agent starts

`ralph org director start` was run in the background. As soon as
`director.jsonl` had d3's `pane_created` (and no `launched`), the start
process was killed with SIGKILL (exit 137, at 18:08:29). No `herdr agent
start` process was left, and `herdr agent list` stayed empty: the kill came
before the agent was started in the pane.

```
2026-10-18T18:08:28Z attached d3
2026-10-18T18:08:28Z workspace_created d3
2026-10-18T18:08:28Z pane_created d3
```

The watch's next cycle (18:08:38), and the one after:

```
$ ralph status --state-dir /tmp/rlv/ledger
…
director: attached d3 (driver=codex, headless starting, model=gpt-6-astra, pane <pane-d3>, workspace w3, attached at 2026-10-18T18:08:28Z) -- checked by the watch at 2026-10-18T18:08:48Z: ended (start_ended); as it has ended, ralph org director start --driver <claude|codex> --state-dir '/tmp/rlv/ledger' cleans it up and then starts a new headless director, or ralph org director stop --id d3 --state-dir '/tmp/rlv/ledger' stops it; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
```

The next start cleaned d3 up and started d4:

```
$ ralph org director start --driver codex --model gpt-6-astra …
cleaned up d3, which had ended (start_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped
started d4 (driver=codex, model=gpt-6-astra, permission=guarded, pane=<pane-d4>, workspace=w4)
…
rc=0
$ herdr workspace list
… one workspace, w4: w3 was closed
```

## A start against a live director

After the watch's next cycle had read d4 as `alive (agent_found)`, one more
start was refused with its own check in the text, and wrote nothing (the last
line of `director.jsonl` stayed d4's `launched`):

```
$ ralph org director start --driver codex --model gpt-6-astra …
org: director: start: a director is already attached: d4 (driver codex, attached 2026-10-18T18:09:07Z, headless running): stop it with ralph org director stop --id d4 --state-dir '/tmp/rlv/ledger' from a terminal outside herdr, then run start again (start has no --replace); start checked d4 just now: alive (agent_found)
rc=1
```

## The ledger after the runs

`director.jsonl` (the start pids left out):

```
2026-10-18T18:04:59Z attached d1 launch=herdr starter_pid=<pid> starter_start_time=Sun Oct 18 18:04:58 2026
2026-10-18T18:04:59Z workspace_created d1
2026-10-18T18:04:59Z pane_created d1
2026-10-18T18:05:03Z launched d1
2026-10-18T18:07:08Z detached d1
2026-10-18T18:07:08Z stopped d1
2026-10-18T18:07:08Z attached d2 launch=herdr starter_pid=<pid> starter_start_time=Sun Oct 18 18:07:06 2026
2026-10-18T18:07:08Z workspace_created d2
2026-10-18T18:07:08Z pane_created d2
2026-10-18T18:07:12Z launched d2
2026-10-18T18:08:13Z detached d2
2026-10-18T18:08:13Z stopped d2
2026-10-18T18:08:28Z attached d3 launch=herdr starter_pid=<pid> starter_start_time=Sun Oct 18 18:08:27 2026
2026-10-18T18:08:28Z workspace_created d3
2026-10-18T18:08:28Z pane_created d3
2026-10-18T18:09:07Z detached d3
2026-10-18T18:09:07Z stopped d3
2026-10-18T18:09:07Z attached d4 launch=herdr starter_pid=<pid> starter_start_time=Sun Oct 18 18:09:06 2026
2026-10-18T18:09:07Z workspace_created d4
2026-10-18T18:09:07Z pane_created d4
2026-10-18T18:09:10Z launched d4
2026-10-18T18:09:45Z detached d4
2026-10-18T18:09:45Z stopped d4
```

The cleanups of d1 and d3 wrote the same two lines as the stops of d2 and d4
(`detached`, then `stopped`), each in the same second as the next director's
`attached`. The watch wrote nothing to `director.jsonl`. `watch.log` held only
its start line (`watching every org (interval=10s once=false
state-dir=/tmp/rlv/ledger (source: flag))`): no check failed. With `wake =
["pull"]` the wake sends nothing, so its skip of an ended director (D5) was
not exercised here; `internal/org/director_wake_test.go` covers it.
`/tmp/rlv/osascript.log` was never created.

## Cleanup

- `ralph org director stop --id d4` rc 0 (`ctrl_c=ok, pane=closed,
  workspace=closed`, `the director is no longer attached`), `ralph org watch
  --stop` rc 0 (`stopped ralph org watch --all (pid 99467)`), `herdr
  workspace list` empty, `herdr server stop` rc 0
- Still running afterwards: the codex app-server daemon from
  `/tmp/rlv/home/.codex/packages/app-server-daemon/releases/0.160.1-aarch64-apple-darwin/bin/`
  (`codex app-server --listen unix:// --managed-daemon` and `codex app-server
  daemon pid-update-loop`, parent pid 1, started 18:05:05 with d1;
  `codex-code-mode-host`, its child), and `ralph org wait --inbox
  --timeout-ms 300000 --state-dir /tmp/rlv/ledger --config
  /tmp/rlv/proj/ralph.toml`, a child of the daemon started at 18:09:28 by
  d4's first turn. All four were stopped by pid, `/tmp/rlv` (the `auth.json`
  copy included) was removed, and afterwards no process named `/tmp/rlv`
- The user's herdr state, read only: `~/.config/herdr/session.json` sha256
  prefix 004173699fb3c932 (mtime Sep 19 14:09:37) and
  `~/.config/herdr/herdr-server.log` sha256 prefix fe019d6d40e422ca (mtime
  Oct 1 14:35:45), the same values before and after

## Findings

- After `/quit`, herdr 0.7.5 had no agent for the director (`agent list`
  empty) and kept the pane with `agent_status: unknown`; the watch's next
  cycle read d1 as `ended (agent_ended)`, as in the probe of the plan's
  Assumptions. While codex was in its trust dialog and in its turns, the
  check read `alive (agent_found)`.
- A start killed after `pane_created` and before `launched` left d3
  `starting` with its starter recorded; the watch read it as `ended
  (start_ended)` in its first cycle after the kill.
- In both runs the next `ralph org director start` cleaned up the ended
  registration (C-c, then the pane and the workspace closed, then `detached`
  and `stopped`), said so in one line, and started the next director. Against
  a live one it refused and said what it had checked.
- The codex app-server daemon keeps the background terminals of a codex
  director's turns: `ralph org wait --inbox` from d4's turn kept running after
  d4 was stopped, its pane closed, and the herdr server stopped. The wait only
  reads the inbox and ends at its own timeout (5 minutes here), but neither
  stop nor start's cleanup ends it. The same daemon outlived the TUI in the
  probe and in `director-wake-livefire-2026-10-18.md`.

## What this run does not cover

- A launch_cmd director (`process_ended`, `pid_reused`, the refused late
  check-in): the unit and CLI tests cover them with processes the tests start
- A Claude Code director
- `wake = ["push"]` with an ended director (the wake's skip, D5)
- Two starts at once (the version check of D3 and D6): the tests in
  `internal/org/director_start_cleanup_test.go` fix the order with seams
- unknown results (`agent get` failing, a pane whose label differs, ps
  failing): the tests in `internal/org/director_check_test.go` cover them with
  a fake herdr and a fake ps
