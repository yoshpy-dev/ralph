# overlay-scaffold-v2 — Phase 3: upgrade 非対話化配線 + 対話エンジン/baseline 撤去

- Status: Draft
- Owner: Claude Code
- Date: 2026-08-18
- Related request: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 3 / 全 5 段。Phase 1 = PR #143、Phase 2 = PR #144 マージ済み)
- Related issue: N/A
- Type: feat
- Branch: feat/overlay-scaffold-v2-p3

## Objective

`ralph upgrade` を Phase 1 のエンジンプリミティブ(置換プランナ + コミットバリア / managed block / settings 3-way マージ / advisory diff / レポート)に配線し、v2 レイアウトに対する完全非対話の upgrade を実現する(スペック FR-1)。同時に、対話型コンフリクト解消エンジンと baseline 機構を完全撤去する(FR-13)。これによりシリーズの中核目標 — 「下流カスタマイズに触れない、PR レビュー可能な upgrade」— が動作可能になる。

## Scope

**v2 upgrade フローの配線(`internal/cli/upgrade.go` 全面書き換え)**

v2 manifest(`meta.layout = "v2"`)に対する `ralph upgrade` は次を非対話で実行する:

1. **desired state の構築**: base テンプレート FS + `manifest.Meta.Packs` の各パック(payload は `packs/languages/<lang>/`、rule は `.claude/rules/ralph/<lang>.md` へ remap)を単一の「パス → 内容」マップに合成。`PlanCoreReplace` はこのマップを受けられるよう入力を拡張する(fs.FS 直接入力は薄いアダプタとして維持)。**利用不能なパックは preserve 機構で保護する**: 該当パックの名前空間(payload prefix + rule パス)を「テンプレート不在でも削除・drift 分類しない preserve プレフィクス」としてプランナに渡し、warning + ディスク/manifest エントリ完全保全をテストで保証(Phase 1 プランナは template 不在の core を削除候補に分類するため、明示機構がないと AC-9 が成立しない)
2. **例外面の除外**: `.claude/settings.json`(3-way マージ対象)と block 面(`AGENTS.md`・`.gitignore`)はプランナの置換対象から除外(skip paths)し、専用処理へ
3. **core 置換**: `PlanCoreReplace` → `--dry-run` はプラン表示のみ / 通常実行は `ApplyOps`(バリデート済み・先頭失敗停止)。**`ApplyOps` に Lstat プリフライトを追加**: 全 write/delete 対象を事前に `os.Lstat` し、symlink・非 regular ファイルが 1 つでもあれば書き込みゼロでエラー(Phase 2 で init 側に入れた封じ込めガードの ApplyOps 版。symlink/ディレクトリ標的のテスト付き)
4. **block 更新**: テンプレートの block 内容(AGENTS.md は `.ralph/core/AGENTS.core.md`、.gitignore はテンプレート block 内部)で `UpdateManagedBlockStyled`。unchanged → 無書き込み、malformed → 据え置き + レポート
5. **settings 3-way マージ(2 段階スナップショット)**: `MergeOwnedSettings(current, oldOwned, newOwned)`。oldOwned は新設スナップショット `.ralph/core/settings.ralph.json` を **いかなる書き込みよりも前に読み込みキャッシュ**する。スナップショットは core 置換(`ApplyOps`)の対象から**除外**し、settings マージ成功後に新テンプレート内容で明示的に書き込む(部分失敗時に oldOwned が失われない 2 段階更新)。スナップショット欠落時(Phase 2 init 世代)のフォールバックは `{}`: このとき「stale な旧 ralph 所有エントリの自動削除」は機能せず、既存エントリは全てユーザ扱いで保全される(非破壊側への劣化)。フォールバック使用はレポートに明記し、重複/残存 hooks の挙動をテストで固定する
6. **seed / advisory**: seed 欠落は生成、テンプレート側変更は advisory diff(ディスク vs 新テンプレート)としてレポートへ。fork(eject 済み)も同様(Phase 5 まで eject コマンドはないが、manifest 上の fork エントリは尊重)
7. **レポート + manifest バリア**: `docs/reports/upgrade-<version>-<date>.md` を出力し、全工程成功後にのみ manifest を再構築して書き込み(`ManifestRemove` 消費、refresh 適用、`SetLayoutV2` + owner 再付与)。途中失敗時は manifest 不変・再実行で完走(Phase 1 の分類安定性を利用)
8. **未解決 drift**: 該当パスは据え置き、レポート + stderr 通知。**exit code は 3(「完走したが未解決 drift 残存」の専用コード)** — スペック FR-4「exit code で警告する」に準拠し、Phase 5 の `doctor --strict` 到着前も機械検知可能にする。成功 = 0、実行エラー = 1、drift 残存完走 = 3 を CLI ヘルプに明記
9. **後始末**: 成功時に旧 `.ralph/baseline/` ディレクトリを削除

