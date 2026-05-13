# add-terraform-language-pack

- Status: Implementation complete (awaiting pipeline)
- Owner: Claude Code
- Date: 2026-05-13
- Related request: Add Terraform / OpenTofu 向け language pack so ralph users can verify IaC repos.
- Related issue: #52
- Branch: feat/52/add-terraform-language-pack

## Objective

`ralph` に Terraform / OpenTofu プロジェクト向けの language pack を追加し、`ralph pack add terraform` および `./scripts/run-verify.sh` が IaC リポジトリで動作するようにする。

## Scope

- `packs/languages/terraform/` を新設（`README.md` + `verify.sh`）。
- `templates/packs/terraform/` を `packs/languages/terraform/` と byte 一致でミラー（go:embed 経由で `PackFS("terraform")` を解決させる）。
- `.claude/rules/terraform.md` に HCL / モジュール構成 / state 管理の最小規約を追加。
- `templates/base/.claude/rules/terraform.md` を `.claude/rules/terraform.md` と byte 一致でミラー（scaffolded プロジェクトにも配布、`scripts/check-sync.sh` 通過のため）。
- `scripts/detect-languages.sh` に Terraform 判定（`*.tf` / `*.tofu` / `.terraform.lock.hcl`）を追加。
- `docs/recipes/adding-a-language-pack.md` の例として `terraform` を参照可能な水準で追記。

## Non-goals

- Terragrunt 専用の検証フロー（別パック / 別 issue）。
- Sentinel / OPA など provider 固有の policy-as-code。
- `ralph init` のインタラクティブ提案 UI への組み込み（別 issue）。
- `ralph pack add <lang>` 自体の既存バグ修正。`internal/cli/pack.go:64-67` の `addPack` は `RenderFS` に `TargetDir: absDir`（プロジェクトルート）を渡しており、`internal/cli/init.go:158` が使う `packs/languages/<lang>/` サブパスを使っていない。これは事前から存在する一般バグで、Terraform pack 追加とは独立。本 PR の受け入れ条件は「`PackFS("terraform")` が解決される」に留め、`pack add` 経由の配置先修正は別 issue で扱う（Risks 参照）。

## Assumptions

- 既存パック（`packs/languages/golang/verify.sh:3-6`）の流儀に従い、ツール未インストール時は警告メッセージを出して `exit 0`（パック非該当扱い）。
- `templates/packs/<lang>/` は `packs/languages/<lang>/` と byte 一致が `scripts/check-sync.sh` で強制される。
- `PackFS("terraform")` は `internal/scaffold/embed.go` の自動探索により、`templates/packs/terraform/` を置けばコード変更なしで解決される。
- `HARNESS_VERIFY_MODE` は `static` / `test` / `all` の 3 値（`scripts/run-verify.sh:8` および `packs/languages/_template/verify.sh:4-6` 参照）。
- ルールファイル名は既存の慣習（`golang.md` / `python.md` / `rust.md` / `dart.md` / `typescript.md`）に合わせて `terraform.md`（dot 接頭辞なし）。

## Affected areas

- `packs/languages/terraform/README.md` 新設
- `packs/languages/terraform/verify.sh` 新設
- `templates/packs/terraform/README.md` 新設（packs/ ミラー）
- `templates/packs/terraform/verify.sh` 新設（packs/ ミラー）
- `.claude/rules/terraform.md` 新設
- `templates/base/.claude/rules/terraform.md` 新設（base ミラー、scaffolded プロジェクトに配布）
- `scripts/detect-languages.sh` の Terraform 判定追加
- `docs/recipes/adding-a-language-pack.md` の例を `terraform` に更新

## Design decisions

Critical forks resolved with the user: **None**（仕様が issue で十分に固定済み）。

採用したデフォルト（議論不要 + Codex 助言を反映済み）:

1. **IaC CLI は `tofu` を優先し、無ければ `terraform` にフォールバック**
   - 理由: issue は `terraform` CLI 前提だが、`.tofu` 検出を含む以上 OpenTofu 利用者の verify が無意味にならないようにすべき（Codex MEDIUM#3）。`IAC_CLI="$(command -v tofu || command -v terraform || echo "")"` で動的選択。サブコマンド（`fmt -check -recursive`, `validate`, `test`）は両 CLI で互換。
