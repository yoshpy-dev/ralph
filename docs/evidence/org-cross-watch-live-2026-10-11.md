# 横断の見張りの実機確認(--detach、kill -9、2 つ目の見張り、--stop、同時の --detach、Linux)

- Date: 2026-10-11(JST 18:50〜19:05)
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md(AC9、AC10、AC12、AC13、AC14)
- 報告: docs/reports/test-2026-10-11-org-cross-watch.md
- 目的: 一時の台帳で、切り離した見張りの起動、`kill -9` のあとの判定と起動し直し、2 つ目の見張りの拒否、`--stop` を、ビルドしたバイナリで確かめる(verify の報告の「足すと確度がいちばん上がる確認」)。あわせて、同じ台帳に `--detach` を 8 つ同時に打ったときと、Linux の setsid を確かめた

## 環境

| 項目 | 値 |
|---|---|
| ralph | このブランチの HEAD dd63a414 をビルドしたもの。macOS は `go build -o <scratch>/ralph ./cmd/ralph`、Linux は `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build`(`ralph version` は `dev`) |
| 台帳 | macOS は `<scratch>/live/ledger` と `<scratch>/live2/ledger`、Linux はコンテナの中の `/tmp/ledger`。どれも初めは存在しない。manifest も受信箱も置いていない(見張りは herdr も agmsg も呼ばず、期限の件もないのでデスクトップ通知は出ない) |
| ralph の起こし方 | すべて `env -i PATH=<空のディレクトリ> HOME=<scratch の下> ralph ...`。cwd は `ralph.toml` のないディレクトリ(既定の設定、`deadman_minutes` 10、間隔 30 秒)。PATH が空なので herdr・agmsg・git・osascript はどれも見つからない |
| Linux | Docker の `ubuntu:24.04`(aarch64)。バイナリは tar で stdin から渡し、一般ユーザー `ubuntu` で動かした |
| 触れていないもの | 利用者の herdr サーバー、`~/.config/herdr`、本物の台帳。osascript の本物の経路は意図して確かめていない |
| 片づけ | スクリプトの終わりに、起動した見張りを `kill -9` する trap(どれもすでに終わっていた)。最後の `pgrep -fl 'org watch --all'` は何も返さなかった |

`<scratch>` は `/private/tmp/claude-502/<セッション>/scratchpad/ocw-test`。`$HOME` は `~` に、`ls` の所有者名は `<user>` に置き換えた。macOS の `ps` の `SESS` 列は常に 0 なので、新しいセッションであることは `PGID` が自分の pid であること、`TTY` が `??` であること、`STAT` の `s`(セッションの長)で読む。`PPID` が 1 なのは、`--detach` のプロセスが終わって子が launchd に引き取られたため。Linux では `SID` 列が読める。

## 結果

### 1. macOS: 起動、kill -9、起動し直し、2 つ目の見張り、--stop

| 確認 | 結果 |
|---|---|
| 台帳がないときの `--stop` | rc 0、「not running」。台帳のディレクトリは作られない |
| `--detach`(1 回目) | rc 0、`started ralph org watch --all (pid 9450)`、0.13 s で返った。子は `PGID == PID`、`TTY ??`、`STAT Ss`、`PPID 1`。印は running、pid 9450、`interval_seconds` 30 |
| `ralph status` / `--json` | `org watch: running (pid 9450, last cycle …)`、`org_watch.state` は `running` |
| 動いている間の `--detach` | rc 0、`already running (pid 9450)`。何も起動しない |
| `kill -9 9450` のあと | 印は running のまま残り、`ralph status` は `org watch: not running (a running mark is left from pid 9450, last cycle …)`、`--json` は `state: not_running`、`mark_state: running`。kill でロックが放されることを、本物のプロセスで確かめた(verify の AC9 の穴) |
| `--detach --interval-seconds 2`(2 回目) | rc 0、新しい pid 9606。子の引数に `--interval-seconds 2` が渡り、印の `interval_seconds` は 2。4.5 s 後に印の `ts` が 09:50:58 から 09:51:02 に進んだ |
| 動いている間の `--all --once` | rc 1、`another ralph org watch --all is running on this state dir (pid 9606, from …/watch-alive.json)`。ロックを 2 秒待ってから断る。断る前に `watching every org (…)` の banner が 1 行出る(下の注 1) |
| `--stop` | rc 0、`stopped ralph org watch --all (pid 9606)`、0.43 s で返った。プロセスは終わり、印は `stopped` で `stopped_at` つき。`watch-stop.json` は残っていない。`ralph status` は `org watch: not running (stopped at …)` |
| 止まったあとの `--stop` | rc 0、「not running」 |
| `watch.log` | 2 回の起動の banner が追記されている(1 回目は interval=30s、2 回目は 2s) |
| 残ったプロセス | なし |

### 2. macOS: 同じ新しい台帳に `--detach` を 8 つ同時に

4 回打った。どの回も、動いている見張りは 1 つだけで、最後の `--stop` で残らず止まった。

| 回 | exit 0 | exit 1 | exit 0 の内訳 |
|---|---|---|---|
| 1 | 6 | 2 | `started` 1、`already running` 5 |
| 2〜4 | 7 | 1 | `started` 1、`already running` 6 |

exit 1 の `--detach` は、どれも `the watch (pid N) exited before it was running; see …/watch.log` で終わった。ロックの取り合いに負けた子は 2 秒待ってから `another ralph org watch --all is running …` を `watch.log` に書いて終わり、親はそれを「子が先に終わった」と読む。計画の AC12 のとおりの動きだが、見張りが動いているのに `--detach` が失敗を返す(下の注 2)。

### 3. Linux(ubuntu:24.04 のコンテナ): setsid と同じ流れ

| 確認 | 結果 |
|---|---|
| `--detach` | rc 0、pid 20。`ps` は `PID 20  PPID 1  PGID 20  SID 20  TT ?  STAT Ssl`。起こしたシェルの SID は 11 なので、子は自分のセッションを持つ(setsid が Linux でも効く) |
| `kill -9 20` のあと | `org watch: not running (a running mark is left from pid 20, …)` |
| `--detach --interval-seconds 2` | rc 0、pid 52(`SID 52`) |
| 動いている間の `--all --once` | rc 1、pid 52 を名指しする |
| `--stop` | rc 0、`stopped ralph org watch --all (pid 52)`。status は `not running (stopped at …)`。`ps` に見張りは残っていない |

