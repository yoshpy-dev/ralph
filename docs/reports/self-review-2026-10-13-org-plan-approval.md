# Self-review report: org-plan-approval

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)の 2 回目の実行(run 2)。最初の結果(3d73dbf4、HEAD ef503bbd の時点)の指摘を直した 54feb287..1b95bb84 のあとに見直した。cross-review の前の直しなので cycle は 1 のまま。最初の結果は末尾の「付録」にそのまま残した(コードのコメントとテストの doc が「self-review M1 of org-plan-approval」の形でその ID を指すため)
- Scope: diff の品質だけ。ブランチ全体 `git diff 225dff27...HEAD`(HEAD 1b95bb84)を対象にし、直しの範囲 `git diff 6d3369a6..HEAD`(4 コミット、22 ファイル、+1510/-985)を重点に見た。テスト・静的解析・仕様適合・文書のずれの検査は /test・/verify・/sync-docs が受け持つので、ここでは行っていない
- 番号の付け方: 最初の結果の M1、L1〜L11 はそのまま使う。今回足した指摘は N1〜N5

## 最初の結果の指摘の扱い(HEAD 1b95bb84)

| ID | 状態 | 根拠 |
| --- | --- | --- |
| M1 | 9fb3fafa で直った(残りは N1) | 拒否の文は、見つけた状態、目印 `refused by the approval check`、理由の種類ごとの次の手 1 つ、leader がしないこと、の順になった(`planNotApprovedError.Error`、`internal/org/approval.go:624-631`)。理由の部分から次の手の文は消えた。`TestPlanNotApprovedError_OneNextStepPerKind` が 5 種類すべてについて、理由に次の手の語がないことと、次の手がその種類の文だけであることを確かめる。雛形の手順 7 は「目印の後ろの次の手に従う」に変わった。ただし、TASK の `PLAN:` の digest が古いときの種類の決め方が原因に合わない(N1) |
| L1 | 9fb3fafa で tech-debt に移した | 勧めたとおりコードは変えず、`docs/tech-debt/README.md` の 213 行目(tripwire の行)の Debt item に、検査が role の名前 `implementer` と TASK の `PLAN:` だけを読むことと、custom role の座席・reviewer への TASK・本文で別の作業を頼む TASK の 3 つの道を足した |
| L2 | 9fb3fafa で直った | `approvedLineDigest`(`internal/org/approval.go:244-257`)は、`- Approved:` の値が実在する日付 `YYYY-MM-DD` と `sha256:<12 桁の小文字の hex>` の 2 語のときだけ digest を返す。`/plan` の SKILL.md(99 行目)と leader の雛形の手順 4(127 行目)が書く形はどちらもこの形で、docs/plans の既存の計画の `- Approved:` 行もすべて 2 語だった。`TestApprovedLineDigest` の 15 ケースと `TestCheckPlanApproved` の 2 ケースがある |
| L3 | 半分直った(残りは N3) | 先にほかの誰かが resolve した件は、`ErrInboxResolved` を見分けて「承認は記録してある、resolve するものは残っていない」と言い、失敗する `inbox resolve` のコマンドを出さない(`resolveApproved`、`internal/org/approve.go:382-408`)。approve と、直しを頼むつもりの resolve が競合したときに承認が記録される半分は直っておらず、tech-debt にもない |
| L4 | 54feb287 で直った | approve の動詞を `internal/org/approve.go`(408 行)に、そのテストを `approve_test.go` に移した。削除した行と追加した行を取り出して比べると、違いは package 行、import、ファイル頭のコメントだけで、コードは同じだった(Evidence を参照)。approval.go は 635 行になった |
| L5 | 9fb3fafa で直った | `ApprovedPlan.Resolved` と `ApproveResult` の doc に、CLI は `Approved` と `Err` だけを読み、`Resolved` と `Refused` はテストと件ごとの結果がほしい呼び出し元のための欄だと書いた |
| L6 | 9fb3fafa で直った | CLI は `os.Getwd()` を呼ばず、`ApproveParams.Cwd` を空のまま渡す(`internal/cli/org_approve.go:64-66`)。作業ディレクトリは `Approve` が読む |
| L7 | 9fb3fafa で直った | `spawn.go:555-565`、`reserve.go:526-538`、`prompts_test.go` の該当の段落を折り返し直した |
| L8 | 半分直った(残りは N4) | `TestCheckPlanApproved` の中の `replace` は消え、`replaceOnce` を使う。`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` の重なりは残り、残した理由は計画のメモにもない |
| L9 | 9fb3fafa で直った | `templates/base/ralph.toml:127-129` は、`[org.approval]` がなければ既定の per-plan になり `start --plan` の org が `ralph org approve` を待つことと、元の動きに戻すには `mode = "split"` にすることを書く |
| L10 | 9fb3fafa で直った | `taskNotSentError` の欄の名前は `refusal` になった。`taskNotSentError{gate` と `planNotApprovedNext` を grep すると 0 件(`watchgate_positions_test.go` の `gate.gate` は別の型) |
| L11 | 9fb3fafa で直った | 雛形の前置きは「手順 1・4・6 の機能の計画」になり、`prompts_test.go` の固定の語句も合わせてある |

