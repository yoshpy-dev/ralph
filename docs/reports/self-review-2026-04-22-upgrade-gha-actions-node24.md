# Self-review report: upgrade-gha-actions-node24

- Date: 2026-04-22
- Plan: `docs/plans/active/2026-04-22-upgrade-gha-actions-node24.md`
- Reviewer: reviewer subagent (self-review)
- Scope: Diff quality only for branch `ci/upgrade-gha-actions-node24` vs `main`. Spec compliance, CI passing, and test coverage are out of scope (handled by `/verify` and `/test`).

## Evidence reviewed

- `git diff main...HEAD --stat` — 5 files changed: 3 root workflows, 1 template workflow, 1 plan doc. Total +134 / -6 lines (ほぼ全量が plan 新規)。
- `git diff main...HEAD -- .github/workflows/ templates/base/.github/workflows/` — ワークフロー差分は純粋に `uses:` 行のバージョンピン置換のみ。
- `.github/workflows/release.yml` (現行): トリガー `push tags v*`, permissions `contents: write`, secrets 参照は既存の `GITHUB_TOKEN` / `HOMEBREW_TAP_GITHUB_TOKEN` のまま変化なし。
- `.github/workflows/verify.yml` / `check-template.yml`: `uses:` 行のみ変更、step 名・run コマンド・オプションは不変。
- `templates/base/.github/workflows/verify.yml`: tag-only 記法 (`@v4` → `@v6`) を維持。他の step は変更なし。
- Repo-wide grep で旧 SHA (`11bd71901bbe...`, `d35c59abb061...`, `e435ccd777264...`) が残存していないことを確認 (active plan の歴史的記述を除く)。
- Ruby で 4 ファイル全てを `YAML.load_file` してパース成功を確認。
- 新規 SHA はプランの 2026-04-22 API 確認値と 1 文字差なく一致:
  - checkout `de0fac2e4500dabe0009e67214ff5f5447ce83dd` (v6.0.2)
  - setup-go  `4a3601121dd01d1626a1e23e37211e3254c1c06c` (v6.4.0)
  - goreleaser-action `e24998b8b67b290c2fa8b7c14fcfa7de2c5c9b8c` (v7.1.0)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | タグコメントの粒度が action 間で不揃い。`checkout` と `setup-go` は `v6.0.2` / `v6.4.0` とパッチまで書くが、元の `goreleaser-action` は `# v6` とメジャーのみで、今回の更新で `# v7.1.0` に粒度が引き上がった。今回の 3 action はすべて「SHA + `# <タグ>` 」で統一されており、粒度も「パッチまで」で揃っているため現状 OK。ただし、今後 action を追加/再ピンする際の暗黙ルールが明文化されておらず、次のレビューで同じ議論になりやすい。 | `.github/workflows/release.yml` L15, L19, L23 の `# v6.0.2` / `# v6.4.0` / `# v7.1.0` というコメント形式。 | `.claude/rules/` か PR 概要で「タグコメントはパッチ付きで書く」方針を一度言語化する (follow-up)。本 PR では指摘のみで修正不要。 |
| LOW | maintainability | ルート側は SHA ピン / テンプレート側は tag-only という方針の差が、コードだけを見ても意図が伝わらない。今回の diff では `templates/base/.github/workflows/verify.yml` で `@v4` → `@v6` とメジャーだけ上げており、もし将来レビュアが「なぜ SHA ピンしないのか」と疑問に思っても、コメントがないので active plan を開かないと理由にたどり着けない。 | `templates/base/.github/workflows/verify.yml:8` `- uses: actions/checkout@v6` に根拠コメントなし。 | 本 PR ではスコープ外 (non-goal 側)。tech-debt として「template 配布側の action ピン方針を 1 行コメントで明示する」を記録 (下段の表参照)。 |

## Positive notes

- 差分はまさに「プランどおり 4 行の置換 + plan 追加」だけに収まっており、scope creep なし。ワークフローのトリガー・権限・オプション・secrets 参照は一切変更されていない (LGTM: 最小変更)。
- SHA コメントが `v4.2.2` → `v6.0.2` のようにメジャーだけでなくパッチまで明記されており、将来のレビュア・監査で「このピンは何のタグを指していたか」を 1 行で特定できる。
- `HOMEBREW_TAP_GITHUB_TOKEN` を含む secrets 参照は既存のまま。コミット本文や新規ファイルにトークンや API キーのハードコードなし (repo-wide grep で確認済み)。
- ルートは SHA ピン、配布 template は tag-only という既存方針が維持されており、「配布物の利便性」と「自リポの供給網攻撃耐性」のトレードオフが意図的に区別されている。
- plan の Open questions で goreleaser v7 実走検証 (dry-run 追加) を follow-up PR に切り出す意図が明記されており、diff 側でタグ push 前に検知不能なリスクを追加していないことが確認できる。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `release.yml` にタグ push 前ドライラン経路が無い (goreleaser-action v7 は次回タグ push で初めて実走する) | v7 にリリース時破壊変更があるとその場で本番 release が失敗しうる。plan の Open questions / Codex Finding 1 (HIGH) と同一。 | 今回の PR は「警告解消」だけにスコープを絞る合意。ドライランジョブ追加は別 PR で扱う。 | 次に `release.yml` を触るタイミング、または goreleaser v7 のタグを新たに切る前。 | `docs/plans/active/2026-04-22-upgrade-gha-actions-node24.md` Open questions, Codex plan advisory 2026-04-22 Finding 1 |
| `templates/base/.github/workflows/verify.yml` の action ピン方針 (tag-only) が配布テンプレート内でコメント未記載 | 将来、配布先のユーザ / 他の自動化が「なぜ SHA ピンされていないのか」を読み取れず、半端に SHA 化して同期が壊れる可能性。 | 本 PR の non-goal (配布 template の記法維持)。方針自体が決定済みであり、コメント追加は別 PR で十分。 | template 配布ドキュメント (`docs/`) を整備するタイミング、または template 側で security-sensitive action を増やすとき。 | 本レポート (LOW finding #2) |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: Yes — CRITICAL / HIGH 所見なし。差分はワークフローの version pin 置換に閉じており、ロジック・権限・トリガー・secrets 参照・オプションはどれも変わっていない。YAML パースも 4 ファイル全て成功。SHA はプランの 2026-04-22 API 確認値と完全一致しており、旧 SHA が other files / docs に残存していないことも grep で確認済み。LOW 2 件は follow-up で足りる粒度。
- Follow-ups:
  1. (tech-debt) `release.yml` にタグ push 前のドライラン経路を追加する別 PR を立てる (Codex plan advisory Finding 1 / plan の Open questions)。
  2. (tech-debt) 配布 template (`templates/base/.github/workflows/`) の action ピン方針を、一行コメントか `templates/base/` 配下の README で明示する。
  3. (任意) `.claude/rules/` に「ルート workflow の action は `<sha> # <vMAJOR.MINOR.PATCH>` 形式でピン」ルールを明文化して、将来の粒度ブレを予防する。
