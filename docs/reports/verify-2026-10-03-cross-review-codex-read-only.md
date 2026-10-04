# Verify report: cross-review-codex-read-only

- Date: 2026-10-03
- Plan: docs/plans/active/2026-10-03-cross-review-codex-read-only.md
- Verifier: verifier subagent (Claude)、cycle 1
- Branch: security/cross-review-codex-read-only、HEAD 06d54132(push 済み)、base main c9da0a27
- Scope: 仕様適合(AC-1〜AC-6 を該当行と実行記録で確認)、静的解析、文書のずれ、self-review の指摘(M-1、M-2、L-1、L-2、B-1〜B-4)が HEAD で直っているかの確認。振る舞いのテスト suite は /test の担当で、verdict には使っていない。AC-3 と AC-3b は実装者の実行記録(log)を読んで確認し、codex の認証つきの再実行はしていない
- Evidence: `docs/evidence/verify-2026-10-02-233013.log`(full)、`docs/evidence/verify-2026-10-02-233025.log`(default)。どちらも `docs/evidence/*.log` が gitignore のため commit には含まれない。scratch の出力は repo の外に置いた

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS(rc 0) | `Requested scope: full`、`Language scope: full`、golang を選び gofmt ok / 0 issues。`    OK` が 30 行で FAIL なし(`FAIL` の語に当たるのは `post_tool_failure_feedback.sh` と `PostToolUseFailure` の名前だけ)。branch secret scan は `c9da0a27..06d54132 against origin/main: clean` |
| `./scripts/run-static-verify.sh`(default スコープ) | PASS(rc 0) | `Requested scope: changed` が `Language scope: full fallback (unclassified:.codex/config.toml)` になり、golang が選ばれて gofmt ok / 0 issues。`.codex/config.toml` が言語に分類されないので full に倒れた。#195 の merge-base 方式の挙動で、no_changes にはなっていない |
| `./scripts/check-skill-sync.sh` | PASS(rc 0) | `13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | PASS(rc 0) | IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5(`model-routing.md`、`verify.yml`、`CLAUDE.md`、`quality-gates.md`、`adding-a-language-pack.md`。今回触れたファイルは含まない) |
| `./scripts/check-template-purity.sh` | PASS(rc 0) | template の新しい文に meta-repo 固有の参照なし |
| `cmp` で root と template の対 | PASS | `.codex/config.toml`、`docs/recipes/codex-setup.md`、`{.claude,.agents}/skills/{cross-review,plan}/SKILL.md` の 8 対すべて byte 一致(skill は `templates/base/` 側との対) |
| `shellcheck -S warning tests/test-codex-exec-invocation.sh`、`sh -n` | PASS(どちらも rc 0) | 指摘なし |
| `tomllib` の dict 比較(HEAD と `git show main:`) | PASS | root も template も main と HEAD の parse 結果が等しい。root と template も等しい。キーは `approval_policy`、`features`、`model`、`profiles`、`sandbox_mode`、`tui`、`web_search`。変わったのはコメントだけ |
| `git diff main...HEAD --check`、U+FFFD の走査 | PASS | 空白の指摘なし、変更ファイルに U+FFFD なし |

## Spec compliance

呼び出しの行の確認は、HEAD の行から追加した語を取り除いたものが main の行と一致するか、で行った。

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1 8 か所の cross-review の呼び出しが `-c sandbox_mode=read-only` と `--ignore-rules` を含み、4 か所の `/plan` の呼び出しが `--sandbox read-only` と `--ignore-rules` を含む。check-skill-sync と check-sync が green | Pass | cross-review は 4 面とも 2 行(Step 4 の block と CLI execution modes の表。`.claude/` は `:58` と `:167`、`.agents/` は `:57` と `:166`)で、`-c sandbox_mode=read-only` と `--ignore-rules` がどちらも 2 行に入っている。`/plan` は 4 面とも `:71` の 1 行で `--sandbox read-only` と `--ignore-rules` が入っている。HEAD の 12 行から ` -c sandbox_mode=read-only exec review --ignore-rules ` を ` exec review ` に、` exec --sandbox read-only --ignore-rules ` を ` exec --sandbox read-only ` に戻すと、main の 12 行と byte 一致する。足したのはこの語だけ。`4fd7bf5f..HEAD` で `command codex` を含む行の増減は 0 行で、Slice B と C は呼び出しの行を変えていない。check-skill-sync と check-sync は rc 0 |
| AC-2 テストが、どの codex の呼び出しの行にも read-only の sandbox の指定があることを検査する。red: 4 面のどこか 1 か所から read-only の指定か `--ignore-rules` を外すと落ちる。`/plan` も同じ | Pass | 静的に読んだ: `check_sandbox_tokens`(`tests/test-codex-exec-invocation.sh:200`)が awk(`CODEX_ARGS_AWK`、`:138`)で行を sh と同じ規則で語に分け、`command codex` ごとに (i) sandbox の指定が 1 つ以上あり全部 `read-only`、(ii) `is_widening_word`(`:114`)に当たる語がない、(iii) `--ignore-rules` がある、を `report_sandbox_tokens`(`:247`)で 3 件の PASS / FAIL にする。scratch に HEAD のテストと skill 8 ファイルを写して実行した(下の「観測」)。変更なしは 132 / 132(sh と dash)。変異 13 種はどれも 1 件(n1 だけ、2 つ目の呼び出し分の 3 件)で落ち、落ちた assertion の名前が変異と対応した。正しい書き方の対照 1 件(n2)は通った。main のテストと skill でも 96 / 96、main のテストを HEAD の skill に当てても 96 / 96 |
| AC-3 実際の codex で新しい形の cross-review の呼び出しを、差分のある branch に 1 回実行し、`codex rc=0`、`-o` が空でない、header が `sandbox: read-only`。main のチェックアウトで codex を動かさない。実行後に main が clean | Pass(実装者の log を確認。再実行なし) | `impl/ac3-driver.out`: `BASE=main`、`codex rc=0`、`main status=[]` が実行の前後とも。`impl/cross-review-cross-review-codex-read-only-c1.log` の header は `workdir: …/.claude/worktrees/cross-review-codex-read-only`、`approval: never`、`sandbox: read-only`。`-o` のファイルは 217 バイトで空でない。実行した行は skill の Step 4 の行と同じ形(`-c sandbox_mode=read-only exec review --ignore-rules --base "$BASE" -o … </dev/null`)で、`4fd7bf5f..HEAD` で呼び出しの行が変わっていないので HEAD の行にも当てはまる。worktree の `git status --porcelain` は実行の前後で同じ。log には、project の `.codex/config.toml` を読んだ印の `Ignored unsupported project-local config keys … profiles` の warning があり、project の設定が効く trust 済みの条件で実行したと分かる。log には `WARNING: proceeding, even though we could not create PATH aliases: Operation not permitted`(`:666` など 7 行)があり、`Operation not permitted` に当たる行はこれだけで、plan の「sandbox が拒否したのは PATH の警告だけ」と合う |
| AC-3b 許可ルールを回り込めないことの実測: (a) `-c sandbox_mode=read-only` だけだと `touch` が通る、(b) `--ignore-rules` を足すと拒否されファイルができない。終了後に一時ルールと scratch を消し、worktree と main が clean | Pass(実装者の log を確認。再実行なし) | ルール `impl/ac3b-probe.rules` は `touch` の 2 つの scratch パスだけを `decision = "allow"` にする。`impl/ac3b-a-driver.out`: header は `sandbox: read-only`、`approval: never`、`codex rc=0`、`RESULT: target EXISTS`、`-o` は `SUCCESS`。`impl/ac3b-b-driver.out`: 同じ header、`codex rc=0`、`RESULT: target ABSENT`、`-o` は `FAILED: touch: …/ac3b-probe-b: Operation not permitted`。2 回の違いは `--ignore-rules` の有無だけ(`impl/ac3b-run.sh` の 2 つの分岐)。pre / post の driver 出力に `?? .codex/rules/ac3b-probe.rules` があり、ルールが worktree に置かれていたことが分かる。今は `.codex/rules` が存在せず、`ac3b-probe-a` / `-b` も存在せず、worktree と main の `git status --porcelain` は空。`~/.codex` は私は読んでいない |
| AC-4 recipe(+ template)と `.codex/config.toml`(+ template)のコメントが新しい呼び出しの形と合う。root と template が一致 | Pass | 8 対の `cmp` が一致。recipe(`docs/recipes/codex-setup.md:93-106`)とコメント(`.codex/config.toml:19-24`、`:26-33`、`:68-70`)の個々の文を、skill の行・log・help と突き合わせた(下の「Documentation drift」)。食い違う文は 1 件、既存の主張の持ち越しで O-3 |
| AC-5 `docs/tech-debt/README.md` のこの件の行が解決済み | Pass | `:147` の `<!-- RESOLVED 2026-10-03 … Row preserved for traceability. -->` と `:148` の取り消し線の行(`(RESOLVED 2026-10-03 in security/cross-review-codex-read-only)`)の組で、`:16-17` など既存の慣習と同じ。5 列の表の列数も他の行と同じ。comment が書く「credentialed run で `sandbox: read-only`、`codex rc=0`、`-o` が空でない」「`.rules` の allow が read-only だけでは通り、`--ignore-rules` で止まる」は AC-3 / AC-3b の log と一致した |
| AC-6 `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green | 一部確認 | 静的な半分は full スコープの `run-static-verify.sh` が rc 0。`run-verify.sh` の test の半分は /test の担当。実装者の `impl/c-run-verify.out` は `All verifiers passed`(mode all、branch scan は `c9da0a27..1280d797`)だが、f4b03a68 より前の commit が対象で、私は再実行していない |

