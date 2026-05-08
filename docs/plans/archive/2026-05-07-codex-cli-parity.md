# Codex CLI 標準フローパリティ

- Status: Draft
- Owner: Claude Code
- Date: 2026-05-07
- Related request: docs/specs/2026-05-07-codex-cli-parity.md
- Related issue: N/A (follow-up Loop: yoshpy-dev/ralph#44)
- Branch: feat/codex-cli-parity

## Objective

ralph を Claude Code 専用ハーネスから Claude Code / Codex 両対応の単一ハーネスへ拡張し、Codex CLI でも標準フロー (spec → plan → work → self-review → verify → test → sync-docs → cross-review → pr) が Claude Code と同等に完走できるようにする。`ralph init` で `.claude/` / `.codex/` / `.agents/skills/` を always-on で配置し、AGENTS.md を両 CLI 共通の主導指示として運用する。

## Scope

- `templates/base/` への `.codex/` (`config.toml` + `hooks/` + `AGENTS.override.md`) と `.agents/skills/` 追加。
- 既存全 skill (spec, plan, work, self-review, verify, test, sync-docs, pr, cross-review, anti-bottleneck, audit-harness, release) の Codex 並走版作成 + Claude 版への CLI 別ガイダンス追記。
- `codex-review` → `cross-review` 文字列リネーム + 双方向 cross-model 化 (Claude 主→`codex exec review`、Codex 主→`claude -p`)。リネームは ripple **全箇所適用** (Loop scripts の文字列も string-only で同期。Loop の driver / 挙動変更は非対象 = #44)、互換 alias なし。
- Codex の `[features] codex_hooks = true` 設定と project trust 手順の同梱、effective loading の検証 (`ralph doctor`)。
- `internal/scaffold/` (go:embed) と `internal/cli/{init,upgrade,doctor}.go` の拡張。
- `internal/config/` に `[doctor] require_codex_cli` 追加。
- `scripts/check-skill-sync.sh` (新設) と `scripts/run-verify.sh` への統合、CI でゲート化。
- AGENTS.md / CLAUDE.md / README.md / `.claude/rules/` / `docs/quality/` / `docs/recipes/` の更新。
- 実機 smoke test: Codex CLI で `/spec`〜`/pr` 完走確認。

## Non-goals

- Ralph Loop の **Codex driver 化** (#44 で別途扱う)。`ralph-orchestrator.sh` / `ralph-pipeline.sh` を `codex exec` で駆動できるようにする実装変更は本プランでは行わない。
  - 例外: `codex-review` → `cross-review` の **文字列リネームは Loop scripts (`ralph-pipeline.sh`, `ralph-orchestrator.sh`, `check-pipeline-sync.sh`, prompts) にも波及**させる。これは挙動変更なし (Claude 駆動のままで naming だけ揃える)。
- `ralph spec`/`ralph plan`/`ralph work` 等の Go CLI ラッパー追加 (Phase 2 評価)。
- Codex `[features] multi_agent` を使った真の並列 subagent 化 (Phase 2 評価)。
- AskUserQuestion 相当を提供する MCP server の自前実装 (Phase 2 評価)。
- `codex:rescue` プラグインの ralph 同梱 (既存維持)。
- `/codex-review` 互換 alias の作成 (ユーザ指示で不要と確定)。
- Codex 独自 slash command 機構 (built-in `/plan` 等) との衝突を避けるための skill 名変更 — skill 名は Claude と完全一致させ、Codex 側は **`$skill-name` mention** または `/skills` メニューで起動する運用を採用 (docs に明記)。

## Assumptions

- Codex CLI のスキル機構は 2026-05 時点の仕様 ([Agent Skills – Codex](https://developers.openai.com/codex/skills)) で安定しており、`.agents/skills/<name>/SKILL.md` (frontmatter: `name`, `description`) と任意 `agents/openai.yaml` を読む。
- AGENTS.md の階層マージは Codex 公式仕様 ([AGENTS.md – Codex](https://developers.openai.com/codex/guides/agents-md)) に従い、`AGENTS.override.md` で nested 上書きが効く。`project_doc_max_bytes` 既定 32 KiB。
- 既存 `internal/upgrade/` の hash-based diff engine は新規ディレクトリ追加 (`.codex/`, `.agents/skills/`) とファイル削除 (`.claude/skills/codex-review/`) を auto-update / conflict / add / remove のいずれかとして扱える。挙動は Slice 2 着手時に確認する。
- 既存 hooks (`.claude/hooks/commit-msg-guard.sh`, `mojibake_check.sh`) のロジックは CLI 非依存で、`.codex/config.toml` の `[hooks]` 経由でも同じ `scripts/` を参照すれば再利用できる。
- Codex の project-scoped config (`.codex/config.toml`) と `[hooks]` は **trusted project** + `[features] codex_hooks = true` でなければロードされない ([Codex config reference](https://developers.openai.com/codex/config-reference) / [Codex hooks](https://developers.openai.com/codex/hooks))。`ralph doctor` と `.codex/README.md` で明示的にカバーする。
- skill drift check の対象は本文 + frontmatter (`name`, `description`, `agents/openai.yaml` の `policy.allow_implicit_invocation`) を含めた skill 起動メタデータの一致確認。Claude 固有フィールド (`disable-model-invocation`, `allowed-tools`) と Codex 固有フィールドはそれぞれの側のみで保持し、対応関係 (Claude `disable-model-invocation: true` ⇔ Codex `policy.allow_implicit_invocation: false`) を check する。
- Codex の skill 起動構文は **`$skill-name` mention または `/skills` メニュー**である ([Codex skills](https://developers.openai.com/codex/skills))。`/skill-name` 形式は Codex の built-in slash command と衝突するため使わない (例: `/plan` は Codex built-in)。

## Affected areas

| エリア | 影響 |
|--------|------|
| `templates/base/.codex/` (新設) | `config.toml`, `hooks/`, `AGENTS.override.md`, `README.md` 追加 |
| `templates/base/.agents/skills/` (新設) | 全 skill の Codex 仕様 SKILL.md |
| `templates/base/.claude/skills/codex-review/` | `cross-review/` にリネーム、本文を双方向化 |
| `templates/base/.claude/skills/{spec,plan,work,self-review,verify,test,sync-docs,pr,anti-bottleneck,audit-harness,release}/SKILL.md` | CLI 別ガイダンス追記 |
| `templates/base/AGENTS.md` | 主導指示として再構成、Codex 起動手順追加 |
| `templates/base/CLAUDE.md` | 薄型化、Claude 固有のみ残す |
| `templates/base/ralph.toml` | `[doctor] require_codex_cli` 追加 (`[pipeline]` の model は Claude のまま、Codex 用 model は `.codex/config.toml` に gpt-5.5) |
| `internal/scaffold/`, `templates.go` | go:embed パスに `.codex/`, `.agents/skills/` 追加、manifest 更新 |
| `internal/cli/{init,upgrade,doctor}.go` | 新配置の展開、Codex CLI 検出 |
| `internal/config/` | `require_codex_cli` フィールド |
| `internal/upgrade/` | 新規追加・rename・削除の挙動確認 (テスト追加) |
| `scripts/check-skill-sync.sh` (新設) | drift 検出 |
| `scripts/run-verify.sh` | drift check 統合 |
| `.claude/rules/post-implementation-pipeline.md` | `/cross-review` リネーム、CLI 別実行モード |
| `.claude/rules/subagent-policy.md` | Codex=順次 inline 節 |
| `README.md`, `CLAUDE.md`, `AGENTS.md` | リネーム反映、Codex 起動手順 |
| `docs/quality/definition-of-done.md` | 両 CLI green 条件 |
| `docs/recipes/` | Codex 起動例 |
| `tests/`, `internal/scaffold/embed_test.go` | 新配置検証 |
| `.github/workflows/` | drift check ジョブ |
| `scripts/ralph-pipeline.sh`, `scripts/ralph-orchestrator.sh`, `scripts/check-pipeline-sync.sh` | string-only リネーム (codex-review/codex-triage/`.codex_triage`)。挙動変更なし。Loop driver 化は #44 |
| `.claude/skills/loop/prompts/pipeline-*.md` | string-only リネーム (codex-review 参照、レポート glob `codex-triage-*` → `cross-review-triage-*`) |

リネーム対象 (`codex-review` → `cross-review`) で参照を含むファイル: `.claude/skills/work/SKILL.md`, `.claude/skills/loop/SKILL.md`, `.claude/skills/loop/prompts/pipeline-*.md`, `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`, `CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md`, `scripts/ralph-pipeline.sh` (lines 715, 751 ほか), `scripts/ralph-orchestrator.sh`, `scripts/check-pipeline-sync.sh` (line 20)、ほか grep で発見次第。

## Design decisions

### 仕様段階で確定済み (spec から継承)

- **D-1**: `ralph init` 時に CLI 選択は **行わず**、常に `.claude/` と `.codex/` の両方を always-on で配置する。動機: チーム内差異吸収。
- **D-2**: AGENTS.md を両 CLI 共通の source of truth とする (32 KiB cap)。Claude 固有は CLAUDE.md、Codex 固有は `.codex/AGENTS.override.md` (Codex agent 向け指示) と `.codex/README.md` (人間向け説明) に分離。
- **D-3**: skill は `.claude/skills/` と `.agents/skills/` を完全並走し、`scripts/check-skill-sync.sh` で本文 + 起動メタデータ (frontmatter `name`/`description`、`agents/openai.yaml` の `policy.allow_implicit_invocation`) の drift を CI 検出する。Claude 固有 (`disable-model-invocation`, `allowed-tools`) と Codex 固有フィールドは対応関係チェック (`disable-model-invocation: true` ⇔ `allow_implicit_invocation: false`) のみ実施。
- **D-4**: post-implementation pipeline は Codex 側で順次 inline 実行 (Claude のサブエージェント並列ではなく単一 agent 内連続実行)。レポートはファイル経由で同等性を担保。
- **D-5**: 対話は Codex 側で stdin 番号付き選択肢に fallback (`AskUserQuestion` 相当の構造化対話なし)。
- **D-6**: `/codex-review` を **`/cross-review`** にリネーム + 双方向化。Claude 主→`codex exec review`、Codex 主→`claude -p` で reviewer 役。互換 alias は作成しない。参照箇所も全リネーム。
- **D-7**: Codex 用 model 既定値は `gpt-5.5`。`.codex/config.toml` の `model` に明記。
- **D-8**: Codex skill の implicit invocation 制御は Claude `disable-model-invocation` と対応関係を保つ。`spec` と `release` のみ `agents/openai.yaml` で `policy.allow_implicit_invocation: false`、それ以外は既定 (true)。

### 本プランで解決した critical forks

- **D-9 (Skill 本文の CLI 別ガイダンス記載方式)**: **末尾に補遺セクション**を採用。各 SKILL.md 末尾に `## CLI 別実行ガイダンス` を追加し、Claude/Codex それぞれの適用テーブル (subagent vs inline、AskUserQuestion vs stdin、driver コマンド) を記述する。本文本体は CLI 中立、drift check は補遺を含めた本文一致で照合する。理由: 13 skill 全部に同じ構造で適用でき、本文の可読性を保ちつつ drift check を単純化できる。
- **D-10 (Codex `approval_policy` 既定値)**: **`on-request`** を採用。`.codex/config.toml` テンプレートの既定値とする。`sandbox_mode = "workspace-write"` と組み合わせ、ツールが明示的に承認要求した時のみユーザに尋ねる。理由: セッション中の確認頻度を抑えつつ destructive 操作の checkpoint を残す、Claude の `permission_mode = auto` に近い UX を提供する。

## Acceptance criteria

- [ ] **AC-1**: `ralph init <dir>` 実行後、`<dir>/.claude/`, `<dir>/.codex/`, `<dir>/.agents/skills/` の三系統と `<dir>/AGENTS.md`, `<dir>/CLAUDE.md`, `<dir>/ralph.toml` が配置されている。`tests/` で検証。
- [ ] **AC-1b (effective config)**: `ralph doctor` が `.codex/config.toml` の **effective loading** を確認する: project trust 状態、`[features] codex_hooks = true`、少なくとも 1 つの `[hooks]` エントリが Codex から見える。未 trust の場合は warning + trust 手順案内。
- [ ] **AC-2**: Codex CLI で `$spec` → `$plan` → `$work` → post-impl pipeline → `$pr` を **skill mention 形式** または `/skills` メニューで起動した場合、`docs/specs/<file>.md`, `docs/plans/active/<file>.md`, `docs/reports/{self-review,verify,test,sync-docs,cross-review-triage}-*.md` が生成され、PR が作成される。Slice 7 の smoke test で確認。`/skill-name` 形式は Codex built-in と衝突するため使わない。
- [ ] **AC-3**: `./scripts/check-skill-sync.sh` が `.claude/skills/` と `.agents/skills/` の本文 + 起動メタデータ drift を検出する (差分なし: exit 0 / 差分あり: exit 1 + 該当ファイル名)。具体的には: skill インベントリ (両側に同じ skill 名集合)、本文一致 (frontmatter 除外)、`name`/`description` 一致、`policy.allow_implicit_invocation` の対応関係 (Claude `disable-model-invocation` と整合)。CI (GitHub Actions) で実行。
- [ ] **AC-4**: post-implementation pipeline が Codex 側で順次 inline 実行され、各ステップが `docs/reports/*.md` を生成する。`reviewer` / `verifier` / `tester` / `doc-maintainer` 相当の責務が単一 agent 内で連続実行される。
- [ ] **AC-5**: `/cross-review` が双方向で動作する。Codex 主フローでは `claude -p`、Claude 主フローでは `codex exec review` を呼び、いずれも `docs/reports/cross-review-triage-<slug>.md` を生成する。
- [ ] **AC-6**: `ralph doctor` が `claude` / `codex` の両 CLI 検出 + version 表示 + AC-1b の effective config 検証を行う。片方のみインストールでも warning に留め、エラーにしない。
- [ ] **AC-7**: `./scripts/run-verify.sh` が green。AGENTS.md がプロジェクト直下で 32 KiB 以内。`audit-harness` skill が AGENTS.md サイズ警告を出す。
- [ ] **AC-8**: `tests/` に `.codex/` および `.agents/skills/` の go:embed 検証 (`internal/scaffold/embed_test.go` 拡張) と `ralph init` 展開検証が追加され、green。
- [ ] **AC-9 (実装視点)**: `codex-review` を含む全ての参照が `cross-review` にリネーム済みで、grep `codex-review` のヒット 0 件。**string-only rename ripple は Loop scripts (`ralph-pipeline.sh`, `ralph-orchestrator.sh`, `check-pipeline-sync.sh`, `.claude/skills/loop/prompts/`) も対象**。許可リスト (rename 対象外): `.claude/skills/cross-review/` 自身、過去 archive (`docs/plans/archive/`)、`docs/reports/` 配下の不変履歴アーティファクト (過去の self-review / verify / test / sync-docs / walkthrough / codex-triage- レポート — 当時のパイプライン状態を記述する履歴であり書き換えると意味を変える)、本プラン本体 (履歴文脈)。新規生成のレポートは `cross-review-triage-*` 命名を使用。

## Implementation outline

### Slice 1: テンプレート基盤と AGENTS.md 再構成

1. `templates/base/.codex/` 骨組み新設:
   - `config.toml`: `model = "gpt-5.5"`, `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"`, `[features] codex_hooks = true`, `[hooks]` テーブル雛形。
   - `hooks/`: 既存 `scripts/commit-msg-guard.sh` 等を `[hooks]` から参照する設定。
   - `AGENTS.override.md`: Codex agent 向け指示 (順次 inline、`$skill-name` 起動、stdin 番号選択)。
   - `README.md`: **project trust 設定手順** (`codex trust .` 等)、`/skill-name` 構文を使わない理由 (built-in 衝突)、permission/sandbox の Claude 対応表。
2. `templates/base/.agents/skills/` ディレクトリ新設 (中身は Slice 4 で埋める)。
3. `templates/base/AGENTS.md` を両 CLI 主導指示として再構成: Codex 起動手順 (`/skills`, `$<skill>` mention、暗黙起動)、`/skill-name` 構文非推奨の注記、両 CLI 既知差異表、32 KiB 内に収める。
4. `templates/base/CLAUDE.md` 薄型化: Claude 固有のみ残す。
5. 検証: `templates/base/AGENTS.md` のサイズ確認、`go test ./internal/scaffold/...` 通過。

### Slice 2: ralph CLI 拡張

6. `internal/scaffold/`: `.codex/`, `.agents/skills/` の go:embed パス追加、manifest TOML 更新。
7. `internal/cli/init.go`, `upgrade.go`: 新配置展開ロジック追加。upgrade の rename (`codex-review` → `cross-review`) 挙動を確認。`internal/upgrade/` の hash-based diff engine が rename を `add + remove` として扱う場合のテスト fixture を追加。
8. `internal/cli/doctor.go`: codex CLI 検出 + **effective config 検証** を追加。
   - codex CLI 存在 / version (既存 `scripts/codex-check.sh` 相当)
   - project trust 状態の確認 (`codex` 側コマンド or 設定ファイル読み取り)
   - `.codex/config.toml` の `[features] codex_hooks` フラグ確認
   - `[hooks]` テーブルが Codex 視点で見えるかの probe (例: `codex hooks list` 相当があれば呼び出し、無ければ TOML パース)
   - 失敗時は warning に留め、エラーで止めない
9. `internal/config/`: `[doctor] require_codex_cli` フィールド追加。
10. `templates/base/ralph.toml`: `[doctor] require_codex_cli = false` (既定 false で warning のみ) を追加。
11. 検証: `internal/cli/cli_test.go` 拡張、`ralph init` / `ralph doctor` の単体テスト + effective config 確認テスト (mock TOML)。

### Slice 3: `codex-review` → `cross-review` リネームと双方向化

12. `templates/base/.claude/skills/codex-review/` → `cross-review/` リネーム (git mv)。同様に worktree のトップレベル `.claude/skills/codex-review/` もリネーム。
13. SKILL.md を双方向化: driver 検出ロジック (`RALPH_PRIMARY_CLI=claude|codex` env、未設定時は `which` 検出順) で分岐、Claude 主→`codex exec review`、Codex 主→`claude -p` reviewer。
14. 出力ファイル名・glob を `cross-review-triage-<slug>.md` / `cross-review-triage-*` に変更。
15. 全参照リネーム (Claude 系):
    - `.claude/skills/work/SKILL.md`, `.claude/skills/loop/SKILL.md`
    - `.claude/skills/loop/prompts/pipeline-*.md` 内の `codex-review` / `codex-triage-*` / `.codex_triage`
    - `.claude/rules/post-implementation-pipeline.md`, `.claude/rules/subagent-policy.md`
    - `CLAUDE.md`, `AGENTS.md`, `README.md`
    - `docs/quality/definition-of-done.md`, `docs/quality/quality-gates.md`
16. **Loop scripts string-only リネーム** (挙動変更なし、Codex driver 化は #44):
    - `scripts/ralph-pipeline.sh` (line 715, 751 ほか): `codex-review` フェーズ名・ログ名・report glob・checkpoint key (`.codex_triage`)・event 名
    - `scripts/ralph-orchestrator.sh`: 同様の文字列
    - `scripts/check-pipeline-sync.sh` (line 20 ほか): canonical order の `codex-review` 表記、参照 file リスト
    - 検証: 既存 Loop の Claude 駆動が壊れないこと (`./scripts/check-pipeline-sync.sh` を rename 後に走らせて pass)
17. `templates/base/` 内の同等パスもリネーム (`templates/base/.claude/skills/...`、`templates/base/scripts/...` がある場合)。
18. 検証: `grep -r "codex-review"` のヒット 0 件 (許可リスト: `.claude/skills/cross-review/`、`docs/reports/cross-review-triage-*`、`docs/plans/archive/`、本プラン本体)。`./scripts/check-pipeline-sync.sh` を実行して pass。

### Slice 4: Skill 並走化 + CLI 別ガイダンス追記

18. 各 Claude skill の SKILL.md に CLI 別ガイダンス追記 (D-9 で確定する記載方式に従う): spec, plan, work, self-review, verify, test, sync-docs, cross-review, pr, anti-bottleneck, audit-harness, release。
19. `.claude/agents/{reviewer,verifier,tester,doc-maintainer}/` の責務を該当 skill の Codex セクションに展開 (Codex は単一 agent 内連続実行のため subagent 定義不要)。
20. `.agents/skills/<name>/SKILL.md` ミラー作成: Codex frontmatter (`name`, `description` のみ)、本文は Claude 版から継承 (CLI 別ガイダンス節は両方に同一文字列で含める)。
21. `spec`, `release` には `agents/openai.yaml` を併設し `policy.allow_implicit_invocation: false`。
22. 検証: skill 数の一致 (`.claude/skills/` ↔ `.agents/skills/`)、frontmatter 構文確認。

### Slice 5: drift check と verify 統合

23. `scripts/check-skill-sync.sh` 新設。次の 5 種類のチェックを実装:
    1. **インベントリ一致**: `.claude/skills/<name>/` と `.agents/skills/<name>/` の skill 名集合が同一 (片側だけ存在する skill があれば fail)。
    2. **本文一致** (frontmatter 除外、normalize 後): trailing whitespace と改行コード統一後に diff。
    3. **`name` 一致**: 両 SKILL.md の frontmatter `name` フィールドが一致。
    4. **`description` 一致**: 両 frontmatter の `description` が一致 (Codex の暗黙起動に直結)。
    5. **`policy.allow_implicit_invocation` 対応関係**: Claude 側 `disable-model-invocation: true` ⇔ Codex 側 `agents/openai.yaml` の `policy.allow_implicit_invocation: false`。Claude 側に `disable-model-invocation` がない skill は Codex 側も既定 (true) であること。
    差分時は exit 1 + 該当ファイル名 + どのチェックで fail したかを出力。
24. `scripts/run-verify.sh` に drift check 呼び出しを統合。
25. `.github/workflows/` に drift check 実行ジョブ追加 (既存 verify ジョブから呼ぶ)。
26. 検証: 故意に片側 (本文 / name / description / policy のいずれか) を変更して drift check が検出することを確認、両側同期で exit 0。

### Slice 6: ルール / ドキュメント整合

27. `.claude/rules/post-implementation-pipeline.md`: `cross-review` リネーム反映、CLI 別実行モード節 (Codex=順次 inline) 追加。
28. `.claude/rules/subagent-policy.md`: Codex=順次 inline 節追加、リネーム反映。
29. `CLAUDE.md`: 薄型化、Codex 関連を AGENTS.md / `.codex/` に移譲。
30. `README.md`: Codex 起動手順 (`$skill-name` mention または `/skills` メニュー、`/skill-name` は使わない理由)、既知差異表、`cross-review` 記述、project trust 設定手順、pre-upgrade backup 推奨。
31. `docs/quality/definition-of-done.md`: 両 CLI green 条件追加 (effective config 含む)。
32. `docs/recipes/` に Codex 起動レシピ追加 (project trust → `ralph doctor` → `$skill` mention)。upgrade 中断時のリカバリ手順も追加。
33. `audit-harness` skill に AGENTS.md サイズ警告ロジック追加 (32 KiB 接近時)。
34. 検証: `./scripts/run-verify.sh` green、grep `codex-review` 残存 0 件。

### Slice 7: 検証 / smoke test

35. `internal/scaffold/embed_test.go` 拡張: `.codex/`, `.agents/skills/` の埋め込み検証ケース。
36. `tests/` に `ralph init` 展開検証 (新配置すべて存在)、`tests/upgrade_downgrade_test.go` 新設 (新→旧→新 往復で manifest が破綻しない)。
37. Codex CLI で実機試走: `$spec` → `$plan` → `$work` → post-impl → `$pr` の skill mention 起動 (または `/skills` メニュー)。`/skill-name` 形式は **使わない** ことを記録。手動確認結果を `docs/reports/walkthrough-2026-MM-DD-codex-cli-parity.md` に記録。
    - 試走前に `codex trust .` 等で project trust を有効化したことを記録
    - `ralph doctor` の effective config 出力 (AC-1b) もレポートに添付
    - Codex CLI 未インストール環境では SKIP 可、CI 上は対象外
38. 検証: `./scripts/run-verify.sh` final green、smoke test レポート存在。

## Verify plan

- **Static analysis checks**:
  - `go vet ./...` 通過
  - `gofmt -l .` で差分なし
  - `golangci-lint run` (既存設定) 通過
  - `scripts/check-skill-sync.sh` exit 0 (本文 + name + description + policy.allow_implicit_invocation の 5 種 check pass)
  - `scripts/check-template.sh` 通過
  - `scripts/check-sync.sh` 通過
  - `scripts/check-pipeline-sync.sh` 通過 (Loop scripts も rename 反映後)
  - AGENTS.md のサイズが 32 KiB 以内 (`wc -c templates/base/AGENTS.md` で確認)
- **Spec compliance criteria to confirm**:
  - spec の F-1〜F-9 が plan の Slice にマップされていること
  - spec の AC-1〜AC-8 が plan の AC-1〜AC-9 (+ AC-1b) にカバーされていること
  - 互換 alias を作成していないこと
  - `grep -r "codex-review"` の残存 0 件 (許可リスト: `.claude/skills/cross-review/`、`docs/reports/cross-review-triage-*`、`docs/plans/archive/`、本プラン本体の文脈記述)
- **Documentation drift to check**:
  - `/cross-review` 命名が CLAUDE.md / AGENTS.md / README.md / `.claude/rules/` / `.claude/skills/` / `docs/quality/` / `docs/recipes/` で一貫
  - 後方互換 alias を作っていないことが docs にも反映
  - 既知差異表が README.md と AGENTS.md に存在
- **Evidence to capture**:
  - `docs/reports/verify-2026-MM-DD-codex-cli-parity.md` (verifier 出力)
  - `templates/base/AGENTS.md` のサイズ (verify レポートに記載)
  - drift check の正例・反例の挙動 (verify レポートに添付)

## Test plan

- **Unit tests**:
  - `internal/scaffold/embed_test.go`: `.codex/config.toml`, `.codex/hooks/`, `.codex/AGENTS.override.md`, `.codex/README.md`, `.agents/skills/<n>/SKILL.md` の埋め込みアサーション
  - `internal/cli/init_test.go` (新設または既存拡張): `ralph init <tmp>` 後の三系統配置確認
  - `internal/cli/doctor_test.go`: `codex` CLI 未検出時の warning、検出時の version 表示
  - `internal/config/`: `require_codex_cli` フィールドのパース
  - `internal/upgrade/`: `codex-review/` → `cross-review/` のリネームが add+remove として検出されること、新規 `.codex/` の add 検出
- **Integration tests**:
  - `scripts/check-skill-sync.sh` 単体: 同期時 exit 0 / 故意 drift 時 exit 1 (本文 / name / description / policy 各観点で個別ケース)
  - `tests/`: `ralph init` で展開後、`scripts/check-skill-sync.sh` が green
  - `tests/`: `ralph upgrade` で旧 `.claude/skills/codex-review/` 削除と `.claude/skills/cross-review/` 追加
  - `tests/`: `./scripts/check-pipeline-sync.sh` が rename 後も green (Loop scripts 含めた一致確認)
  - `tests/upgrade_downgrade_test.go`: 新→旧→新 往復で manifest と embed 整合性が保たれること
- **Regression tests**:
  - 既存 `internal/scaffold/embed_test.go` の `.claude/`, `CLAUDE.md` テストが引き続き green
  - 既存 `internal/cli/cli_test.go` 全テスト green
  - `./scripts/run-verify.sh` 全体 green
- **Edge cases**:
  - `codex` CLI が未インストールの環境で `ralph init` / `ralph doctor` が破綻しないこと
  - `.codex/config.toml` 内の MCP サーバ placeholder が空でも構文エラーにならないこと
  - skill 名に重複や空白が混入した場合の drift check 挙動
  - 32 KiB 直前まで AGENTS.md を膨らませた場合に audit-harness が警告すること
  - project trust 未取得状態で `ralph doctor` を実行した時、warning + trust 手順案内が表示されること
  - `[features] codex_hooks = true` が設定されていない `.codex/config.toml` でも warning が出ること
  - `disable-model-invocation` がない skill (大半) で Codex 側に `agents/openai.yaml` が **無い** ことが drift check で許容されること (両側既定 true)
  - `codex-review` 文字列が新規 file として混入した場合に AC-9 grep が検出すること (ラインカバレッジ)
  - upgrade を `codex-review` 削除直後に中断 → 再開で `cross-review` が正しく追加されること
- **Evidence to capture**:
  - `docs/reports/test-2026-MM-DD-codex-cli-parity.md` (tester 出力)
  - smoke test (`docs/reports/walkthrough-2026-MM-DD-codex-cli-parity.md`)
  - drift check の正例・反例ログ

## Risks and mitigations

| ID | リスク | 影響 | 緩和策 |
|----|--------|------|--------|
| R-1 | Codex skill 仕様の変更 (frontmatter / agents.yaml の field 変更) | 既存 `.agents/skills/` が読まれなくなる | drift check に Codex 仕様準拠チェックを段階追加、定期キャッチアップを `audit-harness` に組み込む |
| R-2 | 順次 inline 実行で Codex のコンテキスト窓圧迫 | post-impl pipeline の質低下 | 既存 `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2` キャップ流用、レポートはファイル経由で context bloat を抑制 |
| R-3 | stdin fallback で構造化選択肢の取り違え | spec/plan の意図不一致 | skill 本文で番号付き選択肢を必須化、`Codex の場合: 数字のみで返答してください` を明記 |
| R-4 | skill 完全並走で本文 drift | UX 不整合 | `scripts/check-skill-sync.sh` を CI で必須化、PR check で fail させる |
| R-5 | Claude `permission_mode` と Codex `sandbox_mode + approval_policy` の意味差 | テンプレート初期値の意図がずれる | `.codex/README.md` と AGENTS.md に対応表を明記、`ralph.toml` には CLI 別マッピングコメントを置く |
| R-6 | AGENTS.md 32 KiB cap | 拡張時に切れる | 現在 5.2k で余裕。`audit-harness` に閾値警告 (例: 24 KiB / 32 KiB) を追加 |
| R-7 | `internal/upgrade/` の hash-based diff engine が rename を auto-update できない | 既存ユーザの upgrade が手動 conflict 解決を要求、中断時に部分状態が残る | Slice 2 開始時に挙動を確認、必要なら upgrade engine 側に rename 規則を追加。pre-upgrade backup 手順を CHANGELOG で必須化、`tests/upgrade_downgrade_test.go` で往復検証 |
| R-8 | Codex CLI 未インストール環境で smoke test が走らない | AC-2 / AC-5 の最終確認が SKIP | CI 上は SKIP 許可、開発環境で手動確認。`docs/reports/walkthrough-*.md` に試走条件を明記 |
| R-9 | 双方向 cross-review の driver 検出ロジックが脆弱 | Claude 主のときに誤って Claude を呼ぶ等 | env (`RALPH_PRIMARY_CLI=claude\|codex`) を明示設定、未設定時は `which codex` / `which claude` の検出順で fallback、SKILL.md にルール記載 |
| R-10 | `codex-review` リネームで CI/外部参照が壊れる | PR template や docs の死リンク、Loop Claude 駆動の挙動破綻 | grep + lint で残存検出、Slice 3 で Loop scripts も string-only リネーム、`./scripts/check-pipeline-sync.sh` で gate、PR 作成前に `./scripts/run-verify.sh` で再 gate |
| R-11 | Codex skill の暗黙起動が `description` ドリフトで誤動作 | Skill が trigger されない / 別 skill が誤起動 | drift check に `name`/`description`/`policy` 一致チェックを含める (Slice 5)、CI で fail させる |
| R-12 | Codex の `/skill-name` 形式が built-in slash command (例: `/plan`, `/review`) と衝突 | ユーザが `/plan` 入力で Codex built-in が走り、ralph の `plan` skill が起動しない | docs / AGENTS.md / `.codex/README.md` に `$skill-name` mention または `/skills` メニュー使用を明記、`/skill-name` 形式は使わない方針を Non-goals にも記録 |
| R-13 | `.codex/config.toml` が trust 未取得 / `[features] codex_hooks` 未設定で読まれない | hooks / approval / model 設定が silently 効かず、安全性とパリティの主張が無効 | `ralph doctor` で effective config を probe (AC-1b)、`.codex/README.md` で trust 手順案内、テンプレに `[features] codex_hooks = true` を含める |

## Rollout or rollback notes

- **Rollout (リポジトリ側)**:
  - 本プランは単一 PR で標準フロー全体に影響するため、Slice 単位の commit で履歴を分離する。
  - PR マージ後、既存ユーザは `ralph upgrade` で `.codex/` と `.agents/skills/` を取得。`codex-review` が消えて `cross-review` に置き換わるため、CHANGELOG / リリースノートで明記する。
  - `ralph release` (homebrew tag) は本対応のリリースタイミングを別途調整。
- **Rollout (ユーザ側 — 既存プロジェクトの upgrade)**:
  - **Pre-upgrade backup を必須化**: `ralph upgrade` 実行前に `.claude/`, `.codex/` (存在すれば), `AGENTS.md`, `CLAUDE.md`, `ralph.toml` のスナップショットを git commit するか、`ralph upgrade --dry-run` で差分確認。CHANGELOG とドキュメントに手順を明記する。
  - upgrade 中断時のセーフティ: `internal/upgrade/` のトランザクション境界を確認。`codex-review/` 削除と `cross-review/` 追加の間で中断した場合のリカバリ手順を docs/recipes に追加。
- **Rollback**:
  - 全 Slice を 1 PR にまとめる。問題発生時は revert で全戻し可能。
  - 部分 rollback は Slice 単位の commit で実施 (例: drift check のみ disable は CI ジョブ skip で対応)。
  - 既存ユーザの `ralph upgrade` で本変更を取得した後の downgrade は、(a) pre-upgrade backup を git checkout で復元、または (b) `ralph init --version=<previous>` で旧テンプレを再展開する。version pin だけでは追加された `.codex/` ファイルや manifest mutations は自動削除されないため、手順を docs/recipes に明記する。
  - downgrade fixture: `tests/upgrade_downgrade_test.go` (新設) で、新→旧→新 の往復で manifest が破綻しないことを確認。

## Open questions

- なし (D-9: 末尾に補遺セクション、D-10: `on-request` で確定済み)。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (by /work) — `feat/codex-cli-parity`
- [x] Slice 1 (テンプレート基盤) 完了 — commit `12064e6`
- [x] Slice 2 (ralph CLI 拡張) 完了 — commit `247526b`
- [x] Slice 3 (cross-review リネーム + 双方向化) 完了 — commit `4d16323`
- [x] Slice 4 (skill 並走化 + ガイダンス追記) 完了 — commit `1a1bc3d`
- [x] Slice 5 (drift check + verify 統合) 完了 — commit `2e40fac`
- [x] Slice 6 (ルール / ドキュメント) 完了 — commit `6eff2bb`
- [x] Slice 7 (検証 / smoke test) 完了 — commit `8c03ba8` (live Codex smoke test deferred to manual sign-off; recorded in walkthrough)
- [ ] Review artifact created (self-review)
- [ ] Verification artifact created (verify)
- [ ] Test artifact created (test)
- [ ] PR created
