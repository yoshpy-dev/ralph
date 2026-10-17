# Self-review report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認済み。8 段目の 8a-3b)
- Reviewer: reviewer subagent (Claude)。パイプライン 3 回目(cycle 3)。cross-review の cycle 2 の ACTION_REQUIRED 2 件(556f73fb と 4129ae22 で直した)のあとの回し直し。`.harness/state/standard-pipeline/cycle-count.json` は 2 のままで、cap を 2 から 3 に上げた(cross-review の cap 到達の選択肢 1)。この回が cap 3 の最後
- Scope: diff の品質だけ。`git diff 670ec219...HEAD`(HEAD 927d6c26)全体を見たうえで、cycle 2 の self-review(a59220f5 の時点)のあとのコード 5 コミット(ca840a45、5e12dfe0、c621ccc9、556f73fb、4129ae22)、テストだけの f630e216、文書の c60b2651 を重点に読んだ。c60b2651 と plan は文面の品質だけを見た。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)
- ID: 今回の指摘は `C3-n`。cycle 2 のレポート(C2-1〜C2-5)は全文を「付録 B」に、cycle 1 の 2 回目のレポート(M1、L1〜L9、N1〜N4)は全文を「付録 A」に残した。登録簿が「self-review fix L9」で付録 A を指す。付録の `Merge:` の行は `cycle N の判定:` に変えてあり、`Merge:` の行はこのレポートの「Recommendation」の 1 つだけ

## Evidence reviewed

- 読んだコード: `internal/org/regate_merge.go` の全体(825 行)、`regate.go` の全体(738 行)、`regate_git.go` の全体(210 行)。a59220f5 から HEAD までのコードの差分(`regate_merge.go`、`regate.go`、`mergeorder.go`、`internal/cli/org.go`)をコミットごとに。ca840a45 の `regateMergeWrites` と `regateCheckInTheWay`、c621ccc9 の `regatePathInTheWay` と `regateSpellingOnDisk`、556f73fb の `regateIsOwnMerge` と、それを使う `regateHeadNotOnOrigin`・`regateResetHint`、4129ae22 の `regatePush`・`regateSetTrackingRef`・`regateTrackingRef` と test hook 2 つ。`regateSpawnErr` の呼び出し元(`spawn.go:1284`、`feature.go:216`)と dry-run の分岐(`spawn.go:600`)
- 読んだテスト(diff。実行はしていない): 556f73fb の `TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`(3 例)と `TestRegate_AFailedPushIsNotRetriedPastWhatChanged` の新しい subtest、4129ae22 の `TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush`(3 例)、c621ccc9 の 6 例と `regateSkipOnACaseSensitiveFS`、f630e216 の 10 件と 11 件の境目、`regateWrapGit` で fallback を動かす test
- 読んだ文書: c60b2651 の `/org` skill の語単位の差分、README、spec、登録簿の新しい 7 行(`docs/tech-debt/README.md:281-287`)、plan の D4・AC7・Progress、cross-review の triage。skill の 4 面は `cmp` で byte 一致
- 手で確かめたこと(repo の tree には何も書いていない): (1) `regate_merge.go` の行数をコミットごとに数えた(a59220f5 で 644、ca840a45 で 720、c621ccc9 で 760、556f73fb で 790、4129ae22 で 825)。`gate.go` も(base 763、最初の regate のコミット 7b6894ea で 809、HEAD で 833)。(2) regate の help、start の help、spawn の help、README、skill、spec の「regate が走る間に断られるもの」の文面の突き合わせ(C3-1)。(3) `scripts/secret-scan-branch.sh` が `git merge-tree -h` の出力に `--write-tree` があるかで古い git を判じること(`regateMergeWrites` の doc の参照と合う)。(4) 追加行の debug 出力・TODO・`/Users/`・U+FFFD(なし)。`git status --short` は空。cycle 2 の probe(git 2.49.0)で足りるので、git の再現は今回していない

## cycle 2 の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| C2-1 | 直った | merge が書く path は、`git merge-tree --write-tree --no-messages HEAD <tip>` の木と HEAD の `git diff --name-only -z --no-renames --diff-filter=A` の差から取る(`internal/org/regate_merge.go:345-363`)。merge-tree は merge と同じ env(`regateMergeEnv`)で、別の process group で動き、ctx で止まる。`--write-tree` を知らない git(`merge-tree -h` に名がない。`scripts/secret-scan-branch.sh` の判じ方と同じ)は、旧来の 2 つの diff の積に戻る(`:365-391`)。test は directory rename の両向き(`regate_merge_test.go:1496`、`:1508`)、file/directory conflict が HEAD のファイルに付ける名前(`:1520`)、fallback を PATH の wrapper で動かすもの(`:1777`)。`regateCheckInTheWay` の doc の「are those」は消え、`regateMergeWrites` の doc が fallback の限界を書く |
| C2-2 | 直った | `regatePathInTheWay`(`:415`)は最後の component が directory のとき、`git ls-files --others --directory --no-empty-directory` が何も出さなければ断らない。空のディレクトリ(`regate_merge_test.go:1665`)、HEAD が追跡するディレクトリをファイルに置き換える tip(`:1671`)、逆向き(`:1678`)が、断られずに merge まで通る test になった |
| C2-3 | 直った | `mergeorder_test.go:932-942` の doc が「every branch」の規則で書かれ、この test の org は branch が 1 つであること、2 つ以上は `...EveryBranchOfItsOrg` が見ることを言う |
| C2-4 | 一部 | spawn の help は直った(`internal/cli/org.go:677-679`)。regate 自身の help の同じ文(`--dry-run` と start の形)が残っている(C3-1) |
| C2-5 | 直った | (a) `regateMerged` の doc は折り直され、短い行は残っていない(`regate_merge.go:156-170`)。(b) ファイル冒頭の comment は「the merge commit's undo (regateUndoMerge, D5)」になった(`:15-20`)。(c) field は `orgBranchCount`(`mergeorder.go:359-361`)になり、局所変数 `orgBranches` との同名は消え、「in the order they first came」は「each once」になった(`:401-403`) |

cycle 2 の Coverage gaps のうち、関係のない履歴で `git merge-base` が終了コード 1 になる道は `TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse`(`regate_merge_test.go:1738`)が固定した。`/org` skill の ignored のファイルの断りと `start` の断りの範囲は c60b2651 で直った。plan の Progress の行(Review artifact)は直っていない(下の Recommendation の 2)

## 引き継ぎ(a)〜(c)の判断

