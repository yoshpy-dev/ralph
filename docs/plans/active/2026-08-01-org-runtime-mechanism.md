# org-runtime-mechanism

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-01
- Related request: docs/specs/2026-08-01-org-runtime.md (PR① 機構層)
- Related issue: N/A
- Type: feat
- Branch: docs/spec-org-runtime (spec ハンドオフ worktree を採用)

## Objective

org runtime の機構層(PR①)を実装する: `ralph org` 動詞セット(spawn / send / wait / read / stop / status / disband)、`[org]` エンベロープ設定、org manifest 自動記録、doctor プローブ。後続 PR(座席化・Lead 自律編成・Watchdog・旧系撤去)が乗る決定論的土台であり、この PR 単体では既存フローの挙動を変えない(互換 AC-9 で担保)。

## Scope

- `internal/config/`: `[org]` セクション追加 — `model_pool`(`{ driver, model }` 配列、CLI ネイティブモデル名)、`driver_pool`、`[org.roles]` 役割別許可プール、`max_seats`、`budget`(座席別/全体 wall-clock 分、fix ラウンド上限)。`Default()` / `Load()` バリデーション / `defaults_sync_test.go` 更新。
- `templates/base/ralph.toml` + `scripts/ralph-config.sh`: `[org]` の宣言ミラーと `RALPH_ORG_*` env エクスポート(ロックステップ 3 面)。
- `internal/org/`(新規パッケージ): エンベロープ検証、座席状態モデル(saga: `spawn_started` → `spawned` / `spawn_failed` → `stopped`、拒否は `rejected`)、**org_id 名前空間付き** manifest JSONL ストア(`.harness/state/org/manifest.jsonl`)、model receipts 追記(`commanded_model` / `reported_effective_model` / `honored: true|false|unknown` の三値)。
- `internal/org/driver/`(新規): herdr / agmsg の `exec.Command` アダプタ。インターフェース分離でスタブ差し替え可能にし、CI(herdr / agmsg 非導入環境)でもテストできるようにする。
- `internal/cli/org.go`(新規): cobra `ralph org` サブコマンドとして動詞 7 種を実装。全動詞は manifest への自動追記を伴う。`--dry-run` は実プロセスを起動せず検証・記録のみ(`dry_run: true` を刻印)。
- `internal/cli/doctor.go` 拡張: herdr / agmsg バイナリ検査(**org 未使用時は informational 表示のみ、exit code に影響させない**)+ `--probe-models` でプール全エントリの起動プローブ。
- ドキュメント最小更新: AGENTS.md repo map への `internal/org/` 追記、spec への参照。全面改稿は PR⑤。

## Non-goals

- 座席の役割プロンプト・常駐セッション運用(PR②)。
- `/org` skill、Lead 自律編成、`ralph org start`(PR③)。
- Watchdog 二層(PR④)。budget は記録と `status` 表示まで、時間超過の自動遮断は PR④。
- 旧系(orchestrator / pipeline / loop skills / TUI)の変更・削除(PR⑤)。この PR では触らない。
- スコープ外書き込みブロック用の seat-facing hooks(座席が存在する PR② で導入)。
- agmsg メッセージスキーマの完全確定(send / read の透過転送で開始し、typed protocol の厳密検証は PR② で座席プロンプトと同時に固める)。
- 対話セッションからの effective model 実測(PR② の座席化と同時。PR① では観測不能ケースを `honored: unknown` として正直に記録する土台まで)。

## Assumptions

- herdr CLI は `agent start/wait/get`、`pane read/send-text/wait-output`、`workspace/tab/pane create` を提供する(herdr.dev/docs/cli-reference で確認済み)。マイナーバージョン差異はアダプタ層に隔離する。
- agmsg は `send` / `history` / `team` / `mode` を提供し、SQLite DB は agmsg 自身が管理する(直接 SQL は書かない)。
- CI には herdr / agmsg が存在しない前提。テストはスタブバイナリ(PATH 先頭に置くフェイク)+ `--dry-run` で成立させる。
- デッドマン N 分の既定値は 10 分として config に予約フィールドを置く(執行は PR④)。
- 座席 worktree の割当粒度は spawn の `--cwd` / `--worktree` 引数で呼び出し側(将来の Lead)に委ね、機構は方針を持たない。

## Affected areas

- `internal/config/config.go`, `internal/config/defaults_sync_test.go`
- `internal/cli/org.go`(新規), `internal/cli/doctor.go`, `internal/cli/root.go`
- `internal/org/`, `internal/org/driver/`(新規)
- `templates/base/ralph.toml`, `scripts/ralph-config.sh`
- `AGENTS.md`(repo map 1 行)
- 触らない: `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh`, `internal/ui/`, `internal/state/`, `.claude/skills/`

## Design decisions

