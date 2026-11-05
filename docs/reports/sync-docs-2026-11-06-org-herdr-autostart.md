# sync-docs report: org-herdr-autostart

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`(AC1〜AC9、D1〜D5、R1〜R7、S1〜S5)。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の A
- Branch: `docs/spec-org-herdr-autostart-panes`。run 開始時の HEAD は `8ba6020c`、base は main `c206fdce`
- Pipeline cycle: 1
- 先行 report: self-review、verify(pass、LOW 1 件の V1 と INFO 3 件の I1〜I3)、test。verify の I2 a〜h が、この run への依頼の中身
- 制約: Go のファイルと help は変えていない。skill は動詞リファレンスの 3 行だけを直し、S4 が直した節は触っていない。plan と spec の箱、plan の Progress checklist は触っていない(orchestrator が付ける)。plan の digest は `9bda99914bec` のまま

## Summary

I2 の a〜g を直した。h は依頼どおり触っていない。verify が挙げなかった文書として `docs/quality/quality-gates.md` の gate の表に herdr のゲートの行を足し、tech-debt は依頼の 276 行と新しい 2 行のほかに、自分の sweep で当たった 7 行に carry-over の注を足した。

## Changes made

| File | Change |
|------|--------|
| `AGENTS.md` | `internal/org/` の項の、見張りのゲート(`watchgate.go`)の次に「herdr server gate(`herdrgate.go`)」の句を足した。Spawn(`--dry-run` を除く)・StartFeature・StartDirector(herdr の形)が herdr に何かを作らせる前に呼び、CLI が止まった `herdr server` を起動するものに設定する、と書いた。`internal/cli/` の項はサブコマンド名だけなので `org_herdr_server.go` は書いていない |
| `docs/specs/2026-08-01-org-runtime.md` | 「2026-11-06 改訂(docs/spec-org-herdr-autostart-panes)」の節を、2026-11-05 改訂と運用ノートの間に足した。(a) 対象の動詞と、`launch_cmd` の形が対象外であること。(b) 起動の 1 行、断る条件、人に頼むこと。(c) B(1 タブのペイン配置)は対象外で、本文は書き換えない |
| `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md` | 前提の「The herdr socket is up: open the herdr TUI, or run `herdr server` headless.」を、サーバーが動いていなくてよい(`ralph org spawn` が起動する)という文に替えた。自分で先に起動するのは、動く環境を選ぶときだけで、`isolated-herdr-server.md` を指す。`cmp` で 2 つは同一 |
| `README.md` | org の例の下の、見張りの自動起動の文のあとに 1 文を足した。start・spawn(`--dry-run` を除く)・director start(herdr の形)が止まった herdr のサーバーを起動し、stderr に書き、できなければ何も作らずに失敗する |
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 動詞リファレンスの `spawn`・`start`・`director` の行にゲートの文を足した。`spawn` はゲートの位置(見張りのすぐあと、spawn ロックと台帳の読みの前)、`spawn failed:` で終わること(`rejected` ではない)。`start` は両形の位置、worktree もブランチも作らずに失敗すること、`--timeout-ms` の期限に入らないこと(既存の文に「herdr のサーバーの起動」を足した)。`director` は herdr の形だけ、今の登録を確かめる前、`launch_cmd` は確かめないこと、登録で断られても起動したサーバーは残ること。root の `.claude` を Edit し、`scripts/sync-skills.sh` で root の `.agents` を作り直し、template の `.claude` に同じ Edit をして `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で template の `.agents` を作り直した。4 つの写しは `cmp` で一致 |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | 「Org runtime gates」の表に「Herdr server gate」の行を、Cross-org watch gate の行の次に足した(依頼にない追加。下の「依頼にない追加」)。2 つの写しで同じ行 |
| `docs/tech-debt/README.md` | 既存 8 行(180・191・195・201・276・319・371・390)の Trigger の末尾に 2026-11-06 の注を足し、276 行は Related にも plan を足した。新しい 2 行を末尾に足した。下の「tech-debt」 |
| `docs/reports/sync-docs-2026-11-06-org-herdr-autostart.md` | この report |
| `docs/insights/events/2026-11-05-org-herdr-autostart.jsonl` | `sync_docs` の行(cycle 1、pass)を `insights-append.sh --slug org-herdr-autostart --cycle auto` で足す。ファイル名の日付は UTC で、self-review と verify の行と同じファイル |

