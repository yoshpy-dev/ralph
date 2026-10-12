# 機能の計画の承認の実機確認(org-plan-approval、/test cycle 1)

- Date: 2026-10-13(実機確認は JST 06:37〜06:44)
- Plan: docs/plans/active/2026-10-12-org-plan-approval.md(Verify plan の Evidence to capture、AC2〜AC8、AC13、AC14)
- 報告: docs/reports/test-2026-10-13-org-plan-approval.md
- 全出力: docs/evidence/org-plan-approval-live-2026-10-13-output.txt。スクリプト、各段の出力、stub のログ、最後の manifest・受信箱・receipts を、出力のバイト列のまま置いた。下の表の引用は、そこから要る行を抜いたもの
- 目的: ビルドしたバイナリと一時の台帳で、次を確かめる。per-plan の機能の org では、承認の前の implementer の spawn と TASK が断られる。`ralph org approve` のあとは通る。計画が変わったあとと承認し直したあとは、決まった次の手で断られる。座席の pane から打った approve は断られる。split の org は承認なしで進み、leader のいない機能の org は座席を立てない。終わりに、プロセスが何も残っていないことを確かめる

## 環境

| 項目 | 値 |
|---|---|
| ralph | HEAD 9dcdbfab を `go build -o <scratch>/opa/ralph ./cmd/ralph` でビルドしたもの |
| 起こし方 | すべて `<L>/env.sh` を通した。`env -i` のあとに渡すのは `HOME=<L>/home`、`SHELL=/bin/bash`、`USER=tester`、XDG の 5 つ(`<L>/xdg/` の下)、`HERDR_SOCKET_PATH`、`RALPH_ORG_AGMSG_HOME=<L>/agmsg` などだけ |
| PATH | `<L>/stub`(stub の `claude` と `codex`)と `<L>/bin`。`<L>/bin` には `/usr/bin` と `/bin` の全項目へのシンボリックリンクを置き、`osascript` だけを外し、herdr(`/opt/homebrew/bin/herdr`)とビルドした ralph を足した。scratch の HOME の `.bash_profile` と `.bashrc` も、pane のシェルの PATH をこの 2 つにする |
| herdr | 0.7.5。隔離した `herdr server` を、一時の repo の root で起動した。XDG の設定・状態・データ・キャッシュ・実行時のディレクトリは `<L>/xdg/` の下 |
| herdr のソケット | `HERDR_SOCKET_PATH=.harness/state/herdr.sock`(repo の root からの相対パス)。scratch の下の絶対パスは 150 字を超えて `sun_path`(104 バイト)に入らず、herdr が起動しなかった。scratch の外にソケットもリンクも置かないよう相対パスにして、ralph のコマンドはすべて repo の root で打った。ralph が herdr を呼ぶとき、子のプロセスは ralph の cwd を引き継ぐ。見張りの子も repo の root で動いていた(`lsof` の cwd で確かめた)。`.harness/state/` は gitignore されている |
| stub のエージェント | Go の小さなバイナリ(全文は全出力の `stubsrc/main.go`)で、プロセス名が `claude` になる。起動時の argv と、打ち込まれた行を `<L>/stublog/` に記録する。OSC のタイトルを `✳ stub claude` にする(herdr の claude の検出規則は、タイトルが `✳ ` で始まれば idle と読む)。あとは stdin を読むだけで、モデルにもネットワークにもつながらない。ralph を通す前に、使い捨ての workspace で `herdr agent start --kind claude` を打ち、`agent_status: idle` と `interactive_ready: true` になることを確かめた |
| agmsg | `RALPH_ORG_AGMSG_HOME=<L>/agmsg`。`scripts/*.sh` はどれも、呼ばれた引数を `calls.txt` に書いて exit 0 する stub。本物の agmsg の home(`~/.agents/skills/agmsg`)は使っていない |
| repo | `<L>/repo`。`git init -b main` のあと `ralph init --yes` で scaffold し、コミットした。remote はない。`ralph.toml` は scaffold の既定のままで、`[org.approval] mode = "per-plan"` |
| 台帳 | 既定の `<L>/repo/.harness/state/org`(`source: git-main-worktree`)。見張りは `--state-dir` で同じ場所を指して起動した |
| 分割計画 | `<台帳>/splits/demo.md`。機能 a(`Reserve: internal/a/`)と機能 b(`Reserve: internal/b/`)で、どちらも `Depends on: none`。digest は b624ce5d96c2 で、`- Approved:` の行に書いた |
| 触れていないもの | 利用者の herdr サーバー、`~/.config/herdr`、本物の agmsg の home。後の 2 つには、06:25 以降に変わったファイルがない(`find -newermt`)。osascript は PATH にないので、escalate の `notified` の event には `failed: exec: "osascript": executable file not found in $PATH` が残った |

