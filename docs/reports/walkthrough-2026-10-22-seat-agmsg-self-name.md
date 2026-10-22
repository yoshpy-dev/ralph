# Walkthrough: seat-agmsg-self-name(座席の環境 4a: F3、ALERT の計測、#247 の記録)

- Date: 2026-10-22
- Plan: docs/plans/active/2026-10-22-seat-agmsg-self-name.md(/pr で archive に移す)
- PR: #248
- Branch: fix/seat-agmsg-self-name(base 1e47c309、この walkthrough の前の HEAD 4fb0c663。21 commits、37 files、+1914/-100)
- 読む範囲: コードとテストは 14 files(+1094/-51)。Go の本体は `internal/org/driver/` の 3 files、`internal/org/director_cmd.go`、`internal/cli/` の組み立て 2 files で、合わせて約 200 行。残りはテスト(`herdr_test.go` が 418 行)。文書は skill 4 つの写しを含む 13 files。残りは計画、evidence、後工程の報告 5 本と triage、insight event
- 後工程は 1 回(cycle 1)で終え、cross-review の指摘は 0 件

```
何を直すか → 計画 → 計測(S1) → agmsg の呼び出し(S2) → herdr の pane(S3) → 起動スクリプト(S4) → 記録(S5) → 後工程と、その中の直し → AC の根拠 → 確かめていないこと
```

## 1. この PR が何か

