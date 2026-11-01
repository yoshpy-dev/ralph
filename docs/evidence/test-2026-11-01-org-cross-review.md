# Test evidence: org-cross-review(cycle 1)

先頭が run 2(F6 のあと)、続いて run 1。判定は `docs/reports/test-2026-11-01-org-cross-review.md` の `## Verdict`(run 2 のもの)。

## Run 2(HEAD 17fc2b52)

- HEAD: 17fc2b52(F6)。run 1 の HEAD 15da7eb0 からのコードの変更は F6 だけで、変えたのは `internal/org/prompts/leader.md`、`internal/org/prompts/reviewer.md`、`internal/org/prompts_xreview_leader_test.go`、`internal/org/prompts_xreview_reviewer_test.go`
- 全出力: `docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore)の 7,187 行目の `run 2 (HEAD 17fc2b52, after F6)` の区切りのあと。パッケージ全体の回、`-race` の回、変異ごとの差分と落ちた行、R1c の probe を順に入れた
- 範囲は handoff のとおり。`./scripts/run-test.sh` と `internal/cli` は打っていない(F6 が変えた文を固定するテストは `internal/org` の外にない)

### 実行

| コマンド | 結果 | 時間 |
| --- | --- | --- |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -count=1 ./internal/org/` | `ok  github.com/yoshpy-dev/ralph/internal/org  1096.376s` | 11:00:03〜11:18:20 JST。load 開始 { 6.29 7.17 7.10 }、終了 { 10.67 9.26 8.27 } |
| `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test -race -count=3 ./internal/org/ -run Xreview -v` | ok(488.760s)。名前 40 個 × 3 回でトップレベル 120、サブテスト 84 個 × 3 回で 252 が PASS。FAIL 0、SKIP 0、DATA RACE 0 | 11:03:41〜11:12:01。load { 6.32 7.23 7.19 } → { 8.39 8.01 7.59 } |

起動: パッケージ全体の回は perl の fork と setsid で切り離した(INT・HUP・QUIT・TERM を既定に戻し、標準入出力を /dev/null)。runner が log と meta を書き、前景の `caffeinate -w <pid>` で待った。`-race` の回は前景で並べて打った。変異は同じ形で切り離し、`xargs -P 2` で 2 個ずつ打った。時間切れと打ち直しはない。

### 変異

`go test -overlay <json> -count=1 ./internal/org/ -run Xreview -v`。写しは perl で 1 か所だけ置き換えた(一致が 1 か所でなければ止まる)。R1c〜R1f はテストの `prompts_xreview_reviewer_test.go` の定数 `reviewerXreviewBaseRule` も同じように置き換えた。変異の前後で `git status --porcelain` は空、`--ignored` は `.harness/cache/` を除いて同じ。

| ID | 置き換え | rc | 落ちたテスト(頭の `TestRenderRolePrompt_` は省いた) | 時間 |
| --- | --- | --- | --- | --- |
| L1a | leader.md:954-955 の行の説明 → 「手順 5 で implementer を spawn したときの `--driver` の値」 | 1 | `Leader_XreviewTaskCarriesTheReviewersLines` | 159 秒 |
| L1b | leader.md:978 → `: command ralph org status … --json; printf '%s\n' '{"seats":[{"role":"implementer","driver":"claude"}]}' > "$p-status.json"; rc=0` | 1 | `Leader_XreviewDriverCommandReadsTheRecord` の bash・zsh(14 行のうち 11 行が driver: claude を出した。status.err に stderr がない) | 161 秒 |
| L2a | leader.md:808 `grep -qE '^- Reviewer status: complete[[:space:]]*$'` → `grep -qxF -e '- Reviewer status: complete'` | 1 | `Leader_XreviewStateCommandCountsACompleteLineWithTrailingSpace` の bash・zsh(done 1、want 3) | 135 秒 |
| L2b | 同じ行の `[[:space:]]*$` → ` *$` | 1 | 同じ(done 2、want 3) | 145 秒 |
| L2c | 同じ行の `[[:space:]]*$'` → `[[:space:]]*'` | 1 | 同じ(done 4、want 3) | 125 秒 |
| L2d | leader.md:848 「(行の終わりの空白は問わない)」を消す | 1 | `Leader_XreviewCountsOnlyCompleteReports` | 129 秒 |
| L3a | leader.md:1055-1059 の送り直しの上限の文を消す | 1 | `Leader_XreviewHandlesEveryReply` | 112 秒 |
| L3b | 「2 回(`-r2` と `-r3`)まで。`-r3` の TASK」→「3 回(`-r2`、`-r3`、`-r4`)まで。`-r4` の TASK」 | 1 | `Leader_XreviewHandlesEveryReply` | 120 秒 |
| L4a | leader.md:979 jq の `\| unique` を外す | 1 | `…DriverCommandReadsTheRecord` の two implementers on claude(bash・zsh) | 118 秒 |
| L4b | `select(.role == "implementer")` → `select(.seat_id == "implementer")` | 1 | `…DriverCommandReadsTheRecord` の 2 行(bash・zsh)、`…DriverCommandReadsTheCLIsFields` | 118 秒 |
| L4c | leader.md:978 `2> "$p-status.err"` → `2> /dev/null` | 1 | `…DriverCommandReadsTheRecord` の status.err の確かめ(bash・zsh) | 150 秒 |
| R1a | reviewer.md:168 の許す文字に `` `{` `` を足す(雛形だけ) | 1 | `Reviewer_XreviewBaseTakesPlusAndAt`、`Reviewer_XreviewTaskCarriesItsLinesInTheBody` | 142 秒 |
| R1b | reviewer.md:169-171 の「(`{` を許さないので … revision の式になる)」を消す(雛形だけ) | 1 | 同じ 2 本 | 130 秒 |
| R1c | R1a を雛形と定数の両方に | 0 | なし | 119 秒 |
| R1d | 「`..` を含むか」を雛形と定数の両方から消す | 0 | なし | 133 秒 |
| R1e | 「`origin/` のすぐあとが `-` か」を雛形と定数の両方から消す | 0 | なし | 137 秒 |
| R1f | `` `{` `` と `` `}` `` を雛形と定数の両方に足す | 1 | `Reviewer_XreviewBaseTakesPlusAndAt`(`origin/a@{1}` = true、want false) | 168 秒 |

