# Verify report: org-reseat(`ralph org reseat`、#312c の 1 本目)

- Date: 2026-11-13(JST。UTC では 2026-11-12)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(spec `docs/specs/2026-11-12-org-reseat.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 1 回目。テストは走らせていない(`/test` の担当)
- Scope: `git diff 0936aaea...f8579b7c`(22 ファイル)。1 本目の範囲の FR-1〜FR-8(FR-5a を含む)、FR-11、D1〜D6 と D2 の実装中の注、計画の AC-1〜AC-12 と AC-15〜AC-21。AC-13、AC-14、FR-9、FR-10 は 2 本目なので対象外
- Evidence: `docs/evidence/verify-2026-11-12-org-reseat.log`(git の対象外。`run-static-verify.sh` の全出力と、HEAD から作った binary の `ralph org reseat --help`)。ラッパー自身の記録は `docs/evidence/verify-2026-11-12-151852.log`

## Spec compliance

各 AC は、コードを読んで判定し、それを固定するテストの名前を添えた。テストの合否は `/test` が確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 3 席の停止と leader → implementer → reviewer の起動、`scope_reserved` が増えない、承認の検査が通る | 満たす | `Reseat`(`internal/org/reseat.go:987`)、止める順は `PlanReseat` の `stops`(`reseat.go:385-393`)。`TestReseat_StopsAndStartsEverySeatInOrder` が順序、`scope_reserved` の数、`spawnFeatureSeatErr` と `checkTaskPlanApproved` の通過を固定する。実機は leader と reviewer の 2 席で合格(evidence の「AC-1 の判定」) |
| AC-2 leader が新しい workspace の最初の pane | 満たす | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace`。実機でも leader が root の pane に入った |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | 満たす | `reseatSeat`(`reseat.go:691`)、`finish` の `Decision`(`reseat.go:1122-1131`)。`TestReseat_OneSeatOnAnotherDriver`、CLI は `TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader` |
| AC-4 止める前の断り(計画の変更、自分の節か前文、計画のファイル、見張り、worktree の記録) | 満たす | `PlanReseat` の 5〜9(`reseat.go:395-474`)、`reseatWorktreeRecordErr`(`reseat.go:709`)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse` の 13 件と `TestReseat_ARefusalWritesNothing`。見張りは下の I3 |
| AC-5 結びつきの無い org の断りと stop・spawn の打ち方 | 満たす | `reseatNoBindingErr`(`reseat.go:525`)。`TestPlanReseat_RefusesAnOrgWithoutABinding`、`TestOrgReseat_OrgWithoutBindingRefused` |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方 | 満たす | `reseatNoActiveSeatErr`(`reseat.go:589`)。`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`(計画の段)。全座席が止まった状態から `--seat` で立つ実行は `TestReseat_RetryAfterAFailedStart` と `TestPlanReseat_FinishesAReseatThatWasCutOff` が通る |
| AC-7 座席の pane から打った reseat の断り | 満たす | `reseatSelfPaneErr`(`reseat.go:506`)。`TestPlanReseat_RefusedFromASeatPane` |
| AC-8 pane が無い座席の立て直し | 満たす | `TestReseat_SeatsWhosePanesAreGone`(`pane=already closed`)。実機では herdr 0.7.5 が `session.json` から中身の無い pane を戻し、Stop は `pane=closed` で閉じて合格(下の I2) |
| AC-9 `--dry-run` の出力 | 満たす | `printOrgReseatDryRun`(`internal/cli/org_reseat.go:356`)。`TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` |
| AC-10 `stop_failed` で何も立てない | 満たす | `stopAll`(`reseat.go:1045`)。`TestReseat_AStopThatFailsStartsNothing` |
| AC-11 フラグの組み合わせの断り | 満たす | `checkReseatFlags`(`reseat.go:480`)はロックより前(`reseat.go:997`)。`TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、`TestOrgReseat_FlagRefusals` |
| AC-12 実機で AC-1 と AC-8 を 1 回ずつ | 一部 | `docs/evidence/org-reseat-live-2026-11-12.md`。AC-8 は合格。AC-1 は 2 席で合格し、implementer と承認の検査は実機で見ていない(費用のため。計画の S5 の記録どおり)。binary は dd204370 で、後の 93395521 と e5b2cb85 より前(下の I1) |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | 満たす | `noteDetails`(`internal/org/verbs.go:1218`)、`reseatedDetails`(`reseat.go:1171`)。`TestReseat_RecordsItsIDOnEveryStopAndStart`。途中の失敗の `reseated` は `TestReseat_AStopThatFailsStartsNothing` と `TestReseat_RetryAfterAFailedStart` |
| AC-16 skill の 4 つの写しと help が動詞とアカウントの移し方を言う | 満たす | org skill `.claude/skills/org/SKILL.md:369`(表の行)と `:1610`(「座席の立て直し」、移し方は `:1694` から)。director skill `.claude/skills/director/SKILL.md:1393`。help は `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`。4 つの写しは sha256 が一致(org 4d5c49155d47、director ca5f65fd9cef)。ただし skill は P9・P10 より前に書かれている(下の V1) |
| AC-17 `reseat.lock` と止める直前の読み直し | 満たす | `takeReseatLock`(`reseat.go:916`)、`stopAll` の `reseatSeatNow`(`reseat.go:1052-1062`)。`TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop`(テストの hook で順序を決める) |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | 満たす | `reseatRetryCommand`(`reseat.go:1194`)、`--seat` の受け付け(`reseat.go:556`)。`TestReseat_RetryAfterAFailedStart` の 2 件、`TestOrgReseat_AFailedStartPrintsTheRetry` |
| AC-19 `--dry-run` は見張りも server も起動しない | 満たす | `reseatDryRunGates`(`reseat.go:751`)、`wireOrgReseatDryRunGates`(`org_reseat.go:306`)。`TestPlanReseat_DryRunStartsNoWatchOrServer`、CLI の dry run のテストは起動の関数と gate が 0 回呼ばれることを固定する |
| AC-20 切れた reseat の座席と signal | 満たす | `reseatLifeSeat.cutOff`(`reseat.go:247-251`)、`orgReseatHoldSignals`(`org_reseat.go:271`)。`TestPlanReseat_FinishesAReseatThatWasCutOff`、子の process で signal を送る 2 本(`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt`) |
| AC-21 独自の役割の座席の断り | 満たす | `reseatCustomRoleErr`(`reseat.go:637`)。`TestPlanReseat_RefusesACustomRoleSeat` |

仕様の FR と D の対応:

- FR-1・D1: フラグと `--driver`/`--model` の制約は `checkReseatFlags`。仕様にない `--timeout-ms`(既定 60000、座席ごと)もあり、org skill の表の行が書いている
- FR-2・D2・D4: 結びつきの判定、既定の対象、`--seat` の受け付けは上の AC-5、AC-6、AC-18。D2 の実装中の注(切れた reseat、signal、独自の役割)はコードと一致する(`reseat.go:21-36`、`org_reseat.go:249-296`)
- FR-3: AC-7
- FR-4・D3: `PlanReseat` が止める前にすべてを検査する。worktree の記録が結びつきと一致することは、`checkFeatureWorktreeReuse`(記録が `want` と同じ)と `sameFeature`(`internal/org/reserve.go:176`、今の計画の結びつきの Worktree と Branch が台帳のものと同じ)の連鎖で成り立つ。上限と予約は止めたあとの写し `sim` に `spawnPrecheckErr` をかける
- FR-5a・FR-5・FR-6・FR-7・FR-8・D5: 上の AC-17、AC-10、AC-18、AC-3、AC-15、AC-9 と AC-19
- FR-11・D6: help(`org_reseat.go:99-104`)、org skill の移し方、director skill の 3。recipe は変えていない(下の I2)
- 非機能要件: 新しい環境変数を pane に渡さない(`PlanReseat` が作る `SpawnParams` に env の欄は無い)。`reseated` は `stateEvents` に無く(`internal/org/seat.go`)、`RunningOrgs`、`orgsToDisband`、`leaderActivityEventCount` のどれも数えないことをコードで確かめた

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(exit 0) | 1 回で通った。タイムアウトは無かった |
| ├ shellcheck、`sh -n`、hook の awk、`jq -e`、Codex の hook の検査 | pass | 今回の diff に shell は無い |
| ├ `scripts/check-sync.sh` | pass | DRIFTED 0、ROOT_ONLY 0 |
| ├ `scripts/check-skill-sync.sh` | pass | 14 skill が揃っている |
| ├ `scripts/check-pipeline-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照 | pass | |
| ├ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`(2.15.0)、`staticcheck ./...` | pass | `gofmt: ok`、golangci-lint `0 issues.`、vet と staticcheck は出力なし |
| └ `secret-scan-branch.sh` | pass | `0936aaea..f8579b7c` clean |
| `go build -o /tmp/verify-reseat/ralph ./cmd/ralph` と `reseat --help`、`org --help` | pass | help の本文は evidence の log の末尾。`org --help` に `reseat` の行がある |

## Documentation drift

すべて `/sync-docs` に回す。どれもコードの動きは変えない。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| V1 org skill と director skill(各 4 つの写し) | No(LOW) | skill は 51d0129d で書かれ、P9・P10(93395521)と SIGPIPE(e5b2cb85)より前。org skill の表の行(`.claude/skills/org/SKILL.md:369`)は「対象は既定で…active な座席すべて。active な座席がなく `--seat` もなければ断り」と言い、「座席の立て直し」の 1(`:1630`)も同じ。コードは切れた reseat が止めた座席も既定の対象に入れ、active な座席もその座席も無いときだけ断る(`reseat.go:542-555`)。どちらの skill も、独自の役割の座席の断り(AC-21)、切れた reseat を同じ `--seat` で打ち直して終える手順、SIGINT・SIGTERM・SIGHUP・SIGPIPE で reseat が終わらないこと(AC-20)を書いていない。director skill の 7(`.claude/skills/director/SKILL.md:1446`)も、途中で切れた reseat の場合を言わない |
| V2 `AGENTS.md` の `internal/org/` の行(`:91`) | No(LOW) | `reseat.go`(`ralph org reseat`、`reseat.lock`、`reseated`)の説明が無い。計画の Affected areas が sync-docs で足すと書いている |
| V3 `README.md` | No(LOW) | 動詞の行(`:131`)の一覧に `reseat` が無く、`resume` の段落(`:294`)にあたる reseat の段落も無い |
| V4 spec の印 `docs/specs/2026-11-12-org-reseat.md` | No(LOW) | 30 の印がすべて `[ ]`。この report で 1 本目の FR-1〜FR-8、FR-5a、FR-11、非機能要件 4 つ、AC-1〜AC-12 を満たすと判定した(AC-12 は上の「一部」の注つき)。FR-9、FR-10、AC-13、AC-14 は 2 本目まで `[ ]`。Open questions の 2 つは実機の記録が答えている(全座席が Leave しても agmsg の team と履歴は残る。戻った pane には端末もプロセスも無い) |
| V5 `docs/specs/2026-08-01-org-runtime.md` | No(LOW) | 新しい動詞の改訂の項が無い。前例は「2026-11-07 改訂」の (a)(`ralph org resume` を足した) |
| V6 動詞の列挙 | No(LOW) | reseat は `orgLedgerMutating` で台帳を読む(`internal/cli/org_reseat.go:180`)が、`internal/cli/org_legacy_ledger.go:32-40` のコメントと org skill の「前提」の古い台帳の一覧(`.claude/skills/org/SKILL.md:168-172`)に無い。見張りの段落(`:2346` から。start が見張りを起動する、と書く)と director skill の「herdr のサーバーを自分で起動しない」(`:2058`、start・spawn・director start が起動する、と書く)も、同じく起動する reseat を挙げない |
| V7 help のフラグ | No(LOW) | `--seat` の使い方の文「(default: every active seat of the org)」は、Long の本文が言う切れた reseat の座席を落としている。永続フラグ `--config` の使い方(`internal/cli/org.go:66`)は、`[org.approval].mode` と `[reviewer]` を読む動詞に start --plan だけを挙げ、同じく読む reseat(`resolveOrgStartConfig`)を挙げない |
| V8 `docs/tech-debt/README.md` の Trigger | No(LOW) | 次の行の Trigger に当たったが、持ち越しの注が無い(前例はどの行も「YYYY-MM-DD: <branch> hit … did not pay … Carried over」を足している)。`Stop` を変えた(`StopParams.Note`、6a40276e)ことで 361 行目(stop が spawn のロックを取らない。計画の P6 と Risks が名指しする行)、179 行目、183 行目の (a)。`checkCapacityAndStart` を変えた(spawn_started の Details の note)ことで 189 行目。org skill の動詞の表を編集した(`reseat` の行。それ自体も多くの動きを 1 つのセルに詰めている)ことで 180 行目の (a)。`StartFeature` の検査を `startFeatureInput` と `checkLedger` に切り出した(d7389e50)ことで 201 行目の (e)(f) と 391 行目の (b)。`resolvedOrClean` の呼び手(`feature.go`)を編集したことで 396 行目。self-review の L10 と L2〜L6 も、まだ register に無い(self-review の report は orchestrator が決めると書く) |
| `ralph org reseat --help` | Yes(V7 を除く) | Long の本文は FR-1、FR-3、FR-7、FR-8、FR-11、P9、P10 のコードと一致する。アカウントの移し方と、server を止めると全 org の pane と headless の director が消えることを言う |
| `docs/recipes/isolated-herdr-server.md` | Yes(I2) | 計画は「要るなら」。FR-11 は org skill の節で満たす |
| 計画 `docs/plans/active/2026-11-12-org-reseat.md` | Yes(INFO) | AC の印 19 件が `[ ]` のままなのは、PR の前に付ける前例どおり。Affected areas の `internal/org/feature_test.go` は変わっておらず、検査の切り出しの順序のテスト `TestStartFeature_InputRefusalsComeBeforeTheGates` は `reseat_test.go` にある |

## Observational checks

- `go build` した HEAD の binary で `ralph org reseat --help` と `ralph org --help` を見た。表示は上の V7 のとおり
- 実機の記録(`docs/evidence/org-reseat-live-2026-11-12.md`)を読んだ。AC-1(2 席)と AC-8 の判定、開いた問いの答え、確かめていないことの一覧がある
- INFO:
  - I1 実機の binary は dd204370 で、切れた reseat の座席(93395521)と SIGPIPE(e5b2cb85)より前。AC-1 と AC-8 の道は、既定の対象に足した切れた座席(実機には無い)と独自の役割の断り(実機には無い)を除けば同じ。後の変更は実機では見ていない
  - I2 org skill の移し方の 1(`.claude/skills/org/SKILL.md:1700`)は「reseat は pane のない座席を閉じ済みとして止める」と言う。実機では herdr 0.7.5 が `session.json` から中身の無い pane の記録を戻し、Stop は `ctrl_c=failed ... pane=closed` で閉じた。結果は同じだが、`session.json` の復元を書くなら、この文か recipe の「Running again」に 1 文を足せる
  - I3 AC-4 の「見張りが動いていない」: 本当の reseat は start と同じく、見張りが無ければ起動して進む(help と FR-8 が言う)。断るのは、gate が見張りを動かせないとき(古い印、止まりかけ、`--once`、起動の失敗)。テストは断る gate の stub で確かめている。仕様の FR-4 の「start と spawn と同じ」と食い違わない
  - I4 FR-2 の「一度も立っていない座席を名指しすると断る」に対し、コードは `spawned` が無くても今 active な座席(spawn_started か spawn_step のまま)を受け付ける(`reseat.go:556`)。計画の S1 の記録が「今 active な座席」と書き、D2 の既定の対象にも入るので、ずれではない。そのような座席を止めると、361 行目の stop と spawn の競合に当たりうる
  - I5 5 か所のコメントが `docs/plans/active/2026-11-12-org-reseat.md` を指す(self-review の L6)。`/pr` が計画を archive に移すと古いパスになる
  - I6 `internal/org/reseat.go` は 1230 行、`reseat_test.go` は 1637 行、`internal/org/feature.go` は 792 行から 848 行になり、レビューの目安の 800 行を超えた。diff の品質の話なので、記録だけにする

## Coverage gaps

- テストは走らせていない。上の AC の判定は、コードを読み、テストが該当の動きを固定していることを読んで出した。合否は `/test` で確かめる
- AC-1 の implementer と承認の検査、AC-3、AC-10、AC-17、AC-18 は実機で見ていない(テストだけ)
- 止める直前の読み直しと Stop の書き込みのあいだの窓、Stop が座席の spawn のロックを取らないことは、計画の Risks のとおり残る(tech-debt 361 行目)
- 足すとよい確かめ: `/sync-docs` のあと、`grep -c reseat README.md AGENTS.md internal/cli/org_legacy_ledger.go` と、4 つの写しの sha256 の一致をもう一度見る。決定的な検査として足すなら、cobra に登録した `ralph org` の動詞の名前が `README.md` の動詞の行と `org_legacy_ledger.go` のコメントにあることを確かめるテストが、V3 と V6 の型を止める

## Verdict

- Verified: 静的検査はすべて pass。1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、AC-1〜AC-11、AC-15〜AC-21 は、コードとそれを固定するテストの対応で満たす。help は動きと一致し、skill の 4 つの写しは揃っている
- Partially verified: AC-12(実機は AC-1 を 2 席で確かめ、implementer と承認は見ていない。binary は最後の 2 つの直しより前)
- Not verified: テストの実行(`/test`)。実機での AC-3、AC-10、AC-17、AC-18、切れた reseat、signal
- 判定: pass。drift は LOW 8 件(V1〜V8)で、すべて `/sync-docs` に回す
