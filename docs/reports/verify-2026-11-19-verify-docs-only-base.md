# Verify report: verify-docs-only-base

- Date: 2026-11-19(cycle 1。worktree に `.harness/state/standard-pipeline/` はなく、`--cycle auto` は 1 になる)
- Plan: `docs/plans/active/2026-11-19-verify-docs-only-base.md`(issue #341)。承認の digest `37bcd60d52cb` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff a8c5e289...HEAD`(base は `origin/main` の `a8c5e289`、HEAD `d55e9cb1`、18 ファイル、+2229 −283)。コードは S1 `930e6975`、S2 `ec269cb7`、S2b `672e5ded`、S3 `5fe08eff`、S4 `1c3dc457`(self-review の L2・L3・L4・L7 の直し)。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-19-verify-docs-only-base.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)

## Spec compliance

状態の「コードとテストのソース」は、コードを読み、テストがその挙動を固定していることをソースで確かめたという意味で、テストの実行結果ではない。「probe」は HEAD の scripts を scratch の repository で走らせた結果(下の Observational checks)。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: pack のない repository の CI 風の checkout(develop が既定、`origin/HEAD` なし、PR の merge commit に detached、`GITHUB_BASE_REF=develop`)で、コードを足す commit は full scope で exit 2、出力はコードの path を名指し、docs only と言わない | Met(コード、テストのソース、probe) | detector は `GITHUB_BASE_REF` を最初に読み(`scripts/detect-changed-languages.sh:242-262`)、`first_non_docs` を立てる(`:551-554`)。run-verify.sh は full scope でも detector を呼び(`scripts/run-verify.sh:102-118`)、`false` で exit 2 と path の行を出す(`:266-274`)。テスト: `tests/test-run-verify-docs-only.sh:333-342`(exit 2、`for example: app.js` の行、`docs or ralph scaffold` を含まない、scope file の `docs_only=false`)。fixture の形は `:309-322` が確かめる。probe: HEAD で `code ci GITHUB_BASE_REF full rc=2` |
| AC2: 同じ checkout で docs だけの commit は exit 0 | Met(コード、テストのソース、probe) | `scripts/run-verify.sh:263-265`。テスト `:344-354`(exit 0、docs の行、`Add a verifier` を出さない、scope file の `docs_only=true`)。probe: `docs ci GITHUB_BASE_REF full rc=0` |
| AC3: ralph の scaffold だけ(manifest に載った file、manifest 自身、manifest に載った `.gitignore`)を足す commit は exit 0 | Met(コード、テストのソース、probe) | scaffold の判定は `:504-541`。テスト `:356-364`(fixture の manifest は `.gitignore` を載せる、`:301`)。probe: HEAD から build した ralph の `ralph init --yes` が書いた 161 ファイル(manifest は 160 件、載っていないのは `.ralph/manifest.toml` 自身だけ)の commit で `scaffold ci GITHUB_BASE_REF full rc=0` |
| AC4: 手元の clone(feature branch、clean、`GITHUB_BASE_REF` なし)は AC1〜AC3 と同じ exit code を full と changed の両方で返す | Met(テストのソース、probe) | テスト `:383-403`(3 種 × 2 scope の exit code を CI 風の結果と比べ、changed scope の code の行も確かめる)。probe: code は full と changed で rc=2、docs と scaffold は rc=0。scaffold の手元の clone は `[verify].base` を scaffold の前の commit にして、detector が scaffold の集合を読んで `docs_only=true` を出すことも確かめた |
| AC5: `GITHUB_BASE_REF` の ref がない、または merge-base がないとき exit 2 で、読めなかったことと理由を言い、docs only と言わない | Met(コード、テストのソース) | `:254-262`、`:321-324`、run-verify.sh `:275-280`。テスト `:405-431`(`no_base_ref:nosuch`、`no_merge_base:refs/remotes/origin/lonely`、`GITHUB_BASE_REF` なしの `no_diff_base`、どれも docs の行なし、scope file の `docs_only=unknown`)。probe では `no_diff_base` の場合だけ打った(rc=2) |
| AC6: exit 2 の出力が mode ごとに verifier の足し方を言う | Met(コード、テストのソース) | `scripts/run-verify.sh:251-261`。テスト `:433-455` が `run-static-verify.sh`、`HARNESS_VERIFY_MODE=static`、`run-test.sh`、`all` の 4 つで文を固定し、static は `test.d` を、test は `verify.d` を含まないことも確かめる。`ralph pack add` は実在する(`internal/cli/pack.go:26`) |
| AC7: detector が `docs_only=true\|false\|unknown` と、`false` のとき `first_non_docs` を出し、既存の case の scope、reason、languages、roots は変わらない | Met(コード、テストのソース) | `emit_result`(`:175-193`)。既存の case 1〜42 の差分は assert の追加と `unset` だけで、消えた行は 1 つ(config_error の `docs_only false` を `unknown` に置き換えた行、`tests/test-detect-changed-languages.sh:849`)。scope、reason、languages、roots の assert は 1 つも変わっていない。base の版と読み比べると、shared や unclassified の path のあとは言語と root を足さない(`:556-559`)ので、早く抜けていた版と同じ値になる |
| AC8: `GITHUB_BASE_REF` があると `refs/remotes/origin/<name>`、なければ `refs/heads/<name>` を使い、ralph に `[verify].base` を聞かない | Met(コード、テストのソース) | `:242-262`(この枝は `ralph-config.sh` を読まない)。テスト case 50(`:1082-1153`、origin の ref、手元の branch だけ、どちらもない、変な文字、merge-base なし、空の値)と case 51(`:1155-1170`、失敗する stand-in の ralph が呼ばれない、stderr が空) |
| AC9: 2 本の script の `scripts/` と `templates/base/scripts/` の写しが同じで、`check-sync.sh` が通る | Met | `cmp` で 2 本とも一致。`check-sync.sh` は DRIFTED 0、`PASS: all files in sync.` |
| AC10: detector か run-verify.sh を走らせるテストはすべて `GITHUB_BASE_REF` を unset する | Met(テストのソース) | `tests/test-detect-changed-languages.sh:129`、`tests/test-ralph-dispatch.sh:47`、`tests/test-run-verify-scope.sh:9`、`tests/test-run-verify-concurrency.sh:34`、`tests/test-run-verify-docs-only.sh:31`、`tests/test-run-verify-branch-secret-scan.sh:127`、Go は `internal/org/gate_runverify_test.go:85`(`gateRalphStubEnv`。gate の run も `GateParams.Env` にこの env を渡す、`gate_test.go:223`)。scripts の名前を出すほかの 5 本の shell test はファイルの中身を読むだけで、実の scripts を走らせる Go のテストは `gate_runverify_test.go` だけ(`feature_test.go` が写すのは `ralph-worktree.sh` と `ralph-common.sh`) |
| AC11: 4 つの listing コマンドのどれか 1 つが失敗すると `diff_failed` と `unknown` になり、pack のない run-verify.sh は full と changed で exit 2 | Met(コード、テストのソース) | `:346-351`。detector のテスト case 52(`:1172-1250`)が 4 つを 1 つずつ失敗させ、前のコマンドが docs の path だけを出す fixture で、そのコマンドだけが 1 回失敗したことを log で確かめる。run-verify.sh 側は `--cached` の失敗を両 scope で打つ(`tests/test-run-verify-docs-only.sh:457-488`)。run-verify.sh は `unknown` を reason によらず同じ枝で扱う(`scripts/run-verify.sh:275-280`)ので、残る 3 つも同じ結果になる |
| AC12: scaffold の非 `.md` の file を消し、manifest からも消す commit が、CI 風の checkout と手元の clone で exit 0 | Met(コード、テストのソース) | `:527-541`(存在しない path は merge-base の manifest も見る)。テスト `tests/test-run-verify-docs-only.sh:366-381`(CI 風と、手元の clone の full と changed)、detector の case 53(`:1252-1296`、どちらの manifest にもない削除は `false`、今もある file は今の manifest だけを見る) |

### ユーザーの 2 つの決定

- pack のない repository のコードの PR は、CI でも手元でも、commit 済みかどうかによらず exit 2: Met。commit 済みは AC1 と AC4(テストと probe)。commit していない変更は、unstaged が `tests/test-run-verify-concurrency.sh` の Case 5(`app.txt`、full scope、exit 2)。untracked と staged の `tool.sh` は probe で full と changed の 4 通りとも rc=2 だった
- `ralph init` の scaffold だけの PR は失敗しない: Met。AC3 のテストと、HEAD の ralph で作った本物の scaffold の probe(CI 風と手元の clone の full と changed で rc=0)

### detector の header の約束とコード

| header の記述 | コード | 結果 |
| --- | --- | --- |
| 出力の key(9-12 行目): `first_non_docs` は `docs_only=false` のときだけ | `first_non_docs` を立てるのは `all_docs=false` にする所だけ(`:551-554`)。`fallback_unread` はループの前に出る | 合う |
| `unknown` の理由の一覧(22-24 行目)の 8 つ | `fallback_unread` の呼び出しは `git_not_found`、`no_git_repository`、`no_base_ref`、`config_error:verify.base`、`no_remote_default`、`no_diff_base`、`no_merge_base`、`diff_failed` の 8 か所 | 合う |
| scaffold の定義と、削除した path は merge-base の manifest も見ること(26-35 行目) | `:504-541` | 合う |
| 4 つの listing を 1 つずつ確かめること、`no_changes` は 4 つが走って何も出さなかったときだけ(37-41 行目) | `:346-357` | 合う |
| `core.quotePath=false` と、引用符つきで残る path は non-docs(43-48 行目) | `:346-349`。`"` を含む docs の path を固定するのは case 55 | 合う |
| full scope の reason は最初の shared か unclassified の path、そこから言語と root は増えない(50-54 行目) | `:556-581` | 合う。`docs_only` が `false` に決まった時点でループを抜ける(`:557`)が、それ以降で値は変わらない |
| `GITHUB_BASE_REF` の読み方とラベル(56-63 行目) | `:242-262`、`:323` | 合う |
| 「No setting comes from an environment variable」と、`GITHUB_BASE_REF` を先に読み ralph に聞かないこと(78-80 行目) | `GITHUB_BASE_REF` の枝は `ralph-config.sh` を読まない | 合う |

## Documentation drift

| 場所 | 状態 | 内容 |
| --- | --- | --- |
| `docs/quality/quality-gates.md` と template の写し | 同期済み | この diff が足した行は 2 つの写しで同じ。2 つの差は diff の外の 3 か所(72、75-76、101 行目付近)だけで、前からある `KNOWN_DIFF` |
| README の `[verify].base` の文(154 行目) | 同期済み | docs-only の判定にも使うこと、CI では `GITHUB_BASE_REF` が先なことを書いた。設定の誤りで「changed ではなく full を取る」という同じ段落の文は今も正しい。pack のない run が exit 2 になることは quality-gates の新しい項目が書く |
| 2 本の script の comment | 同期済み | detector の header は上の表。run-verify.sh の comment(`:32-42`、`:61-71`、`:90-96`、`:246-249`、`:291-295`)は消えた `run_non_docs_file` に触れず、3 値の `docs_only` を書く |
| `internal/config/config.go:136-141`(`VerifyConfig.Base` の doc)と `templates/base/ralph.toml:99-104` | ずれ(V1) | どちらも `[verify].base` を「changed scope が調べる言語を選ぶための revision」とだけ書き、両 scope の docs-only の判定に使うことも、CI では `GITHUB_BASE_REF` が先なことも書かない。README は直った。defaults_sync のテスト(`commentedTemplateDrift`)が読むのは `# base = ""` のような key の comment 行だけで、説明の文は固定していない |
| `internal/org/gate_run.go:62-66` と `internal/org/gate_runverify_test.go:48-49` | ずれ(V2) | 「No variable sets the diff base」と書くが、この diff から detector は `GITHUB_BASE_REF` を diff base に取り、gate はその変数を落とさない(`gateScriptEnv` が落とすのは `RALPH_VERIFY_` と `HARNESS_VERIFY_` で始まる名前だけ)。同じテストファイルの `gateRalphStubEnv` の comment(75-78 行目、この diff が書いた)は逆のことを言う |
| `scripts/xreview-helpers.sh:47` の「No environment variable chooses the base.」 | ずれなし | `detect_base_branch` 自身の話で、この関数は `GITHUB_BASE_REF` を読まない(`secret-scan-branch.sh` が呼ぶ前に読む) |
| `docs/recipes/adding-a-language-pack.md`(2 写し)、`.claude/rules/ralph/model-routing.md`、`internal/cli/config.go` の help | ずれなし | 言語の検出、`ralph-config.sh` の読み方、設定の誤りで full を取ることの記述で、今も正しい |
| 古い文面(`scaffold-level`、`Add a real verifier`、`docs or scaffold`) | 残りなし | 過去の report のほかに出てこない |

### tech-debt の register(`docs/tech-debt/README.md`)

各行の 4 つめの cell(Trigger)を python で切り出し、この diff が触った file と照らした。

| 行 | 引き金 | 状態 |
| --- | --- | --- |
| 118(template の quality-gates の「Must pass in CI」と配線の不一致) | 「`templates/base/docs/quality/quality-gates.md` または template 側 verify チェーン(verify.yml / run-verify.sh)への次のタッチ」 | 当たる(V3)。この diff は両方を触った。前の 6 回のタッチはそれぞれ持ち越しの注記を足しているが、この diff は足していない。self-review の一覧にもない |
| 237(gate が落とさない変数) | 「a script that starts reading a new variable that narrows what it checks」 | 当たる。self-review L1 が `/sync-docs` に回した。V2 の comment と同じ話 |
| 415(800 行を超えた 5 つのファイル) | 「The next change to any of the five」 | 当たる。`tests/test-detect-changed-languages.sh` は 848 から 1371 行(`wc -l`、base と HEAD)。self-review L8 と plan の Progress が `/sync-docs` に回した |
| 232(evidence の log の名前) | 引き金には当たらないが、項目の文の「the docs-only marker go through per-run `mktemp` files」が古い | self-review L9 が `/sync-docs` に回した |
| 117(quality-gates の `KNOWN_DIFFS`) | 「The next template-content restructure」 | 当たらない。この diff は項目を足しただけで、写しの構成は変えていない |
| 236(gate が worktree の scripts を走らせる) | 「A feature PR that changes either script」 | 当たらない。この PR は機能の org の PR ではなく、行が名指す `run-static-verify.sh` と `run-test.sh` も変えていない |

## Findings

| ID | Severity | 内容 | 勧め |
| --- | --- | --- | --- |
| V1 | LOW | `VerifyConfig.Base` の doc と `templates/base/ralph.toml` の `[verify]` の comment が、`[verify].base` の新しい役割(両 scope の docs-only の判定)と `GITHUB_BASE_REF` が先なことを書かない(上の表) | `/sync-docs` で README と同じ内容を足す。ralph.toml では `# base =` で始まる説明の行を作らない(`commentedTemplateDrift` が key の comment と数える) |
| V2 | LOW | `gate_run.go:62-66` と `gate_runverify_test.go:48-49` の「No variable sets the diff base」が、この diff のあとでは正しくない | 2 つの comment を「`GITHUB_BASE_REF` のほかには」に直す。register の行 237 の注記と一緒に |
| V3 | LOW | register の行 118 の引き金に当たったのに、持ち越しの注記がない | `/sync-docs` で、前の注記と同じ形の 1 文(触ったのは「Must pass locally」の節と template の run-verify.sh で、「Must pass in CI」と配線は範囲外)を足す |

INFO(数えない):

- I1. `docs/evidence/verify-docs-only-live-2026-11-19.md` は 09e63c43 の記録で、S4(`1c3dc457`)より前。S4 の変更(非 ASCII の path、`no_merge_base` のラベル、trap、最後の `emit_result` の引数)は、記録の手順が通る経路(ASCII の path、merge-base あり、signal なし)を変えない。HEAD `d55e9cb1` で同じ手順を打ち直し、10 行とも同じ rc になった。別の点として、記録の scaffold の手元の clone の 2 行は、develop を scaffold の commit に進めたあとの clone なので、おそらく `no_changes` で通っている(記録に手順の script がなく未確認)。打ち直しでは `[verify].base` を scaffold の前に向け、scaffold の集合そのものが `docs_only=true` になることを見た。また結果の code block の 10 行に行末の空白がある(`git diff --check` が rc=2。CI はこの検査をしない)
- I2. quality-gates の新しい項目の括弧の一覧は例の列で、detector の表の `templates/base/docs/` と、読めない理由の no git、ref のない `GITHUB_BASE_REF`、remote の既定がない、detector の不在や失敗を挙げない。plan の R4 は「header と quality-gates に書いた」と言うが、手元の shell に `GITHUB_BASE_REF` が残ると base が変わることを書くのは header だけで、quality-gates は「when CI sets it」と書く
- I3. plan の Acceptance criteria の箱は全部 `[ ]` のままで、Progress の「Review artifact created」も self-review の report があるのに `[ ]`。`/pr` までに埋める記録の遅れ
- I4. self-review が known gap として残した L5(verifier が走る run でも full scope で detector を呼ぶ。設定の誤りでは detector の「taking the full scope, not the automatic base」の行が出る)と L6(full scope でも reason の名前が `changed_detector_*`、base の案内の行が `unknown` のすべての reason で出る)はそのまま。この verify は直していない

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc=0 | 変更に `scripts/detect-changed-languages.sh` があるので full に落ち(`shared:scripts/detect-changed-languages.sh`)、golang の pack を走らせた。gofmt `ok`、golangci-lint `0 issues.`、shellcheck と `sh -n` は OK |
| `scripts/check-sync.sh`(上の中) | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh`、`scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh`、register の plan の参照(上の中) | OK | 14 skill が lock-step |
| `secret-scan-branch.sh`(上の中) | clean | `a8c5e289..d55e9cb1` against `origin/main` |
| `./scripts/check-template.sh` | rc=0 | `Template structure looks good.` |
| `shellcheck --severity=warning`(変わった 2 本の script と 6 本の test) | rc=0 | 既定の severity では info の SC1091 が 3 件出るが、base の版と同じ件数(`# shellcheck source=` の行) |
| `sh -n` と `dash -n`(2 本の script) | rc=0 | |
| `cmp`(2 本の script の root と template) | 一致 | |
| `./scripts/plan-visual.sh digest <plan>` | `37bcd60d52cb` | 計画の `- Approved:` の行と一致 |
| `git diff --check a8c5e289...HEAD` | rc=2 | live の記録の行末の空白 10 行だけ(I1) |
| diff の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | 手元の `origin/main` は `a8c5e289` で、fetch はしていない |

## Observational checks

scratch はすべて `$TMPDIR/verify-341/` の下で、HOME、XDG の 4 変数、TMPDIR、git の global 設定(hooks は空のディレクトリ)をそこに向けた。test suite は走らせていない。

- live の打ち直し: `go build -o $TMPDIR/verify-341/ralph ./cmd/ralph`(HEAD `d55e9cb1`)、その ralph で pack のない repository(`index.html`、`README.md`、`data.json`)に `ralph init --yes .`。init が書いた `scripts/detect-changed-languages.sh` と `scripts/run-verify.sh` は worktree の HEAD の写しと `cmp` で一致し、`scripts/verify.local.sh` も `packs/languages/` もない。develop を既定にした bare の remote、`origin/HEAD` を消した CI 風の checkout、手元の clone で run-verify.sh を走らせた結果は、scaffold の CI 風 rc=0、code の CI 風 rc=2(`for example: app.js`)、docs の CI 風 rc=0、`GITHUB_BASE_REF` のない docs の CI 風 rc=2(`no_diff_base`)、code の手元 full と changed は rc=2、docs と scaffold の手元 full と changed は rc=0
- commit していないコード: docs の手元の clone に untracked の `tool.sh` を置くと full と changed で rc=2、stage しても rc=2(どれも `for example: tool.sh`)。probe のあと元に戻した
- rename: manifest に載った `scripts/old-tool.sh` を `scripts/new-tool.sh` に `git mv` した commit で、detector は `docs_only=false`、`first_non_docs=scripts/new-tool.sh`(git の rename 検出で新しい path だけが出て、manifest にないので code になる。安全側に倒れる)
- `run-static-verify.sh` は worktree に `docs/evidence/verify-2026-11-19-130012.log` を書いた(`.gitignore` で無視される)

## Coverage gaps

- テストの実行は `/test` の担当で、この verify はソースを読んだだけ。plan の Progress にある件数(S1 の 221 など)は S4 より前の値で、S4 が case 55 を足したあとの件数は見ていない
- 本物の GitHub Actions の `actions/checkout` は見ていない。`refs/remotes/origin/HEAD` を書かないという仮定は、`GITHUB_BASE_REF` を先に読むので結果に効かない。`fetch-depth: 0` は root と template の `verify.yml` の両方にある
- AC5 の `no_base_ref` と `no_merge_base` は probe で打っておらず、テストのソースだけで見た
- 足りない決定的な検査: V2 の comment のような「変数が base を決めない」という文を固定するテストはない。足すなら、`GITHUB_BASE_REF` を caller の env に入れた gate の run が、入れない run と同じ `ralph config get` の呼び出しと結果になる(または違うことを明示する)Go のテストが一番小さい

## Verdict

- Verified: AC1〜AC12(コードとテストのソース。AC1〜AC4 は HEAD の probe でも)、ユーザーの 2 つの決定、detector の header の約束、root と template の写しの一致、静的検査(`run-static-verify.sh` rc=0、`check-template.sh` rc=0、shellcheck)
- Partially verified: AC5 の 2 つの理由(テストのソースだけ)、AC11 の run-verify.sh 側(4 つのうち 1 つを打つテストと、`unknown` を 1 つの枝で扱うコード)
- Not verified: テストの実行結果(`/test`)、本物の CI の checkout
- Findings: LOW 3 件(V1〜V3、どれも文書と記録のずれで、`/sync-docs` で直せる)、INFO 4 件
- Result: pass

## Addendum after V2 (orchestrator, S2c)

V2 was fixed in code, not only in comments, in commit 17a059be (plan item
2c; consult `consult-plan-docsonly` round 2): `gateScriptEnv`
(`internal/org/gate_run.go`) drops a variable named exactly `GITHUB_BASE_REF`,
because the detector and `scripts/secret-scan-branch.sh` take it as their base
(B9). The comments in `gate.go`, `gate_run.go`, and `gate_runverify_test.go`,
the help of `ralph org gate` (`internal/cli/org_gate.go`), and its test were
updated; `TestGateScriptEnv` pins the drop (`GITHUB_BASE_REFX` is kept), and
`TestGate_RealRunVerify_FullScopeRunsTheLanguageVerifier` gives the gate's
caller `GITHUB_BASE_REF=feat/auth` and still sees the `verify.base` read.

Re-run after 17a059be:

- `./scripts/run-static-verify.sh`: rc=0 (full fallback,
  `shared:scripts/detect-changed-languages.sh`; golang verifier with
  golangci-lint `0 issues.`; check-sync PASS; check-pipeline-sync,
  check-skill-sync, check-template-purity OK; branch secret scan clean)
- the implementer's slice command (gofmt, `go vet ./internal/org/
  ./internal/cli/`, `go test -run 'TestGateScriptEnv|TestGate_RealRunVerify'`,
  `go test ./internal/cli/ -run OrgGate`): ok, also with `GITHUB_BASE_REF=main`

README's gate paragraph and the org skill's `gate` row (four copies) still
describe the gate's environment without `GITHUB_BASE_REF`; they go to
`/sync-docs` with V1, V3, and the tech-debt rows. The verdict above is
unchanged.
