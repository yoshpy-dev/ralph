# Self-review report: org-xreview-own-model

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