### 注

1. 2 つ目の見張り(`--all --once`)は、ロックを取る前に banner `watching every org (interval=30s once=true …)` を stdout に出し、そのあとエラーで終わる。banner は `RunWatchAll` の前に出るため。動きは AC9 のとおりで、表示の順だけの話(報告の「参考」に書いた)
2. 同時の `--detach` のうち 1〜2 本が exit 1 になる。後半の PR で `ralph org start` が見張りを起動するとき、複数の start が同時に走ると、見張りは動いているのに start が失敗と読む形になりうる(報告の Test gaps に書いた)

## スクリプトと出力

### 1. live.sh(macOS)

```bash
#!/bin/bash
# Live check of `ralph org watch --all --detach/--stop` on a throw-away ledger.
# Every ralph command runs under env -i with an empty PATH (no herdr, agmsg,
# git, or osascript can be found) and a scratch HOME, from a scratch cwd with
# no ralph.toml. The ledger has no manifest and no inbox, so the watch calls
# nothing outside itself and shows no notification.
set -u
S="$(cd "$(dirname "$0")" && pwd)"
L="$S/live/ledger"
B="$S/ralph"
rm -rf "$S/live"
mkdir -p "$S/live/cwd" "$S/live/home" "$S/live/emptybin"
cd "$S/live/cwd" || exit 2

r() { # run a ralph command hermetically, print "$ cmd", output, rc
  echo "\$ ralph $*"
  env -i PATH="$S/live/emptybin" HOME="$S/live/home" "$B" "$@" 2>&1
  local rc=$?
  echo "[rc=$rc]"
  return $rc
}
sh_() { echo "\$ $*"; eval "$@" 2>&1; echo "[rc=$?]"; }
pid_of() { sed -n 's/.*(pid \([0-9][0-9]*\)).*/\1/p' | head -1; }

started=()
cleanup() {
  for p in "${started[@]}"; do kill -9 "$p" 2>/dev/null; done
}
trap cleanup EXIT

echo "## 0. the ledger does not exist yet"
sh_ "ls -la '$L'"
r org watch --stop --state-dir "$L"

echo; echo "## 1. --detach starts the watch"
t0=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
out1=$(r org watch --all --detach --state-dir "$L"); echo "$out1"
t1=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
echo "(--detach returned after $(echo "$t1 - $t0" | bc) s)"
pid1=$(echo "$out1" | pid_of); started+=("$pid1")
echo "pid1=$pid1"
sh_ "ps -o pid,ppid,pgid,sess,tty,stat,command -p $pid1"
sh_ "cat '$L/watch-alive.json'"
sh_ "ls -la '$L'"
r status --state-dir "$L"
r status --state-dir "$L" --json
echo "-- a second --detach while it runs"
r org watch --all --detach --state-dir "$L"

echo; echo "## 2. kill -9 the watch"
sh_ "kill -9 $pid1"
sleep 0.3
sh_ "ps -p $pid1 -o pid,command || true"
sh_ "cat '$L/watch-alive.json'"
r status --state-dir "$L"
r status --state-dir "$L" --json

echo; echo "## 3. --detach again starts a new watch (interval 2 s)"
out2=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out2"
pid2=$(echo "$out2" | pid_of); started+=("$pid2")
echo "pid2=$pid2"
sh_ "ps -o pid,ppid,pgid,sess,tty,stat,command -p $pid2"
sh_ "cat '$L/watch-alive.json'"
sleep 4.5
echo "-- 4.5 s later (two more cycles)"
sh_ "cat '$L/watch-alive.json'"
r status --state-dir "$L"

echo; echo "## 4. a second foreground watch while it runs"
r org watch --all --once --state-dir "$L"
sh_ "cat '$L/watch-alive.json'"

echo; echo "## 5. --stop"
t0=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
r org watch --stop --state-dir "$L"
t1=$(perl -MTime::HiRes=time -e 'printf "%.3f", time')
echo "(--stop returned after $(echo "$t1 - $t0" | bc) s)"
sh_ "ps -p $pid2 -o pid,command || true"
sh_ "cat '$L/watch-alive.json'"
sh_ "ls -la '$L'"
r status --state-dir "$L"
r status --state-dir "$L" --json
r org watch --stop --state-dir "$L"

echo; echo "## 6. watch.log"
sh_ "cat '$L/watch.log'"

echo; echo "## 7. nothing left"
sh_ "pgrep -fl 'org watch --all' || echo '(pgrep: no process)'"
```

