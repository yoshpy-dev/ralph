# Live check: the director's wait mark and the watch's stop (stage 8b-2b)

Plan: `docs/plans/active/2026-10-19-org-director-wait.md` (D8, AC9). Run on
2026-10-19 with the `ralph` binary built from the task branch at fa95a3d8
(S1 to S3). The later commits change documents, test files, the headless
role prompt (which this check does not use), and, in the self-review fixes
c1d31e3f and 33da4185, Go comments, a move of code between files, one
constant in place of two with the same value, and the help sentence about
when exit 5 comes; none of them changes what the wait does or what it prints
while it waits.

## Setup

- macOS 26.5.2, go1.26.0 darwin/arm64, `go build -o /tmp/dw8/ralph ./cmd/ralph`
- An isolated ledger: `--state-dir /tmp/dw8/state` on every command, run from
  `/tmp/dw8`, which is not a git repository and has no `ralph.toml`, so every
  setting is the default (`[org.director].enabled = true`, wake pull)
- No herdr: `HERDR_SOCKET_PATH`, `XDG_CONFIG_HOME`, and `XDG_STATE_HOME` under
  `/tmp/dw8`, and a `herdr` on `PATH` that logs and exits 1. No real
  notification: an `osascript` on `PATH` that logs and exits 0. The shim log
  stayed empty: neither was run
- `USER=tester`, `HERDR_PANE_ID` unset, so the registration records no pane
- The intervals are the defaults: the mark every 30 s, the watch probe every
  5 s, the settle 2 s
- Signals went only to processes the script started: the waits, and the
  detached watch it started (SIGKILL in (c))
- `/tmp/dw8` was left in place after the run for reading and holds nothing
  but this check's ledger

## Results

| Check | What was done | What happened |
|-------|---------------|---------------|
| (a) | `attach --driver claude`, `watch --all --detach`, then `wait --inbox --director d1 --timeout-ms 0` in the background | `ralph org director status` printed `wait: the wait --inbox --director of d1 (pid N) started at T and last wrote its mark at T`; the promoted `director:` line of both status commands added the same fact after "the registration does not tell whether its session is still running"; `--json` had `wait` and `director.wait` with the 8 keys |
| (b) | `watch --all --stop` | the wait exited 4 within 2 s; stderr said `watch.lock is free and watch-alive.json says stopped at T (pid N)` and named `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; the mark: `end` watch, `watch_accepting` false. The next wait printed one `note:` line with the same facts and kept waiting |
| (c) | `watch --all --detach` again, then SIGKILL of that watch | the waiting wait printed `note: a ralph org watch --all accepts work again ...` and rewrote the mark with `watch_accepting` true at once; after the SIGKILL it exited 4 with `watch.lock is free but watch-alive.json still says running (pid N, written at T): the watch ended without writing its stopped mark` |
| (d) | a `--timeout-ms 3000` wait while a watch accepted work, then `--stop`, then a new wait | the first exited 1 with `no open inbox item arrived within 3000 ms` (mark: `end` timeout, `watch_accepting` true); the new wait exited 4 at its start with `accepted work when the last wait of d1 wrote its mark` |
| (e) | a wait for d1, then `attach --driver claude --replace` (d2) | the wait exited 5 at its next mark rewrite, 30 s after it started, naming d2 (mark: `end` not_current); a new wait for d1 was refused with 5 at once, and the mark was byte-for-byte unchanged |
| (f) | SIGTERM to a wait for d2 | mark: `end` signal, `detail` "terminated signal received"; the shell saw the process end by SIGTERM (exit status 143); both status lines said `ended at T (signal)` |
| (g) | SIGKILL to a wait for d2, then 65 s | the mark kept its `seen_at` and no `end`; both status lines added `, and has not written it since (it writes every 30s)` |
| (h) | wait A for d2, then wait B for d2 3 s later, then 32 s | B's start replaced A's mark; at its 30 s rewrite A printed once `wait --inbox --director: another ralph org wait --inbox --director (wait_id …, pid N) took over …director-wait.json, so this wait (wait_id …) does not write it again and keeps waiting` and wrote nothing; after SIGTERM to B and then to A, the mark held B's `end` signal |

None of the texts says that a human stopped the watch or that it crashed;
each says what `watch.lock` and `watch-alive.json` show.

## Script

```bash
#!/bin/bash
set -u
WT=~/MyDev/github.com/yoshpy-dev/ralph/.claude/worktrees/org-director-wait
D=/tmp/dw8
rm -rf "$D"
mkdir -p "$D/state" "$D/bin" "$D/xdg-config" "$D/xdg-state"
(cd "$WT" && go build -o "$D/ralph" ./cmd/ralph) || { echo "build failed"; exit 1; }
printf '#!/bin/sh\necho "osascript shim: $*" >> %s/shim.log\nexit 0\n' "$D" > "$D/bin/osascript"
printf '#!/bin/sh\necho "herdr shim: $*" >> %s/shim.log\nexit 1\n' "$D" > "$D/bin/herdr"
chmod +x "$D/bin/osascript" "$D/bin/herdr"
export PATH="$D/bin:$PATH"
export XDG_CONFIG_HOME="$D/xdg-config" XDG_STATE_HOME="$D/xdg-state" HERDR_SOCKET_PATH="$D/no-herdr.sock" USER=tester
unset HERDR_PANE_ID ORCA_TERMINAL_HANDLE RALPH_ORG_STATE_DIR
cd "$D" || exit 1
R="$D/ralph"
SD="$D/state"
PIDS=()

