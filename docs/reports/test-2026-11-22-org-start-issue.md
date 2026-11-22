# Test report: org-start-issue

- Date: 2026-11-22(実行は 09:01:48〜09:31:42 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-org-start-issue.md`(issue #365)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `8b6bb6a3`(worktree は clean。origin/main `ccdf6293` を 81530fb0 で取り込み済み)。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `pgrep -fl 'scripts/run-test.sh|run-verify.sh'` を見た。別の worktree(`org-xreview-own-model`)の run-verify.sh が走っていたので、それが終わって何も返さなくなってから打った。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で 168 s ずつ待った。`nohup` は使っていない。範囲は wrapper の既定(`Requested scope: changed`、`Language scope: changed (changed_languages)`)で、走った pack は golang だけ。そのあと、変異 46 個、AC2 の base との突き合わせ、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-22-org-start-issue.log`(6,250 行あまり。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、1 分ごとの load、変異の diff と落ちたテスト、AC2 の突き合わせ、coverage の要約、`~/.codex/config.toml` と herdr のプロセスの前後が入っている。wrapper 自身の log は `docs/evidence/verify-2026-11-22-000148.log`(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。下の行) | 09:01:48〜09:31:42 JST、1,794 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0。stub の古い git の行は通る) | 約 13 分(09:01:48〜09:15 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 16 分(09:15 ごろ〜09:31:42)、下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.773 s |
| `internal/cli` | ok | 992.272 s |
| `internal/config` | ok | 18.989 s |
| `internal/gitroot` | ok | 4.277 s |
| `internal/insights` | ok | 1.633 s |
| `internal/org` | ok | 984.668 s |
| `internal/org/driver` | ok | 3.416 s |
| `internal/org/protocol` | ok | 1.933 s |
| `internal/scaffold` | ok | 3.059 s |
| `internal/toolchain` | ok | 24.959 s |
| `internal/upgrade` | ok | 4.556 s |

Go の段の始まりは記録していない。`internal/cli` の 992 s を終わりの時刻から引くと 09:15 ごろになる。`internal/cli` と `internal/org` は golang pack の 1 パッケージの上限 30 分のおよそ 55% で終わった。`(cached)` のパッケージは 1 つもなかったので、`-count=1` の打ち直しは要らなかった。

1 分 load は run の全体で 4.8〜23.7(09:01:50 から 1 分ごとに 30 回測った)。Go の段(09:15 ごろ〜)は 8.8〜23.7 で、最も高かったのは 09:19:52 の 23.0 と 09:28:52 の 23.7。私の変異の run のうち M1〜M9 は shell の段(09:02〜09:12)に回したが、R1〜R6 と F1(09:12〜09:26)は Go の段と重なった。Go の段の load を押し上げた分があるが、落ちたテストはなく、打ち直しもしていない。残りの変異と coverage の run は、run-test.sh が終わったあと(09:32 以降)に回した。

shell の段の `PASS` の行は 5,390 で、split-shared-paths の cycle 2 の run(`docs/reports/test-2026-11-22-split-shared-paths.md`)と同じ数だった。この branch は shell のテストを足していない。

run の前後で、`~/.codex/config.toml` の mtime(1795174910)、大きさ(38,144 バイト)、sha256(`0f076aef…`)は同じだった。書かれていない。読まれたかどうかは測っていない(この machine では読みで atime が変わらない)。足したテストのファイルに `codex`、`HOME`、`config.toml` を読むところはない(`org_start_issue_test.go:133` の `codexPromptPathFor` は台帳の下の prompt のファイルを読む)。herdr のプロセス(`herdr server` の pid 59855 ほか 2 つ)も前後で同じで、本物の herdr のサーバーには触れていない。

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestFeatureLeaderTask_IssueLine`、`TestStartFeature_RecordsTheIssue`、`TestScopeReservedDetails_IssueRoundTrip`、`_IssueTokenLast`、CLI の `TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask`(fake の herdr) | PASS。変異 M7・F1・F2・F4・F5・C3・C4 で落ちる |
| AC2 | `TestFeatureLeaderTask_IssueLine`(issue なしのタスクの sha256 を固定)、`TestStartFeature_NoIssueIsTheStartOfBefore`、`TestScopeReservedDetails_NoIssueIsTheRecordOfBefore`、`TestReseat_NoIssueLeavesTheLeadersTaskWithoutIt`、既存の `TestFeatureLeaderTask` ほか(引数に `""` を足しただけで、期待の文は変えていない) | PASS。下の「AC2 の base との突き合わせ」で、テストが固定する sha256 を base で計り直した |
| AC3 | `TestStartFeature_IssueFormRefused`、`TestValidateFeatureBinding_Issue`、CLI の `TestOrgStartIssue_RefusedWithoutPlanOrBlank`、`_MalformedNumberRefusedWithNothingMade` | PASS。変異 M8・M9・F3・R4・C1・C2・C9 で落ちる |
| AC4 | `TestReseat_IssueLineInTheLeadersTask`、`TestPlanReseat_StartHintNamesTheIssue` | PASS。変異 S1・S2 で落ちる |
| AC5 | `TestStartFeature_IssueRestart`(同じ issue、どちらもなし、違う番号、片方だけ)、`TestReservationDecision_IssueOwnOrg`、`TestSameFeature_Issue` | PASS。変異 M1・R2・R3・R6 で落ちる |
| AC6 | `TestFeatureBinding_IssueRead`(数でない、2 回、空、先頭の 0、`#`、符号、全角、11 桁、note のあと、`shared=` の前後)、`TestScopeReservedDetails_OlderBinaryReadsSevenTokens` | PASS。変異 M6 で落ちる |
| AC7 | `TestOrgStatus_IssueLineAndJSON`(issue あり、10 桁、なし、読めない値 3 種、壊れた記録)、`TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask` の status の部分 | PASS。変異 C5〜C8 で落ちる |
| AC8 | `TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue` | PASS。leader の雛形の変異 L1〜L5 で落ちる |
| AC9 | `TestDirectorSkill_StartsAFiledIssueWithItsNumberAndChecksTheIssueClosed`、`TestOrgSkill_DescribesStartIssueStatusLineBindingAndReseat`、`check-sync.sh` と `check-skill-sync.sh`(local verifier の中の `scripts/verify.local.sh`) | PASS。skill の 4 写しへの変異 K1〜K7 で落ちる。文書の中身そのものは verify の報告が確かめている |
| AC10 | `TestMergeOrder_BindingIssue`、CLI の `TestOrgMergeOrder_JSONBindingHasTheIssue`、既存の `TestMergeOrder_ResultJSONShape` | PASS。変異 G1〜G3・R1 で落ちる |

足したテスト(`*_issue_test.go`、`org_start_issue_test.go`、`prompts_leader_issue_test.go`、`skill_issue_test.go`)は、run-test.sh のあとに関数の名前を並べた `-run` と `go test -count=1 -v` で回し直した。`internal/org` の 6 ファイルの 23 関数(subtest 33)と、`internal/cli` の 1 ファイルの 6 関数(subtest 29)がすべて PASS で、SKIP は 0 だった。skill を読むテストはファイルがないと skip するが、ここでは skip していない。

gate と regate のテスト(`sameFeature` を使う `gate.go:773` と `regate.go:1050`)は、run-test.sh の `internal/org` の中で通った。そのあと coverage の run(`-run '...|Gate|Regate'`、500.593 s)でも `-count=1` で通った。

### AC2 の base との突き合わせ

`TestFeatureLeaderTask_IssueLine` は、issue なしのタスクの sha256 を `f6426197…` に固定し、それを 81530fb0(この変更の前)で計った値だと言う。これを別に確かめた。81530fb0 の `git archive` の写しに、テストと同じ fixture で旧い引数の `featureLeaderTask` を呼ぶ使い捨てのテストを足し、HEAD でも同じものを overlay で足して(引数に `""` を足しただけ)、値を比べた。

| 比べたもの | 81530fb0 | HEAD |
| --- | --- | --- |
| `issueTaskFixture` のタスク(前文、共有のパス、依存、本文つき)の sha256 | `f6426197684a840b…` | 同じ。テストの固定値とも同じ |
| 依存も共有のパスもない機能、reviewer の値つき、`per-plan` のタスクの sha256 | `22e2024b097dfea8…` | 同じ |
| 同じ機能の `split` のタスクの sha256 | `ef6575ab8e13936e…` | 同じ |
| 共有のパスと note つきの `scopeReservedDetails` | `... feature_digest=abcdef012345 shared=go.sum,pnpm-lock.yaml restored: why` | 同じ |

### 変異

変異の写しと overlay の JSON はセッションの scratchpad の下の自分のディレクトリに置き、`go test -count=1 -overlay <json>` で回した。置き換える文字列はファイルの中に 1 回だけあることを確かめてから替えた。どの変異も `build failed` は 0 件だった。M5 だけは nil の参照で panic して落ちた(nil を渡す行がそれを確かめている)。ほかはテストの assert で落ちて rc 1 になった。変異の前の対照は、`internal/org` の絞り込み(`-run 'Issue|SameFeature|FeatureBinding|ReservationDecision|ScopeReserved|StartFeature|ValidateFeatureBinding|Reseat|MergeOrder'`)が rc 0(33 s)、`internal/cli` の `-run 'OrgStart|OrgStatus|OrgMergeOrder'` が rc 0(112 s)だった。leader の雛形は go:embed なので overlay で替わる。skill は実行時にディスクから読むので、HEAD の `git archive` の写しで `internal/org` のテスト binary を 1 つ作り、写しの skill を 1 か所ずつ替えて `-test.run 'TestOrgSkill|TestDirectorSkill'` で回した。毎回元に戻して `cmp` で HEAD と同じことを確かめ、最後に対照を回して rc 0 を見た。

| # | 変異 | 落ちたテスト |
| --- | --- | --- |
| M1 | `sameFeature` が issue を比べない(`\|\| a.Issue != b.Issue` を消す) | `TestSameFeature_Issue`、`TestReservationDecision_IssueOwnOrg` の 3 行、`TestStartFeature_IssueRestart` の 3 行。gate と regate の結びつきが変わったときのテスト 4 つだけで回すと残る(Test gaps の 1) |
| M2 | 共通の比べ `sameFeatureExceptDigestsAndIssue` から `held.Complete()` を消す | `TestSameFeature`、`TestSameFeatureExceptDigestsAndIssue` |
| M3 | 共通の比べから承認方式を消す(切り出しの前の比べにあった項目) | `TestSameFeature`、`TestSameFeatureExceptDigestsAndIssue`、`TestReservationDecision_Feature` の 3 行、`TestStartFeature_RecordsApprovalMode` の 3 行、`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` |
| M4 | `sameFeature` から `!b.Complete()` を消す | `TestSameFeature` |
| M5 | 共通の比べから `want != nil` を消す(切り出しの前の `featureDigestRefusal` にあった nil の確かめ) | `TestSameFeatureExceptDigestsAndIssue`(panic) |
| M6 | 記録の読みが issue の形を確かめない(`#` だけ捨てる) | `TestFeatureBinding_IssueRead` の 9 行 |
| M7 | Details に `issue=` を書かない | 10 関数(round trip、token の並び、古い binary の読み、start の記録と立て直しと形の拒否、reseat、立て直しの案内、予約の判定、merge-order) |
| M8 | `validIssueNumber` が 11 桁を通す | `TestFeatureBinding_IssueRead/eleven_digits`、`TestStartFeature_IssueFormRefused`、`TestValidateFeatureBinding_Issue` |
| M9 | `validIssueNumber` が先頭の 0 を通す | `TestFeatureBinding_IssueRead` の 2 行、`TestStartFeature_IssueFormRefused`、`TestValidateFeatureBinding_Issue` |
| R1 | `IssueNumber` が 16 進で読む | `TestFeatureBinding_IssueNumber`、`TestMergeOrder_BindingIssue` |
| R2 | `featureRestartRefusal` が digest の句だけを返す(両方の句のとき) | `TestReservationDecision_IssueOwnOrg/the_feature_digest_and_the_issue` |
| R3 | `featureIssueRefusal` が常に `""` | `TestReservationDecision_IssueOwnOrg` の 4 行、`TestStartFeature_IssueRestart` の 3 行 |
| R4 | `validateFeatureBinding` が issue の形を確かめない | `TestValidateFeatureBinding_Issue` |
| R5 | `String` が issue を名指さない | `TestFeatureBinding_StringNamesTheIssue` |
| R6 | `reservationDecision` が `featureRestartRefusal` の代わりに `featureDigestRefusal` を呼ぶ | `TestReservationDecision_IssueOwnOrg` の 4 行、`TestStartFeature_IssueRestart` の 3 行 |
| F1 | start の結びつきに issue を入れない(タスクには入る) | `TestStartFeature_RecordsTheIssue`、`_IssueRestart` の 3 行、`_IssueFormRefused`、`TestReseat_IssueLineInTheLeadersTask` |
| F2 | start のタスクに issue を渡さない(結びつきには入る) | `TestStartFeature_RecordsTheIssue`、`_IssueRestart/the_same_issue:_issue_#12`、`TestReseat_IssueLineInTheLeadersTask` |
| F3 | `readStartFeature` が issue の形を確かめない(あとで spawn の `validateFeatureBinding` が断る) | `TestStartFeature_IssueFormRefused`(分割計画を読む前に断ること、何も作らないことを見ている) |
| F4 | タスクの行の `Closes` を `Fixes` に | `TestFeatureLeaderTask_IssueLine`、`TestStartFeature_RecordsTheIssue`、`TestReseat_IssueLineInTheLeadersTask`、`TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue` |
| F5 | issue の行を依存の行の前に置く | `TestFeatureLeaderTask_IssueLine`、`TestStartFeature_RecordsTheIssue` |
| S1 | reseat の leader の start に issue を渡さない | `TestReseat_IssueLineInTheLeadersTask` |
| S2 | 座席のない org の reseat の案内に `--issue` を付けない | `TestPlanReseat_StartHintNamesTheIssue` |
| G1 | merge-order の結びつきに issue を入れない | `TestMergeOrder_BindingIssue` |
| G2 | merge-order の JSON の鍵を `issue_number` に | `TestMergeOrder_BindingIssue` |
| G3 | merge-order の JSON の `omitempty` を消す | `TestMergeOrder_BindingIssue`、`TestMergeOrder_ResultJSONShape` |
| C1 | `--plan` のない `--issue` を Args で断らない | `TestOrgStartIssue_RefusedWithoutPlanOrBlank` の 5 行 |
| C2 | 空白だけの `--issue` を空と読まない(`TrimSpace` を消す) | 同じテストの空白とタブの 2 行 |
| C3 | `runOrgStartPlan` が `StartFeature` に issue を渡さない | `TestOrgStartIssue_MalformedNumberRefusedWithNothingMade` の 10 行、`_RecordedOnTheBindingAndTheLeadersTask` |
| C4 | RunE が `orgStartPlanParams` に issue を渡さない | C3 と同じ(10 行と 1 関数) |
| C9 | RunE が `checkOrgStartPlanInput` に issue の代わりに `"x"` を渡す | `TestOrgStartIssue_RefusedWithoutPlanOrBlank` の空の 4 行 |
| C5 | status の行の `#` を消す | `TestOrgStatus_IssueLineAndJSON` の 3 行 |
| C6 | status の JSON に issue を入れない | 同じ 3 行 |
| C7 | status の issue の行の条件を逆に | `TestOrgStatus_IssueLineAndJSON` の 7 行、既存の `TestOrgStatus_FeatureLineOnlyForBoundReservation`、`TestOrgStatus_IncompleteFeatureBinding` |
| C8 | status の JSON の `omitempty` を消す | `TestOrgStatus_IssueLineAndJSON` の 4 行、既存の 2 関数 |
| L1 | 雛形の手順 14 の `Closes #<番号>` を `Fixes #<番号>` に | `TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue` |
| L2 | 雛形に「行がなければ Closes を書かない。」を挿し込む(ほかの文は変えない) | 同じテスト。「Closes を禁じる文はない」の確かめで落ちた |
| L3 | 「普通の文(コードブロック、HTML のコメント、引用の外)」を消す | 同じテスト |
| L4 | 「`__ISSUE__` を本文に残さない」を「残す」に | 同じテスト |
| L5 | 「行がなければ…`Related issue:` など)で決める」を「では決めない」に | 同じテスト |
| K1 | director skill(`templates/base/.claude`)の出力先の 3 から `<その issue の番号>` を消す | `TestDirectorSkill_StartsAFiledIssueWithItsNumberAndChecksTheIssueClosed` |
| K2 | director skill(`.agents`)の merge のあとの `gh issue view` から `--repo github.com/<owner>/<repo>` を消す | 同じテスト |
| K3 | director skill(`.claude`)の「issue を閉じずに」を「閉じて」に | 同じテスト |
| K4 | director skill(`templates/base/.agents`)の引き継ぎの「閉じずに、」を消す | 同じテスト |
| K5 | org skill(`.claude`)の拒否の「11 桁以上」を「12 桁以上」に | `TestOrgSkill_DescribesStartIssueStatusLineBindingAndReseat` |
| K6 | org skill(`templates/base/.claude`)の「reseat は記録の番号で leader を立てる」を「issue なしで」に | 同じテスト |
| K7 | org skill(`.agents`)の「10 桁まで」を「12 桁まで」に | 同じテスト |

46 個のうち生き残ったものはない。M1 は、`sameFeature` を使う gate と regate のテスト 4 つ(`TestGate_OrgChangedWhileTheScriptsRan`、`TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan`、`TestRegateCI_NothingRecordedWhenTheBranchChangedWhileTheChecksWereRead`、`TestRegate_RecordsWithTheOrgOfTheLatestBinding`)だけで回すと rc 0 で残る。start と予約の判定のテストで落ちる。

## Coverage

- Statement: 狭い run(`internal/org` は `-run 'Issue|SameFeature|FeatureBinding|ReservationDecision|ScopeReserved|StartFeature|ValidateFeatureBinding|Reseat|MergeOrder|RenderRolePrompt_Leader|Gate|Regate'` で 55.1%、`internal/cli` は `-run 'OrgStart|OrgStatus|OrgMergeOrder'` で 11.5%)。パッケージ全体の数字ではない
- Changed lines: `git diff -U0 81530fb0 HEAD` の本番のコード(6 ファイル)の足した行を profile の block に重ねた。文を含む行 121 行はすべて通った(`reserve.go` 57、`cli/org.go` 41、`feature.go` 10、`reseat.go` 7、`org_merge_order.go` 5、`mergeorder.go` 1)。通っていない行は 0。コメントと宣言だけの行は数えていない
- Function: 足した関数と変えた関数は、`validIssueNumber`、`IssueNumber`、`sameFeature`、`sameFeatureExceptDigestsAndIssue`、`featureRestartRefusal`、`featureIssueRefusal`、`describeIssue`、`featureBindingFromTokens`、`scopeReservedDetails`、`validateFeatureBinding`、`featureDigestRefusal`、`readStartFeature`、`startFeatureLeaderParams`、`featureLeaderTask`、`reseatNoActiveSeatErr`、`collectMergeTargets`、`checkOrgStartPlanInput`、`printStatusJSON` がどれも 100%。`newOrgStartCmd` 97.9%、`runOrgStartPlan` 96.0%、`printStatusTable` 96.6% で、届かない分はこの変更の外の分岐
- Branch: Go の標準ツールでは測れない
- Notes: 変更した行がすべて通ることは、上の変異で、行が実行されるだけでなく結果が確かめられていることと合わせて見た

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh は 1 回目で rc 0 だった。打ち直しはしていない。既知の flake(`TestGate_TwoGatesRunTheirScriptsOneAtATime`、`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval` など)は、この run では 1 つも出ていない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `--issue` なしの start のタスクと Details(AC2) | この変更の前とバイト単位で同じ | 上の「AC2 の base との突き合わせ」の 4 行。既存の `TestFeatureLeaderTask` ほか 10 ファイルの変更は、`featureLeaderTask` の引数に `""` を足したこと、help と skill の文の固定に `--issue` の文を足したことだけで、期待のタスクの文は変えていない |
| 同じ機能かの比べ(`sameFeature` と 2 つの拒否の句の 3 つの写しを `sameFeatureExceptDigestsAndIssue` 1 つにした、52e6cfd3) | 前と同じ項目を比べる | 承認方式を消す M3、`held.Complete()` を消す M2、nil の確かめを消す M5 が、既存のテストと足したテストで落ちる |
| `issue=` のない古い記録と、`shared=` のある記録 | 前と同じに読む | `TestFeatureBinding_IssueRead` の「no token」「no token, with shared paths」、`TestScopeReservedDetails_OlderBinaryReadsSevenTokens`。M6・M7 で落ちる |
| gate と regate の「scripts を回している間に org が変わった」の判定 | 通る | run-test.sh と coverage の run の両方で、`-count=1` の gate と regate のテストが通った |
| issue のない org の `ralph org status` と `merge-order --json` | 前と同じ | C7・C8・G3 で既存の `TestOrgStatus_FeatureLineOnlyForBoundReservation`、`TestOrgStatus_IncompleteFeatureBinding`、`TestMergeOrder_ResultJSONShape` が落ちる。今の木ではどれも通る |
| skill の 4 写しの一致 | 一致 | local verifier の中の `check-sync.sh` と `check-skill-sync.sh` が通った |

## Test gaps

1. gate と regate が「scripts を回している間に結びつきが変わった」と読む判定(`gate.go:773`、`regate.go:1050`)は `sameFeature` を使い、この変更で issue の違いもそこに入った。issue だけが変わる場合を固定するテストはない(変異 M1 は gate と regate のテスト 4 つでは残る)。disband していない org の issue は start が変えさせないので、ふつうには起きない。起きるのは手で書いた記録か、古い binary の自己終了の補償が `issue=` のない結びつきを書き戻したときで、そのとき gate は「it is now bound to …」と言って記録しない。それが望む動きかは verify の報告(INFO、版が混ざる期間の gate の差)が書いていて、この報告では判断しない
2. leader が本当に PR の本文の普通の文に `Closes #<番号>` を書くか、GitHub がその merge で issue を閉じるかは、テストで確かめていない。テストが固定するのは雛形の文と、タスクの issue の行の形だけ。director の skill の merge のあとの確かめ(`gh issue view` で state を読み、`OPEN` なら人に伝える)も、文の断片を固定しているだけで、本物の gh と issue では回していない
3. `ralph org start --plan --issue` を本物の herdr で立てたことはない。CLI のテストは fake の herdr で、`ralph org reseat` のテストも org の層の fake で回っている
4. run-test.sh は Go のテストを `-v` なしで回すので、条件つきの `t.Skipf` が走ったかは見えない。足したテストは run-test.sh のあとに `-count=1 -v` で回し、SKIP が 0 であることを確かめた。skill の読みのテストは HEAD の `git archive` の写しでも 0 件の SKIP だった
5. 変異 R1〜R6 と F1 は、run-test.sh の Go の段と重なって回した。load を押し上げた分はあるが、run は落ちず、打ち直しもしていない

## Verdict

- Pass: 1(`./scripts/run-test.sh` が rc 0、1,794 s。local verifier 46 本と `go test ./...` の 11 パッケージがすべて通った。`(cached)` は 0。足したテスト 29 関数は `-count=1 -v` でも SKIP 0 で通った。変異 46 個はどれも落ちた。AC2 は base で計った値と一致した)
- Fail: 0
- Blocked: 0
- Verdict: pass
- Insight event: `./scripts/insights-append.sh --slug org-start-issue --flow standard --phase test --verdict pass --cycle auto --source skill` で cycle 1 の 1 行を足し、このレポートと同じコミットに入れた。UTC の日付が変わっていたので、行は前の 2 行のある `2026-11-21-org-start-issue.jsonl` ではなく、新しい `docs/insights/events/2026-11-22-org-start-issue.jsonl` に入った(同じ slug の file が日付ごとに分かれる例はほかにもある)
