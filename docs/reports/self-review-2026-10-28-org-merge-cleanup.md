# Self-review report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(D1〜D6、AC1〜AC6 と AC3b、Progress checklist の S1〜S5 と「self-review cycle 1」の F1〜F3 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1。run 1 は実装の review、run 2 は run 1 の所見の直しの review、run 3 は run 2 の所見の直しの review)
- Scope:
  - run 3: `git diff fb214c0d..afe9c1ed`(HEAD afe9c1ed、23 ファイル、+460/-146)。コミットは 5ddf5532(F3: R2-L1〜R2-L8、21 ファイル)と afe9c1ed(計画の D2・D3 と仕様 FR-8 の注の文)。run 2 の所見 R2-L1〜R2-L8 の直しが所見どおりか、直しが新しい問題を持ち込んでいないかを見た。
  - run 2: `git diff 97ea508b..HEAD`(HEAD e316d081、22 ファイル、+666/-239)。コミットは 5442c872(計画の D2・D3・AC3 と台帳の 2 行)、a3af6114(F1: M2・L1・L2)、cbaa92f4(F2: M1・L3・L4 と skill・help・README・仕様の文)、794aee4d(F2b: CLI の文の中のブランチの quote)、e316d081(計画の記録)。
  - run 1: `git diff 15e301c8...c55819e9`(44 ファイル、+4684/-263)。
  - どれも、命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。テストと静的解析は打っていない(run 3 は、別のセッションが local CI を流しているため、`go test`・`go build`・`go vet`・`golangci-lint`・`run-*.sh` も打たず、読むことと `git`・`cmp`・scratchpad の小さな git の実験だけで確かめた)。

この report の verdict(下の「Findings」の先頭の表と「Recommendation」の `Merge:` の行)は HEAD afe9c1ed のブランチについてのもの。run 1・run 2 の所見は、その下に当時の文のまま残し、扱いを別の表にした。run 2 の所見の ID は run 1 の L1〜L6 と取り違えないよう `R2-L<n>`、run 3 は `R3-L<n>` にした。

パイプラインの cycle 2(cross-review cycle 1 の指摘の直し、HEAD 33b78f8c)の review は、末尾の「cycle 2(HEAD 33b78f8c)」の節にある。いまのブランチの verdict はその節の `Merge:` の行で、上の verdict は cycle 1 の記録として残している。cycle 2 の所見の ID は `C2-L<n>`。

## Evidence reviewed

### run 3(HEAD afe9c1ed)

- R2-L1。`checkRecord` の最後の断り(`internal/org/cleanup.go:513-516`)は、記録のブランチが binding と同じでパスか kind だけが違うとき、`git log <head>..<branch>` と `git -C <記録のパス> status` を、`--force-branch` の Manual の前に確かめるよう言う。記録のパスは `cleanupGitCmd` を通り、空白を含むパスは単引用符で囲まれる(`TestCleanupFeatureWorktree_RecordOfAnotherPathOrKindNamesTheChecks` の `'/elsewhere/org auth'`)。`scriptCmd` の注釈(`:330-333`)は「binding のブランチの記録」に直った。記録の `worktree_path` が空の記録なら `git -C '' status` になり、git は空の `-C` を無視して打った場所の status を出す。記録は script が絶対パスで書くので、手で壊した記録に限る話で、所見にしていない。
- R2-L2。確かめ 5 の文(`branchMovedSteps`、`:640-651`)は、退避を「main worktree から作る別のブランチ」にし、`git -C <root> branch <branch>-unmerged <branch>` を名指して、feature worktree でブランチを切り替えないよう言う。退避を feature worktree の外で作るので、worktree がそのブランチにいても別のブランチにいても、続く戻し方(`reset --hard` か `branch -f`、登録が残るなら `update-ref`)が動かすのは `<branch>` で、退避のブランチではない。この退避の名前は固定で、すでにあるときの扱いは R3-L1。
- R2-L3。`checkBranchTip`(`:617-638`)は、ブランチの先頭が head と違うとき、`git cat-file -e <head>^{commit}` を `c.root` で打つ。`cleanupGit` は git が走れなかった・`regateGitTimeout` を越えた・signal で終わった・ctx が済んだときに error を返し(`:808-819`)、`checkBranchTip` はそれを `return "", "", err` で返す(`:629-630`)。`checkWorktree`(`:562-564`)と `run`(`:405`、`:409-414`)も error を落とさず返す。git が走ったうえで終わりが 0 でないときだけ「手元にない」の断りになる。scratchpad の repo(git 2.49.0)で打ち、`git cat-file -e <無い SHA>^{commit}`、commit でない object への `^{commit}`、壊れた(空の)loose object は、どれも exit 128 だった。`git rev-parse --verify --quiet` に替えても、無い SHA も壊れた object も exit 1 で、終わりの番号では「無い」と「読めない」を分けられない。repo そのものが使えない場合(所有者違い、repo でない)は、先に走る `readState` の `rev-parse` が error にする(`:440-448`)ので、ここまで来ない。残るのは壊れた object を「無い」と読む場合だけで、断りは何も消さず、文は fetch を勧めるだけなので所見にしていない。
- 番号。`fetchHeadCmd`(`:657-662`)は `c.p.PRNumber > 0` のときだけ `refs/pull/<n>/head` を出し、0 でも負でも head の SHA で fetch する(`TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch` の「番号なし」)。`strconv.Itoa` の数字だけで、head は `gitObjectPattern`(`^[0-9a-f]{40}([0-9a-f]{24})?$`、`gate_record.go:18`)に通った SHA。番号の渡し方は、CLI の 2 か所(`cleanupSteps` の merged、`cleanupNamedStep`、`internal/cli/org_cleanup.go:190`、`:227`)と merge の 1 か所(`mergeCleanup`、`internal/org/merge_run.go:291`、`res.PRNumber`)で、`CleanupParams{` を作る呼び手はこの 3 つだけ。PR を読めなかったブランチの step(`unknown`)は番号が 0 のままだが、片付けの呼び出しに届かない。
- R2-L4。`regate.go:567` の `ralph org cleanup --branch` は `cleanupShellWord(f.Branch)` を通り、テスト(`TestRegateCheckWorktree_QuotesTheBranchInTheCleanupCommand`)は `feat/a;b` を `'feat/a;b'` にすることを見る。同じ文の `git worktree add %s %s`(`f.Worktree` と `f.Branch`)は素のままだが、base の 15e301c8 の時点からある(`git show 15e301c8:internal/org/regate.go` の 555 行)ので、この PR の取りこぼしではない。
- R2-L5。`shellSafeUnquoted` の注釈(`internal/cli/org.go:589-598`)の呼び手の数え方を、呼び手 5 つ(`orgReadCommandHint` と `org_legacy_ledger.go:53-73` がパス、`org_cleanup.go:221` と `org_merge.go:263` がブランチ)と照らした。合っている。2 つの関数が残る件は、台帳の 1 行(`docs/tech-debt/README.md:385`)が受けた。行の主張(`cleanupShellWord` が `[A-Za-z0-9_./:@+,-]`、`shellQuoteIfNeeded` が `[A-Za-z0-9_./-]`、CLI の 2 つの文)は、`cleanup.go:80` と `org.go:598`、`org_cleanup.go:221`、`org_merge.go:263` に合う。
- R2-L6。`Failed` の 2 つの出どころは `remove`(`:720-739`)の `res.Failed` の 2 行(`:726`: 登録の外しの失敗、`so nothing was removed`、`:734`: script の失敗、`what it removed before it stopped stays removed`)。`CleanupResult.Failed` の注釈(`:122-127`)、`/director` skill の 2 か所(`:862-873`、`:973-981`)、`/org` skill の 3 か所(merge の行、cleanup の行、後始末の節)、README(`:268`)、仕様 FR-8 の注、計画の D2・D3 と、固定の断片(`director_skill_sync_test.go` の `directorSkillMergeCleanupRules` と `directorSkillGitHubMergeCleanupSteps`、`org_skill_send_sync_test.go` の `orgSkillCleanupRules`)が、どれも 2 つを言い分けている。skill の 8 つの写し(`/director` と `/org` の `.claude`・`.agents`・`templates/base` の 2 つずつ)は `cmp` で同一。merge と cleanup の help は「the removal itself failed」で 2 つを含む。「script が途中で止まった」だけを言う文は、skill・README・仕様・計画の現行の文にもう残っていない(残るのは計画の Progress の当時の記録だけ)。「A cleanup that stopped part way goes on from what is left」の help は、前の片付けが残したものの再開の話で、登録の外しの失敗はその「残したもの」を作らないので当たらない。
- R2-L7。`/director` skill の手順 3(`.claude/skills/director/SKILL.md:973-981`)に、同じ実行の `CLEANED:` を手順 2 と同じく報告する文が入った。`internal/cli/org_cleanup.go:254-269` のブランチごとに続ける作りと、`:342-351` の「1 本でも残れば終了コード 1」に合う。
- R2-L8。台帳の行(`docs/tech-debt/README.md:383`)から行数が消えた。
- ファイルの行数。`internal/org/cleanup.go` は fb214c0d の 796 行から 841 行になり、800 行を越えた(R3-L3)。`internal/cli/org_cleanup_test.go` は 750 から 800、`internal/org/cleanup_test.go` は 1030 から 1165。
- 確かめ 5 の文の安全。名指されるコマンドの組み立ては `cleanupGitCmd`・`c.gitCmd` で、引数ごとに `cleanupShellWord` を通る。`git -C <root> branch <branch>-unmerged <branch>` は、結合した 1 語を通すので、`'` を含むブランチでも `'a'\''b-unmerged'` の形になる。`git -C <root> fetch origin refs/pull/<n>/head` と SHA の fetch は、語が `[A-Za-z0-9_./:@+,-]` だけで素のまま。退避の名前がすでにあるときの `git branch` の失敗は scratchpad で打ち、`fatal: a branch named 'feat/auth-unmerged' already exists`(exit 128)で、ブランチは動かない(R3-L1)。`reset --hard` は repo の hook が打たせないので打っていない。
- R2-L1 の文と R2-L3 の直しの範囲。`git log <head>..<branch>` を名指す文は 4 か所(`:478`、`:510`、`:515`、確かめ 5 の `branchMovedSteps`)あり、手元にない head への備えは確かめ 5 の 1 か所だけ(R3-L2)。
- F3 の implementer が範囲外として挙げた 3 点(orchestrator 経由)。(1) `cat-file` の非 0 を全部「head が手元にない」と読む件は、上の R2-L3 の項のとおり、終わりの番号では「無い」と「読めない」を分けられない(`cat-file` も `rev-parse --verify --quiet` も壊れた object を 128 か 1 にする)ので、所見にしていない。残る手触りは、壊れた object のときに fetch しても次の片付けが同じ断りを返す点で、断りの文に git の stderr を足せば見分けられるが、repo の壊れが前提で何も消えないため LOW にも立てない。(2) 確かめ 2 の断りの `git log` は R3-L2 に含めている(`:478`、`:510`、`:515`)。(3) `regate.go:567` の同じ文の `git worktree add %s %s` は、base の 15e301c8 の 555 行からある文言で、この PR の diff は `cleanup --branch` の節だけを足した。この PR の取りこぼしではないので所見にしない。直すなら、`internal/cli/org_regate.go:203` の regate の文(計画 F2b が範囲外とした行)と一緒に、`cleanupShellWord` を通す別の変更にする。

