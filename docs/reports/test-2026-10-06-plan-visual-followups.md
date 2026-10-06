# Test report: plan-visual-followups

- Date: 2026-10-07(JST。実行の記録は UTC の 2026-10-06 15:17〜15:40)
- Plan: docs/plans/active/2026-10-06-plan-visual-followups.md
- Tester: tester subagent (Claude)、pipeline cycle 1(`cycle-count.json` は 1)
- Scope: branch fix/plan-visual-followups の HEAD 27d08460 と base origin/main f9baf6a7 の差分(37 ファイル、Go のファイルは 0)。behavioral test だけを実行した(静的解析は /verify で済んでいる)。plan の AC1・AC4・AC5・AC6 と、Test plan の unit・integration・regression・edge case を確かめた。3 本の shell テストと規則の確かめ方のコードブロックは、Docker の ubuntu:24.04(dash、mawk、GNU sed と coreutils)でも実行した
- Evidence: `docs/evidence/test-2026-10-06-plan-visual-followups.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない)。`run-test.sh` 自身が書いたログは `docs/evidence/verify-2026-10-06-151727.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | shell 37 ファイル(1,694 件)、Go 8 パッケージ | すべて | 0 | 1(下の注) | 260 s、rc 0 |
| `go test ./... -count=1 -cover` | 8 パッケージ | 8 | 0 | 0(`[no test files]` の 2 パッケージを除く) | 52 s、rc 0 |
| `go test ./internal/upgrade/... ./internal/cli/... -count=1 -v` | top-level 573 件 | 573 | 0 | 0 | 45 s、rc 0 |
| `bash tests/test-insights-append.sh` | 50 | 50 | 0 | 0 | rc 0 |
| `sh tests/test-skill-insight-cycle.sh`(新規) | 16 | 16 | 0 | 0 | rc 0 |
| `sh tests/test-pr-owner-lookup.sh`(新規) | 19 | 19 | 0 | 0 | rc 0 |
| `sh tests/test-pr-owner-lookup.sh`(PATH から jq を外す) | 15 + SKIP | 15 | 0 | jq の部分 | rc 0 |
| ubuntu:24.04 で上の 3 本 + jq なしの 1 回 | 50 / 16 / 19 / 15 + SKIP | すべて | 0 | jq なしの 1 回だけ | rc 0 |
| 規則の `<record>` の確かめ方(5 ケース × macOS の 3 シェル + ubuntu の 2 シェル) | 25 | 25 | 0 | 0 | 下の節 |
| mutation(私が作った 29 件のうち、適用できた 28 件) | 28 | red 21 件、残り 7 件(テストの失敗ではない) | - | - | 下の節 |

- `run-test.sh` は scope full で走った(`Requested scope: full`、`Language scope: full`)。shell の 37 ファイルは `tests/test-*.sh` の全件で、ディスク上の一覧と実行した一覧を `comm` で比べて差はなかった。
- shell の skip 1 件は `tests/test-secret-scan-branch.sh` の「git 2.41 より古い本物の git」のケース。手元の git が 2.49.0 なので飛ばされる。この変更とは関係がない。
- `run-test.sh` の中の `go test ./...` は `internal/org` 以外がキャッシュの結果だった。そのため `-count=1` で全パッケージを流し直した(上の 2 行目)。
- shell の件数の数え方: 各 suite の `PASS` / `FAIL` / `SKIP` で始まる行を数え、`PASS: 29` や `PASS: 47 / 47` のような集計行は除いた。各 suite 自身の集計行(`insights-append tests: 50 passed` など)と一致することを確かめた。
- 実行の前後で worktree の `git status --porcelain` は 0 行だった。
- この test の insight event は、このブランチの skill のとおり `--cycle auto` で worktree から記録し、`cycle` は 1 になった。worktree の `cycle-count.json` は `plan_path` が `active-plan.json` と一致する cycle 1 なので、値は合っている。ただし fallback でも 1 になるので、この記録だけでは 2 つの経路を区別できない(区別はテストの 8a で見ている)。

### shell の suite ごとの件数

