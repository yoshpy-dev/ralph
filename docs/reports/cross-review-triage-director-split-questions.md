# Cross-review triage report: director-split-questions

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-director-split-questions.md
- Base branch: main (4fef6b8c、origin/main と同じ)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-director-split-questions.md(digest c3e06a2ab308)
- Self-review report: docs/reports/self-review-2026-11-01-director-split-questions.md(cycle 2: MEDIUM 1、LOW 5。S11 で直した)
- Verify report: docs/reports/verify-2026-11-01-director-split-questions.md(cycle 2: pass)
- Test report: docs/reports/test-2026-11-01-director-split-questions.md(cycle 2: pass、run-test.sh rc 0、1,447 秒)
- Implementation context summary: cycle 2 は HEAD 3fed4c55 を codex(gpt-6-astra、xhigh、`ralph_setting` の既定)が `exec review --base main` で読んだ。rc=0、`-o` のファイルは空でなく、走っているあいだの git の状態は前後で同じ(cmp)。codex の sandbox は read-only なので Go のテストは打てなかった(review の冒頭の文。Go のテストは /test の cycle 2 で pass)
- 手順 0 の読み方: PATH の ralph は 6.0.0 で `config` の動詞がない。cycle 1 と同じく、この worktree の `go run ./cmd/ralph config get standard.cross_review` で読み直した(`true`、exit 0)

## Cycle 1 (for the record)

cycle 1(HEAD 405f680f)は ACTION_REQUIRED 2(引き継ぎで待つブランチがすべて merge 済みの issue が浮かばない、start の断りは最初の重なりだけを出すので `after:` がほかのブランチを落とす)と WORTH_CONSIDERING 1(`gh issue list --limit 1000` が切れる)。S9 9248d73b・S10 e95f19e3 で直し、cycle 2 として全 pipeline を回し直した(self-review 25731fdc と S11 80cb8b6b、verify d4fa10f3、test 3b388a55、sync-docs 3fed4c55)。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 起票した機能 A に `- Depends on:` で頼る機能 B が、走っている org C とも重なって起票されると、B の follow-up は C のブランチだけを待つ。C が merge されると、「merge を待つ issue」の 2 の推奨は「着手する」になるが、A の issue はまだ実装されていない(A は予約を持たないので「今の org と重なれば後に回す」の例外に当たらない) | 正しい。「機能ごとの出力先」の 2 は頼られる機能を起票したら頼る機能も起票を勧めるが、merge のあとの推奨は予約の重なりしか見ない。中断がなくても起きる経路で、推奨が誤って着手に向く。2 の推奨に、issue の本文の `- Depends on:` の頼られる機能の issue がまだ open なら「後に回す」を推奨にする、を足す | .claude/skills/director/SKILL.md(「merge を待つ issue」の 2、1345〜1348 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] ブランチ A を待ち終えて done にしたあと、3 の書き換えで印の行をブランチ B を待つ形にし、B を記録する前に止まり、引き継ぎまでに B が merge されると、引き継ぎの「follow-up が 1 件もなければ」に A の古い done の件が当たって問いが出ず、B は merge 済みなので記録もされず、issue が黙って残る | 正しいが、書き換えと中断と merge が重なったときだけ起きる。直すのは安い: 引き継ぎで数える follow-up を、その issue の今の印の行の `after:` のブランチの件に限る(古い待ちの done は数えない)。黙って残る実害があるので直す | .claude/skills/director/SKILL.md(引き継ぎ、1296〜1300 行)と写し 3 つ、internal/org/skill_split_questions_test.go |
| 3 | [P2] A と B を待つ issue で、人の答えのあと A を done にし B を done にする前に止まると、A は done で `ready` にならず(status と merge-order は open の件だけを出す)、done の件があるので「follow-up のないブランチ」にも当たらず、B は open のまま残り、着手の問いも出ない | 正しいが、done を打つ途中で止まったときだけ起きる。直すのは安い: 引き継ぎで、今の印の行の `after:` のブランチの follow-up に done と open が混ざっている issue は、done の note を人の答えとして、残りの open の件を同じ note で done にする(着手は聞き直さない)。黙って残る実害があるので直す | .claude/skills/director/SKILL.md(引き継ぎと「merge を待つ issue」の 3、1350〜1353 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 決めたこと

- Case A で上限に届いている(cycle 2/2)。ユーザーの指示(「ACTION_REQUIRED は聞かずに直し、/self-review から回し直す。WORTH_CONSIDERING は、実害があって安く直せるものは直し、残りは PR の Known gaps に書く。回数の上限(既定 2)に当たったら 3(その次は 4)で上げて回し直す。このとき cycle-count.json は増やさない」)に従い、選択肢 1(上限を上げて回し直す)を選んだ。`cycle-count.json` に `cap: 3` を書き(`cycle` は 2 のまま。手順 9 の打ち方)、3 件とも直して /self-review から回し直す。ユーザーの「export RALPH_STANDARD_MAX_PIPELINE_CYCLES=3」は、#276 で環境変数が撤去されたので、この `cap` の書き込みに読み替えた