## I2 の a〜h

| 項 | 結果 |
|----|------|
| a. AGENTS.md の repo map | 直した。`herdrgate.go` を足した |
| b. 2026-08-01 の spec | 直した。改訂の節を足した |
| c. `internal/org/prompts/director.md` | 変えない。この雛形は herdr のサーバーの起動について何も言わず(`herdr`・`サーバー` の語は、人が pane に入って話す文と stop の頼み方だけ)、手順の全体は `/director` skill に従うと言う(26 行)。skill が「人に起動を頼まない」を持つので、headless の director にも届く。偽になっている文はない。spec の FR-A6 は雛形を名指すが、この判断で足りる、と読む。FR-A6 の箱を付けるかは orchestrator が決める |
| d. `codex-seat-permissions.md` | 直した。18 行目の前提は偽になっていたので、書き換えた。template の写しも同じ |
| e. README | 直した |
| f. `/org` の動詞リファレンス | 直した。4 つの写しが一致 |
| g. tech-debt 276 行と新しい 2 行 | 直した。276 行は「当たったが払わなかった」の注。新しい 2 行は末尾の 431・432 行(D1 の副作用、R7) |
| h. 箱 | 触っていない |

## tech-debt

注の形は 2026-11-05 の org-role-models と同じ(何を変えたか、払わなかった理由、「Carried over to the next change that meets this trigger」)。行数は base `c206fdce` と `8ba6020c` で `wc -l` を取った。

### 依頼の 276 行

`checkDirectorStart` が、herdr の形の start で、今の登録を見る前にゲートを呼ぶ。herdr の呼び出しが 2 つ増える(`herdr status server --json` と、止まっていれば `herdr server`)。codex の director が `start --plan` を打つ実機の run はしていない。plan の実機の記録(`docs/evidence/org-herdr-autostart-2026-11-06.md`)は `ralph org spawn` だけを通したので、2 つの前提はそのままで、「当たったが払わなかった」とした。Related に `docs/plans/active/2026-11-06-org-herdr-autostart.md` を足した(`/pr` の `archive-plan.sh` が書き換える)。

### 新しい 2 行(依頼)

| 行 | 中身 |
|----|------|
| 431 | ゲートが後の拒否より前にあるので、その後に拒否された動詞でも `herdr server` を起動していることがある。Spawn は `checkSpawnInput` と見張りのゲートのあと、spawn ロックと台帳の読みの前に呼ぶので、冪等の早期 return、envelope・permission・`--scope`・上限・予約、実行中の retry の拒否、regate の確かめで断られた spawn が当たる。StartFeature は `spawnPrecheckErr` の前、StartDirector は今の登録を見る前で、昇格したセッションや動いている headless の director の登録で断られた start が当たる。入力だけの検査(`checkSpawnInput`、director の入力の検査)で断られたものは起動しない。依頼の文面は「入力で断られた spawn」だったが、`checkSpawnInput` はゲートより前なので、コードの順どおりに書いた |
| 432 | `launch_cmd` の director start はゲートを呼ばないので、herdr で立てた headless の director の登録が残り herdr が止まっていると、今も `unknown` で断られる(plan R7)。`director stop --id <id> --force` か、herdr を手で起動すれば進める |

### 自分の sweep で足した 7 行

全 open 行の Trigger(取り消し線の外)を python で引き、この diff が変えた関数名・ファイル名・skill・recipe と突き合わせた。当たりは 1 行ずつ読んで、文面どおりに当たるものだけを残した。

