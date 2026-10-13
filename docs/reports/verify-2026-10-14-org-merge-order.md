# Verify report: org-merge-order

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md(`Status: Approved`、`- Approved:` の digest は c0b1e9230999)
- Verifier: verifier subagent (Claude)。パイプラインの 1 回目(`.harness/state/standard-pipeline/cycle-count.json` は 1)。self-review(346304fc、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 8、Merge: yes)のあとに走らせた。self-review が読んだ HEAD 06462bcd から HEAD 346304fc までに増えたのは self-review の報告だけで、コードと文書は変わっていない
- Scope: `git diff ac34ccf3...HEAD`(HEAD 346304fc、7 コミット、17 ファイル、+3618/-66)。仕様適合、静的解析、文書のずれ。テストは走らせていない(/test の担当)。行番号はすべて HEAD 346304fc のもの
- Evidence:
  - `docs/evidence/verify-2026-10-14-org-merge-order-cli.md`: HEAD から作った binary を、一時の bare origin と repo、種をまいた台帳で打った出力(R1〜R13)。PATH は git の wrapper だけ(と偽の `gh`)の一時のディレクトリで、本物の `gh` には届かない。中身は生の出力と `cmp` で一致することを確かめた(手で書いたのは先頭の説明と、最後の節の 1 段落だけ)
  - `docs/evidence/verify-2026-10-13-201008.log`: `./scripts/run-static-verify.sh` が自分で書いた生の出力(名前の日付は UTC)。gitignore の対象なのでコミットしない
  - insight event: `docs/insights/events/2026-10-13-org-merge-order.jsonl` に verify の 1 行(`--cycle auto`、verdict pass、low 2 は D1 と D2)を足した。lead の指示どおり、このファイルはコミットに入れていない(self-review の行と同じく未追跡のまま)

## Spec compliance

