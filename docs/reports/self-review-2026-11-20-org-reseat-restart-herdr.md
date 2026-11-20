# Self-review report: org-reseat-restart-herdr

- Date: 2026-11-20(cycle 1。run 1 のあとに直しの commit `53c12663` が入り、run 2 で見直した。run 1 の節は履歴としてそのまま残し、run 2 は末尾の「Run 2」以下に足した)
- Plan: `docs/plans/active/2026-11-20-org-reseat-restart-herdr.md`
- Branch: `feat/org-reseat-restart-herdr`、base は `origin/main` の `a672b72a`(issue #354)。run 1 は HEAD `5e99a0fe`、run 2 は HEAD `53c12663`
- Reviewer: reviewer サブエージェント(`/self-review`)
- ID の付け方: run 1 の指摘は M1 と L1〜L4 のまま。run 2 の新しい指摘は L5 から
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
