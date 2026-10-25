# Running the org runtime on an isolated herdr server

A live check or a trial of the org runtime (`ralph org ...`) should not
touch your own herdr server, your agmsg database, your desktop
notifications, or a ledger that other orgs use. This recipe runs the org
runtime on a headless herdr server of its own, in a repository with its own
ledger, from a minimal environment.

The steps come from the setup and cleanup of the two-org trial of
2026-10-20 (`docs/evidence/org-trial-2026-10-20.md`: Setup, Findings F1 and
F2, Cleanup) and of the live checks after it
(`docs/evidence/busy-leader-2026-10-23.md` and the checks of 2026-10-22),
all on macOS with herdr 0.7.5, Claude Code 2.1.291, and agmsg 1.5.1. The
herdr facts below come from earlier live checks, also with herdr 0.7.5
(`docs/evidence/org-send-enter-timing-2026-09-19.md`,
`docs/evidence/org-plan-approval-live-2026-10-13.md`, and
`docs/evidence/herdr-pane-close-2026-10-07.md`). On 2026-10-24 and
2026-10-25 the steps were run end to end, also on macOS, in a dedicated
clone with two feature orgs at once and a promoted director, up to the
merge of both PRs with `ralph org merge`
(`docs/evidence/org-recheck-2026-10-24.md`: its Setup goes through this
recipe section by section, and its "Trial deviations" lists where the run
departed from the steps). This version includes the corrections from that
run.

The examples use `/tmp/ralph-iso` as the dedicated directory. Any short
path directly under `/tmp` works (see the socket path limit below). Create
it first, open to you only and without `-p` (`/tmp` is shared by every user
of the machine); the later steps create its subdirectories with `mkdir -p`:

```sh
mkdir -m 700 /tmp/ralph-iso
```

When that `mkdir` fails because the directory exists, do not reuse it: the
wrapper, the shim in `bin/`, and `claude-env` there are run. Remove your
own leftover from an earlier run, or pick another short name.

## Why isolate, and what has to move

- Your own herdr server may be running. Leave it alone: never stop it, and
  send every herdr command of this recipe to the dedicated socket.
  `pgrep -fl 'herdr server'` lists every herdr server on the machine, yours
  included.