| 行 | 当たったもの | 払わなかった理由 |
|----|--------------|------------------|
| 180((a) `/org` の command table) | `spawn`・`start`・`director` の行(表の行全体で 2,514 → 2,802、2,267 → 2,516、6,155 → 6,347 文字) | `stop` と `disband` のセル(1,060 と 1,217 文字)は変わらず、4 つの写しの表の組み直しは Scope の外。3 行が `stop` と `disband` より長くなったので、組み直しは 3 行も覆う、と書いた |
| 191(`Spawn` の doc) | `Spawn` の doc の step 0 と `StartFeature` の段落(`spawn.go` 2740 → 2767 行) | `idempotentRespawn` と `orgWideLimitsHelp` は変えていない(足した `orgHerdrServerHelp` は別の定数)。(b)〜(e) は同じ |
| 195(`newOrgStartCmd` の `RunE`) | `start <task>` の形にゲートをつなぐ 1 行と `--help` の段落 | herdr のゲートは leader のタスクの行ではないので、`start <task>` の leader に台帳の行が付かないことと、版のずれは同じ |
| 201((e)(f) `runOrgStartPlan`・`StartFeature`) | `runOrgStartPlan` の 1 行(`wireOrgHerdrServerGate`)と doc、`StartFeature` のゲートの呼び出しと doc(`feature.go` 776 → 785 行) | (e) の動かない枝と (f) の防御の枝(leader の入力の検査、台帳の読みの失敗)は今も動かない。新しい枝(ゲートが断る)は `TestStartFeature_HerdrServerGate_AFailureMakesNoWorktree` が動かす |
| 319(`director_start.go`) | ゲートの呼び出しと doc(1241 → 1260 行)。同じ行の他のファイルでは `internal/org/driver/herdr_test.go` だけが変わった(1464 → 1693 行、+229。`ServerStatus` のテスト) | move-only の分割は Scope の外 |
| 371(recipe 5 つ) | `isolated-herdr-server.md`(S4 の 1 段落)と `codex-seat-permissions.md`(この sync-docs の前提の文)。どちらも 2 つの写し | `docs/evidence/` と tech-debt の引用は足しても直してもいない。新しい参照 1 つは `ralph init` が配る recipe を指す |
| 390((b) `StartFeature` の検査) | `StartFeature` にゲートの検査が増えた | ゲートは台帳の読みと `regateSpawnErr` の前なので、(b) の窓は広がらず狭まらない。(a) の `runWorktreeScript` は変えていない |

### 退けたもの

| 行 | 退けた理由 |
|----|------------|
| 318(V1: 「2 つの herdr サーバーで director が 2 人になる」) | Trigger の「the next change to how ralph reaches the herdr server」に当たる。verify の V1 が、cross-review の triage で直すか 318 行の注にするかを決めるとしたので、この run では触っていない |
| 275(`startDirector` の拒否の順) | CLI の `startDirector` は `wireOrgHerdrServerGate` の呼び出しを 1 行足しただけで、拒否の順は変えていない。ゲートの拒否は org 層の `StartDirector` にある |
| 262・379(`/director` skill の受信箱の節、送って確かめる手順) | S4 が直した節は「立てる」「機能ごとの出力先」「してはいけないこと」で、受信箱の節にも送って確かめる手順にも触れていない |
| 365(`spawn.go` は 1 つ前の数 +300 を超えたら) | `spawn.go` は +27 行 |
| 397(`internal/cli` の TestMain の agmsg の固定) | `main_test.go` の +11 行はゲートの差し替えで、agmsg の固定は変えていない |
| 118・136・159 | 語が重なっただけで、Trigger の対象には触れていない |

## 依頼にない追加

`docs/quality/quality-gates.md` の「Org runtime gates」の表には、見張りのゲートの行がある。herdr のゲートも同じ形の「動詞が前提を確かめて、足りなければ何も作らずに断る」ゲートなので、行を足した。root と template の「Org runtime gates」の表は、Quality pipeline gate の行を除いて同じ文言なので、同じ行を両方に足した(`quality-gates.md` は `check-sync.sh` の KNOWN_DIFF で、違いは通る)。依頼の範囲から外れる追加なので、要らなければこの行だけ戻せばよい。

## 確かめたこと

- `./scripts/check-sync.sh`: DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、PASS
- `./scripts/check-skill-sync.sh`: 14 skill が一致
- `cmp` で、`/org` の skill の root と template の `.claude` と `.agents` が、root どうし、template どうしで一致
- `go test ./internal/org/ -run 'Skill|SendDefaults|ReportNames|Followup|Verification|PromptsVerification'`: ok(`/org` の skill を読むテスト 7 ファイル)
- tech-debt の編集した 8 行と新しい 2 行を python で数えた。どれも 5 セルで、`| ` で始まり ` |` で終わる

## 残っていること(orchestrator へ)

- plan の AC1〜AC9 と、spec の FR-A1〜A6・AC-A1〜A6 の箱は `[ ]` のまま。Progress checklist の Review・Verification・Test の箱も触っていない
- V1(tech-debt 318 行)は cross-review の triage で決める
- I1(実機の記録の `HERDR_TAB_ID` の注)と I3(help の「refused」の言い回し)は、この run の範囲外で、そのまま
- `/org` skill の spawn・start・director の行は、また長くなった(上の 180 行の注)

