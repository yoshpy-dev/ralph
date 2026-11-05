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
