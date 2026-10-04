# sync-docs report: subagent-model-defaults

## Cycle 1

- Date: 2026-10-04
- Plan: `docs/plans/active/2026-10-04-subagent-model-defaults.md`
- Pipeline cycle: 1 of 2。差分は `main` の `11602fed` から branch HEAD `7e9680c1`(chore/subagent-model-defaults)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-04-subagent-model-defaults.md`(`9b6b491e`、addendum `89ea2a43`。merge 可)、
  `docs/reports/verify-2026-10-04-subagent-model-defaults.md`(`26bd54e4`。pass、D-1 をこの step に渡した)、
  `docs/reports/test-2026-10-04-subagent-model-defaults.md`(`c8eb7e81`。pass)

## Summary

この変更が配る文書(`model-routing.md` の tier 表と pin の記述、escalation の段落、agent の frontmatter)は、HEAD 時点で互いに一致している。ずれていたのは verify の D-1 が見つけた `.codex/README.md` の 1 文だけで、実装の側は直さずに、その文を直した。直し方は verify が挙げた 2 案のうち、モデル名を書かず `model-routing.md` を指す案(2)にした。値の写しを 1 つ減らすので、次に割り振りを変えても同じずれは起きない。plan は verify の D-3 に合わせて AC-2 の文面と、Affected areas、調査の節を直した。

## Changes made

| File | Change |
|------|--------|
| `.codex/README.md`、`templates/base/.codex/README.md` | 74 行目の implementer の説明。「the `sonnet` tier applies to the Claude Code counterpart (`.claude/agents/implementer.md`)」を「the Claude Code counterpart (`.claude/agents/implementer.md`) takes its model from its frontmatter, with the tier table in `.claude/rules/ralph/model-routing.md`」に変えた。文の残り(「Like the other Codex custom agents, no per-agent model is pinned here」と「Codex runs follow the session/config model」)はそのまま。両コピーは `cmp` で byte 一致 |
| `docs/plans/active/2026-10-04-subagent-model-defaults.md` | (1) AC-2 の `diff` の期待を「org runtime の節と『Where the values live』の 1 行だけ」から「org runtime の節と、『Where the values live』にある root だけの bullet(既存の `defaults_sync_test.go` の bullet と新しい `tests/test-agent-models.sh` の bullet)だけ」に直した。チェックボックスは未チェックのまま。(2) Affected areas に `.codex/README.md` と template を足した。(3) 調査の節に、README の 74 行目が計画時の調査で拾えず /verify の D-1 で見つかった旨の bullet を足した |
| `docs/reports/sync-docs-2026-10-04-subagent-model-defaults.md` | この report |

README の書き換え後の文:

> Like the other Codex custom agents, no per-agent model is pinned here — the Claude Code counterpart (`.claude/agents/implementer.md`) takes its model from its frontmatter, with the tier table in `.claude/rules/ralph/model-routing.md`; Codex runs follow the session/config model

`.claude/rules/ralph/model-routing.md` は scaffold にも入る(`templates/base/.claude/rules/ralph/model-routing.md`)ので、利用先でも参照先が存在する。

AC-2 の文面を直した根拠は root と template の `model-routing.md` の `diff` で、差分は 3 hunk。org runtime の節の 2 hunk(`:94-104` と `:108-125`)と、root にだけある「Where the values live」の `defaults_sync_test.go` の bullet 1 行と `tests/test-agent-models.sh` の bullet 3 行(`:133-136`)。

## Surfaces checked for drift

`git grep -nwE 'sonnet|opus|haiku'` をリポジトリ全体に当て、指示された除外(`docs/plans/archive`、`docs/reports`、`docs/evidence`、`docs/insights`)を除いた。`*.go` と `docs/specs` は最初の検索から外していたので、別に当て直した。

| Surface | Finding |
|---------|---------|
| `.codex/README.md:74`(+ template) | D-1。直した |
| `.claude/rules/ralph/model-routing.md`(+ template) | 一致。tier 表(`:8-13`)、pin の記述(`:23`、`model: opus`)、「Overriding a seat's default」(`:49-52`)。ヒットした残りの行(`:55` の `RALPH_CLAUDE_REVIEWER_MODEL` の `opus` fallback、`:67` の alias 一覧、`:79` の `haiku`)は seat の既定とは別の記述 |
| `.claude/agents/*.md`(+ template) | implementer、verifier、tester が `opus`、reviewer と doc-maintainer が `sonnet`(いずれも `:5`) |
| `README.md`、`AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/subagent-policy.md`、`.claude/skills/work/SKILL.md`、`docs/quality/`、`docs/recipes/`、`docs/architecture/` | seat のモデル名を書いていない。旧 tier 表の見出し(`Judgment seats`、`Procedural seats`)も、tier を名指す語(`sonnet tier` など)も、`.codex/README.md:74` と plan 自身以外に残っていない |
| `.claude/skills/cross-review/SKILL.md` ほか 3 コピー | `${RALPH_CLAUDE_REVIEWER_MODEL:-opus}` の fallback。cross-review の claude レビュアーで、plan の Non-goals。変えない |
| `scripts/ralph-config.sh:35,57,88`(+ template) | `RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_ORG_MODEL_POOL`、`RALPH_ORG_WATCHDOG_WATCHER_MODEL`。別の仕組み。変えない |
| `.claude/skills/org/SKILL.md:81-83,136` ほか 3 コピー | org runtime の `--model` の選択肢と `spawn` の例(`--model sonnet`)。座席の話。変えない |
| `templates/base/ralph.toml:25-33,53-54,91` | `model_pool` と `[org.roles]` のコメントの例(`implementer = ["sonnet"]`)、`watcher_model`。変えない |
| `templates/base/docs/insights/README.md:32,67` | 過去の insight event の `requested_model` の例。変えない |
| `docs/tech-debt/README.md:49,151` | `:49` は撤去済みの loop driver の解決済みの行(`RALPH_SELF_REVIEW_MODEL=opus` の記述は当時の事実)。`:151` はこの plan の行 |
| `docs/specs/` | `2026-04-16-ralph-cli-tool.md:238` と `2026-05-07-codex-cli-parity.md:13` は `ralph.toml` の `model = "claude-opus-4-7"`(CLI の既定)。`2026-08-01-org-runtime.md:11,23` は org の `model_pool`。どれも pipeline の agent とは別で、日付つきの spec の記録 |
| `internal/**/*.go` と `testdata` | org runtime と `ralph.toml` の既定値(`internal/config/config.go`、`internal/org/envelope_summary.go`)、および fixture。`.claude/agents/*.md` の `model:` を参照するコードはない |
| `tests/test-insights-append.sh`、`tests/test-ralph-config.sh` | insight event の引数と `RALPH_CLAUDE_REVIEWER_MODEL` の既定値の検査。別の仕組み |

verify の判断(D-1 のほかに seat のモデルを書いた文書はない)を、検索を当て直して確認した。

## Not done in this pass

- plan の Progress checklist と AC のチェックボックス(verify の D-2)は未更新のまま。orchestrator の担当。今回の指示でも「チェック状態はそのまま」とされている。
- plan の Progress notes に sync-docs の行は足していない。この commit の SHA が要るので orchestrator が足す。
- `docs/tech-debt/README.md:151` が参照する `docs/plans/archive/2026-10-04-subagent-model-defaults.md`(verify の D-4)は /pr が plan を archive に移すと解消する。今は前方参照。
- insight event(`./scripts/insights-append.sh`)は追記していない。今回の指示に含まれず、verify と test の report も追記していない。
- verify の Coverage gaps の「D-1 の種類のずれを防ぐ検査はない」は、モデル名を README から外したので、この種のずれは起きなくなった。検査は足していない。
- Claude Code が新しい frontmatter のモデルで subagent を起動するか(Task のモデル解決)は静的に確かめられない。マージ後に確認する項目で、verify の Not verified と同じ。
- `docs/tech-debt/README.md` は `docs/plans/active/` を参照する行が他にも多く、archive 済みの plan を指すものがあると思われるが、今回は行っていない(この task が触る行ではない)。

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-sync.sh` | exit 0。IDENTICAL 159、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(集合は変わらない) |
| `./scripts/check-skill-sync.sh` | exit 0。13 skill が一致 |
| `cmp .codex/README.md templates/base/.codex/README.md` | exit 0。byte 一致 |
| `sh tests/test-agent-models.sh` | exit 0。47 / 47 |
| `./scripts/check-template.sh` | exit 0 |
| `./scripts/check-template-purity.sh` | exit 0。template に meta-repo 固有の参照なし |
| `go test -count=1 ./internal/scaffold/ ./internal/config/` | ok(埋め込み template と skill の fallback) |
| `./scripts/secret-scan.sh --staged` | 引き継ぎのメッセージに記録(commit 時) |
| `./scripts/secret-scan-branch.sh --strict` | 引き継ぎのメッセージに記録(push 前) |

## Diff size (for /pr)

`git diff main...HEAD`(この pass の編集を含む前の HEAD `7e9680c1`)は 16 files changed、1202 insertions、22 deletions。`docs/` 以外は 762 行追加、22 行削除で、そのうち `tests/test-agent-models.sh` が 733 行追加。walkthrough を書くかどうかは /pr が決める。この step では書いていない。

## Files changed in this pass

- `.codex/README.md`、`templates/base/.codex/README.md`
- `docs/plans/active/2026-10-04-subagent-model-defaults.md`
- `docs/reports/sync-docs-2026-10-04-subagent-model-defaults.md`(この report)
