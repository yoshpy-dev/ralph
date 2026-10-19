# Verify report: org-director-wait

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait.md`(8b-2b、director の受信箱の wait の印と見張りの停止の知らせ)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff e38b89a9...HEAD`(HEAD c13455e3、44 ファイル、+5376/-465)。AC1〜AC9、計画の Verify plan(FR-6、FR-9、FR-10、8b-2a の確かめと `director-alive.json`、`internal/org/watch.go`、文書のずれの 4 項目)、Design decisions D1〜D8、Non-goals、Progress の記録。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の 8b-2b の注(90 行)。self-review の報告(「Fixes review (cycle 1)」の節を含む)と evidence 2 つ
- Evidence: `docs/evidence/verify-2026-10-19-org-director-wait.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-19-041544.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)
- Cycle 2: 下の「Cycle 2 (8a33364b..89b81a2a)」の節(HEAD 89b81a2a)。`## Verdict` は、その節を含めたブランチ全体の判定に置き換えた。この行と `## Verdict` の節のほかは cycle 1 の記録のまま

## Spec compliance

テストの実行は `/test` の範囲なので、コードとテストの中身を読み、テストがあることと、そのテストが何を固定しているかを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 フラグ | 満たす | `--inbox` のない `--director` は RunE の最初で `org: wait: --director needs --inbox` として断る(`internal/cli/org.go:1286`)。`d<N>` でない id は `org.CheckDirectorWaitID`(`internal/org/director_wait.go:213`)が、`ResolveOrgStateDir` と `ralph.toml` の読みより前に断る(`internal/cli/org_wait_director.go:49`)。今の登録が違う、ない、stopped のときは `directorWaitCheckCurrent`(`director_wait.go:301`)が `ErrDirectorWaitNotCurrent` を包み、`describe()` で今の登録を名指す。この判定は `begin` より前なので、印も lock のファイルも作らない(`:282`〜`:290`)。テストは `TestOrgWaitDirector_RefusesItsFlagsBeforeReadingAnything`、`..._RefusesAnIDThatIsNotDN`、`..._RefusesABadIDBeforeItLoadsTheConfig`、`..._ARegistrationThatIsNotCurrentExits5`(`internal/cli/org_wait_director_test.go:184`〜`:286`)、`TestDirectorWaitCheckCurrent`、`TestWaitInboxForDirector_RefusesBeforeItWaits`、`TestCheckDirectorWaitID_AcceptsOnlyDN`(`internal/org/director_wait_test.go:294`、`:357`、`:429`) |
| AC2 印 | 満たす | 印の型は D2 の 10 field(`director_wait.go:125`〜`:136`)。書くのは `begin`(`:386`)と `save`(`:600`)の 2 か所で、どちらも `director-wait.lock` の `withFileLock` の中で `writeJSONFileAtomic` を打つ。`save` を呼ぶのは 30 秒ごとの `refresh`(`:562`)、見張りが受け付けるようになったときの `probeAgain`(`:545`)、終わりの `end`(`:593`)。終わり方は `DirectorWaitEnd*`(`:57`〜`:74`)で、items は件を返し(0)、timeout と error は誤り(1)、watch は `ErrDirectorWaitNoWatch`(4)、not_current は `ErrDirectorWaitNotCurrent`(5)、signal は CLI が同じ signal を自分に送り直す(`org_wait_director.go:76`〜`:79`)。書けない印は `sayWriteFailed`(`:659`)が 1 回だけ言い、wait は続く。テストは `TestDirectorWaitMark_JSONKeys`、`TestReadDirectorWait_MissingCorruptAndRoundTrip`、`..._AMarkThatCannotBeWrittenIsSaidOnce`、`..._AWaitWhoseFirstWriteFailedWritesOnceItCan`、`..._AnOpenItemEndsWithItems`、`..._TimesOutWithThePhraseAfterANote`、`..._ACancelledContextEndsWithSignal`、`..._AReadThatFailsEndsWithError`、CLI の `TestOrgWaitDirector_ASignalWritesItsEndThenRaisesItAgain` と `..._ASignalEndsTheProcessByIt` |
| AC3 持ち主 | 満たす | `owns`(`:631`)は、ディスクの `wait_id` が自分のとき、またはまだ 1 回も書けていない wait でディスクが `begin` の読んだ前の印のままのときだけ書かせる。持ち主でなくなると `sayLost`(`:646`)が 1 回言い、`lost` でそれ以降は書かない。`TestDirectorWait_AMarkTakenOverIsNotWrittenAgain`(`director_wait_test.go:572`)が偽の時計で A の開始、B の開始、A の書き直し、B の終わりを打ち、B の印と end が残ること、A の書き込みでファイルが byte で変わらないこと、A の stderr が 1 行であることを確かめる。2 つの wait を並べて走らせる形は `TestWaitInboxForDirector_TwoWaitsLeaveTheLaterOnesMark`(`:638`) |
| AC4 登録の変化 | 満たす | `refresh`(`:553`)が `MarkInterval`(既定は `DirectorWaitMarkInterval` の 30 秒、`:84`、`:357`)ごとに `director.jsonl` を読み直し、id、stopped、`attached_at` の違い(`:313`)で `not_current` を書いて `ErrDirectorWaitNotCurrent` を返す。テストは `TestWaitInboxForDirector_ARegistrationThatChangesEndsTheWait`(`director_wait_watch_test.go:346`)。実機の (e) は `--replace` のあと、wait の開始から 30 秒後の読み直しで 5 を返した |
| AC5 見張り | 満たす | `probeAgain`(`:529`)は 5 秒ごとに見張りを読み、受け付けていた見張りが受け付けなくなると `noWatch` を立てる。`loop` はもう 1 回受信箱を読んでから `endNoWatch`(`:568`)に進むので、open の件が先になる(`:465`〜`:471`)。`begin` は同じ登録の前の印の `watch_accepting` を引き継ぎ、true なら最初の回で 4、そうでなければ `note:` を 1 行出して待つ(`:426`〜`:432`)。stale は `probe`(`:508`)が settle のあいだ読み直す。文は `directorWaitWatchState`(`:695`)が D3 の 6 つを書き分け、印の値を `PrintableInboxText` に通す。テストは `..._AWatchThatStopsAcceptingWorkEndsTheWait`、`..._AnOpenItemWinsOverAWatchThatStopped`、`..._CarriesOverOnlyThisRegistrationsPreviousMark`、`..._AStaleWatchThatWritesItsMarkWithinTheSettle`、`..._AWatchThatStartsAcceptingWorkIsNoted`、`TestDirectorWaitWatchFacts_SaysOnlyWhatTheLockAndTheMarkShow`(`director_wait_watch_test.go:97`〜`:497`)。最後のテストは panic の跡(lock が外れて印が stopped)と stopped を書けなかった跡(lock が外れて印が running)を fixture で作り、`dirWaitCauseWords`(crash、died、killed、panic など、`director_wait_test.go:35`)がどの文にも入らないことを確かめる |
| AC6 終了コード | 満たす | `ExitCode`(`internal/cli/exitcode.go:24`)は nil が 0、drift が 3、`ErrDirectorWaitNoWatch` が 4、`ErrDirectorWaitNotCurrent` が 5、ほかが 1 で、`cmd/ralph/main.go:32` がその値で終わる。`TestExitCode`(`exitcode_test.go:14`)が包んだ sentinel も含めて固定する。`--director` のない wait は `runOrgWaitInbox`(`org.go:1327`)の前からの道のままで、`WaitInbox` の読みを `openInboxItems`(`internal/org/escalate.go:288`、`:311`)に出しただけ(誤りの包み方と nil の扱いは同じ)。既存の inbox の wait のテスト(`internal/cli/org_inbox_test.go:1270`〜`:1350`)と `internal/org/escalate_test.go` は差分がない。`TestWaitInbox_WritesNoDirectorWaitMark`(`director_wait_test.go:675`)と `TestOrgWaitInbox_WithoutDirectorHandlesNoSignal`(`org_wait_director_test.go:525`)が、印を書かないことと signal を扱わないことを固定する |
| AC7 表示 | 満たす | `readOrgDirectorWait`(`internal/cli/org_director_wait_line.go:55`)が今の登録に合う印(id と `attached_at`)だけを読み、`seen_at` が `2*org.DirectorWaitMarkInterval`(60 秒)より古い走っている wait を stale とする。`wait:` の行は `orgDirectorWaitLine`(`:94`)。昇格したセッションの `director:` の行には、`orgDirectorLine` の `case !reg.Headless()`(`internal/cli/org_director_report.go:174`)だけが `orgDirectorWaitClause`(`:108`)を足す。headless の行は `case r.watchChecks` か元の文のままで、`org_director_check_line_test.go` の変更はコメントだけ。`--json` は `wait,omitempty`(`org_director_report.go:65`)。text の値は `inboxField` で escape する。テストは `TestDirectorWaitLine_ShowsTheMarkOfAPromotedSessionsWait`、`..._AHeadlessRegistrationsLineIsUnchanged`、`..._DisabledNotAttachedAndOpenHeadless`、`..._EscapesTheReadError`、`..._EndToEndWithARealWait`、`..._HelpDescribesTheWait`(`org_director_wait_line_test.go:223`〜`:602`) |
| AC8 文書 | 満たす(LOW 1 件、情報 1 件) | skill の 4 面(director と org)は `cmp` が一致し、`check-skill-sync.sh` と `check-sync.sh` が通る。director skill の「wait で待つ」(`.claude/skills/director/SKILL.md:481`〜`:556`)と「してはいけないこと」(`:855`〜`:858`)が D6 を書く。雛形(`internal/org/prompts/director.md:42`〜`:46`)、help(`org.go:1262`〜`:1281`、`internal/cli/org_director.go:308`〜`:323`、`internal/cli/status.go:93`〜`:104`)、README(`:124`、`:125`、`:258`)、仕様の注(`:90`)、tech-debt(209、231 の (j)、247、新しい 248、250、306、307)、AGENTS.md(`:90`)が新しい動きを書く。tech-debt 253 行のきっかけが起きたのに行が変わっていない点を V1 に、README と recipe の言い方を V5 に書いた |
| AC9 隔離と実機 | 満たす(LOW 1 件) | 新しいテストの台帳はすべて `t.TempDir()` と `--state-dir`。herdr、osascript、gh、network を打つ `exec` はなく、`exec` はテストバイナリ自身の子(`org_wait_director_test.go:630`、`:632` の `sh -c "trap ''"`)と `true` だけで、signal はその子にだけ送る(`:655`〜`:677`)。evidence は計画の前の実験(`docs/evidence/director-wait-session-end-2026-10-19.md`)と D8 の (a)〜(h)(`docs/evidence/director-wait-livefire-2026-10-19.md:28`〜`:37`)を持ち、`$HOME` は `~` で、`/Users/` も token も pane の値もない。livefire の「Not checked here」が分割前のテストファイルを指す点を V2 に書いた |

