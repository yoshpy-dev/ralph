# 横断の見張りの検査の実機確認(org-watch-gate、/test cycle 1)

- Date: 2026-10-12(実機確認は JST 14:50〜14:53、base との比較は 14:38〜14:41)
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(AC1〜AC5、AC9、AC13〜AC15、F7、F10)
- 報告: docs/reports/test-2026-10-12-org-watch-gate.md
- 目的: ビルドしたバイナリと一時の台帳で、見張りがないときの spawn と TASK の拒否、`--detach` の起動、`kill -9` のあとの start の起動、古い見張りへの拒否と待ち時間、待つ間にロックの持ち主が消えたときの起動、`--once` と止まりかけの見張りへの拒否、同時の `--detach` を確かめる。終わりに、見張りのプロセスが残っていないことを `pgrep` で見る

## 環境

| 項目 | 値 |
|---|---|
| ralph | このブランチの HEAD ae349cc4 を `go build -o <scratch>/ralph ./cmd/ralph` でビルドしたもの。7 の比較では、base dc41c4f5 を `git archive` で scratch に出してビルドした `<scratch>/ralph-base` も使った |
| 起こし方 | すべて `env -i HOME=<scratch>/live/home PATH=<scratch>/live/bin SHELL=/bin/bash XDG_CONFIG_HOME=… XDG_DATA_HOME=… XDG_STATE_HOME=… XDG_CACHE_HOME=… XDG_RUNTIME_DIR=…`。XDG の 5 つは `<scratch>/live/xdg/` の下 |
| PATH | `/usr/bin` と `/bin` の全項目へのシンボリックリンクを置いたディレクトリから、`osascript` だけを外したもの。依頼の `PATH=/usr/bin:/bin` から、デスクトップ通知の経路を断つために 1 つ外した。herdr と agmsg は、もともとどちらのディレクトリにもない |
| cwd | `<scratch>/live/wd`。`ralph.toml` がなく、git の外にある。設定は既定(`deadman_minutes` 10、間隔 30 秒) |
| 台帳 | どれも `--state-dir <scratch>/live/<A、B、B2、B3、C、D、E1〜E3>` で、初めは存在しない。repo の外なので、起動される見張りは打った場所(cwd)で動く |
| ロックを持つ補助 | `holder.py`(全文は下)。python の `fcntl.flock` で `watch.lock` を取り、印を書いて、SIGTERM まで眠る。ralph の見張りと同じ flock(2) のロックを使う |
| `--once` の長いサイクル | C の台帳に、`e1` の escalated の行と、同じ `e1` の resolved の行 4,000,000 行(約 280 MB)の `inbox.jsonl` を置いた。見張りは 1 サイクルの初めに受信箱を読むので、1 サイクルに 5.7 秒かかる。2 行目からの resolved は無視され、`e1` は解決済みなので、期限の通知は出ない。manifest は置いていないので、走っている org はなく、herdr も agmsg も呼ばれない。`ralph status` は受信箱を読まないので、見張りが動いている間に結果が出る |
| 触れていないもの | 利用者の herdr サーバー、`~/.config/herdr`、本物の台帳。osascript は PATH にないので、デスクトップ通知は出ない |
| 片づけ | 最後に holder を止め、どの台帳にも `--stop` を打った。`pgrep -fl '<scratch>/ralph org watch'` は何も返さず rc 1。EXIT の trap は残った見張りを `kill -9` するが、何も残っていなかった |

`<scratch>` は `/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test`。下の「スクリプトと出力」のコードブロックには出力をそのまま貼ったので、パスは置き換えていない。spawn と start は断られると stdout に `rejected: <エラー>` を出し、同じエラーが stderr にも出る(スクリプトは両方を 1 つにまとめて出す)。

## 結果

### 1. 見張りのない台帳(A)

| 確認 | 結果 |
|---|---|
| `ralph status` | `org watch: not running` |
| `spawn --dry-run` | rc 1、0.1 s。not_running の文で、次の手として `ralph org watch --all --detach --state-dir '<台帳>'` と、新しい org なら `ralph org start` を挙げる。台帳のディレクトリはできていない |
| 型付きの TASK と `--raw` の TASK の `send --dry-run` | どちらも rc 1。`org: send: TASK not sent: no ralph org watch --all is running …`。台帳はまだない |
| QUESTION の `send --dry-run` | rc 0。manifest はこの dry-run の `sent` の 1 行だけ |

### 2. `--detach` の起動(A)

| 確認 | 結果 |
|---|---|
| 1 回目 | rc 0、0.2 s。`started ralph org watch --all (pid 45166)` |
| 2 回目 | rc 0。`already running (pid 45166)` |
| 印 | running、`deadman_minutes: 10`。`once` はない |
| そのあとの spawn と TASK の `--dry-run` | どちらも rc 0 |

### 3. `kill -9` のあと(A)

| 確認 | 結果 |
|---|---|
| `kill -9 45166` のあとの status | `not running (a running mark is left from pid 45166, …)` |
| `spawn --dry-run` | rc 1。not_running の文 |
| `start <task>` | 最初の行が stderr の `ralph org start: started the cross-org watch (pid 45256); log: <台帳>/watch.log`。そのあと leader の spawn が `herdr: exec: "herdr": executable file not found in $PATH` で失敗して rc 1(herdr がないので想定どおり) |
| manifest | start が足したのは leader の `spawn_started` と `spawn_failed` だけ。断られた spawn と send は 1 行も足していない |
| start のあと | pid 45256 の見張りは動き続け、3 秒後も status は running。`--stop` で `stopped ralph org watch --all (pid 45256)` |

