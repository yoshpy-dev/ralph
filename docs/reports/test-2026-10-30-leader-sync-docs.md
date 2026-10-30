# Test report: leader-sync-docs

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-leader-sync-docs.md`(`## Test plan` と AC1〜AC8。issue #268)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1・2・3)
- Scope: `git diff origin/main...HEAD`(merge-base 0351800d、HEAD 91e82463)。本番の Go のコードは変わっていない(`git diff origin/main...HEAD -- '*.go' ':!*_test.go'` は空)。変わったのは埋め込みの雛形 `internal/org/prompts/leader.md`、`/org` と `/director` の skill(写しを含めて 8 ファイル)、仕様、README、tech-debt と、新しいテスト 2 ファイル(`internal/org/prompts_leader_syncdocs_test.go` 10 本、`internal/org/skill_doc_followup_test.go` 16 本)。`./scripts/run-test.sh` は changed scope のまま打ち、detector は `changed_languages` で golang だけを選んだ。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence: `docs/evidence/test-2026-10-30-leader-sync-docs.log`(gitignore。`run-test.sh` の全出力、絞った回の `-v` の出力、変異 12 個の差分とテストの出力、変更前の雛形と skill で新しいテストを打った回の出力。cycle 1 の分が 1〜14,376 行目、cycle 2 の分が 37,093 行目まで、cycle 3 の分がそのあとに続き、全体で 55,886 行)。wrapper 自身の写しは cycle 1 が `docs/evidence/verify-2026-10-30-095405.log`、cycle 2 が `docs/evidence/verify-2026-10-30-114133.log`、cycle 3 が `docs/evidence/verify-2026-10-30-124930.log`(どれも gitignore)

この下の「Test execution」から「Test gaps」までは cycle 1(HEAD 91e82463)の記録で、行番号もその時点のもの。cross-review cycle 1 の ACTION_REQUIRED を直したあとの cycle 2(HEAD b0845cd3)の test は、「Test gaps」のあとの「cycle 2(HEAD b0845cd3)」の節に書いた。cross-review cycle 2 の ACTION_REQUIRED を直したあとの cycle 3(HEAD d751a6e3)の test は、その次の「cycle 3(HEAD d751a6e3)」の節にある。最後の「Verdict」の節は先頭の行が cycle 3 の判定で、cycle 2 と cycle 1 の判定はその下に残している。

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS="-timeout=30m -count=1" ./scripts/run-test.sh`(changed scope、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 4,796)、Go 10 パッケージ ok | 0 | 1(`test-secret-scan-branch.sh` の「git 2.41 より古い実物の git」の行。手元は 2.49.0 で、古い git は stub の行が見る) | 2,019 秒(18:54:05〜19:27:44 JST)。開始時の load 11.05 / 13.34 / 11.54、終了時 61.22 / 91.50 / 58.48。途中の見本は 148.75(18:56)、13.71(19:04)、19.78(19:16)、116.99(19:27) |
| └ Go: `internal/org` | 1,353 本(トップレベル、`go test -list`) | ok | 0 | 0 | 1,102.6 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 620.9 秒 |
| └ Go: `internal/config`、`cmd/ralph`、`internal/{gitroot,insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 0.9〜51.9 秒 |
| 絞った回(変異なし): `go test -p 2 -count=1 -v ./internal/org/ -run '^(<7 ファイルのテスト関数 118 本>)$'` | 118(AC7 の 5 ファイルの 92 本 + 新しい 26 本)、subtest 100 | 118 | 0 | 0 | 18 秒(19:05、load 12.19) |
| 変異 12 個(下の「Regression checks」) | 12 | 12 個とも落ちた | — | — | 1 個 12〜29 秒 |
| 変更前の雛形と skill で新しいテストを打つ回 | 26 | — | 26 本とも落ちた | — | 雛形の回 8 秒。skill の回(build 済みの binary)は計っていない |

起動の仕方: `run-test.sh` は 600 秒を超えるので、Bash tool の `run_in_background` は使わず、perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力は /dev/null)。小さな runner が log と開始・終了・rc の meta を書き、こちらは前景の `caffeinate -w <pid>` で待った。shell の SKIP の行は前の回と同じ 1 行だけだった。`-count=1` は handoff の `GOFLAGS` に足したもので、`(cached)` の結果を使わないため。

