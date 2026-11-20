# Self-review report: org-reseat-restart-herdr

- Date: 2026-11-20(cycle 1。run 1 のあとに直しの commit `53c12663` が入り、run 2 で見直した。そのあとに `f620a6e8` と `562e1ab5` が入り、run 3 で見直した。run 1 と run 2 の節は履歴としてそのまま残し、run 2 は「Run 2」以下、run 3 は末尾の「Run 3」以下に足した)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`
- Branch: `feat/org-reseat-restart-herdr`、base は `origin/main` の `a672b72a`(issue #354)。run 1 は HEAD `5e99a0fe`、run 2 は HEAD `53c12663`、run 3 は HEAD `562e1ab5`
- Reviewer: reviewer サブエージェント(`/self-review`)
- ID の付け方: run 1 の指摘は M1 と L1〜L4 のまま。run 2 の新しい指摘は L5 から L7。run 3 の新しい指摘は L8 から
- Scope(run 1): `git diff origin/main...HEAD` の 28 ファイル(+4020 −120)のうち、コードと文書の diff の質だけ。コードは S1 `558227cf`(driver の `PaneListAll`・`ServerStop`・`HerdrPane.Cwd`、`org_herdr_server_env.go`、`org_herdr_server.go` の env 上書き)と S2 `e00adaf5`(`internal/org/reseat.go`、`internal/cli/org_reseat.go`)。文書は `3a83dbf0`(director・org の skill、README、AGENTS.md)と spec・tech-debt の追記。テストファイルは、本番コードの seam と合うかだけ見た。テスト、静的解析、spec 適合、文書のずれの検査は行っていない(`/verify`、`/test`、`/sync-docs` の担当)。

## Evidence reviewed (run 1)

- `internal/org/reseat.go`: diff 全体と周辺。`PlanReseat` の 9・10 番(`reseatRestartHerdrErr` と gate の順)、`Reseat` の dry run と real の 2 つの経路、`reseatFirstPanesErr`、`reseatOtherPanes`、`reseatOwnPanes`、`restartHerdr`(2 回目の pane の確かめ、`herdr server stop`、`waitHerdrStopped`、gate)、`finish`、`reseatRetryHerdrFlags`、`reseatRetryCommandWithHerdr`。`checkHerdrServerGate`(`herdrgate.go`)と `callWithTimeout`(`verbs.go:1091`)も読んだ
- 打ち直しの command の確認: 失敗の 6 点で `HerdrStopped` と `--restart-herdr` の有無を追った(下の表)
- `internal/cli/org_herdr_server_env.go` 全 358 行、`internal/cli/org_herdr_server.go` の diff、`internal/cli/org_reseat.go` の diff(help、flag、`runOrgReseat`、`printOrgReseatHerdrServerNotes`、probe、dry run と result の出力)
- `internal/org/driver/herdr.go` の diff(`PaneListAll`、`herdrPaneListResult`、`ServerStop`、`Cwd`)
- 文書: `.claude/skills/director/SKILL.md` と `.claude/skills/org/SKILL.md` の diff(`.agents/` と `templates/base/` の写しは見ていない。同期は `/verify`)、README と AGENTS.md の語単位の diff、tech-debt の新しい行、spec の 2 つの追記、`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`
- 周辺の確認: `reseatRetryCommand` の呼び出し元(`grep`)、`confirmSeatPane`(`verbs.go:1195`)が Stop の中で pane の label を確かめること、`HERDR_PANE_ID` の他の読み手
- 自分で走らせた probe(scratch は `$TMPDIR` 配下、repo の外): `orgPSEnvValue` と `orgPSEnvKey` の写しに、識別子でない名前の env を混ぜた `ps eww` 風の行を 4 つ。読み取りだけの `lsof -t` と `ps` をこのマシンの本物の herdr の socket に 1 回(下の「よい点」)

## Findings (run 1)

この run の指摘は M1 と L1〜L4。run 2 での状態は、末尾の「Run 2: run 1 の指摘の状態」にある(5 件とも `53c12663` で直った)。CRITICAL と HIGH はない。

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

## Positive notes (run 1)

- 窓の広さ: 2 回目の `reseatOtherPanes` と `ServerStop` の間に、`r.attempted = true` の代入と空の `len(others)` の確かめがあるだけで、呼び出しはない(`reseat.go:1491-1510`)。tech-debt の行(最後の行)の「その間に ralph の call はない」は、コードと合う
- pane の自分のもの判定は id だけ(`reseatOwnPanes`)だが、`herdr` が id を振り直すと別の pane の id が座席の記録の id と同じになりうる(`verbs.go:1201` のコメントの AC14)。この diff では、そのとき `Stop` が `confirmSeatPane` の label の確かめで失敗する(`verbs.go:1581`)ので、サーバーを止める前に `stopAll` が false で終わる。穴にはならない。label まで見る必要は出ていない
- 読めなくても断らない: `readOrgHerdrServerClaudeConfigDir` はエラーを返さず `orgHerdrServerEnvUnreadable` と理由を返し、`printOrgReseatHerdrServerNotes` はそれを表示するだけで、どの枝も return error しない。`orgHerdrServerEnvUnreadable` を 0 値にして、埋め忘れた結果が unset に読まれないようにしてある
- `/proc/<pid>/environ` の経路は NUL 区切りの完全一致で、値の空白と `=` を扱える(`orgEnvironValue`)。`ps eww` の stdout は、失敗したときのエラーに入れない(`runOrgHerdrServerEnvCommand` は stderr だけ)。`ps` と `lsof` の env は `PATH` と `LC_ALL=C` だけにしてある
- 複数 pid: 1 つならそれ、複数なら `comm` が `herdr` の 1 つだけを選び、0 または 2 以上なら読めない扱い。本物のサーバー(socket `~/.config/herdr/herdr.sock`)で `lsof -t` を 1 回読んだところ、`herdr` の process が 3 つ(サーバー 1 つと端末側の client 2 つ)あっても pid はサーバーの 1 つだけで、TUI をつないだ状態でも 1 pid の経路になる
- `orgHerdrServerEnvOverride` の zero 値は `apply` が env をそのまま返すので、既存の gate(`ensureOrgHerdrServer`、`wireOrgHerdrServerGate`)の挙動は変わらない。`''` は `CLAUDE_CONFIG_DIR` の全 entry(名前だけの `CLAUDE_CONFIG_DIR` も)を消し、別の名前(`CLAUDE_CONFIG_DIRX`、`MY_CLAUDE_CONFIG_DIR`)は残す
- `ServerStop` は `checkHerdrCloseResult` の約束に乗せ、実機の記録(stdout が空で exit 0)に合わせて成功時の reply を読まない
- `reseatOtherPanesText` と `reseatOtherPanesHint` を 1 回目と 2 回目で共有し、断りの文面が分かれない。`Cwd` の `null` は `string` に decode されて空になる

## Tech debt identified (run 1)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この run は新しい行を足さない)。M1 と L1〜L4 はこの branch の中で直せる。最後の pane の確かめと `herdr server stop` の間の窓は、実装側がすでに `docs/tech-debt/README.md` の最終行に書いた(Design decisions 3)ので、重ねて書かない | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。)_

## Recommendation (run 1)

- Merge: yes(CRITICAL と HIGH はない。MEDIUM 1 件と LOW 4 件)
- Follow-ups:
  - M1 は、直す変更が小さく(正規表現 1 行とテスト)、直さないと別の env の値が stderr と director の文脈に出うるので、merge の前に直すことを勧める。直したら、post-implementation pipeline は `/self-review` から全部回し直す(`post-implementation-pipeline.md`)
  - L1〜L4 は同じ直しにまとめるか、記録して進めるかを実装側が決める。L2 はコメント 1 行、L1 は wrapper の削除とテスト 1 行
  - 窓(最後の pane の確かめと stop の間)は、依頼のとおり指摘に入れていない
- Known gaps: `.agents/skills/` と `templates/base/` の skill の写しが `.claude/skills/` と同じかは見ていない(`/verify` の `check-skill-sync.sh` が見る)。実機で `ralph org reseat --restart-herdr` を座席のある org に通しで打つことは、diff の記録(`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md`)のとおり未確認で、この review でも見ていない

---

# Run 2(HEAD `53c12663`、run 1 の直しの見直し)

- Scope(run 2): `53c12663` の diff の全部(コード 5 ファイル)、`git diff a672b72a..HEAD` の通読(`internal/org/reseat.go` の diff 全体、`internal/cli/org_reseat.go` の diff 全体、`internal/org/driver/herdr.go`、`internal/cli/org_herdr_server.go`、`internal/org/spawn.go` の diff、`internal/cli/org_herdr_server_env.go` の全体、org skill の diff)。テスト、静的解析、spec 適合、文書のずれの検査は行っていない。窓(最後の pane の確かめと `herdr server stop` のあいだ)は依頼どおり指摘に入れていない。`53c12663` は窓を広げていない(`restartHerdr` の 2 回目の pane の一覧と `ServerStop` のあいだの変更は、`HerdrStopped` の代入を `ServerStop` が失敗したあとの枝へ出しただけ)

## Run 2: run 1 の指摘の状態

| ID | Severity | `53c12663` の直し | run 2 の確認 | 状態 |
| --- | --- | --- | --- | --- |
| M1 | MEDIUM | `orgPSEnvKey` を `` ` [^ =]+=` `` に広げ(`org_herdr_server_env.go:284`)、`orgPSEnvValue` の doc を書き直した。13 case の table test `TestOrgPSEnvValue_TheValueEndsAtTheNextEntryOfAnyName` を足した | 本物の `ps eww` の出力で確かめた。env を `env -i` で与えた、SIP 保護でない Go の process(`CLAUDE_CONFIG_DIR` の次に `X-Y=SECRET1`)に、旧と新の pattern の写しを当てると、旧の value は `… dir/.claude X-Y=SECRET1`、新は `… dir/.claude` だった。test の 13 case を旧 pattern に手で通すと、落ちるのは hyphen、dot、bash 関数、数字始まり、空値のあとの hyphen、2 つ目の entry の前の hyphen の 6 件で、commit message の「6 件落ちる」と合う。ライブの herdr サーバーの env に新旧を当てると、値の長さ(113、空白を含む)も found も同じで、既存の値は変わらない(値は出さず、長さと真偽だけを読んだ) | 直った。ただし、広げたことで出る別の影響を L5・L6 に書いた |
| L1 | LOW | `reseatRetryCommand` を消し、`reseatRetryCommandWithHerdr` の doc に wrapper の説明を移した。テストは `reseatRetryCommandWithHerdr(…, reseatRetryHerdr{}, …)` を呼ぶ | `.go` で `reseatRetryCommand\b` に当たるのは `TestReseatRetryCommand` の名前と、その上の comment だけ。定義と呼び出しは `reseat.go:1878-1884` の 1 つ | 直った |
| L2 | LOW | `PlanReseat` の 9 番の comment を、「9 番で断られた restart は watch もサーバーも起こさない」「`reseatFirstPanesErr` は gate の後で、断られると watch は起きていることがある。サーバーは 9 番が動いていると確かめ済みで、何も止めない」に直した(`reseat.go:425-433`) | コードの順と合う。real の `Reseat` は `PlanReseat`(`checkWatchGate` を含む)のあとで `reseatFirstPanesErr` を呼ぶ(`reseat.go:1367`、dry run は `:1336`)。dry run は gate を読むだけで起こさない | 直った |
| L3 | LOW | `HerdrStopped` の代入を `if perr == nil && !running` に出し、`switch` は文面だけを選ぶ(`reseat.go:1514-1525`) | `status` の初期値(`it reports the server not running`)は、`if` が立つ場合の文面と一致する。`HerdrStopped` を読むのは `finish` の 1 行(`reseat.go:1682`)だけで、条件は 1 か所で読める | 直った |
| L4 | LOW | 断りの前に出る TUI の note を「`once its checks pass, --restart-herdr stops …`」に変えた。comment が理由を書く(`org_reseat.go:517-523`) | 断られた run の stderr でも偽にならない。`lsof` と `ps` の read は断られる run でも先に走るが、読み取りだけで 1 回 5 秒の上限がある。note を比べるテスト 3 件は `orgReseatTUINote` の定数を使うので、文面を変えても落ちない | 直った(文面を寄せる案) |

## Evidence reviewed (run 2)

- `git show 53c12663` の 5 ファイル。`org_herdr_server_env.go` は全 366 行を読み直し、`readOrgHerdrServerClaudeConfigDir` の `ps eww` の経路と `orgHerdrServerEnvCommandEnv`、`runOrgHerdrServerEnvCommand` まで追った
- `internal/org/reseat.go` の `PlanReseat` の comment(395-450)、`Reseat` の dry run と real の経路(1332-1400)、`restartHerdr`(1473-1541)、`waitHerdrStopped`、`finish`、`reseatRetryHerdrFlags`
- `internal/cli/org_reseat.go` の help、`runOrgReseat`、`printOrgReseatHerdrServerNotes`、`orgReseatHerdrServerLine`、`orgReseatClaudeConfigDirMismatchNote`(270-330、517-660)
- 自分で走らせた probe(scratch は `$TMPDIR` 配下、終わったら消した):
  1. `ps eww -o command= -p <pid>` を、Go で作った sleep する process(`env -i` で env を指定、SIP 保護でない)に、`LC_ALL=C` と `en_US.UTF-8`、`ja_JP.UTF-8` で打った。`CLAUDE_CONFIG_DIR=/Users/x/日本語 dir/.claude` の `日本語` は、`LC_ALL=C` では `M-fM^WM-%M-fM^\M-,M-hM-*M^^` と出て、UTF-8 の locale では生の `日本語` で出た
  2. `orgPSEnvValue` の旧・新の pattern の写し(scratch の Go)を、上の出力と、SIP 保護の `/bin/sh -c 'sleep 3; :' --flag=1` の `ps eww`(env が出ない)に当てた。後者は旧が `found=false anyEnv=false`、新が `found=false anyEnv=true`
  3. このマシンの本物の herdr サーバー(`herdr status server --json` の socket、`lsof -t` が 1 pid)の `ps eww` を読み取りだけで見た。command は `/opt/homebrew/bin/herdr server` で `=` を含む語がなく、env は読めて(旧 pattern で 96 entry)、`CLAUDE_CONFIG_DIR` が 1 つある。値は出さず、旧と新で長さと found を比べただけ

## Findings (run 2)

この run の新しい指摘は L5〜L7(すべて LOW)。CRITICAL、HIGH、MEDIUM はない。L5 と L7 は run 1 が見落とした箇所で、`53c12663` が持ち込んだものではない。L6 は `53c12663` の pattern の広げ方に由来する。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L5) | correctness(読んだ値が壊れる) | `ps eww` の read は `LC_ALL=C` で打つ(`orgHerdrServerEnvCommandEnv`)。macOS の `ps` は `LC_ALL=C` のとき、env の値の 0x80 以上の byte を `M-f` `M^W` の形に escape して出す。`CLAUDE_CONFIG_DIR` が非 ASCII の path(例 `/Users/x/日本語 dir/.claude`)のサーバーでは、`herdr server: CLAUDE_CONFIG_DIR=/Users/x/M-fM^WM-%M-fM^\M-,M-hM-*M^^ dir/.claude` と出る。`--restart-herdr` なしでは、そのあと `got.value == value`(`org_reseat.go:555`)が、座席に使わせる値と同じ path でも偽になり、「seats start with its environment, so they will not use …; add --restart-herdr」の note が出る。表示だけで断りには関わらず、Linux の `/proc` の経路(NUL 区切りの生の byte)は当たらない | `org_herdr_server_env.go:331-337`(`LC_ALL=C` の env)、`:194`(`ps eww` の呼び出し)、`:349-352`(その env で走らせる)。comment の理由は「PS_FORMAT、PS_PERSONALITY、locale が出力を変えない」(`ReadDirectorProcess` の `directorPSEnv` の写し)だが、あちらは `stat=,lstart=`(状態と開始時刻)だけを読み、ここは値をそのまま出して比べる。probe 1 の出力のとおり | `ps eww` の read は locale を `LC_ALL=C` に固定しない。macOS には `en_US.UTF-8` があるので、`ps eww` の環境だけ `LC_ALL=en_US.UTF-8` にする(`lsof` と `comm` の read は `C` のままで構わない)。固定が難しければ、読んだ値に `M-` の escape が入るとき(0x80 以上の byte を出す path は escape される)は、一致を判定せず `cannot be compared` と出す。test には、非 ASCII の値を持つ fake の run を足す |
| LOW (L6) | maintainability(guard の comment と実際のずれ) | `orgPSEnvKey` を 1 語の `=` つきの語まで広げたので、command の語(`--flag=1`、`--config=/x`)も entry に数えられる。env が出ない process(SIP 保護の binary など)の command にそういう語があると、`anyEnv` が真になり、`readOrgHerdrServerClaudeConfigDir` の「env が 1 つも出なければ読めない扱い。unset に読ませない」という保証(doc の 160-163 行、`:203-204`)を抜けて、`unset` と出る。旧 pattern では `anyEnv` は偽で「読めない」と出た。判断: この PR では影響しない。ralph が起こすサーバーの argv は `server` だけ(`org_herdr_server.go:310`)で、本物のサーバーの command も `/opt/homebrew/bin/herdr server`、env は読める。起きるのは、(1) herdr が env を隠す process で、(2) 起動時の引数に `=` の語がある、の両方のときだけで、出る結果も 1 行の表示と note だけ。`orgPSEnvValue` の comment は原因(「a word of the command with "=" in it counts as an entry too」)を書くが、`anyEnv` が上の guard を弱めることは書かない | `org_herdr_server_env.go:278-312`(pattern と comment)、`:160-163`、`:200-206`。probe 2: SIP 保護の `/bin/sh -c 'sleep 3; :' --flag=1` で、旧 `anyEnv=false`、新 `anyEnv=true`。probe 3: 本物のサーバーは `=` の語を持たない | 直さなくてよい。直すなら、`anyEnv` だけは旧の識別子の形(`` ` [A-Za-z_][A-Za-z0-9_]*=` ``)で数える(entry の切れ目は広い pattern のまま)と、guard は旧と同じになる。直さないなら、`orgPSEnvValue` の comment に「anyEnv は command の語も数える」と 1 文足す |
| LOW (L7) | readability(note の文が崩れる) | `orgReseatClaudeConfigDirMismatchNote` は、座席に使わせる値が「なし」のとき `describe` が `no CLAUDE_CONFIG_DIR` を返し、文の `the %s` と `this command's %s` に入れる。`--claude-config-dir ''` でサーバーが別の値を持つと、「they will not use the no CLAUDE_CONFIG_DIR that --claude-config-dir asks for」、`--claude-config-dir` なしで呼んだ側に値がないときは「will not use this command's no CLAUDE_CONFIG_DIR」と出る。director(LLM)がこの行を読むので、文として読める方がよい | `org_reseat.go:598-614`(`describe` と 2 つの `Sprintf`)。テストが文を写して固定している(`org_reseat_restart_test.go:358-360`、`:386`、`:394` の `mismatchEnv` / `mismatchFlag`) | 「なし」を名詞句で埋めない。たとえば「…so they will not run with %s, which --claude-config-dir asks for」と、「なし」のときも文が通る形にする。テストの 2 つの定数を合わせて直す |