判定の言葉は次のとおり。「満たす」はコードを読み、AC の各項目の経路と、それを固めるテストの名前を確かめたもの。テストが通ることは /test が確かめる。「満たす(実機でも確認)」は、加えて HEAD の binary を一時の台帳と repo で打って確かめたもの(R の番号は evidence の節)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 head と base が origin と一致すれば `ready`、main が進めば `rebase`、gate のあとの push は `regate` | 満たす(実機でも確認) | `classifyGatePassed`(`internal/org/mergeorder.go:586-610`)。テスト: `TestMergeOrder_ClassifiesAGatePassedAgainstOrigin`(`internal/org/mergeorder_test.go:305`)、`TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes`(`internal/cli/org_merge_order_test.go:421`)。R1 で feat/beta と feat/eta が ready、M0 で gate を通した feat/alpha が rebase、gate のあとに push した feat/gamma が regate |
| AC2 `failed`(reason つき)、`no_gate`、`no_remote`。通ったあとに落ちたら最新の記録を読む | 満たす(実機でも確認) | `classifyMergeTarget`(`mergeorder.go:556-580`)、最新の記録で上書きする `collectMergeTargets`(`:289-338`)。テスト: `TestMergeOrder_FailedNoGateAndNoRemote`(`mergeorder_test.go:418`)、`TestMergeOrder_LatestGateRecordWins`(`:454`)。R1 で feat/zeta(同じ head で gate_passed のあとに gate_failed)が `failed`(test_failed)、push していない feat/delta が no_remote、gate のない feat/epsilon と feat/alpha2 が no_gate |
| AC3 disband 済みの org のブランチも対象。同じ org_id を別の機能で立て直しても前のブランチが残る | 満たす(実機でも確認) | `collectMergeTargets` は `disbanded` を見ずに、完全な結びつきをすべて対象にする(`mergeorder.go:301-314`)。テスト: `TestMergeOrder_TargetsAreTheBranchesOfEveryBinding`(`mergeorder_test.go:551`)、`TestMergeOrder_SameBranchBoundTwiceShowsTheLatestBinding`(`:589`)。R1 で、disband したあと feat/alpha2 に結び直した org-a の feat/alpha が `org-a/alpha` のまま rebase で残る |
| AC4 dry-run と完全でない結びつきを数えない。読めない記録と、そのあとの壊れた行は `unreadable` で、前の pass に戻らない。読めない記録はその時点の結びつきに付く | 満たす(実機でも確認) | 絞り込み(`mergeorder.go:302`)、event 名と reason の突き合わせ(`:318`)、結びつきへの付け先(`:320-327`)、壊れた行の位置(`:331-336`)。テスト: `TestMergeOrder_DryRunSeatAndIncompleteRecordsDoNotCount`(`mergeorder_test.go:611`)、`TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt`(`:518`)、`TestMergeOrder_LatestGateRecordWins`(`:454`)、`TestMergeOrder_TrailingLineWithoutNewlineIsCorrupt`(`:1013`)。R1 で dry-run の gate_passed を持つ feat/epsilon は no_gate のまま、digest のない結びつきの feat/incomplete は出ない。R10 で branch= のない org-b の記録が feat/beta に付いて unreadable。R11 で改行のない最後の行のあと、gate の記録を持つ 6 本がすべて unreadable になり、注記が壊れた行の数を言う。event 名と reason の不一致は CLI では打っていない(unit test だけ) |
| AC5 `gh` で MERGED / CLOSED を既定で外し `--all` で出す。同じ repo の最大番号の PR。`gh` がない・失敗するときは欄を空にして注記し、終了コード 0 | 満たす(実機でも確認) | `lookupMergePRs`(`mergeorder.go:449-470`)、`pickMergePR`(`:496-507`)、`orderMergeEntries` の隠し(`:513-537`)。テスト: `TestMergeOrder_PicksTheHighestSameRepoPR`(`mergeorder_test.go:722`)、`TestMergeOrder_GHMissingNotesAndIsNotRunAgain`(`:746`)、`TestMergeOrder_GHFailureForOneBranchLeavesItsPROut`(`:778`)、`TestOrgMergeOrder_AllListsMergedAndClosedPRsTheDefaultLeavesOut`(`org_merge_order_test.go:538`)。R1(`gh` なし)の注記、R3 で fork の #12 より番号の小さい #11 を採り、merged と closed の 2 本を隠して数を注記、R4(`--all`)で末尾に merged・closed、R13(全ブランチで `gh` が exit 4)は PR の欄が空で終了コード 0 |
| AC6 並びの順、`ready` の中は gate の時刻順、`ready` を 1 本 merge するとほかが `rebase` になる注記 | 満たす(実機でも確認) | `mergeClassOrder`(`mergeorder.go:69-72`)、`orderMergeEntries`(`:513-537`)、注記(`:632-650`、`ready` が 2 本以上のときだけ出す。`:643`)。テスト: `TestMergeOrder_OrdersByClassThenGatePosition`(`mergeorder_test.go:643`)、`TestMergeOrder_MergedOrClosedComesFirst`(`:698`)。R1 は ready(00:11 の beta、00:12 の eta)、rebase、regate、failed、no_remote、no_gate の順で、2 本の ready の注記がある。ready が 1 本の R3 には注記がない。2 本以上の条件は skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`)に書いてあり、`--help` は条件なしで同じことを言う |
| AC7 ls-remote が打てない・HEAD の symref がないときは何も分類せず終了コード 1。manifest が読めないときも 1 | 満たす(timeout のほかは実機でも確認) | `listOriginBranches`(`mergeorder.go:368-405`)、`MergeOrder` の manifest の読み(`:191-194`)。テスト: `TestMergeOrder_LsRemoteProblemsClassifyNothing`(`mergeorder_test.go:865`)、`TestMergeOrder_ManifestThatCannotBeReadIsAnError`(`:1029`)、`TestOrgMergeOrder_OriginStateUnknownExitsOneWithTheReason`(`org_merge_order_test.go:742`)。R5(origin のない repo)、R6(repo の外)、R7(origin の HEAD が無いブランチを指す: ローカルの転送は HEAD を出さないので「printed no HEAD symref」)、R12(manifest が chmod 000)は、どれも stdout が空で終了コード 1。30 秒の timeout は打っていない |
| AC11 既定のブランチを trunk に変えると main で通したブランチは `rebase`。PR の統合先が違えば `wrong_base` | 満たす(実機でも確認) | 既定は HEAD の symref から読む(`mergeorder.go:388-403`)、`gateBaseBranchName` と比べる(`:599-603`)、PR の統合先(`:595-597`)。テスト: `TestMergeOrder_DefaultBranchComesFromOriginsHEAD`(`mergeorder_test.go:401`)、`TestOrgMergeOrder_GateOnTheOldDefaultBranchIsRebaseAfterOriginHEADMoves`(`org_merge_order_test.go:648`)、`TestOrgMergeOrder_PRIntoAnotherBranchThanTheDefaultIsWrongBase`(`:624`)。R8 で、main を残したまま origin の HEAD を trunk(main と同じ先頭)にすると、ready だった beta と eta が「the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk」の rebase になる。R3 で統合先 release の PR #13 を持つ feat/eta が wrong_base |
| AC8 `--json` は分類・PR・gate・リモートの先頭・結びつき・注記を、テキストと同じ順で出す | 満たす(PR の欄のほかは実機でも確認) | `MergeOrderResult` をそのまま encode する(`internal/cli/org_merge_order.go:109-111`)。テスト: `TestMergeOrder_ResultJSONShape`(`mergeorder_test.go:1102`)、`TestOrgMergeOrder_JSONListsTheSameBranchesInTheSameOrderAsTheText`(`org_merge_order_test.go:464`)。R2 は R1 と同じ 8 本を同じ順で出し、gate の記録(event、readable、head、base、base_ref、reason、ts)、remote_head、binding、notes を持つ。R2 は `gh` なしなので `pr` の欄は出ていない |
| AC9 台帳に何も書かない。ロックを取らず、見張りなしで打てる | 満たす(実機でも確認) | `mergeorder.go` と CLI の経路(`guardLegacyOrgStateDir`、`newOrgRuntimeAt`、`MergeOrder`)にロック、書き込み、`checkWatchGate` の呼び出しはない(grep と読みで確認。見張りの関門を呼ぶのは `feature.go`・`spawn.go`・`verbs.go` だけ)。テスト: `TestMergeOrder_WritesNothing`(`mergeorder_test.go:1044`)、`TestOrgMergeOrder_WritesNothingAndTakesNoLock`(`org_merge_order_test.go:958`)。evidence で、manifest の sha256・行数・バイト数・ディレクトリの中身が全実行の前後で同じ(`manifest.lock` も `watch-alive.json` もできない)、work repo の refs の digest が R1〜R4 の前後で同じ(fetch しない)、R1〜R4 のあと state・work・origin・HOME の下に marker より新しいファイルが 0 件。台帳には見張りの記録がないまま打てた |
| AC10 skill の 4 面、README、仕様の注、AGENTS.md、tech-debt が説明し、skill の 4 面が同じ | 満たす(README の 1 文に LOW のずれ D1) | 4 面は `cmp` で同一、`scripts/check-skill-sync.sh`(13 skill in lock-step)と `scripts/check-sync.sh`(DRIFTED 0)が通る。内容は下の Documentation drift の表 |

### 仕様 FR-7・FR-8 との対応

- FR-8 の「merge してよい順(検証した base が今の main と一致する PR)」: 「検証した base」は台帳の gate の記録の base(AC1、AC2、AC4)、「今の main」は ls-remote で読む origin の HEAD の symref とその先頭(AC1、AC11)、分からなければ推測しない(AC7)。対象(AC3)、PR の状態(AC5)、順(AC6)、director が読む形(AC8)、読むだけであること(AC9)がそれを支える。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`、7c-1 で決めたこと)はこの読み方と、分類の順、`ready` の定義を書いており、コードと合う
- FR-8 の `auto_merge` と director の報告は 8a に、FR-7 の `ralph doctor` の保護ブランチの warn と `ralph status` の保証の表示は 7c-2 に送った。計画の Non-goals、仕様の注の最後の文、tech-debt の 226 行目(Trigger が「Stage 7c-2」に変わった)が同じことを言う
- 7b の B2(ralph は reviewer の RESULT の `HEAD:`・`BASE:` を読まない、`docs/plans/archive/2026-10-13-org-gate.md:87`、仕様 73 行目): merge-order が読むのは `readMergeLedger`(`mergeorder.go:236-248`)の manifest の行だけで、受信箱も agmsg も読まない(`inbox`・`agmsg`・`HEAD:` を grep して、当たるのはコメントの 1 か所だけ)
- 計画の Progress の決定(disband 済みの機能の org では gate を打ち直せない穴を、直さずに文書に書く。持ち主は 8a): skill の「merge してよい順」節(`.claude/skills/org/SKILL.md:871` 以下)、README 262 行目の最後の文、仕様の注、tech-debt の 233 行目、`Detail` の「(the gate runs only while the feature org runs)」に書かれている。`gateOrgLife`(`internal/org/gate.go:333`)が disband 済みの org を断ることは読んで確かめた

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier(shellcheck、hook の `sh -n` 20 本、guard の awk、`jq -e` 2 本、Codex の guard 3 本、`check-sync.sh` PASS(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照)、golang(`gofmt: ok`、`go vet ./...` 出力なし、`golangci-lint run ./...` 0 issues、`staticcheck ./...` 出力なし)、`secret-scan-branch: scanned ac34ccf3..346304fc against origin/main: clean`。生の出力は `docs/evidence/verify-2026-10-13-201008.log` |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod は `go 1.25.8`、手元は go1.26.0。`strings.SplitSeq`(1.24)など新しい API も 1.25.8 で build できる |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/... ./internal/cli/...` | pass | 出力なし |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `go vet ./...`(go1.26.0) | pass | 出力なし |
| `scripts/check-skill-sync.sh`、`scripts/check-sync.sh`(単独で) | pass | 13 skill(s) in lock-step。IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0 |
| `cmp` で `/org` skill の 4 面 | pass | `.claude/`、`.agents/`、`templates/base/.claude/`、`templates/base/.agents/` が同一 |
| `git diff --check ac34ccf3...HEAD` | pass | rc 0 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`) | yes | 分類の上から順(merged・closed、unreadable、no_gate、failed、no_remote、regate、wrong_base、rebase、ready)、並びの順、30 秒と 15 秒、`GIT_TERMINAL_PROMPT=0`、fork でない最大番号の PR、`ready` 2 本以上の注記、終了コード、対象がなければ git を打たない、`--org-id` を断る、が `mergeorder.go` と `org_merge_order.go` に合う。R1〜R13 の出力とも合う |
| skill の「merge してよい順」節(`:871` 以下)と gate の節の 2 項目、見張りなしで使える動詞の一覧、古い台帳の動詞の一覧 | yes | 一覧に足した `approve`・`gate` は、見張りの関門を呼ばないこと(`checkWatchGate` を呼ぶのは start・spawn・send だけ)と、`org_approve.go`・`org_gate.go` が `orgLedgerMutating` を使うことに合う |
| README の動詞の表(`README.md:124`) | yes | `merge-order` を足し、1 文で説明する |
| README の org の節(`README.md:262`) | 一部ずれ(D1、LOW) | 「Each branch gets the first class that applies, and the list is in this order: `ready`, `rebase`, ...」は並びの順だけを挙げ、どの順で「最初に当てはまる」かを言わない。並びの順で読むと、コードの判定と食い違う。R1 で push していない gate なしの feat/epsilon は no_gate(並びでは no_remote が先)、R3 で origin にない merged の feat/delta は merged(並びでは no_remote が先)、regate は wrong_base より先に判定する(`mergeorder.go:592-597`)。skill と `--help` は判定の順と並びの順を分けて書いている。/sync-docs で、判定の順を足すか「first class that applies (checked from merged/closed down to ready)」のように直す |
| AGENTS.md の repo map | yes | `internal/org/` に `mergeorder.go` を足した。`internal/cli/` の行は `org` でまとめており、新しいファイル名を挙げない既存の書き方と同じ |
| 仕様 FR-8 の 7c-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`) | yes | 上の「仕様 FR-7・FR-8 との対応」のとおり |
| `ralph org --help`、`ralph org merge-order --help` | yes | binary で確かめた。`merge-order` がサブコマンドの一覧にあり、`--org-id` の説明に `merge-order` が入る。`--all` と `--json` の説明がある |
| `internal/org/seat.go` の package doc、`internal/cli/org_legacy_ledger.go` のコメント | yes | `git ls-remote` と `gh` を `mergeorder.go` が打つこと、`merge-order` が読むだけの動詞であることを足した |
| `internal/org/gate.go:27`、`:164` のコメント(「Stage 7c's merge order reads that record」) | yes | 7c-1 が読むので今も正しい |
| tech-debt の追加 4 行(`docs/tech-debt/README.md:233-236`)と変更 1 行(`:226`) | yes | 挙げた名前はすべて今のコードにある: `gateOrgLife`(`gate.go:333`)、`StartFeature`(`internal/org/feature.go:181`)、leader の手順 12〜16(`internal/org/prompts/leader.md:207-264`)、`classifyGatePassed`、`listOriginBranches`、`realMergeOrderExec`、`defaultMergeOrderLsRemoteTimeout`、`mergeOrderWaitDelay`(2 秒)、`lookupMergePRs`、`defaultMergeOrderGHTimeout`、`readMergeLedger`、`readJSONLines`、`max_orgs` の既定 10(`internal/config/config.go:209`)。236 行目の「途中まで書かれた最後の行で、gate の記録を持つブランチがすべて unreadable になる」は R11 で再現した |
| tech-debt の 231 行目(org-gate cycle 2 の小さな所見)の Trigger | 発火、未記録(D2、LOW) | Trigger に「and stage 7c, which reuses these files」とある。この PR は 7c-1 で、`gate.go` の `gateRecordFromDetails` と `gitObjectPattern` を読むが、`gate.go`・`gate_run.go`・`lockfile.go` は変えていない。項目 (a)〜(i) はどれも merge-order の動きに関わらない。/sync-docs で「carried over by feat/org-merge-order(7c-1 は gate.go を読むだけ、変えていない)」を足す |
| tech-debt の 229 行目(gate が feature ブランチのスクリプトを走らせる)の Trigger | 発火していない(情報) | 「stage 7c's merge order wanting a stronger guarantee」は条件つき。merge-order は gate の記録だけを読み、保証を強めていない |
| 計画の AC の checkbox と Progress の「Review artifact created」 | 未更新(情報) | AC1〜AC11 はどれも `[ ]`、self-review の報告があるのに「Review artifact created」も `[ ]`。/pr の前に更新されればよい |
| Go のコメントが `docs/plans/active/2026-10-14-org-merge-order.md` を指す(self-review の L8) | 情報 | `scripts/archive-plan.sh:54` が書き換えるのは `docs/tech-debt/README.md` の参照だけ。tech-debt の 4 行の `active/` は /pr で書き換わる。Go のコメントは古くなるが、既存の `approval.go` などと同じ扱い |

## Observational checks

- HEAD の binary を一時の repo で打った出力は `docs/evidence/verify-2026-10-14-org-merge-order-cli.md` にある。テキストの表は tab 区切りで、列は `RANK CLASS BRANCH PR GATE_HEAD GATE_BASE ORG DETAIL`。SHA は 12 桁で出る
- R6b(観察): repo の外で、そのディレクトリに `origin.git` があると、git は設定されていない名前 `origin` をローカルのパス `./origin.git` として読み、merge-order はその repo の分類を出して終了コード 0 になった。git 自身の名前の解決で、merge-order のコードの問題ではない。origin のある repo では設定した remote が先に使われる
- R12(既存の文言): manifest が読めないときの stderr は `read manifest: org: open <path>: open <path>: permission denied` で、パスが 2 回出る。`readJSONLines` の包み方(`internal/org/manifest.go:309`、この PR より前の f466e181)によるもので、この差分の外
- R13 は self-review の L2 を実機で示す。`gh` が未認証のように全ブランチで失敗すると、同じ理由の注記が 8 本並ぶ。JSON の側では PR の lookup の失敗と PR がないことを entry で見分けられない(L2 のとおり)。self-review は、直さないなら L2 と L4 を 1 行で tech-debt に足すよう勧めており、まだ足されていない(orchestrator か /sync-docs の判断)

## Coverage gaps

- テストは走らせていない(/test の担当)。上の「満たす」は、経路を読み、テストの名前と中身を確かめたもの
- ls-remote の 30 秒の timeout と、ssh の passphrase で止まる動きは打っていない。timeout は `TestMergeOrder_LsRemoteProblemsClassifyNothing`(`mergeorder_test.go:865`)が偽の seam で固める。ssh の件は tech-debt の 234 行目
- 本物の GitHub は使っていない(ネットワークはローカルの bare repo だけ)。github.com の ls-remote が `HEAD` の symref を返すこと(計画の Assumptions)と、本物の `gh pr list --json` が 6 つの欄名(`number`・`state`・`url`・`headRefOid`・`baseRefName`・`isCrossRepository`)を返すことは、おそらく正しいが未確認
- 別のプロセスがロックを握っている台帳での実行は打っていない。`TestOrgMergeOrder_WritesNothingAndTakesNoLock`(`org_merge_order_test.go:958`)が受け持ち、静的にはロックの呼び出しがないことだけを確かめた
- `--json` の `pr` の欄、event 名と reason の有無が合わない記録、古い worktree の台帳の注意(`orgLedgerReadOnly`)は CLI で打っていない。それぞれ `TestMergeOrder_ResultJSONShape`(`:1102`)、`TestMergeOrder_LatestGateRecordWins`(`:454`)、`TestOrgMergeOrder_StateDirFlagPicksTheLedger`(`org_merge_order_test.go:878`)とその周りのテストが受け持つ
- 足すと確信がいちばん増える確認: この repo の本物の origin(GitHub)で、存在するリモートのブランチを名指す一時の台帳を使い、`ralph org merge-order --json` を 1 回打つ。HEAD の symref と `gh` の欄名が実際に読めるかが分かる。読むだけだが外へのネットワークを使うので、この verify では打っていない

## Verdict

- Verdict: pass(cycle 2 の HEAD fdb457a8 で確かめ直した。cycle 1 も pass)
- Verified: ブランチ全体(HEAD fdb457a8)で、AC1〜AC9 と AC11 の経路をコードで確かめた。binary では、cycle 1 が AC1〜AC7・AC11・AC8(PR の欄のほか)・AC9 を R1〜R13(346304fc)で、cycle 2 が gh の PR を origin の owner/repo と突き合わせる直しと AC1・AC2・AC5〜AC9・AC11 を K1〜K18(fdb457a8)で再現した。静的解析はすべて通った(go1.25.8 の build と vet を含む)。skill の 4 面は同一。仕様 FR-8 の読み方、7c-2 と 8a への送り、B2 との一致を確かめた。cycle 1 の D1・D2 は 93af1f87 で直った。下の Partially verified と Not verified の 2 行は cycle 1 の時点のもので、cycle 2 の分は `## Cycle 2` の「判定(cycle 2)」にある(D3・D4、LOW)
- Partially verified: AC10。README 262 行目の 1 文が分類の判定の順を言わない(D1、LOW)。tech-debt 231 行目の Trigger が発火したが記録がない(D2、LOW)。どちらも /sync-docs で直せる文書の直しで、動きは変わらない
- Not verified: テストの実行(/test)、ls-remote の timeout と ssh の prompt、本物の GitHub と `gh`、他者がロックを握る台帳

