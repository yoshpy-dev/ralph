# Self-review report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(D1〜D6、AC1〜AC6 と AC3b、Progress checklist の S1〜S5 と「self-review cycle 1」の F1〜F3 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1。run 1 は実装の review、run 2 は run 1 の所見の直しの review)
- Scope:
  - run 2: `git diff 97ea508b..HEAD`(HEAD e316d081、22 ファイル、+666/-239)。コミットは 5442c872(計画の D2・D3・AC3 と台帳の 2 行)、a3af6114(F1: M2・L1・L2)、cbaa92f4(F2: M1・L3・L4 と skill・help・README・仕様の文)、794aee4d(F2b: CLI の文の中のブランチの quote)、e316d081(計画の記録)。
  - run 1: `git diff 15e301c8...c55819e9`(44 ファイル、+4684/-263)。
  - どちらも、命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。テストと静的解析は打っていない。

この report の verdict(下の「Findings」の先頭の表と「Recommendation」の `Merge:` の行)は HEAD e316d081 のブランチについてのもの。run 1 の所見は、その下に run 1 の時点の文のまま残し、扱いを別の表にした。run 2 の所見の ID は run 1 の L1〜L6 と取り違えないよう `R2-L<n>` にした。

## Evidence reviewed

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

### HEAD の所見(run 2)

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

run 1 で挙げた 3 行(L5 と L6 の 2 件)は 5442c872 で台帳に入った(`docs/tech-debt/README.md:383-384`)。383 行の行数は R2-L8 のとおり HEAD と合わない。

この run のコミットはレポートと insight event に限るので、台帳には書いていない。R2-L1〜R2-L8 を直さずに進めるなら、次の 1 行にまとめて足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| merge のあとの片付け(#263)の文の残り。確かめ 2 の記録のパスか kind の違いの断りが `--force-branch` を渡すのに `git log` の確かめを言わない(R2-L1)、確かめ 5 の戻し方が断った時点の worktree の場所に依る(R2-L2)、PR の head が手元にないと確かめ 5 の文のコマンドが動かない(R2-L3)、`regate.go` の `cleanup --branch` が素(R2-L4)、quote の関数が 2 つ(R2-L5)、`Failed` の 2 つ目の出どころが skill と README にない(R2-L6)、`/director` の手順 3 が混じった実行の `CLEANED:` を報告しない(R2-L7)、L5 の行の行数(R2-L8) | R2-L1 と R2-L2 は、人が文のとおりに打つと merge されていないコミットを失いうる。ほかは文のとおりに打っても進めないか、表記がそろわないだけ | この run は報告だけで、直しは orchestrator が決める | 次に `internal/org/cleanup.go` の確かめ 2 か 5 の文、または `/director` skill の後始末の節を変えるとき | `docs/plans/active/2026-10-28-org-merge-cleanup.md`、この report の R2-L1〜R2-L8 |

## Recommendation

- Merge: 可(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8。HEAD e316d081 について)。run 1 の M1・M2・L1〜L4 は解消、L5・L6 は台帳に繰り越した。
- Follow-ups:
  - R2-L1 と R2-L2 は、人が文のとおりに打つと push していないコミットを失いうる道で、M2 と同じ種類。どちらも `internal/org/cleanup.go` の文の直しで済むので、R2-L3 と合わせて 1 つの slice で直すことを勧める。R2-L2 を `switch -C` で直すなら、確かめ 5 の断りの表(`internal/org/cleanup_test.go` の `TestCleanupFeatureWorktree_RefusedInTheWorktree` と `TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten`)の期待する文が変わる。
  - R2-L4 と R2-L5 は quote の 1 行と注釈の直し。R2-L6 と R2-L7 は skill の文で、4 つの写しと固定の断片(`internal/org/director_skill_sync_test.go`、`internal/org/org_skill_send_sync_test.go`)も合わせる。
  - R2-L8 は台帳の 1 行の直し。直さずに進めるなら、上の 1 行を台帳に足す。
