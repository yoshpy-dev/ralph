# subagent-model-defaults

- Status: PR created (#200), awaiting CI and merge
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
- `.codex/README.md` の 74 行目(+ `templates/base/.codex/README.md`、byte 一致)も implementer の tier を「the `sonnet` tier applies to the Claude Code counterpart」と書いていた。計画時の調査は `model-routing.md` だけを見ていて拾えず、/verify の D-1 で見つかった。

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
- `.codex/README.md`、`templates/base/.codex/README.md`(74 行目の implementer の説明。/verify の D-1 で追加。モデル名を書かず `model-routing.md` の tier 表を指す文に直す。両コピーは byte 一致を保つ)
- `tests/test-agent-models.sh`(新規)

## Design decisions

- tier 表は「席 / モデル / 担う作業」の形にし、agent 名をバッククォートで書く。テストが表の行から agent 名とモデルを読めるようにするため。
- テストは表と frontmatter の一致を見る形にする(期待値をテストに直書きしない)。モデルの割り振りを次に変えるときも、表と frontmatter を同時に直せばテストは通り、片方だけ直すと落ちる。
- テストは root と template の両方の組(表と frontmatter)に同じ検査を当てる。Codex plan advisory(MEDIUM 1 件)の指摘で、`check-sync.sh` が `model-routing.md` を丸ごと差分許容にしているため template 側の表のずれを検出する手段がなかった。メンテナの決定: plan を更新。
- Critical forks: None(割り振りはメンテナが指定済み。escalation の例の移し先は 1 行の記述で、後から直しても 1 slice より小さい)

## Acceptance criteria

- [x] AC-1: `.claude/agents/` の frontmatter が implementer `opus`、verifier `opus`、tester `opus`、reviewer `sonnet`、doc-maintainer `sonnet` になっている。template の 5 ファイルは root と byte 一致。
- [x] AC-2: `model-routing.md`(+ template)の tier 表が AC-1 の割り振りを示し、23 行目相当の pin の記述が `model: opus` になり、escalation の段落が新しい既定と矛盾しない(「implementer を opus に上げる」とは書かない)。tier 表から「Where the values live」の直前までは root と template で同じ文面にする(`diff` で、差分が既存の org runtime の節と、「Where the values live」にある root だけの bullet(既存の `defaults_sync_test.go` の bullet と新しい `tests/test-agent-models.sh` の bullet)だけであることを確かめる)。
- [x] AC-3: `tests/test-agent-models.sh` が通る。red: root と template のそれぞれで、(a) agent の `model:` を 1 つ変える、(b) tier 表の 1 行のモデルを変える、(c) tier 表から agent 名を 1 つ消す、(d) pin の記述を旧値(`model: sonnet`)に戻す、のどれでも落ち、落ちたファイルと agent 名を出す。template 側だけを変えた場合も落ちる。
- [x] AC-4: `./scripts/check-sync.sh`、`./scripts/check-skill-sync.sh`、`./scripts/check-template.sh` が green。
- [x] AC-5: `go run ./cmd/ralph init --yes` で scratch に作った fresh scaffold の `.claude/agents/` が AC-1 の値になっている。
- [x] AC-6: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。

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
- 2026-10-04 self-review(cycle 1、9b6b491e、reviewer は新しい既定に合わせて sonnet を明示): CRITICAL 0 / HIGH 0 / MEDIUM 2 / LOW 5、merge 可。F-1(MEDIUM): tier 表の下の「2026-10-04 に変えた」という段落は template 経由で全 scaffold に出るが、新しい利用先には何から変わったのか分からず、次の変更で古くなる。F-2(MEDIUM): `check_side` が 111 行で 4 つの仕事を持ち、大域変数の共有が subshell 前提なのにコメントがない。F-3: 表の列見出しが `Examples` のままで、表が機械検査されることが「Where the values live」にない。F-4: テストの限界(pin の照合は implementer だけ、agent ファイルを消して表の行を残すと通る)がヘッダにない。F-5: self-test のない分岐(pin の記述を丸ごと消す、template だけに余分な agent、ディレクトリがない)と、両側に `model:` がないときの比較の PASS 行。F-6: `_root` と `ROOT_SIDE` の「root」が別の意味、fixture の agent 名の固定に理由がない。F-7: `chore:` は `.goreleaser.yml` の changelog から外れる。orchestrator 判断: F-1〜F-6 を Slice B で直す(F-1 は段落を両コピーから削り、変更の経緯は plan と git の履歴に残す)。F-7 はコードを変えず、PR の本文に利用先への影響を書く(merge commit の件名は changelog に残る)。逸脱: なし(段落は plan の範囲外だったので、削っても AC は変わらない)
- 2026-10-04 work: Slice B は implementer に委譲(de01c50f、3 ファイル、+274 / -100、push 済み。新しい既定に合わせて opus を明示)。F-1: 日付つきの段落を両コピーから削除。F-3: 列見出しを `Seat group | Model | Agents and typical work` にし、root の「Where the values live」にだけテストの bullet を足した。F-2: `check_side` を 6 関数に分け、最長は `check_cross` の 43 行。大域変数の共有が `run_checker` の subshell 前提であることをコメントに書いた。F-4: ヘッダに限界を 2 つ書いた。F-5: self-test に (h) pin の記述を丸ごと消す、(i) template だけに余分な agent、(j) 片側の agents ディレクトリか `model-routing.md` がない、を足した。両側とも `model:` がないとき比較側は PASS 行を出さない(旧ロジックに戻すと 46 / 47 で落ちる)。F-6: `_root` を `_tree` に改名し、fixture の agent に理由のコメントを付け、fixture がないと明示の文言で落ちる。テストは sh と dash で 47 / 47。手動の mutation(template の pin の記述を消す)で exit 1、戻すと 47 / 47。`check-sync.sh`(IDENTICAL 159 / DRIFTED 0 / KNOWN_DIFF 5)、`check-template.sh`、`check-template-purity.sh`、shellcheck、`RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` green。root と template の差分は既存の org runtime の 2 hunk と root だけの bullet。orchestrator も HEAD の一致、porcelain が空、47 / 47、差分を確認
- 2026-10-04 self-review addendum(Slice B、89ea2a43、reviewer は sonnet を明示): CRITICAL / HIGH / MEDIUM 0、merge 可。F-1〜F-4 と F-6 は fixed、F-5 は partially。残りは LOW 2 件: F-5 の一部(self-test が踏まない FAIL 分岐 5 つ)と N-1(`mut_pin` / `mut_drop_pin` が行単位の sed なので、pin の文を折り返すと self-test が「mutation did not apply」で赤になる。checker 本体は影響なし)。orchestrator 判断(consult-advisor に相談、判定は案 A): 2 件とも `docs/tech-debt/README.md` に 1 行で記録し、/verify に進む。N-1 を正しく直すには POSIX awk の全文置換と折り返した pin の fixture が要り、どの AC も進めない。addendum の report には「cycle 2(既定の cap 2 回のうち 2 回目)」とあるが誤りで、pipeline の cycle は 1 のまま(`cycle-count.json` は `{"cycle": 1}`。cycle を進めるのは cross-review の ACTION_REQUIRED によるやり直しだけ)
- 2026-10-04 verify(26bd54e4、verifier は opus を明示): pass。AC-1、AC-2、AC-4、AC-5 は満たす。AC-3 と AC-6 は静的な範囲で満たし、実行は /test。`run-static-verify.sh`(full)、`check-sync.sh`、`check-skill-sync.sh`、`check-template.sh`、`check-template-purity.sh`、shellcheck はすべて exit 0。D-1(MEDIUM): `.codex/README.md:74`(+ template)が「the `sonnet` tier applies to the Claude Code counterpart (`.claude/agents/implementer.md`)」のままで、新しい既定と矛盾する(調査の節で拾えていなかった)。/sync-docs で直す。D-3: AC-2 の文面は「Where the values live の 1 行だけ」だが、Slice B で root だけの bullet が増えた(Progress notes に記録済み)。D-4: tech-debt の行が archive 後の plan のパスを前方参照している(/pr で解消)。テストの実行は /test の担当
- 2026-10-04 test(c8eb7e81、tester は opus を明示): pass。`tests/test-agent-models.sh` は sh と dash で 47 / 47(ubuntu:24.04 の dash + mawk でも 47 / 47)。`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` は exit 0(shell 33 ファイル 1,523 件、Go 8 パッケージ。`go test -count=1 ./...` でも ok)。AC-3 の red を implementer とは別に当てた(`git archive` で取り出した写しで、root の verifier を sonnet、template の reviewer だけを opus、template の表の行のモデル、root の表から tester を削除、root の pin を `model: sonnet`)。どれも exit 1 でファイルと agent 名を出した。edge case(`model:` なし、agent でない backtick 語、同じ agent が 2 行)も期待どおり。G-1 は tech-debt の N-1 の再現。G-4(引用符付きの `model: "opus"` が赤になる)と G-5(`implementer.md` がないときの文言が不正確)は LOW で、tech-debt の同じ行に (c) として足した。G-6 は verify の D-1 と同じ
- 2026-10-04 sync-docs(52de07a2、doc-maintainer は sonnet を明示): verify の D-1 を直した。`.codex/README.md:74`(+ template、byte 一致)はモデル名を書かず、Claude Code 側の implementer は frontmatter のモデルで動き、tier 表は `.claude/rules/ralph/model-routing.md` にある、と指す文にした。D-3: AC-2 の `diff` の期待を root だけの bullet 2 つに合わせ、Affected areas と調査の節に `.codex/README.md` を足した。seat のモデルを書いた文書のスイープでは、ほかに新しい既定と食い違う記述はなかった。`check-sync.sh`(DRIFTED 0)、`check-skill-sync.sh`、`cmp`、`tests/test-agent-models.sh` 47 / 47、`check-template.sh`、`check-template-purity.sh` green
- 2026-10-04 cross-review(cycle 1/2、HEAD 65dc9f5d): driver claude、reviewer codex(gpt-6-astra、xhigh、`sandbox: read-only`、`approval: never`)。`codex rc=0`、`-o` 242 バイトで complete。指摘 0 件(root と template のモデル設定と routing の文書が一致、構文と ShellCheck と独自の整合確認が通った。fixture を書くテストは read-only のため未実行で、tester が実行済み)。Case C で /pr へ。triage は `docs/reports/cross-review-triage-subagent-model-defaults.md`

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [x] Verification artifact created
- [x] Test artifact created
- [x] PR created
