# `ralph org resume` と再試行の設定の実機の確認(2026-11-07)

計画 `docs/plans/active/2026-11-07-org-seat-resume.md` の S10(AC-10、AC-15)の
記録。分離した herdr の上で claude の座席を通信断で止め、`ralph org resume` で
続け、`[org].api_wait_minutes` を小さくしたときのターンの終わりと stall の
ALERT までの時間を測った。

結果を先に書く。

- AC-10: 通信断で止まった座席を `ralph org resume` が「止まった」と判定し、
  続行の文を打ち、manifest に `resumed` を 1 件記録した。座席は文を受けて API に
  要求を出したが、この確認で使ったアカウントが週の利用上限に達していたため、
  作業(`done.txt` の作成)は最後まで進まなかった。「作業が最後まで進む」は
  確かめられていない(0 と 1)。
- AC-15: `api_wait_minutes = 2` で spawn した座席の環境に 3 つの変数が入って
  いた。接続の拒否で、座席は送信から 110 秒でターンを終え、その 25 秒後に
  見張りの stall の ALERT が届いた。再試行のあいだは ALERT が出なかった(2)。
- 週の上限に当たった画面と、再試行の設定があるときの上限の待ち(2 分で
  区切られた)を実機で見た。仕様の Open questions の 1 つ目に当たる(3)。

## 環境と分離

- macOS 26.5.2、Claude Code 2.1.312、herdr 0.7.5、agmsg 1.5.1(専用の写し)。
- ralph はこのブランチの `b5ea321f` を `go build -o /tmp/rlive/bin/ralph ./cmd/ralph`
  で作ったもの(`ralph version` は `dev`)。
- 座席のモデルはすべて `haiku`。

分離は `docs/recipes/isolated-herdr-server.md` のとおり。

- 専用の場所は `/tmp/rlive`(`mkdir -m 700`)。
- 1 つのコマンドを `env -i` で走らせるラッパー `/tmp/rlive/run` を作り、
  `ralph` と `herdr` はすべてこれを通した。渡した変数は `HOME`(実際のもの)、
  `USER`、`LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM`、`TMPDIR`、`PATH`
  (`/tmp/rlive/bin`、claude と node の実体のディレクトリ、`/opt/homebrew/bin`、
  システムのディレクトリ)、`XDG_CONFIG_HOME` と `XDG_STATE_HOME`
  (`/tmp/rlive` の下)、`HERDR_SOCKET_PATH=/tmp/rlive/h.sock`、
  `CLAUDE_ENV_FILE`、`GIT_CONFIG_GLOBAL`、`GH_CONFIG_DIR` と、下のプロキシの
  4 つ(`HTTPS_PROXY`、`https_proxy`、`NO_PROXY`、`no_proxy`)。
- herdr のサーバーは `nohup /tmp/rlive/run herdr server &` で起動した。
  `lsof -p` で見ると、開いているのは `/tmp/rlive` の下のファイル
  (`herdr-server.out`、`xdg-config/herdr/herdr-server.log`、ソケット 2 つ)と
  バイナリ、`/dev/null` だけで、作業ディレクトリは `/private/tmp/rlive` だった。
  `ps eww` の変数名はラッパーが渡したものだけだった。ユーザーの herdr サーバーには
  コマンドを送っていない。
- agmsg は `~/.agents/skills/agmsg` から `scripts`、`plugins`、`types`、
  `SKILL.md`、`VERSION` を `/tmp/rlive/agmsg` に写し、scratch の repo の
  `ralph.toml` の `agmsg_home` で指した。`osascript` は呼ばれた引数を記録して
  exit 0 する shim を `/tmp/rlive/bin` に置いた。
- org の repo は `/tmp/rlive/work` の scratch の git repo(`main` に base の
  コミット、repo の `user.name` と `user.email`、空のディレクトリを指す
  `core.hooksPath`、`.gitignore` に `.harness/state/` と `.harness/logs/`)。