## Cycle 2

- Date: 2026-10-14
- Verifier: verifier subagent (Claude)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の上限 2 の最後)。self-review の cycle 2(5d788e2c、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 6、Merge: yes)のあとに走らせた
- Scope: `git diff cab2fd50..HEAD`(HEAD fdb457a8、5 コミット、14 ファイル、+974/-70)。15d4b947(gh の PR を origin の owner/repo のものだけ使う直し、Detail の `--org-id`、gh の失敗の注記のまとめ、テスト)、ee302b06(skill の 4 面、README、tech-debt の 2 行)、7632bb53(計画の Progress の 1 行)、5d788e2c(self-review の cycle 2)、fdb457a8(その insight event)。AC の確かめ直しに要るところは cycle 1 の差分も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD fdb457a8 のもの。上の cycle 1 の節の行番号は 346304fc のままで、`lookupMergePRs` より後ろは 100 行以上ずれている(例: `classifyMergeTarget` は 556 行から 763 行)
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は c0b1e9230999 で、`- Approved:` の digest と同じ。cycle 2 の逸脱は `## Progress checklist` の最後の項目(計画の 158 行)に書かれている
- Evidence: `docs/evidence/verify-2026-10-14-org-merge-order-c2.md`。静的解析の出力と、HEAD から go1.25.8 で作った binary を一時の bare origin・repo・台帳で打った K1〜K18 を入れた。PATH は一時のディレクトリで、git の wrapper(`git remote get-url origin` の答えだけを差し替え、ほかは本物の git に渡す)と偽の `gh` しか置いていないので、本物の `gh` にも GitHub にも届かない。scratch で組み立てたものを cp し、`cmp` で同じことを確かめた。手で書いたのは先頭の説明だけ。`./scripts/run-static-verify.sh` が自分で書いた生の出力は `docs/evidence/verify-2026-10-13-223656.log`(gitignore の対象、名前の日付は UTC)
- insight event: lead の指示(報告と evidence だけをコミット)と、この cycle の self-review の行を orchestrator が別のコミット(fdb457a8)で足した流れに合わせ、`docs/insights/events/2026-10-13-org-merge-order.jsonl` には書いていない。`insights-append.sh --events-dir <scratch>` で作った行は `"phase":"verify","cycle":2,"verdict":"pass"`、`"low":2`(D3 と D4)

