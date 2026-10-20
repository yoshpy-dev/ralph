# Walkthrough: org-trial(9 段目の試行、FR-12)

- Date: 2026-10-20
- Plan: docs/plans/active/2026-10-20-org-trial.md(/pr で archive に移す)
- Branch: chore/org-trial(PR #243、base a1ce47d8。この walkthrough の前で 20 commits、31 files、+2873/-10)
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 2 files、+25/-10 になる(仕様と tech-debt)。差分の大半は証拠の本文 926 行と、その台帳の写し 21 files。Go、script、skill、rule、template、既定値は変えていない
- 後工程は 1 回(cycle 1)でまわし終えた。self-review・verify・test の指摘は同じ回の中で直し、cross-review の指摘は 0 件だった(下の 7)

```
この PR が何か → 証拠を節ごとに → 17 個の sh block で表を再計算 → 仕様の FR-12 → tech-debt の行 → 計画と後工程の報告 → 確かめていないこと
```

## 1. この PR が何か

- 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 9 段目(FR-12)の記録を足す。1〜8 段目で入れた仕組み(機能ごとの org、横断の見張り、受信箱、計画の承認、gate、merge、director)を、2 つの org で同時に動かした試行の記録で、仕組みは足さない
- 試行は 2026-10-20 の 03:16:42Z から 07:18:53Z。承認した 1 つの分割計画から、機能の org を 2 つ(report-escape と doctor-escape。どちらも leader・implementer・reviewer の 3 席で、すべて claude)、分離した herdr server の中で走らせた。director はこの計画を書いたセッションを `/director` で昇格させたもので、人の役はユーザーが務めた
- 各 org は小さな tech-debt を 1 件直して PR を 1 本ずつ作った。#241(org report の event の `|` の escape)と #242(doctor の merge guarantee の detail の escape)で、どちらも試行の中で `ralph org merge` が merge した。2 本の中身はそれぞれの PR で merge 済みで、この PR には入らない
- 数字は、人の介入が 7 回で 1302 秒、機能の計画の承認待ちが 109 秒と 423 秒、費用が $12.81(座席と watcher。director 自身の分は含まない)。横断の見張りの停止と再開は期待どおりに動いた。org の止め直しでは、新しい leader が雛形を最初からやり直さず、残っていた disband だけを打った
- 既定値 4 つ(`max_orgs` 10、`max_total_seats` 30、`ack_minutes` 10、`resolve_minutes` 60)は変えない。どちらの期限も切れず、どちらの上限にも近づかなかったので、変える理由も、値が正しいという根拠も得られなかった
- 見つけたこと(F1〜F14)の直しはこの PR に入れない。F8(この repo の `scripts/check-sync.sh` が機能の org の report を `ROOT_ONLY` として gate を落とす)は、この PR のあとに小さな PR で直す。ほかは tech-debt の行にした

## 2. 証拠 `docs/evidence/org-trial-2026-10-20.md` を節ごとに

出どころは 2 つ。コミットした台帳の写し(`docs/evidence/org-trial-2026-10-20/`)と、director の作業メモ(試行の間、セッションの scratchpad に置いた。コミットしていない)。台帳の数字は前者から再計算できる。各コマンドの終了コード、出力、pane の画面は後者から来て、Timeline の Source 列の `notes` がそれにあたる。数字を見るなら、Numbers の節と、その下の `sh` block から読むと早い。

- Snapshot(D12、AC10)
  - 写しを取ったのは 07:16Z で、両 org の disband のあと、director の detach と見張りの停止の前。そのため `director.jsonl` に `detached` の行がなく、`watch-alive.json` は `running` のまま
  - コミットしないもの(gate の log 6 本は 1 本あたり約 320 KB、herdr server の log、役の prompt 6 本、setup の script、空の `regate.lock`)と、置き換えた値の規則(pane は `pane-N`、workspace は `ws-N`、Claude Code の session は `session-N`、login 名は `user-1`、per-user の temporary directory は `$TMPDIR/`)を書く。commit SHA、plan の digest、PR 番号、時刻、clone の path(`/private/tmp/ot/` 以下)は元のまま
  - 先頭の label の表(各 label がどの org のどの座席か)は block 1 が manifest から出す
  - org report 2 本(`reports/org-manifest-<org>-2026-10-20.md`)は ralph が書いたままで U+FFFD を含む(F12)。節に「Edit しない」と書いた。mojibake hook は Edit と Write でしか走らないので、CI は落ちない。2 本とも試行の途中で leader が書いたもので、report-escape のは 05:56:21Z の `gate_passed` まで、doctor-escape のは 04:44:51Z までの event しか持たない(029aa209 と a54f7999 で書き方を直した)
  - 見ること: 規則に漏れた生の id がないか。self-review と verify は、`/Users/`、`/home/`、login 名、UUID、pane id の形、認証情報の形を証拠と写しの全ファイルで grep し、0 件だった(計画と報告に `/Users/` が出るのは、grep の対象や結果として書いた行だけ)
- Setup(AC1)
  - clone は f0668004(PR #240 のあとの main)、binary はそこから build。herdr 0.7.5、Claude Code 2.1.291、agmsg 1.5.1。herdr server は `env -i` と、設定・state・socket を `/tmp/ot` に置く変数で分離し、`osascript` は log を取るだけの shim に差し替えた(D2、D3)
  - model の表(block 2)は、座席に指定した model を manifest から出す。split plan は implementer を opus、reviewer を sonnet と決めていたが、leader にその規則が届かず、実際は表のとおりになった(F5)
  - folder-trust dialog が出たのは clone の root で claude を開いたときだけで、director が trust を答えた(S1 の probe。計画が許した範囲)。feature の worktree では出なかった(R1)
  - 見ること: 分離の変数の一覧と、「利用者の herdr server に触れなかった」という記述(Cleanup の末尾と合うか)
- Timeline: 36 行。各行の Source 列が出どころ(`manifest`、`inbox`、`director.jsonl`、`escalations.jsonl`、`interventions.log`、`notes`)を書く。`notes` の行は再計算できない
- Two orgs(AC2): block 3 が両 org の `scope_reserved` に同じ split `trial` と digest `0ffd32661606` が載ることと、重なりの 9370 秒(2 h 36 min 10 s)を出す。block 4 が分割計画と 2 つの機能の計画の `- Approved:` 行を出す。2 本の PR の変更ファイルも書いてある
- Merges(AC3): block 5 の表が 2 本の `pr_merged`(#241 は `auto_merge=false`、#242 は `auto_merge=true`)を出す。そのほかの読みどころは次の 3 つ
  - ruleset(I5、I6): #241 の時点で ruleset に必須の検査がなく、`ralph org merge` は断る状態だった。人が director に直して merge しろと言い、director が `gh api -X PUT` で `check` と `verify` と up-to-date の規則を足した(06:09:55Z)。計画は人が直すとしていたので、これは計画からの逸脱(下の Trial deviations)
  - #242 は #241 の merge で origin/main が動き、`ralph org merge-order` が `rebase` と出した。regate 1 は負荷で test が timeout して失敗し、regate が merge commit を戻した(F10)。`GOFLAGS=-timeout=30m` を付けた regate 2 が通って push し、`auto_merge` の下で director が `--wait-checks 30m` で merge した(checks が pending のあいだ 3m39s 待った)
  - 実際の GitHub に対して `ralph org merge` を走らせたのはこれが初めて。checks がない PR と merge queue は出てこなかった
- Numbers(AC4)
  - Interventions(block 6): 7 回、latency の合計は 1302 秒。時刻の差の合計は 1301 秒で、1 秒の差はミリ秒の丸め(節の冒頭が理由を書く)。人の考えた時間は latency に入っていて、分けていない
  - Approval wait(block 7): report-escape が 109 秒、doctor-escape が 423 秒。後者は、director が F3 を調べている間に質問が 391 秒待ったぶんを含む。leader は図解ページが lint を通り、shot も撮ったと書いたが、PNG を読んだかは確かめておらず、codex の leader も試していない
  - The director's handling of the inbox(block 8、9): 受信箱の 6 件はすべて `delivery: deadline` で、どれも ack の期限(10 分)にも resolve の期限(60 分)にも届かなかった。最長は ack まで 213 秒(e6)、resolve まで 997 秒(e1)。`notified` の event は 0 件
  - Watchdog escalations and notifications(block 10〜12): 見張りが人に直接送った escalation は 30 件(`leader_is_anomaly_subject` が 23、`deadman_timeout` が 7)で、受信箱も director も通っていない(F6)。shim の通知は 49 件で、内訳は見張りの 3 + 11 + 4 + 11 と、座席の Claude Code の 20。doctor-escape の 1 件は通知の記録がなく、理由は分からない(F13)
  - Cost(block 13): 座席が $11.09、watcher が $1.72 で、合わせて $12.81。cache write はすべて 1 時間のもので、その単価で計算した。単価は 2026-10-20 に価格表から読んだ値を `watcher-usage.txt` に書いてある。S1 の事前確認の $0.27 は入れず、director 自身の分は session から分けられないので入れていない
- Gates: block 14 が `gate_passed` と `gate_failed` の 6 件を出す(gate 4 件と regate 2 件。失敗は F8、F9、F10 の 3 件)。log の 6 本はコミットせず、`gate-excerpts.md` が各 log の header、verdict の行、失敗の行を持つ。所要は 28 秒から 1990 秒
- Stop and restart(AC5)
  - Part 2(見張り): 05:48:38Z に `ralph org watch --all --stop`。director の wait は終了コード 4 で終わり、`--dry-run` の TASK の送信は何も記録されずに断られ、05:49:08Z に起動し直した見張りのもとで 05:49:21Z の次の wait が作業を受け付けた(停止から 43 秒)。block 15 が、その窓に manifest・escalation・受信箱・shim の記録が 0 件であることを出す
  - Part 1(org): 計画の D6 は「計画の承認を待つ org を止める」だったが、その時機は F3 の調査の間に過ぎた。そこで doctor-escape の leader を PR を作った直後に止め(06:06:16Z)、同じ機能で start し直した(06:11:16Z)。start は受け付けられ、worktree を使い回した。新しい leader は雛形を最初からやり直さず、git・gh・受信箱を読んで、残っていた disband だけを打った(期待と違う。追加の介入もない)。block 16 と 17 が止めてから disband までの manifest と、受信箱と承認の件数を出す
  - 見ること: 試した形が計画と違うことが、計画の Progress の注、証拠の Part 1 の冒頭、仕様の新しい Open question の 3 か所に書かれているか
- Findings(F1〜F14): 各所見に「Pointers」の行があり、写しの該当の行を指す。グループは次のとおり
  - 隔離の罠: F1(`XDG_CONFIG_HOME` を分けると pane の git と gh が利用者の設定を失う)、F2(login bash の `path_helper` が `/usr/bin` を shim より先にする。`CLAUDE_ENV_FILE` で直した)
  - agmsg と座席: F3(agmsg 1.5.1 が herdr の agent 名を変え、`ralph org send` が `agent_not_found` になる。e1 の resolve が承認の約 15 分後になった)、F4(`agmsg_announce` の timeout が pane と名前を残し、続く 2 回の spawn が `agent_name_taken` で落ちる)、F7(agmsg を polling する leader に `ralph org send` が届かない。49 分)
  - start と見張り: F5(`ralph org start --plan` が分割計画の前書きを leader に渡さない)、F6(見張りの escalation が人へ直接行く)、F13(通知の error が捨てられる)
  - gate: F8(check-sync が `review-` と `org-manifest-` を知らず、gate 1 が `static_failed`)、F9(`TestRegate_FetchFailures/the_fetch_runs_past_the_bound` が負荷で落ちる。次の gate は通った)、F10(regate は `GOFLAGS` がないと Go の 10 分の既定で `internal/cli` と `internal/org` が timeout する)
  - 書き出し: F11(session-end の WIP hook が機能の worktree に commit する)、F12(`truncateForDetails` が多バイト文字を割り、U+FFFD が残る)、F14(event timeline が改行を含む Details で表を崩す。#241 のあとも `|` しか escape しない)
- Trial deviations: 計画と違った 6 つ。R1(trust dialog が出なかった)、D-F3(director が herdr の agent と pane を改名し直し、`AGMSG_SELF_NAME=off` を足した。`/director` skill は director を ralph と gh のコマンドに限るので、これは範囲の外)、F10(regate 2 の `GOFLAGS`)、I6(director が ruleset を変えた。計画の Non-goals は人が変えるとしていた)、intervention の時刻(手書きの初版は 4〜414 秒ずれていたので、director の transcript の時刻に書き直した)、D6 の第 1 部の形
- Cleanup(AC9): 7 段の順(disband、写し、detach、見張りの停止、worktree、herdr server、`rm -rf /tmp/ot`)。利用者の store に残ったものも書いてある。`~/.claude.json` の trust の項 1 つ、`~/.claude/projects/` の session 記録の directory 4 つ、`~/.agents/skills/agmsg/run` の role-session の印 4 つ、利用者の Orca の tab の title。`report-escape` の worktree は、push されなかった WIP commit(F11)があったので `--force-branch` で消した。commit が持っていたのは org report だけで、写しにある
- Not verified: 8 項目(下の 8 に重ねて挙げる)

## 3. 17 個の `sh` block

各 `sh` block は、写しから表の本文か `text` block の本文を作り直す。12 個は直上の表と、5 個は直下の `text` block と比べる。どれも worktree の root から走らせる。

| block | 節 | 比べる先 |
|-------|----|----------|
| 1 | Snapshot | label の表 |
| 2 | Setup | model の表 |
| 3 | Two orgs | org の表(重なりの行まで) |
| 4 | Two orgs | `text` block(`- Approved:` の 3 行) |
| 5 | Merges | merge の表 |
| 6 | Numbers、Interventions | 介入の表(合計の行まで) |
| 7 | Numbers、Approval wait | 承認待ちの表 |
| 8 | Numbers、受信箱の扱い | 受信箱の扱いの表 |
| 9 | Numbers、受信箱の扱い | `text` block(event の数と `notified` 0 件) |
| 10 | Numbers、Watchdog | escalation の表 |
| 11 | Numbers、Watchdog | shim の通知の表 |
| 12 | Numbers、Watchdog | `text` block(`watch.log` の行と receipt の数) |
| 13 | Numbers、Cost | 費用の表(awk) |
| 14 | Gates | gate の表 |
| 15 | Stop and restart、Part 2 | `text` block(停止の窓の件数) |
| 16 | Stop and restart、Part 1 | 止め直しの manifest の表 |
| 17 | Stop and restart、Part 1 | `text` block(受信箱と承認) |

- 道具は jq 1.8.2 と macOS の awk・sort・grep で、GNU の道具では走らせていない
- verify(8e1d5c40)と test(029aa209)が 1 つずつ `sh -c` で走らせ、17 個とも一致した。どちらも期待値を 2 か所変えた写しで同じ比べ方を走らせ、その 2 block だけが不一致になることで、比べ方が差を拾うことを確かめた
- 確かめられるのは、証拠の表とコミットした写しの一致まで。写しが試行の台帳と同じかは、台帳(`/tmp/ot`)が 07:18:53Z に消えたので確かめられない。Cleanup の段 7 は、消す前に `osascript.log`、`manifest.jsonl`、`inbox.jsonl` の 3 つを `cmp` で比べたと書く

## 4. 仕様 `docs/specs/2026-10-07-org-multi-org-director.md`(+5/-3)

- FR-12 の checkbox を付けた。注「9 段目の試行で分かったこと」(1 行の長い箇条)が、試行の形、2 本の PR の merge、4 つの数字、停止と再開の結果、試さなかった形(計画の承認を待つ org を止める)、既定値 4 つを変えない理由を書く
- 注は、FR-12 だけ checkbox を付けた理由を書く。試行の記録と既定の見直しがこの FR のすべてで、9 段目の計画(AC7)がそれを求めたため。FR-1〜FR-11 は今までどおり付けない(verify の V2 を 029aa209 で直した)
- Open questions: 「9 段目の試行の測り方」と「上限と期限の既定は測った根拠がない」の 2 つに答え、注を指す形にした。新しい問いを 1 つ足した。計画の承認を待つ間に止めた org を start し直したとき、新しい leader が何をするか。試行は PR を作ったあとに止めた org だけを start し直した
- 見ること
  - 注の数字(7 回、1302 秒、109 秒と 423 秒、$12.81、43 秒、最長 213 秒と 997 秒、終了コード 4)が証拠の表と同じか
  - 「変えない」の理由が試行の数字の範囲を超えていないか。注は「試行は値が正しいことも示していない」と書いている

## 5. tech-debt `docs/tech-debt/README.md`(+20/-7)

- 閉じた 3 つ
  - org-plan-approval の残りの行の (a)(event timeline の `|`)。#241、merge commit a9e8ea04。`TestBuildOrgReport_EscapesPipesInEventDetails` が名指しで入る
  - org-merge-guarantee の残りの行の (h)(doctor の Detail の escape)。#242、merge commit a1ce47d8。`TestMergeGuaranteeResult_EscapesControlCharacters`。同じ行の総覧の 1 文目からも (h) を外した
  - ruleset の行(必須の検査がなく `ralph org merge` が全 branch を断る)。行の上に `<!-- RESOLVED ... -->` を置き、Debt と Trigger を打ち消した。director が人の言葉で `check` と `verify` と up-to-date の規則を足した(I6)
- 書き足した既存の行
  - leader が図解ページを作って確かめられるかの行。claude の leader は自己申告までで、PNG を読んだかと codex の leader は確かめていないので、開いたまま。Trigger もその残りに書き換えた
  - `ralph org merge` の 3 つの未試行の読みの行。試行が実際の GitHub で当たったもの(checks の名前、pending から pass、`current_user_can_bypass: never`)を足した。3 つの形そのものには当たらなかった
  - F9 の `TestRegate_FetchFailures` の行、F10 の `run_tests` に `-timeout` がない行(Impact と Trigger にも gate と regate を足した)
- 新しい 12 行: 隔離した herdr の罠(F1 と F2。recipe がない)、F3、F4、F5、F6、F7、F8、F11、F12、F13、F14、R1(folder trust)。F9 と F10 は上の既存の 2 行に入れたので、F1〜F14 と R1 のすべてに行がある
- 見ること
  - 欄の数。未エスケープの `|` で数えて 6 個(5 欄)。形の崩れた 22、32、278 行は origin/main と同じ
  - 行は code を関数名で指す(`file:line` は sync-docs で外した)。行番号で別の行を指す文もない(self-review の L3)
  - 計画の参照は `docs/plans/active/...` で、`/pr` の `scripts/archive-plan.sh` が archive に書き換える
  - F8 の行は直し方の選択(除外を足す、reviewer の report の名前を決める、`ralph org report` の書き出し先を変える)が開いていることを書く。直しは別の PR

## 6. 計画 `docs/plans/active/2026-10-20-org-trial.md`

- 519 行。Status は Approved で、digest は `5b82ee1e30a1`。承認のあとに書いたのは Progress checklist の注だけで、digest は前後で変わらない(`plan-visual.sh digest` で確かめた)。AC1〜AC10 に印が付き、`PR created` だけが開いている
- 承認の前: Codex の計画助言(gpt-6-astra、xhigh)の 3 件を直し、consult の判断で D6 の第 1 部(org を止めて start し直す)を足した
- 本文を変えずに記録した逸脱は、Progress の「Deviations from the body」にある。D6 の適応、F8 を別の PR に送ったこと、`interventions.log` の書き直し、R1・D-F3・F10、AC8 の行番号(297 が 298 にずれた)、I6
- 見ること: 逸脱の一覧が、証拠の Trial deviations と食い違わないか

## 7. 後工程の報告

- self-review(`docs/reports/self-review-2026-10-20-org-trial.md`): Merge: yes
  - MEDIUM 1(M1: 図解ページの行の Debt 欄が、試行の前の文と結果を並べて食い違う)、LOW 5(L1 計画の注に I6 がない、L2 総覧に (h) が残る、L3 行番号の参照、L4 `run_tests` の行の Impact と Trigger、L5 証拠の F6 と F13 の取り違え)、INFO 3
  - M1、L1〜L5、I1、I2 は a288d23b で直した。再レビュー(ccd0c9e0)の LOW 1(N1、図解ページの行の Trigger 欄)は 8e1d5c40 で直した。I3(F9 の行の Trigger の文言)は据え置いた
- verify(`docs/reports/verify-2026-10-20-org-trial.md`): pass
  - LOW 2(V1 org report が書かれた時機、V2 FR-12 の checkbox の理由)は 029aa209 で直した。INFO の V3(`gate-excerpts.md` の行末の空白。元の log の空行の写し)などは残した
- test(`docs/reports/test-2026-10-20-org-trial.md`): pass
  - `run-test.sh` は docs だけの差分として `tests/test-*.sh` の 41 本(639 秒)を走らせ、すべて通った。Go のテストは走らない
  - MEDIUM 1(T1: 029aa209 で足した文の、report-escape の org report の終わりの時刻が誤り)は a54f7999 で直した
- sync-docs(`docs/reports/sync-docs-2026-10-20-org-trial.md`): pass。tech-debt を 3 点直した(code の行番号 2 つを名前に、F11 の行の Related に古い 2 行を足す)
- cross-review(`docs/reports/cross-review-triage-org-trial.md`): codex の指摘は 0 件。codex の sandbox は `mktemp` を断られ `check-sync.sh` を走らせられなかった。verifier、tester、doc-maintainer が HEAD で走らせ、exit 0 だった
- insight event(`docs/insights/events/2026-10-20-org-trial.jsonl`): self_review から cross_review までの 5 行で、どれも cycle 1
- 見ること: 029aa209 と a54f7999 は self-review のあとに入れた記録の直しで、self-review は読んでいない。cross-review はこの 2 つを含む head で走った。記録だけの直しの例外は使っていない(cycle 1 のまま)

## 8. 確かめていないこと

- 写しが試行の台帳と同じか。台帳は消えていて、`cmp` で比べたのは 3 ファイルだけ
- director のメモと transcript にだけある値。各コマンドの終了コード、Cleanup の各段の `pgrep` と `lsof` の結果、load average、Timeline の `notes` の時刻、`--wait-checks` の 3m39s。写しの跡(停止の窓の 0 件、`watch.log` の開始 2 回など)とは矛盾しない
- 費用。使用量の和は Claude Code の session 記録(コミットしていない)から足した値で、verify と test は `seat-usage.tsv` と `watcher-usage.txt` からの計算だけを確かめた。価格表のページは開き直していない。director 自身の費用は含まない
- `interventions.log` の時刻(director の transcript から読んだ値)と、session の label と session 記録の対応(Setup と Snapshot の記述)
- 本文の数のうち、block の隣にないもの。test は 19 件だけ照らした
- GNU の jq・awk での `sh` block の実行
- 2 本の PR のコードと CI。それぞれの PR と gate の記録に頼り、この PR では Go のテストを走らせていない(docs だけの差分)
- 試していない形。計画の承認を待つ org を止めて start し直す、codex の leader と座席、leader が図解の PNG を読んだか、checks のない PR と merge queue の ruleset を `ralph org merge` に通す
- 原因を調べていないもの。F4 の `agmsg_announce` の timeout、F9 の flake、F13 の通知 1 件
- 推測や伝聞のままの記述。Orca の tab の title(F3)、`CLAUDE_ENV_FILE` が session の始めに 1 度だけ読まれること(2 つのコマンドで確かめた)、20 件の通知が利用者の Notification hook から来たこと(メモのみ。hook は読んでいない)
- sync-docs の突き合わせは grep と該当箇所の読みで、文書の全文は読み直していない。F8 に関わる `/org` skill の手順も読み直していない
- 直していないこと。F8 は別の PR で直す。そのほかの所見は tech-debt の行に残した

写しのうち org report の 2 つは ralph が書いたままで U+FFFD を含む。Edit や Write で直さない。
