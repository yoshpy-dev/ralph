# Verify report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Verifier: verifier subagent (Claude Opus 5.5)、cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC7)、静的解析、文書のずれ。対象は `git diff origin/main...HEAD`(base 2a22ba78、HEAD c45730cf、27 ファイル、+1511/-129)。self-review の修正コミット 989886f2(L-2、L-3、L-4)と plan の訂正 c45730cf(M-1)を含む。テスト(`./scripts/run-test.sh`、`go test`、`tests/test-*.sh`)は /test の担当なので実行していない。guard、`check_tech_debt_plan_refs` は scratchpad の入力で直接動かした(観察のための probe で、テストスイートではない)
- Evidence: `docs/evidence/verify-2026-10-07-guard-bypass-and-hygiene.log`(`docs/evidence/*.log` は gitignore 対象なので手元にだけ残る)。runner 自身のログは `docs/evidence/verify-2026-10-06-184026.log`

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `c07adf402bdb` を返し、plan の `- Approved: 2026-10-07 sha256:c07adf402bdb` と一致した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: bypass で ask なし、deny 4 規則は全モード、モードなし・`default`・`auto` は ask、ask と deny の両方なら deny、jq なしでもエスケープされた `git commit -m "$(id)"` が deny、テストが 3 値で jq あり・なし | Met(テストの実行は /test) | モードで分かれるのは `.claude/hooks/pre_bash_guard.sh:95` の 1 か所だけで、その前に deny の 2 つの `case`(`:65-75` と `:79-90`)がある。モードは `:22` で `extract_json_field` から読む。`lib_json.sh:27-28` の sed fallback が `"(([^"\\]|\\.)*)"` で値を読み、`\"` と `\\` を戻す。probe(新 guard、root と template、jq あり・なし)で 142 例・284 判定がすべて期待どおり。deny 5 形(4 規則と ask+deny の組み合わせ)は、モードなし・`default`・`plan`・`acceptEdits`・`auto`・`dontAsk`・`bypassPermissions` の 7 通りすべてで deny。ask 3 形は bypass 以外の 6 通りで ask、bypass で none。旧 guard(origin/main)を同じ payload で動かすと、jq なしの `git commit -m "$(id)"` は none、`rm -rf x && git commit -m "$(id)"` は ask で、plan の Assumptions(`:58`)どおりの穴と順序の問題を再現した。テストは `tests/test-pre-bash-guard.sh` の E(`:223`、deny をモードなしと bypass で)、F(`:237`、ask+deny)、G(`:248`、`default` と `auto`)、`run_case` の none / ask / deny の厳密比較、`check` が jq あり・なしの両方を回す。テストは `plan`・`acceptEdits`・`dontAsk` を試していないが、コードの分岐が `bypassPermissions` の文字列一致だけなので、上の probe で足りると判断した。root と template は `cmp` で一致 |
| AC2: `.git` と `.env` の ask は書き込み先だけ。誤検知 6 形は none、書き込み 7 形は bypass 以外で ask。テストが jq あり・なし | Met(テストの実行は /test) | 判定は `pre_bash_guard.sh:48-61` の `redirect_lead`・`tee_lead`・`git_target`・`env_target` と `command_writes_to`、呼び出しは `:99-104`。AC2 の 13 形はすべて `tests/test-pre-bash-guard.sh` の C(`:176`)と D(`:196`)にあり、`cat > .env <<EOF` は複数行の形で入っている。probe でも 13 形とも jq あり・なしで期待どおり。旧 guard では `ls .git/ 2>&1` と `cat .env.example 2>/dev/null` が ask だった(誤検知の再現)。self-review の M-1 の 2 例(`cp hook .git/hooks/pre-commit 2>&1`、`mv x .git/x > /dev/null`)は新 guard で none で、訂正後の plan の Non-goals(`:50`)と Risks(`:136`)の記述と合う |
| AC3: shellcheck の対象が `scripts/*.sh` のグロブで `insights-append.sh` を含む、`run-test.sh` に SC2209 なし、`run-verify.sh` の shellcheck の段が通る | Met | `scripts/verify.local.sh:175` が `scripts/*.sh` のグロブ。展開すると 95 ファイルで、そのうち `scripts/` が 36 本、`scripts/insights-append.sh` を含む。同じ 95 ファイルに `shellcheck --severity=warning` を掛けて rc 0。`scripts/run-test.sh:5` は `HARNESS_VERIFY_MODE='test'` で、root と template とも shellcheck の既定の重さで rc 0。origin/main の `run-test.sh` は SC2209 を出す(rc 1)ので、修正の前後を確かめた。`run-static-verify.sh` の shellcheck の段は OK |
| AC4: 実行権限なしと index 100644 を FAIL、ファイル名と直し方、非 0 で終了、テスト 3 通り | Met(テストの実行は /test。直し方の表示は plan の文言より細かい。V-8) | `scripts/verify.local.sh:245-258` の `hook_test_mode_problem` が working tree の `-x` と `git ls-files -s` の mode を見て、`run_hook_tests`(`:260-288`)が FAIL の行、`chmod +x <file>`、tracked なら `git update-index --chmod=+x`(`:278`)、untracked なら `git add --chmod=+x`(`:280`)を出し、`status=1`(`:283`)で終了コードを非 0 にする。`tests/test-verify-local-hook-tests.sh` は Case 1(`:153`、実行権限なし)、Case 2(`:158`、index 100644)、Case 3(`:176`、両方そろう)に、Case 4・5(untracked)を足した 5 通り。いまの `tests/test-*.sh` は index でもすべて 100755(新しい 2 本と変更した 2 本を `git ls-files -s` で確認) |
| AC5: `/sync-docs` の 4 面に insight event の節、`--phase sync_docs`・`--verdict pass`・`--cycle auto`・`\|\| true`、テストが 5 skill × 4 面 | Met(テストの実行は /test) | 4 面とも `## Insight event (best-effort)`(`.claude/skills/sync-docs/SKILL.md:33`)と 4 つの文字列を grep で確認した。`.claude/` と `.agents/` の差は frontmatter の `allowed-tools` の 1 行だけで、base から変わっていない。`tests/test-skill-insight-cycle.sh:39` の対象に `sync-docs` が入り、`insight_block` は節の最初のコードブロックを読むので、この節の書き方で拾える。`insights-append.sh:199-200` は `sync_docs` と `pass` を受け付ける |
| AC6: README に archive 済みの plan を active で指す参照がない、`archive-plan.sh` の順序と書き換え、`/pr` step 8 の 4 面、テスト 4 点、`verify.local.sh` の参照検査 | Met(テストの実行は /test) | README の `docs/plans/active/` の参照は 0 件(origin/main では 34 か所、16 本)。origin/main の README の `docs/plans/active/` を全部 `docs/plans/archive/` に置き換えた結果が HEAD の README と `cmp` で一致したので、変更はこの置き換えだけ。`scripts/archive-plan.sh` は移動先の衝突(`:49`)→ README の書き換え(`:65-120`、件数は `:115`)→ 移動(`:122`)の順。名前の境界は awk の `is_name_char` で、`.` のあとに名前の文字が続く参照は書き換えない。README がなければ書き換えの段を飛ばす。`/pr` step 8(`.claude/skills/pr/SKILL.md:54`)の `Updated N reference(s) in docs/tech-debt/README.md` は 4 面とも 1 回ずつあり、スクリプトの出力の文言と合う。`tests/test-archive-plan.sh` は Case 1(書き換え)、Case 2(似た名前)、Case 3(README なし)、Case 4(移動の失敗とやり直し、root では SKIP)を持つ。`check_tech_debt_plan_refs`(`verify.local.sh:140-153`、呼び出しは `:235`)は実際の README で OK。関数を取り出して scratchpad で動かすと、正しい参照だけなら rc 0、archive 済みを active で指す参照と存在しない archive の参照を足すと 2 件を挙げて rc 1、README なしで rc 0 だった。root と template の `archive-plan.sh` は `cmp` で一致 |
| AC7: `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`、`run-verify.sh` | Met(`run-verify.sh` は静的解析の部分だけ) | 4 本は `run-static-verify.sh` の中と単独の実行の両方で OK(13 skill が一致、DRIFTED 0、6 文書がパイプラインの全工程を参照、template に meta-repo の参照なし)。`./scripts/run-static-verify.sh`(`HARNESS_VERIFY_MODE=static` の `run-verify.sh`)は rc 0。既定の `all` モードのテストの部分は /test に回した |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | `verify.local.sh`(shellcheck 95 ファイル、hook の `sh -n` 20 本、settings の `jq -e` 2 本、Codex の hook guard 3 本、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README の plan の参照)、golang verifier(`lib_json.sh` が言語に分類されないため full にフォールバック。gofmt ok、lint 0 issues)、branch secret scan(`2a22ba78..c45730cf` clean) |
| `shellcheck --severity=warning`(グロブに広げた 95 ファイル) | PASS(rc 0) | shellcheck 0.11.0。既定の重さで変更した 8 本を見ると info だけが出る(SC2329、SC1091、SC2016、SC2012、SC1003)。gate は warning 以上なので対象外 |
| `shellcheck`(`scripts/run-test.sh` の root と template) | PASS | origin/main の版は SC2209(warning)で rc 1 |
| `./scripts/check-skill-sync.sh` | PASS | 13 skill が一致 |
| `./scripts/check-sync.sh` | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | PASS | 6 文書すべて |
| `bash scripts/check-template-purity.sh` | PASS | |
| `sh -n`(guard、`lib_json.sh`、`archive-plan.sh`、`verify.local.sh`、テスト 3 本) | PASS | |
| `git diff --check origin/main...HEAD` | PASS | 空 |
| 追加行の U+FFFD 検索 | PASS | 0 件 |
| ミラーの `cmp` | PASS | `pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`archive-plan.sh`、`run-test.sh`、`/pr` と `/sync-docs` の SKILL.md(`.claude/` の root と template、`.agents/` の root と template) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.codex/README.md:114-117`(root と template) | 一致 | 「`deny` が実際に止める」という記述で、deny はどのモードでも残るので矛盾しない |
| `.claude/rules/ralph/git-commit-strategy.md:70`(root と template) | 一致 | 「`pre_bash_guard.sh` blocks dangerous patterns」は deny の規則のことで、変わっていない |
| `internal/org/prompts/implementer.md:27` | 一致 | `-m "$(` の deny は残っている |
| `docs/quality/quality-gates.md` | 一致 | `verify.local.sh` の個々の検査を列挙していない。実行権限の検査は test モード、参照の検査は static モードにあり、`:31` の「static と test は重ならない」も保たれる |
| `docs/insights/README.md:117-119`(root と template) | ずれ(V-1) | insight event を書く skill を「`/self-review`, `/verify`, `/test`, `/cross-review`」と書き、`/sync-docs` がない。plan の Affected areas に入っていない |
| `docs/tech-debt/README.md:157` | ずれ(V-2) | (a)「shellcheck の対象に `insights-append.sh` がない」は S2 で解消したが、行は開いたまま(self-review L-5) |
| `docs/tech-debt/README.md:127` | ずれ(V-3) | 返済のきっかけ「Next touch to `pre_bash_guard.sh`」にこの PR が当たる。Codex 0.160.0 も `permission_mode` を送ることと「unsupported permissionDecision:ask」の文字列(self-review L-2)が証拠として入っていない |
| `docs/tech-debt/README.md:124` | ずれ(V-4) | コミットメッセージの guard の誤検知。返済のきっかけ「`pre_bash_guard.sh` への次のタッチ」にこの PR が当たるが、plan は Non-goals(`:49`)で先送りした。きっかけを書き直す必要がある |
| `docs/tech-debt/README.md:128` | ずれ(V-5) | RESOLVED のコメントにある「`/sync-docs` still writes no insight event」が、S3 で事実でなくなった |
| `docs/tech-debt/README.md`(新しい行) | 未記入(V-6) | self-review の Tech debt identified の 2 行(guard が捕まえない書き込みとコミットメッセージの形、M-1 の `cp`・`mv`・`sed -i` を含む。長いコマンドで guard が遅い)が、まだ README にない。plan の Non-goals(`:50`)と Risks(`:136`)は tech-debt に記録すると書いている |
| `.claude/hooks/pre_bash_guard.sh:35-38`(root と template)のコメント | ずれ(V-7) | self-review L-1。sed 経路ではタブも 2 文字の `\t` のまま残るので、タブの後ろの書き込み先を見ないことをコメントが書いていない |
| `scripts/verify.local.sh:9` の冒頭コメント | ずれ(V-9、前から) | `test : hook smoke tests (tests/test-check-mojibake.sh)` のまま。実際は `tests/test-*.sh` を全部回し、今回から実行権限の検査もする。この diff は 1 行上の static の行だけを直した |

