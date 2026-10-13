# 機能の計画の承認の実機確認(org-plan-approval、/test cycle 2)

- Date: 2026-10-13(実機確認は JST 09:37〜09:46)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(cycle 2 の直し。cross-review cycle 1 の #1 と #2、self-review の N6)
- 報告: docs/reports/test-2026-10-13-org-plan-approval.md(cycle 2 の節)
- cycle 1 の記録: docs/evidence/org-plan-approval-live-2026-10-13.md
- 全出力: docs/evidence/org-plan-approval-live-2026-10-13-c2-output.txt。スクリプト、stub の事前確認、各段の出力、stub のログ、最後の manifest・受信箱・receipts・見張りのログを置いた。下の表の引用は、そこから要る行を抜いたもの
- 目的: HEAD ffd6b3ec のビルドで、次の 4 つを確かめる。(f) 承認 A、承認 B の順のあとファイルを A の版に戻すと、B を名指す TASK は「承認のあとに変わった」で断られ、次の手は A を名指さない。(g) 同じ状態で digest を打ち間違えた TASK は、その TASK 自身の「記録がない」で断られる。(h) B の版に Progress checklist の 1 行を足すと、古い A の TASK は今の digest(B)の「digest の外が変わった」で断られる。(i) send が作業中の implementer を待つあいだに計画が変わると、send は何も打たずに断る

## 環境

cycle 1 と同じ隔離の仕方で、場所だけを新しくした(`<L>` は `<scratch>/opa/c2/live`)。一時の repo と台帳は作り直し、cycle 1 のものは使っていない。

| 項目 | 値 |
|---|---|
| ralph | HEAD ffd6b3ec を `go build -ldflags "-X main.version=c2-ffd6b3ec" -o <scratch>/opa/c2/ralph ./cmd/ralph` でビルドしたもの |
| 起こし方 | すべて `<L>/env.sh` を通し、cwd は一時の repo の root。`env -i` のあとに渡すのは `HOME=<L>/home`、`SHELL=/bin/bash`、`USER=tester`、XDG の 5 つ(`<L>/xdg/` の下)、`HERDR_SOCKET_PATH=.harness/state/herdr.sock`(repo の root からの相対パス)、`RALPH_ORG_AGMSG_HOME=<L>/agmsg` などだけ |
| PATH | `<L>/stub`(stub の `claude` と `codex`)と `<L>/bin`。`<L>/bin` は cycle 1 の 964 個のシンボリックリンク(`/usr/bin` と `/bin` の全項目から `osascript` を外し、herdr を足したもの)を写し、ralph だけを cycle 2 のビルドに向けた |
| herdr | 0.7.5。隔離した `herdr server`(pid 96472)を一時の repo の root で起動した。起動時に herdr 自身が agent の検出規則を取りに行き、`<L>/xdg/state/` の下に置いた |
| stub のエージェント | cycle 1 の Go の stub に 1 つ足した。`<L>/busy/<pane>`(`HERDR_PANE_ID` の `:` を `_` にした名前)があるあいだ、OSC のタイトルを `⠂ stub claude` にする。herdr の claude の検出規則では、タイトルが点字の文字と空白で始まると working(規則 `osc_title_working`、優先度 1100)、`✳ ` で始まると idle(`osc_title_idle`、優先度 250)になる。stub は 200 ms ごとにファイルを見て、変わるたびにログに書く。モデルにもネットワークにもつながらない |
| stub の事前確認 | ralph を通す前に、使い捨ての workspace(w1:p1)で `herdr agent start --kind claude` を打った。`idle` → フラグを置くと `working`(タイトル `⠂ stub claude`)→ 消すと `idle` と読まれた。フラグがあるあいだの `herdr agent wait --until idle --until done --timeout 3000` は rc 1(timeout)、消したあとは rc 0 だった |
| agmsg | `scripts/*.sh` はどれも、呼ばれた引数を `<L>/agmsg/calls.txt` に書いて exit 0 する stub |
| 台帳 | 既定の `<L>/repo/.harness/state/org`。見張り `ralph org watch --all --detach --state-dir <台帳>`(pid 33391)を最初に起動した |
| 分割計画 | cycle 1 と同じ `demo.md`(機能 a と b、digest b624ce5d96c2)。使ったのは機能 a だけ |
| 触れていないもの | 利用者の herdr サーバー、`~/.config/herdr`、本物の agmsg の home。後の 2 つには 09:30 以降に変わったファイルがない(`find -newermt`)。受信箱の `notified` には `osascript: failed: exec: "osascript": executable file not found in $PATH` が残った |

表の「台帳」は manifest・受信箱・receipts の 3 つのファイルで、行数と、3 つをつないだ内容の sha256 の先頭 16 桁で比べた。拒否の文は `<見つけたこと>; refused by the approval check: <次の手>; the leader never runs ralph org approve itself and never sends the TASK with --raw` の形をしている。「次の手」は目印 `refused by the approval check: ` より後ろの部分を指す。

## 下ごしらえ(承認 A、承認 B)

| 確認 | 結果 |
|---|---|
| `start --plan … --feature a` | rc 0。leader の pane は w2:p2。結びつきは `… approval=per-plan` |
| 計画 A | worktree の `docs/plans/active/2026-10-13-a.md` を Draft と TBD で書いてコミット(9741e49)。digest は 8e7b0c143bb6 |
| 承認 A | QUESTION(e1)を `ralph org approve e1` が承認した(rc 0)。2 行を A で書いてコミット(fef157e)。この版を A の版として写しておいた |
| implementer の spawn | rc 0、pane は w2:p3。A を名指す TASK の `--dry-run` は rc 0 |
| 計画 B | 本文を変え、2 行を Draft と TBD に戻してコミット(937727a)。digest は 3a36fa20e7a1 |
| 承認 B | QUESTION(e2)を `ralph org approve e2` が承認した(rc 0)。2 行を B で書いてコミット(14677e5)。この版を B の版として写しておいた |
| 記録の順 | `plan_approved` はファイルの順に、00:41:59Z の A(inbox e1)、00:42:07Z の B(inbox e2)の 2 行 |
| 対照 | B を名指す TASK の `--dry-run` は、型付きも `--raw` も rc 0 |

## 結果

### (f) ファイルを A の版に戻し、TASK が B を名指す

A の版を書き戻してコミットした(981e338)。digest は 8e7b0c143bb6(A)。

| 確認 | 結果 |
|---|---|
| B を名指す TASK(型付きの `--dry-run`、`--raw` の `--dry-run`、型付きの本番、`--raw` の本番) | 4 つとも rc 1。文は同じで、`org: send: TASK not sent: plan docs/plans/active/2026-10-13-a.md of org_id "a" changed after its approval: approved digest 3a36fa20e7a1, current digest 8e7b0c143bb6; refused by the approval check: the leader undoes the change, or sets the plan back to - Status: Draft and - Approved: TBD and asks for a new approval with a new QUESTION; …` |
| 次の手に A があるか | ない。A の digest は文の全体で 1 回、見つけたことの `current digest 8e7b0c143bb6` にだけ出る。`sha256:8e7b0c143bb6` は 0 回。`the TASK's PLAN: header names` と `changes the TASK's PLAN: header`(TaskHeader の文と次の手)も 0 回 |
| 台帳と pane | 4 回とも、台帳は 19 行・6 行・2 行、hash 9ee169b83d053322 のまま。stub に打たれた行は 0 |

cycle 1 の (c) の規則では、この状態の B の TASK は TaskHeader の拒否になり、次の手が `PLAN: … sha256:8e7b0c143bb6` へ変えるよう言った(mutation の Q02 がこの規則に戻すもので、単体テストの (f) で落ちる。報告の cycle 2 の節を参照)。

同じ状態で、次の 2 つは通った。どちらも `--dry-run` で打った。

| 確認 | 結果 |
|---|---|
| A を名指す TASK | rc 0、`sent message to seat "implementer"` |
| implementer の spawn(implementer-f) | rc 0、`dry_run=true` |

A にもこの命の承認があり、ファイルが A の版と合うので、検査は通す。計画の AC5 と AC7(最後の disband より後の承認のどれかが計画のファイルに合えば通す)のとおりで、cross-review の #2 の直しが変えたのは、断るときの案内の先だけである。

### (g) 同じ状態で、TASK の digest を打ち間違える

| 確認 | 結果 |
|---|---|
| `sha256:8e7b0c143bb0`(A の最後の 1 桁を変えた)を名指す TASK(型付きの `--dry-run` と本番) | どちらも rc 1。`org: send: TASK not sent: org_id "a" has no approval recorded for plan docs/plans/active/2026-10-13-a.md sha256:8e7b0c143bb0 since its last disband (its latest approval is of sha256:3a36fa20e7a1); refused by the approval check: if the leader has not received an approving DECISION for its plan yet, it waits for the answer to its QUESTION, and if it has, that DECISION came without the record, so it escalates BLOCKED with ralph org escalate; …` |
| A を名指しているか | いない。A の digest は文の全体で 0 回。最新の承認として名指すのは B だけ。TaskHeader の文と次の手も 0 回 |
| 台帳と pane | 2 回とも、台帳は 27 行・6 行・3 行、hash f30e8d01e4af2b87 のまま(19 行から増えたのは、(f) の最後の 2 つの `--dry-run` が書いた分)。打たれた行は 0 |

### (h) B の版に Progress checklist の 1 行を足し、古い A の TASK を送る

B の版を書き戻し、`## Progress checklist` の下に `- skip the tests` を足してコミットした(e0f8292)。digest は 3a36fa20e7a1(B)のまま。

| 確認 | 結果 |
|---|---|
| A を名指す TASK(型付きの `--dry-run`、型付きの本番、`--raw` の `--dry-run`)、打ち間違えた digest の TASK(`--dry-run`)、B を名指す TASK(`--dry-run`) | 5 つとも rc 1。文は同じで、`org: send: TASK not sent: plan docs/plans/active/2026-10-13-a.md of org_id "a" changed after its approval outside its - Status: and - Approved: lines and checkbox marks, which its digest 3a36fa20e7a1 does not show (the digest skips the ## Progress checklist section and every - Status:, - Approved: and - Branch: line); refused by the approval check: the leader undoes the change, or sets the plan back to - Status: Draft and - Approved: TBD and asks for a new approval with a new QUESTION; …` |
| 文に出ないもの | `approved digest 8e7b0c143bb6`(A の PlanChanged)、`has no approval recorded`(NoApproval)、`the TASK's PLAN: header names`(TaskHeader)、A の digest は、5 回とも 0 回 |
| 台帳と pane | 5 回とも、台帳は 27 行・6 行・3 行、hash f30e8d01e4af2b87 のまま。打たれた行は 0 |
| 対照: 足した行を消してコミット(47d2039)したあとの A の TASK | rc 1。`the TASK's PLAN: header names sha256:8e7b0c143bb6 of plan … and the plan's current approved digest is sha256:3a36fa20e7a1; refused by the approval check: the leader changes the TASK's PLAN: header to PLAN: docs/plans/active/2026-10-13-a.md sha256:3a36fa20e7a1 and sends the TASK again; …`。今のファイルが最新の承認 B の版なので、B へ案内する(cycle 1 の (c) と同じ) |
| 対照: 同じ状態の B の TASK | rc 0 |

### (i) 待ったあとの確かめ直し

各ケースの手順は同じにした。B の版に戻してコミットし、B を名指す TASK の `--dry-run` が通ることを確かめる。implementer の pane(w2:p3)の stub にフラグを置き、herdr が `working` と読むのを待つ。そのあと `ralph org send --timeout-ms 90000` の本番を裏で打つ。3 秒後に send がまだ動いていることを確かめ、計画を変えてコミットする。最後にフラグを消す。台帳と打たれた行は、フラグを置いたあと、send の前に取った。

| 確認 | (i-1) 型付き、待つ間に本文を変える | (i-2) `--raw`、待つ間に Progress checklist に 1 行足す | (i-3) 対照: 型付き、待つ間に何も変えない |
|---|---|---|---|
| 送る前 | `--dry-run` は rc 0。pane は `working '⠂ stub claude'` | 同じ | 同じ |
| 3 秒後 | send は動いている(プロセス 1 つ)。pane は working。打たれた行は増えていない | 同じ | 同じ |
| 待つ間の変更 | コミット 0cb78a7、digest は 42388be72cb4 になった | コミット b1bb411、digest は 3a36fa20e7a1 のまま | なし |
| フラグを消したあと | 0.4 秒で返った。rc 1。`org: send: TASK not sent: plan … changed after its approval: approved digest 3a36fa20e7a1, current digest 42388be72cb4; refused by the approval check: the leader undoes the change, or sets the plan back to - Status: Draft …` | 0.2 秒で返った。rc 1。`… changed after its approval outside its - Status: and - Approved: lines and checkbox marks, which its digest 3a36fa20e7a1 does not show …; refused by the approval check: the leader undoes the change, …` | 3.8 秒で返った。rc 0。`sent message to seat "implementer"`。stub は Enter のあと working に変わらないので、cycle 1 と同じ「確かめられない」の警告が出た |
| 台帳 | 変わらない(29 行・6 行・3 行、hash a96aa2d6667c3a37) | 変わらない(30 行・6 行・4 行、hash 968cae0327ab446a) | `sent` が 1 行増えた(`submit_unconfirmed=true TYPE: TASK …`) |
| implementer の stub に打たれた行 | 0 | 0 | 5 行(`TYPE: TASK` から `implement the plan` まで)。00:43:24.65Z に `busy=false` を書き、00:43:24.84Z に 1 行目が打たれた |

implementer の stub(pid 35803)のログには、`busy=true` と `busy=false` が 3 回ずつと、(i-3) の 5 行だけがある。(i-1) と (i-2) の TASK は 1 行も打たれていない。

2 回目の検査で断られる TASK は、herdr の `agent wait` を呼んで待ったあとに断られる。打たず、何も記録しない点は 1 回目の拒否と同じである(verify の D9 の読み)。

全出力の (i) の「lines typed」の数は、すべての stub のログを足した数である。(i-1) のあと 0 から 75 に増えたのは、見張りの on-demand watcher が起動した `claude -p --model haiku --output-format json`(PATH の上の stub、pid 43355)が、stdin で渡された判定の依頼 75 行を読んだ分である。見張りは、(i-1) で計画を書いてからコミットするまでのあいだに、implementer の worktree の ` M docs/plans/active/2026-10-13-a.md` を scope_change と見た。stub は JSON を返さないので、判定は `malformed claude envelope` の誤りで終わり、receipts に `watchdog haiku unknown watcher_error` の 1 行が増えた。leader についての同じ見立ては `escalations.jsonl` に `leader_is_anomaly_subject` として残った。どれも見張りの既存の動きで、この変更とは関係がない。本物のモデルは起動していない(本物の `claude` は隔離した PATH にない)。

### 片づけ

| 確認 | 結果 |
|---|---|
| `ralph org disband --org-id a` | rc 0。implementer と leader を stopped にした。herdr の workspace は 0 個 |
| `ralph org watch --stop --state-dir <台帳>` | rc 0、`stopped ralph org watch --all (pid 33391)`。そのあとの `ralph status` は `org watch: not running (stopped at 2026-10-13T00:44:07Z)` |
| `herdr server stop` | rc 0 |
| 残ったプロセス | cwd `/` から打った `pgrep -fl "<L>"` は何も返さず rc 1。`pgrep -fl "herdr server"` も rc 1。stub の 4 つ(pid 6779、34410、35803、43355)、herdr サーバー(96472)、見張り(33391)を `ps -p` で引くと rc 1。cwd が `<L>` の下にあるプロセスは 0 個(`lsof -d cwd`) |

`ralph status` の表には、(f) の `--dry-run` の spawn が残した `implementer-f … spawned (active) [dry-run]` の行が出る。dry-run の記録で、pane もプロセスもない。

## 結論

- (f) 承認 A、承認 B の順のあとファイルを A の版に戻すと、B を名指す TASK は、型付きでも `--raw` でも、dry-run でも本番でも、B 自身の「承認のあとに変わった」で断られた。次の手は変更を戻すか承認をやり直すかで、A の digest を名指さない。台帳は変わらず、何も打たれなかった
- (g) 同じ状態で打ち間違えた digest の TASK は、その digest の「記録がない」で断られた。文は最新の承認として B を名指し、A は出ない
- (h) B の版に Progress checklist の 1 行を足すと、A の TASK も打ち間違えた digest の TASK も B の TASK も、今の digest B の「digest の外が変わった」で断られた。行を消すと、A の TASK は B へ案内され、B の TASK は通った
- (i) 作業中の implementer へ正しい TASK を送り、待つ間に計画を変えると、send は stub が idle に戻ったあと 1 秒以内に断り、何も打たず何も記録しなかった。本文の変更でも Progress checklist の 1 行でも、型付きでも `--raw` でも同じだった。変えなかった対照は送られた
- 観察: (f) の状態では、A を名指す TASK と implementer の spawn が通る。A にもこの命の承認があるためで、計画の AC5 と AC7 のとおりである
- 確かめていないもの: 本物のモデルの seat が作業中に計画を書き換える流れ。待つ間に承認の記録が増える場合(単体テストにもない。報告の cycle 2 の Test gaps を参照)。Linux
- 副作用: なし。書いたのは scratch の下だけで、利用者の herdr と agmsg の home のファイルは変わっていない