- Move herdr with three variables, all pointing at the dedicated
  directory: `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, and `HERDR_SOCKET_PATH`.
  The live checks isolated herdr this way. In a run on 2026-09-19
  (`docs/evidence/org-send-enter-timing-2026-09-19.md`), a server started
  with only `HOME` and `HERDR_SOCKET_PATH` changed read the user's own
  `~/.config/herdr/session.json`, restored the user's workspaces, and
  appended its startup lines to the user's `herdr-server.log`. This recipe
  does not move `HOME`.
- Keep `HOME` the real one: on macOS claude reads its login from the
  keychain under the real `$HOME/Library/Keychains`, and in the same
  2026-09-19 run a claude seat on a server with a fake `HOME` was "Not
  logged in".
- A unix socket path must fit in `sun_path`, 104 bytes on macOS: in one
  check, a socket path of over 150 characters under a scratch directory
  kept herdr from starting. Keep the directory short and directly under
  `/tmp`.
- A server started from inside a Claude Code session, or from any terminal,
  passes that session's variables on to every pane: `CLAUDE_CODE_*`, the
  terminal's own variables, and the herdr pane variables of a session that
  itself runs in herdr. Start it with `env -i` and an explicit list.

The isolation keeps herdr's state, the socket, and the ledger apart from
yours. It is not a sandbox: the seats run with your real `HOME`, your gh
login, and, under the `autonomous` permission mode, claude's bypass mode.

## A repository with its own ledger

The org runtime keeps its ledger (the manifest, the inbox, the watch's lock
and liveness mark) in the main worktree root's `.harness/state/org/`, and
every linked worktree of a repository uses that same ledger. A linked
worktree of a repository whose ledger other orgs use is therefore not
isolated. Use one of these, under the dedicated directory:

- A dedicated clone, for a run whose orgs push branches and open PRs (the
  trial and the run of 2026-10-24). A `ralph.toml` that the clone does not
  track shows in its main checkout's `git status`. To keep it out, list it
  in the clone's `.git/info/exclude`, and create `.git/info` first
  (`mkdir -p .git/info`) when the clone has none: a git template can leave
  it out, and the clone of 2026-10-24 had none. The settings in such a
  `ralph.toml` do not reach the feature worktrees (see "The environment").
- A scratch git repository, for a check that opens no PR (the live
  checks). The latest ones gave it a base commit on `main`, `user.name` and
  `user.email` in the repository's own config, `core.hooksPath` pointing at
  an empty directory (so that your own git hooks do not run), and
  `.harness/state/` and `.harness/logs/` in `.gitignore`:

  ```sh
  mkdir -p /tmp/ralph-iso/work /tmp/ralph-iso/no-hooks
  cd /tmp/ralph-iso/work
  git init -b main
  git config user.name '<name>'
  git config user.email '<email>'
  git config core.hooksPath /tmp/ralph-iso/no-hooks
  printf '.harness/state/\n.harness/logs/\n' > .gitignore
  # write the run's ralph.toml here
  git add .gitignore ralph.toml
  git commit -m 'chore: base'
  ```

Put the run's settings in that repository's `ralph.toml`.

`ralph org start` and `ralph org watch --all --detach` launch a watch only
when none runs on the ledger. When one already accepts work there, they use
it without comparing its herdr server or its environment with theirs. A
watch started on the ledger from your normal shell would then watch the
isolated orgs through your own herdr socket and run the real `osascript`.
Before the first start, check that the ledger has no watch (see "Checks
before the first start").

## The environment

Write a wrapper that runs one command in the isolated environment, and run
every `ralph`, `herdr`, and `claude` command of the rest of this recipe
through it. `/tmp/ralph-iso/run`:

```sh
#!/bin/sh
# Run one command in the isolated environment.
D=/tmp/ralph-iso
# F1, below. Keep this line only when your git settings are in
# ~/.config/git/config; delete it when they are in ~/.gitconfig.
set -- GIT_CONFIG_GLOBAL="$HOME/.config/git/config" "$@"
# F1, below. Keep this line only when gh keeps its config in ~/.config/gh.
set -- GH_CONFIG_DIR="$HOME/.config/gh" "$@"
exec env -i \
  HOME="$HOME" USER="$USER" LOGNAME="$LOGNAME" \
  SHELL=/bin/bash LANG="$LANG" TERM=xterm-256color TMPDIR="$TMPDIR" \
  PATH="$D/bin:<the directories of ralph, claude, codex, node, git, gh, herdr, jq>:/usr/bin:/bin:/usr/sbin:/sbin" \
  XDG_CONFIG_HOME="$D/xdg-config" \
  XDG_STATE_HOME="$D/xdg-state" \
  HERDR_SOCKET_PATH="$D/h.sock" \
  CLAUDE_ENV_FILE="$D/claude-env" \
  "$@"
