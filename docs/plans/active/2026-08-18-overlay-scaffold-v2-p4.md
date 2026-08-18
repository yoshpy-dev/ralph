# overlay-scaffold-v2 — Phase 4: 旧レイアウト移行 + owner 考慮 untracked 分類

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-18
- Related request: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 4 / 全 5 段。#143 → #144 → #145 マージ済み)
- Related issue: N/A
- Type: feat
- Branch: feat/overlay-scaffold-v2-p4

## Objective

旧レイアウト(v1/v2 manifest、`meta.layout` なし)の下流プロジェクトを、確認付きの一回限りの移行で v2 レイアウトへ引き上げる(スペック FR-6/7/8)。同時に、Phase 3 の承認済み Known gap である「owner 非考慮の untracked 分類」を修正する(新規 seed パス衝突が誤 drift になる問題 — リリース前必須)。これにより Phase 3 で fail-closed にしたレガシー経路が解消され、系列は Phase 5(リリース準備)のみを残す。

## Scope

**A. owner 考慮の untracked 分類(Known gap 解消、独立スライス)**

- `ReplaceOptions` に owner 解決(`OwnerForPath func(string) string` 相当)を追加し、`classifyUntracked`(manifest 未登録 + desired に存在するパス)を owner 考慮にする:
  - seed 所有パス: ディスク欠落 → create(現行どおり)/ ディスク既存(内容不一致含む)→ **書き込みなしで seed として採用**(manifest 再構築で owner=seed・DiskHash=現物を記録)+ テンプレートと差分があれば advisory。drift 分類しない(L5 契約準拠)
  - block 所有パス: 現行どおり skip-paths で planner 外(変更なし)
  - core 所有パス: 現行の分類(equal→refresh / differ→drift)を維持
- `runUpgradeV2` から `ownerForScaffoldPath` を resolver として渡す
- Phase 3 tech-debt 行(seed 衝突誤 drift)を RESOLVED 化

**B. 移行分類器(pure、`internal/cli/migrate.go` または `internal/upgrade`)**

旧 manifest(v1/v2)+ ディスク + 新テンプレート(desired state)から `MigrationPlan` を構築。改変判定の基準は **`LegacyEntryState` 契約**として明文化する(Codex 所見 4): 比較基準は `disk_hash` があれば disk_hash、なければ `hash`(旧 v1 互換)/ `state=partial`(旧対話解消でユーザが編集採用したもの)は常に「改変済み」扱い / `managed=false`(旧 skip)は fork / 空 hash(旧 heal 対象)は「改変済み」扱いに倒す(誤削除より誤 fork が安全)— 表駆動テストで固定:

- 旧エントリごとに旧記録ハッシュで改変判定:
  - **未改変 + 新レイアウトで再配置されたパス**(shipped rules / pack rules → `.claude/rules/ralph/`): 旧パス削除(新パスは移行後の v2 upgrade が生成)
  - **未改変 + 新レイアウトで廃止されたパス**: 削除
  - **未改変 + パス不変**: そのまま(v2 upgrade が内容を収束)
  - **改変済み**: **新レイアウト上の対応パスへ fork として移設**(fork は planner の置換対象外となり、advisory で新 core との diff が出続ける。旧パス残置だと rules の二重ロード等が起きるため新パス集約 — 下記 Design decisions)
  - **unmanaged(旧 skip 記録)**: fork として引き継ぎ(パス再配置対象なら移設)
