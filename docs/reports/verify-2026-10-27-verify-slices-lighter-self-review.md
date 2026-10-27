# Verify report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(AC1〜AC7、D4〜D6、Verify plan、Progress checklist の記録)。issue #261 の受け入れ条件 5 つ
- Verifier: verifier サブエージェント(`/verify`)。パイプラインの cycle 2 / 2(cross-review の ACTION_REQUIRED を直したあとの回し直し)。cycle 1 の verify は 395b9bd8(HEAD 804d05f9、pass)で、この report はそれを置き換える
- Scope: `git diff 44c14b97...HEAD`(HEAD 53326aff、48 ファイル、+1271/-122、22 コミット)。重点は cycle 1 の verify のあとに入った振る舞いの文とテストの 3 コミット: dba1c20f(cross-review の直し: 止まったスライスを再開として出し直す)、d33bb358(cycle 2 の self-review 1 回目の LOW 3 件の直し)、f2f0333c(sync-docs: `.codex/README.md` の implementer の行、tech-debt の行)。`git diff 395b9bd8 HEAD` は 21 ファイル。テストは打っていない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-27-verify-slices-lighter-self-review.log`(gitignore。`run-static-verify.sh` の全出力、同期の検査、shellcheck、`sh -n`、`git diff --check`、TOML の読み込み、写しの `cmp`、digest、5 項目の並びの洗い出し)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 雛形 2 つとその写しの `## Implementation outline` に、検証コマンドの書き方と `RALPH_VERIFY_SCOPE=changed` の注意 | Met | cycle 1 から変わっていない(`git diff 395b9bd8 HEAD` に雛形は入っていない)。`docs/plans/templates/feature-plan.md`、`.claude/skills/plan/template.md`、`.agents/skills/plan/template.md` と `templates/base/` の同名は `cmp` で同一 |
| AC2 `/plan` の手順 7 がスライスごとの検証コマンドを求める | Met | cycle 1 から変わっていない。`.claude/skills/plan/SKILL.md`、`.agents/` の写し、`templates/base/` の 2 つは同一 |
| AC3 handoff と inline のスライスが計画のコマンドを使い、手順 12 と Completion gate にフルの `run-verify.sh` がない。D4 と D6 がある | Met | `.claude/skills/implement/SKILL.md:40`(手順 6: 計画のコマンド、D4、末尾に「the uncommitted changes a resume hands over (step 7) are the one exception」)。`:42`(委譲のスライス: コミットせずに止まった場合を、範囲内で直せない検証の失敗と秘密情報の 2 つにした。先に秘密情報を worktree から出す。再開として出し直し、handoff が残った変更をそのスライスのものと名指す。絞って落とすファイルは `git restore --staged --worktree -- <file>` か削除)。`:43`(inline: 秘密情報はファイルと行で書き、値は書かない。秘密情報を worktree から出してから止まる)。`run-verify` が出るのは `:40`(最後の手段)と `:49`(手順 12 の「打たない」)だけ。Completion gate は `:84-85` で `/verify` と `/test` を求める |
| AC4 Validation Gate、model-routing の handoff と報告の契約、subagent-policy | Met | `.claude/rules/ralph/model-routing.md:39`(handoff の表に「Resume (only when dispatching a stopped slice again)」の行)、`:41-55`(報告の契約: 秘密情報はファイルと行で、値は書かない。止まった秘密情報のスライスは再開として出し直す)。`git-commit-strategy.md:10-13` と `subagent-policy.md:52-59` は cycle 1 から変わっていない。subagent-policy の「must carry」は 6 項目のままで、表の Resume は「(only when …)」の任意の欄なので食い違わない。model-routing の root と `templates/base/` の差は KNOWN_DIFF の org receipts の 3 か所だけで、基準の 44c14b97 と同じ形(行番号が 14 ずれただけ)。ただし `:51-55` の文は手順 7 より弱い(下の LOW 2) |
| AC5 implementer(Claude Code と Codex)に機械的な確かめと報告の欄 | Met | `.claude/agents/implementer.md:21`(再開の例外: handoff が残った変更をそのスライスのものと名指したら、止まらずに引き継ぐ)、`:40`(秘密情報で止まるときはファイルと行を報告し、値は書かない)、`:48`(報告の欄)。`.codex/agents/implementer.toml:22-25,52-57,66-69` が同じ。3 つの文は空白を詰めると 4 つの定義(root と `templates/base/` の 2 形式)に 1 回ずつ出る。TOML 4 つは `tomllib` で読める |
| AC6 `/self-review` が 6 項目で、外した 4 項目は implementer の確かめにある。reviewer の定義と Area も合う | Met | cycle 1 から変わっていない(self-review の skill、template、reviewer の定義は `git diff 395b9bd8 HEAD` に入っていない)。D5: self-review の skill 2 つ、`reviewer.md`、`reviewer.toml` に `run-test.sh`・`run-static-verify.sh`・`run-verify.sh` は 0 件 |
| AC7 同期の検査とテスト 6 つが通る | Partially(静的の半分) | `check-skill-sync.sh`(14 skill)と `check-sync.sh`(DRIFTED 0)は通った。`tests/test-agent-models.sh`、`test-self-review-scope.sh`、`test-new-feature-plan.sh`、`test-slice-verification.sh` の実行は `/test` に残す。2 本の新しいテストは shellcheck と `sh -n` が通る |

