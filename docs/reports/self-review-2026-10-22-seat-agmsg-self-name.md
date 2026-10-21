# Self-review report: seat-agmsg-self-name

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-agmsg-self-name.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 1e47c309...HEAD`(HEAD dd0b77d0、19 ファイル、+1218/-77)。S1 782ceebf(evidence)、S2 b0a607d9(`driver.EnvRunner`、`Agmsg.run`)、S3 f16423da(`driver.NewHerdr`、`--env` の確かめ、本番の組み立て、CLI テストの stub)、S4 a34612fd(起動スクリプトの export、`spawn.go` のコメント)、S5 08d091f5(記録)、計画の 3 コミット。diff の質だけを見た

## Evidence reviewed

- コードの diff を全部読んだ。`internal/org/driver/{driver,agmsg,herdr}.go`、`internal/org/director_cmd.go`、`internal/org/spawn.go` のコメント、`internal/cli/org.go`、`internal/cli/org_director_headless.go`。テストは `driver_test.go`、`agmsg_test.go`、`herdr_test.go`、`director_cmd_test.go`、`internal/cli/org_runtime_env_test.go`、`org_test.go`、`org_director_headless_test.go` の diff を読んだ。テスト、gofmt・vet、doc drift の掃引は範囲外で、実行していない
- agmsg の呼び出しの抜け。`internal/` の非テスト Go を `"send.sh"` などの script 名、`exec.Command`、`AGMSG_SELF_NAME` で grep した。agmsg の script を打つのは `Agmsg.run` だけで、見張りの ALERT(`SendWatchdogAlert`)も `Agmsg.Send` を通る。`AgmsgAvailable` は stat だけ。`driver.Agmsg{` の組み立ては `internal/cli/org.go:357` の 1 か所で、Runner は `ExecRunner`
- herdr の呼び出しの抜け。`TabCreate` の呼び出しは `spawn.go:904`(座席)と `director_start.go:1123`(headless の director)の 2 か所で、組み立ては `org.go:356` と `org_director_headless.go:788` の 2 か所。どちらも `NewHerdr(…, AgmsgSelfNameOff)`。`WorkspaceCreate` が作る workspace の最初の pane は ralph が使わない(座席も director も `TabCreate` の pane に入る)
- 見張りの child は `orgWatchChildDroppedEnv`(`org_watch.go:368`)で `HERDR_PANE_ID` などを落とす。evidence の (c) の「見張りと同じ環境(`HERDR_PANE_ID` なし)」は、この実装と合う
- `--env` の確かめを、本物の herdr 0.7.5 に 3 回打って見た(`herdr tab create --help`、`env -i` に偽の HOME と存在しない `HERDR_SOCKET_PATH` だけを渡し、server には接続していない。「diff と読むだけ」の範囲からは外れる確認で、help の出力先を確かめるために打った)。rc 0、stdout 299 byte、stderr 0 byte、`--env <KEY=VALUE>` の行あり、所要 0.01 s。`ExecRunner.Run` は stdout だけを返すので、help が stderr に出ていれば確かめは常に「ない」になるところだったが、そうではない。`herdrTabCreateHelp` は実際の出力から空の説明行を除いたものと一致し、`herdrHelpEnvFlag` はその `      --env <KEY=VALUE>` の行に当たる
- 記録と evidence の突き合わせ。tech-debt の ALERT の行の数字(346/281 ms、657/533 ms、最大 1.2 s、差 65 ms、失敗 0、load average 8 と 20〜75)は evidence の (c)・(f) の表と合う。印の数は `grep` で数え直した。仕様は `[x]` 22・`[ ]` 10(base は 21・11)、根拠の表の文は 21 と 10 で、`[ ]` に挙げた 10 項目の名前も数と合う。PR の数は 30(9 段の 24 本 + #205、#241・#242、#244、#246、#247)で、NFR の行の 27 本(24 + 3)と合う。F3 の RESOLVED コメントが挙げるテスト名 8 つと commit の SHA 3 つは、tree と `git log` にある。tech-debt の新しい行は区切り `|` が 6 個(5 欄)で、base の行と同じ。#247 の計画と 4 つの report のファイルがある
- 追加行を `/Users/`、`hiroki`、token 形の文字列、16 進の長い列で grep した。evidence の文書を含め、漏れはない。新しい export の値は定数の `off` だけ
- 確かめていない: Go のテストの実行結果(`/test` の範囲)、`gh pr view` で 30 本が `MERGED` であること(#247 は `main` の merge commit で見える)、herdr 0.7.5 より古い版の `tab create --help` の挙動

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | maintainability | `Herdr.PaneEnv` は任意の `KEY=VALUE` を受ける公開の欄だが、`warnNoPaneEnv` の文面は `AgmsgSelfNameOff` 専用(「agmsg が座席の herdr agent を改名しうる」「`agent_not_found`」)で、doc コメントもそれを認めている。警告は `PaneEnv` の値をそのまま stderr に書き、同じ値は `herdr tab create --env K=V` の argv にも載る(`ps` で見える)。今の入口は定数 `AGMSG_SELF_NAME=off` だけなので漏れはないが、あとで token のような値を足すと、警告と argv の両方に出る。`warnNoPaneEnv` の doc の 1 文目(「writes the one warning line of a herdr TabCreate cannot add PaneEnv to, to Warn」)は、主語と述語がつながらず読みにくい | `internal/org/driver/herdr.go:38-46`(`PaneEnv` の doc)、`305-323`(`warnNoPaneEnv`) | `PaneEnv` の doc に「値は argv と警告に出る。秘密は入れない」の 1 文を足す。警告は値でなく key だけにする案もある。doc の 1 文目は書き直す |
| LOW (L2) | naming | 起動スクリプトは `strings.Cut(driver.AgmsgSelfNameOff, "=")` で定数を key と value に割り、`ok` を捨てている。同じ script の `RALPH_ORG_AGMSG_HOME` は名前の定数 `driver.AgmsgHomeEnv` をそのまま使う。定数から `=` が消えると key だけが `AGMSG_SELF_NAME=''` として書かれるが、テストが `AGMSG_SELF_NAME='off'` の行を固定しているので赤になる。実害は出ないが、読む側は割る理由を考えることになる | `internal/org/director_cmd.go:323`、`internal/org/driver/agmsg.go:35`(`AgmsgSelfNameOff`)と `138`(`AgmsgHomeEnv`) | `AgmsgSelfNameEnv = "AGMSG_SELF_NAME"` を足し、`AgmsgSelfNameOff = AgmsgSelfNameEnv + "=off"` にする。起動スクリプトは `AgmsgSelfNameEnv` と value `off` を直に書ける |
| LOW (L3) | readability | `launchScriptFakes` の戻り値が 5 つになり、呼び出し 4 か所が `_, _, _` と位置で受ける。agent の出力は `readLines(t, agentOut)[3]`、`[4]` と添字で読む(`[3]` は前からの書き方で、`[4]` が増えた)。行を足すたびに添字がずれる | `internal/org/director_cmd_test.go:496`、`598`、`604` | 任意。戻り値を構造体にまとめる。添字は `agmsg=`・`selfname=` の prefix で引く。前からの形に合わせたなら、このままでもよい |
| LOW (L4) | readability | evidence (c) の「1 本ずつ打つ条件は 20 回、6 本を同時に打つ条件は 4 回で計 24 本」は、24 が何を数えるか(同時の条件の送信数 4×6 か、測った点の数か)を言わない。同時の列(657/533/698)の値が 1 本ごとの時間か 1 回分の全体の時間かも書かれていない。tech-debt の ALERT の行は「six sends at once took a median of 657 ms」と書き、どちらの読みでも文は成り立つが、n=4 回分の中央値である点は表から読み取れない | `docs/evidence/agmsg-self-name-2026-10-22.md:40`(表は 42-45)、`docs/tech-debt/README.md` の ALERT の行 | 1 文で足す: 同時の列は 4 回の中央値で、各回の値は(全体の時間 / 1 本ごと)、送信は計 N 本 |

重大度の数は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4。

## Positive notes

依頼で挙がった観点ごとの結果を書く。

- 確かめの mutex(`herdr.go:271-303`)。ロックの中にあるのは `R.Run` 1 回(`context.WithTimeout` の 3 s で切る)、regexp、警告の書き込みだけで、ロックの中から `TabCreate` に戻る道はなく、デッドロックは起きない。待つ側はロックを ctx で諦められないが、待ちは持ち主の 3 s までで、本番の `TabCreate` は spawn と director start の中で逐次に呼ばれるので、競合は実際には起きない。8 goroutine の試験(`TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce`)が help 1 回を固めている。呼び出し側の ctx が終わっての失敗は `done` を立てず、警告も出さない(試験あり)ので、次の呼び出しがやり直す
- 警告が 2 回以上出るか。`NewHerdr` の値とそのコピーは `*paneEnvProbe` を共有するので、1 つの値で 1 回。リテラルの `Herdr{PaneEnv: …}` は呼び出しごとに確かめて警告するが、doc とテスト(`PaneEnvLiteralChecksEveryCall`)が決めており、本番は使わない。本番は 1 つの verb の process が `newOrgRuntimeAt` か `newOrgDirectorHerdrRuntime` の片方の値を使う。両方で `TabCreate` を打つ verb は見つからなかった。よって本番では 1 process あたり多くて 1 行
- CLI テストの stub の変更(`org_test.go`)が本物の失敗を隠すか。`tab create --help` の分岐は log の前にあり、help の呼び出しは log にも id の採番にも入らないので、既存テストの `tab create` の数と id は変わらない。stub の help は本物の 0.7.5 と同じ形(`--env <KEY=VALUE>` の行)。一方、help が失敗した場合と `--env` がない場合の枝は CLI の stub では通らないが、driver の試験が 4 通りで固めている。確かめが偽になる退行は、`TestOrgSpawn_HappyPath…` の「どの `tab create` も `--env AGMSG_SELF_NAME=off` で終わる」検査と、`TestOrgDirectorStart_LaunchesRegistersAndStops` の期待行、`TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff` の [help, tab create, tab create] の log で赤になる。隠さない
- env の値と argv。`PaneEnv` の値はコード内の定数で、`exec` の argv に渡るだけ(shell を通らない)ので、注入の道はない。残る点は L1 のみ
- 起動スクリプトの quoting。value は `shellQuote("off")` で、key は定数。`export` は check-in と exec の前、agmsg home の有無によらず出る。テストが全文(`AGMSG_SELF_NAME='off'\nexport AGMSG_SELF_NAME\n`)と順序、端末の `AGMSG_SELF_NAME=on` に勝つこと、check-in と agent の両方で `off` が見えることを固めている。`launch_cmd` の実行は `cmd.Env` を設定しないので、`t.Setenv("AGMSG_SELF_NAME", "on")` は launch_cmd と script まで届く
- `ExecRunner.RunEnv` は `append(os.Environ(), env...)`(`os.Environ()` は複製を返すので、親の slice を壊さない)で、os/exec が同じ key の最後の値を採る点を doc に書いてある。`Run` は `env=nil` で今までどおり。`Agmsg.run` の `EnvRunner` 判定は、本番の Runner が実装することを `TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff` が型と実際の値(親が `on` でも `off`)の両方で固めており、将来 Runner を包む型を挟んで黙って外れる退行を拾う
- テストの設計。fake の `envFakeRunner` は `Run` と `RunEnv` を別に数え、「env の Runner では `Run` を呼ばない」「Run だけの Runner は従来の argv」を両方固めている。`--envelope`・`--environment`・`--env-file`・`--no-env` が `--env` に当たらないことも試験(`TestHerdrHelpEnvFlag`)がある。実 herdr・実 agmsg に触れる試験はない(stub は `PATH` の先頭で、bash と sh の script だけ)
- 記録。tech-debt の F3 の行は、先例のとおり `<!-- RESOLVED … -->` のコメントと、取り消し線の行、Trigger の取り消しの 3 点がそろい、RESOLVED のコメントは「claude の座席、`--env` を付けた隔離 herdr の pane」と範囲を書いている。ALERT の行は、原因が書き換えだと示せなかったので開いたままで、Why deferred にもその理由がある。FR-9 の `[x]` は計画の consult の判定どおり、印の文に根拠のテスト名と evidence の (e)、「ほかに食い違う所見はない」が書かれている
- 不要な変更はない。`spawn.go` のコメントは配線が変わったことに合わせた直し。デバッグ出力、TODO、コメントアウトしたコードはない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

追加なし。L1〜L4 は任意の整理で、先送りの作業ではない。

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: マージ可(CRITICAL・HIGH・MEDIUM なし、LOW 4 件はどれも任意)
- Follow-ups:
  - L1〜L4 は直すなら小さい。L1 の doc 1 文と L2 の定数の分割は、同じ commit で直せる
  - 範囲外として扱ったこと(新しい指摘ではない): 起動スクリプトの export を書く文書(`docs/recipes/director-launch-cmd.md` の 131 行と 248 行、templates/base の写し、`/director` skill とその写し)と `AgmsgSelfNameOff` の doc コメントに起動スクリプトがないこと(`/sync-docs`)、0.7.5 より古い herdr が `tab create --help` を無視して tab を作るかどうか(未確認)、4b の F4・F11
  - 計画の Risks が受け入れた振る舞い: 確かめが一時的に失敗する(3 s の上限、手元の実測は 0.01 s)と、その結果は process の間キャッシュされ、その spawn の pane には `--env` が付かない。警告が唯一の知らせで、その座席には F3 が残る
  - 生の `ralph org spawn` を本物の herdr に打つ実機の確認は、この PR の evidence にない(S1 は手で作った pane と claude の座席)。codex の座席の環境も実機では見ていない。記録の文は claude の座席と書いているので食い違いはないが、`/test` と PR の本文で未確認として扱う
  - 「fix/seat-agmsg-self-name」の文字は、仕様・根拠の表・tech-debt に残る。PR の番号が決まったら置き換える(計画の「後の PR に残すこと」に載っている)
  - F4 の行の「the agmsg self-rename of the row above」は、4b で F3 の RESOLVED に合わせる(同上)
