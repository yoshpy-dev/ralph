# Codex CLI 標準フローパリティ

## 概要

ralph を Claude Code 専用ハーネスから **Claude Code / Codex 両対応の単一ハーネス** へ拡張する。`ralph init` で `.claude/`、`.codex/`、`.agents/skills/` を always-on で配置し、AGENTS.md を主導指示として両 CLI が同じプラン・スキル・ドキュメントを共有して同等の標準フロー (spec→plan→work→post-impl pipeline→pr) を完走できるようにする。

## 背景と課題

### 現状

- ralph は Claude Code 専用ハーネスとして設計されている。
  - `templates/base/` に `.claude/skills/`, `.claude/agents/`, `.claude/hooks/`, `CLAUDE.md`, `AGENTS.md` を `go:embed` で配布
  - `ralph.toml` 既定値: `model = "claude-opus-4-7"`、`[doctor] require_claude_cli = true`
  - `ralph-pipeline.sh` / `ralph-orchestrator.sh` は `claude -p --model … --effort … --permission-mode … --output-format json` をハードコード
- Codex は **advisory レイヤー** として一部組み込み済み:
  - `scripts/codex-check.sh`、`/codex-review` skill (`codex exec review --base ...`)、`codex:rescue` プラグイン
  - 立ち位置は「Claude Code が主、Codex は cross-model 第二意見」
- チームメンバーごとに好む CLI が異なるため、Claude Code を起動できる前提でしかフローが回らない現状はチーム内差異吸収を阻害している。
- Codex CLI は `AGENTS.md` を公式の主導指示として読み、`.agents/skills/`、`.codex/config.toml`、hooks (PreToolUse 他)、MCP サーバを Claude Code とほぼ並行する形でサポートしている (2026-05 時点)。

### 理想状態

- `ralph init` が一回走れば、その後 Claude Code でも Codex でも同じプロジェクトに入って同じ skill を起動でき、同じ plan / report / PR が生成される。
- AGENTS.md が両 CLI の主導指示となり、CLAUDE.md / Codex 固有ドキュメントは薄く保たれる。
- post-implementation pipeline (self-review → verify → test → sync-docs → cross-review → pr) が両 CLI で完走し、`docs/reports/` の成果物が CLI 非依存の同一フォーマットで残る。
- チームメンバーは個人の好みで CLI を選択でき、PR レビューや CI からは差異が見えない。
- 将来 Codex を主軸にしたくなったときも、追加の大改修なしに移行できる柔軟性が確保される。

## 要件

### 機能要件

- [ ] **F-1**: `templates/base/` に `.codex/` (`config.toml` + `hooks/`) と `.agents/skills/` を追加し、`ralph init`/`ralph upgrade` が `.claude/`、`.codex/`、`.agents/skills/` の三系統を always-on で配置する。
- [ ] **F-2**: 既存全 skill (`spec`, `plan`, `work`, `self-review`, `verify`, `test`, `sync-docs`, `cross-review` (旧 `codex-review`), `pr`, `anti-bottleneck`, `audit-harness`, `release`, `loop` ※loop は本スコープ対象外) の Codex 並走版を `.agents/skills/<name>/SKILL.md` として作成する。frontmatter は Codex 仕様 (`name`, `description`)、必要に応じ `agents/openai.yaml` (`policy.allow_implicit_invocation`, dependencies) を併設する。
- [ ] **F-3**: 各 skill 本文に CLI 別の実行ガイダンスを明記する。
  - サブエージェント: Claude=`Task(subagent_type=...)` / Codex=順次 inline 実行
  - 対話: Claude=`AskUserQuestion` / Codex=stdin 番号付き選択肢プロンプト
  - 駆動コマンド: Claude=`claude -p ...` / Codex=`codex exec ...`