### 4. 古い見張り(B: ロックを持ったまま 1 時間前の印)

| 確認 | 結果 |
|---|---|
| status | `org watch: stale (pid 45572, …)` |
| `spawn --dry-run` と TASK の `send --dry-run` | どちらも rc 1。2.1 s と 2.0 s で、settle の 2 秒を待ってから断る。文は `--stop --state-dir '<台帳>'` と `kill 45572` を挙げる |
| `--detach` | rc 1、10.0 s。`a watch holds … but its liveness mark is not fresh (pid 45572), so no new watch runs` |
| `start <task>` | rc 1、10.1 s。spawn と同じ stale の文で、見張りは起動していない |
| 台帳 | `watch-alive.json` と `watch.lock` だけで、manifest はない。B の見張りのプロセスもない |

### 5. 待つ間にロックの持ち主がいなくなる(B2、B3)

verify の報告が「テストがない」とした `ensureOrgWatch` の分岐(stale を待つ間に not_running になると起動へ進む)を、本物のプロセスで確かめた。

| 確認 | 結果 |
|---|---|
| B2: 1 時間前の印を持つ holder を、`--detach` を打ってから 3 秒後に kill | rc 0、3.2 s。`started ralph org watch --all (pid 49475)`。そのあと status は running、`--stop` で止まった |
| B3: 印のない holder(stale)を、`start <task>` を打ってから 3 秒後に kill | 3.2 s で `ralph org start: started the cross-org watch (pid 49710)`。そのあと herdr がないので rc 1。見張りは残り、`--stop` で止まった |

### 6. `--once` の見張りと止まりかけの見張り(C、D)

| 確認 | C: `--once` が動いている間(本物の見張り) | D: 新しい stopping の印を holder が持つ(手で置いた印) |
|---|---|---|
| 印 | running、`once: true`、pid 49827 | stopping、pid 50289 |
| `ralph status` | `org watch: running (--once) (pid 49827, …) -- it ends after its cycle and accepts no new work` | `org watch: stopping (pid 50289, …) -- it has run its last cycle and accepts no new work; …` |
| `status --json` の `org_watch` | `"state":"running"`、`"once":true` | `"state":"stopping"`、`"mark_state":"stopping"` |
| `spawn --dry-run`、TASK の `send --dry-run` | どちらも rc 1、0.1 s 以下。`is a --once run, which ends after its cycle and accepts no new work` | どちらも rc 1、0.1 s 以下。`is stopping and accepts no new work` |
| `--detach` | rc 1、待たずに断る。`is a --once run, which ends after its cycle, so no new watch runs` | rc 1、待たずに断る。`is stopping, so no new watch runs` |
| `start <task>` | rc 1、0.1 s。起動しない | rc 1、0.1 s。起動しない |
| そのあと | 見張りは 5.7 s で終わり rc 0。status は `not running (stopped at …)` | holder を止めた |

本物の見張りが後始末(drain)の間に書く stopping の印は、watcher の呼び出しがなければすぐ stopped に変わるので、実機では捕まえられなかった。D は印を手で置いて、読み手(status、検査、`--detach`、start)の側だけを確かめたもの。

### 7. 同時の `--detach` 8 本(E1〜E3)と、そのすぐあとの `--stop`

| 回 | `--detach` の rc | `started` | すべて返った時点の `ralph org watch --all --state-dir <台帳>` のプロセス | すぐあとの `--stop` |
|---|---|---|---|---|
| 1 | 8 本とも 0 | 1(pid 50394)、ほか 7 本は `already running (pid 50394)` | 6 | rc 1、30.1 s で時間切れ |
| 2 | 8 本とも 0 | 1(pid 55062) | 6 | rc 1、30.0 s で時間切れ |
| 3 | 8 本とも 0 | 1(pid 59530) | 4 | rc 1、30.0 s で時間切れ |

ロックを持つ見張りはどの時点でも 1 つで、8 本とも rc 0 になった(cycle 1 の 6a では 1〜2 本が rc 1 だった。tech-debt 206 の直し)。一方で、`--stop` は 3 回とも時間切れになった。終わりの F で、残った見張り(pid 50398、55071、59544)を `--stop` で止めた。どれも `started` と言われた pid とは違う。

### 8. base との比較(`takeover.sh`)

| 手順 | base dc41c4f5 | HEAD ae349cc4 |
|---|---|---|
| `--detach` で見張りを起動し、foreground の `watch --all` をロック待ちにしてから `--stop` | `--stop` は 30 s で時間切れ。待っていた見張りがロックを取って動き続ける | 同じ |
| 同時の `--detach` 8 本がすべて返った直後に `--stop` | rc 0 が 7 本、rc 1 が 1 本。返った時点の見張りのプロセスは 1。`--stop` は 0.97 s で rc 0、そのあと 0 | rc 0 が 8 本。返った時点のプロセスは 6。`--stop` は 30.06 s で rc 1、そのあと 1 つ動き続ける |

## 見つけたこと: 同時の起動のすぐあとの `--stop` が効かない

7 と 8 の出力から、次のように読める。

