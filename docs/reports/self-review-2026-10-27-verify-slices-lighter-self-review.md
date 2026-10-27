# Self-review report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(D1〜D6、AC1〜AC7、Progress checklist の記録)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 44c14b97...HEAD`(HEAD f6e6e536、39 ファイル、+621/-114、7 コミット)。見たのは diff の品質だけ。テスト、静的検査、spec 適合、文書のずれは見ていない(`/verify`・`/test`・`/sync-docs` の範囲)。この PR 自身が絞った 6 項目(naming、readability、null safety、exception handling、security、maintainability)で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、範囲外のファイル)は implementer が各スライスで確かめた前提で、目に入った分だけ載せる。

## Evidence reviewed

- 依頼された 3 点
  - root と写しの文言の一致: root の 13 ファイルを `templates/base/` の同名と `diff` で比べた。`model-routing.md` を除く 12 ファイルは完全に一致した。`model-routing.md` は KNOWN_DIFFS の箇所(org receipts の節、末尾の Where the values live)だけが違い、この PR の 2 つの hunk(`git diff` の追加行と削除行)は root と `templates/base/` で同一だった。`.agents/skills/{implement,plan,self-review}` の `SKILL.md` と `template.md` は `.claude/skills/` と同一で、`self-review/SKILL.md` の違いは `allowed-tools` の 1 行だけ(`.agents` 側の生成の既存の挙動)。`templates/base/.agents/` は `.agents/` と同一
  - `/implement` の手順 6、7、12 と Completion gate の文が、orchestrator に曖昧なく読めるか: 全文を読んだ。下の表の 1、2、4 行目
  - 2 つの shell テストの読みやすさと、落ちる理由: `block`/`join_lines`/`assert_block_contains` の動き(awk の前方一致、`tr -s ' '`、`-F` の固定文字列、start が見つからないときは空の block で「missing」と出る)を追った。下の表の最後の 2 行
