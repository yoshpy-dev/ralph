# Walkthrough: overlay-scaffold-v2 Phase 3(upgrade 非対話化 + 対話エンジン/baseline 撤去)

- Date: 2026-08-18
- Branch: feat/overlay-scaffold-v2-p3
- Diff: 37 files, +5,881 / -4,198(コミット 39 件)
- Plan: docs/plans/archive/2026-08-18-overlay-scaffold-v2-p3.md(PR 時にアーカイブ)
- Spec: docs/specs/2026-08-17-overlay-scaffold-v2.md(Phase 1 = #143、Phase 2 = #144)

## 読む順番(レビュー導線)

1. **internal/cli/upgrade_v2.go(新規、シリーズの本丸)** — v2 レイアウトへの完全非対話 upgrade。フロー: settings スナップショット先行キャッシュ → desired state 合成(base + packs、利用不能パックは preserve)→ 置換プラン(settings/block/スナップショットは skip)→ ApplyOps → block 更新 → settings 3-way マージ → スナップショット後書き(2 段階)→ advisory → レポート → manifest 再構築(コミットバリア)→ baseline 掃除 → git hooks。収束時は完全 no-op(書き込みゼロ、stdout 通知のみ。seed advisory / version 変化が veto)。exit code: 0 成功 / 1 エラー / 3 drift 残存完走。
2. **internal/upgrade/replaceplan.go の拡張** — `PlanCoreReplaceDesired`(マップ入力)、`ReplaceOptions{SkipPaths, PreservePrefixes}`、`ApplyOps` の Lstat プリフライト(葉 + 親チェーン `ValidateRealParentChain` — symlink 封じ込め)。
3. **settings スナップショット機構**(`internal/upgrade/snapshot.go` + `templates/base/.ralph/core/settings.ralph.json`)— 3-way マージの oldOwned をリポジトリ内 core ファイルとして自己更新。欠落時は `{}` フォールバック(非破壊劣化、レポート明記)。
4. **撤去(-4,100 行)** — 対話型コンフリクト解消(`resolveConflict` 系・`diff.go`・`merge.go`)、baseline 機構(`baseline.go`、manifest の baseline 書き込み系、init の baseline)、`--force` フラグ、レガシー upgrade テスト 34 本(棚卸し分類は self-review レポート参照)。レガシー manifest への upgrade/pack add/再 init は Phase 4 移行を案内する fail-closed。
5. **所有分類の補正** — `.codex/AGENTS.override.md` → seed(スペック L3 準拠、cycle-3 cross-review)。
6. **cmd/ralph/main.go** — `ErrUpgradeDriftRemaining` → exit 3 の配管。

## 品質ゲートの履歴

パイプライン 4 サイクル(cap 2→3→4、各引き上げは操作者承認):

| cycle | self-review | verify | test | cross-review(codex) |
|---|---|---|---|---|
| 1 | MEDIUM 5 + LOW 5 → 修正 | PASS | PASS | P1 2 件(symlink 親チェーン / 例外面ガード)→ 修正 |
| 2 | LOW 4 → 修正 | PASS | PASS | P2 2 件(dry-run 例外面予告 / no-op 書き汚し)→ 修正 |
| 3 | **HIGH 1**(no-op 短絡の seed advisory 隠蔽)+ MEDIUM 1 → 修正 | PASS | PASS | P2 1 件(.codex override の L3 分類漏れ)→ 修正 |
| 4 | MEDIUM 1(テスト空洞化)+ LOW 4 → 修正 | PASS | PASS | P2 1 件 → **Known gap 化(操作者承認)** |

最終: shell 25 ファイル全 green、Go 8/8、カバレッジ cli 76.4% / scaffold 75.7% / upgrade 91.1%(レガシー 4,100 行削除込み。所有系関数は 100% 維持)。

## Known gaps(docs/tech-debt/README.md 追跡)

- **新規 seed パス衝突の誤 drift 分類**(cycle-4 cross-review、承認済み deferral)— 将来リリースが seed パスを新規追加し下流に同パス既存ファイルがある場合、untracked 分類が owner 非考慮で恒久 exit 3 になる。発火にはリリース(Phase 5 後)が必要で現露出ゼロ。Phase 4 移行分類器で owner 考慮分類として対処(リリース前必須とトリガー明記)
- 既存 v2 manifest(本ブランチ内テスト由来のみ)の `.codex/AGENTS.override.md` owner=core 残存 — Phase 4 で再帰属
- レガシーテスト削除に伴う 3 面のカバレッジ喪失(git-hook chaining / --pager / AvailablePacks 失敗分岐)

## 過渡状態の注記

Phase 3 完了時点: v2 レイアウト = フル機能の非対話 upgrade / レガシーレイアウト = fail-closed(Phase 4 が移行を提供)。リリースタグは系列完了(Phase 5)まで発行しない。
