# busy-leader

- Status: Approved
- Approved: 2026-10-23 sha256:f046c0eac2ff
- Owner: Claude Code
- Date: 2026-10-23
- Related request: 試行の所見を直す系列の 7 本目「busy な leader」(ユーザーの依頼「2. 修正可能なものは修正してください」、2026-10-21)
- Related issue: N/A
- Type: fix
- Branch: fix/busy-leader

## Objective

座席を待つ leader に、director の `ralph org send --to leader` が届くようにする。Claude Code の leader は、agmsg の受信箱を Monitor で見張り、長いコマンドを `run_in_background` で走らせて手番を終える。手番を終えた pane は herdr が done と報告するので、`Send` の idle/done の待ちが通る。Codex の leader には Monitor がないので、待つあいだは届かないことを雛形に書き、tech-debt に残す。あわせて、`Send` の待ちが切れたときのエラーに理由と次の手を足し、director の DECISION の送り方を直す。試行(stage 9)の所見 F7(agmsg を polling する leader に `ralph org send` が 4 回待ち切れ、DECISION が 49 分後に届いた)を、Claude Code の leader について閉じる。あわせて、記録に残る「fix/watch-escalation-inbox」を「#252」に直し、#252 で約束した tech-debt の行を足す。

## Scope

- leader の雛形 `internal/org/prompts/leader.md` の待ち方
  - `## agmsg の受信箱の運用` の節を書き直す。「能動的に確認してください」「適当な間隔で確認してください」の文はなくす
  - 前面で長く待たない。手番の途中の leader には `ralph org send` が届かない(`Send` は座席が idle か done になるのを待ってから打つ)ので、director の DECISION も STOP も手番が終わるまで届かない
  - Claude Code の leader は、agmsg の受信箱を Monitor で見張る。コマンドは `p=$PPID; end=$(( $(date +%s) + 1500 )); while kill -0 "$p" 2>/dev/null && [ "$(date +%s)" -lt "$end" ]; do bash <agmsg の scripts>/inbox.sh <team> leader --quiet 2>&1 | grep --line-buffered -v '^[[:space:]]*$'; sleep 10; done; echo '<終わりの 1 行>'` の形で、`timeout_ms` に 1800000 を渡す。ループは自分の期限(25 分)で Monitor の期限(30 分)より先に終わり、親(leader の claude)がいなくなれば 10 秒ほどで終わる。終わりの 1 行か Monitor の期限の知らせが来たら張り直す。張ったら手番を終える。メッセージが届くと Monitor の知らせで次の手番が始まる
  - `inbox.sh --quiet` は表示したメッセージを既読にする。張り直す前と、届くはずのメッセージが来ないときは、agmsg の `history.sh <team> leader` で直近のメッセージを読み、まだ扱っていないものを扱う
  - gate のような長いコマンドと `ralph org wait` は `run_in_background` で走らせ、手番を終える。終わると知らせが来る。`ralph org status` と `ralph org read` はすぐ返るので前面で打ってよい
  - Codex の leader には Monitor がない。座席を待つあいだ(前面のループ、`ralph org wait`、gate)は手番が終わらず、`ralph org send` は届かないと書く。時間を区切れば届くとは書かない。返事(DECISION)だけを待つときは、座席を待たずに手番を終える
  - 雛形のほかの「待つ」(使命の 4、件を上げたあとの「返事を待つ」、機能ごとの org の手順 3・7・12)が、この節の待ち方を指すようにする
- agmsg の置き場所を雛形に渡す
  - 雛形に置き換え文字 `{{AGMSG_SCRIPTS}}` を足す。値は ralph が解いた agmsg の置き場所(`driver.ResolveAgmsgHome`)の下の `scripts` の絶対パスを、shell の単一引用符で囲んだもの
  - CLI の配線(`internal/cli/org.go`)が `Org` に解いた置き場所を渡す。空のときは既定の置き場所を使う
  - 置き場所を固定の `~/.agents/skills/agmsg` にすると、`[org].agmsg_home` か `RALPH_ORG_AGMSG_HOME` で別の agmsg を使う org の leader が、別の受信箱を見張ってしまうため
