# Org recheck: two feature orgs from the director to the merge, on main ffb3e271

Plan: `docs/plans/active/2026-10-24-org-recheck.md` (D1 to D12; AC1 to AC5,
AC8, AC9). Spec: `docs/specs/2026-10-07-org-multi-org-director.md`. The
trial this rechecks: `docs/evidence/org-trial-2026-10-20.md` (f0668004,
before the fixes of its findings). This run used a binary built from main at
ffb3e271, which holds every fix of the trial's findings (#244 to #254), and
followed `docs/recipes/isolated-herdr-server.md` from its first step. It ran
from the start of S1 (2026-10-24T09:40:43Z) to the final snapshot
(2026-10-25T14:24:59Z); the orgs ran from their start (02:23:54Z) to the
second merge (14:22:31Z). All times are UTC, and a time without a date is on
2026-10-25.

The plan's own session (Claude Code) was the director, promoted with the
`/director` skill. The user was the human: they answered the folder trust,
approved the split plan and both feature plans, decided the first merge, and
let the director make the second.

Sources: the sanitized ledger snapshot `docs/evidence/org-recheck-2026-10-24/`
(Snapshot, below), the director's running notes and its intervention log,
both kept in the session's scratchpad during the run (the log is committed in
the snapshot, the notes are not), the git history of main, and the two PRs on
GitHub. Every table under Snapshot, Setup, Two orgs, Fix effects, Merges,
Numbers, and Gates that comes from a file of the snapshot is followed by the
command that recomputes it: run the `sh` block from the repository root, and
it prints the body of the table right above it, or the `text` block right
below it (tested with jq 1.8.2 and macOS's awk, sort, and grep). Exit codes,
command output that was not saved, load averages, and what a pane showed come
from the director's notes.

## Snapshot (D12, AC9)

The director copied the recheck ledger (`/tmp/ralph-iso/ralph/.harness/state/org/`)
twice. The first copy (14:23:07Z) came after both orgs had disbanded and
before the detach. The final copy (14:24:59Z), committed here, came after the
detach of the director, the stop of the watch, and the stop of the isolated
herdr server, so `director.jsonl` has its `detached` line and
`watch-alive.json` says `stopped`.

| File | Content |
|------|---------|
| `manifest.jsonl` | the org saga manifest, 80 events |
| `inbox.jsonl` | the org inbox, items e1 to e12 (36 events) |
| `director.jsonl` | the director registration d1: `attached` and `detached` |
| `director-wait.json` | the mark of the director's last wait (14:03:51Z to 14:23:36Z, `end` watch; RC9) |
| `model-receipts.jsonl` | the model receipts: 10 of seat spawns (4 of them refused spawns, RC10) and 13 of the on-demand watcher |
| `watch-alive.json`, `watch.log`, `watch-status-<org>.json` | the cross-org watch's liveness mark, log, and state per org |
| `merge-guarantee.json` | the merge guarantee cached at 02:17:31Z |
| `splits/recheck.md` | the split plan |
| `plans/2026-10-25-<feature>.md` | the two feature plans as approved |
| `org-msgs/e<N>-decision.txt` | the bodies of the director's ten DECISIONs to the leaders, e1 to e10, as sent |
| `merge-fix-backfill-na.out`, `regate-fix-hooks-matcher.out`, `merge-fix-hooks-matcher.out` | the output of the first merge, of the regate, and of the second merge |
| `ralph-status-final.txt` | `ralph status` after the watch stop |
| `osascript.log` | the notification shim's log, 12 calls |
| `agmsg-summary.tsv` | the private agmsg's 34 messages, counted by team, sender, receiver, and TYPE line, with the unread count (RC1) |
| `seat-usage.tsv` | tokens per seat session, from the Claude Code session records (message ids deduplicated) |
| `watcher-usage.txt` | tokens of the watcher and of the S1 probe, the cache-write split, and the list prices used |
| `interventions.log` | the director's intervention log, I0a, I0b, and I1 to I3 |
| `gate-excerpts.md` | excerpts of the two gate logs and the regate log, with their check-sync lines |
| `setup/run`, `setup/claude-env`, `setup/osascript` | the wrapper, the `CLAUDE_ENV_FILE` file, and the shim, as written (mode 644 here; they ran as executables) |
| `setup/agmsg-VERSION`, `setup/herdr-server.out`, `setup/scratch-automerge.toml` | the private agmsg copy's `VERSION`, the server's startup output, and the scratch `ralph.toml` of D7's check |

Not committed: the three gate logs (about 347 KB each; `gate-excerpts.md`
has their headers, start times, verdict lines, and check-sync lines); the herdr server log (its
`WARN` lines were only about agents of herdr's remote manifest that it does
not know, qwen, muse, and letta, every 30 min, per the notes); the split
plan's visual page `recheck.html` that the human saw at I1 (its source,
`splits/recheck.md`, is committed); and the first copy. Compared with `cmp`,
the first copy's ledger files equal the final ones except `director.jsonl`
(no `detached` line), `director-wait.json`, and `watch-alive.json` (still
`running`). The role prompts ralph wrote under `prompts/` were not copied.
The final copy has no `escalations.jsonl`: ralph writes a line there only
when it sends something to the human (an item sent at once, a deadline's
send, `ralph org inbox notify`, or an escalation the inbox could not take;
the help of `ralph org escalate`, `ralph org watch`, and
`ralph org inbox notify`, and `internal/org/deadline.go`), and nothing went
to the human (Fix effects, F6 and F13).

Sanitizing, done with a script kept outside the repository by plain text
substitution, so each JSON line stays as ralph wrote it:

- herdr pane ids became `pane-N`, numbered in order of first appearance in
  `manifest.jsonl`, and workspace ids `ws-N`; a tab id and a terminal id
  took the number of the pane they held (`tab-N`, `term-N`), read from the
  `agent_get` records of the watch-status files (each seat had its own tab
  and terminal)
- Claude Code session ids (the `agent_session` values in the watch-status
  files) became `session-N`, numbered in order of appearance in
  `watch-status-backfill-na.json`, then `watch-status-hooks-matcher.json`
- the OS login name became `user-1` (`by=` in the manifest, `user` in
  `director.jsonl`)
- the home directory became `~` (only `setup/claude-env` held it; that file,
  run as written here, would not expand the `~` inside its double quotes,
  and the one the seats read held the absolute path)
- kept: commit SHAs, plan digests, `content=` sha256 values, PR numbers,
  timestamps, pids, alert ids, the wait id (a pid and a start time), and the
  paths of the run's clone under `/private/tmp/ralph-iso/` (plan D1's place,
  removed in S7) as they were. The copied files held no U+FFFD and no
  agmsg-style hex agent name

Which org and seat each label was, from the manifest's first and last event
per label:

| Label | Org | Seat | First event | Last event |
|-------|-----|------|-------------|------------|
| ws-1 | backfill-na | workspace | 2026-10-25T02:23:54Z | org_workspace_closed at 2026-10-25T03:10:18Z |
| pane-1 | backfill-na | leader | 2026-10-25T02:23:54Z | stopped at 2026-10-25T03:10:18Z |
| ws-2 | hooks-matcher | workspace | 2026-10-25T02:24:05Z | org_workspace_closed at 2026-10-25T03:16:47Z |
| pane-2 | hooks-matcher | leader | 2026-10-25T02:24:05Z | stopped at 2026-10-25T03:16:47Z |
| pane-3 | hooks-matcher | implementer | 2026-10-25T02:31:53Z | stopped at 2026-10-25T02:49:28Z |
| pane-4 | backfill-na | implementer | 2026-10-25T02:31:57Z | stopped at 2026-10-25T03:09:47Z |
| pane-5 | hooks-matcher | reviewer | 2026-10-25T02:32:12Z | stopped at 2026-10-25T03:02:27Z |
| pane-6 | backfill-na | reviewer | 2026-10-25T02:32:13Z | stopped at 2026-10-25T03:09:48Z |

```sh
jq -sr 'reduce (.[] | select(.pane_id != null)) as $e ({}; .[$e.pane_id] += [$e]) | to_entries[] | .value as $v | "| \(.key) | \($v[0].org_id) | \(if $v[0].seat_id == "" then "workspace" else $v[0].seat_id end) | \($v[0].ts) | \($v[-1].event) at \($v[-1].ts) |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

Every seat ran once: no seat was spawned twice, and no spawn failed. The
herdr agent, tab, terminal, and session of each seat, from the watch's last
`agent_get` of it (03:09:50Z for backfill-na, 03:16:20Z for hooks-matcher):

| Org | Seat | herdr agent | Pane | Tab | Terminal | Session | Status |
|-----|------|-------------|------|-----|----------|---------|--------|
| backfill-na | implementer | backfill-na_implementer | pane-4 | tab-4 | term-4 | session-1 | done |
| backfill-na | leader | backfill-na_leader | pane-1 | tab-1 | term-1 | session-2 | working |
| backfill-na | reviewer | backfill-na_reviewer | pane-6 | tab-6 | term-6 | session-3 | done |
| hooks-matcher | implementer | hooks-matcher_implementer | pane-3 | tab-3 | term-3 | session-4 | done |
| hooks-matcher | leader | hooks-matcher_leader | pane-2 | tab-2 | term-2 | session-5 | working |
| hooks-matcher | reviewer | hooks-matcher_reviewer | pane-5 | tab-5 | term-5 | session-6 | done |

```sh
for o in backfill-na hooks-matcher; do jq -r --arg o "$o" '.seat_snapshots | to_entries[] | (.value.agent_get | fromjson | .result.agent) as $a | "| \($o) | \(.key) | \($a.name) | \($a.pane_id) | \($a.tab_id) | \($a.terminal_id) | \($a.agent_session.value) | \($a.agent_status) |"' "docs/evidence/org-recheck-2026-10-24/watch-status-$o.json"; done
```

In `seat-usage.tsv` each seat has one row (`implementer-1`, `leader-1`, and
`reviewer-1` of its org): each session id is the file name of a session
record under
`~/.claude/projects/-private-tmp-ralph-iso-ralph--claude-worktrees-org-<org>/`
whose first assistant message has that row's `first` time (checked on
2026-10-25 after the run).

`ralph org send` records a message body in the `sent` event cut at a fixed
length. Unlike the trial's four, no `sent` line here holds an escape of
U+FFFD, although 14 of the 17 cut their body, most of them in Japanese text
(F12's fix, #246):

```sh
jq -sr 'map(select(.event == "sent")) | "sent events \(length), body cut \(map(select(.details | endswith("...(truncated)"))) | length), holding U+FFFD \(map(select(.details | explode | index([65533]) != null)) | length)"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

```text
sent events 17, body cut 14, holding U+FFFD 0
```

## Setup (AC1)

