# Test report: org-xreview-own-model

## Run 2 (cycle 2)

- Date: 2026-11-22(実行は 08:38:30〜09:01:00 JST)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)
- Tester: tester サブエージェント(`/test`)。パイプライン 2 回目(`cycle-count.json` は `cycle: 2`)。今の判定は、この節のあとの `## Verdict`。Run 1 は見出しを 1 段下げて、`---` の下にそのまま残す
- Scope: HEAD `5c5ead66`(worktree は clean)。Run 1(HEAD `7c7cdec5`)のあとの変更は、main の取り込み 6b671276(origin/main `ccdf6293`。Run 1 の base `bd079dbb` から増えたのは PR #370(#363)だけで、Go の本番のコードは `internal/org` の 7 ファイルと `internal/cli/org.go`)、sync-docs の 0d44799f(テストのコメント)、cross-review の直し eff83438(reviewer の雛形の手順 2 で `diff: failed` のときも 5 の git の状態の比べを通す)、self-review の LOW の直し 7eb0179d(fan-out の節の決まり、手順 5 と 2 の文、テストの失敗の文、新しいテスト `TestRenderRolePrompt_Reviewer_FanOutKeepsTheChildrensFilesOutOfTheWorktree`)と、各段の report。この branch の Go の本番のコードは Run 1 のあと変わっていない。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。打つ前に `pgrep -fl 'scripts/run-test.sh'` と `pgrep -fl run-verify.sh` が何も返さないことを確かめた。起動は Run 1 と同じで、perl の fork と setsid で切り離し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`。`nohup` は使っていない)、`caffeinate -w` で 168 s ずつ待った。範囲は wrapper の既定(`Requested scope: changed`)で、Run 1 と同じく `Language scope: full fallback (unclassified:scripts/xreview-helpers.sh)` になり、走った pack は golang だけ。そのあと、`(cached)` の 5 パッケージを `-count=1` で打ち直し、eff83438 の道を走らせる使い捨ての probe と変異 16 個を足した
- Evidence: `docs/evidence/test-2026-11-22-org-xreview-own-model.log` の「# Run 2 (cycle 2)」の行から後(Run 1 の 6,843 行のあとに追記した。gitignore の対象で commit されない)。run-test.sh の出力、1 分ごとの load、`(cached)` の打ち直し、probe のテストの全文と結果、変異の harness と各変異の結果と失敗の文、`~/.codex/config.toml` と herdr のプロセスの記録が入っている。wrapper 自身の log は `docs/evidence/verify-2026-11-21-233830.log`(同じく追跡外)

### Test execution (cycle 2)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つき。Run 1 と同じ 2 つ) | 08:38:30〜09:01:00 JST、1,350 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合 | 約 10 分(08:38:30〜08:49 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` の 5 パッケージは `-count=1` で打ち直して ok | 約 12 分(08:49 ごろ〜09:01:00) |

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 0.823 s |
| `internal/cli` | ok | 722.390 s |
| `internal/config` | ok | 13.675 s |
| `internal/gitroot` | ok | run-test.sh では `(cached)`。`-count=1` で 2.131 s |
| `internal/insights` | ok | 1.231 s |
| `internal/org` | ok | 724.259 s |
| `internal/org/driver` | ok | 2.158 s |
| `internal/org/protocol` | ok | run-test.sh では `(cached)`。`-count=1` で 0.396 s |
| `internal/scaffold` | ok | run-test.sh では `(cached)`。`-count=1` で 1.067 s |
| `internal/toolchain` | ok | run-test.sh では `(cached)`。`-count=1` で 15.907 s |
| `internal/upgrade` | ok | run-test.sh では `(cached)`。`-count=1` で 1.962 s |

`(cached)` の 5 パッケージは、run-test.sh が終わったあと(09:01:07〜09:01:23)に `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/gitroot/ ./internal/org/protocol/ ./internal/scaffold/ ./internal/toolchain/ ./internal/upgrade/` で打ち直し、rc 0 だった。5 つとも `go list -test -deps` に `internal/org` も `internal/cli` も入らず、Run 1 のあとに Go のファイルが変わったのは `internal/cli`、`internal/org`、`internal/org/driver` だけなので、cache に当たるのは筋が通る。Go の段の始まりは、終わりの 09:01:00 から `internal/org` の 724 s を引いて 08:49 ごろ。`internal/cli` と `internal/org` は 1 パッケージの上限 30 分のおよそ 40% で終わった(Run 1 は 55%)。

1 分 load は run の全体で 4.94〜19.49(08:38 から 1 分ごとに 23 回測った)。最も高かったのは shell の段の 08:44(19.49)で、自分の変異の run(08:43〜08:46、1 個 12〜14 s)と重なった。Go の段では 17.95(08:51)から 5.01(09:00)へ下がった。Go の段の始めの 08:48〜08:52 には、下の広い絞り込みの変異の run(1 個 76〜101 s)が重なった。

shell の段の `PASS` の行を、Run 1 の log の行と同じ数え方(行頭の空白のあとの `PASS`)で数えると、どちらも 5,390 行だった。数字と一時ディレクトリの名前を伏せて並べ替えると、2 つの集合は 1 行も違わない。main の取り込みは `tests/` と `scripts/` を変えていない。

`~/.codex/config.toml` は、run のあいだ(08:52 と 08:56)と、変異の run がすべて終わったあと(09:19)で、mtime 2026-11-20 20:41:50、大きさ 38,144 バイト、sha256 `0f076aef…` のままだった。書かれていない。読まれたかどうかは測っていない(この machine では読みで atime が変わらない)。本物の herdr server(pid 59855)も、08:52 と 09:19 に同じ pid で動いていて、止めても触ってもいない。run と変異のあと、worktree の変更はこの報告と計画の 2 ファイルだけで、HEAD は `5c5ead66` のまま。

#### eff83438 の道(`diff: failed` のあと 5 の比べ)をテストが覆うか

文は固定されている。`TestRenderRolePrompt_Reviewer_XreviewWritesTheGitStateAndTheDiff` が手順 2 の `diff: failed` の項の文を順に読み、直しを外したり曲げたりした変異 F01〜F07 はどれもこのテストで落ちた(下の表)。

雛形のコマンドを走らせるテストは、この道を覆っていない。`TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes` の `a base git diff cannot read` と `a base that reads as an option` は、手順 2 のコマンドだけを bash と zsh で走らせ、`diff: failed` が出ること(前者は git の stderr の `unknown revision` も、後者はファイルを書かないことも)を見て終わる。そのあとに手順 5 のコマンドを走らせる行はない。手順 5 を走らせる行(`…XreviewCommandsWriteTheStateAndTheDiff` と、git の状態が変わる 4 種類)は、どれも手順 2 が `diff: written` を出したあとに走る。

HEAD では道は動く。使い捨ての probe(`TestProbe_XreviewDiffFailedThenStep5`、HEAD の `git archive` の写しにだけ置き、commit していない。全文は evidence の log)で、既存の fixture と helper を使い、手順 2 のコマンドを `XREVIEW_BASE:` が `origin/gone` と `--output=<file>` の 2 通りで走らせて `diff: failed` を見たあと、手順 5 のコマンドを走らせた。bash と zsh、あいだに何もしない場合とコミットを 1 つ足す場合の 8 通りで、前者は `git: same` で作業ツリーは clean、後者は `git: changed` になり、codex と claude の stand-in は 1 度も呼ばれなかった。8 通りとも PASS。

穴の大きさは、下の変異 B01・B02・B04 で測った。どれも雛形のコマンドとテストのその行の固定を一緒に替えたもので、既存のテストでは落ちず、probe では落ちた。雛形だけを替えれば、コマンドの行の一致で落ちる(B01t)。Run 1 の Test gaps の 1 と同じ形の穴で、Test gaps (cycle 2) の 1 に LOW で記録した。

#### 変異 (cycle 2)

HEAD の `git archive` を scratch に展開した写しで回し、worktree には触れていない。harness は Run 1 と同じ作りで、置き換える文字列の数が決めた数と合うことを確かめてから替え(日本語の折り返しのある文は、文字のあいだの空白を許す一致で数えた)、各変異のあとにファイルを元のバイト列に戻した。最後に、写しを新しく展開した写しと `diff -rq` で比べ、probe のファイルのほかに違いがないことを確かめた。各変異は `go test -count=1 -run 'RenderRolePrompt_Reviewer|RolePrompts_|Skill' ./internal/org/` で回し(1 個 11〜14 s)、B の変異はそのあと probe だけを回した。変異の前の対照は rc 0(上の絞り込みで最上位のテスト 117 個、probe は 8 通り)。どの変異も `build failed` は 0 件。

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| F01 | 手順 2 の `diff: failed` の項を eff83438 の前の文(「出力に出た `git diff` の stderr の要点 … 6 の形の report を … 書き、… RESULT を返す」)に戻した | 落ちた | `XreviewWritesTheGitStateAndTheDiff` |
| F02 | 「3 と 4 を飛ばして 5 に進み、git の状態を比べる(…)。」を消した | 落ちた | 同じ |
| F03 | 「`git: changed` なら 5 のとおり BLOCKED を返す。」を消した | 落ちた | 同じ |
| F04 | その項の `git: changed` と `git: same` を入れ替えた | 落ちた | 同じ |
| F05 | 「5 に進み」を「6 に進み」にした | 落ちた | 同じ |
| F06 | 括弧の「(3 で読み終えられなかったときと同じく、report を書く前に比べる)」だけを消した | 落ちた | 同じ |
| F07 | 7eb0179d の「2 のコマンドの出力に出た」を前の「出力に出た」に戻した | 落ちた | 同じ |
| F08 | 手順 5 の `git: changed` の括弧を、7eb0179d の前の「読んでいるあいだに HEAD か作業ツリーが動いたので、所見は今の branch を見ていない」に戻した | 落ちた | `XreviewComparesTheGitStateBeforeTheReport` |
| F09 | `## 座席内 fan-out` の新しい決まり(子の作業ファイルを worktree の外に置かせる)を項ごと消した | 落ちた | `FanOutKeepsTheChildrensFilesOutOfTheWorktree` |
| F10 | その決まりから、`.harness/state/` にも置かせない文とその理由だけを消した | 落ちた | 同じ |
| F11 | 手順 3 の fan-out の項の終わりに「子の作業ファイルは `$TMPDIR` に置かせる」を足しただけ(固定した文は残る) | 落ちた | 同じ(手順 3 が置き場所を繰り返さない、の検査) |
| B01 | 手順 2 のコマンドで、`diff: failed` のときに `-git-before.txt` を消す(`else rm -f "$p-git-before.txt"; echo 'diff: failed'; fi`)。テストのその行の固定も同じに替えた | 残った。probe では落ちた | 既存のテストではなし。probe の「何もしない」4 通りが `git: changed` を見て落ちた |
| B01t | B01 を雛形だけに入れ、テストの固定は元のまま | 落ちた | `XreviewWritesTheGitStateAndTheDiff`(コマンドの行の一致) |
| B02 | 手順 5 のコマンドを、差分のファイルが空でないときだけ `git: same` を出す形にした(`if [ -s "$p.diff" ] && cmp -s …`)。固定も替えた | 残った。probe では落ちた | 既存のテストではなし。probe の「何もしない」4 通り |
| B04 | 手順 2 のコマンドで、`diff: failed` のときに worktree の root に `xreview-failed.txt` を書く(`echo 'diff: failed' \| tee xreview-failed.txt`)。固定も替えた | 残った。probe では落ちた | 既存のテストではなし。probe の「何もしない」4 通り(`git: changed` と `?? xreview-failed.txt`) |
| B12 | B01 と B02 を一緒に入れた | 残った(`internal/org` 全体でも)。probe では落ちた | 既存のテストではなし。probe の「何もしない」4 通り |

B01・B02・B04 は、上の絞り込みのあと、より広い `-run 'Prompt|Skill|Xreview|Retired'`(probe は写しから外した。1 個 76〜101 s)でも残った。手順 2 と 5 のコマンドを走らせる helper(`reviewerXreviewCommand`)と `diff: failed`・`-git-before.txt` の文字列を使うテストは、grep すると `prompts_xreview_reviewer_test.go`、`prompts_xreview_leader_test.go`、`prompts_leader_plan_test.go` の 3 ファイルにしかなく、どれもこの絞り込みに入る(3 ファイルのテスト関数の名前はすべて、この `-run` の式に当たる)。そのうえで、B12 と B04 を `internal/org` のパッケージ全体(`go test -count=1 -timeout 30m ./internal/org/`、probe は外した)でも回した。run-test.sh のあとの 09:02〜09:19 に、別々の写しで 2 つを並べて回し、パッケージの時間は 992 s と 998 s で、どちらも rc 0 のまま残った。B01 と B02 は、単独では広い絞り込みで残り、合わせた B12 は全体で残った

### Coverage (cycle 2)

- Statement: 測り直していない。この branch の Go の本番のコードは Run 1 のあと変わっていない(eff83438 と 7eb0179d が変えたのは雛形とテストだけ)ので、Run 1 の数字(下の Run 1 の Coverage)がそのまま当てはまる。main から来た Go の変更は #363 の pipeline が測っている(`docs/reports/test-2026-11-22-split-shared-paths.md`)。この run では、取り込んだあとの木で `internal/org` と `internal/cli` が通った
- Changed lines: cycle 2 で変わったのは `go:embed` の雛形とテストで、Go の coverage では測れない。上の変異 F01〜F11 で、変わった文がテストに固定されていることを確かめた
- Branch: Go の標準ツールでは測れない

### Failure analysis (cycle 2)

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

run-test.sh は 1 回目で rc 0 だった。打ち直しはしていない(`(cached)` の 5 パッケージの `-count=1` は打ち直しでなく、走っていなかった分を走らせたもの)。既知の flake(`TestGate_TwoGatesRunTheirScriptsOneAtATime`、`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval` など)は、この run では 1 つも出ていない。

### Regression checks (cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| main の取り込み(6b671276)のあとの、この branch のテストと #363 のテスト | 通る | run-test.sh の `internal/org`(724.259 s)と `internal/cli`(722.390 s)が ok |
| cross-review の所見(`diff: failed` のあいだのコミットが incomplete の RESULT で流れる)を直した eff83438 の文 | テストが固定する | 変異 F01〜F07 |
| eff83438 の道そのもの(`diff: failed` のあと手順 5 が `git: same`、あいだにコミットがあれば `git: changed`) | HEAD で動く | probe が bash と zsh の 8 通りで PASS。既存のテストは走らせていない(Test gaps (cycle 2) の 1) |
| 7eb0179d の fan-out の決まりと手順 5 の文 | テストが固定する | 変異 F08〜F11 |
| Run 1 で確かめた依頼のテスト(bash と zsh でのコマンドの run、git の状態が変わる 4 種類、前の版の TASK の行、R1 の待ちの決まり) | 通る | run-test.sh の `internal/org` の中で通った |
| shell のテストの集合 | Run 1 と同じ | `PASS` の行の集合が、数字と一時ディレクトリの名前を除いて Run 1 と同じ。`tests/test-xreview-helpers.sh` は `PASS: 53`、`FAIL: 0`(AC6) |

### Test gaps (cycle 2)

1. eff83438 の道を走らせるテストがない(LOW)。手順 2 の `diff: failed` のあとに手順 5 のコマンドを走らせる行がなく、手順 2 と 5 のコマンドをテストのその行の固定と一緒に変える変更(失敗のときに `-git-before.txt` を消す B01、差分が空なら `git: changed` にする B02、失敗のときに worktree に書く B04)は、既存のテストでは止まらない。雛形だけの変更はコマンドの行の一致で落ちる(B01t)ので、今の雛形は守られている。`XreviewCommandOutcomes` の 2 つの `diff: failed` の行に続けて手順 5 のコマンドを走らせ、`git: same` と clean な作業ツリーを見る行と、あいだにコミットを足して `git: changed` を見る行を足せば、3 つとも落ちる(probe で確かめた。1 通り 2 s ほど)
2. Run 1 の Test gaps の 1〜5 は残る。eff83438 と 7eb0179d はそれらに触れていない

## Verdict

- Pass: 1(`./scripts/run-test.sh` が rc 0、1,350 s。local verifier 46 本と `go test ./...` の 11 パッケージがすべて通った。`(cached)` の 5 パッケージは `-count=1` でも通った。eff83438 と 7eb0179d の文を外したり曲げたりした変異 F01〜F11 と B01t は、既存のテストで落ちた)
- Fail: 0
- Blocked: 0
- Verdict: pass。eff83438 の道(`diff: failed` のあと手順 5 の比べ)を雛形のコマンドで走らせるテストはなく、コマンドと固定を一緒に変える変異 3 つが残る。雛形だけの変更は落ちるので、Test gaps (cycle 2) の 1 に LOW で記録した。/sync-docs に進んでよい
- Insight event: `./scripts/insights-append.sh --slug org-xreview-own-model --flow standard --phase test --verdict pass --cycle auto --source skill` で cycle 2 の 1 行を足し、このレポートと同じコミットに入れた。足したときには UTC の日付が 11-22 に変わっていたので、行は前の段の行がある `2026-11-21-org-xreview-own-model.jsonl` でなく、新しくできた `docs/insights/events/2026-11-22-org-xreview-own-model.jsonl` に入った

---

## Run 1 (cycle 1、履歴)

Run 1(HEAD `7c7cdec5`)の記録。見出しを 1 段下げたほかは書いたときのまま。

- Date: 2026-11-22(実行は 06:57〜07:31 JST。パイプラインの cycle 1、`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-22-org-xreview-own-model.md`(issue #360)
- Tester: tester サブエージェント(`/test`)
- Scope: HEAD `7c7cdec5`(origin/main `bd079dbb` を取り込み済み、worktree は clean)。`./scripts/run-test.sh` を 1 回だけ、worktree のルートから回した。起動の前に `pgrep -f 'scripts/run-test.sh'` で、ほかの run-test.sh が走っていないことを確かめた。Bash tool の 10 分の上限を超えるので、perl の fork と setsid で切り離して起動し(INT・HUP・QUIT・TERM は既定の扱いに戻し、標準入出力は `/dev/null`。`nohup` は使っていない)、`caffeinate -w` で終わりを待った。範囲は wrapper の既定の changed(`- Requested scope: changed`)で、`scripts/xreview-helpers.sh` が言語に分類されないため `Language scope: full fallback (unclassified:scripts/xreview-helpers.sh)` になった。この repo で見つかる言語は golang だけなので、走る pack は changed のときと同じ。そのあと、依頼の新しいテストの `-v -count=1` の run、変異 25 個と対照 1 個、狭い coverage の run を足した
- Evidence: `docs/evidence/test-2026-11-22-org-xreview-own-model.log`(6,843 行。`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)。run-test.sh の出力、1 分ごとの load、絞った run の `-v` の出力、変異の harness と各変異の落ちたテストと失敗の文、coverage の要約が入っている。run-verify.sh は自分の記録として `docs/evidence/verify-2026-11-21-215713.log` も書いた(同じく追跡外)

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 下の 2 行の合計 | 全部 | 0 | 2(条件つきの skip、下の行) | 2026-11-22 06:57:12 〜 07:27:48 JST、1,836 s、rc 0。`==> All verifiers passed.` |
| local verifier(`tests/test-*.sh` 46 本) | 46 本、`PASS` で始まる行 5,390 | 46 本(`OK` 46 行) | 0(`FAIL:` の要約行 29 本はどれも `FAIL: 0`) | `test-plan-visual.sh` の browser detection(`/Applications` にブラウザーがある)、`test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」の場合(手元は git 2.49.0) | 約 13 分(06:57:12 〜 07:10 ごろ) |
| `go test ./...`(テストのある 11 パッケージ) | 11 | 11 | 0 | なし。`(cached)` は 0 | 約 17 分(07:10 ごろ 〜 07:27:48)、下の表 |
| `go test -count=1 -v -timeout 30m ./internal/org/ -run '<依頼の新しいテストと周りの 10 の型>'` | 127 関数、`--- PASS` 209 行(subtest を含む) | 全部 | 0 | 0 | 55 s(パッケージ 54.377 s) |

絞った run の `-run` は `Reviewer_Xreview|Reviewer_FanOut|Reviewer_ShowsTheSeatsDriverAndModel|NoDriverOrModelPlaceholderLeft|ReviewerPromptFileNamesTheSeatsDriverAndModel|Leader_Xreview|ReseatRuleSeatCheckItem|Skill_|SayNothingOfTheOtherModels|Leader_FeatureOrgPlanAdvisory`。内訳は `TestDirectorSkill` 44、`TestRenderRolePrompt_Leader` 31、`TestOrgSkill` 31、`TestRenderRolePrompt_Reviewer` 18、ほか 3。

shell の 2 つの skip は、前の周回の codex-seat-network の報告と同じ 2 つで、この変更とは関係がない。`tests/test-xreview-helpers.sh` は `PASS: 53`、`FAIL: 0` だった(AC6、下の Regression checks)。

| パッケージ | 結果 | 時間 |
| --- | --- | --- |
| `cmd/ralph` | ok | 1.604 s |
| `internal/cli` | ok | 996.084 s |
| `internal/config` | ok | 28.179 s |
| `internal/gitroot` | ok | 5.267 s |
| `internal/insights` | ok | 2.276 s |
| `internal/org` | ok | 979.256 s |
| `internal/org/driver` | ok | 4.822 s |
| `internal/org/protocol` | ok | 4.327 s |
| `internal/scaffold` | ok | 3.103 s |
| `internal/toolchain` | ok | 25.345 s |
| `internal/upgrade` | ok | 4.116 s |

golang pack の 1 パッケージの上限は 30 分で、`internal/cli` はその 55%、`internal/org` は 54% で終わった。run のあいだの 1 分 load は 7.73〜33.66(1 分ごとに 31 回測った)。変異の run は 07:05〜07:23 に回し、run-test.sh の shell の段の終わりと Go の段の前半に重なった。落ちたテストはなく、打ち直しもしていない。

run のあと、worktree の `git status --porcelain` は空で、HEAD は `7c7cdec5` のまま。`~/.codex/config.toml` の更新時刻は 2026-11-20 20:41:50 のままで、run のあいだに書かれていない(読まれていないことは macOS では示せない)。本物の herdr server(pid 59855)は run の前後で同じ pid で動いていて、止めても触ってもいない。

#### 依頼の新しいテストと AC の対応

| 依頼の項目 / AC | テスト | 結果 |
| --- | --- | --- |
| reviewer の cross-review の手順のコマンドを bash と zsh で走らせる(AC1、AC2) | `TestRenderRolePrompt_Reviewer_XreviewCommandsWriteTheStateAndTheDiff`(bash と zsh)。手順 2 が `diff: written` を出して差分と HEAD で始まる git の状態を書き、手順 5 が `git: same` を出す。codex と claude の stand-in は 1 度も呼ばれず、作業ツリーは clean のまま | PASS |
| git の状態が変わる 4 種類(AC1) | `TestRenderRolePrompt_Reviewer_XreviewCommandOutcomes` の 4 種類 × 2 shell(追跡中のファイルの書き換え、新しい未追跡のファイル、手順 2 の前からある未追跡のファイルの書き換え、新しいコミット)。どれも手順 5 が `git: changed` を出す。同じテストの `a base git diff cannot read` と `a base that reads as an option` も 2 shell で通る | PASS |
| 前の版の TASK の行を読まない(AC2) | `TestRenderRolePrompt_Reviewer_XreviewTaskCarriesItsLinesInTheBody`(3 行だけを挙げ、`IMPLEMENTER_DRIVER:` などを読まないと言う文)、`TestRenderRolePrompt_Reviewer_XreviewReviewsOnTheSeatsOwnModel`(その文と別のモデルを呼ばない文の外に、前の言葉が残らない)、`…CommandsWriteTheStateAndTheDiff` の前の版の本文でのコマンドの比べ | PASS(比べについては Test gaps の 2) |
| R1 の待ちの決まり(self-review の M1、f95a08f9) | `TestRenderRolePrompt_Leader_XreviewBlockedFollowsTheGatesAndStallIsBounded`(手順 12 と cross-review の小節)、`TestRenderRolePrompt_Leader_ReseatRuleSeatCheckItem`(ミッションの 5)、`TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves`、`TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves` | PASS |
| AC3(report の形) | `…XreviewWritesTheReportUncommitted`、`…XreviewReportIsReadByTheTriageCounter`、`…XreviewAnswersBlockedOnlyForActionRequired`、`…XreviewComparesTheGitStateBeforeTheReport` | PASS |
| AC4(雛形の driver と model) | `TestRenderRolePrompt_Reviewer_ShowsTheSeatsDriverAndModel`、`TestRenderRolePrompt_SeatTemplates_NoDriverOrModelPlaceholderLeft`(3 役 × model あり・なし)、`TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel`(fake の herdr で spawn が書いた雛形のファイル) | PASS |
| AC5(leader の小節) | `TestRenderRolePrompt_Leader_XreviewTaskCarriesTheThreeLinesTheReviewerReads`、`…XreviewReadsNoDriverAndNamesNoOtherModel`、`…XreviewHandlesEveryReply`、上限と `Cross-review:` の行を固める残りの `Leader_Xreview` の 16 関数(状態のコマンドを走らせる 8 関数を含む) | PASS |
| AC7(skill) | `TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel`、`TestDirectorSkill_XreviewSaysTheReviewerSeatReviewsOnItsOwnModel`、`TestOrgAndDirectorSkills_SayNothingOfTheOtherModelsCrossReview`(4 写しずつ)。`check-sync.sh` と `check-skill-sync.sh` は local verifier の中で通った | PASS |

#### 変異

HEAD の `git archive` を scratch に展開した写しで回し、worktree には触れていない。harness は置き換える文字列の数が決めた数と合うことを確かめてから替え(日本語の折り返しのある文は、文字のあいだの空白を許す一致で数えた)、各変異のあとにファイルを元のバイト列に戻した。最後に、写しを新しく展開した写しと `diff -rq` で比べ、違いがないことを確かめた。雛形と Go のコードの変異は `go test -count=1 -run <絞り込み> ./internal/org/` を写しで打ち、skill の変異は写しで一度だけ build したテストの binary で打った。どの変異も `build failed` は 0 件。

| # | 変異 | 結果 | 落ちたテスト |
| --- | --- | --- | --- |
| M01 | 手順 2 と 5 のコマンドから未追跡のファイルの hash(`git ls-files … \| xargs -0 git hash-object --;`)を外し、共有の定数 `leaderAdvisoryGitState` からも外した(実行の行だけが落とせる形) | 落ちた | `XreviewCommandOutcomes` の `an edit of an untracked file`(bash、zsh) |
| M02 | 同じく `git rev-parse HEAD;` を外した | 落ちた | `XreviewCommandOutcomes` の `a new commit`(2 shell)、`XreviewCommandsWriteTheStateAndTheDiff`(HEAD で始まらない) |
| M03 | 同じく `git diff;`(作業ツリーの差分)を外した | 残った | なし(Test gaps の 1) |
| M03t | M03 を雛形だけに入れ、定数は元のまま | 落ちた | `XreviewWritesTheGitStateAndTheDiff`、`XreviewComparesTheGitStateBeforeTheReport`(コマンドの行の一致) |
| M04 | 同じく `git diff --cached;` を外した | 残った | なし(Test gaps の 1) |
| M05 | 同じく `git status --porcelain=v1 --untracked-files=all;` を外した | 残った | なし(Test gaps の 1) |
| M06 | 手順 2 のコマンドの頭に `codex review > /dev/null 2>&1;` を足した | 落ちた | 行の一致と「codex を打たない」の検査に加え、`XreviewCommandOutcomes` の全 subtest と `…CommandsWriteTheStateAndTheDiff` で `calls = [["codex" "review"]]`(12 件)。stand-in の呼び出しの記録が効いている |
| M07 | 前の版の行を読まない文を消した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewTaskCarriesItsLinesInTheBody` |
| M08 | 手順 1 の確かめる行に `IMPLEMENTER_DRIVER:` を足した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`(前の言葉の一覧) |
| M08b | TASK の行の一覧に `IMPLEMENTER_DRIVER:` の項目を足した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewTaskCarriesItsLinesInTheBody`(行の一覧が 3 行でない) |
| M09 | report の `Reviewer:` の行から `、{{MODEL}}` を外した | 落ちた | `XreviewWritesTheReportUncommitted` |
| M10 | `## Triage context` の `Diff:` の行を前の `Reviewer output: …-out.md` に戻した | 落ちた | `XreviewReviewsOnTheSeatsOwnModel`、`XreviewWritesTheReportUncommitted` |
| M11 | `prompts.go` の `{{MODEL}}` を `ModelOrCLIDefault(vars.Model)` でなく `vars.Model` で置き換えた | 落ちた | `ShowsTheSeatsDriverAndModel` と `ReviewerPromptFileNamesTheSeatsDriverAndModel` の codex の model なしの行、`XreviewWritesTheReportUncommitted` |
| M12 | `spawn.go` の Spawn の呼び出しで `Model: ""` を渡した | 落ちた | `TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel/a_claude_seat_with_a_model` |
| M13 | 同じ所で `Driver: "claude"` を渡した | 落ちた | `TestOrgSpawn_ReviewerPromptFileNamesTheSeatsDriverAndModel/a_codex_seat_with_no_model` |
| M14 | leader のミッションの 5 を前の例外(「cross-review の差分を読んでいるときは、手順 12 と 9 のとおり、画面が動かなくても待ちます」)に戻した | 落ちた | `ReseatRuleSeatCheckItem`、`XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M14b | ミッションの 5 から「cross-review の差分を読んでいる reviewer は、ほかの座席と同じく扱います」だけを消した | 落ちた | `ReseatRuleSeatCheckItem` |
| M15 | 手順 12 から「10 分たっても画面が動かず … BLOCKED … を上げ、返事が来るまで待つ」を消した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M16 | cross-review の小節の待ちを前の「手順 12 の gate のときと同じく扱う」に戻した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M16b | 小節から「gate と違い、10 分たっても … BLOCKED を上げる」だけを消した | 落ちた | `XreviewBlockedFollowsTheGatesAndStallIsBounded` |
| M17 | leader の TASK の例に `IMPLEMENTER_DRIVER: claude` の行を足した | 落ちた | `XreviewTaskCarriesTheThreeLinesTheReviewerReads`、`XreviewReadsNoDriverAndNamesNoOtherModel` |
| M18 | 「本文に書くのはこの 3 行だけで、implementer の driver は … 読まず」の文を消した | 落ちた | 同じ 2 関数 |
| S1 | director の skill から「10 分たっても画面が動かなければ、それ以上は待たずに上のとおり決める(…)」を消した | 落ちた | `TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves` |
| S2 | org の skill の終わり方の 4 から、10 分で escalate する文を消した | 落ちた | `TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves` |
| S3 | org の skill の「役割ごとのモデル」から、同じ系列でも違う系列でも同じ、の文を消した | 落ちた | `TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel` |
| S4 | director の skill に前の例外の言葉(「reviewer が差分を読んでいれば待つ。」)を足しただけの変異(残した文には触れない) | 落ちた | `TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves`(前の言葉が残っていない、の検査) |

S1〜S4 は `.claude/skills/` の写し 1 つだけを替えた。テストは 4 写しを 1 つずつ読むので、どの写しを替えても同じく落ちる。

### Coverage

- Statement: 狭い run(`-run 'RenderRolePrompt|ReviewerPrompt|DryRun'`)の profile で、`RenderRolePrompt` 100%、`ModelOrCLIDefault` 100%。変えた 2 つの呼び出し(`spawn.go` の Spawn の `RenderRolePrompt` の呼び出しを含む 1116〜1122 行の塊と、dryRunSpawn の 2418〜2425 行の塊)はどちらも実行される
- Branch: Go の標準ツールでは測れない
- Function: 上のとおり
- Notes: この変更の中身の多くは雛形(`go:embed` の markdown)と skill と文書で、Go の coverage では測れない。雛形と skill は文を固定するテストと、雛形のコマンドを bash と zsh で走らせるテストで守られ、上の変異でその効きを確かめた

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| issue #360: 機能の org の reviewer 座席が座席の中から `codex exec` や `claude -p` を入れ子で呼び、codex の sandbox の中で incomplete になる、20 分待つ | 雛形の上では直った(座席の中で実際に動かしてはいない、Test gaps の 3) | `XreviewReviewsOnTheSeatsOwnModel` の前の言葉の検査、コマンドを走らせるテストの stand-in の呼び出しが 0 件、M06 で呼び出しを足すと落ちる |
| 読んでいるあいだに HEAD や作業ツリーが動くと、見ていない変更に pass が付く(計画助言の MEDIUM 1) | 比べは残っている | `XreviewCommandOutcomes` の 4 種類 × 2 shell が `git: changed`。M01・M02 で比べの部品を外すと落ちる |
| 計画助言の reviewer の行(`- reviewer の codex:` と `- reviewer の claude:`) | 変わらない | `Leader_FeatureOrgPlanAdvisory` で始まる 10 関数が絞った run で PASS |
| 上限の数え方のコマンド(`done:` と `k:`) | 変わらない | `Leader_XreviewStateCommand*` の 8 関数が PASS |
| `XREVIEW_BASE:` の値の確かめ(前の計画の F6〜F8) | 保たれている | `XreviewBaseCheckRunsGitCheckRefFormat`、`…RefusesBeforeItCallsGit`、`…BaseNamesWorkInTheCommands` が bash と zsh で PASS |
| 標準フローの `/cross-review` と `scripts/xreview-helpers.sh`(AC6) | 振る舞いは変わらない | `tests/test-xreview-helpers.sh` が `PASS: 53`、`FAIL: 0`。差分はコメントの 1 か所だけ(verify の報告の V7) |

### Test gaps

1. git の状態の比べの実行の行は 4 種類で、HEAD と未追跡のファイルの中身はそれぞれ 1 つの行が落とせる(M01、M02)。一方、作業ツリーの差分(`git diff`)、index の差分(`git diff --cached`)、`git status` の部品は、雛形と共有の定数 `leaderAdvisoryGitState` から一緒に外すと、どの実行の行も落ちない(M03〜M05)。雛形だけを替えればコマンドの行の一致で落ちる(M03t)ので、今の雛形は守られている。残る穴は、定数と雛形を一緒に変える変更を振る舞いで止める行がないこと。手順 2 の前から書き換えてある追跡中のファイルをもう一度書き換える行(M03)、stage 済みの変更にさらに stage を足す行(M04)、同じ中身の未追跡のファイルの名前を変える行(M05)を足せば、それぞれが落ちる。同じ定数を使う leader の計画助言にも同じことが言える(LOW)
2. `XreviewCommandsWriteTheStateAndTheDiff` の「前の版の leader の TASK から埋めたコマンドが 3 行の TASK から埋めたものと同じ」の比べは、テストの埋め方が `REPORT:` と `XREVIEW_BASE:` の値しか使わないので、組み立てのうえで常に等しく、単独では何も落とせない。前の版の TASK でも動くことは、行を読まないと言う文、3 行だけの一覧、前の言葉の検査が守っている(M07、M08、M08b で落ちる)。前の版の TASK を雛形のとおりに確かめる形のテスト(手順 1 の欠けた行の確かめを、前の版の本文で「欠けていない」と読む)はない(LOW)
3. reviewer 座席が herdr の pane の中で、本当に相手の CLI を呼ばずに自分のモデルで差分を読み、report を書くことは確かめていない。雛形の文とコマンドの確かめまでで、座席の振る舞い(モデルが雛形に従うか)は unit テストの外。依頼どおり本物の herdr server には触れていない
4. 前の版の reviewer 座席に新しい leader が 3 行の TASK を送ると QUESTION になる(計画の Assumptions)ことは、前の版の雛形がこの binary にないので確かめていない。release ノートで reseat を勧める扱い
5. dryRunSpawn の `RenderRolePrompt` に渡す `Driver` と `Model` は、dry run では雛形の長さ(prompt のファイルにするかどうか)にしか使われないので、外しても観測できない。固めるテストはないが、振る舞いの穴ではない

### Verdict(cycle 1)

- Pass: `./scripts/run-test.sh` rc 0(1,836 s、shell 46 本、Go 11 パッケージ、`(cached)` なし)、依頼の新しいテスト(bash と zsh でのコマンドの run、git の状態が変わる 4 種類、前の版の TASK の行、R1 の待ちの決まり)を含む絞った run の 127 関数、変異 25 個のうち 22 個と対照 1 個が落ちること
- Fail: 0
- Blocked: 0
- Verdict: pass。残った変異 3 つ(M03〜M05)は、雛形だけの変更なら行の一致で落ちるので、この変更の振る舞いの穴ではなく、Test gaps の 1 に記録した。/sync-docs に進んでよい
