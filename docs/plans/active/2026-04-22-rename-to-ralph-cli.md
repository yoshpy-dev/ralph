# Rename repo & rebrand to `ralph` CLI

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-22
- Related request: リポジトリ名を `harness-engineering-scaffolding-template` から `ralph` に改名し、「harness engineering scaffolding template」から「ralph — harness engineering CLI」へブランディングを刷新する。
- Related issue: N/A
- Branch: refactor/rename-to-ralph-cli

## Objective

リポジトリの実態（`cmd/ralph/` を中心とする Go CLI）にブランディングを合わせる。GitHub リポジトリ名を `yoshpy-dev/ralph` に改名し、Go モジュールパス・README・AGENTS.md・CLAUDE.md・配布スクリプト・ドキュメントを一貫して `ralph` を主役とする CLI ツールとして書き直す。Homebrew formula 名は既に `ralph` なのでユーザ向け導線（`brew install yoshpy-dev/tap/ralph`）は非破壊。

## Scope

- GitHub リポジトリ名の改名（`gh repo rename ralph`、`git remote set-url` 更新含む）
- `go.mod` のモジュールパス `github.com/yoshpy-dev/harness-engineering-scaffolding-template` → `github.com/yoshpy-dev/ralph`
- 全 Go ソース（`cmd/`, `internal/`, `internal/**/_test.go`）のインポートパス一括置換
- `scripts/install.sh` の `REPO` 変数を `yoshpy-dev/ralph` に更新
- `.goreleaser.yml` の `homepage` URL 更新
- `README.md` の全面リライト（CLI ツールとしての位置付け、インストール手順、使い方中心）
- `AGENTS.md` の冒頭説明と 1 箇所の old-name 参照を更新
- `CLAUDE.md` の記述を `ralph` CLI 前提に調整
- `.github/workflows/*.yml` の中で repo 名がハードコードされていないか確認（基本は `${{ github.repository }}` 参照のはず）
- 本プランファイル / 新規 report など active なドキュメントの記述を新名義で統一

## Non-goals

- アーカイブ済みプラン (`docs/plans/archive/`) / 過去レポート (`docs/reports/`) / 過去スペック (`docs/specs/`) の履歴書き換え → **やらない**（履歴は履歴として保持）。`docs/specs/2026-04-16-ralph-cli-tool.md` も歴史的文脈を保つため手を加えない。
- 本プランファイル (`docs/plans/active/2026-04-22-rename-to-ralph-cli.md`) 内の旧名言及の置換 → 記録としての整合性を優先し書き換えない（完了時に `docs/plans/archive/` へ移動される）。
- `templates/base/` 配下の書き換え → 既に repo 名を参照していないため不要
- Homebrew tap リポジトリ（`yoshpy-dev/homebrew-tap`）側の変更 → 不要（formula 名は既に `ralph`）
- CLI バイナリ名・コマンド体系・フラグの変更 → 対象外（既に `ralph` で統一済み）
- メモリファイル（`~/.claude/projects/...-harness-engineering-scaffolding-template/memory/`）の場所変更 → ユーザ作業扱い、本 PR の対象外
- パブリッシュ済み Go module の互換シム提供 → 不要（外部インポートが存在しないことをエビデンスで確認、下記 Assumptions 参照）

## Assumptions

- GitHub は旧リポジトリ URL を新 URL へ自動リダイレクトするため、外部ドキュメントや古い clone URL は即死しない（ただし内部参照の install.sh は更新必須）。
- Homebrew formula の `url` テンプレートは goreleaser が tag push 時に自動更新するため、次回リリースで新 repo URL に切り替わる。
- EMU 制約下でも `gh repo rename` はリポジトリ owner 権限で実行可能（PR 編集系とは別カテゴリ）。失敗した場合は org admin にリクエストする手動フォールバックあり。
- Go モジュールは外部ライブラリとして利用されていない。エビデンス: (1) `gh search code "yoshpy-dev/harness-engineering-scaffolding-template"` が 0 件（2026-04-22 時点）、(2) `pkg.go.dev/github.com/yoshpy-dev/harness-engineering-scaffolding-template` が HTTP 404 でモジュール未インデックス。したがってモジュールパス変更の後方互換シムは不要。