**フラグ**: `--dry-run`(プラン + advisory 予定の表示)/ `--diff`(advisory diff 全文表示、既存 colorize/pager 再利用)を新エンジンで維持。`--force` は削除(FR-3。fork 再採用は Phase 5 の `adopt`。それまでの手動復旧経路は「ディスクを新テンプレート内容に合わせる → refresh 収束」)

**レガシーレイアウトの扱い**: v1/v2(non-layout)manifest への upgrade は fail-closed(「旧レイアウトの移行は Phase 4 で提供」の明確なエラー、書き込みゼロ)。Phase 2 の v2 ガードと対に、旧エンジン削除後の唯一の経路とする

**対話エンジン + baseline の撤去**

- `internal/cli/upgrade.go`: `resolveConflict` / `resolveConflictWithBaseline` / conflict marker 編集 / `editContentBytes` / 確認プロンプト系を削除
- `internal/upgrade/`: `diff.go`(ComputeDiffs 系)・`merge.go`(PlanMerge 3-way)と各テストを削除。`unified_diff.go` / `colorize.go` は advisory 表示で継続利用
- `internal/scaffold/baseline.go` 削除、manifest の baseline 書き込み経路(`SetFileWithBaseline` / `SetFileResolvedWithBaseline` / `addBaselineOnly` 等)を非 baseline 系へ置換。読み込み互換(既存 manifest の baseline フィールド許容)は維持
- `ralph init`(Phase 2 実装)からも baseline 書き込みを除去
- レガシー upgrade テスト群(`clearManifestLayoutV2` 依存の約 28 本)を v2 エンジンの等価テストへ置換/削除

**テンプレート追加**

- `templates/base/.ralph/core/settings.ralph.json`(settings スナップショット。内容はテンプレート settings.json の所有部と同一)+ init での描画(owner=core は catch-all で自動)

## Non-goals

- 旧レイアウト→v2 の移行ロジック(Phase 4)
- `eject` / `adopt` / `doctor --strict` / `status` 所有表示 / 混入ガード(Phase 5)
- `.codex/` hooks の dispatcher パリティ(Phase 5)
- リリースタグ発行(系列完了まで)
- ファイル⇔ディレクトリ形状遷移の分類対応(tech-debt 登録済み。本フェーズではエラー UX を「明確なエラーメッセージで中断」に整えるに留め、分類設計は Phase 4 の移行分類器と合流)

## Assumptions

- Phase 1 プリミティブの API は配線に十分(必要な軽微調整 — desired-state マップ入力等 — は本フェーズのスコープ内)
- リリースタグ未発行のため、レガシー manifest への fail-closed は下流に未到達(Phase 4 が移行を提供してから系列リリース)
- `.claude/settings.json` の manifest owner は Phase 2 時点で core(catch-all)だが、Phase 3 の skip-paths + 専用マージにより全文置換は起こらない(owner 値の再分類は行わず、置換対象からの除外で対応 — manifest スキーマ変更を避ける)
- settings スナップショット欠落時の `{}` フォールバックは 1 世代のみの非破壊的な劣化(stale 所有エントリの自動削除が効かないだけ)

