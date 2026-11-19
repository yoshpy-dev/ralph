# Walkthrough: 承認の DECISION を送るまで、計画の件を resolve できないようにする(approve-resolve-decision)

- 計画: `docs/plans/archive/2026-11-19-approve-resolve-decision.md`(PR の作成のあとに archive へ移す)
- issue: #345。PR: #350
- 差分: `git diff origin/main...HEAD`(この walkthrough を足す前で 34 ファイル、+2,520 / −75、base `32274500`)。Go の本体は 6 ファイル(新しい `approve_resolve.go` のほか、`manifest.go`、`verbs.go`、`lockfile.go` と `approve.go` のコメント、`org_inbox.go`、`org_approve.go`)、Go のテストは 5 ファイル、role prompt と skill の 8 写し、README・仕様・AGENTS.md・quality-gates・tech-debt、残りは report・計画・insight
- パイプライン: 1 cycle。cross-review(codex)の指摘は 0 件

## 1. `ResolveInboxItem`(`internal/org/approve_resolve.go:91`。読む価値が一番高い)

- note の検査、inbox のファイルの有無のあと、inbox を 1 度読む。件が見つかり、計画の QUESTION(`isPlanQuestion`、`:145`。`QUESTION` で `PLAN:` のヘッダがある)でなければ、manifest の lock を取らずに `InboxStore.Resolve` で閉じる(今までと同じ)
- 計画の QUESTION と、最初の読みで見つからない ID は、manifest の lock(`withManifestLock`、approve が承認を記録するのと同じ lock)の中で inbox を読み直し(`readApprovalDecisionCheck`、`:174`)、判定し、書く。書き込みは `InboxStore.Resolve` で、inbox の lock を取る。順は manifest → inbox で、inbox の lock を持ったまま manifest の lock を取る経路はない(`internal/org/lockfile.go` のコメント)
- 判定(`checkApprovalDecision`、`:202`): その org の最後の disband のあとの `plan_approved` のうち、この件のものを探す。なければ今のまま閉じる。あれば、その記録より後の `sent` で、dry run でなく、同じ org の leader 宛てで、`TextSHA256` が `sha256(ApprovalDecisionText(ref))` と一致するものを探す。あれば閉じる。どれも `submit_unconfirmed=true` なら、CLI が `ralph org read` で確かめる 1 行を出す(`sentSubmitUnconfirmed`、`:241`)
- 送っていなければ `ErrApprovalDecisionNotSent` を包んだエラーで、何も書かない(`approvalDecisionNotSentError`、`:268`)。エラーは send の打ち方、resolve の打ち方の順で、`--without-decision` は最後に文で触れるだけ(検査を飛ばす完成した打ち方は出さない。#345 の失敗は「最後の行を写して打つ」形だったため)
- `--without-decision` なら閉じ、結果の `NoDecisionFound` で CLI が stderr に 1 行出す

## 2. `sent` の `text_sha256`(`internal/org/manifest.go:52`、`internal/org/verbs.go:680`)

- `ManifestEvent.TextSHA256`(`omitempty`)。本物の send の `sent` にだけ、paste の印を付ける前の本文全体の sha256(`sentTextSHA256`、`verbs.go:972`)を入れる。dry run の `sent` には入れない
- Details は本文を 200 バイトで切る(`truncateForDetails`)ので、計画のパスが 150 バイトを超えると、承認と直しの依頼と違う digest の DECISION が同じ Details になる。Codex の計画助言の [HIGH] で、Details の照らし合わせから hash に替えた
- この版より前の ralph が書いた `sent` には欄がないので数えない。版をまたいだ承認の件は `--without-decision` で閉じる

## 3. CLI と help

- `ralph org inbox resolve` の `--without-decision`(`internal/cli/org_inbox.go:325`)と Long help、`printResolveNotes` の 2 つの stderr の行
- `ralph org approve --help`(`internal/cli/org_approve.go`): 「never resolves the inbox item」は残し、期限が届くのは件の期限をまだ使っていないときだけ、と書き直した。approve より前に resolve の期限が人に送られた件は、もう期限が届かない(tech-debt の行に持ち越し)

## 4. leader の prompt と skill

- `internal/org/prompts/leader.md` の手順 3: 承認を待つ間に DECISION なしで起こされたら(見張りの stall の ALERT、Monitor の張り直し、`ralph org resume` の文など)、件を `ralph org inbox show` で読む。`resolved` で note が `approved <この QUESTION の PLAN:>` なら承認として手順 4 に進む。`open` か `acked` なら待ち続け、`manifest.jsonl` を直接読まない。ほかの note で閉じていれば BLOCKED を 1 つ上げる。立て直された leader は、立て直しの経過の囲みか `ralph org inbox --all` で件の id を探す
- director skill と org skill の「計画の承認」の手順 4 などの 8 写し: send を先に、resolve をあとに。`--without-decision` を使う 2 つの場合(別の方法で伝えたとき、古い ralph で送ったとき)と、note を approve のままにすること

## 5. テスト

- `internal/org/approve_resolve_test.go`(13 本): 断り、送ったあと、`--without-decision`、数えない `sent` 8 種、160・180 バイトの長いパス、`submit_unconfirmed`、今のままの件、エラーの順(send が resolve より前、最後の行が打ち方でない)、別の件の承認、resolve 済みの件
- `internal/org/approve_resolve_lock_test.go`: approve と resolve の両方の順、計画でない件は lock を待たない、inbox への書き込みが lock の中、最初の読みで見つからない ID も lock を通る。lock のテスト 3 本を `-race` で 720 回走らせて落ちなし
- `internal/cli/org_inbox_resolve_test.go`、`org_approve_test.go`(dry-run の send だけで resolve していた既存のテストを、本物の send のあとに直した)、`prompts_leader_plan_test.go`(prompt と director skill の文)
- 変異: implementer と tester が `-overlay` で当て、残ったものは G1〜G4 のテスト(4c4edcd7)で落ちるようにした

## 見どころ

- 承認を記録した計画の件は、DECISION を送るまで閉じられない。`ralph org inbox resolve` の挙動が変わる(release ノートの候補は PR の本文)
- lock は計画の件のときだけ 2 つを入れ子にする。approve が先に記録すれば resolve は断り、resolve が先に閉じれば approve は今の `checkResolvedMeanwhile` で「ほかの誰かが resolve した」と言う
- 開いている穴: (1) 本物の herdr と leader では試していない。(2) approve より前に resolve の期限が送られた件は期限が届かない。(3) 承認の拾い方と Details の語の読み方が `currentPlanApprovals` と `reseatProgressSentTask` の写し(self-review の L4、tech-debt)
