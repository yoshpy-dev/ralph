# Test report: org-limits-reserve

- Date: 2026-10-09(JST。実行の記録は UTC の 2026-10-09 01:47〜02:10。ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、パイプライン 4 回目(cycle 4)。ユーザーが上限を 4 に上げた回で、`cycle-count.json` は /cross-review の決まりで 2 のままなので、この報告と insight event に cycle 4 と書く(`insights-append.sh --cycle 4`)
- Scope: HEAD 642e7291(テストを足した commit は 69cb7d19)。cycle 3 の test(104be58d)からのコードの差分は 83aec44e だけで、`internal/org/verbs.go` の `reserveAgain` を `releasedReservation` と `startsOrg` で書き直し、`CloseDeferredSelfPane` からも呼ぶ修正と、`verbs_test.go` の改名と追加のケース。ほかは文書の commit(skill の 4 面、tech-debt、plan、triage、self-review と verify の報告)。`git diff --stat 104be58d HEAD -- internal cmd scripts tests templates` が出したのは `verbs.go`、`verbs_test.go` と `templates/base/` の skill の 2 面だけで、`spawn.go`、`envelope.go`、`reserve.go`、`verbs_all.go`、`internal/cli/`、`internal/config/`、`scripts/`、`tests/` は変わっていない。behavioral test だけを実行した(静的解析は /verify の担当)。重点は、改名と追加のテストの安定性、lead が挙げた mutation、self-review C4-1 の入力(ユーザーは 2026-10-09 に今の挙動を残すと決めた)
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log` の末尾の「cycle 4」の節(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-09-014726.log`
- 番号の付け方: cycle 3 の T3-1〜T3-7 と mutation の番号(G、S、L、E、R、H、O、Q1、Q2)は付録 C に、cycle 2 の T2-1〜T2-4 と N1〜N14 は付録 B に、cycle 1 の Test gaps 1〜8 と M、X、P は付録 A に、原文のまま残す。tech-debt の行 172、176、178、179 がその番号を指しているため。この回の mutation は K1 から、仮の修正は Q4 と振り、Test gaps は T4-1 から振る

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(642e7291、テストを足す前) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | shell 1(環境による。下の注) | 396 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(テストを足したあと) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 82 s(`internal/cli` 80 s)、rc 0 |
| `go test -race ./internal/org/... -count=1`(足したあと) | 3 パッケージ | 3 | 0 | 0 | 17 s、rc 0 |
| `^TestOrgCloseDeferredSelf` を `-count=10`(足したあと、単独) | top-level 220(22 本 × 10)、subtest 560 | 780 | 0 | 0 | 5 s、rc 0 |
| 同じものを `-race -count=10` | top-level 220、subtest 560 | 780 | 0 | 0 | 8 s、rc 0 |
| 同じものを `-race -count=10` で、裏で `internal/cli` と `internal/org` の全体を流しながら | top-level 220、subtest 560 | 780 | 0 | 0 | rc 0(裏の実行も rc 0。`internal/cli` 73 s、`internal/org` 12 s) |
| mutation(`go test -overlay`。HEAD のテストのまま) | 27 件と仮の修正 Q4 | red 17 件 | 生き残り 10 件。Q4 も通った | - | 1 件 9〜16 s |
| mutation(足したテストを入れて、生き残りと K16、Q4 を流し直し) | 11 件と Q4 | red 8 件(HEAD のテストですでに red の K16 を含む)と Q4 | 生き残り 3 件(K1、K12、K14。どれも等価) | - | 1 件 13〜16 s |
| 確かめの probe(overlay で足しただけで commit しない) | 1 本 | 結果をログに出すだけ | - | - | 1 s 未満 |

- 足したテストは 4 本(69cb7d19。`verbs_test.go` に +251 行で、テストのファイルだけ)。`TestOrgCloseDeferredSelfPane_StopRetriedAfterForcedDisband_ReservationRestored`、`TestOrgCloseDeferredSelfPane_AfterDisband_StartedAgainWhileClosing`(subtest 2 つ)、`TestOrgCloseDeferredSelf_CloseFailsAgainOnRetry_ReservationRestoredAgain`(pane と workspace)、`TestReleasedReservation`(13 行の表)。理由は Test gaps の T4-1〜T4-5。commit する前に、同じ 4 本を overlay で足した状態で `internal/org` 全体が通ること(14.7 s、rc 0)と、mutation の流し直しに使ったものと tree に入れたものが同じ文字列であることを確かめた
- shell の件数は suite ごとに数えた。cycle 3 の `run-test.sh` と、39 本すべての pass・fail・skip の数が一致した(`diff` が空)。83aec44e は shell のテストに触れていない
- Skipped の 1 件は、cycle 3 と同じ `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)。`tests/test-pre-bash-guard.sh` の `SKIP: 0` は集計行なので数えていない
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し(15.2 s)、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した
- `-count=10` の正規表現 `^TestOrgCloseDeferredSelf` は、`internal/org` の 22 本に当たる。HEAD の 19 本(cycle 3 の 18 本から、`RulesDiffer` が `..._ReservationUnlessStartedAgain` に改名され、83aec44e が `TestOrgCloseDeferredSelfPane_CloseFails_ReservationOnlyAfterDisband` を足した)と、この回に足した 3 本。`TestReleasedReservation` は当たらない(2 行目の全体の実行に入っている)。`internal/cli` に同じ接頭辞のテストはない
- 依頼にあった `tests/test-secret-scan.sh` の flake は出なかった。失敗がないので、単独の再実行はしていない
- 実行のあと(テストの commit の前)、worktree の `git status --porcelain` は足したテストの 1 ファイルだけだった

## Coverage

- Statement: `internal/org` 93.3%(HEAD のテストでも 93.3%)。ほかの 7 パッケージは cycle 3 と同じ(`internal/cli` 85.3%、`internal/config` 92.8%、`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数。「HEAD のテスト」は `verbs_test.go` を HEAD の版に戻した overlay で測った値):

| 関数 | 位置 | HEAD のテスト | 足したあと | 通らない文 |
| --- | --- | --- | --- | --- |
| `reserveAgain` | `verbs.go:1324` | 100.0% | 100.0% | なし |
| `releasedReservation` | `verbs.go:1348` | 87.5% | 100.0% | HEAD のテストでは最後の `return 0, nil`(`disbanded` の前に立ち上げがない台帳、`:1358`)。足した表の「no start before the disbanded」が通す |
| `startsOrg` | `verbs.go:1364` | 100.0% | 100.0% | なし |
| `compensateUnderLock` | `verbs.go:1222` | 100.0% | 100.0% | なし |
| `CloseDeferredSelfPane` | `verbs.go:1023` | 100.0% | 100.0% | なし |
| `CloseDeferredSelfWorkspace` | `verbs.go:1106` | 100.0% | 100.0% | なし |
| (参考)`reservationBeforeLastDisband` | `reserve.go:311` | 100.0% | 100.0% | 下の注 |

- `reservationBeforeLastDisband` は本番では使われていない(self-review C4-3)。`grep` で呼び出しを探すと、定義と `reserve_test.go:323` の `TestReservationBeforeLastDisband` しかない。100.0% はそのテストだけによるもので、`go test -skip '^TestReservationBeforeLastDisband$' -coverprofile` で測ると 0.0% になる。ほかのテストと、テストが通る本番の経路のどれもこの関数に届かない
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 行の coverage は HEAD のテストでも依頼の関数のほとんどが 100.0% だったが、下の mutation では 10 件が生き残った。条件の片方や呼び出しの引数を変える mutation は、行がすべて通っていても落ちないことがある

## Mutation

