# sync-docs report: gate-tool-versions

- Date: 2026-11-13
- Plan: `docs/plans/active/2026-11-12-gate-tool-versions.md`(承認の digest `2774af0aab07`。計画の `- Approved:` の行と同じ)。issue #311
- Branch: `feat/gate-tool-versions`。base は origin/main `bb416c48`(`349f5763` で取り込み済み)。run 開始時の HEAD は `349f5763`
- Pipeline run: 1 回目(cycle 1)
- 先行 report: self-review `245b85f9`(追記 `5b84df9a`。Merge: yes。2 回目の review は LOW 5 件で、`9c19e5ba` が 3 件を直した)、verify `5389757f`(pass。MEDIUM V1、LOW V2〜V6、INFO I1〜I6)、test `432e51ad`(pass。2 回目の実行。Test gaps 4 件)。V1 の直しは `e7274fec`(pnpm を止める 2 変数)と `748ef386`(doctor の環境のテスト)
- 見た範囲: `git diff origin/main...HEAD`(HEAD `349f5763`、55 ファイル)。この run が変えたコードは `internal/cli/doctor.go` の注釈 1 か所だけで、`go build ./... && go vet ./internal/cli/` は rc 0、gofmt も通った。help の文と実装の対応は verify が読んだ結果を使い、build し直していない
- 制約: 計画の本文は変えていない。変えたのは Progress の節だけで、`./scripts/plan-visual.sh digest` は `349f5763` の版も今の版も `2774af0aab07`(`git diff --numstat` は 6 行追加、3 行削除)。Go のテスト、skill、雛形のコードは変えていない
- 差分の大きさ(`/pr` が walkthrough を決める材料。sync-docs の commit の前の数): `git diff origin/main...349f5763` は 55 ファイル、+7,410/-301(計画、report、insights を含む)。`docs/` の外は 47 ファイルで +6,906/-301、そのうちテストと testdata を除くと 27 ファイルで +3,291/-181(skill と recipe の写し、新しい `internal/toolchain` 6 ファイルと `internal/org/gate_tools.go`、`internal/cli/doctor_node.go` を含む)。walkthrough は書いていない

## Summary

verify の V2(README)、V3(spec の FR-7)、V4(`doctor.go` の注釈)、V5(tech-debt の 15 行)、V6(`doctor_node_test.go` の大きさ)は、すべて文書の直しか tech-debt の行で払った。I3 の手順書の参照は、#308 の節から新しい「Version managers (mise, asdf)」の節への 1 文にした。I4(文字を濾す 3 つの処理)は tech-debt の新しい行にし、test の Test gaps の 1 つめ(cli のテストが本物の probe を通ること)ももう 1 行にした。handoff に無い追加が 2 つある。quality-gates の 2 つの写しの gate の段落に、ツールの版を足したこと。計画の Progress に、`ProbeEnv` が doctor の道で 2 回当たることと、2 回目の Ctrl-C の扱いの小さな差を書いたこと。

## Changes made