## Observational checks

guard の probe(`docs/evidence/verify-2026-10-07-guard-bypass-and-hygiene.log` の「guard probe」)。payload は `jq -nc --arg` で組み、jq なしの経路は sh・bash・cat・grep・sed・printf・dirname・env・tr だけを置いた PATH で動かした(jq が見えないことを先に確かめた)。root と template の両方の guard に、AC1 の deny 5 形 × 7 モード、ask 3 形 × 6 モードと bypass、AC2 の 13 形、M-1 の 2 例を流し、142 例・284 判定で不一致は 0 件。手元は macOS(BSD grep / sed)だけで、GNU の環境では動かしていない。

旧 guard(origin/main の `pre_bash_guard.sh` と `lib_json.sh`)の対照: `permission_mode` が `bypassPermissions` の payload で、`git commit -m "$(id)"` は jq=deny / nojq=none、`rm -rf x && git commit -m "$(id)"` は ask / ask、`ls .git/ 2>&1` は ask / ask、`cat .env.example 2>/dev/null` は ask / ask、`grep X .env` は none / none。plan の Objective と Assumptions が書く問題と一致した。

`check_tech_debt_plan_refs` の対照: 関数を awk で取り出し、scratchpad の README で 3 通り(正しい参照だけ、壊れた参照 2 件を足す、README なし)を動かした。rc は 0 / 1 / 0 で、1 のときは壊れた 2 件だけを挙げた。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V-1 | LOW | doc drift | `docs/insights/README.md` の「Appending events」が、event を書く skill に `/sync-docs` を挙げていない | `docs/insights/README.md:117-119`(template も同じ) | /sync-docs で `/sync-docs` を足す(root と template) |
| V-2 | LOW | tech debt | tech-debt の 157 行目の (a) が解消済みなのに開いたまま | `docs/tech-debt/README.md:157`、`scripts/verify.local.sh:175` | /sync-docs で (a) を解消、(b) を継続に分ける(self-review L-5) |
| V-3 | LOW | tech debt | 127 行目のきっかけにこの PR が当たり、L-2 の証拠も入っていない | `docs/tech-debt/README.md:127`、`pre_bash_guard.sh:16-21` | 新しいきっかけ(Codex で ask を live-fire で確かめるときなど)と Codex 0.160.0 の 2 つの文字列を足す |
| V-4 | LOW | tech debt | 124 行目のきっかけにこの PR が当たるが、Non-goals で先送りしたことが行に書かれていない | `docs/tech-debt/README.md:124`、plan `:49` | きっかけを「次にコミットメッセージの guard を直すとき」などに書き直し、この PR で先送りしたことを書く |
| V-5 | LOW | tech debt | 128 行目のコメントの「`/sync-docs` still writes no insight event」が古い | `docs/tech-debt/README.md:128`、`.claude/skills/sync-docs/SKILL.md:33-40` | コメントに S3 で `/sync-docs` も event を書くようになったことを足す |
| V-6 | LOW | tech debt | self-review が提案した 2 行がまだ README にない | self-review の Tech debt identified、plan `:50`、`:136` | /sync-docs で 2 行を足す |
| V-7 | LOW | comment | タブの扱いをコメントが書いていない(L-1) | `pre_bash_guard.sh:35-38`、self-review L-1 | /sync-docs でコメントに 1 文足す(root と template)。正規表現を直すなら hook の変更になるので、/self-review から回し直す |
| V-8 | LOW | plan との差 | plan の Scope(`:34`)と AC4(`:105`)は直し方を `chmod +x` と `git update-index --chmod=+x` の 2 つと書くが、989886f2(L-3)以降は untracked のファイルに `git add --chmod=+x` を出す。index だけが 100644 で working tree に実行権限があるときも `chmod +x` を出す(害はない)。Progress checklist はこの形を書いていない | `scripts/verify.local.sh:272-282`、plan `:166` | AC4 は満たしている(ファイル名と直し方を出し、非 0 で終わる)。plan の本文は digest の対象なので、Progress checklist にメモを 1 行足す程度でよい。/verify では plan を直していない |
| V-9 | LOW | doc drift(前から) | `verify.local.sh` の冒頭コメントの test モードの説明が古い | `scripts/verify.local.sh:9` | 任意。直すなら「every tests/test-*.sh, failing ones without the exec bit」のように書く |

