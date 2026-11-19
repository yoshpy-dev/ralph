# sync-docs report: doctor-alias-herdr-guard

- Date: 2026-11-19
- Plan: `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`(承認の digest `2c2d8e5932b3`。AC の箱を付けたあとに `./scripts/plan-visual.sh digest` で計算し直した値と一致する)
- Branch: `fix/doctor-alias-herdr-guard`。base は origin/main `a8c5e289`、run 開始時の HEAD は `16ec3e20`(test の記録の commit)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
- 先行 report: self-review は `be254791`(Merge yes)。verify は 2 回目で pass(`ce3fcfcd`、1 回目は `7f87aa56` で fail し、`b38e97ba` で直した)。test は `27b08bc6` で pass、G1 は `ac0c403e` で閉じた。引いた commit は各 report を含む commit で、report の中の `HEAD:` の行の値ではない
- 見た範囲: `git diff a8c5e289...HEAD`、verify が渡した 3 項目、test の G1 から G4。tech-debt の全行の Trigger を自分で走査した(下の「Trigger の走査」)
- 差分の大きさ(`/pr` が walkthrough を決める材料。この commit が足す前の HEAD `16ec3e20` の値): `git diff --shortstat a8c5e289...HEAD` は 17 ファイル、+1,960/-77。内訳は Go が +1,393/-52(`doctor_shell_alias.go` +530/-50、そのテスト +859/-0、`doctor.go` +4/-2)、report・計画・insight event が +442/-0、残りの 9 ファイル(recipe 4、org skill 4、tech-debt)が +125/-25。この commit は docs だけを足す
- 制約: Go のコードは触っていない。テストは回していない。`check-sync.sh`、`check-skill-sync.sh`、plan の digest、tech-debt の計画の参照は回した(下の「確認」)

## Summary

verify が渡した 3 項目を片付けた。tech-debt の 373 行目(5 つの recipe の Trigger)と 417 行目(800 行を超えた 5 ファイルの Trigger)に、この branch が Trigger を満たしたが払っていないと日付つきで書き、計画の AC 7 つに箱を付けた。

走査で、handoff の 2 行のほかに、138 行目の行数が古くなっていた。test の G1 を閉じた `ac0c403e` がテストの表に 1 行足したので、`doctor_shell_alias_test.go` は 2576 行でなく 2577 行になっていた。直した。新しく Trigger を満たした行はほかになかった。

