# Test report: org-plan-approval

- Date: 2026-10-13(cycle 2。JST 09:31〜10:00)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(承認済み、digest 951cf09bd7bc)
- Tester: tester subagent (Claude Opus 5.5)。パイプラインの 2 回目(`.harness/state/standard-pipeline/cycle-count.json` は 2)。cycle 1 の結果は末尾の「付録: cycle 1」に残した
- Scope: HEAD ffd6b3ec(base 225dff27)。重点は cycle 2 の差分 `git diff b1f934c9..HEAD`(20 ファイル、+1,324/-94)。非テストのコードで変わったのは `internal/org/verbs.go`(send が宛先の座席を待ったあと、打つ前に承認を確かめ直す)、`internal/org/approval.go`(古い `PLAN:` の案内を最新の承認にだけ移し、今の digest の PlanChanged はその拒否を返す)、`internal/cli/org.go`(spawn と send の `--help`)の 3 つ。テストは 3 ファイルで +248/-1(消えた 1 行は doc のコメント)。足した top-level のテストは 4 本で、`TestCheckTaskPlanApproved_StaleOrMistypedDigest` の subtest が 4 つ増えた。behavioral test だけを実行した(静的解析は /verify の担当)
- Evidence(どれもコミットする。`git check-ignore` は rc 1):
  - `docs/evidence/test-2026-10-13-org-plan-approval-c2-run-test.txt`(`./scripts/run-test.sh` の出力)
  - `docs/evidence/test-2026-10-13-org-plan-approval-c2-go-test.txt`(`go test ./... -count=1 -v -cover`)
  - `docs/evidence/test-2026-10-13-org-plan-approval-c2-go-race.txt`(`go test -race -count=1 -v ./internal/org/ ./internal/cli/`)
  - `docs/evidence/test-2026-10-13-org-plan-approval-c2-mutation.txt`(mutant の一覧と結果、落としたテストの最初の行、生き残りの probe のテストの全文と結果、使ったスクリプト)
  - `docs/evidence/org-plan-approval-live-2026-10-13-c2.md`(実機確認の記録)と `docs/evidence/org-plan-approval-live-2026-10-13-c2-output.txt`(実機確認の全出力)
  - wrapper が書いた `docs/evidence/verify-2026-10-13-003210.log` は gitignore の対象なので、コミットしない
  - insight event: `docs/insights/events/2026-10-13-org-plan-approval.jsonl` に test の 1 行(`--cycle auto`)を足し、この報告と同じコミットに入れた
- テストもコードも変えていない。mutant の写しと probe のテストは scratchpad の `opa/c2/mut/` に置き、`go test -overlay` で使った。worktree で書いたのは、このレポート、上の evidence、insight event だけ

## Latest run: cycle 2(HEAD ffd6b3ec)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope。`scripts/ralph-config.sh` が分類されないので full に落ちた) | shell 40 本(PASS の行 4,081)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 7 分 25 秒、rc 0 |
| `go test ./... -count=1 -v -cover` | top-level 1,505、subtest 1,923 | top-level 1,502、subtest 1,923 | 0 | 3 | 2 分 12 秒(cli 129.4 s、org 31.1 s)、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/` | top-level 1,223、subtest 1,726 | top-level 1,222、subtest 1,726 | 0 | 1 | 2 分 16 秒(cli 131.3 s、org 34.4 s)、rc 0。`DATA RACE` は 0 件 |
| mutation(`go test -overlay`、HEAD のテスト、`./internal/org/` の全体) | 16 件 | red 14 件 | 生き残り 2 件(下の節。どちらも probe で閉じられる) | - | 1 件 42〜86 s、並列 2 |
| 実機確認(macOS、HEAD のビルド、隔離した herdr と stub のエージェント) | (f)〜(i) | すべて期待どおり | 0 | - | JST 09:37〜09:46 |

- `run-test.sh` の Go は `internal/org` だけを流し直し(31.3 s)、ほかの 7 パッケージは `(cached)` だった。キャッシュを使わない結果は 2 行目の `go test ./... -count=1` にある
- shell の Skipped は cycle 1 と同じ `tests/test-secret-scan-branch.sh` の 1 件(git 2.49.0 なので、2.41 より古い git の場合を飛ばす)。PASS の行は、`PASS: <数>` だけの集計の行を除いて数えた。cycle 1 の出力を同じ数え方で数えても 4,081 になる(cycle 1 の報告の 4,082 とは、数え方が 1 行ぶん違う)
- Go の件数は cycle 1(top-level 1,498、subtest 1,907)より、top-level が 4、subtest が 16 増えた。PASS か SKIP になったテストの名前を cycle 1 の出力と比べると、消えたものは 0、増えたものは 20 で、どれも cycle 2 で足したテストとその subtest だった
- Go の Skipped は cycle 1 と同じ 3 件(`-race` は org と cli だけなので 1 件)
- load average は、`run-test.sh` の開始時に 34.84(直前の 1 分は 55 前後)、`go test ./...` の開始時に 3.41、`-race` の開始時に 3.49 だった。前に flake を起こした doctor のテスト(`TestRunDoctor*`)と `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、どの実行でも落ちなかった