1. ロックの取り合いに負けた子(`ralph org watch --all --state-dir <台帳>`)は、`AcquireWatchLock` の待ち(既定 2 秒)の間、生きている
2. HEAD の `--detach`(と start)は、ほかの見張りの印が running になった時点で rc 0 を返す(AC5)。自分の子が終わるのは待たない。base では子が断られて終わるまで待って rc 1 を返していたので、`--detach` がすべて返った時点で負けた子は残っていなかった
3. その 2 秒の間に `--stop` を打つと、勝った見張りは頼みを見て止まり、`stopAndRelease` で頼み(`watch-stop.json`)を消してからロックを放す
4. ロックを待っていた子がロックを取る。頼みはもう消えているので、その子は長く動く見張りとして動き続ける。E の各台帳の `watch.log` には banner が 2 行あり(ロックを取った見張りが 2 つ)、残りの子は `another ralph org watch --all is running on this state dir (pid <2 つ目>)` で断られている
5. `--stop` はロックが放されるのを 30 秒待って時間切れになる。止めたい見張りは止まらず、別の pid の見張りが残る

仕組み(ロックを待つ見張りが、止まった見張りのあとを取る)は base にもある(8 の 1 行目)。HEAD で変わったのは、同時の start と `--detach` が返ったときに、ロックを待つ子がまだ残っている点で、「すべての start が返ってから `--stop`」という普通の順でも起きる。窓はロックの待ちの 2 秒で、打ち手が 2 秒以上空ければ起きない。

## スクリプトと出力

以下のコードブロックは、スクリプトと出力のファイルを、そのまま byte 単位で貼ったもの。`takeover.out` の最後の `rc=0` は、`bash takeover.sh > takeover.out 2>&1; echo "rc=$?" >> takeover.out` の `echo` の行。

### live.sh

