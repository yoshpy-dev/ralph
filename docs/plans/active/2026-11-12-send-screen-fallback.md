# herdr が working と言い続けても、画面が入力待ちの座席へ send が届くようにする(#308)

- Status: Approved
- Approved: 2026-11-12 sha256:1db244296e07
- Owner: Claude Code
- Date: 2026-11-12
- Related request: https://github.com/yoshpy-dev/ralph/issues/308(下流で、herdr が codex の座席を working と報告し続け、入力待ちの座席へ leader が 3 回とも send できなかった)
- Related issue: 308
- Type: fix
- Branch: fix/send-screen-fallback

## Objective

`ralph org send` は herdr の `agent_status` が idle か done になるのを待ってから打つ。herdr が座席の手番の終わりを取りこぼすと、画面は入力待ちでも何も送れない。herdr の待ちが時間切れになったとき、画面が入力待ちだと確かめられれば打つ形に直す(既定で有効、ユーザーの決定)。あわせて、herdr の状態と画面が食い違う座席を `ralph org status --live` で見つけられるようにする。issue #308 の受け入れ条件 2 つを満たす。

## Scope

- 画面の判定: 入力欄が空で(placeholder は空とみなす。claude の `Press up to edit queued messages` は文が積まれている=動作中)、動作中の行(claude の「記号 + 語 + `…`」の行、codex の `Working (… esc to interrupt)`、`Retrying`、`Reconnecting`、`Continuing automatically` など)も選択の画面も無く、さらに手番が終わった肯定の証拠があるときだけ「入力待ち」とする。肯定の証拠は、手番の終わりの行(claude の `<記号> <語> for <時間> · done <時刻>`、codex の `Worked for <時間>`)が最後の状態の行であること、または起動直後の画面(バナーがあり、入力欄より上に送った文の塊も動作中の行も無い)であること。判断できない画面は入力待ちとしない。入力待ちでない理由(終わりの行が見つからない、を含む)をエラーに出す
- 入力欄の見つけ方の版ずれの直し: codex 0.166.0 の footer の 1 行目(`<model> · <effort> · Context N% used · <branch> · …`)が `resumeCodexFooterLine` に合わず、codex の入力欄が見つからない。共有の関数を直す(resume も codex 0.166 で入力欄を見つけられるようになる。resume の判定の規則は変えない)
- 実機の画面の採取: 分離した herdr server で、claude と codex の座席の「手番の途中」と「入力待ち」の画面を取り、`internal/org/testdata/resume/` に足す(記録は `docs/evidence/`)
- `ralph org send` のフォールバック: herdr の待ちに `--timeout-ms` から画面の確かめの分(既定 10 秒、`--timeout-ms` の 3 分の 1 を超えない)を除いた時間を使い、時間切れになったら pane を数秒あけて 2 回読む。2 回とも入力待ちで中身が同じなら pane のロックを取り、ロックの下でもう一度読んで入力待ちを確かめ、打つ。入力待ちでなければ、残りの時間で herdr の待ちに戻る。Enter の前に、打った文が入力欄に入ったことを確かめる(D3)。Enter のあと、送れたかは herdr の状態ではなく画面で確かめる。manifest の `sent` に `via=screen` と herdr が報告していた状態を残す。`--no-screen-fallback` で今の動き(herdr の待ちだけ)に戻せる
- 画面が入力待ちでないときのエラーの文: herdr の状態、画面の判定とその理由(動作中の行、選択の画面、判断できない)、次の手(`ralph org read`、`ralph org status --live`)
- `ralph org status --live`(既定の `ralph org status` は今のまま manifest だけを読む): 座席ごとに herdr の `agent_status` と画面の判定を出し、herdr が working で画面が入力待ちの座席に食い違いの印を付け、`herdr agent explain <name>` で herdr がそう判断した規則を見られると添える
- 文書: `docs/recipes/isolated-herdr-server.md` に「座席が working のまま」の節(herdr は端末タイトルの回転記号と画面の規則で状態を決める。integration は session を知らせるだけで状態を送らない。下流で working が続いた原因は未確認。`herdr agent explain` で規則を見る)、org skill の send と status の行、`ralph org send` と `ralph org status` の help、AGENTS.md の internal/org の行

## Non-goals