仕様と Verify plan の確かめ:

- FR-6: `escalate.go` の変更は `WaitInbox` の読みの抽出だけで、`escalateDelivery` と期限の送り(`deadline.go`)は変わらない。4 の文と skill は、見張りがないあいだの escalate が人にすぐ届くこと(`inbox_no_watch`)と、それより前に director に残された件は見張りがまた動くまで期限で届かないことを言う
- FR-9: `internal/org/director.go` は差分がない。`internal/cli/org_director.go` の差分は report の移動と `director status` の help の 16 行だけで、attach と detach の RunE は変わらない
- FR-10: `internal/org/director_wake.go` は差分がない。`internal/cli/org_director_wake.go` はコメント 1 行
- 8b-2a: `internal/org/director_check.go` と `director_start_cleanup.go` は差分がなく、`director-alive.json` の書き手も読み手も変わらない(`internal/cli/org_director_check.go` はコメント 1 行)
- `internal/org/watch.go`: `git diff e38b89a9...HEAD -- internal/org/watch.go` は空
- 仕様の 8b-2b の注(90 行)は D1〜D7、実験、実機の evidence を書き、8b-2a の注(89 行)が 8b-2b に送った 2 つ(昇格したセッションの印、見張りの停止の知らせ)を受けている

30 秒の読み直し(self-review の Coverage gaps から引き継いだ点): help は "reads the registration again every 30 s and exits 5 at the first such read"(`org.go:1266`〜`:1268`)、skill は「30 秒ごとに登録を読み直し、... その読み直しで終了コード 5 で返る」(`SKILL.md:503`〜`:505`)と書く。実装では `loop` の `markTick`(`director_wait.go:452`、間隔は `newDirectorWaitRun` の既定の `DirectorWaitMarkInterval`、30 秒、`:84`、`:357`)が `markDue` を立て、次の回の `refresh`(`:553`)が `director.jsonl` を読み直す。最初の読み直しは `loop` が始まってから 30 秒後なので、登録が変わってから 5 で返るまで最大で 30 秒ほどかかる。help と skill の文はこの動きと一致する。実機の (e) も 30 秒後に 5 を返している。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | 選ばれた言語は golang。shellcheck、hook の `sh -n`、settings の `jq`、check-sync、check-pipeline-sync、check-skill-sync、template purity、tech-debt の plan 参照、gofmt ok、golangci-lint 0 issues、secret scan(e38b89a9..c13455e3、clean)。log は `docs/evidence/verify-2026-10-19-041544.log` |
| `go vet ./...`(go1.26.0) | pass(rc 0) | |
| `GOTOOLCHAIN=go1.25.8 go build ./... && GOTOOLCHAIN=go1.25.8 go vet ./...` | pass(rc 0) | go.mod の Go 1.25.8 |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass | DRIFTED 0、KNOWN_DIFF 5、TEMPLATE_ONLY 11 |
| `git diff e38b89a9...HEAD -- internal/org/watch.go` | 空 | |
| `git diff --check e38b89a9...HEAD` | pass(出力なし) | self-review L2 の末尾の空白は消えている |