- (a) 指摘にしない。gate だけの regate が通した head X が、origin を戻されたあとに origin より進んでいる状態は、push していない commit と区別できない(記録に push の結果がなく、HEAD は gate した head のまま)。`regateCheckOriginHead` の文(`regate_git.go:97-99`)は「HEAD が持ち、origin が持たない commit がある」という事実を言い、push するのは人で、push すれば次の regate は通常の道に戻る。plan の AC7 も、この入力では「push し直さず、`reset --keep` の手も出さずに断る」とだけ決めている。文が「origin が戻された」可能性を言わないことは、C3-5 の登録簿の行に含める
- (b) `Regate` の doc と `passedOn` の doc は偽ではない。`Regate` の doc(`regate.go:241-245`)は条件を書かず `regateHeadNotOnOrigin` を指し、`passedOn` の doc(`:425-431`)は述語が確かめる 3 つを並べるだけで、`via=regate` の形の確かめは呼び出し側(`regateHeadNotOnOrigin`)にある。偽になったのは、`regate.go:34-38` の冒頭 comment だけで、C3-2 にした
- (c) 指摘にしない。`regate_merge.go:15-20` の冒頭 comment は主題の一覧で、`regateResetHint`、`regateParents`、`regateAbortLeftMerge`、`regateSetTrackingRef` など、ほかの関数も名指さない。`regateIsOwnMerge` は「the push and its retry (regatePush, regateHeadNotOnOrigin, D4)」の項に入る

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C3-1 | LOW | doc-accuracy | `ralph org regate --help` の最後の段落は「While it runs, ralph org spawn and ralph org start --plan of the org it works on are refused」と言う。C2-4 が spawn の help で直した 2 点が、regate 自身の help に残っている。(1) `--dry-run` の spawn は断られない。`Spawn` は dry-run を、regate の確かめ(`spawn.go:1284`)より前の `spawn.go:600` で返し、`Spawn` の doc も dry-run は regate の確かめを省くと書く(`:503-505`)。(2) start は 2 つの形とも断られる。start の help は「Both forms are refused」(`internal/cli/org.go:867`)で、README・skill・spec は c60b2651 で「start (either form)」に直った。`internal/cli/org_regate_test.go:738` は古い文をそのまま固定している。登録簿の新しい行の「so the org's `spawn` and `start --plan` are refused (D9)」(`docs/tech-debt/README.md:286`)も同じ古い形 | `internal/cli/org_regate.go:115-116`、`internal/cli/org_regate_test.go:738`、`internal/org/spawn.go:503-505`、`:600`、`:1284`、`internal/cli/org.go:867-868`、`docs/tech-debt/README.md:286` | help を「ralph org spawn (not a --dry-run) and ralph org start (either form) of the org it works on are refused」にし、`:738` の固定を合わせる。登録簿の行 286 も同じ言い方にする |
| C3-2 | LOW | doc-accuracy | `regate.go` の冒頭 comment は push だけの道を十分条件の形で書く(「a HEAD with origin's tip of the branch under it whose gate_passed on origin's default branch tip is the branch's latest record is pushed, and nothing else is done」)。556f73fb のあと、`via=regate` の記録では HEAD が regate 自身の merge commit(親がちょうど 2 つで、第 1 親が origin の先頭、第 2 親が記録の base。`regateIsOwnMerge`)でなければ push しない。規則の全文は `regateHeadNotOnOrigin` の doc(`regate_merge.go:61-74`)にあるが、冒頭 comment は直っていない(556f73fb が触ったのは `regate_merge.go` と test だけ) | `internal/org/regate.go:34-38`、`internal/org/regate_merge.go:61-74` | 冒頭の文に「(a record Regate wrote only when HEAD is Regate's own merge commit, regateIsOwnMerge)」を足す |
| C3-3 | LOW | maintainability | `internal/org/regate_merge.go` は 825 行で、code review checklist の 800 行を超える。a59220f5 は 644 行で、cycle 2 の直しで 720(ca840a45)、760(c621ccc9)、790(556f73fb)、825(4129ae22)と増えた。`internal/org/gate.go` は base の 763 行から、最初の regate のコミット(7b6894ea)で 809 行になり、HEAD で 833 行。この PR は cycle 1 の L9 で regate.go の 851 行を 800 行未満にし、登録簿の行 278・279 も 800 行を基準に書くが、この 2 ファイルは追っていない。gate.go は前の 2 回の self-review が数えていなかった | `internal/org/regate_merge.go`(825 行)、`internal/org/gate.go`(833 行)、`docs/tech-debt/README.md:277-279` | `regate_merge.go` は merge が書く path の確かめ(`regateInTheWayListed` から `regateSpellingOnDisk` まで、`:267-474` の約 210 行)が切れ目なので、`regate_inway.go` に移す(中身は変えない)。gate.go は regate と共有する git の helper(`gateGit`、`gateGitOutput`、`gateHead`、`gateDirtyLines`、`gateGitPathExists`)が移す候補。直さないなら、下の登録簿の行に入れる |
| C3-4 | LOW | readability | `regatePathInTheWay` の doc は「It is the first directory above p that lstat finds as a non-directory, unless HEAD tracks it」と書く(`regate_merge.go:408-410`)。directory で non-directory を指す文で、読み返さないと、ファイルや symlink(merge が directory にしたい場所にあるもの)のことだと分からない。同じ doc の前の 2 つの「It is ...」は、directory と non-directory を言い分けている | `internal/org/regate_merge.go:408-410` | 「It is the first path above p that lstat finds as a non-directory (a file or a symlink where the merge needs a directory), unless HEAD tracks it」にする |
| C3-5 | LOW | debt-record | 4129ae22 のあとも、push だけの道が許す入力に、直せない残りが 2 つあり、どちらも登録簿にない(plan の Progress `:164` は leader の記録の分を「tech-debt に書く」と言うが、行はまだない)。(1) leader の記録(`via` なし)は `regateIsAncestor(originHead, HEAD)` だけで push を許すので、origin が戻されたあとも push し直す(既知)。(2) regate 自身の merge commit が push で届いたあと、origin がその第 1 親(merge の前の head)に戻されると、HEAD の親は `[origin の先頭, 記録の base]` の形で、記録は `via=regate` の gate_passed なので、push だけの道が merge commit を push し直す。記録に push の結果がないので、「push が落ちた」と「届いたあと戻された」を区別できない。`regateHeadNotOnOrigin` の doc(`regate_merge.go:68-71`)は、この理由を gate だけの記録についてだけ述べる。(2) は読んで導いた(実行していない)。test の `...put back under the merge commit's first parent`(`regate_merge_test.go:1111`)は、第 1 親のさらに 1 つ下へ戻す入力で、第 1 親そのものへの巻き戻しは通る側に残る | `internal/org/regate_merge.go:61-84`、`internal/org/regate_merge_test.go:1111`、`docs/plans/active/2026-10-16-org-regate.md:164` | コードはこの PR で直さない(記録に push を試みた印を足す設計が要り、記録の形が変わる)。下の登録簿の行の(e)に、(1)と(2)を書く |

## Positive notes

- C2-1 の直しは、候補を「merge が書く path」そのものから取るので、directory rename の 2 つの向きと、file/directory conflict の `~HEAD` の名前が、場合分けなしで入る。merge-tree は merge と同じ env と同じ設定で動き、doc がその理由(merge の config と attributes を読む)と、動かない git の fallback の限界を書く。fallback は PATH の wrapper で実際に動かし、呼び出しを記録する test があり、merge-tree が失敗し `-h` に `--write-tree` がある場合は error になる test もある
- `regatePathInTheWay` は、lstat が見つけたものを、ディスク上の綴りで判じ、返す path も綴りのままにした。最後の component が directory のときと、上の component がファイルのときと、綴りが違うときだけ git を呼ぶので、ふつうの merge(何も見つからない)は lstat だけで終わる。error(lstat が読めない、git が失敗する)と断り(邪魔なものがある)の分け方が cycle 2 と同じ
- `regateIsOwnMerge` は push だけの道と reset の手が同じ関数を使うので、2 か所の判定がずれない。556f73fb の 3 つの入力は、どれも直す前のコードで赤になる形で(読んだだけ。変異は走らせていない)、「another branch の merge」の例は第 2 親(base)の比較だけで落ちるので、条件の片方を消した変異に効く
- 4129ae22 の `regateSetTrackingRef` は、push の前に読んだ値と `git update-ref <ref> <new> <old>` の比較で動かすので、読んでから更新するまでの間に動いた ref は更新が落ちる(compare-and-swap)。test は fetch が push の直後に動かした場合(update-ref を呼ばないことを hook の `t.Errorf` で見る)、fetch が update-ref の直前に動かした場合、ref が動かないまま update-ref が落ちる場合(error の前後の文を固定)の 3 つで、doc の 3 つの文に 1 つずつ当たる。push の前の読み取りが失敗すると、push する前に断る(以前は push のあとに失敗した)
- c60b2651 の文面は、skill・README・spec で同じ言い方(spawn は `--dry-run` を除く、start は 2 つの形)に揃っている。登録簿の新しい 7 行は、どれも Trigger と pointer を持ち、pointer の関数名と test 名は実在する
- 追加行に debug 出力、TODO、秘密、home の path、U+FFFD はない。skill の 4 面は byte 一致

