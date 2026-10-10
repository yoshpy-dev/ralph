# guard-wrapper-rows

- Status: Approved
- Approved: 2026-10-10 sha256:595308ce450b
- Owner: Claude Code
- Date: 2026-10-10
- Related request: ユーザーの依頼「残したものを修正して」(2026-10-10)。PR #217(19b06925)で残したもの: 新しい包みを足すときに、`edge_deny` の包みの行を足し忘れても、テストで捕まらない
- Related issue: N/A
- Type: test
- Branch: test/guard-wrapper-rows

## Objective

PR #217 で、guard の包みの名前は `pre_bash_guard_rules.awk` の BEGIN の一覧 `WRAPPER` になった。包みには、`WRAPPER` の名前、`cmd_pos` の分岐、`tests/test-pre-bash-guard.sh` の `edge_deny` の行(`env sh -c 'git commit -n -m x'` の形)の 3 つが要る。今の 9 つの包みは 9 行で固定しているが、10 個目の包みを名前だけ足して分岐を足さないと、`cmd_pos` は名前の位置を返し、後ろの `sh -c` を読まないので、そこに書いた禁止の形を見逃す。このずれは、行を足し忘れるとテストで捕まらない。

テストが実行時に読んだ `WRAPPER` の名前ごとに包みの行を作って確かめ、包みを足したときに行を足し忘れても捕まるようにする。判定は 1 つも変えない。

## Scope

- `tests/test-pre-bash-guard.sh` の F 節の、DATACMD の不変条件の検査のあと: 実行時に読んだ `WRAPPER` の名前 W ごとに、`W sh -c 'git commit -n -m x'` と `W 5 sh -c 'git commit -n -m x'` の 2 形を、guard に jq あり・なしで渡す(直接の検査。キューは使わない)。名前ごとに、jq あり・なしのそれぞれで 2 形の少なくとも一方が deny なら PASS、どちらも deny でなければ FAIL にし、名前と 2 形の判定を出す。`WRAPPER` が読めないときは、この検査も FAIL にする
- 同じファイルのヘッダーの F 項目と、`edge_deny` の包みの 9 行の上のコメントを、新しい検査と役割分担に合わせて直す(9 行は名前が一覧から消えたことを捕まえ、新しい検査は一覧の名前に分岐があることを確かめる)
- `docs/tech-debt/README.md` のテストの穴の行の (h) の閉じの文の「nothing catches it for a tenth wrapper added without a row」を、この PR で解消した、と書き換える

## Non-goals

- guard のファイルの変更(判定の変更を含む)
- 今の 9 行を消すこと(名前が `WRAPPER` から消えたことを捕まえるのは、この 9 行だけ)
- `cmd_pos` の分岐はあるのに名前が `WRAPPER` にない形を捕まえること。その分岐には入れず、その包みは飛ばされない(包みとして足す前と同じ)ので、今より悪くはならない
- `$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'` の形(tech-debt の Debt (b))

## Assumptions

- 9 つの包みのうち、`timeout` は次の語を時間として飛ばすので `timeout sh -c …` では `sh` を読まず、`timeout 5 sh -c …` で deny になる。ほかの 8 つは `W sh -c …` で deny になる(`nice`、`stdbuf` は、オプションがなければ次の語をコマンドとして読む)。分岐が `cmd_pos` にない名前は、2 形とも包みとして飛ばされず none になる(PR #217 の test の mutation で、分岐を消すと包みの行が none になった)
- F 節には、実行時の DATACMD と `WRAPPER` を読む検査がすでにあり、読んだ名前は `wrapper_words` にある。テストの guard の実行はどれもキュー(`enqueue` と `decide`)を通していて、直接動かす前例はない。新しい検査は `decide` と同じ呼び方(`PATH=… hook < payload` と、出力から deny・none を見分ける分岐)で guard を直接動かす。キューは F・G 節のあとに走るので、2 形の「どちらか」をそこで評価するより短い
- jq なしの経路は `minimal_path`、jq ありは `real_path` で動かす(テストの既存の書き方)

## Affected areas

- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (テストに検査を 1 つ足し、tech-debt の 1 文を直すだけで、guard の部品やデータの形を変えない)

## Design decisions

- **包みの行を `WRAPPER` から作る**。行を手で足す今の形では、名前を一覧に足して行を足し忘れると捕まらない。テストが一覧を読んで行を作れば、一覧に足した名前はすぐ確かめられる
- **2 形のどちらかが deny なら通す**。包みごとに引数の取り方が違う(`timeout` は時間を取る)。2 形で 9 つすべてを覆えることを確かめる。将来、2 形のどちらでも読めない引数の取り方の包みを足したときは、この検査が FAIL になるので、テストに形を足す(黙っては通らない)
- **今の 9 行は残す**。名前を `WRAPPER` から消す変更は、一覧から作る検査には見えない。9 行がそれを捕まえる
- Critical forks: None(テストだけの変更で、戻すのも検査を消すだけ)

## Acceptance criteria

- [ ] AC1: 新しい検査が、実行時の `WRAPPER` の名前ごとに 2 形を jq あり・なしで guard に渡し、9 つの名前すべてで PASS する。`WRAPPER` が読めないときは FAIL になる
- [ ] AC2: 写しで `WRAPPER` に 10 個目の名前(分岐のない名前、たとえば `chrt`)を足すと、新しい検査がその名前を出して FAIL になる。写しで `cmd_pos` の分岐を 1 つ消すと(9 通り)、どれも新しい検査が FAIL になる
- [ ] AC3: `bash tests/test-pre-bash-guard.sh` が、今の 2,090 件と新しい検査を含めて全部通る(macOS の awk、mawk、gawk)。guard のファイルは変わっていない(`git diff` が空)
- [ ] AC4: `./scripts/run-verify.sh` が rc 0。`docs/tech-debt/README.md` の (h) の閉じの文が、10 個目の包みの行の足し忘れをこの PR で解消した、と書く

## Implementation outline

1. S1(implementer): 新しい検査、コメントの直し、mutation の確かめ。AC1〜AC3
2. S2(inline、docs だけ): tech-debt の 1 文。AC4

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning tests/test-pre-bash-guard.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC4
- Documentation drift to check: テストのヘッダーの F 項目、`edge_deny` の 9 行の上のコメント、tech-debt のテストの穴の行
- Evidence to capture: AC2 の mutation の結果

## Test plan

- Unit tests: 新しい検査(F 節)
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: 今の 2,090 件。mawk と gawk でもテストを回す
- Edge cases: `WRAPPER` が空、分岐のない名前、`timeout` のように次の語を飛ばす包み
- Evidence to capture: AC2 の mutation で FAIL になる名前と判定

## Risks and mitigations

- 新しい検査の 2 形が、将来の包みの引数の取り方に合わないと、正しく足した包みでも FAIL になる。FAIL の文言に名前と 2 形の判定を出し、テストに形を足せば直る、と書く
- 検査の文字列は `git commit -n` を含むので、手元の Bash の行に書くと別の hook に止められる。テストのファイルに書き、Bash の行には書かない

## Rollout or rollback notes

- テストと docs だけの変更なので、guard の動きは変わらない
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-10: ユーザーが承認ゲートで Approve。承認の前に、consult(consult-plan-wraprows)の指摘(guard を直接動かす前例はない、`decide` と同じ呼び方で動かす)を Assumptions に反映した。Codex の plan advisory は指摘なし。前提(9 つの包みは 2 形のちょうど一方が deny、一覧にない `chrt` は 2 形とも none)は main 382c18c8 の guard で確かめた。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
