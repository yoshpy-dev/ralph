# Cross-review triage report: guard-deny-only

- Date: 2026-10-08
- Plan: docs/plans/active/2026-10-07-guard-deny-only.md
- Base branch: main (f423f230)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-07-guard-deny-only.md(AC7: 旧版が deny にする形は新版でも deny。例外は見張りの一致がすべてデータ区間に収まる場合だけ)
- Self-review report: docs/reports/self-review-2026-10-07-guard-deny-only.md(2 回目は merge。データ区間は深さ 0 に限り、置換とリダイレクト先は区間から外す、と確かめていた)
- Verify report: docs/reports/verify-2026-10-07-guard-deny-only.md(pass。比較はテストの例の集まりの中だけ)
- Implementation context summary: S2c は、旧版の 4 規則を見張りとして生の文字列に当て、一致がデータ区間に収まるときだけ通す。指摘の 3 件はどれも、データ区間と判定した場所が実際には実行される形で、旧版が deny にしていたものを新版が通す。AC7 の趣旨に反するので、3 件とも直す
- 再現(2026-10-08、`/private/tmp/.../scratchpad/xr1/` の probe、jq あり・なしの両方): 新版 none、旧版 deny
  - 指摘 1: `(echo 'git push --force') | sh`、`{ echo 'sudo ls'; } | sh`、`{ echo 'git reset --hard'; } > run.sh`。同じ型として `if`・`for`・`case`・関数の出力を `sh` に流す形と、`echo 'sudo ls' | if true; then sh; fi`
  - 指摘 2: `cat <<'OUT' "$(` の改行のあとに `git push --force`、`)"`、`x`、`OUT` が続く形。bash・zsh・dash・sh のどれでも `$(...)` の中が実行される
  - 指摘 3: `cat <<EOF` のあとに `EO\` と `F` の 2 行、続けて `git push --force`。bash・zsh・sh は 2 行をつないで区切りと見なし、続く行を実行する(dash はつながない)。`<<-EOF` でも同じ

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] サブシェル・ブレースの中の単純コマンドに、外側のパイプとリダイレクトが分かる前にデータ区間を与える。`(echo 'git push --force') \| sh` を新版は止めない | 再現した。`if`・`for`・`case`・関数・パイプの先の `if` でも同じ。グループや複合コマンドが 1 つでもあるコマンドではデータ区間を与えない(見張りが旧版どおりに決める)形にすれば、型ごと閉じる | `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh`、`tests/test-pre-bash-guard.sh` |
| 2 | [P1] `$(...)` の中の改行で、外側のコマンドの未読のヒアドキュメントを読み始める。実行される置換の中身がヒアドキュメントの本文として扱われ、見張りから外れる | 再現した。実際の shell は置換の中を実行する。未読のヒアドキュメントは読み始めた文脈のものとして持ち、入れ子の文脈の改行では読まない。持ち方が崩れる場合はデータ区間を与えない | 同上 |
| 3 | [P1] 区切りに引用符のないヒアドキュメントで、バックスラッシュと改行の組を外さずに区切りの行と比べる。`EO\` と `F` の 2 行で終わったあとのコマンドが本文として扱われる | 再現した(bash・zsh・sh)。引用符のない区切りでは、行末のバックスラッシュと改行をつないでから区切りと比べる。つないだ本文にはデータ区間を与えない | 同上 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
