# Verify report: org-watch-gate

- Date: 2026-10-12(cycle 3)
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み。`scripts/plan-visual.sh digest` の値 de5ca6b7e6f0 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 3 回目(cycle 3)。上限は `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げてあり、これが最後の cycle である。`cycle-count.json` は cross-review の規則で 2 のままなので、cycle 3 の番号はここと insight event の `--cycle 3` で付けた。cross-review cycle 2 の ACTION_REQUIRED 1 件を 8d8ddbd2 で直したあとの、全段のやり直し。cycle 2 と cycle 1 の結果は末尾の「付録」に残してある
- Scope: 仕様適合、静的解析、文書の整合。ブランチ全体は `git diff dc41c4f5...HEAD`(HEAD 522776ac)。主に見たのは cycle 2 の verify(755e082d、HEAD 5739e1ba を見た)のあとのコードの変更 3 つで、`git diff 5739e1ba..HEAD -- internal/` は 4 ファイル、+234/-8。47635690 はテスト 1 本(`orgWatchChildExitWait` が `org.DefaultWatchLockWait` より長いこと)、2fada512 は `runOrgWatchStop` の doc の 2 行、8d8ddbd2 は相対の agmsg home を絶対パスにして子に渡す直し(`internal/` で +223/-6、テスト 13 ケース)。8d8ddbd2 は全文を読んだ。テストは走らせていない(/test の担当)。HEAD のビルドで、相対の agmsg home を持つ `--detach` の子の環境と作業ディレクトリを実機で確かめた
- Evidence: `docs/evidence/verify-2026-10-12-org-watch-gate.log`(gitignore の対象で、コミットしない。cycle 3 の出力は末尾に足した)

## Latest run: cycle 3(HEAD 522776ac)

### Spec compliance

cycle 2 の判定(付録)から、3 つのコミットで変わった AC を見直した。3 つが触れた非テストのコードは `internal/cli/org_watch.go`(`orgWatchChild` の新しい引数 `cfgAgmsgHome`、新しい `orgWatchChildAgmsgHome` と定数 `orgAgmsgHomeEnv`、`runOrgWatchStop` の doc)と `internal/cli/org.go`(start が `orgWatchChild` に `rt.Config.AgmsgHome` を渡す 1 行)だけで、`internal/org/` は変わっていない。検査の位置、TASK の判定、印、status、起動と待ち、雛形のコードは変わっていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC6 子の作業ディレクトリ、`--config`、環境 | 満たす(子の環境の手当てが 1 つ増えた) | `orgWatchChild`(`internal/cli/org_watch.go:325`)の作業ディレクトリ(`org.LedgerMainWorktreeRoot`、`:331-333`)、`--config` の絶対パス(`orgWatchChildArgs`、`:282-293`)、HERDR_* の 4 つを外すことと `PWD`(`:344`、`:349-351`)は cycle 2 と同じ。足されたのは、作業ディレクトリを変えるとき(`:335`)だけ `orgWatchChildAgmsgHome`(`:369`)を呼び、値が返れば継承した `RALPH_ORG_AGMSG_HOME` を落として(`:344`)絶対パスの 1 件を足すこと(`:352-354`)である。AC6 の文(main worktree のルートで動く、`--config` は絶対パス、HERDR_* の 4 つがない)はそのまま成り立つ。AC6 のテスト 3 本(`TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`、`internal/cli/org_watch_test.go:998`、`TestOrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`、`internal/cli/org_start_watch_test.go:643`、`TestOrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv`、`org_watch_test.go:1177`)と helper の `checkOrgWatchChild`(`:1048`)は 1 行も変わっていない。新しいテストは `TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild`(`:1082`、11 ケース)と `TestOrgStart_ARelativeAgmsgHomeIsPassedAbsoluteToTheWatch`(`org_start_watch_test.go:689`、2 ケース)。実機でも、子は main worktree のルートで動き、`PWD` はそのルートの 1 件、親に置いた `HERDR_PANE_ID` は子になく、`RALPH_ORG_AGMSG_HOME` は絶対パスの 1 件だった(下の「Observational checks」) |
| AC5 子が先に終わっても、ほかの見張りが running なら成功 | 満たす(変わらない。本番の待ちの長さが固定された) | `launchAndAwaitOrgWatch` のコードは cycle 3 で変わっていない。47635690 の `TestOrgWatchChildExitWait_OutlastsTheChildsLockWait`(`org_watch_test.go:647`)が、本番の `orgWatchChildExitWait` が `org.DefaultWatchLockWait` より長いことを確かめる。/test cycle 2 の生き残り K8(本番の値を 0 にする)を閉じるテストである |
| AC10 既存のテストを書き換えない | 満たす | `git diff 5739e1ba..HEAD -- 'internal/**/*_test.go'` で消えた行は 0。非テストのコードで消えた行は、`orgWatchChild` の署名・doc の 1 行・`make` の容量・除外の条件、その 2 つの呼び出し、`runOrgWatchStop` の doc の 2 行だけ |
| AC12 文書の整合と同期の検査 | 一部(D11〜D14 を /sync-docs で入れる) | cycle 2 の D1〜D10 は 2fada512 で直った(下の「Documentation drift」で 10 件とも確かめた)。`check-sync.sh` と `check-skill-sync.sh` は rc 0 で、skill の 4 面は同じ sha256(09fb4c13…)。8d8ddbd2 は文書に触れていないので、相対の agmsg home の扱いを書いた文書がない(D11〜D13)。self-review C3-2 の tech-debt の行もまだない(D14) |
| AC1〜AC4、AC7〜AC9、AC11、AC13〜AC15 | 満たす(変わらない) | 関わるコードが cycle 3 で変わっていない。AC3 と AC4 の start の起動は、`orgWatchChild` の呼び出し(`internal/cli/org.go:300`)に引数が 1 つ増えただけで、流れ(`ensureOrgWatch`、`launchAndAwaitOrgWatch`)は同じ。AC8 の残り(座席の `wait` と `inbox show/ack/resolve/notify` は grep だけ)は cycle 1 と同じ |

#### 8d8ddbd2 の確認

- 優先順位: `driver.ResolveAgmsgHome`(`internal/org/driver/agmsg.go:123-132`)は、env `RALPH_ORG_AGMSG_HOME` が空でなければそれ、空なら設定の値、それも空なら既定の `~/.agents/skills/agmsg` を取り、`~/` で始まるときだけ `os.UserHomeDir` で展開する。`orgWatchChildAgmsgHome`(`internal/cli/org_watch.go:369-385`)も、env が空でなければそれ(`:370`)、空なら `--config` を明示して空でないときだけ設定の値(`:371-376`)を取る。空の env を未設定と同じに扱う点も一致する。絶対パスと `~/` で始まる値は触らない(`:377`)。既定の値は `~/` で始まるので触らない。`~` だけの値や `~user/…` は `expandTilde` が展開せず、作業ディレクトリから読む相対の値になるので、それを絶対パスにするのも `ResolveAgmsgHome` の読み方と合う
- 設定の値の出どころ: 2 つの呼び手(`internal/cli/org.go:300`、`internal/cli/org_watch.go:611`)はどちらも `rt.Config.AgmsgHome` を渡す。`rt.Config` は `resolveOrgConfig(configPath)`(`internal/cli/org.go:391`)が読んだ設定で、start の `withMainWorktreeOrgLimits` が書き換えるのは `MaxOrgs` と `MaxTotalSeats` だけなので、`AgmsgHome` は `--config` のファイルの値のままである
- `--config` を渡さないときに触らないこと: 子は `--config` なしで起動し(`orgWatchChildArgs`、`:284`)、作業ディレクトリ(main worktree のルート)の `ralph.toml` を読むので、親が読んだ `./ralph.toml` とは別のファイルの値になる。`--config ""` のときも子に `--config ""` を渡し(`:284-292`)、`resolveOrgConfig("")` が同じく作業ディレクトリの `ralph.toml` を読む。どちらも計画の F6 と README の「without `--config` it reads that root's `ralph.toml`」のとおりで、絶対パスにしないのが正しい
- 起点: 絶対パスにする起点は親の作業ディレクトリ(`filepath.Abs`、`:380`)で、triage と consult が選んだ案 A のとおり
- 作業ディレクトリを変えない子(台帳に main worktree がない)は、継承した値をそのまま使う(`:335`)。base と同じ動きである
- 新しい問題は見つからなかった。self-review の C3-1(コメントが triage の項目番号を指す)と C3-2(`org.CommandWithStateDir` が `org.InboxCommand` の写し)は、どちらも挙動に触れない

#### Design decisions F1〜F10 とコード

cycle 2 の判定(付録)のまま、10 個ともコードと一致する。8d8ddbd2 は F6(子の作業ディレクトリと環境)に、作業ディレクトリを移したことで指す場所の変わる相対の agmsg home の手当てを足した。F6 が決めた 3 つ(main worktree のルートで動く、`--config` を絶対パスで渡す、環境から HERDR_* の 4 つを外す)は変わらない。F6 の「HERDR_* の 4 つだけを外す」(Scope の「ほかの変数は残す」)とのずれは、下の「計画からのずれ」1 で扱う。ほかの決定のコードは cycle 3 で変わっていない。

#### 仕様 FR-6 との対応

cycle 2 の判定(付録)から変わらない。8d8ddbd2 は起動した子の環境の話で、要件の読み方は変えていない。仕様の「6 段目の後半で決めたこと」の注(`docs/specs/2026-10-07-org-multi-org-director.md:64`)は子の性質を 3 つ挙げるが、相対の agmsg home には触れていない(D15、任意)。

#### 計画からのずれ(cycle 3 で Progress に足された記録)の判断

1. 8d8ddbd2 の直し(Progress、`docs/plans/active/2026-10-12-org-watch-gate.md:200`): 妥当。F6(`:103`)と Scope(`:45`)は「HERDR_* の 4 つを外し、ほかの変数は残す」と書くが、作業ディレクトリを移すときは、相対の `RALPH_ORG_AGMSG_HOME` を絶対パスに書き換え、env が空か未設定で、明示した `--config` に相対の `agmsg_home` があればこの変数を足すようになった。どちらも、作業ディレクトリを移したために別の場所を指すようになった値を、打った場所で指していた場所に戻すものである。「ほかの変数は残す」の理由(XDG の変数で herdr の server を分けている環境のため)には触れない。退けた 3 案(`ResolveAgmsgHome` で絶対パスにする、作業ディレクトリを移さない、tech-debt に書くだけ)の理由は triage にある
2. 上限を 3 に上げ、全段を cycle 3 として回し直したこと(Progress `:199`): ユーザーの指示と `post-implementation-pipeline.md` の規則どおり。`cycle-count.json` は cross-review の規則で 2 のまま
3. 47635690 をテスト 1 本の追加として run-verify と cross-review に回したこと(Progress `:197`): テストを足しただけで挙動は変わらない。cycle 3 の self-review(1005f3bc)とこの verify が読んだ
4. C3-1 は cycle 3 の triage が cycle 2 の内容を付録に残して指し先を保ち、C3-2 は /sync-docs が tech-debt に書く(Progress `:201`): 妥当。どちらも挙動に触れない LOW で、直すと全段のやり直しになる。C3-2 の行は D14

### Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし(飛ばした linter はない)。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq -e`、Codex の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(dc41c4f5..522776ac、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest <plan>` | de5ca6b7e6f0 | `- Approved:` 行と一致 |
| `go build -o <scratchpad>/ralph ./cmd/ralph`(HEAD 522776ac) | 成功 | 実機確認のため |

### Documentation drift

cycle 2 の D1〜D10 は 2fada512 で直った。D1(`docs/tech-debt/README.md:183` は `newOrgSpawnRuntime` を消えた名前として挙げるだけ)、D2(`:205` の呼び手に `start <task>`)、D3(`:202` の「still open」に取り消し線)、D4(`README.md:256` に「when the ledger has one」)、D5(`README.md:125` に `stopping` と `running (--once)`)、D6 と D10(skill の「待つ間に持ち主が終われば」と「最大で約 3 秒」。4 面とも同じ sha256)、D7(`runOrgWatchStop` の doc に stopping)、D8(`:211` の新しい行)、D9(`:208` の注に `orgWatchChildExitWait` の待ちとそれを固定するテスト。「as of S3 no live run」の文はなくなった)を確かめた。cycle 3 では D11〜D15 が増えた。どれも 8d8ddbd2 が足した挙動(相対の agmsg home を絶対パスにして子に渡す)か、self-review C3-2 の記録である。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D11 `README.md:256` | いいえ | 「Either way the watch runs detached in the ledger's main worktree root (…), so without `--config` it reads that root's `ralph.toml`.」のあとに、たとえば「A relative `RALPH_ORG_AGMSG_HOME`, or else the relative `agmsg_home` of the `--config` given, is passed to it made absolute from where the command was typed.」を足す。今の文だけを読むと、相対の値は main worktree のルートから読まれると受け取れる |
| D12 `.claude/skills/org/SKILL.md:176`(4 面) | いいえ | `watch` の行の「起動したプロセスは台帳の main worktree のルート(求められなければ打った場所)で動き、`--config` を絶対パスで受け取り、環境から `HERDR_PANE_ID`・…・`HERDR_ENV` を外される。」に、「相対の `RALPH_ORG_AGMSG_HOME`(空か未設定なら、渡した `--config` の相対の `agmsg_home`)は、打った場所から絶対パスにして渡される」を足す。`:177` の `start` の行の括弧(「start に渡した `--config` は絶対パスにして渡す。間隔は設定の値」)にも同じ一言を足すと揃う(こちらは任意)。4 面とも同じに |
| D13 `docs/tech-debt/README.md:206`(RESOLVED の注)と `:207`(行の RESOLVED の文) | いいえ | 注は子の作業ディレクトリ、`--config`、HERDR_* の 4 つ、`PWD` を挙げるが、8d8ddbd2 の手当てがない。注に「a relative agmsg home (`RALPH_ORG_AGMSG_HOME`, or else the `agmsg_home` of an explicit `--config`) is passed to the child made absolute from the launching directory, since the child no longer runs there (cross-review cycle 2 of feat/org-watch-gate, commit 8d8ddbd2)」と、それを固定するテスト 2 本(`TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild`、`TestOrgStart_ARelativeAgmsgHomeIsPassedAbsoluteToTheWatch`)を足す。`:207` の「(RESOLVED 2026-10-12 in feat/org-watch-gate: the child runs in the main worktree root without the herdr pane variables)」に「, with a relative agmsg home made absolute」を足すと揃う |
| D14 `docs/tech-debt/README.md`(新しい行、self-review C3-2) | いいえ | 行がない。self-review の「Tech debt identified」の行から (a)(C3-1)を除いた形で書く。C3-1 は cycle 3 の triage が cycle 2 の内容を付録に残して指し先を保つので、行に入れない。Debt item: `org.CommandWithStateDir`(`internal/org/watchgate.go:80`)は `org.InboxCommand`(`internal/org/escalate.go:85`)の `--state-dir` を足す 4 行の写しで、`--state-dir` の引用(`shellQuote`)の規則が `internal/org` の 2 か所にある。Impact: 実行時の害はない。片方の引用を変えると、もう片方が古いまま残る。Why deferred: cycle 3(上限 3、最後の cycle)で見つけた LOW で、直すと全段のやり直しになる。Trigger: 次に `internal/org` の `--state-dir` の引用を変えるとき、または `InboxCommand` か `CommandWithStateDir` を触るとき。`InboxCommand` の本体を `return CommandWithStateDir("ralph org inbox "+verb+" "+id, stateDir)` にする(`TestInboxCommand`、`internal/org/escalate_test.go:615`、の期待は変わらない見込み)。Related: `docs/reports/self-review-2026-10-12-org-watch-gate.md`(C3-2)、`docs/plans/active/2026-10-12-org-watch-gate.md`(Progress の self-review cycle 3 の記録。`:211` の行と同じく active の path で書く) |
| D15 `docs/specs/2026-10-07-org-multi-org-director.md:64`、`watch --help`(`internal/cli/org_watch.go:61-66`)、start の help(`internal/cli/org.go:691-696`)(任意) | ほぼ合う | どれも子の性質を挙げるが、相対の agmsg home には触れない。挙げていることは今も正しい(help は外す変数と動く場所を言うだけで、環境がそれだけ違うとは言わない)。help の文言は `internal/` の変更になるので、直さずに残すのを勧める。仕様の注に足すなら「相対の agmsg home は打った場所から絶対パスにして渡す」の一言 |
| ほかの文書(`/org` skill の残り、README の残り、AGENTS.md、agent-messaging、quality-gates、leader の雛形、`templates/base/ralph.toml` の `agmsg_home` の説明) | 合う | cycle 2 の判定(付録)から変わらない。`templates/base/ralph.toml:57-59` の `agmsg_home` の説明は相対の値の扱いに触れておらず、8d8ddbd2 でも崩れない |

D11〜D14 は /sync-docs で入れる。D15 は任意。

### Observational checks

HEAD 522776ac のビルドで、scratchpad の使い捨ての git repo(`main` に 1 コミット)を使って確かめた。`env -i` で HOME と XDG の 5 変数を scratchpad に向け、PATH は git・sh・bash の 3 つの symlink だけ(osascript も herdr もない)、`SHELL=/bin/bash`、親の環境に `HERDR_PANE_ID=probe-pane` を置いた。打った場所は repo の `pkg/sub`、stub の agmsg home は repo の `stub-agmsg`(`pkg/sub` から `../../stub-agmsg`)、台帳は `--state-dir <repo>/.harness/state/org`。子の環境は `ps eww`、作業ディレクトリは `lsof -d cwd` で読み、毎回 `--stop` で止めた。herdr の server は立てていない。出力は evidence の log の末尾にある。

| 場合 | `--detach` | 子の作業ディレクトリ | 子の `RALPH_ORG_AGMSG_HOME` | `--stop` |
| --- | --- | --- | --- | --- |
| 相対の env(`../../stub-agmsg`) | rc 0、`started` | repo のルート | `<repo>/stub-agmsg` の 1 件 | rc 0 |
| env なし、`--config ralph.toml`(`agmsg_home = "../../stub-agmsg"`) | rc 0、`started` | repo のルート(子の引数は `--config <repo>/pkg/sub/ralph.toml`) | `<repo>/stub-agmsg` の 1 件 | rc 0 |
| 対照: env なし、`--config` なし、`./ralph.toml` に相対の `agmsg_home` | rc 0、`started` | repo のルート | なし | rc 0 |
| 対照: `--state-dir` が repo の外、相対の env | rc 0、`started` | 打った場所(`pkg/sub`) | `../../stub-agmsg` のまま | rc 0 |

- 4 回とも子の環境に `HERDR_*` はなく、作業ディレクトリを移した 3 回は `PWD` がそのルートの 1 件だった(AC6)
- 各回の `--stop` の 1 秒後と最後に、scratch のバイナリのプロセスは 0 だった(`pgrep -f`)。repo の台帳の `watch.log` の banner は 3 行で、起動 3 回に 1 行ずつ
- 台帳に org がないので、見張りは agmsg を呼ばない。子が絶対パスの home で agmsg の script を呼べるところまでは見ていない
- cycle 1 から残っていた「main worktree の中の台帳での子の作業ディレクトリ」は、`--detach` について今回の実機で確かめた

### Coverage gaps

- テストは走らせていない(/test の担当)。8d8ddbd2 の 13 ケースが通ること、各ガードを外すと落ちることは読んだだけである(self-review も読んで確かめ、Progress は mutation 6 体がすべて落ちたと書く)
- 実機は `--detach` だけで、start の経路(`wireOrgStartWatchGate`)の相対の agmsg home は偽の起動のテスト(`TestOrgStart_ARelativeAgmsgHomeIsPassedAbsoluteToTheWatch`)だけで見ている
- 子が絶対パスの home で agmsg の script を呼ぶところ(ALERT の送信、history の読み取り)は、org のない台帳なので通っていない
- linked worktree から打つ場合、打った場所がシンボリックリンクの下にある場合(`/tmp` から打つなど)、`filepath.Abs` が失敗する枝は試していない
- cycle 2 の Coverage gaps のうち、C2-1 の流れの実機、残る穴(cycle 2 の D8)の再現、本物の後始末の間の拒否、AC8 の残り、`start --plan` の起動と並行の start の実機、Linux、前の版の見張りとの共有は、cycle 3 でも変わらない
- 計画の受け入れ条件のチェックボックス(AC1〜AC15)はまだ付いていない。これまでと同じく、検証のあとに付ける段取りと読んだ

## Verdict

pass

- Verified: 8d8ddbd2 のあとも AC1〜AC11 と AC13〜AC15 を満たす。AC6 は、作業ディレクトリ、`--config` の絶対パス、HERDR_* を外すこと、`PWD` が cycle 2 と同じで、相対の agmsg home を絶対パスの 1 件にして渡すことが足された。その順(env が空でなければ env、なければ明示した `--config` の値)と、絶対パス・`~/` を触らないことは `driver.ResolveAgmsgHome` の優先順位と展開に合う。AC5 の本番の待ちの長さは 47635690 が固定した。F1〜F10 と仕様 FR-6 の対応は cycle 2 のまま。Progress の 8d8ddbd2 のずれ(F6 の「ほかの変数は残す」から、相対の agmsg home だけを書き換える)は妥当。cycle 2 の文書のずれ D1〜D10 は 2fada512 で直った。静的解析(changed scope)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。実機で、相対の env と `--config` の相対の `agmsg_home` の 2 通りで子が絶対パスの 1 件を受け取り、対照の 2 通りでは変わらないこと、毎回の `--stop` のあとに見張りが残らないことを確かめた
- Partially verified: AC12(D11〜D14 を /sync-docs で入れる。D15 は任意)、AC8(座席の `wait` と `inbox show/ack/resolve/notify` は grep だけ)
- Not verified: テストの実行、start の経路の相対の agmsg home の実機、子が絶対パスの home で agmsg を呼ぶところ、linked worktree とシンボリックリンクの下からの起動、C2-1 の流れの実機、本物の後始末の間の拒否、Linux

## 付録: cycle 2 の結果(HEAD 5739e1ba の時点)

cycle 2 の header(Date から Evidence まで)と本文を、そのとき書いたまま残す。`## Latest run: cycle 2(HEAD 5739e1ba)` の見出しはこの付録の見出しに替え、本文の見出しには「(cycle 2)」を付け、判定の見出しは `## Verdict` から 1 段下げて「判定(cycle 2)」にした。見出しの深さは下の cycle 1 の付録と同じになる。本文の行番号は cycle 2 の時点のものである。D1〜D10 は、上の cycle 3 の「Documentation drift」が 2fada512 で直ったと確かめたものと同じものを指す。

- Date: 2026-10-12(cycle 2)
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み。`scripts/plan-visual.sh digest` の値 de5ca6b7e6f0 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 2 回目(cycle 2、上限 2)。/test cycle 1 が実機で見つけた後退を fdfb5b23 で直したあとの、全段のやり直し。cycle 1 の結果は末尾の「付録」に残してある
- Scope: 仕様適合、静的解析、文書の整合。ブランチ全体は `git diff dc41c4f5...HEAD`(HEAD 5739e1ba)。主に見たのは cycle 1 の verify(ae349cc4)のあとのコードの変更 2 つで、`git diff ae349cc4..HEAD -- internal/` は 8 ファイル、+670/-41。fdfb5b23 は後退の直しと /test の probe 5 本の取り込み(`internal/` で +433/-16)、3fb0fd1b は cycle 2 の self-review の C2-1〜C2-3 の直し(+247/-35)。3fb0fd1b は self-review を通していないので全文を読んだ。テストは走らせていない(/test の担当)。HEAD のビルドで、後退を見つけた手順(同時の `--detach` 8 本のすぐあとの `--stop`)を実機で流し直した
- Evidence: `docs/evidence/verify-2026-10-12-org-watch-gate.log`(gitignore の対象で、コミットしない。cycle 2 の出力は末尾に足した)

### Spec compliance(cycle 2)

cycle 1 の判定(付録)から、2 つのコミットで変わった AC を見直した。2 つのコミットが触れたコードは `internal/cli/org_watch.go`、`internal/cli/org.go`(start の起動の失敗の案内に `hintStateDir` を渡す 1 行)、`internal/org/watchalive.go`(`defaultWatchLockWait` を `DefaultWatchLockWait` として出しただけ)、`internal/org/watchgate.go`(`CommandWithStateDir` の切り出し)、`internal/org/watch.go`(コメント 1 行)とテストだけである。検査の位置、TASK の判定、印、status、雛形のコードは変わっていない。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC5 子が先に終わっても、ほかの見張りが running なら成功 | 満たす(返るまでの待ちが足された) | `launchAndAwaitOrgWatch`(`internal/cli/org_watch.go:482`)は、作業を受け付ける見張りを見たとき、印が自分の子の pid なら `started`(`:501-502`)、子が終わっていればほかの見張りの pid を返す(`:504-505`)。子がまだ生きていれば、最初に見た時点から `orgWatchChildExitWait`(`:264`。`org.DefaultWatchLockWait` の 2 秒 + 1 秒)まで待ち、過ぎたら子を止めてから判定し直す(`:507-512`)。AC5 の文(子が先に終わっても ProbeWatch が running なら、start は進み、`--detach` は `already running (pid N)` で終了コード 0)はそのまま成り立ち、子が生きている間は返るのが遅れる。止めたあとの判定で見張りがいなくなっていれば、新しい文で終了コード 1 になる(`:533-536`)。これは「ProbeWatch が running を返す」という AC5 の前提が崩れた場合なので、AC と矛盾しない。cycle 1 からある AC5 のテスト 3 本(`TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`、`internal/cli/org_watch_test.go:462`、`_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning`、`:487`、`TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn`、`internal/cli/org_start_watch_test.go:310`)は cycle 2 で 1 行も消えていない。新しいテストは `TestOrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited`(`org_watch_test.go:569`)、`_AChildThatDoesNotExitInTimeIsStopped`(`:608`)、`_AStoppedChildIsJudgedByAProbeAfterTheStop`(`:652`)、`_AChildThatTakesTheLockMeanwhileIsStarted`(`:726`)、`TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited`(`org_start_watch_test.go:346`)。実機では、同時の `--detach` 8 本 × 3 回が、どの回も 8 本すべて rc 0(`started` 1 本、`already running` 7 本)で、返った時点の見張りのプロセスは 1 つだった。すぐあとの `--stop` は 0.86〜0.87 秒で rc 0、3 秒後のプロセスは 0、`watch.log` の banner は 1 行だった |
| AC15 最初の印の前の stale を待つ | 満たす(変わらない) | 最初の判定が stale のときの待ちは `ensureOrgWatch` の最初のループ(`org_watch.go:437-445`)と `NewWatchGate`(`internal/org/watchgate.go:123-141`)で、どちらも cycle 2 でコードは変わっていない(fdfb5b23 は `ensureOrgWatch` の doc に 1 文足しただけ)。自分の子が、ほかの見張りの最初の印より前に終わる場合は、`launchAndAwaitOrgWatch` の else の枝が今までどおり stale の間だけ待つ。AC15 のテスト(`TestOrgStart_WaitsForAWatchThatWritesItsFirstMarkLate`、`org_start_watch_test.go:513`、`TestOrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning`、`org_watch_test.go:928`、`TestOrgSpawn_WatchGate_AStaleWatchThatTurnsRunningWithinTheSettlePasses`、`internal/cli/org_watch_gate_test.go:212`)は cycle 2 で消えた行がない。本番の settle の 2 秒は、新しい `TestOrgSpawn_WatchGate_TheProductionSettleIsTwoSeconds`(`org_watch_gate_test.go:248`)が固定する |
| AC4 running にならない子の回収 | 満たす(案内の文だけが変わった) | 子を止めてロックの解放を待つ流れ(`org_watch.go:540-547`)は同じ。ロックが残ったときの文が、`--state-dir` を明示したときは `ralph status --state-dir '<台帳>'` を案内するようになった(`:543`)。`TestOrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir`(`org_start_watch_test.go:278`) |
| AC10 既存のテストを書き換えない | 満たす | cycle 2 でテストから消えた行は 2 つだけで、どちらもこのブランチで足したテスト(`TestOrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused` と `TestOrgWatchDetach_AStoppingOrOnceWatch_StartsNothing`)の期待に `--state-dir` が加わったもの(C2-3)。6a のテストは cycle 2 で変わっていない。6a の `TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt` は「ralph org watch --stop」を部分一致で見るので、`--state-dir` が後ろに付いても期待は同じ |
| AC12 文書の整合と同期の検査 | 一部(D1〜D5 と新しい D8・D9 を /sync-docs で直す) | `check-sync.sh` と `check-skill-sync.sh` は rc 0。cycle 1 のあとに文書へのコミットはない(`git log ae349cc4..HEAD -- README.md docs/tech-debt/README.md .claude/skills docs/specs AGENTS.md .claude/rules docs/quality internal/org/prompts` は空)ので、D1〜D7 は残っている。下の「Documentation drift」 |
| AC1〜AC3、AC6〜AC9、AC11、AC13、AC14 | 満たす(変わらない) | どれも、cycle 2 で関わるコードが変わっていない。cycle 1 の Coverage gaps に挙げた「stale を待つ間に持ち主が終わって起動へ進む分岐」は、取り込んだ probe の `TestOrgWatchDetach_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`(`org_watch_test.go:956`)と `TestOrgStart_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch`(`org_start_watch_test.go:550`)が固定するようになった。前半の版の印(`deadman_minutes` なし)を呼んだ側の設定で読むことは `TestOrgWatchDetach_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`(`org_watch_test.go:1308`)、`_TheFirstProbeJudgesAMarkWithoutDeadmanMinutesByTheCallersConfig`(`:1326`)、`TestOrgStart_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig`(`org_start_watch_test.go:588`)が、起動の失敗の文の接頭辞は `TestOrgWatchDetach_ALaunchThatFails_IsTheErrorAfterTheDetachPrefix`(`org_watch_test.go:446`)と `TestOrgStart_ALaunchThatFails_IsRefusedWithThePrefixAndNothingStarted`(`org_start_watch_test.go:257`)が固定する。AC8 の残り(座席の `wait` と `inbox show/ack/resolve/notify` は grep だけ)は cycle 1 と同じ |

#### Design decisions F1〜F10 とコード(cycle 2)

cycle 1 の判定(付録)のまま、10 個ともコードと一致する。2 つのコミットが触れたのは起動の待ち(F8 の tech-debt 206)と、`--detach` と start の案内の文だけである。検査の位置(F1〜F3)、TASK の判定(F4・F5)、子の作業ディレクトリと環境(F6)、stopping と `once`(F9)、stale の待ち(F10)のコードは変わっていない。F8 の 206(同時の `--detach` が終了コード 1 になる)は、直ったまま後退も起きていないことを実機で確かめた(AC5 の行)。

#### 仕様 FR-6 との対応(cycle 2)

cycle 1 の判定(付録)から変わらない。FR-6 の残り 4 項目と、受け入れ条件 107 行の読み替え(F7)を満たす。後退の直しは「start は見張りが動いていなければ起動する」の起動の経路の中だけの話で、要件の読み方は変えていない。

#### 計画からのずれ(cycle 2 で Progress に足された記録)の判断(cycle 2)

1. fdfb5b23 の直し方((a) 自分の子が終わるのを待つ): 妥当。org 層の規則(ロックの待ち、止める頼みの扱い)を変えず、待ちの長さを org 層の定数 `DefaultWatchLockWait` から決めるので、片方だけが変わることもない。退けた 2 案(子のロックの待ちを 0 にする、止める頼みを消さない)の理由も記録にある。直ったことは上の実機確認で確かめた
2. 残る穴(`--detach` が返る前に別の端末から `--stop` を打つと、ロックを待つ子があとを取りうる。foreground の `watch --all` も同じ。base にもある): 計画の範囲の外で、base からある動きなので、この PR で直さないのは妥当。ただし Progress は「tech-debt に書く」としていて、まだ書かれていない(D8)。cycle の上限(2/2)に達しているので、/sync-docs で必ず入れる
3. /test の probe 5 本の取り込み: 妥当。cycle 1 の verify が Coverage gaps に挙げた分岐(stale を待つ間に持ち主が終わる)も閉じた
4. コードの直しなので全段を cycle 2 として回し直したこと: `post-implementation-pipeline.md` の規則どおり。`cycle-count.json` は 2
5. 3fb0fd1b(C2-1〜C2-3)を self-review に通さず verify に回したこと: 下の「3fb0fd1b の確認」で読んだ。C2-4 は /sync-docs に回してあり(D8、D1〜D4)、上限での先送りに決め手が付いている

#### 3fb0fd1b の確認(cycle 2)

- C2-1(子を止めたあとの判定し直し): 子を止めたら `stopped, childExited, exited = true, true, nil` にしてループの頭に戻り、新しい probe で決める(`org_watch.go:510-512`)。結果は 3 通りある。ほかの見張りが作業を受け付けていればその pid を返す(`:504-505`)。stopping・`--once`・期限まで stale の見張りなら `e.refuse` で断る(`:530-531`)。ロックが空いていれば「the watch (pid N) was stopped while another watch held …, and that one has ended too, so no watch runs; see <watch.log>」で断る(`:533-536`)。止めて回収した子はロックを放しているので、その pid が作業を受け付けると読まれることはない。`child.stop()` は SIGTERM のあと最大 `orgWatchChildStopGrace`(5 秒)で SIGKILL を送るので、待ちには上限がある。doc の前提「SIGTERM ではロックの待ちが終わらない」は、`runOrgWatchAll` が SIGTERM を `signal.NotifyContext` で受けること(`:153`)と、`acquireHeldFileLock` が ctx を取らないこと(`internal/org/lockfile.go:118`)に合う。`continue` で select を 1 回飛ばすだけで、次の周回は `exited` が nil なので probe の間隔で待つ。空回りはしない。`exitBy` を決めたあとの枝は全体の期限を見ないが、`exitBy`(3 秒)で抑えられる。`TestOrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop`(`org_watch_test.go:652`)が 3 通りを見る
- C2-2(`runOrgWatchDetach` の doc): doc だけ。「nothing started」は消え、`already running` は自分の子が終わるか止められてから出すことを書いた(`:551-559`)
- C2-3(案内のコマンドに `--state-dir`): `CommandWithStateDir`(`internal/org/watchgate.go:80`)を org 層に出し、`watchGateCommand`(`:72-74`)もそれを使う。組み立ては前と同じなので、`WatchGateError` の文は変わらない。`--detach` の拒否の 2 通り(`org_watch.go:594`、`:601`)と、ロックが残ったときの起動の失敗(`:543`)が使う。start も同じ hint を渡す(`internal/cli/org.go:303`)。付けるのは `--state-dir` を明示したときだけ(`orgInboxHintStateDir`)。実機では、stale の台帳の `--detach` の拒否が `stop it with ralph org watch --stop --state-dir '<台帳>'` と案内した。テストは `TestOrgWatchDetach_TheCommandsRepeatAnExplicitStateDir`(`org_watch_test.go:871`)と、AC4 の行の start のテスト
- 新しい問題は見つからなかった。cycle 1 の「気づいたこと」(`--detach` の文に `--state-dir` が付かない)は、この C2-3 で直った

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の guard、`check-pipeline-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(dc41c4f5..5739e1ba、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5 |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `./scripts/plan-visual.sh digest <plan>` | de5ca6b7e6f0 | `- Approved:` 行と一致(Progress への追記は digest に入らない) |
| `go build ./cmd/ralph`(一時の場所へ、HEAD 5739e1ba) | 成功 | 実機確認のため |

### Documentation drift(cycle 2)

cycle 1 の D1〜D7 は、文書へのコミットがないので、そのまま残っている。行番号も変わっていない(D7 だけ、コードが増えたので `internal/cli/org_watch.go:609-610` に移った)。cycle 2 で D8 と D9 が増えた。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `docs/tech-debt/README.md:183` | いいえ | F-7 の文が、消えた `newOrgSpawnRuntime` を挙げている。3 か所の `org.ResolveOrgStateDir` の呼び出し(`internal/cli/org.go:549`、`:725`、`:822`)と `newOrgSpawnRuntimeAt` の guard の形に書き直す |
| D2 `docs/tech-debt/README.md:205` | いいえ | (b) の `newOrgSpawnRuntimeAt` の呼び手「(spawn and `start --plan`)」に `start <task>` を足す |
| D3 `docs/tech-debt/README.md:202` | いいえ | 閉じた行の「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」を閉じる(取り消し線か RESOLVED の注) |
| D4 `README.md:256` | いいえ | 「in the ledger's main worktree root」に「(when the ledger has one)」を足す |
| D5 `README.md:125` | いいえ | `ralph status` の行の「running, stale, or not running」に stopping と `running (--once)` を足す |
| D6 `.claude/skills/org/SKILL.md:176`、`:177`、`:585`(任意) | ほぼ合う | stale を待つ間に持ち主が終わると起動へ進むことが書かれていない。直すなら 4 面とも |
| D7 `internal/cli/org_watch.go:609-610`(任意、コードのコメント) | ほぼ合う | `runOrgWatchStop` の doc の「running and stale are handled alike」に stopping を足す |
| D8 `docs/tech-debt/README.md`(新しい行、self-review C2-4) | いいえ | 計画の Progress(`docs/plans/active/2026-10-12-org-watch-gate.md:192`)が「tech-debt に書く」とした残る穴の行がない。内容: `--detach` か start が自分の子の終わりを待っている間、または foreground の `watch --all` がロックを待っている間に、別の端末から `--stop` を打つと、止まった見張りが止める頼みを消してロックを放したあとに、待っていたプロセスがロックを取り、長く動く見張りとして動き続ける。`--stop` は 30 秒で時間切れになる。base にもある。/test cycle 1 の「見つけたこと」と `docs/evidence/org-watch-gate-live-2026-10-12.md`(base との比較)を Related に挙げる |
| D9 `docs/tech-debt/README.md:208`(206 を閉じた RESOLVED の注) | いいえ | 注は「`--detach` は同時の `--detach` か start が起動した見張りに `already running` と言う」としか書かず、「as of S3 no live run of several concurrent `--detach` calls had been repeated」で終わる。その後 /test が実機で流して後退を見つけ、fdfb5b23 で `--detach` と start が自分の子の終わりを(`DefaultWatchLockWait` + 1 秒まで)待つようになった。注にこの 2 点と、流し直した結果(`--stop` が 1 秒ほどで通り、見張りが残らない)を足し、「as of S3 …」の文を直す |
| D10 `.claude/skills/org/SKILL.md:176`、`:177`(任意) | ほぼ合う | `watch` の行の「(同時に打った別の `--detach` か start の見張りが先にロックを取った場合も)、何も起動せず `already running (pid N)`」と、`start` の行の「同時に打った別の start か `--detach` の見張りが先にロックを取っても、running になれば進む」は、自分が起動した子が断られて終わるのを待ってから返る(最大で約 3 秒)ことを書いていない。前者の「何も起動せず」は、同時の場合には子を起動しているので正確でない。直すなら 4 面とも |
| ほかの文書(`/org` skill の残り、README の残り、AGENTS.md、仕様の注、agent-messaging、quality-gates、leader の雛形、各 `--help`) | 合う | cycle 1 の判定(付録)から変わらない。cycle 2 のコードの変更は、どれの記述も崩していない。`defaultWatchLockWait` と `orgWatchDetachDeadline` の古い名前は、コードにも文書にも残っていない(`git grep`) |

D1〜D5、D8、D9 は /sync-docs で直す。D6、D7、D10 は任意。

### Observational checks(cycle 2)

HEAD 5739e1ba のビルドで、repo の外の一時の台帳(`--state-dir`)を使って確かめた。herdr の座席は立てていない。出力は evidence の log の末尾にある。

- 同時の `--detach` 8 本を新しい台帳に打ち、すべて返った直後に `--stop` を打つ手順を 3 回流した。3 回とも 8 本すべて rc 0(`started` 1 本、`already running` 7 本)。8 本が返るまでは 2.15〜3.71 秒だった。返った時点の見張りのプロセスは 1 つ、`--stop` は 0.86〜0.87 秒で rc 0、その 3 秒後のプロセスは 0、`watch.log` の banner は 1 行、`ralph status` は `not running (stopped at …)`。/test cycle 1 が見つけた後退(30 秒で時間切れになり、見張りが 1 つ残る)は起きなかった
- python の flock でロックを持ち、1 時間前の印を置いた台帳で、`--detach` は 10 秒待ってから rc 1 で断り、案内は `stop it with ralph org watch --stop --state-dir '<台帳>'` だった(C2-3)
- 終わったあと、一時の ralph のプロセスは残っていない(念のための `pkill` の前に数えて 0)

### Coverage gaps(cycle 2)

- テストは走らせていない(/test の担当)
- `exitBy` を過ぎても子が生きていて止める流れ(C2-1 の 3 通りを含む)は、偽の子のテストだけで見ている。本物の子は 2 秒のロックの待ちで自分から終わるので、実機では通らなかった
- 残る穴(D8)は、計画どおり直していないので、実機でも再現を試していない
- cycle 1 の Coverage gaps のうち、本物の後始末(stopping)の間の拒否、AC8 の残り、main worktree の中の台帳での実機、Linux、前の版の見張りとの共有は、cycle 2 でも変わらない。stale を待つ間に持ち主が終わる分岐は、取り込んだ probe で閉じた

### 判定(cycle 2)

pass

- Verified: 後退の直し(fdfb5b23)のあとも、AC1〜AC11 と AC13〜AC15 を満たす。AC5 は、自分の子が生きている間は返るのを待つようになったが、文は変わらず成り立つ。AC15 の経路はコードが変わっていない。F1〜F10、仕様 FR-6 の対応、受け入れ条件 107 行の読み替えは cycle 1 のまま。cycle 2 で Progress に足されたずれ(直し方の選択、probe の取り込み、全段のやり直し、C2-4 の先送り)は妥当。3fb0fd1b に新しい問題はない。静的解析(changed scope)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。同時の `--detach` 8 本のあとの `--stop` が 1 秒ほどで通り、見張りが残らないことを、実機で 3 回確かめた
- Partially verified: AC12(D1〜D5、D8、D9 の文書を /sync-docs で直す。D6、D7、D10 は任意)、AC8(座席の `wait` と `inbox show/ack/resolve/notify` は grep だけ)
- Not verified: テストの実行、C2-1 の流れの実機での確認、本物の後始末の間の拒否、`start --plan` の起動と並行の start の実機での確認、Linux

## 付録: cycle 1 の結果(HEAD 9823d97d の時点)

cycle 1 の header(Date から Evidence まで)と本文を、そのとき書いたまま残す。見出しは 1 段下げて「(cycle 1)」を付け、判定の見出しは `## Verdict` から「判定(cycle 1)」に変えた。本文の行番号は cycle 1 の時点のものである。D1〜D7 は上の cycle 2 の D1〜D7 と同じものを指す。

- Date: 2026-10-12
- Plan: docs/plans/active/2026-10-12-org-watch-gate.md(承認済み。`scripts/plan-visual.sh digest` の値 de5ca6b7e6f0 は `- Approved:` 行と一致する)
- Verifier: verifier subagent (Claude)。パイプライン 1 回目(cycle 1、上限 2)
- Scope: 仕様適合、静的解析、文書の整合。対象は `git diff dc41c4f5...HEAD`(HEAD 9823d97d、36 ファイル、+3,727/-350)。self-review を通していない bda15444 は全文を読んだ。テストは走らせていない(/test の担当)。ブランチのビルド(`go build ./cmd/ralph`)で、repo の外の一時の台帳を使った実機確認をした。herdr の座席は立てていない(spawn と send は `--dry-run`、start は未知の model で envelope に断らせた)
- Evidence: `docs/evidence/verify-2026-10-12-org-watch-gate.log`(gitignore の対象で、コミットしない)

### Spec compliance(cycle 1)

コードとテストを読み合わせて判定した。テストが通るかどうかは /test で確かめる。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 見張りがないときの spawn の拒否 | 満たす | 検査は `checkSpawnInput` の直後で、manifest の読みと dry-run の枝(`internal/org/spawn.go:493`)より前にある(`:470`)。拒否は `SpawnOutcomeRejected` を返すだけで `reject()` を通らないので、`rejected` も receipt も書かない。CLI の配線は `wireOrgWatchGate`(`internal/cli/org.go:266`、spawn は `:554`)。状態ごとの次の手は `WatchGateError.Error`(`internal/org/watchgate.go:43-68`)。テストは `TestOrgSpawn_WatchGate_RefusesAWatchThatAcceptsNoWork`(`internal/cli/org_watch_gate_test.go:154`。6 状態それぞれで、実行と `--dry-run` の両方を見る。`checkNothingStarted` が manifest・receipt・herdr・agmsg を確かめる)と `TestOrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing`(`internal/org/watchgate_positions_test.go:37`)。running で通ることは `TestOrgSpawn_WatchGate_ARunningWatchPasses`(cli `:181`)。実機では、見張りのない台帳で `spawn --dry-run` が rc=1 になり、manifest は作られなかった |
| AC2 TASK の send だけを断る | 満たす | `Send` は最初に `protocol.Parse` を呼び(`internal/org/verbs.go:293`)、TYPE が `TASK` のときだけ検査する(`:303-307`)。検査は dry-run の枝(`:314`)より前にある。raw のメッセージにも同じ `Parse` を使い、`Parse` が key と値を `TrimSpace` する(`internal/org/protocol/protocol.go:140`、`:144`)。テストは `TestOrgSend_WatchGate_RefusesOnlyATask`(cli `:289`)と `TestOrgSend_WatchGate_RefusesATaskOnly`(`watchgate_positions_test.go:161`)。実機では、見張りなしで型付きの TASK と raw の `  TYPE:  TASK` が rc=1 になり、raw の `type: task` と QUESTION は rc=0 で送れた。manifest に残ったのは、通った 2 件の dry-run の記録だけ |
| AC3 start が見張りを起動する | 満たす | `wireOrgStartWatchGate`(`internal/cli/org.go:290-318`)が `ensureOrgWatch`(`internal/cli/org_watch.go:427`)を呼んでから本物の検査を呼ぶ。起動したときは stderr に 1 行を出す(`org.go:313`)。配線は start の 2 つの形にある(`:730`、`:827`)。起動した子は `WatchAcceptsWork` になるまで待つ(`org_watch.go:473-474`)。テストは `TestOrgStart_LaunchesTheWatchWhenNoneRuns`(`internal/cli/org_start_watch_test.go:152`)。実機では、kill -9 のあとの `start <task>` が `ralph org start: started the cross-org watch (pid 48334); log: …/watch.log` を出し、start が終わったあとも `ralph status` は `org watch: running (pid 48334)` だった。2 回目の start は何も起動しなかった。leader は envelope で断らせたので、「印が running になってから leader が立つ」順は実機では見ていない(テストが見る) |
| AC4 stale の拒否と、running にならない子の回収 | 満たす | stale の間は最大 `orgWatchEnsureDeadline`(10 秒)待ち、そのあとは `e.refuse` を返す(`org_watch.go:427-446`)。start の `refuse` は `WatchGateError` を返し、`StartFeature` は検査を worktree の前で呼ぶ(`internal/org/feature.go:198`、worktree は `:210` 以降)。起動した子が期限までに running にならなければ、子を止めてロックの解放を待ってから返す(`org_watch.go:498-507`)。テストは `TestOrgStart_WatchGate_RefusesAStaleWatch`(cli `org_watch_gate_test.go:396`)、`TestOrgStartPlan_WatchGate_RefusesAStaleWatchBeforeTheWorktree`(`:421`)、`TestOrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused`(`org_start_watch_test.go:213`)。実機では、ロックを持ったまま 1 時間前の印を置いた台帳で、`--detach` が 10 秒待ってから stale の文で rc=1 になった(start で同じ台帳は試していない) |
| AC5 子が先に終わっても、ほかの見張りが running なら成功 | 満たす | `launchAndAwaitOrgWatch` は、どの pid の見張りでも `WatchAcceptsWork` なら成功とし、`launched` を pid で判定する(`org_watch.go:473-474`)。`--detach` は `launched` で `started` と `already running` を出し分ける(`:528`、`:531`)。テストは `TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild`(`internal/cli/org_watch_test.go:442`。終了コードの期待だけを 0 に変え、pid の照合は残した)、`TestOrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning`(`:467`)、`TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn`(`org_start_watch_test.go:256`)。並行の `--detach` を実機で重ねて打つことはしていない |
| AC6 子の作業ディレクトリ、`--config`、環境 | 満たす | `orgWatchChild`(`org_watch.go:312-333`)は `org.LedgerMainWorktreeRoot` で main worktree のルートを求め(`:319`)、`orgWatchChildDroppedEnv` の 4 つ(`:300`)を外し、作業ディレクトリを変えたときは `PWD` も合わせる(`:331`)。`--config` は `filepath.Abs` で絶対パスにする(`orgWatchChildArgs`、`:277`)。`Env` を明示すると os/exec が `PWD` を直さない、というコメントの前提は Go の `Cmd.environ` の動き(`Env` が nil のときだけ `PWD` を足す)と合う。テストは `TestOrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_start_watch_test.go:458`)、`TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv`(`org_watch_test.go:670`)、`TestOrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv`(`:748`) |
| AC7 印の `deadman_minutes` | 満たす | 見張りは設定の値をそのまま印に書く(`internal/org/watch.go:603`、`:618`)。`ProbeWatch` は印に値があればそれを使い(`internal/org/watchalive.go:215-217`)、なければ呼んだ側の値を使う。テストは `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`(`internal/org/watch_all_test.go:861`)、`TestProbeWatch_JudgesTheWindowByTheDeadmanMinutesOfTheMark`(`internal/org/watchalive_test.go:267`)、`TestStatusCmd_OrgWatchJudgesTheMarkByTheWatchsOwnDeadmanMinutes`(`internal/cli/status_test.go:862`)、`TestStatusCmd_OrgWatchOfAWatchWithAnotherConfigReadsByItsMark`(`:899`)。実機の印にも `"deadman_minutes": 10` があった |
| AC8 見張りなしで使える動詞 | 満たす(一部は構造だけで確認) | `wireOrgWatchGate` と `wireOrgStartWatchGate` を呼ぶのは spawn・`start <task>`・`start --plan`・send の 4 か所だけである(`org.go:554`、`:730`、`:827`、`:901`。grep で確認)。`TestOrgVerbsThatStartNoWork_WorkWithNoWatch`(`org_watch_gate_test.go:497`)が本物の検査のもとで `ralph status`・org status・read・report・escalate・inbox・`wait --inbox`・stop・disband・`stop --all`・`disband --all` を通す。座席の `wait` と、`inbox show/ack/resolve/notify` はこのテストに入っていない。どれも検査を配線しないことは grep で確かめた |
| AC9 2 つ目の見張りは banner を出さない | 満たす | banner は `WatchAllParams.OnLocked` から出す(`org_watch.go:170`、`:218`)。`RunWatchAll` はロックを取ってから呼ぶ(`watch.go:539-541`)。テストは `TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock`(`org_watch_test.go:191`)と `TestRunWatchAll_CallsOnLockedOnceItHoldsTheLock`(`watch_all_test.go:736`)。実機では、動いている見張りの台帳で 2 つ目の `watch --all --once` が rc=1 になり、stdout は 0 バイトだった |
| AC10 既存のテストを書き換えない | 満たす(記録の漏れは下の「計画からのずれ」7) | CLI 側は TestMain が `orgWatchGateOverride` を「いつも通す」にし(`internal/cli/main_test.go`)、org 側は `Org.WatchGate` の既定が nil である(`spawn.go` の `Org` の欄)。既存のテストで行が消えたのは `internal/cli/org_watch_test.go` と `internal/org/watch_all_test.go` だけ(`git diff` の `-` 行で確認)。期待が変わったのは、Progress に書いた 4 本(後始末の間の印、`--once` の probe、AC5 の終了コード、stale の待ちを 300ms にしたもの)に、bda15444 の文言の変更(`nothing was started` から `so no new watch runs`)が加わる。ほかは起動の差し替え口の型(`orgWatchLaunchRequest`)に合わせた書き換えと、`orgWatchDetachDeadline` の改名で、期待は変えていない |
| AC11 leader の雛形 | 満たす | `internal/org/prompts/leader.md:236-248` の `## 運用規律` の項目。`TestRenderRolePrompt_Leader_WatchRefusalEscalatesWithoutRetry`(`internal/org/prompts_test.go:705`)が語句を固定し、`WatchGateError` の 5 通りの文が雛形の見分け方の語(`ralph org watch --all` と `no new work`)を含むことも確かめる。send の拒否は `taskNotSentError` が検査の文をそのまま後ろに付けるので、同じ語を含む(実機の文でも確認) |
| AC12 文書の整合と同期の検査 | 一部(文書の残り 5 件を /sync-docs で直す) | `scripts/check-sync.sh`(DRIFTED 0)と `scripts/check-skill-sync.sh`(13 skill)は rc 0。skill の 4 面と agent-messaging の 2 面は `cmp` で一致する。文書の残りは下の「Documentation drift」の D1〜D5 |
| AC13 止まりかけの見張り | 満たす | `drainBeforeStop` が後始末の前に stopping を書き、間隔ごとに書き直す(`watch.go:781-806`)。`stopAndRelease` が stopped を書いてからロックを放す(`:814`)。`ProbeWatch` は stopping を返し(`watchalive.go:222-224`)、`WatchAcceptsWork` は通さない(`watchgate.go:107-109`)。`--detach` と start は `ensureOrgWatch` の最初の判定で断る(`org_watch.go:440-441`)。テストは `TestRunWatchAll_CallsBeforeStopWhileItStillHoldsTheLockAndTheMarkSaysStopping`(`watch_all_test.go:1222`)、`TestProbeWatch_AStoppingMarkIsStoppingOnlyWhileHeldAndFresh`(`watchalive_test.go:224`)、`gateRefusals` の stopping の行(spawn)、`TestOrgStart_AStoppingOrOnceWatch_IsRefusedAndNothingLaunched`(`org_start_watch_test.go:345`)、`TestOrgWatchDetach_AStoppingOrOnceWatch_StartsNothing`(`org_watch_test.go:603`)、`TestStatusCmd_OrgWatchShowsStoppingAndOnce`(`status_test.go:777`)。spawn と start のテストは、ロックを持ったプロセスに stopping の印を置いた台帳で見る。本物の見張りが後始末をしている間に spawn を打つテストはない(印の書き手と検査を別々に固定している) |
| AC14 `--once` の見張り | 満たす | 印の `Once` は `p.Cycles > 0`(`watch.go:619`)。`WatchAcceptsWork` が通さず、`ralph status` は `running (--once)` を出す(`internal/cli/status.go:172-175`)。`--json` には `once` が足された(`:434`)。テストは AC13 と同じ 4 本の `--once` の行と `TestRunWatchAll_TheMarkCarriesTheWatchsDeadmanMinutesAndOnce`。実機では、`--once` の見張りの最後の印に `"once": true` があった。動いている間の status の行は、1 サイクルがすぐ終わったので見られなかった |
| AC15 最初の印の前の stale を待つ | 満たす | spawn と TASK の検査は `NewWatchGate` が stale のときだけ settle(2 秒)まで判定し直す(`watchgate.go:123-141`)。start と `--detach` は `ensureOrgWatch` が 10 秒待つ(`org_watch.go:428-436`)。テストは `TestOrgStart_WaitsForAWatchThatWritesItsFirstMarkLate`(`org_start_watch_test.go:387`)、`TestOrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning`(`org_watch_test.go:640`)、`TestOrgSpawn_WatchGate_AStaleWatchThatTurnsRunningWithinTheSettlePasses`(cli `org_watch_gate_test.go:212`)、`TestNewWatchGate_AStaleWatchThatWritesItsMarkWithinTheSettle`(`internal/org/watchgate_test.go:156`)。実機では、本当に返らない見張りで spawn は 2 秒、`--detach` は 10 秒待ってから断った |