- `ralph doctor` は herdr と agmsg(写し)を見つけた。「Shell aliases」の警告は
  `~/.zshrc` の alias についてで、pane の shell は bash なので届かない。捨てる
  pane で `type claude` は claude の実体を、`alias` は何も出さなかった。

## プロキシで通信を切る

標準ライブラリだけで書いた HTTP CONNECT プロキシ(約 100 行)を
`python3 -I /tmp/rlive/proxy.py 18765` で 127.0.0.1:18765 に立てた。CONNECT の
宛先へ TCP でつなぎ、両方向に中継するだけで、ログには時刻、宛先、バイト数だけを
書く(中身は TLS なので読まない)。ラッパーの `HTTPS_PROXY` が herdr サーバーを
通って座席の pane に入り、座席の claude の API への要求はこのプロキシを通った
(ログに `open api.anthropic.com:443` が出る。止めると座席は下の接続の拒否で
失敗した)。

- プロキシが動いているとき、`curl -x http://127.0.0.1:18765 https://api.anthropic.com/`
  は 404 を返した(API のホストまで届いた)。
- プロキシを止めると、同じ `curl` は
  `curl: (7) Failed to connect to 127.0.0.1 port 18765 after 0 ms: Couldn't connect to server`
  で終わった。座席の claude の画面は次の文になる。

```text
⏺ API Error: Connection refused — a firewall
  or proxy may be blocking it (ECONNREFUSED)
```

## 0. 準備: フォルダの信頼と、週の上限

捨てる workspace と pane を作り、`claude --model haiku` を起動した。信頼の
画面に Down と Enter で答えた。pane の shell では `HTTPS_PROXY` が見え、
`command -v osascript` は `/usr/bin/osascript` を返した(レシピの F2 のとおり、
pane の shell では `path_helper` がシステムのディレクトリを前に置く)。

claude の Bash ツールでの確かめを頼んだところ、要求はプロキシを通って API に
届き、次の画面でターンが終わった。

```text
❯ Run this with the Bash tool and show the raw
  output: command -v osascript ralph; printf "%s\n"
  "$HTTPS_PROXY"
  ⎿  You've hit your weekly limit · resets 11pm
     (Asia/Tokyo)
     /usage-credits to finish what you’re working on.

✻ Brewed for 1s · done 7:42 AM
```

実際の `HOME` の login(keychain)のアカウントが週の上限に達していた。
リセットは同じ日の 23 時(日本時間)で、この確認の約 15 時間後になる。ほかの
アカウントの login と `/usage-credits` は使わなかった。handoff の「止まった
部分はやめて、見せられたことと見せられなかったことを記録する」に従い、API に
届かなくても進む部分(通信断で止まる、判定する、打つ、記録する、時間を測る)を
進めた。claude を `/exit` で終え、捨てる workspace を閉じた。

## 1. AC-10: 通信断で止めて resume で続ける(`api_wait_minutes = 0`)

scratch の repo の `ralph.toml` は次のとおり。`watcher_enabled = false` は、
見張りの判断の層(`claude -p`)を走らせないため。

```toml
[org]
api_wait_minutes = 0
agmsg_home = "/tmp/rlive/agmsg"

[org.watchdog]
interval_seconds = 15
stall_minutes = 15
watcher_enabled = false
```

```sh
R=/tmp/rlive/run
cd /tmp/rlive/work
$R ralph org watch --all --detach
$R ralph org spawn --org-id t1 --id worker --role worker --driver claude --model haiku \
  --cwd /tmp/rlive/work --scope 'live check of ralph org resume'
```

役割 `worker` には role-prompt のテンプレートがなく、spawn はそのまま受け付けた
(最初の prompt は空)。既定の permission mode が `autonomous` なので
`--scope` を付けた。座席の claude のプロセスを `ps eww -o command= -p <pid>` で
読むと、`AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off`、
`RALPH_PRIMARY_CLI=claude` とプロキシの変数があり、`CLAUDE_CODE_RETRY_*` と
`CLAUDE_CODE_MAX_RETRIES` はなかった(`api_wait_minutes = 0` のとおり)。

時刻は日本時間。manifest と Claude Code のセッションの記録は UTC(9 時間前)で
書いてある。

