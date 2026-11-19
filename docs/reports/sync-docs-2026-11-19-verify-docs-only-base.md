# sync-docs report: verify-docs-only-base

- Date: 2026-11-19
- Plan: `docs/plans/active/2026-11-19-verify-docs-only-base.md`(issue #341。承認の digest `f500fb32da2a` は、`./scripts/plan-visual.sh digest` で計算し直した値と一致する。計画の本文は変えていない)
- Branch: `fix/verify-docs-only-base`。base は origin/main `f280f2d6`、run 開始時の HEAD は `7e7cdb3d`(最新の origin/main を取り込んだ merge)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は `fa58d85f`(L1、L8、L9 が `/sync-docs` 行き。L5、L6 は known gap)、verify は `e03a33ef`(V1、V3 が `/sync-docs` 行き。V2 は `17a059be` でコードを直した)、test は `910b1063`(G1、G2、G4)。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 差分の大きさ(`/pr` が walkthrough を決める材料。docs の commit `6a05139e` を足したあと、この report を足す前): `git diff --shortstat origin/main...HEAD` は 32 ファイル、+2,657/-328。Go は +73/-35(comment の直しと gate の環境の変更、そのテスト)
- 制約: Go のコードは comment しか触っていない。`./scripts/run-test.sh` と全テストは回していない。`go test ./internal/config/`、`internal/scaffold`、`check-sync.sh`、`check-skill-sync.sh`、`check-template.sh`、`check-template-purity.sh` は回した(下の「確認」)

## Summary

handoff の 1 から 4 の項目を片付けた。docs の commit は `6a05139e` で、8 ファイルを変えた。そのあとの commit(この report の commit)で、tech-debt の 401 行目の Related の列に足し忘れた計画の参照を足した。

- V1: `VerifyConfig.Base` の doc と `templates/base/ralph.toml` の `[verify]` の説明に、両 scope で docs only かの判定にも使うこと、CI では `GITHUB_BASE_REF` が先なことを足した。`# base = ""` などの key の行は変えていない。root の `ralph.toml` に同じ文はない
- gate の環境: README の `ralph org gate` の段落と、org skill の `gate` 行(4 写し)に `GITHUB_BASE_REF` を足した。spec は変えない
- tech-debt: 9 行を触った(8 行の直しと、新しい 1 行)。下の「tech-debt」

走査(下の「Trigger の走査」)で、handoff と verify が挙げた行のほかに、この branch 自身の編集が当てた行が 4 つ見つかった。240、369、400 は gate.go と gate_run.go の変更で、182 は私が編集した org skill の `gate` 行で当たった。どれも払っていない。

## Changes made

| File | Change |
|------|--------|
| `internal/config/config.go` | `VerifyConfig.Base` の doc。コードは変えていない(1190 行が 1194 行) |
| `templates/base/ralph.toml` | `[verify]` の説明の文。key の comment 行は変えていない。root の `ralph.toml` に同じ説明はない |
| `README.md` | `ralph org gate` の段落の 1 文(292 行目あたり)に `GITHUB_BASE_REF` を足した |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | `gate` 行の 1 文。`.claude` を編集し、`./scripts/sync-skills.sh` で root の `.agents` を作り、template の 2 つは `cp` で写した(sync-skills.sh は template を作らない) |
| `docs/tech-debt/README.md` | 118、182、234、239、240、369、400、417 行の直しと、235 行目の新しい行。`git diff --numstat` は 9/8。ほかの行は変わっていない |
| `docs/reports/sync-docs-2026-11-19-verify-docs-only-base.md` | この report |
| `docs/insights/events/2026-11-19-verify-docs-only-base.jsonl` | sync_docs の 1 行 |

## gate の環境の文

org skill の `gate` 行の文は、次のとおり。

> スクリプトには、打った側の環境から名前が `RALPH_VERIFY_` か `HARNESS_VERIFY_` で始まる変数と `RALPH_LOCAL_CONFIG` と `GITHUB_BASE_REF` を除き、`RALPH_VERIFY_SCOPE=full` と `RALPH_LOCAL_CONFIG=off` を足して渡す(差分で検証の範囲を狭めず、打った人の `ralph.local.toml` で検証を変えない。`GITHUB_BASE_REF` は、スクリプトが差分と branch の secret scan の base に取るので除く)。

README は同じ内容を英語で書いた。`ralph org gate --help` の文(`17a059be`)と同じ理由を言っている。skill の文を固定するテストはない(`打った側の環境` などで `git grep` した)。行の長さは 4,804 字から 4,890 字になった。

## spec

`docs/specs/2026-10-07-org-multi-org-director.md` の 128 行目は変えない。7 段目の後半(7b)の「決めたこと」の記録で、その時点の決定(`RALPH_VERIFY_` と `HARNESS_VERIFY_` の 2 つの prefix を除く)を書く。この行は 10-30 の `RALPH_LOCAL_CONFIG` も書いていないので、今の挙動を写す節ではない。spec に gate の環境の「今の挙動」の節はなく、`RALPH_VERIFY_` などを書く場所はこの行だけだった(`git grep` で確かめた)。

## tech-debt

行番号は、編集前の HEAD(`7e7cdb3d`、origin/main を取り込んだあと)のもの。verify の report の番号から 2 つ前後している。235 行目に新しい行を入れたので、編集後は 239 以降が 1 つずつ後ろにずれる(239 は 240、240 は 241、369 は 370、400 は 401、417 は 418)。

| 行 | 項目 | 直し |
|---|---|---|
| 118(V3) | template の quality-gates の「Must pass in CI」の行 | Trigger のセルの末尾に 2026-11-19 の note。この branch は template の quality-gates の「Must pass locally」の節と template の `run-verify.sh`、`detect-changed-languages.sh` を触ったので Trigger を満たした。「Must pass in CI」の節と template の verify chain(`verify.yml` は触っていない)は変えていないので払っていない。Related に計画(`docs/plans/active/…`)と verify の V3 |
| 234(L9) | `run-verify.sh` の evidence log の名前 | 「exit code、scope、docs-only の印が per-run の `mktemp` を通る」を直した。印の `run_non_docs_file` は消え、`run-verify.sh` は detector の出力(scope file)から `docs_only` と `first_non_docs` を読む。検査は `tests/test-run-verify-concurrency.sh` の Case 5 が持つ。Trigger は evidence の名前の変更なので、当たっていない |
| 239(L1) | gate が落とさない変数 | 項目の文を直した。gate は 2 つの prefix のほかに `RALPH_LOCAL_CONFIG` と `GITHUB_BASE_REF` の名前を落とす(`gateStrippedEnvNames`)。「`GITHUB_BASE_REF` が届く」という例は偽になったので消し、ほかの変数は届く、に変えた。Impact の「2 つの prefix だけ」も直した。Trigger のセルの末尾に note(detector が `GITHUB_BASE_REF` を読み始め、gate が `17a059be` で落とすことで、この変数については払った。行は開いたまま)。Related に計画、self-review の L1、verify の V2 |
| 417(L8) | 800 行を超えた 5 ファイル | note。`tests/test-detect-changed-languages.sh` が 848 行から 1,371 行(origin/main `f280f2d6` と、branch で最後に変えた `1c3dc457` で数えた。43〜55 の節は 894 行目から)。この branch の `internal/config/config.go` の doc(1190 行から 1194 行)も当たりで、払っていない。最初の動きに、43〜55 の節を `tests/test-detect-changed-languages-docs-only.sh` へ、2 つの git の代役と 2 つの `write_manifest` を `tests/lib/` へ、を書いた。「ほかの 4 つ」は「ほかの 3 つ」にした |
| 235(新) | known gaps | L5、L6、G1、G2、G4 の 1 行。5 つのセルすべてを書き、5 つの項目それぞれに Trigger を付けた。最後の行ではなく 234 の次に置いた(ほかの PR が末尾に足す行と衝突しないため) |
| 240 | org-gate cycle 2 の小さな findings | note。`gate_run.go`(`gateScriptEnv` と `gateStrippedEnvPrefixes` の comment)と `gate.go`(`GateParams.Env` の doc)を触った。(a)〜(d)、(f)〜(i) が名指す関数とテストは触っていないので払っていない |
| 369 | `wip:` の commit | note。同じ 2 ファイルを触った。gate が何を検査するかは変えていないので、`wip:` の検査は足していない |
| 400 | `Regate` の環境のテスト | note。(a) の `gateScriptEnv` を触った。テストは `Gate` だけに掛け、`Regate` の本物の script のテストは足していない。`regate.go` の 952 行目が `gateScriptEnv(env)` を呼ぶので、`Regate` では code で守られ、テストでは守られていない。前の note(2026-11-13)の「Carried over」の文を、私の編集が一度消してしまったので戻した |
| 182 | org skill の command table | note。(a) は、私の編集した `gate` 行で当たった。`gate` 行の行全体が 4,804 字から 4,890 字(origin/main `f280f2d6` と、この pass で数えた)。`stop` と `disband` の cell は変わらず(1,208 字と 1,219 字)、table の整理は別の変更なので払っていない |

Related の列には、計画を `docs/plans/active/…` で足した。`archive/` は `/pr` の `archive-plan.sh` が書き換えるので先には書かない。

## Trigger の走査

handoff が挙げたのは 118、234、239、417 の 4 行だった。tech-debt の全行から、Trigger が全部 `~~` の行を除き、Trigger のセルを、この branch が触った物の語(`run-verify`、`detect-changed`、`detector`、`quality-gates`、`gate_run`、`gateScriptEnv`、`gate.go`、`org_gate`、`GITHUB_BASE_REF`、`config.go`、`VerifyConfig`、`ralph.toml`、`skills/org`、`/org skill`、`README`、`test-run-verify`、`test-detect`、`secret-scan-branch`、`verify.yml`、`Must pass`、`docs-only`、`verify.base`、`ralph-config.sh`、`defaults_sync`、`ralph org gate`、`regate`、`800`、`scaffold`、`manifest.toml`、`check-sync`、`ralph_setting` など)で探した(`sweep.py`、scratch)。当たったのは 104 行で、Trigger のセルの頭(Trigger が名指す物)を、関係がありそうな行はすべて読んだ。

| 行 | 判断 | 理由 |
|---|---|---|
| 118、234(文のみ)、239、417 | 当たり | 上のとおり。234 は Trigger に当たらず、項目の文だけが古かった |
| 240、369、400 | 当たり。note を足した | `gate.go` の doc と `gate_run.go` の `gateScriptEnv` を触った。verify の表と handoff の両方に載っていなかった行。(a)〜(d)、(f)〜(i)、`wip:` の検査、`Regate` の本物の script のテスト、どれも払っていない |
| 182 | 当たり。note を足した | 私が編集した org skill の `gate` 行で当たった(own-edit hit)。verify も handoff も挙げていない |
| 117 | 当たっていない | Trigger は template-content の restructure。この branch は項目を足しただけで、写しの構成は変えていない |
| 236 | 当たっていない | Trigger は reviewer の RESULT の `HEAD:`/`BASE:` を頼りにする読み手。gate の環境とは別の話 |
| 161 | 当たっていない | Trigger は org の quality pipeline の変更(手順が変わる変更)。この branch は gate の scripts の環境を変えただけで、pipeline の手順は変えていない。同じ形の変更(`gateScriptEnv` の `RALPH_LOCAL_CONFIG`)にも note は付いていない |
| 407、471、473、483、499 ほか gate と regate の行 | 当たっていない | Trigger は `regate.go`、`regate_ci.go`、`gate.lock`、gate の record の key、`waitGateMaxLoad`、`gateToolsProbe` など、この branch が触っていない物 |
| 455、150 | 当たっていない | Trigger は `[org]` の検証、`resolveOrgConfig` など。私の `config.go` の編集は `VerifyConfig.Base` の comment だけ |
| 143、144、141(`secret-scan-branch.sh`) | 当たっていない | この branch は `secret-scan-branch.sh` を触っていない |
| 373、387、153、264、380、510、511 | 当たっていない | recipe、hook の文、`DefaultRoleModels`、`/director` の受信箱、leader の例など、触っていない場所が Trigger |

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md` | 変えない。地図で、一覧に載る契約は変わっていない。`internal/org/` の行は gate の説明を持つが、環境の変数の名前は書いていない(`GITHUB_BASE_REF`、`RALPH_VERIFY_`、`gateScriptEnv` は 0 件) |
| `docs/quality/quality-gates.md`(2 写し) | 変えない。この branch の項目(base の順と docs only の判定)は、コードと合う(verify が表で確かめた)。gate の環境を書く文はない(`RALPH_VERIFY_` は `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` の 1 つだけ) |
| `docs/architecture/repo-map.md` | 変えない。script の一覧で、新しい script はない |
| `docs/quality/definition-of-done.md`(2 写し) | 変えない。「For docs-only changes」は、ralph の検証の判定の話ではなく、docs だけの変更の完了条件 |
| `.claude/rules/ralph/model-routing.md`(2 写し) | 変えない。`ralph-config.sh` が `[verify].base` を読むという記述(「It reads no environment variable.」を含む)は、`ralph-config.sh` 自身の話で、今も合う。`GITHUB_BASE_REF` を読むのは detector と `secret-scan-branch.sh` |
| `docs/recipes/adding-a-language-pack.md`(2 写し) | 変えない。言語の検出の記述は合う |
| `internal/cli/config.go` の `ralph config get` の help | 変えない。「the safe side (the full verify scope)」は設定の誤りの話で、今も合う |
| 古い文面(`docs or scaffold-level`、`No verifier ran for code-like`、`No environment variable chooses the base`、`run_non_docs`) | docs/、`.claude/`、`.agents/`、template、README、AGENTS.md に残りなし。`docs_only` は docs/ の本文に 0 件(report と plan のほかに出ない) |
| `internal/upgrade/notes/` | 変えない。release ノートは `/release` で書く決まりで、計画の R1〜R3 に候補の文だけがある |

## 直さなかったこと

- 計画の本文(Progress の「PR created」は `[ ]` のまま。`/pr` が付ける)。AC の箱は `805b0445` で付いている。計画の Progress の 353〜355 行目の「Follow-up for /sync-docs」は、tech-debt の 417 行目に書いたので、そのままにした
- L5、L6、G1、G2、G4 は直していない。コードとテストの変更になるので、この step の範囲外。235 行目に Trigger つきで書いた
- verify の INFO I1(live の記録の行末の空白 10 行、scaffold の手元の clone の 2 行が `no_changes` で通っている可能性)と I2(quality-gates の括弧の一覧が例の列であること)は、記録と例の話で、直していない
- `docs/tech-debt/README.md` の 118 と 417 は、branch ごとに note を足してきた行なので、並行する PR と同じ行の末尾で衝突しうる。この branch の note は Trigger のセルの末尾と、Related の列の末尾に足した。衝突したら、両方の note を日付順に残す。新しい行は、末尾ではなく 234 の次に置いた

## 確認

- `go test ./internal/config/ -count=1`: ok(32.9 s)。`gofmt -l internal/config/` は空。key の comment 行(`# base = ""`、`# skip_branch_secret_scan = false`)は変えていないので、`commentedTemplateDrift` は通る
- `go test ./internal/scaffold/ -count=1`: ok(4.4 s)
- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`。`cmp` で、`.claude` と template の `.claude`、`.agents` と template の `.agents` の org skill が一致
- `./scripts/check-template.sh`: `Template structure looks good.`。`./scripts/check-template-purity.sh`: `PASS`
- 計画の digest: `f500fb32da2a`(変わっていない)
- tech-debt の編集: HEAD(編集前)と突き合わせ(`check_register.py`、scratch)、変わった行は 118、182、234、239、240、369、400、417 の 8 行と、新しい 235 行目だけ。どの行も 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし。計画の参照は `docs/plans/active/2026-11-19-verify-docs-only-base.md` が 8 か所(編集後の 118、182、235、240、241、370、401、418 行目の Related の列)で、どれも実在する(`verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索を手で回した。`verify.local.sh` そのものは回していない)。401 行目は、最初の commit で足し忘れ、この report の commit で足した
- テスト(`run-test.sh`、shell の suite、`internal/org` と `internal/cli` の Go のテスト)は回していない。この step が変えたコードは `VerifyConfig.Base` の comment だけで、挙動は変わらない

## Insight event

`./scripts/insights-append.sh --slug verify-docs-only-base --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-verify-docs-only-base.jsonl` に足した。この branch は前の phase の行も同じファイルに commit している。
