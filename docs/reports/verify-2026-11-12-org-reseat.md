# Verify report: org-reseat(`ralph org reseat`、#312c の 1 本目)

- Date: 2026-11-13(JST。UTC では 2026-11-12)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(spec `docs/specs/2026-11-12-org-reseat.md`)
- Verifier: verifier サブエージェント(`/verify`)。パイプライン 2 回目(`cycle-count.json` の `cycle` は 2)。テストは走らせていない(`/test` の担当)
- この report は、1 回目(cycle 1)の report(af927216)を置き換える。1 回目の本文は `git show af927216:docs/reports/verify-2026-11-12-org-reseat.md` で読める。1 回目の drift V1〜V8 は sync-docs(a40c646f)が直したので、この report の drift は V9 から番号を続ける
- Scope: `git diff origin/main...HEAD`(base 4220ea7c、HEAD c50d79d6、33 ファイル)。1 回目からの差分のうち読んだのは、cross-review の triage(2bc28bdc、ACTION_REQUIRED 3 件)、その直し adc3e205・e9461b79・0468d96f(`internal/org/reseat.go`、`internal/cli/org_reseat.go` とそのテスト 2 本)、sync-docs の a40c646f、origin/main の merge 9a295a84(#308)と 8fbd7118(#310)が reseat に触れる所、self-review の cycle 2 と差分の節(f754188f、c50d79d6)。判定の範囲は 1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、計画の AC-1〜AC-12 と AC-15〜AC-21。FR-9、FR-10、AC-13、AC-14 は 2 本目なので対象外
- Evidence: `docs/evidence/verify-2026-11-12-org-reseat.log`(git の対象外。`run-static-verify.sh` の全出力、HEAD から作った binary の `ralph org reseat --help`、`read` と `send` が `ralph.toml` を読むことの probe、行数、plan の digest、skill の 8 つの写しの sha256)。ラッパー自身の記録は `docs/evidence/verify-2026-11-12-174707.log`

## cross-review の 3 件の反映

| # | 直し | 判定 | 根拠 |
| --- | --- | --- | --- |
| 1 | `--seat` が、最初の spawn が `spawned` に届かず reseat が止めた座席を断る | 反映済み | `reseatLifeSeats`(`internal/org/reseat.go:240`)が、今の命でその座席のどれかの event の Details が `reseat=<id>` で終われば `touched` を立て、`startable()`(`reseat.go:204`、`spawned \|\| active() \|\| touched`)を `reseatTargets`(`reseat.go:599`)と `reseatNoActiveSeatErr`(`reseat.go:650`)の両方が呼ぶ。`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`(出た打ち直しで立つ。手で止めた座席は断られたまま)、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes`(断りが並べる座席は `--seat` が受け付ける座席だけ) |
| 2 | leader を立て直さない reseat が途中で失敗すると DECISION が消える | 反映済み | `finish`(`reseat.go:1187`)は、`plan.Start` に leader がいれば DECISION を作らず、いなければ `Outcome == started` の座席だけで作る。CLI は失敗のときも `Retry` のあとに出す(`printOrgReseatResult`、`internal/cli/org_reseat.go:403`)。`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、CLI は `TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |
| 3 | `--config` が打ち直しのコマンドから落ちる | 反映済み | `runOrgReseat`(`org_reseat.go:190`)が `filepath.Abs` した `--config` を `HintConfigPath`(`reseat.go:86`)に入れ、`reseatVerbCommand`(`reseat.go:569`)が `ShellWord` で足す。`reseat.go` の中で `ralph org` の reseat・stop・spawn・start を組む所は、打ち直し(`reseatRetryCommand`)と断り 3 つ(`reseatNoBindingErr`・`reseatNoActiveSeatErr`・`reseatCustomRoleErr`)のすべてがこの関数を通る(grep)。DECISION の `send`(`ReseatDecisionCommand`)と CLI の `read` は通らない。上の CLI のテストが、相対の `--config` と別の承認方式の `./ralph.toml` で、打ち直しを別のディレクトリから打って通す |

## Spec compliance

各 AC は、コードを読んで判定し、それを固定するテストの名前を添えた。テストの合否は `/test` が確かめる。1 回目からの変化がない行は、1 回目と同じ根拠で、関数名と行番号だけを HEAD に合わせた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 3 席の停止と leader → implementer → reviewer の起動、`scope_reserved` が増えない、承認の検査が通る | 満たす | `Reseat`(`reseat.go:1055`)、止める順は `PlanReseat`(`reseat.go:371`)の `stops`。`TestReseat_StopsAndStartsEverySeatInOrder`。実機は leader と reviewer の 2 席で合格(AC-12) |
| AC-2 leader が新しい workspace の最初の pane | 満たす | `TestReseat_LeaderIsTheFirstPaneOfANewWorkspace`。実機でも root の pane |
| AC-3 `--seat implementer --driver codex --model` と DECISION の行 | 満たす | `reseatSeat`(`reseat.go:755`)、`finish`。`TestReseat_OneSeatOnAnotherDriver`、`TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader`。cross-review 2 の直しで、途中で起動が失敗したときも立った座席の DECISION が出る(上の表) |
| AC-4 止める前の断り(計画の変更、自分の節か前文、計画のファイル、見張り、worktree の記録) | 満たす | `PlanReseat` の 5〜9、`reseatWorktreeRecordErr`(`reseat.go:773`)。`TestPlanReseat_RefusesWhatTheStartsWouldRefuse`、`TestReseat_ARefusalWritesNothing`。見張りの読み方は 1 回目の I3 のまま(start と同じく、無ければ起動し、動かせないときに断る) |
| AC-5 結びつきの無い org の断りと stop・spawn の打ち方 | 満たす | `reseatNoBindingErr`(`reseat.go:581`。今は `--config` も付く)。`TestPlanReseat_RefusesAnOrgWithoutABinding`、`TestOrgReseat_OrgWithoutBindingRefused` |
| AC-6 active な座席が無いときの断りと `--seat` の打ち方 | 満たす(注) | `reseatNoActiveSeatErr`。0468d96f から、並べるのは `--seat` が受け付ける座席(`startable()`)だけで、`spawned` も `touched` も無いまま手で止めた座席は並べない。その座席は FR-2 で `--seat` が断るので、並べないほうが AC の目的(打ち方を出す)に合う。`TestPlanReseat_NoActiveSeatNamesTheStoppedSeats`、`TestPlanReseat_NoActiveSeatWithoutStoppedOnes`、`TestPlanReseat_NoActiveSeatNamesOnlySeatsTheSeatFlagTakes` |
| AC-7 座席の pane から打った reseat の断り | 満たす | `reseatSelfPaneErr`(`reseat.go:548`)。`TestPlanReseat_RefusedFromASeatPane` |
| AC-8 pane が無い座席の立て直し | 満たす | `TestReseat_SeatsWhosePanesAreGone`。実機は `pane=closed` で合格 |
| AC-9 `--dry-run` の出力 | 満たす | `printOrgReseatDryRun`(`org_reseat.go:373`)。`TestOrgReseat_DryRunPrintsThePlanAndStartsNothing` |
| AC-10 `stop_failed` で何も立てない | 満たす | `stopAll`(`reseat.go:1118`)。`TestReseat_AStopThatFailsStartsNothing` |
| AC-11 フラグの組み合わせの断り | 満たす | `checkReseatFlags`(`reseat.go:522`)はロックより前。`TestPlanReseat_RefusesBadFlags`、`TestReseat_AFlagErrorTakesNoLock`、`TestOrgReseat_FlagRefusals` |
| AC-12 実機で AC-1 と AC-8 を 1 回ずつ | 一部 | `docs/evidence/org-reseat-live-2026-11-12.md`。AC-8 は合格、AC-1 は 2 席で合格。implementer と承認の検査は実機で見ていない。binary は dd204370 で、切れた reseat の座席、SIGPIPE、cycle 2 の 3 つの直しのどれよりも前 |
| AC-15 `reseat=<id>` の note と `reseated` 1 件 | 満たす | `noteDetails`(`internal/org/verbs.go:1314`。#308 の merge のあとも `Stop` が `noteDetails(details, p.Note)` を呼ぶ)、`reseatedDetails`(`reseat.go:1248`)。`TestReseat_RecordsItsIDOnEveryStopAndStart` |
| AC-16 skill の 4 つの写しと help が動詞とアカウントの移し方を言う | 満たす | 動詞とアカウントの移し方は、org skill の表の行と「座席の立て直し」、director skill の「座席を立て直す」、help が言う。8 つの写しは組ごとに sha256 が一致(org e5af028c0e8b、director d4e690f61a72)。cycle 2 の直しの中身(`touched`、途中の失敗のあとの DECISION、`--config`)を skill はまだ書いていない(下の V12、V13) |
| AC-17 `reseat.lock` と止める直前の読み直し | 満たす | `takeReseatLock`(`reseat.go:984`)、`stopAll` の `reseatSeatNow`(`reseat.go:1215`)。`TestReseat_RefusedWhileAnotherReseatHoldsTheLock`、`TestReseat_ASeatChangedBeforeItsStop` |
| AC-18 起動の失敗のあと、出た打ち直しで立つ | 満たす | `reseatRetryCommand`(`reseat.go:1271`)、`startable()`。cross-review 1 と 3 の直しで、最初の spawn が終わらなかった座席と `--config` を明示した reseat でも、出たコマンドがそのまま通る。`TestReseat_RetryAfterAFailedStart`、`TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded`、`TestReseat_DecisionForTheSeatsStartedBeforeAFailure`、`TestReseatRetryCommand`、`TestOrgReseat_AFailedStartPrintsTheRetry`、`TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig` |
| AC-19 `--dry-run` は見張りも server も起動しない | 満たす | `reseatDryRunGates`(`reseat.go:815`)、`wireOrgReseatDryRunGates`(`org_reseat.go:323`)。`TestPlanReseat_DryRunStartsNoWatchOrServer` |
| AC-20 切れた reseat の座席と signal | 満たす | `reseatLifeSeat.cutOff`、`orgReseatHoldSignals`(`org_reseat.go:288`)。`TestPlanReseat_FinishesAReseatThatWasCutOff`、`TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce`、`TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt` |
| AC-21 独自の役割の座席の断り | 満たす | `reseatCustomRoleErr`(`reseat.go:701`)。`TestPlanReseat_RefusesACustomRoleSeat` |

仕様の FR と D の対応:

- FR-1・D1: 1 回目のまま。`--timeout-ms` は仕様に無いが、org skill の表の行が書いている
- FR-2・D2・D4: コードは、FR-2 の「今の命で一度 `spawned` になった座席」に加えて、今 active な座席(1 回目の I4)と、reseat が触った座席(`touched`、cross-review 1)を受け付ける。後者は、FR-2 の「一度も立っていない座席を名指しすると断る」の字面より広い。FR-2 の目的(出た打ち直しを通す)に沿った cross-review の直しなので、コードの不適合とはせず、仕様と計画の文書の drift として V9 に回す
- FR-3: AC-7
- FR-4・D3: 1 回目のまま。cross-review の直しは検査の中身を変えていない(`PlanReseat` の変更は断りの関数への引数 2 か所と doc comment だけ)
- FR-5a・FR-5・FR-6・FR-7・FR-8・D5: AC-17、AC-10、AC-18、AC-3、AC-15、AC-9、AC-19。D5 の「leader も立て直したときは行を出さない」は、`finish` が `plan.Start` に leader を見つけた時点で戻ることで守られる。FR-6 と FR-7 が言う出力に、`--config` の引き継ぎと、途中の失敗のあとの DECISION が足された(V10)
- FR-11・D6: 1 回目のまま
- 非機能要件: 1 回目のまま。cross-review の直しは pane の環境に何も足していない(`SpawnParams` に env の欄は無い)。`reseated` と `reseat=` の扱いも変わっていない
- origin/main の merge: #308(`Send` の画面の判定)は reseat の道を変えない(reseat は `Send` を呼ばず、`Stop` の `Note` と `noteDetails` は残っている)。#310 は codex の座席を立てるときの hook の承認の通知(`printCodexHookTrustNotice`)を `spawn`・`start` の 2 つの形・`director start` に足したが、reseat は呼ばない(self-review の L14)。reseat の仕様はこの通知に触れず、#310 の文書(`.codex/README.md:206-207` とその template の写し)は通知を出す動詞を spawn・start・director start と書くので、どちらの文書も偽にはならない(I8)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(exit 0) | 1 回で通った。タイムアウトは無かった |
| ├ shellcheck、`sh -n`、hook の JSON(`jq -e`)、Codex の hook の single-source と検出の smoke | pass | 今回の diff に shell は無い |
| ├ `scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| ├ `scripts/check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan 参照 | pass | |
| ├ golang: `gofmt -l`、`go vet ./...`、`golangci-lint run ./...`、`staticcheck ./...` | pass | `gofmt: ok`、golangci-lint `0 issues.`、vet と staticcheck は出力なし |
| └ `secret-scan-branch.sh` | pass | `4220ea7c..c50d79d6` clean |
| `go build ./cmd/ralph` と `ralph org reseat --help`、`ralph org --help` | pass | help の本文は evidence の log にある。`org --help` に `reseat` の行がある |
| `./scripts/plan-visual.sh digest` | 一致 | `3058f6f66b5b`。計画の `- Approved:` の digest と同じ |

## Documentation drift

すべて `/sync-docs` に回す LOW。どれもコードの動きは変えない。V14 だけは Go の文字列(help)の直しで、直すならパイプラインは `/self-review` からやり直しになる。

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| V9 仕様の FR-2 と D2 の注、計画の P7 | No(LOW) | FR-2 は「今の命で一度も立っていない座席を名指しすると断る」、P7 は「今の命で一度も立っていない座席は断る」と言う。コードは、最初の spawn が `spawned` に届かなかった座席も、reseat がその座席を止めたか立てようとした(`reseat=<id>` の note がある)なら受け付ける(`touched`、adc3e205)。今 active な座席(`spawned` の無い `spawn_started`)も受け付ける(1 回目の I4)。D2 の注は、AC-6 の断りが `--seat` の受け付ける座席だけを並べることも言わない。P7 は digest の範囲に入るので、直すと digest が変わる。前例(7ed1fd28)のように承認を記録し直す |
| V10 仕様の D5・FR-6・FR-7 | No(LOW) | (a) D5 と FR-7 は DECISION の行を「非 leader だけを立て直したとき」とだけ言う。コードは、leader が対象に無ければ、途中で起動が失敗したときも、立った座席を名指す DECISION を出し、打ち直しはその残りの座席について自分の DECISION を出す(adc3e205)。(b) FR-7 は「`--state-dir` を渡したときは、出す行にも付ける」とだけ言う。コードは、打ち直しと断りの reseat・stop・spawn・start のコマンドに、渡された `--config` を絶対パスにして付ける(e9461b79)。DECISION の `send` と `read` には付けない |
| V11 `README.md:300`(reseat の段落) | No(LOW) | `--seat` の文(「names a seat spawned since the org's last disband, active or not」)が、reseat が触った座席を言わない。打ち直しの文(「the reseat command that starts the rest」)が、`--config` と `--state-dir` の引き継ぎを言わない。DECISION の文(「When it started only seats other than the leader」)は、途中の失敗のあとも含めて今のコードと合う(leader が対象なら、leader が立てばその文に当たらず、leader が立たなければ何も立たない)ので、直さなくてよい |
| V12 org skill(4 つの写し)の表の `reseat` の行(`.claude/skills/org/SKILL.md:371`)と「座席の立て直し」(`:1612` から) | No(LOW) | (a) 手順 1 の「一度も立っていない座席は断る(`ralph org spawn` で立てる)」は、reseat が触った座席について偽になった(V9 と同じ)。表の行の `--seat` の文も同じ。(b) 手順 1 と表の行の「止めた座席を `--seat` で名指しした打ち方を出して断る」は、0468d96f から、`--seat` が受け付ける止めた座席だけを並べ、止めた座席がなければ立て直せる座席を最新の state つきで並べる。(c) 途中で失敗したときの段落(`:1705` から)が、打ち直しが `--config` と `--state-dir` を引き継ぐことと、leader が対象に無ければ途中の失敗のあとも立った座席の DECISION が出ることを言わない。「leader 以外の座席だけを立て直したとき」の段落(`:1713` から)も、失敗のときを言わない |
| V13 director skill(4 つの写し)の「座席を立て直す」の 6 と 7(`.claude/skills/director/SKILL.md:1458-1475`) | No(LOW) | 7(途中で失敗したとき、終了コード 1)は、出力の最後に立った座席の DECISION が出ることがあり、それを 6 と同じく leader の pane を読んでから送ることを言わない。打ち直しも成功すれば自分の DECISION を出す。cross-review 2 の直しは、この知らせを director が使うことで効くので、V9〜V14 のうち人と director の動きにいちばん関わる |
| V14 `ralph org reseat --help` と `ReseatParams.HintConfigPath` の doc(self-review の L15 と同じ所) | No(LOW) | (a) help の「The `ralph org read` and `ralph org send` in its output read no ralph.toml and do not repeat it.」は偽。どちらも `resolveOrgConfig` で `./ralph.toml`(か `--config`)を読む(`newOrgSendCmd` は `newOrgRuntimeAt`、`newOrgReadCmd` は `newOrgRuntime`)。probe: 一時ディレクトリに壊れた `ralph.toml` を置くと、`ralph org read --seat leader` と `ralph org send --dry-run` はどちらも `org: load config: ralph.toml: toml: expected character ]` で終了コード 1(一時の state dir には何も書かれなかった)。読んだ値で DECISION の送り方が変わる所は見つけていない。`TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(`internal/cli/org_reseat_test.go:837-838`)がこの文を固定している。(b) 「The reseat, stop, spawn, and start commands reseat prints ... in its output or in a refusal, repeat the --config」は、reseat が `res.Err` のまま出す `Spawn` の error の中の `ralph org stop`(`stopSeatCommand`、`internal/org/spawn.go:2343`。呼び元は `spawn.go:2335`、`2863`、`3012`、`3051`)に当たらない。`HintConfigPath` の doc はこの例外を書くが、help は書かない。(c) 「is refused and names the stopped seats」は、V12 (b) と同じく、今は `--seat` が受け付ける座席だけを並べる |
| V15 `docs/tech-debt/README.md` の reseat の 3 行 | No(LOW) | cycle 2 の直しのコミットが Trigger に当たったのに、持ち越しの注が無い。485 行目(「Seven small findings」)の (b)(`PlanReseat` の断りの文: 0468d96f の `reseatNoActiveSeatErr`、e9461b79 の `--config`)、(d)(`PlanReseat` の変更)、(f)(help と `reseatLifeSeats` の変更)。486 行目(行数)は `reseat.go` と `reseat_test.go` の変更で当たり、行数は 1,230 → 1,307、1,637 → 1,855 に増えた。`internal/cli/org_reseat_test.go` は 749 → 852 行で 800 行を超え、どの行にも無い。487 行目(Test gaps)の (i)(j)(`runOrgReseat` の変更、e9461b79。reseat の行は legacy-ledger と scaffold の注のテストに今も無い)と、(c)(d)(`printOrgReseatResult` の doc comment だけの変更、adc3e205。本体と `reseatSeatNow`、停止と起動の順は変わっていない。新しい CLI のテストは `left stopped:` を見るが、`stop failed:`・`left running:`・`left as it is:`・`not started:` は今も見ない)。self-review の持ち越し L13〜L16(L14 を含む)も register に無い。485 行目 (f) の本文の「the refusal of AC-6 names the stopped seats」は、今は `--seat` の受け付ける座席だけ。sync-docs が V12・V13 で skill の reseat の行と節を直すと、487 行目 (h) にも当たる |
| V16 計画の Progress checklist | No(LOW) | 最後の項が 1 回目の sync-docs で、cycle 2 の項(cross-review の triage 2bc28bdc と ACTION_REQUIRED 3 件、直し adc3e205・e9461b79・0468d96f、merge 9a295a84・8fbd7118、self-review の f754188f と c50d79d6)が無い。Progress checklist は digest の範囲の外 |
| `AGENTS.md:91`、`docs/quality/quality-gates.md`(2 つの写し)、`internal/cli/org_legacy_ledger.go` のコメント、`--config` と `--seat` のフラグの usage | Yes | cycle 2 の直しは、ここに書いた中身(reseat の役割、見張りと server の gate、台帳の扱い、読む設定、既定の対象)を変えていない |
| `docs/specs/2026-08-01-org-runtime.md` の「2026-11-12 改訂」 | Yes(I7) | 要約の水準で、偽の文は無い |