1. 7:45:13 にプロキシを止め、7:45:16 にタスクを送った。

   ```sh
   $R ralph org send --org-id t1 --to worker --raw \
     --text 'Create the file done.txt in the current directory containing the word ok, then reply DONE.'
   ```

   出力は `sent message to seat "worker"`。
2. 再試行のあいだ、herdr は `working` を返し、画面には
   `✻ Connection refus… · Retrying in 10s · attempt 6/10` のような行が出た。
   7:45:48 の `ralph org resume --dry-run` は次のとおり。

   ```text
   waiting  t1/worker  claude  herdr reports working
   resume (dry run): 0 would resume, 0 would enter, 1 waiting, 0 skipped
   ```

3. 座席はエラーでターンを終えた。セッションの記録では、タスクの受け取りが
   22:45:17.752Z、エラーが 22:48:14.844Z で、177 秒かかった(既定の 10 回)。
   5 秒ごとの確かめでは、7:48:17 に herdr が `idle` を返した。
   `ralph org read --org-id t1 --seat worker --lines 30` の末尾は次のとおり。

   ```text
   ❯ Create the file done.txt in the current directory
     containing the word ok, then reply DONE.

   ⏺ API Error: Connection refused — a firewall
     or proxy may be blocking it (ECONNREFUSED)

   ✻ Churned for 2m 57s · done 7:48 AM





   ─────────────────────────────────────────────────────
   ❯
   ─────────────────────────────────────────────────────
     🤖 Haiku 5.5 | ⚡ medium | 🌿 main
     ⏵⏵ bypass permissions on (shift+tab to cycle)
   ```

4. 7:48:30 に、プロキシを止めたまま `ralph org resume --dry-run` を打った
   (終了コード 0)。manifest の `resumed` は 0 件のままだった。

   ```text
   would_resume  t1/worker  claude  ⏺ API Error: Connection refused — a firewall or proxy may be blocking it (ECONNREFUSED)
   resume (dry run): 1 would resume, 0 would enter, 0 waiting, 0 skipped
   ```

5. 7:48:34 にプロキシを起動し、7:48:38 に `ralph org resume` を打った。約 1 秒で
   終わり、終了コードは 0。

   ```text
   resumed  t1/worker  claude  ⏺ API Error: Connection refused — a firewall or proxy may be blocking it (ECONNREFUSED)
   resume: 1 resumed, 0 entered, 0 waiting, 0 skipped, 0 failed
   ```

   manifest には次の 1 行が増えた(`pane_id` の項は省いた)。

   ```json
   {"ts":"2026-11-06T22:48:39Z","org_id":"t1","seat_id":"worker","event":"resumed","details":"kind=stopped line=⏺ API Error: Connection refused — a firewall or proxy may be blocking it (ECONNREFUSED)"}
   ```

6. セッションの記録では、22:48:39.498Z に続行の文が利用者の入力として入った
   (先頭の `` `ralph org resume` `` のバッククォートも含む。画面はこれを
   インラインコードとして描くので、バッククォートなしに見える)。座席はすぐに
   API へ要求を出し(プロキシのログに 7:48:38〜7:48:40 の
   `open api.anthropic.com:443`)、0.5 秒後の 22:48:39.978Z に `rate_limit` の
   エラーでターンを終えた。`done.txt` は作られなかった。

   ```text
   ❯ ralph org resume からの続行の指示です。直前の API
     エラー(通信断か利用上限)で作業が止まっていました。
     止まったところから続けてください。
     ⎿  You've hit your weekly limit · resets 11pm
        (Asia/Tokyo)
        /usage-credits to finish what you’re working on.

   ✻ Churned for 0s · done 7:48 AM
   ```

