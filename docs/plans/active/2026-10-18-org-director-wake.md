# org-director-wake

- Status: Approved
- Approved: 2026-10-18 sha256:e26fe4769759
- Owner: Claude Code
- Date: 2026-10-18
- Related request: 機能ごとの org と director の仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)の 8 段目の後半(8b)の 1 本目(8b-1)。FR-10 の起こし口のうち push と、その受け入れ条件 2 つ(仕様 120〜121 行)。8a は PR #229・#231・#232・#234・#235 で merge 済み(main は 9e92d928)。8b の分け方、push を打つ者、検証の置き場所は consult(consult-plan-wake、2026-10-18)の判定による。ユーザーは「すべてが完了するまで次のプランを自律的に進めて」と言った(2026-10-11)
- Related issue: N/A
- Type: feat
- Branch: feat/org-director-wake

## Objective

`[org.director].wake` に `push` を入れると、受信箱に新しい open の件が来たとき、横断の見張り(`ralph org watch --all`)が director を起こす。起こし方は `[org.director].wake_cmd`(利用者が書くコマンド)か、herdr の中の headless の director なら herdr の pane への入力にする。送り先は attach か start で登録した値を使う。起こしたことは受信箱に記録し、起こしても ack の期限は残るので、届かなかった件は今までどおり期限で人に届く。既定の `["pull"]` のままなら、動きは今と変わらない。

## Scope

- 設定 `[org.director].wake`(文字列の配列、既定 `["pull"]`)と `[org.director].wake_cmd`(argv、既定は空)、その形の検証、既定値の 3 面(Go、`scripts/ralph-config.sh` の 2 面、`templates/base/ralph.toml`)
- 登録: `ralph org director attach --wake` と `--wake-target`、start が書く `wake`、送れるかの検証(attach と start)、`director status`・`ralph status`・`--json` の表示
- 見張りのサイクルに「起こす」手順を足す。wake_cmd の実行、herdr の既定の送り方、受信箱の新しい event `woken`、失敗のときの間隔
- 設定の読み方: wake と wake_cmd は launch_cmd と同じく main worktree の `ralph.toml` から読む(attach、start、`watch --all`)
- 文書: `/director` skill の 4 面、雛形 `internal/org/prompts/director.md`、README、仕様 FR-10 の注、`templates/base/ralph.toml` のコメント、新しい recipe `docs/recipes/director-wake.md`(2 面。Orca の `orca terminal send`、tmux、herdr の中の promoted session)、tech-debt、AGENTS.md
- 実機の確認: 分離した herdr server で codex の headless director を push で起こす

## Non-goals

- director の生存の印、落ちた start が残す `starting` の登録の期限、見張りが死んだことに気づく仕組み(8b-2。tech-debt の 247・255 行と 209 行の半分)
- escalate がその場で push を打つこと(consult の判定で見張りに寄せた。leader の座席の sandbox と環境からは打たない)
- 起こしたあとに director が件を処理したかの確かめ。それは ack と期限が受け持つ
- 受信箱の本文や TYPE を起こす文に載せること(セキュリティの節)
- `ralph org director start --wake` のフラグ(start は設定の `wake` を書く)
- 9 段目の試行と既定値の見直し
- Windows

## Assumptions

