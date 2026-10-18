# Self-review report: org-director-wake

- Date: 2026-10-18
- Plan: `docs/plans/active/2026-10-18-org-director-wake.md`(ステージ 8b-1、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 9e92d92822d78d29e96f86e3c60114389bdcf6ec...HEAD`(HEAD 3023dcd5、50 ファイル、+6627/-300)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない

## Evidence reviewed

- 全文を読んだ: `internal/org/director_wake.go`(新規、412 行)
- diff を全部読んだ: `internal/org/director.go`、`inbox.go`、`watch.go`、`director_start.go`、`director_cmd.go`、`deadline.go`、`internal/config/config.go`、`internal/cli/org.go`、`org_director.go`、`org_director_headless.go`、`org_inbox.go`、`org_watch.go`、`status.go`
- 周辺の本体を読んだ: `PrintableInboxText`、`inboxIDNumber`、`directorIDNumber`、`foldDirector` の attached・pane_created の枝、`confirmSeatPane`、`callWithTimeout`、`driver.Herdr.PaneSendText`/`PaneSendKeys`、`runDirectorCmd`、`DirectorStore.readRaw`
- 文書と設定: `/director` skill の diff、`internal/org/prompts/director.md`、`docs/recipes/director-wake.md`(Security と失敗の節)、`templates/base/ralph.toml`、`scripts/ralph-config.sh`、`docs/tech-debt/README.md`(変わった 4 行を単語単位で)、`docs/evidence/director-wake-livefire-2026-10-18.md` の冒頭と末尾
- 既存テストの変更: `director_test.go`、`org_director_test.go`、`org_director_headless_test.go`、`prompts_test.go` の diff。足した引数と期待文字列だけで、期待を弱めた箇所はない
- 新規テストは名前の一覧と、`ValidateDirectorWakeTarget` の表、`TellsWhatTheWakeLineIs` を読んだ。全ケースは読んでいない
- 機械的な確認: 追加行に TODO・FIXME・debug の出力・行末の空白はない。skill の 4 面、recipe の 2 面、`ralph-config.sh` の 2 面は `cmp` と `diff` で一致。tech-debt の変わった 4 行は、`|` の数が前後で 6 のまま(列が増えていない)
- 依頼された重点(注入、stderr の escape、送り先の検証、手で書き換えた `director.jsonl` と `inbox.jsonl`)は、下の「確認して問題がなかったもの」にまとめた

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 1 件、LOW 7 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | security | M1: `{target}` の形の検査は、シェルの記号を通す。一方、文書は `{text}` にシェルの記号がないことだけを言い、`{target}` をシェルの文字列に入れてはいけないとはどこにも書いていない。`wake_target` は leader が書ける `director.jsonl` から来る(計画の Risks が認めている)ので、利用者が `wake_cmd = ["sh", "-c", "... {target} ..."]` と書くと、leader が `$(...)`、`;`、バッククォートを送り先に入れて、見張りが leader の sandbox の外でそれを実行できる。D2 は最初の要素に placeholder を許さないことで「送り先に打つものを選ばせない」を守ったが、同じ理由が引数の文字列にも当てはまる。launch_cmd には `{script_sh}`(シェル向けに引用した値)と「shell string の端末」の注意があるのに、wake_cmd にはどちらもない | `director.go:364`(`ValidateDirectorWakeTarget` が拒むのは空、不正な UTF-8、制御文字、257 文字以上、先頭の `-` だけ)。`director_wake_test.go:49-52` は `a b`、`a{target}` を通す例で、`;` `$` はどちら側にもない。`docs/recipes/director-wake.md:273-290`(Security は `{text}` の「シェルの記号なし」と target の形の検査を書くが、`{target}` の注意はない)、`config.go:240-273`(`WakeCmd` の doc)、`templates/base/ralph.toml:181-194`。対照: `config.go:198-201` と `templates/base/ralph.toml:159` の launch_cmd | recipe の Security、`OrgDirectorConfig.WakeCmd` の doc、`templates/base/ralph.toml` の `wake_cmd` のコメントに「`{target}` は `;`、`$`、バッククォートを含みうる。シェルを通す引数(`sh -c` の文字列、`tmux` の shell-command)には入れず、argv の 1 要素か、`"$1"` と引用して読むスクリプトの引数にする」を足す。tmux の id(`$1`、`%3`、`@2`)があるので、文字を絞る案は取りにくい。代わりに launch_cmd の `{script_sh}` に倣って `{target_sh}` を足すか、見送るなら tech-debt の 1 行にする |
| LOW | exception-handling | L1: herdr の既定の送り方は、`send-text` が通ったあとに Enter の `send-keys` が失敗すると、行は director の入力欄に入ったままで `woken` を書かない。5 分後の再試行はその上にもう 1 度同じ行を打つので、director は 1 つのメッセージに行が 2 回入ったものを受ける(意味は壊れない)。エラー文は「after the line was typed」と言うが、recipe の失敗の節はこの途中の状態を言わない | `director_wake.go:400-410`(`PaneSendText` のあと `time.Sleep`、`PaneSendKeys`)、`director_wake_test.go:1273`(文言だけを固定)、`docs/recipes/director-wake.md:255-271` | recipe の失敗の節に 1 文(Enter だけが失敗すると、行は入力欄に残り、次の試しがもう 1 度打つ)を足す。コードで直すなら、Enter の失敗だけをその場で 1 回やり直す |
| LOW | maintainability | L2: 「wake_cmd のどこかの要素が `{target}` を含むか」の判定が 2 か所にある。`directorWakeCmdHasTarget` を作ったのに、`CheckDirectorAttachWake` は同じ式をインラインで書く。helper の doc は「`CheckDirectorAttachWake` の判定と同じ」と書いていて、片方を直すともう片方を直し忘れやすい | `director.go:412`(`slices.ContainsFunc(wakeCmd, func(arg string) bool { return strings.Contains(arg, config.DirectorWakePlaceholderTarget) })`)、`director_wake.go:341-343` | `CheckDirectorAttachWake` が `directorWakeCmdHasTarget(wakeCmd)` を呼ぶ |
| LOW | naming | L3: launch_cmd と wake_cmd が共有する部品に、launch の名前が残っている。`directorLaunchPlaceholderPattern` を `ValidateDirectorWakeCmd` と `ExpandDirectorWakeArg` も使い、`defaultDirectorLaunchWaitDelay` を wake_cmd の runner も使う。doc は共有を説明するが、`grep directorLaunch` で wake_cmd の経路は見つからない | `config.go:312-317`(定義)、`:334`、`:350`、`:383`、`:477`(使い手)。`director_cmd.go:399-405`(`directorLaunchOutputLimit`、`defaultDirectorLaunchWaitDelay`) | `directorCmdPlaceholderPattern` のような launch を含まない名前に改める。doc が理由を述べるのでコメントの直しだけでもよい |
| LOW | readability | L4: 折り直しの取りこぼしが 2 か所と、定数の説明の置き場所が 1 か所。(a) `DirectorStore.Attach` の doc に「appends the current」だけの短い行が残った。(b) `withMainWorktreeDirectorEnabled` の doc に「to load is an error, which」だけの短い行が残った。(c) `InboxEventWoken` の説明が、定数の行ではなくブロックの doc の 2 行目にあり、`InboxEventEscalated` などと並べた形から外れる | (a) `director.go:553`、(b) `org.go:215`、(c) `inbox.go:28-35` | (a)(b) 段落を折り直す。(c) 説明を `InboxEventWoken` の行の上に移す |
| LOW | maintainability | L5: 登録の wake に関する部品が、すでに大きい `director.go` に入った。`ValidateDirectorWakeTarget`、`CheckDirectorAttachWake`、エラー変数 4 つ、`WakesByPush`、`ResolvedWakeTarget` を足して、`director.go` は 844 行から 1031 行になった。`org_director.go` は 727 行から 867 行で 800 行を超えた。新しい `director_wake.go`(412 行)が、wake の送り手の側だけを持つ。`WakeDirector` も 64 行ある(読む、選ぶ、送る、記録するの 4 段が 1 つの関数に入っている) | `director.go`(1031 行)、`org_director.go`(867 行)、`config.go`(596 行から 789 行。800 行の手前)、`director_wake.go:216-279` | 次にこの 3 ファイルへ足す前に、attach の検査(`ValidateDirectorWakeTarget`、`CheckDirectorAttachWake`、`ErrDirectorWake*`)を `director_wake.go` に移す。今回の PR で動かすかは任意 |
| LOW | readability | L6: 「main worktree の `ralph.toml` から読む設定」の列挙に、新しい 3 つ目の使い手の `ralph org watch --all` がない。`withMainWorktreeOrgLimits` の doc は、escalate、attach、status、merge、start の順に使い手を並べるが、`orgWatchDirectorWake`(`withMainWorktreeDirectorWakeConfigFrom` 経由)は載っていない。`loadMainWorktreeOrgConfig` の doc と `--config` の help には載っている | `org.go:177-186`(列挙)、`org_watch.go:155-188`(`orgWatchDirectorWake`)、`org.go:311-315`(`loadMainWorktreeOrgConfig` の doc は載せている) | 列挙に「`watch --all` は `Director.Wake` と `Director.WakeCmd` を `orgWatchDirectorWake` で読む」を足す |
| LOW | maintainability | L7: テストの 3 つのコメントが、計画の中だけにある slice の名前 `S3b` を引く。ほかのコメントは plan の slug と D・AC の番号だけを引き、計画は `/pr` でアーカイブされる。D・AC の番号は計画に残るが、`S3b` は Progress checklist を読まないと分からない | `internal/cli/org_watch_test.go:1666`、`internal/org/watch_all_test.go:1492`、`internal/org/director_wake_test.go:441` | `S3b: ` を外して D・AC の番号だけにする |

## 確認して問題がなかったもの

依頼された重点について、読んだ範囲で見つかった欠陥はない(M1 は文書の欠落で、コードの欠陥ではない)。

- 起こす文(D7): `DirectorWakeText` は件の ID だけから作る。ID は `foldInbox` が `inboxIDNumber`(`e` と正の 10 進数の正規形)で通したものだけなので、手で書き換えた `inbox.jsonl` が変えられるのは、どの件を名指すかだけで、文の字面ではない(`woken` を足して起こさせないことはできる)。本文、TYPE、org_id、送り先は文に入らない。文にシェルの記号はなく、先頭は `/`、`!`、`#` でない。文の字面はテストが固定している
- wake_cmd の置き換え: 1 回の `ReplaceAllStringFunc` で置くので、値の中の `{...}` は置き直されない。最初の要素の placeholder は `Load` と、起こすたびの `ValidateDirectorWakeCmd`(`directorWakeSender`)で断る。`{text}` が最初の要素に入る道はない。シェルを通さない `exec.CommandContext` で、`cmd.Dir` は絶対パスの設定のディレクトリに固定する。相対の実行ファイルはそこから解く(`director_wake.go:371-373`)
- 送り先の検査: attach(CLI と store の 2 段)と、見張りが使う直前(`directorWakeSender`)の 3 か所が同じ `ValidateDirectorWakeTarget` を使う。herdr の pane id も、`confirmSeatPane` で label を確かめる前に同じ検査を通す。`herdr pane send-text` と `send-keys` は argv で呼ぶので(`driver/herdr.go:286-296`)、先頭が `-` でない pane id と固定の文なら、オプションに読まれない
- 手で書き換えた `director.jsonl`: fold は `directorIDNumber` を通らない id と、token の形に合わない headless の attached を数えて無視する(`director.go:876-895`)。headless の `PaneID` は `pane_created` の最初の値だけで、attached に書いた pane_id は読まない。未知の launch は昇格したセッションとして扱い、送り先は `wake_target` だけ、wake_cmd がなければ「送る手段なし」で止まる。`wake` に `push` を足しても、送れるのは利用者が main の `ralph.toml` に書いた wake_cmd か、label を確かめた herdr の pane だけ
- stderr の escape: `writeDirectorWakeLine` は 1 行まるごと `PrintableInboxText` を通す。`runDirectorCmd` は実行ファイルの名前と出力の末尾を escape する(二重に通っても、先頭の escape が `\x1b` の文字列になっているので壊れない)。`inbox show`、`status`、`director status` の新しい値は `inboxField` を通す。`watch.go` の `wake()` が escape なしで出す `res.Err` は、`director.jsonl` と `inbox.jsonl` を読めなかったときのパスだけを持つ(中身は含まない)
- 失敗の間隔: メモリは見張りの中だけ。`recordSuccess` の `delete` は nil の map でも安全。5 分の保留は受信箱を読んだあとに判定するので、起こす件がないサイクルは何も出さない。`AppendWoken` の失敗は保留に入るので、壊れた受信箱で毎サイクル打つことはない
- `woken` の書き込み: 1 回のロックで、読み直し、未知の ID の拒否、1 回の write。末尾が改行でない行を別の壊れた行として残す既存の保護は、複数 event の形でも保たれる(`appendLocked`)。期限の計算は `woken` を読まない(`deadline.go` の doc とコードが合う)
- 既存の動きの保存: `ValidateDirectorLaunchCmd` の書き換えは、エラーの文と順(未知の placeholder が先、script がないのが後)を変えない。`runDirectorLaunchCmd` は `runDirectorCmd` の薄い包みで、エラーは `launch_cmd` で始まるまま。`isDirectorLaunchPlaceholder` の参照は残っていない
- 見張りの順: 期限の送りを先に、起こす手順を次に置き、doc とコードが合う。`--detach` の親が main の `ralph.toml` を読めないときに断るのは、spawn や start が同じ `ralph.toml` で既に断る範囲に収まる
- evidence: `$HOME` は `~`、token と pane は伏せてある。16 進の 16 文字は利用者の herdr の状態ファイルの sha256 の先頭で、認証情報ではない
- 機密、debug の出力、未使用の import やコード: なし。recipe の script の例は `"$1"` `"$2"` と引用している

## Positive notes

- 設定の読み方を 3 つの動詞(attach、start、watch)で同じ規則にし、`...From` が読んだファイルを返して、wake_cmd を打つ場所を「規則を読み直さず」決めている。feature worktree から前景で立てた見張りの穴(計画の Codex 助言 HIGH)は、作業ディレクトリではなく読んだ設定のディレクトリで塞がっている
- 「送れた」と「記録できた」を分け、送ったあとに `woken` を書けなかった場合を失敗として保留に入れ、そのことを行にも書く。重なる起こしは 1 回までという D6 の受け入れを、コードの形で守っている
- 失敗と「送る手段がない」を別々に扱い、後者は登録ごとに 1 度だけ言う。受信箱を読んでから保留を判定する順で、保留中のサイクルは静かになる
- 実機の確認で、待っていない codex が行を新しい手番として受け、手番の途中の行は今の手番に入ることを確かめ、skill と recipe に「Claude Code は確かめていない」と書き分けている

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この cycle では、M1 と L1〜L7 すべてを cycle 内の直しとして勧める) | | | | |