負荷: 18:56 に 1 分の load が 148.75 まで上がった(そのとき CPU の上位には、ほかのセッションが数分前に起動した iOS シミュレーターのプロセスが並んでいた)。変異の回(18:57〜19:07)は `run-test.sh` の shell の段と重なったが、どちらも落ちなかった。Go の段(19:16 ごろから)のあいだは、こちらからほかのテストを打っていない。時間の条件で落ちたテストはなく、打ち直しはしていない。

### この diff に関わる shell のテスト

`scripts/verify.local.sh` の test mode は `tests/test-*.sh` を glob で全部打つので、写しの検査のテストも範囲に入っている。

| テスト | PASS の行 | 結果 | この diff との関わり |
| --- | --- | --- | --- |
| `tests/test-check-skill-sync.sh` | 14 | OK | `/org`・`/director` skill の写し |
| `tests/test-sync-skills.sh` | 23 | OK | 同上(写しの生成) |
| `tests/test-check-sync.sh` | 17 | OK | template の写しの同期の検査 |
| `tests/test-template-purity.sh` | 11 | OK | `templates/base/` の skill の文(verify の V1 で直した `check-sync.sh` の名指し) |
| `tests/test-check-template.sh` | 60 | OK | 新しい scaffold が通ること |

この表の 5 本は検査の script のテストで、この diff の写しそのものを比べるのは verify の `check-skill-sync.sh` と `check-sync.sh` の役目。Go の skill のテストは 4 つの写しをそれぞれ読む(下の変異で、1 つの写しだけを変えても落ちることを確かめた)。

## Coverage

- Statement: 計測していない。本番の Go のコードの変更がないので、statement coverage の数字は変わらない
- Branch: 同上
- Function: 同上
- Notes: 新しいテストは雛形と skill の文を固定するテストで、届き方は変異で確かめた(下の「Regression checks」)。コードを打つのは 3 か所だけで、`TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezeFindsTheFixedMessage` が雛形の `git log … | awk …` と `git show --name-only` を使い捨ての repo で走らせ、`TestRenderRolePrompt_Leader_FeatureOrgSyncDocsHandsTheDocMaintainerItsLimits` が `featureLeaderTask` の行と `reservePathsOverlap` の読み方を、`TestDirectorSkill_DocFollowupGatheringMarksAGatheredSplitPlanAndNeverGathersItAgain` が分割計画の読み込みと digest が印の行を残すことを確かめる

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (なし) | — | — | — |

## Regression checks

