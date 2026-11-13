# Test report: director-notice-handling

- Date: 2026-11-13(run-test.sh は 13:35:26 から 14:07:36 JST)
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(`## Test plan`、AC6 と AC7、issue #318)
- Tester: tester サブエージェント(`/test`、パイプラインの cycle 1。`.harness/state/standard-pipeline/cycle-count.json` が cycle 1)
- Scope: `git diff main...HEAD`(HEAD `fcd368b0`、base `bb416c48`)。本番の Go のコードは変わっていない。テストは新しい 2 ファイルに 4 本(`internal/org/prompts_notice_test.go` に 3 本、`internal/cli/director_notice_skill_test.go` に 1 本)。既存のテストファイルに削除行は無い。`./scripts/run-test.sh` は changed scope のまま golang だけを選んだ(`Language scope: changed (changed_languages)`、`Language packs selected: golang`)。変異は repo の外の写し(`git archive fcd368b0` を `$TMPDIR` に展開したもの)で流し、worktree の `git status --porcelain --ignored` は前後で同じだった
- Evidence: `docs/evidence/test-2026-11-13-director-notice-handling.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。run-test.sh の全出力の後ろに、runner と meta、名前を絞った 2 回の `-v` の結果、変異 7 つの出力を足した。wrapper 自身のログは `docs/evidence/verify-2026-11-13-043527.log`(名前の日付と時刻は UTC。先頭 5,600 行は上のログと同じ)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0) | shell 45 本 + Go 10 パッケージ | shell 45 本すべて OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0 | 2 行 | 1,930 秒 |
| └ Go: `internal/org` | — | ok | 0 | — | 1,067.4 秒 |
| └ Go: `internal/cli` | — | ok | 0 | — | 943.4 秒 |
| └ Go: `internal/config`、`internal/upgrade`、`internal/scaffold`、`internal/org/driver` | — | ok | 0 | — | 18.0、9.0、8.4、8.0 秒 |
| └ Go: `internal/org/protocol`、`internal/gitroot`、`internal/insights`、`cmd/ralph` | — | ok | 0 | — | 7.6、6.8、3.7、1.7 秒 |
| `go test -count=1 -v -run '^(TestDirectorSkill_\|TestRenderDirectorPrompt_\|TestRenderRolePrompt_Leader_)' ./internal/org/` | 上位 126 本(`TestDirectorSkill_` 37、`TestRenderDirectorPrompt_` 9、`TestRenderRolePrompt_Leader_` 80)、subtest を含めて 207 | 207 | 0 | 0 | 108.0 秒 |
| `go test -count=1 -v -run '^TestDirectorNoticeSkill_' ./internal/cli/` | 上位 1 本、subtest 4(skill の 4 写し) | 5 | 0 | 0 | 0.5 秒 |
| 変異 7 つ(下の「変異の確かめ」) | 7 | 期待どおり 5 つが落ち、隙間を示す 2 つは残った | — | — | 各 1〜10 秒、(b) だけ 2 分弱 |

run-test.sh の Go の段では、10 パッケージとも `(cached)` にならずに走った。`internal/org` は 30 分の上限の 59% を使った。1 分の load は開始時 10.46、走っているあいだ 7.86〜17.33、終了時 5.44 で、ほかのセッションも同じマシンを使っていた。変異の回は shell の段のあいだ(13:36〜13:44)に流し、Go の段(13:47 から 13:50 のあいだに始まり 14:07 まで)には何も重ねていない。名前を絞った 2 回は run-test.sh が終わったあとに流した。空き容量は開始時も終了後も 75 GiB だった。`internal/org` と `internal/cli` の時間は、同じ日の朝の upgrade-behavior-notes の回(722.6 秒と 551.5 秒)より長い。load が高めだったことによるものとみられ、この diff はテストを 4 本足しただけで、4 本とも 0.01 秒以内に終わる。

run-test.sh は Bash tool の 600 秒の上限を超えるので、runner(start、end、rc を meta に書く)を perl の fork と setsid で切り離して起動し(INT、HUP、QUIT、TERM は DEFAULT に戻した)、`caffeinate -w <pid>` を perl の `alarm 175` で区切った前景の待ちで終わりを待った。

Skipped の 2 行は前の PR の /test と同じで、`tests/test-plan-visual.sh` の browser detection(`/Applications` に browser がある手元の環境による)と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case(手元は git 2.49.0)である。どちらもこの diff と関係が無い。

## 受け入れ条件とテスト

| AC | テスト | 結果 |
| --- | --- | --- |
| AC6 skill の 4 写しの小節と禁止の項 | `TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` | PASS。変異 (a) と (c) で落ちる |
| AC6 headless の director の prompt(herdr と launch_cmd) | `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(`/herdr`、`/cmd`) | PASS |
| AC6 leader の prompt | `TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping` | PASS。変異 (b) で落ちる |
| AC7 2 つの行の決まった部分を skill の小節が引く | `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(4 写しの subtest) | PASS。skill の側の変異 (a) と、本番の文面の側の変異 p1、p2 で落ちる |
| Regression: 既存の `TestDirectorSkill_*`、`TestRenderDirectorPrompt_*`、`TestRenderRolePrompt_Leader_*` | 上の名前を絞った回(新しい 3 本を除いて上位 123 本) | すべて PASS。既存のテストは変わっていない |

## 変異の確かめ

どの変異も `$TMPDIR` の写しで当て、当てる前に元を残し、当てたあとに戻して `cmp` で一致を確かめた。skill は実行時に `runtime.Caller` から求めた repo の根で読まれるので、写しの中でビルドしたテストの binary を写しの中で走らせた。leader.md は `go:embed` なので、(b) は写しの leader.md を変えてから binary を作り直した。

| 変異 | 当てた所 | 結果 | 落ちた行 |
| --- | --- | --- | --- |
| (a) 2 つの決まった部分を消す | `templates/base/.agents/skills/director/SKILL.md` の小節の 1452 行の `run 'ralph upgrade' in ` と 1483 行の `; run 'ralph doctor' for ` | AC7 のテストが落ちる(その写しの subtest だけ FAIL、ほかの 3 写しは PASS)。skill のテストも落ちる | `internal/cli/director_notice_skill_test.go:100` が 2 回: `templates/base/.agents/skills/director/SKILL.md: the "### 人に向けた知らせの行" subsection does not quote "run 'ralph upgrade' in ", the fragment of the line scaffoldVersionNotice prints; ...` と `... does not quote "; run 'ralph doctor' for ", the fragment of the line codexHookTrustNotice prints; ...`。skill のテストは `internal/org/prompts_notice_test.go:129`(決まった文の 2 つ)と `:166`(CLI の断片の 2 つ) |
| (b) leader.md の hook の項を消す | `internal/org/prompts/leader.md` の 1335〜1359 行(25 行、運用規律の項) | leader のテストが落ちる。`-run 'TestRenderRolePrompt_'` の上位 125 本のうち落ちたのはこの 1 本だけ | `internal/org/prompts_notice_test.go:306`: ``運用規律 section has no top-level bullet starting "- `ralph org spawn` が stderr に、":`` |
| (c) 「してはいけないこと」の `ralph upgrade` の項を消す | `.claude/skills/director/SKILL.md` の 2039〜2041 行 | skill のテストが落ちる。AC7 のテストは通る(小節だけを読むので、禁止の項は見ない) | `internal/org/prompts_notice_test.go:129` が 3 回: ``.claude/skills/director/SKILL.md: the ralph upgrade bullet of ## してはいけないこと does not contain "`ralph upgrade` を打たない。雛形の版の `note:` の行が名指していても、人に頼まれても打たない" after the fragments before it ...``、同じ形で `codex の設定を書き換えず、codex を起動して hook を承認しない` と `(「進捗の報告」の「人に向けた知らせの行」)` |
| p1 本番の版の行の文面を変える | `scaffold_version_notice.go:61` の `run 'ralph upgrade' in %s` を `at %s` に(`-overlay`) | AC7 のテストが落ちる | `director_notice_skill_test.go:64`(行が断片を含まない)と `:74`(行が断片と絶対パスで終わらない) |
| p2 本番の hook の行の文面を変える | `codex_hook_trust_notice.go:80` の `; run 'ralph doctor' for %s` を `to see %s` に(`-overlay`) | AC7 のテストが落ちる | `director_notice_skill_test.go:64`(行が断片を含まない) |
| p3 hook の行の断片の外を変える | 同じ行の ` and how to %s` を ` and how %s` に | AC7 のテストは通る(残る)。既存の `CodexHookTrust` のテスト 8 本が落ちる | 下の G1 |
| p4 版の行の断片の外を変える | `scaffold_version_notice.go:60` の `older than this CLI (%s)` を `behind this CLI (%s)` に | AC7 のテストは通る(残る)。既存の `ScaffoldVersion` のテスト 10 本が落ちる | 下の G1 |

(a)〜(c) は handoff の 3 つで、どれも名指したテストが落ちた。p1 と p2 は AC7 の「文面が変わったら落ちる」を本番の側から確かめたもので、verify が読んで判断した所を実行で確かめた。p3 と p4 は G1 の隙間を示すために流した。

## Coverage

- Statement: 計測していない。本番の Go のコードが変わっていないため、Go の coverage の数字は変わらない
- Branch: 計測していない
- Function: 該当なし
- Notes: この diff が足したのは skill と prompt の文と、その文を読むテストである。テストが文をどこまで覆うかは、変異の表で確かめた。小節の決まった文(27 断片)、昇格の 2、出力先の 3、禁止の項、director の prompt の 6 断片、leader の prompt の 23 断片と止める規則の例外の一句をテストが読む。(a) は生成された写しの 1 つ、(c) は元の写しの 1 つに当て、どちらも当てた写しだけが落ちた。4 写しすべてに断片の変異を当てる確かめは S3 の implementer が済ませている(計画の Progress)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | 失敗なし | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| director の skill の既存の確かめ(`TestDirectorSkill_*` の既存 36 本) | 通る | 名前を絞った org の回 |
| director の prompt の既存の確かめ(`TestRenderDirectorPrompt_*` の既存 8 本。ralph org の命令に設定のフラグを付ける `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig` を含む) | 通る | 名前を絞った org の回。新しい項は `ralph org start` を名指さない書き方にしてあり、この確かめと食い違わない |
| leader の prompt の既存の確かめ(`TestRenderRolePrompt_Leader_*` の既存 79 本) | 通る | 名前を絞った org の回 |
| 2 つの知らせの文面(#313、#310) | 通る | run-test.sh の `internal/cli`(`(cached)` なし)。本番の Go は変わっていない |
| `internal/org` と `internal/cli` の全体 | 通る | run-test.sh の Go の段 |

## Test gaps

- G1(LOW): AC7 のテストが本番の行と結び付けるのは 2 つの断片だけで、小節の行の形のほかの部分は結び付けていない。小節は版の行を `note: <動詞>: the scaffold in <dir> (.ralph/manifest.toml) is ralph <雛形の版>, older than this CLI (<CLI の版>), ...`(1451 行)、hook の行の終わりを `...; run 'ralph doctor' for <the key(s)> and how to <直し方>`(1483 行)と書く。本番の `older than this CLI` を変える変異 p4 と、` and how to ` を変える変異 p3 は、AC7 のテストでも skill のテストでも落ちなかった。どちらも既存の `ScaffoldVersion`(10 本)と `CodexHookTrust`(8 本)のテストでは落ちるので、文面の変更が気づかれずに入ることはない。ただ、それらのテストの失敗文は skill に触れないので、直す人が skill の 4 写しを直し忘れうる。計画は 2 つの断片だけを結び付けると決めており(Test plan の Edge cases)、計画どおりの範囲である。閉じるなら、AC7 のテストで行の形の英語の部分(`older than this CLI (`、` and how to `)も断片に足す
- G2(LOW、self-review の L3 で受け入れ済み): AC7 のテストは hook の行が断片を含むことを `Contains` で確かめるだけで、断片で終わることまでは見ない。小節の終わりを切る補助は `#`、`##`、`###` の見出しで止まり、internal/org の補助と規則が違う。今は差が出ない
- G3(INFO): director と leader が実際の会話でこの文に従うかは、単体テストでは確かめられない(verify と同じ判断)
- G4(INFO): 変異 (c) では AC7 のテストが通った。AC7 のテストは小節だけを読み、禁止の項は skill のテストが受け持つので、役割の分け方どおりである

## Verdict

- Pass: run-test.sh rc 0(1,930 秒、13:35:26〜14:07:36 JST、1 分の load 5.44〜17.33)。shell 45 本すべて OK(PASS を含む行 5,195)、SKIP 2 行は手元の環境による。Go 10 パッケージすべて ok で `(cached)` なし: `internal/org` 1,067.4 秒、`internal/cli` 943.4 秒、`internal/config` 18.0 秒、`internal/upgrade` 9.0 秒、`internal/scaffold` 8.4 秒、`internal/org/driver` 8.0 秒、`internal/org/protocol` 7.6 秒、`internal/gitroot` 6.8 秒、`internal/insights` 3.7 秒、`cmd/ralph` 1.7 秒。名前を絞った回は `internal/org` が PASS 207・SKIP 0(上位 126 本)、`internal/cli` が PASS 5・SKIP 0。変異は handoff の (a)、(b)、(c) と本番の側の p1、p2 の 5 つがすべて落ちた
- Fail: 0
- Blocked: 0
- Verdict: pass。G1(行の形の断片の外を AC7 のテストが結び付けない。既存の文面のテストが本番の変更を捕まえるので LOW)と G2(受け入れ済みの L3)は、/pr の Known gaps に書くことを提案する
