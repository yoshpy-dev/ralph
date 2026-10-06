# Self-review report: plan-visual-followups

- Date: 2026-10-06
- Plan: docs/plans/active/2026-10-06-plan-visual-followups.md
- Branch: fix/plan-visual-followups(base f9baf6a7、HEAD 93645c03)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、typo、シェルのクォート、jq と sed の式、コメントの正確さ、例外規則の文言が曖昧でないか・悪用に対して閉じているか、保守性)。対象は `git diff f9baf6a7...HEAD`(33 ファイル、+835/-89)。`.agents/skills/` と `templates/base/` はコピーなので、`.claude/` 側を 1 回読み、乖離だけを見た。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テストと linter は実行していない。

## Evidence reviewed

- `scripts/insights-append.sh` の `--cycle auto` 部分(`state_plan_path`、`state_cycle`、`resolve_auto_cycle`、`_cycle` を解決する箇所、header とヘルプの文言)を全行読んだ。`set -euo pipefail` の下で `$(...)` と `|| true` が失敗を握りつぶす形になっていることを確かめた。
- scratchpad で `--cycle auto --state-dir` を 12 通りの状態ファイルで実行した(リポジトリは触っていない)。複数行に整形した JSON、`"cycle": 2.0`、`1e3` は cycle を返し(2、2、1000)、`2.5`、`-1`、`true`、`100000000000000000000`、2 つの JSON の連結、配列、空の `plan_path`、`plan_path: null`、空白だけのファイルはすべて 1 に落ち、どれも exit 0 だった。header コメントの規則と一致する。
- `/cross-review` step 1(1.a〜1.c)と `--cycle auto` の規則を突き合わせた。`active-plan.json` なし、`cycle-count.json` なし、`plan_path` 不一致のどれも cycle 1 で、同じ。
- 新規・変更の 3 テスト(`tests/test-insights-append.sh` の Case 8、`tests/test-skill-insight-cycle.sh`、`tests/test-pr-owner-lookup.sh`)を全行読んだ。`tests/test-pr-owner-lookup.sh` は、any-base の jq フィルタから `ascii_downcase` を外した変更版を fixture に当てて、テストが変更を検出できるかを調べた(F-3)。
- `/pr` 5.c の `sed` と jq 2 本のクォートと優先順位を読んだ。`.headRepositoryOwner.login // "" | ascii_downcase` は `((.login // "") | ascii_downcase)` と読まれ、括弧で囲まれている。単一引用符の中に単一引用符はない。
- `post-implementation-pipeline.md` の例外の節、`/cross-review` step 8・9、`ralph-workflow.md` の一文を読み、`verify` と `test` の template の `## Verdict`、self-review template の `Merge:` と突き合わせた。`/pr` の pre-check 1〜3 が読む verdict の行は、この 3 つで全部かを確かめた。`/pr` は triage report を読まない。
- `ralph-workflow.md` の「PostToolUse の hook は Edit / Write にしか掛からない」を `.claude/settings.json`(`Edit|Write|MultiEdit`)と `.codex/hooks.json`(`Edit|Write|MultiEdit|apply_patch`)で確かめた。主張は正しい。
- ミラー: `scripts/insights-append.sh`、`.claude/settings.json`、`.ralph/core/settings.ralph.json`、2 つの rules は root と `templates/base/` が `cmp` で一致。5 つの skill は `.claude/` と `templates/base/.claude/`、`.agents/` と `templates/base/.agents/` が `cmp` で一致し、`.claude/` と `.agents/` の変更行も同じ。`/pr` 5.c の範囲は 2 面で md5 が同じ。
- 追加行を `#NNN`、`PR #`、`internal/`、`cmd/`、`this PR`、`rollout`、`TODO`、`FIXME`、`console.` で grep した。ヒットなし。`git diff --check` も空。
- step 8 の選択肢の番号を、他の文書が「Option 2」「digit 1–3」の形で参照していないか grep した。古い参照は残っていない(`Cap-reached Option 1` は番号が変わっていない)。
- `docs/tech-debt/README.md` を差分と両方向で突き合わせた。plan の進捗メモにある「`verify.local.sh` の shellcheck 対象に `insights-append.sh` がない」と、plan の Design decisions にある「cap を上げた追加の周は同じ cycle の記録が 2 つになる」は、register に行がなかった(下の「Tech debt identified」)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | 例外の条件 3 は self-review・verify・test の 3 つのレポートの verdict の行を固定するが、triage report の分類(`- After triage:` の行と、ACTION_REQUIRED / WORTH_CONSIDERING / DISMISSED の節)は固定していない。triage report は条件 2 の許可パスに入っており、確かめた結果を書き足す先でもあるので、「修正のコミット」が同じファイルの分類を書き換えても `git diff --name-only` と verdict の行の確認は通る。cross-review の判断(どの指摘が残っているか)を記録しているのはこのファイルで、`/pr` はこれを読まないので、書き換えの痕跡は triage report の履歴にしか残らない。条件 1(指摘が記録そのものを対象にしている)は修正する側の判断で、機械的には確かめられない | `.claude/rules/ralph/post-implementation-pipeline.md:20-30`(条件 2 に `cross-review-triage-<slug>.md` が入り、条件 3 の対象は 3 つのレポートだけ)、`:37-40`(確かめた結果は triage report に書く) | 修正のコミットでは triage report を変えない(許可パスは self-review・verify・test・sync-docs のレポートと insight event)とし、確かめた結果は別のコミットで triage report に書き足す(追記だけで、分類は変えない)と書く。または条件 3 に「triage report の `After triage:` の行と分類の節」を足す。`/cross-review` step 9 の文言も同じにそろえる。変更する面は rules の root と template、`/cross-review` の 4 面 |
| F-2 MEDIUM | security | `Bash(sed:*)` は前置き一致なので、`sed -i` だけでなく、`w` コマンド(任意のパスに書く)と GNU sed の `e` コマンド・`s///e` フラグ(シェルコマンドを実行する)も確認なしで通る。`w` と `e` は Edit / Write を経由しないので、PostToolUse の hook にも掛からない。plan のリスク欄と `ralph-workflow.md` の一文が扱うのは `-i` と hook の素通りだけで、GNU sed の `e` には触れていない。この許可は `templates/base/` の 2 ファイルにも入るので、`ralph upgrade` で下流の全プロジェクトに届く。ユーザーの指示どおりの変更なので、差し戻しの理由にはしない。`e` は GNU sed の機能で、この環境の sed は BSD のため手元では確かめていない(未確認) | `.claude/settings.json:39`、`templates/base/.claude/settings.json`、`templates/base/.ralph/core/settings.ralph.json`(同じ位置)、`docs/plans/active/2026-10-06-plan-visual-followups.md` の Risks and mitigations | PR 本文のリスク欄に `w` と GNU sed の `e` も書く。下流にも届くことを 1 行で足す(plan は「利用者が消したければ消せる」と書いている)。許可を絞るかはユーザーの判断で、コードの変更は要らない |
| F-3 LOW | maintainability | `/pr` 5.c の jq の述語 `(.headRepositoryOwner.login // "" | ascii_downcase) == ("<owner>" | ascii_downcase)` は 2 本のコマンドに別々に書かれていて、テストの fixture が大文字の login を試すのは with-base のフィルタだけ。any-base のフィルタの fixture(`other-base.json`)は login が小文字の `yoshpy-dev` だけなので、any-base の方から `ascii_downcase` を外しても、テストは通る。null の owner は PR 6 で試しているので、`// ""` を外すと落ちる | `tests/test-pr-owner-lookup.sh:186-191`(login は `yoshpy-dev`、`someone-else`、null)、`:207-208`。`ascii_downcase` を外した変更版 `select((.headRepositoryOwner.login // "") == "yoshpy-dev")` を今の fixture に当てると、書かれている式と同じ `release https://github.com/yoshpy-dev/ralph/pull/4` を返す(scratchpad で確認)。login を `Yoshpy-Dev` にすると変更版だけ空になる | `other-base.json` の PR 4 の login を `Yoshpy-Dev` にする(期待値の URL は変わらない)。1 行で済む |
| F-4 LOW | unnecessary-change | `scripts/insights-append.sh:251` に、既存の `_jq_filter` の行へ `# shellcheck disable=SC2016` が足された。SC2016 は info の指摘で、`scripts/verify.local.sh` の shellcheck は `--severity=warning` で、コメントに SC2016 を「gate-worthy defects ではない」と名指ししている。さらにこのスクリプトは shellcheck の対象一覧に入っていない。リポジトリの gate からは到達しない行で、`--cycle auto` と関係のない既存行の変更になる(`templates/base/` のコピーにも同じ行がある) | `scripts/verify.local.sh:149`(対象一覧に `insights-append.sh` なし)、`:154-157`(`--severity=warning` と SC2016 の注記) | 行を外す。手元で shellcheck を既定の重さで回した結果を残したいなら、コメントではなく verify レポートに書く |
| F-5 LOW | readability | `/pr` 5.c の「No PR」の項目が 1 つの段落に、2 回目の検索コマンド、「何か出たら止める」、「何も出なければ作り直す」、「gh が既存を理由に断ったら止める」、「ほかの失敗は止める」を詰めている(約 1100 文字)。前の項目(`PR exists`)は 1 つの手順で 1 文なので、ここだけ分岐が多く、読み手が条件の入れ子を追う必要がある | `.claude/skills/pr/SKILL.md:45` | 子の箇条書きに分ける(2 回目の検索、見つかったとき、見つからなかったとき、create が断られたとき)。`tests/test-pr-owner-lookup.sh` は行の中の ` --jq '` で式を取り出しているので、検索コマンドの行を独立した 1 行にしても壊れない |
| F-6 LOW | readability | `ralph-workflow.md` の一文は「シェルの in-place 編集(`sed -i`、ad-hoc scripts)ではなく Edit/Write」と言うが、理由(PostToolUse の hook が Edit/Write にしか掛からない)は `scripts/sync-skills.sh`、`scripts/insights-append.sh`、`scripts/archive-plan.sh` のような、この repo が追跡ファイルの書き換えを指示しているスクリプトにも同じく当てはまる。「ad-hoc」だけが線引きなので、読み手がこれらを禁止と受け取る余地がある。この種のスクリプトは skill 本文が実行を指示している | `.claude/rules/ralph/ralph-workflow.md:46-48`(`templates/base/` の同じ位置)、`.claude/skills/self-review/SKILL.md:47`(`insights-append.sh` を追跡ファイルに書く指示) | 「この repo のスクリプト(skill 本文が指示するもの)は除く」を半句足す。または「その場で書いたスクリプト」にする |

