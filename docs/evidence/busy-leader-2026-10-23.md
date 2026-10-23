# 手番を終えて待つ leader の実機確認

- Date: 2026-10-23(JST 09:17〜09:38、UTC 00:17〜00:38)
- Plan: `docs/plans/active/2026-10-23-busy-leader.md`(S3、AC5)
- 生の出力は `<D>` の下に置き、要点をこのファイルに写してから `<D>` ごと消した。リポジトリにはコミットしていない。ホームディレクトリは `~`、分離に使った一時ディレクトリは `<D>` に置き換えた。pane の id、session の id、端末の id、Monitor の task の id は写していない。時刻はすべて UTC。

## 環境

| 項目 | 値 |
|---|---|
| マシン | macOS(Darwin 25.5.0) |
| herdr | 0.7.5。`herdr server`(ヘッドレス)を分離して起動した |
| claude | Claude Code 2.1.291、`--model haiku`(Haiku 4.5) |
| agmsg | 1.5.1(`~/.agents/skills/agmsg` の写し) |
| pane の shell | bash 3.2.57(`/bin/bash` の login shell) |
| git | 2.49.0 |
| ralph | このブランチの 3d1946d6(S1 と S2 を含む)を `git archive` で worktree の外に取り出し、go 1.26.0 で build した |
| 見張り | `ralph org start` が起動した `ralph org watch --all`(間隔 30 秒) |

### 分離のしかた

- herdr server は `/tmp` の下の一時ディレクトリ `<D>` で起動した。`env -i` で環境を空にしてから、`HOME`(本当の HOME。claude の認証のため)、`USER`、`LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM`、`TMPDIR`、`PATH`、`XDG_CONFIG_HOME` と `XDG_STATE_HOME`(どちらも `<D>` の下)、`HERDR_SOCKET_PATH`(`<D>/h.sock`)だけを渡した。`PATH` は `<D>/bin`、claude の実体のディレクトリ(`~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/node_modules/.bin`)、node の bin、`/opt/homebrew/bin`(herdr・git・jq)、`/usr/bin`、`/bin`、`/usr/sbin`、`/sbin` の順にした。ralph、herdr、agmsg のコマンドもすべて同じ環境で打った。このセッションの `HERDR_PANE_ID` と Orca の変数は渡していない。server のプロセスが持つ環境変数は、渡した 11 個だけだった。
- `SHELL=/bin/bash` にしたのは、ユーザーの zsh の設定が `claude` に model と permission の引数を足す alias を定義していて、`/etc/zshenv` が `ZDOTDIR` を戻すため。leader を起こす前に、捨ての workspace の pane で shell を確かめた。`$BASH_VERSION` は `3.2.57(1)-release`、`$$` のプロセスは `-bash`、`type claude` は `claude is ~/.local/share/mise/installs/npm-anthropic-ai-claude-code/latest/node_modules/.bin/claude` で、`alias` は何も出さなかった。herdr は pane の shell に `$SHELL` を使った。pane の環境に `CLAUDE_` と `ORCA_` で始まる変数はなかった。
- agmsg は `scripts`・`plugins`・`types`・`SKILL.md`・`VERSION` を `<D>/agmsg` に写した(`db`・`teams`・`run`・`agents` は写していない)。`ralph.toml` の `[org].agmsg_home` を `"<D>/agmsg"` にした。
- `<D>/bin/osascript` は、引数をログに足して 0 で終わるだけの shim にした。見張りの desktop 通知が本物の osascript を打たないようにするため。呼び出しは 1 回だった((e) を見よ)。
- 台帳と設定は `<D>/work` の scratch の git repo に置いた。base のコミットを main に置き、名前とメールは repo の設定で入れ、`core.hooksPath` を `<D>` の下の空のディレクトリにした。`.gitignore` には `.harness/state/` と `.harness/logs/` を書いた。`ralph.toml` の中身は次のとおり。
  - `driver_pool = ["claude"]`、model_pool は claude の haiku だけ、`deadman_minutes = 0`
  - `[org.permissions].default = "autonomous"`
  - `[org.watchdog]` の `watcher_enabled = false` と `stall_minutes = 240`(確かめのあいだに、heartbeat の停滞の ALERT が leader の pane に打たれないようにするため)
  - `[org.director].enabled = false`
