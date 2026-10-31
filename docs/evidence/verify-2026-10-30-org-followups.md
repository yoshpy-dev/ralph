# Verify evidence: org-followups (#270)

先頭が cycle 2、次が cycle 1。cycle 1 の節は中身を変えず、見出しを 1 段下げた。

## Cycle 2

- Date: 2026-11-01 03:09〜03:20 JST(2026-10-31 18:09〜18:20 UTC)
- Worktree: `.claude/worktrees/org-followups`、branch `feat/org-followups`、HEAD 12d28dae(origin/feat/org-followups と同じ)、base origin/main 1cc84ffa(merge-base と同じ)
- 全出力: `docs/evidence/verify-2026-10-30-org-followups.log` の末尾の `===== cycle 2` 以下(コミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-31-180917.log`(同じく対象外)
- パスの置き換え: probe の一時ディレクトリは `<W>`、scratchpad は `<S>` と書いた。token や secret の値は出力にない(静的解析、単独の検査、probe の出力で `grep -c -i 'ghp_\|github_pat\|token'` はどれも 0)

### A. 静的解析

`./scripts/run-static-verify.sh`(rc=0、39 秒)の要点:

```
==> Codex hook single-source guard / inline hook detector smoke test / PR provenance policy guard   (3 つとも OK)
==> scripts/check-sync.sh
  IDENTICAL:      172
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  10
  KNOWN_DIFF:     6
PASS: all files in sync.
==> scripts/check-pipeline-sync.sh   (7 行とも [ok])
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
==> scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
==> tech-debt README plan references
    OK
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned 1cc84ffa..12d28dae against origin/main: clean
==> All verifiers passed.
```

`Skipping` の行は 0(go vet と staticcheck は無出力。`staticcheck` と `golangci-lint` は PATH にある)。

別に打ったもの:

```
./scripts/check-template-purity.sh   -> PASS, rc=0
./scripts/check-skill-sync.sh        -> [ok] 14 skill(s) in lock-step, rc=0
./scripts/check-sync.sh              -> IDENTICAL 172, DRIFTED 0, ROOT_ONLY 0, PASS, rc=0
git diff --check origin/main...HEAD  -> 無出力, rc=0
git diff --name-only origin/main...HEAD | xargs grep -l $'\xef\xbf\xbd'  -> 0 件 (grep rc=1)
./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-followups.md -> 6ef9fe7cb76c (- Approved: の digest と同じ)
cmp: .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills の org と director の SKILL.md は .claude/skills と同じ
diff docs/quality/quality-gates.md templates/base/docs/quality/quality-gates.md -> 違いは root の 54・57-58・82 行(既存の KNOWN_DIFF)。Cross-org watch gate の行(root 80、template 79)は同じ文
go build -o <S>/ralph ./cmd/ralph    -> rc=0 (go1.27.3)。前後で git status --porcelain は空
ralph org merge-order --help の Usage より前の行 -> 最長 72 桁
```

### B. 見張りのない台帳の `followup` の動詞

組み方は cycle 1 の 2 節と同じ(bare origin と main、`env -i`、git の包み、偽の gh)。偽の gh は `PROBE_GH_MODE=fail` のときどの呼び出しも `HTTP 502 probe` で exit 1 にした。台帳 `<W>/ledger-nowatch` は空のディレクトリから始めた。

```
added f1: issue #1 after feat/a                                  [rc=0]
ID	STATE	ISSUE	AFTER_BRANCH	ADDED_AT	NOTE
f1	open	1	feat/a	2026-10-31T18:12:20Z	-                     [rc=0]
list --json                                                      [rc=0]
done f1                                                          [rc=0]
ID	STATE	ISSUE	AFTER_BRANCH	ADDED_AT	NOTE	DONE_AT	DONE_NOTE
f1	done	1	feat/a	2026-10-31T18:12:20Z	-	2026-10-31T18:12:20Z	probe   [rc=0]
台帳の中身: followups.jsonl  followups.lock
watch.lock absent / watch-alive.json absent
```

同じ台帳での対照(`ralph org spawn --org-id o1 --id leader --role leader --driver claude --model opus --cwd <W>/main --allow-unscoped --dry-run`):

```
org: no ralph org watch --all is running on the org state dir <W>/ledger-nowatch, so no new work starts; start one with ralph org watch --all --detach --state-dir '<W>/ledger-nowatch', or, for a new org, start it with ralph org start, which starts one
rc=1。台帳の中身は followups.jsonl と followups.lock のまま(manifest.jsonl はできない)
```

### C. merge-order の SOURCE の列と `gh_available`、status

台帳 `<W>/ledger` に follow-up を 7 件(feat/merged、feat/old、feat/open、feat/closed、feat/none、feat/fail、feat/badtime。偽の gh の答えは cycle 1 の 2 節の表と同じ)。機能のブランチの結びつきはない。

merge-order の text(偽の gh、rc=0):

```
merge guarantee: unknown (not checked: run ralph doctor)
note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it or ralph org gate records a gate on it

follow-ups:
ID	ISSUE	AFTER_BRANCH	STATE	SOURCE	PR	DETAIL
f1	100	feat/merged	ready	gh	#7 https://github.com/acme/app/pull/7	PR #7 was merged at 2099-01-01T00:00:00Z, after the follow-up was recorded at 2026-10-31T18:12:20Z: ask the human about issue #100 now, then run ralph org followup done f1
f2	100	feat/old	waiting	gh	#3 https://github.com/acme/app/pull/3	PR #3 was merged before the follow-up was recorded (merged at 2020-01-01T00:00:00Z, recorded at 2026-10-31T18:12:20Z), so it does not count: ...
f3	100	feat/open	waiting	gh	#12 https://github.com/acme/app/pull/12	PR #12 is open: the follow-up waits for its merge; PR #4 was merged before the follow-up was recorded (at 2020-01-01T00:00:00Z) and does not count
f4	100	feat/closed	closed	gh	#9 https://github.com/acme/app/pull/9	PR #9 was closed without a merge, so the merge the follow-up waits for may not come: ...
f5	100	feat/none	waiting	gh	-	feat/none has no PR yet: the follow-up waits for one to be opened and merged
f6	100	feat/fail	unknown	gh	-	gh pr list --head feat/fail failed (exit status 1: HTTP 502 probe): its PR is unknown
f7	100	feat/badtime	unknown	gh	#5 https://github.com/acme/app/pull/5	PR #5 is merged, but gh gave its merge time as "", which cannot be compared with the follow-up's added_at 2026-10-31T18:12:21Z
タブで割った列の数: 見出しと 7 行とも 7、5 列目は見出しが SOURCE、7 行とも gh
gh の呼び出し: 7
```

`--json` の `gh_available`(jq で `{gh_available, entries, items, states}` を抜き出した。どれも rc=0):

| 場合 | 台帳 | gh | gh_available | entries | items | gh の呼び出し |
| --- | --- | --- | --- | --- | --- | --- |
| C1 | follow-up 7 件だけ | 偽の gh | true | 0 | 7(ready・waiting・waiting・closed・waiting・unknown・unknown) | 7 |
| C2 | follow-up 7 件だけ | すべて失敗 | true | 0 | 7(すべて unknown、`gh pr list --head feat/merged failed (exit status 1: HTTP 502 probe)` など) | 7 |
| C3 | follow-up 7 件だけ | PATH にない | false | 0 | 7(すべて unknown、`gh was not found, so no PR was read: ...`) | - |
| C4 | 空 | 偽の gh | false | 0 | 0 | 0 |
| C5 | done の follow-up 1 件だけ(B の台帳) | 偽の gh | false | 0 | 0 | 0 |
| C6 | 機能のブランチ feat/bound だけ | 偽の gh | true | 1 | 0 | 1 |
| C6' | 機能のブランチ feat/bound だけ | PATH にない | false | 1 | 0 | - |
| C7 | feat/bound と follow-up 7 件 | 偽の gh | true | 1 | 7(C1 と同じ状態) | 8 |
| C7' | feat/bound と follow-up 7 件 | すべて失敗 | true | 1 | 7(すべて unknown) | - |
| C7'' | feat/bound と follow-up 7 件 | PATH にない | false | 1 | 7(すべて unknown) | - |

C1 の `items[].source` は `["gh"]` だけ。gh のない text でも follow-up の節の見出しは `ID ISSUE AFTER_BRANCH STATE SOURCE PR DETAIL` で、行の SOURCE は `gh`。C7 の text でも同じ見出しが出た。

`ralph status`(同じ台帳、偽の gh を PATH に置いたまま):

```
follow-ups:
ID	ISSUE	AFTER_BRANCH	STATE	SOURCE	DETAIL
f1	100	feat/merged	waiting	recorded	no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub
[rc=0]
--json の items[].source: ["recorded"]
gh の呼び出し: 0
```

この台帳の manifest には merge の記録がないので、status の f1 は `waiting`(merge-order は gh で `ready`)。ready と waiting で割れたときの規則が働く場面で、status の DETAIL が merge-order を名指す。

### D. 行数と tech-debt の Trigger(cycle 2 の分)

```
                                   origin/main  9a49f306  25c9fa77(F5)  12d28dae(HEAD)
internal/org/mergeorder.go              1291       1322        1328          1328
internal/org/mergeorder_followup.go        -        360         370           371
internal/cli/status.go                   901        963           -           963
internal/cli/status_followup.go            -        189           -           189
```

`docs/tech-debt/README.md` の 303 行(「1291 to 1322 lines」)と 411 行(「to 1322」「(360 lines)」)は 9a49f306 の数のまま。410 行の 963 と 189 は今と同じ。

Trigger の欄を、cycle 2 で変わったものの名前(`mergeorder`、`merge-order`、`MergeOrder`、`GHAvailable`、`gh_available`、`lookupFollowupMerges`、`checkFollowupPRs`、`mergeOrderFollowups`、`printMergeOrderFollowups`、`printStatusFollowups`、`status_followup`、`follow-up`、`quality-gates`、watch gate、`/org` skill、command table ほか)で引いた(`<S>/v270c2/sweep.py`)。

当たったと判断した行:

- 118: `templates/base/docs/quality/quality-gates.md` への次のタッチ。sync-docs 9717f0e8 が 79 行(Cross-org watch gate の行)を変えた。org-followups の注はない。「Must pass in CI」の 58-59 行は今も `check-coverage.sh` と `check-pipeline-sync.sh` を挙げ、`templates/base/.github/workflows/verify.yml` にはどちらの名前もない(grep 0 件)
- 246: `mergeorder.go`・`org_merge_order.go` とそのテストの次の編集。F5・F6 が当たる。org-followups の注はある(523b9091 までの編集を挙げる)
- 251: `MergeOrder` の次の編集。F5・F6 は説明のコメントだけ。org-followups の注はある
- 303: `mergeorder.go` の次の変更。org-followups の注はあるが、数が古い
- 411: `mergeorder.go` の次の変更。F5・F6 が当たる。注はなく、数が古い
- 412 (4): `checkFollowupPRs` か `lookupFollowupMerges` の次の変更。F5 が `lookupFollowupMerges` を変えた。注はない

当たったが、当たっていないと判断した行: 117(quality-gates の内容の組み直しではなく、1 語句の追加)、157(品質の流れは変えていない)、177(sync-docs が変えたのは `/org` skill の見張りの節で、command table ではない。F4 の command table の編集は sync-docs の注に入っている)、183・186・188・301・309・310・352・373・379(名指された関数や節は変えていない)、235・359・409・410(cycle 2 で名指されたファイルや出来事に触れていない)、240〜243・247・248(出来事)、39・43・76・134・245・337(閉じた行)。

## Cycle 1

- Date: 2026-11-01 01:04〜01:18 JST(2026-10-31 16:04〜16:18 UTC)
- Worktree: `.claude/worktrees/org-followups`、branch `feat/org-followups`、HEAD 9a49f306、base origin/main 1cc84ffa(merge-base と同じ)
- 全出力: `docs/evidence/verify-2026-10-30-org-followups.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-verify 自身のログは `docs/evidence/verify-2026-10-31-160435.log`(同じく対象外)
- パスの置き換え: probe の一時ディレクトリは `<W>`、scratchpad は `<S>` と書いた。token や secret の値は出力にない(`grep -c 'token\|ghp_\|github_pat'` は 0)

### 1. 静的解析

`./scripts/run-static-verify.sh`(rc=0、68 秒)の要点:

```
==> scripts/check-sync.sh
  IDENTICAL:      172
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  10
  KNOWN_DIFF:     6
PASS: all files in sync.
==> scripts/check-pipeline-sync.sh   (7 行とも [ok])
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 14 skill(s) in lock-step
==> scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
==> tech-debt README plan references
    OK
==> Language scope: changed (changed_languages)
==> Language packs selected: golang
gofmt: ok
0 issues.
==> Running branch secret scan
secret-scan-branch: scanned 1cc84ffa..9a49f306 against origin/main: clean
==> All verifiers passed.
```

`go vet ./...` と `staticcheck ./...` は無出力(`packs/languages/golang/verify.sh` の `run_static`。`Skipping` の行はなく、`staticcheck` と `golangci-lint` は PATH にある)。`0 issues.` は golangci-lint。

別に打ったもの:

```
./scripts/check-template-purity.sh   -> PASS, rc=0
./scripts/check-skill-sync.sh        -> [ok] 14 skill(s) in lock-step, rc=0
./scripts/check-sync.sh              -> IDENTICAL 172, DRIFTED 0, ROOT_ONLY 0, PASS, rc=0
git diff --check origin/main...HEAD  -> 無出力, rc=0
git diff --name-only origin/main...HEAD | xargs grep -l $'\xef\xbf\xbd'  -> 0 件 (grep rc=1)
./scripts/plan-visual.sh digest docs/plans/active/2026-10-30-org-followups.md -> 3d00daf5cf87 (- Approved: の digest と同じ)
cmp: .agents/skills, templates/base/.claude/skills, templates/base/.agents/skills の org と director の SKILL.md は .claude/skills と同じ
go build -o <S>/ralph ./cmd/ralph    -> rc=0 (go1.27.3)。前後で git status --porcelain は空
```

### 2. probe(HEAD の binary、一時 repo)

組み方: `<W>/origin.git`(bare)と `<W>/main`(空コミット 1 つを push、`remote set-head origin main`)。`env -i HOME=<W>/home GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=<W>/gitconfig PATH=<W>/bin` で打った。`<W>/bin/git` は `remote get-url origin` にだけ `https://github.com/acme/app.git` と答え、ほかは `/opt/homebrew/bin/git` を exec する。`<W>/bin/gh` は呼び出しを `gh.log` に書き、`--head` のブランチごとに決めた JSON を返す偽物(下の表)。`<W>/nogh` は git だけを持つ。ネットワーク、herdr、agmsg、本物の gh は使っていない。台帳は `--state-dir <W>/ledger`。

| after-branch | 偽の gh が返す PR | 台帳(manifest)の記録 |
| --- | --- | --- |
| feat/merged | #7 MERGED、mergedAt 2099-01-01 | follow-up のあとの `pr_merged` |
| feat/old | #3 MERGED、mergedAt 2020-01-01 | 2020-01-01 の `pr_merged` |
| feat/open | #12 OPEN と #4 MERGED(2020) | なし |
| feat/closed | #9 CLOSED | なし |
| feat/none、feat/a、feat/clean-before | なし(`[]`) | clean-before: 2020 の `pr_merged`(head H5)と、あとの `worktree_cleaned`(同じ head H5) |
| feat/fail | exit 1(`HTTP 502 probe`) | なし |
| feat/badtime | #5 MERGED、mergedAt `""` | なし |
| feat/clean-gh | #6 MERGED、mergedAt 2020 | あとの `worktree_cleaned` だけ(`pr_merged` なし) |
| feat/clean-new | #8 MERGED、mergedAt 2099 | 2020 の `pr_merged`(head H5)と、あとの `worktree_cleaned`(別の head H6) |

予約(AC5): org-r は `scope_reserved`(`paths=docs/,internal/auth/`)だけで座席なし、org-s は leader の `spawned` と `scope_reserved`(`paths=internal/org/`)、org-t は leader の `spawned` だけ。

#### AC1 add・list・done と断り

```
added f1: issue #100 after feat/merged          (f1〜f11 まで同じ形。f8 は feat/merged の 2 件目)
ID	STATE	ISSUE	AFTER_BRANCH	ADDED_AT	NOTE
f1	open	100	feat/merged	2026-10-31T16:10:07Z	wait for feat/merged
...
list --json -> {"items":[{"id":"f1","issue":100,"after_branch":"feat/merged","note":"wait for feat/merged","state":"open","added_at":"2026-10-31T16:10:07Z"}, ...
added f12: issue #200 after feat/x
done f12                                       [rc=0]
f12 is already done; nothing changed           [rc=0]
org: followup: done f99: no such follow-up     [rc=1]
list --all の末尾:
f11	open	100	feat/clean-new	2026-10-31T16:10:08Z	wait for feat/clean-new	-	-
f12	done	200	feat/x	2026-10-31T16:10:08Z	-	2026-10-31T16:10:08Z	human said go
```

断り(どれも rc=1、記録の行は増えない):

```
org: followup: the issue must be a positive number, not 0
org: followup: the after-branch "-feat" starts with -
org: followup: the after-branch "feat/a b" contains whitespace or a control character (U+0020)
org: followup add: --issue is required
org: followup add: --after-branch is required
org: followup: the note must be a single line
org: followup add: --org-id does not apply: a follow-up belongs to the whole ledger, not to one org
org: followup list: --org-id does not apply: a follow-up belongs to the whole ledger, not to one org
org: followup done: --org-id does not apply: a follow-up belongs to the whole ledger, not to one org
```

`--note ''` は「note なし」として受け付けられ f13 を記録した(rc=0)。断りの組に入れたのは probe の誤りで、`ValidateFollowupAdd` は `""` を note なしとして通す(`internal/org/followup.go:226-228`)。記録は 14 行(add 11、f12 の add と done、f13)で、9 つの断りは何も書いていない。

#### AC2 lock、壊れた行、ID

```
<W>/ledger の中身: followups.jsonl  followups.lock
--- <W>/ledger2: f1 の added、知らない event、"id":"f50" で切れた最後の行を置いてから add
added f51: issue #2 after feat/b
warning: 1 unreadable follow-up line(s) skipped in <W>/ledger2/followups.jsonl
warning: 1 follow-up event(s) ignored in <W>/ledger2/followups.jsonl (unknown kind or ID, a second added, an added event that add would refuse, or a done with no open follow-up)
ID	STATE	ISSUE	AFTER_BRANCH	ADDED_AT	NOTE
f1	open	1	feat/a	2026-10-30T00:00:00Z	-
f51	open	2	feat/b	2026-10-31T16:10:09Z	-
--- raw record:
{"ts":"2026-10-30T00:00:00Z","id":"f1","event":"added","issue":1,"after_branch":"feat/a"}
{"ts":"2026-10-30T00:00:00Z","id":"f2","event":"renamed"}
{"ts":"2026-10-30T00:00:00Z","id":"f50","event":"add
{"ts":"2026-10-31T16:10:09Z","id":"f51","event":"added","issue":2,"after_branch":"feat/b"}
--- 空の台帳に 20 の add を同時に打った
異なる ID の数: 20 / 記録の行: 20 / 20 とも "added f<N>" を出した
```

#### AC4・AC5・AC7・AC8 `ralph status`

```
org_id: org-r (active 0/0)
  reserved: docs/, internal/auth/

org_id: org-s (active 1/1)
  SEAT_ID	ROLE	DRIVER	MODEL	STATE	PANE_ID
  leader	leader	claude	opus	spawned (active)	p1
  reserved: internal/org/

org_id: org-t (active 1/1)
  SEAT_ID	ROLE	DRIVER	MODEL	STATE	PANE_ID
  leader	leader	claude	opus	spawned (active)	p2

follow-ups:
ID	ISSUE	AFTER_BRANCH	STATE	SOURCE	DETAIL
f1	100	feat/merged	ready	recorded	the ledger records pr_merged at 2026-10-31T16:10:11Z, after the follow-up was recorded at 2026-10-31T16:10:07Z: ask the human about issue #100 now, then run ralph org followup done f1
f2	100	feat/old	waiting	recorded	no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub
f3	100	feat/open	waiting	recorded	(同上)
f4	100	feat/closed	waiting	recorded	(同上)
f5	100	feat/none	waiting	recorded	(同上)
f6	100	feat/fail	waiting	recorded	(同上)
f7	100	feat/badtime	waiting	recorded	(同上)
f8	100	feat/merged	ready	recorded	the ledger records pr_merged at 2026-10-31T16:10:11Z, ... run ralph org followup done f8
f9	100	feat/clean-before	waiting	recorded	no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub
f10	100	feat/clean-gh	ready	recorded	the ledger records worktree_cleaned at 2026-10-31T16:10:11Z, ... run ralph org followup done f10
f11	100	feat/clean-new	ready	recorded	the ledger records worktree_cleaned at 2026-10-31T16:10:11Z, ... run ralph org followup done f11
f13	1	feat/a	waiting	recorded	no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub
[rc=0]
```

`--json`(jq で抜き出した):

```
{"org_id":"org-r","reservation":["docs/","internal/auth/"],"seats":0}
{"org_id":"org-s","reservation":["internal/org/"],"seats":1}
{"org_id":"org-t","reservation":[],"seats":1}
followups: {"keys":["corrupt_lines","error","ignored_events","items"],"corrupt_lines":0,"ignored_events":0,"error":""}
{"id":"f1","after_branch":"feat/merged","state":"ready","source":"recorded","merged_event":"pr_merged","merged_at":"2026-10-31T16:10:11Z"}
{"id":"f2","after_branch":"feat/old","state":"waiting","source":"recorded","merged_event":"","merged_at":""}
{"id":"f9","after_branch":"feat/clean-before","state":"waiting","source":"recorded","merged_event":"","merged_at":""}
{"id":"f10","after_branch":"feat/clean-gh","state":"ready","source":"recorded","merged_event":"worktree_cleaned","merged_at":"2026-10-31T16:10:11Z"}
{"id":"f11","after_branch":"feat/clean-new","state":"ready","source":"recorded","merged_event":"worktree_cleaned","merged_at":"2026-10-31T16:10:11Z"}
item keys: ["added_at","after_branch","detail","id","issue","merged_at","merged_event","source","state"]
```

`ralph status --org-id org-r` は org-r の組(`reserved:` の行だけ)と、絞らない follow-up の節を出した。status を打つ前に `gh.log` を空にし、text・`--json`・`--org-id` の 3 回のあとも空だった(`no gh call`)。

空の台帳との JSON の形(AC5):

```
empty:     ["director","followups","merge_guarantee","org_watch","orgs","state_dir","state_dir_source"]
populated: ["director","followups","merge_guarantee","org_watch","orgs","state_dir","state_dir_source"]
empty の followups: {"items":[],"corrupt_lines":0,"ignored_events":0,"error":""}
populated の orgs の各要素の keys: ["active_count","org_id","reservation","seats","total_count"] (3 つとも)
status は空の台帳に followups.jsonl を作らない
```

#### AC3・AC7・AC8 `ralph org merge-order`

機能のブランチのない台帳(偽の gh、rc=0):

```
merge guarantee: unknown (not checked: run ralph doctor)
note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it or ralph org gate records a gate on it

follow-ups:
ID	ISSUE	AFTER_BRANCH	STATE	PR	DETAIL
f1	100	feat/merged	ready	#7 https://github.com/acme/app/pull/7	PR #7 was merged at 2099-01-01T00:00:00Z, after the follow-up was recorded at 2026-10-31T16:10:07Z: ask the human about issue #100 now, then run ralph org followup done f1
f2	100	feat/old	waiting	#3 https://github.com/acme/app/pull/3	PR #3 was merged before the follow-up was recorded (merged at 2020-01-01T00:00:00Z, recorded at 2026-10-31T16:10:07Z), so it does not count: the follow-up waits for a new PR of feat/old to be merged; if none is coming, ask the human about issue #100 now and run ralph org followup done f2
f3	100	feat/open	waiting	#12 https://github.com/acme/app/pull/12	PR #12 is open: the follow-up waits for its merge; PR #4 was merged before the follow-up was recorded (at 2020-01-01T00:00:00Z) and does not count
f4	100	feat/closed	closed	#9 https://github.com/acme/app/pull/9	PR #9 was closed without a merge, so the merge the follow-up waits for may not come: reopen PR #9 or open a new PR of feat/closed, or ask the human about issue #100 and run ralph org followup done f4
f5	100	feat/none	waiting	-	feat/none has no PR yet: the follow-up waits for one to be opened and merged
f6	100	feat/fail	unknown	-	gh pr list --head feat/fail failed (exit status 1: HTTP 502 probe): its PR is unknown
f7	100	feat/badtime	unknown	#5 https://github.com/acme/app/pull/5	PR #5 is merged, but gh gave its merge time as "", which cannot be compared with the follow-up's added_at 2026-10-31T16:10:07Z
f8	100	feat/merged	ready	#7 https://github.com/acme/app/pull/7	(f1 と同じ PR)
f9	100	feat/clean-before	waiting	-	feat/clean-before has no PR yet: the follow-up waits for one to be opened and merged
f10	100	feat/clean-gh	waiting	#6 https://github.com/acme/app/pull/6	PR #6 was merged before the follow-up was recorded (merged at 2020-01-01T00:00:00Z, ...), so it does not count: ...
f11	100	feat/clean-new	ready	#8 https://github.com/acme/app/pull/8	PR #8 was merged at 2099-01-01T00:00:00Z, after the follow-up was recorded at 2026-10-31T16:10:08Z: ...
f13	1	feat/a	waiting	-	feat/a has no PR yet: the follow-up waits for one to be opened and merged
[rc=0]
gh の呼び出し(--head ごとの回数): feat/a, badtime, clean-before, clean-gh, clean-new, closed, fail, merged, none, old, open がそれぞれ 1 回(feat/merged は follow-up が 2 件でも 1 回)
呼び出しの形: pr list --head feat/merged --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository,mergedAt
```

`--json`:

```
followups: {"keys":["corrupt_lines","error","ignored_events","items"],"corrupt_lines":0,"ignored_events":0,"error":""}
{"id":"f1","state":"ready","source":"gh","pr":7,"merged_at":"2099-01-01T00:00:00Z"}
{"id":"f2","state":"waiting","source":"gh","pr":3,"merged_at":"2020-01-01T00:00:00Z"}
{"id":"f3","state":"waiting","source":"gh","pr":12,"merged_at":""}
{"id":"f4","state":"closed","source":"gh","pr":9,"merged_at":""}
{"id":"f5","state":"waiting","source":"gh","pr":null,"merged_at":""}
{"id":"f6","state":"unknown","source":"gh","pr":null,"merged_at":""}
{"id":"f7","state":"unknown","source":"gh","pr":5,"merged_at":""}
{"id":"f10","state":"waiting","source":"gh","pr":6,"merged_at":"2020-01-01T00:00:00Z"}
item keys: ["added_at","after_branch","detail","id","issue","merged_at","pr","source","state"]
```

gh のない PATH(`<W>/nogh`): 12 件とも `unknown`、DETAIL は `gh was not found, so no PR was read: ralph status tells it from the ledger's records of merges, without gh`。rc=0。

機能のブランチを 1 つ結びつけた台帳(`scope_reserved` に `branch=feat/bound`): `origin/main 95faf6fde5b8 (listed ...)`、保証の行、表(`1 no_gate feat/bound ...`)、注記のあとに、上と同じ follow-up の節が出た。rc=0。

#### 記録が読めないとき

```
followups.jsonl をディレクトリにした台帳:
ralph org followup list -> org: followup: read <W>/ledger-dir/followups.jsonl: read <W>/ledger-dir/followups.jsonl: is a directory  [rc=1]
ralph status            -> stderr: warning: status: org: followup: read ...: is a directory; no follow-up is shown  [rc=0]
```

### 3. help(HEAD の binary)

```
ralph org --help:      Long の 24 行目から followup の文、Available Commands に "followup    Record issues to ask the human about once another branch's PR is merged"、--org-id の help に "the inbox verbs, and the followup verbs, which refuse it"
ralph org followup --help: add / done / list の 3 つ。Long は "when one says ready and the other waiting, use merge-order's answer, and when merge-order says unknown, go by status's answer and give merge-order's reason."
ralph status --help:   follow-up か reserved を含む行が 8 行
ralph org merge-order --help: 68 行目から follow-ups: の説明
```

### 4. 分かれ目の規則の文(D5、R2-1)

`git grep -c` で数えた、ready と waiting で割れたら merge-order、unknown なら status と理由、の文の出現:

```
英語 ("merge-order says unknown"): AGENTS.md 1, README.md 1, docs/tech-debt/README.md 1, internal/cli/org_followup.go 2, internal/cli/org_merge_order.go 1, internal/cli/status.go 1
日本語 ("merge-order が `unknown` なら"): .claude/skills/director/SKILL.md 1, .claude/skills/org/SKILL.md 1, docs/specs/2026-10-07-org-multi-org-director.md 1, docs/plans/active/2026-10-30-org-followups.md 1 (Progress の F3 の記録だけ)
狭める前の言い方 ("use merge-order's answer" のあとに unknown の節がない文): 0 件
```

計画の D5 の本文(56 行)は「2 つの道が食い違ったら merge-order(gh)の答えを使う」のままで、R2-1 の狭めは Progress にしかない。

### 5. 行数(tech-debt の 410・411 行との照合)

```
                         origin/main  c9cb588f  aae31317(=HEAD)
internal/cli/status.go         901       959       963
internal/cli/status_followup.go  -       187       189
internal/org/mergeorder.go    1291      1322      1322
internal/org/mergeorder_followup.go -    355       360
```

410 行は c9cb588f の数(959 と 187)のまま。aae31317(R2-2 の help と説明)で 963 と 189 になった。411 行の 1322 と 360 は今と同じ。

### 6. tech-debt の Trigger の掃き出し

Trigger の欄(4 列目)を、この branch が変えたファイルと関数の名前(`mergeorder`、`merge-order`、`lookupMergePR`、`ghBranchPRs`、`status.go`、`ralph status`、`lockfile`、`withFileLock`、`manifestLockTimeout`、`seat.go`、`pr_merged`、`worktree_cleaned`、`org_legacy_ledger`、`/org` と `/director` の skill、`RunningOrgs`、`ActiveReservation`、`inboxField`、`InboxNoteMaxRunes` ほか)で引いた(`<S>/v270/sweep.py`)。

当たったと判断し、`org-followups` の注がない行:

- 177 (a): `/org` skill の command table の次の編集。`status`・`merge-order` の行を伸ばし、`followup add`・`list`・`done` の 3 行を足した
- 235: `internal/org/lockfile.go` の次の変更(コメントだけ。`manifestLockTimeout`・`manifestLockPollInterval`・`withFileLock` の説明に follow-up の lock を足した)
- 246: `internal/org/mergeorder.go`・`internal/cli/org_merge_order.go`・それらのテスト(`mergeorder_test.go`)の次の編集
- 251: `MergeOrder` の次の編集(`Followups` の初期化と `mergeOrderFollowups` の呼び出し 2 か所)
- 303: `mergeorder.go` の次の変更(runner を自分のファイルへ)。この branch が足した 411 行も同じファイルの大きさを扱うが、303 には触れていない
- 359: `ralph status` の座席の表の次の変更(`internal/cli/status.go:772`、座席のない組では見出しを出さない)

当たったが、当たっていないと判断した行: 76・134・337・347(閉じた行)、157(品質の流れは変えていない)、170・399(`LegacyWorktreeStateDir`・`guardLegacyOrgStateDir`・`resolveOrgLedger` は変えていない。`org_legacy_ledger.go` はコメントだけ)、174・196・259・376・377(`/director` skill の別の節)、183・184・186・188(予約の関数は変えていない。`RunningOrgs` と `ActiveReservation` は呼ぶだけ)、192・208・212・267(出来事や別の関数)、203(受信箱の `body` と名前)、204(`withFileLock` は説明のコメントだけで、動きは同じ。境目の判断)、240・241・242・243・245・247・248・249(出来事。245 の「internal/org に外部コマンドを足す次の PR」は、gh の同じ `pr list` に `mergedAt` を足しただけなので当たらないと判断)、257(status の先頭の行は増えていない。follow-up の節は org のあと)、301・309・310(merge-order の断りの文・分類は変えていない)、352・373(`/org` skill の別の節)、362・386(名指されたファイルは変えていない)、409・410・411(この branch が足した行)。
