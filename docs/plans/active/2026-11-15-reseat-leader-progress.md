# 立て直した leader のタスクに台帳から作る「立て直しの経過」を足し、leader.md に済んだ手順を飛ばす規則を足す(#312c-2)

- Status: Approved
- Approved: 2026-11-15 sha256:de0cca0b6f78
- Owner: Claude Code
- Date: 2026-11-15
- Related request: https://github.com/yoshpy-dev/ralph/issues/312(立て直した leader へ台帳から経過を渡す手立てが無い。仕様 `docs/specs/2026-11-12-org-reseat.md` の 2 本目: D7、FR-9、FR-10、AC-13、AC-14)
- Related issue: 312
- Type: feat
- Branch: feat/reseat-leader-progress

## Objective

機能の org の leader を立て直したとき(`ralph org reseat` でも、`ralph org start --plan` の立て直しでも)、leader のタスクの終わりに、その org の今の命の経過を台帳から作った「立て直しの経過」の節を足す。中身は、最新の計画の承認、最後の gate の記録、片付いていない受信箱の件、branch の commit、立て直しの記録。`internal/org/prompts/leader.md` には、その節があれば済んだ手順を飛ばして再開する規則を足す。経過の無い org では節を入れない。issue #312 は、この PR で閉じる(受け入れ条件 1 は #312a の PR #327、2 は #312c-1 の PR #333、3 は #312b の PR #320 で満たした)。

## Scope

- 経過を読む関数と節を作る関数(新しい `internal/org/reseat_progress.go`): 今の命の最新の `plan_approved`、最後の gate の記録、org で絞った受信箱の open と acked の件、branch の commit、立て直しの記録
- `StartFeature`(`internal/org/feature.go`)で、leader を起動する前にタスクへ節を足す。reseat は今回の立て直しの id と座席を `StartFeatureParams` で渡す(`internal/org/reseat.go`)
- `internal/org/prompts/leader.md` の「機能ごとの org」の導入に、経過の節があるときの再開の規則
- 役割の prompt のテストで規則の文を固定する。節の中身と、節が無い場合をテストする
- 仕様の FR-9、FR-10、AC-13、AC-14 の印。仕様の AC-12 を `[ ]` に戻し、実機の限りの文を残す(1 本目の引き継ぎ、RC2 と同じ扱い)
- 文書(AGENTS.md の internal/org の行、`/org` と `/director` の skill の reseat の説明、tech-debt)は sync-docs で合わせる

## Non-goals

