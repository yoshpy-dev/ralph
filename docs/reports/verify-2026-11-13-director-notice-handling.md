# Verify report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(issue #318)
- Verifier: verifier subagent(Claude Code)
- Scope: `origin/main`(bb416c48)から HEAD(0d345c45)までの 12 ファイル。director の skill の 4 写し、`internal/org/prompts/director.md`、`internal/org/prompts/leader.md`、新しいテスト 2 ファイル、`docs/tech-debt/README.md`、計画と self-review の記録。本番の Go のコードは変わっていない
- Evidence: `docs/evidence/verify-2026-11-13-director-notice-handling.log`(gitignore 対象。`run-static-verify.sh` の保存した写しは `docs/evidence/verify-2026-11-13-035747.log`)
- Cycle 2: HEAD 3175f3a5 の確認は末尾の「Cycle 2」節に書き、今の判定は最後の「Verdict」節に置いた。「Spec compliance」から「Cycle 1 verdict」までは cycle 1(HEAD 0d345c45)の記録で、行番号もその時点のもの

AC1 と AC4 は、計画の D3 の最初の文面ではなく、HEAD の文で判定した。読み直す skill は、人が従った行の `<dir>` のもの(S4・S5)。伝えた行の `<dir>` が違い、人がどれを upgrade したかを言わなかったときは、読み直す前に人に尋ねる(S6)。以下の行番号は `.claude/skills/director/SKILL.md` のもので、ほかの 3 写しは同じ中身(`cmp` で一致)。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 版の知らせを見たときの動きが skill の 4 写しにあり、「昇格」の 2 と出力先の start から小節を指す | 満たす | 小節「人に向けた知らせの行」は 1443-1489 行で、「進捗の報告」(1414 行)の中、「merge と auto_merge」(1491 行)の前にある。対象の動詞、行の形、`<dir>` の定義は 1447-1455 行。人に 1 度伝えることと 3 つの値は 1456-1457 行。linked worktree の status の行の扱いは 1459-1461 行、流し方の例を書かないことは 1461 行、同じ `<dir>` と版の組で繰り返さないことは 1461-1462 行。`ralph upgrade` を打たないことは 1463-1465 行。行の `<dir>` の skill を絶対パスで読み直すことは 1466-1470 行、`<dir>` が違うときに先に尋ねること(S6)は 1470-1472 行、`<dir>` の方に従い自分の写しが古いと伝えることは 1473-1475 行、headless の立て直しを勧めることは 1475-1478 行。小節を指す文は、昇格の 2 が 91-94 行(attach の `<dir>` が status と違うことも添えている)、機能ごとの出力先の 3 が 641-642 行 |
| AC2: 同じ小節に hook の知らせの動き | 満たす | 対象の動詞と行の形は 1479-1483 行(`codex seat <座席>`、director start では `codex director`、`; run 'ralph doctor' for` で終わる)。org start の行を人に伝えることは 1484 行、spawn の行が leader の QUESTION で届くことは 1484-1486 行。codex の設定を書き換えず hook を承認しないことは 1487-1489 行 |
| AC3: 「してはいけないこと」に 2 つの禁止 | 満たす | 2039-2041 行の項。`ralph upgrade` を打たないこと、codex の設定を書き換えないこと、hook を承認しないことを書き、小節を指している |
| AC4: headless の director の prompt(herdr と launch_cmd)に小節を指す項と upgrade の禁止 | 満たす | `internal/org/prompts/director.md` の 56-61 行のミッションの項。条件分岐の外にあるので、両方の形に入る。小節を指すこと、頼まれても `ralph upgrade` を打たないこと、codex の設定を書き換えないこと、従った行の `<dir>` の skill を絶対パスで読み直すこと、stop と start での立て直しを勧めることを書いている。S6 の「尋ねる」はこの項にない。headless の director は main worktree のルートで動くので status と org の動詞の `<dir>` が同じになり、self-review もこれで足りると判断した。尋ねる規則は小節の 1470-1472 行にあり、項はそこを指している |
| AC5: leader の prompt に spawn の hook の行の動きがあり、止める規則がその例外を指す | 満たす | `internal/org/prompts/leader.md` の 1335-1359 行(運用規律の、spawn failed: の項の直後)。spawn は止まらず、終了コード 0 なら座席を使える(1335-1338 行)。`rejected:` と `spawn failed:` のときは QUESTION を上げず、直上の項、見張りの項、機能ごとの org の手順 7 に従う(1339-1344 行)。codex の設定を書き換えない(1344-1346 行)。行を変えずに座席 id を添えて QUESTION を上げる(1347-1350 行)。作業は止めない(1351-1353 行)。同じ座席の同じ行は返事の前もあとも上げ直さない(1355-1356 行)。別の座席の行は返事の前でも別の QUESTION にする(1357-1359 行)。「件を上げる」の止める規則の例外は 1209-1212 行 |
| AC6: prompt のテストが director と leader の文を確かめ、skill のテストが 4 写しの小節と禁止の項を確かめる | 満たす(コンパイルまで確認、実行は /test) | `internal/org/prompts_notice_test.go`。`TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`(127 行)は、4 写しで見出しの順、小節の位置、決まった文の順(S6 の文を含む)、CLI の文の写し、コードブロックがないこと、昇格の 2、出力先の 3、してはいけないことの項を確かめる。`TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(205 行)は herdr と launch_cmd の両方の形を、`TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping`(298 行)は leader の prompt を確かめる |
| AC7: `internal/cli` のテストが 2 つの行の決まった部分を小節が引いていることを確かめ、文面が変われば落ちる | 満たす(読んで判断、実行は /test) | `internal/cli/director_notice_skill_test.go` の `TestDirectorNoticeSkill_QuotesTheFragmentsOfTheRealLines`(30 行)。本番の `scaffoldVersionNotice` と `codexHookTrustNotice` で 2 つの行を作り、それぞれが決まった部分を含むこと、版の行が決まった部分と絶対パスの `<dir>` で終わること、4 写しの小節が両方を引いていることを確かめる。CLI の文面が変われば、行の側の確かめが落ちる。小節の終わりの判定は `#`、`##`、`###` の見出しだけで、self-review の L3(受け入れ済みの既知の穴) |
| AC8: `check-skill-sync.sh` と `check-sync.sh` が通る | 満たす | ログの 388-393 行は DRIFTED 0 と PASS(IDENTICAL 172、TEMPLATE_ONLY 10、KNOWN_DIFF 6)。405 行は check-skill-sync の 14 skill の一致 |