### 「次の entry の名前を広げた」ことの判断(依頼の確認事項)

- M1 の直しは、旧 pattern が漏らしていた形(`X-Y=SECRET1`)を、本物の `ps eww` で止める。値が「短くなる」側にしか間違えないので(` <word>=` を含む値は手前で切れる)、別の entry の値が出る向きの誤りはなくなった。切れた値は表示と比較にだけ使われ、`unset` や不一致の note になるだけで、断りにも書き込みにも使われない。ライブのサーバーの値は空白を含む path で、新旧で同じ長さに読めた(空白だけの語は区切りにならない)
- 「読めない」が「unset」に変わりうることは、L6 のとおり。この PR の経路では起きず、直さなくてよい。ただ、保証の comment と実際がずれたので、LOW として残した

## Positive notes (run 2)

- 新しい table test は、M1 の漏れの 4 形(hyphen、dot、bash 関数、数字始まり)に、空値、最後の entry、空白を含む値、値の中の `<word>=` で切れる case(`a b=c` が `/Users/x/a` になること)、`CLAUDE_CONFIG_DIRX` の取り違え、最初の entry が勝つ case、env が 1 つもない行を入れた。`strings.Contains(value, "SECRET")` の確かめが、漏れの向きを直接見る。切れる case を test に書いて、comment の「shorter, never longer」と対にしてある
- L3 の直しで、`HerdrStopped` を立てる条件が `if` 1 つになり、`finish` の打ち直しの command(`restartHerdr = RestartsHerdr && !HerdrStopped`)を決める値の変わる場所が 1 つに見える。run 1 の表(6 つの失敗点)は、この変更のあとも同じ結果になる(`HerdrStopped` の値は変わらない)
- L2 の comment は、`reseatFirstPanesErr` が gate のあとだという事実と、サーバーを起こさない理由(9 番が動いていると確かめ済み)を両方書き、run 1 で偽だった「どちらも起こさない」が消えた
- `53c12663` の 5 ファイル(本番コード 3、テスト 2)の変更は、すべて run 1 の指摘に対応する。無関係な変更はない