- 非 leader の座席のタスクに経過を足すこと(仕様は leader だけ。非 leader は D5 の DECISION の行で leader が TASK を送り直す)
- `gate_accepted` の記録を節に入れること(FR-9 に無い。必要なら後で足す)
- 経過を読んで ralph が手順を自動で飛ばすこと(飛ばすのは leader の判断。ralph は材料を渡すだけ)
- 1 本目(`ralph org reseat` の動詞、PR #333)の挙動の変更

## Assumptions

- 実装は PR #333(#312c-1)の merge のあと、その上に重ねる(origin/main を取り込んでから slice を始める)。この計画の関数名と行番号は、#333 の branch(`origin/docs/spec-org-reseat`、49d118c7)で読んだもの
- `featureLeaderTask` の呼び手は `startFeatureLeaderParams` だけで、reseat も `start --plan` も `StartFeature` を通る(reseat は reseat.go の `startAll` から `StartFeature` を呼び、`start.Note = "reseat=<id>"`)。`StartFeature` は leader の spawn の前に台帳の events を読む
- `reseated` の event は、reseat の全座席の停止と起動のあとに書かれる(reseat.go の `finish()`)。leader の起動の時点では、今回の立て直しの記録は台帳にまだ無い
- 受信箱の件には命の印が無い。`OrgID` と、`disbanded` の時刻より後の `EscalatedAt` で今の命の件を選ぶ
- タスクの文は leader の役割の prompt の `{{TASK}}` に入る。`## ` で始まる行は prompt の節の境目として読まれるので、前文と同じく fence で囲む(`fenceFor`)

## Affected areas

- `internal/org/reseat_progress.go`(新)と `internal/org/reseat_progress_test.go`(新)
- `internal/org/feature.go`(`StartFeature` と `StartFeatureParams`)、`internal/org/reseat.go`(`startAll` が今回の立て直しを渡す)、それぞれのテスト
- `internal/org/prompts/leader.md`、役割の prompt のテスト(新しい `internal/org/prompts_leader_reseat_test.go`)
- `docs/specs/2026-11-12-org-reseat.md`(印と AC-12)
- sync-docs で: AGENTS.md の internal/org の行、`/org` と `/director` の skill の 4 つの写し(reseat の説明)、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/reseat-leader-progress.html`(やさしい図 3 枚: overview、easy-section、easy-skip。折りたたみの中に詳しい図 3 枚: detail-overview、detail-flow、detail-rule)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- self-check: 全体と 6 枚の図をそれぞれ `shot` で撮って読んだ(図解のサブエージェントが撮り、orchestrator が overview を読み直した)。計画助言と consult の反映(起動予定の座席、gate を使う分岐、コミット済みの計画)で easy-skip・detail-flow を直し、detail-rule を足して撮り直した。重なり・はみ出し・切れなし。図解の中で見つかった AC2 と R8 の食い違いは計画の AC2・D6 を直した

## Design decisions

critical forks は無い。下の D1〜D7 は、仕様と #333 のコードと既存の型(前文の fence、gate の base の読み方)から決まる。推す案を自律で選んだ。

- D1 節を足す場所(自律で選んだ): `StartFeature` の中で、台帳の events を読んだあと、leader を spawn する前に `s.leader.Task` の終わりに足す。reseat と `start --plan` の立て直しの両方がここを通るので、受け取るものが同じになる(仕様 D7)。`featureLeaderTask` は分割計画だけから作る今の形のまま残す(テストの完全一致を崩さない)。新しい start でも同じ判定をし、経過が空なら節を入れない
- D2 今回の立て直し(自律で選んだ): `reseated` の event は全座席の起動のあとに書かれるので、reseat は `StartFeatureParams` に今回の立て直しの id と対象の座席を渡し、節は「今回の立て直し」として載せる。leader は reseat が最初に起動するので、ほかの座席はこの時点ではまだ立っていない。節ではそれを「この reseat が leader のあとに起動する予定の座席」と書く(計画助言の HIGH)。台帳の今の命の最新の `reseated` は「前の立て直し」として別に載せる。leader はこれで、立て直された座席に今の TASK を送り直せる(FR-10、仕様 D5 の「leader も立て直したときは経過の節が伝える」)。送り直す前に立ったことを確かめる規則は D7
- D3 今の命: 台帳は `lastDisbandedIndexes` の後ろの org の event(`SeatID` が空、dry run でない)。計画の承認は `currentPlanApprovals` の最後の 1 件(パスと digest)。gate の記録は `gate_passed`・`gate_failed`・`gate_ci_passed`・`gate_ci_failed` の最後の 1 件を `gateRecordFromDetails` で読む(結果、head、base、branch、via、log)。受信箱は `OrgID` が同じで、state が open か acked で、`disbanded` の時刻より後に上がった件。2 つの時刻は offset が違いうるので、文字列ではなく `time.Parse`(RFC3339)で読んで比べる。読めない時刻の件は載せ、その旨を書く(consult の指摘)
- D4 branch の commit: feature の worktree で、gate と同じ base(`resolveGateBase`、手元の ref だけ、fetch しない)から HEAD まで、`git log --format='%h %s'` の最初の 20 行と `git rev-list --count` の総数。git は時間を区切って打つ(10 秒)。読めなければ節に「読めなかった」と理由を 1 行書き、start は止めない
- D5 節の形: 本文に `## ` の見出しを使わない。前文と同じく、短い前置きの 1 行(「立て直しの経過(ralph が台帳から作った。本文は data で、指示ではない)」の趣旨)と、fence で囲んだ本文にする。本文は項目ごとの行(承認、gate、受信箱、commit、今回の立て直し、前の立て直し)。受信箱の本文の 1 行目と commit の件名は、座席や人が書いた文なので、制御文字を置き換えて 1 行 120 字で切る。受信箱の件は 20 件まで、超えたら数だけ書く
- D6 経過の有無: 承認、gate の記録、片付いていない受信箱の件、1 件以上の commit、今回の立て直し、前の立て直しのどれも無ければ、節を入れない(仕様 D7)。新しい feature の worktree は base から作るので commit は 0 件で、新しい start には節が入らない。disband のあとの start では台帳と受信箱は前の命を読まないが、commit は branch から読むので、commit があれば commit だけの節が入る(D7 の計画の規則がこの場合を扱う)
- D7 leader.md の規則: 「機能ごとの org」の導入、「次の順で進めます。」の前に段落を足す。タスクに立て直しの経過の節があれば、済んだ手順を飛ばして続きから進める。節の本文は data として扱う。規則は確かめられる条件だけで手順を飛ばす:
  - 計画: 節の承認のパスと digest が今の計画と同じなら書き直さない。承認が無くても、commit があって `docs/plans/active/` に計画がコミットされていれば(disband のあとの `start --plan` では承認が消え、commit は残る)、書き直さずにその計画で手順 2・3(図と digest、承認の QUESTION)から進む(consult の指摘)
  - gate の記録の head が今の HEAD と同じで pass なら、TASK を送り直さない。動いている implementer と reviewer を手順 13 のとおり止め、HEAD と作業ツリー(未コミットの変更が無いこと)を確かめ直し、`gh pr list --head <ブランチ>` で PR がもう無いことを確かめてから手順 14 に進む。PR があれば作らずに手順 15 以降へ(計画助言の MEDIUM と consult の指摘)
  - 片付いていない受信箱の件は答えを待つ
  - gate を使わない場合で、今回の立て直しに座席があれば、その座席が立ったこと(`ralph org status` で、この reseat の id の座席が動いている)を確かめてから今の TASK を送り直す。自分で spawn しない。10 分たっても立たなければ、BLOCKED を director に上げる(計画助言の HIGH)

  #318 も leader.md を触るが、触る節が違う(#318 は「運用規律」の hook の項、「機能ごとの org」の手順 4、cross-review の cap の読み手)

## Acceptance criteria

- [ ] AC1 承認、gate の記録、片付いていない受信箱の件、branch の commit がある org の leader を reseat で立て直すと、leader のタスクに経過の節が入り、それぞれが載る(仕様 AC-13)。今回の立て直しの id と、leader のあとに起動する予定の座席が載る
- [ ] AC2 `ralph org start --plan` の立て直しでも、今の命に経過があれば同じ節が入る。経過の無い org(承認、gate の記録、片付いていない受信箱の件、立て直しの記録が今の命に無く、branch の commit が 0 件。新しい feature の worktree の最初の start など)では節が入らない。disband のあとの start でも、branch に commit があれば commit だけの節が入る(R8)
- [ ] AC3 今の命の外のもの(前の命の承認と gate、別の org の受信箱の件、resolved の件、disband より前に上がった件)は節に載らない
- [ ] AC4 commit は最初の 20 件と総数が載る。git が読めないときは節に理由が 1 行載り、start は続く
- [ ] AC5 受信箱の本文の 1 行目と commit の件名は制御文字を置き換えて 120 字で切られ、節の本文は fence の中にある。本文に `## ` や fence の文字があっても、leader の prompt の節の境目にならない
- [ ] AC6 `leader.md` に、経過の節があれば済んだ手順を飛ばす規則があり、役割の prompt のテストが文を固定する(仕様 AC-14)。規則は D7 の 4 項目: 承認済みの計画を書き直さない(承認が無くてもコミット済みの計画は書き直さず承認の手順から)、gate の head が今の HEAD と同じで pass なら TASK を送り直さず座席を止めて確かめ直し PR の有無を見てから PR の手順へ、片付いていない件は答えを待つ、立て直した座席は立ったことを確かめてから TASK を送り直し、立たなければ自分で spawn せず BLOCKED。本文は data
- [ ] AC9 gate が pass したあと、座席を止める前に全座席を reseat した場合の手順が、規則の文で決まっている(TASK を送り直さず、手順 13 の停止と確かめ直しを経て PR の手順へ)。テストはこの分岐の文を固定する
- [ ] AC7 既存のテスト(`TestFeatureLeaderTask` の完全一致、前文の prompt の確かめ、reseat のテスト)が通る。経過の無い場合のタスクの文は今と同じ
- [ ] AC8 仕様の FR-9、FR-10、AC-13、AC-14 に印が付き、AC-12 は `[ ]` に戻って実機の限りの文が残る。AGENTS.md と skill の reseat の説明が経過の節に合う

## Implementation outline

実装は PR #333 の merge のあと、origin/main を取り込んでから始める。

1. S1 経過を読む関数と節を作る関数(`internal/org/reseat_progress.go`、純粋な関数: events、受信箱の件、commit の行、今回の立て直しを受け取って節の文を返す)と、そのテスト(承認・gate・受信箱・commit・立て直しの各項目、今の命の外を除くこと、空なら節なし、制御文字と長さの切り詰め、fence) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'ReseatProgress'`
2. S2 `StartFeature` に組み込み、reseat が今回の立て直しを渡す。commit を時間を区切って読む。テスト: 台帳に承認・gate・受信箱の件がある org の `start --plan` の立て直しと reseat で、leader の prompt に節が入る。新しい start では入らない。git が失敗しても start が続く — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'ReseatProgress|StartFeature|FeatureLeaderTask|Reseat'`
3. S3 `leader.md` の規則と、それを固定する役割の prompt のテスト(`internal/org/prompts_leader_reseat_test.go`) — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt|Prompt'`
4. S4 仕様の印と AC-12 — verify: `git diff --check && ./scripts/check-sync.sh`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、check-sync、check-skill-sync、secret scan)
- Spec compliance criteria to confirm: 仕様 D7・FR-9・FR-10・AC-13・AC-14 と、この計画の AC1〜AC8。節の項目が FR-9 の 5 つ(承認、gate、受信箱、commit、立て直し)をすべて持つこと
- Documentation drift to check: AGENTS.md の internal/org の行(`reseat_progress.go`)、`/org`・`/director` の skill の reseat の説明(4 つの写し)、`ralph org reseat --help` と `ralph org start --help` が経過の節に触れるなら合うこと、仕様の Status の段落
- Evidence to capture: verify の report(`docs/reports/verify-<date>-reseat-leader-progress.md`)と static verify の log

