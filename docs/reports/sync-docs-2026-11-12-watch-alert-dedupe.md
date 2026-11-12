# sync-docs report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-watch-alert-dedupe.md`(承認の digest `e5a18ef9477a`)。issue #307 の前半
- Branch: `fix/watch-alert-dedupe`。base は origin/main `028a8f6d`、run 開始時の HEAD は `fe4eff60`(cycle 2 の test の report のコミット)
- Pipeline cycle: 2(`.harness/state/standard-pipeline/cycle-count.json` は cycle 2。cap 2 の最後の run)
- 先行 report(cycle 2): self-review の 2 回目 `d3a34921`(Merge: yes、LOW 1 件 C2b-1。`3bb1828b` が直した)、verify `0a957745`(pass。LOW 6 件 V9〜V14、INFO 5 件 I6〜I10)、test `fe4eff60`(pass。生き残った変異のうち LOW は M30・M50・C24・C04)
- 制約: 計画の本文は変えていない。変えたのは `## Progress checklist` に足した 5 行だけ。`./scripts/plan-visual.sh digest` は編集の前後とも `e5a18ef9477a`で、`- Approved:` の行と同じ
- コードは変えていない。変えたのは Markdown だけで、`/org` skill の本文を含む
- この report は cycle 2 の run で上書きした。cycle 1 の run の内容は末尾の「cycle 1 の記録」に短く残した

## Summary

verify の V9〜V14 と、test の C24・C04 を処理した。仕様の FR-3 の注と tech-debt の RC3 の注に、commit のパスの取り方(`--cc` で merge commit 自身の編集だけを数える。origin/HEAD が commit を指さないときの落ち方)を足した。既知の穴の行は「Six gaps」から「Eight gaps」になり、(7) 再発を足した直後に止まる窓と、(8) C24・C04 の 2 つの固定されていない枝を足した。S8 と S9 が Trigger に当たった行(既知の穴の行そのもの、210、357、457)に数と引き継ぎの注を足した。計画の Progress に cycle 2 の 5 行を足した。

依頼の外で足したものが 2 つある。(1) `/org` skill(4 つの写し)の「見張りの ALERT の件」の節に、再発の event に載った alert_id の escalation の打ち直しは何も書かず送らない、という 1 文(依頼の 6 の「任意の 1 文」を足すと決めた)。(2) RC3 の注の「the row that starts "Four gaps that fix/watch-alert-dedupe leaves"」は、cycle 1 で見出しが「Six gaps」になったときに取り残されていた。見出しの数を書かない言い方に直した。

## Changes made

| File | Change | Commit |
|------|--------|--------|
| `docs/specs/2026-10-07-org-multi-org-director.md` | (V9) 82 行の FR-3 の注の 1 行だけ。commit のパスは `git log --cc --no-renames`(merge commit は自身の編集だけを数え、merge が upstream から運ぶ変更は数えない)で、rebase や merge で入る upstream の commit は入らない。origin の既定のブランチは、origin/HEAD の指す先が commit を指すならそれ、そうでなければ origin/main、origin/master の順。どれも無ければ `git diff` | `0cc32e6d` |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「同じ alert_id の件がもう受信箱にあれば」の項の末尾に 4 行。alert_id が件の `recurred` の event(件が resolve されていてもよい)にあるときも、何も書かず人にも送らず、stderr に 1 行書くだけ | `0cc32e6d` |
| `docs/tech-debt/README.md` | 下の「tech-debt」。5 行の変更(462、457、375、357、210) | `74e6b1db` |
| `docs/plans/active/2026-11-12-watch-alert-dedupe.md` | (V14) Progress の末尾に 5 行。self-review の 2 回目(`d3a34921`)、`3bb1828b`、verify(`0a957745`)、test(`fe4eff60`)、この sync-docs | この report のコミット |
| `docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` | この report | この report のコミット |
| `docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` | `sync_docs` の行(cycle 2、pass)を `insights-append.sh --slug watch-alert-dedupe --cycle auto` で足した。ファイル名の日付は UTC | この report のコミット |

`.agents/skills/org/SKILL.md` は `scripts/sync-skills.sh` で作り直し、`templates/base/` の 2 つは `cp` で合わせた。4 つの `org/SKILL.md` は md5 が同じ。足した 4 行の表示幅(East Asian wide を 2)は最大 75 で、同じ節の既存の行の最大 79 に収まる。

## 依頼の項目