#### Design decisions F1〜F10 とコード(cycle 1)

| 決定 | コードとの一致 | 根拠 |
| --- | --- | --- |
| F1 拒否は記録しない | 一致 | spawn・StartFeature・Send の 3 か所とも、検査の拒否はエラーを返すだけで、`reject()` も `appendEvent` も通らない(`spawn.go:470-472`、`feature.go:198-200`、`verbs.go:303-307`) |
| F2 org 層の差し込み口、CLI の 3 コマンドだけが設定する | 一致 | `Org.WatchGate`(nil で検査しない、`watchgate.go:144-149`)。`newOrgRuntimeAt` には入れず、spawn・start・send の RunE だけが設定する。StartFeature は自分で呼び、そのあと Spawn がもう一度呼ぶ |
| F3 `--dry-run` も断る | 一致 | 3 か所とも dry-run の枝より前にある(`spawn.go:493`、`verbs.go:314`)。実機でも `--dry-run` が断られた |
| F4 宛先を問わず TASK を断る | 一致 | `Send` は `p.To` を見ずに `msg.Type` だけで決める(`verbs.go:303`) |
| F5 raw は `Parse` の 1 行目で読む | 一致 | `Parse` が `ErrMissingType` を返すのは 1 行目が `TYPE:` でないときだけである(`protocol.go:103-106`)。小文字の `type: task` は key が `TYPE` と一致しないので TASK ではない(実機で rc=0 を確認) |
| F6 子の作業ディレクトリ、`--config`、環境 | 一致 | AC6 の行を参照。start と `--detach` は同じ `orgWatchChild` を使う |
| F7 受け入れ条件の読み替え | 一致 | kill された見張りは `not_running` になり、start は見張りを起動して進む(実機で確認)。断るのは 10 秒たっても stale のときだけである。仕様の注(`docs/specs/2026-10-07-org-multi-org-director.md:64`)にこの読み替えが書いてある。PR の本文と最後の報告に書くのは /pr の役目で、まだ確かめられない |
| F8 tech-debt 201・204 (e)・205・206 を直す | 一致 | 201 は AC7、204 (e) は AC9、205 は AC6、206 は AC5 の行を参照。tech-debt の行は閉じてある(`docs/tech-debt/README.md:201-202`、`:205`、`:206-209`) |
| F9 stopping と `--once` の印 | 一致 | AC13 と AC14 の行を参照 |
| F10 stale の待ち(spawn と TASK は 2 秒、start と `--detach` は 10 秒) | 一致 | AC15 の行を参照 |