### AC7: 今の雛形と skill のテスト

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `prompts_test.go`・`prompts_verification_test.go`・`prompts_leader_plan_test.go`・`director_skill_sync_test.go`・`org_skill_send_sync_test.go` を変えずに通す | 変更 0 行、92 本すべて PASS | `git diff origin/main...HEAD -- <5 ファイル>` が 0 行(`--stat` も空)。絞った回で 92 本(43・18・17・11・3)が PASS、SKIP 0。`run-test.sh` の `internal/org` も ok |
| 手順の番号を振り直さない(手順 10 の 1 行目は `` `ralph org report`` のまま) | 固定されている | `TestRenderRolePrompt_Leader_FeatureOrgSyncDocsEndsStep9`(`internal/org/prompts_leader_syncdocs_test.go:112`)が手順 10 の 1 行目を行で確かめ、今の `prompts_test.go` の固定も通る |
| 手順 9 と 12 の今の文(レポートのコミット、fetch と rebase、gate の TASK)が残る | 固定されている | 同じテストの `:92` と `:102`、`…RunsAgainOnARedo` の `:476` |

### 新しいテストの感度(変異)

雛形の変異は worktree で `go test -p 2 -count=1 -overlay <json> ./internal/org/ -run 'RenderRolePrompt|Skill'` を打った(overlay は `internal/org/prompts/leader.md` を変異の写しに差し替える)。skill の変異は `git archive HEAD` の写し(scratchpad の下)で test binary を 1 回 build し、写しの SKILL.md を 1 つ変えて `-test.run 'RenderRolePrompt|Skill' -test.v` を打ち、終わるたびに HEAD と同じ中身に戻した(`cmp` で確かめた)。追跡しているファイルは変えていない。

| # | 変異 | 結果 | 落ちたテスト(行) |
| --- | --- | --- | --- |
| 1 | 手順 9 の sync-docs の 2 段落(347〜431 行)を手順 10 の終わり(`ralph org report` の段落のあと、11 の前)に移す | 落ちた(11 本) | 雛形の新しいテスト 10 本すべてと `TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate` |
| 2a | 「report のほかに、予約の外のパスか、凍らせたファイルか、機能の計画が 1 つでも変わっていれば、report のほかの sync-docs の変更をすべて戻す。」を消す | 落ちた(1 本) | `…SyncDocsUndoesEveryChangeOrNone`(`prompts_leader_syncdocs_test.go:205`) |
| 2b | 同じ文を一部だけ戻す形(「その予約の外のパスと計画の変更だけを戻す」)にする | 落ちた(1 本) | 同上 |
| 3 | 件名の完全一致の `git log --format='%H %s' … \| awk …` を `git log --format=%H --grep='^docs: sync docs for <機能の slug>$' …` に戻す | 落ちた(3 本) | `…FreezesEarlierSyncDocsFiles`(`:275`、否定の固定 `:292`)、`…FreezeFindsTheFixedMessage`(`:313`)、`TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate`(`skill_doc_followup_test.go:317`) |
| 3b | 3 に加えて、テストの定数 `syncDocsEarlierCommits` も `--grep` の形にし、否定の固定(`:291`)を `false &&` で止める(固定の文ではなく、コマンドを走らせる行だけが落とせるかを見る) | 落ちた(3 本) | `…FreezeFindsTheFixedMessage` の走らせる行(`:354`): `grep.patternType=basic` で 3 件(本文の行にメッセージがあるコミットを含む)が出た。ほかは skill との突き合わせの 2 本(skill は awk の形のまま) |
| 4 | 手順 14 の RESULT の例から `DOC_FOLLOWUP:` の 2 行を消す | 落ちた(1 本) | `…ResultAndPRBodyCarryTheDocFollowups`(`:540`、「got 0」) |
| 5a | `/director` の「予約外の文書の follow-up」の 1 を「どの機能も `merged` なら始める。`merged` でない機能…」にする(`closed` を落とす。`.claude/skills/director/SKILL.md` だけ) | 落ちた(2 本) | `TestDirectorSkill_DocFollowupGatheringStartsOnceEveryFeatureIsMergedOrClosed`(`skill_doc_followup_test.go:650`)、`…GathersTheDocFollowupsIntoADocsOnlyFeatureAfterEveryMerge`(`:525`) |
| 5a2 | 「件の処理」の RESULT の項の「`merged` か `closed` になったあとに」を「`merged` になったあとに」にする(同じ写し。handoff の外で足した) | 落ちた(1 本) | `TestDirectorSkill_ResultItemCarriesTheDocFollowupsAndANoteWithoutThem`(`:471`) |
| 5b | 6 の集めた印の文(「本文の先頭には、集めた印として `元の分割計画: <元の分割計画の id>` の 1 行を書く(1 で読む)。」)を消す(`templates/base/.agents/skills/director/SKILL.md` だけ) | 落ちた(1 本) | `…MarksAGatheredSplitPlanAndNeverGathersItAgain`(`:691`) |
| 5c | 1 の「もう集めた分割計画」の箇条を消す(`.agents/skills/director/SKILL.md` だけ) | 落ちた(1 本) | 同上 |
| O1 | `/org` の「org の終わり方」の 1 の戻し方を「予約の外のパスが変わっていれば、その変更だけを戻す」にする(`templates/base/.claude/skills/org/SKILL.md` だけ。handoff の外で足した) | 落ちた(1 本) | `TestOrgSkill_EndOfOrgStep1UndoesEveryChangeOrNoneAndCommitsOnce`(`:181`) |
| O2 | `/org` の 6 から `DOC_FOLLOWUP:` の文を消す(`.claude/skills/org/SKILL.md` だけ。handoff の外で足した) | 落ちた(2 本) | `TestOrgSkill_EndOfOrgStep6CarriesTheDocFollowups`(`:252`)、`TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate`(`:328`) |

どの回も build は通り(`build failed` は 0 件)、落ちたのは変異が触れた文を固定するテストだけだった。変異の回の load は 107.26(18:57)から 13.61(19:01)まで。

変更前の版でも確かめた。overlay で `origin/main` の `leader.md` に差し替えると、雛形の新しいテスト 10 本がすべて落ちた。写しの 8 つの SKILL.md を `origin/main` の版にした archive の写しでは、skill の新しいテスト 16 本がすべて落ちた。

変異のない木: worktree の絞った回で 118 本が PASS(SKIP 0)、archive の写しの基準の回も PASS、`run-test.sh` は rc 0。

## Test gaps

計画の `## Test plan` の項ごとに書く。

- Unit tests(雛形の文と順): テストがある。手順 9 の末尾の位置と順は `…SyncDocsEndsStep9`、doc-maintainer の Claude と Codex の呼び方と D2b の渡す項目と写しの組の扱いは `…HandsTheDocMaintainerItsLimits`、すべて戻すことと 1 つのコミットは `…UndoesEveryChangeOrNone`、飛ばすときの記録は `…SkipsWithoutTheDocMaintainer` と `…CountsANoReportEndAsASkip`、`DOC_FOLLOWUP:` の例と PR の本文の節は `…ResultAndPRBodyCarryTheDocFollowups`、`GATE: fail` の一文は `…RunsAgainOnARedo` が固定する
- Integration tests(skill の文と写しの一致): skill の文はテストがある(`skill_doc_followup_test.go` の 16 本。4 つの写しをそれぞれ読む)。写しどうしの一致は verify の `check-skill-sync.sh`・`check-sync.sh` の役目で、この回は検査の script のテストが通ったことだけを見た
- Regression tests: テストがある(上の AC7 の表)
- Edge case「doc-maintainer の定義がない」: 文の固定と、配る定義 4 つ(`templates/base/` と root の `.claude/agents/doc-maintainer.md`・`.codex/agents/doc-maintainer.toml`)がその名前で定義していることの確かめはある。飛ばす道を leader が実際にたどるかは打っていない
- Edge case「予約が `.`」: 読み方は `reservePathsOverlap(".", "docs/x.md")` の行が確かめる。doc-maintainer がその読み方で直す範囲を決めるかは打っていない
- Edge case「follow-up が 2,000 字に収まらない」: 文の固定と `protocol.DefaultMaxBodyChars == 2000` の確かめはある。件数の 1 行(`DOC_FOLLOWUP: 14 件(…)`)の形の RESULT を `validateEscalation` に通す行はない(通しているのは 2 行の例だけ)。足すなら `…ResultAndPRBodyCarryTheDocFollowups` に、例の 2 行を件数の 1 行に置き換えた本文を通す行を 1 つ
- Edge case「gate のやり直しで sync-docs が 2 回目に何も変えない」: 文の固定(「コミットせずに手順 10 に進む」)だけで、コマンドで確かめる行はない
- 戻し方のコマンド(`git restore --source=HEAD --staged --worktree -- <パス>` と未追跡の削除)は、implementer が使い捨ての repo で確かめた記録が計画にあるだけで、自動のテストはない。走らせているのは凍らせるファイルを引く `git log … | awk …` と `git show --name-only` だけ
- `git show --name-only` が `core.quotePath`(ASCII でないパスを引用符で囲む)と `log.showSignature` で変わりうることは未確認で、tech-debt の行になっている。awk は手元では macOS の `/usr/bin/awk` だけで走った。CI(`.github/workflows/verify.yml`、ubuntu-latest の `run-verify.sh`)では別の awk で同じテストが走るはずだが、どの awk かは確かめていない
- leader と director は prompt に従う LLM なので、動きそのもの(doc-maintainer を呼ぶ、戻す、コミットする、follow-up を集めて分割計画を人と書く)は実機の org でしか確かめられない。この回は打っていない。Codex の leader が pane の中で custom agent を呼べるかも未確認(計画のリスクの項)

## cycle 2(HEAD b0845cd3)

cross-review cycle 1 の ACTION_REQUIRED の直しのあとの回。a5e28500(cycle 1 の test のコミット)からのテストに関わる変更は、雛形と `/org` skill の sync-docs の段落の 3 つ。F5(4d0f70e4)は git status を `git status --porcelain --untracked-files=all` にそろえ、未追跡のファイルを 1 つずつ消す形にした。F5b(aec1381f)は doc-maintainer を呼ぶ前に何も出ないことを確かめる文を足し、F6(41468881)はその文の implementer への TASK に per-plan の `PLAN:` のヘッダを足した。新しいテストは 3 本増えて 29 本になった(`prompts_leader_syncdocs_test.go` 12 本、`skill_doc_followup_test.go` 17 本)。増えたのは `TestRenderRolePrompt_Leader_FeatureOrgSyncDocsStatusListsEachNewFile`、`…SyncDocsChecksForLeftoversBeforeTheCall`、`TestOrgSkill_EndOfOrgStep1ChecksForLeftoversBeforeTheCall`。本番の Go のコードは今回も変わっていない。

### Test execution(cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS="-timeout=30m -count=1" ./scripts/run-test.sh`(changed scope、golang、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 4,796)、Go 10 パッケージ ok | 0 | 1(cycle 1 と同じ git 2.41 の行) | 1,685 秒(20:41:32〜21:09:37 JST)。開始時の load 7.31 / 8.15 / 8.85、終了時 9.77 / 9.55 / 9.01。途中の見本は 10.83(20:43)、7.95(21:03) |
| └ Go: `internal/org` | 1,356 本(トップレベル、`go test -list`) | ok | 0 | 0 | 933.5 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 645.0 秒 |
| └ Go: `internal/config`、`internal/gitroot` | — | ok | 0 | 0 | 86.1 秒、57.7 秒 |
| └ Go: `cmd/ralph`、`internal/{insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 0.8〜3.6 秒 |
| 絞った回(変異なし): 7 ファイルのテスト関数を `-v` で | 121(AC7 の 5 ファイルの 92 本 + 新しい 29 本)、subtest 100 | 121 | 0 | 0 | 15 秒(20:53、load 7.27) |
| 変異 21 個(下の表) | 21 | 21 個とも落ちた | — | — | 雛形 22〜28 秒、skill 11〜18 秒。load 6.69〜12.43 |

起動の仕方は cycle 1 と同じ(perl の fork と setsid で切り離し、前景の `caffeinate -w` で待った)。変異と確かめの回(20:44〜20:53)は `run-test.sh` の shell の段と重なった。Go の段に入った時刻は記録しておらず、最後の絞った回(20:53、15 秒)がその始まりと重なったかどうかは分からない。落ちたテストはなく、打ち直しはしていない。

AC7: `git diff origin/main...HEAD -- <5 ファイル>` は今回も 0 行で、5 ファイルの 92 本は絞った回で PASS した。

### 変異(cycle 2)

やり方は cycle 1 と同じ。雛形は worktree で `-overlay`、skill は HEAD b0845cd3 の `git archive` の写しで binary を 1 回 build して、写しの SKILL.md を 1 つずつ変えた。どの回も build は通り、写しは終わるたびに HEAD と `cmp` で同じに戻した。cycle 1 の 12 個は今の HEAD の文に当て直した。

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| 1 | sync-docs の段落を手順 10 の終わりに移す | 落ちた(13 本) | 雛形の新しいテスト 12 本すべてと `TestOrgSkill_EndOfOrgNamesTheSyncDocsLikeTheLeaderTemplate` |
| 2a / 2b | すべて戻す文を消す / 一部だけ戻す形にする | どちらも落ちた(1 本) | `…SyncDocsUndoesEveryChangeOrNone` |
| 3 | `--grep` の形に戻す | 落ちた(4 本) | cycle 1 の 3 本と `…SyncDocsChecksForLeftoversBeforeTheCall`(呼ぶ前の確かめの文から `git log` の文までを続けて固定している) |
| 3b | 3 に加えてテストの定数も `--grep` の形にし、否定の固定を止める | 落ちた(4 本) | `…FreezeFindsTheFixedMessage` の走らせる行(`prompts_leader_syncdocs_test.go:496`)が `grep.patternType=basic` で 3 件を拾った。ほかは skill との突き合わせの 3 本 |
| 4 | RESULT の例から `DOC_FOLLOWUP:` の 2 行を消す | 落ちた(1 本) | `…ResultAndPRBodyCarryTheDocFollowups` |
| 6a | 失敗の枝の「`git status …` に出るパス」(359 行)を短い `git status --porcelain` に戻す | 落ちた(3 本) | `…StatusListsEachNewFile`(`:306`、手渡しの文のあとに短い形がないことの確かめ)、`…CountsANoReportEndAsASkip`、`…ChecksForLeftoversBeforeTheCall` |
| 6b | 失敗の枝の「何も出さないことを確かめてから」(365 行)を短い形に戻す | 落ちた(3 本) | 6a と同じ 3 本 |
| 6c | 呼ぶ前の確かめ(369 行)を短い形に戻す | 落ちた(2 本) | `…StatusListsEachNewFile`(`:306`)、`…ChecksForLeftoversBeforeTheCall` |
| 6d | doc-maintainer のあとの確かめ(411 行)を短い形に戻す | 落ちた(9 本) | `syncDocsCheck` の文を固定する 9 本(`…StatusListsEachNewFile` の `:306` を含む) |
| 6e | コミットのあとの確かめ(439 行)を短い形に戻す | 落ちた(2 本) | `…StatusListsEachNewFile`(`:306`)、`…UndoesEveryChangeOrNone`(`:255`) |
| 7 | 呼ぶ前の確かめの 2 文(369〜377 行)を消す | 落ちた(1 本) | `…ChecksForLeftoversBeforeTheCall`(`:610`) |
| 8 | 呼ぶ前の確かめの「(per-plan では `PLAN:` のヘッダを付ける)」を消す(F6 の分。handoff の外で足した) | 落ちた(1 本) | 同上(`:610`) |
| 5a / 5a2 / 5b / 5c | director の `closed` を落とす(1 と「件の処理」の RESULT の項)、集めた印の行を消す、「もう集めた分割計画」の箇条を消す | どれも落ちた | cycle 1 と同じテスト(5a は 2 本、ほかは 1 本ずつ) |
| O1 / O2 | `/org` の 1 を一部だけ戻す形にする / 6 から `DOC_FOLLOWUP:` の文を消す | どちらも落ちた | cycle 1 と同じテスト(O1 は 1 本、O2 は 2 本) |
| O3 | `/org` の 1 のコミットのあとの確かめを短い `git status --porcelain` に戻す(`.agents/skills/org/SKILL.md` だけ。handoff の外で足した) | 落ちた(1 本) | `TestOrgSkill_EndOfOrgStep1UndoesEveryChangeOrNoneAndCommitsOnce`(`skill_doc_followup_test.go:185` と、短い形がないことの確かめ `:224`) |
| O4 | `/org` の 1 の呼ぶ前の確かめの 2 文を消す(`templates/base/.agents/skills/org/SKILL.md` だけ。handoff の外で足した) | 落ちた(1 本) | `TestOrgSkill_EndOfOrgStep1ChecksForLeftoversBeforeTheCall`(`:411`) |

雛形の 5 か所の `git status --porcelain --untracked-files=all` は、どの 1 か所を短い形に戻しても `…StatusListsEachNewFile` の否定の確かめ(`:306`)で落ちる。このテストは手渡しの文のあとの手順 9 全体に短い形がないことを見るので、今後この範囲に git status を足したときも同じ確かめに掛かる。

変更前の版でも確かめた。新しいテストを `origin/main` の雛形と skill で打つと、29 本(12 本と 17 本)がすべて落ちた。cycle 1 のコミット a5e28500 の版で打つと、雛形のテストは 9 本が落ちて 3 本が通り、skill のテストは 7 本が落ちて 10 本が通った。落ちた側には増えた 3 本が入っている。通った 13 本は、F5・F5b・F6 が変えていない文(director の skill の全部、`/org` の fan-out と 6、`FreezeFindsTheFixedMessage` など)を固定するテストだった。

### Test gaps(cycle 2 で変わったこと)

- 縮んだ穴: 未追跡のファイルの扱いを、テストが使い捨ての repo で確かめるようになった。`…StatusListsEachNewFile` は、新しいディレクトリの下の予約したファイル、予約していないファイル、ignore されたファイルを置く。`status.showUntrackedFiles` が normal のときと no のときの両方で、雛形の形が 2 つのファイルを 1 行ずつ出し、短い形はディレクトリの 1 行しか出さないか何も出さないことを確かめる。一覧のファイルを 1 つずつ消すと何も出なくなり、ignore されたファイルが残ることも確かめる。消すのは Go の `os.Remove` で、leader が打つ shell のコマンドではない
- 残る穴: 呼ぶ前の確かめで何か出たときの振り分け(reviewer のレポートのコミット、implementer への TASK、BLOCKED)は文の固定だけ。leader の動きなので、実機の org でないと確かめられない
- 残る穴: `git restore --source=HEAD --staged --worktree -- <パス>` を走らせるテストはまだない
- 確かめの範囲について: 雛形と skill の呼ぶ前の確かめの文は、implementer の項の途中まで同じ文で突き合わせている。そのあとは写しごとの言い方で、雛形は「手順 8 の往復に戻り」、skill は「reviewer にレビューを頼み直し」と書く。この違いはテストが見ない(テストのコメントに書いてある設計どおり)
- cycle 1 の残りの穴(件数の 1 行の RESULT、2 回目の sync-docs で何も変わらないとき、`core.quotePath`、awk を BSD の awk だけで走らせたこと、実機の org での動き、Codex の leader の custom agent)は変わっていない

### 判定(cycle 2)

pass。`run-test.sh` が rc 0、AC7 の 5 ファイルは差分 0 行、新しいテスト 29 本を含む 121 本が通り、変異 21 個はすべて落ちた。

## cycle 3(HEAD d751a6e3)

cross-review cycle 2 の ACTION_REQUIRED の直しのあとの回(上限を 3 に上げた回し直し)。147e84bc(cycle 2 の test のコミット)からのテストに関わる変更は、`/director` skill の「予約外の文書の follow-up」の手順 6 だけ。F7(44694f5b)は、案を見せる前に follow-up の文書を読むテストと、文を固定するテストと fixture を探して `- Reserve:` に入れ、人に言う文を足した。F8(b6a8060a)はその文を言い直した(「一覧のファイルを読むか、」を「一覧のファイルを読むテストと、」に)。新しいテストは `TestDirectorSkill_DocFollowupGatheringReservesTheTestsThatPinTheDocs` の 1 本で、合わせて 30 本(`prompts_leader_syncdocs_test.go` 12 本、`skill_doc_followup_test.go` 18 本)。雛形、`/org` skill、`prompts_leader_syncdocs_test.go` は cycle 2 から変わっていない(`git diff --quiet 147e84bc HEAD -- <それらのファイル>` が通る)。本番の Go のコードも今回も変わっていない。

### Test execution(cycle 3)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS="-timeout=30m -count=1" ./scripts/run-test.sh`(changed scope、golang、rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 4,796)、Go 10 パッケージ ok | 0 | 1(cycle 1・2 と同じ git 2.41 の行) | 1,762 秒(21:49:29〜22:18:51 JST)。開始時の load 14.28 / 14.08 / 11.19、終了時 15.58 / 30.87 / 30.66。途中の見本は 22.16(22:01)、13.24 / 29.24 / 29.13(22:12) |
| └ Go: `internal/org` | 1,357 本(トップレベル、`go test -list`) | ok | 0 | 0 | 776.6 秒 |
| └ Go: `internal/cli` | — | ok | 0 | 0 | 580.1 秒 |
| └ Go: `internal/config`、`internal/gitroot` | — | ok | 0 | 0 | 65.0 秒、24.1 秒 |
| └ Go: `cmd/ralph`、`internal/{insights,org/driver,org/protocol,scaffold,upgrade}` | — | ok | 0 | 0 | 0.7〜4.8 秒 |
| 絞った回(変異なし): 7 ファイルのテスト関数を `-v` で | 122(AC7 の 5 ファイルの 92 本 + 新しい 30 本) | 122 | 0 | 0 | 33 秒(22:01、load 22.16) |
| 変異 27 個(当て直し 21 個、手順 6 の 6 個。下の表) | 27 | 26 個が落ち、1 個(handoff の外で足した D6f)が残った | — | — | 雛形 23〜34 秒、skill 8〜46 秒。load 8.56〜19.47 |

起動の仕方は cycle 1・2 と同じ。変異と確かめの回(21:50〜22:02)は `run-test.sh` の shell の段と重なった(22:00 の時点で log は shell の段の途中だった)。落ちたテストはなく、打ち直しはしていない。

AC7: `git diff origin/main...HEAD -- <5 ファイル>` は今回も 0 行で、5 ファイルの 92 本は絞った回で PASS した。

### 変異(cycle 3)

当て直し: cycle 2 の 21 個を今の HEAD に当てた。雛形の 13 個は、雛形と `prompts_leader_syncdocs_test.go` が cycle 2 から変わっていないので、cycle 2 の overlay をそのまま使った。skill の 8 個は HEAD d751a6e3 の `git archive` の写しで当てた。D5b の文は手順 6 の折り返しが変わったので、今の改行の位置に合わせた。21 個とも落ち、落ちたテストの組は cycle 2 と同じだった(1 個ずつ比べた)。

手順 6 の新しい変異:

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| D6a | 手順 6 からテストと fixture を予約する文(「案を見せる前に、…gate のテストが落ちる)。」と「足したテストと fixture は、…人に言う。」)を消す(`.claude/skills/director/SKILL.md` だけ) | 落ちた | `TestDirectorSkill_DocFollowupGatheringReservesTheTestsThatPinTheDocs`(`skill_doc_followup_test.go:706`) |
| D6b | F8 より前の「一覧のファイルを読むか、その文を固定するテストと fixture を探し」の形に戻す(`templates/base/.claude/skills/director/SKILL.md` だけ) | 落ちた | 同上(`:706`) |
| D6c | 「足したテストと fixture は、どれをなぜ足したかを案と一緒に人に言う。」だけを消す(`.agents/skills/director/SKILL.md` だけ。handoff の外で足した) | 落ちた | 同上(`:706`) |
| D6d | 検索の例の括弧の中に `internal/org/skill_doc_followup_test.go` を足す(`templates/base/.agents/skills/director/SKILL.md` だけ。handoff の外で足した) | 落ちた | 同上(`:706` と、テストのファイルを名指さないことの確かめ `:723`) |
| D6e | 手順 6 の終わりに「テストの例は `docs_test.go`。」を足すだけ(固定した文は壊さない。handoff の外で足した) | 落ちた | 同上(`:723` だけ) |
| D6f | 手順 6 の終わりに「テストの例は `tests/test-check-sync.sh` と `internal/org/testdata/`。」を足すだけ(handoff の外で足した) | 残った | なし(rc 0) |

