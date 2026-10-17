# sync-docs report: org-regate

- Date: 2026-10-17(JST)
- Plan: `docs/plans/active/2026-10-16-org-regate.md`(承認済み。digest は sha256:3c417c48763c。この run で計画は触っていない)
- Pipeline cycle: 3。base は `670ec219`、branch は `feat/org-regate`。run 開始時の HEAD は `cf5aae54`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-16-org-regate.md`(cycle 3。C3-5 とその登録簿の行の勧め)、`docs/reports/verify-2026-10-16-org-regate.md`(cycle 3 の 2 回目は pass。LOW 6 件のうち V3-2〜V3-4 を下の変更 1〜4 で直し、V3-3 が勧めた行を変更 5 にする。V3-1・V3-5〜V3-7 はコードの commit で直っている)、`docs/reports/test-2026-10-16-org-regate.md`(cycle 3 は pass。Test gap 12 を変更 6 の行にする)
- cycle 1 と cycle 2 の sync-docs が直した面のうち、この run で触ったのは `/org` skill の `regate` の行、`AGENTS.md` の `internal/org/` の行、tech-debt の表だけ。cycle 2 の表は c60b2651 の版の report にある
- コードとテストは変えていない。手で直した Markdown は 4 ファイル(`/org` skill の `.claude` 側 2 面、`AGENTS.md`、tech-debt)、`scripts/sync-skills.sh` で再生成した skill の写し 2 面、insight event の 1 行

## Summary

cycle 3 のコード側の変更は、push だけの打ち直しと `reset --keep` の手が regate 自身の merge commit の形(親がちょうど 2 つ、件名まで。556f73fb、0844ad2b)を前提にしたことと、D3 の確かめと gate の記録を別のファイルへ移したこと(0e08999f、14252f06)。文書で合っていなかったのは、`/org` skill の条件(V3-2)、tech-debt の 4 つの行が D3 の関数を置くファイル(V3-3)、`AGENTS.md` の repo-map(V3-4)。この 3 つを直し、残る 2 点を tech-debt の 2 行にした。`ralph org regate --help` は、コードに当てて読み、直す箇所を見つけなかった(f850de13 と 3a452a71 が hook の件名の断りを足し済み)。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`(`regate` の行) | V3-2。(1) `reset --keep` の手の条件を、HEAD の第 1 親が origin の先頭、から、HEAD の親がちょうど 2 つで、第 1 親が origin の先頭、第 2 親がその記録の base、件名が regate の書く `Merge origin/<既定のブランチ> <base の 12 桁> into <ブランチ>` に直した。(2) 打ち直しの文に、push だけをする条件を足した。最新の gate の記録がその HEAD と既定のブランチの先頭の読める `gate_passed` で、origin のそのブランチの先頭が HEAD の祖先のときだけ。regate が書いた記録(`via=regate`)では、そのうえで HEAD が(1)の merge commit の形でなければならない。hook が件名を変えた merge commit は形に当たらないので、push も手も出さずに断り、HEAD を自分で push するよう言う(そのあとの regate は `UP_TO_DATE:`)。leader の記録(`via` なし)は、origin の先頭が HEAD の祖先であればよい。help(`internal/cli/org_regate.go`)の文面に合わせた |
| `templates/base/.claude/skills/org/SKILL.md` | 同じ 2 か所(root と同じ文字列で Edit した) |
| `.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `scripts/sync-skills.sh` で再生成した(root は引数なし、template は `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)。差分は上の 2 か所だけで、4 面は同一バイト |
| `docs/tech-debt/README.md`(D3 の 4 つの行) | V3-3。`regateMergeWrites`・`regatePathInTheWay`・`regateSpellingOnDisk` を置くファイルを `internal/org/regate_merge.go` から `internal/org/regate_inway.go` に直した(2.38 より前の git の行は本文と最後の欄、`mergeOptions` の行は最後の欄で `regateMergeWrites` は `regate_inway.go`・`regateMergeTip` は `regate_merge.go` と分けた、大文字小文字と入れ子の repository の行は本文と最後の欄、綴りだけ違うディレクトリのファイルごとの `git ls-files` の行は本文と最後の欄)。最後の行の影響の欄は、regate の間に断られるものを「`spawn` と `start --plan`」から「`spawn`(`--dry-run` を除く)と `start`(どちらの形も)」(D9)に直した |
| 同(表の末尾に 1 行) | self-review の C3-5 の(e)。push だけの打ち直しは、push が落ちたのか、届いたあとに origin のブランチが戻されたのか(force push)を見分けられない。leader の記録は origin の先頭が下にあるどの HEAD も push し直し、regate の記録は regate 自身の merge commit(親が origin の先頭と記録の base、件名が regate のもの)を、origin が第 1 親に戻されたときに push し直す。テストは、第 1 親の 1 つ下に戻した場合と gate だけの記録のあとを押さえる。gate だけの head への断り(`regateCheckOriginHead`)は「push してください」と言い、origin が戻された可能性は言わない。直し方は、push を試みた印を記録に足し、落ちた記録のあとだけ push し直すこと。この印は記録の形を変える(D2、D4)ので今は直さない。件名の確かめ(0844ad2b)のあとの形に書いた |
| 同(表の末尾にもう 1 行) | test report の Test gap 12。`newTestWatchRun`(`internal/org/watch_test.go`)は `os.MkdirTemp("", "watch-status-*")` で作るディレクトリを消さず、`internal/org` のテストを 1 回走らせるごとに `$TMPDIR` に約 29 個が残る(cycle 3 の /test で 4 回の実行で 116 個増えた)。直し方は `t.TempDir()`。ただし helper が `testing.T` を取らないので、28 の呼び出し元(どれも `watch_test.go`)に引数を足すことになる。この PR は `watch_test.go` を触らないので直さない |
| `AGENTS.md`(`internal/org/` の行) | V3-4。gate の句に `gate_record.go`(gate と regate が書き、merge-order が読む gate の記録)、regate の句に `regate_inway.go`(merge の前に、merge が書く場所に ignored のものがないかを確かめる)を足した |
| `docs/insights/events/2026-10-17-org-regate.jsonl` | `sync_docs`、cycle 3、pass、findings 0 の 1 行を `insights-append.sh` で足した(スクリプトは UTC の日付でファイルを選ぶので、test の event と同じ `2026-10-17-` のファイルに入る) |