## Tech debt identified (run 2)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この run は新しい行を足さない)。L5〜L7 はこの branch の中で直せる小さい指摘で、直すか記録するかは実装側が決める。記録に回すなら、L5 は `docs/tech-debt/README.md` の reseat の行に足す | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。run 1 の注記のとおり、tech-debt の行 518〜520 の `reseatRetryCommand` の履歴と、行 519 の trigger(`PlanReseat` を `reseat_plan.go` へ移す)が当たったことは `/sync-docs` の担当。)_

## Recommendation (run 2)

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。LOW 3 件 L5〜L7。run 1 の M1 と L1〜L4 は `53c12663` で全部直った)
- Follow-ups:
  - L5 は、非 ASCII の path の `CLAUDE_CONFIG_DIR` を使う人(日本語の dir 名など)に、誤った不一致の note と壊れた表示を出す。直しは `ps eww` の環境の `LC_ALL` 1 行とテスト 1 つ。merge を止めるほどではないが、`--restart-herdr` の note が間違った理由で出るので、直すことを勧める
  - L6 は記録だけでよい。L7 は文面の直し(2 つの `Sprintf` とテストの 2 定数)
  - コードを変える直しなら、パイプラインは `/self-review` から全部回し直す(`post-implementation-pipeline.md`。cycle は 1 で、上限は 2)
