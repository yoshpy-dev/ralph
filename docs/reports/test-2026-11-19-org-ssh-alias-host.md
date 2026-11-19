# Test report: org-ssh-alias-host

- Date: 2026-11-20 JST(UTC では 2026-11-19)。pipeline の 3 回目の run(`.harness/state/standard-pipeline/cycle-count.json` は `cycle: 2`、`cap: 3`)。cross-review の cycle 2 の ACTION_REQUIRED #1 を直した 664d4771 と 04161b43、run 3 の self-review の直し 74fef538 のあとに走らせた。前の報告は 530bf97c(cycle 2)と 63e1bc72(cycle 1)にある
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`(issue #340)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `34f2dc6d`(worktree は clean)。merge base は `32d6dc51` で、ローカルの `main` と `origin/main` も `32d6dc51`(fetch はしていない)。cycle 2 で試した HEAD `9bd15b6f` からの Go の変更は 2 つに分かれる。1 つは main の取り込み `e6a98fad`(#348〜#350)が持ち込んだ 19 ファイル(`internal/cli` 6、`internal/config` 1、`internal/org` 12)。もう 1 つはこの branch の直し 3 commit で、`internal/org` の 6 ファイル(`mergeorigin.go`、`mergeorder.go`、`mergeguarantee.go` とその test)と、`internal/cli/doctor_merge_guarantee_signal_test.go` のコメントを変えた。`./scripts/run-test.sh` を worktree のルートから 1 回だけ走らせた。scope は wrapper の既定の changed で、log には `==> Language scope: changed (changed_languages)` と出た。そのあと、名前を挙げられたテストと、直しで足した行を `-v` で絞って打った。full run の shell の段の間に、直しのコードへの変異 10 個を overlay で打った
- Evidence: `docs/evidence/test-2026-11-19-org-ssh-alias-host.log`(6,612 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。先頭の 5,829 行が full run の出力で、その後ろに runner の meta、負荷の記録、絞った run、変異、canary、changed-line coverage の出力を足した。run-verify.sh が同じ出力を `docs/evidence/verify-2026-11-19-190241.log` にも書いたが、前者の先頭 5,827 行と同じだったので消した

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(前からある条件つきの skip) | 2026-11-20 04:02:41 〜 04:29:33 JST、1,612 s、rc 0 |
| local verifier(`tests/test-*.sh` 46 本) | 46 本 | 46 本(`OK` 46 行) | 0(`FAIL:` で始まる集計の行は 29 行とも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 9.5 分(04:02:41 〜 04:12 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。4 つが `(cached)`(下の注) | 下の表 |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.747 s |
| `internal/cli` | ok | 907.581 s |
| `internal/config` | ok | 9.977 s |
| `internal/gitroot` | ok | (cached) |
| `internal/insights` | ok | 1.577 s |
| `internal/org` | ok | 1,055.948 s |
| `internal/org/driver` | ok | 2.727 s |
| `internal/org/protocol` | ok | (cached) |
| `internal/scaffold` | ok | 2.227 s |
| `internal/toolchain` | ok | (cached) |
| `internal/upgrade` | ok | (cached) |

`(cached)` の 4 パッケージについて: `go list -test -deps` で見ると、4 つとも `internal/org`、`internal/cli`、`internal/config` を import しない。full run のあとに `go test -count=1` で 4 つを打ち直し、すべて ok だった(15 s。gitroot 2.055 s、protocol 0.381 s、toolchain 14.840 s、upgrade 1.894 s)。

実行の条件:

- 負荷(`uptime` を 30 s ごとに記録、54 点): 始まりの 1 分値は 4.25。shell の段は 3.5〜16.0 で、16.0 は自分の変異の run(04:06〜04:09)と重なった時刻。Go の段は 5.2〜68.0 で、04:16 〜 04:26 の 21〜68 の出どころは確かめていない。Go の段の間、自分の側では run-test.sh のほかに何も走らせていない
- 空きディスクは 140〜143 GiB で、ENOSPC の心配はなかった
- `GOFLAGS` は足していない。golang の pack が `-timeout=30m` を入れる。`internal/org` の 1,055.9 s は上限の 59%、`internal/cli` の 907.6 s は 50%
- 起動: 前の cycle と同じく `run_in_background` は使わず、perl の fork と `setsid` で runner(`$TMPDIR` の下)を切り離した。INT、HUP、QUIT、TERM、PIPE は DEFAULT に戻し、標準入力は `/dev/null`、出力は scratch のファイルにした。runner は始まりと終わりの時刻と rc を meta file に書く。終わるまでは `caffeinate -w <pid>` を 168 s ずつ区切って繰り返し待った。shell の SKIP は上の 2 件だけで、signal を使う suite は skip していない
- 走らせる前に `ps` で見たとき、ほかの `run-test.sh`、`run-verify.sh`、`go test` はなかった。full run は 1 回だけ
- 落ちたテストがないので、full run の打ち直しも、単独での打ち直しもしていない

この run には、main の取り込みで入った 46 本目の suite `tests/test-run-verify-docs-only.sh` と、#348〜#350 の Go の変更が入っている。cycle 2 の報告の Test gaps 5 に書いた `docs/tech-debt/README.md` の衝突は `e6a98fad` で解かれ、今の `git merge-tree --write-tree HEAD origin/main` は rc 0。

## 名前を挙げられたテストの確認

full run の `go test` は `-v` なしなので、個々のテストの結果と skip は出ない。そこで full run が終わってから、次の絞った run を 1 本ずつ打った(`GOCACHE` は full run と同じ `.harness/cache/go-build`)。

| Command | 結果 | 負荷(1 分値) |
| --- | --- | --- |
| `go test ./internal/org/ -run '^(<下の 22 関数>)$' -count=1 -v` | ok(6.470 s)。top-level 22 件と subtest 114 件が PASS。FAIL と SKIP は 0 | 5.9 |
| `go test ./internal/org/ -run '^(TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren\|TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten)$' -count=10 -v` | ok(37.810 s)。2 関数と前者の subtest 2 つが、どれも 10 回中 10 回 PASS。SKIP 0 | 約 5 |
| `go test ./internal/cli -run 'MergeGuaranteeInterrupt\|MergeGuaranteeSignalContext' -count=3 -v`(依頼の形) | ok(5.000 s)。top-level 2 件と subtest 6 件が 3 回ずつ PASS。FAIL と SKIP は 0 | 4.9 |
| `go test -count=1` で `(cached)` の 4 パッケージ | ok(上の注) | 約 5 |

1 行目の 22 関数は、cycle 1 の 4 ファイルの 20 関数(cycle 1 の 19 関数と、664d4771 で足した `TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten`)と、664d4771 で期待値の変わった表を持つ 2 関数。

| ファイル | テスト関数 |
| --- | --- |
| `internal/org/mergeorigin_test.go` | `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes`、`TestMergeSSHHostName`、`TestResolveMergeAPIHost_DefaultsAndInterruption`、`TestMergeSSHAliasHint`、`TestReadMergeOrigin_ResolvesAnSSHAlias`、`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(3.37 s。`the_ssh_-G_timeout` 3.01 s、`the_caller's_ctx` 0.36 s)、`TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault`、`TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten`(新規、0.36 s) |
| `internal/org/mergeguarantee_alias_test.go` | `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`、`TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`、`TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`、`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`、`TestMergeGuarantee_InterruptedWhileSSHRunsIsUnknown`、`TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan`、`TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn` |
| `internal/org/merge_ssh_alias_test.go` | `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`、`TestCheckMerge_ReadOriginRunsSSHThroughRunDetached`、`TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost` |
| `internal/org/regate_ci_ssh_alias_test.go` | `TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`、`TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec` |
| `internal/org/mergeguarantee_test.go`、`internal/org/merge_test.go`(期待値の変わった表) | `TestMergeGuarantee_HostComesFromOriginsURL`、`TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot` |

