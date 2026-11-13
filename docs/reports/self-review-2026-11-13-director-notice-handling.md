# Self-review report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(issue #318)
- Branch: `docs/director-notice-handling`、base は `origin/main`。レビューした HEAD は `50c9345c`(S3 の記録)。スライスは `a933346c`、`aa3d1878`、`21026fd4`
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(cycle 1)
- Scope: `git diff origin/main...HEAD` のうち、director の skill の 4 写し(新しい小節、指しの 2 か所、「してはいけないこと」の 1 項)、`internal/org/prompts/director.md` と `internal/org/prompts/leader.md` の追加文、`internal/org/prompts_notice_test.go`、`internal/cli/director_notice_skill_test.go`、`docs/tech-debt/README.md` の 1 行。diff の質だけを見た。文が書く動き(どの動詞がどの行を出すか、行の後ろで動詞が続くか、`<dir>` が何か)は、`internal/cli/scaffold_version_notice.go`、`internal/cli/codex_hook_trust_notice.go`、呼び出し側の `org.go`、`org_director.go`、`org_director_headless.go`、`status.go` を読んで確かめた。仕様への適合、テストの網羅、文書の食い違い(`--help` など)は見ていない(`/verify`、`/test`、`/sync-docs` の範囲)。テストは走らせていない
- Addendum: 修正の commit `579a35a6`(親は `450535e2`)を `git show 579a35a6` で見た。7 ファイル、137 行追加、82 行削除。HEAD は `839a88d3`(計画の Progress に直しを記録した commit)。パイプラインは cycle 1 のまま(`/verify` の前)。下の本文と表の M1、L1 から L4 は `50c9345c` の時点の文で書いたままにし、直したあとの確認は「Addendum」の節にまとめた

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
| MEDIUM | readability | [解決済み 579a35a6。M1 の (a) が名指した「昇格」の 2 の括弧は残り、N1。読み直す先の直しは N2 を生んだ] M1: skill の小節が「`<dir>` は main worktree のルートの絶対パス」と、4 つの動詞すべての行について書いているが、`ralph status` の行はそうではない。`status.go` は `printScaffoldVersionNotice(..., "status", ".")` と `"."` を渡すので、`<dir>` は打った cwd の絶対パス(cwd に `.ralph/manifest.toml` があるときだけ行が出る)。linked worktree の director が `ralph status` を打つと、`<dir>` は main ではなく自分の worktree になる。ほかの 3 つの動詞は `orgScaffoldRoot` で main worktree のルートを渡す。結果として、(a) 「昇格」の 2 の status の行と 3 の attach の行で `<dir>` が違いうるのに、小節は 1 つの `<dir>` として書いている、(b) 「linked worktree で動いていて、自分の作業ディレクトリの skill が `<dir>` のものと違うとき」の分岐は、status の行では起きえない(`<dir>` が自分の worktree)ので、読んだ director が話を取り違える、(c) 同じ `<dir>` と版の組の「1 度だけ伝える」が、status と org 系で別の組になる。同じ主張を、テストの `directorNoticeRules` の 1 つが逐語で固定している(`internal/org/prompts_notice_test.go:74`)ので、誤りがテストで守られる。計画の D3(「main のチェックアウトの絶対パス、`orgScaffoldRoot`」)も同じ前提で、`ralph status --help` の段落(「current directory's」)とは食い違う | `.claude/skills/director/SKILL.md:1451-1452`(4 写しとも)、`internal/cli/status.go:223`(`"."`)、`internal/cli/scaffold_version_notice.go:81-92` と `:101-105`、`internal/cli/scaffold_version_notice.go:119-126`(`statusScaffoldVersionNoticeHelp`: current directory's)、`internal/org/prompts_notice_test.go:74` | 小節を「`<dir>` は、`ralph org ...` の動詞の行では main worktree のルートの絶対パス、`ralph status` の行では打った cwd の絶対パス(linked worktree の中なら、その worktree)」に直す。3 つ目の bullet の linked worktree の分岐は、org 系の行のときの話だと分かる書き方にする。`directorNoticeRules` の該当の断片も同じ文に合わせる。`director.md` の「行の `<dir>`」は、行の値を指すだけなので直さなくてよい |
| LOW | readability | [解決済み 579a35a6。`rejected:` の従う先は N3] L1: leader の「運用規律」の hook の項の止める規則に、端が 2 つ書かれていない。(a) 行は `rt.Spawn` の前に出る(`org.go:966`)ので、行が出たあとに spawn が `spawn failed:` で失敗することがある。項は「終了コード 0 で終われば座席は使える」とだけ書き、QUESTION を上げる条件は終了コードに結び付けていない。読んだ leader は、まだ存在しない座席について QUESTION を上げ、続けて失敗の BLOCKED も上げうる。打ち直しの spawn は同じ行をもう 1 度出す。(b) 同じ座席の同じ行を上げない期間は「返事が来る前に」で切れていて、返事が来たあとに同じ座席が打ち直されて同じ行が出たときに上げるかは書かれていない | `internal/org/prompts/leader.md:1335-1350`、`internal/cli/org.go:966-967` | 「spawn が終了コード 0 で終わったあとに上げる」と書く(失敗したときは、打ち直しが通ってから上げる)。返事のあとの扱いも 1 句で決める(たとえば、返事が来たあとは同じ座席の同じ行を上げない)。テストの断片(`leaderHookNotePhrases`)も合わせる |
| LOW | readability | [解決済み 579a35a6] L2: leader の項は、QUESTION の中身を決めていない。「本文には、その行を変えずに書き、座席 id を添えます」とあるだけで、何を人に聞く QUESTION なのか(この座席をそのまま使うか、止めるか)、座席 id を書く位置が書かれていない。「件を上げる」の節の例は BLOCKED で、近くの QUESTION(計画の承認)は `PLAN:` と `VISUAL:` のヘッダと本文の形まで書いている。director は件を読んで人に聞くので(skill の「件の処理」)、人に聞く 1 文が QUESTION になければ、人は行を読んで何を決めるのか分からない | `internal/org/prompts/leader.md:1340-1341`、`:208-214`(計画の承認の QUESTION は形を決めている) | 本文の 2 行を決める。たとえば 1 行目に `SEAT: <座席 id>`、続けて行、最後に「この座席をそのまま使うか止めるか」の問い。形を決めた文を断片に足す |
| LOW | maintainability | [未解決・直さない決定(`839a88d3`、計画の Progress)。理由は Addendum] L3: `internal/cli` のテストの 2 点。(a) skill の小節は hook の行を「`...; run 'ralph doctor' for <the key(s)> and how to <直し方>` で終わる」と書くが、テストが行の末尾まで確かめるのは scaffold の行だけ(`strings.HasSuffix`)。hook の行は `strings.Contains` なので、行の後ろに句が足されても落ちない。(b) 小節を切り出す `directorNoticeSkillSubsection` は、次の `# `、`## `、`### ` の行で終わる。`internal/org` 側の `directorSkillSubsection` は `#` で始まる行すべてで終わる。同じ小節を 2 つの読み方で切るので、`####` の見出しを小節の中に足すと、2 つのテストで切れ方が変わる(今は小節に見出しも fenced block もなく、差は出ない) | `internal/cli/director_notice_skill_test.go:63-67`(Contains)、`:73-76`(HasSuffix は scaffold のみ)、`:109-129`、`internal/org/director_skill_sync_test.go:198-215` | (a) hook の行も、`doctorFragment` のあとに ` and how to ` が 1 回あり、そのあとに `; ` がないことを確かめるか、skill の「で終わる」を「を含む」に弱める。(b) `directorNoticeSkillSubsection` の comment に、`internal/org` の読み方との違いを 1 句書くか、`#` で始まる行に合わせる |
| LOW | readability | [解決済み 579a35a6] L4: `prompts_notice_test.go` の先頭 comment の 14 から 16 行目が、前後の行(約 75 字で折り返し)より長い 1 行(約 95 字)で、折り返しが途中で崩れている(`... these tests pin the director's rules. scripts/check-skill-sync.sh keeps the four copies of`)。読みづらいだけで動きは変わらない | `internal/org/prompts_notice_test.go:13-16` | 15 行目を前後と同じ幅で折り直す |
| MEDIUM | readability | N2(addendum で追加): M1 の直しは読み直す先を行の `<dir>` から main worktree のルート `<ルート>` に替えたが、`ralph status` の行が linked worktree W を名指すときは、人が upgrade を打つ場所(行の `run 'ralph upgrade' in <dir>` の W)と、director が読み直す場所(main の M)が別になる。人が行のとおり W だけを upgrade して「済ませた」と言うと、director は upgrade されていない M の skill を読み、続く一句(「自分の作業ディレクトリの skill が `<ルート>` のものと違うときは、`<ルート>` の方に従い、自分の worktree の写しが古いままだと人に伝える」)に従って、新しくなった W の写しを「古いまま」と伝え、古い M に従う。直す前の文(行の `<dir>` の skill を読み直す)は、行が M を名指しても W を名指しても、人が upgrade した場所と読み直す場所が同じなので、この食い違いは無かった。計画の前提(Risks の「人は main を更新した」、D3)は、行が main を名指す org の動詞の行の話で、status の行には当てはまらない。同じ文を `directorNoticeRules` の `<ルート>` を含む 4 断片と `director.md` の文が固定する。隣の「それ以上のことはしない」も、直後の読み直しの規則を止めるのか、伝える文の長さを言うのか、読み分けられない。linked worktree の動きは計画も未確認としているので、これは読みで見つけた(走らせていない) | `.claude/skills/director/SKILL.md:1458-1461`(「それ以上のことはしない」)、`:1465-1476`(読み直し、4 写しとも同じ)、`internal/org/prompts/director.md:59-60`、`internal/org/prompts_notice_test.go` の `directorNoticeRules`(`<ルート>` を含む 4 断片)と `directorPromptNoticePhrases`(「main worktree のルートの skill を絶対パスで読み直し」)、`internal/cli/status.go:223`(`"."`)、計画の `## Risks and mitigations` の 4 つ目の項と D3 | 読み直す先を、人が upgrade した場所(行の `<dir>`)に戻す。たとえば「人が upgrade を済ませたと言ったら、どの `<dir>` で打ったかを人に確かめ(行が 2 つ出ていたら両方)、その `<dir>` の skill を絶対パスで読み直す」。「自分の作業ディレクトリの skill が `<dir>` のものと違うときは `<dir>` の方に従う」は、`<dir>` が M の行(`org start`、`attach`、`director start`)のときにだけ起きるので、そのまま残せる。`<ルート>` と `git worktree list` の一句は要らなくなる(残すなら、平文の `git worktree list` はパスに空白があると列と区別できないので、`--porcelain` の最初の `worktree <path>` の行にする)。「それ以上のことはしない」は、何をしないのか(`ralph upgrade` を打たない、M の雛形を調べに行かない、など)に書き替える。`director.md:59` と 4 断片も同じ文に合わせる |
| LOW | readability | N1(addendum で追加): 「昇格」の 2 の `(3 の attach も同じ行を出す)` が、M1 の直しのあとは linked worktree で偽になる。status の行の `<dir>` は打ったディレクトリ W、attach の行の `<dir>` は `orgScaffoldRoot` が返す main のルート M で、W と M は雛形の版も違いうるので、同じ行ではない。直した小節は「同じ `<dir>` と版の組の行はもう 1 度伝えない」と書くので、小節の規則では 2 本とも伝えるが、括弧は「attach の行は status の行の繰り返し」と読ませ、2 本目を伝えずに済ませる読み方を許す。M1 の (a) が名指した 2 か所(「昇格」の 2 と 3)のうち、小節は直り、この括弧は残った。括弧は `directorNoticeRules` とは別に、`TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` の「step 2 of the section ## 昇格」が逐語で固定しており(M1 と同じ形)、直さない限りテストが守る | `.claude/skills/director/SKILL.md:92`(4 写しとも同じ)、`internal/org/prompts_notice_test.go:134`、`internal/cli/org_director.go:151`(attach は `orgScaffoldRoot`)、`internal/cli/status.go:223`(`"."`) | 括弧を「(3 の attach の行は main worktree のルートを名指す。linked worktree で打っていると、status の行と `<dir>` が違いうる)」にするか、括弧を外して指しだけにする。`prompts_notice_test.go:134` の断片も同じ文に合わせる(4 写し) |
| LOW | readability | N3(addendum で追加): L1 の直しの「上の項など失敗のときの決まりに従ってください」は、`rejected:` のとき従う先が決まらない。直上の `spawn failed:` の項は、自分で「stdout の `rejected:` の行で断られたとき(…)は、ここには当たりません」と `rejected:` を外している。`rejected:` の断りに当たる項は、見張りの項の 1 つだけ(見張りが動いていない断り)で、同じ座席の別の spawn が動いている断りや承認の検査の断りには、従う先が書かれていない。QUESTION を上げない(この文の目的)は伝わるので害は小さいが、読んだ leader は `spawn failed:` の項を開いて、そこに外されている | `internal/org/prompts/leader.md:1339-1340`(新しい文)、`:1308-1313`(`spawn failed:` の項と、`rejected:` を外す一句)、`:1295-1307`(見張りの項)。断片は `internal/org/prompts_notice_test.go:254` | 従う先の指しをやめ、「失敗した spawn では QUESTION を上げず、打ち直しが終了コード 0 で通ったとき、その座席の行で上げます」にする。指しを残すなら、`spawn failed:` は直上の項、`rejected:` は見張りの項に当たるときだけその項、当たらないときは QUESTION を上げず別の決まりもない、と分けて書く |