- 見張りは台帳ごとに 1 つ(`watch.lock`)なので、起こすのも台帳ごとに 1 つのプロセスだけになる
- Claude Code と codex は、作業中に pane へ入った文と Enter を、次の入力として待たせる。codex については未確認なので実機で確かめる(Risks)
- herdr の `pane send-text` のあと Enter が貼り付けの一部として読まれることがある(#155 の evidence)。そのため send-text と Enter のあいだに、`ralph org send` と同じ 750 ms(`DefaultSendEnterDelayMS`)を置く
- `orca terminal send --terminal <handle> --text <text> --enter`(Orca CLI 1.4.229 の `--help`)で、Orca の端末に文と Enter を送れる。Orca の中の session は env の `ORCA_TERMINAL_HANDLE` で自分の handle を知る(`orca agent-context` に名前がある)。recipe の例はどちらも実行していない

## Affected areas

- `internal/config/config.go`、`internal/config/config_test.go`、`internal/config/defaults_sync_test.go`(`OrgDirectorConfig.Wake` と `WakeCmd`、検証、置き換え)
- `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh`(`RALPH_ORG_DIRECTOR_WAKE`)、`templates/base/ralph.toml`
- `internal/org/director.go`(`DirectorWakePush`、`attached` の `wake` と `wake_target`、`DirectorAttach`、登録の送り先)
- `internal/org/director_start.go`(start が書く `wake`、launch_cmd と push の断り)
- `internal/org/director_wake.go`(新規。起こす件の選び方、起こす文、送り方の選び方、wake_cmd の実行、herdr の送り方、送り先の形の確かめ、失敗の間隔のメモリ)
- `internal/org/inbox.go`(`InboxEventWoken`、fold、`AppendWoken`、`InboxItem.Woken` と `WokenAt`)
- `internal/org/watch.go`(サイクルの順に「起こす」を足す)
- テスト: `internal/org/director_wake_test.go`(新規)、`director_test.go`、`director_start_test.go`、`inbox_test.go`、`watch_all_test.go` など見張りのテスト、`internal/cli/org_director_test.go`、`org_director_headless_test.go`、`org_inbox_test.go`、`org_watch_test.go`、`status_test.go`
- `internal/cli/org_director.go`(attach のフラグ、status の表示)、`internal/cli/org_director_headless.go`(start の断り)、`internal/cli/org_watch.go`(main worktree からの読み込み)、`internal/cli/org.go`(読み込みの関数)、`internal/cli/org_inbox.go`(show の `woken`)、`internal/cli/status.go`
- `internal/org/prompts/director.md`、`.claude/skills/director/SKILL.md` と写しの 3 面、`README.md`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/recipes/director-wake.md` と `templates/base/docs/recipes/director-wake.md`、`docs/tech-debt/README.md`、`AGENTS.md`

## Visual review

- ページ: `.harness/state/plan-visual/org-director-wake.html`(かんたん版 2 枚: overview、easy-where。折りたたみに detail-overview、detail-cycle、detail-check、detail-records)
- 自己点検: `plan-visual.sh lint` が exit 0。全体と 6 枚の図を撮って読んだ。easy-where の箱の間が 10px で狭く、見張りの説明の「1 回に 1 度」がわかりにくかったので、間を 30px に広げて言い換え、撮り直して確かめた。Codex の計画助言で D2・D9 を直したあと、detail-check の読み込みと見張りの箱に、最初の要素の placeholder の断りと打つ場所を足し、重なった行を直して lint と撮り直しで確かめた

## Design decisions

- D1 押す者: 横断の見張り(`RunWatchAll`)が、各サイクルで新しい open の件を見つけたら打つ(consult)。理由は 3 つ
  - 設定と台帳を読む場所が 1 つになる
  - 期限の送り(人への経路)と同じ場所で件を見る
  - 見張りは人か start が立てた ralph の外のプロセスで、leader の座席の sandbox に入っていない

  見張りが動いていなければ、escalate は件を記録と同時に人へ送る(8a-1 の `inbox_no_watch`)ので、push がないために件が止まることはない。遅れは最大で見張りの間隔(既定 30 秒)
- D2 設定の形: 読み込みでは値の形だけを見る(consult)
  - `wake` は文字列の配列で、既定は `["pull"]`。要素は `pull` と `push` だけで、重複と空の配列は断る。順は問わない
  - `wake_cmd` は文字列の配列(argv)で、既定は空。シェルを通さずに打つ。置き換えは `{target}`(送り先)と `{text}`(D7 の文)の 2 つだけにし、ほかの `{...}` と、空白だけの最初の要素は断る。最初の要素(打つ実行ファイル)は placeholder を含めない(送り先は leader も書ける `director.jsonl` から来るので、打つものを選ばせない。Codex の計画助言)。placeholder の見つけ方は launch_cmd と同じ規則(`{` と次の `}`)にする
  - 断るエラーはキーの名前と理由を言う
- D3 登録の起こし方と送り先
  - attach は `--wake <pull|push|pull,push>`(省略すれば設定の `wake`)と `--wake-target <値>` を受け、`attached` に `wake` と `wake_target` を書く
  - start は設定の `wake` を書く。送り先は登録から導き、`wake_target` は書かない。herdr の director は `pane_created` の pane id、launch_cmd の director は端末の名前 `director_<token>`(tmux の例では session の名前)
  - 見張りは登録の `wake` を読む。設定の `wake` を変えても今の登録には効かず、attach か start をし直す
- D4 送れるかの検証は attach と start で行う(consult)。断るときは何も書かない
  - attach で wake に push を含むとき、wake_cmd が空なら断る(promoted session には既定の送り方がない)。wake_cmd が `{target}` を含むのに `--wake-target` がなければ断る
  - attach で wake に push を含まないのに `--wake-target` があれば断る
  - `--wake-target` の形: 空でない、制御文字を含まない、256 文字以下、`-` で始まらない(wake_cmd の先のコマンドがオプションと読むため)
  - start で wake に push を含むとき、herdr の起動は通す(pane が送り先で、wake_cmd が空なら herdr の pane への入力)。launch_cmd の起動は、wake_cmd が空なら断る(launch_cmd も herdr も打たない)
  - 仕様の受け入れ条件 121 行の「when 設定を読む、then 検証エラー」は、「送り先は登録で決まるので、読み込みでは形を、送れるかは attach と start で確かめる」と読み替え、FR-10 の注に書く
- D5 起こす件: 状態が open で、`woken` の event がない件。director が登録される前からある open の件も、1 件につき 1 回だけ起こす。ack、resolve、notified の追記では起こさない。delivery が now の件(人にすぐ送った件)も、`woken` がなければ起こす(consult)
- D6 1 サイクルに 1 回: サイクルの順を「期限の送り → 起こす → 各 org の pulse → 生存の印」にする。期限の送りを先に置くので、起こす手順が詰まっても、そのサイクルの人への送りは遅れない
  - 起こすのは、今の登録があり、その wake に push を含み、headless なら `running` のときだけ
  - 起こす件があれば、その ID の列を載せた文を 1 回だけ送る。列は 20 件までで、残りは数だけを書く
  - 送れたら(wake_cmd の終了コード 0、herdr の send-text と Enter の成功)、各件に `woken`(`director_id` と `via`)を 1 回のロックの下で書く
  - 送れなければ `woken` を書かず、stderr に 1 行を出す。同じ登録への次の試しは 5 分あける(見張りのメモリだけ。起動し直した見張りはすぐ試す)
  - 送ったあと `woken` を書く前に見張りが落ちると、次の見張りが同じ件をもう 1 回起こす。重なるのは 1 回までとし、そのまま受け入れる
- D7 起こす文: ralph が決めた 1 行。件の ID(`e<N>` の形だけ)と、読み方の案内だけを載せる
  - 本文、TYPE、org_id、送り先は載せない
  - シェルの記号(`;`、`|`、`&`、`$`、`` ` ``、`<`、`>`、`(`、`)`、`*`、`?`、引用符)を含めない。端末のシェルに届いても、存在しないコマンドの名前として終わる
  - 先頭を agent の特別な入力(`/`、`!`、`#`)にしない
  - 文はテストで固定する
