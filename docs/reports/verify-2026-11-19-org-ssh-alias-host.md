# Verify report: org-ssh-alias-host

- Date: 2026-11-20(パイプライン cycle 2。`cycle-count.json` は `cycle: 2`。cycle 1 の verify は 2 回走り、1 回目 `bc221a8d` が fail、2 回目 `b01d04c1` が pass)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`。`./scripts/plan-visual.sh digest` の値 `f4a9803ca80e` は、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `e6d5d0ed`、HEAD `e41aeeca`、40 ファイル、+2877 −216)。前回の verify(`b01d04c1`)のあとの commit は 10 個で、Go を変えたのは cross-review の ACTION_REQUIRED #1 を直した `240324fe` だけ(`git diff --name-only b01d04c1 HEAD -- '*.go'` は `internal/cli/doctor.go`、`doctor_merge_guarantee.go`、`doctor_merge_guarantee_signal_test.go`、`doctor_merge_guarantee_test.go` の 4 つ。`templates/` は 0 件)。ほかは計画、report、event、sync-docs の文書(`5f2f0a0e`)。手元の `main` と `origin/main` は `32274500`(#348、#349)に進んでいる。依頼に従いマージはしていない(試しのマージは Static analysis)。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-org-ssh-alias-host.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。cycle 1 の内容のあとに cycle 2 の節を足した(足す前のファイルが新しいファイルの先頭のバイト列と一致することを `head -c | cmp` で確かめた)。静的検査の出力、試しのマージ、行数、`ssh -G` の呼び手、binary の probe の出力が入っている。ラッパーは自分の記録として `docs/evidence/verify-2026-11-19-165628.log` も書いた(これも gitignore に当たる)

## Spec compliance

`internal/org` は前回の verify から変わっていないので、下の表の `internal/org` の行の引用は前回と同じ行を指す。`240324fe` が変えたのは doctor の CLI の層で、その分を AC5b と最後の行に足した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 別名の origin で doctor の Merge guarantee が gh を `--hostname github.com` で打ち、種類を決める | Met | doctor の `readOrigin`(`internal/org/mergeguarantee.go:297`)は get-url のあと `resolveMergeAPIHost`(`internal/org/mergeorigin.go:169`、呼び出しは `mergeguarantee.go:311`)で `apiHost` を決める。`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`(`internal/org/mergeguarantee_alias_test.go:49`)が 5 つの場合を確かめる。今回、HEAD `e41aeeca` の binary で、このマシンの `Host github.com.emu` を origin にした repo の doctor を打ち直し、`✓ Merge guarantee: pass — github_protection: ...(origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)` を見た |
| AC2: merge が使う host(`ReadMergeOrigin` の `-R`、`CheckMergeEnforcement` の `--hostname`)と doctor の host が一致する | Met | 解く関数は 1 つで、呼び手は `ReadMergeOrigin`(`mergeorigin.go:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。merge は `r.apiHost` を `CheckMergeEnforcement` に渡す(`internal/org/merge.go:361-362`)。`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`(`mergeguarantee_alias_test.go:250`)と `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`(`internal/org/merge_ssh_alias_test.go:23`)。`240324fe` は doctor の検査を走らせる ctx だけを変え、host の決め方は変えていない |
| AC3: 別名でない origin の gh の host は変わらない。https と `git://` では ssh を打たない | Met | `resolveMergeAPIHost` は `mergeOriginIsSSH` が偽なら何も打たずに origin の host を返す(`mergeorigin.go:171-173`)。表のテストは前回と同じ(`internal/org/mergeorigin_test.go`、`internal/org/mergeguarantee_test.go`、`internal/org/merge_test.go`)。`internal/org` は不変 |
| AC4: キャッシュの `host` は origin に書かれた host、`api_host` は gh に渡した host。`ReadMergeGuarantee` は別名の origin で `origin_changed` にならない | Met | `TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`(`mergeguarantee_alias_test.go:182`)。今回の HEAD の binary の doctor が書いたキャッシュは `{"kind":"github_protection","host":"github.com.emu","api_host":"github.com","repo":"yoshpy-dev/ralph","api_repo":"yoshpy-dev/ralph","default_branch":"main"}` だった |
| AC5: `ssh -G` の失敗・期限切れ・hostname なし・不正な値で origin の host のまま進み、gh が失敗したら doctor と merge の案内が理由と `git config core.sshCommand` の手を言う | Met | 理由の文は `resolveMergeAPIHost` の分岐(`mergeorigin.go:174-209`)、案内は `mergeSSHAliasHint`(`:275`)。テストは `TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`(`mergeguarantee_alias_test.go:107`)と `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost`(`merge_ssh_alias_test.go:77`)。今回の probe では、修正前の binary(`240324fe^` = `79eb08dd`)が SIGINT を受けずに 5 秒の期限まで進み、`ssh -G github.com.emu timed out after 5s, so gh read the host github.com.emu as origin names it: ... git config core.sshCommand ...` の Detail を出した(Observational checks) |
| AC5b: 本物の ssh は `realMergeDetachedExec` で打ち、専用の期限と親の ctx の取り消しの両方で戻り、子プロセスも終わる。本番の 4 つの経路が `realMergeDetachedExec` を渡す | Met | org の層のテストは前回と同じ(`TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` は `mergeorigin_test.go:410`、`TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn` は `mergeguarantee_alias_test.go:376`、`TestMerge_CallsAfterGHPRMergeRunDetached` は `internal/org/merge_run_test.go:764`、`TestRegateCI_ResolvesAnSSHAliasOriginThroughExec` は `internal/org/regate_ci_ssh_alias_test.go:27`、cleanup の nil の既定は `TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault` が `mergeorigin_test.go:462`)。`14597e36` の条件つきの `t.Skipf` は計画の Progress(151〜152 行)に書かれた。`240324fe` で、doctor でも「親の ctx の取り消し」が起こりうるようになった。それまで doctor は `context.Background()` で検査を呼び、signal で ctx が終わる道がなかった。今は `runMergeGuaranteeCheck`(`internal/cli/doctor_merge_guarantee.go:111-119`)が `doctorMergeGuaranteeSignalContext`(`:66-68`、`signal.NotifyContext` に SIGINT・SIGTERM・SIGHUP)の ctx で検査を走らせる。テストは `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor`(`internal/cli/doctor_merge_guarantee_signal_test.go:101`、seam、台帳の解けない run、本物の SIGINT の 3 通り)と `TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP`(`:240`)。HEAD の binary の probe(PATH に置いた偽の `ssh` が子を立てて待つ)では、SIGINT でも SIGTERM でも doctor が 0.04 秒で終わり、ssh とその子の両方が消えた。修正前の binary に SIGTERM を送ると doctor だけが死に、ssh とその子は残った |
| AC6: 別名の origin で `https://github.com/<o>/<r>/pull/<n>` が origin の PR と認められる | Met | `TestReadMergeOrigin_ResolvesAnSSHAlias`(`mergeorigin_test.go:309`)。`internal/org` は不変 |
| AC7: 実機で、別名の origin の doctor が `--hostname github.com` で読み、unknown にならない | Met | `docs/evidence/ssh-alias-host-livefire-2026-11-19.md`(binary `7798b966`)。cycle 1 の 1 回目の verify は HEAD `0a71a930` の binary で、test は HEAD `6a17dfad` の binary で同じ場面を打ち直していた。`240324fe` が本番の Go を変えたので、今回は HEAD `e41aeeca` の binary で打ち直した。別名の origin で `github_protection`、キャッシュの `host` は別名、`api_host` は `github.com`、doctor は `All checks passed.` で rc 0。AC は「一時 clone」と言うが、evidence も今回も `git init` と `git remote add origin` で作った repo を使った(doctor が読むのは origin の URL と GitHub 側なので判定は変わらない) |
| AC8: 文書が新しい挙動と合う。`check-skill-sync.sh` と `check-sync.sh` が通る | Met(注つき) | 2 つの検査は通った(Static analysis)。cycle 1 の D1(README の doctor の行の括弧)は `5f2f0a0e` で直り、今の文は `mergeSSHHostName` の規則と合う。org skill の 4 写し、spec の 7c-2、register の RESOLVED の行は変わらず合う。AC8 が名指す文書に誤りはない。ただし `240324fe` の挙動(signal で doctor が止まる)を言う文書はない(D6)。register の 419 行の注は `240324fe` のあとの数と合わない(D7)。どちらも AC8 が名指す行の外で、`/sync-docs` に渡す |
| 追加(cross-review の ACTION_REQUIRED #1、計画からのずれとして Progress の 155 行に記録): `ssh -G` を打つ道はすべて、signal で終わる ctx の下で走る | Met | 非 test で `ssh` を実行するのは `mergeorigin.go:186` の 1 か所で、`resolveMergeAPIHost` の中にある。その呼び手は `ReadMergeOrigin`(`:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`、`g.ctx` は `CheckMergeGuarantee` が呼び手の ctx から作る 20 秒の ctx)。`ReadMergeOrigin` の呼び手は 3 つで、`merge.go:429`(`ralph org merge`、`internal/cli/org_merge.go:274` の `signal.NotifyContext`)、`regate_ci.go:94`(`ralph org regate`、`internal/cli/org_regate.go:254`)、`internal/cli/org_cleanup.go:206`(`ralph org cleanup`、`:168`)。`org.CheckMergeGuarantee` の呼び手は doctor の `runMergeGuaranteeCheck`(`doctor_merge_guarantee.go:114`)だけで、新しい seam の ctx の下にある。`context.WithoutCancel` を使う所(`merge_run.go:393`、`:574`、`cleanup.go:722`、`regate_merge.go`、`regate_ci.go:348`)は gh、manifest の lock、git で、どれも `ReadMergeOrigin` に届かない。`ralph status` と `ralph org merge-order` は ssh を打たない |

計画の Verify plan が挙げる「gh の解き方と同じ順序」(解く、小文字にする、`mergeGuaranteeAPIHost`)は、cycle 1 の 1 回目に go-gh の `pkg/ssh/ssh.go` と照らした。`mergeorigin.go` はそのあと変わっていないので、その判断はそのまま当てはまる。

### `240324fe` の読み

- 検査の中だけで signal を捕まえる。`runMergeGuaranteeCheck` の `defer stop()` が検査の返りで signal の扱いを元に戻し、`SaveMergeGuarantee` と後の検査は handler の外で走る
- 割り込まれたら、Merge guarantee の行(`unknown: the check was interrupted (<原因>) while ssh -G <host> ran, ...`)とそれまでの行を出し、Check 12(`--probe-models`)、Check 13(scaffold の検査)、Check 14(Node toolchain)を打たずに、`Interrupted: the Merge guarantee check did not finish. Run ralph doctor again.` を出して error で終わる(`internal/cli/doctor.go:244-249`、`:319-327`)。`--strict` でも Check 13 を飛ばしたまま exit 1 になるので、検査の抜けが合格に見えることはない
- キャッシュの規則は変えていない。割り込みの unknown も、ほかの unknown と同じく書かれる(probe で、`merge-guarantee.json` に `the check was interrupted (interrupt signal received) while ssh -G github.com.emu ran` の unknown が入った)。そのため Ctrl-C で、それまでの良いキャッシュが unknown に置き換わる。README の「caches the result, `unknown` included」と矛盾はしない
- Node の検査の印字の文字列は変わらない(self-review が `240324fe^` と照らした。今回は読んだだけで、打っていない)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=0 | `==> All verifiers passed.`。gofmt は `ok`、go vet と staticcheck は出力なし、golangci-lint は `0 issues.`。shellcheck、`sh -n`、guard の awk の解析、settings の `jq -e`、Codex の hook の 3 つの guard、`tech-debt README plan references` は OK |
| `scripts/check-sync.sh`(上の中で) | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh`(上の中で) | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh`(上の中で) | OK | |
| `secret-scan-branch.sh`(上の中で) | clean | `e6d5d0ed..e41aeeca` |
| `./scripts/plan-visual.sh digest <plan>` | `f4a9803ca80e` | 計画の `- Approved:` の行と一致 |
| `git diff --check main...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only main HEAD`(`main` = `32274500`) | 衝突 1 件(rc=1) | `docs/tech-debt/README.md` の 1 行だけが衝突する(I1)。README と org skill の 4 写しは自動でマージされる |

## Documentation drift

`/sync-docs` に渡すもの(依頼に従い、ここでは直していない)。D1〜D8 はこの report の文書のずれの番号で、cycle 1 の D1〜D5 に続けた。計画の Design decisions の D1〜D6 とは別のもの:

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の doctor の行(`README.md:137`)と spec の 7c-2(`docs/specs/2026-10-07-org-multi-org-director.md:129`) | 一部(D6、LOW、新規) | doctor の検査が signal で止まるときの動きを、README も spec も言わない。doctor は `Interrupted: the Merge guarantee check did not finish. Run ralph doctor again.` を出し、後の検査を打たずに exit 1 で終わり、キャッシュは割り込みの unknown になる。README の doctor の行は exit code を変える条件を挙げている(「It fails when ...」「a `warn` never changes the exit code」「`--strict` exits 1 on ...」)が、この道はそこにない。先例を探すと、Node の検査の割り込み(#311 から、`Interrupted: the Node toolchain check did not finish. Run ralph doctor again.`)も README・spec・recipe のどこにも書かれていない(`git grep 'Run ralph doctor again'` は、Go 以外ではこの branch の計画と report にしか当たらない)。そのため書かないことは先例と揃っている。それでも利用者が見る行と exit code なので、README の doctor の行に両方の検査をまとめた 1 文を足すことを勧める。例: 「A SIGINT, SIGTERM, or SIGHUP while the `Merge guarantee` or `Node toolchain` check runs stops that check's commands (for `ssh -G`, its whole process group): doctor prints the lines so far and `Interrupted: the <check> check did not finish. Run ralph doctor again.`, runs no later check, and exits 1; an interrupted `Merge guarantee` is cached as `unknown` like any other.」。spec 7c-2 の「`unknown` が `info` で、doctor の終了コードを変えない」は種類と状態の対応の文で、割り込みは種類ではないので偽にはならない(INFO。直すかは `/sync-docs` の判断) |
| register の 419 行(「Five files went past the 800-line guide」、`docs/tech-debt/README.md:419`) | No(D7、LOW、新規) | この branch の注は「`internal/cli/doctor.go` (1651 → 1653 lines, counted at the merge base e6d5d0ed and at the sync-docs pass, the only commit of the branch that changed it being 3c369ff4 ...)」「the edit is a comment」と言う。`240324fe` が `printDoctorResults`、`doctorInterrupt`、Merge guarantee の割り込みの早い return、コメントの句を足し、HEAD では 1683 行(base 1651、`b01d04c1` 1653)。変えた commit は `3c369ff4` と `240324fe` の 2 つになり、2 つ目はコードの変更。数、commit、「comment だけ」の 3 点を直す。払わない理由(cross-review の直しで、計画の Scope は `doctor.go` を名指さない。移すだけの分割は直しと混ざる)は今も当てはまる。`240324fe` が変えたほかの 3 ファイルは 161、266、550 行で、800 行の目安の内側 |
| register に、cycle 2 の self-review が足すと書いた行がまだない | No(D8、LOW、新規) | `docs/reports/self-review-2026-11-19-org-ssh-alias-host.md` の Tech debt identified は、残した LOW の L4・L6・L7・L9・L11・L12 を 1 行にまとめ、`/sync-docs` が足すと書いている。register にはまだない(`L11`、`L12`、`signalSelf` で register に当たる行はない) |
| register のほかの行の Trigger | Yes | `240324fe` の 4 ファイルと、関数(`runDoctorStateDir`、`printDoctorResults`、`doctorInterrupt`、`checkMergeGuarantee`、`runMergeGuaranteeCheck`、`doctorMergeGuaranteeSignalContext`)、それに doctor・signal・interrupt の語で、開いた行の Trigger の欄(4 番目の欄)を走査した。当たったのは 419 行だけ。242 行(gate・merge・regate の `signal.NotifyContext`)の Trigger は 3 つの command の signal の扱いの変更で、doctor の seam は当たらない。500 行(`doctor_node_test.go`)と 241 行(`signalCauseContext`)のファイルは `240324fe` が触らない |
| `internal/cli/doctor.go` の `runDoctorOpts` の上のコメント、Check 11c のコメント、`doctor_merge_guarantee.go` のファイルの頭のコメントと `checkMergeGuarantee` の doc | Yes | signal の句が足され、`checkMergeGuarantee` の doc の「exit code never changes because of it」は「because of the kind」に直された |
| 計画(`docs/plans/active/2026-11-19-org-ssh-alias-host.md`) | Yes(INFO) | `240324fe` は Progress の 155 行に「計画からのずれ」として書かれ、申し送り(README に signal の動きがない)も書かれている(D6 はその判断の材料)。Scope と Affected areas は `internal/cli/doctor.go` を名指さず、AC にも doctor の signal はない。digest で固定された計画を触らずに Progress で記録する形で、`/pr` が archive する記録に残る。この verify の行は、まだ Progress にない(次の段で足される) |

cycle 1 から変わらず合っているもの(`5f2f0a0e` の文書の直しのあと、本番の `internal/org` と文書は変わっていない):

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の `ralph org merge` の段落、キャッシュの文 | Yes | |
| org skill の 4 写し | Yes | `check-skill-sync.sh` は ok |
| spec の 7c-2 の決定(`:129`)、8a-3a と #263 の決定(`:144`、`:145`) | Yes | 7c-2 は「この割り切りは #340 で改めた」。後の 2 つは日付のある決定の記録 |
| register の RESOLVED と打ち消しの行、director skill の行(`docs/plans/active/...` を引く) | Yes | `tech-debt README plan references` は OK。`/pr` の `scripts/archive-plan.sh` が archive のパスに書き換える |
| register の大きさ(cycle 1 の D3) | 解消 | `da80502f` で branch の新しいテストが新しいファイルに移った。`merge_test.go`(834 行、base で既に 819 行)は `5f2f0a0e` が 532 行に足した行が追う |
| `mergeorigin.go`・`mergeguarantee.go` の doc comment、`MergeEnforcementParams.Host`、`RegateParams.Exec` | Yes | |
| `ralph org merge`、`cleanup`、`regate` の `--help` | Yes(INFO) | 解けないときに origin の host のまま進むことは言わない(self-review の L8 (c)、残した) |
| AGENTS.md の repo map | Yes | 計画の D5 のとおり変えていない |
| 計画の Scope の 21〜22 行 | INFO | 解いた値を断る規則が計画を書いた時点の版のまま。計画の Design decisions の D6 と Risks が今の規則を言う |
| `.claude/skills/pr/SKILL.md:46` | 範囲外 | 計画の Progress(156 行)が PR の follow-up に回すと書いている |

## Observational checks

- HEAD `e41aeeca` の binary(`go build ./cmd/ralph`)を `$TMPDIR` に作り、`git init` と `git remote add origin ssh://git@github.com.emu/yoshpy-dev/ralph.git` の repo で `ralph doctor --state-dir <空の dir>` を打った。結果の行は Evidence の log の cycle 2 の節にある
  - signal なし、本物の `ssh`(このマシンの `Host github.com.emu`): rc 0、`✓ Merge guarantee: pass — github_protection`、キャッシュの `host` は `github.com.emu`、`api_host` は `github.com`。AC7 の evidence と同じ内容
  - PATH の先頭に偽の `ssh` を置き(自分の pid と pgid を書き、`sleep 300` の子を立てて `wait` する)、子が立ったあとに doctor の pid へ signal を送った。偽の ssh の pgid は自分の pid で、doctor の pgid と違った(`realMergeDetachedExec` の自分のグループ)
    - SIGINT: doctor は 0.037 秒で rc 1。ssh と子はどちらも消えた。出力は `ℹ Merge guarantee: info — unknown: the check was interrupted (interrupt signal received) while ssh -G github.com.emu ran, ...`、`Interrupted: the Merge guarantee check did not finish. Run ralph doctor again.`、`doctor: interrupted (interrupt signal received) while the Merge guarantee check read origin's repository`。Node toolchain の行は出ない。キャッシュは割り込みの unknown
    - SIGTERM: doctor は 0.038 秒で rc 1、ssh と子は消えた。原因は `terminated signal received`
  - 修正前(`240324fe^` = `79eb08dd` を `git archive` で取り出して作った binary)で同じ probe
    - SIGTERM: doctor は既定の動きで死に(rc 143)、ssh と子が残った(probe が `kill -9` で片付けた)。キャッシュは書かれない。cross-review の指摘が実プロセスで再現し、`240324fe` で閉じたことになる
    - SIGINT: doctor は SIGINT を受けず、5 秒の期限で ssh のグループを止めて最後まで走り、rc 0 だった。probe は doctor を非対話の `sh` の背景 job として起こすので、SIGINT が無視の状態で受け継がれる。HEAD の binary が同じ起こし方で SIGINT を受けたのは、`os/signal` の `Notify` が、無視の状態で起動したプロセスの SIGINT と SIGHUP にも handler を入れるため(I2)
- probe のあと、`sleep 300` と偽の ssh のプロセスが残っていないことを `ps` で確かめた

## Coverage gaps

- テストは走らせていない。`doctor_merge_guarantee_signal_test.go` の 2 つのテストが通るか、本物の SIGINT を自分に送るテストが負荷の高い実行で安定するかは `/test` で確かめる。go vet と golangci-lint はこのファイルを含めて通った(コンパイルは通る)
- probe は端末の Ctrl-C を打っていない。SIGINT は doctor の pid に送った。端末の Ctrl-C は前面のプロセスグループに届き、ssh の自分のグループには届かないが、そのことは pgid が違うことで確かめただけ
- SIGHUP は binary では打っていない(seam のテストが原因の文字列を確かめる)。本物の `Match exec` を書いた ssh の設定での実走、`ralph org merge`・`cleanup`・`regate_checks = "ci"` の regate の実走はしていない(計画の Non-goals)
- self-review の L12(signal が検査の最後の命令のあとに来たとき、本物の結果の行の下に「did not finish」が出る)の窓は probe で狙っていない
- I1: `main`(`32274500`)とのマージは `docs/tech-debt/README.md` の 419 行で衝突する。`main` の #349(fix/verify-docs-only-base)が同じ行の末尾に自分の carry-over の注を足したため。両方の注を残せば解ける。D7 の直しも同じ行なので、マージのときに合わせて直すと 1 回で済む
- I2: `signal.NotifyContext` は、`nohup` や非対話の shell の背景 job のように SIGINT・SIGHUP を無視して起動した doctor でも、Merge guarantee の検査の間はその signal を受けて止まる。`stop` のあとは無視に戻る。Node の検査の seam と `ralph org gate`・`merge`・`regate`・`cleanup` も同じで、先例と揃っている(INFO)
- 足りない決定的な検査: doctor の既定の経路を、本物の子プロセスと本物の signal で通すテストはない(in-process のテストは偽の exec が ctx の終わりを待つ形で、子が消えることは org の層の AC5b のテストが見る)。今回の probe がその間を埋めた。テストにするなら、`internal/cli` の TestMain の外で doctor の binary を子として起こし、偽の `ssh` の子の pid が消えることを見る形になり、大きい。今の 2 層のテストと今回の probe の記録で足りると判断した
- 確信を最も上げる追加の確かめ: `/test` で `go test ./internal/cli -run 'MergeGuaranteeInterrupt|MergeGuaranteeSignalContext' -count=3` を負荷のあるときにも打ち、本物の SIGINT の場合が安定することを見る
- Insight event: 依頼に従い、この report の commit のあとに `./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase verify --verdict pass --cycle auto --source skill --low 3` を打ち、events のファイルだけを別の commit にする。`--low 3` はこの report の LOW の D6・D7・D8 の数

## Cycle 1 の記録

cycle 1 の report の ID を引く文書があるので(計画の Progress の 159 行、sync-docs の report)、その ID の行き先を残す。

| ID | 中身 | 扱った commit | HEAD で |
| --- | --- | --- | --- |
| 1 回目の FAIL | register の director skill の行が計画を `docs/plans/archive/` で引いていた | `9783b9a7` | 解消(`tech-debt README plan references` は OK) |
| D1 | README の doctor の行の「no hostname that `gh` can take」 | `5f2f0a0e` | 解消 |
| D2 | Trigger に当たった register の行の carry-over の注 | `5f2f0a0e`(250、251、258、310、394、409、419、501) | 419 行の注は `240324fe` のあとの数と合わない(D7)。ほかの 7 行の数は `240324fe` が触らないファイルのもので、今も合う |
| D3 | register の大きさ(`merge_test.go`、`regate_ci_test.go`) | `da80502f`、`5f2f0a0e`(532 行) | 解消 |
| D4 | 計画の Progress に `9783b9a7`、`da80502f`、`14597e36` と skip の条件がない | `6a17dfad` | 解消 |
| D5 | evidence のテストのファイル名 | `5f2f0a0e` | 解消 |

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC5、AC5b(doctor の signal の道を含む。binary の probe で ssh とその子が消えることと、修正前の binary では残ることを見た)、AC6、AC7(HEAD の binary で打ち直した)、AC8(名指しの文書は合う)。`ssh -G` を打つ道がすべて signal で終わる ctx の下にあること。gofmt、go vet、golangci-lint、staticcheck、shellcheck、`check-sync.sh`、`check-skill-sync.sh`、`tech-debt README plan references`、secret scan は通った
- Partially verified: 文書。D6(README と spec に doctor の signal の動きがない。Node の先例も書かれていない)、D7(register の 419 行の数と commit)、D8(self-review が足すと書いた register の行)が `/sync-docs` に残る
- Not verified: テストの合否(`/test`)、端末から打った Ctrl-C と SIGHUP の binary での動き、本物の `Match exec` での実走、merge・cleanup・regate(ci)の実走、`main` を取り込んだ tree での検査(I1)
- Result: pass。`./scripts/run-static-verify.sh` は rc=0。AC はすべて満たし、cross-review の ACTION_REQUIRED #1 は `240324fe` で閉じた。`/sync-docs` に D6、D7、D8 を渡す。`/pr` の前に `main` を取り込むときは、419 行の衝突(I1)を両方の注を残して解く
