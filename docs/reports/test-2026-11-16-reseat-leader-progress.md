# Test report: reseat-leader-progress

## Cycle 5 (cap 5)

- Date: 2026-11-16
- HEAD `02497a99`(run 7 の cross-review の直し 8ed3fe97 と、cycle 5 の self-review の直し 866f6ab9 のあと。変わったのは `internal/org/prompts/leader.md` とそのテストだけ)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-c5.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 11:20:00〜11:40:04 JST(負荷 3.8〜5.0) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 598 s、org 702 s、config 11.5 s |

run 7 からの差: 受信箱の項が codex の hook の QUESTION を待たない例外と、その見分けを `ralph org inbox show` の本文でする文(`TestRenderRolePrompt_Leader_ReseatRuleInboxItem`、120 を `reseatProgressLineRunes` と結ぶ)、座席の行が無い座席の画面に作業のあとが見えれば TASK を送らず BLOCKED を上げる文と、立ち上がりの 1 行を `PromptFilePointer` と結ぶ確かめ(`..._SeatCheckItem`、`..._SeatCheckUnknownTaskID`)が通った。直しのコミットの前に、直す前の leader.md を `go test -overlay` で当て、新しいテストで落ちることを確かめた(計画の Progress checklist)。

## Verdict

cycle 5 の判定(前の回の判定は下の各節にある)。

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Run 7 (cycle-count 4, cap 5)

- Date: 2026-11-16
- HEAD `bdb516ce`(cycle 4 の cross-review の直し 3a5922e3 と、run 7 の self-review の直し 9e00043f のあと。変わったのは `internal/org/prompts/leader.md` とそのテストと、`internal/org/reseat_progress.go` のコメントだけ)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-run7.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 10:03:32〜10:23:38 JST(負荷 3.4〜5.4) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 592 s、org 698 s、config 10.0 s |

cycle 4 からの差: 入力待ちの座席の入力欄に前の send の残りがあれば何も送らず BLOCKED を上げる規則と、空の入力欄の案内文を残りに数えないこと(`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckSendsNothingOverLeftoverInput`。規則が名指す案内文が `codexComposerPlaceholders` のすべてと `sendClaudePlaceholder` が合う形であることも見る)と、書き換えた `..._SeatCheckItem` が通った。直しのコミットの前に、直す前の leader.md を `go test -overlay` で当て、新しいテストで落ちることを確かめた(計画の Progress checklist)。

### Verdict(run 7)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Cycle 4 (cap 4)

- Date: 2026-11-16
- HEAD `655bfd10`(run 5 の cross-review の直し 4096ae88、cycle 4 の self-review の直し 70b1511c、verify の直し fda4c415 のあと。変わったのは `internal/org/reseat_progress.go`、`internal/org/prompts/leader.md` と、そのテスト 2 つ)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-c4.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 08:51:10〜09:11:22 JST(負荷 2.8〜5.3) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 598 s、org 695 s、config 11.3 s |

run 5 からの差: branch の commit を origin と手元の既定のブランチのどちらにも無いものに限ること(`TestReseatProgress_CommitsLeaveOutTheLocalDefaultBranchAheadOfOrigin`、worktree に `HEAD` という名前のファイルがある場合を含む)、手元の main が origin より進んだ最初の start に節が入らないこと(`TestReseatProgress_StartFeatureAddsNothingForAnOrgWithNoProgress` の新しい行)、形の違う gate や cross-review の TASK についての reviewer の QUESTION を手順 12 と「cross-review」の小節の項へ回す規則の文と、回し先の項の文(`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckReadsOnlyThisTasksReplies`)、計画の項目の `git log` の範囲(`TestRenderRolePrompt_Leader_ReseatRulePlanItem`)が通った。直しのコミットの前に、直す前の `reseat_progress.go` と leader.md、`--` の無い形を `go test -overlay` で当て、新しいテストで落ちることを確かめた(計画の Progress checklist)。

### Verdict(cycle 4)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Run 5 (cycle-count 3, cap 4)

- Date: 2026-11-16
- HEAD `dff7bfb2`(cycle 3 の cross-review の直し 5ab60578、run 5 の self-review の直し 262b9df7、verify の直し 8a6676e4 のあと。変わったのは `internal/org/prompts/leader.md` とそのテストだけ)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-run5.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 07:27:08〜07:47:17 JST(負荷 5.3〜6.0) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 597 s、org 693 s、config 9.7 s |

cycle 3 からの差: 座席の確認で、数える返事を履歴の時刻が座席の行の時刻と同じかそれよりあとのものに限ること、返事がないときに何かを送る前に座席の QUESTION を探して DECISION で答えること(不明の行の枝も含む、履歴は境の時刻に届くまで数を増やして読む)を固定するテスト(`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckReadsOnlyThisTasksReplies` と、書き換えた `..._SeatCheckItem`、`..._SeatCheckNeverDiscardsAReply`、`..._SeatCheckUnknownTaskID`)が通った。各直しのコミットの前に、直す前の leader.md と変異を `go test -overlay` で当て、新しいテストで落ちることを確かめた(計画の Progress checklist)。

