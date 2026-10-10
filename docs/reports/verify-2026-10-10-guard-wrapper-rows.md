# Verify report: guard-wrapper-rows

- Date: 2026-10-10
- Plan: docs/plans/active/2026-10-10-guard-wrapper-rows.md(`- Approved:` の digest 595308ce450b は `./scripts/plan-visual.sh digest` の値と一致)
- Verifier: verifier subagent (Claude)、pipeline cycle 1
- Scope: `git diff 382c18c8...HEAD`(base 382c18c8、HEAD 6ae86c5f、5 ファイル、+273/-15)。コードの変更は `tests/test-pre-bash-guard.sh` だけで、0eebc7e7 が F 節の包みの行の検査と `run_guard` の切り出し、35f00d33 が FAIL の文言とコメント、fc97d108 と 6ae86c5f がコメントだけ(6ae86c5f はコメントでない変更行が 0 行)。docs は `docs/tech-debt/README.md` の 1 行(bf3241dc)、plan、self-review report、insight event。テストスイートと `run-verify.sh` の test mode は実行していない(AC2 の mutation でテストが FAIL を記録すること、AC3 のテスト全件は /test が見る)。probe は scratch に置いた guard の写しに、コマンドを hook の経路で直接渡した(macOS の /usr/bin/awk、jq あり・なし)
- Evidence: `docs/evidence/verify-2026-10-10-guard-wrapper-rows.log`(gitignore の対象なのでコミットしない。scratchpad の `vf-wraprows/` の `static.sh`・`probe.sh`・`probe2.sh` の出力と、スクリプトそのもの)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: 実行時の `WRAPPER` の名前ごとに 2 形を jq あり・なしで guard に渡し、9 つの名前すべてで PASS。`WRAPPER` が読めないときは FAIL | Met(静的な読みと guard の probe。テストの実行は /test) | 検査は `tests/test-pre-bash-guard.sh:1600-1649`。`wrapper_words` は直前の DATACMD の検査が実行時に 3 つの `.awk` から読んだ `WRAPPER` の名前(`:1566-1576`)。名前と経路の組ごとに `wrapper_row_forms`(`:1614-1617`)の 2 形を `run_guard` で直接動かす(`:1638`、キューは通らない)。jq ありは `real_path`、なしは `minimal_path`(`:1629`、`:1631`)、jq がなければ jq 側だけ SKIP(`:1625-1627`)。どちらかが deny かつ exit 0 なら PASS(`:1640-1643`)、そうでなければ名前と 2 形の判定を出して FAIL(`:1645`)。`WRAPPER` が空なら `:1618-1619` で FAIL。worktree の guard の写し(`cmp` で同じ内容)に 9 名 × 2 形を jq あり・なしで渡すと、8 つは 1 形目が deny/deny で 2 形目が none/none、`timeout` だけ逆だった。deny を出す `emit_deny` は `exit 0`(`.claude/hooks/pre_bash_guard.sh:169-172`)なので、9 名とも PASS の条件を満たす |
| AC2: `WRAPPER` に分岐のない 10 個目の名前(`chrt`)を足すと、その名前を出して FAIL。`cmd_pos` の分岐を 1 つ消す 9 通りも FAIL | Met(guard の写しでの抜き取り 10 通りと空の一覧。テストの中で FAIL が記録されることは /test) | `rules.awk:407` の一覧に `chrt` を足した写し(違いはこの 1 行)では、`chrt` の 2 形が none/none。F と同じ dump の読み方で、この写しから `chrt` を含む 10 語が読める。名前はラベル(`:1623`)と FAIL の文言(`:1645` の `wrapper $w … got: $got_forms`)に出る。`cmd_pos`(`pre_bash_guard_commands.awk:132-140`)の `nm == "<name>"` を 1 つずつ `"<name>__nobranch"` に変えた 9 通り(どれも 1 行の違い)では、その名前の 2 形がどれも none/none になる。`WRAPPER` を空にした写しでは、dump の `W ` の行が 0 になり、`:1618-1619` の FAIL に入る |
| AC3: テスト全件が通る(2,090 件と新しい検査、macOS の awk・mawk・gawk)。guard のファイルは変わっていない | Partially verified | `git diff --stat 382c18c8 -- .claude/ templates/` は空。`tests/test-pre-bash-guard.sh` の写しは `templates/base/` にない。`decide`(`:269-272`)は `run_guard`(`:255-266`)を呼んで結果をファイルに書くだけで、判定の 4 分岐と `rc` の取り方は base の `decide` と同じ。テスト全件の実行は /test の担当で、回していない |
| AC4: `./scripts/run-verify.sh` が rc 0。tech-debt の (h) の閉じの文が、10 個目の包みの行の足し忘れをこの PR で解消したと書く | Partially verified | `./scripts/run-static-verify.sh`(`run-verify.sh` の static mode)は rc 0。test mode を含む `run-verify.sh` は /test。`docs/tech-debt/README.md:163` の Debt 列は、元の「nothing catches it for a tenth wrapper added without a row」を含む 2 文を `~~…~~` で消し、「Closed in test/guard-wrapper-rows (0eebc7e7): …」を続ける。表は 5 列のまま、Related にこの plan がある。ただし閉じの文の 1 句が、テストのヘッダーの F 項目と食い違う(V-2) |