- Known gaps: `.agents/skills/` と `templates/base/` の skill の写しが `.claude/skills/` と同じかは見ていない(`/verify` の `check-skill-sync.sh`)。実機で `ralph org reseat --restart-herdr` を座席のある org に通しで打つことは、run 1 と同じく未確認。L5 は macOS の `ps` で確かめた。Linux の `ps` は `/proc` の経路を通るので見ていない。ライブのサーバーの `CLAUDE_CONFIG_DIR` の値は、値を出さずに長さと真偽だけを確かめた

---

# Run 3(HEAD `562e1ab5`、run 2 の直しと、verify の I6 の追加の見直し)

- Scope(run 3): `f620a6e8`(run 2 の L5・L7 の直し)と `562e1ab5`(verify の I6、V4・I2・I5 の comment の直し、6 つの関数の移動)の diff の全部。ほかは、この 2 つが持ち込んだ後退があるかだけ見た(`reseat.go` の `HerdrRestarted` の立つ所、`printOrgReseatResult`、`orgReseatHoldSignals`、`readOrgHerdrServerClaudeConfigDir`)。テスト、静的解析、spec 適合、文書のずれの検査は行っていない。窓(最後の pane の確かめと `herdr server stop` のあいだ)は、run 1 と同じく指摘に入れていない