```text
## 0. the ledger does not exist yet
$ ls -la '<scratch>/live/ledger'
ls: <scratch>/live/ledger: No such file or directory
[rc=1]
$ ralph org watch --stop --state-dir <scratch>/live/ledger
ralph org watch --all is not running on <scratch>/live/ledger
[rc=0]

## 1. --detach starts the watch
$ ralph org watch --all --detach --state-dir <scratch>/live/ledger
started ralph org watch --all (pid 9450); log: <scratch>/live/ledger/watch.log
[rc=0]
(--detach returned after .128 s)
pid1=9450
$ ps -o pid,ppid,pgid,sess,tty,stat,command -p 9450
  PID  PPID  PGID   SESS TTY      STAT COMMAND
 9450     1  9450      0 ??       Ss   <scratch>/ralph org watch --all --state-dir <scratch>/live/ledger
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 9450,
  "started_at": "2026-10-11T09:50:57.381811Z",
  "ts": "2026-10-11T09:50:57Z",
  "interval_seconds": 30
}
[rc=0]
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 11 18:50 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 11 18:50 ..
-rw-r--r--@ 1 <user>  wheel  145 Oct 11 18:50 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 11 18:50 watch.lock
-rw-r--r--@ 1 <user>  wheel  220 Oct 11 18:50 watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 9450, last cycle 2026-10-11T09:50:57Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "running",
    "pid": 9450,
    "started_at": "2026-10-11T09:50:57.381811Z",
    "last_cycle_ts": "2026-10-11T09:50:57Z",
    "mark_state": "running"
  }
}
[rc=0]
-- a second --detach while it runs
$ ralph org watch --all --detach --state-dir <scratch>/live/ledger
ralph org watch --all is already running (pid 9450); log: <scratch>/live/ledger/watch.log
[rc=0]

## 2. kill -9 the watch
$ kill -9 9450
[rc=0]
$ ps -p 9450 -o pid,command || true
  PID COMMAND
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 9450,
  "started_at": "2026-10-11T09:50:57.381811Z",
  "ts": "2026-10-11T09:50:57Z",
  "interval_seconds": 30
}
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (a running mark is left from pid 9450, last cycle 2026-10-11T09:50:57Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "not_running",
    "pid": 9450,
    "started_at": "2026-10-11T09:50:57.381811Z",
    "last_cycle_ts": "2026-10-11T09:50:57Z",
    "mark_state": "running"
  }
}
[rc=0]

## 3. --detach again starts a new watch (interval 2 s)
$ ralph org watch --all --detach --interval-seconds 2 --state-dir <scratch>/live/ledger
started ralph org watch --all (pid 9606); log: <scratch>/live/ledger/watch.log
[rc=0]
pid2=9606
$ ps -o pid,ppid,pgid,sess,tty,stat,command -p 9606
  PID  PPID  PGID   SESS TTY      STAT COMMAND
 9606     1  9606      0 ??       Ss   <scratch>/ralph org watch --all --state-dir <scratch>/live/ledger --interval-seconds 2
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 9606,
  "started_at": "2026-10-11T09:50:58.080265Z",
  "ts": "2026-10-11T09:50:58Z",
  "interval_seconds": 2
}
[rc=0]
-- 4.5 s later (two more cycles)
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 9606,
  "started_at": "2026-10-11T09:50:58.080265Z",
  "ts": "2026-10-11T09:51:02Z",
  "interval_seconds": 2
}
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 9606, last cycle 2026-10-11T09:51:02Z)
Run `ralph doctor` to check environment readiness.
[rc=0]

## 4. a second foreground watch while it runs
$ ralph org watch --all --once --state-dir <scratch>/live/ledger
watching every org (interval=30s once=true state-dir=<scratch>/live/ledger (source: flag))
org: watch: another ralph org watch --all is running on this state dir (pid 9606, from <scratch>/live/ledger/watch-alive.json)
[rc=1]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 9606,
  "started_at": "2026-10-11T09:50:58.080265Z",
  "ts": "2026-10-11T09:51:04Z",
  "interval_seconds": 2
}
[rc=0]

## 5. --stop
$ ralph org watch --stop --state-dir <scratch>/live/ledger
stopped ralph org watch --all (pid 9606)
[rc=0]
(--stop returned after .433 s)
$ ps -p 9606 -o pid,command || true
  PID COMMAND
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "stopped",
  "pid": 9606,
  "started_at": "2026-10-11T09:50:58.080265Z",
  "ts": "2026-10-11T09:51:04Z",
  "interval_seconds": 2,
  "stopped_at": "2026-10-11T09:51:05Z"
}
[rc=0]
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 11 18:51 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 11 18:50 ..
-rw-r--r--@ 1 <user>  wheel  184 Oct 11 18:51 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 11 18:50 watch.lock
-rw-r--r--@ 1 <user>  wheel  439 Oct 11 18:50 watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-11T09:51:05Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "not_running",
    "pid": 9606,
    "started_at": "2026-10-11T09:50:58.080265Z",
    "last_cycle_ts": "2026-10-11T09:51:04Z",
    "stopped_at": "2026-10-11T09:51:05Z",
    "mark_state": "stopped"
  }
}
[rc=0]
$ ralph org watch --stop --state-dir <scratch>/live/ledger
ralph org watch --all is not running on <scratch>/live/ledger
[rc=0]

## 6. watch.log
$ cat '<scratch>/live/ledger/watch.log'
watching every org (interval=30s once=false state-dir=<scratch>/live/ledger (source: flag))
watching every org (interval=2s once=false state-dir=<scratch>/live/ledger (source: flag))
[rc=0]

## 7. nothing left
$ pgrep -fl 'org watch --all' || echo '(pgrep: no process)'
(pgrep: no process)
[rc=0]
```

### 2. live2.sh(macOS、N=8)

```bash
#!/bin/bash
# Live check part 2: N `ralph org watch --all --detach` started at the same
# time on one fresh throw-away ledger. Same hermetic rules as live.sh.
set -u
S="$(cd "$(dirname "$0")" && pwd)"
N="${N:-8}"
L="$S/live2/ledger"
B="$S/ralph"
rm -rf "$S/live2"
mkdir -p "$S/live2/cwd" "$S/live2/home" "$S/live2/emptybin" "$S/live2/out"
cd "$S/live2/cwd" || exit 2

cleanup() {
  pgrep -f "$B org watch --all --state-dir $L" | while read -r p; do kill -9 "$p" 2>/dev/null; done
}
trap cleanup EXIT

echo "## $N --detach at once on a fresh ledger"
for i in $(seq 1 "$N"); do
  ( env -i PATH="$S/live2/emptybin" HOME="$S/live2/home" "$B" org watch --all --detach --state-dir "$L" \
      > "$S/live2/out/$i.out" 2>&1; echo $? > "$S/live2/out/$i.rc" ) &
done
wait
for i in $(seq 1 "$N"); do
  echo "[$i] rc=$(cat "$S/live2/out/$i.rc") $(tr '\n' ' ' < "$S/live2/out/$i.out")"
done
echo "-- exit 0: $(grep -lx 0 "$S"/live2/out/*.rc | wc -l | tr -d ' '), exit 1: $(grep -lx 1 "$S"/live2/out/*.rc | wc -l | tr -d ' ')"
echo "-- processes of this ledger:"
pgrep -fl "$B org watch --all --state-dir $L" || echo "(none)"
echo "-- mark:"
cat "$L/watch-alive.json"
echo "-- status:"
env -i PATH="$S/live2/emptybin" HOME="$S/live2/home" "$B" status --state-dir "$L" 2>&1 | grep 'org watch:'
echo "-- watch.log (lines by kind):"
sed -E 's/pid [0-9]+/pid N/; s/\(interval=.*/(...)/' "$L/watch.log" | sort | uniq -c
echo; echo "## --stop"
env -i PATH="$S/live2/emptybin" HOME="$S/live2/home" "$B" org watch --stop --state-dir "$L" 2>&1; echo "[rc=$?]"
pgrep -fl "$B org watch --all --state-dir $L" || echo "(no process left)"
```