### run 2(HEAD e316d081)

- 断りの順と、それぞれの断りが渡す手のコマンド(Manual)。HEAD で走る順に並べた。
  - 引数の確かめ、`regate.lock` が取れない、確かめ 1(結びつきがない、org_id が ralph の形でない、org が走っている)、記録が読めない、script がない: どれも Manual なし。印は確かめ 1 のどの断りにも当たらなかったときだけ書く(`internal/org/cleanup.go:295`)。
  - 確かめ 2: 記録がない(`recordGone`)は `git worktree remove` と `git branch -D`、記録のブランチが違うときは `git branch -D`。どちらも理由の文が `git log <head>..<branch>` で確かめるよう言う(`cleanup.go:466-470`、`:499-501`)。記録のパスか kind だけが違うときは script の `--force-branch`(`:503-505`、R2-L1)。
  - パスがディレクトリでない、確かめ 3、確かめ 4(worktree のルートでない、rebase か merge の途中、未コミット・未追跡): Manual なし(`:518-521`、`:535-536`)。
  - 確かめ 5: ブランチが動いていれば Manual なしで、理由の文が戻し方のコマンドを名指す(`:604-618`)。ローカルのブランチがなければ script の `--force-branch` で、理由の文が `git -C <worktree> status` で確かめるよう言う(`:609-611`)。
  - 確かめ 6: Manual なし(`:555`、`:640-651`)。
  - 消す途中の失敗(登録だけ残った worktree の `git worktree remove` の失敗と、script の失敗)は Manual あり。どちらも確かめをすべて通ったあと(`:678-693`)。
  - 確かめ 5 と 6 より前で `--force-branch` を渡すのは、確かめ 2 のパスか kind の違いだけになった。確かめ 5 のブランチがない場合は 6 の前だが、6 が守るもの(worktree の HEAD のコミット)は理由の文が `git status` で見るよう言う。