- `Send`(`internal/org/verbs.go`)の idle/done の待ちが時間切れで終わったときのエラーに、座席の手番が `--timeout-ms` のうちに終わらなかったこと、何も打っていないこと、前面で待つ座席には手番が終わるまで届かないこと、あとで打ち直すか `--timeout-ms` を長くして打ち直すことを足す。今の書き出し `org: send: wait for seat %q idle/done:` は残す。時間切れでない失敗(herdr の失敗、座席がない)の文は変えない。動きは変えない(org-runtime の仕様 FR-3 の idle の待ちは残す)
- skill の送り方(4 写しずつ)
  - `/director` skill の「答える順」と「計画の承認」の 3: leader への send が待ちで切れたら resolve せず、`--timeout-ms` を長くして打ち直す。Claude Code の director は、send を `--timeout-ms 1500000` で Bash の `run_in_background` で走らせ(Bash の前面の打ち切りは最長 10 分なので)、そのあいだも受信箱のほかの件を処理する。同じ leader への send は 1 本ずつにする(`Send` には座席ごとのロックがない)。Codex の director は `--timeout-ms` を区切った send を打ち直す
  - 終了コード 0 でも、それだけで resolve しない。`ralph org read --org-id <org> --seat leader` で leader の pane を読み、DECISION の SUMMARY の行が送られた入力として出ていて、入力欄に残っていないことを確かめてから resolve する。入力欄に残っていれば、`/org` skill の `send` の行のとおり Enter を 1 回だけ送り、もう一度読む。`Send` の submit の確かめは、座席が working になったことしか見ないので、Monitor の知らせなど別の理由で手番が始まっても送れたと読むため(`confirmSubmitted` の doc comment)
  - `/org` skill の動詞の表の `send` の行に、送る前に座席が idle か done になるのを待つこと、手番の途中の座席には届かないこと、待ちが切れたら何も打たずにエラーで終わり、そのまま送り直せることを書く。「受信箱」節の答える順と、leader の agmsg の受信箱の段落(今の「`/agmsg` skill の手順に従う」)を雛形に合わせる
- 実機の確かめ(分離した herdr server、XDG の 3 変数、本物の claude haiku、記録は `docs/evidence/busy-leader-2026-10-23.md`)
  - (a) 新しい雛形で spawn した haiku の leader が、Monitor を自分で張って手番を終え、herdr が leader を done(か idle)と報告する
  - (b) その leader に、本物の `ralph org send --to leader` を既定の 30 秒で打つと、終了コード 0 で届き、新しい手番が始まる
  - (c) そのあと、座席の名前で agmsg の `send.sh` から leader に送った文で、leader が Monitor の知らせで起こされる
  - (d) 対照: leader が前面で 90 秒ほど待っているあいだに `--timeout-ms 5000` の send を打つと、終了コード 1 で、新しいエラーの文が出て、何も打たれない
  - (e) leader の claude を `kill -KILL` で止めると、Monitor のループが 20 秒のうちに終わり、残らない(`ps` で読む)。ループの `$PPID` が leader の claude であることも読む
  - (f) 重なり: `--enter-delay-ms 15000` の send を打ち、その待ちのあいだに座席の名前で `send.sh` から送って Monitor の知らせで手番を始めさせる。DECISION が送られた入力になったか、入力欄に残ったかを `ralph org read` で読み、submit の確かめの結果(stderr の注意の有無と manifest の `submit_unconfirmed`)と並べて記録する
  - 終わりに `ralph org disband` で org を閉じ、leader の Monitor のループが残っていないことを `ps` で読む