### Verdict(run 5)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Cycle 3

- Date: 2026-11-16
- HEAD `ee9e41dc`(3 回目の cross-review の直しと、cycle 3 の self-review・verify の直しのあと。変わったのは `internal/org/prompts/leader.md` とそのテストだけ)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-c3.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 05:49:09〜06:09:47 JST(負荷 3.5〜4.0) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 602 s、org 704 s、config 10.3 s |

run 3 からの差: 座席の確認の規則(見つけた返事を捨てない、済んだと確かめられる手だけ飛ばす、済んだかが分からない直しは重なっても頼む)を固定するテストと、外した 2 つの例(時刻を比べて飛ばす、直しの commit で飛ばす)が戻らないことのテストが通った。

### Verdict(cycle 3)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Run 3 (cycle-count 2, cap 3)

- Date: 2026-11-16
- HEAD `c49a6bc5`(cross-review の cycle 2 の直し d85fc0ef のあと)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-run3.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 04:22:15〜04:42:56 JST(負荷 3.7〜4.1) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 601 s、org 708 s、config 10.4 s |

cycle 2 からの差: d85fc0ef の新しいテスト(200 byte で切れた TASK_ID を不明にする 4 つの場合、短い TASK_ID は本文が切れても残ること、gate の項目の PR の head の分岐、扱い済みの返事を読み飛ばす条件、不明の TASK_ID の扱い)がすべて通った。cycle 2 の sync-docs で変えた `ralph org reseat --help` の文とその固定も、この実行で通った。

### Verdict(run 3)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Cycle 2

- Date: 2026-11-16
- HEAD `46b7993c`(cross-review の cycle 1 の直し ec10ba6f と、cycle 2 の self-review の直しのあと)。inline の実行で、理由は cycle 1 と同じ
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress-c2.log`(git の追跡外)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(cycle 1 と同じ、前からある条件つきの skip) | 2026-11-16 02:26:46〜02:48:31 JST(負荷 5〜7) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 654 s、org 761 s、config 11.4 s、toolchain は cache |

cycle 1 からの差: ec10ba6f の新しいテスト(座席ごとの最後の TASK を `sent` の記録から取ること、Send が書く前置き 3 種をはがすこと、200 byte の切り詰めで TYPE と TASK_ID が残ること、上げた RESULT を全状態で新しい 5 件、QUESTION の行の PLAN)と、leader.md の規則の新しい分岐(TASK_ID の突き合わせ、PR の状態の 3 分岐と RESULT の確かめ、計画助言の確かめ)の固定がすべて通った。cycle 2 の self-review の直し(最後の TASK の行の書式、レビューと gate の TASK の頼み直し)の期待値も通った。

Test gaps(cycle 2 で足したもの): leader が規則の新しい分岐(PR の状態、RESULT の有無、計画助言の有無)をどう読むかは文の固定だけで、実機の leader では確かめていない(cycle 1 の「実機の動き」の穴と同じ)。

### Verdict(cycle 2)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし

## Cycle 1

- Date: 2026-11-16
- Plan: docs/plans/active/2026-11-15-reseat-leader-progress.md(issue #312 の #312c-2)
- Cycle: 1
- Tester: orchestrator の inline 実行。この日の opus のサブエージェントは 2 回止まった(週の上限と、編集の前の応答の停止)ので、subagent-policy の Fallback に従ってこのセッションで `/test` を回した
- Scope: `./scripts/run-test.sh`(変更のある言語の範囲。golang と local の verifier、Go は `go test ./...`)。HEAD `1a1d8981`(origin/main の 48a8a661 を取り込み済み)
- Evidence: `docs/evidence/test-2026-11-16-reseat-leader-progress.log`(git の追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `GOFLAGS=-timeout=60m ./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-16 00:10:12〜00:32:38 JST(負荷 4.5〜5.6) |
| local verifier(`tests/*.sh` 45 本) | 45 本すべて | 全部 | 0 | browser detection、git 2.41 より古い本物の git | — |
| `go test ./...`(11 パッケージ) | 全パッケージ ok | 全部 | 0 | — | cli 677 s、org 770 s、toolchain 25.7 s、config 12.2 s |

