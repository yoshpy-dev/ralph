# Walkthrough: org-director-wait-bound(8 段目の後半の 2 本目の残り、8b-2c)

- Date: 2026-10-19
- Plan: docs/plans/active/2026-10-19-org-director-wait-bound.md(/pr で archive に移す)
- Branch: fix/org-director-wait-bound(base 3a803a87。この walkthrough を書いた時点で 20 commits、19 files、+1928/-67)
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 7 files になる。内訳は次のとおり
  - `/director` skill の 4 面(各 +38/-16)
  - テスト `internal/org/director_skill_sync_test.go`(+484/-1。`func Test` は 2 本から 5 本になった)
  - 仕様の注 1 行と tech-debt の 2 行
- Go の本番のコードと `ralph org wait` の既定は変えていない。後工程は 1 回でまわし終えた。self-review・verify・test の指摘は同じ回の中で直し、cross-review は 0 件だった(下の 6)

## 1. なぜ直したか(実験、e9cd5ea9)

- `docs/evidence/director-wait-monitor-2026-10-19.md` の内容
  - Claude Code 2.1.291 の Monitor で始めた処理は、`/exit` の既定、「Move to background and exit」、pane を閉じたとき、Monitor の期限のどれでも SIGTERM で止まった。残るのは claude の SIGKILL のときだけだった
  - Monitor は `timeout_ms`(既定 5 分、最大 30 分)が過ぎた処理に SIGTERM を送る
- 3a803a87 の skill は、Claude Code に `--timeout-ms 0`(時間切れなし)の wait を Monitor で見張らせ、signal での終わりを失敗(打ち直さない)としていた。そのため Monitor を選んだ director は、30 分以内に受信箱を待つのをやめていた
- 8b-2b の実験(`docs/evidence/director-wait-session-end-2026-10-19.md`)では、Bash の `run_in_background` の処理は Move to background のあとも残っていた。これと合わせて、Monitor を主の手にした(計画の D1)

## 2. Claude Code の箇条(S1、5aea657f。直し dc98d5ec)

- `### wait で待つ` の Claude Code の箇条
  - `ralph org wait --inbox --director d1 --timeout-ms 1500000 2>&1` を Monitor で見張り、Monitor の `timeout_ms` に `1800000` を渡す
  - 25 分の区切りが Monitor の 30 分より先に来るので、wait が自分の時間切れで返る
- 時間切れ(終了コード 1 で、stderr が `no open inbox item arrived within` を含む)は、Codex と同じく打ち直す(D2)
- Bash の `run_in_background` は代わりの手として残した。その wait は Move to background のあとも、claude の SIGKILL のあとも走り続け(25 分のうちに終わる)、`note:` の行は wait が終わるまで届かない、と書いた
- Monitor の wait が止まる場面は、確かめた 4 つに絞って書いた(self-review の L1)
- Codex の箇条(`--timeout-ms 300000`)と headless の雛形 `internal/org/prompts/director.md` は変えていない。雛形は「skill のとおり」と書くだけで、数を持たない

## 3. 終了コードと Monitor の期限切れの例外(D3。直し dc98d5ec、de2b05d1)

- 終了コードの `- 1:` の箇条: 時間切れの文があれば、Claude Code でも Codex でも打ち直す
- 失敗の段落の前の文: 例外を「Monitor が期限切れを知らせた wait の signal での終わり」と名指す
- 例外の段落(de2b05d1 の形)
  - wait が signal で終わったのを見た時点で、その wait を見張っていた Monitor の「Monitor expired」の知らせが届いていれば(同じ通知に載っているときも含む)、Monitor が止めたので打ち直す。それからは `timeout_ms` に `1800000` を渡す
  - 届いていなければ失敗として扱う。あとからその Monitor の知らせが届いたら、そこで打ち直し、失敗とした終わりは期限切れだったと人に伝える
  - ほかの signal での終わり、panic、時間切れの文のない終了コード 1 は、失敗のままで打ち直さない
- この段落は 2 回直した
  - self-review の L2: その wait の Monitor に結び、順を外した
  - 再レビューの N1: 終わりを先に見たときにも決められる形にした
- 計画の D3 と R1 の「知らせのあと」の言い方は、承認の digest を保つため本文では直さず、Progress checklist に書いた

