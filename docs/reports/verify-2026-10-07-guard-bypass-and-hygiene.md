# Verify report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Verifier: verifier subagent (Claude Opus 5.5)、cycle 1(`cycle-count.json` は 1)。cycle 2 は「Cycle 2」節、cap を 3 に上げた追加の回は「Cycle 2 (extra run)」節で、判定は最後の `## Verdict`
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

## Cycle 1 verdict

- Verdict: pass
- Verified: AC1〜AC7 の静的な部分(上の表)。plan の digest が承認の行と一致すること。静的解析の gate 全部(shellcheck は広げた 95 ファイル)。ミラーの一致。guard の probe 284 判定と旧 guard の対照、`check_tech_debt_plan_refs` の対照
- Partially verified: AC7 の `run-verify.sh`(静的解析の部分だけ)。AC4 の直し方の表示は plan の文言より細かい(V-8)。文書のずれ V-1〜V-7、V-9 は LOW で、/sync-docs で扱える
- Not verified: テストの実行(/test の担当)、GNU 環境での guard の判定、新しい guard の live での発火、Codex の ask の扱い

## Cycle 2

- Date: 2026-10-07
- Verifier: verifier subagent (Claude Opus 5.5)、cycle 2(`cycle-count.json` は 2)
- Scope: cycle 1 の verify(HEAD c45730cf)のあとのコミット。コードの変更は 0db1a97e(cross-review の 2 件)と ab3ee31c(self-review の C2-M1、C2-L1、C2-L3)で、触ったのは `pre_bash_guard.sh` の `tee_lead` の 1 行とそのコメント、`post_edit_verify.sh` のコメント、`tests/test-pre-bash-guard.sh`(どれも root と template)。ほかは記録と文書(4575949f の sync-docs、811e1452 の triage、14d48a77 の self-review)。AC1・AC2・AC7 は HEAD(ab3ee31c)で確かめ直し、AC3〜AC6 は戻っていないことを確かめた。静的解析は全部やり直した。テストは実行していない(/test の担当)。guard は scratchpad に置いた 4 つの版(origin/main、811e1452、0db1a97e、HEAD)と template の HEAD に、`jq -nc --arg` で組んだ同じ payload を jq あり・なしの両方で渡した
- Evidence: `docs/evidence/verify-2026-10-07-guard-bypass-and-hygiene.log` の「Cycle 2」節(手元だけ)。runner のログは `docs/evidence/verify-2026-10-07-020227.log`

plan の承認: `./scripts/plan-visual.sh digest` は `c07adf402bdb` を返し、承認の行と一致した。

### Spec compliance (cycle 2)

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 | Met(テストの実行は /test) | 811e1452 から HEAD までの guard の差は、コメントを除くと `tee_lead`(`.claude/hooks/pre_bash_guard.sh:62`)の 1 行だけ。判定の順は cycle 1 と同じで、deny の `case` が `:77-87` と `:91-102`、bypass の終了が `:107`、ask が `:111-125`。probe は deny 8 形(4 規則、バッククォートのコミットメッセージ、ask+deny の 2 形)× 7 モード(なし、`default`、`plan`、`acceptEdits`、`auto`、`dontAsk`、`bypassPermissions`)が deny、ask 4 形 × bypass 以外の 6 モードが ask、bypass で none。root と template、jq あり・なしで 336 判定、不一致 0。テストは E(`tests/test-pre-bash-guard.sh:254`)、F(`:265`)、G(`:271`、`default` と `auto`)、H の deny(`:295-296`) |
| AC2 | Met(テストの実行は /test) | AC2 が挙げる誤検知 6 形はモードなし・`default`・bypass で none、書き込み 7 形はモードなしと `default` で ask、bypass で none。156 判定、不一致 0。cycle 2 の修正の形(改行のあとの `tee .env </dev/null` と `.git/config`、`tee /tmp/out < .env` と `.git/config`、`cat >/tmp/o</repo/.git/HEAD`、`tee .env < input.txt`、`\tee .env`、`\tee .git/x`、バッククォートの中の `tee .env` と `.git/x`、`2>/dev/null` を挟む `tee` の 3 形、タブのあとの `tee`、`/usr/bin/tee .git/x`、`cat >.env<<EOF`)も HEAD で期待どおり(68 判定、不一致 0)。テストは C(`:181-202`、足した行は `:195-198`)、D(`:206-242`、足した行は `:219-233`)、H(`:291-294`)。AC2 が挙げていない形のうち、旧 guard より弱くなった形が残っている(V2-1)。AC2 の文言には当たらないが、plan の Risks と合わない |
| AC3 | Met(戻っていない) | `scripts/verify.local.sh` の変更は冒頭のコメント(`:9-10`、cycle 1 の V-9)だけ。shellcheck の対象は 95 ファイル(`scripts/` は 36 本、`scripts/insights-append.sh` を含む)で、`--severity=warning` で rc 0。`scripts/run-test.sh` は cycle 1 から変わっていない |
| AC4 | Met(戻っていない) | `run_hook_tests` と `hook_test_mode_problem` は cycle 1 から変わっていない(差はコメントだけ)。変更したテストを含む 4 本は index で 100755 |
| AC5 | Met(戻っていない) | `/sync-docs` の SKILL.md 4 面と `tests/test-skill-insight-cycle.sh` は cycle 1 から変わっていない。root と template は `cmp` で一致、`check-skill-sync.sh` は 13 skill が一致 |
| AC6 | Met(戻っていない) | `archive-plan.sh` は cycle 1 から変わっていない(root と template は `cmp` で一致)。`docs/tech-debt/README.md` の `docs/plans/active/` の参照は 3 か所で、どれもまだ active にあるこの plan を指す(157・158・160 行目、4575949f が足した)。`verify.local.sh` の参照の検査は OK。scratchpad に README と plan を写して HEAD の `archive-plan.sh` を動かすと、`Updated 3 reference(s) in docs/tech-debt/README.md` を出し、結果は単純な置き換えと一致し、active の参照は 0 になった |
| AC7 | Met(`run-verify.sh` は静的解析の部分だけ) | `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh` は単独でも `run-static-verify.sh` の中でも OK。`./scripts/run-static-verify.sh` は rc 0 |

