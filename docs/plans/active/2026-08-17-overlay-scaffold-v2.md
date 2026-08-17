# overlay-scaffold-v2 — Phase 1: エンジン基盤

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-17
- Related request: docs/specs/2026-08-17-overlay-scaffold-v2.md(オーバーレイ型スキャフォールド配布と非対話 upgrade)
- Related issue: N/A
- Type: feat
- Branch: docs/spec-overlay-scaffold-v2(/spec ハンドオフ worktree を採用。PR タイトルは feat 接頭辞)

## Objective

layout v2 系列(全 5 段)の 1 本目として、スペック文書と、後続フェーズが依存する upgrade エンジン基盤を導入する: manifest v3 スキーマ(所有属性 core/fork/seed/block)と、新 upgrade プリミティブ(core 全置換プランナ / managed block 更新 / settings ディープマージ / advisory diff / レポート出力)を **CLI 未配線のまま** 追加する。既存の `ralph upgrade` 挙動はこの PR では一切変えない。

## PR 系列ロードマップ(この plan は Phase 1 のみを対象)

| Phase | 内容 | plan/worktree |
|---|---|---|
| 1(本 plan) | スペック + manifest v3 + エンジンプリミティブ(未配線) | この worktree |
| 2 | `templates/base/` の v2 再編(`.claude/rules/ralph/` 移設、AGENTS.core.md、settings dispatcher 化、`.ralph/local/` 骨格)+ `ralph init` v2 生成 | 個別 /plan |
| 3 | `ralph upgrade` 非対話化への配線 + 対話型コンフリクト解消・baseline の撤去 | 個別 /plan |
| 4 | 旧レイアウト検出と確認付き移行(自動 fork 保全 + 移行レポート) | 個別 /plan |
| 5 | `ralph eject`/`adopt`/`doctor --strict`/`status` 所有表示 + 上流固有記述混入ガード + メタリポジトリ自己移行 | 個別 /plan |

後続フェーズの canonical reference はスペック(`docs/specs/2026-08-17-overlay-scaffold-v2.md`)とする。

## Scope

- `internal/scaffold/manifest.go`: manifest v3 — `meta.layout` フィールド、パスごとの所有属性(`owner = core|fork|seed|block`)、fork 記録(`forked_from_version`)。**v3 の書き込みはオプトイン専用 API に隔離**し、既存コンストラクタ/セッター(`NewManifest`/`SetFile` 系)の出力は legacy のまま不変とする。v1/v2 manifest の読み込みは owner を確定させず legacy として保持する(owner の実付与は Phase 4 の移行分類器の責務。旧 upgrade コード向けには互換ビューのみ提供)
- `internal/upgrade/` 新規プリミティブ(すべて未配線・単体テスト付き):
  - core 全置換プランナ: 埋め込み FS + manifest → **順序付き操作プラン(delete/create/update/report/manifest)+ コミットバリア**(manifest 前進は全工程成功後のみ)。fork 記録パスの置換抑止、fork 記録なしハッシュ不一致の「未解決 drift」分類(書き込みなし)、致命/警告の結果分類と再実行時の分類安定性
  - managed block エンジン: `<!-- BEGIN RALPH MANAGED (ralph:<surface>) -->` / `<!-- END RALPH MANAGED -->` の検出・生成・block 内のみの置換。マーカー欠落時は末尾追記プラン、破損(片側欠落・重複)時はエラー分類
  - settings.json 3-way マージ: 入力は `current / 旧所有テンプレート / 新所有テンプレート` + 所有 JSON パス集合。所有配列エントリ(permissions.allow、hooks イベント配列)の追加・更新・削除を旧→新テンプレート差分から導出し、ユーザ由来エントリと未知キーを保全。出力は決定的順序
  - advisory diff: 「ディスク実体 vs 新テンプレート」の unified diff(スペックの定義に従う。テンプレート側変更の検出は記録済みハッシュ比較。既存 `unified_diff.go` を再利用)
  - upgrade/migration レポートライタ(`docs/reports/` 形式)
  - 共有パスバリデータ: 全プリミティブが操作生成前に非ローカルパス(`..`・絶対パス・セパレータ異常)を拒否する(既存 `cleanTemplateRelPath` を共通化)
- スペック文書の同梱(コミット済み)

## Non-goals

- CLI(`internal/cli/upgrade.go` ほか)への配線・挙動変更(Phase 3)
- `templates/base/` の変更(Phase 2)
- 対話型コンフリクト解消・baseline の削除(Phase 3)
- 移行ロジック(Phase 4)、新コマンド eject/adopt(Phase 5)
- リリースタグの発行