各 mutation は、`internal/org/verbs.go` の写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off ./internal/org/` でパッケージ全体を流した(`-run` で絞っていない)。worktree のファイルは書き換えていない。1 回目は HEAD のテストのまま 28 件を流し、2 回目は足したテストを `zz_probe_test.go` として overlay に足して、生き残りと K16、Q4 を流し直した。K16 は最初の版が `d` の未使用で build に失敗したので、条件全体を `false && (...)` で包む形に直して流し直した(下の表は直した版の結果)。83aec44e は `internal/org` の外を変えていないので、`internal/cli` では流していない。

lead が挙げた 7 件(4 つめは K3 と K4 に、7 つめは K7 と K7b に分けた。理由は表の下):

| # | Mutation | HEAD のテスト | Red になった subtest(HEAD のテスト) | 足したテスト |
| --- | --- | --- | --- | --- |
| K1 | `reserveAgain` の `ActiveReservation(now, orgID) != nil` を外す(self-review C4-2) | 生き残り(等価) | なし | 生き残り(等価。下の注) |
| K2 | `startsOrg` の `EventScopeReserved` の枝を外す | **生き残り** | なし | red: 足した `CloseFailsAgainOnRetry` の pane と workspace、`TestReleasedReservation` の「started again: scope_reserved」 |
| K3 | `releasedReservation`(Step A)の `slices.ContainsFunc(events[d+1:], ...)` を外す | red | pane の表の「ordinary stop in a run started again without --reserve」 | - |
| K4 | `reserveAgain`(Step B)の `now[d+1:]` の走査を外す | red | 「started again without --reserve」、`_Stay` の「org started again and disbanded again」 | - |
| K5 | `now[d] != before[d]` を外す | red | 「manifest replaced with an event before the disbanded」 | - |
| K6 | pane の経路の `reserveAgain` の呼び出しを外す | red | pane の表の「disband deferred the pane」 | - |
| K7 | Step B で、`now` に増えた `disbanded`(打ち直しの disband)があれば戻さない | red | 「only disband run again」 | - |
| K7b | Step A で、`disbanded` が続くとき前の予約を引き継がない(`ActiveReservation(events[:d])`、a94c914f の前の規則) | red | 「disband run again before the close read the manifest」 | - |

依頼の「Step A's restart check」は、self-review の言い方(Step A は close の前の写しで戻す予約を決める段)では K3 と同じ式になる。そのため Step B の走査を外す K4 も流した。依頼の最後の「no-op の `disbanded` が戻すのを止める」は、Step B の側(K7)と Step A の側(K7b)の 2 つに分けた。

ほかの mutation:

| # | Mutation | HEAD のテスト | Red になった subtest | 足したテスト |
| --- | --- | --- | --- | --- |
| K4b | Step B の走査と `ActiveReservation(now)` を両方外す | red | 「started again without --reserve」と `_Stay` の workspace の 3 ケース | - |
| K5b | `len(now) <= d` を外す | red(panic) | 「manifest removed」で添字の範囲外(`index out of range [9] with length 0`) | - |
| K8 | `startsOrg` の `EventSpawnStarted` の枝を外す | **生き残り** | なし | red: 足した pane のテストの「another seat's spawn in flight」、表の「started again: spawn_started」 |
| K9 | `startsOrg` の `EventSpawned` の枝を外す | **生き残り** | なし | red: 表の「started again: spawned」だけ |
| K10 | `startsOrg` が dry-run の event も数える | **生き残り** | なし | red: 表の「a dry-run start after it」だけ |
| K11 | `startsOrg` がほかの org の event も数える | red | `ReservationRestored_RiskWindowClearedByRetry` | - |
| K1K2 | Step B で `ActiveReservation(now)` を外し、走査も `scope_reserved` を数えない | red | `_Stay` の「a spawn holds the manifest lock」 | - |
| K2A | Step A の走査だけ `scope_reserved` を数えない | **生き残り** | なし | red: 表の「started again: scope_reserved」だけ |
| K12 | `disbanded` の前に立ち上げがないとき、`d` の直前の予約を返す(元は nil) | 生き残り(等価) | なし | 生き残り(等価) |
| K13 | `d` の前の最初の立ち上げの時点の予約を取る(窓を切らない) | red | `..._ReservationRestored` の「a later run without --reserve」 | - |
| K14 | `disbanded` がない org でも走査に進む | 生き残り(等価) | なし | 生き残り(等価) |
| K15 | Step B の skip を黙って返す(skipped を `""` にする) | red | skip を見る 6 ケース | - |
| K16 | Step B の条件全体を外す(いつも戻す) | red | skip の 6 ケース | - |
| K17 | pane の経路の `reserveAgain` に、`now` の代わりに close の前の写しを渡す | **生き残り** | なし | red: 足した pane のテストの 2 ケース |
| K18 | workspace の経路の `reserveAgain` に写しを渡す(cycle 3 の S4 と同じ形) | red | skip の 6 ケース | - |
| K19 | pane の経路の `reserveAgain` に、`reactivateSeat` の追記のあとで読み直した台帳を渡す(self-review C4-4 の (c)) | red | pane の表の「disband deferred the pane」 | - |
| K20 | workspace の経路で同じことをする | red | `..._ReservationRestored` の「restored」、「only disband run again」など 4 ケース | - |
| K21 | pane の経路で `reserveAgain` を `reactivateSeat` より先に呼ぶ | red | pane の表の「disband deferred the pane」(エラー文と event の順) | - |
| K22 | pane の経路の `reserveAgain` に、写しの代わりに `now` を before として渡す | **生き残り** | なし | red: 足した pane のテストの 2 ケース |

仮の修正(足したテストが、C4-1 のもう一方の選択で落ちることの確かめ):

| # | 仮の修正 | HEAD のテスト | 足したテスト |
| --- | --- | --- | --- |
| Q4 | pane の経路で、座席の `stopped` が `disbanded` より後ろにあれば `reserveAgain` を呼ばない(self-review C4-1 と verify V4-1 の「戻さない」案) | すべて通る | red: 足した `StopRetriedAfterForcedDisband_ReservationRestored` だけ |

- 27 件の mutation のうち、HEAD のテストで red は 17 件、生き残りは 10 件だった。足したテストで 7 件(K2、K2A、K8、K9、K10、K17、K22)が red になり、残る 3 件(K1、K12、K14)は等価である
- K1 の等価(self-review C4-2 の読みのとおり): `now[d] == before[d]` なら、`now` の org の最後の `disbanded` は d 以降にある。`ActiveReservation(now)` が nil でないのは、その `disbanded` より後ろに real で org 単位の `scope_reserved` があるときで、それは `now[d+1:]` に入り、`startsOrg` は `scope_reserved` を数えるので走査が先に真になる。K1 と K2 はそれぞれ単独では Step B の結果を変えないが、K1K2(両方)は「a spawn holds the manifest lock」で落ちる。Step B では 2 つが互いを覆っている。`ActiveReservation(now)` の式を外す(C4-2 の推奨)と、Step B の `scope_reserved` の判定は走査だけが持つことになり、そのときは「a spawn holds the manifest lock」が固定する
- K12 の等価: `scope_reserved` は立ち上げに数えるので、`d` の前に立ち上げがなければ `d` の前に予約の記録もなく、`ActiveReservation(events[:d])` も nil になる
- K14 の等価: `disbanded` がないとき d は 0 で、走査が真でも偽でも、そのあとのループは `i = -1` から始まって何もせず、どちらも `0, nil` を返す。違うのは空の台帳で `events[1:]` が panic することだけで、2 つの close は台帳に座席か workspace の記録がないと先に戻るので、空の写しで呼ばれることはない
- tech-debt の行 176 の (a) の RESOLVED の括弧は、cycle 3 の G4(`ActiveReservation(now, orgID) != nil` を外す)が red になると書く。HEAD では同じ式を外す K1 は生き残る(等価)。HEAD で同じ窓を固定するのは Step B の走査(K4、K4b が red)と、K1K2 が示す 2 つの判定の組である。/sync-docs で書き直す材料になる
- K5b は panic でパッケージの実行が止まる。panic したのは狙いの「manifest removed」で、その前のテストは通っていた。止まったあとのテストは走っていないので、K5b の「red」は panic した 1 ケースだけの結果である

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 3 の V3-1: close を待つ間に同じ org の `disband` だけが打ち直されると、予約が戻らない | 直っている | 「only disband run again」と「disband run again before the close read the manifest」が通る。打ち直しの `disbanded` で戻すのを止める K7、K7b が red |
| cycle 1 の F-10: `disband` が自分の pane を後回しにして close が失敗すると、予約が戻らない | 直っている | pane の表の「disband deferred the pane」が通る。呼び出しを外す K6 が red |
| cycle 2 の F-5: 補償が close の前の写しで予約を書き戻し、立て直した org の新しい予約を古いものに替える | 保たれている | `_Stay` の「org started again with another reservation」と「a spawn holds the manifest lock」が通る。写しを渡す K18(workspace)が red。pane の経路の K17 は足したテストで red |
| 通常の `stop` は予約について何も書かない | 保たれている | pane の表の 2 ケースが通る。Step A の走査を外す K3 が red |
| `--force` は何も書かない | 保たれている | `TestOrgCloseDeferredSelf_Force_NoCompensation` と、足した C4-1 のテストの前半(`CloseDeferredSelfPane(..., true)` のあと台帳の長さが変わらない)が通る |
| plan のリスクの項目(ほかの org が枠か範囲を取った窓は、打ち直しの disband で解ける) | 保たれている | 変わっていない `ReservationRestored_RiskWindowClearedByRetry` が通る |
| AC1〜AC14 の判定 | 変わっていない | 判定のファイルは 104be58d から変わっていない(上の Scope)。shell の suite ごとの件数は cycle 3 と一致した。cycle 1〜3 の mutation(付録 A〜C)は流し直していない |

## Test gaps

この回に見つけたもの。cycle 3 の T3-1〜T3-7 は付録 C に、cycle 2 の T2-1〜T2-4 は付録 B に、cycle 1 の Test gaps 1〜8 は付録 A にある。T3-1(打ち直しの disband で予約が戻らない)と T3-2(`--reserve` なしの立て直しに古い予約が戻る)は 83aec44e で挙動が変わり、記録のテストは `..._ReservationUnlessStartedAgain` に改名されて新しい挙動を固定している。T3-3〜T3-7 は 83aec44e の差分に関係しないので変わらない。

- T4-1(self-review C4-1、verify V4-1 の 1 つめの入力): `disband --force` で自分の pane の close が失敗し、同じ pane から `stop` を打ち直してその close も失敗した場合のテストがなかった。ユーザーの決定(2026-10-09、今の挙動を残す)に合わせて `TestOrgCloseDeferredSelfPane_StopRetriedAfterForcedDisband_ReservationRestored` を足した。実行で、コードを読んだ結論のとおりだった。forced disband のあと補償は何も書かず予約は解けたまま、打ち直しの `stop` は `disbanded` の後ろに 2 つめの `stopped` を書いて pane を後回しにし、その close が失敗すると `[stopped, spawned, scope_reserved]` の順で座席と、forced disband が解いた予約が戻り、エラーは「recorded seat "leader" ... active and the reservation internal/auth/ ... again」で終わり、org-b の重なる予約は拒否される。「戻さない」案の仮の修正 Q4 はこのテストだけで落ちるので、テストは 2 つの選択を見分ける。この入力の窓(forced disband から打ち直しの `stop` まで)はオペレーター次第の長さで、その間にほかの org が同じ範囲を取った場合のテストはない(verify V4-1 の注と同じ)
- T4-2(K2): `startsOrg` の `scope_reserved` の枝を外しても HEAD のテストは通った。等価ではない。打ち直しの disband の close もまた失敗すると、2 回目の補償は「最後の立ち上げ」を探して、1 回目の補償が書いた `scope_reserved` に当たる(座席の復帰の `spawned` より後ろにある)。この枝がないと 1 つ前の `spawned` に当たり、その時点の予約は nil なので、2 回目は予約を戻さず、org-a は範囲を持たずに走っている扱いになる。エラー文は、コマンドを打ち直せば close をやり直すと書く。そのとおり打ち直して herdr がまた答えなければ、この入力になる。`TestOrgCloseDeferredSelf_CloseFailsAgainOnRetry_ReservationRestoredAgain` を pane と workspace の 2 経路で足して埋めた
- T4-3(K17、K22): pane の経路で、disband のあと close を待つ間に新しい記録が入るテストがなかった。`_Stay` の pane のケースは通常の `stop`(`disbanded` がない)なので、Step A が何も返さず、pane の呼び出しの引数を変える mutation がどちらも生き残った。workspace の経路にはあるテストが、もう 1 つの呼び出し元にはなかった。`TestOrgCloseDeferredSelfPane_AfterDisband_StartedAgainWhileClosing` の「leader spawned again without --reserve」を足して埋めた(座席も予約も戻らず、エラーは両方と手で閉じるコマンドを書き、org-b は同じ範囲を予約できる)
- T4-4(K8): spawn の途中(`spawn_started` だけがあり `spawned` はまだ)で close を待つ間を通るテストがなかった。同じテストの「another seat's spawn in flight」を足した。この入力では、leader は自分の最新の状態イベントが変わらないので戻り、予約は戻らない(seat-2 の spawn が org の新しい run を始めたため)。エラーは leader を戻したことと予約を戻さなかったことを書き、手で閉じるコマンドで終わる。座席と予約で結果が分かれる入力で、今の挙動の記録である
- T4-5(K9、K10、K2A): どれも足した `TestReleasedReservation` の表の 1 行だけで落ち、close の経路のテストでは見分けがつかない。K10 は、close を待つ間に同じ org の `ralph org spawn --dry-run` が打たれると挙動が変わるので CLI から届くが、その経路のテストは足していない。K9(`spawned` だけが `disbanded` の後ろにある)と K2A(写しの `disbanded` の後ろに `scope_reserved` だけがある)は、CLI から届く入力を見つけられなかった(読みによる推測で、未確認です)。表は関数の doc が書く規則(立ち上げに数える 3 種、dry-run とほかの org を数えないこと)を直接固定する。self-review C4-3 が挙げた 5 ケースもこの表に入っている
- T4-6(self-review C4-3): `reservationBeforeLastDisband` は本番で使われていない(上の Coverage の注)。そのテストの「held none in its last life」(`[予約, disbanded, disbanded]` に nil)は、`TestReleasedReservation` の「disbanded again」(同じ形の並びに `docs/`)と逆の答えを固定している。関数とテストを消すのはテストのファイル以外も変えるので、この回はしていない(tech-debt に送る項目)
- T4-7(verify V4-1 の 2 つめの入力): 古い ralph が workspace を閉じずに解散した org を、その workspace の中から新しい ralph が disband し、close が失敗した場合。overlay だけの probe で確かめた(commit していない)。古い ralph の解散を「leader の `stop` と、`org_workspace_closed` を書かない raw の `disbanded`」で作り、`HERDR_PANE_ID=pane-9`(座席のない pane)、`HERDR_WORKSPACE_ID=ws-1` から disband した。新しい disband は ws-1 を後回しにし、close が失敗すると `[org_workspace_created, scope_reserved]` が書かれ、古い ralph の `disbanded` が解いた `internal/auth/` が戻って、org-a は動いている座席なしで ws-1 と予約を持って走っている扱いになった。verify の読みのとおりだった。どちらの挙動にするかは決まっておらず、新旧の ralph が同じ台帳を使ったときにしか起きないので、テストとしては足していない
- T4-8: pane の経路で、`disbanded` から補償までの間にほかの org が同じ範囲か最後の枠を取った場合(plan のリスクの窓)のテストはない。workspace の経路は `ReservationRestored_RiskWindowClearedByRetry` が固定している(verify の Coverage gaps と同じ)
- T4-9: ロックの競合は同じプロセスの goroutine だけで見ている。別プロセス、本物の herdr、ロックの外の書き込みが読み直しと append の間に入る窓(cycle 3 の T3-4)は、これまでの回と同じく見ていない(tech-debt の行 176 の (c)(e)、行 179 の (a))。この回の mutation は `internal/cli` では流していない

## Verdict

- Pass: pass。`run-test.sh`(full)は rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。テストを足したあとの `go test ./... -count=1`、`-race`、`^TestOrgCloseDeferredSelf` の `-count=10`(単独、`-race`、負荷の下)も 0 failed。mutation は HEAD のテストで 27 件のうち 17 件が red で、lead が挙げた 7 件は K1(等価)と K2 を除いてすべて red だった。生き残った 10 件のうち 7 件は足したテストで red になり、残る 3 件(K1、K12、K14)は等価である。ユーザーが残すと決めた C4-1 の挙動は足したテストが固定し、「戻さない」案の Q4 で落ちることを確かめた
- Fail: なし
- Blocked: なし。T4-1〜T4-9、付録 C の T3-1〜T3-7、付録 B の T2-1〜T2-4、付録 A の Test gaps 2〜8 は merge を止めない

## 付録 C: cycle 3 の test レポート(104be58d 時点の原文)

cycle 3 の本文を、見出しを 1 段下げただけでそのまま残す(`## Verdict` が 2 つにならないようにするため)。tech-debt の行 176 と 179 が、ここの G、S、L、E、R、H、O、Q の番号と T3 番号を指している。`file:line` と coverage の値は 8b83abeb と 066bf282 の時点のもので、`internal/org/verbs.go` は 83aec44e で行がずれた(関数名で探す)。G4 と G5 は 83aec44e の前の式についての結果で、HEAD では G5 の式はなく、G4 と同じ式を外す K1 は等価で生き残る(この報告の Mutation)。`RulesDiffer` のテストは `..._ReservationUnlessStartedAgain` に改名され、固定する挙動も変わった。本文の中の「付録 B」「付録 A」は、この後ろの同じ名前の付録を指す。

