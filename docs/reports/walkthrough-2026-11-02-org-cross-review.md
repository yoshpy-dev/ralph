# Walkthrough: org-cross-review(issue #269)

- 計画: docs/plans/archive/2026-11-01-org-cross-review.md
- ブランチ: feat/org-cross-review(作り始めの base は 4fef6b8c。4424b257 で origin/main 31812a3b を取り込んだ)
- PR: #298
- 差分: 49 ファイル、+6,939/−250。Go のコマンドと設定のキーは変えていない。変わったのは leader と reviewer の役割の雛形、`/director` と `/org` の skill(各 4 つの写し)、pane の警告文、文書、テスト、実機の証跡、pipeline の記録

## 読む順

1. `internal/org/prompts/leader.md` の手順 9(349〜365 行あたり)。rebase のあと、sync-docs の前に cross-review を挟む位置が書いてある
2. 同じファイルの `### cross-review`(780〜1081 行)。設定の読み方、回数と上限、TASK の形、返事ごとの扱い、上限の QUESTION がここにある
3. `internal/org/prompts/reviewer.md` の `## cross-review の TASK`(134〜464 行)。TASK の行の確かめ、相手の選び方と呼び方、triage、report、返事
4. `.claude/skills/director/SKILL.md` の「### cross-review の上限」(1070 行あたり)と「人への聞き方」(476 行)
5. `internal/org/prompts_xreview_leader_test.go` と `internal/org/prompts_xreview_reviewer_test.go`。雛形から抜き出したコマンドを、スタブの CLI と bash・`zsh -f` で走らせる
6. `docs/evidence/xreview-live-probe-2026-11-01.md`。座席の外の shell で、本物の codex と claude を相手に呼んだ記録

`.agents/skills/` と `templates/base/` の写しは `scripts/sync-skills.sh` と同期の検査がそろえるもので、4 つとも同じ内容。

## leader の流れ

- 手順 9 で rebase のあと、worktree で `ralph config get org.review.cross_review` と `ralph config get org.review.max_cross_review_cycles` を打つ。off なら `skipped (off: [org.review].cross_review)` を控えて sync-docs に進む(D1)
- 小節の初めのコマンドが状態を数行で出す。数えるのは、このブランチが足した complete の report(`git diff --diff-filter=A`)だけ。skipped と incomplete の回は数えない。次の回の番号 `k` は、この org_id の report の番号のいちばん大きいもの +1(D3・D7b)
- 上限は TASK を送る前に確かめる。上限は `[org.review].max_cross_review_cycles` と、人が前に上げた値(`.harness/state/xreview-<org>-cap`)の大きいほう(D7)
- 上限に届いていて次のどちらかなら、`XREVIEW: cap_reached` のヘッダと `FORK 1:` の行(a 上限を上げる(推奨)、b 残りか見せていない変更を PR に書いて進む、c 止める)で人に聞く。新しい上限は人の DECISION の `FORK 1: … 新しい上限 <数>` からだけ読む(D7b)
  - 最後の complete の回の ACTION_REQUIRED が残っている
  - 最後の complete の report の `HEAD:` から、`docs/reports/` 直下の生成された `.md` のほかが変わった(`since: changed`)。`docs/reports/templates/` は skill が読む入力なので数える(cross-review cycle 1 の 1 件目、F7)
- TASK は `xreview-<org>-c<k>` で reviewer 座席に送る。本文には次の行を書く。implementer の driver は、TASK を作るたびに `ralph org status --state-dir <台帳> --json` を jq で読む(F6)
  - `IMPLEMENTER_DRIVER`
  - `XREVIEW_BASE`
  - `CYCLE`
  - `REPORT`
  - `REVIEWER_CODEX`
  - `REVIEWER_CLAUDE`
- 返事ごとの扱い:
  - `pass` は `Cross-review: ran (<相手>)` を控えて sync-docs に進む
  - `skipped` と `incomplete` は、その値を控えて進む。上限には数えない
  - `action_required` は所見を 1 件ずつ決め、直すものを implementer に頼んで手順 8 に戻る
  - `git_changed` は BLOCKED を上げて止まる
- 同じ `<k>` の送り直しは、QUESTION のあとと `git_changed` のあとを合わせて 2 回(`-r2`・`-r3`)まで。`-r3` から先は BLOCKED で止まり、人の DECISION が言うときだけ番号を続けて送る。`git_changed` のあとは `git status --porcelain` が空のときだけ送り直す(F8)
- report は返事を受けるたびに leader がコミットする。sync-docs の前に置き、git の記録から数えられるようにする(D8)
- 停滞の ALERT は gate と同じ扱い(D9)。手順 14 で PR の本文に `Cross-review:` の行を書く(D10)