## Affected areas

- `internal/cli/upgrade.go`(全面書き換え)、`internal/cli/init.go`(baseline 除去)、`internal/cli/language_pack.go`(desired-state 合成への追従)
- `internal/upgrade/`(replaceplan 入力拡張、diff.go/merge.go 削除)
- `internal/scaffold/`(baseline.go 削除、manifest baseline 書き込み系の整理)
- `templates/base/.ralph/core/settings.ralph.json`(新規)
- テスト全域(レガシー upgrade テスト置換、新 e2e)
- docs(AGENTS.md repo map、スペックの Open question 解決追記、README/recipes の upgrade 記述)

## Design decisions

- 【スペック Open question の解決 — Codex 所見 5 で修正】upgrade の exit code: 成功 = 0 / 実行エラー = 1 / **未解決 drift 残存で完走 = 3**。スペック FR-4 の「exit code で警告」に準拠し、doctor --strict(Phase 5)到着前の機械検知を確保。スペックの Open questions 節にこの決定を追記する(スライス 5)
- settings の oldOwned は `.ralph/core/settings.ralph.json` スナップショット方式。バイナリにはテンプレートが 1 世代しか埋め込まれないため、「前回適用時のテンプレート内容」はリポジトリ内に core 所有ファイルとして残すのが唯一のオフライン成立解
- 【Codex 所見 1 で修正】スナップショット更新は 2 段階: (1) 全書き込み前に oldOwned を読み込みキャッシュ、(2) スナップショットを ApplyOps から除外、(3) settings マージ成功後に新内容を明示書き込み。スナップショット書き込み後〜settings 書き込み前の失敗注入テストを必須とする
- 【Codex 所見 2/3 で追加】プランナに preserve プレフィクス機構(利用不能パック名前空間の削除・drift 分類抑止)、ApplyOps に Lstat プリフライト(symlink/非 regular 拒否)を追加する — いずれも Phase 1 API のスコープ内拡張
- レガシー manifest は Phase 3 で fail-closed(旧エンジン全削除のため)。「動く旧エンジンを残して段階削除」より、未リリース期間中に一括で削除する方が中間状態の維持コストがない(Phase 2 の単一メジャー移行方針と整合)
- `--force` は削除(FR-3)。fork の再採用経路は Phase 5 `adopt` まで手動(ディスクを新テンプレートに一致させる → refresh)
- Critical forks: なし(上記はいずれもスペック既定・フェーズ順序・オフライン制約から一意に定まる技術判断。ユーザ判断を要する製品トレードオフはない)

## Acceptance criteria