## Coverage gaps

- テストは実行していない。新しい test が直す前のコードで赤になるか(556f73fb の 3 例、4129ae22 の 3 例、c621ccc9 の 6 例)は、コードと fixture を読んで判断した
- 4129ae22 が足した、push の前の `regateTrackingRef` の失敗の分岐(`regate_merge.go:737-744`)に届く test がない(`was not pushed to origin` を `internal/` の test で grep して 0 件)。git を落とす PATH の wrapper(`regateWrapGit`)の形で足せるが、/test の範囲なので指摘にしていない。lstat が読めない path の error と、`merge-base` が 128 で落ちる error の test がないことも cycle 2 から変わらない
- 大文字小文字を区別しない file system の 6 例は、CI(ubuntu-latest)で skip される(登録簿の行 284)。この環境では走る(macOS)が、実行していない
- `git merge-tree` は local の `merge.<name>.driver` を動かす(`scripts/secret-scan-branch.sh` の注と同じ)ので、driver を設定した repo では merge-tree と `git merge` の 2 回動くはず。確かめていない。登録簿の `branch.<name>.mergeOptions` の行(`:282`)は driver に触れていない
- c60b2651 は 556f73fb と 4129ae22 の前なので、`/org` skill の regate の行(`.claude/skills/org/SKILL.md:188`、4 面)は reset の手の条件を「HEAD の第 1 親が origin の先頭」とだけ書き、push だけの打ち直しにも形の確かめを書かない。文書のずれは /sync-docs の範囲なので、指摘にせず引き継ぐ

## Tech debt identified

この回は cap 3 の最後で、C3-1 から C3-4 は Go の comment・help・ファイル配置なので、直すと全体の再実行(cap を 4 に上げる)になる。直さないなら、登録簿に次の 1 行を足す(C3-5 の(e)を含む)。reviewer は登録簿に書かない。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-regate PR, found by its cycle 3 self-review (all LOW; (a) to (d) change no behavior). (a) The last paragraph of `ralph org regate --help` (`internal/cli/org_regate.go`) says `ralph org spawn` and `ralph org start --plan` are refused while a regate runs, and `internal/cli/org_regate_test.go` pins it; the row on the per-file `git ls-files` of D3's check says the same. A `--dry-run` spawn is not refused (`Spawn` returns before `regateSpawnErr`), and `start` is refused in both forms, as the `start` help, the README, and the `/org` skill say. (b) The file header of `internal/org/regate.go` says a HEAD with origin's tip under it, whose gate_passed is the branch's latest record, is pushed; for a record Regate wrote (via=regate) it is pushed only when HEAD is Regate's own merge commit (`regateIsOwnMerge` in `internal/org/regate_merge.go`). (c) `internal/org/regate_merge.go` is 825 lines and `internal/org/gate.go` 833, over the 800-line file size of the code review checklist (regate_merge.go crossed it in the cycle 2 fixes, gate.go in the first regate commit); the in-the-way check (`regateInTheWayListed` to `regateSpellingOnDisk`, about 210 lines) is a seam. (d) The doc of `regatePathInTheWay` calls a non-directory "the first directory above p". (e) The push-only retry cannot tell a push that failed from one that landed and was then undone by putting origin's branch back. For a leader's record (no via), any HEAD with origin's tip under it is pushed again. For a record Regate wrote, the merge commit is pushed again when origin's branch was put back to the merge commit's first parent (HEAD's parents are then origin's tip and the record's base). The tests put origin back one commit below the first parent (`TestRegate_AFailedPushIsNotRetriedPastWhatChanged`) or after a gate-only record (`TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`). The refusal for a gate-only head after such a put-back (`regateCheckOriginHead`) says to push the commits, as it does for any HEAD ahead of origin | (a) A help line that says both more and less than is true. (b) A reader of the header expects a push that regate refuses. (c) Long files to read and review. (d) Misleading reading only. (e) After someone puts origin's branch back on purpose (a force push), running regate again pushes the commit that was put back, so the put-back is undone without a word | The pipeline cap of 3 ends with this cycle; (a) to (d) are Go comments, help text, and file layout, so fixing them starts another full pipeline run. (e) needs a mark in the gate record that the push was tried, which changes the record's shape (D2, D4) | The next change to `internal/cli/org_regate.go`, the header of `internal/org/regate.go`, `internal/org/regate_merge.go`, or `internal/org/gate.go`: fix (a) to (d) as in the report's C3-1 to C3-4. For (e), a report of a commit pushed again after a force push: record that the push was tried (an event, or a field of the gate record) and push only again after a recorded failure | `docs/reports/self-review-2026-10-16-org-regate.md` (cycle 3, C3-1 to C3-5); `docs/plans/active/2026-10-16-org-regate.md` (D4, AC7); `internal/org/regate_merge.go`; `internal/org/regate.go`; `internal/cli/org_regate.go`; `internal/org/gate.go` |

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。LOW 5 件。cycle 2 の指摘は C2-4 が一部のほか直り、cross-review の cycle 2 の 2 件の直し(556f73fb、4129ae22)は、規則も test も合っている)
- Follow-ups:
  1. 直すなら、C3-1 が一番価値が高い(利用者が読む help の文で、C2-4 と同じ種類の取りこぼし)。C3-1 から C3-4 は Go の comment・help・配置で、直すと cap 3 を超える再実行になる。直さずに `/pr` へ進むなら、上の登録簿の 1 行を `/pr` の前に足す
  2. `/sync-docs` への引き継ぎ: (i) `/org` skill の regate の行(`.claude/skills/org/SKILL.md:188`、4 面)に、reset の手と push だけの打ち直しの条件として「regate 自身の merge commit の形(親がちょうど 2 つ、第 1 親が origin の先頭、第 2 親が記録の base)」を足す。(ii) plan の Progress「Review artifact created(…1 周目の 2 回目)」(`docs/plans/active/2026-10-16-org-regate.md:155`)は、このファイルの先頭が cycle 3 になったので、「cycle 3。付録 B に cycle 2、付録 A に cycle 1 の 2 回目」に直す
  3. insight event は `--cycle 3` を明示して足した(`cycle-count.json` が 2 のままなので、`auto` に頼らない)。`docs/insights/events/2026-10-16-org-regate.jsonl` には `cross_review` の行が 1 つもない(cycle 1 と 2 の triage は commit 済み)。`/cross-review` の側で足すか決めてほしい

## 付録 B: cycle 2 のレポート(a59220f5 の時点、8e2e8372 で書いた)