## Run 3: run 2 の指摘の状態

| ID | Severity | 直し | run 3 の確認 | 状態 |
| --- | --- | --- | --- | --- |
| L5 | LOW | `orgHerdrServerEnvCommandEnv(name, args...)` が `ps eww` にだけ `LC_ALL=en_US.UTF-8` を返す(`org_herdr_server_env.go:336-358`)。`lsof` と `ps -o comm=` は `C` のまま。runner の test を command ごとの locale に、非 ASCII の値を fake の `lsof`・`ps` に通す test を足した | macOS の本物の `ps eww` で確かめた(Go の sleep する process に `env -i` で `CLAUDE_CONFIG_DIR=/Users/x/日本語 dir/.claude` を与えた)。`en_US.UTF-8` は `日本語` を生で、`C` は `M-fM^WM-%M-fM^\M-,M-hM-*M^^` で出した。存在しない locale(`xx_YY.UTF-8`)も `C` と同じ escape で、doc の「A system without that locale runs ps under C, which escapes such a value as before」と合う。`orgHerdrServerEnvCommandEnv` の呼び出しは `runOrgHerdrServerEnvCommand` の 1 か所(`:372`)だけ。`ps eww` を呼ぶのは `readOrgHerdrServerClaudeConfigDir` の 1 か所(`:199`)だけで、`ps -o comm=` の経路(`orgHerdrServerPID`)は `C` のまま。narrow な `go test -run` で、runner の test と新しい read の test(`TestOrgHerdrServerReadClaudeConfigDir_PSReadsANonASCIIValueUnderUTF8`)が通ることを確かめた | 直った |
| L6 | LOW | 直さず、doc に書く方針。`562e1ab5` が `readOrgHerdrServerClaudeConfigDir` の doc(`org_herdr_server_env.go:165-168`)に `orgPSEnvValue` の「`=` を含む語も entry に数える」と、ralph の起こすサーバーには当たらない理由を足した(I5) | 書いてあることは run 2 の確認と合う(`herdr server` の argv は `=` の語を持たない)。guard の doc が条件なしで書いていた食い違いは消えた | 解消(記録のみ) |
| L7 | LOW | note の 2 文を「will not run with <値>」に替えた(`org_reseat_herdr_server.go:117-127`)。test に「server が set で呼んだ側に値がない」「server が unset でフラグの値」「unset でフラグが消す(note なし)」を足した | 文は set と unset のどちらでも、flag あり・なしのどちらでも読める。ほかの文書・skill・spec に旧い文面(`will not use`)は残っていない(`grep` は新しいコードの 1 件だけ)。`TestOrgReseat_HerdrServerLineAndNotes` は `562e1ab5` では変わらず、そのまま通る | 直った |