### Coverage

- Statement: `internal/org` 94.4%(cycle 1 は 94.3%)、`internal/cli` 86.6%、`internal/config` 93.3%、`internal/org/protocol` 100.0%。ほかの 4 パッケージ(driver 93.1%、insights 86.1%、scaffold 75.7%、upgrade 91.2%)は cycle 1 と同じ
- Function(cycle 2 で変わった関数): `latestApprovedDigest`、`checkPlanApproved`、`checkAnyPlanApproved`、`sendApprovalErr` は 100%。100% でないもの:
  - `checkTaskPlanApproved` 94.1%: 今の digest が先頭の 2 行でも PlanChanged でもない理由で断られたときの `default: return err`(`approval.go:514-515`)。今の digest はその path の最新の承認で、ファイルの digest そのものなので、ここに来るのは `currentPlanRef` が読んでから `checkPlanApproved` が読むまでのあいだに、ファイルが変わるか消えたときだけである
  - `checkSendApproval` 75.0%: manifest が読めないときの枝(`verbs.go:503-505`)
  - `Send` 96.8%: 通らないのは `findSeat` の読み取りの失敗(`verbs.go:355-357`)と、残りの時間が負のときに 0 にする行(`:418-420`)で、どちらも cycle 2 の前からある
  - `currentPlanRef` 90.0%: cycle 1 と同じ(ファイルが解決できない、読めない枝)
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: 関数ごとの数値は `go test -count=1 -coverprofile ./internal/org/`(rc 0、35.2 s、94.4%)で取り直した。通らない行は、profile で回数が 0 の区間から抜いた

### Mutation

各 mutant は、ソースの文字列(ファイルに 1 回だけ現れるもの。Q06 と Q08 は 2 か所)を置き換えた写しを `go test -overlay` で読ませた。worktree は書いていない。対象のテストは `./internal/org/` の全体で、16 件はどれもコンパイルできた(先に `go vet -overlay` で確かめた)。

| ID | 何を壊したか | 結果 | 落としたテスト |
| --- | --- | --- | --- |
| Q01 | `Send` の待ったあとの承認の確かめ直し(2 回目の `checkSendApproval`)を消す | red | `TestOrgSend_ApprovalCheck_AgainAfterTheWait`(`send = <nil>; want a refusal that is ErrPlanNotApproved`) |
| Q05 | 2 回目を型付きの TASK だけで回す(`--raw` では回さない) | red | 同じ |
| Q06 | 2 回目を `PaneSendText` のあと(Enter の前)に移す | red | 同じ(`herdr calls [… agent_wait pane_send_text]; want nothing typed`) |
| Q08 | 2 回目が、待つ前に読んだ manifest を使い回す | 生き残り | - |
| Q02 | 「今の digest がその path の最新の承認のときだけ案内を移す」条件を消す(cycle 1 の規則に戻す) | red | `TestCheckTaskPlanApproved_StaleOrMistypedDigest`((f) と (g) が、最初の承認へ案内する TaskHeader の拒否になった) |
| Q03 | 今の digest の PlanChanged を返す枝(N6)を消す | red | 同じ((h) の 2 つが TASK 自身の拒否になった) |
| Q10 | 今の digest の PlanHeader を返す枝を消す | red | 同じと `TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest` |
| Q09 | 今の digest が通るときに TaskHeader の拒否を返さない | red | 4 本(上の 2 本、`TestOrgSend_ApprovalCheck_PassesAndApprovedAgain`、`TestPlanNotApprovedError_OneNextStepPerKind`) |
| Q04 | `latestApprovedDigest` が最後でなく最初の承認を返す | red | Q09 と同じ 4 本 |
| Q07 | `latestApprovedDigest` がパスを見ない(どの計画でも最後の承認を返す) | 生き残り | - |
| Q11 | `checkAnyPlanApproved` が古い承認から見る(cycle 1 の R02 を cycle 2 のコードに合わせた) | red | `TestCheckAnyPlanApproved` |
| Q12 | `checkPlanApproved` が承認をパスで絞らない(cycle 1 の R12 を合わせた) | red | `TestCheckPlanApproved` |
| R03 | `Send` で見張りの関門と承認の検査の順を入れ替える(`checkSendApproval` に合わせて書き直した) | red | `TestOrgSend_WatchGateBeforeApprovalCheck` |
| R04 | dry-run の spawn で、機能の org の検査を容量の検査の前に置く | red | `TestOrgSpawn_CapacityCheckBeforeFeatureOrgChecks`(`dry_run=true`) |
| R05 | 本番の spawn で同じ入れ替え | red | 同じ(`dry_run=false`) |
| R11 | approve の、resolve の note が受信箱の上限を超える件を前もって断る検査を消す | red | `TestOrgApprove_RefusesAnItemItCannotResolve` |

cycle 1 の生き残り 4 件(R03、R04、R05、R11)は、HEAD のテストで 1 回ずつ流し、どれも cycle 2 で足したテストだけで落ちた。

#### 生き残りと probe