全体の run-verify(lead が回したもので、ここでは回していない): 7368dee5 の最初の run は、`internal/org` が go test の既定の 10 分で `panic: test timed out after 10m0s`(601.926 s、`TestRegate_RefusedBeforeTheFetch` の途中)になり、`internal/cli` は 591.982 s で通った(`docs/evidence/verify-2026-10-19-030343.log:5052`、`:5055`、`:5159`)。`GOFLAGS=-timeout=30m` の再走は 03:56:54Z に 9f7a883e の木で始まり、"All verifiers passed" で終わった(`internal/cli` 260.195 s、`internal/org` 328.283 s、`docs/evidence/verify-2026-10-19-035654.log:5052`、`:5055`)。9f7a883e から HEAD までの差は self-review の報告、tech-debt、insight event だけで、コードは同じ。tech-debt 307 行の数字はこの 2 つの log と合う。

## Findings

CRITICAL、HIGH、MEDIUM はない。LOW 2 件は記録の文で、コードと出力は変わらない。

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | tech-debt 253 行(`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` のテストの穴 3 つ)のきっかけ「`/director` skill の受信箱の処理の次の変更、または `WaitInbox` の次の変更」がこの PR で起きたが、行は base と byte で同じで、払ったとも持ち越したとも書いていない。この PR は受信箱の処理の wait の段落(Codex の打ち直しの条件の箇条も含む)を書き直し(aba16ec2、c1d31e3f)、`WaitInbox` の読みを `openInboxItems` に出した(67938662) | `docs/tech-debt/README.md:253` は `git show e38b89a9:docs/tech-debt/README.md` の同じ行と一致。`.claude/skills/director/SKILL.md:495`〜`:498`。`internal/org/escalate.go:288`、`:311`。(2) の「each copy holds it once」は今も成り立つ(`grep -c` で 1) | `/sync-docs` で 253 行の Trigger に「feat/org-director-wait(8b-2b)が受信箱の処理と `WaitInbox` を変えたが、(1)〜(3) は払わずに持ち越した」と足す。あわせて、Codex の打ち直しの文が `--director` の wait の時間切れ(`WaitInboxForDirector`)にも依るようになり、その文面は `TestOrgWaitDirector_TimesOutWithThePhrase`(`internal/cli/org_wait_director_test.go:401`)が文字どおり固定していることも書くとよい |
| V2 | LOW | 実機の evidence の「Not checked here」は、open の件が 4 より先になる場合を「the unit tests cover it (`internal/org/director_wait_test.go`)」と書く。そのテスト `TestWaitInboxForDirector_AnOpenItemWinsOverAWatchThatStopped` は c1d31e3f の分割で `internal/org/director_wait_watch_test.go:158` に移った | `docs/evidence/director-wait-livefire-2026-10-19.md:380`〜`:382`。evidence は c780bc11 で書かれ、分割はそのあと | `/sync-docs` でパスを `internal/org/director_wait_watch_test.go` に直す。記録の中身は変わらない |
| V3 | INFO | 計画の D1 は「待つあいだも印を書き直すたびに登録を読み直し」と書くが、実装が登録を読み直すのは 30 秒ごとの書き直し(`refresh`)だけで、見張りが受け付けるようになったときの書き直し(`probeAgain` から `save`)と終わりの書き込みでは読み直さない。help、skill、仕様の注、AC4(「次の印の書き直しで」)は 30 秒ごとと書き、実装と一致する。読み直さない書き直しが挟まっても、印は前の登録の id と `attached_at` を持つので、表示は `DirectorWaitFor` で今の登録の印としては読まない | 計画 64 行。`director_wait.go:541`〜`:546`、`:553`〜`:564`、`:589`〜`:594` | 直さなくても誤りにはならない。直すなら D1 を「30 秒ごとの書き直しの前に登録を読み直し」とする |
| V4 | INFO | 計画の Progress checklist の「Review artifact created」が `[ ]` のまま。self-review の報告は 73a6310b でコミットされ、e7940546 で直しの確かめの節が足された | 計画 169 行 | `/sync-docs` か `/pr` で付ける |
| V5 | INFO | README の 260 行と `docs/recipes/director-wake.md`(6 行、270 行、templates/base の写しも)は、pull の director が `ralph org wait --inbox` で待つと書く。動詞としては今も正しいが、README の 258 行と skill は `--director <id>` を付けた形を書く | `README.md:258`、`:260`。`docs/recipes/director-wake.md:6`、`:270` | 揃えるなら `/sync-docs` で `--director <id>` を足す。必須ではない |
| V6 | INFO | 実機は fa95a3d8 の binary で行い、evidence はその後の変更を「the later S4 commits change only documents, test files, and the headless role prompt」と書く。これは c780bc11 の時点の文で、そのあとの c1d31e3f と 33da4185 も Go を変えた。中身は、定数の名前を 1 つにした(値は 30 秒のまま)、stderr の前置きを同じ byte の定数にした、`org.go` の help の 5 行、コメント、`org_director.go` から `org_director_report.go` への移動(コメントと空行を除いた行は `package` と `import` のほか一致)だけで、実機が見た動きは変わらない | `git diff --stat fa95a3d8 HEAD -- '*.go' ':!*_test.go'`(8 ファイル)。evidence 4〜6 行 | evidence に「c1d31e3f と 33da4185 は動きを変えない」と 1 文足すかを `/sync-docs` が決める |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 「registration does not tell whether its session is still running」 | 合う | 昇格したセッションの `director:` の行は、その文のあとに wait の印の事実か `no wait --inbox --director mark for it` を足す(`org_director_report.go:170`〜`:176`)。help(`status.go:93`、`org_director.go:308`)、README(`:125`、`:258`)、skill(`SKILL.md:41`〜`:45`)、tech-debt 247 行は、印は wait が書いたことでありセッションの生死ではないと書く |
| skill の待ち方(Claude Code と Codex) | 合う | 2 つとも `--director d1` を付ける(`SKILL.md:487`、`:495`)。Claude Code の `--timeout-ms 0` の 1 は失敗、Codex の時間切れの文は打ち直し、4、5、2 種の `note:`、`wait --inbox --director: ` で始まる行、古い ralph の 1 を書き分け(`:514`〜`:542`)、失敗の段落(`:544`〜`:553`)は 0、4、5、時間切れを除く。食い違いはない |
| director が自分の判断で見張りを起動し直す手順 | 残っていない | skill の `watch --all --detach` と「見張りを起動」は、昇格の 2 と 3(人が頼んだ昇格の中で attach の前に起動する手順、`SKILL.md:72`〜`:88`)、headless の節、4 の扱い(事実とコマンドを人に伝える、`:516`〜`:525`)だけで、「してはいけないこと」(`:855`〜`:858`)が昇格の 2 と分けて禁じる。雛形の `watch --all --detach`(`director.md:29`)は start が済ませたことを言う |
| tech-debt 209・247・248 | 合う | 209 は `--director` の wait が気づくこと、push だけの director、`--director` なしの wait、leader はまだ気づかないこと、誰も自動では起動し直さないことを書く。247 は 8b-2b で入った印と残り(印はセッションの生死を言わない、push だけと `--director` なしには印がない)を書き、きっかけは 8b-2c。248(新規、part B)は漏れた wait、残る穴、実験の事実、まだ確かめていない 3 つ、8b-2c を書く。どれも 5 セル |
| tech-debt 231 (j)・250・306・307 | 合う | 231 (j) は `signalCauseContext` が残ることと 2 つのテストの移し先(self-review N2)、250 は exit 5 を書く。306 は今の行数(`wc -l` で 342、505、683、590、743)と `director_wait.go` のきっかけ、307 は go test の時間切れで、数字は log と合う |
| tech-debt 253 | ずれ(V1) | きっかけが起きたのに行が変わらない |
| help(wait、director status、status) | 合う | wait の help は 30 秒の読み直し、5 秒の確かめ、4 と 5、`note:`、signal と nohup を書く。director status と status の help は `wait:` の行、`director.wait` の 8 キー、60 秒と 30 秒を書く |
| README、AGENTS.md、仕様の注 | 合う(V5 は情報) | README は wait の行、status の行、director の段落。AGENTS.md は `director_wait.go` |
| `/director` と `/org` の skill の 4 面、雛形 | 合う | cmp がすべて一致。雛形は wait に `--director {{ID}}` を付け exit 5 の扱いを書き、`prompts_test.go` が描画を固定する |
| evidence | ずれ(V2)、情報(V6) | |

