# Self-review report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(issue #318)
- Branch: `docs/director-notice-handling`、base は `origin/main`。レビューした HEAD は `50c9345c`(S3 の記録)。スライスは `a933346c`、`aa3d1878`、`21026fd4`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(cycle 1)
- Scope: `git diff origin/main...HEAD` のうち、director の skill の 4 写し(新しい小節、指しの 2 か所、「してはいけないこと」の 1 項)、`internal/org/prompts/director.md` と `internal/org/prompts/leader.md` の追加文、`internal/org/prompts_notice_test.go`、`internal/cli/director_notice_skill_test.go`、`docs/tech-debt/README.md` の 1 行。diff の質だけを見た。文が書く動き(どの動詞がどの行を出すか、行の後ろで動詞が続くか、`<dir>` が何か)は、`internal/cli/scaffold_version_notice.go`、`internal/cli/codex_hook_trust_notice.go`、呼び出し側の `org.go`、`org_director.go`、`org_director_headless.go`、`status.go` を読んで確かめた。仕様への適合、テストの網羅、文書の食い違い(`--help` など)は見ていない(`/verify`、`/test`、`/sync-docs` の範囲)。テストは走らせていない

## Evidence reviewed

- `git diff origin/main...HEAD` の全体(10 ファイル、+791 -6)。skill の 4 写しは、追加行が互いに同じ(`diff` で差なし)
- 行を出す場所の読み取り(production code):
  - `printScaffoldVersionNotice` の呼び出し: `org.go:1196`(task 形の `org start`)、`org.go:1312`(feature 形)、`org_director.go:151`(`director attach`)、`org_director_headless.go:344`(`director start`)は、どれも `orgScaffoldRoot(...)` を root に渡す。`status.go:223` だけが `"."` を渡す
  - `printCodexHookTrustNotice` の呼び出し: `org.go:966`(`spawn`、`rt.Spawn` の前)、`org.go:1201` と `org.go:1331`(`start`、`rt.Spawn` / `rt.StartFeature` の前)、`org_director_headless.go:366`(`director start`)
  - どちらの行も `_, _ = fmt.Fprintln` で書き、エラーを返さない。行の後ろで動詞は続く(「どちらが出ても動詞はそのまま進む」は正しい)
  - 行の形: `scaffoldVersionNotice` の `Sprintf`(`note: <verb>: the scaffold in <dir> (.ralph/manifest.toml) is ralph <m>, older than this CLI (<c>), ...; run 'ralph upgrade' in <dir>`)と `codexHookTrustNotice` の `Sprintf`(`note: <verb>: <who>: <head>, so <skip> in <where>; run 'ralph doctor' for <key> and how to <fix>`)は、skill の小節の「行は〜の形」と合っている。`codexHookTrustAgent` の `codex director` も合っている