| Suite | Passed / Total |
| --- | --- |
| `test-agent-models.sh` | 47 / 47 |
| `test-agent-phase-boundaries.sh` | 44 / 44 |
| `test-branch-name.sh` | 26 / 26 |
| `test-check-mojibake.sh` | 15 / 15 |
| `test-check-skill-sync.sh` | 13 / 13 |
| `test-check-template.sh` | 55 / 55 |
| `test-codex-exec-invocation.sh` | 144 / 144 |
| `test-detect-changed-languages.sh` | 83 / 83 |
| `test-detect-languages-terraform.sh` | 8 / 8 |
| `test-ensure-pr-ready.sh` | 7 / 7 |
| `test-ensure-pr-title-prefix.sh` | 13 / 13 |
| `test-gc-artifacts.sh` | 11 / 11 |
| `test-hook-wiring.sh` | 68 / 68 |
| `test-insights-append.sh` | 50 / 50(39 から 11 件増) |
| `test-language-pack-monorepo-roots.sh` | 29 / 29 |
| `test-new-feature-plan.sh` | 21 / 21 |
| `test-no-loop-references.sh` | 1 / 1 |
| `test-plan-visual.sh` | 103 / 103 |
| `test-post-edit-verify.sh` | 24 / 24 |
| `test-pr-owner-lookup.sh`(新規) | 19 / 19 |
| `test-pre-bash-guard.sh` | 4 / 4 |
| `test-ralph-config.sh` | 19 / 19 |
| `test-ralph-dispatch.sh` | 33 / 33 |
| `test-ralph-worktree.sh` | 143 / 143 |
| `test-run-verify-branch-secret-scan.sh` | 32 / 32 |
| `test-run-verify-scope.sh` | 19 / 19 |
| `test-secret-scan-branch.sh` | 246 / 246(skip 1) |
| `test-secret-scan.sh` | 123 / 123 |
| `test-self-review-scope.sh` | 64 / 64 |
| `test-skill-insight-cycle.sh`(新規) | 16 / 16 |
| `test-sync-skills.sh` | 22 / 22 |
| `test-template-purity.sh` | 10 / 10 |
| `test-terraform-gitignore.sh` | 47 / 47 |
| `test-terraform-pack-verify.sh` | 36 / 36 |
| `test-terraform-rule-frontmatter.sh` | 11 / 11 |
| `test-verify-mode-split.sh` | 59 / 59 |
| `test-xreview-helpers.sh` | 29 / 29 |

合計 1,694 件。#203 の test レポート(2026-10-06、35 ファイル 1,648 件)から 46 件増えた。内訳は `test-insights-append.sh` の Case 8 の 11 件と、新しい 2 ファイルの 16 件と 19 件で、ほかの suite の件数は変わっていない。

## Plan の edge case と、それを見ているテスト

plan の Test plan と team-lead が挙げた edge case ごとに、テストを読んで対応させた。コードには手を入れていない。

| Edge case | 見ているテスト | 結果 |
| --- | --- | --- |
| URL の末尾の `/` | `test-pr-owner-lookup.sh` の `https://github.com/yoshpy-dev/ralph/`。`sed` から `s#/+$##` を外す mutation(P1)で red | PASS |
| 大文字の owner | login の側: fixture の `YoshPy-Dev` と `Yoshpy-Dev`。login 側の `ascii_downcase` を外す mutation(P2)で red。origin の URL の側の owner が大文字のとき: テストがない(P4、Test gaps の 1) | 一部 |
| null の owner | fixture の `"headRepositoryOwner": null`(3 つの fixture にある)。`// ""` を外す mutation(P3)で jq が exit 5 になり red | PASS |
| jq がない(SKIP) | テストの中に分岐はあるが、それを通す自動テストはない。手で 2 回確かめた。macOS で jq を除いた PATH(`/bin` と `/usr/bin` の symlink から jq だけを抜いた)では 15 件 PASS、`SKIP: jq not on PATH`、rc 0。ubuntu:24.04 で jq を入れる前にも同じ結果。EXIT trap の `_tmp` が空でも rc 0 で終わった | 手で確認(Test gaps の 4) |
| `cycle-count.json` がない | `test-insights-append.sh` の 8b2(verify の指摘を受けて 25c45213 で追加) | PASS |
| `active-plan.json` がない | 8b | PASS |
| `plan_path` の不一致 | 8c。比較を外す mutation(M1)と、`cycle-count.json` 側の plan_path を読まない mutation(M11)で red | PASS |
| `cycle-count.json` が壊れている | 8d(exit 0)と 8e(cycle 1) | PASS |
| `cycle` が 0 / 文字列 | 8f / 8g。`>= 1` を外す mutation(M3)で red | PASS |
| 数値の `--cycle 3` は状態を読まない | 8h | PASS |
| `--state-dir` を省いたときの既定 | 8i(git のサブディレクトリから repo の root を見つける)、8j(git の外では今のディレクトリ)。`git rev-parse` を `pwd` にする mutation(M6)で 8i が red | PASS |