#### 仕様 FR-6 との対応(cycle 1)

| 仕様の項目 | 判定 | 根拠 |
| --- | --- | --- |
| 印が古いとき、start・spawn・実装の TASK の送信を断る | 満たす | AC1、AC2、AC4。「実装の TASK」を TYPE が TASK のすべての送信と読むことは、仕様の注(`:64`)に理由とともに書いてある(F4) |
| start は見張りが動いていなければ起動する | 満たす | AC3 |
| 横断の stop は見張りなしで使える | 満たす | AC8 |
| 雛形と現セッション昇格の手順の書き換え | 満たす | AC11。昇格の手順は `.claude/skills/org/SKILL.md:681-683` が `ralph org watch --all --detach` を先に打つと書く |
| 受け入れ条件 107 行(kill のあとの start) | 読み替えで満たす | F7 の行。kill -9 のあとの start が見張りを起動して進むことを実機で確かめた。ロックを持ったまま印の古い台帳では、`--detach` が 10 秒待ってから断ることを実機で確かめた(start は同じ経路の `ensureOrgWatch` を使う) |
| 「6 段目の後半で決めたこと」の注 | コードと一致 | 注の各文(2 秒と 10 秒の待ち、stopping と `once`、印の `deadman_minutes`、raw の読み方、入力の検査で断られる start は見張りを起動しないこと、それより後の検査で断られた start は見張りを残すこと)は、上の各行のコードと合う。後の検査で断られた start が見張りを残すことは、実機(未知の model で断られた start のあとも見張りが running)で確かめた |

