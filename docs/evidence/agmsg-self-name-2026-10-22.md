# agmsg の自己命名と座席の環境の実機確認

- Date: 2026-10-22(JST 06:30 ごろ〜06:45、UTC 2026-10-21 21:30〜21:45)
- Plan: `docs/plans/active/2026-10-22-seat-agmsg-self-name.md`(S1、AC6)
- 生の出力はセッションの scratchpad に置き、リポジトリにはコミットしていない。要点はこのファイルに写した。ホームディレクトリは `~`、分離に使った一時ディレクトリは `<D>` に置き換えた。pane の id、session の id、agmsg が付けた名前の値は写していない。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0)、12 CPU。(c) を測った間の load average は 8 前後 |
| herdr | 0.7.5。`herdr server`(ヘッドレス)を分離して起動した |
| agmsg | 1.5.1(`~/.agents/skills/agmsg` の写し) |
| claude | Claude Code 2.1.291、`--model haiku`(Haiku 4.5)、`--allowedTools Bash`。(e) だけで使った |
| ralph | 使っていない。agmsg の script を直接打った |

### 分離のしかた

- herdr server は `/tmp` の下の短い一時ディレクトリ `<D>` で起動した。unix socket の path には長さの上限があるため、scratchpad ではなく `/tmp` を使った。`env -i` で環境を空にしてから、`HOME`(`<D>/home` の偽の HOME)、`PATH`、`XDG_CONFIG_HOME` と `XDG_STATE_HOME`(どちらも `<D>` の下)、`HERDR_SOCKET_PATH`(`<D>/herdr.sock`)だけを渡した。この server に打つ herdr のコマンドも同じ環境で打った。
- agmsg は `scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` だけを `<D>` の下に写した。ユーザーの `db`・`teams`・`run`・`agents` は写していないので、DB と team は写しの下に新しく作られる。このシェルから打つ agmsg は、`env -i` に `HOME`・`PATH`・`HERDR_SOCKET_PATH` と `AGMSG_SELF_NAME=off` を付けて打った。このセッションの `HERDR_PANE_ID` と Orca の変数は渡していない。
- (a)〜(c) と (e) は 1 つ目の server で打った。(d) は 2 つ目の server(新しい `<D>` と、agmsg の新しい写し 2 つ)で打った。(d) は最初、(b)・(c) と同じ DB の新しい team で打った。AC6 は条件ごとに DB を作り直すよう求めているので、作り直して打ち直した。下の (d) は打ち直した結果を書く。
- (e) の claude は、認証のために本当の HOME で起動した。pane を `--env HOME=~` と `--env PATH=<claude のある PATH>` 付きで作り、off の pane には `--env AGMSG_SELF_NAME=off` も足した。server の HOME は偽のままにした。

## 結果

### (a) `tab create --env` は pane の環境に値を足す

`herdr tab create --env FOO=bar --env AGMSG_SELF_NAME=off` で作った pane と、`--env` なしで作った pane で、`env` の出力を比べた。変数の名前の差は `AGMSG_SELF_NAME` と `FOO` の 2 つだけだった。`HOME`・`PATH`・`SHELL`・`USER` と herdr の `HERDR_*` は、どちらの pane でも同じ値だった。pane の shell から起動した子のプロセス(`bash -c`)も、`FOO=bar` と `AGMSG_SELF_NAME=off` を読めた。`--env` は環境を置き換えずに足すので、計画の D2 を見直す必要はない。

### (b) pane の中の agmsg は、off なら herdr を打たない

agent のいない pane の shell で `inbox.sh` を打った。

- off の pane では、出力は `No new messages.` だけで、herdr の呼び出しはなかった。
- `--env` なしの on の pane では、agmsg が `herdr agent rename` を打ち、`agent_not_found` で失敗した。pane に agent がいないためである。続けて `agmsg: could not name this herdr pane ... (runtime_error)` が出たが、`inbox.sh` は rc 0 で終わった。
- どちらの pane でも、打つ前と後で `herdr pane get` の記録は変わらなかった。

### (c) send.sh の時間

`send.sh` で ALERT と同じ形のメッセージを送り、時間を測った。1 本ずつ打つ条件は 20 本、6 本を同時に打つ条件(座席の数を想定)は 6 本ずつ 4 回で計 24 本を送った。失敗は 0 本だった。表の値は、どの列も send.sh 1 本ごとの壁時計の時間(ミリ秒)で、条件ごとに 20 本か 24 本の中央値(最小〜最大)を書く。

| 環境 | on、1 本ずつ | off、1 本ずつ | on、6 本同時 | off、6 本同時 |
|---|---|---|---|---|
| 見張りと同じ環境(`HERDR_PANE_ID` なし) | 346(317〜803) | 281(262〜627) | 657(552〜1219) | 533(419〜576) |
| 座席の pane の環境(`HERDR_PANE_ID` あり) | 1278(1223〜2227) | 314(278〜649) | 2648(2301〜3073) | 698(598〜757) |