## Observational checks

- c1d31e3f の分割: 分割前の `internal/org/director_wait_test.go`(1252 行)の `func` の行の集合と、分割後の 2 ファイルの集合は一致した(40)。分割前の `internal/cli/org_director.go`(831 行)と、分割後の `org_director.go` と `org_director_report.go` の `func` の行の集合も一致した(21)。移した範囲からコメントと空行を除いた行は、新しいファイルの `package` と `import` を除いて byte で一致した
- 実機の binary(fa95a3d8)より後の非テストの Go の変更は V6 のとおりで、動きを変えるものはない
- self-review の直しの節の N1 と N2 は e7940546 で直っている(306 行の `is 743 lines`、231 行の (j) の 2 つのテストの移し先)。L6 は `/pr` で扱うと決まっている
- `git status --short` は検証の前もあとも空(gitignore の log だけが増えた)

## Coverage gaps

- テストは走らせていない(`/test` の範囲)。AC の「満たす」は、実装とテストの中身を読んで照らした結果
- 全体の run-verify は lead が回した 2 回の log を読んだだけで、回し直していない
- 実機の wait は shell から打ったもので、Monitor で打った wait と codex の繰り返しで打った wait は実機で見ていない(evidence の「Not checked here」のとおり)。nohup で始めた wait と、open の件が 4 より先になる場合はテストだけが見ている
- Linux での signal の継承(`signal.Ignored`)は手元で確かめていない
- insight event は、ほかのファイルを変えない指示のため `docs/insights/events/` に追記していない。`--events-dir` で一時の dir に作った行は次のとおり: `{"schema":1,"ts":"2026-10-19T04:30:00Z","slug":"org-director-wait","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":2},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}`

