# Verify report: org-ssh-alias-host

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`。`./scripts/plan-visual.sh digest` の値 `f4a9803ca80e` は、計画の `- Approved:` の行と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff origin/main...HEAD`(base は merge-base の `e6d5d0ed`、HEAD `0a71a930`、30 ファイル、+2020 −186)。self-review の 2 回目(`a77538ab`)のあとの `3c369ff4`(L8・L10 の直し)、`9bd30e99`(計画)、`0a71a930`(origin/main のマージ)を含む。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-org-ssh-alias-host.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。静的検査の出力、register の検査の試し、HEAD の binary で打った doctor と status の出力を入れた。ラッパーは自分の記録として `docs/evidence/verify-2026-11-19-140349.log` も書いた(これも gitignore に当たる)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 別名の origin で doctor の Merge guarantee が gh を `--hostname github.com` で打ち、種類を決める | Met | doctor の `readOrigin`(`internal/org/mergeguarantee.go:297`)は get-url のあと `resolveMergeAPIHost`(`internal/org/mergeorigin.go:169`)で `apiHost` を決め、gh の呼び出しはすべてその host を `--hostname` に取る。`TestMergeGuarantee_SSHAliasOriginIsReadOnTheHostSSHResolves`(`internal/org/mergeguarantee_alias_test.go:49`)が、scp 形、大文字と port つきの `ssh://`、`ssh.github.com` に解ける別名、大文字の hostname、保護のない別名の 5 件で、種類、`Host`、`APIHost`、Detail、gh の endpoint と `--hostname github.com` を確かめる。HEAD の binary で打った doctor も `github_protection` になった(Observational checks) |
| AC2: merge が使う host(`ReadMergeOrigin` の `-R`、`CheckMergeEnforcement` の `--hostname`)と doctor の host が一致する | Met | 解く関数は 1 つで、呼び手は `ReadMergeOrigin`(`mergeorigin.go:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。merge は `r.apiHost` を `CheckMergeEnforcement` に渡す(`internal/org/merge.go:362`)。`TestMergeGuarantee_DoctorAndMergeReadAnAliasOnTheSameHost`(`mergeguarantee_alias_test.go:250`)が、同じ fake の `ssh -G` の答えで、doctor の `APIHost`、`ReadMergeOrigin` の `Arg`、`CheckMergeEnforcement` の gh の host を、解けたとき(`github.com`)と失敗したとき(別名のまま)の両方で照らす。merge の側は `TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames`(`internal/org/merge_test.go:621`) |
| AC3: 別名でない origin の gh の host は変わらない。https と `git://` では ssh を打たない | Met | `resolveMergeAPIHost` は `mergeOriginIsSSH`(`mergeorigin.go:260`)が偽なら何も打たずに origin の host を返す。表のテスト(`internal/org/mergeorigin_test.go:112-117`、`:153`、`:166-168`)、`TestMergeGuarantee_HostComesFromOriginsURL`(`internal/org/mergeguarantee_test.go:755`。https、`git://`、port と認証情報つきの https は ssh なし、ssh の 5 件は 1 回だけ打って期待値は以前と同じ)、`TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot`(`merge_test.go:846`。`Arg` の期待値は以前と同じ)。HEAD の binary でも、`git@github.com:` と `https://github.com/` の origin の Merge guarantee の行は、evidence に写した 8.2.1 の行と同じだった |
| AC4: キャッシュの `host` は origin に書かれた host、`api_host` は gh に渡した host。`ReadMergeGuarantee` は別名の origin で `origin_changed` にならない | Met | `MergeGuaranteeCheck.APIHost`(`mergeguarantee.go:137`、`omitempty`)。`ReadMergeGuarantee`(`:815`)は今のまま `Host` と origin の host を比べ、ssh を打たない(doc comment もそう言う)。`TestMergeGuarantee_CacheKeepsOriginsHostAndTheHostGHRead`(`mergeguarantee_alias_test.go:182`)が、キャッシュの JSON(version 1、`host` は別名、`api_host` は `github.com`)、読み戻しの値、`--json` の 2 つの key、git 1 回で gh と ssh 0 回、実ホストに向け直した origin の `origin_changed`、`api_host` のない古いキャッシュの読み込みを確かめる。HEAD の binary の `ralph status` も `origin_changed` にならなかった |
| AC5: `ssh -G` の失敗・期限切れ・hostname なし・不正な値で origin の host のまま進み、gh が失敗したら doctor と merge の案内が理由と `git config core.sshCommand` の手を言う | Met | 理由の文は `resolveMergeAPIHost` の 6 つの分岐(`mergeorigin.go:174-209`)、案内の節は `mergeSSHAliasHint`(`:275`)。doctor は `readRepository`(`mergeguarantee.go:357`)、merge は `ghAccess`(`merge.go:398`)と `ghMissing`(`:405`)で足す。表のテストが失敗、未インストール、期限切れ、hostname の行なし、空、`-` で始まる値、空白・タブ・`/`・制御文字・不正な UTF-8・`;`・`$`・引用符・ASCII でない文字を確かめる(`mergeorigin_test.go:118-139`、`:170-176`)。案内は doctor の `TestMergeGuarantee_GHFailureNamesTheSSHKeyFixOnlyForAnUnresolvedAlias`(`mergeguarantee_alias_test.go:107`、8 件)と merge の `TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost`(`merge_test.go:675`、5 件)。HEAD の binary で ssh を PATH から外すと、案内に理由と `core.sshCommand` が出た |
| AC5b: 本物の ssh は `realMergeDetachedExec` で打ち、専用の期限と親の ctx の取り消しの両方で戻り、子プロセスも終わる。本番の 4 つの経路が `realMergeDetachedExec` を渡す | Met(cleanup はコードと既定の値のテスト) | 実プロセスのテストは `TestResolveMergeAPIHost_RealSSHThatHangsEndsWithItsChildren`(`mergeorigin_test.go:410`。PATH の偽の ssh が子を立てて待ち、1 秒の期限と親の取り消しのどちらでも戻り、`kill -0` が ESRCH を返す)。本番の経路: doctor は `newMergeGuaranteeRun`(`mergeguarantee.go:216`)を reflect で照らし、実プロセスのグループも確かめる(`mergeguarantee_alias_test.go:376`)。merge は `newMergeCheck` の既定(`merge.go:202`)を `TestMerge_CallsAfterGHPRMergeRunDetached`(`internal/org/merge_run_test.go:764`)が照らし、`readOrigin` が ssh を `runDetached` で打つことを `TestCheckMerge_ReadOriginRunsSSHThroughRunDetached`(`merge_test.go:651`)が確かめる。regate(ci)は `newRegateCI`(`internal/org/regate_ci.go:62`)を reflect で照らし、実プロセスのグループも確かめる(`internal/org/regate_ci_test.go:778`、`:813`)。cleanup は `org.ReadMergeOrigin(ctx, "", nil, nil, 0)`(`internal/cli/org_cleanup.go:206`)で nil を渡し、`ReadMergeOrigin` の nil の既定が `realMergeDetachedExec` で自分のグループになることは `TestReadMergeOrigin_RunsSSHInAProcessGroupOfItsOwnByDefault`(`mergeorigin_test.go:455`)が確かめる。cleanup が nil を渡すこと自体を固定するテストはない(Coverage gaps) |
| AC6: 別名の origin で `https://github.com/<o>/<r>/pull/<n>` が origin の PR と認められる | Met | `MergeOrigin.hasPR` は解いた `apiHost` で比べる。`TestReadMergeOrigin_ResolvesAnSSHAlias`(`mergeorigin_test.go:309`)が、github.com の PR は真、別名の host の PR は偽、`foreignPRWhy` が空を確かめる。merge(`merge_test.go:621`)と regate(ci)(`regate_ci_test.go:778`、`gate_ci_passed` を記録)も github.com の PR を origin のものとして進む |
| AC7: 実機で、別名の origin の doctor が `--hostname github.com` で読み、unknown にならない | Met | `docs/evidence/ssh-alias-host-livefire-2026-11-19.md`(binary `7798b966`)。この binary は `55cdb7c3` と `3c369ff4` より前なので、HEAD `0a71a930` の binary を作って同じ 4 つの場面を打ち直した。別名の origin は `github_protection` と別名の注記、キャッシュは `host` が別名で `api_host` が `github.com`、ssh なしは理由と `core.sshCommand`、別名でない 2 つは evidence と同じ行で、どれも evidence の行と path を除いて同じだった。AC は「一時 clone」と言うが、evidence も打ち直しも `git init` と `git remote add origin` で作った repo を使った(evidence はそう書いている。doctor が読むのは origin の URL と GitHub 側なので、判定は変わらない) |
| AC8: 文書が新しい挙動と合う。`check-skill-sync.sh` と `check-sync.sh` が通る | 一部 | 2 つの検査は通った(Static analysis)。org skill の 4 写し、spec の 7c-2、register の RESOLVED と新しい行、doc comment、`--help` は挙動と合う。README の doctor の行の 1 句が `3c369ff4` のあとの挙動と合わない(Documentation drift の D1)。register の新しい行は内容は正しいが、計画のパスが静的検査を落とす(Static analysis の FAIL) |

計画の Verify plan が挙げる「gh の解き方と同じ順序」も確かめた。go-gh の `pkg/ssh/ssh.go` を GitHub の main で読んだ(`gh api repos/cli/go-gh/contents/pkg/ssh/ssh.go`)。`Translate` は scheme が ssh のときだけ `ssh -G <host>` を打ち、`SplitN(line, " ", 2)` の最初の語が `hostname` の行のうち最後の値を取り、失敗か空なら元の host を使い、そのあと `strings.EqualFold(resolved, "ssh.github.com")` を `github.com` に読み替える。ralph の順序(解く、小文字にする、`mergeGuaranteeAPIHost`)はこれと同じ。違いは 3 つで、どれも計画に書いてある。ralph は値の前後の空白を落とす(CRLF の行のため)、小文字にしてから比べる(`mergeGuaranteeAPIHost` が完全一致で比べるため)、普通の host 名でない値を使わない(D6)。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=1 | 落ちたのは `tech-debt README plan references` だけ。`docs/tech-debt/README.md references plans that do not exist at that path: docs/plans/archive/2026-11-19-org-ssh-alias-host.md`。ほかは通った。gofmt は `ok`、`go vet ./...` と staticcheck は出力なし、golangci-lint は `0 issues.`、shellcheck、`sh -n`、guard の awk の解析、settings の `jq -e`、Codex の hook の 3 つの guard は OK |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh` | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh` | OK | |
| `secret-scan-branch.sh` | clean | `e6d5d0ed..0a71a930` |
| register の検査の試し | 足りない参照 0 件 | scratch に写した register で、そのパスを `docs/plans/active/` に変え(1 行)、`check_tech_debt_plan_refs`(`scripts/verify.local.sh:141`)と同じ grep と `-e` の loop を worktree の root で回した |
| `./scripts/plan-visual.sh digest <plan>` | `f4a9803ca80e` | 計画の `- Approved:` の行と一致 |
| `git diff --check origin/main...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | 手元の `origin/main` は `e6d5d0ed`(base と同じ)。fetch はしていない |

### FAIL の原因と直し

- 原因: slice 3(`44a56862`)が足した register の director skill の行(`docs/tech-debt/README.md:254`)は、最後の欄で計画を `docs/plans/archive/2026-11-19-org-ssh-alias-host.md` と引く。計画はまだ `docs/plans/active/` にあるので、`check_tech_debt_plan_refs` が落ちる。register の行は PR のあいだ `docs/plans/active/` を引き、`/pr` の `scripts/archive-plan.sh` が archive に書き換える決まりで、doctor-alias-herdr-guard も同じ形で run 1 が落ち、`b38e97ba` で active に戻した。slice 3 の検証は `check-skill-sync.sh` と `check-sync.sh` だけだったので、この検査はここで初めて走った
- 直し: 254 行の `docs/plans/archive/2026-11-19-org-ssh-alias-host.md` を `docs/plans/active/2026-11-19-org-ssh-alias-host.md` にする(1 語)。上の試しのとおり、それで検査は通る。register の中で archive のこの計画を引くのはこの 1 か所だけ(`grep -o` で 1 件)。Go のソースとテストの comment(`internal/org/mergeorigin.go:18`、`mergeguarantee.go:39` など、`internal/` の中に 12 か所)は active のパスを引き、どの検査も読まない。`/pr` のあとはそれらが古いパスを指すが、register 以外を書き換えないのは `archive-plan.sh` の前例どおり
- 直したあと `./scripts/run-static-verify.sh` を打ち直す。register の 1 行だけの直しなので、ほかの行の結果は変わらないはず(未確認)

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の doctor の行(`README.md:137`) | 一部(D1) | D1: 「When `ssh -G` does not resolve a host other than `github.com` (it fails, runs out, or prints no hostname that `gh` can take)」の「that `gh` can take」が `3c369ff4` のあとの挙動と合わない。値を断るのは ralph の `mergeSSHHostName`(`mergeorigin.go:231`)で、gh は hostname の文字を確かめない(同じ関数の doc と `mergeSSHHostNameWhy` の doc が言うとおり)。また、origin の host が普通の host 名でないときは `ssh -G` を打たずに理由を返す場合(`mergeorigin.go:174-176`)が括弧の中にない。`/sync-docs` で、たとえば「(ssh -G is not run for a host that is not a plain host name, only ASCII letters, digits, `.`, `_`, `:` and `-`, or it fails, runs out, or prints no hostname or one that is not a plain host name)」のように直す。依頼の申し送り(136 行と書かれていたもの。#346 の pack の行が入って 137 行になった)と同じ箇所 |
| README の `ralph org merge` の段落(`README.md:296`) | Yes | ssh の origin の host を doctor と同じく `ssh -G` で解くと書く。コード(`merge.go:429`)と合う |
| README のキャッシュの文(`host` と `api_host`) | Yes | `ralph status` と merge-order が origin の host を ssh も gh も打たずに比べる、は `ReadMergeGuarantee` と合う |
| org skill の 4 写し(`.claude/skills/org/SKILL.md:2000-2005` ほか 3 つ) | Yes | 4 つの段落は同じ文(行の範囲を比べて一致、`check-skill-sync.sh` も ok)。doctor と merge が `ssh -G` で実ホストを解くこと、解けずに gh が失敗したら行の理由のとおり origin を実ホストに向けて `core.sshCommand` で鍵を選ぶことは、コードと合う |
| spec の 7c-2 の決定(`docs/specs/2026-10-07-org-multi-org-director.md:129`) | Yes | 「別名は解かず」の文のあとに「この割り切りは #340 で改めた」と、`ssh -G`、`host`、`api_host` を足した |
| spec の 8a-3a と #263 の決定(同じ spec の `:144`、`:145`) | Yes(INFO) | どちらも gh を「origin の URL から読んだ host」に固定すると書く。日付のある決定の記録で、別名を解かないとは言っていない。7c-2 の注記に合わせて #340 の一言を足すかは任意 |
| register の RESOLVED と打ち消しの行(`docs/tech-debt/README.md:252-253`) | Yes | 4 つの経路、最後の `hostname` の行、`ssh.github.com` の読み替え、`host` と `api_host`、`mergeSSHAliasHint` の案内はコードと合う。引く SHA(`554d04aa`、`40afa91e`)は branch にある |
| register の director skill の行(`docs/tech-debt/README.md:254`) | 内容は Yes | 「about line 772」は `.claude/skills/director/SKILL.md:772` の grep の行と合う。計画のパスは上の FAIL |
| register の Trigger | No(D2、D3) | D2: この PR が Trigger を満たす開いた行が 6 つあり、どれにも「hit, not paid, carried over」の注がない。251(`internal/org/mergeorder_test.go` の `moExec` に ssh の答えを足した、3281 → 3291 行、「their tests」に当たる)、258(`internal/org/mergeguarantee.go` の編集、877 → 937 行)、310(同じ編集と `mergeguarantee_test.go` の 1334 → 1396 行。行の数え方の 877 と 1334 は古い)、394(`internal/cli/org_merge_test.go` 1243 → 1248)、409(`internal/org/regate.go` 1058 → 1061、`regate_ci.go` 901 → 913、`internal/cli/org_regate_test.go` 1335 → 1339)、419(`internal/cli/doctor.go` 1651 → 1653、comment だけ)。行数は merge-base `e6d5d0ed` と HEAD の `wc -l`。どれも move だけの直しが Scope の外なので、払わずに注を足すのが前例。D3: `internal/org/regate_ci_test.go` がこの PR で 800 行を越え(767 → 833)、`internal/org/merge_test.go` は base で既に 819 行だったところに 3 つの merge のテストが入って 940 行になった。register の 309 行の RESOLVED の註は「A new merge test goes in a new file when the file it belongs in nears 800 lines」と言い、2 つのファイルの大きさを追う開いた行はない。新しい行を足すか、`/sync-docs` の判断で注に書く |
| `internal/org/mergeorigin.go` と `mergeguarantee.go` の doc comment | Yes | `MergeOrigin`、`ReadMergeOrigin`、`resolveMergeAPIHost`、`mergeSSHHostName`、`MergeGuaranteeCheck.Host` と `APIHost`、`CheckMergeGuarantee` の手順 1、C7 の段落、`readOrigin`、`ReadMergeGuarantee` はコードと合う。`APIHost` の comment の「It is another than Host」は英語として座りが悪いが、意味のずれはない |
| `MergeEnforcementParams.Host`(`internal/org/mergeenforce.go`) | Yes | merge が `r.apiHost` を渡し、`CheckMergeEnforcement` は ssh を打たない、はコードと合う |
| `ralph org merge`、`cleanup`、`regate` の `--help` | Yes(INFO) | HEAD の binary の `--help` に ssh -G の文が出る。3 つとも「host は ssh -G が出す hostname」と言い、解けないときに origin の host のまま進むことは言わない(self-review の L8 (c)、残した)。要約と端の場合の差で、誤りではない |
| `runDoctorOpts` の comment(`internal/cli/doctor.go:84-87`) | Yes | 20 秒の期限の中で、普通の host 名の ssh の origin にだけ、自分のプロセスグループで 5 秒まで `ssh -G` を打つ、はコードと合う |
| AGENTS.md の repo map | Yes | 計画の D5 のとおり変えていない。`mergeorigin.go` の `ReadMergeOrigin` を cleanup と merge が使う、という記述は今も正しい |
| 計画の Scope の 21〜22 行 | INFO | 解いた値を断る規則が 1 回目の版(空、`-`、空白・制御文字・`/`)のまま。D6 と Risks が今の規則を言い、ずれとして記録されているので、矛盾は記録の上で説明がつく。archive に残る前に合わせると読み違いがない(self-review の申し送りと同じ) |
| `.claude/skills/pr/SKILL.md:46` の「SSH host aliases that `gh` cannot resolve」 | 範囲外 | 計画の Progress が PR の follow-up に回すと書いている |

## Observational checks

- HEAD `0a71a930` の binary(`go build -o <scratch>/ralph ./cmd/ralph`)を、`git init` した scratch の repo で打った。台帳は `--state-dir` で scratch に向けた。gh の呼び出しは github.com の読み取りだけ
  - origin `ssh://git@github.com.emu/yoshpy-dev/ralph.git`: `✓ Merge guarantee: pass — github_protection: ... (origin's host github.com.emu is an ssh alias of github.com, which ssh -G resolved and gh reads)`。キャッシュは `kind github_protection`、`host github.com.emu`、`api_host github.com`、`version 1`
  - 同じキャッシュで `ralph status`: `merge guarantee: github_protection (main of github.com.emu/yoshpy-dev/ralph, ...)`。`origin_changed` にならない
  - `PATH=/opt/homebrew/bin:/bin`(ssh なし): `ℹ Merge guarantee: info — unknown: gh api --hostname github.com.emu ... ; ssh -G github.com.emu failed (exec: "ssh": executable file not found in $PATH), so gh read the host github.com.emu as origin names it: ... git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'`
  - `git@github.com:yoshpy-dev/ralph.git` と `https://github.com/yoshpy-dev/ralph.git`: どちらも `✓ Merge guarantee: pass — github_protection: ...` で、別名の注記なし。evidence の 8.2.1 と同じ行
  - 4 つとも、AC7 の evidence(`7798b966`)の行と、scratch の path と `checked_at` を除いて同じ。`55cdb7c3` と `3c369ff4` が変えたのは、普通の host 名でない host と hostname の扱いと理由の文で、この 4 つの場面は通らない(コードを読んだ判断と、打ち直しの結果が一致した)
- 上のほかに、手で 1 回 `ssh -G github.com.emu` を打って `hostname github.com` を確かめた(接続はしない)

## Coverage gaps

- テストは走らせていない。上の AC の「Met」はテストのソースとコードを読んだ判断と、HEAD の binary の観察による。テストが通るかは `/test` で確かめる
- cleanup が `ReadMergeOrigin` に ssh の実行関数として nil を渡すことは、コード(`internal/cli/org_cleanup.go:206`)を読んで確かめた。`internal/cli` に、cleanup の本番の経路の ssh の実行関数を固定するテストはない。nil の既定が `realMergeDetachedExec` になることは org のテストが確かめる
- `Match exec` を書いた本物の ssh の設定での実走はしていない。期限切れでプロセスグループごと止まることは、偽の ssh の実プロセスのテストだけが確かめる
- `ralph org merge`、`cleanup`、`regate_checks = "ci"` の regate の実走はしていない(計画の Non-goals)
- `ssh -G` は既定の ssh の設定を読み、`core.sshCommand` や `GIT_SSH_COMMAND` の `-F` は見ない(gh も同じ。self-review の Known gaps)
- 足りない決定的な検査: 文書の slice の検証コマンドが `check_tech_debt_plan_refs` を含まないので、register の計画のパスは `/verify` で初めて落ちる(doctor-alias-herdr-guard に続いて 2 回目)。`scripts/verify.local.sh` は `HARNESS_VERIFY_MODE=static` で走らせると、`check-sync.sh`、`check-skill-sync.sh`、この検査をまとめて走らせる(`:22`、`:317`。コードを読んだ判断で、単独では打っていない)。register を触る slice の検証コマンドを `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` にすれば、次から slice の時点で見つかる
- 確信を最も上げる追加の確かめ: 254 行の 1 語を直したあと `./scripts/run-static-verify.sh` を打ち直し、rc=0 を見ること
- Insight event: 依頼が「report だけを commit」なので、repo の `docs/insights/events/` には書かず、`--events-dir <scratch>` で作った行をここに写す(self-review も同じ理由で書いていない)。`{"schema":1,"ts":"2026-11-19T14:13:10Z","slug":"org-ssh-alias-host","flow":"standard","phase":"verify","cycle":1,"verdict":"fail","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Verdict

- Verified: AC1、AC2、AC3、AC4、AC5、AC5b(cleanup はコードと既定の値のテスト)、AC6、AC7(HEAD の binary で打ち直した)。go-gh の `Translator` と同じ順序で解くこと。gofmt、go vet、golangci-lint、staticcheck、shellcheck、`check-sync.sh`、`check-skill-sync.sh`、secret scan は通った
- Partially verified: AC8。README の doctor の行の 1 句が `3c369ff4` のあとの挙動と合わない(D1)。register の Trigger の注と大きさの行が足りない(D2、D3)
- Not verified: テストの合否(`/test`)、本物の `Match exec` での実走、merge・cleanup・regate(ci)の実走
- Result: fail。原因は静的検査の `tech-debt README plan references` の 1 件だけで、register の 254 行の計画のパスを `docs/plans/active/` に戻す 1 語の直しで通る(試しで確認済み)。直したあと `/verify` を打ち直す。D1〜D3 は `/sync-docs` に渡す
