# Self-review report: org-merge-cleanup

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-org-merge-cleanup.md`(D1〜D6、AC1〜AC6 と AC3b、Progress checklist の S1〜S5 の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff 15e301c8...HEAD`(HEAD c55819e9、44 ファイル、+4684/-263)。中心は `internal/org/cleanup.go`、`internal/org/merge_run.go`、`internal/cli/org_cleanup.go`、`internal/cli/org_merge.go`。残りは leader の雛形、`/org` と `/director` の skill(4 つの写しずつ)、help、README、AGENTS.md、仕様、recipe、テスト。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。テストと静的解析は打っていない。

## Evidence reviewed

- `internal/org/cleanup.go` の全体と、そこから呼ぶ `acquireHeldFileLock`・`release`・`fileLockHeld`(`internal/org/lockfile.go`)、`writeLockMarker`・`readRegateMarker`・`readRegateBranch`(`internal/org/regate.go`)、`boundedGit`(`internal/org/regate_git.go`)、`samePath`・`resolvedOrClean`(`internal/org/statedir.go`)、`runWorktreeScript`・`featureWorktreeCleanup`・`shellQuote`(`internal/org/feature.go`)、`validateFeatureBinding`(`internal/org/reserve.go`)、`scripts/ralph-worktree.sh` の `cleanup_worktree`・`state_path`・`sanitize_id`。
- 消す相手を取り違えないか。消す操作は script の `cleanup --id org-<org_id> --force-branch` だけで、その前に記録の branch・worktree_path・kind を binding と照らす(`checkRecord`、`cleanup.go:450-474`)。worktree のパスが別の worktree への symlink のときは、`samePath` が両側を解決するので確かめ 2 と 4 は通るが、確かめ 6 の `symbolic-ref` がその先のブランチを読んで断る(`cleanup.go:566-583`)。main worktree を指す symlink も同じく 6 で止まる。登録だけ残った worktree は `git worktree remove <そのパス>` だけで外し、`prune` は使っていない(`cleanup.go:607-619`)。
- git に渡す値。branch は `refs/heads/` の後ろか Manual の中だけで argv の位置引数にならず、worktree は binding の時点で絶対パスに限られる(`reserve.go:174`)。head は `gitObjectPattern` で SHA に限る(`cleanup.go:231`)。gh の `headRefOid` もこの確かめを通る。
- 端末に出す文。Refused・Failed・Manual はどれも `PrintableInboxText` を通り(`cleanup.go:156`、`:209`、`:328`、`:613-624`)、CLI の `CLEANUP:`・`CLEANED:`・`SKIPPED:`・`MANUAL:` の行も通す(`internal/cli/org_merge.go:333-357`、`internal/cli/org_cleanup.go:295-330`)。`ralph org cleanup` がそのまま返す error(`org_cleanup.go:312-313`)は main.go が素で出すが、regate の CLI も同じ形なので所見にしていない。
- `regate.lock` の印。片付けは印を、manifest のロックの下で `RunningOrgs` を確かめたあとに書く(`cleanup.go:254-283`)。lock を放したあとも印はファイルに残るが、これは Regate と同じ作りで(`regate.go:500-503`)、読み手の `regateSpawnErr` は `fileLockHeld` が持たれていると答えたときだけ印を読む(`spawn.go:1473-1486`)。lock を取る側は Regate(`regate.go:317`)、Merge(`merge_run.go:233`)、`CleanupFeatureWorktree`(`cleanup.go:173`)のどれも取った直後に印を空にする。放したあとの古い印が spawn を断る窓は、次の持ち手が取ってから空にするまでの間だけで、前からある窓と同じ。
- `ralph org merge` の終了コード。片付けの結果は `MergeResult.Cleanup` と `CleanupErr` にだけ入り、`mergeLocked` が返す err は `mergeAndRecord` のものだけ(`merge_run.go:258-273`)。`printMergeResult` の分岐は変わっていない。exit 0 は記録した merge のままで、片付けの途中の割り込みが exit 1 になる点は help に書いてある(`org_merge.go:167-175`)。
- skill の 4 つの写しは `cmp` で同一(`/org`、`/director` とも)。recipe の 2 つも同一。
- ファイルの行数。base と HEAD で `wc -l` を比べた(L5)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: script が途中で失敗した片付け(`Failed`)も `CLEANUP: skipped` と出し、`/director` skill はその行を「何も消していない」と読ませる。`Failed` の文は「止まる前に消したものは消えたまま」と言うので、director が人に伝える文が食い違う。`ralph org cleanup` の `SKIPPED:` も同じく、断りと途中の失敗を分けない | `internal/cli/org_merge.go:353-354`(`cmp.Or(c.Refused, c.Failed, ...)` を skipped で出す)、`internal/org/cleanup.go:620-625`(Failed の文)、`.claude/skills/director/SKILL.md:864-865`、その文を固定する `internal/org/director_skill_sync_test.go:852`、`internal/cli/org_merge_test.go:404-406`、`internal/cli/org_cleanup.go:317` | `Failed` は別の語で出す(たとえば `CLEANUP: failed (<理由>)` と `FAILED: <branch> <理由>`)。少なくとも skill の文を「skipped なら消していないか、途中まで消した。理由の文がどちらかを言う」に直し、固定の断片も合わせる |
| MEDIUM | security | M2: 確かめ 3(worktree の中から打った)、4(未コミット・未追跡・rebase か merge の途中)の断りに付く Manual は script の `--force-branch` で、確かめ 5(ブランチの先頭が merge された head か)はまだ走っていない。理由を直してから Manual を打つと、push していないコミットのあるブランチも `git branch -D` で消える。同じ場面で `ralph org cleanup --branch` を打ち直せば確かめ 5 で止まる。help は Manual を「理由を確かめたら打つコマンド」と書く | `internal/org/cleanup.go:482-484`(確かめ 3 が `c.scriptCmd()` を返す)、`:498-505`(4 の次に 5)、`:310-312`(`scriptCmd`)、`internal/org/cleanup_test.go:339`(未追跡の断りの Manual が `fx.scriptCmd()`)、`internal/cli/org_merge.go:131`、`internal/cli/org_cleanup.go:81-82` | ralph が続きから片付けられる断り(3、4、6)では Manual を出さないか、`ralph org cleanup --branch <b>` にする。script の Manual を残すなら、確かめ 5 を 3 と 4 より前に動かし、先頭が head と一致したときだけ出す |
| LOW | security | L1: org_id を台帳から読んだまま使う。`scriptCmd` は org_id を quote せずに Manual に入れ(root と git の引数は `cleanupShellWord` を通すのに、ここだけ素)、`c.id()` は script の `--id` になる。script の `sanitize_id` は大文字を小文字に、記号を `-` に変えるので、書き換えた台帳の org_id `Auth` は org `auth` の記録を指し、`RunningOrgs` は `Auth` で引くので `auth` が走っていても止まらない(probe: `Auth;x` は `auth-x` になる)。`ValidateIdentifier` の注釈は「不正な id がパスを組む箇所に届かない」と言うが、台帳から読む道はその 2 層を通らない | `internal/org/cleanup.go:306`、`:311`、`:620`、`scripts/ralph-worktree.sh:60-73`、`internal/org/identifier.go:40-47` | `cleanupFirstCheck` で `b.orgID` を `identifierPattern` に照らし、合わなければ断る。それで Manual の素の語と、記録の id の取り違えの両方が消える |
| LOW | readability | L2: 断りの文の中のコマンド(`git -C <root> log <head>..<branch>`、`git -C <worktree> switch <branch>`、`git -C <worktree> status`)はパスを quote しない。同じ関数が Manual では `cleanupShellWord` で quote する。repo のパスに空白があると、文のとおりに打ったコマンドは動かない | `internal/org/cleanup.go:437-438`、`:468-469`、`:553-555`、`:557-559`、`:573-575`、`:580-582` | 文の中のパスも `cleanupShellWord` を通す。ブランチ名は空白を持てないのでそのままでよい |
| LOW | readability | L3: main worktree が見つからないときの文に、`FeatureRepoRoot` の error がそのまま入り、`start --plan` の説明と「run it from ...」がもう 1 つの次の手として混ざる(`errWorktreeScriptMissing` の「run start again」は S4 で外したが、こちらは残った) | `internal/cli/org_merge.go:247-253`、`internal/cli/org_cleanup.go:117-121`、`internal/org/statedir.go:136-140` | 原因だけを取り出す(`errors.Is` で 2 つの場合を分けて片付けの言葉で言う)か、`FeatureRepoRoot` の文を動詞によらない言い方にする |
| LOW | maintainability | L4: `/director` skill の「GitHub で merge した PR の後始末」は、手順 1 で「`merge-order --all` に `merged` が出たら打つ。何度打ってもよい」と言い、手順 3 で「終了コード 1 なら打ち直さない」と言う。片付けたあとも `merged` の行は残る(`/org` skill)ので、断られたブランチが残る間は、`--all` を読むたびに手順 1 の条件が成り立ち、手順 3 とぶつかる。どちらが先かが書いていない | `.claude/skills/director/SKILL.md:960-972`、`.claude/skills/org/SKILL.md` の merge-order の節(「片付けたあとも `merged` の行は残る」) | 手順 1 に「前に `SKIPPED:` を伝えたブランチだけが残っているなら、人に頼まれるまで打たない」を足すか、手順 3 の「打ち直さず」の範囲を `--branch` で同じブランチを打つことに限ると書く |
| LOW | maintainability | L5: この PR で 800 行を越えたテストのファイルが 4 つあり、台帳に行がない。`internal/cli/org_merge_test.go` 798 から 1085、`internal/org/merge_run_test.go` 790 から 1077、`internal/org/director_skill_sync_test.go` 795 から 936、新しい `internal/org/cleanup_test.go` 885 | `wc -l` の base と HEAD の比較。`docs/tech-debt/README.md` にこの 4 ファイルの行はない | 片付けのテストを `org_merge_cleanup_test.go`・`merge_cleanup_test.go` などに分けるか、台帳に行を足す(下の表) |
| LOW | maintainability | L6: 計画の Progress に「既知の穴」と書いた 2 件が台帳にない。(a) `runWorktreeScript` は script を時間の上限も ctx もなしで打つので、script か git が止まると `regate.lock` を持ったままになる。(b) `StartFeature` の `regateSpawnErr` は manifest のロックの外で打たれ、片付けの印が書かれる前に通った start は、片付けが消す worktree を `ensure` で使い直しうる(最後の spawn は Spawn 側の確かめで断られる)。Progress は `/pr` で archive され、PR の本文は台帳の代わりにならない | 計画 `:165`、`:190`、`internal/org/feature.go:691-703`、`internal/org/cleanup.go:620`、`internal/org/feature.go:208-218` | 台帳に 2 行足す(下の表) |

