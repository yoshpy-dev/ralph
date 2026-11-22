# org-result-missing

- Status: Approved
- Approved: 2026-11-22 sha256:b05c342c6a28
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
  - 台帳の `sent` から、その座席への最後の TASK の TASK_ID と、基準の時刻を読む。基準の時刻は、その TASK 以後に leader がその座席に送った最後のメッセージ(TASK そのもの、QUESTION への答えの DECISION など)の時刻
  - agmsg の履歴から、基準の時刻以後(同じ秒を落とさない。台帳も agmsg も UTC の秒で比べられる)の、`<seat> → leader` の行のうち、その TASK_ID のメッセージを読む(座席を名指した履歴には座席が受けた行も入るので、送り手と宛先で絞る)。RESULT か BLOCKED か REVIEW があれば済み(上げない)。QUESTION があれば leader の答え待ち(上げない。leader が答えると基準の時刻が進み、数え直す)。どちらもなく、座席が入力待ち(herdr の状態が working でない)のまま基準の時刻から `[org.watchdog].stall_minutes` を超えていれば、ALERT(`CONDITION: result_missing`、`SEAT`、`TASK_ID`、本文に「TASK <id> を受けた座席 <seat> が入力待ちのまま、RESULT も BLOCKED も送っていない」)を leader に送る(codex の計画助言の指摘 1: QUESTION に答えたあとの送り損ねを見逃さないため)
  - 履歴の範囲: 座席を名指して(`history.sh <team> <seat> <limit>`)読み、読めた範囲が基準の時刻まで遡れたかを確かめる。いちばん古い行が基準の時刻より前か、取れた行の数が上限より少なければ(全部を読めた)遡れた、とする。遡れなければ上限を上げて読み直し(決まった回数と上限まで)、それでも遡れなければ「わからない」として上げない。基準の時刻より後の行が 1 つもなく、範囲が遡れているなら、それは送っていないことなので上げる。一度 RESULT か BLOCKED を確かめた TASK_ID は見張りの状態に覚え、あとで件数の上限で行が範囲から落ちても上げない(codex の計画助言の指摘 2)
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

## Design decisions

Critical forks: None

- 見張りの待ち時間は新しい設定の鍵を足さず、`[org.watchdog].stall_minutes` を使う。座席が入力待ちで結果を送らない時間は、座席の停滞と同じ尺度で見てよい
- 履歴を読めなかったときは ALERT を上げない。誤った ALERT で leader を止めないため。読めないことが続くときの扱いは tech-debt に書く
- 座席の送信の確かめは、終了コードと履歴の両方で行う。終了コードが 0 でも、送るコマンドを打たずに手番を終えた場合(issue の claude の 1 件)は、履歴の確かめで気づける

## Acceptance criteria

- [ ] AC1(S0): `ralph-dispatch.sh PreToolUse` に座席の形の送信を渡した結果(1 段目)と、1 段目で落ちなかったときは codex の座席と同じ sandbox で送信と hook を走らせた結果(2 段目)が `docs/evidence/` にある。原因が見つかったなら直しがこの計画に入っているか、tech-debt と issue に書いてある
- [ ] AC2(S1): implementer と reviewer の雛形が、送信のコマンドの形、送信の終了コードの確かめ、1 回の送り直し、失敗したときの `SEND FAILED:` の 1 行、終了コードが 0 のときの履歴の確かめ、TASK に結果を送らずに手番を終えないことを言う(テストで固定)
- [ ] AC3(S2): 見張りが、TASK を受けた座席が入力待ちのまま `stall_minutes` を超えて、その TASK_ID の RESULT・BLOCKED・QUESTION・REVIEW を送っていないとき、`CONDITION: result_missing` と `SEAT` と `TASK_ID` を持つ ALERT を leader に送る(fake の herdr と fake の agmsg の履歴のテストで固定)
- [ ] AC4(S2): 一致するメッセージがあるとき、座席が作業中のとき、時間がまだのとき、履歴を読めないとき、leader の座席には上げない。一致するメッセージが現れるか新しい TASK が来ると消える。同じ座席に stall と重ねて出さない
- [ ] AC6(S2): 何周期にもわたるテストで、次の流れの答えを固定する。(a) TASK → QUESTION → leader の DECISION → 座席が入力待ちのまま RESULT を送らない、で上げる。(b) TASK → QUESTION のまま leader が答えていない、で上げない。(c) TASK のあと座席からの行が 1 つもなく、範囲が遡れている、で上げる。(d) 正常な RESULT が件数の上限で範囲から落ちた(最新の N 件がすべて基準の時刻より後)、で上限を上げて読み直し、それでも遡れなければ上げない。一度確かめた RESULT は、あとで範囲から落ちても上げない。(e) 座席あての行(`leader → <seat>` の TASK や DECISION)だけがあり、`<seat> → leader` の行がない、で上げる(受けた行を送った行と取り違えない)。(f) RESULT が基準の時刻と同じ秒、で上げない
- [ ] AC5: leader の雛形と skill と rule が `result_missing` の扱いを言う。`./scripts/check-sync.sh` と `./scripts/check-skill-sync.sh` が通る

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
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