### Static analysis (cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | `verify.local.sh` の static の段すべて、golang verifier(`lib_json.sh` が言語に分類されず full にフォールバック。gofmt ok、lint 0 issues)、branch secret scan(`2a22ba78..ab3ee31c` clean) |
| `shellcheck --severity=warning`(`verify.local.sh` と同じグロブ、95 ファイル) | PASS(rc 0) | shellcheck 0.11.0 |
| `shellcheck`(既定の重さ、811e1452 のあとに変えた `.sh` 5 本) | info だけ | SC1091、SC2016、SC1003、SC2329。guard の HEAD と 811e1452 はどちらも SC2016 の info が 1 件で、新しいコードはない |
| `./scripts/check-skill-sync.sh` | PASS | 13 skill |
| `./scripts/check-sync.sh` | PASS | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5(cycle 1 と同じ) |
| `./scripts/check-pipeline-sync.sh` | PASS | 6 文書 |
| `bash scripts/check-template-purity.sh` | PASS | |
| `sh -n` と `dash -n`(`pre_bash_guard.sh`、`post_edit_verify.sh`、root と template)、`bash -n tests/test-pre-bash-guard.sh` | PASS | 二重引用符の中のバッククォートは `` \` `` とエスケープされていて、構文は通る |
| ミラーの `cmp` | PASS | `pre_bash_guard.sh`、`lib_json.sh`、`post_edit_verify.sh`、`archive-plan.sh`、`run-test.sh`、`/pr` と `/sync-docs` の SKILL.md(`.claude/` と `.agents/`) |
| `git diff --check`(`811e1452 HEAD` と `origin/main...HEAD`) | PASS | 空 |
| 追加行の U+FFFD 検索(`origin/main...HEAD`) | PASS | 0 件 |
| `post_edit_verify.sh` のコメント以外の行の比較(811e1452 と HEAD) | 一致 | ab3ee31c の C2-L3 はコメントの折り返しだけ |

### Documentation drift (cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| plan の Risks(`docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md:136`) | ずれ(V2-1) | 旧 guard より弱くなる形を `.github/`・`.gitignore` と M-1(`cp`・`mv`・`sed -i` の後ろにリダイレクト)だけと書く。HEAD ではほかに `tee` の形が残る(Observational checks の表)。Risks は digest の対象なので、直すと承認の digest を取り直すことになる(c45730cf と同じ手順) |
| `docs/tech-debt/README.md:158` | ずれ(V2-2) | 「The old guard let all of them through as well, except the five M-1 examples」は実測と合わない。(a) に `tee` の形がない。返済のきっかけ「The next change to `pre_bash_guard.sh`」には 0db1a97e と ab3ee31c が当たった。(c) の「`redirect_lead` と `tee_lead` に 2 文字の `\t` を受け付けさせれば直る」は、`tee` の前の `\t` については 0db1a97e で済み、残るのは `>` のあとと `tee` のあとのタブ(probe で `echo x \| tee\t.env`、`tee -a\t.env`、`echo x >\t.env` は HEAD で jq=ask・nojq=none、origin/main は両方 none で、退行ではない) |
| `docs/tech-debt/README.md:160` | ずれ(V2-3) | (a) が「テストにない」と書く 3 形は、0db1a97e がすべて足した(`tests/test-pre-bash-guard.sh:233` の `cat >.env<<EOF`、`:220` の `/usr/bin/tee .git/x`、`:198` の `cat >/tmp/o</repo/.git/HEAD`)。self-review の C2-L2 と同じ。(a) に残るのは `plan`・`acceptEdits`・`dontAsk` をテストしていないことだけ |
| plan の Progress checklist | 未記入(V2-4) | cycle 2(triage 811e1452 の ACTION_REQUIRED 2 件、0db1a97e、self-review 14d48a77、ab3ee31c、この verify)の記録がない。Progress checklist は digest の対象外 |
| `pre_bash_guard.sh:53-61` のコメント(root と template) | 一致 | `tee` の前に置ける文字、`<` で止める理由、`>` で止めない理由、`tee < in.txt .env` を見ないこと、を HEAD の正規表現どおりに書いている |
| `pre_bash_guard.sh:37-43` のコメント | 一致 | `tee_lead` が `\n` と `\t` を受け付けること、`>` と `tee` のあとのタブは sed の経路で見ないこと |
| `tests/test-pre-bash-guard.sh:21-44` の見出しコメント | 一致 | C、D、H の列挙が足した行と合う |
| `post_edit_verify.sh:35-41`(root と template) | 一致 | C2-L3 で折り返しを直した |
| cycle 1 の V-1〜V-7、V-9 | 解消 | 4575949f で直っている(`docs/insights/README.md:117-119` に `/sync-docs`、tech-debt の 124・127・128・157 行目、guard のタブのコメント、`verify.local.sh:9-10`)。V-8 は cycle 1 の記録のとおり |
| `.codex/README.md:114-117`、`git-commit-strategy.md:70` | 一致 | deny の規則は cycle 1 から変わっていない |
| triage report の `pre_bash_guard.sh:51-52` | 記録(直さない) | 811e1452 の行番号。HEAD の `tee_lead` は `:62` |

### Observational checks (cycle 2)

4 つの版の比較(jq あり/なし、モードなし)。旧 guard より弱くなった形と、その形が入った版を表にした。手元は macOS(BSD grep / sed)だけ。

| Command | origin/main | 811e1452 | 0db1a97e | HEAD |
| --- | --- | --- | --- | --- |
| `tee < in.txt .env > /dev/null`(`.git/x` も同じ) | ask/ask | ask/ask | none/none | none/none |
| `tee out 2>&1 .env > /dev/null`(`.git/x` も同じ) | ask/ask | none/none | none/none | none/none |
| `tee out &>/dev/null .env > /dev/null` | ask/ask | none/none | none/none | none/none |
| `tee out 1>&2 .env > /dev/null` | ask/ask | none/none | none/none | none/none |
| `tee $(mktemp) .env > /dev/null` | ask/ask | none/none | none/none | none/none |
| `cmd \| tee -a log 2>&1 .git/config > /dev/null` | ask/ask | none/none | none/none | none/none |
| `tee out\ x .env > /dev/null` | ask/ask | none/none | none/none | none/none |
| `tee "a\|b" .env > /dev/null`、`tee "a;b" .git/x > /dev/null` | ask/none | none/none | none/none | none/none |
| `tee out 2>&1 .env`(後ろにリダイレクトなし) | none/none | none/none | none/none | none/none |
| `tee $(mktemp) .env`、`tee out 1>&2 .env`(後ろにリダイレクトなし) | none/none | none/none | none/none | none/none |
| `tee < in.txt .env`(後ろにリダイレクトなし。`.env` に書く) | none/none | ask/ask | none/none | none/none |

どの形も、`tee` の書き込み先の前の引数に、`tee_lead` の引数の読み取りが止まる文字(`<`、`&`、`)`、`;`、`|`、`\`)がある。旧 guard がこれらに ask を返したのは、後ろに `>` があって `*".env"*">"*` か `*".git/"*">"*` に当たったからで、M-1 と同じ偶然による検出だった。後ろに `>` がなければ旧 guard も何も返さない。依頼にあった `tee out 2>&1 .env` は、origin/main も HEAD も jq あり・なしとも none で、旧 guard より弱くはない。`2>&1` を挟んで後ろに `>` が続く形は弱くなっていて、これは 811e1452(cycle 1 の S1)のときからそうだった。`<` を挟む形は 0db1a97e で弱くなった。

逆に HEAD が旧 guard より多く ask を返す形もある: `tee -a .env`、`echo x > ./.env`、`tee out >/dev/null .env`、`tee out.txt 2>/dev/null .env`、`echo x \| \tee -a .env`(どれも書き込みで、ask が正しい)。誤検知は `echo x \| tee log > "x .env"`(空白を含む引用符付きのリダイレクト先)の 1 例で、self-review の C2-L1 が挙げたものと同じ。

cycle 1 の記録の訂正: cycle 1 の Spec compliance の AC2 の行は、M-1 の 2 例が訂正後の Non-goals と Risks に合うと書いた。cycle 1 では `tee` の引数に `&` などを挟む形を試しておらず、`2>&1` を挟む形が c45730cf の時点ですでに弱くなっていたことを見落としていた。cycle 1 の Coverage gaps の「今の README にはこの plan への参照がないので、`Updated` の行は出ないはず」は、4575949f が参照を 3 か所足したので当たらなくなった。/pr の archive は `Updated 3 reference(s)` を出すはずで(scratchpad で確認)、step 8 のとおり README も同じコミットに入れる必要がある。

### Findings (cycle 2)

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V2-1 | LOW | doc drift | plan の Risks が、旧 guard より弱くなった形を実際より狭く書いている。M-1 のほかに、書き込み先の前の引数に `<`・`&`・`)`・`;`・`\|`・`\` がある `tee` で、後ろに `>` が続く形が残る | plan `:136`、上の比較の表、`.claude/hooks/pre_bash_guard.sh:62` | /sync-docs で Risks に足し、承認の digest を取り直す。PR 本文が Risks を写すなら同じ内容にする |
| V2-2 | LOW | tech debt | tech-debt 158 行目の「except the five M-1 examples」が実測と合わない。(a) に `tee` の形がなく、返済のきっかけはもう当たった。(c) の直し方は半分済んでいる | `docs/tech-debt/README.md:158`、上の比較の表 | (a) に `tee` の形(`tee < in.txt .env > /dev/null`、`tee out 2>&1 .env > /dev/null` など)を足し、冒頭の文を直し、きっかけを書き直す。(c) を `>` のあとと `tee` のあとのタブに絞る |
| V2-3 | LOW | tech debt | tech-debt 160 行目の (a) の 3 形は、もうテストにある | `docs/tech-debt/README.md:160`、`tests/test-pre-bash-guard.sh:198`、`:220`、`:233` | (a) のその部分を解消済みにする(self-review C2-L2) |
| V2-4 | LOW | plan | plan の Progress checklist に cycle 2 の記録がない | plan `:154-173` | /sync-docs で 1〜2 行足す(digest は変わらない) |

### Coverage gaps (cycle 2)

- /test に回したもの: `tests/test-pre-bash-guard.sh`(0db1a97e と ab3ee31c で足した行を含む)と、`./scripts/run-verify.sh` のテストの部分。
- guard の probe は macOS の BSD grep / sed だけで動かした。GNU(CI の ubuntu)では動かしていない。
- 上の表の弱くなった形は、どれもテストで固定していない(none とも ask とも)。plan の AC には入っていないので AC は満たすが、次に `tee_lead` を変えたときに判定が動いても気づけない。
- 新しい guard が live の payload で発火すること、Codex の ask の扱いは、cycle 1 と同じく未確認です。

## Cycle 2 (extra run)

- Date: 2026-10-07
- Verifier: verifier subagent (Claude Opus 5.5)。cross-review cycle 2(44ab9ef7)のあと cap を 3 に上げた追加の回。`cycle-count.json` は 2 のままなので、insight event の cycle も 2 になる
- Scope: 69d1c581..b94a9106。コードの変更は 3c0ba22a、206d8335、b94a9106 で、`pre_bash_guard.sh` の `word_end`・`tee_lead`・`env_target` とコメント(root と template)、`tests/test-pre-bash-guard.sh` の C に 2 行、D に 9 行。ほかは記録(77e4e542 の self-review など)。AC3〜AC6 のファイル(`lib_json.sh`、`post_edit_verify.sh`、`scripts/`、skill の 4 面、新しいテスト 3 本)は `git diff --quiet 69d1c581 HEAD` で変化なし。テストは実行していない(/test の担当)
- Evidence: `docs/evidence/verify-2026-10-07-guard-bypass-and-hygiene.log` の「Cycle 2 (extra run)」節(手元だけ)。runner のログは `docs/evidence/verify-2026-10-07-042818.log`

plan の承認: `./scripts/plan-visual.sh digest` は `d8f86292d5f1` を返し、承認の行と一致した。

AC の判定: AC1 と AC2 は HEAD で Met。scratchpad の HEAD と template の guard に、`jq -nc --arg` で組んだ payload を jq あり・なしで渡し、920 判定で不一致は 0 だった。内訳は、deny 11 形 × 7 モード、ask 6 形 × 7 モード(bypass だけ none)、AC2 の誤検知 6 形を含む none 14 形 × 3 モード、AC2 の書き込み 7 形を含む書き込み 23 形 × 3 モード。AC3〜AC6 は戻っていない(上の `git diff --quiet`。tech-debt README の参照の検査は OK)。AC7 は Met で、`run-verify.sh` は静的解析の部分だけ確かめた。

静的解析: `./scripts/run-static-verify.sh` は rc 0(verify.local.sh の static の段、golang verifier、secret scan `2a22ba78..b94a9106` clean)。単独で走らせた `check-skill-sync.sh`(13 skill)、`check-sync.sh`(IDENTICAL 164、DRIFTED 0)、`check-pipeline-sync.sh`、`check-template-purity.sh` も rc 0。`shellcheck --severity=warning` は広げた 95 ファイルで rc 0(0.11.0)。既定の重さでは、guard は 44ab9ef7 と HEAD がどちらも SC2016 の info 2 件、テストの SC2016 の info は 7 件から 17 件に増えた。増えたのは、足した行がシングルクォートの中にバッククォートを書いているためで、意図した文字列。`sh -n` と `dash -n`(guard の 2 面)、`bash -n`(テスト)、guard の `cmp`、`git diff --check`、追加行の U+FFFD 検索(0 件)も通った。

版ごとの比較(モードなし、jq/nojq):

| Command | origin/main | 44ab9ef7 | 3c0ba22a | 206d8335 | HEAD |
| --- | --- | --- | --- | --- | --- |
| ``x=`tee .git` ``、``x=`printf x > .git` `` | ask/ask | none/none | ask/ask | ask/ask | ask/ask |
| ``x=`tee /tmp/a` .env`` | none/none | ask/ask | ask/ask | none/none | none/none |
| `tee /tmp/build.log # .env is read separately` | none/none | ask/ask | none/none | none/none | none/none |
| ``echo x > `pwd`/.git/x``、``tee `pwd`/.git/x`` | none/none | ask/ask | none/none | none/none | ask/ask |
| ``echo x > `pwd`/.git/x 2>/dev/null``、``tee `pwd`/.git/x > /dev/null`` | ask/ask | ask/ask | none/none | none/none | ask/ask |
| `tee a#b .env > /dev/null`、``tee `mktemp` .env > /dev/null`` | ask/ask | ask/ask | none/none(`` `mktemp` `` は ask/ask) | none/none | none/none |
| `tee "a #b" .env > /dev/null` | ask/none | ask/ask | none/none | none/none | none/none |
| `tee a#b .env`、``tee `cmd` .env`` | none/none | ask/ask | none/none(`` `cmd` `` は ask/ask) | none/none | none/none |
| `tee "build .env.log"` | none/none | ask/ask | ask/ask | ask/ask | ask/ask |
| `echo x > $(pwd)/.env 2>&1`、`echo x > $(pwd)/.git/x 2>/dev/null` | ask/ask | none/none | none/none | none/none | none/none |
| `echo x > "$(pwd)/.git/x"` | none/none | none/none | none/none | none/none | none/none |
| ``echo x > `echo .env` 2>/dev/null`` | ask/ask | none/none | none/none | none/none | none/none |

