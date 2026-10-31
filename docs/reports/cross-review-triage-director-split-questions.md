# Cross-review triage report: director-split-questions

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-director-split-questions.md
- Base branch: main (4fef6b8c、origin/main と同じ)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-director-split-questions.md(digest c3e06a2ab308)
- Self-review report: docs/reports/self-review-2026-11-01-director-split-questions.md(Run 1〜3。M1 と L1〜L8、R2-1・R2-2、R3-1〜R3-3 は直した)
- Verify report: docs/reports/verify-2026-11-01-director-split-questions.md(pass)
- Implementation context summary: director の skill の文とその固定のテストの変更。HEAD 405f680f を codex(gpt-6-astra、xhigh、`ralph_setting` の既定)が `exec review --base main` で読んだ。走っているあいだの git の状態は前後で同じ(cmp)。codex の sandbox は read-only なので Go のテストは打てなかった(review の冒頭の文)
- 手順 0 の読み方: PATH の ralph は 6.0.0 で `config` の動詞がない(exit 1、`unknown command "config"`)。skill の決まりでは「古い ralph」として飛ばすが、この repo の `ralph.toml` は `[standard].cross_review = true` なので、この worktree の `go run ./cmd/ralph config get standard.cross_review` で読み直した(`true`、exit 0)。#267 の PR #286 と同じ扱い

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 引き継ぎで、起票のあと記録の前に止まり、引き継ぎまでに待つブランチがすべて merge された issue は、どのブランチも記録されず(merge 済みは数えるだけ)、着手の問いは `ready` の follow-up を求めるので、二度と聞かれない | 正しい。「起票と記録」の 4 は S5(self-review L1)ですべて merge 済みならすぐ着手を聞くが、引き継ぎの段落には同じ扱いがない。記録より前の merge は数えないので、あとから記録しても `ready` にならない。引き継ぎにも「すべて merge 済みならすぐ着手を聞く」を足す | .claude/skills/director/SKILL.md(「merge を待つ issue」の冒頭、1286〜1289 行)と写し 3 つ、internal/org/skill_split_questions_test.go |
| 2 | [P2] 着手のあと start が重なりで断ったとき、エラーは最初に重なった org だけを出す(`internal/org/reserve.go` の `reservationDecision` は最初の重なりで返す)。`after:` をそのブランチ 1 つに書き換えると、ほかの待つブランチが落ち、最初の merge で着手を聞き、推奨の「後に回す」で follow-up がすべて done になり、残りの merge を知らせるものがなくなる | 正しい(`reserve.go` の 574〜577 行で確かめた)。起票の手順は `ralph status` の `reserved:` の行から重なる org を全部数えるが、書き換えはエラーの 1 つだけを使う。書き換えの前に `ralph status` を読み直し、重なる org を全部数えて、そのブランチを全部 `after:` に書き、全部を記録する | .claude/skills/director/SKILL.md(「merge を待つ issue」の 3、1345〜1347 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 3 | [P2] `gh issue list --limit 1000` は、issue が 1,000 本を超える repository では途中で切れる。切れた外の起票した機能は「予約外の文書の follow-up」で除外されず、集め始めが止まる。引き継ぎの open の一覧も、途中で止まった起票を見落とす | 起きるのは issue が多い repository だけだが、起きると黙って止まる。直すのは安い: 出力の行数ではなく、`--jq` の前の件数が上限に届いたら一覧が切れたとみなし、印の行がないと決めずに人に言う(または上限を上げて打ち直す)。実害があり安いので直す | .claude/skills/director/SKILL.md(1283 行と 1392 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 決めたこと

- Case A(ACTION_REQUIRED あり、cycle 1/2 で上限に届いていない)。ユーザーの指示(「ACTION_REQUIRED は聞かずに直し、/self-review から回し直す。WORTH_CONSIDERING は、実害があって安く直せるものは直し、残りは PR の Known gaps に書く」)に従い、選択肢 1(Fix)を選んだ。3 件とも直し、`cycle-count.json` を 2 にして /self-review から回し直す
