# sync-docs report: test-stability

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-21-test-stability.md`(承認の digest `edae494b2ed1` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。Progress checklist は digest の範囲の外)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値。上限は 2)。base は `293dff17`、branch は `fix/test-stability`。run 開始時の HEAD は `4416c4c9`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-21-test-stability.md`(LOW 3 件、L1 と L3 は 6d990dfc で直り、L2 は残した)、`docs/reports/verify-2026-10-21-test-stability.md`(pass。INFO V1〜V4)、`docs/reports/test-2026-10-21-test-stability.md`(pass。G1 は ec852215 で埋まった。G2〜G6 は INFO)
- この run が変えたのは、tech-debt の 3 つの RESOLVED コメント、計画の Progress checklist の 1 項目、この report、insight event。コード、script、skill、rule、README、仕様、根拠の表、先行 3 report には触っていない

## Summary

`293dff17...4416c4c9` の差分が変えたのは、Go の pack の `run_tests`(`GOFLAGS` の先頭に `-timeout=30m` を置く)、F9 の subtest の stub、doctor の probe の bound(`doctorProbeTimeout`)、2 つの shell suite、dry-run の `sent` のテスト、それを書く記録(tech-debt の 5 行、根拠の表、仕様の印)である。この変更を書く文書がほかにあるか、また「Go の pack は go の既定の 10 分で止まる」「run-verify や gate に `GOFLAGS=-timeout=30m` を付ける」と書く文書が今も残っていないかを、README、AGENTS.md、`docs/recipes/`、`docs/quality/`、`docs/architecture/`、`.claude/rules/ralph/`、`/org` と `/director` の skill とその写し、pack の README 2 つ、`.github/workflows` から探した。偽になった文はなかった。

直した点は 3 つある。どれも tech-debt の RESOLVED コメントで、S6 が書いたあとに pipeline が見つけたことと照らして足りなかった所である。

1. doctor の行のコメントに、test report の G1 を埋めた `TestMain_RaisesTheDoctorProbeTimeout`(ec852215)を名指した。あわせて、test report の G2(元の失敗が負荷 150 でも再現しなかった)を 1 文で足した。
2. F9 の行のコメントで、「負荷の下では何も測っていない」を、test report の 13(base と HEAD の stub を繰り返した結果)に置き換えた。
3. F10 の行のコメントで、「`ralph upgrade` が下流に届ける」を、コードとテストで読んだ範囲と、upgrade を実行していないことに言い直した(test report の G6、verify の Coverage gaps)。

## Changes made

| File | Change |
|------|--------|
| `docs/tech-debt/README.md`(304 行、F9 の RESOLVED コメント) | 末尾の文「Why the failures came only under heavy load is not explained: ... nothing was measured under load ...」を、test report の 13 の数に置き換えた。旧 stub は `-count=20`(1 つの process)で 20 回のうち 15 回と 16 回(load average 8 と 16)、別々の `-count=1` で 10 回のうち 1 回、`-count=10` を load 93〜123 で回して 10 回のうち 4 回、すべて「failed (exit 128)」で落ちた。新しい stub は load 7〜153 の 60 回すべて通った。したがって重い負荷がなくても race は見える。なぜ 1 つの process の繰り返しのほうが別々の実行より落ちやすいか、なぜ試行の失敗が重い負荷の下で出たかは説明がつかないので、「git の起動が 2 回目以降は速い」と「遅れた Go の timer」を、確かめていない推測として残した。test report の G3 の「記録に足すかは `/sync-docs` に任せる」への答え |
| `docs/tech-debt/README.md`(310 行、F10 の RESOLVED コメント) | 「`ralph upgrade` brings the change to downstream Go projects」を、「`ralph upgrade` は pack の `verify.sh` を template から core のファイルと同じに置き換える(`internal/upgrade/replaceplan.go` を読んだ。`TestPlanCoreReplaceDesired_PreservePrefixes_TemplatePresentClassifiesNormally` が、template に中身のある pack のファイルは更新されると固定する。実際の project で upgrade は打っていない)ので、下流の Go project は次の upgrade で受け取る」に変えた。事実の範囲は同じで、確かめた根拠と確かめていないことを書いただけ |
| `docs/tech-debt/README.md`(313 行、doctor の RESOLVED コメント) | `TestMain_RaisesTheDoctorProbeTimeout`(`internal/cli/cli_test.go`、ec852215)を名指した。`doctorProbeTimeout` が 5 s の既定を超えていなければ落ちる。足す前は、`TestMain` が bound を上げない mutant が `internal/cli` の全体(813 本)でも緑だった(stub が 5 s を超えたときにだけ差が出るため)。続けて、元の失敗は `/test` で再現しなかったこと(doctor の 5 試験が load 20〜160 で 115 回すべて通り、load 150 の 15 回は 1 試験 0.7〜1.4 秒)と、2026-10-19 と 10-21 に stub の `--version` が 5 s を超えた理由は負荷だけとは確かめられていないことを書いた。末尾の引きに test report を足した |
| `docs/plans/active/2026-10-21-test-stability.md`(Progress checklist) | 「Test artifact created」の下に、この run の sub-bullet を足した。digest の範囲の外なので digest は変わらない。Risks の文には触っていない(下の V1) |
| `docs/reports/sync-docs-2026-10-21-test-stability.md` | 新規 |
| `docs/insights/events/2026-10-21-test-stability.jsonl` | sync_docs の event を 1 行足した(下の「Insight event」) |