cross-review cycle 2 の 1 件目(``x=`tee .git` ``)と 2 件目のコメントの側、206d8335 の誤検知、self-review の C3-L1(`` `pwd` `` で組んだ書き込み先)は HEAD で直っている。C3-L1 の形は、HEAD では後ろにリダイレクトがなくても ask を返し、origin/main より多く捕まえる。self-review の extra run は「コードは直さず記録する」と勧めていたが、b94a9106 がそのあと `word_char` にバッククォートを戻した。self-review の記録はその時点のもので、C3-L1 の `` `pwd` `` の形を未解決として register に写す必要はない。HEAD が旧 guard より弱いまま残るのは、`#` かバッククォートで引数の読み取りが止まる `tee` に後ろの `>` が付く形(C3-L2)と、`$(...)` で組んだ引用符なしの書き込み先に後ろのリダイレクトが付く形。`$(pwd)` の形は 44ab9ef7 の時点ですでに none で、この回の退行ではない。``echo x > `echo .env` 2>/dev/null`` も同じ「コマンド置換で組んだ書き込み先」に入る。HEAD の 54〜68 行目と 45〜47 行目のコメント(`$(pwd)` は見ない、`` tee `cmd` .env `` は見ない、`tee a#b .env` は見ない)は、この結果と合う。