- [ ] **F-4**: `/codex-review` を **`/cross-review`** にリネームし、双方向 cross-model レビューに改修する。
  - Claude 主フロー → `codex exec review --base "$BASE"` を呼ぶ
  - Codex 主フロー → `claude -p` で reviewer 役を呼ぶ
  - 参照箇所も全てリネーム (互換 alias は作成しない): `.claude/skills/cross-review/`, `.agents/skills/cross-review/`, `.claude/skills/work/SKILL.md`, `.claude/skills/loop/SKILL.md`, `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`, `CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/quality/definition-of-done.md` ほか `codex-review` を含むファイル全て。
- [ ] **F-5**: AGENTS.md を両 CLI 共通の source of truth として再構成する (32 KiB cap 内)。CLAUDE.md は Claude Code 固有事項のみに絞る。Codex 固有事項は `.codex/AGENTS.override.md` または `.codex/README.md` に切り出す。
- [ ] **F-6**: `.codex/config.toml` テンプレートを定義する。
  - `model = "gpt-5.5"` を既定値とする
  - `sandbox_mode`、`approval_policy`、`mcp_servers.<id>`、`[hooks]` (PreToolUse / PostToolUse / SessionStart / PermissionRequest)、`[features]` (必要に応じて)、`[tui.notifications]`
  - profiles 定義例: `[profiles.work]`, `[profiles.review]`
- [ ] **F-7**: `scripts/check-skill-sync.sh` を新設する。`.claude/skills/<name>/SKILL.md` と `.agents/skills/<name>/SKILL.md` の本文 (frontmatter 除外) を比較し、drift があれば exit 1。`run-verify.sh` から呼び出し、CI でゲート化する。
- [ ] **F-8**: `ralph doctor` に Codex CLI 検出と version 確認を追加する。`ralph.toml` の `[doctor]` セクションに `require_codex_cli = true` を追加し、未インストール時に warning。`require_claude_cli` と並列で扱う。
- [ ] **F-9**: README / AGENTS.md / `docs/recipes/` に Codex 起動手順と既知差異を明記する。
  - 起動: `codex` 起動 → `/skills` メニューまたは `$spec` mention、暗黙起動の例
  - 既知差異表: skill frontmatter、subagent、対話、permission、cross-review

### 非機能要件

- [ ] AGENTS.md は 32 KiB (Codex の `project_doc_max_bytes` 既定値) を超えないこと。`audit-harness` skill に AGENTS.md サイズ警告を追加する。
- [ ] skill drift check は CI で 30 秒以内に完走すること。
- [ ] テンプレート展開後の `.codex/config.toml` は Codex CLI で構文エラーなくロードできること (`codex config validate` 相当の確認)。
- [ ] `ralph init` の所要時間は現状から +1 秒以内に抑えること (`.codex/`, `.agents/skills/` の追加配置のみ)。

## 受け入れ基準

- [ ] **AC-1**: `ralph init <dir>` 実行後、`<dir>/.claude/`, `<dir>/.codex/`, `<dir>/.agents/skills/` の三系統と `<dir>/AGENTS.md`, `<dir>/CLAUDE.md`, `<dir>/ralph.toml` が配置されている。
- [ ] **AC-2**: Codex CLI で `/spec` → `/plan` → `/work` → post-implementation pipeline → `/pr` を実行した場合、`docs/specs/<file>.md`, `docs/plans/active/<file>.md`, `docs/reports/{self-review,verify,test,sync-docs,cross-review-triage}-*.md` が生成され、PR が作成される。生成物は Claude Code 実行時と同等品質 (構造、必須セクション、findings 記法) であること。
- [ ] **AC-3**: `./scripts/check-skill-sync.sh` が `.claude/skills/` と `.agents/skills/` の本文 drift を検出する。差分なしで exit 0、差分ありで exit 1 + 該当ファイル名出力。CI (GitHub Actions) で実行される。
- [ ] **AC-4**: post-implementation pipeline が Codex 側で順次 inline 実行され、各ステップが `docs/reports/*.md` を生成する。`reviewer` / `verifier` / `tester` / `doc-maintainer` 相当の責務が単一 agent 内で連続実行される。
- [ ] **AC-5**: `/cross-review` が双方向で動作する。Codex 主フローでは `claude -p` を、Claude 主フローでは `codex exec review` を呼び、いずれの方向でも `docs/reports/cross-review-triage-<slug>.md` が生成される。
- [ ] **AC-6**: `ralph doctor` が `claude` / `codex` の両 CLI 検出 + version 表示 + verify ステータスを表示する。片方のみインストールされていてもエラーにならず warning に留める。
- [ ] **AC-7**: `./scripts/run-verify.sh` が green。AGENTS.md がプロジェクト直下で 32 KiB 以内。`audit-harness` がサイズ警告を出すこと。
- [ ] **AC-8**: `tests/` に `.codex/`, `.agents/skills/` の go:embed 検証 (`internal/scaffold/embed_test.go` 拡張) と `ralph init` の展開検証が追加され、green。