- **フロー**: Standard flow (/work)。config ロックステップと動詞→設定→doctor の依存が密で、並列スライスの利得が薄いため(ユーザー確定)。
- **動詞の実装言語**: Go サブコマンド。ralph.toml のエンベロープを直読でき(shell は toml を読めない現行制約の回避)、バリデーション / manifest が単体テスト可能、Homebrew 配布バイナリに同梱。herdr / agmsg は `exec.Command` で呼ぶ(`run.go` の script exec に前例)(ユーザー確定)。
- **純追加+互換 AC**: PR① は既存コードパスに分岐を入れず、互換性は AC-9 で明示検証する(Codex 所見 4)。
- **アダプタ分離**: herdr / agmsg 呼び出しは interface 越しに行い、テストはスタブ実装。外部 CLI の仕様変化を 1 パッケージに閉じ込める。
- **spawn は saga として実装**(Codex 所見 2): 副作用前に `spawn_started` を追記し、herdr pane id / agmsg 参加などの外部 ID を各ステップ成功時に永続化、終端は `spawned` か `spawn_failed`(補償: 作成済みリソースの停止を試行し、結果も記録)。同一 `--id` の再実行は中間状態から再開または補償後の再作成。
- **dry-run の監査分離**(Codex 所見 1): dry-run イベントは `dry_run: true` を必須刻印し、`status` のロスター表示・`max_seats` 集計から既定で除外する(`status --all` でのみ表示)。
- **manifest の名前空間**(Codex 所見 3): 全イベントに `org_id`(実行単位)/ `seat_id` / `worktree` を必須フィールド化。`max_seats` 集計・`status` / `disband` の対象は同一 `org_id` 内に限定。
- **receipts の正直な三値**(Codex 所見 5): `commanded_model`(argv で渡した値)と `reported_effective_model`(ドライバから観測できた値)を分離し、観測できない場合は `honored: unknown`。`true` はドライバ観測による確認がある場合のみ。

## Acceptance criteria

- [ ] AC-1: プール外モデルで `ralph org spawn` → 非ゼロ終了、拒否イベントが manifest に記録され、receipts に `commanded_model` / `honored: false` が残る。
- [ ] AC-2: 同一 `org_id` 内で `max_seats` 到達後の spawn → 拒否+manifest 記録。別 `org_id` の座席は集計に混入しない(並行 2 名前空間テストで検証)。
- [ ] AC-3: 同一 `--id` での spawn 再実行 → 既存座席を返すか中間状態から再開し、pane / agmsg 参加が重複しない(冪等)。
- [ ] AC-4: `ralph org status` は herdr / agmsg / LLM が全て停止・不在でも manifest から全座席の状態(saga 状態含む)と履歴を表示する。dry-run イベントは既定で除外される。
- [ ] AC-5: `ralph doctor` は herdr / agmsg 不在を報告し、`--probe-models` でプール内の無効なモデル ID を警告する。
- [ ] AC-6: `[org]` 既定値が config.go / templates/base/ralph.toml / ralph-config.sh の 3 面で一致し、`defaults_sync_test.go` が drift を検出する。
- [ ] AC-7: `go test ./...` と `./scripts/run-verify.sh` が green。既存フロー(`/work`・Ralph Loop)の関連ファイルに差分がない。
- [ ] AC-8: 全動詞が `--dry-run` で実プロセス起動なしに検証+`dry_run: true` 付き記録を行える。
- [ ] AC-9(互換): `[org]` セクションのない既存 ralph.toml が警告なしで load でき、herdr / agmsg 不在でも `ralph doctor` の exit code が悪化しない(informational のみ)。既存テスト全件が新規警告なしで pass する。
- [ ] AC-10(saga): spawn の各外部ステップ(pane 作成 / agmsg 参加 / プロンプト投入)で失敗を注入すると、manifest に `spawn_failed` と補償結果が記録され、孤児リソースの ID が manifest から追跡できる。

## Implementation outline

