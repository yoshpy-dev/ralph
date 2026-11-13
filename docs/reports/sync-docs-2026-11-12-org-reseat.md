# sync-docs report: org-reseat

- Date: 2026-11-13(JST。UTC では 2026-11-12)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(承認の digest `3058f6f66b5b`)。仕様は `docs/specs/2026-11-12-org-reseat.md`。issue #312 の 3 本目(#312c)の 1 本目
- Branch: `docs/spec-org-reseat`。run 開始時の HEAD は `9a295a84`(origin/main の `14531858`、#308 を取り込んだ merge)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)。以降の節はこの 1 回目のもの。2 回目(`cycle` 2)の sync-docs は末尾の「Cycle 2」の節にある。2 回目は P7 を変えたので、承認の digest は `3058f6f66b5b` から `f7e78c0dd96c` に変わった。3 回目(cap を 3 に上げた回。`cycle-count.json` は `cycle` 2、`cap` 3)は末尾の「Cycle 3」の節にあり、P6 と Risks を変えたので digest は `95a056c00a9a` に変わった
- 先行 report: self-review(3 回目の実行、`f8579b7c`。blocking 0、LOW L2〜L6・L10)、verify(`af927216`。pass、drift V1〜V8、INFO I1〜I6)、test(`69908490`。pass、変異 10 個のうち 5 個が生き残り、Test gaps 8 件)
- 見た範囲: この task の差分は `git diff 14531858...HEAD`(24 ファイル)。`14531858` が持ち込んだ #308 の編集(org skill の send と status の行、director skill の「答える順」、AGENTS.md の `send_fallback.go` と `status_live.go` の節、README、tech-debt、`docs/specs/2026-08-01-org-runtime.md` の 2026-11-12 改訂)は残した。send と status を含む 5 つの行の長さが HEAD と同じことを、`git show HEAD:` との比較で確かめた
- 制約: 計画の本文は変えていない。変えたのは Progress の箱 3 つと、末尾の sub-bullet 4 行で、`./scripts/plan-visual.sh digest` は編集の前後とも `3058f6f66b5b`。Go のコードは、使い方の文字列 2 つ(`--seat` と `--config`)とコメント 1 か所だけを変えた(挙動は変わらない。下の「V7」「V6」)。テストは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report と編集の commit の前の数): `git diff origin/main...HEAD` は 24 ファイル、+6,036/-31(計画、仕様、report、evidence、insights を含む)。docs 以外でテストと testdata を除くと 15 ファイルで +2,571/-31(skill と quality-gates の写し 8 つと Go の 7 ファイルを含む)。walkthrough は書いていない

## Summary

verify の V1〜V8 をすべて直した(V8 は tech-debt の持ち越しの注で、verify が挙げた 8 行に、掃引で見つけた 4 行を足した 12 行)。INFO I2 は org skill の移し方の 1 に 1 文足した。仕様の印は 1 本目の 26 項目を `[x]` にし、AC-12 は実機の限りを書いたうえで `[x]` にした。2 つの Open questions には実機の記録の答えを足した。tech-debt の末尾に新しい 3 行(小さな指摘、ファイルの大きさ、テストの穴)を足した。

verify が挙げていない drift も 3 つ直した。quality-gates の「Cross-org watch gate」と「Herdr server gate」の行(2 つの写し)が列挙する動詞に reseat が無かった。README の「`start` (both forms), `director attach`, and `director start` print one note」の文と、README の見張りと herdr server を起動する動詞の文も同じだった。reseat は `printScaffoldVersionNotice` を呼ぶ 5 つ目の動詞である。

## Changes made

| File | Change |
|------|--------|
| `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md`(と `.agents/skills/` の写し、`templates/base/.claude/skills/` と `templates/base/.agents/skills/` の写し) | V1、V6、I2。下の各節。`./scripts/sync-skills.sh` のあと、2 つの template の写しを `cp` した。skill ごとに 4 つの写しの sha256 が一致する(org `e5af028c0e8b`、director `d4e690f61a72`) |
| `AGENTS.md` | V2。`internal/org/` の行に reseat の節を 1 つ |
| `README.md` | V3 と V6。動詞の一覧に `reseat`、動詞の行に 1 文、note の文に `reseat`、見張りと herdr server の文に本当の reseat、Org runtime の節に reseat の段落 |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | 「Cross-org watch gate」と「Herdr server gate」の行に本当の reseat。2 つの写しに同じ 4 か所 |
| `docs/specs/2026-11-12-org-reseat.md` | V4。26 項目を `[x]`、Open questions の答え、Status に 1 段落 |
| `docs/specs/2026-08-01-org-runtime.md` | V5。「2026-11-12 改訂(docs/spec-org-reseat)」の項(a)〜(d)を、#308 の同じ日の項の次、運用ノートの前に足した |
| `docs/tech-debt/README.md` | V8。12 行の Trigger の末尾に note、Related の末尾に計画の path。末尾に新しい 3 行。下の「tech-debt」 |
| `internal/cli/org_reseat.go`、`internal/cli/org.go`、`internal/cli/org_legacy_ledger.go` | V7 と V6。使い方の文字列 2 つ、コメント 1 か所。挙動は変えていない |
| `docs/plans/active/2026-11-12-org-reseat.md` | Progress の `Review artifact created`・`Verification artifact created`・`Test artifact created` を `[x]` に。末尾に self-review、verify、test、sync-docs の sub-bullet。AC の箱 19 個は `[ ]` のまま(前の PR と同じく、`/pr` か lead が付ける) |
| `docs/reports/sync-docs-2026-11-12-org-reseat.md` | この report |
| `docs/insights/events/2026-11-12-org-reseat.jsonl` | `sync_docs` の行(cycle 1、pass)を 1 行。slug は日付のない `org-reseat` で、新しい file はできていない(`git status --short` で確かめた) |

commit は 1 つ(編集、report、insights をまとめた)。SHA は返答に書く。

## verify が挙げた drift

### V1: skill の写しが、切れた reseat・独自の役割・signal を言っていない

コード(`reseatTargets`、`reseatLifeSeats` の `cutOff`、`reseatCustomRoleErr`、`orgReseatHoldSignals`、`orgReseatHelp`)と突き合わせて書いた。

- org skill の動詞の表の `reseat` の行(1,473 字から 1,932 字): 既定の対象に、切れた reseat が止めた座席(止めずに立てる)を足した。独自の役割の座席の断りと、本当の reseat が SIGINT・SIGTERM・SIGHUP・SIGPIPE で終わらず、終了コードが reseat の結果のままであることと、切れた reseat を同じ `--seat` で打ち直して終えることも足した
- org skill の「座席の立て直し」の小節: 1 の対象に切れた reseat の座席を足し、断る条件を「active な座席も切れた reseat の座席もなく」にした。小節に 3 つの段落(「途中で切れた reseat」「signal」「独自の役割の座席」)を足した。`--seat` を付けた reseat の代わりに付けない reseat を打つと active な座席も止めて立て直すことは、help の文のとおりに書いた
- director skill の「座席を立て直す」: 2 に独自の役割の座席の断り(その座席を止めて `--prompt` つきで立て直す打ち方が出る。人が頼んだら打つ)、4 に signal と終了コード、7 に process が止められずに終わったときの打ち直しを足した
- 手を入れていない所: org skill の send と status の行、director skill の「答える順」

