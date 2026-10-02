# cross-review-codex-read-only

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-03
- Related request: issue #197。`/cross-review` の codex reviewer(`command codex -m … -c model_reasoning_effort=… exec review --base "$BASE" -o <file> </dev/null`)は sandbox を指定しないので、trust 済みの project では `.codex/config.toml` のトップレベルの `sandbox_mode = "danger-full-access"` と、`exec` の `approval: never` で動く。レビューする差分に紛れた指示に codex が従えば、承認なしでコマンドを実行できる。PR #198 の self-review(M-2)で見つかった
- Related issue: 197
- Type: security
- Branch: security/cross-review-codex-read-only

## Objective

`/cross-review` の codex reviewer を read-only の sandbox で動かす。レビューする差分の中身にかかわらず、reviewer の codex がファイルを書いたり、書き込みを伴うコマンドを実行したりできないようにする。

## 調査で確認したこと(2026-10-03、main c9da0a27)

- 呼び出しは 4 面(`.claude/skills/cross-review/SKILL.md`、`.agents/skills/cross-review/SKILL.md`、template の 2 つ)に、それぞれ 2 か所ある: step 4 の fenced block(58 行目)と「CLI execution modes」の表(167 行目)。
- `codex exec review --help`(codex-cli 0.154.0)に `--sandbox` はなく、`-c, --config <key=value>` はある。PR #198 の reviewer の scratch の probe で、`-c sandbox_mode=read-only` を付けると header が `sandbox: read-only` になることを確認済み(認証なし。`-o` が書かれるかは未確認)。
- `/plan` の Codex plan advisory は `exec --sandbox read-only` を渡している(`.claude/skills/plan/SKILL.md` step 11.c)。
- `tests/test-codex-exec-invocation.sh` が、4 面の `/plan` と `/cross-review` の codex の呼び出しの行に、`command codex `、`</dev/null`、`-m "${RALPH_CODEX_REVIEWER_MODEL:-`、`model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-`、`-o` の 5 つがあることを固定している(sandbox は見ていない)。
- 呼び出しの形を説明している文書: `docs/recipes/codex-setup.md` 85〜100 行(+ template)、`.codex/config.toml` の profile の節のコメント(「`/cross-review` は `-m` と `-c model_reasoning_effort` を明示する」、+ template)。
- `docs/tech-debt/README.md:147` にこの件の行がある(PR #198 で追加)。
- 逆方向(Codex が driver で reviewer が claude)は `claude -p --model … --permission-mode auto --output-format json` で呼ぶ。この issue の範囲外(Open questions に記録)。

## Scope

- 4 面の cross-review の codex の呼び出し(各 2 か所)に `-c sandbox_mode=read-only` を足す。ほかの `-c` と同じく `exec` の前に置く。
- `tests/test-codex-exec-invocation.sh`: どの codex の呼び出しの行も read-only の sandbox を指定していることを検査に足す(`/plan` は `--sandbox read-only`、`/cross-review` は `sandbox_mode=read-only`)。
- `docs/recipes/codex-setup.md`(+ template)の呼び出しの説明に、両方の呼び出しが read-only の sandbox で動くことと、その理由(project の設定の `danger-full-access` を引き継がない)を足す。
- `.codex/config.toml`(+ template)の profile の節の、`/cross-review` の呼び出しを説明するコメントを新しい形に合わせる(値は変えない)。
- `docs/tech-debt/README.md` のこの件の行を、ファイルの慣習(取り消し線と `RESOLVED` のコメント)で解決済みにする。

## Non-goals

- `.codex/config.toml` のトップレベルの `sandbox_mode = "danger-full-access"` の変更(project の codex の使い方全体に関わる別の判断)。
- 逆方向の claude reviewer(`--permission-mode auto`)の権限の見直し。
- `/plan` の advisory の変更(すでに read-only)。
- org runtime の codex 座席の sandbox(`internal/org/permissions.go` の別の仕組み)。

## Assumptions

- `-o` のファイルは sandbox の外で codex CLI 自身が書くので、read-only でも書かれる(AC-3 で確かめる)。

## Affected areas

- `.claude/skills/cross-review/SKILL.md`、`.agents/skills/cross-review/SKILL.md`、`templates/base/.claude/skills/cross-review/SKILL.md`、`templates/base/.agents/skills/cross-review/SKILL.md`
- `tests/test-codex-exec-invocation.sh`
- `docs/recipes/codex-setup.md`、`templates/base/docs/recipes/codex-setup.md`
- `.codex/config.toml`、`templates/base/.codex/config.toml`(コメントだけ)
- `docs/tech-debt/README.md`

## Design decisions

- sandbox は `read-only` にする(`workspace-write` にはしない)。reviewer の役割に書き込みは要らず、`workspace-write` では reviewer がレビュー中の worktree を書き換えられてしまう。`/plan` の advisory と同じ扱いになる。代わりに、reviewer は書き込みを伴うコマンド(`go test` のビルドキャッシュなど)を動かせなくなる。テストは pipeline の tester が別に走らせるので、受け入れる。
- 指定は `-c sandbox_mode=read-only`(`exec review` に `--sandbox` がないため)。
- Critical forks: None(issue と `/plan` の advisory で方針が決まっている)

## Acceptance criteria

- [ ] AC-1: 4 面の cross-review の codex の呼び出し(各 2 か所)がすべて `-c sandbox_mode=read-only` を含む。`./scripts/check-skill-sync.sh` と `./scripts/check-sync.sh` が green。
- [ ] AC-2: `tests/test-codex-exec-invocation.sh` が、どの codex の呼び出しの行にも read-only の sandbox の指定があることを検査する。red: 4 面のどれか 1 か所から外すと落ちる。`/plan` の `--sandbox read-only` を外しても落ちる。
- [ ] AC-3: 実際の codex(いつもの認証)で、新しい形の cross-review の呼び出しを、差分のある branch(この branch)に対して 1 回実行し、`codex rc=0`、`-o` のファイルが空でない、log の header が `sandbox: read-only` になる。main のチェックアウトで codex を動かさない。実行後に main のチェックアウトが clean であることを確かめる。
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

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [ ] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created

## Readiness checklist

- [x] 4 面 8 か所の呼び出しと、それを固定しているテストを特定した
- [x] critical fork なし
- [ ] Codex plan advisory
