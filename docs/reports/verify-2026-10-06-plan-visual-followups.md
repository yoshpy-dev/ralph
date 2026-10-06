# Verify report: plan-visual-followups

- Date: 2026-10-06
- Plan: docs/plans/active/2026-10-06-plan-visual-followups.md
- Verifier: verifier subagent (Claude)、cycle 1(`cycle-count.json` は 1)
- Scope: 仕様への適合(AC1〜AC6)、静的解析、文書のずれ。対象は `git diff f9baf6a7...HEAD`(base origin/main f9baf6a7、HEAD 62a83664、36 ファイル、+1053/-89)。self-review の修正コミット c24eab98(F-1、F-3、F-5、F-6)、3a426bb8(R-1)、38f6931a(R-2、R-3)を含む。テスト(`./scripts/run-test.sh`、`go test`、`tests/test-*.sh`)は /test の担当なので実行していない
- Evidence: `docs/evidence/verify-2026-10-06-plan-visual-followups.log`(`docs/evidence/*.log` は gitignore 対象なので、手元にだけ残る)

## Spec compliance

plan の承認: `./scripts/plan-visual.sh digest` は `eb13fda57381` を返し、plan の `- Approved: 2026-10-06 sha256:eb13fda57381` と一致した。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 3 つの settings に 2 つの許可が 1 回ずつ、3 ファイルが同じ、JSON として読める、`go test ./internal/upgrade/...` と `check-sync.sh` | Met(`go test` は /test) | 3 ファイルとも `jq -e .` が通り、`Bash(git remote get-url:*)` と `Bash(sed:*)` はそれぞれ 1 回。並びは `Bash(git config:*)` の直後(index 15→16)と `Bash(sort:*)` の直後(31→32)で、plan の指定どおり。`cmp` で root と template、template と `settings.ralph.json` が一致した。`internal/upgrade/snapshot_test.go:50` が確かめるのはこの byte 一致なので、`cmp` で同じ条件を見たことになる。`check-sync.sh` は DRIFTED 0。repo の中に許可リストの写しはほかにない(`git grep 'Bash(sort:\*)'` は 3 ファイルだけ) |
| AC2: 例外の節(3 条件、確かめ方、記録、`cycle-count.json` を上げない、外れたら全工程)と `ralph-workflow.md` の一文 | Met(plan の文言より厳しい。V-2) | `.claude/rules/ralph/post-implementation-pipeline.md:13-67`。条件 1 は `:17`、条件 2 は `:20-27`(plan の slug で名付けた `docs/reports/` 直下のレポートと `docs/insights/events/<date>-<slug>.jsonl`、`docs/reports/templates/` は対象外)、条件 3 は `:28-31`(self-review の重大度と `Merge:` の行、verify と test の `## Verdict`)。確かめ方は `:33-39`(`git diff --name-only`、`git show --stat`、verdict の行の差分)。PR 本文への記録、`cycle-count.json` を上げないこと、外れたら全工程に戻ることは `:58-63`。条件 3 が挙げる行は、`/pr` の pre-check 1〜3(`.claude/skills/pr/SKILL.md:11-13`)が読む verdict と同じ。`ralph-workflow.md:46-50` に Edit / Write を使う一文がある。root と template は `cmp` で一致 |
| AC3: `/cross-review` step 8 の Case A・B(cap の前後とも)の選択肢と step 9 の扱い。条件は rules を指す | Met | `.claude/skills/cross-review/SKILL.md:126` が選択肢を出す条件を rules の節名で指す。選択肢は Case A の cap 前 `:133`、cap 後 `:140`、Case B の cap 前 `:149`、cap 後 `:155` の 4 か所。step 9 は `:166` で、修正、確かめ、triage レポートへの記録、`cycle-count.json` を上げない、`/pr` へ、外れたら全工程、を書いている。条件の中身は書き写さず rules に委ねている。`:166` の「修正のコミットは triage レポートに触れず、結果は追記だけの別コミットで書く」は手順の要約で、条件の二重化ではないと判断した。Codex の番号入力も `digit 1–4` に直っている(`:175`) |
| AC4: `--cycle auto` の解決規則、テスト 4 通り、4 skill × 4 面、root と template が同じ | Met(テストの実行は /test) | `scripts/insights-append.sh:153-186` の `state_plan_path`・`state_cycle`・`resolve_auto_cycle` が、2 つの状態ファイルがそろい `plan_path` が一致するときだけ `cycle` を使い、それ以外は 1 にする。`/cross-review` step 1.a〜1.c(`.claude/skills/cross-review/SKILL.md:23-28`)の規則と同じ。`tests/test-insights-append.sh:342-355` に 8a(一致で 2)、8b(`active-plan.json` なしで 1)、8c(`plan_path` 違いで 1)、8d・8e(壊れた JSON で exit 0 と 1)がある。`tests/test-skill-insight-cycle.sh` は 4 skill × 4 面で、`## Insight event` 節の最初のコードブロックに `--cycle auto` がなければ FAIL にする(grep の条件を読んで確かめた。外した版での実行はしていない)。insight event を書く skill はこの 4 つだけ(`git grep insights-append` で確認)。root と template の `insights-append.sh` は `cmp` で一致 |
| AC5: `/pr` 5.c の `sed`・jq・2 回目の検索・止める条件と、`tests/test-pr-owner-lookup.sh` の URL 7 形と fixture | Met(テストの実行は /test) | `.claude/skills/pr/SKILL.md:42` の `sed -E 's#/+$##; s#...#\1#'` が末尾の `/` を落とす。`:41` と `:46` の jq は `(.headRepositoryOwner.login // "" \| ascii_downcase) == ("<owner>" \| ascii_downcase)` で比べる。`:45-50` に、base を限らない 2 回目の検索、何か出たら作らずに止める、gh が「すでにある」と断ったら止める、がある。テストは URL 7 形(`tests/test-pr-owner-lookup.sh:156-162`)と、大文字の login・null の owner・別の owner・別の base の fixture を持ち、SKILL.md から式を取り出して 4 面の一致も見る。jq がなければ jq の部分を SKIP する。5.c の範囲は 4 面とも md5 が同じ |
| AC6: `check-skill-sync.sh`、`check-sync.sh`、`check-pipeline-sync.sh`、`check-template-purity.sh`、`run-verify.sh` | Met(`run-verify.sh` は静的解析の部分だけ) | 4 本を単独で実行してすべて rc 0(13 skill が一致、DRIFTED 0、6 文書がパイプラインの全工程を参照、template に meta-repo の参照なし)。`./scripts/run-static-verify.sh`(`HARNESS_VERIFY_MODE=static` の `run-verify.sh`)も rc 0。既定の `all` モードのテスト部分は /test に回した |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | PASS(rc 0) | `verify.local.sh`(shellcheck、hook の `sh -n`、settings の `jq -e`、Codex の hook guard 3 本、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity)、golang verifier(`.claude/settings.json` が言語に分類されないため full にフォールバック。gofmt ok、lint 0 issues)、branch secret scan(`f9baf6a7..62a83664` clean)。runner 自身のログは `docs/evidence/verify-2026-10-06-092559.log` |
| `./scripts/check-skill-sync.sh` | PASS | 13 skill が一致 |
| `./scripts/check-sync.sh` | PASS | IDENTICAL 164、DRIFTED 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5 |
| `./scripts/check-pipeline-sync.sh` | PASS | 6 文書すべて |
| `bash scripts/check-template-purity.sh` | PASS | |
| `shellcheck --severity=warning`(新規・変更の 5 本) | PASS | `tests/test-pr-owner-lookup.sh`、`tests/test-skill-insight-cycle.sh`、`tests/test-insights-append.sh`、root と template の `insights-append.sh`。テストは `verify.local.sh` の `tests/test-*.sh` で gate に入っている。`insights-append.sh` は gate の対象一覧にない(tech-debt の行 (a) に記録済み)ので単独で回した。既定の重さでも rc 0 |
| `sh -n`(新しいテスト 2 本と `insights-append.sh`) | PASS | |
| `git diff --check f9baf6a7...HEAD` | PASS | 空 |
| 追加行の U+FFFD 検索 | PASS | なし |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/rules/ralph/post-implementation-pipeline.md`(全体) | 一部ずれ(V-4) | 冒頭(`:11`)と cap の節(`:112`)は例外に触れている。`:112` の「(1) raise the cap…、(2) proceed to `/pr`…、(3) abort」の番号は、skill の cap 後の選択肢(2 が Fix records only、3 が記録して PR、4 が Abort)と合わなくなった。「Re-run after cross-review ACTION_REQUIRED fix」の節(`:97-106`)は例外に触れないが、冒頭の文が例外を断っているので矛盾ではない |
| `docs/quality/definition-of-done.md` | 一部ずれ(V-3、/sync-docs で判断) | `:30` は「修正を選べば全工程を回し直す」とだけ書き、records-only の選択肢に触れていない。`:32` は cap の詳細を rules に委ねている |
| `README.md`(Quick start、Operating loop) | 一致 | 再実行と `/pr` の復旧の手順は書いていない |
| `AGENTS.md`(Primary loop) | 一致 | 同上 |
| `.claude/rules/ralph/subagent-policy.md` | 一致 | 段階の役割と順序だけで、再実行の規則は書いていない |
| `docs/insights/README.md` | 一部ずれ(V-5、/sync-docs で判断) | `:27` は「`--cycle` を省くと 1」とだけ書き、`--cycle auto` と `--state-dir` に触れていない。`:129` の例は `--cycle 1` |
| `docs/tech-debt/README.md` | 一部ずれ(V-6) | `:128` の「insight event の cycle スタンプ機構が脆弱」の行は、このブランチで 4 skill が `--cycle auto` を渡すようになったので解消に当たるが、解消の印がない。返済のきっかけ(次に post-implementation の skill 群を編集するタスク)もこのタスクに当たる。新しい行 `:156` は self-review の指摘どおり入っている |

## Observational checks

規則の `<record>` の確かめ方(`post-implementation-pipeline.md:50-51` のコードブロック)を、rules から awk でそのまま取り出し、`<record>` を `HEAD`、`<triage>` を `docs/reports/cross-review-triage-demo.md` に置き換えて、scratchpad の使い捨ての git リポジトリで実行した。実行したシェルは `/bin/sh`、`/bin/bash`、`/bin/zsh` の 3 つ。

| `<record>` の中身 | rc(3 シェルとも) | 実行後の `git status --porcelain` |
| --- | --- | --- |
| 末尾に節を追記 | 0 | 0 行 |
| `After triage:` の行の上に `ACTION_REQUIRED=0` の行を挿入 | 1 | 0 行 |
| 改行で終わらない旧ファイルの最後の行を延長 | 1 | 0 行 |
| `After triage:` の行をその場で書き換え(同じ長さ) | 1 | 0 行 |
| triage レポートを `<record>` で初めて作る(Bash tool の zsh で実行) | 0 | 0 行 |

依頼された 2 通り(追記で rc 0、`After triage:` の上への挿入で rc 1)は期待どおりだった。最後の行は V-1。

`--cycle auto` の補足: `cycle-count.json` だけがない状態で `scripts/insights-append.sh --cycle auto --state-dir <scratch>` を 1 回実行し、exit 0、`cycle` 1 を確かめた(plan の Test plan の Edge cases にあるが、テストのケースにはない。Coverage gaps を参照)。

## Findings

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| V-1 | LOW | spec(規則の確かめ方) | `<record>^` に triage レポートがないと、コードブロックは rc 0 を返す。`git show <record>^:<triage>` の失敗は捨てられ、空の `old` は何の先頭にも一致するため。`rc=0 means an append at the end`(`:54`)と食い違う。`:33` の「triage レポートを先にコミットする」を守れば起きない。macOS では `head -c 0` が `illegal byte count` を出すが、rc は変わらない | `post-implementation-pipeline.md:50-54`。上の probe の最後の行(`fatal: path ... not in 'HEAD^'` のあと rc=0) | 1 行目の先頭に `git cat-file -e <record>^:<triage> &&` を足すなど、旧ファイルがないときは rc を 0 以外にする。直さないなら既知のギャップとして PR 本文に書く |
| V-2 | LOW | plan との差 | plan の Scope(`:24`)、AC2(`:81`)、Design decisions(`:68`、`:72`)は、条件 2 を「slug を名前に含む `docs/reports/` 直下のレポート」と書き、triage レポートも許可パスに入る書き方になっている。実装は triage レポートを修正コミットから外し、確かめた結果を追記だけの別コミット `<record>` に書かせ、旧ファイルが新ファイルの byte 単位の先頭であることを確かめる(c24eab98、3a426bb8、38f6931a)。Progress checklist の S2 のメモは「plan の slug で名付けたファイル」への絞り込みだけを書き、triage レポートの除外と `<record>` コミットを書いていない。どちらも plan の範囲を狭める変更で、plan のリスク欄(例外が広く読まれる、verdict の書き換え)の意図に沿うので、AC2 と AC3 は満たしていると判断した | plan `:24`、`:68`、`:72`、`:81`、`:135`。rules `:20-27`、`:41-51` | Progress checklist に 1 行足す(F-1 と R-1〜R-3 の修正で triage レポートを外し `<record>` を足したこと)。Progress checklist は digest の対象外なので、承認は変わらない。plan の本文は /verify では直していない |
| V-3 | LOW | doc drift | `docs/quality/definition-of-done.md:30` が records-only の例外に触れていない | 上の Documentation drift の表 | /sync-docs で、rules の例外の節を指す半文を足すか、今のままでよいかを決める |
| V-4 | LOW | doc drift | rules `:112` の cap 後の選択肢の番号 (1)〜(3) が、skill の番号(1〜4、2 が Fix records only)と合わない。もとは同じ番号だった | `post-implementation-pipeline.md:112`、`cross-review/SKILL.md:138-142` | 番号を外して列挙にするか、4 つを skill と同じ番号で書く(root と template の 2 面) |
| V-5 | LOW | doc drift | `docs/insights/README.md:27` と `:129` の例が `--cycle auto` と `--state-dir` に触れていない | 同上 | /sync-docs で、`cycle` の行に「skill は `--cycle auto`」を足すかを決める |
| V-6 | LOW | tech debt | `docs/tech-debt/README.md:128` の行(insight event の cycle スタンプ機構が脆弱)が、このブランチで解消したのに印がない。行が挙げる `/sync-docs` に insight event がない点は、記録がないという別の話で、cycle の誤記ではない | `docs/tech-debt/README.md:128`。4 skill の `--cycle auto`(AC4) | 行を取り消し線にして「RESOLVED 2026-10-06 in fix/plan-visual-followups」を足す。`/sync-docs` に event がない点を残すなら、その部分だけを書いた行にする |

self-review の F-2(`Bash(sed:*)` の `w` と GNU sed の `e`)は PR 本文に回す扱いで、F-4 は判断で残す扱いのまま。どちらもこのブランチの文書では処理済みの扱いなので、ここでは数えない。

## Coverage gaps

- /test に回したもの: `go test ./internal/upgrade/...` と `./internal/cli/...`(AC1)、`tests/test-insights-append.sh`(AC4)、`tests/test-skill-insight-cycle.sh`(AC4)、`tests/test-pr-owner-lookup.sh`(AC5)、`./scripts/run-verify.sh` のテスト部分(AC6)。
- `tests/test-skill-insight-cycle.sh` が `--cycle auto` を外すと落ちることは、grep の条件を読んで確かめただけで、外した版での実行はしていない。
- plan の Test plan の Edge cases にある「`cycle-count.json` がないとき」は `tests/test-insights-append.sh` の Case 8 にない(8b は `active-plan.json` がない場合)。上の probe で cycle 1 になることは確かめた。一番小さい追加の検証は、`write_state "${TMP8}/no-count/state" "$PLAN_A" ""` と assert 1 行の 1 ケース。
- `<record>` の確かめ方は macOS の BSD coreutils でだけ実行した。GNU の `wc -c` と `head -c` での実行(CI と同じ ubuntu)はしていない。GNU の `head -c 0` はエラーにならないので、結果は変わらないはず。未確認です。
- F-2 の GNU sed の `e` コマンドは、手元の sed が BSD のため確かめていない(self-review と同じ)。
- `/pr` 5.c の gh の実際の挙動(`--head` と別の base の PR、「すでにある」の文言)は、実際の PR を作らないと確かめられない。/pr の時点で起きたときに分かる。

## Verdict

- Verdict: pass
- Verified: AC1〜AC6 の静的な部分(上の表)。plan の digest が承認の行と一致すること。静的解析の gate 全部。ミラーの一致(5 skill × 4 面、2 rules、`insights-append.sh`、3 settings)。規則の `<record>` の確かめ方を使い捨ての git リポジトリで 5 通り、3 シェルで実行した結果
- Partially verified: AC2(plan の文言より厳しい実装で、plan の本文が追いついていない。V-2)、文書のずれ(V-3〜V-6 は LOW で、/sync-docs か次の修正で扱える)
- Not verified: テストの実行(/test の担当)、GNU coreutils での確かめ方の実行、GNU sed の `e`、gh の実際の応答
