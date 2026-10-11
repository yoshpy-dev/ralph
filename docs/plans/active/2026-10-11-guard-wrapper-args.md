# guard-wrapper-args

- Status: Approved
- Approved: 2026-10-11 sha256:e69b20224a60
- Owner: Claude Code
- Date: 2026-10-11
- Related request: ユーザーの依頼「残したものを修正して」(2026-10-11)。PR #219(d41604f1)の完了報告で「残したもの」に挙げた 2 件: (1) 包みの行の検査は分岐があることだけを見て、包みの引数の読み方を見ない。引数を取る包みを足すときに手書きの行が要ることは、文書に書いただけで、テストは強制しない (2) `edge_deny` のコメントの 1 文の字面が広い(verify V-1)
- Related issue: N/A
- Type: test
- Branch: test/guard-wrapper-args

## Objective

guard の包み(`pre_bash_guard_rules.awk` の BEGIN の `WRAPPER`)には、`cmd_pos` の分岐と、`tests/test-pre-bash-guard.sh` の `edge_deny` の手書きの行(`nice -n 5 sh -c 'git commit -n -m x'` の形)が要る。手書きの行だけが、分岐が包みの引数を読む読み方を押さえる。PR #219 の検査は一覧の名前ごとに決まった 2 形を流すので、分岐があることは確かめるが、引数の読み方は見ない。新しい包みを足して手書きの行を足し忘れても、テストは通る。

手書きの行を専用の配列に移し、テストが実行時の `WRAPPER` の名前と手書きの行の 1 語目を突き合わせて、行のない名前と、一覧にない名前の行を FAIL にする。あわせて、`edge_deny` のコメントの字面が広い 1 文(V-1)を、確かめた事実に合わせて直す。判定は 1 つも変えない。

さらに、今ある包みの引数の読み方を押さえる行を `wrapper_deny` に足す(承認後の self-review の LOW 2 で範囲を広げた。Progress checklist を参照)。今の 9 行のうち包みの引数を `sh -c` の前に置くのは `nice -n 5`、`stdbuf -o0`、`timeout 5` の 3 行だけで、`env -u`、`exec -a`、`time -o`、`xargs -I{}`、`command -p` などの読み方は、`git commit -n` の形のどの行も押さえない。`time` と `exec` の `skip_opts` の旗を空にした写しで、テストは緑のままだった。

## Scope