7. 「すぐにもう 1 回 resume」は、座席が週の上限でまた止まっていたので、次の
   3 回にした。
   - 7:49:13 の `--dry-run` は `would_resume`。照合した行は、画面の prompt の
     行から上限の文までをつないだ 1 行だった(4 の 1 つ目)。

     ```text
     would_resume  t1/worker  claude  ❯ ralph org resume からの続行の指示です。直前の API エラー(通信断か利用上限)で作業が止まっていました。 止まったところから続けてください。 ⎿  You've hit your weekly limit · resets 11pm (Asia/Tokyo) /usage-credits to finish what you’re working on.
     ```

   - 7:49:24 の `ralph org resume` は同じ行で `resumed`。座席は 22:49:25.018Z に
     文を受け、22:49:25.701Z に同じ上限でターンを終えた。
   - その約 1 秒後(7:49:25)の `ralph org resume` は、座席が動いているので
     打たなかった。終了コードは 0。

     ```text
     waiting  t1/worker  claude  herdr reports working
     resume: 0 resumed, 0 entered, 1 waiting, 0 skipped, 0 failed
     ```

   manifest の t1 の `resumed` は 2 件(7:48:39 と 7:49:25)で、セッションの記録の
   続行の文も 2 件だった。

AC-10 について確かめたこと: 通信断(接続の拒否)で止まった claude の座席を、
判定(`would_resume`)、文の入力と Enter、`resumed` の記録、動いている座席を
飛ばすことまで実機で確かめた。続行の文を受けた座席は、新しいターンで API に
要求を出し直した。確かめていないこと: 座席が作業を最後まで進めること。API が
週の上限を返したため。

## 2. AC-15: 再試行の設定で待ち、stall の ALERT が届くまで

t1 を `ralph org disband --org-id t1` で終え、見張りを
`ralph org watch --all --stop` で止め、`ralph.toml` を次のように変えてから
見張りを起動し直した(見張りは設定を起動時に読む)。

```toml
[org]
api_wait_minutes = 2
agmsg_home = "/tmp/rlive/agmsg"

[org.watchdog]
interval_seconds = 15
stall_minutes = 1
watcher_enabled = false
```

`stall_minutes` は handoff の例の 2 ではなく 1 にした。stall の条件は「座席の
最後のイベントから `stall_minutes` を過ぎた」と「`herdr agent get` の出力が前の
周回と同じ」の両方で、送信(`sent` のイベント)のあと座席は約 110 秒再試行する。
2 分にすると、再試行のあいだは時間の条件がまだ満たされないので、ALERT が出ない
ことが「出力が変わり続けるから」なのかを確かめられない。1 分にすると、送信の
60 秒後から再試行が終わるまでの約 50 秒(見張りの周回で 3 回ほど)、時間の条件が
満たされたまま、出力の条件だけで ALERT が止まっているかを見られる。

```sh
$R ralph config get org.api_wait_minutes          # 2
$R ralph config get org.watchdog.stall_minutes    # 1
$R ralph org watch --all --detach
$R ralph org spawn --org-id t2 --id worker --role worker --driver claude --model haiku \
  --cwd /tmp/rlive/work --scope 'live check of the retry variables'
```

座席の環境は、herdr サーバーの子の bash の下の claude のプロセスを
`ps eww -o command= -p <pid>` で読み、名前で絞って確かめた。

```text
CLAUDE_CODE_MAX_RETRIES=8
CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=120000
CLAUDE_CODE_RETRY_WATCHDOG=1
```

8 は換算の式(`docs/evidence/org-seat-resume-retries-2026-11-07.md`)の 2 分の値、
120000 は 2 × 60000 で、FR-8 のとおり。

7:50:49 にプロキシを止め、7:50:56 にタスクを送った(本文は t1 の `done.txt` を
`done2.txt` にしたもの)。セッションの記録の受け取りは 22:50:58.379Z、manifest の
`sent` は 22:50:58Z。画面には `✻ Connection refus… · Retrying in 17s · attempt 7/8`
のような行が出た。

- (i) 再試行のあいだ: 7:52:14(送信の 76 秒後、時間の条件はもう満たされている)に
  `--dry-run` なしの `ralph org resume --org-id t2` を打った。打たずに終わり、
  終了コードは 0。

  ```text
  waiting  t2/worker  claude  herdr reports working
  resume: 0 resumed, 0 entered, 1 waiting, 0 skipped, 0 failed
  ```

  5 秒ごとに読んだ `watch-status-t2.json` の有効な条件は、ターンの終わりまで
  空だった。専用の agmsg の写しにも、この間の ALERT はなかった。