`<L>` は `/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/b9879111-7368-4e90-bbc8-067a69a8c6d8/scratchpad/opa/live`、`<scratch>` はその 2 つ上の scratchpad を指す。spawn は断られると stdout に `rejected: <エラー>` を出し、同じエラーを stderr にも出す。表の「台帳」は manifest・受信箱・receipts の 3 つのファイルで、行数と、3 つをつないだ内容の sha256 の先頭 16 桁で比べた。

## 手順

1. 分割計画を書き、`ralph org watch --all --detach --state-dir <台帳>` で見張りを起動した(pid 48215)
2. `ralph org start --plan <台帳>/splits/demo.md --feature a --driver claude --model sonnet` で機能 a の org を立てた。leader は stub の claude で、何もしない
3. leader がすることは外から打った。worktree の中で計画 `docs/plans/active/2026-10-13-a.md`(`- Status: Draft`、`- Approved: TBD`)を書いてコミットし、spawn・send・escalate を repo の root から打った
4. (a)〜(e) を順に打った。(d) は (c) の途中、2 回目の QUESTION を承認する前に打った
5. 2 つの org を disband し、見張りを `--stop` で止め、herdr サーバーを止めた

## 結果

### 立ち上げ

| 確認 | 結果 |
|---|---|
| `start --plan … --feature a` | rc 0。`spawned seat "leader" (org_id=a … pane_id=w2:p2 dry_run=false)`、worktree `<L>/repo/.claude/worktrees/org-a`、branch `feat/a` |
| 結びつき | `scope_reserved` の Details は `paths=internal/a/ split=demo feature=a digest=b624ce5d96c2 branch=feat/a approval=per-plan` |
| leader のタスク | 役割指示のファイルに `- 承認方式: per-plan` の行がある |
| 計画 | worktree でコミット 80dde2a。digest(`scripts/plan-visual.sh digest`)は 8e7b0c143bb6 |

### (a) 承認がないうちの implementer の spawn と TASK

| 確認 | 結果 |
|---|---|
| implementer の spawn(本番) | rc 1。`rejected: org: no plan of org_id "a" has been approved since its last disband; refused by the approval check: if the leader has not received an approving DECISION for its plan yet, it waits for the answer to its QUESTION, and if it has, that DECISION came without the record, so it escalates BLOCKED with ralph org escalate; the leader never runs ralph org approve itself`。manifest に `rejected`、receipts に `honored: false` を 1 行ずつ書いた(計画の A6 のとおり)。herdr の workspace は 1 つのまま |
| implementer の spawn `--dry-run` | rc 1。同じ文。dry-run の `rejected` と receipt を 1 行ずつ書く |
| 型付きの TASK(本番の send) | rc 1。`org: send: TASK not sent: org_id "a" has no approval recorded for plan docs/plans/active/2026-10-13-a.md sha256:8e7b0c143bb6 since its last disband; refused by the approval check:` のあとに spawn と同じ「記録がない」場合の次の手が続き、`the leader never runs ralph org approve itself and never sends the TASK with --raw` で終わる。台帳の hash は変わらない。stub に打たれた行は 0 |
| `--raw` の TASK(本番の send) | rc 1。同じ文。台帳は変わらず、打たれた行は 0 |
| 型付きの TASK `--dry-run` | rc 1。同じ文。台帳は変わらない |

断られた本番の spawn は、`rejected` の状態の座席 `implementer`(role implementer)を台帳に残す。TASK の検査はその座席を引いて当たったので、座席を手で台帳に書く必要はなかった。

### (b) approve のあと

