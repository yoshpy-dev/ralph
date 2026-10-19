# Experiment: what happens to a task started through the Claude Code Monitor tool when its session ends? (stage 8b-2c, before the plan)

Plan: `docs/plans/active/2026-10-19-org-director-wait-bound.md`. Run on
2026-10-19, before the plan was written, to decide what stage 8b-2c does
about part B of stage 8b-2b: a director's `ralph org wait --inbox
--director` that does not stop when the session that started it ends (the
tech-debt row in `docs/tech-debt/README.md`). Stage 8b-2b's experiment
(`docs/evidence/director-wait-session-end-2026-10-19.md`) started its task
with the Bash tool's `run_in_background`; this one starts it the way the
`/director` skill names first, through the Monitor tool.

Values left out: session ids are `<id>`, and `$HOME` is `~`.

## Setup

- Claude Code 2.1.291, model haiku, `--allowedTools Bash Monitor ToolSearch`
  (the user's settings put the session in bypass permissions mode)
- An interactive claude in a pane of an isolated, headless herdr server
  (XDG_CONFIG_HOME, XDG_STATE_HOME, and HERDR_SOCKET_PATH under
  `/private/tmp/x9m`; the real HOME for claude's auth). No herdr server of
  the user was touched
- The task was `/bin/sh m.sh <dir>`, started with the Monitor tool. It writes
  its pid, its parent's pid, and a start time; traps SIGHUP, SIGINT, SIGTERM,
  and SIGPIPE (each trap writes a file named after the signal, with the time,
  and exits); prints `m-started`; polls once a second for the file
  `<dir>/go` (at most 1700 s); prints `m-go-seen`; and writes a finish time
- Only processes this experiment started were signalled

## Process chain while the session ran

- `/bin/sh m.sh` (S), whose parent is
- `/bin/zsh -c source ~/.claude/shell-snapshots/snapshot-zsh-….sh …`
  (Ss: its own session and process group), whose parent is
- the claude TUI process (S+, the foreground of the pane), whose parent is
  the pane's login zsh, whose parent is the herdr server

The same chain as a Bash `run_in_background` task in stage 8b-2b's
experiment. The task's stdin is `/dev/null`; its stdout and stderr are unix
sockets whose other ends the claude process holds (the `zsh -c` shell holds
the same two sockets).

## Runs

| Run | How the session ended | The task afterwards |
|-----|-----------------------|---------------------|
| m1 | `/exit`, option 1 "Exit and stop tasks" (the default) | got SIGTERM and was gone within 1 s, with the claude process and the `zsh -c` shell |
| m2 | `/exit`, option 2 "Move to background and exit" | got SIGTERM; claude printed "1 monitor couldn't be moved and was stopped." |
| m3 | `kill -KILL <claude pid>`, pane left open | still running; its `zsh -c` shell reparented to pid 1; its stdout and stderr became sockets with no other end |
| m4 | the Monitor's own `timeout_ms` (20000) | got SIGTERM 20 s after it started; the session was told "Monitor expired after 20s with 1 e…" |
| m5 | `herdr pane close` (the pane's processes get SIGHUP) | got SIGTERM and was gone within 1 s, with the claude process |

Notes on the runs:

- m1: `/exit` with the task running showed "Background work is running /
  The following will stop when you exit: monitor · m1 script", with
  "1. Exit and stop tasks" (default), "2. Move to background and exit", and
  "3. Stay".
- m2: claude printed "backgrounded · <id>" with `claude agents`,
  `claude attach <id>`, `claude logs <id>`, and `claude stop <id>`.
  `claude logs <id>` showed the backgrounded session with the line
  "Background shell command didn't finish before the previous session
  ended". `claude stop <id>` printed "stopped <id>", and no `bg-spare`
  process with a cwd under the experiment's directory was left.
- m3: in the task, the parent stayed the `zsh -c` shell; the shell's
  parent became pid 1. After the file `go` was created, the task printed its
  line to the socket with no other end, its SIGPIPE trap did not run, and it
  ran to its end. Probably SIGPIPE was ignored when it started, inherited
  from claude's node process (not confirmed).
- m4: the session replied "Monitor expired." and did nothing else.
- Control (the Bash tool's `run_in_background`, from the session that ran
  this experiment, same Claude Code version): the same script started at
  11:41:37Z, was told to finish by the file `go` at 11:53:56Z, and ended
  on its own with exit code 0. It was not stopped at 10 minutes, the limit
  of a foreground Bash command.

## Reading

- A Monitor task is stopped with SIGTERM in every ending claude handles:
  the default exit, "Move to background and exit" (a Bash
  `run_in_background` task survived that in stage 8b-2b's experiment, b2),
  a pane close, and the Monitor's own expiry. Only a SIGKILL of claude (a
  crash) leaves it running.
- The Monitor tool ends every monitor at its `timeout_ms` (default 5
  minutes, at most 30 minutes) with SIGTERM. At 3a803a87, before stage
  8b-2c, the `/director` skill told a Claude Code director to run
  `ralph org wait --inbox --director d1 --timeout-ms 0 2>&1` under Monitor
  or with Bash `run_in_background`, and counted an end by a signal as a
  failure not to run again; a director that picked Monitor therefore
  stopped waiting for the inbox within 30 minutes and reported a failure.
- A task that outlives its session after a SIGKILL keeps its own parent,
  the `zsh -c` shell, which is reparented to pid 1. The wait's parent pid
  does not change; telling this case apart would need the grandparent, and
  how many levels lie between the wait and claude depends on how the shell
  wraps the command.
- Not checked here: a real `ralph org wait` under Monitor (this used a shell
  script), whether Go's runtime ends on SIGPIPE when it writes to a socket
  with no other end, whether the Bash tool's `run_in_background` has a time
  limit longer than the control's 12 minutes, and codex (whose background
  wait's parent is its app-server daemon, as stage 8b-2a's live check saw).

Left on the machine: claude's own records of these sessions
(`~/.claude/projects/-private-tmp-x9m-proj/`, the trust entry for
`/private/tmp/x9m/proj`). `/private/tmp/x9m` was removed after the run.