issue #318 の受け入れ条件との対応。1 つ目(director が知らせの行を見たときの動きが skill と headless の prompt に書かれている)は AC1 と AC4 で満たす。2 つ目(prompt のテストがその文を確かめる)は AC6 で満たす。issue の 3 つ目の項目(ほかの座席)について、leader、implementer、reviewer の prompt は `ralph status`、`ralph org start`、`director attach` と `director start` のどれも打たないことを grep で確かめた。leader が打つ spawn の行は AC5 の規則が扱う。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | 通過(rc=0、1 分 15.9 秒) | 対象の言語は golang。Go 1.27.3 は go.mod と同じ |
| gofmt / go vet | 通過 | 新しいテスト 2 ファイルはコンパイルが通る |
| golangci-lint | 通過 | ログの 420 行が `0 issues.` |
| staticcheck | 通過 | 指摘なし |
| check-sync.sh / check-skill-sync.sh / check-pipeline-sync | 通過 | AC8 のとおり |
| template purity | 通過 | ログの 410 行 |
| tech-debt の計画参照の検査 | 通過 | |
| secret-scan-branch(bb416c48..0d345c45) | 通過 | ログの 422 行が clean |
| U+FFFD と空白の誤り | 通過 | 変わった 12 ファイルに U+FFFD は 0 件。`git diff --check` も指摘なし |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md` の #313 の行(464-465 行) | 一致 | S1 で RESOLVED の注と取り消し線を付けた。注は S6 の尋ねる規則、leader の規則、cli のテストに触れていないが、まとめとしては足りる(INFO) |
| tech-debt 188 行の (a) | 不一致(LOW) | Trigger は「`RolePromptVars` か leader の prompt の次の変更」。leader.md が 1332 行から 1359 行に増えたので発火している。`RolePromptVars` と `{{SCOPE}}` は変わっておらず、払ってもいない。持ち越しの注がないので、/sync-docs で 2026-11-13 の注を足すよう勧める |
| tech-debt 365 行 | 不一致(LOW) | Trigger は「`Send` か leader の template のテストの次の変更」。新しいファイル `prompts_notice_test.go` に leader の template のテストを足したので発火している。数えている 6 ファイル(verbs.go 2597 行、verbs_test.go 5893 行、prompts_test.go 2722 行、spawn.go 3101 行、spawn_test.go 6408 行、director_skill_sync_test.go 1307 行)は変わっていない。持ち越しの注がないので、/sync-docs で足すよう勧める |
| tech-debt のほかの行 | 一致 | 159、177、195、199、200(g)、206、239、244-252、259、262、269(b)、276、318、319、322、337、338、367、371、377、380、381、384、390、397、421、422、426、466、467、468、480、482 の Trigger は発火していない。200(g) は機能ごとの org の節(118-1094 行)、262 は「受信箱の処理」、384 は運用規律の最初の項の止める規則が、それぞれ変わっていない。426 の対象は節ではなく項なので、増えたことだけを INFO とする |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 不一致(LOW) | 151 行は「director は実装しない。許すのは、分割計画を書くこと、repo を読むこと、ralph と gh の操作に限る」。`ralph upgrade` は ralph の操作なので、#318 はこの許可を狭めている。この spec は issue ごとの director の決定を日付つきの項で残している(161 行の #271、82 行と 117 行の #307、95 行の #312、139 行の #309)が、#318 の項はない。/sync-docs で「2026-11-13 に決めたこと(issue #318)」の項を足すよう勧める。なお、handoff に書かれた `docs/specs/2026-10-07-multi-org-director.md` は存在せず、実際の名前はこちら |
| `docs/specs/2026-08-01-org-runtime.md` | 一致 | 日付つきの改訂の節(2026-11-07、2026-11-12)はあるが、leader の QUESTION で返事まで止める規則はどの spec にも rule にもなく、leader.md にだけある。その例外を足すのに spec の行は要らない(INFO) |
| `--help` の段落 | 一致 | 本番の Go を変えていないので、`orgScaffoldVersionNoticeHelp`、`statusScaffoldVersionNoticeHelp`、`orgCodexHookTrustNoticeHelp` は変わらない。計画の「変えない」という決定のまま |
| AGENTS.md / README.md | 一致 | どちらも director の skill の小節や知らせの扱いを書いていない。README の 133-134 行の知らせの説明は変わらず、新しい文と食い違わない。285 行の director の段落に触れてもよいが、必須ではない(INFO) |
| 計画の受け入れ条件のチェックボックス | 遅れ(INFO) | AC1 から AC8 がまだ `[ ]`。いつもの遅れで、/pr の前に付ければよい |
| 次の release の振る舞いのノート | 未作成(INFO) | `internal/upgrade/notes/<X.Y.Z>.md` は /release で書く。PR 本文に「director が知らせの行を人に伝える」ことを載せるよう勧める |

## Observational checks

- 小節は 47 行で、計画のリスクの欄にある「20 行ほど」を超えている。中身は AC1 と AC2 が求める項目だけで、余分な説明はない(INFO)。
- 計画の Scope はテストの置き場所に `director_skill_sync_test.go` と `prompts_test.go` を挙げていたが、テストは新しい `prompts_notice_test.go` に入った。計画は新しいファイルを認めているので、逸脱ではない(INFO)。
- hook の知らせの対象を「承認されていないか切られている」と書いた文は、codex の設定がない場合も含む。CLI の文と食い違わない。
- leader の規則は spawn の行が座席を立てる前に出ること(`internal/cli/org.go` の 966 行で `rt.Spawn` の前に出す)を踏まえ、spawn が失敗したときは QUESTION を上げないとしている。コードの順序と一致する。

## Coverage gaps

- テストは実行していない。handoff の制約(別のセッションが全体のテストを回している)に従い、`go test` と `./scripts/run-test.sh` は /test に任せた。AC6 と AC7 のテストは go vet でコンパイルが通ることまでを確かめた。
- AC7 の「文面が変われば落ちる」は、テストを読んだ判断と、implementer が S3 で回した変異の記録による。この verify では変異を回していない。
- self-review の L3(小節の終わりを `####` の見出しで判定しない)と N6 は、受け入れ済みの既知の穴として PR に持ち越す。
- director と leader が実際の会話でこの文に従うかは、静的にも単体テストでも確かめられない。

