# Headless director in your own terminal: `[org.director].launch_cmd`

`ralph org director start` launches a headless director, a claude or codex
session that runs the `/director` skill. With `[org.director].launch_cmd`
empty (the default) it opens a herdr workspace of its own. Set
`launch_cmd`, and start makes no herdr call: it runs your command to open a
terminal (tmux, Orca, WezTerm, ...), and the director runs there. This
recipe covers that second way.

## What launch_cmd does

`launch_cmd` is an array of arguments. start runs its first element with
the others as arguments, without a shell, after replacing these
placeholders in every element:

| Placeholder | Replaced by |
| --- | --- |
| `{script}` | the absolute path of the launch script start writes, `<state-dir>/launch/director_<token>.sh` |
| `{script_sh}` | the same path in single quotes, for a terminal that reads its command as a shell string |
| `{name}` | `director_<token>`, a name for the terminal (`<token>` is 12 hex digits, new for every start) |
| `{cwd}` | the directory the director runs in: the main worktree root when the ledger is the main worktree's, else the directory start runs in |

Use `{script}` when the terminal takes the command as an argv (WezTerm's
`cli spawn -- sh {script}`). Use `{script_sh}` when it hands the command to
a shell as one string (tmux's shell-command, Orca's `--command`): the
quotes keep a path with spaces or quotes in one word.

The array must contain `{script}` or `{script_sh}`, since the director
starts only when the terminal runs the launch script. A placeholder is a
`{` and the next `}` with no other brace between them; any other one (a
shell brace group such as `{ a; }` included), and a first element that is
empty or whitespace only, make `ralph.toml` fail to load with an error that
names `[org.director].launch_cmd`. A `{` or `}` outside such a pair is
passed as written.

The command must return once the terminal is open. start runs it in
`{cwd}`, with start's environment and an empty stdin, and stops it after
10 s: a command that stays attached to the terminal it opened (an attached
`tmux new-session`, `wezterm start`) is killed and the start fails.

Without `--config`, start reads `launch_cmd` from the main worktree's
`ralph.toml` when the ledger is the main worktree's, as it reads
`[org.director].enabled`, `[org.permissions]`, `[org.roles]`, and
`model_pool`, so a feature worktree's `ralph.toml` cannot change the
command ralph runs. With `--config`, start reads that file; otherwise
`./ralph.toml`.

## Examples

One example per terminal. Put one of them in `[org.director]`.

tmux (its shell-command is a shell string, hence `{script_sh}`; `-L ralph`
keeps the director on a tmux server of its own):

```toml
[org.director]
launch_cmd = ["tmux", "-L", "ralph", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}"]
```

Attach to it with `tmux -L ralph attach -t director_<token>`.

Orca (it types `--command` into the terminal's login shell, hence
`{script_sh}`; the `--worktree` selector must name a checkout Orca knows):

```toml
[org.director]
launch_cmd = ["orca", "terminal", "create", "--worktree", "path:{cwd}", "--title", "{name}", "--command", "sh {script_sh}"]
```

WezTerm (the program after `--` is an argv, hence `{script}`; `wezterm cli`
needs a running WezTerm GUI):

```toml
[org.director]
launch_cmd = ["wezterm", "cli", "spawn", "--new-window", "--cwd", "{cwd}", "--", "sh", "{script}"]
```

## Start

```sh
ralph org director start --driver claude --model opus
```

start first applies the same refusals as a herdr start, writing nothing and
running no launch_cmd: `[org.director].enabled = false`, a codex director
in anything but `guarded`, a `--config` whose `[org].agmsg_home` is a
relative path, a promoted session already attached, a headless director
already attached that the check below finds alive or cannot tell about,
and a watch that is stale, stopping, or a `--once` run (start launches
`ralph org watch --all` when none runs). It also refuses when the
`--driver` agent (`claude` or `codex`) is not on its own `PATH`, and when
`[org.director].wake` has push and `[org.director].wake_cmd` is empty, since
nothing could then type into the terminal launch_cmd opens (see
`docs/recipes/director-wake.md`).
`HERDR_PANE_ID` does not refuse it: you may run start from inside herdr.

