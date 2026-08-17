# Cross-review triage report: overlay-scaffold-v2-p2 (Phase 2)

- Date: 2026-08-17(cycle 2 で更新)
- Plan: docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2(cycle 2。cycle 1 の 2 件は f80e60f で解消・再指摘なし)
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md
- Self-review report: docs/reports/self-review-2026-08-17-overlay-scaffold-v2-p2.md(Cycle 2 節あり)
- Verify report: docs/reports/verify-2026-08-17-overlay-scaffold-v2-p2.md(Cycle 2 節あり)
- Implementation context summary: cycle 1 の 2 件(.ralph/local 所有・pack add 所有)は f80e60f で解消済み。cycle 2 の新規 2 件はどちらも本 PR が導入した退行で、reviewer が再現手順付きで確認している。(1) は v2 scaffold 全プロジェクトで `ralph doctor` が偽 fail する correctness 退行、(2) は既存ファイル skip だった init が symlink 越しに target 外へ書き得る path-containment 退行。cap 到達だが、既知ギャップとして出荷するには重い。

## Cycle 1 findings resolution

cycle 1 の ACTION_REQUIRED #1(`.ralph/local/**` の owner=core 誤分類 → seed へ)・#2(`ralph pack add` の owner 欠落 → v2 時 SetOwner)は f80e60f で修正済み。verify Cycle 2 節がトリアージ契約との一致を確認済み。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `ralph doctor` の hooks integrity チェックが dispatcher コマンド文字列(`./.claude/hooks/ralph-dispatch.sh SessionStart`)全体を stat し、v2 scaffold の全プロジェクトで「7 hook script(s) missing」の偽 fail になる(reviewer が fresh init + doctor で再現) | 実退行。settings.json の hooks が「コマンド + 引数」形式になったのは本 PR が初。doctor 側でコマンド文字列の実行ファイルトークンを分離して stat する小修正 + v2 init 後 doctor green のテストで塞がる | internal/cli/doctor.go(checkHooks)、templates/base/.claude/settings.json:76 |
| 2 | [P2] init の block-append(非 --force)が pre-existing `AGENTS.md`/`.gitignore` の symlink を追跡して os.WriteFile し、target 外への書き込みになり得る(従来は既存ファイル skip で発生しなかった) | 実退行(path containment)。Lstat で regular file を要求し、symlink は warn + 据え置き(malformed block と同じ非破壊姿勢)にする小修正 + テストで塞がる | internal/cli/init.go:373(reconcileBlockSurfaces) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