書かなかった行: self-review が登録簿に足すよう勧めた行の(a)〜(d)(help と comment の文面、ファイルの行数、doc の言い回し)と、verify が勧めた V3-5〜V3-7 は、どれもコードの commit で直った(C3-1〜C3-4 は 82cf0d39・0e08999f・14252f06・ba84bccc、V3-1 は 0844ad2b、V3-5 と V3-6 は f850de13、V3-7 は 3989d914)ので、行にしていない。

tech-debt の新しい 2 行は、コードに当てて確かめてから書いた。`regateHeadNotOnOrigin` の 2 つの枝(`via=regate` なら `regateIsOwnMerge`、そうでなければ `regateIsAncestor`)と、`regateIsOwnMerge` の確かめ(親が `[origin の先頭, 記録の base]` のちょうど 2 つで、件名が `regateMergeSubject` と同じ)、`regateCheckOriginHead` の「push them」の文は読んだ。`newTestWatchRun` の呼び出しは `grep` で 28 と数えた。ただし(e)の「push し直す」の 2 つの場合は、読んで導いたもので、binary では打っていない(verify の Coverage gaps と同じ)。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `internal/cli/org_regate.go` の help | 合っている | 打ち直しの段落が「push だけをするのは、記録が HEAD の `gate_passed`、origin のブランチが HEAD の下、regate が書いた記録では HEAD が regate 自身の merge commit(親と件名)」で、hook が件名を変えたら断って「push HEAD yourself」と言う。変更 1 の skill の文はこれに合わせた。最後の段落の spawn(`--dry-run` を除く)と start(どちらの形も)も skill と同じ |
| `README.md`(regate の段落) | 変更なし | 「記録のあとの push の失敗は記録と merge commit を残し、打ち直すと push だけをする」と言い、条件は書かない。形と件名の条件は計画の D4 と help が持ち、この文は条件のない通常の場合として誤りではない。spawn と start の断りは `--dry-run` を除く spawn と start(どちらの形も)で、コードと合う |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(8a-3b の注) | 変更なし | 「打ち直すと push だけをする」は設計の要約で、条件は計画の D4 が持つ。cycle 2 で直した spawn と start の言い方のまま |
| `/director` skill(4 面) | 変更なし | regate が出した文(衝突したファイル、`REASON:` と `LOG:`、push の失敗の理由、断りの理由と次の手)をそのまま人に伝える、と言い、手と打ち直しの条件を並べない。hook が件名を変えた場合の断りもこれに入る |
| `/org` skill のほかの regate の記述(「merge してよい順」の分類と手順 4) | 変更なし | 分類 `regate` の「push が落ちたあとなら、regate は push だけをする」と、手順 4 の「push の失敗: 理由を直してから打ち直すと、regate は push だけをする」は、通常の場合の記述。hook が件名を変えた場合は手順 4 の最後の「断り(…HEAD が origin の先頭と違う、など): stderr の次の手に従う」に入る。条件をここに複製しない |
| `docs/quality/quality-gates.md` と template の写し | 変更なし | regate の文は「同じ 2 つのスクリプトを走らせ、`via=regate` で記録し、通れば push する」だけで、打ち直しの条件は書かない |
| `docs/architecture/repo-map.md`、`docs/quality/definition-of-done.md`、`.ralph/core/AGENTS.core.md` | 変更不要 | regate と `internal/org/` のファイルの一覧を持たない(grep で 0 件) |
| `docs/tech-debt/README.md` の D3 の 4 行と `regateMergeTip` の行 | 直した | 変更のとおり。`regateMergeTip` の行(断りの文が hook を挙げる)と、struck-through の行(S4b が merge などを `regate_merge.go` に移した履歴)は、そのときの記述として変えていない |
| `internal/org/prompts/`(`leader.md`、`reviewer.md`、`director.md`) | 変更不要 | 計画の Non-goals |
| `.claude/rules/ralph/` | 変更不要 | regate は座席にも message にも触れない |
| 計画の Progress | 変更不要 | 計画は触らない依頼。「Review artifact created」の行はすでに「今の版は cycle 3」になっている。`PR created` は /pr が付ける |
| 絶対パスの `$HOME` | なし | 今回の差分に `/Users/` も `hiroki.yoshioka` もない。U+FFFD もない(変更したファイルを数えた) |