## Observational checks

- HEAD から `go build` した binary で `ralph org reseat --help` と `ralph org --help` を見た。表示は V14 のとおり
- `ralph org read` と `ralph org send --dry-run` を、壊れた `ralph.toml` のある一時ディレクトリで、一時の `--state-dir` を付けて打った(V14 (a))。herdr にも台帳にも触れていない。一時ディレクトリは消した
- `./scripts/plan-visual.sh digest` で計画の digest を計算し直した(一致)
- INFO:
  - I7 `docs/specs/2026-08-01-org-runtime.md` の改訂 (b)(c) は、DECISION を「leader 以外の座席だけを立て直したとき」と書き、`--config` に触れない。要約で、V11 の README の文と同じく偽にはならない
  - I8 reseat は #310 の hook の承認の通知を出さない(self-review の L14)。今の文書は偽ではないが、L14 を払うときは `.codex/README.md:206-207` と `templates/base/.codex/README.md` の動詞の並びに reseat を足す
  - I9 AC-12 の実機の binary(dd204370)は、cycle 2 の 3 つの直しよりも前。直しの道(`touched` の座席の打ち直し、途中の失敗のあとの DECISION、`--config` の引き継ぎ)は実機で見ていない
  - I10 1 回目の I5(5 か所のコメントが `docs/plans/active/...` を指す、self-review の L6)と I6(行数、V15 で更新)は残っている。self-review の L13(テストのコメントが cross-review の triage を番号で引く)も同じ型で、cycle 2 の `/cross-review` が triage を書き直すと引用が読めなくなる