## Positive notes

- `--cycle auto` の失敗側がすべて 1 に落ちる。12 通りの状態で exit 0 のまま、header コメントの規則と一致した。壊れた JSON、複数の JSON、非整数、巨大な値、空の `plan_path` まで、`jq -rs` と `length == 1` の 1 つの条件で扱っていて、`case` の数字だけの検査が最後の保険になっている。
- 規則を 4 つの skill に書き写さず、スクリプトの 1 か所に置き、skill には `--cycle auto` と 1 行の注記だけを足した。`/cross-review` step 1 の規則と同じ内容で、`tests/test-skill-insight-cycle.sh` が 4 skill × 4 面を縛っている。
- `tests/test-pr-owner-lookup.sh` は SKILL.md から式を取り出して実行し、見つからなければ FAIL にする。EXIT trap の `[ ] && rm` が `set -e` の下で exit status を変える落とし穴は、コメント付きで `if` にして避けている。
- 例外の節は `git diff --name-only`、`git show --stat`、verdict の行の差分という機械的な確かめ方を書き、条件が 1 つでも外れたときの戻り先(全工程)まで `/cross-review` step 9 に書いている。step 8・9 は条件の文言を rules に委ね、二重に書いていない。
- ミラーが byte 単位で一致し(15 組の `cmp`)、step 8 の番号の付け替えに古い参照が残っていない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (a) `scripts/verify.local.sh` の shellcheck 対象一覧に `scripts/insights-append.sh` がない。(b) cap を上げた追加の `/cross-review` は `cycle-count.json` を上げないので、同じ cycle の insight event が 2 つになり、`ralph insights` は後の方を残す | (a) 警告レベルの欠陥が lint を通らずに入る。(b) `ralph insights` の cycle ごとの履歴で追加の周を区別できない | (a) 今回は cycle の値の変更で、lint の対象は範囲外(plan の進捗メモ)。(b) #203 で確かめた挙動で、変えると step 9 の書き込みが変わる | (a) `insights-append.sh` か shellcheck 対象一覧の次の変更。(b) `ralph insights` が追加の周を取り違えた報告 | docs/plans/active/2026-10-06-plan-visual-followups.md、このレポート |

