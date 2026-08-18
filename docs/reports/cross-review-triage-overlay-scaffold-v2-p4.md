# Cross-review triage report: overlay-scaffold-v2-p4 (Phase 4)

- Date: 2026-08-18
- Plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md
- Self-review report: docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md (Cycle 2: pass)
- Verify report: docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md (Cycle 2: pass)
- Cycle-1 findings AR#1〜AR#3 は c51497e で修正済みで、cycle-2 の verify/test が契約一致を確認済み。本サイクルの 3 件はいずれも cycle-2 で新規に報告されたもので、cycle-1 所見の再発ではない。3 件とも移行経路(`internal/cli/migrate.go`)の分類・検証の取りこぼしで、triager がコードを直接読んで各主張のメカニズムを確認した(下表の Evidence 欄)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] 未改変・同一パスのレガシー seed ファイル(`ralph.toml`、`docs/recipes/*` 等)が OpKeepInPlace + owner=seed で v3 manifest に旧ハッシュのまま引き継がれ、連鎖 v2 upgrade の classifySeed は advisory を出すだけで置換しない — FR-7「未改変ファイルは新レイアウトへ置換」に反し、旧テンプレート内容が恒久採用される | 実問題(確認済み)。classifyUnmodifiedGeneric(migrate.go:553-561)は owner を見ずに一律 OpKeepInPlace を返し、classifySeed(internal/upgrade/replaceplan.go:396-420)は disk 存在時に書き込みを一切行わない。未改変(=ユーザが触っていない)なので新テンプレートへの置換は安全で、fresh init との収束性の要でもある。修正: 未改変・同一パス・owner=seed は OpReplaceWithTemplate に分類する | internal/cli/migrate.go:553-561 |
| 2 | [P2] `buildDesiredStateV2` が返す unavailable-pack の preservePrefixes を移行経路が破棄(`desired, _, _, _, err :=`)しているため、新バイナリで読めない pack の未改変ファイルが「テンプレート廃止」と誤分類され OpDeleteOldPath で削除される — 警告は「preserve する」と言いながら実際は消す | 実問題(確認済み)。ClassifyMigration(migrate.go:292)は preservePrefixes を受け取るシグネチャすら持たない。v2 upgrade 本体は同 prefix を保全するので、移行経路だけが NFR-2(非破壊性)を破る。修正: preservePrefixes を ClassifyMigration に渡し、該当 prefix 配下は OpUntouched(manifest 引き継ぎ)にする | internal/cli/migrate.go:292, 841 |
| 3 | [P2] 未改変・移設対象のルールが OpDeleteOldPath(NewPath 付き)になる経路で NewPath が検証されない — `.claude/rules/ralph` が symlink だと旧ファイル削除後、連鎖 upgrade の NewPath 生成が封じ込めガードで拒否され、その失敗は警告降格なので、exit 成功のままルールが消える | 実問題(確認済み)。validateMigrationOp(migrate.go:1004-1011)は OpDeleteOldPath では OldPath の葉しか見ず、NewPath 検証は AdoptFork のみ(cycle-1 AR#1 の修正が plain-delete 側を覆っていない)。修正: NewPath が空でない OpDeleteOldPath にも ValidateRealParentChain を適用し、削除前に拒否する | internal/cli/migrate.go:1004-1011 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cap note

`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2` の cap に到達している(本サイクルが 2/2)。このレポートの ACTION_REQUIRED 3 件の扱い(cap 引き上げ再実行 / Known gap 化して PR / 中止)は操作者判断に委ねる。

## Cycle-1 record (superseded)

Cycle-1 のトリアージ(AR#1: OpDeleteOldPathAdoptFork の NewPath 検証欠落、AR#2: 移行経路での seed 衝突 advisory 消失、AR#3: 採用 fork の diff レポート欠落 — いずれも ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0)は c51497e で修正され、cycle-2 の verify(0cc50e5)と test(d686226)で契約一致・回帰テスト green を確認済み。
