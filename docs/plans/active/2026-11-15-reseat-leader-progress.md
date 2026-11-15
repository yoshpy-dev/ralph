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
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