- 試行の所見を直す系列の 4 本目の前半(1 本目は仕様の印の #245、2 本目は記録の文字の #246、3 本目はテストの安定の #247)。4 本目は consult の判定で 4a と 4b に分け、F4 と F11 は 4b に送った
- F3: agmsg 1.5.1 は herdr の pane の中で打たれると、その pane の agent と pane の名前を team と agent から作った名前に書き換える。ralph は座席の agent を `<org_id>_<seat_id>` で引くので、書き換えのあとは `agent_not_found` になる。ralph 自身が打つ agmsg(join、HELLO の send など)と、座席が自分で打つ `inbox.sh` の両方で起きる
- tech-debt の「leader への ALERT」の行: 見張りの ALERT が 10 秒で時間切れになった原因が書き換えかどうかを計測で確かめる。示せなければ行は開いたまま
- V4: 根拠の表と仕様に残る「fix/test-stability」を「#247」に直す

## 2. 読む順(commit ごと)

1. 4b85e5d8 計画
   - Objective、Scope、Non-goals、Design decisions(D1〜D4)、AC1〜AC8 を読む。承認の digest は `dc40c452f092` で、後工程のあいだ本文は変えていない(Progress checklist と AC の checkbox だけが動く)
   - D1: `driver.Runner` を変えず、任意の interface `EnvRunner` を足す。argv に `env` を前置する案は、stub の試験の argv の assert をすべて変えるのでやめた
   - D2: `HerdrClient` の `TabCreate` の引数は変えず、`driver.Herdr` に pane に足す環境の欄を持たせる
   - D3: 古い herdr は `herdr tab create --help` で確かめ、`--env` がなければ付けずに作って警告する。確かめは同じ値で 1 回
   - D4: ALERT の行は、原因が書き換えだと示せたときだけ閉じる
   - Codex の計画の助言 2 件(受け取る側に届くか、本物の座席の agent の名前)は、AC6 の (d) と (e) に入れた
2. 782ceebf S1 計測(`docs/evidence/agmsg-self-name-2026-10-22.md`)
   - 分離した herdr server(`/tmp` の短い directory、偽の HOME、XDG の変数、`HERDR_SOCKET_PATH`)と agmsg の写しで打った。ユーザーの herdr server と Orca には触れていない
   - (a) `tab create --env` は足すだけで、HOME と PATH は残る。D2 はそのまま
   - (b)〜(d) off の pane の agmsg は herdr を打たず、off で送ったメッセージは受け取る側に届く
   - (e) 本物の claude の座席で、off の pane なら名前が残り、`--env` のない対照では `agent_not_found` になった
   - (c)・(f) 見張りの環境での上乗せは中央値で約 65 ms。座席の pane の環境では on が約 1〜2 秒遅い。ALERT の時間切れは再現しなかったので行は開いたまま。F4 に効いていた可能性を 4b に渡す
   - 見ること: 「触れていないもの」と「計測で残ったもの」(`~/.claude.json` の信頼の項目 1 つ、session の記録 2 つ)
3. b0a607d9 S2 agmsg の呼び出し
   - `internal/org/driver/driver.go:36` の `EnvRunner` と、`driver.go:58` の `ExecRunner.RunEnv`(`append(os.Environ(), env...)`。os/exec は同じ key の最後の値を採るので、親の on より off が勝つ)。`Run` と `RunEnv` は同じ `execRun` を通る
   - `internal/org/driver/agmsg.go:48` の `Agmsg.run`。Runner が `EnvRunner` なら `RunEnv(ctx, []string{AgmsgSelfNameOff}, "bash", ...)`、そうでなければ今までの `Run`。7 つの呼び出しはすべてここを通る
   - `internal/cli/org_runtime_env_test.go` の `TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff` が、本番の組み立ての Runner が `EnvRunner` であることと、親が on でも子が off を読むことを固める。Runner を包む型を挟んで黙って外れる退行を拾う
4. f16423da S3 herdr の pane
   - `internal/org/driver/herdr.go:61` の `NewHerdr(r, paneEnv...)`。欄は `PaneEnv`、`Warn`、確かめの結果を持つ `envProbe`(pointer なので、値をコピーしても確かめは 1 回)
   - `herdr.go:241` の `TabCreate` は、`PaneEnv` が空なら今までどおり。空でなければ `paneEnvSupported`(`herdr.go:283`)が `--help` を 3 秒の上限で 1 回打ち、`herdrHelpEnvFlag`(`herdr.go:85`)で `--env` を探す。`--envelope` や `--env-file` には当たらない
   - 呼び手の ctx が切れた失敗は結果として覚えない。ほかの失敗と `--env` がない場合は覚え、警告を 1 行出す
   - 本番の組み立ては `internal/cli/org.go:356` と `internal/cli/org_director_headless.go:788` の 2 か所で、どちらも `NewHerdr(…, driver.AgmsgSelfNameOff)`
   - CLI の試験の herdr の stub(`internal/cli/org_test.go`)は `tab create --help` に 0.7.5 と同じ形の help を返す。help の呼び出しは log と id の採番に入らないので、既存の試験の数は変わらない
   - 見ること: mutex の中は help 1 回と regexp と警告だけで、再入しないこと
5. a34612fd S4 起動スクリプト
   - `internal/org/director_cmd.go:323`。`RALPH_DIRECTOR_TOKEN`(と、あれば `RALPH_ORG_AGMSG_HOME`)の export のあと、check-in と exec の前に `AGMSG_SELF_NAME='off'` を export する。試験は全文と順序、端末の on に勝つこと、check-in とエージェントの両方で off が見えることを固める
   - `internal/org/spawn.go` は HerdrClient のコメントの配線の文だけを直した
6. 5f3a37ad、08d091f5、dd0b77d0 S5 記録
   - `docs/tech-debt/README.md`: F3 の行を RESOLVED にした(上に RESOLVED のコメント、打ち消し線)。「leader への ALERT」の行に計測の数字と新しい再開の条件を足し、開いたまま
   - `docs/evidence/org-spec-evidence-2026-10-21.md` と `docs/specs/2026-10-07-org-multi-org-director.md`: F3 と D-F3 は「いいえ(今は)」。F3 を直すと D-F3 の回り道が要らなくなるので、consult の判定で FR-9 を `[x]` にした(F8・F10 の前例と同じ)。印の数は仕様が 22/10、根拠の表が 21/10。「fix/test-stability」を「#247」に直し、PR 一覧を 30 本にした
7. 後工程と、その中の直し
   - 995f3b0f self-review(マージ可、LOW 4)。4991744e で L1(`PaneEnv` に秘密を入れないこと、警告の doc の文)、L2(`AgmsgSelfNameEnv` を足し、起動スクリプトが `strings.Cut` をやめる)、L4(evidence の (c) の数え方)を直し、22b36fda で報告に書いた。L3 は残した
   - 73fd8c78 verify(pass、LOW 2・INFO 3)
   - 8bddd2c6 test(pass、mutant 40 本中 35 本が赤)。9c88b759 で G1(Warn が nil のとき os.Stderr に出すことと、CLI の水準の AC4)と G2(正規表現の端)のテストを足し、警告の文を `warning:` で始めて座席と headless の director の両方を指す文にした(G5、V4)。c7d4d85b で報告に書いた
   - ab5c6c96 sync-docs(recipe とその写し、`/director` と `/org` の skill とその写し、README、tech-debt の F3 の RESOLVED、FR-9 の注)、06222876 その報告
   - 4fb0c663 cross-review の triage(指摘 0)
   - 計画の Progress: 20fa1808、5eaeaf95

## 3. AC の根拠

| AC | 根拠 |
|----|------|
| AC1 | S2。`TestAgmsg_EnvRunner_EveryCallRunsWithSelfNameOff` が 7 つの呼び出しを、`TestExecRunner_RunEnv_AddsToTheParentEnvironment` が本物の `sh` の子で親の on に off が勝つことを見る |
| AC2 | S2。`TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff` |
| AC3 | S3。`TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt`、`...ConcurrentCallsCheckOnce`、`TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff`、spawn と director start の CLI の試験 |
| AC4 | S3 と 9c88b759。`TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce`(4 通り)、`TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr`、`TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr`(spawn と director start が成功し、`--env` が付かず、stderr に 1 行) |
| AC5 | S4。`TestDirectorLaunchScript_TextQuotesEveryValue`、`...ChecksInThenExecsTheAgent`、`TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn` |
| AC6 | S1。verify が (c) の 8 つの値を生の出力と突き合わせて一致した |
| AC7 | S5。verify が印の数(22・10、21・10)、30 本と 27 本、「fix/test-stability」の置き換えを数え直して合った |
| AC8 | `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` は 08d091f5 で rc 0(760 秒)、テストを 5eaeaf95 の tree で打った回も rc 0(852 秒、最後の secret scan は 4fb0c663 まで)。go1.25.8 では tester が 73fd8c78 で、implementer が 9c88b759 で driver・org・cli を通した |

## 4. 確かめていないこと

- 新しい binary で本物の herdr に `ralph org spawn` と `ralph org director start` を打つ実機の確認。S1 の (e) は、手で作った pane に claude の座席を起動して確かめた
- codex の座席の環境
- 0.7.5 より古い herdr が `tab create --help` を help として扱わずに tab を作るかどうか。そうなら確かめのときに tab が 1 つできる
- ALERT の時間切れの原因。見張りの環境での上乗せは約 65 ms で、今回は時間切れが起きなかった
- 試行で書き換えられた herdr の名前は戻していない
- tech-debt の F4 の行は、RESOLVED にした F3 の行を "the row above" で指したまま。記録に残る「fix/seat-agmsg-self-name」も #248 に置き換えていない。どちらも 4b でする
- Linux では、手元で走らせていない(CI が見る)