| ID | 何が変わるか | probe のテスト(scratch のみ) | HEAD | mutant |
| --- | --- | --- | --- | --- |
| Q08 | 待つ間に manifest が変わったときだけ違いが出る。待つ間に計画が改訂され、人がその版を承認し(`plan_approved` が足され)、leader が 2 行を書いた場合、最初の承認を名指す TASK への拒否は、HEAD では改訂版の digest へ案内する TaskHeader になる。Q08 では「承認のあとに変わった」になり、次の手は人が承認した改訂を戻すよう言う。どちらも断り、何も打たない | `TestProbe_SendRecheckReadsTheManifestAfterTheWait`(型付きと `--raw`) | PASS | FAIL(`changed after its approval: approved digest f05eab2b3a96, current digest fc9b51dd3eac; …`) |
| Q07 | 同じ org に別の計画の承認があり、それが最後に記録されたときだけ違いが出る。(a) 承認し直した計画の古い digest の TASK は、HEAD では最新の承認へ案内され、Q07 では「承認のあとに変わった」になる。(b) 承認のない計画を名指す TASK の文に、Q07 では別の計画の digest が `(its latest approval is of sha256:…)` として入る | `TestProbe_LatestApprovalIsPerPlanPath`(subtest 2 つ) | PASS | FAIL(2 つとも) |

2 件とも HEAD の動きを変える mutant で、テストを 1 本ずつ足せば閉じられる(等価ではない)。probe はコミットしていない。依頼が「コードを変えない」だったため。全文は mutation の evidence にある。どちらも、断ることと何も打たないことは変わらず、変わるのは拒否の文と次の手だけである。Q07 は、計画が 1 つの org(`ralph org start --plan` の使い方)では出ない。

### 実機確認

cycle 1 と同じ隔離の仕方で、新しい一時の repo と台帳を使った。HEAD ffd6b3ec のビルド、隔離した herdr 0.7.5 の `herdr server`、repo の root からの相対の `HERDR_SOCKET_PATH`、`osascript` を外した PATH、`claude` という名前の Go の stub と stub の agmsg。stub には、フラグのファイルがあるあいだ OSC のタイトルを `⠂ ` で始める仕掛けを足した。herdr の claude の検出規則は、これを working と読む(ralph を通す前に、使い捨ての workspace で idle → working → idle の切り替えと、working のあいだ `herdr agent wait` が待つことを確かめた)。利用者の herdr サーバー、`~/.config/herdr`、本物の agmsg の home、osascript には触れていない。モデルのセッションは起動していない。詳細と出力は `docs/evidence/org-plan-approval-live-2026-10-13-c2.md`。

下ごしらえとして、承認 A(digest 8e7b0c143bb6、e1)、implementer の spawn(pane w2:p3)、計画を変えて承認 B(3a36fa20e7a1、e2)、2 行を B で書いてコミット、の順に進めた。

| 項目 | 結果 |
| --- | --- |
| (f) A の版に戻してコミットし、TASK が B を名指す | 型付きと `--raw`、dry-run と本番の 4 通りとも rc 1 で、`changed after its approval: approved digest 3a36fa20e7a1, current digest 8e7b0c143bb6; refused by the approval check: the leader undoes the change, or sets the plan back to - Status: Draft …`。次の手に A の digest はない(文の全体でも、見つけたことの `current digest` に 1 回出るだけ)。TaskHeader の文と次の手も出ない。台帳は変わらず、打たれた行は 0 |
| (g) 同じ状態で digest を打ち間違える(8e7b0c143bb0) | dry-run と本番とも rc 1 で、`has no approval recorded for plan … sha256:8e7b0c143bb0 since its last disband (its latest approval is of sha256:3a36fa20e7a1); refused by the approval check: if the leader has not received an approving DECISION …`。A の digest は文に 1 回も出ない |
| (h) B の版に Progress checklist の 1 行を足してコミットし、古い A の TASK を送る | A の TASK(型付きの dry-run と本番、`--raw` の dry-run)、打ち間違えた digest の TASK、B の TASK の 5 通りとも rc 1 で、今の digest B の `changed after its approval outside its - Status: and - Approved: lines and checkbox marks, which its digest 3a36fa20e7a1 does not show …`。A の PlanChanged、NoApproval、TaskHeader の文は出ない。足した行を消すと、A の TASK は B へ案内され(TaskHeader)、B の TASK は通った |
| (i) 待ったあとの確かめ直し | implementer の stub を working にして、B を名指す正しい TASK を本番で送った。3 秒後も send は待っていて、何も打たれていない。(i-1) 型付きで、待つ間に本文を変えてコミット(digest 42388be72cb4)し、stub を idle に戻すと、0.4 秒で rc 1(`changed after its approval: approved digest 3a36fa20e7a1, current digest 42388be72cb4`)。(i-2) `--raw` で、待つ間に Progress checklist に 1 行足すと、0.2 秒で rc 1(digest の外が変わったの拒否)。どちらも台帳は変わらず、implementer の stub に打たれた行は 0。(i-3) 対照として何も変えないと、3.8 秒で rc 0、5 行が打たれ、`sent` が 1 行増えた。stub の仕掛けは 10 分ほどで動いた |
| 観察: (f) の状態の A の TASK と spawn | A を名指す TASK の dry-run と、implementer の spawn の dry-run は rc 0 で通った。A にもこの命の承認があり、ファイルが A の版に合うため(AC5 の「すべて揃えば今までどおり送れる」と AC7 のとおり) |
| 見張りとの重なり | (i-1) で計画を書いてからコミットするまでのあいだに、見張りが implementer の worktree の変更を scope_change と見て、on-demand watcher が `claude -p --model haiku --output-format json` を起動した。起動したのは PATH の上の stub(pid 43355)で、依頼の 75 行を読んで終わり、判定は `malformed claude envelope` の誤りになった。見張りの既存の動きで、この変更とは関係がない |
| 片づけ | org a を disband し、見張りを `--stop` で止め、herdr サーバーを止めた。cwd `/` から打った `pgrep -fl '<L>'` は rc 1、`pgrep -fl 'herdr server'` も rc 1。stub の 4 つ、herdr サーバー、見張りの pid を `ps -p` で引くと rc 1。cwd が `<L>` の下にあるプロセスは 0 個 |

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

