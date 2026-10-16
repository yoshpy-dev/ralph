# Verify report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 260c539299bc で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 のうち、merge のあとの取り直し。8 段目の 8a-3b)
- Verifier: verifier subagent (Claude)。パイプライン 2 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2)の 2 度目の verify。1 度目は HEAD 5e12dfe0 で partial-pass(V2-1 MEDIUM と LOW 3 件)
- Scope: `git diff 670ec219...HEAD`(HEAD c621ccc9、43 コミット、42 ファイル、+8213/-389)。1 度目のあとの差分(`git diff 5e12dfe0 HEAD`)は 06db67ca(1 度目の verify の報告と insight event)と c621ccc9(`internal/org/regate_merge.go` と `internal/org/regate_merge_test.go` だけ)で、V2-1 の直しと、それが触る AC1・AC4・D3 を確かめ直した。ほかの結果は、差分が触らないので 1 度目のものを引き継ぐ。テストは走らせていない(/test の受け持ち)。行番号は HEAD c621ccc9 のもの
- Evidence: `docs/evidence/verify-2026-10-16-org-regate.log` の「cycle 2, second pass」の節(1 度目の節のあとに足した。run-static-verify の出力、Go の検査、probe 5 と 6 のスクリプトと出力、probe 3 の打ち直し。`$PROBE3` と `$PROBE5` は scratchpad の probe のディレクトリ)。wrapper の全ログは `docs/evidence/verify-2026-10-16-195222.log`。どちらも gitignore の対象

## 1 度目からの変化

