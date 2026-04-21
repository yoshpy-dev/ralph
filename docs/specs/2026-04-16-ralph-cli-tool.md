# ralph CLI ツール化

## 概要

harness-engineering-scaffolding-template を、テンプレートリポジトリから **`ralph` CLI ツール**に変換する。Homebrew tap / GitHub Releases / curl ワンライナーで配布し、`ralph init` でプロジェクトにハーネスエンジニアリングのスキャフォールドを展開、`ralph upgrade` でテンプレート更新を取り込む。パイプラインのコアロジックは Go バイナリに統合し、ユーザーカスタマイズはTOML設定とプロンプトテンプレートで行う。

## 背景と課題

### 現状

- テンプレートリポジトリとして `git clone` / `Use this template` で配布
- パイプラインロジックはシェルスクリプト群（`ralph-orchestrator.sh`, `ralph-pipeline.sh` 等）
- TUI は `cmd/ralph-tui/` として別バイナリ
- テンプレート更新を取り込む手段がない（フォーク後は手動マージ）

### 理想状態

- `brew install yoshpy-dev/tap/ralph` または `curl | sh` でインストール
- `ralph init` でインタラクティブにスキャフォールド展開
- `ralph upgrade` でテンプレート更新をファイル単位で取り込み（コンフリクト解決UI付き）
- パイプラインロジックは Go バイナリ内で完結（外部スクリプト不要）
- ユーザーはTOML設定とプロンプトテンプレートでカスタマイズ

## 要件

### 機能要件

- [ ] **`ralph init`**: インタラクティブセットアップ（プロジェクト名、言語パック選択、Ralph Loop有無、TUI有無）。デフォルトは全てYes。プロジェクト名はカレントディレクトリ名をデフォルト挿入。`.git` が存在しなければ自動で `git init`、存在すればスキップ
- [ ] **`ralph upgrade`**: マニフェストベースのファイル追跡。コンフリクト時はファイルごとに上書き/スキップ/diff表示を選択可能
- [ ] **`ralph run`**: パイプラインコアロジックをGoで実行。`claude -p` を `os/exec` で呼び出し。設定はプロジェクト側のTOMLを参照
- [ ] **`ralph status`**: TUI統合。TTY + バイナリ内蔵で自動起動。`--no-tui` でテーブル出力、`--json` でJSON出力
- [ ] **`ralph retry <slice>`**: 失敗スライスの再実行
- [ ] **`ralph abort [--slice <name>]`**: パイプライン停止
- [ ] **`ralph doctor`**: 環境チェック（Claude Code CLI、hooks整合性、マニフェストバージョン差分、言語パックverify.sh、Go有無）
- [ ] **`ralph pack add <lang>`**: 言語パック追加
- [ ] **`ralph version`**: バージョン表示（semver + commit + build date）
- [ ] **`ralph help`**: ヘルプ表示
- [ ] **テンプレート埋め込み**: `go:embed` でスキャフォールドファイル群をバイナリに格納
- [ ] **マニフェスト管理**: `.ralph/manifest.toml` でファイルパス、テンプレートバージョン、SHA256ハッシュを記録
- [ ] **ファイル所有権**: `templates/` に存在するパスはralph管理。存在しないパスはユーザー所有で一切触らない
- [ ] **TOML設定**: プロジェクト側に `ralph.toml` を配置。パイプライン設定（モデル、イテレーション上限、並列数、タイムアウト等）を記述
- [ ] **プロンプトテンプレート**: `.ralph/prompts/` にユーザーカスタマイズ可能なプロンプトファイルを配置（全文差し替え方式）。存在しない場合はバイナリ内蔵のデフォルトにフォールバック。upgradeでコンフリクトする場合はoverwrite/skip/diffで確認
- [ ] **配布**: Homebrew tap + GitHub Releases (goreleaser) + `curl | sh` ワンライナー
- [ ] **スクリプト不要**: パイプラインコアロジックがGoに移植された後、`templates/base/scripts/` にはユーザー向けスクリプトを残さない（全てGoバイナリで代替）

### 非機能要件

- [ ] シングルバイナリ（外部ランタイム依存なし）
- [ ] クロスプラットフォーム（darwin/amd64, darwin/arm64, linux/amd64, linux/arm64）
- [ ] `ralph init` は10秒以内に完了
- [ ] `ralph upgrade` はファイル数100未満で5秒以内に完了
- [ ] テンプレート更新 = バイナリ更新（`go:embed` のため）