- 記録
  - 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-5 を、(a)〜(c) の実機の記録がそろえば `[x]` にし、印(2026-10-23)の注で Codex の leader の欠けを NFR ツール非依存に結ぶ。NFR ツール非依存の注に Codex の leader の欠けを足す。Status の節の `[ ]` の数を直す
  - 根拠の表 `docs/evidence/org-spec-evidence-2026-10-21.md` の F7 の行、FR-5 の行、NFR ツール非依存の行、「確かめていないこと」
  - tech-debt `docs/tech-debt/README.md` の F7 の行を Codex の leader に限る文に書き換え、Claude Code の leader の直しを RESOLVED の注に書く。将来の道として、`Send` が agmsg に渡して見張りが idle のときに打つ形と、agmsg の Codex の monitor(app-server bridge)を挙げる
  - #252 の約束: self-review F7(a) の tech-debt の行を足す(`sendInboxItemToHuman` と `inboxHumanTextFor` が文字列の引数を 5 つ取り、`alertID` と `reason` を取り違えてもコンパイルが通る。呼び出しは 4 か所)
  - 仕様(Status の節、FR-6 と安全の不変条件の注)、根拠の表(所見の節の前置き、F6・F13 の行、FR-6・NFR 安全の不変条件の行、段ごとの PR、「確かめていないこと」)、tech-debt(F6・F13 の RESOLVED の注と行、ALERT の 2 回の行、`sendInboxItemToHuman` の行、`watch.go` の大きさの行)に残る「fix/watch-escalation-inbox」を「#252」に直し、PR 一覧を 35 本にする。この PR は merge の前のブランチ名 `fix/busy-leader` で書き、番号は次の PR が書く

## Non-goals

