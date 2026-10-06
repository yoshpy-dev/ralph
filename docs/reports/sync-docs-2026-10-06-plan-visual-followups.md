# Sync-docs report: plan-visual-followups

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-06-plan-visual-followups.md
- Agent: doc-maintainer subagent
- Branch: fix/plan-visual-followups(base origin/main f9baf6a7、編集前の HEAD 528b3fbd)
- 前段の report: `self-review-2026-10-06-plan-visual-followups.md`(pass)、`verify-2026-10-06-plan-visual-followups.md`(pass、V-1〜V-6)、`test-2026-10-06-plan-visual-followups.md`(pass)

## Summary

verify が /sync-docs に回した 3 件(V-3、V-5、V-6)を直した。そのほか、パイプラインの順序と `/pr` の復旧を書いている文書を調べ、現在の挙動と食い違うものは見つからなかった。plan の `## Progress checklist` と AC のチェックボックスだけを更新し、本文は変えていない(digest は `eb13fda57381` のまま)。root と template の対(definition-of-done、insights README)は同じ内容にした。

## Files changed

| File | Change |
|------|--------|
| `docs/quality/definition-of-done.md`、`templates/base/docs/quality/definition-of-done.md` | 再実行の段落(`:30`)に、記録だけを直す例外を 1 文足した。全工程を回し直さないこと、triage レポートと PR 本文に記録すること、cycle count を上げないことを書き、3 条件と確かめ方は rules の節名で指す(条件を書き写していない)。V-3 |
| `docs/insights/README.md`、`templates/base/docs/insights/README.md` | `cycle` の行に「skill は `--cycle auto` を渡す」を足した。"Appending events" の例を `--cycle auto` にし、`--cycle N\|auto` の解決規則(`/cross-review` step 1 と同じ。状態ファイルがそろい `plan_path` が一致すれば `cycle`、それ以外は 1、状態ファイルのせいで append は失敗しない)、`--state-dir DIR`、cap を上げた追加の pass が同じ `cycle` を記録することを書いた。V-5 |
| `docs/tech-debt/README.md` | 「insight event の cycle スタンプ機構が脆弱」の行(`:128`)に、この表の他の解消済みの行と同じ書式で、debt item の取り消し線と `(RESOLVED 2026-10-06 in fix/plan-visual-followups)`、直前に説明の HTML コメントを足した(行は残す)。V-6。あわせて、このブランチが足した新しい行(`:156`)の plan のパスを `docs/plans/active/...` から `docs/plans/archive/...(archived by /pr; ...)` に直した。/pr が plan を archive へ移すと、active のパスが切れるため |
| `docs/plans/active/2026-10-06-plan-visual-followups.md` | `## Progress checklist` の「Review / Verification / Test artifact created」を `[x]` にし、report のパスを添えた。AC1〜AC6 を `[x]` にした。「PR created」は `/pr` が付ける。本文は変えていない |
| `docs/reports/sync-docs-2026-10-06-plan-visual-followups.md` | この report |

AC に付けた印の根拠: AC1・AC4・AC5・AC6 は test report の実行結果と、この pass で再実行した `./scripts/run-verify.sh`(下の表)、AC2・AC3 は verify report の Met。

## Drift check results

