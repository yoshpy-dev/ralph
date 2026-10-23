# Self-review report: busy-leader

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-busy-leader.md`(D1〜D12、AC1〜AC7)
- Reviewer: reviewer サブエージェント(`/self-review`、pipeline cycle 1/2)
- Scope: `git diff a58f7c64...HEAD`(HEAD d4922368、24 ファイル、+1673/-206)。S1 24e99eb1・10e284af・02e27384(leader の雛形の待ち方、`{{AGMSG_SCRIPTS}}`、`Org.AgmsgHome` と CLI の配線)、S2 f0961a8b・3d1946d6(`Send` の時間切れの文)、S3 4f9babac(実機の記録)、S4 460e341c(skill 4 写し × 2、`ralph org send --help`、skill の文を固めるテスト 2 つ)、S5 2df13dfd・d4922368(仕様、根拠の表、tech-debt、plan の印)。差分の質だけを見た。再レビュー(「Re-review of 1183b08e (S6)」の節)は S6 の 1183b08e(`git diff 81f30f43 1183b08e`、5 ファイル、+389/-301)までを見る

## Evidence reviewed

- `internal/org/prompts.go` の差分を全文読んだ。`agmsgScriptsWord`(79)は `home` が空なら `driver.ResolveAgmsgHome("")` を使い、`filepath.Join(home, "scripts")` を `filepath.Abs` で絶対にしてから、`feature.go:439` の `shellQuote`(単一引用符で囲み、中の `'` を `'\''` にする)に通す。`Abs` が失敗するのは相対パスで `os.Getwd` が失敗するときだけで、そのときは元のパスのまま。`RolePromptVars.AgmsgHome` の doc は、`Org.AgmsgHome` と CLI の配線(`internal/cli/org.go:353`、366)の文と合う。`internal/org/driver` の import は `director_cmd.go` が先に持つので、依存の向きは新しくない
- `AgmsgHome` が届く道を洗った。`RenderRolePrompt` の呼び出しは `spawn.go:1012` と `:2268`(`dryRunSpawn`)の 2 つで、どちらも `o.AgmsgHome` を渡す。`org.Org{` を作る所は `internal/cli/org.go:354`(配線あり)、`org_inbox.go:351`、`org_director_headless.go:789`、`director_check.go:361` の 4 つで、後ろの 3 つは spawn しない。`{{AGMSG_SCRIPTS}}` を持つ雛形は `leader.md` だけ(`implementer.md`・`reviewer.md`・`director.md` に `agmsg` の語がない)。`spawn.go:1009` の「Task, Envelope, and AgmsgHome are only referenced by prompts/leader.md」は合う
- `internal/org/prompts/leader.md` の差分を全文読んだ。節 `## agmsg の受信箱の運用` の Monitor の 1 行を、`bash` と `zsh -f` で自分で走らせて確かめた(スクラッチに stub の `inbox.sh` を置き、実物の `date` と `sleep` を使った)。親の shell が先に終わる形(`/bin/sh -c '<shell> -c "$(cat cmd)" & sleep 1'`)で、どちらも約 11 秒で終わりの 1 行だけを出して消え、`ralph-x leader` を含むプロセスは残らなかった。`$PPID`、`$(( ))`、`kill -0 "$p" 2>/dev/null`、`grep --line-buffered -v '^[[:space:]]*$'` は bash と zsh で同じに読める。`{{TEAM}}` は引用符なしで入るが、org_id が `identifierPattern` を通るので shell の語として安全。`{{AGMSG_SCRIPTS}}` は `'<path>'/inbox.sh` の形で、1 つの語につながる。終わりの 1 行は、本文の `(`agmsg の見張りが終わった。`)` の前半と、`echo` の中身が合う。「待つ」を言う雛形のほかの 7 か所(使命の 4、手順 3・7・12、件を上げたあと、運用規律の 2 項)は、すべて節を指す。`ralph org wait` の語は節と使命の 4 にしかない
- `internal/org/verbs.go` の差分。`errors.Is(ctx.Err(), context.DeadlineExceeded)` は、`Send` が `context.WithTimeout(context.Background(), ...)` で作った ctx を読むので、`herdr: timed out: context deadline exceeded`(試行の文)も、ctx の後で終わる herdr 自身の `--timeout` も拾う。`driver.Herdr.AgentWait`(`herdr.go:402`)は ctx つきで `Run` を呼ぶだけで、ctx より先に herdr の期限が来る道はない。hint を付けるのは待ちの 1 つの失敗だけで、予算の検査、承認の再検査、submit の確かめの文は変えていない。`Send` の doc の追記と `sendWaitDeadlineHint` の doc(plan の D7)は、コードと合う。`Send` を呼ぶ所は `internal/cli/org.go:1178` だけで、エラーの文を読み返す所はない(`wait for seat` は verbs.go のほかにない)
- `internal/cli/org.go` の `ralph org send --help` の追記(1131〜1139)を、`Send` と `confirmSubmitted` の動きと合わせて読んだ。`confirmSubmitted` の doc が先に「ほかの何かが座席を動かしても confirmed と読む」と書いているので、追記と矛盾しない
- テスト。`prompts_test.go`(+300)、`verbs_test.go`、`spawn_test.go`、`internal/cli/org_runtime_env_test.go`、`director_skill_sync_test.go`、`org_skill_send_sync_test.go` の差分を全文読んだ。`deadlineWaitHerdr`(verbs_test.go)の 2 つ目のケース(herdr 自身の `--timeout` が ctx のあとに切れる形)は、hint の判定を `errors.Is(err, ...)` に戻すと落ちる。`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` の全文一致は、hint を常に付ける変更で落ちる。`TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` の 6 ケース(単一引用符と空白、末尾の `/`、相対、環境変数、空)は、`t.Setenv`・`t.Chdir` で `$HOME` と cwd を持ち込まない。`TestRenderRolePrompt_Leader_MonitorCommandRuns` は stub の `date`・`sleep` を PATH の先頭に置き、本物の `sleep` を打たず、zsh を `-f` で起動する(コメントにその理由がある)。skill のテスト 2 つは、4 写しを読み、断片を順に探す。`withoutSpaces` を使うので折り返しに強く、見つからない断片は 1 行ずつエラーになる。`sendWaitDeadlineHint` と skill の英語の断片(`directorSkillSendWaitTimeoutPhrase`)を結ぶ検査もある
- skill の 4 写しは、`director` と `org` のどちらも `cmp` で root の `.claude/skills/` と一致した。`org` の差分は語単位で読み、行(`send`、`approve`)・「受信箱」節の答える順・経路 B の手順 4 と typed protocol 節の段落が、`director` の「答える順」と矛盾しないことを確かめた。typed protocol 節の段落が指す「agmsg の受信箱の段落」は、実際にその節にある。経路 A(現セッション昇格)には leader の座席がなく `ralph org send --to leader` の宛先にならないので、そこが雛形を持たない文のままで構わない
- 文書の差分(仕様、根拠の表、tech-debt、実機の記録)。tech-debt の追加行の `|` を `(?<!\\)\|` で数えると、どの行も 6(見出しと同じ)。F7(a) の行の数字(文字列 5 つ、呼び出し 4 か所、テストが各 1 回)を `escalate.go:435`・`:472` と grep で確かめた(合う)。差分の文書が挙げる Go のテスト名 27 個を `func Test…(` で grep し、すべて存在した。実機の記録は、`$HOME` を `~`、一時ディレクトリを `<D>` と書き、pane・session の id、token を写していない。追加行の `/Users/`・メール・`ghp_`・`sk-`・UUID の grep は、`task-notification` の `sk-notification` に当たっただけ(秘密ではない)
- 追加行を `TODO`・`FIXME`・`XXX`・`fmt.Print`・`println` で grep した。当たりなし。`git diff a58f7c64...HEAD --check` は空、U+FFFD は 0 件
- 確かめていない: Go とシェルのテストの実行(`/test` の範囲)、`go vet`・`gofmt`・`check-sync`・`check-skill-sync`(`/verify` の範囲)、仕様の `[x]`・`[ ]` の数と根拠の表の突き合わせ、実機の記録の数値と手順の再現、skill の文面全体と実装の食い違いの全監査(目についた箇所だけ読んだ)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (F1)<br>fixed in 1183b08e | maintainability | Monitor の 1 行の「親がいなくなったら終わる」が、単体テストでは文字列としてしか固まっていない。`TestRenderRolePrompt_Leader_MonitorCommandRuns` は stub の `date` を 1000、1000、5000 と返し、ループを期限(`end`)で終わらせるので、`kill -0 "$p"` が偽になる分岐を通らない。`leaderMonitorCommandFor`(prompts_test.go:1387)はテンプレートの 1 行を写した文字列で、`AgmsgInboxWaitsByEndingTheTurn` と `AgmsgScriptsIsTheQuotedScriptsDir` はそれと一致を見るだけなので、文字が変わったことは分かるが、`p=$PPID` を `p=$$` に直す変更のような「文字を変えて動きが変わる」誤りは、走らせるテストでしか落ちない。D11 の寿命の保証(古いループが既読を取り合わない)は、実機の確かめ (e) だけが持つ。今のコマンドが正しいことは、レビューで bash と `zsh -f` に打って確かめた(約 11 秒で終わり、残らない) | `internal/org/prompts_test.go:1387`、`:1622`、`:1634-1643`(stub の `date`)。probe: `/bin/sh -c '<shell> -c "$(cat cmd)" & sleep 1'` で親だけ先に終える | `runLeaderMonitorCommand` に、親の shell が先に終わる形(`sh -c '<shell> -c "$CMD" & sleep 1'`)を 1 ケース足し、stub の `date` は固定、`sleep` は 0.1 秒の stub にして、終わりの 1 行が出ることだけを見る。足さないなら、`MonitorCommandRuns` の doc に「親の死は実機の確かめ (e) だけが見る」と書く |
| LOW (F2)<br>fixed in 1183b08e | maintainability | `RenderRolePrompt` は、どの role でも `agmsgScriptsWord(vars.AgmsgHome)` を replacer の引数として評価する。値を使うのは `leader.md` だけなのに、`implementer`・`reviewer` の描画でも環境変数の読み、`os.UserHomeDir`、`filepath.Abs`(`Getwd`)が走る。もう 1 つ、agmsg の置き場所が 2 つの欄(`Org.AgmsgHome` と `driver.Agmsg.Home`)に分かれ、片方が空のときの落ち先(環境変数か既定)は、もう片方の値と一致するとは限らない。今の配線は `internal/cli/org.go:353-366` の 1 か所で、`TestNewOrgRuntimeAt_AgmsgHomeIsTheAgmsgDriversHome` が両方の一致を固めるので、今は崩れないが、`Org` を別の `Agmsg` で組む次の呼び出しは、プロンプトの置き場所だけが静かに環境変数か既定になる | `internal/org/prompts.go:79-86`、`:129`。`internal/org/spawn.go:183-189` | 描画の前に `strings.Contains(text, "{{AGMSG_SCRIPTS}}")` のときだけ値を作る(replacer の引数の評価を遅らせる)。欄が 2 つの件は、`Org.AgmsgHome` の doc に既にある「CLI が同じ値を両方に入れる」で足りるので、直さなくてよい |
| LOW (F3)<br>left (`/pr`) | unnecessary-change | 460e341c の件名が `docs:` なのに、`internal/cli/org.go`(`ralph org send --help` の本文 +9 行、利用者に見える出力)と Go のテスト 2 ファイル(+300 行)を含む。`.goreleaser.yml` の changelog は `^docs:` を除くので、この help の変更と「新しい skill の送り方(25 分の send を裏で)は古い binary でも使える」という新しい skill の変更が、自動の changelog に出ない。plan の「release のノートに書く文」は plan の中にしかなく、`/pr` が plan を archive したあとは探しにくい | `git show --stat 460e341c`。`.goreleaser.yml:33-39`。`docs/plans/active/2026-10-23-busy-leader.md` の「Rollout or rollback notes」 | コミットは直さない(まだ push していないが、並べ替えるほどではない)。`/pr` で、その文を PR の本文に書き、release の issue にもコメントで 1 行足す(`chore-commit-release-note-handoff` の形) |
| LOW (F4)<br>fixed in 1183b08e | readability | `TestRenderRolePrompt_AgmsgScriptsIsTheQuotedScriptsDir` の doc コメントに、折り返しの残りが 1 行ある。`made absolute` だけの行(1556)のあとに `against the working directory, ...` が続く。書き直しの途中で段落を組み直さなかった形 | `internal/org/prompts_test.go:1555-1557` | 1555〜1557 を組み直す(`a relative home made absolute against the working directory, and an empty home read as`) |
| LOW (F5)<br>fixed in 1183b08e (doc, no rename) | naming | `/org` skill のテストが、director の名前の補助関数を使う。`orgSkillInboxAnswerBullet` が `directorSkillSubsection(text, "## 受信箱")` で `/org` skill の節を読み、`TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping` が `directorSkillRepoRoot` を使う。`directorSkillSubsection` の doc(director_skill_sync_test.go:192)は一般の形で書いてあるが、名前を見て grep する読み手は、`/org` のテストが `director` の補助を使うと気づけない。同じ関数は「`#` で始まる次の行」まででしか切らないので、節の中のコードブロックに `#` で始まる行が入ると、節が早く終わる(その場合は断片が見つからずに落ちるので、静かには通らない) | `internal/org/org_skill_send_sync_test.go:63`、`:103`。`internal/org/director_skill_sync_test.go:186-209` | `skillSubsection` と `skillRepoRoot` に名前を変えるか、`directorSkillSubsection` の doc に「`/org` skill のテストも使う」と 1 文足す |
| LOW (F6)<br>withdrawn: no change needed | maintainability | tech-debt の F7 の行が、Related 欄で `docs/plans/active/2026-10-23-busy-leader.md` を指す。`/pr` が plan を `docs/plans/archive/` に動かすので、指す先がなくなる。同じ register の他の行は archive の名前で書く(`docs/plans/archive/` が 226 か所、`active/` の plan 名は、この行の 1 か所だけ。直近の #252 の行も `docs/plans/archive/2026-10-23-watch-escalation-inbox.md`)。実機の記録の `Plan:` 行も同じ形だが、既存の記録(`seat-wip-checkpoint-2026-10-22.md` など)に先例がある | `docs/tech-debt/README.md:327`(Related 欄)。`docs/evidence/busy-leader-2026-10-23.md:4` | register の行は `docs/plans/archive/2026-10-23-busy-leader.md` に直す。実機の記録は、先例どおりでよい |
| LOW (F7)<br>fixed in 1183b08e | maintainability | 800 行の目安を超える既存のファイルが、この PR でさらに伸びた。`prompts_test.go` 2163 → 2463(+300、新しいテスト 4 つと補助 4 つ)、`verbs.go` 2255 → 2277、`spawn.go` 2667 → 2674、`spawn_test.go` 5859(register の数)→ 5904、`director_skill_sync_test.go` 629 → 794(800 まで 6 行)。register に行があるのは `spawn.go`・`spawn_test.go`(351 行目、数は 2665・5859 のまま)だけで、`prompts_test.go`・`verbs.go`・`verbs_test.go`(5854)に行がない。新しい 4 つのテストは、補助(`leaderMonitorCommandFor`、`renderLeaderWithAgmsgHome`、`leaderMonitorCommand`、`runLeaderMonitorCommand`)を自分で持ち、`prompts_test.go` の `markdownSection`・`markdownItem` だけに依る | `wc -l`(上の数)。`docs/tech-debt/README.md:351` | 次の cycle で直さないなら、最後の cycle で 1 行にまとめて足す(`prompts_test.go`、`verbs.go`、`verbs_test.go` と、351 行目の数の更新)。直すなら、新しい 4 つのテストを `prompts_leader_inbox_test.go` に動かす(同じ package、補助はそのまま使える) |

## Positive notes

- `agmsgScriptsWord` の引用が、中身で確かめてある。空白と単一引用符を含む置き場所(`/opt/a b/it's`)が `'/opt/a b/it'\''s/scripts'` になることと、その値が `bash` と `zsh -f` で本物の `inbox.sh` に届くことまでを、同じ fixture(`agmsg it's here`)でテストが打つ
- `Send` の hint の判定が、エラーの文ではなく ctx の状態を読む。文を読む形だと、herdr の文言変更や、ドライバが ctx を包まない経路(テストの 2 つ目のケース)で黙って外れる。時間切れでない失敗の文は、全文一致で変えていないことを固めた
- skill の英語の断片(`the seat's turn did not end within the --timeout-ms deadline`)と `sendWaitDeadlineHint` が、`directorSkillSendWaitTimeoutPhrase` で 1 つの定数に結ばれ、片方だけ変えると落ちる。skill が stderr の文で分岐するという弱い結びつきを、テストが補う形になっている
- 実機の記録が、確かめた事実と確かめていないもの(25 分の期限での終わり、`run_in_background` の gate、`inbox.sh --quiet` の既読の取りこぼし、Codex の leader)を分けて書く。手順(分離した herdr server、shim の osascript、`~` と `<D>` への置き換え)と、触れていないもの(ユーザーの herdr server、`~/.agents/skills/agmsg` の中身)も書く。(f) の 2 回目で、submit の確かめが Monitor の手番の `working` で通ることを見つけて、D12(pane を読んでから resolve)の根拠にした流れが、コードの形(`confirmSubmitted` の doc、help、skill)に残る
- 文を固めるテストの量は大きいが(`AgmsgInboxWaitsByEndingTheTurn` は 25 個ほどの文を逐語で見る)、既存の `FeatureOrgGate` と同じ形で、1 つの文を変えると 1 行だけが落ちる。S1 の 3 つのコミットがそれぞれ文を直し、そのたびにテストの断片も直しているので、文言の変更とテストの変更が同じコミットに揃っている
- tech-debt の F7 の行が、取り消し線で閉じずに「Codex の leader に限る文」に書き換えた形で残り(D10)、RESOLVED のコメントが Claude Code の直しを、効いたこと(実機の数値)と効かせていないこと(25 分の期限、`guarded`、`02e27384` 後の実機)に分けて書く

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| F7: `prompts_test.go`(2463 行)、`verbs.go`(2277 行)、`verbs_test.go`(5854 行)が 800 行の目安を超え、register に行がない。`spawn.go`・`spawn_test.go` の行の数(2665・5859)も古い | 読む・差分を取るコストが上がる | 動かすだけの変更で、この PR の挙動とは別。依頼が report 以外を変えないとしたので `docs/tech-debt/README.md` には足していない(初回のレビュー時点) | 次に `prompts_test.go` の leader の節か `Send` に触る変更、または `prompts_test.go` が 2700 行を超えたとき | `docs/plans/active/2026-10-23-busy-leader.md` |

_(初回のレビューは、依頼どおりこの報告書以外のファイルを変えず、`docs/tech-debt/README.md` には足さなかった。F7 の行は 1183b08e が足した(「Re-review of 1183b08e (S6)」の節)。直さない LOW が最後の cycle に残れば、1 行にまとめて足す。この PR が自分で足した open の行(F7 の Codex の leader、F7(a))は、diff の一部として読み、どちらにも閉じ損ないはなかった。)_

## Coverage gaps

- Go のテスト、シェルのテスト、`gofmt`・`go vet`・`check-sync`・`check-skill-sync` は走らせていない(`/verify` と `/test` の範囲)。このため、`internal/org` と `internal/cli` のコンパイル、`go1.25.8` でのテストの通過、`t.Chdir` の挙動は確かめていない。読んだ範囲では、`t.Chdir`(go 1.24 以上)は go.mod の Go 1.25.8 で使える
- Monitor の 1 行を走らせたのは bash 3.2(macOS の `/bin/bash`)と zsh(`-f`)で、この機械の `PATH` の `date`・`sleep`・`grep` を使った。BSD でない `grep`(GNU)や、Linux の dash は走らせていない。`--line-buffered` は両方の grep にある
- skill の文面が実装と食い違わないかの全監査はしていない。読んだのは、`send`・`approve` の行、「受信箱」節の答える順、経路 B の手順 4、typed protocol 節の段落、`/director` の「答える順」と「計画の承認」の 3・5 と不変条件の行
- 実機の記録の数値(秒数、pid、時刻)を、元の出力と突き合わせてはいない(元は消えている)。記録内の数値どうし(13 分 40 秒、7〜10 秒、8.7 秒、約 1 秒)の整合は、読んで確かめた

## Re-review of 1183b08e (S6)

範囲は `git diff 81f30f43 1183b08e`(5 ファイル、+389/-301)。差分の質だけを見た。テストは走らせていない。

### 初回の指摘の扱い

| ID | 扱い | 根拠 |
| --- | --- | --- |
| F1 | fixed | 新しいテスト `TestRenderRolePrompt_Leader_MonitorCommandEndsWhenItsParentIsGone`(下の「F1 の読み」) |
| F2 | fixed | `prompts.go` の `RenderRolePrompt` が、テンプレートに `{{AGMSG_SCRIPTS}}` があるときだけ `agmsgScriptsWord` を呼ぶ。replacer には空の値のキーが残るだけで、ほかの role の出力は変わらない。`Org.AgmsgHome` と `driver.Agmsg.Home` の 2 欄は、初回に「直さなくてよい」とした |
| F3 | left | `/pr` の手順(PR の本文、release の issue)に回した。この cycle のコードでは直せない |
| F4 | fixed | `made absolute against the working directory, and an empty home read as` が 1 行になった(`prompts_leader_inbox_test.go:193`) |
| F5 | fixed(doc、名前は変えない) | `directorSkillRepoRoot` と `directorSkillSubsection` の doc に「`/org` skill のテストも使う」の 1 文が入った。初回の推奨のどちらか一方(doc の 1 文)にあたる |
| F6 | withdrawn: no change needed | 初回の指摘が誤り。`scripts/archive-plan.sh`(「Rewrite the plan's links in docs/tech-debt/README.md BEFORE moving the plan」の節)が、`/pr` の archive で README の `docs/plans/active/<name>` を `docs/plans/archive/<name>` に書き換える。リンクの直後が名前の文字でなければ当たり、F7 の行の Related 欄の書き方(`.md` の直後がバッククォート)は当たる。register の `active/` の plan 名が 1 か所だけなのは、archive のたびに書き換わる結果で、新しい行だけが残っていた |
| F7 | fixed(この PR の増分について) | この PR の新しい leader inbox のテスト 4 つ、補助 4 つ、定数 1 つが `prompts_leader_inbox_test.go`(379 行、新しいテストと補助 1 つずつを含む)に動き、`prompts_test.go` は 2165 行(base 2163)で、増えたのは既存のテストの文の編集だけ。register に行が足された(`docs/tech-debt/README.md:352`)。もとから 800 行を超えている 5 ファイルの分割そのものは、その行が持つ |

### Evidence

- 動かしたものが逐語であること。`git diff -U0 81f30f43 1183b08e -- internal/org/prompts_test.go` の削除行 298 行と、新ファイルの移動部分(17〜307 行)を順に `diff` した。差は、旧 `prompts_test.go` から外れた import 6 行(`context`、`os`、`os/exec`、`path/filepath`、`time`、`driver`)と、F4 の組み直し 1 か所だけ。`prompts_test.go` に残る import(`errors`、`regexp`、`slices`、`strings`、`testing`、`config`、`protocol`)はどれも使われ、外した 6 つの接頭辞は `prompts_test.go` に 0 件。新ファイルの import 10 個もそれぞれ使われる。移動した補助(`testRolePromptVars`、`markdownSection`、`markdownItem`、`containsPhrase`、`squashSpace`、`renderSeatPrompt`、`writeTestExecutable`)は同じ package に残り、新ファイルから届く。`prompts_test.go` に、動かした側の名前(`leaderMonitorEndLine`、`renderLeaderWithAgmsgHome` など)を使う所はない
- tech-debt の新しい行(`docs/tech-debt/README.md:352`)。数は `git show a58f7c64:<path> | wc -l` と `wc -l` に一致する(verbs.go 2255 → 2277、verbs_test.go 5762 → 5854、prompts_test.go 2163 → 2165、spawn.go 2667 → 2674、spawn_test.go 5859 → 5904)。行の `|` は 6 個。引くテスト 2 つは存在する。「prompts_test.go が増えたのは既存のテストの編集だけ」は +2 行と合う
- `director_skill_sync_test.go` の差分は doc の 2 文だけで、コードは動かしていない
- `git diff 81f30f43 1183b08e --check` は空、U+FFFD は 0 件

### F1 の読み

新しいテストの `/bin/sh -c '"$MONITOR_SHELL" -c "$MONITOR_CMD" & /bin/sleep 1'` の形を、全文と照らして読んだ。

- 期限では終われない。stub の `date` は 1000 を返し続けるので、`end` は来ない。ループが終わる道は `kill -0 "$p"` が偽になることだけ
- sh がゾンビで残らない。`Cmd.Wait` は `Process.Wait`(刈り取り)のあとでパイプの EOF を待つ(`$GOROOT/src/os/exec/exec.go:930`、`:948`、go1.26.0)。ループがパイプを持ち続けるあいだも sh は刈り取り済みで、`kill -0` は pid が消えたとして失敗する。doc の説明はこの順と合う
- 落ち方。`CommandContext` の 30 秒は刈り取り済みの sh にしか効かないので、テストが自分の `select` で `Kill(-pgid, SIGKILL)` を打ち、`<-done` で Wait を終わらせてから `Fatalf`。`Setpgid` があるので、ループと子のプロセスが残らない
- `args: ralph-org-a leader --quiet` が出力にあることを確かめるので、`$PPID` が最初から死んでいて 1 回も回らずに終わる形では通らない
- `p=$PPID` を `p=$$` にする変更と、`kill -0 "$p"` の項を外す変更は、どちらも親の確認を失ってループが期限なしで回るので、30 秒で落ちる。これは読んで確かめた(実行はしていない)。実装者の報告(`p=$$` で 2 つのサブテストが 30 秒で落ちた)と合う。初回に自分で打った親が先に終わる形の probe(約 11 秒、残らない)と、テストの形は同じ
- 対象は darwin と linux(`.goreleaser.yml`)で、`syscall.Kill`・`Setpgid` を使うテストは既存にある(`director_cmd_test.go`、`gate_test.go`、`mergeorder_test.go`)ので、build tag は要らない

観察(指摘にしない): 新しい補助 `runLeaderMonitorCommandUnderAnExitingParent` と、`runLeaderMonitorCommand` は、`bash`・`zsh` を回すラッパー、stub の置き場、zsh の `-f` の分岐を写している(十数行)。分けた理由(1 つは deadline、1 つは親の死)が違うので、共通にするほどではない。もう 1 つ、親の sh が 1 秒で終わるので、負荷が極端に高く Monitor の shell が 1 秒以内に `p=$PPID` を読めないと、`args:` の行がなくて落ちる。誤って通る向きではなく、起きる見込みは小さい

### 新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (F8) | maintainability | 1183b08e がテストを `prompts_test.go` から `prompts_leader_inbox_test.go` に動かしたが、動かしたテストの置き場所を `internal/org/prompts_test.go` と書く 5 か所が古いまま残った。テスト名は合い、ファイル名だけが違う。新しい register の行(352)は新しい名前で書いてあるので、同じ commit の中で 2 つの書き方が並ぶ。`/sync-docs` の範囲にも入るが、この commit が作った食い違いなので、ここで挙げる | `docs/specs/2026-10-07-org-multi-org-director.md:92`(FR-5 の印)と `:151`(NFR ツール非依存の印)、`docs/evidence/org-spec-evidence-2026-10-21.md:51`(FR-5 の行)と `:59`(NFR ツール非依存の行)、`docs/tech-debt/README.md:326`(F7 の RESOLVED コメントの「Tests:」)。どれも `TestRenderRolePrompt_Leader_AgmsgInboxWaitsByEndingTheTurn`(と、spec 92・evidence 51・register 326 では `MonitorCommandRuns`)の前に `internal/org/prompts_test.go` を置く。新しい場所は `internal/org/prompts_leader_inbox_test.go:74`、`:258` | その 5 か所の `internal/org/prompts_test.go` を `internal/org/prompts_leader_inbox_test.go` に直す(同じ行にある別のテスト、たとえば evidence の 50・57 行の `TestRenderRolePrompt_Leader_FeatureOrgProcedure` と `EscalatesThroughRalphOrgEscalate` は動いていないので触らない)。この修正は spec・根拠の表・register の本文なので、`post-implementation-pipeline.md` の記録だけの例外の対象ではない |

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2 件が未了: F3(`/pr` の手順)と F8(新規、S6 が動かしたテストを指す文書 5 か所)。F1・F2・F4・F5・F7 は 1183b08e で直り、F6 は取り下げ。初回は LOW 7)
- Follow-ups: 未了は F3 と F8 の 2 件で、どちらも LOW。F8 は 5 か所の文字の置き換え(`internal/org/prompts_test.go` → `internal/org/prompts_leader_inbox_test.go`、同じ行の別のテストは触らない)で、spec・根拠の表・register の本文なので、`post-implementation-pipeline.md` の記録だけの例外の対象ではない。直すなら `/sync-docs` の範囲として扱うか、`/self-review` から回し直す。F3 は `/pr` の手順(PR の本文と release の issue の 1 行)。この PR が足す register の行(800 行の行、試行の所見 F7 の Codex の leader、self-review F7(a))に閉じ損ないはない。insight event は未実行(依頼が report 以外を変えないとしたため。F8 を直さずに進めるなら `--low 2`、直したあとなら `--low 1`): `./scripts/insights-append.sh --slug busy-leader --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 2 --cycle auto --source skill || true`