issue #261 の受け入れ条件:

| issue の受け入れ条件 | Status | Evidence |
| --- | --- | --- |
| 雛形のスライスに検証コマンドの欄と `RALPH_VERIFY_SCOPE=changed` の注意 | Met | AC1 と同じ |
| handoff が計画の検証コマンドを写し、手順 12 にフルの `run-verify.sh` がない | Met | AC3 と同じ |
| implementer の報告の契約に機械的な確かめの結果(Claude Code と Codex) | Met | AC5 と同じ。正本は model-routing の報告の契約 |
| `/self-review` が 6 項目で、外した 4 項目は implementer の確かめにある | Met | AC6 と同じ |
| 既存の同期の検査が通る | Met(静的の半分) | `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh` が通った。`test-agent-models.sh` は `/test` |

cycle 2 の直し:

- cross-review の ACTION_REQUIRED 1 件(止まったスライスを新しい implementer に出し直すと、範囲内の未コミットの変更を見て止まる)は、文の上では解消している。止まる規則(`implementer.md:21` と Codex の同じ段落)に再開の例外が入り、出し直す側の手順 7(`implement/SKILL.md:42`)、handoff の表(`model-routing.md:39`)、手順 6 の「baseline をきれいにする」の例外(`:40`)が同じ再開の handoff を指す。実際に止まったスライスを再開して確かめてはいない(Coverage gaps)
- cycle 2 の self-review 1 回目の LOW 3 件(`git restore -- <file>` で新規とステージ済みが戻らない、秘密情報の値を書かない指示がない、表と手順 6 に再開がない)は、d33bb358 で 3 件とも文が入った。手順 7 が理由に挙げる「`wip:` のチェックポイントは `git add -A` で全部を stage する」は `.claude/hooks/session_end_summary.sh:43` と `.claude/hooks/precompact_checkpoint.sh:46` で確かめた
- 計画の digest: HEAD で `bd5c0d0ce1bb`。`- Approved: 2026-10-27 sha256:bd5c0d0ce1bb` と一致する。cycle 2 の計画の変更は `## Progress checklist`(digest が読まない節)と AC の印だけ
- tech-debt の解決済みの行(手順 6 の「narrowest」の検査の穴)が挙げる「手順 6 の narrowest を全部変えると 8 件落ちる」は HEAD でも文の上で成り立つ。手順 6 の「narrowest」は 2 か所(`implement/SKILL.md:40`)で、どちらも needle に入る(`tests/test-slice-verification.sh:159,165`)。4 写し × 2 = 8。コメントの「134/138」「130/138」は dba1c20f の時点の件数で、HEAD の件数は Progress の記録では 148。数え直しは `/test`

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | changed scope を頼んだが、`.codex/agents/implementer.toml` を分類できず full にした(`unclassified`)。golang の pack は gofmt ok、golangci-lint `0 issues.`。local verifier の shellcheck、hook の `sh -n` 20 本、guard の awk の読み込み、settings の `jq`、Codex の hook の 3 つの guard、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画参照はすべて OK。secret scan は 44c14b97..53326aff で clean。wrapper は `docs/evidence/verify-2026-10-27-133320.log`(gitignore)にも出力を残した |
| `./scripts/check-skill-sync.sh` | rc 0 | 14 skill が一致 |
| `./scripts/check-sync.sh` | rc 0、PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | rc 0 | 6 ファイルともパイプラインの全段に触れる |
| `shellcheck tests/test-slice-verification.sh tests/test-self-review-scope.sh` | rc 0 | 出力なし |
| `sh -n` で 2 本のテスト | rc 0 | |
| `git diff --check 44c14b97...HEAD` | rc 0 | 出力なし |
| `tomllib` で `.codex/agents/{implementer,reviewer}.toml` と `templates/base/` の 2 つ | OK | 4 つとも `name`、`description`、`developer_instructions` |
| `./scripts/plan-visual.sh digest` | `bd5c0d0ce1bb` | 承認の値と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 再開の文の写し(root と `templates/base/`、`.agents/`、`.codex/`) | Yes | `implementer.md`、`implementer.toml`、`implement/SKILL.md`、`.agents/skills/implement/SKILL.md`、`.codex/README.md` は `templates/base/` の同名と `cmp` で同一。`.claude/skills/implement/SKILL.md` と `.agents/` の写しも同一。model-routing は KNOWN_DIFF の 3 か所だけが違い、cycle 2 の 2 つの hunk は両方に同じ形で入っている |
| implementer の「止まる規則」をほかの文書が書いていないか | Yes | `pre-existing modification` と `overlap … files in scope` を履歴の外で洗うと、implementer の定義 4 本と手順 6 の 4 写しだけに出る。README.md、AGENTS.md、`docs/quality/`、`.codex/*.md` に止まる規則の言い換えはない |
| 5 項目の並び(model-routing の「every other mention repeats word for word」) | No(LOW 1) | 空白を詰めて洗うと、22 か所のうち 20 か所が正本と同じ語。`.codex/README.md:74` と `templates/base/.codex/README.md:74` の 2 か所だけが「secrets, files outside the slice」で、「and」がない。f2f0333c(sync-docs)で入った行で、2 本のテストが固める 7 ファイルに README は入っていない |
| 秘密情報で止まったときの orchestrator の動き(model-routing と手順 7) | 一部ずれ(LOW 2) | 手順 7(`implement/SKILL.md:42`)は d33bb358 で「First take a secret out of the worktree … Then narrow the slice or revise the plan (step 10) if needed」になり、秘密情報を出すのが先で、絞るのは必要なときだけになった。`model-routing.md:51-55` は「the orchestrator removes the secret or narrows the slice and dispatches it again as a resume」のままで、どちらか一方でよいと読める。絞っても秘密情報のファイルが範囲に残る場合、rule だけを読むと秘密情報を worktree に残したまま出し直せる。段落は末尾で手順 7 を指し、手順 7 の方が強い。self-review の 2 回目も、所見にしなかった気づきの 1 つ目で同じ点を挙げた |
| `.codex/README.md` の implementer の行と再開 | Yes(情報) | 行は要約で、再開の例外を書いていない。定義の本文(`.codex/agents/implementer.toml:22-25`)にあるので食い違いではない |
| タスク全体の「done の前に `run-verify.sh`」 | Yes(情報、tech-debt に記録済み) | `.claude/rules/ralph/ralph-workflow.md`、`README.md`、`.codex/AGENTS.override.md`、hook の文(`.claude/hooks/post_edit_verify.sh:129`、`.claude/hooks/session_start_context.sh:20`、`templates/base/` の同名)。どれもタスクの完了の話で、`/verify` と `/test` が包みのスクリプトを通して打つので食い違わない。`post_edit_verify.sh` はコードのファイルを編集するたびに「Run ./scripts/run-verify.sh before claiming done」を足すので、implementer がスライスの報告の前にフルの検査を打つよう読む余地がある。cycle 1 のこの指摘は f2f0333c で `docs/tech-debt/README.md` の行になった(行の主張「実際の文を固めるテストはない」は、`tests/test-ralph-dispatch.sh:180` が stub の出力であることで確かめた) |
| org 側の検証の文 | 範囲外(#262) | `internal/org/prompts/implementer.md` は cycle 1 から変わらない。計画の Non-goals(#262) |
| cycle 1 の情報の所見の残り | 解消 | テストの置き場の逸れは Progress checklist の「逸れ(Scope の「テスト」の項)」に記録された。AC1〜AC7 の印は `[x]` になった |
| 計画の Progress checklist | 遅れ(情報) | 「Verification artifact created(verify 395b9bd8 …)」は cycle 1 の report を指す。この report の commit のあとに、orchestrator が cycle 2 の verify を書き足す |

## Observational checks

- 手順 7 の inline は「If a secret cannot be removed within scope, do not commit: unstage it, take it out of the worktree as for a delegated slice, and stop. Then commit with conventional format」と続く(`implement/SKILL.md:43`)。cycle 1 で書いた「Then commit」の読み方の話は変わらない。文の読みやすさの話なので判定には入れない
- 手順 7 の「`wip:` のチェックポイントは `git add -A` で全部を stage する」は標準フローの既定の話で、`RALPH_WIP_CHECKPOINT=off` のときは hook がコミットしない(`session_end_summary.sh:40-41`)。標準フローの session では既定で on なので、文の主張は合っている

## Coverage gaps

- AC7 のテスト 4 つと `./scripts/run-test.sh` の全体は `/test` に残す。cycle 2 で増えた needle(`tests/test-self-review-scope.sh:268-273`、`tests/test-slice-verification.sh:166-168,179-191,201-203,259-270`)が通るかは `/test` が数える
- 再開の手順を実際に使った例はない。この PR の直し(dba1c20f、d33bb358)は、どちらもコミット済みの状態から新しい implementer に渡しており、止まったスライスの再開ではない。implementer が再開の handoff を読んで止まらずに引き継ぐかは、文の上でしか確かめていない
- Codex の implementer と reviewer の定義は、TOML として読めることまでを確かめた。Codex の実行での振る舞いは確かめていない
- 変異でテストが落ちることは、実装側の記録(dba1c20f の「4 件落ちる」)と上の文の上の数え方による。この verify では変異を打っていない

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC5、AC6。issue の受け入れ条件 1〜4。cross-review の ACTION_REQUIRED 1 件と cycle 2 の self-review 1 回目の LOW 3 件は、文の上で解消している。計画の digest は承認の値のまま。静的解析はすべて通った(`run-static-verify.sh` rc 0、`check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、shellcheck、`sh -n`、`git diff --check`、TOML の読み込み)。root と `templates/base/`、`.agents/`、`.codex/` の写しはずれていない
- Partially verified: AC7 と issue の受け入れ条件 5(同期の検査は通った。テスト 4 つは `/test`)
- Not verified: テストの実行、止まったスライスの再開を実際の `/implement` で使ったときの振る舞い、Codex の実行での agent の定義の読まれ方
- Findings: LOW 2 件。(1) `.codex/README.md:74` と `templates/base/.codex/README.md:74` の 5 項目の並びに「and」がなく、model-routing の「word for word」に合わない(直すなら「secrets, and files outside the slice」)。(2) `model-routing.md:51-55` の「removes the secret or narrows the slice」が、d33bb358 のあとの手順 7(秘密情報を先に出し、絞るのは必要なときだけ)より弱い(直すなら「removes the secret, narrows the slice if needed, and dispatches it again as a resume」、`templates/base/` も同じ。`tests/test-slice-verification.sh:264` の needle も合わせる)。どちらも振る舞いの契約の要は手順 7 と implementer の定義にあり、判定を変えない。ほかに情報 4 件(README の要約に再開がない、hook の文は tech-debt に記録済み、org 側は #262、Progress の verify の行の遅れ)。CRITICAL・HIGH・MEDIUM はない
- 判定: pass(HEAD 53326aff、cycle 2)
