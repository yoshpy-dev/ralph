# codex-model-observation-followups

- Status: In progress
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
| 2 | 観測に ctx | `internal/org/codex_session.go`(`ObserveCodexEffectiveModel`) | 第 1 引数に `ctx context.Context` を足す。**Codex advisory による改訂**: `ctx.Err()` の確認は 3 箇所に置く: 候補の収集(日付ディレクトリごと、エントリごと。列挙・Lstat・ソートは確認なしだと全件走る)、候補のファイルを開く前、記録の読み取りループ(1 行ごと。4 MiB は読む量の上限で、時間の上限ではない)。途中で打ち切られた走査は、それまでに一致を見つけていても採らない(残りの候補に 2 件目があり得るので「特定できた」と言えない)。打ち切りは not-found と ctx のエラーで返す。走査を最後まで終えた結果は、終わった時点で期限を過ぎていても有効(全候補を見た結果なので座席の特定は正しい)。中断は協調的で、同期ファイル I/O そのものは止められない: 1 回の読み取り(最大 256 KiB の 1 行)や 1 回のシステムコールの分だけ上限を超え得る。これを doc comment に明記する |
| 3 | spawn の待ち | `internal/org/spawn.go`(`observeCodexSpawnReceipt`) | 観測の上限(既定 8 秒)で区切った ctx を作って観測関数に渡す。走査の途中でも上限と `--timeout-ms`(親の ctx)が効く。`unknown` の理由の区別は保つ: 親の ctx が終わったら「spawn の timeout で打ち切り」、観測の上限なら従来の「見つからない」/「読めない」 |
| 4 | stop の観測 | `internal/org/verbs.go`(`observeStopModelReceipt`) | 1 回の観測にも同じ上限(`codexModelObserveTimeout`)の ctx を渡す。sessions ディレクトリが大きくても stop が長く止まらない。打ち切られたら何も追記しない(従来の「見つからない」と同じ扱い) |
| 5 | テスト | `internal/org/codex_session_test.go`、`spawn_test.go`、`verbs_test.go` | 下の Test plan |
| 6 | 文書 | `docs/tech-debt/README.md`(該当行があれば)、`docs/reports/`(pipeline の成果物) | 挙動の約束(最大 8 秒、stop は 1 回)は変わらないので、rule / recipe / skill の本文は原則そのまま。実装に合わせて直す箇所があれば sync-docs で拾う |

## Non-goals

- 走査の結果を待ちの間で覚えておく仕組み(読み直しの削減)。通常は spawn 以降に更新された数件しか読まず 1 回の走査は数十 ms で済むので、ctx の確認だけで上限は守れる。増分スキャンは状態が増える割に効果が小さい
- `rejected` イベントが roster の `Model` を上書きする挙動そのものの変更(org runtime の既存の仕様。今回は観測側が roster に依存しないようにする)
- receipts / manifest のスキーマ、CLI の警告文、doctor、claude 座席
- 同期ファイル I/O そのものの中断(1 回の読み取りやシステムコールは ctx で止められない。協調的な中断として、その分の超過は許容し明記する)

## Assumptions

