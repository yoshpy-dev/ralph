# subagent-model-defaults

- Status: Draft
- Owner: Claude Code
- Date: 2026-10-04
- Related request: メンテナの依頼(2026-10-04)。「implementer と verifier、tester は sonnet となっているが、既定で opus で実行するようにしてほしい。reviewer と doc-maintainer は既定で sonnet にしてほしい」
- Related issue: N/A
- Type: chore
- Branch: chore/subagent-model-defaults

## Objective

`/work` の slice と post-implementation pipeline のサブエージェントの既定モデルを、implementer、verifier、tester は `opus`、reviewer と doc-maintainer は `sonnet` にする。frontmatter の値と `model-routing.md` の説明を一致させ、両者のずれをテストで検出できるようにする。

## 調査で確認したこと(2026-10-04、main 11602fed)

- 現在の frontmatter の `model:`: implementer `sonnet`、verifier `sonnet`、tester `sonnet`、doc-maintainer `sonnet`、reviewer `opus`。`.claude/agents/` と `templates/base/.claude/agents/` の 5 ファイルは byte 一致。
- `.codex/agents/*.toml`(+ template)には model の指定がない。Codex 側はこの変更の影響を受けない。
- `.claude/rules/ralph/model-routing.md`(+ template)で値を書いている箇所は 3 つ: tier 表の 2 行(11〜12 行目)、「Standard flow delegation」の「`model: sonnet` pinned in frontmatter」(23 行目)、「Escalating a judgment-heavy slice」の例「`opus` for security-sensitive changes」(50 行目)。この 3 か所は root と template で同じ。両者の差分は org runtime の節と「Where the values live」の 1 行だけで、`scripts/check-sync.sh` の KNOWN_DIFF になっている。
- agent の `model:` の値を検査するテストはない(`tests/test-agent-phase-boundaries.sh` と `tests/test-self-review-scope.sh` は model を見ない)。`scripts/verify.local.sh` の test モードは `tests/test-*.sh` をすべて走らせ、static モードは同じファイルを shellcheck にかける。
- 似た名前で別の仕組みのもの: `RALPH_CLAUDE_REVIEWER_MODEL`(既定 `opus`。`/cross-review` で Codex が運転するときの `claude -p` のレビュアー)、`[org].model_pool` と `[org.roles]`(org runtime の座席)、`templates/base/ralph.toml` のコメントの例 `implementer = ["sonnet"]`(org runtime の役割制限の例)。

## Scope

- `.claude/agents/{implementer,verifier,tester}.md` と template の 3 ファイルの `model:` を `opus` に、`.claude/agents/reviewer.md` と template を `sonnet` にする。doc-maintainer は `sonnet` のまま。
- `.claude/rules/ralph/model-routing.md`(+ template)の tier 表を新しい割り振りに書き直す。「Judgment seats / Procedural seats」の見出しは割り振りと合わなくなるので、席の名前で引ける表にする。23 行目の pin の記述と、50 行目の escalation の例を新しい既定に合わせる。
- 新しいテスト `tests/test-agent-models.sh`: root(`.claude/`)と template(`templates/base/.claude/`)のそれぞれで、`model-routing.md` の tier 表と pin の記述(「`model: <x>` pinned in frontmatter」)が、同じ側の `agents/*.md` の frontmatter と一致することを検査する。あわせて root と template の agent の `model:` が一致することも見る。`check-sync.sh` は `model-routing.md` を丸ごと差分許容にしているので、template 側の表はこのテストでしか守れない(Codex plan advisory の MEDIUM)。

## Non-goals

- `RALPH_CLAUDE_REVIEWER_MODEL`(cross-review の claude レビュアー、既定 `opus`)の変更。
- org runtime のモデル(`[org].model_pool`、`[org.roles]`、`templates/base/ralph.toml` のコメントの例)。
- `.codex/agents/` への model の追加。
- archive 済みの plan とレポートの過去の記述(当時の事実なので直さない)。
- リリース(`/release` は手動。この PR は main に入るだけで、利用先には次のリリースで届く)。