以下は、cycle 3 の前にこのファイルの先頭にあった cycle 2 のレポートの全文。見出しを 1 段下げ、`Merge:` の行を `cycle 2 の判定:` に変え、ID の行の「今回」を「この回」に変えただけで、ほかは変えていない。その「Tech debt identified」の行は登録簿に入っていない(C2-1 から C2-3 と C2-5 は直り、C2-4 は C3-1 に続く。c60b2651 が足した 7 行は別のもの)。

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認済み。8 段目の 8a-3b)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の cap 2 の最後の回
- Scope: diff の品質だけ。`git diff 670ec219...HEAD`(HEAD a59220f5)全体を見たうえで、cycle 1 の cross-review のあとの 3 コミット(85558f2c、f793924d、a59220f5)と、前回の self-review(07b9a465 の時点)が見ていない 0b6ba232、d00da42c、5e4ce6cd、413ab307 を重点に読んだ。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)
- ID: この回の指摘は `C2-n`。cycle 1 の 2 回目のレポート(M1、L1〜L9、N1〜N4)は、登録簿が「self-review fix L9」で指しているので、全文を末尾の「付録 A」に残した。その `Merge:` の行は `cycle 1 の 2 回目の判定:` に変えてある

### Evidence reviewed

- 読んだコード: 85558f2c の `regateCheckInTheWay`・`regatePathInTheWay`・`regateMerged` の呼び出し(`internal/org/regate_merge.go:235-327`、`:139-155`)。f793924d の `collectMergeTargets`(`internal/org/mergeorder.go:373-461`)、`mergeGateRecord.orgBranches`、`classifyMergeTarget` の Detail(`:935-950`)。`collectMergeTargets` を読む `readRegateBranch`・`passedOn`・`recordRegate`(`internal/org/regate.go`)。0b6ba232 の `regateMergeUnchecked`・`regateAbortLeftMerge`・`regateMergeTip`(`regate_merge.go:349-477`)。413ab307 の help と `regateSpawnErr` の呼び出し元(`spawn.go:1284`、`feature.go:216`)、dry-run の分岐(`spawn.go:600`)。a59220f5 の help と doc
- 読んだテスト(diff。実行はしていない): 85558f2c の 186 行(`TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay` の 6 例と `TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch`)、f793924d の `mergeorder_test.go` と `regate_test.go`(期待を変えた 3 つの test と、足した 1 つ)、0b6ba232・d00da42c・5e4ce6cd の test
- 手で確かめたこと(scratchpad の使い捨ての repo、git 2.49.0。repo の tree には何も書いていない): (1) directory rename を挟む merge が ignored のファイルを上書きし、`merge --abort` のあとも戻らないこと(C2-1 の(a)(b))。(2) その 2 例を `git merge-tree --write-tree HEAD <tip>` の結果の tree と HEAD の `--diff-filter=A` の差で見ると、上書きされる path が出ること。(3) 空のディレクトリ、HEAD が追跡するディレクトリをファイルに置き換える tip は、clean に merge できること(C2-2)。(4) 関係のない履歴では `git merge-base` が終了コード 1・stdout 空、`git merge --no-ff` が終了コード 128・`MERGE_HEAD` なしで終わること(`regateCheckInTheWay` の doc のとおり)
- 追加で grep: 追加行の debug 出力・TODO・`/Users/`・秘密の語(なし)、cross-review の ID を指す comment(新しい行にはなし。`recordGate` の「cross-review cycle 1 #1」は 670ec219 からある)、`gatedBranch` の残り(なし)、`mergeorder.go` の `cmp`・`slices` の import が今も使われていること(`:501`、`:894`、`:406`)、`SortStableFunc` で `pos` が同じ target が並ぶ順(first appearance、決まった順)

### cycle 1 の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| N1 | 直った | `regateMergeTip` の HEAD と親の読み取り 2 か所と、親が違う場合が `regateMergeUnchecked`(`regate_merge.go:418`)を通る。error は merge commit ができたこと、読めないものとその理由、script も記録もしなかったこと、`git reset --keep <merge 前の HEAD>` を言う。`TestRegate_AMergeCommitItCannotReadNamesResetKeep`(`regate_merge_test.go:411`)が HEAD と親の 2 通りを、error の全文と、merge commit が残ること、manifest・origin・script が動かないことまで固定している。推奨は `regateMergeNotBack` に通すことだったが、専用の constructor になった。`regateMergeNotBack` にある「run ralph org regate --branch X again」は `regateMergeUnchecked` にないが、元の親違いの文にもなかった |
| N2 | 直った | `regateAbortLeftMerge`(`:434`、44 行)を切り出し、`regateMergeTip` は 99 行から 62 行。closure の引数は `c rune` になり、外の `r` を隠さない |
| N3 | 直った | `git var` の `true` の上に、group が別だと端末の Ctrl-C が ctx 経由でだけ届き、`regateGitSignaled` との競争にならない、という理由の 2 行がある(`:216-217`) |
| N4 | 直った | 登録簿の行数と行番号を外し、「800 行を超える」だけにした(`docs/tech-debt/README.md:277`、`:279`)。`regate_test.go` は 1508 行、`regate_merge_test.go` は 1371 行で、どちらも 800 行を超えているので文は今も合う |
| P1(cross-review) | 一部 | 直接の道(tip が足した path に ignored のファイルがある、その上のディレクトリが非ディレクトリ)は 85558f2c で閉じた。directory rename を挟む道が残る(C2-1)。確かめたこと: 6 例の test は、候補の積、parent がファイルの例(`!info.IsDir()`)、最後の component が symlink の例、同じ parent への束ね(`seen`)、10 件での打ち切り(`min`、`more`)のどれを消しても赤くなる形で(parent が symlink の例はない)、`onTipSide` を消すと `TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch` が、`--diff-filter=A` を消すと tip が書き換える追跡済みのファイル(`README.md`)で断る既存の test が赤くなる(読んだだけで、変異は走らせていない) |
| P2(cross-review) | 直った | 読めない gate の記録は、org が結びつきまたは読める記録を持った branch すべてに付く。読める記録の道と `unattached` の数え方は、旧コードと同じ入力で同じ結果(`orgBranches[org]` が空なのは、旧コードの `boundBranch` と `gatedBranch` が両方空のときだけ)。期待を変えた 3 つの test は、どれも `ready` が `unreadable` になる向きで、緩めたものはない。1 つは旧 test(`...LatestOfTwoEarlierBindings`)の入力のままで期待を反転した形(`...EveryBranchOfItsOrg` の 2 例目) |

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | MEDIUM | data-safety | `regateCheckInTheWay` の候補は「merge-base..tip で変わり、HEAD が持たない path」で、`--no-renames` で出す。doc は「merge が書きうる path はこれだ」と言い切る(`:243-246`)。git の merge は directory rename を検出し、片方が足したファイルを、もう片方が名前を変えたディレクトリの中の path に書く。その path は候補に入らないので、そこにある ignored のファイルを黙って上書きし、`git merge --abort` も戻さない。cross-review の P1 と同じ消え方で、道が違う。(a) base に `dir/a`。feat(HEAD)が `dir` を `dir2` に rename、main(tip)が `dir/new` を足し、worktree に ignored の `dir2/new` がある: 候補は `dir/new` だけで、`lstat dir` は無いので断らない。merge は `CONFLICT (file location)` で終了コード 1、`dir2/new` は tip の内容になり、abort のあと `dir2/new` ごと無い。(b) 逆向き(main が rename、feat が `dir/new` を足す): 候補は `dir2/a` だけで、`dir2/new` が上書きされる。条件は、directory rename、その中への追加、ignored のファイルが移った先と同じ path にあること、の 3 つ。重くない(HIGH にしない)理由はそこと、conflict で止まって利用者が気づくこと。消えるのは戻らないファイルで、直した指摘と同じ種類 | `internal/org/regate_merge.go:243-246`、`:255-306`(候補を作る `:265-294`)、probe(上の Evidence reviewed の(1)(2))、`docs/plans/active/2026-10-16-org-regate.md` D3 | 候補を merge の結果から取る。`git merge-tree --write-tree HEAD <tip>`(git 2.38 以降。`scripts/secret-scan-branch.sh` が既に使い、git が古いときや merge-tree が使えないときの扱いも持つ。衝突でも終了コード 1 で 1 行目に tree の ID を出す。git の既定の merge 設定で動くので、利用者の merge driver までは見ない)の tree と HEAD の `git diff --name-only -z --no-renames --diff-filter=A` を候補にすると、(a)(b) とも `dir2/new` が出る(確認した)。`regatePathInTheWay` はそのまま使える。git が古いなら今の求め方に戻すか、断る。この回で直さないなら、doc の「are those」を「may be those(a directory rename moves a path)」に直し、下の登録簿の行を足す |
| C2-2 | LOW | exception-handling | `regatePathInTheWay` は最後の component が directory なら、中身を見ずに「邪魔」とする(`:323`)。git が clean に merge する 2 つの入力でも断り、断りの文は「ignored files or directories ... that git merge overwrites without a word」と言う。(a) 空のディレクトリ(clean の確かめに見えない)の上に tip が同名のファイルを足す: merge は成功する。(b) HEAD が追跡するディレクトリ `x/` を tip がファイル `x` に置き換える(`x/a` を消して `x` を足す): merge は成功する。(b) で断りの指示どおり退かすと作業ツリーが汚れ、次の打ち直しは別の断りで止まる(`x/` の削除を commit して初めて通る) | `internal/org/regate_merge.go:313-327`、probe(Evidence reviewed の(3)) | 最後の component が directory のとき、空か、HEAD が追跡している(`git ls-tree HEAD -- <path>` が tree を返す)なら断らない。直さないなら、断りの文を「ignored」と言い切らず、「on disk」と言う |
| C2-3 | LOW | readability | `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` の doc は、読めない記録が「its org's latest complete binding before it」の branch に付くと書く(`mergeorder_test.go:932-938`)。f793924d は同じ規則を書いた 3 つの test の名前と doc を「every branch」に直したが、これは直していない。この test は org-a が記録の前に持つ branch が 1 つ(feat/x)だけなので、新旧どちらの規則でも通り、旧規則の文だけが残る。D2 が置き換えた規則を、次に読む人が学ぶ | `internal/org/mergeorder_test.go:932-938`(`:938` が関数)。同じ commit で直した 3 つは `:977`、`:1031`、`:1068` | 「binding の branch に付く(ここでは org が持った唯一の branch)。複数あるときは `...EveryBranchOfItsOrg`」の形に直す |
| C2-4 | LOW | doc-accuracy | spawn の help に足した「A spawn into an org is refused, with nothing recorded, while `ralph org regate` works on that org's branch; run it again once the regate ends.」は、`--dry-run` の spawn に当てはまらない。dry-run は `checkCapacityAndStart` の前の `spawn.go:600` で戻り、regate の確かめ(`:1284`)に届かない。`Spawn` の doc も、dry-run は step 6 の regate の確かめを省くと書く(`spawn.go:503-505`)。直前の段落は「Either refusal, --dry-run included」と書いているので、続けて読むと dry-run も断られると読める。「run it again」の「it」は spawn と regate のどちらにも取れる(start の help は「run start again」) | `internal/cli/org.go:677-679`、`internal/org/spawn.go:503-505`、`:600`、`:1284` | 「A spawn that is not a --dry-run is refused ... run the spawn again once the regate ends.」にする |
| C2-5 | LOW | readability | 新しい行が残した小さなずれ 3 つ。(a) `regate_merge.go:127-129`: 挿入のあと、`// (regateMergeTip, D3),` だけの短い行が、折り返した 2 行の間に残った(折り返しの直し漏れ)。(b) `:15-18`: 「the check of what is on disk in its way (regateCheckInTheWay), its undo」で、「its」が直前の「its way」(merge の)に続くので、undo が check のものに読める(元は「the merge commit (...), its undo」)。(c) `mergeorder.go:361` の field `orgBranches int`(数)と、同じ関数の `:404` の局所変数 `orgBranches map[string][]string` が同じ名前で、`:442` に `g.orgBranches = len(branches)` と並ぶ。`:402` の「in the order they first came」は、順に意味がない(足す branch は必ず target が既にあり、`target(branch)` は何も作らない。target の順は `targets` の first appearance のまま)ので、slice を dedupe にだけ使っているなら map で足りる | `internal/org/regate_merge.go:15-18`、`:127-129`、`internal/org/mergeorder.go:359-361`、`:402-409`、`:442` | (a) 折り直す。(b)「the merge commit's undo」と書く。(c) field を `orgBranchCount` などに変え、順の記述を外す(slice のままなら理由を 1 行) |

### Positive notes

- P2 の直しは、読める記録の道と `unattached` を旧コードと同じに保ったまま、読めない記録を付ける先だけを広げた。`target(branch)` は付ける先に必ず既にあるので、target の順は変わらず、`pos` が同じ target は `SortStableFunc` で first appearance の順に並ぶ(`mergeorder.go:894`)。Detail の追記は `g.orgBranches > 1` のときだけで、「another branch」が必ず別にある。`...EveryBranchOfItsOrg` が 2 例とも Detail の全文を固定している
- `regate_test.go` の新しい 1 例(`GatesWhenNoRecordLetsItStop` の org_id を別の branch に結んだあと)は、旧コードでは feat/auth が `ready` のまま `UP_TO_DATE` で止まる入力で、直しがないと赤くなる形(読んだだけ)
- `regateCheckInTheWay` は何も変えない前に走り、中断されても戻すものがない。error(git の失敗、lstat の読めない path、ctx)と断り(path が邪魔)の分け方が一貫している。関係のない履歴(merge-base の終了コード 1、stdout 空)は断らず、`git merge` が自分で断る(確かめた)。断りの一覧の 10 件の打ち切りは `more := len - 10` と `min` で合う
- N1 の直しの test は、error の全文だけでなく、merge commit が残ること、manifest と origin の ref が動かないこと、script が呼ばれないことまで見ている。d00da42c の legacy ledger の test は、shared と旧 ledger の manifest、origin の ref、fetch 前の tracking ref、`gates`・`regate.lock` が作られないことまで見ている
- 追加行に debug 出力、TODO、秘密、home の path はない。登録簿の行番号の pointer は外れた(N4)。cross-review の ID を指す新しい comment もない

### Coverage gaps

- テストは実行していない。C2-1 と C2-2 は git 2.49.0 の手の probe で確かめた(Go の関数は走らせていない。`regatePathInTheWay` の結果は、読んだコードと lstat の結果から導いた)
- `regateCheckInTheWay` の doc が言う 3 つの道に test がない。関係のない履歴(merge-base の終了コード 1)、merge-base が 128 で落ちる error、lstat が読めない path(権限のない parent)。挙動は上の probe と読みで確かめた。test の網羅は /test の範囲なので、finding にはしていない
- 大文字小文字を区別しない file system、submodule の path(空のディレクトリが断られる、C2-2 と同じ型)は確かめていない
- `/org` skill(4 面)の regate の行は、`user.name` の断りを書くが、ignored のファイルの断り(a59220f5 で help と doc にだけ足した)は書いていない。`start` の断りは `start --plan` とだけ書き、help の「Both forms」と食い違う。文書の整合は /sync-docs の範囲なので、finding にせず引き継ぐ
- plan の「Review artifact created(...1 周目の 2 回目)」の行は、このレポートの先頭が cycle 2 になったので、付録 A を指す文になる。/sync-docs で直す

### Tech debt identified

この回は cap 2 の最後なので、直さない finding は持ち越しになる。登録簿には書いていない(reviewer は書かない)。C2-2 から C2-5 を直さないなら、次の 1 行にまとめる。C2-1 も直さないなら、(a)として足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-regate PR, found by its cycle 2 self-review (all LOW except (a)). (a, MEDIUM, only if not fixed) `regateCheckInTheWay` (`internal/org/regate_merge.go`) derives the paths a merge may write from two `--no-renames` diffs, so the destination of a directory rename (a file one side adds into a directory the other side renamed) is not checked: `git merge` overwrites an ignored file there, and `git merge --abort` does not bring it back. A candidate set read from `git merge-tree --write-tree HEAD <tip>` covers it. (b) `regatePathInTheWay` refuses any directory at the path, including an empty one and a directory HEAD tracks that the tip replaces with a file, which git merges cleanly, and the refusal says "ignored". (c) `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` (`internal/org/mergeorder_test.go`) still describes the old attach rule. (d) The `ralph org spawn` help says a spawn is refused while a regate runs, which a `--dry-run` spawn is not. (e) Three comment and name slips: a short orphan line in `regateMerged`'s doc, "its undo" after "its way" in the file header of `regate_merge.go`, and `orgBranches` as both a field (a count) and a local map in `collectMergeTargets` | (a) A local ignored file is lost when a directory rename on one side meets an addition into it on the other, with an ignored file at the moved path; three coincidences, and the merge stops with a conflict. (b) A refused regate that git would have merged, with a misleading reason. (c) to (e) Misleading reading only | The plan's cap of 2 pipeline runs ends with this cycle, and (c) to (e) change no behavior | The next change to `regateCheckInTheWay`, `regatePathInTheWay`, `collectMergeTargets`, or the spawn help: (a) read the candidates from `git merge-tree --write-tree`; (b) skip an empty directory and a tree HEAD has; (c) to (e) fix as above | `docs/reports/self-review-2026-10-16-org-regate.md` (cycle 2, C2-1 to C2-5); `internal/org/regate_merge.go`; `internal/org/mergeorder.go`; `internal/cli/org.go` |

### Recommendation

- cycle 2 の判定: yes(CRITICAL・HIGH はなし。MEDIUM 1 件、LOW 4 件。cross-review の P2 は直り、P1 は直接の道が閉じて directory rename を挟む道が残る。N1〜N4 は直った。期待を変えた 3 つの test に緩めたものはない)
- Follow-ups:
  1. C2-1 は直すことを勧める。cross-review が ACTION_REQUIRED にした P1 と同じ種類(戻らないファイルの消失)で、直しは `regateCheckInTheWay` の候補の求め方と test の入力 2 つに収まる。直すなら cap 2 の次の回(cap を上げた 3 回目)になり、`regate_merge.go` に触れるので `/self-review` から回し直す。直さないなら、doc の「are those」を直し、上の登録簿の行に (a) を足す
  2. C2-2 から C2-5 は LOW。C2-1 を直す回に一緒に入れてよい。入れないなら、上の 1 行にまとめる
  3. `/sync-docs` への引き継ぎ: `/org` skill の regate の行に ignored のファイルの断りと、`start` の断りの範囲、plan の Review artifact の行(Coverage gaps)
  4. insight event は `--cycle 2` を明示して足した(自動の cycle 判定に頼らない)

## 付録 A: cycle 1 の 2 回目のレポート(07b9a465 の時点、931752d2 で書いた)

以下は、cycle 2 のレポートの前にこのファイルにあったレポートの全文。cycle 2 で見出しを 1 段下げ、`Merge:` の行を `cycle 1 の 2 回目の判定:` に変えた。cycle 3 で変えたのは、この 1 文だけ。

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認済み。8 段目の 8a-3b)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)の中の 2 回目。1 回目(8e515ca7)の M1 と L1〜L9 の直しを確かめた回で、`.harness/state/standard-pipeline/cycle-count.json` は 1 のまま
- Scope: diff の品質だけ。`git diff 670ec219...HEAD`(HEAD 07b9a465)全体を見たうえで、1 回目のあとの 6 コミット(d6a006dc、94ff3493、d441b921、a3cd529c、c4b9479b、07b9a465。21 ファイル、+1045/-369)を重点に読んだ。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)