- 特別面(FR-8):
  - CLAUDE.md: 未改変 → 最小シードへ置換 / 改変済み → 完全不可侵(ralph ガイダンスは rules/ralph から供給されるためどちらでも機能)
  - AGENTS.md / .gitignore: 未改変 → 新テンプレート(block 入り)へ置換 / 改変済み → 据え置き(移行後の v2 upgrade の block エンジンが block を追記し、既存内容は block 外に保全)。改変済みの場合は旧テンプレート由来内容の重複が残り得ることをレポートで案内(FR-8 の「未改変内容の block 置換」は未改変ケースで完全充足。改変ケースの内容手術は行わない — Design decisions に記録し、スペックに一文追補)
  - settings.json(移行専用ステップ — Codex 所見 1): 未改変(旧記録ハッシュ一致)→ 新テンプレート settings へ丸ごと置換 + スナップショット生成(fresh v2 init と同一状態)。改変済み → 旧テンプレート世代が出荷していた既知のレガシー ralph hook コマンド(`./.claude/hooks/<name>.sh` 直接参照の 8 スクリプト)に一致する hooks エントリを剪定してから v2 の 3-way マージへ委ね、剪定内容をレポートに記録(ユーザ追加 hooks は保全)。これを怠ると旧 direct-hook が「ユーザ所有」として dispatcher と恒久二重実行になる
  - `.codex/AGENTS.override.md`: owner=seed で再帰属(Phase 3 tech-debt 行の解消)
- `.ralph/baseline/` は削除対象
- プレビュー描画: パスごとの処置(relocate / fork 移設 / delete / keep / seed 置換)を表形式で表示

**C. 移行実行 + CLI 配線**

- `ralph upgrade` のレガシー検出(現行 fail-closed)を移行フローに置換: (1) **git 前提チェック**(git リポジトリであること + 作業ツリークリーン。非 git または dirty → 中断) → (2) 移行プレビュー表示 → (3) **明示確認**(対話 y/N。`--yes` でスキップ — init と同じ規約。`--dry-run` はプレビューのみで終了)→ (4) 移行実行(移設/fork/削除 → v3 manifest 書き込み(owner 付与、fork 記録、Packs 継承))→ (5) **そのまま既存の v2 upgrade エンジンへ連鎖**(dispatcher・`.ralph/core`・新規ファイル等の生成は v2 エンジンが収束)→ (6) 移行レポート `docs/reports/ralph-migration-<date>.md`(分類一覧、fork 化された改変ファイルの diff、案内)
- **プリフライト + 再実行安定性(Codex 所見 2)**: 全ファイル操作(移設/削除/置換)を実行前に一括検証(存在・衝突・パス安全)し、検証失敗は書き込みゼロで中断。分類は再実行安定にする — 部分適用後の再実行でも「移設済み(移設先が期待内容と一致 + 旧パス不在)= 完了扱い」等で残作業のみが計画される。v3 manifest 書き込みはファイル操作成功後のバリア。manifest 書き込み後のレポート/連鎖 upgrade の失敗は警告に留め(移行自体は完了、再実行は v2 経路で収束)、失敗時エラーは git 復元手順を案内
- **移設先衝突マトリクス(Codex 所見 3)**: fork 移設先に既存ファイルがある場合 — (a) 内容が移設元と一致 → 移設済みとして旧パス削除のみ、(b) 内容が新テンプレートと一致 + 移設元が未改変 → 旧パス削除 + core 採用、(c) それ以外(発散)→ 書き込みゼロで中断し衝突レポート(ユーザが手動解決後に再実行)
- `ralph init` の再 init・`ralph pack add` のレガシー fail-closed メッセージを「`ralph upgrade` で移行可能」に更新

**D. ドキュメント同期**

- README / docs/recipes に移行手順(1 節)、スペックの FR-8 追補(改変済み block 面の扱い)、AGENTS.md map、tech-debt 2 行(seed 衝突・codex override 再帰属)の RESOLVED 化

## Non-goals

- `eject` / `adopt` / `doctor --strict` / 混入ガード / Codex dispatcher パリティ(Phase 5)
- リリースタグ発行(Phase 5)
- 改変済み AGENTS.md の内容手術(テンプレート由来部分の自動識別・除去)— 非決定的で危険。block 追記 + レポート案内に留める
- rename/move 検出の一般化(既存 tech-debt 行のまま)

## Assumptions

