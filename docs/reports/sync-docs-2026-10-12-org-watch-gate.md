# sync-docs report: org-watch-gate

## Latest run: cycle 3 (HEAD 288a807b)

- Date: 2026-10-12(JST)
- Plan: `docs/plans/active/2026-10-12-org-watch-gate.md`(承認済み、digest `de5ca6b7e6f0`。計画は触っていないので値は変わらない)
- Pipeline cycle: 3(上限は `RALPH_STANDARD_MAX_PIPELINE_CYCLES=3` に上げてあり、これが最後の cycle)。`cycle-count.json` は cross-review の規則で 2 のままなので、cycle の番号はこの report と insight event の `--cycle 3` で付けた。差分は base `dc41c4f5` から branch HEAD `288a807b`(feat/org-watch-gate)まで。cycle 2 の report は末尾の「Appendix: cycle 2」に、小見出しを 1 段下げただけで残してある
- 先行 report:
  `docs/reports/self-review-2026-10-12-org-watch-gate.md`(Merge 可。cycle 3 の LOW は C3-1・C3-2)、
  `docs/reports/verify-2026-10-12-org-watch-gate.md`(pass。cycle 3 の「Documentation drift」の D11〜D15)、
  `docs/reports/test-2026-10-12-org-watch-gate.md`(pass。cycle 3)
- 項目の番号: 依頼の D11〜D15 は、verify cycle 3 の「Documentation drift」の表の番号と同じ。D14 は self-review の C3-2 だけを書く(C3-1 は入れない)
- 内容の commit: この report と insight event を含む 1 つの commit(`docs: sync docs for org-watch-gate (cycle 3)`)

### Summary

cross-review cycle 2 の直し(8d8ddbd2)は、`--detach` と start が起動する見張りの子の環境に 1 つ手当てを足した。子の作業ディレクトリを main worktree のルートに移したので、相対の `RALPH_ORG_AGMSG_HOME`(空か未設定なら、明示した `--config` の相対の `agmsg_home`)を、打った場所から絶対パスにして渡す(`orgWatchChildAgmsgHome`)。この挙動を書いた文書がなかったので、README、`/org` skill の 4 面、tech-debt の 205 行目に D11〜D13 で足した。仕様の 6 段目の後半の注にも、任意の D15 として一節足した。

self-review の C3-2(`org.CommandWithStateDir` が `org.InboxCommand` の `--state-dir` を足す 4 行の写しで、引用の規則が `internal/org` に 2 か所ある)は、最後の cycle で見つけた LOW で、直すと全段のやり直しになるので、D14 として tech-debt の新しい行に書いた。コードと `--help` の文言は触っていない。

### Changes made