log() { echo; echo "### $(date -u +%H:%M:%S) $*"; }
mark() { echo "mark: $(cat "$SD/director-wait.json" 2>/dev/null)"; }
dstatus() { "$R" org director status --state-dir "$SD" 2>&1 | grep -E '^(director|wait):'; }
sline() { "$R" status --state-dir "$SD" 2>&1 | grep -E '^(org watch|director):'; }
startwait() { # name id timeout
  "$R" org wait --inbox --director "$2" --timeout-ms "$3" --state-dir "$SD" >"$D/$1.out" 2>"$D/$1.err" &
  local p=$!; PIDS+=("$p"); eval "$1=$p"
}
waitfor() { # pid seconds
  local i=0
  while kill -0 "$1" 2>/dev/null && [ "$i" -lt "$2" ]; do sleep 1; i=$((i + 1)); done
}
endof() { # name pid
  waitfor "$2" 60
  if kill -0 "$2" 2>/dev/null; then echo "$1 still running after 60s"; return; fi
  wait "$2"; echo "$1 exit=$?"
  echo "$1 stdout: $(cat "$D/$1.out")"; echo "$1 stderr:"; cat "$D/$1.err"
}
watchpid() { python3 -I -c 'import json,sys; print(json.load(open(sys.argv[1]))["pid"])' "$SD/watch-alive.json"; }
```

The steps after these helpers are the ones the output below logs under each
`###` heading, in that order (the full script was kept in the session's
scratch directory).

## Output

The bash job lines (`… Terminated: 15`, `… Killed: 9`) name the script by its
scratch path, written `<scratch>` here. Each `wN stdout:` line ended with
one space after the colon, since every wait's stdout was empty; that space is
left out here.

