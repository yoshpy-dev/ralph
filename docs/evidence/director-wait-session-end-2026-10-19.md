# Experiment: does a Claude Code background task outlive the session that started it? (stage 8b-2b, before the plan)

Plan: `docs/plans/active/2026-10-19-org-director-wait.md` (Non-goals, D7,
AC9). Run on 2026-10-19, before the plan was written, to decide whether
stage 8b-2b could also make a director's `ralph org wait --inbox` stop when
the session that started it ends (part B of the consult, split off to a later
8b-2c). It is recorded here because D7 and the new tech-debt row rest on it.

Values left out: session ids are `<id>`, the daemon's socket path is
`/tmp/cc-daemon-502/<…>`, and `$HOME` is `~`.

## Setup

- Claude Code 2.1.291, model haiku, `--allowedTools Bash` (the user's
  settings put the session in bypass permissions mode)
- The background task was `/bin/sh bg.sh <dir>`: it writes its pid and a
  start time, then runs `sleep 1777`, then writes a finish time. The session
  started it with the Bash tool and `run_in_background: true`
- (a) ran `claude -p` in a scratch dir, with no herdr
- (b) ran an interactive claude in a pane of an isolated, headless herdr
  server (XDG_CONFIG_HOME, XDG_STATE_HOME, and HERDR_SOCKET_PATH under
  `/private/tmp/x8b`; the real HOME for claude's auth). No herdr server of
  the user was touched
- Only processes this experiment started were signalled

## Process chain while the session ran

The same in every interactive run:

- `/bin/sh bg.sh` (S), whose parent is
- `/bin/zsh -c source ~/.claude/shell-snapshots/snapshot-zsh-….sh … && eval …`
  (Ss: its own session and process group), whose parent is
- the claude TUI process (S+, the foreground of the pane), whose parent is
  the pane's login zsh, whose parent is the herdr server

## Runs

| Run | How the session ended | The task afterwards |
|-----|-----------------------|---------------------|
| a | `claude -p` exited | gone within 3 s; no finish file |
| b1 | `/exit`, option 1 "Exit and stop tasks" (the default, Enter) | the claude process, the task, and its sleep gone within 5 s; claude printed `claude --resume <id>` |
| b2 | `/exit`, option 2 "Move to background and exit" | still running; its `zsh -c` shell reparented to pid 1 at once |
| b3 | `herdr pane close` (the pane's processes get SIGHUP) | the claude process and the task gone within 6 s |
| b4 | `kill -KILL <claude pid>`, pane left open | still running; its `zsh -c` shell reparented to pid 1 |

Notes on the runs:

- `/exit` with a task running shows a dialog: "Background work is running /
  The following will stop when you exit: shell · <command>", with
  "1. Exit and stop tasks" (default), "2. Move to background and exit", and
  "3. Stay".
- b2: claude printed "backgrounded · <id>" with `claude agents`,
  `claude attach <id>`, `claude logs <id>`, and `claude stop <id>`. The TUI
  process ended, and a `claude bg-spare --bg-spare /tmp/cc-daemon-502/<…>`
  process (with its MCP servers) carried the session. After
  `claude stop <id>` ("stopped <id>"), the bg-spare process was gone, but the
  task and its sleep still ran; they were killed by hand.
- b4: the task and its sleep were killed by hand.

## Reading

- A background task outlives its session when claude cannot clean up
  (SIGKILL, a crash) and when the human picks "Move to background and exit".
  In the second case the session itself lives on in Claude Code's daemon
  until `claude stop`, and the task outlives even that.
- In both cases the task's shell is reparented to pid 1; while the session
  runs normally, the shell's parent is the claude TUI process. A wait could
  therefore tell both cases from the normal one by its shell's parent, but in
  the "Move to background" case that reads the session as gone although it
  runs on in the daemon (reachable only with `claude attach`).
- The task's shell leads its own session, so a terminal hang-up does not
  reach it directly; in b3 claude stopped it.
- Not checked here: the chain of a wait started through the Monitor tool
  (the `/director` skill's way), what a session in the daemon does when a
  task ends, and codex. An earlier live check (stage 8b-2a) saw codex
  0.160.1's app-server daemon keep a turn's background `ralph org wait
  --inbox` after the director was stopped; that wait's parent is the daemon,
  not pid 1.

## What the plan took from it

- A director's wait that outlives its session (b2, b4) keeps writing
  `director-wait.json`, and the status shows it as a wait that started at t
  and last wrote its mark at t2, never as a director that is alive. A new
  `ralph org wait --inbox --director` takes over the mark, and the leaked wait
  stops writing it (D2, D7; checked live in
  `docs/evidence/director-wait-livefire-2026-10-19.md`, (h)).
- Stopping the wait when its session ends is left to stage 8b-2c, after the
  checks listed under "Not checked here" (the tech-debt row of part B in
  `docs/tech-debt/README.md`).

Left on the machine: claude's own records of these sessions
(`~/.claude/projects/-private-tmp-x8b-proj/`, the trust entry for
`/private/tmp/x8b/proj`). `/private/tmp/x8b` was removed after the run.