signal の段落は、`note: <signal> signal received:` の行が SIGINT・SIGTERM・SIGHUP のときだけ出て、SIGPIPE では出ないことまで、コードのとおりに書いた。`--dry-run` は signal を受け止めない。

L10(出力を失った reseat のあとの案内)は skill に書いていない。tech-debt の小さな指摘の (f) にあり、最小の直しは help の 1 文である。

### V2: AGENTS.md の `internal/org/` の行

`ralph org resume` や `status --live` の節の次、pane lock の節の前に 1 つ足した。「the reseat of a feature org's seats (`reseat.go`: `ralph org reseat` stops and starts them again without a disband, after the checks and the watch and herdr server gates that `start` runs, under `reseat.lock`, recording `reseat=<id>` on each stop and start and one `reseated` event)」。節は 1 つで、ほかの節は変えていない。`check-sync.sh` は通る。

### V3: README

- 動詞の行(`ralph org spawn/start/.../resume`)の一覧に `/reseat`。その行に、`resume` の文の次に reseat の文を 1 つ
- 同じ行の「`start` (both forms), `director attach`, and `director start` print one note on stderr」に `reseat`(下の「verify が挙げていない drift」)
- Org runtime の節の `[org].api_wait_minutes` の段落の次に、reseat の段落を足した。`resume` の段落の並びにある。中身は、何をするか、アカウントを移す手順、対象、断る場合、検査の内容、停止と起動の順、`reseat.lock`、記録、signal、DECISION の行、`--dry-run`、経過を渡さないこと

### V4: 仕様の印

- `[x]` にした 26 項目: FR-1〜FR-8、FR-5a、FR-11、非機能要件 4 つ、AC-1〜AC-12。判定は verify の `## Spec compliance` による
- `[ ]` のまま: FR-9、FR-10、AC-13、AC-14(2 本目)
- AC-12 は `[x]` にして、行の中に限りを書いた。実機の記録は 2 席(leader と reviewer)で AC-1 に合格し、AC-8 に合格した。implementer の座席と承認の検査は実機で見ておらず(テストだけ)、binary は切れた reseat の座席と SIGPIPE の直しより前のものである。verify の判定は「一部」だった。計画の S5 が 2 席に絞った決定を記録し、AC の文(分離した herdr で AC-1 と AC-8 を 1 回ずつ確かめる)はその範囲で行われているので、限りつきで付けた。外すなら、その行の `[x]` を `[ ]` に戻す
- Open questions の 2 つに答えを足した(実機の記録 `docs/evidence/org-reseat-live-2026-11-12.md`): (1) 全座席が Leave しても agmsg の team と履歴は消えない(`team_id` は同じ、名簿と 5 件のメッセージが残った)。leader が最初に Join し直す今の順で足りる。(2) herdr 0.7.5 が戻した pane には端末もプロセスもなく(`pane read` と `process-info` は `pane_not_found`)、新しい環境が入る pane はなかった。reseat の停止は Ctrl-C が失敗してもその pane を閉じて立て直した
- Status に、2026-11-13 の 1 段落(1 本目を実装して印を付けた、2 本目は未着手、Open questions に答えが出た)を足した

### V5: org runtime の仕様

「### 2026-11-12 改訂(docs/spec-org-reseat)」を、同じ日の #308 の項の次に足した。(a) 動詞と対象、(b) 検査、停止、起動の順、(c) 記録(`reseat=<id>`、`reseated`、`reseat.lock`、signal、切れた reseat)、(d) アカウントの移し方と、経過を leader に渡さないこと。末尾の「本文の記述は書き換えない。FR-1 の `ralph org` 動詞の列挙は、上の (a) を足して読む。」は、2026-11-07 改訂の書き方と同じ。本文には印を付けていない(2026-11-07 改訂と同じ)。#308 の項は変えていない。

### V6: 動詞の列挙

reseat が実際にそう動くかを、コードで確かめてから足した。

- 台帳の古い形の拒否: `runOrgReseat` は dry run も本当の reseat も `resolveOrgLedger(cmd, ..., orgLedgerMutating)` を呼ぶ。`org_legacy_ledger.go` のコメントの一覧(`stop` の次)と、org skill の「前提」の一覧(`reseat`(`--dry-run` を含む))に足した
- 見張り: 本当の reseat は `wireOrgStartWatchGate`、dry run は `wireOrgWatchGate`(読むだけ)。org skill の見張りの段落に、`director start` の文の次に足した
- herdr server: 本当の reseat は `wireOrgHerdrServerGate`、dry run は読むだけの probe。org skill の「前提」の最初の項の一覧(`ralph org start`・`spawn`・`reseat`・`director start`)に足し、`reseat --dry-run` が起動しないことの文も足した。director skill の「herdr のサーバーを自分で起動しない」の項の一覧に足した
- quality-gates の 2 つの行と README の 2 つの文は、verify が挙げていない。同じ列挙なので同じ根拠で足した(2 つの写しの `check-sync.sh` は通る)

### V7: 使い方の文字列

- `--seat`: 「(default: every active seat of the org, and the seats a reseat that was cut off stopped; to finish a cut-off reseat, run it again with the same --seat values)」
- 永続フラグ `--config`: 末尾に「; reseat reads what start --plan reads: those limits, [org.role_models] and driver_pool, [org.approval].mode, and the four keys of [reviewer]」。`TestOrgDirectorHelp_DescribesTheRegistration` が `--config` の使い方の 3 つの断片を固定していて、うち 2 つが「(see ralph org start --help), escalate and director attach」と「spawn and start read ... start --plan also」をつなげて見ているので、start --plan の隣には足さず、固定された断片の外(末尾)に足した

どちらも使い方の文字列だけで、挙動は変わらない。`go build ./internal/cli/` と `gofmt -l` は通り、`go test -count=1 -run 'TestOrgDirectorHelp_DescribesTheRegistration|TestOrgReseatHelp|TestScaffoldVersionHelp|TestRetiredOrgStateDirRefusal|TestOrgHelp|TestLegacyLedger' ./internal/cli/` は ok(2.6 秒)。ビルドした `ralph org reseat --help` に、足した 1 文が出る。

### V8: tech-debt の Trigger

下の「tech-debt」。

## INFO

- I2(herdr 0.7.5 が `session.json` から中身の無い pane を戻す): org skill の移し方の 1 に 1 文足した。止める前に保存した `session.json` から workspace と pane の記録が戻ることがあり(端末もプロセスも戻らない)、その pane は Ctrl-C が `pane_not_found` で失敗しても閉じられて `stopped` になる(Details は `ctrl_c=failed: ... pane=closed`)。recipe は変えていない
- I1(実機の binary が古い): test-gaps の新しい行の (g) に入れた
- I3、I4: 仕様の FR-4 と FR-2 とずれていないと verify が判定しているので、何もしていない
- I5(コメントが `docs/plans/active/` を指す): 小さな指摘の新しい行の (e) に入れた
- I6(`reseat.go` などが 800 行を超えた): ファイルの大きさの新しい行に入れた

## tech-debt

### Trigger に note を足した 12 行

行は先頭の語で探した(番号は HEAD で verify と同じ)。note は Trigger セルの末尾に `2026-11-13: docs/spec-org-reseat ... did not pay it: ... Carried over ...` の形で足し、Related の末尾に `docs/plans/active/2026-11-12-org-reseat.md` を足した。`archive/` とは書いていない(`archive-plan.sh` が `/pr` で書き換える)。

