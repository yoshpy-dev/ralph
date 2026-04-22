# Walkthrough: rename-to-ralph-cli

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-rename-to-ralph-cli.md` (本 PR で archive 予定)
- Author: Claude Code (refactor/rename-to-ralph-cli)

## What changed

GitHub リポジトリを `yoshpy-dev/harness-engineering-scaffolding-template` から `yoshpy-dev/ralph` へ改名し、リポジトリ内の自己参照（Go モジュールパス、配布スクリプト URL、README / AGENTS.md / CLAUDE.md / 一部 docs のブランディング）を一貫して「`ralph` CLI」基準へ刷新。

- 41 files changed / +859 / −294
- 10 commits on `refactor/rename-to-ralph-cli`
- GitHub side: repo renamed (旧 URL は 301 redirect)、description / topics 更新済み

## Key files to read first

レビュアーは以下の順で読むと全体像が把握しやすい:

1. `docs/plans/active/2026-04-22-rename-to-ralph-cli.md` — 本 refactor のプラン（Non-goals で "履歴は書き換えない" を明記、rename-first rollout）
2. `docs/reports/codex-triage-2026-04-22-rename-to-ralph-cli.md` — Codex 指摘なし (Case C) の根拠
3. `README.md` — ブランディング刷新の主要成果物。全面リライト (298 行 → 139 行)
4. `AGENTS.md` (line 3) と `CLAUDE.md` (line 9) — 1 行ずつの reframing
5. `go.mod` — module path 変更 (`github.com/yoshpy-dev/harness-engineering-scaffolding-template` → `github.com/yoshpy-dev/ralph`)
6. `scripts/install.sh` (lines 7, 10) と `.goreleaser.yml` (lines 46-47) — 配布メタデータ更新

## Main control flow

変更の論理的順序:

1. **GitHub repo rename（不可逆、事前実行）**: `gh repo rename ralph` を実行。成功を確認してからコード変更に着手（rename-first gate）。
2. **Go モジュールパス置換**: `go mod edit -module` + `*.go` 25 ファイル一括 sed + testdata 2 件の JSON 修正。検証は `go build ./... && go test ./...`。
3. **配布スクリプト**: `install.sh` の `REPO` 定数と curl 例 URL、`.goreleaser.yml` の `homepage` を更新。Homebrew formula 名は既に `ralph` だったため tap 側変更なし。
4. **ドキュメント刷新**: README 全面リライト、AGENTS.md/CLAUDE.md 軽微調整、`docs/architecture/design-principles.md` と `docs/research/approach-comparison.md` の "this scaffold" 自己参照を曖昧性除去のため修正（sync-docs）。
5. **GitHub メタ情報**: description = `ralph — a CLI for harness engineering with Claude Code`、topics: `cli`, `claude-code`, `go`, `agent-harness`, `harness-engineering`。古い `template` / `harnessengineering` topic は削除。

## Risky code paths

機械的 rename 中心のため実装ロジックへの影響は最小限。それでも目を通すべき領域:

- **`internal/state/testdata/orchestrator-complete.json` と `checkpoint-complete.json`**: PR URL リテラルがテスト期待値と連動。lockstep で更新済みだが、今後 URL スキーマが再度変わる場合の drift source になりうる（self-review の LOW 指摘として記録）。
- **`scripts/install.sh`**: `REPO` 定数変更のみ。テストは `bash -x install.sh --version 0.0.0` による URL assembly 検査で、ダウンロード完遂は意図的に求めていない（sentinel バージョンで 404 になる設計）。
- **`.goreleaser.yml` の `homepage`**: 既存 Homebrew tap (`yoshpy-dev/homebrew-tap`) の `ralph.rb` は次回 release tag push 時に goreleaser が `url` / `homepage` を新 repo へ自動更新する。それまでは GitHub 側 301 redirect が tarball 取得をカバー（Risk R3）。

## What a human reviewer should pay special attention to

1. **Non-goals 領域の残存参照**: `docs/plans/archive/`, `docs/reports/`, `docs/specs/`, 本 active plan の 4 箇所のみ旧名を保持。プラン §Non-goals と §Acceptance criteria (scoped grep) で明示的に許容している。歴史的文脈保全が目的。
2. **Rename-first rollout**: repo rename が先、code merge が後の順序。`main` が存在しない repo URL を参照する窓口を作らない（Codex HIGH#2 への応答）。
3. **External importer evidence**: `gh search code` 0 件、`pkg.go.dev` HTTP 404 を planning フェーズで取得済み（verify report に添付）。モジュールパス変更の後方互換シムは不要と判定。
4. **AGENTS.md line 1 の解釈**: 厳密な line 1 は見出し `# AGENTS.md`。CLI 説明は line 3（最初のプローズ行）。verify report で "wording deviation" として注記されているが、acceptance criterion の意図は満たしている。

## Known limitations

- **次回 release 推奨**: Homebrew formula の `url` / `homepage` を新 repo に追従させるために、本 PR merge 後 `/release` (patch bump) を推奨。直ちに brew 導線が壊れるわけではない（GitHub 301 redirect がカバー）が、`ralph.rb` を正準状態に戻すため。
- **メモリディレクトリのパス**: `~/.claude/projects/-Users-...-harness-engineering-scaffolding-template/memory/` は未移行。ユーザ作業として別途 rename 想定（Risk R5）。
- **EMU 制約下の gh 操作**: `gh repo rename` / `gh repo edit` は active account (`hiroki-yoshioka_dena`, EMU) では admin 権限なし、owner account (`yoshpy-dev`) に切替えて実行した。`gh pr create` は EMU account から実行（履歴で許容実績あり）。