- 見張りの環境では、on と off の差は中央値で約 65 ms、6 本同時で約 120 ms だった。
- pane の環境では、on は off より 1 本ずつで約 1 s、6 本同時で約 2 s 遅かった。on の stderr には (b) と同じ `herdr agent rename` の失敗が出た。この pane には agent がいないので、rename は失敗で終わる。agent がいて rename が通るときの時間は測っていない。
- CPU に負荷をかけた計測はしていない。試行の間の load average は 20〜75 で、今回は 8 前後だった。

### (d) off で送ったメッセージは受け取る側に届く

条件ごとに agmsg の新しい写しを使い、DB と team を新しく作った。

- 見張りと同じ環境: `watchdog` と `leader` を join し、`watchdog` から `leader` へ ALERT と同じ形のメッセージを off で送った。`leader` の `inbox.sh`(off)が 1 件を読み、2 回目は `No new messages.` だった。
- pane の環境: `--env AGMSG_SELF_NAME=off` 付きの pane を leader 用と座席用に 1 つずつ作った。見張りと同じ環境から送った ALERT を、leader の pane の `inbox.sh` が読んだ。座席の pane から `send.sh` で leader へ送った RESULT の形のメッセージも、leader の pane が読んだ。どの呼び出しも rc 0 で終わった。

### (e) 本物の座席: off なら名前が残り、on なら変わる

pane で `herdr agent start <名前> --kind claude --pane <pane> -- --model haiku --allowedTools Bash` を打って座席を起動した。座席には、Bash tool で 1 つのコマンドを打つよう頼んだ。そのコマンドは `AGMSG_SELF_NAME` と `HERDR_PANE_ID` を書き出してから、自分の受信箱を `inbox.sh` で読む。どちらの座席の受信箱にも、leader からの TASK を 1 件入れておいた。

| pane | 座席から見た `AGMSG_SELF_NAME` | `inbox.sh` | 打ったあとの `herdr agent get <名前>` |
|---|---|---|---|
| `--env AGMSG_SELF_NAME=off` あり | `off` | TASK を 1 件読んだ | 名前が引けた。しばらくしてもう一度引いても引けた |
| `--env` なし(対照) | 未設定 | TASK を 1 件読んだ | `agent_not_found`。`herdr agent list` では、その pane の agent が英数字 25 文字の別の名前で出た |

対照では、試行の F3 と同じことが起きた。名前を変えたのは、おそらく (b) の on の pane で stderr に出た agmsg の `herdr agent rename` である。新しい名前は、tech-debt の F3 の行が書く「team と agent から作った hex の名前」の形だった。名前の作られ方は確かめていない。

## (f) ALERT の行は開いたままにする

- 見張りと同じ環境では、self-naming で増える時間は中央値で約 65 ms、6 本同時でも最大 1.2 s だった。見張りの ALERT は `callWithTimeout` の 10 秒で時間切れになっていたが、その原因が self-naming だとは考えにくい。今回の計測では時間切れは起きなかった。そのため tech-debt の「leader への ALERT」の行は閉じない。この数字と、ralph が打つ agmsg のすべての呼び出しに `AGMSG_SELF_NAME=off` を付けたこと(この PR の S2)を、その行に書き足す。
- 座席の pane の環境では、on は 1 本ずつで約 1 s、6 本同時で約 2 s 遅く、同時に打つほど遅くなった。ralph は座席の join と HELLO を leader のプロセスから打ち、leader は herdr の pane で動くので、`agmsg_announce` はこの環境で打たれる。試行の F4(`agmsg_announce` の時間切れ)に self-naming が効いていた可能性はあるが、再現はしていない。F4 は 4b で扱う。

## 触れていないもの

- ユーザーの herdr server には触れていない。計測のあとに `pgrep -fl 'herdr server'` で確かめると、動いている herdr server はなかった。`~/.config/herdr` の下には、計測を始める前の 2026-10-21 19:00(JST)より新しいファイルがない。
- Orca は打っていない。このセッションの Orca の変数は、分離した server にも agmsg にも渡していない。
- 計測のあと、claude を `/exit` で閉じ、2 つの server を `herdr server stop` で止めた。`<D>` を含むプロセスがないことと、cwd が `<D>` の下にある claude がないこと(`lsof -a -d cwd -c claude`)を確かめてから、`<D>` を消した。

### 計測で残ったもの

- (e) では claude のフォルダ信頼の確認に「信頼する」と答えたので、`~/.claude.json` に `<D>/work` を信頼済みとする項目が 1 つ増えた。`<D>` は消したが、この項目は残した。ほかの Claude Code のセッションも同じファイルに書くので、手では消していない。
- (e) の 2 つの claude の session の記録が、`~/.claude/projects/` の下に残っている。