## reviewer の流れ

- 手順 1 で本文の行を確かめる。足りない行や形の違う行があれば、誰も呼ばずに QUESTION を返す
  - `XREVIEW_BASE` は quote した heredoc で読む。`origin/` で始まり、残りが `-` で始まらないことを確かめる
  - `..`・`@{`・空白・制御文字・引用符・`$`・バッククォート・`\` を含まないことも確かめ、そのうえで `git check-ref-format --branch` が通すときだけ `base: ok` を出す(F7)
  - `base: invalid` のときだけ QUESTION にする。どちらの行も出なければ、heredoc が閉じていないので打ち直す(F8)
- 手順 2 で相手を選ぶ。`pick_reviewer` に `IMPLEMENTER_DRIVER` の値を渡す。pane の `RALPH_PRIMARY_CLI` では選ばない
- 相手の呼び方は次のとおり(D6・D6b)。前後で git の状態を比べ、変わっていれば `git_changed` を返す
  - codex: `-c sandbox_mode=read-only exec review --ignore-rules` で、plugins と apps と MCP を切る
  - claude: Read・Grep・Glob だけの形で、diff を書き出したファイルを渡す
- 所見の P0〜P3 を HIGH/MEDIUM/LOW に読み替えて triage し、report を `docs/reports/review-xreview-<date>-<org>-c<k>.md` に書く(D4)。パスは worktree からの相対で書く
- 返事は `XREVIEW: pass | action_required | skipped | incomplete | git_changed`。incomplete の理由には log の 1 行を、200 字までに縮め、制御文字を除いて入れる

## skill と文書

- `/director`: cross-review 中の reviewer の停滞の ALERT の扱いと、「### cross-review の上限」を足した。上限の節には、件の意味の伝え方、FORK の聞き方、`FORK 1: <記号> <案>` での答え方、a のときの新しい上限の数を書いた
  - RESULT の `CROSS_REVIEW:` の行の報告も書いた
  - main の #271 に合わせ、Codex と push の director は番号つきの選択肢で聞く(F9)
- `/org`: 前提の 2 か所と、終わり方 1 の driver の読み方
- `internal/org/driver/paneenv.go` と `spawn.go`: pane に `RALPH_PRIMARY_CLI` がないときの警告は、計画助言のことだけを言う(S5)
- 文書: 次のものを直した(写しを含む)
  - `config.go` と `templates/base/ralph.toml` のコメント
  - 仕様の 2026-11-01 の注
  - README
  - `docs/reports/README.md`
  - recipe 2 つ(codex 座席の network の注、pane の `RALPH_PRIMARY_CLI`)
  - `model-routing.md`
  - tech-debt の行(雛形の大きさ、テストの穴、座席の中の実行)

## テスト

- `prompts_xreview_leader_test.go`(新規、1,515 行)と `prompts_xreview_reviewer_test.go`(新規、1,354 行)
  - 雛形のコマンドを抜き出し、スタブの git・ralph・codex・claude と bash 3.2・`zsh -f` で走らせる
  - 決まりの文は断片で固定する
  - base の確かめは、本物の git と、いつも 0 で終わる git のスタブの両方で回す(F9)
- `skill_xreview_test.go`(新規): `/director` と `/org` の新しい文を 4 つの写しで固定する
- 変えたテスト: 順の固定(`prompts_test.go`、`prompts_leader_syncdocs_test.go`)、計画助言の codex の行(`prompts_leader_plan_test.go`)、pane の警告(`driver/herdr_test.go`、`internal/cli/org_runtime_env_test.go`)、#271 の文の固定(`skill_split_questions_test.go`)

## 経緯

pipeline は 2 回まわした(上限 2 のまま)。

- cycle 1: self-review、verify、test、本物の CLI での 2 回の確かめの所見を、F1〜F6 で直した。cross-review(codex)は ACTION_REQUIRED 2 件(`since:` が `docs/reports/templates/` を見ない、`XREVIEW_BASE` の許可リストが git の許す名前を断る)で、F7 で直した
- cycle 2:
  - self-review の LOW 4 件は F8 で直した
  - verify の指摘で origin/main を merge で取り込んだ
  - merge で入った #271 のテストが director の skill の古い言い方で落ちたので、F9 で直した
  - cross-review(codex)は所見 0 件
- 各回の所見と直しは計画の Progress checklist と、self-review・verify・test・sync-docs・cross-review の report にある。座席の中での実行と、テストの穴は tech-debt の行に送った
