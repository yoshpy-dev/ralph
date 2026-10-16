# Self-review report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認済み。8 段目の 8a-3b)
- Reviewer: reviewer subagent (Claude)。パイプライン 2 回目(cycle 2)。`.harness/state/standard-pipeline/cycle-count.json` は 2 で、既定の cap 2 の最後の回
- Scope: diff の品質だけ。`git diff 670ec219...HEAD`(HEAD a59220f5)全体を見たうえで、cycle 1 の cross-review のあとの 3 コミット(85558f2c、f793924d、a59220f5)と、前回の self-review(07b9a465 の時点)が見ていない 0b6ba232、d00da42c、5e4ce6cd、413ab307 を重点に読んだ。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)
- ID: 今回の指摘は `C2-n`。cycle 1 の 2 回目のレポート(M1、L1〜L9、N1〜N4)は、登録簿が「self-review fix L9」で指しているので、全文を末尾の「付録 A」に残した。その `Merge:` の行は `cycle 1 の 2 回目の判定:` に変えてある

## Evidence reviewed

- 読んだコード: 85558f2c の `regateCheckInTheWay`・`regatePathInTheWay`・`regateMerged` の呼び出し(`internal/org/regate_merge.go:235-327`、`:139-155`)。f793924d の `collectMergeTargets`(`internal/org/mergeorder.go:373-461`)、`mergeGateRecord.orgBranches`、`classifyMergeTarget` の Detail(`:935-950`)。`collectMergeTargets` を読む `readRegateBranch`・`passedOn`・`recordRegate`(`internal/org/regate.go`)。0b6ba232 の `regateMergeUnchecked`・`regateAbortLeftMerge`・`regateMergeTip`(`regate_merge.go:349-477`)。413ab307 の help と `regateSpawnErr` の呼び出し元(`spawn.go:1284`、`feature.go:216`)、dry-run の分岐(`spawn.go:600`)。a59220f5 の help と doc
- 読んだテスト(diff。実行はしていない): 85558f2c の 186 行(`TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay` の 6 例と `TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch`)、f793924d の `mergeorder_test.go` と `regate_test.go`(期待を変えた 3 つの test と、足した 1 つ)、0b6ba232・d00da42c・5e4ce6cd の test
- 手で確かめたこと(scratchpad の使い捨ての repo、git 2.49.0。repo の tree には何も書いていない): (1) directory rename を挟む merge が ignored のファイルを上書きし、`merge --abort` のあとも戻らないこと(C2-1 の(a)(b))。(2) その 2 例を `git merge-tree --write-tree HEAD <tip>` の結果の tree と HEAD の `--diff-filter=A` の差で見ると、上書きされる path が出ること。(3) 空のディレクトリ、HEAD が追跡するディレクトリをファイルに置き換える tip は、clean に merge できること(C2-2)。(4) 関係のない履歴では `git merge-base` が終了コード 1・stdout 空、`git merge --no-ff` が終了コード 128・`MERGE_HEAD` なしで終わること(`regateCheckInTheWay` の doc のとおり)
- 追加で grep: 追加行の debug 出力・TODO・`/Users/`・秘密の語(なし)、cross-review の ID を指す comment(新しい行にはなし。`recordGate` の「cross-review cycle 1 #1」は 670ec219 からある)、`gatedBranch` の残り(なし)、`mergeorder.go` の `cmp`・`slices` の import が今も使われていること(`:501`、`:894`、`:406`)、`SortStableFunc` で `pos` が同じ target が並ぶ順(first appearance、決まった順)

## cycle 1 の指摘の状態

| ID | 状態 | 確かめたこと |
| --- | --- | --- |
| N1 | 直った | `regateMergeTip` の HEAD と親の読み取り 2 か所と、親が違う場合が `regateMergeUnchecked`(`regate_merge.go:418`)を通る。error は merge commit ができたこと、読めないものとその理由、script も記録もしなかったこと、`git reset --keep <merge 前の HEAD>` を言う。`TestRegate_AMergeCommitItCannotReadNamesResetKeep`(`regate_merge_test.go:411`)が HEAD と親の 2 通りを、error の全文と、merge commit が残ること、manifest・origin・script が動かないことまで固定している。推奨は `regateMergeNotBack` に通すことだったが、専用の constructor になった。`regateMergeNotBack` にある「run ralph org regate --branch X again」は `regateMergeUnchecked` にないが、元の親違いの文にもなかった |
| N2 | 直った | `regateAbortLeftMerge`(`:434`、44 行)を切り出し、`regateMergeTip` は 99 行から 62 行。closure の引数は `c rune` になり、外の `r` を隠さない |
| N3 | 直った | `git var` の `true` の上に、group が別だと端末の Ctrl-C が ctx 経由でだけ届き、`regateGitSignaled` との競争にならない、という理由の 2 行がある(`:216-217`) |
| N4 | 直った | 登録簿の行数と行番号を外し、「800 行を超える」だけにした(`docs/tech-debt/README.md:277`、`:279`)。`regate_test.go` は 1508 行、`regate_merge_test.go` は 1371 行で、どちらも 800 行を超えているので文は今も合う |
| P1(cross-review) | 一部 | 直接の道(tip が足した path に ignored のファイルがある、その上のディレクトリが非ディレクトリ)は 85558f2c で閉じた。directory rename を挟む道が残る(C2-1)。確かめたこと: 6 例の test は、候補の積、parent がファイルの例(`!info.IsDir()`)、最後の component が symlink の例、同じ parent への束ね(`seen`)、10 件での打ち切り(`min`、`more`)のどれを消しても赤くなる形で(parent が symlink の例はない)、`onTipSide` を消すと `TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch` が、`--diff-filter=A` を消すと tip が書き換える追跡済みのファイル(`README.md`)で断る既存の test が赤くなる(読んだだけで、変異は走らせていない) |
| P2(cross-review) | 直った | 読めない gate の記録は、org が結びつきまたは読める記録を持った branch すべてに付く。読める記録の道と `unattached` の数え方は、旧コードと同じ入力で同じ結果(`orgBranches[org]` が空なのは、旧コードの `boundBranch` と `gatedBranch` が両方空のときだけ)。期待を変えた 3 つの test は、どれも `ready` が `unreadable` になる向きで、緩めたものはない。1 つは旧 test(`...LatestOfTwoEarlierBindings`)の入力のままで期待を反転した形(`...EveryBranchOfItsOrg` の 2 例目) |

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | MEDIUM | data-safety | `regateCheckInTheWay` の候補は「merge-base..tip で変わり、HEAD が持たない path」で、`--no-renames` で出す。doc は「merge が書きうる path はこれだ」と言い切る(`:243-246`)。git の merge は directory rename を検出し、片方が足したファイルを、もう片方が名前を変えたディレクトリの中の path に書く。その path は候補に入らないので、そこにある ignored のファイルを黙って上書きし、`git merge --abort` も戻さない。cross-review の P1 と同じ消え方で、道が違う。(a) base に `dir/a`。feat(HEAD)が `dir` を `dir2` に rename、main(tip)が `dir/new` を足し、worktree に ignored の `dir2/new` がある: 候補は `dir/new` だけで、`lstat dir` は無いので断らない。merge は `CONFLICT (file location)` で終了コード 1、`dir2/new` は tip の内容になり、abort のあと `dir2/new` ごと無い。(b) 逆向き(main が rename、feat が `dir/new` を足す): 候補は `dir2/a` だけで、`dir2/new` が上書きされる。条件は、directory rename、その中への追加、ignored のファイルが移った先と同じ path にあること、の 3 つ。重くない(HIGH にしない)理由はそこと、conflict で止まって利用者が気づくこと。消えるのは戻らないファイルで、直した指摘と同じ種類 | `internal/org/regate_merge.go:243-246`、`:255-306`(候補を作る `:265-294`)、probe(上の Evidence reviewed の(1)(2))、`docs/plans/active/2026-10-16-org-regate.md` D3 | 候補を merge の結果から取る。`git merge-tree --write-tree HEAD <tip>`(git 2.38 以降。`scripts/secret-scan-branch.sh` が既に使い、git が古いときや merge-tree が使えないときの扱いも持つ。衝突でも終了コード 1 で 1 行目に tree の ID を出す。git の既定の merge 設定で動くので、利用者の merge driver までは見ない)の tree と HEAD の `git diff --name-only -z --no-renames --diff-filter=A` を候補にすると、(a)(b) とも `dir2/new` が出る(確認した)。`regatePathInTheWay` はそのまま使える。git が古いなら今の求め方に戻すか、断る。この回で直さないなら、doc の「are those」を「may be those(a directory rename moves a path)」に直し、下の登録簿の行を足す |
| C2-2 | LOW | exception-handling | `regatePathInTheWay` は最後の component が directory なら、中身を見ずに「邪魔」とする(`:323`)。git が clean に merge する 2 つの入力でも断り、断りの文は「ignored files or directories ... that git merge overwrites without a word」と言う。(a) 空のディレクトリ(clean の確かめに見えない)の上に tip が同名のファイルを足す: merge は成功する。(b) HEAD が追跡するディレクトリ `x/` を tip がファイル `x` に置き換える(`x/a` を消して `x` を足す): merge は成功する。(b) で断りの指示どおり退かすと作業ツリーが汚れ、次の打ち直しは別の断りで止まる(`x/` の削除を commit して初めて通る) | `internal/org/regate_merge.go:313-327`、probe(Evidence reviewed の(3)) | 最後の component が directory のとき、空か、HEAD が追跡している(`git ls-tree HEAD -- <path>` が tree を返す)なら断らない。直さないなら、断りの文を「ignored」と言い切らず、「on disk」と言う |
| C2-3 | LOW | readability | `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` の doc は、読めない記録が「its org's latest complete binding before it」の branch に付くと書く(`mergeorder_test.go:932-938`)。f793924d は同じ規則を書いた 3 つの test の名前と doc を「every branch」に直したが、これは直していない。この test は org-a が記録の前に持つ branch が 1 つ(feat/x)だけなので、新旧どちらの規則でも通り、旧規則の文だけが残る。D2 が置き換えた規則を、次に読む人が学ぶ | `internal/org/mergeorder_test.go:932-938`(`:938` が関数)。同じ commit で直した 3 つは `:977`、`:1031`、`:1068` | 「binding の branch に付く(ここでは org が持った唯一の branch)。複数あるときは `...EveryBranchOfItsOrg`」の形に直す |
| C2-4 | LOW | doc-accuracy | spawn の help に足した「A spawn into an org is refused, with nothing recorded, while `ralph org regate` works on that org's branch; run it again once the regate ends.」は、`--dry-run` の spawn に当てはまらない。dry-run は `checkCapacityAndStart` の前の `spawn.go:600` で戻り、regate の確かめ(`:1284`)に届かない。`Spawn` の doc も、dry-run は step 6 の regate の確かめを省くと書く(`spawn.go:503-505`)。直前の段落は「Either refusal, --dry-run included」と書いているので、続けて読むと dry-run も断られると読める。「run it again」の「it」は spawn と regate のどちらにも取れる(start の help は「run start again」) | `internal/cli/org.go:677-679`、`internal/org/spawn.go:503-505`、`:600`、`:1284` | 「A spawn that is not a --dry-run is refused ... run the spawn again once the regate ends.」にする |
| C2-5 | LOW | readability | 新しい行が残した小さなずれ 3 つ。(a) `regate_merge.go:127-129`: 挿入のあと、`// (regateMergeTip, D3),` だけの短い行が、折り返した 2 行の間に残った(折り返しの直し漏れ)。(b) `:15-18`: 「the check of what is on disk in its way (regateCheckInTheWay), its undo」で、「its」が直前の「its way」(merge の)に続くので、undo が check のものに読める(元は「the merge commit (...), its undo」)。(c) `mergeorder.go:361` の field `orgBranches int`(数)と、同じ関数の `:404` の局所変数 `orgBranches map[string][]string` が同じ名前で、`:442` に `g.orgBranches = len(branches)` と並ぶ。`:402` の「in the order they first came」は、順に意味がない(足す branch は必ず target が既にあり、`target(branch)` は何も作らない。target の順は `targets` の first appearance のまま)ので、slice を dedupe にだけ使っているなら map で足りる | `internal/org/regate_merge.go:15-18`、`:127-129`、`internal/org/mergeorder.go:359-361`、`:402-409`、`:442` | (a) 折り直す。(b)「the merge commit's undo」と書く。(c) field を `orgBranchCount` などに変え、順の記述を外す(slice のままなら理由を 1 行) |

## Positive notes

- P2 の直しは、読める記録の道と `unattached` を旧コードと同じに保ったまま、読めない記録を付ける先だけを広げた。`target(branch)` は付ける先に必ず既にあるので、target の順は変わらず、`pos` が同じ target は `SortStableFunc` で first appearance の順に並ぶ(`mergeorder.go:894`)。Detail の追記は `g.orgBranches > 1` のときだけで、「another branch」が必ず別にある。`...EveryBranchOfItsOrg` が 2 例とも Detail の全文を固定している
- `regate_test.go` の新しい 1 例(`GatesWhenNoRecordLetsItStop` の org_id を別の branch に結んだあと)は、旧コードでは feat/auth が `ready` のまま `UP_TO_DATE` で止まる入力で、直しがないと赤くなる形(読んだだけ)
- `regateCheckInTheWay` は何も変えない前に走り、中断されても戻すものがない。error(git の失敗、lstat の読めない path、ctx)と断り(path が邪魔)の分け方が一貫している。関係のない履歴(merge-base の終了コード 1、stdout 空)は断らず、`git merge` が自分で断る(確かめた)。断りの一覧の 10 件の打ち切りは `more := len - 10` と `min` で合う
- N1 の直しの test は、error の全文だけでなく、merge commit が残ること、manifest と origin の ref が動かないこと、script が呼ばれないことまで見ている。d00da42c の legacy ledger の test は、shared と旧 ledger の manifest、origin の ref、fetch 前の tracking ref、`gates`・`regate.lock` が作られないことまで見ている
- 追加行に debug 出力、TODO、秘密、home の path はない。登録簿の行番号の pointer は外れた(N4)。cross-review の ID を指す新しい comment もない

## Coverage gaps

- テストは実行していない。C2-1 と C2-2 は git 2.49.0 の手の probe で確かめた(Go の関数は走らせていない。`regatePathInTheWay` の結果は、読んだコードと lstat の結果から導いた)
- `regateCheckInTheWay` の doc が言う 3 つの道に test がない。関係のない履歴(merge-base の終了コード 1)、merge-base が 128 で落ちる error、lstat が読めない path(権限のない parent)。挙動は上の probe と読みで確かめた。test の網羅は /test の範囲なので、finding にはしていない
- 大文字小文字を区別しない file system、submodule の path(空のディレクトリが断られる、C2-2 と同じ型)は確かめていない
- `/org` skill(4 面)の regate の行は、`user.name` の断りを書くが、ignored のファイルの断り(a59220f5 で help と doc にだけ足した)は書いていない。`start` の断りは `start --plan` とだけ書き、help の「Both forms」と食い違う。文書の整合は /sync-docs の範囲なので、finding にせず引き継ぐ
- plan の「Review artifact created(...1 周目の 2 回目)」の行は、このレポートの先頭が cycle 2 になったので、付録 A を指す文になる。/sync-docs で直す

## Tech debt identified

この回は cap 2 の最後なので、直さない finding は持ち越しになる。登録簿には書いていない(reviewer は書かない)。C2-2 から C2-5 を直さないなら、次の 1 行にまとめる。C2-1 も直さないなら、(a)として足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Small leftovers of the org-regate PR, found by its cycle 2 self-review (all LOW except (a)). (a, MEDIUM, only if not fixed) `regateCheckInTheWay` (`internal/org/regate_merge.go`) derives the paths a merge may write from two `--no-renames` diffs, so the destination of a directory rename (a file one side adds into a directory the other side renamed) is not checked: `git merge` overwrites an ignored file there, and `git merge --abort` does not bring it back. A candidate set read from `git merge-tree --write-tree HEAD <tip>` covers it. (b) `regatePathInTheWay` refuses any directory at the path, including an empty one and a directory HEAD tracks that the tip replaces with a file, which git merges cleanly, and the refusal says "ignored". (c) `TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` (`internal/org/mergeorder_test.go`) still describes the old attach rule. (d) The `ralph org spawn` help says a spawn is refused while a regate runs, which a `--dry-run` spawn is not. (e) Three comment and name slips: a short orphan line in `regateMerged`'s doc, "its undo" after "its way" in the file header of `regate_merge.go`, and `orgBranches` as both a field (a count) and a local map in `collectMergeTargets` | (a) A local ignored file is lost when a directory rename on one side meets an addition into it on the other, with an ignored file at the moved path; three coincidences, and the merge stops with a conflict. (b) A refused regate that git would have merged, with a misleading reason. (c) to (e) Misleading reading only | The plan's cap of 2 pipeline runs ends with this cycle, and (c) to (e) change no behavior | The next change to `regateCheckInTheWay`, `regatePathInTheWay`, `collectMergeTargets`, or the spawn help: (a) read the candidates from `git merge-tree --write-tree`; (b) skip an empty directory and a tree HEAD has; (c) to (e) fix as above | `docs/reports/self-review-2026-10-16-org-regate.md` (cycle 2, C2-1 to C2-5); `internal/org/regate_merge.go`; `internal/org/mergeorder.go`; `internal/cli/org.go` |

## Recommendation

- Merge: yes(CRITICAL・HIGH はなし。MEDIUM 1 件、LOW 4 件。cross-review の P2 は直り、P1 は直接の道が閉じて directory rename を挟む道が残る。N1〜N4 は直った。期待を変えた 3 つの test に緩めたものはない)
- Follow-ups:
  1. C2-1 は直すことを勧める。cross-review が ACTION_REQUIRED にした P1 と同じ種類(戻らないファイルの消失)で、直しは `regateCheckInTheWay` の候補の求め方と test の入力 2 つに収まる。直すなら cap 2 の次の回(cap を上げた 3 回目)になり、`regate_merge.go` に触れるので `/self-review` から回し直す。直さないなら、doc の「are those」を直し、上の登録簿の行に (a) を足す
  2. C2-2 から C2-5 は LOW。C2-1 を直す回に一緒に入れてよい。入れないなら、上の 1 行にまとめる
  3. `/sync-docs` への引き継ぎ: `/org` skill の regate の行に ignored のファイルの断りと、`start` の断りの範囲、plan の Review artifact の行(Coverage gaps)
  4. insight event は `--cycle 2` を明示して足した(自動の cycle 判定に頼らない)

## 付録 A: cycle 1 の 2 回目のレポート(07b9a465 の時点、931752d2 で書いた)

以下は、この cycle 2 の前にこのファイルにあったレポートの全文。見出しを 1 段下げ、`Merge:` の行を `cycle 1 の 2 回目の判定:` に変えただけで、ほかは変えていない。

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
