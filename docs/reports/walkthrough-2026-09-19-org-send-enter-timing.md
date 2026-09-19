# Walkthrough: org-send-enter-timing

- Date: 2026-09-19
- Plan: docs/plans/archive/2026-09-19-org-send-enter-timing.md(PR 作成時に active から移動)
- Issue: #163(Closes)
- Branch: fix/org-send-enter-timing(base: main @ f006200)
- 差分規模: 24 files / 約 +3,100 −60。Go は 7 ファイル(+1,805 −46): `internal/org/verbs.go`(+379)、`internal/cli/org.go`(+156)、`internal/org/spawn.go`(+10)、残りはテスト(`verbs_test.go` +585、`org_test.go` +536、`spawn_test.go` +74、`send_defaults_sync_test.go` +111)。ほかは skill 4 面、recipe 2 コピー、evidence 2 件、tech-debt 1 行、plan、pipeline レポート 2 cycle 分

## 何が変わったか

`ralph org send` は本文を入力した直後に Enter を送っていた。idle の codex 座席では、その Enter が貼り付けの処理に飲まれて本文が入力欄に残ることがあった(実機で 5 回中 3 回。claude 座席では 0 回)。

1. 本文の入力と Enter の間に 750ms 待つ(`--enter-delay-ms` で変更可。0 は既定値、負の値は拒否)。実機では codex・claude とも 5 / 5 で submit された。
2. Enter の後、`herdr agent wait` で座席が idle / done を離れた(working か blocked になった)ことを最大 3 秒確認する。確認できなくても失敗にはしない(ごく短いターンは待ちが始まる前に終わり得る)。`sent` イベントの Details に `submit_unconfirmed=true` を付け、stderr に注意を出す(exit 0)。
3. ralph は Enter を再送しない。codex の承認ダイアログは「Yes, proceed」が選択済みで Enter で確定するため、submit 済みの座席に盲目的な Enter を送ると承認してしまう(実機で確認。ユーザー決定)。
4. `--timeout-ms` の残りが Enter 前の待ちを賄えないときは、本文を入力する前にエラーで終わる。入力してから時間切れになると入力欄に残骸が残り、送り直しで 2 通が連結されるため。
5. 失敗したときは「どこまで進んだか」を 5 状態で返し(`SendResult.Progress`)、CLI が状態ごとに案内を出す。herdr の呼び出しは期限で打ち切られると、本文や Enter が届いていてもエラーを返す。その場合は「届いたか分からない」と伝え、「未送信」とは言わない。

## 読む順番

1. `internal/org/verbs.go` — `SendProgress`(5 状態と、結果が不明になる理由)→ `SendResult` → `Send`。`Send` の流れ: 検証 → dry-run → 座席の検索 → idle / done の待ち → 予算チェック(入力前に fail-closed)→ `PaneSendText` → `waitBeforeEnter` → `PaneSendKeys("Enter")`(1 箇所だけ)→ `confirmSubmitted` → `sent` イベント。補助は `sendEnterDelay` / `sendSubmitConfirmTimeout` / `waitBeforeEnter` / `confirmSubmitted` / `capMSToContext`(herdr は timeout 0 を無期限と解釈するので最小 1ms)
2. `internal/org/spawn.go` — テスト用の上書き `Org.SendEnterDelay` / `Org.SendSubmitConfirmTimeout`(`AgentStartRetryInterval` と同じ形)
3. `internal/cli/org.go` — `newOrgSendCmd`(フラグ、`result.Progress` の switch と 4 つの案内、exit 0 の未確認警告)、`orgReadCommandHint` / `shellQuoteIfNeeded`(案内に出す `ralph org read` の組み立て。`--state-dir` は明示されたときだけ、解決済みの絶対パスで付ける)
4. `internal/org/verbs_test.go` / `internal/org/spawn_test.go`(fake herdr: `agentWaitErrs`、`paneSendTextErr`、`paneSendTextDelay`、`paneSendKeysErr`)— 呼び出し順の固定、Enter が 1 回であること、5 状態それぞれの return
5. `internal/cli/org_test.go` — 実プロセスの herdr stub 経由。案内ごとに 1 テスト、`--state-dir` の有無、`orgReadCommandHint` の表
6. `internal/org/send_defaults_sync_test.go` — skill 4 面と recipe 2 コピーに書かれた既定値(750)が定数と同じ行で一致することを確認
7. `docs/evidence/org-send-enter-timing-2026-09-19.md` — 実機の結果と環境面の記録
8. skill の `send` の行、recipe の該当段落、tech-debt の行

## 失敗時の 5 状態と案内

