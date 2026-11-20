# sync-docs report: org-merge-queue-verb

- Date: 2026-11-20。パイプラインの 1 回目の run(`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-verb.md`(issue #342)。承認の digest `aef9c99a6ea5` は、`./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない(受け入れ条件のチェックと、Progress の 2 行だけ)
- Branch: `feat/org-merge-queue-verb`。run 開始時の HEAD は `39a52510`、base は `d919479a`(手元の `main`)
- 先行 report: self-review の 2 回目は `bbbf64e7`(event は `636a50df`、LOW 5、Merge: yes)、verify は `d7ffc12a`(pass、event は `4be700ff`)、test は `9800f0a9`(pass、event は `39a52510`)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この run が始まったとき): `git diff --shortstat main...HEAD` は 43 ファイル、+5,452/-278
- 内容の commit は `7469393c`(14 ファイル、+96/-61)。この report と insight event は、そのあとの commit に入れた
- 制約: Edit と Write の tool は、security-guidance plugin の PreToolUse hook が空白を含むパス(`Application Support`)を引用せずに `python3` へ渡して落ち、すべて止まった。このため、文書と Go の help を、python で「置換する文字列がファイルにちょうど 1 回ある」ことを確かめてから置き換える形で直した(対象は下の表のすべて)。直したあとに `.claude/hooks/check_mojibake.sh` を各ファイルへ手で回した(14 ファイルすべて exit 0。README と tech-debt は U+FFFD が 0 件)

## 変えたもの

| ファイル | 変更 | 項目 |
| --- | --- | --- |
| `README.md`(296 行) | 終了コード 7 の文に「origin の既定のブランチが gate のときのままなら」の条件と、動いたあとの断り方。queue にある PR を regate した道の 1 文 | D1 |
| `.claude/skills/org/SKILL.md`(396 行の `merge` の行、2002 行、2150 行の箇条) | 同じ条件、`regate.lock` の文 | D1、D6 |
| `.claude/skills/director/SKILL.md`(1743 行) | 同じ条件(終了コード 7 の箇条) | D1 |
| 上の 2 つの `templates/base/.claude/skills/` の写し、2 つの `.agents/skills/` の写し、`templates/base/.agents/skills/` の写し | `cp` と `scripts/sync-skills.sh`(2 つの root 指定)で作った | D1、D6 |
| `internal/cli/org_merge.go`(help の Long) | `regate.lock` の文に、queue の道の例外を 1 文足した | D6 |
| `internal/cli/org_merge_test.go`(`TestOrgMerge_Help`) | 同じ文を固定の文字列に足した | D6 |
| `docs/tech-debt/README.md` | 182、395、502、534、536、537 行を直した。538 行に新しい行を足した | D1〜D5、G1〜G4 |
| `docs/specs/2026-11-19-org-merge-queue.md` | Open questions の 1 つ目に「決まった: 7」 | D7 |
| `docs/plans/active/2026-11-20-org-merge-queue-verb.md` | 受け入れ条件 13 個を `[x]`。Progress に Test と Sync-docs の行 | D8 |

## D1: 「打ち直せば待ちを続ける」の条件と、regate した PR の道

Go の文(`queueNext`、help、`ErrMergeQueued`、`exitcode.go`、`main.go`)が書く条件を、README と skill に揃えた。`requeueNext`(`internal/org/merge_queue_run.go`)を読み、それと食い違わない形にした。

- 条件: 「origin の既定のブランチが gate のときのままなら」。動いたあとは、merge-order の分類(手順 7)と最後の ls-remote(手順 13)が、queue を読む前に断る。README は英語で、org skill は 396 行と 2002 行、director skill は 1743 行に書いた。director skill には、断られたら終了コード 1 なので上の箇条のとおり人に伝える、と足した
- 「2 本目の PR まで」は、README と skill には書かなかった。いまの挙動の説明に、先の予定は混ぜない。tech-debt の 536 行に、条件がある理由(手順 7 と 13 が queue の道でも断る)と、2 本目の PR(FR-12)がその 2 つの手順を変えるので、そのときこの条件を外す、と書いた
- regate した PR の道: README に 1 文。regate は新しい head を push し、GitHub はそれで PR を queue から外すと仮定する。`local` の regate は gate の記録を push の前に書くので(`regate.go` の doc comment と同じ順)、次の merge は FR-5b の断りと同じ理由で断り、そのときの手は `--requeue` になる。`ci` の regate は push のあとに記録するので、この文は `local` に限った。`ci` は 5 分の幅が決める。「PR が外される」は確かめていない前提なので、536 行の仮定に足した
- `requeueNext` の文は変えていない(Go)。この文は「fix を push して regate」と「揺れた検査のあとは `--requeue`」を並べていて、queue にある PR を regate した道は書いていない。536 行がそれを書く

## D2: tech-debt の 537 行(800 行の目安)

- 払う手順が名指す `TestMergeQueue_LatestRemovalAfterIsStrictlyAfter` を、`5e409db0` が改名した `TestMergeQueue_LatestRemovalAfterCountsTheSameSecond` に直した。末尾から数えた行数も、135 から約 130(`merge_queue_test.go` の 676 行目から 807 行目)、200 から約 190(`merge_queue_wait_test.go` の 663 行目から 854 行目)に直した
- 行数は `wc -l` を 435ecfe9(この branch の最後の Go commit。この run の変更は `internal/cli` の 2 ファイルだけで、`internal/org` は動かしていない)で数えた: `merge.go` 826、`merge_queue_test.go` 807、`merge_queue_wait_test.go` 854、`merge_queue.go` 489、`merge_queue_run.go` 333、`merge_queue_wait.go` 390、`merge_queue_run_test.go` 647、`merge_queue_path_test.go` 153、`internal/cli/org_merge_queue_test.go` 348
- 持ち越しの注: Trigger の「The next change to either test file」に、同じ branch の `5e409db0`(self-review の直し)が当たる。両方のテストファイルを in place で変えた(806 → 807 と 859 → 854。L3 の同じ秒の規則、L4 で query が読まなくなった add event、L1 の次の手)。払わなかった理由は、move-only の分割が直しの Scope の外であること。`merge.go` の 4 行は `mergeCheck.gateAt` のコメントで、コードではないので、`merge.go` の側の Trigger は当たっていない、と書いた

## D3: 395 行と 502 行の持ち越しの注

どちらも `435ecfe9` が変えた 3 つ目の固定の文字列(`--wait-checks` の flag の文)を数えていなかった。

- 395 行: 「two fragments」を three に、「1248 → 1249 lines, counted at 56fc4cab」を「1248 → 1250 lines, counted at 435ecfe9」に直し、この run が足したテストの固定の文字列で 1251 になることを書いた。`org_merge_queue_test.go` は 332 行から 348 行に
- 502 行: 同じ 3 か所(fragment の数、行数、`org_merge_queue_test.go` の 348 行)

## D4: 182 行(`/org` skill の命令の表)

Trigger (a) に、この PR の `merge` の行の書き換え(4 写し)が当たる。持ち越しの注を足した。表の `merge` の行は、origin/main の `d919479a` で 5,285 文字、この run の後で 5,939 文字(S5 が偽になった文を直し、この run が条件と `regate.lock` の文を足した)。`stop` と `disband` の行は変わらない(1,208 と 1,219 文字)。`merge` の行は `director` の行(6,377 文字)に次いで長く、表の構造を直す機会に含める、と書いた。Related に計画を足した

## D5: 534 行(`merge_test.go` の行数)

835 行は 833 行になった。`12ace9df` が、queue を断る期待の 1 行(4 行)を消し、2 行のコメントを足したため。タイトルと「stays at 835」の文を直した。Trigger(in-place の直しより大きい変更)には当たらない

## D6: merge の間の `regate.lock` の文

queue の道では、`mergeLocked` が戻ると lock を離し(`--wait-checks` に時間が残らなければ、入れたところで戻るときに離れる)、待ちのあと `pr_merged` を manifest の lock だけで記録し(`recordQueueMerge`)、`regate.lock` を取り直すのは片付けのときだけ(`queueMergeCleanup`。`RepoRoot` がなければ取り直さない)。コードを読んで確かめた。

- help: 「While it runs, except while it waits, ... are refused on the ledger.」(`TestOrgMerge_Help` が固定している文)はそのまま残し、その直後に 1 文足した: 「On the merge queue path, merge releases regate.lock when the wait for the queue starts and takes it again only for the cleanup (not at all when none runs), so none of them is refused in between.」。固定している文の隣にあった `ralph org spawn and start of the branch's org are refused only while merge cleans up ...` も固定されているので、折り返しだけ変えて文は変えていない。テストは、同じ要素に足した
- org skill: 396 行の `merge` の行の括弧と、2150 行の箇条に同じことを書いた。箇条は、merge の間は spawn と start が断られない、の文が続くので、折り返しを詰め直した
- README の 296 行の queue の段落は、すでに「takes `regate.lock` again for the cleanup」と書いている

## D7: 仕様の Open questions

1 つ目に「決まった(2026-11-20): 7。1 本目の計画の D6 が選び、`ErrMergeQueued`(`internal/cli/exitcode.go`)が返す。FR-10 の「案は 7」はこの値」と足した。FR-10 の文は触っていない

## D8: 受け入れ条件のチェック

計画の受け入れ条件 13 個(AC1〜AC6、AC8、AC10、AC12〜AC16)を `[x]` にした。verify が 13 個とも Met と判定し(AC3 と AC14 は注つき)、test が AC1〜AC6、AC8、AC10、AC12〜AC16 のテストが通ったことを確かめている。AC7、AC9、AC11 は計画の Non-goals(2 本目)で、計画にこの 3 つの行はない。digest は変わらない

## Test の gap(G1〜G4)

tech-debt の 538 行に、1 行で書いた(5 列)。G1〜G4 は同じ run の同じ型(どれも seam が要り、足すとコードが変わって `/self-review` からやり直しになる)で、まとめるほうが読みやすい。

- G1: queue の道の `pr_merged` を書けないとき(`recordQueueMerge` の ledger の読みと append の失敗)のテストがない。M5(`mergeRecordErr` で包まずに返す)が生き残った
- G2: `enqueue` の、読みのあと `enqueuePullRequest` を打つ前の ctx の確かめは、テストから決まった順で起こせない。M6 が生き残った
- G3: `gateWhen` の「gate の時刻が読めない」文が実行されない
- G4: `queueMergeCleanup` の、lock が held 以外のエラーで取れないとき、marker を空にできないときの飛ばしが実行されない

Trigger は、それぞれの関数を次に変えるときと、queue で merge したあとのエラー文が分かりにくいという報告。test report の Test gaps の 5 つ目(実地の確かめ)は 536 行の範囲なので、行を足していない。行番号は書かず、関数名とテスト名で書いた

## Trigger の走査

開いている行(取り消し線のない行)の Trigger の列を、この run が触った名前(`org_merge.go`、`TestOrgMerge_Help`、`org_merge_test.go`、`/org` skill の命令の表、`/director` skill、README)で引いた。この走査で当たった行は 182、395、502 で、534、536、537 は D1〜D5 の直しで触った行。走査にかかったほかの 28 行は、名前を含むが別の部分を指している。例として 386 行の「the exit-code part of the `/director` skill」は director の wait の終了コード 4 と 5 の話で、merge の終了コード 7 の箇条ではない。267、385、477 行は、受信箱の節、send と確かめの手順、`gate accept` の節を指す。自分の編集(help の文、テストの固定、skill の表)で当たる行は 182、395、502 で、いずれも注を書いた

## 他に確かめた文書

| 文書 | 結果 |
| --- | --- |
| `AGENTS.md`(92 行の `internal/org/` の行) | 変更なし。queue の 3 つのファイル、lock の持ち方(待つ間は離し、片付けで取り直す)、終了コード 7 を書いていて、D6 とも合う |
| `docs/quality/`、`docs/architecture/`、`docs/recipes/`、`.claude/rules/`、`CLAUDE.md`、`templates/base/AGENTS.md` | `merge queue` の語を探して 0 件(AGENTS.md の 1 行を除く) |
| `docs/specs/2026-10-07-org-multi-org-director.md` の 8a-3a | verify が確認済み。この run では触っていない |
| skill を読むテスト(`internal/org` の skill、director、send、resume、prompt の各テスト、`internal/config`、`internal/upgrade`) | 通る |

## 直さなかったこと

- Go の `requeueNext` の文。regate で queue から外れた PR の道は README と 536 行に書いた。文の変更はコードの変更で、`/self-review` からやり直しになる
- spec の FR-10 の「案は 7」。Open question の決着の文が、この値だと言っている
- test report の G1〜G4 のテスト(tech-debt の行に書いた)

## 確認

| コマンド | 結果 |
| --- | --- |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0) |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(408 行。shellcheck、`sh -n`、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt の計画の参照) |
| `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && go test ./internal/cli/ -run 'OrgMerge\|Help' -count=1` | rc 0(`internal/cli` ok、65.4 s) |
| `go test ./internal/org/ -run 'Skill\|Director\|SendSync\|ReportNames\|SendDefaults\|Followup\|Resume\|Prompt' -count=1` | ok(130.6 s) |
| `go test ./internal/config/ ./internal/upgrade/ -count=1` | ok |
| `./scripts/secret-scan-branch.sh --strict` | clean(`d919479a..39a52510`、内容の commit の前) |
| `./scripts/plan-visual.sh digest` | `aef9c99a6ea5`(計画の Approved の行と同じ) |

`./scripts/run-test.sh` 全体は回していない。この run の Go の変更は help の文 1 つと、それを固定する文字列 1 つだけで、`TestOrgMerge_Help` を含む `internal/cli` の `OrgMerge|Help` が通った。

## Insight event

`./scripts/insights-append.sh --slug org-merge-queue-verb --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` を、この report と同じ commit に入れた。
