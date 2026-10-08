# sync-docs report: org-limits-reserve

- Date: 2026-10-09 JST(ファイル名は計画の日付)
- Plan: `docs/plans/active/2026-10-08-org-limits-reserve.md`
- Pipeline cycle: 2 of 2(既定の上限の最後の回)。差分は merge base `51855166` から、sync-docs 開始時の branch HEAD `6ee9836b`(feat/org-limits-reserve)まで
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-08-org-limits-reserve.md`(`a919082a`。Merge 可、C2-1〜C2-7)、
  `docs/reports/verify-2026-10-08-org-limits-reserve.md`(`f144021d`。pass、V2-1〜V2-5)、
  `docs/reports/test-2026-10-08-org-limits-reserve.md`(`6ee9836b`。pass、T2-1〜T2-4)
- cycle 1 のこの report は `196205f8` にある。この回が上書きした。cycle 1 の report の節を指す箇所はほかにないので、付録には残していない(`git grep` で 0 件)

## Summary

verify が渡した V2-1〜V2-5 は、文書で直せる部分をすべて直した。V2-2 のうち Go の help の文字列は直していない。self-review の C2-6 と C2-7 も文書だけの修正なので、同じ commit で直した。

直さずに tech-debt へ送ったのは、`internal/` の変更になるもの(C2-1〜C2-4 の Go の doc comment、条件文、help の文字列)と、C2-5(tech-debt の行と test のコメントが report の番号を指す件)である。`internal/` を変えるとパイプラインが `/self-review` から再実行になり、上限の 2 回を超える。tech-debt には新しい 2 行(177、178 行目)を足した。1 行目は C2-1〜C2-5 と V2-1(文言の LOW)、2 行目は T2-1〜T2-4(test の穴)である。

`/org` skill の 4 面は変えていない。cycle 2 のコードの変更(動いていない leader への `--reserve` を `max_orgs` で拒否する枝)を書いた箇所が skill にないためである。

commit は 1 つで、この report と同じ commit に入れた。

## Changes made

| File | Change |
|------|--------|
| `docs/plans/active/2026-10-08-org-limits-reserve.md`(157 行目) | V2-1: Progress checklist に 1 行足した。`975df92b` と `afcbc6c2` の、動いていない leader への `--reserve` を `max_orgs` で拒否する枝は `rejected` を書かない。立っている leader への予約の拒否(AC14)と同じで、`rejected` は状態のイベントなので、書くと座席の最新の状態が `spawned` から変わり、予約なしの再試行が既存の座席を返さなくなる。AC1 の「`rejected` が残る」は新しい座席の spawn に当たる。本文には触れていない。digest は `1a165903b5df` のまま |
| `templates/base/ralph.toml`(41〜46 行目) | V2-2 と C2-4 の `ralph.toml` 側: `max_orgs` のコメントを、「`ralph org spawn` と `ralph org start` が main worktree の `ralph.toml` を読むのは、`--config` がなく、`--state-dir` も `RALPH_ORG_STATE_DIR` も設定されておらず、state dir が main worktree の `.harness/state/org` に決まったときだけ。それ以外は `--config` のファイル、なければ `./ralph.toml`」に書き直した。値は変えていない。ルートに `ralph.toml` はなく(`check-sync.sh` では TEMPLATE_ONLY)、同じ文を持つファイルはほかにない(`internal/config/config.go` と `scripts/ralph-config.sh` を `git grep` して当たらなかった) |
| `AGENTS.md`(90 行目、managed block の外) | V2-3 と C2-7: repo map の `internal/org/` の説明を「org-wide `max_orgs` / `max_total_seats` in `envelope.go`, their counts and scope reservations in `reserve.go`」にした。判定(`ValidateOrgWideCapacity`、`validateMaxOrgs`)は `envelope.go`、数え方(`RunningOrgs`、`TotalActiveSeats`)と予約は `reserve.go` にある |
| `docs/tech-debt/README.md` | V2-4 と C2-6: 173 行目の (a) の「and nothing says so」を「and nothing warns when the path is typed」にした(同じ行の「help と skill が規則を述べる」と食い違わなくなる)。176 行目の (b) の `idempotentRespawn` の 87.5% を「90.9% since the cycle 2 fix」にした(他の 3 つの値は cycle 2 の test report でも同じ)。177、178 行目を新しく足した(下の表) |
| `docs/reports/self-review-2026-10-08-org-limits-reserve.md` | V2-5: 末尾の空行を 1 つ消した。判定の行(finding の重さと `Merge:`)には触れていない |
| `docs/insights/events/2026-10-08-org-limits-reserve.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 2) |
| `docs/reports/sync-docs-2026-10-08-org-limits-reserve.md` | この report(cycle 1 の版を上書き) |

