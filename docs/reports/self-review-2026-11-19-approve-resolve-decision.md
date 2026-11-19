# Self-review report: approve-resolve-decision

- Date: 2026-11-20(cycle 1 の 2 回目の `/self-review`。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)
- Branch: `fix/approve-resolve-decision`、HEAD `a908b2ec`。base は origin/main との merge-base `e6d5d0ed`(branch は `f660f745` で origin/main を 1 度取り込んだので、差分は `git diff e6d5d0ed...HEAD`)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: branch の差分全体(26 ファイル、+1858 −54。うち 1 回目の report と insight の 2 ファイル)の diff の質だけ。1 回目(report `3df4ad1a`、HEAD `20ad8355`)のあとの直しは `42524d6e`(M1、L1、L2、L3、L5、L6、L7、L8)と `a908b2ec`(L9、計画の本文)。テスト、静的解析、spec 適合、文書の drift は見ていない(`/verify`、`/test`、`/sync-docs` の仕事)

## Evidence reviewed

- コード: `internal/org/approve_resolve.go` の全体(270 行)を読み直した。`42524d6e` の `approve_resolve.go`、`lockfile.go`、`internal/cli/org_inbox.go`、`internal/cli/org_approve.go` の差分。branch 全体の `internal/org/approve.go`、`manifest.go`、`verbs.go` の差分
- lock の分け方の裏付け: `internal/org/inbox.go` の `InboxStore.Read`(266 行、lock を取らない)、`Resolve`(469 行)、`update`(524-545 行。inbox の lock を取ってから ID を探す)、`foldInbox` の `apply`(643-672 行。2 度目の `escalated` は無視するので、件の type と本文は escalate のあと変わらない)
- 名前の変更: `LeaderNotTold`、`approvalDecisionSent`、旧 note の文(`has not been told`、`without the approval DECISION`)を `internal/` と `*.md` で grep した。残りは、CLI テストの否定の確かめ(`org_inbox_resolve_test.go:122`)と 1 回目の report だけ
- 文: `internal/org/prompts/leader.md` の手順 3(455-480 行)、`reseat_progress.go` の `reseatProgressPlanSuffix`(457-470 行)と inbox の行の形(351-358 行)、`ralph org inbox` の一覧の列(`org_inbox.go:476`)。director skill と org skill は 4 写しずつ `cmp` で同じ。`--without-decision` が出る README、仕様、org skill の全か所の前後を読んだ
- テスト(読んだだけで、走らせていない): `42524d6e` の `approve_resolve_test.go`、`org_inbox_resolve_test.go`、`org_approve_test.go`、`prompts_leader_plan_test.go` の差分
- register: `docs/tech-debt/README.md` の 260 行目(差分)と 525 行目(`reseatProgressSentTask` の Details の語)。`/pr` の `archive-plan.sh` が register の `docs/plans/active/` のリンクを書き換えることも確かめた(`.claude/skills/pr/SKILL.md:58`)

## 1 回目の指摘の行方