## Coverage gaps

- テストは走らせていない。上の判定は、コードを読み、テストが該当の動きを固定していることを読んで出した。合否は `/test` で確かめる
- 実機では、AC-1 の implementer と承認の検査、AC-3、AC-10、AC-17、AC-18、切れた reseat、signal、cycle 2 の 3 つの直しを見ていない(テストだけ)
- 止める直前の読み直しと Stop の書き込みのあいだの窓、Stop が座席の spawn のロックを取らないことは、計画の Risks のとおり残る(tech-debt 361 行目)
- 足すとよい確かめ: V14 (a) の型(help が「X は Y を読まない」と言う)を止めるには、help の文を固定するテストに、`read` と `send` を壊れた `ralph.toml` のディレクトリで打つ 1 件を足すのがいちばん小さい。V12 と V13 の型は、1 回目の Coverage gaps に書いた「skill の reseat の行と help の既定の対象を比べるテスト」(tech-debt 487 行目 (h))が止める

## Verdict

- Verified: 静的検査はすべて pass。cross-review の 3 件はコードとテストに反映されている。1 本目の FR-1〜FR-8、FR-5a、FR-11、D1〜D6 と D2 の注、AC-1〜AC-11、AC-15〜AC-21 は、コードとそれを固定するテストの対応で満たす(FR-2 の字面より広い `--seat` の受け付けは V9)。skill の 8 つの写しは組ごとに揃い、計画の digest は承認のときと同じ
- Partially verified: AC-12(実機は AC-1 を 2 席で確かめ、implementer と承認と cycle 2 の直しは見ていない)
- Not verified: テストの実行(`/test`)。実機での AC-3、AC-10、AC-17、AC-18、切れた reseat、signal、cycle 2 の直し
- 判定: pass。drift は LOW 8 件(V9〜V16)で、すべて `/sync-docs` に回す。V14 は Go の文字列なので、直すか持ち越すかは orchestrator が決める
