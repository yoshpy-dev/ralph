# Test report: org-ssh-alias-host

- Date: 2026-11-20 JST(UTC では 2026-11-19)。cycle 1(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `6a17dfad`(merge base `e6d5d0ed`、worktree は clean)。`./scripts/run-test.sh` を worktree のルートから 1 回だけ走らせた。scope は wrapper の既定の changed で、log には `==> Language scope: changed (changed_languages)` と出た。変更に入る言語は Go だけで、local verifier は scope によらず `tests/test-*.sh` を全部回す。full run のあと、名前を挙げられたテストを `-v` で絞って打ち直し、変更したコードに変異を入れて、テストが落ちるかを確かめた。HEAD の binary で doctor の実機確認もやり直した
- Evidence: `docs/evidence/test-2026-11-19-org-ssh-alias-host.log`(5,619 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-verify.sh は同じ出力を `docs/evidence/verify-2026-11-19-151433.log` にも書いた。中身は前者の先頭部分と同じ(前者の最後の 1 行 `Evidence saved to: …` がないだけ)なので、前者だけを残して後者は消した

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-20 00:14:33 〜 00:35:43 JST、1,270 s、rc 0 |
| local verifier(`tests/test-*.sh` 45 本) | 45 本 | 45 本(`OK` 45 行) | 0(`FAIL:` の行は 28 行とも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 8.5 分(00:14:33 〜 00:23 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` のパッケージもない | 下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.726 s |
| `internal/cli` | ok | 621.997 s |
| `internal/config` | ok | 12.017 s |
| `internal/gitroot` | ok | 4.767 s |
| `internal/insights` | ok | 1.956 s |
| `internal/org` | ok | 745.340 s |
| `internal/org/driver` | ok | 3.359 s |
| `internal/org/protocol` | ok | 1.555 s |
| `internal/scaffold` | ok | 3.491 s |
| `internal/toolchain` | ok | 19.178 s |
| `internal/upgrade` | ok | 4.583 s |

実行の条件:

- 負荷(`sysctl -n vm.loadavg` を 30 s ごとに記録): 始まりは `9.09 25.24 94.53`。shell の段の 1 分値は 4.0〜15.1。Go の段の 1 分値はほとんどが 7.8〜17 で、00:25 に 61.8、終わる直前の 00:35:34 に 124.4(ほかのセッション)
- `GOFLAGS` は足していない。golang の pack が `-timeout=30m` を入れる。`internal/org` の 745.3 s は上限の 41%、`internal/cli` の 622.0 s は 35%
- 起動: `run_in_background` は使わず、perl の fork と `setsid` で runner(`$TMPDIR` の下)を切り離した。INT、HUP、QUIT、TERM は DEFAULT に戻し、標準入出力は `/dev/null` にした。runner は始まりと終わりの時刻、rc を meta file に書く。終わるまでは `caffeinate -w <pid>` を 175 s ずつ区切って繰り返し待った。shell の SKIP は上の 2 件だけで、signal を使う suite は skip していない
- 走らせる前に `pgrep` で見たとき、ほかのセッションの絞った `go test ./internal/cli/ -run 'Approv|Inbox|Resolve|Send|DirectorNotice'` が 1 本動いていた。別の `run-test.sh` と `run-verify.sh` はなかった。full run は 1 回だけ
- 落ちたテストがないので、full run の打ち直しも、単独での打ち直しもしていない

この branch の merge base は `e6d5d0ed` で、origin/main(`32274500`)はそのあとに #348 と #349 を merge している。そのため、main にある 46 本目の suite `tests/test-run-verify-docs-only.sh` はこの run に入っていない。`git merge-tree --write-tree HEAD origin/main` は衝突なしで終わった(rc 0)。重なるファイルは文書だけで、README、org skill の 4 写し、`docs/tech-debt/README.md` の 6 つ。Go のコードは重ならない。

## 名前を挙げられたテストの確認

full run の `go test` は `-v` なしなので、個々のテストの結果と skip は出ない。そこで full run のあとに、次の絞った run を `-v` で打った。

| Command | 結果 | 負荷(1 分値) |
| --- | --- | --- |
| `go test ./internal/org/ -run '^(TestResolveMergeAPIHost_\|TestMergeSSH\|TestReadMergeOrigin_\|TestMergeGuarantee_\|TestCheckMerge_\|TestRegateCI_\|TestMergeOrigin_)' -count=1 -v -coverprofile=…` | ok(117.4 s)。top-level 88 件と subtest 404 件が PASS、FAIL と SKIP は 0 | 始まり 99.5、終わり 22.3 |
| `go test ./internal/cli/ -run '^(TestOrgMerge\|TestOrgCleanup\|TestOrgRegate\|TestDoctor\|TestRunDoctor)' -count=1 -v` | ok(259.5 s)。top-level 141 件と subtest 201 件が PASS、FAIL と SKIP は 0 | 始まり 22.3、終わり 132.9 |
| `go test ./internal/cli/ -run '^(TestCheckMergeGuarantee_\|TestMergeGuarantee\|TestStatusCmd_MergeGuarantee)' -count=1 -v` | ok(2.8 s)。`doctor_merge_guarantee_test.go` の 8 件と `status_merge_guarantee_test.go` の 8 件、subtest 30 件が PASS | 約 94 |
| `go test ./internal/org/ -run '^TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren$' -count=10 -v` | ok(34.3 s)。subtest 2 つとも 10 回中 10 回 PASS、SKIP 0 | 7〜9 |

指定のファイルのテスト関数は、どれも上の 1 行目の run で PASS した。

| ファイル | テスト関数(結果、時間) |
| --- | --- |
| `internal/org/mergeorigin_test.go` | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`(PASS、0.03 s)、`TestMergeSSHHostName`(PASS)、`TestResolveMergeAPIHost_DefaultsAndInterruption`(PASS)、`TestMergeSSHAliasHint`(PASS)、`TestReadMergeOrigin_ResolvesAnSSHAlias`(PASS)、`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(PASS、3.36 s)、`TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault`(PASS、0.36 s) |
| `internal/org/mergeguarantee_alias_test.go` | `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`、`TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`、`TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`、`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`、`TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown`、`TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan`、`TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn`(0.41 s)。7 件とも PASS |
| `internal/org/merge_ssh_alias_test.go` | `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`、`TestCheckMerge_ReadOriginRunsSSHThroughRunDetached`、`TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost`。3 件とも PASS |
| `internal/org/regate_ci_ssh_alias_test.go` | `TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`(0.87 s)、`TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec`(0.61 s)。2 件とも PASS |
| `internal/org/merge_test.go`(直した `TestMergeOrigin_*`) | `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`、`TestMergeOrigin_GetURLThatFailsTimesOutOrIsInterrupted`。2 件とも PASS |
| `internal/cli` の help と refusal(この branch が直したもの) | `TestOrgCleanup_Help`、`TestOrgMerge_Help`、`TestOrgRegate_Help`、`TestOrgCleanup_OriginRefusalForEachProblem`、`TestOrgCleanup_InterruptedBeforeItRunsRemovesNothing`(1.96 s)。5 件とも PASS(上の 2 行目の run)。doctor の `TestRunDoctorFull_PrintsTheMergeGuarantee` と `TestRunDoctor_TestMainPinRunsNoGitOrGh` も PASS |

### 本物の ssh の期限切れの場合

`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren/the_ssh_-G_timeout` は、`-v` で打った 11 回のすべてで走り、PASS した。skip は一度も出ていない。1 回目は上の 1 行目の run の中で、run を始めたときの 1 分値は 99.5。残りの 10 回は 1 分値 7〜9 のときに打った。full run の中の 1 回は `-v` なしなので、走ったか skip したかは log からはわからない(package は ok)。

もう一つの負荷に弱いテスト `TestRegateCI_AnInterruptOnTheMergePath`(前からある)は、full run で package ごと ok になり、上の 1 行目の run では 4 つの subtest とも PASS した(6.41 s)。

### 本物の ssh がテストから走らないこと

`$TMPDIR` の下に、呼ばれた引数を記録してから `/usr/bin/ssh` を exec する偽の `ssh` を置き、PATH の先頭に足して次の 2 つを打った。

- `go test ./internal/org/ -run '^(TestResolveMergeAPIHost_|TestMergeSSH|TestReadMergeOrigin_|TestMergeOrigin_|TestMergeGuarantee_|TestCheckMerge_|TestRegateCI_|TestMergeOrder)' -count=1`: ok(205.5 s)
- `go test ./internal/cli/ -run '^(TestOrgMerge|TestOrgCleanup|TestOrgRegate|TestDoctor|TestRunDoctor|TestCheckMergeGuarantee_|TestMergeGuarantee|TestStatusCmd)' -count=1`: ok(205.3 s)

記録は 0 行だった。記録する仕組みが働くことは、同じ PATH で `ssh -G github.com.emu` を手で打ち、1 行記録されることで確かめた。偽の ssh を自分で PATH に置くテスト(`soFakeSSH`)は、その偽の ssh のほうを先に見つけるので、記録には出ない。確かめたのはこの 2 つの絞った run だけで、2 つのパッケージの全部ではない。

## Coverage

- Statement: 上の 1 行目の run の coverage profile と `git diff -U0 e6d5d0ed HEAD` の追加行を突き合わせた。`internal/org` のテストでないファイルの追加行のうち、文の block に入る 105 行の中で 103 行を通った(98.1%)。通らない 2 行は `mergeorigin.go:178-179`(`resolveMergeAPIHost` の中の `sshRun == nil` の既定)。本番の呼び手は 2 つとも nil を渡さないので(`ReadMergeOrigin` は先に既定を入れ、doctor は `newMergeGuaranteeRun` が入れる)、本番からは通らない
- Branch: Go の標準の道具では測れない
- Function(同じ profile): `resolveMergeAPIHost` 96.2%、`ReadMergeOrigin` 95.0%(通らないのは前からある `run == nil` の既定で、絞った run が cleanup の CLI を含まないため)、`mergeSSHHostName`・`mergeSSHResolveRuns`・`mergeSSHHostNameWhy`・`mergeOriginIsSSH`・`mergeSSHAliasHint` はどれも 100%。doctor の `readOrigin`・`aliasNote`・`newMergeGuaranteeRun`・`readRepository` は 100%、`CheckMergeGuarantee` 92.3%。merge の `readOrigin`・`ghAccess`・`ghMissing`・`mergeReady.origin` は 100%
- Notes: `internal/cli` の変更は help の文、コメント、cleanup の `ReadMergeOrigin` の呼び出しだけで、行の coverage は測っていない。help の文は上の help のテストが確かめている

### 変異を入れた確かめ

変更したコードの 1 か所ずつを `go test -overlay` で書き換え、関係するテストに絞って `-vet=off -count=1` で打った。24 個のうち M08 は変数が使われなくなって build が通らなかったので、M08b に置き換えた。有効な 23 個はどれも落ちた(生き残り 0)。前後で `git status --porcelain --ignored` は変わらず、偽の ssh の子(`sleep 30`)も残っていない。

| ID | 書き換え | 落としたテスト(top-level) |
| --- | --- | --- |
| M01 | `ReadMergeOrigin` の ssh の既定を `realMergeOrderExec` に | `TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault` |
| M02 | 最初の `hostname` の行を採る | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`(the last hostname line wins) |
| M03 | 解いた値を小文字にしない | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`、`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` |
| M04 | 解いたあとの `ssh.github.com` の読み替えをやめる | 5 件(`TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot/ssh.github.com`、`TestMergeGuarantee_HostComesFromOriginsURL`、`TestCheckMerge_PassesForOtherOriginsAndLedgers` ほか) |
| M05 | doctor の ssh の既定を `realMergeOrderExec` に | `TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn` |
| M06 | merge が ssh に `m.runDetached` でなく nil を渡す | `TestCheckMerge_ReadOriginRunsSSHThroughRunDetached` ほか 3 件 |
| M07 | regate(ci)の ssh の既定を `realMergeOrderExec` に | `TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`、`TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec` |
| M08b | `mergeOriginIsSSH` が scheme を問わず true | 7 件(`TestReadMergeOrigin_ResolvesAnSSHAlias` の https の行ほか) |
| M09 | 案内の `apiHost == "github.com"` の条件を外す | `TestMergeSSHAliasHint`、`TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`、`TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| M10 | キャッシュの `host` に解いた host を書く | `TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead` ほか 4 件 |
| M11 | doctor が ssh -G のあとの ctx を確かめない | `TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown`、`TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan`、`TestMergeGuarantee_DeadlineExceededIsUnknown` |
| M12 | `ReadMergeOrigin` が ssh -G のあとの ctx を確かめない | `TestReadMergeOrigin_ResolvesAnSSHAlias`(interrupted while ssh -G runs) |
| M13 | 取り消しを期限切れと言う | `TestResolveMergeAPIHost_DefaultsAndInterruption`、`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` |
| M14 | `mergeSSHHostName` が `/` を通す | `TestMergeSSHHostName`、`TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes` |
| M15 | `mergeReady.sshAliasHint` が空を返す | `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| M16 | doctor の `readRepository` が案内を足さない | `TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`、`TestMergeGuarantee_FailuresAreUnknownWithAReason` |
| M17 | `ghAccess` が案内を足さない | `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| M18 | `ghMissing` が案内を足さない | `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| M19 | host の文字の検査をやめ、ssh の URL ならいつも ssh -G を打つ | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`、`TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan` ほか |
| M20 | `hostname` の値の `TrimSpace` をやめる | 13 件 |
| M21 | `ReadMergeOrigin` が ssh -G に get-url の期限を渡す | `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`、`TestReadMergeOrigin_ResolvesAnSSHAlias` |
| M22 | doctor が ssh -G を検査の ctx でなく親の ctx で打つ | `TestMergeGuarantee_OneDeadlineCoversEveryCall` |
| M23 | doctor が ssh -G に検査全体の期限を渡す | `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`、`TestMergeGuarantee_OneDeadlineCoversEveryCall` |

## HEAD での実機の確かめ(AC7 のやり直し)

`docs/evidence/ssh-alias-host-livefire-2026-11-19.md` の binary は `7798b966`(slice 2 のコード)で作ったもので、そのあと slice 5 と slice 6 でコードが変わった。そこで HEAD `6a17dfad` を `go build -o <scratch>/ralph ./cmd/ralph` で作り、同じ手順を打ち直した。`git init` した空の repo に origin `ssh://git@github.com.emu/yoshpy-dev/ralph.git` を足し、台帳は `--state-dir <scratch>/ledger` にした。

- `ralph doctor` の行: `✓ Merge guarantee: pass — github_protection: main, the default branch of origin's repository yoshpy-dev/ralph, requires its 2 required status checks to pass … (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads); cached in <scratch>/ledger/merge-guarantee.json`
- キャッシュ: `"kind": "github_protection"`、`"host": "github.com.emu"`、`"api_host": "github.com"`、`"repo": "yoshpy-dev/ralph"`
- `ralph status` の行: `merge guarantee: github_protection (main of github.com.emu/yoshpy-dev/ralph, checked 2026-11-19T15:45:47Z): …`。`origin_changed` にはならない
- PATH を git と gh の symlink だけの dir にした(ssh を外した)doctor の行: `ℹ Merge guarantee: info — unknown: gh api --hostname github.com.emu repos/yoshpy-dev/ralph failed (…): check that gh can reach github.com.emu and is logged in to it (gh auth status --hostname github.com.emu); ssh -G github.com.emu failed (exec: "ssh": executable file not found in $PATH), so gh read the host github.com.emu as origin names it: if github.com.emu is an ssh alias (a Host in ~/.ssh/config), point origin at the real host (git remote set-url origin <URL>) and choose the ssh key for this repository with git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'; cached in <scratch>/ledger2/merge-guarantee.json`

最初は台帳の dir を作らずに打ったので、行の最後が `not cached: there is no org ledger yet at <scratch>/ledger` になった(保護の読みは同じ `github_protection`)。台帳がないときの決まった挙動で、dir を作ってから打ち直したのが上の行。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

full run でも、絞った run でも、落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #340: 別名の origin で doctor の Merge guarantee が unknown になる | 直っている | `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` が PASS。HEAD の binary で doctor が `github_protection` を出した(上の実機の節) |
| 別名の origin で `ralph org merge` が別名の host で gh を打つ | 直っている | `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`、`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost` が PASS |
| 既存の origin の表(https、`ssh.github.com`、GHES、大文字小文字、`-` の host)の期待値 | 変わらない | `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`、`TestMergeGuarantee_HostComesFromOriginsURL` が PASS。M04 でこれらの `ssh.github.com` の行が落ちるので、行は読み替えを確かめている |
| 計画助言の指摘: 期限で ssh を止めても `Match exec` の子が残る | 直っている | `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` が 11 回とも skip せずに PASS。テストのあとに `sleep 30` は残っていない |
| shell の suite(この branch は shell を変えていない) | 変わらない | 45 本とも `OK` |

## Test gaps

- `mergeorigin.go:178-179`(`resolveMergeAPIHost` の `sshRun == nil` の既定)はテストから通らず、本番の呼び手からも通らない。doc comment は「nil means realMergeDetachedExec」と言っているが、壊れても気づけない。LOW。消すか、nil を渡す行を表のテストに 1 つ足す案がある
- 本物の OpenSSH の `ssh -G` と、本物の `Match exec` の命令は、自動のテストでは走らない。テストが使うのは fake の `MergeOrderExec` か、PATH に置いた sh の偽の ssh。本物の `ssh -G` は上の実機の確かめで走ったが、このマシンの `~/.ssh/config` の本体に `Match` の行はない(`Include` の先は見ていない)。`Match exec` の子がプロセスグループごと止まることは偽の ssh でしか確かめていない。命令が `setsid` で自分のグループを作ると残ることは、計画の Risks に書かれている
- `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` の期限切れの場合は、負荷が高いと skip しうる。`run-test.sh` の出力は `-v` なしなので、full run で skip したかどうかは見えない。この cycle では `-v` の 11 回とも走った
- `ralph org cleanup`(`internal/cli/org_cleanup.go`)は `ReadMergeOrigin` に ssh の runner として nil を渡す。nil のときに `realMergeDetachedExec` になることは `TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault` が確かめるが、cleanup の呼び出しそのものを確かめる CLI のテストはない。`internal/cli` からは unexported の runner を渡せないので、壊れ方は限られる。LOW
- テストした HEAD は origin/main より 2 つの merge(#348、#349)だけ古い。`git merge-tree` で衝突はなく、Go のコードも重ならないが、main を取り込んだ後の tree では走らせていない。main の 46 本目の suite `tests/test-run-verify-docs-only.sh` は、この run に入っていない

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(1,270 s、shell の suite 45 本と Go の 11 パッケージ)。名前を挙げられたテスト関数 21 件は `-v` の絞った run で PASS。本物の ssh の期限切れの場合は 11 回とも走って PASS(skip 0)。変異 23 個はどれも落ちた
- Fail: 0
- Blocked: 0

insight event は、依頼が「report を単独で commit」なので repo の `docs/insights/events/` には書かず、`--events-dir <scratch>` で作った。その行: `{"schema":1,"ts":"2026-11-19T15:53:47Z","slug":"org-ssh-alias-host","flow":"standard","phase":"test","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`