---

## Run 2(cycle 2)

この線より上は Run 1(cycle 1)の本文で、1 バイトも変えていない。Run 2 の結果はこの節だけに書いた。

- Date: 2026-11-06
- Plan: `docs/plans/active/2026-11-06-org-herdr-autostart.md`。spec は `docs/specs/2026-11-06-org-herdr-autostart-panes.md` の A
- Branch: `docs/spec-org-herdr-autostart-panes`。run 開始時の HEAD は `9fc316ef`、base は main `c206fdce`
- Pipeline cycle: 2(上限 2 の最後の run)
- 先行 report(cycle 2): self-review `227bfd28`(LOW 5 件。新しい所見は C2-1)、verify `5883b18b`(pass。LOW 2 件の V1・V2-1、INFO 4 件の I2-1〜I2-4)、test `9fc316ef`(pass)
- 制約: plan と spec の箱、plan の Progress checklist は触っていない(orchestrator が付ける)。Go は `internal/cli/org_director_headless.go` の help の文字列だけで、テストのファイルは変えていない

### Summary

C2-1 と I2-1(ゲートの失敗を skill と spec が「断る」と書く語)、V2-1(`director start --help` の「runs no herdr」)、I2-2・I2-3(tech-debt の注と行の書き漏れ)を直した。C2-1 は register に足さず、skill を直した。Run 1 が範囲外とした I1(実機の記録の `HERDR_TAB_ID`)と I3(help の「refused」)は、f0edc152 で直っていたので、解消とする。

### Changes made(Run 2)

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 「前提」の最初の項(35 行)の「何も作らずに断られ」を「何も作らずに失敗し」に、`director` の行の「何も書かず herdr にも何も作らずに断られる」を「…失敗する」に替えた。2 語とも同じ字数なので、`spawn`・`start`・`director` の行の長さ(2,802、2,516、6,347 文字)は変わらない。root の `.claude` と template の `.claude` を Edit し、`scripts/sync-skills.sh` を 2 回(root、`CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills`)回して `.agents` を作り直した。4 つの写しは `cmp` で一致 |
| `.claude/skills/director/SKILL.md` と、その 3 つの写し | 「立てる」の 1(162 行)の「何も書かずに断られ」を「何も書かずに失敗し」に替えた。624 行の start --plan の手順(「worktree もブランチも作らずに失敗し」)と同じ形になった。同じく 4 つの写しは `cmp` で一致 |
| `docs/specs/2026-08-01-org-runtime.md` | 2026-11-06 改訂の (b) の「worktree も台帳の記録も作らずに断る」を「…失敗する」に替えた |
| `internal/cli/org_director_headless.go` | `ralph org director start --help` の拒否の段落。見出しの文を「start is refused, and records nothing, when」にし(「and runs no herdr」を外した)、「start has no --replace.」のあとに 1 文を足した。「All of these refusals but the one for a promoted session run no herdr. That one comes after herdr is run: a start in herdr has made sure the herdr server runs by then (see below), so it has read `herdr status server --json` and has started `herdr server` if none ran, and a server it started stays up.」。「writes nothing」は「records nothing」にした。ゲートが起動した server は台帳のディレクトリに `herdr-launch.log` を書くので、「何も書かない」は言えず、director.jsonl と manifest に何も記録しない、が言えるためである |
| `docs/tech-debt/README.md` | 191・430・431・434・180 行の 5 行。下の「tech-debt」 |
| `docs/reports/sync-docs-2026-11-06-org-herdr-autostart.md` | この節 |
| `docs/insights/events/2026-11-05-org-herdr-autostart.jsonl` | `sync_docs` の行(cycle 2、pass)を `insights-append.sh --slug org-herdr-autostart --cycle auto` で足した。`git status --short` は既存の jsonl を `M` と出しただけで、新しいファイルはできていない |

### 依頼の項目