テストの失敗はなかった。負荷による doctor のテストの flake も、今回は出なかった。

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 1 のテストが消えず、書き換えられていない | cycle 2 が触ったテストのファイル 3 つは、どれもこのブランチで足したもの。消えた 1 行は doc のコメント。PASS か SKIP になったテストの名前を cycle 1 と比べ、消えたものは 0 | `git diff b1f934c9..HEAD -- '*_test.go'`、go-test の evidence |
| cross-review cycle 1 の #1(待つ間の計画の変更) | 閉じた。`TestOrgSend_ApprovalCheck_AgainAfterTheWait` が通り、Q01・Q05・Q06 を落とした。実機の (i-1)・(i-2) も、何も打たずに断った | go-test と mutation の evidence、実機確認の記録 |
| cross-review cycle 1 の #2 と self-review の N7(前の承認の版に戻したファイルへの案内) | 閉じた。単体テストの (f)・(g) が通り、Q02 を落とした。実機の (f)・(g) でも、次の手は A を名指さなかった | 同上 |
| self-review の N6(digest の外の変更) | 閉じた。単体テストの (h) が通り、Q03 を落とした。実機の (h) も今の digest の PlanChanged だった | 同上 |
| /test cycle 1 の生き残り R03、R04、R05、R11 | 閉じた。どれも cycle 2 で足したテストで落ちた | mutation の evidence |
| 承認し直したあとの古い digest の案内(cycle 1 の実機の (c)) | 変わらない。`TestOrgSend_ApprovalCheck_PassesAndApprovedAgain` と `TestOrgSend_ApprovalCheck_StaleOrMistypedPlanDigest` が通る。実機の (h) の対照でも、A の TASK は B へ案内された | go-test の evidence、実機確認の記録 |
| 見張りの関門(PR #223)の TASK の拒否 | 通る(`TestOrgSend_WatchGate_RefusesATaskOnly`)。関門が承認の検査より先であることは `TestOrgSend_WatchGateBeforeApprovalCheck` が固定する | go-test の evidence |
| shell の全 suite | 40 本とも rc 0。PASS の行は cycle 1 と同じ 4,081 | run-test の evidence |

### Test gaps

- mutation の生き残り 2 件。Q08(2 回目の検査が manifest を読み直すこと)は、待つ間に manifest が変わる場合のテストがないので残った。Q07(最新の承認をパスごとに選ぶこと)は、テストの org がどれも計画を 1 つしか持たないので残った。どちらも上の probe を足せば閉じる。断ること自体は変わらず、変わるのは拒否の文と次の手だけ
- 100% でない枝: `checkTaskPlanApproved` の `default: return err`(2 回の読み取りのあいだにファイルが変わったときだけ通る)と、`checkSendApproval` の manifest の読み取りの失敗
- spawn と send の `--help` に足した文を固定するテストはない(verify の D2 のとおり)。この文の mutant は流していない
- 実機の (i) は、stub のタイトルで作業中に見せて確かめた。本物のモデルの seat が作業中に計画を書き換える流れと、Linux は確かめていない
- 観察: 承認 B のあとでも、ファイルを A の版に戻せば、A を名指す TASK と implementer の spawn は通る。検査が見るのは「この命の承認のどれかと合うこと」(AC5、AC7)で、それが最新の承認かどうかは見ない。雛形の手順 5 は「最後に承認された digest」を使うよう言い、cycle 2 の直しは拒否の案内をそれに合わせた。A を名指す TASK そのものは断らない。計画の範囲のとおりなので、記録だけにする
- cycle 1 の Test gaps のうち、`recordApprovals` の途中で追記が失敗する枝、`Approve` の 3 つのエラーの枝、本物のモデルの leader の手順、本物の agmsg での DECISION の受け渡し、別の herdr サーバーと pane の id が重なる場合、古い版の ralph で立てた org の更新の流れは、cycle 2 でも変わらない(付録を参照)

## Verdict

- Verdict: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、PASS の行 4,081、Go 8 パッケージ)。`go test ./... -count=1` は top-level 1,502 本が PASS で 0 fail、cycle 1 のテストは 1 本も消えていない。`-race`(org と cli)も 0 fail で、`DATA RACE` は 0 件。cycle 2 の mutant 16 件のうち 14 件が red で、cycle 1 の生き残り 4 件はどれも足したテストで落ちた。生き残った 2 件(Q07、Q08)はどちらも probe で閉じられる穴で、断ることは変わらない。実機では (f)〜(i) がすべて期待どおりで、片づけのあとに scratch のプロセスは残っていない
- Fail: 0
- Blocked: なし

