# Ralph Loop Codex driver (Phase 2)

- Status: Draft
- Owner: Claude Code
- Date: 2026-05-08
- Related request: Ralph Loop の Codex 対応 (Phase 2)
- Related issue: 44
- Branch: feat/44/ralph-loop-codex-driver

## Objective

`/loop` (Ralph Loop) を Codex CLI でも完走させる。Phase 1 で標準フロー (`/work`) は両 CLI 対応済み。本フェーズでは `ralph-pipeline.sh` がハードコードしている `claude -p ...` を driver 抽象化レイヤー経由に置き換え、`ralph.toml` または環境変数で Claude / Codex を切り替えられるようにする。並列 worktree、integration ブランチ、unified PR、cycle 制御は CLI 非依存のまま維持する。

## Scope

- `scripts/ralph-cli-driver.sh` (新設) — `run_agent <prompt> <log> <extra_args>` を提供する小さなラッパー。`RALPH_LOOP_DRIVER=claude|codex` で `claude -p ...` か `codex exec ...` を切り替え。両者の出力を共通形 (`<log>` テキスト + `<log>.json` メタ) に揃える。
- `scripts/ralph-config.sh` — `RALPH_LOOP_DRIVER` 既定値 (`claude`) と Codex 用の `RALPH_CODEX_SANDBOX` (`workspace-write`) / `RALPH_CODEX_APPROVAL_POLICY` (`on-failure`) 既定値を追加。
- `scripts/ralph-pipeline.sh` — `run_claude()` を `run_agent()` (driver 中立) にリネームしつつ薄い委譲関数化。Preflight Probe 5 (JSON output 検査) は Claude driver 限定にし、Codex driver では `codex exec --help` をパースして `--output-last-message` と `--sandbox` の存在を確認する probe に置換。**cross-review ブロック (現行 line 715-760) を driver-aware ディスパッチャ化** — `driver=claude` → 既存 `codex exec review`、`driver=codex` → `claude -p` を adversarial reviewer プロンプトで呼ぶ。`<driver, reviewer>` ペアをログ + cross-review-triage レポートに記録。
- `scripts/ralph-orchestrator.sh` — `RALPH_LOOP_DRIVER` を子プロセス (`ralph-pipeline.sh`) にエクスポート。直接 CLI を呼ばない箇所は変更不要。
- `internal/config/config.go` — `[loop] driver = "claude"|"codex"` パース。既定 `"claude"`。
- **`internal/cli/run.go`** — `cfg.Loop.Driver` を読み取り、env 未設定時のみ `RALPH_LOOP_DRIVER` をエクスポート (env が勝つ priority)。同様に `RALPH_CODEX_SANDBOX` / `RALPH_CODEX_APPROVAL_POLICY` も TOML から伝播させる。これにより `ralph.toml` 単独設定でも runtime に効く。
- `templates/base/ralph.toml` — `[loop]` セクション (`driver`, `codex_sandbox`, `codex_approval_policy`) を追加。コメントで Codex 切替手順を案内。
- `internal/cli/doctor.go` — Loop driver の表示行追加 (effective: env > TOML > default)。
- `.claude/skills/loop/SKILL.md` & `.agents/skills/loop/SKILL.md` — driver 切替手順を CLI 別ガイダンスに追記。drift check が両者を強制。
- `.claude/skills/cross-review/SKILL.md` & `.agents/skills/cross-review/SKILL.md` — Loop 内のレビュアー反転ロジックを言及 (driver=codex 時は claude -p を呼ぶ)。
- `docs/recipes/ralph-loop.md` — Codex driver 起動例 (`codex trust .` → `RALPH_LOOP_DRIVER=codex ./scripts/ralph run ...`)。
- 既存 `pipeline-*.md` プロンプト群 — sidecar 書き込みを Claude/Codex 双方で必須化する文言を確認。差分があれば最小編集。
- **`tests/test-ralph-cli-driver.sh` (新設、決定論的 fake codex)** — `tests/fixtures/fake-codex` スタブ (PATH 経由で `codex` を上書き) を使い、argv 順序、stdin 経由のプロンプト渡し、`--output-last-message` 出力先、cwd、終了コード非ゼロケース、`.last` 不在ケースを実 wrapper で検証。`scripts/run-verify.sh` から呼ぶ。