_(上の行は `docs/tech-debt/README.md` に 1 行にまとめて追記した。)_

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。F-1 と F-2 は MEDIUM で、F-1 は例外の文言の閉じ方、F-2 はユーザー指示の許可の副作用の説明で、どちらも文書の修正で済む。
- Follow-ups: F-1 は rules と `/cross-review` step 9 の 2 か所の文言(4 面)。F-3 は fixture の 1 語。F-4 は 1 行の削除(root と template)。F-2 は PR 本文に書く。F-5 と F-6 は任意。F-1 と F-3 と F-4 を直すと、文書とテストとスクリプトが変わるので、`/self-review` から回し直す。

## Re-review of c24eab98

- Date: 2026-10-06
- Scope: 修正コミット `c24eab98`(`git diff 4f8924f6 c24eab98`、13 ファイル、+68/-28)だけ。F-1、F-3、F-5、F-6 の修正を確かめ、F-1 の修正が新しい穴を開けていないかを見た。F-2 は PR 本文に回す。F-4 は判断で残す。テストと linter は実行していない。
- Reviewer: reviewer subagent (Claude)、cycle 1(`cycle-count.json` は 1)

### Evidence reviewed

- `.claude/rules/ralph/post-implementation-pipeline.md:13-55` の例外の節を全文読み、修正前の版と突き合わせた。
- F-1 の新しい確かめ方を scratchpad の使い捨ての git リポジトリで実行した。`- After triage: ACTION_REQUIRED=1, ...` を持つ triage report を 1 コミットし、次のコミット(`<record>` の役)で、その行の上に `- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=1` を 1 行だけ挿入した。`git diff --name-only` は triage report だけを返し、`git diff <record>^ <record> -- <triage>` は `+` の 1 行だけを示し、`--numstat` は `1 0` だった。つまり新しい確かめ方の 2 つの条件は両方通る。一方、`scripts/xreview-helpers.sh` の `count_triage_findings` は `grep -m1` で最初の `After triage:` の行を読むので、ACTION_REQUIRED は 1 から 0 に変わった。旧内容が新内容の先頭と一致するかを `cmp` で見ると、一致しない(`differ: char 59, line 3`)。
- #203 の事例 `594354a9`(`docs/insights/events/2026-10-06-plan-visual-review.jsonl` の 1 行だけを変えた)を条件 2 に当てた。許可パスに入るので、例外の対象のままである。#203 の triage report は同じファイルを cycle ごとに書き直していた(`91a9c3f6`、`39c3c820`、`ce2b1d52`)が、これは全工程を回し直す側の話で、例外の経路ではない。
- F-3 と F-5 の式を、テストと同じ `grep` と `sed` の組で SKILL.md から取り出した。any-base は 1 行、with-base は 1 行で、式は変わっていない。新しい fixture(`Yoshpy-Dev`)に対して、書かれている式は `release https://github.com/yoshpy-dev/ralph/pull/4` を返し、`ascii_downcase` を外した変更版は空を返す。変更版を検出できるようになった。
- ミラー: `post-implementation-pipeline.md` と `ralph-workflow.md` は root と `templates/base/` が `cmp` で一致。`cross-review` と `pr` の SKILL.md は `.claude/` と `templates/base/.claude/`、`.agents/` と `templates/base/.agents/` が一致し、`.claude/` と `.agents/` の変更行も同じ。