| 項 | 結果 |
|----|------|
| 1. V9(仕様 82 行)、V10(RC3 の注) | 両方に `--cc` の説明と、origin の既定のブランチの決め方(origin/HEAD の指す先、origin/main、origin/master、`git diff`)を足した。書き方は `scopeCommittedPaths` と `scopeDefaultBranchRef` の doc に合わせた |
| 2. V11 既知の穴の行 | (a) を (7)、C24・C04 を (8) にした。(b) M48 の Trigger を S8 が踏んだことと、(c) (5) の N5 が S8 のあとも残ることは、Trigger の欄の末尾の引き継ぎの注に書いた。(d) (1)〜(3) の引き継ぎもその注に書き、(2) の窓が S8 のあとも同じことを足した。(e) 見出しは「Eight gaps」。C2b-1 は入れていない。Related は関数名とコミットで書き直した |
| 3. V12 210 行 | 予約のある org の座席は、最大 8 回の git(origin の既定のブランチの探索が最大 4 回、commit の一覧、前の HEAD、merge-base、ls-tree)と、毎周期の status と HEAD の 2 回で、合わせて最大 10 回、git が止まれば 1 座席でおよそ 100 秒。`foldIntoPendingAlert` が畳む再発 1 つごとに、manifest の読みと、leader 以外が対象の alert なら `leaderActedSinceAlert` の 2 回の呼び出し(herdr の `agent get`、agmsg の履歴)が入り、止める待ちが 1 回あたり最大およそ 20 秒延びることを足した。manifest の読みはローカルのファイルなので時間切れの数には入れていない |
| 4. V13 357 行 | `watch.go` 2081 行、`watch_test.go` 4188 行、`watch_recurrence.go` 214 行、`watch_scope.go` 461 行を、最後の Go のコミット `3bb1828b` で `wc -l` を取って書いた。前の数(147 と 449)には「3506f99a のもの」と付けた。cycle 2 でも Trigger に当たり、払わないことを書いた。S8 は `watch_test.go` に 4 本のテストを足し、S9 がそのうち 1 本を消した(verify の「3 本」は正味の数) |
| 4. V13 457 行 (f) | 注を足すと決めた。理由は 2 つ。S9 は `leaderActivityEventCount` の doc を書き換えたので、コメントだけの変更を当たったとみなす前例(179 行、cycle 1 の 355 と 258)に当たる。S8 はこの関数に 2 つ目の呼び出し元(`foldIntoPendingAlert` から `leaderActedSinceAlert` 経由)を足し、(f) の Impact を広げた。`resumed` を leader の動きと数えると、deadman の取り消しに加えて、次の再発が新しい ALERT になる |
| 5. V14 計画の Progress | 5 行を足した(上の表)。digest は変わらない |
| 6. 文書の見直し | 下の「確かめたこと」 |

## tech-debt

行は先頭の語で見つけた(行番号は `fe4eff60` のもの)。5 つの列、各列のバッククォートと `~~` が偶数であること、`git show HEAD:` との比較で変わった行が意図した 5 行だけであること(行数は 463 のまま)、新しい `file:line` が無いこと、U+2028・U+2029・U+FFFD が 0 件であることを、スクリプトで確かめた。tech-debt の plan 参照の検査も通った。

### 既知の穴の行(462 行)

- 見出しを「Eight gaps」に直した。(6) の出典の「Test gaps 1 to 3」は cycle 1 の版の番号で、cycle 2 の版では M30 と M50 が 1 と 2、M48 が 6 なので、変異の名前と 2 つの版のコミット(`3506f99a`、`fe4eff60`)で書き直した。(6) の「test repositories の origin/HEAD はいつも `refs/remotes/origin/` の下」は、S8 の指す先の無い origin/HEAD の test も下を指すので、そう書いた
- (7) は C2-1 の残る窓。立ち上がりの `recordRecurrence` が受信箱の件に `recurred` を足したあと、状態ファイルを保存する前に watch が止まると、立て直した watch は件の回数を 1 つ多く足す。escalate の打ち直しの側は S8 の `alertItemWithRecurredID` で閉じたことも書いた
- (8) は C24(leader の herdr `agent get` の文字列の変化を活動と数える枝。`leaderActedSinceAlert`)と C04(1 つの鍵に pending が 2 つある古い status file で、古いほうが答えられたあと新しいほうを試す道。`foldIntoPendingAlert`)。どちらも probe 1 本で死ぬ(`go test -overlay`。コミットしていない)。C24 は S8 より前からある穴とみられる(base では変異を打っていない)
- Trigger の欄に (7) と (8) の直し方と、S8・S9 が行の Trigger と (5)・(6) に当たって何も払わなかったことの注を足した。(2) の窓は S8 のあとも同じ(`escalateToInbox` が件を記録してから回数を足す)
- 足していないもの: C2b-1(`3bb1828b` が直した)。self-review が案に出した (8)(コメントの折り返しの取り残し。`watch.go` の 7 行と `watch_recurrence.go:148` の 87 桁など)は入れない。gofmt の対象でなく、挙動にも影響せず、次にそのコメントを触れば直る程度で、行に足す価値より、行を長くする費用のほうが大きいと判断した。test の C02(INFO)、C18(INFO)、C07b(等価)、M56(等価)、Test gaps 9〜11 は、test の報告に残る。Test gaps 11 の「再発を足した直後の窓に tech-debt の行が無い」は (7) で解消した