AC4 の「`--cycle auto` を外すと落ちる」は、verify では grep の条件を読んだだけだった。今回は実際に外して確かめた。4 面のうち 1 面の 1 skill から外す(K1)、`autox` に変える(K2)、`--cycle 1` に変える(K3)、self-review のコードブロックからだけ外して後ろの説明文には残す(K5)の 4 通りとも、該当の 1 件だけが FAIL になった。K5 で、テストがコードブロックの外の `--cycle auto` に惑わされないことも分かった。

AC1 の snapshot と template の一致は、`TestSettingsSnapshotTemplate_MatchesSettingsJSON` が PASS した。`templates/base/.ralph/core/settings.ralph.json` からだけ `Bash(sed:*)` を消すと、このテストが `has drifted` で FAIL になる(A1)。

## Mutation

`git archive HEAD` で scratchpad に写した木の中で、perl で 1 か所ずつ書き換えて該当の suite を流し、`git checkout` で戻した。worktree の追跡ファイルには触れていない。`/pr` 5.c の式の mutation は 4 面すべてに同じ書き換えをした(4 面の一致の検査で先に落ちないようにするため。P8 だけは 1 面に限った)。

| ID | 書き換え | 結果 |
| --- | --- | --- |
| M1 | `resolve_auto_cycle` の `plan_path` の比較を外す | red(8c) |
| M2 | `[ -z "$_rac_active" ]` を外す | 残った |
| M3 | `state_cycle` の `>= 1` を外す | red(8f) |
| M4 | `state_cycle` の整数の検査(`== floor`)を外す | 残った |
| M5 | `jq -rs` を `jq -r` にする | red(8a、8i、8j) |
| M6 | 既定の state dir を `git rev-parse` ではなく `pwd` から作る | red(8i) |
| M7 | `auto` を解決しない | red(8a、8b、8b2 ほか) |
| M8 | fallback を 1 ではなく 0 にする | red(8b、8b2、8c) |
| M9 | `--state-dir` を無視する | red(8a) |
| M10 | `state_plan_path` の型の検査(`type == "string"`)を外す | 残った |
| M11 | `cycle-count.json` の `plan_path` を読まず、`active-plan.json` の値を使う | red(8c) |
| K1〜K3、K5 | 上の節のとおり | 4 件とも red |
| K4 | self-review の `--cycle auto --source skill` を消す | 適用できなかった(self-review のブロックは行を折り返していて、文字列が一致しない)。K5 で置き換えた |
| P1 | `sed` の `s#/+$##` を外す | red(末尾 `/` の URL) |
| P2 | login 側の `ascii_downcase` を外す | red(2 件) |
| P3 | `// ""` を外す | red(jq exit 5、3 件) |
| P4 | `<owner>` 側の `ascii_downcase` を外す | 残った |
| P5 | 2 回目の検索の出力を `.url` だけにする | red |
| P6 | owner の `select` を外す | red(`"<owner>"` の目印の検査) |
| P7 | 1 回目の検索から `--base <base>` を外す | red(式の数の検査) |
| P8 | `.claude/skills/pr/SKILL.md` の 1 面だけ `sed` を変える | red(残りの 3 面の一致の検査) |
| P9 | 「何か出たら作らずに止めて報告する」の文を消す | 残った |
| P10 | 「gh が『すでにある』と断ったら止めて報告する」の行を消す | 残った |
| P11 | `sed` の `[:/]` を `/` にする(`git@host:` の形が取れなくなる) | red(2 件) |
| A1 | `settings.ralph.json` からだけ `Bash(sed:*)` を消す | red(`go test ./internal/upgrade/`) |
| A2 | 3 つの settings から 2 つの許可をそろって消す | 残った |

残った 7 件の扱い。