- フォルダ信頼: leader の前に、捨ての pane で cwd を `<D>/work` にして `claude --model haiku` を起動し、確認の画面で「Yes, I trust this folder」を選んでから `/exit` で終えた。画面は「No, exit」が選ばれた状態で出たので、Down を 1 回送ってから Enter を送った。捨ての workspace はそのあと閉じた。

## 結果

org は `bl1`、leader の herdr の agent の名前は `bl1_leader`、agmsg の team は `ralph-bl1`。

leader は次のコマンドで起こした。00:21:49 に始まり、00:21:53 に終了コード 0 で終わった。見張りが動いていなかったので、start が `ralph org watch --all` を起動した(stderr に `started the cross-org watch`)。

```
ralph org start --org-id bl1 --cwd <D>/work --driver claude --model haiku --scope 'live check, no edits' '<タスク>'
```

タスクの文は日本語で、次のことを頼んだ。

- ralph の実機の確かめであること
- 座席を spawn しない、ファイルを編集しない、subagent・/consult・skill を使わない、`ralph org report`・`disband`・`escalate` を打たない
- 待つときは、役割の説明の「agmsg の受信箱の運用」の節のとおりにする
- DECISION が届いたら SUMMARY の行を、座席のメッセージが届いたら本文の 1 行目を返事に書いて手番を終える
- DECISION が前面でコマンドを走らせるよう頼んだら、そのとおりにする

### (a) leader が自分で Monitor を張り、手番を終え、herdr が done と報告する

leader の claude の session の記録(`~/.claude/projects/` の下)で、最初の手番のツールの呼び出しを読んだ。

1. 00:21:54 に role prompt のファイルを Read した
2. 00:22:06 に `ralph org status --org-id bl1` を打った
3. 00:22:13 に Monitor を呼んだ。必須の `description` の引数がなく、`Invalid tool parameters` で断られた
4. 00:22:16 に `description`(`agmsg の受信箱を見張って DECISION や座席のメッセージを待つ`)を足して呼び直した。コマンドは雛形の 1 行のままで、`timeout_ms` は 1800000

Monitor に渡したコマンドは次のとおり。`{{AGMSG_SCRIPTS}}` は `'<D>/agmsg/scripts'` に置き換わっていた。

```
p=$PPID; end=$(( $(date +%s) + 1500 )); while kill -0 "$p" 2>/dev/null && [ "$(date +%s)" -lt "$end" ]; do bash '<D>/agmsg/scripts'/inbox.sh ralph-bl1 leader --quiet 2>&1 | grep --line-buffered -v '^[[:space:]]*$'; sleep 10; done; echo 'agmsg の見張りが終わった。history.sh で読み直してから張り直す'
```

`ralph org read --org-id bl1 --seat leader` で読んだ pane の行は次のとおり。`...` は task の id を省いたところ。

```
⏺ Monitor(agmsg の受信箱を見張って DECISION
         や座席のメッセージを待つ)
  ⎿  Monitor started · ... · timeout
     1800s

⏺ agmsg
  の受信箱を見張るモニターを開始しました。DECISION
  や座席のメッセージを待っています。

✻ Worked for 31s · done 9:22 AM · 1 monitor still
  running
```

