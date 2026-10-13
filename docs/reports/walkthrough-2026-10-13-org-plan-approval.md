# Walkthrough: org-plan-approval(7 段目の前半)

- Date: 2026-10-13
- Plan: docs/plans/archive/2026-10-12-org-plan-approval.md(PR 作成時に active から移す)
- Branch: feat/org-plan-approval(base 225dff27。PR 作成の時点で 38 commits、71 files、+76554/-271)
- 読む範囲: 差分の大半(約 7 万行)は `docs/evidence/` のテストの出力で、読むのは結果の表だけでよい(test の報告の各節が要約している)。コードは `internal/` の 25 files、+6035/-205 で、そのうちテストが 13 files、+4178/-46。ほかに `/org` skill の 4 面、leader の雛形、仕様の注、quality-gates、tech-debt

新しいファイルは `internal/org/approval.go`(計画の参照、digest、検査)、`internal/org/approve.go`(`ralph org approve` の org 層)、`internal/cli/org_approve.go`(その CLI)と、それぞれのテスト。次の順に読むと、依存の向きに沿って追える。

## 1. 設定と結びつき(`internal/config/config.go`、`internal/org/feature.go`、`internal/org/reserve.go`、f779cc56)

- `[org.approval].mode` は `"per-plan"`(既定)か `"split"`(`config.OrgApprovalConfig`)。ほかの値と空文字は `Load()` のエラーで、`Default()`・`templates/base/ralph.toml`・`scripts/ralph-config.sh` の 3 面を `TestDefaultsLockStep` が揃える
- 方式を読むのは `ralph org start --plan` だけで、`featureApprovalMode` が結びつきの `scope_reserved` の Details に `approval=<mode>` として書く。leader のタスクには `- 承認方式:` の行が入る(`featureLeaderTask`)
- `FeatureBinding.Approval` と `RequiresApproval()`: `approval=` のない古い記録は split 扱い(その leader の雛形に承認の手順がない)、値が読めなければ `approvalModeUnreadable` で承認を求める。検査はこの記録だけを読み、spawn・send・approve を打つ側の設定は読まない

## 2. 計画の参照と検査(`internal/org/approval.go`、`internal/org/protocol/protocol.go`、c08a1a14、9fb3fafa、4b6d0d55、0326574e、3563129d)

