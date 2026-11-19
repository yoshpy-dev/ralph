# Test report: org-ssh-alias-host

- Date: 2026-11-20 JST(UTC では 2026-11-19)。cycle 2(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 2`)。cross-review の ACTION_REQUIRED #1 を直した 240324fe のあとの run。cycle 1 の報告は 63e1bc72 にある
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `9bd15b6f`(merge base `e6d5d0ed`、worktree は clean)。cycle 1 で試した HEAD `6a17dfad` から変わった Go のファイルは `internal/cli` の 4 つだけ: `doctor.go`、`doctor_merge_guarantee.go`、`doctor_merge_guarantee_signal_test.go`(新規)、`doctor_merge_guarantee_test.go`。`./scripts/run-test.sh` を worktree のルートから 1 回だけ走らせた。scope は wrapper の既定の changed で、log には `==> Language scope: changed (changed_languages)` と出た。そのあと、名前を挙げられたテストを `-v` で絞って打ち、240324fe のコードに変異を 11 個入れて落ちるかを見た。HEAD で作った binary に signal を送り、`ssh -G` が止まるかも確かめた
- Evidence: `docs/evidence/test-2026-11-19-org-ssh-alias-host.log`(5,619 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は同じ出力を `docs/evidence/verify-2026-11-19-170742.log` にも書いた。こちらは前者の先頭 5,617 行と同じ(前者の最後の 2 行がないだけ)なので消した

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-20 02:07:42 〜 02:32:06 JST、1,464 s、rc 0 |
| local verifier(`tests/test-*.sh` 45 本) | 45 本 | 45 本(`OK` 45 行) | 0(`FAIL:` で始まる集計の行は 28 行とも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 9.5 分(02:07:42 〜 02:17 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。7 つが `(cached)`(下の注) | 下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.715 s |
| `internal/cli` | ok | 777.506 s |
| `internal/config` | ok | 9.771 s |
| `internal/gitroot` | ok | (cached) |
| `internal/insights` | ok | (cached) |
| `internal/org` | ok | 913.336 s |
| `internal/org/driver` | ok | (cached) |
| `internal/org/protocol` | ok | (cached) |
| `internal/scaffold` | ok | (cached) |
| `internal/toolchain` | ok | (cached) |
| `internal/upgrade` | ok | (cached) |

`(cached)` の 7 パッケージについて: cycle 1 のあとに変わった Go のコードは `internal/cli` だけで、`go list -test -deps` で見ると 7 つとも `internal/cli` を import しない。full run のあとに `go test -count=1` で 7 つを打ち直し、すべて ok だった(16 s。gitroot 1.757 s、insights 0.805 s、driver 1.961 s、protocol 1.015 s、scaffold 1.782 s、toolchain 15.588 s、upgrade 2.914 s)。

実行の条件:

- 負荷(`sysctl -n vm.loadavg` を 30 s ごとに記録、50 点): 始まりは `8.96 13.73 18.87`。shell の段の 1 分値は 5.7〜10.9。Go の段の 1 分値は 7.1〜152.1 で、02:22:44 〜 02:25:45 の 42〜152 はほかのセッション(`swiftpm-testing-helper` と codex の app-server)による
- `GOFLAGS` は足していない。golang の pack が `-timeout=30m` を入れる。`internal/org` の 913.3 s は上限の 51%、`internal/cli` の 777.5 s は 43%
- 起動: cycle 1 と同じで、`run_in_background` は使わず、perl の fork と `setsid` で runner(`$TMPDIR` の下)を切り離した。INT、HUP、QUIT、TERM は DEFAULT に戻し、標準入出力は `/dev/null` にした。runner は始まりと終わりの時刻と rc を meta file に書く。終わるまでは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返し待った。shell の SKIP は上の 2 件だけで、signal を使う suite は skip していない
- 走らせる前に `pgrep` で見たとき、ほかの `run-test.sh`、`run-verify.sh`、`go test` はなかった。full run は 1 回だけで、その Go の段の間は、ファイルを読むことと変異のファイルを作ること以外は何も走らせていない
- 落ちたテストがないので、full run の打ち直しも、単独での打ち直しもしていない

この branch はまだ #348 と #349 を取り込んでいない(merge base `e6d5d0ed`、origin/main は `32274500`)ので、main にある 46 本目の suite `tests/test-run-verify-docs-only.sh` はこの run に入っていない。cycle 1 と違い、今の HEAD は origin/main と文書で衝突する(下の Test gaps の 5)。

## 名前を挙げられたテストの確認

full run の `go test` は `-v` なしなので、個々のテストの結果と skip は出ない。そこで full run のあとに、次の絞った run を打った。どれも full run が終わってから 1 本ずつ打った。

| Command | 結果 | 負荷(1 分値) |
| --- | --- | --- |
| `go test ./internal/cli -run 'MergeGuaranteeInterrupt\|MergeGuaranteeSignalContext' -count=3 -v`(verifier の挙げた形) | ok(6.2 s)。top-level 2 件が 3 回ずつ、subtest 6 件が 3 回ずつ PASS。FAIL と SKIP は 0 | 10.4 |
| 同じ `-run` を `-count=20`(`-v` なし) | ok(30.7 s) | 約 8 |
| `go test ./internal/cli -run 'TestDoctorNodeToolchain_Interrupt' -count=3 -v` | ok(29.2 s)。`TestDoctorNodeToolchain_InterruptStopsTheProbe`(subtest `SIGINT_to_the_process`、`the_context_ends`)と `TestDoctorNodeToolchain_InterruptFailsDoctor` が 3 回とも PASS | 約 8 |
| `go test ./internal/org/ -run '^(<下の 4 ファイルの 19 関数>)$' -count=1 -v` | ok(6.0 s)。top-level 19 件と subtest 82 件が PASS。FAIL と SKIP は 0 | 7.0 |
| `go test ./internal/org/ -run '^TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren$' -count=10 -v` | ok(35 s)。subtest 2 つとも 10 回中 10 回 PASS、SKIP 0 | 約 7 |
| `go test ./internal/cli/ -run '^(TestRunDoctor\|TestDoctor\|TestCheckMergeGuarantee_\|TestMergeGuarantee\|TestStatusCmd_MergeGuarantee)' -count=1 -v -coverprofile=…` | ok(32.2 s)。top-level 69 件と subtest 129 件が PASS。FAIL と SKIP は 0 | 3.7 |

`doctor_merge_guarantee_signal_test.go` の 6 つの subtest は、上の 1 行目と 2 行目で 23 回ずつ PASS した: `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` の `the_context_ends`、`the_context_ends_after_a_failed_check`、`SIGINT_to_the_process`(本物の SIGINT をテストのプロセスに送る場合)と、`TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP` の `interrupt`、`terminated`、`hangup`。

cycle 1 の 4 ファイルのテスト関数は、上の 4 行目の run で全部 PASS した。

| ファイル | テスト関数 |
| --- | --- |
| `internal/org/mergeorigin_test.go` | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`、`TestMergeSSHHostName`、`TestResolveMergeAPIHost_DefaultsAndInterruption`、`TestMergeSSHAliasHint`、`TestReadMergeOrigin_ResolvesAnSSHAlias`、`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(3.36 s。`the_ssh_-G_timeout` 3.00 s、`the_caller's_ctx` 0.36 s)、`TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault` |
| `internal/org/mergeguarantee_alias_test.go` | `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`、`TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`、`TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`、`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`、`TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown`、`TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan`、`TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn` |
| `internal/org/merge_ssh_alias_test.go` | `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`、`TestCheckMerge_ReadOriginRunsSSHThroughRunDetached`、`TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| `internal/org/regate_ci_ssh_alias_test.go` | `TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`、`TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec` |

`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` は、この cycle の `-v` の 11 回すべてで skip せずに走った。そのあと、テストの子の `sleep 30` は残っていない(`pgrep` で見つかった `sleep 30` 1 つは、親をたどるとほかのセッションの `run-ci-gated.sh` のものだった)。

### 本物の ssh がテストから走らないこと

呼ばれた引数を記録してから `/usr/bin/ssh` を exec する偽の `ssh` を `$TMPDIR` の下に置いて PATH の先頭に足し、上の表の最後の行と同じ `-run` を打った(ok、32.5 s)。記録は 0 行だった。同じ PATH で `ssh -G github.com` を手で打つと 1 行記録されたので、記録する仕組みは働いている。新しい signal のテストは、ssh を fake の `MergeOrderExec`(`params.Exec`)で受ける。

## Coverage

- Statement: 上の表の最後の行の coverage profile と `git diff -U0 240324fe^ 240324fe` の追加行を突き合わせた。`internal/cli/doctor.go` は文の block に入る追加行 16 行のすべて、`internal/cli/doctor_merge_guarantee.go` は 12 行のすべてを通った
- Branch: Go の標準の道具では測れない。代わりに下の変異で確かめた
- Function(同じ profile): `runMergeGuaranteeCheck` 100%、`checkMergeGuarantee` 100%、`printDoctorResults` 100%、`runDoctorStateDir` 95.3%
- Notes: full run は coverage を取っていない。cycle 1 で測った `internal/org` の追加行(105 行中 103 行)は、この cycle でコードが変わっていないので測り直していない

## 変異の確かめ(240324fe)

`go test -vet=off -overlay <mutant>.json ./internal/cli/ -run 'MergeGuaranteeInterrupt|MergeGuaranteeSignalContext|TestDoctorNodeToolchain_Interrupt' -count=1 -timeout 4m` で 1 つずつ打った。変異は `$TMPDIR` の写しにだけ入れ、worktree は書き換えていない(前後の `git status --porcelain --ignored` は同じ)。11 個とも落ち、build の失敗はない。

| # | 変異 | 落ちたテスト(落ちた理由) |
| --- | --- | --- |
| M01 | `runMergeGuaranteeCheck` が seam を通らず `context.WithCancel(ctx)` を使う | `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` の 3 つ(signal のあとに gh が 2 回走り、ssh の ctx は `context deadline exceeded` で終わった) |
| M02 | `signal.NotifyContext` から SIGHUP を外す | `TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP/hangup` |
| M03 | SIGTERM を外す | 同じテストの `terminated` |
| M04 | 検査の ctx を release しない(`defer stop()` を呼ばない形) | 3 つ(`took 2 signal contexts and released 0`) |
| M05 | `runMergeGuaranteeCheck` が interrupted を返さない | 3 つ(doctor が後の検査を続け、エラーが nil) |
| M06 | `context.Cause` でなく `checkCtx.Err()` を返す | 3 つ(エラーが `interrupted (context canceled)` になる) |
| M07 | 台帳を解けない道で interrupted に nil を返す | `the_context_ends_after_a_failed_check` |
| M08 | `runDoctorStateDir` が Merge guarantee の割り込みで止まらない | 3 つ |
| M09 | Node の割り込みを `printDoctorResults` に渡さない | `TestDoctorNodeToolchain_InterruptFailsDoctor` |
| M10 | `Interrupted:` の行がいつも Node toolchain の名前を出す(直す前の形) | 3 つ |
| M11 | 割り込みと失敗した検査がそろったときの分岐を飛ばす | `the_context_ends_after_a_failed_check` |

11 個のあとに、release する時点を動かす変異 M12 を足した。`runMergeGuaranteeCheck` は `stop` を呼ばずに package の変数にためておき、`printDoctorResults` の頭でまとめて呼ぶ。検査が終わってから doctor が終わるまでの間も signal の登録が残るので、その間の検査(`--probe-models` の Check 12 など)で受けた Ctrl-C は飲み込まれる。M12 は上の `-run`、doctor のテスト一式(上の表の最後の行、30.9 s)、`internal/cli` 全体(`-count=1`、519.1 s)のどれでも生き残った(Test gaps の 2)。

## HEAD の binary での signal の確かめ

cross-review の指摘 #1(doctor を signal で止めると、自分のプロセスグループで動く `ssh -G` とその子が残る)を binary で再現し、直ったことを確かめた。

- binary: HEAD を `go build -o <scratch>/ralph ./cmd/ralph` で作った。比べるために、直す前の 79eb08dd を `git archive` した写しからも作った
- repo: `git init` した scratch の repo に origin `git@myalias:acme/app.git` を足し、`.harness/state/org` を作った
- 環境: `HOME` は scratch、`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`SHELL=/bin/sh`。PATH は git の symlink と偽の `ssh` だけの dir。偽の `ssh` は自分の pid と引数を記録し、`/bin/sleep 300` を子に立てて `wait` する。`Match exec` の命令で待つ `ssh -G` の代わりになる
- 送り方: perl が signal を DEFAULT に戻して `ralph doctor` を fork し、偽の ssh の子が立ってから 0.3 s 後に ralph の pid へ signal を 1 つ送る。0.5 s 後に ssh と子が生きているかを `kill 0` で見た

| binary | signal | ralph の終わり方 | 0.5 s 後の ssh と `sleep 300` |
| --- | --- | --- | --- |
| HEAD | SIGINT | 0.03 s 後に exit 1 | どちらも終わっている |
| HEAD | SIGTERM | 0.02 s 後に exit 1 | どちらも終わっている |
| HEAD | SIGHUP | 0.03 s 後に exit 1 | どちらも終わっている |
| 79eb08dd(直す前) | SIGINT | 0.03 s 後に signal 2 で死ぬ。何も出力しない | どちらも生きている(あとで手でグループごと `kill -KILL` した) |

どの run でも、ssh のプロセスグループの id は ssh 自身の pid で、ralph のグループとは違った。HEAD の SIGINT の出力は、Merge guarantee の行が `ℹ Merge guarantee: info — unknown: the check was interrupted (interrupt signal received) while ssh -G myalias ran, so the protection of origin's repository's default branch is unknown; cached in <scratch>/repo/.harness/state/org/merge-guarantee.json` で、その後に Node toolchain の行はない。続いて `Interrupted: the Merge guarantee check did not finish. Run ralph doctor again.`、`Some checks failed. Fix the issues above.` が出て、エラーは `doctor: interrupted (interrupt signal received) while the Merge guarantee check read origin's repository; 2 check(s) failed`(fail の 2 件は、PATH に claude と codex がないため)。SIGTERM と SIGHUP では、括弧の中が `terminated signal received` と `hangup signal received` になる。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

full run でも、絞った run でも、落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review #1: doctor を SIGINT、SIGTERM、SIGHUP で止めると、自分のグループで動く `ssh -G` とその子が残る | 直っている | `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` と `TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP` が 23 回ずつ PASS。HEAD の binary では 3 つの signal とも ssh と子が終わり、直す前の binary では残る(上の節) |
| doctor の Node toolchain の割り込み(240324fe で `printDoctorResults` に移した) | 変わらない | `TestDoctorNodeToolchain_Interrupt*` が 3 回とも PASS。Node の割り込みを渡さない M09 で `TestDoctorNodeToolchain_InterruptFailsDoctor` が落ちる |
| issue #340: 別名の origin で doctor の Merge guarantee が unknown になる、merge が別名の host で gh を打つ | 直ったまま | cycle 1 の 4 ファイルの 19 関数が PASS。full run の `internal/org` と `internal/cli` が ok |
| 計画助言の指摘: 期限で ssh を止めても `Match exec` の子が残る | 直ったまま | `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` が 11 回とも skip せずに PASS |
| shell の suite(この branch は shell を変えていない) | 変わらない | 45 本とも `OK` |

## Test gaps

1. cycle 1 から残るもの(この cycle のコードでは変わらない): `mergeorigin.go:178-179`(`resolveMergeAPIHost` の `sshRun == nil` の既定)はテストからも本番の呼び手からも通らない(LOW)。本物の OpenSSH の `ssh -G` と本物の `Match exec` の命令は、自動のテストでは走らない。`ralph org cleanup` が ssh の runner に nil を渡すことを確かめる CLI のテストはない(LOW)
2. 検査の signal の ctx を、検査が戻るときに release することを確かめるテストがない。`TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` は seam を包んで、取った数と release した数が doctor が戻るまでにそろうことを見る。そのため、release しない M04 は落ちるが、release を doctor の終わりまで遅らせる M12 は `internal/cli` 全体で生き残った。M12 の形になると、検査のあとの Check 12 と Check 14 の間に受けた Ctrl-C を doctor が飲み込む。LOW。足すなら、包んだ `stop` が呼ばれた時点で、検査の後の検査がまだ走っていないこと(たとえば fake の exec の記録に後の検査の呼び出しがないこと)を確かめる行
3. binary での確かめは、ralph の pid にだけ signal を送った。端末の Ctrl-C(前面のプロセスグループに送られる)は試していない。ssh が別のグループにいるので ralph に届く経路は同じだが、本物の端末では確かめていない。偽の ssh は sh の script で、本物の OpenSSH ではない
4. full run の `go test` は `-v` なしなので、`run-test.sh` の中で `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` の期限切れの場合が skip したかは見えない。絞った run の 11 回は走った
5. 今の HEAD は origin/main と衝突する。`git merge-tree --write-tree HEAD origin/main` は rc 1 で、衝突は `docs/tech-debt/README.md` の 1 か所だけ: 800 行の目安を超えたファイルの行の終わりに、この branch の 5f2f0a0e(sync-docs)と main の 6a05139e(#341)がそれぞれ申し送りを書き足している。cycle 1 の時点(5f2f0a0e より前)は rc 0 だった。Go のファイルは重ならない(main が変えたのは `internal/cli/org_gate*.go`、`internal/config/config.go`、`internal/org/gate*.go`)。main を取り込んだ tree ではテストを走らせておらず、main の 46 本目の suite も入っていない。merge の前に、両方の申し送りを残す形で直す必要がある

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(1,464 s、shell の suite 45 本と Go の 11 パッケージ。`(cached)` の 7 つは `-count=1` で打ち直して ok)。`doctor_merge_guarantee_signal_test.go` の 6 つの subtest(本物の SIGINT の場合を含む)が 23 回ずつ PASS、`TestDoctorNodeToolchain_Interrupt*` が 3 回とも PASS、cycle 1 の 4 ファイルの 19 関数が PASS、本物の ssh の期限切れの場合は 11 回とも走って PASS。240324fe への変異 11 個はどれも落ちた(release を doctor の終わりまで遅らせる追加の M12 は生き残り、Test gaps の 2 に LOW として残した)。HEAD の binary は SIGINT、SIGTERM、SIGHUP のどれでも `ssh -G` とその子を止めて exit 1 で終わる
- Fail: 0
- Blocked: 0

insight event は、この報告とは別の commit で `docs/insights/events/` に足す。
