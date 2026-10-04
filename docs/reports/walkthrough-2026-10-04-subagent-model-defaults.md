# Walkthrough: subagent-model-defaults

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-subagent-model-defaults.md
- Branch: chore/subagent-model-defaults(base main 11602fed)
- Diff: plan・報告・insight を除くと 14 files、+765 / -24。うち 733 行が新しいテスト `tests/test-agent-models.sh`

## 何を変えたか

サブエージェントの既定モデルを、メンテナの指定どおりに入れ替えた。

| agent | 変更前 | 変更後 |
| --- | --- | --- |
| implementer | sonnet | opus |
| verifier | sonnet | opus |
| tester | sonnet | opus |
| reviewer | opus | sonnet |
| doc-maintainer | sonnet | sonnet(変更なし) |

値そのものの変更は frontmatter の `model:` の 4 行で、root と template の 8 ファイルにある。残りの差分は、説明の文書と、両者のずれを検出するテスト。

## 読む順

1. `.claude/agents/{implementer,verifier,tester,reviewer}.md`(template と byte 一致)
   - それぞれ 5 行目の `model:` だけが変わる。
2. `.claude/rules/ralph/model-routing.md`(template と同じ文面)
   - tier 表の列見出しを `Seat group | Model | Agents and typical work` にした。旧来の「Judgment seats / Procedural seats」は新しい割り振りと合わないので、`Implementation and verification seats`(opus)と `Review and doc seats`(sonnet)の 2 行に置き換えた。
   - 「Standard flow delegation」の pin の記述を `model: opus` にした。
   - 「Escalating a judgment-heavy slice」を「Overriding a seat's default」に改めた。例は、セキュリティに関わる差分で reviewer の呼び出しに `opus` を渡す場合と、機械的な slice で implementer の呼び出しに `sonnet` を渡す場合。
   - root だけ、「Where the values live」に新しいテストの bullet がある(`tests/` は scaffold に出ないため)。
3. `.codex/README.md`(template と byte 一致)
   - implementer の説明にあった「the `sonnet` tier applies to the Claude Code counterpart」を、モデル名を書かずに frontmatter と tier 表を指す文にした(/verify の D-1)。値の写しを 1 つ減らしたので、次に割り振りを変えても、この文はずれない。
4. `tests/test-agent-models.sh`(新規、meta-repo 専用)
   - 検査の本体(`check_side` とその下の関数): root と template のそれぞれで、tier 表の行から「モデル」と「agent 名」を読み、同じ側の `agents/*.md` の frontmatter と照合する。agent 名は、行の 3 列目のバッククォートのうち `agents/<name>.md` が実在するものだけを数える。pin の記述(「`model: <x>` pinned in frontmatter」)は implementer の frontmatter と照合する。
   - `check_cross`: root と template で、agent ファイルの集合と各 agent の `model:` が一致するかを見る。
   - self-test(`self_test_*`): 一時ディレクトリに写した木に mutation を当て、検査が落ちることを確かめる。当てる mutation は、`model:` の変更、表の行のモデルの変更、表からの agent 名の削除、pin の記述の旧値への戻しと削除、`model:` の欠落、同じ agent を 2 行に書くこと、template だけにある agent など。real tree に FAIL があるときは self-test を SKIP する(壊れた木を写すと、同じ原因の失敗が mutation ごとに重複して出るため)。
   - 期待値はテストに直書きしていない。次に割り振りを変えるときは、表と frontmatter を同時に直せば通り、片方だけ直すと落ちる。
5. `docs/tech-debt/README.md`
   - self-test の残りの穴を 1 行で記録した。踏まれない FAIL 分岐、pin の文の折り返しに弱い mutator、引用符付きの `model:` の扱い。

## 変えていないもの

- `/cross-review` で Codex が運転するときの claude レビュアー(`RALPH_CLAUDE_REVIEWER_MODEL`、既定 `opus`)。pipeline の reviewer サブエージェントとは別の仕組み。
- org runtime の `[org].model_pool`、`[org.roles]`、`templates/base/ralph.toml` のコメントの例。
- `.codex/agents/*.toml`。もともとモデルを指定していない。

## 確かめたこと

- `tests/test-agent-models.sh` は sh、dash、ubuntu の dash + mawk で 47 / 47。
- `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` と `./scripts/run-test.sh`(full)が green。
- tester が implementer とは別に当てた mutation(root の verifier、template の reviewer だけ、template の表の行、root の表からの tester の削除、root の pin)は、どれも exit 1 でファイルと agent 名を出した。
- `go run ./cmd/ralph init --yes` で作った fresh scaffold の `model:` が新しい既定になっている。
- cross-review(Codex、read-only)の指摘は 0 件。

## 確かめていないこと

- Claude Code が新しい frontmatter のモデルで実際にサブエージェントを起動するか。この PR の作業中は main のチェックアウトの定義が読まれていたので、Slice A の implementer は旧既定の sonnet で動いた。Slice B 以降は Agent の `model` 引数で新しい既定を明示した。
- 利用先への配布。次の `ralph` のリリースと `ralph upgrade` で届く。
