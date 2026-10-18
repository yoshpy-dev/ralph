# Live check: the push wake of a headless codex director (stage 8b-1, S5)

Plan: `docs/plans/active/2026-10-18-org-director-wake.md` (D11, AC10).
Two runs against an isolated herdr server and an isolated ledger: a codex
director registered with `wake = ["push"]` and woken while idle, then one
registered with `wake = ["pull", "push"]` and woken while it was working on a
`ralph org wait --inbox` it had been asked to run.

Values left out: the launch tokens are `<token-d1>` and `<token-d2>`, the
herdr panes (the wake targets) are `<pane-d1>` and `<pane-d2>`, and `$HOME`
is `~`.

## Setup

- ralph: built from branch `feat/org-director-wake` at `88e433b2` (the code of
  S3b, `b716d041`, plus a plan note): `go build -o /tmp/rwk/bin/ralph ./cmd/ralph`
- herdr 0.7.5 (`/opt/homebrew/bin/herdr`) and codex-cli 0.160.1 (its native
  binary, `~/.local/share/mise/installs/npm-openai-codex/0.160.1/node_modules/.mise/@openai+codex-darwin-arm64@0.160.1-darwin-arm64/node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/bin/codex`),
  symlinked into `/tmp/rwk/bin` with ralph and a stub `osascript` that only
  appends its arguments to `/tmp/rwk/osascript.log`
- The herdr server was started with `env -i HOME=/tmp/rwk/home
  PATH=/tmp/rwk/bin:/usr/bin:/bin SHELL=/bin/zsh XDG_CONFIG_HOME=/tmp/rwk/cfg
  XDG_STATE_HOME=/tmp/rwk/state HERDR_SOCKET_PATH=/tmp/rwk/h.sock herdr server`
  (it printed `api socket: /tmp/rwk/h.sock`). Every ralph and herdr command ran
  under `env -i HOME=/tmp/rwk/home PATH=/tmp/rwk/bin:/usr/bin:/bin
  HERDR_SOCKET_PATH=/tmp/rwk/h.sock`, with `--state-dir /tmp/rwk/ledger`, and
  the org verbs also with `--config /tmp/rwk/proj/ralph.toml`. No herdr
  server of the user ran before or after
- The fake HOME: `/etc/zshenv` sets `ZDOTDIR=$HOME/.config/zsh`, so
  `/tmp/rwk/home/.config/zsh/{.zshenv,.zprofile,.zshrc}` each held only
  `export PATH=/tmp/rwk/bin:$PATH`. The pane's zsh therefore had no `codex`
  alias, and the director's own `ralph` was the build above. Its
  `.codex/config.toml` set `model = "gpt-6-astra"`, `model_reasoning_effort =
  "low"`, `sandbox_mode = "workspace-write"`, `approval_policy =
  "on-request"`, writable roots `/tmp/rwk/ledger`, and the project
  `/tmp/rwk/proj` trusted. `auth.json` was a 0600 copy, removed with
  `/tmp/rwk`
- `/tmp/rwk/proj`: `git init` with only `ralph.toml` and `.agents/skills/` of
  the branch (`git archive HEAD .agents/skills`). The repository's
  `.codex/config.toml` (`danger-full-access`) was left out on purpose.

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
wake = ["push"]
```

`wake_cmd` is empty, so the watch's sender is the herdr default: input to the
pane of a headless director in herdr.

## Run 1: `wake = ["push"]`, the director idle

```
$ ralph org watch --all --detach …
started ralph org watch --all (pid 45166); log: /tmp/rwk/ledger/watch.log
rc=0
$ ralph org director start --driver codex --model gpt-6-astra …
started d1 (driver=codex, model=gpt-6-astra, permission=guarded, pane=<pane-d1>, workspace=w1)
talk to the director in herdr: pane <pane-d1>, in the tab director_<token-d1> of the workspace labelled ralph_director_<token-d1>
stop it with: ralph org director stop --id d1 --state-dir '/tmp/rwk/ledger'
rc=0
```

codex showed no startup dialog. In its first turn (05:54:48 to 05:55:12) it
read its role prompt, ran `ralph org director status`, read the director
skill, ran `ralph org inbox` (`no open or acked inbox items`), and ended the
turn. It did not start a `ralph org wait --inbox`.

```
$ ralph org escalate --org-id lf1 --text 'TYPE: QUESTION