## Affected areas

- Go モジュール: `go.mod`, `go.sum`（影響なし、sum は path 依存しない）、`cmd/ralph/main.go`, `cmd/ralph-tui/main.go`, `internal/cli/*.go`, `internal/state/*.go`, `internal/ui/**/*.go`, `internal/upgrade/*.go`
- テストデータ: `internal/state/testdata/*.json` — 実パスではなくラベルとして旧名を含む可能性 → 置換要検討
- 配布: `scripts/install.sh`, `.goreleaser.yml`
- ドキュメント (active): `README.md`, `AGENTS.md`, `CLAUDE.md`
- プランファイル: 本ファイル自身（完了時に archive へ移動）
- GitHub 設定: リポジトリ名、description、topics（手動）、ローカル `git remote`

## Design decisions

- **リポジトリ名**: `ralph`（確定。`ralphctl` / `harness-ralph` は却下）。
  - Rationale: 既に CLI バイナリ名・`Ralph Loop` フロー名・`ralph-orchestrator.sh` など repo 内で一貫。grep-ability が高く、Homebrew formula 名とも整合。Allegro の DCIM 製品「Ralph」との SEO 衝突はユーザ判断で許容。
- **README 刷新の深さ**: full rewrite（既存の "scaffold" 前提の前口上を全面入れ替え）。
  - Rationale: 部分置換だと「テンプレート」思想の残滓が残り、CLI ブランディングが中途半端になる。
- **アーカイブ済みドキュメントの扱い**: 書き換えない。
  - Rationale: 過去の plan / report は当時の文脈を保つのが原則（`.claude/rules/documentation.md` の archive 原則に沿う）。
- **モジュールパス移行の手段**: `go mod edit -module` + `find ... -exec sed` による一括置換 + `go build ./...` で整合性検証。
  - Rationale: 外部インポーターがいないのでシムは不要、sed + build が最小コスト。
- **GitHub 改名の手段**: `gh repo rename ralph`（owner 権限前提）+ `git remote set-url origin git@github.com:yoshpy-dev/ralph.git`。
  - Rationale: GitHub が旧 URL を自動リダイレクトするため、外部リンクの即死を回避できる。

Critical forks: なし（主要な判断は上記 5 点ですべて既定解あり、ユーザ確認済み or rules で settled）

## Acceptance criteria

- [ ] `go.mod` の module 行が `github.com/yoshpy-dev/ralph`
- [ ] `go build ./...` と `go test ./...` が成功
- [ ] **user-facing / runtime 表面**に旧名が残らない。具体的には以下のパスセットに対し `rg "harness-engineering-scaffolding-template"` がゼロヒット:
  - `cmd/`, `internal/`, `templates/`, `packs/`
  - `scripts/`, `.github/`, `.goreleaser.yml`, `go.mod`, `go.sum`
  - `README.md`, `AGENTS.md`, `CLAUDE.md`
  - `.claude/skills/`, `.claude/rules/`, `.claude/agents/`, `.claude/hooks/`
  - 除外（残存許容）: `docs/plans/archive/`, `docs/reports/`, `docs/specs/`, 本プランファイル (`docs/plans/active/2026-04-22-rename-to-ralph-cli.md`)
- [ ] `scripts/install.sh` が新 repo URL から release asset を取得できる（URL 組み立てを検証）
- [ ] `.goreleaser.yml` の `homepage` が新 URL を指す
- [ ] README.md が「`ralph` CLI」を主題とし、"scaffolding template" のフレーミングを使っていない
- [ ] AGENTS.md 冒頭 1 行目が CLI ツールとしての性質を示す説明になっている
- [ ] `./scripts/run-verify.sh` が成功
- [ ] **前提ゲート**: `gh repo rename` がマージ前に成功し、`yoshpy-dev/ralph` が到達可能、旧 URL が 301 で新 URL へリダイレクト（未達なら /pr を実行しない）
- [ ] `git remote -v` が新 URL を指している
- [ ] 外部インポーター検証エビデンス（`gh search code` 0 件、`pkg.go.dev` 404）を verify レポートに添付