- M2 と M10 は同じ場面で差が出る。2 つの状態ファイルがどちらも `plan_path` を持たず、`cycle-count.json` に `"cycle": 2` があるとき、元のスクリプトは cycle 1 を書き、どちらの mutation も cycle 2 を書いた(scratchpad で実行して確かめた)。M2 は `active-plan.json` がなく `cycle-count.json` に `plan_path` がないときも cycle 2 を書いた。スクリプトの冒頭のコメントは「同じ空でない文字列の plan_path」と書いているので、テストを 1 件足せば両方を見分けられる(Test gaps の 2)
- M4 は `"cycle": 2.5` のときに差が出る(元は 1、mutation は 2)。`/cross-review` は整数しか書かないので、起きる見込みは低い
- P4 は origin の URL の owner が大文字を含むときに差が出る。`sed` は大文字小文字をそのまま残す(`https://github.com/Yoshpy-Dev/ralph/` から `Yoshpy-Dev` が出ることを確かめた)。owner に `Yoshpy-Dev` を入れて小文字の login と比べると、元の式は PR を見つけ、mutation は見つけなかった。テストは owner に小文字の `yoshpy-dev` しか入れないので見分けられない。この mutation が入っても、2 回目の検索も同じく見つけず、作り直しで gh が「すでにある」と断って止まるので、PR が 2 つになることはない
- P9 と P10 は 5.c の手順の文で、テストは式だけを取り出して実行する作りなので見分けられない。plan の設計(式を SKILL.md から取り出して実行する)の範囲外にあたる
- A2 は、2 つの許可があることを確かめるテストがないことを示す。snapshot のテストと `check-sync.sh` は 3 ファイルが同じことだけを見る。2 つの許可が 1 回ずつあることは verify が `jq` で確かめた

## Linux(ubuntu:24.04)での実行

`git archive HEAD` の写しを tar で標準入力からコンテナに渡した(このマシンの Docker は scratchpad のパスを mount できなかった)。

- 環境: `/bin/sh` は dash、awk は mawk、GNU sed 4.9、GNU coreutils 9.4。jq は最初は入っておらず、途中で `apt-get install jq git` で jq 1.7 と git 2.43.0 を入れた
- jq を入れる前の `sh tests/test-pr-owner-lookup.sh`: 15 件 PASS、`SKIP: jq not on PATH`、rc 0
- jq を入れたあと: `bash tests/test-insights-append.sh` 50 / 50、`sh tests/test-skill-insight-cycle.sh` 16 / 16(mawk で `insight_block` の awk が動いた)、`sh tests/test-pr-owner-lookup.sh` 19 / 19。どれも rc 0

## 規則の `<record>` の確かめ方

`post-implementation-pipeline.md` の「only appends to the triage report」の後ろの最初のコードブロックを awk でそのまま取り出し、`<record>` を `HEAD`、`<triage>` を `docs/reports/cross-review-triage-demo.md` に置き換えて、使い捨ての git リポジトリで実行した。root と template の rules は `cmp` で同じ。git の設定は `HOME`、`GIT_CONFIG_GLOBAL`、`GIT_CONFIG_SYSTEM` を切り離した。

| `<record>` の中身 | 期待 | macOS(`/bin/sh`、`/bin/bash`、`/bin/zsh`、BSD coreutils) | ubuntu(dash、bash、GNU coreutils) |
| --- | --- | --- | --- |
| 末尾に節を追記 | rc 0 | rc 0 | rc 0 |
| `After triage:` の行の上に別の `After triage:` の行を挿入 | rc ≠ 0 | rc 1 | rc 1 |
| `After triage:` の行の数をその場で書き換え | rc ≠ 0 | rc 1 | rc 1 |
| 改行で終わらない旧ファイルの最後の行を延長 | rc ≠ 0 | rc 1 | rc 1 |
| triage レポートを `<record>` で初めて作る | rc ≠ 0 | rc 128(`git cat-file -e` が弾く) | rc 128 |

どのケースでも実行後の `git status --porcelain` は 0 行だった。ubuntu では `TMPDIR` を空のディレクトリにして流し、終わったあとにそこに残ったディレクトリは 0 個だった(GNU の `mktemp -d` は引数なしでも `TMPDIR` に従うので、ブロックの `rm -rf "$d"` が効いたことが分かる。macOS の `mktemp -d` は引数なしだと `TMPDIR` を無視するので、macOS ではこの数え方をしていない)。verify が未確認として残した「GNU の `wc -c` と `head -c` での実行」は、結果が BSD と同じだった。

おまけの確認: self-review の F-2(`Bash(sed:*)` は sed の `w` と GNU sed の `e` も許す)を ubuntu の GNU sed 4.9 で確かめた。`sed "1e echo ..."` はシェルのコマンドを実行し、`sed -n "w <file>"` はファイルを書いた。PR 本文に書く予定の内容と合っている。

