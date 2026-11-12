# herdr が working と言い続けても、画面が入力待ちの座席へ send が届くようにする(#308)

- Status: Approved
- Approved: 2026-11-12 sha256:998740235069
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/308(下流で、herdr が codex の座席を working と報告し続け、入力待ちの座席へ leader が 3 回とも send できなかった)
- Related issue: 308
- Type: fix
- Branch: fix/send-screen-fallback

## Objective

`ralph org send` は herdr の `agent_status` が idle か done になるのを待ってから打つ。herdr が座席の手番の終わりを取りこぼすと、画面は入力待ちでも何も送れない。herdr の待ちが時間切れになったとき、画面が入力待ちだと確かめられれば打つ形に直す(既定で有効、ユーザーの決定)。あわせて、herdr の状態と画面が食い違う座席を見つける手段と、herdr の状態通知の連携(integration)の状態を doctor で見る手段を足す。issue #308 の受け入れ条件 2 つを満たす。

## Scope

- 画面の判定: 座席の手番が終わり、入力欄が空で、動作中の行(`esc to interrupt`、claude の進行中の行、`Retrying`、`Reconnecting`、`Continuing automatically` など)も選択の画面も無いときだけ「入力待ち」とする。判断できない画面は入力待ちとしない
- 実機の画面の採取: 分離した herdr server で、claude と codex の座席の「手番の途中」と「入力待ち」の画面を取り、`internal/org/testdata/resume/` に足す(記録は `docs/evidence/`)
- `ralph org send` のフォールバック: herdr の待ちに `--timeout-ms` から画面の確かめの分(既定 10 秒、`--timeout-ms` の 3 分の 1 を超えない)を除いた時間を使い、時間切れになったら pane を数秒あけて 2 回読む。2 回とも入力待ちで中身が同じなら pane のロックを取り、ロックの下でもう一度読んで入力待ちを確かめ、打つ。入力待ちでなければ、残りの時間で herdr の待ちに戻る。Enter の前に、打った文が入力欄に入ったことを確かめる(resume の `ResumeInputHolds` と同じ)。Enter のあと、送れたかは herdr の状態ではなく画面で確かめる。manifest の `sent` に `via=screen` と herdr が報告していた状態を残す。`--no-screen-fallback` で今の動き(herdr の待ちだけ)に戻せる
- 画面が入力待ちでないときのエラーの文: herdr の状態、画面の判定とその理由(動作中の行、選択の画面、判断できない)、次の手(`ralph org read`、`ralph org status --live`)
- `ralph org status --live`(既定の `ralph org status` は今のまま manifest だけを読む): 座席ごとに herdr の `agent_status` と画面の判定を出し、herdr が working で画面が入力待ちの座席に食い違いの印を付ける
- `ralph doctor`: claude か codex が座席になりうるとき、`herdr integration status` の該当の行を読み、`current` でなければ warn にして `herdr integration install <target>` を案内する。herdr が無いときや命令が失敗したときは info
- 文書: `docs/recipes/isolated-herdr-server.md` に「座席が working のまま」の節(見込みの原因: herdr の状態通知の hook が python3 を呼び、負荷や mise の shim で時間切れになる。未確認)、org skill の send と status の行、`ralph org send` と `ralph org status` の help、AGENTS.md の internal/org の行

## Non-goals