- herdr の状態は、5 秒ごとに `herdr agent get bl1_leader` で読んだ。00:22:20 まで `working`、00:22:25 から `done` で、そのあとの 1 分半も `done` のままだった。`interactive_ready` は true。Monitor が動いたまま手番を終えた pane を、herdr は `done` と報告した。
- `ps -o pid,ppid,command` で読むと、ループの shell(pid 98061)は `/bin/bash -c source ~/.claude/shell-snapshots/... && ... eval '<上のコマンド>'` で、PPID は 95791 だった。95791 は leader の claude(`claude.exe --permission-mode bypassPermissions --model haiku 役割指示を読み込んで従ってください: ...`)で、その親は pane の `-bash`、さらにその親は分離した herdr server。ループの `$PPID` は leader の claude を指す。
- ループの子の `bash <D>/agmsg/scripts/inbox.sh ralph-bl1 leader --quiet` は、続けて読んだあいだに 4 回現れ、間は 10.23 秒、10.36 秒、10.23 秒だった。
- 雛形の文は直していない。sonnet の leader では打っていない。

### (b) 既定の 30 秒の send が届く

00:25:34 に、herdr が `done` と報告している leader へ次を打った。

```
ralph org send --org-id bl1 --to leader --text $'TYPE: DECISION\n\nSUMMARY: tsubaki-b'
```

- 00:25:35 に終了コード 0 で終わった。stdout は `sent message to seat "leader"`、stderr は空。直後の herdr の状態は `working`。
- pane には、送られた入力として `❯ TYPE: DECISION` と `SUMMARY: tsubaki-b` の 2 行が出て、その下に leader の返事 `⏺ tsubaki-b` が出た。入力欄は空だった。00:25:41 に `done` に戻った。
- 台帳の `sent` の Details は `TYPE: DECISION\n\nSUMMARY: tsubaki-b` で、`submit_unconfirmed=true` は付いていない。

### (c) 座席の名前で agmsg から送った文で、Monitor が leader を起こす

`<D>/agmsg/scripts/join.sh ralph-bl1 seat-x claude-code <D>/work` で偽の座席 `seat-x` を team に入れた(`Joined team ralph-bl1 as seat-x`、終了コード 0)。00:26:25 に `send.sh ralph-bl1 seat-x leader '<本文>'` で leader に送った。本文は `MARK: momiji-c`、`TYPE: HEARTBEAT`、空行、`live check の座席からのメッセージ` の 4 行。

- leader の session の記録では、Monitor の知らせ(`<task-notification>`)が 00:26:34.7 に入った。送ってから 9.7 秒。中身は `1 new message(s):` と、`seat-x: MARK: momiji-c\nTYPE: HEARTBEAT\n\nlive check の座席からのメッセージ` の 1 行だった(`inbox.sh` は本文の改行を `\n` の 2 文字にして 1 行で出す)。
- herdr は 00:26:35〜00:26:37 に `working` になり、00:26:38 に `done` に戻った。pane には `⏺ Monitor event: 1 new message(s): · agmsg の受信箱を…` と、leader の返事 `⏺ live check の座席からのメッセージ` が出た。
- leader は、header の後ろの空行の次の行を「本文の 1 行目」として返事に書いた。固有の印 `momiji-c` は header の行に置いたので返事には出ていないが、知らせには載っている。(f) からは固有の語を本文の 1 行目に置いた。

### (f) 重なり: DECISION が入力欄にあるあいだに Monitor が leader を起こす

2 回打った。1 回目は、Enter の前に Monitor の手番が終わった。2 回目は、Monitor の手番の途中に Enter が入るように時刻を合わせた。

#### 1 回目

- 00:28:09 に、`ralph org send --org-id bl1 --to leader --enter-delay-ms 15000 --timeout-ms 60000 --text $'TYPE: DECISION\n\nSUMMARY: kaede-f'` を裏で打った。00:28:12 の pane の入力欄に `TYPE: DECISION` と `SUMMARY: kaede-f` が入っていた。
- 00:28:11 に `send.sh` で `seat-x` から本文 `hagi-f の座席のメッセージ` を送った。
- 00:28:18.9 に Monitor の知らせが入り、leader の手番が始まった(herdr は 00:28:20〜00:28:21 に `working`)。leader は `hagi-f の座席のメッセージ` と返事を書き、00:28:22 に `done` に戻った。この手番のあいだ、DECISION の文は入力欄に残ったままで、手番に取り込まれなかった。
- 00:28:24 に send が Enter を送った。session の記録では、DECISION は 00:28:24.7 にふつうの入力として入り、leader は `kaede-f` と返事を書いた(00:28:28 に `done`)。
- send は 00:28:25 に終了コード 0 で終わった。stderr は空で、submit を確かめられなかった旨の注意は出ていない。`sent` の Details に `submit_unconfirmed=true` は付いていない。

