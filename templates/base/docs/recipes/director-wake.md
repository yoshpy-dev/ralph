# Waking the director by push: `[org.director].wake` and `wake_cmd`

The director is the session the human talks to for every feature org: a
promoted interactive session (`ralph org director attach`) or a headless
director (`ralph org director start`). By default it learns of a new open
inbox item by waiting on the inbox with `ralph org wait --inbox` (pull).
With push, the cross-org watch (`ralph org watch --all`) wakes it instead,
by typing one fixed line into its terminal. This recipe covers push: when
the watch sends the line, how to set it up for each kind of director, and
what happens when it cannot be sent.

## What push does

`[org.director].wake` is `["pull"]` (the default), `["push"]`, or
`["pull", "push"]` (either order). A registration records the wake it was
made with: `ralph org director attach` takes `--wake pull`, `--wake push`,
or `--wake pull,push` (default: the setting), and `ralph org director start`
records the setting. The watch acts on the recorded wake, not on the
setting, so attach or start the director again after changing it.

Each cycle, after it has sent the inbox items past a deadline and before
the pulse of each org, the watch:

1. reads the current registration, and goes on only when its wake has push
   and, for a headless director, its state is `running`;
2. takes the open inbox items that have no `woken` event;
3. sends the director this line, once for all of them:

   ```
   New org inbox items for the director: e3 e5. Read them with ralph org inbox and handle each as the director skill says.
   ```

   With more than 20 items it names the first 20 and then `plus N more`
   before the period.
4. once the line was sent, appends one `woken` event per item (with the
   registration's `director_id` and `via`, `wake_cmd` or `herdr`).

`ralph org inbox show <id>` lists the event as
`<ts> woken director_id=d1 via=herdr`, and `--json` gives the item `woken`
and `woken_at`. A woken event changes no state and does not stop a
deadline: an item the director does not ack still reaches the human at the
ack deadline (`[org.director].ack_minutes`, 10 minutes by default), as with
pull. Once a line naming an item was sent, later cycles skip that item; a
line that failed writes no event, so its items are named again in a later
try. If the watch dies after sending the line and before writing the
events, the next watch names those items once more.

`ralph org director status` shows the registration's `wake:` (and
`wake_target:` for a promoted session given one), and the `director:` line
of `ralph status` and of `director status` adds `, wake=<wake> to <target>`
when the wake has push.

## Settings

```toml
[org.director]
wake = ["push"]
wake_cmd = ["tmux", "-L", "ralph", "send-keys", "-t", "{target}", "-l", "{text}", ";", "send-keys", "-t", "{target}", "Enter"]
```

`wake_cmd` (empty by default) is an array of arguments that the watch runs
without a shell, after replacing two placeholders in every element:

| Placeholder | Replaced by |
| --- | --- |
| `{target}` | where the registration says to send (the table below) |
| `{text}` | the line above |

Loading `ralph.toml` checks only the values' shape: a `wake` that is empty,
holds anything but `pull` and `push`, or lists a value twice fails, and so
does a `wake_cmd` whose first element is empty or whitespace only, whose
first element holds a placeholder, or that holds any `{...}` other than
`{target}` and `{text}`. A placeholder is a `{` and the next `}` with no
other brace between them; a `{` or `}` outside such a pair is passed as
written. A value is replaced once, so a `{...}` inside the target or the
line is not replaced again. `wake = ["push"]` with an empty `wake_cmd`
loads: whether a director can be pushed depends on its registration, and
attach and start check that.

## Where the line goes: `{target}`

| Registration | `{target}` | Without `wake_cmd` |
| --- | --- | --- |
| a promoted session (`ralph org director attach`) | its `--wake-target` | cannot be pushed |
| a headless director in herdr (`ralph org director start`) | its herdr pane id | the watch types into the pane |
| a headless director started with `[org.director].launch_cmd` | its terminal name `director_<token>`, the `{name}` launch_cmd gave it | cannot be pushed |

attach is refused, and writes nothing, when:

- the wake has push and `wake_cmd` is empty: a promoted session has no
  default way to be sent the line;
- the wake has push, `wake_cmd` has `{target}`, and no `--wake-target` is
  given;
- `--wake-target` is given and the wake has no push;
- `--wake-target` is empty, has a control character, is longer than 256
  characters, or starts with `-` (the command could read it as an option).