## Cycle 2 (8a33364b..89b81a2a)

- Date: 2026-10-19
- Verifier: verifier subagent(Claude Code、pipeline cycle 2)
- Scope: `git diff 8a33364b..HEAD`(HEAD 89b81a2a、10 ファイル、+491/-57)。コードを変えたのは 38cac6f1(`begin` がロックの中で登録を読み直す)、b3b692e1(G1〜G5 と G7 のテスト、`DirectorWaitWatchProbeInterval` の export、`reraiseSignal` のコメント)、6086af65(`begin` を `decideStart` と `noteStartWatch` に分け、コメントを直した)の 3 つ。ほかは計画、tech-debt の 306 行と 308 行、self-review の cycle 2 の節、insight event。ブランチ全体(`e38b89a9...HEAD`、53 ファイル、+6850/-472)は、cycle 2 の差分が触れた所と、文書が cycle 2 の動きをどう書いているかを見直した
- Evidence: `docs/evidence/verify-2026-10-19-org-director-wait.log` の末尾に cycle 2 の生の出力を足した。run-static-verify.sh が書いた log は `docs/evidence/verify-2026-10-19-081829.log`(どちらも gitignore の対象)
- 計画の digest: `./scripts/plan-visual.sh digest` の出力は `e67985a2f2d7` で、計画の `- Approved:` の行と一致する

### Spec compliance(cycle 2 で変わった所)

