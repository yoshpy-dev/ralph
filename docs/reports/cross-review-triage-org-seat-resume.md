# Cross-review triage report: org-seat-resume

- Date: 2026-11-07
- Plan: docs/plans/active/2026-11-07-org-seat-resume.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached; raised to 3 for this task by the user's standing policy)
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-07-org-seat-resume.md(承認済み、digest 54630558b591)
- Self-review report: docs/reports/self-review-2026-11-07-org-seat-resume.md(cycle 2 after R4: CRITICAL 0、HIGH 0、MEDIUM 0、LOW 5)
- Verify report: docs/reports/verify-2026-11-07-org-seat-resume.md(cycle 2、pass)
- Implementation context summary: cycle 1 の 4 件を R1・R2 で直し、self-review の cycle 2 の指摘を R3・R4 で直した版へのレビュー。3 件とも cycle 1 の直し(R1 のロックの中の待ち直し、R2 の先頭一致)が作った穴か、その直しが届いていない経路を突いている。ユーザーの方針(2026-10-11)で、上限に届いても ACTION_REQUIRED は聞かずに直し、上限を上げてパイプラインを回し直す。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] per-plan の implementer への TASK で、最後の承認の確認(`checkSendApproval`)がペインロックの待ちと idle/done の待ち直しより前にある。待つあいだに計画が変わるか承認し直されると、古い TASK を確認なしで送る | 実在する。R1 が待ちを足したことで、元の `Send` が守っていた「待ちのあと、打つ直前に承認を確かめ直す」が崩れた(agent-messaging の承認の規則)。直し: 最後の承認の確認を、ロックの中の待ち直しのあと、打つ直前に移す。待ちのあいだの計画の改訂を回帰テストにする | internal/org/verbs.go |
| 2 | [P2] 返事が `⏺ Documented the expected failure:` のあと空行を挟んで `API Error: Connection refused` で終わると、空行で分かれた 2 つ目のまとまりが印なしの先頭一致で止まったと判定される | 実在する。R2 は印のないまとまりも先頭一致で受け入れていた。直し: 止まった文は、本物のエラーの印(claude は `⏺`・`⎿`、codex は `■`・`⚠`)で始まるまとまりにだけ一致させる。複数段落の返事の回帰テストを足す。codex の上限の文の印は実機で見ていないので、違えば打たない側に倒れる | internal/org/resume_screens.go |
| 3 | [P2] herdr の headless の director に `wake_cmd` が設定されていると、`directorWakeSender` は `runDirectorWakeCmd` を呼び、ペインロックを通らない。`wake-herdr.sh` のような命令が resume と同時に同じ pane に打ちうる | 実在する。cycle 1 の #3 の直しが `typeDirectorWakeLine` にしか届いていない。直し: 登録が herdr で pane を持つときは、設定された wake の命令もペインロックの中で走らせる | internal/org/director_wake.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 1 の記録

cycle 1(2026-11-07、reviewer codex、complete)は指摘 4 件、4 件とも ACTION_REQUIRED だった。R1 `0aa8e274` と R2 `306ee58f` で直した。

1. [P1] `Send` がペインロックを取ったあと座席の状態を確かめ直さない → R1: ロックの中で idle/done を待ち直す(internal/org/verbs.go)
2. [P2] codex の入力欄の 1 行目が空で 2 行目以降に下書きがあると空と判定する → R2: 入力欄のすべての行を見る(internal/org/resume_screens.go)
3. [P2] director の push の wake(`typeDirectorWakeLine`)がペインロックを取らない → R1: 同じロックを取る(internal/org/director_wake.go)
4. [P2] 止まった文の照合が部分一致で、普通の返事を止まったと判定する → R2: 先頭の印を 1 つ落とした先頭一致(internal/org/resume_screens.go)