- (ii) ターンの終わり: エラーは 22:52:48.750Z で、受け取りから 110.4 秒だった
  (状態行は `✻ Baked for 1m 50s · done 7:52 AM`)。画面のエラー文は t1 と同じ。
  換算の記録が m = 2 に見込んだ約 120 秒より 10 秒短い。
- (iii) stall の ALERT: `watch-status-t2.json` の `t2/worker/stall` の `first_ts` が
  22:53:12Z、agmsg の写しの ALERT が 22:53:14Z だった。ターンの終わりから
  25 秒、送信から 136 秒。ALERT の本文は次のとおり(改行を ` | ` にした)。

  ```text
  TYPE: ALERT | ORG_ID: t2 | SEAT: worker | CONDITION: stall |  | seat worker heartbeat stalled for over 1m
  ```

  見張りの周回は 15 秒ごとで、ターンの終わりのあとの最初の周回は出力が
  `working` から `idle` に変わったのを見て、次の周回で同じ出力を見て ALERT を
  出した。

そのあと 7:53:49 にプロキシを起動し、`--dry-run` で `would_resume`(t1 と同じ
接続の拒否の行)を確かめてから、7:53:53 に `ralph org resume` を打った。結果は
`resumed  t2/worker  claude  ⏺ API Error: Connection refused — …`、
`resume: 1 resumed, 0 entered, 0 waiting, 0 skipped, 0 failed`、終了コード 0 で、
manifest に `resumed`(22:53:55Z)が増えた。座席は続行の文で要求を出し直したが、
t1 と同じく週の上限が返った(3)。

AC-15 について確かめたこと: 3 つの変数が spawn した claude の座席の環境に入る。
接続の拒否では座席が送信から約 110 秒でターンを終え、そのあと 25 秒で stall の
ALERT が届く。再試行のあいだは、時間の条件が満たされていても ALERT が出ない。

## 3. 週の上限に当たったときの待ち(再試行の設定あり)

2 の最後の resume のあと、座席はすぐにはターンを終えなかった。herdr は
`working` を返し、画面の状態行は次のように数え下げた。

```text
✻ Weekly lim… · Retrying in 1m 57s (11pm) · attempt 1/8
```

- 7:54:39 の `ralph org resume --dry-run` は
  `waiting  t2/worker  claude  herdr reports working`。
- 待ちのあいだ、stall の条件は消え(7:54:04 に空になった)、ALERT は出なかった。
  `resumed`(22:53:55Z)の 1 分後から待ちの終わりまでの約 60 秒、時間の条件は
  満たされていた。
- 続行の文の受け取り(22:53:54.918Z)から 121.0 秒後の 22:55:55.927Z に、
  `rate_limit` のエラーでターンを終えた(状態行 `✻ Baked for 2m 1s`)。
  `attempt` は 1/8 のまま増えなかったので、待ちを区切ったのは
  `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS` の 120 秒で、回数ではない。
  画面は t1 の 6 と同じ形(prompt の行の下の `⎿  You've hit your weekly limit`)。
- stall の ALERT は 22:56:20Z(ターンの終わりから 24 秒)に届いた。
- そのあとの `--dry-run` は、t1 と同じく prompt の行から上限の文までをつないだ
  行で `would_resume` だった。

再試行の設定がない t1 では、同じ上限で 0.5〜0.7 秒でターンを終えた。リセットが
約 15 時間先の上限でも、再試行の設定があると `api_wait_minutes` の分だけ待って
からターンを終える。仕様の Desired state の「上限と混雑の待ちはこの長さで区切
られる」に合う。

## 4. 仕様どおりでなかった点と、気づいた点

コードはこの slice では直していない。