- 確かめ 5 の戻し方の場面分け(`cleanup.go:388-397`、`:542-553`)を git 2.49.0 で scratchpad の repo に打って確かめた。登録が残ったまま中身が消えた worktree のブランチには `git branch -f` が `cannot force update the branch ... used by worktree` で断り、`git update-ref refs/heads/<b> <head>` は動かす。`<head>` が手元にないと `git log <head>..<b>` は `Invalid revision range` で終わる。`git -C <worktree> switch -C <b> <head>` は、worktree がそのブランチにいても別のブランチにいても、そのブランチを head に動かして worktree をそこに置き、退避に作ったブランチは残す。そのブランチをほかの worktree が持っていれば断る(R2-L2 の勧め)。`reset --hard` は repo の hook が打たせないので打っていない。別のブランチの上で打てばそのブランチが動くことは、コードの注釈(`cleanup.go:542-546`)と git の reset の作りによる。
- quote。org 側の `cleanupShellWord`(`cleanup.go:789-795`)は `[A-Za-z0-9_./:@+,-]` の外の文字を含む語を単引用符で囲み、`'` を `'\''` にする。CLI 側の `shellQuoteIfNeeded`(`internal/cli/org.go:601-606`)は同じ囲み方で、素のままにする文字が `[A-Za-z0-9_./-]` と狭い。`PrintableInboxText` が書き換えるのは制御文字だけ(`internal/org/inbox.go:180-194`)なので、引用符と `\` は出力まで残る。テストを除く Go の `--branch %s` を grep し、この PR が書いた `ralph org cleanup --branch` のうち `internal/org/regate.go:567` だけが素のまま(R2-L4)。ほかの素のもの(`regate --branch`、`merge --branch`)はこの PR の前からある。
- org_id の確かめ(`cleanup.go:280-284`)は `cleanupFirstCheck` の中にあり、`ralph org cleanup` も `ralph org merge` の続きの片付けも `cleanupFeatureWorktreeLocked` からここを通る。印を書く `return` の前の `switch` の 1 つなので、断れば印は書かない。
- `/director` と `/org` の skill の 4 つの写しは `cmp` で同一。「GitHub で merge した PR の後始末」の 5 手順を、`ralph org cleanup` の出力の作り(`internal/cli/org_cleanup.go:254-351`、ブランチごとに `CLEANED:` か `SKIPPED:` を出し、1 本でも残れば終了コード 1)と照らした(R2-L7)。
- 台帳の 2 行(`docs/tech-debt/README.md:383-384`)の主張を HEAD で打ち直した。384 行の (a)(`runWorktreeScript` に時間の上限も ctx もない、`internal/org/feature.go:691-703`)と (b)(`StartFeature` の `regateSpawnErr` が manifest のロックの外、`feature.go:224`)は今も当たる。383 行の行数は、その行を書いた 5442c872 のあとの F1・F2 のテストの追加で古くなった(R2-L8)。

### run 1(c55819e9 の時点。行番号もその時点のもの)

- `internal/org/cleanup.go` の全体と、そこから呼ぶ `acquireHeldFileLock`・`release`・`fileLockHeld`(`internal/org/lockfile.go`)、`writeLockMarker`・`readRegateMarker`・`readRegateBranch`(`internal/org/regate.go`)、`boundedGit`(`internal/org/regate_git.go`)、`samePath`・`resolvedOrClean`(`internal/org/statedir.go`)、`runWorktreeScript`・`featureWorktreeCleanup`・`shellQuote`(`internal/org/feature.go`)、`validateFeatureBinding`(`internal/org/reserve.go`)、`scripts/ralph-worktree.sh` の `cleanup_worktree`・`state_path`・`sanitize_id`。
- 消す相手を取り違えないか。消す操作は script の `cleanup --id org-<org_id> --force-branch` だけで、その前に記録の branch・worktree_path・kind を binding と照らす(`checkRecord`、`cleanup.go:450-474`)。worktree のパスが別の worktree への symlink のときは、`samePath` が両側を解決するので確かめ 2 と 4 は通るが、確かめ 6 の `symbolic-ref` がその先のブランチを読んで断る(`cleanup.go:566-583`)。main worktree を指す symlink も同じく 6 で止まる。登録だけ残った worktree は `git worktree remove <そのパス>` だけで外し、`prune` は使っていない(`cleanup.go:607-619`)。
- git に渡す値。branch は `refs/heads/` の後ろか Manual の中だけで argv の位置引数にならず、worktree は binding の時点で絶対パスに限られる(`reserve.go:174`)。head は `gitObjectPattern` で SHA に限る(`cleanup.go:231`)。gh の `headRefOid` もこの確かめを通る。
- 端末に出す文。Refused・Failed・Manual はどれも `PrintableInboxText` を通り(`cleanup.go:156`、`:209`、`:328`、`:613-624`)、CLI の `CLEANUP:`・`CLEANED:`・`SKIPPED:`・`MANUAL:` の行も通す(`internal/cli/org_merge.go:333-357`、`internal/cli/org_cleanup.go:295-330`)。`ralph org cleanup` がそのまま返す error(`org_cleanup.go:312-313`)は main.go が素で出すが、regate の CLI も同じ形なので所見にしていない。
- `regate.lock` の印。片付けは印を、manifest のロックの下で `RunningOrgs` を確かめたあとに書く(`cleanup.go:254-283`)。lock を放したあとも印はファイルに残るが、これは Regate と同じ作りで(`regate.go:500-503`)、読み手の `regateSpawnErr` は `fileLockHeld` が持たれていると答えたときだけ印を読む(`spawn.go:1473-1486`)。lock を取る側は Regate(`regate.go:317`)、Merge(`merge_run.go:233`)、`CleanupFeatureWorktree`(`cleanup.go:173`)のどれも取った直後に印を空にする。放したあとの古い印が spawn を断る窓は、次の持ち手が取ってから空にするまでの間だけで、前からある窓と同じ。
- `ralph org merge` の終了コード。片付けの結果は `MergeResult.Cleanup` と `CleanupErr` にだけ入り、`mergeLocked` が返す err は `mergeAndRecord` のものだけ(`merge_run.go:258-273`)。`printMergeResult` の分岐は変わっていない。exit 0 は記録した merge のままで、片付けの途中の割り込みが exit 1 になる点は help に書いてある(`org_merge.go:167-175`)。
- skill の 4 つの写しは `cmp` で同一(`/org`、`/director` とも)。recipe の 2 つも同一。
- ファイルの行数。base と HEAD で `wc -l` を比べた(L5)。

## Findings

### HEAD の所見(run 3)

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3(HEAD afe9c1ed について)。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security | R3-L1: R2-L2 の直しで、確かめ 5 の文が退避に名指す `git -C <root> branch <branch>-unmerged <branch>` の名前は固定で、すでにあるときの扱いを文が言わない。`<branch>-unmerged` がすでにあると `git branch` は `fatal: a branch named ... already exists`(exit 128)で止まり、何も作らない。文の次の手は `reset --hard <head>` か `branch -f <branch> <head>` で、退避が作られないまま打つと merge されていないコミットはどのブランチからもたどれなくなる(reflog には残る)。エラーは出るので、人が続けて打つ場合に限る。名前がぶつかるのは、同じブランチ名の機能を前に片付けようとして退避を残したままのとき。テストは名前がまだないときの道だけを通す | `internal/org/cleanup.go:646-651`(`c.f.Branch+"-unmerged"` が固定)、`:640-645`(注釈)、probe: 同名のブランチがあるときの `git branch feat/auth-unmerged feat/auth` は exit 128 で、`feat/auth` は動かない(git 2.49.0)。テスト `internal/org/cleanup_test.go` の `TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten` と `…ResumesWhatAnEarlierCleanupLeft` は新しい repo | 退避の名前に、ブランチの先頭の短い SHA を入れる(`<branch>-unmerged-<tip の先頭 7 桁>`。すでにあるなら、同じ先頭を残してあるので、そのまま次へ進んでよい)。または、文に「退避が作れなければ、次の手を打たずに止まる」を足す |
| LOW | exception-handling | R3-L2: R2-L3 の直し(手元にない head のときの `fetchHeadCmd`)は確かめ 5 の断りにだけ入った。`git log <head>..<branch>` を名指す文はほかに 3 か所あり、R2-L1 の直しが 1 か所足した。どれも `<head>` が手元になければ `fatal: Invalid revision range` で終わり、fetch を勧めない。head が手元にないのは `ralph org cleanup` の head が gh の `headRefOid` のとき(GitHub 上で足されたコミット)で、確かめ 2 の断りも同じ場面で出うる。何も消さず、エラーも出るので、先に進めないだけ | `internal/org/cleanup.go:478`(`recordGone`)、`:510`(`checkRecord`、記録のブランチが違う)、`:515`(`checkRecord`、R2-L1 で足した文)、対して `:632-634`(確かめ 5 の fetch)。probe: 手元にない SHA の `git log <sha>..<branch>` は `fatal: Invalid revision range`(run 2 の R2-L3 の probe) | `cat-file` の確かめと fetch の文を小さな関数(たとえば `headFetchNote`)に出し、4 か所で使う。または台帳に「確かめ 2 の断りの `git log` は head が手元にあることを前提にする」と残す |
| LOW | maintainability | R3-L3: F3 で `internal/org/cleanup.go` が 796 行から 841 行になり、800 行を越えた(新しいファイルで、base にはない)。台帳の行 383 は、800 行を越えたテストの 4 ファイルだけを挙げ、`cleanup.go` は入っていない。`internal/cli/org_cleanup_test.go` は 750 から 800 行で、境界にある | `internal/org/cleanup.go` の `wc -l`(fb214c0d は 796、HEAD は 841)、`docs/tech-debt/README.md:383`、`internal/cli/org_cleanup_test.go`(800) | 台帳の行 383 に `internal/org/cleanup.go`(841)を足す。切り出すなら、git を打つ道具(`cleanupGit`・`cleanupGitOutput`・`cleanupInProgress`・`cleanupShellWord`・`cleanupGitCmd`)を `cleanup_git.go` に移す |

### run 2 の所見の扱い(HEAD afe9c1ed で確かめた)

| ID | run 2 の重さ | 扱い | 状態 |
| --- | --- | --- | --- |
| R2-L1 | LOW | 確かめ 2 で記録のパスか kind だけが違う断りの文に、`git log <head>..<branch>` と、記録のパスへの `git -C <path> status` を足し、`scriptCmd` の注釈を直した(`internal/org/cleanup.go:513-516`、`:330-333`、`:483-490`)。パスは `cleanupGitCmd` で quote される。テストは空白を含むパスの文を見る | 解消。文の中の `git log` が手元にない head のとき動かない点は R3-L2 |
| R2-L2 | LOW | 退避を main worktree から作る別のブランチにし、feature worktree でブランチを切り替えないよう文で言う(`:640-651`)。名指すのは `git -C <root> branch <branch>-unmerged <branch>`。テストは、名指したコマンドを打ってから片付けが終わるまでを通し、退避が動いていないことも見る | 解消。退避の名前がすでにあるときの扱いは R3-L1 |
| R2-L3 | LOW | 確かめ 5 の前に `git cat-file -e <head>^{commit}` を打ち、手元になければ、fetch(`refs/pull/<n>/head`、番号がなければ SHA)を先に出す断りにした(`:617-638`、`:657-662`)。`CleanupParams.PRNumber` を merge と cleanup の呼び手が渡し、0 と負は SHA の fetch にする。git の失敗は error で返る | 解消(確かめ 5)。ほかの `git log` の文は R3-L2 |
| R2-L4 | LOW | `regate.go:567` の `ralph org cleanup --branch` を `cleanupShellWord` に通した。テストは `feat/a;b` を見る | 解消 |
| R2-L5 | LOW | `shellSafeUnquoted` の注釈を「パスとブランチ名」に直し、2 つの関数が残る件を台帳に足した(`internal/cli/org.go:589-598`、`docs/tech-debt/README.md:385`) | 注釈は解消、関数が 2 つの件は繰り越し(台帳あり) |
| R2-L6 | LOW | `Failed` の 2 つの出どころを、`Failed` の注釈、skill の 8 つの写し、README、仕様 FR-8 の注、計画の D2・D3、固定の断片に書いた。コードの `remove` の 2 つの `res.Failed` と合う | 解消 |
| R2-L7 | LOW | `/director` skill の手順 3 に、同じ実行の `CLEANED:` を報告する文を足し、固定の断片を足した | 解消 |
| R2-L8 | LOW | 台帳の行 383 から行数を消した | 解消。台帳の行は `cleanup.go` を挙げていない点が R3-L3 |

### run 2 の所見(e316d081 の時点の文のまま。行番号もその時点のもの)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | security | R2-L1: 確かめ 2 で記録のパスか kind だけが binding と違うときの断りは、今も script の `--force-branch` を Manual に渡す。確かめ 4〜6 はまだ走っておらず、記録のブランチは binding のブランチと同じなので、人が「この機能の記録だ」と判断して打つと、push していないコミットのあるブランチも `git branch -D` で消える(script は未コミットの変更だけを断る)。同じ確かめ 2 のほかの 2 つの断り(記録がない、記録のブランチが違う)は、手のコマンドの前に `git log <head>..<branch>` で確かめるよう言うのに、この断りの文は「記録を確かめ、この機能のものなら手で片付ける」としか言わない。`scriptCmd` の注釈の「binding の記録の Manual」もこの使い方には当たらない。run 1 でも同じ文だった(run 1 の見落とし)。F1 は Manual の注釈でこの場合を残すと書き直した | `internal/org/cleanup.go:503-505`、比べる文 `:466-470`・`:499-501`、`:324-325`(`scriptCmd` の注釈)、`scripts/ralph-worktree.sh:390-402`(dirty だけを断り、`--force-branch` で `branch -D`) | 理由の文に、ほかの 2 つと同じく「ブランチに merge されていないものがないこと(`git log <head>..<branch>`)と、記録のパスの worktree の状態(`git -C <記録のパス> status`)を確かめてから」を足す。`scriptCmd` の注釈を「記録のブランチが binding のブランチのときの Manual」に直す |
| LOW | security | R2-L2: ブランチが動いたときの確かめ 5 の文は、「要るものを別のブランチに置き」のあとに、断った時点の worktree の場所で選んだ戻し方を名指す。worktree がそのブランチにいれば `git -C <worktree> reset --hard <head>` だが、退避を worktree の中の `git switch -c <退避>` で済ませると worktree は退避のブランチに移り、名指された `reset --hard` は退避のブランチのほうを head に戻す(F1 の注釈自身が、別のブランチの上では reset がそちらを動かすと書いている)。次の片付けは worktree が別のブランチにいる場面として `git branch -f` を名指すので、退避したつもりの人が打つと、merge されていないコミットはどのブランチからもたどれなくなる(退避のブランチの reflog には残る) | `internal/org/cleanup.go:542-553`(場面分けと注釈)、`:612-617`(文の順)、probe: `git -C <worktree> switch -C feat/a <head>` は worktree がそのブランチにいても別のブランチにいても動き、退避のブランチを残し、ほかの worktree が持つブランチは断る(git 2.49.0) | worktree がそのブランチにいる場面の戻し方を `git -C <worktree> switch -C <branch> <head>` にする(worktree がどこにいても同じに効き、確かめ 6 も同じ一手で通る)。または退避の手順を worktree を動かさないコマンド(`git -C <root> branch <名前> <先頭>`)で名指す |
| LOW | exception-handling | R2-L3: 確かめ 5 の文が名指すコマンド(`git log <head>..<branch>`、`reset --hard <head>`、`branch -f <branch> <head>`、`update-ref ... <head>`)は、`<head>` が手元の repo にある前提で書かれている。`ralph org cleanup` の head は gh の `headRefOid` で、GitHub の上で足されたコミット(Update branch の merge コミット、suggestion のコミット、別の clone からの push)が PR の head なら手元にない。そのとき確かめ 5 は、手元のブランチが head の祖先で失うものがなくても断り、名指すコマンドはどれも `Invalid revision range` か `not a valid object name` で終わる。片付けは fetch を打たない。危険ではないが、文のとおりに打っても先に進めない。`git log` は run 1 の文にもあり、戻し方の 3 つは F1 で足した | `internal/org/cleanup.go:612-617`、`:388-397`、`:542-553`、probe: 手元にない SHA で `git log <sha>..feat/a` が `fatal: Invalid revision range`。`cleanup.go` と `internal/cli/org_cleanup.go` に fetch はない | 確かめ 5 の前に `git cat-file -e <head>^{commit}` を打ち、手元になければ別の断り(「PR の head が手元にない。fetch してから打ち直す」)にする。少なくとも文に「git が `<head>` を知らなければ先に fetch する」を足す |
| LOW | security | R2-L4: この PR が S4b(111c62a2)で書いた regate の断りの `ralph org cleanup --branch %s` は、ブランチを素のまま入れる。F1 と F2b はほかの片付けのコマンド(`cleanupAgain`、`mergeCleanupInterrupted`、`mergeCleanupRoot`、`cleanupNamedStep`)を quote したが、ここは残った。計画の F2b の記録が範囲外として挙げるのは、この PR の前からある `internal/cli/org_regate.go:203` だけ | `internal/org/regate.go:567`(`f.Branch` が素)、比べる `internal/org/cleanup.go:249-251`、`internal/org/merge_run.go:277-281`、計画の F2b の行 | `cleanupShellWord(f.Branch)` を通す |
| LOW | maintainability | R2-L5: ブランチを shell の 1 語にする関数が 2 つあり、素のままにする文字が違う。org 側の `cleanupShellWord` は `:@+,` を素のままにし、CLI 側の `shellQuoteIfNeeded` は囲む。F2b で CLI 側にブランチ名の呼び手が 2 つ増えたので、`ralph org merge` と `ralph org cleanup` の出力の中で、同じブランチ(たとえば `feat/a+b`)が文によって素と単引用符の両方で出る。どちらも shell として正しい。`shellSafeUnquoted` の注釈の「呼び手は 1 つで、渡すのはファイルシステムのパスだけ」は base の時点で呼び手の数が合っておらず、F2b のあとはパスだけというところも外れた | `internal/cli/org.go:589-606`、呼び手 `internal/cli/org_cleanup.go:220`・`internal/cli/org_merge.go:263`(と base からの `org_legacy_ledger.go:53-73`)、`internal/org/cleanup.go:789-795` | 注釈を「パスとブランチ名」に直す。出力をそろえるなら、片付けの文は org 側の関数(公開する)か CLI 側の関数のどちらか 1 つで quote する |
| LOW | maintainability | R2-L6: `Failed` には出どころが 2 つある。script が途中で止まったもの(消した分は消えたまま)と、script の前の、登録だけ残った worktree の `git worktree remove` の失敗(文は「何も消していない」と言う)。M1 の直しで書いた skipped の読み方は、`/director` skill が「断られて何も消していないか、script が途中で止まったか」、`/org` skill が「消す途中で script が失敗したとき(消した分は消えたまま)」、README が「the removal failed part way and what it removed stays removed」で、どれも後者に当たらない。`CleanupResult.Failed` の注釈も script のことだけを言う。director は理由の文をそのまま伝えるので、伝わる中身は正しい | `internal/org/cleanup.go:678-685`(登録の外しの失敗、`so nothing was removed`)、`:117-120`、`.claude/skills/director/SKILL.md:864-867`・`:971-973`、`.claude/skills/org/SKILL.md:1130-1132`、`README.md:268`、仕様 FR-8 の注と計画の D3 | 「片付けの途中で失敗した(script が消した分は消えたまま。script の前で失敗したなら何も消していない)」のように、失敗の 2 つを含む言い方にする。`Failed` の注釈にも登録の外しの失敗を足す |
| LOW | maintainability | R2-L7: `/director` skill の 5 手順は、終了コード 0 なら `CLEANED:` を報告し(手順 2)、1 なら `SKIPPED:` と `MANUAL:` と stderr を伝える(手順 3)と、終了コードで分ける。`--branch` なしの 1 回で 1 本を片付けて別の 1 本を断られると、出力には `CLEANED:` と `SKIPPED:` の両方が出て終了コードは 1 になり、その `CLEANED:` はどちらの手順も報告しない。手順 4 が書く「残ったブランチがある間に `--branch` なしで打つ」はちょうどこの形の実行。手順 3 の形は run 1 からだが、F2 で書き直した | `.claude/skills/director/SKILL.md:969-980`、`internal/cli/org_cleanup.go:254-269`(ブランチごとに続ける)、`:342-351`(1 本でも残れば 1) | 手順 3 に「`CLEANED:` の行があれば、それも手順 2 と同じく報告する」を足す |
| LOW | maintainability | R2-L8: L5 の台帳の行(5442c872 で追加)の行数が HEAD と合わない。その行を書いたあとの F1・F2 がテストを足したため。`internal/cli/org_merge_test.go` は行の 1085 に対して HEAD で 1119、`internal/org/merge_run_test.go` は 1077 に対して 1091、`internal/org/director_skill_sync_test.go` は 936 に対して 955、`internal/org/cleanup_test.go` は 885 に対して 1030 | `docs/tech-debt/README.md:383`、HEAD の `wc -l` | HEAD の数に直すか、数を書かずに「800 行を越えた」とだけ書く(このあとのコミットでもずれない) |

### run 1 の所見の扱い(HEAD e316d081 で確かめた)

状態の欄が R2-L<n> へ繰り越したものは、HEAD afe9c1ed では上の「run 2 の所見の扱い」の表で閉じている(L5 の台帳の行は R2-L8 と R3-L3)。

| ID | run 1 の重さ | 扱い | 状態 |
| --- | --- | --- | --- |
| M1 | MEDIUM | 行の形は分けず(計画 D3 に理由を記録)、読み方の文を直した。`/director` skill は skipped を「片付けは終わっていない。断りか script の途中の停止のどちらか」と読ませ、`CLEANUP MANUAL:` がない断りは打ち直しで続くと書く(`.claude/skills/director/SKILL.md:864-872`、`:971-976`)。`/org` skill、merge と cleanup の help(`internal/cli/org_merge.go:129-136`、`internal/cli/org_cleanup.go:79-92`)、README、仕様 FR-8 の注も同じ趣旨。「何も消していないので」の文と、それを固定していた断片は消えた(`internal/org/director_skill_sync_test.go` の `directorSkillMergeCleanupRules`) | 解消。言い方の取りこぼしは R2-L6 |
| M2 | MEDIUM | 確かめ 3・4・5(ブランチが動いた)・6 の断りから Manual を外した(`internal/org/cleanup.go:518-521`、`:535-536`、`:612-617`、`:555`)。確かめ 5 の文は、要るものを別のブランチに置き、場面ごとのコマンドでブランチを head に戻してから打ち直すよう言う(`:388-397`、`:542-553`)。テストの断りの表の Manual の列も空に変わった | 解消。残る `--force-branch` は R2-L1、戻し方の文の穴は R2-L2 と R2-L3 |
| L1 | LOW | `cleanupFirstCheck` で org_id を `ValidateIdentifier` に照らし、通らなければ印を書かずに断る(`internal/org/cleanup.go:280-284`、印は `:295`)。merge の続きの片付けも同じ関数を通る | 解消 |
| L2 | LOW | 理由の文の中のコマンドを `cleanupGitCmd` で組み、パスとブランチを `cleanupShellWord` で quote した(`internal/org/cleanup.go:339-345`、`:470`、`:501`、`:611`、`:615`、`:642`、`cleanupAgain` の `:249-251`)。CLI 側も F2b で quote した(`internal/cli/org_cleanup.go:220`、`internal/cli/org_merge.go:263`) | 解消。この PR の regate の文の取りこぼしは R2-L4 |
| L3 | LOW | `FeatureRepoRoot` の error を文に入れず、片付けの言葉の定数 `orgMainWorktreeMissing` にした(`internal/cli/org_merge.go:245-251`、`:263`、`internal/cli/org_cleanup.go:124-127`)。定数の 2 つの場合は `FeatureRepoRoot` の 2 つの error の道(`internal/org/statedir.go:135-140`)と合う | 解消 |
| L4 | LOW | 手順 1 を「前の報告のあとに新しい `merged` が出たとき」にし、「何度打ってもよい」を消し、手順 4 で「`SKIPPED:` のブランチは人が直したと言うまで打ち直さない。その間の新しいブランチは `--branch` で名指す」と書いた(`.claude/skills/director/SKILL.md:964-980`) | 解消。手順 2 と 3 の分け方の穴は R2-L7 |
| L5 | LOW | 台帳に行を足した(`docs/tech-debt/README.md:383`、5442c872) | 繰り越し(台帳あり)。行数は R2-L8 |
| L6 | LOW | 台帳に 2 件を 1 行で足した(`docs/tech-debt/README.md:384`)。HEAD で 2 件とも今も当たる | 繰り越し(台帳あり) |

run 1 の Follow-ups の「`/verify` 向け」(仕様 FR-8 の注と計画 D2 の 3 の、確かめ 3 の理由)は、F2 と 5442c872 で「消す worktree の中に、打った人の shell を残さないため」に直った(`internal/org/cleanup.go:34-35` の注釈も同じ)。

### run 1 の所見(c55819e9 の時点の文のまま。行番号もその時点のもの)

| Severity(run 1) | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: script が途中で失敗した片付け(`Failed`)も `CLEANUP: skipped` と出し、`/director` skill はその行を「何も消していない」と読ませる。`Failed` の文は「止まる前に消したものは消えたまま」と言うので、director が人に伝える文が食い違う。`ralph org cleanup` の `SKIPPED:` も同じく、断りと途中の失敗を分けない | `internal/cli/org_merge.go:353-354`(`cmp.Or(c.Refused, c.Failed, ...)` を skipped で出す)、`internal/org/cleanup.go:620-625`(Failed の文)、`.claude/skills/director/SKILL.md:864-865`、その文を固定する `internal/org/director_skill_sync_test.go:852`、`internal/cli/org_merge_test.go:404-406`、`internal/cli/org_cleanup.go:317` | `Failed` は別の語で出す(たとえば `CLEANUP: failed (<理由>)` と `FAILED: <branch> <理由>`)。少なくとも skill の文を「skipped なら消していないか、途中まで消した。理由の文がどちらかを言う」に直し、固定の断片も合わせる |
| MEDIUM | security | M2: 確かめ 3(worktree の中から打った)、4(未コミット・未追跡・rebase か merge の途中)の断りに付く Manual は script の `--force-branch` で、確かめ 5(ブランチの先頭が merge された head か)はまだ走っていない。理由を直してから Manual を打つと、push していないコミットのあるブランチも `git branch -D` で消える。同じ場面で `ralph org cleanup --branch` を打ち直せば確かめ 5 で止まる。help は Manual を「理由を確かめたら打つコマンド」と書く | `internal/org/cleanup.go:482-484`(確かめ 3 が `c.scriptCmd()` を返す)、`:498-505`(4 の次に 5)、`:310-312`(`scriptCmd`)、`internal/org/cleanup_test.go:339`(未追跡の断りの Manual が `fx.scriptCmd()`)、`internal/cli/org_merge.go:131`、`internal/cli/org_cleanup.go:81-82` | ralph が続きから片付けられる断り(3、4、6)では Manual を出さないか、`ralph org cleanup --branch <b>` にする。script の Manual を残すなら、確かめ 5 を 3 と 4 より前に動かし、先頭が head と一致したときだけ出す |
| LOW | security | L1: org_id を台帳から読んだまま使う。`scriptCmd` は org_id を quote せずに Manual に入れ(root と git の引数は `cleanupShellWord` を通すのに、ここだけ素)、`c.id()` は script の `--id` になる。script の `sanitize_id` は大文字を小文字に、記号を `-` に変えるので、書き換えた台帳の org_id `Auth` は org `auth` の記録を指し、`RunningOrgs` は `Auth` で引くので `auth` が走っていても止まらない(probe: `Auth;x` は `auth-x` になる)。`ValidateIdentifier` の注釈は「不正な id がパスを組む箇所に届かない」と言うが、台帳から読む道はその 2 層を通らない | `internal/org/cleanup.go:306`、`:311`、`:620`、`scripts/ralph-worktree.sh:60-73`、`internal/org/identifier.go:40-47` | `cleanupFirstCheck` で `b.orgID` を `identifierPattern` に照らし、合わなければ断る。それで Manual の素の語と、記録の id の取り違えの両方が消える |
| LOW | readability | L2: 断りの文の中のコマンド(`git -C <root> log <head>..<branch>`、`git -C <worktree> switch <branch>`、`git -C <worktree> status`)はパスを quote しない。同じ関数が Manual では `cleanupShellWord` で quote する。repo のパスに空白があると、文のとおりに打ったコマンドは動かない | `internal/org/cleanup.go:437-438`、`:468-469`、`:553-555`、`:557-559`、`:573-575`、`:580-582` | 文の中のパスも `cleanupShellWord` を通す。ブランチ名は空白を持てないのでそのままでよい |
| LOW | readability | L3: main worktree が見つからないときの文に、`FeatureRepoRoot` の error がそのまま入り、`start --plan` の説明と「run it from ...」がもう 1 つの次の手として混ざる(`errWorktreeScriptMissing` の「run start again」は S4 で外したが、こちらは残った) | `internal/cli/org_merge.go:247-253`、`internal/cli/org_cleanup.go:117-121`、`internal/org/statedir.go:136-140` | 原因だけを取り出す(`errors.Is` で 2 つの場合を分けて片付けの言葉で言う)か、`FeatureRepoRoot` の文を動詞によらない言い方にする |
| LOW | maintainability | L4: `/director` skill の「GitHub で merge した PR の後始末」は、手順 1 で「`merge-order --all` に `merged` が出たら打つ。何度打ってもよい」と言い、手順 3 で「終了コード 1 なら打ち直さない」と言う。片付けたあとも `merged` の行は残る(`/org` skill)ので、断られたブランチが残る間は、`--all` を読むたびに手順 1 の条件が成り立ち、手順 3 とぶつかる。どちらが先かが書いていない | `.claude/skills/director/SKILL.md:960-972`、`.claude/skills/org/SKILL.md` の merge-order の節(「片付けたあとも `merged` の行は残る」) | 手順 1 に「前に `SKIPPED:` を伝えたブランチだけが残っているなら、人に頼まれるまで打たない」を足すか、手順 3 の「打ち直さず」の範囲を `--branch` で同じブランチを打つことに限ると書く |
| LOW | maintainability | L5: この PR で 800 行を越えたテストのファイルが 4 つあり、台帳に行がない。`internal/cli/org_merge_test.go` 798 から 1085、`internal/org/merge_run_test.go` 790 から 1077、`internal/org/director_skill_sync_test.go` 795 から 936、新しい `internal/org/cleanup_test.go` 885 | `wc -l` の base と HEAD の比較。`docs/tech-debt/README.md` にこの 4 ファイルの行はない | 片付けのテストを `org_merge_cleanup_test.go`・`merge_cleanup_test.go` などに分けるか、台帳に行を足す |
| LOW | maintainability | L6: 計画の Progress に「既知の穴」と書いた 2 件が台帳にない。(a) `runWorktreeScript` は script を時間の上限も ctx もなしで打つので、script か git が止まると `regate.lock` を持ったままになる。(b) `StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれ、片付けの印が書かれる前に通った start は、片付けが消す worktree を `ensure` で使い直しうる(最後の spawn は Spawn 側の確かめで断られる)。Progress は `/pr` で archive され、PR の本文は台帳の代わりにならない | 計画 `:165`、`:190`、`internal/org/feature.go:691-703`、`internal/org/cleanup.go:620`、`internal/org/feature.go:208-218` | 台帳に 2 行足す |

