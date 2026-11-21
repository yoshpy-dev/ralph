# Verify report: org-start-issue

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-start-issue.md`(issue #365)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff ccdf6293...HEAD`(origin/main ccdf6293 から HEAD 52e6cfd3 まで、39 ファイル)。仕様への適合(AC1〜AC10)、静的解析、文書のずれを見た。テストは走らせていない(`/test` の担当)。diff の質は self-review の担当で、ここでは self-review の直し(52e6cfd3)が言うとおりに入ったかだけを見た
- Evidence: `docs/evidence/verify-2026-11-22-org-start-issue.log`(gitignore の対象)。`run-static-verify.sh` の出力、HEAD の binary の probe、計画の digest、skill の 4 写しの `cmp`、ファイルの行数、base の読み手の抜き書き。wrapper 自身の log は `docs/evidence/verify-2026-11-21-234907.log`

## Spec compliance

AC の「met」は、実装を読んで AC が言う挙動になっていること、その挙動を固定するテストがあることを確かめたという意味。テストの実行と結果は `/test` が確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `--issue 12` で立てた org の leader のタスクに `- issue: #12(PR の本文に Closes #12 を書く)`、Details の最後に `issue=12` | met | `featureLeaderTask`(`internal/org/feature.go`)は `issue != ""` のときだけ、`- 依存する機能:` の直後、`- 承認方式:` の前にこの行を足す。`scopeReservedDetails`(`internal/org/reserve.go`)は `shared=` のあと、note の前に `issue=` を書く。`startFeatureLeaderParams` が binding とタスクの両方に `p.Issue` を渡す。テスト: `TestStartFeature_RecordsTheIssue`、`TestScopeReservedDetails_IssueRoundTrip`、`TestScopeReservedDetails_IssueTokenLast`、`TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask`(CLI と本物の `ralph-worktree.sh`) |
| AC2: `--issue` なしのタスクと Details はこの変更の前とバイト単位で同じ | met | 実装を読んで確かめた。base(ccdf6293)の `featureLeaderTask` は `- 依存する機能:` から始まる 1 回の `append` で、HEAD はそれを `- 依存する機能:` だけの `append`、issue の行(空なら足さない)、残りの `append` に分けた。`issue == ""` のときの行の並びは同じ。`scopeReservedDetails` に足したのは `f.Issue != ""` で囲んだ 1 か所だけ。テスト: `TestFeatureLeaderTask_IssueLine`(81530fb0 で出した sha256 を固定)、`TestScopeReservedDetails_NoIssueIsTheRecordOfBefore`、`TestStartFeature_NoIssueIsTheStartOfBefore`、`TestReseat_NoIssueLeavesTheLeadersTaskWithoutIt` |
| AC3: `--plan` なし、空、`0`、先頭の 0、`#12`、数でない値、11 桁以上を何も作らずに断る | met | 3 段で断る。(1) cobra の `Args` は、`--plan` の形でないのに `--issue` があれば、引数の数を見る前に `errOrgStartIssueNeedsPlan` を返す。(2) `checkOrgStartPlanInput` は空と空白だけの値を断る。(3) `readStartFeature` は分割計画を読む前に `validIssueNumber` で形を見る。`StartFeature` は watch と herdr server の gate より前に `startFeatureInput` を呼ぶので、副作用はない。HEAD の binary で probe した: `--issue 12 task` と `--plan … --issue ""` は rc=1 で計画どおりの文を出し、state dir は作られなかった(evidence の log)。テスト: `TestOrgStartIssue_RefusedWithoutPlanOrBlank`、`TestOrgStartIssue_MalformedNumberRefusedWithNothingMade`、`TestStartFeature_IssueFormRefused`、`TestValidateFeatureBinding_Issue` |
| AC4: `ralph org reseat` が立て直した leader のタスクに記録の issue の行 | met | `PlanReseat` の `leaderStart` に `Issue: binding.Issue`。座席のない org の断りが出す `start --plan` の打ち方にも、`binding.Issue` があるときだけ `--issue` が付く(`reseatNoActiveSeatErr`)。テスト: `TestReseat_IssueLineInTheLeadersTask`(leader のほかの座席だけの reseat が通ることも見る)、`TestPlanReseat_StartHintNamesTheIssue` |
| AC5: やり直しの start は `--issue` が記録と同じ(どちらもなしを含む)なら通り、違えば両方の値を名指して断る | met | `sameFeature` が `a.Issue != b.Issue` を比べる。断りの句は `featureIssueRefusal` が作り、digest の句があれば `featureRestartRefusal` が `, and` でつなぐ。テスト: `TestStartFeature_IssueRestart`、`TestReservationDecision_IssueOwnOrg`、`TestSameFeature_Issue`、CLI の `TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask` |
| AC6: 壊れた `issue=`(数でない、2 回)は issue なし、結びつきは完全なまま。古い binary の記録(`issue=` なし)も issue なし | met | `featureBindingFromTokens` は `issue` を鍵の表に足し、最後に `validIssueNumber` でない値を `""` にする。`Complete()` は issue を見ない。古い binary の読みは base のコードで確かめた: ccdf6293 の `featureBindingFromTokens` は知らない鍵で `break` するので、新しい記録の `issue=…` から先を note と読み、7 つの鍵と予約のパスは前のとおり読む(evidence の log に抜き書き)。テスト: `TestFeatureBinding_IssueRead`(19 通り)、`TestScopeReservedDetails_OlderBinaryReadsSevenTokens` |
| AC7: `ralph org status --org-id` の `issue: #<番号>` の行と `--json` の `issue`。issue のない org は変わらない | met | `printStatusTable` は `feature:` の行の次に出す。`orgFeatureJSON.Issue` は `int64` に `omitempty` で、0 なら出ない。値は `FeatureBinding.IssueNumber()`(merge-order と同じ関数)。テスト: `TestOrgStatus_IssueLineAndJSON`(壊れた記録、10 桁、2 回、先頭の 0 を含む)、CLI の通しのテスト(issue のない org のキーが 6 つのままであること) |
| AC8: leader の手順 14 が `- issue:` の行があれば普通の文に `Closes #<番号>` を書くと言い、行がないときの決まりを変えない | met | `internal/org/prompts/leader.md` の手順 14。行がないときは「今までどおり `/pr` skill の決まり(計画の `Related issue:` など)で決める」と書き、どちらのときも `__ISSUE__` を残さない一言を添える(計画の Scope のとおり)。self-review L2 の直しで、読む行をタスクの最初の空行までの `- ` の行に限った。その範囲は `featureLeaderTask` が組むタスクの形と一致する(前文の囲みと本文は空行のあと)。テスト: `TestRenderRolePrompt_Leader_FeatureOrgPRClosesTheTasksIssue`(`Closes` を含む文に禁止の語がないこと、雛形に `- issue: #` で始まる行がないことも見る) |
| AC10: `merge-order --all --json` の `binding.issue`。disband のあとも出る、org_id の使い回しではブランチごとの最後の結びつきの issue、`--issue` なしのブランチにはキーがない | met | `MergeOrderBinding.Issue`(`int64`、`omitempty`)を `collectMergeTargets` が `f.IssueNumber()` で埋める。テキストの出力は変わらない。テスト: `TestMergeOrder_BindingIssue`(org-a の one と two、feat/auth の 2 つの結びつき)、`TestOrgMergeOrder_JSONBindingHasTheIssue`(行と note に issue が出ないことも見る) |
| AC9: director skill と org skill。`check-sync.sh` と `check-skill-sync.sh` が通る | met | director skill の 5 か所(機能ごとの出力先の 3、merge と auto_merge の箇条、後始末の 2、merge を待つ issue の引き継ぎと 3)、org skill の 8 か所。どちらも 4 写しが `cmp` で同じ。`check-sync.sh`(DRIFTED 0)と `check-skill-sync.sh`(14 skill)は `run-static-verify.sh` の中で通った。テスト: `TestDirectorSkill_StartsAFiledIssueWithItsNumberAndChecksTheIssueClosed`、`TestOrgSkill_DescribesStartIssueStatusLineBindingAndReseat` |

