# Verify report: upgrade-gha-actions-node24

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-upgrade-gha-actions-node24.md`
- Verifier: verifier subagent (/verify)
- Scope: Spec compliance + static analysis for branch `ci/upgrade-gha-actions-node24` vs `main`. Behavioral tests and CI run-time verification are out of scope (handled by `/test` and GitHub CI).
- Evidence: `docs/evidence/verify-2026-04-22-upgrade-gha-actions-node24.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 4 つのワークフローから Node.js 20 系 action が排除されている | Verified | 旧 SHA (`11bd71901bbe...`, `d35c59abb061...`, `e435ccd77726...`) の repo-wide grep はプラン本文以外に hit なし。4 ファイル (`release.yml`, `verify.yml`, `check-template.yml`, `templates/base/.github/workflows/verify.yml`) すべての `uses:` 行が node24 対応 SHA / タグに更新済み。 |
| AC2: ルートの `.github/workflows/*.yml` の action 参照は `<owner>/<repo>@<40桁SHA> # <タグ>` 形式でピンされている | Verified | `release.yml:15,19,23` / `verify.yml:8` / `check-template.yml:10` の 5 行すべてが `@<40 hex> # vX.Y.Z` 形式。`grep -rn 'actions/(checkout\|setup-go)@\|goreleaser/goreleaser-action@'` で確認。 |
| AC3: `templates/base/.github/workflows/verify.yml` の action 参照は tag-only (`@v6`) のまま維持されている | Verified | `templates/base/.github/workflows/verify.yml:8` = `- uses: actions/checkout@v6`。SHA ピンされていないこと、`@v6` に上げ済みであることを確認。 |
| AC4: YAML パース/構文チェックでエラーが出ない | Verified | `ruby -ryaml -e 'YAML.load_file(f)'` を 4 ファイル全部に対して実行し全て `OK`。Python `yaml` が手元になかったため ruby で代替（プランの Verify plan では python3 例示だが、パース手段としては等価）。 |
| AC5: `./scripts/run-verify.sh` および `./scripts/check-sync.sh` が成功する | Verified | `run-verify.sh` EXIT=0（shellcheck / sh -n 19 スクリプト / jq -e 設定 2 ファイル / check-sync / mojibake テスト 11 件 / gofmt / go vet / go test ... すべて pass）。`check-sync.sh` 単独も EXIT=0、`DRIFTED=0 / IDENTICAL=107`。 |
| AC6: PR 作成後の CI (verify, check-template) が green | Not verified (out of scope) | `/verify` は静的解析までが責務。CI 実走確認は PR 作成後の human-in-loop 側で確認する前提（プラン通り）。 |
| AC7: 次回リリース時に deprecation 警告が消えることを確認する手順が PR description に記載されている | Not verified (out of scope) | PR description はまだ未作成（`/pr` 段階）。自己レビュー報告にも記載なし。`/pr` で網羅すべき内容として flagged。 |
| AC8: goreleaser-action v6→v7 はタグ push 時に初めて実走することを PR description で明示し、フォロー課題を記録している | Not verified (out of scope) | 同上。follow-up 自体はプラン Open questions と自己レビューの tech-debt 表に記録済み。PR 本文記載は `/pr` の責任。 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-verify.sh` | EXIT 0 | shellcheck / sh -n / jq -e / check-sync / mojibake tests / gofmt / go vet / go test all OK. |
| `./scripts/check-sync.sh` | EXIT 0 | DRIFTED=0 / IDENTICAL=107 / KNOWN_DIFF=3 (いずれも本 PR に無関係な既存差分)。 |
| `./scripts/check-template.sh` | EXIT 0 | `Template structure looks good.` |
| `./scripts/run-static-verify.sh` | EXIT 0 | static モード。Go テストを除き同一結果。 |
| `ruby -ryaml -e 'YAML.load_file(f)'` × 4 workflow files | OK × 4 | `release.yml`, `verify.yml`, `check-template.yml`, `templates/base/.github/workflows/verify.yml` いずれもパース成功。 |
| 旧 SHA grep (`11bd71901bbe...`, `d35c59abb061...`, `e435ccd77726...`) | Plan 本文以外 hit なし | 履歴的記述はプラン本文 L73 のみ、実コードからは消失。 |
| 新 SHA grep (`de0fac2e4500...`, `4a3601121dd0...`, `e24998b8b67b...`) | 期待どおり hit | 3 ファイルで 5 回参照。プランの Assumptions と完全一致。 |
| 残存する `@v[1-5]` 旧 Node 系 action 参照 | 該当なし | `actions/(checkout\|setup-go\|setup-node\|upload-artifact\|download-artifact\|cache\|setup-python)@v[12345]\|goreleaser/goreleaser-action@v[12345]` の hit はプラン文言のみ。 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `AGENTS.md` | In sync | action バージョン / SHA への直接言及なし。 |
| `CLAUDE.md` | In sync | 同上。 |
| `.claude/rules/` | In sync | action バージョン / SHA / deprecation 関連ルールはなし（ドリフトの起点なし）。 |
| `docs/` 配下 | In sync | 旧 SHA を書いた他のドキュメントは存在せず（プラン L73 は「消えたことを確認する grep 対象」としての言及で意図どおり）。 |
| プラン Progress checklist | Drifted | `[ ] Verification artifact created` が未チェック（この /verify 実行で作成したためドリフト、ドキュメント上だけの遅延で blocking ではない）。`/sync-docs` 段階で更新される想定。 |
| `docs/tech-debt/` | In sync | 自己レビューに tech-debt 2 件（release.yml dry-run 経路 / template tag-only 方針コメント）が記録済み。`docs/tech-debt/` への転記は `/sync-docs` の役割。 |

## Observational checks

- 自己レビュー (`docs/reports/self-review-2026-04-22-upgrade-gha-actions-node24.md`) は APPROVE・CRITICAL/HIGH なし。LOW 2 件（タグコメント粒度、template コメント不足）は本 PR スコープ外の follow-up として tech-debt に記録済み。
- ルート 3 ファイルの `uses:` 行が完全に `<40 hex SHA> # vX.Y.Z` 形式で統一されており、タグコメントのパッチ粒度も `v6.0.2 / v6.4.0 / v7.1.0` と揃っている。
- `release.yml` のトリガー (`push tags v*`)、permissions (`contents: write`)、secrets (`GITHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`)、goreleaser オプション (`version: "~> v2"`, `args: release --clean`) はいずれも変更なし。ロジック差分ゼロ。
- `templates/base/.github/workflows/verify.yml` は `@v6` に更新されつつ tag-only 記法を維持（SHA 化されていない）。既存方針どおり。
- `check-sync.sh` の `KNOWN_DIFF` 3 件（`.github/workflows/verify.yml` / `AGENTS.md` / `CLAUDE.md`）は本 PR 以前から存在する認識済み差分であり、今回の変更で新たなドリフトは発生していない。

