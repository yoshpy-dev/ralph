# sync-docs report: org-auto-merge

- Date: 2026-10-18(JST。計画の日付は 2026-10-17)
- Plan: `docs/plans/active/2026-10-17-org-auto-merge.md`(この run で計画は触っていない)
- Pipeline cycle: 3(上限を `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げた回し直し。`.harness/state/standard-pipeline/cycle-count.json` は 2 のまま)。base は `15d22b1a`、branch は `feat/org-auto-merge`。run 開始時の HEAD は `375674f6`(run 開始時の値で、この report を含む commit の SHA ではない)
- 前の版: この report は cycle 3 のために書き直した。cycle 2 の版(V3・V4 の直し)は git の `56efae4a` にあり、cycle 1 の版は `b5efa956` にある。cycle 2 の run は HEAD `668ce6d0` で始まり、今回の突き合わせの範囲は `668ce6d0..HEAD`(cycle 2 自身の直しを含む)のうち、`56efae4a` より後の変更を重点に読んだ
- 先行 report: `docs/reports/self-review-2026-10-17-org-auto-merge.md`(cycle 3 の LOW 7 件は `99f47e05`、Merge: yes。直しは S11a `f6e5208e`、S11b `dd1d4867`、`89829dbf`)、`docs/reports/verify-2026-10-17-org-auto-merge.md`(`af36aaac`。pass。情報 I1〜I8)、`docs/reports/test-2026-10-17-org-auto-merge.md`(`dd7d0d73`。pass。LOW の T1・T2 は `d099bab0` で直った)、cross-review cycle 2 の triage(`305738fe`。直しは S10 `590214fb`)
- この run が変えたのは `docs/tech-debt/README.md` の 3 行(292、299、300 行)とこの report だけ。コードも skill も help も触っていない

## Summary

`56efae4a` より後の変更は、コードでは S10(読み直しの `baseRefName`)と S11(Failed の文の整理、待ちの種類の case、注の折り返し)、テストでは `merge_readback_test.go` の新設と `d099bab0` の 2 本の追加、文書では README の `:266`、`/org` skill の 4 面、仕様の FR-8 の注、`seat.go` の注、tech-debt の 292 行と 299 行(新しい行)だった。これらを `ralph org merge --help`、README、skill の 8 面、仕様、`AGENTS.md`、quality-gates、tech-debt の 291〜300 行と突き合わせた。

ずれは tech-debt の 3 か所で、どれも `d099bab0` と、verify の情報 I6 の持ち越しによるものだった。

1. 292 行の注が `merge_readback_test.go` を 112 行と書いていた。`d099bab0` が 2 本足して 210 行になった。
2. 299 行(merge-order が別の base への merge を `merged` と並べる限界)の Trigger と参照欄が、`d099bab0` が足した pin のテスト `TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged` を名指さない。このテストは merge-order の分類を変えると落ち、`otherBaseMerged` の Failed の文を直す合図になる。行はこの合図を持つべきなので足した。Failed の文が base なしのとき「may have got nothing」になる変種も書いた。
3. 300 行(`TestRegate_FetchFailures` の揺れ)の「The PR only rewords a refusal in `regate.go`」が、PR の実際の変更(`regate.go` と `spawn.go` の注、`regate.go` の断り 1 つの文言)を数えていなかった。verify の情報 I2(cycle 2)と I6(cycle 3)が持ち越していた。直した。

`README.md:124` の動詞の表は変えない(理由は「Surfaces checked for drift」の README の行)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(292 行、RESOLVED の注) | `internal/org/merge_readback_test.go` の「(112, added by S10)」を「(210: S10 added it with 112 lines, and the two tests of the cycle-3 test fixes, `TestMerge_ComparesTheBaseWithOriginsDefaultBranch` and `TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged`, grew it)」にした。`wc -l` は 210。ほかの 4 file の行数(706、626、790、748)は合っている。800 行の file size の下であることは変わらない |
| 同(299 行、merge-order が別の base への merge を `merged` と並べる行) | (a) 2 つ目の欄の Failed の引用の末尾に「(S11; `may have got nothing` when gh read back no base)」を足した(`otherBaseMerged` が base なしのとき `got` を `may have got` にする)。(b) 4 つ目の欄(Trigger)の末尾に「and change the Failed's text in `otherBaseMerged` with it. `TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged` pins today's classification (a base of `release` and no base), so that change fails it and points here」を足した。(c) 5 つ目の欄の参照に `internal/org/merge_readback_test.go` を足した。欄は 5 つのまま |
| 同(300 行、`TestRegate_FetchFailures` の揺れの行) | 「The PR only rewords a refusal in `regate.go`」を「The PR changes `regate.go` and `spawn.go` only in comments and in one refusal of `regate.go` (the held-lock one, which now says "regate or merge")」にした。`git diff 15d22b1a..HEAD -- internal/org/regate.go internal/org/spawn.go` で、コードの変更が `regate.go` の断りの文 1 つ(と、それを固定する `regate_test.go` の 2 行)だけで、あとは注であることを確かめた。fetch の道(`regateBoundedGit`)に触れないという行の要点は変わらない |
| `docs/reports/sync-docs-2026-10-17-org-auto-merge.md` | cycle 3 のために書き直した |

行番号について。cycle 2 の report が「299 行」と呼んだ `TestRegate_FetchFailures` の揺れの行は、S11b が 299 行に新しい行を足したので 300 行になった。

## Surfaces checked for drift

| 面 | 結果 | メモ |
|----|------|------|
| `ralph org merge --help`(`internal/cli/org_merge.go`) | 合っている(変更なし) | 記録の条件「MERGED at the gate's head into origin's default branch」(`:89-90`)、別の base と base なしの場合(`:98-102`)、gate の記録だけのブランチの待ち(`:119-121`、`:128-129`)、終了コードの「went into another base than the default branch」(`:147-149`)は `merge_run.go` の `mergeAndRecord`(`:347-379`)、`otherBaseMerged`(`:432-442`)、`mergedElsewhere`(`:450-454`)の動きに合う。1 点だけ緩い: 「names the base on stderr」は、base が読めなかった場合には当たらない(Failed は base ではなく、既定のブランチへ merge されたかが分からないと言う)。head の句「names the head merged on stderr」と同じ緩さで、cycle 2 と verify の cycle 3 が受け入れている。動きを読み違えさせる文ではなく、直すと help と `TestOrgMerge_Help` の断片と Go のテストの再実行が要るので、直していない |
| `README.md` の `:266`(`merge` の段落) | 合っている(変更なし) | 読み直しの head と base、別の head と別の base の merge で記録せず `MERGED:` を出さないこと、gate の記録だけのブランチの待ちを書く |
| `README.md` の `:124`(動詞の表の `merge` のセル) | 変えない | 「and records `pr_merged`」に条件がない点(verify の情報 I7)を決めた。セルは通る道の要約で、隣の `gate` と `regate` のセルも断りと失敗の枝を書かない。条件(head が gate のもの、base が既定のブランチ)と記録しない終わり方は `:266`、help、`/org` skill、仕様が書く。条件は S10 と S11 で動いた箇所なので、表に写しを足すと次に動くときの直し先が 1 つ増える。表を読んだ人が「merge したら必ず記録される」と取っても、記録されない merge は検査のあとに誰かが head か base を動かした場合に限り、そのときは終了コード 1 と stderr の Failed で分かる |
| `/org` skill(4 面) | 合っている(変更なし) | `cmp` で同一バイト(`.claude`、`.agents`、`templates/base` の 2 つ)。`merge` の行(`:189`)は、記録の条件に「merge 先の base が origin の既定のブランチ」を持ち、base が違うか読めない場合を head の場合と並べて書く(S11b の直し)。skill は「merge-order を打ち直し、その merge は人が確かめる」と書き、merge-order がそのブランチを `merged` と出すことは書かない。それは Failed の文が言い、限界は tech-debt 299 行が持つので、skill には足さない |
| `/director` skill(4 面) | 合っている(変更なし) | `cmp` で同一バイト。`56efae4a` 以降に変わっていない。別の base の merge は `MERGED:` を出さない終了コード 1 なので、「終了コード 1 で終わったら、そのブランチはそこで止め、stderr の文をそのまま人に伝える」(`:491-493`)に入る。「終了コード 1 でも `MERGED:` の行が出ていれば」(`:494-496`)の扱いには当たらない。pin の断片(`directorSkillAutoMergeSteps`)に触れていない |
| 仕様 FR-8 の注(`:78`) | 合っている(変更なし) | head と base の読み直し、記録の条件、「head か base を stderr に名指して終了コード 1」、gate の記録だけのブランチの待ちが実装に合う |
| `internal/org/seat.go`(`EventPRMerged` の注) | 合っている | 「read the PR back as MERGED at the gate's head into the default branch」(`:93-94`)。`merge_run.go` の `mergeAndRecord` に合う |
| Go の注(`merge.go`、`merge_run.go`、`merge_wait.go`、`lockfile.go`、`regate.go`、`spawn.go`) | 合っている | `lockfile.go` は折り返しだけ。`regate.go` と `spawn.go` の注は merge が `regate.lock` を持つこと(記録は空の marker)を書き、`regateSpawnErr` の注の「待つ間は持たない」は `merge_wait.go` の動きに合う |
| `docs/tech-debt/README.md` | 直した(292、299、300 行) | この PR が足すか変えた行(236、237、239〜243、259、291、293〜300)の欄はどれも 5 で、U+FFFD はない。291〜300 行が名指す関数とテスト(`mergeLsRemoteRefusal`、`mergeGHSaid`、`listOriginBranches`、`realMergeOrderExec`、`realMergeDetachedExec`、`readChecks`、`mergeNoChecksSentence`、`mergeRefusalOrgRunning`、`runningRefusal`、`orgWaitEnd`、`mergeTargetOrg`、`collectMergeTargets`、`regateBoundedGit`、`enforceClassic`、`CheckMergeEnforcement`、`otherBaseMerged`、`classifyMergeTarget`、`lookupMergePR`、`TestRegate_FetchFailures`、`TestMergeGHSaid`、`TestCheckMerge_ReadsTheChecks`、`mxExec`、新しい 2 本のテスト)は `internal/` に実在する。293 行の行数(877 と 848、1334、318、480、1238 と 1152)は `wc -l` と base の `git show 15d22b1a:<path>` の行数に合う。298 行の gate の記録だけのブランチの限界は `orgRunsOn` と `orgWaitEnd` の動きに合う |
| `AGENTS.md` の `internal/org/` の行 | 変更不要 | `merge_run.go` は「with the merge, the read-back, and `pr_merged`」、`merge_wait.go` は「for checks still pending and for the branch's org to be disbanded」で、head と base の条件までは書かない map の粒度に合う。test の file はどの module も名指さないので、`merge_readback_test.go` と `merge_fake_test.go` は足さない |
| `docs/quality/quality-gates.md` と template の写し | 変更不要 | `merge` の確かめを門の行にする話はない(cycle 1 と同じ) |
| `docs/architecture/repo-map.md`、`.claude/rules/ralph/`、`docs/recipes/`、`docs/quality/` | 変更不要 | `git grep` で `org merge`・`merge_run`・`pr_merged` の語を探すと、この 4 つの場所(と `.claude/rules/`、template の docs)では `AGENTS.md` だけが持つ。`AGENTS.md` は上の行で確かめた |
| 絶対パスの `$HOME` | なし | 今回の差分にも report にも `/Users/` も `hiroki.yoshioka` もない |

## Plan drift notes

計画の本文は承認の digest の下にあるので、この run では書き換えていない。気づいたずれは次の 1 つ。

- AC12(`docs/plans/active/2026-10-17-org-auto-merge.md:116`)は tech-debt の行を「237、238、240、242 行」と名指すが、この PR が足すか変えた tech-debt の行は、243、259、291〜300 行(S6〜S11 が足した行を含む)に広がっている。Affected areas は `docs/tech-debt/README.md` を file として名指している(`:70`)ので、計画の範囲を外れてはいない。AC12 の行番号の列挙だけが古い。PR には影響せず、計画は PR で archive されるので、直すなら `/pr` の前の計画の更新に含める

verify の情報 I5(Affected areas に `merge_readback_test.go` がない)は `2f975743` で直っている。Affected areas の記述は今のファイルの一覧に合う。

## Left open

- test report(`dd7d0d73`)は `d099bab0` より前の HEAD で走った。`d099bab0` の 2 本(`TestMerge_ComparesTheBaseWithOriginsDefaultBranch`、`TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged`)は、計画の Progress が run-verify rc=0 と書く。この run はコードを変えていないので、Go のテストは走らせていない
- help の「names the base on stderr」が base なしの場合に当たらない緩さ(上の help の行)。LOW で、直していない
- gate の記録だけのブランチで `--wait-checks` が disband か時間切れまで待つ動き自体は、tech-debt 298 行が限界として持つ(cycle 2 から変わらない)
- README は `enabled` について「`ralph upgrade` does not add the key, so an existing `ralph.toml` gets the default true」と言うが、`auto_merge` には書いていない。`templates/base/ralph.toml` のコメントは「An absent auto_merge loads as false」と言う。既存の `ralph.toml` に `auto_merge` を足すのは人で、release ノートに書く項目になる(7 段目の `[org.approval]` と同じ)。README には足していない(cycle 1 から変わらない)
- `/director` skill の Codex の文は、Codex の shell ツールの timeout の上限と background の実行の有無を確かめていない。regate の前景の文と `/plan`・`/cross-review` の「its own background run」の書き方に合わせただけ(cycle 1 から変わらない)
- test report の揺れ(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound`、tech-debt 300 行)の原因は未特定。再び落ちたらその行の Trigger に従う(cycle 1 から変わらない。行番号は 299 から 300 に動いた)
- 本物の GitHub が、merge された PR の `gh pr view` で merge した先の base を `baseRefName` に返すかは確かめていない(verify の Coverage gaps と同じ。おそらくそう。未確認)

