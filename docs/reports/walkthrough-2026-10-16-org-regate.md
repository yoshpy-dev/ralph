# Walkthrough: org-regate(8 段目の前半の 3 本目の後半、8a-3b)

- Date: 2026-10-17(計画の日付は 2026-10-16)
- Plan: docs/plans/active/2026-10-16-org-regate.md(/pr で archive に移す)
- Branch: feat/org-regate(base 670ec219。この walkthrough を書いた時点で 72 commits、47 files、+8969/-524)
- PR: #234
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 39 files、+8203/-524 になる。そのうちテストが `internal/org/regate_merge_test.go`(+1991)、`internal/org/regate_test.go`(+1508)、`internal/cli/org_regate_test.go`(+758)、`internal/org/mergeorder_test.go`(+442)、`internal/org/spawn_test.go`(+213)ほか。ほかに `/org` と `/director` の skill の 4 面ずつ、README、`AGENTS.md`、仕様 FR-7 の注、`docs/quality/quality-gates.md` の 2 面、tech-debt がある
- 後工程は 3 回まわした。cross-review の 1 回目と 2 回目がそれぞれ ACTION_REQUIRED を 2 件出し、2 回目の直しのために上限を 2 から 3 に上げた(下の 10)

新しいファイルは `internal/org/regate.go`、`regate_git.go`、`regate_merge.go`、`regate_inway.go`、`gate_record.go`(`gate.go` から移しただけ)、`internal/cli/org_regate.go` と、それぞれのテスト。次の順に読むと、regate が 1 回で通る順に追える。

## 1. 入口と最初の確かめ(`internal/org/regate.go`)

- `(*Org).Regate` が全体の順を持つ。ロック、最初の確かめ、worktree の確かめ、fetch、origin の先頭の確かめ、取り込み、gate、記録、push の順に進む
- 台帳ごとの `regate.lock`(`regateLockPathIn`)を flock の排他で取る。取れなければ 1 秒まで打ち直し、それでも取れなければ断る(D7)。取った直後に印を空にし、manifest のロックの下で org_id、branch、worktree、pid の印を書く(`writeRegateMarker`、D9)
- `regateFirstCheck` は、merge-order と同じ `readMergeLedger` と `collectMergeTargets` で台帳を読み、`readRegateBranch` でブランチを名指す最後の完全な結びつきと最新の gate の記録を取る(D2)。その org が `RunningOrgs` に入っていれば断る。読めない gate の記録は、その org のすべてのブランチに付く(cross-review cycle 1 の P2)
- `regateCheckWorktree` は gate と同じ確かめ(worktree、ブランチ、きれいな作業ツリー、スクリプト)を `gate.go` の関数で行う。gate の関数には、誤りの接頭辞と文面を変える `gateWording` を渡すようにした(`regateWords`)。gate の振る舞いは変えていない
- `regateCheckInProgress` は merge と rebase の途中を断る。rebase の途中は `git rebase --abort` だけを案内する(終えた rebase は force push が要るため)
- 何もしない道: 最新の記録が今の既定のブランチの先頭の上の HEAD の `gate_passed`(`passedOn`)で、HEAD が origin のブランチと同じなら、`UP_TO_DATE` で終わる(AC3)

## 2. fetch と origin の確かめ(`internal/org/regate_git.go`)

- `regateFetch` は既定のブランチと `<b>` を、明示の refspec で 2 回に分けて取る(D8)。`<b>` が origin にない(`couldn't find remote ref` を `LC_ALL=C` で読む)ことを、既定のブランチの失敗と分けて言う
- remote を叩く git は `regateRemoteGit` と `regateBoundedGit` を通し、`GIT_TERMINAL_PROMPT=0` と 1 回 120 秒の上限を付ける。fetch と push は ssh の鍵の入力を止めないよう ralph のプロセスグループに残す
- `regateCheckOriginHead` は、HEAD が `refs/remotes/origin/<b>` と違えば断る(D4)。push していない作業を巻き込まず、誰かが先に push した先を上書きしないため。例外は 6 の push だけの打ち直し

## 3. merge の前の確かめ(`internal/org/regate_inway.go`)

