# Walkthrough: overlay-scaffold-v2 Phase 5(eject/adopt + doctor --strict + status 所有表示 + 混入ガード + Codex hooks パリティ)

- Date: 2026-08-19
- Branch: feat/overlay-scaffold-v2-p5
- Diff: 41 files, +6,107 / -189(コミット 45 件)
- Plan: docs/plans/archive/2026-08-19-overlay-scaffold-v2-p5.md(PR 時にアーカイブ)
- Spec: docs/specs/2026-08-17-overlay-scaffold-v2.md FR-2/3/9/10/12(Phase 1 = #143、Phase 2 = #144、Phase 3 = #145、Phase 4 = #146 — 本 PR で系列完了)

## 読む順番(レビュー導線)

1. **internal/cli/eject.go / adopt.go(新規、FR-2/3)** — 所有遷移コマンド。eject: core(または未解決 drift)→ fork 記録(`forked_from_version`+ディスク実体ハッシュ、ディスク書き込みゼロ)。adopt: fork/drift → core(現行テンプレートで置換)。安全設計: git クリーン前提(migration と共通の `checkGitCleanForDestructiveOp`)、y/N 確認+`--yes`、全対象プリフライト(封じ込め+symlink 検証、1 件でも不合格ならゼロ書き込み)、v2 例外面(settings.json / snapshot / block 面)は各面の正規カスタマイズ手段へ誘導して拒否。分類は `resolveOwnershipPlan`(upgrade と同一の PlanCoreReplaceDesired ドライ実行)を共用し、eject/adopt/doctor/status の「fork/drift/retired」判定が upgrade と乖離しない構造。
2. **internal/cli/doctor.go の FR-9 検査群 + --strict** — (a) core ハッシュ(fork 除く)、(b) managed block 健在、(c) settings 所有キー、(d) コンフリクトマーカー、(e) manifest/実体整合。`--strict` で違反時 exit 1(CI ゲート)。検証不能な meta-failure(manifest パース不能、planning エラー、読めない block 面、不正 JSON settings 等)も strict-eligible — パイプライン 3 サイクルにわたる fail-open クラスの修正+warn 経路全数監査で「クラス枯渇」を verify が確認済み。
3. **internal/cli/status.go の scaffold 所有セクション(FR-12)** — パスごとの owner(core/fork/seed/block)+未解決 drift 一覧。`--json` は additive な `scaffold` キーのみ(既存 org スキーマ不変)。corrupt manifest は「unavailable(エラー表示)」に劣化して org 表示は継続(doctor がゲート、status はレポートという役割分担)。
4. **scripts/check-template-purity.sh(新規、FR-10)+ tests/test-template-purity.sh** — templates/ のメタリポ固有参照ガード。3 走査次元: 固定文字列 / 正規表現(日付付き成果物パス)/ パス(maintainer 専用面の同梱検出、例: `.claude/skills/release/`)。allowlist は並列配列(現在空 — 検出済みリーク 5 件は本 PR 内で修正済み)。verify.local.sh 経由で CI 実行。
5. **Codex hooks dispatcher パリティ + pre_bash_guard 修正** — `.codex/config.toml`(root/template byte-identical)の直接呼び出しを `ralph-dispatch.sh` 経由化(`.d` 3 層が Codex でも機能。dispatcher は stdin 無解釈透過のためペイロード形差異に非依存)。同時に `pre_bash_guard.sh` の jq 経路が実ペイロードの `.tool_input.command` を読めず常に素通しだった欠陥を修正(tech-debt 行 100 清算、実ペイロード fixture テスト付き)。
6. **FR-4 案内の整合** — upgrade の未解決 drift 案内が eject/adopt を指すよう 4 面(stderr ×3 は `writeDriftGuidanceV2` に統合、markdown レポート、doctor (a) 検査)で更新。未追跡 drift(eject/adopt が扱えないケース)は "(untracked)" 注記+手動解消案内で区別。
7. **ドキュメント/スペック** — スペック FR/NFR 全チェックボックス+系列 AC を tick(AC-10 は Codex 実発火未検証の脚注付き)、README コマンド表、AGENTS.md repo map、tech-debt 清算(行 100/103/C2-L4 RESOLVED、新規 Known gap 行)。

## 品質ゲートの履歴

パイプライン 4 サイクル(cap 2→3→4、各引き上げは操作者承認):

| cycle | self-review | verify | test | cross-review(codex) |
|---|---|---|---|---|
| 1 | **HIGH 1**(purity ガードの正規表現分岐が区切り文字バグで不動作)+ MEDIUM 8 + LOW 4 → 修正 | PASS | PASS(614 shell + Go 8/8) | P2 2 件(doctor --strict の fail-open ×2、実測付き)→ 修正 |
| 2 | MEDIUM 2 + LOW 5 → 5 修正 + 2 deferral | PASS | PASS | **P2 4 件**(fail-open 残余 3 + purity パス走査欠落)→ cap 引き上げ後修正+warn 経路全数監査 |
| 3 | MEDIUM 1 + LOW 7 → 修正 | PASS(fail-open クラス枯渇と結論) | PASS | P2 1 + P3 1(未追跡 drift 誤誘導 / config コメントのメタリポ履歴)→ cap 引き上げ後修正 |
| 4 | MEDIUM 2 + LOW 7 → 修正 | PASS(下流出力のメタリポ引用ゼロを掃引確認) | PASS | P2 1 件 → **Known gap 化(操作者承認)** |

最終: shell 617/617(27 ファイル)、Go 8/8、カバレッジ cli 80.6% / upgrade 91.2%。

## Known gap(操作者承認済み)

`adopt --all` は unavailable pack 配下の `owner=fork` パス(plan.Preserved 分類)を一覧に出さず「Nothing to adopt」と報告する。列挙されても retired 扱いの skip になるだけで最終状態は同一(誤書き込み・データ損失なし、単一パス adopt は正しい拒否メッセージ)。詳細: docs/tech-debt/README.md 最終行、docs/reports/cross-review-triage-overlay-scaffold-v2-p5.md(Cycle 4, WC#1)。

## 系列完了とリリース hand-off

本 PR で overlay-scaffold-v2 系列(全 5 フェーズ)は完了。リリースタグは maintainer が `/release` で手動発行する(スペック「## Status」参照)。残る検証ギャップは Codex project-scoped hooks の実発火確認(trust 制約、tech-debt 記録済み)のみ。