## 付録: cycle 1

cycle 1 の header と本文を、そのとき書いたまま残す。本文の見出しは 1 段下げて「(cycle 1)」を付け、`## Verdict` は「cycle 1 の判定」に変えた。cycle 1 の生き残り 4 件(R03、R04、R05、R11)が閉じたことは、上の cycle 2 で確かめた。

- Date: 2026-10-13(cycle 1。JST 06:27〜07:50)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md
- Tester: tester subagent (Claude Opus 5.5)。pipeline cycle 1
- Scope: HEAD 9dcdbfab(base 225dff27)。変えた Go のファイルは 24 で、そのうちテストは 13(ほかに leader の雛形、`scripts/ralph-config.sh` と template の写し、`templates/base/ralph.toml`、skill・rule・文書。`git diff --shortstat 225dff27..HEAD -- '*_test.go'` は +3,845/-46)。足したテストの関数は 54 本で、消したものはない。behavioral test だけを実行した(静的解析は /verify の担当)。依頼のとおり、`run-test.sh`、`go test ./...`、`go test -race`(org と cli)、`go test -overlay` の mutation、ビルドしたバイナリと一時の台帳での実機確認を行った
- Evidence(どれもコミットする。`git check-ignore` は rc 1):
  - `docs/evidence/test-2026-10-13-org-plan-approval-run-test.txt`(`./scripts/run-test.sh` の出力)
  - `docs/evidence/test-2026-10-13-org-plan-approval-go-test.txt`(`go test ./... -count=1 -v -cover`)
  - `docs/evidence/test-2026-10-13-org-plan-approval-go-race.txt`(`go test -race -count=1 -v ./internal/org/ ./internal/cli/`)
  - `docs/evidence/test-2026-10-13-org-plan-approval-mutation.txt`(mutant の一覧と結果、生き残りの再実行、probe のテストの全文と結果、使ったスクリプト)
  - `docs/evidence/org-plan-approval-live-2026-10-13.md`(実機確認の記録)と `docs/evidence/org-plan-approval-live-2026-10-13-output.txt`(実機確認の全出力)
- テストもコードも変えていない。mutant の写しと probe のテストは scratchpad の `opa/mut/` に置き、`go test -overlay` で使った。worktree で書いたのは、このレポート、上の evidence、insight event だけ

### Test execution(cycle 1)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(changed scope。`scripts/ralph-config.sh` が分類されないので full に落ちた) | shell 40 本(PASS の行 4,082)、Go 8 パッケージ | すべて | 0 | shell 1(下の注) | 12 分 49 秒、rc 0 |
| `go test ./... -count=1 -v -cover` | top-level 1,501、subtest 1,907 | top-level 1,498、subtest 1,907 | 0 | 3 | 3 分 02 秒(cli 179.3 s、org 53.6 s)、rc 0 |
| `go test -race -count=1 -v ./internal/org/ ./internal/cli/` | top-level 1,219、subtest 1,710 | top-level 1,218、subtest 1,710 | 0 | 1 | 2 分 59 秒(cli 171.9 s)、rc 0。`DATA RACE` は 0 件 |
| mutation(`go test -overlay`、HEAD のテスト) | 71 件(コンパイルできなかった 2 件は直して数え直した) | red 67 件 | 生き残り 4 件(下の節。どれも probe で閉じられる) | - | 1 件 4〜114 s(org は全体、cli は対象のテスト、config は全体)。生き残りは cli の全体でも流した |
| 実機確認(macOS、HEAD のビルド、隔離した herdr と stub のエージェント) | (a)〜(e) と AC14 | すべて期待どおり | 0 | - | JST 06:37〜06:44 |

- shell の Skipped は `tests/test-secret-scan-branch.sh` の 1 件で、実物の git が 2.41 より古い場合を扱う(git 2.49.0 では飛ばし、stub の古い git が同じ場合を覆う)。`tests/test-lib-json.sh` と `tests/test-pre-bash-guard.sh` の集計の行 `SKIP: 0` は数えていない
- Go の Skipped は、`TestEscalate_NoDesktopNotificationOffDarwinIsSkipped`(darwin では飛ばす)と、scaffold の `TestBaseFS_WithMockFS`・`TestAvailablePacks_WithMockFS` で、base と同じ。`-race` は org と cli だけなので 1 件
- load average は、`go test ./...` の開始時に 3.09、`-race` の開始時に 8.20 だった。前に flake を起こした `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` と `TestRunWatcher_TimeoutIndependentOfSmallInterval` は、3 つの実行のどれでも落ちなかった

