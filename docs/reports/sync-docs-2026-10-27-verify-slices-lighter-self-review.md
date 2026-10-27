# sync-docs report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(承認の digest `bd5c0d0ce1bb` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは AC の `[ ]` と `## Progress checklist` だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `44c14b97`、branch は `refactor/verify-slices-lighter-self-review`。run 開始時の HEAD は `869dcab4`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-27-verify-slices-lighter-self-review.md`(2 回目まで Merge yes、MEDIUM 1 と LOW 6 は解決、新しい LOW 3 も inline で解決)、`docs/reports/verify-2026-10-27-verify-slices-lighter-self-review.md`(pass、AC1〜AC6 Met、AC7 は静的の半分)、`docs/reports/test-2026-10-27-verify-slices-lighter-self-review.md`(pass、AC7 のテスト側を満たす、test gap 1 件)
- 差分の大きさ(`git diff 44c14b97...HEAD --shortstat`、この run の前): 43 ファイル、+1109/-118。walkthrough を書くかどうかは `/pr` が決める

## Summary

差分は、スライスごとの検証コマンドを計画の `## Implementation outline` に書かせ、`/implement` の手順 6 と 7 がそれを使い、手順 12 のフルの `./scripts/run-verify.sh` を外す変更である。implementer は commit の前に 5 項目の機械的な確かめをし、`/self-review` は 6 項目(命名、読みやすさ、null safety、例外処理、security、保守性)に絞られた。

root と `templates/base/` を洗い、旧い言い方が残る文書を探した。矛盾する文はなかった。足りなかったのは `.codex/README.md` の implementer の行だけで、これを root と template の両方で直した。skill と agent の frontmatter の description は、この差分がすでに 6 項目に直している。計画の外にある hook の文 2 つと、テストの穴 1 つは `docs/tech-debt/README.md` に記録した。

## Changes made

