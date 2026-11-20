# Self-review report: org-reseat-restart-herdr

- Date: 2026-11-20(cycle 1)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`
- Branch: `feat/org-reseat-restart-herdr`、HEAD `5e99a0fe`、base は `origin/main` の `a672b72a`(issue #354)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD` の 28 ファイル(+4020 −120)のうち、コードと文書の diff の質だけ。コードは S1 `558227cf`(driver の `PaneListAll`・`ServerStop`・`HerdrPane.Cwd`、`org_herdr_server_env.go`、`org_herdr_server.go` の env 上書き)と S2 `e00adaf5`(`internal/org/reseat.go`、`internal/cli/org_reseat.go`)。文書は `3a83dbf0`(director・org の skill、README、AGENTS.md)と spec・tech-debt の追記。テストファイルは、本番コードの seam と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)。

## Evidence reviewed

- `internal/org/reseat.go`: diff 全体と周辺。`PlanReseat` の 9・10 番(`reseatRestartHerdrErr` と gate の順)、`Reseat` の dry run と real の 2 つの経路、`reseatFirstPanesErr`、`reseatOtherPanes`、`reseatOwnPanes`、`restartHerdr`(2 回目の pane の確かめ、`herdr server stop`、`waitHerdrStopped`、gate)、`finish`、`reseatRetryHerdrFlags`、`reseatRetryCommandWithHerdr`。`checkHerdrServerGate`(`herdrgate.go`)と `callWithTimeout`(`verbs.go:1091`)も読んだ
- 打ち直しの command の確認: 失敗の 6 点で `HerdrStopped` と `--restart-herdr` の有無を追った(下の表)
- `internal/cli/org_herdr_server_env.go` 全 358 行、`internal/cli/org_herdr_server.go` の diff、`internal/cli/org_reseat.go` の diff(help、flag、`runOrgReseat`、`printOrgReseatHerdrServerNotes`、probe、dry run と result の出力)
- `internal/org/driver/herdr.go` の diff(`PaneListAll`、`herdrPaneListResult`、`ServerStop`、`Cwd`)
- 文書: `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` の diff(`.agents/` と `templates/base/` の写しは見ていない。同期は `/verify`)、README と AGENTS.md の語単位の diff、tech-debt の新しい行、spec の 2 つの追記、`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`
- 周辺の確認: `reseatRetryCommand` の呼び出し元(`grep`)、`confirmSeatPane`(`verbs.go:1195`)が Stop の中で pane の label を確かめること、`HERDR_PANE_ID` の他の読み手
- 自分で走らせた probe(scratch は `$TMPDIR` 配下、repo の外): `orgPSEnvValue` と `orgPSEnvKey` の写しに、識別子でない名前の env を混ぜた `ps eww` 風の行を 4 つ。読み取りだけの `lsof -t` と `ps` をこのマシンの本物の herdr の socket に 1 回(下の「よい点」)

## Findings

