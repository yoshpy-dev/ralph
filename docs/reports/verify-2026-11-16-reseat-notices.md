# Verify report: reseat-notices

- Date: 2026-11-16(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-16-reseat-notices.md`。承認の digest `2aa9b9bde56c` は `./scripts/plan-visual.sh digest` で計算し直した値と一致する
- Verifier: verifier サブエージェント(`/verify`)。前の verifier は静的検査を終えたあと、report を書く前に API のネットワークエラー(ENOTFOUND)で止まった。この run はその静的検査のログを使い、spec 適合と文書のずれを調べ直した
- Scope: `git diff origin/main...HEAD`(base `30e122a1`、HEAD `8c6b60e0`、20 ファイル、+831 −105)。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-11-16-reseat-notices.log`(`.gitignore` の `docs/evidence/*.log` に当たるので commit されない)

## 静的検査のログを使い回せるか

- ログの先頭の時刻は 2026-11-16T09:50:13Z(18:50:13 JST)。HEAD `8c6b60e0` の commit は 18:48:52 JST なので、ログはそのあとに取られた。branch の secret scan の行も `scanned 30e122a1..8c6b60e0` と HEAD を名指す。worktree は clean で、HEAD のあとの commit もない
- `scripts/check-sync.sh`(DRIFTED 0、ROOT_ONLY 0、`PASS: all files in sync.`)と `scripts/check-skill-sync.sh`(`14 skill(s) in lock-step`)がログに入っている。末尾は `All verifiers passed.` と `rc=0`
- ログが HEAD と合い、2 つの検査も含むので、打ち直していない

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 本物の reseat が codex の座席ごとに `org reseat` の行を、どの座席も止める前に出す。claude の座席と、計画の前に断られた reseat は出さない | Met(コードとテストのソース) | `internal/cli/org_reseat.go:226` が `OnPlan` に `printOrgReseatCodexHookTrustNotices` を渡す。`:270-277` は `plan.Start` の座席ごとに `printCodexHookTrustNotice(w, seat.Driver, "org reseat", seat.SeatID, plan.Binding.Worktree)` を呼ぶ。claude の座席は `codex_hook_trust_notice.go:38` の driver の判定で何も出さない。`internal/org/reseat.go:1154-1156` は、印を書いたあと(`:1151`)、`reseatRun` を作る前(`:1158`)、`stopAll` の前(`:1170`)に 1 度だけ呼ぶ。flag(`:1138`)、ロック(`:1141`)、`PlanReseat`(`:1147`)、印の書き込み(`:1151`)で断られると、その前に return する。テスト: `TestCodexHookTrustNotice_ReseatPrintsOneLineForACodexSeat`(`codex_hook_trust_notice_test.go:448`。stderr は implementer の 1 行だけで、行を書いた時点の manifest が reseat の前と同じ)、`TestCodexHookTrustNotice_ReseatOfClaudeSeatsPrintsNothing`(`:513`)、`TestCodexHookTrustNotice_ReseatRefusedBeforeItsPlanPrintsNothing`(`:533`。flag と `PlanReseat` の断り、本物と dry run)、`TestReseat_OnPlanRunsOnceBeforeTheFirstStop`(`reseat_test.go:1473`。記録の順は `plan` のあとに `stop:…`、呼ばれた時点で印に reseat の id があり、台帳に新しい event がない)、`TestReseat_OnPlanIsNotCalledOnARefusal`(`:1561`。ロックの断りを含む 5 件) |
| AC2: `--dry-run` も同じ行を出す | Met(コードとテストのソース) | `reseat.go:1126-1130` が、`PlanReseat` が通ったときだけ `OnPlan` を呼ぶ。`TestCodexHookTrustNotice_ReseatDryRunPrintsTheSameLine`(`codex_hook_trust_notice_test.go:485`。stderr は同じ 1 行で、manifest は変わらず、herdr は呼ばれない)、`TestReseat_OnPlanRunsOnceBeforeTheFirstStop` の `a dry run`(1 度だけ呼ばれ、`DryRun` の計画を受け、止める処理は 0 回) |
| AC3: `reseat --help` が 2 つの段落を持ち、3 つの comment が reseat を挙げる | Met | `orgReseatHelp` は `orgScaffoldVersionNoticeHelp`、空行、`orgCodexHookTrustNoticeHelp` の順で終わる(`org_reseat.go:133-135`)。固定するテストは `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts`(`org_reseat_test.go:910` の `HasSuffix`)、`TestCodexHookTrustHelp_EachVerbDescribesTheNote`(`codex_hook_trust_notice_test.go:718`)、`TestScaffoldVersionHelp_EachVerbDescribesTheNote`(`scaffold_version_notice_cli_test.go:292`)の 3 つ。comment は `printCodexHookTrustNotice`(`codex_hook_trust_notice.go:20-25`)、`orgCodexHookTrustNoticeHelp`(`:107-111`)、`orgScaffoldVersionNoticeHelp`(`scaffold_version_notice.go:107-114`)の 3 つで、ほかに `codexHookTrustAgent`(`codex_hook_trust_notice.go:94-99`)も reseat を挙げる。段落の行は最長 75 文字で、テストの上限 76 に収まる |
| AC4: `/director` skill の 4 写しの「人に向けた知らせの行」が、2 つの行の動詞に reseat を挙げ、「4 つの動詞」になり、reseat の hook の行も「そのまま人に伝える」に入る。テストが断片を確かめる | Met | `.claude/skills/director/SKILL.md:1617`、`:1622`、`:1648-1656`。root と `templates/base` の写しは `cmp` で一致し、4 写しの差分の追加行と削除行は同じ SHA-1 になる。`directorNoticeRules`(`internal/org/prompts_notice_test.go:96` から)が変わった 4 つの断片と足した 2 つの断片を持ち、`TestDirectorSkill_SaysWhatToDoWithTheHumanNotes`(`:176`)が `directorSkillCopies` の 4 写しを読む。S4 で足した手順 2・4 の文は `TestDirectorSkill_ReseatPointsAtTheHumanNotes`(`:250`)が固定する |
| AC5: `internal/org/prompts/director.md` が 2 つの行を出す動詞に reseat を含め、テストが確かめる | Met | `director.md:62`(「`ralph status`、機能ごとの org の start か reseat が」)。`directorPromptNoticeMarker`(`prompts_notice_test.go:272`)と `directorPromptNoticePhrases`(`:285`)を `TestRenderDirectorPrompt_PointsAtTheHumanNotesSubsection`(`:301`)が読む |
| AC6: 仕様の #318 の項と `.codex/README.md` の 2 写しが reseat を挙げ、`check-sync.sh` と `check-skill-sync.sh` が通る | Met | `docs/specs/2026-10-07-org-multi-org-director.md:164` の末尾に日付つきの 1 文(「2026-11-16 に足したこと: `ralph org reseat` も 2 つの行を出す…」)。項の中の古い動詞の一覧は、2026-11-13 の決定の記録として残してある。`.codex/README.md:206-209` は `templates/base/.codex/README.md` と `cmp` で一致する。2 つの検査はログで PASS |
| AC7: `OnPlan` が nil の reseat は今までどおり動く | Partially verified(コードだけ) | 2 つの呼び出しは `p.OnPlan != nil` のときだけ呼ぶ(`reseat.go:1128`、`:1154`)。既存の reseat のテストは `st.reseatParams` を使い、`OnPlan` を渡さない。テストが通ることは `/test` で確かめる |

### self-review の N1 と N2(S6、`299fb172`)

- N1 は直っている。dry run の段落の最後に「Like a real reseat, it prints the codex hook trust note (see the last paragraph) on stderr for each codex seat it would start.」が入った(`org_reseat.go:99-100`)。`org_reseat_test.go:885-888` がこの文を固定し、「the last paragraph」が hook の段落を指すことは `:910` の `HasSuffix` が保証する
- N2 は直っている。`printCodexHookTrustNotice` の doc の 3 つめの段落は「printCodexHookTrustNotice reads codex's user config …」で始まる(`codex_hook_trust_notice.go:27`)
- S6 はこの 2 点と、それに合わせたテストの文と doc の直しだけで、ほかの経路は変わっていない。S6 の見直しで新しい指摘は出なかった

### 共有の段落 `orgCodexHookTrustNoticeHelp` は 4 つの動詞に合うか

S5 で「an agent the command starts」「that agent」「one note on stderr for each such agent」の言い方になった。呼び出し元ごとに照らした結果は次のとおり。

| 動詞 | 呼び出し | 行の数 | 読む dir | 段落と合うか |
| --- | --- | --- | --- | --- |
| `ralph org spawn` | `internal/cli/org.go:967` | 座席 1 つに 1 行 | 座席の cwd | 合う |
| `ralph org start <task>` | `org.go:1202` | leader の 1 行 | cwd | 合う |
| `ralph org start --plan` | `org.go:1332` | leader の 1 行 | main worktree のルート(worktree はまだ無い) | 合う。leader が動く worktree のルートは main のチェックアウトのルートに読み替わるので、括弧の「the main checkout's, for a linked worktree」と同じ hooks.json を読む |
| `ralph org director start` | `org_director_headless.go:366` | director の 1 行 | cwd | 合う(行の名前は `codex director`) |
| `ralph org reseat` | `org_reseat.go:275` | `plan.Start` の codex の座席ごとに 1 行 | 機能の worktree | 合う。dry run は何も立てないが、dry run の段落(N1 の直し)が行を出すことを言う |

座席 1 つの動詞では「for each such agent」は 1 行と読める。偽になる動詞は見つからなかった。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope、golang の pack) | rc=0 | ログを使い回した(上の節)。gofmt は `ok`、`go vet ./...` と staticcheck は出力なし、golangci-lint は `0 issues.`。ログに `Skipping` の行はない |
| `scripts/check-sync.sh` | PASS | DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `scripts/check-skill-sync.sh` | ok | 14 skill が lock-step |
| `scripts/check-pipeline-sync.sh`、`scripts/check-template-purity.sh`、tech-debt の計画の参照 | OK | |
| `secret-scan-branch.sh` | clean | `30e122a1..8c6b60e0` |
| `./scripts/plan-visual.sh digest <plan>` | `2aa9b9bde56c` | 計画の `- Approved:` の行と一致 |
| `git diff --check origin/main...HEAD` | 出力なし | |
| 差分の U+FFFD | 0 件 | |
| `git merge-tree --write-tree --name-only origin/main HEAD` | 衝突なし | 手元の `origin/main`(`30e122a1`)との試し merge。fetch はしていない |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/director` skill の 4 写し | Yes | AC4 |
| `internal/org/prompts/director.md` | Yes | AC5 |
| `.codex/README.md` の 2 写し | Yes | AC6 |
| 仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の #318 の項 | Yes | AC6 |
| `README.md:133`(`ralph org` の行) | No | V1 |
| `docs/tech-debt/README.md` | No | V2 から V4 |
| `internal/upgrade/notes/8.2.0.md:11` | Yes | 雛形の版の行の動詞に reseat がある。hook の行は動詞を並べない。次の版の注記は計画の Non-goals |
| `AGENTS.md`、`CLAUDE.md` | 変更不要 | どちらも、どの動詞についても 2 つの行を書いていない(`hook trust`、`scaffold version`、`older than this CLI` は 0 件)。計画の Verify plan の問い(internal/org の行に reseat の知らせを書くか)の答えは「書かない」 |
| `/org` skill の動詞の表 | 変更不要 | spawn と start の行も 2 つの行を挙げていない |

検索は、`docs/reports`、`docs/plans/archive`、`docs/evidence`、`docs/insights` を除いた tracked file に対して行った。`director start` の前後 260 文字と 120 文字に hook、note、scaffold、雛形、知らせ、trust のどれかがあって reseat がない箇所を python で拾い、`spawn` か `start` と hook・雛形の語が近い行も `git grep` で拾った。

### 指摘

| ID | Severity | 箇所 | 内容 | 直し方 |
| --- | --- | --- | --- | --- |
| V1 | MEDIUM | `README.md:133` | README のコマンド表の `ralph org …` の行が、hook の行を出す動詞を「`spawn`, `start` (both forms), and `director start` print one note on stderr when the agent they launch runs on codex …」と書き、reseat を挙げない。同じ行の雛形の版の文は `reseat` を挙げているので、片方だけが古い。利用者向けの一覧で、この PR が直そうとした種類のずれにあたる。self-review(「依頼された 5 点の確認」の表の 5 行目)は「`README.md:133` は動詞の総覧で、hook の行を並べていない」と書いたが、この行は hook の行の動詞を並べている。`README.md` は root だけのファイルで、`check-sync.sh` は見ない | `/sync-docs` で `/pr` の前に直す。例: 「`spawn`, `start` (both forms), `reseat`, and `director start` print one note on stderr when an agent they launch runs on codex … (`reseat` prints one for each codex seat it starts, before it stops any seat, with `--dry-run` too)」 |
| V2 | LOW | `docs/tech-debt/README.md` の行「Fourteen small findings of docs/spec-org-reseat (#312c …)」(今は 513 行目)の (i)(L14) | (i) は「`ralph org reseat` prints no codex hook-trust note」で、Trigger の (i) は「print one line per codex seat of `plan.Start`, as the other verbs do; paying it also means adding `reseat` to … `.codex/README.md` … and … `templates/base/.codex/README.md`」。この PR はちょうどそれをした(`72cc7d72`、`0d7dfe1f`)。それでも行は開いたままに読める | (i) を払った、と commit を挙げて書く |
| V3 | LOW | tech-debt の 4 行の Trigger | この PR は次の Trigger を満たしたが、払っておらず、持ち越しの note もない(この PR が触った tech-debt の行は 478 行目だけ)。(a) 513 行目の (f)、(j)、(m)、(n): `orgReseatHelp`(`72cc7d72` の hook の段落、`299fb172` の dry run の文)と `TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts` を変えた。help は今も、止まった座席を見るのに `ralph org status --org-id <id>` を読めとは言わない((f))。(b) 514 行目(`reseat.go` と `reseat_test.go` の大きさ、move-only の分割): `wc -l` で `reseat.go` は 1,553 から 1,574 行、`reseat_test.go` は 1,868 から 2,004 行、`internal/cli/org_reseat_test.go` は 927 から 954 行(origin/main と HEAD)。(c) 515 行目の (h): S4(`f18738bd`)が `/director` の「座席を立て直す」を変えた。足した `TestDirectorSkill_ReseatPointsAtTheHumanNotes` が固定するのは新しい 2 文で、(h) が求める AC-16 の文ではない。(i)、(j): `runOrgReseat`(`72cc7d72`)と `scaffold_version_notice_cli_test.go`(`0d7dfe1f`)を変えた。(i) は一部だけ払い(V4)、(j)(`TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` に reseat の行がない)は払っていない。(d) 480 行目: Trigger は「The next change to `codex_hook_trust.go`, `doctor_codex_hook_trust.go`, or `codex_hook_trust_notice.go`: add (a) first …」。`72cc7d72`、`64746c2b`、`299fb172` が `codex_hook_trust_notice.go`(comment と help の定数)を変えたが、(a) と (b) は払っていない | `/sync-docs` が、各行に「2026-11-16: fix/reseat-notices hit … but did not pay it」の形の note を足す。数字は左の値 |
| V4 | LOW | tech-debt の 3 行の今の状態を言う文 | (a) 515 行目の (i)「`TestScaffoldVersionHelp_EachVerbDescribesTheNote` lists the other four verbs, and the help paragraph of `reseat` is pinned only by `TestOrgReseatHelp_…`」: `0d7dfe1f` からこのテストは reseat を挙げ、`72cc7d72` で `TestCodexHookTrustHelp_EachVerbDescribesTheNote` も reseat を見る。残るのは、reseat を走らせて雛形の版の行と `model_pool` の行を確かめるテストがない、の半分。(b) 479 行目の (a)「warns forever, in doctor and in the launch note of `ralph org spawn`, `start`, and `director start`」: reseat の行も同じ関数なので同じく出続ける。(c) 480 行目の (c)「The launch note of `org spawn`, `start`, and `director start` was not run against a real herdr」: reseat の行も stub の herdr でしか走らせていない | 3 か所の動詞の一覧と文を今に合わせる |

### INFO

- I1: tech-debt 468 行目の (b) は `orgScaffoldVersionNoticeHelp` を「the paragraph of three `--help` texts」と書く。reseat の help は #312c(`ecb84fa8`)からこの段落を持つので 4 つで、この PR の前から古い。Trigger の (b)「The next change to `orgScaffoldVersionNoticeHelp`」については、この PR は定数の doc comment だけを変え、値は変えていない。満たしていないと読んだ
- I2: `internal/cli/scaffold_version_notice_cli_test.go:13-19` のファイルの先頭の comment は、雛形の版の行を出す動詞を org start、director attach、director start、status と書き、reseat を挙げない。このファイルには reseat を走らせるテストがない(V4 (a))ので、ファイルの中身の説明としては合っている。動詞の一覧と読まれうる
- I3: 計画の Risks の「関数は計画を読むだけにし」は、self-review の L1 の直しで `OnPlan` の契約(`reseat.go:74-87`)が「ralph の動詞も遅い処理も待ちも持たない。計画を読み、書き換えない」になり、CLI 側は小さなファイルを読んで stderr に書く、と書き分けた。計画の文は承認の digest の内側にあるので、そのままにしてよい

## Observational checks

画面も実機の動きもない変更なので、観察の証拠は取っていない。行が止める前に出ることは、org 側のテストの順の記録(`plan` のあとに `stop:…`)と、CLI 側のテストが行を書いた時点で取った manifest のバイト列で確かめる形になっている(どちらもソースを読んだだけ)。

## Coverage gaps

- テストは走らせていない。AC1 から AC6 の動きと AC7 は `/test` で確かめる
- 1 回の reseat で codex の座席が 2 つ以上あるテストがない(計画の Edge cases)。座席ごとの繰り返しは `printOrgReseatCodexHookTrustNotices` の `for` にあるが、CLI の fixture は `--driver` を座席 1 つにしか渡せないので、codex の座席を 2 つ持つ台帳が要る
- codex の leader を reseat するテスト(行は `codex seat leader`)がない
- 印の書き込みが失敗したときに `OnPlan` を呼ばない経路のテストがない(計画の Progress に記録済み)
- reseat を走らせて雛形の版の行を確かめる CLI のテストがない。この PR の前からの穴で、515 行目の (i)
- 本物の herdr と codex での通しの実行はしていない
- 計画の Progress にある変異の結果は、ここでは打ち直していない

足すと効く小さな検査: `printCodexHookTrustNotice` と `printScaffoldVersionNotice` に渡す動詞の一覧を 1 つの変数にまとめ、2 つの help のテストがそれを回し、同じ変数で `README.md` の `ralph org` の行、`.codex/README.md`、`/director` skill の小節がどの動詞も挙げることを確かめる。V1 はこの形の検査で見つかった。

## Insight event

`./scripts/insights-append.sh --slug reseat-notices --flow standard --phase verify --verdict pass --cycle auto --source skill --events-dir "$TMPDIR/…"` で行を作業用のディレクトリに書いた。依頼は report だけの commit で、この run の self-review も event を書いていないので、`docs/insights/events/` には足していない。行は次のとおり(`verdict` に partial の値がないので `pass`)。

```
{"schema":1,"ts":"2026-11-16T14:46:44Z","slug":"reseat-notices","flow":"standard","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"source":"skill"}
```

## Verdict

- Verified: 静的検査(`run-static-verify.sh` rc=0。check-sync と check-skill-sync を含み、HEAD `8c6b60e0` で取ったログ)、計画の digest、`git diff --check`、U+FFFD 0 件、`origin/main` との試し merge。AC1 から AC6 の本体とテストの対応(file:line とテスト名)。skill の 4 写しと `.codex/README.md` の 2 写しの一致。self-review の N1 と N2 が S6(`299fb172`)で直ったこと。共有の段落 `orgCodexHookTrustNoticeHelp` が spawn、start(2 つの形)、director start、reseat のどれにも合うこと
- Partially verified: AC7(nil の扱いはコードで確かめた。既存のテストが通るかは `/test`)
- Not verified: テストの実行、codex の座席が 2 つ以上の reseat、codex の leader の reseat、本物の herdr と codex での動き
- Verdict: **partial pass**。AC を満たさないものはなく、静的検査はすべて通った。文書のずれが残る。V1(MEDIUM、`README.md:133` の hook の行の動詞に reseat がない)は `/sync-docs` で `/pr` の前に直す。V2 から V4(LOW、tech-debt の行が払われたこと、満たした Trigger の持ち越し、今の状態を言う文)も `/sync-docs` で直す。I1 から I3 は記録だけ