- [x] AC-1 v2 プロジェクトへの `ralph upgrade` が非対話で完走する: core 置換(パック含む)、block 内のみ更新(block 外バイト保全)、settings マージ(ユーザ permissions 保全 + ralph 所有キー更新 + stale 所有エントリ削除)、seed 欠落生成、advisory 収集、レポート出力、manifest 再構築(ManifestRemove/refresh 反映)
- [x] AC-2 いかなる分岐でもプロンプト/stdin 読み取りが発生しない(対話コードパスの不存在をコードレベルで確認)
- [x] AC-3 `.claude/settings.json` が全文置換されない: ユーザ変更ありの settings + テンプレート変更ありの状況で、マージ結果のみが書かれることをテストで保証
- [x] AC-4 未解決 drift のパスは 1 バイトも変更されず、レポートと stderr に列挙され、exit 3(専用コード。drift なし成功は exit 0)
- [x] AC-4b `ApplyOps` は write/delete 対象に symlink・非 regular ファイルが含まれる場合、書き込みゼロでエラーになる(テストで保証)。cycle 2 で `ValidateRealParentChain` を追加し、親ディレクトリが symlink の場合(葉ノードのみの Lstat では検知不能なケース)も同じく書き込みゼロでエラーになることをテストで保証(cross-review AR#1 の fix、`fc8e9a9`)
- [x] AC-4c settings スナップショットの 2 段階更新: スナップショット書き込み後〜settings 書き込み前の失敗を注入しても、再実行で stale 所有エントリ削除が正しく機能する(oldOwned 喪失なし)。qualification: `TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced` は root 実行の CI では自己スキップする既知の制約(このフェーズの回帰ではない)
- [x] AC-5 冪等性: 同一バージョンでの再実行が書き込みゼロの no-op(レポートに no-op 明記)
- [x] AC-6 部分失敗: 途中失敗で manifest 不変、再実行で残工程が完走(分類の安定性含む)
- [x] AC-7 レガシー manifest への upgrade は書き込みゼロで fail-closed し、Phase 4 の移行を案内する
- [x] AC-8 対話エンジン・baseline の完全撤去: `resolveConflict` / `PlanMerge` / `ComputeDiffs` / `baseline` 書き込みへの参照がゼロ。成功した v2 upgrade が旧 `.ralph/baseline/` を削除する
- [x] AC-9 パック: installed packs の payload + rule が同一プランで更新され、利用不能パックは preserve プレフィクスにより warning + ディスク/manifest エントリ完全保全(削除も drift 分類もされないことをテストで保証)。cycle 2 で settings.json / settings スナップショット / upgrade レポートの例外面書き込み(ApplyOps 非経由)にも同じ containment チェックを追加(cross-review AR#2 の fix、`fc8e9a9`)
- [x] AC-10 `--force` が存在せず、`--dry-run` / `--diff` が新エンジンで機能する(dry-run は書き込みゼロ)
- [x] AC-11 新規 `ralph init` が baseline を書かず、settings スナップショットを含む。スナップショット欠落プロジェクト(Phase 2 init 世代)の初回 upgrade は `{}` フォールバックで成功し、(a) stale 所有エントリが削除されず保全されること(劣化の正確な形)、(b) フォールバック使用がレポートに記録されることをテストで保証
- [x] AC-12 block 面の managed block が malformed の場合、据え置き + レポート記載で upgrade 自体は完走する

## Implementation outline

1. エンジン入力拡張 + desired-state 合成(base + packs + rule remap、settings/block の skip-paths)+ settings スナップショットのテンプレート/init 追加 + テスト
2. v2 upgrade フロー配線(プラン → apply → block → settings → seed/advisory → レポート → manifest バリア → baseline 掃除、--dry-run/--diff)+ e2e テスト(AC-1〜6, 9, 10, 12)
3. レガシーエンジン + baseline 撤去(diff.go/merge.go/baseline.go、CLI 対話系、init の baseline、レガシーテスト置換、legacy fail-closed)+ AC-7/8/11 テスト
4. 冪等性・部分失敗・境界の追加 e2e(シミュレートした旧テンプレート状態からの upgrade、drift/fork/seed/block 混在シナリオ)
5. sync-docs(スペック Open question 解決の追記、AGENTS.md repo map、README/recipes の upgrade 記述、`--force` 削除の反映)

各スライスは implementer 委譲、1 スライス 1 コミット。スライス 2 と 3 は依存が強いため、3 完了までは旧エンジンを残したまま新経路を並存させ、スライス 3 で切り替え+削除する(スライス 2 の時点では v2 経路のみ新エンジン)。

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(golang + shell、check-sync/check-skill-sync/check-pipeline-sync 含む)
- Spec compliance criteria to confirm: FR-1(工程 1-8)、FR-3(--force 削除)、FR-4(drift 非破壊)、FR-5(block 更新)、FR-13(撤去)、NFR-1(冪等)、NFR-2(非破壊)、NFR-4(ネットワーク非依存)。Phase 4/5 スコープの FR は明記して除外
- Documentation drift to check: AGENTS.md repo map(diff.go/merge.go/baseline 削除、upgrade 記述の刷新)、スペックへの exit-code 決定追記、README / docs/recipes の upgrade・--force 記述
- Evidence to capture: verify レポートに静的解析ログ要約 + AC 対応表

## Test plan

- Unit tests: desired-state 合成(packs/rule remap/skip-paths)、スナップショット読み込みとフォールバック、manifest 再構築(ManifestRemove/refresh/owner 再付与)
- Integration tests: 一時ディレクトリ e2e — v2 init → テンプレート変更シミュレーション(embedded FS のモック差し替え)→ upgrade → 全面検証(AC-1)。ユーザ編集混在(block 外・settings・seed・drift・fork)シナリオ、冪等再実行、部分失敗(書き込み不能注入)→ 再実行完走
- Regression tests: 既存スイート全パス(レガシー upgrade テストは v2 等価に置換)、Phase 1/2 のプリミティブ・init テスト非退行
- Edge cases: settings スナップショット欠落、malformed block、パック利用不能、`.ralph/baseline/` 存在/不在、レガシー manifest、`--dry-run` の書き込みゼロ、report ディレクトリ欠落
- Evidence to capture: `docs/reports/test-*.md` にカバレッジと AC 対応

## Risks and mitigations

- settings.json の owner=core による全文置換事故(最重要)→ skip-paths を AC-3 の専用テストで固定。プランナ入力構築時に settings/block パスの除外をアサート
- レガシーテスト大量置換に伴う保護の穴 → 置換は「削除ではなく v2 等価への書き換え」を原則にし、スライス 3 のレビューで消えたアサーションの棚卸しを要求
- 旧エンジン削除とスライス並行の中間状態 → スライス 2 では旧エンジンを温存(v2 経路のみ新設)し、スライス 3 で一括削除。スライス単位 revert 可能
- 形状遷移(file⇔dir)での plan 中断(既知 tech-debt)→ 本フェーズはエラーメッセージ整備のみ、分類は Phase 4
- スナップショット `{}` フォールバックの意味論(stale 削除 1 世代欠落)→ Assumptions に明記、レポートに fallback 使用を記録

## Rollout or rollback notes

- リリースタグは切らない。main 上の過渡状態(レガシー fail-closed)は下流未到達
- スライス 3(削除)は単独 revert で旧エンジン復元可能(スライス 2 までは並存構造)

## Deviations

- スライス 3 で発見: v2 upgrade フローが `installManagedGitHooks` を呼ばず、テンプレート変更時に git hooks が再インストールされない(レガシーエンジンは呼んでいた)。スライス 4 でクローズ済み: v2 成功パスに `installManagedGitHooks` 呼び出しを追加し、部分失敗カバレッジも強化(コミット 5b1b675)
- スライス 3 の判断: `packPrefixFor` 等は language_pack.go へ移設、`runUpgrade`/`runUpgradeIO` ラッパは呼び出し元消滅により削除、`addPack` の冗長 manifest 二重読みを簡約

## Open questions

- ~~desired-state マップ入力の具体 API 形状~~ → スライス 1 で確定: `PlanCoreReplaceDesired(m, targetDir, desired map[string][]byte, opts ReplaceOptions)` が主入口、`PlanCoreReplace`(fs.FS)は薄いアダプタ。`ReplaceOptions{SkipPaths, PreservePrefixes}`、preserve は「テンプレート完全不在の名前空間のみ保護、desired に内容があれば通常分類」の意味論
- report の `--diff` 表示と docs/reports への full-diff 記録の分担 — スライス 2 で確定

## Progress checklist

- [ ] Plan reviewed
- [x] Branch created(feat/overlay-scaffold-v2-p3)
- [x] Implementation started(スライス 1〜5: 104cc7b / e08ebe7 / c834f44 / 5b1b675 + 本スライス)
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] Sync-docs artifact created
- [ ] PR created
