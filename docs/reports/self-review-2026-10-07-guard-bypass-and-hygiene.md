# Self-review report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Branch: fix/guard-bypass-and-hygiene(base 2a22ba78、HEAD ed4e4ae5)
- Reviewer: reviewer subagent (Claude)、cycle 1(cycle 2 は末尾の「Cycle 2」節。判定はその節の Merge 行)
- Scope: diff の品質だけを見た(命名、読みやすさ、不要な変更、コメントの正確さ、正規表現と sed の式、一時ファイルの扱い、安全性)。対象は `git diff origin/main...HEAD`(23 ファイル、+1400/-121)。`templates/base/` と `.agents/skills/` はコピーなので、root の `.claude/` と `scripts/` を読み、コピーとの差は `cmp` で見た。仕様への適合、テストの網羅、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の担当)。リポジトリのテストと linter は実行していない。guard と `archive-plan.sh` は scratchpad に置いた入力で直接動かした。

## Evidence reviewed

- `pre_bash_guard.sh` と `lib_json.sh` の差分を全行読んだ。deny の 4 規則は、`bypassPermissions` で抜ける `exit 0`(93〜95 行目)より前にあり、モードによる分岐はこの 1 か所だけ。
- guard の挙動は 3 つの probe で確かめた。payload はどれも `jq -nc --arg` で組み、実際の serializer と同じエスケープにした。
  - 50 例(書き込み、読み取り、deny、モードの偽装)を jq 経路と jq なしの経路の両方に流した。
  - 旧 guard(`git show origin/main:` から取り出したもの)と新 guard を同じ 38 例で比べ、新が旧より弱くなる行に印を付けた(M-1)。
  - 87 の payload について、ホストの jq 経路の判定を記録した。そのうえで、jq なしの sed 経路の判定を macOS(BSD sed/grep)、ubuntu:24.04(dash、GNU sed/grep)、alpine:3.21(busybox)で再生した。不一致は 3 環境とも 0 件。