| 確認 | 結果 |
|---|---|
| leader の QUESTION | `escalate --org-id a` に `PLAN: docs/plans/active/2026-10-13-a.md sha256:8e7b0c143bb6` と `VISUAL: .harness/state/plan-visual/2026-10-13-a.html` のヘッダを付けた。rc 0、`escalated e1 (org=a type=QUESTION)`。通知は osascript がないので `failed: …` と記録された |
| `ralph org approve e1`(`HERDR_PANE_ID` なし) | rc 0。`approved e1 (org=a plan=docs/plans/active/2026-10-13-a.md sha256:8e7b0c143bb6)`、leader へ DECISION を送る `ralph org send` のコマンド、approve がその時点のファイルを記録したという注を出した |
| manifest | `plan_approved` が 1 行。Details は `plan=docs/plans/active/2026-10-13-a.md digest=8e7b0c143bb6 content=f657692403640e58dadee89a3ae090b0d2c9e7dfb1de44c3cc9b3ecd2ef8ec0c inbox=e1 by=tester herdr_pane=unset cwd="<L>/repo"` |
| 受信箱 | e1 に `resolved`(note は `approved docs/plans/active/2026-10-13-a.md sha256:8e7b0c143bb6`)。`inbox show e1` は `state: resolved` |
| 計画のファイル | approve の前と後で sha256 は同じ(先頭 16 桁 4c04e4e94506d266)。approve はファイルを書かない |
| 2 行を書く前の spawn `--dry-run` | rc 1。`… is not marked approved: its - Status: is "Draft", want Approved; refused by the approval check: the leader writes - Status: Approved and - Approved: <YYYY-MM-DD> sha256:8e7b0c143bb6 before the plan's first ## heading, commits them, and tries again; …` |
| 2 行を書いてコミット(24ad0a1) | digest は 8e7b0c143bb6 のまま |
| implementer の spawn(本番) | rc 0。`spawned seat "implementer" (org_id=a … pane_id=w2:p3 dry_run=false)`。status は leader と implementer の 2 席とも `spawned (active)` |
| `PLAN:` の正しい TASK の `--dry-run`(型付き、`--raw`) | どちらも rc 0 で `sent message to seat "implementer"`。dry-run の `sent` が 1 行ずつ増え、`--raw` のほうは `raw=true` |
| 同じ TASK の本番の send | rc 0。stub の pane に `TYPE: TASK` から `implement the plan` までの 5 行が打たれた。stub の表示は working に変わらないので、ralph は `could not confirm that seat "implementer" started working after Enter` と警告し、`sent` に `submit_unconfirmed=true` を記録した。警告は stub を使ったために出たもので、承認の検査とは関係ない |

### (c) 計画が変わったあと、承認し直したあと

| 確認 | 結果 |
|---|---|
| 計画の本文を変えてコミット(7b7c30f) | digest は 3a36fa20e7a1 に変わった |
| 古い digest の TASK `--dry-run` | rc 1。`… changed after its approval: approved digest 8e7b0c143bb6, current digest 3a36fa20e7a1; refused by the approval check: the leader undoes the change, or sets the plan back to - Status: Draft and - Approved: TBD and asks for a new approval with a new QUESTION; …`。台帳は変わらない |
| 別の implementer の spawn `--dry-run` | rc 1。理由と次の手は TASK と同じ |
| 次の手に従う | 2 行を Draft と TBD に戻してコミットした(865f98d)。digest は 3a36fa20e7a1。新しい QUESTION を escalate し、e2 になった |
| e2 の approve(`HERDR_PANE_ID=w99:p9`、どの座席の pane でもない) | rc 0。`plan_approved` は `digest=3a36fa20e7a1 … inbox=e2 by=tester herdr_pane=set` |
| 古い digest の TASK(2 行はまだ Draft と TBD) | rc 1。`… is not marked approved: its - Status: is "Draft", want Approved; refused by the approval check: the leader writes - Status: Approved and - Approved: <YYYY-MM-DD> sha256:3a36fa20e7a1 …`。先に出るのは PlanHeader の次の手で、新しい digest を名指す |
| 2 行を 3a36fa20e7a1 で書いてコミット(9998354)したあと、古い digest の TASK | rc 1。`the TASK's PLAN: header names sha256:8e7b0c143bb6 of plan … and the plan's current approved digest is sha256:3a36fa20e7a1; refused by the approval check: the leader changes the TASK's PLAN: header to PLAN: docs/plans/active/2026-10-13-a.md sha256:3a36fa20e7a1 and sends the TASK again; …`。TaskHeader の次の手で、新しい digest を名指す |
| 新しい digest の TASK `--dry-run`(型付き、`--raw`) | どちらも rc 0 |
| 別の implementer の spawn `--dry-run` | rc 0。`spawned seat "implementer-2" (… dry_run=true)` |

### (d) 座席の pane からの approve

e2 を承認する前に打った。台帳にある leader の pane は w2:p2、implementer の pane は w2:p3。

| 確認 | 結果 |
|---|---|
| `HERDR_PANE_ID=w2:p2 ralph org approve e2` | rc 1。`org: approve: refused, since HERDR_PANE_ID "w2:p2" is the pane of the active seat "leader" of org_id "a": ralph org approve records a human's approval of a plan, so a human runs it from a terminal outside the seats' panes (…)` |
| `HERDR_PANE_ID=w2:p3 ralph org approve e2` | rc 1。同じ文で、座席の名前が `"implementer"` になる |
| 台帳 | 2 回の前後とも、manifest 22 行、受信箱 5 行、receipts 6 行、hash 3be29cf93dd262f0。e2 の event は `escalated` と `notified` だけで、resolve されていない |

### AC14 の確認(digest が読まない場所の変更)

| 確認 | 結果 |
|---|---|
| checkbox に印を付けてコミット | digest は 3a36fa20e7a1 のまま。TASK `--dry-run` は rc 0 |
| `## Progress checklist` に 1 行足してコミット | digest は 3a36fa20e7a1 のまま。TASK `--dry-run` は rc 1 で、理由は `… changed after its approval outside its - Status: and - Approved: lines and checkbox marks, which its digest 3a36fa20e7a1 does not show …`。spawn `--dry-run` も rc 1 |
| その行を消してコミット | TASK `--dry-run` は rc 0 |

