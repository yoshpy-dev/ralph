# sync-docs report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(承認の digest `e5a18ef9477a`)。issue #307 の前半
- Branch: `fix/watch-alert-dedupe`。base は origin/main `028a8f6d`、run 開始時の HEAD は `24ca3804`(cycle 3 の test の report のコミット)
- Pipeline cycle: 3(cap は cycle 2 の cross-review `7aee125e` で 3 に上がった。`.harness/state/standard-pipeline/cycle-count.json` の `cycle` は 2 のままなので、lead の指示どおり 3/3 と書き、insight event は `--cycle 3` で足した。cap 3 の最後の run)
- 先行 report(cycle 3): self-review の 2 回目 `521edd5b`(Merge: yes、LOW 1 件 C3-2 の持ち越し)、verify `10a86e28`(pass。LOW 3 件 V15〜V17、INFO 4 件 I11〜I14)、test `24ca3804`(pass。Test gaps 12 と 13 は INFO)
- 制約: 計画の本文は変えていない。変えたのは `## Progress checklist` だけ(cycle 3 の 4 行を足し、cycle の行を時の順に並べ直した)。`./scripts/plan-visual.sh digest` は編集の前後とも `e5a18ef9477a` で、`- Approved:` の行と同じ
- コードは変えていない。変えたのは Markdown だけ
- この report は cycle 3 の run で上書きした。cycle 1 と cycle 2 の run の内容は末尾に短く残した

## Summary

verify の V15〜V17 と、任意の I11 を処理した。tech-debt の既知の穴の行(462 行)の「cap 2 of 2」と「cap of 2 runs」を、cap を 3 に上げたことに合わせて直し、S10 と S11 が行の Trigger に `scopeCommittedPaths`(`watch_scope.go`)で当たったが払わなかったことを 1 文足した。357 行の行数の注は、最後の Go の commit を `e90c85da` にして `watch_scope.go` を 467 行に直した。仕様の FR-3 の注と tech-debt の RC3 の注に、commit のパスは `-c diff.relative=false` により、利用者の設定と座席の cwd によらず root からの相対で取れることを足した。計画の Progress は、cycle 3 の行を 4 つ足し、並びを時の順にした。