## Cycle 1 verdict

- Verified: AC1 から AC8 を HEAD の文とコードで確かめた。issue #318 の 2 つの受け入れ条件は AC1、AC4、AC6 で満たす。`run-static-verify.sh` は rc=0 で、gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、template purity、secret scan が通った
- Partially verified: AC6 と AC7 のテストはコンパイルまでで、実行は /test が行う。AC7 の「落ちる」は読んだ判断
- Not verified: テストの実行結果(/test)、director と leader の実際の振る舞い
- Verdict: pass。文書の食い違いは LOW 3 件(tech-debt 188 行 (a) と 365 行に持ち越しの注がない。multi-org director spec に #318 の日付つきの項がない)で、どれも /sync-docs で直すかを決める

## Cycle 2(HEAD 3175f3a5)

### Scope(cycle 2)

- 範囲は cycle 1 の verify(`fcd368b0`)のあとの commit。`f65575ad`(origin/main e902705c の取り込み)、`f78e2b1b`(sync-docs)、`8ae5ae1c`(S7)、`7bcef73e`(S8)、`8e8d26b5`(orchestrator の 1 行の直し)と、report と計画の記録の commit。取り込みのあとに変わった本体は、director の skill の 4 写し、`internal/org/prompts/leader.md`、`internal/org/prompts_notice_test.go`、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/tech-debt/README.md`、`README.md`(`git diff --stat f65575ad..HEAD` の 14 ファイルから記録のファイルを除いたもの)
- ブランチ全体(merge-base e902705c から HEAD)で変わった Go はテスト 2 ファイル(`internal/cli/director_notice_skill_test.go`、`internal/org/prompts_notice_test.go`)だけで、本番の Go のコードは変わっていない。`internal/org/prompts/director.md` と `internal/cli/director_notice_skill_test.go` は cycle 2 で変わっていない
- 計画からのずれ: leader.md に、計画の最初の文面にない `HOOK_SEAT:` の決まりと、手順 4 と cap の読み手の除外の文が入った。計画の Progress の 139-141 行に記録がある。AC は HEAD の文で判定した
- 以下の行番号は HEAD のもの。skill の行は `.claude/skills/director/SKILL.md` で、ほかの 3 写しは同じ中身(`.claude` と `templates/base/.claude`、`.agents` と `templates/base/.agents` は `cmp` で一致、`.claude` と `.agents` は先頭 20 行を除いた本文の `diff` で一致)
- Evidence: `docs/evidence/verify-2026-11-13-director-notice-handling-c2.log`(gitignore 対象。`run-static-verify.sh` の保存した写しは `docs/evidence/verify-2026-11-13-064233.log`)。以下の「ログ」はこのファイルを指す

### Spec compliance(cycle 2)

| Acceptance criterion | cycle 2 のあと | Evidence |
| --- | --- | --- |
| AC1: 版の知らせの動きが 4 写しにあり、「昇格」の 2 と出力先の start から小節を指す | 保つ | 版の知らせの部分(1474-1504 行)は cycle 2 で変わっていない。cycle 1 の 1447-1478 行から行がずれたのは、取り込んだ #312a の文が上に入ったため。小節を指す文は、昇格の 2 が 93-95 行、機能ごとの出力先の 3 が 642-643 行で、どちらも変わっていない。cycle 2 で小節を指す所が 1 つ増えた(`### 件の処理` の手順 3、964-967 行。AC2 の行に書く) |
| AC2: 同じ小節に hook の知らせの動き | 保つ | hook の知らせは 1505-1532 行で、項が 3 つに分かれた。`ralph org start` の行は人にそのまま伝える(1510 行)。`ralph org spawn` の行は leader の QUESTION で届き、leader が止まらないので director も人の答えを待たない(1511-1513 行)。行と leader の問いを人に伝えたら、DECISION は送らず、すぐに決まった note で resolve する(1513-1516 行)。note を決まった文にする理由は 1516-1518 行、resolve を send のあとに回さない理由は 1518-1520 行。人があとで答えたら、`ralph org status --org-id <org>` の leader の行の STATE に `(active)` があるかを見て(1521-1523 行)、あれば「答える順」の 1 と 2 で DECISION を送る。DECISION は `TYPE: DECISION` の次の行に `HOOK_SEAT: <座席 id>` を書き、本文を `SUMMARY: hook の QUESTION への答え。<人の判断>` で始める(1523-1527 行)。件は resolve 済みなので「答える順」の 3 は打たない(1528 行)。`(active)` がなければ、送る先がないことを人に伝える(1528-1529 行)。codex の設定を書き換えず hook を承認しないことは 1530-1532 行で、cycle 1 と同じ。`### 件の処理` の手順 3 には 964-967 行の項が入り、本文に `; run 'ralph doctor' for ` を含む QUESTION を小節へ送って「人の答えを待たずに resolve する」と書く。AC2 が求める 3 つ(人に伝える、設定を書き換えない、承認しない)は残っている |
| AC3: 「してはいけないこと」に 2 つの禁止 | 保つ | 2082-2084 行の項。cycle 2 で変わっていない |
| AC4: headless の director の prompt に小節を指す項と upgrade の禁止 | 保つ | `internal/org/prompts/director.md` は cycle 2 で変わっていない。56-61 行の項は条件分岐の外にあり、herdr と launch_cmd の両方の形に入る。headless の director が受信箱で hook の QUESTION に出会ったときは、42-52 行(skill の「受信箱の処理」のとおりに件を処理する)から手順 3 の 964-967 行の項に入る。prompt には「QUESTION には DECISION で答える」という文がないので、先に resolve する規則と食い違わない |
| AC5: leader の prompt に spawn の hook の行の動きがあり、止める規則がその例外を指す | 保つ | `internal/org/prompts/leader.md` の 1339-1369 行。codex の設定を書き換えない(1348-1350 行)、QUESTION で上げる(1350-1355 行)、作業を止めない(1355-1357 行)、同じ座席の同じ行は上げ直さない(1364-1366 行)、別の座席の行は返事の前でも別の QUESTION にする(1367-1369 行)。「件を上げる」の止める規則の例外は 1213-1216 行で、cycle 1 と同じ。cycle 2 で足した文は 3 つ。人の答えは `HOOK_SEAT:` の DECISION で届く(1357-1359 行)。その DECISION はこの QUESTION だけへの答えで、計画の承認、cross-review の上限の QUESTION、ほかに答えを待っている QUESTION や BLOCKED の答えではない(1359-1362 行。BLOCKED は 8e8d26b5 が足した)。director は何も送らずに resolve するので、答えを待たずに進める(1362-1364 行)。DECISION の読み手の側では、手順 4 の頭(251-253 行)と cap の QUESTION の読み手(937-939 行)が、それぞれ 1 文で `HOOK_SEAT:` の DECISION を除く |
| AC6: prompt のテストと skill のテスト | 保つ(コンパイルまで確認、実行は /test) | `internal/org/prompts_notice_test.go`(436 行)。`directorNoticeRules`(90-140 行)は cycle 2 の小節の文を 119-136 行の断片で持つ。`TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`(168 行)の places に手順 3 の項(185-191 行、既存の `directorSkillStep3Bullet` で切る)が入った。`TestRenderRolePrompt_Leader_RaisesTheCodexHookNoteWithoutStopping`(356 行)の `leaderHookNotePhrases` は、`HOOK_SEAT:` の文と resolve の文を 329-335 行に持つ。新しい `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer`(424 行)は、手順 4 と cap の読み手の除外の文を、前後の既存の文との順で確かめる。断片が HEAD の文に順に入っていることを `python3 -I` で別に確かめた(skill の小節 38、手順 3 の項 2、leader の項 27、除外の 2 文、止める規則の 1 文で、欠けも順の誤りもない。ログの 433-438 行)。手順 4 と cap の読み手を固める既存の断片 12 個(`prompts_test.go:694-697`、`prompts_leader_plan_test.go:300-304`、`prompts_xreview_leader_test.go:305-307`)も、挿入のあと空白を除いて連続している(ログの 469 行)。go vet が通るので、テストはコンパイルできる |
| AC7: `internal/cli` のテストが 2 つの行の決まった部分を小節が引いていることを確かめる | 保つ(読んで判断、実行は /test) | テストのファイルは cycle 2 で変わっていない。小節は 2 つの断片を 1 行のまま引いている(1478 行の `run 'ralph upgrade' in <dir>`、1509 行の `...; run 'ralph doctor' for <the key(s)> ...`)。小節には `####` の見出しも fenced block もなく、`### merge と auto_merge`(1534 行)の前で終わる(ログの 439 行)。手順 3 の項(965 行)も `; run 'ralph doctor' for ` を引くが、小節の外にあるので、このテストの判定には入らない |
| AC8: `check-skill-sync.sh` と `check-sync.sh` が通る | 保つ | ログの 389-395 行(IDENTICAL 172、DRIFTED 0、PASS)と 407 行(check-skill-sync の 14 skill の一致) |

### cross-review の所見 1 と 2(cycle 2)

| 所見 | HEAD の文で閉じたか | 閉じている文 |
| --- | --- | --- |
| 1: 開いたままの hook の QUESTION が、見張りに leader の stall を黙らせる | director がいる設定では閉じた。director のない設定では残る(V2-1) | 小節の「行と leader の問い(座席をこのまま使うか止めるか)を人に伝えたら、DECISION は送らず、すぐに `ralph org inbox resolve <id> --note '人に伝えた(hook の承認の知らせ)'` で件を resolve する。」(1513-1516 行)と「件が resolve されないあいだ、見張りは leader が答えを待っているとみなし、leader が止まっても停滞を知らせないので、resolve を、時間切れで終わりうる send のあとに回さない。」(1518-1520 行)。`### 件の処理` の手順 3 の「下の「進捗の報告」の「人に向けた知らせの行」のとおり、人の答えを待たずに resolve する。」(966-967 行)があるので、一般則(人に聞いて答えを待つ)から入った director も小節に行き着く。`orgHasOpenLeaderItem`(`internal/org/watch.go:1220-1242`)が数えるのは resolved でない件だけなので、件が開いている時間は、ほかの件と同じく director が処理するまでに縮む。leader の側は「director はこの QUESTION を何も送らずに resolve するので、この QUESTION の答えは待たずに進めてください」(leader.md 1362-1364 行) |
| 2: 人の答えの前に org が disband すると件が残る | 閉じた | 件は send を待たずに resolve するので、人の答えより先に org が disband しても、resolve できない件は残らない。`InboxStore.Resolve`(`internal/org/inbox.go:469`)は org が動いているかを見ないので、disband のあとでも resolve は通る。人があとで答えたときは「あとで人が答えたら、`ralph org status --org-id <org>` を打ち、SEAT_ID が `leader` の行の STATE の欄に `(active)` が付いているかを見る。」(1521-1523 行)と「付いていなければ、leader が動いておらず送る先がないことを人に伝える。」(1528-1529 行)が、送る先のない答えの扱いを決めている |

### Static analysis(cycle 2)

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed-language scope) | 通過(rc=0、1 分 22.8 秒) | 対象の言語は golang。Go 1.27.3 は go.mod と同じ |
| gofmt / go vet / golangci-lint / staticcheck | 通過 | ログの 421 行が `gofmt: ok`、422 行が golangci-lint の `0 issues.`。go vet と staticcheck は指摘がなく何も出さない。`Skipping` の行もない |
| check-sync / check-pipeline-sync / check-skill-sync | 通過 | AC8 のとおり。check-pipeline-sync は 6 か所とも ok |
| template purity / tech-debt の計画参照 | 通過 | ログの 412 行と 414 行 |
| secret-scan-branch(e902705c..3175f3a5) | 通過 | ログの 424 行が clean |
| U+FFFD と空白の誤り | 通過 | ブランチの差分に U+FFFD は 0 件、`git diff --check` も指摘なし(ログの 454-455 行) |

