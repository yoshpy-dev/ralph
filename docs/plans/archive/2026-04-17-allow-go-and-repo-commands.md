# allow-go-and-repo-commands

- Status: In review (post-implementation pipeline)
- Owner: Claude Code
- Date: 2026-04-17
- Related request: goコマンドなどや自作コマンドなど、このリポジトリのプログラム実行で必要なものをすべてsettings.jsonでallowする
- Related issue: N/A
- Branch: chore/allow-go-and-repo-commands

## Objective

このリポジトリの開発・検証で**日常的に使うプログラム実行コマンド**を `.claude/settings.json` (共有・チェックイン) の `permissions.allow` に集約し、Claude Code が許可ダイアログなしに同じ操作を実行できる共有ベースラインを確立する。`.claude/settings.local.json` は開発者固有の一時的許可として残す。

## Scope

### In scope — `.claude/settings.json` に追加する共有ベースライン

**(A) Go ツールチェーン**（`go.mod` 存在・主要言語 Go のため必須）
- `go build:*`, `go test:*`, `go vet:*`, `go run:*`, `go mod:*`, `go get:*`, `go install:*`, `go tool:*`, `go generate:*`, `go list:*`, `go env:*`, `go fmt:*`, `go version`, `go clean:*`, `go doc:*`

**(B) Go リンタ・フォーマッタ**（`packs/languages/golang/verify.sh` が参照）
- `gofmt:*`, `golangci-lint:*`, `staticcheck:*`, `goimports:*`

**(C) シェル静的解析**（`tests/` で `shellcheck -s sh ...` 形式が頻出）
- `shellcheck:*`

**(D) ローカル成果物 (`ralph` バイナリ)**（`scripts/build-tui.sh` と `cmd/ralph/` のエントリポイント）
- `./ralph:*`, `./bin/ralph:*`, `bin/ralph:*`

**(E) リポジトリ内テストスクリプト**（`./scripts/*` と同様の形式）
- `./tests/*`

**(F) Syntax チェック用シェル呼び出し**（既存 `sh -n:*` と対になる）
- `bash -n:*`

### Non-scope — `.claude/settings.json` には追加しない

**(X) 汎用シェル prefix（Codex [HIGH] 指摘を受け除外）**
- ❌ `sh:*`, `bash:*`, `xargs:*`
- 理由: `pre_bash_guard.sh` の deny リストが限定的（`sudo`/force push/hard reset/`rm -rf` 程度）であり、汎用シェル prefix を allow すると、既存 deny を通らない任意コマンドのバイパス手段になる。syntax チェック用途は `sh -n:*` / `bash -n:*` で代替。`xargs` の具体形は必要に応じて exact 追加（`xargs -I {} ...` などの定型）。

**(Y) 開発者固有の一時的許可**
- `HARNESS_VERIFY_MODE=... ./scripts/run-*-verify.sh`（env var prefix を伴う形式）は、ラッパーである `./scripts/run-static-verify.sh` / `./scripts/run-test.sh` が既に `./scripts/*` で包括済みなので、env var prefix を付けない形を推奨する。local にある env var prefix 付きのエクザクト許可はそのまま残す。
- その他 local にある一回きりの exact コマンド（`git -C /Users/...`, PR 番号付き操作等）は local 管理で十分。
- 本タスクでは local の整理・削除は**行わない**（Non-goals 参照）。

## Non-goals

- `.claude/settings.local.json` の編集・削除・整理（スコープ外。既存エントリはそのまま温存）
- グローバル `~/.claude/settings.json` の変更
- `deny` エントリの追加・変更
- 新しいフック追加・既存フック変更
- CI（GitHub Actions）ワークフローの変更
- 不要 prefix の過剰な将来拡張（必要になった時点で追加）

## Assumptions

- `permissions.allow` のエントリは `Bash(<prefix>:*)` または `Bash(<exact>)` 文字列で、Claude Code が prefix マッチで判定する（既存エントリで実証済み: `.claude/settings.json` line 7 `Bash(./scripts/*)` が全 `./scripts/...` 呼び出しを許可）。
- `./tests/*` は `tests/` 配下がすべて `.sh` 実行可能ファイルで構成されており、将来の追加も同形と仮定（`ls -la tests/` で確認済み）。
- `pre_bash_guard.sh` は `allow` より先に実行され、危険パターン（`sudo`/`git push --force`/`git reset --hard`/`rm -rf`）を deny するため、本変更で安全性は相対的に劣化しない。
- `golangci-lint` / `staticcheck` / `goimports` / `shellcheck` は任意ツール（`verify.sh` で `command -v` チェック済み）、未導入環境でも allow 自体は無害。