| 行(先頭の語) | 当たった所 | 払ったか |
|----|----|----|
| 361「`ralph org stop` ... do not take a seat's spawn lock」 | `Stop`(`StopParams.Note`、`noteDetails`)と、`Reseat` から `Stop` を呼ぶこと | 払っていない。計画の Risks が名指しする行で、`reseat.lock` と直前の読み直しが窓を狭める |
| 179「Code-shape findings of org-stop-all」 | `Stop` に足した 1 行 | 払っていない。`result.Err = err` と 5 つの pane の確かめは前のまま |
| 183「Test gaps left by org-stop-all」(a) | 同じ 1 行 | 払っていない。足した `Stop` のテストは `Reseat` の側 |
| 189「Code-shape findings of org-limits-reserve」 | `checkCapacityAndStart` と `dryRunSpawn` の `spawn_started` の `Details` | 払っていない。F-2 と F-6 は開いたまま(この行には「Carried over」の形がないので、「stay open」で閉じた) |
| 180「Two documentation findings of org-stop-all」(a) | 動詞の表の新しい `reseat` の行(1,473 字、sync-docs のあと 1,932 字) | 払っていない。`stop` と `disband` の cell は変わっていない |
| 201「Test gaps left by org-feature-worktree」(e)(f) | `StartFeature`(検査を 2 つの関数に切り出し、`Note`)、`feature.go` 792 → 848 行 | 払っていない。(e) の未実行の枝と (f) の防御の枝は測り直していない |
| 391「Two windows the merge cleanup of #263 leaves open」(b) | `StartFeature` の検査の移動(`regateSpawnErr` が `checkLedger` の中へ) | 払っていない。ロックの外のまま |
| 396「`resolvedOrClean` in `internal/org/statedir.go`」 | `feature.go` の呼び出し 1 つが `startFeatureInput` へ移った | 払っていない。名前も呼び出しの数も同じ |
| 457「Seven mutants of `ralph org resume`」 | 自分の編集: `org_legacy_ledger.go` のコメント | 払っていない。コードは変えていない |
| 464「The `/director` skill ... do not say what a director does when it sees the scaffold version note」 | director skill の 4 つの写しを編集(新しい節と sync-docs の編集)。reseat は note を出す 5 つ目の動詞 | 払っていない。skill は scaffold と `ralph upgrade` に触れないまま |
| 118「`templates/base/docs/quality/quality-gates.md` の「Must pass in CI」」 | 自分の編集: quality-gates の 2 行 | 払っていない。「Must pass in CI」と verify chain は変えていない |
| 192「Test gaps left by the org-limits-reserve cycle 2 fix」(c)(d) | 自分の編集: `--config` の説明と、`orgLedgerAccess` のコメント | 払っていない。新しい clause は固定されていない |

verify が挙げたのは 361、179、183、189、180、201、391、396 の 8 行。457、464、118、192 は、この sync-docs の自分の編集か、verify が見なかった写し(director skill の編集と reseat が note を出すこと)で当たったので足した。

### 当たりとみなさなかった候補

Trigger の実際の文を読んで外した。

- 133: Trigger の本文が `~~` で閉じている
- 148(`resolveOrgConfig`、`withMainWorktreeOrgLimits`、`MainWorktreeRoot`、`ResolveOrgStateDir` の変更): どれも変えていない。`--config` の説明は、この行の Trigger が指す文ではない
- 177、195、199、244〜252、259、381 ほか「director が…するとき」の行: Trigger が headless の director の自動化か、その行の関数の変更で、この diff は触れていない
- 195 と 201 の `runOrgStartPlan` / `newOrgStartCmd` の部分: `runOrgReseat` は新しい関数で、start の `RunE` を編集していない
- 200(`StartFeature` の `p.StateDir` の扱い): 動かした呼び出しは `p.RepoRoot`
- 203、204: `checkFeatureWorktreeReuse` と `checkEnsuredFeatureWorktree` は呼ぶだけで変えていない。`TestStartFeature_*` も編集していない
- 262(`/director` skill の受信箱の節): 編集は「座席を立て直す」と「してはいけないこと」で、受信箱の節ではない
- 269: `director_skill_sync_test.go` を編集していない
- 288(`regateSpawnErr` と `Spawn` の idempotent の枝): 変えていない
- 364(spawn saga の失敗の経路と補償の code): `Note` は経路でも補償でもない
- 365(`Send`、+300): `verbs.go` は 2597 → 2617、`spawn.go` は 3101 → 3109(`wc -l` を 14531858 と HEAD で)。+300 に届かず、`Send` は変えていない
- 377(split-plan の例と leader の手順): 触れていない
- 380(RC6、送って確かめる手順): 新しい節の 6 は「答える順」の 2 を指すだけで、手順自体は編集していない
- 401(`RALPH_ORG_STATE_DIR` の拒否の条件): 編集した「古い台帳」の項は別の項
- 427(`ralph org status` の JSON): 変えていない
- 432(`HerdrServerGate` を呼ぶ場所): `Spawn`、`StartFeature`、`checkDirectorStart` の呼び出しの位置は変わっていない。`PlanReseat` は新しく呼ぶ場所で、ほかの検査のあとに gate を回す(start の順の逆)
- 466、467(`printScaffoldVersionNotice` の root の引数、`orgScaffoldVersionNoticeHelp`): 既存の呼び出しも定数も変えていない。reseat は 5 つ目の呼び出しで、464 の note と、新しい test-gaps の行の (i) に書いた

### 新しい 3 行(末尾)

- 小さな指摘(7 つ): self-review の L2、L3(b)、L4、L5、L6(と verify の I5)、L10 と、SIGPIPE の直しの残り。lettered (a)〜(g)
- ファイルの大きさ: `reseat.go` 1,230 行、`reseat_test.go` 1,637 行、`feature.go` 792 → 848 行(`wc -l` を `772da303` で。`feature.go` は `14531858` でも)。`verbs.go` と `spawn.go` は +20 と +8 で、それぞれの行の +300 の下
- test の穴: test report の Test gaps 1〜8 を (a)〜(h) に(M9、M68、M6、C1、M4 を含む。番号は test report の cycle 1 のもの)。sync-docs の掃引で見つけた 2 つを (i)(j) に足した。(i) `runOrgReseat` は scaffold version の note と `model_pool` の note を出すが、どのテストも reseat でそれを読まない。(j) `TestOrgLegacyLedger_MutatingVerbsRefusedWhileLegacySeatsActive` の表に `reseat` の行がない。live check の穴(implementer と承認の検査はテストだけ、binary が S6・S7 より前)は (g)

形は `tdcheck.py`(scratch)で確かめた。変わった行は 12 と、新しい 3 行だけで、どれも 5 列(エスケープしていない `|` で割ると 7 つ)、backtick と `~~` の数は列ごとに偶数、`file:line` の参照は足していない。12 行は、古い cell の文字列を先頭に残したままの挿入である。

## 確かめたこと

| コマンド | 結果 |
|---|---|
| `./scripts/sync-skills.sh`、2 つの template の写しへの `cp`、`shasum -a 256` | skill ごとに 4 つの写しが一致 |
| `./scripts/check-sync.sh` | PASS: all files in sync(KNOWN_DIFF 6、TEMPLATE_ONLY 10 は前と同じ) |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `./scripts/run-static-verify.sh`(changed scope) | exit 0、39 秒。`gofmt: ok`、golangci-lint `0 issues.`、branch secret scan(`14531858..9a295a84`)clean |
| `go test -count=1 -run 'Skill' ./internal/org/ ./internal/cli/ ./internal/scaffold/` | ok |
| 上の V7 の `go test -run ...  ./internal/cli/` | ok |
| `./scripts/plan-visual.sh digest` | 編集の前後とも `3058f6f66b5b` |
| tech-debt の `docs/plans/active/...` の参照 | `docs/plans/active/2026-11-12-org-reseat.md` が存在する |
| `docs/tech-debt/README.md` の `tdcheck.py` | 上のとおり、問題 0 |

