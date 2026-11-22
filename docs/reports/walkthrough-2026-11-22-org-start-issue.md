# Walkthrough: org-start-issue

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-org-start-issue.md
- Issue: #365
- Base: main ccdf6293

差分は 45 ファイル、+2,990 −209 行。そのうち 637 行は報告 5 本、計画、insight 2 本で、Go は 23 ファイル(+1,969 −138。テスト以外が 6 ファイルで +355 −110、テストが 17 ファイルで +1,614 −28)。残りは leader の雛形、skill の 8 写し(director と org の 4 写しずつ)、仕様 2 本、README、AGENTS.md、tech-debt の台帳。下の順に読むと早い。

## 1. 何が足りなかったか

issue に起票した機能(本文に `ralph-split:` の印の行がある issue)の org を director が立てても、leader はその issue の番号を知らなかった。PR の本文に `Closes #<番号>` が入らないので、PR が merge されても issue は閉じない。待つ側の issue の「頼られる issue」の行は、merge 済みの機能を open と読み続けた。

## 2. `--issue` と結びつき

- `ralph org start --plan <分割計画> --feature <slug> --issue <番号>` を足した。`--plan` と一緒のときだけ受け、値は 1 から始まる 10 桁までの 10 進数(`#` は付けない)。形は 3 段で確かめ、どれも何も作らずに断る。cobra の `Args`(`--plan` なしの `--issue`、`internal/cli/org.go:1185`)、`checkOrgStartPlanInput`(空と空白、`org.go:1288`)、`readStartFeature`(`validIssueNumber`、分割計画を読む前、`internal/org/feature.go:376`。`validIssueNumber` は `internal/org/reserve.go:104`)
- 番号は `FeatureBinding.Issue`(`reserve.go:161`)に入れ、予約の記録の Details の最後に `issue=<番号>` を書く(`scopeReservedDetails`、`reserve.go:451`。`shared=` のあと)。読みは `featureBindingFromTokens`(`reserve.go:559`)。数でない値、2 回、先頭の 0 は「issue なし」と読み、結びつきは完全なまま。古い binary は知らない鍵から先を note と読むので、予約のパスとほかの 7 つの鍵はこれまでどおり読める
- `sameFeature`(`reserve.go:286`)は issue も比べる。disband していない org の `start --plan` のやり直しは、`--issue` が記録と同じとき(どちらもないときを含む)だけ通る。違えば `featureIssueRefusal`(`reserve.go:916`)が、記録の番号と今の値と打ち直し方を名指して断る。digest の断りと同時に起きたときは `featureRestartRefusal`(`reserve.go:895`)が `, and` でつなぐ
- 「同じ機能か」の比べは、fix 52e6cfd3 で `sameFeatureExceptDigestsAndIssue`(`reserve.go:306`)の 1 つにまとめた。`sameFeature`、`featureDigestRefusal`、`featureIssueRefusal` がこれを呼ぶ。self-review の M1(比べの 3 つ目の写し)への直し
- `ralph org reseat` は記録の `Issue` を leader の start に渡す(`internal/org/reseat.go:519`)。座席のない org の断りが案内する `start --plan` の打ち方にも、`--issue` が付く(`reseat.go:903`)

## 3. leader のタスクと表示

- `featureLeaderTask`(`feature.go:608`)は、`Issue` があるときだけ `- 依存する機能:` の直後に `- issue: #<番号>(PR の本文に Closes #<番号> を書く)` の行を足す(`feature.go:628`)。ないときのタスクは前とバイト単位で同じ。issue の行は分割計画の項目ではないので、機能ごとの digest(`featureDigest`)には入れない。そのことは `featureLeaderTask` のコメントに書いた(`feature.go:575`)
- `ralph org status --org-id` は、結びつきに issue があれば `feature:` の行の次に `issue: #<番号>` を出し(`printStatusTable`、`org.go:2323`)、`--json` の `feature` に `issue`(数)を出す(`org.go:2267`)。ない org の出力は変わらない
- `ralph org merge-order --json` は、各ブランチの `binding` に、そのブランチの最後の結びつきの `issue`(数)を出す(`MergeOrderBinding.Issue`、`internal/org/mergeorder.go:340`。`collectMergeTargets` が埋める、`mergeorder.go:582`)。結びつきの値なので、org が disband したあとも、director が入れ替わったあとも、台帳からブランチの issue を引ける。テキストの出力は変えていない
- status と merge-order の数は、どちらも `FeatureBinding.IssueNumber`(`reserve.go:180`)が作る。変換の写しを 2 つ置いていた self-review の M2 への直し(52e6cfd3)