- D8 送り方の選び方
  - wake_cmd が空でなければ、どの登録にも wake_cmd を使う(仕様の「既定の送り方」は wake_cmd が空のときの話と読む)。`{target}` は D3 の送り先、`{text}` は D7 の文。`DriverCallTimeout`(10 秒)で区切り、stdin は空、出力は捨てて、失敗の 1 行に短く載せる。打つ場所は D9 の信頼できる設定のディレクトリに固定する
  - wake_cmd が空で登録が herdr の headless なら、pane がこの登録のものであることを stop と同じ label の照合で確かめる。確かめたら `herdr pane send-text <pane> <text>` を打ち、750 ms 待って `herdr pane send-keys <pane> Enter` を打つ。照合が合わなければ送らない
  - どちらでもないとき(設定をあとで変えて wake_cmd が空になった promoted session や launch_cmd の director)は送らず、stderr に登録ごとに 1 回だけ書く
  - 見張りは、使う前にも送り先を D4 の形で確かめる(手で書き換えた `director.jsonl` のため)。形に合わなければ送らない
- D9 設定の読み方と打つ場所: wake と wake_cmd は launch_cmd と同じ規則で読む(`--config` がなく台帳が main worktree のものなら、main worktree の `ralph.toml`)。対象は attach、start、`watch --all`。leader は feature worktree の `ralph.toml` を書き換えられるが、見張りが打つコマンドは変わらない。見張りは起動したときに読むので、wake_cmd を変えたら見張りを起動し直す(help と recipe に書く)
  - wake_cmd は、それを読んだ `ralph.toml` のあるディレクトリで打つ。main の規則で読んだなら main worktree のルート、`--config` ならそのファイルのディレクトリ(絶対パス)、どちらでもなければ見張りの起動時の作業ディレクトリ(そこの `ralph.toml` を読んだため)。見張りの作業ディレクトリでは打たない(Codex の計画助言 HIGH: feature worktree から前景で立てた見張りが、leader の書き換えた同じ名前のスクリプトを sandbox の外で打たないため)
  - 最初の要素が区切り(`/`)を含む相対パスなら、そのディレクトリから絶対パスに解いてから打つ。区切りを含まない名前は PATH で探す(Go の `exec` は PATH の相対の項目を使わない)