- 移行対象の旧レイアウトは Phase 1 以前の ralph リリース(v0.x 系タグ)が生成した manifest v1/v2。それらのハッシュ記録は改変判定に十分(現行 `ComputeDiffs` が使っていたものと同じフィールド)
- 旧テンプレートの内容自体は再構築不能(バイナリ 1 世代埋め込み)だが、移行の改変判定は「旧 manifest の記録ハッシュ vs 現ディスク」で完結し、旧テンプレート内容は不要
- 移行は git クリーン前提のため全操作が git で監査・復元可能。バックアップディレクトリは作らない
- メタリポジトリは manifest を持たないため移行対象外(Phase 2 で root 移行済み)

## Affected areas

- `internal/upgrade/replaceplan.go`(untracked 分類の owner 考慮)
- `internal/cli/migrate.go`(新規: 分類器 + 実行)、`internal/cli/upgrade.go` / `upgrade_v2.go`(レガシー経路の置換、resolver 受け渡し)、`internal/cli/init.go` / `pack.go`(fail-closed メッセージ更新)
- テスト: レガシーレイアウト fixture(v1 世代 init の再現)による e2e
- docs(README、recipes、スペック FR-8 追補、tech-debt)

## Design decisions

- 【改変済みファイルの fork 移設先 = 新レイアウトパス】旧パス残置だと `.claude/rules/` 直下に旧 rule が残り、`rules/ralph/` の新 core と二重ロードされる(矛盾ガイダンスの常時注入)。新パスへ fork として集約すれば、planner の fork 抑止で置換されず、advisory で新 core との diff が出続け、二重ロードも起きない — 技術的に一意
- 【改変済み AGENTS.md/.gitignore は据え置き + v2 block 追記】FR-8 の「旧テンプレート由来未改変内容の block 置換」は改変済みファイルでは内容手術(テンプレート断片の識別)を要し非決定的。据え置き + block 追記(block 外完全保全)+ レポートで重複整理を案内する。スペックに一文追補(スライス 5)
- 【確認 UX】対話 y/N + `--yes`(init の既存規約)。`--dry-run` はプレビューのみ。Phase 3 の「非対話」原則は v2 レイアウトの upgrade に対するもので、一回限りの移行の明示確認はスペック FR-6 の要求
- 【非 git ターゲットは移行拒否】git が唯一のロールバック手段(バックアップを作らない設計)のため
- 【untracked 分類の owner 解決は callback 注入】planner(internal/upgrade)は CLI 層の `ownerForScaffoldPath` に依存できないため、`ReplaceOptions` に関数を渡す。nil のとき現行挙動(後方互換)
- 【Codex アドバイザリ反映(5 件)】(1) settings は移行専用ステップ(未改変=丸置換 / 改変=既知レガシー hook 剪定)で hooks 二重実行を防止 (2) プリフライト一括検証 + 再実行安定分類で「git 復元頼み」を脱却 (3) 移設先衝突マトリクスを AC 化 (4) LegacyEntryState 契約(partial/空 hash は改変扱いに倒す) (5) 移行操作にも既存のパス/symlink 安全契約を明示適用
- Critical forks: なし(いずれもスペック既定・Phase 2 の承認済み方針(upgrade 統合)・技術的一意性から決定)

## Acceptance criteria