```bash
#!/bin/bash
# Live check of org-watch-gate (HEAD ae349cc4) with the built binary.
# Every ralph runs under env -i with a scratch HOME, XDG dirs under the
# scratch dir, and PATH = a dir of symlinks to every /usr/bin and /bin entry
# except osascript (so no desktop notification can be shown). No herdr or
# agmsg is on that PATH. cwd has no ralph.toml and is not a git repo.
set -u
SC=$(cd "$(dirname "$0")" && pwd)
B=$SC/ralph
L=$SC/live
rm -rf "$L"
mkdir -p "$L/home" "$L/bin" "$L/wd" "$L/xdg/config" "$L/xdg/data" "$L/xdg/state" "$L/xdg/cache" "$L/xdg/run"
for d in /usr/bin /bin; do
  for f in "$d"/*; do
    n=${f##*/}
    [ "$n" = osascript ] && continue
    [ -e "$L/bin/$n" ] || ln -s "$f" "$L/bin/$n"
  done
done
cd "$L/wd" || exit 1
HOLDERS=()
cleanup() {
  for p in "${HOLDERS[@]}"; do kill "$p" 2>/dev/null; done
  pkill -9 -f "$B org watch" 2>/dev/null
}
trap cleanup EXIT

r() {
  env -i HOME="$L/home" PATH="$L/bin" SHELL=/bin/bash \
    XDG_CONFIG_HOME="$L/xdg/config" XDG_DATA_HOME="$L/xdg/data" XDG_STATE_HOME="$L/xdg/state" \
    XDG_CACHE_HOME="$L/xdg/cache" XDG_RUNTIME_DIR="$L/xdg/run" "$B" "$@"
}
now() { perl -MTime::HiRes=time -e 'printf "%.2f", time'; }
# x <args>: print the command, run it (stdout+stderr), print rc and elapsed.
x() {
  printf '\n$ ralph'; printf ' %q' "$@"; printf '\n'
  local t0 t1 rc
  t0=$(now); r "$@" 2>&1; rc=$?; t1=$(now)
  printf '[rc=%d elapsed=%.1fs]\n' "$rc" "$(echo "$t1 - $t0" | bc)"
}
h() { printf '\n## %s\n' "$*"; }
watchers() { pgrep -f "$B org watch --all --state-dir $1" | tr '\n' ' '; echo; }
holder() { # holder <dir> <state> <age>: start holder.py, wait for "holding"
  local out=$SC/holder-$$-$RANDOM.out
  python3 "$SC/holder.py" "$1" "$2" "$3" > "$out" 2>&1 &
  HOLDERS+=($!)
  until grep -q holding "$out" 2>/dev/null; do sleep 0.05; done
  echo "(holder pid $! holds $1/watch.lock, mark: $2, age ${3}s)"
}
SPAWN=(org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd "$L/wd" --scope test-scope)
TASK=$'TYPE: TASK\nTASK_ID: t-1\n\ndo it'
QUESTION=$'TYPE: QUESTION\n\nwhich plan?'

echo "binary: $B (built from $(git -C "$SC/../../../../.." rev-parse --short HEAD 2>/dev/null || echo ae349cc4))"
echo "date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

h "A. no watch: spawn and TASK refused with nothing written, QUESTION allowed"
A=$L/A
x status --state-dir "$A"
x "${SPAWN[@]}" --state-dir "$A" --dry-run
echo "\$ ls -A A"; ls -A "$A" 2>&1
x org send --org-id org-live --to seat-1 --text "$TASK" --dry-run --state-dir "$A"
x org send --org-id org-live --to seat-1 --raw --text "$TASK" --dry-run --state-dir "$A"
echo "\$ ls -A A"; ls -A "$A" 2>&1
x org send --org-id org-live --to seat-1 --text "$QUESTION" --dry-run --state-dir "$A"
echo "\$ jq -c '{event,org_id,seat_id,dry_run}' A/manifest.jsonl"; jq -c '{event,org_id,seat_id,dry_run}' "$A/manifest.jsonl"

h "A. watch --all --detach starts one, again says already running"
x org watch --all --detach --state-dir "$A"
x org watch --all --detach --state-dir "$A"
echo "\$ cat A/watch-alive.json"; cat "$A/watch-alive.json"; echo
echo "\$ pgrep watch processes on A"; watchers "$A"
x status --state-dir "$A"
x "${SPAWN[@]}" --state-dir "$A" --dry-run
x org send --org-id org-live --to seat-1 --text "$TASK" --dry-run --state-dir "$A"

h "A. kill -9 the watch: spawn refused (not_running), then start launches a watch first"
W1=$(jq -r .pid "$A/watch-alive.json")
echo "\$ kill -9 $W1"; kill -9 "$W1"; while kill -0 "$W1" 2>/dev/null; do sleep 0.05; done; echo "(pid $W1 gone)"
r status --state-dir "$A" 2>&1 | grep '^org watch:'
x "${SPAWN[@]}" --state-dir "$A" --dry-run
x org start --org-id org-live --driver claude --model sonnet --cwd "$L/wd" --scope org-wide --state-dir "$A" "do the task"
echo "\$ jq -c '{event,org_id,seat_id,dry_run}' A/manifest.jsonl"; jq -c '{event,org_id,seat_id,dry_run}' "$A/manifest.jsonl"
echo "\$ pgrep watch processes on A"; watchers "$A"
r status --state-dir "$A" 2>&1 | grep '^org watch:'
sleep 3
echo "(3 s later)"
echo "\$ pgrep watch processes on A"; watchers "$A"
r status --state-dir "$A" 2>&1 | grep '^org watch:'
echo "\$ cat A/watch.log"; cat "$A/watch.log"
x org watch --stop --state-dir "$A"

h "B. stale: a holder keeps watch.lock with a mark one hour old"
BB=$L/B
holder "$BB" running 3600
r status --state-dir "$BB" 2>&1 | grep '^org watch:'
x "${SPAWN[@]}" --state-dir "$BB" --dry-run
x org send --org-id org-live --to seat-1 --text "$TASK" --dry-run --state-dir "$BB"
x org watch --all --detach --state-dir "$BB"
x org start --org-id org-live --driver claude --model sonnet --cwd "$L/wd" --scope org-wide --state-dir "$BB" "do the task"
echo "\$ ls -A B"; ls -A "$BB"
echo "\$ pgrep watch processes on B"; watchers "$BB"

h "B2. stale, and the holder goes away during the 10 s wait: --detach launches"
B2=$L/B2
holder "$B2" running 3600
HP=${HOLDERS[${#HOLDERS[@]}-1]}
( sleep 3; kill "$HP"; echo "(holder pid $HP killed at +3 s)" ) &
KP=$!
x org watch --all --detach --state-dir "$B2"
wait "$KP"
echo "\$ pgrep watch processes on B2"; watchers "$B2"
r status --state-dir "$B2" 2>&1 | grep '^org watch:'
x org watch --stop --state-dir "$B2"

h "B3. stale, and the holder goes away during the 10 s wait: start launches"
B3=$L/B3
holder "$B3" none 0
HP=${HOLDERS[${#HOLDERS[@]}-1]}
( sleep 3; kill "$HP"; echo "(holder pid $HP killed at +3 s)" ) &
KP=$!
x org start --org-id org-live --driver claude --model sonnet --cwd "$L/wd" --scope org-wide --state-dir "$B3" "do the task"
wait "$KP"
echo "\$ pgrep watch processes on B3"; watchers "$B3"
r status --state-dir "$B3" 2>&1 | grep '^org watch:'
x org watch --stop --state-dir "$B3"

h "C. a --once watch is running (its cycle reads a large inbox of one resolved item)"
C=$L/C
mkdir -p "$C"
printf '%s\n' '{"ts":"2026-10-12T04:00:00Z","id":"e1","event":"escalated","org_id":"org-live","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich plan?"}' > "$C/inbox.jsonl"
perl -e 'print qq({"ts":"2026-10-12T04:00:01Z","id":"e1","event":"resolved","note":"done"}\n) x 4000000' >> "$C/inbox.jsonl"
echo "(C/inbox.jsonl: $(wc -l < "$C/inbox.jsonl" | tr -d ' ') lines: e1 escalated, then resolved again and again; no manifest)"
echo "\$ ralph org watch --all --once --state-dir C &"
t0=$(now)
( r org watch --all --once --state-dir "$C" > "$L/once.out" 2>&1; echo "rc=$?" >> "$L/once.out" ) &
OP=$!
for _ in $(seq 1500); do [ -f "$C/watch-alive.json" ] && jq -e '.state == "running" and .once == true' "$C/watch-alive.json" >/dev/null 2>&1 && break; sleep 0.02; done
echo "\$ cat C/watch-alive.json"; cat "$C/watch-alive.json"; echo
x "${SPAWN[@]}" --state-dir "$C" --dry-run
x org send --org-id org-live --to seat-1 --text "$TASK" --dry-run --state-dir "$C"
x org watch --all --detach --state-dir "$C"
x org start --org-id org-live --driver claude --model sonnet --cwd "$L/wd" --scope org-wide --state-dir "$C" "do the task"
echo "\$ ralph status --state-dir C | grep '^org watch:'"; r status --state-dir "$C" 2>&1 | grep '^org watch:'
echo "\$ ralph status --json --state-dir C | jq -c .org_watch"; r status --json --state-dir "$C" 2>/dev/null | jq -c .org_watch
wait "$OP"
t1=$(now)
printf '(the --once watch ended %.1fs after it started)\n' "$(echo "$t1 - $t0" | bc)"
echo "\$ cat once.out"; cat "$L/once.out"
r status --state-dir "$C" 2>&1 | grep '^org watch:'
rm -f "$C/inbox.jsonl"

h "D. stopping: a holder keeps watch.lock with a fresh stopping mark (hand-placed, not a real drain)"
D=$L/D
holder "$D" stopping 0
r status --state-dir "$D" 2>&1 | grep '^org watch:'
echo "\$ ralph status --json --state-dir D | jq -c .org_watch"; r status --json --state-dir "$D" 2>/dev/null | jq -c .org_watch
x "${SPAWN[@]}" --state-dir "$D" --dry-run
x org send --org-id org-live --to seat-1 --text "$TASK" --dry-run --state-dir "$D"
x org watch --all --detach --state-dir "$D"
x org start --org-id org-live --driver claude --model sonnet --cwd "$L/wd" --scope org-wide --state-dir "$D" "do the task"
echo "\$ ls -A D"; ls -A "$D"

h "E. 8 concurrent --detach on a new ledger, 3 rounds"
for round in 1 2 3; do
  E=$L/E$round
  echo
  echo "### round $round: 8 x ralph org watch --all --detach --state-dir E$round"
  pids=()
  for i in 1 2 3 4 5 6 7 8; do
    ( r org watch --all --detach --state-dir "$E" > "$L/e$round-$i.out" 2>&1; echo "rc=$?" >> "$L/e$round-$i.out" ) &
    pids+=($!)
  done
  wait "${pids[@]}"
  for i in 1 2 3 4 5 6 7 8; do printf '%d: %s\n' "$i" "$(tr '\n' ' ' < "$L/e$round-$i.out")"; done
  echo "\$ pgrep watch processes on E$round"; watchers "$E"
  r status --state-dir "$E" 2>&1 | grep '^org watch:'
  x org watch --stop --state-dir "$E"
done

h "F. end: no ralph org watch process of this binary remains"
for d in "$A" "$BB" "$B2" "$B3" "$C" "$D" "$L/E1" "$L/E2" "$L/E3"; do
  r org watch --stop --state-dir "$d" 2>&1 | grep -v 'is not running on' || true
done
for p in "${HOLDERS[@]}"; do kill "$p" 2>/dev/null; done
sleep 0.5
echo "\$ pgrep -fl '$B org watch'"; pgrep -fl "$B org watch"; echo "[pgrep rc=$?]"
```