- D10 受信箱の `woken`: inbox.jsonl の新しい event で、状態を変えない
  - fold は item に `Woken` と `WokenAt` を載せる
  - `ralph org inbox show` の events に出し、`--json` の item に `woken` と `woken_at` が増える
  - 期限の計算(`InboxDeadlineDue`)は `woken` を読まない。起こしても ack の期限は残るので、FR-10 の「届かなかった件は FR-6 の期限で人に届く」はこれで守る
  - この段より前の ralph はこの行を、知らない event として数えて無視する
- D11 実機: 分離した herdr server(XDG の 3 変数)と隔離した台帳で、codex の headless director を `wake = ["push"]` で立てる。escalate の件で見張りが pane に文を送り、director が動き出すことを確かめる。`["pull", "push"]` で、codex が foreground の `wait --inbox` を走らせているあいだに入れた文がどうなるかも見る(consult の未確認の点)

Critical forks: 8b の分け方(8b-1 = FR-10、8b-2 = 生存の印と `starting` の期限)、押す者(見張り)、検証の置き場所(読み込みは形、送れるかは attach と start)、起こす件と記録(open で `woken` のない件、受信箱の event)を consult(consult-plan-wake)に聞いた。判定は「直してから進める」で、上の D1・D2・D4・D5・D6 に入れた。細部(attach の `--wake` による上書き、wake_cmd を herdr の既定より先に使うこと、launch_cmd の送り先を端末の名前にすること、失敗のあとの 5 分の間隔)は、ユーザーの指示「計画の分かれ目も自分で選ぶ」に従って自分で選んだ。そのあと Codex の計画助言が「wake_cmd を打つ場所と相対パスの解き方が決まっておらず、feature worktree から前景で立てた見張りが leader の書き換えたスクリプトを sandbox の外で打ちうる」と指摘した(HIGH)。ユーザーの指示どおり直し、打つ場所を読んだ設定のディレクトリに固定し、相対の実行ファイルもそこから解き、最初の要素に placeholder を許さない形にした(D2、D8、D9、AC1、AC4)。

## Acceptance criteria

- [ ] AC1 設定: `[org.director].wake` は既定で `["pull"]` で、`["push"]`、`["pull", "push"]`、`["push", "pull"]` を読める。空の配列、`pull` と `push` 以外の値、重複は読み込みで断る。`wake_cmd` は既定で空で、最初の要素が空白だけのもの、最初の要素が placeholder を含むもの、`{target}` と `{text}` 以外の `{...}` を含むものは読み込みで断る。断るエラーはキーの名前と理由を言う。`wake = ["push"]` で `wake_cmd` が空でも、読み込みは通る。既定値は Go、`scripts/ralph-config.sh` の 2 面、`templates/base/ralph.toml` で同じ(`defaults_sync_test.go`)
- [ ] AC2 attach: `--wake` と `--wake-target` の値を `attached` の `wake` と `wake_target` に書き、`--wake` を省くと設定の `wake` を書く。次のときは断り(終了コード 1)、何も書かない
  - wake に push を含み、wake_cmd が空
  - wake に push を含み、wake_cmd が `{target}` を含むのに `--wake-target` がない
  - wake に push を含まないのに `--wake-target` がある
  - `--wake-target` が空、制御文字を含む、257 文字以上、`-` で始まる
  - `--wake` に `pull` と `push` 以外の値や重複がある

  `ralph org director status`、`ralph status` の `director:` の行、`--json` は、登録の wake と送り先を出す(制御文字は今の表示と同じく escape する)
