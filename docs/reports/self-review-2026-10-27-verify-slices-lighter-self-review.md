# Self-review report: verify-slices-lighter-self-review

- Date: 2026-10-27
- Plan: `docs/plans/active/2026-10-27-verify-slices-lighter-self-review.md`(D1〜D6、AC1〜AC7、Progress checklist の記録)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 2 / 2 の 2 回目の run。cycle 2 の 1 回目の run(2478fd5d)は LOW 3 件を出し、直しのコミット d33bb358 が入った。cycle 1 は 2 回走った: 90f4ecca と d87d250a)
- Scope: `git diff 44c14b97...HEAD`(HEAD d33bb358、48 ファイル、+1270/-122、20 コミット)。先に直しのコミット d33bb358(12 ファイル)を通読し、cycle 2 の 1 回目の LOW 3 件が解決したかを確かめ、cycle 1 の 10 件が解決したままかを見て、diff 全体をこの PR 自身が絞った 6 項目(naming、readability、null safety、exception handling、security、maintainability)で見直した。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、範囲外のファイル)は implementer が各スライスで確かめた前提で、目に入った分だけ載せる。テスト、静的検査、spec 適合、文書のずれは見ていない(`/verify`・`/test`・`/sync-docs` の範囲)。

## Previous findings

cycle 2 の 1 回目の LOW 3 件は、d33bb358 で 3 件とも解決した。cycle 1 の 10 件は解決したままで、d33bb358 が戻したものはない。

| 前回の所見 | 状態 | 根拠 |
| --- | --- | --- |
| 1 回目 LOW 1: 手順 7 の `git restore -- <file>` は、止まり方によっては何も戻さず、新規ファイルではエラーになる。「either way, no secret stays in the worktree」は 2 つ目の選択で成り立たない | 解決 | 手順 7(`.claude/skills/implement/SKILL.md:42`)は、先頭に命令「First take a secret out of the worktree (edit the value out, or delete a new untracked file that holds it)」を置き、成り立たなかった「either way, …」の文を削った。範囲を絞って落とすファイルは `git restore --staged --worktree -- <file>`(追跡中のファイル)か削除(untracked)と書き分けた。使い捨てリポジトリ(git 2.49.0)で 4 つの状態を試した: 追跡中でステージ済み → rc 0、`base` に戻る。追跡中でステージなし → rc 0、戻る。新規でステージ済み(検証の失敗で止まった implementer が残す形)→ rc 0、索引と worktree の両方から消える。untracked → `error: pathspec … did not match`(rc 1)。untracked だけがエラーで、文はそこを「delete」に分けている。先頭の命令の理由に挙げた「`wip:` のチェックポイントは `git add -A` で全部を stage する」は、`.claude/hooks/session_end_summary.sh:43` と `.claude/hooks/precompact_checkpoint.sh:46` で確かめた。inline の手順にも「take it out of the worktree as for a delegated slice」が入った |
| 1 回目 LOW 2: 秘密情報で止まるときの報告に、値を書かない指示がない | 解決 | 「file and line, never its value」が 4 面に届いた: implementer の止まる規則と報告の契約(`.claude/agents/implementer.md:40,48`、`.codex/agents/implementer.toml:55-58,66-69`)、手順 7 の inline が計画の Progress checklist に書く文、model-routing の report contract の段落。root と `templates/base/` の `cmp` が一致し、テストが各文を固める(`tests/test-self-review-scope.sh` に報告の契約の needle を 1 本足し、`tests/test-slice-verification.sh` の needle を書き換えた) |
| 1 回目 LOW 3: 再開の handoff の中身が、手順 7・model-routing の段落・implementer の「Before any edit」にしか書かれず、handoff の表と手順 6 に再開の欄がない | 解決 | `model-routing.md:39` の表に「Resume (only when dispatching a stopped slice again)」の行が入った(`templates/base/` と hunk が同一)。手順 6 の末尾に「the uncommitted changes a resume hands over (step 7) are the one exception」が入った。どちらもテストが固める(表の行、手順 6 の末尾) |
| cycle 1 の 1 回目 MEDIUM 1 件と LOW 6 件、2 回目 LOW 3 件の計 10 件 | 解決のまま | 手順 7 の「a result for each of its five items and no item found and not fixed」は 1 か所に残る。Completion gate、inline の `git diff --cached` の順序、5 項目の正本(`model-routing.md:43-47`)は d33bb358 の差分の文脈に変わりなく出ている。scope テストと slice テストの `assert_joined_contains_exact` の `[ ! -f ]` の分岐も残る(`tests/test-self-review-scope.sh:74-84`)。d33bb358 の追加行に `no longer` などの履歴語はない |

