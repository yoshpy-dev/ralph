# API エラーで止まった座席の画面と herdr の状態(2026-11-07)

仕様 `docs/specs/2026-11-07-org-seat-resume.md` の調査で確かめた事実の記録。
環境は macOS、Claude Code 2.1.312、codex-cli 0.165.0、herdr 0.7.5。herdr は
`docs/recipes/isolated-herdr-server.md` の手順で分離したサーバー
(`XDG_CONFIG_HOME`、`XDG_STATE_HOME`、`HERDR_SOCKET_PATH` を `/tmp` の下の専用の
場所に向け、`env -i` の最小の環境で起動)を使い、ユーザーの herdr には触れて
いない。調べ終えたあと、ワークスペースを閉じ、サーバーを止め、専用の場所を
消した。claude の信頼の記録(`~/.claude.json` の `/private/tmp/...` の項)と
セッションの記録(`~/.claude/projects/` の下)は残っている。

通信断は、存在しないホストを API の宛先にして起こした
(`ANTHROPIC_BASE_URL=https://ralph-probe.invalid`。`.invalid` は名前解決できない
と決まっているので、リクエストはどこにも届かない)。利用上限(429)には実際に
当たっていないので、その画面と状態は確かめていない。

## 1. `claude -p` の再試行(herdr の外)

`claude -p --model haiku "say hi"` を `</dev/null` で打ち、終わるまでの時間を
測った。`perl -e 'alarm N; exec @ARGV'` で打ち切りを付けた。

| 環境変数 | 結果 |
|---|---|
| `CLAUDE_CODE_MAX_RETRIES=2` | 13 秒で終了コード 1。stderr は `API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)` |
| なし(既定の 10 回) | 183 秒で終了コード 1。同じ文 |
| `CLAUDE_CODE_RETRY_WATCHDOG=1` | 420 秒の打ち切り(SIGALRM、終了コード 142)まで何も出さずに続いた |

## 2. 対話の claude がエラーでターンを終えた画面

pane の環境に `ANTHROPIC_BASE_URL=https://ralph-probe.invalid` と
`CLAUDE_CODE_MAX_RETRIES=2` を入れ(`herdr workspace create --env`)、
`herdr agent start probe-a --kind claude --pane <pane> -- --model haiku` で起動
した。新しいディレクトリなので、最初にフォルダ信頼の画面が出て、herdr は
`agent_status: blocked` と報告した(Down と Enter で答えた)。

`say hi` を送るとすぐにエラーでターンが終わり、herdr は `agent_status: idle` と
報告した。`herdr agent wait <name> --until idle --until done` もすぐ返った。
そのあと `herdr pane send-text` で `Continue from where you left off.` を打ち、
750ms 待って `herdr pane send-keys <pane> Enter` を送ると、状態は working に
なってから idle に戻った(宛先が壊れたままなので、同じエラーでまた止まった)。
そのときの `herdr pane read --source recent --lines 30 --format text` の末尾は
次のとおり。

```text
▎ Your Ultra plan includes $200 a month in API
▎ credits to power your apps and agents. Claim them
▎ at https://clau.de/api-credits

❯ say hi

⏺ API Error: Can't reach the API server —
  check your internet or DNS (ENOTFOUND)

✻ Worked for 1s · done 4:33 AM

❯ Continue from where you left off.

⏺ API Error: Can't reach the API server —
  check your internet or DNS (ENOTFOUND)

✻ Brewed for 1s · done 4:34 AM

─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ medium
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

この画面から、判定に関わる 3 点が分かる。

- エラー文は pane の幅で 2 行に折り返す。
- エラー文と入力欄の間に、`✻ … · done …` の状態行が 1 行入る。その前の動詞
  (`Worked`、`Brewed`)は回ごとに変わる。
- 入力欄は罫線 2 本に挟まれた `❯` の行で、その下に footer が 2 行続く。

## 3. 対話の claude が `CLAUDE_CODE_RETRY_WATCHDOG=1` で待つ画面

pane の環境に `ANTHROPIC_BASE_URL=https://ralph-probe.invalid` と
`CLAUDE_CODE_RETRY_WATCHDOG=1` を入れて同じように起動し、`say hi` を送った。
ターンは終わらず、5 秒ごとに 12 回(60 秒)読んだ `herdr agent get` は毎回
`agent_status: working` だった。`state_change_seq` は変わらず、`revision` は
7 → 12 → 17 → … → 65 と 5 秒ごとに増え、出力の文字列は毎回違った。この
pane を閉じる前(codex の確認を終えたあと、約 4 分後)にもう一度読むと、
状態は `working` のまま、`revision` は 295 だった。
最後に読んだ画面の末尾は次のとおり。