## Test plan

- Unit tests: 節を作る関数の表のテスト(各項目、空の org、今の命の外、resolved の件、別の org の件、disband より前の件、21 件以上の commit と受信箱、制御文字、長い件名、本文の `## ` と fence 文字)
- Integration tests: `StartFeature` に台帳と受信箱を用意して、立て直しの leader の prompt に節が入ること、新しい start に入らないこと。reseat の leader の立て直しで今回の立て直しの id と座席が載ること。本物の git の worktree で commit の行と総数
- Regression tests: `TestFeatureLeaderTask` の完全一致、`TestStartFeature_LeaderPromptHoldsPreamble`、prompt のテスト(タスクが `- 分割計画:` で始まる)、reseat の既存のテスト
- Edge cases: 受信箱が nil(`o.Inbox` が無い)、受信箱のファイルが壊れた行を持つ、gate の Details が読めない記録、base が見つからない worktree、git の時間切れ、`EscalatedAt` と disband の時刻の offset が違う(`+09:00` と `Z`)、時刻が読めない件、disband のあとの `start --plan`(承認なしで commit だけの節)
- Prompt tests: leader.md の規則の 4 項目と AC9 の分岐の文(TASK を送り直さない、手順 13 の停止、`gh pr list --head`、立ったことを確かめる、BLOCKED、本文は data)を固定する
- Evidence to capture: test の report(`docs/reports/test-<date>-reseat-leader-progress.md`)と `run-test.sh` の log

## Risks and mitigations

- R1 #333 が merge までに変わる: 実装は merge のあとに始め、関数名と行番号は merge 後の main で読み直す
- R2 #318 と leader.md で衝突する: 触る節が違う(D7)。計画の承認のあと、触る節を調整役に知らせ、merge の前に origin/main を取り込む
- R3 節が leader の prompt を壊す: fence で囲み、本文を data と明記する(D5)。座席や人が書いた文は制御文字を置き換えて切る
- R4 leader が古い経過で手順を飛ばしすぎる: 規則は「gate の head が今の HEAD と同じとき」など、確かめられる条件だけで飛ばす(D7)。節の時刻を載せる
- R5 git の読み取りで start が遅れる、止まる: 10 秒で区切り、失敗は 1 行の理由にして start を続ける(D4)
- R6 新しい leader が、まだ立っていない座席に TASK を送る、自分で spawn して reseat と競合する: 節は「起動する予定」と書き、規則は立ったことを確かめてから送り、立たなければ BLOCKED にする(D2、D7)。reseat の後の座席の起動の失敗は、仕様 D5 のとおり reseat の打ち直しが立てる
- R7 gate の記録を使って PR に進むとき、座席を止めずに PR を作り、同じ worktree で作業が続く: gate を使う分岐では TASK を送り直さず、手順 13 で止めてから確かめ直す(D7、AC9)
- R8 disband のあとの `start --plan` で、承認が消えて commit だけの節が入り、leader が計画を書き直す: コミット済みの計画は書き直さず承認の手順から(D7)

## Rollout or rollback notes

- 戻すときはこの PR の revert で済む。台帳の形は変えない(読むだけ)。経過の無い org の leader のタスクは今と同じ
- 次の release のノート: 立て直した leader のタスクに経過の節が入ること(新しい binary が要る。走っている org の leader には、次の立て直しから入る)

## Open questions

