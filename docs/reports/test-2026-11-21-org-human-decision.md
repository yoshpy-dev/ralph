# Test report: org-human-decision

- Date: 2026-11-22 JST(cycle 2 の test。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 2`。cycle 1 の run 1 は、下の「Run 1」に、見出しの階層を 1 つ下げた以外は変えずに残す)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366、AC1〜AC7)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b337156e`。worktree は clean。1 周目の test(HEAD `b97a336c`)のあとに Go が変わった commit は 4 つある。`7e8ac5b1`(origin/main の `9cd63d60` の取り込み。#367 の `internal/org/gate_lock.go` とそのテスト)、`d78f379a`(テストの 2 ファイル)、`6dc7c22f`(V8: `SendResult.HumanOrigin`、人の決定の send が失敗したときの note、`ralph org send --help` の 1 文)、`a9b160eb`(R3: その note を「しないこと」と規則の節の名指しに絞った)。`./scripts/run-test.sh` はこの worktree で 1 回だけ実行した。起動の前に `pgrep -f 'scripts/run-test.sh'` で調べ、当たった 1 件は別の repo(bakeroku)の座席が送るメッセージの文で、その中身は `run-static-verify.sh` だった。この repo の run-test.sh と run-verify.sh は動いていなかった。起動の形は 1 周目と同じ(perl の fork と setsid で切り離し、INT/HUP/QUIT/TERM を DEFAULT に戻し、標準入出力を `/dev/null` に向け、出力はファイル)。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。そのあと、AC に絞った `-v` の run を 2 本、変異を 12 個(`go test -overlay` で 8 個、`git archive HEAD` の写しで 4 個)、絞った run の coverage を取った。追跡しているファイルは何も変えていない(`git status --porcelain --ignored` に出るのは、無視されている `.harness/` と evidence の log だけ)
- Evidence: `docs/evidence/test-2026-11-21-org-human-decision-run2.log`(7,246 行。開始と終了の時刻、30 秒ごとの load と空き容量、run-test.sh の出力 5,829 行、`(cached)` だったパッケージの打ち直し、AC の run、変異ごとの diff と出力、coverage)。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-21-194624.log` にも書いた(同じく追跡外)。1 周目の log は書き換えていない
- 環境: go1.27.3 darwin/arm64、git 2.49.0

## Run 2 (cycle 2)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(1 周目と同じ条件つきの skip) | 2026-11-22 04:46:23 〜 05:16:58 JST、1,835 s、rc 0(`==> All verifiers passed.`)。1 分 load は開始 8.82、終了 10.53、30 秒ごとの観測で 8.08〜32.00。空き容量は 77〜80 GiB |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合 | 約 15 分 |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし | 約 16 分、下の表 |
| `(cached)` だった 8 パッケージの打ち直し: `go test -count=1 ./cmd/ralph/ ./internal/gitroot/ ./internal/insights/ ./internal/org/driver/ ./internal/org/protocol/ ./internal/scaffold/ ./internal/toolchain/ ./internal/upgrade/` | 8 | 8 | 0 | なし | 20 s |
| AC run cli: `go test -count=1 -v ./internal/cli/ -run 'OrgSendHumanDecision\|OrgSend_HumanDecision\|OrgSend_HelpDescribesTheOriginHeader\|OrgSend_Origin\|OrgSend_AppendEventFailsAfterEnter\|OrgSend_CtxExpiresDuringEnterDelay\|OrgSend_.*Note'` | 関数 12、subtest 12 | 全部 | 0 | 0 | 15 s |
| AC run org: `go test -count=1 -v ./internal/org/ -run 'SendOrigin\|HumanDecision\|DirectorSkill\|SendTextTypedNotSubmitted\|AgentMessagingRule\|ReseatProgress\|RenderRolePrompt_Leader\|RenderDirectorPrompt'` | 関数 208、subtest 229 | 全部 | 0 | 0 | 82 s |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok(`(cached)`、打ち直しも ok) | 0.700 s(打ち直し) |
| `internal/cli` | ok | 907.142 s |
| `internal/config` | ok | 15.227 s |
| `internal/gitroot` | ok(`(cached)`、打ち直しも ok) | 2.468 s(打ち直し) |
| `internal/insights` | ok(`(cached)`、打ち直しも ok) | 1.714 s(打ち直し) |
| `internal/org` | ok | 914.495 s |
| `internal/org/driver` | ok(`(cached)`、打ち直しも ok) | 2.380 s(打ち直し) |
| `internal/org/protocol` | ok(`(cached)`、打ち直しも ok) | 2.478 s(打ち直し) |
| `internal/scaffold` | ok(`(cached)`、打ち直しも ok) | 2.882 s(打ち直し) |
| `internal/toolchain` | ok(`(cached)`、打ち直しも ok) | 18.615 s(打ち直し) |
| `internal/upgrade` | ok(`(cached)`、打ち直しも ok) | 3.653 s(打ち直し) |

golang pack の `go test` は `-count=1` を付けないので、入力が前の run と同じパッケージは `(cached)` になる。1 周目は 0 だったが、今回は 8 パッケージが `(cached)` だったので、`-count=1` で打ち直してすべて通った。Go が変わった `internal/cli` と `internal/org`、文書を読む `internal/config` は実際に走った。`internal/org` と `internal/cli` は 1 パッケージの上限(30 分)の約 51% で終わった。run-test.sh の実行中に、この report の AC の run、変異、coverage の run(04:47〜05:04)が重なった。多くは shell の段だが、終わりの数分は Go の段の始まりにかかった。別の repo の `run-static-verify.sh` も起動の時点で動いていた。落ちたテストはなく、打ち直したテストもない(`(cached)` の打ち直しは落ちたからではない)。既知の flake の `TestGate_TwoGatesRunTheirScriptsOneAtATime` も、`7e8ac5b1` で入った `TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty` も通った。

#### 1 周目のあとに足したテストと変えたテスト

| テスト | 種類 | 確かめること |
| --- | --- | --- |
| `TestOrgSendOrigin_SendResultHumanOrigin`(org、12 subtest) | 新規 | `SendResult.HumanOrigin` は、ORIGIN の確かめを通った人の決定では送れたとき、dry run、send-text の失敗、Enter の前の待ちの時間切れ、send-keys の失敗、leader がいないときのどれでも true。ORIGIN のない DECISION と、ORIGIN・貼り付けマーカー・protocol の確かめで断られたときは false |
| `TestOrgSendHumanDecisionFailureNote_Text`(cli、3 subtest) | 新規 | 入力欄に文が残りうる 3 つの状態の note の全文。`submit` と `clear the box` を含まない。規則の節の名前を含む。`SendProgressNothingSent` と `SendProgressEnterPressed` には note を返さない |
| `TestOrgSendHumanDecisionNote_NamesTheRuleSectionAndTheSkillStep`(cli、6 subtest) | 新規 | note が名指す規則の節の見出しが `agent-messaging.md` の 2 写しにあり、`### 人の決定を送る` が director skill の 4 写しにある |
| `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote`(cli、3 subtest) | 新規 | fake の herdr で 3 つの失敗を起こし、人の決定には専用の note が 1 行だけ出る。同じ leader への ORIGIN のない DECISION には今までの note が出る |
| `TestOrgSend_HumanDecisionEnterPressedNotRecorded_NoteUnchanged`(cli) | 新規 | Enter は押され `sent` だけ書けなかったときは、人の決定でも今までの「送り直さない」note のまま |
| `TestDirectorSkill_HumanDecisionQuotesTheSendNotes`(org) | 新規 | director skill が引く note の語が、`internal/cli/org.go` の該当の case の文字列にまだある |
| `TestAgentMessagingRule_OriginHumanIsRecordedOnlyByASendThatExits0`(org) | 新規 | 規則の 2 写しの `ORIGIN: human` の節に、終了コード 0 の send だけが記録すること、Enter で回復しないこと、空の入力欄にだけ送り直すことの文がある |
| `TestOrgSend_HelpDescribesTheOriginHeader`(cli) | 1 文を追加 | help の「Only a send that exits 0 records the decision」の文 |
| `TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen`(org) | 3 句を追加 | headless director の prompt が、人の決定にはこの Enter を押さないと言う |
| `TestRenderRolePrompt_Leader_HumanDecisionBulletSaysWhatTheLeaderDoes`(org) | 3 句を追加 | 全文が同じ人の決定は記録のための送り直しで、同じ全文の別の決定かもしれないときは QUESTION を上げる |
| `TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader`(org) | 句と場所を追加 | `### 答える順` の Enter が人の決定には使えないこと、`### 人の決定を送る` の 2 の項の順と中身 |

新しい 7 関数と文を足した 4 関数は、AC run と run-test.sh の両方で通った。

### 変異

1 周目のあとに変わった本番のコードと、それを固定するはずのテストの組を確かめた。N1〜N9 は `go test -overlay`(N8 と N9 は `//go:embed` の prompt を overlay で差し替え)。N10〜N13 はテストが disk から読むファイルなので、`git archive HEAD` の写しで 1 個ずつ書き換えて走らせ、元に戻して `cmp` で一致を確かめた。

| # | 変異 | run | 結果 |
| --- | --- | --- | --- |
| N1 | `verbs.go` の `defer func() { result.HumanOrigin = humanOrigin }()` を消す | org `-run 'SendOrigin'`(9 s)、cli `-run 'OrgSend'`(87 s) | 検出。org で `TestOrgSendOrigin_SendResultHumanOrigin` の人の決定の 6 subtest、cli で `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の 3 subtest |
| N2 | 同じ defer で常に `true` を入れる | 同上(10 s、83 s) | 検出。org で ORIGIN のない DECISION の 2 subtest。cli で `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の 3 subtest と、既存の `TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr`・`TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr` |
| N4 | `org.go` の `if result.HumanOrigin {` を `if false {` にする(配線を切る) | cli `-run 'OrgSend'`(90 s) | 検出。`TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の 3 subtest |
| N5 | 同じ行を `if true {` にする | 同上(102 s) | 検出。同じ 3 subtest(ORIGIN のない DECISION に人の決定の note が出る)と、既存の note のテスト 2 本 |
| N6 | `orgSendHumanDecisionFailureNote` から `SendProgressEnterUnacknowledged` の case を外す(どの値にも当たらない case にする) | 同上(117 s) | 検出。`TestOrgSendHumanDecisionFailureNote_Text/enter-unacknowledged` と `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote/Enter_is_not_acknowledged` |
| N7 | 人の決定の note を出したあとの `return result.Err` を消す(今までの note も続けて出る) | 同上(92 s) | 検出。`TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の 3 subtest(`note:` の行が 2 本)。`_Text` のテストは関数だけを見るので通る |
| N8 | `prompts/leader.md` から「(済んだ手は繰り返しません)」を消す | org `-run 'RenderRolePrompt_Leader\|ReseatProgressLead'`(85 s) | 検出。`TestRenderRolePrompt_Leader_HumanDecisionBulletSaysWhatTheLeaderDoes` |
| N9 | `prompts/director.md` の「この Enter を押さない」を「押す」にする | org `-run 'RenderDirectorPrompt\|DirectorSkill'`(13 s) | 検出。`TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen` の herdr と cmd の 2 subtest |
| N10 | 雛形の `agent-messaging.md` の見出しを `## A human decision (...)` にする(写しの 1 つだけ) | 写しで cli `-run 'OrgSendHumanDecisionNote'`(7 s) | 検出。`TestOrgSendHumanDecisionNote_NamesTheRuleSectionAndTheSkillStep/templates/base/.claude/rules/ralph/agent-messaging.md` |
| N11 | `internal/cli/org.go` の `SendProgressEnterUnacknowledged` の note の「do not send the message again」を「do not resend the message」にする | 写しで org `-run 'DirectorSkill_HumanDecisionQuotesTheSendNotes'`(8 s) | 検出。`TestDirectorSkill_HumanDecisionQuotesTheSendNotes` |
| N12 | 雛形の `agent-messaging.md` から「Only a send that exits 0 records the decision:」を消す | 写しで org `-run 'AgentMessagingRule'` | 検出。`TestAgentMessagingRule_OriginHumanIsRecordedOnlyByASendThatExits0` |
| N13 | 雛形の `.agents/skills/director/SKILL.md` の `### 人の決定を送る` を `### 人の決定の送り方` にする | 写しで cli `-run 'OrgSendHumanDecisionNote'`(1 s) | 検出。`TestOrgSendHumanDecisionNote_NamesTheRuleSectionAndTheSkillStep/templates/base/.agents/skills/director/SKILL.md` |

12 個すべてが検出された(番号の N3 は使っていない)。ビルドが落ちた変異はなく、写しの run に SKIP は出ていない(`readRepoText` はファイルがないと `t.Skipf` するので確かめた)。

### Coverage

- Statement: 絞った run(`go test -count=1 -coverprofile … ./internal/cli/ -run 'OrgSend'`、88 s と `./internal/org/ -run 'SendOrigin'`、3 s)の profile で、1 周目のあとに足した本番の行のうち文の block に入る行は、すべて実行された。`internal/cli/org.go` は追加 77 行のうち block に入る 21 行、`internal/org/verbs.go` は追加 24 行のうち block に入る 1 行(defer の行。残りはコメントと struct の欄)。`git diff -U0 b97a336c..HEAD` の追加行と profile の block の突き合わせ。関数ごとには `orgSendHumanDecisionFailureNote` が 100%、`newOrgSendCmd` が 92.7%
- Branch: 計測していない(Go の標準の道具にない)
- Function: 絞った run なので、パッケージ全体の数字は出していない
- Notes: `7e8ac5b1` で入った `gate_lock.go` の変更は #367 のもので、この branch の変更ではない。run-test.sh で通ったことだけを見た

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 人の決定でないメッセージの失敗の note(send-text の失敗、Enter の前の待ちの時間切れ、send-keys の失敗、`sent` の書き込みの失敗) | 変わらず通る | `TestOrgSend_PaneSendTextFails_NotesTextUnacknowledgedOnStderr`、`TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr`、`TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr`、`TestOrgSend_AppendEventFailsAfterEnter_NotesEnterPressedNotRecorded`、`TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の ORIGIN のない側 |
| Enter は押され `sent` だけ書けなかったときの note | 人の決定でも変わらない | `TestOrgSend_HumanDecisionEnterPressedNotRecorded_NoteUnchanged` |
| 1 周目の AC1〜AC7 のテスト | 変わらず通る | AC run org(`SendOrigin`、`ReseatProgress`、`HumanDecision`、`DirectorSkill` を含む 208 関数)、run-test.sh |
| gate.lock の待ちが名指しの holder を残す(#367、`7e8ac5b1` で取り込み) | 通る | run-test.sh の `internal/org`(`TestAcquireGateLock_KeepsTheNamedHolderWhenTheMarkerReadsEmpty`、`TestGate_TwoGatesRunTheirScriptsOneAtATime`) |

### Test gaps

1 周目の Test gaps 1〜5 は変わらない(下の Run 1 の「Test gaps」)。6 と、新しい 7〜9 は次のとおり。

6. (1 周目の 6 を狭めた)`TestAgentMessagingRule_OriginHumanIsRecordedOnlyByASendThatExits0` が、規則の 2 写しの `ORIGIN: human` の節の見出しと、終了コード 0 の文から「The leader carries out the decision」までを固定するようになった(N12 で確かめた)。節の前半(ralph が ORIGIN を許す形、16 KiB、断られる形)と最後の「ralph cannot tell who wrote the message」の文は、まだどのテストも固定していない。重さは LOW(送信の規則そのものは `checkSendOrigin` のテストが固定している)。verify run 2 の「いちばん小さく足せる確かめ」と同じもの
7. 人の決定の note を読んだ director が pane で正しく振る舞うこと(キーを押さない、空の入力欄にだけ送り直す)は、prompt と skill の文までしか固定していない。振る舞いは実機の org でしか確かめられない
8. `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` の Enter の前の待ちの時間切れの subtest は、fake の herdr の send-text に 1.1 s の sleep を入れ、`--timeout-ms 2500` と `--enter-delay-ms 1800` で時間切れを起こす。既存の `TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr` と同じ時間の組なので、負荷で外れるとすれば同じように外れる。今回の run(1 分 load 最大 32)では通った
9. origin/main は `10e9322d`(#368、codex-seat-network)まで進んでいる。`git merge-tree --write-tree HEAD origin/main` は `docs/tech-debt/README.md` で衝突し、`internal/cli/org.go` は自動でまとまる。この run は取り込む前の HEAD のもので、`/pr` の前に main を取り込んだら、そのあとの Go をもう一度通す必要がある

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、`PASS` 行 5,390、Go 11 パッケージ、rc 0、1,835 s)、`(cached)` だった 8 パッケージの `-count=1` の打ち直し、AC run cli(関数 12、subtest 12)、AC run org(関数 208、subtest 229)、1 周目のあとに足したテスト関数 7 本と文を足した 4 本、変異 12 個(N1、N2、N4〜N13)をすべて検出
- Fail: 0
- Blocked: なし
- cycle 1 の判定(pass)は、下の「Run 1」の「Verdict」にそのまま残す

---

## Run 1 (cycle 1、履歴)

この下は run 1(HEAD `b97a336c`)の report を、見出しの階層を 1 つ下げた以外は変えずに残したもの。

- Date: 2026-11-22 JST(cycle 1。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366、AC1〜AC7)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b97a336c`(verify のあとに `ralph org reseat --help` へ人の決定を足した commit)。worktree は clean。`./scripts/run-test.sh` はこの worktree で 1 回だけ実行した。起動の時点で、この worktree を cwd にした run-test.sh / run-verify.sh は動いていなかったので、新しく起動した。起動の形は perl の fork と setsid による切り離しで、INT/HUP/QUIT/TERM を DEFAULT に戻し、標準入出力を `/dev/null` に向けた(出力はファイル)。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと、AC に絞った `-v` の run を 2 本、`go test -overlay` による変異を 5 個(依頼の 3 種と追加 2 個)、絞った run の coverage を取った。追跡しているファイルは何も変えていない(`git status --porcelain --ignored` の前後の差は、無視されている `.harness/cache/go-build/` と evidence の log だけ)
- Evidence: `docs/evidence/test-2026-11-21-org-human-decision.log`(14,884 行。run-test.sh の出力 5,829 行のあとに、開始と終了の時刻、AC の run の出力、変異ごとの overlay・diff・出力、coverage を足した。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-21-154654.log` にも書いた(同じく追跡外)
- 環境: go1.27.3 darwin/arm64、git 2.49.0

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-22 00:46:54 〜 01:28:25 JST、2,491 s、rc 0(`==> All verifiers passed.`)。1 分 load は開始 14.84、終了 17.80、途中の観測で 10.05〜27.25 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0 | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0)。前の cycle までと同じ 2 件 | 約 22 分(00:46:54〜01:09 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 19 分(01:09 ごろ〜01:28:25)、下の表 |
| AC run 1: `go test -count=1 -v ./internal/org/ -run 'Origin\|Send\|Reseat\|Progress\|Prompt\|Director'` | 関数 808、subtest 1,732 | 関数 807、subtest 1,732 | 0 | 1(`TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved`、Linux だけのテスト) | 426 s(パッケージ 424.154 s) |
| AC run 2: `go test -count=1 -v ./internal/cli/ -run 'OrgSend\|Reseat\|Help'` | 関数 111、subtest 71 | 関数 111、subtest 71 | 0 | 0 | 275 s(パッケージ 273.869 s) |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.484 s |
| `internal/cli` | ok | 1,087.958 s |
| `internal/config` | ok | 27.236 s |
| `internal/gitroot` | ok | 10.643 s |
| `internal/insights` | ok | 3.988 s |
| `internal/org` | ok | 1,084.799 s |
| `internal/org/driver` | ok | 5.378 s |
| `internal/org/protocol` | ok | 4.742 s |
| `internal/scaffold` | ok | 1.960 s |
| `internal/toolchain` | ok | 25.588 s |
| `internal/upgrade` | ok | 3.187 s |

`internal/org` と `internal/cli` は golang pack の 1 パッケージの上限(30 分)の 60% で終わった。run-test.sh には、ほかの run が重なった。local verifier の段で、この report の変異の run(00:47〜00:54)と AC の run(00:54〜01:05)と coverage の run(01:07〜01:09)。Go の段の前後で、別のレーンの `go test ./internal/org/ -run 'Reserve|Reservation|Feature'`(00:46 ごろと 01:26 ごろ)。落ちたテストはなく、打ち直したテストもない。

この branch で足したテスト関数は、`internal/org` に 24(`send_origin_test.go` 8、`reseat_progress_decision_test.go` 10、`prompts_leader_human_decision_test.go` 6)、`internal/cli` に 3(`TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded`、`TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded`、`TestOrgSend_HelpDescribesTheOriginHeader`)。どれも AC run と run-test.sh の両方で通った。文を変えた既存のテスト(`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`、`prompts_leader_reseat_test.go`)も通った。

#### 計画の AC とテスト

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText`、`TestOrgSendOrigin_DryRun_MarksDetailsWithoutText`、`TestOrgSendOrigin_ScreenPath_DetailsOrder`、`TestCheckSendOrigin_Table` の通す行、CLI の `TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded` | `ORIGIN: human` の DECISION を leader へ送ると `sent` の Details が `origin=human ` で始まり(`raw=true ` のあと、screen の道の `via=screen` の前)、全文が `text` に残る。dry run は印だけで `text` を持たない |
| AC2 | `TestCheckSendOrigin_Table` の断る行(TYPE が違う、値が `Human`・`HUMAN`・`director`・空、宛先が `implementer`・`Leader`、ORIGIN が 2 行、16 KiB を 1 バイト超える)、`TestCheckSendOrigin_EveryTypeButDecisionRefused`、`TestOrgSendOrigin_Refused_NothingTypedNothingRecorded`(6 形 × dry run の有無の 12 subtest)、`TestSendOriginRule_Text`、CLI の `TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded`、変異 M1 | 断るときは規則の文を含む誤りを返し、台帳の event も driver の呼び出しも増えない。`--raw` と dry run でも同じ |
| AC3 | `TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore`、`TestCheckSendOrigin_Table` の ORIGIN のない行・`Origin:` の行・本文の `ORIGIN: human` の行、AC run 1 の既存の Send のテスト、run-test.sh の `internal/org` と `internal/cli` | ORIGIN のない送信は `origin=` の印も `text` の key も持たず、今のとおり記録される |
| AC4 | `TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor`、`..._HumanDecisionBulletSaysWhatTheLeaderDoes`、`..._ReseatRuleFollowsTheHumanDecisions`、`TestReseatProgressLead_NamesTheHumanDecisionsAsTheException`、変異 M3 と M3b | leader の prompt の手順 4(承認待ち)と cross-review の上限の答え待ちの両方に、`ORIGIN: human` の DECISION を待っている答えとみなさない文がある。どちらの文を消してもテストが落ちる |
| AC5 | `TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader`、`TestDirectorSkill_ReseatSendsWhatIsDoneAsAHumanDecision`、local verifier の `tests/test-check-skill-sync.sh` と `tests/test-check-sync.sh` | director skill の写しが件のない人の決定の送り方を書く。本物の tree での `check-skill-sync.sh` と `check-sync.sh` は `/verify` の役目で、verify report で通っている。b97a336c は Go のコードと help のテストだけを変えたので、その結果は変わらない |
| AC6 | なし(段落があることの確かめは `/verify`、verify report の AC6) | agent-messaging.md の `ORIGIN: human` の段落を読むテストはない。この文書を名指すテスト(`internal/org/prompts_test.go`、`internal/cli/org_test.go`)は path が書かれていることだけを見る。Test gaps の 6 |
| AC7 | `TestReseatProgress_AHumanDecisionIsListedWhole`(200 バイトを超える決定)、`..._HumanDecisionsKeepTheLedgerOrder`、`..._AHumanDecisionBeforeTheLastDisbandIsNotListed`、`..._OnlyARealHumanDecisionToTheLeaderIsListed`、`..._AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole`(`ralph org reseat`)、`..._AStartThatRestartsTheLeaderGivesItTheHumanDecisions`(`start --plan`)、`..._AStartAfterADisbandGivesNoDecisionOfTheOldLife`、変異 M2a と M2b | 立て直した leader の進捗の節に、決定の全文と時刻が台帳の順で載る。disband の前の命の決定、dry run、ORIGIN のない leader 宛ての送信、印の文字で始まるだけの `--raw` は載らない |

#### 変異(`go test -overlay`、追跡ファイルは変えていない)

`internal/org/prompts/leader.md` は `internal/org/prompts.go` の `//go:embed prompts/*.md` で埋め込まれるので、M3 と M3b は埋め込むファイルそのものを overlay で差し替えた。

| # | 変異 | run | 結果 |
| --- | --- | --- | --- |
| M1(依頼 1) | `send_origin.go` の `checkSendOrigin` から宛先の確かめ(`if to != LeaderIdentity { … }`)を消す | `./internal/org/ -run 'Origin'`(88 s)、`./internal/cli/ -run 'OrgSend_Origin'`(10 s) | 検出。org で `TestCheckSendOrigin_Table` の「to a seat that is not the leader」「to Leader, not leader」と、`TestOrgSendOrigin_Refused_NothingTypedNothingRecorded` の宛先が座席の 4 subtest(raw の有無 × dry run の有無)が落ちた。本物の送信の行は `Send = {Err:<nil> … Progress:enter-pressed PaneID:split-1}; want the ORIGIN refusal`。CLI で `TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded` が落ちた(`send = <nil> (output: sent message to seat "seat-1")`) |
| M2a(依頼 2) | `reseat_progress.go` の `readReseatProgress` で、`Text` の有無を見ずに印だけで決定とし、本文に Details を使う | `./internal/org/ -run 'ReseatProgress'`(19 s) | 検出。200 バイトを超える決定の `TestReseatProgress_AHumanDecisionIsListedWhole` が落ちた(`reseat_progress_decision_test.go:79`、載った本文が `origin=human TYPE: DECISION` で始まり、`...(truncated)` で切れて最後の行がない)。ほかに 7 関数(台帳の順、disband の前、印だけ・raw の 3 subtest、entry の中、手で書いた長い決定、reseat、start の立て直し)も落ちた |
| M2b(追加) | `Text` が空のときだけ Details で代える(`Text` があればそれを使う) | 同上(21 s) | 検出。`TestReseatProgress_OnlyARealHumanDecisionToTheLeaderIsListed` の「the mark with no Text」と「a raw text to the leader that starts with the words of the mark」が落ちた。200 バイトを超える決定のテストは、fixture が `Text` を持つので通る(この変異の道に入らない) |
| M3(依頼 3) | `prompts/leader.md` の手順 4(承認待ち)から `ORIGIN: human` の DECISION の文(491〜495 行)を消す | `./internal/org/ -run 'RenderRolePrompt\|ReseatProgressLead\|DirectorSkill\|Prompt'`(170 s) | 検出。`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor` だけが落ちた(`prompts_leader_human_decision_test.go:72`、手順 4 の項に `leaderHumanDecisionApprovalSkip` がない)。ほかの 451 件は通った |
| M3b(追加) | 同じファイルの cross-review の上限の答え待ちから `ORIGIN: human` の DECISION の文(1182〜1184 行)を消す | `./internal/org/ -run 'RenderRolePrompt_Leader\|ReseatProgressLead\|DirectorSkill'`(76 s) | 検出。同じテストが `prompts_leader_human_decision_test.go:78`(cross-review の小節に `leaderHumanDecisionCapSkip` がない)で落ちた。AC4 の 2 つの文がどちらも固定されている |

ビルドが落ちた変異はない(どの log にも `build failed` はない)。

### Coverage

- Statement: 絞った run(`go test -count=1 -coverprofile … ./internal/org/ -run 'Origin|ReseatProgress|HumanDecision|DirectorSkill|OrgSend|Send_'`、110 s)の profile で、変更行のうち文の block に入る行はすべて実行された。`send_origin.go` 29/29、`reseat_progress.go` 31/31、`verbs.go` 12/12(`git diff -U0 main...HEAD` の追加行と profile の block の突き合わせ)。関数ごとには `checkSendOrigin`・`sentHumanText`・`sentDetailsHumanOrigin`・`sendOriginError.Error`・`readReseatProgress`・`reseatProgressSection`・`reseatProgressEscape`・`reseatProgressDecisionText`・`reseatProgressText` が 100%、`Send` が 97.8%
- Branch: 計測していない(Go の標準の道具にない)
- Function: 絞った run なので、パッケージ全体の数字は出していない(パッケージの 41.2%)
- Notes: `internal/cli` の変更(`ralph org send --help` と `ralph org reseat --help` の文、`org_reseat.go` の 6 行)は coverage を取っていない。文は help のテスト 2 本が固定していて、AC run 2 と run-test.sh で通った

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| ORIGIN のない send の記録(AC3) | 変わらず通る | `TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore`、AC run 1 の Send のテスト、run-test.sh の `internal/org` |
| Details の印の順(`raw=true ` → `origin=human ` → `via=screen …` → `submit_unconfirmed=true `) | 通る | `TestOrgSendOrigin_ScreenPath_DetailsOrder`、`TestReseatProgress_TheHumanOriginMarkIsReadWhereSendWritesIt` |
| 立て直しの進捗の節の既存の項(承認、gate、TASK、受信箱、RESULT、commit、前の立て直し) | 変わらず通る | AC run 1 の `ReseatProgress` と `Reseat` のテスト。進捗の節の頭(`reseatProgressLead`)の文の変更は `prompts_leader_reseat_test.go` と `TestReseatProgressLead_NamesTheHumanDecisionsAsTheException` が固定 |
| `ralph org reseat --help` の段落(b97a336c で人の決定を足した) | 通る | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` が AC run 2 と run-test.sh で通った |
| 貼り付けマーカーの確かめ、watch gate、per-plan の承認の確かめ(ORIGIN の確かめはその間に入った) | 変わらず通る | AC run 1 の `Send` のテスト(`TestOrgSend_WatchGate_*` を含む)、run-test.sh |

### Test gaps

1. leader が本物の LLM として `ORIGIN: human` の DECISION を承認や上限の答えと取り違えないことは、テストしていない。テストが固定しているのは prompt の文(M3・M3b で両方の文が固定されていると確かめた)までで、振る舞いは実機の org でしか確かめられない
2. compaction のあとに leader が人の決定を読み直す手段はない(計画の S3 の気づき 4)。進捗の節に載るのは立て直したときだけで、それを確かめるテストも、compaction の道のテストもない。設計の範囲の外として残る
3. ralph は人が書いたことを確かめない(計画の Non-goals)。`ORIGIN: human` は director skill の決まりで守る。テストは「台帳の印が人の決定として送られた記録である」ことまでを固定する
4. `reseat` と `start --plan` の立て直しのテストは fake の herdr と agmsg で回る。本物の herdr の pane に進捗の節が届くことは確かめていない(依頼どおり本物の herdr server には触れていない)
5. `internal/cli` の help の文の変更は coverage を取っていない。help のテストが文を固定している
6. `.claude/rules/ralph/agent-messaging.md` の `ORIGIN: human` の段落(AC6)を固定するテストはない。雛形の写しとの一致は `check-sync.sh` が見るが、段落を消しても落ちるテストはない。重さは LOW(人が読む契約の文書で、送信の規則そのものは `checkSendOrigin` のテストが固定している)

### Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、2,491 s、`(cached)` なし)、AC run 1(関数 807 通過、1 skip は Linux だけのテスト)、AC run 2(関数 111、subtest 71)、新しいテスト関数 27 本、AC1〜AC5 と AC7 のテスト(AC6 は文書の段落で、テストはない)、依頼の変異 M1・M2a・M3 を検出、追加の変異 M2b・M3b も検出
- Fail: 0
- Blocked: なし