- git の merge は ignored のファイルを黙って上書きし、`--no-overwrite-ignore` は `--no-ff` の merge に効かない(git 2.49 で確かめた)。そこで merge の前に、merge が書くパスを `regateMergeWrites` で取る(D3、cross-review cycle 1 の P1)
- `git merge-tree --write-tree` が使えるときは、merge と同じ env と設定で作った木の、HEAD に対して足されたパスを取る。directory rename の移った先も入る(self-review cycle 2 の C2-1)。`--write-tree` を知らない git(2.38 より前)では、2 つの `git diff` の積に落とす
- `regatePathInTheWay` はパスごとに、ディスクの上に追跡していないものがあるかを見る。大文字小文字を区別しない FS では `regateSpellingOnDisk` でディスク上の綴りを読み、追跡済みのファイルを ignored と取り違えない(verify cycle 2 の V2-1)
- ぶつかるパスがあれば、何も merge も記録もせずに断り、最大 10 個を並べる

## 4. merge と戻し(`internal/org/regate_merge.go` の前半)

- `regateMerged` が取り込みの道を持つ。`regateCheckIdentity` で user.name と user.email を先に確かめ、`regateMergeTip` で `git merge --no-ff --no-edit -m 'Merge <base_ref の短い名前> <base の 12 桁> into <b>'` を打つ(件名は `regateMergeSubject`)
- merge、abort、戻し、`git var` は別のプロセスグループで動かす(`regateUncancelledGit`)。Ctrl-C や端末の hangup が merge の途中に届いても、merge を終えてから止まる。その間は stderr に 1 行出す
- 衝突したら `regateAbortLeftMerge` で merge を戻し、ぶつかったファイルを並べて断る。人は worktree で `git merge` を自分で打って直し、push してから regate を打ち直す
- 戻し(D5、`regateUndoMerge`): `gate_passed` を記録する前に、スクリプトが落ちたとき、中断されたとき、結びつきや org が変わったときは、`git reset --keep <merge の前の HEAD>` で戻す。HEAD がまだ regate の merge commit で、作業ツリーがきれいなときだけ戻す。`head_moved` と `tree_changed` では戻さず、戻すコマンドを出す

## 5. gate と記録(`regate.go` の後半、`gate_record.go`、`gate.go`、`gate_run.go`)

- `regateGate` は `gate_run.go` の `runGateScripts` でスクリプトを走らせる。ログの見出しは `# ralph org regate` になる(`createGateLog` に `gateWording` を渡す)
- `recordRegate` は manifest のロックの下で読み直し、ブランチの最後の完全な結びつきが同じで、その org が走っていないときだけ書く(D6)。`recordGate` の life の確かめは走っている org 用のまま残した
- 記録は gate と同じ形に `via=regate` を足す。`gateRecordFromDetails` は ` log=` のあとを丸ごと log の値として読むので、`via` は `log` より前に置く。記録の型と読み書きは `gate_record.go` に移した(中身は変えていない)
- gate だけの道(AC9): HEAD が既定のブランチの先頭を含んでいれば、merge も push もせずに gate を走らせて記録する。人が衝突を直して push したあとの取り直しがこの道を通る

## 6. push と打ち直し(`regate_merge.go` の後半)

- `regatePush` は記録のあと force なしで push する。記録のあとの失敗、120 秒の超過、中断は、記録と merge commit を残して終了コード 1 で終わる(origin が受け取ったあとに応答が届かなかった場合に、戻すと origin と worktree が食い違うため)
- `regateHeadNotOnOrigin` が push だけの打ち直しを決める(D4)。最新の記録が読める `gate_passed` で、head が HEAD、base が今の既定のブランチの先頭なら、次の 2 つで分ける
  - regate が書いた記録(`via=regate`): `regateIsOwnMerge` が、HEAD の親がちょうど `[origin/<b> の先頭, 記録の base]` で、件名が `regateMergeSubject` の値のときだけ push を許す。gate だけの記録の head は gate の時点で origin にあったので、origin がその親に戻されたあとに push し直すと人が消した commit を戻すため(cross-review cycle 2 の P2-1)。件名まで見るのは、人が既定のブランチを merge した commit が親だけなら同じ形になるため(verify cycle 3 の V3-1)。件名は `log.showSignature` の署名の行を避けて `--no-show-signature` で読む
  - leader の記録(`via` なし): `origin/<b>` が HEAD の祖先なら push する
- 例外に当たらずに断るとき、`regateResetHint` は、最新の記録が regate の書いたもので、head が HEAD で、HEAD が同じ merge commit の形のときだけ、`git -C <worktree> reset --keep refs/remotes/origin/<b>` のあとに打ち直す手を出す。それ以外では人の作業を消しうるので手を出さない
- push のあとの `refs/remotes/origin/<b>` は、`regateSetTrackingRef` が push の前に読んだ値から動いていないときだけ HEAD に向ける。CAS が落ちたら読み直し、動いていれば残す(cross-review cycle 2 の P2-2)

