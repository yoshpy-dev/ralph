# Self-review report: subagent-model-defaults

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-subagent-model-defaults.md
- Branch: chore/subagent-model-defaults(HEAD 1f4bb094、実装のコミットは 173cf76d)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(命名、読みやすさ、不要な変更、誤字、null 安全、デバッグコード、secret、例外処理、セキュリティ、保守性)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff 11602fed...HEAD` から plan の記録コミット(90dce49a、0916e3ab、1f4bb094)を除いた 173cf76d の 11 ファイル。テストと linter は実行していない(スコープ外)

## Evidence reviewed

- `git diff 11602fed...HEAD -- .claude templates/base/.claude` の全 hunk。agent の frontmatter は 8 ファイルとも `model:` の 1 行だけの変更で、root と template の blob が同一(`caf0c26f..4d6e7ccc`、`9ebb16dd..6735b634`、`d3c27ff2..1d88a1ad`、`90819b93..0050fe20`)。`doc-maintainer.md` は未変更
- `model-routing.md` の root と template。変更した 3 hunk(tier 表、pin の記述、escalation の段落)は両コピーで同じ文面。`diff` で残る差は org runtime の節 2 か所と「Where the values live」の既存の差だけ
- `tests/test-agent-models.sh` の全文(554 行)。モードは `100755`、shebang は `#!/usr/bin/env sh` で、隣の `tests/test-agent-phase-boundaries.sh:1` と同じ。POSIX の観点(`local`、`[[`、配列、`$'..'` の不使用、`case` の `[!...]`、`set --` の関数内スコープ、`if cmd; then ... else _rc=$?`)は読んで確かめた
- 一時ディレクトリ: `mktemp -d "${TMPDIR:-/tmp}/test-agent-models.XXXXXX"`(`:49`)の直後に `trap 'rm -rf "$TMP_ROOT"' EXIT`(`:50`)と `trap 'exit 1' HUP INT TERM`(`:51`)。シグナルでも EXIT trap が走る。fixture はすべて `$TMP_ROOT/case-N` 配下で、実ツリーへの書き込み経路はない
- `git grep -nE 'Judgment seats|Procedural seats|Escalating a judgment|judgment-heavy'`(`docs/plans/archive`、`docs/reports`、この plan を除く): 該当なし。今回消した見出しと段落名への参照は残っていない
- `.claude/agents/*.md` の本文に model 名を書いた箇所はない(`model:` 行のみ)
- `.goreleaser.yml:33-39`(changelog の除外)、`scripts/check-template-purity.sh:44-80`(FIXED_PATTERNS、REGEX_PATTERNS)、`templates/base/tests` が存在しないこと(この test は scaffold に出ない)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| F-1 MEDIUM | maintainability | 変更日と経緯を書いた段落が、生きた rules 文書に入り、template 経由で全 scaffold に出る。「Seat defaults changed on 2026-10-04 (maintainer decision): `implementer`, `verifier`, and `tester` moved from `sonnet` to `opus`; `reviewer` moved from `opus` to `sonnet`.」は、`ralph init` で新規に作った利用先には「何から変わったのか」が通じない。「maintainer」も利用先では別人を指す。次に割り振りを変えたとき、この段落は直されないか、変更履歴が積み上がる。model 名が入る箇所だが `tests/test-agent-models.sh` は読まない(pin の記述と tier 表の行だけを見る)。`check-template-purity.sh` の reason 文は「template content must not cite meta-repo build history」と言うが、パターンは固定文字列(`overlay-scaffold` など)と日付つきの `docs/(reports\|plans)/` だけで、この文は通る | `.claude/rules/ralph/model-routing.md:15-17`、`templates/base/.claude/rules/ralph/model-routing.md:15-17`、`scripts/check-template-purity.sh:44-69` | 2 つのコピーから段落を削る。履歴は commit、plan、release note にある。残すなら日付と「maintainer decision」を外し、現在の割り振りだけを述べる(それは表がすでに述べている) |
| F-2 MEDIUM | maintainability | `check_side` が 111 行あり、4 つの仕事を 1 つの関数に持つ。入力の存在確認(`:116-123`)、tier 表の行から `<agent> <model>` の対応を組む(`:125-153`)、agent ごとの照合(`:155-190`)、pin の記述の照合(`:192-213`)。`_line`、`_name`、`_f`、`_bad`、`_rows` など `_` 始まりの大域変数を `check_cross` と `emit_results` と共有している。衝突しないのは `run_checker`(`:255-262`)が `$(...)` の subshell で呼ぶからで、その前提は `run_checker` の 1 行コメントにしかない。規約(1 関数 50 行未満)の 2 倍。今回の変更範囲の中でいちばん読みにくい箇所 | `tests/test-agent-models.sh:106-216`、`:100-103`(checker の説明にこの前提がない) | `build_assignments`(対応を出力)、`check_agent_models`、`check_pin_sentence` に分ける。分けないなら、`:100-103` に「checker は必ず `run_checker` 経由で呼ぶ(状態を subshell に閉じ込めている)」を足す |
| F-3 LOW | readability | tier 表の列見出しは `Examples` のまま、その列が agent と model の対応そのものになった。test は同じ列から backtick の語を拾って対応表として読む(`:74`、`:88-93`、冒頭の「the table is the declaration」`:12-14`)。見出し「Implementation and verification seats」は、行に入っている最後の 2 例(design trade-offs、ambiguous root-cause debugging)を表さない。この表を機械が読むこと、守っている test の場所も文書のどこにも書かれていない。次の編集者は列を例示として書き換えて、test が落ちて初めて契約に気づく | `.claude/rules/ralph/model-routing.md:8,11`、`:131-137`(「Where the values live」に新しい test がない) | 列見出しを `Seats / work` などに改める。root の「Where the values live」(既知の差の hunk なので root だけ足せる)に `tests/test-agent-models.sh` を 1 行足す。後者は /sync-docs の担当 |
| F-4 LOW | maintainability | ヘッダのコメントは table、pin の記述、frontmatter が「lock-step」と言うが、実際の限界が書かれていない。(1) pin の記述の照合は implementer 固定(`:167-169`、`:198-208`)。ファイル全体を `tr \| grep -oE` で走査するので(`:194`)、別の agent を同じ形で書く文が 1 つ増えると、implementer の model と比べて「implementer が違う」と誤った名指しで落ちる。(2) 表が agent 名を拾うのは `agents/` にファイルがある名前だけ(`:144` の `\|\| continue`)。agent ファイルを消して表の行を残しても、どちらの照合も通る。表にない agent は落ちる(`:171-173`)ので、検出が非対称。(2)は plan の edge case(汎用の語を agent 名と誤認しない)から来た意図的な選択で、欠陥ではないが、どこにも書かれていない | `tests/test-agent-models.sh:2-6,144,167-169,194,198-208,171-173` | ヘッダに限界を 2 行足す(pin は implementer 専用、表の古い行は検出しない)。(1)は pin の正規表現を `implementer` を含む文に絞ってもよい |
| F-5 LOW | maintainability | self-test が踏まない分岐と、紛らわしい PASS がある。(i) pin の記述を丸ごと消す分岐(`:195-197`)。mutation (d) は値を戻すだけで、文を書き換えて消す編集のほうが起きやすい。(ii) template だけが余分な agent を持つ分岐(`:243-250`)。`self_test_cross`(`:518-535`)は template から消す側だけ。(iii) `agents` ディレクトリや routing 文書がない分岐(`:116-123`)と agent 0 件の分岐(`:187-190`)。(iv) `check_cross` は両側とも `model:` がないとき `PASS: agent X: root and template frontmatter model both <none>` を出す(`:237`)。`check_side` が別に落とすので結果は赤だが、PASS 行が 1 つ数えられ、読む人には「両方 none で一致」が成功に見える | `tests/test-agent-models.sh:195-197,243-250,116-123,187-190,237` | (i)と(ii)に mutation を 1 つずつ足す。(iv)は両側が空のとき PASS を出さずに `continue` する(`check_side` の FAIL に任せる) |
| F-6 LOW | naming | 「root」が 2 つの意味で使われている。tree の根(`check_side` / `check_cross` の `_root`、`:107`、`:221`)と、meta-repo 側の `.claude`(`ROOT_SIDE`、`:26`、`:222`)。`$_root/$ROOT_SIDE/agents`(`:222`)は読み違えやすい。また `TARGET_AGENT="tester"`(`:28`)と `doc-maintainer.md`(`:531`)が理由なしで固定されている。`doc-maintainer` を改名すると `mut_remove_file` は存在しないパスを消し、`cmp -s` は 2 を返すので `mutate` の「変化なし」ガードをすり抜け(`:423`)、`check_cross` は通って「checker passed but should have failed」と出る。改名が原因だと分かる出力ではない | `tests/test-agent-models.sh:26-28,107,221-222,423,531` | `_root` を `_tree` にする。`TARGET_AGENT` に 1 行のコメント(どの agent でもよい)を足す。`mut_remove_file` の後に、元のファイルが存在したかを `mutate` で確かめる |
| F-7 LOW | maintainability | 実装コミットの prefix が `chore:` で、`.goreleaser.yml:36-39` が `^chore:` を changelog から外す。この変更は利用先の費用に効く(3 席が sonnet から opus に、reviewer が opus から sonnet に)。plan の Rollout は「次のリリース後 `ralph upgrade` で届く」と書くだけで、利用者に知らせる経路がない。release note は手書きで、PR の記述を release issue に結びつける仕組みがない | `git log -1 173cf76d`、`.goreleaser.yml:36-39`、plan の Rollout | `/pr` の記述に加えて、手動の release issue に「agent の既定 model が変わる(implementer / verifier / tester は opus、reviewer は sonnet)。手で `model:` を変えた利用先は `ralph upgrade` で drift として残る」の 1 行コメントを足す |