### Coverage(cycle 1)

- Statement: `internal/org` 94.3%、`internal/cli` 86.6%、`internal/config` 93.3%、`internal/org/protocol` 100.0%。ほかの 4 パッケージ(driver 93.1%、insights 86.1%、scaffold 75.7%、upgrade 91.2%)は base と同じ
- Function(この PR で足した関数): `approval.go` と `approve.go` の 41 関数のうち 30 関数が 100%。`spawnFeatureSeatErr`、`sendApprovalErr`、`seatRoleFromEvents`、`RequiresApproval`、`featureBindingFromTokens`、`validateFeatureBinding`、`featureApprovalMode`、cli の `newOrgApproveCmd`・`printApprovedPlans`・`withMainWorktreeOrgLimits`、config の `Default` も 100%。100% でないもの:
  - `recordApprovals` 76.5%: 何件かを書いたあとに追記が失敗し、書けた ID を `recorded: …` と出す枝(`approve.go:319-324`)。`TestOrgApprove_AppendFailureRecordsNothing` は 1 件目で失敗させるので、この枝を通らない
  - `approvalCheckErrorOf` 75%: `approvalCheckError` でないエラーを Other に包む行(`approval.go:445`)
  - `Approve` 89.3%: 受信箱がない、manifest が読めない、lock が取れない、の 3 つのエラーの枝
  - `ApprovalDecisionCommand` は org の profile では 0% だが、cli のテスト(`TestOrgApprove_RecordsResolvesAndPrintsTheDecision`、`TestOrgApprove_TheDecisionCommandRunsInAPOSIXShell`、`TestOrgApprove_TheDecisionCommandRepeatsAnExplicitStateDir`)が出力を照合している。cli の profile は cli の文だけを数えるので、ここには出ない
  - `approveCandidates` 92.0%: resolve の note が受信箱の上限を超える件を前もって断る枝(`approve.go:263-266`)。mutation の生き残り R11 と同じ場所
  - ほかに 90〜93% のもの(`currentPlanRef`、`approvalNextStep`、`recordUnderLock`、`checkApproveCandidate`、`checkApprovedPlanFile`、`resolvePlanFile`)は、ファイルと manifest の読み取りのエラーの枝と、`<digest>` を補う 1 行が残る。config の `Load` は 93.5%
- Branch: Go の標準ツールには branch coverage がないので測っていない
- Notes: cli の profile は、doctor の 5 秒の timeout を避けるため `-skip '^TestRunDoctor'` で取った。全体の数値(86.6%)は上の `go test ./...` のもの

### Mutation(cycle 1)

各 mutant は、ソースの 1 か所(ファイルに 1 回だけ現れる文字列)を置き換えた写しを `go test -overlay` で読ませた。worktree は書いていない。並列は 3。対象のテストは、org の mutant が `./internal/org/` の全体、config が `./internal/config/` の全体、cli が `-run '^(TestOrgStartPlan_|TestOrgStartHelp_|TestOrgStatus_|TestOrgApprove|TestOrgSendSpawn_)'`。

| 群 | 何を壊したか | 件数 | red |
| --- | --- | --- | --- |
| P(`PLAN:` のヘッダとパス) | ヘッダが 2 行でも通す、digest の形を見ない、`..`・絶対パス・worktree の外の symlink・普通でないファイルを通す、`## ` で 2 行の読み取りを止めない、日付を見ない | 8(P05 は P05b に直した) | 8 |
| H(内容の hash、今の命の承認) | checkbox の印を hash に入れる、2 行の判定を行番号でなく接頭辞にする、disband の境を見ない、dry-run・別の org の記録を数える | 5(H02 は H02b に直した) | 5 |
| K(検査) | digest の照合、ファイルの digest、2 行、内容の hash、Status、Approved の digest、Status の行がない、最後の承認から探す、次の手の種類の選び方、不完全な結びつき | 12 | 12 |
| B(結びつきの mode) | `RequiresApproval` の条件 2 通り、`approval=` が壊れた結びつき | 3 | 3 |
| S(spawn) | leader がいない機能の org、implementer に限る条件 2 通り、leader の判定、dry-run と本番の検査の削除、`spawnPrecheckErr` | 7 | 7 |
| V(send) | dry-run で検査しない、implementer に限らない、知らない座席を通す、spawn 中の座席の role、split で検査する、`--raw` で見張りの関門を通す | 6 | 6 |
| T(tripwire) | 検査しない、active でない座席の pane でも断る、受信箱を読んでから検査する | 3 | 3 |
| A(approve) | 全部か無しかを崩す 2 通り、resolved・QUESTION でない件、digest、不完全な結びつき、`herdr_pane`、ID の重複、resolve しない | 9 | 9 |
| F・L・C(start と設定) | 空の mode を split にする、雛形の行を per-plan に固定する、main worktree の mode を渡さない、空の mode を通す、既定を split にする | 5 | 5 |
| R(2 回目。検査の順と細かな条件) | R01〜R13 | 13 | 9 |