`AGENTS.md`、`README.md`、`CLAUDE.md` は変えない。recipe と org skill の文は、コードと食い違っていなかった。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md` | 138、373、417 の 3 行。ほかの行は変わっていない。下の「tech-debt」 |
| `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md` | Acceptance criteria の AC1 から AC7 を `[x]` に。`git diff --numstat` は 7/7 で、本文はほかに変えていない |
| `docs/reports/sync-docs-2026-11-19-doctor-alias-herdr-guard.md` | この report |
| `docs/insights/events/2026-11-19-doctor-alias-herdr-guard.jsonl` | sync_docs の 1 行 |

## tech-debt

### 373 行(`templates/base/docs/recipes/` が引く `docs/evidence/` の行)

Trigger は「The next edit of any of these five recipes」。この branch は `codex-seat-permissions.md`(shell alias の段落に 3 つ目の回避策、doctor が読む形の文)と `isolated-herdr-server.md`(`Shell aliases` 節に 1 文)を、どちらも 2 写しとも編集したので満たしている。払っていない。引用は足さず、言い換えもしていない。`grep -o` で数えた引用の数は、origin/main `a8c5e289` と、recipe を最後に変えた `754fd9a3` で同じだった。

| recipe | `docs/evidence/` の引用 | `docs/tech-debt/README.md` の引用 |
|---|---|---|
| `codex-seat-permissions.md` | 3 件、3 ファイル | 0 件 |
| `isolated-herdr-server.md` | 14 件、7 ファイル | 1 件 |

Trigger のセルの末尾に 2026-11-19 の note を足した。形は前の branch の note(`2026-11-15: its cycle 3 …`)に合わせ、末尾は「The row stays open」にした。払わなかった理由は計画の Scope の「文書」が shell alias の箇所だけを名指していること。Related の末尾に計画を `docs/plans/active/…`(Scope, documents; the carry-over note)で足した。`archive/` は `/pr` の `archive-plan.sh` が書き換えるので先には書かない。

### 417 行(800 行を超えた 5 ファイルの行)

Trigger は「The next change to any of the five」。5 つのうち `internal/cli/doctor.go` だけが変わった。`1af2fcee` が Check 8b の comment に 2 行足し、1648 行が 1650 行になった(`git show a8c5e289:internal/cli/doctor.go | wc -l` と HEAD の `wc -l` で測った。この branch で `doctor.go` を変えた commit はこの 1 つだけ)。ほかの 4 つ(`tests/test-detect-changed-languages.sh`、`tests/test-plan-visual.sh`、`tests/test-secret-scan-branch.sh`、`internal/config/config.go`)は `git diff --stat` が空だった。

払っていない。comment だけの変更で、move-only の分割は計画の Scope の外。計画の Scope は `doctor_shell_alias.go` と文書を名指し、`doctor.go` の Check 8b の comment は Verify plan の「Documentation drift to check」に出る。note はこの事実のとおりに書き、Scope が `doctor.go` を名指すとは書かなかった。ほかの note と同じ「Carried over to the next change that meets this trigger」で終えた。Related の末尾に計画を `docs/plans/active/…`(Verify plan; the carry-over note)で足した。

### 138 行(この branch が足した 800 行の行)

行は `doctor_shell_alias.go` を 1448 行、テストを 2576 行(`4fc29299` で数えた値)としていた。`git log 4fc29299..HEAD` が 2 つのファイルに見つけるのは `ac0c403e`(テストの表に 1 行)だけで、HEAD の `wc -l` は本体 1448、テスト 2577。「counted at 4fc29299」を「counted at ac0c403e, the last commit that changed either file」に、2576 を 2577 にした。136 行目の「counted … at 4fc29299」は本体だけの数(968 から 1448)で、本体は `4fc29299` のあと変わっていないので、そのままにした。

## Trigger の走査

handoff は 373 行と 417 行を挙げた。tech-debt の全行から、Trigger が全部 `~~` の行を除き、Trigger のセルをこの branch が変えた物の語(`doctor_shell_alias`、`doctor.go`、`checkShellAliases`、`shellAliasDef`、`Shell aliases`、`HERDR_PANE_ID`、recipe 2 つの名前、`skills/org`、`/org skill`、`sync-skills`、`check-skill-sync`、`alias`、`doctor`、`any of`、`the five` など)で探した(`sweep.py`、scratch)。当たったのは 62 行で、Trigger のセルの頭(Trigger が名指す物)を 62 行すべてで読んだ。`doctor`、`alias`、`any of`、`these files` の語だけで当たった行が多く、どれも別の Check、別のファイル、別の関数が Trigger だった。handoff にない新しい当たりはなかった。

| 行(先頭の語) | 判断 | 理由 |
|---|---|---|
| 373、417 | 当たり。note を足した | 上のとおり |
| 136「`checkShellAliases` (doctor's "Shell aliases…")」 | この branch が済ませている | Trigger は `doctor_shell_alias.go` の次の変更。self-review の M1 で 2026-11-19 の note と Related が入っている(`7cf73b5a`) |
| 137、138 | この branch が足した行 | 138 は上の行数の直しだけ |
| 182「Two documentation findings of org-stop-all」(a) | 当たっていない | Trigger は `/org` skill の command table の編集。この branch の編集は前提の節(`.claude/skills/org/SKILL.md` の 216〜241 行目あたりの 2 つの hunk)で、table は 374 行目から。`stop` と `disband` のセルは変わっていない |
| 117、118、329、476、517、522 | 当たっていない | 117 と 118 の Trigger は `quality-gates.md` と `adding-a-language-pack.md` の変更。329 は `isolated-herdr-server.md` を最初から最後まで live で通すこと(RESOLVED)。476、517、522 は send の画面と reseat の項。この branch は `quality-gates.md` を触らず、`isolated-herdr-server.md` には `Shell aliases` 節の 1 文を足しただけ |
| 273、278、430「`codex-seat-permissions.md` に記録する」 | 当たっていない | Trigger は codex の director や cross-review の live の実行。recipe の編集ではない |
| 135、146、155(`doctor` と `.codex/config.toml` の行) | 当たっていない | Trigger は `doctor_codex_writable_root.go`、そのテスト、`.codex/config.toml`。どれも触っていない |
| 468「`checkManifestVersion`」 | 当たっていない | `doctor.go` の行だが、Trigger は `checkManifestVersion` の変更。この branch の `doctor.go` の変更は Check 8b の comment だけ(`git diff` で読んだ) |
| 161、188、190、251、264、340、357、379、382、386、403、478、510 | 当たっていない | `/org` skill の別の節、role prompt、`Send`、director skill などの変更が Trigger。前提の節の shell alias の項は名指しされていない |
| 232、252、374、476 | 当たっていない | 語は `doctor` や `alias`(ssh の alias など)で、Trigger は別の変更 |
| 288「Thirteen Go comments … `plans/active/2026-10-15-org-director-start`」 | 当たっていない | Trigger は「あの 12 ファイルの次の編集」。12 ファイルは `org_director*.go`、`director*.go`、`prompts*.go`、`spawn*.go`、`skill_frontmatter_test.go`(`git grep -l` で確かめた)で、この branch の Go の 3 ファイルは 1 つも入っていない。この branch の comment は `docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md` を引く(self-review の L6)。`/pr` が計画を archive に動かすまでは、そのパスにファイルがない |
| 99、108、109、158、181、196、202、240、254、256、355、359、367、392、481、482、484、485、494、496、498、501、506、509、515、516 | 当たっていない | Trigger は `check-sync.sh` の比較、`gate.go`、`watch.go`、`verbs.go`、`Stop`、reseat、codex の hook trust、Node toolchain の Check など、この branch が触っていないファイルと関数 |
| 123、134、311、326、354 | 当たっていない | Trigger が全部 `~~` か RESOLVED で、払う物が残っていない |

## 他に確かめた文書

| 文書 | 結果 |
|---|---|
| `AGENTS.md`(`internal/cli/` と `internal/org/` の行) | 変えない。`internal/cli/` の行は subcommand の一覧で、doctor の個々の Check を書いていない。「shell alias」「Shell aliases」「checkShellAliases」は 0 件。計画の Verify plan が AGENTS.md に答えを求めた箇所はない |
| `README.md` | 変えない。`ralph doctor` の行(136 行目)は Codex hook trust と Merge guarantee の Check を詳しく書くが、Shell aliases の Check は前から書いていない。この branch が変えたのはその Check の severity の 1 つの判定だけ。`HERDR_PANE_ID` を書くのは 289 行目の approve の段落だけで、alias の話ではない |
| `CLAUDE.md`、`.claude/rules/`、`.codex/` | 変えない。「Shell aliases」の語は 0 件。`HERDR_PANE_ID` を書く rule もない |
| `docs/quality/`、`docs/architecture/repo-map.md` | 変えない。新しいファイル、script、recipe、gate はない。`alias` の語は `docs/quality/`、`docs/architecture/`、`AGENTS.md`、`CLAUDE.md` に 0 件で、`quality-gates.md` の 86 行目は `HERDR_PANE_ID` を approve の仕掛けとして書く |
| `docs/specs/` | 変えない。`HERDR_PANE_ID` を書く spec は 5 つ(org-runtime、multi-org-director、herdr-autostart-panes、seat-resume、org-reseat)あるが、どれも別の話(herdr server の環境、自分の pane を飛ばす規則、approve)。alias の Check の spec はない |
| `docs/recipes/codex-seat-permissions.md`(2 写し) | コードと合う。severity の括弧(claude の `--model` だけ、herdr なし、`HERDR_PANE_ID` の test なら info、それ以外は warn)と、doctor が読む形の文は、verify が `4fc29299` のコードと 1 つずつ照らして Yes にした。そのあとコードは変わっていない(`git log 4fc29299..HEAD` が 2 つの Go ファイルに見つけるのはテストの 1 行だけ)。文面の「checked on herdr 0.7.5」は計画の仮定と同じ |
| `docs/recipes/isolated-herdr-server.md`(2 写し) | 合う。1 文で書き方を示し、読む形は codex の recipe と org skill に任せている |
| org skill の 4 写し | 合う。`.claude/skills/org/SKILL.md` の差分を読み、recipe と同じ内容を日本語で言っていることを確かめた。`check-skill-sync.sh` と `check-sync.sh` が通る |
| `internal/cli/doctor.go` の Check 8b の comment | 合う。「An alias a HERDR_PANE_ID test keeps out of herdr's panes is informational whatever its flags」は `checkShellAliases` の判定と同じ向き |
| `internal/upgrade/notes/` | 変えない。release ノートは `/release` で書く決まりで、計画の Rollout にも候補の文だけが書いてある |

## 直さなかったこと

- verify の INFO(severity の括弧の要約と、値を読み切れていない def の端のケース): 引用符が閉じない alias は、`HERDR_PANE_ID` の test があっても warn に残る。recipe の括弧は「test なら info」と要約しているので、閉じない引用符の rc では少しずれる。壊れた rc の話で、Detail に not-fully-parsed の節が出る。recipe 2 つと skill の 4 写し、合計 6 か所の文を増やすほどではないとして、直していない
- 計画の Progress の「PR created」は `[ ]` のまま。`/pr` が付ける。Progress に sync-docs の行は足していない(handoff が AC の箱以外は計画の本文を変えない決まりだったため)
- Go の comment が引く計画のパス(`doctor_shell_alias.go` の 5 か所とテストの 1 か所)は archive のまま。verify の指摘どおり、`/pr` のあとに正しいパスになる
- test の G2 から G4(herdr の pane に `HERDR_PANE_ID` が入ることの実機の確認、印を誤って付ける 3 つの形の打ち直し、build した doctor の観察)は、どれも PR を止めない。G3 の 3 つの形は 137 行目が持っている。G2 は計画の Assumptions と recipe の「checked on herdr 0.7.5」が持つ。新しい行は足していない

## 確認

- `./scripts/check-sync.sh`: `PASS: all files in sync.`(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6)
- `./scripts/check-skill-sync.sh`: `14 skill(s) in lock-step`
- 計画の digest: `2c2d8e5932b3`(変わっていない)。`git diff --numstat` は計画に 7/7 で、AC の 7 行だけが動いた
- tech-debt の編集: HEAD と突き合わせ、変わった行は 138、373、417 だけ。どの行も 5 つのセル(`(?<!\\) \| ` で割る)、バッククォートと `~~` は列ごとに偶数、新しい `file:line` の参照なし
- tech-debt の計画の参照: `verify.local.sh` の `check_tech_debt_plan_refs` と同じ検索(`grep -o -E 'docs/plans/(active|archive)/…'`)を手で回し、足りない参照は 0 件。この計画の active のパスは 5 件(136、137、138、373、417)で、archive のパスは 0 件
- テストと `run-static-verify.sh` は回していない。変えたのは docs だけで、Go のコードと recipe と skill の本文は verify と test の run の HEAD から変わっていない

## Insight event

`./scripts/insights-append.sh --slug doctor-alias-herdr-guard --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-11-19-doctor-alias-herdr-guard.jsonl` に足した。この branch は前の phase の行(self_review、verify の 2 行、test)も同じファイルに commit しているので、同じ扱いにした。
