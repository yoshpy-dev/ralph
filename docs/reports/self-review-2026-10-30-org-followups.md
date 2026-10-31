# Self-review report: org-followups

- Date: 2026-10-31
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(issue #270、D1〜D5、AC1〜AC8、スライス S1 de7dd522・S2 dc589374・S3 4d78c28e・S4 4a44ef30・S4b ef1663e9・S5 2d86a71e)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 1cc84ffa、HEAD d58f5652、32 ファイル、+5263/-79)。台帳の follow-up の store(`internal/org/followup.go`)、merge-order の follow-up の節(`internal/org/mergeorder_followup.go` と `mergeorder.go` の差し込み)、動詞(`internal/cli/org_followup.go`)、`ralph status` の予約と follow-up(`internal/cli/status_followup.go` と `status.go` の差し込み)、`lockfile.go` のコメント、テスト、`/org`・`/director` の skill と写し、README、AGENTS.md、仕様の注。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L9(LOW)。
- Run 2: 2026-11-01、HEAD c9cb588f、パイプラインは cycle 1 のまま。run 2 の所見は R2-1〜R2-4(すべて LOW)で、run 1 の ID とは別の番号にした。
- Cycle 2: 2026-11-01、HEAD 32c73433、パイプラインの cycle 2(cross-review cycle 1 の ACTION_REQUIRED #1 の直しのあと)。所見は C2-1・C2-2(どちらも LOW)。
- 構成: 先頭が cycle 2 の節、次が run 2 の節、最後が run 1 の節。run 2 の節は、見出しに「(run 2)」を足し、`Merge:` の行の名前を「Run 2 の判定」に変えたほかは、run 2 のままにした。run 1 の節は run 2 のときに直したまま。ファイルの `Merge:` の行は cycle 2 の節の 1 行だけ。

## Cycle 2(HEAD 32c73433)

### Scope(cycle 2)

- ブランチ全体: `git diff origin/main...HEAD`(base 1cc84ffa、45 ファイル、+6828/-128)。
- 重点: run 2 の report(f93a35f6)のあとの直し。F3 aae31317(R2-1〜R2-4 の直し。run 2 のあとで、どの self-review もまだ見ていない)、F4 a4cab358(merge-order の follow-up の表の SOURCE の列、help、`/org` skill と写し、固定のテスト、`status_followup.go` のコメント)、F5 25c9fa77(follow-up だけの経路の `gh_available`: `lookupFollowupMerges` が gh を見つけたかを返し、`mergeOrderFollowups` が `GHAvailable` に OR で足す。欄と `MergeOrder` の説明、テスト)、文書の commit(sync-docs 9717f0e8 の tech-debt の注と行、quality-gates と `/org` skill の見張りのいらない動詞、計画の a5372aaf・32c73433、triage 6c9281aa)。
- DAG: `git log --format='%h parent=%p'` で、F4 は verify(d2890dbf)と計画の再承認(a5372aaf)のあと、F5 は triage(6c9281aa)の直接の子と確かめた。F4 と F5 はどちらもこの節が初めて見る。
- 見ていないもの: テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。走らせたのは `go run ./cmd/ralph org merge-order --help` の行の幅の計測と、写しの `cmp` だけ。

### Run 2 の所見の扱い(cycle 2)

| ID | 扱い | 根拠 |
| --- | --- | --- |
| R2-1 | 解決 | 「ready と waiting で割れたら merge-order、merge-order が `unknown` なら理由を添えて status」の文が、status・`followup`・`followup add`・merge-order の help(`internal/cli/status.go`、`org_followup.go`、`org_merge_order.go:126-128`)、README の 132 行、AGENTS.md の 91 行、`/org`・`/director` skill と写し、仕様の注、`mergeorder_followup.go:26-30` の package の注にある。`director/SKILL.md:1108-1115` は折り返し直した |
| R2-2 | 解決 | `statusFollowupJSON` の説明(`internal/cli/status_followup.go:33-42`)が「worktree_cleaned なら片付けの時刻」と書き、`MergeOrderFollowups`(`mergeorder_followup.go:88-99`)は「same outer keys」と書いて項目の違い(`pr` と `merged_event`、`merged_at` の意味)を挙げる |
| R2-3 | 解決。ただし F5 で数が古くなった(C2-1) | `docs/tech-debt/README.md:411` の行(1291 行から 1322 行) |
| R2-4 | 解決 | `moFollowupPR`(`internal/org/mergeorder_followup_test.go:35-42`)は `ghPullRequest` を埋め込むだけで自分の欄を持たず、`moMergedPR` は埋め込んだ `MergedAt` に入れる。説明が理由を書く |

### 依頼された点の確認(cycle 2)

- `GHAvailable` は、どちらかの検索が gh を見つければ true で、follow-up の検索が false に戻すことはない: `res.GHAvailable = res.GHAvailable || ghFound`(`internal/org/mergeorder_followup.go:135`)。`MergeOrder` は対象のある経路で `lookupMergePRs` の値を入れたあとに `mergeOrderFollowups` を呼ぶ(`internal/org/mergeorder.go:343`、`:350`)。対象のない経路(`:332`)では初期値の false に OR する。記録が読めない早い return(`mergeorder_followup.go:121-125`)と `p.Followups` が false の return は値に触らない。
- gh の失敗となしの見分けは `lookupMergePRs`(`mergeorder.go:673-729`)と同じ: `errors.Is(err, exec.ErrNotFound)` で「見つからない」にして止め(`mergeorder_followup.go:199-202`。前の枝で答えていても false に戻すのは、`lookupMergePRs` が同じ場合に false を返すのと同じ)、ほかの失敗(時間切れを含む)は見つけたに数え(`:204-208`)、origin のものでない PR(`checkFollowupPRs` の why)も見つけたに数える(`lookupMergePRs` の `return true, withNotes(why …)` と同じ)。ctx が終わった場合だけ違う(`lookupMergePRs` はエラーを返して結果が捨てられ、follow-up 側はそれまでの値を残す)が、関数の説明の「gh answered for a branch … false when no branch was looked up」と合う。
- テストは直しを区別する: 「gh fails for the branch」と CLI の `TestOrgMergeOrder_FollowupOnlyJSONSaysGHAvailable` は直す前のコードで落ちる。「gh goes missing after an answer」は `ghFound = false` を消すと落ち、`TestMergeOrderFollowup_LookupNeverClearsTheEntriesGHAvailable` は `||` を `=` にすると落ちる。CLI のテストは `gh_available` を JSON の生の値(`"true"`)で見て、gh の呼び出しが follow-up の 1 回だけであることも見る。
- JSON と help: merge-order の help は `gh_available` を書かない(main からそう)ので、食い違いは生まれていない。`/org` skill と README も同じ。`MergeOrderResult.GHAvailable` の説明(`mergeorder.go:192-197`)は、2 つの検索と「after-branch も聞く相手」を書き、false の場合をコードと同じに書く。
- ほかの出力と終了コード: F5 はテキストの出力(`printMergeOrder` は `GHAvailable` を読まない)、注記、終了コードを変えない。`GHAvailable` を読むもう 1 つの場所 `cleanupSteps`(`internal/cli/org_cleanup.go:256`)は `Followups` を立てない呼び出し(`org_cleanup.go:167`)の結果を読むので、値は前と同じ。
- F4: 表の見出しは `ID ISSUE AFTER_BRANCH STATE SOURCE PR DETAIL`(`internal/cli/org_merge_order.go:280`)で、値は `inboxField` を通る。`printStatusFollowups` の説明(`status_followup.go:170-178`)は「PR の列がない」に直した。テストの見出しの定数の説明が名指す `statusFollowupHeader`(`status_followup_test.go:62`)はある。help、skill の 4 つの写し、固定のテストが同じ列を挙げ、古い並び(`STATE`・`PR`・`DETAIL`)はコードと skill に残っていない(`docs/evidence/verify-2026-10-30-org-followups.md:190` は F4 の前の binary の出力の記録)。
- 写し: `/org` と `/director` の skill の 4 つの写しは `.claude/skills/` とバイト単位で同じ。`docs/quality/quality-gates.md` の見張りの行(80 行)は `templates/base/` と同じで、ほかの違いは `KNOWN_DIFFS` の既存の行だけ。
- 見張りのいらない動詞の一覧: `followup` の動詞は `NewWatchGate` を呼ばない(`internal/cli/org_followup.go` に gate の配線はない)ので、quality-gates と `/org` skill の追記はコードと合う。

### Findings(cycle 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1 F5 は、同じ PR の sync-docs(9717f0e8)と F3 が台帳に書いた 3 つの引き金に当たったが、払わず、持ち越しの注もない。どれも引き金を「次の変更」にしている: (a) `mergeorder.go` の大きさの行の「The next change to `internal/org/mergeorder.go`」、(b) `mergeguarantee.go` の行の「at the next change of `mergeorder.go`, to move the runners」、(c) 7 つのテストの穴の行の (4)「the next change to `checkFollowupPRs` or `lookupFollowupMerges`」。F5 は `mergeorder.go` のコメントと `lookupFollowupMerges` の本体を変えた。加えて、(a) と (b) の行の数が古くなった: どちらも `mergeorder.go` を 1322 行と書くが今は 1328 行、(a) は `mergeorder_followup.go` を 360 行と書くが今は 370 行。sync-docs の pass は同じ形の行に「2026-11-01: feat/org-followups hit this trigger … Carried over」の注を足しているので、F5 の分だけ書き方がそろわない。 | `docs/tech-debt/README.md:303`(「1291 to 1322 lines」)、`:411`(「to 1322」「(360 lines)」、引き金の欄)、`:412`(テストの穴 (4) と引き金)、`internal/org/mergeorder.go`(1328 行)、`internal/org/mergeorder_followup.go`(370 行)、F5 25c9fa77 | cycle 2 の sync-docs で、303 行と 411 行の数を 1328 行と 370 行に直し、3 つの行に F5 が当たって払わなかったことを 9717f0e8 と同じ形で一言足す。(c) は、`checkFollowupPRs` の中で ctx を止めるテストを 1 つ足して払ってもよい(`TestMergeOrderFollowup_InterruptedLookupIsNoError` の形で、`git remote get-url origin` のときに `cancel()` する)。 |
| LOW | readability | C2-2 F5 のコメントの 2 か所。(1) package の注に足した「The section changes neither the entries, nor their lookups, nor MergeOrder's error; it only sets GHAvailable when its lookup found gh」の「only」は、記録が読めないときに `res.Notes` に注記を足すこと(同じファイルの `mergeOrderFollowups`)を数えない。F5 の前は「only」がなく、列挙した 3 つについての文だったので正しかった。(2) `MergeOrder` の説明は、「returns at once … (the follow-ups below may still run gh). Otherwise it」の行で折り返しが途切れたまま次の行に続く。「returns at once」と「may still run gh」(1 本 15 秒まで)も並ぶと読みにくい。同じ種類の小さな点として、F4 が直した merge-order の help の 1 行(「followup add\` recorded, under follow-ups: (ID, ISSUE, AFTER_BRANCH, STATE,」)は 74 桁で、この help のほかの行(72 桁まで)より長い。 | `internal/org/mergeorder_followup.go:24-25`、`:123`、`internal/org/mergeorder.go:294-296`、`internal/cli/org_merge_order.go:103-104` | (1) は「it sets only GHAvailable, when its lookup found gh, and adds a note when the record cannot be read」のように注記も挙げる。(2) は段落を折り返し直し、「With no target it skips the ls-remote and the entries' gh calls」のように「at once」を外す。help の行は「STATE,」を次の行に送る。 |

### Positive notes(cycle 2)

- F5 は、`lookupMergePRs` の `available` と同じ判定を follow-up の検索に持たせ、関数、`mergeOrderFollowups`、`MergeOrder`、`GHAvailable` の 4 つの説明を同じ commit で直した。直しは結果の 1 つの欄だけに閉じている。
- テストは 2 層にある。org 層の表は、gh の答え、失敗、なし、答えのあとのなし、記録なし、done だけの 6 つで、どれも entries が空であることと gh の回数まで見る。CLI 層は JSON の生の値を見る。
- F5 のテストとコメントは「cross-review cycle 1's ACTION_REQUIRED #1」と cycle の番号を付けて triage を指すので、cycle 2 の triage が同じ番号を使っても取り違えない。
- F4 は列の追加を、表の印刷、help、skill の 4 つの写し、固定のテスト、テストの列の数(6 から 7)、`printStatusFollowups` の説明まで同じ commit で合わせた。

### Tech debt identified(cycle 2)

この report では `docs/tech-debt/README.md` を変えていない(依頼の範囲が report と insight event だけのため)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-1: 台帳の 3 つの行の数と持ち越しの注 | LOW | 台帳の行は cycle 2 の sync-docs が直す範囲 | cycle 2 の `/sync-docs` | この report の C2-1 |

C2-2 を直さずに PR に進むなら、台帳の org-followups の行のどれかに一言足すか、1 行にまとめる。

### Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2。cross-review cycle 1 の ACTION_REQUIRED #1 は F5 で直り、`GHAvailable` は 2 つの検索のどちらかが gh を見つければ true で、follow-up の検索が false に戻すことはなく、gh の失敗となしの見分けは `lookupMergePRs` と同じ。run 2 の R2-1〜R2-4 は F3 で直った。C2-1 は台帳の数と注、C2-2 はコメントと help の行で、挙動は変えない)
- Follow-ups:
  - C2-1 は cycle 2 の `/sync-docs` で台帳を直せば足りる。C2-2 はコメント 2 つと help の 1 行で、記録だけの直しの例外には入らない。
  - テストと静的検査はこの report では走らせていない(`/verify` と `/test` の範囲)。

## Run 2(HEAD c9cb588f)

### Scope(run 2)

- ブランチ全体: `git diff origin/main...HEAD`(base 1cc84ffa、37 ファイル、+5731/-97)。
- 重点: run 1 の report(8e87b003)からの直し `git diff 8e87b003..HEAD`(29 ファイル、+836/-446)。F1 a77e7d18(org 層: M1 のコード、L1、L2、L3 の struct、L4、L8、L9 のテストの分割)、F2 24716dac(CLI と文書: M1 の文書、L3 の CLI、L5、L6、L7)、c9cb588f(仕様の注、計画の D5、tech-debt の 2 行)。
- 見ていないもの: テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。走らせたのは、R2-4 の確かめに使った scratchpad の 20 行の `go run` だけ。

### Run 1 の所見の扱い(run 2)

| ID | 扱い | 根拠 |
| --- | --- | --- |
| M1 | 解決。残る 1 つの形は文書と tech-debt に送った | `FollowupRecordedMerge`(`internal/org/followup.go:470`)は、記録より前の `pr_merged` の head を集め(`mergedBefore`)、記録以後の `worktree_cleaned` のうち同じ head のものを数えない。別の head(ブランチの使い回し)は数える。2 つの書き手の head は、どちらも PR の `headRefOid` になる(`pr_merged` は gate の head で、`merge.go:546` と `merge_run.go:420` が PR の head と同じことを確かめる。`worktree_cleaned` は `org_cleanup.go:273`・`:309` の `e.PR.HeadRefOid`、merge の続きの片付けは `merge_run.go:291` の `res.Head`)。同じ merge なら一致する。全件を集めてから判定するので、ファイルの中の順は答えを変えない(`followup_recorded_merge_test.go` の「the earlier merge's pr_merged after its worktree_cleaned in file order」)。テストの期待は反転した(「a pr_merged before the add and the worktree_cleaned of its head after it」は waiting)。GitHub で記録より前に merge された PR(`pr_merged` がない)の片付けは今も数える。この形は help 4 つ、skill 2 つ(写し 4 つずつ)、README、AGENTS.md、仕様の注に「割れたら merge-order の答えを使う」と書き、`docs/tech-debt/README.md` の行(片付けが `mergedAt` を記録する直し、#271 を引き金に)に送った。ただし R2-1 と R2-2 |
| L1 | 解決 | `ghBranchPRs`(`internal/org/mergeorder.go:970`)が gh の引数、`-R` を最後に足すこと、`GH_PROMPT_DISABLED=1`、timeout、「timed out after」の条件(timeout が止めて ctx は止めていない)、JSON でない出力の文を 1 か所に持ち、`lookupMergePR`(`:951`)と `lookupFollowupPRs` が呼ぶ。引数の並びとエラーの文は前と同じ。`limitFullOfForks` と `limitFull` は、どちらも fork を含む全件の数で決める(前と同じ)。`prchecks.go:90` の `lookupMergePR` の呼び出しは形が変わらない |
| L2 | 解決 | unknown の文は `FollowupAddedTime`(`followup.go:510`)、ready の文は `FollowupReadyDetail`(`:523`)にまとめ、status(`internal/cli/status_followup.go:155`、`:161`)と merge-order(`mergeorder_followup.go:284`、`:291`)が呼ぶ。`added_at` を 2 回 parse することは残るが、文は 1 か所になった |
| L3 | 解決(R2-2 を除く) | merge-order の JSON の `followups` は `MergeOrderFollowups`(`mergeorder_followup.go:93`。`items`・`corrupt_lines`・`ignored_events`・`error`、`Path` は `json:"-"`)で、status の `statusFollowupsJSON` と外側のキーが同じ。両方のテストが同じキーの組を固定する(`mergeorder_test.go:2929`、`TestMergeOrderFollowup_JSONCarriesEveryFieldAndANullPR`、`TestOrgMergeOrder_NoFollowupPrintsNoSectionAndAnEmptyJSONList`、status 側の `TestStatusCmd_FollowupJSONOnBothPaths`)。`MergeOrderResult.Followups` を読むのは CLI の 3 か所だけで、`ralph org cleanup` は Followups を立てない |
| L4 | 解決 | `seat.go` の 2 つの event の説明に、`ralph status` が読むことと、`branch=` が最初の語で `head=` を読むので語の順とキーが約束であることを書いた。org のテストの `pr_merged` は、本物の `recordMerge` が書いた行を読み戻して作る(`followup_recorded_merge_test.go` の `prMergedEvent`)。CLI のテストの Details は同じ形の手書きのままだが、`via=cleanup` になり、本物の形は `internal/cli/org_merge_test.go:318` と `internal/cli/org_cleanup_test.go:150` が固定する(ヘルパーのコメントのとおり) |
| L5 | 解決 | (a) `reopen PR #%d`(`mergeorder_followup.go:312`)。(b) 「an added event that add would refuse」(`internal/cli/org_followup.go:242`)。(c) done の RunE が台帳を解決する前に `org.ValidateFollowupNote` を打つ(`org_followup.go:181`)。古い台帳の断りと注記より先に出ることを `TestOrgFollowup_LegacyLedgerGuard` が見る |
| L6 | 解決 | `FollowupMergeUnknown` の説明は 8 つの理由を挙げ、merge-order の help、README、`/org` skill は CLI から起きない中断を除く 7 つを挙げる。`lookupFollowupMerges` と `followupMergeState` の unknown の枝と数が合う |
| L7 | 解決 | `list --all` に `DONE_AT`・`DONE_NOTE` の 2 列(`org_followup.go:254` の `printFollowupTable`)。open の件は `-`、値は `inboxField` を通す。help、skill、テスト(制御文字の escape も)を合わせた |
| L8 | 解決 | `lookupFollowupMerges` の説明を折り返し直した |
| L9 | 一部解決。`mergeorder.go` の分は直しにも台帳にもない | `internal/org/followup_test.go` は `followup_recorded_merge_test.go` を分けて 678 行になった。`internal/cli/status.go`(959 行)と出力の関数の引数は c9cb588f の tech-debt の行にした。`internal/org/mergeorder.go` は R2-3 |

### 依頼された点の確認(run 2)

- M1 の直し: 記録より前の同じブランチの `pr_merged` と `head=` が同じ `worktree_cleaned` だけを数えない。別の head は数える。順を入れ替えたテストがある。`recordedMergeHead` は `worktree=` の前の語だけを読むので、引用したパスの中の `head=` に引っかからない(テストの「a worktree_cleaned of another head whose quoted worktree holds the earlier head」)。head の値は `gitObjectPattern` で確かめ、読めない head の `worktree_cleaned` は数える側に倒れるが、書き手は検査済みの SHA しか書かない(`cleanup.go:264`)。
- `ghBranchPRs`: merge、cleanup、merge-order の gh の呼び出しは、引数・`-R`・timeout・エラーとも前と同じ(L1 の行)。
- JSON の `followups`: 外側の 4 つのキーは 2 つのコマンドで同じで、文書(merge-order の help、README の 284 行、`/org` skill の merge-order の行)は「status と同じキーの object」と書く。項目の中身の違いは R2-2。
- 「割れたら merge-order の答えを使う」: status、`followup`、`followup add`、merge-order の help、`/org`・`/director` skill、README、AGENTS.md、仕様の注の 9 か所が同じことを言う。4 つの写しは `.claude/skills/` とバイト単位で同じ。規則が `unknown` を含むかは R2-1。
- 直しが入れたもの: `MergeOrderFollowups` と `mergeOrderFollowups`、`ValidateFollowupNote` と `validateFollowupNote` は大文字小文字だけが違うが、`internal/org` には同じ形の組が 30 ほどあり(`ActiveReservation`/`activeReservation` など)、この package の書き方に合っている。`ValidateFollowupAdd` の末尾 3 行は `ValidateFollowupNote` と同じ処理で、呼べば写しが 1 つ減る(所見にはしない)。古い名前(`FollowupRecord`、`MergeOrderFollowupRecord`、`ghFollowupPR`)はどこにも残っていない。ファイルの大きさは R2-3。

### Findings(run 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | R2-1 「2 つが割れたら merge-order の答えを使う」は、merge-order が `unknown` のときも字のうえでは当てはまる。gh がないとき、merge-order はすべての follow-up を `unknown` にし、DETAIL に「ralph status tells it from the ledger's records of merges, without gh」と書く。status は同じ件を、記録以後の `pr_merged`(gh で merged を確かめたあとの記録、D2)から `ready` にできる。規則に従うと `unknown` を使うことになり、merge-order の DETAIL は status を読めと言うので、2 つの文が逆を指す。gh が 1 本のブランチで失敗したか時間切れになったときも同じ。9 か所の規則は、どれも ready と waiting の割れ方しか例に挙げない。`/director` skill は続けて「`unknown` の件は DETAIL の理由をそのまま人に伝える」と書くので、理由は人に届くが、status の `ready` を着手できるとして伝えてよいかは決まらない。同じ段落の折り返しは `director/SKILL.md:1113` で短い行(「使う。`closed`(PR が merge」)になっている(写し 4 つとも)。 | `internal/org/mergeorder_followup.go:188`、`internal/cli/status.go:142`、`internal/cli/org_followup.go:30`、`:77`、`internal/cli/org_merge_order.go:126`、`.claude/skills/director/SKILL.md:1110-1114`、`.claude/skills/org/SKILL.md:1637`、`README.md:132`、`AGENTS.md:91`、`docs/specs/2026-10-07-org-multi-org-director.md:153` | 規則を「一方が `ready`、もう一方が `waiting` なら merge-order の答えを使う。merge-order が `unknown` なら、その理由を添えて status の答えを伝える」のように書き分ける。文を 9 か所で変えたくなければ、gh がないときの DETAIL から status を指す後半を外す。`director/SKILL.md` の段落は折り返し直す。 |
| LOW | maintainability | R2-2 2 つのコマンドの `followups` は外側のキーが同じになったが、項目の `merged_at` の意味が違い、status 側の説明はそれを書いていない。status の `merged_at` は `merged_event` の event の ts で、`worktree_cleaned` なら片付けの時刻になる。M1 の直しは `FollowupRecordedMerge` の説明と tech-debt の行にそのことを書いたが、`statusFollowupJSON` の説明は今も「the time of the recorded merge that makes it ready」と書き、status の help は `merged_at` の意味を書かない。merge-order の `merged_at` は gh の `mergedAt`(merge の時刻で、waiting の件では記録より前の merge の時刻も入る)。`MergeOrderFollowups` の説明は「the same object as `ralph status --json` gives」と書くが、項目のキーも `pr` と `merged_event` で違う。#271 が 2 つの JSON の `merged_at` を同じものとして比べると、片付けの時刻を merge の時刻と読む。 | `internal/cli/status_followup.go:38-39`、`internal/cli/status.go:146`、`internal/org/mergeorder_followup.go:60-63`、`:86` | `statusFollowupJSON` の説明を「MergedAt is that event's ts: for a worktree_cleaned, when the cleanup ran, not when the PR merged」にし、status の help の `merged_at` に同じことを一言足す。`MergeOrderFollowups` の説明は「the same outer keys as」にして、項目の違い(`pr` と `merged_event`、`merged_at` の意味)を挙げる。 |
| LOW | maintainability | R2-3 run 1 の L9 は `status.go` と `mergeorder.go` の 2 つを挙げたが、c9cb588f が tech-debt に送ったのは `status.go` だけで、計画の Progress も「L9 の残り(`status.go` 959 行…)」と書く。`mergeorder.go` は main で 1291 行、run 1 で 1306 行、今は 1322 行で、F1 の `ghBranchPRs` の分離だけで 16 行増えた。800 行を越えたのはこの PR の前からだが、台帳にこのファイルの大きさの行はない(同じ形の `mergeguarantee.go` には行がある)。計画は archive されるので、ほかに残る場所はない。 | `internal/org/mergeorder.go`(1322 行)、`docs/tech-debt/README.md`(c9cb588f が足した `status.go` の行、`mergeguarantee.go` の行)、`docs/plans/active/2026-10-30-org-followups.md` の Progress checklist | `status.go` の行に `mergeorder.go`(main で 1291 行、この PR で +31)を足すか、別の 1 行にする。引き金は「次に merge-order に gh の読み方を足すとき、gh の lookup(`lookupMergePRs`・`lookupMergePR`・`ghBranchPRs`・`pickMergePR`)を別のファイルに出す」でよい。 |
| LOW | maintainability | R2-4 F1 が `ghPullRequest` に `MergedAt`(`json:"mergedAt,omitempty"`)を足したので、テストの `moFollowupPR` は `ghPullRequest` を埋め込んだうえに同じ JSON 名の `MergedAt` を持つ。encoding/json は浅いほうのフィールドを使うので、埋め込んだ側に入れた値は出力されない。scratchpad の `go run` で確かめた: 埋め込んだ側だけに入れると `{"number":7}`、外側に入れると `{"number":7,"mergedAt":"…"}`。今のテストは外側にしか入れないので落ちないが、`moPR` の値に `MergedAt` を入れて渡すフィクスチャを書くと、merge の時刻のない PR として黙って読まれる。 | `internal/org/mergeorder_followup_test.go:37-40`、`internal/org/mergeorder.go:645` | `moFollowupPR` を消して `ghPullRequest` をそのまま使う。`moMergedPR` は埋め込んでいた `MergedAt` に入れ、`moFollowupPRs` は `moPRs` で足りる。 |

### Positive notes(run 2)

- M1 のテストは、書き手を通した event で、順の入れ替え、別の head(使い回し)、記録以後の別の head の `pr_merged`、引用したパスに `head=` を含む `worktree_cleaned` を見ている。
- `prMergedEvent` は `recordMerge` が書いた行を読み戻すので、`recordMerge` の語の順を変えるとテストが落ちる(run 1 の L4 が求めた守り)。
- `ghBranchPRs` への集約は、gh の呼び出しの形を変えずに写しをなくした。prchecks、merge、cleanup の呼び出しは変わらない。
- done の `--note` を台帳の前に検査することを、古い台帳の repository で、注記が出ないことまで確かめている。
- M1 の残りの形は、9 か所の文書と tech-debt の行が同じ例(GitHub で記録より前に merge、記録のあとに `ralph org cleanup`)を挙げている。

### Tech debt identified(run 2)

この report では `docs/tech-debt/README.md` を変えていない(依頼の範囲が report と insight event だけのため)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| R2-3: `internal/org/mergeorder.go` の大きさ(1322 行)が台帳にない | LOW | run 2 では直さない。直すか台帳に足すかは orchestrator が決める | 上の R2-3 の推奨 | この report の R2-3、run 1 の L9 |

R2-1、R2-2、R2-4 を直さずに PR に進むなら、R2-3 とまとめて 1 行にして台帳に足す。

### Recommendation(run 2)

- Run 2 の判定: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。run 1 の M1 と L1〜L8 は解決し、M1 の残りの形は文書と tech-debt に送った。L9 は `status.go` を tech-debt に送り、`mergeorder.go` の分は R2-3。R2-1〜R2-4 は文、コメント、テストのヘルパー、台帳の行で、挙動は変えない)
- Follow-ups:
  - R2-1 は help 4 つと skill の写し 8 つ、README、AGENTS.md、仕様の注のどれか(または merge-order の DETAIL 1 つ)を変える。R2-2 はコメントと help、R2-4 はテストだけ、R2-3 は tech-debt の行だけ。どれも記録だけの直しの例外には入らない。
  - テストと静的検査はこの report では走らせていない(`/verify` と `/test` の範囲)。

## Run 1(HEAD d58f5652)

### Evidence reviewed(run 1)

- 計画の D1〜D5、AC1〜AC8、Risks、Progress checklist(S1〜S5 の決めたことと計画からの変更)。
- store: `followup.go` の Add・Done・Read・fold を `internal/org/inbox.go` の同じ部分(`update`、`appendLocked`、`ValidateInboxNote`)と突き合わせた。lock の中で読み直してから追記する、ID は壊れた行も含めた最大の次、壊れた最後の行の前に改行を足す、2 度目の done は書かずに `changed=false`、ファイルがなければ lock を作らずに `ErrFollowupUnknownID`、の 5 点は inbox と同じ形になっている。fold は Add が断る `added`(`-` 始まりのブランチなど)を数えて捨てるので、手で書いた行が gh の引数に届かない(`followup.go:386`)。
- `FollowupRecordedMerge`(`followup.go:453-472`)と、`pr_merged`・`worktree_cleaned` を書く側(`merge_run.go:557-566` の `recordMerge`、`cleanup.go:1001-1019` の `record` と `cleanupDetails`)と、2 つの event の説明(`seat.go:119-153`)。どちらの event にも dry-run の書き手はないので、DryRun で絞らないことは問題にならない。
- merge-order: `lookupFollowupMerges`・`lookupFollowupPRs`・`checkFollowupPRs`・`followupMergeState`・`pickFollowupPRs` を、`lookupMergePRs`・`lookupMergePR`・`pickMergePR`(`mergeorder.go:667-723`、`:953-989`)と比べた。`MergeOrderParams.Followups` を立てるのは `org_merge_order.go:179` だけで、`ralph org cleanup`(`org_cleanup.go:167`)は立てない。ランク付きの entry と cleanup の gh の呼び出しは変わらない。merge-order の CLI は signal で ctx を切らないので、「中断は error にしない」枝に CLI から入るのはテストだけになる。
- status: `withStatusReservations` の予約の組み方、`printStatusReservation` が素の値で出すパスは `reservedPathsFromDetails` → `NormalizeReservePaths` を経て、空白・制御文字・`,` を持たない(`reserve.go:209-216`、`:294-308`)。`readStatusFollowups` は gh も exec も呼ばない。JSON の `followups` は空のときも中身のあるときも同じ key の object。
- 出力の escape: merge-order と status の follow-up の表、`followup list` の表は、値をすべて `inboxField` に通す。`followup add` と `done` が素で出す値は、検査を通ったブランチ名と、fold が `f<N>` と認めた ID だけ。
- skill と文書: `/org` の動詞の行と「merge を待つ issue」(`SKILL.md:254-256`、`:1605-1650`)、`/director` の「merge を待つ issue」(`SKILL.md:1090-1116`)、README の status と merge-order の段落、AGENTS.md の地図、仕様 FR-9 の注、`ralph org` と `status` と `merge-order` と `followup` の help。
- ファイルの大きさ: 新しい本体のファイルは 484・363・253・189 行。新しいテストは 814・711・628・570・318・265 行。`status.go` は 901 行から 956 行、`mergeorder.go` は 1291 行から 1306 行。
- 確かめていない: テスト、静的検査、写しの検査(skill の規則どおり走らせていない)。

### Findings(run 1)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1 `ralph status` は、follow-up より前の merge を `ready` にすることがある。`FollowupRecordedMerge` は `worktree_cleaned` の ts を merge の時刻として follow-up の `added_at` と比べるが、その ts は片付けを打った時刻で、merge の時刻ではない。`ralph org merge` が merge して `pr_merged` を書き、片付けが断られ(cleanup は直したあとで打ち直せる)、そのあとで同じ名前のブランチを待つ follow-up が記録され、`ralph org cleanup` が通ると、`worktree_cleaned` が follow-up より後になって status は `ready` と「issue を今聞く」を出す。merge-order は同じ PR を「記録より前の merge」として `waiting` にするので、2 つの道の答えが割れる。テストはこの形を `ready` で固定している(行の名前も「a pr_merged before the add and a worktree_cleaned after it」)。一方で status の help、add の help、merge-order の help(「status decides the same」)、`/org` と `/director` の skill、AGENTS.md は「記録より前の merge は数えない」と書く。`/director` skill は、merge-order が `ready` で status が `waiting` のときの読み方は書くが、逆の割れ方には何も書かない。status の JSON の `merged_at` も、`worktree_cleaned` のときは片付けの時刻になる。GitHub で人が merge した PR を follow-up のあとに片付けたときも同じ割れ方になる。 | `internal/org/followup.go:440-472`(ts の比較)、`internal/org/followup_test.go:736-742`(固定する行)、`internal/org/cleanup.go:1001-1003`(片付けの時刻で書く)、`internal/cli/status.go:130-135`、`internal/cli/org_followup.go:68-73`、`internal/cli/org_merge_order.go:115-116`、`.claude/skills/org/SKILL.md:1625-1627`、`.claude/skills/director/SKILL.md:1103`、`:1104-1110`、`AGENTS.md:91`(「both count only a merge after the follow-up was recorded」) | 同じブランチに `added_at` より前の `pr_merged` があり、その `head=` が `worktree_cleaned` の `head=` と同じなら、その `worktree_cleaned` は前の merge の片付けとして数えない(テストの行の期待も反転する)。あわせて status の help と `FollowupRecordedMerge` の説明に「`worktree_cleaned` の時刻は片付けの時刻で、GitHub で follow-up より前に merge し、あとで片付けた PR は status では `ready`、merge-order では `waiting` になる」と書き、`/director` skill の 2 に「2 つが割れたら merge-order の答えを使う」と足す。コードを変えないなら、help・skill・AGENTS.md の「前の merge は数えない」を、この例外が分かる文に直す。 |
| LOW | maintainability | L1 `lookupFollowupPRs` は `lookupMergePR` の写しになっている。gh の引数(`--head`、`--state all`、`--limit`、`-R`)、`GH_PROMPT_DISABLED=1`、timeout と「timed out after」の包み方、JSON でない出力の文が同じで、違うのは `--json` の項目と返し方だけ。ファイルの頭のコメントは「same shape, timeout, environment and -R pin as lookupMergePR's」と書くが、それを保つのは写しだけなので、片方の引数や環境を変えると、もう片方が黙ってずれる。 | `internal/org/mergeorder_followup.go:21-24`、`:218-245`、`internal/org/mergeorder.go:953-973` | 項目を引数に取って `gh pr list` を打ち、出力の bytes を返す関数を 1 つ作り、2 つの lookup はそれを呼んで JSON を自分の型に読む。 |
| LOW | maintainability | L2 status と merge-order が、follow-up の状態の一部を別々に組み立てている。`added_at` が読めないときの `unknown` の判定と文は、`statusFollowup` と `followupMergeState` に同じ文で 2 つあり、`ready` の文の後半(「ask the human about issue #N now, then run ralph org followup done fN」)も 2 つある。`statusFollowup` は `added_at` を自分で読み、`FollowupRecordedMerge` がもう一度読む。片方の文を直すと、2 つの道の DETAIL が食い違う。 | `internal/cli/status_followup.go:155-163`、`internal/org/mergeorder_followup.go:290-299`、`internal/org/followup.go:454-457` | `internal/org` に、manifest の events から state・event・時刻・detail を返す関数(たとえば `FollowupRecordedState`)を置き、status はそれを JSON と表に写すだけにする。共通の文は定数か小さな関数にする。 |
| LOW | maintainability | L3 2 つのコマンドの JSON で、同じ key `followups` の形が違う。status は `{items, corrupt_lines, ignored_events, error}` の object、merge-order は配列。merge-order は記録の傷の数と読めなかった理由を `json:"-"` にして stderr と `notes` の文にだけ出すので、`--json` を読む側(#271 の director)は「記録が読めない」と「open の件がない」を、`notes` の文を読まないと区別できない。S4b が status でなくそうとした区別が、merge-order には残っている。 | `internal/org/mergeorder.go:206-212`、`internal/org/mergeorder_followup.go:95-105`、`internal/cli/status_followup.go:54-66` | merge-order の JSON に `followup_record`(`corrupt_lines`、`ignored_events`、`error`)を足すか、2 つの形が違うことと、merge-order では `notes` を読むことを help と `/org` skill に書く。 |
| LOW | maintainability | L4 `FollowupRecordedMerge` は「Details の最初の語が `branch=`」に頼るが、2 つの event の説明(`seat.go`)は読み手を列挙していて、この読み手を挙げていない。`EventPRMerged` の説明は今も「ralph org merge-order does not read it」で終わる。`recordMerge` の Details は `fmt.Sprintf` に直に書いてあり、テストの `prMergedEvent` は同じ文を手で写したものなので、`recordMerge` の語の順を変えてもテストは落ちず、status は黙って `ready` を出さなくなる。`worktree_cleaned` のほうは `cleanupDetails` を呼ぶので守られている。status のテストの `worktree_cleaned` は `via=gh` で、書き手が書かない値になっている。 | `internal/org/seat.go:119-153`、`internal/org/merge_run.go:561-563`、`internal/org/followup.go:474-484`、`internal/org/followup_test.go:680-685`、`internal/cli/status_followup_test.go:362` | 2 つの event の説明に「`FollowupRecordedMerge`(`ralph status`)が最初の `branch=` を読む」と足す。`recordMerge` の Details を `cleanupDetails` と同じく関数にし、`prMergedEvent` はそれを呼ぶ。テストの `via=` は `cleanup` か `merge` にする。 |
| LOW | readability | L5 文の読み違えうる所が 3 つある。(a) `closed` の DETAIL は、古い merge の節を差し込んだあとに「: reopen it or open a new PR」が続くので、`it` が直前の merge 済みの PR を指して読める(merge 済みの PR は reopen できない)。テストは部分ごとに見るので気づかない。(b) `warnFollowupDamage` の「an added add would refuse」は、`added` の event と動詞の `add` が並んで読みにくい。(c) `add` は CLI で入力を検査してから台帳を解決し、その理由をコメントに書くが、`done` の `--note` は台帳を解決したあとの `store.Done` で初めて検査されるので、旧台帳の断りや注記のほうが先に出る。 | (a) `internal/org/mergeorder_followup.go:310-321`、`internal/org/mergeorder_followup_test.go:187-188`。(b) `internal/cli/org_followup.go:230-231`。(c) `internal/cli/org_followup.go:83-89`、`:171-176`、`internal/org/followup.go:182-186` | (a) `reopen PR #%d` と番号で書く。(b) 「an added event that add would refuse」にする。(c) `done` の RunE でも、`note` が空でなければ store を作る前に検査する(`org` 側に検査の関数を出す)。 |
| LOW | readability | L6 `unknown` になる理由の列挙が、コードより少ない。`followupMergeState` は gh の上限(1000 本)に届いたとき、merge 時刻が読めないとき、gh の知らない state、`added_at` が読めないときも `unknown` にする。定数 `FollowupMergeUnknown` の説明は上限と知らない state を挙げず、merge-order の help、README、`/org` skill は「gh がない・失敗した・origin と照らせない」の 3 つだけを書く。どれも全部を挙げたように読める。 | `internal/org/mergeorder_followup.go:42-45`、`:290-305`、`:328-329`、`internal/cli/org_merge_order.go:113-115`、`README.md:284`(merge-order の段落の follow-up の文)、`.claude/skills/org/SKILL.md:1639-1641` | 全部を挙げるか、「たとえば」と書き、理由は DETAIL にあると添える。 |
| LOW | readability | L7 `followup list --all` の表は、done の件にも ADDED_AT と add の NOTE しか出さず、いつ done になったかと done の note(人が決めたこと)は `--json` にしか出ない。`/org` skill は「done の件は `followup list --all` にだけ出る」と書き、`/director` skill は新しい director が `followup list` で引き継ぐと書くので、引き継ぐときに読む表で決めたことが見えない。 | `internal/cli/org_followup.go:117-123`、`:238-252`、`.claude/skills/org/SKILL.md:1649-1650` | `--all` のときは DONE_AT と DONE_NOTE の列を足す。足さないなら help に「done の時刻と note は `--json` に出る」と書く。 |
| LOW | readability | L8 `lookupFollowupMerges` の説明のコメントの折り返しが途中で切れている。「then leaves every PR out. ctx」で行が終わり、次の行が「done makes the branches ...」で始まる。 | `internal/org/mergeorder_followup.go:150-153` | 段落を折り返し直す。 |
| LOW | maintainability | L9 800 行の目安を越えるファイルがある。新しいテストの `followup_test.go` は 814 行。計画は `status.go`(901 行)と `mergeorder.go`(1291 行)を太らせないために新しいファイルを分けたが、help と引数の受け渡しで `status.go` は 956 行、`mergeorder.go` は 1306 行になった。`printStatusEmpty`・`printStatusJSONAllOrgs`・`printStatusTableAllOrgs` の引数は 11〜12 個に増えた。 | `internal/org/followup_test.go`、`internal/cli/status.go:499`、`:695`、`:731` | 今回は直さなくてよい。status の出力の引数は、次に項目を足すときに 1 つの struct にまとめる。テストは `FollowupRecordedMerge` の表のテスト(`:680-814`)を別のファイルに移せば 800 行に収まる。 |

### Positive notes(run 1)

- store は inbox と同じ作りで、ID の採番(壊れた行と JSON の escape で書いた ID を数える)、壊れた最後の行、2 度目の done、lock の取り合い(Add と Done の並行)、lock を作れないときに何も書かないこと、をそれぞれテストで見ている(`followup_test.go:290-376`、`:528-677`)。
- fold が Add の断る `added` を捨てるので、`followups.jsonl` を手で直しても、`-` で始まるブランチが gh の `--head` に渡らない。
- merge-order の節は `MergeOrderParams.Followups` の下にだけあり、`ralph org cleanup` の gh の呼び出しと entry の分類を変えない。そのことと、Followups なしでは何も読まないことをテストが見ている(`mergeorder_followup_test.go:263`、`:500`)。follow-up がなければ gh を 1 回も足さない。
- gh がないとき、別の repository の PR を読んだとき、1 本のブランチで失敗したとき、の 3 つで、どの follow-up が `unknown` になるかの範囲が `lookupMergePRs` の扱いと揃っている。
- status は gh を打たないことを、PATH に gh の stub を置いて呼ばれないことで確かめている(`status_followup_test.go:636`)。JSON の `followups` を object にして、読めない記録と件のない記録を `error` で分けた(S4b)。
- 表の値はすべて `inboxField` を通し、`add` と `done` が素で出す値は検査済みのものに限られている。
- 計画の Risks で決めた「既存の検索結果を使い回す」を採らなかった理由(cleanup の gh の呼び出しを変えないため)を Progress に書いている。

### Tech debt identified(run 1)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(この cycle では台帳に足していない。所見を直さずに残すなら、M1 は 1 行、LOW はまとめて 1 行にして `docs/tech-debt/README.md` に足す。)_

### Recommendation(run 1)

- Run 1 の判定: yes(CRITICAL・HIGH は 0 件。M1 は計画の D5 と AC8 が守ろうとする「前の merge は数えない」が status の一方の道で崩れ、2 つのコマンドの答えが割れる形なので、PR の前に直すことを勧める)
- Follow-ups:
  - M1 はコードとテストの期待(`followup_test.go:736-742`)を変えるか、help・skill の写し 4 つ・AGENTS.md・README の文を変える。どちらも記録だけの直しの例外には入らないので、直したら `/self-review` から通し直す。
  - L1〜L9 は直さなくても挙動は変わらない。直すなら M1 と同じ cycle にまとめる。
  - テストと静的検査はこの report では走らせていない(`/verify` と `/test` の範囲)。
