# upgrade-gha-actions-node24

- Status: Draft
- Owner: Claude Code
- Date: 2026-04-22
- Related request: Release ワークフロー実行時に GitHub から「Node.js 20 actions are deprecated」警告が出た。2026-06-02 以降は Node.js 24 強制、2026-09-16 に Node.js 20 は削除予定。
- Related issue: N/A
- Branch: ci/upgrade-gha-actions-node24

## Objective

`.github/workflows/*.yml` で利用中の GitHub Actions を Node.js 24 ランタイム対応版にピン更新し、deprecation 警告を解消する。

## Scope

- `.github/workflows/release.yml` の 3 action を更新
  - `actions/checkout@v4.2.2` → `v6.0.2`
  - `actions/setup-go@v5.5.0` → `v6.4.0`
  - `goreleaser/goreleaser-action@v6` → `v7.1.0`
- `.github/workflows/verify.yml` の `actions/checkout@v4.2.2` → `v6.0.2`
- `.github/workflows/check-template.yml` の `actions/checkout@v4.2.2` → `v6.0.2`
- `templates/base/.github/workflows/verify.yml` の `actions/checkout@v4` → `v6`（配布 template はタグのみの記法を維持）
- メインリポのワークフローは SHA ピン + バージョンコメントの形式を維持

## Non-goals

- ワークフローのロジック変更、ジョブ追加・削除
- GoReleaser 設定 (`.goreleaser.yaml` 等) の書き換え
- 他の action (まだ登場していない action) の先回り更新
- ランタイム Node バージョン固定 (`FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true`) 等の一時的回避策

## Assumptions

- 最新 SHA は API 確認済み（2026-04-22 時点）:
  - checkout v6.0.2: `de0fac2e4500dabe0009e67214ff5f5447ce83dd` (node24)
  - setup-go v6.4.0: `4a3601121dd01d1626a1e23e37211e3254c1c06c` (node24)
  - goreleaser-action v7.1.0: `e24998b8b67b290c2fa8b7c14fcfa7de2c5c9b8c` (node24)
- goreleaser-action v7 は GoReleaser v2 系と互換（`version: "~> v2"` のまま使用可）。
- `actions/checkout@v6`, `actions/setup-go@v6` のメジャー更新は、現在利用中のオプション (`fetch-depth`, `go-version-file`) に破壊的変更なし。

## Affected areas

- `.github/workflows/release.yml`
- `.github/workflows/verify.yml`
- `.github/workflows/check-template.yml`
- `templates/base/.github/workflows/verify.yml`

## Acceptance criteria

- [ ] 4 つのワークフローから Node.js 20 系 action が排除されている（`.github/workflows/*.yml` ×3 + `templates/base/.github/workflows/verify.yml`）
- [ ] ルートの `.github/workflows/*.yml` の action 参照は `<owner>/<repo>@<40桁SHA> # <タグ>` 形式でピンされている
- [ ] `templates/base/.github/workflows/verify.yml` の action 参照は tag-only (`@v6` 等) のまま維持されている（template 側はこれまでの方針を踏襲）
- [ ] YAML パース/構文チェックでエラーが出ない
- [ ] `./scripts/run-verify.sh` および `./scripts/check-sync.sh` が成功する
- [ ] PR 作成後の CI (verify, check-template) が green
- [ ] 次回リリース時に deprecation 警告が消えることを確認する手順が PR description に記載されている
- [ ] goreleaser-action v6→v7 はタグ push 時に初めて実走することを PR description で明示し、フォロー課題（下記 Open questions）を記録している

## Implementation outline

1. 作業ブランチを切る（/work で作成）
2. `release.yml` の 3 行を SHA+タグコメントで置換
3. `verify.yml` / `check-template.yml` / `templates/base/.github/workflows/verify.yml` の checkout 行を置換
4. `./scripts/run-verify.sh` および `./scripts/check-sync.sh` で静的チェックを実行（template 同期ズレを検出）
5. 必要なら `docs/reports/` に verify/test レポートを生成
6. コミットは `ci: bump workflow actions to node24-compatible versions` 等の conventional 形式

