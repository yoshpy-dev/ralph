# Verify report: director-notice-handling

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-13-director-notice-handling.md`(issue #318)
- Verifier: verifier subagent(Claude Code)
- Scope: `origin/main`(bb416c48)から HEAD(0d345c45)までの 12 ファイル。director の skill の 4 写し、`internal/org/prompts/director.md`、`internal/org/prompts/leader.md`、新しいテスト 2 ファイル、`docs/tech-debt/README.md`、計画と self-review の記録。本番の Go のコードは変わっていない
- Evidence: `docs/evidence/verify-2026-11-13-director-notice-handling.log`(gitignore 対象。`run-static-verify.sh` の保存した写しは `docs/evidence/verify-2026-11-13-035747.log`)

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

## Verdict

- Verified: AC1 から AC8 を HEAD の文とコードで確かめた。issue #318 の 2 つの受け入れ条件は AC1、AC4、AC6 で満たす。`run-static-verify.sh` は rc=0 で、gofmt、go vet、golangci-lint、staticcheck、check-sync、check-skill-sync、template purity、secret scan が通った
- Partially verified: AC6 と AC7 のテストはコンパイルまでで、実行は /test が行う。AC7 の「落ちる」は読んだ判断
- Not verified: テストの実行結果(/test)、director と leader の実際の振る舞い
- Verdict: pass。文書の食い違いは LOW 3 件(tech-debt 188 行 (a) と 365 行に持ち越しの注がない。multi-org director spec に #318 の日付つきの項がない)で、どれも /sync-docs で直すかを決める