- [ ] AC3 start: 設定の `wake` を `attached` に書く。wake に push を含み、launch_cmd の起動で wake_cmd が空なら、断って何も書かず、launch_cmd も herdr も打たない。herdr の起動は wake_cmd が空でも通す
- [ ] AC4 設定の読み方と打つ場所: `--config` がなく台帳が main worktree のとき、feature org の worktree の `ralph.toml` の `wake` と `wake_cmd` を変えても、attach、start、`watch --all` が使う値は main worktree の `ralph.toml` のものになる。main と feature worktree に同じ相対パスの別のスクリプトを置き、main の wake_cmd がそれを `./` で指すとき、feature worktree から前景で立てた見張りは main 側のスクリプトだけを、main worktree のルートを作業ディレクトリにして打つ。`--config` なら、そのファイルのディレクトリで打つ
- [ ] AC5 起こす手順(wake_cmd): 偽の wake_cmd を書き、`wake = ["pull", "push"]` で attach した台帳で escalate すると、`ralph org wait --inbox` がその件で返り、見張りの次のサイクルで wake_cmd が 1 回打たれる(仕様 120 行の受け入れ条件。「escalate が記録されると」を「見張りの次のサイクルで」と読む)
  - 打たれる argv は置き換えたもので、シェルを通さない。`{text}` は件の ID を含み、本文を含まない
  - 件には `director_id` と `via` 付きの `woken` が書かれ、次のサイクルでは打たれない
  - 同じサイクルに 2 件あれば、1 回の打ちに 2 つの ID が載る。21 件以上なら 20 件の ID と残りの数が載る
  - acked、resolved、`woken` 済みの件では打たない。登録がない、wake に push がない、headless で `starting` か `stopped` のときも打たない
