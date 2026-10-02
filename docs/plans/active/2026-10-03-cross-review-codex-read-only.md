# cross-review-codex-read-only

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-03
- Related request: issue #197。`/cross-review` の codex reviewer(`command codex -m … -c model_reasoning_effort=… exec review --base "$BASE" -o <file> </dev/null`)は sandbox を指定しないので、trust 済みの project では `.codex/config.toml` のトップレベルの `sandbox_mode = "danger-full-access"` と、`exec` の `approval: never` で動く。レビューする差分に紛れた指示に codex が従えば、承認なしでコマンドを実行できる。PR #198 の self-review(M-2)で見つかった
- Related issue: 197
- Type: security
- Branch: security/cross-review-codex-read-only

## Objective

`/cross-review` の codex reviewer を read-only の sandbox で動かし、利用者や project の execpolicy の許可ルール(`.rules`)も読ませない。`/plan` の advisory にも同じ指定を入れる。レビューする差分の中身にかかわらず、reviewer の codex がファイルを書いたり、書き込みを伴うコマンドを実行したりできないようにする。

## 調査で確認したこと(2026-10-03、main c9da0a27)

- 呼び出しは 4 面(`.claude/skills/cross-review/SKILL.md`、`.agents/skills/cross-review/SKILL.md`、template の 2 つ)に、それぞれ 2 か所ある: step 4 の fenced block(58 行目)と「CLI execution modes」の表(167 行目)。
- `codex exec review --help`(codex-cli 0.154.0)に `--sandbox` はなく、`-c, --config <key=value>` はある。PR #198 の reviewer の scratch の probe で、`-c sandbox_mode=read-only` を付けると header が `sandbox: read-only` になることを確認済み(認証なし。`-o` が書かれるかは未確認)。
- `/plan` の Codex plan advisory は `exec --sandbox read-only` を渡している(`.claude/skills/plan/SKILL.md` step 11.c)。
- `tests/test-codex-exec-invocation.sh` が、4 面の `/plan` と `/cross-review` の codex の呼び出しの行に、`command codex `、`</dev/null`、`-m "${RALPH_CODEX_REVIEWER_MODEL:-`、`model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-`、`-o` の 5 つがあることを固定している(sandbox は見ていない)。
- 呼び出しの形を説明している文書: `docs/recipes/codex-setup.md` 85〜100 行(+ template)、`.codex/config.toml` の profile の節のコメント(「`/cross-review` は `-m` と `-c model_reasoning_effort` を明示する」、+ template)。
- `docs/tech-debt/README.md:147` にこの件の行がある(PR #198 で追加)。
- codex の execpolicy の `.rules` で `decision = "allow"` のルールに一致したコマンドは、承認なしで sandbox の外で実行される(Codex plan advisory の HIGH。codex の Rules の仕様)。`codex exec` と `codex exec review` には `--ignore-rules`(「user or project の execpolicy `.rules` を読まない」)がある(codex-cli 0.154.0 の help で確認)。この repo は `.rules` を配っていない。利用者の `~/.codex` にあるかは、読まないので分からない。
- 逆方向(Codex が driver で reviewer が claude)は `claude -p --model … --permission-mode auto --output-format json` で呼ぶ。この issue の範囲外(Open questions に記録)。

## Scope

- 4 面の cross-review の codex の呼び出し(各 2 か所)に `-c sandbox_mode=read-only` と `--ignore-rules` を足す。`-c` はほかの `-c` と同じく `exec` の前、`--ignore-rules` は `exec review` のオプションとして置く(help で受け付ける位置を確かめる)。
- 4 面の `/plan` の Codex plan advisory の呼び出し(各 1 か所、すでに `--sandbox read-only`)に `--ignore-rules` を足す。
- `tests/test-codex-exec-invocation.sh`: どの codex の呼び出しの行も read-only の sandbox(`/plan` は `--sandbox read-only`、`/cross-review` は `sandbox_mode=read-only`)と `--ignore-rules` を指定していることを検査に足す。
- `docs/recipes/codex-setup.md`(+ template)の呼び出しの説明に、両方の呼び出しが read-only の sandbox で動くことと、その理由(project の設定の `danger-full-access` を引き継がない)を足す。
- `.codex/config.toml`(+ template)の profile の節の、`/cross-review` の呼び出しを説明するコメントを新しい形に合わせる(値は変えない)。
- `docs/tech-debt/README.md` のこの件の行を、ファイルの慣習(取り消し線と `RESOLVED` のコメント)で解決済みにする。

## Non-goals

- `.codex/config.toml` のトップレベルの `sandbox_mode = "danger-full-access"` の変更(project の codex の使い方全体に関わる別の判断)。
- 逆方向の claude reviewer(`--permission-mode auto`)の権限の見直し。
- org runtime の codex 座席の sandbox(`internal/org/permissions.go` の別の仕組み)。

## Assumptions

- `-o` のファイルは sandbox の外で codex CLI 自身が書くので、read-only でも書かれる(AC-3 で確かめる)。

## Affected areas

- `.claude/skills/cross-review/SKILL.md`、`.agents/skills/cross-review/SKILL.md`、`templates/base/.claude/skills/cross-review/SKILL.md`、`templates/base/.agents/skills/cross-review/SKILL.md`
- `.claude/skills/plan/SKILL.md`、`.agents/skills/plan/SKILL.md`、`templates/base/.claude/skills/plan/SKILL.md`、`templates/base/.agents/skills/plan/SKILL.md`
- `tests/test-codex-exec-invocation.sh`
- `docs/recipes/codex-setup.md`、`templates/base/docs/recipes/codex-setup.md`
- `.codex/config.toml`、`templates/base/.codex/config.toml`(コメントだけ)
- `docs/tech-debt/README.md`

## Design decisions

- sandbox は `read-only` にする(`workspace-write` にはしない)。reviewer の役割に書き込みは要らず、`workspace-write` では reviewer がレビュー中の worktree を書き換えられてしまう。`/plan` の advisory と同じ扱いになる。代わりに、reviewer は書き込みを伴うコマンド(`go test` のビルドキャッシュなど)を動かせなくなる。テストは pipeline の tester が別に走らせるので、受け入れる。
- 指定は `-c sandbox_mode=read-only`(`exec review` に `--sandbox` がないため)と `--ignore-rules`。read-only の sandbox だけでは、許可ルールに一致したコマンドが sandbox の外で動く経路が残る。reviewer にも advisory にも、利用者のルールで許したコマンドを動かす必要はない。
- Critical forks: None(issue と `/plan` の advisory で方針が決まっている)

## Acceptance criteria

- [ ] AC-1: 4 面の cross-review の codex の呼び出し(各 2 か所)がすべて `-c sandbox_mode=read-only` と `--ignore-rules` を含み、4 面の `/plan` の advisory の呼び出しが `--sandbox read-only` と `--ignore-rules` を含む。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が green。
- [ ] AC-2: `tests/test-codex-exec-invocation.sh` が、どの codex の呼び出しの行にも read-only の sandbox の指定があることを検査する。red: 4 面のどれか 1 か所から read-only の指定か `--ignore-rules` を外すと落ちる。`/plan` の `--sandbox read-only` か `--ignore-rules` を外しても落ちる。
- [ ] AC-3: 実際の codex(いつもの認証)で、新しい形の cross-review の呼び出しを、差分のある branch(この branch)に対して 1 回実行し、`codex rc=0`、`-o` のファイルが空でない、log の header が `sandbox: read-only` になる。main のチェックアウトで codex を動かさない。実行後に main のチェックアウトが clean であることを確かめる。
- [ ] AC-3b: 許可ルールを回り込めないことを実際に確かめる(Codex plan advisory の HIGH)。この worktree(trust 済み)に、追跡しない一時的な project の `.rules` を置き、scratch のファイルへの無害な `touch` を `decision = "allow"` で許す(置き場所と書式は codex の Rules の仕様で確かめる)。同じ worktree で、codex に「その `touch` を実行して」と頼む `codex exec` を 2 回実行する: (a) `-c sandbox_mode=read-only` だけ → ルールが効いて書き込みが通る(脆弱性の再現)、(b) `-c sandbox_mode=read-only --ignore-rules` → 書き込みが拒否され、ファイルができない。(a) で書き込みが通らない場合は、ルールの置き方が効いていないので、その状況を report に書く。終わったら一時的な `.rules` と scratch のファイルを消し、worktree と main のチェックアウトが clean であることを確かめる。`~/.codex` は読まず、書き換えない。
- [ ] AC-4: `docs/recipes/codex-setup.md`(+ template)と `.codex/config.toml`(+ template)のコメントが新しい呼び出しの形と合う。root と template が一致。
- [ ] AC-5: `docs/tech-debt/README.md` のこの件の行が解決済みになっている。
- [ ] AC-6: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。

## Implementation outline

1. Slice A(implementer、opus。security の変更なので): Scope を 1 コミットで。
2. pipeline: self-review → verify → test → sync-docs → cross-review(この PR の cross-review 自体を新しい形で実行する) → PR(`Closes #197`)。

## Verify plan

- Static analysis checks: `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-sync.sh`、`shellcheck -S warning tests/test-codex-exec-invocation.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-6。
- Documentation drift to check: `docs/recipes/codex-setup.md`、`.codex/README.md`、`.codex/config.toml` のコメント、`.claude/rules/ralph/model-routing.md`、`README.md`、`.claude/rules/ralph/post-implementation-pipeline.md`。
- Evidence to capture: AC-3 の log の header、`-o` の内容。

## Test plan

- Unit tests: `sh tests/test-codex-exec-invocation.sh`(sh と dash)。
- Integration tests: AC-3 の実際の実行。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`。
- Regression tests: AC-2 の mutation。
- Edge cases: read-only の reviewer が書き込みを伴うコマンドを試みたとき、review が失敗せずに完了すること(AC-3 の log で、拒否されたコマンドがあっても rc 0 で `-o` が書かれること)。
- Evidence to capture: test report。

## Risks and mitigations

- read-only では reviewer がテストを動かせず、review が浅くなりうる: テストは pipeline の tester が走らせる。review の結果に「テストを動かせなかった」と出ても、完了の判定(`codex rc=0` かつ `-o` が空でない)は変わらない。AC-3 で実際の出力を確かめる。
- codex の将来の版で `-c sandbox_mode` の扱いが変わる: header に `sandbox:` が出るので、AC-3 と同じ確かめ方で検知できる。

## Rollout or rollback notes

skill の呼び出しと文書とテストの変更だけ。問題があれば revert する。下流には `ralph upgrade` で届く(skill は core)。

## Open questions

- 逆方向の claude reviewer(`claude -p --permission-mode auto`)の権限が reviewer として広すぎないか。この PR の範囲外。必要なら別の issue にする。

## Deviation notes

- 2026-10-03 plan: Codex plan advisory(gpt-6-astra、xhigh、watchdog の 1 行、`codex rc=0`、`-o` 1511 バイト)は HIGH 1: `sandbox_mode` だけを上書きしても、利用者や trust 済みの project の execpolicy の `.rules` の `decision = "allow"` に一致したコマンドは、承認なしで sandbox の外で動く。AC-3 の header と出力の確認ではこの経路を検出できない。orchestrator が `codex exec --help` と `codex exec review --help` に `--ignore-rules` があることを確認した。ユーザー決定: `/plan` の advisory も含めて更新。両方の呼び出しに `--ignore-rules` を足し、テストで固定し、一時的な project の `.rules` で回り込みの再現と遮断を確かめる AC-3b を足した
- 2026-10-03 work: Slice A は implementer(opus、security の変更のため)に委譲(4fd7bf5f、14 ファイル、+73 / -26、push 済み)。cross-review の呼び出しは `… -c "model_reasoning_effort=…" -c sandbox_mode=read-only exec review --ignore-rules --base "$BASE" -o … </dev/null`、`/plan` の advisory は `… exec --sandbox read-only --ignore-rules -o … "<prompt>" </dev/null`(`--ignore-rules` は codex-cli 0.154.0 の `exec` と `exec review` の両方のオプション)。それぞれに理由の 1 文を足した。テストは、どの呼び出しの行にも read-only の sandbox と `--ignore-rules` があることを確かめ、さらに `danger-full-access`、`workspace-write`、`--dangerously-bypass` のどれかが行にあれば落ちる(後ろから広い sandbox で上書きする改変を止めるため。handoff からの追加)。96 / 96 から 132 / 132(sh と dash)。red: 5 種の変異でそれぞれ 1 件だけ落ちる。commit の type は guard が `security:` を受け付けないので `fix:` にした(orchestrator が了承)。AC-3: この worktree で新しい形の cross-review を `--base main` に流し、`codex rc=0`、`-o` 217 バイト、header は `sandbox: read-only` と `approval: never`。sandbox が拒否したのは入れ子の codex の PATH の警告だけ。AC-3b: 公式の Rules の仕様(https://developers.openai.com/codex/rules、trust 済みの project の `.codex/rules/*.rules`)に従って、追跡しない一時的なルールで scratch のファイル 2 つへの `touch` だけを `decision = "allow"` にした(`codex execpolicy check` で allow を確認)。(a) read-only だけでは `touch` が通ってファイルができた(回り込みの再現)、(b) `--ignore-rules` を足すと `Operation not permitted` でファイルはできない。ルールと probe のファイルは消し、worktree と main は clean、`~/.codex` には触れていない。範囲外の観測: read-only の reviewer も `~/.codex/memories/MEMORY.md` などのファイルを読める(read-only は書き込みを止めるだけ)。orchestrator も 6 面の行と、テスト 132 / 132 を確認
- 2026-10-03 self-review(cycle 1、3c11bf74): CRITICAL 0 / HIGH 0 / MEDIUM 2 / LOW 2、merge 可。範囲内の経路(後ろからの `-c`、project とユーザー設定の `default_permissions`、v2 と旧式の profile、`.rules`)は塞がっていると確認した。M-1: テストの禁止の検査が 3 つの文字列しか見ず、help に出ない別名(`--yolo` は read-only の指定に勝って danger-full-access になる、`--approve-for-me` は workspace-write になる)を見逃す。M-2: skill と recipe の「reviewer に書き込ませない」は言い過ぎ: (a) ユーザー設定の MCP server は reviewer でも起動し、その tool に sandbox は掛からない(`-c 'mcp_servers={}'` では消えず、server ごとの `enabled=false` か `--ignore-user-config` なら消える)、(b) repo の外も読める(ネットワークは塞がっている)、(c) 逆方向の claude reviewer は Open questions にしかない。L-1: `.codex/config.toml` のコメントが「approval は on-request」と書くが `codex exec` は never で動く(前からある)。L-2: tech-debt の解決のコメントに plan のパスがない。全件を in-cycle で扱う(orchestrator 判断): M-1 は許可の一覧の形に変え(sandbox の指定は read-only だけ、広げる flag はすべて落とす)、M-2 は文を実際に効く範囲(sandbox と `.rules`)に絞り、(a)〜(c) は残るリスクとして tech-debt に記録する。`--ignore-user-config` で MCP も塞ぐ案は、ユーザー設定(独自の provider など)をまるごと無視するので影響が大きく、この PR には入れずメンテナに相談する
- 2026-10-03 work: Slice B は implementer に委譲(ac6300ad、14 ファイル、+158 / -56、push 済み、呼び出しの行は変えていない)。M-1: テストに `check_sandbox_tokens` を足した。`command codex ` から `</dev/null` までだけを読み(表の行の claude の列の `-p` を読まないため)、(i) `--sandbox <v>`、`--sandbox=<v>`、`-s <v>`、`-s<v>`、`sandbox_mode=<v>` はすべて `v = read-only` で、少なくとも 1 つある、(ii) sandbox や承認を広げる・迂回する語(`--dangerously-bypass-approvals-and-sandbox`、`--yolo`、`--approve-for-me`、`--full-auto`、`--add-dir`、`-a` / `--ask-for-approval`、`-p` / `--profile`、`--dangerously-bypass-hook-trust`、`-c` の `approval_policy=`、`default_permissions=`、`sandbox_permissions=`、`sandbox_workspace_write`)があれば落ちる。一覧は codex-cli 0.154.0 の 3 つの help と、観測した隠れた別名から作った。132 / 132(sh と dash)。red: 21 種の変異がそれぞれ 1 件だけ落ちる。M-2: skill(cross-review と plan の 4 面)と recipe の文を、read-only の sandbox は実行するコマンドの書き込みとネットワークを止める、`--ignore-rules` は allow のルールを読ませない、に絞り、ユーザー設定の MCP server と repo の外の読み取りはどちらも対象外、と書いた。tech-debt に残るリスクの行を足した((a) MCP、(b) 読んだ中身がコミットされる triage report に入りうる、(c) 逆方向の claude reviewer)。L-1: `.codex/config.toml` のコメントを「対話の session では approval_policy が効くが、`codex exec` は承認を求めないので、ralph の `codex exec` は read-only の sandbox を明示する」に直した(byte 一致、TOML として読んだ中身は変わらない)。L-2: 解決のコメントと Related の列に plan の archive のパスと report を足した。check-skill-sync、check-sync、shellcheck、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。orchestrator も 132 / 132(sh、dash)、cmp、TOML の一致、呼び出しの行が変わっていないことを確認

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 4 面 8 か所の呼び出しと、それを固定しているテストを特定した
- [x] critical fork なし
- [x] Codex plan advisory(HIGH 1、`/plan` の advisory も含めて plan を更新)
