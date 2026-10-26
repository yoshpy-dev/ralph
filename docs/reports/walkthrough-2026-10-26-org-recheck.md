# Walkthrough: org-recheck(release の前の通しの確かめの記録)

- Date: 2026-10-26
- Plan: docs/plans/active/2026-10-24-org-recheck.md(/pr で archive に移す)
- PR: ブランチ `docs/org-recheck` の PR(番号はまだない。merge のあとは `gh pr list --state merged --head docs/org-recheck` で引ける)
- Branch: docs/org-recheck(この walkthrough の前の HEAD 109fb3f9。20 commits(merge commit 1 つを含む)、54 files、+2979/-51)
- Base: 98d38391(origin/main、「Merge pull request #256」)。189cd55d で取り込み済み
- 読む範囲: evidence 本文 `docs/evidence/org-recheck-2026-10-24.md`(+1246)と伏せ字の写し `docs/evidence/org-recheck-2026-10-24/`(39 files、+863)。手順書 `docs/recipes/isolated-herdr-server.md` と `templates/base/` の同じ写し(どちらも +108/-15)。記録は tech-debt(+18/-7)、仕様(+6/-2)、根拠の表(+35/-12)、試行の evidence(+14)。残りは計画(+159)、後工程の報告 5 本(cross-review は triage。+417)、insight event(2 files、+5)。Go とシェルの変更はない
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件。記録だけの修正の例外は使っていない(triage は 109fb3f9 の 1 commit だけで、追記の commit はない)

```
何を確かめたか → 計画 → evidence と写し(S4) → 手順書(S5) → 記録(S6) → 後工程と、その中の直し → 設計の判断 → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- release の前の通しの確かめの記録で、コードの変更はない。前回の試行(2026-10-20、binary は f0668004)は所見の直しの前の binary で回した。今回は直し(#244〜#254)をすべて含む main の ffb3e271 から build した binary で、director から merge までを通した
- 実行(S1〜S3、S7): 分離した herdr(`/tmp/ralph-iso`)の上で、このセッションを `/director` で昇格させた director、人をユーザーとして、承認済みの分割計画 1 つから機能の org 2 つ(backfill-na、hooks-matcher)を同時に立てた。機能の PR #255 と #256 を `ralph org merge` で main に merge した(536271a2、98d38391)。この PR の base は 98d38391。実行は 2026-10-24T09:40Z(S1 の開始)から 2026-10-25T14:24Z(最後の写し)で、`/tmp/ralph-iso` は消した
- 測ったこと(evidence の表。数は `sh` ブロックで写しから出し直せる):
  - 2 つの org が並んだのは 46 min 13 s。最初の start から最後の disband まで 52 min 53 s。そのあとの、人の返事を待った時間(I3)が 10 h 42 min
  - gate は 910 s と 846 s、regate は 862 s で、3 回とも 1 回目で通った。試行は 6 回のうち 3 回が落ちた
  - 受信箱の 12 件(QUESTION 2、ALERT 8、RESULT 2)は、どれも ack の期限(10 分)にも resolve の期限(60 分)にも届かず、director が処理した。人への通知はなく、写しに `escalations.jsonl` はない
  - 介入は 5 回、98237 s。そのうち 2 回は不在(I0a が 16 h 27 min、I3 が 10 h 42 min)で、残り 3 回は 479 s。費用は座席と watcher で $10.40(試行は $12.81)
- 所見の直しの効き(Fix effects の 12 行): F3、F5、F6、F8・F9・G1 は効いた。F7 は director から leader への send 10 件が 1 度も切れなかった(leader が Monitor で待ったかは未確認)。F11 は push したコミットについて効いた。F13 は director が処理した件に通知が出ないことまで。F4 は spawn の失敗が起きず、F10 は wrapper の `GOFLAGS` のために Go の verifier の既定が使われず、どちらも確かめられていない。開いたまま残る行(leader への ALERT の時間切れ、watcher の時間切れ)は、どちらも 0 回
- 新しい所見は RC1〜RC12(試行の F1〜F14 とは別)。扱いは次の表のとおりで、evidence の Findings(`:912`)と計画の Progress にも書いてある。

  | 所見 | 内容 | 扱い |
  |------|------|------|
  | RC1 | 座席の Claude Code が私用の agmsg の写しでなく利用者の agmsg の skill を使い、RESULT を返す 4 席のうち 3 席の RESULT が agmsg で leader に届かなかった | tech-debt の行。手順書に、feature org では写しを使わないと書いた |
  | RC2 | leader の雛形の spawn の手順が `--scope` を書かず、両方の org の implementer と reviewer の最初の spawn 4 回が断られた | tech-debt の行。FR-4 と受け入れ条件 5 を `[ ]` に戻した |
  | RC3 | 機能の org の 3 席は worktree を共有するので、予約の中の編集でも見張りの `scope_change` と `stall` が出て、受信箱 12 件のうち 8 件がそれだった | tech-debt の行 |
  | RC4 | 分割計画の前文の「予約の外は編集しない」が、leader の雛形が予約の外に書かせる計画と report に文字どおりには合わない。hooks-matcher の leader が計画の 2 行を `sed -i` で書き換えた(notes による) | tech-debt の行 |
  | RC5 | 2 本の PR のタイトル(と #255 の本文)が英語で、`/pr` skill の日本語の規則に沿わなかった | tech-debt の行 |
  | RC6 | `/director` skill の send のあとの確かめ(pane で DECISION を読む)が、承認の DECISION 2 件でできなかった。`ralph org read` は約 22 行で、入力が画面の外へ流れた | tech-debt の行 |
  | RC7 | 手順書に、人が分離した server に入る打ち方がなかった。人の端末で trust に答える道は、画面が出なかった | 手順書で直した(行なし。人の端末で画面が出なかった理由は不明) |
  | RC8 | 計画の D8 の probe のコマンドで、`--allowedTools` が依頼を tool の名前と読んだ | 計画のコマンドの誤りで、行なし。手順書には語順を書いた |
  | RC9 | director の wait が、detach が先だったのに終了コード 4 で終わった(コードと時刻からの推論) | tech-debt の行 |
  | RC10 | 断られた spawn 4 回が `honored=false` の receipt を残し、`ralph insights` が 4 席を honored でないと数える(コードからの読み) | tech-debt の行 |
  | RC11 | watcher の `claude -p` が利用者の hook を読み、13 回のうち 12 回で通知の shim を打った。試行の「Claude Code」の通知 20 行のうち 18 行も同じ疑い | tech-debt の行。試行の evidence に Correction を足した |
  | RC12 | leader の status line が effort low だった(おそらく利用者の設定) | 利用者の設定で、行なし |

- 戻し方: この PR を revert する。#255 と #256 は別の PR で、それぞれ revert できる

## 2. 読む順(commit ごと)

1. f561aa80 計画
   - Scope、Non-goals、Design decisions(D1〜D12)、AC1〜AC9 を読む。承認の digest は 1e218f3d7907 で、HEAD で `./scripts/plan-visual.sh digest` が返した値と同じ。`Approved` の行は最初の commit からあり、本文は変えていない(`git diff f561aa80 HEAD` で動いているのは AC の checkbox と Progress だけ)
   - Visual review は None(今ある仕組みを動かすだけで、何も足さないため)
   - 見ること: Non-goals。停止と再開(前回の D6)、開いたまま残る行の直し、codex の座席と director、headless の director、既定値(`max_orgs` など)の見直し、release、題材の 2 つの直しの中身(leader と座席が書く。director は実装しない)
2. bcb9db16、77badb04、f7ad239d S4 evidence と写し(AC1〜AC5、AC9)
   - bcb9db16 が evidence(このときは 1226 行)と伏せ字の写し 36 ファイル。77badb04 が agmsg の未読の列を足し、probe の文を弱めた(director が consult の指摘で直した分も同じ commit)。f7ad239d は、利用者の global gitignore の `*.log` で漏れた写し 3 本(`interventions.log`、`osascript.log`、`watch.log`)を `git add -f` で足した(前回の試行の写しと同じ扱い)
   - HEAD の evidence は 1246 行。節は Snapshot(`:32`)、Setup(`:157`。手順書の節ごとの記録は `:250`)、Timeline(`:336`)、Two orgs(`:380`)、Fix effects(`:424`)、Merges(`:538`)、Numbers(`:664`)、Gates(`:893`)、Findings(`:912`)、Comparison with the 2026-10-20 trial(`:1137`)、Trial deviations(`:1162`)、Cleanup(`:1186`)、Not verified(`:1226`)
   - 表の直上か直下に `sh` ブロックが 29 個ある。写しから表を出し直す `jq`・`awk`・`grep`・`git log` で、リポジトリの root から打つ
   - 伏せ字: pane・workspace・tab・terminal の id は `pane-N` などの label に、Claude Code の session id は `session-N` に、`$HOME` は `~` に、OS のログイン名は `user-1` にした(Snapshot の Sanitizing)
   - 見ること: Fix effects の判定の列(起きなかった項目を「効いた」にしていないこと)、Trial deviations(計画から逸れた 8 点)、Not verified
3. 29b54de0、59779049 S5 手順書(AC7)
   - 29b54de0 が root と `templates/base/` の 2 写しに +190/-30(2 写しの合計)、59779049 が 2 写しに +8 ずつ(専用の clone の機能の worktree の片付け。S5 の報告の範囲外の指摘を director が inline で足した)。a688fcf0 で一部を組み直した。HEAD の 2 写しは 579 行で、`cmp` が一致し、base との差分は各 +108/-15(12 か所)
   - 冒頭(`:17`)と「What this recipe does not cover」(`:572`)の「通しで打っていない」とする 2 文を、今回の実行を書く文に替えた
   - 実行で分かったことを手順に足した。専用の clone に `.git/info` がなければ `mkdir -p .git/info`(`:82`)。agmsg の私用の写しは wrapper の `RALPH_ORG_AGMSG_HOME` で渡す(`:174`、D2)。写しは ralph 自身の agmsg 呼び出しにしか届かず、座席の Claude Code は利用者の agmsg を使うので、座席が leader に報告する org では写しを使わない(`:291`、RC1)。shim は利用者の hook の `osascript` も受ける(`:324`、RC11)。`pgrep -fl 'herdr server'` は単独の command で打つ(`:344`)。trust は人が `/tmp/ralph-iso/run herdr` で捨ての pane に入って答え、自分の端末で `claude` を起動する道は効かなかった(`:386-397`、RC7)。キーを打たない probe(`claude -p`。依頼は `-p` の直後。`:407`、RC8)。片付けに機能の worktree(`:529`)
   - 見ること: 2 写しが同じバイト列であること(`cmp`、`scripts/check-sync.sh` の IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0)。scaffold 先に届く写しなので、source の path(`internal/` の下)を書かないこと(self-review の F4。a688fcf0 で直した)
4. 59964c5c S6 記録(AC6)
   - tech-debt(`docs/tech-debt/README.md`): 試行の所見の 2 行を RESOLVED にした(注は `:126` と `:146`)。`checkCodexEffectiveConfig` の warn 精度の 2 件は #256、`ralph insights` の backfill が `Reviewer status:` を読まない件は #255。新しい行は 9 つ(`:363`〜`:371`。RC1〜RC6、RC9〜RC11)。開いたまま残る 2 行(`:345` の leader への ALERT、`:346` の watcher の時間切れ)には今回の回数(0 回)を書き足し、閉じていない。手順書の evidence 引用の行(`:362`)の数を 5 から 6 にした
   - 仕様(`docs/specs/2026-10-07-org-multi-org-director.md`): Status(`:35`)に今回の確かめを書き、FR-4(`:78`)と受け入れ条件 5(`:170`)を `[x]` から `[ ]` に戻して「印(2026-10-25)」(`:87`、`:172`)を足した。`[ ]` は 3 から 5 になった(FR-3、FR-4、FR-10、ツール非依存、受け入れ条件 5。`grep -c` で base 3、HEAD 5)
   - 根拠の表(`docs/evidence/org-spec-evidence-2026-10-21.md`): 「通しの確かめ(2026-10-24)」の節(`:44`)に RC1〜RC12 の表(`:52-63`)。判定の数は `[x]` が 26、`[ ]` が 5(`:105`)。「確かめていないこと」の 8 項目(F4、F11、F5、F6・F13、F7、手順書、leader への ALERT、director)に、書き足すか書き換えた
   - 試行の evidence(`docs/evidence/org-trial-2026-10-20.md:475`): 「Claude Code の通知は座席のもの」とした文の下に、日付つきの Correction を足した。古い文は消さない。RC11 が出どころの記述を覆すため。計画の Affected areas にはなく、consult の指摘で足した
   - 見ること: 仕様の `[ ]` の 2 項目に「印(2026-10-25)」が付いていること
5. ab304ce7、189cd55d
   - ab304ce7 は、計画の Progress に S1〜S7 の commit と所見の扱いを書く(+6/-1)。189cd55d は origin/main(98d38391)を取り込む merge commit。branch は ffb3e271 から切ってあり、実行のあいだに #255 と #256 が main に入ったので、15 commits、10 files(2 本の機能の PR の変更)が入った。これ以降の差分の base は 98d38391
6. 後工程と、その中の直し
   - 2b399912 self-review(Merge yes、LOW 6・INFO 4)。a688fcf0(5 files、+120/-104)で 10 件とも直し、7f004c11 で報告に追記した。F1: RC2 の「どの機能の org でも毎回」を「雛形のとおりに spawn を打った leader は断られる」にした(試行の leader は雛形になくても `--scope` を付け、断りは 0 回)。F2: F3 の行の読み先から command B を外した(command B は失敗した send を見ない)。F3: 「watcher の 1 回ごとに通知」を「多く(13 回のうち 12 回)」にした。F4: 手順書から `internal/` の path を外し、裸の RC 番号に evidence の名前を添えた。F5: `--config` の注を囲みの直後に動かした。F6: evidence の冒頭の、org が merge まで動いたと読める文を直した。N1〜N4 は折り返し、`--allowedTools Bash` の注意、`path_helper` の文、仕様の F11 の但し書き
   - 0c0a6a6e verify(pass、LOW 1・INFO 4)。LOW-1(根拠の表の RC11 の行に残った「1 回ごとに通知」)は 2d223b19 で director が inline で直した。verify は evidence の `sh` ブロック 29 個を打ち直して表と全部合うことを見て(対照に期待値を 2 か所変えた写しでは 2 個だけが不一致になった)、`scripts/archive-plan.sh` を scratchpad の写しに打って archive のあとの参照を調べた
   - f5bc474b が AC1〜AC9 と、Review・Verification の artifact に印を付けた。124ac68e test(pass、HEAD f5bc474b)。`./scripts/run-test.sh` が rc 0(697 s、shell 44 本、scope は docs_only)、`go test -count=1 -timeout 30m ./...` が go1.26.0 と go1.25.8 でどちらも rc 0(493 s と 396 s、8 パッケージ)。#255 と #256 が足したテスト 5 本(18 の `--- PASS`)も通った。変わった文書を読む Go のテストはない。acdac00a が計画に印を付けた
   - 0341b929 sync-docs。AGENTS.md、README、rules、skills、recipes、quality、repo-map を RC1〜RC12 に照らし、この差分の外に誤りと示された文はなかった。skill・雛形・コードの直しを求める所は tech-debt の行が持ち、sync-docs の「範囲外、残す」に 7 項目ある。verify の INFO-4(self-review の Follow-ups が書く insight event の追記先が UTC の日付とずれていた)も直した
   - 109fb3f9 cross-review の triage(cycle 1/2、指摘 0)。`codex exec review --base main` を read-only で打ち、rc 0。reviewer の報告は、29 ブロックの再現、手順書の 2 写しの一致、pipeline と template-purity の検査の通過
   - insight event は `docs/insights/events/2026-10-25-org-recheck.jsonl` に 2 行(self_review、verify)、`2026-10-26-org-recheck.jsonl` に 3 行(test、sync_docs、cross_review)で、どれも cycle 1。ファイルが 2 つなのは、`scripts/insights-append.sh` が UTC の日付で選ぶため

## 3. 設計の判断

計画の D1〜D12。結果は evidence の Trial deviations と Findings にある。

- D1 場所と環境は、手順書のとおり `/tmp/ralph-iso`、専用の clone、`env -i` の wrapper にした。wrapper に `GOFLAGS=-timeout=30m` を足した(前回と同じ)。これがあるので、Go の verifier の既定の timeout(F10)は使われず、確かめていない
- D2 agmsg の私用の写しは、clone の `ralph.toml` の `[org].agmsg_home` でなく wrapper の `RALPH_ORG_AGMSG_HOME` で渡した。clone の追跡されない `ralph.toml` は機能の worktree に入らず、leader の spawn は worktree の `ralph.toml` を読むので、leader の座席だけが利用者の agmsg を使って team が分かれる、という推論による(コードからの推論で、実機では確かめていない)。手順書との違いは deviation として記録し、手順書の穴を S5 で直した
- D3 通知は shim にして、本物の `osascript` を打たない。shim の 12 行はすべて ralph の通知でなかった(RC11)
- D4 題材は、開いた tech-debt の行の直し 2 つ(backfill-na は `internal/insights/backfill.go`、hooks-matcher は `internal/cli/doctor.go`)。別 package で、触るファイルが重ならず、どちらも 50 行ほどで、テストは git・chmod・プロセス・待ち時間に依存しない。どちらも `docs/tech-debt/README.md` を書かず、行の RESOLVED はこの PR が書く
- D5 model は、leader を `--model opus` にし、分割計画の前文に「implementer は opus、reviewer は sonnet」と書いた。前文が leader に届いたかを `spawn_started` の model で読むため(F5)。届いていた
- D6 フォルダ信頼は人が答える(#254 で `/org` と `/director` の skill に書いた決まり)。計画は人が自分の端末で `claude` を起動して答える道にしたが、画面が出ず記録も残らなかった(I0a。一晩待った)。手順書の道(`/tmp/ralph-iso/run herdr` で捨ての pane に入る)で答えた(I0b)。人の端末で出なかった理由は分かっていない(RC7)
- D7 merge は、両方の org が同じ base の `gate_passed` を記録して disband し、`merge-order` が両方を `ready` と出すまで 1 本目を merge しない。1 本目の merge のあとに 2 本目が直接 `ready` にならず、regate の道を通すため。1 本目(#255)は人の指示で、2 本目(#256)は人の了解のあと、clone の `ralph.toml` に `auto_merge = true` を書いて director が merge した。書く前に、scratch の `ralph.toml` で `ralph org director status --config` が `auto_merge: true` を出すことを確かめた。結果は計画のとおりで、`merge-order` が #256 を `rebase` と出し、regate が `via=regate` の `gate_passed` を記録して push し、`ralph org merge --wait-checks 60m` が checks を 3 min 10 s 待って merge した。`ralph org merge` も `ralph org regate` も断らなかった
- D8 probe は、キーを送らず、捨ての pane で `claude -p` を打つ。計画のコマンドは `--allowedTools` の後ろに依頼を置いて失敗し(RC8)、依頼を `-p` の直後にして通った
- D9 数え方は前回の D5 と同じ。介入は AskUserQuestion の呼び出しと結果の時刻を transcript から読む。費用は座席の session の token を model ごとに足し、その日の定価で見積もる。director 自身の費用は分けられないので外す
- D10 所見ごとに読み先(台帳、`git log`、座席の pane、機能の計画)を evidence の各行に書き、台帳から読める数は `jq` で写しから出し直せるようにした
- D11 evidence と記録は implementer の座席が、director が scratchpad に残したデータから書く。director は実行のあいだ repo のファイルを書かない。f7ad239d と 59779049 は、実行のあとに director が inline で足した
- D12 台帳の写しを 2 回取る。1 回目は disband のあと detach の前(片付けの途中でやめても記録が残る)、最後は director の detach、見張りの停止、server の停止のあと、`/tmp/ralph-iso` を消す前。`detached` の行と見張りの停止の印を写しに入れるため

実行と記録で足した判断は次のとおり。

- RC2 で FR-4 と受け入れ条件 5 を `[ ]` に戻した。根拠の表の印の規則は、項目の動きを起こすのに回り道が要ったものを食い違いとする。座席は立ったが、雛形のとおりに打った spawn は断られ、`--scope` を足して打ち直した。前例は試行の F4。consult も同じ判定だった(計画の Progress)
- RC1(FR-7)と RC6(FR-5)は判断が分かれうるが、`[x]` のままにした。項目の動き(reviewer の結果に head と base の SHA を残す、director から leader へ send する)は起きていて、回り道が要ったのは項目の外(座席から leader への agmsg、送ったことの確かめ)だったため
- RC7 は手順書で直して行を作らず、RC8 と RC12 は repo の欠陥でないので行を作らない
- skill・雛形・コードは直さず、tech-debt の行にした。計画の Scope は記録と手順書だけで、所見が求める skill・雛形・コードの直しは入っていない
- S6 は、計画が sonnet としたが、仕様の印の判断が入るので opus にした(計画の Progress)
- 試行の evidence には、古い文を消さず日付つきの Correction を足した。「most likely」「not checked further」と強さを保っている

## 4. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 環境 | evidence の Setup(`:157`)に、binary の commit ffb3e271、wrapper の変数 16 個の名前、shim、agmsg の写し、models の表がある。手順書の節ごとの記録(`:250`)は、節ごとに「as written」か deviation とその理由を書く。herdr と Claude Code の版は計画の Assumptions のままで、実行中に読み直していない |
| AC2 2 つの org | manifest の `scope_reserved` 2 件が `split=recheck digest=4f955a6f4269` を持ち、並んだのは 46 min 13 s(2773 s)。verify が `gh pr view 255` と `256` で、どちらも MERGED、変更は 5 files +322/-2 と 5 files +326/-1 であることを読んだ |
| AC3 所見の直しの効き | Fix effects(`:426-439`)の 12 行に、期待、観察、読んだ場所、判定がある。表と写しから出る数は verify が確かめた。load average、send の終了コード、pane の読みは director の notes による |
| AC4 merge | 両方の `gate_passed` が base ffb3e271 で、両方が disband したあとに 1 本目を merge した(manifest の 03:09:33Z、03:10:18Z、03:16:07Z、03:16:47Z)。`pr_merged` は #255 が `auto_merge=false`(536271a2)、#256 が `auto_merge=true`(98d38391)。#256 の前に `via=regate` の `gate_passed`(14:18:21Z、head 77c631b2)がある。`merge-order` が `rebase` と出したことは notes にだけある |
| AC5 数値と前回との比較 | 介入、承認待ち、受信箱の処理、見張りの escalation、費用、gate の表が、`sh` ブロックで写しから出る。前回との比較(`:1137`)の試行側の値は、試行の写しの manifest から出し直せた分と、試行の PR の verify の報告に書かれた値と合う(verify が確かめた) |
| AC6 記録 | tech-debt の 2 行が #256 と #255 で RESOLVED、新しい 9 行、開いたまま残る 2 行への回数の書き足し、仕様の Status と根拠の表の節。節 2 の 4 を見る |
| AC7 手順書 | 2 写しが `cmp` で一致し、`scripts/check-sync.sh` が rc 0。「通しで打っていない」とする 2 文は消え、D2 の渡し方は `:174`、attach の打ち方は `:386`、`.git/info` は `:82`、機能の worktree の片付けは `:529` |
| AC8 片付け | evidence の Cleanup(`:1186`)の 8 段が AC8 の順に並ぶ。写しの痕が順と合う: `director.jsonl` の `detached`(14:23:22Z)、`watch-alive.json` の `stopped`(14:23:34Z)、`director-wait.json` の `end=watch`(14:23:36Z)、`ralph-status-final.txt` の `org watch: not running` |
| AC9 写し | 39 ファイルがすべて追跡されている(`.log` の 3 つは `git add -f`)。`sh` ブロック 29 個が表と合う(verify と、この walkthrough を書く前に打ち直した)。漏れの grep(利用者のホームの絶対 path、pane id、session id、token の形、U+FFFD)で、漏れにあたる値は verify の記録で 0 件(その path の文字列が計画と報告に 3 件あるのは、grep の語として書かれたもの) |

## 5. 確かめていないこと

- AC3 と AC8 の一部は、director の notes による。load average、send の終了コード、pane の読み(trust の画面が出なかったこと、RC6 の約 22 行)、`merge-order` の `rebase` の出力、AC8 の各段の終了コード、利用者の herdr と agmsg の更新時刻。notes は scratchpad にあり、commit していない(evidence の冒頭にもそうある)。AC8 の `pgrep -fl /tmp/ralph-iso` は、`rm -rf` のあと(Cleanup の 8 段目、終了コード 1)の記録だけで、見張りを止めた直後の記録は「ralph の見張りがない」までしかない(計画の Progress の「AC の印」の行、verify の AC8)
- verify の INFO の扱い。INFO-1(仕様 `:29` の「3 項目(2026-10-23 の時点)」)は直していない。日付つきの文で、今の数は `:35` にある。INFO-2(`/org` skill のフォルダ信頼の項 `.claude/skills/org/SKILL.md:49-65` と 3 つの写しが、人が main のチェックアウトで `claude` を起動して trust すると書くこと。この実行の clone ではその道で画面が出なかった)は、skill の直しでこの記録の branch の範囲外とし、tech-debt の行もない。分離しない環境でも同じかは確かめていない。INFO-3(計画の印)は f5bc474b で直した。INFO-4 は、verify のあとの時点では直さないとしたが、0341b929(sync-docs)で直した。計画の Progress の verify の行は、それより前に書いたので「INFO-4 は直さない」のまま残っている
- archive のあと、この計画の `docs/plans/active/` の path を指す参照が、HEAD に 23 か所ある。`scripts/archive-plan.sh` が書き換えるのは tech-debt の 12 か所だけで、残る 11 か所(8 ファイル)は `docs/plans/active/` のまま切れる。evidence の冒頭(`:3`)、仕様の Status(`:35`)、根拠の表(`:46`)、報告 5 本(self-review、verify、test、sync-docs、triage)。この walkthrough の `Plan:` の行も同じになる。前の PR(trial-docs-wrapup)の報告と試行の evidence に同じ形で残っているので前例はある。verify の「4 つ」は 7f004c11 の時点の数で、test、sync-docs、triage の報告と verify 自身の報告がまだなかった。直すなら `/pr` の archive の手順の外で、別に書き換えることになる
- 停止と再開(前回の D6)は Non-goals で、実行していない。実行のあいだに org を止めることもなかった。仕様の open question は開いたまま
- evidence の Not verified: F7 で leader が Monitor で待ったか、F13 で通知の条件を満たした件の記録、F10 の Go の verifier の既定の timeout、F11 で push しなかったコミットがローカルのブランチにあったか、実行中の herdr と Claude Code の版、director 自身の費用、RC9 の読む順序(コードと時刻からの推論)、RC11 の hook の中身(director の notes による)、RC1 を `[org].agmsg_home` が既定でないときに、RC3 を director なしのときに、codex の座席と codex の director と headless の director
- 開いたまま残る行: RC1〜RC6 と RC9〜RC11 の 9 行は、skill・雛形・コードの直しを要し、この PR は直さない。RC2 は雛形のとおりに spawn を打つ機能の org で毎回起きるので、release の前に直すかどうかはユーザーが判断する(sync-docs の Left open)。仕様の `[ ]` は FR-3(leader への ALERT)、FR-4、FR-10、ツール非依存(Codex の leader)、受け入れ条件 5 で、ほかに watcher の時間切れ(今回は 0 回)と N4(template の recipe が scaffold に届かない evidence を引く)が開いている
- この walkthrough を書くために打ったのは、`git` の読み取り、`wc`、`cmp`、`./scripts/plan-visual.sh digest`、evidence の `sh` ブロック 29 個(`jq`・`awk`・`grep`・`git log`)。29 個とも出力が表か `text` ブロックと合い、rc 1 は F11 の `grep -c '^wip:'` が 0 件のときの 1 個だけだった。herdr、agmsg、`gh`、network には触れていない。PR の内容(#255 と #256 の変更)は verify の `gh pr view` の記録に頼り、この walkthrough では読み直していない