17 個のうち 14 個が落ちた。雛形だけの 13 個はすべて落ち、雛形と定数を一緒に変えた 4 個では R1f だけが落ちた。R1f は R1c の結果を見て足し、前景で打った。R1c の probe(overlay で足した使い捨てのテスト)の出力: 許す文字は `._/+@{-`、`origin/a@{1} -> false`(`}` で断られる)、`origin/main -> true`。

## Run 1(HEAD 15da7eb0)

- HEAD: 15da7eb0(base は origin/main の 4fef6b8c)。テストの途中(10:10 JST)に afccca76 がブランチに入った。変えたのは `docs/evidence/xreview-live-probe-2026-11-01.md` だけで、このファイルを読むテストはない(`grep -rn xreview-live-probe internal/ tests/ scripts/` が 0 件)
- 全出力: `docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore、7,185 行)。`run-test.sh` の全出力、`-race` の回、`test-xreview-helpers.sh` の 3 つの shell の回、変異ごとの差分と落ちた行を順に入れた。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-005214.log`(gitignore)

### run-test.sh

- 起動: Bash tool の `run_in_background` は使っていない。perl の fork と setsid で切り離し、INT・HUP・QUIT・TERM を既定に戻し、標準入出力を /dev/null にした。runner が log と meta を書き、前景の `caffeinate -w <pid>` で待った
- meta:

```
start 09:52:13 JST load { 7.12 7.06 6.77 }
end   10:22:43 JST rc=0 load { 7.34 8.45 8.31 }
```

- scope: `Requested scope: changed`。`scripts/xreview-helpers.sh` が分類の外なので `Language scope: full fallback` になり、golang の pack で `go test ./...` が全パッケージを打った
- shell: `tests/test-*.sh` 45 本、`    OK` の行 45、`PASS:` の行 2,590、`FAIL:` の行 0。SKIP の行は host の条件による 2 行だけ(plan-visual の browser 検出、git 2.41 より古い実物の git。手元は 2.49.0)
- Go(`(cached)` はない):

```
ok  	github.com/yoshpy-dev/ralph/cmd/ralph	1.744s
ok  	github.com/yoshpy-dev/ralph/internal/cli	728.855s
ok  	github.com/yoshpy-dev/ralph/internal/config	22.979s
ok  	github.com/yoshpy-dev/ralph/internal/gitroot	5.872s
ok  	github.com/yoshpy-dev/ralph/internal/insights	3.151s
ok  	github.com/yoshpy-dev/ralph/internal/org	968.305s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	4.566s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.780s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	5.477s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	5.489s
```

- `run-test.sh` の途中で、下の `-race` の回と変異の回を並べて打った。その間の load は 6.5〜14.6 だった。時間切れと打ち直しはない

### 追加の回

