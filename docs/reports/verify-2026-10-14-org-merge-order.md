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

- Verdict: pass(cycle 4 の HEAD a7cbdd87 で確かめ直した。cycle 1〜3 も pass)
- Verified: ブランチ全体(HEAD a7cbdd87)で、AC1〜AC9 と AC11 の経路をコードで確かめた。binary では、cycle 1 が AC1〜AC7・AC11・AC8(PR の欄のほか)・AC9 を R1〜R13(346304fc)で、cycle 2 が gh の PR を origin の owner/repo と突き合わせる直しと AC1・AC2・AC5〜AC9・AC11 を K1〜K18(fdb457a8)で、cycle 3 が gh の `--limit 1000`、上限ぶんの fork の PR だけが返ったブランチの注記、Detail の gate の記録の org_id と、AC1・AC2・AC5・AC6・AC8(PR の欄を含む)・AC9・AC11 を Q1〜Q7(48f0aed7)で、cycle 4 が repository の不一致の注記に改名の直し方(`git remote set-url origin`)を並べる直しと、結びつきのない org の読めない gate の記録をその org の前の読める記録のブランチに付ける直し、AC1・AC4〜AC6・AC8・AC9 を V1〜V9(a7cbdd87)で再現した。直す前の binary を対照にした回が 2 つある。Q8(e463eb9d)は、fork の PR 30 本に埋もれた PR を注記なしで落とした。V4・V7(999e0394)は、古い注記を出し、読めない gate_failed のあとの 3 本を古い pass で `ready` に戻した。静的解析はすべて通った(go1.25.8 の build と vet を含む)。skill の 4 面は同一。仕様 FR-8 の読み方、7c-2 と 8a への送り、B2 との一致を確かめた。cycle 1 の D1・D2 は 93af1f87、cycle 2 の D3・D4 は f1a05bc7、cycle 3 の D5〜D7 は f269c0db で直った。下の Partially verified と Not verified の 2 行は cycle 1 の時点のもので、cycle 2〜4 の分はそれぞれ `## Cycle 2` の「判定(cycle 2)」、`## Cycle 3` の「判定(cycle 3)」、`## Cycle 4` の「判定(cycle 4)」にある(cycle 4 は D8・D9、LOW)
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

## Cycle 3

