# org の座席を disband せずに立て直す: `ralph org reseat` と立て直しの経過

## Status

2026-11-12 に書いた。issue #312 の 3 本目(#312c)にあたる。人は「推奨案を自動承認する。承認は不要」と伝えており、/spec で人に聞く予定だった分かれ目は推奨案を自律で選んだ(下の「Decisions」)。選ぶ前に consult(別のモデルの助言)を通し、その指摘で D5 を変え、D3 と D2 に条件を足した。

2026-11-13、1 本目(D1〜D6、FR-1〜FR-8、FR-5a、FR-11、非機能要件、AC-1〜AC-12)を実装し、印を付けた(AC-12 の実機の限りは AC-12 の行にある)。2 本目(D7、FR-9、FR-10、AC-13、AC-14)は未着手で、印は `[ ]` のまま。Open questions の 2 つは実機の記録で答えが出た。

## Summary

`start --plan` で立てた機能の org の座席を、disband せずに止めて、今の環境で立て直す動詞 `ralph org reseat` を足す。計画の承認(`plan_approved`)、予約、機能との結びつきは、org の今の命に残るので引き継がれる。あわせて、立て直した leader のタスクに、台帳から作る経過(承認した計画、最後の gate、片付いていない受信箱の件、branch の commit)を足し、leader が済んだ手順を飛ばして再開できるようにする。

## Background and problem