## Evidence reviewed

- 直しの範囲で読んだもの: `internal/org/approval.go`(新しい `approvalRefusalKind`、`approvalCheckError`、`checkTaskPlanApproved`、`checkApprovedPlanHeader`、`approvedLineDigest`、`approvalNextStep`、`planNotApprovedError`)、`internal/org/approve.go`(`ApproveParams` から `resolveApproved` まで)、`internal/cli/org_approve.go`、`internal/org/verbs.go`(`Send` の中の検査の位置、`sendApprovalErr`、`taskNotSentError`)、`internal/org/spawn.go`(`spawnFeatureSeatErr` とその 3 つの呼び出し元)、`internal/org/prompts/leader.md` の「機能ごとの org」節、4 つの `/org` SKILL.md の承認の節、`docs/quality/quality-gates.md` の新しい行、`docs/tech-debt/README.md` の 213 行目と 217 行目、`templates/base/ralph.toml` の `[org.approval]`、1b95bb84 で計画の Progress checklist に足したメモ
- テストの差分は書き方だけを見た: `approval_test.go`(`TestApprovedLineDigest`、`TestPlanNotApprovedError_OneNextStepPerKind`)、`approve_test.go`(`TestOrgApprove_ResolvedBySomeoneElse`)、`approval_enforce_test.go`、`prompts_test.go`
- 機械的に確かめたこと
  - 54feb287 が移しただけであること: `git show 54feb287 -- internal/org/approval.go` の `-` 行(375 行)と、`-- internal/org/approve.go` の `+` 行(392 行)を取り出して `diff` した。違いは追加の側の package 行、import の 11 行、ファイル頭のコメント 3 行と空行だけだった。テストも同じ方法で比べ、違いは package 行、import、2 つのファイルの頭のコメントだけだった。コミットの本文は新しいファイルのコメントが 2 つとだけ書くが、`approval_test.go` の頭のコメント 4 行も書き換わっている。中身の違いではないので指摘にしない
  - 拒否を作る場所: `planNotApprovedError` を作るのは `spawnFeatureSeatErr`(`internal/org/spawn.go:1294`)と `sendApprovalErr`(`internal/org/verbs.go:503`)の 2 か所だけである。`spawnFeatureSeatErr` は本番のロックの中(`spawn.go:546`)、dry-run(`:1192`)、`spawnPrecheckErr`(`:1227`)の 3 か所から同じ events で呼ばれる。Send の検査は dry-run の枝より前にある(`verbs.go:316-329`)。種類は `checkPlanApproved` の各 return と `checkApprovedPlanHeader` で決まり、`checkAnyPlanApproved` は最後の承認の種類と digest を受け継ぐ。種類を持たない error は Other(BLOCKED)に倒れる
  - 直しの範囲で足した Go のコメントにある CamelCase の識別子 77 個は、どれもコメント以外の行に定義か使い道がある(`git grep -w`)
  - 4 つの `/org` SKILL.md は md5 が同じ。`docs/quality/quality-gates.md` と `templates/base/` の写しは、新しい行(81 行目と 80 行目)が同じで、ほかの違いはこの PR より前からある 54 行目と 57 行目の 2 か所だけ。2 つの `agent-messaging.md` は `cmp` で同じ
  - tech-debt の 213 行目と 217 行目は、エスケープしていない `|` がどちらも 6 本で、列は崩れていない。217 行目は (d) を Debt item、Impact、Trigger のどこからも消してある。この PR で足した行なので、取り消し線ではなく消すのでよい
  - 足したコメントの行はどれも 80 桁以内(awk が 81 と数えた 1 行は多バイト文字のため)
  - HEAD は読み始めと書く前のどちらも 1b95bb84 で、作業ツリーはきれいだった
- 確かめていないもの: テスト、静的解析、実機の確認はしていない。N1 と N3 はコードを読んで組み立てたもので、動かしてはいない

## 依頼された点の結論