_(今回は台帳に行を足していない。コミットの範囲が、この report と insight event の 2 ファイルに決まっているため。上限の cycle で M1 や L1 が直らないまま残る場合は、1 行にまとめて `/sync-docs` が足す。)_

## Recommendation

- Merge: yes(CRITICAL・HIGH なし。MEDIUM 1 件(M1)は、`{target}` が利用者のシェルの文字列に入ったときの注意が文書にない点で、コードの欠陥ではない。LOW 7 件は重複した判定、命名、折り直し、列挙、ファイルの大きさ、コメントの slice 名、Enter の部分失敗の文書で、動きを変えない。注入、escape、送り先の検査、手で書き換えたファイルへの耐性に欠陥は見つからなかった)
- Follow-ups:
  1. M1: recipe の Security、`WakeCmd` の doc、`templates/base/ralph.toml` に `{target}` の注意を足す。動きは変わらない。recipe は root と `templates/base/` を同じ内容に保つ
  2. L2 と L3 は Go のコードに触れる数行の直し。L4、L6、L7 はコメントだけ。L1 は文 1 つか、Enter のやり直し 1 回。L5 は任意
  3. どれも、この task の記録(report と insight event)だけの直しではないので、`post-implementation-pipeline.md` の記録だけの修正の例外には入らない