## Affected areas

- `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/harness-engineering-scaffolding-template/.claude/settings.json`
  - `permissions.allow` 配列末尾（line 41 `"Bash(claude:*)"` の直後）にエントリ追加
  - `hooks` セクションは無変更

## Implementation outline (Discovery → Change → Verify)

### Phase 1: Discovery（実使用コマンドの inventory 化）

1. 現行 `.claude/settings.json` の `permissions.allow` 集合 A を抽出する。
2. `.claude/settings.local.json` の `permissions.allow` 集合 L を抽出する。
3. `scripts/`, `tests/`, `packs/languages/`, `.claude/skills/`, `.claude/agents/` から実際に呼ばれる外部プログラムの grep 結果 S を作る（Go toolchain / linters / ralph / shellcheck）。
4. 「共有すべき (A ∪ S から導出) かつ 汎用シェル bypass でない」集合を算出し、追加対象リスト T を確定する（本プラン Scope 節の (A)–(F)）。
5. 集合 L のうち T でカバーされるエクザクト/prefix は削除候補だが、本タスクの Non-goals に従い削除は行わない。

### Phase 2: Change

6. `Edit` ツールで `.claude/settings.json` の `allow` 配列末尾に、(A)–(F) の prefix を **カテゴリコメントなし**（JSON なのでコメント不可）で、論理的な順序（Go → Linters → Shellcheck → Ralph → Tests → Syntax shells）で追記する。既存エントリと重複するものは追加しない。

### Phase 3: Verify

7. `jq -e . .claude/settings.json` で JSON 構文妥当性を確認。
8. **Runtime canary**: 以下の代表コマンドを実際に実行し、既存 `.claude/settings.local.json` を一時的に空にした状態でも（※実際には local を空にせず、`settings.json` 単独で許可されるかは「エントリ文字列が `allow` 配列に存在し、prefix が該当形式と一致する」ことで判定する保守的代替とする）、`allow` マッチが成立することを**エントリ文字列確認**で検証する：
   - `go test ./...`（`go test:*` prefix）
   - `go vet ./...`（`go vet:*` prefix）
   - `gofmt -l .`（`gofmt:*` prefix）
   - `staticcheck ./...`（`staticcheck:*` prefix）
   - `shellcheck -s sh scripts/ralph`（`shellcheck:*` prefix）
   - `./tests/test-ralph-config.sh`（`./tests/*` prefix）
   - `bin/ralph version`（`bin/ralph:*` prefix）
   - `bash -n scripts/ralph-pipeline.sh`（`bash -n:*` prefix）
9. `./scripts/run-verify.sh` を実行し、リポジトリ全体の verifier が成功することを確認。
10. 証跡を `docs/evidence/verify-<ts>.log` に保存（`run-verify.sh` が自動生成）。

## Acceptance criteria

### 形式的（JSON 内容）
- [ ] AC1: `.claude/settings.json` の `permissions.allow` に以下の prefix が**すべて**存在する：
  `go build:*`, `go test:*`, `go vet:*`, `go run:*`, `go mod:*`, `go get:*`, `go install:*`, `go tool:*`, `go generate:*`, `go list:*`, `go env:*`, `go fmt:*`, `go version`, `go clean:*`, `go doc:*`
- [ ] AC2: `.claude/settings.json` に `gofmt:*`, `golangci-lint:*`, `staticcheck:*`, `goimports:*`, `shellcheck:*` の prefix が存在する。
- [ ] AC3: `.claude/settings.json` に `./ralph:*`, `./bin/ralph:*`, `bin/ralph:*` の prefix が存在する。
- [ ] AC4: `.claude/settings.json` に `./tests/*`, `bash -n:*` の prefix が存在する。
- [ ] AC5: `.claude/settings.json` に `sh:*`, `bash:*`, `xargs:*` の**汎用シェル prefix が存在しない**こと（Codex 指摘対応）。
- [ ] AC6: 既存 `allow` エントリが削除されていない（diff は純粋な追加のみ）。
- [ ] AC7: `hooks` セクションが無変更。
- [ ] AC8: `jq -e . .claude/settings.json` が exit 0。

