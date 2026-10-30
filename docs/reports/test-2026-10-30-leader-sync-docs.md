# Test report: leader-sync-docs

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-leader-sync-docs.md`(`## Test plan` と AC1〜AC8。issue #268)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(merge-base 0351800d、HEAD 91e82463)。本番の Go のコードは変わっていない(`git diff origin/main...HEAD -- '*.go' ':!*_test.go'` は空)。変わったのは埋め込みの雛形 `internal/org/prompts/leader.md`、`/org` と `/director` の skill(写しを含めて 8 ファイル)、仕様、README、tech-debt と、新しいテスト 2 ファイル(`internal/org/prompts_leader_syncdocs_test.go` 10 本、`internal/org/skill_doc_followup_test.go` 16 本)。`./scripts/run-test.sh` は changed scope のまま打ち、detector は `changed_languages` で golang だけを選んだ。test mode なので `tests/test-*.sh` の 45 本もすべて走った
- Evidence: `docs/evidence/test-2026-10-30-leader-sync-docs.log`(gitignore。`run-test.sh` の全出力、絞った回の `-v` の出力、変異 12 個の差分とテストの出力、変更前の雛形と skill で新しいテストを打った回の出力。14,376 行)。wrapper 自身の写しは `docs/evidence/verify-2026-10-30-095405.log`(gitignore)

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

## Verdict

- Pass: `./scripts/run-test.sh` が rc 0(shell 45 本、Go 10 パッケージ)。AC7 の 5 ファイルは差分 0 行で 92 本とも通り、新しいテスト 26 本も通った。変異 12 個はすべて新しいテストで落ちた
- Fail: 0
- Blocked: 0