- Date: 2026-10-09(JST。実行の記録は UTC の 2026-10-08 23:36〜23:57。ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、パイプライン 3 回目(cycle 3)。ユーザーが上限を 3 に上げた回で、`cycle-count.json` は /cross-review の決まりで 2 のままなので、この報告と insight event に cycle 3 と書く
- Scope: HEAD 8b83abeb(テストを足した commit は 066bf282)。cycle 2 の test(6ee9836b)からのコードの差分は a94c914f だけで、`internal/org/verbs.go` の補償(`compensateUnderLock` を足し、`reactivateSeat`・`reopenWorkspace`・`reserveAgain` が close の前の写しとロックの下で読み直した台帳の両方を見る)と、`verbs_test.go` の 8 ケース。ほかは文書の commit で、27fefc47 の `AGENTS.md` と `templates/base/ralph.toml` のコメントを含む。`spawn.go`、`envelope.go`、`reserve.go`、`verbs_all.go`、`manifest.go`、`lockfile.go`、`internal/cli/`、`internal/config/`、`scripts/`、`tests/` は 6ee9836b から変わっていない(`git diff --stat` が空)。behavioral test だけを実行した(静的解析は /verify の担当)。重点は、新しいテストの安定性(`-race`、`-count=10`、負荷の下)、self-review の削除の表の 7 行の mutation、cycle 1 と 2 で生き残った X18 の後継、verify V3-1 の挙動
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log` の末尾の「cycle 3」の節(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-233606.log`
- 番号の付け方: cycle 1 の Test gaps 1〜8 と mutation の番号(M1〜M14、X1〜X24、P1)は付録 A に、cycle 2 の T2-1〜T2-4 と N1〜N14 は付録 B に、原文のまま残す。tech-debt の行 172 と 176 がその番号を指しているため。この回の mutation は、self-review の表の 7 行を G1〜G7 とし、ほかは S(写しで判断する)、L(ロック)、E(エラー文と黙った skip)、R、H、O、Q(仮の修正)で振る。Test gaps は T3-1 から振る

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(8b83abeb、テストを足す前) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | shell 1(環境による。下の注) | 434 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(テストを足したあと) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 120 s(`internal/cli` 117 s)、rc 0 |
| `go test -race ./internal/org/... -count=1`(足したあと) | 3 パッケージ | 3 | 0 | 0 | 20 s、rc 0 |
| `^TestOrgCloseDeferredSelf` を `-count=10`(足したあと、単独) | top-level 180(18 本 × 10)、subtest 450 | 630 | 0 | 0 | 5 s、rc 0 |
| 同じものを `-race -count=10` | top-level 180、subtest 450 | 630 | 0 | 0 | 7 s、rc 0 |
| 同じものを、裏で `internal/cli` と `internal/org` の全体を流しながら 3 回(1 回は `-race`) | top-level 540、subtest 1,350 | 1,890 | 0 | 0 | 1 回 4〜7 s、rc 0(裏の実行も rc 0。`internal/cli` 70 s、`internal/org` 13 s) |
| mutation(`go test -overlay`。HEAD のテストのまま) | 23 件 | red 20 件 | 生き残り 3 件(R1、H1、O1) | - | 1 件 13〜21 s |
| mutation(足したテストを入れて流し直し) | 25 件(上の 23 件と仮の修正 Q1、Q2) | red 23 件 | 生き残り 2 件(H1、O1) | - | 1 件 13〜16 s |
| ロックの mutation を `-count=10`(`a spawn holds the manifest lock` の subtest だけ) | 3 件 × 10 回 | G1、L2、L3 とも 10 回中 10 回 red(同じ subtest は HEAD のコードで 10 回とも pass) | - | - | - |
| 確かめの probe(overlay で足しただけで commit しない) | 1 本 | 結果をログに出すだけ | - | - | 1 s 未満 |

- 足したテストは 2 本(066bf282。`verbs_test.go` に +114 行で、テストのファイルだけ)。`TestOrgCloseDeferredSelfWorkspace_NewerRecordsWhileClosing_RulesDiffer`(subtest 2 つ)と `TestOrgCloseDeferredSelfWorkspace_ReservationAppendFails_NamesIt`。理由は Test gaps の T3-1〜T3-3
- shell の件数は suite ごとに数えた。cycle 2 の `run-test.sh`(`docs/evidence/verify-2026-10-08-154409.log`)と、39 本すべての pass・fail・skip の数が一致した(`diff` が空)。a94c914f は shell のテストに触れていない
- Skipped の 1 件は、cycle 2 と同じ `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」(手元の git は 2.49.0)。`tests/test-pre-bash-guard.sh` の `SKIP: 0` は集計行なので数えていない
- `run-test.sh` の中の `go test ./...` は `internal/org` だけを実行し(19.0 s)、ほかはキャッシュの結果だった。そのため 2 行目で全パッケージを `-count=1` で流し直した
- `-count=10` の正規表現 `^TestOrgCloseDeferredSelf` は、`internal/org` の 18 本(a94c914f の 2 本と足した 2 本を含む)に当たる。`internal/cli` に同じ接頭辞のテストはない(CLI の後回しの close は `TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput` で、2 行目の全体の実行に入っている)
- 依頼にあった `tests/test-secret-scan.sh` の flake は出なかった。失敗がないので、単独の再実行はしていない
- 実行のあと(テストの commit の前)、worktree の `git status --porcelain` は足したテストの 1 ファイルだけで、main のチェックアウトは空、main の `.harness/state/org` もできていない

### Coverage

- Statement: `internal/org` 93.3%。HEAD のテストでは 93.2%、cycle 2 は 93.1%。ほかの 7 パッケージは cycle 2 と同じ(`internal/cli` 85.3%、`internal/config` 92.8%、`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数。「HEAD のテスト」は `verbs_test.go` を HEAD の版に戻した overlay で測った値):

| 関数 | 位置 | HEAD のテスト | 足したあと | 通らない文 |
| --- | --- | --- | --- | --- |
| `CloseDeferredSelfPane` | `verbs.go:1017` | 100.0% | 100.0% | なし |
| `CloseDeferredSelfWorkspace` | `verbs.go:1098` | 100.0% | 100.0% | なし |
| `compensateUnderLock` | `verbs.go:1208` | 100.0% | 100.0% | なし(ロックが取れない枝と、2 回目の read の失敗の枝も通る) |
| `reactivateSeat` | `verbs.go:1237` | 100.0% | 100.0% | なし |
| `reopenWorkspace` | `verbs.go:1268` | 100.0% | 100.0% | なし |
| `reserveAgain` | `verbs.go:1300` | 88.9% | 100.0% | HEAD のテストでは追記の失敗の戻り(`:1309-1311`) |
| `selfCompensation.add` / `errorFor` | `verbs.go:1163` / `:1182` | 100.0% / 100.0% | 100.0% / 100.0% | なし |
| (参考)`lastHerdrAgentName` | `verbs.go:1330` | 80.0% | 80.0% | `return ""`(`:1337`)。tech-debt の行 171 の (a) と同じ |

- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: a94c914f の新しい枝(ロックが取れない、2 回目の read の失敗、3 つの skip)の文はすべて HEAD のテストで通る。`reserveAgain` の追記の失敗(`verbs.go:1309-1311`)だけが通らず、足した `ReservationAppendFails_NamesIt` が通す。行の coverage では、各条件の片側を外せるかは分からないので、下の mutation で確かめた。tech-debt の行 176 の (b) が書く `reserveAgain` の 75.0% は a94c914f の前の関数の値で、今は 88.9%(HEAD のテスト)、足したあとは 100.0%

### Mutation

各 mutation は、`internal/org/verbs.go` の写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off ./internal/org/` でパッケージ全体を流した(`-run` で絞っていない)。worktree のファイルは書き換えていない。最初の 23 件は HEAD のテストのまま流し、テストを足したあとで 25 件を流し直した。どの mutation も build は通った。a94c914f は `internal/org` の外を変えていないので、`internal/cli` では流していない(補償の呼び出し元は `internal/cli` の `closeDeferredSelf` だけ)。

self-review の削除の表の 7 行(依頼):

| # | Mutation | HEAD のテスト | Red になった subtest | self-review の読みとの違い |
| --- | --- | --- | --- | --- |
| G1 | ロックを外す(`withManifestLock` の代わりに `fn` を直接呼ぶ。読み直しは残す) | red | `NewerRecordsWhileClosing_Stay` の「a spawn holds the manifest lock」、`NothingRestored_NamesTheManualClose` の「pane / workspace, manifest lock unavailable」 | 同じ |
| G2 | `reactivateSeat` の `current != seat` を外す | red | 「pane: seat spawned again in another pane」「workspace: org started again with another reservation」「workspace: org started again and disbanded again」 | 同じ |
| G3 | `reopenWorkspace` の `current != last` を外す | red | 「workspace: herdr gave the id to another org」 | 同じ |
| G4 | `reserveAgain` の `ActiveReservation(now) != nil` を外す(X18 の後継) | red | 「workspace: org started again with another reservation」「workspace: a spawn holds the manifest lock」 | 同じ |
| G5 | `reserveAgain` の `!slices.Equal(...)` を外す | red | 「workspace: org started again and disbanded again」 | 同じ |
| G6 | `errorFor` が skip のときに byHand を足さない | red | 「pane: seat spawned again in another pane」「workspace: herdr gave the id to another org」「workspace: org started again with another reservation」 | self-review は「skip の 4 ケース」と書くが、`HasSuffix` で byHand を見る skip のケースは 3 つ。「org started again and disbanded again」と「a spawn holds the manifest lock」は skip の文だけを見る |
| G7 | `compensateUnderLock` が 2 回目の read の失敗を無視して、空の台帳で補償する | red | 「workspace: manifest unreadable at the second read」 | 同じ |
| G7b | 2 回目の read の失敗で nil を返す(失敗を記録しない) | red | 同じ subtest | - |

写しで判断する形に戻す、ロック、エラー文:

| # | Mutation | HEAD のテスト | Red になった subtest |
| --- | --- | --- | --- |
| S1 | pane の経路の `reactivateSeat` に、`now` の代わりに close の前の写しを渡す | red | 「pane: seat spawned again in another pane」 |
| S2 | workspace の経路の `reopenWorkspace` に写しを渡す | red | 「herdr gave the id to another org」 |
| S3 | workspace の経路の `reactivateSeat` に写しを渡す | red | 「org started again with another reservation」「org started again and disbanded again」 |
| S4 | `reserveAgain` に写しを渡す(cross-review cycle 2 の指摘の形) | red | 「a spawn holds the manifest lock」「org started again with another reservation」「org started again and disbanded again」 |
| S5 | workspace の経路の 3 つすべてに写しを渡す(ロックは残す。a94c914f の前の判断) | red | `NewerRecordsWhileClosing_Stay` の workspace の 4 ケース |
| L2 | 台帳をロックを取る前に読む | red | 「a spawn holds the manifest lock」 |
| L3 | 別のディレクトリの lock file をロックする | red | 「a spawn holds the manifest lock」と、lock unavailable の 2 ケース |
| E1 | `errorFor` が「did not record ... again」の文を出さない | red | skip の 5 ケース |
| E2 | `reactivateSeat` が skip を黙って返す(skipped を `""` にする) | red | G2 と同じ 3 ケース |
| E3 | `reopenWorkspace` が skip を黙って返す | red | 「herdr gave the id to another org」 |
| E4 | `reserveAgain` が skip を黙って返す | red | 「a spawn holds the manifest lock」「org started again with another reservation」「org started again and disbanded again」 |
| E5 | `compensateUnderLock` がロックか read の失敗を `c` に積まない | red | 「manifest unreadable at the second read」と lock unavailable の 2 ケース |
| R1 | `reserveAgain` が追記の失敗を握りつぶす | **生き残り**(HEAD のテスト)。足したテストで red | 足した `ReservationAppendFails_NamesIt` |
| H1 | `reactivateSeat` の herdr agent name を `now` でなく写しから取る(a94c914f の前の形) | 生き残り(等価) | なし。下の T3-6 |
| O1 | workspace の経路で、自分の pane の座席を写しでなく `now` から探す | 生き残り | なし。下の T3-5 |

仮の修正(足した記録のテストが、規則をそろえる修正で落ちることの確かめ):

| # | Mutation | 結果 | Red になった subtest |
| --- | --- | --- | --- |
| Q1 | `reserveAgain` が、その org が `now` で走っていれば戻さない(`--reserve` なしの立て直しで古い予約を戻さない形) | red | 足した `RulesDiffer` の「started again without --reserve」だけ |
| Q2 | workspace の経路が、`now` に新しい `disbanded` があれば 3 つとも戻さない | red | 足した `RulesDiffer` の「only disband run again」と、既存の「org started again and disbanded again」(skip の文が変わる) |

- 依頼の 7 行はどれも HEAD のテストで red になった。self-review の表の読みは G6 の数え方を除いて実行と合った
- X18: cycle 1 と 2 で生き残った「すでに予約を持つなら書かない」の早期 return(写しの `ActiveReservation(events) != nil`)は a94c914f で消え、同じ判定は今の台帳の `ActiveReservation(now, orgID) != nil` だけになった。それを外す G4 は HEAD のテストで red になる(2 つの subtest)。tech-debt の行 176 の (a) は、/sync-docs で取り消し線にできる
- ロックを外す 3 件(G1、L2、L3)を、時間に頼る subtest(「a spawn holds the manifest lock」。別の goroutine がロックを持って 200 ms 待つ)だけで `-count=10` 流した: G1、L2、L3 とも 10 回中 10 回 red で、同じ subtest は HEAD のコードで 10 回とも pass した。補償は、ロックを持つ goroutine が 200 ms 待つ間に読むので、cycle 1 の M14(ロックを外して 30 回中 29 回 red)のような取りこぼしはなかった。G1 と L3 は lock unavailable の 2 ケースでも決定的に落ちる
- 足した `RulesDiffer` は、HEAD の挙動を変える mutation のうち G2、G5、G6、S3、S4、S5、E1、E2、E4 でも red になる。記録のテストなので、補償を変えるときには一緒に直す

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review cycle 2 の指摘(F-5): 補償が close の前の写しで予約を書き戻し、立て直した org の新しい予約を古いものに替える | 直っている | 「org started again with another reservation」「a spawn holds the manifest lock」が通る。写しで判断する S4 と S5 が red |
| AC15 のふつうの失敗(close を待つ間に台帳に新しい記録がない): workspace、座席、予約の順に書き戻し、ほかの org は同じ範囲を予約できない | 保たれている | 変わっていない `CloseFails_ReservationRestored` が `-count=10` と `-race` で通る |
| plan のリスクの項目(ほかの org が枠か範囲を取った窓は、打ち直しの disband で解ける) | 保たれている | 変わっていない `ReservationRestored_RiskWindowClearedByRetry` が通る |
| `--force` は何も書かない | 保たれている | `TestOrgCloseDeferredSelf_Force_NoCompensation` が通る(`--force` はロックを取る前に戻る) |
| pane の経路の補償(S8)、確かめのあとの close、打ち直しの close | 保たれている | `TestOrgCloseDeferredSelfPane_*` と `TestOrgCloseDeferredSelf_RecheckedBeforeTheClose` が `-count=10` で通る |
| cycle 1 と 2 の上限と予約の判定 | 変わっていない | 判定のファイルは 6ee9836b から変わっていない(上の Scope)。cycle 2 の mutation(付録 B)は流し直していない |

### Test gaps

この回に見つけたもの。cycle 2 の T2-1〜T2-4 は付録 B に、cycle 1 の Test gaps 1〜8 は付録 A にある。どちらも a94c914f の差分に関係しないので変わらない。ただし、cycle 1 の Test gaps 3 と tech-debt の行 176 の (a)(どちらも X18)は、上の G4 で埋まった。

- T3-1(verify V3-1): close を待つ間に、同じ org の disband だけが打ち直された場合の挙動を、足したテスト `RulesDiffer` の 1 つめの subtest で記録した。verify の読みのとおりだった。座席は最新の状態イベントが変わらない(org の `disbanded` は seat_id を持たないので座席の最新の状態イベントにならず、`stopped` の座席は前後どちらでも Active が false)ので戻り、ws-1 も開き直す。予約は、新しい `disbanded` の前に予約がないので戻らない。org-a は走っている扱い(`RunningOrgs`)に戻り、ほかの org が同じ範囲を予約できる。エラーは workspace と座席を戻したことと、予約を戻さなかったことを書き、手で閉じる herdr のコマンドで終わる。打ち直しの disband で解ける。挙動の記録で、要件ではない。3 つの規則をそろえるか(Q2 の形)は判断が要るので、self-review が案を書いた C3 のまとめの行(tech-debt、/sync-docs の担当)に入れるのがよい
- T3-2(self-review C3-4 の (a)): `--reserve` なしで org を立て直した場合を、同じテストの 2 つめの subtest で記録した。座席は戻らず、古い予約は立て直した run に書き戻され、ほかの org の重なる予約は拒否される。Q1 で落ちることを確かめた
- T3-3: `reserveAgain` の追記の失敗の枝にテストがなく、追記の失敗を握りつぶす R1 が生き残った。`ReservationAppendFails_NamesIt` を足して埋めた(manifest を読み取り専用にする既存の `readOnlyManifest` を使い、エラーに予約の追記の失敗が出ることを見る)。これで tech-debt の行 176 の (b) から `reserveAgain` を外せる
- T3-4(self-review C3-1 の (b)、verify V3-3): ロックを取らない書き込みが、読み直しと append の間に入る窓。overlay だけの probe で確かめた(commit していない)。最初の補償の append の `TS` が `o.Now` を呼ぶ時に disband を打ち直すと、台帳は `disbanded`、`org_workspace_created`、`spawned`、`scope_reserved` の順になり、新しい `disbanded` のあとで 3 つとも戻る。org-a は予約つきで走っている扱いに戻り、エラーは「recorded ... again, so running the command again retries the close」で、手で閉じるコマンドは付かない。同じ disband が読み直しの前に入ると T3-1 の結果(予約だけ戻らない)になるので、結果はどちらの側に入るかで変わる。窓は読み直しから最後の append までで、herdr を待つ時間は含まない。probe は `reopenWorkspace` の中の呼び出しの順に頼るので、テストとしては足していない
- T3-5(O1): workspace の経路は自分の pane の座席を写し(`rr.Events`)の `lastSeatOnPane` で探す。`now` で探す O1 は生き残った。違いが出るのは、close を待つ間に同じ pane id で別の座席が記録された場合だけで、tech-debt の行 171 の (b) の C2-2 の側(同じ pane id を持つ古い `stopped` の座席)と同じ欠けである。a94c914f の前からある
- T3-6(H1): `reactivateSeat` の herdr agent name を写しから取っても結果は変わらない。等価な mutation である。`HerdrAgentName` を書くのは `spawned` だけ(`spawn.go:979` と補償の `reactivateSeat`)で、`spawned` は座席の状態イベントなので、新しい名前が台帳にあれば `current != seat` で先に skip する。self-review の「状態イベントが変わらない座席では結果が同じ」の読みは実行でも合った
- T3-7: ロックの競合は同じプロセスの goroutine だけで見ている。別プロセス、本物の herdr、古い ralph が書いた台帳は、cycle 1 と 2 と同じく見ていない(tech-debt の行 176 の (c)(e))。この回の mutation は `internal/cli` では流していない

### Verdict

- Pass: pass。`run-test.sh`(full)は rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。テストを足したあとの `go test ./... -count=1`、`-race`、`^TestOrgCloseDeferredSelf` の `-count=10`(単独、`-race`、負荷の下)も 0 failed。mutation は HEAD のテストで 23 件のうち 20 件が red で、依頼の 7 行(G1〜G7)はすべて red。X18 の後継の G4 も red になった。生き残りの R1 は足したテストで red になり、H1 は等価、O1 は a94c914f の前からある欠け(T3-5)。V3-1 の挙動は、テストの実行で verify の読みのとおりと確かめ、記録のテストにした
- Fail: なし
- Blocked: なし。T3-1〜T3-7、付録 B の T2-1〜T2-4、付録 A の Test gaps 2〜8 は merge を止めない

## 付録 B: cycle 2 の test レポート(6ee9836b 時点の原文)

cycle 2 の本文を、見出しを 1 段下げただけでそのまま残す(`## Verdict` が 2 つにならないようにするため)。この報告の本文が、ここの T2 番号と N 番号を指している。`file:line` と coverage の値は f144021d 時点のもので、`internal/org/verbs.go` は a94c914f で行がずれた(関数名で探す)。`reserveAgain` の 75.0% は a94c914f の前の関数の値で、この回は 88.9%(HEAD のテスト)と 100.0%(テストを足したあと)。本文の中の「付録 A」は、この後ろの付録 A(cycle 1)を指す。