1. 種類ごとの次の手は、検査が見つけたことに合っているか: 5 種類のうち 4 つは合っている。PlanHeader(先頭の 2 行)は、記録があり、digest も全文のハッシュも合うときにしか出ない。2 行はどちらのハッシュにも入らないので、書き直して打ち直せば通る。TaskHeader は `ParsePlanHeader` の誤りだけで、ヘッダを直して送り直せば次の段へ進む。Other は結びつきの不完全、worktree の外を指す symlink、ファイルがないか読めないときで、leader は自分で直せないので BLOCKED でよい。合わないのは、TASK の `PLAN:` が古い digest か打ち間違えた digest を名指すときである。この場合は NoApproval か PlanChanged に振り分けられ、どちらの次の手も実際の原因(ヘッダの digest)に向かない(N1)
2. 手順 7 に従う leader が止まらなくなるか: N1 の (a)、承認し直した計画に古い digest の TASK を送る場合がそうなる。次の手の「新しい QUESTION を上げる」に従っても TASK のヘッダは変わらないので、TASK を送るたびに同じ拒否が繰り返され、そのたびに人に QUESTION が届く。もう一方の「変更を戻す」に従うと、人が最後に承認した計画が古い版に戻り、古い承認(同じ命の中なので数えられる)で implementer が進む。ほかに、NoApproval の「QUESTION の返事を待つ」は、QUESTION を上げていない leader にとっては終わりがない。これは手順 3 を飛ばしたときだけで、deadman の escalation が人に知らせるので、指摘にはしない
3. 5 種類の振り分け: Evidence のとおり、6 つの場所(`checkTaskPlanApproved`、`checkAnyPlanApproved`、Send、Spawn の本番、dry-run、`spawnPrecheckErr`)はどれも `sendApprovalErr` か `spawnFeatureSeatErr` を通るので、同じ入力には同じ種類がつく。spawn には `PLAN:` がないので TaskHeader は出ず、`--raw` の一文も付かない(`planNotApprovedNeverRaw` は `task` のときだけ付く)
4. `- Approved:` を厳しく読むことで、`/plan` か雛形が書く行を断るか: 断らない。どちらも `<YYYY-MM-DD> sha256:<digest>` を書き、`scripts/plan-visual.sh digest` は `sha256sum` の小文字の hex を出す。断るのは `TBD`、語を足した行、存在しない日付、大文字の hex で、どれも雛形の手順では書かない。ただ `/implement` は同じ行の `sha256:` の値だけを見るので、`/implement` が承認済みとみなす計画の一部を org の検査は断る。このずれを `featurePlanHeaderLines` の doc が言っていない(N5)
5. L3 の `ErrInboxResolved` の文: 失敗するコマンドを出さなくなった点は直った。ただし、先に resolve した人が直しを頼むつもりだった場合に、その人のメモを読む手を示さない(N3)
6. 54feb287: Evidence のとおり、移しただけである
7. 文書の面: 4 つの SKILL.md、quality-gates.md の 2 つの面、tech-debt の 213 行目と 217 行目、`templates/base/ralph.toml` は、コードの 5 種類と次の手、`--raw` の一文が TASK だけに付くこと、`- Approved:` の形と合っている。合わないのは、雛形と SKILL.md の「同じコマンドをそのまま打ち直さず」と、PlanHeader の次の手の「打ち直す」の組み合わせである(N2)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | readability | N1: TASK の `PLAN:` が、今のファイルのものでない digest を名指すと、拒否の種類は検査が最初につまずいた段で決まり、次の手が原因に向かない。(a) 手順 6 で計画を直して新しい digest Z の承認を受けたあと、前の digest Y のヘッダで TASK を送ると、Y の記録があるので PlanChanged になり、次の手は「変更を戻すか、Draft に戻して新しい QUESTION を上げる」になる。Z の記録が同じ命にあることを、検査は読まない。戻せば人が最後に承認した計画を捨てて Y で進み、頼み直せば承認済みの計画の QUESTION が人に届く。どちらでもヘッダは Y のままなので、次の TASK も同じように断られる。(b) digest を打ち間違えると NoApproval になる。理由の文は「(its latest approval is of sha256:Z)」と書くのに、DECISION を受けた leader への次の手は「その DECISION は記録なしに来たので BLOCKED を上げる」で、記録はあるので原因の説明が誤っている。雛形の手順 5 の「手順 4 で承認された digest」も、承認し直したあとにどちらの digest を使うかを言わない。どちらに転んでも承認のない計画では進めないので、安全の問題ではない | `internal/org/approval.go:465-483`(`matching` は TASK の digest の記録だけを集め、`latest` は理由の文にしか使わない)、`:498-501`(PlanChanged)、`:592-594` と `:601-603`(次の手)、`internal/org/prompts/leader.md:138`(手順 5)、`:148-150`(手順 6) | `checkPlanApproved` で、TASK の digest に記録がないとき、またはファイルがその digest から変わっているときに、同じパスに今のファイルの digest の記録があるかを見る。あれば TaskHeader の種類にし、次の手で「`PLAN: <path> sha256:<今の digest>` に直して送り直す」と digest を名指す(`approvalCheckError.digest` が使える)。コードを変えないなら、NoApproval と PlanChanged の次の手の先頭に「まず TASK の `PLAN:` が最後に承認された digest かを確かめる」を足し、手順 5 を「最後に承認された digest」にする。どちらの場合も、`TestPlanNotApprovedError_OneNextStepPerKind` に、承認し直した計画と古いヘッダの組を足す |
| LOW | readability | N2: 打ち直しを禁じる文と、打ち直しを求める次の手が、同じ拒否について並ぶ。雛形の手順 7 と SKILL.md は「どの場合も、同じコマンドをそのまま打ち直さず」と書き、PlanHeader の次の手は「2 行を書いてコミットし、tries again」と言う。spawn では打ち直すのは引数も同じ `ralph org spawn` なので、「そのまま」の読み方によっては禁じられた操作になる。ralph が出す拒否の文の終わり(`planNotApprovedNever`)は打ち直しに触れないので、この禁止は雛形と SKILL.md にしかない。また PlanChanged の次の手は、雛形の手順の番号を「step 6 of its procedure」と Go の文字列に書いている。雛形の番号が変わってもテストは気づかない(この PR でも番号は 1〜14 に繰り下がった) | `internal/org/prompts/leader.md:154`、`.claude/skills/org/SKILL.md:673-674` と `:682`(写しの 3 つも同じ)、`internal/org/approval.go:599-600`(tries again)、`:603`(step 6) | 雛形と SKILL.md の文を「次の手をする前に、同じコマンドを打ち直さず」にする。「step 6 of its procedure」は、手順の中身(Draft に戻して新しい QUESTION)がすでに同じ文にあるので番号を消すか、`prompts_test.go` で Go の文字列の番号と雛形の手順の頭を突き合わせる |
| LOW | exception-handling | N3: L3 の残り。approve が受信箱を読んだあと、別の人が直しを頼むつもりで件を resolve すると、承認はそのまま記録され、approve は「承認は記録してある、resolve するものは残っていない」と言って終了コード 1 を返す。CLI は同じ件の DECISION のコマンドも出すので、approve した人がそれを打つと、leader には同じ `PLAN:` の承認の DECISION と直しの DECISION が両方届く。文は、先に resolve した人のメモを読む手(`ralph org inbox show <ID>`)を示さない。この競合は tech-debt にもない。テストは競合する resolve のメモに「handled by hand」を使うが、その場合の扱いは確かめていない。2 人が同じ件を同時に扱うときだけの話である | `internal/org/approve.go:399-401`、`internal/cli/org_approve.go:104-106`(承認した件ごとに DECISION のコマンドを出す)、`internal/org/approve_test.go:432` と `:447` | 文に「`ralph org inbox show <ID>` で先に resolve した人のメモを読んでから、下の DECISION を送る」を足す。直さないなら tech-debt に 1 行足す |
| LOW | maintainability | N4: L8 の残り。`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` は、中身がほぼ同じまま残った。9fb3fafa の本文と計画のメモは、L8 を直したとだけ書く | `internal/org/approval_test.go:483-497`、`internal/org/approval_enforce_test.go:44-61`、9fb3fafa の本文の「L8: TestCheckPlanApproved uses replaceOnce instead of its own helper」 | 2 つの fixture の土台が違う(fake の driver があるかどうか)ので分けておくなら、その理由を helper の doc に書く。まとめるなら、承認の記録と 2 行を書く helper を、manifest と worktree を引数に取る関数 1 つずつにする |
| LOW | readability | N5: `featurePlanHeaderLines` の doc は、2 行を「/implement の承認の検査が計画を読むのと同じように」読むと書く。L2 の直しで、org の検査は `- Approved:` の値を 2 語に限った。一方 `/implement` は同じ行の `sha256:` の値だけを見るので、`- Approved: 2026-10-12 sha256:<digest> (再承認)` のような行を `/implement` は通し、org の検査は断る。行の探し方は同じで、値の読み方が違うことを doc が言っていない | `internal/org/approval.go:191-195`、`.claude/skills/implement/SKILL.md:34`、`approvedLineDigest`(`internal/org/approval.go:237-257`) | doc を「行の探し方は /implement と同じ(最初の `## ` より前の最初の行)で、`- Approved:` の値は approvedLineDigest がより厳しく読む」とする |

