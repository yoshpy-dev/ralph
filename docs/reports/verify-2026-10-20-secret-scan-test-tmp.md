# Verify report: secret-scan-test-tmp

- Date: 2026-10-20
- Plan: `docs/plans/active/2026-10-20-secret-scan-test-tmp.md`(tech-debt 170 行の解消。Status: Approved、`plan-visual.sh digest` は `3ee6f70482f6` で `- Approved:` 行と一致)
- Verifier: verifier subagent(Claude Code、pipeline cycle 1)
- Scope: `git diff e333200c...HEAD`(HEAD 90fc10d1、6 ファイル。コードは `tests/test-secret-scan.sh` だけ)。AC1〜AC5、計画の Verify plan、D1、R1・R2、Progress の注(self-review の M1 で 178 行の項目 (e) も閉じた範囲の広げ)。self-review の報告 `docs/reports/self-review-2026-10-20-secret-scan-test-tmp.md` と evidence `docs/evidence/secret-scan-test-tmp-2026-10-20.md`
- Evidence: `docs/evidence/verify-2026-10-20-secret-scan-test-tmp.log` と、run-static-verify.sh が書いた `docs/evidence/verify-2026-10-20-014911.log`(どちらも `docs/evidence/*.log` の gitignore でコミットしない生の出力)

## Spec compliance

テストの実行は `/test` の範囲なので、ここでは script と register を読み、AC3・AC4 は evidence の記載だけを確かめた。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: `/tmp/ralph-secret-scan-test.out` と `.err` を名指す行がなく、`expect_exit`・`expect_stderr_contains`・`expect_stderr_not_contains` ほかの読み手が実行ごとの capture dir を使う | 満たす | `grep -n 'ralph-secret-scan-test' tests/test-secret-scan.sh` は 44 行の `mktemp -d` の prefix だけ。`grep -n '/tmp'` は 44・46 行の `${TMPDIR:-/tmp}` と、signal の check の `$sig_dir/tmp`(899〜922 行、base の 894〜917 行と同じ)だけ。base の固定パス 8 行(26・34・36・41・116・120・127・129 行)は、HEAD の `"$capdir/out"`・`"$capdir/err"` の 7 行(26・34・36・121・125・132・134 行)と trap に 1 対 1 で置き換わっている。capture を読む箇所はこの 7 行のほかにない |
| AC2: capture dir が `$workdir` の中になく、EXIT・HUP・INT・TERM の `trap` が消す | 満たす | 44 行 `capdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-secret-scan-test.XXXXXX")"`、45 行 `trap 'rm -rf "$capdir" "${workdir:-}"' EXIT HUP INT TERM`、46 行で `workdir` を別の `mktemp -d` で作る。2 つは同じ親の下の兄弟で、入れ子にならない。`capdir=` と `workdir=` の代入はこの 1 回ずつで、script の中で `capdir` を消す・置き換える箇所はない。trap を `workdir` の作成前に置いたので、46 行の `mktemp` が失敗しても trap が `capdir` を消す。scratchpad の小さな sh で確かめた(同じ形の `set -eu` と trap、2 つ目の作成を失敗させると dir は残らず、macOS の `rm -rf ""` は rc 0 で何も出さない) |
| AC3: 単独で走らせて、変更前と同じ数で `FAIL: 0` | evidence に記載あり(実行は `/test`) | evidence 11〜12 行: 変更前も変更後も exit 0、`PASS: 123`、`FAIL: 0` |
| AC4: 2 つ同時に走らせて、両方 exit 0 と `FAIL: 0` | evidence に記載あり(実行は `/test`) | evidence 13 行: 計画の Test plan と同じ書き方(PID を取り、`wait` を別々に)で `ra=0 rb=0`、両方 `PASS: 123`、`FAIL: 0`。14 行で、その書き方が失敗した方の終了コード(3)を拾うことも記録している。変更前の script は 15 行で、片方が `FAIL: 2` |
| AC5: register の 170 行を、register の書き方で解消済みにする | 満たす | HEAD の 170 行が `<!-- RESOLVED 2026-10-20 in fix/secret-scan-test-tmp: … -->`(`-->` で閉じる)、171 行(元の 170 行)が Debt 欄の `~~…~~ (RESOLVED 2026-10-20 in fix/secret-scan-test-tmp)` と Trigger 欄の打ち消しと `— RESOLVED:`。self-review の M1 で 178 行(org-stop-all の Test gaps)の項目 (e) も Debt・Impact・Trigger の 3 欄で打ち消され、Trigger は既存の 181・185 行と同じ `~~(…)~~ (done in …)` の形。`\|` 以外の `|` で分けると、見出し 14 行・171 行・178 行はどれも 7 欄(5 列)。2 行が指す計画は active の道で、`scripts/archive-plan.sh` が `/pr` のときに `docs/tech-debt/README.md` の `docs/plans/active/<name>` を書き換える |