#### 2 回目

- ループの `inbox.sh` が 00:30:53.95 に走ったのを `ps` で見て、2 秒後の 00:30:56.0 に `send.sh` で本文 `kikyo-f2 の座席のメッセージ` を送った(次の `inbox.sh` で読まれる)。00:30:56.8 に、`--enter-delay-ms 10000 --timeout-ms 60000` で `SUMMARY: susuki-f2` の DECISION を裏で打った。
- Monitor の知らせは 00:31:04.7 に入り、herdr は 00:31:05.3 から `working` になった。leader は 00:31:06.6 に `kikyo-f2 の座席のメッセージ` と返事を書いた。
- 00:31:06.9 ごろ、その手番の Stop の hook が走っているあいだに、send が Enter を送った。session の記録には 00:31:06.9 に DECISION を待ち行列に入れた記録(`queue-operation` の `enqueue`)が残り、00:31:07.0 の pane では入力欄に `Press up to edit queued messages` が出ていた。
- 00:31:07.8 に DECISION は待ち行列から出て、次の手番の入力になった。leader は 00:31:09.1 に `susuki-f2` と返事を書き、herdr は 00:31:10.9 に `done` に戻った。herdr は 00:31:05.3〜00:31:10.4 のあいだ `working` のままで、2 つの手番の境目に `done` は出ていない。
- send は 00:31:07.0 に終了コード 0 で終わった。stderr は空で、`sent` の Details に `submit_unconfirmed=true` は付いていない。submit の確かめが見た `working` は、Monitor の知らせで始まった手番のものだった。

どちらの回も、DECISION は入力欄に残らず、leader は 2 つのメッセージを両方扱った。

### (d) 対照: 前面で待っている leader には、5 秒の send が届かない

1 回目は、DECISION で Bash ツールの前面の `sleep 90` を頼んだ(00:32:17、終了コード 0)。leader は Bash ツールで `sleep 90` を呼んだが、Claude Code が次の文で断った。

```
Blocked: standalone sleep 90. To wait for a condition, use Monitor with an until-loop (e.g. `until <check>; do sleep 2; done`). To wait for a command you started, use run_in_background: true. Do not chain shorter sleeps to work around this block.
```

leader はそのエラーを返事に書いて手番を終えた。前面で待つ状態にならなかったので、この回の 5 秒の send(00:33:23、`SUMMARY: ume-d`)は終了コード 0 で届き、leader は `ume-d` と返事を書いた。

2 回目は、前面で 90 秒かかる別のコマンド `perl -e "select(undef,undef,undef,90)"` を、`run_in_background` を使わずに走らせるよう DECISION で頼んだ(00:34:12、終了コード 0)。

- 00:34:16 に herdr が `working` と報告し、`ps` に `perl -e select(undef,undef,undef,90)` が出た。その親の shell の親は leader の claude だった。pane には `Running perl -e "select(undef,undef,undef,90)"` と `(ctrl+b to run in background)` が出ていた。
- 00:34:16 に `ralph org send --org-id bl1 --to leader --timeout-ms 5000 --text $'TYPE: DECISION\n\nSUMMARY: sakura-d'` を打った。00:34:21 に終了コード 1 で終わり、stderr は次の 1 行だった。

```
org: send: wait for seat "leader" idle/done: herdr: timed out: context deadline exceeded (the seat's turn did not end within the --timeout-ms deadline of 5000 ms; nothing was typed; a seat busy in the foreground with a loop, `ralph org wait`, or a gate cannot receive a message until its turn ends; send it again later, or with a larger --timeout-ms)
```