## Positive notes

- 種類を error の型(`approvalCheckError`)で運び、理由の文と次の手を別の場所で作るようにしたので、理由の文に次の手が混ざらないことを、テストが種類ごとに確かめられる。種類を持たない error は zero value の Other(BLOCKED)に倒れる
- `TestPlanNotApprovedError_OneNextStepPerKind` は、どの次の手の文がどの種類に出るかを全部の組み合わせで確かめ、最後に 5 種類すべてにケースがあることを数える。次の手を 1 つ足してケースを足し忘れると落ちる
- `TestOrgApprove_ResolvedBySomeoneElse` は `Org.Now` を差し替え、受信箱を読んだあと resolve の前に別の resolve を割り込ませる。割り込みが起きなければ `t.Fatal` で落ちるので、テストが空振りしない
- 54feb287 は移すだけのコミットで、直しは次の 9fb3fafa に分けてある。そのため、移しただけであることを `diff` 1 回で確かめられた
- `--raw` の一文を TASK の拒否だけに付け、spawn の拒否から外した。tech-debt の 217 行目の (d) はこれで閉じ、行からも消えている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| N1(直さないときだけ): TASK の `PLAN:` が今のファイルのものでない digest を名指すと、承認の検査は NoApproval か PlanChanged で断り、次の手が「変更を戻す」「新しい QUESTION を上げる」「BLOCKED を上げる」のどれかになる。同じパスに今の digest の記録があっても読まない | 計画を承認し直したあとに古いヘッダで TASK を送る leader が、承認済みの計画を古い版に戻すか、TASK を送るたびに同じ QUESTION を人に上げる | この cycle で直すかは orchestrator が決める | 次に `checkPlanApproved` か `approvalNextStep` を変えるとき、または leader が承認済みの計画の QUESTION を上げ直した例が出たとき | この報告の N1、`internal/org/approval.go`(`checkPlanApproved`、`approvalNextStep`) |
| N2〜N5(直さないときだけ): 打ち直しの文の食い違いと Go の文字列の手順の番号(N2)、approve と直しを頼む resolve の競合で先に resolve した人のメモを示さないこと(N3)、テストの fixture の helper の重なり(N4)、`featurePlanHeaderLines` の doc の /implement との比べ方(N5) | N3 は、2 人が同じ件を同時に扱うと leader に食い違う DECISION が届く。ほかは読み手が迷うだけで、挙動は変わらない | どれも LOW で、挙動を変えるのは N3 だけ | 次にそれぞれのファイル(`internal/org/prompts/leader.md` と `/org` SKILL.md、`internal/org/approve.go`、`internal/org/approval_test.go` と `approval_enforce_test.go`、`internal/org/approval.go`)を変えるとき | この報告の N2〜N5 |