2026-11-11 の 0 時台に、2 つの機能の org の座席 6 席(claude)が、アカウントの週の利用上限に達して止まった。解除は 3 日後だった。別のアカウントへ移すため、人は次の手順を手で踏んだ(issue #312)。

1. 6 席を `ralph org stop` する
2. herdr のサーバーを止める
3. `CLAUDE_CONFIG_DIR` の無い環境で `ralph org start --plan` を流し直す

片方の org は、分割計画のほかの機能の節を直していたため start に断られ、disband して start し直した。disband で計画の承認の記録が前の命に移って効かなくなり、leader に承認を求め直させた(この断りは #312b、PR #320 で直した)。新しい leader は前の会話を持たず、director が、済んだスライス、レビューの結果、予約の外の許可、gate の状態を手で DECISION に書いて伝えた。

### Current state

- 座席を立て直す動詞は無い。leader は `ralph org stop` のあと `ralph org start --plan` で、非 leader の座席は leader が `ralph org spawn` で立て直す(`internal/org/prompts/leader.md` の手順 5、reviewer の作り直し)。
- `Stop` は座席レベルの `stopped` だけを書き、org の命を終えない。今の命の `plan_approved`(`internal/org/approval.go` の `currentPlanApprovals`)と予約(`internal/org/reserve.go` の `currentOrgLives`)は、disband までそのまま効く。pane が無い座席は「already closed」として止める(`internal/org/verbs.go` の `stopSeatPane`)。
- `start --plan` の立て直しは、#312b のあと、前文とその機能の節と承認方式が同じなら通り、台帳に何も足さない。
- 立て直した leader のタスクは分割計画からだけ作られ(`internal/org/feature.go` の `featureLeaderTask`)、経過を持たない。`leader.md` には再開の分岐が無く、手順 1(計画を書く)から始まる。
- pane の環境は herdr server の環境に、ralph の `--env`(`AGMSG_SELF_NAME=off`、`RALPH_WIP_CHECKPOINT=off`、`RALPH_PRIMARY_CLI`、claude の再試行の設定)を足したもの。`CLAUDE_CONFIG_DIR` と `CODEX_HOME` を ralph は pane に渡さない。ralph が起動する herdr server は、起動した shell の `CLAUDE_CONFIG_DIR` を継ぐ(`internal/cli/org_herdr_server.go` の `orgHerdrServerEnv`)。
- herdr は 1 タブの最後の pane を閉じると workspace ごと消す。台帳の `org_workspace_created` は開いたまま残り、次の spawn の `resolveWorkspace` が「not found at spawn」で閉じて新しく作る(`internal/org/spawn.go`)。

### Desired state

director か人が `ralph org reseat --org-id <id>` を打つと、その org の座席が止まり、今の環境(herdr server の環境と今の設定)で同じ役割の座席として立ち直る。承認と予約は残り、立て直した leader は経過を読んで、承認済みの計画を書き直さずに続きから進める。別のアカウントへ移すときは、herdr server を新しい環境で立て直してから reseat を打つ手順が、help と skill に書いてある。

## Decisions

すべて自律で選んだ(人の方針「推奨案を自動承認」に従う)。理由を各項に書く。

- D1 動詞の形は `ralph org reseat --org-id <id> [--seat <id>]... [--driver <d> --model <m>] [--dry-run]`(`--state-dir` と `--config` はほかの動詞と同じ)。`--driver` と `--model` は `--seat` がちょうど 1 つのときだけ受け付ける。org 全体を立て直すときは、各座席の台帳の driver と model をそのまま使う。理由: 元の spawn がフラグで決めたか役割の既定で決めたかは台帳から分からないので、記録を正とし、設定の変更で座席の driver が黙って変わらないようにする。別の driver で立てたいときは座席ごとに指定する。
- D2 既定の対象は、その org の今の命で active な座席すべて。`--seat` は、今の命の座席(active か、最後の state event が `stopped`)を名指しできる。active な座席が 1 つも無く `--seat` も無いときは断り、今の命で `stopped` の座席を並べ、`--seat` を付けた打ち方を出す。理由: leader が意図して止めた座席(reviewer の作り直しの途中など)を勝手に蘇らせない。issue の場面のように人が先に全座席を止めたときも、一覧があれば打ち方に迷わない(consult の指摘)。
  実装中に足した(self-review の指摘): 途中で切れた reseat が止めた座席(最新の state が `stopped` で `reseat=<id>` を持ち、その id の `reseated` が無い座席)も既定の対象に入れる。同じ reseat を同じ `--seat` のまま打ち直せば立て直しが終わる。CLI は本当の reseat のあいだ SIGINT・SIGTERM・SIGHUP と SIGPIPE を受け止めて最後まで続ける。役割の prompt の雛形が無い独自の役割の座席は、台帳から指示を作り直せないので断る。
- D3 順序は「検査 → 停止 → 起動」。何も止めないうちに、start と spawn が後で断る条件をすべて確かめる(FR-4)。通ったら、非 leader、leader の順に止め、leader、ほかの座席の元の spawn の順に立てる。理由: 止めたあとに start が断ると座席が 0 で残る(consult の指摘)。leader を最初に立てると、全座席を止めたときに leader が新しい workspace の最初の pane(左半分)になり、元の配置に戻る。非 leader の spawn は leader が active でないと断られるので、leader が先に要る。
- D4 leader は `start --plan` と同じ道で立てる。台帳の結びつきの split と feature と org_id で、`<state dir>/splits/<split>.md` から `StartFeature` を呼ぶ。対象は `start --plan` で立てた org(今の命に完全な結びつきがある org)だけで、ほかの org は断る。理由: worktree の検査、予約と結びつきの判定、leader のタスクを start と一つの道で保てる。結びつきの無い org(`start <task>` や昇格した leader)は leader のタスクを台帳から作り直せない(長い prompt のファイルは同じ id の spawn で上書きされる)。
- D5 非 leader の座席だけを立て直したとき(leader は動き続ける)、ralph は leader に何も送らない。出力の最後に、leader へ送る DECISION の `ralph org send` の行を出す(立て直した座席は TASK を持たずに立つので、今の TASK を送り直すよう頼む文)。理由: `send` は pane のロックを取り、座席の手が空くのを待って打つ道で、送り手の名乗りと待ちの責任を reseat の中に抱えさせない。`ralph org merge` と `cleanup` も、次に打つコマンドを出して director に任せる(consult の指摘で、自動で送る案から変えた)。leader も立て直したときは行を出さない(経過の節が伝える、D7)。
- D6 アカウントの切り替えは案内にとどめる。reseat は pane ごとに `CLAUDE_CONFIG_DIR` を渡さない。理由は 3 つ。(1) `--env` は値を足すだけで、server の値を消せない(issue の場面は `CLAUDE_CONFIG_DIR` の無い環境への切り替え)。(2) 明示した `CLAUDE_CONFIG_DIR` と既定の場所で、macOS の keychain の扱いが同じかは repo から確かめられない。(3) あとで leader が立てる座席(reviewer の作り直しなど)には渡らず、前のアカウントで立つ。org ごとの環境を台帳に持たせる案は別の issue にする。
- D7 経過の節は、leader の立て直しのたびに入れる。reseat だけでなく、`start --plan` の立て直しでも、今の命に経過があれば同じ節を入れる。経過が空なら入れない。理由: 立て直しの道が 2 つあっても、leader が受け取るものを同じにする。
- D8 PR は 2 本に分ける。1 本目は動詞(D1〜D6、FR-1〜FR-8、FR-11)で、issue #312 の受け入れ条件 2 を満たす。2 本目は経過の節と `leader.md` の規則(D7、FR-9、FR-10)。理由: 1 本目だけで、承認と予約を残したまま座席を立て直せる。経過は leader の雛形を変えるので、別に見てもらう。

## Requirements

### Functional requirements

- [x] FR-1 動詞: `ralph org reseat --org-id <id> [--seat <id>]... [--driver <driver> --model <model>] [--dry-run] [--state-dir <dir>] [--config <path>]` を足す。`--driver`・`--model` を `--seat` が 1 つでないときに渡すと、何もせず終了コード 1 で断る(D1)。
- [x] FR-2 対象: 今の命に完全な結びつきのある org だけを受け付ける。ほかの org は断り、`ralph org stop` と `ralph org spawn`(leader は `ralph org start`)の打ち方を言う。既定の対象と `--seat` の扱いは D2 のとおり。`--seat` は、今の命で一度 `spawned` になった座席なら、最新の state が active、`stopped`、`spawn_failed`、`rejected` のどれでも受け付ける(起動に失敗した座席を打ち直せるようにする。計画の助言の指摘で足した)。今の命で一度も立っていない座席を名指しすると断る。
- [x] FR-3 打つ場所: `HERDR_PANE_ID` がその org の座席の pane なら、何も読まずに断る(自分の pane を閉じるため)。director か人が、座席の外から打つ。
- [x] FR-4 先の検査: 何も止めないうちに、次をすべて確かめる。1 つでも通らなければ、何も止めず何も記録せずに終了コード 1 で、通らなかった理由を言う。
  - 分割計画 `<state dir>/splits/<split>.md` があり、形式どおりで承認済み(`CheckApproved`)で、その機能がある
  - 今の分割計画から作る結びつきが、台帳の結びつきと同じ機能である(`sameFeature`、#312b の機能ごとの digest と承認方式)
  - 機能の worktree の記録が有り、結びつきの worktree と branch に一致し、start の再利用の規則を通る(start より厳しい。reseat は新しい worktree を作らない)
  - 見張りの検査と herdr server の検査(start と spawn と同じ。server が動いていなければ起動する)と、regate のロックの検査
  - 対象を止めたあとの台帳で、上限(`max_orgs`、`max_total_seats`、`max_seats`)と予約の判定が通る
  - 立てる非 leader の座席について、leader が立つか動き続けること。per-plan の org の implementer なら、今の命の承認が効くこと
- [x] FR-5a 重なり: reseat は、検査の前から記録を書き終えるまで、台帳ごとのロック `reseat.lock` を待たずに取る。取れなければ、持っている reseat の org を言って断る。止める座席ごとに、止める直前に台帳を読み直し、その座席の最新の state event が検査のときと同じであることを確かめる。違えば、その座席を止めずにそこで止め、止めた座席と残った座席と打ち直し方を出して終了コード 1(計画の助言の指摘で足した)。
- [x] FR-5 停止: 対象を、非 leader、leader の順に `Stop` と同じ処理で止める。pane が無い座席は「already closed」の `stopped` で進む。どれかが `stop_failed` になったら、そこで止め、どの座席も立てず、止めた座席と残った座席と打ち直し方を出して終了コード 1。
- [x] FR-6 起動: leader が対象なら最初に D4 の道で立てる。driver と model は、`--seat leader` に `--driver`・`--model` があればそれ、無ければ台帳の leader の記録。非 leader の座席は、台帳の role・driver・model(または `--driver`・`--model`)で、cwd は結びつきの worktree、`--scope` は予約のパスを `", "` でつないだものとし、元の spawn の順(`spawn_started` の順)で立てる。起動の途中で失敗したら、立った座席と立たなかった座席、打ち直し方(`ralph org reseat --org-id <id> --seat <立たなかった座席>`)を出して終了コード 1。
- [x] FR-7 出力: 止めた座席と立てた座席を 1 行ずつ出す。非 leader だけを立て直したときは、最後に leader へ送る DECISION の `ralph org send` の行を出す(D5)。`--state-dir` を渡したときは、出す行にも付ける。
- [x] FR-8 記録: 1 回の reseat に 1 つの id を振り、止めた座席の `stopped` と立てた座席の `spawned`(と `spawn_started`)の Details に `reseat=<id>` を足す。最後に、org レベルの event `reseated`(id、座席の一覧、driver と model を変えた座席)を 1 件書く。`--dry-run` は FR-4 の検査までを行い、止める座席、立てる順、各座席の driver と model を出し、何も書かない。見張りと herdr server は動いているかを読むだけで起動しない(動いていなければ、本当の reseat が起動すると出す。計画の助言の指摘で足した)。
- [ ] FR-9(2 本目)経過の節: leader のタスクに「立て直しの経過」の節を足す。中身は今の命の、最新の `plan_approved`(計画のパスと digest)、最後の gate の記録(結果、head、base、branch、via)、受信箱の open と acked の件(ID、TYPE、TASK_ID、本文の 1 行目)、branch の commit(origin の既定のブランチから HEAD まで、1 行ずつ最大 20 件と総数)、直近の `reseated`(座席と時刻)。`start --plan` の立て直しでも、今の命に経過があれば入れる。何も無ければ節を入れない(D7)。
- [ ] FR-10(2 本目)`leader.md`: タスクに経過の節があれば、済んだ手順を飛ばして再開する規則を足す。計画が承認済みなら書き直さない。gate の記録が今の HEAD と同じで pass なら、PR の手順に進む。片付いていない受信箱の件は答えを待つ。座席が立て直されていれば、その座席の今の TASK を送り直す。
- [x] FR-11 案内: `ralph org reseat --help`、org skill(4 つの写し)、`docs/recipes/isolated-herdr-server.md` か org skill の該当節に、別のアカウントへ移す手順を書く。herdr server を止める(全 org の pane と、herdr の中の headless の director も消える)、新しい環境の shell で `ralph org reseat` を打つ(server の検査が、その shell の環境で server を起動する)。director skill(4 つの写し)には、利用上限で止まった座席を人が別のアカウントで続けたいと言ったときの打ち方を足す。

### Non-functional requirements

- [x] 先の検査が通らないときは、どの座席も止めない(FR-4)。止めたあとに失敗したときは、座席の状態と打ち直し方を必ず出す(FR-5、FR-6)。
- [x] pane の環境に秘密を入れない(今のまま)。reseat は新しい環境変数を pane に渡さない。
- [x] 古い binary との互換: `reseated` は座席の状態に関わらない org レベルの event で、台帳の読み手は知らない event を読み飛ばす(`internal/org/manifest.go` の state event の扱い)。Details の `reseat=` は note として残るだけで、古い binary の判定を変えない。
- [x] 実機の確かめは `docs/recipes/isolated-herdr-server.md` の分離した herdr でだけ行い、ユーザーやほかのセッションの herdr に触れない。

## Acceptance criteria

1 本目:

- [x] AC-1 Given 機能の org の leader・implementer・reviewer が active で、per-plan の計画の承認が今の命にある。When `ralph org reseat --org-id <id>` を打つ。Then 3 席が止まって leader、implementer、reviewer の順に立ち、`scope_reserved` は増えず、implementer の spawn と implementer への TASK は承認の検査を通る(issue #312 の受け入れ条件 2)。
- [x] AC-2 全座席を立て直したとき、leader が最初に立ち、新しい workspace の最初の pane になる。
- [x] AC-3 `--seat implementer --driver codex --model <m>` では implementer だけが codex で立ち、leader は動き続ける。出力の最後に leader へ送る DECISION の `ralph org send` の行が出る。
- [x] AC-4 分割計画が承認のあと変わった、自分の節か前文を直した、分割計画のファイルが無い、見張りが動いていない、機能の worktree の記録が無いか結びつきと合わない、のどれかのとき、reseat はどの座席も止めず、何も記録せずに終了コード 1 で理由を言う。
- [x] AC-5 結びつきの無い org の reseat は断られ、stop と spawn の打ち方が出る。
- [x] AC-6 active な座席が無く `--seat` も無いとき、reseat は断り、今の命の `stopped` の座席と `--seat` を付けた打ち方を出す。`--seat` を付けて打ち直すと立つ。
- [x] AC-7 その org の座席の pane から打った reseat は断られる。
- [x] AC-8 herdr server を立て直して座席の pane が無いときも、reseat で座席が立ち直る。
- [x] AC-9 `--dry-run` は何も書かず、止める座席、立てる順、各座席の driver と model を出す。
- [x] AC-10 `Stop` が `stop_failed` になった座席があれば、reseat は座席を立てず、止めた座席と残った座席と打ち直し方を出して終了コード 1。
- [x] AC-11 `--driver` だけ、または `--seat` を 2 つ付けて `--model` を渡すと断られる。
- [x] AC-12 分離した herdr の実機で、AC-1 と AC-8 を 1 回ずつ確かめる。実機の記録は `docs/evidence/org-reseat-live-2026-11-12.md`。AC-8 は合格した。AC-1 は leader と reviewer の 2 席で合格した。implementer の座席と承認の検査は実機で見ておらず(テストだけで確かめた)、binary は切れた reseat の座席と SIGPIPE の直しより前のものである。

2 本目:

- [ ] AC-13 承認、gate の記録、片付いていない受信箱の件、branch の commit がある org を reseat すると、leader のタスクに経過の節が入り、それぞれが載る。経過の無い org では節が入らない。`start --plan` の立て直しでも同じ節が入る。
- [ ] AC-14 `leader.md` に、経過の節があれば済んだ手順を飛ばす規則があり、役割の prompt のテストが文を固定する。

## User stories

1. As 人, I want to director に「上限で止まった座席を別のアカウントで続けて」と頼むだけで済ませたい, so that 座席を 1 つずつ止めて start し直す手順を覚えなくてよい。
2. As director, I want to 1 つの動詞で org の座席を立て直したい, so that 計画の承認と予約を失わずに、座席の環境だけを入れ替えられる。
3. As 立て直した leader, I want to 台帳の経過をタスクで受け取りたい, so that 承認済みの計画を書き直したり、済んだ gate を流し直したりしない。

## Constraints

### In scope

- `ralph org reseat`(機能の org だけ)、先の検査、停止と起動の順、出力、記録、`--dry-run`
- 経過の節と `leader.md` の規則(2 本目)
- help、org skill と director skill の 4 つの写し、案内の文

### Out of scope

- pane ごとに `CLAUDE_CONFIG_DIR` などの環境を渡すこと、org ごとの環境を台帳に持たせること(D6。別の issue)
- 結びつきの無い org の立て直し(D4)
- 見張りが `usage_limited` の件を見て自動で reseat すること(#312a は知らせるだけ)
- 画面の判定(`internal/org/resume_screens.go`)の変更(レーン A の #312a)
- 分割計画の再承認の規則(#312b のまま)

## Impact

| Target | Impact | Severity |
|--------|--------|----------|
| `internal/org/`(新しい reseat のファイル、`verbs.go` の Stop の再利用、`feature.go` の StartFeature の再利用、`spawn.go` の Spawn、`seat.go` の event) | 新しい動詞と event | High |
| `internal/cli/org.go` と新しい CLI のファイル | 新しいサブコマンドと help | Medium |
| `internal/org/prompts/leader.md`、`feature.go` の `featureLeaderTask`(2 本目) | leader のタスクと雛形 | Medium |
| org skill と director skill の 4 つの写し、`docs/recipes/isolated-herdr-server.md`、`AGENTS.md` の `internal/org/` の行 | 文書 | Low |

## Dependencies

- #312b(PR #320): `start --plan` の立て直しが、ほかの機能の節の変更で断らないこと。reseat の leader は同じ道を使う
- #312a(レーン A): `usage_limited` の知らせ。reseat はそれに依存しないが、director skill の案内で打ち方をつなぐ

## Research findings

### Codebase analysis

- `Stop`(`internal/org/verbs.go`): 座席の pane を確かめ、C-c と close、agmsg の Leave、`stopped`。pane が無ければ「already closed」で進む。close に失敗すると `stop_failed` で座席は active のまま。呼び出し元の pane は閉じずに最後に閉じる。
- `Spawn`(`internal/org/spawn.go`): `stopped` の座席は新しい spawn の道を通る。他の座席の pane が残れば同じ workspace に分割、残っていなければ新しい workspace を作り、最初の座席が root の pane になる。leader の spawn は agmsg に Join し直す。
- `StartFeature`(`internal/org/feature.go`): worktree を再利用し、leader を spawn する。worktree の記録が一致すれば、main のチェックアウトがきれいである必要はない(`scripts/ralph-worktree.sh` の ensure)。
- 台帳から読み戻せるもの: `spawn_started` と `spawned` の role・driver・model・cwd、`spawned` の Details の scope(自由文で読み戻しは曖昧なので、予約から作る)。読み戻せないもの: `--prompt`、`--timeout-ms`、TASK の本文(`sent` は先頭 200 バイトだけ)。
- 経過の読み手: `currentPlanApprovals`、`gateRecordFromDetails`、`InboxStore.Read`。org の今の命の最新の gate と、org で絞った受信箱を返す関数は無いので、2 本目で足す。

### Best practices

- 壊す操作の前に、あとで断られる条件をすべて確かめる(`ralph org merge` と `cleanup` が検査のあとに動くのと同じ形)。
- 動詞は次に打つコマンドを出し、送る判断は director に残す(`ralph org approve` が DECISION の send の行を出すのと同じ形)。

### Alternatives considered and trade-offs

| Option | Pros | Cons | Adopted |
|--------|------|------|---------|
| leader を `start --plan` と同じ道で立てる | 検査とタスクが start と一つ | 機能の org だけ | Yes(D4) |
| leader を台帳の prompt ファイルから立てる | 結びつきの無い org も扱える | ファイルは上書きされ、分割計画の変更を反映しない | No |
| 非 leader の立て直しを leader に ralph が送る | 人の手が 1 つ減る | 送り手の名乗りと待ちを動詞が抱える | No(D5) |
| pane ごとに `CLAUDE_CONFIG_DIR` を渡す | server を止めずに済む | 値を消せない、keychain が未確認、後の座席に渡らない | No(D6) |
| 全座席を止めてから検査する | 実装が単純 | 検査の失敗で座席が 0 で残る | No(D3) |

## Security considerations

- reseat は既存の `Stop` と `Spawn` だけで pane を操作し、所有の確認(pane と workspace のラベル)をそのまま通す。他人の pane は閉じない。
- 座席の pane から打った reseat は断る(座席が自分の org を立て直して止めることを防ぐ。境界ではなく、うっかりを止める仕掛け)。
- pane の環境に新しい値を渡さない。秘密が ps に出る道を増やさない。

## Open questions

- 全座席が agmsg の team から Leave したとき、team が消えるか(履歴ごと)。消えるなら、leader が最初に Join し直す今の順で足りるかを実機で確かめる(AC-12)。**答え(2026-11-12、`docs/evidence/org-reseat-live-2026-11-12.md`)**: 消えない。全座席が Leave しても、team の `team_id` は最初の作成から disband のあとまで同じで、名簿と 5 件のメッセージの履歴も残った。leader が最初に Join し直す今の順で足りる。
- herdr server を立て直したあと、復元された pane(agent の無いシェル)に新しい server の環境が入るか。reseat はそれらの pane を閉じて新しく作るので、結果には影響しない見込み。実機で確かめる。**答え(同じ記録)**: herdr 0.7.5 は、止める前に保存した `session.json` から workspace と pane の記録を戻したが、pane に端末もプロセスもなく(`herdr pane read` と `process-info` は `pane_not_found`)、新しい server の環境が入る pane はなかった。reseat の停止は Ctrl-C が `pane_not_found` で失敗したが、その pane を閉じて `stopped` を書き(`pane=closed`)、新しい workspace に立て直した。結果に影響しない見込みのとおりだった。

## References

- issue #312(https://github.com/yoshpy-dev/ralph/issues/312)
- PR #320(#312b、機能ごとの digest)
- `docs/specs/2026-11-07-org-seat-resume.md`(resume と画面の判定)
- `docs/specs/2026-11-06-org-herdr-autostart-panes.md`(herdr の自動起動と pane の配置、herdr 0.7.5 の実測)
- `docs/recipes/isolated-herdr-server.md`(分離した herdr)
