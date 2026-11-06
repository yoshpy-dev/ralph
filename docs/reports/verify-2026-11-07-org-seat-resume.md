# Verify report: org の座席を API エラーから続ける(`ralph org resume` と再試行の設定)

- Date: 2026-11-07
- Plan: `docs/plans/active/2026-11-07-org-seat-resume.md`(spec `docs/specs/2026-11-07-org-seat-resume.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプラインの cycle 1
- Scope: `git diff 11239c8d...HEAD`(base main 11239c8d = origin/main、HEAD 40f7dbc5、71 ファイル)。仕様適合、静的解析、文書の drift。テストは走らせていない(/test の担当)
- Evidence: `docs/evidence/verify-2026-11-07-org-seat-resume.log`(git の対象外)

## Spec compliance

計画の AC は仕様の AC をそのまま使っている(計画の「仕様の AC をそのまま使う」)。下の表の AC は両方を指す。テストの欄は、テストがあって assert が AC を言っていることを読んで確かめたもので、結果は /test が出す。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| FR-1 動詞、対象、`--org-id`、`--dry-run`、`--state-dir` | Met | `internal/cli/org_resume.go`(`--dry-run`、`checkResumeOrgID`)、`internal/cli/org.go`(登録と `--org-id` の説明)。`Resume` は `--org-id` があると director を含めない(`resume.go:231`)。director は `OpenHeadless` のうち herdr の中で pane があるもの(`resumeDirectorTargets`)。`TestOrgResume_OrgIDFilter_TwoOrgsAndTheDirector` |
| FR-2 判定の順(1〜4)と 2026-11-07 の改訂 | Met(INFO I1) | `resumeOne`: 自分の pane → `herdr agent get`(idle/done 以外は waiting か skipped)→ `pane read --lines 60` → `ClassifyResumeScreen`。footer の読み飛ばし、折り返しの結合、状態行 1 行の許容、prompt の行と prompt の後の最初の `⎿` の行での区切り(`resumeScreenBlocks`)。fixture 18 個(`testdata/resume/`) |
| FR-3 文の一覧を 1 ファイルに、driver ごとに 3 種類、出どころ | Met(INFO I3) | `resume_screens.go` の `resumeScreenSentences`。各項目に `seen:`/`docs:`/`binary strings:`。`TestResumeScreenSentences_EveryEntryHasASource` |
| FR-4 leader 以外 → leader → director | Met | `resumeSeatTargets`、`TestOrgResume_OrderIsSeatsThenLeadersThenDirectors` |
| FR-5 続行の文、Enter 1 回、Enter だけの道、watch gate と承認の確認をかけない | Met | `ResumeText` は仕様の文と一字一句同じ(`TestResumeText_IsTheSentenceOfTheSpec`)。`typeAndSubmit` を `Send` と共有。`Resume` は `checkWatchGate` と承認の確認を呼ばない |
| FR-5a pane ごとのロック、ロックの中の読み直し、Enter の前の確かめ | Met(INFO I4) | `pane_lock.go`、`Send` は idle/done の待ちのあとに取る(`verbs.go:443`)、`resumeOne` はロックの中で `resumeLook` をやり直す、`ResumeInputHolds`。`pane_lock_test.go` の 10 本 |
| FR-6 `resumed`(状態を変えない、種類と照合した行)、dry run は記録しない | Met | `EventResumed` は `stateEvents` にない。Details は `kind=... line=...`。`latestSeatEventTS` は種類を問わない(D3)。`leaderActivityEventCount` は数えない |
| FR-7 座席ごとの 1 行、リセット時刻はそのまま、終了コード | Met(INFO I2) | `printResumeResult`。`TestOrgResume_ExitsOneWhenASendFails`、`..._ExitsOneWhenTheManifestCannotBeRead`。記録の失敗でも 1 になる(仕様の外の拡張、org の skill と org runtime の仕様の改訂には書いてある) |
| FR-8 `[org].api_wait_minutes`、3 つの変数、0 で無し、負は断る、`ralph.local.toml` | Met | `config.go`(既定 30、`validateLoaded`)、`driver.ClaudeRetryEnv`、`seatPaneExtraEnv`、`claudeRetryPaneEnv`。`api_wait_minutes_test.go` の 3 本 |
| FR-9 codex の座席と director には足さない | Met | `claudeRetryPaneEnv` は driver が claude のときだけ。spawn、director の herdr、launch_cmd のテストに codex の行がある |
| FR-10 director の手順書と役割 prompt | Partially met(LOW V1) | `/director` skill の「止まった座席を続ける」、ALERT の項、「してはいけないこと」。3 つの役割 prompt に 1 行。ただし `waiting` の座席に `ralph org read` の打ち方を添える指示がない(V1) |
| FR-11 文書 | Met(LOW V2) | `/org` skill の動詞表と 2 つの列挙、org runtime の仕様の 2026-11-07 改訂、AGENTS.md の `internal/org/` の行、テンプレートの `ralph.toml`、`agent-messaging.md`(と写し)。README は FR-11 の一覧の外だが動詞の列挙を持つ(V2) |
| NFR: 決定的、LLM を呼ばない | Met | `ClassifyResumeScreen` は I/O のない関数。`resume.go` が呼ぶのは herdr と manifest だけ |
| NFR: 一覧にない画面、選択と承認の画面には打たない。herdr の状態だけでは打たない | Met | 打つのは stopped と enter_only のときだけ。番号付きの選択肢は choice dialog。`TestOrgResume_CodexHooksReviewDialog_Skipped`、`TestOrgResume_ScreenChangedUnderTheLock_TypesNothing`(フォルダ信頼の画面) |
| NFR: herdr の呼び出しを時間で区切る、1 座席の失敗で止めない | Met | `resumeLook` は `callWithTimeout`、打つ部分は `resumeSubmitBudget`(driverCallTimeout × 3 + 待ち + 確認)。ロックの待ちは 30 秒。`TestOrgResume_OneTargetFails_TheNextIsStillResumed` |
| NFR: 偽物の Runner、実機の画面を使う | Met | `fakeResumeHerdr`。`claude-enotfound.txt` は記録の 2 の画面と同じ文字列(突き合わせた) |
| NFR: 変数はこの版で spawn した座席から | Met | spawn のときの `--env` で入る。文書(skill、template、spec の改訂)にも書いてある |
| AC-1 止まった画面に文と Enter 1 回、`resumed` 1 件、「続けた」 | Met | `TestOrgResume_StoppedClaudeSeat_TypesTheSentenceAndEnterOnce`(herdr の呼び出しの順、打った文、manifest の 1 件まで見る) |
| AC-2 再試行中は working でも idle でも打たない | Met | `TestOrgResume_RetryingOrWorking_Waits` |
| AC-3 codex のフック確認の 3 択は飛ばす | Met | `TestOrgResume_CodexHooksReviewDialog_Skipped`(理由は `unknown screen: choice dialog`) |
| AC-4 `Press enter to continue` には Enter だけ | Met | `TestOrgResume_PressEnterToContinue_SendsEnterOnly` |
| AC-5 自分の pane は飛ばす | Met | `TestOrgResume_CallersOwnPane_Skipped` |
| AC-6 `--dry-run` は send-text と send-keys を呼ばず、記録しない | Met | `TestOrgResume_DryRun_TypesAndRecordsNothing`(ロックのファイルも作らない)、`internal/cli` の `TestOrgResume_DryRunTypesNothingAndSummarizes` |
| AC-7 implementer → leader | Met | `TestOrgResume_OrderIsSeatsThenLeadersThenDirectors` |
| AC-8 claude の座席と director には 3 つ、codex と 0 には無し | Met | `TestOrgSpawn_ClaudeRetryEnvFollowsEachSeatsDriver`(6 通り)、`..._Layout_TabOfAClaudeSeatGetsTheClaudeRetryEnv`、`TestDirectorStart_ClaudeRetryEnvFollowsTheDirectorsDriver`、launch script の 3 本。実機でも `ps eww` で 3 つを見た(live の記録の 2) |
| AC-9 負は断る、既定値がテンプレートと一致 | Met | `validateLoaded`、`orgTemplateDrift` に `org.api_wait_minutes`、入れ替えの検査にも足した。テンプレートは `api_wait_minutes = 30` |
| AC-10 実機で通信断 → 戻す → resume で作業が最後まで進む | Partially met | 下の「AC-10 の判断」 |
| AC-11 30 分に見合う回数を実測し、計画と記録に書く | Met | `docs/evidence/org-seat-resume-retries-2026-11-07.md`(10 回 189 秒、20 回 554 秒、40 回 1265 秒、式、30 分で 55 回)。計画の Progress の S2 の行にも同じ数。式の 5 つの値を計算で確かめた |
| AC-12 手順書に手順と禁止、写しとテンプレートの一致の検査が通る | Met(LOW V1) | `check-skill-sync.sh`(14 skill)、`check-sync.sh`(DRIFTED 0)が今回の静的検証で通った。4 つの写しは cmp で一致。V1 の 1 点を除き FR-10 の中身がある |
| AC-13 3 つの役割 prompt の 1 行とテスト | Met | `prompts_resume_test.go` の 2 本。文の先頭を `ResumeText` から作るので、prompt と定数がずれるとテストが落ちる |
| AC-14 同じ pane への同時の打ち込み、承認画面への変化、入力欄の確かめ | Met(INFO I4) | `TestOrgResume_TwoResumesOfOnePane_...`、`..._SendWaitsForAResumeHoldingThePaneLock`、`..._ScreenChangedUnderTheLock_...`(4 通り)、`..._TypedTextNotInTheInputBox_NoEnter`(2 通り)、`..._PaneLockHeld_...` |
| AC-15 小さい `api_wait_minutes` で止め、ターンの終わりと stall の ALERT までの時間を測る | Met | live の記録の 2: `api_wait_minutes = 2` で 3 つの変数(8 回、120000)、送信からターンの終わりまで 110.4 秒、そこから stall の ALERT まで 25 秒、再試行中は ALERT なし |
| 計画 D1〜D8 | Met | D2 `typeAndSubmit`、D3 非状態イベント、D4 式は `driver.ClaudeMaxRetriesForMinutes` と doc、D5 60 行、D6 `--org-id` 任意、D7 終了コード、D8 ロック。計画の digest は `0893ac30252e` で `- Approved:` の行と同じ |

### AC-10 の判断

実機で確かめたこと(`docs/evidence/org-seat-resume-live-2026-11-07.md` の 1):

- プロキシを止めた接続の拒否で、claude の座席が既定の 10 回の再試行のあとエラーでターンを終え、herdr が `idle` を返した。
- `--dry-run` は `would_resume` を出し、何も記録しなかった。
- プロキシを戻して `ralph org resume` を打つと、続行の文を打って Enter を押し、Enter の前の入力欄の確かめも通り、manifest に `resumed` が 1 件増えた。終了コードは 0。
- Claude Code のセッションの記録に、続行の文が利用者の入力として入った。座席はすぐに API へ要求を出し直した(プロキシのログ)。
- 動いている座席には打たなかった(`waiting`)。

確かめていないこと:

- 座席が作業(`done.txt` の作成)を最後まで進めること。アカウントが週の利用上限に達していて、要求が `rate_limit` で返った。
- 役割 prompt の 1 行の効き目。実機の座席は役割 `worker` で、役割 prompt のテンプレートを持たない。

判断: ralph の側の約束(止まった画面を見つける、文を 1 回打つ、Enter を 1 回押す、記録する、止まっていない座席に打たない)は実機とテストの両方で確かめた。残るのは、文を受けたモデルが作業を続けるかどうかで、ralph の決定的な部分の外にある。単体と結合のテストは、実機で通らなかった道(Enter だけ、codex、director、順番、ロック)を覆っている。merge を止める穴とは見ないが、AC-10 の文言は満たしていないので Partially met とし、PR の既知の穴に書くよう勧める。

いちばん効く追加の確かめ: 週の上限のリセット(2026-11-07 23 時、日本時間)のあとに、live の記録の 1 の手順を、役割 `implementer` の座席でもう一度走らせ、`done.txt` ができることを見る。役割 prompt の 1 行もそこで効く。

なお、live の確認は `b5ea321f` の binary で、画面の区切りを直した S11(`8e9504f6`)より前である。S11 が変えたのはまとまりの区切り方だけで、接続の拒否の画面は prompt とエラーの間に空行があるので、直しの前後で同じ判定になる(`resumeScreenBlocks` を画面の行で追った)。週の上限の画面の照合した行は、直しのあと prompt の文を含まなくなる。これは fixture `claude-weekly-limit-under-prompt.txt` で確かめていて、実機ではやり直していない。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | Pass(exit 0) | testdata の `.txt` が言語に分類されないため、Go は全体で走った |
| shellcheck と `sh -n`(hooks、verify scripts、両方の写し) | OK | |
| `jq -e .`(2 つの `settings.json`) | OK | |
| Codex の hook の guard 3 つ | OK | |
| `scripts/check-sync.sh` | PASS | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| `scripts/check-pipeline-sync.sh` | OK | |
| `scripts/check-skill-sync.sh` | OK | 14 skill |
| `scripts/check-template-purity.sh` | PASS | |
| tech-debt の plan の参照 | OK | |
| gofmt、`go vet ./...`、golangci-lint、staticcheck | OK | golangci-lint は 0 issues、staticcheck は出力なし |
| `secret-scan-branch.sh` | clean | 11239c8d..40f7dbc5 |
| `git merge-tree --write-tree origin/main HEAD` | 衝突なし | origin/main は base と同じ 11239c8d |
| HEAD の binary の `ralph org resume --help`、`ralph org --help`、`ralph org send --help` | 読んだ | V3 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill(4 つの写し) | ほぼ | V1: `waiting` の座席に `ralph org read` の打ち方を添える指示がない。FR-10 は待ち中と飛ばした座席の両方に求めている |
| `/org` skill(4 つの写し) | Yes | 動詞表の `resume`、`send` の行のロック、台帳を書き換える動詞と読むだけの動詞の列挙、「見張りがなくても使える」の列挙。`Resume` は `checkWatchGate` を呼ばないので、後者に入るのは正しい |
| `internal/cli/org_legacy_ledger.go` の列挙 | Yes | `resume` は書き換える側、`resume --dry-run` は読むだけの側 |
| 役割 prompt(leader、implementer、reviewer) | Yes | `director.md` は手順を skill に任せているので、FR-10 の「手順を持つならそこも」には当たらない |
| `.claude/rules/ralph/agent-messaging.md` と写し | Yes | |
| org runtime の仕様の 2026-11-07 改訂 | Yes | (a)〜(d) をコードと突き合わせた(30 秒、55 回、記録の失敗で 1、director は記録しない) |
| AGENTS.md の `internal/org/` の行 | Yes | `resume.go`、`resume_screens.go`、`pane_lock.go` がそれぞれ 1 回ずつ出る |
| `templates/base/ralph.toml` と `config.Default()` | Yes | `api_wait_minutes = 30`、`orgTemplateDrift` が比べる。リポジトリの `ralph.toml` は既定と違う値だけを書くので足さないのが正しい |
| README.md | No | V2: 動詞の列挙(131 行)に `resume` がなく、段落もない。`api_wait_minutes` もない |
| CLI の help | 一部 | V3 |
| `docs/tech-debt/README.md` | 一部 | V4、V5、V6 |
| 仕様と計画の印、Open questions | 遅れ | I5 |
| Go の doc comment | 一部 | self-review の L2(`ResumeOutcomeFailed` と `Resume` の doc は記録の失敗で failed と書くが、コードは Outcome を変えない)は直っていない |

### 所見

- V1(LOW、FR-10、AC-12): `/director` skill の「止まった座席を続ける」の 3 の `waiting` の項は、理由は伝えるが `ralph org read --org-id <org> --seat <座席>` を添える指示がない。`skipped` の項にはある。4 つの写しに 1 文を足せば済む。
- V2(LOW): README.md の `ralph org spawn/start/.../followup` の行(131 行)に `resume` がなく、org の節に escalate、watch、merge-order、merge、cleanup のような `resume` の段落もない。`[org].api_wait_minutes` も出てこない。FR-11 の一覧に README は入っていないが、README は動詞を列挙している。
- V3(LOW): help が振る舞いより狭い。(1) `ralph org resume --help` は「It exits 1 when typing into a seat failed or the manifest could not be read」とだけ書くが、`resumed` の記録に失敗しても 1 になる(`Errs` に入る。org の skill と org runtime の仕様の改訂はそう書いている)。(2) `ralph org send --help` は断る場合(idle の待ち、watch gate、承認)を並べるが、新しい断り(別の send か resume がロックを持ったまま `--timeout-ms` が尽きる)を書いていない(`lock` の語は 0 回)。`/org` skill の send の行には書いてある。
- V4(LOW): tech-debt の 448 行(上限の画面は実機で見ていない)は S10 のあと古い。S10 は claude の週の上限の画面を実機で見て、S11 がその項目の出どころを `seen:` にし、まとまりの区切りを直した。この行の Trigger(「org の座席が初めて上限で止まったとき」)は満たされ、一部は払われたが、行は 387a290a のままである。445 行の Trigger(「上限の画面を初めて実機で見たとき: 項目ごとに照合を締める」)も同じく満たされ、照合は締めておらず、記録もない。
- V5(LOW): この PR が満たしたのに、引き継ぎの記録がない Trigger。188(a)(leader の prompt)、226(a)(b)(`Send` の変更。`verbs.go:403-405` の「the fail-closed budget check right after that wait」は、あいだにロックが入ったのでさらに古くなった)、262(`/director` skill の「受信箱の処理」の節の中の「件の処理」と「答える順」)、276(`StartDirector` の herdr の呼び出し)、319(`director_start.go` 1307 → 1311 行)、365(`Send` の変更。`spawn.go` 3059 → 3093、`verbs.go` 2352 → 2441)、372(implementer と reviewer の役割 prompt)、379(「答える順」の send の手順)、414(`config.go` 1121 → 1138)、435(`warnNoPaneEnv` の doc と、その警告の文を作る `paneEnvConsequence` の 3 つの項)、436、441、442(`placeSeatPane`、`tab_create` の枝を含む)。境界のもの: 137(偽物の herdr に送信を止める hook を足した。`fakeResumeHerdr` の `onSendText`)、88 の (2)(`spawn_test.go` のコメントの変更)。どれも払う必要はなく、/sync-docs が「2026-11-07: docs/spec-org-seat-resume hit ... Carried over」の形で記録すればよい。
- V6(LOW): self-review は「L1 か L10 を直さずに残すときは register に行を足す」と書いたが、L1(`Send` がロックを待つあいだに resume が手番を始めた座席にも打つ)も L10(enter_only の照合も部分一致)も、直しも行もない。445 行の Impact を広げるか、行を足すかを /sync-docs か cross-review の後で決める。

### INFO

- I1(FR-2 の 1): 動いているが pane の id がない座席(spawn の途中)は、出力に行が出ない。理由を付けて「飛ばした」とは出さない。director は pane がなければ `no pane_id recorded` で飛ばした行が出る。help と skill は「pane のある動いている座席」と書いているので文書とは合う。仕様の文がどちらとも読める。
- I2(FR-7): 終了コード 1 は、仕様の「送信の失敗か台帳を読めない」に加え、`resumed` の記録の失敗と、ロックのファイルを開けないときにも出る。`director.jsonl` を読めないときは警告だけで 0。
- I3(FR-3): 仕様の一覧にない待ち中の文が 2 つある(claude の `Waiting for API response`、codex の `waiting to retry`)。どちらも打たない側に倒れる。
- I4(AC-14): 「2 つ目はロックの中で読み直す」のは resume だけで、2 つ目が `Send` のときは直列になるだけである(計画の D8、仕様の FR-5a の文はこちら)。`Send` が先で resume が後の順のテストはないが、2 つの resume のテストと同じ道を通る。
- I5: 仕様の FR と AC、計画の AC はすべて `[ ]` のまま(計画の Affected areas は仕様の印を付けると書いている)。計画の Open questions の「S10 でも当てられない」と、仕様の Open questions の 1 つ目の「画面の文はまだ見ていない」は、S10 で週の上限の画面を見たので古い(仕様の FR-2 の改訂は見たことを書いている)。印を付けるとき、AC-10 は「作業を最後まで進める」が未確認と書き添える。

## Observational checks

- 実機の記録 `docs/evidence/org-seat-resume-live-2026-11-07.md`(S10)を読み、AC-10 と AC-15 の数と手順を突き合わせた。分離(専用の herdr、`env -i`、ユーザーのサーバーに送っていない)も記録にある。
- 回数の実測の記録 `docs/evidence/org-seat-resume-retries-2026-11-07.md`(S2)の式を計算でたどった(1 → 6、2 → 8、3 → 10、30 → 55、60 → 105)。live の 2 分の座席は 8 回だった。
- HEAD の binary を `go build` で作り、`ralph org resume --help` と `ralph org --help` と `ralph org send --help` を読んだ。

## Coverage gaps

- Go のテストの結果は見ていない(/test)。上の表のテストの欄は、テストがあり assert が AC を言っていることまで。
- 実機で確かめていない道: Enter だけ、codex の座席、headless の director、leader とほかの座席の順、応答が返らない通信断、S11 のあとの binary。
- 決定的な検査がないもの: README の動詞の列挙と `ralph org` の cobra の動詞の一覧の突き合わせ(V2 は 2 回目以上)。最小の検査は、`newOrgCmd` の子の名前(hidden を除く)が README の `ralph org .../...` の行にすべて出ることを見る Go のテスト 1 本。

## Verdict

pass

- Verified: FR-1〜FR-9、FR-5a、FR-11、非機能要件 5 つ、AC-1〜AC-9、AC-11、AC-13〜AC-15 をコードとテストの assert で、AC-8 と AC-15 を実機の記録で確かめた。静的解析の全項目(gofmt、go vet、golangci-lint、staticcheck、shellcheck、sync の 4 つ、template purity、secret scan)。計画の digest は `0893ac30252e` のまま。origin/main との衝突はない
- Partially verified: AC-10(作業を最後まで進めることは週の上限で未確認。ralph の側は実機で確認済み)。FR-10 と AC-12(V1)。live の確認は S11 の前の binary
- Not verified: Go のテストの結果と `run-test.sh`(/test)。上の「Coverage gaps」の実機の道
- 所見: LOW 6 件(V1〜V6)、INFO 5 件(I1〜I5)。どれも merge を止めない
- /sync-docs に回すもの: V1(4 つの写し)、V2、V3(help と、それを固定するテスト)、V4、V5、V6 の tech-debt の記録、I5 の印と Open questions、self-review L2 の doc