_(この 2 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。直さずに残すなら orchestrator が足す。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。MEDIUM は N1 の 1 件で、PR の前に直すことを勧めるが、止める理由にはしない。LOW は N2〜N5 の 4 件)
- Follow-ups:
  - N1: `checkPlanApproved` で同じパスの今の digest の記録を見て、TASK のヘッダを直す次の手を出す。直さないなら、次の手と雛形の手順 5 の文を直し、上の 1 行目を tech-debt に足す
  - N2〜N5: この cycle でまとめて直せる。直さないなら上の 2 行目を tech-debt に足す
  - /verify への申し送り: 計画の AC10 の読み方と、Scope・Risks の「BLOCKED を escalate する」の書き方は、1b95bb84 のメモ(逸脱 2)が扱っている
  - insight event: 同じ cycle の 2 回目なので足していない。phase と cycle で集計するので、足すと self_review の cycle 1 が 2 回数えられる。最初の結果の event は 6d3369a6 にある

## 付録: cycle 1 の最初の結果(3d73dbf4、HEAD ef503bbd の時点)

最初の結果の本文を、見出しを 1 段下げ、`Merge:` の行の名前を「最初の結果の判定」に変えたほかは、そのまま残す。

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: `git diff 225dff27...HEAD`(HEAD ef503bbd)。コミットは f779cc56(設定と結びつきの `approval=`)、c08a1a14(`internal/org/approval.go`)、e3d34062(Send と Spawn の拒否)、5596561a(`ralph org approve`)、36b9d3a3(leader の雛形と文書)、ef503bbd(計画のメモ)。見たのは差分の質だけで、仕様との突き合わせ、テストの網羅、文書のずれは `/verify` と `/test` に任せる

### Evidence reviewed(最初の結果)

- 全文を読んだコード: `internal/org/approval.go`(853 行)、`internal/cli/org_approve.go`
- 差分を読み、周りのコードも読んだもの: `internal/org/verbs.go`(`Send`、`sendApprovalErr`、`seatRoleFromEvents`、`seatFromEvents`、`taskNotSentError`)、`internal/org/spawn.go`(`Spawn` の doc と dry-run の枝、`checkCapacityAndStart`、`spawnPrecheckErr`、`spawnFeatureSeatErr`、`idempotentRespawn`、`reject`、`checkSpawnInput`)、`internal/org/reserve.go`(`FeatureBinding`、`RequiresApproval`、`featureBindingFromTokens`、`sameFeature`、`validateFeatureBinding`、`ActiveFeature`、`lastDisbandedIndexes`)、`internal/org/feature.go`、`internal/org/protocol/protocol.go`(`Parse` と新しい `HeaderValues`)、`internal/org/split.go`(`PlanDigest`、`planLines`、`checkedBoxIndex`、`approvedDigestFrom`)、`internal/org/manifest.go`(`Roster`)、`internal/org/inbox.go`(`PrintableInboxText`)、`internal/cli/org.go`(`withMainWorktreeOrgLimits`)、`internal/cli/org_inbox.go`(`newOrgInboxRuntime`)、`internal/config/config.go`、`scripts/ralph-config.sh`
- 文書の差分で読んだもの: `internal/org/prompts/leader.md`、`templates/base/ralph.toml`、`docs/tech-debt/README.md` の新しい 5 行、`docs/quality/quality-gates.md`、`.claude/rules/ralph/agent-messaging.md`、`/org` skill の追加行のうち承認に触れる行
- テストの差分は、helper の重複と書き方だけを見た(`internal/org/approval_test.go`、`internal/org/approval_enforce_test.go`、`internal/cli/org_approve_test.go`、`internal/org/prompts_test.go`、`internal/cli/org_feature_test.go`)
- 機械的な確認:
  - 追加行にデバッグ出力・TODO・secret らしい文字列はない(`git diff ... | grep '^+'` で `TODO|FIXME|fmt.Print|println|t.Log|api_key|token=|password` を探して 0 件)
  - 新しい `func` の直上の doc はどれも、その関数の直上にある
  - `docs/tech-debt/README.md` の新しい 5 行は、エスケープしていない `|` がどれも 6 本で、列は崩れていない
  - `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` は `cmp` で一致する。`RALPH_ORG_APPROVAL_MODE` は export の行に入っていない
  - `approval.go` の関数で 50 行を超えるものはない(いちばん長いのは `Approve` の 43 行)。`org_approve.go` の `newOrgApproveCmd` は 60 行だが、その大半は `Long` の文字列である
  - S1 で implementer が `gofmt -w` をかけた `internal/cli/org_feature_test.go` の差分は追加の 95 行だけで、既存の行の書式だけの変更は入っていない