1. 上限の画面で、照合した行に prompt の行が入る。Claude Code は上限の文を、
   空行を挟まずに prompt の行の下に `⎿` の行として出す。画面の判定は空行で
   区切ったまとまりを 1 行につなぐので、prompt の行(ここでは続行の文)と上限の
   文が 1 つのまとまりになった。判定は正しく「止まった」だったが、FR-7 の結果の
   行と manifest の `resumed` の `line=` には続行の文がそのまま入る。照合はまとまり
   全体の部分文字列なので、一覧の文を含む prompt のすぐ下に `⎿` の行が出た画面も
   「止まった」になりうる(実機では確かめていない)。
2. AC-10 の「座席が作業を最後まで進める」は、週の上限のため確かめられなかった。
3. 画面の文の一覧で `You've hit your weekly limit` の出どころはドキュメント
   (`docs:`)になっている。この確認で Claude Code 2.1.312 の実際の画面に出た。
4. 接続の拒否の文は、名前解決の失敗の文(記録
   `docs/evidence/org-seat-resume-probe-2026-11-07.md` の 2)と違い
   `API Error: Connection refused — a firewall or proxy may be blocking it (ECONNREFUSED)`
   だった。一覧の `API Error:` に当たった。
5. 再試行のあいだ、座席はユーザーの `~/.claude` のプラグインで動いており、
   状態行と入力欄のあいだに `MCP server "atlassian" disconnected · open /mcp …` の
   行が出ていた。ターンの終わりには消えており、判定には影響しなかった。
6. 入力欄の確かめ(FR-5a)は 3 回の resume ですべて通り、`failed` も Enter の
   見送りもなかった。

## 確かめていないこと

- 続行の文を受けた座席が作業を最後まで進めること(週の上限のため)。上限の
  リセット(2026-11-07 23 時、日本時間)のあとか、別のアカウントで、1 の手順を
  繰り返すと確かめられる。
- 応答が返らない通信断(この確認は接続の拒否だけ)。
- `Press enter to continue` の Enter だけの道、codex の座席、headless の director、
  leader とほかの座席の順、複数の org。
- 見張りの判断の層(`watcher_enabled = false` にした)と、deadman の escalation
  (10 分たつ前に片付けた)。
- 専用の agmsg の写しを座席が使うこと(`worker` の座席はメッセージを送らない)。
- handoff の例の `stall_minutes = 2`(上の理由で 1 にした)。

## 片付け

1. `ralph org disband --org-id t1`(2 の前)と `--org-id t2` は、どちらも
   `stopped seat "worker"`、`disbanded org "<id>"` で終わった。
   `ralph org inbox` は `no open or acked inbox items`。
2. `ralph org watch --all --stop` のあと、`ralph status` は
   `org watch: not running (stopped at 2026-11-06T22:58:22Z)` を出した。
3. プロキシを止め、18765 番で待ち受けるプロセスがないことを `lsof` で確かめた。
4. `~` に移ってから、`pgrep -fl /tmp/rlive` は何も出さず、`lsof +D /tmp/rlive` は
   専用の herdr サーバーだけを出した。
5. `/tmp/rlive/run herdr server stop` のあと、`pgrep -fl 'herdr server'` に出るのは
   ユーザーのサーバーだけになった。`h.sock` と `h-client.sock` は消え、
   `lsof +D /tmp/rlive` は何も出さず、`session.json` もなかった。
6. ログと台帳の写しを `$TMPDIR` に移してから `rm -rf /tmp/rlive` を打った。

専用の場所の外には、次のものが残った。

- `~/.claude.json` の信頼の項(`/private/tmp/rlive/work`)。
- `~/.claude/projects/-private-tmp-rlive-work/` のセッションの記録 3 つ(捨てる
  pane、t1、t2)と `memory` のディレクトリ。
- ユーザーの agmsg の DB には、この確認のチーム(`ralph-t1`、`ralph-t2`)の行は
  messages と events のどちらにもなかった。DB の更新時刻は確かめの途中で
  変わっていたが、その間の行はユーザー自身の org の見張りのものだった。
- `osascript` の shim のログは作られなかった(通知は 1 回も呼ばれていない)。
