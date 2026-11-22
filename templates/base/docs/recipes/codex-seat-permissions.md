# Codex seat permission modes: verify per machine, opt out if it fails

`[org.permissions].codex_verified` defaults to `true`, which enables the
mapping autonomous → `--sandbox workspace-write --ask-for-approval never` and
edits → `--sandbox workspace-write`, each followed by
`-c sandbox_workspace_write.network_access=true` (see "The network of a
codex seat" below). How those flags behave depends on the
installed codex CLI version and on your own `~/.codex/config.toml`, so check
them **on each machine** with this recipe. If the check does not pass, set
the flag to `false`: codex seats then accept only `guarded` (the codex CLI's
own interactive default) and reject `autonomous` / `edits` with a
fail-closed error.

The meta-repo's own run (codex-cli 0.154.0, herdr 0.7.5, agmsg 1.1.13) is
recorded in `docs/evidence/codex-seat-permissions-2026-09-18.md`.

## Prerequisites

- `ralph doctor` passes for codex, herdr, and agmsg.
- The herdr server need not be running: `ralph org spawn`, which the probes
  below use, starts a stopped one itself, as `ralph org start` and
  `ralph org director start` do. Start it yourself first (open the herdr TUI,
  or run `herdr server` headless) only to choose the environment it runs in,
  as `docs/recipes/isolated-herdr-server.md` does.
- **Shell aliases break every codex spawn, not just this recipe.** If your
  shell rc defines `alias codex="codex -m ..."`, herdr sends the seat command
  to the pane's interactive shell as is, the alias expands, and codex exits
  with `error: the argument '--model <MODEL>' cannot be used multiple
  times`; `ralph org spawn` then reports `spawn_failed` after the
  `agent_start` timeout. The short form `-m` collides the same way. Remove
  the alias, or start herdr from a HOME / rc that does not define it. A
  third way is to define it only outside herdr's panes: herdr sets
  `HERDR_PANE_ID` in the shell of each pane it opens (checked on herdr
  0.7.5), so a line such as
  `[[ -n $HERDR_PANE_ID ]] || alias codex="codex -m ..."`, one alias per
  line, defines nothing in a seat's pane. An
  alias that adds `--sandbox` (or `-s`) fails the same way on edits and
  autonomous seats, where ralph passes that flag itself; on a guarded seat
  the alias's sandbox silently applies. ralph passes `--ask-for-approval`
  to autonomous seats only, so an alias that adds it (or `-a`) fails on
  autonomous seats and silently sets the approval policy of edits and
  guarded seats. claude accepts a repeated flag and the last value
  wins (claude 2.1.274 CLI; not verified on a live seat): ralph passes
  `--model` after the alias for every seat whose model is not `""`, so a
  `claude` alias that adds `--model` does not break the spawn (for a seat
  whose `[org.role_models]` model is `""`, ralph passes no `--model`, so the
  alias's model silently applies, on codex as well as claude, and a codex
  alias with `-m` no longer collides), but ralph passes `--permission-mode`
  only on edits and autonomous seats, so a guarded claude seat runs with the
  alias's permission mode. Every other flag in the alias still reaches
  each seat. The same holds for a headless director that `ralph org
  director start` opens in herdr, since it runs through the pane's shell
  too; a director started with `[org.director].launch_cmd` runs the agent
  by its absolute path (`exec` in its launch script), so no alias applies
  there. `ralph doctor`'s "Shell aliases (codex/claude)" check reports
  such aliases with the rc file and line (info when a claude alias only
  adds `--model`, when herdr is not installed, or when a `HERDR_PANE_ID`
  test keeps the alias out of herdr's panes; warn otherwise). It reads
  such a test written as `[[ ... ]]`, `[ ... ]`, or `test ...` with `-n` or
  `-z` and `$HERDR_PANE_ID` or `${HERDR_PANE_ID}`, quoted or not, in two
  places: at the head of a list (the start of a line that does not
  continue the previous one, or after `;`, `then`, `else`, `do`, or `{`),
  followed directly by one alias statement after `||` for `-n` or `&&` for
  `-z`; or as the whole condition of an `if`, on one line or several, with
  the alias on its then side for `-z` or its else side for `-n`. It treats
  any other form (another condition, the opposite operator or side, an
  alias inside a `{ ...; }` group after the test, an `elif` and what
  follows it, fish syntax) like an alias with no test.
- **Add the agmsg database to the sandbox's writable roots.** Under
  `--sandbox workspace-write` codex can write only to the working directory
  and to `/tmp`-style temp roots; the agmsg SQLite database
  (`~/.agents/skills/agmsg/db/`) is outside them, so a seat's `send.sh` fails
  with `attempt to write a readonly database` and no RESULT reaches leader.
  Add to `~/.codex/config.toml`:

  ```toml
  [sandbox_workspace_write]
  writable_roots = ["<your home directory>/.agents/skills/agmsg/db"]
  ```

  Use the absolute path (`echo "$HOME/.agents/skills/agmsg/db"`). The
  meta-repo run used an absolute path; a `~` prefix was not tested. List
  the `db` directory itself: a broader root such as your home directory
  does not help, because codex keeps `.git`, `.agents`, and `.codex`
  directories under a writable root read-only, recursively. If
  `AGMSG_STORAGE_PATH` is set, agmsg keeps its database there instead, so
  list that directory. `ralph doctor`'s "Codex sandbox (agmsg writable
  root)" check warns when a codex seat of this project could run under
  `workspace-write` and no writable root covers the agmsg store. That is
  the case when `codex_verified = true` and a role resolves to edits or
  autonomous, or when your codex config sets `sandbox_mode =
  "workspace-write"` and a role resolves to guarded. A `director` entry in
  `[org.permissions.roles]` counts only when it is guarded, because
  `ralph org director start` refuses a codex director in edits and
  autonomous (see "The org ledger and `--add-dir`" below). With codex
  missing from `[org].driver_pool` nothing is needed; with codex in it,
  every role counts, since `--driver codex` can start any role on codex
  whatever its `[org.role_models]` entry says. `/tmp` and
  `$TMPDIR` count as writable unless `exclude_slash_tmp` /
  `exclude_tmpdir_env_var` is set. The check reads only the user-level
  `config.toml`; profiles, project-level config, and `-c` overrides are not
  evaluated. When the agmsg store itself lives inside this project (and not
  under its `.git`, `.agents`, or `.codex` directory), a seat whose working
  directory contains it can already write there and the warn says so; a
  seat whose working directory does not contain it (a task worktree, for
  example) still needs the writable root.
- **Keep the scratch working directory out of `/tmp`.** `/tmp` is itself a
  writable root, so a target under it does not test the sandbox boundary.
  Use a throwaway directory under `$HOME` (`git init` it) for the seat's cwd
  and a second throwaway file under `$HOME` as the "outside" target.

## The org ledger and `--add-dir`

Without `--state-dir` or an `[org].state_dir` in `ralph.local.toml`, ralph
uses one org ledger per repository: from the main checkout and from every
linked worktree, the state directory is the main worktree's
`.harness/state/org/` (the `/org` skill's prerequisites give the full order
and the exceptions, such as the worktrees of a bare repository). ralph no
longer reads `RALPH_ORG_STATE_DIR`. A leader whose cwd is a linked worktree
therefore writes the ledger outside its own cwd, which
`--sandbox workspace-write` refuses. ralph adds `--add-dir <state dir>` right
after the sandbox flags and the network `-c` (see "The network of a codex
seat" below) of a codex leader seat, and of the codex reviewer
seat of a feature org (an org started with `ralph org start --plan`), when
the seat runs in edits or autonomous mode and the state directory is not
inside the seat's `--cwd`. The leader runs `ralph org` verbs; a feature
org's reviewer runs `ralph org gate` at the leader's TASK and so writes the
gate record and the gate log to the ledger. Like the leader, that reviewer can
then write the whole ledger (the manifest, the inbox, the seats' prompt files),
and only its role prompt limits it to the gate's records. Seats of every
other role (the implementer, the reviewer of an org with no feature binding,
or a role with no template that runs from `--prompt`) report to the leader
over agmsg and do not write the ledger, and the flag would let them rewrite
the manifest, the receipts, and other seats' prompt files. Guarded codex
seats and claude seats get no `--add-dir` either. The headless director
that `ralph org director start` launches writes the ledger too, but gets no
`--add-dir`: start refuses a codex director in edits and autonomous,
whatever `codex_verified` says, because under `workspace-write` codex keeps
`.git` read-only and the director could not run `git worktree add` to start
a feature org, so a codex director runs guarded only. The flag covers only
the state directory; the agmsg database still needs the writable root
above. The procedure below spawns reviewer seats in orgs with no feature
binding, so their child process arguments carry no `--add-dir` even when
the scratch `--state-dir` lies outside `<scratch-cwd>`.

Checked on 2026-10-07 with codex-cli 0.160.0 on macOS, with the cwd and
the target directory both outside `/tmp`, `$TMPDIR`, and the repository:
under `--sandbox workspace-write` without `--add-dir`, `touch` created
nothing and printed `touch: <target>: Operation not permitted` (macOS's BSD
`touch`; other platforms may word the error differently). With
`--add-dir <target dir>` the file was created. The commands and their output
are recorded in the ralph repository's
`docs/evidence/codex-add-dir-2026-10-07.md`.

To re-check on your machine, run the same probe twice. Keep both
directories outside `/tmp` and `$TMPDIR`, which are writable by default, and
outside any repository, so that no project's `.codex/config.toml` or
`AGENTS.md` changes the result:

```sh
p="$HOME/codex-add-dir-probe"
mkdir -p "$p/cwd" "$p/target" && cd "$p/cwd"
ask="Run this command once. If it fails, do not retry or work around it; report the exit status and the error text: touch $p/target/probe"
command codex exec --skip-git-repo-check --sandbox workspace-write "$ask" </dev/null
ls "$p/target"   # expected: empty
command codex exec --skip-git-repo-check --sandbox workspace-write --add-dir "$p/target" "$ask" </dev/null
ls "$p/target"   # expected: probe
```

`</dev/null` keeps `codex exec` from waiting for more input on stdin, and
`command` skips a shell alias that adds its own flags. Both runs exit 0 even
when `touch` fails, so judge by whether `probe` exists, not by the exit code.

If the first run already creates `probe`, a writable root in your
`~/.codex/config.toml` covers the target; move the probe directory out of
it and run again. Remove `$HOME/codex-add-dir-probe` afterwards.

## The network of a codex seat

Under `--sandbox workspace-write` codex blocks the network unless
`sandbox_workspace_write.network_access` is true, and codex's own default is
false. A command a seat runs that reaches the network then fails inside the
seat: tflint's plugin init (`tflint --init`), a dependency download, a gate
script that fetches a tool, or anything else that resolves a host name. So
ralph lets codex seats in edits and autonomous reach the network: it passes
`-c sandbox_workspace_write.network_access=true` right after their sandbox
flags, to the leader, the implementer, and the reviewer alike. The command
of such a seat begins as below. In place of `...` come
`--add-dir <state dir>` when the seat gets it (above), `--model <slug>`
unless its model is `""`, and the pointer to its role prompt:

```sh
codex --sandbox workspace-write --ask-for-approval never -c sandbox_workspace_write.network_access=true ...   # autonomous
codex --sandbox workspace-write -c sandbox_workspace_write.network_access=true ...                            # edits
```

The network this opens is not limited to some hosts: codex's
`workspace-write` sandbox has no list of allowed hosts, so the agent in such
a seat can reach any host and send there what it has read in the seat. To
close it, set the key below to `false` and start the seats again (see "A
change reaches only the seats started after it" below).

`[org.permissions].codex_network_access` (bool, default `true`; a
`ralph.toml` without the key reads as `true`) closes it:

```toml
[org.permissions]
codex_network_access = false
```

ralph then passes `-c sandbox_workspace_write.network_access=false` to
close the network of those seats whatever `~/.codex/config.toml` says, since
a `-c` override wins over that file. That was checked once, with
`codex sandbox` on codex-cli 0.166.0 on macOS, where `=false` closed the
network that file opened; it was not checked inside a seat (see "Check it
on your machine" below). With the key `false`, a check that needs
the network fails inside codex seats in edits and autonomous, such as
tflint's plugin init or a dependency download. Run such checks outside
those seats (in a claude seat or in your own shell), or set the key back to
`true`. `ralph config get org.permissions.codex_network_access` prints the
value in effect.

The key does not reach:

- guarded codex seats: ralph passes them no sandbox flag and no `-c`, so
  `~/.codex/config.toml` decides (`sandbox_mode` and
  `[sandbox_workspace_write] network_access`);
- codex seats while `codex_verified = false`: they start only in guarded;
- a headless codex director: `ralph org director start` refuses edits and
  autonomous for it (see "The org ledger and `--add-dir`" above), so it runs
  guarded;
- claude seats: codex's sandbox does not apply to them;
- the codex calls of the standard flow's plan advisory and `/cross-review`:
  they are not seats, and they run codex read-only, which blocks the
  network by default.

A shell alias that adds its own `-c sandbox_workspace_write.network_access=...`
to `codex` does not change it on edits and autonomous seats: codex takes `-c`
any number of times, a later value of the same key wins, and ralph's `-c`
comes after the alias's words (read in codex's source at rust-v0.166.0, not
checked on a live seat). On a guarded seat the alias's value applies.

`ralph doctor`'s "Codex sandbox (seat network)" check reports the key when a
codex seat of this project can run under `workspace-write`: codex is in
`[org].driver_pool`, `codex_verified = true`, and a role other than
`director` resolves to edits or autonomous (the test the "Codex sandbox
(agmsg writable root)" check uses for its first case). It is `info` while
the key is `true` and `warn` while it is `false`, never `fail`, and
`--strict` leaves the `warn` a `warn`. Otherwise it passes with a
`not needed:` detail. It reads ralph's settings only, not
`~/.codex/config.toml`.

### Check it on your machine

`codex sandbox` runs a command under the sandbox codex gives a seat, with
the same `-c` overrides. Run a command that reaches the network both ways,
from a directory outside any repository so that no project's
`.codex/config.toml` changes the result. `command` skips a shell alias that
adds its own flags:

```sh
cd /tmp
command codex sandbox -c sandbox_mode=workspace-write -c sandbox_workspace_write.network_access=true -- \
  /usr/bin/curl -sS -o /dev/null -w '%{http_code}\n' --max-time 15 https://example.com   # expected: 200
command codex sandbox -c sandbox_mode=workspace-write -c sandbox_workspace_write.network_access=false -- \
  /usr/bin/curl -sS -o /dev/null -w '%{http_code}\n' --max-time 15 https://example.com   # expected: curl: (6) Could not resolve host, then 000
```

On codex-cli 0.166.0 on macOS, with `network_access = true` in
`~/.codex/config.toml`, the first printed 200 and the second
`curl: (6) Could not resolve host: example.com` and 000, so the `-c` value
won over that file. A check run inside a live seat (tflint's plugin init in
a seat's pane) was not tried, nor a machine whose codex config has no
`network_access`. The commands and their output are recorded in the ralph
repository's `docs/evidence/codex-seat-network-2026-11-21.md`.

### A change reaches only the seats started after it

ralph fixes a seat's arguments when it starts the seat, and a running seat
keeps them. A change to `codex_network_access`, and an upgrade or a
downgrade of ralph that changes what ralph passes, reaches only the codex
seats started after it. `ralph org spawn` of a seat that is already running
returns that seat as it is, and `ralph org resume` only types into a seat's
pane. A seat started by a ralph older than this key got no network `-c` and
follows `~/.codex/config.toml`, so it has no network when that file does not
set `network_access`; after a downgrade to such a ralph, the seats it starts
do the same.

`ralph org spawn`, `ralph org start`, and `ralph org reseat` read
`[org.permissions]` from `--config`, else from the `ralph.toml` of the
directory they run in, with the main worktree's `ralph.local.toml` on top.
A seat that a feature org's leader spawns therefore reads the key from the
feature worktree's `ralph.toml`, the copy on the feature branch, not from
the main worktree's.

To apply a change to running seats, start them again:

- The seats of a feature org (one started with `ralph org start --plan`):
  run `ralph org reseat --org-id <id>` from outside the org's seats (the
  director's session or your own shell), in a directory whose `ralph.toml`
  has the value you want. `--dry-run` first lists the seats it stops and
  starts. reseat stops the org's active seats and starts them again with
  their recorded role, driver, and model; the plan approval and the
  reservations stay in effect. When it starts only seats other than the
  leader, it ends with a DECISION `ralph org send` for the leader, asking it
  to send the current TASK again; read the leader's pane, then send it.
- The seats of any other org: reseat refuses an org with no feature binding
  and prints a `ralph org stop` and a `ralph org spawn` command for each
  seat. Run them, the stop first.

Then check that no seat from before is left:

- `ralph org status --org-id <id> --live` lists each seat of the org with
  its state and its herdr pane, and for each active seat herdr's state and
  the seat's screen. Each seat you started again is `spawned (active)` on
  the pane the restart printed (the pane at the end of each of reseat's
  `started:` lines).
- `herdr pane list` lists the panes of the herdr server with their id,
  workspace, label, and cwd. A seat's pane sits in the org's workspace and
  is labelled with its seat id. Each pane of the org it lists should be the pane of an
  active seat in `ralph org status`. One that is not may still run a seat
  with its old arguments; close it with `herdr pane close <pane_id>`.

When a reseat stops part way, it exits 1, and its stdout names the seat
that failed (`stop failed:` or `start failed:`), the seats it left
(`left stopped:`, `left running:`, and the like), and, when seats are left
to start, `to start the seats left, run:` with a reseat command. Fix the
cause stderr names, run that command as printed, and check again as above.
A seat on a `left running:` line was not stopped, so it still runs with its
old arguments. The `/org` skill's reseat section (座席の立て直し) gives every
way a reseat can end.

## The cross-review of a codex reviewer seat

With `[org.review].cross_review` on, the leader of a feature org asks the
reviewer seat for a cross-review, and the reviewer seat reads the branch's
diff and the files it touches again on its own model and triages its
findings into a report. It calls no other CLI from inside the seat: no
`claude -p` and no nested `codex exec`. A codex reviewer seat under
`--sandbox workspace-write` (edits and autonomous) therefore needs only what
its review TASK already uses, `git` and its file tools in the feature
worktree, and the cross-review asks for no change to `codex_network_access`
or to any other sandbox setting.

The reviewer seat reviews on its own model whether or not the implementer
runs on the same model family, and the built-in `[org.role_models]` sets put
both on one family. To have the other family review, set
`[org.role_models].reviewer` to the driver the implementer does not use.
When the reviewer cannot finish reading the diff (`git diff` fails, for
example) and its git state did not move meanwhile, it writes its report with
`Reviewer status: incomplete (<reason>)` and returns the same value to the
leader. Such a run does not count toward
`[org.review].max_cross_review_cycles`, and the leader copies the value into
the `Cross-review:` line of the PR body, so the PR shows that the
cross-review did not finish, and why. If HEAD or the work tree moved between
the reviewer's first look at the git state and the report, it writes no
report and returns a BLOCKED with `XREVIEW: git_changed` instead, which the
leader raises to the inbox, and that run is not counted either. A reviewer seat started by an earlier
ralph keeps its earlier role prompt, which calls the other CLI; `ralph org
reseat` (above) starts the org's seats again with the current one.

## Scratch configuration

Keep the verification config separate from your project's `ralph.toml` and
pass it with `--config`; use a throwaway `--state-dir` as well.

```toml
# ralph-autonomous.toml
[org]
driver_pool = ["claude", "codex"]
max_seats = 2

[org.permissions]
default = "autonomous"
codex_verified = true
```

```toml
# ralph-edits.toml: same as above, plus
[org.permissions.roles]
reviewer = "edits"
```

## Procedure

Spawn one seat per mode and observe the points below. The manifest records
only the logical mode (`spawned` Details carry `permission_mode=<mode>`), not
the flags, so capture the child process command line with
`ps -axo args= | grep 'codex --sandbox'`.

### 1. autonomous

```sh
ralph org spawn --org-id perm-auto --id reviewer --role reviewer --driver codex \
  --model <slug> --cwd <scratch-cwd> --scope "<scratch-cwd>/**" \
  --config ralph-autonomous.toml --state-dir <scratch>/state-auto
```

- The child process arguments contain
  `--sandbox workspace-write --ask-for-approval never` followed by
  `-c sandbox_workspace_write.network_access=true` (the scratch config does
  not set `codex_network_access`, so it is `true`).
- The pane right after startup (`herdr pane read <pane> --lines 40`) shows no
  approval, trust, or login dialog. A model-retirement notice
  ("Try new model / Use existing model") may appear; it is not an approval
  prompt, and the choice is persisted to the codex config. codex can also
  switch a retiring model to its replacement without asking. ralph records
  what the seat really runs: the spawn receipt in
  `<state-dir>/model-receipts.jsonl` carries `reported_effective_model`,
  `honored=false` on a mismatch comes with a stderr warning, and
  `ralph doctor` lists pool slugs that codex has scheduled for retirement.
  While the dialog is open no turn has started, so the spawn receipt stays
  `unknown`; `ralph org stop` looks once more (bounded by the same
  up-to-8-s budget as spawn's poll) and, when it finds the seat's session
  record, appends the observed one (codex appears to create the record
  only once the dialog is answered; stop searches from the spawn date to
  the stop date, up to about a month).
- Write the TASK as a typed-protocol message (`ralph org send` validates it:
  `TYPE` must be one of the protocol's enum values, `TASK` needs a
  `TASK_ID`, and the body is capped at 2,000 characters), e.g. `task.txt`:

  ```
  TYPE: TASK
  TASK_ID: t-1

  This is a sandbox probe, not a review: do not run the gate scripts and do
  not review a diff.

  1. Create hello.txt in the working directory.
  2. Try to write to <throwaway path under $HOME>. If it is refused, do not
     retry and do not work around it; record the exact error text.
  3. Send a RESULT (TYPE: RESULT, TASK_ID: t-1) to leader through the agmsg
     skill's send script, with the outcome of each step.
  ```

  The seat is spawned with `--role reviewer` so that the
  `[org.permissions.roles] reviewer` override in `ralph-edits.toml` applies
  to it. The reviewer role template
  starts a review task by running the commands the TASK names, or both
  `run-static-verify.sh` and `run-test.sh` when it names none, and returns
  BLOCKED, without doing the work, when they fail or cannot run. The first
  paragraph of the TASK keeps that out of this probe, so leave it in.

- Send it. Every follow-up verb needs the same `--org-id`, `--config`, and
  `--state-dir` as the spawn, otherwise it fails with
  `org: --org-id is required` or reads the default state directory:

  ```sh
  ralph org send --org-id perm-auto --to reviewer --text "$(cat task.txt)" \
    --config ralph-autonomous.toml --state-dir <scratch>/state-auto
  ```

  `send` types the message, waits 750 ms (`--enter-delay-ms`), presses
  Enter once, and then checks through herdr that the seat left idle/done.
  Without that wait an idle codex seat often kept the pasted text in the
  composer unsent. If `send` warns that it could not confirm the submit,
  check the pane with `ralph org read`, and only if the message is still in
  the composer press Enter yourself (`herdr pane send-keys <pane> Enter`);
  ralph never resends Enter, because a blind keystroke could confirm an
  approval dialog. If `send` exits non-zero and prints a stderr note,
  follow the note rather than retrying blindly (a failure before anything
  was sent to the pane, such as a validation error, an unknown seat, or a
  `--timeout-ms` too small for the pause, prints no note and is safe to
  retry). Every note starts with reading the pane; the `ralph org read`
  command it prints carries `--state-dir` when you passed one. A herdr
  call that is cut off by `--timeout-ms` returns an error even if the text
  or the Enter already reached the pane, so in that case ralph reports what
  it does not know instead of guessing. Read what the note says before
  deciding whether to press Enter yourself or send again. Then wait for
  the seat:

  ```sh
  ralph org wait --org-id perm-auto --seat reviewer --until idle,done \
    --timeout-ms 300000 --config ralph-autonomous.toml --state-dir <scratch>/state-auto
  ```

- Expected: no approval prompt, the outside write fails with
  `operation not permitted`, and the RESULT arrives
  (`bash ~/.agents/skills/agmsg/scripts/history.sh ralph-perm-auto leader 20`).
  Confirm from the leader side that the `$HOME` target does not exist.

### 2. edits

Spawn a second seat under a different org id and state dir (reusing
`perm-auto` would hit the idempotent respawn path and re-observe the
autonomous seat):

```sh
ralph org spawn --org-id perm-edits --id reviewer --role reviewer --driver codex \
  --model <slug> --cwd <scratch-cwd> --scope "<scratch-cwd>/**" \
  --config ralph-edits.toml --state-dir <scratch>/state-edits
```

- The child process arguments contain `--sandbox workspace-write` and
  `-c sandbox_workspace_write.network_access=true`, and no
  `--ask-for-approval`.
- The same TASK (sent with `--org-id perm-edits --config ralph-edits.toml
  --state-dir <scratch>/state-edits`) completes the in-cwd edit without a
  prompt and the outside write is denied.
- A follow-up TASK that tells the seat to **request approval to run the
  denied command outside the sandbox** makes the pane show
  `Would you like to run the following command? … 1. Yes, proceed (y) /
  2. Yes, and don't ask again … / 3. No … (esc)`. Deny it with Esc and have
  the seat send its RESULT.
- **Caveat:** with `--ask-for-approval` omitted, edits inherits
  `approval_policy` from `~/.codex/config.toml`. If that is `"never"`, no
  prompt ever appears and edits behaves like autonomous. Record the
  `approval_policy`, `sandbox_mode`, and `writable_roots` in effect.
- **Caveat:** a prompt appears only when the model asks for an escalation
  (`on-request`). Writes outside the writable roots are always denied by the
  sandbox; whether the seat then asks is the model's call.

### 3. Cleanup

For each org you spawned, pass the same `--org-id`, `--config`, and
`--state-dir` as its spawn (a `status` run against the default state
directory shows an empty roster while the scratch seat keeps running):

```sh
ralph org stop --org-id perm-auto --seat reviewer \
  --config ralph-autonomous.toml --state-dir <scratch>/state-auto
ralph org disband --org-id perm-auto \
  --config ralph-autonomous.toml --state-dir <scratch>/state-auto
ralph org status --org-id perm-auto \
  --config ralph-autonomous.toml --state-dir <scratch>/state-auto   # no active seat
```

Repeat with `perm-edits` / `ralph-edits.toml` / `<scratch>/state-edits`.
`stop` closes the seat's pane and `disband` closes the org's herdr workspace;
if either exits 1, run it again once herdr answers. Then confirm
`herdr agent list` is empty and no throwaway file is left under `$HOME`.

## Verdicts and opt-out

- autonomous: no approval prompt + outside write denied + RESULT delivered →
  **pass**
- edits: in-cwd edit without a prompt + outside write denied + approval
  prompt when the model requests an escalation → **pass**. If an inherited
  `approval_policy = "never"` suppresses the prompt → **partial** (edits has
  no benefit over autonomous in that configuration; use autonomous or change
  the config).
- Either mode **inconclusive** (the seat never reaches the tool call, the
  pane cannot be read) → set `codex_verified = false`.

After a pass, keep `codex_verified = true` (the default, and what
`ralph init` writes). Earlier versions of `ralph init` wrote `false`, and
`ralph upgrade` leaves that line alone; set it to `true` by hand once this
recipe passes. When either mode fails or is inconclusive, set
`[org.permissions] codex_verified = false` in your `ralph.toml`, which keeps
every codex seat fail-closed to `guarded`. A pass holds only for the codex
version and the effective `~/.codex/config.toml` you checked with; re-run
this recipe after upgrading codex or changing `approval_policy`, `sandbox_mode`, or `writable_roots`.
`ralph doctor` does not validate `codex_verified`.

## Result on the meta-repo machine (2026-09-18)

codex-cli 0.154.0: autonomous = pass, edits = pass (mechanism). The CLI
`--sandbox workspace-write` also overrode a user config with
`sandbox_mode = "danger-full-access"`. Details and the environment problems
found on the way (alias collision, `ralph org send` needing an extra Enter,
the agmsg writable root, retired-model auto-migration) are in the evidence
file.