重大度の数(`50c9345c` の最初の run): CRITICAL 0、HIGH 0、MEDIUM 1、LOW 4(M1、L1 から L4)。

重大度の数(addendum 後、`579a35a6` 時点): CRITICAL 0、HIGH 0、MEDIUM 1 件が未解決(N2)、LOW 3 件が未解決(L3、N1、N3)。M1、L1、L2、L4 の 4 件は解決済み。

## Addendum: `579a35a6` の確認

読んだもの。`git show 579a35a6` の全部(skill の 4 写し、`director.md`、`leader.md`、`prompts_notice_test.go`)、`839a88d3`(計画の Progress)、skill の小節(`:1442-1488`)、「昇格」の 1 から 3(`:78-135`)、「headless の director」の「始める」(`:350-372`)、`status.go:205-224`、`scaffold_version_notice.go` の全部、`org.go:930-1000`(spawn)、`org_director.go:151`、`org_director_headless.go:344,366`、`gitroot.go:35-80`、`leader.md:1286-1355`、`director_notice_skill_test.go:60-110`、計画の D3、D5、Risks。skill の 4 写しは `cmp` で同一。テストと linter は走らせていない。

| 指摘 | 結果 | 確かめたこと |
| --- | --- | --- |
| M1 | 解決(小節、`director.md`、断片)。残りは N1 と N2 | 小節は `<dir>` を「`ralph org` の 3 つの動詞の行では main worktree のルート、`ralph status` の行では status を打ったディレクトリ」と書き分けた。`status.go:223` は `printScaffoldVersionNotice(..., "status", ".")` で、`scaffoldVersionNotice` が `filepath.Abs(".")`(cwd)を行に書く。`org.go:1196`、`:1312`(`org start` の 2 つの形)、`org_director.go:151`(`attach`)、`org_director_headless.go:344`(`director start`)の 4 呼び出しは `orgScaffoldRoot` を渡す。3 つの動詞と数が合う。`orgScaffoldRoot` は main のルートが取れないときだけ toplevel か `.` に落ちるので、小節の「main worktree のルート」は通常の道の言い方として足りる。`git worktree list` の 1 行目は、`gitroot.MainWorktree` が読む `--porcelain` の最初の record と同じ(git は main worktree を最初に出す)。`directorNoticeRules` の断片は skill の語順(人が upgrade を…、`(Claude Code は…)`、セッションが…、`<ルート>` は…、linked worktree で…)に合い、`director.md` の文も `<ルート>` 側に揃った。残り: 「昇格」の 2 の括弧(N1)、読み直す先(N2) |
| L1 | 解決。`rejected:` の従う先は N3 | `printCodexHookTrustNotice` は `rt.Spawn` の前(`org.go:966-967`)にある。失敗は `printSpawnResult` が stdout に `rejected:`(`SpawnOutcomeRejected`)か `spawn failed:`(`SpawnOutcomeFailed`)を出し、どちらも `result.Err` を返す。直した文は、この 2 つの失敗では QUESTION を上げず、上げるのは「終了コード 0 で終わり、行を出していた」ときだけ、と肯定の条件で書く。打ち直しが失敗した行は数えず、通った打ち直しで 1 回上げる形になる。返事のあとは「DECISION が届いたあとに同じ座席の spawn が同じ行を出したときも上げません」で決まった |
| L2 | 解決 | 「QUESTION では、その座席をこのまま使うか止めるかを、hook の中身を読んで決めるよう人に尋ねます」を足した。座席 id を書く位置は決めていないが、行そのものが `note: <動詞>: codex seat <座席>: ...` の形で座席を含む(`codexHookTrustAgent`)ので、本文の「座席 id を添えます」と合わせて、人が座席を取り違える形ではない。「DECISION が届いたら、そのとおりにしてください」も足り、止める答えの行き先(`ralph org stop`)は同じ節の前の項が使っている |
| L3 | 未解決。直さない決定の理由は妥当 | `839a88d3` が計画の Progress に「今は差が出ず、断片が行に含まれることは確かめている」と書いた。確かめ: `579a35a6` は `internal/cli/director_notice_skill_test.go` に触れていない。skill の小節には `####` の見出しも fenced block も無く(`TestDirectorSkill_SaysWhatToDoWithTheHumanNotes` が fenced block を禁じる)、2 つの切り出しの違いは出ない。hook の行の末尾は `strings.Contains` のままだが、今の行は skill が書く「…and how to <直し方>」で終わる。LOW のまま。理由が残るのは計画の Progress だけで、計画は `/pr` で archive され、`docs/tech-debt/README.md` には行が無い。PR まで持ち越すなら、PR の Known gaps に 1 行書くか register に足す |
| L4 | 解決 | `prompts_notice_test.go:12-16` の comment が、前後と同じ約 75 字の折り返しになった |
| N1 | 新規(LOW) | 上の表。M1 の (a) が名指した 2 か所のうち、「昇格」の 2 の括弧が残った |
| N2 | 新規(MEDIUM) | 上の表。M1 の直しが読み直す先を替えたことで生じた。status の行が linked worktree を名指す場合の、人が upgrade する場所と director が読み直す場所の食い違い |
| N3 | 新規(LOW) | 上の表。L1 の直しの指しが、`rejected:` のとき従う先を持たない |