## Positive notes

### run 3

- 手元にない head の道を、origin に `refs/pull/7/head` だけがある repo で作り、断りの文のコマンド(fetch、`log`、退避、`reset --hard`)を `sh -c` で文のとおり打ってから、次の片付けが通ることまで見るテストがある(`TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch`、CLI の `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead`)。番号がある場合とない場合の 2 つの fetch を両方通している。
- `cat-file` の失敗を「手元にない」と取り違えない作りになっている。git が走れない・時間切れ・ctx の取り消しは error で返り、`checkBranchTip` から `checkWorktree` と `run` まで落とさずに運ばれる。
- R2-L2 の直しは、退避を feature worktree の外で作るという決め方で、戻し方の 3 つ(`reset --hard`、`branch -f`、`update-ref`)を変えずに済ませている。テストは退避が戻し方で動かないことを、退避の先を見て確かめる。
- Failed の 2 つの出どころの言い直しが、skill の 8 つの写し、固定の断片 3 つ、README、仕様、計画で同じ形にそろっている。`remove` の 2 つの `res.Failed` の文(「nothing was removed」と「what it removed ... stays removed」)をそのまま日本語にしている。

### run 2

- 確かめ 5 の文が名指すコマンドを、空白を含むパスの repo で `sh -c` に文のとおり渡して打ち、そのあとの片付けがすべての確かめを通ることまで確かめるテストがある(`TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten`)。文の中のコマンドが動くことを、文字列の一致でなく実行で押さえている。
- 登録の残った worktree で `git branch -f` が断られることを実装側が git で確かめ、`update-ref` に分けた。こちらの probe でも同じ結果が出た。テストは場面ごとに、名指したコマンドを打ってから片付けが終わるまでを通す。
- org_id の確かめは、走っている org の確かめと印を書く前に置かれ、テストの表の行が印を書かないことも見る。
- 中断のときの文を `mergeCleanupInterrupted` に切り出し、quote のテストを付けた。CLI の 2 つの文にも、`;`・`$(...)`・`'` のブランチで quote を確かめるテストがある。