## 受け入れ基準

- [ ] `ralph init` を新規ディレクトリで実行し、`.claude/`, `AGENTS.md`, `CLAUDE.md`, `docs/`, `ralph.toml`, `.ralph/manifest.toml` が生成され、`.git` が自動初期化されること
- [ ] `ralph init` を既存プロジェクト（`.claude/` 存在、`.git` 存在）で実行し、ファイル単位で上書きされること（ユーザー独自ファイルは残ること）。`.git` 初期化はスキップされること
- [ ] `ralph upgrade` でテンプレート更新時、ユーザー未編集ファイルは自動上書き、編集済みファイルはコンフリクト解決UIが表示されること
- [ ] `ralph run --plan <path>` で Go バイナリ内からパイプラインが実行されること
- [ ] `ralph status` でTTY環境下でTUIが起動すること
- [ ] `ralph doctor` で環境問題が検出・報告されること
- [ ] `brew install yoshpy-dev/tap/ralph` でインストールできること
- [ ] `curl -fsSL <url> | sh` でインストールできること
- [ ] プロジェクト側に `.ralph/prompts/self-review.md` が存在する場合はそれが使われ、存在しない場合はバイナリ内蔵デフォルトが使われること

## ユーザーストーリー

1. 開発者として、新規プロジェクトに `ralph init` でハーネスエンジニアリングのベストプラクティスを一発セットアップしたい。なぜなら手動コピーは手間がかかりミスも起きるから。
2. 既存プロジェクトの開発者として、`ralph init` で後付けセットアップし、既存の `.claude/` 設定を壊さずに拡張したい。なぜなら既にカスタマイズ済みの設定を失いたくないから。
3. テンプレートが改善されたとき、`ralph upgrade` で最新のベストプラクティスを取り込みたい。なぜならテンプレートリポジトリの手動マージは面倒で漏れやすいから。
4. パイプラインの動作をプロジェクト固有に調整したい。なぜならプロジェクトごとにモデル、イテレーション数、プロンプトの要件が異なるから。

## 制約条件

### スコープ内

- `ralph` CLI の Go 実装（cobra + huh + bubbletea）
- `go:embed` によるテンプレート埋め込み
- マニフェストベースの init/upgrade
- パイプラインコアロジックの Go 移植
- TOML設定ファイル（`ralph.toml`）
- カスタマイズ可能なプロンプトテンプレート（全文差し替え + フォールバック）
- Homebrew tap + GitHub Releases + curl installer
- goreleaser によるリリース自動化

### スコープ外

- Claude Code 以外のエージェント対応（Codex, Gemini CLI等）
- リモートテンプレートレジストリ（将来のv2で検討）
- Web UI / GUI
- テンプレートのGo template構文による動的レンダリング（静的コピーで十分）
- プラグインシステム

## 影響範囲

| 影響対象 | 影響内容 | 深刻度 |
|---------|---------|--------|
| `cmd/ralph-tui/` | `cmd/ralph/` に統合・リネーム | 高 |
| `scripts/ralph` | Go実装に置換 | 高 |
| `scripts/ralph-*.sh` | コアロジックは Go に移植。設定は `ralph.toml` に | 高 |
| `packs/languages/` | `templates/packs/` に移動 | 中 |
| `.claude/`, `AGENTS.md`, `CLAUDE.md`, `docs/` | `templates/base/` に移動（embed対象） | 中 |
| `go.mod` | 依存追加: cobra, huh, go-toml/v2 | 低 |
| CI/CD | goreleaser + Homebrew tap ワークフロー追加 | 中 |

## 依存関係

| ライブラリ | 用途 |
|-----------|------|
| `spf13/cobra` | CLIフレームワーク |
| `charmbracelet/huh` | インタラクティブプロンプト |
| `charmbracelet/bubbletea/v2` | TUI（既存） |
| `charmbracelet/lipgloss/v2` | TUIスタイル（既存） |
| `pelletier/go-toml/v2` | TOML設定読み書き |
| `goreleaser/goreleaser` | リリース自動化 |

## 調査結果

### コードベース分析

