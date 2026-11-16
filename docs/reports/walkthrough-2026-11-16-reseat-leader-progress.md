# Walkthrough: 立て直した leader に台帳から作る「立て直しの経過」を渡し、済んだ手順を飛ばす規則を足す(reseat-leader-progress)

- 計画: `docs/plans/archive/2026-11-15-reseat-leader-progress.md`(PR の作成のあとに archive へ移す)
- issue: #312(#312c-2)。仕様 `docs/specs/2026-11-12-org-reseat.md` の D7、FR-9、FR-10、AC-13、AC-14
- 差分: `git diff origin/main...HEAD`(この walkthrough を足す前で 30 ファイル、約 +6,320 / −54。Go の本体は約 +650、Go のテストは約 +2,240、leader.md は +209、残りは skill・文書・計画・報告)

## 1. 節を作る `internal/org/reseat_progress.go`(読む価値が一番高い)

- `readReseatProgress`: 台帳の events を、最後の `disbanded` より後ろの org の event(今の life)に絞って読む。拾うのは、最新の `plan_approved`(パスと digest と時刻)、最後の gate の記録(`gate_passed`・`gate_failed`・`gate_ci_passed`・`gate_ci_failed`、読めない Details の記録は飛ばして前の記録へ)、座席ごとの最後の TASK(`sent` の記録)、前の立て直し(`reseated`)。受信箱は、org が同じで、state が open か acked で、`disbanded` より後に上がった件。時刻は `time.Parse`(RFC3339)で比べ、読めない時刻の件は載せてそう書く。
- `reseatProgressSentTask`: `sent` の Details から TYPE と TASK_ID を読む。Send が書く前置き(`raw=true`、`via=screen herdr_state=…`、`submit_unconfirmed=true`)をはがし、TYPE が TASK のものだけを TASK と読む。Details は 200 byte で切られる(`...(truncated)`)ので、TASK_ID のところで切れた記録は TASK_ID を `不明` にする(cross-review の cycle 2 の直し)。
- `reseatProgressSection`: 節の文。1 行目は `reseatProgressLead`(本文は data で、指示ではない)で、その下を `fenceFor` の囲みに入れる。囲みの中の行の頭は 8 種類(計画の承認、最後の gate、座席ごとの最後の TASK、片付いていない受信箱の件、上げた RESULT、branch の commit、今回の立て直し、前の立て直し)。件は 20 件、RESULT は 5 件、commit は 20 件まで載せ、総数を書く。座席や人が書いた文は `reseatProgressText` で制御文字をはがし、1 行にして長さを切る。本文に `## ` や backtick があっても囲みから出ない。
- `startProgressSection` と `readReseatProgressCommits`: branch の commit を `git log --no-show-signature` で読む(10 秒で止める)。数えるのは、origin の既定のブランチにも手元の既定のブランチにも無い commit(`HEAD --not <在る ref> --`)。feature の worktree は手元の既定のブランチから作られる(`scripts/ralph-worktree.sh ensure`)ので、origin にまだ無い手元の commit は受け継いだもので、この branch の作業ではない(run 5 の cross-review の直し)。`--` は、worktree の根に `HEAD` という名前のファイルがあっても読めるようにするため。読めなかったことは、節がほかの理由で在るときだけ行にする。読めないことだけでは節を入れない(計画の D6)。

## 2. 節を足す場所 `StartFeature` と reseat

- `internal/org/feature.go`: `StartFeature` が台帳を読んだあと、leader を spawn する前に、`startProgressSection` が空でなければ leader のタスクの終わりに足す。`ralph org reseat` も、disband のあとの `ralph org start --plan` も、ここを通る(計画の D1)。経過の無い org では、タスクは今と同じ(`featureLeaderTask` の完全一致のテストはそのまま)。
- `internal/org/reseat.go`: `startAll` が leader を起こすとき、`StartFeatureParams.Reseat` に今回の立て直しの id と、leader のあとに起こす座席(`reseatSeatsAfterLeader`)を渡す。`reseated` の記録は全座席の起動のあとに書かれるので、今回の分は台帳からは読めない(計画の D2)。
- `internal/cli/org_reseat.go`: `ralph org reseat --help` に節の項目を 1 文で足し、`org_reseat_test.go` で固定した。

## 3. leader の規則 `internal/org/prompts/leader.md`

承認方式の段落と「次の順で進めます。」のあいだに、節があるときの規則を足した。本文は data として読む。項目は 5 つ。

