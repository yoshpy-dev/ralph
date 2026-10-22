# spawn の失敗の補償(pane を閉じる)の実機確認

- Date: 2026-10-22(JST 13:35〜13:37、UTC 04:35〜04:36)。その前の UTC 03:34〜03:36 に、S1 だけを入れた binary で同じ環境の下見をした
- Plan: `docs/plans/active/2026-10-22-seat-spawn-close.md`(S4、AC6)
- 生の出力はセッションの scratchpad に置き、リポジトリにはコミットしていない。要点はこのファイルに写した。ホームディレクトリは `~`、分離に使った一時ディレクトリは `<D>`、pane の id は `<pane>` に置き換えた。端末の id と session の id は写していない。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`herdr server`(ヘッドレス)を分離して起動した |
| agmsg | 1.5.1(`~/.agents/skills/agmsg` の写し) |
| ralph | このブランチの abb11e8c(S1〜S3 を含む)を `git archive` で worktree の外に取り出して build した |
| 座席の agent | 偽の `claude`。Go で書いた小さな binary で、入力欄に似た枠を出して stdin を読むだけのもの。`herdr agent start --kind claude` はこれを受け付け、`interactive_ready: true` を返した。本物の claude は起動していない |

### 分離のしかた

- herdr server は `/tmp` の下の短い一時ディレクトリ `<D>` で起動した。`env -i` で環境を空にしてから、`HOME`(`<D>/home` の偽の HOME。中の `.zshrc` は空)、`PATH`(先頭が `<D>/bin`)、`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH`(`<D>/herdr.sock`)だけを渡した。ralph と herdr のコマンドもすべて同じ環境で打った。Orca の変数とこのセッションの `HERDR_PANE_ID` は渡していない。
- agmsg は `scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` を `<D>` の下に写し、`scripts/send.sh` を stub に替えた。stub は本物の `send.sh` で送ってから、本文が `TYPE: HELLO` で始まり、かつ印のファイルがあるときだけ 40 秒眠る。HELLO は leader の受信箱に届いたうえで、ralph の `agmsg_announce` が時間切れになる。
- `<D>/bin` には次の 3 つを置いた。
  - 偽の `claude`(上の表)
  - `herdr` の包み。ふだんは本物の herdr に渡す。印のファイルがあるときだけ `herdr pane close` を断り、herdr と同じ形のエラー(code `internal_error`、rc 1)を返す。(e) で close の失敗を起こすのに使った
  - 記録するだけの `osascript`。見張りの desktop 通知が本物の osascript を打たないようにするため。呼び出しは 0 回だった
- 台帳と設定は `<D>/work` の scratch の git repo に置いた。`ralph.toml` は、model_pool が claude の haiku だけ、`[org.permissions].default = "guarded"`、watcher なし、`[org.director].enabled = false`、`deadman_minutes = 0`。見張りは新しい binary の `ralph org watch --all --detach` で起動した。

## 結果

org は `ssc`、座席は `seat-a`・`seat-b`・`seat-c`(role は implementer)。

### (a) HELLO の時間切れで pane が閉じ、agent の名前が空く

`ralph org spawn --org-id ssc --id seat-a ... --timeout-ms 15000` を打った。

- 04:35:30 に spawn が始まった。座席の join が戻ったのは 6 秒後の 04:35:36 で、同じ秒に stub が HELLO を送った(rc 0)。`history.sh` で、leader の受信箱にその HELLO が 1 件あることを確かめた。
- 04:35:46 に `spawn_failed` を記録して終わった(始まりから 16 秒)。Details は `step=agmsg_announce error=bash: timed out: context deadline exceeded compensation=ctrl_c=ok pane=closed leader_join=ok leave=ok`。
- エラーは `org: spawn step agmsg_announce failed: bash: timed out: context deadline exceeded (the spawn's --timeout-ms deadline of 15000 ms, shared by every herdr and agmsg step, ran out; retry with a larger --timeout-ms)` で、終了コードは 1。
- `herdr pane list` には org の workspace の最初の pane だけが残り、座席の pane はなかった。`herdr agent get ssc_seat-a` は `agent_not_found` を返した。

### (b) 同じ座席の次の spawn が通る

stub の印を外し、同じ `seat-a` を `--timeout-ms 30000` で打った。`agent_start` を通って 6 秒で `spawned` になり、`herdr agent get ssc_seat-a` が新しい pane の agent を返した。

### (c) status に 1 行だけ active で出る

`ralph status` の org `ssc` には、`seat-a  implementer  claude  haiku  spawned (active)  <pane>` の 1 行が出た(active 1/1)。

### (f) 同じ座席の spawn が重なると、あとのほうが断られる