- herdr 自身の直し(状態通知の hook の時間切れ)
- 見張り(`ralph org watch`)に食い違いの条件を足すこと。見張りは pane を読まない(#312a で利用上限のときだけ読む予定)
- `ralph org resume` の判定の変更(working の座席は今までどおり飛ばす)。入力欄の見つけ方の版ずれの直しは入る(Scope)
- herdr の integration を ralph が入れること。doctor で herdr の integration を見ること(当初の D7。S1 で integration は session を知らせるだけで working/idle を送らないと分かり、warn は直す場所を誤らせるので外した)

## Assumptions

- 手番の途中に打った文は、今の手番の作業を変える(S1 で確認、各 1 回。claude は走っているツールの結果の直後に同じ手番でモデルへ渡し、codex は Enter が steer になって待ちを切り上げた。どちらも最初の依頼の答えを書かなかった)。当初は「次の入力として持ち、今の作業は壊れない」と置いていたが成り立たなかった。誤判定のコストが大きいので、入力待ちの判定に手番が終わった肯定の証拠を要る(Scope、D4)。選択の画面への Enter は選択になるので、選択の画面は入力待ちとしない
- 動作中の画面は 3 秒あけた 2 回の読みで必ず違った(秒数、回転記号、ツールの経過)。入力待ちは 10〜40 秒あけても同じだった(S1)
- 動作中の行と手番の終わりの行の文は CLI の版で変わりうる。判断できない画面は入力待ちとしない(打たない側に倒す)。終わりの行が見つからないときは、その理由をエラーに出す
- herdr 0.7.5 は、working を端末タイトルの回転記号(`osc_title_working`)と画面の規則で決める(`herdr agent explain` で確認)。integration のスクリプトは SessionStart で session を知らせるだけで、working/idle は送らない(S1 と consult でスクリプトを読んで確認)

## Affected areas

- 新しい `internal/org/send_screen.go`: 入力待ちの判定と動作中の行・終わりの行の一覧
- `internal/org/resume_screens.go`: `resumeCodexFooterLine` の codex 0.166 の footer への対応だけ
- `internal/org/testdata/resume/`(実機の画面)、`docs/evidence/org-send-screen-probe-2026-11-12.md`(採取の記録)
- `internal/org/verbs.go`(`Send` の待ちとフォールバック、エラーの文)、`internal/org/pane_lock.go`(変えない見込み)
- `internal/org/verbs.go` の `Status` か新しい関数(`--live` の読み取り)
- `internal/cli/org.go`(`send` の `--no-screen-fallback`、`status --live`、help)
- `docs/recipes/isolated-herdr-server.md`、`.claude/skills/org/SKILL.md` と写し、AGENTS.md

## Visual review

- Page: `.harness/state/plan-visual/send-screen-fallback.html`
- Lint: `./scripts/plan-visual.sh lint` が exit 0
- Self-check: 全体、overview、easy-see、detail-overview、detail-send を撮って読んだ。detail-overview の doctor のパスが枠の端に届いていたので列の幅を変え、detail-send の補足の文が矢印に重なっていたので短くした。撮り直して重なりとはみ出しはない。plan advisory の直しのあと、overview の 2 つ目の箱と detail-send(herdr の待ちに戻る矢印、Enter のあとの画面の確かめ)を描き直し、lint と撮影をやり直して問題はない。S1 のあとの直しで、easy-see(doctor の箱を外し、`herdr agent explain` の案内を足した)、detail-overview(doctor のファイルを `resume_screens.go` の footer の直しに替えた)、「承認の前に」の箱を直し、lint と全体・4 つの図の撮影をやり直して問題はない

## Design decisions

S1 のあと(2026-11-12)に、実機の事実に合わせて Assumptions、Scope、D4、D6、D7、D9、AC6〜AC8、Implementation outline の 2 と 4、Test plan、Risks を直し、承認をやり直した(consult の判定「直してから進める」)。

Critical forks: None。次の点は、ユーザーの決定(フォールバックは既定で有効)とコードから決められるので自律で選んだ(2026-11-12 の指示「今後も自律的に進めて」)。

- D1 フォールバックの時間: herdr の待ちに `--timeout-ms` から画面の確かめの分(既定 10 秒、`--timeout-ms` の 3 分の 1 まで)を除いた時間を使う。全体は `--timeout-ms` を超えない。2 回の読みのあいだは 3 秒。画面が入力待ちでなければ(動作中の行、選択の画面、判断できない、2 回で違う)、残りの時間で herdr の idle/done の待ちに戻り、そこで idle か done が返れば今までどおりの道で打つ。これで、確かめの分の手前で終わらなかったが `--timeout-ms` のうちに手番を終える座席には、今と同じく届く(plan advisory の 2 で足した)
- D2 ロックの下: herdr の待ちで入った道では、今までどおりロックの下で herdr をもう一度待つ。画面で入った道では、その代わりに画面をもう一度読み、入力待ちのときだけ打つ(herdr はまだ working と言うので、待ち直すと必ず時間切れになるため)
- D3 Enter の前に、打った文が入力欄に入ったことを確かめる(send 専用の `sendInputHolds`)。入っていなければ Enter を押さず、文を打ったが送っていないことと pane を返す(今の `SendProgressTextTyped` と同じ扱い)。当初は resume の `ResumeInputHolds`(文の頭の 12 字が入力欄にある)を使う予定だったが、S3b の実機の確かめで、中くらい以上の複数行の文は両 CLI とも入力欄に文の終わりしか出ず、claude は約 800 バイトを超える塊を `[Pasted text #N +M lines]` に畳むと分かった。そのため、文の頭が見える、入力欄の文が打った文の終わりと一致する(codex は `↑` の印があるときだけ)、claude の貼り付けの placeholder と文の一部だけでできている、のどれかで確かめる。同じ実機の確かめで、claude が 1117 バイトの文の頭を落とすことも分かった。打つ段階の問題で画面からは見分けられないので、#319 で扱う。cross-review(cycle 1)の指摘で、入力欄の印の行が番号付きの手順(`7. Run the checks`)で始まると、resume と同じ見つけ方では選択の画面とみなされ、Enter が押されないと分かった。send の確かめでは、印の行の文が打った文の一部で、入力欄の形(claude は印の行の上下の罫線、codex は footer)があるときに限り、番号付きの行を入力欄とみなす。resume の判定、打つ前の `ClassifySendScreen`、Enter のあとの `SendScreenSubmitted` は、番号付きの行を今までどおり選択の画面とみなす(打たない側、確かめられない側に倒す)
- D4 入力待ちの判定は「入力欄が空」に加えて「動作中の行が無い」と「選択の画面でない」と「手番が終わった肯定の証拠(終わりの行が最後の状態の行、または起動直後の画面)」を要る。判断できない画面は入力待ちとしない。2 回の読みの中身が違えば入力待ちとしない(画面が動いている)。肯定の証拠はユーザーの決定の文(空の入力欄・動作中の行なし・選択画面なしを 2 回の読みで確かめる)より条件を 1 つ足すが、決定の趣旨(入力待ちだと確かめられれば打つ)の内側なので自律で足した(S1 で手番の途中に打つと作業が変わると分かったため。consult の判定も同じ)。起動直後の画面を入れるのは、起動直後の座席へ最初の TASK が届かない形を残さないため(S1 で herdr は起動直後の codex の入力待ちを working と返した)。動作中の行は行頭の「記号 + 語 + `…`」で緩く取る(折り返した行や herdr の規則に合わない形もあるため)
- D5 `--no-screen-fallback` で今の動きに戻せる。設定のキーは足さない(座席ごと・送りごとに選べれば足り、既定はユーザーの決定どおり有効)
- D6 食い違いは `ralph org status --live` で見る。既定の `ralph org status` は manifest だけを読む今の約束を変えない。食い違いの印には `herdr agent explain <name>` の案内を添える(herdr がどの規則で working と判断したかを見られる)
- D8 画面で入った道では、Enter のあとの確かめ(`confirmSubmitted`)を herdr の状態で行わない。`confirmSubmitted` は herdr が working か blocked になれば送れたとみなすが、この道では herdr が working のまま動かないので、文が入力欄に残っていても送れたことになってしまう。代わりに、Enter のあと短いあいだ(今の確かめと同じ上限)画面を読み、入力欄が空になったか動作中の行が出たら送れたとする。入力欄に打った文が残る、画面が読めない、判断できないときは `submit_unconfirmed=true` を残す。Enter をもう一度押すことはしない(今の約束どおり)(plan advisory の 1 で足した)
- D7 (外した)当初は doctor で `herdr integration status` を見て warn を出す予定だった。S1 で integration のスクリプトは SessionStart で session を知らせるだけで working/idle を送らないと分かった。warn は直す場所を誤らせるので外し、`status --live` の案内と recipe の `herdr agent explain` で代える。issue の受け入れ条件 2 は「status か doctor」なので status で満たす
- D9 codex 0.166 の footer に合わない `resumeCodexFooterLine` は、send の判定も使う共有の関数なので、同じ PR で狭く直す(`Context N% used` のあとの項目を許す)。0.165 の fixture の判定が変わらないことと、0.166 の入力待ちの fixture で入力欄が空と読めることをテストで固める。resume も codex 0.166 で入力欄を見つけられるようになるが、resume の判定の規則は変えない

## Acceptance criteria

- [x] AC1 herdr が working と言い続け、画面が入力待ち(2 回の読みで同じ)の座席へ、`ralph org send` が `--timeout-ms` のうちに文を打ち、Enter を押す。manifest の `sent` に `via=screen` が残る(issue の受け入れ条件 1)
- [x] AC2 画面に動作中の行がある、選択の画面である、手番の終わりの行も起動直後の画面の形も無い、判断できない、2 回の読みで中身が違う、のどれかのときは打たない。エラーの文に herdr の状態、画面の判定と理由、次の手が出る(issue の受け入れ条件 1 の「届かない理由と次の手」)
- [x] AC3 ロックの下の読み直しで入力待ちでなくなっていれば打たない
- [x] AC4 Enter の前に打った文が入力欄に無ければ Enter を押さず、文を打ったが送っていないことを返す
- [x] AC10 画面で入った道で、Enter のあとも herdr が working のままで、打った文が入力欄に残っているときは、`sent` に `submit_unconfirmed=true` が残る。入力欄が空になったか動作中の行が出たときだけ確かめが済んだとする
- [x] AC11 herdr の待ちが確かめの分の手前で時間切れになり、画面が入力待ちでない座席が、`--timeout-ms` のうちに idle か done になれば、今までどおりの道で文が届く(たとえば `--timeout-ms` 30 秒で、20 秒の時点で動作中、25 秒で done → 30 秒のうちに送れる)
- [x] AC5 `--no-screen-fallback` では今までどおり herdr の待ちだけで、時間切れは今の文で返る。herdr が idle か done を返す道は今と同じ
- [x] AC6 `ralph org status --live` が座席ごとに herdr の状態と画面の判定を出し、working で入力待ちの座席に食い違いの印と `herdr agent explain` の案内を付ける(issue の受け入れ条件 2)。`--live` なしの出力は今と同じ
- [x] AC7 codex 0.166 の footer の画面で入力欄が見つかり、codex の入力待ちの fixture で入力欄が空と読める。既存の fixture の `ClassifyResumeScreen` の結果は変わらない
- [x] AC8 入力待ちの判定が、採取した実機の画面(claude と codex の起動直後、手番の途中、入力待ち、積まれた文、送った直後)と既存の testdata のすべてで期待どおりに分かれる
- [x] AC9 help、org skill、recipe、AGENTS.md がこの挙動に合う

## Implementation outline

1. 実機の画面の採取(`docs/recipes/isolated-herdr-server.md` の分離した herdr server で、claude と codex の座席を 1 つずつ立て、手番の途中と入力待ちの pane を読む)。testdata に足し、記録を `docs/evidence/org-send-screen-probe-2026-11-12.md` に書く — verify: `test -s docs/evidence/org-send-screen-probe-2026-11-12.md && ls internal/org/testdata/resume/ | grep -E 'idle|working'`
2. 入力待ちの判定(動作中の行・終わりの行・起動直後の画面、選択の画面、判断できない画面)と `resumeCodexFooterLine` の 0.166 対応(D9)、testdata すべてに対する表のテスト — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Screen|Resume'`
3. `Send` のフォールバック(D1〜D5)とエラーの文、`--no-screen-fallback` — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Send|PaneLock|Screen'`
4. `ralph org status --live` — verify: `test -z "$(gofmt -l internal/org internal/cli)" && go vet ./internal/org/ ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ ./internal/cli/ -run 'Status|Live'`
5. 文書(recipe、org skill の 4 つの写し、help の残り、AGENTS.md) — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ ./internal/org/ -run 'Send|Status|Skill'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`
- Spec compliance criteria to confirm: AC1〜AC11。issue #308 の受け入れ条件 2 つ
- Documentation drift to check: `ralph org send` と `ralph org status` の help、`ralph org resume` の help(入力欄の見つけ方に触れるなら)、org skill の 4 つの写し、`docs/recipes/isolated-herdr-server.md`、AGENTS.md、`.claude/rules/ralph/agent-messaging.md`(send の約束に触れるなら)
- Evidence to capture: 採取した実機の画面と、その記録

## Test plan

- Unit tests:
  - 入力待ちの判定: 採取した画面と既存の testdata の表(claude と codex の起動直後、入力待ち、手番の途中(考え中、ツール、折り返した動作中の行)、積まれた文、送った直後、再試行中、通信待ち、選択の画面、hook の確認、自動の続行の待ち、入力欄に文がある、終わりの行が無い、判断できない)。claude の終わりの行(`✻ … for … · done …`)と動作中の行(`✻ …… `)を取り違えないこと
  - 入力欄の見つけ方: codex 0.166 の footer の画面で入力欄が見つかる。既存の fixture の `ClassifyResumeScreen` の結果が変わらない(AC7)
  - `Send`: herdr の偽物が working を返し続け、pane の偽物が入力待ちを 2 回返す → 打って Enter、`sent` に `via=screen`。2 回目の読みが違う、動作中の行がある、選択の画面 → 打たずにエラー(文に理由)。ロックの下の読み直しで動作中 → 打たない。打った文が入力欄に無い → Enter を押さない。Enter のあとも herdr が working のままで入力欄に文が残る → `submit_unconfirmed=true`(AC10)。Enter のあと入力欄が空になる → 確かめ済み。画面が動作中で、そのあと herdr が残りの時間のうちに done を返す → 今の道で打つ(AC11)。`--no-screen-fallback` → 今のエラー。herdr が idle を返す → 今と同じ道(画面を読まない)
  - 時間: herdr の待ちに渡す時間が `--timeout-ms` から確かめの分を引いたものであること、全体が `--timeout-ms` を超えないこと
  - `status --live`: herdr の状態と画面の判定の列、食い違いの印と `herdr agent explain` の案内、herdr に届かない座席の扱い
- Integration tests: CLI の `send` と `status --live` を偽物の herdr で通す
- Regression tests: 既存の send、pane ロック、resume、承認の確かめのテストがそのまま通ること
- Edge cases: `--timeout-ms` が短い(確かめの分が取れない → フォールバックしない)、pane が読めない、空の画面、herdr が blocked を返す
- Evidence to capture: `go test` の結果、採取した画面

## Risks and mitigations

- 入力待ちでない座席に打つ(判定の誤り)。手番の途中に打つと、その手番の作業が変わる(S1 で確認)。対策: 入力欄が空、動作中の行が無い、選択の画面でない、手番が終わった肯定の証拠(終わりの行か起動直後の画面)、2 回の読みで同じ(動作中は S1 で必ず違った)、ロックの下でもう一度、Enter の前に入力欄の文を確かめる、の 7 段。判断できない画面は打たない。前提が崩れたことは PR 本文にも書き、既定で有効を見直せるようにする
- 終わりの行の文が CLI の版で変わり、フォールバックが黙って効かなくなる。対策: 入力待ちでない理由に「終わりの行が見つからない」を出す
- 送れていないのに送れたと記録する(herdr が working のままなので、今の Enter のあとの確かめが必ず通る)。対策: 画面で入った道では Enter のあとの確かめを画面で行い、確かめられなければ `submit_unconfirmed=true`(D8)
- 確かめの分を取ったせいで、手番を普通に終える座席に届かなくなる。対策: 画面が入力待ちでなければ残りの時間で herdr の待ちに戻る(D1)
- 選択の画面に文が入る。対策: 選択の画面は入力待ちとしない。Enter の前の確かめで入力欄に文が無ければ Enter を押さない
- 動作中の行の文が CLI の版で変わる。対策: 実機の画面を testdata にし、判断できない画面は打たない側に倒す
- `resumeCodexFooterLine` の直しで resume の挙動が変わる。対策: 直しは footer の行の形に限り、既存の fixture の判定が変わらないことをテストで固める。codex 0.166 で resume が入力欄を見つけられるようになるのは版ずれの直しとして PR 本文に書く
- `internal/cli/org.go` はレーン C(#311)も触る。対策: merge の前に origin/main を取り込み、ほかのセッションに知らせる。doctor には触らない

## Rollout or rollback notes

- 新しい binary の `ralph org send` から効く。座席や herdr の設定の変更は要らない
- 戻すときは revert する。`--no-screen-fallback` で 1 回ずつ今の動きにも戻せる
- skill の文面は `ralph upgrade` で新しくなる

## Open questions

- なし。実機の画面は S1 で採る

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件をどちらも計画に入れた(ユーザーの常設の指示により聞かずに直した)。[HIGH] 画面で入った道では herdr が working のままなので `confirmSubmitted` が必ず通り、入力欄に残った文も送れたことになる → D8・AC10(Enter のあとを画面で確かめ、確かめられなければ `submit_unconfirmed=true`)。[MEDIUM] 確かめの分を取ると、`--timeout-ms` のうちに手番を終える座席に届かなくなる → D1 に「入力待ちでなければ残りの時間で herdr の待ちに戻る」、AC11
- [x] Plan approved — 2026-11-12、自動承認(ユーザーの常設の指示「今後も自律的に進めて」)。S1(11d94016)のあと、手番の途中に打つと作業が変わる、integration は状態を送らない、codex 0.166 の footer に合わない、と分かったので計画を直し、consult(判定「直してから進める」: 起動直後の画面も肯定の証拠に入れる)を入れて自動で承認し直した
- [x] S1 実機の画面の採取 — 11d94016(implementer、claude と codex の 23 fixture と記録。verify の命令と `go test ./internal/org/ -run Resume` が通った。自己点検 5 項目すべて clean または fixed)
- [x] S2 入力待ちの判定と codex 0.166 の footer — d3765dce(implementer。verify の命令が通った。自己点検: 貼り間違い 1 件を fixed、ほか clean。footer の正規表現の説明の 1 文も直した(handoff より少し広い))
- [x] S3 send の画面のフォールバック — a18db28f(implementer。verify の命令と resume のテストが通った。自己点検: 語順と関数名の書き違いを fixed、ほか clean。最低の確かめの分は「読みの間隔×2 + Enter の前の待ち」で既定 6.75 秒、`--timeout-ms` が 20.25 秒未満ならフォールバックしない。範囲外の所見: `pane_lock.go` 冒頭と `resume.go` の `defaultResumePaneLockWait` のコメントが画面の道を書いていない → S5 で直す)
- [x] S3b 複数行の文の入力欄の実機の確かめと `sendInputHolds` — 814f8fb7(implementer。verify の命令が通った。自己点検: 誤字 2 か所を fixed、ほか clean。claude の頭の欠けは #319 に起票)。origin/main(#316、d4bafaa7)を 3db272db で取り込み、build と cli のテストが通った。計画の D3 を実装に合わせて直し、承認をやり直した
- [x] S4 `ralph org status --live` — c08730fb(implementer。verify の命令が通った。自己点検: テストのコメント 2 か所と期待値の抜け 1 件を fixed、ほか clean。JSON に `herdr_error` と `screen_line` を足した)
- [x] S5 文書 — 91623b11(implementer。check-skill-sync、check-sync、対象のテストが通った。自己点検すべて clean。recipe の templates/base の双子も同じ編集(check-sync の対象)。send の help に「Enter の前の待ちが 4 秒を超えると画面を読まない」を足した)
- [x] 範囲外の所見の直し(inline、1 ファイルのコメントだけ) — 398eee06。Send の doc コメントの `ResumeInputHolds` を `sendInputHolds` に。gofmt と go vet が通った。自己点検: 不要な変更 clean、誤字と貼り間違い clean、デバッグの残り clean、秘密 clean、範囲外のファイル clean
- [x] S6 self-review の L1・L3・L8 の直し — 07e01081(implementer。確かめ全体を reserve の半分で区切り、herdr の呼び出しを 1 回ごとに区切る。自己点検すべて clean)。deddebc9 で Send の doc コメントに時間の配分を足した(inline、コメントだけ、gofmt と go vet が通った、自己点検 5 項目すべて clean)
- [x] S7 CLI から画面の道を通すテスト — 88742506(implementer、テストだけ。自己点検: コメントの誤り 1 件を fixed、ほか clean)
- [x] パイプライン cycle 1: self-review(LOW 7、Merge: yes)、verify(pass)、test(pass、生き残った変異 M02)、sync-docs、cross-review(codex、ACTION_REQUIRED 1: 入力欄の番号付きの行を選択の画面とみなす)。常設の指示により聞かずに直し、cycle 2 へ
- [x] S8 番号付きの行の直しと M02 のテスト — f24290b5(implementer。resume の判定は全 fixture で変わらないことを overlay で確かめた。自己点検: 誤字を fixed、ほか clean)。origin/main(#321、0936aaea)を 39004c89 で取り込み(AGENTS.md と tech-debt の衝突は両側の追記をどちらも残した)、build、drift の確かめ、関係するテストが通った。D3 に番号付きの行の扱いを足し、承認をやり直した
- [ ] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [ ] PR created
