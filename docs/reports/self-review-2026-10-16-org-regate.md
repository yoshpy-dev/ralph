# Self-review report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(承認済み。8 段目の 8a-3b)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)の中の 2 回目。1 回目(8e515ca7)の M1 と L1〜L9 の直しを確かめた回で、`.harness/state/standard-pipeline/cycle-count.json` は 1 のまま
- Scope: diff の品質だけ。`git diff 670ec219...HEAD`(HEAD 07b9a465)全体を見たうえで、1 回目のあとの 6 コミット(d6a006dc、94ff3493、d441b921、a3cd529c、c4b9479b、07b9a465。21 ファイル、+1045/-369)を重点に読んだ。テストの実行、静的解析、仕様適合、文書のずれの検査は行っていない(/test・/verify・/sync-docs の担当)

## Evidence reviewed

- 読んだコード(全体): `internal/org/regate_git.go`(210 行すべて)、`internal/org/regate_merge.go`(503 行すべて)、`internal/org/regate.go` の 6 コミット分の diff すべてと `Regate` の doc・`regateInterrupted`・`regateWords`・`regateCheckInProgress`、`internal/org/gate.go`・`lockfile.go`・`mergeorder.go`・`internal/cli/org_regate.go` の diff すべて、`gateGit`・`gateGitOutput`・`gateHead` の本体、`gateSignalContext`(`internal/cli/org_gate.go:126`)
- 読んだテスト(diff): `regate_merge_test.go`、`regate_test.go`、`org_regate_test.go` の 6 コミット分すべて。実行はしていない
- 読んだ文書: `/org`・`/director` skill の diff(単語単位)、`AGENTS.md`、plan の Progress の追記、`docs/tech-debt/README.md` の追加と書き換え 3 行。skill は 2 つとも 4 面を `cmp` で比べ、byte 一致
- 手で確かめたこと(scratchpad の使い捨ての repo で git を直接打った。repo の tree には何も書いていない): (1) `pre-merge-commit` hook が `kill -KILL $PPID` する merge は、終了コード 137、`MERGE_HEAD` なし、`A  h` が staged、HEAD は merge 前のまま。`git merge --abort` は「There is no merge to abort」で終わり、`git reset --keep <pre>` は `h` を untracked として files に残し、`git reset --merge <pre>` は `h` ごと消して clean に戻す(`regateMergeNotBack` の `moved=false` の主張どおり)。(2) `post-merge` hook が git を殺す merge は、merge commit ができ、`MERGE_HEAD` が残り(`git status` は「still merging」)、`git reset --keep <pre>` は「Cannot do a keep reset in the middle of a merge」で断られる。`git merge --abort` を先に打つと `MERGE_HEAD` が消え、HEAD は merge commit のままで、そのあと `git reset --keep <pre>` が通る。`regateMergeTip` は abort を `reset` の前に打つので、この順は `TestRegate_AMergeEndedAfterItsCommitNamesResetKeep` の流れと合う
- Gate の文面が変わっていないことの確認: `gateCheckBranch` の rebase の文(`"%s, and %s"` に `gateRebaseSteps` を入れた形が元の文と 1 字ずつ一致)、`gateCheckScript` の 4 文と `gateDirtyRefusal`(`afterCommit` が空なら元の文と一致)を 1 文ずつ見た。`TestGateWording_RefusalsThatAskForACommit` が Gate の 5 文を byte で固定している。`gateWording{` の literal は `gateWords` と `regateWords` の 2 つだけで、`rebaseSteps` が空の語り口は作られない
- 追加で grep: 追加行の debug 出力・TODO・`/Users/`・秘密の語・生成物(なし)、`regateRemoteTimeout`・`regateRemoteWaitDelay`・`regateNoIdentity` の旧名の残り(コードと文書にない。`regateNoIdentityEnv` はテストの helper で別物)、`regate.go` の import の使用(`bytes`・`io`・`fs`・`strings` はどれも使われている)、テスト名と関数名(登録簿の Trigger に書かれた 5 つのテストと `writeRegateMarker`・`readRegateMarker` が実在)、登録簿の追加行の列数(5 列ずつ)と取り消し線の形、`regateMergeTip`・`regateMerged`・`Regate` の行数。`gofmt -l` を 1 回、一覧表示だけで打った(出力なし、書き換えなし)。「整形ツールを走らせない」に触れるので、証拠には数えていない

## 1 回目の指摘の状態

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

### stderr の 1 行(d441b921)