## Self-review cross-check

HEAD 06d54132 で、self-review report(`docs/reports/self-review-2026-10-03-cross-review-codex-read-only.md`)と addendum の指摘が直っているかを、コードと独立した変異で確認した。

| # | 指摘 | HEAD での状態 | 根拠 |
| --- | --- | --- | --- |
| M-1 | 禁止の検査が 3 つの文字列しか見ない(`--yolo`、`--approve-for-me` を見逃す) | 直っている | `is_widening_word`(`:114-122`)が `--yolo`、`--approve-for-me`、`--full-auto`、`--add-dir`、`-a`、`-p`、`--dangerously-bypass-*`、`approval_policy=`、`default_permissions=` などを列挙。変異 `/plan` に `--add-dir`(m9)、cross-review に `--approve-for-me`(n3)、`-c approval_policy=on-request`(n5)、`--sandbox=danger-full-access`(n4)、`sandbox_mode=workspace-write`(m5)、`--sandbox danger-full-access`(m8)は、どれも 1 件だけ落ちる |
| M-2 | 文が「reviewer に書き込ませない」と言い過ぎ。残るリスクが register に無い | 直っている | `.claude/skills/cross-review/SKILL.md:60`(4 面)、`.claude/skills/plan/SKILL.md:73`(4 面)、`docs/recipes/codex-setup.md:93-106` が「実行するコマンドの書き込みとネットワークを止める」に絞り、MCP server と repo の外の読み取りは対象外と書く。残るリスクは `docs/tech-debt/README.md:149`(a MCP、b 読み取りと triage report、c 逆向きの claude reviewer、d `ralph doctor --probe-models`) |
| L-1 | config のコメントが「approval は on-request」と書くが `codex exec` は never | 直っている | `.codex/config.toml:19-24`、`:26-33` が対話の session と `codex exec` を分けて書く |
| L-2 | 解決のコメントに plan のパスがない | 直っている | `docs/tech-debt/README.md:147` と `:148` の Related の列に `docs/plans/archive/2026-10-03-cross-review-codex-read-only.md` と self-review report のパスがある(plan の archive 先は O-2) |
| B-1 | `check_sandbox_tokens` が (1) `=` の前後に空白のある `-c`、(2) `</dev/null` の後ろの語、(3) 同じ行の 2 つ目の `command codex` を読まない | 直っている | 変異 m7(`-c 'sandbox_mode = "danger-full-access"'` を足す)は 1 件落ちる。m6(`</dev/null` の後ろに `--yolo`)は 1 件落ちる。n1(同じ行に `; command codex --yolo exec review …` を足す)は `[codex #2]` として 3 件落ちる。対照 n2(`-c 'sandbox_mode = "read-only"'`)は 132 / 132 のまま通る |
| B-2 | 「ネットワークも止める」は既定の挙動 | 直っている | 3 か所すべてが「by default, network access」(`cross-review/SKILL.md:60`、`plan/SKILL.md:73`、`codex-setup.md:94-95`) |
| B-3 | config のコメントが「ralph の `codex exec` はすべて read-only」と読める(`--probe-models` は違う) | 直っている | `.codex/config.toml:22-23` は `/plan` の advisory と `/cross-review` の reviewer の 2 つに絞った。`--probe-models` は debt の行 (d) に記録。`internal/org/driver/probe.go:26` の `r.Run(ctx, "codex", "exec", "--model", model, "--skip-git-repo-check", "ping")` で、sandbox の指定がないことを確認した |
| B-4 | 選択肢に codex が受け付けない `"untrusted"` が載っている | 直っている | `.codex/config.toml:29-31` の選択肢は `never`、`on-request`、`granular` で、`:32` が `untrusted` は拒否、`on-failure` は deprecated と書く。実装者が取得した codex の config reference(`impl/config-ref.html`)に「untrusted is unsupported, and on-failure is deprecated」と `granular` の記述があり、self-review の S3 / S8 / S9 は 0.154.0 と 0.159.2 の両方で `untrusted` がエラーになったと記録している |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | PASS | 上の表のとおり |
| `./scripts/run-static-verify.sh`(default) | PASS | `full fallback (unclassified:.codex/config.toml)`。default が何を選んだかの記録 |
| `shellcheck -S warning tests/test-codex-exec-invocation.sh` | PASS | `run-static-verify.sh` の shellcheck の対象にこのテストは入っていないので、別に実行した |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.claude/skills/cross-review/SKILL.md:60`(+3 面) | Yes | 「`.codex/config.toml` が `danger-full-access`」「`codex exec` は承認を求めない」「`exec review` に `--sandbox` がない」「`--ignore-rules` が allow のルールを止める」を、config、R1(`approval: never`)、`codex exec review --help`(0.154.0 と 0.159.2 で `--ignore-rules` あり、`--sandbox` なし。isolated HOME で私も取得)、AC-3b と突き合わせて一致した。MCP の文は R12 と、AC-3 の log にある `failed to refresh OAuth tokens for server atlassian`(reviewer の session が user 設定の MCP server を起動した印)と合う |
| `.claude/skills/plan/SKILL.md:73`(+3 面) | Yes | 「see `/cross-review` Step 4」の参照先(`cross-review/SKILL.md:52` の `4. Invoke reviewer`)に該当の文がある |
| `docs/recipes/codex-setup.md:93-106`(+ template) | Yes | 内容は skill と同じ。代名詞の「Without it」が `-c sandbox_mode=read-only` を指す点は読みにくいが、意味は取れる(指摘ではない) |
| `.codex/config.toml:19-33`、`:68-70`(+ template) | 一部 | 事実の文はすべて合う。`:20-21` の「In an interactive session, approval_policy below gates destructive shell calls」は確認できていない(O-3) |
| `docs/tech-debt/README.md:147-149` | Yes | 解決の行は上の AC-5。残るリスクの行は (a)〜(d) を R12(`enabled=false` と `--ignore-user-config` が効き、`mcp_servers={}` が効かない)、R13、`probe.go:26`、`ralph doctor --probe-models`(`internal/cli/doctor.go:36` のフラグ)と突き合わせて一致。(a) の「model が承認なしで tool を呼べるかは未確認」は言い過ぎていない |
| `.codex/README.md:63`、`README.md:285`、`.claude/rules/ralph/post-implementation-pipeline.md:24`、`.claude/rules/ralph/model-routing.md`、`scripts/ralph-config.sh:36-42` | Yes | 前の3つは「`/cross-review` が `codex exec review` を呼ぶ」とだけ書き、呼び出しの形を書いていない。`model-routing.md` の sync note は環境変数の fallback の話で、sandbox に触れない。`ralph-config.sh` のコメントは `-m` と `-c model_reasoning_effort=` の説明で、今も正しい |
| plan の Status / AC のチェックボックス / Progress checklist | 遅れている(O-1) | `Status: Draft`、AC-1〜AC-6 が `[ ]`、「Verification artifact created」が `[ ]` のまま |

### 観測(fail にしない)

- O-1(INFO): plan の `Status: Draft`、AC の `[ ]`、Progress の「Verification artifact created」は実装の進みに遅れている。plan の更新は orchestrator の担当で、verify は編集しない。
- O-2(INFO): `docs/tech-debt/README.md:147-149` が `docs/plans/archive/2026-10-03-cross-review-codex-read-only.md` を引くが、plan は今 `docs/plans/active/` にある。`/pr` が archive すると解消する前方参照で、`(archived by /pr)` のような印はない。
- O-3(LOW): `.codex/config.toml:20-21`(+ template)の「In an interactive session, approval_policy below gates destructive shell calls」は、元の「Pair with approval_policy below to gate destructive shell calls」の主張を持ち越した文で、この PR が断定の形に書き直した。`on-request` は model が求めたときに承認を挟む仕組みなので、`danger-full-access` と組んで破壊的なコマンドを止めるかは、認証なしの probe では確かめられない(self-review addendum の Known gaps と同じ)。この PR の目的(`codex exec` の呼び出しを read-only にする)は、この文が正しくなくても満たされる。

## Observational checks

- AC-2 の独立した変異(scratch、`PROJECT_ROOT` の形に写した HEAD の `tests/test-codex-exec-invocation.sh`、`scripts/ralph-config.sh`、skill 8 ファイル。repo 内のファイルは触っていない)。結果は assertion の名前で突き合わせた。
  - m1 `.claude` cross-review `:58` から `-c sandbox_mode=read-only` を削除: 1 件(selects no sandbox)
  - m2 `.agents` cross-review `:166` から `--ignore-rules` を削除: 1 件(missing --ignore-rules)
  - m3 template `.agents` の `/plan` `:71` から `--sandbox read-only` を削除: 1 件
  - m4 template `.claude` の `/plan` `:71` から `--ignore-rules` を削除: 1 件
  - m5 template `.claude` cross-review `:167` を `sandbox_mode=workspace-write` に: 1 件(selects a sandbox other than read-only)
  - m6 `.claude` cross-review `:58` の `</dev/null` の後ろに `--yolo`: 1 件(widens)
  - m7 template `.agents` cross-review `:57` に `-c 'sandbox_mode = "danger-full-access"'` を追加: 1 件
  - m8 `.agents` の `/plan` `:71` を `--sandbox danger-full-access` に: 1 件
  - m9 `.claude` の `/plan` `:71` に `--add-dir /tmp`: 1 件
  - n1 `.claude` cross-review `:58` に `; command codex --yolo exec review --base main </dev/null` を追加: 3 件(`[codex #2]` の 3 つの規則)
  - n2(対照)`.agents` cross-review `:57` を `-c 'sandbox_mode = "read-only"'` に: 132 / 132
  - n3 template `.claude` cross-review `:58` に `--approve-for-me`: 1 件
  - n4 `.claude` cross-review `:167` を `--sandbox=danger-full-access` に: 1 件
  - n5 `.claude` cross-review `:58` に `-c approval_policy=on-request`: 1 件
- 変更なしの HEAD は `sh` で 132 / 132、`dash` で 132 / 132。main のテストと skill の組は 96 / 96(plan の「96 から 132」と一致)、main のテストを HEAD の skill に当てても 96 / 96(#184 の形を壊していない)。これらは AC-2 の red 条件の証拠として scratch で実行したもので、/test の verdict の代わりではない。
- codex の help(isolated の `HOME` / `CODEX_HOME`、認証なし、10〜20 秒の watchdog): 0.154.0 と 0.159.2 のどちらも `exec review --help` に `--ignore-rules`(「Do not load user or project execpolicy `.rules` files」)があり `--sandbox` はない。`exec --help` には両方ある。
- 手順の逸脱(開示): 最初の help 取得で、glob が mise の `node/23.10.0/bin/codex`(0.120.0)に当たり、偽 `HOME` の下で `--version` が戻らなかった。私の `node …/codex --version` と子プロセス(PID 32449、32450)を kill した。`HOME` と `CODEX_HOME` は scratch を向けていたので `~/.codex` には触れていない。以降は `node/24.15.0`(0.154.0)と `npm-openai-codex/0.159.2` だけを使った。

## Coverage gaps

- AC-3 と AC-3b は Slice A の時点の worktree(未 commit の変更つき)で実行された log に頼っている。実行した呼び出しの行は HEAD と同じだが、HEAD での再実行はしていない。再実行する場合は worktree だけ、`</dev/null`、watchdog つきで、main のチェックアウトでは動かさない。
- read-only の sandbox が実際に書き込みを拒否することは AC-3b の `Operation not permitted` で確かめられている。ネットワークの遮断は、組み込みの `:read-only` の profile での観測(self-review R13)に頼っていて、`-c sandbox_mode=read-only` が同じ規則かは未確認。文は「by default」と限定している。
- `--ignore-rules` を受け付ける最小の codex の版は調べていない(0.154.0 と 0.159.2 は受け付ける)。古い版では `unexpected argument` で非 0 終了になり、skill の reviewer incomplete の経路に入る。閉じる側に倒れるので害は小さいが、毎回 incomplete になる。
- awk の差: テストは BSD awk(macOS)で 132 / 132 を確認した。CI と同じ ubuntu(mawk)の確認は orchestrator の記録(plan の Deviation notes)に頼っていて、私は再実行していない。
- 残るリスク (a) MCP、(b) 読み取りと triage report、(c) 逆向きの claude reviewer、(d) `ralph doctor --probe-models` は、plan が範囲外とし、`docs/tech-debt/README.md:149` に記録されている。この verify では解消の確認ではなく、記録の正確さの確認にとどめた。
- `.codex/config.toml:20-21` の「対話の session で approval_policy が破壊的なコマンドを止める」は確認できていない(O-3)。
- `run-verify.sh` の test の半分、`go test`、`tests/test-codex-exec-invocation.sh` の正式な実行は /test の担当。

## Verdict

- Verified: AC-1(12 行すべてが期待どおりで、足した語だけが main との差)、AC-2(静的に読み、独立した変異 13 種と対照 1 件で red を確認)、AC-3 と AC-3b(実装者の log から header・rc・`-o`・回り込みの再現と遮断・後始末を確認)、AC-4(8 対 byte 一致、個々の文の事実確認)、AC-5(行の形と comment の内容)、self-review の M-1、M-2、L-1、L-2、B-1〜B-4 が HEAD で直っていること、静的解析一式(full スコープ rc 0、default スコープ rc 0 は full fallback)
- Partially verified: AC-6(静的な半分のみ。test の半分は /test)、ネットワーク遮断(R13 の観測に依存)
- Not verified: HEAD での codex の再実行、mawk 上のテスト、`--ignore-rules` を受け付ける最小の codex の版、`.codex/config.toml:20-21` の対話 session での承認の挙動

Verdict: **PASS**(CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 1(O-3)、INFO 2(O-1、O-2))。/test に進んでよい。
