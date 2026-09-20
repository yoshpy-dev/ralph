# Walkthrough: codex-model-observation-followups

- Date: 2026-09-20
- Plan: docs/plans/archive/2026-09-20-codex-model-observation-followups.md(PR 作成時に active から移動)
- Issue: #173(Closes)。#165 / PR #174 の最終 cross-review の残件
- Branch: fix/codex-model-observation-followups(base: main @ 3d355bd)
- 差分規模: 約 23 files / +2,500 −190(うち pipeline のレポートと plan が約 1,000 行)。Go の実装は `internal/org/` の 3 ファイル(`verbs.go`、`codex_session.go`、`spawn.go`)。残りの Go はテスト(`internal/org/` の 3 ファイルと `internal/cli/org_test.go`)。ほかは rule 2 面、recipe 2 コピー、skill 4 面に各 1 節、plan、pipeline レポート

## 何が変わったか

PR #174 は、codex 座席の実効モデルを codex の session 記録から観測して model receipts に記録する(spawn 時に最大 8 秒、stop 時に未観測なら 1 回)。その最終 cross-review で出た 2 件を直す。

1. **stop 時の比較対象**: stop は receipt の指定モデル・driver・role と「codex 座席かどうか」を roster の最新状態から取っていた。`rejected` は拒否された要求の値を載せた state イベントなので、「spawn が途中で中断 → 別のモデルでの再試行が拒否 → stop」の順で、指定どおり動いている座席に誤った `honored=false` と警告が出た。これを、対応付けに使う(dry-run でない最新の)`spawn_started` イベントの値に変えた。
2. **観測の期限と取り消し**: 観測関数は 1 回の走査の途中で期限も ctx の取り消しも見なかった(約 3.8 MiB の記録 50 件で、8 秒の上限に対し 9.45 秒)。ctx を通し、候補の収集(日付ディレクトリごと・エントリごと)、ファイルを開く前、1 行ごとに確認する。途中で打ち切った走査は、一致を見ていても結果を返さない(残りの候補に 2 件目があり得る)。最後まで終えた走査はそのまま返す。spawn の観測は観測の上限と `--timeout-ms` の両方で、stop の 1 回の観測も同じ上限で区切られる。

## 読む順番

1. `internal/org/verbs.go` — `Stop`(manifest を 1 回読み、`seatFromEvents` と対応付けに同じ events を渡す)→ `codexSpawnInfo` / `codexSpawnCorrelation`(`Model` / `Driver` / `Role` も返す)→ `observeStopModelReceipt`(戻り値は receipt、観測できたか、codex の spawn か。`model_observed=` のトークンは codex の spawn の本物の stop でだけ付く)。`stopped` イベントが roster の値を記録し続けるのは意図どおり(追記箇所のコメント)
2. `internal/org/codex_session.go` — `ObserveCodexEffectiveModel(ctx, …)` の doc comment(中断は協調的で、1 回の読み取りやシステムコールの分だけ超え得る)→ `codexRolloutCandidates(ctx, …)` → `scanRolloutRecord(ctx, …)` → `isCtxDoneErr`
3. `internal/org/spawn.go` — `observeCodexSpawnReceipt`: 観測の上限で区切った ctx を作る。`unknown` の理由は、親の ctx が終わっていれば「spawn の timeout で打ち切り」、そうでなければ最後に完了した走査が「読めない」か「見つからない」かを決める
4. テスト: `verbs_test.go`(中断 → 拒否 → stop、driver の食い違い 2 方向、`spawn_started` のない座席)、`codex_session_test.go`(回数で終わる fake の ctx `countingCtx`。4 つの確認の位置をそれぞれ固定するテスト、打ち切り後に found を返さない、完了した走査は返す)、`spawn_test.go`(理由の区別、後の完了した走査が前の読み取りエラーを消す、親の取り消しが優先)

## コミット単位

| SHA | 内容 |
|---|---|
| 099e028, e64725d | plan。Codex plan advisory の 1 件(ファイル単位の確認だけでは上限を守れない)を反映 |
| 1205775 | stop 時の比較対象を、起動した spawn の値に |
| b133022, 6b63b69 | 観測関数に ctx。spawn の待ちで、完了した走査が前の読み取りエラーを消すように |
| 6564680 | self-review の 7 件(doc comment、確認の位置を固定するテスト 3 件、manifest の読み取りを 1 回に、コメントの整理) |
| 4300307 | tester が足したテスト 5 件(壊しても落ちなかった 3 種類の変更を検出するテストを含む) |
| 7c95d3f | 文書: stop の観測も同じ上限で区切られる旨を rule・recipe・skill に 1 節 |
| 5e239b0, 711cd9d, a1afc0c | cross-review cycle 1 の 1 件と self-review cycle 2 の指摘(テストだけ): 走査の途中でも上限が効くようになったため、テスト用の 1ms の上限が「観測の成功を期待するテスト」と競争していた。成功を期待するテストには 30 秒(見つかればすぐ返る)、spawn が `unknown` で終わる前提の stop のテストは spawn の後に広げる、上限切れの前に 1 回の走査の完了が必要なテストは 200ms、timeout を確かめるテストは小さいまま。本番コードは `verbs.go` のコメントだけ |
| その他 | plan の進捗・逸脱記録、各レポート、insight events |

## 注意して見てほしい点

- `Stop` の観測まわりが roster の値を一切使わないこと(`stopped` イベントだけが roster の値を記録する)。
- 打ち切られた走査が found を返す経路がないこと、走査の完了後に余計な ctx の確認がないこと。
- ctx のエラーや読み取りエラーの文言が、receipt の理由・manifest・stderr に出ないこと。
- 既存の挙動(receipts / manifest のスキーマ、CLI の警告、doctor、claude 座席)は変えていない。`internal/cli/` の差分はテスト(`org_test.go`)だけ。
- テストの上限の 3 分類(`internal/org/spawn_test.go` の `testCodexObserveGenerousBudget` のコメント)。走査に 3 / 10 / 25 / 60ms の遅延を注入しても全テストが通ることを tester が確認した。

## Known limitations

- 中断は協調的で、同期ファイル I/O そのものは止められない。1 回の読み取り(最大 256 KiB の 1 行)や 1 回のシステムコール、`ReadDir` 1 回の分だけ上限を超え得る。
- 走査のたびに、spawn 以降に始まった一致しない記録を読み直す(増分スキャンはしない)。通常は数件・数十 ms で、上限は ctx で守られる。
- `rejected` イベントが roster の Model / Driver / Role を上書きする挙動そのものは変えていない(org runtime の既存の仕様)。
- 新しい座席を起動しての end-to-end は行っていない(fixture と fake の ctx のテストで確認)。