`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` は、1 行目と 2 行目の 11 回すべてで skip せずに走った。そのあと、テストの子の `sleep 30` は残っていない(`pgrep -fl 'sleep 30$'` は何も返さなかった)。

cycle 2 の signal のテスト(`internal/cli/doctor_merge_guarantee_signal_test.go`)の 6 つの subtest は 3 回とも PASS した: `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor` の `the_context_ends`、`the_context_ends_after_a_failed_check`、`SIGINT_to_the_process`(本物の SIGINT をテストのプロセスに送る場合)と、`TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP` の `interrupt`、`terminated`、`hangup`。

### 664d4771、04161b43、74fef538 で足した行

上の 1 行目と 2 行目の run で、どれも PASS した。

| テスト(subtest) | origin | 確かめること |
| --- | --- | --- |
| `TestResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes/an_scp-like_host_in_capitals_is_passed_as_written` | `git@GitHub.COM:o/r.git` | `ssh -G GitHub.COM` を打ち、gh の host は `github.com` |
| 同 `an_ssh://_alias_in_capitals_is_passed_as_written` | `ssh://git@Work-GH/o/r.git` | `ssh -G Work-GH` |
| 同 `an_alias_in_capitals_with_a_hostname_in_capitals` | `git@Work-GH:o/r.git` | 解いた `GHE.Example.COM` は小文字の `ghe.example.com` になる |
| 同 `an_alias_in_capitals_whose_ssh_-G_fails_falls_back_lowercased` | `git@Work-GH:o/r.git` | fallback は `work-gh`、理由は `ssh -G Work-GH failed (exit status 255)` |
| 同 `an_alias_in_capitals_with_a_port_and_no_hostname_line` | `ssh://git@Work-GH:22/o/r.git` | 理由は `ssh -G Work-GH printed no hostname line` |
| 同 `SSH.GitHub.com_whose_ssh_-G_fails_is_github.com` | `git@SSH.GitHub.com:o/r.git` | fallback は `github.com`、理由は書かれたとおりの host を名指す |
| 同 `a_host_in_capitals_that_is_no_plain_host_name_is_quoted_as_written` | `git@Work;GH:o/r.git` | ssh は走らず、理由は `"Work;GH"` を引用する |
| 同 `a_host_whose_lowercase_is_ASCII_runs_no_ssh` | U+212A(Kelvin)の `K.example.com` | 小文字にすると ASCII になるが、書かれたとおりの host で確かめるので ssh は走らない |
| 同 `a_host_whose_lowercase_is_not_EqualFold_with_it_runs_no_ssh`(74fef538) | U+0130 の `İ.example.com` | 小文字の `i.example.com` で ssh は走らない(self-review の run 3 の L13) |
| `TestReadMergeOrigin_ResolvesAnSSHAlias/a_host_in_capitals_reaches_ssh_-G_as_written` | `git@GitHub.COM:o/r.git` | `Arg` と `originHost` は小文字、ssh の引数は `GitHub.COM` |
| 同 `an_alias_in_capitals_whose_ssh_-G_fails` | `git@Work-GH:o/r.git` | `Arg` は `work-gh/o/r`、理由は `ssh -G Work-GH` を名指す |
| `TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten` | `git@Work-GH:o/r.git` | PATH の偽の `ssh` を `realMergeDetachedExec` で打ち、偽の ssh が受けた引数は `-G Work-GH`(11 回とも PASS) |
| `TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves/ssh://_alias_in_capitals_with_a_port` | `ssh://git@GitHub.com.EMU:22/acme/app.git` | doctor の検査が `ssh -G GitHub.com.EMU` を打ち、保護を読む |
| 同 `scp-like_alias_in_capitals` | `git@GitHub.Com.Emu:acme/app.git` | 同上 |
| `TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan/an_alias_in_capitals`(04161b43) | `git@GitHub.com.EMU:acme/app.git` | 中断の文は `while ssh -G GitHub.com.EMU ran` |
| 同 `a_host_whose_lowercase_is_ASCII` | U+212A の host | ssh が走らないので、中断の文は ssh -G を名指さない |
| `TestMergeGuarantee_HostComesFromOriginsURL/git@SSH.GitHub.com:acme/app.git`(期待値を変えた行) | `git@SSH.GitHub.com:acme/app.git` | ssh の引数は `SSH.GitHub.com`、キャッシュの host は `ssh.github.com` |
| `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot/GitHub_Enterprise_in_other_case`(期待値を変えた行) | `git@GHE.example.com:Team/App.git` | ssh の引数は `GHE.example.com`、`Arg` は `ghe.example.com/Team/App` |