#### 計画からのずれ(Progress checklist の記録)の判断(cycle 1)

1. S1: `ralph status` の stopping と `(--once)` の表示を S2 から S1 に移した。スライスの境目だけの話で、問題はない
2. S1: StartFeature の検査を `checkSpawnInput` の直後(`Manifest.Read` と `spawnPrecheckErr` の前)に置いた。F2 の「worktree の前」を満たし、Spawn と同じ順になる。max_orgs と見張りなしが重なれば見張りのエラーが先に出るが、どちらも記録を書かない拒否なので害はない。妥当
3. S1: 検査が打ち直しの冪等の戻りより前にあるので、立っている座席の打ち直しも見張りなしでは断られる。F2 の位置からの帰結で、何も書かない拒否なので状態は壊れない。妥当
4. S1 と S3: 期待を変えた 6a のテスト(後始末の間の印、`--once` の probe、AC5 の終了コード、stale の待ちの短縮)は、AC10 が許す範囲に入る
5. S2: spawn と `start <task>` が台帳を 1 回だけ解決するようにし、`newOrgSpawnRuntime` を消した。挙動は変わらない。ただし tech-debt の 183 行と 205 行の文が古くなった(D1、D2)
6. S3: エンベロープ・max_orgs・予約の検査と打ち直しの戻りで断られる start でも、起動した見張りは残る。見張りは止めるまで動くという前半の決定と合い、仕様の注と skill に書いてある。妥当。`PWD` の書き換えも正しい(AC6)
7. 記録にない小さなずれが 2 つある。どちらも妥当で、直す必要はない。(a) `ensureOrgWatch` は stale を待つ間にロックを持つ見張りが終わって `not_running` になると、そのまま起動に進む(`org_watch.go:428-446`。関数の doc には書いてある)。計画と図 2 は「stale なら起動せず待ち、running にならなければ断る」としか言わない。持ち主が消えたら起動するのは not_running の規則と同じ扱いで、合理的である。skill の文が少しずれる(D6、任意)。この分岐を固定するテストはない(Coverage gaps)。(b) AC10 が並べた期待の変更のほかに、bda15444 で 6a のテスト `TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt` の期待の文言が変わった。Progress の N2 の記録で理由は分かる
8. 7d2e4bbb(CLI の send が `org: send:` を重ねていたのを直した): この PR より前からある包みの直しで、計画の範囲の外にある。変わるのは二重だった `org: send: org: send: …` が 1 つになることだけで、文書もテストもその二重の文を前提にしていない(grep で確認)。Progress に記録がある。妥当