- `PlanRef` は `PLAN: <worktree の中のパス> sha256:<12 桁>`。`ParsePlanHeader` はヘッダの塊の `PLAN:` をちょうど 1 行要求する(`protocol.HeaderValues` は空行のあとの本文を読まない)。パスは `resolvePlanFile` が worktree の中の普通のファイルに限る(絶対パス、`..`、外を指す symlink を断る)
- `PlanDigest` は `scripts/plan-visual.sh digest` と同じ値(`TestPlanDigest_MatchesScript`)。digest が読まない場所(`## Progress checklist`、`- Status:`・`- Approved:`・`- Branch:` の行)を見張るため、`PlanContentHash`(飛ばすのは先頭の 2 行と checkbox の印だけ)も記録して照らす
- `checkPlanApproved`: この org の今の命(最後の `disbanded` より後、`currentPlanApprovals`)の `plan_approved` に同じパスと digest があり、ファイルの digest が同じで、先頭の `- Status: Approved` と `- Approved: <YYYY-MM-DD> sha256:<digest>` があり、全文のハッシュが記録のどれかと同じなら通る
- 拒否は `approvalCheckError` で、理由(見つけたこと)と kind を持つ。`approvalNextStep` が kind ごとに次の手を 1 つ選ぶ(記録がない → 待つか BLOCKED、2 行がない → 書いてコミット、変わった → 戻すか承認し直す、`PLAN:` の形 → 足すか直す、それ以外 → BLOCKED)。どの拒否も `refused by the approval check` を持ち、leader が自分で approve を打たないこと、TASK なら `--raw` で送らないことで終わる
- `checkTaskPlanApproved`(TASK の検査): TASK の digest が「記録がない」か「変わった」で断られ、ファイルの今の digest がその path の最新の承認(`latestApprovedDigest`)なら、今の digest の検査で案内を選ぶ(通れば `PLAN:` を今の digest に直す、2 行がまだなら 2 行を書く、digest の外が変わったならその変更を戻す)。今の digest が最新の承認でなければ TASK 自身の拒否を返し、人が置き換えた版へは案内しない(cross-review cycle 1 #2、self-review N6・N7)
- `checkAnyPlanApproved`(spawn の検査): 計画を名指さないので、今の命のどれかの承認が通れば通る

## 3. spawn と send の検査(`internal/org/spawn.go`、`internal/org/verbs.go`、e3d34062、3563129d)

- `spawnFeatureSeatErr`: 結びつきのある org で、leader の座席が動いていなければ leader 以外の spawn を方式を問わず断る(4 段目から持ち越し)。per-plan なら role implementer の spawn に `checkAnyPlanApproved` をかける。置き場所は `spawnCapacityErr` のあとで、dry-run も同じ順(`TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks`)。拒否は上限の拒否と同じく `rejected` と receipt を書く
- `Send`: TYPE が TASK(`--raw` は 1 行目)なら、見張りの関門のあと、dry-run の枝の前に `checkSendApproval`(manifest を読んで `sendApprovalErr`)を呼ぶ(`TestOrgSend_WatchGateBeforeApprovalCheck`)。`sendApprovalErr` は、結びつきが per-plan のときだけ宛先の座席の role を台帳で引き、implementer なら `checkTaskPlanApproved` をかける。拒否は何も記録せず、herdr も呼ばない
- 宛先の手が空くのを `AgentWait` で待ったあと、打つ前に `checkSendApproval` をもう一度呼ぶ(cross-review cycle 1 #1)。待つ間に作業中の座席が同じ worktree の計画を書き換えうるため。2 回目は manifest を読み直す(`TestOrgSend_ApprovalCheck_AfterTheWaitReadsTheManifestAgain`)。2 回目で断られる TASK は herdr の `agent wait` を 1 回呼んだあとになる。見張りの関門は確かめ直さない(その時点の判定)

## 4. `ralph org approve`(`internal/org/approve.go`、`internal/cli/org_approve.go`、5596561a、54feb287、9fb3fafa、4b6d0d55)

- `Approve(ApproveParams{IDs})`: まず `approveTripwire` が、`HERDR_PANE_ID` が動いている座席の pane と同じなら何も読まずに断る(座席がうっかり打つのを止める仕掛けで、境界ではない)。次に受信箱の件ごとに、`approveCandidates`(件があって resolved でない QUESTION、`PLAN:` が 1 行、resolve の note が 500 字に収まる)と `checkApproveCandidate`(結びつきが完全で disband されていない、パスが worktree の中、今のファイルの digest がヘッダと同じ)が確かめる。1 つでも通らなければ何も書かない
- 通れば、manifest のロックの下で `recordApprovals` が件ごとに `plan_approved`(`plan=`・`digest=`・`content=`・`inbox=`・`by=`・`herdr_pane=`・`cwd=`)を書き、`resolveApproved` が件を resolve する。resolve に失敗しても記録は残し、打ち方を出して終了コード 1。ほかの誰かが先に resolve していたら、DECISION をまだ送らず、先に `ralph org inbox show <ID>` で note を読むよう言う
- approve は計画のファイルを書き換えず、leader にも送らない。CLI は `approved <ID> (...)` の行と、leader に送る `ralph org send` の DECISION のコマンド(`ApprovalDecisionCommand`)を出す。`--org-id` は断る(org は件から決まる)

## 5. leader の雛形と文書(36b9d3a3、7e025b45、5e809ac6、b9a7923d)

- `internal/org/prompts/leader.md` の「機能ごとの org」: per-plan の手順(計画をコミット → `/plan` の雛形で図解ページ → `PLAN:` と `VISUAL:` の QUESTION を escalate → DECISION を待つ → 2 行を書いてコミット → implementer を spawn し、TASK に最後に承認された digest の `PLAN:` を書く)と、拒否の次の手に従うこと。`TestRenderRolePrompt_Leader_FeatureOrgPlanApproval` が固定する
- `/org` skill の 4 面(動詞表の `approve`、「機能の計画の承認」節)、`.claude/rules/ralph/agent-messaging.md` の 2 面、仕様 FR-7 の「7 段目の前半で決めたこと」、quality-gates の root と template、README、AGENTS.md、spawn と send の `--help`
- `docs/tech-debt/README.md` に 8 行を足した: tripwire は境界ではない、`ralph org status` に方式も承認も出ない、disband の自己 close の補償の失敗で承認が効かなくなる、headless leader の図解ページは実機で確かめていない、Small leftovers(report の `cwd=` の `|`、`approval=unreadable` の補償のテストなど)、ほかの誰かが resolve した件の approve の文(N8)、テストの helper の重なり(N4)、Small findings(C2-6・C2-7・C2-10)

## パイプラインの往復

上限は既定の 2 のまま(cycle 2 で cross-review が指摘 0)。

| 回 | 見つかったもの | 直した commit |
|---|---|---|
| self-review cycle 1 の 1 回目 | MEDIUM 1(M1: 拒否の次の手が理由と食い違う)、LOW 11 | 54feb287(approve を別のファイルへ移す、L4)、9fb3fafa(M1、L1〜L3、L5〜L11) |
| self-review cycle 1 の 2 回目 | MEDIUM 1(N1: 古い `PLAN:` の TASK が戻す・聞き直す次の手になる)、LOW 4 | 4b6d0d55(N1、N2、N3、N5)、0326574e(承認し直したあとの 2 行の段) |
| self-review cycle 1 の 3 回目 | LOW 3(N6〜N8) | 78b3e873(tech-debt の行) |
| verify cycle 1 | 文書のずれ D1〜D5 | 7e025b45(D1、D3)、5e809ac6(D2) |
| test cycle 1 | 生き残った mutant 4 件(R03〜R05、R11) | 2d238702(テスト 3 本) |
| cross-review cycle 1 | ACTION_REQUIRED 2 件(待ったあとの確かめ直し、最新の承認だけへの案内) | 3563129d(N6・N7 も閉じた) |
| self-review cycle 2 の 1〜3 回目 | LOW C2-1〜C2-10 | edee787f(C2-1、C2-3、C2-4)、ほかは b9a7923d で template の直しと tech-debt の行 |
| verify cycle 2 | 文書のずれ D6〜D12 | b9a7923d |
| test cycle 2 | 生き残った mutant 2 件(Q07、Q08) | 6a7d35b0(テスト 2 本) |
| cross-review cycle 2 | なし | - |

## 実機確認

隔離した herdr 0.7.5、stub の claude と agmsg、PATH から osascript を外した環境で、ビルドしたバイナリを打った。cycle 1 は (a) 承認の前の拒否、(b) approve のあとの通過、(c) 計画の変更と承認のし直し、(d) 座席の pane からの approve、(e) split の org と leader のいない org、AC14(`docs/evidence/org-plan-approval-live-2026-10-13.md`)。cycle 2 は (f) 承認 A・B のあとファイルを A に戻した TASK、(g) 打ち間違えた digest、(h) digest の外の変更、(i) 待つ間の計画の変更(`docs/evidence/org-plan-approval-live-2026-10-13-c2.md`)。本物のモデルの座席は起動していない。
