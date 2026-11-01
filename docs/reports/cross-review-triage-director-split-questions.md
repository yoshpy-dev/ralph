# Cross-review triage report: director-split-questions

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-director-split-questions.md
- Base branch: main (4fef6b8c、origin/main と同じ)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/4(5 回目の run。cycle 2 と 3 で上限に届き、`cap` を 3、4 に上げたので cycle の数は 3)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-director-split-questions.md(digest c3e06a2ab308)
- Self-review report: docs/reports/self-review-2026-11-01-director-split-questions.md(`## Cycle 5`: LOW 5、`## Cycle 5 (S18)`: LOW 4。C5-3 は S18 で直した)
- Verify report: docs/reports/verify-2026-11-01-director-split-questions.md(`## Cycle 5` と `## Cycle 5 (S18)`: pass)
- Test report: docs/reports/test-2026-11-01-director-split-questions.md(`## Cycle 5`: pass、run-test.sh rc 0、1,234 秒。S19 のあとの絞った打ち直しも ok)
- Implementation context summary: 5 回目の run は HEAD 19156a3b を codex(gpt-6-astra、xhigh)が `exec review --base main` で読んだ。rc=0、`-o` のファイルは空でなく、走っているあいだの git の状態は前後で同じ(cmp)。codex の sandbox は read-only なので Go のテストは打てなかった
- 手順 0 の読み方: PATH の ralph は 6.0.0 で `config` の動詞がない。前の 4 回と同じく、この worktree の `go run ./cmd/ralph config get standard.cross_review` で読み直した(`true`、exit 0)

## Earlier cycles (for the record)

- cycle 1(HEAD 405f680f): ACTION_REQUIRED 2・WORTH_CONSIDERING 1。S9・S10 で直した
- cycle 2(HEAD 3fed4c55、2/2 cap reached): ACTION_REQUIRED 1・WORTH_CONSIDERING 2。上限を 3 に上げ、S12・S13 で直した
- 3 回目の run(HEAD 573f5265、2/3): ACTION_REQUIRED 1・WORTH_CONSIDERING 1。S14・S15 で直した
- 4 回目の run(HEAD 844c0061、3/3 cap reached): ACTION_REQUIRED 1(ブランチ名の使い回し)・WORTH_CONSIDERING 2(start が残した worktree、fork の checkout の gh)。上限を 4 に上げ、S16・S17 で直し、self-review の C5-3 を S18、test の穴を S19 で直した

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `GH_HOST` が Enterprise の host のとき、`--repo <owner>/<repo>` はその host で解決され、origin が github.com の repository でも、別の host の repository に issue を作り書き換えうる。ralph の merge と follow-up の読み手は origin の host を使う(`MergeOrigin.Arg()`) | 正しい。self-review の C5S-3 と同じ。外への書き込みの向きの誤りで、ralph の Go は gh を origin の host に固定している(`internal/org/mergeguarantee.go` の `--hostname`)。直しは `--repo github.com/<owner>/<repo>` と `gh api user --hostname github.com` | .claude/skills/director/SKILL.md(678〜683 行ほか、gh の issue と pr のコマンド)と写し 3 つ、internal/org/skill_split_questions_test.go |
| 2 | [P2] 「merge を待つ issue」の 1 で記録した普通の issue(印の行がない、または別の人が書いた issue)で、人が着手を選び、start が重なりで断り、人が待つと答えると、印の行の確かめが記録し直しを禁じ、前の follow-up は done なので待ちが消える | 正しい。計画の Test plan の端の場合「着手を聞くときに新しい org と重なっている」の経路で、#270 からの普通の follow-up で起きうる。直しは「印の行の確かめに通らない issue は本文を書き換えず、1 のとおり相手の org の `feature:` のブランチで `followup add` だけする」 | .claude/skills/director/SKILL.md(1479〜1481 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 3 | [P2] 書き換えた待ちに前の待ちと同じブランチが入ると(A/B の待ちを A/C に替えるなど)、`AFTER_BRANCH` だけで絞るので前の done の件が残り、done と open の混在の片付けが古い `DONE_NOTE` で新しい待ちを done にしてしまう | 正しいが、書き換えは前の待ちがすべて ready のあとなので、同じブランチが再び重なるのはブランチ名の使い回しのときくらい。直すのは安い: 引き継ぎで数える件の絞り込みを、`AFTER_BRANCH` ではなく「NOTE が今の印の行(今の `after:` を持つ)と同じ件」に替える。note は確かめた値から組んだ印の行なので、書き換えの前後で値が違い、待ちの世代を区別できる。新しい待ちを黙って消す実害があるので、同じ run で直す | .claude/skills/director/SKILL.md(1334〜1337 行)と写し 3 つ、internal/org/skill_split_questions_test.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 決めたこと

- `cycle-count.json` は cycle 3・cap 4 で、skill の手順 8 の CAP_REACHED(cycle >= cap)は false。Case A で上限に届いていないので、ユーザーの指示(「ACTION_REQUIRED は聞かずに直し、/self-review から回し直す。WORTH_CONSIDERING は、実害があって安く直せるものは直す」)のとおり、選択肢 1(Fix)を選んだ。3 件とも直し、cycle を 4 にして 6 回目の run を回す。6 回目の cross-review は 4/4 で上限に届き、ユーザーの指示は上限 4 までなので、そこで残った所見は PR の Known gaps に書いて /pr に進む
- 訂正: 4 回目の run の triage の「決めたこと」に「5 回目の run の cross-review で残った所見は PR の Known gaps に書く」と書いたが、これはユーザーの指示ではなく私の読み替えで、cycle 3 < cap 4 の skill の決まりとも合わなかった。consult(consult-plan-splitq)に確かめ、この triage で訂正した
- 5 回の run の指摘は、どれも印の行・follow-up・引き継ぎ・書き換えの状態の扱いの端の場合だった。散文の状態の扱いの限界なので、PR の本文で、`ralph org followup` に分割計画の結びつき(split/slug、待ちの世代)を持たせ、印の行の読み書きと重なりの判定を動詞にする follow-up を勧める