## Left open

- test report の穴のうち、tech-debt に行がないもの: D4 と AC7 の誤りの枝と追跡の ref の読みの失敗(穴 1)、署名が ssh の鍵ファイルの場合だけであること(穴 3)、fetch と push の途中の中断(穴 4)、push・`git var`・merge・`reset --keep`・`git merge-tree` の上限(穴 5)、`merge --abort` 自体の失敗と `regateUndoMerge` の読めない場合(穴 6)、D3 の確かめの error の道と本物の古い git(穴 7)、skill の文の中身を固定するテストがないこと(穴 9)、端末の本物の Ctrl-C・ssh の origin・本物の GitHub(穴 10)、変異を走らせていないこと(穴 11)。計画の Progress のとおり PR の既知の穴に書く。穴 2 は変更 5 の行、穴 12 は変更 6 の行、穴 8 は tech-debt の大文字小文字の行(cycle 2)になった
- 本物の GitHub の「Update branch」の件名と、`-m` で regate と同じ件名を書いた人の merge commit(verify の I1、I3)。件名で見分ける以上避けられず、D4 が前提にしている。GitHub の件名が別の文でも、断る側に倒れる
- `git merge-tree` は `merge.<name>.driver` を動かすので、driver を設定した repository では merge-tree と `git merge` で 2 回動くはず(self-review の Coverage gaps)。確かめていないので、tech-debt の行にしていない。`branch.<name>.mergeOptions` の行は driver に触れていない
- `internal/org/director_cmd_test.go` の sh の trap を 100 ms の sleep で待つテストが、負荷が高い go1.25.8 の全体の実行で 1 回落ちたこと(tech-debt に行がある)。cycle 3 の /test では 5 回走らせて落ちなかった。この PR では直さない

## Verification

| Command | Result |
|---------|--------|
| `./scripts/sync-skills.sh`(root と template) | 14 skill と 13 skill を再生成。変わったのは `org` の 2 面だけ |
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/run-static-verify.sh` | 終了コード 0。scope は changed(golang)。tech-debt の plan の参照 OK、gofmt ok、golangci-lint(`0 issues.`)、`secret-scan-branch: scanned 670ec219..cf5aae54 against origin/main: clean`。出力は scratch に保存(repo の外) |
| `/org` skill の 4 面 | `cmp` で同一バイト(`.claude` と `.agents`、root と template) |
| `docs/tech-debt/README.md` の 276 行目以降 | 区切りの縦線はどの行も 6 つ(5 列)。新しい 2 行が挙げる関数とテストの名前(`regateHeadNotOnOrigin`・`regateIsOwnMerge`・`regateCheckOriginHead`・`newTestWatchRun`、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged`・`TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`)はコードに実在する。D3 の 3 つの関数は `internal/org/regate_inway.go`、`regateMergeTip` は `internal/org/regate_merge.go` にある。参照する plan と report は実在する |
| この report を書く前の変更 6 ファイルの U+FFFD、差分の `/Users/` と `hiroki.yoshioka` | どちらも 0 件 |
| `git diff --check` | 出力なし、終了コード 0 |

`./scripts/verify.local.sh` は走らせていない。変えたのは文書と skill の写しだけで、`run-static-verify.sh` が同じ範囲の検査(skill の lock-step、template の一致、tech-debt の plan の参照、secret scan)を含んで通っている。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Go のテスト(コードを変えていないので走らせていない。verify と test の report が先行している)。tech-debt の(e)の 2 つの場合(leader の記録と regate の merge commit が、origin を戻されたあとに push し直されること)を regate の binary で打つこと(読んで導いたまま)