## 4. leader の雛形と skill

- leader の雛形の手順 14(`internal/org/prompts/leader.md:860`)は、タスクに `- issue:` の行があれば、PR の本文の普通の文(コードブロック、HTML のコメント、引用の外)に `Closes #<番号>` を 1 行書くと言う。読む行は、タスクの初めに並ぶ `- ` の行(最初の空行まで)の 1 行に限り、前文の囲み、機能の本文、立て直しの経過の囲みの中の同じ形の行は読まない(self-review の L2 の直し)。行がないときは今までどおり `/pr` skill の決まりで決め、新しく禁じることはない。どちらのときも、雛形の `__ISSUE__` を本文に残さない
- director skill(`.claude/skills/director/SKILL.md`)は、着手を決めた issue の機能の org を `--issue <その issue の番号>` 付きで立てると言う(「機能ごとの出力先」の 3 の 657 行目、「merge を待つ issue」の 3 の 2468 行目)。起票した機能の PR の merge を人に報告するとき(「merge と auto_merge」の 2104 行目、「GitHub で merge した PR の後始末」の 2270 行目)と引き継ぎ(2355 行目)は、`ralph org merge-order --all --json` でそのブランチの `binding.issue` を読み、あれば `gh issue view` で閉じたかを確かめ、`OPEN` なら閉じずに人に伝える。issue を閉じるのは人が頼んだときだけ、の決まりは変えていない。gh の失敗、merge の直後の遅れ、引き継ぎで確かめる範囲(self-review の L3)も同じ箇所に書いた
- org skill(`.claude/skills/org/SKILL.md`)は、コマンド表の `start`・`status`・`merge-order` の行、`--issue` の説明(999 行目)、start が断るもの(1046 行目)、reseat が記録の番号で leader を立てること(1835 行目)、「merge を待つ issue」(2428 行目)に足した。skill の 4 写し(`.claude/skills`、`.agents/skills`、`templates/base` の 2 つ)は、director も org も `cmp` で同じ
- README(280 行目)、AGENTS.md(92 行目)、仕様 2 本(`docs/specs/2026-10-07-org-multi-org-director.md` の FR-4 の 98 行目、`docs/specs/2026-11-12-org-reseat.md` の FR-6 の 90 行目)にも足した。計画の Scope は仕様の FR-2 と書いたが、`start --plan` を扱うのは FR-4 なので FR-4 に置いた(計画の Progress に記録)

## 5. 先に読むテスト

- `TestFeatureBinding_IssueRead`(`internal/org/reserve_issue_test.go:160`。壊れた値 19 通り)、`TestScopeReservedDetails_OlderBinaryReadsSevenTokens`(`:123`)、`TestScopeReservedDetails_NoIssueIsTheRecordOfBefore`(`:76`)
- `TestFeatureLeaderTask_IssueLine`(`internal/org/feature_issue_test.go:42`。issue なしのタスクの sha256 を、この変更の前に計った値で固定)
- `TestStartFeature_RecordsTheIssue`(`feature_issue_test.go:119`)、`TestStartFeature_IssueRestart`(`:173`)、`TestStartFeature_IssueFormRefused`(`:238`)
- `TestSameFeature_Issue`(`reserve_issue_test.go:206`)、`TestReservationDecision_IssueOwnOrg`(`:311`)
- `TestReseat_IssueLineInTheLeadersTask`(`internal/org/reseat_issue_test.go:38`)、`TestMergeOrder_BindingIssue`(`internal/org/mergeorder_issue_test.go:33`)
- `TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue`(`internal/org/prompts_leader_issue_test.go:30`。行がないときに閉じるキーワードを禁じる文がないことも見る)
- CLI(fake の herdr): `TestOrgStartIssue_RefusedWithoutPlanOrBlank`(`internal/cli/org_start_issue_test.go:29`)、`TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask`(`:150`)、`TestOrgStatus_IssueLineAndJSON`(`:295`)、`TestOrgMergeOrder_JSONBindingHasTheIssue`(`:361`)
- `internal/org/skill_issue_test.go`(`:57`、`:112`): skill の 4 写しの文と並び