### 実行的（runtime canary）
- [ ] AC9: 本 PR マージ後、以下を**現在のセッションで**実行し、いずれも許可ダイアログ表示なしに動くこと（または既に許可されていて差分のないことを確認する）：
  - `go test ./...`
  - `gofmt -l .`
  - `./tests/test-ralph-config.sh`（存在する場合）
  - `shellcheck -s sh scripts/ralph`（`shellcheck` 導入済みの場合）
  - `bash -n scripts/ralph-pipeline.sh`
- [ ] AC10: `./scripts/run-verify.sh` が exit 0 で完了する。

### 品質ゲート
- [ ] AC11: `docs/evidence/verify-<ts>.log` が生成されている。
- [ ] AC12: AGENTS.md / CLAUDE.md / .claude/rules/ に `settings.json` の具体エントリを列挙した箇所がないこと（ドキュメント drift なし）。

## Verify plan

- **Static analysis checks**:
  - `jq -e . .claude/settings.json` で JSON 構文
  - `HARNESS_VERIFY_MODE=static ./scripts/run-verify.sh`（gofmt, go vet, staticcheck, golangci-lint）
- **Spec compliance criteria to confirm**:
  - AC1–AC8（形式）を `jq` で一括チェック可能
  - AC5（汎用シェル不在）を `jq '.permissions.allow | map(select(. == "Bash(sh:*)" or . == "Bash(bash:*)" or . == "Bash(xargs:*)"))'` が空配列を返すことで確認
- **Documentation drift to check**:
  - `grep -r "Bash(go " AGENTS.md CLAUDE.md .claude/rules/` — ヒットなしを確認
  - `grep -r "settings.json" AGENTS.md CLAUDE.md .claude/rules/` — 列挙形式のヒットなし
- **Evidence to capture**:
  - `docs/evidence/verify-<ts>.log`
  - 修正前後の `.claude/settings.json` の diff

## Test plan

- **Unit tests**: 該当なし（設定ファイル）
- **Integration tests**:
  - `./scripts/run-verify.sh` 全体実行（go test ./... 含む）
  - Runtime canary（AC9 リスト）
- **Regression tests**:
  - `./scripts/check-sync.sh` を実行し、テンプレート同期差分がないこと
  - `./scripts/check-template.sh` が成功すること
  - `./scripts/check-pipeline-sync.sh` が成功すること
- **Edge cases**:
  - JSON 末尾カンマなし
  - `allow` 配列に重複文字列なし（`jq '.permissions.allow | length as $l | (unique | length) == $l'` で true）
  - 既存 `Bash(gofmt:*)` 等が local にある場合でも shared と共存可能（重複は許可されるが non-goal で整理せず）
- **Evidence to capture**:
  - `docs/evidence/verify-<ts>.log`

## Risks and mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| JSON 構文破壊で settings 読み込み失敗 | Low | Med | `jq -e` 事前確認、Edit 後 diff 目視 |
| allow 追加で危険コマンドが素通り | Low (汎用 shell を除外済) | Med | 汎用 `sh:*`/`bash:*`/`xargs:*` を追加しない。`pre_bash_guard.sh` が上位 deny を維持 |
| Go サブコマンドの prefix 形式が将来変わる | Low | Low | `go <sub>:*` は `go` のサブコマンド構造が大きく変わらない限り有効 |
| Runtime canary が期待通り動かない（実際には許可不一致） | Med | Med | canary で実証、不一致があれば prefix 形式を修正してから再マージ |
| `.claude/settings.local.json` の冗長化 | Med | Low | Non-goals で明示、後続タスクで整理可能 |
| ドキュメントに列挙がありドリフトする | Low | Low | AC12 で drift チェック |

## Rollout or rollback notes

- **Rollout**: `.claude/settings.json` の単一ファイル diff をコミット → PR → マージ。Claude Code は次回 tool 呼び出し時に新しい settings を読み込む。
- **Rollback**: `git revert <commit>` で元に戻る。allow 追加のみなので、ロールバックしても「以前許可されていたものが不許可になる → 確認ダイアログに戻る」という良性な劣化のみ。
- **Partial failure**: Edit が途中で止まり JSON が破損した場合は `git checkout -- .claude/settings.json` で復旧。作業中はこまめに `jq -e` で検証。

## Open questions

- なし（Codex 指摘は Scope / AC / Non-goals に反映済み）。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created (`docs/reports/self-review-2026-04-17-allow-go-and-repo-commands.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-04-17-allow-go-and-repo-commands.md`)
- [x] Test artifact created (`docs/reports/test-2026-04-17-allow-go-and-repo-commands.md`)
- [x] PR created (https://github.com/yoshpy-dev/harness-engineering-scaffolding-template/pull/24)