When the current registration is a headless director, one still
`starting` included, start checks on the spot, before it looks at the
watch, whether that director is still running, the way
`ralph org watch --all` checks it each cycle: one still `starting` by the
start process that recorded it, one running in herdr with
`herdr agent get` and its pane, and one started with `launch_cmd` by the
pid its launch script checked in with and that pid's start time.

- `alive` or `unknown`: start is refused, writing nothing, with what the
  check found and the `ralph org director stop --id <id>` that stops it.
- `ended`: start cleans it up as stop would, but only what the registration
  recorded when the check read it. For a `launch_cmd` director it sends
  nothing to the process that has ended and leaves the terminal for you to
  close. Once it has recorded the director stopped, it prints one line such
  as `cleaned up d1, which had ended (process_ended): ...` and goes on; the
  line stays even when start is then refused for another reason.
- A cleanup that cannot finish (a close that fails, a registration that
  changed after the check) records nothing, starts no director, exits 1,
  and names the `ralph org director stop --id <id>` that finishes it.

Then, in this order:

1. start writes `attached` (with `launch` `cmd`, the token, the model, and
   the permission mode) to `<state-dir>/director.jsonl`.
2. It writes the role prompt to `<state-dir>/prompts/director_<token>.md`
   and the launch script to `<state-dir>/launch/director_<token>.sh`
   (mode 0700, in a 0700 directory).
3. It runs `launch_cmd`, for up to 10 s.
4. It waits for the launch script's check-in, for up to `--timeout-ms`
   (default 60000).

The launch script, run by the terminal, single-quotes every value and:

1. goes to `{cwd}` (it exits 1 if it cannot);
2. exports `RALPH_DIRECTOR_TOKEN`, and `RALPH_ORG_AGMSG_HOME` when it was
   set where start ran (a relative value made absolute from there);
3. checks in with `<ralph> org director launched --id <id> --token <token>
   --pid "$$" --state-dir <ledger>`, where `<ralph>` is the absolute path of
   the ralph that ran start, and exits 1 if the check-in fails;
4. execs the agent at the absolute path start found on its `PATH`, with the
   permission flags, `--model`, and the pointer to the prompt file, as a
   herdr start gives them.

The check-in is accepted only for the current registration, still
`starting`, started with `launch_cmd`, with the same token, and with a
`--pid` equal to the check-in's parent process (the script itself), and
only while the `ralph org director start` that recorded the registration
still runs. start records its own pid and that pid's start time on
`attached` (`starter_pid` and `starter_start_time`); once that pid is
gone, a zombie, or has another start time, no start waits for the
check-in, so it is refused. When ps cannot read that pid, and when the
registration has no start process recorded (an older ralph wrote it, or
start could not read its own start time), the check-in is accepted. It
records `launched` with that pid and its start time (`ps -o lstart=`, read
with `LC_ALL=C TZ=UTC`). The exec keeps the pid and the start time, so the
recorded pid is the agent's from then on. A refused check-in writes
nothing, and the script exits without starting the agent.

On success start prints:

```
started d1 (driver=claude, model=opus, permission=autonomous, launch=cmd, name=director_<token>, pid=12345)
talk to the director in the terminal that [org.director].launch_cmd opened (name director_<token>)
stop it with: ralph org director stop --id d1
```

The stop command carries `--state-dir` when start was given one.

When `launch_cmd` cannot be run, exits non-zero, or runs out its 10 s,
when no check-in comes within `--timeout-ms`, or when the registration is
replaced (`attach --replace`) while start waits, start stops what checked
in, by its pid and start time as stop does, records the director stopped,
and exits 1. When it cannot confirm that the process ended, it records
`detached` only and prints the `ralph org director stop --id <id>` that
finishes the cleanup. Either way the terminal launch_cmd may have opened
is left for you to close; a launch script that checks in from it
afterwards is refused and starts no agent.

