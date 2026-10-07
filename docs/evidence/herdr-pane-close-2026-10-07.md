# herdr の pane と workspace を閉じる操作の実機確認

- Date: 2026-10-07
- Plan: docs/plans/archive/2026-10-07-org-stop-all.md(PR 作成時に active から移動)
- 目的: `ralph org stop` が座席の pane を閉じ、`ralph org disband` が org の workspace を閉じるときに使う herdr の操作(`herdr pane close` / `herdr workspace close`)の応答と、閉じたあとの pane のプロセスを確かめる。あわせて、コマンドを打った pane を見分ける手がかりになる環境変数を確かめる。ralph はこの結果を前提に、見つからないという応答を閉じ済みとして扱い(`internal/org/driver/herdr.go` の `IsNotFound`)、`HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` で自分の pane と workspace を見分ける(`internal/org/verbs.go`)

## 環境

| 項目 | 値 |
|---|---|
| herdr | 0.7.5 |
| OS | macOS |
| サーバー | 隔離した `herdr server`。`XDG_CONFIG_HOME` / `XDG_STATE_HOME` / `HERDR_SOCKET_PATH` を `/tmp/hp` の下に向けた。利用者の herdr サーバーと `~/.config/herdr/` には触れていない |

## Run 1: 見つからない id を閉じる

```sh
herdr pane close w99:p99
herdr workspace close w99
```

- `pane close`: rc=1、`{"error":{"code":"pane_not_found","message":"pane w99:p99 not found"},"id":"cli:pane:close"}`
- `workspace close`: rc=1、`{"error":{"code":"workspace_not_found","message":"workspace w99 not found"},"id":"cli:workspace:close"}`

## Run 2: pane に C-c を送る、pane を閉じる

`herdr tab create --workspace w1 --cwd /tmp/hp --label seat1` で作った tab の pane(w1:p2)で確かめた。

1. `herdr pane run w1:p2 "sleep 777"` のあと、`sleep 777` が動いていた
2. `herdr pane send-keys w1:p2 C-c` で `sleep 777` は終わった(ふつうのプロセスは C-c で終わる)
3. `herdr pane run w1:p2 "sleep 778"` のあと `herdr pane close w1:p2` を打った。rc=0、`{"id":"cli:pane:close","result":{"type":"ok"}}`。`sleep 778` は終わり、pane が 1 つだけだった tab は tab の一覧から消えた

## Run 3: workspace を閉じる

`herdr workspace close w1` は `{"id":"cli:workspace:close","result":{"type":"ok"}}` を返し、workspace の一覧は空になった。

## pane の中の環境変数

pane の中のプロセスの環境には `HERDR_ENV=1`、`HERDR_PANE_ID=w1:p2`、`HERDR_SOCKET_PATH=/tmp/hp/s.sock`、`HERDR_TAB_ID=w1:t2`、`HERDR_WORKSPACE_ID=w1` があった。`herdr pane current` は、pane の中で打つとその pane を返すが、pane の外で打つとフォーカス中の pane(w1:p1)を返す。このため、コマンドを打った pane を知る方法には使えない。

## Run 4: id の振り直しと、持ち主を確かめる get の応答

self-review の F-1(台帳の id が別の pane や workspace を指しうる)を受けて、同じ形の隔離サーバーで確かめた。

- 同じサーバーの中では、閉じた `w1` のあとに作った workspace は `w2` になり、閉じた id は使い回されなかった
- サーバーを起動し直しても、セッションの保存ファイル(`session.json`)が残っていれば、番号は続きから振られた(`w3`)
- 保存ファイルがない状態で起動し直すと、番号は `w1` から振り直された。このとき台帳に残った古い id は、ralph と関係のない workspace や pane を指しうる

get の応答は次のとおり(ralph が読まない項目は省いた)。

- `herdr pane get w1:p2`: `{"result":{"pane":{"pane_id":"w1:p2","tab_id":"w1:t2","workspace_id":"w1"},"type":"pane_info"}}`
- `herdr tab get w1:t2`: `{"result":{"tab":{"label":"seatZ","tab_id":"w1:t2","workspace_id":"w1"},"type":"tab_info"}}`
- `herdr workspace get w1`: `{"result":{"type":"workspace_info","workspace":{"label":"orgA","workspace_id":"w1"}}}`
- 見つからない id には、それぞれ `pane_not_found` / `tab_not_found` / `workspace_not_found` をコードに持つエラーが返った(`tab get` は rc=1 も確かめた)

ralph は座席の tab を座席 id、org の workspace を org_id の label で作る(`internal/org/spawn.go` の `TabCreate` と `WorkspaceCreate` の呼び出し)。そこで C-c を送る前と閉じる前に、`pane get` で pane のある tab と workspace を引き、tab の label が座席 id で workspace の label が org_id のときだけ C-c を送って閉じる。workspace は `workspace get` の label が org_id のときだけ閉じる(`internal/org/verbs.go` の `confirmSeatPane` と `confirmOrgWorkspace`)。

## Run 5: エージェントを起動したあとの label

2 回目の self-review の指摘(`herdr agent start` のあとも tab の label が保たれるか)を受けて、同じ形の隔離サーバーで確かめた。workspace を label `org-a`、tab を label `seat-1` で作り、その root pane(`w1:p2`)で次の 2 つを試した。

1. pane の中で端末タイトルを書き換える制御文字(`ESC ]0;…BEL` と `ESC ]2;…BEL`)を出した。そのあとも `herdr tab get w1:t2` の label は `seat-1`、`herdr workspace get w1` の label は `org-a` のままだった
2. `herdr agent start probe-agent --kind claude --pane w1:p2 --timeout 60000` で Claude Code を起動した。応答は `agent_started` で、`terminal_title` はシェルが展開した `claude --model opus ...` に変わった。起動の直後と 10 秒後のどちらでも、tab の label は `seat-1`、workspace の label は `org-a` のままだった

端末のタイトルは pane の `terminal_title` に入り、tab と workspace の label とは別に持たれている。Claude Code はフォルダの信頼の確認で止まった状態(`agent_status: blocked`)だったので、起動を終えた Claude Code が出すタイトルでは確かめていない。ただ、1 の結果から、タイトルの書き換えで label が変わることはないと見ている。codex の座席は試していない。

## Claude Code の C-c(文書で確認)

https://code.claude.com/docs/en/interactive-mode の Keyboard shortcuts によると、Ctrl+C は動いている処理を中断する。何も動いていないときは、1 回目で入力欄を消し、2 回目で終了する。このため、何もしていない Claude Code のセッションは C-c 1 回では終わらない。codex の C-c は確かめていない。

## 結論

- 何もしていない Claude Code の座席は C-c 1 回では終わらない。`herdr pane close` を打てば pane の中のプロセスは終わる
- 見つからない id には rc=1 と `pane_not_found` / `workspace_not_found` のコードが返る。ralph はこのコードを閉じ済みとして扱える
- 自分の pane と workspace は `HERDR_PANE_ID` / `HERDR_WORKSPACE_ID` で見分ける。`herdr pane current` は使わない
- herdr のセッションの保存ファイルが失われると id は `w1` から振り直されるので、台帳の id だけでは閉じない。閉じる前に tab と workspace の label で持ち主を確かめる
- Claude Code を起動しても、端末のタイトルが変わっても、tab と workspace の label は変わらない。label による持ち主の確認は、ふつうに spawn した座席で通る

確認のあと、Run 1〜4 は `herdr server stop` で、Run 5 は隔離サーバーのプロセスを終わらせて止め、どちらも `/tmp/hp` を削除した。