- V2-1(c621ccc9): `regatePathInTheWay`(`internal/org/regate_merge.go:385-417`)は、lstat が見つけたものを、ディスク上の綴り(`regateSpellingOnDisk`、`:427-444`)で判じる。最後の要素の非ディレクトリが merge の書くパスと違う綴りで、HEAD がその綴りで追跡していれば邪魔としない(`:408-412`)。追跡していなければ、ディスク上の綴りで名指して断る。ディレクトリの中の追跡していないものも、上の非ディレクトリが追跡されているかも、ディスク上の綴りで git に聞く(`:402-412`)。断りの文はディスク上の綴りを出す(doc comment は `:241-254`)
- テストは、断る側に 3 case(12 case に)、merge する側に 3 case(6 case に)を足した。足した 6 case は、case を区別するファイルシステムで飛ばす(`regateSkipOnACaseSensitiveFS`、`internal/org/regate_merge_test.go:1239`)
- 直しは、1 度目に見落とした逆向きの穴も塞いだ。pre-fix の binary(5e12dfe0)は、worktree に ignored のディレクトリ `cache/`(中に `cache/inner`)があり、既定のブランチがファイル `CACHE` を足すと、確かめを通って merge し、`cache/inner` を消して push した(probe 5 の C4)。`ls-files --others -- CACHE` が大文字小文字を区別して何も返さなかったため。HEAD の binary は `cache` を名指して断る

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 断る 12 の場合(何も記録せず、worktree も origin も変えず、終了コード 1、理由と次の手を stderr) | 満たす(binary でも確認) | 1 から 11 番目の場合はコードが変わっていない(`internal/cli/org_regate.go:119`・`:123`、`regateFirstCheck` は `internal/org/regate.go:442`、`regateCheckWorktree` は `:531`、`regateCheckInProgress` は `:609`、ほかの regate は `:282`)。12 番目(D3)は HEAD の binary で打ち直した。probe 3 の H1(`local.env`)と H1b(2.38 より前の git の落とし先)と H2(directory rename の移った先 `pkg/new.gen`)は 1 度目と同じく終了コード 1 で、HEAD、origin の 2 つの ref、作業ツリー、manifest の行数と sha256、ファイルの中身は前のまま。probe 5 の C3(ignored の `notes.txt`、tip が `NOTES.txt` を足す)は終了コード 1、stderr は `notes.txt`(ディスク上の綴り)を名指し、`notes.txt` の中身も snapshot も前のまま。C4(ignored の `cache/inner`、tip がファイル `CACHE` を足す)も `cache` を名指して断り、何も変えない。pre-fix の binary は C3 を `NOTES.txt` と名指して断り、C4 は merge して `cache/inner` を消した。テスト(読んだだけ): `TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay`(`internal/org/regate_merge_test.go:1290`、12 case) |
| AC2 fetch の失敗、origin にブランチがない、HEAD が `origin/<b>` と違う。fetch と push に `GIT_TERMINAL_PROMPT=0` | 満たす(cycle 1 の binary の確認のまま) | 差分が触らない。`regateFetch`(`internal/org/regate_git.go:30`、`:45`)、`regateCheckOriginHead`(`:82`)、`regateRemoteEnv`(`:128`)、`regateFetchEnv`(`:136`) |
| AC3 何もしない道と、壊れた記録のあとの取り直し | 満たす(binary でも確認) | 差分が触らない。probe 3 の打ち直しの D2-3 も、1 度目と同じく UP_TO_DATE にならず gate だけの道を走った。`passedOn`(`internal/org/regate.go:423`)、`RegateOutcomeUpToDate`(`:334`) |
| AC4 取り込みの道 | 満たす(binary でも確認。V2-1 の入力も通る) | probe 5 の C1(既定のブランチが `readme.md` を `README.md` に)と C2(`Lib/` を `lib/` に)は、どちらも `GATE: pass`・`MERGED:`・`HEAD:`・`BASE:`・`LOG:`・`PUSHED:` を出して終了コード 0。merge commit の件名は `Merge origin/main <12 桁> into feat/auth`、親は merge の前の HEAD と取り込んだ先頭。manifest の最後の行は `gate_passed`(head は merge commit、base は先頭、`base_ref=refs/remotes/origin/main`、`branch=feat/auth`、`via=regate`)。push のあと `origin/feat/auth` は HEAD と同じ。作業ツリーはきれいで、ディスク上の名前は `README.md` と `lib/` になった。C1 のあとの merge-order は `feat/auth` を `ready` と出した。pre-fix の binary は C1 と C2 を断った(C2 は `lib/a.txt, lib/b.txt` を名指した)。probe 3 の H1c(退かしたあと)も 1 度目と同じく merge して push した。流れのコードは変わっていない(`regateMergeTip` は `internal/org/regate_merge.go:465`、記録の `Via: gateViaRegate` は `internal/org/regate.go:657`、push の refspec は `regate_merge.go:700` で `+` なし)。テスト: `TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay`(`:1501`、6 case) |
| AC5 merge commit を作れない | 満たす | コードは変わっていない(`regateCheckIdentity` は `regate_merge.go:216`、`regateAbortLeftMerge` は `:550`)。probe 3 の打ち直しの H3(関係のない履歴)は、git merge が exit 128 で断り、regate は何も記録せずに終了コード 1 |
| AC6 スクリプトの失敗、記録の前の中断、`head_moved` と `tree_changed` | 満たす(cycle 1 の確認のまま) | 差分が触らない。`regateMerged`(`regate_merge.go:141`)、`regateUndoMerge`(`:628`)、`regateMergeLeft`(`:671`) |
| AC7 push の失敗と記録のあとの中断、打ち直し | 満たす(cycle 1 の確認のまま) | 差分が触らない。`regatePush`(`:696`)、`regatePushInterrupted`(`:731`)、`regateHeadNotOnOrigin`(`:68`)、`regateResetHint`(`:98`) |
| AC8 スクリプトの間の org か結びつきの変化 | 満たす(テストのコードで確認) | 差分が触らない。`recordRegate`(`internal/org/regate.go:683`) |
| AC9 gate だけの道 | 満たす(binary でも確認) | probe 3 の打ち直しの D2-3 は、merge も push もせずに gate を走らせ、HEAD と origin の ref は変わらず、manifest に 1 行足した。そのあとの merge-order は `feat/auth` を `ready` と出した |
| AC10 merge-order の理由の欄 | 満たす(binary でも確認) | 差分が触らない。probe 3 の打ち直しの D2-2 は、1 度目と同じく 2 つのブランチを `unreadable` と出し、「may be a record of another branch of the same org, auth, which has had 2 branches」を添えた |
| AC13 Spawn との排他(D9) | 満たす(テストのコードで確認) | 差分が触らない。`regateSpawnErr`(`internal/org/spawn.go:1327`)、spawn は `:1284`、start は `internal/org/feature.go:216` |
| AC11 文書 | 一部(LOW のずれ 3 件、1 度目から持ち越し) | 下の V2-2〜V2-4。c621ccc9 は文書を変えていない。V2-1 の直しは、仕様、skill、README、help の書く範囲(「ignored のファイルかディレクトリがあれば断る」)を変えないので、新しいずれは生まない |
| AC12 テストの隔離、go1.26 と go1.25.8 | 一部(静的な部分を確認) | 足した 6 case は `t.TempDir()` の下の fixture だけを使い、URL、`osascript`、本物の `herdr` の呼び出しはない。git の偽物は `regateWrapGit`(`internal/org/regate_merge_test.go:1599-1611`)のまま。`go build` と `go vet` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で通った。テストの実行は /test |

