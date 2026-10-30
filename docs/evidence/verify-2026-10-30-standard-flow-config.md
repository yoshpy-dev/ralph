# verify evidence: standard-flow-config(cycle 1)

- Date: 2026-10-30(JST 20:16〜20:35)
- HEAD: f1685e87(merge-base 0351800d = origin/main)
- 全出力: `docs/evidence/verify-2026-10-30-standard-flow-config.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-static-verify 自身のログは `docs/evidence/verify-2026-10-30-111611.log`(同じく対象外)。
- 下の `<S>` はセッションの scratchpad の `v276b1/`、`<WT>` は task worktree。probe の script は `<S>/ac1.sh`、`ac3.sh`、`ac5.sh`、`ac6.sh`、`init.sh`。
- binary: `go build -o <S>/ralph ./cmd/ralph`(HEAD、go1.27.3、rc 0)。PATH の ralph は 6.0.0(44c14b97)で `config get` がない。

## 1. 静的解析

```
./scripts/run-static-verify.sh            rc=0
  Requested scope: changed
  Language scope: full fallback (unclassified:scripts/ralph-config.sh)
  local verifier(shellcheck --severity=warning、sh -n、Codex の hook の検査) OK
  check-sync: IDENTICAL 172 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 10 / KNOWN_DIFF 6 → PASS
  check-pipeline-sync OK、check-skill-sync 14 skill(s) in lock-step、check-template-purity PASS
  golang: gofmt: ok / go vet 無出力 / golangci-lint 0 issues. / staticcheck 無出力(Skipping の行なし)
  secret-scan-branch: scanned 0351800d..f1685e87 against origin/main: clean
go build ./...                            rc=0
git diff --check origin/main...HEAD       空
変えたファイルの U+FFFD                     0 件
plan-visual.sh digest <plan>              d8e03ef62de9(`- Approved:` と同じ)
```

shellcheck を既定の severity で直接打った結果(info を含む):

| ファイル | 結果 | origin/main |
|---|---|---|
| `scripts/ralph-config.sh` と template の写し(`cmp` で一致) | rc 0 | — |
| `tests/test-ralph-config.sh` | rc 0 | — |
| `tests/test-codex-exec-invocation.sh` | SC2016(info)5 件、`--severity=warning` で rc 0 | SC2016 4 件 |
| `tests/test-insights-append.sh` | SC2329(info)1 件、`--severity=warning` で rc 0 | 同じ 1 件 |

## 2. AC1 `ralph config get`(一時の git repo、`RALPH_LOCAL_CONFIG` は外した)

```
== 1. no ralph.toml, no ralph.local.toml (defaults)
  standard.max_pipeline_cycles   rc=0 out=2
  reviewer.claude_model          rc=0 out=fable
  reviewer.claude_effort         rc=0 out=xhigh
  reviewer.codex_model           rc=0 out=gpt-6-astra
  reviewer.codex_effort          rc=0 out=xhigh
== 2. ralph.toml sets all five (4 / opus / high / gpt-6-sol / medium)
  → 4 / opus / high / gpt-6-sol / medium(すべて rc=0)
== 3. ralph.local.toml overrides three of them (max_pipeline_cycles=7, claude_model=sonnet, codex_effort=low)
  → 7 / sonnet / high / gpt-6-sol / low(キーごとに上書き)
== 3b. linked worktree から: 7 / sonnet / high(主 worktree の root の個人用のファイル)
== 3c. RALPH_LOCAL_CONFIG=off: 4 / opus(個人用のファイルを飛ばす)
== 3d. 4 つの撤去した変数を 9/haiku/gpt-x/minimal に設定: 7 / sonnet / high / gpt-6-sol / low(変わらない)
== 4. validation errors (ralph.toml)、すべて rc=3、stderr の 1 行目
  max_pipeline_cycles=0    [standard].max_pipeline_cycles must be >= 1, got 0
  max_pipeline_cycles=-1   [standard].max_pipeline_cycles must be >= 1, got -1
  claude_effort=minimal    [reviewer].claude_effort "minimal" must be one of low, medium, high, xhigh, max
  claude_effort=XHIGH      [reviewer].claude_effort "XHIGH" must be one of ...
  claude_effort ""         [reviewer].claude_effort "" must be one of ...
  claude_model ""          [reviewer].claude_model "" must be non-empty and hold only A-Z, a-z, 0-9, '.', '_', ':', '@', '/', '+', and '-'
  codex_model ""           [reviewer].codex_model "" must be non-empty ...
  codex_effort ""          [reviewer].codex_effort "" must be non-empty ...
  claude_model "opus$HOME" [reviewer].claude_model "opus$HOME" must be non-empty ...
  codex_model "gpt 6"      [reviewer].codex_model "gpt 6" must be non-empty ...
  codex_effort "x'h"       [reviewer].codex_effort "x'h" must be non-empty ...
  claude_model "a`id`"     [reviewer].claude_model "a`id`" must be non-empty ...
  claude_model "fable\n"   [reviewer].claude_model "fable\n" must be non-empty ...
  codex_model "a;b"        [reviewer].codex_model "a;b" must be non-empty ...
  (どの行も "config get: ralph.toml: cannot be read or is invalid: " で始まる)
