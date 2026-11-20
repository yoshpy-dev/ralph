# sync-docs report: org-merge-queue-order

- Date: 2026-11-20(Run 1。パイプラインの 1 回目で、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-20-org-merge-queue-order.md`(issue #342 の 2 本目)。承認の digest `357148b1a947` は、この run の前も後も `./scripts/plan-visual.sh digest` の値と一致する。計画の本文は変えていない。変えたのは Progress の行と AC1〜AC7 のチェックボックスだけで、`git diff --numstat` は、AC の 7 行と Progress の 3 行が置き換わり、Progress の 2 行が増えたことだけを示した
- Spec: `docs/specs/2026-11-19-org-merge-queue.md`(FR-12、FR-13、AC7、AC9 の skill の部分、AC11)
- Branch: `feat/org-merge-queue-order`。run 開始時の HEAD は `b3b4b6ca`、base は `main` の `1d05c9ac`(手元の `main` は `a672b72a`で、merge base のあとに #355 の仕様の文書が 1 つ入っているだけ)
- 先行 report: self-review の run 1 は `1ce09b06`(event は `27723e15`、MEDIUM 1、LOW 8、Merge: yes。直しは `fbdb30b1` と `a019591f`)、verify の run 1 は `ec97fdd8`(pass、event は `8bec7f39`。`d729352e` の固定のテストの記録は `b9ec3ae9`)、test の run 1 は `9726617e`(pass、event は `b3b4b6ca`)。引いた SHA は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report の commit の前): `git diff --shortstat main...HEAD` は 39 ファイル、+3,973/-345
- この run が変えたもの: 文書と、Go の help の文字列 1 か所(`internal/cli/org_merge_order.go`)と doc comment 1 か所(`internal/config/config.go`)。コードの振る舞いは変えていない。`./scripts/run-test.sh` は回していない(依頼どおり)

## この run に渡されたもの

| 項目 | 扱い | commit |
| --- | --- | --- |
| D1: 仕様 NFR-1 の印 | 2 本目の印を足した。queue のない GitHub の repo の出力と呼び出しの並びは 1 本目と同じで、変わるのは merge-order が queue を読めないときの `merge queue: unknown` の行と注記、既存の CLI テスト 7 本の期待出力、`merge_test.go` の base の動いた行の移動、merge-order が毎回 1 つずつ多く打つ読み | `001fd717` |
| D2、D3、D4: tech-debt の Trigger に当たった行 | 下の表。当たったのは 182(a)、247、248、259、317、318、502、535(h) の 8 行。318、502、535 は、verify が「文言の上で当たる(判断による)」とした行で、当たったと読んで注を足した | `e6c81e40` |
| D5: `/org` skill の regate の手順 3 | 「merge queue のない repo では」を足した(4 写し) | `cfed6c98` |
| D6: 「読めない」の理由の列 | merge-order の help、README、`/org` skill の `merge-order` の行に、`git remote get-url origin` の失敗と時間切れを足した。help の文は `org_merge_order_test.go` が pin していない(pin しているのは「or that whether it has one is unknown, and why」まで)ので、テストは変えず、`go test ./internal/cli/ -run OrgMergeOrder` が通ることだけ確かめた | `f2e01fa9` |
| D7: 「queue を読むのは 1 回だけ」 | README と `/org` skill の `merge` の行を「確かめの 1 回の実行ごとに 1 回。`--wait-checks` の待ちのあとの実行は読み直す」にした(4 写し) | `d562d672` |
| D8: AGENTS.md の `mergeorder_queue.go` | 行を組むのは `internal/cli/org_merge_order.go` の `mergeQueueLine` と書き、`mergeorder_queue.go` は理由と注記の文だけにした | `06b74e8f` |
| D9: 311 行と 423 行の行数 | `mergeorder_queue.go` は 154 行(`7b3acbcd`)から 160 行、`mergeorder_queue_test.go` は 807 行から 862 行(どちらも `fbdb30b1` のあと。増えたのは ssh の別名の hint の読みとそのテスト 55 行)。commit に結んで書いた | `e6c81e40` |
| D10: L6、L8、M1 の根 | L6 と L8 は新しい 1 行にした(下)。M1 の根(skill が stderr の文で断りの種類を見分ける)は `internal/org/director_skill_queue_test.go`(`d729352e`)が、見分けに使う断片 14 個と Go の値 5 つを Go の文字列に結びつけるので、行は足さない。残る穴は、断りの種類を typed にして出す口がないことだけで、文が変わればそのテストが落ちて skill を指す | `28b8a789` |
| D11: 2026-10-07 の仕様の FR-7 | 2026-11-20 の追記を FR-7 の最後に足した(取り直しを queue に任せること、`regate` と別のブランチを base にした gate は今までどおり)。FR-8 の 8a-3a の注の 2026-11-20 の文を指す | `f45cb832` |
| D12: FR-12 の印 | step 13 が既定のブランチの名前とブランチの有無も確かめること(計画の D3、安全側の差)を足した | `6f5e1088` |
| D13: 計画のチェックボックス | AC1〜AC7 を、verify の AC 表(Met)と test の AC 表の突き合わせのあとで `[x]` にした。Progress の 3 つの成果物の行と、test と sync-docs の run 1 の行を足した | `b00f7608`、`4f68069f`、`d1c90528` |
| test の Test gaps 1〜3(LOW、INFO) | L6、L8 と同じ 1 行に (c)〜(e) として足した | `28b8a789` |
| test の Test gaps 4、5 | 1 本目の fake gh の行(536 行)を広げた。Debt に 2 本目の読みと手順、Impact に AC7、AC11、AC9 の skill の部分、Trigger に実地の手順、Related に計画と test report | `e6c81e40` |

## 自分の検査で見つけたもの

- `templates/base/ralph.toml` と `internal/config/config.go` の `auto_merge` の comment が「merge のほかに、ほかのブランチを regate する」と書いていた。queue のある repo の director は regate の輪を回さず 2 段の手順で queue に入れるので、「merge queue のない repo では」を付けた(comment だけ。`go test ./internal/config/ ./internal/scaffold/ ./internal/upgrade/` が通過、`check-sync.sh` も通過)。テストが pin している文ではない(`git grep` で 2 か所だけ)。commit は `639043bb`
- この comment の編集が、tech-debt の 420 行(`config.go` を含む 5 つのファイルの 800 行の目安)の Trigger「The next change to any of the five」を満たした。comment の編集も数える先例(同じ行の 2026-11-01 の注)に従い、注を足した(`config.go` は 1194 から 1195 行)。`70cdf181`
- 自分で書いた注の言い間違い 3 か所を、書いたあとに HEAD のコードと突き合わせて直した(259 行の origin を読む回数は最大 4 回、502 行の help のテストが変わった中身、535 行の呼び出し元の数)。`4108c041`

## tech-debt の行

| 行(最初の語) | 当たった Trigger | 足したもの |
| --- | --- | --- |
| 182 「Two documentation findings of org-stop-all」(a) | `/org` skill のコマンド表の次の編集 | `merge-order` の行が 4,705 から 5,761 字、`merge` の行が 5,939 から 6,269 字(4 写し)。`stop` と `disband` のセルは 1,208 と 1,219 字のまま。持ち越し |
| 247 「Without `gh`, `ralph org merge-order`」 | merge-order の gh の重さと、人のいない輪の director | 1 回の実行に get-url(ssh なら `ssh -G` も)と GraphQL が 1 つずつ増えたこと。2 段の手順が 5 分ごとに merge-order を打ち直すこと。遅い実行の報告はなく、行の範囲(ageing out と `gh pr list` の輪)は払わない |
| 248 「`ralph org merge-order` reads the manifest without the lock」 | merge-order を輪で読む director | 2 段の手順の 2 がその輪。誤った `unreadable` で後回しから外れ、理由の欄に従う。`readMergeLedger` は変えていない |
| 259 「Small leftovers of the org-merge-guarantee PR」 | `MergeOrder` の次の編集 | (b) と (d) を広げる(origin の URL を最大 4 回読む)。(c) にも関わる(GraphQL の行が doctor の読まない classic の queue を言う) |
| 311 | 行数(注に既にある) | `mergeorder_queue.go` の行数を `fbdb30b1` のあとの値に直した |
| 317 | merge-order の分類の次の変更 | `classifyGatePassed` と `classifyAccepted` の queue の分岐。`orgRunsOn` は変えていないので払わない |
| 318 | merge-order の class の次の変更 | class は足さず、`ready`、`accepted`、`rebase` の意味が変わった。MERGED の PR の扱いは変えていない |
| 420 「Five files went past the 800-line guide」 | 5 つのうちどれかの次の変更 | 上のとおり。`config.go` の comment、1194 から 1195 行 |
| 423 | 行数(注に既にある) | `mergeorder_queue.go` と `mergeorder_queue_test.go` の行数を直した |
| 502 「Test gap of feat/gate-tool-versions」 | cli の gate、regate、merge のテストの次の変更 | `org_merge_queue_test.go`(348 から 362 行)と `org_merge_order_test.go`(1403 から 1671 行)。gate も probe も回さないので払わない |
| 535 「Small leftovers of fix/org-ssh-alias-host」(h) | `mergeSSHAliasHint` の次の変更 | 呼び出し元が増えたこと(`readMergeOrderQueue`、`fbdb30b1`)と doc の一覧(`a019591f`)。新しい呼び出しも小文字の host を出す。そのテストは小文字の host だけ |
| 536 「The merge queue path of `ralph org merge`」 | 実地の確かめ | Debt に、2 本目の読み(merge-order の GraphQL、ruleset だけの queue を「ない」と読む D4 の前提)、base の動いた PR を queue が最新の既定のブランチの上で検査して merge すること、director の 2 段の手順を足した。Impact、Trigger、Related も合わせた |
| 新しい 1 行「Small leftovers of feat/org-merge-queue-order」 | (新規) | L6、L8、test の Test gaps 1〜3 を (a)〜(e) にした。Impact は LOW、(d) と (e) は到達できない守り |

Related には、どの行にも `docs/plans/active/2026-11-20-org-merge-queue-order.md` を足した(`archive-plan.sh` が `/pr` で書き換える)。tech-debt の 541 行を `git show HEAD:` の版と行ごとに比べた(`tdcheck.py`)。変わった行は 182、247、248、259、311、317、318、420、423、502、535、536 と、足した 1 行だけで、5 つの欄に分かれ、backtick は欄ごとに偶数、`~~` も偶数、U+FFFD は 0 件。計画の参照の検査(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ grep)は欠けるパスなし。

### 当たらないと判断した候補

走査は、branch の変更ファイル 32 と、変えた Go の関数名とを、Trigger の欄(`~~` の取り消しを除く)に当てた。候補になったのに足さなかった行は次のとおり。

| 行 | 理由 |
| --- | --- |
| 88、179、241、249、251、254、257、298、309、316、395、424 | Trigger が別のファイルか、報告という出来事で、その名前が古い注の中にあっただけ。249 と 254 と 257 は director が人なしで merge-order の PR を読む、または保護の種類から決める話で、この branch は `lookupMergePRs` も保護の確かめも変えていない。251 は「外部コマンドを足す」で、この branch は既存の runner(`MergeOrderExec`)で `git`、`ssh -G`、`gh` を打つだけ。309 は ls-remote の失敗の文と `realMergeOrderExec` の誤りで、変えていない。395 が名指す 4 つのテストファイルは変えていない |
| 458 | `[org]` の検証(`validateLoaded`)の変更が Trigger。`config.go` の編集は `OrgDirectorConfig.AutoMerge` の doc だけ |
| 518 | (g)「`cmd/ralph/main.go` の誤りの出力」。この branch は、終了コード 7 の comment を直しただけ |
| 534 | 「`merge_test.go` の in place でない変更」。833 から 832 行で、base の動いた行を新しいテストへ移しただけの in place の直し。新しいテストは新しいファイルに置いた。行の「833」は書いたときの数のまま |
| 537 | この branch の S3 の注が既にある。`fbdb30b1` が `merge.go` に足した差は doc comment の正味 1 行(843 から 844)で、コードは足していない。注の数は `7b3acbcd` に結んである |
| 252 | この branch の注が既にある(`fbdb30b1`)。注の `org_merge_order.go` の 318 から 383 行は `27723e15` で数えた値で、この run の help の編集で 384 行になった。数は commit に結んであるので書き換えない |

## 確かめたもの

- `./scripts/check-skill-sync.sh`: 14 skill が一致。`./scripts/check-sync.sh`: 全部一致。skill を直した 3 回のあと、`.agents/skills/` は `./scripts/sync-skills.sh` で、`templates/base/.claude/skills/org/SKILL.md` と `templates/base/.agents/skills/org/SKILL.md` は `cp` で写した
- `./scripts/run-static-verify.sh`: rc 0(`gofmt`、`go vet`、golangci-lint、staticcheck、同期の検査、template の純度、tech-debt の計画の参照、branch の secret scan `1d05c9ac..70cdf181` が clean)。この report を書く前の HEAD `70cdf181` での値で、そのあとの commit は文書だけ
- `go test ./internal/cli/ -run OrgMergeOrder`(help の文字列を変えたあと)、`go test ./internal/config/ ./internal/scaffold/ ./internal/upgrade/`: 通過
- `plan-visual.sh digest`: `357148b1a947`(計画の `- Approved:` の行と一致)
- 文書のずれの検査(skill の「Harness-internal sync」): skill の追加、削除、改名はなく、hook、rule、language pack、script は変わっていない。`docs/quality/quality-gates.md`(と template の写し)は merge-order を動詞の一覧で挙げるだけで、class の定義は書かないので直すところはない。`docs/architecture/repo-map.md` は `internal/org/` のファイルを挙げない。AGENTS.md の repo map は D8 で直した

## 申し送り

- コードを変えたのは、help の 1 文(`internal/cli/org_merge_order.go`)と `config.go` の doc comment だけで、振る舞いは変わらない。パイプラインを再び回すかは lead の判断。`.claude/skills/` を触るので、記録だけの修正の例外には当たらない
- help の「読めない」の理由の文は pin されていない。`TestOrgMergeOrder_HelpNamesTheMergeQueue` に足せば、README と skill の列との食い違いをテストが拾う。足すと `org_merge_order_test.go`(1671 行)が増えるので、今回は足していない
- self-review の M1 は `fbdb30b1` の skill の文と `d729352e` の固定のテストで閉じた。L5 は `fbdb30b1` で直った。L6 と L8 は新しい行にある
- 本物の GitHub での確かめは、536 行の Trigger に、release のあとの手順を書いた

## Insight event

`./scripts/insights-append.sh --slug org-merge-queue-order --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`(この report の次の commit)。
