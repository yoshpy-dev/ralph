# Self-review report: trial-docs-wrapup

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-trial-docs-wrapup.md`(D1〜D5、AC1〜AC6)
- Reviewer: reviewer サブエージェント(`/self-review`、pipeline cycle 1/2)
- Scope: `git diff 3c1cd14b...HEAD`(HEAD f310e994、17 ファイル、+1359/-83)。S1 e9884d9b・493e4ce8・e07bf30b(recipe と templates の写し)、S2 be8acda8(skill 8 写し、leader の雛形、`prompts_test.go`、`report_names_sync_test.go`)、S3 63e629a3・f310e994(仕様、根拠の表、tech-debt、plan の進捗)

## Evidence reviewed

- recipe `docs/recipes/isolated-herdr-server.md`(445 行)を全文読み、`templates/base/docs/recipes/` の写しと `cmp` で一致を確かめた。skill 4 つの写し(`org`、`director` の root と templates の `.claude` と `.agents`)も `cmp` で一致した。
- recipe の主張を出典と突き合わせた。`org-send-enter-timing-2026-09-19.md` の 40〜43 行(HOME だけ変えた server がユーザーの `session.json` を読み、偽 HOME の claude が「Not logged in」)、`org-plan-approval-live-2026-10-13.md` 17 行(150 字超の socket path で herdr が起動しない)、`herdr-pane-close-2026-10-07.md` 46 行(`session.json` が残ると番号が続く)、`org-trial-2026-10-20.md` の Setup・F1・F2・Trust・Cleanup、`busy-leader-2026-10-23.md` の「分離のしかた」。3 つの旧記録の herdr はどれも 0.7.5 だった。
- recipe が頼る ralph の挙動をコードで確かめた。`internal/org/driver/herdr.go` の `workspace create`・`tab create`・`pane read/send-text/send-keys/run/close`・`workspace close` の引数と JSON の path(`result.workspace.workspace_id`、`result.root_pane.pane_id`)、`--env` を `tab create` に足す条件、`internal/cli/org_watch.go` の `orgWatchChildDroppedEnv`(4 変数)・`--stop` の 30 秒と時間切れのエラー、`internal/cli/status.go` の `org watch: not running`・`state-dir:`、`org director stop/detach` の `--id`、`ralph org report` の出力先(`docs/reports`)、`internal/cli/org_director_headless.go:404` の相対 `agmsg_home` の拒否、`doctor_shell_alias.go` の候補ファイル(zsh・bash・fish、HOME の下)。どれも recipe の文と合う。
- 試行の記録の数字を `docs/evidence/org-trial-2026-10-20/` の snapshot で数え直した。watcher の時間切れは `watch.log` の 9・22・27・29・34・41 行目の 6 件、seat 別は doctor-escape の implementer 3・reviewer 1・leader 1 と report-escape の leader 1、条件は `scope_change` 4 と `stall` 2、`model-receipts.jsonl` の時刻 6 件(03:38:24Z、03:49:46Z、03:51:04Z、03:52:22Z、04:25:54Z、04:44:39Z)も一致した。receipt の内訳は `verdict=normal` 16・`circular` 1・`fake_progress` 1・`watcher_error` 6 で 24 件、「watcher already in flight」は 11 行。gate の所要は 28、1947、1375、1664、1990、792 秒で、根拠の表の「1375〜1990 秒」「792 秒」「28 秒」と合う。
- 仕様の PR の一覧を数えた。#205 + 9 段の 24 本 + #241・#242 + 所見の直し 9 本(#244、#246〜#253)= 36 本。根拠の表の「33 本(9 段の 24 本と所見の直し 9 本)」とも合う。#253 の merge 時刻は `git log -1 --format=%cI 3c1cd14b` が 2026-10-23T12:45:45+09:00(= 03:45:45Z)で、表の `mergedAt` と合う。`gh pr view` は打っていない。
- `grep -rn 'fix/busy-leader' docs/specs docs/evidence docs/tech-debt` は 0 件。`数分` は skill 8 写し、`internal/org/prompts`、`internal`、`templates/base` のどこにも残っていない。
- tech-debt の追加 4 行と根拠の表の追加 5 行・書き換え行を、`(?<!\\)\|` で数えた。tech-debt は 7 分割(5 セル)、根拠の表の所見の行は 8 分割、FR と NFR の行は 6 分割、PR の表は 5 分割で、どれも見出しと同じ。RESOLVED の行は取り消し線と `<!-- RESOLVED ... -->` の注の両方を持つ。
- Go の差分(`report_names_sync_test.go` の +109、`prompts_test.go` の 1 行、`leader.md` の 1 文)を全文読み、`markdownSection`・`readRepoFileOrSkip`・`hasLine`・`protocol.HeaderValues`・`protocol.Validate` の定義と突き合わせた。`codeBlockAfterLine` の 5 つのケースを手で追った。
- recipe の手順の一部を、herdr を使わずに手元で確かめた。`lsof +D <dir>` の出力(F2)と `ps -E` の出力(F3)。`strings` で herdr の実体が `XDG_CONFIG_HOME`・`HERDR_CONFIG_PATH`・`XDG_STATE_HOME` を読む文字列を持つことを見た(実行はしていない)。
- 追加行を `/Users/`・`hiroki`・`@dena`・`ghp_`・`sk-`・`AKIA`・`xox`・`eyJ`・`BEGIN .* KEY`・`.local/share/mise`・`/tmp/ot` で grep した。当たりなし。U+FFFD は 0 件、`git diff --check` は空、TODO・FIXME・debug の追加なし。
- 確かめていない: テスト・gofmt・vet・`check-sync`・`check-skill-sync` の実行(`/verify` と `/test` の範囲)、herdr・agmsg・claude・codex の実行、recipe の通しの実行、`gh pr view`、`ps -E` が環境変数を出すかの確認(F3。この sandbox の shell では `ps -E`・`ps eww`・`ps -wwE` のどれも環境を出さなかった)、`herdr --help` の `Config:`・`Logs:` の行(F5)。

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (F1) | security | recipe の「Clean up」の 1・2 は `ralph org report`・`ralph org disband`・`ralph org director stop/detach`・`ralph org inbox`・`ralph org watch --all --stop`・`ralph status` を素の形で書き、3 だけが `/tmp/ralph-iso/run herdr server stop` と wrapper を付けている。wrapper を使えという文は 92〜94 行と 352 行にあるだけで、ユーザー自身の herdr server に届く恐れがいちばん大きい片付けの節にない。素の `ralph org disband` は `HERDR_SOCKET_PATH` を持たないので既定の socket(ユーザーの server)の herdr を呼ぶ。disband は閉じる前に tab の label と workspace の label を確かめるので、ユーザーの pane を閉じる見込みは小さいが、その確かめに頼る書き方になっている | `docs/recipes/isolated-herdr-server.md:371-383` と `:395`。`internal/cli/org.go:1823-1840`(`ralph org disband` の help: closing の前に label を確かめ、合わないものは閉じない) | 「Clean up」の冒頭に「この節のコマンドはすべて `/tmp/ralph-iso/run` を通し、repo の main のチェックアウトから打つ」の 1 文を足す。または各コマンドの頭に wrapper を付ける |
| LOW (F2) | maintainability | 「Clean up」2 と 3 の確かめ「`lsof +D /tmp/ralph-iso` が server 以外の process を出さない」「出さない」は、ふつうに打つと満たされない。`lsof +D` は cwd がその下にある process を出し、recipe は repo(`/tmp/ralph-iso/work`)で `ralph` を打たせるので、読む人の shell と `lsof` 自身が出る。`cd` で出るのは 4(399 行)で、確かめより後である。手元の確かめ(scratch の下で `sleep` と `lsof +D`)では、shell(cwd がその下)、`sleep`、`lsof`、`head` が全部出た。`/tmp` の link 越しの path でも実 path でも同じだった。誤って「まだ残っている」と読む側に倒れるので消してしまう危険はないが、読む人が何を無視してよいか分からない | `docs/recipes/isolated-herdr-server.md:385-388`、`:398-400`。手元の `lsof +D` の出力(上の確かめ) | 2 の前に「`cd /` で出る」を置く。または「自分の shell と `lsof` 自身は数えない」と書く |
| LOW (F3) | maintainability | server の環境を見る確かめ `ps -E -o command= -p <pid>` は、端末に出すと ps の `-w` の指定がないので、窓の幅で切れる(ps(1): 端末への出力は幅で切れ、`ww` で切れなくなる)。環境の「渡した変数だけ」を見る確かめが、長い環境で最後まで見えない。repo の前例は `ps eww -o command= -p $pid \| tr ' ' '\n'`(`docs/evidence/org-watch-gate-live-2026-10-12.md:1223`)で、切れず、1 行 1 変数になる。この sandbox の shell では `ps -E` も `ps eww` も `ps -wwE` も環境を出さなかったので、`-E` が実機で動くかは確かめられていない | `docs/recipes/isolated-herdr-server.md:282-284`。前例の行 | 前例の形(`ps eww -o command= -p <pid> \| tr ' ' '\n'`)に合わせるか、`ps -wwE -o command= -p <pid>` にして、`/verify` か `/test` が sandbox の外で 1 度打つ |
| LOW (F4) | security | 専用の場所が、共有の `/tmp` の固定名 `/tmp/ralph-iso` で、`mkdir -p` は他のユーザーが先に作った directory も黙って受け入れる。その中のファイルは実行される(`run`、PATH の先頭の `bin/osascript`、claude の Bash tool が読む `claude-env`)。macOS の 1 人用の machine では実害はほぼないが、recipe は `/tmp` を「どの短い path でもよい」と案内していて、共有 machine での注意がない | `docs/recipes/isolated-herdr-server.md:20-21`、`:120`(`mkdir -p /tmp/ralph-iso/bin ...`)、`:184-187`(`claude-env`) | 最初に `mkdir -m 700 /tmp/ralph-iso` を打たせる(既にあれば失敗するので、前回の残りか他人の directory に気づく)。1 行で足りる。「Running again」の `session.json` の手順とも合う |
| LOW (F5) | maintainability | `herdr --help` について 2 つの文が食い違う読み方を許す。35〜37 行は「herdr 0.7.5 の `--help` は、渡された `HOME` の下の config と log の path を出す」と書き、147〜149 行は「herdr 0.7.5 は、使う config と log の path を出す」と書く。35 行の直前の文(2026-09-19 の記録)は、HOME だけ変えた server が HOME に従わず、ユーザーの `~/.config/herdr` を読んだと書く。`--help` が HOME に従い server が従わないなら、`--help` は server が開くファイルの予告にならない。「HOME の下を出す」ことの出典はどの evidence にもなく(`git log -S` で `Config:`・`Logs:` を足したのは e9884d9b だけ)、herdr を動かせなかったので私は確かめていない。147 行のすぐあとに「server が実際に開くファイルは `lsof` で確かめる」とあるため害は小さい | `docs/recipes/isolated-herdr-server.md:35-37`、`:147-151`。`docs/evidence/org-send-enter-timing-2026-09-19.md:40` | 出典のない 35〜37 行の後半(HOME の下を出す)を、実機で打った記録を足すか削る。147 行は「使う path を出す」を「設定した変数が効いているかの見当」に弱める |
| LOW (F6) | maintainability | recipe は、昇格した director に「`ralph` と `herdr` のコマンドを wrapper で打て」と言う。同じ PR で `/director` skill は、director がしてよいのは ralph と gh の操作だけ(既存の文)と書き、新しい禁止の項で `herdr pane send-keys` を名指しで使わせない。recipe の文は試行の director が herdr を打った逸脱(D-F3)をそのまま写している。skill と recipe で、director が herdr を打つかどうかの答えが違う | `docs/recipes/isolated-herdr-server.md:359-363`、`.claude/skills/director/SKILL.md:954-956`、`:962-964`(してはいけないこと)、`docs/evidence/org-trial-2026-10-20.md` の Trial deviations D-F3 | recipe の文を「`ralph` のコマンド」だけにし、herdr を打つのは人だと分ける(director の `ralph` が herdr を呼ぶので `HERDR_SOCKET_PATH` は要る、という理由は残せる) |
| LOW (F7) | naming | `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate` は、名前と doc コメントのほどには reviewer の雛形に従っていない。(1) 雛形を読まず、結びつきは定数 `reviewerReportPrefix` だけで、`GATE`・`HEAD`・`BASE` のヘッダ名はテストに直書きなので、雛形がヘッダを変えても緑のまま。(2) doc コメントは EVIDENCE が「`reviewerReportPrefix` で始まる」と言うが、実際の検査は行全体の完全一致(`EVIDENCE: ` + prefix + `<YYYY-MM-DD>-<org_id>-<seat_id>.md`)で、実名の例(`review-2026-10-23-org-a-reviewer.md`)にすると落ちる。AC3 の中身は満たしているので、名前とコメントの正確さの問題 | `internal/org/report_names_sync_test.go:250-251`、`:264`、`:291` | コメントを「例が雛形の形のプレースホルダをそのまま持つ」に直す。雛形のヘッダ名まで結ぶなら `promptFS` の `prompts/reviewer.md` から `GATE:`・`HEAD:`・`BASE:` の行を読む |
| LOW (F8) | readability | 新しい tech-debt の行「The on-demand watcher ... timed out」の Why deferred が「the docs wrap-up that wrote this row ... does not change code」と書くが、この PR は Go のテスト 1 つと leader の雛形の文(embed されて spawn に効く)を変える。意図は「`watcherInvokeTimeout` や watcher のコードは変えない」で、plan の Rollout の言い方(「文書とテストと leader の雛形の文だけ」)のほうが正確 | `docs/tech-debt/README.md:344`(Why deferred の列)、`docs/plans/active/2026-10-23-trial-docs-wrapup.md` の Rollout | 「does not change code」を「does not change the watcher or `watcherInvokeTimeout`」にする |
| INFO (I1) | readability | recipe の冒頭は何から分けるか(herdr server、agmsg、通知、台帳)を正しく列挙している。ただし seat は実 HOME、実の gh の login、`bypassPermissions`(autonomous)のまま動くので、「分離」を安全の囲いと読む人がいる。recipe はそれを言っていない | `docs/recipes/isolated-herdr-server.md:3-7`、`:38-41` | 「これは安全の囲いではない。seat はあなたの gh と git の権限で動く」の 1 文 |
| INFO (I2) | maintainability | `env -i` は `GH_TOKEN`・`GITHUB_TOKEN`・`SSH_AUTH_SOCK` も落とす。recipe は試行の所見 F1 として config の置き場所だけを扱い、start 前の確かめも `gh auth status` までで、push の経路(`git ls-remote origin` など)は確かめない。試行は専用の clone から push して PR を 2 本 merge したので、その環境では通った。token や agent だけで認証する利用者のことは未確認 | `docs/recipes/isolated-herdr-server.md:105-113`、`:309-317` | probe の 3 に `git ls-remote origin` を足すか、「token を env で渡している人は wrapper に足す」の 1 文 |
| INFO (I3) | typo | 「Running again」の 1 つ目は「`session.json` を delete せよ」と言い、括弧の中は `find ... -name session.json` で、見つけるだけである。消す手が書かれていない | `docs/recipes/isolated-herdr-server.md:423-425` | `find ... -name session.json -delete` にする |

## Positive notes

- recipe の事実の文(herdr の分離の 3 変数、HOME を動かさない理由、sun_path の 104 byte、試行の所見 F1 の git と gh の読み方、同 F2 の `path_helper` と `CLAUDE_ENV_FILE`、見張りが環境を引き継ぐ範囲、`ensureOrgWatch` が既存の見張りを比べずに使うこと、`--stop` の 30 秒、`ralph org report` を main のチェックアウトで打つ理由)を、記録とコードの両方で確かめた。食い違うものは見つからなかった。出典のない文は F5 の 1 つだけである。
- 片付けの順(org と director、見張り、server、ディレクトリ)が、見張りを止めて止まったことを確かめるまで `herdr server stop` と `rm -rf` に進ませない。`--stop` が 30 秒で終わらなかったときの分岐も書いてある(AC1 の片付けの条件を満たす)。
- 利用者固有の値は入っていない(`mise`、`/tmp/ot`、利用者の名前、`~` 以外の home の path、token、pane id、hex id はどれも 0 件)。試行の所見 F1 の `GIT_CONFIG_GLOBAL` と `GH_CONFIG_DIR` は「該当するときだけ残す行」として書かれ、`~/.gitconfig` に設定がある人が誤って git の設定を外す罠(`GIT_CONFIG_GLOBAL` が両方を置き換える)を先に潰している。
- recipe の末尾の「What this recipe does not cover」と冒頭の「通しでは打っていない」が、plan の「確かめていないこと」と根拠の表の記録と同じ内容で一致している。
- skill の folder trust の項と試行の記録(Setup の Trust、R1)、`blocked` の報告(`org-send-enter-timing-2026-09-19.md:34`)、`ralph org send` が idle か done を待つこと(`verbs.go:390-394`)が合う。`codex trust .` の文は `doctor.go:314`・`:398` と `docs/recipes/codex-setup.md:18` に合う。codex の trust の画面は「確かめていない」と書いている。
- RESULT の例のテストは、skill の節と囲みが動いたときに「囲みがない」で落ちる作りで(`t.Fatalf` に節の全文を出す)、`codeBlockAfterLine` 自体に 5 ケースの表のテストがある。`self-review-foo.md` に戻すと EVIDENCE の完全一致で落ち、`GATE:` を消すと `HeaderValues` で落ちる。
- 記録の書き換えは、ブランチ名を番号に直す所と、系列の最後の PR だけ番号を書かない所が、仕様・根拠の表・tech-debt の 3 つで同じ文になっている。watcher の時間切れの行は、負荷の読みがある時刻と読みがない時刻を分けて書き、「高い負荷のあいだに起きた」と言い切らない(plan の言い方より保守的で、数え直した結果と合う)。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし | - | F1〜F8 は、この PR の中で直せる文書とテストの文言である | - | - |

_(If any rows were added above, also append them to `docs/tech-debt/`.)_

## Recommendation

- Merge: yes。CRITICAL・HIGH・MEDIUM はない。LOW は 8 件(F1〜F8)で、どれも文言か 1 行の足しで直せる。直す場合の優先は F1(片付けの節の wrapper)と F2(lsof の確かめ)。
- Follow-ups:
  - `/verify` で、recipe を `scripts/check-sync.sh` の対象として確かめる(`cmp` は一致を確認済み)。F3 の `ps -E` は sandbox の外で 1 度打って環境が出るか見る。
  - F5 の `herdr --help` の `Config:`・`Logs:` の行は、herdr を使わずには確かめられなかった。`/test` か人が、`HOME` だけを変えた場合と `XDG_CONFIG_HOME` を変えた場合の 2 つで 1 度見る。
  - 直した場合は pipeline cycle 2 の `/self-review` から回す(記録だけの修正の例外は、recipe・skill・テストの直しには使えない)。
  - insight event は、次の 1 行で `docs/insights/events/2026-10-23-trial-docs-wrapup.jsonl` に追記する:
    `./scripts/insights-append.sh --slug trial-docs-wrapup --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 8 --cycle auto --source skill`

## Re-review (5ac0af89)

`git diff 58ca93cc 5ac0af89`(recipe とその templates の写し、`internal/org/report_names_sync_test.go` のコメント、tech-debt の 1 行)を読んだ。recipe の 2 つの写しは `cmp` で一致した。herdr と recipe のコマンドは打っていない。`check-sync` と 3 つのテストの合格はリードの報告で、私は確かめていない。

### cycle 1 の指摘の扱い

| 指摘 | 結果 | 確かめたこと |
| --- | --- | --- |
| F1 | 直った | 「Clean up」の冒頭が、この節のコマンドはすべて wrapper を通すと言い、素の `ralph org disband` が自分の server の herdr を呼ぶ理由を書く。1・2 のコマンドはすべて `/tmp/ralph-iso/run` 付きになった。節の中に素の `ralph`・`herdr` のコマンドは、下の N1 の 1 か所(427 行の説明文)を除いて残っていない |
| F2 | 直った | 2 で `ralph status` を repo から打ってから `cd ~` し、`pgrep` と `lsof +D` をそこから打つ。3 の `lsof +D` も同じ場所で打つ。4 は「外から写して消す」になり、確かめより前に出る順になった |
| F3 | 直った(実機は未確認) | repo の前例と同じ `ps eww -o command= -p ... \| tr ' ' '\n'` になり、`ww` の理由も書いた。この sandbox では環境が出ないので、出力は `/verify` が sandbox の外で見る |
| F4 | 直った | 先頭に `mkdir -m 700 /tmp/ralph-iso`(`-p` なし)を置き、既にあれば使い回さないと書いた。後の `mkdir -p` は subdirectory だけを作る。「Running again」(自分で残した directory の再利用)とは、時点が違うので食い違わない |
| F5 | 直った | 出典のない「HOME の下を出す」の文を消し、`--help` の段落は、`XDG_CONFIG_HOME`・`XDG_STATE_HOME`・`HERDR_SOCKET_PATH` を scratch に向けた herdr 0.7.5 の出力(`Config:` と `Logs:`)と「何も作らない」を書く形になった。出典は下の N3 |
| F6 | 直った | 昇格した director は `ralph` と `gh` を wrapper で打ち、recipe の `herdr` のコマンドは人のものだと書き、`/director` skill(`.claude/skills/director/SKILL.md:954`、`:962`)を指す。recipe の中で director に herdr を打たせる文は残っていない(`grep -n director` で確かめた) |
| F7 | 直った(名前は変えず、コメントで限界を言う) | doc コメントが、EVIDENCE は placeholder の形の完全一致で、実名の例ならテストを変える必要があり、雛形は読まず `reviewerReportPrefix` を通して `TestReportNames_DocsMatchConstants` に結ぶと言う。この結びは `report_names_sync_test.go:105` の既存の検査と合う |
| F8 | 直った | 「does not change the watcher or `watcherInvokeTimeout`」になり、PR が Go のテストと雛形の文を変えることと矛盾しない。row の他のセルは変わっていない(字単位で比べた) |
| I1 | 直った | 「Why isolate」の終わりに「sandbox ではない」の段落。`autonomous` で claude の bypass mode という説明は、`/org` skill の前提と合う |
| I2 | 直った(ただし N2) | `SSH_AUTH_SOCK` と `GH_TOKEN` を落とすことと、start 前の確かめが push を試さないことを書いた |
| I3 | 直った | 「`find` が出した `session.json` を消す」に直った |

### 新しい指摘

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (N1) | readability | 「Clean up」2 の 30 秒の分岐の段落が、`cd ~` のあとに「`ralph status` が `org watch: not running` と言うまで server を止めない」と書く。コマンドは素の形で、どこで打つかも書かない。冒頭の「すべて wrapper を通す」と食い違い、読む人は `~` のまま打つ。`ralph status` は repo の外では別の台帳を見るので、動いている見張りに対して `not running` と出て、server の停止と `rm -rf` に進みうる(見張りが残るだけで壊れるものはないが、この段落は、止まるまで消さないための歯止めである) | `docs/recipes/isolated-herdr-server.md:423-427`(`cd ~` は 413 行)。`internal/org/statedir.go:52-54`(git の作業木の外では cwd の下の `.harness/state/org` が台帳になる) | 段落の `ralph status` を `/tmp/ralph-iso/run ralph status`(repo から)にする。または `cd ~` を段落のあとに移す |
| LOW (N2) | security | I2 の直しは、必要なら `GH_TOKEN` を wrapper に足せと言い、F3 の直しは `ps eww ... \| tr ' ' '\n'` で server の環境をそのまま端末に出させる。足した人の `GH_TOKEN=<値>` が端末とその履歴(Claude Code などのセッションなら記録)に出る。同じ recipe が末尾の近くで、token を evidence に写すなと言う | `docs/recipes/isolated-herdr-server.md:158-161`(I2)、`:303-306`(F3)、`:451-452`(token を写さない) | 確かめたいのは変数の名前だけなので、`... \| tr ' ' '\n' \| cut -d= -f1` にして名前だけを出させる |
| INFO (N3) | maintainability | 新しい `--help` の段落は、herdr 0.7.5 が実際に打った出力(`Config: <S>/xc/herdr/config.toml`、`Logs: <S>/xc/herdr/herdr.log (plus ...)`)と「何も作らなかった」を、直接の観測として書く。観測は 2026-10-23 のオーケストレーターの 1 回の実行で、recipe の冒頭が挙げる記録のどれにもなく、`docs/evidence/` にもない。私は herdr を動かせないので確かめていない | `docs/recipes/isolated-herdr-server.md:163-173`、冒頭の出典の段落(9〜18 行) | `/verify` の report に出力を写すか、冒頭の出典に「`--help` の出力は 2026-10-23 の実行」を足す |

### 判定

cycle 1 の 8 件の LOW と 3 件の INFO は、すべて直った。直しが足した新しい LOW は 2 件(N1、N2)、INFO は 1 件(N3)で、どれも recipe の文言を 1 行直すだけで済む。CRITICAL・HIGH・MEDIUM はない。Merge の行(yes)は変えない。cycle 1 の insight event の件数(low 8)も変えず、N1・N2 はこの節にだけ書く。pipeline のこの cycle で直すか、`/pr` で既知の穴として残すかはリードが決める。直す場合は、N1・N2 とも recipe の本文と templates の写しの 2 か所を同じに直す(`cmp` を保つ)。