== 5. validation errors in ralph.local.toml (ralph.toml なし)、rc=3
  max_pipeline_cycles=0    ... got 0 (with <S>/ac1repo/ralph.local.toml applied)
  codex_model "$x"         [reviewer].codex_model "$x" must be non-empty ... (with ... ralph.local.toml applied)
== 6. full IDs with : @ / + pass (rc=0)
  reviewer.claude_model    us.anthropic.claude-opus-5-5-v1:0
  reviewer.codex_model     arn:aws:bedrock:us-east-1:acct:inference-profile/us.anthropic.claude-x@2026+beta
  reviewer.codex_effort    x_high.2
== 7. unknown key and table
  reviewer.bogus rc=2 err=config get: reviewer.bogus: not a key of one value in ralph.toml: no such key
  reviewer       rc=2 err=config get: reviewer: not a key of one value in ralph.toml: it is a table
```

## 3. AC2・AC3 `ralph_setting`(偽の ralph、`env -i`、PATH は `<偽の ralph>:/usr/bin:/bin`、4 つの撤去した変数を毎回 9/haiku/gpt-x/minimal に設定)

`/bin/sh` の結果(出力 `値|戻り値`、stderr の改行は `~`):

```
ok       reviewer.codex_model          -> out=fake-reviewer.codex_model|0 stderr=
empty    reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 0 (no value); using the default gpt-6-astra~
none     reviewer.codex_model          -> out=gpt-6-astra|0 stderr=                      (PATH に ralph なし = 127)
old      reviewer.codex_model          -> out=gpt-6-astra|0 stderr=                      (exit 1 + unknown command "config")
other1   reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 1 (Error: something else); using the default gpt-6-astra~
e2       reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 2 (config get: reviewer.codex_model: not a key of one value in ralph.toml: no such key); using the default gpt-6-astra~
e3       reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 3 (config get: ralph.toml: cannot be read or is invalid: boom); using the default gpt-6-astra~
e3quiet  reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 3; using the default gpt-6-astra~
sig      reviewer.codex_model          -> out=gpt-6-astra|0 stderr=ralph-config.sh: ralph config get reviewer.codex_model exited 143; using the default gpt-6-astra~
none     standard.max_pipeline_cycles  -> out=2|0 stderr=
none     reviewer.claude_model         -> out=fable|0 stderr=
none     reviewer.claude_effort        -> out=xhigh|0 stderr=
none     reviewer.codex_effort         -> out=xhigh|0 stderr=
ok       reviewer.bogus                -> out=|2 stderr=ralph-config.sh: unknown setting reviewer.bogus~
```

- bash と `zsh -f` は sh と同じ出力。dash は `sig` の行だけが違い、`exited 143 (Terminated: 15)` になった(dash が偽の ralph の stderr に書く 1 行を helper がそのまま拾った)。
- `set -eu` の呼び出し側で exit 3 のあとも続く: `after set -e: xhigh`(stderr に 1 行)。
- 偽の ralph に渡った引数は `config get reviewer.codex_model` だけ。
- HEAD の binary を PATH に置き、`RALPH_LOCAL_CONFIG=off` と 4 つの変数を設定して worktree の root で読むと、2 / fable / xhigh / gpt-6-astra / xhigh、stderr は空。
- PATH の 6.0.0 で読むと `fable`、stderr は空(黙って既定値)。
- Bash tool の shell(zsh 5.9、ユーザーの設定を読んだまま)で `/plan` 11.c の 1 行目を打った: 6.0.0 で `codex model=gpt-6-astra effort=xhigh`、HEAD の binary を PATH の先頭に置いて `claude model=fable effort=xhigh cap=2`。
- 一時ディレクトリの後片付けを TMPDIR で数えようとしたが、macOS の `mktemp -d` は TMPDIR を見ない(`/var/folders/...` に作った)ので、この数は意味がない。後片付けはコード(`scripts/ralph-config.sh:76` の `rm -rf` がどの return より前)と `tests/test-ralph-config.sh` の偽の mktemp の照合で見る。

## 4. AC5 cross-review の jq(`.claude/skills/cross-review/SKILL.md` から grep で取り出したコマンドをそのまま `sh -c` で打った、jq-1.7.1-apple)

```
read  cmd: f=.harness/state/standard-pipeline/cycle-count.json; jq -r '.cap | select(type == "number" and . >= 1 and . == floor)' "$f"
write cmd: f=.harness/state/standard-pipeline/cycle-count.json; jq -c '.cap = .cycle + 1' "$f" > "$f.tmp" && mv "$f.tmp" "$f" || rm -f "$f.tmp"; jq -c . "$f"