- 事実の照合: 雛形の「`RALPH_VERIFY_SCOPE=changed` は pack を選ぶだけで、Go の中では `go test ./...` を走らせる」は `packs/languages/golang/verify.sh:91,115` に合う。「`/verify` の branch secret scan」は `scripts/run-verify.sh:240-257` にある。`gofmt -l` が一覧しても 0 を返すことは既知の挙動
- 前提の確認: `.codex/agents/*.toml` に足した文に、`"""` や `\` はない(TOML の複数行文字列を壊さない)。`f6e6e536` が変えたのは計画の `## Progress checklist` だけ(digest が読まない節)
- 旧 10 項目の名前の残り: `grep -i "debug code|unnecessary changes"` は、この PR が足した「外した」旨の文とテストの needle だけに当たる。skill・agent・rules に 10 項目のまま残る一覧はない。`internal/org/prompts/` に `/self-review` の観点に頼る文はない
- 確かめていない: テストと静的検査の実行、各テストの変異の再現

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | 手順 7 の委譲スライスの確認は「mechanical self-check result present」で、中身を見ない。implementer は「直すか、報告する(`or report it`)」でよく、報告の欄は「clean か、見つけて直した内容」しか書けない。直らなかった指摘(範囲外の秘密情報、消せないデバッグ行)を載せる欄も、載っていたときに orchestrator がどうするかの文もない。`/self-review` から外れた 4 項目を見る場所はここだけになった | `.claude/skills/implement/SKILL.md:42`(`mechanical self-check result present`)、`.claude/agents/implementer.md:40`(`or report it`)と `:48`(`checked and clean, or what was found and how it was fixed`)、`.codex/agents/implementer.toml:45-46,55-56` | 手順 7 を「present and clean (or every finding fixed)。直らなかった指摘があれば、検証の失敗と同じに扱って次へ進まない」のように中身まで言う。implementer の報告の欄には「見つけて直さなかったもの」を足し、`or report it` はその欄に出すと読めるようにする |
| LOW | readability | 手順 6 の「検証コマンドのない計画」の文は `before dispatching` で終わるが、手順 7 の inline のスライスは「計画のコマンド(step 6)」とここへ戻す。inline では dispatch がないので、コマンドをいつ書くかが読み取りにくい。また inline の自己確認は「検証 → 自己確認 → そのあと stage」の順で、implementer が使う `git diff --cached` は stage 後でないと使えない。inline 側には何を読むか(`git diff`)も、直したら検証を打ち直すこと(implementer にはある)も書いていない | `.claude/skills/implement/SKILL.md:40`(`write it under ... before dispatching`)、`:43`(`then do the same mechanical self-check ... Then stage the slice's files`)、`.claude/agents/implementer.md:34-41` | 手順 6 は `before dispatching (or before editing, for an inline slice)` に。手順 7 の inline は「stage してから `git diff --cached` を読む。直したら検証を打ち直して stage し直す」と implementer と同じ順にする |
| LOW | readability | implementer の定義に「`/self-review` no longer checks the first four items」とある。「no longer」は変更の履歴を前提にした書き方で、この定義は `ralph init` で新しい project にも入る(その project に以前の挙動はない)。「first four」は箇条書きの位置に頼るので、順番を変えると意味が変わる | `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:46`(`templates/base/` の写しも同じ) | `/self-review` does not check unnecessary changes, typos, debug leftovers, or secrets, so do not skip them. のように、項目の名前で書く |
| LOW | maintainability | `## Completion gate` の「フルの検査を通した」項目が、「各スライスの検証が通った」に置き換わった。`/verify` と `/test` の verdict を gate に入れる項目はない(括弧の中は説明だけ)。末尾の「If verification has not run, say so explicitly」は、どの検証を指すのかが 2 通りに読める(スライスの検証か、手順 13 のフルの検査か) | `.claude/skills/implement/SKILL.md:84`(Completion gate の 1 行目)、`:89` | 括弧を「…(and `/verify` and `/test` in step 13 passed before you present the task as complete)」のように gate の条件にするか、末尾の文がどちらを指すかを書く。PR の pre-check が verdict を読む(R2)ので、後者だけでも足りる |
| LOW | maintainability | 「機械的な 5 項目」が、少しずつ違う言い方で複数の場所にある。implementer の箇条書き(「typos or copy-paste errors」)、model-routing の括弧(「typos」)、手順 7 の inline の括弧(「files outside the slice」)、reviewer の定義(4 項目で「typos and copy-paste errors」)、`self-review/SKILL.md`(5 項目の並び)。正本がどこかを書いた文はない(手順 7 の inline が「the implementer does」と指すだけ)。語がずれても、どのテストも落ちない | `.claude/rules/ralph/model-routing.md`(report contract の括弧)、`.claude/skills/implement/SKILL.md:43`、`.claude/agents/reviewer.md:13`、`.claude/skills/self-review/SKILL.md:23-28` | implementer の定義を正本と決め、ほかは「the implementer's mechanical self-check」と指すか、項目名の語を 1 つにそろえる(例: 「unnecessary changes / typos / debug leftovers / secrets / files outside the handoff」) |
| LOW | maintainability | 6 項目のテストが守る範囲が狭い。`tests/test-self-review-scope.sh` が旧 4 項目の復活を見るのは `SKILL.md` の 4 つだけで、reviewer の定義(`.md`、`.toml`)と `template.md` の Area の値には掛からない。ここに `debug-code` や「debug code」が戻っても落ちない。同じ関数が旧項目の見出しを大文字小文字を区別して照合するので、`**unnecessary changes**` と書き直されても通る。また、手順 7 が要る implementer の報告の欄(`mechanical self-check result`)は、見出し(`Mechanical self-check before commit`)しか固めていない。手順 7 の「mechanical self-check result present」も固めていない。欄が implementer の定義から消えても、手順 7 だけが残っても落ちない | `tests/test-self-review-scope.sh:30,41,71-101`、`tests/test-slice-verification.sh`(implementer と手順 7 の委譲側の検査なし)、計画の Test 節(「implementer の定義の機械的な確かめの文を固める」) | reviewer の 4 ファイルに旧 4 項目の `assert_not_contains` を足し、implementer の 4 ファイルに報告の欄の文(`mechanical self-check result`)を足し、手順 7 の委譲側の文を `assert_block_contains` で固める |
| LOW | readability | 2 つのテストで同じ名前の `assert_contains`/`assert_not_contains` が、大文字小文字の扱いが逆になっている(`test-self-review-scope.sh` は contains が `-Fi`、not_contains が `-F`。`test-slice-verification.sh` は contains が `-F`、not_contains が `-Fi`)。どちらもコメントに書いてあるが、片方から関数を写すと挙動が変わる | `tests/test-self-review-scope.sh:30,41`、`tests/test-slice-verification.sh:47,60`(説明は `:10-11`) | 新しいテストの関数名を `assert_contains_exact`/`assert_absent_any_case` のように挙動で区別するか、この 2 つを同じ向きにそろえる |

## Positive notes

- root、`templates/base/`、`.agents/` の写しの本文が、`model-routing.md` の KNOWN_DIFFS 以外で一致している。この PR の hunk も、`model-routing.md` を含めて root と `templates/base/` で同一
- 雛形の `test -z "$(gofmt -l …)"` に理由(`gofmt -l` は一覧しても 0)が括弧で付き、`RALPH_VERIFY_SCOPE=changed` の誤解も Go の `go test ./...` を根拠に書いている。いずれもリポジトリの内容と合う
- `/self-review` を絞った理由(外した 4 項目の行き先、秘密情報の別の確かめ)が skill・reviewer・report の雛形に同じ向きで書かれ、スクリプト名は書かれていない(D5 の守り)
- `test-slice-verification.sh` は `block` で「手順 12 の中」「Completion gate の中」に範囲を限り、ファイルのどこかに文があるだけの偽の合格を避けている。needle が欠けたときは何が足りないかとファイルがメッセージに出る
- 手順 6 の「計画に検証コマンドがない場合」は、`## Progress checklist` が digest を読まないことを理由に、再承認を求めない旨が書かれている。drift の節の既存の文と矛盾しない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

なし(サイクル 1。LOW は修正か triage に回す)。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件と LOW 6 件で、どれも文言とテストの足しで直せる。ブロックしない)
- Follow-ups: 表の 1 行目(MEDIUM)の手順 7 の確認の文と implementer の報告の欄は、`/verify` に進む前に直す価値がある。LOW のうち、手順 6・7 の inline の順序、「no longer」、Completion gate の括弧は 1 コミットで直せる。テストの穴(表の下から 2 行目)は、直した文を固めるときに合わせて足す