## Evidence reviewed (run 3)

- `git show f620a6e8` と `git show 562e1ab5` の全文(コード 6 ファイル、テスト 3 ファイル)。新しい `internal/cli/org_reseat_herdr_server.go` は全 179 行を読んだ
- 移動の確かめ: `562e1ab5^` の `org_reseat.go` から、移った塊(`orgReseatTUINote` の doc の先頭から、`// The notes a dry run ends with` の直前まで、107 行)を取り出し、新しい file の 18-134 行と `diff` した。違いは 2 か所だけで、どちらも commit message が言う 2 つの helper の取り出し。(a)`printOrgReseatHerdrServerNotes` の `serverSet == set && (!set || got.value == value)` が `orgReseatHasWantedClaudeConfigDir(got, value, set)` に、(b)`orgReseatClaudeConfigDirMismatchNote` の局所の `describe` が `orgReseatClaudeConfigDirText` に(定義の移動と、3 つの呼び出し、doc の 1 行)。残りの `orgReseatTUINote`、`orgReseatHerdrServerLine`、`orgReseatWantedClaudeConfigDir`、`orgReseatClaudeConfigDirRunningNote` は、comment まで 1 byte も変わらない。`orgReseatHasWantedClaudeConfigDir` の式 `(got.state == orgHerdrServerEnvSet) == set && (!set || got.value == value)` は、旧の `serverSet == set && …` と同じ(`==` は `&&` より先に結ぶ)。`org_reseat.go` から消した import は `context` だけで、ほかに `context.` の使用はなく、`os` と `io` は残りの使用がある
- 出力の確かめ: 事前の stderr の行は、`562e1ab5` の文字列が `f620a6e8` と同じ(上の `diff` に文字列の変更がない)。`TestOrgReseat_HerdrServerLineAndNotes`(事前の行と note の全文を固定する)は `562e1ab5` で変わっておらず通る
- `HerdrRestarted` の立つ所: `reseat.go:1539` の 1 か所で、gate が通ったあと。断り(`res.Seats == nil`)、止める前の失敗、`herdr server stop` の失敗、止まるのを待つ時間切れ、gate の失敗では立たない。立ったあとの座席の start の失敗では立ったままで、`printOrgReseatResult` の `started: herdr server` と同じ条件で読み直す
- `runOrgReseat` の呼び出し(`org_reseat.go:336-340`): `err = printOrgReseatResult(...)` のあとに、戻り値のない `printOrgReseatRestartedHerdrServerNotes` を呼ぶ。`err` はそのあと触らず、read の失敗は行になるだけ。dry run は別の枝(`:328-334`)で、読み直さない
- test の通読: `org_reseat_restart_test.go` の増分(+209 行)、`setOrgReseatClaudeConfigDirReader` の variadic 化(0 個の呼び出しは `t.Fatal`、1 個は旧と同じ、`min(*reads, len(gots)-1)` で最後の答えを繰り返す)、`reseatCombined`(既存の `reseat` は stdout と stderr を分けて返すので、重複ではない)
- 自分で走らせた probe(scratch は `$TMPDIR` 配下の scratchpad、Go の sleep する process を `env -i` で起こして `ps eww` を読んだ。終わったら process を止めた):
  1. `en_US.UTF-8`、`C`、存在しない locale の `ps eww` の値の出し方(L5 の確認)
  2. 値に ESC(0x1b)、^A(0x01)、タブ、C1 の制御文字(U+009B、U+0085)、改行を入れた env を、`en_US.UTF-8` と `C` で。ESC と ^A は両方の locale で生で出る。タブは `\011`、改行は `\012`、C1 は `M^[` と `M^E` で、両方の locale で escape される