- 確かめていないもの: テスト、静的解析、実機での確認はしていない。下の指摘のうち、競合(L3)と迂回(L1、L2)は、コードを読んで組み立てたもので、動かしてはいない

### 迂回と誤判定の確認(依頼された観点)(最初の結果)

1. パスの解決と symlink: `normalizePlanPath` は空、絶対パス、制御文字、`..` の要素、`.` を断り、`filepath.Clean` と `ToSlash` で揃える。`resolvePlanFile` は worktree とファイルの両方を `EvalSymlinks` で解き、`filepath.Rel` が `..` で始まれば断り、普通のファイルでなければ断る。worktree の中から外を指す symlink は断られる。ファイルを解いたあと `os.ReadFile` までの間に差し替える TOCTOU は残るが、A2 のとおり境界ではないので指摘にしない
2. ヘッダの読み方: `HeaderValues` は `Parse` と同じ規則(1 行目が `TYPE:`、空行か `KEY: value` でない行で終わる、キーと値を trim する)で読む。`PLAN:` が 2 行あれば断るので、検査と座席が読む行はずれない。キーの大文字小文字(`Plan:`)と本文に書いた別の計画は検査の外で、L1 に含めた
3. 前の命の承認: `currentPlanApprovals` は最後の実の `disbanded` より後の、org 全体の(seat_id が空の)実の `plan_approved` だけを読む。読めない Details の記録は数えない。disband の補償で承認が数えられなくなる件は、閉じる側に倒れており、tech-debt の行にもある
4. 読めない方式の印: `approvalModeUnreadable` は `per-plan` でも `split` でもないので、`RequiresApproval` は承認を求める。新しい結びつきには `validateFeatureBinding` が使わせない。補償で書き戻しても同じ値が読み戻される。`approval=` のない記録だけが split として通り、これは A8 で決めた扱いである。古い版の補償が結びつきを書き直すと `approval=` が落ちる(古い版はこのトークンを note として読み、書き直すときに note を落とす)が、版を混ぜたときだけの話で、release のノートが扱う範囲と読んだ
5. split と古い記録の枝: `sendApprovalErr` と `spawnFeatureSeatErr` は、記録した方式だけを読み、`o.Config` を読まない。leader の spawn だけを外す条件は `SeatID == leader && Role == leader` の両方で、片方だけでは外れない
6. `Org.Approve` の書き込み: ロックを 1 回取り、その中で manifest を読み直し、全部の件を確かめてから追記する。全文のハッシュは、digest を確かめたのと同じ読み込みから取る。1 件でも断られれば何も追記しない。追記の途中で I/O が失敗したときは、記録した件と残りの件と打ち直すコマンドを返し、`ApproveResult` の doc もそう書いている。resolve はロックの外で、失敗しても記録は残る。受信箱をロックの外で読むことは L3 に書いた
7. 座席の role の引き方: `seatRoleFromEvents` は role を持つ最後の状態の event を読むので、`spawn_step` で止まった座席も implementer として検査される。`rejected` が role を上書きしても、`Roster` の最後の状態の event に pane がなくなるので、`Send` の本番の枝は pane のない座席として断る。この順で検査を外す道はない