## Implementation outline

rename-first 方式（Codex HIGH#2 への対応）。

1. **ブランチ作成**（/work が実施）。
2. **GitHub リポジトリ改名を先行実行**
   - `gh repo rename ralph --repo yoshpy-dev/harness-engineering-scaffolding-template`
   - 失敗時（EMU 拒否など）はここで停止。以降のコード変更は rename 完了まで実施しない。
   - 成功時: `git remote set-url origin git@github.com:yoshpy-dev/ralph.git` とし、`gh repo view yoshpy-dev/ralph` と旧 URL の 301 リダイレクトを確認。
3. **Go モジュールパス置換**
   - `go mod edit -module github.com/yoshpy-dev/ralph`
   - 全 `*.go` で `github.com/yoshpy-dev/harness-engineering-scaffolding-template` → `github.com/yoshpy-dev/ralph` 置換
   - `go build ./... && go test ./...` で検証
4. **配布スクリプト更新**
   - `scripts/install.sh`: `REPO="yoshpy-dev/ralph"`、ヘッダコメントの curl URL も更新
   - `.goreleaser.yml`: `homepage` URL 更新
5. **README.md 全面リライト**
   - タイトルを `# ralph` に変更
   - リード文を「`ralph` is a CLI for harness engineering ...」形式へ
   - インストールセクションを筆頭に据える
   - リポジトリ構造セクションは残すが「CLI が配る中身」という立ち位置で再構成
6. **AGENTS.md / CLAUDE.md 調整**
   - AGENTS.md 冒頭の `This repository is a scaffold for harness engineering.` を `This repository hosts ralph, a CLI for harness engineering.` 等へ
   - `harness-engineering-scaffolding-template` 参照があれば除去
   - CLAUDE.md は ralph CLI 前提で軽く表現調整
7. **GitHub メタ情報更新**
   - description / topics を CLI 向けに更新（`gh repo edit yoshpy-dev/ralph --description "..." --add-topic cli --add-topic claude-code` 等）
8. **検証**
   - `./scripts/run-verify.sh`
   - acceptance criteria で定義したパスセットに対して `rg "harness-engineering-scaffolding-template"` がゼロヒットであることを確認
   - `git remote -v` 確認
9. **post-implementation pipeline**: /self-review → /verify → /test → /sync-docs → /codex-review → /pr（`post-implementation-pipeline.md` 準拠）

## Verify plan

- Static analysis checks:
  - `go vet ./...`
  - `go build ./...`
  - `gofmt -l .` が空
  - `./scripts/run-verify.sh`
- Spec compliance criteria to confirm:
  - 全 acceptance criteria が検査可能であり満たされる
  - モジュールパス変更により外部 import が壊れない（外部インポーターは存在しない前提）
- Documentation drift to check:
  - README / AGENTS.md / CLAUDE.md の相互参照整合
  - `.claude/skills/release/SKILL.md` に旧 repo 名がないこと（確認済み: 無し）
  - ユーザ向け `brew install yoshpy-dev/tap/ralph` 導線が壊れていないこと
- Evidence to capture:
  - `rg "harness-engineering-scaffolding-template"` の出力（残存 = アーカイブのみ）
  - `go build ./...` / `go test ./...` の成功ログ
  - `git remote -v` の出力

## Test plan

- Unit tests:
  - 既存の `go test ./...` が全てパスすること（パス名変更に伴う testdata 参照の破損がないこと）
- Integration tests:
  - `ralph init` でスキャフォールド可能なこと（バイナリ動作確認）
  - `ralph doctor` が通ること
- Regression tests:
  - `scripts/install.sh` の URL 組み立てを dry-run（`set -x` で URL を echo）して新 repo を指すこと
  - `./scripts/run-verify.sh` のフル通過