### live.sh の出力(2026-10-12 14:50〜14:53 JST)

```text
binary: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/ralph (built from ae349cc4)
date: 2026-10-12T05:50:22Z

## A. no watch: spawn and TASK refused with nothing written, QUESTION allowed

$ ralph status --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
no org runtime state found (state-dir=/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A (source: flag)) — run `ralph org spawn` to start one.
org watch: not running
Run `ralph doctor` to check environment readiness.
[rc=0 elapsed=0.0s]

$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A --dry-run
rejected: org: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
org: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
[rc=1 elapsed=0.1s]
$ ls -A A
ls: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A: No such file or directory

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
org: send: TASK not sent: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
[rc=1 elapsed=0.0s]

$ ralph org send --org-id org-live --to seat-1 --raw --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
org: send: TASK not sent: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
[rc=1 elapsed=0.0s]
$ ls -A A
ls: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A: No such file or directory

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: QUESTION\n\nwhich plan?' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
sent message to seat "seat-1"
[rc=0 elapsed=0.0s]
$ jq -c '{event,org_id,seat_id,dry_run}' A/manifest.jsonl
{"event":"sent","org_id":"org-live","seat_id":"seat-1","dry_run":true}

## A. watch --all --detach starts one, again says already running

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
started ralph org watch --all (pid 45166); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A/watch.log
[rc=0 elapsed=0.2s]

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
ralph org watch --all is already running (pid 45166); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A/watch.log
[rc=0 elapsed=0.0s]
$ cat A/watch-alive.json
{
  "state": "running",
  "pid": 45166,
  "started_at": "2026-10-12T05:50:23.187463Z",
  "ts": "2026-10-12T05:50:23Z",
  "interval_seconds": 30,
  "deadman_minutes": 10
}

$ pgrep watch processes on A
45166 

$ ralph status --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
no org runtime state found (state-dir=/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A (source: flag)) — run `ralph org spawn` to start one.
org watch: running (pid 45166, mark updated 2026-10-12T05:50:23Z)
Run `ralph doctor` to check environment readiness.
[rc=0 elapsed=0.0s]

$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A --dry-run
spawned seat "seat-1" (org_id=org-live driver=claude model=sonnet pane_id= dry_run=true)
[rc=0 elapsed=0.1s]

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
sent message to seat "seat-1"
[rc=0 elapsed=0.0s]

## A. kill -9 the watch: spawn refused (not_running), then start launches a watch first
$ kill -9 45166
(pid 45166 gone)
org watch: not running (a running mark is left from pid 45166, mark updated 2026-10-12T05:50:23Z)

$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A --dry-run
rejected: org: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
org: no ralph org watch --all is running on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A, so no new work starts; start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A', or, for a new org, start it with ralph org start, which starts one
[rc=1 elapsed=0.1s]

$ ralph org start --org-id org-live --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope org-wide --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A do\ the\ task
ralph org start: started the cross-org watch (pid 45256); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A/watch.log
spawn failed: org: spawn step workspace_create failed: herdr: exec: "herdr": executable file not found in $PATH
org: spawn step workspace_create failed: herdr: exec: "herdr": executable file not found in $PATH
[rc=1 elapsed=0.2s]
$ jq -c '{event,org_id,seat_id,dry_run}' A/manifest.jsonl
{"event":"sent","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_started","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_step","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_step","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_step","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_step","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_step","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawned","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"sent","org_id":"org-live","seat_id":"seat-1","dry_run":true}
{"event":"spawn_started","org_id":"org-live","seat_id":"leader","dry_run":null}
{"event":"spawn_failed","org_id":"org-live","seat_id":"leader","dry_run":null}
$ pgrep watch processes on A
45256 
org watch: running (pid 45256, mark updated 2026-10-12T05:50:23Z)
(3 s later)
$ pgrep watch processes on A
45256 
org watch: running (pid 45256, mark updated 2026-10-12T05:50:23Z)
$ cat A/watch.log
watching every org (interval=30s once=false state-dir=/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A (source: flag))
watching every org (interval=30s once=false state-dir=/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A (source: flag))

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/A
stopped ralph org watch --all (pid 45256)
[rc=0 elapsed=0.8s]

## B. stale: a holder keeps watch.lock with a mark one hour old
(holder pid 45572 holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch.lock, mark: running, age 3600s)
org watch: stale (pid 45572, mark updated 2026-10-12T04:50:27Z) -- a watch holds watch.lock, but its liveness mark is missing, stopped, or older than the larger of deadman_minutes (10 when below 1) and two of its intervals

$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B --dry-run
rejected: org: the ralph org watch --all that holds the lock of the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B has not refreshed its liveness mark (pid 45572), so no new work starts; stop it with ralph org watch --stop --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B', or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
org: the ralph org watch --all that holds the lock of the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B has not refreshed its liveness mark (pid 45572), so no new work starts; stop it with ralph org watch --stop --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B', or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=2.1s]

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B
org: send: TASK not sent: the ralph org watch --all that holds the lock of the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B has not refreshed its liveness mark (pid 45572), so no new work starts; stop it with ralph org watch --stop --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B', or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=2.0s]

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B
org: watch --detach: a watch holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch.lock but its liveness mark is not fresh (pid 45572), so no new watch runs; stop it with ralph org watch --stop, or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=10.0s]

$ ralph org start --org-id org-live --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope org-wide --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B do\ the\ task
rejected: org: the ralph org watch --all that holds the lock of the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B has not refreshed its liveness mark (pid 45572), so no new work starts; stop it with ralph org watch --stop --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B', or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
org: the ralph org watch --all that holds the lock of the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B has not refreshed its liveness mark (pid 45572), so no new work starts; stop it with ralph org watch --stop --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B', or kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=10.1s]
$ ls -A B
watch-alive.json
watch.lock
$ pgrep watch processes on B


## B2. stale, and the holder goes away during the 10 s wait: --detach launches
(holder pid 49341 holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B2/watch.lock, mark: running, age 3600s)

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B2
(holder pid 49341 killed at +3 s)
started ralph org watch --all (pid 49475); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B2/watch.log
[rc=0 elapsed=3.2s]
$ pgrep watch processes on B2
49475 
org watch: running (pid 49475, mark updated 2026-10-12T05:50:55Z)

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B2
stopped ralph org watch --all (pid 49475)
[rc=0 elapsed=0.8s]

## B3. stale, and the holder goes away during the 10 s wait: start launches
(holder pid 49560 holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B3/watch.lock, mark: none, age 0s)

$ ralph org start --org-id org-live --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope org-wide --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B3 do\ the\ task
(holder pid 49560 killed at +3 s)
ralph org start: started the cross-org watch (pid 49710); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B3/watch.log
spawn failed: org: spawn step workspace_create failed: herdr: exec: "herdr": executable file not found in $PATH
org: spawn step workspace_create failed: herdr: exec: "herdr": executable file not found in $PATH
[rc=1 elapsed=3.2s]
$ pgrep watch processes on B3
49710 
org watch: running (pid 49710, mark updated 2026-10-12T05:50:59Z)

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B3
stopped ralph org watch --all (pid 49710)
[rc=0 elapsed=0.9s]

## C. a --once watch is running (its cycle reads a large inbox of one resolved item)
(C/inbox.jsonl: 4000001 lines: e1 escalated, then resolved again and again; no manifest)
$ ralph org watch --all --once --state-dir C &
$ cat C/watch-alive.json
{
  "state": "running",
  "pid": 49827,
  "started_at": "2026-10-12T05:51:01.686081Z",
  "ts": "2026-10-12T05:51:01Z",
  "interval_seconds": 30,
  "deadman_minutes": 10,
  "once": true
}


$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C --dry-run
rejected: org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (pid 49827) is a --once run, which ends after its cycle and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C')
org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (pid 49827) is a --once run, which ends after its cycle and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C')
[rc=1 elapsed=0.0s]

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C
org: send: TASK not sent: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (pid 49827) is a --once run, which ends after its cycle and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C')
[rc=1 elapsed=0.0s]

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C
org: watch --detach: the watch that holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C/watch.lock (pid 49827) is a --once run, which ends after its cycle, so no new watch runs; run ralph org watch --all --detach again once it has ended
[rc=1 elapsed=0.0s]

$ ralph org start --org-id org-live --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope org-wide --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C do\ the\ task
rejected: org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (pid 49827) is a --once run, which ends after its cycle and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C')
org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (pid 49827) is a --once run, which ends after its cycle and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C')
[rc=1 elapsed=0.1s]
$ ralph status --state-dir C | grep '^org watch:'
org watch: running (--once) (pid 49827, mark updated 2026-10-12T05:51:01Z) -- it ends after its cycle and accepts no new work
$ ralph status --json --state-dir C | jq -c .org_watch
{"state":"running","pid":49827,"started_at":"2026-10-12T05:51:01.686081Z","updated_at":"2026-10-12T05:51:01Z","mark_state":"running","once":true}
(the --once watch ended 5.7s after it started)
$ cat once.out
watching every org (interval=30s once=true state-dir=/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/C (source: flag))
rc=0
org watch: not running (stopped at 2026-10-12T05:51:07Z)

## D. stopping: a holder keeps watch.lock with a fresh stopping mark (hand-placed, not a real drain)
(holder pid 50289 holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D/watch.lock, mark: stopping, age 0s)
org watch: stopping (pid 50289, mark updated 2026-10-12T05:51:07Z) -- it has run its last cycle and accepts no new work; it releases watch.lock once its in-flight work ends
$ ralph status --json --state-dir D | jq -c .org_watch
{"state":"stopping","pid":50289,"started_at":"2026-10-12T05:51:07Z","updated_at":"2026-10-12T05:51:07Z","mark_state":"stopping"}

$ ralph org spawn --org-id org-live --id seat-1 --role worker --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope test-scope --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D --dry-run
rejected: org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D (pid 50289) is stopping and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D')
org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D (pid 50289) is stopping and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D')
[rc=1 elapsed=0.1s]

$ ralph org send --org-id org-live --to seat-1 --text $'TYPE: TASK\nTASK_ID: t-1\n\ndo it' --dry-run --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D
org: send: TASK not sent: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D (pid 50289) is stopping and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D')
[rc=1 elapsed=0.0s]

$ ralph org watch --all --detach --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D
org: watch --detach: the watch that holds /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D/watch.lock (pid 50289) is stopping, so no new watch runs; run ralph org watch --all --detach again once it has ended
[rc=1 elapsed=0.0s]

$ ralph org start --org-id org-live --driver claude --model sonnet --cwd /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/wd --scope org-wide --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D do\ the\ task
rejected: org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D (pid 50289) is stopping and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D')
org: the ralph org watch --all on the org state dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D (pid 50289) is stopping and accepts no new work; run this again once it has ended (ralph org start then starts a long-running watch itself; for a spawn or a TASK send, start one with ralph org watch --all --detach --state-dir '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D')
[rc=1 elapsed=0.1s]
$ ls -A D
watch-alive.json
watch.lock

## E. 8 concurrent --detach on a new ledger, 3 rounds

### round 1: 8 x ralph org watch --all --detach --state-dir E1
1: started ralph org watch --all (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
2: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
3: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
4: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
5: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
6: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
7: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
8: ralph org watch --all is already running (pid 50394); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch.log rc=0 
$ pgrep watch processes on E1
50394 50396 50398 50399 50400 50401 
org watch: running (pid 50394, mark updated 2026-10-12T05:51:07Z)

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1
org: watch --stop: the watch did not stop within 30s; the stop request stays in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 50398 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E1/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=30.1s]

### round 2: 8 x ralph org watch --all --detach --state-dir E2
1: started ralph org watch --all (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
2: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
3: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
4: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
5: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
6: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
7: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
8: ralph org watch --all is already running (pid 55062); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch.log rc=0 
$ pgrep watch processes on E2
55062 55064 55068 55069 55070 55071 
org watch: running (pid 55062, mark updated 2026-10-12T05:51:38Z)

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2
org: watch --stop: the watch did not stop within 30s; the stop request stays in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 55071 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E2/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=30.0s]

### round 3: 8 x ralph org watch --all --detach --state-dir E3
1: started ralph org watch --all (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
2: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
3: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
4: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
5: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
6: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
7: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
8: ralph org watch --all is already running (pid 59530); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch.log rc=0 
$ pgrep watch processes on E3
59530 59534 59538 59544 
org watch: running (pid 59530, mark updated 2026-10-12T05:52:08Z)

$ ralph org watch --stop --state-dir /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3
org: watch --stop: the watch did not stop within 30s; the stop request stays in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 59544 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/E3/watch-alive.json; check that it is still a ralph org watch first)
[rc=1 elapsed=30.0s]

## F. end: no ralph org watch process of this binary remains
org: watch --stop: the watch did not stop within 30s; the stop request stays in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 45572 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/B/watch-alive.json; check that it is still a ralph org watch first)
org: watch --stop: the watch did not stop within 30s; the stop request stays in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 50289 (the pid in /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/live/D/watch-alive.json; check that it is still a ralph org watch first)
stopped ralph org watch --all (pid 50398)
stopped ralph org watch --all (pid 55071)
stopped ralph org watch --all (pid 59544)
$ pgrep -fl '/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/ralph org watch'
[pgrep rc=1]
```