`regateUncancelledGit`(`regate_merge.go:28-44`)の `context.AfterFunc` を読んだ。(1) data race はない。f が書く `stderr` は f の中だけで、f の `defer close(said)` と、`stop()` が false のときの `<-said` で、関数が返る前に f が終わる。`stop()` が true なら f は走らない。f が読む `regateGitTimeout` を書くテストの `t.Cleanup` は、`Regate` が返ったあとに走る。(2) 1 回の Regate で 1 行以下。ctx は一度 done になると戻らないので、行を書けるのは ctx が done でなかった最初の `regateUncancelledGit` だけで、その後の呼び出しは先頭の `ctx.Err() != nil`(`:29`)で別の道に入る。(3) 通常の終わりでは書かれない(`stop()` が true)。(4) 割り込みの後に始まる abort と undo は書かない(`:29`)。残る窓は、ctx が `:29` の確かめと `AfterFunc` の登録の間に done になる場合(行は書かれ、git は走るので内容は正しい)と、git が終わる直前に done になる場合(待ちの行が、待つ必要のない git について出る)で、どちらも害はない。`TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits` は ralph 自身にだけ SIGINT を送る(`signalSelf`)ので、端末の Ctrl-C の group 宛ての届き方は、`pgid` が ralph と違うという記録で代わりに確かめている

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | exception-handling | merge が成功したあと、`regateMergeTip` は HEAD と親を読む(`gateHead`、`regateParents`)。この 2 つが失敗すると、git の素の error をそのまま返し、merge commit ができたこと、記録も undo もされていないこと、戻し方を言わない。関数の doc(`:232-236`)と呼び出し側の注(`regateMerged` の `:144-145`)は、error が「worktree の状態と戻し方を言う」と読めるが、その約束が通るのは `regateMergeNotBack` を通る返り値だけ。この読み取りは ralph の group で、`context.WithoutCancel` の ctx で、上限なしで動く(`gateGit` は timeout を持たない)ので、M1 が merge・abort・reset について閉じた端末の Ctrl-C は、merge が終わった直後の数 ms の間は、この git を直接殺す(`gateGit` は signal で終わった git を `signal: interrupt` の error にする)。そうなると merge commit は記録も undo もなく残り、次の regate は `regateCheckOriginHead` で「HEAD has commits that origin's tip does not: push them」と言い、`regateResetHint` は gate の記録がないので何も足さない。つまり、gate を通っていない merge commit の push を勧める。窓は狭い。確かめたのはコードの読みで、窓を作る再現はしていない。同じ形の読み取り(`:279`、`:280`、`:302`、`:306`)は、失敗すると `regateMergeNotBack` を返すので、足りないのはこの 2 つだけ | `internal/org/regate_merge.go:250-257`、`:232-236`、`internal/org/regate.go` の `regateInterrupted`(ctx が done でなければ err をそのまま返す)、`internal/org/gate.go:745-762`(`gateGit`)、`internal/org/regate_git.go:77-107`(`regateCheckOriginHead`) | `:250-257` の 2 つの return を、`regateMergeNotBack(t, command, "made a merge commit", fmt.Sprintf("HEAD cannot be read (%v)", err), true)` の形にする(merge commit が作られたあとの読み取りなので `moved=true`、案内は `reset --keep <t.head>`)。doc の「error は…」の列挙に、この読み取りも入れる |
| N2 | LOW | readability | `regateMergeTip` が 99 行になった(1 回目の時点で 82 行、この直しで +17 行)。次の 3 つを 1 つの関数が持つ: merge を走らせて親を確かめる(`:242-265`)、失敗した merge の状態を読んで abort する(`:274-313`)、断りの文を組む(`:315-335`)。`:274-313` は `aborted` の文字列か error だけを返す閉じた塊で、切り出せる。同じ塊の中に、`switch conflicted := strings.FieldsFunc(files, func(r rune) bool { return r == 0 }); {` の `r` が、外の `r`(`gitResult`、`:242`)を隠している箇所がある(`:319`)。1 回目に `Regate` は 76 行、`regateMerged` は 47 行(今 60 行)で、リポジトリは長い関数を許していないわけではないが、この関数だけが 1 回の直しで伸び続けている | `internal/org/regate_merge.go:238-336`、`:319` | `:274-313` を `regateAbortLeftMerge(ctx, t, env, stderr, command, outcome) (aborted string, err error)` のような関数に切り出す。`:319` の closure の引数は `c rune` などに変える |
| N3 | LOW | maintainability | `regateCheckIdentity` だけが `regateBoundedGit` を `ownGroup=true` の裸の `true` で直に呼ぶ(`:204`)。ほかの `true` は `regateUncancelledGit` の中にあり、その doc が理由(merge・abort・undo を中途で止めない)を書いている。`git var` は hook を走らせず、端末も読まず、ctx で止めてよい git なので、この `true` の理由(端末の Ctrl-C が `git var` を signal で終わらせて、`regateGitSignaled` の error と ctx の error のどちらが返るかが競争になるのを避ける、が筋に見える)はコードにも doc にも plan にも書かれていない(plan は「`git var` を別のプロセスグループで動かす」とだけ書く)。次にこの行を触る人は、`ownGroup=false` に戻してよいかを判断できない | `internal/org/regate_merge.go:204`、`internal/org/regate_git.go:160-172`(`ownGroup` の doc)、`docs/plans/active/2026-10-16-org-regate.md:151` | `:204` の上に 1 行、理由を書く |
| N4 | LOW | maintainability | 登録簿の新しい行と RESOLVED の注が書く行数が、HEAD で合っていない。`regate.go is 713 lines`(`docs/tech-debt/README.md:277`、書いた 94ff3493 の時点では 713、HEAD では 728)と、`regate_test.go is 1495 lines`(`:279`、同じく 1495 から 1499)。どちらも書いた 1〜4 コミットあとの本 PR の中で崩れている。1 回目の L7(pointer に行番号を使わない)と同じ種類で、L7 の直しは `:299-303` に直しただけで行番号を残した | `docs/tech-debt/README.md:276`、`:277`、`:279`、`git show c4b9479b:internal/org/regate.go`(728 行)、`git show c4b9479b:internal/org/regate_test.go`(1499 行) | 行数は「約 700 行」「約 1500 行」にするか、数を外して「800 行を超える」だけにする。`:276` は行番号を外して関数名だけにする |