### run 1

- 消す前の確かめが、消すと失うものの種類ごとに 1 つずつあり、順番と理由が `cleanup.go` の先頭の注釈と関数の注釈で追える。読めない記録、壊れた JSON、`rev-parse --git-path` の行数の食い違いは、どれも消さずに止まる。
- 途中で止まった片付けの続きを、残っているものの組み合わせ(`cleanupState`)で分け、それぞれに要る確かめだけをかける。`git worktree prune` を使わず、そのパスの登録だけを外している。
- `worktree_cleaned` の追記は `context.WithoutCancel` で、消したあとの割り込みで記録が抜けない。追記に失敗したときは `Cleaned` を立てたまま error を返し、CLI は `CLEANUP: done (...)` に error を入れる。
- 確かめの git は ralph の process group で走らせ(`ownGroup=false`)、Ctrl-C が git にも届く。regate の上限つきの git を prefix だけ変えて使い回し、`MergeParams.Exec` の固定の検査とは分けている。
- 印を書くのは manifest のロックの下で、`RunningOrgs` の確かめと同じ区間なので、spawn が確かめの前に来れば org が走っていると断り、あとに来れば印で断られる。

## Tech debt identified

台帳に入っている行は 3 つ。run 1 の L5(800 行を越えたテスト 4 ファイル)と L6(2 つの窓)は 5442c872 で入り(`docs/tech-debt/README.md:383-384`)、L5 の行の行数は F3 で消えた(R2-L8)。run 2 の R2-L5(quote の関数が 2 つ)は F3 の 5ddf5532 で入った(`:385`)。