### cycle 2 の変更の確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| gh の PR は、PR の url の owner/repo が origin のもの(`git remote get-url origin`)と同じときだけ使う。合わなければ全部の PR を捨て、注記 1 行で `gh repo set-default <owner/repo>` を言い、gh をそれ以上打たない(cross-review cycle 1 の ACTION_REQUIRED #1) | 満たす(実機でも確認) | `lookupMergePRs`(`internal/org/mergeorder.go:472`)、`mergeOriginRepo.check`(`:572`)、`readMergeOriginRepo`(`:605`)、`mergeRepoFromOriginURL`(`:632`)、`mergeRepoFromPRURL`(`:664`)。テスト: `TestMergeOrder_PRIsUsedWhenItsURLNamesOriginsRepository`(`internal/org/mergeorder_test.go:899`)、`TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`(`:951`)、`TestMergeOrder_LocalPathOriginSkipsTheRepositoryCheck`(`:1018`)、`TestMergeOrder_ReadsOwnerAndRepoFromOriginAndPRURLs`(`:1041`)、`TestMergeOrder_GetURLRunsOnlyOnceGHReturnsAPR`(`:1092`)、`TestMergeOrder_OriginURLThatCannotBeReadLeavesPRsOut`(`:1120`)、`TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`(`internal/cli/org_merge_order_test.go:782`)。K1(scp 形の `git@github.com:acme/app.git` と acme/app の PR)は PR を使い、get-url 1 回、gh 9 回。K3(`https://GitHub.com/ACME/App.git`)は大文字小文字を無視して使う。K4(gh が upstream-org/app を引く)は PR を全部捨て、注記は `gh reads upstream-org/app, not origin's acme/app: ... run gh repo set-default acme/app in this repository`、gh は最初の 1 回で止まる。merged だった feat/m は no_remote、wrong_base だった feat/w と closed だった feat/c は ready になる。K6(先の 3 本は acme/app、4 本目の feat/c だけ upstream-org/app)は、先に付けた PR も含めて全部捨てる(gh 4 回)。K7(url に owner/repo がない)と K8(get-url が exit 2)も全部捨て、理由を注記する。K9(ssh の別名 `github.com.emu`)と K10(別の host で同じ owner/repo)は host を比べないので PR を使う。K11(本物の get-url がローカルのパスを返す)と K12(`git+ssh://`)は確かめずに upstream-org/app の PR を使う。K14(gh なし)は get-url を打たない |
| Detail の `ralph org gate` に最新の結びつきの `--org-id` を付ける(self-review の L1) | 満たす(実機でも確認) | `mergeGateCommand`(`mergeorder.go:793`)、`mergeGateRunsWhileOrgRuns`(`:101`)。テスト: `TestMergeOrder_DetailsNameTheGateCommandWithTheOrg`(`mergeorder_test.go:470`)。K1 の regate・failed・no_gate、K17 と K18 の rebase は、どれも `ralph org gate --org-id <org>` と「(the gate runs only while the feature org runs)」を出す。結びつきのない feat/x(gate の記録だけが名指す)の Detail は `--org-id` のない `ralph org gate` で、K16 でその形を CLI が `org: --org-id is required` と断ることを確かめた(self-review の C2-1 の再現) |
| 同じ理由の gh の失敗を 1 つの注記にまとめる(self-review の L2 の半分) | 満たす(実機でも確認) | `mergeGHFailures`(`mergeorder.go:509`、`notes` は `:528`)。テスト: `TestMergeOrder_GHFailingTheSameWayForSeveralBranchesIsOneNote`(`mergeorder_test.go:1245`)、`TestOrgMergeOrder_MultiLineGHStderrStaysOneNoteLine`(`org_merge_order_test.go:754`)。K13(gh が全ブランチで exit 4)の注記は 1 行で、9 本のうち 5 本を名指して「and 4 more」と言う。PR が 1 つも返らないので get-url は打たない |