D1〜D9:

- D1、D2、D4〜D9: 差分が触らない。1 度目の結果のまま(D2 は probe 3 の打ち直しでも同じ出力)
- D3: 手順は 1 度目に計画の文と突き合わせたまま変わっていない。merge-tree の木から候補を取る道は `regateMergeWrites`(`regate_merge.go:315-362`、2.38 より前の git の落とし先は `:334-362`)で、probe 3 の打ち直しの H1b は、偽の古い git で merge-tree を 2 回(`--write-tree`、`-h`)打ってから断った。共通の祖先がなければ確かめない(`:260`)。断りの文は 10 件まで名指す(`:239`、`:284-287`)。c621ccc9 が変えたのは、邪魔かどうかの判じ方(`regatePathInTheWay`)だけ。計画の D3 は「そのパスに非ディレクトリがあるか、追跡していないものを含むディレクトリがあるか、その上のディレクトリが HEAD の追跡していない非ディレクトリなら断る」と書き、ファイルシステムが大文字小文字を区別しない場合を書かない。直しは、HEAD が別の綴りで追跡するものを「そのパスの非ディレクトリ」から外した。AC1 の 12 番目の文(「ignored のファイルかディレクトリがある」)により近づく向きで、計画の文とぶつからない。計画の Progress はこの直しを書いていない(情報 I4)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は changed、言語は golang。hook の shellcheck と `sh -n`、`check-sync.sh`(IDENTICAL 170、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、gofmt ok、golangci-lint `0 issues.`、`secret-scan-branch: scanned 670ec219..c621ccc9 against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `git diff --check 670ec219...HEAD` | pass | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 260c539299bc |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V2-1 | MEDIUM | 振る舞い(AC4 の端の入力) | 直った(c621ccc9)。case を区別しない APFS で、既定のブランチが追跡するファイルかディレクトリの名前の大文字小文字だけを変えても、regate は merge して push する。ignored のファイルが別の綴りで邪魔をするときは、ディスク上の綴りで名指して断る | probe 5 の C1〜C4(HEAD と pre-fix の両方の binary)。`internal/org/regate_merge.go:394-414`、`:427-444` | なし |
| V2-2 | LOW | 文書(仕様) | 1 度目から持ち越し。仕様の 7c-1 の注は「読めない記録は、打った org のその時点の結びつきのブランチに付ける」と書くが、f793924d のあと、付く先はその org が持ったすべてのブランチ。8a-3b の注はこの変更と、ignored のファイルの断り(D3)を書いていない | `docs/specs/2026-10-07-org-multi-org-director.md:77`、`:75` | 8a-3b の注に D2 の付け方の変更と D3 の断りを足すか、7c-1 の注に「8a-3b で、org が持ったすべてのブランチに付けるようにした」と添える(/sync-docs) |
| V2-3 | LOW | 文書(skill、README) | 1 度目から持ち越し。ignored のファイルの断りは CLI の help(`internal/cli/org_regate.go:73-76`)にあるが、`/org` skill の 4 面の `regate` の行(`.claude/skills/org/SKILL.md:188`)と README の regate の段落(`README.md:264`)にない | 上の各行 | 4 面と README に 1 文足す(/sync-docs) |
| V2-4 | LOW | 文書(tech-debt) | 1 度目から持ち越し。7c-1 の残りの行が、f793924d で変わったものを古い規則と名前で指す(cycle 1 の付け方の説明、`...GatedBranch` という今はない関数名、消えた `gatedBranch`) | `docs/tech-debt/README.md:239` | 3 か所を今の規則と名前に直す(/sync-docs) |
| V2-5 | LOW | 振る舞い(V2-1 の残りの端の入力) | HEAD が `readme.md` を追跡し、ディスク上のファイルが git の外で `README.md` に名前を変えられている worktree(core.ignorecase で `git status` はきれい)で、既定のブランチが `readme.md` を `README.md` に変えると、regate は「`README.md` は HEAD が追跡していない」と言って断る。ディスク上の綴りが merge の書くパスと同じなので、`:408` の「綴りが違えば追跡を聞く」分岐に入らず、`:414` で邪魔として返る。同じ worktree で素の `git merge --no-ff` は成功し、作業ツリーはきれいに終わる。何も書かずに断る側に倒れるので、失われるものはない。ディスク上の綴りが index とずれるのは、Finder などで git の外から大文字小文字だけを変えたときで、V2-1 より起こりにくい | probe 5 の C5(HEAD の binary は終了コード 1、snapshot は前のまま。素の git merge は rc 0)。`internal/org/regate_merge.go:408-414` | 直すなら、綴りが同じときも、最後の要素の非ディレクトリについて `git ls-files -z --` の出力を case を区別せずに見て、HEAD が追跡していれば邪魔としない。直さないなら tech-debt に 1 行足す。回避は `git mv` か、ディスク上の名前を index の綴りに戻して打ち直すこと |

情報(ずれに数えない):

- I1 2.38 より前の git の落とし先は、directory rename の移った先を見ない(1 度目のまま)。probe 3 の打ち直しの H2b も同じ振る舞い(merge が衝突して `git merge --abort` で戻り、`pkg/new.gen` はなくなった)。tech-debt に行がない
- I2 `git merge-tree --write-tree` は到達できないオブジェクトを書く(1 度目のまま)
- I3 `/org` skill と start の help の「spawn と start」の書き方(1 度目のまま)
- I4 計画の Progress は 5e12dfe0 と c621ccc9 の注を持たない。「Review artifact created」の行の「1 周目の 2 回目」は、self-review の報告の付録 A を指す
- I5 self-review の報告(HEAD a59220f5)は C2-1〜C2-5 を開いたまま書く(1 度目のまま。報告は記録の写しなので直さない)
- I6 c621ccc9 で、別の綴りで見つかったファイルごとに `git ls-files` を 1 回打つようになった。既定のブランチが N 個のファイルを持つディレクトリの名前の大文字小文字だけを変えると、確かめは N 回 git を打つ。probe 6 の計測で、N=200 は regate 全体で 4 秒、N=2000 は 32 秒(素の clone と merge は 1 秒以下)。結果は正しく(どちらも merge して push した)、git の 1 回ずつに `regateGitTimeout` が掛かり、全体の上限はない。その間 regate のロックを持つので、その org の spawn と start は断られる。急ぐなら、別の綴りの候補をまとめて 1 回の `ls-files -z` で聞ける

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | 一部(V2-3) | 1 度目のまま。c621ccc9 は skill を変えていない |
| `/director` skill の 4 面 | yes | 1 度目のまま |
| CLI の help(`ralph org regate`、`ralph org`、`spawn`、`start`、`merge-order`) | yes | regate の help の D3 の断り(`internal/cli/org_regate.go:73-76`)は、ignored のファイルかディレクトリを断ると書き、綴りの規則を書かない。直しの振る舞いとぶつからない |
| `AGENTS.md` の repo-map | yes | 1 度目のまま |
| `README.md`(264 行) | 一部(V2-3) | 1 度目のまま |
| 仕様(75 行と 77 行) | 一部(V2-2) | 1 度目のまま |
| `docs/tech-debt/README.md` | 一部(V2-4) | 1 度目のまま。V2-1 は直ったので、1 度目の勧めにあった tech-debt の行は要らない |
| `docs/quality/quality-gates.md` の 2 面 | yes | 1 度目のまま |
| `internal/cli/org_legacy_ledger.go` | yes | 1 度目のまま |
| 計画の AC と Progress | ほぼ(情報 I4) | AC1〜AC13 は `[x]`。digest は 260c539299bc のまま |

## Observational checks

HEAD c621ccc9 の binary(`go build ./cmd/ralph`)と、1 度目の verify の時点の 5e12dfe0 から `git archive` で作った pre-fix の binary を、Homebrew の git 2.49.0 の symlink と `/usr/bin`・`/bin` だけの PATH で打った。gh は PATH にない。HOME と `GIT_CONFIG_GLOBAL` は probe の中の偽の場所で、origin は probe の中の bare repository、台帳は scratch の main checkout の `.harness/state/org`。gate のスクリプトは `echo` と exit 0 だけの stub。probe のディレクトリは case を区別しない APFS の上(probe の最初の行で確かめた。git は `core.ignorecase=true` を付けた)。ネットワーク、herdr、本物の GitHub は打っていない。

- probe 5(HEAD の binary): C1 ファイルの名前の大文字小文字の変更は merge して push、merge-order は `ready`。C2 ディレクトリの名前の大文字小文字の変更も merge して push。C3 ignored の `notes.txt` と tip の `NOTES.txt` は `notes.txt` を名指して断る。C4 ignored の `cache/` と tip のファイル `CACHE` は `cache` を名指して断る。C5 ディスク上の綴りが index とずれた追跡しているファイルは断る(V2-5)
- probe 5(pre-fix の binary): C1 と C2 は断り(V2-1)、C3 は `NOTES.txt` と名指して断り、C4 は merge して `cache/inner` を消して push した。c621ccc9 が C1、C2、C4 の結果を変えたことの対照
- probe 3 の打ち直し(HEAD の binary): H1、H1b、H1c、D2-1〜D2-4、H2、H2b、H3 のどれも 1 度目と同じ結果
- probe 6(HEAD の binary): ディレクトリの名前の大文字小文字の変更で、ファイルの数と regate の時間(I6)

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/` は /test で確かめる
- 足した 6 case は case を区別しないファイルシステムでだけ走る。CI(`.github/workflows/verify.yml` は `ubuntu-latest`)では飛ぶので、綴りの規則を固定するのは macOS の手元の実行だけ
- `regateSpellingOnDisk` の「一覧を読めないディレクトリ」の誤りの道と、Unicode の正規化の違いで見つかる名前(doc comment が「そのまま残す」と書くもの)は、コードを読んだだけ
- 1 度目の Coverage gaps はそのまま残る: 中断の道、`head_moved` と `tree_changed`、hook が git を殺す場合、fetch と push の env はテストを読んだだけ。AC13 の spawn と `start --plan` は binary で打っていない。端末の Ctrl-C の実機、ssh の origin、本物の GitHub への push、Linux、本物の 2.38 より前の git。D3 の確かめのうち、`lstat` が読めないパス、merge-base が 128 で落ちる場合、merge-tree の 120 秒の上限

## Verdict

- Verdict: pass(1 度目の V2-1 MEDIUM は c621ccc9 で直り、HEAD の binary で確かめた。残りは LOW 4 件で、V2-2〜V2-4 は /sync-docs で直す文書のずれ、V2-5 は git の外で綴りを変えた worktree でだけ起き、何も書かずに断る側に倒れる端の入力。MEDIUM 以上はない)
- Verified: V2-1 の直し(ファイルとディレクトリの名前の大文字小文字の変更で merge して push すること、別の綴りの ignored のファイルとディレクトリで断ること)と、pre-fix の binary との対照 4 件(そのうち 1 件は 1 度目に見落とした、ignored のディレクトリを消す穴)。AC1 の 12 番目の場合と AC4 の取り込みの道を HEAD の binary でもう一度。D3 の merge-tree の道と 2.38 より前の落とし先、関係のない履歴を git merge に任せること。D2 と AC3・AC9・AC10 を probe 3 の打ち直しで。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、`git diff --check`、check-sync、skill の lock-step、plan の digest)
- Partially verified: AC6〜AC8 と AC13 のうちテストだけが固定する場合、AC11(LOW のずれ 3 件)、AC12 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、case を区別するファイルシステムでの binary の打ち直し、本物の古い git、端末の Ctrl-C の実機、ssh と GitHub の origin、Linux
- Insight event: `insights-append.sh` で `--verdict pass --cycle 2 --medium 0 --low 4` を記録した