Live check of the push wake: please ack this item, then resolve it with the note "live check".' …
ORG ESCALATION RECORDED: org=lf1 item=e1 type=QUESTION -- left in /tmp/rwk/ledger/inbox.jsonl for the director; …
escalated e1 (org=lf1 type=QUESTION)
rc=0
```

The watch's next cycle, 10 s later, typed the line into the pane and recorded
it:

```
{"ts":"2026-10-18T05:55:56Z","id":"e1","event":"escalated","org_id":"lf1","type":"QUESTION","body":"…","delivery":"deadline"}
{"ts":"2026-10-18T05:56:06Z","id":"e1","event":"woken","director_id":"d1","via":"herdr"}
{"ts":"2026-10-18T05:56:13Z","id":"e1","event":"acked"}
```

codex's session record (`/tmp/rwk/home/.codex/sessions/…`, read with a
script that prints the user messages, the commands, and the replies):

```
05:56:06 USER: New org inbox items for the director: e1. Read them with ralph org inbox and handle each as the director skill says.
05:56:08 ASSISTANT: 受信箱の `e1` を読み、種類と内容に応じて処理します。
05:56:10 CALL: ralph org inbox --state-dir '/tmp/rwk/ledger' --config '/tmp/rwk/proj/ralph.toml'
         CALL: ralph org inbox show e1 --state-dir '/tmp/rwk/ledger' …
05:56:13 CALL: ralph org inbox ack e1 --state-dir '/tmp/rwk/ledger' --config '/tmp/rwk/proj/ralph.toml'
05:56:19 CALL: ralph status --state-dir '/tmp/rwk/ledger', ralph org merge-order …
05:56:26 ASSISTANT: `e1` を読み、acked にしました。push による起床履歴も確認できました。  本文は「動作確認として `live check` の注記で解決してほしい」という依頼です。解決してよいですか？ …「判断が要る件(QUESTION、BLOCKED): 人に聞く」と定めているため、現在は未解決のままです。
```

The pane at the end of that turn:

```
› New org inbox items for the director: e1. Read the…
  └ director: attached d1 (driver=codex, headless run…
…
• e1 を読み、acked にしました。push による起床履歴も確
  認できました。

  本文は「動作確認として live check の注記で解決してほ
  しい」という依頼です。解決してよいですか？
…
  Worked for 20s • 14:56
```

The line started a new turn of an idle codex. The director acked the item
and did not act on the body's request on its own: it asked the human, as the
skill says for a QUESTION. Typed into the pane as the human (`herdr pane
send-text`, then Enter): `はい、e1 は note "live check" で resolve してください。`
codex ran `ralph org inbox resolve e1 --note 'live check'`
(`{"ts":"2026-10-18T05:57:28Z","id":"e1","event":"resolved","note":"live check"}`).
Later cycles did not wake d1 again: e1 had a woken event.

```
$ ralph org inbox show e1 …
id: e1
state: resolved
…
events:
  2026-10-18T05:55:56Z escalated
  2026-10-18T05:56:06Z woken director_id=d1 via=herdr
  2026-10-18T05:56:13Z acked
  2026-10-18T05:57:28Z resolved note=live check
```

## Run 2: `wake = ["pull", "push"]`, the line arrives while the director works

```
$ ralph org director stop --id d1 --state-dir /tmp/rwk/ledger
stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)
the director is no longer attached
rc=0
```

`wake` in `/tmp/rwk/proj/ralph.toml` was changed to `["pull", "push"]`, and
the watch was restarted (`ralph org watch --stop`, then `--all --detach`, pid
63863).

```
$ ralph org director start --driver codex --model gpt-6-astra …
started d2 (driver=codex, model=gpt-6-astra, permission=guarded, pane=<pane-d2>, workspace=w2)
…
rc=0
```

d2's first turn ended like d1's (05:59:08 to 05:59:29), without a wait. The
pane then showed `Reconnecting to server… (52s)` for under a minute and
recovered by itself. Typed into the pane as the human:
`受信箱の処理の節のとおり、ralph org wait --inbox --timeout-ms 300000 で open の件を待ち、届いたら処理してください。`
codex started that wait at 06:00:54 as a background terminal of the running
turn (`Working (20s • esc to interrupt) · 1 background ter…`). Then:

```
$ ralph org escalate --org-id lf1 --text 'TYPE: QUESTION

