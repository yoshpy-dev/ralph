# codex の座席への bracketed paste の実機の確かめ(2026-11-15、#326)

計画 2026-11-15-send-codex-paste の AC5 の記録。この branch で
作った `ralph` の `ralph org send` で、分離した herdr の上の codex の座席に、本文が
日本語 2000 字の DECISION を 1 回送った。

結果は AC5 の条件をすべて満たした。

- send は exit 0 で、約 1 秒で終わった(12:30:17Z〜12:30:18Z)。`sent` の記録に
  `submit_unconfirmed` はなく、ESC(`\x1b`)の字もなかった。send は herdr の道を
  通った(`via=` はない)。
- codex のセッションの記録の user の文は、送った文(3938 バイト、2016 字)と
  バイト単位で同じだった。sha256 は両方とも
  `f4ba222f1521d599ae9836fe2aca6b050f62aeb92186e4ca4d88ab181b342210`。
- 座席は、文の最後の行が求めた語「受信326」を返した。
- codex の手番は、この 1 回だけ使った。claude の座席は使っていない。

## 環境と分離

- macOS 26.5.2、codex-cli 0.166.0、herdr 0.7.5、agmsg(専用の写し)。
- `ralph` は branch `fix/send-codex-paste` の 4403bef4 を
  `go build -o /tmp/ml308/bin/ralph ./cmd/ralph` で作った。
- herdr は `docs/recipes/isolated-herdr-server.md` のとおりに分けた。`env -i` の
  包みのスクリプトで `XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を
  `/tmp/ml308/` の下に向けた。`HOME` は本物のまま。codex の binary は
  `/tmp/ml308/bin/codex` から symlink を張った。分離した server は
  `nohup /tmp/ml308/run herdr server &` で起動した。
- agmsg は `~/.agents/skills/agmsg` から `scripts`、`plugins`、`types`、`SKILL.md`、
  `VERSION` を `/tmp/ml308/agmsg` に写した。`osascript` は呼ばれた引数を記録する
  shim を置いた(一度も呼ばれなかった)。
- org の repo は `/private/tmp/ml308/work` の scratch の git repo(codex で信頼済み)。
  `ralph.toml` は次のとおり。

```toml
[org]
agmsg_home = "/tmp/ml308/agmsg"

[org.watchdog]
watcher_enabled = false
```

- ユーザーの `herdr server`(pid 34407)には、コマンドを 1 つも送っていない。

## 手順

```sh
R=/tmp/ml308/run
cd /tmp/ml308/work
$R ralph org watch --all --detach
$R ralph org spawn --org-id t1 --id worker --role worker --driver codex --model gpt-6-luna \
  --cwd /tmp/ml308/work --scope 'live check of ralph org send bracketed paste to codex (#326)'
```

spawn は `spawned seat "worker" (org_id=t1 driver=codex model=gpt-6-luna pane_id=w1:p1 dry_run=false)`
を返した。codex の更新の案内は `2. Skip`、フックの確認は
`3. Continue without trusting (hooks won't run)` で、pane に Down と Enter を送って
答えた。入力待ちの画面(`› Ask Codex to do anything`)になった直後、herdr は座席を
`working` と報告していたが、15 秒のうちに `idle` に変わった。

送った文は、本文 29 行、2000 字(`utf8.RuneCountInString`)の DECISION。全体は
3938 バイト、2016 字。最後の行は次のとおり。

```
最後の行です。この文を頭から最後まで受け取れたら、返事には「受信326」とだけ書き、ほかのことはしないでください。
```

```sh
$R ralph org send --org-id t1 --to worker --text "$(cat msg.txt)"
```

`sent message to seat "worker"` と exit 0 で終わった。手番は `Worked for 5s` で終わり、
画面に `• 受信326` が出た。返事の前後には、ユーザーの codex のフックの
`• Hook failed` の行が出た(信頼しないを選んだため)。

届いた文は、codex のセッションの記録
(`~/.codex/sessions/2026/11/15/rollout-2026-11-15T21-29-35-….jsonl`)の user の文と
比べた。user の文は 2 つで、2 つ目が送った文とバイト単位で同じだった。ESC の字は
どちらにもなかった。

## 片付け

1. `ralph org disband --org-id t1` は `stopped seat "worker"` と `disbanded org "t1"`
   で終わった。`ralph org inbox` は `no open or acked inbox items` を返した。
2. `ralph org watch --all --stop` のあと、`ralph status` は
   `org watch: not running (stopped at 2026-11-15T12:30:56Z)` を返した。
3. `~` に移ってから `/tmp/ml308/run herdr server stop` を打った。
   `pgrep -fl 'herdr server'` に出るのはユーザーのサーバー(pid 34407)だけで、
   `pgrep -fl ml308` は何も出さず、`h.sock` もなかった。
4. `rm -rf /tmp/ml308` を打った。

専用の場所の外には、codex のセッションの記録と、`~/.codex/history.jsonl` の
送った文の行が残った。

## 確かめていないこと

- 画面の道(herdr が `working` と言い続ける場合)を、ralph で通しで走らせること。
  今回は herdr が 15 秒のうちに `idle` に変わったので、send は herdr の道を通った。
  画面の道の codex の置き換えの規則は、実機の画面の fixture のテストで固定している。
- codex-cli 0.166.0 のほかの版。