### Trigger に当たり、数か注を足した行

| 行(先頭の語) | 当たったもの | 判断 |
|----|----|----|
| 462(既知の穴の行) | S8・S9 が `recordRecurrence`、`escalateToInbox`、`checkDeadman`、`scopeDefaultBranchRef`、`scopeCommittedPaths` を変えた | 払わない。(5) と (6) の Trigger には名前で当たった。注は上のとおり |
| 357(`watch.go` と `watch_test.go` の大きさ) | `escalateToInbox`、`checkDeadman`、`watch_test.go` の 4 本のテスト | 払わない。数は 3bb1828b のもの |
| 457 (f)(`leaderActivityEventCount`) | S9 の doc の書き換えと、S8 の 2 つ目の呼び出し元 | 払わない。deadman のテストは計画の Scope の外 |
| 210(止める頼みが周期の終わりまで待つ) | Trigger(打ち切り方の変更)には当たっていない | Debt の数え方だけを直した。S8 で `scopeDefaultBranchRef` の git が 3 回から 4 回になり、`foldIntoPendingAlert` が probe を呼ぶようになった |
| 375(RC3 の注) | verify の V10 | 直した。取り残された「Four gaps」の言及も直した |

### 退けたもの(自分の掃き出しで読んで外した)

Trigger の欄を python で切り出し(取り消し線だけの行は除く)、cycle 2 の hunk の関数名とファイル名、自分が編集した文書(仕様の FR-3 の注、`/org` skill、計画)で照合した。候補は 30 行余りで、Trigger を読んで上の行を残した。退けた主なもの:

- 303(`newTestWatchRun`): `watch_test.go` の変更で当たるが、呼び出しは 29 のまま(`newTestWatchRun(` を含む行は定義を入れて 30)で、cycle 1 の注の数は今も正しい。同じ PR の注なので足さない(verify の I10)
- 179、238、262、338、355、356: 名前が引き継ぎの注か別の条件の中にあるだけ、または `InboxDeadlineDue`・`sendInboxItemToHuman` を変えていない(verify の掃き出しと同じ)
- 353(`sendAlert`): hunk の見出しが `sendAlert` でも、変わったのはその後ろの `leaderActivityEventCount` の doc で、送り方も時間切れも変えていない
- 180 (a)、377(`/org` skill の表と split-plan の例): 足した文は表の外の「見張りの ALERT の件」の節にあり、`stop` と `disband` の cell も、split-plan の例も変えていない
- 118、211、44、195 など: 「stall」「recurrence」の一般の語、または `install` に当たっただけ

## 確かめたこと