- Date: 2026-10-09(JST。実行の記録は UTC の 2026-10-08 15:44〜16:01。ファイル名は計画の日付)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 2(`cycle-count.json` は 2。既定の上限の最後の回)
- Scope: HEAD f144021d。cycle 1 の test(fff1ec23)からのコードの差分は 3 つの commit にある。76d1cf1c は `internal/cli/org.go` の help の文言(`orgWideLimitsHelp`、`--config` の説明、`status` の `Long`)。975df92b と afcbc6c2 は `internal/org` の修正で、動いていない leader の予約の前に max_orgs を判定し、その判定を `validateMaxOrgs` に切り出して `ValidateOrgWideCapacity` と共有する(`envelope.go`、`spawn.go`、テストは `spawn_test.go` の 1 本)。ほかは文書の commit。behavioral test だけを実行した(静的解析は /verify の担当)。重点は、AC1 と AC14 にかかる新しい枝、self-review C2-3 の `if !seat.Active`、cycle 1 の安全側の mutation の再確認
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log` の末尾の「cycle 2」の節(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-154409.log`
- 番号の付け方: cycle 1 の Test gaps 1〜8 と mutation の番号(M1〜M14、X1〜X24、P1)は、付録 A に原文のまま残す。tech-debt の行 172 と 176 がその番号を指しているため。この回の新しい mutation は N1 から、新しい Test gaps は T2-1 から振る。cycle 1 の mutation を流し直したものは cycle 1 の番号で書く

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(f144021d) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | shell 1(環境による。下の注) | 319 s、rc 0 |
| `go test ./... -count=1 -coverprofile` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 79 s(`internal/cli` 77 s)、rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1` | 4 パッケージ | 4 | 0 | 0 | 86 s(`internal/cli` 82 s)、rc 0 |
| 変わったテストを `-count=3`(`internal/org`) | top-level 12(4 本 × 3)、subtest 60 | 72 | 0 | 0 | rc 0 |
| 変わったテストを `-count=3`(`internal/cli`) | top-level 24(8 本 × 3)、subtest 24 | 48 | 0 | 0 | rc 0 |
| mutation(`go test -overlay`。worktree には触れていない) | 26 件 | red 23 件 | 生き残り 3 件(N3、N8、X18) | - | 1 件 12〜82 s |
| 確かめのテスト(overlay で足しただけで commit しない。HEAD、N8、N3 で 1 回ずつ) | 1 本 × 3 | HEAD と N3 で pass | N8 で red | - | 各 1 s 未満 |
| 実バイナリの help(`ralph org`、`org spawn`、`org start`、`org status` の `--help`) | 4 | 4(rc 0) | 0 | 0 | - |

- `-count=3` の正規表現は、`internal/org` が `^(TestOrgSpawn_Reserve_InactiveLeaderChecksMaxOrgs|TestValidateOrgWideCapacity|TestOrgSpawn_Reserve_ExistingLeader|TestOrgSpawn_Reserve_ExistingLeader_Phase2)$`、`internal/cli` が `^(TestOrgSpawn_DeprecatedDriverFlagAlias_HiddenFromHelp|TestOrgStatus_.*|TestOrgStart_Reserve_SameListPassesDifferentListRejected|TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml)$`。help の文言を固定するテストはない(`spawn --help` を打つのは `HiddenFromHelp` だけで、`--leader-driver` の有無しか見ない)。そのため、76d1cf1c の 3 か所は実バイナリの出力で確かめた(Regression checks の 4 行目)
- shell の件数は suite ごとに数えた。cycle 1 の 2 回目の `run-test.sh`(`docs/evidence/verify-2026-10-08-124051.log`)と、39 本すべての pass・fail・skip の数が一致した(`diff` が空)。この差分は shell のテストを変えていない
- Skipped の 1 件は `tests/test-secret-scan-branch.sh` の「the real git older than 2.41 case」で、手元の git が 2.49.0 なので毎回 SKIP になる。suite ごとの数が一致するので、cycle 1 の 2 回の実行にも同じ SKIP があった。付録 A の表の Skipped の 0 は、この 1 件を数えていない。`tests/test-pre-bash-guard.sh` の `SKIP: 0` は集計行なので数えていない
- `run-test.sh` の中の `go test ./...` は、`internal/org` だけを実行し、ほかはキャッシュの結果だった(cycle 1 と同じ)。そのため 2 行目で全パッケージを `-count=1` で流し直した
- 依頼にあった `tests/test-secret-scan.sh` の flake は出なかった。失敗がないので、単独の再実行はしていない
- 実行のあと、worktree と main のチェックアウトの `git status --porcelain` はどちらも空で、main の `.harness/state/org` もできていない

### Coverage

- Statement: `internal/org` 93.1%(cycle 1 は 93.0%)。ほかの 7 パッケージは cycle 1 と同じ(`internal/cli` 85.3%、`internal/config` 92.8%、`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数と、cycle 1 の一覧):

| 関数 | 位置 | Coverage | cycle 1 との違い | 通らない文 |
| --- | --- | --- | --- | --- |
| `validateMaxOrgs` | `envelope.go:107` | 100.0% | 新しい関数 | なし |
| `ValidateOrgWideCapacity` | `envelope.go:92` | 100.0% | 同じ | なし |
| `idempotentRespawn` | `spawn.go:1098` | 90.9% | 87.5% から上がった(足した文はすべて通る) | `scope_reserved` の追記の失敗(`:1110-1112`)。cycle 1 と同じ文 |
| `NormalizeReservePaths` / `normalizeReservePath` | `reserve.go:45` / `:61` | 100.0% / 100.0% | 同じ | なし |
| `reservePathsOverlap` | `reserve.go:90` | 100.0% | 同じ | なし |
| `RunningOrgs` | `reserve.go:233` | 100.0% | 同じ | なし |
| `ActiveReservation` | `reserve.go:256` | 100.0% | 同じ | なし |
| `TotalActiveSeats` | `reserve.go:265` | 100.0% | 同じ | なし |
| `spawnCapacityErr` | `spawn.go:1065` | 100.0% | 同じ | なし |
| `reservationDecision` | `reserve.go:283` | 100.0% | 同じ | なし |
| `reserveAgain` | `verbs.go:1225` | 75.0% | 同じ | すでに予約を持つときの戻りと、追記の失敗 |
| `orgsToDisband` | `verbs_all.go:309` | 100.0% | 同じ | なし |
| `withMainWorktreeOrgLimits` | `cli/org.go:148` | 100.0% | 同じ | なし |
| `MainWorktreeRoot` | `statedir.go:81` | 100.0% | 同じ | なし |

- そのほかの関数: `checkCapacityAndStart` 69.2%、`dryRunSpawn` 93.7%、`autonomousScopeGateErr` 100.0%、`CloseDeferredSelfWorkspace` 100.0%、`newOrgSpawnRuntime` 90.0%、`orgReservation` 75.0%、`printStatusTable` / `printStatusJSON` 100.0%、`config.Load` 92.9%、`config.Default` 100.0% で、どれも cycle 1 と同じ。help を変えた関数は `newOrgCmd` 100.0%、`newOrgSpawnCmd` 95.0%、`newOrgStartCmd` 100.0%、`newOrgStatusCmd` 90.0%(通らないのは `Status` と `orgReservation` のエラーの戻り)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 新しい枝(`spawn.go:1100-1103`)の文はすべて通る。行の coverage からは `if !seat.Active` を外せるかどうかは分からないので、下の N3 で確かめた。tech-debt の行 176 の (b) は `idempotentRespawn` を 87.5% と書くが、今は 90.9% である。通らない文は変わっていない(self-review C2-5 の指摘と同じ)

### Mutation