### 本物の ssh がテストから走らないこと

引数を記録してから `/usr/bin/ssh` を exec する偽の `ssh` を `$TMPDIR` の下に置いて PATH の先頭に足し、上の 1 行目と同じ `-run` の `internal/org`(ok、6.432 s)と、doctor のテスト一式 `go test ./internal/cli/ -run '^(TestRunDoctor|TestDoctor|TestCheckMergeGuarantee_|TestMergeGuarantee|TestStatusCmd_MergeGuarantee)'`(ok、32.883 s)を打った。記録は 0 行だった。同じ PATH で `ssh -G github.com` を手で打つと 1 行記録されたので、記録する仕組みは働いている。`TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten` は自分の偽の ssh を PATH の先頭に置くので、本物には届かない。

## 変異の確かめ(664d4771、04161b43、74fef538)

`go test -vet=off -overlay <id>.json ./internal/org/ -run 'TestResolveMergeAPIHost|TestReadMergeOrigin|TestMergeGuarantee|TestCheckMerge|TestRegateCI_ResolvesAnSSH|TestRegateCI_RunsSSH|TestMergeSSH|TestMergeOrigin' -count=1 -timeout 4m` で 1 つずつ打った(1 つ 8〜14 s。変異のない対照は ok、7 s)。最初は `TestRegateCI` を全部入れたが、対照だけで 135 s かかったので止め、ssh の別名の 2 関数に絞った。変異は `$TMPDIR` の写しにだけ入れ、worktree は書き換えていない(前後の `git status --porcelain --ignored` は、run-verify.sh が書いた log を除いて同じ)。10 個とも落ち、build の失敗も ENOSPC もない。