| File | D | Change |
|------|---|--------|
| `README.md:256` | D11 | 「so without `--config` it reads that root's `ralph.toml`.」のあとに、「A relative `RALPH_ORG_AGMSG_HOME`, or else (the variable empty or unset) the relative `agmsg_home` of the `--config` given, is passed to the watch made absolute from where the command was typed.」を足した。条件はコード(`orgWatchChildAgmsgHome`、`internal/cli/org_watch.go`)と同じ順で、env が空でなければ env、空なら明示した `--config` の値 |
| `.claude/skills/org/SKILL.md:176`(動詞表の `watch`)と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行 | D12 | `HERDR_PANE_ID`・…・`HERDR_ENV` を外す文のあとに、「main worktree のルートで動くときは、相対の `RALPH_ORG_AGMSG_HOME`(空か未設定なら、渡した `--config` の相対の `agmsg_home`)は、打った場所から絶対パスにして渡される。」を足した。verify の提案の文に「main worktree のルートで動くときは」を足したのは、台帳に main worktree がなく打った場所で動く子は、値をそのまま使うため |
| `.claude/skills/org/SKILL.md:177`(動詞表の `start`)と 3 つの写し | D12 | 括弧の「start に渡した `--config` は絶対パスにして渡す。」のあとに、「相対の `RALPH_ORG_AGMSG_HOME`(空か未設定なら、渡した `--config` の相対の `agmsg_home`)も、打った場所から絶対パスにして渡す。」を足した(verify は任意とした。`watch` の行と揃えた) |
| `docs/tech-debt/README.md:206`(205 行目の行を閉じた RESOLVED の注) | D13 | 「A ledger with no main worktree … keeps the caller's directory, as plan F6 chose.」のあとに、子の作業ディレクトリが変わるときは、相対の agmsg home(`RALPH_ORG_AGMSG_HOME`、なければ明示した `--config` の `agmsg_home`)を、もう子がそこで動かないので打った場所から絶対パスにして渡すことを足した(`orgWatchChildAgmsgHome`、cross-review cycle 2 of feat/org-watch-gate、commit 8d8ddbd2)。固定するテストの文には `TestOrgWatchDetach_ARelativeAgmsgHomeIsPassedAbsoluteToTheRelocatedChild` と `TestOrgStart_ARelativeAgmsgHomeIsPassedAbsoluteToTheWatch` を足した |
| `docs/tech-debt/README.md:207`(205 行目の行の RESOLVED の文) | D13 | 「without the herdr pane variables」のあとに「, with a relative agmsg home made absolute」を足した |
| `docs/tech-debt/README.md:212`(新しい行、末尾) | D14 | `org.CommandWithStateDir`(`internal/org/watchgate.go`)が `org.InboxCommand`(`internal/org/escalate.go`)の `--state-dir` を足す 4 行の写しで、`shellQuote` の引用の規則が `internal/org` に 2 か所あること。実行時の害はなく、片方の引用を変えるともう片方が古いまま残る。cycle 3(最後の cycle)で見つけた LOW で、直すと全段のやり直しになるので書くだけにした。Trigger は、次に `internal/org` の `--state-dir` の引用を変えるとき、または `InboxCommand` か `CommandWithStateDir` を触るとき(`InboxCommand` の本体を `CommandWithStateDir("ralph org inbox "+verb+" "+id, stateDir)` を返す形にする)。Related は計画の active の path、self-review の C3-2、3fb0fd1b。形は 211 行目の行と同じ(5 列、plan は `docs/plans/active/...`) |
| `docs/specs/2026-10-07-org-multi-org-director.md:64`(6 段目の後半の注) | D15 | `ralph org start` が起動する子の性質の列(main worktree のルートで動く、`HERDR_*` の 4 つを外す、`--config` を絶対パスで受け取る)に、「相対の agmsg home(`RALPH_ORG_AGMSG_HOME`、空か未設定なら渡した `--config` の `agmsg_home`)を打った場所から絶対パスにして受け取る」を足した |
| `docs/insights/events/2026-10-12-org-watch-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-watch-gate`、`--cycle 3`、verdict pass、findings 0)。既存のファイルへの追記で、新しいファイルはできていない |
| この report | - | cycle 3 の節を先頭に足し、cycle 2 の節を「Appendix: cycle 2」にした |

skill の 4 面は、`.claude/skills/org/SKILL.md` を編集し、`./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つには `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md` を `cp` で写した(sync-skills.sh は `templates/base/` を作り直さない)。4 面とも sha256 は `ee98fd29…`。

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `RALPH_ORG_AGMSG_HOME` と `agmsg_home` を挙げる文書(`git grep` で `internal/`、`docs/reports`、`docs/plans`、`docs/evidence` を除いた全部) | README、skill の 4 面、仕様の注、tech-debt が今回の更新の対象。ほかは `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh`(シェル側で `RALPH_ORG_AGMSG_HOME` を export しない理由のコメント)と `templates/base/ralph.toml:57-60`(`agmsg_home` の説明)だけで、相対の値の扱いには触れておらず、8d8ddbd2 でも崩れない。変更なし |
| `AGENTS.md`(repo map の `internal/org/`)、`docs/quality/quality-gates.md` と `templates/base/` の写し、`.claude/rules/ralph/agent-messaging.md` と写し、`internal/org/prompts/leader.md`、`docs/architecture/`、`docs/recipes/`、`.codex/` | 8d8ddbd2 は子の環境の話で、検査の位置、TASK の判定、印、status、座席のプロトコルのどれの記述も崩していない。cycle 2 の同じ面の確認から変わらない。変更なし |
| `README.md:256` のほかの文、`ralph status` の行(`README.md:125`) | 8d8ddbd2 は起動、`watch.log`、`--stop`、`org watch:` の状態に触れていない。変更なし |
| `watch --help`(`internal/cli/org_watch.go:61-66`)、start の help(`internal/cli/org.go:691-696`) | 外す変数と動く場所を言うだけで、環境がそれだけとは言わないので今も正しい。verify の勧めに従い、`internal/` の変更になるので直さなかった |

### Left open

- `--help` の文言(D15 の後半)は直していない。上のとおり正しさは崩れておらず、直すと `internal/` の変更になる
- README と skill の文(D11・D12)は、台帳に main worktree がなく子が打った場所で動く場合の扱いを書かない。そのとき子は値に触れず、相対の値は同じ場所を指す。skill の文は「main worktree のルートで動くときは」で条件を付けたが、README の文は verify の提案どおりに書いた
- self-review の C3-1(3 つのコメントが「cross-review cycle 2 #1」と書くが、triage の報告は cycle ごとに書き直される)は、依頼どおり tech-debt の行に入れていない。verify の記述では、cycle 3 の triage が cycle 2 の内容を付録に残して指し先を保つ
- 子が絶対パスの home で agmsg の script を呼ぶところ、start の経路の実機、linked worktree とシンボリックリンクの下からの起動は、verify の Coverage gaps のまま開いている。文書では埋まらない

### Verification

- `./scripts/check-sync.sh`: PASS(DRIFTED 0)。`./scripts/check-skill-sync.sh`: 13 skill(s) in lock-step
- `./scripts/run-verify.sh`(worktree のルート、この変更を作業ツリーに置いた状態): rc 0、「All verifiers passed.」。`go test` はテストのある 8 パッケージすべてが ok(一部は cached)で、既知の不安定なテスト(`TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)は落ちず、再実行はしていない。末尾の branch secret scan(`dc41c4f5..288a807b`)は clean で、commit 済みの範囲を読むので、この commit は含まない。この節の run-verify の行は run-verify のあとに書いたため、run-verify の対象には入っていない