## Verification

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | 終了コード 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | 終了コード 0。`PASS: all files in sync.`(TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `director` と `org` の skill の 4 面 | `.claude` と、`.agents`・`templates/base/.claude`・`templates/base/.agents` の 6 組を `cmp` で比べ、どれも終了コード 0(同一バイト) |
| `gofmt -l internal` | 出力なし、終了コード 0 |
| `GOTOOLCHAIN=go1.25.8 ./scripts/run-static-verify.sh` | 終了コード 0。tech-debt の plan の参照 OK、gofmt ok、`0 issues.`、`secret-scan-branch: scanned 15d22b1a..375674f6 against origin/main: clean`、`All verifiers passed.`。出力は scratch に保存(repo の外)。evidence の log は gitignore の対象で、commit に入らない |
| `git diff --check` | 出力なし、終了コード 0 |
| `docs/tech-debt/README.md` の 236〜300 行のうち、この PR が足すか変えた 17 行 | 区切りの縦線で割った欄はどれも 5。U+FFFD なし |
| 291〜300 行が名指す識別子 | `git grep -w` で `internal/` に実在することを確かめた |

`./scripts/verify.local.sh` と Go のテストは走らせていない。変えたのは tech-debt の 3 行とこの report だけで、help の文もコードも skill も触っていない。上の範囲(skill の lock-step、template の一致、tech-debt の plan の参照と欄の数、secret scan)を通した。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: `d099bab0` の 2 本のテストを含む Go のテストの全体(この run はコードを変えておらず、走らせていない)、Codex の director が `--wait-checks 60m` を打つ実機の動き、`TestRegate_FetchFailures` の揺れの原因、本物の GitHub の merge 後の `baseRefName`

## Insight event

orchestrator が足す行(この run では jsonl に書いていない)。`ts` は足す時刻、ファイルは `docs/insights/events/<UTC の日付>-org-auto-merge.jsonl`:

```
./scripts/insights-append.sh --slug org-auto-merge --flow standard --phase sync_docs --cycle 3 --verdict pass --source skill --critical 0 --high 0 --medium 0 --low 0
```

```
{"schema":1,"ts":"<UTC>","slug":"org-auto-merge","flow":"standard","phase":"sync_docs","cycle":3,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```
