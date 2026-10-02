# Verify report: codex-config-model-gpt-5-6-sol

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md
- Verifier: verifier subagent (Claude)、cycle 1
- Branch: chore/codex-config-model-gpt-5-6-sol、HEAD cd73f252(push 済み)、base main 47345820
- Scope: 仕様適合(AC-1〜AC-5 を該当行と実行結果で確認)、静的解析、文書のずれ。振る舞いのテスト suite は /test の担当なので verdict には使っていない。AC-4 と AC-4b は実装者の実行記録(log)を読んで確認した。codex の認証つきの再実行はしていない
- Evidence: `docs/evidence/verify-2026-10-02-094930.log`(full)、`docs/evidence/verify-2026-10-02-094954.log`(default)。どちらも `docs/evidence/*.log` が gitignore のため commit には含まれない。scratch の probe 出力は repo の外に置いた

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS(rc 0) | shellcheck と `sh -n` の全項目 OK(`    OK` が 30 行、FAIL なし)。check-sync は IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5。check-pipeline-sync OK、check-skill-sync は 13 skill で lock-step、check-template-purity OK。golang は gofmt ok、0 issues。branch secret scan は `47345820..cd73f252` で clean |
| `./scripts/run-static-verify.sh`(default スコープ) | PASS(rc 0) | `Requested scope: changed` が `Language scope: full fallback (unclassified:.codex/config.toml)` になり、golang が選ばれて gofmt ok / 0 issues。`./scripts/detect-changed-languages.sh` の直接実行は `scope=full`、`reason=unclassified:.codex/config.toml`。branch は push 済みで upstream が HEAD と同じだが、`no_changes` にはなっていない(PR #195 の merge-base 化が効いている)。golang が選ばれた直接の理由は未分類ファイルによる full fallback で、この branch に Go の差分はない |
| `cmp .codex/config.toml templates/base/.codex/config.toml` | PASS(rc 0) | byte 一致。sha1 はどちらも `ad3c012e5a3dd4152fcad4128c1aa047de869896` |
| `./scripts/check-sync.sh` | PASS(rc 0) | 上と同じ集計、`PASS: all files in sync.` |
| `./scripts/check-template-purity.sh` | PASS(rc 0) | template の新しいコメントに meta-repo 固有の参照なし。コメントが触れるのは `.codex/agents/`、`/cross-review`、`codex-cli` の版、`user-level config.toml` で、どれも scaffold 先にもある |
| TOML の parse(`python3` の `tomllib`)、HEAD と `git show main:` の比較 | PASS | 下の AC-1 / AC-2 の節。root と template の parse 結果が一致 |
| 新しいコメント行の幅(79 桁超)と U+FFFD、`no multi-agent` などの旧語句の残り | PASS | 79 桁超の追加行は 0、追加行の U+FFFD は 0、旧語句は 2 ファイルとも 0 hit |

## AC-1 / AC-2: TOML の parse と main との比較

`tomllib` で HEAD と main の root / template を読み、キーを `a.b.c` の形に平たくして比べた。

| ファイル | キーの集合 | 値が違うキー | 3 つの model の値(HEAD) |
| --- | --- | --- | --- |
| `.codex/config.toml` | main と同じ(11 キー) | `model`、`profiles.work.model`、`profiles.review.model` の 3 つだけ(いずれも `gpt-5.5` から `gpt-5.6-sol`) | すべて `gpt-5.6-sol` |
| `templates/base/.codex/config.toml` | main と同じ(11 キー) | 同じ 3 つだけ | すべて `gpt-5.6-sol` |

- root と template の parse 結果は一致する
- `#` で始まらない変更行は、`git diff main...HEAD -U0` で `model = "gpt-5.5"` から `model = "gpt-5.6-sol"` への 3 行だけ
- 0c45e78d(Slice A)と dadc1b40(Slice B)の `.codex/config.toml` を parse した結果は、どちらも HEAD と同じ。Slice B と inline の `cd73f252` は parse 結果を変えていない
- codex 自身も、この worktree の `.codex/config.toml` を読んで `model: gpt-5.6-sol` を選んでいる(実装者の log と、下の独立 probe)

## AC ごとの verdict

| AC | Verdict | 根拠 |
| --- | --- | --- |
| AC-1 3 か所が `gpt-5.6-sol`、両ファイルが byte 一致、check-sync が green | Pass | 上の節と `cmp` rc 0、check-sync rc 0 |
| AC-2 TOML として読める。値のほかを変えていない(`git diff` で 3 行の置換だけ) | Pass(文言は Slice B で拡張) | 読める(`tomllib` と codex の読み込み)。parse した値で違うのは 3 つの model だけ、設定行の変更も 3 行だけ。ただし AC の文言「コメント、空行、並びを変えていない」は HEAD では成り立たない。各ファイルは +18 / -9 で、増減はコメント行だけ。これは Slice B(M-1、L-2 の直し)と L-4 の日付で、plan の Deviation notes に記録がある。AC 本文と Scope は書き換わっていない(下の D-2) |
| AC-3 `git grep -n 'gpt-5\.5'` に、project の codex の既定として書いた箇所が残らない | Pass | 下の分類のとおり。299 hit、51 ファイル。`.codex/`、`templates/`、`.claude/`、`.agents/`、`README.md`、`docs/recipes/`、`docs/quality/` は 0 hit。残るのは履歴、spec の記録、任意の値の fixture だけ |
| AC-4 `gpt-5.6-sol` への実際の要求が、effort 未指定と `max` の 2 通りで成功する | Pass(rc は log に残っていない) | 実装者の log(下の表)。header の `model: gpt-5.6-sol`、`reasoning effort: low` と `max`、`sandbox: read-only`。2 回とも turn が完了し(`tokens used` の行)、`-o` のファイルは `ok`(2 バイト、事前に `rm -f` 済み)。400 と `invalid_request` はなし。負の対照の `b_control.log` は、未対応のモデルなら log に 400 が出て `-o` のファイルが作られないことを示し、この確認が成功と失敗を区別できる。rc 0 は plan の Deviation notes にだけ記録があり、log には残っていない(実行スクリプトが rc を stdout に出す作りで、log に書かれない)。再実行はしていない |
| AC-4b この worktree で、既定、`--profile work`、`--profile review` の 3 通りが `gpt-5.6-sol` を選ぶ | Pass(profile の値が効いた結果ではない) | `b_default2.log`、`b_work.log`、`b_review.log` とも `workdir` は worktree、header は `model: gpt-5.6-sol`。3 本に `Ignored unsupported project-local config keys in <worktree>/.codex/config.toml: profiles` の warning があり、project の設定を codex が読んでいる(trust 済み)ことが分かる。対照の `b_control.log`(worktree の外、`-m` なし)は利用者の既定のモデル `gpt-6.1-sol` になった。したがって `gpt-5.6-sol` はこの project の設定から来ている。`--profile` の 2 通りで `gpt-5.6-sol` になったのは、トップレベルの `model` の効果。`~/.codex` の設定と trust は変えていない(実装者の記録と、下の独立 probe) |
| AC-5 `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green、`internal/scaffold` と `internal/cli` のテストが green | 一部確認 | 静的な半分は確認した(full スコープの `run-static-verify.sh` が rc 0)。`run-verify.sh` の振る舞い側と `go test ./internal/scaffold/... ./internal/cli/...` は /test の担当で、実行していない。実装者が green と記録している(plan の Deviation notes)が、私は再確認していない |

### AC-4 / AC-4b の log の集計

`/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/af447917-2575-46cc-b5ea-2212e0576047/scratchpad/i156b/impl/` の log を読んだ。

| log | 実行 | model | effort | sandbox | 400 系のエラー | `-o` |
| --- | --- | --- | --- | --- | --- | --- |
| `a4a.log` | AC-4、`-m gpt-5.6-sol`、effort 未指定、scratch から | gpt-5.6-sol | low | read-only | 0 | `ok`(2 バイト) |
| `a4b.log` | AC-4、同上に `-c model_reasoning_effort=max` | gpt-5.6-sol | max | read-only | 0 | `ok`(2 バイト) |
| `b_default2.log` | AC-4b、worktree、`-m` なし | gpt-5.6-sol | low | read-only | 0 | `ok` |
| `b_work.log` | AC-4b、`--profile work` | gpt-5.6-sol | low | read-only | 0 | `ok` |
| `b_review.log` | AC-4b、`--profile review` | gpt-5.6-sol | low | read-only | 0 | `ok` |
| `c_default.log` | Slice B 後、worktree、`-m` なし | gpt-5.6-sol | low | read-only | 0 | `ok` |
| `b_control.log` | 負の対照、worktree の外、`-m` なし | gpt-6.1-sol(利用者の既定) | low | read-only | 2(400、「ChatGPT account では未対応」) | なし |

`cfg.sha`(86c7a21a...)は Slice A(0c45e78d)の、`cfg2.sha`(5d8e76d9...)は Slice B(dadc1b40)の `.codex/config.toml` の sha1 と一致し(`git show <rev>:.codex/config.toml | shasum` で確認)、どちらも root と template が同じ。HEAD(ad3c012e...)は `cd73f252` の日付 1 行ぶん違い、その後に codex は実行されていない。変更はコメント 1 行だけで、parse 結果は変わらない。

## 独立 probe: コメントの主張を codex で確かめた

認証なしの scratch(`HOME` と `CODEX_HOME` を scratch に向け、`CODEX_HOME/config.toml` には scratch の repo を trusted にする記述だけ、git repo に HEAD の `.codex/config.toml` を置く)で、codex 0.154.0 と 0.159.2 を 15 秒の watchdog つきで起動し、header と warning を読んだ。`~/.codex` は読んでも書いてもいない。probe の前後で、main のチェックアウトと worktree の `git status --porcelain` は空。

| 版 | 実行 | model | sandbox | `Ignored unsupported project-local config keys ...: profiles. If you want these settings to apply, manually set them in your user-level config.toml.` |
| --- | --- | --- | --- | --- |
| 0.154.0 | `exec` | gpt-5.6-sol | danger-full-access | あり(2 回) |
| 0.154.0 | `--profile review exec` | gpt-5.6-sol | danger-full-access | あり(2 回) |
| 0.159.2 | `exec` | gpt-5.6-sol | danger-full-access | あり(2 回) |
| 0.159.2 | `--profile review exec` | gpt-5.6-sol | danger-full-access | あり(2 回) |

認証がないので 401 で終わるが、header と warning は起動直後に出る。`reasoning effort` は models cache がないため `none` と出た(コメントの「low」は cache のある環境の値。self-review の L-4 と同じ指摘で、日付がコメントに入った)。

## コメントの主張と、コード・証拠の突き合わせ

`.codex/config.toml`(template も同じ)の新しいコメント。

| 主張(位置) | 確認 | 結果 |
| --- | --- | --- |
| `.codex/agents/` の agent は model を指定しない(`:12-14`) | `.codex/agents/*.toml` と `templates/base/.codex/agents/*.toml` の 10 ファイルに `model` という語がない(`grep -l -w model` が rc 1) | 一致 |
| agent がこの model を引き継ぐ(`:12-14`) | `.codex/README.md:74` が「Codex runs follow the session/config model」と書く。実際に agent を動かして model を観測してはいない。session を `-m` つきで起動した場合に project の model と session の model のどちらを使うかは未確認(self-review も同じ指摘) | 静的には矛盾なし。未観測 |
| `model_reasoning_effort` を project では設定していない(`:15-16`) | `grep -n model_reasoning_effort .codex/config.toml` の hit は `:15` と `:63` の 2 行で、どちらもコメント。設定の行はない | 一致 |
| 既定の effort は `gpt-5.6-sol` で low(2026-10-02 時点)(`:15-16`) | 実装者の `a4a.log` の header が `reasoning effort: low`(認証つき、cache あり)。cache のない独立 probe では `none` | cache のある環境で一致。日付は入っている |
| codex は project の `[profiles.*]` を捨てて `Ignored unsupported project-local config keys ...: profiles` を出す(0.154.0 と 0.159.2)(`:46-53`) | 実装者の log 5 本と、上の独立 probe 4 本 | 一致 |
| `--profile review` でも read-only にならない(`:52-53`) | 独立 probe で 2 版とも `sandbox: danger-full-access` | 一致 |
| `/cross-review` は profile を使わず、`codex exec review` に `-m` と `-c model_reasoning_effort` を明示する(`:62-63`) | `.claude/skills/cross-review/SKILL.md:58` は `command codex -m "${RALPH_CODEX_REVIEWER_MODEL:-gpt-6-astra}" -c "model_reasoning_effort=…" exec review --base "$BASE" -o …`。`--profile` は、履歴(`docs/reports/`、`docs/plans/archive/`、`docs/evidence/`、`docs/insights/`)を除く repo 全体で、config のコメントとこの task の plan にしか出てこない | 一致 |

spec の注記。

| 場所 | 確認 | 結果 |
| --- | --- | --- |
| `docs/specs/2026-05-07-codex-cli-parity.md:45`(F-6) | 「(2026-10-02 に `gpt-5.6-sol` へ変更。#156)」。日付、新しい値、issue 番号とも正しい。`:47` の「profiles 定義例」は、新しいコメントの「examples」と矛盾しない | 一致 |
| 同 `:180` | F-6 と同じ括弧が付いた | 一致 |
| `docs/specs/2026-08-01-org-runtime.md:23`((e) の最後の文) | 「この文は #196 の時点の記録。その設定は 2026-10-02 に別の変更で `gpt-5.6-sol` に変わった。#156」。#196 の文を残したまま、いまの値を足している | 一致 |

## `git grep -n 'gpt-5\.5'` の分類(HEAD cd73f252)

合計 299 hit、51 ファイル。

| 分類 | ファイル / hit | 内容 | AC-3 への影響 |
| --- | --- | --- | --- |
| 履歴: 過去の plan | 10 / 56 | `docs/plans/archive/`(#196 の plan が 31 hit を占める) | なし |
| 履歴: 日付つきの evidence | 3 / 15 | `docs/evidence/`(2026-08-24、09-18、09-20 の実機記録) | なし |
| 履歴: 過去の reports | 21 / 70 | `docs/reports/` の過去分。#196(drop-gpt-5-5-default-pool)の verify / self-review / test / sync-docs / cross-review-triage が 49 hit を占め、残りは 2026-09 の他の task | なし |
| この task 自身の記録 | 2 / 16 | `docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md`(11)と `docs/reports/self-review-2026-10-02-codex-config-model-gpt-5-6-sol.md`(5)。変更前の値の説明 | なし |
| spec の記録 | 2 / 4 | `docs/specs/2026-05-07-codex-cli-parity.md:45`、`:180`(どちらも「`gpt-5.6-sol` へ変更」の括弧つき)、`docs/specs/2026-08-01-org-runtime.md:11`(`model_pool` から外した記録)、`:23`((e)、末尾に変更の注記つき)。project の codex の現在の値を `gpt-5.5` と書いた文は残っていない | なし(AC-3 が許す「spec の記録」) |
| tech-debt の記録 | 1 / 1 | `docs/tech-debt/README.md:144`(#196 の行。org skill の表から外した `gpt-5.5` の行を戻す mutation の話)。`.codex/config.toml` の値とは無関係 | なし |
| 任意の値として使う shell の fixture | 3 / 27 | `scripts/verify.local.sh`(2)、`tests/test-hook-wiring.sh`(3)、`tests/test-ralph-worktree.sh`(22)。自前の codex 設定を組む fixture で、追跡している `.codex/config.toml` を読まない | なし(plan の Non-goals) |
| 任意の値として使う Go のテスト fixture | 8 / 109 | `internal/cli/doctor_org_test.go`(62)、`cli_test.go`(15。`:43` は mock の FS で template の中身を作る、`writeCodexConfigToml` の呼び出しが 14)、`org_test.go`(13)、`config_test.go`(12)、`internal/org/verbs_test.go`(2)、`codex_session_test.go`(2)、`internal/cli/doctor_shell_alias_test.go`(2)、`internal/org/spawn_test.go`(1)。実ファイルの `.codex/config.toml` の model を assert するテストはない(`embed_test.go:215` はパスの存在確認だけ) | なし(plan の Non-goals) |
| コメントの例 | 1 / 1 | `internal/cli/doctor_shell_alias.go:627`(`-mgpt-5.5` という短縮形の例) | なし |

出荷する面(`.codex/`、`templates/`、`.claude/`、`.agents/`、`.ralph/`、`packs/`、`README.md`、`docs/recipes/`、`docs/quality/`)に `gpt-5.5` は 0 hit。

## self-review の指摘が HEAD で直っているか

| # | 状態 | 確認 |
| --- | --- | --- |
| M-1(profile のコメントが codex の挙動と合わない) | 直った | 上の「コメントの主張と、コード・証拠の突き合わせ」。主張 7 つのうち 6 つが一致し、1 つ(agent が model を引き継ぐ)は静的に矛盾せず未観測。旧語句(`switch between flows`、`used by the /cross-review skill`)は 0 hit。表と値は残してある |
| L-1(spec に project の codex の model を `gpt-5.5` と書いた文が 2 か所) | 直った | `2026-05-07-codex-cli-parity.md:180` と `2026-08-01-org-runtime.md:23` に注記が付いた |
| L-2(`no multi-agent` が古く、effort の既定に触れていない) | 直った | `no multi-agent` は 0 hit。`.codex/agents/` が model を指定せずこの model を引き継ぐこと、effort を設定していないこと、既定が `low` であることが書かれた |
| L-4(「low」に時点がない) | 直った | `:16` に `as of 2026-10-02`(root と template の両方) |
| M-2(`/cross-review` の `codex exec review` が `danger-full-access` で動く) | 延期(この PR の外) | follow-up の issue #197(OPEN、`gh issue view 197` で確認)。tech-debt の行は未追加(D-3) |
| L-3(`chore:` のコミットは release notes に出ない。effort の件を #186 に残す) | 延期 | merge 後に orchestrator が #186 にコメントする、という plan の取り決め。この段階では未実施 |

## 文書のずれ

project の codex の model と profiles について、コードや証拠と食い違う記述は、出荷する文書(`docs/recipes/codex-setup.md`、`.codex/README.md` とその template、`.claude/rules/ralph/model-routing.md` とその template、`README.md`)には見つからなかった。`codex-setup.md:93` の「`-m` と `-c` を明示し、`.codex/config.toml` の project の model に依存しない」は、今回の変更後も正しい。`codex-setup.md:164` の `[profiles.*]` は user-level の config の話で、正しい。`model-routing.md` は project の codex の model に触れていない。`README.md:174` の `config.toml  # model, sandbox, approval` は値を書いていない。次は記録しておく項目で、verdict には影響しない。

| # | 重さ | 場所 | 内容 |
| --- | --- | --- | --- |
| D-1 | LOW(記録の更新待ち) | `docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md:58-63`、`:114` | AC-1〜AC-5 のチェックボックスと「Verification artifact created」が未チェック。この report の後に orchestrator が更新する項目で、/verify は plan を編集しない |
| D-2 | LOW | 同 `:58-63`(AC-2)、`:28-32`(Scope)、`:45-48`(Affected areas)、`:34-39`(Non-goals) | AC-2 の「コメント、空行、並びを変えていない(3 行の置換だけ)」と Scope の「3 か所の変更と spec F-6 の注記」は、Slice B(コメントの書き直しと `2026-08-01-org-runtime.md` の注記)を含まない。Affected areas に `2026-08-01-org-runtime.md` がない。Deviation notes(Slice B は `:104`)には記録されているので、実態と記録は合っているが、AC 本文を読むと diff が AC より大きく見える。PR の本文で補うか、plan の AC-2 の文言を直すかは orchestrator の判断 |
| D-3 | 情報(sync-docs 待ち) | `docs/tech-debt/README.md` | self-review が sync-docs に回した 2 つの行がまだない。(1) M-1: codex が project の `[profiles.*]` を捨てる件(表を消すか、コメントだけにするかの判断)。(2) M-2: `/cross-review` の codex 経路の sandbox(issue #197)。`/sync-docs` の担当で、この branch の diff には入っていない |
| D-4 | 情報(メンテナの判断) | `docs/specs/2026-05-07-codex-cli-parity.md:77` | ユーザーストーリー 4 が「`.codex/config.toml` の profiles を増やせば現行 ralph フローのまま Codex 中心に切り替えられる」ことを望みとして書いている。codex が project の profiles を捨てる実測と合わない。2026-05 時点の要望の記録で、F-6(`:45`、`:47`)は「定義例」と書いており新しいコメントと矛盾しない。注記を足すかはメンテナの判断 |

## Coverage gaps

- 認証つきの codex の再実行はしていない。AC-4 と AC-4b は実装者の log(7 本)の読みと、認証なしの独立 probe(header と warning まで)に基づく。codex の終了コード 0 は plan の記録に頼っていて、log には残っていない
- `cd73f252`(L-4 の日付 1 行)の後に codex は実行されていない。変更はコメント 1 行で、parse 結果は変わらない
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` の振る舞い側、`go test ./internal/scaffold/... ./internal/cli/...`、fresh scaffold(`ralph init`)が `gpt-5.6-sol` になること、`tests/test-ralph-worktree.sh` は /test の担当で実行していない。実装者が green と記録している
- `.codex/agents/` の agent が実際にこの model を引き継ぐことは観測していない(agent に model の指定がないことだけ確認した)
- 利用者の `~/.codex` に `[profiles.work]` や `[profiles.review]` があるかは、読まない制約のため未確認。ある場合、`--profile` はその値を使う
- models cache のない環境で、codex が effort を指定せずに送ったときの server 側の既定は確かめていない。コメントの「low」は cache のある環境の値
- 下流の project で `.codex/config.toml` に手を加えている場合の `ralph upgrade` の drift 報告は実行していない(plan の Risks にある、従来どおりの挙動)
- M-2 の `/cross-review` が `danger-full-access` で動く件と、`-c sandbox_mode=read-only` を足したときに `-o` が書かれるかは、この PR の外(issue #197)

## Verdict

- Verdict: PASS(CRITICAL / HIGH / MEDIUM の指摘なし。LOW は文書の記録のずれ 2 件)。/test へ進める
- Verified: AC-1、AC-2(値と設定行。文言の拡張は D-2)、AC-3、AC-4、AC-4b。静的解析(full と default の両スコープ)。コメントの主張 7 つのうち 6 つ(うち profile に関する 2 つは 2 版の codex で独立に確認)。spec の 3 か所の注記。self-review の M-1、L-1、L-2、L-4 が HEAD で直っていること。M-2(issue #197)と L-3(merge 後に #186)が延期であること
- Partially verified: AC-5(静的は確認、`run-verify.sh` の振る舞い側と Go のテストは /test)。agent が model を引き継ぐ主張(静的に矛盾なし、未観測)
- Not verified: 上の Coverage gaps の項目
