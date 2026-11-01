# Test evidence: org-cross-review(cycle 1)

- HEAD: 15da7eb0(base は origin/main の 4fef6b8c)。テストの途中(10:10 JST)に afccca76 がブランチに入った。変えたのは `docs/evidence/xreview-live-probe-2026-11-01.md` だけで、このファイルを読むテストはない(`grep -rn xreview-live-probe internal/ tests/ scripts/` が 0 件)
- 全出力: `docs/evidence/test-2026-11-01-org-cross-review.log`(gitignore、7,185 行)。`run-test.sh` の全出力、`-race` の回、`test-xreview-helpers.sh` の 3 つの shell の回、変異ごとの差分と落ちた行を順に入れた。wrapper 自身の写しは `docs/evidence/verify-2026-11-01-005214.log`(gitignore)

## run-test.sh

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

## 追加の回

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

## 変異

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