start with `launch_cmd` is refused, writing nothing and running neither
`launch_cmd` nor herdr, when the wake has push and `wake_cmd` is empty. A
herdr start with push and an empty `wake_cmd` is accepted, since its pane
is the target.

## How `wake_cmd` runs

- It runs as an argv, without a shell, with an empty stdin, for up to 10 s.
  Exit 0 counts as sent. Its output is discarded, except the last 512 bytes,
  which go into the failure line.
- It runs in the directory of the `ralph.toml` it was read from: the main
  worktree root when it was read by the main-worktree rule below, the
  directory of the `--config` file, or else the directory the watch started
  in (whose `./ralph.toml` it read). It never runs in another working
  directory of the watch.
- A first element that contains `/` and is relative (`./scripts/wake.sh`)
  is resolved from that directory. A name without `/` (`tmux`, `orca`) is
  looked up in the watch's `PATH`, which is the `PATH` of the shell that
  started the watch, `--detach` included (and of the `ralph org start` or
  `ralph org director start` that launched it).
- Without `--config`, when the ledger is the main worktree's, attach, start,
  and `ralph org watch --all` read `wake` and `wake_cmd` from the main
  worktree's `ralph.toml`, so a leader cannot change them by editing its
  feature worktree's `ralph.toml`.
- The watch reads `wake_cmd` when it starts. After changing it, restart the
  watch: `ralph org watch --stop`, then `ralph org watch --all --detach`.

## Without `wake_cmd`: a headless director in herdr

With an empty `wake_cmd`, the watch can push only a headless director in
herdr. It confirms the pane as `ralph org director stop` does (the tab
labelled `director_<token>` in the workspace labelled
`ralph_director_<token>`), types the line with `herdr pane send-text`, waits
750 ms so Enter is not read as part of the text, and presses Enter with
`herdr pane send-keys`. A pane it cannot confirm gets nothing.

A promoted session, or a director started with `launch_cmd`, whose
registration has push while `wake_cmd` is empty (the setting was emptied
after attach or start) is not woken: the watch says so once on its stderr,
and the items reach the human at the ack deadline.

## Examples

### herdr: no `wake_cmd`

```toml
[org.director]
wake = ["push"]
```

```sh
ralph org director start --driver codex --model <slug>
```

### Orca

For a director session in an Orca terminal:

```toml
[org.director]
wake_cmd = ["orca", "terminal", "send", "--terminal", "{target}", "--text", "{text}", "--enter"]
```

```sh
ralph org director attach --driver claude --wake push --wake-target <handle>
```

`<handle>` is the runtime-issued handle of the terminal the session runs
in: `orca terminal list` shows it, and inside an Orca terminal the
`ORCA_TERMINAL_HANDLE` environment variable holds it. These commands come
from `orca terminal send --help` (Orca CLI 1.4.229) and were not run
against a live terminal. A director started with `launch_cmd` gets
`director_<token>` as its `{target}`, which is not an Orca handle, so this
example does not fit it as written.

### tmux, one call

```toml
[org.director]
wake_cmd = ["tmux", "-L", "ralph", "send-keys", "-t", "{target}", "-l", "{text}", ";", "send-keys", "-t", "{target}", "Enter"]
```

The `";"` element is an argument that tmux reads as its own command
separator; no shell is involved. `-l` sends the line literally.

### tmux, with a pause before Enter

Some agents read text and an Enter that arrive together as one paste and
do not submit it. A script can pause between the two:

```toml
[org.director]
wake_cmd = ["./scripts/wake-director.sh", "{target}", "{text}"]
```

```sh
#!/bin/sh
# scripts/wake-director.sh <target> <text>: type the line, then press Enter.
set -eu
tmux -L ralph send-keys -t "$1" -l "$2"
sleep 1
tmux -L ralph send-keys -t "$1" Enter
```

Make it executable (`chmod +x scripts/wake-director.sh`). The path is
resolved from the directory of the `ralph.toml`, and the script must finish
within the 10 s.

### A headless director started with `launch_cmd`

`{target}` is the terminal name `director_<token>`, which `launch_cmd`'s
`{name}` gave the terminal. Pair `wake_cmd` with the terminal `launch_cmd`
opens. With the tmux example of `docs/recipes/director-launch-cmd.md`
(`new-session -d -s {name}` on `-L ralph`), either tmux `wake_cmd` above
sends to that session:

```toml
[org.director]
launch_cmd = ["tmux", "-L", "ralph", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}"]
wake = ["push"]
wake_cmd = ["tmux", "-L", "ralph", "send-keys", "-t", "{target}", "-l", "{text}", ";", "send-keys", "-t", "{target}", "Enter"]
```

For a promoted session in tmux, pass its tmux target to attach instead:
`--wake push --wake-target <tmux target>`. `-L ralph` in these examples
names the tmux server; use the server the session runs on (drop `-L ralph`
for your default one).

## When to stay on pull

Keep a session a human types into on pull. The line and its Enter land in
the session's input, so text the human was typing when the line arrived is
sent along with it. The same holds while you type into a headless
director's pane or terminal (not checked live). With pull the director
waits with `ralph org wait --inbox` and nothing is typed into its input.

## What the agents do with the line

From a live check with codex-cli 0.160.1 as a headless director in herdr,
woken by the herdr default (`docs/evidence/director-wake-livefire-2026-10-18.md`):

- An idle codex took the line as a new turn. Following the `/director`
  skill, it read the inbox, acked the item, and asked the human about the
  QUESTION; it did not act on the item's body on its own.
- A line and Enter that arrived while a turn ran went into that running
  turn (the same `turn_id`), not into a queue for the next turn. The line
  was not lost, and codex read the inbox again in that turn.
- With `["pull", "push"]`, the director's wait and the line named the same
  item within a second, and the line can name an item already acked (its
  `woken` event then follows the `acked` one). The director acked it once.
- In its first turn codex did not start `ralph org wait --inbox` on its own.
  With `["push"]` that is what is wanted; with pull, ask the director to
  start the wait.

A Claude Code director was not run here. Claude Code is expected to hold
text typed during a turn as its next input; that was not checked live.

## When the line cannot be sent

- A `wake_cmd` that exits non-zero or runs out its 10 s, and a herdr call
  that fails, write one line on the watch's stderr (`watch.log` with
  `--detach`): `watchdog: director d1 not woken for e3 e5: ...`, with the
  error and the end of `wake_cmd`'s output. No `woken` event is written,
  and that registration is not tried again for 5 minutes. The gap lives in
  the watch's memory only: a restarted watch tries at once, and a new
  registration (a new `d<N>`) starts fresh.
- A registration that cannot be sent to at all (no sender, a target that
  fails the checks above, a `wake_cmd` that does not validate) is said once
  per registration: `watchdog: director d1 is not woken: ... (said once)`.
- When the line was sent but the `woken` events could not be written, the
  line says so, and the registration is held back for 5 minutes too.

In every case the items stay open, and the ack deadline sends them to the
human.

## Security

- The line is fixed text and item ids (`e<N>`) only. It never holds an
  item's body, its TYPE, its org_id, or the target. It contains none of the
  shell characters `;`, `|`, `&`, `$`, `` ` ``, `<`, `>`, `(`, `)`, `*`,
  `?`, `'`, `"`, and does not start with `/`, `!`, or `#`, so wherever it
  lands it is no command.
- `wake_cmd`'s first element may not hold a placeholder, so a
  registration, which a leader can write to `director.jsonl`, never chooses
  the program. Before every send, the watch checks the target it is about
  to use, `{target}` or the herdr pane (not empty, no control character, at
  most 256 characters, not starting with `-`), since `director.jsonl` can be
  edited by hand.
- Without `--config`, when the ledger is the main worktree's, `wake_cmd`
  comes from the main worktree's `ralph.toml` and runs in its root, and a
  relative program is resolved from there, so a script with the same name
  in a feature worktree is never run. A name without `/` follows the
  watch's `PATH`: do not start the watch from an environment whose `PATH` a
  leader can change.

## What was verified

The herdr default was run end to end with codex-cli 0.160.1 on an isolated
herdr server and an isolated ledger, with `["push"]` and with
`["pull", "push"]` (`docs/evidence/director-wake-livefire-2026-10-18.md`).
`wake_cmd`, the failure lines, and the 5-minute gap are covered by the unit
and CLI tests, with programs the tests write and a fake herdr. The Orca and
tmux examples were not run against a real terminal, and neither a Claude
Code director nor a promoted session attached with `--wake push
--wake-target` was run.