```text
### 02:29:16 setup
ralph dev (unknown unknown)
attached d1 (driver=claude)
detach it with: ralph org director detach --id d1 --state-dir '/tmp/dw8/state'
warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger /tmp/dw8/state (none runs), so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach --state-dir '/tmp/dw8/state'
attach exit=0
started ralph org watch --all (pid 61613); log: /tmp/dw8/state/watch.log
watch --detach exit=0
org watch: running (pid 61613, mark updated 2026-10-19T02:29:16Z)
director: attached d1 (driver=claude, attached at 2026-10-19T02:29:16Z, by tester) -- the registration does not tell whether its session is still running; no wait --inbox --director mark for it; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)

### 02:29:19 (a) the wait: line of a running wait
director: attached d1 (driver=claude, attached at 2026-10-19T02:29:16Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 61952) started at 2026-10-19T02:29:20Z and last wrote its mark at 2026-10-19T02:29:20Z; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
wait: the wait --inbox --director of d1 (pid 61952) started at 2026-10-19T02:29:20Z and last wrote its mark at 2026-10-19T02:29:20Z
org watch: running (pid 61613, mark updated 2026-10-19T02:29:16Z)
director: attached d1 (driver=claude, attached at 2026-10-19T02:29:16Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 61952) started at 2026-10-19T02:29:20Z and last wrote its mark at 2026-10-19T02:29:20Z; inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
json wait: {"wait_id": "61952-1792376960028151000", "pid": 61952, "started_at": "2026-10-19T02:29:20Z", "seen_at": "2026-10-19T02:29:20Z", "watch_accepting": true, "ended_at": "", "end": "", "detail": ""}
status json director.wait: {"wait_id": "61952-1792376960028151000", "pid": 61952, "started_at": "2026-10-19T02:29:20Z", "seen_at": "2026-10-19T02:29:20Z", "watch_accepting": true, "ended_at": "", "end": "", "detail": ""}

### 02:29:23 (b) --stop: the running wait exits 4
stopped ralph org watch --all (pid 61613)
watch --stop exit=0
w1 exit=4
w1 stdout:
w1 stderr:
org: wait --inbox --director: no watch accepts work: a ralph org watch --all accepted work when this wait last probed it, and none does now: watch.lock is free and watch-alive.json says stopped at 2026-10-19T02:29:23Z (pid 61613); the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
mark: {
  "wait_id": "61952-1792376960028151000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 61952,
  "started_at": "2026-10-19T02:29:20Z",
  "seen_at": "2026-10-19T02:29:25Z",
  "watch_accepting": false,
  "ended_at": "2026-10-19T02:29:25Z",
  "end": "watch",
  "detail": "no watch accepts work: watch not_running, pid 61613"
}
director: attached d1 (driver=claude, attached at 2026-10-19T02:29:16Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 61952) ended at 2026-10-19T02:29:25Z (watch); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
wait: the wait --inbox --director of d1 (pid 61952) ended at 2026-10-19T02:29:25Z (watch)

### 02:29:26 (b) the next wait prints note: and keeps waiting
w2 still waiting after 4s
w2 stderr so far:
note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: watch.lock is free and watch-alive.json says stopped at 2026-10-19T02:29:23Z (pid 61613); the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
mark: {
  "wait_id": "62604-1792376966195269000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 62604,
  "started_at": "2026-10-19T02:29:26Z",
  "seen_at": "2026-10-19T02:29:26Z",
  "watch_accepting": false
}

### 02:29:30 (c) a watch starts again: note: and the mark is rewritten
started ralph org watch --all (pid 63029); log: /tmp/dw8/state/watch.log
watch --detach exit=0
w2 stderr so far:
note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: watch.lock is free and watch-alive.json says stopped at 2026-10-19T02:29:23Z (pid 61613); the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
note: a ralph org watch --all accepts work again on this ledger: watch.lock is held and watch-alive.json says running (pid 63029, written at 2026-10-19T02:29:30Z)
mark: {
  "wait_id": "62604-1792376966195269000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 62604,
  "started_at": "2026-10-19T02:29:26Z",
  "seen_at": "2026-10-19T02:29:31Z",
  "watch_accepting": true
}

### 02:29:38 (c) SIGKILL of the watch (a process this script started): the wait exits 4
sent SIGKILL to the watch
w2 exit=4
w2 stdout:
w2 stderr:
note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: watch.lock is free and watch-alive.json says stopped at 2026-10-19T02:29:23Z (pid 61613); the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
note: a ralph org watch --all accepts work again on this ledger: watch.lock is held and watch-alive.json says running (pid 63029, written at 2026-10-19T02:29:30Z)
org: wait --inbox --director: no watch accepts work: a ralph org watch --all accepted work when this wait last probed it, and none does now: watch.lock is free but watch-alive.json still says running (pid 63029, written at 2026-10-19T02:29:30Z): the watch ended without writing its stopped mark; the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
mark: {
  "wait_id": "62604-1792376966195269000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 62604,
  "started_at": "2026-10-19T02:29:26Z",
  "seen_at": "2026-10-19T02:29:41Z",
  "watch_accepting": false,
  "ended_at": "2026-10-19T02:29:41Z",
  "end": "watch",
  "detail": "no watch accepts work: watch not_running, pid 63029"
}

### 02:29:41 (d) a --timeout-ms wait ends by its timeout while the watch accepts work
started ralph org watch --all (pid 65008); log: /tmp/dw8/state/watch.log
watch --detach exit=0
w3 exit=1
w3 stdout:
w3 stderr:
org: wait --inbox --director: no open inbox item arrived within 3000 ms
mark: {
  "wait_id": "65466-1792376984903400000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 65466,
  "started_at": "2026-10-19T02:29:44Z",
  "seen_at": "2026-10-19T02:29:47Z",
  "watch_accepting": true,
  "ended_at": "2026-10-19T02:29:47Z",
  "end": "timeout",
  "detail": "no open inbox item within 3000 ms"
}

### 02:29:48 (d) the watch stops between two waits; the next wait exits 4 at its start
stopped ralph org watch --all (pid 65008)
watch --stop exit=0
w4 exit=4
w4 stdout:
w4 stderr:
org: wait --inbox --director: no watch accepts work: a ralph org watch --all accepted work when the last wait of d1 wrote its mark, and none does now: watch.lock is free and watch-alive.json says stopped at 2026-10-19T02:29:48Z (pid 65008); the command that starts one is `ralph org watch --all --detach --state-dir '/tmp/dw8/state'`; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then reaches the human at its ack deadline only once a watch runs again
mark: {
  "wait_id": "66013-1792376988948937000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 66013,
  "started_at": "2026-10-19T02:29:48Z",
  "seen_at": "2026-10-19T02:29:48Z",
  "watch_accepting": false,
  "ended_at": "2026-10-19T02:29:48Z",
  "end": "watch",
  "detail": "no watch accepts work: watch not_running, pid 65008"
}

### 02:29:50 (e) attach --replace: the wait for d1 exits 5 at its next mark rewrite (30 s)
started ralph org watch --all (pid 66199); log: /tmp/dw8/state/watch.log
watch --detach exit=0
attached d2 (driver=claude), replacing d1 (driver=claude, attached at 2026-10-19T02:29:16Z)
detach it with: ralph org director detach --id d2 --state-dir '/tmp/dw8/state'
attach --replace exit=0
w5 exit=5
w5 stdout:
w5 stderr:
org: wait --inbox --director: d1 is not the current director registration: the current registration is d2 (driver claude, attached 2026-10-19T02:29:56Z)
mark: {
  "wait_id": "66783-1792376994139533000",
  "id": "d1",
  "attached_at": "2026-10-19T02:29:16Z",
  "pid": 66783,
  "started_at": "2026-10-19T02:29:54Z",
  "seen_at": "2026-10-19T02:30:24Z",
  "watch_accepting": true,
  "ended_at": "2026-10-19T02:30:24Z",
  "end": "not_current",
  "detail": "the current registration is d2"
}

### 02:30:25 (e) a wait for the replaced id is refused at once with 5, writing nothing
w5b exit=5
w5b stdout:
w5b stderr:
org: wait --inbox --director: d1 is not the current director registration: the current registration is d2 (driver claude, attached 2026-10-19T02:29:56Z)
the mark is unchanged

### 02:30:26 (f) SIGTERM: end=signal, the process ends by SIGTERM
sent SIGTERM to w6
<scratch>/s5-livefire.sh: line 31: 71933 Terminated: 15          "$R" org wait --inbox --director "$2" --timeout-ms "$3" --state-dir "$SD" > "$D/$1.out" 2> "$D/$1.err"
w6 exit=143
w6 stdout:
w6 stderr:
mark: {
  "wait_id": "71933-1792377026763807000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 71933,
  "started_at": "2026-10-19T02:30:26Z",
  "seen_at": "2026-10-19T02:30:29Z",
  "watch_accepting": true,
  "ended_at": "2026-10-19T02:30:29Z",
  "end": "signal",
  "detail": "terminated signal received"
}
director: attached d2 (driver=claude, attached at 2026-10-19T02:29:56Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 71933) ended at 2026-10-19T02:30:29Z (signal); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
wait: the wait --inbox --director of d2 (pid 71933) ended at 2026-10-19T02:30:29Z (signal)

### 02:30:31 (g) SIGKILL of the wait: no end, seen_at stays
sent SIGKILL to w7
<scratch>/s5-livefire.sh: line 31: 72756 Killed: 9               "$R" org wait --inbox --director "$2" --timeout-ms "$3" --state-dir "$SD" > "$D/$1.out" 2> "$D/$1.err"
w7 exit=137
w7 stdout:
w7 stderr:
mark: {
  "wait_id": "72756-1792377031307302000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 72756,
  "started_at": "2026-10-19T02:30:31Z",
  "seen_at": "2026-10-19T02:30:31Z",
  "watch_accepting": true
}
waiting 65 s
director: attached d2 (driver=claude, attached at 2026-10-19T02:29:56Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 72756) started at 2026-10-19T02:30:31Z and last wrote its mark at 2026-10-19T02:30:31Z, and has not written it since (it writes every 30s); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
wait: the wait --inbox --director of d2 (pid 72756) started at 2026-10-19T02:30:31Z and last wrote its mark at 2026-10-19T02:30:31Z, and has not written it since (it writes every 30s)
org watch: running (pid 66199, mark updated 2026-10-19T02:31:20Z)
director: attached d2 (driver=claude, attached at 2026-10-19T02:29:56Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 72756) started at 2026-10-19T02:30:31Z and last wrote its mark at 2026-10-19T02:30:31Z, and has not written it since (it writes every 30s); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)

### 02:31:40 (h) a second wait takes over the mark; the first stops writing it
mark: {
  "wait_id": "81462-1792377104064582000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 81462,
  "started_at": "2026-10-19T02:31:44Z",
  "seen_at": "2026-10-19T02:31:44Z",
  "watch_accepting": true
}
waiting 32 s for w8's mark rewrite
w8 stderr so far:
wait --inbox --director: another ralph org wait --inbox --director (wait_id 81462-1792377104064582000, pid 81462) took over /tmp/dw8/state/director-wait.json, so this wait (wait_id 81173-1792377100996725000) does not write it again and keeps waiting
mark: {
  "wait_id": "81462-1792377104064582000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 81462,
  "started_at": "2026-10-19T02:31:44Z",
  "seen_at": "2026-10-19T02:32:14Z",
  "watch_accepting": true
}
sent SIGTERM to w9
<scratch>/s5-livefire.sh: line 31: 81462 Terminated: 15          "$R" org wait --inbox --director "$2" --timeout-ms "$3" --state-dir "$SD" > "$D/$1.out" 2> "$D/$1.err"
w9 exit=143
w9 stdout:
w9 stderr:
mark: {
  "wait_id": "81462-1792377104064582000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 81462,
  "started_at": "2026-10-19T02:31:44Z",
  "seen_at": "2026-10-19T02:32:19Z",
  "watch_accepting": true,
  "ended_at": "2026-10-19T02:32:19Z",
  "end": "signal",
  "detail": "terminated signal received"
}
w8 still waiting
sent SIGTERM to w8
<scratch>/s5-livefire.sh: line 31: 81173 Terminated: 15          "$R" org wait --inbox --director "$2" --timeout-ms "$3" --state-dir "$SD" > "$D/$1.out" 2> "$D/$1.err"
w8 exit=143
w8 stdout:
w8 stderr:
wait --inbox --director: another ralph org wait --inbox --director (wait_id 81462-1792377104064582000, pid 81462) took over /tmp/dw8/state/director-wait.json, so this wait (wait_id 81173-1792377100996725000) does not write it again and keeps waiting
mark: {
  "wait_id": "81462-1792377104064582000",
  "id": "d2",
  "attached_at": "2026-10-19T02:29:56Z",
  "pid": 81462,
  "started_at": "2026-10-19T02:31:44Z",
  "seen_at": "2026-10-19T02:32:19Z",
  "watch_accepting": true,
  "ended_at": "2026-10-19T02:32:19Z",
  "end": "signal",
  "detail": "terminated signal received"
}
director: attached d2 (driver=claude, attached at 2026-10-19T02:29:56Z, by tester) -- the registration does not tell whether its session is still running; its wait --inbox --director (pid 81462) ended at 2026-10-19T02:32:19Z (signal); inbox items it does not ack reach the human at the ack deadline; merges are left to the human ([org.director].auto_merge = false)
wait: the wait --inbox --director of d2 (pid 81462) ended at 2026-10-19T02:32:19Z (signal)

### 02:32:21 cleanup
stopped ralph org watch --all (pid 66199)
watch --stop exit=0
shim log:
(none: no osascript or herdr was run)
director-wait.json
director-wait.lock
director.jsonl
director.lock
watch-alive.json
watch.lock
watch.log
done
```

## Not checked here

- A wait started by the Claude Code Monitor tool or by codex's loop (the
  skill's two ways); this check started the waits from a shell
- An open inbox item ending the wait while the watch had just stopped (open
  items win over exit 4); the unit tests cover it
  (`TestWaitInboxForDirector_AnOpenItemWinsOverAWatchThatStopped` in
  `internal/org/director_wait_watch_test.go`), and this check raised no item so
  that no notification path ran
- A wait started with SIGHUP or SIGINT ignored (nohup); the CLI tests cover it
  in a child process (`internal/cli/org_wait_director_test.go`)
