# Cross-review triage report: org-cross-watch

- Date: 2026-10-11
- Plan: docs/plans/active/2026-10-11-org-cross-watch.md
- Base branch: main(merge-base 577ebeaf)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-11-org-cross-watch.md(承認済み、digest d0a6f2735552。生存の印の窓の規則は Progress checklist に記録したずれ)
- Self-review report: docs/reports/self-review-2026-10-11-org-cross-watch.md(再実行の結果は Merge: yes。最初の結果の M1・L1〜L7 は 1b64ae2b と 188a640d で直し、再実行の N1・N4・N5 は 94d5248c、N2・N3 は tech-debt)
- Verify report: docs/reports/verify-2026-10-11-org-cross-watch.md(pass)
- Implementation context summary: `codex exec review --base main` を 26355d61 で実行した(モデルは gpt-6-astra、sandbox は read-only)。Codex はテストを走らせていない(読み取り専用の環境のため)。指摘は P2 が 2 件で、どちらも該当のコードを読んで、書かれたとおりの窓があることを確かめた。self-review の L1 の直し(`--once` でない見張りは watcher の待ちの前に ctx を取り消す)は、ロックを放したあとに watcher を待つ順そのものは変えていない。止める頼みの 2 つ目は、S3a のずれ(起動のときに消すのは `started_at` より前の頼みだけ)を入れたときの、読んでから消すまでの間の問題

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] on-demand の watcher が終わるまで見張りのロックを持つ。`--all --once` で遅い意味判定が走ると、`RunWatchAll` は stopped の印を書いて `watch.lock` を放してから、CLI が watcher の goroutine を待つ。その間 `--stop` は「動いていない」と答え、2 つ目の `--all` が通るのに、1 つ目のプロセスはまだ ALERT を送り receipt を書きうる | 本物の問題。計画の「1 プロセスの保証」と、`--stop` の「止まった」の答えが、watcher の後片付けの間は成り立たない。`--once` では watcher の待ちが最大 60 秒+送信 10 秒続く(L1 の直しは `--once` でない場合だけ取り消す)。直し方は小さい(`RunWatchAll` が stopped を書いてロックを放す前に、CLI の watcher を待たせる口を作る) | internal/cli/org_watch.go:180-195、internal/org/watch.go(RunWatchAll の終わり) |
| 2 | [P2] 起動のときの古い頼みの削除が、同時に書かれた新しい頼みを消しうる。kill された見張りが古い頼みを残していると、起動の `dropOldStopRequest` がそれを読んだ直後に、同時の `--stop` が新しい頼みに置き換え、そのあとの無条件の削除が新しい頼みを消す。見張りは止まらず、`--stop` は 30 秒で「頼みは残っている」と言って終了コード 1 になる(頼みは残っていない) | 本物の問題。窓は狭い(起動の数ミリ秒)が、`--stop` の答えが事実と違う。読んでから消すまでの間に書き手と同期していない。直し方は小さい(頼みを一意の名前に rename して取り出し、取り出したものが自分より後なら従う、など) | internal/org/watch.go:660-664(dropOldStopRequest)、internal/org/watchalive.go:204-231 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
