# sync-docs report: org-gate

- Date: 2026-10-13(JST)
- Plan: `docs/plans/active/2026-10-13-org-gate.md`(承認済み。`./scripts/plan-visual.sh digest` は `69f088275e54` で、`- Approved:` の行と同じ。この run は計画を編集していない)
- Pipeline cycle: 1(`--cycle auto`)。差分の base は `1bddc871`、branch は `feat/org-gate`。run 開始時の HEAD は `bb2ed920`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-13-org-gate.md`(追記を含む。Merge 可、CRITICAL・HIGH 0)、`docs/reports/verify-2026-10-13-org-gate.md`(pass。「Documentation drift」の D1〜D4)、`docs/reports/test-2026-10-13-org-gate.md`(pass、失敗 0)
- 項目の番号: D1〜D4 は verify の「Documentation drift」の表の番号
- 内容の commit: この report、insight event、直した文書を含む 1 つの commit(`docs: sync docs for org-gate`)

## Summary

verify の D1〜D3 を文書だけで直した。D4(計画の Progress checklist と Notes)は、依頼どおり計画を編集していない。Go のコード、テスト、`--help` の文字列、leader と reviewer の雛形(`internal/org/prompts/`、go:embed)、skill、`scripts/` は触っていない。

そのほかに、動詞の一覧を持つ文書を掃いて、2 件を足した。README の `ralph org` の動詞の表に `gate` がなかった。quality-gates の「Cross-org watch gate」の行が「見張りがなくても動く動詞」を並べるのに、`gate` を挙げていなかった。

## Changes made

| File | D | Change |
|------|---|--------|
| `AGENTS.md:90`(repo map の `internal/org/` の行) | D1 | gate の項目を「`gate.go` with the script running in `gate_run.go`: ...」にした。スクリプトを走らせるのは e0adb359 で分けた `gate_run.go` で、この行に名前がなかった。同じ句の中に収め、行は 1 句しか伸びていない |
| `README.md`(機能ごとの org の段落、「One org handles one feature ...」) | D2 | 「The leader opens the PR with `gh pr create` and disbands the org」を「The leader rebases the branch on the default branch, runs `ralph org gate` (below), opens the PR with `gh pr create`, and disbands the org」にした。ほかの文は変えていない |
| `README.md`(`approve` の段落の次に新しい段落) | D2 | `ralph org gate` の段落を足した。escalate・watch・approve の段と同じ並びと文体で、leader が打つ時点(review のあと、rebase と全コミットのあと、push する head)、走らせる 2 つのスクリプトと環境の固定(`RALPH_VERIFY_*`・`HARNESS_VERIFY_*` を除き `RALPH_VERIFY_SCOPE=full`)、ログの置き場所(`<state dir>/gates/`)、`gate_passed`/`gate_failed` と `reason` の 4 つ、出力の `GATE:`・`HEAD:`・`BASE:`、終了コード 1、走らせる前の拒否、見張りと承認が要らないこと、gate なしで PR を出せること、を書いた。値と文言は `go run ./cmd/ralph org gate --help` と `gate.go` の拒否の文から取った |
| `README.md:124`(コマンド表の `ralph org ...` の行) | 掃引 | 動詞の列の末尾に `/gate` を足し、`approve ...` の文の次に `gate` の 1 文を足した |
| `docs/quality/quality-gates.md:82` と `templates/base/docs/quality/quality-gates.md:81`(「Quality pipeline gate」の行) | D3 | 「refuses before running ... on a dirty tree, a head that does not contain ..., or a directory outside ...」を、拒否を全部挙げる文にした: 完全な結びつきを持って走っている機能の org でない、worktree の外かブランチ違い(detached HEAD と rebase の途中を含む)、作業ツリーが汚れている、既定のブランチが見つからない、head が既定のブランチの先頭を含まない、`scripts/run-static-verify.sh` か `scripts/run-test.sh` がない・実行できない。2 面の追加文は同じで、残る差は元からの KNOWN_DIFF の「(see docs/tech-debt)」だけ(Python で、root の行からその句を除くと template の行と一致することを確かめた) |
| `docs/quality/quality-gates.md:80` と `templates/base/docs/quality/quality-gates.md:79`(「Cross-org watch gate」の行) | 掃引 | 「`stop`, `disband`, `escalate`, and the inbox verbs work with no watch」に `gate` を足した(両面とも同じ文)。gate は見張りの関門をかけない(AC7、`TestGate_NoWatchGateNoApprovalCheck`) |
| `docs/insights/events/2026-10-13-org-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-gate`、`--cycle auto`、cycle 1、pass)。既存のファイルへの追記で、新しいファイルは増えていない |
| この report | - | 新規 |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `/org` skill の 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 面) | 変更なし。4 面の sha256 は 1 種類。動詞の表の `gate` の行(181 行)、「org の終わり方」の手順 1〜7(758 行から)、「gate の記録」(833 行から)を `--help` と `gate.go`・`gate_run.go`・`org_gate.go` に当て直した。拒否の条件、記録のキー、4 行と 5 行の出力、3 つの信号、拒否のときに stdout に何も出ないこと、見張りと承認が要らないことが合う。leader 雛形の手順 9〜16 とは順と revert の規則が同じ |
| `internal/org/prompts/leader.md`、`reviewer.md`(go:embed。触らない) | ずれなし。leader の手順 9〜16 と skill の手順 1〜7 の対応、reviewer の RESULT の `HEAD:`・`BASE:`(ralph は読まない)は、skill と tech-debt の記述と合う |
| `docs/specs/2026-10-07-org-multi-org-director.md:73`(FR-7「7 段目の後半(7b)で決めたこと」) | 変更なし。gate の動き、環境の固定、revert、`run-verify.sh` の一時ファイル、7c に残したもの(doctor、status、merge してよい順)が実装と合う。FR-7 と FR-8 の checkbox は 7c が残るので未完のまま |
| `docs/specs/2026-08-01-org-runtime.md` | 変更なし。`ralph org` の動詞は `start`・`spawn`・`status` を本文で挙げるだけで、`escalate`・`watch`・`approve` も載せていない。機能ごとの org の PR の手順は書いていない。冒頭の「2026-10-07 後継の仕様」の注が、機能ごとの org は後継の仕様を優先すると言っており、本文は書き換えない方針(同じ冒頭の注) |
| `docs/recipes/`(`codex-seat-permissions.md` とその template の写し、`worktrees.md`) | 変更なし。reviewer 雛形の変更は、通ったときの RESULT に `HEAD:`・`BASE:` を足しただけで、recipe の TASK(sandbox probe、ゲートを走らせない、RESULT を送る)の前提は変わらない。live の codex 座席では確かめていない。`worktrees.md` の `cleanup` の手順は変わっていない |
| `.claude/rules/ralph/`(`agent-messaging.md` と template の写し、`post-implementation-pipeline.md`、`subagent-policy.md`、`model-routing.md`、`git-commit-strategy.md`) | 変更なし。gate は agmsg のメッセージを送らず、`HEAD:`・`BASE:` は自由ヘッダなので、TYPE の表と TASK_ID の規則に影響しない。機能の org の PR の手順を書く rule はない。`run-verify.sh` の受け渡しのファイル名(`verify-exit-code`・`verify-scope`・`non_docs_change`)を挙げる rule や文書もない(`git grep` で出るのは `scripts/`、`tests/`、計画だけ) |
| `docs/quality/quality-gates.md` の `run-verify.sh` の説明(25〜39 行)と `docs/quality/definition-of-done.md` | 変更なし。`run-verify.sh` の既定の範囲が `full` であること、wrapper の既定が `changed` であることは、gate が `RALPH_VERIFY_SCOPE=full` を足す動きと矛盾しない。受け渡しのファイルを一時ファイルにしたのは内部の変更で、文書は名前を挙げていない。definition-of-done の org の節は、仕様と rule を指すだけ |
| `docs/architecture/repo-map.md`(`tests/` の行、`.harness/state/org/` の行) | 変更なし。`tests/` は `tests/test-*.sh` の総称で、新しい `tests/test-run-verify-concurrency.sh` を含む。`.harness/state/org/` の行は manifest・saga・receipts・watchdog の状態を挙げる要約で、受信箱や `splits/` も挙げておらず、新しい `gates/`(gate のログ)だけを足すと粒度がずれる |
| `templates/base/ralph.toml` | 変更なし。gate は設定を持たない(計画の Non-goals) |
| `templates/base/scripts/run-verify.sh`(root と同じ中身) | `cmp` で同一。ヘッダのコメントが `ralph org gate` を名指すが、`ralph` の CLI の名前で、meta-repo のパスではない。`./scripts/check-template-purity.sh` は rc 0 |
| `docs/tech-debt/README.md`(`:155` の追記と、末尾の 10 行) | 変更なし。参照するパスと関数・テストの名前をすべて取り出して `git grep` で確かめた(欠け 0。スクリプトは scratchpad)。行番号は使っておらず、関数名で指している。計画のパスは `docs/plans/active/` で、`archive-plan.sh` が `/pr` で書き直す。`run-static-verify.sh` の `tech-debt README plan references` も OK。leader 雛形の手順番号(10、12、13、14)は雛形の今の番号と合う |
| `AGENTS.core.md`(`.ralph/core/` と template の写し)、`templates/base/AGENTS.md` | 変更なし。`gate.go` を挙げていない(meta-repo の repo map だけが `internal/org/` を書く)。AGENTS.md の 90 行は managed block の外の meta-repo の行 |
| `docs/reports/`、`docs/evidence/`、`docs/plans/active/2026-10-13-org-gate.md` | 編集していない。計画の Progress checklist(`Test artifact created`)と AC の checkbox の印は、依頼に従って触っていない |

## Left open

- D4: 計画の Progress checklist の `Test artifact created`、AC1〜AC13 の checkbox は未チェックのまま。依頼が計画を編集しないと言っているので、`/pr` の前に計画の持ち主が付ける
- `docs/evidence/test-2026-10-13-org-gate-go-test.txt`(4 行)、`-race.txt`(4 行)、`-mutation.txt`(10 行)に、ホームの絶対パス(`/Users/<user>/...`)が残っている。`docs/quality/definition-of-done.md` の末尾は、`docs/evidence/` にコミットする evidence で `$HOME` を `~` にしてから commit するよう求めている。verify の evidence 4 つには残っていない。同じ形は前の PR の `org-plan-approval` の evidence(`go-test` と `go-race` は各 4 行、mutation は 1〜2 行)にもあり、base の 1bddc871 に入っている。evidence は /test の成果物で、この run の範囲(文書の同期)の外なので直していない。直すなら `/Users/<user>` を `~` に置き換える 1 コミットで足りる。mutation のファイルの `-overlay` の JSON の中のパスは、置き換えると JSON の意味(実在するパス)が変わる点に注意
- leader と reviewer の雛形どおりに、herdr の pane の中の本物の leader が gate を打てるかは未確認(verify の Coverage gaps と tech-debt の `:222`・`:228`)。文書では埋まらない
- release のノートと PR の本文は、この run の範囲に含めていない(`/pr`)

## Verification

コミットの前に、worktree のルートで打った。

- `./scripts/check-sync.sh`: rc 0、`IDENTICAL: 167`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `./scripts/check-skill-sync.sh`: rc 0、`13 skill(s) in lock-step`
- `./scripts/check-template-purity.sh`: rc 0、`PASS: no meta-repo-specific references found in templates.`(`templates/base/docs/quality/quality-gates.md` を変えたので打った)
- `./scripts/run-static-verify.sh`(changed の範囲): rc 0。`shared:scripts/run-verify.sh` の full fallback で golang の pack が走り、`gofmt: ok`、`0 issues.`、branch の secret scan は `1bddc871..bb2ed920` で clean。`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、`tech-debt README plan references` も OK。wrapper が書いた `docs/evidence/verify-2026-10-13-083024.log` は gitignore の対象で、コミットしていない
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-13-org-gate.md`: `69f088275e54`(変わっていない)
- skill の 4 面の `shasum -a 256`: 1 種類
- 2 つの quality-gates の「Quality pipeline gate」の行: root の行から `(see docs/tech-debt)` を除くと template の行に一致する(Python で確認)
- `./scripts/insights-append.sh --slug org-gate ...` のあとの `git status --short`: 既存の `docs/insights/events/2026-10-13-org-gate.jsonl` が `M` で、新しい `??` のファイルはない