### Documentation drift(cycle 2)

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| AGENTS.md | 一致 | repo map の `internal/org/` の項は、director の skill の小節も `HOOK_SEAT:` も書いていない。食い違う文はない |
| README.md の director の段落(285 行) | 一致 | cycle 1 の sync-docs が足した文(the director passes it on to the human, a leader raises the one from `ralph org spawn` as a QUESTION, it never runs `ralph upgrade`, edits the codex config, or approves the hooks)は、cycle 2 の文でも正しい。すぐ resolve することと `HOOK_SEAT:` は書いていないが、段落の粒度では要らない(INFO) |
| `.claude/rules/ralph/agent-messaging.md` | 一致 | メッセージの形は `<any other KEY: value header lines>`(62 行)で任意のヘッダを認める。`protocol.Parse` と `splitHeaderLine`(`internal/org/protocol/protocol.go:100-127`、`:171-182`)は空白のないキーをすべてヘッダとして読むので、`TYPE: DECISION` の次の行の `HOOK_SEAT: reviewer` はヘッダになる。この rule が名前を挙げるヘッダは、send が検査する `PLAN:`(105 行)だけ。同じ種類の決まりのヘッダ(`XREVIEW:`、`VISUAL:`、`FORK 1:`)は multi-org director の spec にだけ書かれていて、`HOOK_SEAT:` も同じ spec の 163 行にある。前例と同じ置き方になる。root と template の写しは `cmp` で一致 |
| `docs/specs/2026-10-07-org-multi-org-director.md` の 163 行 | 不一致(LOW、V2-2) | 下の V2-2 |
| `docs/specs/2026-08-01-org-runtime.md` | 一致 | #318 の日付つきの項はない。leader の雛形の「QUESTION を上げたら止める」規則も、DECISION のヘッダの決まりも、この spec にはない。#318 の記録は multi-org director の spec にある(INFO) |
| `/org` skill の受信箱の項(`.claude/skills/org/SKILL.md:520-529`) | 不一致(LOW、V2-1) | 下の V2-1 |
| `docs/tech-debt/README.md` 188 行 (a) の注 | 不一致(LOW、V2-3) | 下の V2-3 |
| tech-debt 262、377、427 行 | 不一致(LOW、V2-4) | 下の V2-4 |
| tech-debt 200 行 | INFO | Trigger の (g)「`leader.md` の機能ごとの org の節を次に変えたとき」は、手順 4(251-253 行)と `### cross-review` の 937-939 行の変更で当たる。(g) は f1b0352e で閉じていて、払うものはない。前の branch は節を変えるたびに注を残している(2026-10-26 の 2 本、10-30、11-01、11-07)ので、そろえるなら sync-docs で 1 文足す |
| tech-debt 365 行 | 一致(INFO) | 注の「436 lines」は HEAD の `prompts_notice_test.go` と同じ(最後に変えた 8e8d26b5 は 1 行を同じ行数で置き換えた)。注が挙げる leader の template のテストは 1 本で、cycle 2 で同じファイルに入った `TestRenderRolePrompt_Leader_HookSeatDecisionIsNotAnotherAnswer` は挙げていないが、数と論拠(数えた 6 ファイルは増えていない)は変わらない |
| tech-debt 495 行 | 一致(INFO) | Trigger は小節の変更のうち「`####` の見出し、または hook の行の末尾のあとの文」。cycle 2 は小節の hook の項を書き換えたが、`####` は足しておらず、hook の行の末尾(1509 行)と `internal/cli/director_notice_skill_test.go` も変わっていない。発火していないと読んだ。ただ括弧は、CLI の行の末尾に足された句とも、小節で行の説明のあとに足した文とも読める。次に注を書くときに言い方を決めるとよい |
| tech-debt のほかの行 | 一致 | 384(運用規律の、座席を止める規則)、426(leader.md に節を足す)、380(「答える順」の send と確かめ)、269 (b)(`director_skill_sync_test.go`)、372(implementer と reviewer の prompt)は cycle 2 で当たらない。#313 の行は RESOLVED のまま |
| `--help` の段落 | 一致 | 本番の Go を変えていないので変わらない |
| 計画 | 遅れ(INFO) | Progress の 138-141 行に cycle 2 のずれ(leader.md の `HOOK_SEAT:` と除外の文)の記録がある。AC1 から AC8 のチェックボックスはまだ `[ ]` |

