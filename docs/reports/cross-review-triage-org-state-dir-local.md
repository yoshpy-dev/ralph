# Cross-review triage report: org-state-dir-local

- Date: 2026-10-30 (cycle 1 and cycle 2)
- Plan: docs/plans/active/2026-10-30-org-state-dir-local.md
- Base branch: main (cycle 1: c03647f9; cycle 2: 1e266f96, after the merge of #289 in 668e9349)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: cycle 1 = 1, cycle 2 = 0
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0
- Note: the summary line reflects cycle 2 (the current state). Cycle 1 was ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0; it was fixed in S9 (6415aabd), the row is kept below as history and marked resolved, and the full pipeline re-ran as cycle 2.
- User decision: the user's standing instruction for this run (fix ACTION_REQUIRED without asking and re-run from /self-review), so cycle 1 took the non-cap "Fix" path and `cycle-count.json` went to 2.

## Triage context

- Active plan: docs/plans/active/2026-10-30-org-state-dir-local.md(A2: `RALPH_ORG_STATE_DIR` を撤去し、台帳を `--state-dir` と個人用の `ralph.local.toml` の `[org].state_dir` で決める。AC4: `ralph.toml` に `[org].state_dir` があると Load がエラーにする)
- Self-review report: docs/reports/self-review-2026-10-30-org-state-dir-local.md(cycle 1: `Merge: yes`、LOW 9、S7 で L1・L4・L6 を直した。cycle 2: S9 に所見なし、C2-L1 は /sync-docs で register に入れた、`Merge: yes`)
- Verify report: docs/reports/verify-2026-10-30-org-state-dir-local.md(cycle 1: pass、V3 は S8 で直した。cycle 2: pass、AC4 を大文字小文字を混ぜた 5 つの綴りで確かめた)
- Test report: docs/reports/test-2026-10-30-org-state-dir-local.md(cycle 1・cycle 2 とも pass)
- Implementation context summary: `checkLocalOnlyKeys`(`internal/config/local.go`)は `ralph.toml` を map に読み、`tomlDocHasKey` が区切りごとに引く。cycle 1 の時点では大文字小文字を区別して引いていたが、`toml.Unmarshal` で struct に読むと go-toml v2 はキーを大文字小文字を区別せずに対応させる。`LocalOrgStateDir` も struct で読むので、台帳を決める側は大文字小文字を区別しない。reviewer は codex(gpt-6-astra、xhigh、`-c sandbox_mode=read-only`、`--ignore-rules`)。sandbox が Go のビルドのディレクトリを作れず、reviewer 自身はどちらの cycle でもテストを打てなかった。
- Reproduction (triager, cycle 1): `go test -overlay` で一時のテストを足し(作業ツリーは変えていない)、`RALPH_LOCAL_CONFIG=off` で `[ORG]` / `STATE_DIR = "/ledger"` を書いた `ralph.toml` を `Load` すると、`err=<nil> state_dir="/ledger"` になった。

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| AR-1 (cycle 1, resolved in 6415aabd) | [P2] local-only の検査が go-toml の大文字小文字の扱いと合わない。`ralph.toml` の `[ORG]` / `STATE_DIR = "/ledger"` は検査をすり抜けて `cfg.Org.StateDir` に入る。`ralph config get org.state_dir` は `/ledger` を答えるが、実際のコマンドは黙って既定の台帳を使う。decoder と同じキーの扱いにし、大文字小文字を混ぜた table と key の回帰テストを足す。 | 実在した(triager が overlay のテストで再現)。AC4 の約束(`ralph.toml` に書くと Load が断る)が大文字小文字を変えただけで破れ、表示と実際の台帳が黙って食い違う。安く直せるので、ユーザーの指示どおり聞かずに直し、/self-review から回し直した。S9 は `tomlDocHasKey` を go-toml v2.3.0 の decoder(`structFieldPath`: 完全一致、なければ `strings.ToLower` で一致)と同じ規則にし、一致したキーをすべて辿る。cycle 2 の verify が HEAD の binary で 5 つの綴りを断ることを確かめた。 | internal/config/local.go:143-155 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 2 (2026-10-30, after the cycle 1 fix and the merge of #289)

- Reviewed HEAD: 4c9d975d, against main at 1e266f96 (78 files, +5070/−596). Same reviewer command as cycle 1. codex rc=0, the `-o` file is not empty, and the git state before and after the review is the same (`cmp`).
- Reviewer verdict, verbatim: "No actionable regressions were found in the configuration, ledger-resolution, or CLI changes. Go formatting and changed template mirrors checked out. Tests could not execute because the read-only sandbox blocked Go's temporary build directory."
- Findings: 0. Nothing to triage; Case C (proceed to /pr).
- Between the two reviews: S9 (6415aabd) fixed AR-1, and the full pipeline re-ran as cycle 2 (self-review a689bac7, verify fcbd5dbf, test 23af7887, sync-docs 5c01a2c9). Then origin/main (#289, 1e266f96) was merged in 668e9349; the only conflict was the end of the `docs/tech-debt/README.md` table (both sides kept). After the merge, go build, go vet, gofmt, check-skill-sync, and check-sync passed, and the Go tests of the touched packages were run again (config and cmd ok, cli ok; in org, `TestRegateCI_AnInterruptOnTheMergePath/while_the_push_runs` of #289 failed once under load and passed 3 of 3 alone; see the plan's Progress checklist).
- Known gap carried to the PR: the reviewer could not run tests in its sandbox in either cycle, so the reviewer's verdict rests on reading the diff; the tests are the pipeline's (/test cycle 1 and cycle 2) and the re-run after the merge.
