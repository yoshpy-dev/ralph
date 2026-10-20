# Walkthrough: secret-scan-test-tmp(9 段目の trial の前段、D11)

- Date: 2026-10-20
- Plan: docs/plans/active/2026-10-20-secret-scan-test-tmp.md(/pr で archive に移す)
- Branch: fix/secret-scan-test-tmp(base e333200c。この walkthrough の前で 10 commits、10 files、+528/-10)
- 読む範囲: `docs/reports/`、`docs/evidence/`、`docs/insights/` と計画を除くと 2 files になる
  - `tests/test-secret-scan.sh`(+13/-8)
  - `docs/tech-debt/README.md`(+4/-2)
- 差分の大半は後工程の記録で、本体の変更は小さい。後工程は 1 回でまわし終えた。self-review・verify・test の指摘は同じ回の中で直し、cross-review は 0 件だった

## 1. なぜ直したか

- 9 段目の trial では 2 つの feature org の gate が同時に `run-test.sh` を打つ。`run-test.sh` は `tests/test-*.sh` をすべて走らせるので、2 本の `test-secret-scan.sh` が重なりうる
- base の script は、各 `expect_exit` の stdout と stderr を固定の `/tmp/ralph-secret-scan-test.out` と `.err` に書き、stderr の check もそのファイルを読み、`trap` で消していた。2 本が重なると、互いのファイルを上書きし、先に終わった方が消す
- tech-debt の 170 行は、2 本を同時に走らせて両方が 2 件ずつ落ちた記録(2026-10-07)を持っていた

## 2. 本体の変更(ece77f30。コメントは 90fc10d1)

- `capdir="$(mktemp -d "${TMPDIR:-/tmp}/ralph-secret-scan-test.XXXXXX")"` を `workdir` の `mktemp` の前に置いた
- `capdir` は `$workdir` の外に置いた(計画の D1)。`$workdir` は check が検査する fixture の repository で、中にファイルを書くと untracked のファイルとして現れ、check の見るものが変わりうる
- 固定パスを使っていた 8 行を `$capdir/out` と `$capdir/err` に置き換えた
  - `expect_exit` の redirect と、失敗時に出す `cat` の 2 行
  - `expect_stderr_contains` と `expect_stderr_not_contains` の grep と `cat`
  - `trap`
- `trap 'rm -rf "$capdir" "${workdir:-}"' EXIT HUP INT TERM`。`trap` が `workdir` より先にあるので、`set -u` のもとで `${workdir:-}` と書く
- check の中身(assertion)は変えていない

## 3. 記録の変更

- tech-debt の 170 行を閉じた(ece77f30。旧版の走らせ方の言い方は 74598215 で直した)
- org-stop-all の test の穴の行の (e) も閉じた(90fc10d1、self-review の M1)。同じ欠陥の 2 つめの記録で、計画の Scope は 170 行だけを挙げていた。理由は計画の Progress checklist にある
- test の G1 を 1 行足した(82fc0eb3)。固定パスに戻す変更を止める check が suite になく、AC4 は suite の外の harness だけが確かめた
- evidence の旧版の取り出し元を `HEAD` から e333200c に直し、TERM の行は「`trap` が消した」ことを示す形に直した(90fc10d1、self-review の L1 と L2)

## 4. 確かめたこと

- 1 本だけ: exit 0、123 件通過、失敗 0(直す前と同じ数)
- 2 本同時: 実装のときに 1 回、test で 3 回。どれも両方 exit 0 で `FAIL: 0`
- 旧版の対照: 2 本目を 0.2 秒遅らせて起動すると、あとの方が `FAIL: 2` で落ちた(実装のときに 1 回、test で 2 回)
- harness の自己確認: 片方を `sh -c 'exit 3'` にすると rd=3 を拾う
- TERM: 4 回とも、終わったあと scratch の `TMPDIR` は空。終了コードは 128 か 1。`trap` が `exit` しないので(base と同じ形)、script は消えた directory で次のコマンドが失敗するまで進む
- `GOFLAGS=-timeout=30m ./scripts/run-test.sh` は rc 0、`./scripts/run-static-verify.sh` も rc 0

## 5. 確かめていないこと

- Linux での 2 本同時の実行(1 本だけの実行は CI が `ubuntu-latest` で走らせる)
- 3 本以上の同時の実行
- HUP と INT の `trap`(非対話の sh の background の job は SIGINT を無視するので、この harness では INT を送れない)