- `tests/test-pre-bash-guard.sh`
  - `edge_deny` の包みの手書きの 9 行(926〜934 行)を、新しい配列 `wrapper_deny` に移す。D 節で `edge_deny` と同じ `check_modes D deny absent` で流す(ラベルは節とコマンドだけで作るので、ラベルも件数も変わらない)
  - F 節の PR #219 の検査のあとに、新しい検査を足す。実行時に読んだ `WRAPPER` の名前の集合と、`wrapper_deny` の各行の 1 語目の集合を比べ、どちらかにしかない名前があれば FAIL にする。行のない名前には「`wrapper_deny` に W の行を足す。W が自分の引数を取るなら、その引数を `sh -c` の前に置く」と案内し、一覧にない 1 語目には「名前が `WRAPPER` から消えた」と出す。あわせて、`wrapper_deny` の各行が ` sh -c 'git commit -n -m x'` で終わることを確かめ、行ごとに、その末尾を ` sh -c 'git commit -m x'` に替えた対照の形を jq あり・なしで guard に渡して、none になることを確かめる(Design decisions を参照)。PASS の文言にも「行があることだけを確かめた。行が包みの引数を押さえているかは見ていない」と出す。`WRAPPER` が読めないときは FAIL
  - コメントの直し。`edge_deny` の 9 行の上のコメント(912〜925 行)の V-1 の 1 文を、次の事実に合わせる: 包みに自分の引数を付ける行はほかにもある(D の `nice -n 5 sudo ls`、B の `echo 'sudo ls' | xargs -I{} sh -c {}` など)が、それらは旧版から引き継いだ 4 つの文字列照合(sentinel)に掛かるので、分岐が引数を読み違えても deny のまま。sentinel に掛からない `git commit -n` の手書きの行だけが、そのとき落ちる。ほかに、「`edge_deny` の 9 行」を前提に書いた箇所(ヘッダーの D 項目 59〜60 行、F 項目 98〜105 行、F 節の PR #219 の検査の上のコメント 1600〜1613 行)を、`wrapper_deny` と新しい検査に合わせて直す
  - 今ある包みの引数の行を `wrapper_deny` に足す。基準は「`cmd_pos` の分岐ごとに、値を取る引数の書き方ごとに 1 行以上」で、man にある形だけを使う。短い旗と次の語(`exec -a n`、`time -o f`、`time -f %e`、`timeout -s KILL 5s`、`timeout -k 1 5`)、長いオプションと次の語(`time --format %e`、`nice --adjustment 5`、`stdbuf --output L`)、長いオプションと `=` の 1 語(`time --format=%e`)、`--` の終端(`nohup --`)、専用の関数で読む包み(`env -u HOME`、`env -i PATH=/bin`、`env -C /tmp`、`command -p`、`xargs -I{}`、`xargs -n 1`)の 16 行。どれも ` sh -c 'git commit -n -m x'` で終わる。足した行にも、新しい検査の末尾の形と対照(none)がそのまま掛かる
  - self-review の LOW 1・3・4・5・6 の直し(コメント、FAIL の文言、変数の名前。判定とテストの結果は変えない)
- `docs/tech-debt/README.md` のテストの穴の行(163 行)の (h) の閉じの文の「so a wrapper with arguments of its own still needs a row there」を、行があることはこの PR の検査が強制し、行が引数を押さえているかは人の判断に残る、と書き換える。今ある包みについては、足した行が押さえる引数の書き方と、押さえない形(分岐の表にない旗)を書く(self-review の LOW 2)

## Non-goals

- guard のファイルの変更(判定の変更を含む)
- 手書きの行が包みの引数を押さえているかを、テストが判断すること。テストは実際のツールのオプションを知らない。引数なしで `W sh -c '…'` と書いた行でも、集合は一致する
- `cmd_pos` の引数の表を guard に持たせて、テストがそこから行を作ること(consult で退けた案 B。guard の構造を変えるのに、実際のツールとずれた表の誤りは防げない)
- PR #219 の検査(決まった 2 形)を消すこと。分岐の足し忘れを、理由つきで直接出すのはこの検査
- tech-debt の Debt (b) の `$'\echo' 'sudo ls'` と `$"echo" 'sudo ls'` の形。直前の報告の「残したもの」には入っていない。guard の判定を変える別の変更になる

## Assumptions

- テストのラベルは `check` の中で節とコマンドから作り(302 行)、配列の名前を含まない。D 節は `collect_corpus=no` のあとなので、G 節の corpus にも入らない(consult が確かめた)。9 行を別の配列に移しても、G 節と件数は変わらない
- 9 行の 1 語目は、`$'…'` で書いた行でも、配列の要素では名前そのもの(`env`、`nice` など)になる。`${row%% *}` で取れる
- main 577ebeaf の写しで `cmd_pos` の `xargs` の分岐を消すと、`xargs sh -c 'git commit -n -m x'` と `xargs -I{} sh -c 'git commit -n -m x'` は none になり、`echo 'sudo ls' | xargs -I{} sh -c {}`、`echo x | xargs sh -c 'git reset --hard'`、`xargs -I{} sudo ls {}`、`xargs -n 1 sudo ls`、`xargs -I{} flock /tmp/l git push -f` は jq あり・なしとも deny のままだった(2026-10-11 に確かめた)。V-1 の直しはこの結果に沿って書く
- 同じ main の guard で、9 行の末尾を ` sh -c 'git commit -m x'` に替えた対照の形は、9 つとも jq あり・なしで none だった。包みの引数に sentinel の語を入れた `time -o 'sudo log' sh -c 'git commit -m x'` は deny だった(2026-10-11 に確かめた)
- HEAD 76f2f826 の guard(main と同じ)で、足す 16 行は jq あり・なしとも deny、その対照(`-n` を外した形)は 16 行とも none だった(2026-10-11 に確かめた)

## Affected areas

- `tests/test-pre-bash-guard.sh`
- `docs/tech-debt/README.md`

## Visual review

None (テストの行を別の配列に移して足し、検査を 1 つ足し、コメントと tech-debt の文を直すだけで、guard の部品やデータの形を変えない)

## Design decisions

- **集合の一致を両方向で確かめる**(consult の判定)。`WRAPPER` にあって行のない名前は、手書きの行の足し忘れ。行にあって `WRAPPER` にない名前は、名前が一覧から消えたこと。後者は今も行の deny が none になって捕まるが、新しい検査は理由を直接出す。足す行は数行
- **行の末尾を決め、末尾を無害にした対照の形が none になることも確かめる**(末尾は Codex の plan advisory の前から、対照の形はその指摘で足した)。sentinel に掛かる中身(sudo、force push など)の行は、分岐が引数を読み違えても deny のままなので、何も押さえない(Assumptions の確かめ)。末尾を ` sh -c 'git commit -n -m x'` に決めると、中のコマンドに sentinel の語を書いた行は FAIL になる。ただ、包みの引数の側に sentinel の語を書いた行(`time -o 'sudo log' sh -c …`)は末尾だけでは捕まらない。対照の形(末尾を ` sh -c 'git commit -m x'` に替えたもの)が none であれば、行の deny は中の `git commit -n` を読んだことからしか来ない
- **今ある包みの引数の行は、分岐ごと・値を取る引数の書き方ごとに足す**(self-review の LOW 2 を受けた consult の判定)。行を足すと、新しい検査の末尾の形と対照もそのまま掛かる。旗の名前を全部並べるのではなく、分岐が値を読む書き方(短い旗と次の語、長いオプションと次の語、`=` の 1 語、`--`、専用の関数)をそれぞれ 1 行以上で押さえる。man にない形(guard の都合に合わせた形)は足さない
- **強制できるのは行の存在までと明記する**。行が包みの引数を本当に押さえているかは、包みのツールを知る人の判断に残る。FAIL の文言とコメントでそれを促し、報告でもそう書く
- Critical forks: None(テストだけの変更で、戻すのも検査と配列を元に戻すだけ。案 A と案 B の選択は consult の判定で決めた)

## Acceptance criteria

- [ ] AC1: 包みの手書きの 9 行が `wrapper_deny` にあり、D 節で今と同じラベルの deny の行として流れる。Scope の 16 行も `wrapper_deny` にあり、jq あり・なしとも deny になる。新しい検査が、実行時の `WRAPPER` の 9 つの名前と 25 行の 1 語目が一致し、25 行の末尾が決まった形で、25 行の対照の形が jq あり・なしとも none であることで PASS する。`WRAPPER` が読めないときは FAIL になる
- [ ] AC2: 写しで次の 5 つの変更をすると、どれも新しい検査がその名前か行を出して FAIL になる。(a) `WRAPPER` に分岐つきの 10 個目の名前を足す(たとえば `chrt` と、`cmd_pos` に `skip_opts` の分岐。PR #219 の検査は通る) (b) `wrapper_deny` から 1 行消す (c) `WRAPPER` から名前を 1 つ消す (d) `wrapper_deny` の 1 行の中のコマンドを sentinel に掛かる形にする (e) `wrapper_deny` の 1 行の包みの引数に sentinel の語を入れ、末尾は保つ(たとえば `time -o 'sudo log' sh -c 'git commit -n -m x'`)。さらに、(f) `cmd_pos` の `time` と `exec` の分岐の値を取る旗を空にした写しで、足した行(`time -o f`、`exec -a n` など)が FAIL になる(16 行を足す前は、この写しでテストは緑だった)
- [ ] AC3: `bash tests/test-pre-bash-guard.sh` が全部通る(macOS の awk、mawk、gawk)。件数は 2,108 に新しい検査の件数と足した行の件数を足したもの(D 節は 16 行×2 経路、F 節は 1 件と 25 行×2 経路の対照)。A〜C と E の件数は変わらない。guard のファイルは変わっていない(`git diff` が空)
- [ ] AC4: `./scripts/run-verify.sh` が rc 0。コメントの V-1 の文が、sentinel に掛かる行は分岐の読み違えで落ちない、と書く。「`edge_deny` の 9 行」を前提にしたコメントが残っていない。tech-debt の (h) の閉じの文が、行があることはテストが強制し、行が引数を押さえているかは人の判断に残る、と書き、足した行が押さえる引数の書き方を書く。self-review の LOW 1〜6 が直っていることを、reviewer が verify の前に確かめる

## Implementation outline

1. S1(implementer): 9 行を `wrapper_deny` に移す、新しい検査、コメントの直し、AC2 の mutation の確かめ。AC1〜AC3
2. S2(inline、docs だけ): tech-debt の閉じの文。AC4
3. S3(implementer): 16 行を足し、self-review の LOW 1・3・4・5・6 を直す(行の追加とコメント・文言・名前の直しで 2 commit)。AC1〜AC3、AC2 の (f)
4. S4(inline、docs だけ): tech-debt の閉じの文に、足した行が押さえる書き方と押さえない形を書く(LOW 2)。AC4
5. reviewer に S3・S4 を見せてから verify に進む

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`shellcheck -S warning tests/test-pre-bash-guard.sh`、`./scripts/check-sync.sh`
- Spec compliance criteria to confirm: AC1〜AC4
- Documentation drift to check: テストのヘッダーの D・F 項目、`wrapper_deny` の上のコメント、F 節の 2 つの包みの検査の上のコメント、tech-debt のテストの穴の行
- Evidence to capture: AC2 の mutation の結果

## Test plan

- Unit tests: 新しい検査(F 節)
- Integration tests: `./scripts/run-test.sh`、`./scripts/run-verify.sh`
- Regression tests: 今の 2,108 件。mawk と gawk でもテストを回す。base のテストのファイルを HEAD の guard で回して、節ごとの件数を比べる
- Edge cases: `WRAPPER` が空、分岐はあるが行のない名前、行はあるが一覧にない名前、sentinel に掛かる中身の行、包みの引数に sentinel の語を入れた行
- Evidence to capture: AC2 の 6 つの mutation で FAIL になる名前と文言

## Risks and mitigations

- 行の存在しか強制できないので、引数を取る包みに引数なしの行を書いても通る。FAIL の文言とコメントで促し、PASS の文言にも見ていないことを出す
- 将来の包みで、対照の形(中が `git commit -m x`)が none にならないもの(包みの名前や引数そのものが規則に掛かるもの)があると、正しく足した包みでも FAIL になる。FAIL の文言に行と対照の判定を出し、その行は包みの引数の読み方を押さえられない(deny が中のコマンドから来ない)と案内する
- 「`edge_deny` の 9 行」を前提にした文言が 5 か所にあり、直し漏れが起きやすい。AC4 で `grep` して残っていないことを確かめる
- 検査の文字列は `git commit -n` を含むので、手元の Bash の行に書くと別の hook に止められる。テストのファイルと scratch のファイルに書き、Bash の行には書かない

## Rollout or rollback notes

- テストと docs だけの変更なので、guard の動きは変わらない
- ロールバックはこの PR の revert

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed
- [x] Plan approved
  - 2026-10-11: ユーザーが承認ゲートで Approve。承認の前に、consult(consult-plan-wrapperargs)の判定(案 A、両方向の一致、V-1 も同じ PR、`$'\echo'` は範囲外と明記)を反映した。Codex の plan advisory の指摘 1 件(MEDIUM: 包みの引数に sentinel の語を入れた行は末尾の検査をすり抜ける)は、ユーザーの選択「plan を直す」で、対照の形の検査と AC2 の (e) を足した。前提(`xargs` の分岐を消した写しでの判定、9 行の対照が none、`time -o 'sudo log' …` の対照が deny)は main 577ebeaf の guard で確かめた。図解ページは描いていない(Visual review を参照)
- [x] Branch created
- [x] Implementation started
  - S1 完了(78616500、implementer/opus): 包みの手書きの 9 行を `wrapper_deny` に移し(D 節で同じ引数の `check_modes` で流す)、F 節の PR #219 の検査のあとに新しい検査(集合の両方向の一致と末尾の形で 1 件、行ごとの対照で 9×2=18 件)を足した。V-1 のコメントと、「`edge_deny` の 9 行」を前提にしたヘッダーの D・F 項目、PR #219 の検査のコメントを直した。テストは 2,127/0(macOS の awk、mawk 1.3.4、gawk 5.2.1)、節ごとの件数は F だけが 68 → 87 で、A〜E のラベルは HEAD と一致。shellcheck は指摘なし、run-verify は rc 0。AC2 の mutation (a)〜(e) はどれも新しい検査が FAIL した(写しの木、macOS の awk)。(a) chrt を一覧と分岐に足す: 新しい検査だけが FAIL、PR #219 の chrt の行は PASS (b) nohup の行を消す: FAIL 1 (c) 一覧から nohup を消す: 新しい検査と D の nohup の行 2 件 (d) nohup の行を force push に: 末尾の検査と対照 2 件 (e) time の行を `time -o 'sudo log' …` に: 対照 2 件が deny。`WRAPPER` を空にした写しでも FAIL。handoff からのずれ: V-1 の例に `xargs -I{} flock /tmp/l git push -f` を足した(括弧の「force push」に合う例がなかったため)。末尾の形でない行の対照は jq の SKIP より先に FAIL にした
  - implementer が範囲外で気づいたこと: `pre_bash_guard_rules.awk` の `WRAPPER` の上のコメント「a D row of tests/test-pre-bash-guard.sh pins each wrapper」は、`wrapper_deny` も D 節で流れるので今も正しい。guard のファイルは Non-goals なので直さない
  - S2 完了(この commit、inline、docs だけ): tech-debt のテストの穴の行の (h) の閉じの文で、引数を取る包みに行が要ることをこの PR の検査が強制し、行が引数を押さえているかは人の判断に残る、と書き、Related にこの plan を足した。表の中のパイプは `\|` にした(列の数が base と同じ 5 であることを確かめた)。run-verify は rc 0
  - self-review(a57b6758、reviewer/sonnet): Merge yes、LOW 6。LOW 2(今の 9 行のうち包みの引数を `sh -c` の前に置くのは 3 行だけで、`env -u`、`exec -a`、`time -o`、`xargs -I{}`、`command -p` の読み方はどの行も押さえない。`time` と `exec` の旗を空にした写しでテストは緑)を受けて consult(consult-plan-wrapperargs、2 往復目)に聞き、判定は「範囲を広げる。今ある包みの引数の行を、分岐ごと・値を取る書き方ごとに足し、plan を直して承認を取り直す」。足す 16 行は HEAD の guard で deny、対照は none と確かめた
  - 2026-10-11: plan の Objective・Scope・Assumptions・Design decisions・AC1〜AC4・Implementation outline を直し(16 行の追加、LOW の直し、AC2 の (f))、ユーザーが承認ゲートで Approve(digest e69b20224a60)。順序は S3(行の追加、LOW 1・3・4・5・6)→ S4(tech-debt、LOW 2)→ reviewer の再確認 → verify
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