1 回目の出力:

```text
## 8 --detach at once on a fresh ledger
[1] rc=1 org: watch --detach: the watch (pid 95459) exited before it was running; see <scratch>/live2/ledger/watch.log 
[2] rc=0 started ralph org watch --all (pid 95460); log: <scratch>/live2/ledger/watch.log 
[3] rc=0 ralph org watch --all is already running (pid 95460); log: <scratch>/live2/ledger/watch.log 
[4] rc=1 org: watch --detach: the watch (pid 95463) exited before it was running; see <scratch>/live2/ledger/watch.log 
[5] rc=0 ralph org watch --all is already running (pid 95460); log: <scratch>/live2/ledger/watch.log 
[6] rc=0 ralph org watch --all is already running (pid 95460); log: <scratch>/live2/ledger/watch.log 
[7] rc=0 ralph org watch --all is already running (pid 95460); log: <scratch>/live2/ledger/watch.log 
[8] rc=0 ralph org watch --all is already running (pid 95460); log: <scratch>/live2/ledger/watch.log 
-- exit 0: 6, exit 1: 2
-- processes of this ledger:
95460 <scratch>/ralph org watch --all --state-dir <scratch>/live2/ledger
-- mark:
{
  "state": "running",
  "pid": 95460,
  "started_at": "2026-10-11T10:02:20.285533Z",
  "ts": "2026-10-11T10:02:20Z",
  "interval_seconds": 30
}
-- status:
org watch: running (pid 95460, last cycle 2026-10-11T10:02:20Z)
-- watch.log (lines by kind):
   2 org: watch: another ralph org watch --all is running on this state dir (pid N, from <scratch>/live2/ledger/watch-alive.json)
   3 watching every org (...)

## --stop
stopped ralph org watch --all (pid 95460)
[rc=0]
(no process left)
```

2 回目の出力:

```text
## 8 --detach at once on a fresh ledger
[1] rc=0 started ralph org watch --all (pid 98901); log: <scratch>/live2/ledger/watch.log 
[2] rc=1 org: watch --detach: the watch (pid 98903) exited before it was running; see <scratch>/live2/ledger/watch.log 
[3] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
[4] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
[5] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
[6] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
[7] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
[8] rc=0 ralph org watch --all is already running (pid 98901); log: <scratch>/live2/ledger/watch.log 
-- exit 0: 7, exit 1: 1
-- processes of this ledger:
98901 <scratch>/ralph org watch --all --state-dir <scratch>/live2/ledger
-- mark:
{
  "state": "running",
  "pid": 98901,
  "started_at": "2026-10-11T10:02:35.217706Z",
  "ts": "2026-10-11T10:02:35Z",
  "interval_seconds": 30
}
-- status:
org watch: running (pid 98901, last cycle 2026-10-11T10:02:35Z)
-- watch.log (lines by kind):
   1 org: watch: another ralph org watch --all is running on this state dir (pid N, from <scratch>/live2/ledger/watch-alive.json)
   2 watching every org (...)

## --stop
stopped ralph org watch --all (pid 98901)
[rc=0]
(no process left)
```

3 回目の出力:

```text
## 8 --detach at once on a fresh ledger
[1] rc=0 started ralph org watch --all (pid 99265); log: <scratch>/live2/ledger/watch.log 
[2] rc=1 org: watch --detach: the watch (pid 99266) exited before it was running; see <scratch>/live2/ledger/watch.log 
[3] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
[4] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
[5] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
[6] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
[7] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
[8] rc=0 ralph org watch --all is already running (pid 99265); log: <scratch>/live2/ledger/watch.log 
-- exit 0: 7, exit 1: 1
-- processes of this ledger:
99265 <scratch>/ralph org watch --all --state-dir <scratch>/live2/ledger
-- mark:
{
  "state": "running",
  "pid": 99265,
  "started_at": "2026-10-11T10:02:38.491442Z",
  "ts": "2026-10-11T10:02:38Z",
  "interval_seconds": 30
}
-- status:
org watch: running (pid 99265, last cycle 2026-10-11T10:02:38Z)
-- watch.log (lines by kind):
   1 org: watch: another ralph org watch --all is running on this state dir (pid N, from <scratch>/live2/ledger/watch-alive.json)
   2 watching every org (...)

## --stop
stopped ralph org watch --all (pid 99265)
[rc=0]
(no process left)
```

4 回目の出力:

```text
## 8 --detach at once on a fresh ledger
[1] rc=0 started ralph org watch --all (pid 99615); log: <scratch>/live2/ledger/watch.log 
[2] rc=1 org: watch --detach: the watch (pid 99616) exited before it was running; see <scratch>/live2/ledger/watch.log 
[3] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
[4] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
[5] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
[6] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
[7] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
[8] rc=0 ralph org watch --all is already running (pid 99615); log: <scratch>/live2/ledger/watch.log 
-- exit 0: 7, exit 1: 1
-- processes of this ledger:
99615 <scratch>/ralph org watch --all --state-dir <scratch>/live2/ledger
-- mark:
{
  "state": "running",
  "pid": 99615,
  "started_at": "2026-10-11T10:02:41.712268Z",
  "ts": "2026-10-11T10:02:41Z",
  "interval_seconds": 30
}
-- status:
org watch: running (pid 99615, last cycle 2026-10-11T10:02:41Z)
-- watch.log (lines by kind):
   1 org: watch: another ralph org watch --all is running on this state dir (pid N, from <scratch>/live2/ledger/watch-alive.json)
   2 watching every org (...)

## --stop
stopped ralph org watch --all (pid 99615)
[rc=0]
(no process left)
```

### 3. live-linux.sh(ubuntu:24.04、ユーザー ubuntu)