### Documentation drift (extra run)

直していない(/sync-docs の担当)。

| ID | Severity | Doc | Drift |
| --- | --- | --- | --- |
| V3-1 | LOW | `docs/tech-debt/README.md:158` | (a) の止める文字の列挙(`&`、`<`、`\`、`;`、`\|`、`)`)に `#` とバッククォートがない。例として `tee a#b .env > /dev/null`、``tee `mktemp` .env > /dev/null``、`tee "a #b" .env > /dev/null`(origin/main は jq の経路だけ ask)を足す。既知の誤検知 `tee "build .env.log"`(triage の WORTH_CONSIDERING 2 件目)もない。(a) が `"$(pwd)/.git/x"` を旧 guard も通した形として書くのは引用符付きでは正しいが、`echo x > $(pwd)/.env 2>&1` のように引用符なしで後ろにリダイレクトが続くと、origin/main は ask を返していた(C3-L1 の後半)。`` `pwd` `` で組んだ形は b94a9106 で見えるようになったので、残る注意は `$(...)` の形だけ |
| V3-2 | LOW | plan の Progress checklist(`:154-175`) | cross-review cycle 2(44ab9ef7、ACTION_REQUIRED 1、WORTH_CONSIDERING 1)、cap を 3 に上げたこと、3c0ba22a・206d8335・b94a9106 と self-review 77e4e542 の記録がない。digest の対象外。Risks の「`&`、`<`、`\` など」は `#` とバッククォートも含む書き方なので、digest の対象の本文は直さなくてよい |
| V3-3 | LOW | `tests/test-pre-bash-guard.sh:28-35` の見出しコメント | D の列挙に、バッククォートを含む書き込み先(`` `pwd`/.git/x ``、b94a9106 の 4 行)がない。C の ``x=`tee /tmp/a` .env`` は「...」に含まれると読める |