### Findings(最初の結果)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | readability | M1: 承認の検査の拒否の文で、理由の部分と最後の決まり文句が、leader に別々の次の手を言う。理由の部分は「新しい QUESTION で承認を頼み直す」「先頭の 2 行を書いてコミットする」「変更を戻すか、頼み直す」と言い、続く `planNotApprovedNext` は「打ち直さず、`--raw` にせず、approve を打たず、BLOCKED を escalate する」と言う。記録はあるが leader が 2 行を書き忘れた場合、leader は自分で直せるのに、同じ文の後ろ半分が打ち直しを禁じる。leader は LLM で、エラーの文を字義どおりに読む。leader の雛形も、手順 6 は計画を変えたら新しい QUESTION、手順 7 は断られたらいつも BLOCKED と書いており、どちらに従うかが決まらない。どちらに転んでも承認なしでは通らないので、安全の問題ではない | `internal/org/approval.go:381`(記録がないとき)、`:400`(digest が変わったとき)、`:403-418`(2 行がないか違うとき。`write := ...` の文)、`:425`(全文のハッシュが違うとき)、`:465`(`planNotApprovedNext`)、`internal/org/prompts/leader.md` の手順 4・6・7 | 次の手を書く場所を 1 つにする。理由の部分は状態だけを言い(何が足りないか)、次の手は理由ごとに変わる末尾に移す。たとえば 2 行がないときは「2 行を書いてコミットし、送り直す」、digest か全文のハッシュが違うときは「手順 6 のとおり新しい QUESTION を上げる」、記録がないときは「返事を待つか BLOCKED を上げる」。leader の雛形の手順 7 も同じ分け方に合わせる。tech-debt の (d)(spawn の拒否が `--raw` に触れる)も同じ直しで閉じられる |
| LOW | security | L1: 検査は role の名前 `implementer` だけを見るので、別の名前の role で実装させると承認を通らずに済む。`checkSpawnInput` は任意の role を受け、`--prompt` を付ければ雛形のない custom role として起動する(`spawn.go:1375` の doc がそう書く)。その座席への TASK も `seatRoleFromEvents` が implementer を返さないので検査されない。ほかに、reviewer への TASK(A5 で通す)で実装を頼むことも、`PLAN:` には承認済みの計画を書いて本文で別の作業を頼むこともできる。A2 のとおり境界ではないが、tech-debt の tripwire の行は、迂回の道として `env -u HERDR_PANE_ID` と台帳への直接の書き込みしか挙げていない | `internal/org/spawn.go:1293`、`internal/org/verbs.go:498`、`internal/org/spawn.go:1371-1376`、`docs/tech-debt/README.md` の新しい行のうち `approveTripwire` の行 | コードは変えず、tech-debt の tripwire の行の Debt item に「検査は role の名前と `PLAN:` のヘッダを見るだけで、座席がする作業は見ない」と、上の 3 つの道を足す |
| LOW | security | L2: A9 の残り。全文のハッシュは、見出しの前の最初の `- Approved:` の行をまるごと飛ばし、検査はその行の最初の `sha256:` の値だけを読む。そのため承認のあとに、その 1 行の日付の位置や digest の後ろに文を書き足しても、検査は通る(例: `- Approved: 2026-10-13 sha256:<digest> あわせて X も実装する`)。`- Status:` は値が `Approved` と完全に一致する必要があるので、こちらは閉じている | `internal/org/approval.go:229`(`approvedDigestFrom` に渡すだけ)、`internal/org/split.go:463-473`、`PlanContentHash`(`approval.go:250-264`) | `readFeaturePlanHeader` で、`- Approved:` の値が `<日付> sha256:<digest>` の 2 語だけであることを求める。直さないなら、skill の承認の節か tech-debt に、この 1 行が残ることを書く |
| LOW | exception-handling | L3: `Approve` は受信箱をロックの外で 1 回読むだけで、ロックの中で件の状態を読み直さない。同じ件に 2 人が同時に approve を打つと、どちらも記録し(`plan_approved` が 2 行になる)、遅いほうの resolve は失敗する。そのとき出す案内 `ralph org inbox resolve <ID> --note ...` は、件がもう resolved なので、打っても失敗する。approve と、人が直しを頼むつもりで打った `ralph org inbox resolve` が競合したときも、承認が記録される。どちらも窓は短い | `internal/org/approval.go:568`(受信箱を読む)、`:603-621`(`recordUnderLock` は manifest だけを読み直す)、`:836-853`(`resolveApproved` の案内) | ロックの中で追記の直前に受信箱を読み直して状態を確かめるか、resolve の失敗が `ErrInboxResolved` のときは「ほかの誰かが resolve した。承認は記録してある」と言い分け、打っても失敗するコマンドを案内しない |
| LOW | maintainability | L4: `approval.go` が 853 行で、ファイルの目安の 800 行を超える。中身は 3 つに分かれている: `PLAN:` のヘッダとパスの読み方とハッシュ、検査(`checkPlanApproved` など)、approve の動詞(`ApproveParams` から `resolveApproved` まで、373 行) | `wc -l internal/org/approval.go` | approve の動詞とその helper を `internal/org/approve.go` に分ける。どちらも 600 行を切る |
| LOW | maintainability | L5: `ApproveResult.Refused` と `ApprovedPlan.Resolved` を読むのはテストだけである。CLI は `Approved` と `Err` しか読まない | `internal/cli/org_approve.go:72-78`(`printApprovedPlans` に渡すのは `res.Approved`、返すのは `res.Err`)、`grep -rn 'Refused\|Resolved' internal/cli/org_approve.go` が 0 件 | CLI の出力に使う(たとえば resolve できなかった件に印を付ける)か、テストのための欄だと doc に書く |
| LOW | unnecessary-change | L6: 作業ディレクトリの取り方が 2 か所にある。CLI は `os.Getwd()` が失敗すると `""` を渡し、`Approve` は `""` を受けると `os.Getwd()` をもう一度呼ぶ。同じ呼び出しを 2 回試すだけで、2 回目が成功する場面はほぼない | `internal/cli/org_approve.go:65-71`、`internal/org/approval.go:574-581` | どちらか一方にする。CLI は `ApproveParams.Cwd` を空のまま渡し、`Approve` に任せれば、CLI 側の 4 行と注記が消える |
| LOW | readability | L7: コメントの折り返し直しが途中で止まっている。`spawn.go` は `appendEvent" window` のあとで行が短く切れ、次の行に `docs/tech-debt/README.md` だけがある(切れ方は base にもあったが、この差分で 3 行目が増えた)。`reserve.go` は `before. When orgID holds none, a want with` で短く切れる。`prompts_test.go` には、ほかより長い行が 1 行ある | `internal/org/spawn.go:557-560`、`internal/org/reserve.go:531`、`internal/org/prompts_test.go:467` | 段落ごとに折り返し直す |
| LOW | maintainability | L8: 同じ package のテストに、同じ働きの helper が重なっている。「置き換える文字列がなければ panic する」置き換えが `replaceOnce` とテストの中の `replace` の 2 か所にあり、`checkFixture` の `record`・`markApproved` と `enforceFixture` の `record`・`mark` もほぼ同じ中身である | `internal/org/approval_enforce_test.go:44-74`、`internal/org/approval_test.go:451-475` | `replaceOnce` と、承認の記録と 2 行を書く helper を 1 つずつにまとめる |
| LOW | readability | L9: `templates/base/ralph.toml` の `[org.approval]` のコメントが「Absent [org.approval] is fully compatible」と書く。`[org.director]` の節から写した決まり文句だが、この節は、ないと既定の `per-plan` になり、既存のプロジェクトの `start --plan` の org は `ralph org approve` を待つようになる。同じコメントの 2 行上は、`split` が「as before this setting existed」だと書いている。`ralph init` で配る面にある | `templates/base/ralph.toml:127-128` | 「Absent [org.approval] loads as the default per-plan; set mode = "split" to keep the behavior from before this setting」のように、既定で動きが変わることを書く |
| LOW | naming | L10: `taskNotSentError` の欄の名前が `gate` のままで、見張りの拒否のほかに承認の検査の拒否も入るようになった。doc は直してあるが、欄の名前だけが古い | `internal/org/verbs.go:226` | `refusal` などに改める(同じ package の中だけの変更) |
| LOW | readability | L11: leader の雛形の前置きは、予約の外に書くものとして「手順 1 と 4 の機能の計画」を挙げるが、手順 6(計画を `- Status: Draft` に戻して直す)も計画を書く | `internal/org/prompts/leader.md:62-64`、同じファイルの手順 6 | 「手順 1・4・6 の機能の計画」とする。`prompts_test.go` の固定の語句も合わせる |