- 狭い `go test -run`(`TestOrgReseat_HerdrServerLineAndNotes`、`TestOrgReseat_RestartHerdrPrints`、`TestOrgReseat_NoServerLineAgain`、`TestOrgHerdrServerEnvCommand_`、`TestOrgHerdrServerReadClaudeConfigDir_PSReads`): `ok`(52 s)。全体の test は走らせていない

## Findings (run 3)

この run の新しい指摘は L8(LOW)1 件。CRITICAL、HIGH、MEDIUM はない。run 2 の L5・L7 の直しは、依頼の観点(移動が純粋か、事前の出力が変わらないか、事後の行が `HerdrRestarted` のときだけか、read の失敗が終了コードを変えないか、値が出ないか)のどれにも問題がない。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L8) | readability(note の文が、起きていないことを言う) | `orgReseatRestartedClaudeConfigDirMismatchNote` は「…and the seats started with the server's environment」と言い切る。`HerdrRestarted` は gate が通った時点で立ち(`reseat.go:1539`)、そのあとの最初の座席の start が失敗すると、起こされた座席は 0 なのにこの note が出うる(leader の start が失敗する場合など)。そのとき note は「座席がサーバーの環境で起きた」と言い、読む側(director の LLM)は、立っていない座席の環境を疑う。もう 1 つ、note は原因を「別の process がサーバーを起こしたかもしれない」の 1 つだけにしている。`ps eww` の値は、値の中に ` <語>=` があると手前で切れる(spec FR-7 が許す。M1 の直しの「shorter, never longer」)ので、その path でも不一致の note は出る。どちらも文面だけの問題で、事前の note(`orgReseatClaudeConfigDirMismatchNote`)は原因を言わず、この点が出ない | `org_reseat_herdr_server.go:176-178`(文面)、`reseat.go:1539`(`HerdrRestarted` は座席の start の前)、`org_reseat_restart_test.go` の `TestOrgReseat_RestartHerdrPrintsTheServerLineAfterAFailedStart` は leader が起きたあとの失敗だけで、起こされた座席が 0 の場合を持たない | 「and the seats started with the server's environment」を「and any seat reseat started has the server's environment」のように、座席がなくても偽にならない形にする。原因の 1 つだけという点は、今の「may have started」の hedge で足りるので、文面を足さなくてよい(`printOrgReseatRestartedHerdrServerNotes` の doc の「most likely」は、note の文面より強いので、doc 側を「may」に寄せてもよい)。文面を変えたら `restartedClaudeConfigDirNote`(test の定数)も合わせる |

### 判断して指摘にしなかった点

