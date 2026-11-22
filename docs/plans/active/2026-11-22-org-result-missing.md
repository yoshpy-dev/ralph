# org-result-missing

- Status: Approved
- Approved: 2026-11-22 sha256:698af1d189e8
- Owner: Claude Code
- Date: 2026-11-22
- Related request: issue #361(座席の RESULT が leader に届かず、leader が待ち続けることがある)
- Related issue: 361
- Type: fix
- Branch: fix/org-result-missing

## Objective

機能の org の座席が TASK の RESULT(または BLOCKED)を leader に送れなかったとき、それが見えるようにする。座席は送信の失敗に気づいて送り直すか、失敗を画面に分かる形で残す。見張りは、TASK を受けた座席が入力待ちのまま結果を送っていないとき、その座席と TASK_ID を名指す ALERT を上げる。今は leader の停滞としか出ないので、leader が何を待っていて、何が届いていないかがわからない。

## Scope

- S0 調べ: codex の座席の `Hook failed` と送信の関係を、手元で再現を試す。2 段で行う。1 段目: `.codex/hooks.json` は Bash に PreToolUse だけを掛け(PostToolUse は Edit・Write・apply_patch だけ)、`ralph-dispatch.sh` は `.d` の非 0 をそのまま返すので、送信の直後の `Hook failed … code 1` は PreToolUse で送信のコマンドが実行の前に落とされた見込みがある。まず `ralph-dispatch.sh PreToolUse` に、座席が打つ形の送信のコマンドを載せた JSON を直に渡し、終了コードと出力を見る。ここで落ちれば 2 段目は要らない。2 段目(1 段目で落ちなかったときだけ): 座席と同じ sandbox(`codex sandbox` か `codex exec --sandbox workspace-write`、#362 の通信の `-c` と、機能の org の `--add-dir`)の中で、座席が打つ形の agmsg の送信(`~/.agents/skills/agmsg/scripts/send.sh <team> <from> leader "$(cat <ファイル>)"`)と、この repo の hook(`.codex/hooks.json` が呼ぶ `ralph-dispatch.sh PreToolUse` と `PostToolUse` に、その送信のコマンドを載せた入力)を走らせ、終了コードと出力を記録する(`docs/evidence/`)。本物の agmsg の DB や herdr のサーバーには触れず、一時の `HOME` か一時の agmsg の home で行う。原因が見つかり、その直しがこの repo の hook か雛形の小さな変更で済むなら、この計画の中で直す。済まなければ証拠と仮説を残し、tech-debt と issue に書く
- S1 座席の側(`internal/org/prompts/implementer.md`、`reviewer.md`): 今の雛形は送信のコマンドを名指さず agmsg の skill に任せているので、終了コードを確かめる対象として、送信のコマンドの形(agmsg の skill の `send.sh <team> <自分> leader <本文>`)を雛形に書く。RESULT、BLOCKED、QUESTION を送るとき、そのコマンドの終了コードを確かめる。0 でなければ、エラーの要点を読んで 1 回だけ送り直す。それでも 0 でなければ、送らずに終わらせず、画面に `SEND FAILED: <TYPE> <TASK_ID> <本文のファイルのパス>: <エラーの要点>` の 1 行を書いて止まる(leader が画面を読んで見分けられる)。終了コードが 0 のときは、agmsg の履歴(`history.sh <team> <自分の座席> <数>`)を読み、`<自分> → leader` の行のうち最後のものにその TASK_ID があることを確かめてから手番を終える(`history.sh` は座席を名指すと、その座席が受けた行も返すので、送り手と宛先で絞る)。座席は `ralph org send` を使わないので、agent-messaging.md の「(text typed but not submitted) のときは送り直さない」とはぶつからない。TASK を受けたのに RESULT も BLOCKED も QUESTION も送らずに手番を終えない、と書く
- S2 見張りの側(`internal/org/watch.go` と新しいファイル): 座席ごとの条件 `result_missing` を足す。leader 以外の座席について、次のとおり判定する。
  - 台帳の `sent`(dry run を除く)から、その座席への最後の TASK の TASK_ID と時刻を読む。TASK のあとに leader がその座席に STOP を送っていれば上げない
  - agmsg の履歴から、TASK の時刻以後(同じ秒を落とさない。台帳も agmsg も UTC の秒で比べられる)の `<seat> → leader` の行を読む(座席を名指した履歴には座席が受けた行も入るので、送り手と宛先で絞る)。その TASK_ID の RESULT か REVIEW があれば済み(上げない)。済みの判定は TASK の時刻から数え、あとで leader が座席に送る返事(RESULT への DECISION など)では動かさない(RESULT のあとの返事で誤って上げないため)
  - 座席の最後の行が QUESTION か BLOCKED(QUESTION は TASK_ID がなくても数える)なら leader の答え待ち。その行より後に leader がその座席に送った `sent` がなければ上げない。あれば、その最初の `sent` の時刻を数え始めの時刻にする。座席からの行がなければ、TASK の時刻を数え始めの時刻にする(codex の計画助言の指摘 1: QUESTION に答えたあとの送り損ねを見逃さないため)
  - 座席が入力待ち(herdr の状態が working でない)のまま、数え始めの時刻と入力待ちが始まった時刻(見張りが続けて working でないと見た最初の周の時刻、working を見たら消す)の遅いほうから `[org.watchdog].stall_minutes` を超えていれば、ALERT(`CONDITION: result_missing`、`SEAT`、`TASK_ID`)を leader に送る。本文は失敗と決めつけず、「TASK <id> を受けた座席 <seat> が入力待ちのまま、RESULT も BLOCKED も送っていない。座席の画面を読む。バックグラウンドのコマンドを待っているなら待つ」の形にする
  - 履歴の範囲: 座席を名指して(`history.sh <team> <seat> <limit>`)読み、読めた範囲が TASK の時刻まで遡れたかを確かめる。いちばん古い行が TASK の時刻より前か、取れた行の数が上限より少なければ(全部を読めた)遡れた、とする。遡れなければ上限を上げて読み直し(決まった回数と上限まで)、それでも遡れなければ「わからない」として上げない。TASK の時刻より後の座席の行が 1 つもなく、範囲が遡れているなら、それは送っていないことなので上げる。一度 RESULT か REVIEW を確かめた TASK は見張りの状態に覚え、あとで件数の上限で行が範囲から落ちても上げない(codex の計画助言の指摘 2)。履歴を読むのは、入力待ちと時間の条件がそろった座席だけにする(周ごとの呼び出しを増やさない)
  - 同じ座席の stall は、`result_missing` が立っている周には上げない(重ねない)
  - 履歴を読めなかったとき、台帳の `sent` の TASK_ID が切れていて読めないときは上げない(わからないものは上げない)。一致するメッセージが現れるか、その座席に新しい TASK が送られると消える。leader が動かないときの受信箱への escalate は、今の ALERT の決まり(deadman)のまま。同じ座席の停滞(stall)の ALERT と重ねて出さない(result_missing が立つ座席の stall は上げない、など、どちらか一方にする)
- leader の雛形(`internal/org/prompts/leader.md`): `result_missing` の ALERT を受けたら、座席の画面を `ralph org read` で読み、`SEND FAILED:` の行か、RESULT の文が画面にあるのに送られていないなら、その座席に結果を送り直すよう頼む(TASK か DECISION の形は今の決まりに合わせる)。座席が作業中なら待つ。わからなければ BLOCKED を上げる
- 文書: `.claude/rules/ralph/agent-messaging.md`(と雛形の写し)の ALERT の節に `result_missing`、org と director の skill(4 写しずつ)の ALERT の扱い、仕様 `docs/specs/2026-10-07-org-multi-org-director.md` か `docs/specs/2026-08-01-org-runtime.md` の見張りの項目、README の見張りの説明(あれば)

## Non-goals

- agmsg そのもの(`~/.agents/skills/agmsg/` の scripts、DB)の直し。この repo の外
- 座席の送信を ralph のコマンド(`ralph org send` のような)に置き換えること
- leader が座席に送る TASK の届き方(`ralph org send` がすでに確かめている)

## Assumptions

- 座席は agmsg の skill の `send.sh` で leader に送り、`history.sh <team> [<agent> [<limit>]]` は 1 件を 1 行(`<status> [<ts>] <from> → <to>: <本文>`、本文の改行は `\n`)で出す(`internal/org/watch.go` の `leaderHistoryFromField` の doc comment が読んだ形)
- 台帳の `sent` の Details から、座席に送った TASK の TASK_ID を読める(`internal/org/reseat_progress.go` の `reseatProgressSentTask` が読んでいる)
- 見張りは座席ごとに herdr の状態を読んでいる(`evaluateSeat`、`seatBusy`)

## Affected areas

- `internal/org/prompts/implementer.md`、`reviewer.md`、`leader.md`
- `internal/org/watch.go`、新しいファイル(例: `internal/org/watch_result_missing.go`)とテスト
- `.claude/rules/ralph/agent-messaging.md` と写し、org と director の skill(4 写しずつ)、仕様、README
- `docs/evidence/`(S0 の記録)

## Visual review

- ページ: `.harness/state/plan-visual/org-result-missing.html`。かんたんな図 1 枚(`overview`)と、詳しい図 2 枚(`detail-overview`、`detail-flow`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: ページ全体と 2 枚の詳しい図を `shot` で撮って読んだ。はみ出しや重なりはなかった
- 計画助言のあと: `detail-flow` に基準の時刻、QUESTION の答え待ち、遡れないときの扱いを足し、lint(exit 0)と `--fragment detail-flow` の撮り直しをした。最初の撮り直しで 1 つめの箱の文字がはみ出したので短くし、2 度目は clean
- S1 のあとの直し(S2 の判定の変更、下の Progress checklist の注): `detail-flow` の箱の文と注を、TASK の時刻から済みを数える形、QUESTION・BLOCKED の答え待ち、入力待ちが続く時間に合わせた。lint exit 0、`--fragment detail-flow` の撮り直しは clean

## Design decisions

Critical forks: None

- 見張りの待ち時間は新しい設定の鍵を足さず、`[org.watchdog].stall_minutes` を使う。座席が入力待ちで結果を送らない時間は、座席の停滞と同じ尺度で見てよい
- 履歴を読めなかったときは ALERT を上げない。誤った ALERT で leader を止めないため。読めないことが続くときの扱いは tech-debt に書く
- 座席の送信の確かめは、終了コードと履歴の両方で行う。終了コードが 0 でも、送るコマンドを打たずに手番を終えた場合(issue の claude の 1 件)は、履歴の確かめで気づける

## Acceptance criteria

- [x] AC1(S0): `ralph-dispatch.sh PreToolUse` に座席の形の送信を渡した結果(1 段目)と、1 段目で落ちなかったときは codex の座席と同じ sandbox で送信と hook を走らせた結果(2 段目)が `docs/evidence/` にある。原因が見つかったなら直しがこの計画に入っているか、tech-debt と issue に書いてある
- [x] AC2(S1): implementer と reviewer の雛形が、送信のコマンドの形、送信の終了コードの確かめ、1 回の送り直し、失敗したときの `SEND FAILED:` の 1 行、終了コードが 0 のときの履歴の確かめ、TASK に結果を送らずに手番を終えないことを言う(テストで固定)
- [x] AC3(S2): 見張りが、TASK を受けた座席が入力待ちのまま `stall_minutes` を超えて、その TASK_ID の RESULT・BLOCKED・QUESTION・REVIEW を送っていないとき、`CONDITION: result_missing` と `SEAT` と `TASK_ID` を持つ ALERT を leader に送る(fake の herdr と fake の agmsg の履歴のテストで固定)
- [x] AC4(S2): 一致するメッセージがあるとき、座席が作業中のとき、時間がまだのとき、履歴を読めないとき、leader の座席には上げない。一致するメッセージが現れるか新しい TASK が来ると消える。同じ座席に stall と重ねて出さない
- [x] AC6(S2): 何周期にもわたるテストで、次の流れの答えを固定する。(a) TASK → QUESTION → leader の DECISION → 座席が入力待ちのまま RESULT を送らない、で上げる。(b) TASK → QUESTION のまま leader が答えていない、で上げない。(c) TASK のあと座席からの行が 1 つもなく、範囲が遡れている、で上げる。(d) 正常な RESULT が件数の上限で範囲から落ちた(最新の N 件がすべて TASK の時刻より後)、で上限を上げて読み直し、それでも遡れなければ上げない。一度確かめた RESULT は、あとで範囲から落ちても上げない。(e) 座席あての行(`leader → <seat>` の TASK や DECISION)だけがあり、`<seat> → leader` の行がない、で上げる(受けた行を送った行と取り違えない)。(f) RESULT が TASK の時刻と同じ秒、で上げない。(g) TASK → RESULT → leader の DECISION(返事)のあと座席が入力待ち、で上げない。(h) 座席が working から入力待ちに変わった直後は、入力待ちが `stall_minutes` 続くまで上げない。(i) TASK のあとに STOP を送った座席には上げない
- [x] AC5: leader の雛形と skill と rule が `result_missing` の扱いを言う。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

## Implementation outline

1. S0 調べと記録(AC1) — verify: `test -s docs/evidence/org-result-missing-hook-probe-2026-11-22.md`
2. S1 座席の雛形(AC2) — verify: `test -z "$(gofmt -l internal/org)" && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt|Prompt'`
3. S2 見張りの条件(AC3、AC4) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'Watch|ResultMissing|Stall'`
4. S3 leader の雛形と文書(AC5) — verify: `./scripts/check-sync.sh && ./scripts/check-skill-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/ -run 'RenderRolePrompt_Leader|Skill|DocsMatch|AgentMessaging' && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/cli/ -run 'Agents|Readme|Docs|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC5。見張りの新しい条件が、今の ALERT の escalate、recurred、usage_limited の扱いと食い違わないこと
- Documentation drift to check: agent-messaging.md の ALERT の節、org と director の skill、仕様、README
- Evidence to capture: S0 の記録、見張りのテストの出力

## Test plan

- Unit tests: 履歴の行から TYPE と TASK_ID を読む関数、台帳の `sent` から座席の最後の TASK を読む関数
- Integration tests: 見張りの 1 周(fake の herdr の状態、fake の agmsg の履歴)で、上げる場合、上げない場合、消える場合
- Regression tests: 今の stall、liveness、scope_change、usage_limited の ALERT が変わらないこと
- Edge cases: 同じ TASK_ID の古い RESULT が TASK の前にある、別の座席が同じ TASK_ID を送った、本文が改行を含む、履歴の行が切れている、座席が作業中から入力待ちに変わった直後
- Evidence to capture: `go test` の出力、`./scripts/run-test.sh` の rc

## Risks and mitigations

- agmsg の履歴は件数の上限で古い行が落ちる: 座席を名指して読み、範囲が基準の時刻まで遡れたかで判定する(遡れなければ上限を上げて読み直し、それでもだめならわからないとして上げない)。確かめた完了は覚える(Scope の S2)。`history.sh <team> <seat>` は、その座席が送った行と受けた行の両方を、最新 N 件を時刻の昇順で返す(agmsg の `scripts/drivers/storage/sqlite.sh` の `to_agent=… OR from_agent=…`、jsonl.sh も同じ。consult で確かめた)。座席と見張りの両方で `<seat> → leader` に絞る
- claude の reviewer は gate と cross-review のコマンドを `run_in_background` で走らせて手番を終えるので、そのあいだ入力待ちに見え、長い gate では `result_missing` が上がる。今もこの間は stall が上がりうるので、知らせの数は増えない(stall の代わりに出る)。本文と leader の雛形は、画面を読んでバックグラウンドのコマンドを待っているなら待つ、と言う
- herdr が working のまま固まる座席: `result_missing` は上がらないが、画面が変わらなければ今の stall が拾う(`seatBusy` は画面の変化を要る)ので、ずっと見逃されることはない
- 誤った ALERT で leader が座席を止める: ALERT の本文は座席の画面を読むよう言い、止めるとは言わない。leader の雛形も、作業中なら待つと言う
- S0 で原因がわからない: 座席の側と見張りの側の 2 つの手当てで、届かなかったことは見えるようになる(受け入れ条件 3 の「届かないことが分かる」)

## Rollout or rollback notes

- 新しい ALERT の条件と雛形の文。動いている org は座席を立て直すまで前の雛形のまま(見張りは新しい binary で動く)
- revert すると、result_missing の ALERT と座席の確かめがなくなる

## Open questions

なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。指摘 2 件、どちらも計画に入れた。[HIGH] 1: 同じ TASK_ID の QUESTION があれば上げないと、leader の DECISION のあとに座席が RESULT を送り損ねても見逃す → 基準の時刻を leader が座席に最後に送った時刻にし、QUESTION は答え待ちとして止め、答えたら数え直す(Scope の S2、AC6 (a)(b))。[HIGH] 2: 履歴の範囲の判定が逆(TASK のあとに行がないのは送っていない典型。件数の上限で正常な RESULT が落ちることもある) → 範囲が基準の時刻まで遡れたかで判定し、遡れなければ上限を上げて読み直し、確かめた完了は覚える(Scope の S2、AC6 (c)(d)、Risks)。git の状態は助言の前後で同じ。consult(plan): 判定「直してから進める」。`history.sh` が座席の受けた行も返すので `<seat> → leader` に絞る(S1、S2、AC6 (e))、雛形に送信のコマンドの形を書く(S1、AC2)、S0 を 2 段にする(AC1)、同じ秒と切れた TASK_ID の扱い(S2、AC6 (f))を入れた
- [x] Plan approved — 系列の決まり(推奨案で自動承認、2026-10-11 のユーザーの指示)で承認
- [x] Branch created
- [x] Implementation started
  - S0 b42383cb(`docs/evidence/org-result-missing-hook-probe-2026-11-22.md`): hook は送信を止めていない。codex 0.166.0 は PreToolUse の hook の exit 1 を Failed と出すがコマンドは走らせる(止めるのは Blocked だけ)。座席の形の 18 コマンドを dispatcher に 66 回渡して exit 1 は 0 回(`TMPDIR` を壊したときだけ `ralph-dispatch.sh:108` の `mktemp` で 1)。2 段目(1 段目で座席の形の入力が 1 を出さず、hook だけでは届かなかったことを説明できないため実施)で、agmsg の DB のディレクトリに書けない sandbox では `send.sh` が stdout も stderr も空のまま exit 1 で何も保存せず、`history.sh` も exit 1 になった。見込みの原因はこれ(未確認、issue の環境で成り立っていたかはわからない)。hook は変えない。S1 への引き継ぎ: 送信の失敗は stderr が空のことがある(`SEND FAILED:` の要点が空の場合を扱う)、`history.sh` が失敗したときは送れたと数えない。計画の外の 2 点(dispatcher の `mktemp` の失敗ですべての hook が 1 になる、agmsg の `send.sh` の黙った失敗)は S3 で tech-debt に書く
  - S1 8f29daf7: implementer と reviewer の雛形に、送信のコマンドの形(`bash {{AGMSG_SCRIPTS}}/send.sh {{TEAM}} {{SEAT_ID}} leader "$(cat <ファイル>)"`)、終了コードの確かめ、1 回の送り直し、`SEND FAILED:` の行、`history.sh` の `{{SEAT_ID}} → leader:` の行での確かめ、TASK に答えずに手番を終えないこと(`run_in_background` の待ちは除く)を書いた。TASK について送る QUESTION にも TASK_ID を付ける。新しいテスト `prompts_seat_send_test.go`、雛形への 10 個の変異はすべて落ちた。座席にも AgmsgHome は渡っていた。`spawn.go:225`、`spawn.go:1111`、`internal/cli/org.go:434-435` のコメントが「leader だけが AGMSG_SCRIPTS を使う」のままなので S3 で直す
  - 計画の直し(S2 の前、2026-11-22): S2 の判定を 2 点直した。(1) 済みの判定を「leader が座席に最後に送った時刻」からではなく TASK の時刻から数える。RESULT のあとに leader が返事の DECISION を送ると、前の形では RESULT を範囲の外に置いて誤って上げるため。数え直すのは座席の最後の行が QUESTION か BLOCKED で、leader がそのあと座席に送ったときだけにした(AC6 (g))。(2) 入力待ちが `stall_minutes` 続いてから上げる(AC6 (h))、TASK のあとの STOP では上げない(AC6 (i))。claude の reviewer が gate を `run_in_background` で待つあいだの扱いを Risks に足した。digest が変わったので図を直し、系列の決まりで承認し直した
  - S2 f73a895a: `internal/org/watch_result_missing.go`(新規)、`watch.go` の `condResultMissing` と `watchSeatSnapshot` の `IdleSince`・`ResultSeen`、`reseat_progress.go` から `sentDetailsText` を切り出した。agmsg の履歴の時刻は `date -u +%Y-%m-%dT%H:%M:%SZ`(`drivers/storage/sqlite.sh:21`、`jsonl.sh:32`)で、RFC3339 で読む。座席の今の life は最後の `spawn_started` か `spawned` より後。QUESTION・BLOCKED のあとの leader の `sent` は秒で後のものだけ数える。テスト 14 本、変異 24 個のうち 23 個が落ちた(残る 1 個は結果の変わらない変異)。ALERT の本文は英語(ほかの ALERT に合わせた)。範囲の外の発見: 受信箱の ALERT の件の本文に TASK_ID が入らない、`semantic=true` なので watcher が有効なら on-demand watcher が走る(stall と同じ)。どちらも S3 で台帳に書く
  - S2b 43b7777e: S2 では、長く作業した座席が入力待ちになると stall が先に上がり、そのあと result_missing も上がった(計画の「stall の代わりに出る」と合わない)。leader 以外の座席の stall を、result_missing の時計(TASK の時刻と入力待ちの始まりの遅いほうから `stall_minutes`)が回るあいだ保留する。時計が切れたあと履歴が「済み」「答え待ち」「わからない」なら stall は今までどおり。その結果、RESULT を送ってから入力待ちになった座席の stall も最大 `stall_minutes` 遅れる(テストで固定)。テスト 4 本、変異 9 個のうち 8 個が落ちた(残る 1 個は等価)。S3 の文書に、保留と遅れの 2 点を書く
  - S3 832efa70: leader の雛形の運用規律の末尾に `result_missing` の扱い(座席の画面を読み、`SEND FAILED:` か送っていない結果があれば、その座席への DECISION で送り直しを頼む。TASK の送り直しは作業のやり直しになり per-plan の `PLAN:` も要るので採らない。DECISION の `sent` は時計を戻さない。同じ TASK でまた `SEND FAILED:` なら BLOCKED。バックグラウンドの待ちなら待つ)。agent-messaging.md(と写し)に「How a seat sends to the leader」の節と `result_missing`・stall の保留の段落、/org と /director の skill(4 写しずつ)、org-runtime の仕様の 2026-11-22 の改訂、README、AGENTS.md、古いコメント 3 か所、台帳の新しい行 4 つと既存 10 行の持ち越しの注、`prompts_leader_result_missing_test.go`(5 本、変異 8 個がすべて落ちた)。範囲の外の発見: `docs/recipes/isolated-herdr-server.md` の「Optional: a private agmsg」が S1 のあと古い(/sync-docs で直す)、held の座席でも stall が立つと利用上限の画面読みが毎周期走る、`result_missing` は座席ごとに履歴を最大 3 回読む(1 回 10 秒まで)
  - b0c56556 で main(#365 と #360、2a069b50)を取り込んだ。衝突は org skill、AGENTS.md、台帳、`spawn.go` のコメント。前の 3 つは scratchpad の merge3.py(main の行を先に)で両側の追記を残して解き、`spawn.go` は両側のコメントを 1 つにまとめた。check-sync、check-skill-sync、gofmt、vet、`internal/org` と `internal/cli` の narrow test は通った
- [x] Review artifact created — `docs/reports/self-review-2026-11-22-org-result-missing.md`(cycle 1)。CRITICAL 0、HIGH 0、MEDIUM 1、LOW 5、Merge: yes。M1: 前の TASK の `result_missing` の件が open か acked のあいだに次の TASK で立つと、空の note の `recurred` になり leader に届かない(scratchpad の調べのテストで確かめた)
  - Run 2(R1 の直し `f57cb92d`、cycle 1 のまま): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4、Merge: yes。M1 は直っている(別の TASK は前の TASK の件にも保留中の ALERT にも入らない)。R2-L2: 座席の記録がないときや読み書きの失敗のあとは、同じ TASK に 2 つ目の件ができ、rule はそれを書いていない。/sync-docs に渡すもの 6 件は report の「範囲外で /sync-docs に渡すもの」
  - Run 3(cycle 2、cross-review の ACTION_REQUIRED 1 の直し `cc04cb0a` と、self-review の見ていない `b0a9b28a`・`e976f232`・`31d69757`): CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4、Merge: yes。lag の 2 つの広げ方はどの分岐も上げない側に動き、上げなくなる道では stall が前どおり立つ。R3-L1: `e976f232` の遡りの境目のテストが `cc04cb0a` のあと変異 Rc を見分けない(Test gaps 1 が開き直した)。R3-L2: 定数と rule の「10 seconds」、`execWaitDelay` の写し。R3-L3: rule に古い窓と新しい窓が並ぶ。R3-L4: 台帳 565 行目が古い(/sync-docs に渡す)
- [x] Verification artifact created — `docs/reports/verify-2026-11-22-org-result-missing.md`(cycle 1、HEAD `b0a9b28a`)。Verdict: pass。AC1〜AC6 は満たす、full の static verify は rc=0。MEDIUM 1(V1: 別の TASK の `result_missing` は新しい ALERT と件になる決まりが、`--help`、quality-gates、README、`/org`・`/director` skill、仕様 (b) の再発の文に届いていない)、LOW 5、INFO 5。/sync-docs に渡すものは report の同名の節
  - Run 2(cycle 2、HEAD `f9476afb`): Verdict: pass(MEDIUM 0、LOW 2、INFO 6)。cross-review の ACTION_REQUIRED 1 は `cc04cb0a` で閉じ、10 秒の幅はコードの Enter から `sent` の記録までの上限(確かめの 3 s、WaitDelay の 2 s、余白 5 s)と合い、2 つの広げ方はどちらも上げない側に動く。R3-L1〜L3 は `f9476afb` で入った。full の static verify は rc=0。V12・V13(台帳 565 行目と 362・370・431 行目の数と注)と INFO は report の「/sync-docs に渡すもの (Run 2)」
- [x] Test artifact created — `docs/reports/test-2026-11-22-org-result-missing.md`(cycle 1、HEAD `46c5040b`)。Verdict: pass。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は rc 0(1,574 s、`(cached)` 0)。足したテスト 40 関数は `-v -count=1` で SKIP 0。変異 13 個のうち 11 個が落ちた。残る 2 個(履歴が遡れたとみなす境目、leader が応じた保留中の ALERT のあとの再発)は report の Test gaps の 1 と 2
- [x] Sync-docs artifact created — `docs/reports/sync-docs-2026-11-22-org-result-missing.md`(cycle 1、HEAD `e976f232` の木。1 コミット)。verify の一覧 13 件と候補 2 行の判断(台帳 431 と 389 は注を足した)を片づけた。台帳は既存 13 行に注か直し、新しい 1 行(`watch_result_missing_test.go` が 1,672 行)。V1(別の TASK の `result_missing` は新しい ALERT と件)を、`--help`、quality-gates(2 写し)、README、`/org` と `/director` の skill(4 写しずつ)、仕様の (b) に足し、V3(遡れなかった TASK は stall を抑えない)、V4(読み書きの失敗は 2 つ目の件。rule の 2 写し)、V6(recipe の 2 写し)、V7(README と leader の雛形の REVIEW)も直した。掃き出しで、verify の一覧にない 267、376、118、517 にも注を足した。Go に触れた(`--help` の文字列、leader の雛形、テスト 3 本。論理は変えていない)ので、記録だけの修正の例外には当たらない。`e976f232`(test だけ、+92 行、Test gaps 1、2、4 を閉じる。narrow の `go test` は ok、変異 4 つが落ちた)のあとの `run-test.sh` は回していない。AC1 から AC6 を `[x]` にした(verify が Met、test が pass)
- [ ] PR created