step 1.d read(cycle 2 の file に cap を置いた)
  cap なし   → []      cap 3    → [3]     cap "3"  → []      cap 2.5 → []
  cap 0      → []      cap -1   → []      cap null → []      cap true → []
  cap 3.0    → [3.0]   cap 1e1  → [1E+1]
  壊れた JSON → [] rc=5(parse error)     file なし → [] rc=2
step 9 write
  {"cycle": 2}            → printed {"plan_path":…,"cycle":2,"cap":3}、file も同じ、tmp は残らない
  {"cycle": 3, "cap": 3}  → printed …"cycle":3,"cap":4
  {"cycle": "2"}          → jq: string ("2") and number (1) cannot be added。file は元のまま、printed に cap なし、tmp は残らない
  {"plan_path": …}(cycle なし)→ printed …"cap":1
  壊れた JSON             → printed なし rc=5、file は元のまま
  jq が PATH にない       → printed なし rc=127、file は元のまま、tmp は残らない
round trip: cycle 2 に write のあと read → [3]
```

## 5. AC6 `ralph doctor`(一時の git repo、`RALPH_LOCAL_CONFIG=off`、`--state-dir <S>/ac6state`、4 つの変数だけを変えた 4 回の実行の diff)

```
== diff unset vs empty → no difference
== diff unset vs set → 4 行が増えた(17a18,21)
  ⚠ RALPH_STANDARD_MAX_PIPELINE_CYCLES: warn — set to "3", but ralph no longer reads it, once `ralph upgrade` has replaced the project's skills and scripts/ralph-config.sh, which read it until then; write the value in [standard].max_pipeline_cycles of ralph.toml, or of ralph.local.toml (next to ralph.toml in the main worktree root, ignored by git) for this machine only, or, to raise the cap of one task only, have /cross-review write cap in that task's .harness/state/standard-pipeline/cycle-count.json when it reaches the cap, and unset RALPH_STANDARD_MAX_PIPELINE_CYCLES
  ⚠ RALPH_CLAUDE_REVIEWER_MODEL: warn — set to "opus", but ralph no longer reads it, …; write the value in [reviewer].claude_model of ralph.toml, or of ralph.local.toml (…) for this machine only, and unset RALPH_CLAUDE_REVIEWER_MODEL
  ⚠ RALPH_CODEX_REVIEWER_MODEL: warn — set to "gpt-6-sol", … [reviewer].codex_model …, and unset RALPH_CODEX_REVIEWER_MODEL
  ⚠ RALPH_CODEX_REASONING_EFFORT: warn — set to "high", … [reviewer].codex_effort …, and unset RALPH_CODEX_REASONING_EFFORT