#### bda15444 の確認(cycle 1)

- N1(テストのコメントの空白): コメントだけ
- N2(`--detach` の拒否の文): 3 通りの文とも「so nothing was started」を「so no new watch runs」に変え、関数の doc に「起動した子がロックを断られて終わったあとも同じ文」と足した(`org_watch.go:535-551`)。文は子を起動した場合にも正しい。古い文言はコードにも文書にも残っていない(`git grep 'nothing was started'` はこの PR の文書とコードで 0 件)。期待を変えたテストは 3 本(`org_watch_test.go` の stale、子が終わったときの 3 状態、stopping と `--once`)
- N3(dry-run の追記のエラー): `verbs.go:314-322` が追記の失敗を `org: send: ` で包むようになった。`Send` のすべての戻り口を読み、どのエラーも `org: send: ` で始まることを確かめた(`:300`、`:305`(`taskNotSentError`)、`:319`、`:326`、`:329`、`:332`、`:357`、`:383`、`:396`、`:407`、`:418`、`:444`)。CLI は `result.Err` をそのまま返す(`org.go:953-954`)ので、二重にも欠けにもならない。`rt.Send(` を呼ぶのは send のコマンドだけである(`org.go:902`)。`%w` で包むので `errors.Is` の鎖も保たれる。新しい org 層のテスト `TestOrgSend_DryRun_AppendFails_ErrorStartsWithTheSendPrefix`(`verbs_test.go:965`)が接頭辞が 1 回であることと EISDIR を固定する
- 新しい問題は見つからなかった。気づいたのは名前だけである。CLI のテスト `TestOrgSend_AnErrorWithoutTheSendPrefixGetsItOnce`(`org_watch_gate_test.go:349`)は、CLI が接頭辞を足さなくなったあとも「GetsItOnce」の名前のまま残っている。コメントは直してあり、期待も正しいので、直さなくてよい