### Positive notes(最初の結果)

- 承認の記録は 1 回のロックの中で、読み直し、全部の件の検査、追記の順に行う。全文のハッシュは、digest を確かめたのと同じ読み込みから取るので、記録した中身と確かめた中身がずれない
- 読めない方式、不完全な結びつき、方式のない新しい結びつき(`featureApprovalMode` が `""` を `per-plan` にする)は、どれも承認を求める側に倒れる。`approval=` のない古い記録だけを split として通し、その理由が `RequiresApproval` の doc にある
- `HeaderValues` は `Parse` の規則をそのまま写し、`PLAN:` が 2 行あるときを数えられるようにした。A7 の理由(後の行で上書きされる)は doc に書いてある
- `Send` の検査は dry-run の枝の前、`Spawn` の検査は本番と dry-run と `spawnPrecheckErr` の 3 か所で同じ順に置かれ、`Spawn` の doc の手順 7 と本文のコメントが一致している
- `seatRoleFromEvents` は、`spawn_step` が role を持たないために `Roster` の role が空になる場合を拾っており、その理由が doc にある
- 新しい tech-debt の 5 行は、どれも Trigger が具体的で、列も崩れていない

### Tech debt identified(最初の結果)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L1: 承認の検査は role の名前 `implementer` と TASK の `PLAN:` のヘッダを見るだけで、座席がする作業は見ない。custom role(`--role <名前> --prompt ...`)の座席、reviewer への TASK、承認済みの `PLAN:` を付けて本文で別の作業を頼む TASK は、承認なしで実装を進められる | leader が誤って、または注入された指示で、人が見ていない作業を始められる。A2 で境界ではないとした範囲の中の話 | A2 で境界を作らないと決めており、tripwire の行と同じ扱い。行を足すだけで、コードは変えない | 既存の tripwire の行と同じ(8a 段目、または本物の境界が要るとき) | この報告の L1、`internal/org/spawn.go:1293`、`internal/org/verbs.go:498` |
| L2: 計画の見出しの前の最初の `- Approved:` の行は、両方のハッシュから外れ、検査は最初の `sha256:` の値しか読まないので、承認のあとにこの 1 行へ文を書き足せる | 計画の中で 1 行だけ、承認のあとに指示を足せる場所が残る | 直すなら `readFeaturePlanHeader` の小さな変更で済む。この cycle で直すかは orchestrator が決める | 次に `readFeaturePlanHeader` か `PlanContentHash` を変えるとき | この報告の L2 |

_(この 2 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。直さずに残すなら、orchestrator が足す。M1 は tech-debt の (d) の行と合わせて直すか、その行に足す。)_

### Recommendation(最初の結果)

- 最初の結果の判定: yes(CRITICAL と HIGH はない。M1 は PR の前に直すことを勧めるが、止める理由にはしない)
- Follow-ups:
  - M1: 拒否の文の次の手を理由ごとに 1 つにし、leader の雛形の手順 7 を合わせる。直さないなら tech-debt の (d) の行に足す
  - L1・L2: 直さないなら上の 2 行を `docs/tech-debt/README.md` に足す
  - L3〜L11: 次にこのファイルを触るときにまとめて直せる
  - insight event: 依頼の指示でこの報告のほかには何も書いていないので、`./scripts/insights-append.sh --slug org-plan-approval --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 11 --cycle auto --source skill || true` は打っていない