## Re-review (S6, 2129f422..705e56bf)

- Reviewer: reviewer サブエージェント(`/self-review` の再実行。パイプラインの cycle は増やさない)
- Scope: `git diff 2129f422..705e56bf`(705e56bf の 1 コミット、15 ファイル、+406/-296)。cycle-1 の M1 と L1〜L7 の直しと、その直しが持ち込んだものだけを見た。diff の品質だけを見て、テスト、静的解析、整形は走らせていない
- 読んだもの:
  - 15 ファイルの diff を全部読んだ。新規の `internal/cli/org_director_wake.go`(105 行)は全文、`internal/org/director_wake.go` は `WakeDirector` の前後と移ってきた部分を読んだ
  - 移動の突き合わせ: `git diff -U0` の消えた行と増えた行を、行の文字列そのまま(tab も)の多重集合で比べた。`director.go` から `director_wake.go` へ: 消えた行のうち増えた側にないのは、L2 の 1 行(`case push && target == ""` の式)、pointer の 1 行(`// CheckDirectorAttachWake, and by StartDirector.`)、L4(a) の折り直しの 12 行だけ。`org_director.go` から `org_director_wake.go` へ: 消えた行はすべて増えた側にある。定義の順も diff の並びで同じ。移った 10 個の名前は、repo に 1 か所ずつしか定義がない(`grep -rn`)。`director.go` が外した import は `unicode` と `unicode/utf8` だけで、残る import は本文で使われている(`slices`、`strings`、`sort`、`regexp`、`config`)。`org_director.go` の `strings` と `config` も同じ
  - `WakeDirector` の分割: 前の `2129f422:internal/org/director_wake.go` の本体と、`WakeDirector`、`directorToWake`、`sendDirectorWake` を並べて、読む順、skip の種類、返す result の項目、banner の文を 1 つずつ突き合わせた
  - M1 の文面は、recipe の `wake_cmd` の例、`/director` skill、`launch_cmd` の `{script_sh}` の扱いと突き合わせた
  - 機械的な確認: `git diff --check` は空。追加行に TODO・FIXME・print・`/Users/`・U+FFFD はない。recipe の 2 面は `cmp` で一致し、diff も同じ。tech-debt の変わった行は 2 つ(239 行目の改名と、末尾 301 行目の新しい行)で、301 行目は 5 セル、`|` の漏れはない