### holder.py

```python
#!/usr/bin/env python3
"""Hold <state dir>/watch.lock (flock, as ralph's watch does) and write a
liveness mark, until SIGTERM. Usage: holder.py <state dir> <state> <age seconds>
  state: running | stopping | none (no mark)
"""
import datetime
import fcntl
import json
import os
import signal
import sys
import time

d, state, age = sys.argv[1], sys.argv[2], int(sys.argv[3])
os.makedirs(d, exist_ok=True)
f = open(os.path.join(d, "watch.lock"), "a+")
fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
if state != "none":
    ts = (datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(seconds=age)).replace(microsecond=0)
    mark = {"state": state, "pid": os.getpid(), "started_at": ts.strftime("%Y-%m-%dT%H:%M:%SZ"),
            "ts": ts.strftime("%Y-%m-%dT%H:%M:%SZ"), "interval_seconds": 30, "deadman_minutes": 10}
    tmp = os.path.join(d, "watch-alive.json.tmp")
    with open(tmp, "w") as m:
        json.dump(mark, m)
    os.rename(tmp, os.path.join(d, "watch-alive.json"))
signal.signal(signal.SIGTERM, lambda *a: sys.exit(0))
print("holding", flush=True)
while True:
    time.sleep(1)
```

### takeover.sh

