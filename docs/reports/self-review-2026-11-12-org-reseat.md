# Self-review report: org-reseat

- Date: 2026-11-12
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(issue #312 の受け入れ条件 2、`ralph org reseat`)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff d2e5c6e9...HEAD`(HEAD `0befaf57`、10 コミット、20 ファイル)。`internal/org/reseat.go`(全部)、`internal/org/feature.go`・`seat.go`・`spawn.go`・`verbs.go` の diff 全部と、そこから呼ぶ `Roster`、`Stop`、`spawnPrecheckErr`、`ensure_worktree`、`ProbeWatch`、`fileLockHeld`、`acquireHeldFileLock`、`orgsToDisband`、`RunningOrgs`、`leaderActivityEventCount`。`internal/cli/org_reseat.go`(全部)と、そこが呼ぶ `ensureOrgHerdrServer`、`wireOrgWatchGate`、`resolveOrgLedger`。テストの diff(`reseat_test.go`、`org_reseat_test.go`)は、フックの後始末と `t.Parallel` の有無だけ見た。skill の写し 8 本、evidence、spec、plan は diff 品質の対象外として読んでいない(`$HOME` の写り込みと `fmt.Print` の残りだけ grep した)。テストの実行、静的解析、仕様適合、文書の drift は見ていない
- ID の付け方: `M1`(MEDIUM)、`L1` から `L6`(LOW)

## Evidence reviewed

lead に頼まれた 4 点を、コードを読んで追った(実行はしていない)。

1. `StartFeature` の抜き出し(`feature.go`)。`d2e5c6e9` の `StartFeature` と並べた。旧版の順は、`readStartFeature`、`checkSpawnInput`、`ValidateReviewer`、watch gate、herdr gate、manifest の読み、`spawnPrecheckErr`、`regateSpawnErr`、`checkFeatureWorktreeReuse`、`Ensure`。新版は `startFeatureInput`(最初の 3 つ)、2 つの gate、manifest の読み、`checkLedger`(あとの 3 つ)、`Ensure` で、順も拒否の文も同じ。`res.Split` と `res.OrgID` の入り方も同じ(`s.orgID` は `readStartFeature` が成功したときだけ入る)。`want` が gate より前に作られるが、`featureCanonicalRef` は引数だけから決まる。`o.worktrees()` が `regateSpawnErr` の前に呼ばれるようになったが、どちらも副作用が無い。ファイル先頭の注釈の番号(1、2、3)と `startFeatureInput`・`checkLedger` の対応も合っている。
2. `Note` の 3 つの欄(`spawn.go`、`verbs.go`)。`noteDetails` は空の `note` で `details` をそのまま返す。`spawnedEventDetails` の `parts` は `permission_mode=` で必ず空でないので、空の `Note` の `spawned` は前と同じ。`spawn_started` の Details は前は無く、`noteDetails("", "")` は `""`。`dryRunSpawn` の `base` は Details を持たないので、`step.Details = noteDetails("", p.Note)` は何も潰さない。`Stop` は `details` が組み上がったあと、event を書く直前にだけ足す。`Details` を読む側は `permissionModeFromDetails`(欄ごとに見る)と `reason=watchdog_` の `Contains` だけで、末尾の欄が増えても壊れない。`model_observed=` を読むコードは本番側に無い。
3. `seat.go` の `EventReseated` の注釈の 3 つの主張。`Roster` は `SeatID == ""` の event を `disbanded` 以外読み飛ばす(`manifest.go:192`)。`RunningOrgs` は active な座席、開いた workspace、予約のどれかで決まり、`orgsToDisband` は `lastRecord` に入れる event を名前で挙げていて、どちらにも `reseated` は入らない。`leaderActivityEventCount` の `switch` にも無い。注釈の通り。
4. dry run が何も書かず何も起こさないか。`Reseat` は `DryRun` なら `PlanReseat` だけを呼び、lock を取らない。`PlanReseat` は manifest の `Read`、`worktrees.Lookup`(`state-path` を呼んで JSON を読むだけ)、`regateSpawnErr`、`ProbeWatch`(`os.Open` と `flock(LOCK_SH|LOCK_NB)`、ファイルが無ければ作らない)だけを呼ぶ。CLI 側の `resolveOrgLedger`、`resolveOrgStartConfig`、`newOrgRuntimeFromConfig`、`wireOrgInbox` は store の組み立てで、書かない。dry run が組む `rt.HerdrServerGate` は、`PlanReseat` の dry run の道から呼ばれない(`o.HerdrServerGate != nil` の判定にだけ使う)。
5. 止めたあとで起動できない道。`stopAll` の終わり方(record が変わった、manifest が読めない、stop が失敗)は、どれも `startAll` に進まず、`finish` が `reseated` と `Retry` を出す。`startAll` の失敗も同じ。`stop_failed` の座席は active のままなので(`Stop` の注釈、`isActiveEvent`)、打ち直しの `--seat` で再び止められる。`StartFeature` の `refuse` は `Outcome: Rejected` と `Err` を持つので、`startAll` の `cause` に拒否の文が残る。残る穴は M1 に書いた。
6. 起動してはいけない座席。`--seat` を指定しない reseat の対象は active な座席だけ(`reseatTargets`)で、leader が止めた reviewer を起こさない。leader が止まっているとき非 leader だけの reseat は、`sim` で leader が active でないので `spawnFeatureSeatErr` が止めの前に断る。`reseatSelfPaneErr` は自分の pane を閉じる道を止める。残る穴は L1 に書いた。
7. shell に渡る文字列。`Retry` と `Decision` の座席 id は `ValidateIdentifier` を通った値で、`ShellWord` と `shellQuote` を通す。`ReseatDecisionCommand` は `ApprovalDecisionCommand` と同じ作りで、`--text` を単引用符で包む。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | Exception handling | 最初の stop から `reseated` を書くまでの間に、SIGINT・SIGTERM・SIGHUP(端末を閉じる)を受けると、止めた座席が止まったまま残り、直し方を案内する出力も record も出ない。同じ `reseat --org-id <id>` を打ち直しても、止まった座席は起こされない。`--seat` なしの対象は active な座席だけで(D2)、起動の順が leader 先頭なので、leader を起こしたあとに切れた場合は leader だけがまた止められて起こされ、ほかの止まった座席には何も言わない。stop の途中で切れた場合も、残った active な座席と leader だけが対象になる。AC-6 の拒否が止まった座席を名指しするのは、active な座席が 1 つも無いときだけ。止める側の窓は座席 N 個の stop と N 個の start(1 座席 60 秒まで)で、数分になりうる。同じ台帳を複数の段で変える `merge`・`regate`・`gate`・`cleanup` は 3 つの signal を `signal.NotifyContext` で受けるが、`org_reseat.go` に signal の扱いは無い | `internal/org/reseat.go:477-489`(`reseatTargets`)、`reseat.go:898-901`(`stopAll`、`startAll`、`finish`)、`internal/cli/org_reseat.go:185`(`rt.Reseat(params)`)、`internal/cli/org_merge.go:263`、`org_regate.go:214`、`org_gate.go:104`、`org_cleanup.go:165`。`Stop` と `Spawn` は ctx を取らないので、`NotifyContext` をそのまま渡しても止められない | 次のどちらか。(a) CLI が、plan のあと `Reseat` が返るまでの間 3 つの signal を受けて捨て、1 行(「reseat の途中です。終わるまで待ちます」)を出す。(b) 最初の stop の前に `reseat=<id>` つきの意図の record(例 `reseat_started`)を 1 本書き、`reseated` が無いまま残った id を `ralph org status` か reseat の拒否の文が名指しする。どちらもやらないなら、`--help` と skill に「途中で切れたら `ralph org status` で止まった座席を見て、`--seat` で名指しして打ち直す」と書き、tech-debt に 1 行足す |
| L1 | LOW | Exception handling | 役割の雛形が無い座席(`--role <名前> --prompt ...` で立てた custom role)を reseat すると、`--prompt` が台帳から読めないので、指示の無い座席が立つ。`PlanReseat` も dry run も、この座席に何も言わない(`start: foo (custom, claude opus)` と出て、止めたあとで空の prompt の座席が立つ)。spec は `--prompt` が読めないことを認めているが(`spec` の 142 行)、止めの前の拒否や dry run の注意にはしていない。custom role の座席が feature org に居ることは、tech-debt の 219 行目の行が前提にしている | `reseat.go:377-381`(`SpawnParams` に `Prompt` が無い)、`spawn.go:1097-1107`(未知の role は `p.Prompt` だけを使い、空なら空のまま)、`prompts.go:107`(`prompts/<role>.md` が無いと `ok=false`) | `PlanReseat` の座席ごとの検査に、`RenderRolePrompt(role, ...)` の `ok` が false の role を止めの前に拒否する 1 件を足す。拒否しないなら、dry run の `start:` 行に注意を 1 行足す |
| L2 | LOW | Maintainability | dry run の herdr probe `orgReseatHerdrServerProbe` が、`ensureOrgHerdrServer` の前半(status の読みと 3 つの分岐)を写している。「`herdr status server --json` が無い」の warning の文は 2 か所に同じ文字列で書かれ、`ErrNotFound` と読めない場合の文も同じ形で 2 つある。gate の文言や分岐が変わると、dry run の言うことと本物の reseat が外れる(dry-run の divergence の型) | `internal/cli/org_reseat.go:224-240`、`internal/cli/org_herdr_server.go:127-143` | 「status を読んで running かを返す(warning と拒否の文は gate と同じ)」関数を 1 つ出し、gate と probe の両方から呼ぶ |
| L3 | LOW | Error text | reseat が、start --plan の言い方のまま拒否する道が 2 つある。(a) `org.FeatureRepoRoot` の拒否を `runOrgReseat` がそのまま返し、文が「start --plan makes the feature worktree from the main worktree」で始まる。reseat は worktree を作らない。(b) leader が対象でない reseat(implementer だけ、など)でも `startFeatureInput` が走り、`[reviewer]` の値が不正だと「the leader's task copies the [reviewer] values」と言う。(b) の検査自体は、予約の比較(`reservationDecision`)に leader の spawn が要るので必要で、文だけが合わない | `internal/cli/org_reseat.go:181-183`、`internal/org/statedir.go:197,200`、`internal/org/feature.go:413`、`reseat.go:340-343` | (a) は `runOrgReseat` で包んで「org: reseat: ...」にする。(b) は `refuse` の文に「leader は起こさないが、予約の比較に leader の起動の検査を通す」の 1 句を足す |
| L4 | LOW | Naming | 関数 `reseatSeat`(`*reseatLifeSeat` から `ReseatSeat` を作る)と型 `ReseatSeat` が、頭文字の大小だけで違う。grep で両方に当たり、`reseatSeatNow`・`reseatLifeSeat` も並ぶ | `reseat.go:71`(型)、`reseat.go:575`(関数)、`reseat.go:1013` | 関数を `newReseatSeat` にする |
| L5 | LOW | Readability | `PlanReseat` が 142 行(270-411)で、起動順の slice `plan.Start`、止める添字の `stops`、シミュレーション用の `sim` の 3 つを同時に持って回す。`plan.Start[0]` が leader であることに 2 か所が頼る(326 行と 338 行)が、その保証は別の関数 `reseatLeaderFirst`(556 行)にある。順が変わると、`Start` の pointer が leader でない座席につき、leader の座席の `Start` が nil のまま `startAll` の `*seat.Start` で nil を引く | `reseat.go:321-339`、`reseat.go:556-569` | `leaderIdx` を返す形にして(leader が対象でなければ -1)、326・338 行をそれで書く。leader 向けの組み立て(`leaderStart` と `checked`)と、非 leader の `Spawn` の組み立て(372-390 行)を、別の関数に分ける |
| L6 | LOW | Maintainability | 3 つの Go の注釈が `docs/plans/active/2026-11-12-org-reseat.md` を指す。`/pr` がこの plan を `archive/` へ動かすので、指す先が消える。repo に同じ劣化がすでにある(`merge.go:19` の `docs/plans/active/2026-10-17-org-auto-merge.md` は `archive/` にある) | `internal/org/reseat.go:22`、`internal/org/seat.go:166`、`internal/cli/org_reseat.go:16` | `plan 2026-11-12-org-reseat`(slug だけ)にするか、spec の `docs/specs/2026-11-12-org-reseat.md`(動かない)を指す |

## Positive notes

- `PlanReseat` の拒否の順が、関数の頭の注釈の番号(1 から 9)と、コードの並びで一致している。
- 止める前の再読(`reseatSeatNow`)は、index と event の全体の比較の両方を見る。外から `stopped`・`spawn_failed` を足す手(watchdog など)にも反応する。
- `Reseat` が `checkReseatFlags` をロックの前に置き(`dd204370`)、flag の誤りで `reseat.lock` を作らない。dry run は lock を取らない。
- `startAll` が `*seat.Start` と `*seat.Spawn` を値で写してから `Note` を足すので、`ReseatPlan` の pointer を書き換えない。
- 失敗の結果の語彙(`started`・`stopped`・`stop_failed`・`not_stopped`・`not_started`・`start_failed`)が、`reseated` の Details、CLI の行、`Retry` の対象の 3 つで同じ。
- `ReseatDecisionCommand` が、既存の `ApprovalDecisionCommand` と同じ作り(単引用符、座席 id は検証済み)で、reseat は送らず文を出すだけ。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| M1: 途中で切れた reseat が、止めた座席を止めたまま残し、打ち直しで拾えない | 数分の窓で SIGINT か端末の切断が来ると、非 leader の座席が止まったまま。`status` を見る人だけが気づく | 直すなら signal の扱いか意図の record が要り、spec の P4(`reseated` は 1 本)を変える | 実機で reseat の途中切断が 1 度でも起きたとき、または `ralph org status` に reseat の途中の印を足すとき | 本 report の M1 |
| L1: custom role の座席の `--prompt` が、reseat で空になる | 指示の無い座席が立つ | spec が `--prompt` を読めないものとして認めている | custom role の座席を feature org で使う報告があったとき | 本 report の L1、`docs/specs/2026-11-12-org-reseat.md` の 142 行 |

_(上の 2 行は、lead の指示「report だけをコミットする」に従って `docs/tech-debt/README.md` へはまだ足していない。直さずに持ち越す場合は、orchestrator が足す。)_

## Recommendation

- Merge: yes。CRITICAL と HIGH は無い。MEDIUM 1 件(M1)は、直すか、tech-debt に載せて持ち越すか、lead が決める
- Follow-ups: M1 の(a) か(b)。L1 の拒否。L2 の共通化。ほかの LOW(L3 から L6)は同じコミットで直せる。直したら、パイプラインは `/self-review` からやり直す(`post-implementation-pipeline.md`)