## 6. 確かめたこと

- self-review(`70fd0fc2`、diff は `d486f4b5` まで): CRITICAL 0、HIGH 0、MEDIUM 3、LOW 6、Merge: yes。M1、M2、L2 の読み位置、L3、L6 は 52e6cfd3 で直した。M3(台帳の繰り越し)は sync-docs が入れた
- verify(`8b6bb6a3`、HEAD 52e6cfd3): pass。AC1〜AC10 は実装と固定するテストで met。`./scripts/run-static-verify.sh` は rc 0(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、branch secret scan)。承認の digest は `cbeb26942ff9` のまま。古い binary の読みは base のコードで確かめた
- test(`701421da`、HEAD 8b6bb6a3): pass。`./scripts/run-test.sh` は rc 0(1,794 s、shell 46 本と Go 11 パッケージ、`(cached)` 0)。変異 46 個はどれも落ちた。`--issue` なしのタスクと Details は、この変更の前(81530fb0)で計った値とバイト単位で同じ。足した本番の Go の文 121 行はすべて通った
- sync-docs(`a13adfa2`): 台帳の Trigger 27 行に引き継ぎの注、新しい行 3 つ(555〜557)。482 の (a) は 52e6cfd3 で払って取り消し線で閉じた。`check-sync.sh` と `check-skill-sync.sh` は通り、絞った Go のテスト(org 110、cli 79)は SKIP 0 で通った
- cross-review(`78423d49`): codex(read-only、`--ignore-rules`、HEAD a13adfa2)の指摘は 0 件。cycle 1/2。codex の sandbox では Go のテストが走らなかった(テストは /test が回した)

## 7. 残したもの

1. 計画の Risks の 4 つ目(古い binary の補償が書き戻して `issue=` が落ちても、director の merge のあとの確かめで拾える)は成り立たない。落ちた記録は `binding.issue` も落ちるので、確かめるものがない。reseat の leader も issue の行を持たない。版が混ざる期間に限る。計画の本文は digest が変わるので直さず、仕様 FR-4 の項と計画の Progress に書いた(self-review の L1)
2. gate と regate の `sameFeature` の比べ(`internal/org/gate.go:773`、`regate.go:1050`)で、issue だけが変わる場合を固定するテストがない。変異 M1(比べから issue を外す)は、gate と regate のテスト 4 つだけでは残る(start と予約の判定のテストでは落ちる)。起きるのは手で書いた記録か古い binary の書き戻しのときで、gate は記録せず「it is now bound to …」と断る。止まる側に倒れる。台帳 556 の (1)
3. 本物の GitHub で `Closes #<番号>` が issue を閉じること、director の `gh issue view` の確かめを本物の gh で回すこと、本物の herdr での `start --plan --issue`、古い binary と混ぜた台帳は未確認。台帳 556 の (2)(3)
4. 手順 14 は、`- issue:` の行と計画の `Related issue:` が別の番号のとき、どちらを閉じるかを言わない。台帳 555
5. fix 52e6cfd3 は self-review のあとに入った。self-review は回し直さず、verify、test、cross-review が直したあとの木を見た
6. sync-docs が `internal/cli/org_start_issue_test.go` のコメント 1 行を、/test のあとに直した(引く計画の path を slug の形に。`/pr` の archive が書き換えないので宙に浮くため)。絞ったテストと gofmt、go vet は回したが、`./scripts/run-test.sh` の全体は回し直していない
