# Verify report: subagent-model-defaults

- Date: 2026-10-04
- Plan: docs/plans/active/2026-10-04-subagent-model-defaults.md
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: branch chore/subagent-model-defaults の HEAD 60d98954 と base main 11602fed の差分。実装のコミットは 173cf76d(Slice A)と de01c50f(Slice B)で、残りは plan と report の記録。仕様への適合(AC-1〜AC-6)、static analysis、文書のずれを見た。behavioral test(`tests/test-agent-models.sh` の実行、`./scripts/run-test.sh`、`go test`)は実行していない(/test の担当)
- Evidence: `docs/evidence/verify-2026-10-04-subagent-model-defaults.log`(`docs/evidence/*.log` は gitignore 対象なので commit しない。`run-verify.sh` 自身が書いたログは `docs/evidence/verify-2026-10-04-094401.log`)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC-1: frontmatter が implementer / verifier / tester は `opus`、reviewer / doc-maintainer は `sonnet`。template の 5 ファイルは root と byte 一致 | met | `grep -n '^model:'` の結果は implementer、verifier、tester が `5:model: opus`、reviewer と doc-maintainer が `5:model: sonnet`(`.claude/agents/*.md:5`)。5 ファイルとも `model:` は先頭の `---` 2 行の間にあり、`model` を含む行は各ファイル 1 行だけ。`cmp` で root と `templates/base/.claude/agents/` の 5 組が一致。差分は 8 ファイルとも 1 行ずつ(`git diff --stat main...HEAD`)。doc-maintainer は変更なし |
| AC-2: tier 表、pin の記述、escalation の段落。tier 表から「Where the values live」の直前まで root と template が同じ | met(文面の注記は D-3) | tier 表(`.claude/rules/ralph/model-routing.md:8-13`)は `Implementation and verification seats` が `opus` で `implementer`、`verifier`、`tester` を、`Review and doc seats` が `sonnet` で `reviewer`、`doc-maintainer` を挙げる。pin の記述は `:23` の `` `model: opus` pinned in frontmatter ``。`:49-52` の「Overriding a seat's default」は reviewer に `opus` を、implementer に `sonnet` を渡す例で、「implementer を opus に上げる」とは書いていない。`## Tier table` から `## Org runtime model receipts` の直前までを両コピーから切り出すと `cmp` で同一。ファイル全体の `diff` を main 時点の `diff` と比べると、残る差は既存の org runtime の 2 hunk(行番号が 2 行ずれただけ)と「Where the values live」の既存の 1 行で、増えたのは root だけの bullet 3 行(`:134-136`)だけ |
| AC-3: `tests/test-agent-models.sh` が通り、(a)〜(d) の mutation と template 側だけの変更で落ち、ファイル名と agent 名を出す | 静的には met(実行は /test) | テストを全文読んだ。`self_test_side_drift`(`:570`)は root と template のそれぞれに (a) `:584`、(b) `:590`、(c) `:596`、(d) `:602`、(h) `:608` を当てる。(d) は implementer の現在値の反対、つまり `sonnet` に戻す。`assert_detected`(`:524`)は非 0 終了と `FAIL:` 行を要求し、(a)〜(c) では agent ファイルのパスと `tester` を、(d) では `model-routing.md` のパスと `implementer` を出力に求める。template 側だけの変更は `self_test_side "$TEMPLATE_SIDE"` と `self_test_cross_models` の `:666` が見る。mutation は `new_fixture`(`:363`)が作る `$TMP_ROOT/case-N` の複製にだけ当たる。PASS の件数を数え直すと、real tree が 17(各側で agent 5 と pin 1、比較で 5)、self-test が 30(guard 1、各側 11、比較 7)、合計 47 で、plan の「47 / 47」と合う。red の記録は plan の Progress notes にある(手動の mutation 3 通りと、template の pin の削除で exit 1)。別に、表の各行の model と frontmatter を shell で突き合わせ、両側とも 5 agent が一致した。テストそのものは実行していない |
| AC-4: `check-sync.sh`、`check-skill-sync.sh`、`check-template.sh` が green | met | `./scripts/check-sync.sh` exit 0(IDENTICAL 159 / DRIFTED 0 / ROOT_ONLY 0 / TEMPLATE_ONLY 11 / KNOWN_DIFF 5)。`./scripts/check-skill-sync.sh` exit 0(13 skill)。`./scripts/check-template.sh` exit 0(「Template structure looks good.」) |
| AC-5: fresh scaffold の `.claude/agents/` が AC-1 の値 | met | scratchpad で `go run ./cmd/ralph init --yes <scratch>/ac5-scaffold` を実行し exit 0。`grep '^model:' .claude/agents/*.md` は implementer、tester、verifier が `opus`、reviewer と doc-maintainer が `sonnet`。5 ファイルとも worktree の `.claude/agents/` と `cmp` で同一。scaffold の `model-routing.md` は template のコピーと同一で、`:23` は `model: opus`。root だけの bullet(`tests/test-agent-models.sh` への参照)は scaffold に入っていない。scaffold に `tests/` はない |
| AC-6: `RALPH_VERIFY_SCOPE=full ./scripts/run-verify.sh` が green | partially(static の半分) | `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` は exit 0(「All verifiers passed.」)。このラッパーは `HARNESS_VERIFY_MODE=static` で `run-verify.sh` を呼ぶので、test の半分(`tests/test-*.sh` と `go test ./...`)はここでは走っていない。plan の Progress notes では implementer が full の `run-verify.sh` を green と記録している。test の半分は /test で確かめる |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` | pass(exit 0) | shellcheck(hooks、verify 系、`tests/test-*.sh`。`--severity=warning`)、`sh -n` 20 ファイル、`jq -e` 2 ファイル、Codex hook の guard 3 つ、`check-sync.sh`、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、golang(gofmt ok、lint「0 issues.」)、branch secret scan(`11602fed..60d98954` は clean) |
| `./scripts/check-sync.sh` | pass(exit 0) | DRIFTED 0 |
| `./scripts/check-skill-sync.sh` | pass(exit 0) | 13 skill が一致 |
| `./scripts/check-template.sh` | pass(exit 0) | FAIL 行なし |
| `./scripts/check-template-purity.sh` | pass(exit 0) | template に meta-repo 固有の参照なし |
| `shellcheck -S warning tests/test-agent-models.sh` | pass(exit 0) | 参考に severity を指定せずに回すと SC2016(info)が 3 件出る: `:242`、`:489`、`:494`。どれも backtick を正規表現の文字として single quote に入れた箇所で、展開させる意図はない。誤検知で、warning の gate より下 |
| `sh -n` と `dash -n tests/test-agent-models.sh` | pass | 構文のみ |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `.codex/README.md:74`、`templates/base/.codex/README.md:74` | no(D-1) | 下の D-1 |
| `.claude/rules/ralph/model-routing.md`(+ template) | yes | AC-2 のとおり |
| `.claude/rules/ralph/subagent-policy.md`(+ template) | yes | モデル名を書いていない。`:3` が「Model tier assignment per seat is defined in `model-routing.md`」と参照するだけ |
| `.claude/skills/work/SKILL.md:34`(+ mirror) | yes | implementer への委譲の説明で、モデル名はなく `model-routing.md` を参照する |
| `README.md`、`AGENTS.md`、`CLAUDE.md` | yes | seat のモデルの記述はない。`model` を含む行は cross-model review と `.codex/config.toml` の説明だけ |
| `docs/recipes/`、`docs/quality/` | yes | `--model` は org runtime の codex 座席と codex の呼び出しの話。`docs/quality/quality-gates.md:77` は org の model pool。どれも別の仕組み |
| 変えてはいけない別の仕組み | yes(未変更) | `scripts/ralph-config.sh:35`(+ template)の `RALPH_CLAUDE_REVIEWER_MODEL` 既定 `opus`、cross-review SKILL の `:-opus` fallback(`.claude/skills/cross-review/SKILL.md:63` ほか 4 コピー)、`RALPH_ORG_MODEL_POOL`(`scripts/ralph-config.sh:57`)、`internal/config/config.go:143-144`、`templates/base/ralph.toml:31-32` と `:53-54`(org runtime の pool と役割制限の例)。`git diff main...HEAD` でこれらのパスと `.codex/agents/`、`templates/base/.codex/` に差分はない |
| `docs/tech-debt/README.md:151` | yes(D-4 を注記) | 新しい行は self-review addendum が挙げた 6 項目(self-test が踏まない FAIL 分岐 5 つと N-1)をすべて含む。関数名で書かれていて、名前は現在のテストと一致する |
| plan | partially(D-2、D-3) | 下の D-2、D-3 |
| self-review report | 注記のみ(D-5) | 下の D-5 |

### D-1(MEDIUM、/sync-docs で直す): Codex README が implementer を `sonnet` の tier と書いている

`.codex/README.md:74` と `templates/base/.codex/README.md:74`(両者は byte 一致で、`check-sync.sh` の IDENTICAL 側)に次の文がある。

> Like the other Codex custom agents, no per-agent model is pinned here — the `sonnet` tier applies to the Claude Code counterpart (`.claude/agents/implementer.md`)

`.claude/agents/implementer.md:5` は今は `model: opus` なので、この文は新しい既定と矛盾する。main では正しかった文が、この変更で古くなった。template 経由で全 scaffold に出る(AC-5 で作った scaffold の `.codex/README.md:74` にも同じ文がある)。`tests/test-agent-models.sh` は `model-routing.md` しか読まないので、この文はテストで守られていない。plan の Verify plan はこのファイルを確認先に挙げている。

直し方は 2 つある。(1) 両コピーの `sonnet` を `opus` にする。(2) モデル名を書かず、「the Claude Code counterpart's tier is set by its frontmatter (see `.claude/rules/ralph/model-routing.md`)」のように参照にする。(2) のほうが値の写しが 1 つ減り、次に割り振りを変えたときに同じずれが起きない。どちらでも両コピーを同じ文面にしないと `check-sync.sh` が DRIFTED を出す。

### D-2(LOW、plan): チェックボックスが未更新

AC-1〜AC-6 のチェックボックスと、Progress checklist の「Verification artifact created」以降が未チェックのまま。実装は進んでいるので orchestrator が更新する。

### D-3(LOW、plan): AC-2 の文面と調査の記述

AC-2 は「差分が既存の org runtime の節と「Where the values live」の 1 行だけ」と書くが、Slice B で root だけの bullet 3 行(`.claude/rules/ralph/model-routing.md:134-136`)が足された。Progress notes(Slice B)に記録があり、bullet は scaffold に出ないテストを指すので root だけに置くのは正しい。AC の文面を実態に合わせるか、Progress notes の記録で足りるとするかは orchestrator が決める。また「調査で確認したこと」は値を書いている文書として `model-routing.md` だけを挙げ、`.codex/README.md:74`(D-1)を拾っていない。

### D-4(LOW): tech-debt の行の参照先がまだない

`docs/tech-debt/README.md:151` は `docs/plans/archive/2026-10-04-subagent-model-defaults.md` を参照するが、plan は今 `docs/plans/active/` にある。/pr が plan を archive に移すと参照は正しくなる。過去の PR と同じ書き方で、/pr の前の時点だけの前方参照。/pr が中止になった場合は参照が切れる。

### D-5(注記): self-review addendum の cycle 表記

addendum の「cycle 2(既定の cap 2 回のうち 2 回目)」は誤り。plan の Progress notes が訂正していて、`.harness/state/standard-pipeline/cycle-count.json` は `{"cycle": 1}`。report はその時点の記録なので直さない。

## Observational checks

- AC-5 の scaffold: `go run ./cmd/ralph init --yes` の出力は 21 行で exit 0。scaffold の `.claude/agents/` の 5 ファイルは worktree と同一で、`model-routing.md` は template のコピーと同一。
- tier 表と frontmatter の独自の照合: 両側の `## Tier table` の行を `|` で分け、Model 列と、backtick で書かれた 5 つの agent 名を frontmatter と比べた。両側とも implementer、verifier、tester が `opus`、reviewer と doc-maintainer が `sonnet` で一致した。テストのコードは使っていない。

## Coverage gaps

- `tests/test-agent-models.sh` の実行結果(47 / 47、dash での挙動、mutation が実際に落ちること)は確かめていない。/test の担当。
- AC-6 の test の半分(`tests/test-*.sh`、`go test ./...`)は /test の担当。
- Claude Code が新しい frontmatter のモデルで subagent を起動するか(Task のモデル解決)は静的には確かめられない。plan の記録では、この session の agent 定義は main のチェックアウトから読まれる。マージ後に main で reviewer が sonnet、verifier が opus で動くかは未確認。
- D-1 の種類のずれを防ぐ検査はない。いちばん費用が小さいのは、D-1 を (2) の参照で直して README からモデル名をなくすこと。値を残すなら、`tests/test-agent-models.sh` に「`.codex/README.md` の `` the `<x>` tier applies to the Claude Code counterpart (`.claude/agents/<agent>.md`) `` の `<x>` が frontmatter と一致する」検査を足す。
- insight event は追記していない(commit をこの report だけにする指示のため)。

## Verdict

- Verdict: pass。AC-1、AC-2、AC-4、AC-5 は満たし、AC-3 と AC-6 は static で確かめられる範囲を満たす。D-1 は /sync-docs で直してから /pr に進む
- Verified: AC-1、AC-2、AC-4、AC-5、AC-6 の static の半分、static analysis の全項目、変えてはいけない別の仕組みに差分がないこと
- Partially verified: AC-3(テストの内容と件数は読んで確認、実行は /test)、AC-6(test の半分は /test)
- Not verified: 実行時の subagent のモデル解決、`.codex/README.md` の文が直ること(/sync-docs の後に確かめる)
