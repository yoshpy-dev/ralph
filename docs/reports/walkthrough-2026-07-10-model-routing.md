# Walkthrough: model-routing

対象ブランチ: `chore/model-routing`（base: main）
差分: 29 files / +481 -47（うち約270行はプラン・レポート等のドキュメント）

## 読む順番

1. `.claude/rules/model-routing.md`（新規・本PRの中心）
   モデル階層表（判断席=opus / 手続き席=sonnet / バルク=haiku）とルーティングの
   落とし穴を定義。ルート側コピーのみ「シェル・toml・Go の3箇所 lock-step 更新」の
   bullet を追加で持つ（Go 層は scaffold されないため。check-sync.sh の
   KNOWN_DIFFS に登録済み）。`templates/base/.claude/rules/model-routing.md` は
   汎用版で、下流プロジェクトに配布される。
2. `.claude/agents/` と `templates/base/.claude/agents/`（各3ファイル）
   `verifier` / `tester` / `doc-maintainer` の frontmatter を `model: sonnet` に。
   `reviewer` は判断席として `model: opus` 維持（変更なし）。
3. モデルデフォルトの3層同期（本PRの正しさの要）:
   - `scripts/ralph-config.sh` + `templates/base/scripts/ralph-config.sh`
     （シェルフォールバック、md5 一致を維持）
   - `templates/base/ralph.toml`（scaffold 配布値）
   - `internal/config/config.go` `Default()`（Go 層。`ralph run` が
     `RALPH_*` env として export するため、ここを残すとシェル側の変更が
     実行時にマスクされる — self-review で検出し 1e5f3d0 で修正）
   いずれも `claude-opus-4-7`/`xhigh` → `opus`/`high`（安定エイリアス化）。
4. テスト期待値の同期:
   `tests/test-ralph-config.sh`、`internal/config/config_test.go`、
   `internal/cli/doctor_loop_test.go`、`internal/scaffold/embed_test.go`
5. `scripts/check-sync.sh`
   KNOWN_DIFFS に `.claude/rules/model-routing.md` を追加
   （CLAUDE.md / AGENTS.md と同じ「ルート固有拡張」の前例に倣う）。
6. cross-review SKILL.md 4ミラーと `docs/recipes/ralph-loop.md` ×2 の表記更新。

## 検証

- `./scripts/run-test.sh` exit 0（shell 27/27 + go test 9 パッケージ ok）
- `./scripts/run-verify.sh` exit 0（full scope）
- `./scripts/check-skill-sync.sh` 13 skills lock-step
- `./scripts/check-sync.sh` DRIFTED 0 / KNOWN_DIFF 3

## 意図的に触っていないもの

- `docs/specs/` 内の旧モデルID（歴史的記録）
- `.codex/agents/`（GPT 系で動作、対象外）
- detect-changed-languages.sh の harness 分類（koalive 固有の拡張で、
  本リポジトリの detector 構造では不要）