## Talk to the director

Talk to it in the terminal launch_cmd opened (for tmux, attach as above).
Its role prompt tells it that this terminal is where it talks, that you run
stop from another terminal, and that it never runs stop itself.
`ralph org director status`, the `director:` line of `ralph status`, and
their `--json` show a launch_cmd registration with `launch cmd`, its state
(`starting` or `running`), the terminal name, the pid, and the model (the
JSON field is `terminal_name`), and what `ralph org watch --all` found
when it last checked the director, such as
`checked by the watch at <time>: alive (process_found)` (`check` in the
JSON; the `/director` skill says how to read it).

By default the director waits on the inbox itself. To have
`ralph org watch --all` wake it when a new inbox item arrives, set
`wake = ["push"]` and a `wake_cmd` that types into this terminal, with
`{target}` replaced by its name `director_<token>`;
`docs/recipes/director-wake.md` pairs a tmux `wake_cmd` with the tmux
example above.

## Stop it

Run stop from a terminal other than the director's:

```sh
ralph org director stop --id d1
```

- stop runs no herdr. It reads the recorded pid with ps and sends SIGTERM to
  that pid alone, and only while its start time is the one the check-in
  recorded. That covers the launch script before its exec and the agent
  after it. It then waits up to 10 s for the process to end and records
  `stopped` (with `detached` when it is the current registration).
- A pid that is gone, a zombie, a pid with another start time (another
  process has it now), and a registration still starting with no check-in
  count as ended and are sent nothing. A check-in after that is refused.
- When the process outlives its SIGTERM, stop records nothing, exits 1, and
  prints `ps -o pid,lstart,command -p <pid>` to look at it. Run stop again
  once it has ended. When ps cannot be read (before the SIGTERM, nothing is
  sent; while stop waits after it, the SIGTERM has already been sent), stop
  records nothing and exits 1 too: run it again once ps can read the pid.
  Either way, `--force` records the stop instead. stop never sends SIGKILL.
- `--force` takes the same steps, so a process confirmed as this
  director's still gets its normal SIGTERM and nothing is sent when ps
  cannot be read before the SIGTERM. When the end is still not confirmed,
  it records a forced stop anyway, sends nothing more, and leaves the
  process as it is.
- stop is refused, `--force` included, when `RALPH_DIRECTOR_TOKEN` is the
  registration's token: it is then running inside the director, whose
  process would end before stop records it.
- When the launch script checks in after stop read the registration, stop
  has sent that process nothing, records nothing, and says to run it again.
- `detach` and `attach --replace` end the registration only: the process
  keeps running until a stop. `ralph org stop --all` leaves the director
  running and its `note:` line names the stop.

ralph never closes the terminal. With the tmux example, the session's only
pane runs the launch script, which becomes the agent, so the tmux session
closes by itself once stop ends the director. Other terminals may leave the
tab or window open; close it yourself.

If `claude` or `codex` on your `PATH` is a wrapper that starts the agent as
a child instead of exec-ing it, the wrapper must pass SIGTERM on to that
child. stop signals the recorded pid, which is the wrapper's; otherwise
stop ends the wrapper and the child may keep running. The npm codex wrapper
(`@openai/codex/bin/codex.js`) passes SIGTERM on. A shim that execs the
real binary, such as a mise shim, keeps the pid and the start time, so
nothing changes.

## Environment

- `RALPH_DIRECTOR_TOKEN`: the launch token, which stop compares to refuse a
  stop run from inside the director.
- `RALPH_ORG_AGMSG_HOME`: exported by the launch script when it was set
  where start ran, so the director's `ralph org` commands use the agmsg
  install it names. A relative value is made absolute from where start ran,
  as for the watch start launches, since the script first goes to `{cwd}`.
  A herdr director does not get it (its pane has the herdr server's
  environment), and start warns there; with `launch_cmd` it does not warn.

