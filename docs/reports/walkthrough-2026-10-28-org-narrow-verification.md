# Walkthrough: org-narrow-verification(機能の org のスライスごとの検証を絞り、フルの test を gate の 1 回にする)

- Date: 2026-10-28(UTC)
- Plan: docs/plans/active/2026-10-27-org-narrow-verification.md(/pr で archive に移す)
- Issue: #262
- Branch: perf/org-narrow-verification(この walkthrough の前の HEAD f0d762c1。48 commits、31 files、+2650/-112)
- Base: 622cdcfb(main、「Merge pull request #278」)
- 読む範囲: Go の実行コードは変えていない。中身は go:embed の役割雛形 3 つ(implementer +91/-13、leader +103/-4、reviewer +34/-9)、`/org` skill の文、文書、Go のテスト 3 ファイル、shell のテスト 1 本。
  - 同じ文の写しが 4 か所にある。`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`(`sync-skills.sh` で生成)、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md`。読むのは `.claude/` の側だけでよい。plan の雛形にも 6 つの写しがある(各 1 行)。
  - Go のテストは `internal/org/prompts_verification_test.go`(新規、+908)、`prompts_verification_predate_test.go`(+328)、`prompts_verification_multi_test.go`(+90)。`internal/org/prompts_test.go` は main の版と同じ。shell は `tests/test-slice-verification.sh`(+7)。
  - 記録は計画(+216)、後工程の report 5 本(+465)、insight event(+17)、tech-debt(+8/-5)。
- 後工程は 3 回まわした。1 回目の cross-review が ACTION_REQUIRED を 1 件、cycle 2 の cross-review が 2 件出した。2 回目で cap の 2 に届いたので、`/cross-review` の cap 到達時の選択肢 1 で cap を 2 から 3 に上げ、`/self-review` から全体を回し直した(`cycle-count.json` は 2 のまま)。最後の cross-review の所見は 0 件。記録だけの修正の例外は使っていない

```
何を変えるか → implementer の雛形(S1) → reviewer の雛形(S2) → leader の雛形(S3) → /org skill(S4) → 文書(S5) → レビューで足した直し(5 つの主題) → 後工程とテスト → 確かめていないこと
```

## 1. この PR が何か

- これまで: 機能の org では、implementer がスライスごとに `./scripts/run-verify.sh`(静的検査とテストを全言語で)を打っていた。leader の TASK が検証コマンドを書かないため、雛形の既定がそのまま使われていた。reviewer はレビューの TASK で `run-static-verify.sh` と `run-test.sh` を打ち、PR の前の gate の TASK でも `ralph org gate` が同じ 2 つを走らせる。フルの test は「スライス数 + 2」回走り、このリポジトリの `run-test.sh` は多くが 640〜1,090 秒かかる。
- この PR から、役割ごとに次のようになる。
  - leader は、機能の計画の `## Implementation outline` に、スライスごとの検証コマンドを「— verify:」で書く。implementer への TASK では、そのコマンドを本文の `VERIFY:` の行に写す(S3)。
  - implementer は、`VERIFY:` の行のコマンドをスライスごとに打つ。行がなければ、触ったファイルを覆ういちばん狭い検査を選ぶ。コミットの前に、stage した差分で 5 項目の機械的な確かめを行い、RESULT に打ったコマンドと 5 項目の結果を書く(S1)。
  - reviewer は、レビューの TASK の本文の `CHECKS:` の行にあるコマンドだけを打つ。機能ごとの org の leader は `CHECKS: ./scripts/run-static-verify.sh` だけを書くので、その org のレビューはテストを打たない。フルの test は gate の TASK で `ralph org gate` が 1 回走らせる(S2、S3)。差分のレビューは 10 項目と受け入れ条件で行う。
- D6(issue の文より狭い): issue は「reviewer の雛形から `run-test.sh` を外す」と書くが、この PR は reviewer の既定を両方のまま残した。初稿は既定を静的検査だけにしていた。Codex の計画助言(HIGH)が、leader が書き忘れるとテストを走らせないまま `GATE: pass` になること、雛形は spawn のたびに展開されるので古い leader と新しい reviewer が組み合わさることを指摘した。今の形は、書き忘れても版が混ざっても、テストを打つ側に倒れる。古い leader と新しい reviewer は、指示がないので両方を打つ。新しい leader と古い reviewer は、古い雛形も「指示があればそれだけ」なので、指示された静的検査だけを打つ。issue の受け入れ条件 3 は、機能ごとの org のレビューの TASK について満たす。
- release のノートに書くこと: この branch の `perf:` 11 件と `fix:` 4 件が changelog に出る(`.goreleaser.yml` は `^docs:`、`^test:`、`^chore:` を除く)。ノートの案は sync-docs report の「`/pr` と release に渡すこと」にある。
  - `fix:` の 4 件(3c64d65d、662d3e47、246c7432、d721ed7d)は、この PR が入れた狭めた検査の直しで、main の不具合の修正ではない。ノートでは新しい形の一部として書く。
  - org の役割雛形は go:embed で binary に入り、spawn のたびに展開される。新しい binary で spawn した座席から効き、古い binary で立てた座席は古い文のまま動く。
  - plan の雛形 `docs/plans/templates/feature-plan.md` は seed のファイルで、`ralph upgrade` は上書きしない。既存のプロジェクトでは、ユーザーが自分で雛形を更新する。
- 戻し方: この PR を revert する。古い雛形に戻るだけで、台帳の形は変わらない。

## 2. implementer の雛形(S1、8ee986bd)

- `internal/org/prompts/implementer.md:32-57`: スライスごとに、leader の TASK の本文の `VERIFY:` の行のコマンドを打つ。行がなければ、触ったファイルを覆ういちばん狭い検査を選ぶ(`:36-50`、D1)。`RALPH_VERIFY_SCOPE=changed` では絞れず(`:51-52`)、`./scripts/run-verify.sh` はそれより狭い検査で覆えないときだけ使う(`:52-54`)。選んだコマンドは RESULT に書く。
- `:70-89`: コミットの前に、`git diff --cached` で 5 項目を見る。不要な変更、誤字とコピペの誤り、デバッグの残り、秘密情報、scope の外のファイルの順。範囲内で直せない項目は found and not fixed として理由とともに書く(D2)。
- RESULT の例(`:117-131`)に `VERIFY:` と `SELF-CHECK:` の行があり、レポート契約(`:163-181`)が同じ内容を求める。
- テスト: `TestRenderRolePrompt_Implementer_MissionRunsTheSliceCommand`(`prompts_verification_test.go:52`)、`..._MechanicalSelfCheckBeforeEachCommit`(`:99`)、`..._ResultCarriesVerifyAndSelfCheck`(`:155`)。
- 見ること: 計画の S1 は `internal/org/prompts_test.go` に足すと書くが、F2(f0870d6e)で `prompts_verification_test.go` に移した。`prompts_test.go` は main と同じで、計画の Progress がこの逸れを記録している。
- 見ること: `VERIFY:` の行や RESULT の例には `$(gofmt -l …)` が入る。座席が本文を二重引用符の引数に直接書くと shell が展開してしまうので、ファイルに書いて `"$(cat <ファイル>)"` で渡す 1 文を足した(`:110-115`、S3 の 3989e87f で入れた)。

## 3. reviewer の雛形(S2、3478cb68)

- `internal/org/prompts/reviewer.md:17-29`: レビューの TASK では、leader の指示があればそのコマンドだけを打ち(足しも削りもしない)、なければ `run-static-verify.sh` と `run-test.sh` の両方を打つ(D6)。機能ごとの org のフルの test は、PR の前の gate の TASK で 1 回走ると書く。
- `:36-43`: 差分のレビューは 10 項目(命名、読みやすさ、不要な変更、誤字とコピペの誤り、null の扱い、デバッグの残り、秘密情報、例外処理、security、保守性)と受け入れ条件(D3)。
- `:53-62`: RESULT と BLOCKED の `GATE:` は、打ったコマンドの結果を指す。打ったコマンドは本文の `CHECKS:` の行に書く(D5)。機能ごとの org の `GATE: pass` は、静的検査が通ったことだけを意味する。
- `:148-149` と例(`:151-177`): `CHECKS:` はヘッダではなく本文の行。座席内 fan-out の例(`:205-209`)は、子にも同じコマンドだけを打たせる。
- gate の TASK の節(`:64-129`)は変えていない。`git diff 622cdcfb...HEAD -- internal/org/prompts/reviewer.md` の hunk は、この範囲を通らない。
- テスト: `TestRenderRolePrompt_Reviewer_ReviewTaskRunsExactlyTheNamedCommands`(`prompts_verification_test.go:243`)、`..._GateMeansTheCommandsOnTheChecksLine`(`:292`)、`..._DiffReviewCoversTenItemsAndTheCriteria`(`:339`)、`..._ReviewExamplesCarryTheChecksLine`(`:374`)。
- 見ること: RESULT の例のヘッダ(`GATE:`・`HEAD:`・`BASE:`)と `EVIDENCE:` の行の形は変えていない。`/org` skill の RESULT の例と突き合わせる `TestOrgSkill_ResultExampleFollowsTheReviewerTemplate`(`internal/org/report_names_sync_test.go:269`)があるため、`CHECKS:` は本文側に置いた。

## 4. leader の雛形(S3、3989e87f)

- `internal/org/prompts/leader.md:28-73` のミッションの 3: どの org でも、implementer への TASK に、そのスライスの検証コマンドを本文の `VERIFY:` の行で書く(ヘッダにはしない)。Go の形、複数スライスのときの行、止まったスライスの `RESUME:` の行もここにある。機能ごとでない org は、レビューの TASK にコマンドを書かず、reviewer の既定(両方)に任せる(`:70-73`)。
- 機能ごとの org の手順 1(`:149-157`): 計画の `## Implementation outline` のスライスごとに「— verify:」を書く。`RALPH_VERIFY_SCOPE=changed` は、走らせる言語パックを選ぶだけで Go では `go test ./...` を全部走らせるので、絞り込みに使わない。手順 5(`:229-232`)は、そのコマンドを書き換えずに `VERIFY:` の行へ写す。
- 手順 8(`:256-275`): レビューの TASK に `CHECKS: ./scripts/run-static-verify.sh` の 1 行を書く。返った RESULT に `CHECKS:` の行がないか、その行に `run-static-verify.sh` がなければ、新しい TASK_ID で送り直す。2 回続けて通らなければ 3 回目は送らず、escalate する。
- typed protocol の節(`:470-505`): TASK の本文はファイルに書いて `"$(cat <ファイル>)"` で渡す。`VERIFY:` は ralph が読まず implementer が読むので、ヘッダでなく本文の行にする(D4)。implementer への TASK の例に `VERIFY:` の行がある。
- gate の TASK の手順 12(`:322` から)は変えていない。既存の `TestRenderRolePrompt_Leader_FeatureOrgAsksTheReviewerForTheGate` が通り続ける。
- テスト: `TestRenderRolePrompt_Leader_MissionPutsTheSliceCommandOnAVerifyLine`(`prompts_verification_test.go:442`)、`..._FeatureOrgPlanCarriesTheSliceCommands`(`:622`)、`..._FeatureOrgReviewTaskNamesOnlyTheStaticScript`(`:670`)、`..._OnlyAFeatureOrgNamesTheStaticScriptAlone`(`:760`)ほか。機能ごとでない org にその指示がないことも照合する。

## 5. /org skill(S4、a8253fa9)

- `.claude/skills/org/SKILL.md` の 5 か所を S1〜S3 に合わせた。
  - 「org の終わり方」の冒頭(`:950-965`)に、検証の分け方の段落。
  - 「座席内 fan-out」の reviewer の例(`:1312-1316`)。
  - 「役割」の implementer(`:1324-1350`)と reviewer(`:1351` から)。
  - 「Leader 運用 2 経路」の手順 3(`:1455-1470`)。
  - typed protocol の RESULT の例に `CHECKS:` の行(`:1510`)。
- 4 つの写しは `check-skill-sync.sh` と `check-sync.sh` で一致を確かめる。
- 見ること: 「Leader 運用 2 経路」の手順 3 は、計画の handoff の外で足した。現セッションから昇格した leader は役割雛形を読まず、この skill だけを読むためである。

## 6. 文書(S5、3801a512)

- `docs/quality/quality-gates.md:82` の Quality pipeline gate の行と、`templates/base/docs/quality/quality-gates.md:81`。reviewer の記述を、指示されたコマンドを打つ形に直し、機能ごとの org のフルの test は gate の 1 回と足した。
- `docs/recipes/codex-seat-permissions.md` の reviewer の段落(`:221` 付近)と `templates/base/` の写し。
- `docs/specs/2026-08-01-org-runtime.md:33-39` に「2026-10-27 改訂(perf/org-narrow-verification)」の (a)〜(d)。本文の履歴は書き換えず、FR-7 の行(`:79`)に印を付けた。(d) が D6 の理由を残している。director の仕様 `docs/specs/2026-10-07-org-multi-org-director.md:127-128` の FR-7 にも、2026-10-27 に決めたことと印を足した(sync-docs)。
- `docs/tech-debt/README.md`: 「A feature org's PR runs the tests twice」の行(`:225`)を解決済みにし、reviewer の文が古くなった行(`:157` の「quality gate is prompt-instructed」の行ほか)の文を直した。
- 見ること: 仕様は `ralph org gate` と `ralph org regate` を変えていないと明記している。この PR の範囲に gate と regate の振る舞いは入らない(計画の Non-goals)。

## 7. レビューで足した直し

番号は計画の Progress のとおり。F7 と F11 は計画の記録にない。

### 7.1 CHECKS: の行と送り直し(F1 0ea899f5、F2 f0870d6e、F3 8cda482e)

- 1 回目の self-review(06f60e10、LOW 8、Merge yes)への直し。reviewer が受け取る指示を TASK の本文の `CHECKS:` の行に決めた(F1)。返った `CHECKS:` の行が欠けたら leader が送り直す。F2 でその回数に上限を付けた(2 回続けて通らなければ escalate、`leader.md:266-273`)。
- F3: reviewer が指示とみなすのは `CHECKS:` の行だけにした(`reviewer.md:21-24`)。行がなければ、本文のほかの所にコマンドの名前があっても両方を打つ。散文の指示を受け入れる文は、打つ検査が増える側にしか動かないため外した。
- F2 で、このブランチのテストを `prompts_test.go` から `prompts_verification_test.go` に移した。

### 7.2 複数スライスの VERIFY: と RESUME:(F1、F3、F8 28d83b96)

- 1 つの TASK が複数のスライスを頼むときは、`VERIFY: S1: <コマンド>` の行をスライスごとに書く(`implementer.md:32-36`、`leader.md:46-48`)。RESULT の `COMMIT:`・`VERIFY:`・`SELF-CHECK:` も、スライスの名前で始める(F3、`implementer.md:173-179`)。`RESUME:` の行もスライスの名前で始め、その名前のスライスのコミットに入れる(F8、`implementer.md:22-26`、`leader.md:55-57`)。
- テスト: `TestRenderRolePrompt_SeveralSlicesInOneTaskGetAVerifyLineEach`(`prompts_verification_test.go:585`)、`TestRenderRolePrompt_Implementer_ResultOfSeveralSlicesNamesEachSlice`(`prompts_verification_multi_test.go:20`)、`TestOrgSkill_RoleBulletsNameEachSliceAndReadOnlyTheChecksLine`(`:72`)。

### 7.3 秘密情報の片付けの境界と、前からあった変更(F4 ffafc9d2、F5 a7d879d9)

- 1 回目の cross-review(triage 8438af88、ACTION_REQUIRED 1)の指摘: implementer の秘密情報の片付けが、座席が始める前から stage されていた scope の外の変更まで消しうる。
- F4: 直すのはこのスライスの変更だけにした。前からあった変更は、書き換えも削除も unstage もしない。前からあった変更の秘密情報と、前から stage されていた scope の外のパスは、コミットせずに BLOCKED を返す。
- F5: 「前からあった変更」を、TASK ごとの最初の編集の前の `git status --porcelain` の記録で 1 度だけ定義した(`implementer.md:15-18`)。コミットしない 3 つの場合を 1 文に並べ(`:80-89`)、unstage は `git restore --staged -- <このスライスのパス>` にして、パスなしの `git reset` を禁じた(`:77-79`)。
- テスト: `prompts_verification_predate_test.go` の `..._RecordsPredatingChangesAtTheStartOfEachTask`(`:22`)、`..._SelfCheckLeavesChangesThatPredateTheSlice`(`:73`)、`TestOrgSkill_ImplementerBulletLeavesChangesThatPredateTheSlice`(`:124`)。
- 標準フローの implementer と `/implement` 手順 7 にも同じ形の穴がある。#261 で merge 済みの文でこの計画の範囲の外なので、tech-debt に記録した(`docs/tech-debt/README.md:382`)。

### 7.4 止まったスライスの再開(F6 854be9db、51cf59e2、F8)

- F5 のあと、コミットせずに止まった implementer の変更は、次の TASK では「前からあった変更」になり、頼み直しても座席がまた止まる。F6: leader は、頼み直す TASK の本文の `RESUME:` の行に、止まった TASK がコミットせずに残したパスを並べる(`leader.md:49-68`)。implementer はそのパスの変更を引き継ぎ(`implementer.md:19-28`)、scope の外のパスがあれば BLOCKED を返す。
- 秘密情報で止まったあと、leader は送る前に `git diff HEAD -- <パス>` で値が消えたことを確かめる。消えていないとき、または前からあった変更にあるときは、再開せず escalate する。
- 51cf59e2(inline): F6 の implementer が挙げた穴。スライスを狭めるときも、残したパスは外すものも含めてすべて `RESUME:` に並べ、外すパスの変更は元に戻すよう TASK に書く。leader はコードに触れないため。
- F8: 複数スライスの `RESUME:` にスライスの名前を付け、skill の手順 3 に escalate と狭める文を足し、「前からあった変更」の説明を implementer の定義にそろえた。
- テスト: `TestRenderRolePrompt_Leader_ResendsAStoppedSliceWithAResumeLine`(`prompts_verification_predate_test.go:192`)、`..._Implementer_TakesOverThePathsOfAResumeLine`(`:242`)、`TestOrgSkill_ImplementerBulletAndLeaderCycleCarryTheResumeLine`(`:294`)。

### 7.5 Go のスライスの検査の 30 分 timeout と入れ子の module の root(F9 3c64d65d、F10 662d3e47、246c7432、F12 d721ed7d)

- 2 回目の cross-review(triage 45f8b554、ACTION_REQUIRED 2)の指摘。どちらも、狭めた検査が旧 `run-verify.sh` にあった言語パックの性質を落としていた。
  - `go test` に timeout がなく、Go の既定の 10 分で打ち切られる。言語パックの `run_tests` は `GOFLAGS` の先頭に `-timeout=30m` を置く。`internal/org` は負荷の下で 943 秒かかった。
  - 入れ子の `go.mod` の module を repo の root から打つと落ちる。言語パックは module ごとに cd する。
- F9: implementer と leader の雛形のコマンドを `test -z "$(gofmt -l <dir>)" && go vet ./<dir>/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./<dir>/` にした(`implementer.md:39`、`leader.md:34`)。timeout の理由(`implementer.md:41-45`)と、入れ子の module でコマンド全体を `(cd <module> && …)` で包む規則(`:45-50`)を足し、skill の 4 つの写し(`.claude/skills/org/SKILL.md:1330-1333`)も同じにした。
- F10: leader の手順 1 は「雛形に欄があればそれに沿う」ので、plan の雛形が裸のコマンドのままだと、F9 の直しが計画の「— verify:」の行で戻る。plan の雛形の 6 つの写し(`docs/plans/templates/feature-plan.md:39` ほか)を同じ形にし、`tests/test-slice-verification.sh:119-125` に写しごとの pin を 2 本ずつ足した。
- 246c7432(inline): 雛形の「呼び手が」を「環境の `GOFLAGS` か `go env -w` で timeout が決めてあれば」に直した。
- F12: 回し直しの verify の LOW 1。「Leader 運用 2 経路」の手順 3 に Go の形がなかった。昇格した leader は skill だけを読むので、手順 3 の括弧に、「役割」の節の implementer の項の形(`GOFLAGS` の timeout と `(cd <module> && …)` の包み)をそのまま使うと足した(`.claude/skills/org/SKILL.md:1458-1460`)。
- テスト: `TestRenderRolePrompt_GoSliceCheckKeepsThePackTimeoutAndEntersANestedModule`(`prompts_verification_test.go:489`)は、言語パックの `run_tests` が今も `-timeout=30m` で始まることも照合する。`TestOrgSkill_ImplementerBulletNamesTheGoSliceCheck`(`:535`)は skill の implementer の箇条と手順 3 を照合する。
- 見ること: F12 は verify の回し直しのあとに入ったので、最後の cross-review の前に HEAD 1b678012 で `./scripts/run-static-verify.sh` を打ち直した(rc 0)。

## 8. 後工程とテスト

- 1 回目: self-review 2 run(06f60e10 LOW 8、f2d075c8 LOW 2)、verify 584f8103 pass、test 3afdd03a pass(`run-test.sh` が rc 0)、sync-docs 02afccd2、cross-review が ACTION_REQUIRED 1(7.3)。
- cycle 2: F4〜F8 で直して `/self-review` から回し直した。self-review 2 run(dd2ffa05、bd21c5fd、どちらも LOW 5)、verify 7458318d pass、test 1637025d pass、sync-docs 6a5bed1e、cross-review が ACTION_REQUIRED 2(7.5)。cap の 2 に届いた。
- cap 3 の回し直し: F9・F10 のあと `/self-review` から全体を回した。self-review ea7a6a5b(LOW 3、Merge yes)、verify bbb02949 pass、test 69b112f6 pass、sync-docs b43464d2、cross-review(triage 1b678012)は指摘 0 件。
- 最後の test の判定は pass だが、`run-test.sh` 自体は rc 1 を返している。落ちたのは、前からある不安定な `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime` 1 本だけである(`docs/tech-debt/README.md:285` の行。base でも単独の 20 回中 11 回落ちる)。`internal/org` を `-count=1` で打ち直すと 620.7 秒で通った。`go test` の既定の 10 分は越えたが、`-timeout=30m` の下だった。
- 通ったテスト: 雛形の組のトップレベル 74 本、`tests/test-slice-verification.sh` の 162 本。使い捨ての module で、新しい検査の形の振る舞いも確かめた(結果は「確かめていないこと」の 2 つ目)。
- 変異: 雛形の文を変えるとテストが落ちることを、スライスごとに確かめた(S1 は 6 個、S2 は 14 個、S3 は 13 個で、それぞれ狙ったテストが落ちた)。最後の test の変異 m1〜m4 も、それぞれ 1 本以上のテストが落とし、対照は通った。

## 9. 確かめていないこと

- 実機の org での振る舞い。leader が `VERIFY:`・`CHECKS:`・`RESUME:` の行を書くこと、implementer が入れ子の module のパッケージで `(cd <module> && …)` を選ぶこと、reviewer が `CHECKS:` の行のコマンドだけを打つこと、新旧の binary の座席が混ざったときの動き。雛形の文があるかをテストで見ただけで、モデルが従うかは見ていない。director の仕様の FR-7 は `[ ]` のままで、次の実行の記録で付け直す。
- 新しい Go の検査の形は、使い捨ての module で `go vet` と `go test` を直接打って確かめた。timeout は既定で 30 分、環境の `GOFLAGS=-timeout=45m` と `go env -w` の 50m があとに来て勝つ。入れ子の module は root からだと落ち、`(cd service && …)` で通る。implementer の座席が打った結果ではない。
- 古い binary で立てた座席は、裸の `go test` を打つ。10 分を越えるパッケージのスライスの検証は、そこで打ち切られうる。
- `run-test.sh` の 1 回の実行は、前からある不安定な 1 本で rc 1 になることがある。このテストは直していない(計画の範囲の外)。
- tech-debt に繰り越した LOW。`docs/tech-debt/README.md:381` の行に、テストの整理(同じ文を 2 か所で固定している、`predate` のファイルに再開のテストも入っている、など)を (a)〜(f) で記録した。`prompts_verification_test.go` は 908 行で、800 行の目安を越えている。plan の雛形の 6 つの写しは、timeout の理由の文を shell のテストが固定しない。
- hook の「Run ./scripts/run-verify.sh before claiming done」の文は、#261 の tech-debt の行のまま残っている(計画の Non-goals)。
- `ralph org gate` と `ralph org regate` の振る舞い。変えていない。regate の test を CI に任せる件は #264 で扱う。
