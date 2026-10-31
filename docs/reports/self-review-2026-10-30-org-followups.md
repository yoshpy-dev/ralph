# Self-review report: org-followups

- Date: 2026-10-31
- Plan: `docs/plans/active/2026-10-30-org-followups.md`(issue #270、D1〜D5、AC1〜AC8、スライス S1 de7dd522・S2 dc589374・S3 4d78c28e・S4 4a44ef30・S4b ef1663e9・S5 2d86a71e)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 1cc84ffa、HEAD d58f5652、32 ファイル、+5263/-79)。台帳の follow-up の store(`internal/org/followup.go`)、merge-order の follow-up の節(`internal/org/mergeorder_followup.go` と `mergeorder.go` の差し込み)、動詞(`internal/cli/org_followup.go`)、`ralph status` の予約と follow-up(`internal/cli/status_followup.go` と `status.go` の差し込み)、`lockfile.go` のコメント、テスト、`/org`・`/director` の skill と写し、README、AGENTS.md、仕様の注。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L9(LOW)。

## Evidence reviewed

- 計画の D1〜D5、AC1〜AC8、Risks、Progress checklist(S1〜S5 の決めたことと計画からの変更)。
- store: `followup.go` の Add・Done・Read・fold を `internal/org/inbox.go` の同じ部分(`update`、`appendLocked`、`ValidateInboxNote`)と突き合わせた。lock の中で読み直してから追記する、ID は壊れた行も含めた最大の次、壊れた最後の行の前に改行を足す、2 度目の done は書かずに `changed=false`、ファイルがなければ lock を作らずに `ErrFollowupUnknownID`、の 5 点は inbox と同じ形になっている。fold は Add が断る `added`(`-` 始まりのブランチなど)を数えて捨てるので、手で書いた行が gh の引数に届かない(`followup.go:386`)。
- `FollowupRecordedMerge`(`followup.go:453-472`)と、`pr_merged`・`worktree_cleaned` を書く側(`merge_run.go:557-566` の `recordMerge`、`cleanup.go:1001-1019` の `record` と `cleanupDetails`)と、2 つの event の説明(`seat.go:119-153`)。どちらの event にも dry-run の書き手はないので、DryRun で絞らないことは問題にならない。
- merge-order: `lookupFollowupMerges`・`lookupFollowupPRs`・`checkFollowupPRs`・`followupMergeState`・`pickFollowupPRs` を、`lookupMergePRs`・`lookupMergePR`・`pickMergePR`(`mergeorder.go:667-723`、`:953-989`)と比べた。`MergeOrderParams.Followups` を立てるのは `org_merge_order.go:179` だけで、`ralph org cleanup`(`org_cleanup.go:167`)は立てない。ランク付きの entry と cleanup の gh の呼び出しは変わらない。merge-order の CLI は signal で ctx を切らないので、「中断は error にしない」枝に CLI から入るのはテストだけになる。
- status: `withStatusReservations` の予約の組み方、`printStatusReservation` が素の値で出すパスは `reservedPathsFromDetails` → `NormalizeReservePaths` を経て、空白・制御文字・`,` を持たない(`reserve.go:209-216`、`:294-308`)。`readStatusFollowups` は gh も exec も呼ばない。JSON の `followups` は空のときも中身のあるときも同じ key の object。
- 出力の escape: merge-order と status の follow-up の表、`followup list` の表は、値をすべて `inboxField` に通す。`followup add` と `done` が素で出す値は、検査を通ったブランチ名と、fold が `f<N>` と認めた ID だけ。
- skill と文書: `/org` の動詞の行と「merge を待つ issue」(`SKILL.md:254-256`、`:1605-1650`)、`/director` の「merge を待つ issue」(`SKILL.md:1090-1116`)、README の status と merge-order の段落、AGENTS.md の地図、仕様 FR-9 の注、`ralph org` と `status` と `merge-order` と `followup` の help。
- ファイルの大きさ: 新しい本体のファイルは 484・363・253・189 行。新しいテストは 814・711・628・570・318・265 行。`status.go` は 901 行から 956 行、`mergeorder.go` は 1291 行から 1306 行。
- 確かめていない: テスト、静的検査、写しの検査(skill の規則どおり走らせていない)。

## Findings

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

## Positive notes

- store は inbox と同じ作りで、ID の採番(壊れた行と JSON の escape で書いた ID を数える)、壊れた最後の行、2 度目の done、lock の取り合い(Add と Done の並行)、lock を作れないときに何も書かないこと、をそれぞれテストで見ている(`followup_test.go:290-376`、`:528-677`)。
- fold が Add の断る `added` を捨てるので、`followups.jsonl` を手で直しても、`-` で始まるブランチが gh の `--head` に渡らない。
- merge-order の節は `MergeOrderParams.Followups` の下にだけあり、`ralph org cleanup` の gh の呼び出しと entry の分類を変えない。そのことと、Followups なしでは何も読まないことをテストが見ている(`mergeorder_followup_test.go:263`、`:500`)。follow-up がなければ gh を 1 回も足さない。
- gh がないとき、別の repository の PR を読んだとき、1 本のブランチで失敗したとき、の 3 つで、どの follow-up が `unknown` になるかの範囲が `lookupMergePRs` の扱いと揃っている。
- status は gh を打たないことを、PATH に gh の stub を置いて呼ばれないことで確かめている(`status_followup_test.go:636`)。JSON の `followups` を object にして、読めない記録と件のない記録を `error` で分けた(S4b)。
- 表の値はすべて `inboxField` を通し、`add` と `done` が素で出す値は検査済みのものに限られている。
- 計画の Risks で決めた「既存の検索結果を使い回す」を採らなかった理由(cleanup の gh の呼び出しを変えないため)を Progress に書いている。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(この cycle では台帳に足していない。所見を直さずに残すなら、M1 は 1 行、LOW はまとめて 1 行にして `docs/tech-debt/README.md` に足す。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH は 0 件。M1 は計画の D5 と AC8 が守ろうとする「前の merge は数えない」が status の一方の道で崩れ、2 つのコマンドの答えが割れる形なので、PR の前に直すことを勧める)
- Follow-ups:
  - M1 はコードとテストの期待(`followup_test.go:736-742`)を変えるか、help・skill の写し 4 つ・AGENTS.md・README の文を変える。どちらも記録だけの直しの例外には入らないので、直したら `/self-review` から通し直す。
  - L1〜L9 は直さなくても挙動は変わらない。直すなら M1 と同じ cycle にまとめる。
  - テストと静的検査はこの report では走らせていない(`/verify` と `/test` の範囲)。