走らせていないもの: `./scripts/run-test.sh` の全体(Go のコードは使い方の文字列 2 つとコメント 1 か所の変更で、挙動を持つ行は変えていない。help を固定するテストと skill のテストは走らせた)。

## 人に残すもの

- 編集の commit が Go のコードを含む(使い方の文字列 2 つとコメント 1 か所)。`post-implementation-pipeline.md` の決まりでは、コードの修正は `/self-review` からのやり直しの対象になりうる。この run は、ハンドオフが「コードのコメントと使い方の文字列だけ」と範囲を決めたので、やり直しの要否を lead に任せる
- AC-12 の `[x]` は、限りを書いたうえで付けた。外す判断は lead か人
- 計画の AC の箱 19 個は `[ ]` のまま(`/pr` か lead が付ける)
- 2 本目(FR-9、FR-10、AC-13、AC-14)の印は `[ ]` のまま
- L10 の案内(help の 1 文か、`--seat` なしの reseat が失敗した座席を出す形)は直していない。tech-debt の小さな指摘の (f) に Trigger つきで入れた
- release ノートに書く項目(計画の Rollout): `ralph org reseat`、別のアカウントへ移す手順(herdr server を新しい環境で立て直してから reseat)

## Cycle 2

2 回目の pipeline run の sync-docs。上の節(1 回目)は書き換えていない。

- Date: 2026-11-13(JST)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`。承認の digest は 1 回目の `3058f6f66b5b` から `f7e78c0dd96c` に変わった(下の「計画の digest」)。仕様は `docs/specs/2026-11-12-org-reseat.md`
- Branch: `docs/spec-org-reseat`。run 開始時の HEAD は `2d37fa32`(2 回目の test report)。origin/main の `4220ea7c` から分かれている
- Pipeline run: 2 回目(`cycle-count.json` は `{"cycle":2}`、`cap` なし。上限 `max_pipeline_cycles` 2 に届いた回)
- 先行 report: cross-review の triage(`2bc28bdc`。ACTION_REQUIRED 3 件)、その直し(`adc3e205`、`e9461b79`、`0468d96f`)、origin/main の #310 と #308 の取り込み(`8fbd7118`、`9a295a84`)、self-review(`f754188f`、`c50d79d6`。blocking 0、LOW 10 件)、verify(`80f58f86`。pass、drift V9〜V16、INFO I7〜I10)、test(`2d37fa32`。pass、変異 11 本をすべて殺した)
- 見た範囲: 2 回目の verify が挙げた drift V9〜V16 と INFO I7〜I10、self-review の持ち越し L13〜L16、直しの 3 commit の差分(`git diff 2bc28bdc..HEAD` の reseat の Go ファイル)
- 制約: Go のコードは変えていない(V14 の help の文と L15・L16 は Go の文字列で、直すと pipeline を `/self-review` からやり直す。上限に届いているので tech-debt に入れた)。テストは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report と編集の commit の前の数): `git diff --shortstat origin/main...HEAD` は 34 ファイル、+6,984/-81(計画、仕様、report、evidence、insights を含む)。`docs/` と `_test.go` を除くと 19 ファイルで +2,889/-67(skill の写し 8 つと quality-gates の写し 1 つ、AGENTS.md、README、Go の 8 ファイルを含む)。walkthrough は書いていない

### Summary

verify の V9〜V13 と V16 を直し、V14 と V15 は tech-debt に入れた。INFO I8 は L14 の項に書いた。残りの INFO(I7、I9、I10)は直さない(下の表)。コードを読み直して文を書いた。`--seat` が受け付ける座席の定義(`startable()`: `spawned`、今 active、または `reseat=<id>` の note を持つ)は `reseatTargets` と `reseatNoActiveSeatErr` が共有し、`finish` が作る DECISION は、leader が対象に無ければ、途中で起動が失敗したあとも立った座席について出る。この 2 点と、打ち直しの command に渡された `--state-dir` と絶対パスの `--config` が付くことが、文書に無かった。

### 計画の digest

`3058f6f66b5b` から `f7e78c0dd96c` に変わった。変えたのは計画の P7(`--seat` が受け付ける座席の文)で、これは digest の範囲に入る。Progress の項(下の V16)だけなら変わらない。HEAD の計画に Progress の項だけを足した写しで `./scripts/plan-visual.sh digest` を回すと `3058f6f66b5b` のままで、P7 も変えた今の計画で `f7e78c0dd96c` になった。承認の `- Approved:` 行の digest は書き換えていない。承認の記録し直しは orchestrator が行う。

### Changes made

| File | Change |
|------|--------|
| `docs/specs/2026-11-12-org-reseat.md` | V9、V10。FR-2: `--seat` は active な座席と、reseat が止めたか立てようとした座席(Details が `reseat=<id>` で終わる)も受け付け、一度も `spawned` にならず active でもなく reseat も触れていない座席(手で止めた座席など)は断る。D2: 「今 active な座席がない」の断りは `--seat` が受け付ける座席だけを挙げる、の段落を足した。D5: 途中の失敗のあとの DECISION と、打ち直しが自分の DECISION を出すこと、leader が対象で後の座席が失敗したら DECISION は出ないこと。FR-6: 打ち直しは渡された `--state-dir` と絶対パスの `--config` を引き継ぐ。FR-7: DECISION の条件を書き直し、`--config` は reseat・stop・spawn・start の command に付くが DECISION の `send`、`read`、`Spawn` の error の中の `stop` には付かないことを足した。Status に 2 回目の段落。チェック欄は変えていない |
| `docs/plans/active/2026-11-12-org-reseat.md` | V9、V16。P7 に、受け付ける座席の範囲と、断りが挙げる座席の範囲を足した。Progress に 2 回目の項 5 つ(cross-review の直し、self-review、verify、test、sync-docs)。AC の箱は `[ ]` のまま |
| `README.md` | V11。reseat の段落の `--seat` の文に、reseat が止めたか立てようとした座席も受け付け、手で止めて reseat が触れていない座席は断ること。打ち直しの command に `--state-dir` と絶対パスの `--config` が付くこと。DECISION の文は、途中の失敗のあとも含めて今のコードと合うので直していない |
| `.claude/skills/org/SKILL.md`(と `.agents/skills/org/`、`templates/base/.claude/skills/org/`、`templates/base/.agents/skills/org/` の写し) | V12。表の `reseat` の行: `--seat` が受け付ける座席、断りが `--seat` が受け付ける座席だけを挙げること、`--config` と `--state-dir` の引き継ぎ、途中の失敗のあとの DECISION。「座席の立て直し」の手順 1、途中で失敗したときの段落(引き継ぎと、付かない 3 つ)、DECISION の段落(失敗のあとの打ち直しと、leader が対象の場合)。`send` と `status` の行(#308)は触れていない。表の `reseat` の行は 1,932 字から 2,286 字になった |
| `.claude/skills/director/SKILL.md`(と `.agents/skills/director/`、`templates/base/.claude/skills/director/`、`templates/base/.agents/skills/director/` の写し) | V13。「座席を立て直す」の 7(途中で失敗したとき): 打ち直しの command が `--state-dir` と `--config` を引き継ぐこと。leader が対象でなく先に立った座席があれば、失敗の出力にも `read` と DECISION の `send` が出て、6 と同じに leader の pane を読んでから送ること。打ち直しが自分の DECISION を出すこと。leader が対象で後の座席の起動が失敗したら DECISION は出ないこと |
| `docs/tech-debt/README.md` | V15 と L13〜L16。5 行を編集(485、486、487、180、464)。下の「tech-debt」 |
| `docs/reports/sync-docs-2026-11-12-org-reseat.md` | 先頭に 2 回目への指し示しを 1 文、この節を足した |
| `docs/insights/events/2026-11-13-org-reseat.jsonl` | `sync_docs` の event(cycle 2、pass)を 1 行足した |

skill の写しは `./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、2 つの template の写しを `cp` した。写しの shasum(先頭 12 桁): org の 4 つが `41bbb524348c`、director の 4 つが `edd4447ddc84`(1 回目は `e5af028c0e8b` と `d4e690f61a72`)。

