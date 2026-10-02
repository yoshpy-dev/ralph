# Test report: codex-config-model-gpt-5-6-sol

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-codex-config-model-gpt-5-6-sol.md
- Tester: tester subagent (Claude)、cycle 1(pipeline cap 2)
- Branch: chore/codex-config-model-gpt-5-6-sol、HEAD 7f8fd32c(push 済み)、base main 47345820
- Scope: 振る舞いのテストだけ。静的解析、formatter、linter、check-sync、TOML の構文だけの検査は verifier の担当なので走らせていない(AC-1〜AC-3 は verify report を参照)。コード、設定、テスト、plan は編集していない。mutation は `git archive HEAD` で作った scratch の複製だけで行い、worktree の追跡ファイルは変えていない(`git status --porcelain` は最後まで空)
- Evidence: `docs/evidence/verify-2026-10-02-100806.log`(default スコープの `run-test.sh`)、`docs/evidence/verify-2026-10-02-101225.log`(`RALPH_VERIFY_SCOPE=full`)。`docs/evidence/*.log` は gitignore のため commit には含まれない。scaffold、upgrade、mutation、codex の出力は repo の外の scratch(`.../scratchpad/i156b/test`)に置いた
- 隔離: scaffold と upgrade の probe はすべて `HOME=<scratch>/home`、`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`。利用者の `~/.codex` の設定と trust は読んでも書いてもいない。herdr server には触れていない。例外は `ralph doctor` が絶対パスで読む利用者の shell rc(読み取りのみ)と、step 5 の codex 実行が通常どおり書く session 記録

## この変更で変わったもの

`.codex/config.toml` と `templates/base/.codex/config.toml`(byte 一致)の `model` 3 か所が `gpt-5.5` から `gpt-5.6-sol` に変わり、コメントが書き直された。spec 2 本に注記が入った。Go と shell のコード、テストは変わっていない(`git diff --stat main...HEAD` は設定 2、spec 2、plan と report と insight)。このため、既存テストの件数と coverage は前回(drop-gpt-5-5-default-pool の cycle 1)と同じになるのが想定どおりで、実際に同じだった。

## Test execution

default の `./scripts/run-test.sh` が選ぶもの: `==> Language scope: full fallback (unclassified:.codex/config.toml)` → `Language packs selected: golang`。`./scripts/detect-changed-languages.sh` の直接実行は `scope=full`、`reason=unclassified:.codex/config.toml`、`docs_only=false`。`.codex/config.toml` が未分類のパスなので、変更が設定だけでも Go の package が全部走る。default でも `RALPH_VERIFY_SCOPE=full` でも実行内容は同じ(shell 32 suite + golang)。

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh`(default スコープ、フォアグラウンド、rc 0) | shell 32 suite + Go 8 package | shell 32/32 suite OK、Go 8/8 package ok | 0 | 0 | 3:54.6 |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`(フォアグラウンド、rc 0) | 同上 | shell 32/32 suite OK、Go 8/8 package ok | 0 | 0 | 3:47.5 |
| 上の full 実行の shell 出力の `PASS` 行の数(`FAIL` 行は集計の `FAIL: 0` が 22 行だけで、非 0 なし) | 1,450 | 1,450 | 0 | 0 | — |
| `go test ./... -count=1 -v -cover`(uncached、rc 0) | 1,397(subtest 込み) | 1,397 | 0 | 2(既存の `TestBaseFS_WithMockFS`、`TestAvailablePacks_WithMockFS`。mock embed.FS 前提) | cli 45.8s、org 11.1s、insights 3.3s、upgrade 2.6s、protocol 2.5s、scaffold 1.8s、driver 1.6s、config 0.8s(全体 47.7s) |
| `go test ./internal/scaffold/... ./internal/cli/... ./internal/upgrade/... -count=1 -v`(plan の指定、rc 0) | 836 | 834 | 0 | 2 | scaffold 0.4s、cli 40.6s、upgrade 1.5s |
| `sh tests/test-ralph-worktree.sh` × 3 | 143 × 3 | 143、143、143 | 0 | 0 | — |

