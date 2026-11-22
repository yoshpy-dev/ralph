# Test report: director-wait-types-note

- Date: 2026-11-22(実行は 16:03:45〜16:27:08 JST。パイプラインの cycle 1)
- Plan: `docs/plans/active/2026-11-22-director-wait-types-note.md`(issue #374)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `ea088c24`(worktree は clean、push 済み。base は origin/main `b263b1f3`)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `ps -axo pid,command` で `run-test.sh` と `run-verify.sh` が走っていないことを見た。perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 170 s ずつ待った。`nohup` は使っていない。そのあと `-v` の run、`-race -count=5` の run、変異 8 個と穴を測る変異 2 個、CLI の overlay の probe を足した
- Evidence: `docs/evidence/test-2026-11-22-director-wait-types-note.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力の後ろに、load、`-v` と `-race` の run の要約、変異ごとの落ちたテストとその行、probe の結果、被覆率、`~/.codex/config.toml` と herdr の前後を足した。wrapper 自身の log は `docs/evidence/verify-2026-11-22-070345.log`(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 16:03:45〜16:27:08 JST、1,403 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本 | 0(`FAIL` の行はすべて `FAIL: 0`) | 2 つ: `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」(git 2.49.0 なので。古い git の stub の行は通った) | 約 12 分(16:03:45〜16:15 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` も 0 なので打ち直しは要らなかった | 約 11 分(16:15 ごろ〜16:27:08)、下の表 |
| `go test -count=1 -v ./internal/org/ -run '^(足した 12 関数と変えた 3 関数と TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase)$'` | 16 関数、`=== RUN` 51 | 51 | 0 | 0 | 3.934 s、rc 0 |
| `go test -count=1 -v ./internal/cli/ -run '^(TestOrgWaitInbox_TypesPrintOnlyTheListedItems\|TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem\|TestOrgWait_TypesIsRefusedWithoutInboxOrWithABadValue\|TestOrgWait_HelpDescribesTypes)$'` | 4 関数、`=== RUN` 18 | 18 | 0 | 0 | 1.170 s、rc 0 |
| `go test -race -count=5 -v ./internal/org/ -run '^TestWaitInboxForDirector_TwoOverlapping'` | 2 関数 × 5 回(サブテスト 10) | 10 と 10 | 0 | 0 | 7.162 s、rc 0。`WARNING: DATA RACE` は 0 |
| CLI の overlay の probe(下の「足した確かめ」) | 1 | 1 | 0 | 0 | 1.606 s、rc 0 |

| パッケージ | run-test.sh |
| --- | --- |
| `cmd/ralph` | ok 0.840 s |
| `internal/cli` | ok 737.156 s |
| `internal/config` | ok 13.462 s |
| `internal/gitroot` | ok 3.016 s |
| `internal/insights` | ok 1.702 s |
| `internal/org` | ok 743.613 s |
| `internal/org/driver` | ok 3.890 s |
| `internal/org/protocol` | ok 4.347 s |
| `internal/scaffold` | ok 3.734 s |
| `internal/toolchain` | ok 20.618 s |
| `internal/upgrade` | ok 3.570 s |

shell の段の `PASS` の行は 5,390 で、直前の issue-depends-wait の cycle 2(同じ日、base の 1 つ前)と同じ数だった。`internal/cli` と `internal/org` は、golang pack の 1 パッケージの上限 30 分のおよそ 41% で終わった。

1 分 load は 9.48〜22.92(run の始まりと終わりを含めて 9 回測った)。22.92 は shell の段の 16:10 ごろで、私の変異の run が終わった直後。`-v` の run、`-race` の run、変異 10 個とその対照はどれも shell の段(16:04〜16:10)に、`git archive HEAD` の写しで回した(変異の写しの 3 ファイルは最後に worktree と `cmp` で同じ)。被覆率と probe の run は run-test.sh のあとに回した。空き容量は 34 GiB から 33 GiB だった。

run の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じだった。読まれたかどうかは測っていない。herdr のプロセス(`herdr server` の pid 59855、11 月 20 日 13:39:32 起動)は前後で同じで、触れていない。本物の org の台帳には触れていない(どのテストも `t.TempDir()` の台帳)。worktree の `git status --porcelain` は run のあとも空だった。

### 足したテストと変えたテスト

`-v` の run で、次のすべてが PASS し、SKIP は 0 だった。skill を読む 3 関数は skill がないと `t.Skipf` するが、4 写しとも読めた。

- `internal/org/director_wait_types_note_test.go`(9 関数): `TestInboxWaitOtherTypesText`(11 行。切る境目の 5 件と 6 件以上、エスケープを含む)、`TestInboxWaitTimeoutNote`、`TestReadInboxWaitOtherTypesText_ReadsTheOpenItemsOrNothing`(acked の件は数えない、読めない受信箱は空の句)、`TestWaitInbox_TypesTimeoutNamesTheOpenItemsOfOtherTypes`、`TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`(4 行。待つあいだに resolve された件を含む)、`TestWaitInboxForDirector_TypesTimeoutNamesTheOpenItemsOfOtherTypes`、`TestWaitInboxForDirector_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`、`TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems`(見張りが director に任せた場合と人へすぐ送った場合)、`TestWaitInboxForDirector_TwoOverlappingTypesWaitsBothNameTheOpenAlerts`
- `internal/org/director_skill_wait_types_note_test.go`(3 関数): `TestDirectorSkillStopsAWait`(38 行の言い回しの表)、`TestDirectorSkill_RunsOneInboxWaitAtATime`、`TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout`
- 固定を変えた関数: `TestWaitInboxForDirector_TypesTimeOutWhileOnlyAnAlertIsOpen` と `TestDirectorWaitTimeoutDetail`(`director_wait_test.go`)、`TestOrgWaitInbox_TypesKeepWaitingPastAnAlertItem`(stderr を完全一致に強めた)と `TestOrgWait_HelpDescribesTypes`(`internal/cli/org_inbox_test.go`)。前からの `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` も通った

### 足した確かめ: CLI の `--director` の時間切れ

verify の報告が「Not verified」に挙げた director の wait の時間切れを、CLI の層で 1 回確かめた。worktree に何も書かず、`-overlay` で `internal/cli` にテスト 1 つ(`TestProbeDWTN_DirectorTypesTimeoutViaCLI`、scratchpad に置いた)を足した。`attachD1` のあと ALERT の件を 1 つ記録し、`ralph org wait --inbox --director d1 --types QUESTION,BLOCKED,RESULT --timeout-ms 200` を cobra で回した。終了コード 1、エラーの文は `org: wait --inbox --director: no open inbox item arrived within 200 ms (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT)` と完全に一致し、印の detail は `no open inbox item of type QUESTION,BLOCKED,RESULT within 200 ms (open items of other TYPEs: e1 ALERT)`、`ralph org director status --json` の出力にその detail が入っていた。ビルドした binary では回していない。

## Coverage

- Statement: `internal/org` を `-run 'WaitInbox|DirectorWait|Inbox|DirectorSkill|Skill'` で回した被覆率で、足した `readInboxWaitOtherTypesText`、`inboxWaitOtherTypesText`、`inboxWaitTimeoutNote` と変えた `directorWaitTimeoutDetail` は 100%。`WaitInbox` は 95.8%、`directorWaitRun.loop` は 97.6% で、通らないブロックは変えた行の外にある(変えた行の範囲で回数 0 のブロックは 0)
- Branch: 測っていない(Go の被覆率は statement だけ)
- Function: 同上
- Notes: パッケージ全体の数字は、`-run` で絞ったので意味がない(25.9%)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--types` なしの時間切れの文と、外の件がないときの文が変わらない | 通った | `TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`、`TestWaitInboxForDirector_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType`、`TestOrgWaitDirector_TimesOutWithThePhrase`、`TestOrgWaitInbox_WithoutDirectorHandlesNoSignal`(完全一致) |
| skill が照合する時間切れの句が文の先頭に残る | 通った | `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` |
| 乗っ取り、終了コード 4 と 5、Monitor の期限の既存のテスト | 通った | `internal/org` と `internal/cli` の全体(run-test.sh) |
| 重なった 2 つの director wait が、乗っ取られた側も件を返す(issue #374 の期待の 2 つ目) | 通った | `TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems`、`-race -count=5` で 5 回とも |

## 変異

HEAD の `git archive` の写しで、1 つずつ替え、`go test -count=1 -v ./internal/org/ -run 'WaitInbox|DirectorWait|Inbox|DirectorSkill|Skill'`(634 の `=== RUN`)を回した。置き換える文字列は写しに 1 回だけ現れることを確かめてから替え、毎回元に戻して `cmp` で確かめた。対照(変異なし)は rc 0、SKIP 0。どの変異の run も SKIP 0 で、build の失敗はない。

依頼の 3 つは M1、M2、M3。

| # | 替えたもの | 変異 | 落ちたテスト |
| --- | --- | --- | --- |
| M1 | `director_wait.go` | `WaitInboxForDirector` の時間切れのエラーから句を外す(`inboxWaitTimeoutNote(...)` を `""` に。detail には残る) | `TestWaitInboxForDirector_TypesTimeoutNamesTheOpenItemsOfOtherTypes`、`TestWaitInboxForDirector_TwoOverlappingTypesWaitsBothNameTheOpenAlerts` |
| M2 | `escalate.go` | 5 件で切るのをやめる(`if true \|\| len(others) <= …`) | `TestInboxWaitOtherTypesText/more_than_are_shown`、`TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout` |
| M3 | `.claude/skills/director/SKILL.md` | 「wait で待つ」の重なりの箇条に「新しい wait が動き出したら、古い wait の Monitor を TaskStop で止める。」の 1 行を足す(挿入だけ) | `TestDirectorSkill_RunsOneInboxWaitAtATime` だけ。落ちた行は止める指示の確かめ(225 行、`"MonitorをTaskStopで止める"`)で、順の確かめは通った |
| G1 | `escalate.go` | `WaitInbox` の句を外す(読んだ一覧を `""` に) | `TestWaitInbox_TypesTimeoutNamesTheOpenItemsOfOtherTypes` |
| G2 | `director_wait.go` | detail に外の件を足さない(`if false && others != ""`) | `TestDirectorWaitTimeoutDetail`、`TestWaitInboxForDirector_TypesTimeOutWhileOnlyAnAlertIsOpen`、ほか 2 つ |
| G3 | `escalate.go` | 句のための読みを `openInboxItems(types)` にする(数えた種類しか読まない) | `TestReadInboxWaitOtherTypesText_ReadsTheOpenItemsOrNothing` ほか 4 つ |
| G4 | `escalate.go` | TYPE のエスケープを外す | `TestInboxWaitOtherTypesText`(制御文字の行) |
| G5 | `escalate.go` | 境目を `<` にする(5 件ちょうどで切る) | `TestInboxWaitOtherTypesText`(5 件の行) |
| P1 | `.claude/skills/director/SKILL.md` | 「重なったら、古いほうの Monitor に TaskStop をかけて終える。」の 1 行を足す(穴を測る変異) | 落ちない(`-run '^TestDirectorSkill'`、85 の `=== RUN`) |
| P2 | 同じ | 「重なったら、古いほうは TaskStop で片づける。」の 1 行を足す(穴を測る変異) | 落ちない |

P1 と P2 は Test gaps の 1。implementer の 38 個の変異(S1 で 10、S2 で 28)とは別に回した。

## Test gaps

1. `directorSkillStopsAWait` は字句で見る。`Monitor` か `wait` のすぐ後に「を」「も」「は」が来て、決まった動詞の語幹(止め、終わらせ、停止、打ち切、kill、TaskStop)が続くときだけ見つける。「古いほうの Monitor に TaskStop をかけて終える」「古いほうは TaskStop で片づける」のような言い換えは見つけない(P1、P2)。今の skill にこう書いた文はなく、4 写しは `check-skill-sync.sh` でそろうので、手で書き換えたときにレビューで見る範囲になる。LOW
2. 句のための読みは、wait の最後の読みのあとにもう 1 回受信箱を読む。そのあいだに外の件が resolve される場合は、テストの doc comment のとおり、テストが手を入れる場所がないので確かめていない(句がないか、resolve された件が載るかのどちらか。どちらも時間切れの扱いは変わらない)
3. ビルドした `ralph` の binary での `--director` の時間切れは回していない。CLI の層(cobra)では上の probe で確かめた
4. issue #374 の本当の原因(別の repo の台帳の e144・e145)はテストの外。計画の Rollout のとおり、ユーザーに確かめてもらう

## Verdict

- Pass: run-test.sh(full)rc 0、1,403 s。shell 46 本と Go 11 パッケージがすべて通った。足した 12 関数と固定を変えた 4 関数は `-count=1 -v` で PASS、SKIP 0。重なった wait のテストは `-race -count=5` で 10/10、data race 0。依頼の変異 3 つはすべて落ちた(ほかの 5 つも落ちた)
- Fail: 0
- Blocked: 0
- Skipped: 2(shell の条件つきの 2 つ。どちらもこの変更と関係しない)
- Insight event: `./scripts/insights-append.sh --slug director-wait-types-note --flow standard --phase test --verdict pass --cycle auto --source skill` で 1 行を足し、このレポートと同じコミットに入れた