```text
❯ say hi






✻ Can't reach the… · Retrying in 4s · attempt 13/300

─────────────────────────────────────────────────────
❯
─────────────────────────────────────────────────────
  🤖 Haiku 5.5 | ⚡ medium
  ⏵⏵ bypass permissions on (shift+tab to cycle)
```

`ralph org watch --all` の stall の条件(`internal/org/watch.go` の
`evaluateSeat`)は、`herdr agent get` の出力の文字列が前の周回と同じことを
求める。この待ちでは文字列が変わり続けるので、stall の ALERT は出ない見込み
(見張りを走らせて確かめてはいない)。

## 4. 対話の codex が通信断で待つ画面

宛先に届かない独自の provider を `-c` で指定して起動した
(`-c model_provider=probe -c model_providers.probe.base_url=https://ralph-probe.invalid/v1 -c model_providers.probe.wire_api=responses -c model_providers.probe.request_max_retries=1 -c model_providers.probe.stream_max_retries=1`)。

起動の直後、フックの確認の 3 択画面が出た。herdr はこれを `blocked` ではなく
`agent_status: idle` と報告した。

```text
  Hooks need review
  7 hooks are new or changed.
  Hooks can run outside the sandbox after you trust
  them.


› 1. Review hooks
  2. Trust all and continue
  3. Continue without trusting (hooks won't run)

  enter confirm · esc skip
```

Esc で飛ばしたあと `say hi` を送ると、ターンを終えずに待ち続けた。2 分 56 秒の
時点でも同じで、そのあいだ `herdr agent get` は `agent_status: working`、
`revision: 1` のまま変わらなかった。

```text
› say hi


• Hook failed
  └ plugin contents changed; hook approval required

• Hook failed
  └ plugin contents changed; hook approval required

• Reconnecting... waiting for network (2m 56s • esc t…
  └ Connection failed: error sending request
  └ Tip: Use /mcp to list configured MCP tools.


› Ask Codex to do anything

  GPT-6-Astra · xhigh · Context 0% used
  ? for shortcuts                           ⚠ 11 · f2
```

この 2 つの画面から、次の 3 点が分かる。

- codex は再試行を 1 回にしても、通信断ではターンを終えずに回復を待つ。いつ
  まで待つかは確かめていない。
- 待っているあいだ `herdr agent get` の出力が変わらないので、stall の ALERT の
  条件に当たる見込み(既定の `stall_minutes` は 15)。
- herdr は知らない選択画面を idle と報告する。herdr の状態だけを見て文字を
  打つと、この画面に入力が入る。

## 5. codex のバイナリの文字列

codex-cli 0.165.0 のバイナリ(`strings -n 8`)に次の文があった。上限に当たった
ときの画面での並びは確かめていない。

- `You've hit your usage limit.`、`... or try again at ...`、`Try again at`
- `Usage limit reached. You've reached your usage limit. Increase your limits to continue using codex.`
- `stream connection failed; waiting to retry`、`Reconnecting... waiting for network`

上限のあとに自動で続ける機能を示す文(`continue automatically`、`auto-resume`
など)はなかった。