### Findings(cycle 2)

| ID | Severity | Finding | Recommendation |
| --- | --- | --- | --- |
| V2-1 | LOW | director のない設定では、hook の QUESTION が cycle 2 の前提どおりに扱われない(self-review の C2-7 に 1 点を足したもの)。`[org.director].enabled = false` のとき(multi-org director spec の 101 行、leader.md 1166-1173 行)、escalate は件を記録すると同時に人へ届け、人は `/org` skill の受信箱の項で答える。その項(`.claude/skills/org/SKILL.md:520-529`)は、DECISION を send して pane で確かめてから resolve する順だけを書き、`HOOK_SEAT:` の語も、hook の QUESTION をすぐ resolve する決まりも持たない。(a) 人の答えはヘッダのない DECISION で届き、承認待ちの leader は直しの依頼(leader.md 261-263 行)、cap の待ちの leader は上げ直し(939-941 行)と読む。(b) leader.md 1362-1364 行の「director はこの QUESTION を何も送らずに resolve する」が、この設定では成り立たない。(c) この verify で足す点: 人は答えるまで件を resolve しないので、その間 `orgHasOpenLeaderItem`(`internal/org/watch.go:1220-1242`)は leader が待っているとみなし、leader が止まっても停滞を知らせない。cross-review の所見 1 の穴は、この設定では残る。既定の設定(enabled = true で、見張りが仕事を受け付けている)では起きない。計画の Non-goals は `/org` skill に足さない理由を「人が座席を spawn するセッションでは行が人の目に入る」とするが、director のない機能ごとの org で spawn を打つのは leader なので、この理由は当たらない | 直さずに進めるなら、PR の Known gaps か register に 1 行書く(self-review の C2-7 と同じ勧め)。直すなら、`/org` skill の 520 行の項に「hook の QUESTION は人に伝えたらすぐ resolve し、あとの答えには `HOOK_SEAT:` を付ける」の一句を足す(4 写し。計画の Non-goals を変える) |
| V2-2 | LOW | multi-org director spec の 163 行の「変えたのは `/director` skill の 4 つの写し(「進捗の報告」の「人に向けた知らせの行」、それを指す「昇格」の 2 と「機能ごとの出力先」の 3、「してはいけないこと」の 1 項)」に、`### 件の処理` の手順 3 の項(964-967 行)がない(self-review の C2-8 のまま)。ほかに、leader の除外を「計画の承認とも cross-review の上限の QUESTION の答えとも読まない」とまとめて、ほかの QUESTION と BLOCKED の返事待ち(leader.md 1361-1362 行)を挙げていないこと、人のあとの答えの前に leader の `(active)` を見ることを書いていないことは、要約の粒度なので INFO とする | cycle 2 の sync-docs で括弧に手順 3 の項を足す |
| V2-3 | LOW | tech-debt 188 行 (a) の注は「1,332 to 1,359 lines, ... at e95aa15f, the last commit of that branch that changed the file」と書くが、HEAD の leader.md は 1,369 行で、最後に変えた commit は 8e8d26b5。注が挙げる変更の中身にも、`HOOK_SEAT:` の文と、手順 4 と cap の読み手の除外がない | handoff のとおり cycle 2 の sync-docs で、数、commit、中身を直す |
| V2-4 | LOW | cycle 2 の変更で Trigger が当たったのに持ち越しの注がない行が 3 つある。262 行: Trigger は「`/director` skill の受信箱の節(`受信箱の処理`)の次の変更」で、`### 件の処理`(946 行)は `## 受信箱の処理`(787 行)の中にあり、7bcef73e が手順 3 に項を足した。(1) から (3) は `director_skill_sync_test.go` と `TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` の話で、この branch は触れていないので払っていない。今ある注は #312a の分だけ。377 行: Trigger は「leader.md の機能ごとの org の手順の次の変更」で、7bcef73e が手順 4 の頭に `HOOK_SEAT:` の除外の文を足した。行が求める「手順 4 に、2 行をファイルの編集ツールで書き換えると書く」は入っておらず、払っていない。427 行の (1)、(4)、(7): Trigger は「leader.md の `### cross-review` 小節の次の変更」で、7bcef73e が cap の QUESTION の読み手に 1 文を足した(937-939 行)。返事の `REPORT:` のパスを確かめるコマンドは足しておらず、払っていない | cycle 2 の sync-docs で、3 行に 2026-11-13 の持ち越しの注を足す。そろえるなら 200 行にも足す(上の INFO) |