## 4. テスト(`internal/org/director_skill_sync_test.go`)

- 数を固定するテスト(S1。dc98d5ec で広げた)
  - `TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap` が、4 つの写しのそれぞれに `directorSkillWaitBoundProblems` を当てる
  - Claude Code の箇条の `--director d1 --timeout-ms N` と「`timeout_ms` に `M`」がちょうど 1 つずつで、0 < N < M ≤ 1800000(`directorSkillMonitorMaxTimeoutMs`)であること
  - 小節 `### wait で待つ` の中の `timeout_ms` に `X` がどれも M であること
  - N が分で割り切れ、「K 分」の K が N/60000・既定 5・最大 30 のどれかであること
  - 全文に `--timeout-ms 0` がないこと
- `TestDirectorSkillWaitBoundProblems` は、この関数を小さな文で試す表のテスト(21 件)
- 決まりを固定するテスト(S6、9eb98897)
  - `TestDirectorSkill_PinsTheClaudeCodeWaitRules` が、AC2 と AC3 の文の断片 12 個を `withoutSpaces` で比べる
  - 5 個は Claude Code の箇条の中で探す: 時間切れの打ち直し、Bash の代わりの手と残り方、`note:` の届き方
  - 7 個は小節の中で探す: `1:` の打ち直し、例外の 3 つの枝
  - test の T1(数だけを固定していた)を埋めた
- mutant は、`git archive HEAD` の写しで打った。追跡しているファイルは変えていない
  - 数の側は 22 回、決まりの側は 22 回で、すべて赤になった
  - 記録: `docs/evidence/test-2026-10-19-org-director-wait-bound-mutation.txt`、`...-rules-mutation.txt`

## 5. 実機の確認(S2、f9c60c0c。記録の直し ffc56a2e、de2b05d1)

- 用意: 分離した headless の herdr server の pane で claude を動かした。台帳は `--state-dir` で分け、binary はこのブランチから作った
- (b)〜(d) で wait 1 回ごとの終わり方を見た
  - (b) 前の形の wait が Monitor の期限で `end=signal` になった
  - (c) `/exit` で止まった
  - (d) SIGKILL のあとに残った 40 秒の wait が、区切りで `end=timeout` を書いて終わった
- (a)(e)(f)(g) では、skill の節を試験の数字にした指示を読むセッションで、人の入力なしに次を確かめた
  - (e) 時間切れを 3 回続けて打ち直し、件を返して ack した
  - (f) Monitor の期限切れのあとに打ち直した
  - (g) 外からの SIGTERM では打ち直さず、人に伝えた
- 最初のセッションは、指示を区切りのある確認と読み、5 回目の時間切れのあとに自分で止まった。2 本目には「この役に終わりはない」の 1 行を足した
- 確かめた範囲
  - セッションが読んだのは 5aea657f の文で、そのあとの文(dc98d5ec、de2b05d1)では打ち直していない
  - PR の前の consult は、(f)(g) で director が取る手が変わらないので、打ち直さなくてよいと判定した

## 6. 記録と後工程

- tech-debt の B の行(248 行)は閉じずに書き換えた(D4)。書いたことは次のとおり
  - runner ごとの残り方
  - 25 分と 300 秒の区切り
  - 残った wait が返した件の行き先
  - 親の pid で見分けない理由
  - 確かめていないこと
- 247 行は、終わった 8b-2c を Trigger から落とした(8324476b)
- 仕様の FR-10 に 8b-2c の注を足した(98 行)。8b-1 の注の次に置いている
- 後工程(cycle 1)
  - self-review(060b9650)は Merge: yes。MEDIUM 1 件と LOW 6 件を S4(dc98d5ec、ffc56a2e)で直した。直しの再レビュー(8796abec)で出た LOW の N1 は S5(de2b05d1)で直した
  - verify(024e2ba6)は pass で、INFO 2 件
  - test(35280638)は pass。LOW の T1 は S6(9eb98897)で直した
  - sync-docs(b7e16c60)は、tech-debt の 248 行に 1 文足した
  - cross-review(codex、5b53b5c3)は 0 件
- run-verify は 83c0d34d で rc 0 だった。9eb98897 でテストが変わったので、b7e16c60 で打ち直し、こちらも rc 0 だった(internal/cli 353.7 s、internal/org 461.2 s)