- 現在の Go コードは `cmd/ralph-tui/` + `internal/{action,state,ui,watcher}` の6パッケージ構成
- Go 1.25.0、Bubble Tea v2 を使用中
- TUI は `--orch-dir`, `--worktree-base`, `--plan-dir` フラグで起動
- `scripts/ralph` はシェルスクリプトのディスパッチャで `run/status/retry/abort` を処理
- パイプラインの実体は `ralph-orchestrator.sh` → `ralph-pipeline.sh` → `claude -p` 呼び出し

### ベストプラクティス

- **テンプレート埋め込み**: `go:embed` + `embed.FS` + `fs.WalkDir` がGo CLIツールの標準パターン
- **マニフェスト**: SHA256ハッシュでユーザー編集検出。atlas (HashiCorp) 方式
- **コンフリクト解決**: 未編集→自動上書き、編集済み→選択UI（atlas/CRA方式）
- **CLI+プロンプト**: cobra（サブコマンド）+ charmbracelet/huh（インタラクティブフォーム）が現在のデファクト
- **配布**: goreleaser がクロスコンパイル + チェックサム + Homebrew Formula 自動プッシュを一括処理

### 検討した代替案とトレードオフ

| 選択肢 | メリット | デメリット | 採用 |
|--------|---------|-----------|------|
| go:embed | シングルバイナリ完結 | テンプレート更新=バイナリ更新 | **採用** |
| 外部テンプレートリポ | テンプレート独立更新 | 追加ネットワーク依存、複雑性増 | 不採用（v2検討） |
| パイプライン全Go移植 | 完全統合 | 開発コスト大 | **採用**（ハイブリッド） |
| パイプラインはシェル維持 | カスタマイズ性高い | バージョン不整合リスク | 不採用 |
| YAML設定 | 広く普及 | 複雑な構造に弱い、型安全でない | 不採用 |
| TOML設定 | 明示的な型、セクション構造 | YAMLほど普及していない | **採用** |

## アーキテクチャ概要

### リポジトリ構造（変更後）

```
ralph/
├── cmd/ralph/                  # メインエントリポイント
│   └── main.go
├── internal/
│   ├── cli/                    # cobra サブコマンド群
│   │   ├── root.go
│   │   ├── init.go
│   │   ├── upgrade.go
│   │   ├── run.go
│   │   ├── status.go
│   │   ├── retry.go
│   │   ├── abort.go
│   │   ├── doctor.go
│   │   ├── pack.go
│   │   └── version.go
│   ├── scaffold/               # テンプレート展開・マニフェスト管理
│   │   ├── embed.go            # go:embed templates/*
│   │   ├── render.go           # ファイル展開ロジック
│   │   └── manifest.go         # .ralph/manifest.toml 読み書き
│   ├── upgrade/                # diff・コンフリクト解決
│   │   ├── diff.go
│   │   └── conflict.go
│   ├── pipeline/               # パイプラインコアロジック（Go移植）
│   │   ├── orchestrator.go
│   │   ├── runner.go
│   │   └── claude.go           # claude -p 呼び出し
│   ├── config/                 # ralph.toml パーサー
│   │   └── config.go
│   ├── prompt/                 # プロンプトテンプレート解決
│   │   └── resolver.go         # プロジェクト側 → 内蔵フォールバック
│   ├── state/                  # 既存
│   ├── watcher/                # 既存
│   ├── ui/                     # 既存（TUI）
│   └── action/                 # 既存
├── templates/                  # go:embed 対象
│   ├── base/
│   │   ├── AGENTS.md
│   │   ├── CLAUDE.md
│   │   ├── .claude/
│   │   ├── docs/
│   │   └── ralph.toml          # デフォルト設定
│   ├── packs/
│   │   ├── golang/
│   │   ├── typescript/
│   │   ├── python/
│   │   ├── rust/
│   │   └── dart/
│   └── prompts/                # デフォルトプロンプトテンプレート（フォールバック用）
│       ├── self-review.md
│       ├── verify.md
│       ├── test.md
│       └── sync-docs.md
├── scripts/                    # ralph自体のビルド・CI用
├── docs/                       # ralph自体のドキュメント
├── .goreleaser.yml
└── go.mod
```

### マニフェスト形式

