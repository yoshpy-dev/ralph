# Self-review report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Branch: fix/guard-bypass-and-hygiene(base 2a22ba78、HEAD ed4e4ae5)
- Reviewer: reviewer subagent (Claude)、cycle 1
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

- Merge: 可。CRITICAL と HIGH はない。M-1 は MEDIUM で、guard の判定を狭めた結果、旧 guard が偶然捕まえていた `cp`・`mv`・`sed -i` の書き込みも外れた。plan と PR 本文はそれを書いていない。最低限の対応は plan の 2 行と PR 本文の修正、tech-debt の 1 行で、コードは変えなくてよい。
- Follow-ups: M-1 は plan の 50 行目と 136 行目、PR 本文、tech-debt。L-1 は正規表現 2 か所とテスト 2 例(root と template)。L-2 はコメント 1 行、plan の Non-goals、tech-debt の 127 行目。L-3 は `verify.local.sh` の表示の分岐。L-4 はコメント 1 文(root と template)。L-5 は `/sync-docs` の register の更新。L-1・L-2・L-3・L-4 はスクリプトと hook の変更なので、直したら `/self-review` から回し直す。M-1 と L-5 を文書だけで直す場合は、`/sync-docs` の範囲で済む。