### Evidence reviewed

- 読んだコード(全体): `internal/org/regate_git.go`(210 行すべて)、`internal/org/regate_merge.go`(503 行すべて)、`internal/org/regate.go` の 6 コミット分の diff すべてと `Regate` の doc・`regateInterrupted`・`regateWords`・`regateCheckInProgress`、`internal/org/gate.go`・`lockfile.go`・`mergeorder.go`・`internal/cli/org_regate.go` の diff すべて、`gateGit`・`gateGitOutput`・`gateHead` の本体、`gateSignalContext`(`internal/cli/org_gate.go:126`)
- 読んだテスト(diff): `regate_merge_test.go`、`regate_test.go`、`org_regate_test.go` の 6 コミット分すべて。実行はしていない
- 読んだ文書: `/org`・`/director` skill の diff(単語単位)、`AGENTS.md`、plan の Progress の追記、`docs/tech-debt/README.md` の追加と書き換え 3 行。skill は 2 つとも 4 面を `cmp` で比べ、byte 一致
- 手で確かめたこと(scratchpad の使い捨ての repo で git を直接打った。repo の tree には何も書いていない): (1) `pre-merge-commit` hook が `kill -KILL $PPID` する merge は、終了コード 137、`MERGE_HEAD` なし、`A  h` が staged、HEAD は merge 前のまま。`git merge --abort` は「There is no merge to abort」で終わり、`git reset --keep <pre>` は `h` を untracked として files に残し、`git reset --merge <pre>` は `h` ごと消して clean に戻す(`regateMergeNotBack` の `moved=false` の主張どおり)。(2) `post-merge` hook が git を殺す merge は、merge commit ができ、`MERGE_HEAD` が残り(`git status` は「still merging」)、`git reset --keep <pre>` は「Cannot do a keep reset in the middle of a merge」で断られる。`git merge --abort` を先に打つと `MERGE_HEAD` が消え、HEAD は merge commit のままで、そのあと `git reset --keep <pre>` が通る。`regateMergeTip` は abort を `reset` の前に打つので、この順は `TestRegate_AMergeEndedAfterItsCommitNamesResetKeep` の流れと合う
- Gate の文面が変わっていないことの確認: `gateCheckBranch` の rebase の文(`"%s, and %s"` に `gateRebaseSteps` を入れた形が元の文と 1 字ずつ一致)、`gateCheckScript` の 4 文と `gateDirtyRefusal`(`afterCommit` が空なら元の文と一致)を 1 文ずつ見た。`TestGateWording_RefusalsThatAskForACommit` が Gate の 5 文を byte で固定している。`gateWording{` の literal は `gateWords` と `regateWords` の 2 つだけで、`rebaseSteps` が空の語り口は作られない
- 追加で grep: 追加行の debug 出力・TODO・`/Users/`・秘密の語・生成物(なし)、`regateRemoteTimeout`・`regateRemoteWaitDelay`・`regateNoIdentity` の旧名の残り(コードと文書にない。`regateNoIdentityEnv` はテストの helper で別物)、`regate.go` の import の使用(`bytes`・`io`・`fs`・`strings` はどれも使われている)、テスト名と関数名(登録簿の Trigger に書かれた 5 つのテストと `writeRegateMarker`・`readRegateMarker` が実在)、登録簿の追加行の列数(5 列ずつ)と取り消し線の形、`regateMergeTip`・`regateMerged`・`Regate` の行数。`gofmt -l` を 1 回、一覧表示だけで打った(出力なし、書き換えなし)。「整形ツールを走らせない」に触れるので、証拠には数えていない

