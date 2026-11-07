# Walkthrough: org の座席を API エラーから続ける(org-seat-resume)

- 計画: `docs/plans/archive/2026-11-07-org-seat-resume.md`(PR の作成時に archive へ移す)
- 仕様: `docs/specs/2026-11-07-org-seat-resume.md`
- 差分: `git diff 11239c8d...HEAD`(95 ファイル、約 +8,600 / −230。うちテストを除く Go は約 +1,900、Go のテストは約 +4,000)

差分の大半はテストと文書なので、次の順に読むと仕組みが追いやすい。

## 1. 画面の判定(読む価値が一番高い)

- `internal/org/resume_screens.go`: pane の末尾の text を、止まった・Enter だけ・待ち中・判定できない、の 4 つに分ける。冒頭のコメントが画面のモデル(まとまりの区切り、入力欄の見つけ方、footer、判定の順)を手順として書いている。
  - 文の一覧は `resumeSentenceTable` に driver ごとにあり、各文に出どころ(実機の版、ドキュメント、バイナリの文字列)が付く。
  - 止まった文は、まとまりが「エラーの印(claude `⏺`・`⎿`、codex `■`・`⚠`)、空白、文」の順で始まるときだけ一致する(R2 `306ee58f`、R6 `6993faca`)。
  - 入力欄は、すべての行が空のときだけ空とする(R2)。codex の footer は形が既知のときだけ footer とする(R3 `3714f6d5`、R4 `6ee7c1dd`)。
- `internal/org/resume_screens_test.go` と `internal/org/testdata/resume/`: 実機で取った画面(記録 `docs/evidence/org-seat-resume-probe-2026-11-07.md`、`org-seat-resume-live-2026-11-07.md`)と、ドキュメントの文から作った画面の表。cross-review で見つかった誤判定は、それぞれ fixture になっている。

## 2. 打つ道とロック

- `internal/org/resume.go`: `(*Org).Resume`。対象(動いている座席と herdr の中の headless の director)、順(leader 以外 → leader → director)、自分の pane を飛ばす、`herdr agent get` の状態、画面の判定、ロックの中の読み直し、Enter の前の入力欄の確かめ、manifest の `resumed`。
- `internal/org/pane_lock.go`: pane ごとのロック。冒頭のコメントに、誰がいつ取るかの一覧がある。
- `internal/org/verbs.go`: `Send` の打ち方を `typeAndSubmit` に切り出して resume と共有した。`Send` は idle/done の待ちのあとにロックを取り、ロックの中で idle/done を待ち直し(R1 `0aa8e274`)、打つ直前に計画の承認を確かめ直す(R5 `323e94d5`)。それ以外の `Send` の挙動とエラーの文は変えていない。
- `internal/org/director_wake.go`: director の push の wake も同じロックを取る(R1)。herdr の director に設定された `wake_cmd` もロックの中で走らせる(R5)。
- テスト: `resume_test.go`、`pane_lock_test.go`、`pane_lock_approval_test.go`、`director_wake_lock_test.go`。ロックの待ちと割り込みの順は偽物の herdr で決めて確かめている。

## 3. CLI

- `internal/cli/org_resume.go`: `ralph org resume [--org-id] [--dry-run]`。座席ごとの 1 行と要約の行、submit を確かめられなかったときの `note:` の行、終了コード(送信の失敗か記録の失敗で 1)。pane の文字は表示用の関数を通して出す。
- `internal/cli/org.go`: 登録と `--org-id` の説明。`send --help` にロックの断りを足した。

## 4. 予防の設定(claude の座席の再試行)

- `internal/config/config.go`: `[org].api_wait_minutes`(既定 30、0 で無効、負は断る)。`templates/base/ralph.toml` に既定値とコメント。
- `internal/org/driver/paneenv.go`: 分から 3 つの変数(`CLAUDE_CODE_RETRY_WATCHDOG=1`、`..._MAX_WAIT_MS`、`CLAUDE_CODE_MAX_RETRIES`)を作る。回数の換算式は `docs/evidence/org-seat-resume-retries-2026-11-07.md` の実測から。
- `internal/org/spawn.go`、`director_start.go`、`director_cmd.go`: driver が claude の座席と headless の director にだけ足す。codex には足さない。

## 5. 座席と director への指示

- `internal/org/prompts/{leader,implementer,reviewer}.md`: pane に直接入った「`ralph org resume` からの続行の指示です。」で始まる文を、止まったところから続ける指示として扱う 1 行。
- `.claude/skills/director/SKILL.md` の `## 止まった座席を続ける`: 人が「続けて」と言ったときだけ resume を打ち、座席ごとの結果を報告する。ALERT の件で画面が API エラーなら、人に伝えて自分では打たない。`.claude/skills/org/SKILL.md` の動詞リファレンスに `resume` の行。どちらも `.agents/` と `templates/base/` の写しがある。

## 6. 記録と台帳

- `docs/evidence/`: 実機の画面(probe)、再試行の回数の実測(retries)、分離した herdr での確認(live)。live は S11・R2・R6 より前の binary で走らせた。
- `docs/reports/`: self-review、verify、test、sync-docs は cycle 3 の版。cross-review の仕分けは cycle 3(指摘 0)と、cycle 1・2 の記録。
- `docs/tech-debt/README.md`: 直さずに残した穴(cycle 1 の self-review の L3〜L9、codex の footer の形、テストの穴、AC-10 の残りなど)。

## レビューで見てほしいところ

1. `resume_screens.go` の判定が、知らない画面・承認画面・下書きのある入力欄で必ず「打たない」側に倒れるか。
2. `verbs.go` の `Send` の変更が、ロックと待ち直しと承認の確かめ以外の挙動を変えていないか(既存の `Send` のテストは期待値の数え直しだけで、緩めた assert はない)。
3. claude の座席の待ち(既定 30 分)の間、見張りの停滞の ALERT が出ないこと(記録の live の AC-15)を、運用として受け入れられるか。
