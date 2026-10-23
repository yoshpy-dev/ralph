# Verify report: trial-docs-wrapup

- Date: 2026-10-23
- Plan: `docs/plans/active/2026-10-23-trial-docs-wrapup.md`(パイプライン cycle 1/2。`./scripts/plan-visual.sh digest` は承認の digest と同じ `932bafc59d5c` を返した)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 3c1cd14b...HEAD`(HEAD edf04ef7、19 ファイル、+1562/-83)。S1 の e9884d9b・493e4ce8・e07bf30b、S2 の be8acda8、S3 の 63e629a3・f310e994、self-review の直しの 5ac0af89・f15b696c、self-review の report と insight event を含む。テストは実行していない(`/test` の範囲)
- Evidence: `docs/evidence/verify-2026-10-23-055828.log`(`run-static-verify.sh` が自分で書いたログ。git が無視する場所)。依頼どおりコミットするのはこの report と insight event だけなので、`verify-<date>-<slug>.log` は作っていない

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 recipe | 満たす(文書と記録とコードで確認。recipe の手順は打っていない) | `cmp docs/recipes/isolated-herdr-server.md templates/base/docs/recipes/isolated-herdr-server.md` は rc 0(483 行)。Scope の項目はすべてある: 分離する理由と 3 つの変数、`HOME` を動かさない理由(keychain)、`sun_path` の 104 バイト(`Why isolate`)、`env -i` の wrapper と変数の表(`The environment`)、F1(`GIT_CONFIG_GLOBAL`・`GH_CONFIG_DIR`)、F2(`path_helper`、`BASH_ENV` は効かない、`CLAUDE_ENV_FILE`)、`SHELL` と doctor の「Shell aliases (codex/claude)」(`internal/cli/doctor_shell_alias.go:818` の名前と同じ)、agmsg の写しと `[org].agmsg_home`、`osascript` の shim、台帳を分ける repo と `ensureOrgWatch` の振る舞い、最初の start の前の `org watch: not running` の確かめ、wrapper で打つこと、`lsof` と捨ての pane の確かめ、folder trust、片付け。片付けの 2(見張りを止め、repo から `ralph status` で `not running` を確かめる)は 3 の `herdr server stop` と 4 の `rm -rf` より前にあり、`--stop` が 30 秒で終わらなかったときは止まるまで消さない分岐がある(`:425-432`)。`mise`、`/tmp/ot`、`/Users/`、利用者の名前、`/opt/homebrew`、`.local/share`、`ghp_` は 0 件。`/org` skill の前提の `.claude/skills/org/SKILL.md:65` が recipe を指す |
| AC2 folder trust | 満たす(文書で確認) | `/org` skill の前提 `:49` の「フォルダ信頼(repo ごとに 1 回)」は、人が答えること、main のチェックアウトで一度、`.claude/worktrees/` の下は聞かれないこと(試行、Claude Code 2.1.291)、repo の外の `--cwd`、codex の画面は確かめていないことと `codex trust .`(`docs/recipes/codex-setup.md`)、`blocked` と `ralph org send` が届かないこと、`ralph org read --org-id <id> --seat <seat>` で読んで人が答えることを書く。bypass の項目(`:43`)の主語は「人が」。`/director` skill の「してはいけないこと」(`.claude/skills/director/SKILL.md:962`)に、trust と bypass の画面にキーを送らない項がある。4 つの写しは `cmp` で org も director もバイト一致、`./scripts/check-skill-sync.sh` は rc 0(14 skill) |
| AC3 RESULT の例 | 満たす(文書とテストのコードで確認。テストの実行と mutant は `/test`) | 例(`.claude/skills/org/SKILL.md:1331` の次の囲み)は `GATE: pass`・`HEAD:`・`BASE:` のヘッダを持ち、EVIDENCE は `docs/reports/review-<YYYY-MM-DD>-<org_id>-<seat_id>.md`(`:1342`)。reviewer の雛形の RESULT の例(`internal/org/prompts/reviewer.md:59-71`)とレポート契約(`:114-122`)のヘッダとファイル名の形に合う。`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`(`internal/org/report_names_sync_test.go:269`)は、`## typed protocol` の節から「RESULT の例」の行のあとの最初の囲みを読み、`protocol.Parse` と `protocol.Validate`、`GATE` が `pass` 1 つ、`HEAD` と `BASE` が空でない 1 つ、EVIDENCE の行の完全一致を見る。囲みが見つからなければ `t.Fatalf` で落ちる。`codeBlockAfterLine` には 5 ケースのテスト(`:220`)がある。go1.26 と go1.25.8 の `go vet ./internal/org/...` は rc 0 |
| AC4 所要時間 | 満たす(grep で確認) | AC4 の grep(skill の 8 写しと `internal/org/prompts`)は 0 件(base の 3c1cd14b では 13 件: org の写し 4 つに 2 件ずつ、director の写し 4 つに 1 件ずつ、`leader.md` に 1 件)。`docs/evidence`・`docs/reports`・`docs/plans` を除く repo 全体でも 0 件(base では `prompts_test.go` の 1 件を足して 14 件)。新しい文「repo のテストの量とマシンの負荷によっては数十分かかる」は `/org` skill の `gate` の行(`:220`)と regate(`:1165`)、`/director` skill(`:908`)、`leader.md:230` にあり、`prompts_test.go:830` の固定の文も同じ。試行の秒数は skill と雛形に入っていない |
| AC5 記録 | 満たす(記録、試行の snapshot、gh で確認) | `grep -rn 'fix/busy-leader' docs/specs docs/evidence docs/tech-debt` は 0 件(base は仕様 3、根拠の表 6、tech-debt 2 で、AC5 の数と同じ)。仕様の Status の一覧は PR の番号が重複なしで 36 個(9 段の 24 本、#205、#241・#242、所見の直し 9 本)。`gh pr view 253 --json state,mergedAt,mergeCommit`(読むだけ)は `MERGED`、mergedAt `2026-10-23T03:45:45Z`、merge commit 3c1cd14b で、根拠の表の「段ごとの PR」の文と一致する。根拠の表の F1・F2・folder trust の行(`:20`、`:21`、`:35`)は `docs/trial-docs-wrapup` で直したと書き、tech-debt の 2 行(`docs/tech-debt/README.md:318`、`:340`)は取り消し線と直前の `<!-- RESOLVED ... -->` の注を持ち、セルは 5 つ。watcher の時間切れは所見の表(`:38`)と tech-debt(`:344`)にある。仕様の Status の節と根拠の表の所見の節の前置き(「予定の順」)は、系列が `docs/trial-docs-wrapup` で終わったことと、開いたまま残る 3 行を書く。印は動いていない: 仕様のチェックボックスの行は base も HEAD も `[x]` 29・`[ ]` 3、根拠の表で書き換えた FR と NFR の行は注か根拠の列だけが変わり、判定の列は変わっていない。所見の行の「食い違う」の列の「いいえ」も変わっていない |
| AC6 run-verify と go1.25.8 | 一部だけ確認 | `docs/evidence/verify-2026-10-23-050654.log` は Mode all、scope full で「All verifiers passed」。Timestamp は 05:06:54Z(14:06:54 JST)で、f310e994 のコミット(14:06:45 JST)の直後に始まり、秘密の走査の行は `3c1cd14b..f310e994` を挙げる。そのあとの 5ac0af89 と f15b696c が変えた Go は `report_names_sync_test.go` のコメントだけ(`git diff f310e994 HEAD -- '*.go'` の変更行はすべて `//` の行)で、ほかは recipe、tech-debt、report、insight event である。HEAD では静的解析(下)を回した。`go test` の実行(go1.26 と go1.25.8)は `/test` の範囲 |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed、HEAD edf04ef7) | rc 0、約 29 秒 | shellcheck、hook の `sh -n`、guard の awk の構文、`jq -e`(settings.json 2 つ)、Codex の hook の 3 検査、`check-sync.sh`(IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、KNOWN_DIFF 5、TEMPLATE_ONLY 11)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の計画の参照、Go の pack(gofmt ok、golangci-lint 0 issues)、`secret-scan-branch.sh` は `3c1cd14b..edf04ef7` で clean |
| `./scripts/check-skill-sync.sh` | rc 0 | 依頼どおり単独でも打った。14 skill が一致 |
| `./scripts/check-sync.sh` | rc 0 | 単独でも打った。IDENTICAL 172、DRIFTED 0。`docs/recipes/` は走査の対象で(`adding-a-language-pack.md` が KNOWN_DIFF に出る)、新しい recipe は DRIFTED にも ROOT_ONLY にも出ない |
| `cmp`(recipe 2 つ、skill の org と director の 4 写しずつ) | どれも rc 0 | `.claude` と `.agents`、root と `templates/base` のすべての組 |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/...`、`GOTOOLCHAIN=go1.25.8 go build ./...`、`go vet ./internal/org/...`(go1.26.0) | どれも rc 0 | go.mod は `go 1.25.8`。新しいテストが使う `slices.IndexFunc` は go1.21 から |
| `gofmt -l`(変わった 2 つの Go のファイル)、`git diff 3c1cd14b...HEAD --check` | 出力なし | 作業ツリーに残ったのは、git が無視する verify のログ 2 つだけ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| recipe と試行の記録(`docs/evidence/org-trial-2026-10-20.md` の Setup、Findings の F1・F2、Trial deviations の R1、Cleanup) | はい | 変数の組(`HOME`、`USER`、`LOGNAME`、`SHELL=/bin/bash`、`LANG`、`TERM`、`TMPDIR`、`PATH`、XDG の 2 つ、`HERDR_SOCKET_PATH`、F1 の 2 つ、`GOFLAGS=-timeout=30m`、`CLAUDE_ENV_FILE`)、F2 の probe a〜c、trust の画面の向き(「No, exit」が選ばれ、Down と Enter)、`.claude/worktrees/` の下と 2 つの機能の worktree で画面が出なかったこと、`~/.claude.json` の `/private/tmp/...` の記録、片付けで外に残る 3 種が記録と合う。試行の片付けの順(disband、director の detach、watch の stop と `ralph status` の確かめ、server の stop、`rm -rf`)を recipe も保つ |
| recipe と直近の実機(`docs/evidence/busy-leader-2026-10-23.md` の「分離のしかた」) | はい | scratch の repo の作り方(main の base のコミット、repo の設定の名前とメール、空のディレクトリへの `core.hooksPath`、`.gitignore` の 2 行)、agmsg の写しの 5 項目と写さない 4 項目、`[org].agmsg_home`、shim、`SHELL=/bin/bash` の理由、trust の答え方が合う。版(herdr 0.7.5、Claude Code 2.1.291、agmsg 1.5.1)も合う |
| recipe の herdr の事実と古い実機の記録 | はい | `HOME` と `HERDR_SOCKET_PATH` だけを変えた server がユーザーの `session.json` を読んだこと(`docs/evidence/org-send-enter-timing-2026-09-19.md:40`)、150 字を超える socket の path で herdr が起動しなかったこと(`docs/evidence/org-plan-approval-live-2026-10-13.md:17`)、`session.json` が残ると番号が続くこと(`docs/evidence/herdr-pane-close-2026-10-07.md:46`) |
| recipe が頼る ralph のコード | はい | `orgWatchChildDroppedEnv` の 4 変数(`internal/cli/org_watch.go:378`)、`--stop` の 30 秒と時間切れの文(`:340`、`:766`)、`ralph status` の `org watch: not running (stopped at ...)`(`internal/cli/status.go:324`)、`RALPH_ORG_AGMSG_HOME`(`internal/org/driver/agmsg.go:144`)、`osascript` の PATH 引き(`internal/org/watch.go:415`)、座席の pane の 2 つの `--env`(`internal/org/driver/paneenv.go:25`) |
| recipe の `herdr --help` の段落(`:163-172`、self-review の F5 と N3) | はい | 下の F5 の probe の出力が、recipe が書く 2 行と「何も作らない」に一致した |
| recipe の `ps eww` の確かめ(`:303-308`、self-review の F3) | はい(INFO 1 件) | 下の F3 の probe。非 Apple の binary では変数の名前が出る。Findings の INFO-1 |
| `/org` と `/director` の skill の folder trust の文と試行の Setup の Trust | はい | 画面の初期の向き、子のディレクトリを聞き直さないこと、機能の worktree で画面が出なかったこと、版が試行と合う。試行では director がキーを送った(R1 が許した準備の操作)が、skill はこれを禁じて人に答えさせる。根拠の表の folder trust の行と tech-debt の RESOLVED の注も同じことを書く。`blocked` の報告は `docs/evidence/org-send-enter-timing-2026-09-19.md:34`、`ralph org read` の `--seat` は `internal/cli/org.go:1418`、`codex trust .` を促すだけの doctor は `internal/cli/doctor.go:398` に合う |
| `/org` skill の RESULT の例と reviewer の雛形 | はい(INFO 1 件) | ヘッダの 3 行と EVIDENCE のファイル名の形が合う。本文の件数の書き方は Findings の INFO-2 |
| 仕様、根拠の表、tech-debt と #253 の状態 | はい | gh の出力(上の AC5)。根拠の表の #253 の 4 つの report と計画のファイルはどれもある |
| 根拠の表と tech-debt の watcher の時間切れと試行の snapshot | はい | `docs/evidence/org-trial-2026-10-20/watch.log` の「watcher error」は 9・22・27・29・34・41 行目の 6 件で、seat 別(doctor-escape の implementer 3、reviewer 1、leader 1、report-escape の leader 1)と条件別(`scope_change` 4、`stall` 2)が合う。「watcher already in flight」は 11 行。`model-receipts.jsonl` の watchdog の receipt は 24 件(normal 16、circular 1、fake_progress 1、`watcher_error` 6)で、`watcher_error` の時刻 6 つも合う。`watcherInvokeTimeout` は 60 秒の package 変数(`internal/org/watcher.go:74`)、失敗で stderr に 1 行書いて戻るのは `internal/cli/org_watch.go:853-856` で、記録の説明と合う。試行の Gates の秒数(28、1947、1375、1664、1990、792)と load average(12 CPU、20〜29、最大 51、45〜75、7〜10)も根拠の表の文と合う |
| tech-debt の行の指し方 | はい | 根拠の表と仕様が挙げる 5 つの行の書き出し(取り消し線の 2 つを含む)は、どれも tech-debt に 1 件だけある |
| 計画の進捗 | INFO | Findings の INFO-4 |

## Observational checks

依頼の 2 つの probe を打った。どちらもユーザーの herdr server に触れていない。

F3(`ps eww -o command= -p $$ | tr ' ' '\n' | cut -d= -f1 | grep -c '^HOME$'`):

- 出力は `0`。この Bash の shell(`$$`)は `/bin/zsh` である
- 0 の理由を確かめるため、値を出さない形で 3 つ足した。(1) `env -i HOME=/fake-home PATH=/usr/bin:/bin XDG_CONFIG_HOME=/fake/xc MARKER=1 sleep 30` を裏で起こして同じ形で読むと、出たのは `sleep` と `30` だけで、変数の名前は 1 つも出なかった。sandbox を外して打っても、`ps -wwE` でも同じだった。(2) 同じ `env -i` で `/usr/bin` の外にある python3 を起こすと、コマンドの語のあとに `HOME`、`PATH`、`MARKER` が 1 行ずつ出た。sandbox の中でも同じ数だった。(3) 途中で `HOME` が 1 と出た回は、打ったコマンドの文字列にあった `HOME` の語を数えたもので、環境ではなかった
- macOS 26.5.2 の ps は、Apple の binary(`/bin/zsh`、`/bin/sleep`)の環境を見せず、ほかの binary の環境は見せた。self-review の sandbox で何も出なかったのも、おそらく同じ理由である(未確認)。herdr は Apple の binary ではないので、recipe の確かめは server の変数の名前を出すと見込まれる。herdr の server そのものでは確かめていない。repo の前例(`docs/evidence/org-watch-gate-live-2026-10-12.md:1337`)も、ralph の子(Go の binary)に同じ形を打って変数を読んでいる

F5(`<S>` は scratch の下の新しいディレクトリ、`$HOME` は `~`):

```
$ env -i HOME=~ PATH="<herdr のディレクトリ>:/usr/bin:/bin" XDG_CONFIG_HOME="<S>/xc" XDG_STATE_HOME="<S>/xs" HERDR_SOCKET_PATH="<S>/h.sock" herdr --help | grep -E '^(Config|Logs):'
Config: <S>/xc/herdr/config.toml
Logs:   <S>/xc/herdr/herdr.log (plus herdr-client.log, herdr-server.log)
$ ls -A "<S>"
(出力なし。打つ前も 0 件)
```

recipe の `:167-171` の 2 行と同じで、`<S>` の下には何もできなかった。

依頼の外で、herdr を 1 回だけ打った: F5 のあとに、版を見るため `herdr --version` を素の環境で打ち、`herdr 0.7.5` が出た。server には触れない読み取りだが、依頼が許した 2 つの probe の外である。打ったあと、`~/.config/herdr` と `~/.local/state/herdr` の下に 10 分以内に変わったファイルがないこと(`find -mmin -10` が 0 件)を確かめた。

## Findings

- INFO-1(recipe `:303-308`): `ps eww` の確かめは、ps が環境を見せない process では、コマンドの語だけを出して何も警告しない。上の F3 のとおり、macOS 26.5.2 は Apple の binary の環境を見せない。herdr はそれに当たらないので今の recipe で困ることはおそらくないが、変数の名前が 1 つも出ないときに「渡した変数だけ」と読み違える余地がある。直すなら「変数の名前が 1 つも出ないなら ps がその process の環境を読めていない」の 1 文を足す
- INFO-2(`/org` skill `:1331-1343`): reviewer の雛形のレポート契約は、RESULT に「レポートパスと severity 別件数」を付けるよう言う(`internal/org/prompts/reviewer.md:119-120`)。例の本文は「CRITICAL なし。」で、件数の形ではない。計画と AC3 が求めたヘッダと EVIDENCE は合っており、テストも本文の件数を見ない。直すなら SUMMARY を「CRITICAL 0、HIGH 0、MEDIUM 0、LOW 2」のような形にする
- INFO-3(`/org` skill `:49` の folder trust の項、recipe `:349`): 「repo の外のディレクトリを `--cwd` に渡す座席には、そのディレクトリでも同じことをする」は、試行で見た規則(信頼したディレクトリの下は聞き直さない)からの推論で、repo の外の `--cwd` の座席は試行でも実機でも動かしていない。文としては正しいと見込まれるが、根拠の表の「確かめていないこと」には書かれていない
- INFO-4(計画の Progress checklist): self-review の report はあるが「Review artifact created」は `[ ]` のまま(AC6 も `[ ]`。これは `/test` の前なので正しい)。`/pr` までに直せばよい

LOW 以上はない。self-review の N4(templates の写しの recipe が挙げる `docs/evidence/` のファイルは scaffold に届かない)は、既存の template の recipe 3 つと同じ扱いで、`check-template-purity.sh` も通るので、ここでは数えない。

## Coverage gaps

- recipe の手順は打っていない(計画の Non-goals)。上の 2 つの probe は `herdr --help` と ps の読み方だけを見た。server、pane、trust の画面、片付けの順は記録とコードで確かめた
- `ps eww` の確かめを herdr の server そのものに打っていない(INFO-1)
- テストの実行(`TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`、`TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead`、`TestRenderRolePrompt_Leader_FeatureOrgGate`、`internal/org/director_skill_sync_test.go` の節の構成の固定)と、`self-review-foo.md` と `GATE:` を消した mutant は `/test` の範囲
- HEAD での `./scripts/run-verify.sh`(Mode all)は打っていない。f310e994 での run-verify が rc 0 で、そのあとの Go の変更はコメントだけである
- codex の trust の画面、codex の座席の `PATH`、Linux、login の zsh の pane は、計画と recipe のとおり確かめていない

## Verdict

- Verdict: pass
- Verified: AC1〜AC5 を、recipe と skill の写し(`cmp`)、試行と実機の記録、試行の snapshot、ralph のコード、`gh pr view 253`(読むだけ)で確かめた。静的解析(`run-static-verify.sh`、`check-skill-sync.sh`、`check-sync.sh`、go1.25.8 と go1.26 の go vet、go1.25.8 の build、gofmt、`git diff --check`)はすべて rc 0。仕様の印(29 / 3)と根拠の表の判定の列は動いていない。F5 の probe は recipe の `herdr --help` の段落と一致した
- Partially verified: AC6 は、f310e994 での run-verify(Mode all、scope full)が rc 0 であることと、HEAD での静的解析まで。F3 の recipe の形は、非 Apple の binary で変数の名前が出ることまで確かめ、herdr の server では打っていない
- Not verified: テストの実行(go1.26 と go1.25.8)と mutant、recipe の通しの実行
- Findings: INFO 4 件。LOW 以上はない。依頼の外で `herdr --version` を 1 回打った(Observational checks)

## Re-verify (eed46c2c)

`git show eed46c2c`(c6d98b0a の上、8 ファイル、+28/-15: recipe と templates の写し、`/org` skill の 4 写し、根拠の表の 1 行、計画の Progress)を読んだ。herdr もテストも打っていない。上の節の行番号は c6d98b0a のもので、`/org` skill は `:57` より後が 1 行、recipe は `:308` より後が 2 行、`:351` より後が 3 行ずれた。

| 指摘 | 結果 | 確かめたこと |
| --- | --- | --- |
| INFO-1 | 直った | recipe の `ps eww` の項(`:303-310`)に、コマンドのあとに名前が 1 つも出なければ ps が環境を読めておらず確かめにならないこと、すぐ上の `lsof` の確かめは使えることが足された。上の F3 の probe(Apple の binary では名前が出ない)と合う |
| INFO-2 | 直った | `/org` skill の RESULT の例の SUMMARY(`:1342`)が「CRITICAL 0、HIGH 0、MEDIUM 1、LOW 2。」になり、reviewer の雛形の「severity 別件数」(`internal/org/prompts/reviewer.md:119-120`)の形になった。ヘッダ 3 行と EVIDENCE の行は変わっていないので、AC3 のテストが見る所は動いていない(テストの実行は `/test`)。「CRITICAL なし」は `docs/reports`・`docs/evidence`・`docs/plans/archive` の外に残っていない |
| INFO-3 | 直った | `/org` skill の folder trust の項(`:56-58`)、recipe(`:350-352`)、根拠の表の「確かめていないこと」の「文書のまとめ」の項(`docs/evidence/org-spec-evidence-2026-10-21.md:122`)の 3 か所が、repo の外の `--cwd` の信頼を推論で実機では確かめていないと書く。3 つの言い方は同じ中身である |
| INFO-4 | 直った | 計画の Progress の「Review artifact created」と「Verification artifact created」が `[x]` になり、self-review と verify の結果の行が足された。Acceptance criteria の AC6 は `[ ]` のままで、`/test` の前なので正しい。`./scripts/plan-visual.sh digest` は `932bafc59d5c` のままで、承認の digest と同じ。Progress の「`GOTOOLCHAIN=go1.25.8 go test ./internal/org/...` は ok」はリードの報告で、私は確かめていない |

AC2 と AC3 をもう一度見た。AC2: folder trust の項の中身(人が答える、main のチェックアウトで一度、`.claude/worktrees/` の下は聞かれない、codex は未確認と `codex trust .`、`blocked` と `ralph org read`)は残り、推論の注が足されただけである。bypass の項と `/director` skill の禁止の項は変わっていない。AC3: 例のヘッダと EVIDENCE は c6d98b0a のときと同じである。

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(scope changed、HEAD eed46c2c) | rc 0 | `check-sync.sh` は IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0。`check-skill-sync.sh` は 14 skill が一致。`check-template-purity.sh` は PASS。Go の pack は gofmt ok、golangci-lint 0 issues。`secret-scan-branch.sh` は `3c1cd14b..eed46c2c` で clean。ログは `docs/evidence/verify-2026-10-23-061554.log`(git が無視する場所) |
| `cmp`(recipe 2 つ、`/org` と `/director` の 4 写しずつ) | どれも rc 0 | root と `templates/base`、`.claude` と `.agents` のすべての組 |
| `git diff --check c6d98b0a eed46c2c` | 出力なし | U+FFFD も 0 件 |

新しい指摘はない。上の Verdict(pass)は変えない。
