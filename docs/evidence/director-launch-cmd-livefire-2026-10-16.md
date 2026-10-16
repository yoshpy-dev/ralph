# Live run: `[org.director].launch_cmd` with tmux (2026-10-16)

Evidence for AC10 of plan 2026-10-16-org-director-launch-cmd (stage 8a-2b).
Real `claude` and `codex` directors were started and stopped through a tmux
`launch_cmd`. The tmux server ran on its own socket (`tmux -L ralph-lf-8a2b`)
and the ledger was an isolated state dir, so the human's tmux, herdr, and repo
ledger were not touched. No herdr command and no osascript ran.

`<scratch>` is the session scratchpad. `~` is `$HOME`. `<user>` is `$USER`.
`<token-dN>` is the launch token of registration dN: twelve lowercase hex
digits that name one launch and grant nothing, written as a placeholder so the
page holds no `--token` value a secret scanner would report.

## Setup

- ralph: built from branch `feat/org-director-launch-cmd` at `88bb29fd`
  (`git archive` into `<scratch>/lf8a2b/src`, `go build -o <scratch>/lf8a2b/bin/ralph ./cmd/ralph`)
- tmux 3.5a, Claude Code 2.1.291, codex-cli 0.160.1 (both through mise shims,
  `~/.local/share/mise/shims/claude` and `.../codex`), macOS (Darwin 25.5.0)
- `<scratch>/lf8a2b/proj/ralph.toml` (not a git checkout, so the director
  runs in the directory start runs in):

```toml
[org]
agmsg_home = "<scratch>/lf8a2b/agmsg"

[org.director]
enabled = true
launch_cmd = ["tmux", "-L", "ralph-lf-8a2b", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}"]

[org.permissions.roles]
director = "guarded"
```

Every command ran with `--state-dir <scratch>/lf8a2b/state`, and start also
ran with `--config <scratch>/lf8a2b/proj/ralph.toml` and the built ralph first
on `PATH`.

## Run 1: claude, stopped at its startup dialog

```
$ ralph org director start --driver claude --model sonnet --state-dir … --config …
started d1 (driver=claude, model=sonnet, permission=guarded, launch=cmd, name=director_<token-d1>, pid=78605)
talk to the director in the terminal that [org.director].launch_cmd opened (name director_<token-d1>)
stop it with: ralph org director stop --id d1 --state-dir '<scratch>/lf8a2b/state'
(stderr) ralph org director start: started the cross-org watch (pid 78595); log: <scratch>/lf8a2b/state/watch.log
rc=0
```

The launch script `<state>/launch/director_<token-d1>.sh` (mode 0700):

```sh
#!/bin/sh
# The launch script of the headless director d1 (director_<token-d1>), written by ralph org director start.
# It checks in with ralph, which records its pid, and only then execs the agent in its place.
cd '<scratch>/lf8a2b/proj' || exit 1
RALPH_DIRECTOR_TOKEN='<token-d1>'
export RALPH_DIRECTOR_TOKEN
'<scratch>/lf8a2b/bin/ralph' org director launched --id 'd1' --token '<token-d1>' --pid "$$" --state-dir '<scratch>/lf8a2b/state' || exit 1
exec '~/.local/share/mise/shims/claude' '--model' 'sonnet' '役割指示を読み込んで従ってください: <scratch>/lf8a2b/state/prompts/director_<token-d1>.md'
```

(`~` stands for the absolute home path the script holds.) `RALPH_ORG_AGMSG_HOME`
was not set in start's environment, so the script exports none.

The process the check-in recorded. The mise shim execs the real binary, so the
pid and the start time are the ones the check-in read:

```
$ LC_ALL=C TZ=UTC ps -o pid,ppid,stat,lstart,command -p 78605
  PID  PPID STAT STARTED                      COMMAND
78605 78601 Ss+  Fri Oct 16 00:44:39 2026     ~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/…/@anthropic-ai/…
```

The tmux pane showed claude's "Is this a project you … trust?" dialog.

Status:

```
$ ralph org director status --state-dir … --config …
director: attached d1 (driver=claude, headless running, model=sonnet, launch cmd, name director_<token-d1>, pid 78605, attached at 2026-10-16T00:44:39Z, by <user>) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline
enabled: true
id: d1
…
launch: cmd
state: running
model: sonnet
permission_mode: guarded
token: <token-d1>
pid: 78605
start_time: Fri Oct 16 00:44:39 2026
terminal_name: director_<token-d1>

$ ralph org director status … --json   (the registration)
{"id": "d1", "driver": "claude", "wake": ["pull"], "launch": "cmd", "state": "running", "token": "<token-d1>", "model": "sonnet", "permission_mode": "guarded", "pid": 78605, "start_time": "Fri Oct 16 00:44:39 2026", "attached_at": "2026-10-16T00:44:39Z", "user": "<user>", "cwd": "<scratch>/lf8a2b/proj", "herdr_pane": false, "terminal_name": "director_<token-d1>"}

$ ralph status --state-dir … | grep '^director:'
director: attached d1 (driver=claude, headless running, model=sonnet, launch cmd, name director_<token-d1>, pid 78605, attached at 2026-10-16T00:44:39Z, by <user>) -- …
```