```sh
#!/bin/sh
# Linux live check inside ubuntu:24.04, as user ubuntu: --detach (setsid),
# kill -9, --detach again, a second --once, --stop. Empty ledger, empty PATH
# for ralph.
L=/tmp/ledger; B=/t/ralph; mkdir -p /tmp/emptybin /tmp/cwd; cd /tmp/cwd
r() { echo "\$ ralph $*"; env -i PATH=/tmp/emptybin HOME=/tmp "$B" "$@" 2>&1; echo "[rc=$?]"; }
out=$(r org watch --all --detach --state-dir "$L"); echo "$out"
p1=$(echo "$out" | sed -n 's/.*(pid \([0-9]*\)).*/\1/p' | head -1)
echo "\$ ps -o pid,ppid,pgid,sid,tty,stat,args -p $p1"; ps -o pid,ppid,pgid,sid,tty,stat,args -p "$p1"
echo "(this shell: pid $$, sid $(ps -o sid= -p $$))"
r status --state-dir "$L" | grep 'org watch:\|rc='
echo "\$ kill -9 $p1"; kill -9 "$p1"; sleep 0.3
r status --state-dir "$L" | grep 'org watch:\|rc='
out=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out"
p2=$(echo "$out" | sed -n 's/.*(pid \([0-9]*\)).*/\1/p' | head -1)
ps -o pid,ppid,pgid,sid,tty,stat,args -p "$p2"
r org watch --all --once --state-dir "$L"
r org watch --stop --state-dir "$L"
r status --state-dir "$L" | grep 'org watch:\|rc='
echo "\$ ps -eo pid,args | grep 'org watch'"; ps -eo pid,args | grep '[o]rg watch' || echo "(none)"
echo "\$ cat $L/watch.log"; cat "$L/watch.log"
```

```text
$ ralph org watch --all --detach --state-dir /tmp/ledger
started ralph org watch --all (pid 20); log: /tmp/ledger/watch.log
[rc=0]
$ ps -o pid,ppid,pgid,sid,tty,stat,args -p 20
    PID    PPID    PGID     SID TT       STAT COMMAND
     20       1      20      20 ?        Ssl  /t/ralph org watch --all --state-dir /tmp/ledger
(this shell: pid 12, sid      11)
org watch: running (pid 20, last cycle 2026-10-11T10:05:01Z)
[rc=0]
$ kill -9 20
org watch: not running (a running mark is left from pid 20, last cycle 2026-10-11T10:05:01Z)
[rc=0]
$ ralph org watch --all --detach --interval-seconds 2 --state-dir /tmp/ledger
started ralph org watch --all (pid 52); log: /tmp/ledger/watch.log
[rc=0]
    PID    PPID    PGID     SID TT       STAT COMMAND
     52       1      52      52 ?        Ssl  /t/ralph org watch --all --state-dir /tmp/ledger --interval-seconds 2
$ ralph org watch --all --once --state-dir /tmp/ledger
watching every org (interval=30s once=true state-dir=/tmp/ledger (source: flag))
org: watch: another ralph org watch --all is running on this state dir (pid 52, from /tmp/ledger/watch-alive.json)
[rc=1]
$ ralph org watch --stop --state-dir /tmp/ledger
stopped ralph org watch --all (pid 52)
[rc=0]
org watch: not running (stopped at 2026-10-11T10:05:04Z)
[rc=0]
$ ps -eo pid,args | grep 'org watch'
(none)
$ cat /tmp/ledger/watch.log
watching every org (interval=30s once=false state-dir=/tmp/ledger (source: flag))
watching every org (interval=2s once=false state-dir=/tmp/ledger (source: flag))
```

## cycle 2: HEAD b0a226a9 での再確認(2026-10-12)

上の cycle 1 の記録は dd63a414 のビルドの出力なので、`ralph status` の `last cycle …` と `--json` の `last_cycle_ts` が出ている。この 2 つは d2a750ec で `mark updated …` と `updated_at` に名前が変わった。

- Date: 2026-10-12(JST 00:13〜00:15)
- 報告: docs/reports/test-2026-10-11-org-cross-watch.md の「Latest run: cycle 2」
- 目的: cross-review の直し(49d8d605: watcher の後始末の間もロックを持つ、起動のときの止める頼みを rename で取り出す)と d2a750ec(`mark updated` と `updated_at`)のあとの HEAD をビルドし、cycle 1 の流れをもう一度回す(verify の cycle 2 の「足すと確度がいちばん上がる確認」)。あわせて、起動の前に台帳に置いた古い止める頼みと、残った `.watch-stop.json.taken` の扱いを見た

### 環境(cycle 2)

| 項目 | 値 |
|---|---|
| ralph | このブランチの HEAD b0a226a9 を `go build -o <scratch>/ralph ./cmd/ralph` でビルドしたもの(`ralph version` は `dev`) |
| 台帳 | `<scratch>/live/ledger` と `<scratch>/live2/ledger`。どちらも初めは存在しないか空で、manifest も受信箱も置いていない(見張りは herdr も agmsg も呼ばず、期限の件もないのでデスクトップ通知は出ない) |
| ralph の起こし方 | cycle 1 と同じ。すべて `env -i PATH=<空のディレクトリ> HOME=<scratch の下> ralph ... --state-dir <台帳>`。cwd は `ralph.toml` のないディレクトリ(既定の設定) |
| 触れていないもの | 利用者の herdr サーバー、`~/.config/herdr`、本物の台帳。osascript の本物の経路は確かめていない。seat は置いていない |
| 片づけ | スクリプトの終わりに、起動した見張りを `kill -9` する trap(どれもすでに終わっていた)。始める前と最後の `pgrep -fl 'org watch --all'` は何も返さなかった |

`<scratch>` は `/private/tmp/claude-502/<セッション>/scratchpad/ocw2`。`ls` の所有者名は `<user>` に置き換えた。`ls -la` は `.` で始まるファイルも出すので、`.watch-stop.json.taken` があれば一覧に出る。

### 結果(cycle 2)