- 直後の herdr の状態は `working`。pane の入力欄は空で、`sakura-d` はどこにも出ていない。台帳に `sakura-d` の `sent` は増えていない。
- 待ちが終わると、leader は `wait-done-d` と返事を書き、00:35:49 に `done` に戻った(`Churned for 1m 37s`)。Monitor のループは同じプロセス(pid 98061、PPID 95791)のまま動いていて、pane の状態の行は `1 monitor still running` だった。leader は張り直さず、前の見張りを使い続けた。

### (e) leader の claude を止めると、ループが残らない

00:35:57 に、leader の claude(pid 95791)へ `kill -KILL` を送った。ループ(pid 98061)は 13 分 40 秒動いていた。

- `ps` で 2 秒ごとに読んだ。00:35:57・00:35:59・00:36:01・00:36:03 にはまだあり、PPID は 1 に変わっていた。00:36:05 にはなかった。kill から 8 秒で、次の `kill -0 "$p"` で抜けて終わった。
- `inbox.sh ralph-bl1` のプロセスも、ループの `sleep 10` も残っていない。leader の claude が起動した MCP のサーバー(ユーザーの設定のもの)のプロセスも残っていない。
- 見張りは 00:36:22 に、leader の生死の ALERT を受信箱の件 `e1`(TYPE `ALERT`)として記録した。director を切っているので、その場で人に送った。watch.log に `WATCHDOG ESCALATION: org=bl1 item=e1 type=ALERT ...` の行が出て、osascript の shim が 00:36:23 に 1 回呼ばれた(`display notification "org bl1 raised e1 (ALERT)" with title "ralph org watch"`)。shim の呼び出しは、確かめ全体でこの 1 回だけだった。
- このとき `herdr agent get bl1_leader` は agent を返さず、`ralph org status` は leader を `spawned (active)` と出していた。

### disband

00:36:35 に `ralph org disband --org-id bl1` を打った。終了コード 0 で、stdout は `stopped seat "leader"` と `disbanded org "bl1"`、stderr は空。台帳には `stopped`(Details `ctrl_c=ok pane=closed leave=ok`)、`org_workspace_closed`(`workspace=closed`)、`disbanded` が増えた。`herdr workspace list` と `herdr agent list` は空になった。

- `ps` に `inbox.sh ralph-bl1` のループはない。`lsof -a -d cwd -c claude` に、cwd が `<D>` の下の claude はない。cwd が `<D>` の下のプロセスは、分離した herdr server と見張りだけだった。
- 見張りは `ralph org watch --stop` で止めた(00:37:04、終了コード 0、`stopped ralph org watch --all (pid 95778)`)。プロセスはなくなった。
- herdr server は `herdr server stop` で止めた(00:37:10、終了コード 0)。プロセスはなくなり、`pgrep -fl 'herdr server'` は何も返さなかった。
- そのあと `<D>` を消した。

### agmsg の hook と見張り

leader の claude の session で走った hook は、ユーザーの設定にある plugin のもの(SessionStart、PreToolUse、PostToolUse、UserPromptSubmit、Stop)で、agmsg の hook はなかった。agmsg の `watch.sh` に触れた文は、session の記録の中では role prompt の「張りません」の文だけだった。leader が張った見張りは、雛形の Monitor の 1 つだけだった。なお、SessionStart の plugin の hook は、別の session の要約を leader に渡していた。

## 分かったこと