## 7. Spawn と start との排他(`internal/org/spawn.go`、`feature.go`、`lockfile.go`)

- `regateSpawnErr` は Spawn の `withManifestLock` の中で、冪等な打ち直しの判定のあとに `regate.lock` の共有を待たずに試す。取れれば regate は走っていないのですぐ放す。取れなければ印を読み、印の org_id が自分と同じときだけ断る(D9)
- `--dry-run` の spawn は新しい life を作らないので断らない。`ralph org start --plan` は ensure の前にも同じ確かめをする
- 待たない共有の flock(`fileLockHeld`)は、見張りの確かめ(`ProbeWatch`)がすでに使っているもの。`lockfile.go` は、regate.lock も扱うことを注に足しただけ

## 8. merge-order(`internal/org/mergeorder.go`)

- 結びつきの org が走っていないブランチでは、`rebase` と `regate` の理由の欄に `ralph org gate --org-id` の代わりに `ralph org regate --branch <b>` を出す(`mergeRegateCommand`、`mergeRegateNext`、AC10)
- regate が書いた `gate_failed` の Detail は `mergeRegateFailedDetail` が書く。JSON には `via` を足した
- 読めない gate の記録は、その org のすべてのブランチに付け、理由の欄はその記録がほかのブランチのものかもしれないと言う(D2)

## 9. CLI(`internal/cli/org_regate.go` ほか)

- `ralph org regate --branch <b>` は `--org-id` を断る。台帳は他の動詞と同じ規則で決め、台帳を書く動詞として扱う。新しい作業を始めないので、見張りも承認された計画も要らない(D1)
- 出力は `UP_TO_DATE:`、`GATE:`、`MERGED:`、`PUSH_ONLY:`、`HEAD:`、`BASE:`、`LOG:`、`PUSHED:` の行。終了コードは、UP_TO_DATE と通った push が 0、断りと失敗が 1
- help は、push だけの打ち直しの条件と、hook が件名を変えた merge commit を断ることまで書く。`TestOrgRegate_Help` がその文を押さえる
- `org.go`、`org_gate.go`、`org_merge_order.go`、`org_legacy_ledger.go` は、動詞の一覧、gate の help の regate の注、merge-order の help、古い台帳の注に regate を足しただけ

## 10. 後工程の経過

- cycle 1: self-review の M1(merge と戻しを別のプロセスグループで動かす)と L1〜L9 を直した。cross-review は ACTION_REQUIRED 2 件。P1 は merge が ignored のファイルを上書きする件で、merge の前の確かめ(D3)を足した。P2 は org_id を立て直した org で、読めない gate の記録が別のブランチに付く件で、org のすべてのブランチに付けるようにした
- cycle 2: self-review の C2-1(directory rename の移った先)で、D3 の候補を `git merge-tree --write-tree` の木から取る形にした。verify の V2-1(綴りだけ違う追跡済みのファイル)を直した。cross-review は ACTION_REQUIRED 2 件。P2-1 は gate だけの regate のあとに `reset --keep` の手と push し直しが出る件で、regate 自身の merge commit の形で絞った。P2-2 は push のあとの tracking ref の巻き戻しで、push の前の値とだけ比べるようにした
- cycle 3(上限を 3 に上げた回し直し): self-review の C3-1〜C3-4(help の文、注、`regate_inway.go` と `gate_record.go` への移動)を直した。verify の V3-1(人の merge commit を regate のものと読む)は件名の確かめで直し、計画の D4 と AC7 を書き直して承認し直した。V3-5〜V3-7(watch.go の注、help の文、署名のテスト)も直した。cross-review は指摘 0 件
- self-review の cycle 3 のあとに入ったコードの 9 commits(82cf0d39〜3a452a71)は、self-review をやり直していない。verify の 2 回目、test、cross-review で確かめた
- tech-debt に回したもの: push だけの打ち直しが、push の失敗と force push で戻された場合を見分けない件(C3-5)、D3 の保守的な断り 2 件と綴りだけ違うディレクトリでの遅さ、`branch.<name>.mergeOptions`、2.38 より前の git の代わりの道、大文字小文字の 6 行が CI で skip されること、テストのファイルの長さ、`newTestWatchRun` の一時ディレクトリ(`docs/tech-debt/README.md` の 275 行と 279〜289 行)