## Assumptions

- テンプレートはリリースタグを切るまで下流に届かないため、main 上の中間状態は下流に影響しない
- manifest v3 フィールドは additive とし、v1 互換フィールド(`hash`/`managed`)は書き続ける(v2 導入時と同じ方針)
- プリミティブの公開 API は Phase 3 での配線を唯一の消費者として最小に保つ

## Affected areas

- `internal/scaffold/manifest.go` / `manifest_test.go`
- `internal/upgrade/`(新規ファイル: replace / block / settingsmerge / advisory / report + 各テスト)
- `docs/specs/2026-08-17-overlay-scaffold-v2.md`(新規、コミット済み)
- `AGENTS.md` Repo map の `internal/upgrade/` 記述(sync-docs で追従)

## Design decisions

- 5 層オーバーレイ + 全置換 + managed block + seed-once(スペックで確定。対話解消は完全廃止)
- 段階 PR 系列(全 5 段)。本 worktree は「スペック + Phase 1」を 1 本目の PR とする — ユーザ確認済み
- /spec ハンドオフ worktree(branch `docs/spec-overlay-scaffold-v2`)をそのまま採用し、二重 worktree を作らない。PR タイトルは `feat:` 接頭辞とし、branch 名との不一致が PR ゲートに影響する場合のみ rename を検討
- 所有権は物理ディレクトリでなく manifest 属性で表現(スペック「層モデル」)。新規 init と移行後は単一レイアウト
- 予期しない core 改変は上書きせず「未解決 drift」として据え置く(非破壊デフォルト)— スペック FR-4
- Codex アドバイザリ反映(6 件): (1) v3 書き込みはオプトイン API に隔離し既存経路の manifest 出力を不変に保つ (2) advisory diff は「ディスク実体 vs 新テンプレート」と定義(スペックにも明記済み) (3) settings マージは 3-way(current/旧所有/新所有 + 所有 JSON パス)とする (4) 置換プランナは順序付き操作プラン + コミットバリアの形にする (5) legacy manifest の owner は読み込みで確定せず Phase 4 の移行分類器に委ねる (6) 共有パスバリデータを Phase 1 の AC に昇格

## Acceptance criteria

- [ ] AC-1 manifest v3: owner/fork/layout フィールドが TOML round-trip する。v1/v2 manifest はエラーなく読め、owner は legacy(未確定)として保持される(読み込み正規化で core/fork を確定させない)
- [ ] AC-2 置換プランナ: fork 記録のあるパスが操作プランから除外され、fork 記録なしのハッシュ不一致が「未解決 drift」(書き込みなし)に分類される。操作プランは順序付き(delete/create/update → report → manifest)で、途中失敗時に manifest 前進操作が発行されないこと、部分適用後の再実行で分類が安定することがテストで保証される
- [ ] AC-3 managed block: block 内のみ更新・block 外バイト保全・マーカー欠落時は末尾追記プラン・破損時はエラー分類(書き込みなし)がテストで保証される
- [ ] AC-4 settings 3-way マージ: ユーザ追加キー/配列エントリが保全され、所有 JSON パス集合の外に書き込まず、旧→新テンプレート差分から所有エントリの追加・更新・削除が導出され、出力順序が決定的であることがテストで保証される
- [ ] AC-5 advisory diff: 「ディスク実体 vs 新テンプレート」の unified diff が、記録済みハッシュ比較でテンプレート側変更を検出したファイルについて収集・描画される
- [ ] AC-6 既存挙動不変: 既存テストスイートが全パスし、`internal/cli` に本 PR での挙動変更がない
- [ ] AC-7 スペック文書が同一 PR に含まれる
- [ ] AC-8 v3 オプトイン隔離: Phase 1 時点の `ralph init` / `ralph upgrade` が書き出す manifest に v3 フィールド(`layout`/`owner`/fork 記録)が含まれないことがテストで保証される
- [ ] AC-9 パス検証: 全プリミティブが操作生成前に非ローカルパス(`..`・絶対パス)を共有バリデータで拒否することがテストで保証される

## Implementation outline

1. 共有パスバリデータ + manifest v3 スキーマ(オプトイン書き込み・legacy 読み込み保持)+ テスト
2. managed block エンジン + テスト(表駆動: 正常更新 / 欠落追記 / 破損 / CRLF / 末尾改行なし)
3. settings.json 3-way マージ + テスト(所有配列の追加・更新・削除・重複・決定的順序)
4. core 全置換プランナ(順序付き操作プラン + コミットバリア、fork 抑止・未解決 drift 分類、部分失敗テスト)+ テスト
5. advisory diff(ディスク vs 新テンプレート)+ レポートライタ + テスト
6. AGENTS.md Repo map 追従(sync-docs)