| 項目 | Status | Evidence |
| --- | --- | --- |
| AC1 と D2: 最初の確かめとロックのあいだに登録が変わった wait | 満たす | `WaitInboxForDirector` の最初の読み(`internal/org/director_wait.go:291`〜`:297`)のあと、`begin` は見張りを確かめ、`director-wait.lock` の中で `director.jsonl` を読み直して、`refresh` と同じ `directorWaitCheckCurrent(w.p.ID, &w.reg, ...)` で確かめる(`:420`〜`:429`)。今の登録でなければ、`ErrDirectorWaitNotCurrent` を包み今の登録を名指す誤りを返し、`decideStart` も印の書き込みも `noteStartWatch` も通らない(`:433`〜`:435`)。CLI はこの誤りを `ExitCode` で 5 にする(`internal/cli/exitcode.go:32`)。読めなければ prefix 付きの plain error(終了コード 1)で、やはり印を書かない。lock のファイルは残りうるが、それは `WaitInboxForDirector` の doc が書いている。`TestWaitInboxForDirector_ARegistrationReplacedBeforeTheStartTakesTheLock`(`internal/org/director_wait_owner_test.go:50`)は、seam `directorWaitBeforeBeginLock`(`director_wait.go:413`、`:489`)で d1 の確かめとロックのあいだに `attach --replace` と d2 の wait の書き込みを差し込む。そのうえで、d1 が d2 を名指して 5 で返ること、印が d2 のもののまま byte で同じこと、d1 の stderr が空であること、d2 のその後の書き直しと end が載ることを確かめる。`..._ARegistrationThatCannotBeReadUnderTheStartsLockWritesNothing`(`:137`)は、読めない場合を前の印がないときとあるときの 2 通りで確かめる |
| 直列の議論 | 満たす(コードを読んで確かめた) | 新しい wait は、新しい登録を読んだあとで、同じロックの中で印を書く。古い wait がロックを取るのがその書き込みより後なら、ロックの中の読み直しで新しい登録が見えて断る。先なら、古い wait の印が先に載り、新しい wait があとで上書きする。どちらの順でも、今の登録の wait の印を古い wait が上書きすることはない。`DirectorStore.Read` はロックを取らないので、`director-wait.lock` を持ったまま別の lock を待つ道はない(self-review cycle 2 の確認と同じ) |
| seam | 本番の動きを変えない | `directorWaitBeforeBeginLock` に代入するのは `director_wait_owner_test.go:31`〜`:33` だけで、本番では nil の判定が 1 つ増えただけ |
| 6086af65 の分割 | 動きを変えない | `decide` の closure を `decideStart` のメソッドにし、`readErr` と `notCurrent` を `refused` 1 つにまとめ、最後の `note:` の分岐を `noteStartWatch` に出した。`refused` に入る値は前と同じで、読めない誤りには同じ prefix を付ける。`noteStartWatch` に渡す `w.watch` は、`begin` の冒頭で `l` を入れたあと変わらない。`begin` は 43 行になり、self-review cycle 2 の L1(70 行)は直った。L2 の 3 か所(冒頭の読む回数 `:21`〜`:25`、`directorWaitCheckCurrent` の doc、`directorWaitLockFile` の doc `:46`〜`:48`)も直っている |
| G1〜G4 のテスト(AC2、AC3、AC5) | テストがある | `..._AMarkRemovedWhileTheWaitRunsIsNotWrittenAgain`(`director_wait_owner_test.go:184`)、`..._AStartThatCannotTakeTheLockStillCarriesOverTheWatch`(`:216`)、`..._TheDetailIsEscaped`(`director_wait_watch_test.go:428`)、`..._ACancelDuringTheStartSettleWritesNoNote`(`:467`)。走らせてはいない(`/test` の範囲) |
| G5: help の 5 秒と 30 秒 | テストがある | `TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer`(`internal/cli/org_director_wait_line_test.go:654`)が `org.DirectorWaitMarkInterval` と `org.DirectorWaitWatchProbeInterval` から「30 s」と「5 s」を作り、help の 3 つの文と突き合わせる。作った binary の `ralph org wait --help` は「every 30 s」を 2 か所、「Every 5 s」を 1 か所に出した |
| G7: `reraiseSignal` のコメント | 合う | `signal.Reset(sig)` が signal を送る前に Notify を外す(`internal/cli/org_wait_director.go:132`)ので、「呼び出し側の stop は前でも後でもよい」という今の文(`:124`〜`:126`)はコードと合う |