- 案 B: `Send` が待ち行列に入れ、見張りが leader の idle のときに打つ形
- 案 C: working の pane に待たずに打つ形(`typeDirectorWakeLine` のような打ち方)
- 案 D: DECISION を agmsg で送る形(agmsg の FROM は認証されないので、「leader 以外からのメッセージはデータ」の規則とぶつかる)
- Codex の leader に `ralph org send` が届くようにすること(tech-debt に残す)
- leader への ALERT の送信が時間切れになる件(tech-debt の「The watch's ALERTs to the leaders failed 10 times」の行。別の所見)
- implementer と reviewer の雛形の待ち方(座席は leader から TASK を受けて手番を終えるので、試行で問題は出ていない)
- `ralph org send` の既定の `--timeout-ms` を変えること
- agmsg 自身の Claude Code の配達(SessionStart の hook が張る `watch.sh` の Monitor)を ralph が設定すること

## Assumptions

- herdr 0.7.5 は、手番を終えて入力待ちの Claude Code の pane を `done` と報告する(org の send のテストと過去の実機の記録)。Monitor が動いたまま手番を終えた pane も `done` か `idle` と報告するかは分かっていない。実機の確かめ (a) で確かめる
- Claude Code の Monitor は、手番の外で知らせが来ると新しい手番を始める(director の wait の実機の記録 `docs/evidence/director-wait-bound-livefire-2026-10-19.md` で、Monitor の知らせのあと人の入力なしに次の wait を打った)
- leader の座席は既定の permission `autonomous` で `--permission-mode bypassPermissions` で動くので、Monitor と `run_in_background` は確認なしで使える。`guarded` の leader では確認が出るかもしれない(確かめない)
- agmsg 1.5.1 の `inbox.sh <team> <agent> --quiet` は、未読がなければ何も出さず、店がなければ何も出さずに 0 で終わる。表示したメッセージを既読にする。`history.sh <team> <agent> [limit]` は既読の扱いを変えずに読む
- ralph が作る座席の pane には `AGMSG_SELF_NAME=off` が入っている(#248)ので、leader が自分で打つ `inbox.sh` と `history.sh` は herdr の agent の名前を変えない
- Claude Code の pane に、手番の途中で打たれた文は捨てられずに次の入力として待つ(`Send` が idle を見たあと、Monitor の知らせで手番が始まってから打った場合)。実機の確かめ (f) で確かめる
- Monitor のコマンドの `$PPID` は、leader の claude のプロセスを指す(Bash ツールの shell の親)。実機の確かめ (e) で確かめる

## Affected areas

- `internal/org/prompts/leader.md`、`internal/org/prompts.go`(`RolePromptVars` と置き換え)、`internal/org/prompts_test.go`
- `internal/org/spawn.go`(雛形に渡す値と `Org` の欄)、`internal/org/spawn_test.go`、`internal/cli/org.go`(配線)と、配線を見るテスト
- `internal/org/verbs.go`(`Send` のエラー)、`internal/org/verbs_test.go`
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と写し(`.agents/skills/`、`templates/base/.claude/skills/`、`templates/base/.agents/skills/`)、skill の文を固めるテスト(`internal/org/director_skill_sync_test.go` か新しいテスト)
- `docs/evidence/busy-leader-2026-10-23.md`(新規)
- `docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/org-spec-evidence-2026-10-21.md`、`docs/tech-debt/README.md`

## Visual review

- ページ: `.harness/state/plan-visual/busy-leader.html`(かんたん版の図 2 つ: `overview`、`easy-retry`。折りたたみの中に `detail-overview`、`detail-flow`、`detail-error`)
- lint: `./scripts/plan-visual.sh lint` が exit 0
- 自己点検: 全体、かんたん版の 2 つ、詳細の 3 つを `shot` で撮って読んだ。詳細 2 の S4 の印が左端に離れていたので director の送信の横に移し、撮り直して直ったことを確かめた
- Codex の計画助言(HIGH 2 件)を入れた: ループの寿命(D11、確かめ (e))と、DECISION が届いたかの確かめ(D12、確かめ (f))。図は、かんたん版 2 の「届いた」のあとに画面で確かめる箱を足し、詳細 2 と詳細 3 と承認の前の箱に同じことを書いた

## Design decisions

- D1 方式(consult-plan-busyleader の案 A): `Send` の idle/done の待ちは残し、leader が手番を終えて待つようにする。案 B は見張りに打つ役目と待ち行列が増え、案 C は手番の途中の pane に打つので承認の画面に Enter が入る危険が戻り、案 D は agmsg の FROM が認証されないので DECISION の権威が崩れる。どれも 1 スライスに収まらない
- D2 見張りの形: `inbox.sh --quiet` のループを Monitor で見張る。`ralph org wait --seat` は座席の herdr の状態を見るだけで、見張りの ALERT や座席の QUESTION を拾えない。agmsg の `watch.sh` は project のパスと agent の型から team と agent を解くので、leader の team だけに絞れるかが分からない。空行は grep で落とし、Monitor の知らせの数を減らす。10 秒ごとの読みは、試行の leader の 9 秒ごとと同じくらいで、agmsg の店を読むだけの重さ
- D3 取りこぼし: `inbox.sh --quiet` は表示した時点で既読にするので、Monitor が表示の直後に止まると、そのメッセージは知らせに載らないまま既読になりうる。雛形には、張り直す前と、届くはずの RESULT が来ないときに `history.sh` で読み直すと書く。既読にしない読み方を作るのは agmsg の側の仕事なので、ralph は持たない
- D4 agmsg の置き場所: 雛形に `{{AGMSG_SCRIPTS}}` を足し、ralph が解いた置き場所の `scripts` を単一引用符で囲んで埋める。`Org` に欄を 1 つ足し、CLI の配線が `driver.ResolveAgmsgHome` の値を入れる。空なら `driver.ResolveAgmsgHome("")`(既定の置き場所)を使う。leader が agmsg skill の手順から自分で置き場所を探す案は、試行の leader がそうしたように動くが、`[org].agmsg_home` を変えた org では別の店を読むので採らない
- D5 長いコマンド: Claude Code の leader は gate と `ralph org wait` を `run_in_background` で走らせる。gate の手順 12 はもうバックグラウンドで走らせてファイルに書く形なので、終わるまで前面で待たず、終わりの知らせで読むように直す。Codex の leader は前面で走らせ、そのあいだは届かない
- D6 Codex の leader: 届くようにはしない(Non-goals)。雛形に「待つあいだは届かない」と書き、区切れば届くとは書かない。区切った wait のあいだの手番も working なので、区切っても届かないため。返事だけを待つときは手番を終えるよう書く。leader の pane に打たれた DECISION は、手番が終わっていれば届く
- D7 エラーの文: 時間切れは、`Send` の ctx の期限が来たこと(`errors.Is(err, context.DeadlineExceeded)` か `ctx.Err()` が期限切れ)で見分ける。試行の文は `herdr: timed out: context deadline exceeded` で、`driver.execRun` が ctx の期限を包んで返したもの。herdr 自身の `--timeout` は ctx より後に作るので、ctx が先に切れる。時間切れのときだけ文を足し、ほかの失敗の文は変えない。どちらも pane に何も打っていないので、送り直せる(今の `/org` skill の文と同じ)
- D8 director の送り方: Claude Code の director は、`--timeout-ms 1500000` の send を Bash の `run_in_background` で走らせる。Bash の前面の打ち切り(最長 10 分)より長く待てて、終わりの知らせが 1 回来る。Monitor ではなく `run_in_background` にするのは、終わりの 1 回だけを知りたいため(Monitor は行ごとに知らせる)。同じ leader への send は前の send が終わってから打つ。Codex の director は `--timeout-ms 300000` のように区切って打ち直す
- D9 FR-5 の印: (a)〜(c) の実機の記録がそろったときだけ `[x]` にする。注で、Codex の leader には届かないことを NFR ツール非依存(すでに `[ ]`)に結ぶ。根拠の表の印の規則(「外の道具に頼る項目はその道具で動かした実機の記録も 1 つあり、試行の所見のどれとも食い違わない」)では、試行の leader は Claude Code で、直したあとは食い違わない。Codex の leader は試行で動かしていない。(b) か (c) が通らなければ、FR-5 は `[ ]` のまま残し、consult で方針を見直す
- D10 tech-debt の F7 の行: 取り消し線で閉じず、Codex の leader に限る文に書き換えた行として残す。Claude Code の leader の直しは、その上の `<!-- RESOLVED ... -->` の注に書く。所見の表の tech-debt の欄は、書き換えた行の書き出しで指す
- D11 ループの寿命(Codex の計画助言の指摘 1): Monitor のループは、自分の期限(1500 秒)と親の生死(`kill -0 $PPID`)で終わる。Monitor の `timeout_ms` だけに任せると、leader の claude が落ちたときや Monitor の期限が子のプロセスを残したときに、古いループが `inbox.sh --quiet` を打ち続けて、立て直した leader のメッセージを先に既読にしうる。期限を Monitor の 30 分より短くするのは、ループが自分で終わって終わりの 1 行を出し、leader が張り直す合図にするため。古いループが残らないことは実機の確かめ (e) と disband のあとの `ps` で見る。agmsg の読み手を 1 つに限る仕組み(ロック)は agmsg の側の仕事なので持たない
- D12 DECISION が届いたかの確かめ(Codex の計画助言の指摘 2): `Send` の submit の確かめは、Enter のあとに座席が working になったかしか見ない。Monitor の知らせで leader の手番が別に始まると、DECISION が入力欄に残っていても送れたと読む。director は send の終了コード 0 のあと、`ralph org read` で leader の pane を読み、DECISION が入力として送られたことを確かめてから resolve する。`Send` に「打った文が入力欄から消えたか」の確かめを足す案は、herdr の pane の読みを Claude Code と Codex の画面の形に結びつけるので採らない(範囲の外)。実機の確かめ (f) で、重なったときに何が起きるかを記録する
- Critical forks: None(D1〜D3 と D9 は consult で決めた。D4〜D8 と D10〜D12 は、どれも 1 スライスの中で戻せる)

## Acceptance criteria

- [x] AC1: leader の雛形の `## agmsg の受信箱の運用` の節が、Scope の待ち方を書く。前面で長く待たない理由(`Send` は idle か done を待つ)、Claude Code の Monitor のコマンド(`{{AGMSG_SCRIPTS}}` と `{{TEAM}}` を使い、空行を落とし、10 秒ごと、1500 秒の期限、`kill -0 $PPID` の親の確かめ、終わりの 1 行)と `timeout_ms` 1800000 と張り直し、手番を終えること、張り直す前の `history.sh` での読み直し、長いコマンドの `run_in_background`、Codex の leader には待つあいだ届かないこと、返事だけを待つときは手番を終えること。「能動的に確認してください」と「適当な間隔で」の文はない。使命の 4、件を上げたあとの待ち、手順 12 の gate の待ちがこの節に合う。prompts_test.go が節の文と、ほかの待ちの文を固める
- [x] AC2: `{{AGMSG_SCRIPTS}}` は、`Org` の agmsg の置き場所の `scripts` を単一引用符で囲んだ値に置き換わる。置き場所が空なら既定の置き場所を使う。描いた leader の雛形に `{{` が残らない。spawn が書く leader の role prompt に、渡した置き場所の `scripts` が入る(spawn_test.go)。CLI の配線は `driver.ResolveAgmsgHome` の値を渡す
- [x] AC3: `Send` の idle/done の待ちが ctx の期限で切れると、エラーは `org: send: wait for seat "<seat>" idle/done:` で始まり、手番が `--timeout-ms` のうちに終わらなかったこと、何も打っていないこと、前面で待つ座席には手番が終わるまで届かないこと、`--timeout-ms` を長くして打ち直せることを言う。Progress は `SendProgressNothingSent` で、`sent` の event は増えない。時間切れでない失敗の文には足さない(verbs_test.go)
- [x] AC4: `/director` skill(4 写し)の「答える順」と「計画の承認」の 3、`/org` skill(4 写し)の `send` の行、「受信箱」節の答える順、leader の agmsg の受信箱の段落が、Scope の送り方と待ち方を書く。写しはバイト一致。director の送り方(`run_in_background`、`--timeout-ms 1500000`、同じ leader への send は 1 本ずつ、切れたら resolve しない、終了コード 0 のあとも `ralph org read` で DECISION が送られたことを確かめてから resolve する)をテストが 4 写しで固める
- [x] AC5: 実機の確かめ (a)〜(f) と disband のあとの `ps` を分離した herdr server と本物の claude haiku で打ち、結果を `docs/evidence/busy-leader-2026-10-23.md` に残す。ユーザーの herdr server と `~/.config/herdr` に触れず、本物の osascript を打たない(PATH の shim)。記録には pane の id、session の id、token の値を書かず、`$HOME` は `~` と書く
- [x] AC6: 記録を合わせる。(a)〜(c) が通れば FR-5 を `[x]` にし、印(2026-10-23)の注で Codex の leader の欠けを NFR ツール非依存に結ぶ。根拠の表の F7・FR-5・NFR ツール非依存の行と「確かめていないこと」、仕様の Status の節の数、tech-debt の F7 の行(D10)を直す。F7(a) の tech-debt の行を足す。「fix/watch-escalation-inbox」を「#252」に直し、PR 一覧を 35 本にする(#252 が MERGED であることを `gh pr view 252 --json state --jq .state` で読む)。この PR は `fix/busy-leader` で書く
- [ ] AC7: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が rc 0。変えたパッケージの Go のテストが `GOTOOLCHAIN=go1.25.8` でも通る

## Implementation outline

1. S1(implementer): 雛形の待ち方と `{{AGMSG_SCRIPTS}}`(AC1、AC2)。leader.md、prompts.go、spawn.go、cli の配線、prompts_test.go、spawn_test.go
2. S2(implementer): `Send` の時間切れのエラー(AC3)。verbs.go、verbs_test.go
3. S3(implementer): 実機の確かめ(AC5)。S1 と S2 を入れた binary で (a)〜(f) と disband のあとの `ps` を打つ。(e) で古いループが残るか、(f) で DECISION が入力欄に残ったときは、記録して止め、雛形か skill の文を consult で見直す。(a) で leader が Monitor を張らないときは、雛形の文を 1 回だけ直して打ち直し、それでも張らなければ sonnet でも打って両方を記録する。(b) か (c) が通らなければ止めて、consult で方針を見直す
4. S4(implementer): skill の送り方(AC4)。`scripts/sync-skills.sh` で写しを作り、`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` で確かめる。skill の文を固めるテスト
5. S5(implementer、sonnet): 記録(AC6)

## Verify plan

- Static analysis checks: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh`(gofmt、go vet、shellcheck、check-sync、check-skill-sync ほか)
- Spec compliance criteria to confirm: AC1〜AC7。仕様の印が FR-5 の 1 つだけ動くこと(実機が通ったとき)
- Documentation drift to check: leader の待ち方と director の送り方を書いた文(leader の雛形、`/org` と `/director` の skill、`Send` の doc comment、仕様の FR-5 の注、tech-debt)が食い違わないか。「能動的に確認」「適当な間隔」の古い文が残っていないか。`ralph org wait` を前面で打つよう読める文が雛形に残っていないか
- Evidence to capture: run-verify の rc と時間、実機の記録

## Test plan

- Unit tests: prompts_test.go に、節の文(Monitor のコマンドの期限と `kill -0 "$p"` と終わりの 1 行、`timeout_ms`、手番を終える、`history.sh`、`run_in_background`、Codex の「届かない」)と、置き換え文字が描いた値になること(単一引用符、空のときの既定)、古い文がないこと。verbs_test.go に、ctx の期限で切れた待ちのエラーの文と、期限でない失敗の文が変わらないこと
- Integration tests: spawn_test.go で、`Org` に置き場所を渡した leader の spawn が、role prompt のファイルにその `scripts` を書くこと。skill の文のテストが 4 写しを読む
- Regression tests: `TestRenderRolePrompt_Leader_NamesWhichInbox` を新しい文に合わせて通す(節の見出しと、スター型の規則の agmsg の受信箱の文は残す)。`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` が通る(期限でない失敗)。prompts の「`{{` が残らない」テストが leader で通る
- Edge cases: 置き場所に単一引用符や空白を含むとき(引用の形)、置き場所が空のとき、`--timeout-ms` が小さく待ちの前に予算が尽きるとき(今の予算の検査の文は変えない)、待ちが ctx の期限でなく herdr の失敗で終わるとき
- Evidence to capture: `go test -count=1 -timeout 30m ./internal/org/... ./internal/cli/...` を go1.26 と go1.25.8 で。実機は `docs/evidence/busy-leader-2026-10-23.md`

## Risks and mitigations

- herdr が、Monitor の動く Claude Code の pane を working と報告するかもしれない(そうなら案 A は効かない): S3 の (a) と (b) で最初に確かめる。通らなければ止めて consult に戻る。S3 の前に入れるのは雛形の文とエラーの文だけで、どちらも戻しやすい
- 本物の leader(opus)は雛形の文に従っても、haiku は従わないかもしれない: S3 で 1 回だけ文を直し、それでも従わなければ sonnet でも打って両方を記録する。試行の leader は opus で、雛形には従わない書き方の余地(「能動的に確認」)があった
- Monitor の知らせが多すぎて Claude Code に止められる: 空行を落とし、未読があるときだけ出す。止められたら、雛形の張り直しの文で張り直す
- Monitor が切れたあと、leader が張り直さずに止まる: 切れると知らせが来て手番が始まるので、雛形に張り直すと書く。張り直さなかった leader も、director の DECISION と見張りの ALERT は届く(手番が終わっているので)
- `inbox.sh --quiet` の既読で、メッセージを取りこぼす: D3 のとおり `history.sh` で読み直す文を書く。座席の手番が終わったのに RESULT が来ないことは、`ralph org status` と `ralph org read` で leader が気づける
- director の 25 分の send が、ほかの件の処理を止める: `run_in_background` で走らせ、そのあいだもほかの件を処理すると skill に書く(D8)
- 同じ leader に 2 本の send が重なり、2 つの文が混ざる: skill に 1 本ずつと書く。`Send` にロックを足すのは範囲の外
- `Send` が idle を見たあと、Monitor の知らせで leader の手番が始まり、その途中に文が打たれる。submit の確かめは working を見て通るので、DECISION が入力欄に残っていても send は 0 で終わる: director は `ralph org read` で確かめてから resolve する(D12)。重なったときの動きは実機の確かめ (f) で記録する
- leader の claude が落ちたり立て直されたりしたあと、古い Monitor のループが残って受信箱を先に既読にする: ループに自分の期限と親の確かめを入れる(D11)。(e) と disband のあとの `ps` で確かめる。`$PPID` が claude でない(間に shell が入る)と、claude が落ちても期限まで残る。そのときは 25 分で終わるので、張り直した leader は `history.sh` で読み直す
- agmsg 自身の hook(SessionStart の `watch.sh` の Monitor、Stop の `check-inbox.sh`)が座席の claude で動くと、知らせが 2 重になるか、既読を取り合う: S3 で座席の claude にその hook があるかを見て記録する。ralph はユーザーの agmsg の設定を変えない
- 雛形に agmsg の置き場所の絶対パスが入る: 座席の role prompt のファイルは台帳の下にあり、コミットされない。ログにパスが出るのは今の spawn と同じ

## Rollout or rollback notes

- 効くのは次の release の binary から。scaffold の skill の文は `ralph upgrade` で入る。leader の雛形は binary に入っているので、新しい binary で spawn した leader から効く
- release のノートに書く文: 「Claude Code の leader は agmsg の受信箱を Monitor で見張って手番を終えるので、director の `ralph org send --to leader` が届く。Codex の leader は待つあいだ届かない。`ralph org send` の待ちが切れたときのエラーは、手番が終わらなかったことと打ち直し方を言う。新しい skill の送り方(25 分の send をバックグラウンドで)は古い binary でも使える」
- 戻すときはこの PR を revert する

## Open questions

- Monitor の動く pane の herdr の状態(S3 で確かめる)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの常設の指示「作成したプランは自動承認します」による自動承認、2026-10-23)
- [x] Branch created
- [x] Implementation started
  - S1: 24e99eb1(雛形の待ち方と `{{AGMSG_SCRIPTS}}`)、10e284af(consult の助言で、見張りを重ねない 1 文、`history.sh` の 50 件、zsh でもコマンドを走らせるテスト。zsh は `-f` で走らせる。この機械の `/etc/zshenv` が `ZDOTDIR` を戻すため)、02e27384(S3 の (a) で haiku が Monitor の必須の `description` を付けずに 1 回断られたので、雛形に書いた)
  - S2: f0961a8b(待ちの時間切れの文)、3d1946d6(英語の言い回しの直し)
  - S3: 4f9babac(実機の確かめ (a)〜(f) と disband はすべて通った。雛形の直しは要らなかった。(d) は Claude Code が前面の `sleep 90` を断るので、perl の 90 秒の待ちで確かめた。(f) の 2 回目で、待ち行列に入った DECISION も submit の確かめを通ることを確かめ、D12 の pane の確かめが要ることが分かった)
  - S4: 460e341c(director と org の skill の送り方、`ralph org send --help`。org skill の `approve` の行と「機能の計画の承認」の 4・5 も director に合わせた)
  - S5: 2df13dfd(仕様の FR-5 を `[x]`、根拠の表、tech-debt の F7 の行を Codex の leader に限る文に、F7(a) の行、#252 の書き換え)
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