| 確認 | 結果 |
|---|---|
| 台帳がないときの `--stop` | rc 0、「not running」 |
| `--detach`(1 回目) | rc 0、`started ralph org watch --all (pid 41263)`、0.134 s で返った。子は `PGID == PID`、`TTY ??`、`STAT Ss`、`PPID 1` |
| `ralph status` / `--json` | `org watch: running (pid 41263, mark updated 2026-10-11T15:13:45Z)`。`org_watch` は `updated_at` を持ち、`last_cycle_ts` はない |
| `kill -9 41263` のあと | 印は running のまま残り、status は `org watch: not running (a running mark is left from pid 41263, mark updated …)`、`--json` は `state: not_running`、`mark_state: running` |
| `--detach --interval-seconds 2`(2 回目) | rc 0、新しい pid 41370。4.5 s 後に印の `ts` が 15:13:46 から 15:13:50 に進んだ |
| 動いている間の `--all --once` | rc 1、`another ralph org watch --all is running on this state dir (pid 41370, from …/watch-alive.json)` |
| `--stop` | rc 0、`stopped ralph org watch --all (pid 41370)`、0.436 s で返った。印は `stopped` で `stopped_at` つき。台帳に `watch-stop.json` も `.watch-stop.json.taken` もない。status は `org watch: not running (stopped at …)` |
| 古い頼み(`requested_at` 2026-10-01)を手で置いてからの `--detach` | rc 0、pid 42521。起動の直後に `watch-stop.json` は消え、`.taken` も残らない。4.5 s 後もプロセスは動き、`ts` は進んでいた(古い頼みに従わない)。`--stop` で止まった |
| `.taken` と古い `watch-stop.json` が並ぶ台帳での `--detach` | 起動の取り出しが `.taken` を上書きしてから消すので、どちらも残らない |
| `.taken` だけが残った台帳での `--detach` | 見張りは普通に動き、`--stop` で止まる。`.taken` は残り続ける(次に頼みのある起動まで消えない。読まれないファイルなので動きには関わらない) |
| 残ったプロセス | なし |

### 1. live.sh(macOS、cycle 2)

cycle 1 の live.sh に、手順 6(起動の前に置いた古い頼み)と手順 8 の `watch-stop` の数を足したもの。

```bash
#!/bin/bash
# Cycle 2 live check of `ralph org watch --all --detach/--stop` at HEAD
# b0a226a9, on a throw-away ledger. Same rules as cycle 1's live.sh: every
# ralph command runs under env -i with an empty PATH (no herdr, agmsg, git,
# or osascript can be found) and a scratch HOME, from a scratch cwd with no
# ralph.toml; --state-dir on every command; no manifest and no inbox in the
# ledger, so the watch calls nothing outside itself and shows no
# notification.
set -u
S="$(cd "$(dirname "$0")" && pwd)"
L="$S/live/ledger"
B="$S/ralph"
rm -rf "$S/live"
mkdir -p "$S/live/cwd" "$S/live/home" "$S/live/emptybin"
cd "$S/live/cwd" || exit 2

r() { # run a ralph command hermetically, print "$ cmd", output, rc
  echo "\$ ralph $*"
  env -i PATH="$S/live/emptybin" HOME="$S/live/home" "$B" "$@" 2>&1
  local rc=$?
  echo "[rc=$rc]"
  return $rc
}
sh_() { echo "\$ $*"; eval "$@" 2>&1; echo "[rc=$?]"; }
pid_of() { sed -n 's/.*(pid \([0-9][0-9]*\)).*/\1/p' | head -1; }
now() { perl -MTime::HiRes=time -e 'printf "%.3f", time'; }

started=()
cleanup() {
  for p in "${started[@]}"; do kill -9 "$p" 2>/dev/null; done
}
trap cleanup EXIT

echo "## 0. the ledger does not exist yet"
sh_ "ls -la '$L'"
r org watch --stop --state-dir "$L"

echo; echo "## 1. --detach starts the watch"
t0=$(now)
out1=$(r org watch --all --detach --state-dir "$L"); echo "$out1"
t1=$(now)
echo "(--detach returned after $(echo "$t1 - $t0" | bc) s)"
pid1=$(echo "$out1" | pid_of); started+=("$pid1")
echo "pid1=$pid1"
sh_ "ps -o pid,ppid,pgid,sess,tty,stat,command -p $pid1"
sh_ "cat '$L/watch-alive.json'"
sh_ "ls -la '$L'"
r status --state-dir "$L"
r status --state-dir "$L" --json

echo; echo "## 2. kill -9 the watch"
sh_ "kill -9 $pid1"
sleep 0.3
sh_ "ps -p $pid1 -o pid,command || true"
sh_ "cat '$L/watch-alive.json'"
r status --state-dir "$L"
r status --state-dir "$L" --json

echo; echo "## 3. --detach again starts a new watch (interval 2 s)"
out2=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out2"
pid2=$(echo "$out2" | pid_of); started+=("$pid2")
echo "pid2=$pid2"
sh_ "ps -o pid,ppid,pgid,sess,tty,stat,command -p $pid2"
sh_ "cat '$L/watch-alive.json'"
sleep 4.5
echo "-- 4.5 s later (two more cycles)"
sh_ "cat '$L/watch-alive.json'"
r status --state-dir "$L"

echo; echo "## 4. a second foreground watch while it runs"
r org watch --all --once --state-dir "$L"

echo; echo "## 5. --stop"
t0=$(now)
r org watch --stop --state-dir "$L"
t1=$(now)
echo "(--stop returned after $(echo "$t1 - $t0" | bc) s)"
sh_ "ps -p $pid2 -o pid,command || true"
sh_ "cat '$L/watch-alive.json'"
sh_ "ls -la '$L'"
r status --state-dir "$L"
r status --state-dir "$L" --json

echo; echo "## 6. a stop request left in the ledger before --detach (requested_at older than the start)"
printf '{\n  "requested_at": "2026-10-01T00:00:00Z"\n}\n' > "$L/watch-stop.json"
sh_ "cat '$L/watch-stop.json'"
out3=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out3"
pid3=$(echo "$out3" | pid_of); started+=("$pid3")
echo "pid3=$pid3"
sh_ "ls -la '$L'"
sleep 4.5
echo "-- 4.5 s later: still running, the old request did not stop it"
sh_ "ps -p $pid3 -o pid,command"
sh_ "cat '$L/watch-alive.json'"
r status --state-dir "$L"
r org watch --stop --state-dir "$L"
sh_ "ls -la '$L'"
r status --state-dir "$L"

echo; echo "## 7. watch.log"
sh_ "cat '$L/watch.log'"

echo; echo "## 8. nothing left"
sh_ "ls -a '$L' | grep -c 'watch-stop' || true"
sh_ "pgrep -fl 'org watch --all' || echo '(pgrep: no process)'"
```