## Positive notes

- M1 の直しは、1 回目の推奨(merge・abort・reset を別の group に、fetch と push は ralph の group に残す)をそのまま実装し、`git var` まで同じ形に揃えた。signal で終わった git の 5 つの状態(merge の前で止まる、merge commit の前で止まって index に結果が残る、merge commit のあとで止まる、fetch、push)に、状態を言う error とテストがある。そのうち 2 つ(index に結果が残る、merge commit のあと)は上の(1)(2)の手の再現と、`reset --merge` と `reset --keep` の使い分けが一致した
- `regateMergeNotBack` の `moved` の分け方が正しい。HEAD が動いていないときは `reset --merge`(merge の途中も、index に残った結果も戻す)、動いたときは `reset --keep`(利用者の変更を落とさず止まる)。`reset --keep` が index に残った結果を files に残すことと、merge の途中では断られることを手で確かめた。doc の説明(`:338-350`)はそのとおり
- Gate の文面を変えずに共通の語り口を持たせた。`gateRebaseSteps` を const にし、`gateWords` に入れ、regate は別の文を持つ。文の 1 字ずつの一致をテストが固定している。`afterCommit` は commit を頼む 5 文すべてに入り、ほかに commit を頼む文は `gate.go` と `regate*.go` にない
- 割り込みの待ちの 1 行は、`AfterFunc` と `said` の組が小さく、race と二重書き込みの道がない(上の確認)。`RegateParams.Stderr` が nil のときの既定は `os.Stderr` で、CLI は `cmd.ErrOrStderr()` を渡す
- `regateGitTimeout`、`regateGitWaitDelay` と旧名(`regateRemoteTimeout` など)の置き換えは、コード・テスト・文書のどこにも旧名を残していない
- skill の `/org` と `/director` の 4 面が byte 一致し、割り込みと git var の文(`/org` skill の `regate` の行)がコードと合う。`<名前>` の決め方の追記は両方の skill にある
- 登録簿の解消した行が、同じ表の他の解消済みの行と同じ形で、新しい行も 5 列。Trigger に書かれた 5 つのテスト名と 2 つの関数名がすべて実在する

## Coverage gaps

- テストは実行していない。`pgid` の記録を使うテストが `ps -o pgid= -p $$` を使うので、実行環境に `ps` が要る(macOS と Linux の標準の `ps` で動く形)
- `git var`(`regate_merge.go:204`)の group は、hook も端末もない git なので、どのテストからも観測できない。`ownGroup=true` を `false` に変えてもテストは落ちないはず(読んだだけで、変異は走らせていない)。N3 の 1 行の理由が書かれれば、この箇所の意図は残る
- 端末の Ctrl-C を ralph の group に送る実機の確認はしていない(1 回目と同じ。process group が分かれていることをソースと `pgid` の記録で確かめた)
- N1 の窓(merge 直後の読み取りに Ctrl-C が当たる)は再現していない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(今回は足さない)。N1〜N4 は直すコストが小さい。cycle-count は 1 なので最終回ではなく、今は登録簿に書かず直す | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。N1〜N4 を直さずにパイプラインが進み、cycle 2(cap 2 の最後の回)に入るなら、cycle 2 のレビューで 1 行にまとめる。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM はなし。1 回目の M1 は直り、L1〜L9 は L7 が一部のほかすべて直った。新しい指摘は LOW 4 件で、merge 直後の読み取りの error が merge commit の存在を言わない、`regateMergeTip` の長さと名前の隠れ、`git var` の group の理由が書かれていない、登録簿の行数の古さ)
- Follow-ups:
  1. N1 は Go の小さな直し(2 つの return)。N2 と N3 は同じ commit に入れてよい。N4 は文書だけで、N1〜N3 の commit に添えるか `/sync-docs` で入れる
  2. N1〜N4 の直しは、`regate_merge.go` の返り値と doc の文面に触れるので、直すなら `/self-review` から回し直す(cycle-count は 1 のままで、この回の延長)
  3. この回は同じ cycle の確認なので、`docs/insights/events/2026-10-16-org-regate.jsonl` に `self_review` の行を足していない(phase と cycle が集計の鍵で、足すと二重に数える)
