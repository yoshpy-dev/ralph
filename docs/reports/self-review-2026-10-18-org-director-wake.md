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
