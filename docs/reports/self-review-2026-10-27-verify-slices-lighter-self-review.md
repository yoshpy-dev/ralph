# Self-review report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(D1〜D6、AC1〜AC7、Progress checklist の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 / 2。cross-review の ACTION_REQUIRED を直したあとの回し直し。cycle 1 は 2 回走った: 90f4ecca と d87d250a)
- Scope: `git diff 44c14b97...HEAD`(HEAD a9b63c8a、48 ファイル、+1252/-122、18 コミット)。先に直しのコミット dba1c20f(13 ファイル)を通読し、そのあとで cycle 1 の所見が解決したままかを確かめ、diff 全体をこの PR 自身が絞った 6 項目(naming、readability、null safety、exception handling、security、maintainability)で見直した。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、範囲外のファイル)は implementer が各スライスで確かめた前提で、目に入った分だけ載せる。テスト、静的検査、spec 適合、文書のずれは見ていない(`/verify`・`/test`・`/sync-docs` の範囲)。

## Previous findings

cycle 1 の 1 回目(MEDIUM 1、LOW 6)と 2 回目(LOW 3)の計 10 件は、cycle 2 でも解決したまま。直しの dba1c20f が戻したものはない。

| cycle 1 の所見 | 状態 | 根拠 |
| --- | --- | --- |
| 1 回目 MEDIUM: 手順 7 が self-check の中身を見ない | 解決のまま | `.claude/skills/implement/SKILL.md:42` の「a result for each of its five items and no item found and not fixed」が残り、dba1c20f はその文の後ろに再開の文を足しただけ |
| 1 回目 LOW 6 件(inline の順序、`no longer` の履歴語、Completion gate、5 項目の言い方、6 項目のテストの範囲、同名 helper の向き) | 解決のまま | dba1c20f の差分はこれらの文を触っていない。5 項目の正本(`model-routing.md:43-47`)と各所の語は変わらず、テストの定数も同じ |
| 2 回目 LOW: 秘密情報で止まったときと最後のスライスが orchestrator 側に届いていない | 解決のまま | 804d05f9 が手順 7 の delegated・inline と model-routing に足した。dba1c20f はその文を「再開として出し直す」に進めた(下の Findings の 1〜3 行目は、この進めた文への所見) |
| 2 回目 LOW: Progress checklist の `Review artifact created` が 2 行 | 解決のまま | 計画に該当行は 127 行目の 1 行だけ。「このコミット」は SHA(bc56cc4f、d87d250a、dba1c20f)に置き換わり、a9b63c8a の追記も同じ形 |
| 2 回目 LOW: 2 つのテストの `assert_joined_contains_exact` の guard の差 | 解決のまま | `tests/test-self-review-scope.sh:74-84` と `tests/test-slice-verification.sh:47-57` の両方に `[ ! -f ]` の分岐があり、本体が同じ |

cross-review の指摘(秘密情報で止まったスライスを出し直すと、新しい implementer が「Before any edit」で止まる)は、dba1c20f で直った。`.claude/agents/implementer.md:21` と `.codex/agents/implementer.toml:22-25` に再開の例外が入り、手順 7 と model-routing が出し直しを「再開」と呼んで handoff に未コミットの変更を名指しさせる。例外の条件は「再開である」と「範囲内の未コミットの変更を、そのスライス自身のものとして名指ししている」の両方なので、名指しのない範囲内の変更では今も止まる。

## Evidence reviewed