## Coverage

- Statement: shell には計測の道具がない。Go は `go test ./... -count=1 -cover` で `internal/cli` 84.7%、`internal/config` 92.3%、`internal/insights` 86.1%、`internal/org` 90.8%、`internal/org/driver` 92.0%、`internal/org/protocol` 97.9%、`internal/scaffold` 75.7%、`internal/upgrade` 91.2%。2026-10-04 と 2026-10-05 の数値と同じで、このブランチは Go のファイルを変えていない
- Branch: 計測していない
- Function: 計測していない
- Notes: shell の範囲は、上の edge case の表と mutation の結果で示した。`insights-append.sh --cycle auto` の分岐は、M2・M4・M10 の 3 件を除いて mutation で見分けられた

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | - | - | - |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| cycle 2 以降の insight event が cycle 1 で記録される(plan の Objective 3) | 直った | 8a(状態がそろえば cycle 2)、K1〜K3・K5(4 skill × 4 面の `--cycle auto` を外すと FAIL) |
| `--cycle` を省いたときは 1、数値の `--cycle` はそのまま | 変わらず | 7c、8h |
| origin の URL の末尾が `/` のとき owner が取れない(plan の Objective 4) | 直った | `https://github.com/yoshpy-dev/ralph/` のケース、P1 |
| owner の大文字小文字の違い、null の owner で lookup が誤る・落ちる | 直った(URL 側の大文字は Test gaps の 1) | P2、P3 |
| #203 の 5.c が扱っていた URL の形(https、`ssh://`、ポートつき、`git@host:`、SSH の別名ホスト) | 変わらず | URL 7 形のケース、P11 |
| `settings.ralph.json` と template の `settings.json` の byte 一致 | 保たれている | `TestSettingsSnapshotTemplate_MatchesSettingsJSON` PASS、A1 |
| 既存の shell 35 ファイルと Go 8 パッケージ | 変わらず | 件数は #203 の test レポートと同じ(`test-insights-append.sh` の追加分を除く)。Go の coverage も同じ |

## Test gaps

コードは足していない。小さい順に、足すなら次の形になる。

1. origin の URL の owner が大文字を含むときの jq の比較(P4)。`tests/test-pr-owner-lookup.sh` の `run_jq` の 1 件で、`"<owner>"` を `"Yoshpy-Dev"` に置き換えて小文字の login の fixture と比べれば見分けられる。plan の Assumptions は `https://github.com/Yoshpy-Dev/ralph/` を手で確かめたと書いているが、テストには入っていない
2. 2 つの状態ファイルがどちらも `plan_path` を持たないとき(M2、M10)。`write_state "${TMP8}/no-plan/state" '{"x": 1}' '{"cycle": 2}'` と、cycle 1 を期待する assert の 1 件で両方を見分けられる
3. 小数の `cycle`(M4)。`/cross-review` は整数しか書かないので優先度は低い
4. jq がない環境での `test-pr-owner-lookup.sh` の SKIP の経路。CI(ubuntu-latest)には jq があるので、この分岐は CI では通らない。今回は手で 2 回確かめた。なお `insights-append.sh` 自体は以前から jq がないと動かない
5. `/pr` 5.c の手順の文(作らずに止める、「すでにある」で止める。P9、P10)と、gh の実際の応答。実際の PR を作らないと確かめられず、/pr の時点で起きたときに分かる
6. 2 つの許可が settings にあること(A2)。3 ファイルの一致だけがテストで縛られている
7. 規則の `<record>` の確かめ方には自動テストがない。今回と verify の手での実行(BSD と GNU、計 5 シェル)だけが根拠になる

## Verdict

- Pass: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(shell 37 ファイル 1,694 件、Go 8 パッケージ、rc 0)、`go test ./... -count=1`(8 / 8)、`go test ./internal/upgrade/... ./internal/cli/... -count=1`(top-level 573 件)、指定の 3 本(50、16、19 件)、ubuntu:24.04 での 3 本と jq なしの SKIP、規則の `<record>` の確かめ方 25 通り。plan の edge case は「origin の URL 側の大文字の owner」を除いてテストがあり、jq がない場合は手で確かめた
- Fail: なし
- Blocked: なし

テストは通っており、/pr に進めない理由はない。Test gaps の 1 と 2 は 1 件ずつで足せるので、/sync-docs や次の修正の機会に足すかを判断してほしい。