### Static analysis(cycle 1)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | rc 0 | golang の verifier は `gofmt: ok`、`go vet` は出力なし、`golangci-lint` は `0 issues.`、staticcheck は出力なし。shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の guard、`check-pipeline-sync.sh`、`check-template-purity.sh`、tech-debt の plan 参照の検査、branch の secret scan(dc41c4f5..9823d97d、clean)も通った |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、KNOWN_DIFF 5(`docs/quality/quality-gates.md` を含む。この PR は両面に同じ行を足した) |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill(s) in lock-step |
| `cmp` で mirror の一致 | 一致 | `/org` skill の 4 面、`agent-messaging.md` の 2 面 |
| `./scripts/plan-visual.sh digest <plan>` | de5ca6b7e6f0 | `- Approved:` 行と一致 |
| `go build ./cmd/ralph`(一時の場所へ) | 成功 | `ralph org start/spawn/send/watch --help` と `ralph status --help` の文を読むためと、実機確認のため |

### Documentation drift(cycle 1)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| D1 `docs/tech-debt/README.md:183`(self-review L5 (a)) | いいえ | F-7 の文が、この PR で消えた `newOrgSpawnRuntime` を挙げている(「`newOrgSpawnRuntime` (`internal/cli/org.go`) copies the first two steps of `newOrgRuntime`」)。写しは今、spawn・`start <task>`・`start --plan` の 3 か所の `org.ResolveOrgStateDir` の呼び出し(`internal/cli/org.go:549`、`:725`、`:822`。self-review の 550・726・823 から 1 行ずつずれている)と `newOrgSpawnRuntimeAt` の guard にある。文をこの形に書き直す。Trigger はまだ起きていないので、行は開いたままでよい |
| D2 `docs/tech-debt/README.md:205`(L5 (b)) | いいえ | (b) が `newOrgSpawnRuntimeAt` の呼び手を「(spawn and `start --plan`)」と書く。今は `start <task>` も呼ぶので、呼び手に足す |
| D3 `docs/tech-debt/README.md:202`(L5 (c)) | いいえ | 閉じた行の Impact の欄に「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」が残る。印が見張り自身の deadman を持つようになったので、この残りも閉じている。取り消し線を引くか、RESOLVED の注に「この残りも閉じた」と足す |
| D4 `README.md:256`(L6) | いいえ | 「Either way the watch runs detached in the ledger's main worktree root, so without `--config` it reads that root's `ralph.toml`.」に、台帳に main worktree がないとき(repo の外を指す `--state-dir`)は打った場所で動く、という条件がない。「in the ledger's main worktree root (when the ledger has one)」のように足す。help(`internal/cli/org.go:693-694`、`internal/cli/org_watch.go:63`)と skill はこの条件を書いている |
| D5 `README.md:125`(新しく見つけた) | いいえ | `ralph status` の行が `org watch:` の行を「running, stale, or not running」と書いている。この PR で `stopping` と `running (--once)` が増え、`--json` には `once` が足された(`internal/cli/status.go:172-178`、`:434`)。「running (with `(--once)` for a `--once` watch), stopping, stale, or not running」のように直す。同じ README の 256 行は stopping を書いている |
| D6 `.claude/skills/org/SKILL.md:176`、`:177`、`:585`(任意) | ほぼ合う | 「ロックを持つ見張りが stale なら、起動せず最大 10 秒 running になるのを待ち、ならなければ断る」は、待つ間に持ち主が終わった(`not_running` になった)ときに起動へ進む動き(`org_watch.go:428-446`)を書いていない。直すなら 4 面とも同じに直し、`scripts/sync-skills.sh` で揃える。直さなくても誤った操作には導かない |
| D7 `internal/cli/org_watch.go:558-559`(任意、コードのコメント) | ほぼ合う | `runOrgWatchStop` の doc の「running and stale are handled alike」に stopping がない。`--stop` は `not_running` 以外をすべて同じに扱う(`:563-566`)。「running, stopping, and stale」にする |
| `/org` skill の 4 面 | 合う | `spawn`・`send`・`watch`・`start` の行(`:168`、`:169`、`:176`、`:177`)、`start` が拒否するもの(`:584-588`)、(A) 現セッション昇格(`:681-683`)、(B) headless(`:688-689`)、長時間運用の項(`:780-814`)が新しい動きと合う。4 面はバイト一致 |
| `README.md:235`、`:246` | 合う | 例の先頭に `ralph org watch --all --detach` を足し、spawn と TASK が断られることと start が起動することを書いた |
| `AGENTS.md:90` | 合う | repo map に `watchgate.go` を足した。「refuses spawn, start, and TASK sends」の start は、起動できないときに断る、の意味で読める |
| 仕様の注 `docs/specs/2026-10-07-org-multi-org-director.md:64` | 合う | F4 と F7 の読み替えを含め、上の「仕様 FR-6 との対応」のとおりコードと合う。前半の注(`:63`)の 3 状態の説明には「後半で stopping に変えた」の括弧書きが付いている。仕様のチェックボックスは FR-1〜FR-12 のどれにも付いていないので、FR-6 が未チェックなのは揃っている |
| `.claude/rules/ralph/agent-messaging.md:96-100`(と template) | 合う | TASK の拒否と raw の読み方。2 面はバイト一致 |
| `docs/quality/quality-gates.md:80`(と template) | 合う | 「Cross-org watch gate」の行。両面に同じ行がある |
| tech-debt の 201・204 (e)・205・206 と新しい行 | 合う(D1〜D3 を除く) | 201 は `:201-202`、205 は `:206-207`、206 は `:208-209` で閉じ、204 (e) は `:205` の中で取り消し線と RESOLVED の注が付いた。見張りが途中で落ちたときの穴は `:210` に新しい行がある。`:210` は計画を `docs/plans/active/` のパスで指すが、/pr の `scripts/archive-plan.sh` が `docs/tech-debt/README.md` のこのリンクを書き換えるので、ずれではない |
| `ralph org start/spawn/send/watch --help`、`ralph status --help` | 合う | ブランチのビルドで読んだ。start は起動と 10 秒の待ちと拒否の 3 状態、spawn と send は `--dry-run` も断ること、watch は `--detach` の作業ディレクトリ・環境・終了コード、status は stopping と `(--once)` を書く。`TestOrgStartSpawnSendHelp_SayWhatTheWatchGateDoes`(`org_start_watch_test.go:499`)が固定する |
| leader の雛形 `internal/org/prompts/leader.md:236-248` | 合う | AC11 の行 |