| # | 変異 | 落ちたテスト |
| --- | --- | --- |
| K1 | `mergeSSHHost` が小文字の host を返す(664d4771 の中心を戻す) | `--- FAIL` の行が 24(top-level を含む、以下同じ)。上の表の大文字と U+212A・U+0130 の行のほぼすべてと `TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten` |
| K2 | ssh -G の引数だけ小文字の host に戻す(文は書かれたとおりのまま) | 引数を見る行: `an_scp-like_host_in_capitals…`、`an_ssh://_alias_in_capitals…`、`TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten`、doctor の大文字の 2 行、`HostComesFromOriginsURL/git@SSH.GitHub.com…`、`GitHub_Enterprise_in_other_case` など(`--- FAIL` の行が 17) |
| K3 | 理由の文だけ小文字の host で ssh -G を名指す(引数はそのまま) | `…ssh_-G_fails_falls_back_lowercased`、`…port_and_no_hostname_line`、`SSH.GitHub.com_whose_ssh_-G_fails…`、`TestReadMergeOrigin_ResolvesAnSSHAlias/an_alias_in_capitals_whose_ssh_-G_fails` |
| K4 | 「ssh -G was not run」の文が小文字の host を引用する | `…quoted_as_written`(Work;GH)、U+212A と U+0130 の行 |
| K5 | `mergeSSHResolveRuns` が小文字の host を確かめる(664d4771 の前の形) | U+212A と U+0130 の行、`ReadOriginNamesSSHAsWhatRanOnlyWhenItRan/a_host_whose_lowercase_is_ASCII` |
| K6 | 74fef538 を戻す(`!strings.EqualFold(written, host)` なら小文字の host) | U+0130 の行 `a_host_whose_lowercase_is_not_EqualFold_with_it_runs_no_ssh` だけ |
| K7 | 04161b43 を戻す(doctor の中断の文が小文字の host) | `TestMergeGuarantee_ReadOriginNamesSSHAsWhatRanOnlyWhenItRan/an_alias_in_capitals` だけ |
| K8 | fallback の host が書かれたとおりの大文字を残す | fallback を見る 6 行(Work;GH、U+212A、U+0130、ssh -G の失敗、hostname なし、SSH.GitHub.com の失敗)と `ReadMergeOrigin` の失敗の行 |
| K9 | `mergeOriginFromURL` が小文字にしない(キャッシュの host と `originHost` が大文字のまま) | `--- FAIL` の行が 19。`HostComesFromOriginsURL` の https と GHE の行、`TestMergeGuarantee_ReadsTheHostWithTheOwnerAndRepo` など |
| K10 | `ssh -G` が出した hostname を小文字にしない | `a_hostname_in_capitals_is_lowercased`、`an_alias_in_capitals_with_a_hostname_in_capitals`、doctor の `hostname_in_capitals`、`TestCheckMerge_PassesForOtherOriginsAndLedgers/GitHub_Enterprise_origin` など(`--- FAIL` の行が 14) |