### AC の確かめ直し(HEAD fdb457a8)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(実機でも確認) | K1 で feat/r が ready、gate のあとに push した feat/g が regate。K17(origin の写しで main を gate のあとに 1 つ進めた)で、ready だった 3 本が rebase。`classifyGatePassed`(`mergeorder.go:804`)で cycle 2 に変わったのは Detail の文言だけ |
| AC2 | 満たす(実機でも確認) | K1 で feat/f が failed(static_failed)、feat/n と feat/q が no_gate。no_remote は K4・K14 の feat/m(PR を使わないとき)。最新の記録で上書きする `collectMergeTargets`(`:308`)は cycle 2 で変わっていない |
| AC3 | 満たす(cycle 1 の実機確認のまま) | `collectMergeTargets` は cycle 2 で変わっていない(`git diff cab2fd50..HEAD` で確かめた)。K の台帳には disband の行を置いていない |
| AC4 | 満たす(cycle 1 の実機確認のまま) | 分類の経路は変わっていない。テストが 2 本増えた: `TestMergeOrder_UnreadableRecordAttachesToTheLatestOfTwoEarlierBindings`(`mergeorder_test.go:632`)、`TestMergeOrder_RecordsWithoutAnOrgIDAreNotRead`(`:745`)。evidence の先頭に書いたとおり、捨てた最初の試みでは、`gateRecordFromDetails` が読まない reason(`lint_failed`)の gate_failed を置いたブランチが unreadable になった(その出力は evidence に入れていない) |
| AC5 | 満たす(実機でも確認) | K1 は merged の feat/m と closed の feat/c を隠して数を注記し、K2(`--all`)は末尾に merged・closed を出す。K13(gh の失敗)、K14(gh なし)、K4〜K8(PR が origin の repository のものと確かめられない)は、どれも PR の欄を空にして注記し、終了コード 0。`pickMergePR` は変わっていない |
| AC6 | 満たす(実機でも確認) | K1〜K18 の並びは `mergeClassOrder`(`:75`)のとおり。K4 で ready が 3 本のときの注記は「3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)」で、cycle 1 の括弧書きから文言が変わった |
| AC7 | 満たす(実機でも確認) | K15: origin のない repo で、gh を PATH に置き get-url の偽物も用意しても、stdout は空、終了コード 1、gh 0 回、get-url 0 回。ls-remote(`listOriginBranches`、`:387`)は gh より前に打つ。timeout は cycle 1 と同じくテストだけ |
| AC11 | 満たす(実機でも確認) | K18: origin の写しの HEAD を trunk(main より 1 つ先)に向け、main を残すと、ready だった 3 本が「the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk」の rebase になる。wrong_base は K1 の feat/w(統合先が release の PR #23) |
| AC8 | 満たす(PR の欄のほかは実機でも確認) | K5(`--json`)は K4 のテキストと同じ 9 本を同じ順で出す。PR を捨てた回なので `pr` の欄はどの entry にもなく、`gh_available` は true で、`notes` の先頭に set-default の文がある。PR の欄つきの JSON は binary では打っていない(`TestMergeOrder_ResultJSONShape`、`mergeorder_test.go:1578`) |
| AC9 | 満たす(実機でも確認) | K1〜K14 の前後で、manifest の sha256・行数・バイト数・台帳のディレクトリの中身が同じ(15 行、3718 バイト、`manifest.jsonl` だけ)、work と origin の refs の digest が同じ、state・work・origin・HOME の下に marker より新しいファイルは 0 件。K15〜K18 のあとも manifest は打つ前の写しと `cmp` で同じ。`git remote get-url origin` は設定を読む git の命令で、fetch も書き込みもしない |
| AC10 | 一部(D3・D4、LOW) | skill の 4 面は `cmp` で同一、`check-skill-sync.sh`(13 skill in lock-step)と `check-sync.sh`(DRIFTED 0)が通る。中身のずれは下の表の D3 と D4 |

計画の逸脱の記録: Progress の 158 行は、`--repo` を渡さないこと、gh が PR を返したときに `git remote get-url origin` を 1 回だけ打つこと、owner/repo を大文字小文字を無視して比べること、合わなければ PR をすべて捨てて注記 1 行(`gh repo set-default <origin の owner/repo>`)を出し、gh をそれ以上打たないこと、host を比べないこと、host のない origin では比べないこと、それが「git は ls-remote を 1 回」からの逸脱であることを書いている。K1〜K14 の結果はこの記述と合う。Scope と Design decisions の C5 の本文は変わっていない(digest が承認時と同じことと合う)。

仕様: FR-8 の「merge してよい順」の読み方は cycle 2 で変わらない。足したのは、PR の state と統合先を使う前の前提(その PR が origin の repository のもの)で、`ready` の定義(gate の head・base・base_ref を origin と比べる)には触れていない。

### 静的解析(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK、`check-sync.sh` PASS(IDENTICAL 167、DRIFTED 0)、`check-skill-sync.sh` 13 skill in lock-step、golang(`gofmt: ok`、`go vet ./...` と `staticcheck ./...` は出力なし、`golangci-lint run ./...` 0 issues)、`secret-scan-branch: scanned ac34ccf3..fdb457a8 against origin/main: clean` |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod は `go 1.25.8`、手元は go1.26.0。cycle 2 で使い始めた `net/url`、`strconv`、組み込みの `min` も 1.25.8 で build できる |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし。cycle 1 は `./internal/org/... ./internal/cli/...` だけで、今回は全体 |
| `go vet ./...`(go1.26.0)、`gofmt -l internal/ cmd/` | pass | 出力なし |
| `/org` skill の 4 面の `cmp` | pass | 同一 |
| `git diff --check ac34ccf3...HEAD` | pass | rc 0 |

### 文書のずれ(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| help(`internal/cli/org_merge_order.go:47-50`)、skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`)と「merge してよい順」節の項目(`:928-933`)、README の org の節(`README.md:262`)、`mergeorder.go` の先頭の説明(`:32-37`) | 一部ずれ(D3、LOW) | どれも「gh の PR は repository が origin のものと同じときだけ使い、違えば PR を使わずに set-default を打つよう注記で言う」と言い切るが、コードには例外が 3 つある。(a) origin の URL から owner/repo を読めないとき(ローカルのパス、`file://`、`git+ssh://` などのスキーム)は確かめずに使う(`mergeorder.go:585`)。K11 と K12 では、gh が引いた upstream-org/app の PR がそのまま使われ、merged の feat/m と closed の feat/c が隠れ、feat/w がその PR の統合先で wrong_base になり、ready の feat/r に upstream の PR #21 が付いた。注記は出ない。(b) host は比べない(K10 では、host の違う origin でも github.com の PR を使った)。(c) 注記は gh が別の repository の PR を返したときだけ出る。別の repository にその名前の PR がなければ(`:489` の `case pr == nil`)、origin に PR がないのと同じに見え、注記も出ない。3 つとも `docs/tech-debt/README.md:237` と `mergeOriginRepo.check` の doc(`:565-571`)には書いてあり、計画の Progress(158 行)にも (a) と (b) の決定がある。動きは決めたとおりで、ずれは読む人に見える 4 か所の言い方だけ。self-review の C2-3 が挙げた 2 つ((a)(b))に (c) を足した。上限に届いているので、PR の本文の known gaps に tech-debt の 237 行を挙げるのが妥当。直すなら help・skill・README に「origin の URL が owner/repo を持たないときは確かめない、host は比べない、注記は gh が別の repository の PR を返したときだけ出る」の 1 文を足す |
| `internal/org/seat.go:7-10` の package doc | ずれ(D4、LOW) | 外のコマンドの一覧が、mergeorder.go の git を「`git ls-remote` in mergeorder.go」とだけ書く。15d4b947 から `git remote get-url origin` も打つ。動きは変わらない。次に触るときに「`git ls-remote` and `git remote get-url origin` in mergeorder.go」にする |
| `internal/cli/org_merge_order.go` の関数の doc(15-24 行)、`MergeOrderEntry.PR`・`MergeOrder`・`lookupMergePRs` の doc | yes | ls-remote、gh、`git remote get-url origin` の 3 つを打つこと、PR を捨てる 3 つの場合、注記のまとめ方が、コードと K の出力に合う |
| tech-debt の追加 2 行(`docs/tech-debt/README.md:237`、`:238`) | yes | 挙げた名前は今のコードにある(`mergeOriginRepo.check`、`lookupMergePRs`、`mergeRepoFromOriginURL`、`mergeRepoFromPRURL`、`realMergeOrderExec` と `mergeOrderWaitDelay`、`realGitStatus` と `gitStatusWaitDelay`(`internal/org/watch.go:358`)、`gateKillGrace`(`internal/org/gate_run.go:54`)、JSON の `gh_available`)。237 行の (a)(b) は K10〜K12 で再現した。238 行の「entry の `pr` がないだけで、gh の失敗と PR がないことは notes の文でしか分からない」は K13 と K5 の出力と合う。どちらも 5 列 |
| 仕様 FR-8 の 7c-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`) | yes(情報) | cycle 2 で変わっていない。「`gh` は任意で、あればブランチごとに PR を引き(fork でない PR のうち番号が最大のもの)」までを書き、origin の repository との突き合わせは書かない。注は「詳細は ... 計画に書いた」で終わり、計画の Progress にその決定があるので、食い違う記述はない |
| cycle 1 のずれ D1(README の判定の順)と D2(tech-debt 231 行の Trigger) | 直った | 93af1f87(sync-docs)。README 262 行は「checked in this order: ... The list is in a different order: ...」と書き、231 行は「Stage 7c-1 (`feat/org-merge-order`) carried this over without paying it」と書く |
| 計画の AC の checkbox と Progress | yes | cycle 1 の情報の項目は解消した。AC1〜AC11 は `[x]`、「Review artifact created」なども `[x]` で、「PR created」だけが `[ ]` |
| self-review の cycle 2 が勧めた tech-debt の 1 行(C2-1〜C2-6、L3・L5・L6) | 未追加(情報) | self-review の報告は「orchestrator が足す」と書く。`docs/tech-debt/README.md` にはまだない。/sync-docs か orchestrator が足す |
| cycle 1 の evidence(`docs/evidence/verify-2026-10-14-org-merge-order-cli.md`)の Detail の文 | 情報 | 346304fc の時点の出力で、L1 の前の文言(`run ralph org gate again` など)を引いている。その時点の記録として正しいので、直す必要はない。HEAD の文言は cycle 2 の evidence にある |

### 観察

- 本物の git で、`git remote get-url origin` は `url.<base>.insteadOf` を展開した(`https://example.invalid/acme/app.git` が `git@alias.invalid:acme/app.git` になった)。self-review が未確認とした点のうち、merge-order が読む URL は書き換えたあとのものだと分かった。ネットワークに触れないことは、git の doc の記述と 0.043 s で返ったことに合う。ただ `.invalid` は名前の解決もすぐ失敗するので、時間だけでは言い切れない
- この worktree の本物の origin は `ssh://git@github.com/yoshpy-dev/ralph.git`(設定を読んだだけ)。`mergeRepoFromOriginURL` は `ssh` を読むスキームに入れており、owner/repo は yoshpy-dev/ralph になる。cycle 1 の /test の evidence(`docs/evidence/test-2026-10-14-org-merge-order-real-origin.txt`)にある本物の gh の url `https://github.com/yoshpy-dev/ralph/pull/225` も yoshpy-dev/ralph と読めるので、この repo では突き合わせが通り、PR が使われるはず。おそらく正しいが、cycle 2 では本物の GitHub に打っていないので未確認
- K6 は、self-review が「通常は到達しない」と書いた防御(後の PR の不一致で、先に付けた PR も全部外す)を偽の gh で通し、出力で外れたことを確かめた
- self-review の C2-5(skill の 1112 行だけが短い)は 4 面とも同じで、`cmp` が通る。読みやすさの所見で、ずれではない

### Coverage gaps(cycle 2)

- テストは走らせていない(/test の担当)
- 本物の GitHub と本物の gh は使っていない。fork の作業場所で本物の gh が upstream を引くこと(cycle 2 の直しの前提)は、gh の help の記述(cross-review の triage)によるもので、観察していない
- get-url の timeout(`GHTimeout` を流用する。self-review の C2-2)と、get-url の最中の中断は binary では打っていない。`TestMergeOrder_OriginURLThatCannotBeReadLeavesPRsOut`(`mergeorder_test.go:1120`、中断は 1152〜1164 行)が受け持つ
- D3 の (c)(別の repository にその名前の PR がないとき注記が出ない)は、コードを読んで確かめただけ
- `--json` の PR の欄は binary で打っていない(cycle 1 と同じ)
- 足すと確信がいちばん増える確認: この worktree で、merge 済みの本物のブランチ(feat/org-gate、PR #225)を名指す一時の台帳を使い、HEAD の binary で `ralph org merge-order --json` を 1 回打つ。get-url の ssh の URL と本物の gh の url が突き合わせを通り、PR #225 が merged として隠れることが分かる。読むだけだが GitHub に出るので、この verify では打っていない

### 判定(cycle 2)

- Verdict: pass
- Verified: cycle 2 の 3 つの直しをコードとテストの名前で辿り、HEAD fdb457a8 の binary で K1〜K16 として再現した。AC1・AC2・AC5〜AC9・AC11 を同じ binary で確かめ直した(AC8 の PR の欄のほか)。AC3・AC4 は経路が変わっていないことを diff で確かめた。静的解析はすべて通った(go1.25.8 の build と vet を含む)。計画の逸脱は Progress に記録されている
- Partially verified: AC10。読む人に見える 4 か所が gh の PR の突き合わせを例外なしに言い切る(D3、LOW)。`seat.go` の package doc が get-url を挙げない(D4、LOW)。どちらも動きは変わらず、D3 の例外は tech-debt の 237 行に書いてある
- Not verified: テストの実行(/test)、本物の GitHub と gh、get-url の timeout と中断(テストだけ)、D3 の (c)
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2(D3、D4)。情報は 4 つ(仕様の注、self-review の tech-debt の行が未追加、cycle 1 の evidence の文言、K16 で再現した C2-1)