2. **マーカー検出と CLI 検出を分離（fail-open を回避）**
   - 理由: issue の文言は曖昧だが、既存パック（`packs/languages/golang/verify.sh:3-12`, `python/verify.sh:3-12`, `rust/verify.sh:3-12`）の慣習は「マーカー無し → exit 0（pack 不発火）、マーカー有り + CLI 無し → exit 1（環境不備）」。Codex HIGH#2 を採用。具体的には:
     - `.tf` / `.tofu` / `.terraform.lock.hcl` が見つからない → メッセージ + `exit 0`
     - 上記のいずれかが存在するが `terraform`/`tofu` のどちらも未インストール → エラー + `exit 1`
3. **ルールファイル名は `terraform.md`（dot 接頭辞なし）+ template/base ミラー**
   - 理由: 既存 5 言語すべて dot 無しかつ `templates/base/.claude/rules/` にミラー存在（`golang.md` 等）。`scripts/check-sync.sh:178` の SCAN_DIRS が `.claude` を走査するため、ミラーが無いと ROOT_ONLY エラー（Codex MEDIUM#4）。
4. **`terraform validate` は `.terraform/` ディレクトリ未存在ならスキップ + 警告**
   - 理由: init 済み前提のコマンドで、未 init での失敗は誤検知になる。issue 仕様通り。
5. **`tflint` / `tfsec` / `trivy config` は `command -v` で optional 実行**
   - 理由: Go pack（`packs/languages/golang/verify.sh:30-41`）の `golangci-lint` / `staticcheck` と同じ作法。core IaC CLI（terraform/tofu）と optional linter は扱いが違う（前者は必須、後者は欠落許容）。

### Codex 助言の取り扱い

- **HIGH#1（`pack add` の配置先バグ）**: 受け入れ。ただし**スコープ外**として処理（Non-goals に明記）。受け入れ条件は「`PackFS("terraform")` が解決される」までで止め、`ralph pack add terraform` の end-to-end 動作確認は本 PR の必須項目から外す（既存バグ依存）。フォローアップ issue を `/pr` 時に提案。
- **HIGH#2（fail-open）**: 受け入れ。設計決定 #2 で反映。
- **MEDIUM#3（OpenTofu 名目サポート）**: 部分採用。`tofu` CLI フォールバック追加（設計決定 #1）。`.tofu` 固有ファイル fixture の自動テストは入れず、コードレベルでの dispatch のみ。
- **MEDIUM#4（template/base ミラー欠落）**: 受け入れ。Affected areas / Scope / Acceptance criteria に追加。

## Acceptance criteria

- [x] `packs/languages/terraform/README.md` と `packs/languages/terraform/verify.sh` が存在し、`verify.sh` は実行可能（`chmod +x`）。
- [x] `templates/packs/terraform/` が `packs/languages/terraform/` と byte 一致（`scripts/check-sync.sh` が PASS）。
- [x] `.claude/rules/terraform.md` が存在し、`paths:` frontmatter で `**/*.tf` / `**/*.tofu` / `**/*.tftest.hcl` / `**/.terraform.lock.hcl` をスコープする。
- [x] `templates/base/.claude/rules/terraform.md` が `.claude/rules/terraform.md` と byte 一致で存在（`scripts/check-sync.sh` の ROOT_ONLY を回避）。
- [x] `scripts/detect-languages.sh` が `*.tf` / `*.tofu` / `.terraform.lock.hcl` のいずれかが存在するリポジトリで `terraform` を emit する（`.terraform/` ディレクトリは prune）。
- [x] `internal/scaffold.PackFS("terraform")` が `templates/packs/terraform/` を解決し、`ralph pack list` の出力に `terraform` が含まれる。（`ralph pack add terraform` 経由の配置先確認は本 PR の必須項目に含めない — 既存 `addPack` バグのため。Risks 参照）
- [x] `HARNESS_VERIFY_MODE=static` で IaC CLI（`tofu` 優先、無ければ `terraform`）の `fmt -check -recursive` と（`.terraform/` 有る時のみ）`validate` を実行し、未 init はスキップ + 警告。
- [x] `HARNESS_VERIFY_MODE=test` で `*.tftest.hcl` がある場合のみ IaC CLI の `test` サブコマンドを実行、無ければ "no terraform tests" を表示して skip。
- [x] `tflint` / `tfsec` / `trivy config` は未インストール時にスキップ表示（CI で fail しない）。
- [x] `.tf` / `.tofu` / `.terraform.lock.hcl` 不在時は警告メッセージ + `exit 0`（pack 不発火）。
- [x] 上記マーカー有りで `terraform` / `tofu` のどちらも未インストール時は明示エラー + `exit 1`（fail-open 回避、Codex HIGH#2）。
- [x] `docs/recipes/adding-a-language-pack.md` で `terraform` を例として全面差し替え、`detect-languages.sh` への手書き追記が必要なことを明記、mirror-checklist セクション追加。
- [x] `scripts/check-skill-sync.sh` がグリーン（今回 skill は触らないが念のため）。

