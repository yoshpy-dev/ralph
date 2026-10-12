# Self-review report: org-plan-approval

- Date: 2026-10-13
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Reviewer: reviewer subagent (Claude)。パイプライン 1 回目(cycle 1)
- Scope: `git diff 225dff27...HEAD`(HEAD ef503bbd)。コミットは f779cc56(設定と結びつきの `approval=`)、c08a1a14(`internal/org/approval.go`)、e3d34062(Send と Spawn の拒否)、5596561a(`ralph org approve`)、36b9d3a3(leader の雛形と文書)、ef503bbd(計画のメモ)。見たのは差分の質だけで、仕様との突き合わせ、テストの網羅、文書のずれは `/verify` と `/test` に任せる

## Evidence reviewed

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

## 迂回と誤判定の確認(依頼された観点)

1. パスの解決と symlink: `normalizePlanPath` は空、絶対パス、制御文字、`..` の要素、`.` を断り、`filepath.Clean` と `ToSlash` で揃える。`resolvePlanFile` は worktree とファイルの両方を `EvalSymlinks` で解き、`filepath.Rel` が `..` で始まれば断り、普通のファイルでなければ断る。worktree の中から外を指す symlink は断られる。ファイルを解いたあと `os.ReadFile` までの間に差し替える TOCTOU は残るが、A2 のとおり境界ではないので指摘にしない
2. ヘッダの読み方: `HeaderValues` は `Parse` と同じ規則(1 行目が `TYPE:`、空行か `KEY: value` でない行で終わる、キーと値を trim する)で読む。`PLAN:` が 2 行あれば断るので、検査と座席が読む行はずれない。キーの大文字小文字(`Plan:`)と本文に書いた別の計画は検査の外で、L1 に含めた
3. 前の命の承認: `currentPlanApprovals` は最後の実の `disbanded` より後の、org 全体の(seat_id が空の)実の `plan_approved` だけを読む。読めない Details の記録は数えない。disband の補償で承認が数えられなくなる件は、閉じる側に倒れており、tech-debt の行にもある
4. 読めない方式の印: `approvalModeUnreadable` は `per-plan` でも `split` でもないので、`RequiresApproval` は承認を求める。新しい結びつきには `validateFeatureBinding` が使わせない。補償で書き戻しても同じ値が読み戻される。`approval=` のない記録だけが split として通り、これは A8 で決めた扱いである。古い版の補償が結びつきを書き直すと `approval=` が落ちる(古い版はこのトークンを note として読み、書き直すときに note を落とす)が、版を混ぜたときだけの話で、release のノートが扱う範囲と読んだ
5. split と古い記録の枝: `sendApprovalErr` と `spawnFeatureSeatErr` は、記録した方式だけを読み、`o.Config` を読まない。leader の spawn だけを外す条件は `SeatID == leader && Role == leader` の両方で、片方だけでは外れない
6. `Org.Approve` の書き込み: ロックを 1 回取り、その中で manifest を読み直し、全部の件を確かめてから追記する。全文のハッシュは、digest を確かめたのと同じ読み込みから取る。1 件でも断られれば何も追記しない。追記の途中で I/O が失敗したときは、記録した件と残りの件と打ち直すコマンドを返し、`ApproveResult` の doc もそう書いている。resolve はロックの外で、失敗しても記録は残る。受信箱をロックの外で読むことは L3 に書いた
7. 座席の role の引き方: `seatRoleFromEvents` は role を持つ最後の状態の event を読むので、`spawn_step` で止まった座席も implementer として検査される。`rejected` が role を上書きしても、`Roster` の最後の状態の event に pane がなくなるので、`Send` の本番の枝は pane のない座席として断る。この順で検査を外す道はない

## Findings

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

## Positive notes

- 承認の記録は 1 回のロックの中で、読み直し、全部の件の検査、追記の順に行う。全文のハッシュは、digest を確かめたのと同じ読み込みから取るので、記録した中身と確かめた中身がずれない
- 読めない方式、不完全な結びつき、方式のない新しい結びつき(`featureApprovalMode` が `""` を `per-plan` にする)は、どれも承認を求める側に倒れる。`approval=` のない古い記録だけを split として通し、その理由が `RequiresApproval` の doc にある
- `HeaderValues` は `Parse` の規則をそのまま写し、`PLAN:` が 2 行あるときを数えられるようにした。A7 の理由(後の行で上書きされる)は doc に書いてある
- `Send` の検査は dry-run の枝の前、`Spawn` の検査は本番と dry-run と `spawnPrecheckErr` の 3 か所で同じ順に置かれ、`Spawn` の doc の手順 7 と本文のコメントが一致している
- `seatRoleFromEvents` は、`spawn_step` が role を持たないために `Roster` の role が空になる場合を拾っており、その理由が doc にある
- 新しい tech-debt の 5 行は、どれも Trigger が具体的で、列も崩れていない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L1: 承認の検査は role の名前 `implementer` と TASK の `PLAN:` のヘッダを見るだけで、座席がする作業は見ない。custom role(`--role <名前> --prompt ...`)の座席、reviewer への TASK、承認済みの `PLAN:` を付けて本文で別の作業を頼む TASK は、承認なしで実装を進められる | leader が誤って、または注入された指示で、人が見ていない作業を始められる。A2 で境界ではないとした範囲の中の話 | A2 で境界を作らないと決めており、tripwire の行と同じ扱い。行を足すだけで、コードは変えない | 既存の tripwire の行と同じ(8a 段目、または本物の境界が要るとき) | この報告の L1、`internal/org/spawn.go:1293`、`internal/org/verbs.go:498` |
| L2: 計画の見出しの前の最初の `- Approved:` の行は、両方のハッシュから外れ、検査は最初の `sha256:` の値しか読まないので、承認のあとにこの 1 行へ文を書き足せる | 計画の中で 1 行だけ、承認のあとに指示を足せる場所が残る | 直すなら `readFeaturePlanHeader` の小さな変更で済む。この cycle で直すかは orchestrator が決める | 次に `readFeaturePlanHeader` か `PlanContentHash` を変えるとき | この報告の L2 |

_(この 2 行は、依頼の指示(この報告のファイル以外は書かない)に従い、`docs/tech-debt/README.md` に足していない。直さずに残すなら、orchestrator が足す。M1 は tech-debt の (d) の行と合わせて直すか、その行に足す。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。M1 は PR の前に直すことを勧めるが、止める理由にはしない)
- Follow-ups:
  - M1: 拒否の文の次の手を理由ごとに 1 つにし、leader の雛形の手順 7 を合わせる。直さないなら tech-debt の (d) の行に足す
  - L1・L2: 直さないなら上の 2 行を `docs/tech-debt/README.md` に足す
  - L3〜L11: 次にこのファイルを触るときにまとめて直せる
  - insight event: 依頼の指示でこの報告のほかには何も書いていないので、`./scripts/insights-append.sh --slug org-plan-approval --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 1 --low 11 --cycle auto --source skill || true` は打っていない