新しいテストは、変更前の版の 3 つでも落ちた。`origin/main`、cycle 2 の 147e84bc、F7 だけの 44694f5b の director の skill を archive の写しに書いて打ったところ、どれも FAIL だった。

### Test gaps(cycle 3 で変わったこと)

- 新しい穴(LOW): 手順 6 がこの repo のテストを名指さないことの確かめ(`skill_doc_followup_test.go:723`)は `_test.go` の文字だけを見る。D6f のとおり、shell のテスト(`tests/test-*.sh`)や `testdata/` の名前を足しても通る。足すなら、同じ否定の確かめに `tests/test-` と `testdata/` を加える 1 行が一番小さい
- 残る穴: director が実際にテストと fixture を探し当てて予約に入れるかは、director の動き(LLM)なので、実機でないと確かめられない。テストが固定するのは手順の文だけ
- cycle 2 までの残りの穴は変わっていない

## Verdict

- Pass: cycle 3(HEAD d751a6e3)。`./scripts/run-test.sh` が rc 0(shell 45 本、Go 10 パッケージ)。AC7 の 5 ファイルは差分 0 行で 92 本とも通り、新しいテスト 30 本も通った。当て直しの 21 個と、頼まれた手順 6 の 2 個を含む変異 26 個が落ちた。handoff の外で足した D6f だけが残り、Test gaps に書いた
- Pass(cycle 2、HEAD b0845cd3)。`./scripts/run-test.sh` が rc 0(shell 45 本、Go 10 パッケージ)。AC7 の 5 ファイルは差分 0 行で 92 本とも通り、新しいテスト 29 本も通った。変異 21 個はすべて新しいテストで落ちた
- Pass(cycle 1、HEAD 91e82463): `./scripts/run-test.sh` が rc 0(shell 45 本、Go 10 パッケージ)。AC7 の 5 ファイルは差分 0 行で 92 本とも通り、新しいテスト 26 本も通った。変異 12 個はすべて新しいテストで落ちた
- Fail: 0
- Blocked: 0