implementer からの質問「implementer の定義の Required handoff fields に Resume の項目がない」は、所見にしない。`.claude/agents/implementer.md:11-17` の一覧は、欠けると implementer が止まる 6 項目(次の行が「If any required field is missing, stop immediately」)で、`subagent-policy.md:54` と手順 6 の handoff の項目の一覧も同じ 6 項目を挙げる。Resume は止まったスライスを出し直すときだけの欄なので、この一覧に入れると、ふつうの dispatch のたびに Resume 欠落で止まる。implementer 側の読み方は、同じ定義の「Before any edit」にすでに書いてある(「If the handoff says it resumes a slice … and names the uncommitted changes … as that slice's own, those changes are yours」)。表の行は「(only when …)」で任意と読める。

## Evidence reviewed

- 直しの d33bb358 を `git show` で通読した(12 ファイル、+50/-32)。root の implementer の定義 2 本、手順 7 の `implement/SKILL.md`、`model-routing.md`、テスト 2 本
- 写しの一致: `.claude/agents/implementer.md`、`.codex/agents/implementer.toml`、`.claude/skills/implement/SKILL.md` は `templates/base/` の同名と `cmp` で一致し、`.claude/skills/implement/SKILL.md` と `.agents/skills/implement/SKILL.md`、`.agents/skills/…` と `templates/base/.agents/skills/…` も一致した。`model-routing.md` の d33bb358 の hunk は root と `templates/base/` で同一(`git diff … | grep '^[+-]'` の突き合わせ)
- `git diff 44c14b97...HEAD --check` と `git diff d33bb358^ d33bb358 --check` は空
- 「値を書かない」の置き場所を `grep -rn 'never by its value\|by its file and line'` と手順 7 の `secret` 全行で洗った。秘密情報を報告・計画に書きうる経路(implementer の報告、inline の Progress checklist、model-routing の report contract)の全部に届いている
- `git restore` の挙動は、scratchpad の使い捨てリポジトリ(`/private/tmp/claude-502/-Users-hiroki-yoshioka-MyDev-github-com-yoshpy-dev-ralph/871f359e-5be2-4cd7-98c6-e94a56f90dbb/scratchpad/probe-restore`、git 2.49.0)で、追跡中・新規ステージ済み・untracked の 4 状態を試した(上の表の 1 行目)
- tech-debt の台帳: d33bb358 は触っていない。この PR の 3 行(hook の通知文の行、手順 6 の「narrowest」の検査の穴の解決済みコメントと取り消し線の行)は前回までに確かめた形のまま。RESOLVED のコメントが挙げる手順 6 の文末(「…the narrowest check that covers the slice's files.」)は d33bb358 のあとも手順 6 にある。前回の run が下書きにした「再開の手順の穴 3 つ」の行は台帳に足していなかった(`grep -c` が 0)ので、取り下げる行はない
- 確かめていない: テストと静的検査の実行(実装側の記録では slice 138/138、scope 202/202、agent-models 47/47 だが、d33bb358 のあとの数は未確認)、各テストの変異の再現

## Findings

所見はない(CRITICAL・HIGH・MEDIUM・LOW とも 0 件)。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| — | — | なし | 上の Previous findings と Evidence reviewed | — |

所見にしなかった気づき:

- `model-routing.md:52-55` の段落は「the orchestrator removes the secret or narrows the slice and dispatches it again as a resume」と書き、手順 7 の「First take a secret out of the worktree. Then narrow the slice … if needed」の順序を「or」で言い換えている。段落は末尾で手順 7 を指しており、手順 7 の順序の方が強い。読み違えて秘密情報が残るのは、段落だけを読み、手順 7 を開かず、かつ絞るだけで秘密情報のファイルが落ちない場合に限る。絞ってファイルを落とす場合は手順 7 が破棄を求める。次にこの段落を触るときに「removes the secret, narrows the slice if needed」にすれば足りる
- 手順 7 の「First take a secret out of the worktree」は、検証の失敗で止まった場合(秘密情報がない)にも先頭に出る。「a secret」は「ある場合は」と読めるが、「the secret, if there is one」の方が迷わない。文意は変わらないので所見にしない

## Positive notes

- 直しは 4 つの implementer 定義、`implement/SKILL.md` の 4 写し、model-routing の 2 本に同時に届き、写しは `cmp` で一致している
- 手順 7 の片付けが、状態ごとの挙動を確かめた手順になった。追跡中のファイルとステージ済みの新規ファイルは同じコマンドで消え、untracked だけを削除に分けている。先頭の命令に理由(`wip:` が `git add -A` で履歴に入れる)が付いていて、hook の実装で裏が取れる
- 「値を書かない」を、報告を書く側(implementer)、報告を計画に写す側(inline の手順)、契約(model-routing)の 3 つに入れた。他の契約の「証拠は値でなく指す」(`agent-messaging.md` の EVIDENCE-as-pointers)と同じ方向になった
- Resume の欄は表に任意として足し、implementer 側の必須の一覧は 6 項目のまま保った。必須の一覧と任意の欄の読み分けが崩れていない
- 直しの各文に、前の文面を落とさずに新しい文面を固めるテストが付いた。変わった needle は 1 つずつ元の文面を書き換え、足した文は別の assert にしている

## Tech debt identified

なし。cycle 2 の 1 回目の run が下書きにした「再開の手順の穴 3 つ」の行は、3 件とも解決したので取り下げた(台帳には足していなかった)。`docs/tech-debt/README.md` はこの report では変えていない。

## Recommendation

- Merge: yes(CRITICAL・HIGH・MEDIUM・LOW とも 0 件)
- Follow-ups: 所見はない。気づき 2 件は文言の磨きで、直すなら次に model-routing の段落か手順 7 を触るときでよい。計画の `## Progress checklist` には、cycle 2 の 1 回目の self-review(2478fd5d、LOW 3 件)と d33bb358 の記録がまだない(a9b63c8a が cross-review の直しを足したのと同じ形で、orchestrator の plan upkeep)。`.goreleaser.yml` の changelog は `docs:` と `test:` を落とし、この PR の 20 コミットはすべてその 2 種なので、downstream に届く skill と agent の変更は自動の changelog に Merge 行しか出ない。`/pr` の本文と release の issue に 1 行(「`/implement` がフルの検査を打たない」「`/self-review` が 6 項目になった」「止まったスライスは再開として出し直す」)を足す価値がある