- 文の相互の整合: skill の小節、「昇格」の 2 と「機能ごとの出力先」の 3 の指し、「してはいけないこと」の項、`director.md` の ミッションの項、`leader.md` の「運用規律」の項と「件を上げる」の例外の一句。skill の「件の処理」(QUESTION は人に聞く)、「終える」(director は stop を打たない)と食い違わない
- テスト 2 本(全文)と、使っている既存の補助(`checkSkillPlaces`、`checkSkillHeadingOrder`、`directorSkillSubsection`、`assertPhrasesInOrder`)の規約(コピーがないときの `t.Skipf`)との一致
- tech-debt の行: 前例どおり、コメント行(`<!-- RESOLVED ... -->`)を行の前に置き、行を `~~` で消している(`docs/tech-debt/README.md:464-465`)

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | readability | M1: skill の小節が「`<dir>` は main worktree のルートの絶対パス」と、4 つの動詞すべての行について書いているが、`ralph status` の行はそうではない。`status.go` は `printScaffoldVersionNotice(..., "status", ".")` と `"."` を渡すので、`<dir>` は打った cwd の絶対パス(cwd に `.ralph/manifest.toml` があるときだけ行が出る)。linked worktree の director が `ralph status` を打つと、`<dir>` は main ではなく自分の worktree になる。ほかの 3 つの動詞は `orgScaffoldRoot` で main worktree のルートを渡す。結果として、(a) 「昇格」の 2 の status の行と 3 の attach の行で `<dir>` が違いうるのに、小節は 1 つの `<dir>` として書いている、(b) 「linked worktree で動いていて、自分の作業ディレクトリの skill が `<dir>` のものと違うとき」の分岐は、status の行では起きえない(`<dir>` が自分の worktree)ので、読んだ director が話を取り違える、(c) 同じ `<dir>` と版の組の「1 度だけ伝える」が、status と org 系で別の組になる。同じ主張を、テストの `directorNoticeRules` の 1 つが逐語で固定している(`internal/org/prompts_notice_test.go:74`)ので、誤りがテストで守られる。計画の D3(「main のチェックアウトの絶対パス、`orgScaffoldRoot`」)も同じ前提で、`ralph status --help` の段落(「current directory's」)とは食い違う | `.claude/skills/director/SKILL.md:1451-1452`(4 写しとも)、`internal/cli/status.go:223`(`"."`)、`internal/cli/scaffold_version_notice.go:81-92` と `:101-105`、`internal/cli/scaffold_version_notice.go:119-126`(`statusScaffoldVersionNoticeHelp`: current directory's)、`internal/org/prompts_notice_test.go:74` | 小節を「`<dir>` は、`ralph org ...` の動詞の行では main worktree のルートの絶対パス、`ralph status` の行では打った cwd の絶対パス(linked worktree の中なら、その worktree)」に直す。3 つ目の bullet の linked worktree の分岐は、org 系の行のときの話だと分かる書き方にする。`directorNoticeRules` の該当の断片も同じ文に合わせる。`director.md` の「行の `<dir>`」は、行の値を指すだけなので直さなくてよい |
| LOW | readability | L1: leader の「運用規律」の hook の項の止める規則に、端が 2 つ書かれていない。(a) 行は `rt.Spawn` の前に出る(`org.go:966`)ので、行が出たあとに spawn が `spawn failed:` で失敗することがある。項は「終了コード 0 で終われば座席は使える」とだけ書き、QUESTION を上げる条件は終了コードに結び付けていない。読んだ leader は、まだ存在しない座席について QUESTION を上げ、続けて失敗の BLOCKED も上げうる。打ち直しの spawn は同じ行をもう 1 度出す。(b) 同じ座席の同じ行を上げない期間は「返事が来る前に」で切れていて、返事が来たあとに同じ座席が打ち直されて同じ行が出たときに上げるかは書かれていない | `internal/org/prompts/leader.md:1335-1350`、`internal/cli/org.go:966-967` | 「spawn が終了コード 0 で終わったあとに上げる」と書く(失敗したときは、打ち直しが通ってから上げる)。返事のあとの扱いも 1 句で決める(たとえば、返事が来たあとは同じ座席の同じ行を上げない)。テストの断片(`leaderHookNotePhrases`)も合わせる |
| LOW | readability | L2: leader の項は、QUESTION の中身を決めていない。「本文には、その行を変えずに書き、座席 id を添えます」とあるだけで、何を人に聞く QUESTION なのか(この座席をそのまま使うか、止めるか)、座席 id を書く位置が書かれていない。「件を上げる」の節の例は BLOCKED で、近くの QUESTION(計画の承認)は `PLAN:` と `VISUAL:` のヘッダと本文の形まで書いている。director は件を読んで人に聞くので(skill の「件の処理」)、人に聞く 1 文が QUESTION になければ、人は行を読んで何を決めるのか分からない | `internal/org/prompts/leader.md:1340-1341`、`:208-214`(計画の承認の QUESTION は形を決めている) | 本文の 2 行を決める。たとえば 1 行目に `SEAT: <座席 id>`、続けて行、最後に「この座席をそのまま使うか止めるか」の問い。形を決めた文を断片に足す |
| LOW | maintainability | L3: `internal/cli` のテストの 2 点。(a) skill の小節は hook の行を「`...; run 'ralph doctor' for <the key(s)> and how to <直し方>` で終わる」と書くが、テストが行の末尾まで確かめるのは scaffold の行だけ(`strings.HasSuffix`)。hook の行は `strings.Contains` なので、行の後ろに句が足されても落ちない。(b) 小節を切り出す `directorNoticeSkillSubsection` は、次の `# `、`## `、`### ` の行で終わる。`internal/org` 側の `directorSkillSubsection` は `#` で始まる行すべてで終わる。同じ小節を 2 つの読み方で切るので、`####` の見出しを小節の中に足すと、2 つのテストで切れ方が変わる(今は小節に見出しも fenced block もなく、差は出ない) | `internal/cli/director_notice_skill_test.go:63-67`(Contains)、`:73-76`(HasSuffix は scaffold のみ)、`:109-129`、`internal/org/director_skill_sync_test.go:198-215` | (a) hook の行も、`doctorFragment` のあとに ` and how to ` が 1 回あり、そのあとに `; ` がないことを確かめるか、skill の「で終わる」を「を含む」に弱める。(b) `directorNoticeSkillSubsection` の comment に、`internal/org` の読み方との違いを 1 句書くか、`#` で始まる行に合わせる |
| LOW | readability | L4: `prompts_notice_test.go` の先頭 comment の 14 から 16 行目が、前後の行(約 75 字で折り返し)より長い 1 行(約 95 字)で、折り返しが途中で崩れている(`... these tests pin the director's rules. scripts/check-skill-sync.sh keeps the four copies of`)。読みづらいだけで動きは変わらない | `internal/org/prompts_notice_test.go:13-16` | 15 行目を前後と同じ幅で折り直す |