依頼の外で足したものが 1 つある。462 行の cycle 3 の文に「`evaluateReservedScope` の doc は 2 つの git を `-c` なしで要約している(verify I12)。その doc を次に直すときに一緒に直す」という 1 句を入れた。(5) の Trigger は「その doc の次の編集」なので、I12 を直す機会はそこしかなく、行に書かなければ誰も気づかないため。I12 そのもの(コードの変更)は直していない。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/tech-debt/README.md` | 3 行の変更(357、375、462)。下の「tech-debt」 | `baee33db` |
| `docs/specs/2026-10-07-org-multi-org-director.md` | (I11)82 行の FR-3 の注の 1 行だけ。fallback の `git diff` の文のあとに、「この `git log` と `git diff` には `-c diff.relative=false` を付けるので、利用者の `diff.relative` の設定や座席の cwd によらず、パスは repo の root からの相対で取れる」を足した | `baee33db` |
| `docs/plans/active/2026-11-12-watch-alert-dedupe.md` | (V17)Progress だけ。cycle 2 の cross-review、S10、cycle 3 の self-review の 1 回目の 3 行を、cycle 2 の sync-docs の行の後ろへ移した(行の内容は変えず、移しただけ)。その後ろに、cycle 3 の self-review の 2 回目(`521edd5b`)、verify(`10a86e28`)、test(`24ca3804`)、この sync-docs の 4 行を足した。「PR created」の箱は元の位置 | この report のコミット |
| `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` | この report | この report のコミット |
| `docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` | `sync_docs` の行(cycle 3、pass)を `insights-append.sh --slug watch-alert-dedupe --cycle 3` で足した。ファイル名の日付は UTC(この run は 2026-11-12 02:46 UTC)。10 行から 11 行になった | この report のコミット |

## 依頼の項目

| 項 | 結果 |
|----|------|
| 1. V15 / C3-2(462 行の cap の文と引き継ぎ) | (8) の Why deferred の「(cap 2 of 2)」を「(cycle 3 of 3: the cap of 2 was raised to 3 for this task, and its third run went to S10 and S11)」にした。Trigger の欄の cycle 2 の段落の「the pipeline's cap of 2 runs, which the cross-review findings spent」は、「the pipeline's last run: the cross-review findings spent the cap of 2 runs, and the third run, from the cap raised to 3 for this task, went to S10 and S11」にした(cap 2 を使い切ったのも事実なので、2 つとも書いた)。cycle 3 の 1 文は、段落の「Carried over to the next change that meets this trigger」の前に入れた。S10(`ec538952`)と S11(`e90c85da`)が `scopeCommittedPaths` の引数と doc とそのテストを変え、行の Trigger に `watch_scope.go` で当たったこと、(5) と (6) の Trigger には当たっていないこと、(1)〜(3) と (5)〜(8) は払っていないことを書いた。(4) は #307b の範囲なので数えていない。Related は、verify に V15、self-review に C3-2、test の report に「cycle 3 の版(`24ca3804`)は 1〜11 の番号を保つ」を足した |
| 2. V16(357 行) | 「At 3bb1828b, the last Go commit of the branch」を「At e90c85da, ...」にし、`watch_scope.go` を 461 から 467 にした。`wc -l` を HEAD と `e90c85da` で取って、`watch.go` 2081、`watch_test.go` 4188、`watch_recurrence.go` 214 は `3bb1828b` から変わらないことを確かめ、その旨を括弧で足した。`3bb1828b` から HEAD の Go の差は `watch_scope.go` と `watch_scope_test.go` の 2 ファイルだけ(`git diff --stat`) |
| 3. V17(計画の Progress) | 上の表のとおり。時の順は、cycle 1 の cross-review、cycle 2 の self-review(2 回)・verify・test・sync-docs、cycle 2 の cross-review(S10)、cycle 3 の self-review(2 回)・verify・test・sync-docs。移した 3 行は `git show HEAD:` の行と同じ文字列で(python で全行を数えて照合し、消えた行は 0、足した行は 4)、digest は変わらない |
| 4. I11(任意) | 足すと決めた。仕様の FR-3 の注と、tech-debt の RC3 の注(375 行)の 2 か所。RC3 の注は、「`git log --cc --no-renames` ... a `git diff` from the previous HEAD when there is none」の括弧の中に、「both commands carry `-c diff.relative=false`, so the paths are relative to the repository root whatever the user's `diff.relative` is and wherever the seat's working directory is」を足した。`scopeCommittedPaths` の doc と、`log` と `diff` の引数列(両方とも先頭が `-c diff.relative=false`)で確かめた |
| 5. I12〜I14 | 記録のまま。I12 は doc の直し(コードの変更)で、cap の最後の run のあとなので直さない。ただし上に書いた 1 句を 462 行に入れた。I13 は計画の本文(digest の内)で、記録されたずれとして扱う。I14 は INFO(2.26.3 と 2.49.0 のあいだの git の版は未確認)。Test gaps 12 と 13 も INFO で、test の report に残る |

## tech-debt

行は先頭の語で見つけた(行番号は `24ca3804` のもの)。`git show HEAD:` との比較で、変わった行は 357、375、462 の 3 行だけ(行数は 463 のまま)。3 行とも、列の数(357 と 462 は 5、375 は HTML コメントの 1 行)が変わらず、列ごとのバッククォートと `~~` が偶数で、新しい `file:line` が無く、U+2028・U+2029・U+FFFD が 0 件であることを、スクリプトで確かめた。tech-debt の plan 参照(115 件)は、無い参照が 0 件。

### 掃き出し(S10・S11 の名前で Trigger を読み直した)

Trigger の欄を python で切り出し、S10 と S11 が触った名前(`watch_scope`、`scopeCommittedPaths`、`newReservedScopeFixture`、`scopeStubGit`、`scopeSubcommand`、`diff.relative`、`evaluateReservedScope`、`scopeDefaultBranchRef`、新しいテストの名前)と、この sync-docs が編集するファイル(仕様、計画、tech-debt)で照合した。hunk は `git diff -U0 0a957745 HEAD -- internal/org/watch_scope.go` の 3 つで、どれも `scopeCommittedPaths` の doc と 2 行の引数の中にある(最初の hunk の見出しが `scopeDefaultBranchRef` なのは、その関数の閉じ括弧の次の空行から始まるため。関数は 254〜270 行で変わっていない)。

| 行(先頭の語) | 判断 |
|----|----|
| 462(Eight gaps) | 行全体の Trigger に当たった。上の表の 1 |
| 357(`watch.go` と `watch_test.go` の大きさ) | Trigger は「watch の escalation か、その tests の次の変更」。S10 と S11 が変えたのは予約の判定とそのテストで、escalation ではないので当たらない(verify の V16 と同じ)。数だけ直した。Trigger の欄に `watch_scope` が出るのはこの branch の注の中 |
| 210(止める頼みが周期の終わりまで待つ) | Trigger(打ち切り方の変更)には当たらない。S10・S11 は git の呼び出しの数を変えず、引数に 2 語足しただけなので、Debt の「最大 10 回」は変えない |
| 376(取り消し線の RC3 の行) | 閉じた行。Debt の欄に `watch_scope` が出るだけ。375 の HTML コメントは I11 で直した |
| 303、179、238、262、338、355、356、353、457 など | cycle 2 で読んだものと同じ。S10・S11 は `watch.go`、`watch_recurrence.go`、`inbox.go` を変えていない |

`watch_scope_test.go` の行数を書いた行は無い。`cap 2 of` と `cap of 2 runs` を grep すると、191 行と 443 行(別の PR の行)と、462 行の新しい文(cap 2 を使い切ったという事実の文)が残る。

## 確かめたこと

- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 本が lock-step、`./scripts/check-template-purity.sh` は PASS、`./scripts/check-pipeline-sync.sh` は ok。`baee33db` の前と、計画の Progress を直したあとの 2 回打ち、2 回とも同じ結果だった。skill、template、script は変えていない
- 計画の digest は 3 回(最初、tech-debt と仕様のコミットのあと、Progress を直したあと)とも `e5a18ef9477a`。Progress の差分は、移した 3 行(内容は同じ)と足した 4 行だけ
- README、AGENTS.md、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその写し、`/org` と `/director` の skill、`ralph org watch --help`、agent-messaging は、commit の一覧の引数を書いていない(`no-renames`、`diff.relative`、`no-relative` を `git grep` すると、計画、仕様、tech-debt、secret-scan のスクリプトとテスト、leader の role prompt の 817 行だけが出る。leader の 817 行は leader 自身が打つコマンドで、watch の一覧ではなく、この branch で変えていない)。直したものはない
- 変えたファイルに U+2028・U+2029・U+FFFD は 0 件(tech-debt と計画はスクリプト、そのほかは hook の検査)

## 残っていること(lead へ)

- 払っていないもの: (1)〜(3)、(5)〜(8) の引き継ぎの項目(コードの変更で、cap 3 の最後の run を使った)。M30・M50・M48・C24・C04 の probe テスト、N5 と I12 の doc の直し、(8) のコメントの折り返し
- `/director` skill の ALERT の件の項は、件に回数が付くことを書かない。#307b で直す(計画の Non-goals)
- 計画の Rollout の文(「skill は変えないので `ralph upgrade` は要らない」)は、本文を変えない指示のため直していない。cycle 1 の sync-docs が Progress に書いた(`/org` skill の文面を変えたので、文面については `ralph upgrade` が要る)。`/pr` の本文に release のノートの項として入れるかは lead が決める
- 実機での件の数え直しは、verify と test のとおり未実施のまま
- git 2.28 ちょうどの版と、2.26.3 から 2.49.0 のあいだの版は、どの report でも打っていない(I14、Test gaps 13)

## cycle 1 と cycle 2 の記録

cycle 1 の run(HEAD `3506f99a`、コミット `45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)は verify の V1〜V8、self-review の N5、test の M30・M50 を処理した。`/org` skill(4 つの写し)、README の 2 か所、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその template の写しに、今回の挙動(続いて立つ ALERT は件の `recurred`、待っている leader の stall は立てない、予約のある org の scope_change は予約の外だけ)を足した。tech-debt は既知の穴の行を 4 つから 6 つに増やし、Trigger に当たった 9 行に引き継ぎの注か数を足した。self-review の報告の U+2028 は 3 か所とも 6 文字のテキストに直し、計画の AC の 15 個の箱を `[x]` にした。cycle 1 の report の全文は `git show 2bebc1ac:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

cycle 2 の run(HEAD `fe4eff60`、コミット `0cc32e6d`、`74e6b1db`、report は `cd4a9268`)は verify の V9〜V14 と test の C24・C04 を処理した。仕様の FR-3 の注と tech-debt の RC3 の注に `--cc` と origin の既定のブランチの決め方を足し、既知の穴の行を「Eight gaps」にして (7) 再発を足した直後に止まる窓と (8) C24・C04 を足した。`/org` skill(4 つの写し)に、`recurred` の event に載った alert_id の escalation の打ち直しは何も書かず送らない、という 1 文を足した。357、457 (f)、210 に数か注を足した。cycle 2 の report の全文は `git show cd4a9268:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

## Insight event

```
./scripts/insights-append.sh --slug watch-alert-dedupe --flow standard --phase sync_docs \
  --verdict pass --cycle 3 --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` が 10 行から 11 行になり、足した行は cycle 3。ファイルの日付は UTC(この run は 2026-11-12 02:46 UTC)。`git status --short` は既存の jsonl と計画の `M` だけで、迷子のファイルはない。