Apart from its directory and these two variables, the director's
environment is the terminal's (for the tmux example, the `-L ralph`
server's).

## Troubleshooting

- `did not check in within ... (--timeout-ms N)`: launch_cmd returned, but
  no launch script checked in. The terminal did not run it (a wrong
  command or worktree selector, or no running WezTerm GUI), it opened later
  than `--timeout-ms`, or the check-in was refused or failed. start has
  recorded the director stopped. Close any terminal it opened, run the
  command by hand with the placeholders filled in to see what it does, and
  start again, with a larger `--timeout-ms` for a slow terminal.
- A terminal that opened late: its launch script's check-in is refused
  (the registration is already stopped, or the start that recorded it has
  ended), so it exits 1 without starting the agent. Close the terminal.
- A start that died before the check-in (killed, or the shell it ran in
  closed) leaves the registration `starting`. Once the watch has checked
  it, `ralph status` shows it `ended (start_ended)`; a launch script that
  checks in after that is refused, and the next
  `ralph org director start` cleans the registration up and starts a new
  director.
- `launch_cmd ... could not be run`, `failed`, or `did not return in time`:
  the error names the command and the end of its output. Make the command
  return once the terminal is open.
- stop exits 1 with `ps -o pid,lstart,command -p <pid>`: the process did
  not end within 10 s of its SIGTERM. Look at the pid, end it yourself if
  needed, and run stop again, or record the stop with `--force`.
- stop exits 1 with `again once ps can read pid <pid>`: ps could not be
  read, so stop could not tell whether the process ended. The message holds
  ps's error. Run stop again once ps works, or record the stop with
  `--force`.
- `run ralph org director stop from a terminal other than the director's`:
  the shell you ran stop in has the director's `RALPH_DIRECTOR_TOKEN`.
  Run it from another terminal.
- The director's first command, `ralph org director status`, fails (for
  example, an older ralph has no `ralph org director`): the director's own
  `ralph org` and `ralph status` commands use the `ralph` first on the
  `PATH` of the agent's command shell (the agent runs its commands through
  your shell, whose startup files can set `PATH` again), which can be another
  version than the one that ran start. Make sure the `ralph` your shell's
  startup files put first on `PATH` is the one you run start with. The
  check-in is not affected: the script runs it with the absolute path of
  the ralph that ran start.

## Security

- ralph runs `launch_cmd` as an argv, with no shell of its own between. The
  placeholders take only values ralph makes (the script path,
  `director_<token>`, `{cwd}`); no inbox text, seat output, or other
  message goes into the command. `{script_sh}` is single-quoted for the
  shell you hand it to.
- The launch script single-quotes every value, and it and its directory are
  0700. It holds the absolute paths and the `RALPH_ORG_AGMSG_HOME` value, and
  start does not remove it after the director stops; a check-in from it
  later is refused.
- Without `--config`, when the ledger is the main worktree's, the command
  comes from the main worktree's `ralph.toml`, so a leader cannot change it
  by editing its feature worktree's `ralph.toml`.

## What was verified

The tmux example (tmux 3.5a) was run end to end with claude (Claude Code
2.1.291) and codex (codex-cli 0.160.1) on a tmux server of its own and an
isolated ledger: start, the check-in, status, the `RALPH_DIRECTOR_TOKEN`
refusal, and stop, plus two failure paths. A launch_cmd that never runs the
script timed out on the check-in and was recorded stopped, and the launch
script run by hand afterwards was refused with nothing written and no
agent; a launch_cmd that exits 3 was recorded stopped. Both agents ended on
SIGTERM. The evidence is
`docs/evidence/director-launch-cmd-livefire-2026-10-16.md`. The Orca (CLI
1.4.224) and WezTerm (20250424-220948-1439661d) examples come from their
`--help` and were not run.
