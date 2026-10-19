# Cross-review triage report: org-director-wait

- Date: 2026-10-19
- Plan: docs/plans/active/2026-10-19-org-director-wait.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: `docs/plans/active/2026-10-19-org-director-wait.md`(8 段目の後半の 2 本目の後半、8b-2b。director の wait の印と、見張りの停止に気づくこと)
- Self-review report: `docs/reports/self-review-2026-10-19-org-director-wait.md`(Merge: yes。MEDIUM 1 件と LOW 7 件は直した。直しの再確認で出た LOW 2 件も直した)
- Verify report: `docs/reports/verify-2026-10-19-org-director-wait.md`(pass。LOW 2 件と INFO 4 件は直すか、sync-docs で tech-debt に書いた)
- Test report: `docs/reports/test-2026-10-19-org-director-wait.md`(pass。mutant 56 個のうち 50 個が赤。生き残りの穴は tech-debt の行にした)
- Sync-docs report: `docs/reports/sync-docs-2026-10-19-org-director-wait.md`
- Reviewed range: `main...HEAD`、HEAD 877325ed(base e38b89a9)
- Reviewer: `codex exec review`(codex-cli 0.160.1、`gpt-6-astra`、reasoning effort `xhigh`、`-c sandbox_mode=read-only`、`--ignore-rules`)。rc 0、`-o` のファイルは 1076 バイト
- Reviewer の結論(原文): "A registration-replacement race can overwrite the current director's wait mark and suppress its subsequent updates. Formatting and diff checks passed; Go tests could not run because the read-only sandbox prevented temporary-directory creation."
- Implementation context summary: `ralph org wait --inbox --director <id>` は、始める前に `director.jsonl` を読んで今の登録を確かめ、見張りを確かめてから、`director-wait.lock` の中で前の印を読み、自分の印を書いて持ち主になる(D2)。持ち主の確かめ(`owns`)は 2 回目からの書き込みだけに掛かり、始めの書き込みは前の印をそのまま置き換える。登録の読み直しは 30 秒ごとの `refresh` だけで行う

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Revalidate the registration before taking over the wait mark. d1 の wait が登録の確かめを通ったあと、見張りの確かめ(settle は最大 2 秒)で止まっているあいだに、`attach --replace` が d2 を登録し、d2 の wait が印を書くことがある。そのあと d1 の `begin` は、手元の d1 の登録のまま、ロックの中で d2 の印を無条件に上書きする。d2 は次の `save` で持ち主でないと分かり、印を二度と書かない。status は d1 の印を今の登録のものとして出さないので、今の wait の印は始まりの時刻のまま止まり、終わり方も書かれない | 本物の欠陥。`begin` の書き込み(`internal/org/director_wait.go:412-415`)は `director.jsonl` を読み直さず、確かめ(`WaitInboxForDirector` の `directorWaitCheckCurrent`)とロックのあいだに登録が変わると、今の登録でない wait が印を書く。AC1(今の登録でなければ印を書かずに終了コード 5)と D2(印は今の登録の wait のもの)に反し、新しい director の印が止まる。直し方は小さい: `begin` のロックの中で `director.jsonl` を読み、`refresh` と同じ `directorWaitCheckCurrent(id, &w.reg, cur)` で確かめ、違えば印を書かずに `ErrDirectorWaitNotCurrent` で返す。読めなければ書かずにその誤りで返す。テストは、確かめとロックのあいだに登録を差し替える差し込みで、印が d2 のまま残り、d1 が 5 で返り、d2 がそのあとも印を書けることを固定する | `internal/org/director_wait.go`(`begin`、`WaitInboxForDirector`)、`internal/org/director_wait_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