## Appendix: cycle 2 (HEAD 783c1a4b)

- Date: 2026-10-12(JST)
- Plan: `docs/plans/active/2026-10-12-org-watch-gate.md`(承認済み、digest `de5ca6b7e6f0`。計画は触っていないので値は変わらない)
- Pipeline cycle: 2(上限 2)。差分は base `dc41c4f5` から branch HEAD `783c1a4b`(feat/org-watch-gate)まで。cycle 1 は sync-docs に届かなかったので、この report が最初で、cycle 1 の節はない
- 先行 report:
  `docs/reports/self-review-2026-10-12-org-watch-gate.md`(Merge 可。cycle 2 の LOW は C2-1〜C2-4、付録から開いたままの L5・L6)、
  `docs/reports/verify-2026-10-12-org-watch-gate.md`(pass。cycle 2 の「Documentation drift」の D1〜D10)、
  `docs/reports/test-2026-10-12-org-watch-gate.md`(pass。cycle 2 の Test gaps)
- 項目の番号: 依頼の D1〜D10 は、verify cycle 2 の「Documentation drift」の表の番号と同じ。依頼の D8 は self-review の C2-4 と同じ項目で、D9 の次に D10 がある
- 内容の commit: この report と insight event を含む 1 つの commit(`docs: sync docs for org-watch-gate (cycle 2)`)

#### Summary

必須の D1〜D5・D8・D9 と、任意の D6・D7・D10 をすべて直した。古くなっていた文は 2 種類ある。1 つは、この PR が変えたコードより前の形を書いた tech-debt の文(消えた `newOrgSpawnRuntime`、`newOrgSpawnRuntimeAt` の呼び手、閉じたのに「still open」が残る行、`--detach` の同時起動の結果)で、もう 1 つは、台帳の main worktree のルートと `ralph status` の状態の書き方が `--help` と `status.go` より少なかった README である。`/org` skill の `watch`・`start` の行は、同時に打ったときに自分の子の終わりを待ってから返ることと、stale の持ち主が待つ間に終われば起動へ進むことを足した。

D8 は、計画の Progress が「tech-debt に書く」と決めた残る穴の新しい行で、`docs/tech-debt/README.md` の末尾に足した。self-review C2-1 の現れ方(3fb0fd1b で直した、止めたあとに判定し直す流れ)も同じ行に書いた。

#### Changes made