### Per-finding status

| Finding | Status | Note |
| --- | --- | --- |
| F-1 MEDIUM | 一部解消、新しい MEDIUM(R-1)が残る | 修正のコミットが triage report を変えられなくなったのは確かめた(条件 2 に「触らない」が入り、`<commit>` の確かめも許可パスだけを通す)。`/cross-review` step 9 は「rules の書いたとおりに」と委ねていて、二重に書いていない。残る穴は `<record>` の確かめ方(R-1) |
| F-2 MEDIUM | PR 本文へ | `sed` の `w` と GNU `e` も含めて、PR 本文のリスク欄に書く。コードの変更なし |
| F-3 LOW | 解消 | `other-base.json` の PR 4 の login が `Yoshpy-Dev`。`ascii_downcase` を外した変更版は空になり、テストが落ちる |
| F-4 LOW | 判断で残す | SC2016 の disable は、既定の重さの shellcheck がその行を通るようにする。害はない。repo の gate(`--severity=warning`、対象一覧に `insights-append.sh` なし)からは到達しない行だが、残すと決めた。register の行(a)は残る |
| F-5 LOW | 解消 | 「No PR」が子の箇条書き 5 つになった。検索コマンドは 1 行のままで、テストの式の取り出しは壊れない。`PR exists` の項目との並びも自然 |
| F-6 LOW | 解消 | 「skill が実行を指示するスクリプトは構わない。規則の対象はシェルコマンドで追跡ファイルの中身をその場で書き換えること」が入り、`sync-skills.sh` を例に挙げている。`(sed -i, ad-hoc scripts)` の括弧は残り、新しい一文と矛盾しない |

