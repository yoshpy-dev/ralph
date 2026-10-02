# Self-review report: codex-config-model-gpt-5-6-sol

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md
- Branch: chore/codex-config-model-gpt-5-6-sol(HEAD 37b6879c、コードのコミットは 0c45e78d)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(正しさ、不要な変更、安全性、保守性、文の正確さ)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いた 3 ファイル

## Evidence reviewed

- `git diff main...HEAD --numstat`(plan を除く): `.codex/config.toml` +3 / -3、`templates/base/.codex/config.toml` +3 / -3、`docs/specs/2026-05-07-codex-cli-parity.md` +1 / -1。設定の 2 ファイルで変わった行は `model = "gpt-5.5"` から `model = "gpt-5.6-sol"` への 3 行だけで、コメント、空行、並びは変わっていない
- `cmp .codex/config.toml templates/base/.codex/config.toml`: 一致
- `tomllib` で両ファイルを読んだ。トップレベルのキーは `approval_policy`、`features`、`model`、`profiles`、`sandbox_mode`、`tui`、`web_search` で、3 つの `model` はどれも `gpt-5.6-sol`
- codex の project 設定の `model` や `profiles` を読む Go のコードはない。`toml:"model"` は `ralph.toml` の `[org].model_pool` のエントリ(`internal/config/config.go:105`)だけ
- `/cross-review` の codex 経路は `command codex -m … -c model_reasoning_effort=… exec review` で、`--profile` を渡さない(`.claude/skills/cross-review/SKILL.md:58`、`:167`。`.agents/` と template の写しも同じ)
- `.codex/agents/*.toml` はどれも `model` を指定していない
- `git grep -n 'gpt-5\.5'`(`docs/reports/`、`docs/plans/archive/`、`docs/insights/` を除く)で残るのは次のものだけ。plan、spec の記録、`docs/evidence/` の日付つきの記録、テストと fixture(`internal/**/*_test.go`、`scripts/verify.local.sh`、`tests/test-hook-wiring.sh`、`tests/test-ralph-worktree.sh`)、`internal/cli/doctor_shell_alias.go:627` のコメントの例。今の project の設定として書いているのは L-1 の 2 か所

### probe

scratch に git repo を作り、この branch の `.codex/config.toml` を置いた。`HOME` と `CODEX_HOME` を scratch に向け、その `config.toml` には scratch の repo を trusted にする記述だけを書いた(認証なし、models cache なし)。`~/.codex` は読んでも書いてもいない。

| probe | 実行 | 結果 |
| --- | --- | --- |
| P1 | codex 0.154.0(PATH の先頭)、`exec --sandbox read-only 'say ok'` | header は `model: gpt-5.6-sol`。`warning: Ignored unsupported project-local config keys in <scratch>/.codex/config.toml: profiles. If you want these settings to apply, manually set them in your user-level config.toml.` 認証がないので 401 で終わる |
| P2 | 0.154.0、`--profile review exec --sandbox read-only` | P1 と同じ warning。ユーザー設定に `[profiles.review]` がなくてもエラーにならず、トップレベルの値で動く |
| P3 | 0.154.0、`--profile review exec`(`--sandbox` なし) | header は `sandbox: danger-full-access`。`[profiles.review]` の `sandbox_mode = "read-only"` は効かない |
| P4 | codex 0.159.2(同じマシンにある新しい版)、P2 と同じ | 同じ warning。新しい版でも project の `[profiles.*]` は捨てられる |