```toml
# .ralph/manifest.toml（ralph initが生成、gitにコミット）
[meta]
version = "0.5.0"
created = "2026-04-16T10:00:00Z"
updated = "2026-04-16T10:00:00Z"

[files]

[files.".claude/skills/plan/SKILL.md"]
hash = "sha256:abc123..."
managed = true

[files."AGENTS.md"]
hash = "sha256:def456..."
managed = true

[files."ralph.toml"]
hash = "sha256:ghi789..."
managed = true
```

### TOML設定

```toml
# ralph.toml（プロジェクトルート）
[pipeline]
model = "claude-opus-4-7"
effort = "xhigh"
max_iterations = 20
max_parallel = 4
slice_timeout = "30m"
permission_mode = "auto"

[pipeline.prompts]
dir = ".ralph/prompts"   # カスタムプロンプトのパス

[doctor]
require_claude_cli = true
require_go = false        # TUI不要なら false
```

### プロンプトテンプレート解決順序

```
1. .ralph/prompts/self-review.md   <- プロジェクト側（ユーザーカスタマイズ、全文差し替え）
2. templates/prompts/self-review.md <- バイナリ内蔵（フォールバック）
```

### upgrade フロー

```
ralph upgrade

Checking for updates...
  Current: 0.5.0 -> Available: 0.6.0

  Files to update: 15
  ✓ .claude/hooks/pre_bash_guard.sh (unchanged, auto-update)
  ✓ .claude/skills/verify/SKILL.md (unchanged, auto-update)
  ⚠ .claude/rules/testing.md (modified locally)
    [o]verwrite / [s]kip / [d]iff ? d
    --- ralph template (0.6.0)
    +++ local
    @@ -5,3 +5,5 @@
     ...
    [o]verwrite / [s]kip ? s
  ✓ AGENTS.md (unchanged, auto-update)
  ...

  Updated: 13 files
  Skipped: 2 files (user-modified)
  Manifest updated: .ralph/manifest.toml
```

#### 冪等性と自動修復 (idempotency & heal)

- **同一バージョン冪等性**: 同じバージョン間 (`X.Y.Z → X.Y.Z`) で `ralph upgrade` を複数回実行しても、未編集ファイルは `modified locally` / `removed from template` / `new file` のいずれにも表示されず、内部的に `ActionSkip` として処理される。`ActionSkip` は新しい sha256 ハッシュ (`NewHash`) を保持するため、マニフェストの `hash` フィールドは常に最新テンプレートのハッシュに揃う。
- **空ハッシュ自動修復 (heal)**: マニフェストエントリが `hash = ''` で壊れている場合でも、ディスク上の内容がテンプレートと一致するなら `ActionSkip` として扱い、対話プロンプトなしでハッシュを復旧する。ディスクが異なる場合のみ従来どおり `ActionConflict` となり、ユーザー確認を求める。`--force` や `--repair` フラグは不要で、通常の `ralph upgrade` 1 回で回復する。
- **pack の名前空間化**: 言語パックのファイルはマニフェスト上で `packs/languages/<pack>/<rel>` としてキー付けされる。diff 計算は base（`packs/languages/` 外）と pack ごとの 2 段階で独立に走り、base の removal sweep が pack ファイルを「削除」と誤判定することはない。pack 側は `checkRemovals=false` で計算し、同一ファイルが `removed from template` と `new file` の両方に現れることはない。
- **pack diff 失敗時のエントリ保持**: pack の埋め込み FS ロードや diff 計算が失敗した場合、その pack に対応する旧マニフェストのエントリは新マニフェストへそのままコピーされ、追跡情報は失われない（警告は stderr に出力）。これにより pack が一時的に利用不可能でも、既存エントリが消えて再生成扱いになることを防ぐ。

## セキュリティ考慮事項

- `curl | sh` インストーラーは HTTPS + チェックサム検証を含むこと
- goreleaser のチェックサムファイル (`checksums.txt`) を Releases に添付
- `ralph run` が `claude -p` を `os/exec` で呼ぶ際、ユーザー入力をシェル経由で渡さない（引数リスト形式）
- テンプレートにシークレットを含めない（`.env` 等はテンプレート対象外）

## 参考資料

- goreleaser ドキュメント
- charmbracelet/huh（Go インタラクティブフォーム）
- pelletier/go-toml/v2
- atlas (HashiCorp) のスキャフォールディングパターン