- Clone (D1): `/tmp/ralph-iso/ralph`, `git clone ssh://git@github.com/yoshpy-dev/ralph.git`
  at ffb3e271 (main after PR #254). The clone had no `ralph.toml` until
  14:03Z (D7, under Merges), so every setting was the default: `max_orgs` 10,
  `max_total_seats` 30, `max_seats` 5, `ack_minutes` 10, `resolve_minutes`
  60, approval `per-plan`, the director enabled with wake `pull`, the
  watcher on `haiku`, a 30 s watch interval, `deadman_minutes` 10,
  `stall_minutes` 15 (`internal/config/config.go`, `Default`)
- Binary: `go build -o /tmp/ralph-iso/bin/ralph ./cmd/ralph` in the clone at
  ffb3e271, go1.26.0 (`ralph version` prints `dev (unknown unknown)`)
- Tools, as the plan's Assumptions list them: herdr 0.7.5, agmsg 1.5.1,
  Claude Code 2.1.291, gh 2.102.0 with the `yoshpy-dev` account active,
  git 2.49.0, go 1.26.0, on macOS. The notes confirm go (the build), agmsg
  (the copy's `VERSION`, which `ralph doctor` printed too), and gh's account
  (the probe). herdr's and Claude Code's versions were not read again during
  the run; git, gh, and go printed the versions above on this machine after
  the run
- The wrapper (`setup/run`): the recipe's wrapper with two more lines,
  `RALPH_ORG_AGMSG_HOME=/tmp/ralph-iso/agmsg` (D2, a deviation from the
  recipe) and `GOFLAGS=-timeout=30m` (D1; the recipe's "Add any other
  variable"). `ps eww` on the server listed only these variable names:
  `HOME`, `USER`, `LOGNAME`, `SHELL`, `LANG`, `TERM`, `TMPDIR`, `PATH`,
  `XDG_CONFIG_HOME`, `XDG_STATE_HOME`, `HERDR_SOCKET_PATH`, `CLAUDE_ENV_FILE`,
  `GOFLAGS`, `RALPH_ORG_AGMSG_HOME`, `GH_CONFIG_DIR`, and `GIT_CONFIG_GLOBAL`.
  Their values: `HOME`, `USER`, `LOGNAME`, `LANG`, and `TMPDIR` as in the
  director's shell; `SHELL=/bin/bash`; `TERM=xterm-256color`;
  `XDG_CONFIG_HOME=/tmp/ralph-iso/xdg-config`;
  `XDG_STATE_HOME=/tmp/ralph-iso/xdg-state`;
  `HERDR_SOCKET_PATH=/tmp/ralph-iso/h.sock`;
  `CLAUDE_ENV_FILE=/tmp/ralph-iso/claude-env`; `GOFLAGS=-timeout=30m`;
  `RALPH_ORG_AGMSG_HOME=/tmp/ralph-iso/agmsg`; `GH_CONFIG_DIR=~/.config/gh`;
  `GIT_CONFIG_GLOBAL=~/.config/git/config`; and `PATH` with
  `/tmp/ralph-iso/bin` first, then claude's install
  (`~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/node_modules/.bin`),
  node 24 and python 3.12 under `~/.local/share/mise/installs/`,
  `~/.local/bin`, `/opt/homebrew/bin`, and the system directories (no codex).
  No `SSH_AUTH_SOCK` and no `GH_TOKEN`. `setup/claude-env` exports the same
  `PATH`
- Shim (D3): `/tmp/ralph-iso/bin/osascript` (`setup/osascript`), the
  recipe's script: it appends the UTC time and its arguments to
  `/tmp/ralph-iso/osascript.log` and exits 0. In the D8 probe, claude's Bash
  tool in a pane resolved `osascript` to the shim. `osascript.log` holds
  every `osascript` call of the run that went through `PATH`
- Private agmsg (D2): a copy of `scripts`, `plugins`, `types`, `SKILL.md`,
  and `VERSION` (1.5.1) from `~/.agents/skills/agmsg` in
  `/tmp/ralph-iso/agmsg`, named by the wrapper's `RALPH_ORG_AGMSG_HOME`
  rather than by `[org].agmsg_home` in the clone's `ralph.toml` (the
  feature worktrees do not get the clone's untracked `ralph.toml`). ralph
  used the copy; the seats' own Claude Code did not (RC1)
- Models (D5): both leaders started with `--model opus`. The split plan's
  preamble told the leaders to spawn the implementer with `--model opus` and
  the reviewer with `--model sonnet`, and they did (F5). The watcher ran on
  the default `haiku` (`model-receipts.jsonl`, `commanded_model`). Both
  leaders' status line read Opus 5.5, effort low (RC12). Every seat ran with
  the default permission envelope

| Org | Seat | Model commanded (`spawn_started`) | Spawns refused (`rejected`) | Spawns started |
|-----|------|-----------------------------------|-----------------------------|----------------|
| backfill-na | implementer | opus | 1 | 1 |
| backfill-na | leader | opus | 0 | 1 |
| backfill-na | reviewer | sonnet | 1 | 1 |
| hooks-matcher | implementer | opus | 1 | 1 |
| hooks-matcher | leader | opus | 0 | 1 |
| hooks-matcher | reviewer | sonnet | 1 | 1 |

```sh
jq -sr 'map(select(.event == "spawn_started" or .event == "rejected")) | group_by([.org_id, .seat_id])[] | "| \(.[0].org_id) | \(.[0].seat_id) | \(map(select(.event == "spawn_started").model) | unique | join(", ")) | \(map(select(.event == "rejected")) | length) | \(map(select(.event == "spawn_started")) | length) |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
jq -r 'select(.event == "spawned") | .details | capture("permission_mode=(?<m>[^ ]+)").m' docs/evidence/org-recheck-2026-10-24/manifest.jsonl | sort | uniq -c
```

```text
   6 autonomous
```

- `ralph doctor` in the clone (no `ralph.toml`) before the first start:
  "All checks passed"; agmsg "info — available at /tmp/ralph-iso/agmsg
  (version 1.5.1; tested against 1.1.13)"; "Merge guarantee: pass —
  github_protection" with the two required checks and the up-to-date rule,
  not cached yet (no ledger); Codex warn (not on `PATH`, not needed);
  Manifest version warn (no manifest found, as in any clone of this
  meta-repo). `ralph status`: the state directory
  `/private/tmp/ralph-iso/ralph/.harness/state/org` (source:
  git-main-worktree), "org watch: not running", "director: not attached",
  "merge guarantee: unknown (no org ledger yet)"
- D7's check: `/tmp/ralph-iso/scratch-automerge.toml` (`setup/scratch-automerge.toml`,
  `[org.director]` with only `auto_merge = true`) loaded, and
  `ralph org director status --config <it>` printed `enabled: true` and
  `auto_merge: true`
- The S1 probe (D8) ran as one two-message claude session on haiku,
  02:15:20Z to 02:15:36Z. Its tokens are in `watcher-usage.txt` and are not
  counted as the run's cost

### The recipe, section by section

Each section of `docs/recipes/isolated-herdr-server.md` as the run met it
(the notes' "手順書の節ごとの記録"):

- Opening (`mkdir -m 700 /tmp/ralph-iso`): as written; the directory was
  `drwx------`
- Why isolate, and what has to move: nothing to run. No herdr server of the
  user ran (plan Assumptions), and none was touched
- A repository with its own ledger: a dedicated clone, as written, at
  `/tmp/ralph-iso/ralph` (plan D1; the recipe's examples say
  `/tmp/ralph-iso/work`). The run used the defaults, so the clone got no
  `ralph.toml` until D7's. The clone had no `.git/info` (the user's git
  template makes none), so for D7's `.git/info/exclude` the director ran
  `mkdir -p .git/info` first; that step is the plan's (D7), outside the
  recipe
- The environment: the wrapper as written, plus the two lines named in
  Setup. The `herdr --help` check printed `Config` and `Logs` paths under
  `/tmp/ralph-iso/xdg-config/herdr/` and created nothing, as written
- git and gh config (F1): the user's git settings are only in
  `~/.config/git/config` (no `~/.gitconfig`), and gh keeps its config in
  `~/.config/gh`, so both lines stayed. `SSH_AUTH_SOCK` was not passed, as
  the recipe says; the probe's `git push --dry-run` over ssh still
  authenticated
- PATH in claude's Bash tool (F2): `claude-env` exports the wrapper's
  `PATH`, as written
- Shell aliases: `SHELL=/bin/bash`. `ralph doctor`'s alias check warned about
  the `codex` and `claude` aliases in `~/.zshrc`, a file of a shell the panes
  do not run, and its scanned files included `~/.profile`. In a pane,
  `type claude` named claude's binary, `alias` listed nothing, and codex was
  not found. As written
- Optional: a private agmsg and a notification shim: the shim as written. The
  agmsg copy as written, but passed with `RALPH_ORG_AGMSG_HOME` instead of
  `[org].agmsg_home` (D2). The seats' Claude Code still used the user's agmsg
  skill (RC1), which the recipe does not cover
- Start the server: as written (`nohup`, pid 21921, at 2026-10-24T09:42:04Z).
  `pgrep -fl 'herdr server'` also listed the director's own Bash command,
  whose text held the string `herdr server` (a compound command); that is not
  a gap of the recipe
- Checks before the first start, the server: `lsof` listed only files under
  `/private/tmp/ralph-iso` (`herdr-server.out`,
  `xdg-config/herdr/herdr-server.log`, the working directory), `/dev/null`,
  unnamed unix sockets and pipes, the binary, and dyld; `ps eww` listed only
  the wrapper's variable names (Setup); `h.sock` was `srw-------`. As written
- Folder trust, and a probe in a throwaway pane: the plan's D6 (the human
  answers in their own terminal) did not work, and the recipe's way did, by a
  route the recipe does not name. I0a: the human ran
  `cd /tmp/ralph-iso/ralph && command claude --model haiku` in their own
  terminal; no trust screen appeared, and `~/.claude.json` then had no entry
  for `/private/tmp/ralph-iso/ralph` (nor a trusted parent). An interactive
  `claude --model haiku` in the director's throwaway pane then showed the
  trust dialog, cursor on "No, exit", with the warning "This folder
  pre-approves 62 tool permissions in .claude/settings.json" (the director
  sent no key). I0b: the human attached to the isolated server with
  `/tmp/ralph-iso/run herdr`, a command the recipe does not give (RC7), chose
  "Yes, I trust this folder" in the probe pane, and ran `/exit`;
  `~/.claude.json` then recorded the clone with `trust=true`. The probe: the
  plan's form `claude -p --model haiku --allowedTools Bash '<request>'` failed
  ("Input must be provided either through stdin or as a prompt argument when
  using --print"), because `--allowedTools` took the request as one more tool
  name (RC8); with the request right after `-p` it ran. Its results:
  `command -v` named `/opt/homebrew/bin/git`, `/opt/homebrew/bin/gh`,
  `/tmp/ralph-iso/bin/ralph`, claude's binary, and the shim; `git config
  user.name` printed the user's name; gh showed `yoshpy-dev` active;
  `git -C /tmp/ralph-iso/ralph push --dry-run origin HEAD:refs/heads/org-recheck-probe`
  printed `* [new branch] HEAD -> org-recheck-probe` (no ref is made). The
  shim coming first in the `claude -p` session's Bash tool fits a `-p`
  session reading `CLAUDE_ENV_FILE` too (the plan's open question), but the
  pane's own shell did not print its `command -v osascript`, so that
  `path_helper` had put `/usr/bin` first in this pane (F2) was not
  confirmed, and the answer stays an inference.
  The workspace was closed (`workspace list` empty). The recipe's step 3
  types the request into an interactive claude with `send-text`; the probe
  ran `claude -p` through `pane run` instead (D8)
- ralph doctor and ralph status: as written (Setup)
- Run: the watch was launched from the wrapper
  (`ralph org watch --all --detach`, 02:17:13Z), and so were the starts, the
  sends, the regate, and both merges (the director's transcript). The
  director read panes with `ralph org read` and also with `herdr pane read`,
  which this section leaves to the human (a director runs only `ralph` and
  `gh`), and sent no key to any pane (Trial deviations)
- Clean up: as written, in the recipe's order (Cleanup, below). The recipe
  has no step for the feature worktrees of a dedicated clone; the director
  removed them with `./scripts/ralph-worktree.sh cleanup` (plan AC8)
- Running again, and What this recipe does not cover: not exercised

## Timeline (UTC)

| Time | What | Source |
|------|------|--------|
| 2026-10-24T09:40:43Z | S1 started | notes |
| 2026-10-24T09:42:04Z | the isolated herdr server started | notes |
| 2026-10-24T09:42:54Z to 2026-10-25T02:10:18Z | I0a: folder trust in the human's own terminal: no trust screen, no record | `interventions.log` |
| 02:11:25Z to 02:14:35Z | I0b: trust answered in a throwaway pane of the isolated server: `trust=true` | `interventions.log` |
| 02:15:20Z to 02:15:36Z | the D8 probe | `watcher-usage.txt`, notes |
| 02:16:15Z | S1 ended | notes |
| 02:17:13Z | `ralph org watch --all --detach` started (pid 18252) | `watch-alive.json`, notes |
| 02:17:22Z | director d1 attached (driver claude, wake pull) | `director.jsonl` |
| 02:17:31Z | the merge guarantee `github_protection` cached | `merge-guarantee.json` |
| 02:23:03Z to 02:23:28Z | I1: the human approved the split plan `splits/recheck.md` (`sha256:4f955a6f4269`) | `interventions.log` |
| 02:23:54Z to 02:23:58Z | `ralph org start --plan ... --feature backfill-na --driver claude --model opus`: scope reserved, leader pane-1 spawned, branch `fix/backfill-na` | manifest |
| 02:24:05Z to 02:24:12Z | the same for hooks-matcher: leader pane-2, branch `fix/hooks-matcher` | manifest |
| 02:25:45Z, 02:25:56Z | the leaders raised the plan-approval QUESTIONs e1 (hooks-matcher) and e2 (backfill-na) | inbox |
| 02:26:49Z to 02:31:12Z | I2: the human approved both feature plans in one call; `plan_approved` at 02:31:17Z for both | `interventions.log`, manifest |
| 02:31:32Z, 02:31:34Z | the approval DECISIONs sent to the leaders | manifest |
| 02:31:47Z, 02:31:49Z | four spawns refused for want of `--scope` (RC2) | manifest |
| 02:31:53Z to 02:32:35Z | the implementers (opus) and the reviewers (sonnet) spawned | manifest |
| 02:33:12Z, 02:33:51Z | TASK t-1 to the implementers | manifest |
| 02:34:34Z | the first watchdog ALERT item (e3); backfill-na's implementer failed to send its RESULT with agmsg (RC1) | inbox, notes |
| 02:35:05Z | backfill-na's implementer typed its RESULT t-1 into its leader's pane with `ralph org send` | manifest, notes |
| 02:35:17Z, 02:43:10Z | TASK t-2 to the reviewers of backfill-na and hooks-matcher | manifest |
| 02:42:51Z | hooks-matcher's implementer sent its RESULT with the private copy's `send.sh` | notes, `agmsg-summary.tsv` |
| 02:53:48Z | backfill-na's reviewer typed its RESULT t-2 into its leader's pane with `ralph org send` | manifest, notes |
| 02:54:23Z | backfill-na's gate started | gate log name |
| 03:00:01Z | hooks-matcher's reviewer failed to send its RESULT with agmsg and sent nothing more (RC1) | notes |
| 03:02:01Z | hooks-matcher's gate started; the two gates ran side by side until 03:09:33Z | gate log names, manifest |
| 03:09:33Z | backfill-na `gate_passed` | manifest |
| 03:10:13Z to 03:10:18Z | backfill-na: RESULT e11 naming PR #255, disbanded | inbox, manifest |
| 03:16:07Z | hooks-matcher `gate_passed` | manifest |
| 03:16:40Z to 03:16:47Z | hooks-matcher: RESULT e12 naming PR #256, disbanded | inbox, manifest |
| 03:16:57Z | `ralph org merge-order`: #255 and #256 both ready; D7's conditions met | notes |
| 03:20:32Z to 14:02:25Z | I3: the first merge and the second by D7, while the human was away | `interventions.log` |
| 14:02:46Z to 14:03:09Z | `ralph org merge --branch fix/backfill-na` merged #255 | notes, manifest |
| 14:03:16Z | `ralph org merge-order`: #256 `rebase` | notes |
| 14:03Z to 14:03:42Z | `auto_merge = true` in the clone's `ralph.toml`; `ralph org director status` read it | notes |
| 14:03:50Z to 14:18:25Z | `ralph org regate --branch fix/hooks-matcher`: merge commit 77c631b2, `gate_passed` via=regate at 14:18:21Z, pushed | notes, manifest |
| 14:18:39Z | `ralph org merge-order`: #256 ready | notes |
| 14:18:51Z to 14:22:31Z | `ralph org merge --branch fix/hooks-matcher --wait-checks 60m` merged #256 after 3 min 10 s of waiting for checks | director's transcript, manifest |
| 14:23:07Z to 14:24:59Z | cleanup (AC8) and the final snapshot | notes |

## Two orgs (AC2)

Both orgs came from the one split plan `splits/recheck.md`, approved by the
human at 02:23:28Z (I1). Each `scope_reserved` event names the split and its
digest, and each org opened one PR:

| Org | Split | Digest | Branch | Scope reserved | Disbanded | PR |
|-----|-------|--------|--------|----------------|-----------|----|
| backfill-na | recheck | 4f955a6f4269 | fix/backfill-na | 2026-10-25T02:23:54Z | 2026-10-25T03:10:18Z | #255 |
| hooks-matcher | recheck | 4f955a6f4269 | fix/hooks-matcher | 2026-10-25T02:24:05Z | 2026-10-25T03:16:47Z | #256 |
| overlap | | | | 2026-10-25T02:24:05Z | 2026-10-25T03:10:18Z | 2773 s |

```sh
jq -sr 'group_by(.org_id) | map({org: .[0].org_id, sr: (map(select(.event == "scope_reserved"))[0]), end: (map(select(.event == "disbanded")) | last.ts), pr: (map(select(.event == "pr_merged"))[0].details | capture("pr=(?<p>[0-9]+)").p)}) | (.[] | "| \(.org) | \(.sr.details | capture("split=(?<s>[^ ]+)").s) | \(.sr.details | capture("digest=(?<d>[^ ]+)").d) | \(.sr.details | capture("branch=(?<b>[^ ]+)").b) | \(.sr.ts) | \(.end) | #\(.pr) |"), "| overlap | | | | \(map(.sr.ts) | max) | \(map(.end) | min) | \((map(.end) | min | fromdate) - (map(.sr.ts) | max | fromdate)) s |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

The orgs ran side by side for 46 min 13 s. The approval lines of the split
plan and of the two feature plans, and the digest of each copy computed now
(the same as the approved one, so the copies are the plans as approved):

```sh
for f in docs/evidence/org-recheck-2026-10-24/splits/recheck.md docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-backfill-reviewer-status.md docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-hooks-matcher.md; do echo "$f: $(grep '^- Approved:' "$f"), digest now $(./scripts/plan-visual.sh digest "$f")"; done
```

```text
docs/evidence/org-recheck-2026-10-24/splits/recheck.md: - Approved: 2026-10-25 sha256:4f955a6f4269, digest now 4f955a6f4269
docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-backfill-reviewer-status.md: - Approved: 2026-10-25 sha256:1d3a60a7cd8b, digest now 1d3a60a7cd8b
docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-hooks-matcher.md: - Approved: 2026-10-25 sha256:fc16043d3a2d, digest now fc16043d3a2d
```

- PR #255 (`fix/backfill-na`), "fix: mark incomplete cross-review cycles n/a
  in insights backfill", 5 files, +322 −2: `internal/insights/backfill.go`,
  `internal/insights/backfill_test.go`, the archived feature plan, the
  reviewer's report `docs/reports/review-2026-10-25-backfill-na-reviewer.md`,
  and the org report `docs/reports/org-manifest-backfill-na-2026-10-25.md`
- PR #256 (`fix/hooks-matcher`), "fix: tighten Codex hooks checks in ralph
  doctor", 5 files, +326 −1: `internal/cli/doctor.go`,
  `internal/cli/doctor_hooks_test.go`, the archived feature plan,
  `docs/reports/review-2026-10-25-hooks-matcher-reviewer.md`, and
  `docs/reports/org-manifest-hooks-matcher-2026-10-25.md`
- Neither PR touches `docs/tech-debt/README.md`: the split plan's preamble
  leaves the RESOLVED mark of the two rows to this plan's PR
- Both PR titles, and #255's body, are in English (RC5)

## Fix effects (AC3)

| Item | Expected | Observed | Where read | Verdict |
|------|----------|----------|------------|---------|
| F3 | no send fails with `agent_not_found`; the seats' herdr agents keep the names `<org>_<seat>` | the six agents were named `<org>_<seat>` at the watch's last cycle of each org; no manifest line names `agent_not_found`; every director send exited 0 | `watch-status-<org>.json` (the second table under Snapshot), `manifest.jsonl` (command B), notes. The director read the watch's `agent_get` records instead of `herdr agent list` | effective |
| F4 | a failed spawn closes its pane (`spawn_failed`, or `spawn_close_failed` when the close fails) | no `spawn_failed` and no `spawn_close_failed`. The four `rejected` spawns (RC2) were refused before a pane was made: none has a pane or a `spawn_step` | `manifest.jsonl` (command B) | did not occur, so not checked |
| F5 | both feature plans hold `## 全機能の座席に適用する決まり` with the split's preamble; `spawn_started` has the implementer on opus and the reviewer on sonnet | both plans hold the section with all four rules of the preamble, in a code block (hooks-matcher's at the top, backfill-na's after Open questions); implementer opus and reviewer sonnet in both orgs | `plans/*.md` (command C), `manifest.jsonl` (the models table under Setup) | effective |
| F6 | each watchdog escalation becomes a TYPE ALERT item of the inbox, left to the director | the watch's 8 escalations (8 `WATCHDOG ESCALATION` lines in `watch.log`, each naming its item) are the 8 ALERT items e3 to e10, each with the headers ORG_ID, SEAT, CONDITION, REASON, ALERT_ID, and ALERT_TS and `delivery: deadline`; the director acked each in 6 to 15 s and resolved it in 30 to 128 s | `inbox.jsonl` (Numbers), `watch.log` (command F) | effective |
| F13 | an item the director handles within the ack deadline gets no notification; an item that meets a notification condition gets a `notified` event with its reason and result, and its shim line | no item met a condition (ack deadline, resolve deadline, no watch, `inbox notify`). `inbox.jsonl` has no `notified` event, the snapshot has no `escalations.jsonl`, and none of the 12 shim lines is ralph's (RC11) | `inbox.jsonl`, `osascript.log` (Numbers) | the first half effective; a due notification did not occur, so its record was not checked |
| F7 | the director's sends to a leader do not time out | the director sent 10 DECISIONs to the leaders (e1 to e10), and each body has its `sent` event, whose details start with the body (so none carries `submit_unconfirmed=true`); every send exited 0 in 1 to 11 s. A send that fails writes nothing to the manifest (`internal/org/verbs.go`, `EventSent` only after the submit), so the notes' exit codes are the record of the failures, and they show none | `org-msgs/`, `manifest.jsonl` (command D), notes | effective. Whether the leaders waited under the Monitor tool is not in the notes: not checked |
| F11 | no `wip:` commit on either feature branch | the 15 commits that the two PRs brought to main hold no `wip:` commit | git (command E) | effective for what was pushed. The local branches were removed at cleanup with `--force-branch`; whether they held a commit that was not pushed was not recorded |
| F8, F9, F10, G1 | no gate or regate fails for a reason of the repository unrelated to the change | 2 gates and 1 regate passed at their first run. `scripts/check-sync.sh` passed with each branch's reviewer report (`review-`) and org report (`org-manifest-`) committed, `ROOT_ONLY` 0 (F8); the two gates overlapped for 452 s and passed (F9); the longest package was `internal/org`, 349 to 400 s, under 10 min (F10); `tests/test-check-sync.sh` passed its 16 cases in each run (G1) | `manifest.jsonl`, `gate-excerpts.md` (its last section for F8 and G1), the PRs' files | effective for F8 and F9. F10: the wrapper's `GOFLAGS=-timeout=30m` is the value the Go verifier now sets by itself, and no package came near 10 min, so the verifier's own default was not exercised |
| Open row: the watch's ALERT to a leader times out | count it | 0 `failed to ALERT leader` lines in `watch.log`; the watch's 29 ALERTs to the leaders are in the private agmsg | `watch.log`, `agmsg-summary.tsv` | did not occur (load average 5 to 17) |
| Open row: the watcher times out | count it | 0 `watcher error` lines; each of the 13 watcher runs ended with a verdict | `watch.log`, `model-receipts.jsonl` | did not occur |
| Gate and regate duration | record it | 910 s and 846 s for the gates, 862 s for the regate | Gates | - |
| Folder trust in the feature worktrees | no trust dialog in a feature worktree | none in the two leaders' panes read right after the start (02:24Z); the seats started and worked | notes (`herdr pane read`) | effective, after I0b trusted the clone |

Command B (F3, F4):

```sh
jq -sr '"spawn_failed \(map(select(.event == "spawn_failed")) | length)", "spawn_close_failed \(map(select(.event == "spawn_close_failed")) | length)", "rejected \(map(select(.event == "rejected")) | length), with a pane \(map(select(.event == "rejected" and .pane_id != null)) | length)", "lines naming agent_not_found \(map(select(tostring | test("agent_not_found"))) | length)"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

```text
spawn_failed 0
spawn_close_failed 0
rejected 4, with a pane 0
lines naming agent_not_found 0
```

Command C (F5): the preamble's rules found in each feature plan, and where
the section starts:

```sh
awk 'NR == FNR {if (/^## 座席の決まり/) p = 1; else if (/^## /) p = 0; else if (p && /^- /) {want[$0] = 1; w++}; next} FNR == 1 {f = FILENAME; sub(".*/", "", f)} ($0 in want) {n[f]++} END {for (k in n) printf "%s: %d of the %d preamble lines\n", k, n[k], w}' docs/evidence/org-recheck-2026-10-24/splits/recheck.md docs/evidence/org-recheck-2026-10-24/plans/*.md | LC_ALL=C sort
grep -n '^## 全機能の座席に適用する決まり' docs/evidence/org-recheck-2026-10-24/plans/*.md
```

```text
2026-10-25-backfill-reviewer-status.md: 4 of the 4 preamble lines
2026-10-25-hooks-matcher.md: 4 of the 4 preamble lines
docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-backfill-reviewer-status.md:98:## 全機能の座席に適用する決まり
docs/evidence/org-recheck-2026-10-24/plans/2026-10-25-hooks-matcher.md:12:## 全機能の座席に適用する決まり
```

Command D (F7): each DECISION body the director wrote, and the `sent` event
whose details are that body (or its start, when ralph cut it):

```sh
for i in 1 2 3 4 5 6 7 8 9 10; do jq -r --rawfile t "docs/evidence/org-recheck-2026-10-24/org-msgs/e$i-decision.txt" --arg e "e$i" 'select(.event == "sent" and .seat_id == "leader") | select(.details | rtrimstr("...(truncated)") as $d | ($t | startswith($d))) | "\($e) \(.org_id) \(.ts)"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl; done
```

```text
e1 hooks-matcher 2026-10-25T02:31:32Z
e2 backfill-na 2026-10-25T02:31:34Z
e3 backfill-na 2026-10-25T02:36:15Z
e4 hooks-matcher 2026-10-25T02:36:18Z
e5 hooks-matcher 2026-10-25T02:43:24Z
e6 backfill-na 2026-10-25T02:53:02Z
e7 backfill-na 2026-10-25T02:54:36Z
e8 hooks-matcher 2026-10-25T02:58:58Z
e9 hooks-matcher 2026-10-25T03:00:27Z
e10 hooks-matcher 2026-10-25T03:02:09Z
```

Command E (F11), on main's history after both merges (98d38391 is #256's
merge commit): the number of commits the PRs brought, then the number of
`wip:` commits among them:

```sh
git log --format=%s ffb3e271..98d38391 | wc -l | tr -d ' '
git log --format=%s ffb3e271..98d38391 | grep -c '^wip:'
```

```text
15
0
```

Command F (F6): the inbox item each `WATCHDOG ESCALATION` line of
`watch.log` names, and where the line says it went:

```sh
grep -oE '^WATCHDOG ESCALATION: org=[^ ]+ .* item=e[0-9]+ -- left in [^ ]+ for the director' docs/evidence/org-recheck-2026-10-24/watch.log | sed -E 's/^WATCHDOG ESCALATION: org=([^ ]+) .* item=(e[0-9]+) -- left in .* for the director$/\2 \1 left for the director/'
```

```text
e3 backfill-na left for the director
e4 hooks-matcher left for the director
e5 hooks-matcher left for the director
e6 backfill-na left for the director
e7 backfill-na left for the director
e8 hooks-matcher left for the director
e9 hooks-matcher left for the director
e10 hooks-matcher left for the director
```

The load average on the machine's 12 CPUs, from the notes (the director
wrote it at these points, not every 30 min as the plan said):

| Time | Load average (1, 5, 15 min) | What ran (notes) |
|------|-----------------------------|------------------|
| 2026-10-24T09:40:43Z | 5.15 5.23 6.11 | S1 start |
| 02:16:15Z | 6.30 5.54 4.69 | S1 end |
| 02:22:48Z | 5.17 4.72 4.51 | before I1 |
| 02:34:46Z | 17.30 10.12 7.09 | before any gate; most likely the two implementers' `go test` (the notes' guess) |
| 02:36:41Z | 13.53 10.68 7.66 | the implementers, and backfill-na's reviewer from 02:35:17Z |
| 02:43:42Z | 6.65 7.60 7.33 | both reviewers had their TASK |
| 02:53:16Z | 6.00 6.68 6.94 | backfill-na's reviewer waiting for `run-test.sh` |
| 02:59:10Z | 6.97 7.20 7.08 | backfill-na's gate; hooks-matcher's reviewer waiting for its tests |
| 03:01:58Z | 7.14 7.26 7.13 | both gates, from 03:02:01Z |
| 14:03:16Z | 11.59 6.55 4.78 | right after the first merge |
| 14:18:39Z | 5.98 6.45 6.10 | right after the regate |

## Merges (AC4)

Both PRs were merged by `ralph org merge`: the first when the human said so,
the second by the director under `auto_merge` after `ralph org regate`.

| PR | Branch | `pr_merged` | Head | Method | Merge commit | `auto_merge` |
|----|--------|-------------|------|--------|--------------|--------------|
| #255 | fix/backfill-na | 2026-10-25T14:03:09Z | 5d72c239 | merge | 536271a2 | false |
| #256 | fix/hooks-matcher | 2026-10-25T14:22:31Z | 77c631b2 | merge | 98d38391 | true |

```sh
jq -r 'select(.event == "pr_merged") | (.details | capture("branch=(?<b>[^ ]+) pr=(?<p>[0-9]+) head=(?<h>[0-9a-f]+) method=(?<m>[^ ]+) merge_commit=(?<c>[0-9a-f]+) auto_merge=(?<a>[^ ]+)")) as $d | "| #\($d.p) | \($d.b) | \(.ts) | \($d.h[0:8]) | \($d.m) | \($d.c[0:8]) | \($d.a) |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

GitHub's `mergedAt` is 14:03:08Z for #255 and 14:22:29Z for #256
(`gh pr view <n> --json mergedAt`).

- D7's wait: the director merged nothing until both orgs had recorded
  `gate_passed` on the same base and disbanded, and `ralph org merge-order`
  listed both as ready. At 03:16:57Z it did: #255 and #256 ready, the gate
  base ffb3e271 equal to origin/main, with the note that merging one makes
  the other `rebase` and that a disbanded org's branch takes
  `ralph org regate --branch`. The ledger up to that time:

```sh
jq -r 'select(.ts <= "2026-10-25T03:16:57Z" and (.event == "gate_passed" or .event == "disbanded")) | "\(.ts) \(.org_id) \(.event)\(if .event == "gate_passed" then " base=" + (.details | capture("base=(?<b>[0-9a-f]{8})").b) else "" end)"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

```text
2026-10-25T03:09:33Z backfill-na gate_passed base=ffb3e271
2026-10-25T03:10:18Z backfill-na disbanded
2026-10-25T03:16:07Z hooks-matcher gate_passed base=ffb3e271
2026-10-25T03:16:47Z hooks-matcher disbanded
```

- I3 (03:20:32Z): the director asked both merge questions in one call; #255's
  checks had all passed and #256's `verify` was pending. The human answered at
  14:02:25Z: "merge #255", and for the second, go ahead as D7 says
- #255: at 14:02:46Z, `ralph org merge --branch fix/backfill-na` in the
  foreground (`auto_merge` false, no `--wait-checks`), exit 0 at 14:03:09Z,
  23 s. Its output:

```sh
cat docs/evidence/org-recheck-2026-10-24/merge-fix-backfill-na.out
```

```text
MERGED: https://github.com/yoshpy-dev/ralph/pull/255 (PR #255 of fix/backfill-na)
METHOD: merge
HEAD: 5d72c2393a9eb4d29d2ed2b94570ea726395f67e
MERGE COMMIT: 536271a2c0a274be93b124905a3f622c1ea73ad0
AUTO MERGE: false
PROTECTION: the ruleset "MAIN PROTECTION" (id 15133768) of main, the default branch of origin's repository yoshpy-dev/ralph, requires branches to be up to date before merging with its 2 required status checks, and you cannot bypass it (current_user_can_bypass: never)
```

- `ralph org merge-order` at 14:03:16Z: #256 `rebase` ("gate base ffb3e271,
  origin/main 536271a2", "run ralph org regate --branch fix/hooks-matcher").
  #256's three checks had passed by then
- `auto_merge`: at about 14:03Z, with the human's I3 answer, the director
  wrote `[org.director]` `auto_merge = true` to the clone's `ralph.toml`
  (none until then), excluded through `.git/info/exclude`. At 14:03:42Z both
  `ralph org director status` and `ralph status` read it as true, and
  `git status` in the clone was empty
- The regate: `ralph org regate --branch fix/hooks-matcher` in the
  background, 14:03:50Z to 14:18:25Z (14 min 35 s), exit 0. It merged
  origin/main into b31e623a with a merge commit, ran the gate scripts on it
  (`gate_passed` via=regate at 14:18:21Z, Gates), and pushed without force.
  Its output:

```sh
cat docs/evidence/org-recheck-2026-10-24/regate-fix-hooks-matcher.out
```

```text
GATE: pass
MERGED: 77c631b2a635faaa7f79600302da08b925770c58 (origin/main into b31e623ac24761ca652b08e36c26c0e140f2ce5e)
HEAD: 77c631b2a635faaa7f79600302da08b925770c58
BASE: 536271a2c0a274be93b124905a3f622c1ea73ad0
LOG: /private/tmp/ralph-iso/ralph/.harness/state/org/gates/hooks-matcher-20261025T140359Z.log
PUSHED: 77c631b2a635faaa7f79600302da08b925770c58 to origin's fix/hooks-matcher
```

- `ralph org merge-order` at 14:18:39Z: #256 ready (gate head 77c631b2, base
  536271a2 equal to origin/main); its `verify` check pending
- #256: at 14:18:51Z, without asking (`auto_merge = true`, the
  `/director` skill's step for it), the director ran
  `ralph org merge --branch fix/hooks-matcher --wait-checks 60m` in the
  background; exit 0, `pr_merged` at 14:22:31Z. Its output:

```sh
cat docs/evidence/org-recheck-2026-10-24/merge-fix-hooks-matcher.out
```

```text
org: merge: PR #256 has checks pending, so ralph org merge waits for them for up to 1h0m0s (--wait-checks 1h0m0s), with regate.lock released, reading gh pr checks 256 -R github.com/yoshpy-dev/ralph every 30s
org: merge: stopped waiting for the checks of PR #256 after 3m10s: the checks passed; running every check again
MERGED: https://github.com/yoshpy-dev/ralph/pull/256 (PR #256 of fix/hooks-matcher)
METHOD: merge
HEAD: 77c631b2a635faaa7f79600302da08b925770c58
MERGE COMMIT: 98d38391c908eeceb865ad88ff921e4e45b14cae
AUTO MERGE: true
PROTECTION: the ruleset "MAIN PROTECTION" (id 15133768) of main, the default branch of origin's repository yoshpy-dev/ralph, requires branches to be up to date before merging with its 2 required status checks, and you cannot bypass it (current_user_can_bypass: never)
```

- Neither `ralph org merge` nor `ralph org regate` refused anything. No PR
  other than these two was merged into main during the run. Main after both
  (`git log --oneline ffb3e271..98d38391`):

```text
98d38391 Merge pull request #256 from yoshpy-dev/fix/hooks-matcher
77c631b2 Merge origin/main 536271a2c0a2 into fix/hooks-matcher
536271a2 Merge pull request #255 from yoshpy-dev/fix/backfill-na
b31e623a docs: archive the hooks-matcher plan
af6be8bb docs: add the hooks-matcher org report
ee134928 docs: add the hooks-matcher review report
5d72c239 docs: archive the backfill-reviewer-status plan
7cb69d46 docs: add the backfill-na org report
5e8748a7 docs: add the backfill-na review report
1c31dcf9 fix: tighten Codex hooks checks in ralph doctor
5babe702 fix: mark incomplete cross-review cycles n/a in backfill
99d31882 docs: approve the backfill-reviewer-status plan
96903683 docs: approve the hooks-matcher plan
d2890590 docs: add the backfill-reviewer-status plan
632ea30c docs: add the hooks-matcher plan
```

## Numbers (AC5)

### Interventions

Five interventions, 98237 s of latency in all (27 h 17 min). "Asked" is the
time of the director's AskUserQuestion call in its Claude Code transcript and
"answered" the time of that call's result; the log's latency is the
difference of the transcript's millisecond times, rounded. Two of them were
absence rather than thinking time: I0a waited overnight (16 h 27 min) and I3
waited while the human was away (10 h 42 min). The other three took 479 s.
The planning question of 2026-10-23 came before the run and is not counted,
as in the trial (the log's header). Apart from the two plan approvals (e1 and
e2, I2), no inbox item needed the human, and the inbox holds no BLOCKED.

| # | What | Asked | Answered | Latency (s, log) | Answered − asked (s) | Answer and the ledger events it caused |
|---|------|-------|----------|------------------|----------------------|----------------------------------------|
| I0a | folder trust of the clone in the human's own terminal (plan D6) | 2026-10-24T09:42:54Z | 2026-10-25T02:10:18Z | 59245 | 59244 | "the trust screen did not appear"; no trust record for the clone in ~/.claude.json afterwards (the question waited overnight) |
| I0b | folder trust in the throwaway pane of the isolated server (the recipe's way): attach with /tmp/ralph-iso/run herdr, choose Yes, /exit | 2026-10-25T02:11:25Z | 2026-10-25T02:14:35Z | 190 | 190 | done -> ~/.claude.json records the clone with trust=true |
| I1 | split plan approval (splits/recheck.md) | 2026-10-25T02:23:03Z | 2026-10-25T02:23:28Z | 26 | 25 | approved -> Approved 2026-10-25 sha256:4f955a6f4269 written; start backfill-na 02:23:53Z, hooks-matcher 02:24:04Z |
| I2 | feature plan approval hooks-matcher (inbox e1) and backfill-na (inbox e2), one call with two questions | 2026-10-25T02:26:49Z | 2026-10-25T02:31:12Z | 263 | 263 | both approved -> plan_approved 02:31:17Z for both (ralph org approve e1, e2) |
| I3 | first merge (#255 with ralph org merge) and the second by D7 (auto_merge = true, regate and merge #256), one call with two questions | 2026-10-25T03:20:32Z | 2026-10-25T14:02:25Z | 38513 | 38513 | "merge #255" and "go ahead" -> pr_merged #255 14:03:09Z; regate gate_passed 14:18:21Z; pr_merged #256 14:22:31Z (auto_merge=true) |
| total | 5 interventions | | | 98237 | 98235 | |

```sh
jq -nrR '[inputs | select(test("^I[0-9]")) | split(" | ")] | (.[] | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3]) | \(.[4]) | \((.[3] | fromdate) - (.[2] | fromdate)) | \(.[5]) |"), "| total | \(length) interventions | | | \(map(.[4] | tonumber) | add) | \(map((.[3] | fromdate) - (.[2] | fromdate)) | add) | |"' docs/evidence/org-recheck-2026-10-24/interventions.log
```

The trial had seven, 1302 s in all. This run had no ruleset question (the
ruleset was fixed in the trial) and no stop-and-restart question (a non-goal
here), asked both feature plans in one call and both merges in one call, and
added the two folder-trust interventions.

### Approval wait per feature plan

From the inbox item's `escalated` time to the manifest's `plan_approved`
event that names it:

| Item | Org | Plan | Digest | Escalated | `plan_approved` | Wait (s) |
|------|-----|------|--------|-----------|-----------------|----------|
| e1 | hooks-matcher | docs/plans/active/2026-10-25-hooks-matcher.md | fc16043d3a2d | 2026-10-25T02:25:45Z | 2026-10-25T02:31:17Z | 332 |
| e2 | backfill-na | docs/plans/active/2026-10-25-backfill-reviewer-status.md | 1d3a60a7cd8b | 2026-10-25T02:25:56Z | 2026-10-25T02:31:17Z | 321 |

```sh
jq -nr --slurpfile inbox docs/evidence/org-recheck-2026-10-24/inbox.jsonl 'inputs | select(.event == "plan_approved") | (.details | capture("plan=(?<plan>[^ ]+) digest=(?<dg>[^ ]+) .* inbox=(?<id>e[0-9]+)")) as $d | ($inbox | map(select(.id == $d.id and .event == "escalated"))[0]) as $q | "| \($d.id) | \(.org_id) | \($d.plan) | \($d.dg) | \($q.ts) | \(.ts) | \((.ts | fromdate) - ($q.ts | fromdate)) |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

With `interventions.log`, each wait splits into the director's part, the
human's, and the recording. e1: asked 64 s after it arrived, answered 263 s
later, recorded 5 s after the answer. e2: asked 53 s after it arrived, in the
same call, so the same answer and recording. The trial's waits were 109 s and
423 s.

Both leaders' QUESTIONs carried a `VISUAL:` header naming
`.harness/state/plan-visual/<slug>.html` and said the page passed `lint` and
was shot. That is the leaders' own claim; the director opened both pages for
the human at I2.

### The director's handling of the inbox

| Item | Org | Type | Condition | Delivery | Escalated | Acked | To ack (s) | Resolved | To resolve (s) | `notified` events |
|------|-----|------|-----------|----------|-----------|-------|------------|----------|----------------|-------------------|
| e1 | hooks-matcher | QUESTION | - | deadline | 2026-10-25T02:25:45Z | 2026-10-25T02:26:02Z | 17 | 2026-10-25T02:32:42Z | 417 | 0 |
| e2 | backfill-na | QUESTION | - | deadline | 2026-10-25T02:25:56Z | 2026-10-25T02:26:16Z | 20 | 2026-10-25T02:32:43Z | 407 | 0 |
| e3 | backfill-na | ALERT | scope_change | deadline | 2026-10-25T02:34:34Z | 2026-10-25T02:34:46Z | 12 | 2026-10-25T02:36:42Z | 128 | 0 |
| e4 | hooks-matcher | ALERT | scope_change | deadline | 2026-10-25T02:34:57Z | 2026-10-25T02:35:12Z | 15 | 2026-10-25T02:36:42Z | 105 | 0 |
| e5 | hooks-matcher | ALERT | scope_change | deadline | 2026-10-25T02:43:06Z | 2026-10-25T02:43:14Z | 8 | 2026-10-25T02:43:42Z | 36 | 0 |
| e6 | backfill-na | ALERT | stall | deadline | 2026-10-25T02:52:18Z | 2026-10-25T02:52:27Z | 9 | 2026-10-25T02:53:17Z | 59 | 0 |
| e7 | backfill-na | ALERT | scope_change | deadline | 2026-10-25T02:53:56Z | 2026-10-25T02:54:03Z | 7 | 2026-10-25T02:54:50Z | 54 | 0 |
| e8 | hooks-matcher | ALERT | stall | deadline | 2026-10-25T02:58:35Z | 2026-10-25T02:58:43Z | 8 | 2026-10-25T02:59:10Z | 35 | 0 |
| e9 | hooks-matcher | ALERT | scope_change | deadline | 2026-10-25T03:00:09Z | 2026-10-25T03:00:18Z | 9 | 2026-10-25T03:00:39Z | 30 | 0 |
| e10 | hooks-matcher | ALERT | scope_change | deadline | 2026-10-25T03:01:43Z | 2026-10-25T03:01:49Z | 6 | 2026-10-25T03:02:19Z | 36 | 0 |
| e11 | backfill-na | RESULT | - | deadline | 2026-10-25T03:10:13Z | 2026-10-25T03:10:20Z | 7 | 2026-10-25T03:10:37Z | 24 | 0 |
| e12 | hooks-matcher | RESULT | - | deadline | 2026-10-25T03:16:40Z | 2026-10-25T03:16:48Z | 8 | 2026-10-25T03:16:57Z | 17 | 0 |

```sh
jq -sr 'group_by(.id) | sort_by(.[0].id | ltrimstr("e") | tonumber)[] | (map(select(.event == "escalated"))[0]) as $e | (map(select(.event == "acked"))[0].ts) as $a | (map(select(.event == "resolved"))[0].ts) as $r | "| \($e.id) | \($e.org_id) | \($e.type) | \(if $e.type == "ALERT" then ($e.body | capture("CONDITION: (?<c>[^\n]+)").c) else "-" end) | \($e.delivery) | \($e.ts) | \($a) | \(($a | fromdate) - ($e.ts | fromdate)) | \($r) | \(($r | fromdate) - ($e.ts | fromdate)) | \(map(select(.event == "notified")) | length) |"' docs/evidence/org-recheck-2026-10-24/inbox.jsonl
```

Every item was recorded with `delivery: deadline` (the director enabled and a
watch accepting work) and went escalated, acked, resolved; none reached the
ack deadline of 10 min or the resolve deadline of 60 min, so no item reached
the human path with any reason. The longest were 20 s to ack (e2) and 417 s
to resolve (e1, which waited for the human's approval). The director answered
each ALERT item with a DECISION saying no action was needed (`org-msgs/`,
RC3) and resolved the RESULT items with the PR's URL as the note, without an
answer. The events in the inbox:

```sh
jq -r '.event' docs/evidence/org-recheck-2026-10-24/inbox.jsonl | awk '{n[$0]++} END {for (e in n) printf "%s %d\n", e, n[e]}' | LC_ALL=C sort
jq -sr '"notified events: \(map(select(.event == "notified")) | length)"' docs/evidence/org-recheck-2026-10-24/inbox.jsonl
jq -r 'select(.event == "escalated" and .type == "ALERT") | .body | split("\n\n")[0] | split("\n") | map(split(": ")[0]) | join(" ")' docs/evidence/org-recheck-2026-10-24/inbox.jsonl | sort | uniq -c
```

```text
acked 12
escalated 12
resolved 12
notified events: 0
   8 TYPE ORG_ID SEAT CONDITION REASON ALERT_ID ALERT_TS
```

### Watchdog escalations and notifications

The watch escalated 8 times, every time as an ALERT item of the inbox left to
the director (F6), and every time about a leader:

| Org | Seat | Reason | Condition | Count | First | Last |
|-----|------|--------|-----------|-------|-------|------|
| backfill-na | leader | leader_is_anomaly_subject | scope_change | 2 | 2026-10-25T02:34:34Z | 2026-10-25T02:53:56Z |
| backfill-na | leader | leader_is_anomaly_subject | stall | 1 | 2026-10-25T02:52:18Z | 2026-10-25T02:52:18Z |
| hooks-matcher | leader | leader_is_anomaly_subject | scope_change | 4 | 2026-10-25T02:34:57Z | 2026-10-25T03:01:43Z |
| hooks-matcher | leader | leader_is_anomaly_subject | stall | 1 | 2026-10-25T02:58:35Z | 2026-10-25T02:58:35Z |
| total | | | | 8 | 2026-10-25T02:34:34Z | 2026-10-25T03:01:43Z |

```sh
jq -sr 'map(select(.event == "escalated" and .type == "ALERT") | . + (.body | capture("SEAT: (?<seat>[^\n]+)\nCONDITION: (?<cond>[^\n]+)\nREASON: (?<reason>[^\n]+)"))) | (group_by([.org_id, .seat, .reason, .cond])[] | "| \(.[0].org_id) | \(.[0].seat) | \(.[0].reason) | \(.[0].cond) | \(length) | \(.[0].ts) | \(.[-1].ts) |"), "| total | | | | \(length) | \(map(.ts) | min) | \(map(.ts) | max) |"' docs/evidence/org-recheck-2026-10-24/inbox.jsonl
```

None of the 12 shim lines came from ralph. Each one falls 3 or 4 s before a
receipt of the on-demand watcher, and all have the title "Claude Code" and
the text `` ```json `` (RC11):

| Watcher receipt | Org | Seat watched | Verdict | Shim line | Seconds before the receipt | Shim title |
|-----------------|-----|--------------|---------|-----------|----------------------------|------------|
| 2026-10-25T02:34:54Z | backfill-na | implementer | normal | - | - | - |
| 2026-10-25T02:35:26Z | hooks-matcher | reviewer | role_violation | 2026-10-25T02:35:22Z | 4 | Claude Code |
| 2026-10-25T02:43:33Z | hooks-matcher | implementer | normal | 2026-10-25T02:43:30Z | 3 | Claude Code |
| 2026-10-25T02:49:05Z | backfill-na | implementer | fake_progress | 2026-10-25T02:49:02Z | 3 | Claude Code |
| 2026-10-25T02:49:53Z | hooks-matcher | implementer | fake_progress | 2026-10-25T02:49:50Z | 3 | Claude Code |
| 2026-10-25T02:51:14Z | backfill-na | reviewer | normal | 2026-10-25T02:51:11Z | 3 | Claude Code |
| 2026-10-25T02:52:43Z | backfill-na | leader | normal | 2026-10-25T02:52:40Z | 3 | Claude Code |
| 2026-10-25T02:54:17Z | backfill-na | implementer | normal | 2026-10-25T02:54:14Z | 3 | Claude Code |
| 2026-10-25T02:55:28Z | backfill-na | reviewer | role_violation | 2026-10-25T02:55:25Z | 3 | Claude Code |
| 2026-10-25T02:59:02Z | hooks-matcher | leader | normal | 2026-10-25T02:58:59Z | 3 | Claude Code |
| 2026-10-25T03:00:28Z | hooks-matcher | leader | normal | 2026-10-25T03:00:25Z | 3 | Claude Code |
| 2026-10-25T03:02:10Z | hooks-matcher | reviewer | role_violation | 2026-10-25T03:02:06Z | 4 | Claude Code |
| 2026-10-25T03:04:38Z | backfill-na | implementer | normal | 2026-10-25T03:04:35Z | 3 | Claude Code |

```sh
jq -nr --rawfile shim docs/evidence/org-recheck-2026-10-24/osascript.log '($shim | split("\n") | map(select(length > 0) | capture("^(?<ts>[^ ]+) -e display notification \"(?<text>.*)\" with title \"(?<title>[^\"]*)\"$"))) as $n | inputs | select(.role == "watchdog") | .ts as $t | ($n | map(select((.ts | fromdate) <= ($t | fromdate) and ($t | fromdate) - (.ts | fromdate) <= 10))) as $m | "| \(.ts) | \(.org_id) | \(.seat_id) | \(.reason | ltrimstr("verdict=")) | \(if ($m | length) == 0 then "- | - | -" else "\($m[0].ts) | \(($t | fromdate) - ($m[0].ts | fromdate)) | \($m[0].title)" end) |"' docs/evidence/org-recheck-2026-10-24/model-receipts.jsonl
jq -nr --rawfile shim docs/evidence/org-recheck-2026-10-24/osascript.log '$shim | split("\n") | map(select(length > 0) | capture("display notification \"(?<text>.*)\" with title \"(?<title>[^\"]*)\"$")) | group_by([.title, .text])[] | "\(length) shim lines: title \(.[0].title), text \(.[0].text)"'
```

```text
12 shim lines: title Claude Code, text ```json
```

The watch's own lines, and the watcher's verdicts. The 13 watcher runs judged
8 cases normal, 3 a role violation (all about reviewers), and 2 fake
progress (both about implementers); 11 more triggers were skipped because a
watcher run was in flight. The watch neither failed an ALERT to a leader nor
had a watcher error (the trial had 10 and 6):

```sh
grep -oE '^(WATCHDOG ESCALATION|watchdog: failed to ALERT leader|watchdog: watcher already in flight|watchdog: watcher error|watching every org)' docs/evidence/org-recheck-2026-10-24/watch.log | awk '{n[$0]++} END {for (l in n) printf "%d %s\n", n[l], l}' | LC_ALL=C sort -k2
jq -r 'select(.role == "watchdog") | .reason' docs/evidence/org-recheck-2026-10-24/model-receipts.jsonl | awk '{n[$0]++} END {for (r in n) printf "%d watchdog receipts %s\n", n[r], r}' | LC_ALL=C sort -k4
```

```text
8 WATCHDOG ESCALATION
11 watchdog: watcher already in flight
1 watching every org
2 watchdog receipts verdict=fake_progress
8 watchdog receipts verdict=normal
3 watchdog receipts verdict=role_violation
```

### Cost

Tokens per seat session and model (`seat-usage.tsv`), the on-demand watcher
and the S1 probe (`watcher-usage.txt`), and the estimate in USD:

| Org | Seat | Session | Model | Messages | Input | Cache write (1h) | Cache read | Output | Estimate (USD) |
|-----|------|---------|-------|----------|-------|------------------|------------|--------|----------------|
| backfill-na | implementer | implementer-1 | claude-opus-5-5 | 20 | 40 | 86610 | 2052139 | 6132 | 1.23 |
| backfill-na | leader | leader-1 | claude-opus-5-5 | 48 | 96 | 128814 | 6561751 | 17180 | 2.69 |
| backfill-na | reviewer | reviewer-1 | claude-sonnet-5-5 | 19 | 38 | 114462 | 1868510 | 5595 | 0.70 |
| hooks-matcher | implementer | implementer-1 | claude-opus-5-5 | 23 | 46 | 91932 | 2368194 | 7395 | 1.36 |
| hooks-matcher | leader | leader-1 | claude-opus-5-5 | 50 | 100 | 126962 | 6845784 | 16224 | 2.71 |
| hooks-matcher | reviewer | reviewer-1 | claude-sonnet-5-5 | 14 | 28 | 111860 | 1329011 | 4988 | 0.63 |
| both orgs | watcher | - | claude-haiku-4-5-20251001 | 13 | 117 | 470182 | 238017 | 25918 | 1.09 |
| S1 setup | probe | - | claude-haiku-4-5-20251001 | 2 | 18 | 55074 | 53304 | 1083 | 0.12 |
| backfill-na | all seats | | | | | | | | 4.61 |
| hooks-matcher | all seats | | | | | | | | 4.70 |
| both orgs | all seats | | | | | | | | 9.31 |
| run | seats and watcher | | | | | | | | 10.40 |

```sh
awk -F'\t' '
  BEGIN { p["claude-opus-5-5"] = "4 8 0.20 20"; p["claude-sonnet-5-5"] = "2 4 0.10 10"; p["claude-haiku-4-5-20251001"] = "1 2 0.10 5" }
  function usd(m, i, w, r, o,   q) { split(p[m], q, " "); return (i * q[1] + w * q[2] + r * q[3] + o * q[4]) / 1e6 }
  FILENAME ~ /seat-usage/ && FNR > 1 {
    c = usd($4, $6, $7, $8, $9); org[$1] += c; seats += c
    printf "| %s | %s | %s | %s | %d | %d | %d | %d | %d | %.2f |\n", $1, $2, $3, $4, $5, $6, $7, $8, $9, c
  }
  FILENAME ~ /watcher-usage/ && /^(watcher|probe) / {
    for (k = 3; k <= NF; k++) { split($k, kv, "="); v[kv[1]] = kv[2] }
    c = usd($2, v["input"], v["cache_write_1h"], v["cache_read"], v["output"]); if ($1 == "watcher") watcher = c
    printf "| %s | %s | - | %s | %d | %d | %d | %d | %d | %.2f |\n", ($1 == "watcher" ? "both orgs" : "S1 setup"), $1, $2, v["msgs"], v["input"], v["cache_write_1h"], v["cache_read"], v["output"], c
  }
  END {
    printf "| backfill-na | all seats | | | | | | | | %.2f |\n", org["backfill-na"]
    printf "| hooks-matcher | all seats | | | | | | | | %.2f |\n", org["hooks-matcher"]
    printf "| both orgs | all seats | | | | | | | | %.2f |\n", seats
    printf "| run | seats and watcher | | | | | | | | %.2f |\n", seats + watcher
  }' docs/evidence/org-recheck-2026-10-24/seat-usage.tsv FS=' +' docs/evidence/org-recheck-2026-10-24/watcher-usage.txt
```

- Prices: USD per million tokens (base input / 5m cache write / 1h cache
  write / cache hit / output), read again on 2026-10-25 from
  <https://platform.claude.com/docs/en/about-claude/pricing>, the same as on
  2026-10-20, and written in `watcher-usage.txt`: Claude Opus 5.5 4 / 5 / 8 /
  0.20 / 20, Claude Sonnet 5.5 2 / 2.50 / 4 / 0.10 / 10, Claude Haiku 4.5 1 /
  1.25 / 2 / 0.10 / 5
- Every cache write of the seats and the watcher was a 1h write (each
  message's `usage.cache_creation.ephemeral_1h_input_tokens`; the seats'
  are opus 434,318 and sonnet 226,322, with 0 written for 5m), so the
  estimate uses the 1h price and never the 5m one
- The run's cost is the seats and the watcher, $10.40 (the trial's $12.81).
  The S1 probe ($0.12) is not part of it. The director's own cost is not
  included: it ran in this plan's Claude Code session and cannot be separated
  from it
- The seats' cost ended at 03:16:47Z, when the last seat stopped; the
  director's regate and merges after that ran in no seat. The seats ran 52
  min 53 s from the first start; R3's limit, five hours from the first
  start, was far off. The two orgs cost about the same: one seat run each,
  one gate each, and the reviewer on sonnet in both (the trial's
  doctor-escape had an opus reviewer, three gates, and a second leader)

## Gates

| Log | Org | Verb | Started (log name) | Recorded | Duration (s) | Event | Reason | Head | Base |
|-----|-----|------|--------------------|----------|--------------|-------|--------|------|------|
| backfill-na-20261025T025423Z.log | backfill-na | gate | 02:54:23Z | 2026-10-25T03:09:33Z | 910 | gate_passed | - | 5d72c239 | ffb3e271 |
| hooks-matcher-20261025T030201Z.log | hooks-matcher | gate | 03:02:01Z | 2026-10-25T03:16:07Z | 846 | gate_passed | - | b31e623a | ffb3e271 |
| hooks-matcher-20261025T140359Z.log | hooks-matcher | regate | 14:03:59Z | 2026-10-25T14:18:21Z | 862 | gate_passed | - | 77c631b2 | 536271a2 |

```sh
jq -r 'select(.event == "gate_passed" or .event == "gate_failed") | (.details | capture("gates/(?<l>[^\"]+)").l) as $log | ($log | capture("T(?<t>[0-9]{6})Z").t) as $st | "| \($log) | \(.org_id) | \(if (.details | test("via=regate")) then "regate" else "gate" end) | \($st[0:2]):\($st[2:4]):\($st[4:6])Z | \(.ts) | \((.ts | fromdate) - ($log | capture("-(?<s>[0-9]{8}T[0-9]{6}Z)").s | strptime("%Y%m%dT%H%M%SZ") | mktime)) | \(.event) | \((.details | capture("reason=(?<r>[^ ]+)").r) // "-") | \(.details | capture("head=(?<h>[0-9a-f]{8})").h) | \(.details | capture("base=(?<b>[0-9a-f]{8})").b) |"' docs/evidence/org-recheck-2026-10-24/manifest.jsonl
```

The duration runs from the time in the log's name (when the gate scripts
started) to the manifest event. The static scripts took 25 s, 18 s, and 21 s
and the tests the rest (`gate-excerpts.md`, the two `Timestamp` lines of
each log). The two gates overlapped from 03:02:01Z to 03:09:33Z, with the
load average at about 7. The trial's passing gates took 1375 s and 1664 s,
and its passing regate 792 s.

## Findings

Ids RC1 to RC12 are this run's; F1 to F14 are the trial's.

- RC1 (major) The seats did not use the private agmsg. ralph's own agmsg
  calls used the copy that `RALPH_ORG_AGMSG_HOME` named: the leaders' joins,
  each seat's join and HELLO at its spawn, and the watch's ALERTs to the
  leaders. The seats' Claude Code sessions instead ran agmsg as the user's
  skill (`~/.claude/skills/agmsg/SKILL.md`, whose commands run
  `~/.agents/skills/agmsg/scripts/`): 5 of the 6 seat session records ran
  those scripts, whose store has no team of this run, so their sends failed
  with "Team not found" and "has no registered agents". Of the four seats
  that owed their leader a RESULT, backfill-na's implementer (its agmsg
  send failed at 02:34:34Z) and reviewer typed theirs into the leader's pane
  with `ralph org send --to leader` (the `sent` events at 02:35:05Z and
  02:53:48Z); hooks-matcher's implementer found `RALPH_ORG_AGMSG_HOME` with
  `env | grep` (02:33:03Z) and sent with the copy's `send.sh` (02:42:51Z);
  hooks-matcher's reviewer failed (03:00:01Z) and sent nothing, and its
  leader read the committed review report instead (e12's body says so). So
  one seat message reached the private agmsg:

  | Team | From | To | TYPE line | Messages | First | Last | Unread |
  |------|------|----|-----------|----------|-------|------|--------|
  | ralph-hooks-matcher | implementer | leader | TYPE: HELLO | 1 | 2026-10-25T02:32:11Z | 2026-10-25T02:32:11Z | 0 |
  | ralph-backfill-na | implementer | leader | TYPE: HELLO | 1 | 2026-10-25T02:32:13Z | 2026-10-25T02:32:13Z | 0 |
  | ralph-hooks-matcher | reviewer | leader | TYPE: HELLO | 1 | 2026-10-25T02:32:33Z | 2026-10-25T02:32:33Z | 0 |
  | ralph-backfill-na | reviewer | leader | TYPE: HELLO | 1 | 2026-10-25T02:32:34Z | 2026-10-25T02:32:34Z | 0 |
  | ralph-backfill-na | watchdog | leader | TYPE: ALERT | 12 | 2026-10-25T02:34:22Z | 2026-10-25T03:04:15Z | 0 |
  | ralph-hooks-matcher | watchdog | leader | TYPE: ALERT | 17 | 2026-10-25T02:34:46Z | 2026-10-25T03:02:10Z | 0 |
  | ralph-hooks-matcher | implementer | leader | TYPE: RESULT | 1 | 2026-10-25T02:42:53Z | 2026-10-25T02:42:53Z | 0 |

  ```sh
  awk -F'\t' '{printf "| %s | %s | %s | %s | %d | %s | %s | %d |\n", $1, $2, $3, $4, $5, $6, $7, $8}' docs/evidence/org-recheck-2026-10-24/agmsg-summary.tsv
  ```

  The director made `agmsg-summary.tsv` at S7 from the private copy's
  `messages.db`, opened with `sqlite3 -readonly`, with this query (the
  expression it used for the first line of the body is not recorded here):

  ```text
  select team, from_agent, to_agent, <the first line of the body>, count(*), min(created_at), max(created_at), sum(read_at is null) from messages group by 1,2,3,4 order by 6
  ```

  The last column counts the messages whose `read_at` is NULL: 0 in every
  row, so the receiver read each of the 34. The database went with
  `/tmp/ralph-iso`, so the snapshot can recompute the table from the TSV
  only, not the TSV from the database. The HELLOs are ralph's (spawn sends them), and the
  TASKs and DECISIONs to seats do not go through agmsg (`ralph org send`
  types them into the pane). Cause: the implementer and reviewer role prompts
  tell the seat to report to the leader with a RESULT or BLOCKED message
  (`internal/org/prompts/implementer.md:102`,
  `internal/org/prompts/reviewer.md:119`) but give no command or agmsg path;
  only the leader's prompt gets the scripts directory of the agmsg ralph uses
  (`{{AGMSG_SCRIPTS}}`, `internal/org/prompts.go:62-71`, used at
  `internal/org/prompts/leader.md:422`, `:436`, and `:447`). The trial used
  the user's own agmsg, so it did not show there. The user's agmsg store was
  not written: `messages.db` and `messages.db-wal` kept their time
  (2026-10-23T06:22:39Z) from S1 to S7 and still had it after the run;
  `messages.db-shm` changed at 03:02:53Z (a reader changes it, and other
  sessions of the user use that agmsg); no team or role-session mark of this
  run is under `~/.agents/skills/agmsg/teams` or `run` (checked on 2026-10-25
  after the run). Impact: with a private copy, a seat's typed report does
  not reach its leader's agmsg, and the leaders got their results by detours.
  The same would happen with any `[org].agmsg_home` other than the user's
  default (inferred from the prompts; not verified). Tech-debt row: yes
  (added by this PR)
- RC2 The leader template's spawn step leaves out `--scope`. Step 5 of the
  leader's role prompt (`internal/org/prompts/leader.md:148-154`) says to pass
  `--cwd .` and `--model` to each spawn, and names neither `--scope` nor
  `--reserve`. Under the default `autonomous` permission envelope, spawn
  refuses a seat without one ("org: autonomous permission mode requires
  --scope or --reserve (--reserve on the leader seat only; or
  --allow-unscoped to explicitly bypass)"). Both leaders' first implementer
  and reviewer spawns were refused (02:31:47Z and 02:31:49Z, the four
  `rejected` events), and each leader spawned them again with a scope: the
  implementers 6 and 8 s after the refusal (`spawn_started` at 02:31:53Z and
  02:31:57Z), the reviewers after them (02:32:12Z and 02:32:13Z). The scopes,
  from the `spawned` details: the reservation for the implementers and for
  backfill-na's reviewer, `review hooks-matcher; writes docs/reports/ only`
  for hooks-matcher's reviewer. This is the default path of every feature
  org. The trial's manifest has no `rejected` event. Impact: one wasted round
  per seat, and the four receipts of RC10. Tech-debt row: yes (added by this
  PR)
- RC3 In a feature org the three seats share one worktree, so the watch's
  `scope_change` (the worktree's `git status --porcelain` differs from the
  last cycle) fires for all three seats on every edit inside the reservation,
  and the leader's goes straight to the inbox as `leader_is_anomaly_subject`.
  A leader that waits more than `stall_minutes` (15) for its reviewer's tests
  raises `stall` the same way. 8 of the 12 inbox items were these: 6
  `scope_change` (the implementers' edits, e3 and e4; an implementer's commit
  that left the worktree clean, e5; the reviewers' reports and the org report,
  e7, e9, and e10) and 2 `stall` (the leaders waiting for their reviewers,
  e6 and e8). The director answered each with "no action needed"
  (`org-msgs/e3-decision.txt` to `e10-decision.txt`). Without a director
  each would reach the human at the ack deadline (inferred from the
  delivery rules; not verified). Tech-debt row: yes (added by this PR)
- RC4 The split plan's preamble says 「予約(`- Reserve:`)の外のファイルは
  編集しない」, the rule of the `/org` skill's example
  (`.claude/skills/org/SKILL.md:529`, 「予約の外のファイルは編集しない」).
  Read literally, it conflicts with the files the feature-org steps of the
  leader template have the seats write outside the reservation: the feature plan, the reviewer's report, and the
  org report. No seat stopped over it, but the director's answers to e7, e9,
  and e10 had to cite the template's steps. Also, hooks-matcher's leader
  rewrote its plan's `Status` and `Approved` lines with `sed -i ''` (notes);
  the repository's rule is to edit tracked files with the Edit and Write
  tools (`.claude/rules/ralph/ralph-workflow.md`). Tech-debt row: yes (one
  row for both, added by this PR)
- RC5 Both PR titles are in English, and so is #255's body (#256's body is in
  Japanese). The `/pr` skill says the title and body must be in Japanese
  (`.claude/skills/pr/SKILL.md:31`). The leader template says to follow the
  project's rules and to match the title and body to the `/pr` skill's
  template (`internal/org/prompts/leader.md:273-275`); the language rule is
  in the skill's step, not in `template.md`, so a leader that follows only
  the template misses it (inferred; not verified). Tech-debt row: yes (added
  by this PR)
- RC6 The `/director` skill's check after a send, that the DECISION shows in
  the leader's pane as sent input (`.claude/skills/director/SKILL.md:728-731`
  and `:771-772`), could not be done for the two approval DECISIONs.
  `ralph org read` reads `herdr pane read --source recent`
  (`internal/org/driver/herdr.go:474-476`), which held about 22 lines of the
  screen, and the input had scrolled off once the leader went to work; the
  input line was empty. The director took the leaders' approval commits
  (99d31882, 96903683) as the delivery. For e3 to e10 the sent input
  (`❯ TYPE: DECISION`) and the leader's reply were still on the screen.
  Tech-debt row: yes (added by this PR)
- RC7 The recipe gives no command for the human to attach to the isolated
  server, and the plan's way around it failed. Trust answered in the human's
  own terminal (I0a) showed no trust screen and recorded nothing in
  `~/.claude.json`, though the folder was not trusted (the director's
  throwaway pane then showed the dialog); why is not known. The recipe's
  throwaway pane worked once the human attached with
  `/tmp/ralph-iso/run herdr` (I0b). Tech-debt row: no: the recipe is fixed in
  this PR (the attach command). The cause of I0a stays unverified
- RC8 The plan's D8 probe put the request after `--allowedTools`, whose
  values are variadic, so claude took the request as a tool name and failed;
  with the request right after `-p` the probe ran. Tech-debt row: no: an error
  of this plan's command, not of the repository
- RC9 The director's wait ended with exit 4 (the watch) rather than 5 (the
  registration), although the detach came first:

  ```sh
  jq -r '"\(.ts) director.jsonl \(.event) \(.id)"' docs/evidence/org-recheck-2026-10-24/director.jsonl
  jq -r '"\(.stopped_at) watch-alive.json \(.state) pid \(.pid)"' docs/evidence/org-recheck-2026-10-24/watch-alive.json
  jq -r '"\(.ended_at) director-wait.json end=\(.end), started_at \(.started_at): \(.detail)"' docs/evidence/org-recheck-2026-10-24/director-wait.json
  ```

  ```text
  2026-10-25T02:17:22Z director.jsonl attached d1
  2026-10-25T14:23:22Z director.jsonl detached d1
  2026-10-25T14:23:34Z watch-alive.json stopped pid 18252
  2026-10-25T14:23:36Z director-wait.json end=watch, started_at 2026-10-25T14:03:51Z: no watch accepts work: watch not_running, pid 18252
  ```

  The wait ran 14 s past the detach without noticing it. The wait reads
  `director.jsonl` every 30 s and probes the watch every 5 s
  (`DirectorWaitMarkInterval` and `DirectorWaitWatchProbeInterval`,
  `internal/org/director_wait.go:84` and `:87`); counted from its start at
  14:03:51Z, its registration reads fell at about 14:23:21Z and 14:23:51Z and
  a watch probe at 14:23:36Z, so it read the watch's stop first. That is an
  inference from the code and the times, not checked by a run; the notes'
  guess (a read every 2 s) does not match the code. The `/director` skill
  says the wait can take about 30 s to return after the registration changes
  (`.claude/skills/director/SKILL.md:540`), but not that a watch stop in that
  time turns the 5 into a 4. A director that detaches and then stops the
  watch within 30 s gets exit 4, which the skill handles as the watch going
  away, not as the end of its registration. Tech-debt row: yes (added by
  this PR)
- RC10 The four refused spawns of RC2 each left a model receipt with
  `honored=false` and the refusal's text as its reason, so `ralph insights`
  counts four models as not honored where no seat ran:

  ```sh
  jq -r 'select(.honored == "false") | "\(.ts) \(.org_id) \(.seat_id) commanded \(.commanded_model) honored=\(.honored)"' docs/evidence/org-recheck-2026-10-24/model-receipts.jsonl
  ```

  ```text
  2026-10-25T02:31:47Z hooks-matcher implementer commanded opus honored=false
  2026-10-25T02:31:47Z hooks-matcher reviewer commanded sonnet honored=false
  2026-10-25T02:31:49Z backfill-na implementer commanded opus honored=false
  2026-10-25T02:31:49Z backfill-na reviewer commanded sonnet honored=false
  ```

  Tech-debt row: yes (added by this PR)
- RC11 The on-demand watcher's `claude -p` sessions load the user's Claude
  Code hooks. Its 13 sessions (haiku, the `role=watchdog` receipts) ran in the
  clone's root (`watcher-usage.txt`), and per the notes the user's Stop hook
  that posts a desktop notification ran `osascript` (here the shim, first on
  the watch's `PATH`) about 3 s before 12 of the 13 receipts, with the first
  line of the watcher's answer, `` ```json ``, as its text (Numbers, the
  pairing table). The first run (02:34:54Z) has no shim line; why was not
  checked. Outside isolation, each watcher run would post one desktop
  notification to the user. The director's note during S3 put these lines on
  the seats' sessions; that was wrong, and the director corrected it at S7.
  The trial's 20 "Claude Code" lines most likely had the same cause, not the
  seats' own Claude Code that its evidence names (the user's Notification
  hook, per that director's notes): 18 of them fall within 10 s before a
  watcher receipt (not checked further):

  ```sh
  jq -nr --rawfile shim docs/evidence/org-trial-2026-10-20/osascript.log --slurpfile rc docs/evidence/org-trial-2026-10-20/model-receipts.jsonl '($rc | map(select(.role == "watchdog") | .ts | fromdate)) as $w | $shim | split("\n") | map(select(length > 0) | capture("^(?<ts>[^ ]+) .*with title \"(?<title>[^\"]*)\"\\]$") | select(.title == "Claude Code") | .ts | fromdate) | "trial Claude Code lines \(length), followed by a watcher receipt within 10 s: \(map(. as $x | select([$w[] | select(. >= $x and . - $x <= 10)] | length > 0)) | length)"'
  ```

  ```text
  trial Claude Code lines 20, followed by a watcher receipt within 10 s: 18
  ```

  Tech-debt row: yes (added by this PR)
- RC12 Both leaders' status line showed `Opus 5.5` with effort `low` (read
  in their panes right after the start, 02:24Z), most likely from the user's
  own Claude Code setting (not verified). Tech-debt row: no: the user's
  setting, not a defect of the repository

The on-demand watcher, recorded here without a finding of its own: its 13
runs judged 8 cases `normal`, 3 `role_violation` (all about reviewers:
hooks-matcher's twice, backfill-na's once), and 2 `fake_progress` (both
implementers, 02:49:05Z and 02:49:53Z), and the watch skipped 11 more
triggers with `watchdog: watcher already in flight` (Numbers, Watchdog
escalations and notifications). Whether each verdict was right was not
checked against the panes.

## Comparison with the 2026-10-20 trial

| Item | 2026-10-20 trial | This recheck | Read from |
|------|------------------|--------------|-----------|
| binary | f0668004 | ffb3e271 (all fixes #244 to #254) | Setup |
| orgs side by side | 2 h 36 min 10 s | 46 min 13 s | Two orgs |
| first start to the last disband | 2 h 51 min 55 s | 52 min 53 s | manifest |
| interventions | 7, 1302 s | 5, 98237 s (479 s without I0a and I3, which were absence) | Interventions |
| approval waits | 109 s, 423 s | 332 s, 321 s | Approval wait |
| inbox items | 6 (2 QUESTION, 2 BLOCKED, 2 RESULT) | 12 (2 QUESTION, 8 ALERT, 2 RESULT) | inbox |
| longest to ack, to resolve | 213 s, 997 s | 20 s, 417 s | inbox |
| `notified` events | 0 | 0 | inbox |
| watchdog escalations | 30, each sent to the human directly (29 shim lines; one has none, F13) | 8, all inbox ALERT items left to the director | `escalations.jsonl` / inbox |
| shim lines from ralph | 29 | 0 | `osascript.log` |
| shim lines "Claude Code" | 20 | 12 (RC11) | `osascript.log` |
| failed ALERTs to a leader, watcher errors | 10, 6 | 0, 0 | `watch.log` |
| `agent_not_found` sends | yes (F3) | none | notes, manifest |
| refused spawns (`rejected`) | 0 | 4 (RC2) | manifest |
| failed spawns (`spawn_failed`) | 4 | 0 | manifest |
| gates and regates | 6, 3 failed (F8, F9, F10) | 3, none failed | Gates |
| passing gate duration | 1375 s, 1664 s; regate 792 s | 910 s, 846 s; regate 862 s | Gates |
| `wip:` commits on a feature branch | 1 (F11) | 0 | git |
| models as the split plan said | no (F5) | yes | manifest |
| cost, seats and watcher | $12.81 | $10.40 | Cost |

## Trial deviations

- D2: the private agmsg copy was named by `RALPH_ORG_AGMSG_HOME` in the
  wrapper instead of `[org].agmsg_home` in the clone's `ralph.toml`
- D6: trust answered in the human's own terminal did not work (I0a); the
  human answered in the isolated server's throwaway pane instead (I0b), as the
  recipe says, which took one more intervention
- D8: the probe's request moved to right after `-p` (RC8)
- D7: `mkdir -p .git/info` in the clone before `.git/info/exclude` was
  written
- F3 was read from the watch's `agent_get` records instead of
  `herdr agent list`, and F11 from main's history after the merges instead of
  `git log origin/main..<branch>` before cleanup
- The plan's AC3 F7 names a `send_failed` event of the manifest; ralph has no
  such event (a failed send writes nothing), so the notes' exit codes were
  read instead
- The director wrote the load average at events, not every 30 min, and did
  not write the token totals every 30 min (plan S3); the cost comes from the
  session records, read after the run
- During S3 the director read panes with `herdr pane read` as well as
  `ralph org read` (both through the wrapper, no keys sent). The recipe's
  Run section leaves `herdr` commands to the human and limits a director to
  `ralph` and `gh`

## Cleanup (AC8)

In this order:

1. Both orgs had disbanded (03:10:18Z and 03:16:47Z, manifest). The first
   copy of the ledger (D12) at 14:23:07Z; `ralph org inbox` printed "No open
   or acked inbox items"
2. `ralph org director detach --id d1` at 14:23:22Z ("detached d1")
3. `ralph org watch --all --stop` at 14:23:34Z (it stopped pid 18252); the
   director's wait ended at 14:23:36Z with exit 4 (RC9)
4. `ralph status`, from the clone: "org watch: not running (stopped at
   2026-10-25T14:23:34Z)" and "director: not attached"
   (`ralph-status-final.txt`). From `~`, `pgrep` found no ralph watch and
   `lsof` found nothing left but the herdr server
5. The feature worktrees: `./scripts/ralph-worktree.sh cleanup --id org-<org> --force-branch`
   for both orgs
6. `/tmp/ralph-iso/run herdr server stop`: the server's pid, the socket, and
   its `lsof` lines were gone
7. The final copy (D12) at 14:24:59Z, with `setup/` added, then the
   reconciliation: the inbox's 12 items all went escalated, acked, resolved
   with no `notified` event; the shim's 12 lines are all RC11's, none
   ralph's; `ralph-status-final.txt` has the stop line; the user's agmsg
   `messages.db` and `messages.db-wal` were unchanged since S1, and
   `messages.db-shm` since S3 (RC1)
8. `rm -rf /tmp/ralph-iso`; `pgrep -fl /tmp/ralph-iso` then exited 1 (the
   director's handoff to this record; the notes give no time)

The user's herdr server was not touched, and no file under `~/.config/herdr`
changed after 2026-10-24 (checked 2026-10-25 after the run). The shim's log
holds every `osascript` call that went through `PATH` (12). Left outside
`/tmp/ralph-iso`, in the user's stores:

- one trust entry in `~/.claude.json` (`/private/tmp/ralph-iso/ralph`)
- three Claude Code session-record directories under `~/.claude/projects/`
  (`-private-tmp-ralph-iso-ralph` and the two org worktrees)
- no agmsg team or role-session mark (RC1)
- on GitHub: the two merged PRs. `git ls-remote origin` lists neither
  `fix/backfill-na` nor `fix/hooks-matcher` (checked 2026-10-25 after the
  run), and the probe's dry run made no `org-recheck-probe` branch

## Not verified

- Stopping and restarting an org or the watch (a non-goal of the plan; the
  spec's open question stays open)
- Why the human's own terminal showed no trust screen (RC7)
- Why the wait ended on the watch (RC9): inferred from the code's intervals
- RC1 with a `[org].agmsg_home` other than the default, and RC3 without a
  director
- That the user's Stop hook is what posted the watcher's lines (RC11) is
  from the director's notes; the hook was not read for this file. Why the
  first watcher run has no line
- Whether the leaders waited under the Monitor tool (F7)
- The record of a notification that is due (F13), and the Go verifier's own
  test timeout (F10)
- Whether a feature branch held a commit that was not pushed before cleanup
  (F11)
- The versions of herdr and Claude Code during the run (from the plan's
  Assumptions)
- The director's own cost
- codex seats, a codex director, and a headless director
  (`ralph org director start`)