## Implementation outline

1. `./scripts/new-language-pack.sh terraform` でスケルトン作成（`packs/languages/terraform/` に README.md + verify.sh）。
2. `packs/languages/terraform/README.md` を Go pack 風に書き直し（検証順序の説明、カスタマイズポイントを箇条書き、`tofu`/`terraform` CLI 選択の挙動を明記）。
3. `packs/languages/terraform/verify.sh` を実装:
   - shebang `#!/usr/bin/env sh` + `set -eu`
   - `HARNESS_VERIFY_MODE` 解決（default `all`）
   - マーカー検出: `find . -maxdepth 3 -name '*.tf' -o -name '*.tofu' -o -name '.terraform.lock.hcl'` 等で 1 個以上見つからなければ警告 + `exit 0`
   - IaC CLI 選択: `IAC_CLI="$(command -v tofu || command -v terraform || true)"`。空なら エラー + `exit 1`（fail-open 回避）
   - `run_static`: `$IAC_CLI fmt -check -recursive` → `[ -d .terraform ] && $IAC_CLI validate` → `tflint`（あれば）→ `tfsec`（あれば、無ければ `trivy config .`）
   - `run_tests`: `*.tftest.hcl` を検索、あれば `$IAC_CLI test`、無ければ "no terraform tests" + 終了 0
   - `case "$mode"` で static / test / all を分岐、各 sub-step での失敗を `status=1` に集約
4. `templates/packs/terraform/{README.md,verify.sh}` を `packs/languages/terraform/` から `cp` で byte 一致複製（permission も含めて確認）。
5. `.claude/rules/terraform.md` を新設:
   - frontmatter `paths: ["**/*.tf", "**/*.tofu", "**/*.tftest.hcl"]`
   - モジュール分割、`terraform.tfstate` をリポジトリにコミットしない、`variable`/`output` の `description` 必須、`required_providers` 明示、`backend` 設定の明示、など最小ルール。
6. `templates/base/.claude/rules/terraform.md` に同内容を byte 一致でコピー。
7. `scripts/detect-languages.sh` に `terraform` 判定ブロックを追加（`.terraform.lock.hcl` 単体 or `*.tf` / `*.tofu` のいずれか）。
8. `docs/recipes/adding-a-language-pack.md` の例を更新（`new-language-pack.sh terraform` を出して、`.claude/rules/terraform.md` 追加・`templates/base/.claude/rules/terraform.md` ミラー必須・`detect-languages.sh` への手書き追記が必要なことを明記）。
9. `./scripts/check-sync.sh` / `./scripts/check-skill-sync.sh` / `./scripts/run-verify.sh` / `go test ./...` を走らせて緑にする。

## Verify plan

- 静的解析チェック:
  - `./scripts/check-sync.sh` が PASS（`packs/languages/terraform/` ↔ `templates/packs/terraform/` の byte 一致）
  - `./scripts/check-skill-sync.sh` が PASS
  - `shellcheck packs/languages/terraform/verify.sh`（手元にあれば）
  - `gofmt`/`go vet` は本パックでは Go コード変更が無いので影響なし
- 仕様適合チェック:
  - 受け入れ条件チェックリストの全項目
  - `HARNESS_VERIFY_MODE=static`/`test`/`all` 各モードでの分岐実行
  - 未インストール時の `exit 0` 動作
- ドキュメントドリフトチェック:
  - `docs/recipes/adding-a-language-pack.md` が新パックを反映
  - `AGENTS.md` の repo map で `packs/languages/` の説明が古くなっていないか確認（言語名は列挙されていないので変更不要見込み）
  - `templates/packs/_template/` のメンテ要否は無し
- エビデンス:
  - `docs/evidence/verify-<timestamp>.log`
  - `docs/reports/verify-2026-05-13-add-terraform-language-pack.md`

## Test plan

- 単体テスト:
  - `internal/scaffold` の既存テスト（`AvailablePacks` / `PackFS`）が `terraform` を新規 pack として認識することを確認。新規テスト追加は不要（自動探索の汎用テストが既に存在）。