## Positive notes

- 消す前の確かめが、消すと失うものの種類ごとに 1 つずつあり、順番と理由が `cleanup.go` の先頭の注釈と関数の注釈で追える。読めない記録、壊れた JSON、`rev-parse --git-path` の行数の食い違いは、どれも消さずに止まる。
- 途中で止まった片付けの続きを、残っているものの組み合わせ(`cleanupState`)で分け、それぞれに要る確かめだけをかける。`git worktree prune` を使わず、そのパスの登録だけを外している。
- `worktree_cleaned` の追記は `context.WithoutCancel` で、消したあとの割り込みで記録が抜けない。追記に失敗したときは `Cleaned` を立てたまま error を返し、CLI は `CLEANUP: done (...)` に error を入れる。
- 確かめの git は ralph の process group で走らせ(`ownGroup=false`)、Ctrl-C が git にも届く。regate の上限つきの git を prefix だけ変えて使い回し、`MergeParams.Exec` の固定の検査とは分けている。
- 印を書くのは manifest のロックの下で、`RunningOrgs` の確かめと同じ区間なので、spawn が確かめの前に来れば org が走っていると断り、あとに来れば印で断られる。

## Tech debt identified

この run のコミットはレポートと insight event に限るので、台帳(`docs/tech-debt/README.md`)には書いていない。L5 と L6 を直さずに進めるなら、次の行を足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `internal/cli/org_merge_test.go`(1085)、`internal/org/merge_run_test.go`(1077)、`internal/org/director_skill_sync_test.go`(936)、`internal/org/cleanup_test.go`(885)が 800 行を越えた。最初の 3 つは base で 800 未満だった | 読むのとレビューに時間がかかる。動きは変わらない | テストの移動は差分を大きくし、片付けの動きの review と混ざる | 次にこの 4 ファイルのどれかを変えるとき。merge の片付けのテストを別ファイルに分ける | `docs/plans/active/2026-10-28-org-merge-cleanup.md`、この report の L5 |
| `runWorktreeScript`(`internal/org/feature.go`)は `ralph-worktree.sh cleanup` を時間の上限も ctx もなしで打ち、`ralph org merge` と `ralph org cleanup` はその間 `regate.lock` を持つ | script か git が止まると、その台帳の regate・merge・cleanup と、片付ける org の spawn と start が、ralph を止めるまで断られる | 中身は `git worktree remove` と `git branch -D` のローカルの操作で、計画は script を変えないことにした(Non-goals) | 止まった片付けの最初の報告か、`runWorktreeScript` を次に変えるとき。`boundedGit` と同じく上限と ctx を付ける | 計画の Progress(S4b の既知の穴)、この report の L6 |
| `StartFeature` の `regateSpawnErr`(`internal/org/feature.go`)は manifest のロックの外で打たれる。片付けの印が書かれる前に通った start は、片付けが消す worktree を `ensure` で使い直しうる | start は最後の spawn で断られるが、それまでに `ensure` した worktree が片付けで消える。Regate にも同じ窓がある | Regate と同じ窓で、計画は直さないことにした(S1 の既知の穴) | 同じ org_id の start と片付けが重なった最初の報告か、`StartFeature` の確かめを次に変えるとき | 計画の Progress(S1 の既知の穴)、この report の L6 |

## Recommendation

- Merge: 可(CRITICAL 0、HIGH 0、MEDIUM 2、LOW 6)。M1 と M2 は merge の前に直すことを勧める。M2 は人が Manual を打ったときに push していないコミットを消しうるので、M1 より先に見る。
- Follow-ups:
  - M2 を直すときは、確かめ 5 を前に動かすか、Manual を `ralph org cleanup --branch` にするかで、`cleanup_test.go` の断りの表の Manual の列が変わる。
  - L1 は `cleanupFirstCheck` の 1 行の確かめで済む。regate の案内(`regateCheckWorktree`)も同じ org_id を素で使うが、消す操作はないので、この PR の範囲では見ていない。
  - `/verify` 向け: 仕様の FR-8 の注と計画の D2 の 3 は、確かめ 3 の理由を「script は中から打つとブランチを残して記録だけ消すため」と書く。ralph は script を main worktree で打ち(`feature.go:697` の `cmd.Dir = repoRoot`)、Manual も `cd <root> &&` で始まるので、この理由はどちらにも当たらない。文書の食い違いなので `/verify` の範囲として残す。