この run の指摘は M1 と L1〜L4。CRITICAL と HIGH はない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM (M1) | security | `orgPSEnvValue` は、`CLAUDE_CONFIG_DIR` の次の entry の名前が `[A-Za-z_][A-Za-z0-9_]*` に合わないと、その entry を丸ごと `CLAUDE_CONFIG_DIR` の値に含める。値は `herdr server: CLAUDE_CONFIG_DIR=<値>` の行と mismatch の note にそのまま出る(stderr)。ハイフンやドットや `%` を含む名前の env(`npm_config_foo-bar`、`foo.bar`、bash の関数の `BASH_FUNC_x%%`)や、数字で始まる名前が後ろにあると、その値(secret のこともある)が出る。`reason` の「読んだ environment を持たない」という保証(`org_herdr_server_env.go:108-111`)が、value の側では成り立たない。この read は mac の経路(`ps eww`)だけで、Linux の `/proc` は NUL 区切りなので影響しない。director は stderr を LLM の文脈に取り込むので、出た値は端末の外にも出る | `org_herdr_server_env.go:281` の `regexp ` [A-Za-z_][A-Za-z0-9_]*=``、`:290-304` の `orgPSEnvValue`(終わりを「次の `KEY=`」で決める)。表示は `org_reseat.go:574` と `:601`。scratch で `orgPSEnvValue` の写しに `…CLAUDE_CONFIG_DIR=/Users/x/.claude-b x-y=SECRET1 PATH=/bin` を渡すと `value="/Users/x/.claude-b x-y=SECRET1"`、`…CLAUDE_CONFIG_DIR=/Users/x/.claude-b BASH_FUNC_ml%%=() { secret } PATH=/bin` で `value="/Users/x/.claude-b BASH_FUNC_ml%%=() { secret }"`、`…/.claude-b 9LIVES=SECRET2` で `"/Users/x/.claude-b 9LIVES=SECRET2"` が返った。`--claude-config-dir` つきで ralph が起こしたサーバーは `CLAUDE_CONFIG_DIR` が env の最後(`apply` が末尾に足す)なので当たらないが、呼んだ側の env をそのまま継いだサーバーは並びが継いだ順になる | entry の始まりを「空白の次の、`=` と空白を含まない 1 語と `=`」(` [^ =]+=`)に広げ、`foo-bar=` や `BASH_FUNC_x%%=` でも値が切れるようにする。値に空白と `KEY=` が入るときの切れ(spec FR-7 が許す)は残る。出す前に、値が絶対 path で `os.Stat` できる、などの形の確かめを足すのも手。名前が識別子でない entry を混ぜたテストを `orgPSEnvValue` に足す(今は `orgPSEnvValue` を呼ぶテストがない: `grep -n orgPSEnvValue internal/cli/*_test.go` が空) |
| LOW (L1) | maintainability | `reseatRetryCommand` は本番の呼び出し元が 0 で、`reseat_test.go:1999` のテストだけが呼ぶ。`reseatRetryCommandWithHerdr` を呼ぶ 1 行の wrapper が、前のテストのためだけに残っている | `internal/org/reseat.go:1879`(定義)、`grep 'reseatRetryCommand(' internal --include='*.go'` は `reseat.go:1879` と `reseat_test.go:1999` の 2 件。`finish`(`reseat.go:1678`)は `reseatRetryCommandWithHerdr` を直接呼ぶ | wrapper を消し、`reseat_test.go` の呼び出しを `reseatRetryCommandWithHerdr(…, reseatRetryHerdr{}, …)` に直す。または `reseatRetryCommandWithHerdr` を `reseatRetryCommand` に戻し、herdr の flag の引数を足す形にする |
| LOW (L2) | readability(コメントが事実と合わない) | `PlanReseat` の 9 番の説明の「(確かめは)gate の前に来る。gate は watch と止まったサーバーを起こすので、断られた restart はどちらも起こさない」は、`reseatRestartHerdrErr` が断る場合にしか当たらない。いちばん起きやすい断り(ほかの pane がある、`reseatFirstPanesErr`)は `Reseat` の中の、10 番の gate が通ったあとなので、real の run では `wireOrgStartWatchGate` が watch を起こしたあとに断られうる(サーバーは 9 番で動いていると確かめ済みなので起こさない) | `internal/org/reseat.go:425-429`(コメント)、`:1362-1366`(1 回目の pane の確かめは `PlanReseat` の後)、`:582`(real の `checkWatchGate`)。README と help の「何も止めずに断る」は正確で、この指摘は `reseat.go` のコメントだけ | 「断られた restart はどちらも起こさない」を「9 番で断られた restart は、watch もサーバーも起こさない」に直す。pane の確かめで断られると watch は起きていることがある、と 1 文足す。順を変える(pane の確かめを gate の前に出す)ほどの害ではない |
| LOW (L3) | readability | `restartHerdr` の `herdr server stop` が失敗したときの枝で、エラーの文面を作る `switch` が `r.res.HerdrStopped = true` も立てる。状態を変える行が、文字列を選ぶ `default` に隠れている。`status` の初期値も `default` の文面と同じ文字列で、読み手が `default` を見ないと対応が分からない | `internal/org/reseat.go:1510-1521`(`status := "it reports the server not running"`、`switch { case perr != nil … case running … default: r.res.HerdrStopped = true }`) | `running, perr := r.herdrProbe()` の直後に `if perr == nil && !running { r.res.HerdrStopped = true }` を別に書き、`switch` は文面だけを選ぶ。打ち直しの command が変わる条件(`HerdrStopped`)が 1 か所で読める |
| LOW (L4) | readability | `printOrgReseatHerdrServerNotes` は `rt.Reseat` の前に呼ばれるので、`--restart-herdr` の TUI の note(「stops the herdr server, which also closes any herdr TUI …」)が、そのあとの `PlanReseat` の断り(herdr の pane の中から打った、サーバーが動いていない、ほかの pane がある)の前に出る。断られた run の stderr が「止める」の note に続けて「何も止めずに断る」と読める。`lsof` と `ps` の read も、断られる run で先に走る | `internal/cli/org_reseat.go:324`(呼び出し)、`:520-521`(`orgReseatTUINote`)。`PlanReseat` の断りは `rt.Reseat`(`:327` と `:335`)の中 | TUI の note は `OnPlan`(real では 1 回目の pane の確かめのあと、dry run では plan のあと)で出す。または文面を「止めるときは」に寄せる。`herdr server:` の行は spec FR-7 のとおり断りの前で構わない |

## 打ち直しの command の確認(`finish`、`reseatRetryHerdrFlags`)

`herdr.restartHerdr = r.plan.RestartsHerdr && !r.res.HerdrStopped` を、失敗の 6 点で追った。どれも打ち直しの command は文書(director の skill、help)の説明と合う。