### verify の drift と INFO の扱い

| 項目 | 扱い |
|------|------|
| V9 仕様 FR-2、D2 の注、計画 P7 | 直した。上の表。D2 の注は、`--seat` が受け付ける座席の定義と、断りが挙げる座席(止めた座席があればそれと打ち方、なければ受け付ける座席の最新の state、どれもなければ leader の `start --plan`)を言う |
| V10 仕様 D5、FR-6、FR-7 | 直した。D5 は途中の失敗のあとの DECISION、FR-6 は `--state-dir` と `--config` の引き継ぎ、FR-7 は DECISION の条件と `--config` が付く command の範囲 |
| V11 README の reseat の段落 | 直した。DECISION の文は直していない(verify が「直さなくてよい」とした。leader が対象のとき、leader が立てばその文に当たらず、立たなければ何も立たない) |
| V12 org skill の表と節 | 直した。手順 1 の「一度も立っていない座席は断る」が、reseat が触った座席について偽だった点も直した |
| V13 director skill の 7 | 直した。6 の送り方(leader の pane を読んでから、「答える順」の 2 と同じに送る)に 7 から指すだけで、送る手順は書き写していない |
| V14 help と `ReseatParams.HintConfigPath` の doc | 直していない。help は Go の文字列で、直すと pipeline を `/self-review` からやり直す。上限に届いているので、tech-debt の (j) に入れた(self-review の L15 と同じ所) |
| V15 tech-debt の reseat の 3 行 | 直した。下の「tech-debt」 |
| V16 計画の Progress | 直した。上の表 |
| I7 `docs/specs/2026-08-01-org-runtime.md` の改訂 (b)(c) | 直していない。DECISION を「leader 以外の座席だけを立て直したとき」と書き、`--config` に触れないが、要約で偽にはならない(verify も同じ判定) |
| I8 `.codex/README.md` と template の写し | 直していない。今の文書は偽ではない(reseat は #310 の note を出さない)。L14 を払うときに、その文の動詞の並びに reseat を足す、と tech-debt の (i) に書いた。2 つの写しは `scripts/check-sync.sh` が同一に保つ |
| I9 AC-12 の実機の binary(`dd204370`)が 2 回目の直しより前 | 直していない。仕様の AC-12 は、実機の限りを書いたうえで `[x]` のまま。実機で見ていない直し(`touched` の座席の打ち直し、失敗のあとの DECISION、`--config` の引き継ぎ)は tech-debt の (g) の note に書いた |
| I10 L6 と L13 の型の注釈 | tech-debt の (e) と (h) に入っている。L13 は (h)、Trigger は (e) と一緒に払う |

### tech-debt

編集した行は 5 つだけで、新しい行は足していない。形は `check_td.py`(scratch)で確かめた。変わった行は 180、464、485、486、487 の 5 行で、どれも 5 列(エスケープしていない `|` で割る)、backtick と `~~` の数は列ごとに偶数、`file:line` の参照は足していない。変更は挿入と、数と名前の置き換えだけで、古い cell の文字列は先頭に残してある。

| 行(先頭の語) | 編集 |
|----|----|
| 485「Eleven small findings of docs/spec-org-reseat」(題は Seven から。(a)〜(k) は 11 件) | self-review の L13〜L16 を (h)〜(k) に足した。(h) L13: テストの doc comment が cross-review の triage を番号で引く 4 か所。(i) L14: reseat が #310 の hook の note を出さない。(j) L15: help と `HintConfigPath` の doc の 2 つの文の誤り(`read` と `send` は ralph.toml を読む。`Spawn` の error の中の `stop` は `--config` を繰り返さない)。(k) L16: 断りが custom role の座席も並べ、次の run で `reseatCustomRoleErr` に断られる。L11 は 0468d96f で直ったので入れず、L12 は L15 になった。(f) の「断りは止めた座席を挙げる」を、今の `--seat` が受け付ける座席に直し、失敗のあとの DECISION が stdout にしか無いことを足した。Trigger に (h)〜(k) の払い方を足し、末尾に (b)(d)(f) の持ち越しの note(`reseatNoActiveSeatErr`、`PlanReseat` の引数、`reseatLifeSeats` と help の変更で当たった。直しが cross-review の 3 件に絞られ、上限に届いていたので払っていない)。Related に self-review と verify の 2 回目の指摘の番号。L14 の払い方には、`.codex/README.md` と template の写しの文に reseat を足すことを書いた |
| 486 ファイルの大きさ | `reseat.go` 1,230 → 1,307 行、`reseat_test.go` 1,637 → 1,855 行、`internal/cli/org_reseat_test.go` 749 → 852 行(800 行を超えた)。`wc -l` は、Go を最後に変えた commit `0468d96f` で数えた。`feature.go` は 848 のまま、`internal/cli/org_reseat.go` は 510 行で下。Trigger の末尾に持ち越しの note |
| 487 テストの穴 | 末尾に持ち越しの note: (c)(d) は `printOrgReseatResult` の doc comment だけの変更(adc3e205)で当たり、新しい CLI のテストは `left stopped:` を見るが、`stop failed:`・`left running:`・`left as it is:`・`not started:` は今も見ない。(i)(j) は `runOrgReseat` の変更(e9461b79)で当たり、reseat の行も scaffold の note のテストも無い。(e) は I/O の戻りが 1 つ増えた(`runOrgReseat` の `filepath.Abs`)。(g) は 3 つの直しも実機で見ていない。(h) はこの pass が skill を編集して当たり、skill の文を help と比べるテストは新しいテストなので払っていない |
| 180 | 引用している `reseat` の行の長さに、2 回目の sync-docs のあとの 2,286 字を足した。Trigger は (a) のまま、払っていない |
| 464 | `/director` の「座席を立て直す」を 2 回目にも編集した(7)ので、当たったが払っていない、と 1 文足した。skill は scaffold version の note に今も触れない |

当たりとみなさなかった候補(Trigger の文を読んだ。1 回目の候補で外したものは繰り返さない):

- 380(RC6、送って確かめる手順): director skill の 7 は、6 の送り方を指すだけで、送って確かめる手順自体(「答える順」の 2)は編集していない
- 148(`resolveOrgConfig`、`withMainWorktreeOrgLimits`、`MainWorktreeRoot`、`ResolveOrgStateDir`): `runOrgReseat` の絶対パスの `--config` は、どれも変えていない
- 192(c)(`ralph org status` の `Long` と `ralph org` の `--config` の説明): 2 回目の直しは reseat の help の `--config` の段落で、`ralph org` の `--config` の説明ではない
- 195(`newOrgStartCmd` の `RunE`)、201: 直しの 3 commit は start の `RunE` を編集していない。#310 の `printCodexHookTrustNotice` の呼び出しは origin/main から入ったもの
- 218(`InboxCommand` と `CommandWithStateDir`): `reseatVerbCommand` は別の関数で、この 2 つは編集していない
- 270(`ralph status` に `--config` が無い): 触れていない

### 確かめたこと

| コマンド | 結果 |
|---|---|
| `./scripts/sync-skills.sh`、2 つの template の写しへの `cp`、`shasum` | skill ごとに 4 つの写しが一致(org `41bbb524348c`、director `edd4447ddc84`) |
| `./scripts/check-sync.sh` | PASS: all files in sync(KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `go test -count=1 -run 'Skill\|Director' ./internal/org/` | ok(70 秒)。reseat の文を固定するテストは無いので、これは send と director の他の節を固定するテストが壊れていないことの確認 |
| `./scripts/run-static-verify.sh`(changed scope) | exit 0、20 秒。`gofmt: ok`、golangci-lint `0 issues.`、tech-debt README の plan 参照 OK、template の meta-repo 参照 PASS、branch secret scan(`4220ea7c..2d37fa32`)clean |
| `./scripts/plan-visual.sh digest` | 編集の前 `3058f6f66b5b`、後 `f7e78c0dd96c`。Progress だけの変更では `3058f6f66b5b` のまま(上の「計画の digest」) |
| tech-debt の `check_td.py` | 上のとおり、問題 0 |
| 行の幅(skill の変更行、全角を 2) | 最大 78。既存の行の最大 77 とほぼ同じ |

走らせていないもの: `./scripts/run-test.sh` の全体(Go のコードは変えていない)。`ralph org reseat --help` の実出力は読んでいない(help は変えていない)。

### 人に残すもの

- 計画の digest が `3058f6f66b5b` から `f7e78c0dd96c` に変わった。`- Approved:` の行は書き換えていない。承認の記録し直しは orchestrator が行う
- V14 と L15・L16 は Go の文字列(help、doc comment、断りの文)で、直していない。直すなら上限(2)を上げ、`/self-review` からやり直す。内容は tech-debt の 485 行の (j)(k)
- L14(reseat が #310 の note を出さない)は設計が要る。tech-debt の (i) に払い方がある
- 計画の AC の箱は `[ ]` のまま(`/pr` か orchestrator が付ける)。仕様の AC-12 の `[x]` は、実機の binary が 2 回目の直しより前であること(I9)を含めて、人が持つ
- origin/main は `bb416c48`(#325、upgrade-behavior-notes)まで 18 commit 進んでいる。この branch は `4220ea7c` から分かれていて、`git merge-tree` での確認では `docs/tech-debt/README.md` の 1 か所だけが衝突する(どちらも行を足している)。`AGENTS.md` と `README.md` は自動で入る。`/pr` の前に取り込む必要がある。この確認はこの commit の前の HEAD で行った。tech-debt の 5 行の編集を含めた衝突は、取り込むときに見る
- 2 回目の test report の注: `run-test.sh` の Go の段で org と cli のパッケージが 40 分の上限に 1 度当たり(load average 7〜16 のとき)、同じ上限での再実行で通った。reseat のテストではないが、`/pr` の本文に書く材料
- release ノートに書く項目は 1 回目のまま。2 回目の直し(座席が `--seat` で名指せる範囲、失敗のあとの DECISION、`--config` の引き継ぎ)も、reseat の項に含まれる

## Cycle 3

3 回目の pipeline run の sync-docs。上の節(1 回目と 2 回目)は書き換えていない。

- Date: 2026-11-13(JST)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`。承認の digest は 2 回目の `f7e78c0dd96c` から `95a056c00a9a` に変わった(下の「計画の digest」)。仕様は `docs/specs/2026-11-12-org-reseat.md`
- Branch: `docs/spec-org-reseat`。run 開始時の HEAD は `c856fc27`(3 回目の test report)。origin/main の `e902705c`(#327 と #325)を `8460bf64` で取り込み済み
- Pipeline run: 3 回目(`cycle-count.json` は `cycle` 2、`cap` 3。2 回目の cross-review の ACTION_REQUIRED 1 件を受けて cap を 3 に上げた回)
- 先行 report: cross-review の triage(`bcadf0b4`。ACTION_REQUIRED 1 件)、その直し(`def8d2a3`、`df8f674e`、`b142944d`)、origin/main の取り込み(`8460bf64`)、self-review(`6c24bfa2`、`975e9f20`、`1326092b`。blocking 0、LOW の open は L2、L3(b)、L4、L5、L6、L10、L13〜L16、L19〜L21)、verify(`bab7b33a`、`2194a9e2`。pass、drift V17、V18、V20〜V24、INFO I11〜I14)、test(`c856fc27`。pass、変異 6 本をすべて殺した、Test gaps 10〜13)
- 見た範囲: 3 回目の verify が挙げた drift V17、V18、V20〜V24 と INFO I14、self-review の L19〜L21、test の Test gaps 10〜13、直しの 3 commit の差分(`git diff bcadf0b4..HEAD` の reseat の Go ファイルと skill)
- 制約: Go のコードは変えていない(help は b142944d で直っている)。テストは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。この report と編集の commit の前の数): `git diff --shortstat origin/main...HEAD` は 35 ファイル、+7,802/-81(計画、仕様、report、evidence、insights を含む)。`docs/` と `_test.go` を除くと 19 ファイルで +3,029/-67。walkthrough は書いていない

### Summary

verify の V24(MEDIUM)、V17、V18、V20〜V23 と INFO I14 を直した。cycle 3 の直しは、reseat が座席を止めてから立てるまでのあいだにほかのコマンドが同じ座席を別の役割・driver・モデルで立てた場合(`reseatIdempotentMismatchErr`)の動きを足した。その座席を `start_failed` にして、あとの座席を立てずに終わる。`--driver`・`--model` が渡されていなければ、座席を `ReseatResult.Changed` にして打ち直しから外す。渡されていれば、打ち直しがその座席を止めて渡された設定で立てる。外した座席のほかに立てていない座席がなければ、打ち直しは出ない。help は b142944d でこれを言うようになっていた。仕様、README、org skill、director skill がそれに追いつくよう直した。`ReseatResult.Changed` の doc とコードを読んで文を書き、書いた文を `startAll`、`reseatRetryCommand`、`orgReseatLeftLine` に当てた(外した座席は `start failed:` の行には出るが、`left as it is:` の行は出ない)。

### 計画の digest

`f7e78c0dd96c` から `95a056c00a9a` に変わった。変えたのは計画の P6 と Risks の 2 つ目で、どちらも digest の範囲に入る。Progress に足した 5 項は範囲の外で、足したあとも `95a056c00a9a` のまま(`./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-org-reseat.md` を P6 と Risks の編集の直後と、Progress の編集のあとに回した)。承認の `- Approved:` 行の digest は書き換えていない。承認の記録し直しは orchestrator が行う。

### Changes made

| File | Change |
|------|--------|
| `.claude/skills/director/SKILL.md`(と `.agents/skills/director/`、`templates/base/.claude/skills/director/`、`templates/base/.agents/skills/director/` の写し) | V24: 受信箱の処理の「件の処理」で、`usage_limited` の件の「座席を立て直す動詞は issue #312 で扱っていて、この版にはない」を、別のアカウントや driver で立て直すと人が決めたら「座席を立て直す」に従って `ralph org reseat` を打つ(人が頼んだときだけ)、立て直したあとにその件を resolve する、に替えた。`usage_limited` の件のほかの文は #312a のまま。V22: 「座席を立て直す」の 7 に、stderr に `started by another command` が出る場合を足した(error が動いている設定と、reseat が立てるつもりだった設定を言う。2 つを人に伝え、後者に移すかを人に決めてもらう。`--driver`・`--model` を渡した reseat なら、打ち直しがその座席を止めて渡した設定で立てる。渡していなければ座席は打ち直しから外れて今の設定のまま動き、外した座席のほかに立てていない座席がなければ打ち直しは出ない) |
| `.claude/skills/org/SKILL.md`(と 3 つの写し) | V21: 表の `reseat` の行に、立てるときの比べ(返った座席の役割・driver・モデルが計画と違えば、動かしたまま `start failed:` にしてあとの座席を立てない。同じ設定なら立てた座席と数える)、`--driver`・`--model` なしでは座席を打ち直しから外し `left as it is:` の行も出さないこと、外した座席のほかに座席が無ければ打ち直しも出ないことを足した。「座席の立て直し」の手順 4 に立てるときの比べを、手順 5 に `reseated` を書かない場合(最初に止める座席の止める直前の読み直しで終わった reseat)を足した。途中で失敗したときの段落は、打ち直しのコマンドを「立てていない座席が残るとき」に直し、新しい段落「ほかのコマンドが別の設定で立てた座席」を足した |
| `README.md` | V20: reseat の段落に、立てるときの比べと外す座席(error が両方の設定を言う。`--driver`・`--model` なしでは打ち直しから外し、残る座席が無ければ打ち直しは出ない。あれば打ち直しがその座席を止めて立てる)を足し、`reseated` の文に「止めるか立てようとした reseat が 1 件書く。最初に止める座席の止める直前の読み直しで終わった reseat は書かない」の条件を足した |
| `docs/specs/2026-11-12-org-reseat.md` | V17。FR-6 に、立てるときの比べ(cycle 3 の cross-review の指摘で足した)と、打ち直しに入る座席の規則(止めたまま残した座席はいつも入る。ほかのコマンドが立てて動いている座席は `--driver`・`--model` が渡されていなければ入れない)を足した。FR-8 の `reseated` に書く条件を足した。非機能要件 1 の「打ち直し方を必ず出す」を、同じ規則に直した。D3 に重なりの扱い(止める前の検査では防げないので起動のときに見る)を 1 文足した。Status に 3 回目の段落を足した。印は変えていない |
| `docs/plans/active/2026-11-12-org-reseat.md` | V18。P6 に起動のときの比べを足し(違えば `start_failed`、`--driver`・`--model` なしは打ち直しから外す)、Risks の 2 つ目に Spawn が断らずに idempotent で返す場合を足した。Progress に 3 回目の項を 5 つ足した(triage と cap と 3 つの直しと main の取り込み、self-review、verify、test、sync-docs)。AC の箱は `[ ]` のまま |
| `docs/specs/2026-08-01-org-runtime.md` | I14: 2026-11-13 改訂の (d) に、動詞は上の 2026-11-12 改訂の `ralph org reseat` が受け持つ、の指し先を足した |
| `docs/tech-debt/README.md` | V23 と L19〜L21 と Test gaps 10〜13。6 行を編集(180、262、464、494、495、496)。下の「tech-debt」 |
| `docs/reports/sync-docs-2026-11-12-org-reseat.md` | 先頭に 3 回目への指し示しを足し、この節を足した |
| `docs/insights/events/2026-11-13-org-reseat.jsonl` | `sync_docs` の event(`cycle` 2、pass)を 1 行足した |

skill の写しは `./scripts/sync-skills.sh` で `.agents/skills/` を作り直し、2 つの template の写しを `cp` した。写しの shasum(先頭 12 桁): org の 4 つが `04366d42dbbd`、director の 4 つが `5b2c1c81ccdc`(2 回目は `41bbb524348c` と `edd4447ddc84`、merge 8460bf64 のあとは `c75fc6d937fd` と `e5dcf7466095`)。

### verify の drift と INFO の扱い

| 項目 | 扱い |
|------|------|
| V24 director skill の「この版にはない」 | 直した。上の表。人が頼んだときだけ打つ、という「座席を立て直す」の 1 を指す。仕様の「依存」の #312a の行(「director skill の案内で打ち方をつなぐ」)は、これで満たされる |
| V17 仕様 | 直した。(a)(b)(c) は FR-6 と非機能要件 1、(d) は Status、(e) は FR-8。D3 の 1 文は verify が挙げていないが、(a) の「D3 が言わない」に当たるので足した |
| V18 計画 | 直した。P6 と Risks は digest の範囲なので、digest が変わった(上の「計画の digest」)。Progress は範囲の外 |
| V20 README | 直した。help の b142944d の文を手本にして、起動のときの比べと外す座席を言い、`reseated` の条件を足した |
| V21 org skill | 直した。表の行、手順 4・5、途中で失敗したときの段落。外した座席には `left as it is:` の行が出ないこと(L19)も書いた。止める側の `Changed` が唯一の対象のときも打ち直しが出ないことは、1 回目からの偽だったので、同じ段落で直した |
| V22 director skill | 直した。上の表 |
| V23 tech-debt | 直した。下の「tech-debt」 |
| I11 同じ設定で立てた座席の `spawned` に `reseat=<id>` が無い | 直していない。verify が害を見つけていないとした。前からの動きで、`TestReseat_ASeatAnotherCommandStartedOnTheSameSettings` が固定する |
| I12 `--driver`・`--model` を渡した reseat の `start_failed` でも `reseated` の `changed=` は頼んだ変更を書く | 直していない。前からの動きで、座席の結果は `seats=` で分かる |
| I13 triage の理由の欄が `--driver`・`--model` なしの reseat で偽になった | 直していない。triage は次の `/cross-review` が書き直す記録 |
| I14 org runtime の仕様の指し先 | 直した。上の表 |

### tech-debt

編集した行は 6 つだけで、新しい行は足していない。形は `tdcheck3.py`(scratch)で確かめた。変わった行は 180、262、464、494、495、496 の 6 行で、どれも 5 列(エスケープしていない `|` で割ると 7 つ)、backtick と `~~` の数は列ごとに偶数、`file:line` の参照は足していない(`git diff -U0` の追加行を正規表現で探した)。変更は挿入と、数と名前の置き換えだけで、古い cell の文字列は先頭に残してある。

| 行(先頭の語) | 編集 |
|----|----|
| 494「Thirteen small findings of docs/spec-org-reseat」(題は Eleven から) | self-review の L19、L20、L21 を (l)(m) に足した((l) L19: `--driver`・`--model` なしで外した座席の理由が stdout に出ない。(m) L20 と L21: help の短い行と、`reseated` の文)。見出しの件数は、self-review の指定どおり (m) を L20 と L21 でまとめて 13。(e) の「five comments」は 6 か所(`reseat_start_test.go` が増えた)、(h) の「four places」は 5 か所(同じファイルの「cross-review cycle 2, finding 1」)に直し、Trigger の「four comments」「five of (e)」も合わせた。(h) の Trigger の「cite the commit」に def8d2a3 を足した。Why deferred の「after the pipeline reached its cap (`max_pipeline_cycles` 2)」を落とした(cap を 3 に上げたので偽になる)。Impact と Trigger に (l)(m) を足した。Trigger の末尾に持ち越しの注: b142944d が `orgReseatHelp` とそのテストを変えて (f) と (j) の Trigger に当たり、払っていない。def8d2a3 と df8f674e が変えた `startAll` などは、この行のほかの Trigger が名指すものではない。Related に L19〜L21 の置き場と、2 回目の verify の `git show 80f58f86:...` を足した |
| 495 ファイルの大きさ | Trigger の末尾に持ち越しの注: def8d2a3 と df8f674e が `reseat.go`(1,307 → 1,361 行)で Trigger に当たり、b142944d が `org_reseat_test.go`(852 → 858 行)で「分けるのは次の変更と一緒に」に当たったが、払っていない。2 つの直しのテストは新しい `reseat_start_test.go`(237 行)に入ったので、`reseat_test.go` は 1,855 行のまま。`internal/cli/org_reseat.go` は 510 → 516 行で 800 行の下。数は b142944d(Go を最後に変えた commit)の `wc -l`。Related に `git show 80f58f86:...` を足した |
| 496 テストの穴 | test report の Test gaps 10〜13 を (k)〜(n) に足した((k) role だけ・driver だけの割り込みは表の test だけが見る。(l) 割り込みは同じ goroutine のフックで、本当の並行 spawn ではない。(m) 立てる時点で終わった reseat の CLI の出力を見る test が無い。(n) b142944d の 2 文は文ごとの部分一致)。題、Impact、Why deferred、Trigger、Related に合わせて足した。Trigger の末尾に持ち越しの注: この sync-docs が skill の reseat の行と節、director skill の「座席を立て直す」を編集して (h) に当たり、払っていない(skill の文と help を比べる test は新しい test になる)。(g) は 2 つの直しも実機で見ていない |
| 262 director skill の受信箱の節 | 持ち越しの注: V24 の編集が「件の処理」の `usage_limited` の項で、この Trigger に当たり、払っていない。verify はこの行を挙げていなかった(自分の編集で当たった) |
| 464 scaffold version note | 持ち越しの注: 3 回目の sync-docs が director skill を編集して当たり、払っていない。skill は scaffold, `manifest.toml`, `ralph upgrade` に今も触れない(root の写しを `grep -c -i` で数えて 0) |
| 180 org skill の command table | 引用している `reseat` の行の長さに、3 回目の sync-docs のあとの 2,619 字(表の 1 行として)を足した。2,286 から +333。Trigger は (a) のまま、払っていない |

当たりとみなさなかった候補(Trigger の文を読んだ。1 回目と 2 回目で外したものは繰り返さない):

- 380(送って確かめる手順)、381(wait の終了コード): V22 の編集は 7(途中で失敗したとき)で、送る手順(「答える順」の 2)も、終了コードの文も編集していない
- 365(`director_skill_sync_test.go`): 編集していない
- 361(`Stop` が座席の spawn のロックを取らない)、179、183: `Stop` は cycle 3 の差分の外
- 191、192、288、431(`Spawn` の idempotent の枝、`idempotentRespawn`): `startAll` は `Spawn` の戻りを読むだけで、`Spawn` も `idempotentRespawn` も変えていない
- 159(org の quality pipeline)、377(split-plan の例と leader の手順)、401(`[org].state_dir` の段落)、249(merge-order の follow-up の文): 編集したのは reseat の行と節、director skill の 2 か所で、どれの Trigger も名指さない
- 494 の (a)〜(e)、(g)〜(i)、(k)、496 の (a)〜(g)、(i)、(j): verify が読んだとおり、cycle 3 の差分は `startAll` の終わり方、`reseatRetryCommand` の引数、`ReseatResult.Changed` の doc、help の 2 文で、その Trigger が名指すものではない

### 確かめたこと

| コマンド | 結果 |
|---|---|
| `./scripts/sync-skills.sh`、2 つの template の写しへの `cp`、`shasum -a 256` | skill ごとに 4 つの写しが一致(org `04366d42dbbd`、director `5b2c1c81ccdc`) |
| `./scripts/check-sync.sh` | PASS: all files in sync(KNOWN_DIFF 6) |
| `./scripts/check-skill-sync.sh` | 14 skill が lock-step |
| `GOFLAGS=-timeout=30m go test -count=1 -run 'Skill\|Director\|Reseat' ./internal/org/` | ok(109.9 s)。skill の文を固定するテストと reseat のテストが、編集のあとも通る |
| `go test -count=1 -run 'Skill\|Reseat\|OrgHelp' ./internal/cli/ ./internal/scaffold/` | ok(cli 44.7 s。scaffold は該当テストなし) |
| `./scripts/run-static-verify.sh`(changed scope) | exit 0。`gofmt: ok`、golangci-lint `0 issues.`、branch secret scan(`e902705c..c856fc27`)clean、tech-debt README の plan 参照 OK |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-org-reseat.md` | P6 と Risks の編集の後も、Progress の編集の後も `95a056c00a9a`(編集の前は `f7e78c0dd96c`) |
| tech-debt の `tdcheck3.py` | 上のとおり、問題 0 |
| 行の幅(skill の追加行、全角を 2) | org の最大 76、director の最大 77。既存の行にも 80 を超えるものがある |
| 文字化け(U+FFFD) | 差分に 0 件 |

走らせていないもの: `./scripts/run-test.sh` の全体(Go のコードは変えていない)。`ralph org reseat --help` の実出力は読んでいない(help は変えていない。b142944d の後の出力は verify が読んだ)。

### 人に残すもの

- 計画の digest が `f7e78c0dd96c` から `95a056c00a9a` に変わった。`- Approved:` の行は書き換えていない。承認の記録し直しは orchestrator が行う
- L19〜L21 は Go の文字列と分岐で、直していない。tech-debt の 494 行の (l)(m) に払い方がある。直すなら、パイプラインを `/self-review` からやり直す
- help の `reseated` の文(L21)は「途中で終わっても書く」のままで、README と skill と仕様の FR-8 は条件を書いた。help と文書がこの点でずれたままになる
- 計画の AC の箱は `[ ]` のまま(`/pr` か orchestrator が付ける)。仕様の AC-12 の `[x]` は、実機の binary が cycle 2 と cycle 3 の直しより前であること(I9)を含めて、人が持つ
- origin/main は `bbe2807d`(#328、send-claude-paste)まで 20 commit 進んでいる(ローカルの `origin/main`。`git fetch` はしていない)。HEAD(`c856fc27`)で `git merge-tree --write-tree --name-only HEAD origin/main` を回すと、`AGENTS.md` と `docs/tech-debt/README.md` の 2 か所が衝突する。org skill の 4 つの写し、`docs/specs/2026-08-01-org-runtime.md`、`internal/cli/org.go`、`internal/org/verbs.go` は自動で入る。この確認は、この commit の前の HEAD で行った。tech-debt の 6 行の編集を含めた衝突は、取り込むときに見る。`/pr` の前に取り込む必要がある
- release ノートに書く項目は 1 回目のまま。cycle 3 の直し(ほかのコマンドが別の設定で立てた座席の扱い)も、reseat の項に含まれる