- herdr 自身の直し(状態通知の hook の時間切れ)
- 見張り(`ralph org watch`)に食い違いの条件を足すこと。見張りは pane を読まない(#312a で利用上限のときだけ読む予定)
- `ralph org resume` の判定の変更(working の座席は今までどおり飛ばす)
- herdr の integration を ralph が入れること

## Assumptions

- Claude Code と codex は、手番の途中で打たれた入力を、次の入力として持つ(キューに入る)。手番の途中に打っても今の作業は壊れない。選択の画面への Enter は選択になるので、選択の画面は入力待ちとしない
- 動作中の行の文は CLI の版で変わりうる。判断できない画面は入力待ちとしない(打たない側に倒す)
- `herdr integration status` は 1 行に 1 つの連携を `<target>: <状態> (<パス>)` の形で出す(herdr 0.7.5 で確かめた。状態は `current (v7)`、`not installed` など)
- このマシンの codex の連携は、古い場所(`~/.codex/hooks/herdr-agent-state.sh`、`~/.codex/hooks.json` の SessionStart)に残っていて、herdr 0.7.5 からは `not installed` と見える

## Affected areas

- `internal/org/resume_screens.go`(または新しい `internal/org/send_screen.go`): 入力待ちの判定と動作中の行の一覧
- `internal/org/testdata/resume/`(実機の画面)、`docs/evidence/org-send-screen-probe-2026-11-12.md`(採取の記録)
- `internal/org/verbs.go`(`Send` の待ちとフォールバック、エラーの文)、`internal/org/pane_lock.go`(変えない見込み)
- `internal/org/verbs.go` の `Status` か新しい関数(`--live` の読み取り)
- `internal/cli/org.go`(`send` の `--no-screen-fallback`、`status --live`、help)
- `internal/cli/doctor.go` と新しい `internal/cli/doctor_herdr_integration.go`(herdr の連携の Check)
- `docs/recipes/isolated-herdr-server.md`、`.claude/skills/org/SKILL.md` と写し、AGENTS.md

## Visual review

- Page: `.harness/state/plan-visual/send-screen-fallback.html`
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体、overview、easy-see、detail-overview、detail-send を撮って読んだ。detail-overview の doctor のパスが枠の端に届いていたので列の幅を変え、detail-send の補足の文が矢印に重なっていたので短くした。撮り直して重なりとはみ出しはない。plan advisory の直しのあと、overview の 2 つ目の箱と detail-send(herdr の待ちに戻る矢印、Enter のあとの画面の確かめ)を描き直し、lint と撮影をやり直して問題はない

## Design decisions

Critical forks: None。次の 8 点は、ユーザーの決定(フォールバックは既定で有効)とコードから決められるので自律で選んだ(2026-11-12 の指示「今後も自律的に進めて」)。

- D1 フォールバックの時間: herdr の待ちに `--timeout-ms` から画面の確かめの分(既定 10 秒、`--timeout-ms` の 3 分の 1 まで)を除いた時間を使う。全体は `--timeout-ms` を超えない。2 回の読みのあいだは 3 秒。画面が入力待ちでなければ(動作中の行、選択の画面、判断できない、2 回で違う)、残りの時間で herdr の idle/done の待ちに戻り、そこで idle か done が返れば今までどおりの道で打つ。これで、確かめの分の手前で終わらなかったが `--timeout-ms` のうちに手番を終える座席には、今と同じく届く(plan advisory の 2 で足した)
- D2 ロックの下: herdr の待ちで入った道では、今までどおりロックの下で herdr をもう一度待つ。画面で入った道では、その代わりに画面をもう一度読み、入力待ちのときだけ打つ(herdr はまだ working と言うので、待ち直すと必ず時間切れになるため)
- D3 Enter の前に、打った文が入力欄に入ったことを確かめる(`ResumeInputHolds`)。入っていなければ Enter を押さず、文を打ったが送っていないことと pane を返す(今の `SendProgressTextTyped` と同じ扱い)
- D4 入力待ちの判定は「入力欄が空」に加えて「動作中の行が無い」と「選択の画面でない」を要る。判断できない画面は入力待ちとしない。2 回の読みの中身が違えば入力待ちとしない(画面が動いている)
- D5 `--no-screen-fallback` で今の動きに戻せる。設定のキーは足さない(座席ごと・送りごとに選べれば足り、既定はユーザーの決定どおり有効)
- D6 食い違いは `ralph org status --live` で見る。既定の `ralph org status` は manifest だけを読む今の約束を変えない。doctor は座席を見ない(静的な確かめだけ)ので、herdr の連携の状態を見る
- D8 画面で入った道では、Enter のあとの確かめ(`confirmSubmitted`)を herdr の状態で行わない。`confirmSubmitted` は herdr が working か blocked になれば送れたとみなすが、この道では herdr が working のまま動かないので、文が入力欄に残っていても送れたことになってしまう。代わりに、Enter のあと短いあいだ(今の確かめと同じ上限)画面を読み、入力欄が空になったか動作中の行が出たら送れたとする。入力欄に打った文が残る、画面が読めない、判断できないときは `submit_unconfirmed=true` を残す。Enter をもう一度押すことはしない(今の約束どおり)(plan advisory の 1 で足した)
- D7 doctor の連携の Check は、claude と codex のうち座席になりうるもの(`[org].driver_pool` と `[org.role_models]` から)だけを見る。`current` 以外は warn。herdr が無い、命令が失敗した、出力が読めないときは info にする

## Acceptance criteria

- [ ] AC1 herdr が working と言い続け、画面が入力待ち(2 回の読みで同じ)の座席へ、`ralph org send` が `--timeout-ms` のうちに文を打ち、Enter を押す。manifest の `sent` に `via=screen` が残る(issue の受け入れ条件 1)
- [ ] AC2 画面に動作中の行がある、選択の画面である、判断できない、2 回の読みで中身が違う、のどれかのときは打たない。エラーの文に herdr の状態、画面の判定と理由、次の手が出る(issue の受け入れ条件 1 の「届かない理由と次の手」)
- [ ] AC3 ロックの下の読み直しで入力待ちでなくなっていれば打たない
- [ ] AC4 Enter の前に打った文が入力欄に無ければ Enter を押さず、文を打ったが送っていないことを返す
- [ ] AC10 画面で入った道で、Enter のあとも herdr が working のままで、打った文が入力欄に残っているときは、`sent` に `submit_unconfirmed=true` が残る。入力欄が空になったか動作中の行が出たときだけ確かめが済んだとする
- [ ] AC11 herdr の待ちが確かめの分の手前で時間切れになり、画面が入力待ちでない座席が、`--timeout-ms` のうちに idle か done になれば、今までどおりの道で文が届く(たとえば `--timeout-ms` 30 秒で、20 秒の時点で動作中、25 秒で done → 30 秒のうちに送れる)
- [ ] AC5 `--no-screen-fallback` では今までどおり herdr の待ちだけで、時間切れは今の文で返る。herdr が idle か done を返す道は今と同じ
- [ ] AC6 `ralph org status --live` が座席ごとに herdr の状態と画面の判定を出し、working で入力待ちの座席に食い違いの印を付ける(issue の受け入れ条件 2)。`--live` なしの出力は今と同じ
- [ ] AC7 `ralph doctor` が、座席になりうる claude と codex について herdr の連携が `current` でないとき warn を出し、入れ方を示す。`current` なら pass、herdr が無いときは info
- [ ] AC8 入力待ちの判定が、採取した実機の画面(claude と codex の手番の途中と入力待ち)と既存の testdata のすべてで期待どおりに分かれる
- [ ] AC9 help、org skill、recipe、AGENTS.md がこの挙動に合う

## Implementation outline

1. 実機の画面の採取(`docs/recipes/isolated-herdr-server.md` の分離した herdr server で、claude と codex の座席を 1 つずつ立て、手番の途中と入力待ちの pane を読む)。testdata に足し、記録を `docs/evidence/org-send-screen-probe-2026-11-12.md` に書く — verify: `test -s docs/evidence/org-send-screen-probe-2026-11-12.md && ls internal/org/testdata/resume/ | grep -E 'idle|working'`
2. 入力待ちの判定(動作中の行の一覧、選択の画面、判断できない画面)と、testdata すべてに対する表のテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Screen|Resume'`
3. `Send` のフォールバック(D1〜D5)とエラーの文、`--no-screen-fallback` — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Send|PaneLock|Screen'`
4. `ralph org status --live` と doctor の herdr の連携の Check — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Status|Doctor|Herdr'`
5. 文書(recipe、org skill の 4 つの写し、help の残り、AGENTS.md) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Send|Status|Doctor'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC9。issue #308 の受け入れ条件 2 つ
- Documentation drift to check: `ralph org send` と `ralph org status` と `ralph doctor` の help、org skill の 4 つの写し、`docs/recipes/isolated-herdr-server.md`、AGENTS.md、`.claude/rules/ralph/agent-messaging.md`(send の約束に触れるなら)
- Evidence to capture: 採取した実機の画面と、その記録

## Test plan

- Unit tests:
  - 入力待ちの判定: 採取した画面と既存の testdata の表(claude と codex の入力待ち、手番の途中、再試行中、通信待ち、選択の画面、hook の確認、自動の続行の待ち、入力欄に文がある、判断できない)
  - `Send`: herdr の偽物が working を返し続け、pane の偽物が入力待ちを 2 回返す → 打って Enter、`sent` に `via=screen`。2 回目の読みが違う、動作中の行がある、選択の画面 → 打たずにエラー(文に理由)。ロックの下の読み直しで動作中 → 打たない。打った文が入力欄に無い → Enter を押さない。Enter のあとも herdr が working のままで入力欄に文が残る → `submit_unconfirmed=true`(AC10)。Enter のあと入力欄が空になる → 確かめ済み。画面が動作中で、そのあと herdr が残りの時間のうちに done を返す → 今の道で打つ(AC11)。`--no-screen-fallback` → 今のエラー。herdr が idle を返す → 今と同じ道(画面を読まない)
  - 時間: herdr の待ちに渡す時間が `--timeout-ms` から確かめの分を引いたものであること、全体が `--timeout-ms` を超えないこと
  - `status --live`: herdr の状態と画面の判定の列、食い違いの印、herdr に届かない座席の扱い
  - doctor: `herdr integration status` の出力の読み(current、not installed、知らない状態、herdr が無い、命令の失敗)と、座席になりうる driver の絞り込み
- Integration tests: CLI の `send` と `status --live` を偽物の herdr で通す
- Regression tests: 既存の send、pane ロック、resume、承認の確かめのテストがそのまま通ること
- Edge cases: `--timeout-ms` が短い(確かめの分が取れない → フォールバックしない)、pane が読めない、空の画面、herdr が blocked を返す
- Evidence to capture: `go test` の結果、採取した画面

## Risks and mitigations

- 入力待ちでない座席に打つ(判定の誤り)。対策: 入力欄が空、動作中の行が無い、選択の画面でない、2 回の読みで同じ、ロックの下でもう一度、Enter の前に入力欄の文を確かめる、の 6 段。判断できない画面は打たない。手番の途中に打っても CLI は次の入力として持つ見込み
- 送れていないのに送れたと記録する(herdr が working のままなので、今の Enter のあとの確かめが必ず通る)。対策: 画面で入った道では Enter のあとの確かめを画面で行い、確かめられなければ `submit_unconfirmed=true`(D8)
- 確かめの分を取ったせいで、手番を普通に終える座席に届かなくなる。対策: 画面が入力待ちでなければ残りの時間で herdr の待ちに戻る(D1)
- 選択の画面に文が入る。対策: 選択の画面は入力待ちとしない。Enter の前の確かめで入力欄に文が無ければ Enter を押さない
- 動作中の行の文が CLI の版で変わる。対策: 実機の画面を testdata にし、判断できない画面は打たない側に倒す
- doctor の連携の Check が、連携なしでも動く環境で余計な warn を出す。対策: warn の文に「herdr が座席の手番の終わりを知るための連携」と入れ方を書き、座席になりうる driver だけを見る
- `doctor.go` と `internal/cli/org.go` はレーン B(#310)とレーン C(#311)も触る。対策: 新しい Check は別ファイルに置き、merge の前に origin/main を取り込み、ほかのセッションに知らせる

## Rollout or rollback notes

- 新しい binary の `ralph org send` から効く。座席や herdr の設定の変更は要らない
- 戻すときは revert する。`--no-screen-fallback` で 1 回ずつ今の動きにも戻せる
- skill の文面は `ralph upgrade` で新しくなる

## Open questions

- なし。実機の画面は S1 で採る

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件をどちらも計画に入れた(ユーザーの常設の指示により聞かずに直した)。[HIGH] 画面で入った道では herdr が working のままなので `confirmSubmitted` が必ず通り、入力欄に残った文も送れたことになる → D8・AC10(Enter のあとを画面で確かめ、確かめられなければ `submit_unconfirmed=true`)。[MEDIUM] 確かめの分を取ると、`--timeout-ms` のうちに手番を終える座席に届かなくなる → D1 に「入力待ちでなければ残りの時間で herdr の待ちに戻る」、AC11
- [x] Plan approved — 2026-11-12、自動承認(ユーザーの常設の指示「今後も自律的に進めて」)
- [ ] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
