# org send の bracketed paste の実機の確かめ(2026-11-13、#319)

計画 `docs/plans/active/2026-11-13-send-claude-paste.md` の AC5 の記録。
この branch で作った `ralph` の `ralph org send` で、分離した herdr の上の
claude の座席に、本文が日本語 2000 字の DECISION を 1 回送った。

結果は AC5 の条件をすべて満たした。

- send は exit 0 で、約 2 秒で終わった。`sent` の記録に
  `submit_unconfirmed=true` はなかった。
- 座席のセッションの記録には user の文が 1 つあり、その中は
  `<pasted_content>` が 1 つだった。中身は送った文(4414 バイト)と
  バイト単位で同じだった。
- 座席は、文の最後の行が求めた語「受信319」だけを返した。
- claude の手番は、この 1 回だけ使った。

typed protocol の本文の上限は 2000 字(`DefaultMaxBodyChars`)で、この文は
その上限ちょうどの本文を持つ。上限までの大きさで、届かない大きさは
見つからなかった。そのため send に大きさで断る規則は足していない。

## 環境と分離

- macOS 26.5.2、Claude Code 2.1.316、herdr 0.7.5、agmsg 1.5.1(専用の写し)。
- `ralph` は branch `fix/send-claude-paste` の 331eebd3 を
  `go build -o /tmp/ml319b/bin/ralph ./cmd/ralph` で作った。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおりに分けた。
  `env -i` の包みのスクリプトで `XDG_CONFIG_HOME`、`XDG_STATE_HOME`、
  `HERDR_SOCKET_PATH` を `/tmp/ml319b/` の下に向け、`CLAUDE_ENV_FILE` も
  そこに置いた。`HOME` は本物のまま、`GIT_CONFIG_GLOBAL` は
  `$HOME/.config/git/config` にした。分離した server は
  `nohup /tmp/ml319b/run herdr server &` で起動した。
- ユーザーの `herdr server`(pid 34407)には、コマンドを 1 つも送っていない。
  確かめの前とあとで、同じ pid で動いていた。
- agmsg は `~/.agents/skills/agmsg` から `scripts`、`plugins`、`types`、
  `SKILL.md`、`VERSION` を `/tmp/ml319b/agmsg` に写し、scratch の repo の
  `ralph.toml` の `agmsg_home` で指した。`osascript` は呼ばれた引数を記録して
  exit 0 する shim を `/tmp/ml319b/bin` に置いた。
- org の repo は `/tmp/ml319b/work` の scratch の git repo にした。`main` に
  base のコミットが 1 つあり、repo の `user.name` と `user.email`、空の
  ディレクトリを指す `core.hooksPath` を設定し、`.gitignore` に
  `.harness/state/` と `.harness/logs/` を書いた。`/private/tmp/ml319b/work` は
  `~/.claude.json` で信頼済み(probe のときの項)で、trust の画面は出なかった。

`ralph.toml` は次のとおり。`watcher_enabled = false` は、見張りの判断の層
(`claude -p`)を走らせないため。

```toml
[org]
agmsg_home = "/tmp/ml319b/agmsg"

[org.watchdog]
watcher_enabled = false
```

```sh
R=/tmp/ml319b/run
cd /tmp/ml319b/work
$R ralph org watch --all --detach
$R ralph org spawn --org-id t1 --id worker --role worker --driver claude --model haiku \
  --cwd /tmp/ml319b/work --scope 'live check of ralph org send bracketed paste (#319)'
```

spawn は `spawned seat "worker" (org_id=t1 driver=claude model=haiku pane_id=w1:p1 dry_run=false)`
を返した。画面の表示は `Haiku 5.5 with low effort` で、pane の幅は 53 桁だった。
herdr は座席を `idle` と報告し、入力欄は空だった。

## 送った文

本文は 40 行で、ちょうど 2000 字(`utf8.RuneCountInString`)。全体は
`TYPE: DECISION`、空の行、本文の順で 4414 バイト、改行は 40 個。
各行の頭に `第01行:` から `第38行:` までの番号を付け、頭が欠けたときに
どの行から届いたかがわかるようにした。最後の行は次のとおり。

