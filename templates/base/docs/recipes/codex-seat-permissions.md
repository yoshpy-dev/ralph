# Codex seat permission modes: verify per machine, then opt in

`[org.permissions].codex_verified` defaults to `false`: codex seats accept only
`guarded` (the codex CLI's own interactive default) and reject `autonomous` /
`edits` with a fail-closed error. Setting it to `true` enables the mapping
autonomous → `--sandbox workspace-write --ask-for-approval never` and
edits → `--sandbox workspace-write`. How those flags behave depends on the
installed codex CLI version and on your own `~/.codex/config.toml`, so verify
them **on each machine** with this recipe before flipping the flag.

The meta-repo's own run (codex-cli 0.154.0, herdr 0.7.5, agmsg 1.1.13) is
recorded in `docs/evidence/codex-seat-permissions-2026-09-18.md`.

## Prerequisites

- `ralph doctor` passes for codex, herdr, and agmsg.
- The herdr socket is up: open the herdr TUI, or run `herdr server` headless.
- **Shell aliases break every codex spawn, not just this recipe.** If your
  shell rc defines `alias codex="codex -m ..."`, herdr sends the seat command
  to the pane's interactive shell as is, the alias expands, and codex exits
  with `error: the argument '--model <MODEL>' cannot be used multiple
  times`; `ralph org spawn` then reports `spawn_failed` after the
  `agent_start` timeout. Remove the alias, or start herdr from a HOME / rc
  that does not define it. The short form `-m` collides the same way. An
  alias that adds `--sandbox` (or `-s`) fails the same way on edits and
  autonomous seats, where ralph passes that flag itself; on a guarded seat
  the alias's sandbox silently applies. ralph passes `--ask-for-approval`
  to autonomous seats only, so an alias that adds it (or `-a`) fails on
  autonomous seats and silently sets the approval policy of edits and
  guarded seats. claude accepts a repeated flag and the last value
  wins (claude 2.1.274 CLI; not verified on a live seat): ralph always
  passes `--model` after the alias, so a `claude` alias that adds `--model`
  does not break the spawn, but ralph passes `--permission-mode` only on
  edits and autonomous seats, so a guarded claude seat runs with the
  alias's permission mode. Every other flag in the alias still reaches
  each seat. `ralph doctor`'s "Shell aliases (codex/claude)" check reports
  such aliases with the rc file and line (info when a claude alias only
  adds `--model` or when herdr is not installed, warn otherwise).
- **Add the agmsg database to the sandbox's writable roots.** Under
  `--sandbox workspace-write` codex can write only to the working directory
  and to `/tmp`-style temp roots; the agmsg SQLite database
  (`~/.agents/skills/agmsg/db/`) is outside them, so a seat's `send.sh` fails
  with `attempt to write a readonly database` and no RESULT reaches lead.
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
  "workspace-write"` and a role resolves to guarded. Only roles that may
  use a codex model count, and with codex missing from `[org].driver_pool`
  or `[org].model_pool` nothing is needed. `/tmp` and `$TMPDIR` count as
  writable unless `exclude_slash_tmp` / `exclude_tmpdir_env_var` is set.
  The check reads only the user-level `config.toml`; profiles,
  project-level config, and `-c` overrides are not evaluated.
- **Keep the scratch working directory out of `/tmp`.** `/tmp` is itself a
  writable root, so a target under it does not test the sandbox boundary.
  Use a throwaway directory under `$HOME` (`git init` it) for the seat's cwd
  and a second throwaway file under `$HOME` as the "outside" target.

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
  `--sandbox workspace-write --ask-for-approval never`.
- The pane right after startup (`herdr pane read <pane> --lines 40`) shows no
  approval, trust, or login dialog. A model-retirement notice
  ("Try new model / Use existing model") may appear; it is not an approval
  prompt, and the choice is persisted to the codex config.
- Write the TASK as a typed-protocol message (`ralph org send` validates it:
  `TYPE` must be one of the protocol's enum values, `TASK` needs a
  `TASK_ID`, and the body is capped at 2,000 characters), e.g. `task.txt`:

  ```
  TYPE: TASK
  TASK_ID: t-1

  1. Create hello.txt in the working directory.
  2. Try to write to <throwaway path under $HOME>. If it is refused, do not
     retry and do not work around it; record the exact error text.
  3. Send a RESULT (TYPE: RESULT, TASK_ID: t-1) to lead through the agmsg
     skill's send script, with the outcome of each step.
  ```

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
  (`bash ~/.agents/skills/agmsg/scripts/history.sh ralph-perm-auto lead 20`).
  Confirm from the lead side that the `$HOME` target does not exist.

### 2. edits

Spawn a second seat under a different org id and state dir (reusing
`perm-auto` would hit the idempotent respawn path and re-observe the
autonomous seat):

```sh
ralph org spawn --org-id perm-edits --id reviewer --role reviewer --driver codex \
  --model <slug> --cwd <scratch-cwd> --scope "<scratch-cwd>/**" \
  --config ralph-edits.toml --state-dir <scratch>/state-edits
```

- The child process arguments contain `--sandbox workspace-write` and no
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

Repeat with `perm-edits` / `ralph-edits.toml` / `<scratch>/state-edits`, close
the panes, then confirm `herdr agent list` is empty and no throwaway file is
left under `$HOME`.

## Verdicts and opt-in

- autonomous: no approval prompt + outside write denied + RESULT delivered →
  **pass**
- edits: in-cwd edit without a prompt + outside write denied + approval
  prompt when the model requests an escalation → **pass**. If an inherited
  `approval_policy = "never"` suppresses the prompt → **partial** (edits has
  no benefit over autonomous in that configuration; use autonomous or change
  the config).
- Either mode **inconclusive** (the seat never reaches the tool call, the
  pane cannot be read) → keep `codex_verified = false`.

After a pass, set `[org.permissions] codex_verified = true` in your
`ralph.toml`. The opt-in is tied to the codex version and the effective
`~/.codex/config.toml` you verified with; re-run this recipe after upgrading
codex or changing `approval_policy`, `sandbox_mode`, or `writable_roots`.
`ralph doctor` does not validate `codex_verified`.

## Result on the meta-repo machine (2026-09-18)

codex-cli 0.154.0: autonomous = pass, edits = pass (mechanism). The CLI
`--sandbox workspace-write` also overrode a user config with
`sandbox_mode = "danger-full-access"`. Details and the environment problems
found on the way (alias collision, `ralph org send` needing an extra Enter,
the agmsg writable root, retired-model auto-migration) are in the evidence
file.
