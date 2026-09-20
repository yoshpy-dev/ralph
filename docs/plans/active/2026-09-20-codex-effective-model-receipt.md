# codex-effective-model-receipt

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-20
- Related request: #155 の実機検証(`docs/evidence/codex-seat-permissions-2026-09-18.md` P5、Run E1)で、`--model gpt-5.5` を渡した codex 座席の実効モデルが `gpt-5.6-sol` になった(codex 側の退役モデルの自動移行)。ralph の model receipts は対話座席を常に `honored=unknown`(`interactive session; effective model not yet observable`)で記録しており、このケースを拾えていない(issue #165)
- Related issue: 165
- Type: feat
- Branch: feat/codex-effective-model-receipt

## Objective

codex 座席の実効モデルを観測して model receipts に `honored=true|false` と `reported_effective_model` を記録する。指定したモデルで動いていない座席(退役モデルの自動移行など)を、lead が spawn の時点で、遅くとも stop の時点で知れるようにする。spec の受け入れ条件「codex 座席の receipts の effective_model が一致する(honored=true)」(`docs/specs/2026-08-01-org-runtime.md`)は現状満たされておらず、これを実装で満たす。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | 観測 | `internal/org/codex_session.go`(新規) | codex の session 記録(`$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl`、`CODEX_HOME` が空なら `$HOME/.codex`)から、座席の実効モデルを読む。座席との対応付けは、user メッセージ(`response_item` / `message` / `role=user`)の本文に座席の役割指示ファイルの絶対パス(`<state-dir>/prompts/<org>_<seat>.md`)が含まれること。実効モデルはその記録の最初の `turn_context` の `model`。**Codex advisory による改訂**: 記録の 1 行目 `session_meta` の開始時刻(`timestamp`)が、その座席の spawn 開始時刻以降であることも条件にする(役割指示ファイルのパスは org と seat が同じなら再 spawn でも同じで、stop は Ctrl-C を送るだけなので、生き残った古い session が後から更新され得る。ファイルの更新時刻では区別できない)。条件を満たす記録が 2 件以上あれば、どれも採らず「特定できない」とする。返すのは model だけで、記録の内容はログ・エラー・receipts のどこにも出さない |
| 2 | 読む範囲の制限 | 同上 | spawn 開始時刻以降に更新された通常ファイルだけを読む(symlink・FIFO・ディレクトリは開かない。更新時刻は読む量を減らすための足切りで、座席の特定には使わない)。日付ディレクトリは spawn の前日から当日まで。ファイル数と 1 ファイルあたりの読み取りバイト数に上限を置く(`turn_context` と役割指示のメッセージは記録の先頭 10 行以内に出る。実測 5 件)。1 行が 64 KiB を超えても読める方法で読む(実測の最大は約 44 KiB) |
| 3 | spawn | `internal/org/spawn.go` | driver が codex で dry-run でなく、最初のプロンプトを役割指示ファイルで渡した spawn は、`spawned` イベントの後に最大 8 秒(500ms 間隔)だけ記録を探す。見つかれば spawn の receipt を `honored=true`(一致)か `honored=false`(不一致、`reported_effective_model` と理由付き)で書く。見つからなければ従来どおり `unknown`(理由は「codex の session 記録がまだない」)。役割指示ファイルのない座席(テンプレートのない role で、プロンプトが短く inline で渡されたか空の場合)は対応付けの手掛かりがないので、待たずに理由付きの `unknown` を書く(**Codex advisory による改訂**。プロンプトに相関 ID を足す案は、モデルに見える内容が全 codex 座席で変わり、空のプロンプトの座席がターンを始めてしまうので採らない)。観測の失敗で spawn を失敗させない。上限と間隔は `Org` のフィールドで上書きできる(テスト用。`AgentStartRetryInterval` と同じ形) |
| 4 | stop | `internal/org/verbs.go`(`Stop`) | driver が codex で dry-run でない stop は、その座席の spawn 開始時刻以降に `reported_effective_model` の入った receipt が 1 件もないときだけ(**Codex advisory による改訂**。receipt が 1 件もない場合を含む: spawn の待ちの途中でプロセスが落ちると receipt が残らない。拒否や dry-run の receipt は `honored=false` でもモデルを持たないので、観測済みとはみなさない)、1 回(待たずに)記録を探し、見つかれば receipt を 1 件追記する。見つからなければ何も追記しない。観測の失敗で stop を失敗させない。`stopped` イベントの Details に観測結果の短い注記を足す |
| 5 | CLI | `internal/cli/org.go`(`spawn` / `start`、`stop`) | `honored=false` の receipt を書いたときは stderr に警告を出す(指定したモデル、codex が報告したモデル、退役モデルの自動移行か config の上書きの可能性、確認先の `ralph doctor`)。exit code は変えない |
| 6 | doctor | `internal/cli/doctor_codex_models.go`(`checkCodexModelSlugs`) | `models_cache.json` の各モデルの `upgrade`(`model` / `retirement_at`)を読み、`[org].model_pool` の codex スラッグに退役予定があれば Detail に「<スラッグ> は <日付> に退役し、codex は <移行先> に切り替える」を出す。スラッグが cache にない場合の warn が優先。退役予定だけなら info |
| 7 | テスト | `internal/org/codex_session_test.go`(新規)、`spawn_test.go`、`verbs_test.go`、`internal/cli/org_test.go`、`internal/cli/doctor_codex_models_test.go`、`internal/org` / `internal/cli` の `TestMain` 相当 | 実記録の形(`session_meta` / `turn_context` / `response_item`)を最小限に写した fixture。テストが実ホームの `~/.codex` を読まないよう、観測先のディレクトリを seam で固定する |
| 8 | 実記録での確認 | `docs/evidence/codex-effective-model-receipt-2026-09-20.md`(新規) | #155 / #163 の実機実行で偽 HOME に残っている session 記録 5 件(うち 1 件が `gpt-5.5` → `gpt-5.6-sol`)に対して観測を実行し、model だけが取れることを記録する。新しい座席の起動はしない |
| 9 | 文書 | `.claude/rules/ralph/model-routing.md`(+ template があれば)、`docs/specs/2026-08-01-org-runtime.md`、`docs/recipes/codex-seat-permissions.md` + template、`.claude/skills/org/SKILL.md` + 3 ミラー、`docs/evidence/codex-seat-permissions-2026-09-18.md`(P5 に追記 1 行) | receipts の codex 座席の観測元、`honored=false` の意味、1 座席に receipt が複数件になり得ること(spawn の unknown + stop の観測)、`CODEX_HOME` が座席側と違うと観測できないこと |