```

The two `set --` lines put their assignment in front of the command, where
`env` reads it like the others.

```sh
mkdir -p /tmp/ralph-iso/bin /tmp/ralph-iso/xdg-config /tmp/ralph-iso/xdg-state
chmod +x /tmp/ralph-iso/run
```

Replace the placeholder with the absolute directories that hold those
commands on your machine, and leave out the ones a run does not use. The
trial and the latest live check named the directories of the installed
binaries themselves, not of a version manager's shims.

| Variable | Value | Why |
| --- | --- | --- |
| `HOME` | the real one | claude's login on macOS is in the keychain under it |
| `USER`, `LOGNAME`, `LANG`, `TMPDIR` | as in your shell | the basic login environment |
| `SHELL` | a shell whose startup files add no flags to `claude` or `codex` | herdr starts each pane's shell from it (see "Shell aliases") |
| `TERM` | `xterm-256color` | the terminal type of the panes |
| `PATH` | `/tmp/ralph-iso/bin` first, then the directories above | the server, the panes, the watch, and the seats' commands find their binaries here |
| `XDG_CONFIG_HOME`, `XDG_STATE_HOME` | under `/tmp/ralph-iso` | herdr's config, logs, and session state |
| `HERDR_SOCKET_PATH` | `/tmp/ralph-iso/h.sock` | the dedicated server's socket |
| `GIT_CONFIG_GLOBAL`, `GH_CONFIG_DIR` | your git and gh config, each only when it lives under `~/.config` | git and gh in the panes find your config (F1, below) |
| `CLAUDE_ENV_FILE` | `/tmp/ralph-iso/claude-env` | the `PATH` of claude's Bash tool in a pane (F2, below) |

Add any other variable the seats' commands need; the trial added
`GOFLAGS=-timeout=30m` for the gate of a Go repository. `env -i` drops
everything else, among them `CLAUDE_CODE_*`, the herdr variables of a pane
(`HERDR_PANE_ID`, `HERDR_TAB_ID`, `HERDR_WORKSPACE_ID`, `HERDR_ENV`), and
`RALPH_ORG_AGMSG_HOME` (which would override `[org].agmsg_home`). It also
drops `SSH_AUTH_SOCK` and `GH_TOKEN`: add them to the wrapper when your
pushes or gh need them. The checks before the first start do not test a
push.

A run that uses the private agmsg copy (below) from a dedicated clone adds
`RALPH_ORG_AGMSG_HOME=/tmp/ralph-iso/agmsg` to the wrapper's `env -i` list,
as the run of 2026-10-24 did. A `ralph.toml` that the clone does not track
is not in the feature worktrees that `ralph org start --plan` creates, and
a leader's spawn there reads that worktree's own `ralph.toml`, so
`[org].agmsg_home` in the clone's untracked `ralph.toml` would reach ralph's
commands in the main checkout but not the leader's spawns (inferred from
the code, not checked). ralph reads the variable before `[org].agmsg_home`
(`ResolveAgmsgHome` in the ralph repository's
`internal/org/driver/agmsg.go`). With the variable in the wrapper, the
server, the panes, the watch, and the director's commands all get the same
value. In a scratch repository whose committed `ralph.toml` sets
`agmsg_home`, either way works.

As a first check before the server starts, run
`/tmp/ralph-iso/run herdr --help | grep -E '^(Config|Logs):'`. Both lines
should name paths under `/tmp/ralph-iso/xdg-config/herdr/` (in herdr 0.7.5
the logs too are under the config directory), and the check reaches no
server. On 2026-10-23, with herdr 0.7.5, `XDG_CONFIG_HOME=<S>/xc`,
`XDG_STATE_HOME=<S>/xs`, and `HERDR_SOCKET_PATH=<S>/h.sock` for a scratch
directory `<S>`, it printed `Config: <S>/xc/herdr/config.toml` and
`Logs:   <S>/xc/herdr/herdr.log (plus herdr-client.log, herdr-server.log)`,
and it created nothing under `<S>`. What the server really opens is checked
with `lsof` (see "Checks before the first start").

### git and gh config (trial finding F1)

With `XDG_CONFIG_HOME` moved, git and gh in the panes lose a config kept
under `~/.config`. git reads `~/.gitconfig` and
`$XDG_CONFIG_HOME/git/config`; gh reads `$GH_CONFIG_DIR`, else
`$XDG_CONFIG_HOME/gh`. In the trial's setup probe a pane had no
`user.name` and gh was not logged in; seats in that state cannot commit or
open a PR.

- Keep the wrapper's `GIT_CONFIG_GLOBAL` line only when your git settings
  are in `~/.config/git/config`. git then reads that file instead of both
  `~/.gitconfig` and the XDG file, so with your settings in `~/.gitconfig`
  the variable would make git skip them: delete the line, and git finds
  `~/.gitconfig` by itself, since `HOME` is the real one.
- Keep the `GH_CONFIG_DIR` line only when gh keeps its config in
  `~/.config/gh` (its default when you set neither `GH_CONFIG_DIR` nor
  `XDG_CONFIG_HOME` yourself). If you set `GH_CONFIG_DIR` in your shell,
  put that value in the line instead.
- Any other tool the seats run that reads `$XDG_CONFIG_HOME` loses its
  config the same way.

### PATH in claude's Bash tool (trial finding F2)

A herdr pane runs its shell as a login shell. For bash on macOS,
`/etc/profile` runs `path_helper`, which puts the system directories,
`/usr/bin` among them, ahead of the server's `PATH`. A command that has a
copy there then resolves to that copy in the pane and in claude's Bash
tool: in the trial, `command -v osascript` printed the real
`/usr/bin/osascript` instead of the shim. `BASH_ENV` did not help. A file
that exports the `PATH`, named by `CLAUDE_ENV_FILE`, did:

```sh
# /tmp/ralph-iso/claude-env
export PATH="/tmp/ralph-iso/bin:<the same directories as in run>:/usr/bin:/bin:/usr/sbin:/sbin"
```

- This is for claude seats, and it fixes claude's Bash tool only. The
  pane's own shell keeps the `path_helper` order. codex seats were not
  checked for this.
- Claude Code reads the file when a session starts. The trial inferred
  this and then checked it on one command each from a seat started before
  a change to the file and one started after it. Write the file before the
  server starts any claude; a change reaches only sessions started later.
- ralph creates seat panes with
  `herdr tab create --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off`
  when the herdr has `--env` (herdr 0.7.5 does), so the file needs neither
  value. With a herdr that has no `--env`, ralph warns on stderr and the
  panes get neither; the trial, run before ralph set the value, exported
  `AGMSG_SELF_NAME=off` in this file.

### Shell aliases

herdr types a seat's command into the pane's interactive shell, so an
alias for `claude` or `codex` in that shell's startup files is expanded. A
`codex` alias that adds `--model` makes the seat fail to start, and a
`claude` alias that adds flags changes how the seat runs; the /org skill's
prerequisites (`.claude/skills/org/SKILL.md`) list each case. Choose
`SHELL` so that the pane's startup files define no such alias. The trial
and the latest live check used `/bin/bash`. Then:

- Run `/tmp/ralph-iso/run ralph doctor` in the repository and read its
  "Shell aliases (codex/claude)" check. It reports such aliases with the
  file and line, from the zsh, bash, and fish startup files under your home
  directory, whichever `SHELL` you chose; a finding in a file of a shell
  the panes do not run does not reach them.
- In the throwaway pane below, `type claude codex` prints a path for each,
  and `alias` lists no alias for them.

## Optional: a private agmsg and a notification shim

The latest live checks used both; the trial used the shim with the user's
own agmsg.

- A private copy of agmsg. Copy `scripts`, `plugins`, `types`, `SKILL.md`,
  and `VERSION` from your agmsg home (`~/.agents/skills/agmsg` by default)
  into the dedicated directory, without `db`, `teams`, `run`, or `agents`,
  and point ralph at the copy with an absolute path: in a scratch
  repository, in its committed `ralph.toml` as below; in a dedicated clone,
  with `RALPH_ORG_AGMSG_HOME` in the wrapper (see "The environment" for
  why):

  ```sh
  mkdir -p /tmp/ralph-iso/agmsg
  cd ~/.agents/skills/agmsg
  cp -R scripts plugins types SKILL.md VERSION /tmp/ralph-iso/agmsg/
  ```

  ```toml
  [org]
  agmsg_home = "/tmp/ralph-iso/agmsg"
  ```

  The copy reaches ralph's own agmsg calls only: the leader's join to the
  org's team, each seat's join and HELLO at its spawn, the watch's ALERTs
  to the leader, and the leader's inbox and history commands (the leader's
  role prompt names the copy's scripts directory). A claude seat's own
  session sends its messages with its agmsg skill, which runs the scripts
  of your own agmsg home (`~/.agents/skills/agmsg` by default). In the run
  of 2026-10-24 (`docs/evidence/org-recheck-2026-10-24.md`, RC1) that home
  had no team of the run, so the seats' sends failed with "Team not found".
  One seat's message reached its leader through agmsg (that seat found the
  copy's path in `RALPH_ORG_AGMSG_HOME`); the others went through
  `ralph org send` or were not sent, and one leader read its reviewer's
  committed report instead. Your own agmsg database was not written (its
  `messages.db` and `messages.db-wal` kept their times). So for a run whose
  seats report to their leader, which is any feature org, leave the copy
  out and use your own agmsg, as the trial of 2026-10-20 did, until ralph
  gives the seats the path (tracked in `docs/tech-debt/README.md`, RC1).
  The copy suits checks whose seats send no messages.
  (`ralph org director start --config` refuses a relative `agmsg_home`.)
- An `osascript` shim. ralph runs `osascript` by `PATH` lookup for its
  desktop notifications to the human (from the watch, and from
  `ralph org escalate` when it sends an item at once). A shim first on
  `PATH` that logs its arguments and exits 0 keeps them off your desktop,
  and its log holds every call:

  ```sh
  #!/bin/sh
  # /tmp/ralph-iso/bin/osascript
  printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >> /tmp/ralph-iso/osascript.log
  exit 0
  ```

  Make it executable. Inside a pane it reaches claude's Bash tool only
  through `CLAUDE_ENV_FILE` (F2).

  The shim also receives the `osascript` calls of your own Claude Code
  hooks in the sessions the run starts. In the run of 2026-10-24 all 12
  shim lines came from a Stop hook (a desktop notification of the
  session's last message) in the on-demand watcher's `claude -p` sessions,
  with the title "Claude Code" (RC11; the hook is named from the run's
  notes, and the hook itself was not read). ralph's own notifications carry
  the title "ralph org watch" or "ralph org escalate". Outside isolation,
  each watcher run would post such a notification.

## Start the server

```sh
cd /tmp/ralph-iso
nohup /tmp/ralph-iso/run herdr server > /tmp/ralph-iso/herdr-server.out 2>&1 &
echo $! > /tmp/ralph-iso/herdr-server.pid
```

`nohup`, the wrapper, and `env` each replace themselves with the next
command, so the saved pid is the server's. Check that
`pgrep -fl 'herdr server'` lists it. Run that check as a command of its
own: in a compound command whose own text contains those words, `pgrep`
lists that shell too (seen in the run of 2026-10-24).

## Checks before the first start

### The server

- `lsof -p "$(cat /tmp/ralph-iso/herdr-server.pid)"`: apart from its
  binary, the system libraries, and devices such as `/dev/null`, the files
  and sockets the server has open are under `/tmp/ralph-iso`, and none is
  under your own herdr config directory.
- `ps eww -o command= -p "$(cat /tmp/ralph-iso/herdr-server.pid)" | tr ' ' '\n' | cut -d= -f1`
  (macOS; `ww` keeps a terminal from cutting the line at the window width)
  prints the server's command and then the names of its environment
  variables, one to a line: only the wrapper's variables should appear.
  `cut` drops the values, so a token you added to the wrapper does not
  reach your terminal or a session record. When no name follows the
  command, `ps` could not read the environment and the check tells you
  nothing; the `lsof` check above still applies.
- The `herdr --help` check under "The environment".

### Folder trust, and a probe in a throwaway pane

In a repository claude has not seen, claude shows a folder-trust dialog
with the cursor on "No, exit". Answer it once yourself, in a throwaway
pane, before the first start, and use the same pane to check what a seat's
shell and claude's Bash tool resolve:

```sh
R=/tmp/ralph-iso/run
REPO=/tmp/ralph-iso/work
$R herdr workspace create --cwd "$REPO" --label probe | jq -r .result.workspace.workspace_id
$R herdr tab create --workspace <workspace-id> --cwd "$REPO" --label probe | jq -r .result.root_pane.pane_id
$R herdr pane run <pane-id> 'type claude codex; alias'
$R herdr pane read <pane-id> --source recent --lines 40 --format text
$R herdr pane run <pane-id> 'claude --model haiku'
```

1. The shell: `type claude codex` names the binaries you meant, and
   `alias` lists no alias for them.
2. Trust: read the pane, move the cursor to "Yes, I trust this folder",
   and press Enter. To answer it yourself, run `/tmp/ralph-iso/run herdr`
   in your own terminal: it opens herdr's interface on the dedicated
   server (the wrapper gives it `HERDR_SOCKET_PATH`), where you go to the
   probe pane. In the run of 2026-10-24 the human answered the dialog
   there (`docs/evidence/org-recheck-2026-10-24.md`, RC7), since a director
   does not send keys to a pane (the /director skill). The trial and the latest live check sent the Down
   key, then Enter, to the pane. Running `cd <the repository> && claude`
   in your own terminal instead did not work in the run of 2026-10-24: it
   showed no trust dialog and left no trust entry in `~/.claude.json`,
   although the throwaway pane showed the dialog afterwards (why was not
   found). Answer the dialog in the pane.
3. The probe: have claude run `command -v git gh ralph claude osascript`
   (without `osascript` when there is no shim), `git config user.name`,
   and `gh auth status` with its Bash tool. Type the request with
   `$R herdr pane send-text <pane-id> '<request>'`, then
   `$R herdr pane send-keys <pane-id> Enter`, and read the pane. Each
   `command -v` line names the binary you meant (for `osascript`, the
   shim), `git config user.name` prints your name, and `gh auth status`
   shows the account you expect as logged in. Approve the commands if
   claude asks.

   A way without typing keys, which the run of 2026-10-24 used after the
   trust answer and `/exit`: run
   `$R herdr pane run <pane-id> "claude -p '<request>' --model haiku --allowedTools Bash"`
   and read the pane for the answer. The request must come right after
   `-p`: `--allowedTools` takes every word after it as a tool name, and
   with the request after it claude exits with "Input must be provided
   either through stdin or as a prompt argument when using --print"
   (RC8). In that run the shim came first in the `-p` session's Bash tool.
   That fits a `-p` session reading `CLAUDE_ENV_FILE` too, but it is not
   confirmed: the pane's own shell was not checked, so it is not known
   that `path_helper` had put `/usr/bin` first in that pane (F2).
4. Exit claude with `/exit`, then close the workspace:
   `$R herdr workspace close <workspace-id>`.

claude does not ask again under a trusted directory: in the trial, a claude
started in a new git worktree under the trusted clone's `.claude/worktrees/`
showed no dialog, and neither did the two feature worktrees (Claude Code
2.1.291; another version may behave differently). A seat whose `--cwd` is
outside the repository needs its own answer (inferred, since that directory
is not under a trusted one; not checked). claude records the trust in
`~/.claude.json` under the resolved path (`/private/tmp/...` on macOS,
where `/tmp` is a link). codex's trust screen was not checked here; for
codex seats see `docs/recipes/codex-setup.md` (`codex trust .`). The rest
of the prerequisites, the one-time consent to claude's bypass mode among
them, are in the /org skill's prerequisites (`.claude/skills/org/SKILL.md`).

### ralph doctor and ralph status

In the repository:

```sh
cd /tmp/ralph-iso/work
/tmp/ralph-iso/run ralph doctor
/tmp/ralph-iso/run ralph status
```

- `ralph doctor`: the herdr and agmsg checks pass, and "Shell aliases
  (codex/claude)" reports nothing that reaches the panes.
- `ralph status`: the state directory it prints is the repository's
  `.harness/state/org`, and its watch line begins with
  `org watch: not running`. If a watch runs there, stop it
  (`/tmp/ralph-iso/run ralph org watch --all --stop`), so that the first
  start launches a new one from the wrapper.

## Run

Run every `ralph` and `herdr` command through the wrapper, from the
repository. The watch that `ralph org start` launches when none runs, like
the one `ralph org watch --all --detach` launches, keeps the environment of
the command that launched it, apart from `HERDR_PANE_ID`, `HERDR_TAB_ID`,
`HERDR_WORKSPACE_ID`, and `HERDR_ENV`. Launched from the wrapper, it uses
the dedicated socket, the shim, and the F1 variables.

A promoted director, an interactive session outside the server, runs its
`ralph` and `gh` commands through the wrapper too, or at least with the
same `HERDR_SOCKET_PATH`, `PATH`, and F1 variables (the trial's director set
the same `HERDR_SOCKET_PATH` and `PATH` on each of its commands). The
`herdr` commands of this recipe (the server, the throwaway pane, the trust
answer) are the human's: the /director skill
(`.claude/skills/director/SKILL.md`) limits a director to `ralph` and `gh`
and does not let it send keys to a pane. A headless director started
through the wrapper (`ralph org director start`) runs in a pane of the
dedicated server and has the server's environment.

## Clean up

Every `ralph` and `herdr` command of this section runs through the
wrapper, as written below: a bare `ralph org disband` has no
`HERDR_SOCKET_PATH` and calls the herdr of your own server. In this order:

1. Stop the orgs and the director. `ralph org disband` writes no org
   report, so to keep one, run
   `/tmp/ralph-iso/run ralph org report --org-id <id>` first, from the
   repository's main checkout (the trial's leader ran it in its feature
   worktree, and the report ended up in a commit on the feature branch).
   Disband each org that still runs
   (`/tmp/ralph-iso/run ralph org disband --org-id <id>`), stop a headless
   director (`/tmp/ralph-iso/run ralph org director stop --id <id>`) or
   detach a promoted one
   (`/tmp/ralph-iso/run ralph org director detach --id <id>`), and look in
   `/tmp/ralph-iso/run ralph org inbox` for items still open.
2. Stop the ledger's watch with
   `/tmp/ralph-iso/run ralph org watch --all --stop`. The watch does not
   stop when the orgs are gone. Then check, from the repository, that
   `/tmp/ralph-iso/run ralph status` prints `org watch: not running` (with
   the time it stopped). Next, `cd ~`, out of the directory, so that the
   checks below list neither your own shell nor the `lsof` itself. From
   there, check that `pgrep -fl /tmp/ralph-iso` lists no process (the
   detached watch's command line names its ledger, which is under the
   dedicated directory), and that `lsof +D /tmp/ralph-iso` lists no
   process other than the dedicated herdr server. `lsof +D` lists every
   process whose working directory or an open file is under the directory,
   whatever its process name, so it also finds a claude that runs under
   another name.

   If `--stop` exits 1 because the watch did not stop within its 30 s, the
   stop request stays in the ledger, and the watch stops once its current
   cycle, or its wait for an in-flight watcher call, ends; the error also
   says how to stop it at once. Do not stop the server or remove the
   directory until `/tmp/ralph-iso/run ralph status`, run from the
   repository, prints `org watch: not running`. Outside a git work tree,
   ralph reads the ledger under the current directory, so from `~` it
   would print `not running` for a watch that still runs.
3. Stop the dedicated server: `/tmp/ralph-iso/run herdr server stop`.
   Check that `pgrep -fl 'herdr server'` no longer lists its pid (your own
   server, if it runs, is still listed), that `/tmp/ralph-iso/h.sock`
   is gone, and that `lsof +D /tmp/ralph-iso` lists no process.
4. From outside the directory, copy what you keep as evidence out of it,
   and `rm -rf /tmp/ralph-iso`.

`rm -rf` removes only the local copies. When the orgs of a dedicated clone
pushed branches or opened PRs, those stay on origin; close or delete the
ones you do not keep there.

Left outside the dedicated directory, in your own stores:

- a trust entry for the repository in `~/.claude.json` (under
  `/private/tmp/...`). Every Claude Code session writes that file, so the
  live checks left the entry rather than edit the file by hand;
- Claude Code's session-record directories under `~/.claude/projects/`,
  one for each directory a claude ran in (the repository and each feature
  worktree, for example), and whatever else your Claude Code setup
  writes per session, such as history lines and plugin data;
- when you use your own agmsg instead of a copy, the role-session marks of
  the run's seats under the agmsg home's `run/`. With the copy, the seats'
  agmsg skill still uses your own agmsg home (RC1), though in the run of
  2026-10-24 it wrote nothing there.

In the evidence you write about the run, do not copy gh or git config,
tokens, pane ids, or session ids, and write your home directory as `~`.

## Running again

- Before you start the dedicated server again, remove the `session.json`
  files that
  `find /tmp/ralph-iso/xdg-config /tmp/ralph-iso/xdg-state -name session.json`
  lists. A server restores the workspaces saved there, the dedicated one
  included, and a restarted server whose `session.json` was left continued
  its workspace numbering from it.
- Use new org ids. A seat whose herdr agent name (`<org>_<seat>`) is still
  taken from the earlier run fails with `agent_name_taken`.

## What this recipe does not cover

- codex seats: the `PATH` of their shell commands (F2 was checked for
  claude's Bash tool only), codex's trust screen, and the agmsg writable
  root of a sandboxed codex seat (`docs/recipes/codex-seat-permissions.md`).
- A login zsh as the pane shell. macOS's `/etc/zprofile` runs `path_helper`
  as well, so such a pane probably reorders `PATH` the same way. Not
  checked.
- Linux. The keychain and `path_helper` are macOS's, and the steps were
  not tried on Linux.
- What the end-to-end run of 2026-10-24
  (`docs/evidence/org-recheck-2026-10-24.md`) did not exercise: the scratch
  repository (it used a dedicated clone), a headless director started with
  `ralph org director start` inside the dedicated server (its director was
  a promoted one), stopping and restarting an org or the watch, the steps
  of "Running again", and the seats of a feature org working with the
  private agmsg copy (they did not; see "Optional: a private agmsg and a
  notification shim").