K6 と K7 は、それぞれの直しが足した 1 行だけで落ちる。その行が直しを固定しているとわかる一方で、固定しているのはその 1 行だけでもある。

`mergeSSHHost` の `if !ok || written == ""` の枝には変異を入れていない。呼び手はどれも、`mergeOriginFromURL` が読めた URL とその host を渡すので `ok` は真になる。`mergeOriginFromURLAsWritten` は scp 形でも URL 形でも空の host を断るので、`written` も空にならない。今の呼び手から見ると、この枝を消す変異は振る舞いを変えない(Test gaps の 3)。

## Coverage

- Statement(changed lines): 上の変異と同じ `-run` に `-coverprofile` を付けた run(7.6 s)の profile と、`git diff -U0 ebbbfd1c HEAD` の `mergeorigin.go`、`mergeorder.go`、`mergeguarantee.go` の追加行を突き合わせた。文の block に入る追加行 16 行のうち 14 行を通った。通らない 2 行は `mergeorigin.go:248-249`(上の段落の枝)
- Function(同じ profile): `mergeOriginFromURL` 100%、`mergeOriginFromURLAsWritten` 95.2%、`mergeSSHResolveRuns` 100%、`mergeSSHHost` 75.0%、`resolveMergeAPIHost` 96.3%(`sshRun == nil` の既定 `mergeorigin.go:188` を通らない)、`ReadMergeOrigin` 95.0%、`mergeSSHAliasHint` 100%
- Branch: Go の標準の道具では測れない。代わりに上の変異で確かめた
- Notes: full run は coverage を取っていない。`internal/cli/doctor.go` と `doctor_merge_guarantee.go` は cycle 2 から変わっていないので、cycle 2 の追加行の測定(28 行すべて)は測り直していない

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