## Non-goals

- claude 座席の実効モデルの観測(今回は codex だけ。claude は従来どおり `unknown`)
- pane の文字列(ステータス行)の読み取り。TUI の描画に依存するため使わない
- 退役ダイアログの自動操作、モデルの自動差し替え、`[org].model_pool` の既定値の変更(#156 の範囲)
- effort(`turn_context.effort`)の記録。receipts のスキーマは変えない
- `ralph insights` の集計方法の変更(1 座席の receipt が複数件でも、true / false / unknown をそのまま数える)
- codex の session 記録の場所を `ralph.toml` で設定できるようにすること(`CODEX_HOME` で足りる)

## Assumptions

- codex CLI 0.154.0 の session 記録の形(実測): 1 行 1 JSON、`type` と `payload`。`turn_context.payload.model` が実効モデル。役割指示ファイルのパスは、ralph が最初のプロンプトをファイルで渡したとき(改行を含むか 200 文字を超える場合。標準の 4 role は常に該当)に user メッセージに含まれる(`promptFilePointer`)。短いプロンプトは inline で渡され、パスは含まれない
- 記録の形は codex の内部仕様で、予告なく変わり得る。読めない・見つからない場合は `unknown` に倒し、spawn / stop は失敗させない
- ralph のプロセスと座席の codex が同じ `CODEX_HOME` / `HOME` を見ていること。herdr server を別の HOME で起動している場合は観測できず `unknown` になる(`CODEX_HOME` を ralph 側で指定すれば観測できる)
- 最初のターンは spawn の直後に始まる(実測: session 開始から `turn_context` まで 2.3〜3.2 秒、役割指示のメッセージはその 0.4 秒以内)。退役ダイアログで止まった座席はターンが始まらないので spawn 時には `unknown` になり、stop 時の観測で拾う

## Affected areas

- `internal/org/`(新規ファイル、`spawn.go`、`verbs.go`、テスト)
- `internal/cli/`(`org.go`、`doctor_codex_models.go`、テスト)
- 文書: rule、spec、recipe、skill、evidence
- 影響しないもの: manifest と receipts のスキーマ、`ralph insights` の集計、claude 座席、`ralph.toml`

## Design decisions

- **観測の時点(ユーザー決定、2026-09-20)**: 「spawn 直後 + stop 時」。spawn の直後に数秒だけ探し、取れなければ `unknown` のまま進む。stop 時に、まだ `unknown` の座席だけもう一度探して receipt を追記する。理由: lead が早く気付け、退役ダイアログや遅い起動で spawn 時に取れなかった座席も拾える。代わりに spawn が最大で数秒長くなる。
- **観測元は codex の session 記録**。pane のステータス行は使わない(TUI の描画に依存する。#163 でも pane の文字列は読まない方針だった)。`[notice.model_migrations]` からの推測もしない: 実 config にそのキーがなくても移行が起きた実例(Run E1)があり、「Use existing model」を選んだ後にも同じキーが書かれるため、移行の有無を決められない。
- **記録の内容は外に出さない**。session 記録には会話の本文が入っている。読むのは spawn 以降に更新されたファイルだけ、取り出すのは model だけ、エラーにも本文を含めない(#164 の config 読み取りと同じ方針)。
- **理由の文言は観測した事実だけを書く**(#163 の教訓)。不一致の理由は「指定は X、codex の session 記録は Y」で、原因(退役モデルの自動移行、config の上書き)は可能性として添える。`models_cache.json` の `upgrade.model` が Y と一致する場合だけ「codex の退役移行(退役日)」と書く、という分岐は足さない(`internal/org` から cache を読む経路を増やさない。退役予定は doctor が出す)。
- **Codex plan advisory(2026-09-20、HIGH 2 / MEDIUM 1、ユーザー決定: 対応案で plan を更新)**: (1) 再 spawn で古い session を拾い得る → `session_meta` の開始時刻が spawn 開始以降であることを条件にし、複数該当は「特定できない」。(2) stop 時の条件「直近の receipt が unknown」は、待ちの途中の中断(receipt なし)や、拒否・dry-run の receipt に弱い → 「spawn 開始以降に、モデルを観測した receipt がない」に変更。スキーマは変えない。(3) 役割指示ファイルのない座席は対応付けできない → 観測せず、待たず、理由付きの `unknown`。相関 ID をプロンプトに足す案は不採用。
- **stop 時は receipt を追記する**(既存の行は書き換えない。receipts は追記専用)。1 座席に `unknown` と `false` が 1 件ずつ残るのは仕様として文書に書く。

## Acceptance criteria

- [ ] AC-1: 役割指示ファイルのパスを含む user メッセージと `turn_context` を持つ記録から、観測関数が `turn_context.model` を返す。パスを含まない記録、spawn 開始より前に更新された記録、通常ファイルでないもの、壊れた行、64 KiB を超える行があっても、誤った座席のモデルを返さず、panic もしない
- [ ] AC-2: codex 座席の spawn で記録が見つかり model が指定と一致すれば、receipt は `honored=true`、`reported_effective_model=<model>`。不一致なら `honored=false`、`reported_effective_model=<実効モデル>`、`reason` に指定と実効の両方のモデル名が入る(fixture: `gpt-5.5` 指定で `gpt-5.6-sol`)
- [ ] AC-2b: 同じ org・seat の古い session(`session_meta` の開始時刻が spawn 開始より前)が、後から更新されて別のモデルを報告していても、新しい spawn の receipt はそれを採らない。条件を満たす記録が 2 件ある場合は `unknown`(理由に「特定できない」)
- [ ] AC-2c: 役割指示ファイルのない codex 座席(inline のプロンプト、空のプロンプト)の spawn は待たずに返り、receipt は理由付きの `unknown`
- [ ] AC-3: 記録が見つからない spawn は、上限時間だけ待った後に `honored=unknown` の receipt を書き、spawn は成功する。観測中のエラー(ディレクトリなし、権限なし)でも spawn は成功する
- [ ] AC-4: claude 座席と dry-run の spawn は観測を行わず、receipt は従来と同じ
- [ ] AC-5: codex 座席の stop で、その座席の spawn 開始以降にモデルを観測した receipt がないときだけ観測し、見つかれば receipt を 1 件追記する。receipt が 1 件もない場合(spawn の待ちの途中で中断)も、間に拒否や dry-run の receipt が挟まっている場合も観測する。観測済みの receipt があるとき、見つからないとき、dry-run のとき、役割指示ファイルのない座席のときは追記しない。観測の成否にかかわらず stop は従来どおり成功する
- [ ] AC-6: `honored=false` の receipt を書いた `ralph org spawn` / `ralph org stop` は stderr に警告(指定と実効の両方のモデル名を含む)を出し、exit code は 0 のまま
- [ ] AC-7: `ralph doctor` の codex スラッグ Check は、pool のスラッグに `upgrade` がある場合に退役日と移行先を Detail に出す(info)。cache にないスラッグの warn が優先される。`upgrade` がない、`retirement_at` が読めない場合も壊れない
- [ ] AC-8: テストは実ホームの `~/.codex` を読まない(観測先を seam で固定)。`TMPDIR=/tmp` でも通る
- [ ] AC-9: 実記録 5 件での確認結果が evidence にあり、model 以外の内容が evidence に含まれない
- [ ] AC-10: 文書(rule / spec / recipe / skill 4 面 / evidence の追記)が実装と一致し、同期ゲート 3 本が通る
- [ ] AC-11: `./scripts/run-verify.sh` と `./scripts/run-test.sh` が green。PR 本文に `Closes #165`

## Implementation outline

1. Slice A(観測): `codex_session.go` とテスト。関数の入力は sessions ディレクトリ、役割指示ファイルのパス、spawn 開始時刻。出力は model と「見つかったか」。環境の解決(`CODEX_HOME` / `HOME`)は別関数にして seam から注入する
2. Slice B(配線): `Spawn` と `Stop` への組み込み、`SpawnResult` / `StopResult` への receipt の結果の追加、CLI の警告、テスト。`testOrg()` と CLI のテストの既定は「空の sessions ディレクトリ・待ち時間ほぼ 0」
3. Slice C(doctor): `upgrade` の読み取りと Detail、テスト
4. Slice D(実記録での確認): ビルドした観測関数を scratch の記録 5 件に対して実行し、evidence を書く(コミットしない一時テストで実行)
5. Slice E(文書): rule、spec、recipe、skill、evidence の追記。同期ゲート

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`gofmt -l internal/org internal/cli`、`go vet`
- Spec compliance criteria to confirm: AC-1〜AC-11。Non-goals(pane を読まない、スキーマを変えない、claude 座席は不変、`ralph.toml` のキーなし)
- Documentation drift to check: `model-routing.md` の「Org runtime model receipts」、spec の FR-9 と codex の受け入れ条件、recipe、skill の receipts の説明、evidence P5 の追記。`check-skill-sync.sh` / `check-sync.sh` / `check-template-purity.sh`
- Evidence to capture: `docs/evidence/codex-effective-model-receipt-2026-09-20.md`、verify / test のログ

## Test plan

- Unit tests: 観測関数(一致、不一致、パスなし、古いファイル、symlink / FIFO、壊れた行、長い行、`turn_context` が user メッセージより後、複数の記録で同じ cwd・別の座席、上限超過)。環境の解決(`CODEX_HOME` あり / なし)
- Integration tests: `Spawn`(true / false / unknown / claude / dry-run / 観測エラー)、`Stop`(unknown のとき追記、true / false のとき追記なし、見つからない、dry-run)。CLI の警告(spawn / stop、stderr、exit 0)。doctor の Detail
- Regression tests: 既存の receipts のテスト(watcher、reject、dry-run)と `ralph insights` のテストが不変で通ること
- Edge cases: 同じ org・seat id を stop 後に再 spawn した場合に、後から更新された古い記録を拾わない(`session_meta` の開始時刻で切る)。該当が 2 件。spawn の待ちの途中で中断した後の stop。拒否 / dry-run の receipt を挟んだ stop。役割指示ファイルのない座席。sessions ディレクトリがない。`retirement_at` が不正な形式
- Evidence to capture: `./scripts/run-test.sh`、race、`TMPDIR=/tmp`、反復実行の結果

## Risks and mitigations

| リスク | 影響 | 対策 |
|---|---|---|
| codex が session 記録の形を変える | 観測できなくなる | 見つからなければ `unknown` に倒す。形の前提を doc comment と文書に明記。fixture は実記録から作る |
| 別の座席・別のセッションのモデルを誤って記録する | 誤った `honored` | 役割指示ファイルの絶対パス(org と seat を含む)で対応付け、spawn 開始時刻以降のファイルだけを見る |
| 会話の本文が ralph の出力に漏れる | プライバシー | 取り出すのは model だけ。エラーは種類だけ。テストで receipts と stderr に fixture の本文が出ないことを確認 |
| sessions ディレクトリが巨大で spawn が遅くなる | spawn の遅延 | 日付ディレクトリ 2 日分、更新時刻で足切り、ファイル数とバイト数の上限、全体は 8 秒で打ち切り |
| spawn が最大 8 秒長くなる | lead の待ち時間 | 見つかり次第すぐ返る(実測では 3 秒前後)。codex 座席だけ。上限はフィールドで変えられる |
| テストが実ホームの記録を読む | 不安定・プライバシー | 観測先を seam で固定(AC-8) |

## Rollout or rollback notes

- 追加だけの変更。receipts と manifest のスキーマは変わらない。revert すれば従来の「常に unknown」に戻る
- 既存の receipts ファイルはそのまま読める

## Open questions

- codex の退役ダイアログが出ている間に session 記録が作られるかどうかは未確認(作られないなら spawn 時は `unknown`、stop 時に観測という設計のままで足りる)

## Deviation notes

- 2026-09-20 work: Slice A(観測関数)は implementer に委譲(78692f4、2 ファイル、テスト 20 件)。逸脱 2 件: 日付ディレクトリの上端は壁時計の「今日 + 1 日」(stop 時は spawn から日が経っていることがあるため)。Lstat を通った記録を開けなかった場合(権限なしなど)だけ、内容を含まないエラーを返す(それ以外は not-found)。orchestrator が HEAD 一致・porcelain 空・コードを確認し、#155 / #163 の実機実行で scratch に残っている実記録 5 件に対して観測を実行した(コミットしない一時テスト): `gpt-5.5` 指定の Run E1 は `gpt-5.6-sol`、Run E2 は `gpt-5.5`、#163 の座席は `gpt-6-astra`、spawn 時刻を session 開始より後にすると not-found、存在しないパスは not-found。1 回あたり 1〜11ms
- 2026-09-20 work: Slice B(配線)は implementer に委譲(d2bf6e4、7 ファイル)。`Org` に `CodexSessionsDir` / `CodexModelObserveTimeout`(既定 8 秒)/ `CodexModelObserveInterval`(既定 500ms)を追加。`checkCapacityAndStart` が `spawn_started` を記録した時刻を返すようにし、観測はその時刻を基準にする。役割指示ファイルのパスは `Spawn` のローカル変数を持ち回り、stop 時は manifest の `agent_started prompt_file=` から読む(同じ定数 `codexPromptFileDetailsPrefix` を両方で使う)。`SpawnResult` / `StopResult` に `ModelReceipt` を追加し、CLI は `honored=false` かつ実効モデルが入っているときだけ警告する(エンベロープの拒否では出さない)。CLI 側の seam は package 変数 3 つを `TestMain` で固定。逸脱: 指定した `HOME=/nonexistent… go test` は go 自体が build cache に HOME を使うため実行できず、テストバイナリをビルドしてからその環境で実行して pass を確認した。orchestrator が HEAD 一致・porcelain 空・`Spawn` / `Stop` の差分を確認
- 2026-09-20 work: Slice C(doctor)は implementer に委譲(7963da0、2 ファイル)。逸脱: テストは plan に書いた `doctor_codex_models_test.go` ではなく、既存のテストがある `internal/cli/doctor_org_test.go` に追加(該当ファイルは存在しなかった)。`upgrade` は `json.RawMessage` で受け、型が想定外の 1 件で cache 全体の decode が失敗しないようにした。`migration_markdown` は decode しない。orchestrator が実 cache での表示を確認し、退役スラッグの一覧と固定文の間に区切りがなく読みにくかったためピリオドを足した(80a50c2、1 行)
- 2026-09-20 work: Slice D / E(a660122、orchestrator)。実記録 5 件での確認を `docs/evidence/codex-effective-model-receipt-2026-09-20.md` に記録(model と status 以外は出力していない)。文書: `model-routing.md`(meta と template。template 側は source のパスを書かない)、spec の FR-9、recipe(2 コピー)、skill(4 面)、#155 の evidence P5 に追記 1 行。同期ゲート 3 本 pass
- 2026-09-20 メモ: 既定の `[org].model_pool` に退役予定の `gpt-5.5`(2026-10-14)が入っているため、既定の設定でも doctor の info が出る。既定値の見直しは #156 の範囲(次回の観測時に issue へ記録する)

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 観測元を実記録 5 件で確認した(`turn_context.model`、役割指示ファイルのパス、先頭 10 行以内、session 開始から約 3 秒)
- [x] critical fork(観測の時点)はユーザー決定済み
- [x] AC は fixture と既存の実記録で決定的に確認できる
- [x] Codex plan advisory(3 件、対応案で plan を更新)
