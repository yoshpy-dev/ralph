# Walkthrough: repo-wide-drift-fixes

- Date: 2026-07-12
- Plan: docs/plans/archive/2026-07-12-repo-wide-drift-fixes.md
- Branch: docs/repo-wide-drift-fixes (base: main @ b75a56d)
- Diff size: 25 files, +939 / -30(うち約6割はプラン/レポート成果物)

## 読む順序(推奨)

1. `scripts/check-skill-sync.sh` — 新設の check 6(prompts/ パリティ)。
   欠落・余剰・バイト差分の3方向を再帰的に検査(cross-review 指摘で
   `-maxdepth 1` を撤廃)。テンプレートミラーとバイト同一
2. `tests/test-check-skill-sync.sh` — 新ケース H–M(欠落/差分/同一/実ツリー
   + ネストの fail/pass)。ゲートが本当に閉じることを合成フィクスチャで証明
3. `.agents/skills/cross-review/prompts/adversarial-claude.md`(+ template)—
   これまで欠落していたミラーの補完(check 6 が今後の再発を CI で防ぐ)
4. `skills/loop/prompts/pipeline-outer.md` ×4 — `git diff main...HEAD` の
   ハードコードを撤廃。cycle 1 では upstream 追跡refを使ったが、push 済み
   ブランチで diff が空になる退行を cross-review が検出 →
   `git symbolic-ref refs/remotes/origin/HEAD`(リポジトリのデフォルト
   ブランチ)+ `main` フォールバックへ修正
5. `docs/tech-debt/README.md` — stale パス(active→archive ×4)と関数行番号
   の修正 + 新規エントリ1件(下記 follow-up)
6. 残り: quality-gates のワークフロー注記、ralph-loop.md の legacy 注記、
   repo-map の3ディレクトリ追加、run.go ヘルプの env 言及、
   OrchestratorState コメント、check-sync の休眠 KNOWN_DIFFS 削除、
   codex-setup の「six axes」更新

## 却下した監査指摘(記録)

cross-review SKILL の `--output-format json` は「パイプラインの text と
矛盾」と報告されたが、json は標準フロー(インライン実行側が JSON を
パース)、text は Ralph Loop 内部ディスパッチで、同一 SKILL 内で両経路が
別々に正しく文書化されている — 意図的差異として DISMISSED。

## 既知の follow-up(tech-debt 記録済み・本PRのスコープ外)

同じ `HEAD@{upstream}` ベース検出の弱点が、cross-review を実際にゲートする
`scripts/ralph-pipeline.sh:807`(+ミラー)と cross-review SKILL.md の
記述に残存 — push 済みブランチでは diff が空になり cross-review が
サイレントにスキップされ得る。本プランの Non-goals(挙動変更なし)により
別PRで修正するのが適切。

## エビデンス

- Self-review: docs/reports/self-review-2026-07-12-repo-wide-drift-fixes.md(MERGE ×2 cycles)
- Verify: docs/reports/verify-2026-07-12-repo-wide-drift-fixes.md(PASS ×2)
- Test: docs/reports/test-2026-07-12-repo-wide-drift-fixes.md(cycle 2: 13/13 + フル回帰 0 失敗)
- Cross-review: docs/reports/cross-review-triage-repo-wide-drift-fixes.md(cycle 1: 2件→修正、cycle 2: 指摘なし)
