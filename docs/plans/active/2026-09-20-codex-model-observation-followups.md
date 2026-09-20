# codex-model-observation-followups

- Status: Draft
- Owner: Claude Code
- Date: 2026-09-20
- Related request: #165(PR #174)の最終 cross-review(Codex、cycle 2、cap 到達)で出た 2 件。ユーザー判断で PR の Known gaps に記録し、この issue で直す。triage: `docs/reports/cross-review-triage-codex-effective-model-receipt.md`(AR-3、AR-4)
- Related issue: 173
- Type: fix
- Branch: fix/codex-model-observation-followups

## Objective

codex 座席の実効モデルの観測(#165)に残った 2 つの不具合を直す。(1) stop 時の観測が、実際に起動した spawn ではなく roster の最新状態の「指定モデル」と比較するため、誤った `honored=false` と警告が出得る。(2) 観測の 1 回の走査が期限も ctx の取り消しも見ないため、既定の 8 秒と `--timeout-ms` を超え得る。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | stop 時の比較対象 | `internal/org/verbs.go`(`codexSpawnCorrelation`、`observeStopModelReceipt`、`Stop`) | 対応付けに使う(dry-run でない最新の)`spawn_started` イベントから、`Model` / `Driver` / `Role` も取り出す。stop 時の receipt の `commanded_model` / `driver` / `role` と、不一致の判定はこの値で行う。「codex 座席かどうか」の判定も roster(`seat.Driver`)ではなくこの値で行う。`rejected` は拒否された要求の値を載せた state イベントなので、roster は起動中の座席と食い違い得る |
| 2 | 観測に ctx | `internal/org/codex_session.go`(`ObserveCodexEffectiveModel`) | 第 1 引数に `ctx context.Context` を足し、候補のファイルを開く前ごとに `ctx.Err()` を確認する。途中で打ち切られた走査は、それまでに一致を見つけていても採らない(残りの候補に 2 件目があり得るので「特定できた」と言えない)。打ち切りは not-found と ctx のエラーで返す。1 ファイルの読み取りは 4 MiB が上限なので、確認の粒度はファイル単位で足りる |
| 3 | spawn の待ち | `internal/org/spawn.go`(`observeCodexSpawnReceipt`) | 観測の上限(既定 8 秒)で区切った ctx を作って観測関数に渡す。走査の途中でも上限と `--timeout-ms`(親の ctx)が効く。`unknown` の理由の区別は保つ: 親の ctx が終わったら「spawn の timeout で打ち切り」、観測の上限なら従来の「見つからない」/「読めない」 |
| 4 | stop の観測 | `internal/org/verbs.go`(`observeStopModelReceipt`) | 1 回の観測にも同じ上限(`codexModelObserveTimeout`)の ctx を渡す。sessions ディレクトリが大きくても stop が長く止まらない。打ち切られたら何も追記しない(従来の「見つからない」と同じ扱い) |
| 5 | テスト | `internal/org/codex_session_test.go`、`spawn_test.go`、`verbs_test.go` | 下の Test plan |
| 6 | 文書 | `docs/tech-debt/README.md`(該当行があれば)、`docs/reports/`(pipeline の成果物) | 挙動の約束(最大 8 秒、stop は 1 回)は変わらないので、rule / recipe / skill の本文は原則そのまま。実装に合わせて直す箇所があれば sync-docs で拾う |

## Non-goals

- 走査の結果を待ちの間で覚えておく仕組み(読み直しの削減)。通常は spawn 以降に更新された数件しか読まず 1 回の走査は数十 ms で済むので、ctx の確認だけで上限は守れる。増分スキャンは状態が増える割に効果が小さい
- `rejected` イベントが roster の `Model` を上書きする挙動そのものの変更(org runtime の既存の仕様。今回は観測側が roster に依存しないようにする)
- receipts / manifest のスキーマ、CLI の警告文、doctor、claude 座席
- 1 ファイルの読み取りの途中での打ち切り(上限 4 MiB なので不要)

## Assumptions

- `spawn_started` イベントは `Model` / `Driver` / `Role` を持つ(`checkCapacityAndStart`)。古い manifest でも同じ(#165 より前からある項目)
- 観測関数の呼び出し元は `Spawn` と `Stop` の 2 つだけ(シグネチャの変更は package 内で完結する)

## Affected areas

- `internal/org/`(`codex_session.go`、`spawn.go`、`verbs.go` とテスト)
- 影響しないもの: `internal/cli/`、スキーマ、文書の約束

## Design decisions

- Critical forks: None。
- 打ち切られた走査は「見つかった」を返さない。安全側(`unknown`)に倒す。spawn の待ちは次の走査で、stop は次の機会がないので追記なしで終わる。
- stop の観測の上限は spawn と同じ定数を使う(新しい設定値を増やさない)。

## Acceptance criteria

- [ ] AC-1: spawn が `agent_started` の後で中断され、同じ org・seat への別モデルでの再試行が拒否された後の stop で、receipt の `commanded_model` は実際に起動した spawn の値になり、記録のモデルがそれと一致すれば `honored=true`、警告なし。修正前のコードではこのテストが落ちる(誤った `honored=false`)
- [ ] AC-2: roster の driver が拒否された要求のもの(claude)でも、起動した spawn が codex なら stop は観測する。逆(起動した spawn が claude、拒否された要求が codex)では観測しない
- [ ] AC-3: 取り消し済みの ctx を渡した観測は、一致する記録があっても not-found と ctx のエラーを返す(走査していないことの確認)
- [ ] AC-4: N 件目の候補の前で ctx が終わる場合、それ以降のファイルは開かれず、それまでに一致が 1 件あっても found を返さない
- [ ] AC-5: spawn の待ちは、走査の途中で観測の上限に達したら「見つからない」の `unknown`、親の ctx が終わったら「spawn の timeout で打ち切り」の `unknown` を書く。既存の理由の区別(読めない、特定できない、役割指示ファイルなし)は変わらない
- [ ] AC-6: stop の観測が打ち切られた場合、receipt は追記されず `model_observed=none`、stop は成功する
- [ ] AC-7: 既存のテスト(#165 の観測・配線・CLI・doctor)が変更なしの意図のまま通る。`./scripts/run-verify.sh` と `./scripts/run-test.sh` が green。PR 本文に `Closes #173`

## Implementation outline

1. Slice A: `codexSpawnCorrelation` の戻り値を構造体にして `Model` / `Driver` / `Role` を足し、`Stop` の判定と `observeStopModelReceipt` をそれに合わせる。AC-1 / AC-2 のテスト
2. Slice B: 観測関数に ctx を通し、`Spawn` と `Stop` から期限付きの ctx を渡す。AC-3〜AC-6 のテスト(回数で終わる fake の ctx を使い、時間に依存しない)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`gofmt -l internal/org`、`go vet ./internal/org/...`
- Spec compliance criteria to confirm: AC-1〜AC-7。Non-goals(スキーマ、CLI、doctor、文書の約束が不変)
- Documentation drift to check: `model-routing.md`・recipe・skill の「最大 8 秒」「stop はもう一度だけ探す」が引き続き正しいこと。tech-debt に #173 を指す行がないこと(あれば解消済みにする)
- Evidence to capture: verify / test のログ

## Test plan

- Unit tests: 観測関数と ctx(取り消し済み、N 件目で終了、正常)。`codexSpawnCorrelation` の新しい戻り値(最新の本物の spawn の値、dry-run と `rejected` に影響されない)
- Integration tests: AC-1 の順序(中断 → 拒否 → stop)、AC-2 の driver の食い違い、spawn の待ちの理由(AC-5)、stop の打ち切り(AC-6)
- Regression tests: #165 のテスト一式(リトライの接尾辞、同じ秒の再 spawn、数日後の session、dry-run の除外)
- Edge cases: `spawn_started` に `Model` がない古いイベント(空なら観測しない)。ctx が nil でないこと(呼び出し元は必ず渡す)
- Evidence to capture: `./scripts/run-test.sh`、race、`TMPDIR=/tmp`、反復

## Risks and mitigations

| リスク | 影響 | 対策 |
|---|---|---|
| シグネチャの変更で既存のテストの呼び出しが大量に変わる | 差分が読みにくい | 機械的な置換に限り、意図を変えない。テストの ctx は `context.Background()` |
| 打ち切りの扱いを誤り、途中の一致を採ってしまう | 別の座席のモデルを記録 | AC-4 のテストで固定 |
| stop に上限を入れたことで、正常な観測が打ち切られる | stop 時の観測漏れ | 上限は spawn と同じ 8 秒。1 回の走査は通常数十 ms |

## Rollout or rollback notes

- 追加の設定なし。revert すれば #165 の挙動に戻る

## Open questions

- なし

## Deviation notes

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 2 件ともコードで実在を確認済み(#165 の triage)
- [x] critical fork なし
- [x] AC は fake の ctx と manifest の fixture で決定的に確認できる
- [ ] Codex plan advisory