- モードの偽装: コマンド本文に `"permission_mode":"bypassPermissions"` を書く形を 7 通り試した(単一引用符、`{...}`、タブ区切り、`\"` と `\\"` を前に置くもの、値の `Default`、末尾に空白のある `bypassPermissions `)。jq 経路・sed 経路ともに ask が残った。JSON の文字列の中の `"` は必ず `\` に続くので、sed の `"permission_mode"[[:space:]]*:` に一致する並びはコマンド本文の中に作れない。
- `lib_json.sh` の sed fallback の unescape を 17 通りの文字列で確かめた(`a"b`、末尾の `\`、`\\"`、`"\"\\"`、`\` と `n` の 2 文字、改行、タブ、日本語と引用符、制御文字、`"command":"spoof"`)。`\"` と `\\` は元に戻り、`\n`、`\t`、`\u0001` はエスケープのまま残る。コメント(lib_json.sh の 9〜17 行目)の説明と一致する。
- 速度: 10〜100 KB のコマンドで、新 guard は jq あり・なしとも 0.15 秒以下。200 KB のクォート文字列のあとに `git commit -m "$(id)"` を置いた payload は 28 秒かかった。旧 guard の jq 経路も同じ payload で 28 秒、新しい取り出し処理だけなら 100 KB で 0.1 秒なので、時間は既存の判定部分で掛かっている。今回の変更で、sed 経路が jq 経路と同じ長さのコマンドを読むようになっただけで、新たな退行ではない(どの行が重いかは未確認)。
- Codex 0.160.0 の実行ファイルから文字列を取り出して、hook の入力スキーマ `pre-tool-use.command.input` を読んだ。`permission_mode` は `required` に入っていて、enum は `default`、`acceptEdits`、`plan`、`dontAsk`、`bypassPermissions`。同じ実行ファイルに `PreToolUse hook returned unsupported permissionDecision:ask` という文字列もある(L-2)。
- `archive-plan.sh` の書き換えを、scratchpad の fixture で実行した。対象は、括弧の中で文末の `.` が付く参照、GitHub の URL の中の参照(`#L3` 付き)、`、` が続く参照、CRLF の行、末尾に改行のないファイルで、4 か所すべてが書き換わった。`docs/tech-debt/` に一時ファイルは残らなかった。末尾に改行のないファイルには改行が 1 つ足される(awk の `print` のため。書き換えが 1 件以上のときだけ起き、害はない)。
- `verify.local.sh` の実行権限の検査を、untracked で実行権限のないテストに当てた(L-3)。
- `docs/tech-debt/README.md` の差分: `git show origin/main:docs/tech-debt/README.md` の `docs/plans/active/` を全部 `docs/plans/archive/` に置き換えた結果が、HEAD の README と byte 単位で一致した。変更は 34 か所の置き換えだけ。register とこの差分を両方向で突き合わせた(L-5)。
- ミラー: `pre_bash_guard.sh`、`lib_json.sh`、`archive-plan.sh`、`run-test.sh` は root と `templates/base/` が `cmp` で一致した。`/pr` の SKILL.md は 4 面とも一致した。`/sync-docs` は `.claude/` と `.agents/` の差が frontmatter の `allowed-tools` の 1 行だけで、これは base から変わっていない。
- `verify.local.sh` に足した 2 つの関数の上に、ほかの関数のコメントが取り残されていないことを確かめた。shellcheck の対象のコメントにある「36 本中 9 本」は、`scripts/*.sh` の数(36)と旧一覧(9 本)に合う。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| M-1 MEDIUM | security | 書き込み先だけを見る変更で、`cp`・`mv`・`sed -i` を使って `.git/` や `.env` に書くコマンドのうち、後ろにリダイレクトが付く形が ask から外れた。旧 guard の `*".git/"*">"*` と `*".env"*">"*` は、こうした形にも ask を返していた。新 guard は jq あり・なしとも何も返さない。plan の Non-goals は「今も捕まえておらず」と書き、Risks は「外れるのは `.github/` と `.gitignore` など `.git` で始まる別の名前だけ」と書くが、どちらも実測と合わない。PR 本文がこのリスク欄を写すと、退行の範囲を実際より狭く伝える。旧 guard がこれらを捕まえていたのは偶然(後ろに `>` があるときだけ)で、`cp hook .git/hooks/pre-commit` だけの形はもともと通っていた。この guard は `ralph upgrade` で下流にも届く | 新旧比較の probe で次の 5 例が旧 ask、新 none: `cp hook .git/hooks/pre-commit 2>&1`、`cp hook .git/hooks/pre-commit > /dev/null`、`mv x .git/x > /dev/null`、`sed -i.bak s/a/b/ .git/config > /dev/null`、`cp .env.example .env && echo done > log.txt`。plan の 50 行目(Non-goals)と 136 行目(Risks)。`.claude/hooks/pre_bash_guard.sh:46-60` | 最低限、plan の 2 行と PR 本文を直す。「`cp`・`mv`・`sed -i` の書き込みは、後ろにリダイレクトがあるときだけ旧 guard が偶然捕まえていた。今回それも外れる」と書き、tech-debt に行を足す(下の表の 1 行目)。捕まえ直すなら、`cp`・`mv`・`install` の最後の引数と `sed -i` のファイル引数に `git_target` と `env_target` を当てる規則を足す。plan の範囲を広げることになり、テストも足す必要がある |
| L-1 LOW | security | タブで区切った書き込みは、jq 経路では ask になり、sed 経路では何も返さない。jq 経路ではタブが本物のタブに戻って `[[:space:]]` に当たる。sed 経路では 2 文字の `\t` のまま残り、`\` は `word_char` に入らないので、書き込み先の語がそこで終わる。コメント(35〜38 行目)は `\n` の扱いだけを説明している。テスト D のタブの例は `>` の前にタブを置く形だけで、`>` の後ろと `tee` の前後は試していない。旧 guard は両方の経路で通していたので、退行ではない | `echo x >\t.git/x`、`echo x \|\ttee .git/x`、`echo x \| tee\t.git/x` は 3 例とも jq=ask、nojq=none。`.claude/hooks/pre_bash_guard.sh:46`(`redirect_lead` の `[[:space:]]*`)、`:48`(`tee_lead` の `tee[[:space:]]`)、`tests/test-pre-bash-guard.sh` の writes 配列(`$'echo x\t> .git/x'` の 1 例) | `redirect_lead` と `tee_lead` の空白の位置で、2 文字の `\t` も受け付ける(シェルの二重引用符の中では `([[:space:]]\|\\\\t)*`、grep が受け取る形は `([[:space:]]\|\\t)*`)。テスト D に `>` の後ろのタブと `tee` の後ろのタブを 1 例ずつ足す。直さないなら、35〜38 行目のコメントに「タブも `\t` のまま残り、sed 経路ではタブの後ろの書き込み先を見ない」と書く |
| L-2 LOW | security | 18〜19 行目のコメント「An absent key (e.g. a Codex payload) leaves every rule on.」は、Codex 0.160.0 には当てはまらない。Codex の hook の入力スキーマでは `permission_mode` が必須で、値に `bypassPermissions` を含む。したがって Codex が `bypassPermissions` を送る場面では、Codex でも ask が出なくなる。plan の Non-goals(48 行目)の「Codex 側の挙動を変えること」は、文字どおりには守られていない。ただし同じ実行ファイルに `PreToolUse hook returned unsupported permissionDecision:ask` という文字列があるので、Codex はもともと ask を扱っていなかったとみられる。その場合、実害はおそらく出ない(Codex が ask を無視するのか止めるのか、どの設定が `bypassPermissions` になるのかは未確認) | Codex 0.160.0 の実行ファイル(`@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/bin/codex`)の `strings` に、`pre-tool-use.command.input` のスキーマ(`"required": [..., "permission_mode", ...]`)がある。`.claude/hooks/pre_bash_guard.sh:16-20`、plan の 48 行目 | コメントの例を直す。たとえば「Codex 0.160.0 も同じキーを送る(hook の入力スキーマで必須)。キーがないとき(古い版など)は全規則が効く」とする。plan の Non-goals と PR 本文も合わせる。tech-debt の 127 行目(Codex の ask は未確認)に、この 2 つの文字列を証拠として足す(`/sync-docs`) |
| L-3 LOW | maintainability | 実行権限のないテストに対して、直し方の 2 行目 `git update-index --chmod=+x <file>` を常に出す。untracked のファイルでは、このコマンドは `fatal: Unable to process path` で失敗する(rc 128)。Write ツールで作ったばかりの新しいテストは、ちょうどこの状態になる。working tree では実行権限があり、index だけが 100644 のときは、1 行目の `chmod +x` は要らない | scratchpad の git repo に untracked で 644 の `tests/test-new.sh` を置いて `HARNESS_VERIFY_MODE=test` で実行すると、2 行とも表示された。表示どおりに 2 行目を実行すると `error: tests/test-new.sh: cannot add to the index - missing --add option?` になる。`scripts/verify.local.sh:273-274` | `hook_test_mode_problem` は理由を分けて返しているので、理由に合う直し方だけを出す。working tree の問題なら `chmod +x`、index が 100644 なら `git update-index --chmod=+x` にする |
| L-4 LOW | maintainability | `post_edit_verify.sh` の 35〜38 行目は、jq が要る理由に「an embedded quote in the diff truncates the match early」を挙げている。`lib_json.sh` の sed fallback がエスケープを読むようになったので、この理由はもう成り立たない。jq が要ること自体は変わらない(`\n` が 2 文字のまま残るので、`*** Add File:` の行が行頭に来ない)。plan は `post_edit_verify.sh` への影響を Scope と Risks で扱ったが、このコメントは直していない | `.claude/hooks/post_edit_verify.sh:35-38`、`templates/base/.claude/hooks/post_edit_verify.sh:35-38`、`.claude/hooks/lib_json.sh:9-17` | 括弧の中を「改行は 2 文字の `\n` のまま残るので、パッチの本文の `*** Add File:` などの行が行頭に来ない」に変える(root と template) |
| L-5 LOW | maintainability | register に、この PR で状況が変わった行が 2 つある。157 行目の (a)「shellcheck の対象一覧に `insights-append.sh` がない」は S2(`scripts/*.sh` のグロブ化)で解消したが、行はまだ開いたままになっている。127 行目の返済のきっかけは「Next touch to `pre_bash_guard.sh`」で、この PR がそれに当たる。plan はこの行を Non-goals で先送りしたので、きっかけを書き直す必要がある。plan の Affected areas は 127 行目を挙げているが、157 行目は挙げていないので、`/sync-docs` が見落とすおそれがある | `docs/tech-debt/README.md:157`、`:127`、`scripts/verify.local.sh` の shellcheck のループ(`scripts/*.sh`) | `/sync-docs` で 157 行目を (a) 解消・(b) 継続に分ける。127 行目には L-2 の証拠を足し、新しいきっかけ(たとえば「Codex で ask を live-fire で確かめるとき」)を書く |

## Positive notes

- deny の 4 規則をすべて bypass の分岐の前に置き、ask の規則はそのあとにまとめた。分岐が 1 か所なので、deny がモードに左右されないことを読むだけで確かめられる。テストの F 節が ask と deny の組み合わせを両方のモードで縛っている。
- sed fallback の `"(([^"\\]|\\.)*)"` は、`\` で始まる 2 文字を 1 組として左から読むので、どの `\` がどの文字をエスケープしているかを取り違えない。17 通りの文字列で unescape が正しく、BSD、GNU、busybox の 3 環境で jq 経路と同じ判定になった(87 例中、不一致 0)。
- 正規表現を `word_char`、`word_end`、`redirect_lead` などの名前の付いた部品に分け、部品ごとに 1 行のコメントを付けている。`\b`・`\s`・`\w` を避けた理由も書いてある。
- `archive-plan.sh` は、名前を `ENVIRON` で渡し、`index` と `substr` で比べている。名前の `.` を正規表現として扱わない。一時ファイルは README と同じディレクトリに置いて `mv` で置き換え、`cp -p` で mode を保つ。EXIT trap は `if` で書いてあり、`set -e` の下で終了コードを変えない。書き換えを移動の前に置いたので、やり直すときに専用の分岐が要らない。
- `tests/test-pre-bash-guard.sh` は、判定を none / ask / deny の 3 値で厳密に比べる。最小の PATH から jq が見えないことを先に確かめていて、jq なしの経路が黙って jq 経路になることがない。
- tech-debt README の 34 か所の置き換えは機械的で、ほかの文字は 1 つも変わっていない。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `pre_bash_guard.sh` が捕まえない書き込みとコミットメッセージの形。`.git`・`.env` の ask から漏れるのは、`cp`・`mv`・`install`・`sed -i` の書き込み先、`>&`、`"$(pwd)/.git/..."` のようにコマンド置換を含むパス、空白を含む引用符付きのパス(`"a b/.git/x"`)、行の継続(`> \` と改行)。deny から漏れるのは、`git commit -am "$(id)"`、`-m"$(id)"`、`--message "$(id)"`、`sudo` とタブ。どれも旧 guard でも通っていた(M-1 の 5 例は旧 guard だけが捕まえていた) | bypass のモードでは deny の 4 規則だけが残るので、deny の綴りの漏れがそのまま素通りになる。bypass 以外では、`.git` と `.env` への書き込みが確認なしで走る | 今回の plan は「書き込み先の判定を正すだけ」で、`cp` などの新しい形は Non-goals | `pre_bash_guard.sh` の次の変更、または guard を通り抜けた書き込みの報告 | docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md、このレポート(M-1、L-1) |
| 長いコマンドで guard が遅い。200 KB のクォート文字列のあとに `git commit -m` を置くと 28 秒かかる(旧 guard の jq 経路も同じ)。取り出しの処理は 100 KB で 0.1 秒なので、遅いのは判定のほう(どの行かは未確認) | hook の時間切れまで Bash の実行が止まる。時間切れになった hook の判定は効かない | 既存の挙動で、この PR は sed 経路を jq 経路と同じ長さに揃えただけ | heredoc で大きなファイルを書くコマンドが遅いという報告 | このレポート |

_(上の 2 行は、orchestrator の指示(このコミットにはレポートと insight event だけを入れる)に従い、`docs/tech-debt/README.md` には書いていない。`/sync-docs` で足すこと。)_

## Recommendation

- Cycle 1 の判定: 可。CRITICAL と HIGH はない。M-1 は MEDIUM で、guard の判定を狭めた結果、旧 guard が偶然捕まえていた `cp`・`mv`・`sed -i` の書き込みも外れた。plan と PR 本文はそれを書いていない。最低限の対応は plan の 2 行と PR 本文の修正、tech-debt の 1 行で、コードは変えなくてよい。
- Follow-ups: M-1 は plan の 50 行目と 136 行目、PR 本文、tech-debt。L-1 は正規表現 2 か所とテスト 2 例(root と template)。L-2 はコメント 1 行、plan の Non-goals、tech-debt の 127 行目。L-3 は `verify.local.sh` の表示の分岐。L-4 はコメント 1 文(root と template)。L-5 は `/sync-docs` の register の更新。L-1・L-2・L-3・L-4 はスクリプトと hook の変更なので、直したら `/self-review` から回し直す。M-1 と L-5 を文書だけで直す場合は、`/sync-docs` の範囲で済む。

## Cycle 2

- Date: 2026-10-07
- Reviewer: reviewer subagent (Claude Opus 5.5)、cycle 2
- Scope: 811e1452..HEAD(0db1a97e の 1 コミット、3 ファイル、+42/-10)を中心に読んだ。cycle 1 の指摘を直した 989886f2・c45730cf・4575949f は cycle 1 の self-review より後のコミットで、どの self-review も読んでいなかったので、そのコード部分(`scripts/verify.local.sh`、`post_edit_verify.sh`、guard のコメント)も読んだ。テストと linter は実行していない。guard は 3 つの版(origin/main、811e1452、HEAD)と、HEAD を 1 か所だけ変えたコピー 5 つを scratchpad に置き、同じ payload を jq あり・なしの両方で渡して比べた。payload はどれも `jq -nc --arg` で組んだ。

### Evidence reviewed (cycle 2)

- 0db1a97e の差分を全行読んだ。root と `templates/base/` の `pre_bash_guard.sh` は `cmp` で一致した。
- 3 版の比較(28 例と 22 例)。cross-review の 2 件は直っている。
  - 改行のあとの `tee .env </dev/null` と `tee .git/config </dev/null` は、811e1452 では jq=ask・nojq=none、HEAD では両方 ask になった。`|` とタブのあとの `tee .env`、CRLF の行のあとの `tee .env` も、HEAD は両方 ask。
  - `tee /tmp/out < .env` と `tee /tmp/out < .git/config` は、811e1452 では両方 ask、HEAD では両方 none(origin/main も none)。`tee .env < input.txt` は 3 版とも ask。
- 新しいテストの行は、どれも直す前のコードで落ちる。H の 4 件と D の `$'echo x |\ttee .env'` は 811e1452 の jq なしの経路で none、C の `tee /tmp/out < ...` の 2 件は 811e1452 で ask。
- 0db1a97e は、test report の Test gaps 1〜3 にあたる 3 行も足している(`tests/test-pre-bash-guard.sh:225` の `cat >.env<<EOF`、`:219` の `/usr/bin/tee .git/x`、`:197` の `cat >/tmp/o</repo/.git/HEAD`)。HEAD に mutation G8・G9・G14 を 1 つずつ入れたコピーでは、この 3 行の判定がそれぞれ none、none、ask に変わる。3 つの mutation は、今はテストで見分けられる。
- cycle 1 の指摘の扱いは次のとおり(ID は cycle 1 の表のもの)。

| ID | cycle 2 の時点 | 根拠 |
| --- | --- | --- |
| M-1 | 文書で対応した。ただし同じ「旧 guard が捕まえていた形」の一覧に、C2-M1 と C2-L1 の漏れが見つかった | plan の 50 行目と 136 行目(c45730cf)、`docs/tech-debt/README.md:158` の (a) |
| L-1 | 一部を直し、残りは register にある | 0db1a97e で、`tee` の前のタブは jq なしの経路でも見るようになった。`>` のあとと `tee` のあとのタブは今も jq=ask・nojq=none(`echo x \| tee\t.env`、`tee -a\t.env` で確認)。コメント(`pre_bash_guard.sh:42-43`)と register 158 行目の (c) に書いてある |
| L-2 | 直した | `pre_bash_guard.sh:16-21`。register 127 行目に証拠と新しいきっかけがある |
| L-3 | 直した | `scripts/verify.local.sh:275-283`。追跡済みのファイルには `git update-index --chmod=+x`、未追跡には `git add --chmod=+x` を出し、git の work tree の外では git の行を出さない。working tree では +x で index だけ 100644 のときも `chmod +x` を出すが、何も変えないので害はない |
| L-4 | 直した。行の折り返しが途中で止まっている(C2-L3) | `.claude/hooks/post_edit_verify.sh:35-41`(root と template) |
| L-5 | 直した | register 157 行目は (a) を解消済みにし、127 行目はきっかけを書き直した(4575949f) |

### Findings (cycle 2)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-M1 MEDIUM | security | `tee` の前の区切りの集合にバッククォートと `\` がない。そのため `\tee .env`(alias を避けるときの書き方)と、バッククォートの中の `tee .env` は、jq あり・なしとも ask にならない(`.git/x` も同じ)。origin/main は `*"tee .env"*` と `*"tee .git"*` の部分一致で、どちらにも ask を返していた。この後退は cycle 1 の S1 で入り、cycle 1 の self-review と cross-review はどちらも見落とした。0db1a97e はちょうどこの選択肢を書き直し、新しいコメント(53〜54 行目)で区切りを「行頭、空白、`;` `&` `\|` `(` `/`、2 文字の `\n` と `\t`」と列挙したが、この 2 つは入っていない。plan の 136 行目(Risks)と register の 158 行目は、旧 guard より弱くなる形を M-1 の例だけと書いており、実測と合わない。cycle 1 の M-1 と同じ形の食い違いで、PR 本文がこの 2 か所を写すと、退行の範囲を実際より狭く伝える | 3 版の比較で、`\tee .env`、`\tee .git/x`、`` x=`tee .env </dev/null` ``、`` x=`tee .git/x </dev/null` `` の 4 例は、origin/main が jq あり・なしとも ask、811e1452 と HEAD は両方 none。`.claude/hooks/pre_bash_guard.sh:53-58`、plan の 136 行目、`docs/tech-debt/README.md:158` | 区切りの括弧に `` \` `` と `\\\\` を足す(`` (^\|[[:space:];&\|(/\`\\\\]\|\\\\[nt])tee ``)。二重引用符の中なので、バッククォートは `` \` `` とエスケープする。エスケープしないと guard 全体が構文エラーになる(scratchpad のコピーで確認)。このコピーは 4 例を両経路とも ask に戻し、ほかに判定が変わったのは `echo x \| \tee -a .env`(origin/main も none、ask に変わる)だけだった(50 例中)。テスト D に `'\tee .env'` と `` 'x=`tee .git/x`' `` を足し、53 行目のコメントの列挙も直す(root と template)。直さない場合は、plan の 136 行目、register の 158 行目、PR 本文に「`\tee` とバッククォートの中の `tee` は、旧 guard では ask、今は通る」と書く |
| C2-L1 LOW | security | 0db1a97e で `tee` の引数の読み取りが `<` と `>` で止まるようになり、リダイレクトより後ろにある `tee` の書き込み先を見なくなった。orchestrator は `tee out.txt 2>/dev/null .env` を「origin/main も ask を返さなかった」として受け入れたが、この前提は、後ろにもう 1 つリダイレクトが続く形では成り立たない。`tee out.txt 2>/dev/null .env > /dev/null`、`tee < in.txt .env > /dev/null`、`tee out.txt 2>/dev/null .git/x > /dev/null` は、origin/main(`*".env"*">"*` と `*".git/"*">"*` が偶然に当たる)と 811e1452 が ask、HEAD は jq あり・なしとも none。M-1 と同じ「旧 guard が後ろのリダイレクトで偶然捕まえていた形」に、`cp`・`mv`・`sed -i` 以外の例が加わった。新しいコメント(55〜57 行目)が書いているのは `<` で止める理由と例だけで、`>` で止める理由も、その結果リダイレクトの後ろの書き込み先を見ないことも書いていない | 3 版の比較。`>` を除外から外したコピー(`[^;&\|)<\\\\]*`)では、`2>/dev/null` を挟む 3 例が両経路で ask に戻り、`tee /tmp/out < .env` などの none は変わらなかった。`<` を挟む `tee < in.txt .env > /dev/null` は none のまま。代わりに ask になったのは `echo x \| tee log > "x .env"`(空白を含む引用符付きのリダイレクト先。811e1452 も ask)の 1 例だけ。`.claude/hooks/pre_bash_guard.sh:55-58` | どちらかに決めて書く。(1) `>` を除外から外す(1 文字、root と template)。`2>/dev/null` を挟む形が戻り、損は上の作為的な 1 例だけ。(2) 今のままにするなら、55〜57 行目に「リダイレクトより後ろの `tee` の引数は見ない(`tee out 2>/dev/null .env`)」と書き、register 158 行目の (a) と plan の 136 行目に、後ろに `>` が続く形は旧 guard が ask していたことを足す |
| C2-L2 LOW | maintainability | register の 2 行が、0db1a97e のあと事実と合わない。160 行目の (a) は、`cat >.env<<EOF`(G8)、`/usr/bin/tee -a .git/x`(G9)、`cat >/tmp/out.txt</repo/.git/HEAD`(G14)の 3 つがテストにないと書くが、0db1a97e が 3 つとも足した(G9 の行は `-a` のない形だが、mutation は見分ける)。158 行目は返済のきっかけを「The next change to `pre_bash_guard.sh`」としていて、0db1a97e がそれに当たる。同じ行の (c) は「`redirect_lead` と `tee_lead` に 2 文字の `\t` を受け付けさせれば単独で直る」と書くが、`tee_lead` は `tee` の前の `\t` だけを受け付けるようになったので、(c) が半分済んだように読める。0db1a97e のコミットメッセージも、Test gaps 1〜3 を足したことに触れていない | `docs/tech-debt/README.md:158`、`:160`、`tests/test-pre-bash-guard.sh:197`、`:219`、`:225`、G8・G9・G14 を入れたコピーでの判定 | `/sync-docs` で、160 行目の (a) を解消済みにする(0db1a97e、どの行がどの mutation を見分けるかを添える)。158 行目は、(c) を「`>` のあと、`tee` のあと、`tee` の引数のあいだのタブ」に絞り、きっかけを書き直し、C2-M1 と C2-L1 の形を (a) に足す |
| C2-L3 LOW | readability | L-4 の修正(989886f2)で書き直したコメントのうち、38 行目 `# Without jq, an apply_patch payload falls` が文の途中で短く切れ、39 行目からは元の折り返しのまま残っている。動作には関係しない | `.claude/hooks/post_edit_verify.sh:35-41`、`templates/base/.claude/hooks/post_edit_verify.sh:35-41`(両者は `cmp` で一致) | 38〜41 行目を折り返し直す(root と template)。急がない |

### Positive notes (cycle 2)

- 2 件の修正は、どちらも正規表現の 1 か所ずつに収まっている。root と template は byte 単位で一致している。
- 新しいテストの行は、どれも直す前のコード(811e1452)で落ちる。H の 4 件は jq なしの経路で、C の 2 件は両経路で、修正の前後の判定が分かれる。
- テストの見出しコメント(C、D、H の列挙)を、足した行に合わせて書き直している。
- 37〜43 行目のコメントは、jq なしの経路での `\n` と `\t` の扱いを、修正後の挙動どおりに説明している(`tee` の前は見る、`>` と `tee` のあとのタブは見ない)。

### Known gaps(orchestrator が受け入れたものを確かめた結果)

- `tee out.txt 2>/dev/null .env` は ask にならない。origin/main も jq あり・なしとも none だったことを確かめた。ただし後ろに `>` が続く形は origin/main が ask していた(C2-L1)。
- `printf "a\\ntee .env"` は ask になる。origin/main が ask だったのは jq の経路だけで、jq なしの経路では none だった(旧 sed fallback が最初の `\"` で取り出しを止めていたため)。HEAD は両経路で ask なので、jq のない環境ではこの形の誤検知が増えた。害は確認が 1 回出ることだけ。
- `echo x > /tmp/tee .env`(書き込み先は `/tmp/tee`)は 3 版とも ask になる。`/` を区切りに入れたための誤検知で、origin/main も同じ判定なので退行ではない。
- triage report の `pre_bash_guard.sh:51-52`、`:52` は 811e1452 の行番号で、HEAD では `tee_lead` は 58 行目にある。

### Tech debt identified (cycle 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (158 行目の更新)C2-M1 と C2-L1 を直さない場合、`\tee` とバッククォートの中の `tee`、リダイレクトの後ろの `tee` の書き込み先を (a) に足し、「except the five M-1 examples」を実測に合わせる。(c) を「`>` のあと、`tee` のあと、`tee` の引数のあいだのタブ」に絞る | bypass 以外のモードで、これらの形の `.git`・`.env` への書き込みが確認なしで走る | cap 2 の最後の cycle で、直すと全工程の回し直しが要る | `pre_bash_guard.sh` の `tee_lead` に次に触れるとき、または guard を通り抜けた書き込みの報告 | このレポート(C2-M1、C2-L1)、docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md |
| (160 行目の更新)(a) は 0db1a97e で解消。(b)〜(d) は残る | (a) が開いたままだと、register が実際より多くのテストの穴を示す | — | `/sync-docs`(この cycle) | このレポート(C2-L2)、docs/reports/test-2026-10-07-guard-bypass-and-hygiene.md(Test gaps 1〜3) |

_(orchestrator の指示で、このコミットにはレポートと insight event だけを入れる。上の 2 行は `/sync-docs` で `docs/tech-debt/README.md` に反映すること。cap 2 の最後の cycle なので、ここで直さない指摘は register に入れないと PR のあとに残らない。)_

### Recommendation (cycle 2)

- Merge: 可(条件付き)。CRITICAL と HIGH はない。cross-review の 2 件は直っていて、新しいテストの行は直す前のコードで落ちる。条件は C2-M1 の扱いで、区切りの括弧を直す(1 か所、root と template、テスト 2 行)か、plan の 136 行目、register の 158 行目、PR 本文に、旧 guard より弱くなった形として書く。
- Follow-ups: C2-M1 と C2-L1 はコードを直すなら `/self-review` から回し直しになる(cap 2 に達しているので、cap を上げるか、文書で記録するかを orchestrator が選ぶ)。C2-L2 は `/sync-docs` で register の 158 行目と 160 行目を直す。C2-L3 は急がない。plan の Progress に cycle 2(cross-review の 2 件と 0db1a97e)の記録がないので、`/sync-docs` で足す。cycle 2 の `/cross-review` は、guard の位置を行番号ではなく `tee_lead`(`pre_bash_guard.sh`)のように名前で書くと、後のコミットで指す先がずれない。
