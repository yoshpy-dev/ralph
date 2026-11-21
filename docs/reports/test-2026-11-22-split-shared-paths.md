# Test report: split-shared-paths

- Date: 2026-11-22(実行は 2026-11-22 03:01〜03:34 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `77a6961a`(base `9cd63d60`、worktree は clean)。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `pgrep -fl run-test.sh` で他の run がないことを確かめた。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定(`- Requested scope: changed`、`Language scope: changed (changed_languages)`)で、走った pack は golang だけ。そのあと、計画の Test plan が求める「共有の除外を外す変異で AC4 のテストが落ちること」を含む変異 11 個と、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-22-split-shared-paths.log`(6,100 行あまり。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、Go の段の 1 分ごとの load、変異 11 個の diff と落ちたテストと失敗の文、coverage の要約、`~/.codex/config.toml` と herdr のプロセスの前後が入っている。run-verify.sh は自分の記録として `docs/evidence/verify-2026-11-21-180133.log` も書いた(同じく追跡外)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-22 03:01:32 〜 03:30:47 JST、1,755 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,389 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0。stub の古い git の行は通る) | 約 14 分(03:01:33 〜 03:15 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 16 分(03:15 ごろ 〜 03:30:47)、下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.832 s |
| `internal/cli` | ok | 938.699 s |
| `internal/config` | ok | 19.812 s |
| `internal/gitroot` | ok | 2.963 s |
| `internal/insights` | ok | 1.237 s |
| `internal/org` | ok | 941.907 s |
| `internal/org/driver` | ok | 5.480 s |
| `internal/org/protocol` | ok | 2.123 s |
| `internal/scaffold` | ok | 3.039 s |
| `internal/toolchain` | ok | 22.590 s |
| `internal/upgrade` | ok | 4.866 s |

golang pack の 1 パッケージの上限は 30 分で、`internal/cli` と `internal/org` はどちらもその 52% で終わった。Go の段の 1 分 load は 7.3〜35.3(03:18 から 1 分ごとに 13 回測った)。私の変異の run(M1〜M9)と coverage の run は shell の段(03:03〜03:15)に重ねた。M9 の終わり(03:14:59)は Go の段の始まりと数秒重なったかもしれない。M10 と M11 は run-test.sh が終わったあと(03:32〜03:34)に回した。落ちたテストはなく、打ち直しもしていない。

run のあいだ、`~/.codex/config.toml` の mtime、大きさ、sha256 は前後で同じだった(書かれていない。読まれたかどうかは測っていない)。herdr のプロセス(`herdr server` の pid 59855 ほか 2 つ)も前後で同じで、本物の herdr のサーバーには触れていない。

### 計画のテストと AC の対応

| AC | テスト | 結果 |
| --- | --- | --- |
| AC1 | `TestLoadSplitPlan_Shared`(1 つ、複数、空白、正規化、囲みの中の `- Shared:` は本文)、`TestLoadSplitPlan_SharedRejects`(2 行、`.` と `./`、絶対パス、`..`、空の値と空の要素、機能の `- Reserve:` との重なり、予約が `.` の機能、ヘッダより後のフェンスの外の `- Shared:`) | PASS。変異 M8・M9 で落ちる |
| AC2 | `TestFeatureDigest_WithoutSharedIsUnchanged`(base で計った digest を固定)、`TestFeatureDigest_Encoding`、`TestFeatureDigest_Shared` | PASS。変異 M5 で落ちる |
| AC3 | `TestScopeReservedDetails_SharedRoundTrip`、`_SharedTokenLast`、`_OlderBinaryReadsSixTokens`、`TestFeatureBinding_SharedRead`(古い記録、壊れた値)、`TestStartFeature_RecordsSharedPaths` | PASS。変異 M11 で落ちる |
| AC4 | `TestWatch_SharedScope_ChangesToTheSharedPathsRaiseNothing`(新しい共有のファイル、共有のディレクトリの深い下、commit、共有のディレクトリの下の rename、予約と共有を 1 周で)、`_AnAlertNamesOnlyThePathsOutsideTheReservationAndTheSharedPaths`、`_AnOrgWithoutSharedPathsAlertsOnThemAsBefore` | PASS。共有の除外を外す変異 M1・M2 で落ちる(計画の Evidence の項目) |
| AC5 | `TestReservationDecision_Shared` の「the same shared paths」「a shared directory holding another org's shared directory」、`TestStartFeature_SharedAdmission` の「another split plan that shares the same paths starts alongside」、既存の予約どうしの重なりのテスト | PASS。共有どうしも断る変異 M10 で落ちる |
| AC7 | `TestReservationDecision_Shared`(ファイルとファイル、ファイルとディレクトリ、ディレクトリとディレクトリ、同じ prefix の別ディレクトリを両方の順で)、`TestStartFeature_SharedAdmission`(`StartFeature` と素の `Spawn --reserve` の両方の順) | PASS。向きごとの変異 M3・M4 で落ちる |
| AC8 | `TestFeatureLeaderTask_SharedPaths`、`_SharedScopeIsOneShellWord`、`TestStartFeature_SharedPathsLineAtStartAndRestart`、`TestReseat_SharedPathsLineInTheLeadersTask`、`reseat_shared_test.go` の 3 つ、`TestRenderRolePrompt_Leader_FeatureOrgSharedPaths` | PASS。start の `--scope` の変異 M7、reseat の `--scope` の変異 M6 で落ちる |
| AC6 | `skill_shared_paths_test.go` の 3 つ(director skill の書き方、org skill の直し方、skill の例の読み)、`check-sync.sh` と `check-skill-sync.sh` | local verifier の中で通った(`scripts/verify.local.sh` の check-sync と check-skill-sync)。文書の中身は /test の範囲の外で、verify の報告が確かめている |

計画の Integration tests の「`ralph org start --plan` で共有のパスが予約の記録に入り、見張りの 1 周で共有のパスの変更に ALERT が立たない」は、1 つのテストではなく 2 つのつながりで確かめられる。見張りのテストの fixture(`newBoundScopeFixture`)は本物の `Spawn` で `shared=` を記録し、見張りはその記録から共有のパスを読む。M11(記録に `shared=` を書かない)で `StartFeature` のテストと見張りのテストがそろって落ちるので、記録から見張りまでがつながっていることを確かめた。

### 変異

変異の写しと overlay の JSON は scratch(セッションの scratchpad の下の自分のディレクトリ)に置き、`go test -count=1 -overlay <json> ./internal/org/ -run <絞り込み>` で回した。置き換える文字列はファイルの中に 1 回だけあることを確かめてから替えた。どの変異も `build failed` は 0 件で、テストが落ちて rc 1 になった。変異の前に、同じ範囲を overlay なしで回した対照(`-run 'Scope|ReservationDecision|Reserve|StartFeature|Digest|Split|Reseat|FeatureLeaderTask'`)は rc 0 だった(58 s)。

| # | 変異 | 絞り込み | 落ちたテスト |
| --- | --- | --- | --- |
| M1 | 見張りに共有のパスを渡さない: `watch.go` の `r.Paths, r.Feature.SharedPaths(), snap, now)` を `r.Paths, nil, snap, now)` に替えた | `Scope` | 2 関数、5 subtest。`ChangesToTheSharedPathsRaiseNothing` は「after a new shared file」で `pnpm-lock.yaml` の ALERT が立ち、`AnAlertNamesOnly…` は ALERT が `other/x.go` のほかに共有のパスも名指した |
| M2 | 「外」の判定に共有のパスを足さない: `watch_scope.go` の `scopeOutsideEntries(slices.Concat(reserved, shared), …` を `scopeOutsideEntries(slices.Clone(reserved), …` に替えた | `Scope` | M1 と同じ 2 関数、5 subtest |
| M3 | 新しい側の共有と相手の予約を比べない: `reserve.go` の `reserveOverlaps(want.Feature.SharedPaths(), held.Paths)` を `reserveOverlaps(nil, held.Paths)` に替えた | `ReservationDecision\|Reserve\|StartFeature_Shared` | 2 関数、5 subtest。`TestReservationDecision_Shared` の共有が予約に重なる 3 行、`TestStartFeature_SharedAdmission` の予約の org が先の 2 行 |
| M4 | 新しい側の予約と相手の共有を比べない: `reserveOverlaps(want.Paths, held.Feature.SharedPaths())` を `reserveOverlaps(want.Paths, nil)` に替えた | 同上 | 2 関数、6 subtest。`TestReservationDecision_Shared` の 4 行、`TestStartFeature_SharedAdmission` の共有の org が先の 2 行 |
| M5 | 共有のパスがなくても digest に欄を足す: `split.go` の `featureDigest` の `if len(shared) > 0 {` を `if true {` に替えた | `Digest\|Split` | 2 関数、6 subtest。`TestFeatureDigest_WithoutSharedIsUnchanged` は「the digest it had before the - Shared: line existed」と違う値を報告した |
| M6 | reseat の座席の `--scope` に共有のパスを入れない: `reseat.go` の `slices.Concat(ActiveReservation(events, orgID), binding.SharedPaths())` を `slices.Clone(ActiveReservation(events, orgID))` に替えた | `Reseat` | 3 関数、2 subtest。`reseat_shared_test.go` の 3 つ |
| M7 | leader のタスクの `--scope` に共有のパスを入れない: `feature.go` の `slices.Concat(f.Reserve, plan.Shared)` を `slices.Clone(f.Reserve)` に替えた | `FeatureLeaderTask\|StartFeature` | 3 関数。`TestFeatureLeaderTask_SharedPaths`、`_SharedScopeIsOneShellWord`、`TestStartFeature_SharedPathsLineAtStartAndRestart` |
| M8 | ヘッダより後の `- Shared:` を断らない: `split.go` の `if !fenced && strings.HasPrefix(line, splitSharedPrefix) {` の頭に `false &&` を足した | `Split\|Digest\|Skill` | 1 関数、7 subtest。`TestLoadSplitPlan_SharedRejects` の前文、機能の節、`## Features` の最初の `###` より前、`## Features` より後の節、閉じたフェンスのあと、空の値、ヘッダにもある場合 |
| M9 | フェンスの中の `- Shared:` も断る: 同じ行の `!fenced &&` を外した | 同上 | 1 関数、2 subtest。`TestLoadSplitPlan_Shared` の前文と機能の節のフェンスの中の行 |
| M10 | 共有どうしの重なりも断る: `reservationOverlapErr` の `return nil` の前に、共有と共有を比べて断る判定を足した | `ReservationDecision\|Reserve\|StartFeature\|FeatureBinding\|ScopeReservedDetails\|Watch_SharedScope` | 2 関数、3 subtest。`TestReservationDecision_Shared` の「the same shared paths」「a shared directory holding another org's shared directory」、`TestStartFeature_SharedAdmission` の「another split plan that shares the same paths starts alongside」 |
| M11 | 記録に `shared=` を書かない: `scopeReservedDetails` の `if f.Shared != "" {` の頭に `false &&` を足した | 同上 | 11 関数、19 subtest。Details の往復と並びの 3 つ、受け付けの 3 つ、`StartFeature` の記録と立て直しの 3 つ、見張りの 2 つ |

M1〜M9 は run-test.sh の shell の段に重ねて回し、M10 と M11 は run-test.sh が終わったあとに回した。M10 は AC5 の「共有どうしで断らない」が、M11 は AC3 の記録と、記録から見張りまでのつながりが、テストで固定されているかを見るために足した。

## Coverage

- Statement: 狭い run の profile で測った(`go test -count=1 -coverprofile ./internal/org/ -run 'Scope|ReservationDecision|Reserve|StartFeature|Digest|Split|Reseat|FeatureLeaderTask|Shared|RenderRolePrompt_Leader|Skill'`、162 s、パッケージ全体の 35.6%)。変更した関数は `parseSplitPlan`、`parseSplitShared`、`finishSplitPlan`、`featureDigest`、`scopeReservedDetails`、`sharedFromToken`、`SharedPaths`、`reservationDecision`、`reservationOverlapErr`、`featureDigestRefusal`、`describeSharedPaths`、`featureLeaderTask`、`scopePathInside`、`scopeOutsideEntries` が 100%、`evaluateReservedScope` 95.0%、`PlanReseat` 94.7%、`featureBindingFromTokens` 94.7%、`validateFeatureBinding` 71.4%
- Changed lines: `git diff -U0 9cd63d60 HEAD` の `internal/org` と `internal/cli` のテスト以外の `.go` の追加行を、上の profile の block と突き合わせた。実行される文を持つ追加行 125 行がすべて実行され、実行されない追加行は 0 行。`internal/cli/org.go` と `internal/org/seat.go` の変更は help の文とコメントだけで、実行される文はない
- Branch: Go の標準ツールでは測れない
- Notes: 100% に届かない 4 関数の残りは、狭い run に当たるテストがない既存の枝で、追加行ではない(上の突き合わせ)。help の文の変更のうち、`ralph org start --help` の立て直しの条件は `internal/cli/org_feature_test.go` が固定する。`--reserve` の help の「or the shared paths of a running feature org's split plan」を固定するテストはない(Test gaps の 2)

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh は 1 回目で rc 0 だった。既知の flake(`TestGate_TwoGatesRunTheirScriptsOneAtATime` の gate.lock の待ちの持ち主、#357 で直したもの。ほかに `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` など)は、この run では 1 つも出ていない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `- Shared:` のない分割計画の、計画の digest と機能の digest(既存の計画の承認と、止めた機能の org の立て直し) | 変わらない | `TestFeatureDigest_WithoutSharedIsUnchanged` が base で計った値を固定し、M5 で落ちる。verify の報告が base の符号化から独立に計算した `1f1742b5c65c` と同じ |
| 予約どうしの重なりの判定と、その断りの文 | 変わらない | 既存の `reserve_test.go` と `TestReservationDecision_Shared` の「paths that overlap a reservation」が通る |
| 共有のパスのない org の見張りの scope_change | 変わらない | `TestWatch_SharedScope_AnOrgWithoutSharedPathsAlertsOnThemAsBefore` と既存の `watch_scope_test.go` が通る。M1・M2 でもこのテストは通る(対照として正しい向き) |
| 共有のパスのない計画の leader のタスク | 1 バイトも変わらない | `TestFeatureLeaderTask_SharedPaths` が、共有のパスが nil の計画と空の計画のタスクを、`feature_shared_task_test.go` の固定した文字列と比べて通る |
| `shared=` のない古い記録の読み | `Shared` が空 | `TestScopeReservedDetails_OlderBinaryReadsSixTokens`、`TestFeatureBinding_SharedRead` が通る |
| 既存の split、reserve、feature、watch、reseat のテスト | 通る | run-test.sh の `internal/org` が ok(941.907 s) |
| gate.lock の待ちの持ち主の flake(#357) | 出ていない | run-test.sh の `internal/org` が 1 回目で ok |

## Test gaps

1. 古い binary の自己終了の補償が binding を書き戻すと、`feature_digest=` と同じく `shared=` も落ち、その org の共有のパスは「なし」と読まれる。これを確かめるテストはない。verify の報告の Coverage gaps の 9(tech-debt の「Two gaps of the feature digest of a restarted feature org」(b))と同じもので、この PR では払っていない
2. `ralph org start --reserve` の help(`internal/cli/org.go` の `orgReserveFlagUsage`)に足した「or the shared paths of a running feature org's split plan」の句を固定するテストはない。verify の V3 と同じ指摘で、文が消えても気づけない
3. 壊れた `shared=` の値は「共有のパスなし」と読まれる(`TestFeatureBinding_SharedRead` が読みの側を固定する)。その読みのまま見張りが共有のパスの変更に scope_change を上げることを、見張りの 1 周で確かめるテストはない。読みの関数と見張りの関数はそれぞれ固定されているので、2 つをつなぐ 1 本が足りないだけである
4. 統合の確かめは、`StartFeature` の記録のテストと、本物の `Spawn` で記録する見張りの fixture の 2 つに分かれている。`StartFeature` から見張りの 1 周までを 1 本で回すテストはない。M11 で両方がそろって落ちるので、今のテストでもつながりの切れ目は見つかる
5. この branch の base は `9cd63d60` で、そのあと main に PR #368(`10e9322d`)が merge された。この run は main を取り込む前の HEAD で回した。`/pr` の前に main を取り込んだら、run-test.sh を回し直す必要がある(`feature.go` は別のレーンも触る、と計画の Risks が書いている)

## Verdict

- Pass: 1(`./scripts/run-test.sh` が rc 0。local verifier 46 本と `go test ./...` の 11 パッケージがすべて通った。AC1〜AC8 に対応するテストが通り、変異 11 個はすべて落ちた)
- Fail: 0
- Blocked: 0
- Verdict: pass
- Insight event: `./scripts/insights-append.sh --slug split-shared-paths --flow standard --phase test --verdict pass --cycle auto --source skill` で 1 行を足し、このレポートと同じコミットに入れた