| ID | 行方 | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | register 260 行目に 2026-11-19 の 1 文が入り、#345 が trigger の前半に当たったこと、期限は変えていないので行の穴は残ることを書いて carry-over にした。approve の help は「resolve の期限がまだ人に送っていないあいだ」に限る言い方になり、`org_approve.go` の doc も同じ。`approve.go` のファイルの doc だけ限らない言い方のまま(N2) |
| L1 | 直った | 最初の読みで見つかった計画の QUESTION でない件は、manifest の lock を取らずに `Inbox.Resolve` を呼ぶ(`approve_resolve.go:103-105`)。type と本文は変わらないので、lock の外で決めても競合しない。見つからない ID は lock を取って読み直す。lock の待ちが切れたときのエラーに `inbox: resolve <id>` の前置きがない点は、計画の QUESTION と見つからない ID に残る。resolve を打った直後に出るので、指摘には数えない |
| L2 | 直った | `withFileLock` の doc に入れ子の場所、順、deadlock の条件が入った。ただし入れ子を「計画の QUESTION のとき」とだけ書いている(N1) |
| L3 | 直った | `checkApprovalDecision` に改め、本文のない case は `if c.approved` の形に変わった。旧名の残りはない |
| L4 | 残す(follow-up) | 依頼どおり直していない。register には行がない(下の Tech debt identified) |
| L5 | 直った | refusal、director skill、org skill の 3 か所、flag、help、stderr の note、README が、どれも「別の方法で知らせたとき、または前の版の ralph で送ったとき」の 2 つを挙げる。note は「届いていない」と言わず「見つからなかったので確かめる」と言う。仕様の追記(`docs/specs/2026-10-07-org-multi-org-director.md:156`)だけは 1 回目の形のまま(下の /sync-docs に渡すもの) |
| L6 | 直った | refusal の最後の行が approve の resolve に `--without-decision` を足したもので、note は approve のまま。テストも note を `approveNote(ref)` にした。refusal の最後に出すことの副作用は N3 |
| L7 | 直った | 立て直した leader は「片付いていない受信箱の件」の行の終わりの `PLAN` で件を探し、なければ `inbox --all` の ORG 列で探す。行の形は `reseatProgressPlanSuffix` の ` / PLAN <path> sha256:<digest>` と、列は `ID STATE ORG ...` と合う。テストは `reseatProgressPlanSuffix` の出力も固めた |
| L8 | 直った | `select` で `paused`、`resolveDone`、30 秒の期限を待ち、ほかの待ちも `receiveWithin` で区切った。失敗したときは `t.Cleanup(releaseOnce)` が止めた resolve を進めて lock を離す |
| L9 | 直った | 計画の Risks が `text_sha256` の照らし方になった |

## 依頼された点

### L1 の、lock を取らない道と取る道の分かれ目

- 分かれ目は、lock の外の 1 回目の読みで「件があり、計画の QUESTION でない」こと。escalate は ID を、ファイル中で最大の `e<N>` の次にとり、`foldInbox` は同じ ID の 2 度目の `escalated` を数えないので、一度見えた件の type と本文はあとで変わらない。lock を取らない道に入った件が、あとで計画の QUESTION になることはない。
- 1 回目の読みが途中まで書かれた行に当たって件が見えないときは、lock を取る道に入って読み直すので、安全側に倒れる。
- lock を取る道に入るのは、計画の QUESTION(resolved のものも含む)と、1 回目の読みで見つからない ID。見つからない ID も `InboxStore.update` で inbox の lock を取ってから ID を探すので(`inbox.go:529-537`)、入れ子は計画の QUESTION のほかに、知らない ID の resolve でも起きる。順は manifest → inbox のままで deadlock の道は増えていないが、doc の 2 か所が狭い(N1)。
- 計画の QUESTION の resolve は inbox を 3 回読む(1 回目、`readApprovalDecisionCheck`、`update`)。inbox は 1 行ずつの JSONL で、手で打つ verb なので、指摘にしない。

### 名前を変えた `NoDecisionFound`

- `!c.sent` を入れるのは `c.approved` のときだけで、`withoutDecision` のないときはその前に断るので、true になるのは `--without-decision` で閉じた場合に限る。doc(`approve_resolve.go:46-50`)、CLI の `printResolveNotes` の doc と note、テストの名前(`...SaysNoDecisionWasFound`)が同じことを言う。`SubmitUnconfirmed` は `c.sent` のときだけ true なので、2 つの欄が同時に立つことはなく、`printResolveNotes` の `switch` の順に頼る所もない。
- 旧名 `LeaderNotTold` は grep で残っていない。`InboxResolveResult` は JSON に出さないので、名前を変えても外の形は変わらない。

## Findings

この run の新しい指摘は N1 から振り、残した L4 はそのままの ID で載せる。CRITICAL、HIGH、MEDIUM はない。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | maintainability | L1 と L2 の直しで足した 2 つの doc が、manifest → inbox の入れ子を「計画の QUESTION のとき」とだけ言う。code は、1 回目の読みで見つからない ID(知らない ID、その読みのあとに escalate された件)でも lock を取る道に入り、`InboxStore.update` は inbox の lock を取ってから ID を探すので、知らない ID の resolve でも inbox の lock を manifest の lock の中で取る。resolved の計画の QUESTION でも同じ。`ResolveInboxItem` 自身の doc は「For a plan QUESTION, and for an ID the read did not find」と正しく言うので、3 つの doc で範囲が食い違う。次に lock を足す人が読む `lockfile.go` が狭い | `internal/org/lockfile.go:104-110`(「for a plan QUESTION, takes the inbox lock」)、`internal/org/approve_resolve.go:53-56`(hook の doc の「for a plan QUESTION」)、`:65-66` と `:86-88`(本体の doc)、`internal/org/inbox.go:529-537` | 2 か所を「for a plan QUESTION, or an ID its first read of the inbox did not find」にそろえる。順の決まりは今のままで正しいので、文だけの直し |
| N2 | LOW | readability | M1 で限った言い方が `approve.go` のファイルの doc に届いていない。S1 で書き直したこの doc は「an approve whose DECISION is never sent leaves the item open, and the ack and resolve deadlines ... still reach it」と、期限が残っているかに触れずに言い切る。同じことを言う `org_approve.go` の doc と approve の help は、M1 の直しで「the inbox deadlines it has not used up」「while its resolve deadline has not sent it yet」に限った。件は acked のこともあるのに「open」とだけ言う | `internal/org/approve.go:19-24`、`internal/cli/org_approve.go:17-21`・`:60-67`、`docs/tech-debt/README.md:260` | 「leaves the item open or acked, under the deadlines it has not used up (the row of docs/tech-debt/README.md on an approved item whose resolve deadline was already sent)」の形にそろえる |
| N3 | LOW | maintainability | refusal のエラーの最後の行が、そのまま打てる `--without-decision` 付きの resolve になった(L6 の直し)。#345 は、approve が出した 2 つのコマンドのうち send を飛ばして resolve だけを打った director で起きた。今の refusal は send のコマンドを真ん中に、検査を飛ばすコマンドを最後に置くので、エラーの最後の行を写して打つ読み手は、#345 と同じく DECISION を送らずに件を閉じうる。条件の文(別の方法で知らせたとき、前の版の ralph で送ったとき)は前にあり、director skill の 4 はその 2 つに限って使うよう言い、閉じたときは stderr の note が人に確かめるよう言うので、閉じたことは見える | `internal/org/approve_resolve.go:261-268`(順は send → 条件 → `--without-decision` 付きの resolve)、`.claude/skills/director/SKILL.md:1237-1243`、`internal/cli/org_inbox.go:348-351` | 判断で決めてよい。変えるなら、検査を飛ばす resolve を send より前に置き、send を最後の行にする。または完成したコマンドを出さず「approve が出した resolve に --without-decision を足す(note は変えない)」とだけ書く。今のままにするなら理由を cross-review の triage に書く |
| L4 | LOW | maintainability | 1 回目と同じ(依頼どおり follow-up に残す)。(a) `checkApprovalDecision` の承認の拾い方は `currentPlanApprovals` と同じ条件の写しで、(b) `sentSubmitUnconfirmed` の語の読み方は `reseatProgressSentTask` のループの写し。どちらも「as X reads them」の comment だけでつながる。(b) と同じ危うさ(Send が Details の前に新しい語を書くと、読み手ごとに読み方がずれる)は register 525 行目が `reseatProgressSentTask` について持っているが、3 つ目の読み手になった `sentSubmitUnconfirmed` はそこに名前がない | `internal/org/approve_resolve.go:200-232`・`:239-251`、`internal/org/approval.go:376`、`internal/org/reseat_progress.go:253`、`docs/tech-debt/README.md:525` | 下の Tech debt identified の行を register に写す。(b) は 525 行目に `sentSubmitUnconfirmed`(`approve_resolve.go`)を読み手として足し、(a) は新しい行にする |

## Positive notes

- L1 の直しは、lock の外で決める根拠(type と本文は escalate のあと変わらない)を `isPlanQuestion` の doc に書き、見つからない ID は lock を取る側に倒した。新しいテストは、lock を持ったまま `PLAN:` のある BLOCKED と RESULT、`PLAN:` のない QUESTION を resolve し、対照として計画の QUESTION が lock を待つことを `waitForLockWaiter` で確かめる。
- 名前を変えたとき、CLI のテストは「has not been told」と言わないことも確かめ、件の note が approve のものであることまで見るようにした。
- director skill のテストは、refusal の最後の行が `ApprovalResolveCommand` に `--without-decision` を足したもので、note が `approveNote` であることを Go の関数から組み立てて確かめるので、skill の文と code が別々に変わると落ちる。
- 4 写しずつの skill は、直しのあとも `cmp` で同じ。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L4: fix/approve-resolve-decision が足した `checkApprovalDecision`(`internal/org/approve_resolve.go`)は、org の今の命の承認を `currentPlanApprovals`(`internal/org/approval.go`)と同じ条件で手書きで拾い(dry run でない、org-level の `plan_approved`、最後の disband のあと、`planApprovalFromDetails` が読める)、`sentSubmitUnconfirmed` は `sent` の Details の先頭の `key=value ` の語を `reseatProgressSentTask`(`internal/org/reseat_progress.go`)と同じループで読む。つながりは「as X reads them」の comment だけ | LOW。承認の条件か Send の Details の語が変わると片方だけが変わる。(a) がずれると、resolve が今の命でない承認で断るか、今の命の承認を見落として DECISION なしで閉じる。(b) がずれると、submit を確かめられなかった送信の 1 行が出なくなる | self-review の 1 回目の L4。1 つの述語と 1 つの Details の分け方にまとめると `approval.go` と `reseat_progress.go` も変わるので、この PR では写しのままにした(orchestrator の判断) | 承認の条件(`currentPlanApprovals`)か Send の Details の語(`verbs.go`、`send_fallback.go` の `details()`)の次の変更。(b) は register 525 行目と同じ trigger なので、その行に `sentSubmitUnconfirmed` を足してまとめて払う | `docs/reports/self-review-2026-11-19-approve-resolve-decision.md`(L4) |

_(この行は register にまだない。`/self-review` の決まりでは register にも足すが、この run の commit は report と insight だけと頼まれたので、写すのは orchestrator に任せる。)_

## /sync-docs に渡すもの(数えない)

- org skill の verb 表の `inbox` の行(`.claude/skills/org/SKILL.md:387`)は、1 回目と同じく resolve の終了コード 1 を「知らない ID と resolved の件」とだけ言い、断りと `--without-decision` がない。
- 仕様の 2026-11-19 の追記(`docs/specs/2026-10-07-org-multi-org-director.md:156`)は、`--without-decision` を「その DECISION を送らずにわざと閉じるときに使い」と言い、前の版の ralph で送った場合を使いどころとしてでなく「数えない」とだけ言う。L5 で直したほかの面と同じ 2 つの場合に合わせる。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。M1 と L1〜L3、L5〜L9 は直っている)
- Follow-ups: N1 と N2 は doc の文だけの直しなので、この PR の中で 1 つの commit にできる。N3 は順を変えるか今のままにするかを決め、今のままなら理由を cross-review の triage に書く。L4 は Tech debt identified の行を register に写す(525 行目に `sentSubmitUnconfirmed` を足す)
