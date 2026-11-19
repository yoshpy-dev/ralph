# Verify report: org-ssh-alias-host

- Date: 2026-11-20(パイプラインの 3 回目の run。`cycle-count.json` は `cycle: 2`、`cap: 3`。cross-review の 2 回目で上限を 3 に上げ、cycle は 2 のまま。これまでの verify は cycle 1 の 2 回(`bc221a8d` が fail、`b01d04c1` が pass)と cycle 2 の `fd76b7c0`(pass))
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`。`./scripts/plan-visual.sh digest` の値 `f4a9803ca80e` は、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff main...HEAD`(merge base `32d6dc51`、HEAD `47d3276a`、41 ファイル、+3253 −217)。`main` と `origin/main` は `32d6dc51` で、branch は `e6a98fad` で取り込み済み。前回の verify(`fd76b7c0`)のあとの branch の commit は、cycle 2 の sync-docs(`a42eb2d0`、`d4a91a01`)、cycle 2 の cross-review triage(`ebbbfd1c`)、その直し `664d4771`(`mergeOriginFromURLAsWritten`、`mergeSSHHost`)と `04161b43`(doctor の中断の文)、run 3 の self-review(`ee9bae65`、`dced8301`)とその直し `74fef538`(L13、L15)、計画の記録(`8226d125`、`47d3276a`)、main の取り込み(`e6a98fad`、#348〜#350)。Go を変えたのは `664d4771`、`04161b43`、`74fef538` の 3 つで、`internal/org` の 6 ファイルと `internal/cli/doctor_merge_guarantee_signal_test.go` のコメントだけ。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-org-ssh-alias-host.log`(`.gitignore` の `docs/evidence/*.log` に当たり、commit されない)。cycle 2 までの内容のあとに run 3 の節を足した(足す前のファイルが新しいファイルの先頭のバイト列と一致することを `head -c | cmp` で確かめた)。静的検査の出力、行数、OpenSSH・git・gh・go-gh の確かめ、binary の probe、register の Trigger の走査が入っている。ラッパーは自分の記録として `docs/evidence/verify-2026-11-19-185028.log` も書いた(これも gitignore に当たる)

## Spec compliance

run 3 の直しが変えたのは、`ssh -G` に渡す host(origin の URL に書かれたとおり)と、その host を名指す文。AC の判定は前回と同じで、引用の行番号を HEAD に合わせ、run 3 の分を足した。最後の 2 行は計画の AC の外の約束で、cycle 1 と cycle 2 の cross-review の ACTION_REQUIRED #1 にあたる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 別名の origin で doctor の Merge guarantee が gh を `--hostname github.com` で打ち、種類を決める | Met | doctor の `readOrigin`(`internal/org/mergeguarantee.go:297`)は get-url のあと `resolveMergeAPIHost`(`internal/org/mergeorigin.go:178`、呼び出しは `mergeguarantee.go:311`)で `apiHost` を決める。`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`(`internal/org/mergeguarantee_alias_test.go:52`)は `664d4771` で `sshHost` の欄が増え、大文字の別名の 2 行(`:61`、`:64`)が書かれたとおりの host で `ssh -G` を打つことも確かめる。HEAD `47d3276a` の binary で、このマシンの `Host github.com.emu` を origin にした repo の doctor を打ち直し、`✓ Merge guarantee: pass — github_protection: ...(origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)` を見た(Observational checks) |
| AC2: merge が使う host(`ReadMergeOrigin` の `-R`、`CheckMergeEnforcement` の `--hostname`)と doctor の host が一致する | Met | 解く関数は 1 つで、呼び手は `ReadMergeOrigin`(`mergeorigin.go:143`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。`664d4771` は `ssh -G` に渡す host をこの関数の中(`mergeSSHHost`)で決めるので、2 つの呼び手は同じ値を渡し続ける。merge は `r.apiHost` を `CheckMergeEnforcement` に渡す。`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`(`mergeguarantee_alias_test.go:269`)と `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`(`internal/org/merge_ssh_alias_test.go:23`) |
| AC3: 別名でない origin の gh の host は変わらない。https と `git://` では ssh を打たない | Met | `resolveMergeAPIHost` は `mergeOriginIsSSH` が偽なら何も打たずに origin の host を返す(`mergeorigin.go:180-182`)。`mergeOriginFromURL`(`internal/org/mergeorder.go:945`)は `mergeOriginFromURLAsWritten`(`:954`)の host を小文字にして返すだけになり、4 つの呼び手が受ける値は変わらない(失敗のときも `""`)。既存の期待値で変わったのは `internal/org/merge_test.go` の 3 行で、入力に大文字のある行の ssh の引数だけ |
| AC4: キャッシュの `host` は origin に書かれた host、`api_host` は gh に渡した host。`ReadMergeGuarantee` は別名の origin で `origin_changed` にならない | Met | `TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`(`mergeguarantee_alias_test.go:201`)。HEAD の binary の doctor が書いたキャッシュは `"host":"github.com.emu","api_host":"github.com"` だった。キャッシュの `host` は今も小文字にした host で(`MergeGuaranteeCheck.Host` の doc は「as written, lowercased」)、AC の「origin に書かれた host のまま」は「解かない host」の意味で満たす。この計画の前から `host` は小文字だった(D15、INFO) |
| AC5: `ssh -G` の失敗・期限切れ・hostname なし・不正な値で origin の host のまま進み、gh が失敗したら doctor と merge の案内が理由と `git config core.sshCommand` の手を言う | Met | 理由の文は `resolveMergeAPIHost` の分岐(`mergeorigin.go:183-219`)、案内は `mergeSSHAliasHint`(`:306`)。テストは `TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`(`mergeguarantee_alias_test.go:126`)と `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost`(`merge_ssh_alias_test.go:77`)。`664d4771` のあと、理由の文は書かれたとおりの host で `ssh -G` を名指し(`ssh -G Work-GH failed (exit status 255)`、`internal/org/mergeorigin_test.go:165-166`)、fallback の host は小文字のまま。案内の文が小文字の host を出すことは L14 で、表示だけの差(D14) |
| AC5b: 本物の ssh は `realMergeDetachedExec` で打ち、専用の期限と親の ctx の取り消しの両方で戻り、子プロセスも終わる。本番の 4 つの経路が `realMergeDetachedExec` を渡す | Met | run 3 は実行関数とプロセスグループの扱いを変えていない。テストは `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(`mergeorigin_test.go:461`)、`TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault`(`:513`)、`TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn`(`mergeguarantee_alias_test.go:404`)、`TestMerge_CallsAfterGHPRMergeRunDetached`(`internal/org/merge_run_test.go:764`)、`TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`(`internal/org/regate_ci_ssh_alias_test.go:27`)、doctor の signal の `TestRunDoctor_MergeGuaranteeInterruptStopsSSHAndDoctor`(`internal/cli/doctor_merge_guarantee_signal_test.go:102`)と `TestDoctorMergeGuaranteeSignalContext_EndsOnSIGINTSIGTERMAndSIGHUP`(`:241`)。`664d4771` は `TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten`(`mergeorigin_test.go:538`)を足し、PATH の偽の ssh を `realMergeDetachedExec` で打つ道で書かれたとおりの host を確かめる。`internal/cli/doctor.go` と `doctor_merge_guarantee.go` は `fd76b7c0` から変わらないので、cycle 2 の signal の binary の probe は打ち直していない |
| AC6: 別名の origin で `https://github.com/<o>/<r>/pull/<n>` が origin の PR と認められる | Met | `hasPR` は解いた `apiHost` と比べる(変わらない)。`TestReadMergeOrigin_ResolvesAnSSHAlias`(`mergeorigin_test.go:339`) |
| AC7: 実機で、別名の origin の doctor が `--hostname github.com` で読み、unknown にならない | Met | `docs/evidence/ssh-alias-host-livefire-2026-11-19.md`(binary `7798b966`)。`664d4771`・`04161b43`・`74fef538` が本番の Go を変えたので、今回は HEAD `47d3276a` の binary で打ち直した。別名の origin で `github_protection`、キャッシュの `host` は別名、`api_host` は `github.com`、rc 0、`ssh -G` の引数は `github.com.emu`。AC は「一時 clone」と言うが、evidence も今回も `git init` と `git remote add origin` で作った repo を使った(doctor が読むのは origin の URL と GitHub 側なので判定は変わらない) |
| AC8: 文書が新しい挙動と合う。`check-skill-sync.sh` と `check-sync.sh` が通る | Met(注つき) | 2 つの検査は通った(Static analysis)。AC8 が名指す README の doctor の行、org skill の 4 写し、spec の 7c-2、register の RESOLVED の行は run 3 の挙動とも合う。どれも `ssh -G` が小文字の host を受けるとは言わない(Documentation drift の最初の行)。cycle 2 の D6〜D8 は `a42eb2d0` で直った。AC8 が名指す行の外に、register の注のずれ(D9〜D12)と Go の doc comment(D14、L14)が残る |
| 追加(cycle 1 の cross-review の ACTION_REQUIRED #1、計画の Progress の 155 行): `ssh -G` を打つ道はすべて、signal で終わる ctx の下で走る | Met | 非 test で `ssh` を実行するのは `mergeorigin.go:196` の 1 か所のまま。呼び手は `merge.go:429`、`regate_ci.go:94`、`internal/cli/org_cleanup.go:206`(どれも `ReadMergeOrigin`)と、doctor の `runMergeGuaranteeCheck`(`internal/cli/doctor_merge_guarantee.go:114`、`CheckMergeGuarantee` の唯一の呼び手)で、run 3 はこの形を変えていない |
| 追加(cycle 2 の cross-review の ACTION_REQUIRED #1、計画の Progress の 156〜158 行): `ssh -G` は origin の URL に書かれたとおりの host を受ける。git と gh が渡す host と同じ | Met | コード: `mergeSSHHost`(`mergeorigin.go:245`)は `mergeOriginFromURLAsWritten` の host を返す。`resolveMergeAPIHost` はそれを not-run の文、`cmd`、引数に使い(`:183`、`:193`、`:196`)、`mergeSSHResolveRuns`(`:228`)はそれが plain な名前かを見る。doctor の中断の文もそれを名指す(`mergeguarantee.go:313`、`04161b43`)。`74fef538` のあと、小文字の host に戻るのは URL が host を名指さないときだけ(`!ok \|\| written == ""`)で、同じ行から host を読む 2 つの呼び手はそこに届かない。テスト: `mergeorigin_test.go:159-168`(scp 形と `ssh://` の大文字、port、失敗、hostname なし)、`:178`(U+0130)、`:376-393`(`ReadMergeOrigin`)、`:538`(本物の実行関数、偽の ssh は `Work-GH` にだけ答える)、`mergeguarantee_alias_test.go:61`、`:64`、`:372`(doctor)。実物での確かめ: git 2.49.0 は `git@GitHub.COM.EMU` と `git@Work-GH` を書かれたとおり ssh に渡した(偽の `GIT_SSH_COMMAND`)。go-gh の `pkg/ssh/ssh.go` は `u.Hostname()` をそのまま `ssh -G` に渡す(`:35`、`:78`。gh の `git.ParseURL` は小文字にしない)。gh 2.103.3 は大文字の origin で `ssh -G GitHub.com.EMU` を打った。HEAD の ralph も `ssh -G GitHub.com.EMU` を打ち、直す前の `ebbbfd1c` は `ssh -G github.com.emu` を打った。このマシンの設定(`Host github.com.emu`)では、大文字の origin に対して gh は既知の host を見つけられず、HEAD の doctor は unknown、直す前の doctor は github.com を読んで pass だった。HEAD は gh と揃い、直す前は揃っていなかった |

計画の Verify plan が挙げる「gh の解き方と同じ順序」(解く、小文字にする、`ssh.github.com` の読み替え)は、cycle 1 の 1 回目に go-gh の `pkg/ssh/ssh.go` と照らした。今回、同じファイルを `gh api` で取り直し、`ssh -G` に渡す host の大文字小文字も gh と揃ったことを見た。

### run 3 の Go の直しの読み

- `664d4771`: parse は `mergeOriginFromURLAsWritten` の 1 か所に残り、`mergeOriginFromURL` はその host を小文字にする薄い wrapper になった。解いた hostname、fallback、キャッシュの `host`、`MergeOrigin.originHost` は小文字のまま。`ssh -G` の引数、名前の確かめ、理由の文は書かれたとおりの host を使う
- `04161b43`: doctor の中断の文(`while ssh -G <host> ran`)が `cmd` と同じ host を名指す
- `74fef538`: L13。`mergeSSHHost` から `strings.EqualFold` の条件を外した。origin が `git@İ.example.com:o/r.git` なら、書かれたとおりの host は ASCII でないので ssh は走らず、gh の host は小文字の `i.example.com`、理由は `ssh -G was not run for the host "İ.example.com", ...` になる(`mergeorigin_test.go:178`)。L15 はテストのコメントの指し先だけ
- 残した L14: `mergeSSHAliasHint` と `aliasNote` は小文字の host を出す。origin が `git@Work-GH:o/r.git` で ssh -G が失敗すると、「`ssh -G Work-GH failed (...)`, so gh read the host work-gh as origin names it: if work-gh is an ssh alias」と同じ文に 2 つの綴りが並ぶ(`mergeorigin.go:306-312`、`mergeguarantee.go:326-331`)。動きは変わらない
- 意図した挙動の変化(cycle 2 の binary との差): origin の host の大文字小文字が `~/.ssh/config` の `Host` の行と違うとき、ralph は git と gh と同じく、その `Host` を使わなくなった。このマシンでは、大文字の origin の doctor が pass から unknown(gh の到達の案内だけ)に変わった。そのような origin では git の push も別名を使わないので、git で動く設定が ralph で動かなくなることはない

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=0 | `==> All verifiers passed.`。gofmt は `ok`、go vet と staticcheck は出力なし、golangci-lint は `0 issues.`。shellcheck、`sh -n`、guard の awk の解析、settings の `jq -e`、Codex の hook の 3 つの guard、`tech-debt README plan references` は OK |
| `scripts/check-sync.sh`(上の中で) | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh`(上の中で) | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh`(上の中で) | OK | |
| `secret-scan-branch.sh`(上の中で) | clean | `32d6dc51..47d3276a` |
| `./scripts/plan-visual.sh digest <plan>` | `f4a9803ca80e` | 計画の `- Approved:` の行と一致 |
| `git diff --check main...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| base | 試しのマージは不要 | `main`、`origin/main`(`git ls-remote`)、merge base がどれも `32d6dc51`。cycle 2 の I1(register の衝突)は `e6a98fad` で解けた |

## Documentation drift

`/sync-docs` に渡すもの(依頼に従い、ここでは直していない)。D9〜D15 はこの report の文書のずれの番号で、cycle 2 の D6〜D8 に続けた。計画の Design decisions の D1〜D6 とは別のもの。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の doctor の行(`README.md:137`)と `ralph org merge` の段落(`:296`)、org skill の 4 写し(`.claude/skills/org/SKILL.md:2019-2024` ほか)、spec の 7c-2(`docs/specs/2026-10-07-org-multi-org-director.md:129`)、register の RESOLVED の行(`docs/tech-debt/README.md:253`)、実機の evidence | Yes | `ssh -G` が小文字の host を受けると言う文書はない。README は「the last `hostname` that `ssh -G <host>` prints」「it is not run when origin's host is not a plain host name」、skill は「`ssh -G <別名>`」、RESOLVED の行は「`ssh -G <host>` as gh does」と書き、どれも origin に書かれた host と読める。README の「not run when origin's host is not a plain host name」は、書かれたとおりの host を確かめる今のコード(`mergeSSHResolveRuns`)と合う。evidence(binary `7798b966`)の origin は小文字なので、run 3 の直しの影響を受けない |
| D14: `internal/org/mergeguarantee.go` と `merge.go` の doc comment、案内の文(self-review の L14) | No(LOW) | 計画の Verify plan が名指す `CheckMergeGuarantee` の doc の step 1(`mergeguarantee.go:168-172`)は「read by mergeOriginFromURL (host and owner/repo). For an ssh origin, `ssh -G <host>`」で、小文字の host を渡すと読める。`MergeGuaranteeCheck.APIHost` の doc(`:132-136`)は「the hostname `ssh -G <Host>` gave」で、`Host` の doc(`:127-130`)は小文字と書く。`merge.go:131-134` の `originHost` は「as its URL names it」で、小文字とも書かない。`mergeSSHAliasHint` と `aliasNote` の文は、隣の理由の文が書かれたとおりの host を名指すのに、小文字の host を出す。既知の L14 で、依頼では `/sync-docs` が register に回す。直すと `internal/` の Go が変わるので記録だけの修正の例外に入らず、cap 3/3 では register に書くほうが小さい |
| D9: register の「Small leftovers of fix/org-ssh-alias-host (#340)」の行(`docs/tech-debt/README.md:535`) | No(LOW) | (1)「left in place at the last pipeline run (cycle 2 of 2)」と Why deferred の「the pipeline was at its last run (cap 2)」は、cap を 3 に上げたので偽。(2) (a) の「takes seven arguments」は 6 つ(`ctx, dir, sshRun, timeout, originURL, host`)。`664d4771` で `originURL, host` の組を取る関数が 1 つ増えた(`mergeSSHHost`)。(3) (c) の「seven positional fields」は、`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves` の表が `664d4771` で 8 欄になった(`mergeorigin_test.go` の主な表は `fd76b7c0` で既に 8 欄)。(4) L14 がない。L13 と L15 は `74fef538` で直ったので、残りとしては足さない。(5) この行自身の Trigger「The next change to `resolveMergeAPIHost` or to the tests of `mergeorigin.go` and `mergeguarantee.go` for (a) to (d)」は `664d4771` と `74fef538` が、(g) の「the next change to ... `doctor_merge_guarantee_signal_test.go`」は `74fef538`(コメントだけ)が満たし、どれも払っていない。同じ PR の中の変更だと注で言うか、Trigger を「この PR のあとの変更」と読める形にする。Related の欄に run 3 の self-review(L14)を足す |
| D10: register の「Small leftovers of the org-merge-order PR (stage 7c-1)」の行(`:252`) | No(LOW) | #340 の carry-over の注は「`mergeorder.go` and `internal/cli/org_merge_order.go` were not touched」と言うが、`664d4771` が `mergeorder.go` を変えた(1438 → 1447 行。`mergeOriginFromURL` が新しい `mergeOriginFromURLAsWritten` を包む形になり、Trigger が名指す `mergeRepoFromOriginURL` はそれを呼ぶ)。Trigger「The next edit to `internal/org/mergeorder.go` ...」に当たり、払っていない。注を直す |
| D11: register の `mergeguarantee.go` と `mergeorder.go` の大きさの行(`:311`) | No(LOW) | #340 の注の「`mergeorder.go` was not touched (the runners stay in it), so the other part of this trigger did not fire」は、`664d4771` のあとは偽。「For `mergeorder.go`, its next change: move the runners」に当たり、runner は動かしていない。同じ注の `mergeguarantee_test.go`「1334 → 1396」は HEAD で 1398、`mergeguarantee_alias_test.go`「(418 lines)」は 446 |
| D12: register の「`internal/org/mergeorder.go` grew from 1291 lines at main」の行(`:423`) | No(LOW) | Trigger「The next change to `internal/org/mergeorder.go`」に `664d4771` が当たる(1438 → 1447 行)が、#340 の carry-over の注がない。ranking と PR の lookup を動かしていない注を足す |
| D13: register の `merge_test.go` の行(`:534`)と、stage 7c-2 の残りの行(`:259`) | INFO | `:534` は「834 lines」「the branch added 15 lines」と書くが、`664d4771` が大文字の入力の 3 行の期待値をその場で直して 835 行、16 行になった。数は「at the sync-docs pass」と時点つきで、Trigger(「more than an in-place fix」)には当たらない。`:259` の Trigger「The next edit to `internal/org/mergeguarantee.go`」に `04161b43` の 1 行が当たるが、注の 877 → 937 行は HEAD でも合い、`readOrigin` を変えたことも注に書かれている |
| D15: キャッシュの `host` の言い方(spec 7c-2 の「キャッシュは origin に書かれた host(`host`)のまま」、README の「with origin's host as its URL names it (`host`)」、計画の Scope 26 行と AC4) | INFO | キャッシュの `host` は小文字にした host(`MergeGuaranteeCheck.Host` の doc)。`664d4771` のあと、計画の Progress(157 行)と `mergeSSHHost` の doc では「書かれたとおり」が大文字小文字を保つ意味になったので、大文字の origin では spec の「書かれた host のまま」が実際と違って読める。spec は同じ段落で照合が大文字小文字を無視すると書き(「大文字小文字を無視してもキャッシュのものと違う」)、小文字の origin では何も偽にならない。直すなら「(小文字にした)」を足す |
| register の大きさ(cycle 1 の D3。register の `merge_test.go` の行が引く) | 解消のまま | `da80502f` で branch の新しいテストが新しいファイルに移り、`merge_test.go` は `5f2f0a0e` が足した行(今の `:534`)が追う。HEAD の数は D13 |
| register のほかの行の Trigger | Yes | run 3 が触ったファイルと関数(`mergeorigin.go`、`mergeorder.go`、`mergeguarantee.go`、その test、`merge_test.go`、`doctor_merge_guarantee_signal_test.go`、`mergeOriginFromURL`、`resolveMergeAPIHost`、`mergeSSH*`、`readOrigin`、`ReadMergeOrigin`)と ssh -G・merge guarantee の語で、開いた行の Trigger の欄(4 番目の欄)を python で走査した。当たった行のうち、run 3 の変更が満たすのは `:252`、`:259`、`:311`、`:423`、`:534`、`:535`(上の D9〜D13)。`:241`、`:249`、`:256`〜`:258`、`:298`、`:316`、`:317`、`:395`、`:410`、`:420`、`:502` は、語が経緯の文にだけあるか、run 3 が触らないファイルを名指す |
| cycle 2 の D6(README に doctor の signal の動きがない) | 解消 | `a42eb2d0` が README の doctor の行に 1 文を足した。コードと合う: 割り込みの文は `internal/cli/doctor.go:320` で両方の検査に共通、Merge guarantee のあとの検査は打たない(`:244-249`)、Node の probe は自分のプロセスグループで動き、グループごと止まる(`internal/toolchain/run.go:74`、`:81`、`:109-126`) |
| cycle 2 の D7(register の `doctor.go` の行の数と commit) | 解消 | 今の `:420` は 1651 → 1683 行と、`3c369ff4`(コメント)と `240324fe`(コード)の 2 commit を言う。HEAD は 1683 行で、run 3 は `doctor.go` を触らない |
| cycle 2 の D8(self-review が足すと書いた register の行) | 解消。ただし D9 | `a42eb2d0` が `:535` を足した |
| 計画(`docs/plans/active/2026-11-19-org-ssh-alias-host.md`) | Yes(INFO) | Progress の 156〜160 行が `664d4771`、`04161b43`、run 3 の self-review、`74fef538` と、L14 を register に回すことを記録している。digest で固定された本文に「書かれたとおりの host」の AC はなく、cross-review の直しとして Progress に記録する形は前の run と同じ。AC4 と Scope 26 行の言い方は D15。この verify の行は、まだ Progress にない(次の段で足される) |
| insight event の cycle | INFO | `--cycle auto` は、この 3 回目の run の event に cycle 2 を付ける。run 3 の self-review の event(`dced8301`)と同じで、`internal/insights` は run 2 と run 3 を同じ cycle 2 で数える |
| `.claude/skills/pr/SKILL.md:46` | 範囲外 | 計画の Progress(161 行)が PR の follow-up に回すと書いている |

## Observational checks

HEAD `47d3276a` と、直す前の `ebbbfd1c`(= `664d4771^`、`git archive` で取り出した)の binary を `$TMPDIR` に作った。repo は `git init` と `git remote add origin <URL>` で作り、`ralph doctor --state-dir <空の dir>` を打った。PATH の先頭に、引数を記録してから `/usr/bin/ssh` を exec する包みを置いた。出力は Evidence の log の run 3 の節にある。

| binary | origin | `ssh -G` の引数 | Merge guarantee | キャッシュ |
| --- | --- | --- | --- | --- |
| HEAD | `ssh://git@github.com.emu/yoshpy-dev/ralph.git` | `github.com.emu` | pass、`github_protection`、別名の注記つき、rc 0 | `host` `github.com.emu`、`api_host` `github.com` |
| HEAD | `ssh://git@GitHub.com.EMU/yoshpy-dev/ralph.git` | `GitHub.com.EMU` | info、unknown: `gh api --hostname github.com.emu repos/yoshpy-dev/ralph failed (... error connecting to github.com.emu ...)`、gh の到達の案内だけ、rc 0 | `host` `github.com.emu`、`api_host` `github.com.emu` |
| `ebbbfd1c` | 小文字の別名 | `github.com.emu` | pass(HEAD と同じ行) | HEAD と同じ |
| `ebbbfd1c` | 大文字の別名 | `github.com.emu` | pass、`github_protection` | `host` `github.com.emu`、`api_host` `github.com` |

- OpenSSH_10.2p1: scratch の `-F` 設定(`Host Work-GH`、`HostName github.com`)で `ssh -G Work-GH` は `hostname github.com`、`work-gh` と `WORK-GH` は `hostname work-gh`。このマシンの設定では `ssh -G github.com.emu` が `hostname github.com`、`ssh -G GitHub.com.EMU` が `hostname github.com.emu`。`Host` の照合は大文字小文字を区別する
- git 2.49.0: 偽の `GIT_SSH_COMMAND` に `git ls-remote` を打たせると、`-G -o SendEnv=GIT_PROTOCOL git@GitHub.COM.EMU`、`git@GitHub.COM.EMU git-upload-pack 'o/r.git'`、`ssh://git@Work-GH/o/r.git` では `git@Work-GH` を渡した
- gh 2.103.3: 上の 2 つの repo で `gh repo view` を打つと、小文字の別名は `yoshpy-dev/ralph` を返し、大文字の別名は `none of the git remotes configured for this repository point to a known GitHub host` で rc 1。包みの記録では、gh は `ssh -G github.com.emu` と `ssh -G GitHub.com.EMU` を打った
- go-gh(`gh api repos/cli/go-gh/contents/pkg/ssh/ssh.go`): `Translate` は `t.resolve(u.Hostname())`、`resolve` は `newCommand(sshPath, "-G", hostname)`。小文字にするのはキャッシュの key だけ。gh の `git/url.go` の `ParseURL` も host を小文字にしない
- signal の probe(cycle 2 で HEAD `e41aeeca` の binary に SIGINT・SIGTERM を送った)は打ち直していない。`internal/cli/doctor.go` と `doctor_merge_guarantee.go` は `fd76b7c0` から変わらず、`mergeguarantee.go` の変更は中断の文の host だけ

## Coverage gaps

- テストは走らせていない。`664d4771` と `74fef538` が足した表の行(大文字の host、U+0130)と `TestReadMergeOrigin_RealSSHGetsOriginsHostAsWritten` が通るかは `/test` で確かめる。go vet と golangci-lint はこれらのファイルを含めて通った(コンパイルは通る)
- `ssh -G` に渡すのは host だけで、user は渡さない(gh と同じ)。git は `git@<host>` を渡すので、ssh の設定に `Match user` の区画があると、git と ralph・gh で読む設定が違いうる。run 3 で変わった点ではない(INFO)
- 大文字の host で ssh -G が失敗したときの案内の文(L14)は、binary の probe では通っていない(大文字の probe は ssh -G が解けた形になった)。理由の文はユニットテスト(`mergeorigin_test.go:165-166`)が固定し、案内の文の大文字の組を固定するテストはない
- 本物の `Match exec` を書いた ssh の設定での実走、`ralph org merge`・`cleanup`・`regate_checks = "ci"` の regate の実走はしていない(計画の Non-goals)。HEAD の binary に signal を送る probe は打ち直していない(上の理由)
- 確信を最も上げる追加の確かめ: `/test` で `go test ./internal/org -run 'ResolveMergeAPIHost_ResolvesAnSSHOriginsHostAsGHDoes|ReadMergeOrigin_ResolvesAnSSHAlias|ReadMergeOrigin_RealSSHGetsOriginsHostAsWritten|ReadOriginNamesSSHAsWhatRanOnlyWhenItRan|SSHAliasOriginIsReadOnTheHostSSHResolves' -count=3` を打ち、`mergeSSHHost` が `host` を返すように戻した変異でこれらが落ちることを見る
- Insight event: 依頼に従い、この report の commit のあとに `./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase verify --verdict pass --cycle auto --source skill --low 5` を打ち、events のファイルだけを別の commit にする。`--low 5` はこの report の LOW の D9、D10、D11、D12、D14 の数(D13 と D15 は INFO)

## Cycle 2 の記録

計画の Progress(166 行と 169 行)と sync-docs の report が cycle 2 の report の ID を引くので、その ID の行き先を残す。cycle 2 の判定は pass(`fd76b7c0`、HEAD `e41aeeca`)だった。

| ID | 中身 | 扱った commit | HEAD で |
| --- | --- | --- | --- |
| D6 | README と spec に doctor の signal の動きがない | `a42eb2d0`(README の doctor の行) | 解消 |
| D7 | register の `doctor.go` の行(当時 419 行、今 420 行)の数と commit | `a42eb2d0` | 解消 |
| D8 | self-review が足すと書いた register の行がない | `a42eb2d0`(今の 535 行) | 行はある。cap と cycle、引数の数、表の欄の数、L14 がずれた(D9) |
| I1 | `main`(当時 `32274500`)とのマージが register の 419 行で衝突する | `e6a98fad` | 解消(`main` = `origin/main` = merge base `32d6dc51`) |
| I2 | `signal.NotifyContext` は、無視の状態で起動した doctor の SIGINT・SIGHUP も検査の間は受ける | なし(INFO) | 変わらない |

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

- Verified: AC1、AC2、AC3、AC4、AC5、AC5b、AC6、AC7(HEAD `47d3276a` の binary で打ち直した)、AC8(名指しの文書は合う)。cycle 2 の cross-review の直しの約束(`ssh -G` は origin の URL に書かれたとおりの host を受ける)は、コード、テストの行、HEAD と直す前の binary の比較、git・gh・go-gh の実物の確かめで、git と gh に揃ったことを見た。`ssh -G` を打つ道がすべて signal で終わる ctx の下にあることは変わらない。gofmt、go vet、golangci-lint、staticcheck、shellcheck、`check-sync.sh`、`check-skill-sync.sh`、`tech-debt README plan references`、secret scan は通った
- Partially verified: 文書。register の注(D9 の 535 行、D10 の 252 行、D11 の 311 行、D12 の 423 行)と、Go の doc comment と案内の文(D14、L14)が `/sync-docs` に残る。D13 と D15 は INFO
- Not verified: テストの合否(`/test`)、大文字の host で ssh -G が失敗したときの案内の文の実物での表示、本物の `Match exec` での実走、merge・cleanup・regate(ci)の実走、HEAD の binary への signal の probe(該当コードは cycle 2 から不変)
- Result: pass。`./scripts/run-static-verify.sh` は rc=0。AC はすべて満たし、cycle 2 の cross-review の ACTION_REQUIRED #1 は `664d4771` と `04161b43` で、run 3 の self-review の L13 は `74fef538` で閉じた。`/sync-docs` に D9〜D12 と D14 を渡す(D13、D15 は任意)