| 項目 | 結果 |
|------|------|
| 1. C2-1・I2-1(語) | 直した。skill は 2 skill × 4 つの写し、spec は (b)。ほかに残る「断られ」は本当の断り(`enabled = false`、昇格したセッションの登録、見張りのゲート)で、直していない。`/org` の `director` の行の「昇格したセッションの登録などで断られた start でも、起動したサーバーは動き続ける」もそのままでよい。この語で書いた文を固定するテストはない(`git grep` で 0 件) |
| 2. V2-1(help) | 直した。下の「help の確かめ」 |
| 3. I2-2(191・430・288 行) | 191 は注を直した。430 の (e) は注を足した。288 は当たらない。下の「tech-debt」 |
| 4. I2-3(431・434 行) | 431 に見張りのゲートの場合を足した。434 の (c) に置き場所を足した。下の「tech-debt」 |
| Run 1 の I1・I3 | 解消(f0edc152)。I1 は `docs/evidence/org-herdr-autostart-2026-11-06.md` の表と段落が直り、I3 は 3 つの `--help` の「refused」が「fails」になった。I3 の skill 側の語は、この Run の項目 1 で直した |

### help の確かめ

- `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Help|Director'` の出力の末尾: `GOFMT_OK`、`VET_OK`、`ok  	github.com/yoshpy-dev/ralph/internal/cli	47.198s`、`rc=0`
- 新しい文は、help を固定するテストの 3 つの文字列を壊さない場所に置いた。`TestOrgDirectorStart_HelpStatesTheCleanup` の「…while a promoted session is attached (detach it; for a headless director, see below), and inside a herdr pane」と、`org_director_test.go` の 2 つ(「…run start from a terminal outside herdr. start has no --replace.」と「With launch_cmd, HERDR_PANE_ID does not refuse start, which runs no herdr then, …」)は、どれも文の続きがそのままで通る。古い「writes nothing and runs no herdr」を固定するテストはなく、テストのファイルは変えていない
- 新しい文を固定するテストは、まだない。verify の V2-1 が勧めた「昇格したセッションの場合が『runs no herdr』の一覧に並ばないことを見るテスト」は、この Run の範囲(古い文を固定するテストだけ)の外なので足していない。足すなら `TestOrgDirectorStart_HelpStatesTheCleanup` の `want` に新しい文を 1 つ足し、古い「writes nothing and runs no herdr」が help に無いことを見る 1 行が最小である。この欠けは 431 行に「テストが固定しない」と書いた
- コミット前の作業ツリーから作った binary の `ralph org director start --help` で、段落の見え方を読んだ(1 行 72 字前後、バッククォートの語が行をまたがない)

### tech-debt

行の編集は列ごとに短い anchor で行い(9 回、anchor は事前に README 全体で数えて全部 1)、編集後に `git show HEAD:` と突き合わせた。変わった行は 180・191・430・431・434 の 5 行だけで、どれも 5 セル、バッククォートと `~~` はセルごとに偶数、`file:line` の引用はない。

| 行 | 変更 |
|----|------|
| 191(`Spawn` の doc) | 2026-11-06 の注を直した。Run 1 は step 0 だけと「2740 から 2767 行」を書いたが、f0edc152 が step 1(冪等の早期 return の前でもゲートが動く)と `Spawn` の本文の comment も書き換えた。`spawn.go` は 2740 行(c206fdce)から 2778 行で、2767 行は最初の編集(b8580f0a と d6951690)の値、f0edc152 が +11。3 つの commit で `wc -l` を取って確かめた。(b)〜(e) は払っていない、という結論は変わらない |
| 430(org-role-models の LOW) | (e) の Trigger「the next change to `ResolveRoleModel` or to the idempotent respawn」に、f0edc152 が当たった。step 1 の doc を書き換え、`TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo` が冪等な打ち直しもゲートを通ることを固定した。払わなかった: `ResolveRoleModel` も (e) の CLI の 2 つの拒否も変えておらず、doc は 2 つの拒否が早期 return の前に動くことをまだ言わない。Trigger の末尾に 2026-11-06 の注を足し、Related に `docs/plans/active/2026-11-06-org-herdr-autostart.md` を足した(`/pr` の `archive-plan.sh` が `archive/` に直す) |
| 431(ゲートの副作用) | Debt の列に、見張りのゲート(`checkWatchGate`)で断られた herdr の形の director start も、サーバーを起動したまま残す、を足した。`checkDirectorStart` は見張りのゲートを今の登録の確認のあとに置くので、ゲートの後ろにある。`/director` skill の start の 3 は見張りの断りを書くがサーバーのことを言わず、help の見張りの文も言わない。V2-1 の help の文は昇格したセッションについて言うが、固定するテストがない、とも書いた。Why deferred に「見張りのゲートはその確認のあと」を、Trigger に「その関数の検査の順(`director start` の help が書く)の変更」を足した |
| 434(L2〜L4) | (c) の「10 秒が書かれた場所」に、`/org` skill の「前提」・`spawn`・`start` の行、`docs/quality/quality-gates.md`(2 つの写し)、2026-08-01 の spec の 2026-11-06 改訂 (b)、feature spec(`docs/specs/2026-11-06-org-herdr-autostart-panes.md` の FR-A1・FR-A5・AC-A4)を足した。最後の feature spec は依頼の 3 つのほかに、10 秒を書いていると確かめて足した。Impact の (c) も「help と skill」から「help、skill、quality-gates の行、spec」に直した。`/org` の `director` の行の「10 秒」は launch_cmd と stop の待ちで、サーバーの待ちではない |
| 180(`/org` の command table) | 2026-11-06 の注に 1 文足した。この Run が `director` の行の 1 語を替えた(失敗の語)が、長さは 6,347 文字のままで、注の数字はそのまま成り立つ |
| 288(冪等な枝の `regateSpawnErr`) | 当たらない。Trigger は「the next change to the idempotent branch of `Spawn`」で、枝のコード(locked phase の `idempotentRespawn` の return)は base から 1 行も変わっていない。変わったのは、枝の前に置いたゲートの呼び出しと、その枝を説明する doc の 2 か所(step 1 と、`ResolvePermissionMode` の上の comment)で、どちらも枝が `regateSpawnErr` を呼ぶかは動かさない。doc の変更は 191 行の Trigger が受ける。枝が新しい検査を通るようになった点を「変更」と読めば当たるが、行が求める直し(`regateSpawnErr` を枝にも呼ぶ)に近づく変更ではないので、注は足さなかった。読みが違うなら 430 と同じ形の注を足せばよい |