```bash
#!/bin/bash
# Does a --stop right after concurrent --detach (or with a watch waiting for
# watch.lock) end every watch? Run with the base (dc41c4f5) and HEAD
# (ae349cc4) binaries, same isolation as live.sh.
set -u
SC=$(cd "$(dirname "$0")" && pwd)
L=$SC/takeover
rm -rf "$L"
mkdir -p "$L/home" "$L/bin" "$L/wd"
for d in /usr/bin /bin; do
  for f in "$d"/*; do n=${f##*/}; [ "$n" = osascript ] && continue; [ -e "$L/bin/$n" ] || ln -s "$f" "$L/bin/$n"; done
done
cd "$L/wd" || exit 1
trap 'pkill -9 -f "$SC/ralph(-base)? org watch" 2>/dev/null' EXIT
now() { perl -MTime::HiRes=time -e 'printf "%.2f", time'; }
for name in base head; do
  B=$SC/ralph; [ $name = base ] && B=$SC/ralph-base
  r() { env -i HOME="$L/home" PATH="$L/bin" SHELL=/bin/bash "$B" "$@"; }
  echo
  echo "### $name: a foreground watch --all waits for the lock, then --stop"
  X=$L/$name-wait
  r org watch --all --detach --state-dir "$X"; echo "[detach rc=$?]"
  ( r org watch --all --state-dir "$X" > "$L/$name-waiter.out" 2>&1; echo "[waiter rc=$?]" >> "$L/$name-waiter.out" ) &
  sleep 0.3
  t0=$(now); r org watch --stop --state-dir "$X" 2>&1 | sed "s#$L#<L>#g"; echo "[stop rc=${PIPESTATUS[0]} elapsed=$(echo "$(now) - $t0" | bc)s]"
  echo "watch processes after the stop: $(pgrep -f "$B org watch --all --state-dir $X" | wc -l | tr -d ' ')"
  r org watch --stop --state-dir "$X" 2>&1 | sed "s#$L#<L>#g"
  sleep 0.5; sed "s#$L#<L>#g" "$L/$name-waiter.out"

  echo
  echo "### $name: 8 x --detach at once, then --stop at once"
  X=$L/$name-eight
  pids=()
  for i in 1 2 3 4 5 6 7 8; do ( r org watch --all --detach --state-dir "$X" > "$L/$name-e$i.out" 2>&1; echo "rc=$?" >> "$L/$name-e$i.out" ) & pids+=($!); done
  wait "${pids[@]}"
  echo "detach rcs: $(cat "$L"/$name-e*.out | grep -c '^rc=0') x 0, $(cat "$L"/$name-e*.out | grep -c '^rc=1') x 1"
  echo "watch processes when every --detach has returned: $(pgrep -f "$B org watch --all --state-dir $X" | wc -l | tr -d ' ')"
  t0=$(now); r org watch --stop --state-dir "$X" 2>&1 | sed "s#$L#<L>#g"; echo "[stop rc=${PIPESTATUS[0]} elapsed=$(echo "$(now) - $t0" | bc)s]"
  echo "watch processes after the stop: $(pgrep -f "$B org watch --all --state-dir $X" | wc -l | tr -d ' ')"
  echo "banners in watch.log: $(grep -c '^watching every org' "$X/watch.log")"
  r org watch --stop --state-dir "$X" 2>&1 | sed "s#$L#<L>#g"
done
```