### 1 回目の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | merge(`regate_merge.go:242`)、abort(`:287`)、reset(`:391`)は `regateUncancelledGit`(`:28`)を通り、`regateBoundedGit` の `ownGroup=true`(`regate_git.go:179-185`、`Setpgid` と、bound のときの group への SIGKILL)で別の group に入る。`git var`(`regate_merge.go:204`)も `ownGroup=true`。fetch(`regate_git.go:35`)と push(`regate_merge.go:448`)は `regateRemoteGit`(`ownGroup=false`)で ralph の group に残り、ssh の鍵の入力を止めない。git が signal で終わった場合は `regateGitSignaled` で状態を言う(merge は `regateMergeNotBack`、fetch と push は「nothing was recorded」と「origin may or may not have it」)。`merge` を ralph の group 外で動かすことは `TestRegate_MergesTheTipInAndPushes` の `pgid` の記録で、fetch と push が group 内にあることは同じ形の記録で固定されている。割り込みの待ちの 1 行は下の「stderr の 1 行」に書いた |
| L1 | 直った | 名前が `regateBoundedGit`(上限と env を付ける runner)、`regateRemoteGit`(origin に出る git)、`regateUncancelledGit`(別 group で ctx が止めない git)に分かれた。env は fetch が `GIT_TERMINAL_PROMPT=0` と `LC_ALL=C`(`regateFetchEnv`)、push が `GIT_TERMINAL_PROMPT=0` だけ(`regateRemoteEnv`)、merge が `GIT_MERGE_AUTOEDIT=no` だけ(`regateMergeEnv`)、identity の確かめ・abort・undo は env そのまま。merge の hook は呼び出し元の locale で動く。`RegateParams.Env` の doc がこの対応を書き直している |
| L2 | 直った | `git diff --name-only --diff-filter=U -z` を NUL で分ける(`regate_merge.go:279`、`:319`)。`a b.txt` と `ä.txt` の衝突を `TestRegate_RefusedWhenTheMergeCommitCannotBeMade` が固定し、`\303` と `"` が出ないことも見ている |
| L3 | 直った | (a) `case GateReasonTreeChanged:` と書き、`default:` は「どの reason も書いていない」ときの別の文になった(`:169-190`)。reason は 4 つ(`gate.go:48-56`)でどれも case がある。(b) `printRegateResult` は `res.Outcome == UpToDate || ... == PushOnly || res.Passed` になり(`org_regate.go:187`)、4 つの Outcome の値で元の switch と同じ結果 |
| L4 | 直った | `mergeRegateFailedDetail` の doc(`mergeorder.go:970-982`)が折り直された |
| L5 | 直った | `heldFileLock` の doc に `regate.lock`(長さ、印を書くこと、`regateSpawnErr` が読むこと)が入り、`acquireHeldFileLock` の doc が「spawn の確かめと `ProbeWatch` の確かめ」を挙げる。`fileLockHeld` の呼び出しは `spawn.go:1329` と `watchalive.go:209` の 2 か所で、doc と一致 |
| L6 | 直った | `/director` skill の例が `feat/auth-token` と `regate-feat-auth-token.out` になり、`<名前>` の決め方(`/` を `-` に)が 1 文で書かれた(`.claude/skills/director/SKILL.md:467-475`)。`/org` skill の例も同じ名前に揃った(`:1024-1028`)。4 面とも byte 一致 |
| L7 | 一部 | 登録簿の pointer は `:299-303` になり、関数の宣言から `time.Sleep` までを指す(正しい)。1 回目の推奨(行番号を外して関数名だけにする)は採られていない。N4 に含める |
| L8 | 直った | `gateWording.afterCommit` で Gate は空、regate は「push the commit to origin's <branch>」を足す(`gate.go:401-408`、`regate.go:346-355`)。Gate の文は変わらず、`TestGateWording_RefusalsThatAskForACommit` が 2 つの語り口の 10 文を固定している |
| L9 | 直った | fetch・origin head の確かめ・env・bounded git が `regate_git.go` に移り、`regate.go` は 728 行(800 行未満)。テストは分けておらず、`regate_test.go`(1499 行)と `regate_merge_test.go`(984 行)が 800 行を超えることを登録簿の新しい行が追跡している(`docs/tech-debt/README.md:279`)。解消した行は同じ表の他の行と同じ形(上の `<!-- RESOLVED ... -->`、Debt と Trigger の取り消し線) |