## Coverage gaps

- /test に回したもの: `tests/test-pre-bash-guard.sh`(AC1、AC2)、`tests/test-verify-local-hook-tests.sh`(AC4)、`tests/test-skill-insight-cycle.sh`(AC5)、`tests/test-archive-plan.sh`(AC6)、`./scripts/run-verify.sh` のテストの部分(AC7)、`go test ./...`(Test plan の Regression)。
- guard の probe は macOS の BSD grep / sed だけで動かした。GNU(CI の ubuntu)では動かしていない。self-review は ubuntu:24.04 と alpine で sed 経路の一致を確かめている。CI の結果で確かめるのがいちばん安い。
- Claude Code の実際の payload に `permission_mode: "bypassPermissions"` が入ることは、Claude Code の hook の文書と plan の Assumptions(`:56`)に頼っている。この session で動いている guard は main のチェックアウトの旧版なので、新しい guard の live の確認はマージのあとになる。おそらく文書どおり。未確認です。
- Codex が ask をどう扱うかは未確認のまま(tech-debt 127 行目)。
- `tests/test-verify-local-hook-tests.sh` の Case 1 は、`mktemp -d` の置き場所が git の work tree の外にあることを前提にしている(work tree の中なら `git add --chmod` の行が出て期待と食い違う)。CI の `/tmp` と macOS の既定では満たされる。`TMPDIR` を repo の中に向けた環境では動かしていない。
- `archive-plan.sh` の書き換えは、/pr の実際の archive(この plan 自身)で初めて実データに当たる。今の README にはこの plan への参照がないので、`Updated` の行は出ないはず。

## Verdict

- Verdict: pass
- Verified: AC1〜AC7 の静的な部分(上の表)。plan の digest が承認の行と一致すること。静的解析の gate 全部(shellcheck は広げた 95 ファイル)。ミラーの一致。guard の probe 284 判定と旧 guard の対照、`check_tech_debt_plan_refs` の対照
- Partially verified: AC7 の `run-verify.sh`(静的解析の部分だけ)。AC4 の直し方の表示は plan の文言より細かい(V-8)。文書のずれ V-1〜V-7、V-9 は LOW で、/sync-docs で扱える
- Not verified: テストの実行(/test の担当)、GNU 環境での guard の判定、新しい guard の live での発火、Codex の ask の扱い