### takeover.sh の出力(2026-10-12 14:38〜14:41 JST)

```text

### base: a foreground watch --all waits for the lock, then --stop
started ralph org watch --all (pid 84244); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/takeover/base-wait/watch.log
[detach rc=0]
org: watch --stop: the watch did not stop within 30s; the stop request stays in <L>/base-wait/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 84247 (the pid in <L>/base-wait/watch-alive.json; check that it is still a ralph org watch first)
[stop rc=1 elapsed=30.04s]
watch processes after the stop: 1
stopped ralph org watch --all (pid 84247)
watching every org (interval=30s once=false state-dir=<L>/base-wait (source: flag))
[waiter rc=0]

### base: 8 x --detach at once, then --stop at once
detach rcs: 7 x 0, 1 x 1
watch processes when every --detach has returned: 1
stopped ralph org watch --all (pid 86068)
[stop rc=0 elapsed=.97s]
watch processes after the stop: 0
banners in watch.log: 2
ralph org watch --all is not running on <L>/base-eight

### head: a foreground watch --all waits for the lock, then --stop
started ralph org watch --all (pid 86257); log: /private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/owg-test/takeover/head-wait/watch.log
[detach rc=0]
org: watch --stop: the watch did not stop within 30s; the stop request stays in <L>/head-wait/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 86260 (the pid in <L>/head-wait/watch-alive.json; check that it is still a ralph org watch first)
[stop rc=1 elapsed=30.05s]
watch processes after the stop: 1
stopped ralph org watch --all (pid 86260)
watching every org (interval=30s once=false state-dir=<L>/head-wait (source: flag))
[waiter rc=0]

### head: 8 x --detach at once, then --stop at once
detach rcs: 8 x 0, 0 x 1
watch processes when every --detach has returned: 6
org: watch --stop: the watch did not stop within 30s; the stop request stays in <L>/head-eight/watch-stop.json, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, kill 88241 (the pid in <L>/head-eight/watch-alive.json; check that it is still a ralph org watch first)
[stop rc=1 elapsed=30.06s]
watch processes after the stop: 1
banners in watch.log: 2
stopped ralph org watch --all (pid 88241)
rc=0
```