#### stderr の 1 行(d441b921)

`regateUncancelledGit`(`regate_merge.go:28-44`)の `context.AfterFunc` を読んだ。(1) data race はない。f が書く `stderr` は f の中だけで、f の `defer close(said)` と、`stop()` が false のときの `<-said` で、関数が返る前に f が終わる。`stop()` が true なら f は走らない。f が読む `regateGitTimeout` を書くテストの `t.Cleanup` は、`Regate` が返ったあとに走る。(2) 1 回の Regate で 1 行以下。ctx は一度 done になると戻らないので、行を書けるのは ctx が done でなかった最初の `regateUncancelledGit` だけで、その後の呼び出しは先頭の `ctx.Err() != nil`(`:29`)で別の道に入る。(3) 通常の終わりでは書かれない(`stop()` が true)。(4) 割り込みの後に始まる abort と undo は書かない(`:29`)。残る窓は、ctx が `:29` の確かめと `AfterFunc` の登録の間に done になる場合(行は書かれ、git は走るので内容は正しい)と、git が終わる直前に done になる場合(待ちの行が、待つ必要のない git について出る)で、どちらも害はない。`TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits` は ralph 自身にだけ SIGINT を送る(`signalSelf`)ので、端末の Ctrl-C の group 宛ての届き方は、`pgid` が ralph と違うという記録で代わりに確かめている

### Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | exception-handling | merge が成功したあと、`regateMergeTip` は HEAD と親を読む(`gateHead`、`regateParents`)。この 2 つが失敗すると、git の素の error をそのまま返し、merge commit ができたこと、記録も undo もされていないこと、戻し方を言わない。関数の doc(`:232-236`)と呼び出し側の注(`regateMerged` の `:144-145`)は、error が「worktree の状態と戻し方を言う」と読めるが、その約束が通るのは `regateMergeNotBack` を通る返り値だけ。この読み取りは ralph の group で、`context.WithoutCancel` の ctx で、上限なしで動く(`gateGit` は timeout を持たない)ので、M1 が merge・abort・reset について閉じた端末の Ctrl-C は、merge が終わった直後の数 ms の間は、この git を直接殺す(`gateGit` は signal で終わった git を `signal: interrupt` の error にする)。そうなると merge commit は記録も undo もなく残り、次の regate は `regateCheckOriginHead` で「HEAD has commits that origin's tip does not: push them」と言い、`regateResetHint` は gate の記録がないので何も足さない。つまり、gate を通っていない merge commit の push を勧める。窓は狭い。確かめたのはコードの読みで、窓を作る再現はしていない。同じ形の読み取り(`:279`、`:280`、`:302`、`:306`)は、失敗すると `regateMergeNotBack` を返すので、足りないのはこの 2 つだけ | `internal/org/regate_merge.go:250-257`、`:232-236`、`internal/org/regate.go` の `regateInterrupted`(ctx が done でなければ err をそのまま返す)、`internal/org/gate.go:745-762`(`gateGit`)、`internal/org/regate_git.go:77-107`(`regateCheckOriginHead`) | `:250-257` の 2 つの return を、`regateMergeNotBack(t, command, "made a merge commit", fmt.Sprintf("HEAD cannot be read (%v)", err), true)` の形にする(merge commit が作られたあとの読み取りなので `moved=true`、案内は `reset --keep <t.head>`)。doc の「error は…」の列挙に、この読み取りも入れる |
| N2 | LOW | readability | `regateMergeTip` が 99 行になった(1 回目の時点で 82 行、この直しで +17 行)。次の 3 つを 1 つの関数が持つ: merge を走らせて親を確かめる(`:242-265`)、失敗した merge の状態を読んで abort する(`:274-313`)、断りの文を組む(`:315-335`)。`:274-313` は `aborted` の文字列か error だけを返す閉じた塊で、切り出せる。同じ塊の中に、`switch conflicted := strings.FieldsFunc(files, func(r rune) bool { return r == 0 }); {` の `r` が、外の `r`(`gitResult`、`:242`)を隠している箇所がある(`:319`)。1 回目に `Regate` は 76 行、`regateMerged` は 47 行(今 60 行)で、リポジトリは長い関数を許していないわけではないが、この関数だけが 1 回の直しで伸び続けている | `internal/org/regate_merge.go:238-336`、`:319` | `:274-313` を `regateAbortLeftMerge(ctx, t, env, stderr, command, outcome) (aborted string, err error)` のような関数に切り出す。`:319` の closure の引数は `c rune` などに変える |
| N3 | LOW | maintainability | `regateCheckIdentity` だけが `regateBoundedGit` を `ownGroup=true` の裸の `true` で直に呼ぶ(`:204`)。ほかの `true` は `regateUncancelledGit` の中にあり、その doc が理由(merge・abort・undo を中途で止めない)を書いている。`git var` は hook を走らせず、端末も読まず、ctx で止めてよい git なので、この `true` の理由(端末の Ctrl-C が `git var` を signal で終わらせて、`regateGitSignaled` の error と ctx の error のどちらが返るかが競争になるのを避ける、が筋に見える)はコードにも doc にも plan にも書かれていない(plan は「`git var` を別のプロセスグループで動かす」とだけ書く)。次にこの行を触る人は、`ownGroup=false` に戻してよいかを判断できない | `internal/org/regate_merge.go:204`、`internal/org/regate_git.go:160-172`(`ownGroup` の doc)、`docs/plans/active/2026-10-16-org-regate.md:151` | `:204` の上に 1 行、理由を書く |
| N4 | LOW | maintainability | 登録簿の新しい行と RESOLVED の注が書く行数が、HEAD で合っていない。`regate.go is 713 lines`(`docs/tech-debt/README.md:277`、書いた 94ff3493 の時点では 713、HEAD では 728)と、`regate_test.go is 1495 lines`(`:279`、同じく 1495 から 1499)。どちらも書いた 1〜4 コミットあとの本 PR の中で崩れている。1 回目の L7(pointer に行番号を使わない)と同じ種類で、L7 の直しは `:299-303` に直しただけで行番号を残した | `docs/tech-debt/README.md:276`、`:277`、`:279`、`git show c4b9479b:internal/org/regate.go`(728 行)、`git show c4b9479b:internal/org/regate_test.go`(1499 行) | 行数は「約 700 行」「約 1500 行」にするか、数を外して「800 行を超える」だけにする。`:276` は行番号を外して関数名だけにする |

### Positive notes

- M1 の直しは、1 回目の推奨(merge・abort・reset を別の group に、fetch と push は ralph の group に残す)をそのまま実装し、`git var` まで同じ形に揃えた。signal で終わった git の 5 つの状態(merge の前で止まる、merge commit の前で止まって index に結果が残る、merge commit のあとで止まる、fetch、push)に、状態を言う error とテストがある。そのうち 2 つ(index に結果が残る、merge commit のあと)は上の(1)(2)の手の再現と、`reset --merge` と `reset --keep` の使い分けが一致した
- `regateMergeNotBack` の `moved` の分け方が正しい。HEAD が動いていないときは `reset --merge`(merge の途中も、index に残った結果も戻す)、動いたときは `reset --keep`(利用者の変更を落とさず止まる)。`reset --keep` が index に残った結果を files に残すことと、merge の途中では断られることを手で確かめた。doc の説明(`:338-350`)はそのとおり
- Gate の文面を変えずに共通の語り口を持たせた。`gateRebaseSteps` を const にし、`gateWords` に入れ、regate は別の文を持つ。文の 1 字ずつの一致をテストが固定している。`afterCommit` は commit を頼む 5 文すべてに入り、ほかに commit を頼む文は `gate.go` と `regate*.go` にない
- 割り込みの待ちの 1 行は、`AfterFunc` と `said` の組が小さく、race と二重書き込みの道がない(上の確認)。`RegateParams.Stderr` が nil のときの既定は `os.Stderr` で、CLI は `cmd.ErrOrStderr()` を渡す
- `regateGitTimeout`、`regateGitWaitDelay` と旧名(`regateRemoteTimeout` など)の置き換えは、コード・テスト・文書のどこにも旧名を残していない
- skill の `/org` と `/director` の 4 面が byte 一致し、割り込みと git var の文(`/org` skill の `regate` の行)がコードと合う。`<名前>` の決め方の追記は両方の skill にある
- 登録簿の解消した行が、同じ表の他の解消済みの行と同じ形で、新しい行も 5 列。Trigger に書かれた 5 つのテスト名と 2 つの関数名がすべて実在する

### Coverage gaps

- テストは実行していない。`pgid` の記録を使うテストが `ps -o pgid= -p $$` を使うので、実行環境に `ps` が要る(macOS と Linux の標準の `ps` で動く形)
- `git var`(`regate_merge.go:204`)の group は、hook も端末もない git なので、どのテストからも観測できない。`ownGroup=true` を `false` に変えてもテストは落ちないはず(読んだだけで、変異は走らせていない)。N3 の 1 行の理由が書かれれば、この箇所の意図は残る
- 端末の Ctrl-C を ralph の group に送る実機の確認はしていない(1 回目と同じ。process group が分かれていることをソースと `pgid` の記録で確かめた)
- N1 の窓(merge 直後の読み取りに Ctrl-C が当たる)は再現していない

### Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(今回は足さない)。N1〜N4 は直すコストが小さい。cycle-count は 1 なので最終回ではなく、今は登録簿に書かず直す | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。N1〜N4 を直さずにパイプラインが進み、cycle 2(cap 2 の最後の回)に入るなら、cycle 2 のレビューで 1 行にまとめる。)_

### Recommendation

- cycle 1 の 2 回目の判定: yes(CRITICAL・HIGH・MEDIUM はなし。1 回目の M1 は直り、L1〜L9 は L7 が一部のほかすべて直った。新しい指摘は LOW 4 件で、merge 直後の読み取りの error が merge commit の存在を言わない、`regateMergeTip` の長さと名前の隠れ、`git var` の group の理由が書かれていない、登録簿の行数の古さ)
- Follow-ups:
  1. N1 は Go の小さな直し(2 つの return)。N2 と N3 は同じ commit に入れてよい。N4 は文書だけで、N1〜N3 の commit に添えるか `/sync-docs` で入れる
  2. N1〜N4 の直しは、`regate_merge.go` の返り値と doc の文面に触れるので、直すなら `/self-review` から回し直す(cycle-count は 1 のままで、この回の延長)
  3. この回は同じ cycle の確認なので、`docs/insights/events/2026-10-16-org-regate.jsonl` に `self_review` の行を足していない(phase と cycle が集計の鍵で、足すと二重に数える)
