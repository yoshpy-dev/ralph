# Verify report: review-switches-skills

- Date: 2026-10-29
- Plan: `docs/plans/active/2026-10-29-review-switches-skills.md`(issue #266、AC1〜AC9、D1〜D5、Progress の S1〜S4・F1・N1)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff 9837f744...HEAD`(HEAD fd5daaaf、16 コミット、59 ファイル、+862/-292)。self-review の run 2(Merge: OK)と、N1 のインラインの直し(e2d842cd)のあとの全体。`go test` とシェルのテストは走らせていない(/test の担当)。
- Evidence: `docs/evidence/verify-2026-10-29-review-switches-skills.log`(gitignore の対象でコミットしない。`run-static-verify.sh` の全出力、3 つの検査の個別実行、probe P1〜P8 の出力)。wrapper 自身の保存先は `docs/evidence/verify-2026-10-28-225814.log`(UTC の日付)。
- Insight event: `docs/insights/events/2026-10-28-review-switches-skills.jsonl` に 1 行(phase verify、`--cycle auto`、verdict pass、low 4)。file 名は UTC の日付で、self-review の 2 行と同じ file。

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 `/plan` 手順 11 が相手のモデルを呼ぶ。driver は skill を動かす CLI で `RALPH_PRIMARY_CLI` が上書きし、`which` で推さない。claude の呼び出しは読む道具だけ、許可の要る操作を断り、MCP と project の hook を止め、`RALPH_WIP_CHECKPOINT=off` を渡す。モデルは `RALPH_CLAUDE_REVIEWER_MODEL`(fallback `opus`)。名前の改め。S2 の小さな実行 | 満たす | `.claude/skills/plan/SKILL.md:73`(名前「Other-model plan advisory」)、`:79-83`(driver の規則、`which` を使わないと明記、1 回の Bash の snippet)、`:86-91`(codex の形は従来のまま)、`:93-98`(`RALPH_WIP_CHECKPOINT=off command claude -p --model "${RALPH_CLAUDE_REVIEWER_MODEL:-opus}" --restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk --output-format text`)。`claude --help`(2.1.304)にこの 6 つの flag がある(P6)。S2 の小さな実行は implementer の scratchpad の記録を読んだ(再実行はしていない、P7): feature branch `feat/x` に未コミットの変更・staged・未追跡を置いた状態で、前後の git の状態が `cmp` で一致、init event の道具は Glob・Grep・Read だけ、`mcp_servers` は空、出力は計画の中身(greet.sh の AC)に触れた所見。手順 12 の「Codex findings」も改めた(`:110`、`:116`) |
| AC9 `.codex/AGENTS.override.md` と写しが `/plan` と cross-review の前に `RALPH_PRIMARY_CLI=codex` を渡す。`/cross-review` 手順 2 の判別が AC1 と同じ。claude の道に `RALPH_WIP_CHECKPOINT=off` と前後の git の確かめ。triage の雛形に `skipped (<reason>)` | 満たす | `.codex/AGENTS.override.md:24-28`(`/plan` と post-implementation pipeline の前に設定)、`templates/base/.codex/AGENTS.override.md` と `cmp` で一致。`.claude/skills/cross-review/SKILL.md:52-58` の driver の規則と snippet は `/plan` 11.b と同じ文字列(下の一致の確かめ)。`:75`(claude の道の `RALPH_WIP_CHECKPOINT=off`)、`:66` と `:76`(前後の git の状態)。`docs/reports/templates/cross-review-triage-report.md:9` と写しの `:9` |
| AC2 最初に repo の root で `ralph config get` を打ち、読み方の約束に従う。飛ばした理由を計画の Progress と triage の report に残し、`/pr` が本文に書く | 満たす | `/plan` 11.a(`.claude/skills/plan/SKILL.md:74-78`)と `/cross-review` 手順 0(`.claude/skills/cross-review/SKILL.md:30-36`)の分岐は、0 の true と false、1 と `unknown command "config"`、127、それ以外は止まる、の 4 つで、キー名以外は同じ文。記録先は `/plan` が `Plan reviewed` の項目(`:73`)、`/cross-review` が `Reviewer status: skipped (<reason>)`(`:115-128`)。`/pr` は手順 5 の対応表(`.claude/skills/pr/SKILL.md:34`)と雛形の 2 行(`.claude/skills/pr/template.md:33-34`)。終了コードは実測と合う(P1、P2) |
| AC3 on で相手の CLI がないときは飛ばして記録する。`/cross-review` は report を書く形 | 満たす | `/plan` 11.b の後半(`.claude/skills/plan/SKILL.md:83`、`plan advisory: skipped (<reviewer> not available)`)。`/cross-review` 手順 3(`:61`、手順 6 で skipped の report)、手順 6 の「Skipped run」(`:128`)、手順 9(`:179`) |
| AC4 backfill が `Reviewer status: skipped …` の cycle を n/a として読む(テスト) | 満たす(静的) | `internal/insights/backfill.go:456-459`(小文字にして `incomplete` か `skipped` で始まれば `pendingNoReview`)、`:159-163`(n/a)。テストは `internal/insights/backfill_test.go:690` と `:707`(大文字の `Skipped` と cycle 2 だけの skipped を含む)。`reviewerStatusRe`(`backfill.go:431`)は先頭の語だけを取るので、`skipped (off: …)` の括弧は影響しない。実行は /test |
| AC5 ルートの `ralph.toml` を知らないキーを拒む読み方で読み、4 つが true。キーを改名した写しで落ちる | 満たす(静的) | `internal/config/root_ralph_toml_test.go:32`(`DisallowUnknownFields`、ゼロ値の `Config` に読む)、`:40`(`StrictMissingError` からキーと行を出して Fatal)、`:47-58`(4 つの switch)。ルートの `ralph.toml:9-15` は 4 つとも true。改名・false・削除の写しで落ちたことは Progress の S1 の記録で、この verify では走らせていない(/test) |
| AC6 plan skill の `RALPH_CLAUDE_REVIEWER_MODEL` の fallback と `scripts/ralph-config.sh` の照合が `TestDefaultsLockStep` にある | 満たす(静的) | `internal/config/defaults_sync_test.go:334`。照合は file の全出現を見る(`:352-363`)。plan skill の出現は 1 か所(`.claude/skills/plan/SKILL.md:96`、値 `opus`)で、`scripts/ralph-config.sh:35` の既定 `opus` と同じ |
| AC7 「optional」の文が Scope の文書で「設定が on で相手の CLI があるときに走る、そうでなければ飛ばして記録する」の意味 | 満たす(残りは下の V1〜V4) | `git grep` で「cross-review・advisory・codex・reviewer の近くの optional」は、履歴(`docs/plans/archive`、`docs/reports`、`docs/insights`、`docs/evidence`)を除いて 0 件。`Codex plan advisory`・`Codex advisory`・`silently skip`・`auto-detect` の残りは、過去の計画の所見番号を引く Go のコメント(`internal/org/*.go`、`internal/cli/migrate*.go` など)、codex の hook の話(`.codex/README.md:157`、`docs/recipes/codex-setup.md:32`)、別の話題の `silently skip`、それと /sync-docs 待ちの `docs/tech-debt/README.md:31`・`:154` |
| AC8 `check-skill-sync.sh`、`check-sync.sh`、`go test ./internal/config/ ./internal/insights/`、skill の文を固定するテストが通る | 一部(静的の半分) | `check-skill-sync.sh` rc 0(14 skill)、`check-sync.sh` rc 0(DRIFTED 0)。`go test` とシェルのテストは /test。削った文言(`Codex not available`、`Claude CLI not available`、`auto-detect`、`origin/$BASE`、`optional, inline`)を固定するテストは `tests/`・`internal/` にない(`git grep`) |
| issue #266 受け入れ条件 1: driver が Codex なら `/plan` が claude を呼ぶ | 満たす(skill の文) | AC1。snippet を zsh で走らせ、running=codex なら reviewer=claude(P4) |
| issue #266 受け入れ条件 2: off なら飛ばして記録 | 満たす | AC2 |
| issue #266 受け入れ条件 3: on で相手の CLI がなければ飛ばして記録 | 満たす | AC3 |
| issue #266 受け入れ条件 4: 写しの同期の検査と fallback の同期のテストが通る | 一部 | 写しの検査は rc 0。fallback のテストの実行は /test |

### 一致の確かめ(依頼の 3)

- `/plan` 11.a と `/cross-review` 手順 0: 終了コードの 4 つの bullet が、記録の文言(`off: [standard].<key>`、`ralph without config get; treated as off`、`ralph not found; treated as off`)を含めてキー名以外は同じ(`.claude/skills/plan/SKILL.md:75-78`、`.claude/skills/cross-review/SKILL.md:31-34`)。相手の CLI がない理由も `<reviewer> not available` で同じ(plan `:83`、cross-review `:61`)。
- `/plan` 11.b と `/cross-review` 2.b の snippet: `grep` で取り出した 1 行が 2 つの file で同じ文字列(evidence では `uniq -c` が 2)。plan 側の行をそのまま取り出して `<claude|codex>` を置き換え、zsh で 6 通り走らせた(P4)。上書きなしなら running の CLI の反対、`CODEX` は小文字にして driver=codex、`gemini` は無視される。
- git の状態のコマンド: `{ git rev-parse HEAD; … xargs -0 git hash-object --; }` が `/plan` 11.c と `/cross-review` 手順 4 で一字違わず同じ(P5)。未追跡がないときに GNU の `xargs` が引数なしで打つ `git hash-object --` も rc 0 で何も出さない(BSD の `xargs` は打たない)。どちらでも前後で同じ出力になる。
- claude の prompt の雛形: `<base>` を手順 4 が表示した値に置き換える形(`.claude/skills/cross-review/SKILL.md:225-239`)。手順 4 の確かめ(`:65`)を bash・zsh・sh で走らせると `base=main rc=1` を出す(P3)。前の版の `origin/$BASE` から、codex の道(`--base "$BASE"`)と空の差分の確かめと同じ local の base に変わった。
- `/pr` の対応表: `/plan` が記録する 4 つ(`skipped (<reason>)`、`incomplete (<reason>)`、`no findings (<reviewer>)`、`ran with <reviewer>`)と、`/cross-review` が書く 3 つ(`complete`、`incomplete (<reason>)`、`skipped (<reason>)`)が、`.claude/skills/pr/SKILL.md:34` の対応ですべて雛形の 4 値(`ran (<reviewer>)`、`incomplete (<reason>)`、`skipped (<reason>)`、`not recorded`)のどれかになる。
- triage の雛形: `- Reviewer status: <complete | incomplete (<reason>) | skipped (<reason>)>`(root と `templates/base/` の両方の `:9`)。
- `.codex/AGENTS.override.md:24-28` と写し: `/plan` と post-implementation pipeline の前に `RALPH_PRIMARY_CLI=codex`。
- 4 つの写し: `plan`・`cross-review`・`pr`・`implement` の `SKILL.md` と `pr/template.md` は、`.claude/` と `templates/base/.claude/`、`.agents/` と `templates/base/.agents/` がそれぞれ `cmp` で一致。

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0(All verifiers passed) | `.codex/config.toml` が分類の外なので全体に広がった(Language scope: full fallback)。shellcheck、hook の `sh -n`、guard の awk、`settings.json` の `jq -e`、Codex の 3 つの guard、check-sync、check-pipeline-sync(6 file)、check-skill-sync、check-template-purity、tech-debt の計画参照、golang(gofmt ok、go vet は出力なし、golangci-lint 0 issues、staticcheck は出力なし)、branch の secret scan(9837f744..fd5daaaf、clean) |
| `./scripts/check-sync.sh`(単独) | rc 0 | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 10、KNOWN_DIFF 6 |
| `./scripts/check-skill-sync.sh`(単独) | rc 0 | 14 skill が lock-step |
| `./scripts/check-template-purity.sh`(単独) | rc 0 | meta-repo の参照なし |
| `sh -n` と `cmp`(変更したシェル 5 つ) | rc 0 | `scripts/ralph-config.sh`、`scripts/xreview-helpers.sh` と両方の写し、`tests/test-codex-exec-invocation.sh`。写しは一致。`shellcheck -S warning` を 2 つの script に打って rc 0 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `README.md:135`(`ralph config get` の行) | No(V1、LOW、新規) | 「The skills and seats … treat only an older ralph without this command (exit 1 …) as off」と言い切る。この PR の skill(D1)は exit 127 も off として飛ばすので、「only」が skill の動きと食い違う。`templates/base/ralph.toml:23-29` は F1 で 127 を足してあり、README だけが残った |
| `docs/tech-debt/README.md:31` | No(V2、LOW、/sync-docs 待ち) | 「the only surviving caller is `/cross-review`'s auto-detect」は事実と違う(呼び手は `/plan` 11.b と `/cross-review` 2.b の 2 つで、auto-detect はなくなった)。行番号の `scripts/xreview-helpers.sh:66-73`(2 か所)も古い。今の `pick_reviewer` は `:70-78`(base でも `:69-77` で、すでにずれていた) |
| `docs/tech-debt/README.md:154` | No(V3、LOW、/sync-docs 待ち) | 「`/plan`'s Codex advisory」の名前と、(c)「the reverse direction's claude reviewer … has not been reviewed」が古い。`/plan` の claude の呼び出しはこの PR で道具・MCP・hook を止める形になり、`/cross-review` の claude の道は env と git の確かめだけが入った(self-review の L5) |
| 飛ばす条件を数え上げる文(空の差分を言わない) | 一部(V4、LOW、/sync-docs 待ち) | 「〜のとき走る」と十分条件として書く文は、空の差分のときに外れる: `.claude/skills/implement/SKILL.md:59`(4 つの写し)、`.claude/rules/ralph/ralph-workflow.md:38-44`(と写し)、`docs/recipes/codex-setup.md:64-66`(と写し)。`docs/quality/definition-of-done.md:30`(と写し)は例を挙げる文。「only when」の文(`post-implementation-pipeline.md:13`、`subagent-policy.md:19-21`、README の `:97-100`・`:232`、`AGENTS.md:37` と `.ralph/core/AGENTS.core.md:26` と両方の写し、`docs/architecture/repo-map.md:52`)は必要条件の言い方で、偽ではない。空の差分になるのは branch に commit がないときだけ |
| `templates/base/ralph.toml:19-29` | Yes | F1 で「in a later release」を消し、127 を足した。`[org.review]` 側の「later release」(`:255-256`)は #267・#269 まで正しい |
| `internal/cli/config.go:56-66`(`config get --help`) | Yes(情報) | 「Treat that, and only that, as off」は exit 1 の行の中の話で、ralph 自身が 127 を返すことはない。127 の行を足すかは任意で、足すなら `internal/cli/config_test.go:228` の固定も直す |
| `.claude/rules/ralph/model-routing.md:69-75` | Yes | plan skill も 3 つの変数を同じ fallback で読む、と書く |
| `docs/recipes/codex-setup.md:36` | Yes | Codex だけの機械で claude を呼ぶ計画助言と cross-review が走らない、は D5 の判別と合う |
| `internal/org/driver/paneenv.go:34-40`、`scripts/xreview-helpers.sh:57-69`、`scripts/ralph-config.sh` のコメント | Yes | `pick_reviewer` の引数なしの既定と、2 つの skill の呼び方の説明が今のコードと合う |
| 計画の本文 | 情報 | AC の box はすべて `[ ]`(Progress だけが進む、いつもの遅れ)。Verify plan の行が「AC1〜AC8」で AC9 を挙げない。どちらも digest の外の直しで済む |
| issue #266 の「やること」 | 情報 | issue は判別の順を「`RALPH_PRIMARY_CLI`、`which`、`pick_reviewer`」と書くが、計画の D5 は `which` を外した(Codex の計画助言 MEDIUM 2)。受け入れ条件には影響しない。PR の本文でこの外れに触れるとよい |
| `docs/tech-debt/README.md:376`(RC11、watcher の `claude -p` が利用者の hook を走らせる) | 情報 | trigger(`realClaudeInvoke` の変更)は引かれていない。「claude に hook を止める flag があるかは未確認」の部分に、この PR で確かめた `--restricted`(user・project・local の settings を読まない、claude 2.1.304)を足せる。任意 |

### /sync-docs への文案(V1〜V4)

V1 `README.md:135`: 「The skills and seats that read a setting with it treat only an older ralph without this command (exit 1 with `unknown command "config"`) as off, and stop and report exit 2 and 3.」を次に置き換える。

> The skills and seats that read a setting with it treat only two failures as off: an older ralph without this command (exit 1 with `unknown command "config"`), and no `ralph` on PATH (the shell's exit 127). They stop and report any other failure, exit 2 and 3 included.

V2 `docs/tech-debt/README.md:31`:

- `scripts/xreview-helpers.sh:66-73`(debt item と関連の 2 か所)を `scripts/xreview-helpers.sh:70-78` にする。
- 「the only surviving caller is `/cross-review`'s auto-detect (step 2 of `.claude/skills/cross-review/SKILL.md`), which only ever passes `claude`/`codex`/unset, so the fallback stays unreachable today」を次にする。

> the two callers, `/plan` step 11.b and `/cross-review` step 2 (feat/review-switches-skills), resolve the driver with the same one-call snippet, which starts from `claude` or `codex` and takes a `RALPH_PRIMARY_CLI` override only when it lowercases to one of the two, so they only ever pass `claude`/`codex` and the fallback stays unreachable today

- Trigger「Adding a second `pick_reviewer` call site that does not validate its driver argument first」を次にする。

> Adding a `pick_reviewer` call site that does not limit its driver argument to `claude`/`codex` first (the `/plan` step 11.b caller added in feat/review-switches-skills does)

V3 `docs/tech-debt/README.md:154`:

- 冒頭の「and `/plan`'s Codex advisory」を「and `/plan`'s plan advisory when codex is the reviewer」にする。
- (c) の 1 文を次に置き換える。

> (c) The reverse direction's claude calls: `/plan` step 11.c runs `claude -p` with `--restricted --strict-mcp-config --disallowedTools "mcp__*" --tools Read,Grep,Glob --permission-mode dontAsk` (feat/review-switches-skills, checked on claude 2.1.304), but `/cross-review`'s claude reviewer still runs `claude -p --permission-mode auto` with the default tools, the user's MCP servers, and the project's hooks. That branch added only `RALPH_WIP_CHECKPOINT=off` and the before/after git-state check, and the call still has no `command`, no `</dev/null`, no 20-minute watchdog, and no output file (self-review L5).

- Trigger の「the first review of the claude reviewer's permission mode」を「the first review of `/cross-review`'s claude reviewer (its permission mode and tools)」にし、関連の欄に `docs/reports/self-review-2026-10-29-review-switches-skills.md`(L5)を足す。

V4 空の差分(十分条件として書く 3 か所を直す。残りは任意):

- `.claude/skills/implement/SKILL.md:59`(`.claude/` を直して `./scripts/sync-skills.sh`、`templates/base/` の 2 つも同じ文): 「The review runs when `[standard].cross_review` is on and the reviewer CLI exists; otherwise」→「The review runs when `[standard].cross_review` is on, the reviewer CLI exists, and the diff against the base branch is not empty; otherwise」
- `.claude/rules/ralph/ralph-workflow.md:41-42`(と写し): 「and that CLI is available. Otherwise the step is skipped」→「and that CLI is available (for `/cross-review`, also when the diff is not empty). Otherwise the step is skipped」
- `docs/recipes/codex-setup.md:65-66`(と写し): 「is on (default off) and `claude` is available; otherwise」→「is on (default off), `claude` is available, and the diff is not empty; otherwise」
- 任意(例の列挙): `docs/quality/definition-of-done.md:30`(と写し)の「off in `ralph.toml` (the default) or the other CLI unavailable, it writes」→「off in `ralph.toml` (the default), the other CLI unavailable, or an empty diff, it writes」
- 任意(「only when」の文。そろえるなら同じ言い方): `AGENTS.md:37`、`.ralph/core/AGENTS.core.md:26`、`templates/base/AGENTS.md:33`、`templates/base/.ralph/core/AGENTS.core.md:26`、`docs/architecture/repo-map.md:52` の「is on and that CLI is available, otherwise records the skip」→「is on, that CLI is available, and the diff is not empty, otherwise records the skip」(`AGENTS.md` の管理ブロックは `AGENTS.core.md` と同じ文にする)。`post-implementation-pipeline.md:13`、`subagent-policy.md:19-21`、README の `:97-100`・`:232` も同じ形で足せる

## Observational checks

- P1 PATH の `ralph`(`/opt/homebrew/bin/ralph`、6.0.0 44c14b97): `ralph config get standard.plan_advisory` と `standard.cross_review` は、どちらも rc 1、stdout 空、stderr `unknown command "config" for "ralph"`。skill の 11.a と手順 0 では「Exit 1 with `unknown command "config"`」に当たり、`skipped (ralph without config get; treated as off)` として記録される。merge のあと、この repo で新しい binary を入れるまで計画助言と cross-review は飛ぶ(計画の R2 のとおり)。
- P2 build した binary(`go build -o <scratchpad>/v266/ralph ./cmd/ralph`、HEAD fd5daaaf、`GOCACHE` は worktree の gitignore された `.harness/cache/go-build`): worktree の root から `standard.plan_advisory` と `standard.cross_review` はどちらも rc 0 で `true`。知らないキーは rc 2、`ralph.toml` のない dir では rc 0 で `false`、値の型の誤りと TOML の構文の誤りは rc 3。PATH から `ralph` を外すと sh と zsh のどちらも rc 127。skill の 4 つの分岐の入力がすべて実際に出ることを確かめた。
- P3 `/cross-review` 手順 4 の空の差分の確かめ(skill の文のまま)を bash・zsh・sh で走らせ、3 つとも `base=main rc=1`。`$?` は `git diff` の rc を取る。
- P4 driver の snippet(plan skill の行を `grep` で取り出したもの)を zsh で 6 通り。結果は上の「一致の確かめ」。
- P6 `claude --version`(2.1.304)と `claude --help` を読んだ(読むだけの呼び出しで、依頼の一覧にはない。モデルは呼んでいない)。`--restricted` の説明は、コマンドを走らせる道具を外す、user・project・local の settings を読まない、managed settings と `--settings` は効く、MCP を止めるには `--strict-mcp-config` を足す、file の道具を作業 dir に閉じる、で、skill の注釈(`.claude/skills/plan/SKILL.md:98`)と合う。
- P7 implementer の S2 の probe の記録(scratchpad の `impl-266-s2/`)を読んだ。結果は AC1 の欄。claude を呼んでの再実行はしていない。
- 作業のあとの `git status --porcelain` は空(evidence の log は gitignore の対象)。

## Coverage gaps

- `go test ./internal/config/ ./internal/insights/`、`TestDefaultsLockStep`、`TestRootRalphToml_KnownKeysAndReviewSwitchesOn`、backfill の 2 つのテスト、`tests/test-codex-exec-invocation.sh`・`test-check-skill-sync.sh`・`test-slice-verification.sh`・`test-skill-insight-cycle.sh` などの実行は /test。AC5 の写しの変異も /test か Progress の記録に頼る。
- skill を通しで動かした確かめはしていない: Claude Code の `/plan` 11 が codex を呼ぶ道、Codex の driver の `/plan` 11 が claude を呼ぶ道(R4: Codex の sandbox の中から `claude -p` が通るか)、driver が Codex の `/cross-review` の claude の道。S2 の probe は haiku と小さな repo での 1 回。
- `--restricted` で settings を読まないため、settings 経由の認証(`apiKeyHelper`、settings の `env` で Bedrock や Vertex を選ぶ構成)の環境で claude が認証できるかは未確認(self-review と同じ)。できなければ rc が 0 にならず `incomplete` に落ちる作り。
- `/pr` の対応表は、triage の report に cycle が複数あるとき、どの `Reviewer status:` を使うかを言わない(self-review の run 2 の Notes と同じ)。cycle の途中で switch を変えたときに限る。
- PR の本文の release のノート(既定の off、新しい binary が要ること、Codex の driver で claude を呼ぶこと)と Known gaps(N2、L5)は /pr で見る。

## Verdict

pass

- Verified: AC1〜AC3、AC9 は skill の文を読み、snippet と確かめのコマンドを 3 つの shell で走らせ、`ralph config get` の終了コードを古い binary と build した binary で測って確かめた。AC1 の小さな実行は implementer の記録を読んで確かめた。AC4〜AC6 はコードとテストの存在と中身を読んで確かめた。静的検査と 3 つの同期の検査はすべて rc 0。
- Partially verified: AC7(「optional」の grep は 0 件。V1〜V4 の LOW の文書のずれが残る。どれも /sync-docs で直せる文で、V1 は新規)。AC8 と issue の受け入れ条件 4(同期の検査は通った。`go test` とシェルのテストは /test)。
- Not verified: テストの実行、skill の通しの実行(両方の driver)、R4、settings 経由の認証と `--restricted` の組み合わせ。