probe の前後で、main のチェックアウトの `git status --porcelain` は空のまま。

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M-1 | MEDIUM | maintainability(配る設定のコメント) | この diff は `[profiles.work]` と `[profiles.review]` の `model` も書き換えたが、codex はこの 2 つの表を project の設定としては読まずに捨てる(P1〜P4、0.154.0 と 0.159.2)。表の上のコメントは、`codex --profile <name>` で flow を切り替えられる(`:45`)、`[profiles.review]` は read-only の review 用で `/cross-review` が使う(`:54`)と書いている。どちらも事実と違う。`/cross-review` は `--profile` を渡さない。`codex --profile review` で起動しても、sandbox はトップレベルの `danger-full-access` になる(P3)。定義のない profile 名を渡してもエラーにならないので、利用者は warning を読まない限り気づかない。このファイルは core として `ralph upgrade` で全 scaffold に届く。read-only のつもりで `--profile review` を使った利用者は、制限のない sandbox で codex を動かすことになる。誤ったコメントはこの diff より前からある。ただし、この diff は効かない値を新しいモデル名に更新したので、表が生きているように見える状態が続く。plan の Deviation notes は「tech-debt に記録し PR に書く」としているが、この branch の `docs/tech-debt/README.md` にはまだ行がない | `.codex/config.toml:44-57`(template も同じ)、`.claude/skills/cross-review/SKILL.md:58`、P1〜P4、`docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md:102` | この PR の 3 行の置き換えはそのままでよい(`gpt-5.5` を残すと grep の掃除がずれる)。この PR では、tech-debt に 1 行を足し(sync-docs か orchestrator)、PR の本文に「project の `--profile review` は read-only にならない」と書く。直すのは別の PR にし、表を消すか、ヘッダーのコメントを「codex は project の `[profiles.*]` を読まない。使うならユーザー設定に写す」に書き換える。どちらにしても `:54` の「used by the /cross-review skill」は消す。spec F-6 の「profiles 定義例」(`docs/specs/2026-05-07-codex-cli-parity.md:47`)にも同じ注記を足す。tech-debt の行の trigger は codex の版ではなく「次にこのファイルを触るとき」にする(0.159.2 でも同じため) |
| L-1 | LOW | 文の正確さ | project の codex のモデルを `gpt-5.5` と書いた文が 2 か所に残る。(1) `docs/specs/2026-08-01-org-runtime.md:23` の (e) の最後の文「`.codex/config.toml` の `model = "gpt-5.5"` は codex CLI 自体の既定のモデルで `model_pool` とは別の設定なので、この判断の対象外」。#196 で同じ日に書いた文で、現在形で読める。この PR のあとは、ファイルにその値がない。(2) `docs/specs/2026-05-07-codex-cli-parity.md:180` の「主要 OQ はすべて確定済み: model = `gpt-5.5`」。F-6(`:45`)には注記を足したが、こちらにはない | 両ファイルの該当行、`git grep -n 'gpt-5\.5'` | (1) に「(同日、#156 の別の PR で `gpt-5.6-sol` に変えた)」のような括弧を足す。(2) は当時の決定の記録なので、残すなら F-6 と同じ括弧を足す。どちらも /sync-docs で扱える |
| L-2 | LOW | 読みやすさ(変えた行の直上のコメント) | `:12-14` のコメントは、モデルを選んだ理由を「ralph は post-implementation pipeline を順に動かす(multi-agent ではない)ので、長い構造化された編集ができるモデルが要る」と書く。Codex でも pipeline は `.codex/agents/` の custom agent で動き、どの agent も `model` を指定しないので、このモデルを引き継ぐ。「no multi-agent」はこの diff より前から古い。また、`gpt-5.6-sol` の既定の effort は `low` で、`gpt-5.5` の `medium` より低い。ユーザー設定で effort を書いていない利用者が、このコメントから low で動くことを知る手がかりはない | `.codex/config.toml:12-15`、`.codex/agents/*.toml`(`model` なし)、`.claude/rules/ralph/subagent-policy.md:84`、plan の Risks | M-1 の follow-up で一緒に書き換える。例:「Codex の custom agent(`.codex/agents/`)は model を指定せず、このモデルを使う。effort はユーザー設定に任せる(`gpt-5.6-sol` の既定は `low`)」 |
| L-3 | LOW | 申し送りの経路 | plan の Risks は、既定の effort が `low` に下がることを「PR と release の説明で触れる」としている。実装のコミット 0c45e78d は `chore:` で始まるので、release notes の自動生成から外れる(`.goreleaser.yml:36-39`)。残るのは merge コミットの行だけになる。release notes を手で書く #186 に申し送らないと、PR の本文を読まない Homebrew の利用者には届かない。#196 の self-review の L-5 と同じ形 | `.goreleaser.yml:33-39`、`git log --format=%s -1 0c45e78d` | PR の本文に加えて、#186 に 1 行のコメントを残す。内容の例:「project の codex のモデルを `gpt-5.6-sol` に変えた。ユーザー設定で `model_reasoning_effort` を書いていない環境では、project の codex と Codex 側の pipeline の agent が `medium` ではなく `low` で動く。必要ならユーザー設定で effort を指定する」 |