各 mutation は、対象のファイルの写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off` でパッケージ全体を流した(`-run` で絞っていない)。まず 26 件すべてを `internal/org` で流した。生き残った N3 と N8、それに N13 は、`internal/org` と `internal/cli` の両方で流し直した。worktree のファイルは書き換えていない。

この回のコード(975df92b、afcbc6c2)の mutation:

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| N1 | `idempotentRespawn` が `validateMaxOrgs` を呼ばない(依頼) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1, max_orgs 1」と「phase 2, max_orgs 1」だけ |
| N2 | `validateMaxOrgs` の `!slices.Contains(runningOrgs, orgID)` を外す(依頼。新しい spawn と idempotent の両方に効く) | red | `TestValidateOrgWideCapacity`(「running org at / over max_orgs」)、`MaxOrgs_NewOrgRejectedAtLimit`、`OrgWideLimits_DryRunPredictsTheSameRejection`、`Reserve_InactiveLeaderChecksMaxOrgs`(「org running through another seat」) |
| N2b | idempotent の呼び出しだけ、org_id が `runningOrgs` に合わないようにする(`p.OrgID+"#"` を渡す) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の 3 つの subtest |
| N3 | `if !seat.Active` を外し、いつも max_orgs を判定する(依頼。self-review C2-3) | **生き残り**(等価) | なし。下の T2-2 |
| N3b | 条件を逆にする(`if seat.Active`) | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1 / phase 2, max_orgs 1」 |
| N4 | `ValidateOrgWideCapacity` で max_total_seats を max_orgs より先に判定する(依頼) | red | `TestValidateOrgWideCapacity` の「both reached reports max_orgs first」 |
| N5 | `validateMaxOrgs` の `>=` を `>` にする | red | 8 本(`TestValidateOrgWideCapacity`、`ZeroOrgWideLimits_Reject`、`Reserve_InactiveLeaderChecksMaxOrgs` ほか) |
| N6 | 新しい枝の拒否を `SpawnOutcomeFailed` で返す | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「max_orgs 1」の 2 つ |
| N7 | 新しい枝の拒否で `rejected` と receipt を書く(`o.reject`) | red | 同じ 2 つ(最後の記録が `disbanded` のまま、の検査) |
| N8 | `idempotentRespawn` で予約の判定を max_orgs より先にする | **生き残り** | なし。下の T2-1 |
| N9 | 走っている org がないときの `(none)` を空にする | red | `TestValidateOrgWideCapacity`(「zero max_orgs」)、`ZeroOrgWideLimits_Reject` |
| N10 | `ValidateOrgWideCapacity` が org_id の代わりに seat id を渡す | red | `TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`OrgWideLimits_DryRunPredictsTheSameRejection` |
| N11 | `ValidateOrgWideCapacity` だけが `validateMaxOrgs` を呼ばない(新しい spawn の側) | red | 8 本(`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、`ReservationRestored_RiskWindowClearedByRetry` ほか) |
| N12a | Phase 1 の呼び出し(`spawn.go:606`)だけ、座席を Active として渡す | red | `Reserve_InactiveLeaderChecksMaxOrgs` の「phase 1, max_orgs 1」 |
| N12b | Phase 2 の呼び出し(`spawn.go:762`)だけ、座席を Active として渡す | red | 同じテストの「phase 2, max_orgs 1」 |
| N13 | (比べるため)新しい spawn の `spawnCapacityErr` で、予約を全 org の上限より先に判定する | red | `TestOrgDisbandAll_ReservationOnlyOrgDisbanded`(max_orgs と重なりの両方に当たる spawn に max_orgs の拒否を期待する、`verbs_all_test.go:588`) |
| N14 | `idempotentRespawn` が `reservationDecision` だけを飛ばす(max_orgs は残す) | red | `Reserve_ExistingLeader` の 3 つの subtest、`Reserve_ExistingLeader_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` の「max_orgs 2」と「through another seat」 |

cycle 1 の mutation の流し直し(依頼の安全側のもの。番号は付録 A と同じ):

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| M1 | max_orgs の判定を外す。判定が `validateMaxOrgs` に移ったので、新しい spawn と idempotent の両方に効く | red | 9 本(`TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、`Reserve_InactiveLeaderChecksMaxOrgs` ほか) |
| M2 | max_total_seats の判定を外す | red | 6 本(`MaxTotalSeats_Boundary`、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded`、`..._ReactivationCanExceedMaxTotalSeats` ほか) |
| M3 | `RunningOrgs` が `rejected` だけの座席の org も数える | red | 6 本(`TestRunningOrgs`、`MaxOrgs_NewOrgRejectedAtLimit` ほか) |
| M5 | ディレクトリどうしの重なりで末尾の `/` を外して比べる | red | `TestReservePathsOverlap`、`TestReservationDecision`、`Reserve_OverlapWithRunningOrgRejected` |
| X23 | `reservationDecision` がほかの org の予約を見ない(重なりの判定がない) | red | 7 本(`ConcurrentReservations_OnlyOneOfOverlappingWins`、`Reserve_ExistingLeader` ほか) |
| M8 | `idempotentRespawn` が `Reserve` の扱いを丸ごと飛ばす | red | `Reserve_ExistingLeader`、`_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` |
| X24 | Phase 2 の idempotent の戻りが `idempotentRespawn` を通らない | red | `Reserve_ExistingLeader_Phase2`、`Reserve_InactiveLeaderChecksMaxOrgs` の「phase 2」の 2 つ |
| X11 | 立っている leader への予約の拒否で `rejected` を書く | red | `Reserve_ExistingLeader` |
| X18 | `reserveAgain` の「すでに予約を持つなら書かない」を外す | 生き残り(cycle 1 と同じ) | なし。`verbs.go` はこの回変わっていない |

- 依頼の 4 件(N1、N2、N3、N4)のうち、N3 だけが生き残った。cycle 1 の安全側の mutation は、X18 を除いてすべて red のまま。
- N1 を落とすのは新しいテストだけである。新しいテストを戻すと、この回の修正は守られなくなる。

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 の安全側の判定(max_orgs、重なり、`rejected` だけの org、立っている leader への予約) | 保たれている | M1、M3、M5、X23、M8、X24、X11 が HEAD で red |
| `ValidateOrgWideCapacity` の条件、判定の順、文言(`validateMaxOrgs` への切り出しの前後) | 保たれている | `TestValidateOrgWideCapacity` の 10 行が `-count=3` で通る。判定の順は「both reached reports max_orgs first」が、`(none)` の文言は「zero max_orgs」が固定していて、N4、N9、N10、N11 が red |
| 予約のない再試行は既存の座席を返す(AC14 の後半。新しい枝で拒否したあと) | 保たれている | 新しいテストが、拒否のあとも leader が `spawned` のままで、最後の記録が古い `disbanded` のままであることを見る。`rejected` を書く N7 が red。`rejected` が書かれると座席の最新の状態が変わり、次の再試行が idempotent の戻りに入らなくなる(verify の C2-2 の判断) |
| help の 3 か所(76d1cf1c) | 表示を確かめた | 実バイナリで、`org spawn --help` と `org start --help` に `orgWideLimitsHelp` の段落が 1 回ずつ出る。`org status --help` に `Long` が出る。4 つの help の `--config` の説明が新しい文になる。どれも rc 0。`status` の `reserved: <path>, ...` の形は、`printStatusTable` の `strings.Join(reservation, ", ")` と、`TestOrgStatus_ShowsReservation` の期待(`reserved: docs/x.md, internal/auth/`)に合う。文言の内容が正しいかは /verify の担当 |
| 今ある `max_seats`、scope のゲート、spawn の idempotent | 保たれている | `run-test.sh` と `go test ./... -count=1` がすべて通る |

### Test gaps

この回に新しく見つけたもの。cycle 1 の Test gaps 1〜8 は付録 A にある。1 で足した 2 本のテストは今も通り、2〜8 はこの回の差分に関係しないので変わらない(X18 は流し直しても生き残る)。

- T2-1(N8): idempotent の経路では、max_orgs と重なりのどちらを先に判定するかがテストで固定されていない。古い台帳の動いていない leader が max_orgs に当たり、しかも頼んだ範囲がほかの走っている org の予約に重なるとき、今のコードは max_orgs の拒否を返す。これは新しい org が同じ場合に受け取る拒否と同じで、新しい spawn の側の順は `TestOrgDisbandAll_ReservationOnlyOrgDisbanded` が固定している(N13 が red)。`idempotentRespawn` の順を入れ替えても(N8)、どちらの順でも拒否になり何も記録されないので、変わるのは出てくる理由の文だけである。`idempotentRespawn` の doc と新しいテストのコメントは「the same error a new org gets」と書くが、判定の順を決めた AC はない。依頼に従い、AC の穴とはみなさずテストを足さなかった。overlay だけで足した確かめのテストで、subtest 1 つで塞げることを確かめた。org-b の leader が `internal/` を予約し、max_orgs が 1 で、org-a の leader に古い disband があるとき、org-a の leader の `internal/x.go` の予約に `max_orgs 1 reached` を期待する。このテストは HEAD と N3 で通り、N8 で落ちる(`docs/evidence/` の cycle 2 の節に本文と出力がある)
- T2-2(N3、self-review C2-3): `if !seat.Active` を外しても結果は変わらない。これは等価な mutation で、今の呼び出し元のままではどのテストでも落とせない。2 つの呼び出し元(`spawn.go:606` と `:762`)は、座席を `Roster(events, RosterOptions{})` から取り、同じ `events` を `idempotentRespawn` に渡す。`RunningOrgs` は同じ roster の Active な座席の org を「走っている」に数えるので、Active な座席なら `validateMaxOrgs` は MaxOrgs の値にかかわらず nil を返す。条件で省けるのは `RunningOrgs` を 1 回計算することだけである。N12a と N12b(呼び出し元が動いていない座席を Active として渡す)は red になるので、条件が効くのは座席と `events` が食い違うときに限られる。今の呼び出し元では、そうはならない。C2-3 の読みは、実行でも合っていた。外すかどうかはコードの形の判断で、外すなら `internal/org` の変更なのでパイプラインの再実行になる
- T2-3: help の文言(`orgWideLimitsHelp`、`status` の `Long`、`--config` の説明)を固定するテストはない。ほかの help の文も、`HiddenFromHelp` を除いて固定されていない。この回は実バイナリの出力で表示だけを確かめた
- T2-4: 新しいテストの古い台帳は、`spawned` と `disbanded` の 2 行を手で書いて作っている。古い ralph のバイナリが実際に書いた台帳では確かめていない(verify の Coverage gaps と同じ)。CLI から古い台帳に `ralph org start --reserve` を打つテストもなく、CLI の経路は verify の実バイナリの確認だけである

### Verdict

- Pass: pass。`run-test.sh`(full)は rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`go test ./... -count=1`、`-race`、変わったテストの `-count=3` も 0 failed。mutation は 26 件のうち 23 件が red。依頼の 4 件では N1、N2、N4 が red で、N3 は等価なので生き残った(T2-2)。cycle 1 の安全側の mutation は、X18 を除いてすべて red のまま。N8 の生き残り(T2-1)は理由の文の違いだけで、AC の穴ではない。AC1 と AC14 の新しい枝を、Phase 1 と Phase 2 の両方でテストの実行で確かめた
- Fail: なし
- Blocked: なし。T2-1〜T2-4 と、付録 A の Test gaps 2〜8 は merge を止めない

## 付録 A: cycle 1 の test レポート(fff1ec23 時点の原文)

cycle 1 の本文を、見出しを 1 段下げただけでそのまま残す(`## Verdict` が 2 つにならないようにするため)。tech-debt の行 172 と 176 は、ここの Test gaps の番号、Coverage、mutation X18 を指している。`file:line` と coverage の値は fff1ec23 時点のもの(`idempotentRespawn` の 87.5% は、この回は 90.9%)。

- Date: 2026-10-08(JST。実行の記録は UTC の 2026-10-08 11:54〜12:49)
- Plan: docs/plans/active/2026-10-08-org-limits-reserve.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch feat/org-limits-reserve と base 51855166 の差分。コードの commit は 620458d7(設定)、11fc2261 と 8fe95acd(org の層)、78e46f36(CLI と `statedir.go`)、8dd19634(文言)。テスト開始時の HEAD は fbb04f83。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC11、AC13〜AC15 を見た。AC12 は文書の基準なので /verify の担当
- Evidence: `docs/evidence/test-2026-10-08-org-limits-reserve.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-08-115436.log`(テスト追加前)と `docs/evidence/verify-2026-10-08-124051.log`(追加後)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加前、fbb04f83) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 342 s、rc 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加後、c3a95c48) | shell 39 本(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 502 s、rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... ./internal/config/... -count=1`(追加前) | 5 パッケージ | 5 | 0 | 0 | `internal/cli` 94 s、rc 0 |
| `go test -race ./internal/org/ -count=1`(追加後) | 1 パッケージ | 1 | 0 | 0 | 14 s、rc 0 |
| 新しいテストを `-count=3`(追加前。下の正規表現) | top-level 168(56 本 × 3。PR の新しいテスト 41 本をすべて含む) | 168 | 0 | 0 | rc 0 |
| 新しいテストを `-count=3`(追加後。正規表現に `CloseDeferredSelfPane` を足した) | top-level 189(63 本 × 3。新しいテスト 43 本を含む)、subtest 468 | 657 | 0 | 0 | rc 0 |
| `go test ./... -count=1 -coverprofile`(追加後) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 99 s、rc 0 |
| mutation(依頼の 14 種を 16 件に分けたものと、追加の 24 件。`go test -overlay` で差し替え、worktree には触れていない) | 40 件と確認 1 件(P1) | red 39 件 | 生き残り 1 件(X18)。X24 は Phase 2 のテストを足す前は生き残り、足したあと red | - | 1 件 15〜140 s |
| M14(ロックなし)で `-run Concurrent -count=10` | 新しい競合のテスト 3 本 × 10 | - | 29 回 red / 30 回 | - | rc 1 |

- `-count=3` の正規表現は `Reserve|Reservation|OrgWide|RunningOrgs|MainWorktree|ZeroOrgWide|MaxOrgs|MaxTotalSeats|Concurrent|TotalActiveSeats|ReservationOnlyOrg|OrgFleetLimits|ScopeReservedDetails|NormalizeReservePaths|ReservePathsOverlap|ActiveReservation|CloseDeferredSelfWorkspace|ShowsReservation|ReserveFlag|Defaults` で、`internal/org`、`internal/cli`、`internal/config` に掛けた。base からの差分で足された `func Test` 41 本が、1 回目の実行の `--- PASS` にすべて出ることを `comm` で確かめた。既存の `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` なども含む。
- 2 回の `run-test.sh` は、どちらも `Requested scope: full`、`Language scope: full`、`Language packs selected: golang` で走った。2 回とも、中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため、テスト追加後に `-count=1` で全パッケージを流し直した(表の 7 行目)。`internal/cli` は 97 s で、前回の記録(70〜100 s)の範囲に入る。
- shell の件数は、各 suite の集計行(`PASS: 29`、`PASS: 47 / 47`、`22 passed, 0 failed` など)の合計 2,046 件と、集計行のない `tests/test-no-loop-references.sh` の 1 件を足したもの。org-watch-stop-failed の test レポートと同じ 2,047 件で、この差分は shell のテストを変えていない。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回とも出なかった。失敗したテストがないので、単独の再実行はしていない。

### Coverage

- Statement(追加後): `internal/org` 93.0%(org-stop-all cycle 2 の記録は 92.6%)、`internal/cli` 85.3%(同じ)、`internal/config` 92.8%(92.3%)。ほかの 5 パッケージは前回と同じ(`insights` 86.1%、`org/driver` 93.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Function(依頼の関数):

| 関数 | 位置 | Coverage | 通らない文 |
| --- | --- | --- | --- |
| `NormalizeReservePaths` / `normalizeReservePath` | `reserve.go:45` / `:61` | 100.0% / 100.0% | なし |
| `reservePathsOverlap` | `reserve.go:90` | 100.0% | なし |
| `RunningOrgs` | `reserve.go:233` | 100.0% | なし |
| `ActiveReservation` | `reserve.go:256` | 100.0% | なし |
| `TotalActiveSeats` | `reserve.go:265` | 100.0% | なし |
| `ValidateOrgWideCapacity` | `envelope.go:92` | 100.0% | なし |
| `spawnCapacityErr` | `spawn.go:1063` | 100.0% | なし |
| `reservationDecision` | `reserve.go:283` | 100.0% | なし |
| `idempotentRespawn` | `spawn.go:1085` | 87.5% | `scope_reserved` の追記の失敗(`:1092`) |
| `reserveAgain` | `verbs.go:1225` | 75.0% | すでに予約を持つときの戻り(`:1227`)と、追記の失敗(`:1233`) |
| `orgsToDisband` | `verbs_all.go:309` | 100.0% | なし |
| `withMainWorktreeOrgLimits` | `cli/org.go:148` | 100.0% | なし |
| `MainWorktreeRoot` | `statedir.go:81` | 100.0% | なし |

- そのほかの変わった関数: `checkCapacityAndStart` 69.2%(`scope_reserved` と、前からある `spawn_started` の追記の失敗)、`dryRunSpawn` 93.7%、`autonomousScopeGateErr` 100.0%、`CloseDeferredSelfWorkspace` 100.0%、`newOrgSpawnRuntime` 90.0%(`newOrgRuntimeAt` の失敗)、`orgReservation` 75.0%(台帳の読み込みの失敗)、`printStatusTable` / `printStatusJSON` 100.0%、`config.Load` 92.9%、`config.Default` 100.0%
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 通らない文は、`reserveAgain` の `:1227` を除くと、台帳への追記や読み込みが失敗する経路だけである。`:1227` は下の mutation X18 の生き残りにあたる

### Mutation

各 mutation は、対象のファイルの写しを scratchpad に作って 1 か所だけ書き換え、`go test -overlay=<json> -count=1 -vet=off <パッケージ>` で流した。パッケージは、そのファイルを持つパッケージと、CLI から届くものは `internal/cli` も含めた全体である(`-run` で絞っていない)。worktree のファイルは書き換えていない。M14 だけは、全体を流す代わりに競合のテストを `-count=10` で流した。

依頼の 14 種:

| # | Mutation | Result | Red になったテスト(主なもの) |
| --- | --- | --- | --- |
| M1 | `ValidateOrgWideCapacity` の max_orgs の判定を外す | red | `TestValidateOrgWideCapacity`、`MaxOrgs_NewOrgRejectedAtLimit`、`MaxOrgs_StoppedOrgHoldsItsSlotUntilDisbanded`、`ConcurrentSpawns_MaxOrgsNeverExceeded`、CLI の `TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml` ほか、計 9 本 |
| M2 | max_total_seats の判定を外す | red | `TestValidateOrgWideCapacity`、`MaxTotalSeats_Boundary`、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded`、CLI の `MainWorktreeRalphToml_OnlyOrgWideLimitsTaken`、追加した `CloseDeferredSelfPane_..._ReactivationCanExceedMaxTotalSeats` ほか、計 7 本 |
| M3 | `RunningOrgs` が `rejected` だけの座席の org も数える | red | `TestRunningOrgs`、`MaxOrgs_NewOrgRejectedAtLimit`、`ConcurrentSpawns_MaxOrgsNeverExceeded` ほか、計 6 本 |
| M4 | `currentOrgLives` が最後の `disbanded` より前の記録も数える | red | `TestRunningOrgs`、`TestActiveReservation`、`TestReservationBeforeLastDisband`、`Reserve_DisbandReleasesIt`、`DisbandAll_ReservationOnlyOrgDisbanded` ほか、計 8 本 |
| M5 | ディレクトリどうしの重なりで末尾の `/` を外して比べる(区切りの単位を無視) | red | `TestReservePathsOverlap`(`internal/auth/` と `internal/authz/`)、`TestReservationDecision`、`Reserve_OverlapWithRunningOrgRejected` |
| M5b | ディレクトリとファイルの重なりで同じことをする | red | `TestReservePathsOverlap`(`internal/authz.go`、`internal/auth`) |
| M6 | `..` の拒否を外す | red | `TestNormalizeReservePaths_Rejects`、`Reserve_InputRejectedBeforeAnyRecord` |
| M7 | leader 以外の座席の予約を通す | red | `Reserve_InputRejectedBeforeAnyRecord`、CLI の `TestOrgSpawn_Reserve_LeaderOnly` |
| M8 | `idempotentRespawn` が予約を判定しない | red | `Reserve_ExistingLeader`、追加した `Reserve_ExistingLeader_Phase2`、CLI の `TestOrgStart_Reserve_SameListPassesDifferentListRejected` |
| M9 | すでに予約がある org に違う一覧を通す | red | `TestReservationDecision`、`Reserve_SameSetPassesDifferentSetRejected`、`Reserve_ExistingLeader`、CLI の `SameListPassesDifferentListRejected` |
| M10 | `orgsToDisband` から `scope_reserved` を外す | red | `TestOrgsToDisband_Definition`(「reservation only, never disbanded」など) |
| M11 | `CloseDeferredSelfWorkspace` が `reserveAgain` を呼ばない | red | `CloseDeferredSelfWorkspace_CloseFails_ReservationRestored`、`..._RiskWindowClearedByRetry` |
| M12 | main worktree の `ralph.toml` を読むが値を使わない(打った場所の設定のまま) | red | CLI の `ReadFromMainWorktreeRalphToml`(サブディレクトリと linked worktree の 2 件)、`OnlyOrgWideLimitsTaken` |
| M12b | main worktree の `ralph.toml` を読まない | red | M12 の 2 本と `TestOrgStart_MainWorktreeRalphTomlLoadError` |
| M13 | `--state-dir` や env で台帳を決めたときも main の上限を使う | red | CLI の `ReadFromMainWorktreeRalphToml` の「--state-dir naming the shared ledger」と「RALPH_ORG_STATE_DIR naming the shared ledger」 |
| M14 | `withManifestLock` が flock を取らない | red(30 回中 29 回) | `-count=10` で `ConcurrentSpawns_MaxOrgsNeverExceeded` 10/10、`ConcurrentSpawns_MaxTotalSeatsNeverExceeded` 10/10、`ConcurrentReservations_OnlyOneOfOverlappingWins` 9/10 が red。既存の `MaxSeatsNeverExceeded` も 9/10、`TestWithManifestLock_SerializesConcurrentCallers` は 10/10 |

追加の 24 件(境界、読み戻し、CLI の周辺。P1 は mutation ではなく、追加したテストが区別できるかの確認):

| # | Mutation | Result | Red になったテスト |
| --- | --- | --- | --- |
| X1 / X2 | max_orgs / max_total_seats の `>=` を `>` にする | red / red | `TestValidateOrgWideCapacity`、`ZeroOrgWideLimits_Reject`、境界のテスト |
| X3 | max_orgs を走っている org にも掛ける | red | `MaxOrgs_NewOrgRejectedAtLimit`(org-a の 2 席目)、`TestValidateOrgWideCapacity` |
| X4 | dry-run の org の記録も数える | red | `TestRunningOrgs`、`TestActiveReservation`、`Reserve_DryRunPreviewsAndHoldsNothing` |
| X5 | 読めない `scope_reserved` を無視する(repo 全体として扱わない) | red | `TestActiveReservation`(「unreadable is the whole repo」) |
| X6 | scope のゲートが `Reserve` を見ない | red | `Reserve_SatisfiesAutonomousScopeGate`、`--scope` なしの `--reserve` で leader を立てる CLI のテスト 5 本 |
| X7 | `--config` を渡しても main の上限を使う | red | CLI の `ReadFromMainWorktreeRalphToml`(「--config with max_orgs = 10」) |
| X8 | main から `max_seats` も取る | red | CLI の `OnlyOrgWideLimitsTaken`(「max_seats stays the caller's」) |
| X9 | main の `ralph.toml` の読み込みの失敗を無視する | red | CLI の `MainWorktreeRalphTomlLoadError` |
| X10 | `status` が `reserved:` の行を出さない | red | CLI の `TestOrgStatus_ShowsReservation`、`ReservationWithoutSeatsAndAfterDisband` |
| X11 | 立っている leader への予約の拒否で `rejected` を書く | red | `Reserve_ExistingLeader`(記録が増えない、の検査)、CLI の `SameListPassesDifferentListRejected` |
| X12 | dry-run の trail に `scope_reserved` を出さない | red | `Reserve_DryRunPreviewsAndHoldsNothing` |
| X13 / X14 | `config.Load` が `max_orgs` / `max_total_seats` の 0 を通す | red / red | `TestLoad_OrgFleetLimitsBoundary`(X13 は CLI の `MainWorktreeRalphTomlLoadError` も) |
| X15 / X16 | 既定の 10 / 30 を 11 / 31 にする | red / red | `TestDefault_Org`、`TestDefaultsLockStep` |
| X17 | 予約のパスを並べ替えない | red | `TestNormalizeReservePaths`、`Reserve_RecordedBeforeSpawnStarted` ほか、計 4 本 |
| X18 | `reserveAgain` の「すでに予約を持つなら書かない」を外す | **生き残り** | なし(下の Test gaps 3) |
| X19 | `MainWorktreeRoot` の末尾の検査を外す | red | `TestMainWorktreeRoot` |
| X20 | `reservedPathsFromDetails` が注記(`restored: …`)を切らない | red | `TestScopeReservedDetails_RoundTrip`、`..._ReservationRestored` |
| X21 | `checkCapacityAndStart` が予約を記録しない | red | 14 本 |
| X22 | `org_workspace_closed` で workspace を外さない | red | `TestRunningOrgs`(「workspace created then closed」) |
| X23 | `reservationDecision` がほかの org の予約を見ない | red | 7 本 |
| X24 | Phase 2 の idempotent の戻り(`spawn.go:760`)で予約を判定しない | red(テスト追加前は生き残り) | 追加した `TestOrgSpawn_Reserve_ExistingLeader_Phase2` |
| P1 | (確認)pane の補償が max_total_seats に達していたら座席を戻さない、という仮の修正 | red | 追加した `CloseDeferredSelfPane_..._ReactivationCanExceedMaxTotalSeats`。このテストは今の振る舞い(上限を 1 つ超える)を固定しており、補償が上限を見るように変わると落ちる |

- mutation は 2 回に分けて流した。1 回目(M3〜M6、M10、M11、X1〜X5、X12、X14〜X20、X22〜X24)は、`verbs_test.go` に V-2 のテストを足したあと、`spawn_test.go` に Phase 2 のテストを足す前に流した。2 回目(M1、M2、M7〜M9、M12、M12b、M13、X6〜X11、X13、X21、X24、P1)と M14 は、2 本とも足したあとに流した。
- 1 回目で生き残ったのは X18 と X24 の 2 件である。X24 は Phase 2 のテストを足し、2 回目で red になることを確かめた。X18 にはテストを足していないので、流し直していない。
- M14 の 30 回中 1 回の pass は、goroutine の読みと書きがたまたま直列に並んだ回とみている。ロックがあれば `-count=3` の 3 回とも、`-race` でも通る。

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 今ある `max_seats` の判定と、その競合のテスト | 保たれている | `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` が `-count=3` と `-race` で通る。`spawnCapacityErr` は `ValidateSpawnCapacity` を最初に呼ぶ |
| scope のゲート(AC-2b) | 保たれている | `--scope` も `--reserve` もない autonomous の spawn は今どおり拒否される(`Reserve_SatisfiesAutonomousScopeGate` の 2 つ目の検査)。X6 で red |
| spawn の idempotent(立っている座席は上限の判定の前に返す) | 保たれている | `MaxTotalSeats_Boundary` の上限ちょうどでの respawn がイベントを増やさない。Phase 2 の idempotent の既存テスト(`StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent`)も通る |
| 2 段目の自分の close の補償(workspace と pane) | 保たれている | 既存の `CloseDeferredSelfWorkspace_*` と `CloseDeferredSelfPane_*` がすべて通る。予約のない org では補償の記録が増えない(`..._ReservationRestored` の「no reservation, nothing restored」) |
| `disband --all` の対象 | 保たれている | `TestOrgsToDisband_Definition` の既存の場合がすべて通り、予約だけの org が加わった。M10 で red |

### Test gaps

1. 追加したテスト(c3a95c48、`internal/org/spawn_test.go` と `internal/org/verbs_test.go` だけ、+75 行。本番のコードは変えていない):
   - `TestOrgSpawn_Reserve_ExistingLeader_Phase2`: AC14 の Phase 2 側の穴。stale な saga の補償の間に別の呼び出しが leader を立てた場合、Phase 2 の新しい読みで見つけた leader にも予約を判定して記録する。既存の `afterStaleCompensation` の継ぎ目を使った。X24 が生き残ったので足し、red になることを確かめた
   - `TestOrgCloseDeferredSelfPane_CloseFails_ReactivationCanExceedMaxTotalSeats`: verify の V-2 の pane の経路を、今の振る舞いのまま固定したもの。座席の `stopped` から補償までの間に別の座席が立つと、補償は上限を見ずに座席を戻すので `max_total_seats` を 1 つ超える(1 席の上限で 2 席が動く)。新しい座席は拒否され、herdr が答えるようになってからほかの pane で stop をやり直すと 1 席に戻る。要件ではなく記録であり、補償が上限を見るように直すときは、このテストも一緒に直す。P1 で、そうした修正を入れるとこのテストが落ちることを確かめた
   - どちらも不要なら、この commit だけを戻せば元に戻る
2. V-2 の窓そのものは残る。workspace の経路は既存の `..._RiskWindowClearedByRetry` が、pane の経路は追加したテストが、上限を超えた状態と打ち直しで解けることを固定している。補償が上限や重なりを見るようにするかは、/sync-docs か cross-review の判断になる
3. X18(`reserveAgain` の `verbs.go:1226-1228`、すでに予約を持つ org には書かない)はテストがない。この戻りが効くのは、`disbanded` のあとに同じ org_id が `--reserve` で立て直され、それを `CloseDeferredSelfWorkspace` の冒頭の読みが見た場合だけである。`ralph org disband` は `Disband` のすぐあとに同じプロセスでこの読みをするので、その間に立て直しが入る窓は短い。現実に起きやすいのは、herdr の呼び出しの間(最大 3 回 × 10 秒)に立て直される場合で、そのときは読みが古いのでこの戻りでは防げない(self-review F-5)。そのため AC の穴とはみなさず、テストを足さなかった。F-5 を直す(追記の直前に読み直す)なら、そのテストでこの戻りも押さえられる
4. 別プロセスの競合は再現していない。AC4 の 3 本は同じプロセスの goroutine で flock を競わせる。flock は open ごとに掛かるので、プロセスが分かれても同じ仕組みで直列になるはずだが、未確認である
5. 台帳への追記と読み込みの失敗の経路(`checkCapacityAndStart` の `scope_reserved`、`idempotentRespawn`、`reserveAgain`、`orgReservation`)はテストがない。どれもエラーを返すだけの分岐で、前からある `spawn_started` の追記の失敗と同じ扱いである
6. 予約のパスは大文字小文字を区別して比べる(self-review の Coverage gaps)。macOS の既定のファイルシステムで `Internal/` と `internal/` が重ならないことは、テストでも固定していない。plan に書かれていない扱いで、tech-debt の候補として /sync-docs に渡っている
7. bare リポジトリの linked worktree(台帳の source が `git-toplevel`)で main の `ralph.toml` を読まないことは、`TestMainWorktreeRoot` の単体の場合だけで見ている。CLI からは打っていない(verify の V-1 と同じ)
8. 本物の herdr と agmsg では動かしていない

### Verdict

- Pass: pass。`run-test.sh`(full)は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ)。`-race`、新しいテストの `-count=3`、`go test ./... -count=1` も 0 failed。mutation は依頼の 14 種(16 件)がすべて red で、追加の 24 件のうち X24 はテストを足して red になり、X18 だけが生き残った。AC1〜AC11 と AC13〜AC15 をテストの実行で確かめた
- Fail: なし
- Blocked: なし。Test gaps 2〜8 は merge を止めない