- Date: 2026-10-14
- Verifier: verifier subagent (Claude)。パイプラインの 3 回目で最後(cross-review cycle 2 の ACTION_REQUIRED を直すために上限を 2 から 3 に上げた回。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。self-review の cycle 3(a8fd9992、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 5、Merge: yes)のあとに走らせた
- Scope: `git diff e463eb9d..HEAD`(HEAD 48f0aed7、5 コミット、11 ファイル、+340/-46)。984a163e(計画の Progress の 1 行)、d7599d82(`gh pr list` に `--limit 1000`、上限ぶんの fork の PR だけが返ったブランチは PR を使わずに注記、help と skill の 4 面、テスト)、c885fcab(self-review の C2-1 の `mergeGateCommand`、C2-2 の `GHTimeout` の doc)、a8fd9992(self-review の cycle 3)、48f0aed7(その insight event)。AC の確かめ直しに要るところは前の cycle の差分も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD 48f0aed7 のもの
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は c0b1e9230999 で、`- Approved:` の digest と同じ。cycle 3 の決定は `## Progress checklist` の最後の項目(計画の 159 行)に書かれている
- Evidence: `docs/evidence/verify-2026-10-14-org-merge-order-c3.md`。静的解析の出力、手元の gh の binary から読んだ `PullRequestList` の query、HEAD から go1.25.8 で作った binary を一時の bare origin・repo・台帳で打った Q1〜Q8 と、その probe の script を入れた。Q8 は直す前の e463eb9d から作った binary で同じ台帳を打った対照。PATH は一時のディレクトリで、git の wrapper(`git remote get-url origin` の答えだけを差し替える)と偽の `gh`(`--limit` を読み、なければ gh の既定の 30 で切る。fork の PR を新しい順に並べ、その後ろに本物の PR を置く)しか置いていないので、本物の `gh` にも GitHub にも届かない。scratch で組み立てたものを cp し、`cmp` で同じことを確かめた(U+FFFD は 0 件)。手で書いたのは先頭の説明だけ。`./scripts/run-static-verify.sh` が自分で書いた生の出力は `docs/evidence/verify-2026-10-14-011142.log`(gitignore の対象、名前の日付は UTC)
- insight event: lead の指示どおり書いていない。`insights-append.sh --events-dir <scratch>` で作った行は `"phase":"verify","cycle":3,"verdict":"pass"`、`"low":3`(D5〜D7)。`--cycle auto` は `cycle-count.json` の 2 を読むので、`--cycle 3` を渡した

### cycle 3 の変更の確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| `gh pr list` に `--limit 1000` を渡す(cross-review cycle 2 の ACTION_REQUIRED #1) | 満たす(実機でも確認) | `mergeOrderGHLimit`(`internal/org/mergeorder.go:103`)、`lookupMergePR` の argv(`:734`)。テスト: `TestMergeOrder_RunsGitAndGHWithTheirArgsDirAndEnv`(`internal/org/mergeorder_test.go:1569`、期待は `:1587`)、CLI の `mergeOrderGHCall`(`internal/cli/org_merge_order_test.go:80`)。Q1〜Q5 の gh の argv(各 9 本)と Q6 の 3 本は、どれも `--limit 1000` を含む。Q8(e463eb9d)の 9 本には `--limit` がない。Q1 で、本物の PR #23 より新しい fork の PR が 30 本ある feat/w は、HEAD では #23(統合先 release)を読んで `wrong_base`、e463eb9d では 30 本の fork だけを受けて PR なしの `ready` になり、注記も出ない。fork の PR 40 本のあとに閉じた #20 がある feat/c は、HEAD では隠れて「1 branch(es) whose PR is merged or closed are not shown」に数えられ、`--all`(Q3)の末尾に `closed` で出る。e463eb9d では `ready` |
| 上限ぶん返った PR がすべて fork のブランチは、PR を使わずに注記 1 行で名指す | 満たす(実機でも確認) | `lookupMergePR` の戻り(`mergeorder.go:746`、`pr == nil && len(prs) >= mergeOrderGHLimit`)、`lookupMergePRs` の上限の枝(`:508-510`)と `withNotes`(`:494-496`)、`mergeGHLimitNotes`(`:568`)、`mergeNamedBranches`(`:583`)。テスト: `TestMergeOrder_GHLimitFullOfForksLeavesThePROut`(`mergeorder_test.go:1341`)。Q1 で、fork の PR が 1000 本返り本物の PR が上限の外にある feat/m と feat/k は PR なしで、注記は「gh returned 1000 PRs, its limit, for each of 2 branches (feat/m, feat/k), all from forks, ...」の 1 行。fork 999 本と本物の #24 で計 1000 本が返る feat/j は #24 を使い、注記に出ない(上限ちょうどでも本物が 1 本混じれば使う境界)。Q4(feat/m だけが上限)は単数の文「gh returned 1000 PRs for feat/m, its limit, ...」。Q5(feat/j で gh が exit 4)は、gh の失敗の注記、上限の注記の順。Q6(本物の PR の url が upstream-org/app)は、上限の 2 本のあと feat/r の PR で突き合わせが外れて gh は 3 回で止まり、上限の注記と set-default の注記の 2 行が残る。上限のブランチは PR なしとして分類するので、push 済みの feat/k は PR の欄が `-` の `ready` と出る。計画の Progress 159 行の決定どおりで、注記が名指すので黙ってはいない |
| Detail の `ralph org gate` に、結びつきがなければ gate の記録の org_id を付ける(self-review の C2-1) | 満たす(実機でも確認) | `mergeGateCommand`(`mergeorder.go:845`)。テスト: `TestMergeOrder_DetailsNameTheGateCommandWithTheOrg`(`mergeorder_test.go:473`)の feat/orphan と、結びつきの org と gate の記録の org が違う `twice`(`:519-526`)。Q1 で、結びつきがなく org-q の gate の記録だけが名指す feat/x の Detail は `run ralph org gate --org-id org-q again`。同じ台帳で Q8(e463eb9d)は `run ralph org gate again` で、cycle 2 の K16 で CLI が断った形 |
| `GHTimeout` の doc が get-url にも使うことを言う(self-review の C2-2 の半分) | 満たす(読んで確認) | 定数の説明(`mergeorder.go:84-86`)と `MergeOrderParams` の doc(`:132-134`)。get-url の `GIT_TERMINAL_PROMPT=0`(`:653`)は残る(self-review の C3-1)。動きは変わらない |

### AC の確かめ直し(HEAD 48f0aed7)

cycle 3 の `mergeorder.go` の hunk は、定数、`MergeOrderParams`・`MergeOrder` の doc、`lookupMergePRs`、`mergeGHFailures.notes`、新しい 2 関数、`lookupMergePR`、`mergeGateCommand` だけで(evidence の `git diff -U0` の hunk の一覧)、`collectMergeTargets`(`:321`)、`listOriginBranches`(`:400`)、`classifyMergeTarget`(`:812`)の本体、`classifyGatePassed`、`orderMergeEntries`、`readMergeLedger` は変わっていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(実機でも確認) | Q1 で feat/k・feat/r・feat/j が ready、gate のあとに push した feat/x が regate。main が進んだ rebase は cycle 2 の K17 のまま(分類の経路は変わっていない) |
| AC2 | 満たす(実機でも確認) | Q1 で push していない feat/m が no_remote、feat/n と feat/q が no_gate で、Detail はどちらも `--org-id` を付ける。failed は cycle 2 の K1 のまま |
| AC3 | 満たす(cycle 1 の実機確認のまま) | `collectMergeTargets` は cycle 3 で変わっていない |
| AC4 | 満たす(cycle 1 の実機確認のまま) | 分類の経路は変わっていない |
| AC5 | 満たす(実機でも確認) | 上の 1 行目と 2 行目のとおり。Q1 は閉じた PR の feat/c を隠して数を注記し、Q3(`--all`)は末尾に `closed` を出す。fork でない PR のうち番号が最大のものを採ることは feat/w(#23)・feat/j(#24)で見えた。Q5(1 本で gh が失敗)、Q6(PR が origin の repository のものでない)、Q7(gh なし)は、どれも PR の欄を空にして注記し、終了コード 0。Q8 で、直す前の binary は、本物の PR が fork の PR 30 本の後ろにあった 3 本(feat/w・feat/c・feat/j)を、注記なしで PR なしにしていた(feat/m・feat/k も注記なし) |
| AC6 | 満たす(実機でも確認) | Q1 の並びは ready(gate の時刻の 01:00:12 の feat/k、:13 の feat/r、:16 の feat/j)、wrong_base、regate、no_remote、no_gate(結びつきの順に feat/n、feat/q)で、`mergeClassOrder` のとおり。ready が 3 本なので「3 branches are ready, but merging one moves origin/main ...」の注記が出る |
| AC7 | 満たす(cycle 2 の実機確認のまま) | `listOriginBranches` は cycle 3 で変わっていない。ls-remote は gh より前に打つので、gh の引数を変えても AC7 の経路に触れない(cycle 2 の K15) |
| AC11 | 満たす(実機でも確認) | wrong_base は Q1 の feat/w(統合先が release の PR #23)。既定のブランチを変えた rebase は cycle 2 の K18 のまま |
| AC8 | 満たす(実機でも確認、PR の欄も) | Q2(`--json`)は Q1 のテキストと同じ 8 本を同じ順で出す。feat/r・feat/j・feat/w の entry に `pr`(`number`・`state`・`url`・`head_ref_oid`・`base_ref_name`)が出る。cycle 1・2 で binary では見ていなかった PR の欄を、ここで初めて見た。上限で PR を使わなかった feat/k と feat/m の entry には `pr` がなく、それを知らせるのは `notes` の先頭の文だけ(D6)。`gh_available` は true |
| AC9 | 満たす(実機でも確認) | Q1〜Q8 の前後で、manifest の sha256(f9f7a96f…)・行数・バイト数(15 行、3697 バイト)・台帳のディレクトリの中身(`manifest.jsonl` だけ)が同じ。work と origin の refs の digest も同じで、state・work・origin・HOME の下に marker より新しいファイルは 0 件 |
| AC10 | 一部(D5〜D7、LOW) | skill の 4 面は `cmp` で同一、`check-skill-sync.sh`(13 skill in lock-step)と `check-sync.sh`(DRIFTED 0)が通る。中身のずれは下の表の D5〜D7 |

計画の逸脱の記録: Progress の 159 行は、`--limit 1000` を定数で渡すこと、gh が `--head` だけのときは作成の新しい順に 1 ページ 100 件で辿ること(consult が gh の source で確かめたと triage にある)、返った数が上限と同じで fork でない PR が 1 本もないときはそのブランチの PR を使わずに注記すること、別のコミットで C2-1 と C2-2 を直すことを書いている。d7599d82 と c885fcab、Q1〜Q8 の結果はこの記述と合う。Scope の 26 行の gh の argv には `--limit` がないが、承認したあとの計画の本文は変えない決まり(digest が承認時と同じことと合う)で、逸脱は Progress にある。

仕様: FR-8 の「merge してよい順」の読み方と `ready` の定義は cycle 3 で変わらない。変わったのは PR の候補を読む数と、読み切れないときの扱いだけで、gate の head・base・base_ref を origin と比べる部分には触れていない。

### gh の並び順(任意の確認)

- 確かめたこと: 手元の gh 2.102.0(`/opt/homebrew/Cellar/gh/2.102.0/bin/gh`)の binary の文字列に、`query PullRequestList(` が 1 つあり、`pullRequests(states: $state, baseRefName: $baseBranch, headRefName: $headBranch, first: $limit, after: $endCursor, orderBy: {field: CREATED_AT, direction: DESC})` と `pageInfo { hasNextPage endCursor }` を持つ(evidence の「gh」の節、`strings` で読んだだけでネットワークは使っていない)。ブランチの名前で絞る `gh pr list` の GraphQL の query は、作成の新しい順に `$limit` まで、cursor でページを辿る。PR の番号は作られた順に振られるので、新しい順は番号の大きい順と同じで、本物の PR より番号の大きい fork の PR が 1000 本を超えない限り、本物の最新の PR は返る 1000 本に入る
- `gh pr list --help` で、`--limit` の既定が 30、`--head` が `<owner>:<branch>` を受けないことを見た(self-review と同じ)
- 確かめていないこと: `--head` だけ(`--search`・`--author`・`--label` などなし)のときに gh がこの query を使い、検索の API を使わないこと、1 ページが 100 件であること。binary の文字列からは分岐とページの大きさは読めない。どちらも triage が consult の source の読みとして書いている。おそらく正しいが、未確認です

### 静的解析(cycle 3)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK、`check-sync.sh` PASS(IDENTICAL 167、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照、golang(`gofmt: ok`、`go vet ./...` と `staticcheck ./...` は出力なし、`golangci-lint run ./...` 0 issues)、`secret-scan-branch: scanned ac34ccf3..48f0aed7 against origin/main: clean` |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし |
| `go vet ./...`(go1.26.0)、`gofmt -l internal/ cmd/` | pass | 出力なし |
| `scripts/check-skill-sync.sh`、`scripts/check-sync.sh`(単独で) | pass | 13 skill(s) in lock-step。DRIFTED 0 |
| `/org` skill の 4 面の `cmp` | pass | 同一 |
| `git diff --check ac34ccf3...HEAD`、`git diff --check e463eb9d..HEAD` | pass | どちらも rc 0 |

### 文書のずれ(cycle 3)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| help(`internal/cli/org_merge_order.go:45-48`) | yes | `--limit 1000` と「when all 1000 are from forks, it takes none and a note says so」は、Q1 の argv、Q1・Q4 の注記と合う |
| skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`、4 面とも同じ) | yes | `--limit 1000` と「返った 1000 本がすべて fork の PR なら、PR を使わずに注記で言う」は、コードと Q1・Q4 に合う |
| skill の「merge してよい順」節(`.claude/skills/org/SKILL.md:871` 以下、gh の項目は `:924-937`) | yes(情報) | gh があるときとないとき、repository の突き合わせの 3 つの穴を挙げるが、上限の場合は挙げない。183 行が書いているので、食い違う記述はない |
| README の org の節(`README.md:262`) | yes(情報) | gh については「looks up each branch's PR」と repository の突き合わせだけを書き、fork を外すことも上限も書かない。食い違う記述はない |
| 仕様 FR-8 の 7c-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`) | yes(情報) | 「fork でない PR のうち番号が最大のもの」は、返った候補の中の話として今も正しい。上限の場合は書かないが、注は「詳細は ... 計画に書いた」で終わり、計画の Progress 159 行にその決定がある |
| `MergeOrderEntry.PR` の doc(`internal/org/mergeorder.go:168-170`) | ずれ(D7、LOW) | PR が nil になる場合を「gh was not there, failed, found none from this repository, or read another repository than origin's」と並べるが、上限ぶんの fork の PR で使わなかった場合を言わない。Q2 の feat/k・feat/m がその場合で、`--json` の `pr` の欄の契約の説明が 1 つ欠ける。self-review の C3-4 の (d) と同じ所見。直すなら 1 句足す |
| tech-debt の merge-order の残りの行(`docs/tech-debt/README.md:239`) | ずれ(D5、LOW) | cycle 3 の直しで古くなった。(a)(gate の記録だけが名指すブランチの Detail に `--org-id` がない)は c885fcab で直った(Q1 と Q8 の feat/x の比較)。(k)(Detail の org を gate の記録から読む mutant が通る)は `twice`(`mergeorder_test.go:519-526`)で閉じた。(b) は doc が直り、get-url の env だけが残る。先頭の「at the pipeline cap (2/2)」と Why deferred の「Found at the pipeline cap (2/2): fixing any of them needs a third pipeline run」は、3 回目が走ったので偽。Impact の「Only (a) reaches a reader」は、読む人に当たるものが ready が 2 本以上のときの注記(self-review の C3-2、Q1 で再現)に替わった。self-review の C3-5 と同じで、直し方はその報告の「register の 239 行の古くなった所」の表にある。/sync-docs が直す |
| tech-debt の `--json` の lookup の結果の行(`docs/tech-debt/README.md:238`) | ずれ(D6、LOW) | 「gh failed for this branch (or gh's PRs were left out for the run) ... both leave out the entry's `pr`」は、run 全体で PR を捨てる場合までを書き、ブランチ 1 本だけ上限で PR を使わない場合を書かない。Q2 で feat/k と feat/m の entry には `pr` がなく、それを知らせるのは `notes` の文だけだった。8a の director が JSON を読むとき、上限で読めなかったブランチは PR のないブランチに見え、feat/k なら `ready` と読める。self-review の C3-5 が 238 行に足すよう勧めている。/sync-docs が直す |
| `internal/org/mergeorder.go` の先頭の説明(`:31-39`) | yes | 「each branch's PR from `gh pr list` when gh is there」とだけ書き、引数や上限を言い切らない |
| `internal/org/seat.go:7-11` の package doc | yes | cycle 2 の D4 は f1a05bc7 で直った(`git ls-remote`, plus `git remote get-url origin` once gh returned a PR)。cycle 3 で新しい外のコマンドはない |
| cycle 2 のずれ D3(gh の突き合わせの例外) | 直った | f1a05bc7。help、skill の 183 行と 928 行以下、README 262 行、`mergeorder.go` の先頭の説明が 3 つの穴を書く |
| 計画の AC の checkbox と Progress | yes | AC1〜AC11 は `[x]`、「PR created」だけが `[ ]`。Progress の最後の項目が cycle 3 の決定を書く |

### 観察

- self-review の C3-2 を binary で再現した。Q1 で ready が 3 本のときの注記は「3 branches are ready, but merging one moves origin/main: the other ready ones then become rebase, so rebase them on origin/main and run ralph org gate again (the gate runs only while the feature org runs)」で、`--org-id` を付けない。同じ出力の Detail は、結びつきのあるブランチも gate の記録だけのブランチ(feat/x)も、どれも `--org-id` を付ける。`ralph org gate` を裸で勧める文は、この注記の 1 つだけになった(`mergeorder.go:917-920`、テストの期待は `mergeorder_test.go:515`)。help と skill はこの文を引用していない
- 上限で PR を使わなかったブランチは、PR がないブランチと同じに分類される。Q1 の feat/k は `ready` で、本物の PR が閉じていても merge 済みでも、それは分からない。テキストでは注記が名指し、JSON では `notes` の文だけが知らせる(D6)。計画の決定どおりで、cross-review の指摘の「黙って `ready` に寄る」のうち、「黙って」が直った
- Q6 で、上限の注記(「no PR is used for them」)と set-default の注記(「no PR is used」)が両方出る。repository の突き合わせが外れると全部の PR を捨てるので、前の注記は結果の説明として重なるが、どちらも正しい。self-review の「所見にしなかったもの」に同じ判断がある
- 偽の gh は 1000 本の JSON を 1 回で返す。本物の gh が 1000 本を 100 件ずつ 10 回の GraphQL で取ること、それが 15 秒(`GHTimeout`)に収まることは見ていない。収まらなければ、そのブランチは timed out の注記になり、結果は PR なしと同じ(self-review の読みのとおり)。通常は PR が 1 本で、1 回で終わる

### Coverage gaps(cycle 3)

- テストは走らせていない(/test の担当)
- 本物の GitHub と本物の gh は使っていない。本物の gh が `--head` と `--limit 1000` を一緒に受けること、`--head` だけのときに作成の新しい順の query を使うこと(上の「gh の並び順」の未確認の 2 点)、1000 本を 15 秒で取り切れることは見ていない
- rebase(main が進んだ、既定のブランチを変えた)、failed、unreadable、AC7 の終了コード 1 は cycle 3 では打っていない。経路が変わっていないことを hunk の一覧で確かめ、cycle 1・2 の R と K の結果を引いた
- `withNotes` が、gh が途中で見つからなくなったときにそれまでの上限の注記を残すことは、binary では打てない(PATH の gh が run の途中で消える場合)。読んで確かめただけ
- 足すと確信がいちばん増える確認: この worktree で、merge 済みの本物のブランチ(feat/org-gate、PR #225)を名指す一時の台帳を使い、HEAD の binary で `ralph org merge-order --json` を 1 回打つ。本物の gh が `--limit 1000` つきで 1 回で返り、PR #225 が merged として隠れることが分かる。読むだけだが GitHub に出るので、この verify では打っていない

### 判定(cycle 3)

- Verdict: pass
- Verified: cycle 3 の 2 つの直し(gh の `--limit 1000` と上限ぶんの fork の PR の扱い、Detail の gate の記録の org_id)をコードとテストの名前で辿り、HEAD 48f0aed7 の binary で Q1〜Q7 として再現した。直す前の e463eb9d の binary(Q8)は同じ台帳で、fork の PR 30 本に埋もれた PR を注記なしで落とし、feat/w と feat/c を `ready` と出した。AC1・AC2・AC5・AC6・AC8(PR の欄を含む)・AC9・AC11 を同じ binary で確かめ直し、AC3・AC4・AC7 は経路が変わっていないことを diff で確かめた。静的解析はすべて通った(go1.25.8 の build と vet を含む)。cycle 3 の決定は計画の Progress に記録されている。gh の PR の一覧の query が作成の新しい順であることは、手元の gh の binary で確かめた
- Partially verified: AC10。tech-debt の 239 行が cycle 3 の直しで古くなった(D5、LOW)。238 行が上限で PR を使わない場合を書かない(D6、LOW)。`MergeOrderEntry.PR` の doc が同じ場合を書かない(D7、LOW)。D5 と D6 は /sync-docs が直す文書の直しで、D7 は doc の 1 句。どれも動きは変わらない
- Not verified: テストの実行(/test)、本物の GitHub と gh(`--head` だけのときの query の選び方、1 ページの件数、1000 本の取得時間)、gh が run の途中で消えたときの注記
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3(D5、D6、D7。D5 は self-review の C3-5、D7 は C3-4 の (d) と同じ所見)。情報は 5 つ(skill の節・README・仕様の注が上限を書かないこと、計画の Scope の argv、C3-2 の再現、Q6 の注記の重なり、1000 本の取得時間)

## Cycle 4

- Date: 2026-10-14
- Verifier: verifier subagent (Claude)。パイプラインの 4 回目で最後(cross-review cycle 3 の ACTION_REQUIRED 1 件と WORTH_CONSIDERING 1 件を直すために上限を 3 から 4 に上げた回。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。self-review の cycle 4(2e3d272a、CRITICAL 0・HIGH 0・MEDIUM 0・LOW 4、Merge: yes)と、その insight event(a7cbdd87)のあとに走らせた
- Scope: `git diff 999e0394..HEAD`(HEAD a7cbdd87、6 コミット、14 ファイル、+338/-59)。546e9f66(cross-review cycle 3 の triage)、fd1f3f30(計画の Progress の 1 行)、813d6d2d(repository が違うときの注記に `git remote set-url origin` と `gh repo set-default` を並べる。help、skill の 4 面、README、tech-debt の 237 行、テスト)、1052a12a(`collectMergeTargets` の `gatedBranch`、付け先のない記録の注記の文、テスト)、2e3d272a(self-review の cycle 4)、a7cbdd87(その insight event)。AC の確かめ直しに要るところは前の cycle の差分も読んだ。テストは走らせていない(/test の担当)。この節の行番号は HEAD a7cbdd87 のもの
- Plan: `Status: Approved`。`./scripts/plan-visual.sh digest` は c0b1e9230999 で、`- Approved:` の digest と同じ。cycle 4 の決定と逸脱は `## Progress checklist` の最後の項目(計画の 160 行)に書かれている
- Evidence: `docs/evidence/verify-2026-10-14-org-merge-order-c4.md`。静的解析の出力と、HEAD から go1.25.8 で作った binary を一時の bare origin・repo・台帳で打った V1〜V9、その probe の script を入れた。V4 と V7 は、直す前の 999e0394 から作った binary で同じ台帳を打った対照。PATH は一時のディレクトリで、git の wrapper(`git remote get-url origin` の答えだけを差し替え、ほかは本物の git に渡す)と偽の `gh`(feat/a にだけ fork でない PR を 1 本返し、ほかのブランチには `[]` を返す)しか置いていないので、本物の `gh` にも GitHub にも届かない。V8 は台帳の写しで `ralph org gate` を打ち、gate はスクリプトを走らせる前に断った。scratch で組み立てたものを cp し、`cmp` で同じことを確かめた(U+FFFD は 0 件、`$HOME` は `~` に置き換えた)。手で書いたのは先頭の説明だけ。`./scripts/run-static-verify.sh` が自分で書いた生の出力は `docs/evidence/verify-2026-10-14-033350.log`(gitignore の対象、名前の時刻は UTC)
- insight event: lead の指示どおり書いていない

### cycle 4 の変更の確かめ

| 変更 | Status | Evidence |
| --- | --- | --- |
| gh が読む repository が origin のものと違うときの注記に、改名の直し方(`git remote set-url origin <its URL>`)と別の repository の直し方(`gh repo set-default <origin の owner/repo>`)を並べる。分類は今までどおり PR なし(cross-review cycle 3 の ACTION_REQUIRED #1) | 満たす(実機でも確認) | `mergeOriginRepo.check`(`internal/org/mergeorder.go:630`、注記は `:652-654`、doc は `:619-629`)。テスト: `TestMergeOrder_RepositoryNoteNamesTheFixForARenameAndForAnotherRepository`(`internal/org/mergeorder_test.go:1120`、2 つの subtest が全文を固定)、`TestMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`(`:1047`)、`TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut`(`internal/cli/org_merge_order_test.go:791`)。V1(origin は `git@github.com:acme/app.git`、gh の PR の url は acme/app-v2)の注記は「gh reads acme/app-v2, not origin's acme/app: no PR is used, so ...; if acme/app was renamed to acme/app-v2, point origin at it (git remote set-url origin <its URL>); otherwise run gh repo set-default acme/app in this repository」で、get-url 1 回、gh は最初の 1 回で止まる。V2(PR の url は upstream-org/app)も同じ形。V3 で origin の URL を acme/app-v2 にすると(注記の 1 つめの直し方。get-url の答えを差し替えて模した)、feat/a の merged の PR が使われて隠れ、gh は 7 本すべてに打たれる。V4(999e0394)は同じ台帳で「...; run gh repo set-default acme/app in this repository」とだけ言う。V1 と V4 の分類と並びは同じで、変わったのは注記の文だけ |
| 結びつきのない org の読めない gate の記録を、その org の前の最新の読める gate の記録のブランチに付ける。結びつきがあればそちらを先に使い、どちらもなければ今までどおりどのブランチにも付けずに注記で数える(cross-review cycle 3 の WORTH_CONSIDERING #2、計画の Scope と C8 を広げる逸脱) | 満たす(実機でも確認) | `collectMergeTargets`(`mergeorder.go:325`、doc は `:312-324`、`gatedBranch` は `:337`、`cmp.Or` の付け先は `:364`)、`mergeLedgerNotes`(`:383`、注記は `:390-391`)。テスト: `TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch`(`mergeorder_test.go:679`)、`TestMergeOrder_UnreadableRecordPrefersTheBindingToAGatedBranch`(`:716`、結びつきと読める記録の前後を 2 通り)、doc を直した `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt`(`:618`)。V5(HEAD、gh なし): 結びつきのない org-c の feat/c は、読める gate_passed のあとの head= が SHA でない gate_failed で `unreadable`。org-e は feat/e と feat/e2 に読める gate_passed を書いたあとに読めない gate_failed を書き、後の feat/e2 だけが `unreadable` で feat/e は `ready`。org-f の feat/f は、reason= のない details を持つ gate_failed(event の名前と reason の有無が合わない記録)で `unreadable`。org-b は feat/x の読める記録のあとに feat/b に結びつき、読めない記録は feat/b に付いて feat/x は `ready` のまま。結びつきも前の読める記録もない org-d の 1 件だけが「1 gate record(s) cannot be read and their org had neither a feature binding nor a readable gate record before them, ...」と数えられる。V6(`--json`)の feat/c は `gate` が `{"event":"gate_failed","readable":false,...}`、`binding` が null。V7(999e0394)は同じ台帳で feat/c・feat/e2・feat/f を `ready` と出し(古い gate_passed に戻る)、注記は「4 gate record(s) cannot be read and their org had no feature binding before them」。feat/b の `unreadable` は V7 でも同じ |

### AC の確かめ直し(HEAD a7cbdd87)

cycle 4 の `mergeorder.go` と `org_merge_order.go` の hunk は、help の 1 か所(`org_merge_order.go:51`)、`collectMergeTargets` の doc と本体、`mergeLedgerNotes` の注記、`mergeOriginRepo.check` の doc と注記だけで(evidence の `git diff -U0` の hunk の一覧)、`listOriginBranches`、`lookupMergePRs`、`pickMergePR`、`classifyMergeTarget`(`:826`)、`classifyGatePassed`、`orderMergeEntries`、`readMergeLedger` は変わっていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | 満たす(実機でも確認) | V5 で、gate の head と base が origin と同じ feat/a・feat/e・feat/x が `ready`。rebase と regate は cycle 2 の K17、cycle 3 の Q1 のまま(分類の経路は変わっていない) |
| AC2 | 満たす(cycle 2・3 の実機確認のまま) | 読める記録を最新のもので上書きする経路は変わっていない。V の台帳には failed・no_gate・no_remote を置いていない |
| AC3 | 満たす(cycle 1 の実機確認のまま) | 結びつきの読み方(`EventScopeReserved` の枝)は変わっていない |
| AC4 | 満たす(実機でも確認) | 上の表の 2 行目。AC4 の最後の文「読めない記録は、打った org のその時点の結びつきのブランチに付く」は feat/b(V5・V7)のとおり今も成り立つ。cycle 1 の Coverage gaps に残っていた、event の名前と reason の有無が合わない記録を、feat/f で初めて CLI から打った |
| AC5 | 満たす(実機でも確認) | V1・V2・V4 は PR の欄を空にして注記し、終了コード 0。V3 は merged の feat/a を隠して「1 branch(es) whose PR is merged or closed are not shown」と注記する。V5 は gh がない注記を出す |
| AC6 | 満たす(実機でも確認) | V5 の並びは `ready`(gate の時刻の 01:00:11 の feat/a、:15 の feat/e、:20 の feat/x)、`unreadable`(manifest の順に feat/c、feat/e2、feat/f、feat/b)で、`mergeClassOrder` のとおり。ready が 3 本なので「3 branches are ready, but merging one moves origin/main ...」の注記が出る |
| AC7 | 満たす(cycle 2 の実機確認のまま) | `listOriginBranches` は変わっていない。V のどの回も ls-remote は 1 回 |
| AC11 | 満たす(cycle 1・2 の実機確認のまま) | 分類の経路は変わっていない |
| AC8 | 満たす(実機でも確認、PR の欄のほか) | V6(`--json`)は V5 のテキストと同じ 7 本を同じ順で出し、gate の記録(event、readable、head、base、base_ref、reason、ts)、binding、notes を持つ。gh なしの回なので `pr` の欄は出ない(PR の欄つきは cycle 3 の Q2) |
| AC9 | 満たす(実機でも確認) | V1〜V7 の前後で、manifest の sha256(7a192592…)・行数・バイト数(13 行、3381 バイト)・台帳のディレクトリの中身(`manifest.jsonl` だけ)が同じ。work と origin の refs の digest も同じで、state・work・origin・HOME の下に marker より新しいファイルは 0 件。V8 の gate は台帳の写しで打ち、その写しも打つ前と sha256 が同じで、増えたファイルはない |
| AC10 | 一部(D8・D9、LOW) | skill の 4 面は `cmp` で同一、`check-skill-sync.sh`(13 skill in lock-step)と `check-sync.sh`(DRIFTED 0)が通る。中身のずれは下の表の D8 と D9 |

計画の逸脱の記録: Progress の 160 行は、(1) 注記に改名と別の repository の 2 つの直し方を並べ、分類は PR なしのままで、正規の名前を gh で引かないこと、(2) 読めない記録を、結びつき、その org の前の読める記録(最新のもの)、の順に付け、どちらもなければ付けないこと、(2) が計画の Scope と C8 の「前に結びつきがなければどのブランチにも付けない」を広げる逸脱であることを書いている。813d6d2d と 1052a12a、V1〜V7 の結果はこの記述と合う。Scope の 23 行と C8 の 82 行は承認時の文のまま(digest が承認時と同じことと合う)で、cycle 2・3 の逸脱と同じく Progress に置かれている。AC4(91 行)は付け先の主な規則だけを言い、今も正しい。

仕様: FR-8 の 7c-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`)の「読めない記録は、打った org のその時点の結びつきのブランチに付ける(gate は結びつきのブランチでしか打てないため)」は、要約として今のままでよく、直す必要はない。結びつきを先に使う規則は cycle 4 でも変わらない。新しい付け先は org が完全な結びつきを持たないときだけ使われ、`ralph org gate` はそういう org を断る(`gateOrgLife`、`internal/org/gate.go:333`、V8)ので、gate が書いた台帳には起きない。注は前から結びつきのない場合を書いておらず(「どこにも付けない」も書いていない)、「詳細は ... 計画に書いた」で終わり、計画の Progress 160 行にその決定がある。食い違う記述はない。`ready` の定義と分類の順も変わらない。

### 静的解析(cycle 4)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | local verifier の全項目が OK、`check-sync.sh` PASS(IDENTICAL 167、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh` 13 skill in lock-step、`check-template-purity.sh`、tech-debt の plan 参照、golang(`gofmt: ok`、`go vet ./...` と `staticcheck ./...` は出力なし、`golangci-lint run ./...` 0 issues)、`secret-scan-branch: scanned ac34ccf3..a7cbdd87 against origin/main: clean` |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod は `go 1.25.8`、手元は go1.26.0。cycle 4 で使い始めた API はない(`cmp.Or` は cycle 1 から使っている) |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `scripts/check-skill-sync.sh`、`scripts/check-sync.sh`(単独で) | pass | 13 skill(s) in lock-step。IDENTICAL 167、DRIFTED 0 |
| `/org` skill の 4 面の `cmp` | pass | 同一 |
| `git diff --check ac34ccf3...HEAD`、`git diff --check 999e0394..HEAD` | pass | どちらも rc 0 |

### 文書のずれ(cycle 4)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| help(`internal/cli/org_merge_order.go:51-55`) | yes | V9 で、999e0394 との差は追加した 5 行だけ。「the note also says to point origin at the new name with git remote set-url origin if it was renamed」は V1 の注記と合う |
| skill の `merge-order` の行(`.claude/skills/org/SKILL.md:183`)と「merge してよい順」節の gh の項目(`:928-941`)、4 面とも同じ | yes | 改名なら `git remote set-url origin`、そうでなければ `gh repo set-default <origin の owner/repo>`、改名と別の repository を見分けられないこと、が注記の文と V1・V2 に合う |
| README の org の節(`README.md:262`) | yes | 足した 1 文は help と同じことを言う |
| `mergeOriginRepo.check` の doc(`internal/org/mergeorder.go:619-629`)、`collectMergeTargets` の doc(`:312-324`)、`mergeLedgerNotes` の doc(`:381-382`) | yes | 付け先の順(結びつき、前の読める記録、なし)と 2 つの直し方が、コードと V1〜V7 に合う |
| `internal/org/mergeorder.go` の先頭の説明(`:28-40`) | yes | gh の repository の決まり方と突き合わせの 3 つの穴を書き、注記の直し方は言い切らない |
| 仕様 FR-8 の 7c-1 の注(`docs/specs/2026-10-07-org-multi-org-director.md:75`) | yes(情報) | 上の「仕様」の段落のとおり、要約として正しい |
| tech-debt の repository の突き合わせの行(`docs/tech-debt/README.md:237`) | 本文は yes、Related の指し方がずれる(D9、LOW) | Debt・Impact・Why deferred・Trigger に足した改名の場合は 813d6d2d と V1・V3 に合う。Related の「`docs/reports/cross-review-triage-org-merge-order.md` (the cycle 3 section at the top, ACTION_REQUIRED #1)」は、triage の先頭の節が cycle 3 の今は正しいが、次の cross-review(cycle 4)が triage を書き換えて最新の cycle を先頭に置くと、別の節の行を指す。self-review の C4-4 と同じ所見。直し方はその報告の表のとおり(「cross-review cycle 3 of this plan, the renamed-repository finding」のように cycle と中身で指す)で、cross-review の前に /sync-docs が直せる |
| tech-debt の merge-order の残りの行(`docs/tech-debt/README.md:239`) | ずれ(D8、LOW) | cycle 4 が走ったので古くなった。先頭のセルの「Cycle 3 was the last pipeline run (the cap was raised from 2 to 3 for cross-review cycle 2's ACTION_REQUIRED)」と、Why deferred の「Found at the last pipeline run (cycle 3 of 3, after the cap was raised from 2): fixing any of them needs a fourth pipeline run」は偽(上限は 4 で、4 回目が走った)。Related の報告の節は cycle 3 まで(`section Cycle 3: C3-1 to C3-5`、`section Cycle 3: D5 to D7`)で、cycle 4 の節がない。self-review の C4-1 と C4-2 を載せる項目もない。self-review の C4-3 と同じ所見で、直し方はその報告の「register の古くなった所」の表にある。/sync-docs が直す |
| tech-debt の 236 行(途中まで書かれた最後の行)と 238 行(`--json` の lookup の結果) | yes | 壊れた行の扱いと PR の lookup の結果は cycle 4 で変わっていない |
| AGENTS.md の repo map | yes | cycle 4 で新しいファイルはない |
| 計画の AC の checkbox と Progress | yes | AC1〜AC11 は `[x]`、「PR created」だけが `[ ]`。Progress の最後の項目(160 行)が cycle 4 の決定を書く |
| cycle 3 のずれ D5〜D7 | 直った(D5 は cycle 4 で再び古くなった) | f269c0db。D6 は 238 行が上限ぶんの fork の PR の場合を書き、D7 は `MergeOrderEntry.PR` の doc(`mergeorder.go:168-171`)が同じ場合を書く。D5 の 239 行は cycle 3 に合わせて直ったが、cycle 4 が走ったので D8 になった |

### 観察

- self-review の C4-1 を binary で再現した。V5 の feat/c の Detail は「the latest gate record (gate_failed at 2026-10-14T01:00:13Z) cannot be read: run ralph org gate --org-id org-c again in the feature worktree (the gate runs only while the feature org runs)」で、V8 でそのコマンドを台帳の写しで打つと「org: gate: org_id "org-c" is not running and holds no split plan feature binding: name the org_id of a feature org that ralph org start --plan started (ralph org status lists them)」で終了コード 1、写しは変わらない。org が走っていても、結びつきがなければ `gateOrgLife` の 1 つめの枝(`gate.go:344-347`)で断る(この枝は打っていない、読んだだけ)。skill の 903 行は「gate を打てるのは、機能の org が結びつきを持って走っている間だけ」と正しく書き、Detail の括弧書きは結びつきを言わない。ブランチは `unreadable` のままで `ready` にはならないので、誤って merge に進む向きではない。self-review が直さずに 239 行の (s) に載せると決めた所見で、verify の所見としては数えない
- V2 の注記は、fork の作業場所で gh が upstream を読むよくある場合にも「if acme/app was renamed to upstream-org/app」と言う。owner の変わる移管も GitHub が転送するので、仮定の文として誤りではない。self-review の「所見にしなかったもの」と同じ判断
- V1 と V4 の分類は同じ(PR なしの `ready` 3 本と `unreadable`)。計画の決定のとおり、(1) は注記だけの直し
- ready が 2 本以上のときの注記は、cycle 3 と同じく `--org-id` を付けない「run ralph org gate again」のまま(self-review の C3-2、239 行の (o))

### Coverage gaps(cycle 4)

- テストは走らせていない(/test の担当)
- 直し (1) の前提(GitHub で改名した repository に origin の古い URL で git が届き、`gh pr list` の PR の url は新しい名前になること)は観察していない。V3 は get-url の答えを差し替えて `git remote set-url origin` を模しただけで、ls-remote はローカルの bare origin を読んでいる
- 走っているが結びつきのない org での `ralph org gate` の断り(`gate.go:344-347`)は打っていない。V8 は走っていない org の場合
- AC2 の failed・no_gate・no_remote、AC7、AC11 は cycle 4 では打っていない。経路が変わっていないことを hunk の一覧で確かめ、前の cycle の R・K・Q の結果を引いた
- 足すと確信がいちばん増える確認: GitHub で改名した本物の repository(または改名の転送が効く既存の repository)で、origin を古い URL のまま HEAD の binary で `ralph org merge-order` を 1 回打ち、注記が出ること、`git remote set-url origin` のあとに PR が使われることを見る。読むだけだが GitHub に出るので、この verify では打っていない

### 判定(cycle 4)

- Verdict: pass
- Verified: cycle 4 の 2 つの直し(repository が違うときの注記の 2 つの直し方、結びつきのない org の読めない記録の付け先)をコードとテストの名前で辿り、HEAD a7cbdd87 の binary で V1〜V3・V5・V6・V8・V9 として再現した。直す前の 999e0394 の binary(V4・V7)は同じ台帳で、古い注記を出し、feat/c・feat/e2・feat/f を古い gate_passed に戻して `ready` と出した。AC1・AC4・AC5・AC6・AC8・AC9 を同じ binary で確かめ直し、AC2・AC3・AC7・AC11 は経路が変わっていないことを diff で確かめた。静的解析はすべて通った(go1.25.8 の build と vet を含む)。計画の逸脱は Progress の 160 行に記録されている。仕様 FR-8 の注は要約として直さなくてよい
- Partially verified: AC10。tech-debt の 239 行が cycle 4 で古くなった(D8、LOW)。237 行の Related が triage の節を位置で指し、次の cross-review で指し先がずれる(D9、LOW)。どちらも /sync-docs が直す文書の直しで、動きは変わらない
- Not verified: テストの実行(/test)、本物の GitHub での改名の転送と gh の PR の url、走っているが結びつきのない org での gate の断り
- Findings: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2(D8、D9。D8 は self-review の C4-3、D9 は C4-4 と同じ所見)。情報は 4 つ(C4-1 の再現、V2 の注記の読み方、仕様の注を直さなくてよいこと、計画の Scope と C8 の本文が承認時のままで逸脱は Progress にあること)
