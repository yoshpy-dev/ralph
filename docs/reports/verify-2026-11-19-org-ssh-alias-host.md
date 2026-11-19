# Verify report: org-ssh-alias-host

- Date: 2026-11-20(cycle 1 の 2 回目。`cycle-count.json` は `cycle: 1`。1 回目は 2026-11-19 の `bc221a8d` で fail)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`。`./scripts/plan-visual.sh digest` の値 `f4a9803ca80e` は、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(base は merge-base の `e6d5d0ed`、HEAD `14597e36`、33 ファイル、+2149 −186)。1 回目のあとの 3 つの commit を含む。`9783b9a7`(register の director skill の行の計画のパスを active に戻した)、`da80502f`(merge と regate(ci)の ssh の別名のテストを新しい 2 つのファイルに移した)、`14597e36`(本物の ssh の期限のテストを 3 秒にし、偽の ssh が子の pid を書く前に止まったときは skip する)。手元の `origin/main` は `f280f2d6` に進んでいる(Static analysis の試しのマージを参照)。マージはしていない。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-org-ssh-alias-host.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。静的検査の出力、`da80502f` が移動だけであることの確かめ、行数、digest、試しのマージを入れた。ラッパーは自分の記録として `docs/evidence/verify-2026-11-19-150518.log` も書いた(これも gitignore に当たる)

## Spec compliance

1 回目のあと、本番の Go のファイルは変わっていない(`git diff --name-only 0a71a930 HEAD -- '*.go' ':!*_test.go'` が 0 件)。そのため本番のコードの行の引用と、1 回目に HEAD `0a71a930` の binary で打った観察は、今の HEAD にもそのまま当てはまる。行が動いたのはテストの引用だけで、下の表はそれを直した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 別名の origin で doctor の Merge guarantee が gh を `--hostname github.com` で打ち、種類を決める | Met | doctor の `readOrigin`(`internal/org/mergeguarantee.go:297`)は get-url のあと `resolveMergeAPIHost`(`internal/org/mergeorigin.go:169`、呼び出しは `mergeguarantee.go:311`)で `apiHost` を決め、gh の呼び出しはすべてその host を `--hostname` に取る。`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`(`internal/org/mergeguarantee_alias_test.go:49`)が、scp 形、大文字と port つきの `ssh://`、`ssh.github.com` に解ける別名、大文字の hostname、保護のない別名の 5 件で、種類、`Host`、`APIHost`、Detail、gh の endpoint と `--hostname github.com` を確かめる。1 回目に打った HEAD の binary の doctor も `github_protection` になった |
| AC2: merge が使う host(`ReadMergeOrigin` の `-R`、`CheckMergeEnforcement` の `--hostname`)と doctor の host が一致する | Met | 解く関数は 1 つで、呼び手は `ReadMergeOrigin`(`mergeorigin.go:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。merge は `r.apiHost` を `CheckMergeEnforcement` に渡す(`internal/org/merge.go:362`)。`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`(`mergeguarantee_alias_test.go:250`)が、同じ fake の `ssh -G` の答えで、doctor の `APIHost`、`ReadMergeOrigin` の `Arg`、`CheckMergeEnforcement` の gh の host を、解けたとき(`github.com`)と失敗したとき(別名のまま)の両方で照らす。merge の側は `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`(`internal/org/merge_ssh_alias_test.go:23`、`da80502f` で `merge_test.go` から移った) |
| AC3: 別名でない origin の gh の host は変わらない。https と `git://` では ssh を打たない | Met | `resolveMergeAPIHost` は `mergeOriginIsSSH`(`mergeorigin.go:260`)が偽なら何も打たずに origin の host を返す(`:171-173`)。表のテスト(`internal/org/mergeorigin_test.go:112-117`、`:153`、`:166-168`)、`TestMergeGuarantee_HostComesFromOriginsURL`(`internal/org/mergeguarantee_test.go:755`)、`TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`(`internal/org/merge_test.go:740`。`da80502f` の移動で 846 行目から上がった。`Arg` の期待値は base と同じ)。1 回目の HEAD の binary でも、`git@github.com:` と `https://github.com/` の origin の行は evidence の 8.2.1 の行と同じだった |
| AC4: キャッシュの `host` は origin に書かれた host、`api_host` は gh に渡した host。`ReadMergeGuarantee` は別名の origin で `origin_changed` にならない | Met | `MergeGuaranteeCheck.APIHost`(`mergeguarantee.go:137`、`omitempty`)と `MergeGuarantee.APIHost`(`:781`)。`ReadMergeGuarantee`(`:815`)は今のまま `Host` と origin の host を比べ、ssh を打たない。`TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`(`mergeguarantee_alias_test.go:182`)が、キャッシュの JSON(version 1、`host` は別名、`api_host` は `github.com`)、読み戻し、`--json` の 2 つの key、git 1 回で gh と ssh 0 回、実ホストに向け直した origin の `origin_changed`、`api_host` のない古いキャッシュを確かめる |
| AC5: `ssh -G` の失敗・期限切れ・hostname なし・不正な値で origin の host のまま進み、gh が失敗したら doctor と merge の案内が理由と `git config core.sshCommand` の手を言う | Met | 理由の文は `resolveMergeAPIHost` の分岐(`mergeorigin.go:174-209`)、案内の節は `mergeSSHAliasHint`(`:275`)。doctor は `readRepository`(`mergeguarantee.go:348`)、merge は `ghAccess`(`merge.go:398`)と `ghMissing`(`:405`)で足す。表のテストは `mergeorigin_test.go:118-139`、`:170-176`。案内は doctor の `TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`(`mergeguarantee_alias_test.go:107`)と merge の `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost`(`merge_ssh_alias_test.go:77`、移動のみ) |
| AC5b: 本物の ssh は `realMergeDetachedExec` で打ち、専用の期限と親の ctx の取り消しの両方で戻り、子プロセスも終わる。本番の 4 つの経路が `realMergeDetachedExec` を渡す | Met(条件つき。下の注) | 実プロセスのテストは `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(`mergeorigin_test.go:410`)。`14597e36` で期限の場面は 1 秒から 3 秒になり(期待する理由の文も `timed out after 3s`、`resolveMergeAPIHost` の `%s` の出し方と合う)、戻るまでの上限は `min(3s, 10s) + 5s` = 8 秒になった。本番の経路: doctor は `newMergeGuaranteeRun`(`mergeguarantee.go:216`)を `TestMergeGuarantee_SSHRunsThroughExecOrInAProcessGroupOfItsOwn`(`mergeguarantee_alias_test.go:376`)が照らす。merge は `newMergeCheck`(`merge.go:197`)を `TestMerge_CallsAfterGHPRMergeRunDetached`(`internal/org/merge_run_test.go:764`)が、`readOrigin` が ssh を `runDetached` で打つことを `TestCheckMerge_ReadOriginRunsSSHThroughRunDetached`(`merge_ssh_alias_test.go:53`)が確かめる。regate(ci)は `newRegateCI`(`internal/org/regate_ci.go:62`)を `TestRegateCI_ResolvesAnSSHAliasOriginThroughExec`(`internal/org/regate_ci_ssh_alias_test.go:27`)が reflect で照らし、実プロセスのグループを `TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec`(`:62`)が確かめる(どちらも移動のみ)。cleanup は `org.ReadMergeOrigin(ctx, "", nil, nil, 0)`(`internal/cli/org_cleanup.go:206`)で nil を渡し、nil の既定が自分のグループになることは `TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault`(`mergeorigin_test.go:462`)が確かめる。cleanup が nil を渡すこと自体を固定するテストはない(Coverage gaps)。注: `14597e36` の `t.Skipf`(`mergeorigin_test.go:441-447`)は、期限の場面で偽の ssh が子の pid を書かないまま止められたとき、子が終わることの確かめを飛ばす。呼び出しが戻ることと理由の文の確かめは skip の前の `t.Errorf` で、Go は失敗のあとの skip を失敗として報告するので、その 2 つは弱まっていない。普通の負荷では pid はおそらく 3 秒の期限よりずっと早く書かれ、確かめは走る(未計測)。ただし負荷の高い実行では AC5b の「期限でも子が終わる」がその実行では確かめられない。テストの意図の条件つきの緩めなので、計画と report に書く必要がある(Documentation drift の D4) |
| AC6: 別名の origin で `https://github.com/<o>/<r>/pull/<n>` が origin の PR と認められる | Met | `MergeOrigin.hasPR` は解いた `apiHost` で比べる。`TestReadMergeOrigin_ResolvesAnSSHAlias`(`mergeorigin_test.go:309`)が、github.com の PR は真、別名の host の PR は偽、`foreignPRWhy` が空を確かめる。merge(`merge_ssh_alias_test.go:23`)と regate(ci)(`regate_ci_ssh_alias_test.go:27`、`gate_ci_passed` を記録)も github.com の PR を origin のものとして進む |
| AC7: 実機で、別名の origin の doctor が `--hostname github.com` で読み、unknown にならない | Met | `docs/evidence/ssh-alias-host-livefire-2026-11-19.md`(binary `7798b966`)。1 回目の verify が HEAD `0a71a930` の binary で同じ 4 つの場面を打ち直し、path と `checked_at` を除いて evidence の行と同じだった。そのあと本番の Go は変わっていないので、今回は打ち直していない。AC は「一時 clone」と言うが、evidence は `git init` と `git remote add origin` で作った repo を使った(evidence はそう書いている。doctor が読むのは origin の URL と GitHub 側なので判定は変わらない) |
| AC8: 文書が新しい挙動と合う。`check-skill-sync.sh` と `check-sync.sh` が通る | 一部 | 2 つの検査は通った。register の計画のパスも直り、`tech-debt README plan references` は OK になった。org skill の 4 写し、spec の 7c-2、register の RESOLVED と新しい行、doc comment、`--help` は挙動と合う。README の doctor の行の 1 句は、まだ `3c369ff4` のあとの挙動と合わない(D1)。依頼に従い `/sync-docs` に渡す |

計画の Verify plan が挙げる「gh の解き方と同じ順序」は 1 回目に go-gh の `pkg/ssh/ssh.go` と照らした(解く、小文字にする、`mergeGuaranteeAPIHost`)。`mergeorigin.go` は変わっていないので、その判断はそのまま当てはまる。

### `da80502f` が移動だけであること

- `git show da80502f^` の `merge_test.go` の `^func ` の行の集合は、`da80502f` の `merge_test.go` と `merge_ssh_alias_test.go` を合わせた集合と同じ。`regate_ci_test.go` と `regate_ci_ssh_alias_test.go` も同じ
- `git diff -U0 da80502f^ da80502f` で `merge_test.go` と `regate_ci_test.go` に足された行は 0 行。消えた行を新しいファイルと `diff -B` で比べると、違いは新しいファイルの `package`、`import`、ファイルの頭の comment だけで、`regate_ci_test.go` から消えた `"fmt"` と `"syscall"` の import は新しいファイルの import に入った
- 移したテストを古いファイルの名前で引く文は、repo の中に 1 つもない(`git grep` の結果、この report を除く)。この branch の evidence と self-review report はファイル名で引く(D5)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=0 | `==> All verifiers passed.`。`tech-debt README plan references` は OK。gofmt は `ok`、golangci-lint は `0 issues.`、go vet と staticcheck は出力なし。shellcheck、`sh -n`、guard の awk の解析、settings の `jq -e`、Codex の hook の 3 つの guard は OK |
| `scripts/check-sync.sh`(上の中で) | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh`(上の中で) | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh`(上の中で) | OK | |
| `secret-scan-branch.sh`(上の中で) | clean | `e6d5d0ed..14597e36` |
| `./scripts/plan-visual.sh digest <plan>` | `f4a9803ca80e` | 計画の `- Approved:` の行と一致 |
| `git diff --check origin/main...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし(rc=0) | 手元の `origin/main` は `f280f2d6`(base `e6d5d0ed` のあとに PR #348 が入り、`docs/specs/2026-11-19-org-merge-queue.md` だけを足した)。fetch はしていない。その spec は #340 の解いた host を前提にし(NFR-4、`:106`、`:118`)、「`CheckMergeEnforcement` は解いた host を merge.go から受け取る」はこの branch の `merge.go:362` と合う |

1 回目の FAIL(`docs/tech-debt/README.md:254` の `docs/plans/archive/...`)は `9783b9a7` で直った。今の行は `docs/plans/active/2026-11-19-org-ssh-alias-host.md` を引き、`/pr` の `scripts/archive-plan.sh` が archive に書き換える。

## Documentation drift

`/sync-docs` に渡すもの(依頼に従い、ここでは直していない):

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の doctor の行(`README.md:137`) | 一部(D1、1 回目から変わらず) | 「When `ssh -G` does not resolve a host other than `github.com` (it fails, runs out, or prints no hostname that `gh` can take)」の「that `gh` can take」が挙動と合わない。値を断るのは ralph の `mergeSSHHostName`(`mergeorigin.go:231`)で、gh は hostname の文字を確かめない(`mergeSSHHostNameWhy` の doc、`:245-249`)。origin の host が普通の host 名でないときは `ssh -G` を打たずに理由を返す場合(`:174-176`)も括弧の中にない。直し方の例: 「(ssh -G is not run for a host that is not a plain host name, only ASCII letters, digits, `.`, `_`, `:` and `-`, or it fails, runs out, or prints no hostname or one that is not a plain host name)」 |
| register の Trigger(`docs/tech-debt/README.md`) | No(D2、1 回目から変わらず) | この PR が Trigger を満たす開いた行で、「hit, not paid, carried over」の注がないもの。行数は merge-base `e6d5d0ed` と HEAD の `wc -l` で、1 回目のあと変わっていない。251(`internal/org/mergeorder_test.go` 3281 → 3291、「or their tests」)、258(`internal/org/mergeguarantee.go` の編集、877 → 937)、310(同じ編集と `mergeguarantee_test.go` 1334 → 1396。行の頭の 877 と 1334 は古い)、394(`internal/cli/org_merge_test.go` 1243 → 1248)、409(`internal/org/regate.go` 1058 → 1061、`regate_ci.go` 901 → 913、`internal/cli/org_regate_test.go` 1335 → 1339)、419(`internal/cli/doctor.go` 1651 → 1653、comment だけ)。今回の見直しで 1 つ足す: 501 の Trigger「a gate, regate, or merge test of `internal/cli`」に、この PR の `TestOrgMerge_Help` と `TestOrgRegate_Help` の pin の追加が文字どおりには当たる。行の中身は gate・regate・merge を実際に走らせて本物の probe を払うテストの話で、help のテストは probe を走らせないので、払うものはない。当たるかどうかの判断を `/sync-docs` に任せる。411(regate の断りの文)は、この PR が変えたのが中断の error(`regateCIInterrupted` に渡す `mergeOriginReadStep`)で、`refuse` の断りの文ではないので、当たらないと読んだ |
| 計画の Progress(`docs/plans/active/2026-11-19-org-ssh-alias-host.md:134-154`) | No(D4、新規) | `9783b9a7`、`da80502f`、`14597e36` の 3 つが Progress にない。`14597e36` の `t.Skipf` は AC5b の期限の場面の確かめを負荷の高い実行で飛ばす条件つきの緩めで、`.claude/rules/ralph/testing.md` の「Never make tests pass by weakening the intent without stating that change in the plan and review report」に当たる。commit message とテストの comment は理由を書いているが、計画には書いていない。Progress に 3 つの commit と、skip の条件(偽の ssh が 3 秒の期限の前に子の pid を書かなかったとき)と、残る確かめ(呼び出しが戻ることと理由の文)を書く。この report は上の AC5b の注で書いた |
| `docs/evidence/ssh-alias-host-livefire-2026-11-19.md:86-87` | 一部(D5、新規、LOW) | 「fake のテスト(`internal/org/merge_test.go`、`regate_ci_test.go`、`mergeorigin_test.go`)で確かめた」。`da80502f` のあと、merge と regate(ci)の別名のテストは `merge_ssh_alias_test.go` と `regate_ci_ssh_alias_test.go` にある(`merge_test.go` には branch が直した既存のテストが残るので、誤りではなく足りない)。2 つの新しいファイルの名前を足す。self-review report の L6(`docs/reports/self-review-2026-11-19-org-ssh-alias-host.md:45`)も複製の場所の 1 つを `regate_ci_test.go` と書くが、その run の記録なので INFO |
| register の大きさ(1 回目の D3) | 解消 | `da80502f` で branch の新しいテストが新しいファイルに移り、`regate_ci_test.go` は base と同じ 767 行に戻った。`merge_test.go` は 834 行(base で既に 819 行。branch が既存のテストの中で 15 行足した)。309 行の RESOLVED の註「A new merge test goes in a new file when the file it belongs in nears 800 lines」には合う。`merge_test.go` の大きさを追う開いた行はない(INFO。行を足すかは `/sync-docs` の判断) |

1 回目から変わらず合っているもの(本番のコードと文書が変わっていないので、1 回目の判断がそのまま当てはまる):

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の `ralph org merge` の段落、キャッシュの文 | Yes | `merge.go:429` と `ReadMergeGuarantee` に合う |
| org skill の 4 写し(`.claude/skills/org/SKILL.md:2001-2005` ほか 3 つ) | Yes | `check-skill-sync.sh` は ok |
| spec の 7c-2 の決定(`docs/specs/2026-10-07-org-multi-org-director.md:129`) | Yes | 「この割り切りは #340 で改めた」 |
| spec の 8a-3a と #263 の決定(同じ spec の `:144`、`:145`) | Yes(INFO) | 日付のある決定の記録で、別名を解かないとは言っていない |
| register の RESOLVED と打ち消しの行(`:252-253`)、director skill の行(`:254`) | Yes | 254 行の計画のパスは `9783b9a7` で active になった。「about line 772」は `.claude/skills/director/SKILL.md` の行と合う |
| `mergeorigin.go`・`mergeguarantee.go` の doc comment、`MergeEnforcementParams.Host`、`RegateParams.Exec`、`runDoctorOpts` の comment | Yes | |
| `ralph org merge`、`cleanup`、`regate` の `--help` | Yes(INFO) | 解けないときに origin の host のまま進むことは言わない(self-review の L8 (c)、残した) |
| AGENTS.md の repo map | Yes | 計画の D5 のとおり変えていない |
| 計画の Scope の 21〜22 行 | INFO | 解いた値を断る規則が 1 回目の版のまま。D6 と Risks が今の規則を言う。archive の前に合わせると読み違いがない |
| `.claude/skills/pr/SKILL.md:46` | 範囲外 | 計画の Progress が PR の follow-up に回すと書いている |

## Observational checks

- 今回は binary を打っていない。1 回目の観察(HEAD `0a71a930` の binary、別名の origin で `github_protection`、キャッシュの `host` と `api_host`、`ralph status` が `origin_changed` にならない、ssh を PATH から外すと `core.sshCommand` の案内、別名でない 2 つの origin は 8.2.1 と同じ行)は、そのあと本番の Go が変わっていないので今の HEAD にも当てはまる(binary に入る embed の templates も変わっていない)

## Coverage gaps

- テストは走らせていない。AC の「Met」はテストのソースとコードを読んだ判断と、1 回目の binary の観察による。`da80502f` の 2 つの新しいファイルと `14597e36` のテストが通るかは `/test` で確かめる。go vet と golangci-lint は両方のファイルを含めて通った(コンパイルは通る)
- AC5b の期限の場面は、負荷の高い実行では skip になりうる(上の注)。そのときは子が終わることをその実行では確かめない。`/test` の結果に skip が出たら、負荷の低いときに `-run 'TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren'` を単独で打ち直すと確かめられる
- cleanup が `ReadMergeOrigin` に ssh の実行関数として nil を渡すことは、コード(`internal/cli/org_cleanup.go:206`)を読んで確かめた。固定するテストはない
- `Match exec` を書いた本物の ssh の設定での実走、`ralph org merge`・`cleanup`・`regate_checks = "ci"` の regate の実走はしていない(計画の Non-goals)
- `ssh -G` は既定の ssh の設定を読み、`core.sshCommand` や `GIT_SSH_COMMAND` の `-F` は見ない(gh も同じ。self-review の Known gaps)
- 足りない決定的な検査(1 回目と同じ): 文書の slice の検証コマンドが `check_tech_debt_plan_refs` を含まないので、register の計画のパスは `/verify` で初めて落ちた。register を触る slice の検証コマンドを `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` にすれば、slice の時点で見つかる
- 確信を最も上げる追加の確かめ: `/test` で `internal/org` の新しい 2 つのファイルのテストと `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren` が skip なしで通ることを見る
- Insight event: 依頼が「report を単独で commit」なので、1 回目と同じく repo の `docs/insights/events/` には書かず、`--events-dir <scratch>` で作った行をここに写す。verdict が fail から pass に変わったので 1 行を作った。`{"schema":1,"ts":"2026-11-19T15:09:37Z","slug":"org-ssh-alias-host","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC5、AC5b(cleanup はコードと既定の値のテスト。期限の場面は負荷の高い実行で skip がありうる)、AC6、AC7(1 回目に HEAD の binary で打ち直し、そのあと本番の Go は不変)。`da80502f` が移動だけであること。gofmt、go vet、golangci-lint、staticcheck、shellcheck、`check-sync.sh`、`check-skill-sync.sh`、`tech-debt README plan references`、secret scan は通った
- Partially verified: AC8。README の doctor の行の 1 句(D1)が残る。依頼に従い `/sync-docs` に渡す
- Not verified: テストの合否(`/test`)、本物の `Match exec` での実走、merge・cleanup・regate(ci)の実走
- Result: pass。1 回目の FAIL の原因(register の計画のパス)は `9783b9a7` で直り、`./scripts/run-static-verify.sh` は rc=0。`/sync-docs` に D1、D2(251、258、310、394、409、419、判断待ちの 501)、D4(計画の Progress と skip の記録)、D5(evidence のテストのファイル名)を渡す
