# Cross-review triage report: send-screen-fallback

- Date: 2026-11-13
- Plan: docs/plans/active/2026-11-12-send-screen-fallback.md
- Base branch: main (0936aaea)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=1, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-send-screen-fallback.md(D3、AC4)
- Self-review report: docs/reports/self-review-2026-11-12-send-screen-fallback.md(cycle 2: LOW 8、Merge: yes)
- Verify report: docs/reports/verify-2026-11-12-send-screen-fallback.md(cycle 2: pass)
- Implementation context summary: cycle 1 の ACTION_REQUIRED(入力欄の印の行が番号付きの手順で始まると選択の画面とみなし、Enter を押さない)は f24290b5 と cea96ea4 で直した。send の確かめは、印の行の文が打った文の一部で、入力欄の形があるときだけ番号付きの行を入力欄とみなす。入力欄は今も `findResumeInputBox` の見つけ方(下から最初の、trim して印 `❯`/`›` で始まる行)を土台にしている。Enter の前の確かめが通らないと、画面の道は Enter を押さず `SendProgressTextTyped` で終わり、CLI は打ち直さず pane を読むよう案内する。herdr の道(herdr が idle/done を返す普通の道)は Enter の前の確かめをしない。
- Cycle 1 の経過: 指摘 1 件を ACTION_REQUIRED に分け(入力欄の番号付きの行)、常設の指示により聞かずに直してパイプラインを回し直した。直しは f24290b5(番号付きの行の受け入れ、M02 のテスト)と cea96ea4(L11 の守り)。cycle 1 の版のこの報告は 0efeef81。
- Cycle 2 の判断: consult(判定 A)を入れて、上限を上げずに PR へ進むことにした。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 送る文の中に印で始まる行があると(例: `TYPE: QUESTION` の本文に `❯ explain this`)、`findResumeInputBox` が下から最初の印の行(文の中の行)を入力欄の印とみなす。claude ではその行の真上が罫線でないので入力欄が見つからず、画面の道は文を打ったまま Enter を押さない。入力欄を境界(罫線)で見つけ、印の形の文字を含む文の回帰テストを足す | 実際の問題だが、直すかは議論の余地があると判断した。機構は consult が確かめた(claude は真上が罫線でなく no input box、codex は入力欄は取れるが `holdsHead` も `↑` の条件も満たさない。両 driver で起きる)。失敗は `SendProgressTextTyped` の明示のエラーで、CLI は打ち直さず pane を読むよう案内するので、黙る穴ではない。#307 の線引き(今回の変更が持ち込んだ黙る向きの穴は直し、知らせる側のずれは記録する)では記録する側になる。印で始まる行は画面の引用でしか出ず、agent-messaging.md は画面の生の出力を貼らないよう求めているので、まれとみる(推測、未計測)。上限(2)を上げて直す案は、claude だけを罫線で囲む方式が実機で確かめておらず、codex の `›` が残るので見送った。残るリスク: 入力欄に文が残ったあと、次の send が herdr の道に入ると Enter の前の確かめなしで上に重ねて打ち、成功として記録しうる(前からある TextTyped の契約で、今回の変更はそこへ至る道を 1 つ増やした)。leader と director の prompt に、`text typed but not submitted` で打ち直さず pane を読んで escalate する文がない。PR の Known gaps に記録し、直し方(claude は下の罫線とその上の罫線で入力欄を取る、codex は印の行から footer までの文と打った文の終わりを照らす、両 driver の回帰 fixture)と prompt の案内の追加を後続の issue にする | `internal/org/send_screen.go`(`sendTypedInputBox`、`sendInputHolds`)、`internal/org/resume_screens.go`(`findResumeInputBox`)、`internal/org/prompts/leader.md`、director の skill |

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