## ユーザーストーリー

1. **Codex 派の開発者として**、ralph プロジェクトに入ったら `codex` を起動して `/spec` `/plan` `/work` を回したい。なぜなら Claude Code 派と同じプラン・同じ PR フォーマットでチームに貢献したいから。
2. **Claude Code 派の開発者として**、Codex 派の同僚が作った `docs/plans/active/<plan>.md` をそのまま `/work` で消費したい。なぜならプランは CLI 非依存で書かれているから。
3. **チームリードとして**、CI で skill drift check が走ることで、`.claude/skills/` だけ更新して `.agents/skills/` を放置するレビューをブロックしたい。なぜなら片方のメンバー体験が劣化するから。
4. **将来 Codex を主軸化する保守者として**、`.codex/config.toml` の profiles を増やせば現行 ralph フローのまま Codex 中心に切り替えられる柔軟性が欲しい。

## 制約条件

### スコープ内

- 標準フロー (spec → plan → work → self-review → verify → test → sync-docs → cross-review → pr) の Codex 並走対応。
- `templates/base/` の `.codex/`, `.agents/skills/` 追加と `go:embed`/init/upgrade 拡張。
- 全 skill の Codex 仕様併設、CLI 別ガイダンスの本文追記。
- `/codex-review` → `/cross-review` リネーム + 双方向化、参照箇所の全更新。
- `ralph doctor` の Codex 検出。
- skill drift check スクリプトと CI 連携。
- README / AGENTS.md / docs の更新。

### スコープ外 (別 issue 起票)

- **別 issue (Loop 対応)**: `ralph-orchestrator.sh` / `ralph-pipeline.sh` の codex driver 化。並列 worktree の Codex 駆動。`/loop` skill の双 CLI 化。
- ralph CLI に `ralph spec`/`ralph plan`/`ralph work` ラッパー追加 (両 CLI 自動振り分け) — Phase 2 で評価。
- Codex `[features] multi_agent = true` を使った真の並列 subagent 化 — Phase 2 で評価。
- AskUserQuestion 相当を提供する MCP server の自前実装 — 評価のみ、実装は別 issue。
- `codex:rescue` プラグインの ralph 同梱 — 既存維持。

## 影響範囲

| 影響対象 | 影響内容 | 深刻度 |
|---------|---------|--------|
| `templates/base/` | `.codex/`, `.agents/skills/` 追加 | 中 |
| `templates/base/CLAUDE.md` | 薄型化、Codex 関連を AGENTS.md に移譲 | 中 |
| `templates/base/AGENTS.md` | 主導指示として再構成、両 CLI 起動手順追記 | 中 |
| `templates/base/ralph.toml` | `[doctor] require_codex_cli` 追加、Codex 用 model 既定値 | 小 |
| `internal/scaffold/` (go:embed) | 新ディレクトリの埋め込みパス追加 | 中 |
| `internal/cli/init.go` / `upgrade.go` | 新配置の展開ロジック | 中 |
| `internal/cli/doctor.go` | Codex CLI 検出ロジック | 小 |
| `.claude/skills/codex-review/` | `.claude/skills/cross-review/` にリネーム + 双方向化 | 大 |
| `.claude/skills/{spec,plan,work,self-review,verify,test,sync-docs,pr,anti-bottleneck,audit-harness,release}/SKILL.md` | CLI 別ガイダンス追記 | 中 |
| `.claude/rules/post-implementation-pipeline.md` | `/cross-review` リネーム反映、CLI 別実行モード追記 | 中 |
| `.claude/rules/subagent-policy.md` | Codex=順次 inline の節追加、リネーム反映 | 中 |
| `CLAUDE.md` / `AGENTS.md` / `README.md` | リネーム反映、Codex 起動手順追加 | 中 |
| `docs/quality/definition-of-done.md` | リネーム反映、Codex 完走条件追加 | 小 |
| `scripts/check-skill-sync.sh` (新設) / `scripts/run-verify.sh` | drift check 統合 | 小 |
| `tests/` (`internal/scaffold/embed_test.go` ほか) | 新配置の検証ケース | 中 |
| 既存の Loop 系 (`/loop`, `ralph-pipeline.sh`, `ralph-orchestrator.sh`) | **本スコープでは触らない** が、`/cross-review` リネームの参照修正は実施 | 小 |