依頼にあった個別の確かめ:

- gate と regate の `sameFeature`: 呼ぶ所は `gate.go:773` と `regate.go:1050` の 2 つで、どちらも同じ台帳を 2 回読んだ結びつきを比べる。両方とも `reservationFromEvent` → `featureBindingFromTokens` を通るので、同じ binary が書いた記録なら issue は同じに読め、比べの結果は変わらない。違いが出るのは、間に記録が書き直されたときだけ(下の V3)
- `sameFeature` の書き換え: base の `!a.Complete() || !b.Complete() || 5 項目` は、HEAD の `!b.Complete() || !sameFeatureExceptDigestsAndIssue(a, b) || a.Issue != b.Issue` と同じ条件になる(helper が `held` 側の `Complete()` を見る)。`featureDigestRefusal` の guard も同じ helper になり、base の `want == nil` の検査は helper の nil の検査に入った
- `--issue` なしの start: `Args` と `checkOrgStartPlanInput` の追加は `Changed("issue")` のときだけ動く。変わるのは help の文と `Use` の行だけ
- leader の手順 14 は、行がないときに新しく禁じる文を足していない(AC8 の行)
- #364 の計画(`.claude/worktrees/issue-depends-wait/docs/plans/active/2026-11-22-issue-depends-wait.md`)が使う `binding.issue`: 鍵の名前と場所(`entries[].binding.issue`)は一致する。値は JSON の数(`#` なし、文字列ではない)で、issue がなければ鍵ごと出ない(V4)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(既定の changed scope、HEAD 52e6cfd3) | pass(`rc=0`、`All verifiers passed.`) | `Language packs selected: golang`。local verifier(shellcheck、`sh -n`、jq、check-sync は DRIFTED 0、check-pipeline-sync、check-skill-sync は 14 skill、check-template-purity、tech-debt README plan references)はすべて OK。`gofmt: ok`、`go vet ./...` は出力なし、golangci-lint は `0 issues.`、staticcheck は出力なし(`Skipping` の行はない。go 1.27.3) |
| branch secret scan(上の中) | clean | `scanned ccdf6293..52e6cfd3 against origin/main` |
| `./scripts/run-static-verify.sh`(この report、計画の Progress、insight の行を working tree に入れて 2 回目) | pass(`rc=0`) | 同じ内容。wrapper の log は `docs/evidence/verify-2026-11-21-235833.log` |
| `./scripts/plan-visual.sh digest <plan>` | `cbeb26942ff9` | 計画の `- Approved:` の digest と同じ。Progress の節は digest の外 |
| `cmp`(director skill と org skill、それぞれ 4 写し) | 同じ | |
| HEAD の binary の probe(`go build ./cmd/ralph`、scratchpad) | 計画どおり | `--issue` の断り 3 つ(rc=1、state dir は作られない)と、`start`・`status`・`merge-order` の `--help` の issue の文。evidence の log |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org start --help` | yes | `Use` の行、立て直しの条件の文(self-review L6 の直しで `--issue` が入った)、`--issue` の段落、flag の説明。HEAD の binary で読んだ |
| `ralph org status --help`、`ralph org merge-order --help` | yes | issue の行と鍵、`binding` の issue |
| `ralph org reseat --help` | yes(変更なし) | 結びつきの項目を並べていないので、issue を足す所がない |
| director skill(4 写し) | yes | 計画の Scope の 5 か所。`merge-order` の JSON の名前(`entries`、`branch`、`class`、`merged`、`binding`)は `mergeorder.go` の構造体のタグと合う |
| org skill(4 写し) | yes | コマンド表の `start`・`status`・`merge-order` の行、`### ralph org start --plan`、`### start が拒否するもの`、`### 座席の立て直し` の 1 と 4、`### merge を待つ issue` |
| README、AGENTS.md | yes | README は `--issue` の段落、AGENTS.md は repo map の `feature.go` の括弧 |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` | yes(置き場所は計画と違う) | 項目は FR-4 の 2026-11-22 の項(98 行)。計画の Scope は FR-2 と書く。Progress の S3 に理由が記録済み(V2) |
| 仕様 `docs/specs/2026-11-12-org-reseat.md` | yes | FR-6 の 2026-11-22 の追記(計画どおり) |
| `docs/quality/quality-gates.md` の「Split plan approval」の行 | yes(変更なし) | start の断りを列挙しない一般の文(「spawn の検査を副作用の前に走らせる」)で、issue の検査もその前にある |
| `docs/tech-debt/README.md` | no(`/sync-docs` に回す) | V1。self-review M3 の続き |
| 計画の AC のチェックボックス | no(INFO) | AC1〜AC10 は `[ ]` のまま。付けても digest は変わらない(V6) |

## Findings

| ID | Severity | 内容 | 行き先 |
| --- | --- | --- | --- |
| V1 | MEDIUM(記録) | 台帳 `docs/tech-debt/README.md` で、この PR が Trigger を満たす行は self-review M3 の 7 行(225、480、481、482、547、548、549)のほかにもある。HEAD 52e6cfd3 の行番号と最初の語で書く: 182「Two documentation findings of org-stop-all」(a)(`/org` skill のコマンド表の `start`・`status`・`merge-order` の行)、190「What a leader is told about its reservation」(a)(leader の prompt)、194(org-limits-reserve の test gaps、`ralph org status` の `Long`)、197「A `ralph` of another version on the pane's `PATH`」(`newOrgStartCmd` の `RunE`)、202 の (g)(leader.md の機能ごとの org の節。閉じた項目にも枝ごとの注がある)、203「Test gaps left by org-feature-worktree」(e)(f)(`runOrgStartPlan`、`StartFeature`)、252(`mergeorder.go`、`org_merge_order.go`)、370「`internal/org/verbs.go` is N lines」(leader の雛形のテスト。新しい `prompts_leader_issue_test.go` を含む)、382 RC4(leader.md の機能ごとの org の手順)、389「After step 14 of the leader template」(手順 14)、395「Four test files went past the 800-line guide」(`director_skill_sync_test.go`)、396 の (b)(`StartFeature` の検査に issue の形の検査が入った)、401(`resolvedOrClean` の呼び手 `feature.go`)、424 の (10)(`merge-order --json` の出力)、432 の (6)(`ralph org status` の JSON)、519 の (d)(k)(`PlanReseat`、`reseatNoActiveSeatErr`)、520(`reseat.go` 1938 → 1948 行)、521(`/org` skill の reseat の小節)。482 は (a)(e) に加えて (b)(c) も満たす(`validateFeatureBinding` と `FeatureBinding.String()` を変えた)。(a) は 52e6cfd3 の `sameFeatureExceptDigestsAndIssue` で払った(`sameFeature`、`featureDigestRefusal`、`featureIssueRefusal` が呼ぶ)ので、繰り越しではなく払ったと書く。self-review の繰り越しの文の案にある `reserve.go` の行数「822 -> 936」は d486f4b5 の値で、HEAD では 964 行 | `/sync-docs`(依頼のとおり、M3 と合わせて `/pr` の前に入れる)。注に計画を引くときは `docs/plans/active/…` の path で書く(archive の書き換えは `/pr` がする) |
| V2 | INFO | 計画の Scope は仕様の FR-2 に項目を足すと書くが、実装は FR-4 に置いた。この仕様で `start --plan` を扱うのは FR-4 で、#363 の同じ日の項目も FR-4 にある。Progress の S3 に記録済み。計画の本文を直すと承認の digest が変わるので、ずれは Progress の記録のままでよい | なし |
| V3 | INFO | 版が混ざる期間に限る、gate と regate の挙動の差。issue を知らない古い binary が自己終了の補償で結びつきを書き戻すと、`issue=` が落ちる(計画の Risks の 4 つ目)。その書き戻しが gate や regate の実行中に起きると、`sameFeature` が issue の違いを見て、記録せずに断る(「it is now bound to …」)。gate を打ち直せば、新しい結びつきで通る。止まる側に倒れる差で、承認方式の `approval=` を古い binary が落とすときと同じ形。同じ binary が書いた記録では差は出ない | なし(self-review L1 と同じく、PR の Known gaps に入れるなら一緒に) |
| V4 | INFO | #364 との形の合わせ。`binding.issue` は JSON の数で、issue がなければ鍵がない。#364 の (b) が待つ側の本文の「頼られる issue: #<番号>」と比べるときは、数として比べる(jq なら `.binding.issue == 12`、文字列の `"12"` とは等しくならない)。#364 の計画は型を書いていない | #364 の実装で、比べ方を数にする |
| V5 | INFO | タスクに `- issue:` の行があり、機能の実装計画の `Related issue:` が別の番号のとき、手順 14 は `Related issue:` の番号をどう扱うかを言わない。self-review L2 の後半で、self-review の Tech debt identified の 2 行目(読み位置と `Related issue:`)が同じ点を挙げている | self-review の提案のとおり台帳へ(`/sync-docs`) |
| V6 | INFO | 計画の AC1〜AC10 のチェックボックスは `[ ]` のまま。`plan-visual.sh digest` は `- [x]` を `- [ ]` と読むので、付けても digest は変わらない | `/pr` の前に付けるかどうかは orchestrator が決める |
| V7 | INFO | `--feature a --issue 12`(`--plan` なし)の断りは、`--issue` の文ではなく「`--plan` and `--feature` go together」の文になる。org skill の「`--plan` のない `--issue`」は断られるとだけ言い、文は名指さないので、ずれではない | なし |

## Observational checks

- HEAD の binary の probe(evidence の log の「# probes with the HEAD binary」から): `ralph org start --issue 12 task` は `org: start: --issue needs --plan and --feature: …` で rc=1。`--plan demo.md --feature a --issue ""` は `org: start: --issue must not be blank: …` で rc=1。`--feature a --issue 12` は `--plan and --feature go together` で rc=1。3 回とも state dir は作られなかった。どれも state dir を解決する前に断る道で、herdr は呼んでいない
- 古い binary の読み: base の `featureBindingFromTokens` の `if !ok || !known { break }` を log に写した。新しい記録の `issue=12` はこの `break` に当たり、そこから先は note になる。予約のパスは `reservedPathsFromDetails` が最初の空白で切るので変わらない
- ファイルの行数(base → HEAD): `reserve.go` 822 → 964、`feature.go` 899 → 925、`internal/cli/org.go` 2474 → 2530、`reseat.go` 1938 → 1948、`mergeorder.go` 1512 → 1519、`leader.md` 1668 → 1685

## Coverage gaps

- テストの実行(`/test` の担当)。上の表のテストの名前は、AC ごとに固定するテストがあることを読んで確かめたもので、通ることは確かめていない
- 本物の GitHub で `Closes #<番号>` が issue を閉じること、director の merge のあとの確かめ(`gh issue view`)、merge の直後の遅れは、誰も確かめていない(計画の Assumptions の 2 つ目)
- 古い binary(#363 の版)と新しい binary を同じ台帳で動かす通しの確かめはしていない。読みは base のコードとテストの helper(`oldBindingTokens`)で確かめた
- `/sync-docs` が足す台帳の注(V1)は、この report の時点では入っていない
- 足すとよい最小の確かめ: 台帳の Trigger の照合は毎回手で行っていて、self-review も 7 行しか拾えなかった。diff が触れた関数名とファイル名を台帳の 4 つ目のセルと突き合わせ、候補の行を出す script(たとえば `scripts/tech-debt-trigger-candidates.sh <base>`)があれば、`/self-review` と `/sync-docs` の見落としが減る。今回は提案だけで、作っていない

## Verdict

- Verdict: pass
- Verified: AC1〜AC10 の実装と、AC ごとに固定するテストがあること。`--issue` なしのタスクと Details が base と同じ並びになること(コードの読み)。古い binary の読み(base のコード)。gate と regate の `sameFeature` の比べが同じ binary の記録では変わらないこと。`sameFeature` と `featureDigestRefusal` の書き換えが前と同じ条件であること。leader の手順 14 が行のないときの決まりを変えていないこと。director skill の確かめが読む JSON の名前と、#364 の計画が使う `binding.issue` の形。skill の 4 写しが同じこと。`ralph org start`・`status`・`merge-order` の help。`./scripts/run-static-verify.sh` が rc=0(gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、branch secret scan)。承認の digest が変わっていないこと
- Partially verified: 文書のずれのうち台帳(V1)。self-review M3 のとおり `/sync-docs` で入れる
- Not verified: テストの実行(`/test`)、本物の GitHub で issue が閉じること、版の混ざった台帳での通しの動き
- Findings: MEDIUM 1 件(V1、台帳の記録、`/sync-docs` に回す)、INFO 6 件(V2〜V7)。コードと仕様の食い違いはない
- Insight event: `./scripts/insights-append.sh --slug org-start-issue --flow standard --phase verify --verdict pass --cycle auto --medium 1 --source skill` で 1 行(cycle 1、MEDIUM 1。INFO は数に入れない)を足し、この report と同じコミットに入れた