```text
## 0. the ledger does not exist yet
$ ls -la '<scratch>/live/ledger'
ls: <scratch>/live/ledger: No such file or directory
[rc=1]
$ ralph org watch --stop --state-dir <scratch>/live/ledger
ralph org watch --all is not running on <scratch>/live/ledger
[rc=0]

## 1. --detach starts the watch
$ ralph org watch --all --detach --state-dir <scratch>/live/ledger
started ralph org watch --all (pid 41263); log: <scratch>/live/ledger/watch.log
[rc=0]
(--detach returned after .134 s)
pid1=41263
$ ps -o pid,ppid,pgid,sess,tty,stat,command -p 41263
  PID  PPID  PGID   SESS TTY      STAT COMMAND
41263     1 41263      0 ??       Ss   <scratch>/ralph org watch --all --state-dir <scratch>/live/ledger
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 41263,
  "started_at": "2026-10-11T15:13:45.723319Z",
  "ts": "2026-10-11T15:13:45Z",
  "interval_seconds": 30
}
[rc=0]
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 12 00:13 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 12 00:13 ..
-rw-r--r--@ 1 <user>  wheel  146 Oct 12 00:13 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 12 00:13 watch.lock
-rw-r--r--@ 1 <user>  wheel  216 Oct 12 00:13 watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 41263, mark updated 2026-10-11T15:13:45Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "running",
    "pid": 41263,
    "started_at": "2026-10-11T15:13:45.723319Z",
    "updated_at": "2026-10-11T15:13:45Z",
    "mark_state": "running"
  }
}
[rc=0]

## 2. kill -9 the watch
$ kill -9 41263
[rc=0]
$ ps -p 41263 -o pid,command || true
  PID COMMAND
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 41263,
  "started_at": "2026-10-11T15:13:45.723319Z",
  "ts": "2026-10-11T15:13:45Z",
  "interval_seconds": 30
}
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (a running mark is left from pid 41263, mark updated 2026-10-11T15:13:45Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "not_running",
    "pid": 41263,
    "started_at": "2026-10-11T15:13:45.723319Z",
    "updated_at": "2026-10-11T15:13:45Z",
    "mark_state": "running"
  }
}
[rc=0]

## 3. --detach again starts a new watch (interval 2 s)
$ ralph org watch --all --detach --interval-seconds 2 --state-dir <scratch>/live/ledger
started ralph org watch --all (pid 41370); log: <scratch>/live/ledger/watch.log
[rc=0]
pid2=41370
$ ps -o pid,ppid,pgid,sess,tty,stat,command -p 41370
  PID  PPID  PGID   SESS TTY      STAT COMMAND
41370     1 41370      0 ??       Ss   <scratch>/ralph org watch --all --state-dir <scratch>/live/ledger --interval-seconds 2
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 41370,
  "started_at": "2026-10-11T15:13:46.289478Z",
  "ts": "2026-10-11T15:13:46Z",
  "interval_seconds": 2
}
[rc=0]
-- 4.5 s later (two more cycles)
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 41370,
  "started_at": "2026-10-11T15:13:46.289478Z",
  "ts": "2026-10-11T15:13:50Z",
  "interval_seconds": 2
}
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 41370, mark updated 2026-10-11T15:13:50Z)
Run `ralph doctor` to check environment readiness.
[rc=0]

## 4. a second foreground watch while it runs
$ ralph org watch --all --once --state-dir <scratch>/live/ledger
watching every org (interval=30s once=true state-dir=<scratch>/live/ledger (source: flag))
org: watch: another ralph org watch --all is running on this state dir (pid 41370, from <scratch>/live/ledger/watch-alive.json)
[rc=1]

## 5. --stop
$ ralph org watch --stop --state-dir <scratch>/live/ledger
stopped ralph org watch --all (pid 41370)
[rc=0]
(--stop returned after .436 s)
$ ps -p 41370 -o pid,command || true
  PID COMMAND
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "stopped",
  "pid": 41370,
  "started_at": "2026-10-11T15:13:46.289478Z",
  "ts": "2026-10-11T15:13:52Z",
  "interval_seconds": 2,
  "stopped_at": "2026-10-11T15:13:53Z"
}
[rc=0]
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 12 00:13 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 12 00:13 ..
-rw-r--r--@ 1 <user>  wheel  185 Oct 12 00:13 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 12 00:13 watch.lock
-rw-r--r--@ 1 <user>  wheel  431 Oct 12 00:13 watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-11T15:13:53Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger --json
{
  "state_dir": "<scratch>/live/ledger",
  "state_dir_source": "flag",
  "orgs": [],
  "org_watch": {
    "state": "not_running",
    "pid": 41370,
    "started_at": "2026-10-11T15:13:46.289478Z",
    "updated_at": "2026-10-11T15:13:52Z",
    "stopped_at": "2026-10-11T15:13:53Z",
    "mark_state": "stopped"
  }
}
[rc=0]

## 6. a stop request left in the ledger before --detach (requested_at older than the start)
$ cat '<scratch>/live/ledger/watch-stop.json'
{
  "requested_at": "2026-10-01T00:00:00Z"
}
[rc=0]
$ ralph org watch --all --detach --interval-seconds 2 --state-dir <scratch>/live/ledger
started ralph org watch --all (pid 42521); log: <scratch>/live/ledger/watch.log
[rc=0]
pid3=42521
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 12 00:13 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 12 00:13 ..
-rw-r--r--@ 1 <user>  wheel  144 Oct 12 00:13 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 12 00:13 watch.lock
-rw-r--r--@ 1 <user>  wheel  646 Oct 12 00:13 watch.log
[rc=0]
-- 4.5 s later: still running, the old request did not stop it
$ ps -p 42521 -o pid,command
  PID COMMAND
42521 <scratch>/ralph org watch --all --state-dir <scratch>/live/ledger --interval-seconds 2
[rc=0]
$ cat '<scratch>/live/ledger/watch-alive.json'
{
  "state": "running",
  "pid": 42521,
  "started_at": "2026-10-11T15:13:53.48836Z",
  "ts": "2026-10-11T15:13:57Z",
  "interval_seconds": 2
}
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 42521, mark updated 2026-10-11T15:13:57Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph org watch --stop --state-dir <scratch>/live/ledger
stopped ralph org watch --all (pid 42521)
[rc=0]
$ ls -la '<scratch>/live/ledger'
total 16
drwxr-xr-x@ 5 <user>  wheel  160 Oct 12 00:13 .
drwxr-xr-x@ 6 <user>  wheel  192 Oct 12 00:13 ..
-rw-r--r--@ 1 <user>  wheel  184 Oct 12 00:13 watch-alive.json
-rw-r--r--@ 1 <user>  wheel    0 Oct 12 00:13 watch.lock
-rw-r--r--@ 1 <user>  wheel  646 Oct 12 00:13 watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live/ledger
no org runtime state found (state-dir=<scratch>/live/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: not running (stopped at 2026-10-11T15:13:58Z)
Run `ralph doctor` to check environment readiness.
[rc=0]

## 7. watch.log
$ cat '<scratch>/live/ledger/watch.log'
watching every org (interval=30s once=false state-dir=<scratch>/live/ledger (source: flag))
watching every org (interval=2s once=false state-dir=<scratch>/live/ledger (source: flag))
watching every org (interval=2s once=false state-dir=<scratch>/live/ledger (source: flag))
[rc=0]

## 8. nothing left
$ ls -a '<scratch>/live/ledger' | grep -c 'watch-stop' || true
0
[rc=0]
$ pgrep -fl 'org watch --all' || echo '(pgrep: no process)'
(pgrep: no process)
[rc=0]
```