- `spawn_started` イベントは `Model` / `Driver` / `Role` を持つ(`checkCapacityAndStart`)。古い manifest でも同じ(#165 より前からある項目)
- 観測関数の呼び出し元は `Spawn` と `Stop` の 2 つだけ(シグネチャの変更は package 内で完結する)

## Affected areas

- `internal/org/`(`codex_session.go`、`spawn.go`、`verbs.go` とテスト)
- 影響しないもの: `internal/cli/`、スキーマ、文書の約束

## Design decisions

- Critical forks: None。
- **Codex plan advisory(2026-09-20、MEDIUM 1、ユーザー決定: 対応案で plan を更新)**: ファイル単位の確認だけでは上限を守れない(候補の収集は確認の前に全件走る、4 MiB は時間の上限ではない、最後の候補の途中で期限が切れても found を返す)→ 確認を候補の収集と読み取りループにも置き、途中で打ち切った走査だけを無効にし、協調的な中断であることを明記する。
- 打ち切られた走査は「見つかった」を返さない。安全側(`unknown`)に倒す。spawn の待ちは次の走査で、stop は次の機会がないので追記なしで終わる。
- stop の観測の上限は spawn と同じ定数を使う(新しい設定値を増やさない)。

## Acceptance criteria

- [ ] AC-1: spawn が `agent_started` の後で中断され、同じ org・seat への別モデルでの再試行が拒否された後の stop で、receipt の `commanded_model` は実際に起動した spawn の値になり、記録のモデルがそれと一致すれば `honored=true`、警告なし。修正前のコードではこのテストが落ちる(誤った `honored=false`)
- [ ] AC-2: roster の driver が拒否された要求のもの(claude)でも、起動した spawn が codex なら stop は観測する。逆(起動した spawn が claude、拒否された要求が codex)では観測しない
- [ ] AC-3: 取り消し済みの ctx を渡した観測は、一致する記録があっても not-found と ctx のエラーを返す(走査していないことの確認)
- [ ] AC-4: N 件目の候補の前で ctx が終わる場合、それ以降のファイルは開かれず、それまでに一致が 1 件あっても found を返さない
- [ ] AC-4b: 候補の収集中(エントリの途中)に ctx が終わる場合、走査は打ち切られ not-found と ctx のエラーを返す。唯一の候補を読んでいる途中(行の途中)で ctx が終わる場合も、その記録が一致していても found を返さない。走査を最後まで終えた後に ctx が終わっていた場合は、結果をそのまま返す
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

- 2026-09-20 work: Slice A(1205775)と Slice B(b133022)は implementer に委譲。Slice A: `codexSpawnCorrelation` が `codexSpawnInfo`(開始時刻、役割指示ファイルのパス、`Model` / `Driver` / `Role`)を返し、`Stop` の「codex 座席か」の判定と receipt の比較対象はこの値を使う。AC-1 の順序(中断 → 別モデルでの再試行が拒否 → stop)のテストは、修正前のコードで誤った `honored=false` になることを確認済み。manifest の二重読み取りは残した(正しさを優先)。Slice B: 観測関数に ctx を通し、候補の収集(ディレクトリごと・エントリごと)、ファイルを開く前、1 行ごとに確認する。打ち切られた走査は found を返さない。spawn は観測の上限で区切った ctx、stop も同じ上限。テストは回数で終わる fake の ctx を使い、実時間に依存しない。逸脱: ファイルを開く前の確認だけを外してもテストは落ちず(1 行目の確認が同じ結果を出す)、両方外して初めて落ちることを implementer が確認
- 2026-09-20 work: orchestrator が HEAD 一致・porcelain 空・差分を確認。`observeCodexSpawnReceipt` の `lastErr` は「最後に完了した走査のエラー」という doc comment に反して、エラーなしで完了した走査が前の読み取りエラーを消していなかった(早い走査で読めず、後の走査で読めて一致もしなかった場合に「読めない」と報告する)ため、完了した走査は nil でも代入するよう修正(6b63b69、inline の 1 行)
- 2026-09-20 self-review cycle 1(`docs/reports/self-review-2026-09-20-codex-model-observation-followups.md`、b4b8075): Merge、MEDIUM 2 / LOW 5。2 つの修正が判定を実際に下している箇所に入っていること、「完了した走査は ctx が切れていても結果を返す」がそのとおり実装されていること、ctx のエラーが receipt の理由に漏れないことを確認。全件を同 cycle 内で修正(implementer、6564680。orchestrator が差分を確認): M1 spawn の待ちの doc comment をコードの規則に合わせた(親の ctx が先、それ以外は最後に完了した走査が決める)。M2 ctx の確認の位置が 1 行ごとの確認しかテストで固定されていなかった → ディレクトリごと・エントリごと・ファイルを開く前の確認それぞれに、消すと落ちるテストを追加(3 件とも一時的に消して失敗を確認)。L1 `stopped` イベントが roster の値を記録し続けるのは意図どおりであることをコメントで明記。L2 stop の manifest の読み取りを 1 回に統一(`seatFromEvents` を切り出し、同じ events を対応付けにも渡す。既存の stop のテストは変更なしで通る)。L3 / L4 指摘番号だけの引用と変更前の状態を語るコメントを挙動の説明に書き換え。L5 `StopResult.ModelReceipt` の doc を全 return に合わせた
- 2026-09-20 sync-docs cycle 1: self-review / verify 共通で指摘された既知の drift(stop の 1 回の観測自体が spawn と同じ最大 8 秒の上限を持つことがどこにも書かれていない)に、`.claude/rules/ralph/model-routing.md`(+ `templates/base/` 側)、`docs/recipes/codex-seat-permissions.md`(+ `templates/base/` 側)、`.claude/skills/org/SKILL.md`(+ `.agents/skills/org/`・`templates/base/` 2 面)の既存文に一節ずつ追記。spawn 側の「最大 8 秒」の文言、`rejected` 再試行修正の doc 追記、`docs/tech-debt/README.md`(#173 を指す行なし、既存行は不変)は変更不要と判断(詳細は sync-docs report)
- 2026-09-20 cycle 1 の結果: verify pass(638db39)、test pass(4300307。red/green 8 種類のうち 3 種類は壊しても落ちるテストがなく、tester が決定的なテストを追加して埋めた。ほかに 2 件、計 5 件追加)、sync-docs(7c95d3f。stop の 1 回の観測も spawn と同じ上限で区切られる旨を rule・recipe・skill に 1 節)
- 2026-09-20 cross-review cycle 1(`docs/reports/cross-review-triage-codex-model-observation-followups.md`、e5c0aed): Codex の指摘 1 件(P2)を ACTION_REQUIRED と判定。本番コードへの指摘はなし。テスト用の `testOrg()` が観測の上限を 1ms に固定しており、走査の途中でも上限が効くようになった今回の修正で、観測の成功を期待するテストが 1ms とファイル走査の競争になった(Codex の再現: race 検出つき 300 回中 7 回失敗。このマシンでは 300 回とも pass で、余裕がマシン依存)。ユーザー決定(AskUserQuestion): 修正して cycle 2/2 としてパイプラインを再実行
- 2026-09-20 work(cycle 2): Slice D は implementer に委譲(5e239b0、テスト 3 ファイル。本番コードの変更なし、テストの検証内容も不変)。観測の成功を期待するテストに十分な上限(`testCodexObserveGenerousBudget` = 30 秒。見つかればすぐ返るので遅くならない)を与えた。spawn が `unknown` で終わる前提の stop のテストは、spawn は小さい上限のまま、spawn の後・stop の前に `raiseCodexObserveBudgetForStop` で広げる。timeout を確かめるテストは小さい上限のまま。「観測しない / 追記しない」を確かめるテストのうち、上限が小さいと打ち切りでも通ってしまう 2 件(古い session を年齢で除外する、receipt の追記失敗)と、CLI の「警告が出ない」2 件も対象にした(前者は見つからない前提なので 30 秒ではなく 50ms)。CLI 側は `setCodexModelObserveTimeoutOverride` をテスト単位で使い、package 全体の既定は上げない。修正前のテストでの失敗はこのマシンでは再現できず(race 300 回、`GOMAXPROCS=1`、CLI スイートと並走でも pass)、走査に一時的に 3ms の遅延を入れて仕組みを確認した: 修正前の上限では Found のテストが `unknown` で落ち、修正後は通る(遅延は除去済み、`codex_session.go` に差分なし)。suite の所要時間は変わらず、1 秒を超えるテストはない。orchestrator が HEAD 一致・porcelain 空・差分を確認
- 2026-09-20 self-review cycle 2(同 report の Cycle 2 節、bb1e79b): HIGH を直せば Merge、HIGH 1 / LOW 3。cycle 1 の 7 件はすべて解消と確認。全件を同 cycle 内で修正(最終 cycle のため再確認は行わず、orchestrator が差分を確認)。C2-H1 `TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason` が 1ms の上限のまま残っていた。「読めない」の理由は 1 回の走査が完了して読み取りエラーになることが前提で、上限切れを待つが、その前に 1 回は走査が完了する必要がある、という第 3 の分類(Slice D の規則になかった)。implementer が上限を 50ms にして分類の規則に追記(711cd9d)。3ms の遅延注入では全テスト pass、60ms ではこのテストだけが落ちる(50ms の上限は 60ms の走査に耐えない)ことを確認し、200ms なら通ることも確かめたうえで判断を委ねてきたため、orchestrator が 200ms に広げた(a1afc0c。このテストは上限切れを待つので、所要時間が約 0.2 秒になるだけ)。C2-L1 網羅的に書き直した doc の一覧 2 箇所に receipts の読み取り失敗の枝を追記。C2-L2 50ms にした「古い session を年齢で除外する」テストのコメントに、打ち切られた場合は年齢を読まずに通ることと、年齢の規則を決定的に固定しているテストの名前を追記。C2-L3 plan の AC のチェックは cycle 2 の verify の後に反映する

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 2 件ともコードで実在を確認済み(#165 の triage)
- [x] critical fork なし
- [x] AC は fake の ctx と manifest の fixture で決定的に確認できる
- [x] Codex plan advisory(1 件、対応案で plan を更新)