- 移動と編集を 1 つの commit に入れた(`562e1ab5`): 6 つの関数の移動と 2 つの helper の取り出しが同じ diff に入り、`git diff` は 113 行の削除と 179 行の追加としか出さない。commit message は「moved, unchanged」と、「became two small helpers」の両方を書き、上の `diff` で変更が 2 か所だけだと確かめられた。2 つの commit に分けたほうが読みやすかったが、直す価値のある害ではない
- 事後の読み直しは `orgReseatThroughSignals` の保持の中で走る(`org_reseat.go:336-340`)。`herdr status`、`lsof`、`ps` がそれぞれ 5 秒で打ち切られるので、止まった環境では最大でおよそ 15 秒、SIGINT などが捕まえられたまま待つ。`reseated` と結果はもう出たあとで、signal の note は「記録を書いて、したことを出すまで続ける」と言う。ただ、保持の外に出すと、stderr が閉じた pipe のとき SIGPIPE で process が終わり、終了コードが reseat の結果と変わる。保持の中に置く選択のほうが安全で、止まった環境の 15 秒は事前の read と同じ上限。直さない
- `orgHerdrServerEnvCommandEnv` は `name == "ps" && args[0] == "eww"` で locale を選ぶ。`"ps"` と `"eww"` の 2 つの文字列が、呼び出し側(`org_herdr_server_env.go:199`)とこの関数に 2 度出る。ただ、runner の test が command ごとの locale を固定し、新しい read の test は、fake の `ps` が `C` と `en_US.UTF-8` で違う出力を返すので、呼び出しの argv が変わって locale が `C` に戻れば落ちる。ずれは test が止める
- 表示する値の制御文字: ESC(0x1b)と ^A は、macOS の `ps eww` が両方の locale で生で出す(probe 2)。run 1 から同じで、`f620a6e8` は広げない(C1 は UTF-8 でも escape される)。Linux の `/proc` の経路は NUL 区切りの生の byte を出す。値はサーバーを起こした同じユーザーの env から来るので、権限の境はまたがず、指摘にしない。出す前に制御文字を落とすなら、`readOrgHerdrServerClaudeConfigDir` の 1 か所で足りる
- `orgReseatHasWantedClaudeConfigDir` は、`got` が `orgHerdrServerEnvUnreadable` のとき、`set` が偽なら「望んだ値」と答える(`state != Set` と `set == false` が一致するため)。doc は「read された `got`」と前提を書き、2 つの呼び出し元は両方とも `Unreadable` を先に外す(事前は `switch` の `default`、事後は `if got.state == orgHerdrServerEnvUnreadable { return }`)。3 つ目の呼び出し元ができるときの前提だが、今は害がない

## Positive notes (run 3)

- 6 つの関数を移した新しい file は、先頭の comment が事前と事後の 2 つの出力を並べて言い、関数の並びは事前の 3 つ、共有の helper、事後の 2 つ。`printOrgReseatHerdrServerNotes` と `printOrgReseatRestartedHerdrServerNotes` は、行の関数(`orgReseatHerdrServerLine`)と比較(`orgReseatHasWantedClaudeConfigDir`)と値の名前(`orgReseatClaudeConfigDirText`)を共有し、事前と事後で比較の式が分かれない
- `printOrgReseatRestartedHerdrServerNotes` は、`restarted` が偽、reader が nil、`Unreadable` の 3 つの出口がそれぞれ 1 行で、read の失敗は行の「cannot be read: <理由>」になるだけ。`reason` は lsof・ps の stderr と status のエラーで作られ、`ps eww` の stdout は入らない。出る値は `CLAUDE_CONFIG_DIR` の 1 つで、M1 の直し(` [^ =]+=` で切る)があるので、次の entry は含まない
- 新しい test の組み合わせ: `TestOrgReseat_NoServerLineAgainWithoutARestart` が、`--restart-herdr` なし、断り、stop のあとの gate の失敗の 3 つで、事後の行が出ないことと、read が 1 回であることを固定する。`HerdrRestarted` の条件を外す変更をその場で落とせる。`TestOrgReseat_RestartHerdrPrintsTheServersClaudeConfigDirAgain` は 7 つの組み合わせ(フラグ・呼び出した側・unset・読めない)の stdout と stderr の全行を、順序つきで固定する
- V4・I2・I5 の comment の直しは、どれも実機の記録(`docs/evidence/org-reseat-restart-herdr-live-2026-11-20.md` の 6 番: `herdr server stop` は stdout が空)と、`ServerStop` の doc に合う。I2 は、`/pr` が計画を `docs/plans/archive/` へ移したあとの path を書く(今は `active/`)

## Tech debt identified (run 3)

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(この run は新しい行を足さない)。L8 は文面 1 か所とテストの定数 1 つで、この branch の中で直せる | - | - | - | - |

_(行を足していないので、`docs/tech-debt/` への追記はない。)_

## Recommendation (run 3)

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。LOW 1 件 L8。run 2 の L5・L7 は `f620a6e8` で直り、L6 は `562e1ab5` の doc で解消した)
- Follow-ups:
  - L8 は note の文面だけで、merge を止めない。直すなら、コードを変える直しなので、post-implementation pipeline は `/self-review` から回し直す(`post-implementation-pipeline.md`。cycle は 1 のまま、上限は 2)。直さずに進めるなら、`/pr` の既知の穴に L8 を書く
  - 計画の Progress の「Review artifact created」の行(`docs/plans/active/2026-11-20-org-reseat-restart-herdr.md:127`)は run 2 までで、run 3 を持たない。実装側が足す
- Known gaps: run 1・2 と同じく、実機で `ralph org reseat --restart-herdr` を座席のある org に通しで打つことは見ていない(事後の行が実機で `started: herdr server` のあとにどう並ぶかは、fake の herdr の test の順序だけ)。`.agents/skills/` と `templates/base/` の skill の写しの一致、事後の行を文書(skill、README、spec)が説明しているかは、`/verify`(`check-skill-sync.sh`)と `/sync-docs` の担当で、この run は見ていない。Linux の `ps` は `/proc` の経路を通るので見ていない