full run でも、絞った run でも、落ちたテストはない。

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cross-review の cycle 2 の #1: `ssh -G` に小文字にした host を渡し、大文字の別名(`Work-GH`)を解けない、または git と違う `Host` を読む | 直っている | 上の表の大文字の行がすべて PASS。`TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten` で、偽の ssh が `-G Work-GH` を受けた。直しを戻す K1 と K2 が落ちる |
| doctor の中断の文が小文字の host で ssh -G を名指す(04161b43) | 直っている | `ReadOriginNamesSSHAsWhatRanOnlyWhenItRan/an_alias_in_capitals` が PASS、K7 で落ちる |
| self-review の run 3 の L13: U+0130 の host で、origin が名指さない ASCII の host に ssh -G を打つ | 直っている | U+0130 の行が PASS、74fef538 を戻す K6 で落ちる |
| cross-review の cycle 1 の #1: doctor を signal で止めると `ssh -G` とその子が残る | 直ったまま | signal のテストの 6 subtest が 3 回とも PASS。`internal/cli/doctor.go` と `doctor_merge_guarantee.go` は cycle 2 から変わっていない |
| issue #340: 別名の origin で doctor の Merge guarantee が unknown になる、merge が別名の host で gh を打つ | 直ったまま | cycle 1 の 4 ファイルの 20 関数が PASS。full run の `internal/org` と `internal/cli` が ok |
| 計画助言の指摘: 期限で ssh を止めても `Match exec` の子が残る | 直ったまま | `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` が 11 回とも skip せずに PASS |
| キャッシュの `host`、`MergeOrigin.originHost`、fallback、解いた hostname は小文字のまま | 変わらない | `HostComesFromOriginsURL` と `TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot` が PASS。小文字をやめる K8、K9、K10 がどれも落ちる |
| main の取り込み(`e6a98fad`、#348〜#350) | 通る | 46 本の shell の suite(main の `tests/test-run-verify-docs-only.sh` を含む)と Go の 11 パッケージが ok |

## Test gaps

1. cycle 1 から残るもの(この run の直しでは変わらない): `resolveMergeAPIHost` の `sshRun == nil` の既定(`mergeorigin.go:187-188`)はテストからも本番の呼び手からも通らない(LOW)。本物の OpenSSH の `ssh -G` と本物の `Match exec` の命令は、自動のテストでは走らない。`ralph org cleanup` が ssh の runner に nil を渡すことを確かめる CLI のテストはない(LOW)
2. cycle 2 から残るもの: 検査の signal の ctx を、検査が戻るときに release することを確かめるテストがない。release を doctor の終わりまで遅らせる変異 M12 は cycle 2 で `internal/cli` 全体を生き残った。doctor のコードは cycle 2 から変わっていないので、この run では打ち直していない(LOW)。binary での signal の確かめも cycle 2 のもので、端末の Ctrl-C は試していない
3. `mergeSSHHost` の `if !ok || written == ""` の枝(`mergeorigin.go:248-249`)は、今の呼び手からは通らない(上の「変異の確かめ」の最後の段落)。changed-line coverage の 16 行中 2 行の未通過はこの枝。呼び手が origin の URL と別の host を渡すようになったときにだけ意味を持つ防御で、それを確かめるテストはない(LOW)
4. 大文字の host で ssh -G が失敗したときの案内の文(`mergeSSHAliasHint`)と doctor の別名の注記は、小文字の host を出す(self-review の L14、tech-debt に回す予定)。この大文字と小文字の組を固定するテストはない。表示だけの差で、直しのコードの動きは上の行が固定している
5. full run の `go test` は `-v` なしなので、`run-test.sh` の中で `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` の期限切れの場合が skip したかは見えない。絞った run の 11 回はすべて走った

## Verdict

- Pass: `./scripts/run-test.sh` rc 0(1,612 s、shell の suite 46 本と Go の 11 パッケージ。`(cached)` の 4 つは `-count=1` で打ち直して ok)。cycle 1 の 4 ファイルの 20 関数と期待値の変わった 2 関数が PASS(subtest 114 件)、664d4771・04161b43・74fef538 で足した大文字・`Work-GH`・U+212A・U+0130 の行がすべて PASS、本物の ssh を使う 2 関数は 10 回ずつ走って PASS(skip 0)、cycle 2 の signal のテストは 3 回とも PASS。直しのコードへの変異 10 個はどれも落ちた
- Fail: 0
- Blocked: 0

insight event は、この報告とは別の commit で `docs/insights/events/` に足す。`--cycle auto` は、この 3 回目の run の event に cycle 2 を付ける(run 3 の self-review と verify の event と同じ)。