注意点:
- `run-test.sh` の中の `go test ./...` は `-count=1` を付けないので、2 回とも `internal/org` 以外は `(cached)` だった。cached の結果を実行の証拠にしないよう、uncached の `go test ./... -count=1` を別に走らせた(上から 4 行目)。
- shell の `PASS` 行の数え方は、各 suite 末尾の集計ではなく出力全体の `PASS` で始まる行を数えた。前回 report の 1,427 は suite ごとの集計行の合計なので、数え方が違い、直接は比べられない(照合していない)。suite 数 32 と全 suite OK は前回と同じ。
- 前回 report にある既知の flake(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`、`TestRunWatcher_TimeoutIndependentOfSmallInterval`)は今回の実行では出なかった。この変更はテストを足していないので、新規テストの flakiness 検査は対象がない。

## Coverage

- Statement(`go test -cover`、uncached): cli 84.6%、config 92.3%、insights 86.1%、org 90.5%、org/driver 92.0%、org/protocol 97.9%、scaffold 75.7%、upgrade 91.2%。前回 report と全 package で同じ値
- Branch / Function: 計測していない(Go の標準ツールは statement のみ。shell は計装の手段なし)
- Notes: Go のコードを変えていないので、coverage が動かないのは想定どおり。設定ファイルの内容は coverage の対象外で、その内容を読む経路のテストは次の節と「Test gaps」に書く

## 設定ファイルの内容を使う経路(plan の Test plan の統合テスト)

### fresh scaffold

`go run ./cmd/ralph init --yes <scratch>/scaffold`(branch の build)は rc 0。

| 確認 | 結果 |
| --- | --- |
| `<scaffold>/.codex/config.toml` と `templates/base/.codex/config.toml` の `cmp` | byte 一致 |
| 同じく repo の `.codex/config.toml` との `cmp` | byte 一致 |
| TOML として読んだ `model`、`profiles.work.model`、`profiles.review.model` | 3 つとも `gpt-5.6-sol`(python3 `tomllib`) |
| scaffold 自身の `CI=true sh scripts/check-template.sh` | rc 0、`Template structure looks good.`、`FAIL` 0 件 |
| scaffold の `ralph doctor --strict`(branch の binary) | rc 0、`All checks passed.`。`Codex effective config: pass — hooks.json wired`(doctor が `.codex/config.toml` を TOML として読む check)、`Scaffold: core file hashes: pass`。`Shell aliases` の warn は利用者の shell rc にある `codex` の alias に由来し、この変更とは無関係 |

### ralph upgrade(main の scaffold → branch の upgrade)

main(47345820)の `git archive` から build した binary で scratch の project を作り(`.codex/config.toml` は `gpt-5.5` が 3 か所、sha256 先頭 8904b44b2c51)、branch の binary で `ralph upgrade` した。branch の template の sha256 先頭は 057f4911f4be。

version が同じ(`dev` と `dev`)の run と、binary に `-X main.Version=` で版を入れた run(main の build を v9.0.0、branch の build を v9.1.0)の両方を試した。実際の下流の upgrade は版が違うので、後者のほうが実際に近い。

| project | 事前の状態 | 結果 |
| --- | --- | --- |
| proj-a(dev → dev) | 未編集 | `--dry-run` は `update .codex/config.toml` の 1 行だけで、ディスクは変わらない。本番は rc 0、`updated: 1`、report `docs/reports/upgrade-dev-2026-10-02.md` の `### Updated` に `.codex/config.toml`。実行後の sha は branch の template と byte 一致、manifest の `hash` / `template_hash` / `disk_hash` が 3 つとも新しい値。2 回目の upgrade は `no-op`、rc 0(冪等) |
| ver-a(v9.0.0 → v9.1.0) | 未編集 | rc 0、`created: 0, updated: 1, deleted: 0`、report あり。`model` は 3 か所とも `gpt-5.6-sol` |
| ver-b(v9.0.0 → v9.1.0) | top-level の `model` を `my-pinned-model` に編集 | rc 3。stderr に `Unresolved drift (left untouched; see report for detail): ⚠ .codex/config.toml` と `ralph eject` / `ralph adopt` の案内。ファイルは編集後のまま(`my-pinned-model`、他の 2 か所は `gpt-5.5`)。report の `## Unresolved drift` の表に Recorded(8904b44b2c51…)、Disk(a623704f738a…)、New(057f4911f4be…)が出る。つまり upgrade は上書きせず、drift と、新しい template の hash を報告する |
| proj-b(dev → dev) | ver-b と同じ編集(top-level の `model` を `my-pinned-model` に) | rc 3、ファイルは未変更。版が同じだと `Upgrade no-op: dev (already up to date, zero writes)` の行が出て、report ファイルは書かれず(`no report written this run — tree already converged`)、drift は stderr だけに出る |
| proj-c(dev → dev) | 末尾にコメント 1 行を足した | proj-b と同じ(rc 3、未変更、stderr に drift) |

drift の解消経路: proj-b で `ralph eject .codex/config.toml` は rc 0(owner=core → fork)、以後の upgrade は rc 0 でファイルは変わらない。proj-c で `ralph adopt --yes .codex/config.toml` は rc 0 で、`gpt-5.6-sol` の template に戻り、次の upgrade は rc 0(`--yes` なしだと非対話で確認が `N` になり `Adopt aborted`、ファイル変更なし。scaffold は commit がないと `adopt` が `clean git work tree` を要求するので、scratch で `chore:` の commit を作った)。

proj-b と proj-c の版が同じ場合の `already up to date` の文言は、template に更新がある drift の path があっても出る。Go のコードは今回の branch で変わっておらず、この文言と report の出し分けは main と同じ。実際のリリース間(版が違う)では上の ver-b のとおり report が書かれ、drift が表に出る。

### #185 の書き換え検知(新しい設定の内容に対して)

`tests/test-ralph-worktree.sh` は自分で HEAD の内容を作るので、追跡している `.codex/config.toml` を読まない。plan の edge case「書き換えの検知が新しい値でも従来どおり見分けられる」を、実ファイルで確かめるために、scratch の git repo の HEAD に branch の `.codex/config.toml` を置き、`scripts/ralph-worktree.sh validate-clean-base main` を 8 通りで走らせた(main の config を HEAD にした対照も同じ 8 通り)。

| ケース | 期待 | branch | main(対照) |
| --- | --- | --- | --- |
| 変更なし | 成功 | pass | pass |
| コメントと空行を削る(観測された書き換え) | 書き換え専用の文面 | pass | pass |
| 削った上に `[shell_environment_policy]` を足す | 書き換え専用の文面 | pass | pass |
| `[shell_environment_policy]` を足すだけ | 書き換え専用の文面 | pass | pass |
| 削った上に top-level の `model` の値を変える | 一般の `has uncommitted changes` | pass | pass |
| `model` の値だけ変える | 一般の文面 | pass | pass |
| コメント行を 1 行足す | 一般の文面 | pass | pass |
| `git checkout` で戻した後 | 成功 | pass | pass |

16/16 pass。新しい config にも `"""` / `'''`(複数行文字列。あると検知が無効になる)は 0 個。

### 実際の codex の実行(任意項目)

この worktree で `command codex exec --sandbox read-only -o <file> '<ok とだけ返す指示>' </dev/null`(`-m` なし、alias 迂回、300 秒の watchdog)を 1 回。rc 0、`-o` の中身は `ok`、header は次のとおり。

```
model: gpt-5.6-sol
approval: never
sandbox: read-only
reasoning effort: low
```

project の設定が読まれて、最新の comment-only commit(cd73f252)の後でも `model: gpt-5.6-sol` になる。`warning: Ignored unsupported project-local config keys ...: profiles` も出ていて、plan の Deviation notes にある挙動(project の `[profiles.*]` は捨てられる)と一致する。ログには別に `failed to refresh OAuth tokens for server atlassian` が出たが、利用者側の MCP の認証で、この変更とは無関係。実行の前後で、main のチェックアウトの `git status --porcelain` は空、`.codex/config.toml` の sha と mtime は変わらず、worktree の設定の sha も変わらない。

この cycle で確認しなかったもの: `-m gpt-5.6-sol` に effort `max` を渡す要求と、`--profile work` / `--profile review` の実行(AC-4 / AC-4b の本体。implementer の log を verifier が読んで pass にしている)。ここで走らせた実行は、effort 未指定(`low`)の経路の再確認にあたる。

## Mutation testing(scratch の複製、tracked ファイルは未変更)

「テストはこの値を守っているか」を測った。root と template に同じ変更を当て(byte 一致は保つ)、`go test ./internal/scaffold/... ./internal/cli/... ./internal/upgrade/... -count=1` と shell の 3 本(`test-hook-wiring.sh`、`test-check-template.sh`、`test-ralph-worktree.sh`)を走らせた。

| ID | Mutation | 結果 |
| --- | --- | --- |
| m0 | 変更なし(対照) | Go 3 package ok、shell 3 本 rc 0。32 suite と Go 8 package の全部も rc 0 |
| m1 | top-level の `model` だけ `gpt-5.5` に戻す | 全部通る。**生存** |
| m2 | 3 か所とも `gpt-5.5` に戻す | 全部通る。**生存** |
| m3 | top-level の `model` の閉じ引用符を消す(TOML として不正) | 全部通る。32 suite + Go 8 package の全部でも rc 0。**生存**。同じ複製から build した binary の scaffold では `ralph doctor` が `Codex effective config: fail — invalid .codex/config.toml: toml: basic strings cannot have new lines` で止める |
| m4 | 3 か所を `gpt-5.6-so1`(存在しないモデル名)にする | 全部通る。**生存** |

4 件とも生存。m1、m2、m4 を止めるのは、verifier の `git grep`(AC-3)と TOML を読んだ中身の比較(AC-2)で、behavioral な test suite には止めるものがない。m3 は `ralph doctor` が利用者の環境では止めるが、repo の test suite には、同梱する `.codex/config.toml` を TOML として読むテストがない。root だけ、または template だけを戻す mutation は `scripts/check-sync.sh`(静的 gate、verify report で green)の担当で、この tester の範囲外のため実行していない。

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| なし | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| #185: 追跡している `.codex/config.toml` の外部書き換え(コメント剥がし + `shell_environment_policy` 追加)で worktree 作成が止まる | 新しい config の内容でも専用の文面で検知される。値の変更や行の追加は一般の文面に落ちる | 上の「#185 の書き換え検知」16/16、`tests/test-ralph-worktree.sh` 143/143 × 3 |
| 下流が手を加えた core 設定を `ralph upgrade` が黙って上書きする | 上書きしない(rc 3、drift として報告) | ver-b、proj-b、proj-c |
| project の `gpt-5.5` + 利用者の `model_reasoning_effort = "max"` で 400 | 今回の実行(effort は利用者の設定でなく `low`)では再現しない。`max` の要求は未再実行(implementer の log のみ) | codex の header、AC-4 |

## Test gaps

- **G-1(LOW、前からある。この変更が作ったものではない)**: 同梱する `.codex/config.toml` の中身を読む自動テストがない。`internal/scaffold/embed_test.go` と `internal/cli/cli_test.go` はこのファイルが scaffold に出ることだけを見る(`cli_test.go:43` の mock FS の `gpt-5.5` は任意の値で、plan の non-goal)。mutation の m1〜m4 が全部生存したのはこのため。モデル名の変更(今回のような設定の値の変更)は、`git grep` と TOML の比較という verifier の手作業の確認に守られているだけで、不正な TOML は `ralph doctor` が利用者の環境で止めるまで repo では見えない。案: `internal/scaffold` に、`templates/base/.codex/config.toml` を TOML として読み、`[features] hooks = true` と top-level の `model` が空でないことを見る test を足す。モデル名のリテラルを固定するテストにはしない方がよい(モデルが退役するたびに test も直すことになる)。この PR に入れるかは orchestrator の判断(tech-debt 行きで差し支えない)。
- G-2(情報、前からある): 版が同じ(`dev` → `dev`)の upgrade は、drift の path があっても `Upgrade no-op ... already up to date` と出し、report を書かない。drift は stderr と rc 3 には出る。版が違う実際のリリース間では report が書かれる(ver-b)。今回の変更でも Go のコードでもないので、finding には数えない。
- 未確認: (1) 「Codex 側の pipeline のエージェントがこのモデルを引き継ぐ」の実測(`.codex/agents/*.toml` にモデルの指定がないことまでしか見ていない。verify report と同じ)。(2) 実際に配布された版(v5.x)からの upgrade(ldflags で版を入れた build で代用した)。(3) CI の ubuntu 環境での実行(Go と shell のコードを変えていないので `TMPDIR=/tmp` の再実行はしていない)。(4) AC-4 の `max` の要求と AC-4b の `--profile` 2 通りの再実行。

## Verdict

- Pass: yes。shell 32/32 suite OK(`PASS` 行 1,450、`FAIL` 0)、Go 8/8 package ok(1,397 PASS / 0 FAIL / 2 既存の SKIP)、指定の 3 package は 834 PASS / 0 FAIL。`tests/test-ralph-worktree.sh` は 3 回とも 143/143。fresh scaffold の config は byte 一致で、TOML として読めて 3 つの値が `gpt-5.6-sol`、scaffold 自身の `check-template.sh` と `ralph doctor --strict` は rc 0。upgrade は未編集の project では置き換え、編集済みの project では上書きせず drift を報告した。#185 の検知は新しい内容でも従来どおり。実 codex の header は `model: gpt-5.6-sol`。AC-5 の behavioral な半分は pass。`/pr` に進めてよい。
- Fail: no
- Blocked: no
