# Verify report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 260c539299bc で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 のうち、merge のあとの取り直し。8 段目の 8a-3b)
- Verifier: verifier subagent (Claude)。パイプライン 2 回目(`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2)
- Scope: `git diff 670ec219...HEAD`(HEAD 5e12dfe0、41 コミット、42 ファイル、+8091/-389)。AC1〜AC13 と D1〜D9 を HEAD でもう一度確かめ、cycle 1 の cross-review のあとに変わった D2・D3・AC1 を重く見た。cycle 1 の verify(HEAD 0b6ba232)のあとのコードの変更は 85558f2c、f793924d、a59220f5、ca840a45、5e12dfe0 の 5 コミットで、ほかにテストだけの d00da42c・5e4ce6cd と、sync-docs の 0257bd93・413ab307 がある。テストは走らせていない(/test の受け持ち)。行番号は HEAD 5e12dfe0 のもの
- Evidence: `docs/evidence/verify-2026-10-16-org-regate.log` の「cycle 2」の節(cycle 1 の節のあとに足した。run-static-verify の出力、Go の検査、probe 3 と 4 のスクリプトと出力。`$PROBE3` は scratchpad の probe のディレクトリ、`~` は `$HOME`)。wrapper の全ログは `docs/evidence/verify-2026-10-16-190407.log`。どちらも gitignore の対象

## cycle 1 からの変化

- D2(f793924d、5e12dfe0): 読めない gate の記録は、その org が記録より前に完全な結びつきか読める記録を持ったすべてのブランチに付く(`collectMergeTargets`、`internal/org/mergeorder.go:390-461`、付ける先の集め方は `:404-409` と `:430-446`)。付く先が 2 つ以上なら、Detail が「may be a record of another branch of the same org, <org>, which has had <n> branches」と言う(`:938-943`)。cycle 1 の規則(最新の結びつきのブランチ、なければ最新の読める記録のブランチ)は消え、`gatedBranch` の残りはコードにない
- D3 と AC1(85558f2c、ca840a45、a59220f5): merge の前に `regateCheckInTheWay`(`internal/org/regate_merge.go:254-292`)が、merge が書くパスのうち HEAD が追跡していないもの(`regateMergeWrites`、`:314-361`)の上に何かあれば断る。呼び出しは `git var` の確かめのあと、merge の前(`regateMerged`、`:150`)
- cycle 1 の V1(`docs/quality/quality-gates.md` の 2 面の動詞の一覧)は 0257bd93 で直った。I1(`/org` skill の `reset --keep` の手の条件)も直り、skill は「そのあとに manifest の壊れた行がなく」と書く
- self-review cycle 2 の C2-1〜C2-5 は、ca840a45(C2-1 の merge-tree、C2-2 の空のディレクトリと HEAD が追跡するディレクトリ)と 5e12dfe0(C2-3〜C2-5)ですべて直った。self-review の報告は HEAD a59220f5 の時点のもので、5 件を開いたまま書く

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 断る 12 の場合(何も記録せず、worktree も origin も変えず、終了コード 1、理由と次の手を stderr) | 満たす(binary でも確認。V2-1 の過剰な断りを除く) | cycle 1 の 11 の場合はコードの形が変わらず、行番号だけが動いた: `--branch` なしと `--org-id` は `internal/cli/org_regate.go:119`・`:123`、結びつきなしと走っている org は `regateFirstCheck`(`internal/org/regate.go:442`)、worktree・ブランチ・detached・汚れた作業ツリー・スクリプトは `regateCheckWorktree`(`:531`)、rebase か merge の途中は `regateCheckInProgress`(`:609`)、ほかの regate は `:282` の `acquireHeldFileLock`。cycle 2 で足した 12 番目の「merge が書くパスに ignored のファイルかディレクトリがある」は下の D3。probe 3 の H1(tip が足した `local.env` に ignored のファイル)は終了コード 1、stdout は空、stderr は `local.env` を名指して「move them out of the worktree and run ralph org regate --branch feat/auth again」。HEAD、origin の 2 つの ref、作業ツリー、manifest の行数と sha256、`local.env` の中身はどれも前のまま。H2(directory rename の移った先 `pkg/new.gen`)も同じ。pre-fix の binary(413ab307)は H1 で merge して push し、`local.env` を main の中身に黙って書き換えた。テスト(読んだだけ): `TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay`(9 case)、`TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree` |
| AC2 fetch の失敗、origin にブランチがない、HEAD が `origin/<b>` と違う。fetch と push に `GIT_TERMINAL_PROMPT=0` | 満たす(cycle 1 の binary の確認のまま) | cycle 2 で変わっていない。`regateFetch`(`internal/org/regate_git.go:30`、ブランチがないときの文は `:45`)、`regateCheckOriginHead`(`:82`)、`regateRemoteEnv`(`:128`)と `regateFetchEnv`(`:136`) |
| AC3 何もしない道と、壊れた記録のあとの取り直し | 満たす(binary でも確認) | `passedOn`(`internal/org/regate.go:423`)、`RegateOutcomeUpToDate`(`:334`)。D2 の新しい規則で、org_id を別のブランチに結び直したあとの読めない記録も、前の成功を使わない。probe 3 の D2-3 は、`feat/auth` の regate の通った記録のあとに auth を `feat/billing` に結び、disband し、読めない `gate_failed` を足した台帳で、UP_TO_DATE にならず gate だけの道を走り、`via=regate` の `gate_passed` を書いた。pre-fix の binary は同じ台帳で UP_TO_DATE を出した。テスト: `TestRegate_GatesWhenNoRecordLetsItStop` に足した case(`internal/org/regate_test.go`) |
| AC4 取り込みの道 | 満たす(binary でも確認。V2-1 の入力を除く) | 流れは cycle 1 のまま(`regateMergeTip` は `internal/org/regate_merge.go:426`、記録の `Via: gateViaRegate` は `internal/org/regate.go:657`、push の refspec は `regate_merge.go:661` で `+` なし)。probe 3 の H1c(邪魔なファイルを退かしたあと)は `GATE: pass`・`MERGED:`・`HEAD:`・`BASE:`・`LOG:`・`PUSHED:` を出して終了コード 0、merge commit の件名は `Merge origin/main <12 桁> into feat/auth` で、親は merge の前の HEAD と取り込んだ先頭。case を区別しない APFS で、既定のブランチがファイルの名前の大文字小文字だけを変えたときは merge に届かない(V2-1) |
| AC5 merge commit を作れない | 満たす | 変わっていない(`regateCheckIdentity` は `regate_merge.go:216`、`regateAbortLeftMerge` は `:511`)。D3 の「共通の祖先がないときは git merge に断らせる」は probe 3 の H3 で確認した。origin の main を関係のない履歴に替えると、regate は確かめを飛ばし、`git merge` が「refusing to merge unrelated histories」(exit 128)で断り、regate は何も記録せずに終了コード 1。HEAD と manifest は前のまま |
| AC6 スクリプトの失敗、記録の前の中断、`head_moved` と `tree_changed` | 満たす(cycle 1 の確認のまま) | `regateMerged`(`regate_merge.go:141`)、`regateUndoMerge`(`:589`)、`regateMergeLeft`(`:632`)。cycle 2 は `regateMerged` の先頭に確かめを足しただけで、戻し方は変わっていない |
| AC7 push の失敗と記録のあとの中断、打ち直し | 満たす(cycle 1 の確認のまま) | `regatePush`(`:657`)、`regatePushInterrupted`(`:692`)、`regateHeadNotOnOrigin`(`:68`)、`regateResetHint`(`:98`) |
| AC8 スクリプトの間の org か結びつきの変化 | 満たす(テストのコードで確認) | `recordRegate`(`internal/org/regate.go:683`)。変わっていない |
| AC9 gate だけの道 | 満たす(binary でも確認) | cycle 1 では binary で単独には打っていなかった。probe 3 の D2-3 がこの道で、merge も push もせずに `GATE: pass`・`HEAD:`・`BASE:`・`LOG:` の 4 行を出し、HEAD と origin の ref は変わらず、manifest に 1 行足した。そのあとの merge-order(D2-4)は `feat/auth` を `ready` と出した |
| AC10 merge-order の理由の欄 | 満たす(binary でも確認) | `namesRegate`(`internal/org/mergeorder.go:348`)、`mergeRegateNext`(`:981`)、`mergeRegateFailedDetail`(`:999`)、`regate` の文(`:1047`)。D2 の Detail の追記は probe 3 の D2-2 で、`feat/auth` と `feat/billing` の両方が `unreadable` になり、どちらも「may be a record of another branch of the same org, auth, which has had 2 branches: run ralph org regate --branch <ブランチ> to record a new one」を出した。pre-fix の binary は `feat/auth` を `ready` のまま出した |
| AC13 Spawn との排他(D9) | 満たす(テストのコードで確認) | 変わっていない。`regateSpawnErr`(`internal/org/spawn.go:1327`)を spawn は `:1284`、start は `internal/org/feature.go:216` で呼ぶ。help は 413ab307 と 5e12dfe0 で spawn と start の断りを書き、spawn の help は `--dry-run` を除くと言う(`internal/cli/org.go:677-679`) |
| AC11 文書 | 一部(LOW のずれ 3 件) | 下の Documentation drift の表。AC11 が挙げる面はすべて regate を書くが、cycle 2 の D2 と D3 の変更が、仕様の注、`/org` skill の regate の行、tech-debt の 1 行に届いていない(V2-2〜V2-4) |
| AC12 テストの隔離、go1.26 と go1.25.8 | 一部(静的な部分を確認) | cycle 2 で足したテストの git の偽物は `t.TempDir()` の下の sh で(`regateWrapGit`、`internal/org/regate_merge_test.go:1512-1523`)、`t.Setenv` で PATH の先頭に置く。URL、`osascript`、本物の `herdr` の呼び出しはない。`go build` と `go vet` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で通った。テストの実行は /test |

D1〜D9:

- D1: 変わっていない(`org_regate.go:119`、`:126` の `orgLedgerMutating`)
- D2: 結びつきと記録は `regateFirstCheck` の `withManifestLockContext`(`regate.go:445`)の下で `readRegateBranch`(`:401`)が `collectMergeTargets` で読む。読めない記録を付ける先は上の「cycle 1 からの変化」のとおりで、計画の D2 の文(「その org がそれまでに完全な結びつきか読める記録を持ったすべてのブランチに付ける」「同じ org のほかのブランチのものかもしれないと言う」)と合う。付く先が 1 つの org では Detail に追記はない(`TestMergeOrder_UnreadableRecordAttachesToTheBindingBeforeIt` の doc がそう書く)。どの org も持たない記録は、今までどおり note で数える(`mergeorder.go:473-476`)
- D3: 計画の文と 1 つずつ突き合わせた。merge と同じ env(`regateMergeEnv`)で、`-c` の上書きなしに `git merge-tree --write-tree --no-messages HEAD <tip>` を打ち、終了コード 0 と 1(衝突)の 1 行目の木を `git diff --name-only -z --no-renames --diff-filter=A HEAD <tree>` で HEAD と比べる(`regate_merge.go:314-333`)。`-h` が `--write-tree` を名指さない git では、merge-base からの tip 側の diff と HEAD に対する追加の diff の積に落とす(`:334-361`)。probe 3 の H1b で、2.38 より前の git を真似た偽物(`--write-tree` と `-h` に usage と 129 を返す)を PATH に置くと、merge-tree を 2 回(`--write-tree`、`-h`)打ってから同じ文で断った。邪魔かどうかは `regatePathInTheWay`(`:377-405`): 最後の要素が非ディレクトリ、追跡していないものを含むディレクトリ(`ls-files --others --directory --no-empty-directory`)、上のディレクトリが HEAD の追跡していない非ディレクトリ。共通の祖先がなければ確かめない(`:255-260`)。断りの文は 10 件まで名指し、残りを数える(`:239`、`:283-286`)
- D4〜D9: cycle 1 から変わっていない。D6 の `via` は `log` より前(`internal/org/gate.go:172-175`)、D7 の `regateLockWait = time.Second`(`regate.go:93`)、D8 の `regateGitTimeout` は `:110`、D9 の印は取った直後に空にし(`:293`)、manifest のロックの下で書く(`:468`)

FR-7 と FR-9、8a-3a の持ち越しの判断は cycle 1 と同じで、cycle 2 の差分に `auto_merge` はない。

### 実装中に記録したずれ(cycle 2 の分)

| ずれ | 計画の文と合うか | Evidence |
| --- | --- | --- |
| P1 の直しを、2 つの diff の積から merge-tree の木に替えた(ca840a45) | 合う | 承認し直した D3(sha256:260c539299bc)の文と同じ手順。Progress の 161 行が経緯を書く |
| C2-2 の直し: 空のディレクトリと、HEAD が追跡するディレクトリをファイルに替える tip は断らない | 合う | D3 は「追跡していないものを含むディレクトリ」を邪魔とする。`TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay` の 3 case |
| 5e12dfe0 の名前と注の整理 | 合う(振る舞いは変わらない) | `orgBranches` の field を `orgBranchCount` にし、注と help の文を直しただけ。Progress にはこのコミットの注がない(情報) |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は changed、言語は golang。hook の shellcheck と `sh -n`、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、gofmt ok、golangci-lint `0 issues`、`secret-scan-branch: scanned 670ec219..5e12dfe0 against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `git diff --check 670ec219...HEAD` | pass | 出力なし |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | IDENTICAL 170、DRIFTED 0、KNOWN_DIFF 5 |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 260c539299bc |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V2-1 | MEDIUM | 振る舞い(AC4 の端の入力) | case を区別しないファイルシステム(macOS の APFS の既定)で、origin の既定のブランチが追跡するファイルの名前の大文字小文字だけを変えると(`readme.md` を `README.md` に)、regate は merge を断る。merge-tree の木で `README.md` は HEAD に対する追加なので候補になり、`os.Lstat("README.md")` は追跡している `readme.md` を見つけ、`regatePathInTheWay` は「HEAD が追跡していない非ディレクトリ」として返す。ディレクトリの名前の大文字小文字を変えたときも、中のファイルで同じことが起きる。断りの文は「HEAD does not track」「move them out」と言うが、それは追跡しているファイルで、言われたとおり退かすと作業ツリーが汚れて次も断られる。同じ tip を素の `git merge --no-ff` で取り込むと成功する。何も書かずに断る側に倒れるので、ファイルは失われない。人が worktree で merge して push すれば、regate は gate だけの道で通る | `internal/org/regate_merge.go:377-405`(`:387-388` の最後の要素の分岐)、probe 4(`docs/evidence/verify-2026-10-16-org-regate.log` の「probe 4」。regate は終了コード 1、manifest は 2 行のまま、素の git merge は rc 0) | `core.ignorecase` が真の worktree では、lstat が見つけたものが HEAD の追跡するパス(大文字小文字を区別せずに一致し、merge が消すか名前を変えるもの)なら邪魔としない。直さないなら、tech-debt に 1 行足し、`/org` skill の regate の行に回避の手順(worktree で merge して push し、regate を打ち直す)を書く |
| V2-2 | LOW | 文書(仕様) | 仕様の 7c-1 の注は「読めない記録は、打った org のその時点の結びつきのブランチに付ける(gate は結びつきのブランチでしか打てないため)」と書く。f793924d のあと、付く先はその org が持ったすべてのブランチで、理由(regate は古い結びつきのブランチにも記録を書く)も変わった。8a-3b の注(75 行)はこの変更と、ignored のファイルの断り(D3)を書いていない | `docs/specs/2026-10-07-org-multi-org-director.md:77`、`:75` | 8a-3b の注に D2 の付け方の変更と D3 の断りを足すか、7c-1 の注の該当の文に「8a-3b で、org が持ったすべてのブランチに付けるようにした」と添える(/sync-docs) |
| V2-3 | LOW | 文書(skill、README) | ignored のファイルの断りは CLI の help(a59220f5、`internal/cli/org_regate.go:73-76`)にはあるが、`/org` skill の 4 面の `regate` の行(`.claude/skills/org/SKILL.md:188`)と README の regate の段落(`README.md:264`)にない。どちらも merge が commit を作らない場合として、衝突、hook、名前がないことは並べている。self-review cycle 2 が /sync-docs に引き継いだもの | 上の各行。4 面の md5 は同じ `1220e53006c5909b893daf51faaf2c4c` | 4 面と README に 1 文足す |
| V2-4 | LOW | 文書(tech-debt) | 7c-1 の残りの行が、f793924d で変わったものを古い名前で指す。(1)「`unreadable` for a gate record that cannot be read and is attached to the branch of its org's latest readable gate record」は cycle 1 の規則の説明。(2) `TestMergeOrder_UnreadableRecordWithNoBindingAttachesToItsOrgsGatedBranch` は `...GatedBranches` に改名された(古い名前は新しい名前の前方一致なので grep では見つかるが、その名前の関数はない)。(3) (t) の「could update an org's `gatedBranch`」の `gatedBranch` は消え、今は `orgBranches` | `docs/tech-debt/README.md:239` | 3 か所を今の規則と名前に直す(/sync-docs) |

情報(ずれに数えない):

- I1 2.38 より前の git の落とし先は、directory rename の移った先を見ない(D3 と `regateMergeWrites` の doc がそう書く)。probe 3 の H2b で、偽の古い git では `pkg/new.gen` の ignored のファイルの前で断らず、merge が衝突して `git merge --abort` で戻ったあと、そのファイルはなくなっていた。計画どおりの割り切りだが、tech-debt に行がない。足すなら「git 2.38 より前、directory rename、その中への追加、移った先の ignored のファイル」の 4 つが重なる場合として書く
- I2 `git merge-tree --write-tree` は repository に到達できない木と blob のオブジェクトを書く。ref も作業ツリーも変えないので、AC1 の「worktree も origin も変えず」には当たらない
- I3 `/org` skill は regate の間に断られるものを「`ralph org spawn` と `ralph org start --plan`」と書き、start の help は「Both forms」と書く。素の `ralph org start` も leader の Spawn を通るので断られ、skill の「spawn」に含まれると読める。計画の D9 の文も skill と同じ書き方
- I4 計画の Progress は 5e12dfe0 の注を持たず、「Review artifact created」の行の「1 周目の 2 回目」は、self-review の報告の先頭が cycle 2 になった今は付録 A を指す
- I5 self-review の報告(HEAD a59220f5)は C2-1〜C2-5 を開いたまま書く。上の「cycle 1 からの変化」のとおり、どれも ca840a45 と 5e12dfe0 で直った。self-review の報告は記録の写しなので、直さない

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | 一部(V2-3) | 4 面の md5 は同じ。cycle 1 の I1(`reset --keep` の手の条件)は直った。merge-order の分類、「merge してよい順」の手順、見張りなしで使える動詞の一覧は今のコードに合う。regate の行に ignored のファイルの断りがない |
| `/director` skill の 4 面 | yes | md5 は `2aa01e8014c06fab49ff8b58b5688d34` で同じ。「merge のあとの取り直し」の手順 3 は「断りの理由と次の手をそのまま人に伝える」と書き、D3 の断りもそこに入る |
| CLI の help(`ralph org regate`、`ralph org`、`spawn`、`start`、`merge-order`) | yes | regate の help は D3 の断りを書く(`internal/cli/org_regate.go:73-76`)。spawn と start の help は regate の間の断りを書く(`internal/cli/org.go:677-679`、`:867-869`)。merge-order の help の `unreadable` の説明は付け方を言わないので、D2 の変更とぶつからない |
| `AGENTS.md` の repo-map | yes | `regate.go`・`regate_merge.go`・`regate_git.go` と `regate.lock` の説明は今のコードに合う |
| `README.md`(264 行) | 一部(V2-3) | merge-order の分類と regate の説明は合う。regate の merge が commit を作らない場合に ignored のファイルの断りがない |
| 仕様(75 行と 77 行) | 一部(V2-2) | 7c-1 の注の付け方の文が古い |
| `docs/tech-debt/README.md` | 一部(V2-4) | 8a-3b の行(232、233、275〜280 行)は今のコードに合う。7c-1 の残りの行(239 行)に古い規則と名前が 3 か所 |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md` | yes | cycle 1 の V1 は直った。Cross-org watch gate の行に `regate`、Quality pipeline gate の行に regate の説明がある。2 面の差は check-sync の KNOWN_DIFF のもので、regate の文は同じ |
| `internal/cli/org_legacy_ledger.go` | yes | 台帳を書き換える動詞の doc comment に `regate`。`TestOrgRegate_LegacyLedgerGuard`(d00da42c)がある |
| 計画の AC と Progress | ほぼ(情報 I4) | AC1〜AC13 は `[x]`。Progress は 5e12dfe0 の注がない |

## Observational checks

HEAD の binary(`go build -o $PROBE3/bin/ralph ./cmd/ralph`)と、cross-review の時点の 413ab307 から `git archive` で作った pre-fix の binary を、Homebrew の git 2.49.0 の symlink と `/usr/bin`・`/bin` だけの PATH で打った。gh は PATH にない。HOME と `GIT_CONFIG_GLOBAL` は probe の中の偽の場所で、origin は probe の中の bare repository、台帳は scratch の main checkout の `.harness/state/org`。gate のスクリプトは `echo` と exit 0 だけの stub。ネットワーク、herdr、本物の GitHub は打っていない。

- probe 3(HEAD の binary): H1 tip が足したパスの ignored のファイルで断る、H1b 2.38 より前の git の落とし先でも断る、H1c 退かしたあとの merge・gate・push、D2-1〜D2-4 読めない記録が org の 2 つのブランチに付き、regate が gate だけの道で取り直す、H2 directory rename の移った先で断る、H2b 落とし先は H2 を見落とす(I1)、H3 関係のない履歴は git merge が断る
- probe 3(pre-fix の binary): H1 は `local.env` を黙って書き換えて push し、D2 の台帳で `feat/auth` は `ready` のまま、regate は UP_TO_DATE。cycle 2 の直しがこの 2 つを変えたことの対照
- probe 4(HEAD の binary): case を区別しない APFS で、main が `readme.md` を `README.md` に変えると regate は断る(V2-1)。同じ tip を素の git merge は取り込む

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/` は /test で確かめる
- cycle 1 の Coverage gaps はそのまま残る: 中断の道、`head_moved` と `tree_changed`、hook が git を殺す場合、fetch と push の env はテストを読んだだけ。AC13 の spawn と `start --plan` は binary で打っていない。端末の Ctrl-C の実機の確認、ssh の origin、本物の GitHub への push、Linux
- D3 の確かめのうち、`lstat` が読めないパス、merge-base が 128 で落ちる場合、merge-tree の 120 秒の上限は、コードを読んだだけ(probe も、読んだ範囲のテストも、この 3 つを打たない)
- 本物の 2.38 より前の git は打っていない。偽物は `--write-tree` と `-h` に usage を返すだけで、ほかの subcommand は 2.49.0 が答える
- V2-1 は `README.md` の 1 ファイルの名前の変更でだけ打った。ディレクトリの名前の変更は、コードを読んで同じ道を通ると判断した

## Verdict

- Verdict: partial-pass(AC1〜AC13 は計画が書く道で満たし、D2・D3・AC1 の cycle 2 の直しは binary と pre-fix の対照で確かめた。V2-1 は MEDIUM で、case を区別しない APFS で既定のブランチがファイルの名前の大文字小文字だけを変えたとき、AC4 の merge に届かず、事実と違う断りを出す。何も書かずに断るので失われるものはなく、回避の手もある。V2-2〜V2-4 は LOW の文書のずれで、/sync-docs で直せる)
- Verified: AC1〜AC5・AC9・AC10 の本体(binary とローカルの bare origin)。D2 の新しい付け方と Detail、D3 の merge-tree の道と 2.38 より前の落とし先、関係のない履歴を git merge に任せること、退かしたあとの取り直し。pre-fix の binary との対照 2 件。self-review cycle 2 の C2-1〜C2-5 がどれも直ったこと。cycle 1 の V1 と I1 が直ったこと。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、`git diff --check`、skill の 8 面の一致、check-sync、plan の digest)
- Partially verified: AC6〜AC8 と AC13 のうちテストだけが固定する場合、AC11(LOW のずれ 3 件)、AC12 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、本物の古い git、ディレクトリの大文字小文字の変更、端末の Ctrl-C の実機、ssh と GitHub の origin、Linux
- Insight event: `insights-append.sh` の verdict に partial-pass がないので、これまでの partial-pass の verify と同じく `pass` で、`--cycle 2 --medium 1 --low 3` を付けて記録した
