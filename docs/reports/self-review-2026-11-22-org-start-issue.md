# Self-review report: org-start-issue

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-start-issue.md`(issue #365)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: origin/main `ccdf6293` から HEAD `d486f4b5` までの `git diff ccdf6293...HEAD`(36 ファイル)のうち、diff の質を見るもの。コードは `internal/org/reserve.go`・`feature.go`・`reseat.go`・`mergeorder.go`・`internal/cli/org.go`・`org_merge_order.go`、文書は leader の雛形(`internal/org/prompts/leader.md` の手順 14)と director・org の skill(4 写し)。テストは新しい 7 ファイルと既存テストの呼び出しの更新を読んだ。テストの実行と静的解析、仕様との突き合わせ、文書のずれは見ていない

## Evidence reviewed

- `git diff ccdf6293...HEAD` の `reserve.go`・`feature.go`・`reseat.go`・`mergeorder.go`・`internal/cli/org.go`・`org_merge_order.go` の全 hunk と、`featureBindingFromTokens`・`reservationDecision`・`featureDigestRefusal`・`featureLeaderTask` の周りの全文
- 依頼に挙がった所の呼び出し元の確認:
  - `sameFeature` の呼び出しは 3 か所(`gate.go:773`、`regate.go:1050`、`reserve.go:766`)。gate と regate は台帳を 2 回読んだ結びつき同士を比べ、どちらも `featureBindingFromTokens` を通る。`reserveAgain`(`verbs.go:1997`)は `Reservation` を丸ごと書き戻すので `Issue` が残る
  - `start --plan` のコマンドを組む所は `reseat.go:901` の 1 か所(grep)。`StartFeatureParams{` を作るのは `reseat.go:516` と `cli/org.go:1379` で、どちらも `Issue` を渡している
  - merge-order の JSON の名前(`entries`、`branch`、`class`、`binding`)は `mergeorder.go` の構造体のタグと skill の文が合う
- leader の雛形の手順 14 と `.claude/skills/pr/template.md`・`SKILL.md` の `Closes #__ISSUE__` の置き場の文
- director skill の追加 4 か所と、それが指す節(「起票と記録」の 4・終わり、「merge を待つ issue」の 3)の存在。skill の 4 写しは `cmp` で同一
- 新しいテスト(`reserve_issue_test.go`、`prompts_leader_issue_test.go`、`skill_issue_test.go`、`org_start_issue_test.go` の前半)と、既存テストの呼び出しの更新(`featureLeaderTask` の引数に `""`)
- `docs/tech-debt/README.md` の台帳を、diff が触れた記号(`featureBindingFromTokens`、`sameFeature`、`featureDigestRefusal`、`reservationDecision`、`featureLeaderTask`、`printStatusTable`、`orgFeatureJSON`、`reserve.go`)で引き、Trigger のセルを読んだ。diff に台帳の hunk はない

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | `featureIssueRefusal` は、「同じ機能か」の 6 項目の比べ(split、feature、branch、worktree、approval、held の完全さ)の 3 つ目の写し。`sameFeature`(`reserve.go:266`)と `featureDigestRefusal`(`:842`)にすでに同じ比べがあり、台帳の行 482 の (a)(e) がこの形(「片方に足した項目がもう片方に足されず、断りが間違った原因を言う」)を Trigger「`sameFeature`・`featureDigestRefusal`・`reservationDecision` の次の変更」つきで載せている。この PR は 3 つとも変えたが、行に触れず、写しをもう 1 つ増やした。次に結びつきへ項目を足す人は 3 か所を揃える必要がある | `reserve.go:271`(`sameFeature` に `a.Issue != b.Issue`)、`:844`(`featureDigestRefusal` の guard)、`:887-889`(`featureIssueRefusal` の guard)。台帳 482 の (a) の文と、#363 が足した「hit ... did not pay ... Carried over」の文 | 行 482 の (a)(e) が言う `sameFeatureExceptDigests` を 1 つ置いて 3 か所から呼ぶ(`held` の完全さと 5 項目の比べ)。この PR で払わないなら、482 に日付つきの繰り越しを足す(M3) |
| M2 | MEDIUM | maintainability | `orgFeatureIssueNumber`(`cli/org.go:2231`)は `mergeOrderIssue`(`org/mergeorder.go:348`)の写し。同じ `strconv.ParseInt` で、失敗を 0(issue なし)に握りつぶす。「10 桁なら int64 に入る」という前提のコメントも両方に書いてあり、cli 側は桁数 10 を数字で写している(`maxIssueDigits` は org の非公開)。計画は status の JSON と merge-order の JSON の `issue` が同じ型・同じ読み方であることを求める(AC7、AC10)が、それを 2 つの写しの一致で保っている | `cli/org.go:2224-2236`、`mergeorder.go:343-352`。呼び出しは `org.go:2279`(status の JSON)と `mergeorder.go:595` | org に `(*FeatureBinding).IssueNumber() int64`(nil の受け手と空に 0)を 1 つ置き、2 か所から呼ぶ。コメントも 1 か所になる |
| M3 | MEDIUM | maintainability | この PR が Trigger を満たす台帳の行に何も足していない(diff に `docs/tech-debt/README.md` の hunk がない)。(a) 行 548: `reserve.go` は 822 行から 936 行。行の Trigger は「次の変更、または 900 行を超えたら `feature_binding.go` と `reserve_paths.go` に move-only で分ける」で、2 つとも満たす。(b) 行 547: `printStatusTable` と `orgFeatureJSON` を変えた(`issue:` の行と `issue` のキー)。行が足すと言う `shared` は `feature:` の行の中の形で、今回の `issue` は別の行なので、足し方が 2 通りに分かれた。(c) 行 549 の (b)(d)、480 の (b)、225 の (c): `featureBindingFromTokens` に鍵を足した。`issue=` も読めない値を黙って「issue なし」にして痕跡を残さない形で、549 (b) の `shared=` と同じ。(d) 行 481 の (b): `featureLeaderTask` を変えた。#363 は同じ状況の 480・482 に日付つきの繰り越しを足した | `wc -l`: `reserve.go` 936(base 822)、`feature.go` 926(base 899)、`cli/org.go` 2542(base 2474)。Trigger のセルは `python3` で `re.split(r'(?<!\\)\|', 行)[4]` を読んだ | 各行の Trigger に「2026-11-22: feat/org-start-issue (#365) hit this trigger with ...、did not pay it: ...。Carried over to the next change that meets this trigger」を足す。548 は「936 行」と、行の 900 行の条件も満たしたことを書く。台帳の編集は records-only の例外の外なので、cycle 1 の sync-docs か直しのコミットで入れる(`/pr` の前) |
| L1 | LOW | maintainability | 計画の Risks の 4 つ目「古い binary が補償で結びつきを書き戻すと `issue=` が落ちる ... director の merge のあとの確かめで拾う」は成り立たない。director の確かめは merge-order の `binding.issue` を読み、skill は「`binding.issue` がなければ、確かめるものはない」と言う。落ちた記録は、そのブランチの最新の結びつきなので、`binding.issue` も落ちる。拾えるのは `Closes` を書き忘れた場合(Risks の 2 つ目)だけ | 計画 107 行目、director skill の「merge と auto_merge」の箇条の終わり、`mergeorder.go:595`(最新の結びつきが `t.binding` になる) | Risks の文を「拾えない。痕跡は `issue=` がないことだけで、混在する版の期間に限る。許す」に直す。計画を archive する前に直せば足りる |
| L2 | LOW | security | leader の手順 14 は「タスクに `- issue:` の行があれば」と言い、行の位置を言わない。タスクの後ろには、前文(フェンスの中)、機能の本文、立て直しの経過の囲み(commit の件名、受信箱の件)があり、どれも自由文。そこに `- issue: #N` で始まる行があると、leader が別の issue に `Closes` を書き、PR の merge でその issue が閉じる。`- 承認方式:` と `- 共有のパス:` も同じ読み方で、既存の流儀ではある。ただ、director の確かめは `binding.issue` の issue しか見ないので、間違った issue が閉じても拾わない。行があるときに計画の `Related issue:` が別の番号だった場合も、手順は「1 行書く」としか言わない | `leader.md` の手順 14(`gh pr create` の項)、`feature.go:630`(行は `- 依存する機能:` の直後、本文は `:647` で足す)、`prompts_leader_issue_test.go:37-45` が固定する文 | 「タスクの最初の空行までの行のうち」と位置を足し、計画の `Related issue:` が別の番号のときは書かない(または、`binding` の番号を優先する)と 1 文で言う。固定する文の fragment を同時に直す |
| L3 | LOW | exception-handling | director の merge のあとの確かめ(「merge と auto_merge」の箇条、後始末の 2、引き継ぎ)が、`gh issue view` の結果を `CLOSED` と `OPEN` の 2 つしか決めていない。(a) gh が失敗したとき(認証、通信)に、確かめられなかったことを人に言うとは書いていない。(b) GitHub が merge の直後に issue を閉じるまでの遅れは未確認。あれば、merge の直後の `OPEN` を「Closes を書かなかった」と人に伝える誤報になる。(c) 引き継ぎは `class` が `merged` で `binding.issue` のある全ブランチを毎回読む。merge-order は片付け済みの merged も出し(`mergeorder.go` に片付け済みを外す処理は見つからなかった)、履歴が増えると gh の呼び出しが増え、同じ OPEN を引き継ぎごとに言い直す | director skill: 箇条の終わり(`CLOSED`・`OPEN` と `binding.issue` がないとき)、引き継ぎの段落(「`class` が `merged` で `binding.issue` のあるブランチごとに」) | gh の失敗は「確かめられなかったと人に伝える」を 1 文、`OPEN` は「merge の直後なら 1 回読み直してから」、引き継ぎは「前の報告のあとに merged になったもの」に絞る(後始末の「新しい merged のブランチ」と同じ)。skill は 4 写しで、固定するテストは `skill_issue_test.go` |
| L4 | LOW | readability | `featureLeaderTask` の引数が `stateDir, worktree, branch, approval, issue string` と文字列 5 つ並び、今回 `approval` の次に `issue` が入った。入れ替えても compile が通る。本体の呼び出しは 1 か所で、既存のテストの 11 か所は `approval, "", reviewer` の形に直した。大半が `""` を渡すだけなので、入れ替えたテストは見つからない | `feature.go:609`、`feature.go:527`、`git diff` の既存テストの更新 11 か所 | 次に引数を足すときに、`stateDir` から `issue` までを小さな struct か名前つきの型にする。今回は許容 |
| L5 | LOW | readability | コメントの折り返し。リポジトリは 80 桁で手で折るが、足した行に 84〜92 桁が 3 行ある。`scopeReservedPathsKey` の doc には、折り直さなかった短い行(`// decimal digits, so every token`)が残る | `feature.go:235`(92 桁)、`reserve.go:178`(84 桁)、`:743`(86 桁)、`:70` | 4 か所を折り直す。コメントだけの変更 |
| L6 | LOW | maintainability | `start --help` の立て直しの条件の文は「as long as the plan's preamble, the feature's own section and the paths of the header's `- Shared:` line did not change and the approval mode is the one the org was started with」と条件を言い切る形のまま、`--issue` を入れていない。issue の条件は、少し下の新しい段落に書いてある。その文だけを読む人は、`--issue` を付け外した start も通ると読む | `internal/cli/org.go:1101-1107`、新しい段落は `:1131-1144` | 条件の文に「and the --issue is the one it was started with」を足す。help の固定テストは `TestOrgStartIssueHelp_DescribesTheFlag` |

## Positive notes

- Details の `issue=` は `shared=` のあと(最後)に書き、読む側は `featureBindingFromTokens` が鍵の順を問わず読む。`issue=` が `shared=` の前にある手書きの記録、2 回、空、`0`、先頭の 0、`#`、符号、全角の数字、11 桁、note の後ろは、どれも「issue なし」で、結びつきは完全なまま。テストが 19 通りを固定している(`TestFeatureBinding_IssueRead`)
- 古い binary の読み(知らない鍵から先を note と読む)で、予約のパスとほかの 6 つの結びつきの項目が変わらないことを、`oldBindingTokens` で `shared=` の有無の両方に固定している
- `issue` のないときのタスクと Details は、この変更の前とバイト単位で同じ(`featureLeaderTask` の `lines` の分け方を変えても行の並びは同じで、sha256 を固定するテストがある)
- `sameFeature` に `Issue` を足しても gate と regate の比べは変わらない。比べる 2 つは同じ台帳を別の時刻に読んだもので、同じ読み(`featureBindingFromTokens`)を通る。違いが出るのは記録が実際に書き換わったときだけで、そのときの断りは `String()` が issue を名指す
- 形の確かめが 3 段ある(CLI の空・空白、`readStartFeature` の `validIssueNumber`(分割計画を読む前、副作用なし)、`validateFeatureBinding`(記録の直前))。`--issue` なしの `--plan`・task は `Args` で、引数の数より先に断る
- やり直しの start の拒否の文は、記録と今の値と打ち直し方(`--issue 12` か `--issue` なし)を言い、digest の拒否と同時に起きたときは `, and` でつなぐ。別の機能の拒否には issue の句が付かない(テストで固定)
- reseat の座席なしの案内は、`binding.Issue` があるときだけ `--issue` を付ける(`start --plan` を組む所は 1 か所)。reseat の leader の start も `Issue` を渡すので、`sameFeature` で断られない
- status の `issue` と merge-order の `binding.issue` は、どちらも数(`int64`)で `omitempty`。0 は書き出さない値で、`validIssueNumber` が先頭の 0 と 0 を断るので、0 が本物の番号と衝突しない
- director skill の相互参照(「起票と記録」の 4・終わり、「merge を待つ issue」の 3)はすべて存在し、`--repo github.com/<owner>/<repo>` の決め方も「起票と記録」の終わりにある

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| director の merge のあとの issue の確かめに、gh の失敗、merge 直後の遅れ、引き継ぎの範囲の決まりがない(L3) | gh が使えない、または issue が閉じる前に読んだとき、人に誤報か何も言わない報告が出る。引き継ぎのたびに同じ OPEN を言い直す | 計画の Scope は確かめの手順を足すことで、失敗と範囲の決まりは書いていない。skill は 4 写しとテストを直す | 「merge と auto_merge」の箇条、後始末の 2、引き継ぎの段落の次の変更、または確かめの誤報の報告 | `docs/plans/active/2026-11-22-org-start-issue.md`(Scope、AC9)、この report の L3 |
| leader の `- issue:` の行の読み位置と、計画の `Related issue:` が別の番号のときの扱い(L2) | 本文や立て直しの囲みに `- issue:` で始まる行があると、別の issue が閉じる。director の確かめは拾わない | 既存の `- 承認方式:`・`- 共有のパス:` と同じ読み方で、計画の AC8 は行の有無の決まりだけ | 手順 14 の次の変更、または間違った issue が閉じた報告 | この report の L2、`internal/org/prompts_leader_issue_test.go` |

_台帳 `docs/tech-debt/README.md` には追記していない。依頼の範囲が report・計画の Progress・insight event の 1 コミットで、4 レーンが台帳の末尾に追記して衝突するため。上の 2 行と、M3 の各行への「繰り越し」の文は `/pr` の前に台帳へ入れてほしい(M3 が台帳の編集そのもの)。_

M3 の繰り越しの文の案(各行の Trigger のセルの末尾に足す。数は `wc -l` で、HEAD `d486f4b5` のもの):

- 548: `2026-11-22: feat/org-start-issue (#365) hit both conditions (the next change to reserve.go, and the file past 900 lines: 822 -> 936) with FeatureBinding.Issue, the issue= token, featureIssueRefusal and describeIssue, but did not pay it: a move-only split is outside the plan's Scope. Carried over to the next change that meets this trigger.`
- 547: `... hit this trigger with printStatusTable (an issue: line after feature:) and orgFeatureJSON (an issue key), but did not pay it: the plan's Scope is the issue. If shared is added, put it on a line of its own like issue:, not on the feature: line.`
- 549 (b)(d)、480 (b)、225 (c): `... hit this trigger with a new issue key of featureBindingFromTokens; an unreadable issue= is read as no issue and leaves no trace, as shared= does; the run with an older binary was not made. Carried over.`
- 481 (b): `... hit this trigger with featureLeaderTask (an issue line, not a field of the split plan, so it is left out of featureDigest on purpose; the comment says so), but did not pay it. Carried over.`
- 482 (a)(e): M1 を直さないなら `... featureIssueRefusal is a third copy of the comparison of sameFeature and featureDigestRefusal. Carried over.`

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 3、LOW 6)
- Follow-ups:
  - M1 と M2 は、cross-review の ACTION_REQUIRED で再実行になるときに直すのが安い(どちらも org のコードの数十行で、テストは既存のものが拾う)。直さずに進めるなら M3 の繰り越しの文に M1 を入れる
  - M3 は台帳の編集で、sync-docs か直しのコミットで入れる(`/pr` の前)。L1 は計画の文だけの直しで、archive の前に直せば足りる
  - L2・L3・L6 は skill と雛形と help の文の直しで、固定するテストの fragment を同時に直す(skill は 4 写し)
  - 範囲外で見た 2 点: 計画の Scope は仕様の「FR-2」に項目を足すと書いたまま(実際は FR-4。Progress の S3 に記録済み)、計画の AC の並びが AC10 のあとに AC9(読みづらいだけ)