### tech-debt の新しい行

| 行 | 内容 |
|----|------|
| 177 行目 | cycle 2 の文言の LOW(C2-1〜C2-5、V2-1、V2-2)。(a) C2-1: `idempotentRespawn` の doc で「decided first」が 2 回出る。`Spawn` の doc の手順 1 も「first」と「after」が同じ文にある。(b) C2-2 と V2-1: plain rejection の理由が、すでに inactive な座席に当てはまらない。AC1 の本文(digest の内側)と、quality-gates の 2 面の「Spawn rejected, recorded in manifest」に例外がない。(c) C2-3: `if !seat.Active` は外しても結果が変わらない。(d) C2-4: `orgWideLimitsHelp` の文の組み立てと、`--state-dir` と env の除外が後ろの文にしかないこと。直す文言を Trigger に書いた(`templates/base/ralph.toml` 側はこの回に直済み)。(e) C2-5: 行 144・170・172〜176 の report の番号と、`TestOrgCloseDeferredSelfPane_CloseFails_ReactivationCanExceedMaxTotalSeats` の上のコメントの「verify V-2」。3 つの report は cycle 1 を付録に同じ番号で残しているので、今は指す先が解ける |
| 178 行目 | cycle 2 の test の穴(T2-1〜T2-4)。(a) T2-1(mutation N8): idempotent の経路で、`max_orgs` と予約の判定の順を固定するテストがない。直すための subtest の内容を Trigger に書いた。(b) T2-2(N3): `if !seat.Active` を外す mutation は等価で、テストでは落とせない。(c) T2-3: help の文言を固定するテストがない。(d) T2-4: 古い台帳が手書きの 2 行で、CLI から古い台帳に `--reserve` を打つテストがない |

