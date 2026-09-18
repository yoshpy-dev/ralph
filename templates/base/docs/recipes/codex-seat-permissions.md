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
  shell rc defines `alias codex="codex -m ..."` (or a `claude` alias that adds
  `--model`), herdr sends the seat command to the pane's interactive shell as
  is, the alias expands, and codex exits with
  `error: the argument '--model <MODEL>' cannot be used multiple times`.
  `ralph org spawn` then reports `spawn_failed` after the `agent_start`
  timeout. Remove the alias, or start herdr from a HOME / rc that does not
  define it.
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
  meta-repo run used an absolute path; a `~` prefix was not tested.
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
- Send a TASK with `ralph org send --to reviewer`. When the seat is idle the
  pasted text can stay in the composer unsent; a few seconds later send
  `herdr pane send-keys <pane> Enter` once.
- TASK content: create one file inside the cwd → try to write to the
  throwaway path under `$HOME` (tell the seat explicitly **not to retry or
  work around a denial**) → send the RESULT over agmsg (the seat runs the
  skill's `send.sh`).
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
- The same TASK completes the in-cwd edit without a prompt and the outside
  write is denied.
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

`ralph org stop --seat reviewer` → `ralph org disband` →
`ralph org status --org-id <id>` shows no active seat → `herdr agent list`
is empty → close the pane. Confirm no throwaway file is left under `$HOME`.

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