| File | Change |
|------|--------|
| `README.md` | gate の段落に、`TOOLS:` の行、記録の `tool_dirs` と `tool_<tool>`、probe の 15 秒、doctor の「Node toolchain」を足した。1 行の差分 |
| `docs/specs/2026-10-07-org-multi-org-director.md` | FR-7 に「2026-11-13 の追記(issue #311, ...)」の小項目を 1 つ足した。#309 の 2026-11-12 の注と同じ形で、計画への path は `docs/plans/archive/2026-11-12-gate-tool-versions.md` と書いた(`/pr` が計画を archive へ動かすので、merge のあとに合う path) |
| `internal/cli/doctor.go` | コードの変更なし。`runDoctorOpts` の上の注釈に 1 文足し、subprocess を起こす Check に Check 14(`git ls-files` を 10 秒まで、login shell の probe を 20 秒まで)を挙げた。origin/main の 1,624 行から、この branch の本体で 1,645 行(`349f5763`)、この pass の注釈で 1,648 行 |
| `docs/recipes/isolated-herdr-server.md`、`templates/base/docs/recipes/isolated-herdr-server.md` | 「When herdr keeps reporting a seat as working」(#308 の節)から「Version managers (mise, asdf)」の節への参照を足した。2 つの写しは `cmp` で同じ。`docs/evidence/` の引用は 14 件・7 file のまま |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md` | handoff に無い追加。gate の段落の「記録は gate の前後の load average を持つ」の文に、ツールの版、`tool_dirs` と `tool_<tool>`、probe が時間切れか失敗でも verdict は変わらないことを足した。2 つの写しは KNOWN_DIFF で違うので、別々に編集した |
| `docs/tech-debt/README.md` | 15 行(159、180、238、262、269、293、294、367、371、372、390、398、405、415、464)の Trigger の末尾に持ち越しの note、Related の末尾に計画の path を足した。新しい 3 行(489〜491)を、#317 の 2 行(492、493)の前に足した。490 行から 493 行。`git diff --numstat` は +18/-15(15 行は書き換え、3 行は追加) |
| `docs/plans/active/2026-11-12-gate-tool-versions.md` | Progress の節だけ。test と sync-docs の行、2 つの小さな差の行を足し、`Review artifact created`・`Verification artifact created`・`Test artifact created` を `[x]` にした。`PR created` は `[ ]` のまま |
| `docs/reports/sync-docs-2026-11-13-gate-tool-versions.md` | この report |
| `docs/insights/events/2026-11-13-gate-tool-versions.jsonl` | `sync_docs` の行(cycle 1、pass)。ファイル名の日付は UTC |

commit は 2 つ。内容の commit `4684ecf0`(9 ファイル、+38/-24)と、この report と insights の commit。report が自分の commit の SHA を書けないので分けた。

## verify と test が挙げた項目

### V2: README の gate の段落

`LOAD:` の行と `load_*` の鍵を挙げる段落に、`TOOLS:` と `tool_dirs`、`tool_<tool>` を足した。値は版、`missing`、`unknown`、`mixed` のどれか。probe は 15 秒の 1 回の shell で、時間切れや失敗でも verdict は変わらないことも 1 文で書いた。同じ段落の末尾に doctor の「Node toolchain」を 1 文。`LOAD:` か `load_start` に触れる文書のうち、ツールの版に触れなかったのは README だけだった(verify の grep)ので、これで揃った。

### V3: spec の FR-7

#309 の注と同じ形の小項目にした。中身は、gate が走らせるスクリプトの前にツールの版を測って log と記録に残すこと、verdict は変えないこと、`TOOLS:` を reviewer が BLOCKED の本文に写すこと、doctor の Check。実装の細部は計画に任せ、計画への path を添えた。

### V4: `doctor.go` の注釈

注釈が subprocess を起こす Check として挙げていたのは Check 1・2 の `--version` と Check 11c(と opt-in の Check 12)だけだった。Check 14 を足した。コードは変えていない(`git diff` は注釈の行だけ)。

### V5: tech-debt の 15 行

行を 1 つずつ開き、Trigger を読んで、このブランチが引いたかを決めた。note は Trigger セルの末尾に `2026-11-13: feat/gate-tool-versions (#311) ...` の形で足し、`Carried over to the next change that meets this trigger` で締めた。Related の末尾には、計画の path を `docs/plans/active/...` で足した(`/pr` の `archive-plan.sh` が `archive/` に書き換える)。

| 行(先頭の語) | note の中身 | 計画の節 |
|----|----|----|
| 159「org runtime FR-7 quality gate is prompt-instructed」 | gate が測る段を足したので品質の流れが変わった。規則は今も文章で、hook は無い | Scope、AC5 |
| 180「Two documentation findings of org-stop-all」(a) | `/org` skill の表の `gate` の行が 3,268 から 4,804 字、`regate` の行が 5,919 から 5,936 字(行全体の文字数)。`stop` の 1,208 字と `disband` の 1,219 字は変わらない | Scope |
| 238「Small findings of org-gate cycle 2」(b) | まだ開いている。`gateInterrupted` は `gateRunInterrupted` に書き直したが、`gateCheckAfter` と `recordGate` の待ちは `true` を渡して「the scripts were stopped」を出す。verify の V5 は「直せたので直すか書く」と言ったが、直す所は Scope の外なので書いた | S2 |
| 262「Three test gaps of the org-director-attach PR」 | `/director` skill の `### gate の失敗の受け入れ`(受信箱の節の中)を編集した。(1)〜(3) は開いたまま | S4 |
| 269「Two mutants of the org-director-start PR」(b) | `director_skill_sync_test.go` が 1,307 から 1,318 行。(b) は開いたまま | S4 |
| 293「`regate_test.go` and `regate_merge_test.go` are over the 800-line」 | `regate_test.go` が 1,771 から 1,796 行。`regate_merge_test.go` は 2,112 行のまま(7 行は書き換え) | S2 |
| 294「`ralph org merge-order` reads a `gate_failed` that `ralph org regate` wrote」 | 記録に `tool_dirs` と `tool_<tool>` の鍵が増えた。`merged=` の鍵は足していない | D4 |
| 367「Only the leader's role prompt keeps a `wip:` commit out」 | `gate.go` と `gate_run.go` は gate が測るものを変えたが、`wip:` の確かめは変えていない | Scope |
| 371「The recipes under `templates/base/docs/recipes/` cite `docs/evidence/`」 | 手順書を編集したが、`docs/evidence/` の新しい引用は足していない。行は開いたまま | Scope、S4 |
| 372「RC1 (org recheck 2026-10-24)」 | `reviewer.md` が 594 から 599 行。agmsg の件は変えていない | S4 |
| 390「Four test files went past the 800-line guide with the merge cleanup of #263」 | 4 file のうち `director_skill_sync_test.go` だけが変わった | S4 |
| 398「Two test gaps that the `/test` of local-config-org left」(a) | `Regate` のテストは偽の probe を使い、本物の probe を通るのは `Gate` のテストだけ | S2 |
| 405「Three files went past the 800-line guide with the ci regate of #264」 | `regate.go` が 1,054 から 1,058 行、`org_regate_test.go` が 1,305 から 1,335 行、`regate_ci.go` は 901 行 | S2 |
| 415「Five files went past the 800-line guide, or grew past it, with script-local-config」 | `doctor.go` が 1,624 から 1,648 行、`doctor_node.go` は新しい 691 行(800 未満) | S3 |
| 464「The `/director` skill ... do not say what a director does when it sees the scaffold version note」 | `/director` skill を編集した。ほかの動詞の help と `director.md` は変えていない | S4 |

行数は `git show origin/main:<file> | wc -l` と `wc -l` で数えた(origin/main `bb416c48` と、この pass の HEAD)。

verify が挙げなかった 262 と 269 の (b) と、quality-gates の 2 つの写しは、掃引で見つけた。

### V6 と I4 と test の Test gaps: 新しい 3 行

- 489 行目: 文字を濾す 3 つの 1 行化(`nodeErrText`、`toolchain.cleanText`、`gateToolsOneLine`)。判定が同じなのは 2 つで、`gateToolsOneLine` は C0 と DEL だけを置き換える(C1 は通し、不正な UTF-8 は `strings.Map` が U+FFFD にし、長さの上限が無く、それ自身のテストも無い)。違いは scratch の Go で確かめた(`a\x80b\xffc\u0085d\x1be` を通すと、C1 は残り、不正な 2 byte は U+FFFD になる)。払うなら `toolchain.CleanText(s, limit)` を 1 つ出して 3 か所が使う。self-review の 2 回目の LOW 2 で、計画が「sync-docs で tech-debt に書く」と決めたもの
- 490 行目: `internal/cli/doctor_node_test.go` が 996 行(verify は 994 と数えた。`748ef386` が 2 行足した)で、800 行の目安を超えた。最初の分け方は、中断のテスト(約 165 行)とバージョンファイルのテスト(約 250 行)。V6 に当たる
- 491 行目: test の Test gaps の 1 つめ。`internal/cli` の gate・regate・merge のテストは、`internal/org` が持つ probe の差し替え口(`gateToolsProbe`、非公開のパッケージ変数)を触れないので、本物の probe を通る。1 回の gate で約 0.3 秒、版の値は期待に入れておらず(`TOOLS:` の行は `gateCLIToolsLinePattern` が形だけ見る)、結果は手元の版に左右されない。口を公開するか、cli 側に別の差し替え口を作るときが Trigger

3 行とも、関数とテストの名前で書き(`file:line` は 0)、Related は計画の path にした。test の Test gaps の残り 3 つ(負荷の高い手元で `internal/org` が 30 分を超えること、本物の mise・asdf の shim を通る場面、coverage を取っていないこと)は行にしていない。1 つめは計画の Known gaps(R2)と tech-debt 485 行に、2 つめは前からの傾向で、3 つめはこの repo の通常の扱いで、新しい負債ではない。

### I1、I2、I3、I5、I6

- I1(doctor の中断が終了コード 1): 計画の Progress の S6 の行が理由を書いているので、記録されたずれとして扱った。D6 と AC6 の本文は計画の本文なので変えていない。doctor の help の文書にも、中断の stdout の行(「Interrupted: ...」)は書いていない(help は Check の一覧を持たない)
- I2(git が列挙できないときの doctor の場所): 計画の D8 と食い違うが、テストがある(verify のとおり)。文書に書く所が無い
- I3: 手順書の #308 の節に、新しい節への参照を足した。「ralph doctor and ralph status」の箇条と `/org` skill の前提は、読む doctor の Check を一覧にしていないので、「Node toolchain」を足していない
- I5(Go の注釈が `docs/plans/active/...` を指す): `archive-plan.sh` は Go の注釈を書き換えない。この repo の前からの形で、直していない
- I6(issue の「走った版」と gate が残す版の差): 計画が D1・R2 で決め、tech-debt 485 行に残している。新しい行は足していない

## 掃引

### Trigger の掃引

tech-debt の Trigger の列を python で切り出し(`" | "` で分けた 4 つ目のセル、取り消し線の中を外す)、このブランチが触れたファイルと関数の名前(`gate.go`、`gate_run.go`、`gate_record.go`、`regate.go`、`gateInterrupted`、`gateRunInterrupted`、`doctor.go`、`runDoctorOpts`、`reviewer.md`、`/director` skill、`/org` skill、`isolated-herdr-server.md`、`quality-gates.md`、`director_skill_sync_test.go` ほか)で 3 回探した。当たった行の Trigger を 1 つずつ読んだ。

当たらないとした行。

| 行(先頭の語) | 判断 |
|----|----|
| 118「`templates/base/docs/quality/quality-gates.md` の「Must pass in CI」」 | Trigger は template の quality-gates の「Must pass in CI」節か、template の verify チェーン。この branch が足したのは gate の段落の 1 文で、どちらも変えていない |
| 156「Residual risks of `/cross-review`'s codex reviewer」 | codex の呼び出し(`/cross-review`、`/plan`)に触れていない |
| 177「`ralph org stop --all` and `disband --all`」 | `stop --all` と `disband --all` に触れていない |
| 195「A `ralph` of another version on the pane's `PATH`」 | `newOrgStartCmd` の `RunE` と director の起動に触れていない |
| 240「Since build/go-1-27, `ralph org gate`, `ralph org merge`, and `ralph org regate` cancel with `signal.NotifyContext`」 | Trigger は 3 つのコマンドの signal の登録の編集。この branch が変えたのは、中断された gate が言う文(`gateInterrupted` から `gateRunInterrupted`)で、`newOrgGateCmd` などの `signal.NotifyContext` の呼び出しは変えていない(diff に `NotifyContext` の行が 0) |
| 380「RC6 ... the `/director` skill's check after a send」、381「RC9 ... a director's wait」 | `/director` skill の変更は gate の失敗の受け入れの節で、send の確かめの手順にも wait の終了コードにも触れていない |
| 401「The condition under which the verbs that change the ledger are refused」 | 拒否の条件にも `[org].state_dir` の段落にも触れていない |
| 421「`skill_split_questions_test.go` is over 800 lines」、426「The role prompts grew with org-cross-review」、427「Seven test gaps of org-cross-review」 | Trigger は別の file と、`leader.md` か `reviewer.md` に節を足すこと。`reviewer.md` の変更は gate の BLOCKED の本文と例への数行で、節は足していない(426)。421 の file は変わっていない |
| 465「`checkManifestVersion`」 | `doctor.go` は変えたが、`checkManifestVersion` は変えていない |
| 468「`gate.lock` has no wait limit」、470「An acceptance (`ralph org gate accept`) has no effect on a `via=regate` `gate_failed`」、480「Four test gaps of feat/gate-lock-accept」 | 468 と 470 は `gate.lock` の待ちにも `gate accept` にも leader の役割 prompt にも触れていない。480 の Trigger は `waitGateMaxLoad`、`gateRunLoad.finish`、`newOrgRegateCmd` の `Regate` の呼び出しで、`gate_load.go` は diff に無く、`newOrgRegateCmd` で変わったのは `Long` の文字列と `printRegateLines` の `printGateTools` の 1 行(呼び出しの引数は同じ)。`gate_load_test.go` の変更は期待値にツールの鍵を足しただけ |
| 471「`ralph org gate accept` is the human's step」 | Trigger は「人の答えなしに director が受け入れる」ことの報告か、緩い規則の依頼。`/director` skill の `### gate の失敗の受け入れ` は、ツールの版が原因の失敗を「直して gate をやり直す。受け入れない」と書いただけで、受け入れの規則は緩めていない |
| 473「Known gaps of fix/send-screen-fallback」 | send の画面の道に触れていない |

### 掃引で足したもの

- 262 の note: `### gate の失敗の受け入れ` を編集したので当たると読んだ。(1)〜(3) は払っていない。verify の V5 と handoff の一覧に無く、掃引で見つけた
- 269 の (b): `director_skill_sync_test.go` が変わった。同上
- quality-gates の 2 つの写し: README の V2 を直したあと、`load average` で grep して見つけた。gate の段落が負荷しか言わないと、ツールの版が文書に出ない

## 実装と合っていると確かめたもの

| Doc / contract | 結果 |
|----|----|
| `AGENTS.md` の Repo map | 合っている。`internal/toolchain/` の行が前の slice で入り、`internal/org/` の行にも gate の記録のツールの版が入っている。この pass では変えていない |
| `ralph org gate --help`、`ralph org regate --help` | verify が読んだ結果(probe の上限、`TOOLS:`、止める変数)と README、quality-gates の文が合う。V1 の直し(`e7274fec`)で help の文も直っている |
| `.claude/skills/org/SKILL.md`、`.claude/skills/director/SKILL.md` とその 3 つの写し | `check-skill-sync.sh` が 14 skill で ok。`templates/base/` の写しとの同期は `check-sync.sh` が通る。この pass では触っていない |
| reviewer の役割 prompt(`internal/org/prompts/reviewer.md`) | `TOOLS:` を BLOCKED の本文に写す。テストが固定している(verify の AC5) |
| 手順書の 2 つの写し | `cmp` で同じ |
| `docs/architecture/repo-map.md` | `toolchain` の語が無く、`internal/` の記述を持たない。変える所が無い |
| `internal/toolchain/doc.go` | V1 の直しで「pnpm reads as the pnpm on PATH even where packageManager names another」が入り、言い切りが正確になった。この pass では触っていない |

## 実行したもの

- `./scripts/check-sync.sh`: rc 0、`PASS: all files in sync.`
- `./scripts/check-skill-sync.sh`: rc 0、`14 skill(s) in lock-step`
- `git diff --check 349f5763 HEAD`: rc 0。U+FFFD は 0 件
- tech-debt の確かめ(`git show` した run 開始時の版との比較): 変わった行は 15 行で、どれも Trigger と Related の列だけが伸びた。追加の行は 3 つ(489〜491)。行の列は 5、列ごとのバッククォートと `~~` は偶数。足した文に `file:line` は 0
- 計画の参照: `docs/tech-debt/README.md` の `docs/plans/(active|archive)/...` を `verify.local.sh` の `check_tech_debt_plan_refs` と同じ式で調べて、無い参照は 0 件
- `./scripts/plan-visual.sh digest docs/plans/active/2026-11-12-gate-tool-versions.md`: `349f5763` の版も今の版も `2774af0aab07`
- `go build ./... && go vet ./internal/cli/`: rc 0。`gofmt -l internal/cli/doctor.go` は出力なし
- `./scripts/insights-append.sh --slug gate-tool-versions --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`: 下の「Insight event」
- テストと `run-static-verify.sh` の全体は走らせていない(Go のコードは注釈の 1 文だけで、verify と test の結果が有効)

## 残り

- 計画の AC のチェックボックスは `[ ]` のまま。handoff が Progress の節だけを言ったので触っていない。`/pr` の前に付ける
- 計画の 2 つの小さな差は tech-debt の行にせず、計画の Progress にだけ書いた。(1) doctor の道は `toolchain.ProbeEnv` を 2 回当てる(呼ぶ側の `doctor_node.go` と `Probe`)。冪等で結果は同じ。(2) 2 回目の Ctrl-C は、`stop()` が probe の戻りで登録を外すまで doctor でも捕まえられ、そのあいだ doctor は probe の process group を止める。どちらも害が無く、払うときの Trigger を持つ具体的な変更がまだ無いので、行にすると register の行が増えるだけになる
- 371 行目(recipe の `docs/evidence/` の引用)は、この branch が新しい引用を足さなかったので、言い換えるかの判断は前の branch から変わらず開いている
- 485 行目(MEDIUM。gate が残す版は、スクリプトの前に `/bin/sh` が解決した版で、shim を通した子の版ではない)は、この pass で変えていない。issue の受け入れ条件 1 の「走った版」との差で、PR 本文の Known gaps に同じ名前で書ける
- `docs/tech-debt/README.md` は別のセッションも編集しうる。この pass は 15 行への挿入と書き換えと、#317 の 2 行の前への 3 行の追加で、merge の衝突は末尾付近で起きやすい
- `docs/plans/active/...` への参照を Related に足した。`/pr` の `archive-plan.sh` が書き換える

## Insight event

```
./scripts/insights-append.sh --slug gate-tool-versions --flow standard --phase sync_docs \
  --verdict pass --cycle auto --source skill || true
```

終了コード 0。`--slug` は日付なしの slug。`docs/insights/events/2026-11-13-gate-tool-versions.jsonl` に足した行は cycle 1、pass、ts は 2026-11-13T04:14:44Z(UTC の日付なので、ファイル名は 2026-11-13 で、前の phase の行と同じ file)。この時点の `git status --short` は、その jsonl の `M` と、未追跡のこの report だけで、迷子のファイルは無かった。