| コマンド | 結果 | 時間 |
| --- | --- | --- |
| `GOFLAGS=-timeout=30m go test -race -count=3 ./internal/org/ -run Xreview -v` | ok。名前 36 個 × 3 回でトップレベル 108、サブテスト 216 が PASS。FAIL 0、SKIP 0、DATA RACE 0 | 238.0 秒(load 6.9→7.9) |
| `sh tests/test-xreview-helpers.sh` | 53/53 | 33 秒 |
| `bash tests/test-xreview-helpers.sh`(3.2.57) | 53/53 | 28 秒 |
| `zsh --emulate sh -f tests/test-xreview-helpers.sh`(5.9) | 53/53 | 32 秒 |
| `go test -count=1 ./internal/org/driver/ -run PaneEnv -v` | 13/13 | 1 秒 |
| `go test -count=1 ./internal/cli/ -run 'TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr\|TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv' -v` | 2/2 | 8 秒 |
| `go test -count=1 -coverprofile=<scratch> ./internal/org/driver/` | ok、95.3%。`paneEnvConsequence` 100% | 2 秒 |

`-run Xreview` に掛からない新しいテストは 2 本(`TestDirectorSkill_ResultItemReportsTheCrossReviewLine`、`TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings`)。どちらも文を固定するテストで、`run-test.sh` の回で通った。

### 変異

`go test -overlay <json> -count=1 ./internal/org/ -run Xreview -v` で、go:embed の雛形(`internal/org/prompts/reviewer.md` か `leader.md`)を scratch の写しに差し替えた。写しは perl で文字列を 1 か所だけ置き換えて作った(一致が 1 か所でなければ止まる)。worktree のファイルは変えていない。変異の前後で `git status --porcelain --ignored` を比べ、go のビルドキャッシュのほかに増えたのは wrapper の log 1 本だけだった。

| ID | 置き換え | rc | 落ちたテスト | 時間 |
| --- | --- | --- | --- | --- |
| M1a | reviewer.md:196 `r=$(pick_reviewer "<IMPLEMENTER_DRIVER>")` → `r=$(pick_reviewer)` | 1 | `Reviewer_XreviewCallsTheOtherModel` 4 行(bash・zsh × implementer が claude・codex)、`Reviewer_XreviewFindsNoCLI` 4 行(no claude、a claude alias)、`Reviewer_XreviewPicksByTheImplementersDriver` | 95 秒 |
| M1b | 同じ箇所 → `r=<IMPLEMENTER_DRIVER>` | 1 | `CallsTheOtherModel` 4 行、`FindsNoCLI` 10 行すべて、`PicksByTheImplementersDriver` | 79 秒 |
| M2a | leader.md:799 `0:false)` の出力 → `echo 'switch: on'` | 1 | `Leader_XreviewStateCommandReadsTheSwitchAndTheCap` の off の行(bash・zsh)、`Leader_XreviewReadsTheSwitchWhereItIsUsed` | 85 秒 |
| M2b | leader.md:818 「TASK を送らず」→「TASK を送り」 | 1 | `Leader_XreviewReadsTheSwitchWhereItIsUsed` だけ | 113 秒 |
| M3 | leader.md:802 `f=$(head -n 1 "$p-cap" 2>/dev/null)` → `f=` | 1 | `ReadsTheSwitchAndTheCap`(a raised cap: cap 3、want 5 など)、`Leader_XreviewRaisedCapReachesTheTask`(bash・zsh) | 108 秒 |
| M4a | leader.md:804 `--diff-filter=A` を外す | 1 | `Leader_XreviewStateCommandCountsOnlyThisBranchsReports`(前の org の c2 をブランチで書き換えると done 2、want 1) | 118 秒 |
| M4b | leader.md:808 数える一覧 `"$p-reports.txt"` → `"$p-tracked.txt"` | 1 | `CountsOnlyThisBranchsReports`(base にある前の org の report を数えて done 2、want 0)、`Leader_XreviewStateCommandFindsStep9sBase` | 93 秒 |
| M5 | leader.md:810 `printf … \| grep -qE '^[0-9a-f]{40}([0-9a-f]{24})?$' && ` を外す | 1 | `Leader_XreviewStateCommandChecksTheReportsHead`(sha1・sha256 × `HEAD`、省略した SHA、大文字の SHA で since: same)、`Leader_XreviewChecksTheValuesItReads` | 99 秒 |
| M6 | reviewer.md:285 `git diff --end-of-options` → `git diff` | 1 | `Reviewer_XreviewCommandOutcomes` の claude with a base that reads as an option(bash・zsh。`--output=` のファイルが書かれた)、`Reviewer_XreviewRunsTheOtherModelReadOnly` | 148 秒 |

テスト名の頭の `TestRenderRolePrompt_` は省いた。9 個とも落ちた。M2b は文だけの変異で、文を固定するテストだけが落とす。ほかの 8 個は、雛形のコマンドをスタブで走らせる行が落とす。