## Non-goals

- Codex `[features] multi_agent = true` を使った真並列 subagent 化 (preview 機能、別タスク)。
- 標準フロー (`/work`) 側の追加変更 — Phase 1 で完了済み。
- `codex exec resume <session_id>` による Codex セッション復元 — 既存プロンプトはチェックポイント context を毎サイクル inline 注入する設計のため、Phase 2 では Codex driver はステートレス起動で運用する。
- `--output-schema` を用いた構造化 JSON 出力強制 — sidecar ファイル方式と冗長になるので採用しない。
- `--dangerously-bypass-approvals-and-sandbox` の利用 — 「外部 sandbox 環境専用」と公式ドキュメントに明記されているため Phase 2 では選ばない。
- 既存 ralph CLI (`ralph init`/`upgrade`) のテンプレ拡張 — `.codex/`, `.agents/skills/` は Phase 1 で配置済み。
- 既存 `ralph` shell wrapper のサブコマンド追加。

## Assumptions

- Phase 1 の `.codex/`, `.agents/skills/`, `scripts/check-skill-sync.sh`, `cross-review` 双方向化はすでに main にマージ済み (PR #45)。
- Codex CLI は `>= 0.128.0` (`AGENTS.md` の Codex setup checklist 既定) を前提。`--output-last-message` (`-o`) と `--full-auto` フラグが利用可能。
- Codex driver 利用前に `codex trust .` 済み (`.codex/config.toml` が読まれる前提)。
- ralph-pipeline.sh の sidecar 検出 (`.agent-signal` / `.self-review-result` / `.verify-result` / `.test-result` / `.pr-url`) は CLI 非依存ですでに動作する設計。プロンプトが正しく書き出していれば driver 切替で破綻しない。
- 並列 worktree (`git worktree add` × N) は Codex sandbox `workspace-write` と矛盾しない。各 worktree が独立した working root のため。

## Affected areas

| 影響対象 | 影響内容 | 深刻度 |
|---------|---------|--------|
| `scripts/ralph-cli-driver.sh` | 新設 (driver 抽象化、~80 行) | 中 |
| `scripts/ralph-config.sh` | `RALPH_LOOP_DRIVER` ほか追加 | 小 |
| `scripts/ralph-pipeline.sh` | `run_claude` → `run_agent` (delegating)、preflight probe 修正 | 中 |
| `scripts/ralph-orchestrator.sh` | driver 環境変数のエクスポート、ログ表示 | 小 |
| `internal/config/config.go` | `[loop]` セクションパース | 小 |
| `internal/config/config_test.go` | `[loop] driver` パーステスト | 小 |
| `internal/cli/doctor.go` | Loop driver 行追加 | 小 |
| `internal/cli/cli_test.go` | doctor 出力テスト追加 | 小 |
| `templates/base/ralph.toml` | `[loop]` セクション追加 | 小 |
| `.claude/skills/loop/SKILL.md` & `.agents/skills/loop/SKILL.md` | CLI 別ガイダンス、driver 切替コマンド | 中 |
| `docs/recipes/ralph-loop.md` | Codex driver 起動例追加 | 小 |
| `docs/quality/definition-of-done.md` | Codex driver 完走条件 | 小 |
| `tests/test-ralph-cli-driver.sh` (新設) | driver wrapper の単体テスト (dry-run) | 中 |
| `scripts/run-verify.sh` | 新テスト呼び出し | 小 |
| `AGENTS.md` / `templates/base/AGENTS.md` | Loop driver 言及を 1〜2 行追記 | 小 |
| `README.md` | Codex で Loop を回す手順 1 段落 | 小 |

## Design decisions

- **Driver 抽象化サイト**: `scripts/ralph-cli-driver.sh` を新設する。
  - 採用理由: `architecture.md` が「small, boring, well-named abstractions」を推奨。grep-able、単独テスト可能、`ralph-pipeline.sh` の `run_claude()` がただ `run_agent` を呼ぶだけになり関数肥大化を防ぐ。
  - 不採用案: `ralph-pipeline.sh` 内の if-branch — 関数肥大化、テスト分離困難。

- **Codex 構造化出力の取り回し**: ハイブリッド方式 — `codex exec -s "$RALPH_CODEX_SANDBOX" -c approval_policy="$RALPH_CODEX_APPROVAL_POLICY" --output-last-message <log_file>.last -` でエージェント最終メッセージを取得し、`.last` を `<log_file>` テキストとして扱う。構造化シグナルは既存 sidecar ファイル (`.agent-signal` / `.self-review-result` / `.verify-result` / `.test-result` / `.pr-url`) をそのまま流用する。`<log_file>.json` は driver 中立な薄い JSON `{"result": "<last message>", "session_id": null}` を Codex driver でも書き出して呼び出し側のパース経路を変えない。
  - 採用理由: 直接 `codex exec --help` で 0.128.0 上のフラグ可用性を検証 (2026-05-08)。`-s/--sandbox`, `-c/--config`, `-o/--output-last-message`, stdin プロンプト読み (`-`) のみが安定 API。`--full-auto` は **存在しない** ため当初案を破棄。`--ephemeral` は Codex セッション履歴に残さない用途で副作用が読みづらいため採用しない。
  - 不採用案 1: `--full-auto` 利用 — codex-cli 0.128.0 に存在しない (Codex 公式 SDK の experimental 名残)。実行時に「unknown flag」で Loop が起動前 fail するリスク。
  - 不採用案 2: `--dangerously-bypass-approvals-and-sandbox` — 外部 sandbox 環境専用と明記。
  - 不採用案 3: `--output-schema` で JSON Schema 強制 — phase ごとにスキーマを定義する保守コスト、エージェントのナラティブ消失、sidecar と二重化。
  - 不採用案 4: `--json` (JSONL event stream) — verbose で消費側が増え、現在 ralph に消費者が居ない。

- **Sandbox / approval policy**: Codex driver は `-s workspace-write` + `-c approval_policy=on-failure` を既定とし、`RALPH_CODEX_SANDBOX` / `RALPH_CODEX_APPROVAL_POLICY` で上書き可能にする。`approval_policy` は `-c` 経由 (TOML override) で渡す。
  - 採用理由: ralph-pipeline.sh は autonomous 実行を前提とするため Claude `--permission-mode bypassPermissions` 相当が必要。`workspace-write` は worktree 配下のみ書き込み可で、`approval_policy=on-failure` はコマンド失敗時のみ確認を求めるため、概ね Claude の bypass 相当の autonomy を確保する。

- **Cross-review の reviewer 反転**: `ralph-pipeline.sh` の cross-review ブロックは driver-aware ディスパッチャに変更する。
  - `driver=claude` → 既存の `codex exec review --base "$base"` (cross-model 維持)
  - `driver=codex` → `claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" --permission-mode plan` を adversarial reviewer プロンプトで呼び、出力を `docs/reports/cross-review-triage-*.md` に書く
  - どちらの分岐に入ったかを `report_event "cross-review"` の `details` に `driver`/`reviewer` フィールドで残し、後続の triage parsing で参照可能にする。
  - 採用理由: 「cross-review は別モデルが見る」という contract が driver=codex でも成立しないと品質ゲートが失われる。Phase 1 の `/cross-review` skill (双方向化済み) が同じパターンを採用しているので、Loop 側も足並みを揃える。

- **Driver スコープの設定 (ランタイム連携を含む二段構成)**: `ralph.toml [loop] driver` (静的既定) と `RALPH_LOOP_DRIVER` 環境変数 (実行時上書き) の二段。
  - 採用理由: Phase 1 と同じ priority pattern (`RALPH_PRIMARY_CLI`)。`internal/cli/run.go` で `cfg.Loop.Driver` を読み、env 未設定時のみ env として export することで TOML 設定が `./scripts/ralph run ...` 経由でも実行時に効く。`./scripts/ralph` shell wrapper を直接叩かれた場合は env 未経由なら既定の `claude` にフォールバックする (Codex driver を使うユーザは env か TOML 経由の `ralph run` を案内する)。Go 側 (`config.go`) は `ralph doctor` 表示用も兼ねる。

## Acceptance criteria

- [ ] **AC-1**: `scripts/ralph-cli-driver.sh` が存在し、`run_agent <prompt> <log> [extra_args]` を提供。`RALPH_LOOP_DRIVER=claude` で `claude -p --model ... --output-format json ...` を、`=codex` で `codex exec -s "$RALPH_CODEX_SANDBOX" -c approval_policy="$RALPH_CODEX_APPROVAL_POLICY" --output-last-message <log>.last -` を組み立てる。fake-CLI スタブを `PATH` 先頭に置いた状態で実 wrapper を実行し、両 driver の argv / stdin / cwd / 出力ファイル内容が assertion 通りであることを `tests/test-ralph-cli-driver.sh` で検証する。
- [ ] **AC-2**: `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` が green。preflight は `codex exec --help` をパースして `--output-last-message`, `-s/--sandbox`, `-c/--config` のいずれかが欠けていれば fail する Codex 専用プローブを行い、`json_output_format` プローブは driver=claude 限定。`claude_md_readable` プローブは driver=codex のとき `AGENTS.md_readable` に切り替わる。
- [ ] **AC-3**: `RALPH_LOOP_DRIVER=codex` で fake-codex を使った `ralph-pipeline.sh --dry-run` を完走させた際、Inner/Outer 各フェーズで `<log>.json` に `{"result":"...","session_id":null}` 形式の薄い JSON が書かれ、既存 sidecar 検出ロジック (`.agent-signal` / `.self-review-result` / `.verify-result` / `.test-result` / `.pr-url`) が一切変更不要で動く。
- [ ] **AC-4** (TOML→runtime ブリッジ): `internal/cli/run.go` が `cfg.Loop.Driver` / `cfg.Loop.CodexSandbox` / `cfg.Loop.CodexApprovalPolicy` を読み取り、対応する env (`RALPH_LOOP_DRIVER` 等) が未設定のときに限り export する。`ralph.toml [loop] driver = "codex"` のみ設定し env 未指定の状態で `ralph run --preflight --dry-run` を呼ぶと Codex 専用プローブが走る (= TOML 単独で Codex driver が選ばれる) こと、env と TOML 両方設定時は env が勝つことを `internal/cli/cli_test.go` 相当で検証する。
- [ ] **AC-5** (cross-review reviewer 反転): `ralph-pipeline.sh` の cross-review ブロックは driver-aware ディスパッチャになっている。`driver=claude` → `codex exec review --base "$base"` を呼び、`driver=codex` → `claude -p --permission-mode plan ...` を adversarial reviewer プロンプトで呼ぶ。fake-claude / fake-codex を組み合わせた dry-run で両分岐が選択され、`docs/reports/cross-review-triage-*.md` の `Reviewer:` フィールドが想定 CLI を指し、`report_event "cross-review"` JSONL に `{"driver":"...","reviewer":"..."}` が記録されていることを `tests/test-ralph-cli-driver.sh` で検証する。
- [ ] **AC-6**: `ralph status` / `ralph doctor` が effective driver (env > TOML > default) を表示する。env と TOML 食い違い時に effective 値と source ("env"/"toml"/"default") の両方を出す。
- [ ] **AC-7**: `.claude/skills/loop/SKILL.md` と `.agents/skills/loop/SKILL.md` の本文が同期しており、`./scripts/check-skill-sync.sh` が green。両側に Codex driver 切替コマンド (`RALPH_LOOP_DRIVER=codex ./scripts/ralph run ...`) と reviewer 反転の説明が記載されている。同様に `.claude/skills/cross-review/` 双方も同期。
- [ ] **AC-8**: `docs/recipes/ralph-loop.md` に Codex driver セクションがあり、`codex trust .` → `RALPH_LOOP_DRIVER=codex ./scripts/ralph run --plan ...` の最低 3 行手順 + sandbox/approval 上書き例が示されている。
- [ ] **AC-9**: `./scripts/run-verify.sh` が green。`tests/test-ralph-cli-driver.sh` が green。
- [ ] **AC-10**: `internal/config/config_test.go` に `[loop] driver = "codex"` のパーステストと不正値 (`"foo"`) エラー、`codex_sandbox` / `codex_approval_policy` のパーステストが追加されている。
- [ ] **AC-11** (実 Codex walkthrough — Codex CLI 利用可能なときのみ必須): Codex CLI が利用可能な開発環境で 1 度 `RALPH_LOOP_DRIVER=codex ./scripts/ralph run ...` を short prompt で実行し、`docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` に成功 / 失敗 / 既知差異を記録する。利用不可環境では fake-codex 統合テスト (AC-1, AC-3, AC-5) のみで AC を満たす。
- [ ] **AC-12** (後方互換): env も TOML も未設定のとき、`RALPH_LOOP_DRIVER` が `claude` として扱われ、既存 Claude Code ユーザのフローが一切変わらない。`ralph-pipeline.sh --dry-run` の出力が main ブランチ時点と diff ゼロ (driver 表示行を除く) であることを smoke test で確認する。

## Implementation outline

1. **設定基盤**: `ralph-config.sh` に `RALPH_LOOP_DRIVER` (既定 `claude`), `RALPH_CODEX_SANDBOX` (既定 `workspace-write`), `RALPH_CODEX_APPROVAL_POLICY` (既定 `on-failure`), `RALPH_CLAUDE_REVIEWER_MODEL` (既定 `claude-opus-4-7`) を追加。`templates/base/ralph.toml` に `[loop]` セクション (`driver`, `codex_sandbox`, `codex_approval_policy`) を追加。
2. **Go 側 config パース**: `internal/config/config.go` に `Loop` 構造体を追加 (`Driver`, `CodexSandbox`, `CodexApprovalPolicy`)。`Driver` は `claude`/`codex` のみ許容、`CodexSandbox` は `read-only`/`workspace-write`/`danger-full-access`、`CodexApprovalPolicy` は `untrusted`/`on-failure`/`on-request`/`never` のみ許容。`internal/config/config_test.go` にパーステストを追加。
3. **TOML→env ブリッジ**: `internal/cli/run.go` に env propagation を追加 — `cfg.Loop.Driver` / `cfg.Loop.CodexSandbox` / `cfg.Loop.CodexApprovalPolicy` をそれぞれ `RALPH_LOOP_DRIVER` / `RALPH_CODEX_SANDBOX` / `RALPH_CODEX_APPROVAL_POLICY` として export (env 未設定時のみ)。これで `ralph.toml` 単独設定でもランタイムに効く。
4. **`ralph-cli-driver.sh` 新設**: `run_agent <prompt_file> <log_file> [extra_args]` を export。内部で `case "$RALPH_LOOP_DRIVER" in claude) ...; codex) ...; *) echo "unsupported driver"; exit 1;; esac`。Codex 分岐は `codex exec -s "$RALPH_CODEX_SANDBOX" -c approval_policy="$RALPH_CODEX_APPROVAL_POLICY" --output-last-message "${log_file}.last" -C "$PWD" - < "$prompt_file"` を実行 (stdin 経由でプロンプトを渡す Codex 標準)。終了後に `.last` を `<log>` にコピー、`<log>.json` に `jq -n --arg r "$(cat $log)" '{result:$r,session_id:null}'` を書く。`.last` 不在時は空文字 + 警告ログ。Claude 分岐は現行 `run_claude` のロジックを移植。
5. **`ralph-pipeline.sh` 統合**: `run_claude()` を削除し `. "${SCRIPT_DIR}/ralph-cli-driver.sh"` で `run_agent` を読み込み、呼び出し側を `run_agent ...` に統一。Preflight Probe を driver-aware に書き換え:
   - Probe 1 (CLI): `claude` か `codex` のいずれかを driver に応じて要求
   - Probe 3 (context readable): driver=claude → `claude_md_readable`、driver=codex → `AGENTS.md_readable`
   - Probe 5: driver=claude → 既存 `json_output_format`、driver=codex → `codex exec --help` で `--output-last-message`/`-s`/`-c` 全部の存在を確認 (1 つでも欠けたら fail)
6. **cross-review ディスパッチャ化**: `ralph-pipeline.sh` の cross-review ブロック (現行 line 715-760) を driver-aware に書き換え。`driver=claude` → 既存 `codex exec review --base "$base"` を維持、`driver=codex` → `claude -p --model "$RALPH_CLAUDE_REVIEWER_MODEL" --permission-mode plan` を `.claude/skills/cross-review/prompts/adversarial-claude.md` (新設、既存 codex review 指示書を Claude 向けにポート) で呼び出し、出力を triage report に書く。`report_event "cross-review"` の details に `{"driver":"...","reviewer":"...","action_required":N,"worth_considering":N,"dismissed":N}` を入れる。triage report の冒頭に `Driver: codex / Reviewer: claude` 行を追加 (assertion 用フィールド)。
7. **`ralph-orchestrator.sh` の橋渡し**: `RALPH_LOOP_DRIVER` / `RALPH_CODEX_SANDBOX` / `RALPH_CODEX_APPROVAL_POLICY` / `RALPH_CLAUDE_REVIEWER_MODEL` を子プロセス (`ralph-pipeline.sh`) にエクスポート。ログ表示で現在の driver を出力。
8. **doctor 表示**: `internal/cli/doctor.go` に "Loop driver: <effective> (source: env|toml|default)" 行と Codex sandbox/approval の有効値を追加。`internal/cli/cli_test.go` にテーブル駆動テスト追加 (env のみ / TOML のみ / 両方 / どちらもなし)。
9. **skill / docs 同期**: `.claude/skills/loop/SKILL.md` の "CLI 別実行ガイダンス" セクションに Codex driver 切替コマンドを追記、`.agents/skills/loop/SKILL.md` を同内容に同期 (drift check 強制)。`.claude/skills/cross-review/SKILL.md` & `.agents/skills/cross-review/SKILL.md` に「Loop 内では driver=codex のとき reviewer=claude に反転する」旨を 2〜3 行で追記。`docs/recipes/ralph-loop.md` に "Codex driver で実行する" 節を追加。`AGENTS.md` / `README.md` / `docs/quality/definition-of-done.md` を 1〜2 行で更新。
10. **テスト**: `tests/test-ralph-cli-driver.sh` 新設 — `tests/fixtures/fake-codex` (シェルスタブ: argv/stdin を `.harness/fixtures/fake-codex.last-call.json` に書き出し、`-o "$file"` 引数を見て fake last-message を書き、終了コードはテストパラメータで制御) と `tests/fixtures/fake-claude` 同様。`PATH="$PWD/tests/fixtures:$PATH"` の状態で `run_agent` を呼び、組み立て argv が想定通りか、`<log>.json` が書かれるか、sidecar 不在時のフォールバックが動くかを assertion。`run-verify.sh` から呼び出し。`internal/scaffold/embed_test.go` は `templates/base/ralph.toml` に `[loop]` が embed されていることを assertion 追加。
11. **手動 walkthrough**: 既存 `/loop` フローを Claude driver で 1 度回し、回帰がないことを確認。Codex CLI が利用可能ならば Codex driver でも short prompt で 1 度回し、`docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` を残す。Codex CLI 不在ならば fake-codex 統合テスト + dry-run のみで evidence を記録 (AC-11 の代替経路)。

## Verify plan

- Static analysis checks:
  - `gofmt`/`go vet`/`go build ./...`
  - `shellcheck scripts/ralph-cli-driver.sh scripts/ralph-pipeline.sh scripts/ralph-orchestrator.sh scripts/ralph-config.sh`
  - `./scripts/check-skill-sync.sh`
  - `./scripts/check-sync.sh` (root `.codex/` ↔ `templates/base/.codex/`)
- Spec compliance criteria to confirm:
  - `[loop] driver` の三層 priority (CLI 引数なし / env / TOML / 既定) が `RALPH_PRIMARY_CLI` と同様に動作する
  - Codex driver で sidecar 検出ロジック (`.agent-signal`, `.self-review-result`, `.verify-result`, `.test-result`, `.pr-url`) が無改修で動く
  - 後方互換: driver 未指定時は既存 Claude Code 挙動が一切変わらない
- Documentation drift to check:
  - `.claude/skills/loop/SKILL.md` と `.agents/skills/loop/SKILL.md` の本文一致
  - `AGENTS.md` の Primary loop / Codex setup checklist が Loop に言及していること
  - `docs/quality/definition-of-done.md` に Codex driver 完走条件が反映されていること
  - `README.md` の Quick start に Codex driver 1 行例 (任意)
- Evidence to capture:
  - `docs/reports/verify-2026-05-08-ralph-loop-codex-driver.md`
  - `./scripts/run-verify.sh` の実行ログ
  - `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight --dry-run` の出力

## Test plan

- Unit tests:
  - `tests/test-ralph-cli-driver.sh` (新設、fake-claude / fake-codex 経由): `run_agent` がそれぞれの driver で期待 argv (フラグ順序込み) を組み立て、`<log>` テキスト + `<log>.json` を書き出し、stdin 経由でプロンプトが渡されているか assertion。`.last` 不在時の空ログ + 警告経路もテスト。
  - `internal/config/config_test.go`: `[loop] driver = "codex"` パース + 不正値 (`"foo"`) エラー、`codex_sandbox`/`codex_approval_policy` の許容値テーブル + 不正値エラー。
  - `internal/cli/cli_test.go` (env→TOML priority): TOML のみ / env のみ / 両方 / どちらもなし の 4 ケースで `runPipeline` が組み立てる env を assertion (実行は exec しないモック化)。
  - `internal/scaffold/embed_test.go`: `templates/base/ralph.toml` に `[loop]` セクションが embed されている。
- Integration tests:
  - `RALPH_LOOP_DRIVER=claude ./scripts/ralph-pipeline.sh --preflight` (既存挙動維持)
  - fake-codex を `PATH` に置いた状態で `RALPH_LOOP_DRIVER=codex ./scripts/ralph-pipeline.sh --preflight` が green。Codex 専用プローブが走り、必要フラグが全て検出される。
  - fake-codex + `RALPH_LOOP_DRIVER=codex` で `ralph-pipeline.sh --dry-run` 完走時に `<log>.json` が `{"result":"...","session_id":null}` で書かれている。
  - **cross-review reviewer 反転テスト**: fake-codex / fake-claude 両用意の状態で `driver=claude` → fake-codex が `exec review` 呼び出しで起動し、`driver=codex` → fake-claude が adversarial プロンプトで起動。triage report に `Driver:` / `Reviewer:` フィールドが正しく入る。
- Regression tests:
  - 既存 `tests/test-check-skill-sync.sh` / `tests/test-check-mojibake.sh` / `tests/test-check-sync.sh` が green
  - 既存 `internal/cli/cli_test.go` の `TestDoctor*` 群が拡張後も green
  - `./scripts/run-verify.sh` が green
- Edge cases:
  - `RALPH_LOOP_DRIVER` が `"foo"` のとき: bash 側で early exit + エラーメッセージ。
  - `ralph.toml [loop] driver = "codex"` だが Codex CLI 不在: preflight が明示的 fail (config_error 終端)。
  - env と TOML が両方設定されている: env が勝つ (priority 仕様、AC-4 で明示テスト)。
  - Codex driver で `--output-last-message` ファイルが生成されなかった (Codex 異常終了): `.last` 不在を検知して空ログ + 警告。
  - cross-review で reviewer CLI 不在: 既存 `cross-review` skill の挙動 (silently skip) と同じく Loop 側も skip。
- Evidence to capture:
  - `docs/reports/test-2026-05-08-ralph-loop-codex-driver.md`
  - 各 dry-run のログを `.harness/state/pipeline/` 配下に残し、レポートから参照
  - `docs/reports/walkthrough-2026-05-08-ralph-loop-codex-driver.md` (Codex CLI 利用可能時のみ)

## Risks and mitigations

| リスク | 深刻度 | 緩和策 |
|--------|--------|--------|
| Codex `-s/--sandbox` + `-c approval_policy` の組み合わせが並列 worktree で干渉する | 中 | 各 worktree が独立 working root のため理論上は安全。`-C "$PWD"` で working root を明示。問題が再現したら `RALPH_CODEX_SANDBOX=read-only` で原因切り分け。 |
| Codex CLI 0.128.0 のフラグ仕様が将来変わり driver スクリプトが破綻する | 中 | preflight Probe 5 で `codex exec --help` をパースし `--output-last-message`/`-s`/`-c` の存在を毎回確認。バージョン変動時は preflight 段階で fail させて black-box 失敗を回避。 |
| Codex の最終メッセージに sidecar 書き込み命令が反映されない | 中 | `pipeline-*.md` プロンプトの末尾に "必ず sidecar ファイルに JSON を書け" の指示が既にあるが、Codex 用に再確認。fake-codex 統合テストで sidecar 不在を検知するアサーションを追加。 |
| Codex driver で session 復元できないことが Inner Loop の文脈喪失を招く | 低 | 既存プロンプトはチェックポイント context (failure_triage 等) を毎サイクル inline で append しているため、stateless 起動でも情報は欠落しない。 |
| `ralph-pipeline.sh` の `run_claude` リネームが他スクリプトに波及 | 低 | grep で `run_claude` 参照を全件確認し、必要箇所をリネーム。`run_claude` を後方互換 alias として残さない (シンプルに保つ)。 |
| TOML 設定だけで `./scripts/ralph` shell wrapper を直接叩いた場合 driver が反映されない | 中 | shell wrapper 利用時は env 経由のみサポートと `docs/recipes/ralph-loop.md` で明記。`ralph run` (Go CLI) を案内し、shell wrapper でも環境変数があれば動くことを保証。 |
| cross-review reviewer 反転で fake-claude を呼び損なう | 中 | dispatcher テストで両方向 (driver=claude / driver=codex) のフラグ assertion を行う。triage report の `Driver:` / `Reviewer:` 行で実行時にも検証可能。 |
| 既存 Claude ユーザが driver 未設定で挙動変化する | 高 | 既定値 `claude` を死守。AC-12 で明示テスト。CHANGELOG 不要 (挙動変化なし)。 |
| skill 双方の同期が崩れて drift check で fail | 中 | 編集時に必ず `.claude/skills/loop/` と `.agents/skills/loop/`、`.claude/skills/cross-review/` と `.agents/skills/cross-review/` を一回の commit で更新する手順を `progress checklist` に明記。 |

## Rollout or rollback notes

- ロールアウト: PR マージで全ユーザに driver 切替機能が露出する。既定値 `claude` のため挙動は不変。Codex 派は `RALPH_LOOP_DRIVER=codex` か `ralph.toml` 設定で opt-in。
- ロールバック: 単一 PR を revert すれば元の Claude 専用 Loop に戻る。`ralph-cli-driver.sh` は新設ファイルなので削除で完了。`run_agent` → `run_claude` リネームを巻き戻すコミットを 1 つ作る。
- 段階導入: マージ後 1 週間、Codex driver 利用は手動オプトインのみ。CI のデフォルト job は driver=claude のまま。任意の job (`if codex available`) で driver=codex の dry-run preflight だけを回す。

## Open questions

- なし (Codex 構造化出力戦略は本プランで確定: `--output-last-message` + sidecar ハイブリッド)。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (feat/44/ralph-loop-codex-driver)
- [x] Implementation started — Slice 1: foundation (config, env defaults, ralph.toml [loop])
- [x] `ralph-cli-driver.sh` 新設 + fake-CLI 統合テスト (38/38 assertions green)
- [x] `ralph-pipeline.sh` の `run_claude` → `run_agent` 移行 + driver-aware preflight
- [x] `ralph-pipeline.sh` の cross-review dispatcher 化 (driver=codex 時は claude -p をレビュアーに)
- [x] `.claude/skills/cross-review/prompts/adversarial-claude.md` 新設 + .agents/skills 側へ同期
- [x] `ralph-orchestrator.sh` の env エクスポート
- [x] `internal/cli/run.go` で TOML→env propagation
- [x] `ralph.toml` テンプレ更新 + Go 側 `Loop` 構造体 + パーステスト
- [x] `ralph doctor` 出力更新 (effective driver + source) + テスト
- [x] `ralph status` driver 行 + AGENTS.md primary-loop ノート (AC-6 follow-up — commit 3351df2)
- [x] `.claude/skills/loop/` と `.agents/skills/loop/` の同期更新
- [x] `.claude/skills/cross-review/` と `.agents/skills/cross-review/` の同期更新
- [x] `docs/recipes/ralph-loop.md` 更新
- [x] README.md / definition-of-done.md の差分反映
- [x] `./scripts/run-verify.sh` green
- [ ] Review artifact created (post-implementation pipeline)
- [ ] Verification artifact created (post-implementation pipeline)
- [ ] Test artifact created (post-implementation pipeline)
- [ ] Walkthrough artifact (Codex 利用可なら実行、不可なら fake-codex 経路を記録)
- [ ] PR created