- Edge cases:
  - アーカイブ済みドキュメントは意図的に旧名を残す（履歴保全）— 自動 lint が false positive を出さないか確認
  - `git clone` で旧 URL を叩いた場合、GitHub の自動リダイレクトで新 URL に到達できること（手動確認）
- Evidence to capture:
  - `go test ./...` 出力
  - `./scripts/run-verify.sh` 出力
  - 手動確認: 旧 URL → 新 URL リダイレクト

## Risks and mitigations

- **R1: EMU 制約で `gh repo rename` が拒否される**
  - 影響: リポジトリ改名が Claude Code 側から完遂できない
  - 軽減: rename-first 方式により、rename が失敗した時点で実装を停止し、コード変更を main に混入させない。org admin に手動リクエストして完了まで hold。
- **R2: GitHub リダイレクト遅延**
  - 影響: 改名直後に `go get` や `git clone` で一時的に失敗する可能性
  - 軽減: GitHub は通常即時リダイレクト。README に「改名しました」注記を数週間残す（任意）。
- **R3: Homebrew formula の goreleaser 再生成が次リリースまで走らない**
  - 影響: `ralph.rb` の `url` フィールドが旧 repo を指したまま → tarball 取得が壊れる？
  - 軽減: GitHub の自動リダイレクトで tarball URL も 302 される想定。新リリースを切れば goreleaser が URL を更新するため、rename 直後に patch release を打つのが安全。
- **R4: testdata 内の旧 repo 名リテラル**
  - 影響: ユニットテストが旧名を期待して比較している場合 rename 後に fail
  - 軽減: 実装中に `grep` で検出、置換対象に含める（Affected areas に列挙済み）。
- **R5: ローカル記憶ディレクトリ (`~/.claude/projects/...`) のパス変更**
  - 影響: 次回以降のセッションで memory が新パスから読まれ、履歴が分離
  - 軽減: ユーザ作業として別途ディレクトリ rename を案内（PR の notes に記載）。

## Rollout or rollback notes

Codex の指摘を受け、**repo rename を merge の前に実行する**ように順序を変更。これにより「`main` が存在しない URL を指す」状態を回避する。

- **Rollout（rename-first 方式）**:
  1. feat ブランチを作成
  2. **先に GitHub リポジトリ改名を実行**: `gh repo rename ralph --repo yoshpy-dev/harness-engineering-scaffolding-template`
     - 成功確認: `gh repo view yoshpy-dev/ralph` が 200、旧 URL が 301 で新 URL へリダイレクト
     - 失敗した場合: コード変更は commit せず停止、org admin に rename 依頼。依頼完了まで本プランは hold。
  3. ローカル `git remote set-url origin git@github.com:yoshpy-dev/ralph.git`
  4. コード変更を commit（go.mod / imports / install.sh / .goreleaser.yml / README / AGENTS.md / CLAUDE.md）
  5. `./scripts/run-verify.sh` フル通過
  6. `/pr` で PR 作成・マージ
  7. 次回の `/release` で patch release を打ち、Homebrew formula の URL を新 repo に追従させる
- **Rollback**:
  - repo 名変更後、コード未 merge: `gh repo rename harness-engineering-scaffolding-template` で再度改名可能（GitHub は旧名も 90 日間予約）。
  - コード merge 後: 通常の `git revert` で code はロールバック可能。repo 名は別途 rename し直す必要あり。両方を同一 revert PR に含める運用。

## Open questions

- リポジトリ description / topics: 確定（description = "ralph — a CLI for harness engineering with Claude Code"、topics = `cli`, `claude-code`, `go`, `agent-harness`, `harness-engineering`）。`template` / `harnessengineering` トピックは削除済み。
- README のキービジュアル（ヒーロー）を用意するか（対象外・保留）

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] GitHub repo renamed & remote updated
- [x] Go module path migrated, tests passing
- [x] install.sh + .goreleaser.yml updated
- [x] README rewritten
- [x] AGENTS.md + CLAUDE.md reframed
- [x] GitHub description + topics updated
- [x] Scoped grep clean (only archive/specs/this plan retain old name)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
