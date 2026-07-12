# Walkthrough: unify-permission-mode

- Date: 2026-07-12
- Plan: docs/plans/archive/2026-07-12-unify-permission-mode.md
- Branch: fix/unify-permission-mode-default (base: main @ a6ac5d6)
- Diff size: 17 files, +768 / -29(うち約6割はプラン/レポート成果物)

## 読む順序(推奨)

1. `internal/cli/run.go` — 本PRの核。(a) `RALPH_MODEL` / `RALPH_EFFORT` /
   `RALPH_PERMISSION_MODE` の無条件 append → `appendEnvIfMissing`(env >
   toml)、(b) `RALPH_MAX_ITERATIONS` / `RALPH_MAX_PARALLEL` は Cobra
   `Flags().Changed()` の presence bool を `runPipeline` に渡し
   CLI > env > toml を実現
2. `internal/config/config.go` — `Default()` の permission_mode を
   `bypassPermissions` に(shell 側 `ralph-config.sh:20` は元々この値)
3. `templates/base/ralph.toml` — 値の統一 + ガードレールの正確な説明
   コメント(対話フックは `claude -p` では発火しない — 実際の防御は
   worktree 分離 / 事後チェック / verify・test ゲート / PR レビュー)
4. テスト — `internal/cli/run_env_test.go`(env優先 ×3変数、フラグ優先、
   empty-env 契約)、`internal/config/config_test.go`(デフォルト/
   バックフィル/テンプレート toml アサート)、
   `tests/test-ralph-config.sh`(シェル層の empty-env 2ケース)
5. ドキュメント — recipe ×2 / README / `.codex/README.md` ×2:
   統一デフォルトと「env は全入口、toml は Go `ralph run` のみ」の
   オーバーライド規則。`docs/tech-debt/README.md` の乖離行を RESOLVED 化

## 設計判断の要点

- **非空 env が勝つ契約**: `appendEnvIfMissing` は `KEY=`(空)も「存在」
  と見なすが、下流シェルの `${VAR:-default}` が空をシェルデフォルトに
  解決する。テストは Go 層とシェル層の両方でこのエンドツーエンド契約を
  固定している
- **`!= 0` ではなく `Flags().Changed()`**: `--max-iterations 0` を「未指定」
  と誤解しない(Codex plan advisory finding 4)
- Codex plan advisory 7件の採否は plan の advisory セクションと verify
  レポートに全件記録

## エビデンス

- Self-review: docs/reports/self-review-2026-07-12-unify-permission-mode.md(MERGE、LOW 2)
- Verify: docs/reports/verify-2026-07-12-unify-permission-mode.md(PASS、advisory 7件採用確認)
- Test: docs/reports/test-2026-07-12-unify-permission-mode.md(ターゲット 6/6、43/43、フル回帰 exit 0)
- Cross-review: docs/reports/cross-review-triage-unify-permission-mode.md(P3 1件 → 修正済み)