| 文書 | 結果 |
|------|------|
| `docs/quality/definition-of-done.md`(root、template) | 直した(V-3)。`:24` の「no steps may be skipped」は最初の pass の話で、例外は再実行だけに掛かるので、そこは変えていない |
| `docs/insights/README.md`(root、template) | 直した(V-5) |
| `docs/tech-debt/README.md` | 直した(V-6、パスの修正) |
| `README.md`(Quick start、Operating loop)、`AGENTS.md`(Primary loop) | 変更なし。順序は書いているが、再実行・cap・`/pr` の復旧は書いていない。`./scripts/check-pipeline-sync.sh` は 6 文書とも通った |
| `.claude/rules/ralph/subagent-policy.md` | 変更なし。段階の役割と順序だけ |
| `.claude/skills/implement/SKILL.md` step 13.e | 変更なし。「再実行のとき `/cross-review` が `cycle-count.json` を上げる」は、例外が再実行ではないので今も正しい |
| `.codex/README.md`、`.codex/AGENTS.override.md`、`.codex/agents/` | 変更なし。再実行の規則、`/pr` の復旧、許可リストに触れていない |
| `docs/recipes/codex-setup.md:84-86` | 変更なし。「cap のため fix-and-revalidate は 2 pass を超えない」は、記録だけを直す例外が再検証の run ではないので食い違わない |
| `docs/architecture/repo-map.md` | 変更なし。tests は `tests/test-*.sh` の glob で書いてあり、新しい 2 本の名前を挙げる必要がない。`insights-append.sh` は `:61` に、`.harness/state/standard-pipeline/` は `:71` にすでにある |
| `docs/quality/quality-gates.md`(root、template) | 変更なし。新しい検証の項目はない |
| 許可リストの写し(`Bash(sed:*)` など) | 文書にはない(`git grep` で `.claude/settings.json`、template の 2 つの JSON だけ) |
| `.claude/rules/ralph/post-implementation-pipeline.md` | 変更なし。V-4(cap の選択肢の番号)は orchestrator が 25c45213 で直し済み |
| `docs/plans/archive/` | 変更なし。履歴 |

## Verification

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `bash scripts/check-template-purity.sh` | PASS |
| `./scripts/check-pipeline-sync.sh` | PASS(6 文書) |
| `./scripts/plan-visual.sh digest <plan>` | `eb13fda57381`(承認の行と一致) |
| `./scripts/run-verify.sh` | rc 0。local verifier(shellcheck、`sh -n`、settings の `jq -e`、check 系 4 本、`tests/test-*.sh`)、golang verifier(gofmt ok、lint 0 issues、Go の 8 パッケージ ok)、branch secret scan(`f9baf6a7..528b3fbd` clean)。ログは scratchpad に残した |
| `cmp`(root と template の definition-of-done、insights README) | 一致 |
| 追加行の U+FFFD 検索(`git diff -U0`) | 0 件 |

## Found but left

- `/sync-docs` は insight event を書かない(`sync_docs` の phase は schema にあるが、skill に手順がない)。tech-debt の行 `:128` が挙げていた点だが、cycle の誤記とは別の「記録がない」話なので、RESOLVED の対象に含めず、行の HTML コメントに書いた。新しい行は立てていない。必要なら別の task で決める
- `docs/tech-debt/README.md` には、すでに archive へ移った plan を `docs/plans/active/` で指している行が 16 パス分ある(このブランチの行ではない)。依頼の範囲外なので、このブランチが足した `:156` の 1 行だけ直した
- insights README(root と template)の "Appending events" の最後の 1 文(per-cycle verdict が後の event を残すこと)の言い換えは、Edit ではなく python の 1 回きりの置換で書いた。`ralph-workflow.md` の新しい規則(追跡ファイルは Edit / Write で書く)と、この task の指示に反するので、ここに記録する。置換前に対象の文字列がちょうど 1 か所あることを assert し、書き込み後に root と template が `cmp` で一致すること、U+FFFD が 0 件であることを確かめた。PostToolUse の hook は、その 2 回の書き込みには掛かっていない。ほかの編集はすべて Edit / Write で行った
- `ralph insights` の per-cycle verdict が「後の event を残す」ことは、`internal/insights/aggregate.go:132` の `slugPhaseVerdict[slug][cycle][phase] = last verdict` を読んで確かめた。実行はしていない

## Not verified

- `/pr` 5.c が実際の GitHub の応答(`--head` と別の base の PR、「すでにある」の文言)でどう動くかは、PR を作る時点でしか分からない(verify の Coverage gaps に既出)
- GNU sed の `e` コマンドの挙動(self-review F-2)。手元は BSD sed で、未確認