| File | D | Change |
|------|---|--------|
| `docs/tech-debt/README.md:183`(「Code-shape findings of org-limits-reserve」) | D1 | (c) F-7 の文を、消えた `newOrgSpawnRuntime` から今の形に書き直した。`ralph org spawn`、`start <task>`、`runOrgStartPlan`(`start --plan`)の 3 つがそれぞれ `org.ResolveOrgStateDir` を呼び、`newOrgSpawnRuntimeAt` が `guardLegacyOrgStateDir` の呼び出しを繰り返す。Trigger の欄に「F-7 の trigger(`newOrgRuntime` の変更)はこの PR で起きていないので、F-7 は開いたまま」と足し、Related に self-review の L5 を足した。行番号ではなく関数名で書いた |
| `docs/tech-debt/README.md:205`(「Small leftovers of org-cross-watch」) | D2 | (b) の `newOrgSpawnRuntimeAt` の呼び手を「(spawn and `start --plan`)」から「(spawn, `start <task>`, and `start --plan`)」に直した |
| `docs/tech-debt/README.md:202`(閉じた「`ralph status` and the cross-org watch can disagree」) | D3 | Impact の「still open: a cycle that takes longer than the deadman of `./ralph.toml` but not the watch's own」に取り消し線を引き、「closed 2026-10-12 in feat/org-watch-gate: `ProbeWatch` judges by the `deadman_minutes` the mark carries」と足した。同じ文の後ろの「show a hung watch as `running` for longer than the watch's own deadman」も同じ変更で閉じるので、「(closed by the same change)」と足した |
| `docs/tech-debt/README.md:208`(206 を閉じた RESOLVED の注) | D9 | 「as of S3 no live run of several concurrent `--detach` calls had been repeated」を書き直した。/test cycle 1 が実機で後退を見つけたこと(`already running` を返したあとも負けた子がロックを待ち、`--stop` で勝った見張りが止まると、頼みが消えたあとに子がロックを取って動き続け、`--stop` は 30 秒で時間切れ。base は見張りが残らない)、fdfb5b23 で `--detach` と start が自分の子の終わりを(`org.DefaultWatchLockWait` + 1 秒、`orgWatchChildExitWait` まで)待つようにしたこと、それを固定するテスト 2 本、流し直し(verify と test の cycle 2)で 8 本の同時 `--detach` のあとの `--stop` が 0.4〜1.2 秒で通り見張りが残らなかったこと、残る窓(別の端末の `--stop`)が D8 の行であること |
| `docs/tech-debt/README.md:211`(新しい行、末尾) | D8 | 別の端末の `--stop` が、ロックを待つ側(`--detach` と start が待つ自分の子、またはロックを待つ foreground の `watch --all`)に追い越される穴。止まった見張りが `watch-stop.json` を消してロックを放し、待っていた側がロックを取って動き続ける。`--stop` は 30 秒で終了コード 1。base の foreground にもある。C2-1 の現れ方(止めたあとに判定し直し、見張りがなければ「the watch (pid N) was stopped while another watch held <lock>, and that one has ended too, so no watch runs」と断る)を同じ行に書いた。テストも実機の確認もないこと、直さなかった理由(退けた 2 案)、Trigger、Related(計画の Progress、test と self-review の report、evidence の `takeover.sh`、fdfb5b23、3fb0fd1b)を書いた |
| `README.md:256` | D4 | 「in the ledger's main worktree root」に「(when the ledger has one; a `--state-dir` outside a repository has none, and the watch then runs where it was typed)」を足した。`--help`(`internal/cli/org.go` の start、`internal/cli/org_watch.go` の `--detach`)の「when it has one」と同じ条件 |
| `README.md:125` | D5 | `ralph status` の行の `org watch:` の状態を「running, stale, or not running」から「running (`running (--once)` for a one-cycle watch), stopping while it drains, stale, or not running」に直し、`--json` の `org_watch` に `once` が付くことを足した(`statusOrgWatchJSON.Once`、`internal/cli/status.go`) |
| `.claude/skills/org/SKILL.md:176`(動詞表の `watch`)と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行 | D6・D10 | 「(同時に打った別の `--detach` か start の見張りが先にロックを取った場合も)、何も起動せず `already running`」を 2 つに分けた。すでに動いていれば何も起動せず `already running`。同時に打った別の見張りが先にロックを取った場合は、自分の子を起動してから、その子がロックを断られて終わるのを待ち(最大で約 3 秒)、同じく `already running` を出して終了コード 0 で返る。stale の持ち主を待つ文に「待つ間に持ち主が終われば、見張りを起動して進む」を足した |
| `.claude/skills/org/SKILL.md:177`(動詞表の `start`)と 3 つの写し | D6・D10 | 「同時に打った別の start か `--detach` の見張りが先にロックを取っても、running になれば進む」に、「start が起動した子はロックを断られて終わるので、その終わりを最大で約 3 秒待ってから進む」を足した。stale の文に「待つ間に持ち主が終われば、見張りを起動して進む」を足した |
| `.claude/skills/org/SKILL.md:588-590`(start の拒否の一覧の「横断の見張りを使えないとき」)と 3 つの写し | D6 | 末尾に「stale の見張りを待つ間にロックの持ち主が終わったときは、断らずに見張りを起動して進む。」を足した(折り返しは 1 行 70 幅以内) |
| `internal/cli/org_watch.go:608-610`(`runOrgWatchStop` の doc) | D7 | 「running and stale are handled alike」を「running, stopping, and stale are handled alike」に直した。コメントだけで、`gofmt -l internal/cli/` は空 |
| `docs/insights/events/2026-10-12-org-watch-gate.jsonl` | - | `sync_docs` の event を 1 行足した(`--slug org-watch-gate`、cycle 2、verdict pass、findings 0)。既存のファイルへの追記で、新しいファイルはできていない |
| この report | - | 新しいファイル |