INFO(cycle 2):

- origin/main は bbe2807d(#328)に進んでいる。`git merge-tree --write-tree origin/main HEAD` で試すと、`docs/tech-debt/README.md` の 2 か所がぶつかる。365 行(長いファイルの行)に両側が注を足した所と、表の末尾に両側が行を足した所で、ほかのファイルはぶつからない(ログの 460-467 行)。main の `agent-messaging.md` の変更(「How send types a message」の節)は send の打ち方の話で、DECISION のヘッダには触れない。main は 365 行が数える `internal/org/verbs.go` も変えている(+22、-4)ので、取り込んだあとは 365 行の数を見直す
- 小節は 65 行になった(cycle 1 は 47 行)。計画のリスクの欄の「20 行ほど」をさらに超えるが、増えた分は cross-review の 2 件を閉じる文と、self-review の C2-3 から C2-5 の直し

### Coverage gaps(cycle 2)

- テストは実行していない。handoff の制約(別のセッションが全体のテストを回している)に従い、`go test` と `./scripts/run-test.sh` は /test に任せた。AC6 と AC7 のテストは、go vet のコンパイルと、断片を HEAD の文に照らした `python3 -I` の確かめまで
- V2-1 の (c) は `watch.go` を読んだ判断で、見張りを走らせて確かめてはいない
- director と leader が実際の会話でこの文に従うか(とくに、`HOOK_SEAT:` の DECISION を承認待ちや cap の待ちの leader が正しく読み分けるか)は、静的にも単体テストでも確かめられない
- origin/main との取り込みは読むだけの試し(`git merge-tree`)で、ぶつかった所の直し方は確かめていない
- self-review の L3 と N6 は、受け入れ済みの既知の穴として PR に持ち越す。C2-7 は V2-1 として上に書いた

## Verdict

- Verified: cycle 2 の HEAD(3175f3a5)で AC1 から AC8 を確かめ、どれも cycle 2 の変更のあとも満たす。cross-review の所見 1 は director がいる設定で、所見 2 は閉じた。`run-static-verify.sh` は rc=0 で、gofmt、go vet、golangci-lint、staticcheck、check-sync、check-pipeline-sync、check-skill-sync、template purity、tech-debt の計画参照、secret scan が通った。テストの断片が HEAD の文に順に入っていることと、既存の断片 12 個が崩れていないことを `python3 -I` で確かめた
- Partially verified: AC6 と AC7 のテストはコンパイルまでで、実行は /test が行う。所見 1 は、director のない設定では残る(V2-1)
- Not verified: テストの実行結果(/test)、director と leader の実際の振る舞い、origin/main bbe2807d を取り込んだあとの木
- Verdict: pass。食い違いは LOW 4 件。V2-1(director のない設定で、hook の QUESTION がすぐ resolve されず、答えに `HOOK_SEAT:` が付かない)は Known gaps か register に書く。V2-2(spec の変えた場所の並び)、V2-3(tech-debt 188 行の注の数と中身)、V2-4(tech-debt 262、377、427 行に持ち越しの注がない)は cycle 2 の sync-docs で直す