上限を 60 分にしたのは、負荷の高い手元で internal/org と internal/cli が既定の 30 分に近づくことがあるため(#311 の test report)。今回の負荷では 15 分を下回った。

Skipped(どれも前からある条件つきの skip で、この branch の変更とは関係しない):

- `tests/test-plan-visual.sh` の browser detection(/Applications のブラウザが起動するのを避ける)
- `tests/test-secret-scan-branch.sh` の、git 2.41 より古い本物の git の場合(手元は git 2.49.0)

## 新しいテスト(28 本)

- `internal/org/reseat_progress_test.go`(節を作る関数): 各項目を 1 つずつ、全項目の順、経過の無い org は節なし、今の命の外を除く、読めない gate の記録は前の記録へ、片付いていない受信箱の件(org、state、disband の時刻、offset の違う時刻、読めない時刻)、21 件以上の件と commit、座席や人が書いた文の制御文字と長さ、本文の `## ` と backtick が fence を抜けない、commit の読み取りの失敗、承認の行の digest が 1 語、`log.showSignature=true` でも 1 commit 1 行(偽の gpg)
- `internal/org/reseat_progress_start_test.go`(`StartFeature` と reseat に組み込んだ形): 立て直しの leader の prompt に承認・gate・件・commit が載る、経過の無い start は今のタスクのまま、git が失敗しても start は続く、10 秒で応えない git を止める、受信箱が読めない行、reseat の id と leader のあとに起動する座席、2 回目の reseat が 1 回目を前の立て直しとして載せる
- `internal/org/prompts_leader_reseat_test.go`(leader.md の規則): 置き場所(承認方式の段落と「次の順で進めます。」の間)、本文は data、節の行の頭が `reseat_progress.go` の出す頭と合う、規則が指す手順の番号が本当にその手順である、座席の確認・計画・gate・受信箱・座席の各項目の文

各 slice の implementer が `go test -overlay` で変異を当てた: S1 19 個、S2 10 個、S3 13 個、self-review の 1 回目の直し 3 個、2 回目の直し 3 個が、それぞれ新しいテストで落ちた(計画の Progress checklist)。

## 計画の Test plan との対応

| Test plan の項目 | テスト |
| --- | --- |
| Unit: 節を作る関数の表(各項目、空、今の命の外、resolved、別の org、disband より前、21 件以上、制御文字、長い件名、`## ` と fence 文字) | `TestReseatProgress_ASectionHoldsEachPartAlone`、`_AnOrgWithNoProgressHasNoSection`、`_OnlyTheCurrentLifeCounts`、`_InboxItemsOfTheCurrentLifeThatAreNotSettled`、`_MoreThanTwentyItemsAndCommitsAreCounted`、`_ValuesFromPeopleAreOneCleanLine`、`_AHeadingOrBackticksInATextDoNotLeaveTheFence` |
| Integration: `StartFeature` の立て直しに節、新しい start に節なし、reseat の id と座席、本物の git の commit と総数 | `TestReseatProgress_StartFeatureTellsARestartedLeaderWhatTheOrgHolds`、`_StartFeatureAddsNothingForAnOrgWithNoProgress`、`_ReseatNamesItselfAndTheSeatsAfterTheLeader`、`_ASecondReseatNamesTheFirstAsThePreviousOne` |
| Regression: `TestFeatureLeaderTask` の完全一致、前文の prompt、`- 分割計画:` で始まるタスク、reseat の既存のテスト | 全パッケージの実行で通った |
| Edge: 受信箱が nil・壊れた行、読めない gate の Details、base の無い worktree、git の時間切れ、offset の違う時刻、読めない時刻 | `_StartFeatureSaysWhenTheInboxCannotBeRead`、`_AnUnreadableGateRecordFallsBackToThePreviousOne`、`_StartFeatureCarriesOnWhenGitFails`、`_StartFeatureStopsAGitThatDoesNotAnswer`、`_InboxItemsOfTheCurrentLifeThatAreNotSettled` |
| Prompt tests: 規則の各項目と AC9 の分岐 | `TestRenderRolePrompt_Leader_Reseat*` の 9 本 |

## Failure analysis

なし(1 回で通った)。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 経過の無い start の leader のタスク(今と同じ) | pass | `TestReseatProgress_StartFeatureAddsNothingForAnOrgWithNoProgress`、`TestFeatureLeaderTask*` |
| #318(PR #334)が手順 4 の書き出しを変えた件 | pass | 312793a7 で手順 4 の固定を「先頭の 2 行を書き換える」文に直した。`TestRenderRolePrompt_Leader_ReseatRuleStepsAreTheStepsItMeans` |
| 取り込んだ #335(codex の貼り付け)のテスト | pass | 全パッケージの実行 |

## Test gaps

- disband のあとの `start --plan` で、承認が無く commit だけの節が入る場面を、`StartFeature` を通して確かめるテストは無い。部品(今の命の境目、commit だけの節)は別々のテストが見ている(verify の V7)
- commit や受信箱の読み取りの失敗の行は、節がほかの理由で在るときだけ載る(計画の D6・AC7 のとおりで、意図した形。verify の V8)
- 節を読んだ leader の実機での動き(herdr の座席)は確かめていない
- coverage の数値は取っていない

### Verdict(cycle 1)

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし
