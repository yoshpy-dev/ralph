# Self-review report: approve-resolve-decision

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-approve-resolve-decision.md`(issue #345)
- Branch: `fix/approve-resolve-decision`、HEAD `20ad8355`。base は origin/main との merge-base `e6d5d0ed`(branch は `f660f745` で origin/main を 1 度取り込み、origin/main はその後 `f280f2d6` に進んだので、差分は `git diff e6d5d0ed...HEAD`)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: 22 ファイル(+1537 −49)の diff の質だけ。コードは S1 `47257bf2`(`text_sha256`、`ResolveInboxItem`、`--without-decision`、help)、文は S2 `872775b6`(leader の prompt、director skill の 4 写し)と S3 `86f72048`(README、org skill の 4 写し、仕様の追記)。計画の記録の commit `20ad8355` は、Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書の drift は見ていない(`/verify` と `/test` の仕事)

## Evidence reviewed

- コード: `internal/org/approve_resolve.go` の全体(240 行)、`internal/org/manifest.go` の `TextSHA256`、`internal/org/verbs.go` の `Send` の doc と 2 つの `sent` の書き込み(492 行の dry run、679 行の本物)と `sentTextSHA256`、`internal/cli/org_inbox.go` の resolve の配線と `printResolveNotes`、`internal/cli/org_approve.go` と `internal/org/approve.go` の comment と help
- lock: `internal/org/lockfile.go` の `withFileLock`、`internal/org/inbox.go` の `withLock` の 3 か所(296 行 Escalate、371 行 Wake、529 行 update)、`withManifestLock` と `withManifestLockContext` の呼び出し元 13 か所
- 照らし方の材料: `currentPlanApprovals`、`lastDisbandedIndexes`、`planApprovalFromDetails`(パスを `normalizePlanPath` で確かめる)、`reseatProgressSentTask`、`ApprovalDecisionText` と `ApprovalDecisionCommand`、CLI の send が `--text` をそのまま `SendParams.Text` に渡すこと(`internal/cli/org.go:1444`)
- 文: `internal/org/prompts/leader.md` の手順 3・4 と立て直しの受信箱の項、director skill の「計画の承認」の 3・4、org skill の 3 か所(approve の行、受信箱の項、「機能の計画の承認」の 4)、README の 2 文、仕様の追記。4 写しは director、org とも `cmp` で同じ
- テスト(読んだだけで、リポジトリのテストは走らせていない): `internal/org/approve_resolve_test.go`、`internal/cli/org_inbox_resolve_test.go`、`internal/cli/org_approve_test.go` の差分、`internal/org/prompts_leader_plan_test.go` の追加分
- register: `docs/tech-debt/README.md` の 226 行目と 260 行目、`internal/org/deadline.go` の `InboxDeadlineDue`。この差分は `docs/tech-debt/` に触れていない

## 依頼された点

### lock の順と止まりうる所

- 入れ子は `ResolveInboxItem` の manifest → inbox の 1 か所だけ。inbox の lock の中(Escalate、Wake、update)は inbox のファイルを読んで 1 行足すだけで、manifest の lock を取らない。manifest の lock を取るほかの 12 か所(spawn 2、approve、gate_accept、gate、regate 2、merge_run、cleanup 2、`verbs.go:1848`)の閉包で inbox を呼ぶのは approve の `o.Inbox.Read()` だけで、`Read` は lock を取らない(`inbox.go:263-265` の doc)。deadlock の道は見つからなかった。
- 待ちの上限: resolve は manifest の lock を最大 5 秒待ち、持ったまま inbox の lock をさらに最大 5 秒待つ。inbox の lock の持ち主は読んで足すだけなので、実際の待ちは短い。この入れ子で「lock は互いに独立」という前提が変わる点を L1 と L2 に書いた。
- approve との順: approve が先に記録すれば resolve はその記録を読んで断り、resolve が先なら approve は lock を待ったあと記録し、`checkResolvedMeanwhile` が「ほかの誰かが resolve した」と言う。2 本のテストが、待つ側が lock に並んだことを `/dev/fd` の数で確かめてから進めている。

### `sent` の照らし方

- dry run(`ev.DryRun`)、承認より前(`wants` は承認の event のところで足されるので、それより前の `sent` は照らす相手がない)、別の org(`ev.OrgID != orgID`)、leader 以外の座席(`ev.SeatID == LeaderIdentity`)、`text_sha256` のない古い `sent`、最後の disband より前(`start` はその次)の 6 つを、どれも code が見ている。承認の拾い方は `currentPlanApprovals` と同じ条件になっている(写しである点は L4)。
- hash の元: CLI の send は `--text` をそのまま渡し、`Send` は bracketed paste の印を付ける前の `p.Text` の sha256 を書く(`verbs.go:680`)。approve が出す `--text` は承認の記録のパスと digest から作る `ApprovalDecisionText(ref)` を `shellQuote` したもので、resolve も同じ記録から同じ文を作る。approve が出したとおりに打てば一致し、`--raw` を付けても hash は変わらない。
- `submit_unconfirmed`: Details の先頭の `key=value ` の語を、`=` のない語か改行を含む語まで読む。承認の DECISION の本文は `TYPE:` で始まり `=` を含まないので、本文の語を印と取り違えない。数えた送信がすべて未確認のときだけ `SubmitUnconfirmed` になる。

### 何も書かない失敗の道

- note の検査、inbox のファイルがないとき(`os.Stat`)、断り(lock の閉包の中で `Resolve` より前に `return`)、manifest の読みの失敗は、どれも inbox に書かない。`manifest.lock` は作りうるが、inbox のある state dir の中だけ。
- `--without-decision` は `c.sent` の case より後なので、DECISION を送った件に付けても「leader に届いていない」とは出ない。

### prompt と skill の文と code

- director skill の 3・4: 合う。「approve が出したとおりの承認の DECISION」は hash の照らし方と、「終了コード 1。エラーに send の打ち方がもう一度出る」は `approvalDecisionNotSentError` と合う。
- org skill、README、仕様: 断りの文は合う。`--without-decision` の使いどころが refusal の文より狭い(L5)。
- leader.md: `inbox show` の `state:` と `note:` の行は `printInboxItem` の `field("state")` と `field("note")`、note の `approved <PLAN>` は `approveNote(ref)`、`escalated <id>` は escalate の stdout(`org_inbox.go:120`)と合う。立て直しのあとの id の出どころは書いていない(L7)。

## Findings

この run の指摘は M1 と L1 から振る。CRITICAL と HIGH はない。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | register 260 行目の trigger「The first report of a lost approval answer」に #345 が当たるのに、行が更新されていない。行には、resolve の期限で一度人に届いたあとに approve され、DECISION が送られなかった件は、もう期限にかからないと書いてある。resolve の期限は件ごとに 1 回で、ack の期限は open で `notified` のない件にしか効かない。この PR の安全網は、行が例外として挙げているこの期限を頼りにしている。Design decisions の 3 は「DECISION を送らない承認の件は開いたままになり、今の ack と resolve の期限が人に届ける」と言い、approve の help は「an approval whose DECISION was never sent does not leave the leader waiting unseen」と言う。resolve が断るので、閉じて見えなくなる道は塞がった。期限を使い切ったまま開いている道は残る | `docs/tech-debt/README.md:260`、`internal/org/deadline.go:80`(resolve の期限は `inboxNotifiedFor` で 1 回)と `:83`(ack の期限は open かつ `!item.Notified`)、`internal/cli/org_approve.go:57-62`、計画の Design decisions 3 | 行に日付つきの 1 文を足す。たとえば「2026-11-19: #345 で trigger に当たった。fix/approve-resolve-decision は resolve の断りを足しただけで期限は変えていないので、期限を使い切った承認の件は残る。carry-over」。approve の help の「does not leave the leader waiting unseen」は、期限が残っているときに限る言い方に弱める |
| L1 | LOW | maintainability | resolve は、`PLAN:` のない件(ALERT、BLOCKED、RESULT、ふつうの QUESTION)でも manifest の lock を取る。件の type と本文は escalate のあと変わらないので、計画の QUESTION かどうかは lock の外で決められる。今の形では、manifest の lock がほかの呼び出しに 5 秒を超えて持たれると、計画と関係のない件の resolve も落ちる。そのエラーは `org: acquire manifest lock ...: timed out after 5s` で、`inbox: resolve <id>` の前置きがない | `internal/org/approve_resolve.go:93-111`(lock は件を読む前に取る)、`:140-159`(type と PLAN: の判定は lock の中)、`internal/org/lockfile.go:121-128` | `readApprovalDecisionCheck` の inbox の読みと type・`PLAN:` の判定を lock の前に出し、計画の QUESTION のときだけ lock を取って読み直す。今のままにするなら、doc に「every resolve takes the manifest lock」と書く |
| L2 | LOW | maintainability | `withFileLock` の doc は「Each lock file is independent: holding one never blocks a caller of another」と言う。この PR で resolve は manifest の lock を持ったまま inbox の lock を待つので、inbox の lock の持ち主は、resolve を通して、manifest の lock を待つ spawn・approve・gate の記録を遅らせうる。manifest → inbox の順の決まりは `ResolveInboxItem` の doc にしかなく、次に lock を足す人が読む `lockfile.go` にない | `internal/org/lockfile.go:104-105`、`internal/org/approve_resolve.go:76-81` | `withFileLock` の doc の文を「入れ子は `ResolveInboxItem` の manifest → inbox だけで、inbox の lock の中では manifest の lock を取らない」に書き換える |
| L3 | LOW | naming | `approvalDecisionSent` は bool ではなく `approvalDecisionCheck`(承認の有無、ref、送ったか、submit の確認)を返す。名前は述語に読め、`ErrApprovalDecisionNotSent` の doc の「(approvalDecisionSent)」も述語として読める。`ResolveInboxItem` の switch の先頭 `case !c.approved:` は本文がなく、下の hook と `Resolve` に落ちることが読み取りにくい | `internal/org/approve_resolve.go:26-29`、`:99`、`:174` | `approvalDecisionFor` か `checkApprovalDecision` などに改め、空の case に `// no approval recorded: resolve as before` を足す |
| L4 | LOW | maintainability | 2 つの規則を写しで持っている。(a) `approvalDecisionSent` の承認の拾い方(dry run でない、org の org-level の `plan_approved`、最後の disband のあと、`planApprovalFromDetails` が読める)は `currentPlanApprovals` と同じ条件の手書きの写し。(b) `sentSubmitUnconfirmed` の語の読み方は `reseatProgressSentTask` のループの写し。どちらも「as X reads them」の comment だけでつながっている。Send が Details の前に新しい印を足すか、承認の条件が変わると、片方だけが変わる | `internal/org/approve_resolve.go:174-203`、`:213-225`、`internal/org/approval.go:376-391`、`internal/org/reseat_progress.go:254-261` | (a) を 1 つの述語(`isCurrentLifeApproval(ev, orgID)` など)に、(b) を Details を印の語と本文に分ける 1 つの関数にして、両方から呼ぶ |
| L5 | LOW | readability | `--without-decision` の使いどころが面ごとに違う。refusal のエラーと director skill の 4 は「この検査より前の ralph が DECISION を送ったとき」も挙げる。org skill の 3 か所は「DECISION を送らずにわざと閉じるときだけ使う」と「前の版の ralph で送った DECISION は数えない」を並べるだけで、「だけ」がその場合を外す。flag の説明、help、stderr の note は古い ralph で送った場合を言わない。その場合 note の「the leader of org %s has not been told ... so tell it another way」は事実と違い(leader は受けている)、従うと人は 2 度目の承認を送る | `internal/org/approve_resolve.go:237`、`.claude/skills/director/SKILL.md:1239`、`.claude/skills/org/SKILL.md:391`・`:569`・`:1197-1200`、`internal/cli/org_inbox.go:294`・`:321`・`:340-341` | note を「no approval DECISION is recorded as sent ...; unless the leader was told another way or by a ralph from before this check, tell it」の形に、org skill の 3 か所に「(または前の版の ralph で DECISION を送ったとき)」を足す |
| L6 | LOW | readability | 起こされた leader は、件の note が `approved <PLAN>` で始まるときだけ承認として進み、ほかの note なら BLOCKED を上げる。一方、`--without-decision` を案内する面(director skill 4、org skill、stderr の note)は、resolve の note を approve が出したもの(`ApprovalResolveCommand` の `--note`)のままにするとは言わない。CLI のテストの例も `--note "told the leader by phone"`。DECISION の届いていない leader が起こされると、承認が記録されていても BLOCKED になる。安全側に倒れ、人が答えれば進む | `internal/org/prompts/leader.md:464-479`、`.claude/skills/director/SKILL.md:1237-1240`、`internal/cli/org_inbox_resolve_test.go:99` | director skill 4 の `--without-decision` の文に「approve が出した resolve に `--without-decision` を足して打つ(note はそのまま)」を足す |
| L7 | LOW | readability | leader.md の手順 3 の終わりは、`<件の id>` を「上げたときに escalate が出した `escalated <id>` の行の id」とだけ言う。立て直しの受信箱の項は「手順 3 の終わりのとおりその件の」とそこへ送るが、立て直された leader はその escalate を打っておらず、id は「片付いていない受信箱の件」の行にある。「その件の」と書いてあるので id は読み取れるが、手順 3 の括弧と合わない | `internal/org/prompts/leader.md:348-350`、`:467-468` | 手順 3 の括弧に「立て直しのあとは「片付いていない受信箱の件」の行の id」を足す(`TestRenderRolePrompt_Leader_ApprovalWaitReadsTheResolvedItem` の断片も合わせる) |
| L8 | LOW | maintainability | `TestResolveInboxItem_ApproveWaitsForAResolveThatHoldsTheLock` は `<-paused` だけで待つ。`ResolveInboxItem` が hook の前に返る変更(note や型の検査の取り違えなど)が入ると、goroutine は `resolveDone` に送って終わり、テストは go test の timeout まで止まる。slice の verify は `-timeout=30m` | `internal/org/approve_resolve_test.go:531-542` | `select` で `paused`、`resolveDone`(失敗として `Fatalf`)、`time.After(3 * time.Second)` を待つ |
| L9 | LOW | readability | 計画の Risks の「DECISION を送ったのに断る: `sent` の Details は本文の先頭 200 バイトなので、同じ関数で切った承認の DECISION と末尾で照らす」は、計画助言の前の照らし方のままで、Design decisions の 4 と code(`text_sha256` の一致)と合わない | `docs/plans/active/2026-11-19-approve-resolve-decision.md:111`、同 Design decisions 4、`internal/org/approve_resolve.go:197` | 「`text_sha256` を approve が出した文の sha256 と照らす。手で書き換えた DECISION は数えない」に直す |

## Positive notes

- 照らし方を Details の先頭 200 バイトから本文全体の sha256 に変え、160 バイトと 180 バイトのパスで、承認、直しの依頼、違う digest の 3 つを区別できることをテストで固めた。
- 断りは lock の閉包の中で `Resolve` より前に返るので何も書かず、テストは inbox のファイルを前後で比べている。
- lock のテストは、持ち主だけのとき lock のファイルを開いている数が 1 であることを先に確かめ、そのうえで待つ側が並んだことを数で確かめる。数える probe が何も数えないときに、たまたま通ることがない。
- `submit_unconfirmed` と「届いていない」を result の欄で返し、文は CLI が stderr に 1 行で出す。org の層は文を持たない。
- 4 写しずつの skill は `cmp` で同じ。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(新しい行はない。M1 は既存の 260 行目に 1 文を足す直しで、この PR の中で済む。)_

## /sync-docs に渡すもの(数えない)

- org skill の verb 表の `inbox` の行(`.claude/skills/org/SKILL.md:387`)は、resolve の終了コード 1 を「知らない ID と resolved の件」とだけ言い、この PR の断りと `--without-decision` がない。文書の drift は `/sync-docs` の仕事なので、指摘に数えない。

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。M1 は register に 1 文足せば済むので、`/verify` の前に直すことを勧める)
- Follow-ups: L1 から L9 は、直すならこの PR の中で直す。L5 と L6 は `--without-decision` を案内する同じ面を直すので、まとめて 1 つの commit にできる。直さないものは、理由を cross-review の triage に書く
