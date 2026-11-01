# Cross-review triage report: director-split-questions

- Date: 2026-11-01
- Plan: docs/plans/active/2026-11-01-director-split-questions.md
- Base branch: main (4fef6b8c、origin/main と同じ)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached。4 回目の run)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=2, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-01-director-split-questions.md(digest c3e06a2ab308)
- Self-review report: docs/reports/self-review-2026-11-01-director-split-questions.md(`## Cycle 4`: LOW 5。C4-1 と C4-3 は Known gaps のつもりだった)
- Verify report: docs/reports/verify-2026-11-01-director-split-questions.md(`## Cycle 4`: pass、INFO 3)
- Test report: docs/reports/test-2026-11-01-director-split-questions.md(`## Cycle 4`: pass、run-test.sh rc 0、1,625 秒)
- Implementation context summary: 4 回目の run は HEAD 844c0061 を codex(gpt-6-astra、xhigh)が `exec review --base main` で読んだ。rc=0、`-o` のファイルは空でなく、走っているあいだの git の状態は前後で同じ(cmp)。codex の sandbox は read-only なので Go のテストは打てなかった
- 手順 0 の読み方: PATH の ralph は 6.0.0 で `config` の動詞がない。前の 3 回と同じく、この worktree の `go run ./cmd/ralph config get standard.cross_review` で読み直した(`true`、exit 0)

## Earlier cycles (for the record)

- cycle 1(HEAD 405f680f): ACTION_REQUIRED 2・WORTH_CONSIDERING 1。S9・S10 で直した
- cycle 2(HEAD 3fed4c55、2/2 cap reached): ACTION_REQUIRED 1・WORTH_CONSIDERING 2。上限を 3 に上げ、S12・S13 で直した
- 3 回目の run(HEAD 573f5265、2/3): ACTION_REQUIRED 1(最初の start の断りのあと `ralph status` を読み直さない)・WORTH_CONSIDERING 1(add と done のあいだの停止)。S14・S15 で直し、cycle を 3 にした

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] ブランチ名が使い回され、新しい PR がまだないとき、`merge-order --all` は番号が最大の前の PR で `merged` と出す(`pickMergePR` と `classifyMergeTarget` は今の機能の結びつきと比べない)。S15 の「STATE が `waiting` でも merge-order で `merged` なら待ち終えた」はこれを待ち終えたと数え、「後に回す」の答えで done になり、本当の先行する機能の merge では聞かれなくなる | 正しい(`internal/org/mergeorder.go` の `pickMergePR` 1000 行、`classifyMergeTarget` 1068 行)。self-review の C4-1 と同じで、起票と記録の 4 の「もう merge 済みなら記録しない」にも同じ穴がある。直すのは安い: そのブランチに結びついた機能の org がまだ走っている(`ralph status` の org の `feature:` の行にそのブランチがある)なら、merge-order が `merged` と出しても待ち終えたと数えない、を待ち終えたと数える 3 か所(起票と記録の 4、引き継ぎ、2 の判定)にそろえて足す | .claude/skills/director/SKILL.md(1355〜1361 行ほか)と写し 3 つ、internal/org/skill_split_questions_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] worktree を作ったあと leader の spawn の前に別の org が重なる予約を取ると、`StartFeature` は worktree とブランチを残す(`TestStartFeature_SpawnRefusedAfterEnsure`)。そのあと起票して待ち、同じ slug で分割計画を書き直すと、`checkFeatureWorktreeReuse` が `canonical_ref` の違いで断り、先行する機能の merge のあとも着手できない | 正しい。予約の取り合いのときだけ起きるが、起きると書いたとおりには再開できない。直すのは安い: 3 で分割計画を書き直す前に、前の start が残した worktree の記録(`org-<org_id>`)があるかを確かめ、あれば `/org` skill の「start が拒否するもの」のとおり slug を変えるか、要らないかを人に聞いて `cleanup` を人に頼む(director は消さない) | .claude/skills/director/SKILL.md(「merge を待つ issue」の 3、1415〜1418 行)と写し 3 つ、internal/org/skill_split_questions_test.go |
| 3 | [P2] fork の checkout で gh の既定が upstream だと、`gh issue create` は upstream に issue を作るが、merge-order は origin の PR で確かめる。follow-up は issue の番号しか持たないので、gh の既定を origin に直すと、引き継ぎ・view・edit が origin の別の issue を指す | 正しい。fork の checkout のときだけ起きるが、起きると別の issue を読み書きする。直すのは安い: director の gh の issue のコマンド(create、list、view、edit)には、origin の URL から決めた `--repo <owner>/<repo>` を必ず付ける、と 1 か所に書き、各コマンドの例にも付ける | .claude/skills/director/SKILL.md(622〜623 行ほか、gh issue の 12 か所)と写し 3 つ、internal/org/skill_split_questions_test.go |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## 決めたこと

- Case A で上限に届いている(cycle 3/3)。ユーザーの指示(「回数の上限(既定 2)に当たったら 3(その次は 4)で上げて回し直す。このとき cycle-count.json は増やさない」と、ACTION_REQUIRED は聞かずに直し、WORTH_CONSIDERING は実害があって安く直せるものは直す)に従い、選択肢 1(上限を上げて回し直す)を選んだ。`cycle-count.json` に `cap: 4` を書き(`cycle` は 3 のまま)、3 件とも直して /self-review から回し直す(5 回目の run)。ユーザーの指示は 4 までなので、5 回目の run の cross-review で残った所見は PR の Known gaps に書く
