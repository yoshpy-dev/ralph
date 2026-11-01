# Cross-review triage report: director-split-questions

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-director-split-questions.md
- Base branch: main (4fef6b8c、origin/main と同じ)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/3(3 回目の run。cycle 2 で上限に届き、`cycle-count.json` に `cap: 3` を書いたので cycle の数は 2 のまま)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-director-split-questions.md(digest c3e06a2ab308)
- Self-review report: docs/reports/self-review-2026-11-01-director-split-questions.md(`## Cycle 3`: MEDIUM 1、LOW 5。C3-1・C3-3・C3-4・C3-6 は S13 で直した。C3-2・C3-5 は Known gaps)
- Verify report: docs/reports/verify-2026-11-01-director-split-questions.md(`## Cycle 3`: pass、INFO 4)
- Test report: docs/reports/test-2026-11-01-director-split-questions.md(`## Cycle 3`: pass。負荷による watcher のテストの時間切れ 1 本を 1 回だけ打ち直して通った)
- Implementation context summary: 3 回目の run は HEAD 573f5265 を codex(gpt-6-astra、xhigh)が `exec review --base main` で読んだ。rc=0、`-o` のファイルは空でなく、走っているあいだの git の状態は前後で同じ(cmp)
- 手順 0 の読み方: PATH の ralph は 6.0.0 で `config` の動詞がない。前の 2 回と同じく、この worktree の `go run ./cmd/ralph config get standard.cross_review` で読み直した(`true`、exit 0)

## Earlier cycles (for the record)

- cycle 1(HEAD 405f680f): ACTION_REQUIRED 2・WORTH_CONSIDERING 1(引き継ぎで待つブランチがすべて merge 済みの issue、start の断りが最初の重なりだけを出すこと、`--limit 1000` の切れ)。S9・S10 で直した
- cycle 2(HEAD 3fed4c55、Cycle 2/2 cap reached): ACTION_REQUIRED 1・WORTH_CONSIDERING 2(頼られる機能が未実装でも着手を推奨すること、書き換えのあとの中断で古い done の件が問いを止めること、done を打つ途中の中断)。上限を 3 に上げ、S12 で直し、S13(self-review の直し)を足して全 pipeline を回し直した

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 「機能ごとの出力先」の 3 で最初の start が予約の重なりで断られ、2 に戻って「issue に起票」の推奨で聞き直すとき、重なりの組は 1 で読んだ古い `ralph status` のまま。出力先を聞いているあいだに別の org が重なる予約を取ると、start のエラーは最初の 1 つだけを出す(`reservationDecision`)ので、起票の印の行と follow-up がほかの待つブランチを落とす | 正しい。cycle 2 の verify の INFO C2-I2 と同じで、中断がなくても起きる。「merge を待つ issue」の 3 の書き換えには `ralph status` の読み直しがあるのに、最初の起票の経路にはない。「起票と記録」の 1 の前に `ralph status` を打ち直して重なる org を数え直す、を足す | .claude/skills/director/SKILL.md(「機能ごとの出力先」の 3、「起票と記録」の 1)と写し 3 つ、internal/org/skill_split_questions_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] follow-up のない issue に人が答えたあと、`followup add` が通り `followup done` の前に止まると、merge 済みのブランチの open の件が残る(記録より後の merge しか数えないので `waiting` のまま)。引き継ぎは件があるので聞き直さず、done と open の混在でもないので片付けもせず、人が承認した着手が黙って失われる | 正しい。S11 の consult が挙げ、Known gaps に回すつもりだった件。2 つのコマンドのあいだで止まったときだけ起きるが、起きると黙って失われる。直すのは安い: 引き継ぎで、今の `after:` のブランチの follow-up がすべて open で、どのブランチも記録より前に merge 済み(`ralph org merge-order --all` の `merged`、follow-up の STATE は `waiting`)の issue は、答えの記録の途中で止まったものとみなし、人にもう一度聞いて、答えの note で残りを done にする | .claude/skills/director/SKILL.md(引き継ぎと「merge を待つ issue」の 3)と写し 3 つ、internal/org/skill_split_questions_test.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 決めたこと

- Case A で、上限にはまだ届いていない(cycle 2 < cap 3)。ユーザーの指示(「ACTION_REQUIRED は聞かずに直し、/self-review から回し直す。WORTH_CONSIDERING は、実害があって安く直せるものは直し、残りは PR の Known gaps に書く」)に従い、選択肢 1(Fix)を選んだ。2 件とも直し、`cycle-count.json` の cycle を 3 にして /self-review から回し直す(4 回目の run)。4 回目の cross-review は cycle 3/3 で上限に届く。そこで ACTION_REQUIRED が残れば、ユーザーの指示どおり上限を 4 に上げる