skill の 4 つの写しは `./scripts/sync-skills.sh` で `.agents/skills` を作り直し、`templates/base/` の 2 つには `.claude/skills/org/SKILL.md` と `.agents/skills/org/SKILL.md` を `cp` で写した(sync-skills.sh は `templates/base/` を作り直さない)。`./scripts/check-sync.sh` は PASS(DRIFTED 0)、`./scripts/check-skill-sync.sh` は 13 skill で一致。

#### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `AGENTS.md:90`(repo map の `internal/org/`)、`docs/quality/quality-gates.md` と `templates/base/` の写し、`.claude/rules/ralph/agent-messaging.md` と写し、`docs/specs/2026-10-07-org-multi-org-director.md` の 6 段目の後半の注 | cycle 2 のコードの変更(fdfb5b23 の子の終わりの待ち、3fb0fd1b の止めたあとの判定、`CommandWithStateDir`、47635690 のテスト 1 本)は、検査の位置、TASK の判定、印、status のどれの記述も崩していない。変更なし |
| `internal/org/prompts/leader.md`(leader の雛形) | spawn か TASK の送信が見張りなしで断られたときの手順(見張りを leader が自分で打たず、受信箱に上げて返事を待つ)を書いていて、コードと合っている。変更なし |
| `README.md` の `ralph org start` と `--detach` の説明(256 行目のほかの文) | 起動、`watch.log`、`--stop` の説明は `--help` と合っている。256 行目の条件だけ直した |
| `docs/architecture/`、`docs/recipes/`、`.codex/` | `watchgate`、`watchalive`、`org watch` を挙げている行はなく、変更なし |
| 古い名前 `orgWatchDetachDeadline`、`defaultWatchLockWait`、`newOrgSpawnRuntime`(関数名として) | `git grep` で、コードと現行の文書(report と計画の履歴を除く)に残っていない。tech-debt の F-7 の文に「before feat/org-watch-gate」と書いた参照だけが残る |

#### Left open

- tech-debt の 208 行目(RESOLVED の注)の「while a watch holds the lock and has not written its first mark (stale), both wait up to 10 s」には、待つ間に持ち主が終われば起動へ進むことを足していない。注は閉じた行の履歴で、skill と code の doc が現行の動きを書いているため
- test report(`docs/reports/test-2026-10-12-org-watch-gate.md`)の Test gaps に「本番の `orgWatchChildExitWait` の値(mutant K8)」が開いたまま残っている。計画の Progress は 47635690 の `TestOrgWatchChildExitWait_OutlastsTheChildsLockWait` で閉じたと書く。依頼の範囲外なので report は直していない
- 別の端末の `--stop` が追い越される穴(D8)は、コードもテストも実機の確認もない。tech-debt の行が唯一の記録である

#### Verification

- `gofmt -l internal/cli/`: 出力なし
- `./scripts/check-sync.sh`: PASS(DRIFTED 0)。`./scripts/check-skill-sync.sh`: 13 skill(s) in lock-step
- `./scripts/run-verify.sh`(worktree のルート、この変更を作業ツリーに置いた状態): rc 0、「All verifiers passed.」。`go test` はテストのある 8 パッケージすべてが ok で、既知の不安定なテスト(`TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)は落ちず、再実行はしていない。末尾の branch secret scan(`dc41c4f5..783c1a4b`)は commit 済みの範囲を読むので、この commit は含まない。この report の Verification の文は run-verify のあとに書いたため、run-verify の対象には入っていない