== diff unset vs one(RALPH_CODEX_REASONING_EFFORT だけ)→ その 1 行だけ増えた
== rc: unset 0、set 0。set の実行に ✗ の行はない
```

## 6. scaffold と 6.0.0 の読み込み

```
ralph init --yes .(HEAD、一時の git repo、RALPH_LOCAL_CONFIG=off)  rc 0
ralph.toml: 34:[standard] 50:# max_pipeline_cycles = 2 68:[reviewer] 69-72: 4 つのキーは注釈
同梱の helper で読む(HEAD の binary が PATH): 2 / fable / xhigh / gpt-6-astra / xhigh
5 つの注釈を sed で外して 3 / opus / high / gpt-6-sol / medium にする → helper で 3 / opus / high / gpt-6-sol / medium
同じ ralph.toml で PATH の ralph 6.0.0 の doctor: rc 0、"All checks passed."、ralph.toml の warn も parse error もない
```

## 7. grep

- 計画の S6 の否定の grep(4 つの変数の `${` の形、`.claude .agents scripts templates internal tests` の md・sh・go・toml): 当たりは `tests/test-ralph-config.sh:225` の 1 件だけ。source のあとに 4 つの変数が設定されていないことを `${VAR-unset}` で確かめるテストの行で、読み出しではない。
- 4 つの変数の素の `$VAR` の形(`.claude .agents scripts templates internal tests packs .codex`): 0 件。
- skill・scripts・leader の雛形に残る `${RALPH_…:-…}` は `RALPH_PRIMARY_CLI`(内部の受け渡し)、B2 の変数(`RALPH_XREVIEW_BASE`、`RALPH_VERIFY_BASE`、`RALPH_VERIFY_SKIP_BRANCH_SECRET_SCAN`。plan-visual の 2 つはこの grep の対象から外した)、`RALPH_VERIFY_SCOPE`・`RALPH_SECRET_*`・`RALPH_COMMON_SOURCED`・`RALPH_PURITY_SCAN_ROOT` だけ。
- `internal/org/prompts/leader.md` に `ralph-config`、`ralph config get reviewer`、4 つの変数名: 0 件。
- 4 つの変数名の `git grep`(docs/plans・docs/reports・docs/specs・docs/evidence・docs/insights を除く): rules の 2 か所(model-routing.md の「Where the values live」、post-implementation-pipeline.md:118)とその template の写し、`scripts/ralph-config.sh:9-11` と写しは「もう読まない」の説明。`docs/tech-debt/README.md:134`・`:162`・`:215` は過去の run の記録。ほかはテストと doctor の表。
- cross-review の 2 つの jq は 4 つの写しにそれぞれ 1 回ずつ、implement 3.c の `its \`cap\` included when it has one` も 4 つの写しに 1 回ずつ。`/pr` は `.claude/skills/pr/SKILL.md:60` で cycle-count.json を `rm -f` する。

## 8. S8・S9 の確認(2026-10-30 JST 20:57〜21:10)

- 差分: `git diff 6848f496..e430ac4c`(`internal/cli/` の 4 ファイル、+196/−26)。HEAD は 0b713b43(505f4f5b は計画の Progress、0b713b43 は self-review report だけ)。
- binary: `go build -o <S>/ralph-s9 ./cmd/ralph`(HEAD 0b713b43、rc 0)。help は `<S>/s9-config-help.txt` と `<S>/s9-start-help.txt` に取った。

```
uptime(go の直前)          load 7.84、5.99
go build ./...              rc=0
go vet ./internal/cli/      rc=0(無出力)
golangci-lint run ./internal/cli/   0 issues.  rc=0
gofmt -l internal/cli/      無出力
git diff --check 6848f496..HEAD     空
変えたファイルの U+FFFD       0 件
plan-visual.sh digest <plan>        d8e03ef62de9(変わらない)
git status --porcelain(前後)       空
```

`ralph config get --help` の末尾(S8 で足した段落):

```
These are for a skill or seat that runs ralph config get itself, as for
the review switches of [standard] and [org.review]. The skills read
standard.max_pipeline_cycles and the four keys of [reviewer] with
ralph_setting of scripts/ralph-config.sh instead, which does not stop:
when it gets no value it goes on with the key's built-in default,
silently when ralph is not on PATH or is an older ralph without this
verb, and after one line on stderr otherwise, exits 2 and 3 included.
```

`ralph org start --help` の該当の段落(`...` は省略、最後の文は改行を詰めた):

```
... The org's approval mode, [org.approval].mode (per-plan by
default, or split), is read from the same ralph.toml as max_orgs and
max_total_seats below, ... The four keys of [reviewer], the model and
effort of the reviewers the leader's plan advisory calls, are read
from the same ralph.toml as max_orgs and max_total_seats below and
written in the leader's task as two lines, one for codex and one for
claude: the leader uses them as they were when the org started and
does not read them itself.
...
Those two are read from the main worktree's
ralph.toml (built-in defaults when it has none) when --config is not
given and the ledger is the main worktree's .harness/state/org, ...
In every other case (--config, a --state-dir or [org].state_dir naming another
ledger, a ledger found from the git toplevel, no git repository) the --config
file, else ./ralph.toml, is used.
```

`ralph org` の persistent flag `--config` の usage(`internal/cli/org.go:60`、この PR では変わっていない)は「without --config, spawn and start read the org-wide limits (see ralph org spawn --help), escalate and director attach read [org.director].enabled, …」と、主 worktree の ralph.toml から読むものを列挙する。start の `[org.approval]`(前から)と `[reviewer]`(S9)は列挙にない。

skill と雛形が `ralph config get` を自分で打つキー(`git grep -ohE 'ralph config get [a-z_.]+'`、`.claude/skills`・`internal/org/prompts`・template の skill・`scripts`):

```
   2 ralph config get org.gate.regate_checks   (.claude/skills/director/SKILL.md:964 ほか写し)
   2 ralph config get org.review.cross_review
   1 ralph config get org.review.plan_advisory
   2 ralph config get standard.cross_review
   4 ralph config get standard.plan_advisory
```

`README.md:135` の `ralph config get` の行は今も「They stop and report any other failure, exit 2 and 3 included.」と書く(計画の Progress が /sync-docs に回している)。
