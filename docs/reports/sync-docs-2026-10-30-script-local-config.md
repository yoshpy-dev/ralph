# sync-docs report: script-local-config

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-10-30-script-local-config.md`(issue #276 の最後 B2。承認の digest `cdd1e5ba1234` は run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。plan の編集は `## Progress checklist` の 1 行だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json`)。branch は `feat/script-local-config`。run 開始時の HEAD は `a42c50e3`。変更と report は 1 つの commit に入れる(件名 `docs: sync the script-local-config docs`)
- 先行 report: `docs/reports/self-review-2026-10-30-script-local-config.md`(Merge: yes。L1・L2・L3・L5 は 00c494cc で直し、L4 はこの run で行にした。L6 は直さない)、`docs/reports/verify-2026-10-30-script-local-config.md`(pass。文書のずれは AGENTS.md:89、L4 の行、spec の例 I1)、`docs/reports/test-2026-10-30-script-local-config.md`(pass。Test gaps 3 つ)
- 差分の大きさ(`git diff origin/main...HEAD --stat`、`a42c50e3` の時点): 52 ファイル、+3924/-941。walkthrough を書くかどうかは `/pr` が決める
- 制約: コードは触れていない。打ったのは `check-sync.sh`、`check-skill-sync.sh`、`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh`、`git diff --check`、`plan-visual.sh digest`、`wc -l`、`git grep`、`zsh -f tests/test-ralph-config.sh`(途中で止めた。下の項目 3)。Go のテストと `run-verify.sh`・`run-test.sh` は打っていない(help の文字列にも pin のある文にも触れていないため)

## Summary

文書の本体(README の `ralph.local.toml` の節、`quality-gates.md`、model-routing.md、`/cross-review` の 4 つの写し、`config get --help`、template の `ralph.toml`)は S7 までに済んでいた。この run で直したのは次の 3 か所と、確かめて直さなかった 2 か所。

- `AGENTS.md` の repo map: `LocalOnlyKeys` の説明が `[org].state_dir` だけだった。
- `docs/architecture/repo-map.md`: 新しい `tests/lib/` を Tests 節に足した。
- `docs/quality/quality-gates.md`(と template の写し): 手元の secret scan の base が読めないとき "cannot scan" になる(`--strict` は exit 3)こと、CI は `GITHUB_BASE_REF` が先で `ralph` を打たないことを 1 文足した。S4 の挙動で、README にはあったが quality-gates になかった。
- `docs/tech-debt/README.md`: L4 の新しい 1 行、「Three test gaps of standard-flow-config」の (c) への確認の 1 文、Trigger に当たった 4 行(118、235、277、364)への持ち越しの注。