### 文書の確かめ(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `ralph org wait --help` | 合う | 「It waits only while <id> is the current registration and not stopped: otherwise it exits 5 without waiting」(`internal/cli/org.go:1264`〜`:1266`)は、ロックの中の読み直しで断られる場合にも当たる(待たずに 5 で返り、印を書かない)。30 秒の読み直し、30 秒の印、5 秒の確かめ、4 と 5、`note:`、signal の文は cycle 1 から変わらず、実装の値と合う |
| `/director` skill の 4 面 | 合う | cycle 2 では変わっていない。4 面は `cmp` で一致する。「始める前にそうでなければ待たずに終了コード 5 で返る」(`.claude/skills/director/SKILL.md:502`〜`:503`)は、始めるあいだに登録が変わった場合も同じ扱い(5 なら director として動くのをやめる)で足りる。0、1、4、5 の扱いと 5 秒・30 秒は実装と合う |
| 仕様の 8b-2b の注 | 合う(C2-3 は情報) | 「始める前にそうでなければ印を書かずに終了コード 5」と「30 秒ごとの印の書き直しの前に読み直し」は実装と合う。「始めた wait は前の印を自分の `wait_id` で上書きして持ち主になり」は、その前のロックの中の読み直しに触れていない |
| README、recipe、AGENTS.md、雛形 | 合う | cycle 2 では変わっていない。README の 124 行と 258 行、recipe の 7 行と 271 行(root と template)は `--director <id>` の wait と 4・5 を書き、cycle 2 の動きと食い違わない |
| 旧名 `directorWaitWatchProbeInterval`、`directorWaitMarkInterval` | 残っていない | `git grep` で残るのは、cycle 1 の self-review、sync-docs、test の報告と、計画の Progress の 169 行(D2 の名前が変わったという記録)だけ。コード、help、計画の D2・D3、tech-debt にはない |
| 「始めた wait は登録を読み直さずに印を書く」という文 | 残っていない | 現行の文書とコメントにはない。同じ意味の文は、cycle 1 の cross-review triage の「Implementation context summary」、この報告の V3、test の evidence(`begin: decide()` の行番号)にあり、どれも cycle 1 の時点の記録 |
| 実機の evidence | ずれ(C2-1) | |
| 計画の Progress | 遅れ(C2-2) | |

### tech-debt の 306 行と 308 行

- 306 行: cycle 2 の文の数字は `wc -l` と合う(`director_wait.go` 799、`director_wait_test.go` 687、`director_wait_watch_test.go` 630、`director_wait_owner_test.go` 235)。ほかの数字(`director_start.go` 1203、`director_check_test.go` 857、`director_start_cleanup_test.go` 804、`herdr_test.go` 852、`director_stop.go` 782、`org_director_report.go` 342、`org_director.go` 505)も今の行数と合う。挙げた `decideStart` と `noteStartWatch` はある。Trigger の「`director_wait.go` を 800 行より長くする次の変更」はまだ起きていないが、799 行なので 2 行足せば起きる。セルは 5 つ
- 308 行: 挙げたテスト 5 つはどれもある(`director_wait_owner_test.go` に 2 つ、`director_wait_watch_test.go` に 2 つ、`org_director_wait_line_test.go` に 1 つ)。G6 の 4 つの関数(`directorWaitItemsDetail`、`probeAgain`、`directorWaitWatchFacts`、`directorWaitWatchState`)もある。cycle 2 の差分はこの 4 つの中身を変えていない(`noteStartWatch` が `directorWaitWatchFacts` に渡す引数が、`l` から同じ値の `w.watch` になっただけ)ので、Trigger は起きていない。G6 の「どのテストも走らせない」は coverprofile を取る `/test` の範囲で、ここではテストの文面を grep しただけ(2 件以上の `open inbox items` を期待するテストは見つからなかった)。セルは 5 つ

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(changed scope) | pass(rc 0) | 選ばれた言語は golang。shellcheck、hook の `sh -n`、guard の awk の構文、settings の `jq`、Codex の hook の 3 つの確かめ、check-sync(DRIFTED 0、KNOWN_DIFF 5、TEMPLATE_ONLY 11)、check-pipeline-sync、check-skill-sync(14 skill)、template purity、tech-debt の plan 参照、gofmt ok、golangci-lint 0 issues、secret scan(e38b89a9..89b81a2a、clean)。log は `docs/evidence/verify-2026-10-19-081829.log` |
| `go vet ./...`(go1.26.0) | pass(rc 0) | |
| `GOTOOLCHAIN=go1.25.8 go build ./... && GOTOOLCHAIN=go1.25.8 go vet ./...` | pass(rc 0) | |
| `git diff --check 8a33364b..HEAD` と `git diff --check e38b89a9...HEAD` | pass(出力なし) | |
| `gofmt -l internal cmd` | 出力なし | |

### Findings(cycle 2)