## Positive notes

- 文が書く動きは、production code と合っている点が多い。4 つの動詞が出す行の形、`codex director` の呼び名、`ralph status` が stdout を変えないこと、行の後ろで動詞が続くこと(エラーを返さない `Fprintln`)を、呼び出しの場所まで読んで確かめた。M1 の `<dir>` の説明だけが外れている
- 計画の D2 と D4 が、3 つの面(skill の小節、「してはいけないこと」、headless の prompt)で同じ言い方になっている。skill の小節は、director が打たないこと(`ralph upgrade`、codex の設定の書き換え、hook の承認)と、人に任せること(流すか、承認するか)を分けて書き、流し方の例を書かない理由(linked worktree での動きを確かめていない)を計画の側に残している。テストは fenced block がないことまで固定している
- leader の「件を上げる」の止める規則から例外の項へ、項から止める規則へ、双方向に指している。テストは両側を確かめるので、片方だけ変えると落ちる
- テストの失敗メッセージが、何を直すか(skill の 4 写しと `sync-skills.sh`、断片をどこで変えるか)を言う。skill のコピーがないときの `t.Skipf` は、同じファイルの前例(`checkSkillPlaces`)と同じ。`internal/cli` のテストは `internal/org` を import できないという制約のなかで、断片を 2 か所に持ち、skill の側で両方を要求することで、ずれを落とせる形にしている
- `docs/tech-debt/README.md` の変更は、前例どおりのコメント行と取り消し線。行の中身の主張(skill の 4 写し、ミッションの 1 項、`--help` は変えない)は diff と合っている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | M1 は直す前提。L1 から L4 はどれも小さい文とテストの直しで、直せるときに直す。直さずに進めるなら、オーケストレーターが理由を書いて登録する | - | - |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 追加した行はない。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、LOW 4)。CRITICAL と HIGH はない。M1 は `ralph status` の行の `<dir>` を誤って説明している文で、誤りがテストで固定されているので、PR の前に直すことを勧める(skill の 4 写し、`directorNoticeRules` の 1 断片)。直すと文と断片が変わるので、パイプラインは `/self-review` から回し直す(cycle 2)。M1 を直さずに進めるなら、理由を PR の本文に書く
- Follow-ups: M1 を直すときに L1 と L2(leader の 1 つの項)、L3、L4 も同じ commit で直せる。leader.md の直しは `leaderHookNotePhrases` の断片も動かす。レビューしていないもの: spec 適合、テストの網羅と実行結果、`--help` と spec 文書の食い違い、`/org` skill に同じ規則がないこと(計画の Non-goals)