## Verify plan

- Static analysis checks:
  - `./scripts/run-verify.sh`
  - YAML パース確認（`python3 -c 'import yaml,sys; [yaml.safe_load(open(p)) for p in sys.argv[1:]]' .github/workflows/*.yml` 等）
  - `grep` で旧 SHA (`11bd71901bbe5b1630ceea73d27597364c9af683`, `d35c59abb061a4a6fb18e82ac0862c26744d6ab5`, `e435ccd777264be153ace6237001ef4d979d3a7a`) が残っていないこと
- Spec compliance criteria to confirm:
  - ルートワークフロー（`.github/workflows/*.yml`）は SHA + タグコメントでピンされている
  - `templates/base/.github/workflows/verify.yml` は tag-only 記法を維持している
  - 対象 4 ファイルすべてが更新されている
- Documentation drift to check:
  - `AGENTS.md` / `CLAUDE.md` / `.claude/rules/` に action バージョンへの直接言及がないこと（ある場合は更新）
  - `docs/` 配下に同じ SHA を書いたドキュメントが無いこと
- Evidence to capture:
  - `docs/reports/verify-2026-04-22-upgrade-gha-actions-node24.md`

## Test plan

- Unit tests: 対象外（ワークフローのみの変更、Go コードは触らない）
- Integration tests: `./scripts/run-verify.sh` 全体パスで代替
- Regression tests: `go test ./...` を念のため実行し、ビルド系に影響が無いことを確認
- Edge cases:
  - goreleaser v7 でタグから起動して期待どおり `release --clean` がパスするか（PR merge 後の実リリースで確認する旨を記録）
  - PR 上で verify / check-template が失敗しないか
- Evidence to capture:
  - `docs/reports/test-2026-04-22-upgrade-gha-actions-node24.md`

## Risks and mitigations

- **goreleaser v6 → v7 のメジャー更新でリリース挙動が変わる**
  → v7 リリースノートを確認、`version: "~> v2"` 指定と既存 `.goreleaser.yaml` の互換を再確認。必要なら PR 説明で「次のタグ push で実挙動確認」と明示。
- **checkout v6 / setup-go v6 がランナー環境に暗黙の前提を追加している可能性**
  → 利用オプションは `fetch-depth` と `go-version-file` のみ。リリースノートで破壊的変更が無いことを確認し、PR 前に CI を待つ。
- **SHA を打ち間違え Node.js 20 のまま残る**
  → `grep` で旧 SHA が消えたことを確認するステップを verify plan に含める。

## Rollout or rollback notes

- Rollout: PR をマージすれば以降の push/tag で新 action が使われる。
- Rollback (ワークフローコードのみ失敗した場合): 当該コミットを revert すれば旧 SHA に戻せる。
- Rollback (タグ push 後に goreleaser が途中失敗した場合):
  1. 作成された GitHub Release を `gh release delete <tag> --yes --cleanup-tag` で削除（artifact 含む）。
  2. 必要なら `git push origin :refs/tags/<tag>` でタグを消し、コード修正後に同じタグを打ち直す。
  3. Homebrew tap (`HOMEBREW_TAP_GITHUB_TOKEN` で push される先) 側で Formula の不整合があれば該当コミットを revert / 手動修正。
  4. 最後にワークフロー側のコミットを revert して旧 SHA に戻し、再タグ前に再検証する。
  - `release.yml` は `contents: write` 権限で Release 発行 + Homebrew tap 更新を行うため、revert だけでは外部状態は戻らない前提で扱う。

## Open questions

- goreleaser v7 のリリースノートに既存設定への非互換変更が無いか、PR 作成時に最終確認する。
- **Follow-up (別 PR 想定)**: `release.yml` のドライラン経路（`workflow_dispatch` or `goreleaser release --snapshot --clean`）を PR CI に追加して、マージ前に新 action スタックを実走検証する。今回の PR ではスコープ外。Codex plan advisory (2026-04-22) の Finding 1 を参照。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (`ci/upgrade-gha-actions-node24`)
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