- 統合テスト:
  - 一時ディレクトリで `.tf` ファイルを作り、`./scripts/detect-languages.sh` が `terraform` を emit することを確認。
  - 一時ディレクトリで `packs/languages/terraform/verify.sh` を直接呼び、各シナリオで想定通りに動くことを確認:
    - マーカー無し → 警告 + `exit 0`
    - `.tf` 有り + `terraform`/`tofu` 両方無し → エラー + `exit 1`（fail-open 回避の回帰防止）
    - `.tf` 有り + CLI 有り + `.terraform/` 無し → `validate` をスキップ
    - `static` / `test` / `all` の mode 分岐
  - `ralph pack list` 出力に `terraform` が含まれる（`PackFS` 自動探索の確認）。
- 回帰テスト:
  - `go test ./...` 全体で既存パックの検出・解決が壊れていないこと。
  - `./scripts/check-sync.sh` で他パック（golang/dart/python/rust/typescript）の drift が出ないこと。
- エッジケース:
  - `.tofu` のみ存在する OpenTofu 専用リポジトリでも `terraform` が emit される。
  - `.terraform.lock.hcl` のみ（`.tf` 無し）でも emit される。
  - `*.tftest.hcl` が無い場合、test mode で skip して終了 0。
  - `.terraform/` 未存在で `terraform validate` がスキップされ全体は失敗しない。
- エビデンス:
  - `docs/reports/test-2026-05-13-add-terraform-language-pack.md`

## Risks and mitigations

- **R1: `terraform fmt` / `validate` の挙動が CLI バージョンで微妙に違う** → 主要バージョン（1.6+）を前提と明記、未インストール時は明示エラー（マーカー有り）or 早期 exit 0（マーカー無し）。
- **R2: `tfsec` がアーカイブされ trivy 推奨化済み** → `tfsec` を優先しつつ trivy へのフォールバックを順序で記述、両方 optional に保つ。
- **R3: OpenTofu ユーザーの `.tofu` ファイル運用** → `tofu` CLI を `terraform` より優先する dispatch を入れる（設計決定 #1）。`.tofu` 固有挙動の網羅は別 issue。
- **R4: `templates/packs/_template/verify.sh` は exit 2（TODO）のままなので、新パックをコピーした人が混乱** → Non-goals。`_template` は別途見直し対象だが本 PR では触らない。
- **R5: `check-sync.sh` の byte 一致を壊しやすい（コピーミス）** → 実装 outline ステップ 4・6 で「`cp` で複製」と明記、最後の verify でゲートに依拠。
- **R6: `ralph pack add terraform` を試した利用者がプロジェクトルートに `verify.sh` を書き出してしまう（既存 `internal/cli/pack.go:64-67` バグ）** → 本 PR スコープ外。`/pr` 段階でフォローアップ issue を提案、README/recipe に「現状は `pack add` ではなく `init` 時の packs 指定経由で導入してください」と注意書きを足すかは実装時に判断。

## Rollout or rollback notes

- ロールアウト: パック追加のみで既存挙動への破壊は無し。`ralph upgrade` で既存プロジェクトへの自動配布は走らない（pack はオプトインで `ralph pack add` 経由）。
- ロールバック: 追加ファイルを削除すれば元通り。`detect-languages.sh` の変更も 1 ブロック削除のみ。

## Open questions

- なし（issue 仕様が明確、Codex 助言は HIGH#2・MEDIUM#3・MEDIUM#4 をプランに反映済み、HIGH#1 はスコープ外として別 issue 化予定）。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (`feat/52/add-terraform-language-pack`)
- [x] Implementation started
- [x] Implementation complete (4 slices committed)
- [x] Review artifact created (`docs/reports/self-review-2026-05-13-add-terraform-language-pack.md`)
- [x] Verification artifact created (`docs/reports/verify-2026-05-13-add-terraform-language-pack.md`)
- [x] Test artifact created (`docs/reports/test-2026-05-13-add-terraform-language-pack.md`)
- [x] Sync-docs artifact created (`docs/reports/sync-docs-2026-05-13-add-terraform-language-pack.md`)
- [x] Cycle-2 pipeline complete (Codex ACTION_REQUIRED P2 hermeticity fix `f27e1a2` — self-review MERGE / verify PASS / test PASS 114/114 + 3 hermeticity probes; reports: `*-cycle2.md` siblings)
- [x] Cycle-3 pipeline complete (cap raised 2 → 3 by user direction; gitignore safety net `03c5598` for Codex cycle-2 WORTH_CONSIDERING P2 + behavioral test `68cc41f` (`tests/test-terraform-gitignore.sh`, 47 assertions) — self-review MERGE / verify PASS / test PASS 155/155; reports: `*-cycle3.md` siblings)
- [ ] PR created