Plan の Scope、Assumptions、Design decisions との照合:

- Scope 1(2 形、jq あり・なし、直接の検査、少なくとも一方が deny、名前と 2 形の判定を出す、`WRAPPER` が読めないとき FAIL): Met。コードは deny に加えて exit 0 も求める(`:1640`)。plan より厳しいが、キューの判定が `rc` を見るのと同じ扱い
- Scope 2(ヘッダーの F 項目と `edge_deny` のコメントの役割分担): Met。書き方の点が 1 つ残る(V-1)
- Scope 3(tech-debt の 1 文): Met。書き方の点は V-2
- Assumptions 1(9 つは 2 形のちょうど一方が deny、分岐のない名前は 2 形とも none): Met(probe)
- Assumptions 2(`decide` と同じ呼び方で guard を直接動かす): Met。`decide` と F の検査が同じ `run_guard` を使う
- Design decisions(9 行を残す): Met。9 行は `:926-934`。名前を `WRAPPER` から消した写し 2 通り(`timeout`、`nice`)で、その名前の D の行(`timeout 5 sh -c …`、`nice -n 5 sh -c …`)が none/none になる。ヘッダーの「The nine rows in D catch a name removed from WRAPPER」と合う

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | rc 0 | `Language scope: full fallback (unclassified:tests/test-pre-bash-guard.sh)`。shellcheck、`sh -n`(hooks 20 本)、guard の awk の parse(root と template)、settings の `jq -e`、Codex hook の 3 段、check-sync、check-pipeline-sync、check-skill-sync(13)、check-template-purity、tech-debt の plan 参照、gofmt、golangci-lint(0 issues)、branch secret scan(382c18c8..6ae86c5f clean)が通った |
| `shellcheck -S warning tests/test-pre-bash-guard.sh` | rc 0、0 件 | |
| `bash -n tests/test-pre-bash-guard.sh` | rc 0 | |
| `./scripts/check-sync.sh` | rc 0 | IDENTICAL 167、DRIFTED 0、ROOT_ONLY 0、`PASS: all files in sync.` |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0 | FAIL の段なし |
| `./scripts/check-skill-sync.sh` | rc 0 | 13 skill |
| `git diff --check 382c18c8...HEAD` | rc 0 | |
| `git diff --stat 382c18c8 -- .claude/ templates/` | 空 | guard のファイルは変わっていない |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-10-guard-wrapper-rows.md` | 595308ce450b | plan の `- Approved:` と一致 |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| テストのヘッダー D 項目(`:59-60`) | Yes | 「nine hand-written rows before sh -c, one for each name WRAPPER had when they were written」。数が書いた時点に結びつくので、10 個目の名前が入っても偽にならない |
| テストのヘッダー F 項目(`:98-105`) | Yes | 2 形、経路ごと、少なくとも一方が deny、D の 9 行は名前の削除を捕まえ、F は分岐の欠けを捕まえる、引数の読み方は見ない(D の手書きの行が見る)、空の一覧で FAIL。コードと probe に合う |
| `run_guard` のコメント(`:250-254`) | Yes | 「deny (the deny JSON with any permissionDecisionReason)」は `:261` のパターンと合う |
| `edge_deny` の 9 行の上のコメント(`:912-925`) | ほぼ Yes | 下の「self-review LOW 6 の確認」を参照。V-1 |
| F の検査のコメント(`:1600-1613`) | Yes | 「timeout takes the next word as its duration, the other eight read the next word as the command」は `cmd_pos` と probe に合う。キューを使わない理由(`run_queue` は F と G のあと、キューの 1 件は期待値が 1 つ)も書いてある |
| `docs/tech-debt/README.md:163` | 一部ずれ | V-2、I-1 |
| `.claude/hooks/pre_bash_guard_rules.awk:399-406` | Yes | 「a D row … pins each wrapper, and its F section reads this list」は今も正しい。I-3 |
| plan の Progress checklist | 一部ずれ | I-2 |

### self-review LOW 6 の確認(6ae86c5f で直したコメント、`:912-925`)

- 2 形の説明: 正しい。コメントの「two fixed forms, W sh -c ... and W 5 sh -c ...」は `wrapper_row_forms`(`:1614-1617`)の 2 つと同じ
- `5` の読まれ方: 正しい。`cmd_pos` では `timeout` だけが `skip_opts(…) + 1` で次の語を飛ばす(`commands.awk:139`)。ほかの 8 つは、`skip_env`、`skip_command`、`skip_opts`、`skip_xargs` のどれでも、`-` で始まらず代入の形でもない `5` の位置を返すので、`5` がコマンド名になる。probe でも、8 つは `W 5 sh -c …` が none/none、`timeout` は `timeout sh -c …` が none/none で `timeout 5 sh -c …` が deny/deny だった
- 「only these rows run each wrapper with arguments of its own (nice -n 5, stdbuf -o0, timeout 5) before sh -c」: 括弧の 3 つについては正しい。ファイルの中で、`nice`、`stdbuf`、`timeout` に引数を付けて `sh -c` の前で動かす行は `:931-933` だけで、`nice -n 5 sudo ls`(`:903`)、`stdbuf -oL sudo ls`(`:905`)、`timeout 5 sudo ls`(`:901`)は `sh -c` でなく `sudo` の前で動かす。シェルの `-c` を含む行を全部 grep して確かめた。ただし、文を広く「引数を付けた包みを `sh -c` の前で動かす行」と読むと、反例が 1 つある(V-1)

### 指摘(docs は /verify では直していない)

- V-1(LOW、新規、判定に関係しない): `edge_deny` のコメント `:924-925` の「only these rows run each wrapper with arguments of its own」は、字面どおりに読むと 2 点で広い。(1) B 節の `:513`、`echo 'sudo ls' | xargs -I{} sh -c {}` も、`xargs` に自分の引数 `-I{}` を付けて `sh -c` の前で動かす。この行は `xargs` の分岐を消した写しでも deny/deny のまま(sentinel が止める)で、分岐の引数の読み方を押さえていない。このため「引数の読み方を `sh -c` の前で押さえるのは D の 9 行だけ」という中身は保たれる。(2)「each wrapper」は 9 行すべてを指すように読めるが、引数を付けているのは 3 行で、残りの 6 行は包みを素で動かす。直すなら「only these rows run nice, stdbuf and timeout with arguments of their own (nice -n 5, stdbuf -o0, timeout 5) before sh -c」。self-review の Follow-ups が LOW 6 について書いたとおり、コメント 1 文の直しなので、次にこのファイルを触る機会でよい
- V-2(LOW、self-review の表の 4、plan の LOW 3 の tech-debt 側。/sync-docs で直す予定のもの): `docs/tech-debt/README.md:163` の閉じの文の「so a new wrapper needs only its name in `WRAPPER` and its branch in `cmd_pos`」は、テストのヘッダーの F 項目 `:104-105`(「It does not check how the branch reads the wrapper's own arguments (a hand-written row in D does)」)と食い違う。テスト側は fc97d108 で直ったので、今は 2 つの docs が別のことを言っている。self-review の勧めた形(引数や時間を取る包みには `edge_deny` の行も要る)で直す
- I-1(情報、self-review の Follow-ups と同じ): 同じ閉じの文の「a tenth name `chrt` and each of the nine branches removed, all caught」の根拠は、今は plan の Progress checklist にしかない。/test の report ができたら、Related に `docs/reports/test-2026-10-10-guard-wrapper-rows.md` を足す。Related の `docs/plans/active/2026-10-10-guard-wrapper-rows.md` は、`/pr` の `scripts/archive-plan.sh` が archive の path へ書き換えるので、手で直す必要はない
- I-2(情報): plan の Progress checklist の「Review artifact created」が未チェックのまま。self-review report は 336f5856 と 65d608b9 でコミット済み。S3b と S3c の行は「この commit」と書き、hash(fc97d108、6ae86c5f)が入っていない(S2 の行には入っている)
- I-3(情報、self-review の Follow-ups と同じ): `pre_bash_guard_rules.awk:404-406` は F 節がこの一覧から行を作ることを書いていない。偽ではなく、この PR は guard のファイルを変えない決まりなので、次に guard を直す機会に足せばよい

## Observational checks

- 9 名と `chrt` の 2 形(20 件)を、worktree の guard の写し、旧版(`tests/fixtures/guard-1c4cea5a/`)、jq あり・なしに通した。旧版は 20 件とも none で、包みの行は新しい guard でしか deny にならない
- mutant の写しは 13 組(`chrt` を足したもの、分岐を消したもの 9 組、`WRAPPER` を空にしたもの、`WRAPPER` から `timeout`・`nice` を消したもの)。どれも写しと worktree の差が 1 行であることを `diff` で確かめてから probe した
- B 節の `:513` の行は、worktree の guard と `xargs` の分岐を消した写しの両方で deny/deny(旧版も deny)。V-1 の根拠

## Coverage gaps

- テストスイートの実行(2,108 件、macOS の awk・mawk・gawk)と `run-verify.sh` の test mode は /test の担当で、回していない
- AC2 は、guard の写しで 2 形が none になることと、検査のコードの読みから、FAIL になると推した。テストのファイルの中で FAIL が記録されることは見ていない
- awk は macOS の /usr/bin/awk だけ。`for (k in WRAPPER)` の並びは awk ごとに違うが、名前ごとの判定は独立なので結果に関係しない
- F の検査は分岐があることだけを見て、分岐が包みの引数を正しく読むことは見ない(テストのヘッダーと self-review の表の 4 のとおり)。10 個目の包みが引数を取るなら、`edge_deny` に手書きの行を足さない限り、その読み方は押さえられない。plan の Non-goals の範囲で、この PR の前からある穴
- マージ後に main のチェックアウトから動く hook の効き目は、マージ前には確かめられない

## Verdict

- Verdict: pass
- Verified: AC1、AC2(guard の写しでの抜き取り 10 通りと空の一覧)。plan の Scope、Assumptions、Design decisions はコードと probe に合う。guard のファイルは変わっていない。静的解析は全部通り、plan の digest も一致した。self-review LOW 6 の直し(2 形、`5` の読まれ方、`sh -c` の前で引数を付けて動かす行)は、括弧の 3 つの包みについて正しい
- Partially verified: AC3(guard のファイルが変わっていないことは確認済み。テスト全件は /test)、AC4(static mode は rc 0、tech-debt の (h) は閉じた。test mode は /test。閉じの文の 1 句に V-2 が残る)
- Not verified: テストの実行、mutation でテストが FAIL を記録すること、mawk と gawk、マージ後の効き目
- pass にした理由: コードの AC はすべて満たし、残りは /test の担当の項目と docs の書き方だけ。指摘は LOW 2 件(V-1 は新規でコメント 1 文、V-2 は self-review で見つかり /sync-docs に回したもの)と情報 3 件で、どれも判定に関係しない