この run のコミットはレポートと insight event に限るので、台帳には書いていない。R3-L1〜R3-L3 を直さずに進めるなら、次の 1 行を台帳に足す(行 383 の `cleanup.go` の分は、その行を直して足してもよい)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| merge のあとの片付け(#263)の断りの文の残り(self-review cycle 1 run 3)。(a) 確かめ 5 の文が退避に名指す `git branch <branch>-unmerged <branch>` の名前は固定で、すでにあると失敗し、文の次の手の `reset --hard` か `branch -f` が merge されていないコミットをブランチから外す(R3-L1)。(b) 手元にない head への fetch の案内は確かめ 5 の断りにだけあり、`git log <head>..<branch>` を名指す確かめ 2 の 2 つの断りと `recordGone` の断りにはない(R3-L2)。(c) `internal/org/cleanup.go` が 841 行で、800 行の目安を越えた(R3-L3) | (a) 人が失敗を見ても続けて打ったときだけ、reflog にしか残らなくなる。(b) head が手元にないと `git log` が `Invalid revision range` で終わり、先に進めないだけで何も消えない。(c) 読みにくくなるだけ | run 3 の所見が LOW だけなので、直さずに台帳へ記録して `/verify` に進める方針(#262 と同じ扱い) | 次に `internal/org/cleanup.go` の確かめ 2 か 5 の文を変えるとき。(a) は退避の名前に先頭の短い SHA を入れる、(b) は fetch の案内を小さな関数にして 4 か所で使う、(c) は git を打つ道具を `cleanup_git.go` に移す | `docs/plans/active/2026-10-28-org-merge-cleanup.md`、この report の R3-L1〜R3-L3 |

## Recommendation

- Merge: 可(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3。HEAD afe9c1ed について)。run 2 の R2-L1〜R2-L8 は、R2-L5 の関数が 2 つの件を台帳に繰り越した以外は解消した。run 1 の M1・M2・L1〜L4 は解消、L5・L6 は台帳に繰り越している。
- Follow-ups:
  - 方針どおり、LOW だけなので直さず、R3-L1〜R3-L3 を `docs/tech-debt/README.md` に 1 行で記録して `/verify` に進める(上の Tech debt の表。行は orchestrator が足す)。cycle 2 の再実行は、LOW を直すためには起こさない。
  - 3 つのうち R3-L1 だけは、人が失敗を見ても続けて打ったときに merge されていないコミットが外れる道で、R2-L2 の直しが持ち込んだもの。文の一言(退避が作れなければ止まる)か、退避の名前に先頭の短い SHA を入れる 1 行で閉じられる。ledger に入れるか、`/verify` の前に 1 slice で直すかは orchestrator が決める。直すなら、確かめ 5 の断りの期待文(`internal/org/cleanup_test.go` の `cleanupMovedSteps`、`internal/cli/org_cleanup_test.go` の `TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead`、`internal/org/merge_run_test.go`)と、`branch <branch>-unmerged` を打つテストの手も変わる。
  - `/verify` と `/test` は、この run が打たなかった `go test`・`go vet`・`golangci-lint`・`run-static-verify.sh`・`run-test.sh` を打つ。この run が読みで確かめたのは、文の中のコマンドの quote、`cat-file` の分岐、`PRNumber` の 0 の扱い、skill の 8 つの写しの一致(`cmp`)、`Failed` の 2 つの出どころの言い方。

## run 3 のあとの直し(F4)

- F4(3dfd434e)で R3-L1 を直した。確かめ 5 の断りが名指す退避は `git -C <root> branch <branch>-unmerged-<tip の先頭 7 文字> <branch>` になった。同じ名前のブランチがすでにあれば同じ tip から退避したものなので、`git branch` が失敗しても何も失わない。ブランチがまた動けば名前も変わる。起点はブランチ名のままで、断りのあとに足したコミットも退避に入る。
- 差分は `branchMovedSteps` の名前と注釈、テストの期待文、新しいテスト `TestCleanupFeatureWorktree_MovedBranchKeptUnderItsTip` だけ(3 ファイル、+124/-41)。
- run 3 の LOW は台帳に書いて進む方針だった。R3-L1 だけは、文のとおりに打つと merge されていないコミットがブランチから外れる道なので、例外として直した。差分が名前と期待文に限られるため run 4 は回さず、`/verify` と `/test` が通しで確かめる。上の verdict(HEAD afe9c1ed について)と `Merge:` の行は書き換えていない。台帳の行は R3-L2 と R3-L3 の 2 つで、R3-L1 の行は足していない。

## cycle 2(HEAD 33b78f8c)

- Scope: `git diff 732529d9..33b78f8c`(10 ファイル、+993/-127)。コミットは 8e46db0f(S-X1、cross-review cycle 1 の指摘 1)、d3452a9d(S-X2、指摘 2)、33b78f8c(計画の Progress の 2 行)。
- 命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。テスト、静的解析、文書のずれの確かめは打っていない(`/verify` と `/test` の範囲)。確かめたのは、読むことと `git` の読み取り、scratchpad の小さな git の実験(git 2.49.0)だけ。
- 行番号は HEAD 33b78f8c のもの。

### Evidence reviewed(cycle 2)

- 断るか進むかの分かれ目(S-X2)。`run` の 2 つ目の switch(`internal/org/cleanup.go:401-431`)は、パスがディレクトリでない、worktree がある(`checkWorktree`)、登録だけが残る(`checkRegistration`、通れば `update-ref` を戻し方にした確かめ 5)、登録もない(`branch -f` を戻し方にした確かめ 5)、記録だけ、の順。登録が残るときに確かめ 5 より先に登録を見る順は、先頭の注釈(`:45-58`)と `case st.registered:` の注釈に合う。
- 正しい再開を断らないか。登録が binding のブランチにいてブランチもあれば、前と同じく確かめ 5 に進む(`TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft` の「the directory deleted, its registration left」)。ブランチだけが消えて登録が残る形は script の順では作られない。scratchpad で、中身を消した worktree が持つブランチに `git branch -D` を打つと `cannot delete branch 'feat' used by worktree`(exit 1)で断られた。
- 消すべきでないものを消す道。登録が detached、別のブランチ、ブランチなし、HEAD が読めない、のどれでも `checkRegistration` は理由を返し、`remove` に届かない。人が打つ道の残りは C2-L3。
- `git worktree list --porcelain` の読み方(`readState`、`:458-474`)。scratchpad で、空白を含むパス(`worktree /…/wt one`)、`prunable`、`locked` の行を出させた。パスは引用なしで 1 行に出るので、`CutPrefix` がそのまま取る。`prunable` と `locked` の行はどの case にも当たらず読み飛ばされる。改行を含む lock の理由は `locked "line1\nworktree /fake\nbranch refs/heads/feat"` のように 1 行に引用されるので、別の登録の行のふりはできない。main worktree と bare の項目は `samePath` が合わず、`ours` にならない。
- ブランチを消した登録の出力。`update-ref -d` でブランチを消すと、git は `HEAD 0000…` と `branch refs/heads/<名前>` を出す(`regBranch` は残る)。`checkRegistration` の「which no longer exists」の分かれに入る。
- `checkRegistration`(`:768-800`)の 5 つの分かれと注釈を照らした。HEAD が読めない登録は、git が `HEAD 0000…` を出し、branch も detached も出さない。`regBranch == ""` の分かれと、ゼロの HEAD の文に入る。`cleanupState` の注釈(`:331-334`)の「A HEAD git cannot read has neither a branch nor detached」とも合う。HEAD がゼロの登録でも Manual の `git worktree remove <wt>` は通る(scratchpad で exit 0、登録が消えた)。
- テストが意図を見ているか。`TestCleanupFeatureWorktree_GoneWorktreeRegisteredOffItsBranchRemovesNothing`(`internal/org/cleanup_test.go:730`)の 4 ケースは、修正前のコードでは落ちる。detached と別のブランチは確かめ 5 を通って `remove` に進み、ブランチを消したケースは記録だけの再開に入って、どれも登録が消えるため。テストは断りのあとに登録、ブランチ、記録が残ることを見て、退避と Manual を `sh -c` で文のとおり打って結果も見る。
- origin の固定(S-X1)。`ReadMergeOrigin`(`internal/org/mergeorigin.go:87-110`)は、merge の前の `readOrigin` の 4 つの分かれ(ctx、時間切れ、失敗、repository なし。repository なしには `-` で始まる host・owner・repo も入る)をそのまま移した。merge の `readOrigin`(`internal/org/merge.go:360-379`)の文は変わっていない。`m.ghTimeout` には `:168-170` で既定値が入るので、時間切れの文の秒数は実際の上限と合う。
- host と owner/repo の比べ方。`mergeOriginFromURL` は host を小文字にし、ユーザーとポートを外し、パスの末尾の `/` と `.git` を外す。`hasPR` は PR の url の host を `EqualFold` で、owner/repo を `same`(大文字小文字を無視)で比べる。`ssh.github.com` は `mergeGuaranteeAPIHost` で `github.com` になる。http の url と repository を名指さない url は `mergeHostRepoFromPRURL` が `ok=false` を返し、PR を使わない側に倒れる(テストあり)。
- null の扱い。`foreignPRWhy` が受ける `pr` は nil にならない。`lookupMergePRs` が `pr == nil` を先に `continue` するため。
- 端末に出す文。`PRLookupWhy`(gh の error の文そのまま)は、`cleanupSteps` の行では `cleanupPrinter.skipped` が、`--branch` の断りでは `cleanupNamedStep` が `PrintableInboxText` を通す(`\x1b[31m` を含むテストあり)。`cleanupOriginRefusal` も git の error を通す。`PRLookupWhy` は `json:"-"` で、merge-order の JSON に出ない(テストあり)。
- 消した案内の残り。「the notes of ralph org merge-order --all say more」の文は、コード、skill、文書のどこにも残っていない(grep)。
- 行数(`git show 732529d9:<file> | wc -l` と HEAD)。`internal/org/cleanup.go` 846 → 935、`internal/cli/org_cleanup_test.go` 800 → 987、`internal/org/cleanup_test.go` 1243 → 1387、`internal/org/merge_test.go` 626 → 719、`internal/cli/org_cleanup.go` 352 → 420、新しい `internal/org/mergeorigin.go` 110。`mergeorder.go` と `mergeorder_test.go` はこの PR の前から 800 行を越えている。

### cycle 2 の所見

CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6(HEAD 33b78f8c について)。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | readability | C2-L1: HEAD がゼロの登録の断り(案 C)は、reflog の場所を言うために `git -C <root> rev-parse --git-common-dir` を名指す。main worktree でこれを打つと `.git` と相対で出る。`-C` で `<root>` に移るのは git の中だけなので、打った人の場所から読むと別の `.git` を指す。テストもこの相対のパスを `fx.root` につないでから確かめている。`worktrees/` の下のどのディレクトリかも文は言わない。git はパスの最後の部分を名前にし、使えない文字を `-` に替え(`wt one` は `wt-one`)、同じ名前があれば数字を足すので、`org-<org_id>` と同じとは限らない | `internal/org/cleanup.go:789-792`、`internal/org/cleanup_test.go:828-830`。probe(git 2.49.0): `/` から打つ `git -C <main> rev-parse --git-common-dir` は `.git`、`--path-format=absolute --git-common-dir` と `--absolute-git-dir` は絶対パス。`<common>/worktrees/wt-one/gitdir` の中身は `<…>/wt one/.git` | 名指すコマンドを `git -C <root> rev-parse --path-format=absolute --git-common-dir`(git 2.31 から)か `--absolute-git-dir` にする。ディレクトリは「中の `gitdir` に `<worktree>/.git` と書いてあるもの」と言い足す。案 C(退避のコマンドを出さない)は変えない |
| LOW | maintainability | C2-L2: 固定した読み方で gh が origin と違う repository の PR を返すと、cleanup の行(`SKIPPED:` と `--branch` の断り)は次の手を「gh が PR を読めたら、もう一度打つ」とだけ言う。この形は gh が答えたあとに起こるので、打ち直しても同じ行が出る。`-R` を固定したうえで起こる場面は、おそらく GitHub で repository の名前が変わり、origin が古い名前のままの場合(API は古い名前を受け、PR の url は新しい名前になる。固定しない道の `mergeOriginRepo.check` の注釈がこの場面を書いている。`-R` での挙動は未確認)。そのときは全ブランチの片付けが止まったままになる。固定しない道の文は `git remote set-url origin` と `gh repo set-default` を、merge の同じ断り(`mergeForeignPRRefusal`)は「origin を確かめる」を名指す | `internal/org/mergeorigin.go:45-52`、`internal/cli/org_cleanup.go:254`、`:279-281`。比べる文: `internal/org/mergeorder.go:790-795`(注釈)、`:818-820`、`internal/org/merge.go:432-436` | `foreignPRWhy` の文か cleanup の行の次の手に「origin を確かめる(`git remote get-url origin`)。repository の名前が変わったなら origin を新しい名前に向ける(`git remote set-url origin <URL>`)」を足す。gh が答えなかった場合(failed、limit_forks)の「gh が読めたら」は今のままでよい |
| LOW | security | C2-L3: 記録がないときの断りの Manual は、S-X2 が守るもの(中身の消えた worktree の登録の HEAD と reflog)を守らない。`recordGone` は登録が残っていれば Manual に `git -C <root> worktree remove <wt>` を入れ、理由の文は、ブランチに merge されていないものがないかを確かめるようにだけ言う。登録が detached で merge されていないコミットを持つとき、文のとおりに確かめて Manual を打つと、S-X2 の前と同じくそのコミットがどの参照からもたどれなくなる。`readState` は同じ `st` に登録の HEAD と branch を読んでいるので、文で言える。worktree が残っている場合も、Manual の `git worktree remove` は detached のコミットを確かめない。記録は script の最後に消えるので、記録だけがない形は手か gc でしか作られず、人が打つ道に限られる | `internal/org/cleanup.go:494-516`(`:498` の `git worktree remove`、`:509-514` の文)、`:458-474`(`readState` が `regHead`・`regBranch`・`regDetached` を読む)。比べる `checkRegistration`(`:768-800`) | `recordGone` で、登録だけが残り `checkRegistration` なら断る形(detached、別のブランチ、ブランチなし、HEAD が読めない)のときは、`checkRegistration` と同じ HEAD と退避の一文を理由に足す。その一文を小さな関数にすれば、2 か所が同じ文になる |
| LOW | maintainability | C2-L4: 新しいテストの注釈 2 か所が、cycle を書かずに `docs/reports/cross-review-triage-org-merge-cleanup.md` の finding 1、finding 2 を指す。このファイルは slug ごとに 1 つで、cycle 2 の `/cross-review` が書き直すので、そのあとは同じ番号が cycle 2 の指摘を指す。同じ文に計画の S-X1・S-X2 もあるので、たどれなくなるわけではない | `internal/cli/org_cleanup_test.go:222`、`internal/org/cleanup_test.go:717`。比べる書き方: `internal/cli/migrate_test.go:270`(`AR#1 (cycle 2, docs/reports/cross-review-triage-…)`) | 「cross-review cycle 1 の finding 1」のように cycle を足すか、triage への指し先を消して計画の S-X1・S-X2 だけにする。cycle 2 の `/cross-review` の前に直せば、番号の取り違えが起きない |
| LOW | maintainability | C2-L5: 台帳の 2 行が cycle 2 のコードを含まなくなった。R3-L2 の行は、`git log <head>..<branch>` を名指して fetch を言わない断りを 3 つ(記録がない、記録のブランチが違う、記録のパスか kind が違う)と数える。`checkRegistration` の文(ブランチが残るときの「check that the branch … holds nothing that was not merged」)も同じ形で、4 つ目になった。R3-L3 の行は `internal/cli/org_cleanup_test.go` を「800 行にある」と書くが、S-X1 のテストで 987 行になり、800 行を越えた | `docs/tech-debt/README.md:386`(R3-L2)、`:387`(R3-L3)、`internal/org/cleanup.go:794-797`、`wc -l`(732529d9 は 800、HEAD は 987) | R3-L2 の行に `checkRegistration` を足す。R3-L3 の行を「`internal/cli/org_cleanup_test.go` も越えた」に直す(R2-L8 と同じ理由で、数は書かない) |
| LOW | readability | C2-L6: 登録が別のブランチにいるときの断りは、HEAD の退避のコマンド(`git -C <root> branch <branch>-unmerged-<sha7> <HEAD>`)を名指し、登録を外すと HEAD と reflog が消え、そこで作ったコミットへの最後の参照かもしれない、と言う。この場合の HEAD はそのブランチ(テストでは `other`)の先頭で、`git worktree remove` も Manual の script(消すのは binding のブランチだけ)もそのブランチを残す。HEAD のコミットは消えず、消えるのは登録の reflog だけ。文のとおりに打つと、`other` と同じコミットを指す `feat/auth-unmerged-<sha7>` が増える。害はないが、何が危ないのかを読み違えさせる | `internal/org/cleanup.go:775-776`、`:784-787`、テスト `internal/org/cleanup_test.go:746-749`(`other` の場合も退避を打つ) | 別のブランチの分かれでは「HEAD は `<そのブランチ>` が持ち、片付けのあとも残る。消えるのは登録の reflog だけ」と言い、退避のコマンドは detached のときだけ出す |

### cross-review cycle 1 の指摘の扱い(HEAD 33b78f8c で確かめた)

| ID | 扱い | 状態 |
| --- | --- | --- |
| S-X1(指摘 1) | `ReadMergeOrigin` を merge と cleanup で共有し、cleanup は `MergeOrderParams.Origin` で `gh pr list -R <host>/<owner>/<repo>` に固定する。PR の url を host と owner/repo で比べ、違えば全ブランチの PR を使わない。origin を読めなければ何も消さずに exit 1。テストは `-R` の付いた gh の呼び出し、別の owner と別の host の PR、origin の 3 つの断りを見る | 解消。repository の名前が変わったときの次の手は C2-L2 |
| S-X2(指摘 2) | 登録だけが残るときに `checkRegistration` を確かめ 5 より先に通し、detached、別のブランチ、ブランチなし、HEAD が読めない登録を断る。Manual は登録の外しと script の片付け | 解消。記録がないときの Manual は C2-L3、文の細部は C2-L1 と C2-L6 |

### Positive notes(cycle 2)

- `ReadMergeOrigin` の切り出しで、merge の断りの文を変えずに cleanup と同じ読み方を使っている。テストは URL の 10 の形と、失敗、時間切れ、割り込み、既定の 15 秒を見る。
- S-X2 のテストは修正前のコードで落ちる作りで、断りのあとに何が残るかと、名指したコマンドを文のとおり打った結果の両方を見る。
- HEAD がゼロの登録は git の実際の出力(`HEAD 0000…`)で分け、打っても失敗する退避のコマンドを出さない。
- `PRLookupWhy` を JSON から外し、merge-order の出力の形を変えずに cleanup の行だけに理由を足している。

### Tech debt identified(cycle 2)

この run の変更はこの report と insight event だけで、台帳は書き換えていない。cycle の上限(既定 2)に届いた run なので、C2-L1〜C2-L6 のうち直さないものは、`/pr` のあとに運ぶ場所が台帳しかない。直さずに進めるなら、C2-L5 の 2 行の直しとは別に、次の 1 行を足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| merge のあとの片付け(#263)の cross-review の直しの残り(self-review cycle 2)。(a) HEAD がゼロの登録の断りが名指す `git -C <root> rev-parse --git-common-dir` は `.git` と相対で出し、`worktrees/` の下のディレクトリ名も言わない(C2-L1)。(b) 固定した読み方で origin と違う repository の PR が返ったときの行は「gh が読めたら打ち直す」とだけ言い、repository の名前が変わった場合の手を言わない(C2-L2)。(c) 記録がないときの断りの Manual は、中身の消えた worktree の登録が detached でも `git worktree remove` を名指し、理由の文は HEAD に触れない(C2-L3)。(d) 登録が別のブランチにいるときの断りは、そのブランチが残る HEAD の退避を名指す(C2-L6) | (a) 人が別の `.git` を探す。(b) 名前が変わった repository では片付けが止まったままになる。何も消えない。(c) 記録を手か gc で消したあとに人が Manual を打つと、detached のコミットがどの参照からもたどれなくなる。(d) 同じコミットを指すブランチが 1 つ増える | cycle の上限に届いた run の LOW だけで、どれも ralph が自分で消す道ではない | 次に `internal/org/cleanup.go` の断りの文か `foreignPRWhy` を変えるとき。(c) は先に、断りの一文を `checkRegistration` と `recordGone` で共有する | `docs/plans/active/2026-10-28-org-merge-cleanup.md`、この report の C2-L1〜C2-L6 |

### Recommendation(cycle 2)

- Merge: 可(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 6。HEAD 33b78f8c について)。cross-review cycle 1 の S-X1・S-X2 は解消した。
- Follow-ups:
  - C2-L4 は注釈の 2 行なので、cycle 2 の `/cross-review` が triage を書き直す前に直すと、番号の取り違えが起きない。
  - C2-L3 は、人が文のとおりに打つと merge されていないコミットが外れる道で、S-X2 と同じ種類。直すなら `recordGone` の理由に一文を足すだけで済む。直さないなら上の台帳の行に入れる。
  - C2-L5 は台帳の 2 行の文の直しで、orchestrator が台帳を書くときに一緒に直せる。
  - `/verify` と `/test` は、この run が打たなかった `go test`・`go vet`・`golangci-lint`・`run-static-verify.sh`・`run-test.sh` を打つ。
- Known gaps(所見にしていないもの):
  - 登録が binding のブランチにいれば、前に detached で作ってからブランチに戻ったコミットが登録の reflog にだけ残っていても、片付けは進む。worktree が残る場合の確かめ 6 も今の HEAD だけを見るので、同じ作り。
  - ssh の alias の origin は、merge と同じく gh が host を読めず、ブランチごとに SKIPPED になる(決定済み、PR の既知の穴)。
  - cleanup は origin、ls-remote、gh を打った場所で読み、消すのは state dir から引いた main worktree の下。`--state-dir` で別の repo を指すと、読む origin は打った場所のもの。cycle 1 からの作り(Dir を空のまま渡す)で、違う repo の PR なら確かめ 5 の head の比べで止まる。
  - lock された登録は、`remove` の `git worktree remove` も Manual も `-f -f` なしでは外せない。cycle 1 からの作り。