```
最後の行です。この文を頭から最後まで受け取れたら、返事には「受信319」とだけ書き、ほかのことはしないでください。
```

```sh
$R ralph org send --org-id t1 --to worker --text "$(cat msg.txt)"
```

- 04:10:05Z に打ち、04:10:07Z に `sent message to seat "worker"` と exit 0 で
  終わった。
- 台帳の `sent` の記録(04:10:07Z)の `details` は、`TYPE: DECISION` で
  始まる送った文の頭で、ESC(`\x1b`)の字はなかった。`submit_unconfirmed` と
  `via=` は記録のどこにもなかった。send は herdr の道を通り、画面の道は
  使っていない。

## 座席に届いた文

座席のセッションの記録
(`~/.claude/projects/-private-tmp-ml319b-work/<セッション id>.jsonl`)には、
user の文が 1 つだけあった。形は次のとおり。

```
\n\n<pasted_content id="…">\n<送った文>\n</pasted_content id="…">\n
```

- `<pasted_content>` は 1 つで、その外にあるのは前の改行 2 つと後ろの改行 1 つ
  だけだった(probe の B1・B2 と同じ形)。
- 開きの札のあとの改行から閉じの札の前の改行までのあいだを取り出し、送った文と
  `cmp` で比べると同じだった。sha256 は両方とも
  `bf2a1da66a167e96f9f0a9ab93466d6380c31c588e4a2e1ffa8074509af217f7`。
  ESC の字は入っていなかった。
- 画面には、`第35行:` から最後の行までと、返事の `⏺ 受信319` が出た。
  手番は `Crunched for 4s` で終わった。

## 空の入力欄での Ctrl+C(手番を使わない確かめ)

send の前、座席が空の入力欄で待っているあいだに、`herdr pane send-keys w1:p1 ctrl+c`
を 1 回だけ打った。

- 0.3 秒後、入力欄の下に `Press Ctrl-C again to exit` が出た。
- 4 秒後には消え、下の行はふだんの表示に戻った。herdr の状態は `idle` のままで、
  claude は終わらなかった。

この結果を `agent-messaging.md` の「How send types a message」の最後の文に使った。
2 回目の Ctrl+C は打っていない。

## 片付け

1. `ralph org disband --org-id t1` は `stopped seat "worker"` と
   `disbanded org "t1"` で終わった。`ralph org inbox` は
   `no open or acked inbox items` を返した。
2. `ralph org watch --all --stop` のあと、`ralph status` は
   `org watch: not running (stopped at 2026-11-13T04:11:00Z)` を返した。
3. `~` に移ってから `/tmp/ml319b/run herdr server stop` を打った。
   `pgrep -fl 'herdr server'` に出るのはユーザーのサーバー(pid 34407)だけになり、
   `pgrep -fl ml319b` と `lsof +D /tmp/ml319b` は何も出さず、`h.sock` もなかった。
4. `osascript` の shim のログは作られなかった(通知は 1 回も呼ばれていない)。
5. 台帳とログの写しを scratchpad に移してから `rm -rf /tmp/ml319b` を打った。

専用の場所の外には、次のものが残った。

- `~/.claude.json` の信頼の項(`/private/tmp/ml319b/work`、probe のときから)。
- `~/.claude/projects/-private-tmp-ml319b-work/` のセッションの記録。

## 確かめていないこと

- Claude Code 2.1.316 と herdr 0.7.5 のほかの版。
- `--raw` で送る 2000 字を超える文、1 行が 800 バイトを超える文、10 KB を超える文。
- 前の送信の置き換えが入力欄に残っているときの送信。
- 打ち込みの途中で切れた send(probe の「打ち込みが途中で切れたときの確かめ」を見る)。
- 2 回目の Ctrl+C で claude が終わること。

最後の 1 つのほかは、`docs/tech-debt/README.md` の「Known gaps of
fix/send-claude-paste (#319)」に残した。