計画のほかの項目:

- D1(capture は `$workdir` の外、`$workdir` より前に同じ `${TMPDIR:-/tmp}` の書き方で作る): 44〜46 行で守られている
- R2(作る順が変わっても signal の check は自分の `sig_dir` を使う): 898 行 `sig_dir="$workdir/signal"`、901 行で scanner に `TMPDIR="$sig_dir/tmp"` を渡す。順には依存していない
- Non-goals(ほかの test は変えない、`templates/base/` に写しはない): diff の 6 ファイルにほかの test はない。`templates/base/tests/` はない
- I1(`${workdir:-}` のコメント): 40〜43 行のコメントが、trap が `workdir` の前にあることと `set -u` での `:-` の理由を書いている

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m ./scripts/run-static-verify.sh`(HEAD 90fc10d1) | exit 0、`All verifiers passed.` | shellcheck(`.claude/hooks`・`templates/base/.claude/hooks`・`scripts/*.sh`・`tests/test-*.sh` を `--severity=warning` で)、hook の `sh -n`、guard の awk の構文、settings.json の jq、check-sync(DRIFTED 0)、check-pipeline-sync、check-skill-sync(14 skill)、check-template-purity、tech-debt README の計画の参照が OK。`tests/test-secret-scan.sh` が言語の分類に入らないため full に落ち、golang の gofmt ok と golangci-lint 0 issues。branch の secret scan は `e333200c..90fc10d1` で clean |
| `sh -n` / `dash -n` / `bash -n tests/test-secret-scan.sh` | 3 つとも rc 0 | |
| `shellcheck tests/test-secret-scan.sh`(0.11.0、既定の severity) | rc 0、指摘なし | base の script も rc 0。run-static-verify の warning より低い style の指摘も出ていない |
| `git diff --check e333200c...HEAD` | rc 0 | |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `docs/tech-debt/README.md` 170・171 行 | ほぼ合う(V1) | 解消の文は evidence と合う(単独 123/0、2 つ同時に両方 0、TERM のあと dir なし)。ただし「the old script, started the same way」は evidence 15 行の起動方法と違う(V1) |
| `docs/tech-debt/README.md` 178 行の項目 (e) | 合う | 3 欄で閉じ、(a)〜(d) は開いたまま。「the same defect as the resolved … row above」の指す行(171 行)は上にある |
| 計画の Scope・Affected areas・AC5 | 合う(V2) | 3 か所は「row 170」だけを挙げる。178 行 (e) を閉じた範囲の広げは Progress の注(153〜161 行)に理由とともに記録されている |
| `grep -rln 'ralph-secret-scan-test'`(.git を除く全ファイル) | 合う | 当たるのは 8 ファイル。いまの動きとして固定パスを書くものはない。`tests/test-secret-scan.sh` は新しい prefix、register の 2 行は打ち消し済み、evidence は新しい prefix の dir 名、計画は変更前の説明、`docs/reports/` の 4 件はその時点の記録 |
| ほかの test・script の参照 | 合う | `tests/test-secret-scan-branch.sh` 248 行は git の版の読み方の話で、capture のパスには触れない。`scripts/`・`tests/`・`.github`・`.claude/hooks` に `ralph-secret-scan*` の glob で消す・数える箇所はない |

## Observational checks

- evidence の Notes(21〜33 行)は self-review の L2 の指摘どおりに直っている。TERM の行は「trap が両方の dir を消した」ことを示し、終了コードは TERM が入る場所で 128 と 1 になると書く。trap に `exit` がないのは base の 41 行と同じ形で、この PR が入れたものではない
- evidence の「The script makes no other directory with `mktemp`」は、HEAD の `mktemp` が 44・46 行の 2 つだけで、ほかの fixture(`range_dir`・`.home`・`fake-git-bin`・`file-cases`・`signal` ほか)がすべて `$workdir` の下に作られることと合う
- L1 の直し(旧版を `git show e333200c:` で取る)は evidence 15 行に入っている

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | LOW | register 170 行の RESOLVED コメントが、旧版の 2 つ同時の run を「the old script, started the same way」と書く。evidence 15 行では、旧版は scratch の `tests/` に写して `scripts/` への symlink の横から走らせ、2 本目を 0.2 秒遅らせて起動している。AC4 の書き方(遅らせない)と同じではない。結果(片方が `FAIL: 2`)の記述は合っている | `docs/tech-debt/README.md` 170 行、`docs/evidence/secret-scan-test-tmp-2026-10-20.md` 13・15 行 | `started the same way` を `started 0.2 s apart` などに直す。`/sync-docs` で 1 か所直せる。直さなくても AC5 の可否は変わらない |
| V2 | INFO | 計画の Scope・Affected areas・AC5 は「row 170」だけで、178 行 (e) を閉じたことは Progress の注にだけある | 計画 26〜29 行、46〜47 行、77〜78 行、153〜161 行 | 記録の場所として足りている。計画の本文は変えなくてよい |
| V3 | INFO | register の 3 行が 7 欄でない(HEAD の 22 行が 8 欄、32 行が 6 欄、278 行が 11 欄)。base でも 22・32・277 行で同じ数で、この PR は増やしていない | 生ログの `## AC5` | この PR の範囲の外 |
| V4 | INFO | `TMPDIR` が相対パスのとき、`capdir` も相対になり、`cd "$workdir"` のあとの `"$capdir/out"` は別の場所を指す。base も `range_dir="$workdir/range"` の `mkdir`(`cd "$workdir"` のあと)で絶対パスの `TMPDIR` を前提にしているので、回帰ではないとみている。読んだだけで、走らせて確かめてはいない | `tests/test-secret-scan.sh` 44〜48 行、75〜77 行 | 対応は要らない。相対の `TMPDIR` を支えるなら別の変更 |

## Coverage gaps

- AC3・AC4 と TERM の edge case は evidence の記載を読んだだけで、この verify では走らせていない(`/test` の担当)
- V4 は静的な読みだけで、相対の `TMPDIR` で script を走らせてはいない
- Linux(GNU `mktemp`・`rm`)での trap と `rm -rf ""` の動きは確かめていない。macOS の `rm -rf ""` が rc 0 で黙ることは確かめた
- 決定的な verifier は足りている。固定の `/tmp/<name>` パスを `tests/*.sh` で禁じる grep の check は足せるが、計画の Non-goals の調査で使われているほかの固定パスはなく、今回は提案に留める

## Verdict

- Verdict: pass(CRITICAL・HIGH・MEDIUM はない。LOW 1 件(V1)は register の文 1 か所で、`/sync-docs` で直せる。INFO 3 件(V2〜V4)は対応不要)
- Verified: AC1・AC2・AC5 のコードと register の突き合わせ。D1、R2、Non-goals、I1 のコメント。self-review の M1・L1・L2・I1 の直しが HEAD に入っていること。静的解析(run-static-verify exit 0、`sh -n`・`dash -n`・`bash -n`、shellcheck の既定 severity、`git diff --check`)。固定パスの文書のずれ(全ファイルの grep)。計画の digest
- Partially verified: AC3・AC4(evidence に記載があることだけ)
- Not verified: テストの実行(単独・2 つ同時・TERM)、Linux での動き、相対の `TMPDIR`(V4)