CRITICAL、HIGH、MEDIUM はない。LOW 1 件は evidence の文で、コードと出力は変わらない。

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| C2-1 | LOW | 実機の evidence の冒頭は、実機に使った fa95a3d8 の binary より後のコミットについて「none of them changes what the wait does or what it prints while it waits」と書く。45f5d586 の時点では正しかったが、cycle 2 の 38cac6f1 は wait の動きを変えた。最初の確かめとロックのあいだに登録が変わった wait は、前は印を書いて待ち、今は印を書かずに 5 で返る。(a)〜(h) はどれもこの窓を通らない((e) の `--replace` は wait が始まったあとで、5 は 30 秒の読み直しから来た)ので、記録された結果は HEAD の binary でもおそらく同じ。実機で打ち直してはいない | `docs/evidence/director-wait-livefire-2026-10-19.md:4`〜`:10`。`git diff --stat fa95a3d8 HEAD -- internal/org/director_wait.go`(+126/-63) | `/sync-docs` で冒頭に 1 文足す。たとえば「cycle 2 の 38cac6f1 は、最初の確かめとロックのあいだに登録が変わった wait を、印を書かずに 5 で返すようにした。(a)〜(h) はこの窓を通らない。6086af65 は関数の分割とコメント、b3b692e1 は定数の export とコメントだけ」 |
| C2-2 | INFO | 計画の Progress の 173 行は「L1(`begin` が 70 行)と L2(登録を読む回数のコメント)は新しい implementer で直す」のままで、それを直した 6086af65 と、306 行の行数を直した 89b81a2a の記録がない | 計画 173 行。計画を最後に変えたコミットは 498a958c | `/sync-docs` か `/pr` で 1 行足す。digest は計画の本文を覆うので、足したら承認を取り直す |
| C2-3 | INFO | 仕様の 8b-2b の注は、始めた wait が前の印を上書きして持ち主になると書き、その前のロックの中の読み直しには触れていない。読み直さないとは書いていないので食い違いではない。使う人に見える結果(始めるときに今の登録でなければ印を書かずに 5)は、同じ注の前の文が言う。詳しい形は計画の D2 にある | `docs/specs/2026-10-07-org-multi-org-director.md:90`。計画の D2(67 行) | 直さなくてよい。足すなら「始めた wait は、同じロックの中で登録を読み直してから前の印を上書きする」 |
| C2-4 | INFO | cycle 1 の Coverage gaps の最後の項目は、insight event を `docs/insights/events/` に追記していないと書く。実際には、この報告を入れた 45f5d586 が verify の cycle 1 の行(`ts` 04:33:29Z、low 2)も足している。cycle 1 の記録は書き換えず、ここで正す | `git log -1 -S'"phase":"verify","cycle":1' -- docs/insights/events/2026-10-19-org-director-wait.jsonl` は 45f5d586 | なし |

引き継いだ点(新しい指摘ではない):

- self-review cycle 2 の I2: `begin` がロックを取れなかったとき(別のプロセスが 5 秒ロックを持ち続けた場合)は登録を読み直さず、ロックなしで読んだ前の印の `wait_id` を `prevWaitID` に持つ。その前に `attach --replace` と新しい wait の書き込みが入っていて、ロックなしの読みがその新しい印を読んでいれば、30 秒後の `refresh` が書く `not_current` の end は `owns`(`director_wait.go:697`)を通って新しい印を置き換える。ロックのファイルを開けない場合(directory、権限)は `save` も書けないので、残るのはロックの時間切れと競合が重なる場合だけ。6086af65 はここを変えていない。self-review は上げないと決めており、ここでもそれに従う
- self-review cycle 2 の I1(既定値の代入と、`begin` が渡す `&w.reg` を固定するテストがない)は `/test` の範囲

### Coverage gaps(cycle 2)

- テストは走らせていない(`/test` の範囲)。「満たす」はコードとテストを読んで照らした結果で、競合のテストが直しを外すと落ちることは implementer の報告による
- 実機の確認は fa95a3d8 の binary のままで、38cac6f1 以降の binary では打っていない(C2-1)
- 全体の run-verify は回していない

## Verdict

- Verdict: pass(ブランチ全体、cycle 2 のあと、HEAD 89b81a2a)。CRITICAL、HIGH、MEDIUM はない。cycle 2 の指摘は LOW 1 件(C2-1、実機の evidence の冒頭の文)と情報 3 件(C2-2〜C2-4)で、どれも記録の文。cycle 1 の LOW 2 件(V1、V2)と情報の V4〜V6 は cycle 1 の sync-docs までに直り、V3 は計画の Progress に記録された
- Verified: cross-review の ACTION_REQUIRED(始める前の確かめとロックのあいだの `attach --replace`)を 38cac6f1 が閉じたこと(ロックの中の読み直し、印を書かない 2 つの出口、終了コード 5 への対応、直列の議論、seam が本番で nil であること)。6086af65 が動きを変えないこと。help、skill の 4 面、README、recipe、仕様の注、雛形が cycle 2 の動きと食い違わないこと。旧名が現行の文書とコードにないこと。tech-debt の 306 行と 308 行が今の行数と名前に合うこと。静的解析(run-static-verify、go1.26.0 の vet、go1.25.8 の build と vet、diff --check、gofmt)。cycle 1 で確かめた AC1〜AC9、D1〜D8、FR-6、FR-9、FR-10、8b-2a の確かめ、`internal/org/watch.go` は、cycle 2 の差分がそこに触れていないことを確かめたうえで引き継ぐ
- Partially verified: AC9 の実機(fa95a3d8 の binary で shell から打った wait だけ。38cac6f1 以降の binary と、Monitor と codex の道は実機なし)
- Not verified: テストの実行と `-race`、Linux での signal の扱い、全体の run-verify
- cycle 1 の判定(HEAD c13455e3、履歴): pass。LOW 2 件(V1 の tech-debt 253 行、V2 の evidence のパス)と情報 4 件(V3〜V6)。そのとき確かめた範囲は、上の cycle 1 の節と同じ。加えて c1d31e3f が移動だけであることと、実機の binary より後の変更の範囲を確かめた
