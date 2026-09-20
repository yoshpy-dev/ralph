# Walkthrough: codex-effective-model-receipt

- Date: 2026-09-20
- Plan: docs/plans/archive/2026-09-20-codex-effective-model-receipt.md(PR 作成時に active から移動)
- Issue: #165(Closes)
- Branch: feat/codex-effective-model-receipt(base: main @ 642214d)
- 差分規模: 約 30 files / +5,800 −70。Go は 11 ファイル(+4,196 −65)で、うち実装は 5 ファイル: `internal/org/codex_session.go`(新規 488 行)、`internal/org/spawn.go`(+359)、`internal/org/verbs.go`(+248)、`internal/cli/doctor_codex_models.go`(+173)、`internal/cli/org.go`(+51)。残りはテスト(約 2,900 行)。ほかは rule 2 面、spec、recipe 2 コピー、skill 4 面、evidence 2 件、tech-debt 1 行、plan、pipeline レポート 2 cycle 分

## 何が変わったか

`ralph org spawn` は codex 座席に `--model` を渡すが、codex は退役予定のモデルを自動で移行先に切り替えることがある(#155 の Run E1: `gpt-5.5` 指定で `gpt-5.6-sol`)。model receipts は対話座席を常に `honored=unknown` で記録していたので、これを拾えなかった。

1. **観測**: codex の session 記録(`$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl`、`CODEX_HOME` が空なら `~/.codex`)から、その spawn の実効モデル(`turn_context.model`)を読む。pane の文字列は読まない。
2. **spawn**: 役割指示ファイルで起動した codex 座席は、`spawned` の後に最大 8 秒(500ms 間隔)記録を探し、receipt を `honored=true` / `false`(`reported_effective_model` と理由付き)/ `unknown` で書く。
3. **stop**: その spawn にまだ観測済みの receipt がなければ 1 回だけ探し、見つかれば receipt を 1 件追記する。`stopped` の Details に `model_observed=true|false|none`。
4. **CLI**: `honored=false` かつ実効モデルが入った receipt を書いたときだけ stderr に警告(exit code は不変)。
5. **doctor**: codex スラッグ Check が `models_cache.json` の `upgrade` を読み、pool のスラッグの退役予定を `retiring: gpt-5.5 -> gpt-5.6-sol on 2026-10-14` の形で出す(info)。

## 読む順番

1. `internal/org/codex_session.go` — 上から順に読める。上限の定数(200 ファイル、4 MiB / ファイル、256 KiB / 行、32 ディレクトリ)→ `CodexSessionsDir` → `ObserveCodexEffectiveModel`(found / not-found / ambiguous)→ `codexRolloutCandidates`(Lstat で通常ファイルだけ、更新時刻で足切り)→ `codexSessionDateDirs`(spawn 日の前日から `until` の翌日まで)→ `scanRolloutRecord`(`session_meta` の開始時刻、user メッセージの指示文、最初の `turn_context`)
2. `internal/org/spawn.go` — `Org` の seam 3 つ(`CodexSessionsDir` / `CodexModelObserveTimeout` / `CodexModelObserveInterval`)、`checkCapacityAndStart` が返す spawn 開始時刻、`observeCodexSpawnReceipt`(待ちのループと `unknown` の理由 3 種)、`codexFoundReceipt`、`reject()` の `ModelReceipt`、`PromptFilePointer`、`codexPromptFileDetailsPrefix` と接尾辞の定数
3. `internal/org/verbs.go` — `Stop` の観測部分、`codexSpawnCorrelation`(dry-run を除いた最新の `spawn_started` と役割指示ファイルのパス)、`promptPathFromAgentStartedDetails`、`hasObservedCodexReceiptAfter`、`observeStopModelReceipt`
4. `internal/cli/org.go` — `printCodexModelMismatchWarning` と、テスト用の package 変数 3 つ(`internal/cli/main_test.go` の `TestMain` が固定)
5. `internal/cli/doctor_codex_models.go` — `parseCodexModelUpgrade`(`upgrade` は `json.RawMessage` で受ける)、`codexRetirementClause`、`checkCodexModelSlugs` の outcome 一覧
6. テスト: `codex_session_test.go`(観測関数)、`spawn_test.go` / `verbs_test.go`(配線。`testOrg()` が観測先を空の一時ディレクトリに固定)、`internal/cli/org_test.go`(herdr stub 経由の警告)、`internal/cli/doctor_org_test.go`(退役予定)
7. `docs/evidence/codex-effective-model-receipt-2026-09-20.md` — 実記録 5 件での確認、記録の形の実測、ディレクトリの根拠
8. 文書: `.claude/rules/ralph/model-routing.md`(+ template)、spec FR-9、recipe、skill、tech-debt の行

## 座席の特定

1 つの記録がその spawn のものと認められる条件(すべて):

| 条件 | 理由 |
|---|---|
| 通常ファイルで名前が `rollout-*.jsonl` | symlink・FIFO は開かない(FIFO で固まらない) |
| `session_meta.payload.timestamp` が spawn 開始(秒に切り捨て)以降 | 役割指示ファイルのパスは org と seat が同じなら再 spawn でも同じ。古い session が後から更新されても採らない |
| user メッセージが、ralph が座席に渡す指示文全体(`PromptFilePointer(<パス>)`)を含む | パスを引用しただけの別のメッセージでは一致させない |
| `turn_context` に model がある | 最初の `turn_context` を実効モデルとする |

該当が 2 件以上なら ambiguous(推測しない)。役割指示ファイルのない座席(雛形のない role に短いプロンプトを渡した場合)は手掛かりがないので観測せず、待たずに理由付きの `unknown`。

## コミット単位

| SHA | 内容 |
|---|---|
| 6fd45c6, d1d9f76 | plan。観測の時点はユーザー決定(spawn 直後 + stop 時)。Codex plan advisory の 3 件(古い session の混入、stop 時の条件、役割指示ファイルのない座席)を反映 |
| 78692f4 | 観測関数とテスト |
| d2bf6e4 | `Spawn` / `Stop` への配線、`ModelReceipt`、CLI の警告、テスト用の seam |
| 7963da0, 80a50c2 | doctor の退役予定 |
| a660122 | 文書と実記録の evidence |
| 26b03ce, 9961b8d | self-review cycle 1 の 11 件(dry-run のイベントの除外、`ModelReceipt` の意味の統一、doctor の固定文、日付ディレクトリ、`unknown` の理由、指示文全体での一致 ほか) |
| 771e446, a7180ca | テストの追加 |
| d24a030 | cross-review cycle 1 の 2 件: 起動のリトライ後に stop がパスを読めない、同じ秒の再 spawn で前の receipt が観測済み扱いになる |
| 53f6b16, 2af0bce | self-review cycle 2 の 6 件(`until` の追加、`reject()` の `ModelReceipt`、`PromptFilePointer` の export ほか) |
| その他 | plan の進捗・逸脱記録、各レポート(cycle 1・2)、insight events、sync-docs |

## 設計判断(plan Design decisions より)

- 観測の時点は「spawn 直後 + stop 時」(ユーザー決定)。lead が早く気付け、spawn 時に取れなかった座席も拾える。
- 観測元は codex の session 記録。pane のステータス行は使わない。`[notice.model_migrations]` からの推測もしない(キーがなくても移行が起きた実例がある)。
- 記録には会話の本文が入っている。取り出すのは model だけで、エラーにも本文を含めない。呼び出し側は観測関数のエラーの文言を出力しない。
- 理由の文言は観測した事実だけを書く。不一致の原因(退役モデルの自動移行、config の上書き)は可能性として添える。
- receipts と manifest のスキーマは変えない。stop 時は receipt を追記する(1 座席に `unknown` と観測済みが 1 件ずつ残り得る)。
- 観測の失敗で spawn も stop も失敗させない。

## 注意して見てほしい点

- 記録の内容が receipt・manifest・エラー・stderr・テストのログのどこにも出ないこと(`scanRolloutRecord` と、2 つの呼び出し側)。
- spawn の待ち(最大 8 秒)が manifest のロックの外で行われ、ctx の取り消しで終わること。
- `hasObservedCodexReceiptAfter` の厳密な比較: 秒精度で同じ時刻の receipt は数えない。外れた場合の代償は stop 時の重複 receipt 1 件。
- `agent_started prompt_file=<パス> agent_start_retries=N` の接尾辞は末尾の項目でなければならない(コメントとテストで固定)。
- 既定の `[org].model_pool` に退役予定の `gpt-5.5`(2026-10-14)が入っているため、既定の設定でも doctor の info が出る。既定値の見直しは #156 の範囲。

## Known limitations

- 新しい座席を起動しての end-to-end は行っていない。観測関数は実記録 5 件で確認し、`Spawn` / `Stop` / CLI は fixture と herdr stub のテストで確認した。
- codex の記録の形は内部仕様で、codex-cli 0.154.0 でしか測っていない。形が変わると観測は無言で `unknown` に戻る。座席の特定は ralph が渡す指示文の文言にも依存する(tech-debt に記録)。
- 退役ダイアログの表示中に記録が作られるかは直接確認していない。ダイアログが出た実記録の時刻から、記録はダイアログに答えた後に作られると推定している。stop は spawn 日から stop 日まで(上限 32 日分)を探す。
- ralph と座席の codex が別の `CODEX_HOME` / `HOME` を見ている場合(herdr server を別の HOME で起動した場合)は観測できず `unknown` になる。ralph 側で `CODEX_HOME` を指定すれば観測できる。
- claude 座席は観測しない(従来どおり `unknown`)。