### 前回の finding の判定

- M1(`{target}` をシェルの文字列に入れる危険): 直った。注意は 4 面にある。recipe の Security(`docs/recipes/director-wake.md:293-302`)と tmux の例の 1 文(`:180-183`)、`OrgDirectorConfig.WakeCmd` の doc(`config.go:274-281`)、`templates/base/ralph.toml:194-200`。どれも、形の検査だけであること、`;` `$` バッククォート 空白を通すこと(tmux の id が要る)、シェルが読む文字列に入れず argv の 1 要素か `"$1"` と引用する script に渡すこと、を言う。recipe の例はこの規則に合う(`{target}` は argv の 1 要素で `:58`、`:158`、`:177`、`:220`。`:192` は script に渡し、script は `"$1"` と引用している(`:199-201`))。`sh -c` に入れた例はない。選択肢の `{target_sh}` は見送り、tech-debt 301 行目に載せた。5 セルで、Trigger は「シェルの文字列に `{target}` が要る `wake_cmd`、またはその報告」。行が指す `shellQuote`(`internal/org/feature.go:408`)は `{script_sh}` の値を作る関数(`director_cmd.go` の `directorLaunchArgv`)で、行の説明と合う。Related の plan の path は `/pr` の `archive-plan.sh` が書き換える。新しい問題はない
- L1(Enter だけの失敗で行が 2 回入る): 直った(文書の側)。recipe `:266-270` に、行は入力欄に残り、5 分後の再試行がもう 1 度打ち、意味は変わらない、と書いた。括弧の引用 `after the line was typed` は `director_wake.go:571` のエラー文と一致する。コードで Enter だけをやり直す案は取っていない
- L2(`{target}` を含むかの判定が 2 か所): 直った。`CheckDirectorAttachWake` は `directorWakeCmdHasTarget(wakeCmd)` を呼ぶ(`director_wake.go:135`)。式は `:503` の 1 か所で、呼び手は `:135` と `:473` の 2 つ。helper の doc の呼び手(「for CheckDirectorAttachWake and directorWakeSender」)と合う。前の「as CheckDirectorAttachWake's does」は消えた。同じ理由の文が 2 つの doc に残った点は R2
- L3(共有する部品に launch の名前): 直った。`directorCmdPlaceholderPattern`(`config.go:327`)と `defaultDirectorCmdWaitDelay`(`director_cmd.go:405`)に改め、使い手と doc、tech-debt 239 行目も直っている。古い名前は、コード、doc、register のどこにも残らない(repo 全体を grep。残りはこの report の L3 の行と、過去の `sync-docs-2026-10-16-org-director-launch-cmd.md` だけで、時点の記録なので直さない)。`directorLaunchOutputLimit` は launch の名前のまま。使うのは `runDirectorLaunchCmd` だけで、wake_cmd は自分の `directorWakeOutputLimit`(`director_wake.go:173`)を持つので、共有の部品ではなく、名前は合う
- L4(折り直しの取りこぼし): 直った(3 か所とも)。(a) `DirectorStore.Attach` の doc は 1 段落に折り直された(`director.go:429` から)。(b) `withMainWorktreeDirectorEnabled` の doc も同じ(`org.go:217-222`)。(c) `InboxEventWoken` の説明は定数の行の上に移った(`inbox.go:32-34`)。定数の並びはコメント行で揃えが 2 つの塊に分かれ、後ろの 3 行の桁が詰まっているので、整形済みの形に見える(`gofmt` は走らせていない)。折り直した行は 77 桁以内で、周りと同じ幅
- L5(wake の部品の置き場所と `WakeDirector` の長さ): 直った。attach の検査(`ValidateDirectorWakeTarget`、`CheckDirectorAttachWake`、`ErrDirectorWake*`、`MaxDirectorWakeTargetRunes`、`WakesByPush`、`ResolvedWakeTarget`)は `director_wake.go` に、CLI の 4 関数は新しい `org_director_wake.go` に移った。行数は `director.go` が 1031 から 906、`org_director.go` が 867 から 780、`director_wake.go` が 412 から 574。`director.go` はまだ 800 行を超える(PR の前は 844)。`config.go` は M1 の doc で 789 から 799 行になり、800 行の 1 行手前に来た。`WakeDirector` は 15 行、`directorToWake` は 32 行、`sendDirectorWake` は 27 行
- L6(`withMainWorktreeOrgLimits` の使い手の列挙): 直った。`org.go:182-185` に `ralph org watch --all` と `orgWatchDirectorWake`(`withMainWorktreeDirectorWakeConfigFrom` 経由)が入り、`orgWatchDirectorWake` の doc(`org_watch.go:155-166`)と合う
- L7(コメントの slice 名 `S3b`): 直った。3 か所(`org_watch_test.go:1666`、`watch_all_test.go:1492`、`director_wake_test.go:441`)から外れた。この PR の追加行を `S<数字>` で grep しても、残るのは前の PR の slice を引く tech-debt の行だけ