- 直しの dba1c20f と、cycle 1 の直し 804d05f9 を `git show` で通読した
- 写しの一致: `.claude/agents/implementer.md`、`.codex/agents/implementer.toml`、`.claude/skills/implement/SKILL.md` は `templates/base/` と `.agents/skills/` の同名と `cmp` で一致した。`model-routing.md` は KNOWN_DIFFS の箇所(org receipts の節)を除き、この PR の hunk が root と `templates/base/` で同一(`git diff … | grep '^[+-]'` の突き合わせ)
- `git diff 44c14b97...HEAD --check` は空
- 再開の文の置き場所を `grep -rn 'stopped without\|as a resume\|resumes a slice'` で洗った: implementer の定義 4 本(.md 2、.toml 2)、`implement/SKILL.md` の 4 写し(`.claude`、`.agents`、`templates/base/.claude`、`templates/base/.agents`)、model-routing の 2 本。「重なる変更では止まる」を言う場所は、`grep -rn -i 'overlap\|pre-existing\|unambiguous baseline\|in-flight'` で implementer の定義と `implement/SKILL.md` の手順 6・7 のほかになかった
- `git restore -- <file>` の挙動は、scratchpad の使い捨てリポジトリで確かめた(下の 1 行目)
- tech-debt の台帳: dba1c20f が閉じた行(手順 6 の「narrowest」の検査の穴)は、この PR の f2f0333c が足した行。閉じ方は台帳の既存の形(`<!-- RESOLVED … Row preserved for traceability. -->` の行と取り消し線の行、5 セル)で、パイプは 6 本。逆向き(この PR が invalidate したのに開いたままの行)は見つからなかった。hook の通知文の行(`run-verify.sh before claiming done`)は計画の Non-goals どおり開いたまま
- 計画: a9b63c8a が足したのは `## Progress checklist` の 3 行だけ(digest が読まない節)
- 確かめていない: テストと静的検査の実行(計画の記録では slice 138/138、scope 202/202、agent-models 47/47)、各テストの変異の再現

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | 手順 7 の「範囲を絞って落とすファイルは `git restore -- <file>` で戻す」は、止まり方によっては何も戻さず、新規ファイルではエラーになる。使い捨てリポジトリでの結果: tracked でステージなし → 戻る(rc 0)。新規の untracked ファイル → `error: pathspec 'new.txt' did not match any file(s) known to git`(rc 1)。tracked でステージ済み → rc 0 だが index から戻すので変更がそのまま残る(`M  tracked.txt`)。秘密情報で止まった implementer は「unstage it」するので、新規ファイルなら untracked になり、`git restore` が失敗して秘密情報がディスクに残る。検証の失敗で止まった implementer は unstage を求められておらず(「report the failure clearly」)、ステージ済みなら `git restore` が効かない。末尾の「either way, no secret stays in the worktree」は事実の言い方だが、2 つ目の選択(変更を別スライスにする)では、そのスライスがコミットするまで秘密情報が worktree に残る。残ると、セッション終了と圧縮前の hook が feature branch の未コミットの変更を `wip:` でコミットする(`git-commit-strategy.md:43-45`)ので、履歴に入りうる | `.claude/skills/implement/SKILL.md:42`(4 写し)。固める検査は `tests/test-slice-verification.sh:182-184`。秘密情報での stop は `.claude/agents/implementer.md:40`、`.codex/agents/implementer.toml:55-56` | 手順 7 の文を、追跡中のファイルは `git restore --staged --worktree -- <file>`、新規のファイルは削除、と書き分ける。「no secret stays in the worktree」は「次の dispatch の前に秘密情報が worktree に残らないようにする」の命令にする。4 写しとテストの needle を同じ文にそろえる |
| LOW | security | 秘密情報で止まるときの「report it」に、値を書かない指示がない。implementer の報告は「found and not fixed (what you found, …)」を求め、inline の手順は各項目の結果を計画の `## Progress checklist`(追跡されるファイル)に書くよう求める。「what」に秘密情報の値を入れると、orchestrator の会話と、コミットされる計画に値が入る。このリポジトリの他の契約は証拠を値でなく指す形に寄せている(`agent-messaging.md` の EVIDENCE-as-pointers) | `.claude/agents/implementer.md:40,48`、`.codex/agents/implementer.toml:55-56`、`.claude/skills/implement/SKILL.md:43`(inline)、`.claude/rules/ralph/model-routing.md:43-47`(report contract) | 秘密情報の項目だけ「ファイルと行を書き、値は書かない」と一言足す(implementer の定義 2 形式、inline の手順、model-routing の 1 文)。テストで固める文もそろえる |
| LOW | maintainability | 再開の handoff の中身(「止まったスライスの未コミットの変更をそのスライス自身のものとして名指しする」)は、手順 7、model-routing の report contract の段落、implementer の「Before any edit」にしか書かれていない。handoff の契約として読まれる 2 か所には載っていない。(1) model-routing の「Structured handoff must carry」の表は 6 欄で、再開の欄がない。implementer の「Required handoff fields」も同じ。(2) 手順 6 の「Before dispatching, commit any outstanding plan/bookkeeping edits (or confirm they do not overlap the slice's files in scope) so the implementer starts from an unambiguous baseline」は、再開では重なりがあって当然なのに例外を持たない。止まる規則(implementer 側)に例外を足したのに、同じ規則を向こう側から言う手順 6 に足していない | `.claude/skills/implement/SKILL.md:40`(手順 6 の末尾)、`.claude/rules/ralph/model-routing.md:29-38`(表)と `:51-53`(段落)、`.claude/agents/implementer.md:11-17` | 表に「Resume(止まったスライスを出し直すときだけ): 範囲内の未コミットの変更の名指し」の行を足し、手順 6 の括弧に「(a resume of a stopped slice excepted: step 7)」を足す。表に足せば、段落の文は手順 7 を指すだけにできる |

## Positive notes

- 直しは 4 つの implementer 定義、`implement/SKILL.md` の 4 写し、model-routing の 2 本に同時に届き、写しは `cmp` で一致している。cycle 1 の所見の型だった「例外が片方の定義にしか届かない」は起きていない
- 再開の例外は条件が二重(再開であること、変更を名指ししていること)で、名指しのない範囲内の変更では今も止まる。「他人の進行中の変更を吸収しない」という元の保護は残っている
- 検証の失敗で止まる場合を含め、例外を秘密情報に限らず「コミットせずに止まったスライスの再開」に広げた。implementer の定義(「Never commit failing state」)と手順 7 の stop の対象が合っている
- 再開に関する 5 つの文(implementer の定義の例外、手順 7 の stop の範囲、手順 7 の再開、手順 7 の片付け、model-routing の段落)をすべてテストが固め、手順 6 の「narrowest」を 4 写しとも変える変異が落ちる検査も同じコミットで入った。台帳の行の閉じ方は既存の形に従い、計画の Progress checklist に一緒に閉じたことが書いてある(a9b63c8a)
- 前回までの良い点が続いている: 5 項目の語が正本から同じ並びで届く、`/self-review` を絞った理由が履歴を前提にしない書き方で書かれている

## Tech debt identified

このパイプラインは cap(2 回)の最後の回なので、下の LOW 3 件を直さずに `/pr` へ進めば、計画が archive されたあとに残るのは台帳の行だけになる。依頼されたコミットの範囲はこの report と insight event だけなので、`docs/tech-debt/README.md` には足していない。直さずに進める場合は、次の 1 行を足す(1 行にまとめた)。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| 再開の手順の穴 3 つ(verify-slices-lighter-self-review の cycle 2 の self-review): (1) `/implement` 手順 7 の「`git restore -- <file>` で戻す」は新規(untracked)ファイルでエラー、ステージ済みなら何も戻さない。「either way, no secret stays in the worktree」は 2 つ目の選択では成り立たない。(2) 秘密情報で止まる指示に「値を報告や計画に書かない」がない。(3) handoff の表と手順 6 が再開を知らない | 秘密情報で止まった新規ファイルを落とすと、`git restore` が失敗してディスクに残り、セッション終了の `wip:` コミットで履歴に入りうる。値を報告に書けば、inline では計画に入る | cap の最後の回に見つかり、直すと手順 6・7、model-routing、implementer の定義 4 本、写し、2 つのテストの再実行になる。どれも止まったスライスの再開という稀な経路 | 次に `/implement` 手順 7 か implementer の定義を触るとき、または止まったスライスの再開を実際に行うとき: 3 点を直し、needle をそろえる | `docs/reports/self-review-2026-10-27-verify-slices-lighter-self-review.md`(Findings の 1〜3 行目)、`docs/reports/cross-review-triage-verify-slices-lighter-self-review.md` |

_(台帳には足していない。足す場合は上の 1 行を `docs/tech-debt/README.md` の末尾に追記する。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM なし。LOW 3 件で、どれも稀な再開の経路の文言)
- Follow-ups: cross-review の指摘そのもの(再開できない)は直っており、cycle 1 の 10 件も解決したまま。LOW 3 件は cap の最後の回の発見なので、直すなら cap を上げて全体を回し直す(または文書だけの直しとして扱えるか判断する)ことになり、直さないなら上の 1 行を台帳に足す。`.goreleaser.yml` の changelog は `docs:` と `test:` を落とし、この PR の 18 コミットはすべてその 2 種なので、downstream に届く skill と agent の変更は自動の changelog に Merge 行しか出ない。`/pr` の本文と release の issue に 1 行(「`/implement` がフルの検査を打たない」「`/self-review` が 6 項目になった」「止まったスライスは再開として出し直す」)を足す価値がある