- `./scripts/check-sync.sh` は `PASS: all files in sync.`(KNOWN_DIFF 6)、`./scripts/check-skill-sync.sh` は 14 本が lock-step、`./scripts/check-template-purity.sh` は PASS、`./scripts/check-pipeline-sync.sh` は ok。`0cc32e6d` の前と、この report と計画の Progress のコミットの前の 2 回打ち、2 回とも同じ結果だった。計画の digest も 2 回とも `e5a18ef9477a`
- `/org` skill を読むテストを回した。`go test ./internal/org/ -run 'Skill|SendDefaults|ReportNames|CodeBlockAfter|ReadRepoFile' -count=1` は ok、`go test ./internal/cli/ -run 'RetiredStateDir|Docs' -count=1` は ok、`go test ./internal/scaffold/ ./internal/upgrade/ -count=1` は ok
- natural-japanese の lint を `/org` skill に回した(`uv run --quiet`)。足した 666〜669 行は info(文頭の反復)だけで、warn 以上はない。仕様と計画の warn は、足した行の外の既存の行にある。この report の warn は 2 件で、5 行の和欧文間の半角空白(`origin/main` などの識別子の前)と、67 行の文末コロン(箇条書きの前)。cycle 1 の report と同じ書き方なので残した
- tech-debt の plan 参照(`docs/plans/active|archive/...`)は、`verify.local.sh` の検査と同じ式で、無い参照が 0 件
- 計画の digest は編集の前後とも `e5a18ef9477a`。差分は 163 行目のあとに足した 5 行だけ(`git diff -U0` の hunk が 1 つ)
- README(131 行、278 行)、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその template の写しを、S8 の 4 つの直し(leader が動いたあとの再発は新しい ALERT、`--cc`、指す先の無い origin/HEAD、`recurred` の event に載った alert_id)に照らして読み直した。どれも merge commit にも origin/HEAD にも、leader が動いたあとの扱いにも触れない要約で、今も正しい。直したものはない
- `/org` skill の 617〜625 行の「leader の返事待ち(まだ escalate していない)」は、leader が動いた pending を含まない読みで S8 (1) と合う。637 行の「予約の外を触る commit」は、予約の外の衝突を解決した merge commit も含む読みで合う。「origin の既定のブランチにある commit(rebase や merge で入ったもの)も数えない」も合う。直した箇所は、上に書いた 1 文だけ
- 足した文の主張を `escalateToInbox` で確かめた。`alertItemWithRecurredID` が当たると stderr に 1 行書いて nil を返し、件も新しい `recurred` も人への送りもない。件の状態は問わない(`TestWatch_Recurrence_AnEscalationAlreadyOnARecurredEventRecordsNothingNew` の open と resolved の 2 行)
- 変えたファイルに U+2028・U+2029・U+FFFD は 0 件(tech-debt はスクリプトで、そのほかは hook の検査)

## 残っていること(lead へ)

- 払っていないもの: (1)〜(3)、(5)〜(8) の引き継ぎの項目(コードの変更で、cap 2 は使い切っている)。M30・M50・M48・C24・C04 の probe テスト、N5 の doc の直し、(8) のコメントの折り返し
- `/director` skill の 967 行の ALERT の件の項は、件に回数が付くことを書かない。#307b で直す(計画の Non-goals、verify の cycle 1 の記録の I3)
- 計画の Rollout の文(「skill は変えないので `ralph upgrade` は要らない」)は、本文を変えない指示のため直していない。cycle 1 の sync-docs が Progress に書いた(`/org` skill の文面を変えたので、文面については `ralph upgrade` が要る)。この run でも `/org` skill に 1 文足したので、同じことが当たる。`/pr` の本文に release のノートの項として入れるかは lead が決める
- 実機での件の数え直しは、verify と test のとおり未実施のまま

## cycle 1 の記録

cycle 1 の run(HEAD `3506f99a`、コミット `45e84b3e`、`984761f8`、`0c727338`、`2bebc1ac`)は verify の V1〜V8、self-review の N5、test の M30・M50 を処理した。`/org` skill(4 つの写し)、README の 2 か所、AGENTS.md の `internal/org/` の行、`docs/quality/quality-gates.md` とその template の写し、`docs/recipes/worktrees.md` とその template の写しに、今回の挙動(続いて立つ ALERT は件の `recurred`、待っている leader の stall は立てない、予約のある org の scope_change は予約の外だけ)を足した。tech-debt は既知の穴の行を 4 つから 6 つに増やし、Trigger に当たった 9 行に引き継ぎの注か数を足した。self-review の報告の U+2028 は 3 か所とも 6 文字のテキストに直し、計画の AC の 15 個の箱を `[x]` にした。cycle 1 の report の全文は `git show 2bebc1ac:docs/reports/sync-docs-2026-11-12-watch-alert-dedupe.md` で読める。

## Insight event

```
./scripts/insights-append.sh --slug watch-alert-dedupe --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill
```

終了コード 0。`docs/insights/events/2026-11-12-watch-alert-dedupe.jsonl` が 4 行から 5 行になり、足した行は cycle 2。ファイルの日付は UTC(この run は 2026-11-12 01:37 UTC)。`git status --short` は既存の jsonl の `M` だけで、迷子のファイルはない。
