# Walkthrough: overlay-scaffold-v2 Phase 4(旧レイアウト確認付き自動移行)

- Date: 2026-08-18
- Branch: feat/overlay-scaffold-v2-p4
- Diff: 23 files, +6,309 / -62(コミット 30 件)
- Plan: docs/plans/archive/2026-08-18-overlay-scaffold-v2-p4.md(PR 時にアーカイブ)
- Spec: docs/specs/2026-08-17-overlay-scaffold-v2.md FR-6/7/8 + FR-8 Phase-4 追補(Phase 1 = #143、Phase 2 = #144、Phase 3 = #145)

## 読む順番(レビュー導線)

1. **internal/cli/migrate.go(新規 ~1,800 行、本丸)** — レガシー manifest(v1/v2、`meta.layout` なし)から v2 レイアウトへの確認付き一括移行。フロー: git クリーン前提 → `ClassifyMigration`(パスごと分類)→ グループ化プレビュー → y/N 確認(`--yes` で省略)→ 全 op プリフライト(パス封じ込め+symlink 検証、失敗時ゼロ書き込み)→ 実行 → v3 manifest 構築 → 連鎖 `runUpgradeV2` → レポート `docs/reports/ralph-migration-<date>.md`。
2. **分類規則(FR-7 + FR-8 追補)** — `LegacyEntryState`(disk_hash 優先、partial/空 hash は改変扱い)を基礎に: 未改変は置換/移設/削除(owner=seed の同一パスは移行側で `OpReplaceWithTemplate` — 連鎖 upgrade の classifySeed は既存 seed を書かないため)、改変済み/unmanaged は自動 fork 保全(`OpForkRelocate` / `OpForkInPlace`)。特別面: CLAUDE.md(未改変→最小シード置換 / 改変→不触)、AGENTS.md・.gitignore(未改変→block 置換 / 改変→据え置き+連鎖 block 追記)、settings.json(レガシー直接 hook 8 コマンドの exact-match 剪定 `OpSettingsPrune`、near-miss は残置+レポート案内)、`.codex/AGENTS.override.md`(常に seed 再帰属)。
3. **衝突マトリクスと再実行安定性** — 移設先既存時: (a) 内容=ソース → 旧パス削除のみ(改変ソースは `OpDeleteOldPathAdoptFork` で fork 帰属維持)、(b) 内容=新テンプレート(未改変ソースのみ)→ 旧パス削除、(c) 発散 → ゼロ書き込み中断+衝突レポート。中断後の再実行は `classifyRerunRelocatedDestination` が半移行状態を吸収。
4. **保全経路(NFR-2)** — unavailable pack は payload(`packs/languages/<pack>/`)・v2 rule・レガシー rule(`.claude/rules/<pack>.md`、`legacyPackRuleRelPath`)の 3 箇所とも preserve し、v3 manifest へ明示 carry-forward。未追跡 seed 衝突パスは v3 manifest から意図的に除外し、連鎖 upgrade の `classifyUntracked` が advisory を発火(Phase 3 Known gap の解消、リリース前必須)。
5. **プリフライト安全性** — `validateMigrationOp` が全 op 種で `ValidateRealParentChain` + 葉 Lstat を単一ディスパッチ点で強制。削除系は OldPath に加え、移設先 NewPath(plain delete は write-target 検証、AdoptFork は mustExist=true の採用先検証)も削除前に検証。
6. **周辺配線** — `internal/cli/upgrade.go`(レガシー検出 → 移行分岐 + `--yes`)、`internal/cli/init.go` / `pack.go`(owner 付与の整合)、`internal/upgrade/replaceplan.go`(`OwnerForPath` フック + classifyUntracked の seed 分岐)。

## 品質ゲートの履歴

パイプライン 3 サイクル(cap 2→3、引き上げは操作者承認):

| cycle | self-review | verify | test | cross-review(codex) |
|---|---|---|---|---|
| 1 | **HIGH 1**(再実行時の fork 帰属喪失 → `OpDeleteOldPathAdoptFork` 新設)+ MEDIUM 4 + LOW 8 → 修正 | PASS | PASS(599 shell + Go 8/8) | P2 3 件(AdoptFork NewPath 検証 / seed advisory 迂回 / fork diff レポート漏れ)→ 修正 |
| 2 | MEDIUM 1 + LOW 3 → 修正 | PASS | PASS | **P1 1 件 + P2 2 件**(未改変 seed の FR-7 置換欠落 / unavailable pack 削除 / 移設 delete の NewPath 未検証)→ cap 引き上げ(操作者承認)後修正 |
| 3 | **HIGH 1**(unavailable pack のレガシー rule パス削除)+ MEDIUM 3 + LOW 3 → 修正 | PASS(seed 全置換の FR-7 準拠判定含む) | PASS | P2 1 件 → **Known gap 化(操作者承認、tech-debt 記録済み)** |

最終: shell 599/599(25 ファイル)、Go 8/8、カバレッジ cli 78.7% / upgrade 91.2%。migrate.go の回帰テストは self-review/cross-review 所見ごとに対応(migrate_test.go ~2,900 行、テスト関数 202)。

## Known gap(操作者承認済み)

移設先が旧内容で既に存在する settled ケース (a) では v3 manifest に新テンプレートハッシュが記録され、テンプレート進化時に連鎖 upgrade が「未解決 drift」として据え置く(自動収束しない)。データ損失なし・レポート+exit 3 で可視化・`ralph adopt` 一発で解消。詳細: docs/tech-debt/README.md 最終行、docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md(Cycle 3, WC#1)。
