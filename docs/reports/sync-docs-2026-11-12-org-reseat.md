# sync-docs report: org-reseat

- Date: 2026-11-13(JST。UTC では 2026-11-12)
- Plan: `docs/plans/active/2026-11-12-org-reseat.md`(承認の digest `3058f6f66b5b`)。仕様は `docs/specs/2026-11-12-org-reseat.md`。issue #312 の 3 本目(#312c)の 1 本目
- Branch: `docs/spec-org-reseat`。run 開始時の HEAD は `9a295a84`(origin/main の `14531858`、#308 を取り込んだ merge)
- Pipeline run: 1 回目(`cycle-count.json` は `cycle` 1)
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
