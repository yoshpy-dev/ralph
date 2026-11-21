# Test report: org-human-decision

- Date: 2026-11-22 JST(cycle 1。`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-21-org-human-decision.md`(issue #366、AC1〜AC7)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `b97a336c`(verify のあとに `ralph org reseat --help` へ人の決定を足した commit)。worktree は clean。`./scripts/run-test.sh` はこの worktree で 1 回だけ実行した。起動の時点で、この worktree を cwd にした run-test.sh / run-verify.sh は動いていなかったので、新しく起動した。起動の形は perl の fork と setsid による切り離しで、INT/HUP/QUIT/TERM を DEFAULT に戻し、標準入出力を `/dev/null` に向けた(出力はファイル)。scope は wrapper の既定の changed(`- Requested scope: changed`、language pack は golang)。local verifier は scope によらず `tests/test-*.sh` を全部回す。そのあと、AC に絞った `-v` の run を 2 本、`go test -overlay` による変異を 5 個(依頼の 3 種と追加 2 個)、絞った run の coverage を取った。追跡しているファイルは何も変えていない(`git status --porcelain --ignored` の前後の差は、無視されている `.harness/cache/go-build/` と evidence の log だけ)
- Evidence: `docs/evidence/test-2026-11-21-org-human-decision.log`(14,884 行。run-test.sh の出力 5,829 行のあとに、開始と終了の時刻、AC の run の出力、変異ごとの overlay・diff・出力、coverage を足した。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は自分の evidence を `docs/evidence/verify-2026-11-21-154654.log` にも書いた(同じく追跡外)
- 環境: go1.27.3 darwin/arm64、git 2.49.0

## Test execution

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

### 計画の AC とテスト

| AC | テスト | 確かめたこと |
| --- | --- | --- |
| AC1 | `TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText`、`TestOrgSendOrigin_DryRun_MarksDetailsWithoutText`、`TestOrgSendOrigin_ScreenPath_DetailsOrder`、`TestCheckSendOrigin_Table` の通す行、CLI の `TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded` | `ORIGIN: human` の DECISION を leader へ送ると `sent` の Details が `origin=human ` で始まり(`raw=true ` のあと、screen の道の `via=screen` の前)、全文が `text` に残る。dry run は印だけで `text` を持たない |
| AC2 | `TestCheckSendOrigin_Table` の断る行(TYPE が違う、値が `Human`・`HUMAN`・`director`・空、宛先が `implementer`・`Leader`、ORIGIN が 2 行、16 KiB を 1 バイト超える)、`TestCheckSendOrigin_EveryTypeButDecisionRefused`、`TestOrgSendOrigin_Refused_NothingTypedNothingRecorded`(6 形 × dry run の有無の 12 subtest)、`TestSendOriginRule_Text`、CLI の `TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded`、変異 M1 | 断るときは規則の文を含む誤りを返し、台帳の event も driver の呼び出しも増えない。`--raw` と dry run でも同じ |
| AC3 | `TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore`、`TestCheckSendOrigin_Table` の ORIGIN のない行・`Origin:` の行・本文の `ORIGIN: human` の行、AC run 1 の既存の Send のテスト、run-test.sh の `internal/org` と `internal/cli` | ORIGIN のない送信は `origin=` の印も `text` の key も持たず、今のとおり記録される |
| AC4 | `TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor`、`..._HumanDecisionBulletSaysWhatTheLeaderDoes`、`..._ReseatRuleFollowsTheHumanDecisions`、`TestReseatProgressLead_NamesTheHumanDecisionsAsTheException`、変異 M3 と M3b | leader の prompt の手順 4(承認待ち)と cross-review の上限の答え待ちの両方に、`ORIGIN: human` の DECISION を待っている答えとみなさない文がある。どちらの文を消してもテストが落ちる |
| AC5 | `TestDirectorSkill_SendsAHumanDecisionWithTheOriginHeader`、`TestDirectorSkill_ReseatSendsWhatIsDoneAsAHumanDecision`、local verifier の `tests/test-check-skill-sync.sh` と `tests/test-check-sync.sh` | director skill の写しが件のない人の決定の送り方を書く。本物の tree での `check-skill-sync.sh` と `check-sync.sh` は `/verify` の役目で、verify report で通っている。b97a336c は Go のコードと help のテストだけを変えたので、その結果は変わらない |
| AC6 | なし(段落があることの確かめは `/verify`、verify report の AC6) | agent-messaging.md の `ORIGIN: human` の段落を読むテストはない。この文書を名指すテスト(`internal/org/prompts_test.go`、`internal/cli/org_test.go`)は path が書かれていることだけを見る。Test gaps の 6 |
| AC7 | `TestReseatProgress_AHumanDecisionIsListedWhole`(200 バイトを超える決定)、`..._HumanDecisionsKeepTheLedgerOrder`、`..._AHumanDecisionBeforeTheLastDisbandIsNotListed`、`..._OnlyARealHumanDecisionToTheLeaderIsListed`、`..._AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole`(`ralph org reseat`)、`..._AStartThatRestartsTheLeaderGivesItTheHumanDecisions`(`start --plan`)、`..._AStartAfterADisbandGivesNoDecisionOfTheOldLife`、変異 M2a と M2b | 立て直した leader の進捗の節に、決定の全文と時刻が台帳の順で載る。disband の前の命の決定、dry run、ORIGIN のない leader 宛ての送信、印の文字で始まるだけの `--raw` は載らない |

### 変異(`go test -overlay`、追跡ファイルは変えていない)

`internal/org/prompts/leader.md` は `internal/org/prompts.go` の `//go:embed prompts/*.md` で埋め込まれるので、M3 と M3b は埋め込むファイルそのものを overlay で差し替えた。

| # | 変異 | run | 結果 |
| --- | --- | --- | --- |
| M1(依頼 1) | `send_origin.go` の `checkSendOrigin` から宛先の確かめ(`if to != LeaderIdentity { … }`)を消す | `./internal/org/ -run 'Origin'`(88 s)、`./internal/cli/ -run 'OrgSend_Origin'`(10 s) | 検出。org で `TestCheckSendOrigin_Table` の「to a seat that is not the leader」「to Leader, not leader」と、`TestOrgSendOrigin_Refused_NothingTypedNothingRecorded` の宛先が座席の 4 subtest(raw の有無 × dry run の有無)が落ちた。本物の送信の行は `Send = {Err:<nil> … Progress:enter-pressed PaneID:split-1}; want the ORIGIN refusal`。CLI で `TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded` が落ちた(`send = <nil> (output: sent message to seat "seat-1")`) |
| M2a(依頼 2) | `reseat_progress.go` の `readReseatProgress` で、`Text` の有無を見ずに印だけで決定とし、本文に Details を使う | `./internal/org/ -run 'ReseatProgress'`(19 s) | 検出。200 バイトを超える決定の `TestReseatProgress_AHumanDecisionIsListedWhole` が落ちた(`reseat_progress_decision_test.go:79`、載った本文が `origin=human TYPE: DECISION` で始まり、`...(truncated)` で切れて最後の行がない)。ほかに 7 関数(台帳の順、disband の前、印だけ・raw の 3 subtest、entry の中、手で書いた長い決定、reseat、start の立て直し)も落ちた |
| M2b(追加) | `Text` が空のときだけ Details で代える(`Text` があればそれを使う) | 同上(21 s) | 検出。`TestReseatProgress_OnlyARealHumanDecisionToTheLeaderIsListed` の「the mark with no Text」と「a raw text to the leader that starts with the words of the mark」が落ちた。200 バイトを超える決定のテストは、fixture が `Text` を持つので通る(この変異の道に入らない) |
| M3(依頼 3) | `prompts/leader.md` の手順 4(承認待ち)から `ORIGIN: human` の DECISION の文(491〜495 行)を消す | `./internal/org/ -run 'RenderRolePrompt\|ReseatProgressLead\|DirectorSkill\|Prompt'`(170 s) | 検出。`TestRenderRolePrompt_Leader_HumanDecisionIsNotTheAnswerItWaitsFor` だけが落ちた(`prompts_leader_human_decision_test.go:72`、手順 4 の項に `leaderHumanDecisionApprovalSkip` がない)。ほかの 451 件は通った |
| M3b(追加) | 同じファイルの cross-review の上限の答え待ちから `ORIGIN: human` の DECISION の文(1182〜1184 行)を消す | `./internal/org/ -run 'RenderRolePrompt_Leader\|ReseatProgressLead\|DirectorSkill'`(76 s) | 検出。同じテストが `prompts_leader_human_decision_test.go:78`(cross-review の小節に `leaderHumanDecisionCapSkip` がない)で落ちた。AC4 の 2 つの文がどちらも固定されている |

ビルドが落ちた変異はない(どの log にも `build failed` はない)。

## Coverage

- Statement: 絞った run(`go test -count=1 -coverprofile … ./internal/org/ -run 'Origin|ReseatProgress|HumanDecision|DirectorSkill|OrgSend|Send_'`、110 s)の profile で、変更行のうち文の block に入る行はすべて実行された。`send_origin.go` 29/29、`reseat_progress.go` 31/31、`verbs.go` 12/12(`git diff -U0 main...HEAD` の追加行と profile の block の突き合わせ)。関数ごとには `checkSendOrigin`・`sentHumanText`・`sentDetailsHumanOrigin`・`sendOriginError.Error`・`readReseatProgress`・`reseatProgressSection`・`reseatProgressEscape`・`reseatProgressDecisionText`・`reseatProgressText` が 100%、`Send` が 97.8%
- Branch: 計測していない(Go の標準の道具にない)
- Function: 絞った run なので、パッケージ全体の数字は出していない(パッケージの 41.2%)
- Notes: `internal/cli` の変更(`ralph org send --help` と `ralph org reseat --help` の文、`org_reseat.go` の 6 行)は coverage を取っていない。文は help のテスト 2 本が固定していて、AC run 2 と run-test.sh で通った

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| ORIGIN のない send の記録(AC3) | 変わらず通る | `TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore`、AC run 1 の Send のテスト、run-test.sh の `internal/org` |
| Details の印の順(`raw=true ` → `origin=human ` → `via=screen …` → `submit_unconfirmed=true `) | 通る | `TestOrgSendOrigin_ScreenPath_DetailsOrder`、`TestReseatProgress_TheHumanOriginMarkIsReadWhereSendWritesIt` |
| 立て直しの進捗の節の既存の項(承認、gate、TASK、受信箱、RESULT、commit、前の立て直し) | 変わらず通る | AC run 1 の `ReseatProgress` と `Reseat` のテスト。進捗の節の頭(`reseatProgressLead`)の文の変更は `prompts_leader_reseat_test.go` と `TestReseatProgressLead_NamesTheHumanDecisionsAsTheException` が固定 |
| `ralph org reseat --help` の段落(b97a336c で人の決定を足した) | 通る | `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` が AC run 2 と run-test.sh で通った |
| 貼り付けマーカーの確かめ、watch gate、per-plan の承認の確かめ(ORIGIN の確かめはその間に入った) | 変わらず通る | AC run 1 の `Send` のテスト(`TestOrgSend_WatchGate_*` を含む)、run-test.sh |

## Test gaps

1. leader が本物の LLM として `ORIGIN: human` の DECISION を承認や上限の答えと取り違えないことは、テストしていない。テストが固定しているのは prompt の文(M3・M3b で両方の文が固定されていると確かめた)までで、振る舞いは実機の org でしか確かめられない
2. compaction のあとに leader が人の決定を読み直す手段はない(計画の S3 の気づき 4)。進捗の節に載るのは立て直したときだけで、それを確かめるテストも、compaction の道のテストもない。設計の範囲の外として残る
3. ralph は人が書いたことを確かめない(計画の Non-goals)。`ORIGIN: human` は director skill の決まりで守る。テストは「台帳の印が人の決定として送られた記録である」ことまでを固定する
4. `reseat` と `start --plan` の立て直しのテストは fake の herdr と agmsg で回る。本物の herdr の pane に進捗の節が届くことは確かめていない(依頼どおり本物の herdr server には触れていない)
5. `internal/cli` の help の文の変更は coverage を取っていない。help のテストが文を固定している
6. `.claude/rules/ralph/agent-messaging.md` の `ORIGIN: human` の段落(AC6)を固定するテストはない。雛形の写しとの一致は `check-sync.sh` が見るが、段落を消しても落ちるテストはない。重さは LOW(人が読む契約の文書で、送信の規則そのものは `checkSendOrigin` のテストが固定している)

## Verdict

- Pass: `./scripts/run-test.sh`(shell 46 本、Go 11 パッケージ、rc 0、2,491 s、`(cached)` なし)、AC run 1(関数 807 通過、1 skip は Linux だけのテスト)、AC run 2(関数 111、subtest 71)、新しいテスト関数 27 本、AC1〜AC5 と AC7 のテスト(AC6 は文書の段落で、テストはない)、依頼の変異 M1・M2a・M3 を検出、追加の変異 M2b・M3b も検出
- Fail: 0
- Blocked: なし
