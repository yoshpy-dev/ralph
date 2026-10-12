# Cross-review triage report: org-watch-gate

- Date: 2026-10-12
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md
- Base branch: main(merge-base dc41c4f5)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み、digest de5ca6b7e6f0。変わっていない)
- Self-review report: docs/reports/self-review-2026-10-12-org-watch-gate.md(cycle 2 は 236dc72e で Merge: yes、LOW 4 件。C2-1〜C2-3 は 3fb0fd1b で直し、C2-4 は sync-docs の 2fada512 で入れた)
- Verify report: docs/reports/verify-2026-10-12-org-watch-gate.md(cycle 2 は 755e082d で pass)
- Test report: docs/reports/test-2026-10-12-org-watch-gate.md(cycle 2 は 926efd94 で pass。生き残った mutant K8 は 47635690 でテストに入れた)
- Implementation context summary: `codex exec review --base main` を 2f2c3df5 で実行した(モデルは gpt-6-astra、effort は xhigh、sandbox は read-only。`codex rc=0` で `-o` のファイルは空でない)。この計画で cross-review を回すのは今回が初めて(cycle 1 は /test で後退が見つかり、cross-review の前に全段を回し直した)。指摘は 1 件。分類と直し方は consult(plan)に聞き、ACTION_REQUIRED と案 A に同意を得た

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] 見張りの子の作業ディレクトリを移す前に、相対パスの agmsg home を絶対パスにする。サブディレクトリか linked worktree で、相対の `RALPH_ORG_AGMSG_HOME`(または明示した `--config` の中の相対の `agmsg_home`)を使って `--detach` か start を打つと、子の cwd が変わるので、そのパスは別の場所を指す。`driver.ResolveAgmsgHome` は相対パスをそのまま返し、`Agmsg.run` は子の cwd を起点にスクリプトを呼ぶ。見張りは生存の印を書いて検査を通すのに、ALERT の送信と history の読み取りが失敗しうる | 事実と合う。`Agmsg.run` は `filepath.Join(a.Home, "scripts", script)` を bash に渡し(`internal/org/driver/agmsg.go:31-35`)、`ResolveAgmsgHome` は `~/` だけを展開する(`:123-132`)。base では子が親の cwd を引き継いでいたので、この 2 つの形はこの PR(計画の F6、`orgWatchChild` が Dir を `LedgerMainWorktreeRoot` にする)で初めて壊れる。相対の agmsg home は文書で案内していないが、設定は拒まない。迷えば上に分類する規則と、直すのが小さいことから ACTION_REQUIRED。直し方(consult が同意): Dir を変えるときだけ、(i) env が相対なら親の cwd を起点に絶対パスにして子の env に入れ直す、(ii) env が未設定か空で、`--config` を明示していて、その設定の `agmsg_home` が相対なら、親の cwd を起点に絶対パスにした `RALPH_ORG_AGMSG_HOME` を子の env に足す。`--config` がないとき(子は main のルートの `ralph.toml` を読む)、絶対パス、`~/`、空の値は触らない。退けた案: `ResolveAgmsgHome` で絶対パスにする(子は main のルートで解決するので直らない)、Dir を移すのをやめる(tech-debt 205 が開き直る)、tech-debt に書くだけにする(この PR が持ち込んだ後退) | `internal/cli/org_watch.go`(`orgWatchChild`)、`internal/cli/org_watch_test.go` か `internal/cli/org_start_watch_test.go` |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Decision

上限(`RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`)に届いているが、ユーザーの指示(「cross-reviewにおいてのACTION_REQUIRED以上の修正は必ず修正するようにしてください。私の指示待ちは必要ありません」)に従い、上限を 3 に上げて #1 を直し、self-review から全段を回し直す(cap-reached の選択肢 1)。skill の決まりにより `cycle-count.json` は 2 のまま増やさず、回し直しの報告には cycle 3 と書く。