- herdr 0.7.5 は、Monitor が動いたまま手番を終えた Claude Code の pane を `done` と報告する。計画の Open question の答えで、案 A の前提は成り立つ。`ralph org send` の idle/done の待ちが通り、既定の 30 秒の send は 1 秒で届いた。
- Monitor の知らせは、人の入力なしに leader の次の手番を始める。`send.sh` から知らせまでは 7〜10 秒だった((c) が 9.7 秒、(f) の 2 回目が 8.7 秒)。ループの 10 秒の間隔に収まる。
- ループの `$PPID` は leader の claude を指す。claude が `kill -KILL` で落ちると、ループは PID 1 の子になり、次の `kill -0` で終わる(8 秒)。計画の D11 の前提どおりだった。
- Monitor の知らせで始まった手番は、入力欄の文を取り込まない。DECISION の文は入力欄に残り、手番が終わってから Enter が来れば、ふつうの入力として送られる。手番の途中に Enter が来れば、Claude Code は待ち行列に入れ、手番が終わってから送る。どちらの場合も DECISION は失われなかった。
- 一方で、submit の確かめは 2 回目でも「確かめた」になった。見た `working` は Monitor の手番のもので、そのとき DECISION は待ち行列にあった。send が終わる直前(00:31:07.0)の pane では、DECISION はまだ扱われていなかった。待ち行列の DECISION も、pane には送られた入力と同じ `❯ TYPE: DECISION` の形で出て、違いは入力欄の `Press up to edit queued messages` だけだった。submit の確かめは Monitor の手番の `working` でも通るので、この確かめでは起きなかったが、DECISION が入力欄に残った場合とも見分けられない。director が resolve の前に `ralph org read` で読む手順(計画の D12)は要る。待ち行列に入った DECISION は手番のあとに扱われるので、入力欄に残っていなければ送れたと読んでよい。
- Claude Code の Bash ツールは、前面の `sleep 90` を自分で断り、Monitor か `run_in_background` を使うよう返す。それでも、前面で長くかかる別のコマンドは断られず、そのあいだ send は届かない。この確かめでは perl の 90 秒の待ちを使った。試行の leader の 9 秒ごとのループや gate も同じ形になる。新しいエラーの文は、手番が終わらなかったこと、何も打っていないこと、打ち直し方を伝える。
- haiku の leader は、最初の Monitor の呼び出しで必須の `description` を付けずに断られ、自分で付け直した。雛形は `description` に触れていない。

## 確かめていないもの

- Codex の leader(計画の Non-goals)、`guarded` の leader、opus と sonnet の leader
- Monitor のループの 25 分の期限での終わりと、終わりの 1 行のあとの張り直し、張り直す前の `history.sh` での読み直し。leader は確かめのあいだ、1 つの見張りを約 14 分使い続けた
- `ralph org wait` と gate を `run_in_background` で走らせる形。座席を spawn していないため
- `inbox.sh --quiet` が表示した直後に Monitor が止まり、知らせに載らないまま既読になる場合(計画の D3)

## 触れていないもの

- ユーザーの herdr server には触れていない。始める前と終わったあとの `pgrep -fl 'herdr server'` は、どちらも何も返さなかった(確かめのあいだは分離した 1 つだけ)。`~/.config/herdr` の下には、確かめを始めた 00:17 より新しいファイルがない。
- `~/.agents/skills/agmsg` は写しただけで、中身を変えていない。その下の `run/` と `db/messages.db-shm` は確かめのあいだに更新されたが、どちらも確かめの前から動いていたユーザーの別の claude の session(agmsg の見張り)によるもの。
- Orca は打っていない。本物の osascript も打っていない(shim の呼び出しは 1 回)。
- このセッションの herdr の pane、ほかの worktree、main のチェックアウト、ユーザーのほかの claude のプロセスには触れていない。

### 確かめで残ったもの

- フォルダ信頼に「信頼する」と答えたので、`~/.claude.json` に `<D>/work`(`/private/tmp/...` の形)を信頼済みとする項目が 1 つ増えた。ほかの Claude Code のセッションも同じファイルに書くので、手では消していない。
- leader と捨ての pane の claude の session の記録が、`~/.claude/projects/` の下に残っている。leader の claude はユーザーの `~/.claude` の設定で動いたので、plugin の記録(`~/.claude/session-data/`、`~/.claude/homunculus/` など)、`~/.claude/history.jsonl` の行、`~/.claude/teams/` の下の session の team の設定も残った。`kill -KILL` で止めたので、`~/.claude/sessions/` の下にその pid の記録も残っている。どれも消していない。