## Positive notes

- agent の frontmatter の変更は `model:` の 1 行だけで、root と template が同じ。`doc-maintainer.md` は plan どおり触っていない。不要な差分、debug code、secret、TODO はない
- 一時ディレクトリの扱いが正しい。`mktemp` はテンプレートを明示して `TMPDIR` に従い、EXIT trap と HUP / INT / TERM の trap で掃除する。fixture は実ツリーを書き換えない。`mutate` は変化がなかった mutation を FAIL にするので、古くなった mutation が検出として数えられない(`:418-427`)
- 失敗側に倒れる作りが一貫している。`model:` がない agent は落ちる(省略は `inherit` なので規則どおり)。`## Tier table` の見出しを改名すると「no row with a backticked model」で落ちる。checker が PASS / FAIL 以外の行を出すと FAIL になる(`:278`)。終了コードが非 0 で FAIL 行がないときも FAIL になる(`:283-285`)。落ちるときは file と agent 名を出す
- real tree に FAIL があるときの self-test の SKIP(plan の逸脱 1)は妥当と判断した。SKIP の行は件数に入らず、終了コードは real tree の FAIL で決まる。壊れた木を複製すると同じ原因の失敗が mutation ごとに重複して出るだけで、検出力は落ちない
- model-routing.md の escalation の段落は、新しい既定と矛盾しない。「implementer を opus に上げる」とは書いておらず、例は reviewer への `opus`、implementer への `sonnet` になっている。pin の記述も `model: opus` で frontmatter と一致する
- `tests/test-agent-models.sh` は `templates/base/tests` が存在しないので scaffold に出ない。`templates/base/.claude` を参照するのは meta-repo 専用の test として正しい

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この review では登録しない) | - | 7 件はすべて in-cycle fix の推奨。F-1 だけは 2 ファイルの 3 行削除で済む | - | - |

_(orchestrator が F-2 以降を直さずに進める場合は、`docs/tech-debt/README.md` に 1 行にまとめて足すこと。cap は 2 回なので、次の cycle が最後になる。)_

## Recommendation

- Merge: 可。CRITICAL と HIGH はない。MEDIUM 2 件は blocking ではない
- Follow-ups: 直すなら F-1(2 ファイルの 3 行削除)と F-3 の見出し変更を 1 つの fix commit にまとめる。fix すると pipeline が最初から再実行されるので(cap 2 回)、その費用を払わないなら F-2〜F-6 を 1 行の tech-debt にまとめて先へ進める。F-7 は `/pr` の時点で release issue へのコメントとして出せる
- Known gaps: test を実行していないので、POSIX 準拠は読んだ範囲の判断。実際の `/bin/sh`(dash)での挙動は /test の領分
