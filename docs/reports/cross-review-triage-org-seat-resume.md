# Cross-review triage report: org-seat-resume

- Date: 2026-11-07
- Plan: docs/plans/active/2026-11-07-org-seat-resume.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 4
- After triage: ACTION_REQUIRED=4, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-07-org-seat-resume.md(承認済み、digest 0893ac30252e)
- Self-review report: docs/reports/self-review-2026-11-07-org-seat-resume.md(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 10。L1 が指摘 1 と同じ点)
- Verify report: docs/reports/verify-2026-11-07-org-seat-resume.md(pass、AC-10 は一部)
- Implementation context summary: resume は人の頼みで打つ動詞で、打つのは「止まった」と判定した画面だけという安全の条件(仕様の FR-2、FR-5a、非機能要件)を持つ。4 件はどれもこの条件か、計画の D8(打つ直前の割り込みを防ぐ)の穴を突いている。指摘 3 と 4 は S9 で tech-debt に回していたが、resume が director を対象にしたことと、誤判定が人の頼みの範囲でも作業中でない座席に新しい手番を始めさせることから、この PR で直す価値があると判断した。ユーザーの方針(2026-10-11)で、ACTION_REQUIRED は聞かずに直してパイプラインを回し直す。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] `Send` はロックを取ったあと座席の状態を確かめ直さない。ロックを待つあいだに resume が手番を始めるか承認画面に進めると、`Send` はそこに打ち、`confirmSubmitted` が `blocked` を受け入れるので承認画面に Enter が入りうる | 実在する。D8 の目的(打つ直前の割り込みを防ぐ)に対して `Send` 側の確かめが抜けている(self-review L1 と同じ)。直し: ロックの中で残りの期限まで idle/done を待ち直し、そうならなければ何も打たずに断る | internal/org/verbs.go |
| 2 | [P2] codex の入力欄の 1 行目が空で 2 行目以降に下書きがあると、入力欄を空と判定する。上限のエラーの下でこの状態だと、resume が続行の文を足して人の下書きごと送信する | 実在する。FR-2 の「入力欄が空」の条件を破る。直し: `findResumeInputBox` が見つけた入力欄の行をすべて見て、空か placeholder だけのときだけ空とする。先頭が改行の下書きの回帰テストを足す | internal/org/resume_screens.go |
| 3 | [P2] director の push の wake(`typeDirectorWakeLine`)はペインロックを取らずに打つので、resume と同時に同じ director の pane に打つと文が混ざり、`ResumeInputHolds` は先頭だけを見るので通ってしまう | 実在する。resume が headless の director を対象にしたことで、この PR で起こりうるようになった。直し: wake の打つ部分も同じペインロックを取る。tech-debt の行は解消にする | internal/org/director_wake.go, internal/org/resume.go |
| 4 | [P2] 止まった文の照合がまとまり全体の部分一致なので、`⏺ Added regression coverage for API Error: Connection refused.` のような普通の返事を止まったと判定し、作業の終わった座席に新しい手番を始めさせる | 実在する(tech-debt に記録していた点)。FR-2 の「API エラーでターンが終わった」に反する誤判定で、直しは小さい。直し: 止まった文は、まとまりの先頭の UI の印(`⏺`、`⎿`、`■`、`•` など)を除いた先頭に来るときだけ一致とする。待ち中と Enter だけの文は打たない側か Enter だけなので今のまま | internal/org/resume_screens.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
