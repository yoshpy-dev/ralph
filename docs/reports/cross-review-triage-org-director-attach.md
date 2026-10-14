# Cross-review triage report: org-director-attach

- Date: 2026-10-15
- Plan: docs/plans/active/2026-10-14-org-director-attach.md
- Base branch: main(merge-base ef575748)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-14-org-director-attach.md(承認済み、digest 27414f27525b。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-14-org-director-attach.md(CRITICAL・HIGH なし、MEDIUM 2・LOW 10。MEDIUM 2 件と LOW 8 件は 57c551ff で直した。L5 の直しで、replace の `detached` と `attached` を 1 回の write で書き、失敗したら「何も記録されていない、前の登録が残る」と言うようにした)
- Verify report: docs/reports/verify-2026-10-14-org-director-attach.md(pass。AC1〜AC12 を満たす)
- Test report: docs/reports/test-2026-10-14-org-director-attach.md(pass。生き残った mutant 1 件(D5: ほかの id の `detached` が今の登録を終わらせる)は、既存のテストの 2 行が打ち消し合って見えなくしていた)
- Implementation context summary: `codex exec review --base main` を cbd85a66 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で、`-o` のファイルは 1033 バイト)。Codex は read-only の sandbox で Go のテストを走らせられなかったと書いている。テストは test の手順で済ませている

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] `attach --replace` の書き込みが途中で失敗する(ディスクが一杯、ファイルの大きさの上限)と、`os.File.Write` は `detached` の行を書き終えて `attached` の行の途中で止まることがある。`appendLocked` は書けた分をそのまま残すのに、`Attach` は「何も記録されていない、前の登録が今の登録のまま」と言う。次に読むと `detached` が効き、途中で切れた `attached` は読めない行になって、登録がなくなる。replace を transactional にするか、途中まで書けた分を戻し、`detached` のあとで失敗する write のテストを足す | 本当の問題で、直す価値がある。self-review の L5 の直し(57c551ff)で「何も記録されていない」と言い切る文言を入れたので、その文言が事実と違いうる。起きるのはディスクが一杯のときなどに限られるが、登録が黙って消えると director は自分の `id` が今の登録と違うと読んで仕事をやめる(`/director` skill の昇格 5)。直し方: 書き込みか close が失敗したら、ロックを持ったままファイルを読む前の大きさに切り詰めて戻す。戻せたときだけ「何も記録されていない」と言い、戻せなかったときは、途中まで書けたかもしれないことと `ralph org director status` での確かめ方を言う。plain な attach と detach も同じ経路を通る。test の生き残った mutant D5 を落とすテストも同じ直しに入れる | `internal/org/director.go`(`appendLocked`、`Attach`、`Detach`)、`internal/org/director_test.go` |

## WORTH_CONSIDERING

なし。

## DISMISSED

なし。
