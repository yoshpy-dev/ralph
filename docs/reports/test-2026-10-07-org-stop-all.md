# Test report: org-stop-all

- Date: 2026-10-08(JST。実行の記録は UTC の 2026-10-07 16:37〜17:21)
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 2(`cycle-count.json` は 2、既定の上限の最後の回)
- Scope: branch feat/org-stop-all の HEAD e3bef918 と origin/main(f423f230)の差分。cycle 1 の test(ad3baa72)以降のコードの変更は、223258eb(ヘルプ文)と 53807a82(S8: 後回しの close が失敗したときの補償、`CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace` / `closeDeferredSelf` の `force` 引数)だけ。behavioral test だけを実行した(静的解析は /verify の担当)。見たのは AC16 と、S8 のあとも AC10・AC12・AC14 が成り立つか
- Added tests: a6cc5fbf(`test: pin the S8 compensation guards found by mutation`、テストファイル 2 本、+118/-3)。本番のコードは変えていない
- Evidence: `docs/evidence/test-2026-10-07-org-stop-all.log` の末尾に cycle 2 の節を足した(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-07-163756.log`(HEAD)と `docs/evidence/verify-2026-10-07-171445.log`(テスト追加後)
- 構成: cycle 1 の本文は、`docs/tech-debt/README.md` の 168 行が「Test gaps 1 to 6、Mutation M57、Coverage Notes」を指しているので、末尾の付録に残した。見出しを 1 段下げ、cycle 1 の Verdict の見出しを「Verdict(cycle 1)」にしたほかは原文のまま。行番号は de154798 のもの

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(HEAD e3bef918) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 1(下の注) | 316 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(HEAD の `git archive` の写し) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | rc 0 |
| mutation 64 件(scratchpad の `git archive` の写し 3 つで並行に実行。worktree には触れていない) | ビルドが通った 64 件 | HEAD のテストで red 53 件、追加後に 63 件 | 生き残り HEAD で 11 件、追加後に 1 件 | - | 64 件は 3 本並行で約 7 分、生き残りの流し直しを合わせて約 30 分 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1`(テスト追加後) | 4 パッケージ | 4 | 0 | 0 | 78 s、rc 0。DATA RACE の報告なし |
| S8 のテスト `go test ./internal/org/ ./internal/cli/ -count=3 -run 'CloseDeferredSelf\|OwnCloseFails\|Force_OwnCloseFails'`(テスト追加後) | top-level 42 回(14 本 × 3)、subtest 120 回 | すべて | 0 | 0 | 13 s、rc 0 |
| S8 が書き換えた既存のテスト 5 本を `-count=3`(上の `-run` に名前が合わないもの) | top-level 15 回、subtest 27 回 | すべて | 0 | 0 | rc 0 |
| `go test ./... -count=1 -coverprofile`(テスト追加後) | 8 パッケージ | 8 | 0 | 0 | rc 0 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(テスト追加後) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 1 | 364 s、rc 0 |

- 2 回の `run-test.sh` は `Requested scope: full` で走った。中の `go test ./...` には `-count=1` がなく、1 回目は `internal/org` 以外、2 回目は `internal/cli` と `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを別に流した(2 行目と 7 行目)。
- shell の件数は cycle 1 と同じ数え方(各 suite の区間で `PASS` / `FAIL` / `SKIP` で始まる行を数え、`PASS: 324` や `PASS: 47 / 47` のような集計行は除く)。suite ごとの件数は 2 回の実行で同じで、合計 2,047 件も cycle 1 と同じ。この差分は shell のテストを変えていない。
- SKIP の 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の例で、この機械の git が 2.49.0 なので飛ばし、stub の古い git が同じ場合を見ている。この行は cycle 1 のログ(`verify-2026-10-07-135142.log`、`-144820.log`)にもあったが、cycle 1 のレポートは Skipped を 0 と書いていた。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回の実行のどちらでも出なかった(123 / 123)。単独の再実行はしていない。
- `-race` と 2 つの `-count=3` は追加したテストを含む worktree で流した。HEAD と違うのはテストファイル 2 本だけ。
- top-level の Test 関数の数(`go test -list`、テスト追加後): `internal/org` 366(cycle 1 の 357 に S8 の 7 本と今回の 2 本)、`internal/org/driver` 48、`internal/cli` 480(478 に S8 の 2 本)。
- 実行のあと、main のチェックアウトと worktree のどちらにも `.harness/state/org` はなく、main の `git status --porcelain` は 0 行だった。テストが共通の台帳に書くことはなかった。
- 追加したテストのコンパイル確認として `gofmt -l internal/` を流した(出力なし)。判定には使っていない。

## AC16 とテストの対応

| AC16 の要件 | 主なテスト | 結果 |
| --- | --- | --- |
| 自分の pane の close が失敗したら座席を active に戻す | `TestOrgCloseDeferredSelfPane_CloseFails_SeatReactivated`、`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose/pane`、`TestOrgCloseDeferredSelfPane_Outcomes`、追加した `_SameSeatIDInAnotherOrg_KeepsOwnAgentName` | pass。N01・N05・N06・N31〜N35・N39・N41・N46・N48・N51 が red。N36 は HEAD で生き残り、追加したテストで red |
| 自分の workspace の close が失敗したら workspace を開き、自分の座席を active に戻す | `TestOrgCloseDeferredSelfWorkspace_CloseFails_ReopenedAndReactivated`、`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose/workspace`、`TestOrgCloseDeferredSelfWorkspace_Outcomes`、追加した `_CloseFails_CallerPaneSeatNotReactivated`(3 例) | pass。N02〜N04・N08・N38・N40・N47・N49・N50 が red。N07・N13・N14 は HEAD で生き残り、追加した 3 例で red |
| `ralph org status` でも active、終了コード 1 | CLI の `TestOrgStopDisband_OwnCloseFails_LedgerRestoredForRetry`(追加後は `stop --seat`、`disband --org-id`、`stop --all`、`disband --all` の 4 例) | pass。N25・N28(`--all` の経路だけ常に force)は HEAD で生き残り、追加した 2 例で red |
| 同じ `stop` / `disband` の打ち直しか、ほかの pane の `--all` で閉じ直す | `TestOrgCloseDeferredSelfPane_CloseFails_RetryClosesIt`(3 例)、`TestOrgCloseDeferredSelfWorkspace_CloseFails_RetryClosesIt`(2 例)、`TestOrgCloseDeferredSelfPane_AfterDisband_ReactivatedPastDisbanded`、CLI の上のテストの後半 | pass。補償を落とす mutant(N01〜N04)はここでも red |
| `--force` は補償を書かず、警告を出し、終了コード 0 | `TestOrgCloseDeferredSelf_Force_NoCompensation`、CLI の `TestOrgStopDisband_Force_OwnCloseFails_WarnsExitsZero`(追加後は 4 例) | pass。N09〜N12・N16・N17・N22・N23・N26・N29・N30 が red。N24・N27(`--all` の経路で force を渡さない)は HEAD で生き残り、追加した 2 例で red |
| 補償も書けないときは、手で閉じる herdr のコマンドを出す | `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose`(4 例)、`_Outcomes` の表 | pass。N15・N18〜N21・N42・N43 が red。N44・N45(書けなかったのに「記録し直した」と言う)は HEAD で生き残り、追加した 1 行の確認で red |
| `disband --help` と `/org` skill の 1 文 | なし(文書なので /verify の担当) | - |

AC10・AC12・AC14 の確かめ直し: cycle 1 の安全側の mutant を S8 のあとのコードに合わせて書き直し、13 件すべてが red になった(下の表の R 行)。AC10 の `--force` で後回しの close が失敗する経路は、cycle 1 のテストにはなかった。S8 と今回の追加で、4 つのコマンドの形すべてが通るようになった。

## 依頼にあった確認点

- 依頼の mutation はすべて流した。補償を飛ばす(pane は N01、workspace は N02、両方は N04)、台帳がすでに active / open でも補償する(N05・N06・N08 は red、座席の pane の照合を外す N07 は生き残り)、`--force` でも補償する(org 側の N09・N10、CLI が `force` を渡さない N11・N12)、workspace だけ戻して自分の座席を戻さない(N03)、別の org の座席を戻す(N13 は生き残り)、手で閉じる案内を落とす(7 か所、N15〜N21)、`--force` で終了コード 1 を返す(N22)、写し元の記録を間違える(`HerdrAgentName` を `stopped` から取る N31、落とす N32、ほかの欄を落とす N33〜N35 は red、別の org の同じ座席 id から取る N36 は生き残り)。生き残りは、N37 を除いてテストを足して red にした。
- cycle 1 の安全側の mutant のうち安いものを流し直した。`confirmSeatPane` を C-c の前に呼ばない(R01、cycle 1 の M01)、自分の pane を先に閉じる(Stop の中ですぐ閉じる R15、確認の前に自分と判定する R16、disband と stop --all で最後に回さない R28・R42、CLI が出力の前に閉じる R69・R70)。どれも red で、退行はない。
- verify が書いた `stop --all --force` で自分の pane の close が失敗する経路(`internal/cli/org.go:756`)は、テストがないことを N24 で確かめた(org と cli のテストで生き残り)。同じ穴が `disband --all --force`(`:1046`、N27)と、`--force` なしの `--all` の 2 つ(N25・N28、常に force を渡す書き換え)にもあった。AC10 と AC16 の両方に当たるので、CLI の 2 つのテストに `--all` の行を 2 例ずつ足した。

## Mutation

scratchpad に `git archive HEAD` で写しを作り、1 件ずつ書き換えて対象のパッケージのテストを流し、写しの元のファイルで戻した。`internal/org` の書き換えは org のテストを先に流し、生き残ったときだけ cli のテストを流した。cli のテストは cycle 1 と同じく `-run 'TestOrgStop|TestOrgDisband|TestOrgLegacyLedger|TestWithPrefixOnce|TestOrgCleanupVerbs'` で絞った。HEAD で生き残った org 側の 7 件(N07・N13・N14・N36・N37・N44・N45)は、cli の全テスト(`-run` なし)でも生き残ることを確かめた。期限切れや test timeout で落ちた mutant はなく、red はどれも assert の失敗による(並行の負荷で落ちたものではない)。最後に 3 つの写しが元のファイルと一致することを `diff -rq` で確かめた。

| 対象 | red になった mutation | HEAD での生き残り |
| --- | --- | --- |
| 補償の有無(`CloseDeferredSelfPane` / `CloseDeferredSelfWorkspace`) | N01〜N04 | なし |
| 補償の条件(`reactivateSeat` / `reopenWorkspace` / 自分の座席の選び方) | N05・N06・N08 | N07(座席の pane の照合を外す)、N13(別の org の座席を戻す)、N14(`HERDR_PANE_ID` が空でも探す) |
| `--force`(org と CLI) | N09〜N12・N22・N23・N26・N29・N30 | N24・N27(`--all` の経路で force を渡さない)、N25・N28(`--all` の経路で常に force) |
| 手で閉じる案内 | N15〜N21 | なし |
| 写す欄(`reactivateSeat`、`lastHerdrAgentName`、`reopenWorkspace`) | N31〜N35・N38〜N40 | N36(別の org の同じ座席 id の名前を取る)、N37(dry-run の記録も見る) |
| エラー文と `selfCompensation` | N41〜N43・N51 | N44・N45(追記に失敗したのに「記録し直した」と返す) |
| `closeSelfPane` / `closeSelfWorkspace` / `lastWorkspaceEvent` | N46〜N50 | なし |
| cycle 1 の安全側の流し直し(M01・M15・M16・M20・M22〜M25・M28・M42・M69・M70・M77b) | R01・R15・R16・R20・R22〜R25・R28・R42・R69・R70・R77 | なし |

- HEAD で生き残った 11 件のうち 10 件は、a6cc5fbf のテストで red になった。追加後の写し(HEAD のコードに worktree のテストファイル 2 本を重ねたもの)で流し直したのはこの 11 件だけ。追加のテストは既存のテストの assert を変えず、例と関数と確認を 1 行足しただけなので、HEAD で red だった 53 件が生き残りに戻ることはない。
- 新しいテストは、それぞれ狙った mutant だけを落とした。N07 は `CallerPaneSeatNotReactivated/seat_now_in_another_pane`、N13 は `/seat_of_another_org`、N14 は `/no_HERDR_PANE_ID,_a_seat_with_no_pane_id`、N24 と N27 は Force のテストの `stop_--all_--force` と `disband_--all_--force`、N25 と N28 は LedgerRestored のテストの `stop_--all` と `disband_--all` で落ちた。
- 残った N37(`lastHerdrAgentName` から `!ev.DryRun` を外す)は等価とみなした。dry-run の spawn の記録は `HerdrAgentName` を持たない(`dryRunSpawn` の基になる記録に欄がない、`internal/org/spawn.go:1581`)ので、条件を外しても選ばれる記録は変わらない。
- 生き残った 10 件のある関数は、HEAD の時点で statement coverage が 100% だった(`lastHerdrAgentName` だけ 80.0%)。どれも条件の一部を外す書き換えで、行は通っても条件の片側を見るテストがなかった。

## 追加したテスト(a6cc5fbf)

| テスト | 閉じた穴 | red / green の確認 |
| --- | --- | --- |
| CLI の `TestOrgStopDisband_OwnCloseFails_LedgerRestoredForRetry` に `stop --all`、`disband --all` の 2 例 | `internal/cli/org.go:756` と `:1046` が `force` を `closeDeferredSelf` に渡すこと(`--force` なしで補償し、終了コード 1) | N25・N28 が red、HEAD のコードで pass |
| CLI の `TestOrgStopDisband_Force_OwnCloseFails_WarnsExitsZero` に `stop --all --force`、`disband --all --force` の 2 例 | 同じ 2 行の `--force` の側(補償なし、警告、終了コード 0)。verify の指摘 | N24・N27 が red、HEAD のコードで pass |
| `TestOrgCloseDeferredSelfWorkspace_CloseFails_CallerPaneSeatNotReactivated`(3 例) | `CloseDeferredSelfWorkspace` が戻す自分の座席の条件(`internal/org/verbs.go:1103-1104` の `ownPane != ""` と `orgID == last.OrgID`、`reactivateSeat` の `seat.PaneID != paneID`、`:1177`)。どの例も workspace だけを戻し、座席は stopped のまま | N07・N13・N14 が red、HEAD のコードで pass |
| `TestOrgCloseDeferredSelfPane_CloseFails_SameSeatIDInAnotherOrg_KeepsOwnAgentName` | `lastHerdrAgentName` の `ev.OrgID == orgID`(`:1229`)。別の org に同じ座席 id があっても、自分の org の spawn の herdr agent name を写す | N36 が red、HEAD のコードで pass |
| `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose` に 1 行 | 追記に失敗したときのエラーが「記録し直したので打ち直せば閉じる」と言わないこと(`:1186`、`:1206` が "" を返す) | N44・N45 が red、HEAD のコードで pass |

## Coverage

- Statement(HEAD → テスト追加後): `internal/cli` 85.3% → 85.3%、`internal/org` 92.6% → 92.6%、`internal/org/driver` 93.1%(変わらず)。cycle 1 のテスト追加後はそれぞれ 85.3%・92.4%・93.1%。ほかの 5 パッケージは cycle 1 と同じ(`config` 92.3%、`insights` 86.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Branch: Go の標準ツールでは取れない
- Function(各パッケージ自身のテストでの値。HEAD とテスト追加後で同じ):
  - `internal/org/verbs.go`: `CloseDeferredSelfPane` 100%、`closeSelfPane` 100%、`CloseDeferredSelfWorkspace` 100%、`closeSelfWorkspace` 100%、`selfCompensation.add` 100%、`selfCompensation.errorFor` 100%、`reactivateSeat` 100%、`reopenWorkspace` 100%、`lastSeatOnPane` 100%、`lastWorkspaceEvent` 100%、`lastHerdrAgentName` 80.0%
  - `internal/cli/org.go`: `closeDeferredSelf` 100%、`newOrgStopCmd` 96.8%、`newOrgDisbandCmd` 100%
- Notes: 追加したテストで数値は動いていない。S8 の関数は HEAD の時点で行としてはすべて通っていて、今回のテストは条件の片側を固定するものだったため。通らない行は `lastHerdrAgentName` の `return ""`(`:1233`。herdr agent name を持つ記録が 1 つもない座席、つまり古い ralph が spawn した座席)だけ。cycle 1 の Test gaps 2 が挙げた後回しの close の台帳の読み込みの失敗(当時の `verbs.go:1003`、`:1045`)は、S8 の `TestOrgCloseDeferredSelf_NothingRestored_NamesTheManualClose` が通すようになった

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| S8 で書き換わった既存のテストの意図 | 弱めていない | S8 のテストの差分の削除は 18 行で、`CloseDeferredSelf*` の呼び出しに `force` 引数を足した書き換え、`readOnlyManifest` をパッケージの関数に移したこと、`_Outcomes` の表の 2 行の期待する文言を手で閉じる案内まで伸ばしたことだけだった。S8 が書き換えた 5 本(`TestOrgDisband_OwnWorkspace_ClosedLastByCaller` ほか)は `-count=3` で pass |
| AC12(自分の pane と workspace を最後に閉じる) | pass | `TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput`。R15・R16・R28・R42・R69・R70 が red |
| AC14(閉じる前の持ち主の確認) | pass | R01・R22・R24 が red。後回しの close の確認に落ちたときも補償を書くことは `TestOrgCloseDeferredSelf_RecheckedBeforeTheClose` が見ている(N01・N02 が red) |
| AC10(`--force` は終了コード 0) | pass、`--all` の 2 経路も固定した | N22〜N30 がすべて red(N24・N25・N27・N28 は追加後) |
| cycle 1 のテスト追加(f63ae025)で閉じた穴 | pass | M20 に当たる R20 が red。ほかは cycle 1 のテストのまま pass |
| テストが main のチェックアウトの台帳に書かない | pass | 全実行のあと main と worktree に `.harness/state/org` はない |

## Test gaps

1. N37(`lastHerdrAgentName` の `!ev.DryRun` を外す)は等価なので、テストを足していない(上の Mutation の注)。
2. `lastHerdrAgentName` の `return ""`(`internal/org/verbs.go:1233`)は通らない。herdr agent name を持たない古い座席が補償で戻る場合で、補償の記録の `HerdrAgentName` は空になる。名前で座席を呼ぶ動詞は `resolvedHerdrAgentName` で名前を作り直すので、今の動きに問題はないと読んだ(確かめたのはコードだけ)。
3. self-review の C2-1(補償が driver を呼ぶ前に読んだ台帳の写しで判断する窓)と C2-2(同じ org の古い `stopped` の座席が、id を振り直された自分の pane と一致すると戻ること)にはテストがない。C2-2 は今のコードの動きで、追加した `CallerPaneSeatNotReactivated` は座席の pane と org の照合だけを固定し、持ち主の確認(tab の label)はしない今の動きを変えていない。
4. 本物の herdr では動かしていない。後回しの close の失敗は、org では Go の fake の `paneCloseErrs` / `workspaceCloseErrs`、cli では sh の stub の `ORG_STUB_CLOSE_FAIL_IDS` で起こしている。
5. `NothingRestored` に足した確認は、エラー文の "retries the close" という言い回しの有無で見ている。`errorFor` の文言が変わると、この確認は何もせずに通る。
6. cycle 1 の Test gaps 1〜6(付録)は、上の Coverage の Notes に書いた 1 点(後回しの close の台帳の読み込みの失敗)を除いて残っている。

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh`(full)は HEAD とテスト追加後の 2 回とも rc 0(shell 2,047 件 pass、SKIP 1 件は環境による例、Go 8 パッケージ ok)。`-count=1` の Go 全体、`-race`、S8 のテストと S8 が書き換えた既存のテストの `-count=3` もすべて pass で、flake は出なかった。mutation は 64 件中、HEAD のテストで 53 件、テスト追加後に 63 件が red になった。残りの 1 件(N37)は等価
- Fail: なし
- Blocked: なし

## 付録: cycle 1 の test report(原文)

- Date: 2026-10-07(JST。実行の記録は UTC の 2026-10-07 13:51〜14:54)
- Plan: docs/plans/active/2026-10-07-org-stop-all.md
- Tester: tester subagent (Claude Opus 5.5)、pipeline cycle 1
- Scope: branch feat/org-stop-all の HEAD de154798 と origin/main(f423f230)の差分(29 ファイル、+5832/-165)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の Test plan の unit・integration・regression・edge case と、AC1〜AC15 のうちテストで確かめられる部分を見た。AC9 の文書の部分は /verify の担当で、ここでは leader の雛形の順序だけをテストにした
- Added tests: f63ae025(`test: close org-stop-all coverage gaps found by mutation`、テストファイル 3 本、+134 行)。本番のコードは変えていない
- Evidence: `docs/evidence/test-2026-10-07-org-stop-all.log`(`docs/evidence/*.log` は gitignore の対象なので commit しない)。`run-test.sh` 自身のログは `docs/evidence/verify-2026-10-07-135142.log`(HEAD)と `docs/evidence/verify-2026-10-07-144820.log`(テスト追加後)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(既定の changed、HEAD de154798) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 673 s、rc 0 |
| `go test ./... -count=1 -coverprofile`(HEAD) | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 103 s、rc 0 |
| `go test ./internal/cli/ ./internal/org/ ./internal/org/driver/ -count=1 -v -coverpkg=./internal/org/...,./internal/cli/`(HEAD、パッケージをまたいだ coverage) | 3 パッケージ | 3 | 0 | 0 | rc 0 |
| `go test -race ./internal/org/... ./internal/cli/... -count=1`(HEAD) | 4 パッケージ | 4 | 0 | 0 | 109 s、rc 0。DATA RACE の報告なし |
| 差分で増えた 69 本(org 50、driver 6、cli 13)を `-count=3 -run '^(...)$'` | top-level 207 回、subtest 411 回 | すべて | 0 | 0 | 91 s |
| 期限の 16 本を `-count=10`(`Unanswered\|TimesOut\|Timeout\|WaitDelay\|GrandchildHolding`)、同時に `internal/cli` 全体を裏で実行 | 160 回 | 160 | 0 | 0 | 前面 rc 0、裏 rc 0 |
| 追加したテストを `-race -count=3` | org の top-level 12 回、cli の 3 回と subtest 30 回 | すべて | 0 | 0 | rc 0 |
| `go test ./... -count=1 -coverprofile`(テスト追加後) | 8 パッケージ | 8 | 0 | 0 | rc 0 |
| `./scripts/run-test.sh`(テスト追加後) | shell 39 ファイル(2,047 件)、Go 8 パッケージ | すべて | 0 | 0 | 374 s、rc 0 |
| mutation(私が作った 82 件。scratchpad の `git archive` の写し 3 つで並行に実行し、worktree には触れていない) | ビルドが通った 81 件 | HEAD のテストで red 72 件、追加後に 80 件 | 生き残り HEAD で 9 件、追加後に 1 件 | - | 最初の 78 件は 3 本並行で約 11 分、追加の確認を合わせて約 20 分 |

- `run-test.sh` は 2 回とも `Language scope: changed (changed_languages)`、`Language packs selected: golang` で走った。shell の 39 本は `scripts/verify.local.sh` から走る。件数の数え方は前回と同じで、各 suite の区間で `PASS` / `FAIL` で始まる行を数え、`PASS: 47 / 47` のような集計行は除いた。2,047 件は前回(org-state-dir-common)と同じで、この差分は shell のテストを変えていない。
- `run-test.sh` の中の `go test ./...` は、1 回目は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(2 行目)。
- 依頼にあった `tests/test-secret-scan.sh` の flake は、2 回の実行のどちらでも出なかった(123 / 123)。単独の再実行はしていない。
- top-level の Test 関数の数(`go test -list`、テスト追加後): `internal/org` 357、`internal/org/driver` 48、`internal/cli` 478。
- 実行のあと、main のチェックアウトと worktree のどちらにも `.harness/state/org` はなく、main の `git status --porcelain` は 0 行だった。テストが共通の台帳に書くことはなかった。
- 追加したテストのコンパイル確認として、`gofmt -l internal/` と `go vet ./internal/org/ ./internal/cli/` を 1 回ずつ流した(どちらも出力なし)。判定には使っていない。静的解析の判定は /verify のレポートのとおり。

### AC とテストの対応

| AC | 主なテスト | 結果 |
| --- | --- | --- |
| AC1 C-c → close、見つからなければ閉じ済み | `TestOrgStop_CallOrder_CtrlCThenCloseThenLeaveThenStopped`、`_PaneNotFound_CountsAsClosed`、`_PaneGoneAtCheck_RecordsAlreadyClosed`、既存の `_ExistingSeat_RecordsPaneAndLeaveOutcomes`(期待値が `pane=ok` から `ctrl_c=ok` と `pane=closed` に変わった) | pass。M01・M09・M18 が red |
| AC2 閉じられなければ `stop_failed`、打ち直しで `stopped` | `TestOrgStop_CloseFails_RecordsStopFailedThenRetryStops`、CLI の `TestOrgStopAll_OneCloseFails_ExitsOneThenRerunStopsIt` | pass。M11・M12・M13 が red |
| AC3 disband は座席 → workspace → `disbanded`、失敗なら `disbanded` なし | `TestOrgDisband_ClosesSeatPanesThenWorkspace`、`_SeatCloseFails_LeavesWorkspaceAndRetries`、`_WorkspaceCloseFails_NoDisbandedThenRetry`、`_WorkspaceNotFound_CountsAsClosed`、追加した `_WorkspaceGoneAtCheck_CountsAsClosed`、CLI の `TestOrgDisband_OneOrgSeatFails_NoDisbandedLine` | pass。M26・M27・M34・M37・M72 が red。M79 は HEAD で生き残り、追加したテストで red |
| AC4 `stop --all` | `TestOrgStopAll_StopsEveryActiveSeatAcrossOrgsInOrder`、`_OneSeatFails_OthersStoppedThenRetryStopsIt`、CLI の `TestOrgStopAll_StopsEveryActiveSeatAcrossOrgs` | pass。M46・M73 が red |
| AC5 `disband --all`、失敗した org の再対象 | `TestOrgsToDisband_Definition`、`TestOrgDisbandAll_FailedOrgNotDisbandedThenRetried`、CLI の `TestOrgDisbandAll_OneOrgFails_ExitsOneAndNextRunRetriesIt` | pass。M47・M50〜M53 が red |
| AC6 同時指定のエラー | `TestOrgStopDisbandAll_ConflictingFlags_RejectedWithoutChanges`(5 例) | pass。M67・M68 が red |
| AC7 `--all --dry-run` | `TestOrgStopAll_DryRun_NoDriverCallsDryRunRecordsOnly`、`TestOrgDisbandAll_DryRun_NoDriverCallsDryRunRecordsOnly`、CLI の `TestOrgStopDisbandAll_DryRun_RecordsWithoutDriverCalls` | pass。M21・M39・M44 が red |
| AC8 記録していない id は閉じない | `TestOrgStop_ClosesOnlyTheSeatsRecordedPane`、`TestOrgDisband_ClosesOnlyThisOrgsRecordedIDs` | pass。M23・M25 が red |
| AC9 leader の雛形の締めの順 | 追加した `TestRenderRolePrompt_Leader_ReportThenDisbandAsLastCommand` | pass。M78(report と disband の手順を入れ替える)は HEAD で生き残り、追加したテストで red。文書のほかの部分は /verify の担当 |
| AC10 `--force` | `TestOrgStop_Force_CloseFails_RecordsStoppedForced`、`TestOrgDisband_Force_RecordsPastCloseFailures`、CLI の `TestOrgStopAndDisband_Force_FailuresAreWarningsExitZero` | pass。M14・M36・M38・M45・M54・M71 が red |
| AC11 呼び出しごとの期限 | `TestOrgStop_UnansweredHerdr_EachCallTimesOut`、`_UnansweredLeave_`、`TestOrgStopDisband_UnansweredGet_LeavesItAlone`、`TestOrgStopAll_UnansweredHerdrForOneOrg_OthersStillStopped`、`TestOrgDisbandAll_UnansweredHerdrForOneOrg_OthersStillDisbanded` | pass。負荷をかけた `-count=10` でも 160 / 160。M10(期限なし)は 5 分の test timeout で red、M19 が red |
| AC12 自分の pane と workspace は最後 | CLI の `TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput`、org の `TestOrgDisband_OwnWorkspace_ClosedLastByCaller` ほか、追加した `TestOrgStopDisband_OwnRecordFails_NotHandedBack` | pass。M15・M16・M28〜M32・M42・M43・M48・M49・M55・M69・M70・M76・M77b が red。M20・M81 は HEAD で生き残り、追加したテストで red |
| AC13 disband のあとの spawn は新しい workspace | `TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace`、`TestOpenOrgWorkspaces` | pass。M35・M40・M41 が red |
| AC14 label が違えば閉じない | `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose`(追加後は 6 例 × `Force` の有無)、`TestOrgDisband_WorkspaceNotConfirmed_LeftOpen`、`TestOrgStopDisband_OwnIDNotConfirmed_NotDeferredNotClosed`、`TestOrgCloseDeferredSelf_RecheckedBeforeTheClose` | pass。M02〜M06・M08・M17・M22・M24・M33 が red。M07・M80 は HEAD で生き残り、追加した 2 例で red |
| AC15 `ErrWaitDelay` を成功にする | `TestExecRunner_Run_SuccessNotFailedByGrandchildHoldingPipes`、`_TimeoutNotHeldByGrandchild` | pass。M56・M58 が red。M57 は生き残り(下の Test gaps の 1) |

### 依頼にあった確認点

- `internal/org/verbs.go:704-705`(`confirmSeatPane` の `wsGone`): HEAD では通るテストがなかった。M07(この枝を「確認できた」に変える)は org と cli の全テストで生き残った。pane が見つかって tab の label も合うのに、pane の workspace を herdr が知らない場合に、C-c と close が送られる書き換えである。`TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` に例を足し、M07 が red になることを確かめた。
- `internal/org/verbs.go:1008`(`CloseDeferredSelfPane` の `no seat recorded on it`): HEAD で通っている。`TestOrgCloseDeferredSelfPane_Outcomes/no_seat_recorded_on_it:_left_open` が持ち、S7(05977322)で入った。M23(記録のない pane も閉じる)は red。self-review の「この枝を通るテストはない」は、この行については当たらない。
- verify V-1(古い台帳の判定の表に `--all` がない): M74・M75(`--all` の経路だけ判定を「読むだけ」にする)が生き残り、`internal/cli/org.go:742-744` と `:1012-1014` は coverage でも 0 回だった。表に `stop_all`、`stop_all_dry_run`、`disband_all` の 3 行を足し、M74・M75 が red になることを確かめた。
- verify V-3(leader の雛形の締めの順がテストで固定されていない): M78 が生き残った。テストを足して red にした。
- 依頼にあった mutation は次のとおりで、どれも red。持ち主の確認を外す(M01)、get の失敗を「閉じ済み」にする(pane の get は M02、workspace の get は M03。tab の get は M80 で、HEAD では生き残ったので例を足した)、座席が失敗しても disband が workspace を閉じる(M26)、`stop_failed` を書かない(`stopped` を書く M11、何も書かない M12)、自分の pane を先に閉じる(Stop の中ですぐ閉じる M15、確認の前に自分と判定する M16、disband と stop --all で最後に回さない M28・M42、CLI が出力の前に閉じる M69・M70)。

### Mutation

scratchpad に `git archive HEAD` で写しを作り、1 件ずつ書き換えて対象のパッケージのテストを流し、写しの元のファイルで戻した。worktree の本番のコードには触れていない。並行の 3 本で流したので、期限のテストで落ちた mutant(M03、M10、M19)はログで落ちた理由を見た。どれも負荷ではなく書き換えそのものが理由だった(M19 は「C-c の失敗で close を飛ばした」、M03 は「期限切れの get で disband が通った」、M10 は応答しない fake で 5 分止まった)。

`internal/org` の書き換えは org のテストを先に流し、生き残ったときだけ cli のテストを流した。cli のテストは時間を縮めるため `-run 'TestOrgStop|TestOrgDisband|TestOrgLegacyLedger|TestWithPrefixOnce|TestOrgCleanupVerbs'` で流し、HEAD で生き残った M07 と M20 は cli の全テスト(`-run` なし)でも生き残ることを確かめた。

| 対象 | red になった mutation | HEAD での生き残り |
| --- | --- | --- |
| 持ち主の確認(`confirmSeatPane`、`confirmOrgWorkspace`) | M01〜M06、M08、M09 | M07(pane の workspace が消えた枝)、M80(tab の get の失敗を「閉じ済み」にする) |
| 期限(`callWithTimeout`) | M10 | なし |
| `Stop` / `stopSeatPane` | M11〜M19、M21 | M20(`stopped` の記録に失敗しても自分の pane を返す) |
| 後回しの close(`CloseDeferredSelf*`) | M22〜M25 | なし |
| `Disband` / `closeOrgWorkspace` / `DisbandResult` | M26〜M39 | M79(確認の段で消えていた workspace を失敗にする)、M81(記録に失敗しても自分の workspace を返す) |
| `resolveWorkspace` / `openOrgWorkspaces`(AC13) | M40、M41 | なし |
| `StopAll` / `DisbandAll` / `orgsToDisband` | M42〜M55 | なし |
| driver(`ExecRunner`、herdr の close と get) | M56、M58〜M66 | M57(`ctx.Err() == nil` の条件を外す) |
| CLI | M67〜M73、M76、M77b | M74、M75(`--all` の経路の古い台帳の判定) |
| leader の雛形 | - | M78 |

- M77 は最初の書き方でビルドが通らなかった(使わない変数)。コンパイルが通る形に直して M77b として流した。件数はビルドが通った 81 件。
- HEAD で生き残った 9 件のうち M57 以外の 8 件は、f63ae025 のテストで red になった。追加のテストは既存のテストを変えずに例と関数を足しただけなので、HEAD で red だった 72 件が生き残りに戻ることはない。追加後に流し直したのはこの 8 件だけ。
- 作業の途中で私の手順の誤りが 2 件あった。1 件目は、zsh が `$ids` を単語に分けず、最初の起動で mutant が 1 件も流れなかったこと(`${=...}` で流し直した)。2 件目は、M74・M75 の再確認を追加前のテストの写しで流してしまったこと。この結果は捨て、追加後のテストの写しでの結果(red)を採った。

### 追加したテスト(f63ae025)

| テスト | 閉じた穴 | red / green の確認 |
| --- | --- | --- |
| `TestOrgStop_PaneNotConfirmed_NoCtrlCNoClose` に 2 例(`workspace gone although the pane was found`、`tab get fails`)× `Force` の有無 | `verbs.go:704-705`(`wsGone`)と `:694-695`(tab の get の失敗)。AC14 と AC11 の「確認できなければ閉じない」 | M07・M80 が red、HEAD のコードで pass |
| `TestOrgStopDisband_OwnRecordFails_NotHandedBack`(stop の自分の pane、disband の自分の workspace) | `Stop` の `selfPane && err == nil`(`:935`)と `closeOrgWorkspace` の記録の失敗(`:1651-1654`)。AC12 の「記録を済ませてから閉じる」。台帳を読み取り専用にして記録を失敗させる。root では skip | M20・M81 が red、HEAD のコードで pass |
| `TestOrgDisband_WorkspaceGoneAtCheck_CountsAsClosed` | `closeOrgWorkspace` の `case gone`(`:1627-1628`)。本物の herdr では、消えた workspace は close より前の `workspace get` で `workspace_not_found` になるので、こちらが「見つからなければ閉じ済み」の普段の経路になる。既存の `_WorkspaceNotFound_CountsAsClosed` は close の段の not-found だけを見ていた | M79 が red、HEAD のコードで pass |
| `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` に 3 行 | verify V-1 | M74・M75 が red |
| `TestRenderRolePrompt_Leader_ReportThenDisbandAsLastCommand` | verify V-3。`## ミッション` と `## 運用規律` のそれぞれで report が disband より前にあり、disband の項目に「最後のコマンド」があること | M78 が red |

### Coverage

- Statement(HEAD → テスト追加後): `internal/cli` 85.2% → 85.3%、`internal/org` 92.0% → 92.4%、`internal/org/driver` 93.1%(変わらず)。前回の snapshot(org-state-dir-common)はそれぞれ 84.8%・91.2%・92.0%。ほかの 5 パッケージは前回と同じ(`config` 92.3%、`insights` 86.1%、`org/protocol` 97.9%、`scaffold` 75.7%、`upgrade` 91.2%)
- Branch: Go の標準ツールでは取れない
- Function(テスト追加後、各パッケージ自身のテストでの値。括弧は HEAD の値が違うもの):
  - `internal/org/verbs.go`: `Stop` 95.8%、`stopSeatPane` 100%、`confirmSeatPane` 100%(90.9%)、`confirmOrgWorkspace` 100%、`callWithTimeout` 100%、`Disband` 93.3%、`disbandOwnLast` 100%、`closeOrgWorkspace` 100%(88.0%)、`recordWorkspaceClosed` 100%(66.7%)、`appendDisbanded` 50.0%、`CloseDeferredSelfPane` 93.8%、`CloseDeferredSelfWorkspace` 93.8%、`lastSeatOnPane` / `lastOrgOfWorkspace` 100%、`DisbandResult` の `recordStop` / `failSeat` / `failWorkspace` / `forcedNote` 100%
  - `internal/org/verbs_all.go`: `StopAll`、`DisbandAll`、`orgsToDisband`、`splitOwnOrgs`、`holdsCaller`、`add`、`leaveOwnOrg`、`failSeat` は 100%、`StopAllResult.recordStop` 87.5%(`ModelReceipts` に足す行は cli の `TestOrgStopAll_CLI_CodexModelMismatch_PrintsWarning` だけが通る)
  - `internal/org/spawn.go`: `resolveWorkspace` 87.5%、`openOrgWorkspaces` 90.9%
  - `internal/org/driver`: `ExecRunner.Run` 93.8%、`PaneClose` / `WorkspaceClose` / `checkHerdrCloseResult` / `wrapHerdrRunError` / `herdrGetResult` / `NotFound` / `IsNotFound` 100%、`PaneGet` / `TabGet` / `WorkspaceGet` 90.0%
  - `internal/cli/org.go`: `newOrgStopCmd` 96.8%(93.5%)、`newOrgDisbandCmd` 100%(95.7%)、`rejectFlagsWithAll` / `withPrefixOnce` / `printSeatFailure` / `printWorkspaceFailure` / `printStopAllResult` / `closeDeferredSelf` / `printDisbandResult` 100%、`printDisbandAllResult` 95.8%、`printOtherErrs` 75.0%
- Notes: org・driver・cli の 3 パッケージのテストを合わせても通らない行は、台帳の読み込みの失敗(`verbs.go:838`、`:1003`、`:1045`、`:1542`)、`stop_failed` と `disbanded` と `org_workspace_created` の追記の失敗(`:929`、`:1683`、`spawn.go:1716`)、herdr の get の result が JSON として読めない場合(`herdr.go:395`、`:418`、`:442`)、stderr のない非 0 終了(`driver.go:70`)、CLI の `printOtherErrs` が別のエラーを出す行(`org.go:832`)と、`disband --all` が台帳を読めない場合(`:1103`)、`requireSeatIdentifier` の失敗(`:752`)、`getenv` の `os.Getenv` 側(`verbs.go:642`。テストは必ず `o.Getenv` を差し替える)

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存の `stop` / `disband` のテストの意図(plan の Test plan が求める確認) | 弱めていない | 差分の test ファイルの削除は 21 行で、herdr の stub の 2 行(状態を持つ stub に置き換え)、fake のメソッドの引数名、assert の期待値の書き換えだけだった。`pane=ok` は `ctrl_c=ok` と `pane=closed` の 2 つに分かれ、「pane_id も agmsg_team もない座席に driver を呼ばない」は close の呼び出しも数えるようになった。`TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears` は seat-2 に別の pane を与えた(同じ pane だと fake が seat-1 の tab の label を書き換え、持ち主の確認が seat-1 を拒む) |
| spawn → disband → 同じ org_id で spawn(AC13) | pass | `TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace`。M35・M40・M41 が red |
| 古い台帳の判定(1 段目)が書き換えの動詞を止める | pass、`--all` の 3 つも固定した | `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` の 10 行 |
| watch の deadman と見張りのテスト(`fakeWatchHerdr` に get と close が増えた) | pass | `internal/org` 全体が ok |
| テストが main のチェックアウトの台帳に書かない | pass | 全実行のあと main と worktree に `.harness/state/org` はない |

### Test gaps

1. M57(`ExecRunner.Run` の `errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil` から `ctx.Err() == nil` を外す)が生き残った。この 2 つが同時に成り立つのは、コマンドが期限ちょうどに exit 0 で終わり、孫がパイプを握っている場合だけで、決まった結果を出すテストを書くのは難しい。外しても、期限を過ぎたのに途中の出力で成功になる、という差しか出ない。テストは足していない
2. 失敗の経路のうち、台帳の読み込みの失敗と、`stop_failed` / `disbanded` / `org_workspace_created` の追記の失敗(上の Notes の行)は、テストがない。どれも本番のコードはエラーを返す形になっている。足すなら、追加した `TestOrgStopDisband_OwnRecordFails_NotHandedBack` と同じく台帳を読み取り専用にする形で書ける
3. 本物の herdr は自動のテストで動かしていない。org は Go の fake、cli は sh の stub で、`pane close` / `workspace close` の stub は exit 0 で空の出力を返し、状態のファイルを消さない(close のあとに get しても見つかる)。本物の挙動は実装者の `docs/evidence/herdr-pane-close-2026-10-07.md` に頼っている
4. 別の herdr セッションから打った `stop` で本物の pane が `pane_not_found` と読まれる場合(self-review の F-1 の残余 (c))、`Disband` が台帳を読んだあとに別のプロセスが spawn した座席(F-4)、`--all` 全体の所要時間(F-3)は、テストがない。どれも self-review が tech-debt に送る予定のもの
5. `fakeWatchHerdr` は get をすべて not-found で返すので、watch のテストを通る `Stop` は C-c も close も送らない。watch の側で close の経路を見るテストはない(watch のテストが見る対象ではない)
6. spawn の補償の `compensateStale`(`spawn.go:1811`)が台帳の pane id に確認なしで C-c を送る既存の経路は、この PR の Non-goals で、テストもない

### Verdict(cycle 1)

- Verdict: pass
- Pass: `./scripts/run-test.sh` は 2 回とも rc 0(shell 2,047 / 2,047、Go 8 パッケージ ok)。`-count=1` の Go 全体、`-race`、新しいテストの `-count=3`、負荷をかけた期限のテストの `-count=10` もすべて pass。flake は出なかった。mutation は 81 件中、HEAD のテストで 72 件、テスト追加後に 80 件が red になった。生き残りは M57 の 1 件で、Test gaps の 1 に書いた
- Fail: なし
- Blocked: なし