### Coverage gaps (extra run)

- /test に回したもの: `tests/test-pre-bash-guard.sh`(この回で足した 11 行)と、`./scripts/run-verify.sh` のテストの部分。
- 上の表で HEAD が旧 guard より弱い `#`・バッククォート・`$(...)` の形は、どれもテストで固定していない(tech-debt 160 行目の (a) のとおり)。
- guard の probe は macOS の BSD grep / sed だけで動かした。GNU では動かしていない。新しい guard の live での発火と Codex の ask の扱いも、これまでと同じく未確認です。

## Verdict

- Verdict: pass(Cycle 2 (extra run)、HEAD b94a9106)
- Verified: plan の digest(`d8f86292d5f1`)が承認の行と一致すること。AC1 と AC2 を HEAD で確かめ直した(920 判定、不一致 0、root と template、jq あり・なし、7 モード)。AC3〜AC6 のファイルが 69d1c581 から変わっていないこと。静的解析の gate 全部(shellcheck は 95 ファイル)、ミラーの一致、diff の空白と U+FFFD。cross-review cycle 2 の 1 件目と 2 件目のコメントの側、206d8335 の誤検知、C3-L1 が HEAD で直っていること
- Partially verified: AC7 の `run-verify.sh`(静的解析の部分だけ)。文書のずれ V3-1〜V3-3 は LOW で、/sync-docs で扱える。どれも plan の digest の対象ではない
- Not verified: テストの実行(/test の担当)、GNU 環境での guard の判定、新しい guard の live での発火、Codex の ask の扱い