The self-stop refusal (D8). The token is put in the environment by hand,
as the launch script does for the director:

```
$ RALPH_DIRECTOR_TOKEN=<token-d1> ralph org director stop --id d1 --state-dir …
org: director: stop d1: RALPH_DIRECTOR_TOKEN is its launch token, so this runs inside the director, whose launch script exported it, and stopping the director's process would end this command before it records the stop: run ralph org director stop from a terminal other than the director's
rc=1
```

director.jsonl still had 2 lines and pid 78605 was still running (`Ss+`).

The stop:

```
$ ralph org director stop --id d1 --state-dir …
stopped d1 (process=ended after SIGTERM)
the director is no longer attached
the terminal that [org.director].launch_cmd opened (name director_<token-d1>) is left open for you to close
rc=0
$ ps -o pid=,stat= -p 78605; echo $?
1
$ tmux -L ralph-lf-8a2b ls
no server running on /private/tmp/tmux-502/ralph-lf-8a2b
```

The tmux session's only pane ran the launch script, which had become claude,
so the session and the server closed by themselves once claude ended.

## Run 2: claude in an active session

`start` again (d2, pid 82897). `tmux send-keys` answered claude's trust dialog
("Yes, I trust this folder"), and claude read its role prompt and ran its
first commands (see "Finding" below). Then:

```
$ ralph org director stop --id d2 --state-dir …
stopped d2 (process=ended after SIGTERM)
the director is no longer attached
the terminal that [org.director].launch_cmd opened (name director_<token-d2>) is left open for you to close
rc=0
```

pid 82897 was gone. No process mentioning `<state>/prompts` was left
(`pgrep -fl` printed nothing).

## Run 3: codex in an active session, with RALPH_ORG_AGMSG_HOME

```
$ RALPH_ORG_AGMSG_HOME=<scratch>/lf8a2b/agmsg ralph org director start --driver codex --state-dir … --config …
org: --model omitted; falling back to first [org].model_pool entry permitted for role director on codex: gpt-6-astra (pass --model explicitly)
started d3 (driver=codex, model=gpt-6-astra, permission=guarded, launch=cmd, name=director_<token-d3>, pid=90429)
talk to the director in the terminal that [org.director].launch_cmd opened (name director_<token-d3>)
stop it with: ralph org director stop --id d3 --state-dir '<scratch>/lf8a2b/state'
rc=0
```

This launch script exports the variable (lines 7 and 8):

```sh
RALPH_ORG_AGMSG_HOME='<scratch>/lf8a2b/agmsg'
export RALPH_ORG_AGMSG_HOME
exec '~/.local/share/mise/shims/codex' '--model' 'gpt-6-astra' …
```

pid 90429 is codex's npm entry point (`node`, `@openai/codex/bin/codex.js`).
It ran the native codex as its child, pid 90530. The wrapper forwards
SIGINT, SIGTERM, and SIGHUP to the child and waits for it (`forwardSignal`
in `bin/codex.js`).

codex first showed a dialog about its background server's feature settings.
"1. Run without daemon this time" was chosen, since it changes no persistent
setting. codex then read its role prompt and finished its first turn. Then:

```
$ ralph org director stop --id d3 --state-dir …
stopped d3 (process=ended after SIGTERM)
the director is no longer attached
the terminal that [org.director].launch_cmd opened (name director_<token-d3>) is left open for you to close
rc=0
$ ps -o pid= -p 90429 90530; echo $?
1
```

Both the wrapper and the native child ended.

## Run 4: a launch_cmd that never runs the script

`<scratch>/lf8a2b/proj3/ralph.toml` was the same file with
`launch_cmd = ["sh", "-c", "exit 0", "sh", "{script}"]`.

```
$ ralph org director start --driver claude --model sonnet --timeout-ms 3000 --state-dir … --config <scratch>/lf8a2b/proj3/ralph.toml
org: director: start d4: the launch script did not check in before the timeout: launch_cmd returned, but the launch script <scratch>/lf8a2b/state/launch/director_<token-d4>.sh did not check in within 3s (--timeout-ms 3000): no terminal ran it, or its check-in was refused or failed; d4 is recorded stopped; the terminal that [org.director].launch_cmd may have opened (name director_<token-d4>) is left open for you to close, and a launch script that checks in from it now is refused and ends without starting the agent
rc=1
```

The launch script, run by hand afterwards as a terminal that opened late would:

```
$ sh <scratch>/lf8a2b/state/launch/director_<token-d4>.sh
org: director: check in d4: the check-in of the launch script is refused, so its agent must not start: d4 (driver claude, attached 2026-10-16T00:48:48Z, headless stopped, launch cmd) is already recorded stopped (start's cleanup or a stop came first)
rc=1
```

