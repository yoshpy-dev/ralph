# Self-review report: org-xreview-own-model

## Run 2 (cycle 2)

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 2 回目(`cycle-count.json` は cycle 2)。Run 1 は下にそのまま残す。今の判定はこの節の Recommendation の行
- Scope: Run 1 のあと(`66c7820e..HEAD`)にこのブランチ自身が足したコミットのうち、中身のある 4 本(HEAD `eff83438`)。`f95a08f9`(Run 1 の MEDIUM 1 件と LOW 3 件の直し。leader と reviewer の雛形、org と director の skill 4 写しずつ、テスト 4 ファイル)、`7c7cdec5`(`ralph init` の問いの説明とコメント)、`0d44799f`(sync-docs の文書とコメントの直し)、`eff83438`(cross-review の ACTION_REQUIRED の直し。reviewer の雛形の手順 2 とそのテスト)。残りの 4 本(`7a1dd7ef`、`a30b6097`、`82090f46`、`0cd1a32f`)は report と insight event だけ。`66c7820e..HEAD` には main から取り込んだコミット(`b98b1b96` と `6b671276` の側、#362・#363・#366・#370)も並ぶが、このブランチの差分ではないので見ていない。テストと静的解析は走らせていない

### Run 1 の 5 件の行き先

番号は Run 1 の Findings の表の順(MEDIUM の 1 件は M1、LOW は表の順に L1 から L4)。`f95a08f9` のテストのコメントが書く「self-review M1」「L1」「L3」「L4」はこの番号。Run 1 の Follow-ups の「LOW 5」は表の行の通し番号で数えた書き間違いで、L4 のこと。

| Run 1 | 内容 | 結果 | 確かめたこと |
| --- | --- | --- | --- |
| M1 | 待ちの例外に上限がない | 直った | 5 つの面が同じ形になった。動いているあいだは待つ、10 分動かず RESULT も BLOCKED も来なければ座席の名前を書いた BLOCKED を上げる、動かなくても待つのは gate のときだけ。面は `leader.md:189-194`(立て直しの座席の確かめ)、`:774-782`(手順 12)、`:1269-1273`(cross-review の小節)、org の skill 項目 4(`:1540-1547`)、director の skill(`:1091-1096`)。org と director の skill は 4 写しとも同一(`sha256` で比べた)。10 分は `leader.md:189` の既存の規則と同じ数。見張りの stall は、manifest の最後の event と herdr の `state_change_seq` が `stall_minutes`(既定 15)動かないことで、画面の動きは測らない。そのため「ALERT のあとに画面を読み、動いていれば待つ」と矛盾しない。旧版の例外の語は、雛形と skill に残っていない(grep)。テストは旧語を `xreviewStallExempt` で探し、上限を `xreviewStallStill` で固定している。旧名のテスト 3 本への参照も残っていない |
| L1 | `<相手>` の名前 | 直った(名前) | `ran (<report の Reviewer:>)` に、leader、org と director の skill、テストの固定がそろった。`<相手>` と `<その report の Reviewer:>` の残りは、テストの「消した語」の一覧と `leader.md:955` の `command -v <相手>`(計画助言の別の語)だけ。model が `""` の座席で括弧が重なる件(`ModelOrCLIDefault` が `(CLI default)` を返し、`ran (reviewer 座席 reviewer(codex、(CLI default)))` になる)は、Run 1 が任意の提案にしたとおり未対応。見た目だけで、model が空の org でだけ出る |
| L2 | 古いコメントと register の行 | 半分だけ直った | コメント(`org_runtime_env_test.go:182`)と `pick_reviewer` の行(`tech-debt/README.md:31`)は `0d44799f` で直った。同じ文を言う `t.Fatalf` の文字列 2 つが残った。C2-1 |
| L3 | fan-out の子の書き込み | 直った(手順 3 に 1 文)。範囲と理由が狭い | `reviewer.md:265-269`。C2-2 |
| L4 | 文の組み立てと折り返し | 直った | (a) director の文は 2 文に分かれた(「incomplete は…に出る。off でないときの skipped は…」)。(b) 同じ語の繰り返しは「画面が動いているあいだは待つ」の形に変わった。(c) org の skill の `XREVIEW: cap_reached` の行は折り返しが直った。`f95a08f9` と `eff83438` の追加行 69 行(leader、reviewer、skill 2 つ)の表示幅は、ファイルごとの中央値が 73 から 76、最大 80 |

### `eff83438` の手順の流れ

reviewer の雛形の「cross-review の TASK」(`reviewer.md:224-249`、`:297-315`)を、2 の `diff: failed` から通して読んだ。

- 2 の `diff: failed` は 3 と 4 を飛ばして 5 に進む。5 のコマンドが比べる `-git-before.txt` は、2 の 1 行目のブロックが `git diff` より前に書くので、`git diff` が失敗した道にも比べる相手がある
- 5 の `git: changed` は、report を書かずに BLOCKED(本文に `XREVIEW: git_changed`、EVIDENCE に 2 つのファイル)。7 の末尾の括弧(`:368-370`)と、leader の `XREVIEW: git_changed` の項(`leader.md:1302`、「reviewer が読む前と report を書く前で git の状態が違い」)と合う。leader の文は 2 から来た道にもそのまま当てはまる
- 5 の `git: same` は「6 に進む」(`:315`)。2 の項は、6 で `Reviewer status: incomplete (git diff failed: <要点>)` を書き、7 で `XREVIEW: incomplete (git diff failed: <要点>)` の RESULT を返す。3 の「読み終えられなければ」の道(5、6、7、`:270-273`)と同じ形。6 の `Total reviewer findings: <3 で出した所見の数>` は、同じ手順の「incomplete のときも `Total reviewer findings: 0` … を残す」が覆う。7 の `XREVIEW: incomplete (<理由>)` も 2 の道を覆う。leader の `XREVIEW: incomplete` の扱い(この回は上限に数えず、頼み直さない。`leader.md:1288-1290`)は変わらない
- `diff: failed` から incomplete を書く道を述べる面は、雛形とテストだけ(grep。plan、reports、evidence を除く)。skill、README、spec に写しはない
- テストは `diff: failed` の項を 3 つの語句(「3 と 4 を飛ばして 5 に進み」、`git: changed`、`git: same`)に分けて順に固定し、コマンドの 3 行は行単位で固定している。コメントは、この直しの理由を自分の文で言っている

### `7c7cdec5`(`ralph init` の問い)

- 説明の 2 文目と `ChangeReview` のコメントが言うことは成り立つ。問いは `bothOnly: true` で CLI が 2 つのときだけ出て、答えは `[standard].cross_review` と `[org.review].cross_review` の両方に書かれる。説明の行の幅は 75 で、同じファイルの他の説明(最大 73)より 2 桁長いだけ
- 表題が「もう一方の AI」のままで org の分を言い分けられないことは、`docs/tech-debt/README.md:556` の新しい行が開いた項目として持つ。この差分の範囲ではない。行が名指す 3 つのテストファイルと spec の「2026-11-02 に決めたこと」は HEAD にある

## Findings (Run 2)

番号は C2-n。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| C2-1 | LOW | maintainability | Run 1 の L2 が半分だけ直った。コメントは直ったが、同じ文を言う失敗メッセージの文字列が 2 つ残った。どちらも、警告に `a seat's cross-review` の clause が入っていないことを確かめる assertion のメッセージで、理由として「cross-review は TASK から implementer の driver を受ける」を書く。この branch でその受け取りは消えたので、テストが落ちたときに読む説明が前の版の仕組みを指す。台帳にも sync-docs の report にも載っていない(sync-docs の report は「コメントだけ」と書く) | `internal/cli/org_runtime_env_test.go:259` と `internal/org/driver/herdr_test.go:588` の `want no clause about a seat's cross-review, which takes the implementer's driver from its TASK`。同じテストのコメント(`org_runtime_env_test.go:182-186`)は `0d44799f`、`herdr_test.go` のコメントは `7d124a16` で直っている。`git grep "takes the implementer's driver"` は HEAD でこの 2 行だけ | 2 つの文字列を、直したコメントと同じ言い方(a seat's cross-review, which reviews on the seat's own driver and model)にする。Go の文字列だけの変更で、assertion の判定は変わらない |
| C2-2 | LOW | readability | Run 1 の L3 の直し(手順 3 の 1 文)は、理由が文の半分しか支えず、置き場所も狭い。(a) 「`.harness/state/` にも置かせず」と言い、理由は「git が無視しないファイルが worktree にできると、5 の比べで `git: changed` になる」。`.harness/state/` は無視されるので(`.gitignore:50`。`git check-ignore -v` で確かめた)、子がそこに置いても `git: changed` にならず、座席自身も 2 と 5 のコマンドでそこに書く。理由の外の禁止は、座席から見ると何が悪いのか分からない。テストのコメントも同じ形(「not even in .harness/state/, since a file git does not ignore …」)。(b) 文は手順 3 の箇条で、書き出しは「所見を出すのを子サブエージェントに分けるとき」。「座席内 fan-out」の節は、所見と triage のどちらも子に分けてよいと言う。triage(手順 4)を子に分けた座席は、比べの区間(2 から 5)の中にいるのに、この文を読まない | `reviewer.md:265-269`(手順 3 の箇条)、`:469-475`(fan-out の節の「レビューの判断(…cross-review の TASK で出す所見と triage)」)、`prompts_xreview_reviewer_test.go:335`(コメント)と、同じ手順 3 のテストの固定(2 行) | 文を fan-out の節の「子を座席と同じモデルで動かす」規則の隣に移し、区間を「cross-review の TASK の 2 から 5 のあいだ」と言う。`.harness/state/` を禁じる理由が別にあるならそれを書き、なければ「にも」を外す。手順 3 の箇条は、その節を指す 1 行にする。テストの固定は 1 か所の移動 |
| C2-3 | LOW | readability | `eff83438` で手順 5 の入り口が 3 つ(4 から、3 の読めない道から、2 の `diff: failed` から)になったが、5 の `git: changed` の括弧は、読んだあとの道の理由だけを言う。(a) 「読んでいるあいだに HEAD か作業ツリーが動いたので、所見は今の branch を見ていない」。2 から来た道では何も読んでおらず、所見もない。(b) 2 の `git: same` の項の「出力に出た `git diff` の stderr の要点」は、5 のコマンドを打ったあとに読む。5 の出力は `git: same` の 1 行だけなので、どの出力かが文だけでは分からない。手順の流れ自体は上のとおり食い違わず、座席は書かれたとおりに動く | `reviewer.md:308-310`(5 の `git: changed` の括弧)、`:243-246`(2 の `git: same` の項)。テストの固定は `prompts_xreview_reviewer_test.go:304-309` | (a) 括弧を「(所見を出していたなら、それは今の branch を見ていない)」の形にする。(b) 「2 の出力に出た」と書く。雛形の 2 か所の語とテストの固定 2 行 |

## Positive notes (Run 2)

- Run 1 の 5 件のうち M1、L1、L4 は、雛形、skill 4 写し、テストが同じ向きに動いた。直したあとの文書に旧版の言い方が残っていない(grep)。テストは旧語の一覧を持ち、残してよい文だけが通る
- `eff83438` の直しは、triage の「Affected file(s)」が名指した 2 ファイル(`reviewer.md` の 2 と、そのテスト)に収まり、triage が提案した形(「3 と 4 を飛ばして 5 に進み、`git: same` なら 6 で incomplete を書いて 7 のとおり返す」)のとおり
- register の 343 行目(Codex の reviewer と `ralph org send`)の 2026-11-22 の注は、R1 の決まり(動いているあいだは待つ、10 分で BLOCKED、gate だけ動かなくても待つ)と同じことを言う。`pick_reviewer` の行(31)は、4 つ目の呼び出し元が消えたという書き方に変わった
- 差分に秘密、デバッグの残り、範囲外のファイルは見えなかった(実装側の確かめの担当。目に入った範囲)

## Tech debt identified (Run 2)

C2-1 から C2-3 と、L1 の括弧の重なりを今回の周で直さないなら、cycle の上限に達した最後の周なので、直すつもりの所見も繰り越しとして台帳に 1 行要る。この周のコミットは report、plan の Progress、insight event に限るよう頼まれているので、`docs/tech-debt/README.md` にはこの周では足していない。`/pr` の時点で足す行の案:

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Wording leftovers of fix/org-xreview-own-model (#360), found by its self-review cycle 2 (C2-1 to C2-3) and cycle 1 (L1): (1) the `t.Fatalf` messages at `internal/cli/org_runtime_env_test.go:259` and `internal/org/driver/herdr_test.go:588` still say a seat's cross-review "takes the implementer's driver from its TASK"; (2) the sentence that keeps a fan-out child from writing in the worktree (step 3 of the section for the cross-review TASK in `internal/org/prompts/reviewer.md`) forbids `.harness/state/` for a reason that covers only files git does not ignore, and sits in step 3 although the fan-out section lets the seat split the triage of step 4 too; (3) the parenthesis of step 5's `git: changed` explains a change "while reading", which the path from a failed `git diff` (step 2) does not have; (4) `Cross-review: ran (...)` nests three parentheses when the reviewer's model is `""` (`ModelOrCLIDefault`) | LOW. Wording only; the assertions and the steps behave as written | Cycle cap reached | The next change to the section for the cross-review TASK in `reviewer.md`, or to the two tests | `docs/plans/active/2026-11-22-org-xreview-own-model.md`; this report (Run 2, C2-1 to C2-3; Run 1, L1) |

## Recommendation (Run 2)

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 3)
- Follow-ups:
  - C2-1 は 2 つの文字列の直しで、assertion の判定は変わらない。C2-2 と C2-3 は `reviewer.md` の語とテストの固定。どれもコードの振る舞いを変えない。ただし Go のテストと雛形に触れるので記録だけの修正ではなく、直したら `/verify` 以降の再実行が要る。この周の中で直す分は cycle の上限に数えない
  - 直さずに `/pr` に進むなら、上の台帳の行を `/pr` で足す
  - `eff83438` の手順の流れ、R1 の待ちの決まり(5 面)、`7c7cdec5` に直すものは見つからなかった
  - Run 1 の Follow-ups の「LOW 5」は L4 のこと(上の行き先の表)

## Run 1 (cycle 1)

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-22-org-xreview-own-model.md
- Branch: fix/org-xreview-own-model(HEAD c8cbe001、基準 main 10e9322d)
- Reviewer: reviewer subagent (Claude)
- Scope: 差分の質だけ(命名、読みやすさ、雛形の手順の正しさと抜け、leader と reviewer の雛形の食い違い、skill の写しと雛形の食い違い、テストが直しを外すと落ちるか、消した文が残っていないか)。仕様の適合、テストの網羅、文書のずれは /verify と /test と /sync-docs の担当。テストと静的解析は走らせていない(pipeline 周回 1)

## Evidence reviewed

- `git diff 10e9322d...HEAD --stat`: 34 ファイル、+1761 / -1738。コミットは 21df9473(S1)、09367878(S2)、7e4b33b9(S3)、7d124a16(S4)と記録の 4 本
- `internal/org/prompts/reviewer.md` 全文(新版)と、main 版の「cross-review の TASK」の節
- `internal/org/prompts/leader.md` の差分(「cross-review」の小節、手順 12 と 14、立て直しの規則、台帳を読む節)と、小節の新版の全文
- `internal/org/prompts.go`、`spawn.go`(本物と dry run の 2 か所)、`role_models.go` の `ModelOrCLIDefault`、`reseat.go:920`
- テスト: `prompts_xreview_reviewer_test.go`(1〜650 行、650〜905 行、1030〜1092 行)、`prompts_xreview_leader_test.go`(20〜75 行、395〜545 行)、`skill_xreview_test.go` の差分、`prompts_test.go`、`spawn_role_models_test.go`、`prompts_leader_reseat_test.go` の差分
- org と director の skill(`.claude/skills/` の差分。`cmp` で `.agents/skills/`、`templates/base/` の 3 写しと同一)、`model-routing.md`、README、spec の 2026-11-22 の項、2 つの recipe(`cmp` で template の写しと同一)、`config.go`、`paneenv.go`、`templates/base/ralph.toml`、`scripts/xreview-helpers.sh`(`#` で始まらない変更行は 0)、`docs/tech-debt/README.md` の差分
- `git grep` で `IMPLEMENTER_DRIVER|REVIEWER_CODEX|REVIEWER_CLAUDE`、`pick_reviewer`、「implementer と違う系列」の残りを探した(アーカイブ、reports、evidence、spec の履歴、tech-debt を除く)
- 追加行の表示幅(全角 2 桁)を `leader.md`、`reviewer.md`、skill、recipe で測った
- 走らせていないもの: テスト、静的解析、`check-sync.sh`

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | 「reviewer が読んでいるあいだは、画面が動かなくても待つ」の例外だけが 4 面に残り、待つ上限がない。旧版の例外は、他モデルの呼び出しが静かに最大 20 分続き、`sleep 1200` の見張りで切れる、という前提の上にあった。新版の reviewer は自分で差分を読むだけで、見張りがなく、読んでいれば画面は動く。例外の理由が消えたのに、10 分の BLOCKED の規則を外す「画面が動かなくても」が残っている。固まった reviewer(止まった tool call など)を、leader は 10 分の規則で拾わず、director の ALERT の項も「読んでいれば待つ」で人へ上げない | `internal/org/prompts/leader.md:177-180`(「cross-review の差分を読んでいるときは、手順 12 と 9 のとおり、画面が動かなくても待ちます」)、`leader.md:760-762`(手順 12)、`.claude/skills/org/SKILL.md:1505`、`.claude/skills/director/SKILL.md:1034`(それぞれ 4 写し)。固定するテストは `skill_xreview_test.go:60` の `xreviewStallWait` と `prompts_leader_reseat_test.go:308`。旧版の上限は main の `leader.md` 手順 12 の「呼び出しは 20 分まで続くことがあり」 | 例外を「画面が動いているあいだは待つ」に直し、「動かなくても」を外す。10 分の規則は cross-review の TASK にもそのまま掛ける。gate と同じく上限なしと決めるなら、その旨と、止めるのは誰かを書く。直すなら 4 面とテストの固定を一緒に直す |
| LOW | naming | 値の名前が `<相手>` のまま。`Cross-review: ran (<相手>)` の値は report の `Reviewer:` の行で、`reviewer 座席 reviewer(codex、gpt-6.1-sol)` という座席の名前になった。別のモデルを指す語ではない。3 つの面は「`<相手>` は report の `Reviewer:` の値」と注釈を足して名前を残している。同じ値を `leader.md` の別の所は `<その report の Reviewer:>` と書く。model が空の座席では `ran (reviewer 座席 reviewer(codex、(CLI default)))` と括弧が 3 重になる | `leader.md:856`、`:1263-1264`、`:1277`、`:1302`。`.claude/skills/org/SKILL.md:1602`、`.claude/skills/director/SKILL.md:984`。固定: `prompts_xreview_leader_test.go` の `HandlesEveryReply`、`skill_xreview_test.go` の `ResultItemReportsTheCrossReviewLine`。`ModelOrCLIDefault` は `role_models.go:18` で `(CLI default)` | `<report の Reviewer:>` に名前をそろえ、注釈を消す(テストの文字列も合わせる)。括弧の重なりは、model が空のとき `CLI の既定` と書けば避けられる。計画が「`Cross-review:` の決め方は今のまま」と決めているので、直す・残すは担当側の判断 |
| LOW | maintainability | 古くなったコメントと register の行が残っている。`org_runtime_env_test.go` のコメントは「reviewer 座席の cross-review は TASK から implementer の driver を受ける」と言うが、この差分でその受け取りは消えた。同じ言い回しの 3 か所(`herdr_test.go`、`paneenv.go`、`spawn.go`)は今回直したのに、ここだけ残った。`docs/tech-debt/README.md` の `pick_reviewer` の行も、4 つ目の呼び出し元(reviewer 座席が `ralph org status --json` の driver を渡す)を説明したままで、呼び出し元はもうない。同じ差分は register の別の 2 行(Seven test gaps、has not run inside a seat)を直している。計画の Progress の S4 は「/sync-docs に回す」と書いているので、意図した残し | `internal/cli/org_runtime_env_test.go:182-185`、`docs/tech-debt/README.md:31` | 2 つとも、この差分で直すほうが安い(コメントは 1 文、行は「4 つ目の呼び出し元」の文の削除)。/sync-docs に回すなら、そこで漏れないことを確かめる |
| LOW | maintainability | step 3 の「読むあいだは worktree を書き換えず、git の状態を変えない」は座席自身に向けた文で、新しい fan-out の決まりは判断を子に分けさせるのに、子も同じ区間(step 2 から 5)は書かない、とは言わない。子が worktree の `.harness/state/` の外に作業ファイルを作ると、step 5 の `git ls-files --others --exclude-standard` が拾って `git: changed` になり、report なしの BLOCKED が人まで上がる。ユーザーの決定で子に判断を分ける形になった以上、この衝突は今回の変更が作る | `reviewer.md:260-261`(step 3 の箇条)、`reviewer.md:461-469`(fan-out の新しい決まり)、step 5 のコマンド `reviewer.md:296` | fan-out の節か step 3 に「子にも、worktree に書かない、作業ファイルは `.harness/state/` か `$TMPDIR` に置く、と伝える」の 1 文を足す。`FanOutKeepsTheReviewOnTheSeatsModel` に 1 行の固定を足す |
| LOW | readability | 文の組み立てと折り返し。(a) director の skill: 「incomplete は、reviewer 座席が差分を読み終えなかった回(...)で、off のほかの skipped は、ほかのモデルを呼んでいた前の版の reviewer 座席が返すことがある」は、`は` が 2 つ続き、incomplete の説明の続きに skipped の話が混ざる。2 文に分ける。(b) 「reviewer の画面で reviewer が差分を読んでいれば待つ」は同じ語が近くで 2 度出る(`leader.md:760`、org と director の skill)。(c) org の skill の `reviewer に見せていない変更があれば、TASK を送らず、ヘッダに ``XREVIEW: cap_reached``` の行は表示幅 86 で、追加行の中央値 72 を超える(4 写し) | (a) `.claude/skills/director/SKILL.md:986-988`、固定は `skill_xreview_test.go` の `ResultItemReportsTheCrossReviewLine`。(b) `leader.md:760`、`org/SKILL.md:1505`、`director/SKILL.md:1034`。(c) `.claude/skills/org/SKILL.md:1364` | 文を 2 つに分け、折り返しを直す。固定するテストの断片も合わせる。MEDIUM の直しと同じ行を触るので、まとめて直せる |

## Positive notes

- 順番は依頼どおり。reviewer の「cross-review の TASK」は、1 確かめ → 2 読む前の git の状態と差分 → 3 自分で読む → 4 triage → 5 report を書く前の git の比べ → 6 report → 7 返事の順で、7 つの手順の相互参照(2 の「6 の形」、3 の「5 に進み、6、7」、5 の「2 と同じ形」、7 の「5 と 1」)がそろっている。incomplete は、`diff: failed` なら 2 から直接 6、読めなかったときは 3 から 5 を通って 6 に行き、どちらも `Total reviewer findings: 0` で閉じる
- `git_changed` は leader と reviewer で合っている。reviewer の BLOCKED は本文に `XREVIEW: git_changed`、EVIDENCE に `xreview-<org>-c<k>-git-before.txt` と `-git-after.txt`、`REPORT:` の行なし。leader の項は同じ 2 つのパスを EVIDENCE に書き、`REPORT:` の行がないので「まず report をコミットする」は働かない。送り直しは同じ `<k>` を `-r2`、`-r3` で送り、2 つのファイルは上書きされる。比べる区間が、読む前と report を書く前に変わったことは、reviewer の step 5 と leader の文の両方にある
- 前の版の leader の TASK(`IMPLEMENTER_DRIVER:`、`REVIEWER_*:` を持つ)は読まれない。step 1 は 3 つの行だけを読み、step 2 と 5 のコマンドは `<k>` と `<XREVIEW_BASE>` だけを置き換える。テストは 5 行の本文と 3 行の本文から作ったコマンドが同じ文字列であることを bash と zsh の両方で確かめ、`codex` と `claude` の偽物を PATH の先頭に置いて呼ばれないことも確かめる。新しい leader の 3 行の TASK を前の版の reviewer に送ると QUESTION になる(計画の Assumptions のとおり)。leader の QUESTION の項に「わからなければ送り直さず escalate」の出口がある
- `Reviewer:` の行の形は、leader の手順 14 の例 `CROSS_REVIEW: ran (reviewer 座席 reviewer(codex、gpt-6.1-sol))` と一致する。leader が grep するのは `^- Reviewer status: complete[[:space:]]*$` だけで、`- Reviewer: ` の行とは衝突しない。`Triager:` と `count_triage_findings` が読む `After triage:` の行は、テストが実際に `scripts/xreview-helpers.sh` で数えて確かめている
- `RolePromptVars` の `Driver` と `Model`: `Spawn` と `dryRunSpawn` の 2 か所とも `p.Driver` と `p.Model` を渡し、`AgentStart` と `--model` に渡す値と同じ。`strings.NewReplacer` は 1 回の走査なので、leader の `{{TASK}}` に `{{MODEL}}` が入っていても二重に展開されない。`{{DRIVER}}` は `reviewer.md` と `director.md` にだけある。director は `RenderRolePrompt` が `("", false, nil)` を返し、`RenderDirectorPrompt` が埋める。`reseat.go:920` は空の vars で描くが、テキストは捨てる
- 「座席内 fan-out」の新しい決まりは、規則の箇条(例の前)、例の 3 つ目だけが「安いモデル」、と構造まで固定されている。org の skill の同じ節と、leader の雛形の外の文面は食い違わない
- テストが直しを外すと落ちる作り。step 2 と 5 のコマンドを行単位で固定し、変更 4 種(追跡ファイルの編集、新しい未追跡ファイル、未追跡ファイルの内容の編集、新しいコミット)で `git: changed` が出ることを 2 つの shell で走らせている。消した語の一覧(`reviewerXreviewGone`、`leaderXreviewGone`、`skillXreviewGone`)は、残してよい 2 文を取り除いてから探す作りで、残してよい文だけが通る。`readSkillCopies` で、skill の 4 写しすべてに掛けている
- 4 写しの skill、2 つの recipe は `cmp` で同一。`scripts/xreview-helpers.sh` は `#` の行だけが変わり、関数は 1 バイトも変わらない。追加行の表示幅は skill で中央値 72、recipe で 71.5、`leader.md` で 73、`reviewer.md` で 71 で、短い孤立行は段落の終わりだけ
- 差分に秘密、デバッグの残り、範囲外のファイルは見えなかった(実装側の確かめの担当。目に入った範囲)

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 追加なし。LOW の古いコメントと `pick_reviewer` の register 行(上の 3 つ目)は、計画の S4 が /sync-docs に回している既存の持ち越し。MEDIUM と LOW は register ではなく、直すか残すかの判断を担当側に返す | | | | `docs/plans/active/2026-11-22-org-xreview-own-model.md`(Progress の S4) |

_(行は足していない。追記は不要。)_

## Recommendation

- Merge: yes(CRITICAL なし、HIGH なし、MEDIUM 1 件、LOW 4 件)。MEDIUM は待ちの例外の直しで、雛形 1 面とテスト 2 か所の文面の変更で済み、振る舞いの欠陥ではなく文面の抜けなので、止める理由にしない。直さず進めるなら、PR の本文に「cross-review の待ちに上限がない」を known gap として書く
- Follow-ups:
  - MEDIUM 1 と LOW 5 の(b)(c)は同じ行を触る。直すなら 4 面とテストを 1 本で
  - LOW 3 の fan-out の 1 文は、雛形 1 か所とテスト 1 行
  - LOW 2 は /sync-docs で必ず拾う(`org_runtime_env_test.go:182-185`、`tech-debt/README.md:31`)
  - 範囲外の気づき: 計画の AC6(`xreview-helpers.sh` は差分に入らない)は、S4 が `#` の行を直したため字面どおりには満たさない。Progress に記録済みなので、/verify の判断に任せる
  - cycle は進めない。cross-review が ACTION_REQUIRED を返したときだけ進む