D1〜D5 は /sync-docs で直す。D6 と D7 は任意。

### Observational checks(cycle 1)

ブランチのビルドで、repo の外の一時の台帳(`--state-dir`)を使って確かめた。出力は evidence の log にある。

- 見張りなし: `spawn --dry-run`、型付きの TASK と raw の `  TYPE:  TASK` の `send --dry-run` は rc=1。エラーは `--state-dir` を付けた `ralph org watch --all --detach` と、新しい org なら `ralph org start` を案内した。raw の `type: task` と QUESTION は rc=0。manifest には通った 2 件の記録だけが残った
- `watch --all --detach` は `started … (pid N)`、2 回目は `already running (pid N)` で、どちらも rc=0。印は `deadman_minutes: 10` を持つ。そのあとの spawn と TASK の `--dry-run` は通った
- 動いている見張りの台帳で、2 つ目の `watch --all --once` は rc=1、stdout は 0 バイト、stderr は相手の pid を言った
- `--stop` は `stopped ralph org watch --all (pid N)` で、印は stopped
- `--detach` で起動した見張りを kill -9 すると、`ralph status` は `org watch: not running (a running mark is left from pid N, …)` になり、spawn は not_running の文で断られた。続く `start <task>` は新しい見張りを起動して stderr に pid と watch.log を出し、そのあと envelope(未知の model)で断られた。見張りは start のあとも running のまま残り、2 回目の start は何も起動しなかった
- python の flock でロックを持ち、1 時間前の印を置いた台帳では、spawn が 2 秒、`--detach` が 10 秒待ってから stale の文で rc=1 になった
- `--once` の見張りの最後の印には `once: true` があった
- 終わったあと、一時の ralph のプロセスは残っていない(`pgrep` で確認)