- コンパイルできなかった P05 と H02(使わない変数)は、変数を `_ =` で使う P05b と H02b に直し、どちらも red になった。数に入れたのは直したほうだけ
- A01(全部か無しかの検査を消し、エラーのある件も記録に渡す)は、`TestOrgApprove_Refusals` が nil pointer の panic で落ちて red になった(6 s)。主張で落としたのではない。同じ規則は A02(エラーのない件だけを記録する)を `TestOrgApprove_AllOrNothing` が主張で落とす
- 生き残った 4 件は、cli の全体(`./internal/cli/`)でも流した。R05 と R11 は生き残った。R03 と R04 は `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` だけで落ち、その理由は `claude --version timed out after 5s`(負荷による timeout。このテストは HEAD で単独なら 3 回とも通った)だったので、`-skip '^TestRunDoctor'` で流し直し、生き残った

#### 生き残りと probe(cycle 1)

| ID | mutant | 何が変わるか | probe のテスト(scratch のみ) | HEAD | mutant |
| --- | --- | --- | --- | --- | --- |
| R03 | `Send` で、見張りの関門と承認の検査の順を入れ替える | 見張りも承認もないとき、出る拒否が見張りの関門のものから承認の検査のものに変わる。どちらも何も記録しない | `TestProbe_SendWatchGateBeforeApprovalCheck`(型付き、`--raw`、dry-run、両方の 4 通り) | PASS | FAIL(4 通りとも承認の検査の文) |
| R04 | dry-run の spawn で、機能の org の検査を上限(`max_seats`)の検査の前に置く | 上限に届いていて承認もないとき、出る拒否が `max_seats 1 reached` から承認の検査のものに変わる | `TestProbe_SpawnCapacityBeforeFeatureCheck`(`dry_run=true`) | PASS | FAIL |
| R05 | 本番の spawn で同じ入れ替え | R04 と同じ。どちらの順でも `rejected` と receipt を書く | 同じテスト(`dry_run=false`) | PASS | FAIL |
| R11 | approve の、resolve の note が受信箱の上限(500 字)を超える件を前もって断る検査を消す | 計画のパスが長い(note が 500 字を超える)件で、`plan_approved` を書いたあとに resolve が失敗し、記録だけが残る | `TestProbe_ApproveRefusesAnItemItCannotResolve`(パス 490 字ほど) | PASS | FAIL(件を承認して記録した) |

4 件とも、HEAD の動きを変える mutant で、テストを 1 本足せば閉じられる(等価ではない)。probe はコミットしていない。依頼が「コードを変えない」だったため。全文は mutation の evidence にある。R03・R04・R05 は検査の順で、どちらの順でも断ることと、何も書かないか `rejected` を書くかは変わらない。変わるのは、打った側に見える理由と次の手だけ。R11 は全部か無しかの規則の穴にあたるが、note(`approved <パス> sha256:<12 桁>`)が 500 字を超えるのは、計画のパスが 471 字を超えるときだけ。

### 実機確認(cycle 1)

HEAD のビルドと一時の台帳で行った。herdr は 0.7.5 の `herdr server` を隔離して起動した。XDG の 5 つのディレクトリと `HERDR_SOCKET_PATH` は scratch の下に置き、`SHELL=/bin/bash` とした。ソケットの絶対パスは `sun_path` の 104 バイトに入らないので、repo の root からの相対パスにして、ralph のコマンドはすべて root で打った。エージェントは、`claude` という名前の Go の stub(argv と打たれた行を記録し、stdin を読むだけ。モデルにもネットワークにもつながらない)で、agmsg も引数を記録して exit 0 する stub。PATH から `osascript` を外した。利用者の herdr サーバー、`~/.config/herdr`、本物の agmsg の home には触れていない。詳細と出力は `docs/evidence/org-plan-approval-live-2026-10-13.md`。

| 項目 | 結果 |
| --- | --- |
| (a) 承認がないうち | implementer の spawn(本番、dry-run)と TASK(型付き、`--raw`、dry-run)が、どれも rc 1 で `refused by the approval check` と「記録がない」場合の次の手を出した。TASK の拒否は台帳を変えず、pane に何も打たなかった |
| (b) approve のあと | `PLAN:` と `VISUAL:` の付いた QUESTION(e1)を `ralph org approve e1` が承認し、rc 0。`plan_approved` を 1 行書き、e1 を resolve し、計画のファイルは変えなかった(sha256 が同じ)。2 行を書く前の spawn は PlanHeader の次の手で断られ、2 行を書いてコミットしたあと、implementer の spawn と正しい `PLAN:` の TASK(dry-run の型付きと `--raw`、本番)が通った |
| (c) 計画が変わったあと | 古い digest の TASK と spawn が「承認のあとに変わった」の次の手で断られた。Draft に戻して承認し直したあと(e2)、古い digest の TASK は 2 段で断られた。2 行が古いうちは PlanHeader の次の手、2 行を書いたあとは TaskHeader の次の手で、どちらも新しい digest 3a36fa20e7a1 を名指した。新しい digest の TASK と spawn は通った |
| (d) 座席の pane からの approve | `HERDR_PANE_ID` を leader の pane(w2:p2)にしても implementer の pane(w2:p3)にしても rc 1 で断られ、manifest・受信箱・receipts は行数も hash も変わらなかった。どの座席でもない pane(w99:p9)からの approve は通り、`herdr_pane=set` を記録した |
| AC14 | checkbox の印は通り、`## Progress checklist` に足した 1 行(digest は変わらない)は TASK も spawn も断られ、その行を消すと通った |
| (e) split と leader のいない org | `--config` で split にした org b は、承認なしで implementer の spawn と `PLAN:` のない TASK が通った。b の leader を止めたあとの reviewer の spawn は、本番も dry-run も rc 1 で断られた |
| 片づけ | 2 つの org を disband し、見張りを `--stop` で止め、herdr サーバーを止めた。cwd `/` から打った `pgrep -fl '<scratch>'` は何も返さず rc 1 |