5 つの変数(`RALPH_PLAN_VISUAL_BROWSER`、`RALPH_PLAN_VISUAL_OPENER`、`RALPH_XREVIEW_BASE`、`RALPH_VERIFY_BASE`、`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN`)の名前は、読み手に設定を勧める形では、追跡している文書・rule・skill・recipe のどこにも残っていない。

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md`(89 行、`internal/config/` の行) | 項目 1。`LocalOnlyKeys` の括弧を「keys only that file may set, `[org].state_dir` today」から「keys only that file may set: `[org].state_dir` and five script keys in `[plan_visual]`, `[standard]`, and `[verify]`」に。`internal/config/local.go` の `LocalOnlyKeys()` の 6 つと合う。AGENTS.md は管理 block の外の行で、`check-sync.sh` は block だけを比べる |
| `docs/quality/quality-gates.md` と `templates/base/docs/quality/quality-gates.md`(KNOWN_DIFF なので別々に) | 「Local mirror of this same range scan」の項目の末尾に 1 文。手元の base は `ralph.local.toml` の `[standard].base_branch`、`ralph config get standard.base_branch` が失敗すれば自動の base で scan せず "cannot scan"(`--strict` は exit 3)、CI は `GITHUB_BASE_REF` が先で `ralph` を打たない。実装は `scripts/secret-scan-branch.sh` の `cannot_scan` と `GITHUB_BASE_REF` の分岐(S4 7803a530)。2 つの写しは同じ文 |
| `docs/architecture/repo-map.md`(Tests 節) | `tests/lib/` の行を足した。`fake-ralph.sh` が偽の `ralph` で、キーごとに `ralph config get` へ答え、インストール済みの `ralph` に届かないこと。62 行の `ralph-config.sh` の説明(2 つの関数、`ralph.local.toml` だけのキー)は S7 の追い a5a38aff のとおりで、実装と合っていたので変えていない |
| `docs/tech-debt/README.md`(末尾に新しい 1 行) | 項目 2。800 行を超えた 5 ファイルを 1 行にした。数は `1cc84ffa`(merge-base)と `a42c50e3` を `wc -l` で数え直し、依頼の表と一致した。Trigger には、最初に動かす単位を `internal/config/script_values.go`、`internal/cli/doctor_retired_env.go`、3 つの shell test の設定のケースとして例に挙げた。Related の plan は `docs/plans/active/` の形 |
| 同(410 行、「Three test gaps of standard-flow-config」) | 項目 3。(c) の欄に、いまも当てはまることを確かめた旨の 1 文。(a) は S7 が閉じていて、変えていない |
| 同 118 行(template の `quality-gates.md` の記述と配線の不一致) | 持ち越しの注。Trigger は「このファイルへの次のタッチ」で、S7 が template の `quality-gates.md` の changed scope の文を直したので当たった。「Must pass in CI」の節にも verify の chain にも触れていないので払うものはなく、2026-10-26 の注と同じ形で書いた |
| 同 235 行(org-gate cycle 2 の小さい所見) | 持ち越しの注。Trigger は `gate.go`・`gate_run.go`・`lockfile.go` の次の変更で、`gate_run.go` の `gateStrippedEnvPrefixes` の注釈と `gate_runverify_test.go` を直したので当たった。(a) から (i) が名指す関数とテストには触れていない |
| 同 277 行(plan-visual の lint の test gaps) | 持ち越しの注。Trigger に「`plan-visual.sh` の `--help` の文」があり、S3 が変数 2 つの説明を `Settings` の行に直したので当たった。(c) を読み直した: `lint` の行は残り、`tests/test-plan-visual.sh` の「--help lists the lint subcommand」が固定している。払うものはない |
| 同 364 行(`wip:` の commit を止めるのは leader の雛形だけ) | 持ち越しの注。Trigger は `gate.go` か `gate_run.go` の次の変更で、`gate_run.go` の注釈で当たった。gate が確かめるものと `gate_passed` の意味は変わっていないので払わない。leader の雛形の step 12・14 には触れていない |
| `docs/plans/active/2026-10-30-script-local-config.md` | Progress に sync-docs の行。本文の節は触れていない。digest は前後で `cdd1e5ba1234` |
| `docs/insights/events/2026-10-31-script-local-config.jsonl` | `sync_docs` の行(cycle 1、pass)を足した |
| `docs/reports/sync-docs-2026-10-30-script-local-config.md` | この report |

## 依頼の項目ごとの結果

| 項目 | 結果 |
|------|------|
| 1 `AGENTS.md` の `LocalOnlyKeys` | 直した。`check-sync.sh` は DRIFTED 0。AGENTS.md は template 側に同じ行がなく、`KNOWN_DIFFS` にも入っていない(block の外は比べない) |
| 2 tech-debt に L4 の行 | 足した。5 つのファイルの大きさを理由にした既存の行はなかった(`800`、各ファイル名、元の行数で grep)。したがって更新ではなく新しい行にした。5 欄、backtick と `~~` は偶数。`HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` の「tech-debt README plan references」は OK |
| 3 「Three test gaps」の (c) | 当てはまる。直さず、確かめた事実を 1 文足した。下の「判断を残したもの」 |
| 4 `docs/specs/2026-10-07-org-multi-org-director.md:123` | 触らない。下の「判断を残したもの」 |
| 掃き出し | 下の「Surfaces checked for drift」。直す必要のある当たりはなかった |
| `repo-map.md` と README の `ralph.local.toml` の節が実装と合うか | 合っている(下の表) |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 5 つの変数の出現 | 「もう読まない」の説明、doctor の表、テスト、過去の記録だけ | `git grep -nE 'RALPH_(PLAN_VISUAL_BROWSER\|PLAN_VISUAL_OPENER\|XREVIEW_BASE\|VERIFY_BASE\|VERIFY_SKIP_BRANCH_SECRET_SCAN)'`(`tests`、`*_test.go`、`docs/plans`、`docs/reports` を除く)。README:152 は「もう読まない」と doctor の warn の説明。`internal/cli/doctor.go:909-913` は表。`scripts/ralph-config.sh` と template の写し(18-20 行)は「もう読まない」の注釈。`docs/evidence/` は過去の証跡。`docs/tech-debt/README.md` の 56 行は RESOLVED の comment、225 行は RESOLVED の取り消し線の行(過去の出来事)。`docs/specs/` の 1 行は項目 4。`.claude/skills/org/SKILL.md` などの `RALPH_VERIFY_` は gate が除く前置きの説明で、5 つの変数の読み出しではない。tests の当たりはどれも「env を設定しても効かない」の確認 |
| 変数名を `RALPH_` なしで書いた形(`XREVIEW_BASE` など) | 当たりは上と同じ | `git grep -nE '(PLAN_VISUAL_BROWSER\|PLAN_VISUAL_OPENER\|XREVIEW_BASE\|VERIFY_BASE\|SKIP_BRANCH_SECRET_SCAN)'` |
| `docs/architecture/repo-map.md` 62 行 | 合っている | `ralph_setting <key>` と `ralph_setting_or_fail <key>`、標準フローの値と `[plan_visual]`・`[standard].base_branch`・`[verify]` を、読む 4 つのスクリプトの名前つきで書いている |
| README `ralph.local.toml` の節(152 行)と `config get` の行(135 行) | 合っている | 値の形は `scripts/plan-visual.sh`(`resolve_opener`、`resolve_browser`)、`detect_base_branch`(`xreview-helpers.sh`)、`detect-changed-languages.sh` の base の順、`secret-scan-branch.sh` の `GITHUB_BASE_REF` が先、と照合した。gate と regate が `gateScriptEnv`(`RALPH_LOCAL_CONFIG=off`)でスクリプトを走らせることも `internal/org/gate.go`・`regate.go` で確かめた。135 行の「5 つのキー」は 152 行を指している |
| `docs/quality/quality-gates.md`(root と template) | `[verify].base` の案内は合っていた。secret scan の base の 1 文を足した | `[verify].base` の案内と、設定の誤りで full scope になる 1 文は S7 が書いた。`RALPH_VERIFY_SCOPE` の行は別の変数。手元の secret scan の base が読めないときの扱いは、README にあって quality-gates になかった |
| `.claude/rules/ralph/model-routing.md`(root と template) | 変更なし | 「Where the values live」が 10 キー・2 関数の形。KNOWN_DIFF なので行番号は違うが内容は同じ |
| `/cross-review` skill の 4 つの写し | 変更なし | `check-skill-sync.sh` は 14 本が lock-step |
| `templates/base/ralph.toml` の注釈 | 変更なし | 5 つのキーが注釈の形で、「`ralph.local.toml` にだけ書ける」と書いている。`defaults_sync_settings_test.go` が Default() と照合する |
| `CLAUDE.md`、`.claude/rules/ralph/` の他の rule、`docs/recipes/` | 変更なし | 5 つの変数も `LocalOnlyKeys` の一覧も書いていない |
| `docs/quality/definition-of-done.md` | 変更なし | 標準フローの cap の説明だけ |
| `U+FFFD`、`git diff --check` | なし | 変えた 5 ファイルを走査 |

## 判断を残したもの

- 項目 3: (c)「`tests/test-ralph-config.sh` は `zsh -f` で動かない」は、この branch が test を書き直したあとも当てはまる。`zsh -f tests/test-ralph-config.sh` を打つと、「the table of defaults」の節の最初の検査で `ralph_setting_default <全キー> returns 0` が FAIL(expected 0、actual 1)になる。原因は変わっていない。`$KEYS`、`$SET_KEYS`、`$EMPTY_KEYS`、`$NO_VALUE_KEYS`、`$RETIRED_NAMES`、`$shells` を単語に分けて回す loop が、zsh では 1 語のままになる。書き直しは zsh を内側の shell として 4 つの shell の 1 つに加えただけで、外側の runner は sh の前提のまま。行には「いまも当てはまる」と、その原因の loop の名前を足した(行番号は書かない)。zsh の run は 1 本目の FAIL を確かめたあと止めた
- 項目 4: spec の 123 行は、7b で決めたことの記録で、gate が `RALPH_VERIFY_` で始まる変数を除く理由として `RALPH_VERIFY_BASE=HEAD` を例に挙げる。現在の挙動を how-to として案内する文ではない。gate が除くことは今も変わらず(`RALPH_VERIFY_SCOPE=full` と `RALPH_LOCAL_CONFIG=off` を足す)、`RALPH_VERIFY_BASE` はもう読まれないので、例だけが古くなった。spec は過去の決定の記録で、書き換えない。注も足していない
- L4 の行は、`internal/config/defaults_sync_test.go`(S1b で分け、620 行)を含めていない。その分は 405 行の RESOLVED の行が持つ
- Trigger の洗い出しは、branch が変えたコードのファイルの名前と、変えた関数・キーを、Trigger の欄(取り消し線で閉じたものを除く)に当てて行った。当たった候補のうち、持ち越しの注を付けたのは上の 4 行。付けなかった候補と理由は次のとおり
  - 139・141・142(`secret-scan-branch.sh`): Trigger は `allowlist_ls_tree_mode`、git の呼び出し、`use_merge_attributes` の変更。S4 が直したのは base の決め方(`detect_base_branch` の失敗の扱い)で、どれにも触れていない
  - 229・381(`run-verify.sh`): Trigger は evidence の名前、hook の文。S5 は branch の secret scan の skip だけ
  - 161(`verify.local.sh` の shellcheck の一覧): (a) は取り消し線で閉じ済み。S7 が足した `tests/lib/*.sh` は別の変更
  - 396(b)(`config.Load` の error path): S1 の `validateScriptValues` は検証の追加で、Load が error で返す値の扱い(`Default()` を返すか)には触れていない
  - 400(`local_test.go`): S1 は新しいテストを `local_script_keys_test.go` に置き、`local_test.go` に触れていない
  - 149・259(`defaults_sync_test.go`): Trigger はモデルの pool の変更と `/director` skill の受信箱の節。S1b の移動は関係しない
  - 154(`/cross-review` の codex の呼び出し)・156(`model-routing.md` の pin の段落): S4 と S7 はどちらも別の段落・手順を直した

## 直さなかったもの

| 項目 | 理由 |
|------|------|
| L6(doctor の SKIP の warn は固定文で、先頭の「write the value」と末尾の「1 のときだけ」が逆向きに読める) | self-review の判定どおり。誤った書き方は勧めなくなったので、PR の Known gaps に書く |
| verify の I2(既定の `run-verify.sh` は config の誤りで scan を飛ばしても通る)と I3(`= 1` の誤りの文にキー名がない) | 文書のずれではなく挙動。PR の Known gaps に書く |
| test の Test gaps(ブラウザ自動検出の 2 件はこの Mac で SKIP、shell のテストは偽の ralph だけで値を渡す、shell の coverage はない) | コードの穴ではなく実機の確かめ。PR の CI のログで SKIP でなく通ったかを確かめる |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 項目 4 |
| plan 本文と AC の checkbox | 触れない。確かめた事実は Progress の 1 行に書いた |

## Verification (この pass)

| コマンド | 結果 |
|---------|------|
| `./scripts/check-sync.sh` | PASS(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 本が lock-step |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(OK 33。「tech-debt README plan references」は OK) |
| `git diff --check`、`U+FFFD` の走査 | 空 |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-script-local-config.md` | `cdd1e5ba1234`(承認のときと同じ) |
| tech-debt の欄数 | 新しい行と変えた行はどちらも 5 欄 |
| `wc -l`(`1cc84ffa` と `a42c50e3`) | 800 行を超えた変更ファイルは 5 つで、依頼の表と一致(`doctor.go` 1483 → 1514、`config.go` 1065 → 1180、`test-detect-changed-languages.sh` 669 → 848、`test-plan-visual.sh` 857 → 1012、`test-secret-scan-branch.sh` 2116 → 2217) |
| `zsh -f tests/test-ralph-config.sh` | 「the table of defaults」の節で FAIL を確かめ、止めた(項目 3) |

## Insight event

`./scripts/insights-append.sh --slug script-local-config --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` は rc 0。`cycle` は 1。`docs/insights/events/2026-10-31-script-local-config.jsonl` に 1 行が足され、ほかのファイルは増えなかった(UTC の日付が run の間も 10-31 のまま)。