| File | Change |
|------|--------|
| `.codex/README.md`、`templates/base/.codex/README.md` | `## Agent roles` の implementer の行に、計画のスライスの検証コマンド、staged diff への機械的な確かめ(5 項目)、報告に確かめの結果が入ることを足した。2 つは同一のまま(`check-sync.sh` で確認) |
| `docs/tech-debt/README.md` | 末尾に 2 行を足した。(1) hook の「Run ./scripts/run-verify.sh before claiming done」(`post_edit_verify.sh` の `code` の文、`session_start_context.sh` の SessionStart の文、`templates/base/` の写し 2 つ)。(2) `tests/test-slice-verification.sh` の手順 6 の検査が「the narrowest check that covers the slice's files」を固めていない穴(LOW) |
| `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md` | AC1〜AC7 を `[x]` にした(verify と test の report が満たすと示している。AC7 は verify の静的の半分と test のテスト側で 6 つ)。`## Progress checklist` の「Verification artifact created」「Test artifact created」を `[x]` にし、結果と逸れ 3 件(テストの置き場、数の更新、テストの穴)を子の項目に書いた。それ以外の本文は触れていない |
| `docs/reports/sync-docs-2026-10-27-verify-slices-lighter-self-review.md` | 新規(この report) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `.claude/skills/self-review/SKILL.md` の frontmatter `description`(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/` の写しを含む) | 変更なし | 「Covers naming, readability, null safety, exception handling, security, and maintainability.」に直っている(`git diff 44c14b97...HEAD`)。skill 一覧に出る文はこの 6 項目 |
| `.claude/agents/reviewer.md` の frontmatter `description`、`.codex/agents/reviewer.toml` の `description`(`templates/base/` の写しを含む) | 変更なし | 2 つとも 6 項目に直っている。本文の「Focus on diff quality only」の行と、実装の側が見る 5 項目の文も合っている |
| `.claude/skills/self-review/template.md` の Area の値 | 変更なし | `naming, readability, null-safety, exception-handling, security, maintainability`。過去の report にある `unnecessary-change` と `typo` は履歴 |
| `/self-review` の旧い項目を並べた文(`Debug code`、`unnecessary changes`、`typos`、`Secrets and credentials`、`unnecessary-change`) | 変更なし | 履歴(`docs/reports`、`docs/plans/archive`)を除くと、残るのは実装の側の 5 項目として書いた文(model-routing、implement、self-review、implementer、reviewer の定義)と、復活を禁じる `tests/test-self-review-scope.sh` の検査だけ。`README.md`(220 行は「diff quality artifact」)、`AGENTS.md`、`.ralph/core/AGENTS.core.md`、`docs/quality/*.md`、`docs/architecture/repo-map.md`(49 行は「self-review diff quality」)に観点を並べた文はない |
| `/implement` が各スライス、または手順 12 でフルの `run-verify.sh` を打つと書く文(root、`templates/base/`、`.agents/`) | 変更なし | `grep -rn run-verify` を履歴(`docs/reports`、`docs/plans/archive`、`docs/evidence`、`docs/specs`、`docs/insights`)とテスト、`internal/`、`packs/` を除いて読んだ。手順 6・7・12・Completion gate は新しい形で、旧い形は復活を禁じる `tests/test-slice-verification.sh` にだけ残る。`.claude/rules/ralph/git-commit-strategy.md` の Validation Gate(手順 2)、`model-routing.md`(handoff の表と報告の契約)、`subagent-policy.md`(implementer の報告)は差分が直している。`model-routing.md` の root と template の差は KNOWN_DIFF の org receipts の 3 か所だけ |
| `README.md`(103 行「Before claiming a task is done: `./scripts/run-verify.sh`」) | 変更なし | タスク全体の完了の話。`/verify` と `/test` が包みのスクリプトを通して同じ検査を打つので、スライスの検証とは食い違わない。依頼で、この種の一般的な文は残すことになっている |
| `.claude/rules/ralph/ralph-workflow.md` の「Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success」(`templates/base/` の写しを含む) | 変更なし | 同じ理由。計画の Progress にも残した理由が書かれている |
| `.codex/AGENTS.override.md` の「Do not bypass `./scripts/run-verify.sh` before claiming work is done」(`templates/base/` の写しを含む) | 変更なし | 計画の Non-goals が残すと決めている |
| `docs/quality/definition-of-done.md`(`templates/base/` の写しを含む) | 変更なし | 「Each implementation slice is individually committed (see `git-commit-strategy.md`)」は Validation Gate を指し、その手順 2 が新しい形に直っている。self-review の行は「diff quality only」で、観点の個数は書かない。フェーズの境界の段落も新しい分担と食い違わない |
| `docs/quality/quality-gates.md`(`templates/base/` の写しを含む) | 変更なし | 「Must pass locally before done」はタスク全体の検査で、`run-static-verify.sh` と `run-test.sh` の分担を書く。スライスの検証コマンドには触れない |
| `docs/architecture/repo-map.md`、`docs/recipes/adding-a-language-pack.md`(61 行) | 変更なし | repo-map は skill を 1 行で述べるだけ。recipe の `./scripts/run-verify.sh` は言語パックを入れた直後の通し確認で、スライスの話ではない |
| `AGENTS.md`、`.ralph/core/AGENTS.core.md`、`templates/base/AGENTS.md`、`CLAUDE.md` | 変更なし | 手順の列(Implement、Self-review の 1 行)は変わらない。検証の中身は書いていない |
| `.codex/README.md` の implementer の行(`templates/base/` の写しを含む) | 直した | 「runs verification, and returns a report with commit-boundary evidence」は、計画のコマンドを使い、機械的な確かめをし、確かめの結果を報告に入れる、という契約より狭かった。model-routing の報告の契約に合わせて書き直した。この文を固める検査はない |
| `.claude/hooks/post_edit_verify.sh`、`.claude/hooks/session_start_context.sh`(`templates/base/` の写しを含む) | tech-debt に記録 | 「Run ./scripts/run-verify.sh before claiming done」が残る。hook は計画の範囲外。PostToolUse の文はコードのファイルを編集するたびに出るので、スライスごとにフルの検査を読む余地がある。implementer の定義の「run the handoff's verification commands exactly as given」が優先するので、矛盾ではなく後続の候補 |
| `internal/org/prompts/implementer.md`(20 行「指示がなければ `./scripts/run-verify.sh`」)、`git-commit-strategy.md` の「Org runtime seats commit the same way as the standard flow above」 | 範囲外(#262) | org の座席の検証は計画の Non-goals で #262 に渡されている。verify report の Documentation drift にも同じ記録がある |
| `docs/tech-debt/README.md` の追加 2 行 | 同期 | 5 欄(`' | '` で分けて 5 つ)。`file:line` の参照はなく、名前と文で引いた。引いた計画は `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md` で、存在する(`archive-plan.sh` が `/pr` で書き換える)。`docs/plans/(active|archive)/` の参照はすべて存在する |

## Plan drift notes

- Scope の「テスト」の項は、implementer の定義の機械的な確かめの文を固める検査を `tests/test-slice-verification.sh` に置くと書く。実際は S4(`d501efad`)から `tests/test-self-review-scope.sh` にあり、slice のテストには入っていない。verify report が情報として挙げていた逸れで、Progress checklist には書かれていなかったので、「Test artifact created」の子の項目に足した。中身は固められている
- Progress の S5 の「84 件」と bc56cc4f の「scope 198/198」は、その時点の数である。slice のテストは `cb094ae8` と `804d05f9` の検査の追加で 130 件になった(test report の実行)。これも同じ子の項目に書いた
- 計画の Acceptance criteria は AC1〜AC7 を `[x]` にした。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける

## `/pr` と release に渡すこと

- 新しい文を実際の `/implement` の実行で使った結果(implementer が 5 項目の結果を返すか、検証コマンドのない計画で orchestrator が Progress に書くか)は、この PR では確かめられない。今回の 5 スライスと直しのスライスの記録(Progress checklist の bc56cc4f の行)が 1 件の実例になる
- scaffold 先には次の release の `ralph upgrade` で届く。skills と agents は core、雛形の `docs/plans/templates/feature-plan.md` は seed なので upgrade は advisory を出すだけ(計画の Rollout)。release のノートに「雛形は seed なので、更新するにはユーザーが `feature-plan.md` に検証コマンドの欄を足す」と書くと分かりやすい。release はユーザーが切る
- scaffold 先で `/implement` を使う人には、手順 12 でフルの `run-verify.sh` が走らなくなる変更が届く。`/verify` と `/test` が直後に走る(計画の R2)

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.`(`.codex/README.md` を root と template で直したあと) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step`(skill は触れていないので `sync-skills.sh` は打っていない) |
| `sh tests/test-self-review-scope.sh` | rc 0。PASS 198 / 198 |
| `sh tests/test-slice-verification.sh` | rc 0。PASS 130 / 130 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md` | `bd5c0d0ce1bb`(`- Approved:` 行の値と同じ。AC の `[x]` と Progress の変更の前後で同じ) |
| tech-debt の計画参照(`check_tech_debt_plan_refs` と同じ grep) | 欠けた参照なし。追加した 2 行は 5 欄 |
| テスト | 上の 2 本のほかは打っていない(`/test` の範囲。この run が変えたのは文書だけで、コードとスクリプトには触れていない) |
| insight event | `./scripts/insights-append.sh --slug verify-slices-lighter-self-review --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記する(`docs/insights/events/2026-10-27-verify-slices-lighter-self-review.jsonl`) |