| 失敗した点 | `HerdrStopped` | 打ち直しの `--restart-herdr` | サーバーの状態と、打ち直しの先 |
| --- | --- | --- | --- |
| 座席の stop が失敗(`stopAll` が false) | false | 付く | サーバーは触っていない。打ち直しも restart する。残りの座席は全部 `--seat` に入る |
| 2 回目の pane の一覧が読めない、ほかの pane がある | false | 付く | サーバーは動いている。pane が残る限り打ち直しも同じ理由で断られる(skill に書いてある) |
| `herdr server stop` が失敗、probe は running か読めない | false | 付く | サーバーはまだ動いているかもしれない |
| `herdr server stop` が失敗、probe は not running | true | 付かない | サーバーは止まり、gate は走っていない。打ち直しの `PlanReseat` の gate が `--claude-config-dir` でサーバーを起こす |
| 止まるのを待つのが 10 秒で時間切れ | false | 付く | あとで止まれば、打ち直しは「サーバーが動いていない」と断られ、断りの文が `--restart-herdr` を外すよう言う(`reseatRestartHerdrErr`) |
| 止まったあとの gate が失敗、または座席の start が失敗 | true | 付かない | 立った座席をもう 1 度止めずに、残りだけ立てる |

`--claude-config-dir` は、渡していれば 6 点のどれでも付く(`reseatRetryHerdrFlags`、`""` は `ShellWord` で `''`)。

## Positive notes

- 窓の広さ: 2 回目の `reseatOtherPanes` と `ServerStop` の間に、`r.attempted = true` の代入と空の `len(others)` の確かめがあるだけで、呼び出しはない(`reseat.go:1491-1510`)。tech-debt の行(最後の行)の「その間に ralph の call はない」は、コードと合う
- pane の自分のもの判定は id だけ(`reseatOwnPanes`)だが、`herdr` が id を振り直すと別の pane の id が座席の記録の id と同じになりうる(`verbs.go:1201` のコメントの AC14)。この diff では、そのとき `Stop` が `confirmSeatPane` の label の確かめで失敗する(`verbs.go:1581`)ので、サーバーを止める前に `stopAll` が false で終わる。穴にはならない。label まで見る必要は出ていない
- 読めなくても断らない: `readOrgHerdrServerClaudeConfigDir` はエラーを返さず `orgHerdrServerEnvUnreadable` と理由を返し、`printOrgReseatHerdrServerNotes` はそれを表示するだけで、どの枝も return error しない。`orgHerdrServerEnvUnreadable` を 0 値にして、埋め忘れた結果が unset に読まれないようにしてある
- `/proc/<pid>/environ` の経路は NUL 区切りの完全一致で、値の空白と `=` を扱える(`orgEnvironValue`)。`ps eww` の stdout は、失敗したときのエラーに入れない(`runOrgHerdrServerEnvCommand` は stderr だけ)。`ps` と `lsof` の env は `PATH` と `LC_ALL=C` だけにしてある
- 複数 pid: 1 つならそれ、複数なら `comm` が `herdr` の 1 つだけを選び、0 または 2 以上なら読めない扱い。本物のサーバー(socket `~/.config/herdr/herdr.sock`)で `lsof -t` を 1 回読んだところ、`herdr` の process が 3 つ(サーバー 1 つと端末側の client 2 つ)あっても pid はサーバーの 1 つだけで、TUI をつないだ状態でも 1 pid の経路になる
- `orgHerdrServerEnvOverride` の zero 値は `apply` が env をそのまま返すので、既存の gate(`ensureOrgHerdrServer`、`wireOrgHerdrServerGate`)の挙動は変わらない。`''` は `CLAUDE_CONFIG_DIR` の全 entry(名前だけの `CLAUDE_CONFIG_DIR` も)を消し、別の名前(`CLAUDE_CONFIG_DIRX`、`MY_CLAUDE_CONFIG_DIR`)は残す
- `ServerStop` は `checkHerdrCloseResult` の約束に乗せ、実機の記録(stdout が空で exit 0)に合わせて成功時の reply を読まない
- `reseatOtherPanesText` と `reseatOtherPanesHint` を 1 回目と 2 回目で共有し、断りの文面が分かれない。`Cwd` の `null` は `string` に decode されて空になる

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この run は新しい行を足さない)。M1 と L1〜L4 はこの branch の中で直せる。最後の pane の確かめと `herdr server stop` の間の窓は、実装側がすでに `docs/tech-debt/README.md` の最終行に書いた(Design decisions 3)ので、重ねて書かない | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。)_

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。MEDIUM 1 件と LOW 4 件)
- Follow-ups:
  - M1 は、直す変更が小さく(正規表現 1 行とテスト)、直さないと別の env の値が stderr と director の文脈に出うるので、merge の前に直すことを勧める。直したら、post-implementation pipeline は `/self-review` から全部回し直す(`post-implementation-pipeline.md`)
  - L1〜L4 は同じ直しにまとめるか、記録して進めるかを実装側が決める。L2 はコメント 1 行、L1 は wrapper の削除とテスト 1 行
  - 窓(最後の pane の確かめと stop の間)は、依頼のとおり指摘に入れていない
- Known gaps: `.agents/skills/` と `templates/base/` の skill の写しが `.claude/skills/` と同じかは見ていない(`/verify` の `check-skill-sync.sh` が見る)。実機で `ralph org reseat --restart-herdr` を座席のある org に通しで打つことは、diff の記録(`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`)のとおり未確認で、この review でも見ていない