- [x] AC-1 owner 考慮 untracked 分類: manifest 未登録 + desired に存在する seed パスにローカルファイルが既存する場合、drift にならず(exit 0)、ファイル不可侵のまま owner=seed で採用され、テンプレートと差分があれば advisory がレポートに載る(Phase 3 Known gap シナリオの再現テストで保証)
- [x] AC-2 resolver nil の後方互換: 既存の全 upgrade テストが無変更で green
- [x] AC-3 レガシー検出: v1/v2 manifest への `ralph upgrade` が git 前提チェック(非 git / dirty → 書き込みゼロで中断)→ プレビュー → 確認(y/N、`--yes` スキップ、`--dry-run` はプレビューのみ)を経て移行を実行する
- [x] AC-4 分類規則(FR-7): 未改変の再配置パスは旧パス削除 + 新パスは連鎖 v2 upgrade が生成 / 改変済みは新パスへ fork 移設(manifest に fork 記録 + `forked_from_version`)/ unmanaged は fork 引き継ぎ — それぞれ e2e で保証
- [x] AC-5 特別面(FR-8): 未改変 CLAUDE.md → 最小シード置換、改変済み CLAUDE.md → バイト不変 / 未改変 AGENTS.md・.gitignore → block 入りテンプレートへ置換、改変済み → 据え置き + 連鎖 upgrade で block 追記(block 外保全)+ レポート案内
- [x] AC-6 `.codex/AGENTS.override.md` が移行で owner=seed に帰属される(改変有無問わず不可侵)
- [x] AC-7 移行完了後の状態: `meta.layout = "v2"` の v3 manifest(全エントリ owner 付き、Packs 継承)、`.ralph/baseline/` 不在、連鎖 v2 upgrade により dispatcher・`.ralph/core`・`.ralph/local` 骨格が存在、直後の `ralph upgrade` 再実行が no-op(fork の advisory は継続)
- [x] AC-8 移行レポート: `docs/reports/ralph-migration-<date>.md` に分類一覧・fork 化ファイルの diff・改変済み block 面の重複整理案内が出力される
- [x] AC-9 途中失敗: ファイル操作失敗時に v3 manifest が書かれず、エラーが git 復元を案内する。`--dry-run` は書き込みゼロ
- [x] AC-10 fail-closed メッセージ更新: `ralph init`(再 init)・`ralph pack add` のレガシー検出が「`ralph upgrade` で移行」を案内する(pack add は移行自体は行わない)
- [x] AC-11 パック: 旧パス pack rule(`.claude/rules/<lang>.md`)が未改変なら削除 + 新パスへ(連鎖 upgrade)、改変済みなら新パスへ fork 移設。`Meta.Packs` が v3 manifest に継承される
- [x] AC-12 tech-debt 2 行(seed 衝突誤 drift / codex override 再帰属)が RESOLVED 化される
- [x] AC-13 settings 移行: 未改変 settings は新テンプレート + スナップショットへ置換され fresh init と同一状態になる。改変済み settings は既知レガシー ralph hook エントリが剪定され(剪定はレポート記録)、ユーザ追加 hooks/permissions が保全され、dispatcher hook との二重実行が発生しない(テストで保証)
- [x] AC-14 プリフライトと再実行安定: 移行のファイル操作は実行前に一括検証され、検証失敗は書き込みゼロ。部分適用後の再実行が残作業のみを計画して完走する(失敗注入テストで保証)
- [x] AC-15 衝突マトリクス: 移設先既存の 3 ケース((a) 一致 → 完了扱い / (b) テンプレ一致 + 未改変 → core 採用 / (c) 発散 → 書き込みゼロ中断 + 衝突レポート)がテストで保証される
- [x] AC-16 パス安全: 全移行操作(移設元/先・削除対象)が `scaffold.CleanLocalRelPath` + `upgrade.ValidateRealParentChain` + 葉 Lstat で検証され、symlink 親・非 regular・トラバーサルは書き込みゼロで拒否される(テストで保証)

## Implementation outline

1. owner 考慮 untracked 分類(ReplaceOptions callback + classifyUntracked 分岐 + runUpgradeV2 配線 + Known gap 再現テスト)— AC-1/2
2. 移行分類器(pure: MigrationPlan 構築 + プレビュー描画 + 単体テスト)— AC-4/5/6/11 の分類面
3. 移行実行 + CLI 配線(git 前提チェック、確認 UX、実行、v3 manifest、v2 upgrade 連鎖、レポート、fail-closed メッセージ更新)+ e2e — AC-3/4/5/6/7/8/9/10/11
4. エッジ強化 e2e(全面改変 fixture、unmanaged 引き継ぎ、dirty git 中断、部分失敗、--dry-run、直後 no-op)
5. ドキュメント同期(README 移行節、recipes、スペック FR-8 追補、AGENTS.md map、tech-debt RESOLVED 化)