### 2. live2.sh(macOS、残った `.watch-stop.json.taken`)

```bash
#!/bin/bash
# Cycle 2 live check, part 2: a `.watch-stop.json.taken` left in the ledger
# (as if a watch were killed between its take and the remove) before a
# --detach, with and without a watch-stop.json next to it. Same hermetic
# rules as live.sh.
set -u
S="$(cd "$(dirname "$0")" && pwd)"
L="$S/live2/ledger"
B="$S/ralph"
rm -rf "$S/live2"
mkdir -p "$S/live2/cwd" "$S/live2/home" "$S/live2/emptybin" "$L"
cd "$S/live2/cwd" || exit 2

r() {
  echo "\$ ralph $*"
  env -i PATH="$S/live2/emptybin" HOME="$S/live2/home" "$B" "$@" 2>&1
  local rc=$?
  echo "[rc=$rc]"
  return $rc
}
sh_() { echo "\$ $*"; eval "$@" 2>&1; echo "[rc=$?]"; }
pid_of() { sed -n 's/.*(pid \([0-9][0-9]*\)).*/\1/p' | head -1; }

started=()
cleanup() {
  for p in "${started[@]}"; do kill -9 "$p" 2>/dev/null; done
}
trap cleanup EXIT

echo "## A. a left .taken (old request) and an old watch-stop.json"
printf '{"requested_at": "2026-09-30T00:00:00Z"}\n' > "$L/.watch-stop.json.taken"
printf '{"requested_at": "2026-10-01T00:00:00Z"}\n' > "$L/watch-stop.json"
sh_ "ls -a '$L'"
out=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out"
p=$(echo "$out" | pid_of); started+=("$p")
sleep 3
sh_ "ls -a '$L'"
r status --state-dir "$L"
r org watch --stop --state-dir "$L"

echo; echo "## B. a left .taken alone (no watch-stop.json)"
printf '{"requested_at": "2026-09-30T00:00:00Z"}\n' > "$L/.watch-stop.json.taken"
sh_ "ls -a '$L'"
out=$(r org watch --all --detach --interval-seconds 2 --state-dir "$L"); echo "$out"
p=$(echo "$out" | pid_of); started+=("$p")
sleep 3
sh_ "ls -a '$L'"
r status --state-dir "$L"
r org watch --stop --state-dir "$L"
sh_ "ls -a '$L'"
sh_ "grep -c watchdog '$L/watch.log' || true"

echo; echo "## nothing left"
sh_ "pgrep -fl 'org watch --all' || echo '(pgrep: no process)'"
```

```text
## A. a left .taken (old request) and an old watch-stop.json
$ ls -a '<scratch>/live2/ledger'
.
..
.watch-stop.json.taken
watch-stop.json
[rc=0]
$ ralph org watch --all --detach --interval-seconds 2 --state-dir <scratch>/live2/ledger
started ralph org watch --all (pid 48807); log: <scratch>/live2/ledger/watch.log
[rc=0]
$ ls -a '<scratch>/live2/ledger'
.
..
watch-alive.json
watch.lock
watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live2/ledger
no org runtime state found (state-dir=<scratch>/live2/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 48807, mark updated 2026-10-11T15:14:27Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph org watch --stop --state-dir <scratch>/live2/ledger
stopped ralph org watch --all (pid 48807)
[rc=0]

## B. a left .taken alone (no watch-stop.json)
$ ls -a '<scratch>/live2/ledger'
.
..
.watch-stop.json.taken
watch-alive.json
watch.lock
watch.log
[rc=0]
$ ralph org watch --all --detach --interval-seconds 2 --state-dir <scratch>/live2/ledger
started ralph org watch --all (pid 49516); log: <scratch>/live2/ledger/watch.log
[rc=0]
$ ls -a '<scratch>/live2/ledger'
.
..
.watch-stop.json.taken
watch-alive.json
watch.lock
watch.log
[rc=0]
$ ralph status --state-dir <scratch>/live2/ledger
no org runtime state found (state-dir=<scratch>/live2/ledger (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 49516, mark updated 2026-10-11T15:14:31Z)
Run `ralph doctor` to check environment readiness.
[rc=0]
$ ralph org watch --stop --state-dir <scratch>/live2/ledger
stopped ralph org watch --all (pid 49516)
[rc=0]
$ ls -a '<scratch>/live2/ledger'
.
..
.watch-stop.json.taken
watch-alive.json
watch.lock
watch.log
[rc=0]
$ grep -c watchdog '<scratch>/live2/ledger/watch.log' || true
0
[rc=0]

## nothing left
$ pgrep -fl 'org watch --all' || echo '(pgrep: no process)'
(pgrep: no process)
[rc=0]
```