director.jsonl had 15 lines before and after, and no agent started
(`pgrep -fl '<state>/prompts'` printed nothing).

## Run 5: a launch_cmd that fails

`<scratch>/lf8a2b/proj4/ralph.toml` used
`launch_cmd = ["sh", "-c", "echo no terminal here >&2; exit 3", "sh", "{script}"]`.

```
$ ralph org director start --driver codex --state-dir … --config <scratch>/lf8a2b/proj4/ralph.toml
org: --model omitted; falling back to first [org].model_pool entry permitted for role director on codex: gpt-6-astra (pass --model explicitly)
org: director: start d5: launch_cmd sh failed: exit status 3; its output ended with: no terminal here; d5 is recorded stopped; the terminal that [org.director].launch_cmd may have opened (name director_<token-d5>) is left open for you to close, and a launch script that checks in from it now is refused and ends without starting the agent
rc=1
```

## The ledger after the five runs

```
{"ts":"2026-10-16T00:44:39Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"launch":"cmd","token":"<token-d1>","model":"sonnet","permission_mode":"guarded","user":"<user>","cwd":"<scratch>/lf8a2b/proj","herdr_pane":false}
{"ts":"2026-10-16T00:44:39Z","id":"d1","event":"launched","pid":78605,"start_time":"Fri Oct 16 00:44:39 2026","user":"<user>","cwd":"<scratch>/lf8a2b/proj","herdr_pane":false}
{"ts":"2026-10-16T00:45:12Z","id":"d1","event":"detached",…}
{"ts":"2026-10-16T00:45:12Z","id":"d1","event":"stopped",…}
{"ts":"2026-10-16T00:45:25Z","id":"d2","event":"attached",…,"launch":"cmd","token":"<token-d2>",…}
{"ts":"2026-10-16T00:45:25Z","id":"d2","event":"launched","pid":82897,"start_time":"Fri Oct 16 00:45:25 2026",…}
{"ts":"2026-10-16T00:46:40Z","id":"d2","event":"detached",…}
{"ts":"2026-10-16T00:46:40Z","id":"d2","event":"stopped",…}
{"ts":"2026-10-16T00:46:48Z","id":"d3","event":"attached","driver":"codex",…,"launch":"cmd","token":"<token-d3>","model":"gpt-6-astra",…}
{"ts":"2026-10-16T00:46:48Z","id":"d3","event":"launched","pid":90429,"start_time":"Fri Oct 16 00:46:48 2026",…}
{"ts":"2026-10-16T00:48:36Z","id":"d3","event":"detached",…}
{"ts":"2026-10-16T00:48:36Z","id":"d3","event":"stopped",…}
{"ts":"2026-10-16T00:48:48Z","id":"d4","event":"attached",…,"token":"<token-d4>",…}
{"ts":"2026-10-16T00:48:51Z","id":"d4","event":"detached",…}
{"ts":"2026-10-16T00:48:51Z","id":"d4","event":"stopped",…}
{"ts":"2026-10-16T00:49:07Z","id":"d5","event":"attached","driver":"codex",…,"token":"<token-d5>",…}
{"ts":"2026-10-16T00:49:07Z","id":"d5","event":"detached",…}
{"ts":"2026-10-16T00:49:07Z","id":"d5","event":"stopped",…}
```

Nothing was written to the manifest (the state dir has no `manifest.jsonl`).

## Cleanup

```
$ ralph org watch --stop --state-dir …
stopped ralph org watch --all (pid 78595)
rc=0
```

Afterwards nothing whose command line mentions `<scratch>/lf8a2b` was
running (`pgrep -fl` printed nothing), and the tmux socket had no server.

## Finding: the director's own `ralph` comes from its shell's PATH

In runs 2 and 3, the director's first `ralph org director status` answered
with the `ralph org` help: the `ralph` it ran had no `director` verb. start ran
with the development build first on `PATH`. The director, though, runs its
commands through the human's shell, and the human's zsh startup files put
`/opt/homebrew/bin` first even for a non-interactive `zsh -c` (`whence -a
ralph` printed only `/opt/homebrew/bin/ralph`, which is ralph 5.1.0 and has no
`director` verb). Following its role prompt,
each director stopped acting and said so in its terminal. The ralph that
writes the check-in is not affected, because the launch script names it by
its absolute path. This is the same gap as the seats' `ralph` of another
version on the pane's PATH (`docs/tech-debt/README.md`), and it affects the
herdr launch too. It shows up here only because a development build ran
start; with the installed ralph on both sides it does not arise.

## What this run does not cover

- Orca and WezTerm (the recipe's examples come from their `--help`)
- A process that outlives its SIGTERM, ps that cannot be read, and the
  check-in that lands while stop or start's cleanup runs. The unit and CLI
  tests cover these with processes the tests start themselves (S2b and S3
  mutation runs).
- `ralph org stop --all`, which was not run, so that nothing could reach the
  human's herdr. The CLI tests cover its note about launch_cmd directors.