M1 の直しが足した 2 つの文の確かめ(N1、N2 の根拠)。

- 「昇格」の 2 は、`ralph status` の stderr の行を見たら小節に従うと書き、括弧で「3 の attach も同じ行を出す」と言う。`orgScaffoldRoot` は linked worktree の中から打っても main のルートを返す(`scaffold_version_notice.go:81-92`)ので、attach の行の `<dir>` は M、status の行は W になる
- 「昇格」の 1 は、repo の中なら main のチェックアウトでもその linked worktree でも動くと書く。つまり director が W にいる場合は設計の内で、skill 自身が linked worktree の分岐を書いている(`:1471-1473`)
- headless の director は台帳の main のルートで動く(`:344-345`)ので、status の `<dir>` は M になり、N1 と N2 は昇格した director が linked worktree にいる場合だけ起きる

## Positive notes

- 文が書く動きは、production code と合っている点が多い。4 つの動詞が出す行の形、`codex director` の呼び名、`ralph status` が stdout を変えないこと、行の後ろで動詞が続くこと(エラーを返さない `Fprintln`)を、呼び出しの場所まで読んで確かめた。M1 の `<dir>` の説明だけが外れている
- 計画の D2 と D4 が、3 つの面(skill の小節、「してはいけないこと」、headless の prompt)で同じ言い方になっている。skill の小節は、director が打たないこと(`ralph upgrade`、codex の設定の書き換え、hook の承認)と、人に任せること(流すか、承認するか)を分けて書き、流し方の例を書かない理由(linked worktree での動きを確かめていない)を計画の側に残している。テストは fenced block がないことまで固定している
- leader の「件を上げる」の止める規則から例外の項へ、項から止める規則へ、双方向に指している。テストは両側を確かめるので、片方だけ変えると落ちる
- テストの失敗メッセージが、何を直すか(skill の 4 写しと `sync-skills.sh`、断片をどこで変えるか)を言う。skill のコピーがないときの `t.Skipf` は、同じファイルの前例(`checkSkillPlaces`)と同じ。`internal/cli` のテストは `internal/org` を import できないという制約のなかで、断片を 2 か所に持ち、skill の側で両方を要求することで、ずれを落とせる形にしている
- `docs/tech-debt/README.md` の変更は、前例どおりのコメント行と取り消し線。行の中身の主張(skill の 4 写し、ミッションの 1 項、`--help` は変えない)は diff と合っている
- (addendum)M1 の直しは、`<dir>` を動詞ごとに書き分け、書き分けの根拠(`status.go:223` の `"."`)をテストの comment と断片に残した。「status の行だけ cwd」という production code の差を、skill・`director.md`・テストの 3 か所が同じ言い方で持つ
- (addendum)L1 の直しは、QUESTION を上げる条件を「終了コード 0 で終わり、行を出していた」という肯定の 1 文にした。失敗の文はその補足に当たり、打ち直しで行がまた出る場合も条件の 1 文で足りる。返事のあとの扱い(「DECISION が届いたあとも上げません」)と、DECISION に従う文が入り、leader の項は返事の前後を通して決まっている
- (addendum)直しの 4 写しは `cmp` で同一のまま。S4 は計画の本文を書き換えず、D3 からのずれ(`<dir>` は main のルートという前提が status の行に当たらないこと)を Progress の行に残した。ずれの記録は正しい場所にある。N2 はそのずれの先の話で、記録された「読み直す先は main のルート」という決定そのものを見直す指摘になる

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| L3(`internal/cli/director_notice_skill_test.go` の hook の行は `Contains` で、小節の切り出しの終わりの規則が `internal/org` と違う) | 今は差が出ない。hook の行の後ろに句が足されても、`####` の見出しが小節に入っても、テストが気づくのが遅れる | `839a88d3` の決定。今の行は skill の書く末尾で終わり、小節に `####` も fenced block も無い。理由は計画の Progress だけにあり、計画は `/pr` で archive される | hook の行に末尾の句が足されるとき、または小節に `####` を足すとき | この report の L3。`docs/plans/active/2026-11-13-director-notice-handling.md` の Progress |
| N1、N2、N3(直さずに進める場合のみ) | N2 は、director が linked worktree にいて status の行が worktree を名指すとき、古い skill に従う。N1 と N3 は文の食い違いで、害は小さい | 未決。直すかどうかはオーケストレーターが決める | 直さず `/pr` に進むときは、PR の Known gaps に 1 行ずつ書く | この report の N1 から N3 |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_ 行は `docs/tech-debt/README.md` に足していない(この commit の範囲は report と insight event だけ)。L3 を PR まで持ち越すなら、PR の Known gaps に書くか register に足すかを `/pr` の側で決める。N1 から N3 は、直せば register に載せる必要が無い。この diff は register のどの行も閉じていないし、開いてもいない(`579a35a6` は `docs/tech-debt/README.md` に触れない)。

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 1、未解決の LOW 3。addendum 後。`50c9345c` の最初の run は MEDIUM 1、LOW 4 で、M1・L1・L2・L4 は `579a35a6` で解決済み。未解決は N2(MEDIUM)、L3・N1・N3(LOW))。CRITICAL と HIGH はない
- Follow-ups: N2 は、直すなら skill の 1 文(4 写し)、`director.md` の 1 文、`directorNoticeRules` の `<ルート>` を含む 4 断片と `directorPromptNoticePhrases` の 1 断片で、読み直す先を行の `<dir>` に戻す。N1 は skill の括弧 1 つ(4 写し)と `prompts_notice_test.go:134` の断片、N3 は `leader.md` の 1 文と `leaderHookNotePhrases` の 1 断片。どれも文と断片の直しで、同じ commit に入る。パイプラインはまだ `/verify` に進んでいないので、直したあとの確認はこの report の Addendum をもう 1 度更新する形で足り、cycle は増えない。直さずに進める場合は、N2 を PR の Known gaps に書く。レビューしていないもの: spec 適合、テストの網羅と実行結果、`--help` と spec 文書の食い違い(計画の D3 と Risks の本文が、出荷される skill の読み直し先と食い違う点を含む。`/verify` と `/sync-docs` の範囲)、`/org` skill に同じ規則がないこと(計画の Non-goals)。linked worktree での director の動きは、計画も未確認としており、N2 は読みだけで見つけた