1. **Slice 1 — `[org]` 設定**: config.go struct / `Default()` / `Load()` バリデーション(空プール・不明 driver・重複エントリ・roles のプール外モデル拒否)→ templates/base/ralph.toml → ralph-config.sh env → defaults_sync_test.go 更新。互換 AC-9 のテストを含む。
2. **Slice 2 — `internal/org` コア**: エンベロープ検証、saga 状態モデル、org_id 名前空間付き manifest JSONL ストア(追記専用・読取・破損行スキップ・dry-run 除外集計)、三値 receipts。table-driven unit tests。
3. **Slice 3 — driver アダプタ**: herdr / agmsg interface + `exec.Command` 実装 + テスト用スタブ(argv 検証可能なフェイク)。存在チェック(`exec.LookPath`)と失敗時の構造化エラー。
4. **Slice 4 — `ralph org` 動詞**: cobra 配線、spawn saga(`spawn_started` → 外部 ID 永続化 → `spawned` / `spawn_failed` + 補償)、send / wait / read / stop / status / disband、`--dry-run`。スタブによるライフサイクル統合テスト+各 saga 境界への failure-injection テスト。
5. **Slice 5 — doctor 拡張 + ドキュメント最小更新**: informational バイナリ検査、`--probe-models`、AGENTS.md repo map、進捗チェックリスト更新。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(golang スコープ: gofmt / go vet / lint)。
- Spec compliance criteria to confirm: spec FR-1(動詞 7 種+バリデーション拒否)、FR-2(エンベロープ 3 面ロックステップ+doctor プローブ)、FR-9 の機構部分(manifest 自動追記・receipts)。AC-1〜10 との対応を verify レポートに明記。
- Documentation drift to check: AGENTS.md repo map、`docs/specs/2026-08-01-org-runtime.md` との整合(PR① スコープの逸脱がないか。receipts 三値化は spec の記述より厳密化なので spec 側に追記)、`ralph.toml` テンプレートのコメント。
- Evidence to capture: 検証コマンド出力、`ralph org spawn --dry-run` の manifest 追記サンプル、failure-injection 時の manifest サンプル、doctor 出力。

## Test plan

- Unit tests: エンベロープ検証(プール内/外、役割制約、max_seats 境界)、manifest 追記/読取/破損行スキップ/dry-run 除外/org_id 別集計、receipts 三値、config Load バリデーション。
- Integration tests: スタブ herdr / agmsg を PATH に置いた spawn→status→stop ライフサイクル、各 saga 境界(pane 作成 / agmsg 参加 / プロンプト投入)での failure-injection と補償検証、並行 2 `org_id` の相互不干渉、`--dry-run` 全動詞、doctor(バイナリ不在/存在/無効モデル)。
- Regression tests: `defaults_sync_test.go`、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、既存 `go test ./...` 全件(新規警告なし)。
- Edge cases: 空 model_pool、重複座席 id、manifest 並行追記(O_APPEND 単一行)、`[org.roles]` にプール外モデル、`[org]` セクション欠落(AC-9)、spawn 再実行時の中間状態(`spawn_started` のまま放置されたレコード)。
- Evidence to capture: `go test ./...` 出力、`./scripts/run-test.sh` 出力。

## Risks and mitigations

| リスク | 影響 | 緩和 |
|---|---|---|
| herdr / agmsg の CLI 仕様変化(若いツール) | spawn 破損 | アダプタ 1 パッケージに隔離、doctor で最低バージョン検査、実 CLI 統合は手元検証で担保 |
| CI に herdr / agmsg がない | テスト不能 | スタブバイナリ+`--dry-run` を一次テスト経路に設計 |
| config ロックステップ漏れ | 下流 drift | defaults_sync_test 拡張を Slice 1 に内包 |
| 座席ロジックへのスコープクリープ | PR 肥大 | Non-goals 明記、動詞は転送と記録に限定 |
| spawn 途中失敗による孤児リソース | 監査欠落・リーク | saga 化+外部 ID 永続化+補償+failure-injection テスト(AC-10) |
| 実行単位の混線(旧実行・並行実行) | 集計・disband の誤爆 | org_id 名前空間必須化+並行テスト(AC-2) |
| 既存ユーザーへの doctor ノイズ | 「純追加」の毀損 | informational 扱い+互換 AC-9 |

## Rollout or rollback notes

- 純追加のため既定挙動への影響なし(AC-9 で検証)。`ralph org` を呼ばない限り不可視。
- ロールバック: PR revert のみ。データ移行なし(manifest は新規パス、旧 state と非干渉)。
- 下流(`ralph init` 生成物)には `[org]` セクションがテンプレート経由で入るが、未設定でも Load 既定値で動作する。

## Open questions

- doctor `--probe-models` の実行コスト(モデル×driver ごとに CLI 起動)— 実装時に計測し、遅ければ並列化かキャッシュ。
- agmsg チーム名の正規形(`ralph-<org_id>` を仮置き)— PR② の座席プロンプト設計と同時に確定。
- `spawn_started` のまま残った stale レコードの回収ポリシー(gc 動詞 or status 表示のみ)— 実装時に決め、plan に追記。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created (docs/spec-org-runtime, spec ハンドオフ)
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] Spec 承認済み(docs/specs/2026-08-01-org-runtime.md)
- [x] フロー確定(Standard /work)・動詞実装言語確定(Go)
- [x] 影響範囲は既存フロー非干渉(純追加+互換 AC-9)で閉じている
- [x] CI 成立戦略あり(スタブ+--dry-run+failure-injection)
- [x] Codex plan advisory 反映済み(5 件: dry-run 分離 / spawn saga / org_id 名前空間 / 互換 AC / receipts 三値)