気づいたこと(コードの文言、LOW。直すかは任せる): `--detach` の拒否の文は、`--state-dir` を明示しても、案内するコマンド(stale の `ralph org watch --stop`、stopping と `--once` の `run ralph org watch --all --detach again`)に `--state-dir` を付けない(`org_watch.go:541-551`)。検査の文(`WatchGateError`)は付ける。stale の文は 6a からこの形で、stopping と `--once` の文はこの PR で同じ形に足された。実機の stale の拒否でも、spawn の文には付き、`--detach` の文には付かなかった

### Coverage gaps(cycle 1)

- テストは走らせていない(/test の担当)。上の判定は、テストがあり、読んだ範囲で期待が AC と合う、という意味である
- `ensureOrgWatch` が stale を待つ間に持ち主が終わって起動へ進む分岐(`org_watch.go:428-446`)を固定するテストはない。この分岐を「断る」に変える mutant は、今のテストでは落ちない見込みである(読んだだけで、走らせてはいない)
- 本物の見張りが後始末(stopping)をしている間に spawn・TASK・start を打つテストはない。印の書き手(`watch_all_test.go:1222`)と、stopping の印に対する検査(cli のテスト)を別々に固定している
- 実機で見ていないもの: 後始末の間と `--once` が動いている間の拒否(窓が短く再現できなかった)、start が stale の台帳で断ること(`--detach` で同じ経路を見た)、`start --plan` の起動、並行の `--detach` と start、main worktree の中の台帳での子の作業ディレクトリ、Linux での動き
- 前の版の見張り(印に `once` も `deadman_minutes` もない)と台帳を共有したときの動きは見ていない。計画の Risks と Rollout のとおり、release のノートで扱う
- 計画の Verify plan の「実機確認を `docs/evidence/` に」は、上の確認を evidence の log(gitignore の対象)に残しただけで、コミットする記録は作っていない
- 計画の受け入れ条件のチェックボックス(AC1〜AC15)はまだ付いていない。前の PR と同じく、検証のあとに付ける段取りと読んだ

### 判定(cycle 1)

pass

- Verified: AC1〜AC11 と AC13〜AC15 は、コードとテストの読み合わせで満たす。F1〜F10 はコードと一致し、仕様 FR-6 の残りの 4 項目と受け入れ条件 107 行(F7 の読み替え)を満たす。Progress に記録されたずれと、記録のない小さなずれ 2 つはどれも妥当である。bda15444 は文言とエラーの包み方だけで、新しい問題はない。静的解析(changed scope)、`check-sync.sh`、`check-skill-sync.sh` は rc 0。見張りなしの拒否、`--detach` の起動と `already running`、banner を出さない 2 つ目の見張り、kill のあとの start の起動、stale の待ちと拒否、印の `deadman_minutes` と `once` は実機でも確かめた
- Partially verified: AC12(D1〜D5 の文書が古い。/sync-docs で直す。D6 と D7 は任意)、AC8(座席の `wait` と `inbox show/ack/resolve/notify` は配線がないことを grep で確かめただけ)
- Not verified: テストの実行、後始末の間と `--once` の間の拒否の実機での確認、start の stale の拒否と `start --plan` の起動の実機での確認、並行の起動、Linux