前回の「Tech debt identified」は「なし」だった。M1 の `{target_sh}` だけが直しの中で register の 1 行(301 行目)になり、ほかは直った。

### 移動と分割が動きを変えないこと

- 移動は純粋。コードの行は L2 の 1 行を除いて同じ。変わった文は、`director.go:74-76` の pointer(`director_wake.go` にあるものの名前を並べた)、`director_wake.go:18-22` の新しい先頭の説明、既存の説明の「The registration's wake and target are in director.go」を「are read with WakesByPush and ResolvedWakeTarget above」に直した 1 文、`org_director_wake.go:13-18` の先頭の説明
- `WakeDirector` の分割は動きを変えない。順は、director.jsonl を読む、登録がなければ `no_registration`、push でなければ `no_push`、headless で running でなければ `not_running`、inbox の store がなければ Err、inbox を読む、起こす件がなければ `nothing_to_wake`、`mem` と `banner` の既定、`holdsBack`、sender の選択、送信、`AppendWoken`、`recordSuccess`。前も後もこの順で、`directorToWake` が前半を、`sendDirectorWake` が後半を持つ。各 return の result の項目(`RegistrationID`、`IDs`、`Skipped`、`HeldBack`、`Err`)も前と同じ値になる。`sendDirectorWake` の本体は、diff では文脈の行になる(banner の 3 つの文、`Failure`、`Sent`、`Via`、`Recorded` の付け方は変わらない)。変わった行は `o.directorWakeSender(*reg, ...)` が `reg` の値渡しになった 1 行だけ
- cycle-1 の表の `file:line` は、この直しで動いた(`director.go:364` と `:412` は `director_wake.go` へ、`org_director.go` の 4 関数は `org_director_wake.go` へ)。cycle-1 の行番号は 3023dcd5 時点のままで、この節の行番号は 705e56bf で引き直した