Second live check: the director is waiting on the inbox. Please ack this item and ask the human before resolving.' …
escalated e2 (org=lf1 type=QUESTION)
rc=0
```

```
{"ts":"2026-10-18T06:01:18Z","id":"e2","event":"escalated",…,"delivery":"deadline"}
{"ts":"2026-10-18T06:01:25Z","id":"e2","event":"acked"}
{"ts":"2026-10-18T06:01:25Z","id":"e2","event":"woken","director_id":"d2","via":"herdr"}
```

The wait (which reads the inbox every second) returned e2 first, and codex
read and acked it (`ralph org inbox ack e2` at 06:01:24). The watch's cycle
read the inbox while e2 was still open and typed the line in the same second;
its woken event was written after the ack. `AppendWoken` records a woken
event in any state (D10: the wake is recorded as a fact and changes no
state), so the line named an item already acked.

The pane, read every 2 s from the escalate:

```
=== 2 s
• Ran ralph org wait --inbox --timeout-ms 300000 --st…
  └ e2    lf1    QUESTION    Second live check: the d…
• Working (31s • esc to interrupt)
=== 6 s (the line is in the composer while the turn runs)
• Working (35s • esc to interrupt) · 1 background ter…
› New org inbox items for the director: e2. Read them
  with ralph org inbox and handle each as the
  director skill says.
  tab to queue message
=== 8 s (Enter was pressed; the composer is empty again)
  with ralph org inbox and handle each as the
  director skill says.
• Working (38s • esc to interrupt)
› Ask Codex to do anything
```

The session record shows where the line went: into the running turn, not a
new one.

```
06:00:49 task_started 01a14d99-38f9-…
06:00:50 user msg     01a14d99-38f9-…  受信箱の処理の節のとおり、ralph org wait --inbox --timeout-ms 3…
06:01:27 user msg     01a14d99-38f9-…  New org inbox items for the director: e2. Read the…
06:01:41 task_complete 01a14d99-38f9-…
```

After the line, in the same turn, codex said it had acked e2 (06:01:32), ran
`ralph status` and the inbox again (06:01:34), and ended the turn by asking
the human whether to answer the leader (06:01:41). It handled e2 once: one
`acked` event, no second ack. Typed into the pane as the human:
`はい、e2 は note "live check 2" で resolve してください。leader への送信は不要です。`
(`{"ts":"2026-10-18T06:02:50Z","id":"e2","event":"resolved","note":"live check 2"}`).

```
$ ralph org inbox show e2 …
events:
  2026-10-18T06:01:18Z escalated
  2026-10-18T06:01:25Z acked
  2026-10-18T06:01:25Z woken director_id=d2 via=herdr
  2026-10-18T06:02:50Z resolved note=live check 2