HELLO の印を付けて `seat-c` の spawn を裏で打ち、5 秒後に同じ `seat-c` をもう 1 回打った。

- 2 つ目は `rejected: org: spawn: another ralph org spawn of seat "seat-c" in org_id "ssc" is running; retry after it returns: ...`(ロックファイルの path と `the lock is held after 0s` が続く)で、終了コード 1。
- 台帳の `seat-c` の記録は、1 つ目の spawn の 1 回分(`spawn_started` から `spawn_failed` まで 6 件)だけだった。2 つ目の `spawn_started` と `rejected` はなく、receipt も増えていない。
- 1 つ目は期限で失敗し、`spawn_failed` の Details は `compensation=ctrl_c=ok pane=closed`。`herdr agent get ssc_seat-c` は `agent_not_found` を返した。

### (e) 閉じられなかった座席は active のまま残り、あとで片付く

HELLO の印と、`pane close` を断る印の両方を付けて `seat-b` を打った。

- spawn は 04:36:24 に終わり、`spawn_failed` の代わりに `spawn_close_failed` を記録した。Details は `step=agmsg_announce error=... compensation=ctrl_c=ok pane=close failed: herdr pane close: herdr: exit status 1: ...(herdr: internal_error: ...) leader_join=ok leave=ok`。
- エラーは (a) と同じ期限の案内のあとに `; pane "<pane>" not closed, seat left active: herdr pane close: ...; close it with: herdr pane close <pane>, or ralph org stop --org-id ssc --seat seat-b` が続き、終了コードは 1。
- `ralph status` では `seat-b` が `spawn_step (active)` として、pane の id 付きで出た。ROLE・DRIVER・MODEL の欄は空だった。最後の state event が role を持たない `spawn_step` だからで、spawn の途中の座席はどれもこう見える。close が失敗したことは status には出ず、エラーと Details と `ralph org report` の timeline で分かる。
- `herdr agent get ssc_seat-b` は agent を返した(pane も agent も残っている)。

そのあと、次の 3 つを順に打った。

1. model_pool にない `--model opus` で打ち直した。`rejected: org: model "opus" not in [org].model_pool for driver "claude"; seat "seat-b" in org_id "ssc" has an unfinished spawn on pane "<pane>", left as it is: end it with ralph org stop --org-id ssc --seat seat-b` で、終了コード 1。台帳の行数(51)、receipt の数(0)、`rejected` の数(0)は打つ前と同じだった。
2. close を断る印を付けたまま、正しい model で打ち直した。ralph は記録の pane の label を確かめて C-c を送ったが、close が断られた。新しい spawn を断り、`spawn_close_failed`(Details `respawn refused (previous_event=spawn_step compensation=ctrl_c=ok pane=close failed: ...)`)を記録した。エラーには `herdr pane close <pane>` と `ralph org stop --org-id ssc --seat seat-b` が付いた。`spawn_started` は書かれていない。herdr の包みの記録には、断った close が 2 回(最初の spawn と、この打ち直し)残った。
3. 印を外して、もう 1 回正しく打ち直した。04:36:25 に前の途中の記録を `spawn_failed`(Details `superseded by respawn (previous_event=spawn_step compensation=ctrl_c=ok pane=closed)`)で閉じ、新しい spawn が 6 秒で `spawned` になった。

### (d) stop で片付く

`ralph org stop --org-id ssc --seat seat-a` と `ralph org stop --all`(`seat-b` を止めた)のあと、`ralph org disband --org-id ssc` を打った。どれも終了コード 0。`herdr agent list` と `herdr workspace list` は空になった。

## 確かめていないもの

- 本物の claude の座席。偽の `claude` で確かめたのは、herdr の pane と agent の名前と、ralph の台帳までである。
- 負荷の下での時間。HELLO の遅れは stub で作った。試行のときのような負荷はかけていない。
- stale の座席の pane の label が違う場合と、label の確かめ(herdr の get)が失敗する場合。どちらも fake の herdr を使うテストだけで確かめた(`internal/org/spawn_test.go` の stale のテスト)。
- 試行で古い binary が残した pane を、新しい binary が見つけて閉じること(計画の Non-goals)。

## 触れていないもの

- 始める前も終わったあとも、`herdr server` のプロセスは分離した 1 つだけだった。ユーザーの herdr server は動いていなかった。`~/.config/herdr` の下には、確かめを始めてから更新されたファイルがない。
- Orca は打っていない。本物の osascript も打っていない(記録するだけの stub の呼び出しも 0 回)。
- 本物の claude を起動していないので、`~/.claude.json` にフォルダ信頼の項目は増えていない。
- 終わったあと、見張りを `ralph org watch --stop` で、server を `herdr server stop` で止めた。`<D>` を含むプロセスがないことを確かめてから、`<D>` を消した。
