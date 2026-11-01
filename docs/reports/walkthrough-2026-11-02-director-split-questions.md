# Walkthrough: director-split-questions(issue #271)

- 計画: docs/plans/archive/2026-11-01-director-split-questions.md
- ブランチ: feat/director-split-questions(base 4fef6b8c)
- 差分: 25 ファイル、+5,906/−128。Go の本体は変えていない。変わったのは skill の文(director と org の 4 つの写し)、skill の文を固定するテスト 4 ファイル、文書、実機の証跡、pipeline の記録

## 読む順

1. `.claude/skills/director/SKILL.md` の「## 分割計画と機能ごとの org」(468〜709 行あたり)。5 つの `###` 節がこの変更の中心
2. 同じ skill の「### merge を待つ issue」の冒頭(引き継ぎ)と 1〜3、「### 予約外の文書の follow-up」の 1・7、「### 計画の承認」の 1・2、「## してはいけないこと」の新しい項
3. `.claude/skills/org/SKILL.md` の「### merge を待つ issue」の 2 文
4. `internal/org/skill_split_questions_test.go`(新規)。上の文を断片で固定する
5. `docs/evidence/director-ask-herdr-2026-11-01.md`。「人への聞き方」の根拠

`.agents/skills/` と `templates/base/` の写しは `scripts/sync-skills.sh` が作るもので、4 つとも同じ内容(`check-skill-sync.sh`、`check-sync.sh` が確かめる)。

## director の skill の新しい節

### 人への聞き方
- 選択肢で聞く場面(分割計画の前の質問、承認、出力先、計画の承認の `FORK`、merge のあとの着手)は、Claude Code では AskUserQuestion、Codex では番号つきの選択肢
- 登録の wake に push がある director(`wake:` の行に `push` が含まれる)は、AskUserQuestion を使わず番号つきの選択肢を文で書く。証跡の T3〜T5 で、見張りの 1 行と Enter が開いた問いを閉じた(カーソルの下の推奨の案が選ばれる、自由記述の欄なら 1 行が答えになる、選択肢の上では 1 行が消えて wake の件に気づかない)ため
- 番号の選択肢の返事を待つあいだに届いた見張りの 1 行は、人の答えではない

### 分割計画の前の質問
- `/spec` の手順 1〜3 を指して取り込む(聞かずに依頼を理解する、推奨つきの決定木の質問、残った曖昧さの確認)。AskUserQuestion の上限に合わせて 1 回に 4 問まで、推奨の案を先頭に「(推奨)」
- 機能をまたぐ分かれ目(機能の境目、共有のファイルの予約、座席の driver と model、機能の順)はここで決め、機能の内側は leader の `FORK`(#267)に任せる
- `/spec` の手順 4〜5 は分割計画と図解の承認が受け持ち、6〜8 は取り込まない

### 分割計画の承認
- 図解ページを見せ、Approve / Needs changes で聞く。Approve のときだけ digest の 2 行を書く(既存の 5 つの断片の文はこの節に移しただけ)

### 機能ごとの出力先
- `ralph status` の `reserved:` の行と機能の `- Reserve:` を `/org` の予約の規則で比べ、重なる機能は「issue に起票」、重ならない機能は「今すぐ着手」を推奨する
- `- Depends on:` で起票した機能に頼る機能も起票を推奨する
- start が重なりで断ったら、`ralph status` を読み直して重なりを全部数え直し、起票の推奨で聞き直す(start のエラーは最初の重なりだけを出す)

### 起票と記録
- 待つブランチは、重なる org の `feature:` の行のブランチ(全部)。`feature:` の行のない org は人に言う
- issue の本文の最後の行と follow-up の note に、同じ印の行 `ralph-split: <id>/<slug> after: <ブランチ>,...`(待つブランチがなければ `after: none`)。頼られる機能があれば `頼られる issue: #<番号>` の行
- 記録の前に `followup list` を読んで重ねない。ブランチがもう merge 済みなら記録しない(そのブランチの機能の org が走っているあいだは、使い回した名前の前の PR の `merged` を数えない)
- 印の行の形と読み方: 行全体の正規表現、行末の CR を取る、本文にちょうど 1 行、書いた人が `gh api user --hostname github.com` と同じ issue だけを使う。そろわない issue は使わず、番号と理由を人に言う。note は確かめた値から組み直す
- gh の issue と pr のコマンドには `--repo github.com/<owner>/<repo>` を付ける。origin の URL が github.com の repository を名指す形かを `grep -Eq` で確かめ、当たらなければ gh を打たず人に言う(GHES は対象外)

## 「merge を待つ issue」の変更

- 引き継ぎ: `followup list --all` に加え、`gh issue list` で印の行のある open の issue を探し、欠けた follow-up を記録し直す。数える件は NOTE が今の印の行と同じ件(待ちの世代)。done と open が混ざった issue は DONE_NOTE を答えとして残りを done にする
- 2: 印の行の `after:` のどのブランチも待ち終えた(`ready` の follow-up、記録の前に merge 済み、記録の直前に merge 済み)ときだけ、「<ブランチ> の実装が終わりました。コンフリクト回避のために起票した #<番号> に着手してよいですか」を 着手する / 後に回す / 要らない で聞く。頼られる issue が open か、予約が今の org と重なれば「後に回す」を推奨
- 3: 答えのあと、その issue の follow-up を done にする。follow-up のない issue に答えたら add してすぐ done にして台帳に残す。着手なら機能 1 つの分割計画を書き直し(前の start が残した worktree の記録を確かめる)、承認を取って start。start が断って人が待つと答えたら、重なる org を数え直して `after:` を書き換え、記録し直す(印の行が確かめに通らない issue は本文を書き換えず `followup add` だけ)

## テスト

- `internal/org/skill_split_questions_test.go`(新規、約 1,200 行): 新しい節と手順の断片を 4 つの写しで固定する。印の行・origin の URL・頼られる issue の正規表現は、skill の文字列とテストの定数をバイトで同じにし、Go の regexp で当たる行と当たらない行(shell に届く文字だけを持つ行を含む)を確かめる
- `director_skill_sync_test.go`: `## ` の節を返す `directorSkillSection` とその表のテスト(`directorSkillSubsection` は最初の `###` で止まるため)
- `skill_followup_test.go`・`skill_doc_followup_test.go`: 変わった文の断片の差し替え(意図は弱めていない)

## 経緯

pipeline は 6 回まわした(cycle の上限を 3、4 に上げた)。cross-review の指摘は毎回、印の行・follow-up・引き継ぎ・書き換えの状態の扱いの端の場合(中断、ブランチ名の使い回し、待ちの世代、fork と GHES の gh)だった。最後の run で残った 1 件(待ちの世代の識別子)は PR の Known gaps に書いた。各 run の所見と直しは計画の Progress checklist と、self-review・verify・test・sync-docs・cross-review の report にある。