- なし(実装の前に、#333 の merge 後の main で D1〜D4 の関数名を読み直す)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex。指摘 2 件を計画に反映した(ユーザーの指示: 計画助言の修正提案は聞かずに直す)。(1) [HIGH] 新しい leader が、まだ立っていない座席に TASK を送る、自分で spawn して reseat と競合する → 節は「leader のあとに起動する予定の座席」と書き、規則は `ralph org status` で立ったことを確かめてから送り、自分で spawn せず、10 分で立たなければ BLOCKED(D2、D7、AC6、R6)。(2) [MEDIUM] gate の記録を使って PR の手順に進むと手順 13(座席の停止)を飛ばす → gate を使う分岐では TASK を送り直さず、座席を止めて HEAD と作業ツリーを確かめ直してから手順 14(D7、AC9、R7)。consult(consult-plan-reseatprogress)の判定「直してから進める」も反映した: disband のあとの `start --plan` で承認が無くてもコミット済みの計画は書き直さない(D7、R8)、`EscalatedAt` と disband の時刻を `time.Parse` で比べる(D3)、手順 14 の前に `gh pr list --head` で PR の有無を見る(D7)
- [x] Plan approved — ユーザーの指示(調整役から伝えられた「私の承認は不要です。あなたたちの推奨案を自動承認します」)に従って承認を記録した。実装は PR #333(#312c-1)の merge のあと、origin/main を取り込んでから始める
- [x] Branch created
- [x] Implementation started
  - 3ac29c5b: origin/main(24e6723c、PR #333 の `ralph org reseat`)を merge した。衝突なし
  - S1 4b9591f8(implementer、sonnet): `internal/org/reseat_progress.go` に `readReseatProgress`(台帳と受信箱から今の命の承認・gate・前の立て直し・open と acked の件)と `reseatProgressSection`(節の文、空なら "")。承認は TS を持たせるため `currentPlanApprovals` と同じ絞り方で読み直し、テストで一致を確かめた。受信箱の本文の 1 行目は `ralph org inbox` と同じく `protocol.Parse` でヘッダの後ろから取る。制御文字は `PrintableInboxText` と U+2028・U+2029 の置き換え。1 行目と commit の件名は 120 字、パスや digest などは 400 字で切る(パスを途中で切ると使えないため。計画の D5 の「120 字」は座席や人が書いた文に当てた)。検証コマンド rc=0、変異 19 個がすべて落ちた。自己点検 clean
  - S2 9d4523e9(implementer、sonnet): `StartFeatureParams.Reseat`(`StartFeatureReseat{ID, Seats}`、nil は reseat でない)。`StartFeature` は台帳を読んだあと、leader の spawn の前に `startProgressSection` で節を作り、空でなければタスクに足す(`featureLeaderTask` は変えない)。commit は `resolveGateBase` の base から `git log --max-count=20` と `git rev-list --count` を 1 つの 10 秒の context で打ち、base が無ければ 0 件でエラーなし。reseat の `startAll` は `plan.Start` から leader を除いた座席を起動順に渡す。読み取りの失敗(commit、受信箱)だけでは節を作らず、節があるときに 1 行足す(計画の D6・AC7 のとおり、新しい start のタスクは今と同じ。S1 のテストを直した)。`StartFeature` に dry run は無いので分岐なし。検証コマンド rc=0、StartFeature を呼ぶ 6 ファイル 103 件と cli の start・reseat の 104 件も通った。変異 10 個がすべて落ちた。自己点検 clean
  - S3 d5fa6703(implementer、sonnet): leader.md の「機能ごとの org」の導入、「次の順で進めます。」の前に、導入の段落と 4 項目(計画、gate、受信箱、座席)。草案から足した点: 承認の行のパスが手順 10 で archive に移っていれば archive の同名ファイルも見る、先頭 2 行が Approved でなければ手順 4 のとおり直して手順 5 から、パスか digest が違えば承認なしと同じに扱う、コミット済みの計画は `git log --name-only` で探す、gate の分岐の `HEAD:`・`BASE:` は節の gate の行から取り `SYNC_DOCS:` などの控えは branch の report から読む、受信箱は QUESTION と BLOCKED の答えを待つ。`prompts_leader_reseat_test.go` の 8 テストが文と手順の番号と節の行の頭を固定する。検証コマンド rc=0、変異 13 個がすべて落ちた。自己点検 clean。implementer が気づいた範囲外の点(sync-docs で tech-debt と Known gaps に回す): `gh pr list --state all` は merge されずに閉じた PR も「ある」と読む、PR を作ったあと RESULT を上げる前に止まった leader は gate の分岐で手順 15 に進むので RESULT が上がらない、承認なしのコミット済みの計画から手順 2 に進むとき計画助言が済んでいたか分からない
  - 95b5e805: origin/main(df186374、#318 の PR #334)を merge した。文字の衝突なし。#334 が手順 4 の書き出しを変えたので、S3 のテストの手順 4 の固定を「先頭の 2 行を書き換える」文で確かめる形に直した(inline、1 ファイル、自己点検 clean。`-run 'RenderRolePrompt_Leader|ReseatProgress|StartFeature|Reseat|Notice'` が通る)
  - self-review(cycle 1、64c7f828): Merge: yes、MEDIUM 1(M1: leader だけの立て直しや、implementer を動かしたままの立て直しが規則のどの項目にも当たらず、手順 5〜8 で動いている座席に TASK を重ねうる)、LOW 7
  - 71ddcb30(implementer、sonnet、self-review の直し): M1・L7 は規則の先頭に「座席の確認」(いつも行う。`ralph org status` で、今回の立て直しに無く `(active)` の座席には TASK を送り直さず、手順 5 で spawn もし直さず、RESULT か BLOCKED を待つ。手順 5 で spawn するのはどの座席も持っていない役割だけ)。gate の項目は今回の立て直しの座席が立つのを待ってから手順 13 で止める。L6 は計画の項目の行き先(手順 5)。L1 は `git log --no-show-signature`(偽の gpg で `log.showSignature=true` のテスト)。L2 は見出しに順(commit は新しい順、受信箱は古い順で、超えた分は「新しいもの」)。L3 は承認の行を `sha256:<digest> (承認 <TS>)` にして digest を 1 語にした。L4 は Go のコメントの計画のパスを archive に。検証コマンド rc=0、変異 3 個がすべて落ちた。自己点検 clean。残した点: L5(`reseatProgressSection` の長さ、`inboxSummary` との二重、`NewReplacer` を毎回作る)と gate の行の `branch <名前>(<時刻>)` の括弧は tech-debt へ。計画の D4・AC4 の「最初の 20 件」は `git log` が新しい順に出す先頭の 20 行のことで、実装と合う(計画の本文は変えない)
  - self-review(cycle 1 の 2 回目、c9ce0f77): Merge: yes、MEDIUM 1(N1: 「座席の確認」の待ちに出口が無い。座席は手順 13 まで TASK の合間も `(active)` で残るので、前の leader が RESULT を読んだあとに止まっていると来ない RESULT を待つ)、LOW 2(N2 語の範囲、N3 コメントの折り返し)。M1・L1・L2・L4・L6・L7 は閉じた
  - 2a20a1f2(implementer、sonnet、2 回目の直し): 「座席の確認」は leader 自身以外の座席を見て、今回の立て直しに無い `(active)` の座席は `ralph org read --seat` で画面を読む。作業中なら RESULT か BLOCKED を待ち、10 分たっても画面が動かず何も来なければ BLOCKED(reviewer が gate や相手のモデルを走らせているあいだは待つ)。入力待ちなら `history.sh` で、最後の TASK のあとにその座席の RESULT か BLOCKED があれば手順 8 のとおり扱い、無ければ TASK を持たない座席として計画の残りから TASK を送る。手順 3 でも手順 5 でも、座席が持っている役割は spawn し直さない(N1・N2)。gate の項目の BLOCKED は書き下した。N3 はコメントだけ。検証コマンド rc=0、変異 3 個がすべて落ちた。自己点検 clean(テストの 1 か所を python で削ったが U+FFFD は 0 件)。この項より前の 71ddcb30 の項の「座席の確認」の説明は、この直しで「読んでから待つ」形に変わった
  - self-review(cycle 1 の 3 回目、f57e37f4): Merge: yes、N1〜N3 は閉じた。MEDIUM 1(N4: 入力待ちの分岐が gate の項目と逆を指す。gate を通ったあとに leader だけを立て直すと、2 席は入力待ちで残り、座席の確認が TASK を送る)、LOW 1(N5: 見つけた RESULT の扱いを手順 8 に寄せたが、`gate-` と `xreview-` の TASK は手順 12 と cross-review の小節)
  - inline の直し(leader.md と prompts_leader_reseat_test.go の 2 ファイル、文 2 つとテストの数行): 座席の確認の頭に「gate の項目に当たるときは確認の残りをせず、TASK を送らずに gate の項目のとおり止める」、見つけた RESULT は TASK_ID で行き先を分ける(`gate-` は手順 12、`xreview-` は cross-review の小節、それ以外は手順 8)。`-run 'RenderRolePrompt_Leader|ReseatProgress'` が通る。自己点検 clean
  - e3914c93: origin/main(48a8a661、#326 の PR #335)を merge した。衝突なし。build、vet、check-sync、check-skill-sync が通る
  - cross-review(cycle 1、0d660e77): codex の P1 1・P2 3 で、すべて ACTION_REQUIRED(triage は `docs/reports/cross-review-triage-reseat-leader-progress.md`)。(1) leader の TASK は agmsg を通らず manifest の `sent` に残るので、`history.sh` では最後の TASK が分からない、(2) PR があると手順 14 の RESULT が上がらない、(3) 閉じた PR を「ある」と読む、(4) コミット済みの計画から進むと計画助言を飛ばしうる。consult(consult-plan-312c2-xfix)の判定「直してから進める」を handoff に入れた: `sent` の Details の前置き(`raw=true`、`via=screen herdr_state=…`、`submit_unconfirmed=true`)を全部はがしてから読む、RESULT の行に `HEAD:` を、QUESTION の行に `PLAN:` を出し、規則は HEAD と digest で判定する。cycle を 2 にした
  - ec10ba6f(implementer、sonnet): 節に「座席ごとの最後の TASK」(今の命の `sent` から、前置きをはがして TYPE が TASK のもの)と「上げた RESULT」(今の命の RESULT を全状態、`HEAD:` つき、新しい 5 件)を足し、QUESTION の行に `PLAN:` を付けた。leader.md: 座席の確認は最後の TASK の TASK_ID で RESULT と BLOCKED を突き合わせる(無ければ、commit がスライスの完了を示せば次の TASK、示さなければ同じ TASK_ID でもう一度)。gate は `gh pr list --json number,state,url` で状態を読み、OPEN か MERGED なら作らず、TASK_ID が org_id で HEAD が gate の head と同じ RESULT が無ければ手順 14 の RESULT を上げてから手順 15。CLOSED だけなら BLOCKED。計画は per-plan なら今の digest と同じ PLAN の QUESTION、split なら計画の `## Design decisions` の要約で、助言が済んだかを決める。検証コマンド rc=0、golangci-lint 0 issues、変異がすべて落ちた。自己点検 clean(一部を python と heredoc で書いたが U+FFFD は 0 件)
  - self-review(cycle 2、a02c5b5c): Merge: yes、LOW 5。cross-review の 4 件と N4・N5 は閉じた。C2-1(受信箱の節の 1406 行目の「あなたが送った TASK も 50 件に入る」は前からある誤った文で、座席の項目と逆を言う。`prompts_leader_inbox_test.go` が固定しているので tech-debt へ)、C2-2(RESULT の無い入力待ちの出口が implementer のスライスだけを前提にし、レビューや gate の TASK を commit で済んだと見なしうる)、C2-3(最後の TASK の行で TASK_ID に括弧がくっつく)、C2-4(前置きのテストが Send の語を文字列で写す、tech-debt へ)、C2-5(`reseat --help` の列挙が古い、sync-docs へ)
  - inline の直し(C2-2・C2-3、4 ファイル): RESULT の無い入力待ちの座席は、implementer のスライスの TASK なら commit の行で決め、レビューの TASK と `gate-`・`xreview-` の TASK は手順 8・12・cross-review の小節のとおり新しい TASK_ID で頼み直す。最後の TASK の行は `- <座席>: <TASK_ID> (<時刻>)` にした。`-run 'RenderRolePrompt_Leader|ReseatProgress|StartFeature|Reseat'` が通る。自己点検 clean
  - cross-review(3 回目、cycle-count 2・cap 3): codex の P2 が 2 件。(1) d85fc0ef の「ほかの座席にあとで TASK を送っていれば扱い済み」は、片付いていない BLOCKED を捨てうる(知らせが消える向き、ACTION_REQUIRED)。(2) 未コミットの作業を残して止まった implementer のスライスを頼み直すとき `RESUME:` が無い(新しい implementer は BLOCKED を返すので知らせる側、WORTH_CONSIDERING で tech-debt と Known gaps へ)。cycle 2 の「扱い済みの返事を扱い直す」と (1) は逆を向くので、ユーザーの線引き(知らせる側に倒す)で、返事は捨てずに届いたものとして扱い、済んだ手だけ繰り返さない形を選んだ。cycle を 3 にした(cap 3)
  - inline の直し(leader.md と prompts_leader_reseat_test.go): 見つけた返事は前の leader が扱ったかにかかわらず届いたものとして扱い、捨てない。応じる手ごとに済んでいないかを確かめ(計画が active に戻っているか、直しの commit、同じ直しの TASK)、済んだ手だけ繰り返さない。`-run 'RenderRolePrompt_Leader|ReseatProgress|StartFeature|Reseat|RenderRolePrompt|Prompt'` と golangci-lint が通る。自己点検 clean
  - self-review(cycle 3、eaa9bc71): Merge: yes、LOW 2。C3-1(済んだ手の例の「直しの commit があれば頼まない」は、commit の行に時刻も TASK_ID も無く、その返事の直しかを確かめられない)、C3-2(例の一部がテストで固定されていない、時刻の出どころが消えた)
  - inline の直し(C3-1・C3-2、2 ファイル): 「直しの commit」の例を外し、時刻の例に返事の時刻の出どころ(history.sh の `[` と `]` のあいだ)を戻してテストで固定した。外した例が戻らないこともテストで見る。`-run 'RenderRolePrompt|Prompt|ReseatProgress|StartFeature|Reseat'` と golangci-lint が通る。自己点検 clean
  - verify(cycle 3、c693d220): partial-pass、LOW 4(V22〜V25、文書と register)、INFO 2。V26 (b): 時刻の例は、前の leader が返事を読む前に同じ座席へ直しでない TASK を送っていると、それを直しと取り違えて直しを頼まず、来ない RESULT を待ちうる(起きにくいが、知らせが消える側に寄る)
  - inline の直し(V26、2 ファイル): 済んだ手の例を「計画の置き場所」だけにし、済んだかが分からない手(直しの TASK など)は重なっても頼む、と書いた(知らせる側に倒す)。外した 2 つの例が戻らないことをテストで見る。`-run 'RenderRolePrompt|Prompt|ReseatProgress|StartFeature|Reseat'` と golangci-lint が通る。自己点検 clean
  - test(cycle 3、36d80504): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` が rc=0(2026-11-16 05:49〜06:09 JST、cli 602 秒、org 704 秒、全パッケージ ok)。座席の確認の規則(見つけた返事を捨てない、済んだと確かめられる手だけ飛ばす、済んだかが分からない直しは重なっても頼む)のテストと、外した 2 つの例が戻らないことのテストが通った
  - sync-docs(4 回目、cycle 3): V22(仕様の Status に 4 回目の段落と、3 回目の段落の (2) に「4 回目で外した」の 1 句)、V23(行 520 の (e) を、外した枝の縁を落として、新しい規則の縁と `headRefOid` の遅れに書き直し)、V24(新しい行 526: implementer のスライスの送り直しに `RESUME:` が無い)、V25(行 365・426・514・524 の数、行 522 の Trigger の注)を直した。V26 は `91578db5` で直し済みで、縁は行 520 の (e) に書いた。README、AGENTS.md、`/org` と `/director` の skill、`ralph org reseat --help`、org-runtime の仕様、Go のコメントは節の 8 項目の列挙だけで、返事の扱いを述べないので変えていない。Go は変えていない。報告は `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`(Cycle 3 の節)。`/pr` の Known gaps に載せるもの: V16 (a)(b)、V17、行 520 の (e)、`RESUME:` の行(行 526)
  - cross-review(cycle 3、780a7adf): codex の P2 が 2 件で、2 件とも ACTION_REQUIRED。(1) agmsg の team は disband のあとも履歴を残すので、TASK_ID が同じ前の命の RESULT を、行の TASK への返事と取り違えて、済んでいない仕事を済んだと読みうる(知らせが消える向き)。(2) 前の leader の見張りが既読にした座席の QUESTION を見ずに TASK を送り直す(座席はまた聞くので知らせる側に寄るが、(1) と同じ文の 1 句で直せるので直した)。cap は 3 に届いていたので、ユーザーの規則で 4 に上げた(`cycle` は 3 のまま)
  - inline の直し(5ab60578、leader.md と prompts_leader_reseat_test.go): 履歴で見る RESULT と BLOCKED を、時刻がその座席の行の時刻と同じかそれよりあとのものに限った。何かを送る前に(行の TASK_ID が `不明` のときも)、行の時刻からあとのその座席の QUESTION を履歴で探し、あれば DECISION で答えて TASK は送らない。`-run 'Prompt|Reseat'` が通り、`go vet` と gofmt が通る。前の leader.md と、3 つの句を 1 つずつ消した leader.md を `go test -overlay` で当て、どれも新しいテストで落ちた。自己点検 clean(2 ファイルだけ、デバッグの残り・秘密なし)
  - self-review(run 5、cycle-count 3・cap 4、c88c0c26): Merge: yes、LOW 4。C5-1(時刻の条件の 2 文が「あれば」の前に入り、「前のものがあれば今届いたものとして扱う」とも読める)、C5-2(QUESTION の確かめが返事なしの枝に限ると書かれていない、不明の行の枝から `history.sh` のコマンドが見えない、「QUESTION がなければ」が implementer の文にだけかかる)、C5-3(「命」と「life」の混在)、C5-4(`UnknownTaskID` のテストの doc の「reads no history」、新しいテストの末尾の重複)
  - inline の直し(262b9df7、C5-1〜C5-4、2 ファイル): 「数えるのは」「数える返事があれば」「数える返事がなければ」で数える返事を名指しし、前のものは「ないものとして扱います」とした。QUESTION の確かめを「このときは」で返事なしの枝に限り、確かめの文にコマンドを書き、不明の行ではそこで初めて読むと書いた。「QUESTION がなければ、座席の役割で決めます。」「QUESTION もなければ、TASK の種類で決めます。」で後ろの文すべてにかかるようにした。「命」を「life」に。テストの doc と重複を直し、あいまいな形が戻らないことを見る。`-run 'Prompt|Reseat'` と golangci-lint(0 issues)が通る。5ab60578 の leader.md と、「数える返事があれば」を「あれば」に戻した leader.md を `-overlay` で当て、新しいテストで落ちた。自己点検 clean
  - verify(run 5、892703bc): partial-pass。`run-static-verify.sh` が rc=0(golangci-lint 0 issues)。cycle 3 の cross-review の 2 件は規則の文とテストで閉じた。LOW 4(V28 仕様の Status、V29 行 520 の (e)、V30 行 520・522・526 の triage report の位置、V31 数と Trigger)は sync-docs で直す。INFO 3: V32(規則に書かれていない縁 5 つ。(a) は下で直し、(b)〜(e) は行 520 の (e) へ)、V33(テストの doc)、V34(artifact の 3 行の run 5 の分)
  - inline の直し(8a6676e4、V32 (a) と V33、2 ファイル): QUESTION の確かめで、読んだうちでいちばん古い行の時刻がその座席の行の時刻よりあとなら、数を増やして読み直す、と書いた(不明の行の枝もここで読むので、「足りなければ数を増やします」を引き継ぐ)。`NeverDiscardsAReply` の doc に時刻の条件を書いた。`-run 'Prompt|Reseat'` が通る。892703bc の leader.md を `-overlay` で当て、新しいテストで落ちた。自己点検 clean
  - test(run 5、3998e20a): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` が rc=0(2026-11-16 07:27〜07:47 JST、cli 597 秒、org 693 秒、全パッケージ ok)。座席の確認の新しいテスト(`..._ReadsOnlyThisTasksReplies` と、書き換えた `..._SeatCheckItem`、`..._NeverDiscardsAReply`、`..._UnknownTaskID`)が通った
  - sync-docs(5 回目、run 5): V28(仕様の Status に 5 回目の段落、3 回目の段落の (3) と 4 回目の段落の (1) に 1 句)、V29(行 520 の (e) に、時刻の条件と QUESTION の確かめと、V32 (b)〜(e) の縁 4 つ。Why deferred の cap の文)、V30(行 520・522・526 の triage report の位置)、V31(行 365・426・514・524 の数を `8a6676e4` で数え直し、行 522・526 の Trigger に 2 回目の注)を直した。V32 (a) と V33 は `8a6676e4` で直し済み、V34 はこの Progress。README、AGENTS.md、`/org` と `/director` の skill、`ralph org reseat --help`、org-runtime の仕様、Go のコメントは節の項目の列挙だけで返事の扱いを述べないので変えていない。Go は変えていない。報告は `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`(Run 5 の節)。`/pr` の Known gaps に載せるもの: V16 (a)(b)、V17、行 520 の (e)(縁 6 つ)、`RESUME:` の行(行 526)
  - cross-review(run 5、cycle-count 3・cap 4、5528e51c): codex の P2 が 2 件で、2 件とも ACTION_REQUIRED。(1) QUESTION の確かめが、形の違う gate や xreview の TASK を打たなかったと言う reviewer の QUESTION にも DECISION だけを返し、手順 12 と「cross-review」の小節の「TASK を直して送り直す」を飛ばすので、reviewer は TASK を、leader は RESULT を待って止まる。(2) worktree は手元の既定のブランチから作られるが、`resolveGateBase` は origin のものを先に選ぶので、手元の main が origin より進んでいると、その commit を経過と数え、最初の start にも節が入り、計画の項目が関係のない計画を拾いうる(`scripts/ralph-worktree.sh` で確かめた)
  - inline の直し(4096ae88、4 ファイル): (1) reviewer のその QUESTION は手順 12 か「cross-review」の小節の reviewer の QUESTION の項(送り直しの上限と BLOCKED も)へ回し、それ以外の QUESTION だけ DECISION で答える。(2) `reseatProgressCommitsIn` が数えるのを、origin と手元の既定のブランチのどちらにもない commit(`HEAD --not <在る ref>`)にし、計画の項目の `git log` も同じ範囲にした。新しいテスト: `TestReseatProgress_CommitsLeaveOutTheLocalDefaultBranchAheadOfOrigin`、AddsNothing の表に手元の main が進んだ場合、規則の文の固定。前の `reseat_progress.go` と前の leader.md を `-overlay` で当て、新しいテストで落ちた。`-run 'Prompt|Reseat'` と golangci-lint(0 issues)が通る。自己点検 clean。上限の中なので cycle を 4 にした(cap 4)
  - self-review(cycle 4、f4edc533): Merge: yes、LOW 4。C4-1(`HEAD --not …` に `--` が無く、worktree の根に `HEAD` という名前のファイルがあると git が rc 128 で止まり、commit の行が「読めなかった」になる)、C4-2(`reseatProgressSupplied` の doc が古い範囲のまま)、C4-3(doc の折り返しの残りと run の呼び名)、C4-4(「送り直しの上限と BLOCKED も」は言い過ぎで、上限があるのは cross-review の項だけ)。leader.md の手順 9・12・14 に残る `origin/<既定のブランチ>..HEAD` 3 か所は前からある形なので指摘にしていない
  - inline の直し(70b1511c、C4-1〜C4-4、4 ファイル): 改訂の並びの終わりに `--` を足し、`HEAD` という名前のファイルがある worktree でも読めることをテストに足した(前の形では "ambiguous argument" で落ちることを `-overlay` で確かめた)。doc 2 か所と run の呼び名を直し、leader.md の文を「何が違ったのか分からないときの BLOCKED と、cross-review の TASK の送り直しの上限も」にした。`-run 'Prompt|Reseat'` と golangci-lint(0 issues)が通る。自己点検 clean
  - S4(inline、仕様の 1 ファイル): 仕様の FR-9、FR-10、AC-13、AC-14 に印、AC-12 を `[ ]` に戻した(限りの文は残した)。Status に 2026-11-15 の段落。`git diff --check` と check-sync が通る。自己点検 clean
  - verify(12a76fc7): fail。静的検査が 1 件(V1: `prompts_leader_reseat_test.go:94` の staticcheck QF1001)。e483b52a で直した(`if approval >= rule || rule >= steps {`、挙動は同じ)。打ち直し(1a1d8981、inline): `run-static-verify.sh` が rc=0 で、判定は partial-pass。残りは sync-docs の文書の直し(V2〜V6、V9、V10)と、tech-debt に書く V7・V8
  - test(fc8f236a): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` が rc=0(2026-11-16 00:10〜00:32 JST、cli 677 秒、org 770 秒、全パッケージ ok)。opus のサブエージェントが 2 回止まったので orchestrator が inline で回した。Test gaps は 4 件(`StartFeature` を通した commit だけの節、読み取りの失敗の行、実機の動き、coverage の数値)で、はじめの 2 つと実機の動きを tech-debt の行にした
  - sync-docs(1 回目、cycle 1): V2〜V6 と V9 を直した(README の reseat の段落、`/org` と `/director` の skill の 4 つの写しずつ、`docs/specs/2026-08-01-org-runtime.md` の (d) の印と 2026-11-15 改訂、AGENTS.md の `reseat_progress.go`、reseat の仕様の Impact の表)。`ralph org reseat --help` に節の 1 文と、その pin を足した(`ralph org start --help` は触っていない)。tech-debt は、Trigger に当たった 7 行(514、201、365、426、396、513、515)の持ち越しの注と、新しい 3 行(L5 と L3 の残り、`leader.md` の規則の既知の穴、テストの穴)。報告は `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`
  - verify(cycle 2、46b7993c): partial-pass。`run-static-verify.sh` が rc=0(golangci-lint 0 issues)。AC1〜AC6・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は満たす。V11(節の中身を列挙する 14 か所に新しい 2 項目が無い)、V12(コメント 2 か所)、V14(tech-debt の数と状態)は sync-docs の 2 回目で直し、V13(`leader.md` の受信箱の節の誤った文)と C2-4 は tech-debt の新しい行に、V15(`reseat_progress_test.go` が 869 行)は大きさの行にした。V16(規則に書かれていない縁 3 つ)と V17(D6・AC2 の括弧書きが TASK と RESULT を含まない)は `/pr` の Known gaps に載せる(計画の本文は変えない)
  - test(cycle 2、043dbe59): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` が rc=0(2026-11-16 02:26〜02:48 JST、cli 654 秒、org 761 秒、全パッケージ ok)。ec10ba6f と b8a9bd38 の新しいテストが通った。Test gaps: 規則の新しい分岐(PR の状態、RESULT の有無、計画助言の有無)を実機の leader がどう読むかは確かめていない(cycle 1 の「実機の動き」と同じ)
  - sync-docs(2 回目、cycle 2): V11(README、AGENTS.md、reseat の仕様の FR-9 に日付つきの追記と Status の段落、org-runtime の仕様の 2026-11-15 改訂 (a)、`/org` と `/director` の skill の 4 つの写しずつ、`ralph org reseat --help` とその pin)と V12(Go のコメント 2 か所)を直した。tech-debt は、行 519 の数と (d)、行 520 の (a)〜(c) を済みに、行 365・426・514 の数、行 201・396・513・515 に 2 回目の注、新しい 3 行(V13、C2-4、V15)。行 465 は RESOLVED の行なので注を足していない。報告は `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`(Cycle 2 の節)
  - cross-review(cycle 2、51119734): codex の P1 1・P2 2 で、3 件とも ACTION_REQUIRED(triage は `docs/reports/cross-review-triage-reseat-leader-progress.md`)。(1) gate の項目が PR の状態だけを見て、gate を通したコミットが push されていない OPEN の PR を完了と報告する、(2) 座席の確認が、前の leader が扱った返事を新しく届いたものとして扱い直す、(3) `truncateForDetails` が切った TASK の TASK_ID が `-` や切れ目の印つきで出る。cap は 2 に届いていたので、ユーザーの規則(上限に届いたら上げて回し直す)に従って 3 に上げた(`cycle-count.json` の `cap` を 3 にし、`cycle` は 2 のまま)
  - d85fc0ef(3 件の直し): gate の項目が PR の `headRefOid` を gate の head と比べる(head が同じ OPEN か MERGED なら PR は作らない。head が違う OPEN には手順 14 と同じ push のあと一覧を読み直す。head が違う MERGED と CLOSED だけの場合は BLOCKED)。座席の確認は、返事より後の時刻の最後の TASK の行があれば扱い済みとして読み直さない。TASK_ID のところで切れた TASK の行は `- <座席>: 不明 (記録が切れている、<時刻>)` にして、規則は推し量らず新しい TASK_ID で送る。`leader.md`、`reseat_progress.go` と、そのテスト 2 つの 4 ファイル
  - self-review(cycle 2 の run 3、9ea00d76): Merge: yes、LOW 2(R3-1: tech-debt の行の数と状態が `d85fc0ef` のあとの状態でない、R3-2: `reseatProgressUnknownTaskID` と `TaskIDUnknown` の名前、定数の doc の「the marker」、テストの名前)。3 件の ACTION_REQUIRED は閉じ、直しが持ち込んだ欠陥は見つからなかった
  - verify(run 3、c49a6bc5): partial-pass。`run-static-verify.sh` が rc=0(golangci-lint 0 issues)。AC1〜AC6・AC8・AC9 と仕様 FR-9・FR-10・AC-13・AC-14 は `d85fc0ef` を含めて満たす。V18(tech-debt の数と Trigger)と V19(仕様の Status と FR-9 の追記)は sync-docs の 3 回目で直し、V20(この Progress の追記)もその回で直した。V21(規則に書かれていない縁 3 つ)は tech-debt 行 520 の (e) にした。V16・V17・V21 は `/pr` の Known gaps に載せる(計画の本文は変えない)
  - test(run 3、601b4643): pass。`GOFLAGS=-timeout=60m ./scripts/run-test.sh` が rc=0(2026-11-16 04:22〜04:42 JST、cli 601 秒、org 708 秒、全パッケージ ok)。`d85fc0ef` の新しいテスト(200 byte で切れた TASK_ID を不明にする場合、gate の項目の PR の head の分岐、扱い済みの返事を読み飛ばす条件、不明の TASK_ID の扱い)が通った
  - sync-docs(3 回目、cycle 2 の run 3): 行 519(数と状態)、520((a)(b) の RESOLVED に PR の head の 1 句と、(e))、522・523・524(`d85fc0ef` が Trigger に当たったが払っていない注。524 は 3 つのファイルの大きさと cap)、365・426・514(数)を直し、R3-2 を新しい行 525 にした。仕様の Status に 2026-11-16 の 3 回目の段落、FR-9 の追記に `不明` を足した。FR-10 は gate の PR の分岐の細かさを書いていないので変えていない。README、AGENTS.md、`/org` と `/director` の skill、`ralph org reseat --help`、org-runtime の仕様、Go のコメントは `d85fc0ef` の 3 点を述べておらず、偽の文はない。Go は変えていない。報告は `docs/reports/sync-docs-2026-11-16-reseat-leader-progress.md`(Run 3 の節)
- [x] Review artifact created — self-review は 3 回の run(64c7f828、c9ce0f77、f57e37f4)。最後の run は Merge: yes で、N4・N5 は 291ac942 で直した。cycle 2 は a02c5b5c(Merge: yes、LOW 5)、run 3 は 9ea00d76(Merge: yes、LOW 2)、cycle 3 は eaa9bc71(Merge: yes、LOW 2)、run 5 は c88c0c26(Merge: yes、LOW 4)
- [x] Verification artifact created — verify は fail から partial-pass(12a76fc7、打ち直しは 1a1d8981)。V2〜V6 と V9 は sync-docs で直し、V7・V8・V10 は tech-debt の行にした。cycle 2 は 46b7993c(partial-pass、V11〜V17)、run 3 は c49a6bc5(partial-pass、V18〜V21)、cycle 3 は c693d220(partial-pass、V22〜V27)、run 5 は 892703bc(partial-pass、V28〜V34)
- [x] Test artifact created — test は pass(fc8f236a)。cycle 2 も pass(043dbe59)、run 3 も pass(601b4643)、cycle 3 も pass(36d80504)、run 5 も pass(3998e20a)
- [ ] PR created