## Re-review findings (S6)

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 4 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | R1: `sendDirectorWake` は `mem` と `banner` が nil でないことを前提にするが、doc がそう書いていない。nil の既定は `WakeDirector` が 2 つの helper の間で入れる。`DirectorWakeMemory` の method は nil の受け手で止まる(`holdsBack`、`recordFailure`、`warnOnce` はどれも `m` のフィールドに触れる)。呼び手は今 1 つだけだが、分割で「既定を入れる関数」と「使う関数」が分かれた。あわせて `WakeDirector` の step 4 の文(「skips a registration that has none」)は、skip の種類を sender がない場合だけにしていて、前の文(wake_cmd が不正、送り先が使えない、sender がない)より狭い | `director_wake.go:347-350`(既定)、`:414`(`sendDirectorWake`)、`:258-278`(3 つの method)、`:328-330`(step 4 の文)。前の文は `git show 2129f422:internal/org/director_wake.go` の step 4。広い側の文は `sendDirectorWake` の doc と `DirectorWakeSkipped*` の doc にある | `sendDirectorWake` の doc に「`mem` と `banner` は nil でない(WakeDirector が既定を入れる)」を足す。step 4 は「sender を選び(wake_cmd か herdr の pane)、なければ、また wake_cmd か送り先が使えなければ、登録ごとに 1 度だけ banner に言って skip する」と言い直す |
| LOW | maintainability | R2: 「`{target}` の判定は部分文字列で、`config.ValidateDirectorWakeCmd` の placeholder の規則が見つけるものと同じ」という理由の文が、`CheckDirectorAttachWake` の doc と `directorWakeCmdHasTarget` の doc の両方にある。L2 の直しで式は helper の 1 か所になったが、理由は 2 か所に残り、片方を直し忘れやすい形が式から doc に移った | `director_wake.go:122-124`(`CheckDirectorAttachWake` の doc)、`:499-502`(helper の doc) | `CheckDirectorAttachWake` の側は「`{target}` の判定は `directorWakeCmdHasTarget`」と指すだけにして、理由は helper の doc に 1 つ置く |
| LOW | readability | R3: `org_director_wake.go` の先頭の説明の最後の文「newOrgDirectorAttachCmd (org_director.go) registers the two flags and calls these」は、4 関数のうち 3 つに当たる。`orgDirectorPushDetails` を呼ぶのは `orgDirectorLine`(`ralph status` と `director status` の `director:` 行)で、`newOrgDirectorAttachCmd` ではない。同じ説明の前の文は `director:` 行を挙げているので、読み手は呼び手を取り違える | `org_director_wake.go:16-18`、`org_director.go:137`、`:150`、`:164`(3 関数の呼び出し)、`:622`(`orgDirectorPushDetails` の呼び出し。関数は `orgDirectorLine`、`:606`) | 「calls the first three; orgDirectorLine (org_director.go) calls orgDirectorPushDetails」のように呼び手を分ける |
| LOW | naming | R4: `org_director_wake.go:13-14` は plan を `docs/plans/active/2026-10-18-org-director-wake.md` の path で引く。`/pr` が plan を archive へ動かすとこの path は消え、`archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` の links だけ(`scripts/archive-plan.sh:54-65`)。同じ commit が触った `director_wake.go` は slug の形(`plan 2026-10-18-org-director-wake`、`:18`、`:29`、`:59`、`:78`)で引く。同じ腐りは repo にもう 1 つある。`org_director.go:21` が引く `docs/plans/active/2026-10-15-org-director-start.md` は、いま `docs/plans/archive/` にある。この PR のテストの先頭の説明 3 か所も path の形で、cycle 1 では見落とした | `org_director_wake.go:13-14`、`director_wake_test.go:20`、`org_director_wake_test.go:18`、`prompts_test.go:1607`、`org_director.go:21` | この PR の 4 か所を `plan 2026-10-18-org-director-wake` の形に揃える。`org_director.go:21` はこの PR の外なので触らなくてよい |

### Tech debt identified

新しく足す行はない。R1 から R4 は、Go のコードを変えないコメントだけの直し。直さないまま最後の cycle に入るなら、1 行にまとめて `/sync-docs` が足す(Trigger は `director_wake.go` か `org_director_wake.go` の次の変更)。M1 の残りの `{target_sh}` は、直しの commit が tech-debt 301 行目に載せた。

_(上の変更は report だけで、`docs/tech-debt/` には追記していない。同じ cycle の再レビューなので insight event も足していない。phase と cycle が集計の鍵なので、`self_review` の cycle 1 の行がもう 1 つ増えると二重に数える。)_

### Re-review の判定

cycle-1 の M1 と L1〜L7 は 8 件とも直った。この再レビューで足した指摘は LOW 4 件(R1〜R4)で、CRITICAL、HIGH、MEDIUM はない。cycle-1 の件数(MEDIUM 1、LOW 7)からの現在の残りは、LOW 4 件になる。

- Merge: yes(S6 の再レビュー。CRITICAL・HIGH・MEDIUM なし。LOW 4 件はコメントと doc の直しで、動きを変えない。移動は純粋で、`WakeDirector` の分割は動きを変えない)