## Coverage gaps

- **CI 実走（AC6）**: `.github/workflows/verify.yml` / `check-template.yml` の GitHub Runner 上での成功は、PR push 後でなければ検証できない。ローカルでは YAML 構文と再利用スクリプト (`run-verify.sh`, `check-template.sh`, `check-sync.sh`) が通ることまでのみ確認。
- **goreleaser-action v7 の実走（プラン Risks）**: タグ push 時にしか起動しないため、本 PR の merge 後の実リリースまで実挙動を確認できない。プラン Open questions と自己レビュー tech-debt 表に follow-up として明記済み。
- **PR description 記載 (AC7, AC8)**: `/pr` 段階での担保事項。`/verify` 時点では未作成のため検証不能。
- **Behavioral tests**: `/test` の責務。`go test ./...` はローカル（`run-verify.sh` 経由）で pass しているが、ワークフロー自体の挙動は behavior テスト対象外。

## Verdict

- Verified: AC1, AC2, AC3, AC4, AC5（静的解析で検証できる 5 件は全て満たされている）。旧 SHA 完全除去、新 SHA プラン一致、YAML パス、全スクリプト pass、ドキュメントドリフトなし。
- Partially verified: なし。
- Not verified: AC6 (CI green は PR push 後に確認)、AC7 / AC8（`/pr` で PR description に記載する項目）。いずれも `/verify` のスコープ外で、後段で担保される運用設計。

**Pass/Fail: PASS**（`/verify` のスコープ内で確認可能な静的・スペック項目はすべて通過。残る 3 件は後段フェーズで担保する前提の known gap）。

### Minimal additional check that would most increase confidence

PR push 後に GitHub Actions 上で `verify` / `check-template` ワークフローを走らせ、実際に deprecation 警告が消えて全 job が green になることを 1 回確認する（AC6 を満たす最小のフォローアップ）。ローカル static では取れない情報は「GitHub 側の node24 ランタイム選択」だけなので、push 後のチェック 1 回で AC6 は解消できる。
