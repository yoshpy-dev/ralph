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
    --- local
    +++ template (0.6.0)
    @@ 旧 L5–7  →  新 L5–9 @@
      5   5 │  ...
      6     │ -old line
          6 │ +new line A
          7 │ +new line B
      7   8 │  ...
    template hash: sha256:...  local hash: sha256:...
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
- **pack の名前空間化**: 言語パックのファイルはマニフェスト上で `packs/languages/<pack>/<rel>` としてキー付けされる。diff 計算は `splitManifestForBase`（`packs/languages/` 配下を除外）と pack ごとに名前空間を剥いだ `splitManifestForPack` の 2 段階で独立に走る。base スコープ側には pack のキーが渡らないため base の removal sweep が pack ファイルを「削除」と誤判定することはなく、pack スコープ側にはそのパック以外のキーが渡らないため pack 側の add sweep が他 pack のファイルを「新規」と誤判定することもない。pack スコープでも `removed from template` 検出は有効で、真に template から削除された pack ファイルは `packs/languages/<pack>/<rel>` のフルパスで警告に現れる。
- **pack の一時的失敗時のエントリ保持 vs release 削除時の明示的ドロップ**: これらは反対の挙動を持つ 2 経路として区別される。
  - *一時的失敗（preservation）*: pack が `scaffold.AvailablePacks()` には存在するが、埋め込み FS のロードまたは diff 計算が失敗した場合、その pack に対応する旧マニフェストのエントリは新マニフェストへそのままコピーされ、追跡情報は失われない（Warning を stderr に出力）。一時的不具合で既存エントリが消えて再生成扱いになることを防ぐ。
  - *release で削除された pack（explicit drop）*: pack が `scaffold.AvailablePacks()` に存在しない（リリースで削除・改名された）場合、マニフェスト追跡は明示的にドロップされ、`Meta.Packs` からも外れる。ディスク上のファイルはそのまま残され、`Notice: pack "<pack>" no longer exists in templates — manifest tracking dropped (files on disk left untouched)` が stderr に出力される。
  - *pack 列挙自体の失敗（enumeration failure）*: pack 列挙 (`scaffold.AvailablePacks()`) 自体が失敗した場合は、Warning を stderr に出しつつ全 installed pack エントリを preservation 扱いにし、base ファイルの upgrade は継続する。pack メタデータ不具合で base 更新が止まることはない。
- **`ActionRemove` 後のマニフェスト・ドロップ**: `ActionRemove` 後、マニフェストから該当エントリが削除される。「review and delete manually」の通知は 1 回のみで、同一バージョン再実行しても再通知されない。base ファイルの削除にも同じ扱いが適用される。ただし `Managed=false` エントリはこの契約から除外され、テンプレート側から該当ファイルが削除されても `ActionRemove` には昇格せず `ActionSkip` として扱われ、マニフェスト上にそのまま残る（下記 user-owned 収束の契約を参照）。
- **再導入ファイルの安全側判定 (reintroduction safeguard)**: 旧マニフェストに存在せず、かつディスクに同名ファイルが存在する場合、ディスク内容がテンプレートと一致すれば `ActionAdd`、異なれば `ActionConflict` としてユーザに確認を求める。以前のリリースで削除されたファイルをユーザが手元で保持しておき、後のリリースで再導入された際にローカル編集が無言で上書きされるのを防ぐためのガード。
- **ローカル編集検知と `Managed=false` 収束 (local-edit detection & user-owned convergence)**: テンプレートが未変更 (`newHash == manifestHash`) でもディスク内容がマニフェストハッシュと乖離している場合は `ActionConflict` として扱い、`[o]verwrite / [s]kip / [d]iff` を提示する。`[d]iff` 選択時は `internal/upgrade/unified_diff.go` が生成する unified diff（`--- local` / `+++ template (version)`、`-` はローカル行・`+` はテンプレート行）を表示する。各変更行には `<旧行番号> <新行番号> │ <prefix><内容>` 形式の右寄せ行番号ガッター（最小2桁、ファイル長に応じて動的拡張）が付与され、ハンクヘッダは `@@ 旧 L<start>–<end>  →  新 L<start>–<end> @@`（片側が空の場合は `(空)`）で表示される。出力先が TTY かつ `NO_COLOR` 環境変数が未設定（空文字）の場合のみ ANSI カラー（`---` 太字赤 / `+++` 太字緑 / `@@` シアン / `-` 行赤 / `+` 行緑）が付与される。`NO_COLOR=1` を含む `NO_COLOR` の任意の非空値、またはパイプ／ファイルへのリダイレクト時はカラーが抑制される（https://no-color.org 準拠）。`skip` 選択時はマニフェストへ `{Hash: diskHash, Managed: false}` を書き込み、以降そのエントリは `ComputeDiffsWithManifest` で早期 `ActionSkip`（プロンプトも auto-update も抑制）となるため、同一バージョンの再実行で prompt storm が起きない。`overwrite` 選択はローカルをテンプレートへ揃え、マニフェストを `{Hash: newHash, Managed: true}` に戻す。`--force` は `ActionConflict` 経路の上書きに加え、既に `Managed=false` となっているエントリに対してもテンプレート内容をディスクへ書き込み、マニフェストを `{Hash: newHash, Managed: true}` に flip させる（tree 全体を対象とした再管理化エスケープハッチ）。ただしテンプレート側から該当ファイルが削除されている場合は `--force` でも書き戻す元データが無いため `ActionSkip` が維持され、`Managed=false` エントリはディスクとマニフェスト双方で保全される（user-owned 契約がテンプレート変更を跨いで生存）。ディスク読み取り失敗時は警告 + hash サマリにフォールバックし、abort しない。特定パスのみを対象とする `--resync <path>` / `--adopt` は将来スコープ (`docs/tech-debt/README.md` を参照)。

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