3 つのコメントのほかは `git show HEAD:docs/tech-debt/README.md` と 1 行ずつ同じである(変わった行は 304、310、313 の 3 行だけ。行数は 336 のまま)。3 つの行の欄はどれも 5 つで、第 1 欄と第 4 欄の `~~` の打ち消し、第 5 欄の計画の path は変えていない。新しい文に `file:line` の引きはなく、バッククォートは各行で偶数、U+FFFD はない。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 「Go の pack は go test の既定で走る」「run-verify や gate に `GOFLAGS=-timeout=30m` を付ける」と書く文書 | なし | `git grep -i -E 'GOFLAGS\|go test -timeout\|-timeout=30m\|-timeout 30m\|run_tests'` を、reports、evidence、plans、insights、tech-debt、tests、internal の外に打った。当たったのは各 pack の `verify.sh` の `run_tests`、golang の pack の README と `verify.sh`、仕様 105 行の FR-7 の注(この PR が書いた)だけ。`10-minute`、`10 minutes`、`default.*timeout` も README、AGENTS.md、`docs/recipes`、`docs/quality`、`docs/architecture`、rules、skills、packs、templates、scripts、`.github` に打ち、当たりは `[org.director].ack_minutes`(10 分、director-wake の recipe と README 258 行)と golang の pack の新しい文だけ |
| `packs/languages/golang/README.md` と `templates/packs/golang/README.md` | 合っている | 「`GOFLAGS` の先頭に `-timeout=30m`、go test の既定は 10 分、`GOFLAGS` か `go env -w` の timeout があとに来て勝つ、`GOFLAGS=-timeout=10m` で 10 分」は 114〜115 行の `run_tests` と合う(verify も同じ判定)。2 つの README と 2 つの `verify.sh` は `check-sync.sh` で同一 |
| `.claude/rules/ralph/golang.md`、`packs/languages/golang/rule.md` とその写し(11 行) | 変更なし | 「完了前に `go test ./...` を打つ」と言うだけで、bound は言わない |
| `/org` skill の `gate`・`regate` の行、`/director` skill の regate の手順(`.claude/skills/` と `.agents/skills/`) | 変更なし | gate と regate が「数分かかる」ので前景なら tool の timeout を長くする、と言うだけで、go の 10 分の既定にも `GOFLAGS` にも触れない。F10 の修正で偽になる文はない。なお「数分」は、試行で gate が 28〜32 分かかったこと(tech-debt の F9 の行)と合わない。この PR が変えた所ではないので直していない(下の Left open) |
| `docs/quality/`、`docs/recipes/`、`docs/architecture/`(`repo-map.md` と `quality-gates.md` を含む) | 変更なし | go test の bound を書く文も、`tests/test-*.sh` の一覧もない(`git grep -o -E 'tests/test-[a-z-]+\.sh'` が README、AGENTS.md、quality、architecture、recipes、rules、skills、`.github`、scripts に出すのは、`model-routing.md` の `test-agent-models.sh` と cross-review の skill の `test-xreview-helpers.sh` だけ)。新しい 2 つの suite は `scripts/verify.local.sh` が glob で拾うので、一覧に足す文書がない |
| doctor の probe の 5 s を書く文書 | 変更なし | README、recipes、quality、rules、skills に書く文はない。利用者向けの 5 s は変わっていない(`doctorProbeTimeout` の既定は 5 s) |
| `scripts/check-sync.sh` の `ROOT_ONLY_EXCLUSIONS` の説明 | 変更なし | `tests/test-check-sync.sh` が F8 の 2 つの除外を固定する。F8 の記録(#244、tech-debt の 323 行)が書く除外の理由は、この PR で変わっていない |
| tech-debt の 5 行(304〜305、310〜311、313〜314、331〜332、334〜335 行)のコメントの SHA と中身 | 合っている | verify の「記録の確かめ」が S1〜S5 の commit の中身と照らして確かめている。この run は、そのうち 3 つのコメントの末尾の文だけを変えた |
| tech-debt の計画の path | 欠けなし | `docs/plans/` を指す path を重複なしで全部引いて存在を確かめ、欠けた path は 0 件。足した `docs/reports/test-2026-10-21-test-stability.md`、`internal/upgrade/replaceplan.go`、`internal/cli/cli_test.go` も存在する。引いた 2 つのテスト名(`TestMain_RaisesTheDoctorProbeTimeout` は `internal/cli/cli_test.go` の 415 行、`TestPlanCoreReplaceDesired_PreservePrefixes_TemplatePresentClassifiesNormally` は `internal/upgrade/replaceplan_test.go`)は `func <名前>(` が 1 件ずつある |
| 根拠の表と仕様の印(S6 が書いた記録) | 変更なし | test report の 13 と 14 は、F9 の行が書く原因(stub が git の読む stdout を閉じる)と矛盾しない。根拠の表の F9 の行は原因の中身を書かないので、数を足す所がない。FR-7 の 2 つの注(根拠の表 53 行、仕様 105 行)は「負荷の下での regate は打ち直していない」と書いており、test report の G5 と同じ |

## Decisions on the reports' remaining items

| 項目 | 決めたこと |
|------|------------|
| test の G1(LOW)`TestMain` の bound の引き上げを確かめる試験がない | ec852215 で埋まった。doctor の行のコメントに名指した(上の表) |
| test の G2(INFO)元の doctor の失敗が load 150 でも再現しない | 新しい行は立てず、doctor の行のコメントに 1 文で足した。行はすでに RESOLVED で、bound を 1 分に上げた対策と、原因が負荷だけとは確かめられていないことを同じ場所に置くほうが、あとで読む人が両方を見られる |
| test の G3(INFO)base の F9 が 2026-10-18 の記録より落ちる | F9 の行のコメントに数を足した(上の表) |
| test の G4(INFO)`TestCheckScaffoldIntegrity_CorruptManifest_StrictFails` が利用者の `~/.codex/models_cache.json` を読む | 新しい行は立てない。既存の行(tech-debt 247 行、「`internal/cli`'s `TestMain` pins the doctor checks that read the developer's environment ... but not the `Org codex model slugs` check」)が、`runDoctor*` を使う試験すべてがその cache を読むと書いている。行は未解決のままで、この PR は触っていない。この試験はその一例である |
| test の G5(INFO)負荷の下の live の regate、Linux | 計画の Non-goals で、FR-7 の注にも書いてある。CI の verify が ubuntu で run-verify.sh を走らせる。新しい行は立てない |
| test の G6(INFO)`ralph upgrade` が pack の `verify.sh` を下流に届けることを確かめていない | F10 の行のコメントの言い方を直した(上の表)。PR の本文が release のノートに書く事項で、新しい行は立てない |
| verify の V1(INFO)計画の Risks の 3 つ目が、Progress の S2 の記録で誤りとわかった文のまま | 直さない。Progress の S2 が「Risks のその文は誤りで、動きは同じ」と記録している。Risks は digest の範囲の中なので、直すと承認の digest が変わり、`/implement` と `/pr` の承認の確認に掛かる |
| verify の V2(INFO)Progress の「Review artifact created」が `[ ]` | 計画の現在の Progress では `[x]` になっており(37c01116)、対応済み |
| verify の V3(INFO)打ち消した行を名指す引用が `~~` を先頭の数語で閉じる | 直さない。#244 と #246 の書き方と同じで、根拠の文書の規則を満たす |
| verify の V4(INFO)merge のあと、F9・F10・G1 と FR-7 の注の「fix/test-stability」を PR 番号にする | 系列の次の PR の作業。根拠の表 16 行の規則(merge した PR は番号で書く)がすでにそれを求めている。この PR はまだ番号がないので、書けない |

## Plan drift notes

- 計画の Affected areas は、`internal/cli/cli_test.go`(`TestMain_RaisesTheDoctorProbeTimeout`)を「cli_test.go or a doctor test file」の範囲として含み、`docs/tech-debt/README.md` も含む。G1 の直しは AC を変えない(AC4 の「`TestMain` が上げる」を試験で固定しただけ)
- 計画の Risks の 3 つ目(invalid な `GOFLAGS` で `go env` が失敗する)は、Progress の S2 の記録のとおり実際と違う。動きは同じで、直していない(上の V1)
- Progress の記録(S1〜S6、run-verify 818bd4b1、self-review の LOW、`/test`)は、差分と合う。未了は「PR created」だけ

## Left open

- `/org` skill の gate・regate の「数分かかる」: 試行では gate が 28〜32 分かかり(tech-debt の F9 の行は 1947 秒と 1664 秒)、load average 5 前後の run-verify でも 13 分かかる(計画の Progress)。この PR の前から食い違っており、この PR は所要時間を変えていないので直していない。直すなら、各 skill の 4 面(`.claude/skills/`、`.agents/skills/`、`templates/base/` の 2 つ)を同じ commit で変える(`check-skill-sync.sh` が lock-step を見る)
- F9・F10・G1 の「テストの安定(fix/test-stability で直した)」と FR-7 の 2 つの注の「fix/test-stability」: merge のあと、系列の次の PR が番号に直す(verify の V4)
- `ralph upgrade` が pack の `verify.sh` を下流に届けること: 実行していない。PR の本文で release のノートに書く
- 負荷の下での F9 と doctor の元の失敗の原因: 記録のとおり説明がつかない。推測は推測として残してある
- release はユーザーが切る(計画の Non-goals)

## Verification

| 確かめたこと | 結果 |
|--------------|------|
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-21-test-stability.md`(計画の編集のあと) | `edae494b2ed1`(承認の digest と同じ) |
| `./scripts/check-sync.sh`(編集のあと) | rc 0。`PASS: all files in sync.`(IDENTICAL 171、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`14 skill(s) in lock-step`。skill は編集していない |
| tech-debt の構造(scratchpad の python で `git show HEAD:docs/tech-debt/README.md` と比べた) | 行数 336 のまま。変わった行は 304、310、313 のみ。3 行ともバッククォートは偶数、U+FFFD なし、`file:line` の引きなし。305、311、314 行の行は欄が 5 つ(縦棒で割って 7 片) |
| tech-debt の計画の path | 欠けた path は 0 件 |
| 上の「Surfaces checked」の `git grep` と数え方 | 結果は表のとおり。読み取りだけ |

`go test`、`run-static-verify.sh`、`run-verify.sh`、`run-test.sh` は走らせていない(依頼どおり)。この run が変えたのは markdown 3 file(tech-debt、計画、この report)と insight event で、コードも script も skill も変えていない。

## Verdict

- Verdict: pass(sync-docs に stop condition はない)
- Not verified: Go のテストと 2 つの shell suite の実行(`/test` が済ませた)、`ralph upgrade` での配布、Linux での実行、負荷の下での regate と F9

## Insight event

`./scripts/insights-append.sh --slug test-stability --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で `docs/insights/events/2026-10-21-test-stability.jsonl` に 1 行足した。