この Run の編集そのものが当たる Trigger も、全 open 行の Trigger(取り消し線の外)を python で引いて探した。skill・`/director`・`org_director_headless`・`director start` の help・2026-08-01 の spec を指す行のうち、当たったのは 180 だけである。

| 退けた行 | 理由 |
|----------|------|
| 275(`startDirector` の拒否の順) | help の文字列を変えただけで、`startDirector` の拒否の順は変えていない |
| 262・269・379・380 | `/director` skill の受信箱、始める step 3、送って確かめる手順、wait の終了コード。この Run が替えた「立てる」の 1 の 1 語には触れない |
| 276・318 | Trigger は「the herdr calls of `StartDirector`」(276)と「how ralph reaches the herdr server」(318)。この Run は `StartDirector` も herdr の呼び出しも、herdr への到達も変えていない(help の文字列と skill の語だけ) |
| 365・420・421 | テストのファイルの行数や定数。この Run はテストを変えていない |

### 確かめたこと

- `./scripts/check-sync.sh`: DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6、PASS(rc 0)
- `./scripts/check-skill-sync.sh`: 14 skill が一致(rc 0)
- `cmp`: `/org` と `/director` の skill が、root の `.claude`、root の `.agents`、template の `.claude`、template の `.agents` で一致
- `go test ./internal/org/ -run 'Skill|SendDefaults|ReportNames|Followup|Verification|PromptsVerification|Director'`: ok(51.867s)。skill の文面を読むテストが、この Run の語の替えで落ちないことの確かめ
- `go test ./internal/cli/ -run 'Help|Director'`: ok(47.198s)、上の「help の確かめ」
- tech-debt: 5 行だけが変わり、構造は保たれた(上の「tech-debt」の冒頭)

### 残っていること(orchestrator へ)

- plan の AC1〜AC9、spec の FR-A1〜A6・AC-A1〜A6、plan の Progress の Review・Verification・Test の箱は `[ ]` のまま(verify の I2-4)。AC2 と AC5 は「一部」「偽物だけ」なので、付けるなら PR の本文にその範囲を書く
- Go のコードの comment には、ゲートの失敗を「refuses」「A refusal」と書く箇所が残る(`internal/org/herdrgate.go` の冒頭、`internal/cli/org_herdr_server.go` の冒頭、`Org.HerdrServerGate` の doc)。どれも同じ comment の中で `SpawnOutcomeFailed` と書いており、利用者の目に触れる語ではないので、変えていない
- V2-1 の help の新しい文を固定するテストは、まだ無い(上の「help の確かめ」)
- この Run は Go の文字列(help)と skill 8 ファイルを変えた。cycle 2 は上限の最後の run である