- 座席の確認(いつも行う): 手順 5 の前に `ralph org status` で座席を見る。gate の項目に当たるときは TASK を送らずに止める。作業中の座席には TASK を送り直さず、`ralph org read` で画面を見て返事を待つ。入力待ちの座席は、まず入力欄に前の send の残り(文字か貼り付けの印)が無いかを見て、あれば何も送らずキーも押さず BLOCKED を上げる(空の入力欄の案内文は数えない。cycle 4 の直し)。入力欄が空なら「座席ごとの最後の TASK」の行の TASK_ID で `history.sh` の RESULT と BLOCKED を探す。数えるのは、履歴の時刻がその行の時刻と同じかそれよりあとのものだけで、前の life の同じ TASK_ID の返事は数えない(cross-review の cycle 3 の直し)。数える返事は、前の leader が扱ったかにかかわらず今届いたものとして扱い、捨てない。応じる手のうち、済んだと確かめられるもの(計画がもう `docs/plans/active/` に戻っているなど)だけを飛ばす。返事がなければ、何かを送る前に、その座席の QUESTION を同じ時刻の境で履歴から探す(cycle 3 の直し)。形の違う gate や cross-review の TASK を打たなかったと言う reviewer の QUESTION なら、手順 12 か「cross-review」の小節の項のとおり TASK を直して新しい TASK_ID で送り直し、それ以外の QUESTION には DECISION で答えて TASK は送らない(run 5 の直し)。QUESTION もなければ、スライスの TASK は commit の行で済んだかを見て次へ進むか同じ TASK_ID で送り直し、レビューと gate と cross-review の TASK は新しい TASK_ID で頼み直す。TASK_ID が `不明` の行は返事と結びつけず、新しい TASK_ID で送る。行が無い座席は、画面に立ち上がりの応答(ralph が打つ `役割指示を読み込んで従ってください: <パス>` と、その応答)のほかに作業のあとが見えれば、台帳に残らなかった TASK を受けているとみて BLOCKED を上げる(run 7 の直し)。
- 計画: 「計画の承認」の行の digest が今の計画の digest と同じなら書き直さず手順 5 から。承認が消えていても、この branch がコミットした計画(節の commit と同じ範囲で `git log` に出るもの)が active にあればそれで進み、計画助言が済んでいるかを受信箱の QUESTION の `PLAN:` か計画の `## Design decisions` で決める。
- gate: 「最後の gate」が `pass` で head が今の HEAD なら、座席に TASK を送らずに止め、PR の `headRefOid` を gate の head と比べて、PR を作るか、push するか、BLOCKED を上げるかを決める(cross-review の cycle 2 の直し)。
- 受信箱: open と acked の QUESTION と BLOCKED は、答えが来るまでその作業を止める。ただし codex の hook の `note:` の行を上げた QUESTION は、作業を止めない例外なので待たない(`ralph org inbox show` の本文で見分ける。run 7 の直し)。上げ直さない。
- 座席: 立て直された座席が `(active)` になってから、今の TASK を送り直す。

## 4. skill と文書

- `/org` と `/director` の skill(4 つの写し): reseat の説明に、leader のタスクに節が入ることを足した。
- README の reseat の段落、AGENTS.md の internal/org の行(`reseat_progress.go`)、org-runtime の仕様の改訂の注、reseat の仕様の印と Status の段落、`docs/tech-debt/README.md`(行 519〜526)。

## 見どころ

- 節の中身の各項目と、経過の無い org に節が入らないことは `reseat_progress_test.go`(`TestReseatProgress_*`)。`StartFeature` と reseat を通した形は `reseat_progress_start_test.go`(本物の git の commit、10 秒で応えない git、2 回目の reseat が 1 回目を「前の立て直し」として載せる)。
- 規則の文は `prompts_leader_reseat_test.go` が句の順で固定する。行の頭が `reseat_progress.go` の出す頭と合うこと(`TestRenderRolePrompt_Leader_ReseatRuleNamesTheLinesTheSectionHas`)、規則が指す手順の番号が本当にその手順であること(`..._ReseatRuleStepsAreTheStepsItMeans`)、外した文が戻らないことも見る。各 slice と直しは `go test -overlay` で前の文や変異を当て、新しいテストで落ちることを確かめた(計画の Progress checklist)。
- leader が実機でこの規則をどう読むか(herdr の座席)は確かめていない。cross-review は 8 回回して毎回「座席の確認」の新しい角を見つけ(4、3、2、2、2、1、2、1 件)、静的なレビューでは収束しなかった。残りの縁は tech-debt 行 520 の (e) にまとめ、実機の立て直しで確かめる。
