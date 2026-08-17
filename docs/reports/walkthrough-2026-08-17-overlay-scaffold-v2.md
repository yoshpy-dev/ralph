# Walkthrough: overlay-scaffold-v2 Phase 1(スペック + エンジン基盤)

- Date: 2026-08-17
- Branch: feat/overlay-scaffold-v2
- Diff: 27 files, +4,438 / -22(コミット 24 件)
- Plan: docs/plans/archive/2026-08-17-overlay-scaffold-v2.md(PR 時にアーカイブ)
- Spec: docs/specs/2026-08-17-overlay-scaffold-v2.md

## 読む順番(レビュー導線)

1. **docs/specs/2026-08-17-overlay-scaffold-v2.md** — 全 5 フェーズの設計全体。この PR は Phase 1 のみ。5 層オーバーレイ(core 全置換 / thin entry + managed block / user overlay 不可侵 / pin+receipt / seed-once)と、対話型コンフリクト解消の廃止方針。
2. **docs/plans/active → archive の 2026-08-17-overlay-scaffold-v2.md** — Phase 1 の AC(AC-1〜AC-9)と Codex プランアドバイザリ 6 件の反映内容。
3. **internal/scaffold/manifest.go** — manifest v3(`meta.layout`、`owner`、`forked_from_version`)。書き込みはオプトイン API(`SetLayoutV2`/`SetFileOwned`/`SetFileFork`)に隔離し、既存セッターの出力は不変(AC-8)。v1/v2 読み込みは owner を確定させず legacy 保持(AC-1)。
4. **internal/scaffold/paths.go** — 共有パスバリデータ `CleanLocalRelPath`(旧 `cleanTemplateRelPath` の共通化、AC-9)。
5. **internal/upgrade/replaceplan.go** — 中核。所有権ベースの分類(core/fork/seed/block/legacy/untracked)→ 順序付き操作プラン(delete→create→update)。fork 抑止、fork 記録なし改変の「未解決 drift」(非破壊)、`ManifestRemove`(テンプレート削除パスの manifest 掃除信号)、`ApplyOps` の validate-all-upfront + 先頭失敗停止(コミットバリア: manifest 前進は呼び出し側が nil error 後にのみ行う契約)。
6. **internal/upgrade/block.go** — managed block エンジン(純関数)。マーカー厳密一致、block 外バイト保全、欠落時末尾追記、破損時は Content=nil で書き込み不能に。CRLF/末尾改行なし対応。
7. **internal/upgrade/settingsmerge.go** — settings.json の 3-way マージ(current/旧所有テンプレート/新所有テンプレート + 所有 JSON パス)。順序保持 JSON モデル、所有配列のエントリ単位 3-way、ユーザキー保全、決定的出力。
8. **internal/upgrade/advisory.go / report.go** — advisory diff(「ディスク vs 新テンプレート」定義)と upgrade レポートの markdown 描画・書き込み(docs/reports/ 配下強制)。
9. **AGENTS.md / scripts/check-sync.sh / docs/tech-debt/README.md** — repo map 追従、KNOWN_DIFFS(Phase 2 で解除予定、tech-debt 記録済み)、既知ギャップの登録。

## 品質ゲートの履歴

- パイプライン 2 サイクル実行(cap 2/2)。cycle 1: self-review MEDIUM 3 件修正 → verify PASS → test PASS(555 shell + 8 Go pkgs)→ cross-review(codex)3 件 → 修正(1ef5be7)。cycle 2: self-review C2-1〜C2-8 対処(3b32c50)→ verify PASS → test PASS(coverage upgrade 89.9% / scaffold 77.7%)→ cross-review 残 1 件(下記)。
- 各レポート: docs/reports/{self-review,verify,test,sync-docs,cross-review-triage}-*overlay-scaffold-v2.md(Cycle 2 節付き)

## Known gaps

- テンプレートパスの file⇔directory 形状遷移で `PlanCoreReplace` が非破壊エラー中断する(codex cross-review cycle 2 P2)。分類設計(形状不一致→drift 等)は Phase 3 の配線時に対処。docs/tech-debt/README.md 記録済み。
- cycle-1 self-review の未修正 LOW 5 件(可読性/一貫性)— docs/tech-debt/README.md にバッチ登録、トリガーは Phase 3。

## 付随修正(スコープ外の発見)

- tests/test-gc-artifacts.sh の日付依存 fixture(固定日付が --days 30 窓を外れ main でも fail)を実行日相対に修正(7dca14e)。