| `Progress` | どの return か | CLI の案内 |
|---|---|---|
| nothing-sent | 検証、dry-run、座席なし、idle / done の待ちの失敗、予算チェック | なし(そのまま送り直せる) |
| text-unacknowledged | `PaneSendText` がエラー | 本文が届いたか分からない。送り直す前に pane を確認する |
| text-typed | Enter 前の待ちの途中で期限切れ(Enter は未試行) | 本文は入力済みで未送信。pane を確認し、消去か submit をしてから送り直す |
| enter-unacknowledged | `PaneSendKeys` がエラー | submit されたか分からない。先に pane を読み、入力欄に本文が残っているときだけ submit か消去。それ以外なら Enter を押さず、送り直さない |
| enter-pressed | `sent` イベントの記録エラー(と成功時) | Enter は送信済みで、記録だけできなかった。送り直さない |

## コミット単位

| SHA | 内容 |
|---|---|
| c40a83e, 85af422 | plan。Codex plan advisory の HIGH(Enter の自動再送は承認ダイアログを確定し得る)を受け、ユーザー決定で再送を不採用に |
| c990363 | 待ち、submit の確認、`submit_unconfirmed=true`、`--enter-delay-ms`、未確認警告 |
| 4c2d134, 77371a8 | 実機確認の evidence、skill / recipe |
| 7813c2a | self-review cycle 1 の 7 件(入力前の予算チェック、入力後の失敗の報告、`--timeout-ms` のヘルプ、既定値の一本化と同期テスト ほか) |
| bcc1eb2, f7ea44d | 再確認の 4 件(Enter 送信後の記録失敗を「未送信」と案内していた問題、期限切れ経路の決定的なテスト ほか)と、エラー文言の「submitted」を「Enter was pressed」に修正 |
| c3ec8a1, 3e35c4d | テストの追加(待ち時間の下限、idle / done の待ちの失敗、send-text の失敗) |
| 35a3594, f9f0d82 | cross-review cycle 1 の 2 件: 2 つの bool を 5 状態の `Progress` に置き換え、`ralph org read` の案内に `--state-dir` を引き継ぐ。skill / recipe の文言 |
| 646fb4b, 8a78aca | self-review cycle 2 の 6 件(CLI のタイミングテストの余裕、未確認警告の明示、テスト名、コメントの整理、文書) |
| その他 | plan の進捗・逸脱記録、各レポート(cycle 1・2)、insight events、tech-debt |

## 設計判断(plan Design decisions より)

- Enter は 1 回だけ。確認できなくても再送しない(ユーザー決定)。人への案内も同じ基準で書く: どの案内も「無条件に Enter を押せ」とは言わない。
- submit の確認は herdr の状態だけで行い、pane の文字列は読まない(TUI の描画に依存して壊れやすいため)。working と blocked のどちらも「submit された」とみなす(blocked は承認ダイアログなどの入力待ち)。確認は「座席が idle / done を離れた」ことしか観測しておらず、自分の Enter が原因だという証明ではない。
- 既定値 750ms は実機の計測で決めた。`ralph.toml` のキーは足さない(フラグで足りる)。
- 呼び出しがエラーで返っても、pane に届いていないとは限らない(`exec.CommandContext` が期限で herdr の CLI を止めるため)。結果が不明な状態を「失敗」と同じに扱わない。

## 注意して見てほしい点

- `Send` の `PaneSendKeys` の呼び出しが 1 箇所だけで、pane への呼び出しに再試行がないこと。
- 5 状態の doc comment、エラー文言、CLI の案内、skill / recipe の 4 者が、`Send` が観測した範囲を超えていないこと。
- `newOrgSendCmd` は state dir の解決を `newOrgRuntime` から取り出して自前で行う(`watch` と同じ形)。既定値と環境変数の場合の挙動は変わらない。
- CLI のテスト 1 件(`TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr`)は実プロセスの stub を 1.1 秒 sleep させるため、1 回あたり約 2.8 秒かかる。余裕は stub の起動コストが増えるほど広がる向きに設定してある。

## Known limitations

- 実機確認は AR-1 / AR-2 の前に行った。5 状態化と `--state-dir` の引き継ぎは失敗時の報告だけを変えており、待ち・確認・再送なしの仕組みは同じなので再計測はしていない。結果が不明になる 2 状態は fake と stub でのみ再現している。
- 未テストの枝(tech-debt に記録): `Send` 内の manifest 読み取りエラー、予算チェックのエラー文言で残り時間を 0 に切り上げる枝、750ms という値そのものが実行時に使われること。
- CLI のエラー接頭辞が `org: send: org: send:` と二重になるのは main に以前からある挙動で、本 PR では触っていない。
- 案内文は `ralph org read` のコマンドを単一引用符で囲んで出すため、`--state-dir` のパスに引用が必要な文字があると引用符が入れ子になる(内側のコマンドはそのままコピーすれば動く)。
- 実機確認の環境面の副作用は evidence に記録した(ユーザーの `~/.claude.json` にスクラッチディレクトリの信頼記録 1 件、1 回目の分離不足でユーザーの `herdr-server.log` に 8 行)。