各スライスは implementer 委譲、1 スライス 1 コミット。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh` 全ゲート
- Spec compliance criteria to confirm: FR-6(検出 → プレビュー → 確認 → 実行 → レポート、git クリーン前提)、FR-7(分類規則)、FR-8(特別面 — 追補後の記述と一致)、L5 契約(AC-1)。Phase 5 スコープ(eject/adopt/doctor/ガード)は除外明記
- Documentation drift to check: README / recipes の移行記述、スペック追補の整合、AGENTS.md map、tech-debt 行
- Evidence to capture: verify レポートに AC 対応表

## Test plan

- Unit tests: 分類器の表駆動(未改変/改変/unmanaged × 再配置/廃止/不変/特別面/パック)、owner 考慮 untracked 分類、プレビュー描画
- Integration tests: レガシー fixture(v1 世代レイアウトを埋め込み FS モックで再現)→ 移行 → 全 AC 検証 e2e。全面改変 fixture、パック付き fixture
- Regression tests: 既存スイート全 green(resolver nil 互換含む)
- Edge cases: dirty git / 非 git ターゲット、`--yes` / `--dry-run`、旧 manifest の空ハッシュエントリ(v1 heal 対象だったもの)、旧パスと新パスが両方existing、シンボリックリンク(既存ガードの適用確認)、移行直後の upgrade no-op
- Evidence to capture: test レポートにカバレッジと AC 対応

## Risks and mitigations

- 移行は複合ファイル操作でバグの影響が大きい → git クリーン前提で全復元可能 + プレビュー/確認 + `--dry-run`。v3 manifest 書き込みをバリアとし、部分失敗の再実行は「レガシーのまま」なので再移行可能
- 旧レイアウトの多様性(v1 heal 対象の空ハッシュ、途中版の manifest)→ 旧 `ComputeDiffs` が扱っていたケースをエッジテストに引き継ぐ
- fork 新パス移設の意味論が下流の予想と違うリスク → 移行レポートに fork 一覧 + diff + 「advisory が出続ける」説明を明記
- 確認プロンプト導入による Phase 3 非対話原則との緊張 → 移行時のみ・`--yes` 提供・設計判断に記録

## Rollout or rollback notes

- リリースタグは Phase 5 まで発行しない。移行コードは下流未到達のまま Phase 5 の系列リリースに同梱される
- スライス単位 revert 可能

## Open questions

- ~~移行分類器の配置(internal/cli vs internal/upgrade)— スライス 2 で確定(所有マップへの依存方向で決める)~~ **解決(スライス 2)**: `internal/cli/migrate.go` に配置。分類器は `ownerForScaffoldPath`(CLI 層、`internal/cli/init.go`)と `scaffold`/`upgrade` 両方に依存する必要があり、`internal/upgrade` は CLI 層の所有マップに依存できない(パッケージ境界が逆転する — `internal/upgrade/replaceplan.go` の `ReplaceOptions.OwnerForPath` のドキュメント参照)。再配置判定(`.claude/rules/<name>.md` → `.claude/rules/ralph/<name>.md`)は、呼び出し側が組み立てる desired state(インストール済みパックを含む)への単一のメンバーシップ確認(`relocatedRulePath`)で機械的に行い、別途「インストール済みパック名一致」チェックは不要(desired 側にパックのルールファイルが既に含まれる設計のため)。

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created(feat/overlay-scaffold-v2-p4)
- [x] Implementation started (slices 3ccf4bf/0e492b8/66ab80d/8daee5d + docs slice)
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] Sync-docs artifact created
- [ ] PR created