新しい 2 行は、コードを関数名とテスト名で指し、`file:line` を使っていない。`docs/tech-debt/README.md` の 173、176、177、178 行目に `[A-Za-z_./]*:\d+` を当てて 0 件、列は 5 つ、各列のバッククォートは偶数である。plan は `docs/plans/active/...` で指した(`/pr` の `archive-plan.sh` が書き換える)。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `/org` skill 4 面(`.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/` の 2 つ) | 変更なし。cycle 2 のコードの変更は、動いていない leader(古い ralph の台帳で、`spawned` のあとに `stopped` なしで `disbanded` が来たもの)への `--reserve` の拒否だけである。skill は `max_orgs` の拒否を「まだ走っていない org_id への `spawn` / `start`」と書き、`rejected` を書くのは「拒否された新しい座席の `spawn`」と絞っている(186 行目付近)。どちらも新しい枝と食い違わない。skill は idempotent の経路の内側を書いていないので、足す文もない。4 面の sha256 は `771be761…` で同じ |
| `docs/quality/quality-gates.md`(77 行目)、`templates/base/docs/quality/quality-gates.md`(76 行目) | 「Spawn rejected, recorded in manifest」は、立っている leader への `--reserve` の拒否には当たらない(入力検査の拒否と同じ例外)。verify が「任意」としたので足していない。177 行目の (b) に直し方を書いた |
| `README.md`(124 行目、247 行目) | verify が「変わっていない」と確認済み。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md`(41 行目) | 「`--config` がなければ、全 org の上限は main worktree のルートの `ralph.toml` から読む」は、詳細を計画に送る要約なので、verify の判断(V2-2)どおり変えていない |
| `internal/cli/org.go` の `orgWideLimitsHelp` | 変更なし(この回は Go の文字列を変えない)。直す文言は 177 行目の Trigger に書いた |
| `templates/base/ralph.toml` と `internal/config`、`scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh` | コメントだけの変更で、値は同じ。`go test ./internal/config/... ./internal/scaffold/...` が通った(`defaults_sync_test.go` を含む) |
| plan の Progress checklist と承認の digest | 157 行目だけを足した。`./scripts/plan-visual.sh digest` は `1a165903b5df` で、`- Approved:` の行(4 行目)と一致した |
| `.claude/rules/`、`docs/recipes/`、`docs/architecture/`、hook、script、language pack | この回の変更(plan の 1 行、`ralph.toml` のコメント、`AGENTS.md` の 1 行、tech-debt、report)が触れないので該当なし |

## Found but left

- quality-gates の 2 面の「recorded in manifest」に例外を足すこと(verify が「任意」としたもの)。177 行目の (b) に送った
- spec の Rollout の「1. 共通の台帳(FR-1、進行中)」(203 行目)は、cycle 1 から同じ。1 段目がマージされたあとも「進行中」のまま残っている。どの時点で外す慣例かが spec に書かれていないので、触っていない
- verify の V-4(`{{ENVELOPE}}` と doctor の要約が `max_seats` だけを出す)は、174 行目の (b) にあり、この回に変わっていない
- C2-1〜C2-4 の Go 側と C2-5 は直していない(上の Summary と 177 行目)

## Checks run

| Command | Result |
|---------|--------|
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | `PASS: all files in sync.`(IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/run-static-verify.sh`(tech-debt、plan、`ralph.toml`、`AGENTS.md`、self-review の編集のあと、report と insight の前) | rc 0。`check-sync.sh` PASS、`check-pipeline-sync.sh` OK、`check-skill-sync.sh` OK、`check-template-purity.sh` PASS、tech-debt README plan references OK、`gofmt: ok`、`0 issues.`、branch secret scan clean(`51855166..6ee9836b`) |
| `go test ./internal/config/... ./internal/scaffold/... -count=1` | `ok` が 2 パッケージ |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-08-org-limits-reserve.md` | `1a165903b5df`(`- Approved:` の行と一致) |
| `git diff --check` の self-review の編集(作業ツリー) | 出力なし |

commit のあとに走らせる `git diff --check 51855166...HEAD`(verify の V2-5 を直した確認)と `./scripts/secret-scan-branch.sh --strict` の結果は、呼び出し元への報告に書く。この report の commit が入った HEAD でしか確かめられないためである。

## Not verified

- `templates/base/ralph.toml` の新しいコメントは、`withMainWorktreeOrgLimits` と `ResolveOrgStateDir`、`MainWorktreeRoot` を読んで書いた。`--state-dir` で main の台帳を指して `ralph org spawn` を打ち、main の `ralph.toml` が読まれないことを実バイナリで確かめてはいない(self-review C2-4 と verify V2-2 も読みによる結論)。未確認です
- 177、178 行目の「直し方」は、self-review と test の提案を写したもので、実装して試してはいない。T2-1 の subtest は、test report が使い捨てで作り、HEAD で通って N8 で落ちたと書くもので、この回は再現していない(`docs/evidence/` の log は gitignore の対象で、手元にだけある)
- 「skill に idempotent の経路を書いた箇所がない」は、`org` skill の 4 面で `already spawned`、`idempotent`、`立っている`、`rejected` を `git grep` した範囲の確認である。別の言い方の記述は拾えていない可能性がある