### New findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| R-1 MEDIUM | maintainability | `<record>` の確かめ方「`git diff <record>^ <record> -- <triage>` は追加した行だけを示す」は、末尾への追記と、途中への挿入を区別しない。挿入した行も追加の行で、`--name-only` と `--numstat`(`K 0`)も通る。`count_triage_findings` は最初の `After triage:` の行を読む(`grep -m1`)ので、既存の行の上に別の `After triage:` の行を挿入すると、分類(ACTION_REQUIRED の数)が変わるのに確かめは通る。F-1 が塞ごうとした穴の、弱い形での再開になる。ルールの文は「only appends」と「no existing line changes」と言っていて、確かめ方がその意図に届いていない | `.claude/rules/ralph/post-implementation-pipeline.md:41-49`(`templates/base/` の同じ位置)。scratchpad の probe: 挿入後の差分は `+` の 1 行、`--numstat` は `1 0`、ACTION_REQUIRED は 1 から 0、旧内容は新内容の先頭と一致しない | 確かめ方を「旧内容が新内容の先頭と一致する(追記は末尾だけ)」にする。コマンドで書くなら、`<record>^` の `<triage>` の行数を N として、差分が 1 つのハンクで、ヘッダが `@@ -N,0 +N+1,K @@` であること、または `<record>^` の内容と `<record>` の先頭 N バイトが `cmp` で一致すること。末尾に改行がない旧ファイルは一致しないので、全工程の側に倒れる(安全側)。root と template の 2 面だけを変える。`/cross-review` step 9 は rules に委ねているので変更不要 |

観察(指摘ではない): 条件 1 は「記録そのものへの指摘」を許すが、条件 2 が triage report を除いたので、triage report の文言への指摘は例外を使えず、全工程を回し直す。安全側で、#203 の事例(insight event の値)には影響しない。

### Open findings after re-review

| Critical | High | Medium | Low |
| --- | --- | --- | --- |
| 0 | 0 | 2(R-1、F-2 は PR 本文に回す分) | 1(F-4、判断で残す) |

### Recommendation after re-review

- Merge: 可(CRITICAL と HIGH はない)。ただし R-1 は例外の規則を実際に使う将来の PR で効く穴なので、この PR で直すことを勧める。直すのは rules の 2 面(root と template)の 1 文で、直すと規則の変更なので `/self-review` から回し直しになる。直さない場合は、PR 本文の既知のギャップに書く。
- F-1 の「別コミット」という分け方そのものは正しく、#203 の事例も通る。R-1 は確かめ方の精度だけの問題。
