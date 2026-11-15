# Test report: reseat-leader-progress

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

## Verdict

- Pass: yes(`==> All verifiers passed.`、rc 0)
- Fail: なし
- Blocked: なし