CRITICAL と HIGH はない。

## 依頼された観点への回答

- diff は意図した値の変更だけか、root と template は一致するか: 一致する。設定の 2 ファイルとも、変わったのは `model` の 3 行だけで、コメント、空行、並び、キーの集合は変わっていない。spec の変更は F-6 の 1 行に括弧を足しただけ。
- profile の値を残すことが読み手を誤らせるか: 誤らせる。原因はこの diff より前からあるコメント(M-1)で、この diff が持ち込んだ問題ではない。値を `gpt-5.6-sol` にそろえたこと自体は正しい。この PR では tech-debt の行と PR の本文で扱い、直すのは別の PR にすることを勧める。
- `gpt-5.5` を project のモデルとして書いた箇所や、profile が効く前提の箇所: L-1 の 2 か所と、設定ファイル自身のコメント(M-1)。`docs/recipes/codex-setup.md`、`.codex/README.md`(+ template)、skill、`model-routing.md` には、どちらもない。`codex-setup.md:164` の `[profiles.*]` はユーザー設定の話で、正しい。
- F-6 の注記: 日付、新しい値、issue 番号とも正しい。spec のほかの行と同じく、半角の括弧の前に空白を置いている。同じ spec の `:180` にも同じ値が残る(L-1)。
- effort の risk: plan の Risks の記述は正確で、effort を固定しない理由(ユーザー設定を上書きしない)も書いてある。project の codex が `medium` で動くことを前提にした記述やコードは repo にない(`git grep` で `model_reasoning_effort` と `medium` を探した。見つかったのは skill の `-c model_reasoning_effort=xhigh` の明示だけ)。足りないのは release notes への経路(L-3)と、設定のコメントで触れていないこと(L-2)。

## Positive notes

- 置き換えは最小限で、3 か所を同じ値にそろえたので、`git grep` の掃除で取りこぼしが出ない
- root と template の byte 一致を保っている
- implementer は profile が捨てられることを観測して plan に記録し、AC-4b の `--profile` の結果をトップレベルの `model` の効果だと正しく解釈した。P1〜P4 は、この観測を別の `CODEX_HOME` と新しい版でも再現した

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| codex は project の `.codex/config.toml` の `[profiles.*]` を読まずに捨てる(0.154.0、0.159.2)。root と template の `[profiles.work]` / `[profiles.review]` は効かない。コメント(`:45`、`:54`)は `--profile` で切り替えられること、`/cross-review` が review の profile を使うことを書いているが、どちらも事実と違う | `--profile review` を read-only のつもりで使うと、トップレベルの `danger-full-access` で動く。定義のない profile 名もエラーにならない | この PR の scope はモデルの値の変更だけ。表を消すかコメントを直すかは、配る core ファイルの意味を変えるので別の PR で決める | 次に `.codex/config.toml` を触るとき | この report の M-1、plan の Deviation notes |

_(この行は report の中だけにある。この phase では report と insights の events のほかを stage しないので、`docs/tech-debt/README.md` への追記は sync-docs か orchestrator に任せる。)_

## Known gaps

- P1〜P4 は認証のない環境で、header と warning までしか見ていない。要求が成功するかどうかは、plan の AC-4 の記録(利用者の認証で実行)に頼っている
- models cache のない環境では、header が `reasoning effort: none` になった(P1)。そのとき server がどの effort を使うかは確かめていない
- 利用者のユーザー設定に `[profiles.work]` や `[profiles.review]` があれば、`--profile` はその値を使う。AC-4b の `--profile` の 2 回が `gpt-5.6-sol` になったのは、ユーザー設定にその profile がないか、あっても `model` を書いていないためだと思われる。`~/.codex` を読まない制約があるので未確認

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。M-1 はこの diff より前からある問題で、merge を止めない。ただし、PR を作る前に tech-debt の行と PR の本文の説明を足す
- Follow-ups: (1) M-1 の tech-debt の行を `docs/tech-debt/README.md` に足す(sync-docs)。(2) L-1 の 2 か所に括弧を足す(sync-docs)。(3) PR の本文に、既定の effort が `low` になることと、project の `--profile review` が read-only にならないことを書く。(4) #186 に L-3 の 1 行を残す。(5) 別の PR で `[profiles.*]` を消すかコメントを直し、`:12-14` のコメントも書き換える(M-1、L-2)