## Assumptions

- `.claude/agents/` と `.claude/rules/ralph/` は core 所有で、利用先では `ralph upgrade` で置き換わる(手で変えた利用先は drift として残り、advisory diff が出る)。AC-5 の fresh scaffold で、配る中身が新しい値になることだけを確かめる。
- escalation の考え方(セキュリティに関わる変更は `opus` で見る)は残す。implementer は既定で `opus` になるので、例は reviewer の呼び出しに移す。逆向きに、機械的な slice は implementer の呼び出しに `sonnet` を渡してよいことも書く。

## Affected areas

- `.claude/agents/implementer.md`、`verifier.md`、`tester.md`、`reviewer.md`(doc-maintainer は変更なし)
- `templates/base/.claude/agents/` の同じ 4 ファイル
- `.claude/rules/ralph/model-routing.md`、`templates/base/.claude/rules/ralph/model-routing.md`
- `tests/test-agent-models.sh`(新規)

## Design decisions

- tier 表は「席 / モデル / 担う作業」の形にし、agent 名をバッククォートで書く。テストが表の行から agent 名とモデルを読めるようにするため。
- テストは表と frontmatter の一致を見る形にする(期待値をテストに直書きしない)。モデルの割り振りを次に変えるときも、表と frontmatter を同時に直せばテストは通り、片方だけ直すと落ちる。
- テストは root と template の両方の組(表と frontmatter)に同じ検査を当てる。Codex plan advisory(MEDIUM 1 件)の指摘で、`check-sync.sh` が `model-routing.md` を丸ごと差分許容にしているため template 側の表のずれを検出する手段がなかった。メンテナの決定: plan を更新。
- Critical forks: None(割り振りはメンテナが指定済み。escalation の例の移し先は 1 行の記述で、後から直しても 1 slice より小さい)

## Acceptance criteria

- [ ] AC-1: `.claude/agents/` の frontmatter が implementer `opus`、verifier `opus`、tester `opus`、reviewer `sonnet`、doc-maintainer `sonnet` になっている。template の 5 ファイルは root と byte 一致。
- [ ] AC-2: `model-routing.md`(+ template)の tier 表が AC-1 の割り振りを示し、23 行目相当の pin の記述が `model: opus` になり、escalation の段落が新しい既定と矛盾しない(「implementer を opus に上げる」とは書かない)。tier 表から「Where the values live」の直前までは root と template で同じ文面にする(`diff` で、差分が既存の org runtime の節と「Where the values live」の 1 行だけであることを確かめる)。
- [ ] AC-3: `tests/test-agent-models.sh` が通る。red: root と template のそれぞれで、(a) agent の `model:` を 1 つ変える、(b) tier 表の 1 行のモデルを変える、(c) tier 表から agent 名を 1 つ消す、(d) pin の記述を旧値(`model: sonnet`)に戻す、のどれでも落ち、落ちたファイルと agent 名を出す。template 側だけを変えた場合も落ちる。
- [ ] AC-4: `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-template.sh` が green。
- [ ] AC-5: `go run ./cmd/ralph init --yes` で scratch に作った fresh scaffold の `.claude/agents/` が AC-1 の値になっている。
- [ ] AC-6: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。

## Implementation outline

1. Slice A(implementer): agent の 8 ファイル、`model-routing.md` の 2 コピー、`tests/test-agent-models.sh` を 1 コミットで。red の証拠は AC-3 の mutation。
2. pipeline: self-review → verify → test → sync-docs → cross-review → PR。

## Verify plan

- Static analysis checks: `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`(新しいテストの shellcheck を含む)、`./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-template.sh`。
- Spec compliance criteria to confirm: AC-1〜AC-6。
- Documentation drift to check: `model-routing.md` 以外で seat のモデルを書いている文書がないか(`git grep -nwE 'sonnet|opus'` を archive、reports、org runtime を除いて)。`.claude/rules/ralph/subagent-policy.md`、`.claude/skills/work/SKILL.md`、`README.md`、`AGENTS.md`、`.codex/README.md`。
- Evidence to capture: AC-3 の red の出力、AC-5 の scaffold の `grep '^model:'`。

