# Walkthrough: org-merge-order(7 段目の残りの前半、7c-1)

- Date: 2026-10-14
- Plan: docs/plans/active/2026-10-14-org-merge-order.md(/pr で archive に移す)
- Branch: feat/org-merge-order(base ac34ccf3。この walkthrough を書いた時点で 47 commits、46 files、+14165/-73)
- 読む範囲: 差分の大半は `docs/evidence/` と `docs/reports/` で、読むのは各報告の要約だけでよい。報告・evidence・insight を除くと 19 files、+4775/-73 で、そのうちテストが `internal/org/mergeorder_test.go`(+1950)と `internal/cli/org_merge_order_test.go`(+1110)。ほかに `/org` skill の 4 面、README、仕様 FR-8 の注、AGENTS.md、quality-gates の 2 面、tech-debt
- 後工程は 4 回まわした(上限を 3、4 に上げた。`cycle-count.json` は 2 のまま)。cross-review の指摘は cycle 1 から 3 で 1・1・2 件、cycle 4 で 0 件

新しいファイルは `internal/org/mergeorder.go`(台帳と origin と gh を読み、分類して並べる)、`internal/cli/org_merge_order.go`(その CLI)と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 台帳から対象のブランチを集める(`internal/org/mergeorder.go`、d27892f9、1052a12a)

- `readMergeLedger` は manifest をロックなしで読み、壊れた行の位置を持つ(AC9: 書かない、ロックも取らない)
- `collectMergeTargets` は org 全体の event を順に歩く。完全な結びつき(`scope_reserved`、`reserve.go` と同じ読み方)はそのブランチを対象にし、org の最新の結びつきになる。disband 済みの org の結びつきも数える
- 読める gate の記録(`gateRecordFromDetails` が読め、event の名前と reason が合う)は `branch=` のブランチに付く。読めない記録は、その org の前の完全な結びつきのブランチに付き、結びつきがなければその org の最新の読める記録のブランチに付く(cycle 4 の 1052a12a。計画の Scope と C8 を広げた逸脱で、Progress checklist に記録した)。どちらもなければ `unattached` に数えて注記する
- 読める gate の記録のあとに壊れた行があれば `corruptAfterGate` を立てる。読めない証拠は前の `gate_passed` に戻さない(C8)

## 2. origin を読む(`listOriginBranches`、`parseLsRemote`)

- `git ls-remote --symref origin HEAD 'refs/heads/*'` を 1 回、`GIT_TERMINAL_PROMPT=0` と 30 秒の上限で打つ。今の既定のブランチは origin の HEAD の symref が名指すもの(C7)。gate の記録の base_ref の名前からは決めない
- 打てない、時間切れ、HEAD の symref がない・ブランチを名指さない、のどれでも、何も分類せずにエラー(CLI は終了コード 1、AC7)。対象のブランチがなければ git を打たない

## 3. PR を引く(`lookupMergePRs`、`lookupMergePR`、`pickMergePR`、`mergeOriginRepo.check`)

- gh が PATH にあれば、ブランチごとに `gh pr list --head <branch> --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository` を `GH_PROMPT_DISABLED=1` と 15 秒の上限で打つ。fork の PR(`isCrossRepository`)を外し、番号の一番大きい PR を採る(C5)
- `--limit 1000`(d7599d82、cross-review cycle 2): gh の既定の上限 30 を同じ名前の fork の PR が埋めると本物の PR が落ちる。gh は `--head` 単独では作成の新しい順に辿るので、結果に fork でない PR があればそれが最新。上限ぶん返ってすべて fork なら、そのブランチの PR は使わずに注記する
- repository の確かめ(15d4b947、cross-review cycle 1): gh は `gh repo set-default` の既定か remote の名前の優先順で repository を決めるので、fork の作業場所では upstream を読みうる。gh が PR を返したら `git remote get-url origin` を 1 回だけ打ち、PR の url の owner/repo と origin の owner/repo を大文字小文字を無視して比べる。合わなければ PR をすべて捨て、gh をそれ以上打たない。`--repo` は渡さない(host の解決が GH_HOST などに頼り、ssh の別名を解かない)
- 合わないときの注記(813d6d2d、cross-review cycle 3)は、改名なら `git remote set-url origin` で origin を向け直す、そうでなければ `gh repo set-default <origin の owner/repo>` を打つ、の両方を並べる。改名した vuejs/vue-next で、古い注記の set-default を gh が断ることを本物の gh で確かめた(test の cycle 4)
- 確かめの穴(host を比べない、owner/repo を読めない origin では比べない、gh が PR を返さなければ何も言わない)は help・skill・README と tech-debt の行に書いた
- 同じ理由の gh の失敗は 1 つの注記にまとめる(`mergeGHFailures`、5 本まで名指して残りは数)

## 4. 分類と順(`orderMergeEntries`、`classifyMergeTarget`、`classifyGatePassed`、`mergeGateCommand`)

- 分類は最初に当てはまるもの: merged・closed(PR の state)→ unreadable → no_gate → failed → no_remote → regate(gate のあとに push)→ wrong_base(PR の統合先が今の既定でない)→ rebase(既定が動いた、または gate が別のブランチで確かめた)→ ready
- 並びは ready、rebase、wrong_base、regate、unreadable、failed、no_remote、no_gate、`--all` のときだけ merged と closed。同じ分類の中は台帳の古い順
- gate を打ち直す Detail は `ralph org gate --org-id <org>` を名指し、最新の結びつきの org、なければ gate の記録の org を使う(c885fcab)。どの Detail も「(the gate runs only while the feature org runs)」で終わる。ready が 2 本以上なら、1 本 merge すると残りは rebase になると注記する

## 5. CLI(`internal/cli/org_merge_order.go`、`internal/cli/org.go`、65a0d404)

- `ralph org merge-order [--json] [--all]`。`--org-id` は断る(台帳全体を読むので)。見張りがなくても打てる(6b の関門をかけない)
- テキスト(`printMergeOrder`)は、1 行目に `origin/<既定> <先頭 12 字> (listed <時刻>)`、次に `RANK CLASS BRANCH PR GATE_HEAD GATE_BASE ORG DETAIL` のタブ区切りの表、最後に `note:` の行。台帳・git・gh から来た値はすべて `inboxField` を通し、細工したブランチ名や URL で行や列を壊せない。`--json` は `MergeOrderResult` をそのまま出す。分類に関わらず終了コード 0、manifest が読めないか ls-remote が打てなければ 1 で、stdout には何も出さない
- `org_legacy_ledger.go`: merge-order を読むだけの動詞(古い worktree の台帳には注記だけ出す)に入れた。同じ comment の書き換えの動詞の一覧に、7a・7b の `approve` と `gate` も足した

## 6. 文書

- `/org` skill の 4 面(動詞の表と「merge してよい順」の節)、README の段落、仕様 FR-8 の注(7c-2 と 8a への送り)、AGENTS.md の repo map、quality-gates の 2 面
- tech-debt: disband 済みの org のブランチを ready に戻す手がない(持ち主は 8a の director)、ssh の askpass を `GIT_TERMINAL_PROMPT=0` が止めない、gh なしでは merge 済みで消えたブランチが no_remote に残る、読みかけの最後の行を壊れた行と数える、repository の確かめの穴、JSON で「gh の失敗」と「PR なし」を見分けられない、各 cycle の LOW と生き残りの mutant