各スライスは implementer 委譲(`.claude/rules/subagent-policy.md`)、1 スライス 1 コミット。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(go vet / gofmt / staticcheck 相当のリポジトリ標準)
- Spec compliance criteria to confirm: スペック FR-1(工程 1-6 のプリミティブ存在)/ FR-4(未解決 drift の非破壊分類)/ FR-5(block 仕様)/ NFR-2(非破壊性)に対する基盤整合。配線系 FR は Phase 3 以降のスコープであることを verify レポートに明記
- Documentation drift to check: AGENTS.md Repo map(`internal/upgrade/` の説明)、スペックと plan の相互参照
- Evidence to capture: verify レポート(`docs/reports/verify-*.md`)にテスト・静的解析の実行ログ要約

## Test plan

- Unit tests: 各プリミティブの表駆動テスト(manifest round-trip、block 更新、3-way マージ、置換プラン分類、advisory 描画、パスバリデータ)
- Integration tests: 埋め込みテンプレート FS + 一時ディレクトリでの操作プラン end-to-end(書き込みなしのプラン検証)+ 部分失敗シナリオ(report 書き込み失敗・manifest 前進の抑止・部分適用後の再実行分類)
- Regression tests: 既存 `internal/upgrade` / `internal/cli` / `internal/scaffold` テスト全パス + init/upgrade の manifest 出力に v3 フィールドが混入しないこと(AC-8)
- Edge cases: 破損マーカー(片側欠落・重複)、CRLF・末尾改行なしの block ファイル、空/コメントなし settings.json、所有配列エントリの重複・順序、v1 manifest の空ハッシュエントリ、fork 対象が新テンプレートから削除されているケース、core ファイルのディスク欠落、`..`/絶対パスを含む manifest エントリ
- Evidence to capture: `docs/reports/test-*.md` にカバレッジと失敗分析

## Risks and mitigations

- branch 名 `docs/spec-overlay-scaffold-v2` と feat PR の不一致が PR ゲート(`ensure-pr-title-prefix.sh`)に抵触する可能性 → /pr 時にゲートの判定基準を確認し、必要なら branch rename + worktree state 更新
- プリミティブ API が Phase 3 の配線要求と乖離するリスク → 公開面を最小化し、Phase 3 で調整可能な粒度に留める(plan の「Assumptions」に明記)
- manifest v3 を新しい ralph が書いた後、古いバイナリで upgrade するケース → additive フィールド + v1 互換フィールド維持で読み込みは壊れない。古いバイナリは owner 属性を無視するが、Phase 3 まで挙動配線がないため実害なし
- 未配線コードの腐敗(Phase 2-3 が遅延した場合)→ ロードマップをスペックに固定し、各フェーズの canonical ref をスペックに一本化

## Rollout or rollback notes

- 本 PR は動作変更ゼロのため revert 安全
- リリースタグは系列完了(Phase 5 + メタリポジトリ自己移行の検証)まで切らない

## Deviations

- Slice 4 コミット後、orchestrator が staticcheck QF1002(replaceplan.go の tagged switch 化)をインライン修正(`refactor: use tagged switch in replace planner classification`)。挙動変更なし
- 完了ゲートの `run-verify.sh` で既存の日付依存テスト failure を発見(`tests/test-gc-artifacts.sh` の recent フィクスチャが固定日付 `2026-07-12` で、`--days 30` の窓を 2026-08-17 時点で外れた。main でも再現、本タスクと無関係)。verify exit 0 と PR CI を塞ぐため、最小修正としてフィクスチャ日付を実行日相対に変更(`fix: make gc-artifacts recent fixture date dynamic`)。スコープ外の発見への対処として本 plan に記録

## Open questions

- upgrade レポートの exit code 設計(未解決 drift 時の 0/非 0)— Phase 3 で確定(スペック Open questions 引き継ぎ)
- AGENTS.md managed block に残す最低限の要点の分量 — Phase 2 で確定

## Progress checklist

- [x] Plan reviewed
- [x] Branch created(docs/spec-overlay-scaffold-v2、spec コミット済み)
- [x] Implementation started(slice 1: 75de545 / slice 2: 73ad8a7 / slice 3: b317869 / slice 4: a32810a / slice 5: 10c6e2a + 追補 1e38046, 7dca14e)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