### Failure analysis(cycle 1)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

テストの失敗はなかった。mutation の途中で、cli の全体で流した R03 と R04 を `TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected` が落とした。理由は stub の `claude --version` の 5 秒の timeout で、mutant とは関係がない(このテストは HEAD で単独なら 3 回とも通った)。doctor のテストの負荷による flake は前から知られている。

### Regression checks(cycle 1)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| 既存のテストが書き換えずに通る(AC11) | 書き換えた既存のテストは plan の Progress checklist に書かれた 15 本と helper 1 つだけ。結びつきの期待に `approval=per-plan` を足した 11 本と `testFeature`、per-plan の org で implementer に TASK を送るので承認の記録を足した 2 本、leader が動いている状態で seat を spawn するよう直した 1 本、手順の番号の繰り下げで固定の語句 2 つを直した `TestRenderRolePrompt_Leader_FeatureOrgProcedure`。`git diff` に `-func Test` の行はなく、消したテストはない | `git diff 225dff27..HEAD -- '*_test.go'`、plan の Progress checklist(2026-10-13 の AC11 の行) |
| 既定値の一致(`Default()`、`templates/base/ralph.toml`、`scripts/ralph-config.sh`) | 通る(`TestDefaultsLockStep`、`TestDefault_OrgApproval`)。C02(既定を split にする)は 3 本で落ちた | go-test の evidence、mutation の evidence |
| 計画の digest と `scripts/plan-visual.sh digest` の一致 | 通る(`TestPlanDigest_MatchesScript`) | go-test の evidence |
| 見張りの関門(PR #223)の TASK の拒否 | 通る(`TestOrgSend_WatchGate_RefusesATaskOnly`)。V06(`--raw` で関門を通す)はこのテストでも落ちた | go-test の evidence、mutation の evidence |
| 機能の org の結びつきと予約(PR #216、#218) | 通る(`TestReservationFromEvent_DamagedBindingMatchesNothing`、`TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused` など)。R08 と S02 を落とした | go-test の evidence、mutation の evidence |
| leader の雛形の描画 | 通る(`TestRenderRolePrompt_*` と `TestRenderRolePrompt_Leader_FeatureOrgPlanApproval`) | go-test の evidence |
| shell の全 suite | 40 本とも rc 0 | run-test の evidence |

### Test gaps(cycle 1)

- mutation の生き残り 4 件(R03、R04、R05、R11)。上の probe の 3 本を足せば閉じる。R11 は全部か無しかの規則の端(note が 500 字を超える長いパス)にあたる
- 100% でない枝: `recordApprovals` の、何件かを書いたあとに追記が失敗する枝。`Approve` の 3 つのエラーの枝(受信箱がない、manifest が読めない、lock が取れない)
- 実機で確かめていないもの: 本物のモデルの leader が雛形の手順(図解ページ、QUESTION、DECISION を待つ、2 行のコミット)を自分で進めること。本物の agmsg での DECISION の受け渡し。Linux。別の herdr サーバーの pane と id が重なる場合(tripwire は pane の id だけを比べる)
- 承認の前の reviewer への TASK は、unit と CLI のテスト(`TestOrgSend_ApprovalCheck_NotApplied`)だけで確かめた。実機の (e-5) で打ったときは、org a に承認の記録がもうあった
- 古い版の ralph で立てた、`approval=` のない結びつきの org の動き(検査しない)は B02 を落とすテストで固定されているが、実機の更新の流れ(release と upgrade)は確かめていない

### cycle 1 の判定

- cycle 1 の判定: pass
- Pass: `./scripts/run-test.sh` は rc 0(shell 40 本、PASS の行 4,082、Go 8 パッケージ)。`go test ./... -count=1` は top-level 1,498 本が PASS、0 fail。`-race`(org と cli)も 0 fail で、DATA RACE は 0 件。mutant 71 件のうち 67 件が red で、生き残りの 4 件はどれも probe で閉じられる穴(検査の順 3 件と、note の長さの前もった検査 1 件)。実機では (a)〜(e) と AC14 がすべて期待どおりで、片づけのあとに scratch のプロセスは残っていない
- Fail: 0
- Blocked: なし