## Test plan

- Unit tests: `tests/test-agent-models.sh`(root と template のそれぞれで表と pin の記述の解析、frontmatter の読み取り、両者の照合、root と template の frontmatter の比較)。mutation は一時ディレクトリにコピーした木で行い、作業ツリーを書き換えない。
- Integration tests: AC-5 の fresh scaffold。
- Regression tests: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh`(既存の `tests/test-*.sh` と `go test ./...`)。
- Edge cases: frontmatter に `model:` がない agent(テストは落ちる。省略すると親のモデルを継ぐので、model-routing の規則どおり失敗にする)、表にあるが `.claude/agents/` にない名前(implementer 以外の汎用の語を agent 名と誤認しない)、同じ agent が表の 2 行に出る場合(落ちる)。
- Evidence to capture: テストの pass 件数、mutation ごとの失敗の出力。

## Risks and mitigations

- 費用が増える: implementer、verifier、tester は呼び出し回数が多い席で、sonnet から opus に変わる。メンテナの判断として受け入れる。下げたい slice は Task の呼び出しに `model` を渡せば下げられることを escalation の段落に書く。
- reviewer の質が下がる可能性: セキュリティに関わる差分では reviewer の呼び出しに `opus` を渡す、と escalation の段落に書く。既定は sonnet のまま。
- 利用先で手で直した agent ファイルは `ralph upgrade` で置き換わらない(drift として残る)。既存の upgrade の挙動どおりで、この PR では変えない。

## Rollout or rollback notes

- rollback は frontmatter 4 行と `model-routing.md` の該当箇所を戻すだけ。テストは表と frontmatter の一致を見るので、両方を戻せば通る。
- 利用先には次の `ralph` のリリース後、`ralph upgrade` で届く。

## Open questions

- なし

## Progress notes

- 2026-10-04 plan: Codex plan advisory(gpt-6-astra、xhigh、`sandbox: read-only`、`codex rc=0`、`-o` 1259 バイト)は MEDIUM 1: template 側の `model-routing.md` の表と pin の記述を検査する手段がない(`check-sync.sh` はファイル全体を差分許容)。メンテナの決定: plan を更新。テストを root と template の両方に当て、AC-2 に `diff` での確認、AC-3 に template 側だけの mutation を足した
- 2026-10-04 work: Slice A は implementer に委譲(173cf76d、11 ファイル、+584 / -18、push 済み)。この session の agent 定義は main のチェックアウトから読まれるので、implementer は旧既定の sonnet で動いた。frontmatter 8 ファイル、`model-routing.md` の 2 コピー(tier 表を「Implementation and verification seats / Review and doc seats」に、変更日の注記、pin の記述、escalation の段落を「Overriding a seat's default」に)、`tests/test-agent-models.sh`(100755)。テストは 36 / 36。self-test は root と template のそれぞれで mutation (a)〜(d) と、追加の (e) `model:` なし、(f) 同じ agent が 2 行、(g) agent でない backtick 語は無視、を検出する。template だけの `model:` の変更と、agent ファイルの集合差も検出する。逸脱: (1) real tree に FAIL があるときは self-test を SKIP する(壊れた木を複製して偽の失敗を大量に出さないため。real tree 側の FAIL はそのまま出る)、(2) Co-Authored-By は実行モデルに合わせて Sonnet 5.5。手動の mutation 3 通り(root の tester、template の tester だけ、template の pin だけ)はいずれも exit 1 でファイルと agent 名を出し、戻すと 36 / 36。AC-5 の fresh scaffold は implementer・verifier・tester が opus、reviewer・doc-maintainer が sonnet、pin の記述は `model: opus`。`check-sync.sh`(DRIFTED 0、KNOWN_DIFF 5)、`check-skill-sync.sh`、`check-template.sh`、shellcheck、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。root と template の `model-routing.md` の差分は既存の 3 hunk だけ。orchestrator も HEAD の一致、porcelain が空、テスト 36 / 36、差分を確認

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