### (e) split の org、leader のいない機能の org

| 確認 | 結果 |
|---|---|
| `start --plan … --feature b --config <L>/split.toml`(中身は `[org.approval] mode = "split"` だけ) | rc 0。leader の pane は w3:p2。`scope_reserved` の Details は `… feature=b digest=b624ce5d96c2 branch=feat/b approval=split`。leader のタスクには `- 承認方式: split` の行がある |
| b の implementer の spawn(本番、承認なし) | rc 0。`spawned seat "implementer" (org_id=b … pane_id=w3:p3 dry_run=false)`。台帳の `plan_approved` は org a の 2 行だけ |
| `PLAN:` のない TASK(`--dry-run` の型付きと `--raw`、本番の send) | 3 つとも rc 0。本番の send には (b) と同じ「確かめられない」の警告が出た |
| `ralph org stop --org-id b --seat leader` のあと、reviewer の spawn(本番) | rc 1。`rejected: org: org_id "b" was started with ralph org start --plan and its leader seat is not running, so seat_id "reviewer" is not spawned: start the org again with ralph org start --plan, which spawns the leader`。manifest に `rejected` を書いた |
| 同じ spawn の `--dry-run` | rc 1。同じ文 |

per-plan の org a では、reviewer の spawn(本番、pane w2:p4)と、`PLAN:` のない reviewer への TASK `--dry-run` が rc 0 で通った。同じ org で `PLAN:` のない implementer への TASK は rc 1 で、次の手は `the leader puts one PLAN: <path> sha256:<digest> header naming the approved plan on the line right after TASK_ID: and sends the TASK again` だった。このときは承認の記録がもうあったので、承認の前の reviewer への TASK は CLI のテストで確かめた範囲に留まる。

### 片づけ

| 確認 | 結果 |
|---|---|
| `ralph org disband --org-id a` と `--org-id b` | どちらも rc 0。a は 3 席、b は 1 席を stopped にした(b の leader は (e) で止めてあった)。herdr の workspace は 0 個 |
| `ralph org watch --stop --state-dir <台帳>` | rc 0、`stopped ralph org watch --all (pid 48215)`。そのあとの `ralph status` は `org watch: not running (stopped at 2026-10-12T21:44:06Z)` |
| `herdr server stop` | rc 0 |
| 残ったプロセス | cwd `/` から打った `pgrep -fl "<scratch>"` は何も返さず rc 1。`pgrep -fl "herdr server"` も rc 1。stub の 5 つ(pid 53396、56826、62210、62352、62743)と herdr サーバー(31138)を `ps -p` で引くと rc 1。全プロセスの cwd を `lsof` で見て、scratch の下にあるものは 0 個 |

1 回目の `pgrep -fl "<scratch>"` は、確認に使った `sed` と `tee` 自身を拾って rc 0 になった。パイプなしで打ち直し、rc 1 を確かめた。全出力には両方を残した。(b) の途中の `ralph org inbox list` は、存在しない動詞を私が打ったもので、rc 1 で何もしていない。

## 結論

- per-plan の機能の org では、承認がないうちの implementer の spawn(本番、dry-run)と TASK(型付き、`--raw`、dry-run)がどれも rc 1 で断られた。エラーには `refused by the approval check` と、記録がない場合の次の手が付いた。TASK の拒否は台帳に何も書かず、pane にも何も打たなかった
- `ralph org approve e1` は `plan_approved` を書いて e1 を resolve し、計画のファイルは変えなかった。leader の役で 2 行を書いてコミットしたあと、implementer の spawn と、正しい `PLAN:` の TASK が通った
- 計画を変えると、TASK は「承認のあとに変わった」の次の手で断られた。承認し直したあと、古い digest の TASK は 2 段で断られた。2 行が古いうちは PlanHeader の次の手、2 行を書いたあとは TaskHeader の次の手で、どちらも新しい digest を名指した。新しい digest の TASK は通った
- active な座席の pane の id を `HERDR_PANE_ID` に入れた approve は rc 1 で断られ、台帳の 3 つのファイルは変わらなかった
- split で立てた org は、承認なしで implementer を spawn し、TASK を送れた。leader を止めた機能の org では、leader 以外の spawn が断られた
- 確かめていないもの: 本物のモデルの leader が雛形の手順(図解ページ、QUESTION、DECISION を待つこと、2 行のコミット)を自分で進めること。本物の agmsg での DECISION の受け渡し。Linux。別の herdr サーバーの pane と id が重なる場合
- 副作用: なし。書いたのは scratch の下だけで、利用者の herdr と agmsg の home のファイルは変わっていない
