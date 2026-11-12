# Test report: split-feature-digest

- Date: 2026-11-12(run-test.sh は 19:26:29 から 19:58:09 JST、追加の回は 19:30 から 20:30 JST)
- Plan: `docs/plans/active/2026-11-12-split-feature-digest.md`(`## Test plan`、AC1〜AC11)
- Tester: tester サブエージェント(`/test`、pipeline の cycle 1。`cycle-count.json` は `{"cycle": 1}`)
- Scope: `git diff 9ac72c50...HEAD`(HEAD `e0105040`、19 ファイル、+1491/-211)。コードの commit は S1 `e3eeb643`、S2 `80545503` と `06683a35`、S3 `5a749c9b` と `34d74a8d`。run-test.sh は changed scope のままで golang を選んだ(`Language scope: changed (changed_languages)`)。worktree の production code とテストには手を入れていない。変異と probe は `go test -overlay` で当て、scratch は `/private/tmp` の下に置いた
- Evidence: `docs/evidence/test-2026-11-12-split-feature-digest.log`(`.gitignore` の `docs/evidence/*.log` に当たるのでコミットしない)。先頭に run-test.sh の全出力(wrapper のログは `docs/evidence/verify-2026-11-12-102630.log`)、その後ろに coverage、`-race`、変異の定義と結果、probe のソースと出力
- 起動の仕方: run-test.sh は Bash の 600 秒の上限を超えるので、`perl` の fork と `setsid` で切り離し(INT・HUP・QUIT・TERM は DEFAULT、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。shell の SKIP の行は前の PR の回と同じ 2 行だった

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(rc 0、`==> All verifiers passed.`) | shell 45 本 + Go 10 パッケージ | shell 45 本 OK(PASS を含む行 5,195)、Go 10 パッケージ ok | 0(`--- FAIL` の行 0) | 2 行 | 1,900 秒 |
| └ Go: `internal/org` | 上位 1,668 本 | ok | 0 | — | 1,043.1 秒 |
| └ Go: `internal/cli` | 上位 1,088 本 | ok | 0 | — | 751.6 秒 |
| └ Go: `internal/config`、`cmd/ralph`、`internal/gitroot`、`internal/insights`、`internal/org/driver`、`internal/org/protocol`、`internal/scaffold`、`internal/upgrade` | — | ok。`(cached)` は 0 | 0 | — | 28.0、1.0、13.7、3.8、4.4、4.5、1.8、7.5 秒 |
| `-race -count=3 -v`、org: 下の正規表現の 109 本(この diff が足したか変えたテストと、同じ道を通るもの) | 上位 327 回(subtest 込みの PASS 行 1,650) | 1,650 | 0 | 0 | 61 秒。`DATA RACE` 0 |
| `-race -count=3 -v`、cli: `^(TestOrgStatus_\|TestOrgStartAndStatusHelp_FeatureDigest$\|TestOrgStartPlan_\|TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart$\|TestOrgStartHelp_TaskSource$)` | 上位 90 回(PASS 行 282) | 282 | 0 | 0 | 158 秒。`DATA RACE` 0 |
| coverage: org と cli を名前で絞った回(下の Coverage) | org 109 本、cli 絞り込み | ok | 0 | — | 23 秒、117 秒 |
| 変異: production code の 48 個(下の節) | 48 | 42 個が死んだ | — | — | 絞った回で 1 個 14〜123 秒 |
| probe 1: 生き残った 6 個を落とす行(overlay で足した 1 本、コミットしない) | 6 行 × HEAD と 6 個 | HEAD で 6 行とも PASS、6 個はそれぞれ自分の行だけで FAIL | — | — | 1 回 1〜2 秒(build を除く) |
| probe 2: R12c が `StartFeature` から届くか(overlay で足した 1 本、コミットしない) | 1 本 × HEAD と R12c | HEAD で PASS、R12c で FAIL | — | — | 1 回 1 秒未満(build を除く) |

org の `-race` と coverage の回の正規表現は `FeatureDigest|SameFeature|FeatureBinding|ScopeReserved|Reservation|StartFeature|SpawnPrecheck|OrgSpawn_Feature|SplitPlan|OrgCloseDeferredSelf` である。`git diff -U0` の hunk に入る org のテスト関数 30 本(`split_test.go`、`reserve_test.go`、`feature_test.go`、`spawn_feature_test.go`、`verbs_test.go` の `TestOrgCloseDeferredSelf_CloseFails_BindingRestored`)をすべて含む。

上位の本数は `go test -list .` で数えた。Skipped の 2 行は前の PR の回と同じで、`tests/test-plan-visual.sh` の browser detection と、`tests/test-secret-scan-branch.sh` の git 2.41 より古い本物の git の case である。どちらも手元の環境によるもので、この diff と関係がない。

run-test.sh のあいだ、1 分の load は 8.6 から 21.3 だった。org は 1,043 秒で、30 分の上限の 58% だった。落ちたテストは無く、打ち直しはしていない。

## 受け入れ条件とテスト

どのテストも run-test.sh と上の `-race -count=3` の回で PASS した。「変異」の列は、そのテストが落とした変異の id(下の節)である。

| AC | テスト(ファイル) | 変異 |
| --- | --- | --- |
| AC1 機能ごとの digest の入力と、変わる編集・変わらない編集 | `TestFeatureDigest_ChangesOnlyWithTheFeatureAndPreamble`、`TestFeatureDigest_NoPreambleAndEmptyBody`、`TestFeatureDigest_Encoding`(`split_test.go`)。前文と自分の節の変更が start に効くことは `TestStartFeature_OwnSectionOrPreambleChangedRefused` | S01〜S11 がすべて死んだ。欄を 1 つずつ空にする 6 個、長さを書かない符号化、区切りの無い符号化、Depends on を並べ替える形、前後の空行を落とさない前文、Type の既定より前に計算する形 |
| AC2 `feature_digest=` を `approval=` の後ろに書き、読み戻す。2 つある token と空の値は欄を空にする | `TestScopeReservedDetails_FeatureDigestTokenLast`、`TestScopeReservedDetails_FeatureRoundTrip`、`TestFeatureBinding_FeatureDigestRead`(`reserve_test.go`)、`TestStartFeature_StartsLeaderInFeatureWorktree`、`TestOrgSpawn_Feature_RecordedWithReservation`、cli の `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart` | R01(書かない)、R02(`approval=` の前に書く)、R03(読まない)、R04(2 つある token の値を残す)、F01、F02 |
| AC3 `sameFeature` の比べ方。`Complete()` は 5 欄のまま | `TestSameFeature`、`TestFeatureBinding_OlderRecordStillBound`(D4: `ActiveFeature`、`gateOrgLife`、`collectMergeTargets`)、`TestReservationDecision_Feature` | R05〜R09 がすべて死んだ。`b` の Complete を外す、5 欄の比較を 1 つずつ外す、`&&` を `\|\|` にする、片側だけで機能ごとの digest を使う、機能ごとの digest に全体の digest も足す、比較を常に真にする 2 形 |
| AC4 ほかの機能の節だけを直して承認し直した計画で、止めた org を立て直せる。依存先の節でも同じ。承認の記録が効く | `TestStartFeature_OtherFeatureChangedRestarts`(2 つの subtest)、`TestSpawnPrecheckErr_MatchesSpawn` の「only another whole digest」の 2 行、`TestReservationDecision_Feature` の「only another whole digest passes」 | R08(全体の digest も比べる)、F01(機能ごとの欄に全体の digest を入れる) |
| AC5 前文かその機能の節を直した start は断られ、何も記録しない。拒否の文 | `TestStartFeature_OwnSectionOrPreambleChangedRefused`(節と前文)、`TestReservationDecision_Feature`、`TestOrgSpawn_Feature_DifferentBindingRefused` | R08b、R11、R12k(前と今の digest を入れ替える)、R13(文を足さない) |
| AC6 古い記録の org は、ほかの機能の節だけの変更でも断る。拒否の文 | `TestStartFeature_OlderRecordRefusedAfterOtherFeatureChanged`、`TestSpawnPrecheckErr_MatchesSpawn` の古い記録の行、`TestReservationDecision_Feature` の古い記録の 3 行 | R07、R09、R11b、R12j、R12l |
| AC7 承認し直さない start は `CheckApproved` で断られる | `TestStartFeature_OtherFeatureChangedRestarts` の中の「changed after its approval: approved digest」の確かめ | `CheckApproved` は diff に無いので変異は当てていない |
| AC8 古い binary も 5 欄と承認方式を読める | `TestScopeReservedDetails_OlderBinaryReadsFiveTokens` | R02 |
| AC9 `status --json` の `feature_digest`(古い記録では出ない)、`start` と `status` の `--help` | `TestOrgStatus_FeatureLineOnlyForBoundReservation`、`TestOrgStatus_IncompleteFeatureBinding`、`TestOrgStartAndStatusHelp_FeatureDigest`(`internal/cli/org_feature_test.go`) | C01(出さない)、C02(`omitempty` を外す)、C03(全体の digest を出す)。help は文を丸ごと比べるので変異は当てていない |
| AC10 gate.go と regate.go の既存のテストが変更なしで通る | `internal/org` の `gate*_test.go`、`regate*_test.go`、`mergeorder*_test.go`、`approval*_test.go`、`approve_test.go` の 13 ファイルは diff に無い。名前が `Gate\|Regate\|MergeOrder\|Approv` に当たるテストは org に 275 本、cli に 107 本あり、run-test.sh のパッケージ全体の回で通った | — |
| AC11 org skill の 4 つの写し | テストで固定する文は無い。写しの一致と文の中身は /verify が確かめた | — |

AC10 について。cli の `org_approve_test.go` は diff にあるが、変えたのは `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart` の 1 本で、`scope_reserved` の末尾の確かめを「`approval=<mode>` で終わる」から「`approval=<mode> feature_digest=<12 桁>` で終わり、`feature_digest=` は 1 つ」にした。末尾を正規表現で固定したままなので、assert は弱まっていない。org の既存のテストで期待を変えたもの(`spawn_feature_test.go` の 3 本、`verbs_test.go` の 1 本、`feature_test.go` の `TestSpawnPrecheckErr_MatchesSpawn` と `TestStartFeature_RecordsApprovalMode` ほか)も、Details の完全一致に `feature_digest=` を足したか、計画の AC4 のとおり「全体の digest だけが違う」を通す側へ変えたものである。自分の節が変わった場合を断る行は残っている。

計画の Test plan の Edge cases: 前文の無い分割計画と本文の空の機能は `TestFeatureDigest_NoPreambleAndEmptyBody`、最後の機能の後ろへの追加、CRLF、Depends on の並びの変更は `TestFeatureDigest_ChangesOnlyWithTheFeatureAndPreamble` の行にある。

## Coverage

- Statement: この diff が変えた Go のコード(テストを除く)のうち、cover profile の block に入る足した行 73 行をすべて通った(`git diff -U0 9ac72c50...HEAD` の足した行と、上の絞った回の profile を突き合わせた。`internal/org/reserve.go` 39/39、`split.go` 13/13、`feature.go` 1/1、`internal/cli/org.go` 20/20)。名前で絞った回なので、パッケージ全体の数字(org 27.2%、cli 27.8%)は意味を持たない
- Branch: Go の標準ツールに branch coverage は無いので測っていない
- Function: `featureDigest`、`finishSplitPlan`、`sameFeature`、`validateFeatureBinding`、`String`、`Complete`、`scopeReservedDetails`、`featureBindingFromTokens`、`reservationDecision`、`featureDigestRefusal`、`startFeatureLeaderParams`、`printStatusJSON` は 100%。`newOrgStartCmd`(86.0%)と `newOrgStatusCmd`(85.0%)の残りは、この diff が変えていない行である(上の足した行の突き合わせで、cli の足した行は 20/20)
- Notes: 行はすべて通るが、`featureDigestRefusal` の guard のうち 6 つは、外しても絞った回のテストが落ちなかった(下の節)。行の coverage では見えない穴で、変異で見つけた

## 変異と probe

production code に 48 個の変異を当てた。`split.go` に 11 個(S01〜S11)、`reserve.go` に 32 個(R01〜R13)、`feature.go` に 2 個(F01、F02)、`internal/cli/org.go` に 3 個(C01〜C03)。各変異は文字列の置き換え 1 つで、置き換える文字列がファイルに 1 回だけ現れることを確かめてから当てた。org の変異は上の org の正規表現、cli の変異は `Status|Feature|StartPlan|Help|ApprovalMode` で絞って回した。build に失敗した変異は無い。定義と全結果は evidence にある。

42 個が死んだ。生き残った 6 個は、すべて `featureDigestRefusal`(`reserve.go:652`)の guard を外す変異である。

| 変異 | 外したもの | 変異の下で起きること | start から届くか |
| --- | --- | --- | --- |
| R12b | `!held.Complete()` | digest だけが欠けた壊れた記録の org への start に、「its record has no feature digest (an older ralph started it)」の文が付く | 壊れた記録が要る。まれ |
| R12c | `held.Split != want.Split` | 別の分割計画の同じ slug の機能を同じ org_id で start すると、「the feature's section (### auth-core) or the preamble of split plan "auth-split" changed」と、元の分割計画の節が変わったように言う | 届く。org_id の既定は slug なので、2 つの分割計画が同じ slug と Type の機能を持てば、branch と worktree も同じになる。`StartFeature` を回す probe で確かめた(下) |
| R12d | `held.Feature != want.Feature` | 別の機能に、節が変わったという文が付く | branch が `<type>/<slug>` なので、R12e の guard が残る限り届かない(コードから読んだ) |
| R12e | `held.Branch != want.Branch` | 同じ機能の Type を変えた start に、節が変わったという文が付く | 届く。ただし Type は機能の節の一部なので、付く文はおおむね正しい |
| R12f | `held.Worktree != want.Worktree` | main worktree を動かしたあとの start に、節が変わったという文が付く | おそらく届くがまれ。未確認 |
| R12i | 古い記録の場合の `held.Digest != want.Digest` | 古い記録の org で、全体の digest が同じでパスだけが違う予約に、「every change to the split plan is refused」が付く | 届かない。全体の digest が同じならファイルも同じで、Reserve も同じになる |

どの変異でも、断るか通すかは変わらない。変わるのは拒否の文だけである。HEAD の文は正しい。

生き残った理由: `TestReservationDecision_Feature` には、別の分割計画・別の機能・別の branch・別の worktree・壊れた記録の行があり、どれも `wantNot` で「since the feature's section」と「older ralph」が文に無いことを見る。しかし、どの行も want の `FeatureDigest` が held と同じ(`testFeature` の `testFeatureDigest`)で、古い記録の行は全体の digest も同じである。そのため guard を外しても `featureDigestRefusal` の 2 つの case のどちらにも当たらず、文は足されない。`wantNot` の行は guard を固定していない。

probe(`internal/org/zz_probe_refusal_guards_test.go` を overlay で足した 1 本。コミットしない): 上の 6 つの場合を 1 行ずつ持ち、want の `FeatureDigest` を `0a1b2c3d4e5f` に変える(P1〜P4)、digest だけが欠けた壊れた記録を置く(P5)、古い記録に全体の digest が同じでパスの違う予約を当てる(P6)。どの行も、拒否の文に「since the feature's section」と「older ralph」が無いことを見る。HEAD では 6 行とも PASS した。R12b は P5、R12c は P1、R12d は P2、R12e は P3、R12f は P4、R12i は P6 だけで落ちた。R12c の下の P1 の文は次のとおりである(evidence に全文がある)。

```
... so the reservation internal/auth/ for feature "auth" of split plan "split-2" (...) is refused, since the feature's section (### auth) or the preamble of split plan "split-1" changed after org_id "org-bound" was started (feature digest "fedcba987654", now "0a1b2c3d4e5f"): ...
```

R12c が start から届くことは、2 本目の probe(`zz_probe_start_other_split_test.go`、overlay で足した。コミットしない)で確かめた。`newFeatureStart` の分割計画 `auth-split` の `auth-core` を start して leader を止め、前文だけを変えた別の分割計画 `other-split` の `auth-core` を同じ既定の org_id で start する。HEAD の拒否は「is refused: ralph org disband --org-id auth-core first, or use another org_id」で終わり、節の文は付かない(PASS)。R12c の下では「since the feature's section (### auth-core) or the preamble of split plan "auth-split" changed after org_id "auth-core" was started」が付いた(FAIL)。

パッケージ全体の回: 6 個をまとめて当てた回は、cli 全体では ok(552 秒)、org 全体では `TestReservationDecision_Feature/damaged:_the_binding_as_read_is_refused` だけで落ちた(749 秒)。この行の held と want はどちらも digest と branch が空なので、R12e と R12i を一緒に当てたときだけ古い記録の文が付く。1 個ずつでは、この行は guard の手前で止まる。そこで、R12b〜R12f の 5 個をまとめた回と R12i だけの回を、org 全体で回し直した。結果: どちらも org 全体で ok だった(5 個の回 807.5 秒、R12i の回 811.5 秒、2 つを並べて回した)。6 個は、絞った回の外のテストでも落ちない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh、`-race` の回、coverage の回で落ちたテストは無い。変異と probe の FAIL は、上の節の意図した落ち方である。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 自分の節が変わった start の拒否(`TestStartFeature_OtherBindingRefusedWithoutWorktree`) | Pass | run-test.sh の org。worktree を見る前に断り、何も記録しない |
| 承認方式の記録と、別の方式での start の拒否(PR #224 以降) | Pass | `TestStartFeature_RecordsApprovalMode`、`TestReservationDecision_Feature` の承認方式の行、cli の `TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart` |
| 自己終了に失敗した spawn の補償が binding を書き戻すこと | Pass | `TestOrgCloseDeferredSelf_CloseFails_BindingRestored`(期待の Details に `feature_digest=` を足しただけ) |
| `feature_digest=` の無い古い記録の org が gate・merge-order・status で bound のまま読まれること(D4) | Pass | `TestFeatureBinding_OlderRecordStillBound`、cli の `TestOrgStatus_FeatureLineOnlyForBoundReservation` の古い記録の行 |
| gate・regate・merge-order・approval | Pass | 変えていないテスト、org 275 本と cli 107 本(AC10) |

## Test gaps

1. `featureDigestRefusal` の guard 6 つ(R12b〜R12f、R12i)を固定するテストが無い。どれも拒否の文だけの穴で、断るか通すかは変わらない。start から届いて文が誤るのは R12c(別の分割計画の同じ slug の機能)で、probe 2 で確かめた。いちばん小さい足し方は、`TestReservationDecision_Feature` の「another split plan」「another feature」「another branch」「another worktree」の 4 行で want の `FeatureDigest` も変え、「digest だけが欠けた壊れた記録」と「古い記録で全体の digest が同じでパスの違う予約」の 2 行を `noDigestClause` 付きで足すことである。上の probe がその形で、HEAD で通り 6 個をそれぞれ落とすことを確かめた
2. 止めた leader の立て直しは、偽の herdr と偽の worktree で回した。実機の herdr と本物の `ralph-worktree.sh` では回していない(/verify の Coverage gaps と同じ)
3. `featureLeaderTask` に欄を足したら `featureDigest` にも足す約束は、両方の注釈にだけある。欄の増加を機械で捕まえるテストは無い(/verify と self-review の Coverage gaps と同じ)。足すなら、`featureLeaderTask` の出力に入る `SplitFeature` の欄と、`featureDigest` に入る欄を比べるテストがいちばん小さい
4. 古い binary の補償が binding を書き戻して `feature_digest=` を落とす道(計画の Risks の 5 つ目)は、本物の古い binary では起こしていない
5. AC11 の skill の文を固定するテストは無い。文書の範囲として /verify が読んで確かめた

## Verdict

- Pass: run-test.sh(shell 45 本、Go 10 パッケージ)、`-race -count=3` の 2 つの回、coverage の回のすべてが通った。AC1〜AC10 の挙動はテストが固定し、変異 48 個のうち 42 個が死んだ。生き残った 6 個は拒否の文の guard だけで(Test gaps 1)、当てた変異のうち断るか通すかを変えるものはすべて死んだ。AC11 は文書
- Fail: 0
- Blocked: なし