$ ralph org director status …
director: attached d2 (driver=codex, headless running, model=gpt-6-astra, pane <pane-d2>, workspace w2, attached at 2026-10-18T05:59:07Z, wake=pull,push to <pane-d2>) -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
…
wake: pull,push
…
launch: herdr
state: running
…
$ ralph status --state-dir /tmp/rwk/ledger
…
org watch: running (pid 63863, mark updated 2026-10-18T06:03:26Z)
director: attached d2 (…, wake=pull,push to <pane-d2>) -- …
```

## The ledger after the two runs

`director.jsonl`:

```
2026-10-18T05:54:37Z attached d1 wake=push launch=herdr
2026-10-18T05:54:37Z workspace_created d1
2026-10-18T05:54:37Z pane_created d1
2026-10-18T05:54:41Z launched d1
2026-10-18T05:59:02Z detached d1
2026-10-18T05:59:02Z stopped d1
2026-10-18T05:59:07Z attached d2 wake=pull,push launch=herdr
2026-10-18T05:59:07Z workspace_created d2
2026-10-18T05:59:07Z pane_created d2
2026-10-18T05:59:10Z launched d2
2026-10-18T06:03:49Z detached d2
2026-10-18T06:03:49Z stopped d2
```

`watch.log` held only the two start lines (`watching every org (interval=10s
once=false state-dir=/tmp/rwk/ledger (source: flag))`): no wake failed and
none was skipped with a line. `/tmp/rwk/osascript.log` was never created:
neither item reached its ack deadline, and nothing was sent to the human.

## Cleanup

- `ralph org director stop --id d2` rc 0 (`pane=closed, workspace=closed`),
  `ralph org watch --stop` rc 0, `herdr workspace list` empty, `herdr server
  stop` rc 0
- Three processes were still running from `/tmp/rwk/home/.codex/packages/app-server-daemon/releases/0.163.0-aarch64-apple-darwin/bin/`:
  `codex app-server daemon pid-update-loop` (parent pid 1), `codex app-server
  --listen unix:// --managed-daemon`, and `codex-code-mode-host`. codex 0.160.1
  had downloaded that daemon into the fake `CODEX_HOME` and started it; it
  outlived the director's TUI and the herdr server. They were stopped by pid,
  then `/tmp/rwk` (the `auth.json` copy included) was removed. Afterwards no
  process named `/tmp/rwk`
- The user's herdr state, read only: `~/.config/herdr/session.json` sha256
  prefix 004173699fb3c932 (mtime Sep 19 14:09:37) and
  `~/.config/herdr/herdr-server.log` sha256 prefix fe019d6d40e422ca (mtime
  Oct 1 14:35:45), the same values before and after

## Findings

- A codex director (0.160.1) takes the wake line typed into its idle pane as
  a new turn and handles it by the skill: it reads the inbox, acks, and asks
  the human for a QUESTION. The body's request was not carried out on its
  own.
- When the line and Enter arrive while a turn runs, codex shows `tab to queue
  message` and, on Enter, puts the line into the running turn (the same
  `turn_id`), not into a queue for the next turn. The line was not lost, and
  codex read the inbox again in that turn. The Claude Code side was not run
  here.
- With pull and push, the same item can be picked up by the director's wait
  and named by the wake line within a second. The director acked it once.
  The line may name an item that is already acked (the woken event then
  follows the acked one).
- In both runs codex ended its first turn without starting
  `ralph org wait --inbox`, although the skill says to wait on the inbox.
  With `wake = ["push"]` that is what is wanted; with pull, the human or the
  skill has to start the wait.
- codex fetched a curated plugin into the fake `CODEX_HOME`
  (`plugins/cache/openai-curated-remote/build-ios-apps/…/.mcp.json`), and its
  MCP server `xcodebuildmcp` failed to start (`No such file or directory`):
  the pane's warning (`⚠ 1 warning`). Not related to ralph.

## What this run does not cover

- `[org.director].wake_cmd`: the unit and CLI tests cover it with programs
  the tests write (S3b). Orca's `orca terminal send` and tmux `send-keys`
  were not run against a real terminal
- A Claude Code director, and a promoted session attached with
  `--wake push --wake-target`
- A wake that fails (a pane herdr no longer has, a herdr error) and the
  5-minute gap after it, which the tests in `internal/org/director_wake_test.go`
  and `watch_all_test.go` cover with a fake herdr
- An item that reaches its ack deadline after a wake (D10); the deadline
  tests cover that a woken item still comes due