- [ ] AC6 起こす手順(herdr の既定): wake_cmd が空で、herdr の headless の director が `running` のとき、見張りは label の照合、`pane send-text`、Enter の順に偽の herdr を打つ。照合が合わなければ send-text を打たない。wake_cmd があれば、`{target}` を pane id にして wake_cmd を打ち、herdr の send-text は打たない
- [ ] AC7 失敗と期限: wake_cmd の終了コードが 0 でない、時間切れ、herdr の失敗のどれでも、`woken` を書かずに stderr に 1 行を出し、5 分のうちは同じ登録に打たない。送り先が D4 の形に合わない登録(手で書き換えた行)と、送る手段のない登録(promoted session か launch_cmd の director で wake_cmd が空)には打たず、stderr に登録ごとに 1 回だけ書く。送れなかった件も、送れた件も、ack しなければ ack の期限で人に届く(期限の送りは起こす手順より先に走り、`woken` を読まない)
- [ ] AC8 受信箱: `woken` は状態を変えない。`ralph org inbox show` の events に出て、`--json` の item に `woken` と `woken_at` が載る。`woken` のない今までの行の読み方は変わらない。知らない件への `woken` は無視して数える
- [ ] AC9 文書: `/director` skill の 4 面が同じ中身で、push で起こされたときの扱いを書き、「まだないもの」から push を外す。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` が通る。雛形、help、README、仕様 FR-10 の注(D4 と AC5 の 2 つの読み替え)、`templates/base/ralph.toml` のコメント、recipe の 2 面、tech-debt、AGENTS.md が push の動きを説明し、「push はまだない」と書いた箇所が残っていない
- [ ] AC10 テストは本物の herdr、osascript、端末のアプリ、Orca、人の tmux を打たず、repo の台帳に書かない。実機の確認は、分離した herdr server と隔離した台帳で codex の director を起こし、evidence に残す(`$HOME` は `~` に、送り先と token の値は伏せる)

## Implementation outline

1. S1 設定: `OrgDirectorConfig.Wake` と `WakeCmd`、形の検証、置き換えの関数、3 面の既定値とコメント(AC1)
2. S2 登録: `attached` の `wake` と `wake_target`、attach のフラグと検証、start の断り、登録の送り先、status の表示、main worktree からの読み込み(attach と start)(AC2〜AC4 の attach と start の分)
3. S3 起こす手順: 受信箱の `woken`、`director_wake.go`(件の選び方、文、送り方、wake_cmd の実行、herdr の送り方、間隔のメモリ)、見張りのサイクル、`watch --all` の読み込み、`inbox show` の表示(AC4 の見張りの分、AC5〜AC8)
4. S4 文書: skill の 4 面、雛形、README、仕様の注、recipe の 2 面、tech-debt、AGENTS.md(AC9)
5. S5 実機の確認と evidence(AC10)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、golangci-lint、secret scan)、`go vet ./...`、`GOTOOLCHAIN=go1.25.8` の build と vet、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: FR-10 の 5 項目(wake の値、pull の待ち、push と送り先と herdr の既定と Orca の recipe、送れないときの検証、どちらでも期限で人に届く)を AC1〜AC9 で、受け入れ条件 120 行を AC5 で、121 行を AC1〜AC3 と FR-10 の注で確かめる。セキュリティの節の「受信箱の本文は雛形やコマンドに入らない」を AC5 と D7 で、原則の「記録は台帳に置き、push は通知だけに使う」を D10 で確かめる
- Documentation drift to check: `/director` skill の「まだないもの」から push を外したか。tech-debt の 8b を指す行のうち、この段で閉じるものと 8b-2 に残すものを書き分けたか。README と仕様の注が pull だけを書いていないか
- Evidence to capture: 実機の確認(分離した herdr server と隔離した台帳)で打ったコマンドと出力、`director.jsonl` と `inbox.jsonl` の行、pane の読み取り。`$HOME` は `~` に置き換え、送り先と token の値は伏せる

## Test plan

- Unit tests:
  - wake と wake_cmd の検証の表(AC1)
  - wake_cmd の置き換え(`{text}` と `{target}`、値の中の `{...}` を置き換え直さないこと)
  - 送り先の形の表(D4)
  - 起こす文(固定の文、ID の列の 20 件の上限、シェルの記号を含まないこと)
  - 起こす件の選び方の表(open、acked、resolved、woken 済み、delivery の now と deadline)
  - 受信箱の fold の `woken`(状態を変えない、知らない件、`WokenAt`)
  - 送り方の選び方(wake_cmd、herdr、どちらもない)
- Integration tests:
  - CLI の attach の断りの表と、登録に書かれる値(AC2)
  - start の launch_cmd と push の断り、herdr の通り(偽の herdr。AC3)
  - feature worktree の `ralph.toml` を書き換えても値が変わらないこと(AC4)
  - 偽の wake_cmd(引数を印のファイルに書くスクリプト)で escalate、`wait --inbox`、見張りの 1 サイクル(AC5)
  - 偽の herdr で label の照合、send-text、Enter の順と、照合の失敗(AC6)
  - 失敗の 1 行、5 分の間隔、ack の期限で人に届くこと(AC7)
  - `inbox show` と `--json`(AC8)
- Regression tests: `wake = ["pull"]` のままなら、見張りは wake_cmd も herdr の send-text も打たない。8a のテスト(attach、start、stop、escalate、期限)がそのまま通る
- Edge cases:
  - 起こしているあいだに件が ack される(`woken` は事実として書く)
  - 同じサイクルで期限の送りと起こすことが重なる
  - 送り先に制御文字を書いた手の行
  - wake_cmd が戻らない(時間切れ)
  - 登録が入れ替わった直後(新しい登録には新しい件から起こし、`woken` 済みの件は起こさない)
  - headless の `starting` のあいだの件(`running` になったサイクルで起こす)
- Evidence to capture: `go test` の出力(go1.26 と go1.25.8)、`-race`、mutation の結果

## Risks and mitigations

- 人が打ちかけている対話の session(Orca の promoted session など)に文と Enter が届くと、打ちかけの文が一緒に送られる: recipe に書き、人が打つ session は pull を勧める。headless の director は人が打たないので影響しない
- leader は `ralph org director attach` を打てるので、自分の決めた送り先を登録できる(今もある形で、雛形が禁じているだけ): 起こす文は固定で本文を含まず、シェルの記号もないので、どこに届いても命令にならない(D7)。wake_cmd は main worktree の `ralph.toml` からだけ読み、最初の要素に placeholder を許さないので、送り先が打つものを選ぶことはない(D2、D9)。送り先は形を確かめる(D4、D8)
- 見張りを feature worktree から前景で立てると、作業ディレクトリが leader の書ける場所になる: wake_cmd は読んだ設定のディレクトリで打ち、相対の実行ファイルもそこから解く(D9、Codex の計画助言 HIGH)。PATH で探す名前は見張りの env の PATH に従うので、PATH を leader が変えられる環境では見張りを立てない(recipe に書く)
- 送り先の herdr の pane id が別の pane に使い回される: 送る前に stop と同じ label の照合をする(D8)
- herdr が固まると見張りのサイクルが遅れる(tech-debt 203 行): 期限の送りを先に置き(D6)、各呼び出しを 10 秒で区切る。遅れは次のサイクルの期限の送りに及ぶので、tech-debt の行に起こす手順を書き足す
- 送ったあと `woken` を書く前に見張りが落ちると、もう 1 回起こす: 重なりは 1 回までで、件の処理は ack で 1 回になる(D6)
- codex が作業中に入った文をどう扱うかが未確認: 実機で確かめ、結果を skill と recipe に書く(D11)
- 起こしても director が件を処理しない: 起こすことは ack の期限を止めないので、件は期限で人に届く(D10)
- この段より前の ralph に戻すと、`woken` の行を知らない event として数える: rollback note に書く

## Rollout or rollback notes

- 既定は `["pull"]` なので、設定を変えない人の動きは 8a と同じ。`ralph upgrade` は seed の `ralph.toml` に `wake` を足さないが、ない値は既定で読む
- 戻すときは、push の登録を detach か `director stop` で外してから revert する。前の ralph は `woken` の行を無視して数え(`ralph org inbox` が ignored の数を出しうる)、登録の `wake` の `push` は表示するだけで起こさない

## Open questions

- なし(分かれ目は consult の判定と、Design decisions の自分の選択で決めた)

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved(ユーザーの指示「作成したプランは自動承認します」による自動承認。Codex の計画助言 1 件(HIGH)は D2・D8・D9・AC1・AC4 に反映済み)
- [x] Branch created
- [x] Implementation started
  - S1 設定(AC1): 2f08cfe7。launch_cmd と wake_cmd が placeholder の検査と置き換えのヘルパーを共有し、launch_cmd のエラー文は変わらない
  - S2 登録(AC2・AC3、AC4 の attach と start): 0d49b2a0。逸脱 1 件: store の `Attach` は空の `Wake` を断らずに `["pull"]` として記録する(範囲外のテストが `Wake` なしで `DirectorAttach` を組むため。CLI はいつも解決済みの値を渡すので動作は同じ。空でない不正な wake と不正な形の送り先は store でも断る)。`--wake` のエラーは `config.ValidateDirectorWakeNamed` でフラグ名を出す。main worktree の wake と wake_cmd は attach だけが読み、status は登録の値を出す
  - S3 は 2 つのコミットに分けた。S3a は受信箱の `woken`(AC8)、S3b は見張りが起こす手順(AC4 の見張りの分、AC5〜AC7)
  - S3a 受信箱(AC8): 0f9b457f。`appendLocked` を複数の event を 1 回で書く形にした。`inbox show` の help の event の一覧に woken を足した
  - 順の入れ替え: S5(実機)を S4(文書)の前に回す。codex が作業中に届いた入力をどう扱うか(D11、Risks)の結果を skill と recipe に書くため
  - S3b 見張りが起こす手順(AC4 の見張りの分、AC5〜AC7): b716d041。launch_cmd の runner を `runDirectorCmd` に一般化し、wake_cmd は出力の末尾 512 バイトだけを残す(launch_cmd は 4096 バイトのまま、エラー文は変わらない)。`--detach` でも親が main worktree の ralph.toml を読み、読めなければ親のエラーにする。director.jsonl が読めないサイクルは起こす手順だけを飛ばす。S4 に回す残り: `--config` の help に `watch --all` が wake と wake_cmd を読むことを足す、`internal/cli/org_director_wake_test.go:26` の「S3 で入る」のコメントを直す
  - S5 実機(AC10): `docs/evidence/director-wake-livefire-2026-10-18.md`。分離した herdr server で codex 0.160.1 の director を 2 回起こした。待っていない codex は 1 行を新しい手番として受け、受信箱を読んで ack し、QUESTION を人に聞いた。手番の途中に届いた 1 行と Enter は、次の手番まで待たされずに今の手番に差し込まれ(同じ turn_id)、失われなかった。pull と push の併用では、wait が拾った件を同じ秒に 1 行が名指しし、woken が acked のあとに記録されることがある。codex はどちらの回も最初の手番で `wait --inbox` を始めなかった。S4 で skill と recipe にこの結果を書く
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