## 依存関係

- **Codex CLI** バージョン: skills 機構と AGENTS.md 階層マージをサポートするバージョン (2026 年初頭以降の rust 系列、概ね `rust-v0.75.0` 以降)。`ralph doctor` で min version を要求。
- **Codex skills 仕様**: [Agent Skills – Codex](https://developers.openai.com/codex/skills) (2026-05 時点)。
- **AGENTS.md 仕様**: [Custom instructions with AGENTS.md – Codex](https://developers.openai.com/codex/guides/agents-md)。
- **Codex configuration**: [Configuration Reference – Codex](https://developers.openai.com/codex/config-reference)。
- **Claude Code skills / settings**: 既存の `.claude/skills/`, `.claude/settings.json`, `.claude/hooks/`。

## 調査結果

### コードベース分析

- 既存の Codex 連携面:
  - `scripts/codex-check.sh`: `codex` CLI の存在 + functional check。そのまま再利用可能。
  - `.claude/skills/codex-review/SKILL.md`: 既に `codex exec review --base ...` を呼んでおり、cross-model レビューの実装パターンが確立。`/cross-review` 化の土台となる。
  - `scripts/ralph-pipeline.sh` の Probe 7 で `codex_cli` available チェックを実施済 (line 357-365)。
- skill 構造の互換性:
  - Claude Code 規約: `.claude/skills/<name>/SKILL.md` (frontmatter: `name`, `description`, `disable-model-invocation`, `allowed-tools`)
  - Codex 規約: `.agents/skills/<name>/SKILL.md` (frontmatter: `name`, `description` のみ必須) + 任意 `agents/openai.yaml` (`policy.allow_implicit_invocation`, dependencies)
  - 本文 (Steps / 手順) はほぼそのまま流用可能。CLI 別の差は本文末尾に節を追加する形で吸収可能。
- AGENTS.md 現状: 5.2k (32 KiB cap に対して 16% 使用)。Codex 拡張後も余裕あり。
- `ralph.toml` パーサ (`internal/config/`): 新フィールド追加は設計通り柔軟に拡張可能。

### ベストプラクティス

- **`/openai/codex` (Context7)**: 公式 SKILL.md とディレクトリ構造のリファレンス。
- **`/shanraisshan/codex-cli-best-practice`**: agents、skills、orchestration、project 設定パターンのリファレンス実装。
- **`/luohaothu/everything-codex`**: 62 個の production-ready skill、AGENTS.md テンプレート集。
- **`/yeachan-heo/oh-my-codex`**: 多 agent orchestration、worktree、状態管理 — Phase 2 (Loop 対応) で参考にする。
- AGENTS.md のベストプラクティス: 「durable guidance」「stable workflows をskillsに昇格」「MCPで外部接続」のレイヤ分けを保つ。

### 検討した代替案とトレードオフ

| 選択肢 | メリット | デメリット | 採用 |
|--------|---------|-----------|------|
| `ralph init` で CLI 選択 | テンプレートサイズ最小化 | チーム内差異吸収という主動機と矛盾 | ❌ |
| **常に両方配置** | チーム内差異吸収を満たす、メンバーごとの好み吸収 | テンプレートサイズ +30%、drift リスク | ✅ |
| AGENTS.md 主導 | 両 CLI が同じ source of truth を読む、保守性高い | 32 KiB cap | ✅ |
| `.claude/` と `.codex/` 詳細並走 | CLI 固有を明示しやすい | 重複多く保守コスト高 | ❌ |
| Skill 単一 source + 自動 sync 生成 | drift ゼロ | Codex 固有記述の付与が困難、生成タイミングの罠 | ❌ |
| **Skill 完全並走 + drift check** | Codex 固有を自由に書ける、CI で drift 検出 | 二重保守 | ✅ |
| 共有 body + include wrapper | 重複ゼロ | 両 CLI の include サポートが不明、検証コスト | ❌ |
| Codex `[features] multi_agent` 利用 | Claude と同等の subagent UX | preview 機能、仕様変動リスク | ❌ |
| **Codex で順次 inline 実行** | 安定、シンプル、レポートはファイル経由で同等 | 単一コンテキスト圧迫の懸念 | ✅ |
| AskUserQuestion を MCP server で実装 | UX 差を最小化 | 実装コスト高、メンテ負担 | ❌ |
| **stdin 番号付き選択肢に fallback** | 実装容易、token cost 低 | 構造化 UI なし | ✅ |
| Codex 主フローで cross-review スキップ | スコープ最小 | AC-5 を満たさない | ❌ |
| **`/cross-review` で双方向化** | 主使用 CLI に依存しない品質ゲート | 双方向の動作確認が必要 | ✅ |
| `/codex-review` 名のまま使い回し | 既存資産の互換 | 名称が双方向の意味を表さない | ❌ |
| **`/cross-review` にリネーム + 全参照置換** | 名称が意味を正しく反映 | リネーム作業範囲が広い | ✅ |

## セキュリティ考慮事項

- **設定ファイルの権限境界**: Codex の `sandbox_mode` / `approval_policy` と Claude の `permission-mode` は意味が異なる。テンプレート既定値は両者とも保守的 (Codex: `workspace-write`、Claude: `auto`) を採用し、ドキュメントで違いを明記する。
- **MCP サーバ設定**: `.codex/config.toml` の `mcp_servers.<id>` で URL / bearer token を扱う場合は環境変数経由とし、テンプレートには placeholder のみを置く。
- **hooks の実行権限**: 既存 `commit-msg-guard.sh`, `mojibake_check.sh` 等は exec 権限を保ったまま `.codex/hooks/` 側からも参照する。実体は `scripts/` に集約し、`.codex/hooks/` は config.toml の `[hooks]` で参照する形に統一する。
- **シークレット注入リスク**: `git-commit-strategy.md` の安全クォート規則は Codex 経由でも適用されること (commit-msg-guard.sh が git hook 経由で実行されるため CLI 非依存)。

## 未解決の課題

- なし (主要 OQ はすべて確定済み: model = `gpt-5.5`、`/codex-review` → `/cross-review` リネーム、互換 alias なし)。

## 参考資料

- [Agent Skills – Codex | OpenAI Developers](https://developers.openai.com/codex/skills)
- [Custom instructions with AGENTS.md – Codex | OpenAI Developers](https://developers.openai.com/codex/guides/agents-md)
- [Configuration Reference – Codex | OpenAI Developers](https://developers.openai.com/codex/config-reference)
- [Best practices – Codex | OpenAI Developers](https://developers.openai.com/codex/learn/best-practices)
- [Command line options – Codex CLI](https://developers.openai.com/codex/cli/reference)
- Context7: `/openai/codex`, `/shanraisshan/codex-cli-best-practice`, `/luohaothu/everything-codex`, `/yeachan-heo/oh-my-codex`
- 既存スペック: `docs/specs/2026-04-16-ralph-cli-tool.md`
- 既存ルール: `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`
