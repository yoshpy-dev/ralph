# Self-review report: org-ssh-alias-host

- Date: 2026-11-19(2 回目。1 回目 `15066e7c` の LOW 7 件のうち 4 件を直した `55cdb7c3` のあと。`cycle-count.json` は `cycle: 1` のままで、パイプラインの再走ではなく、`/verify` の前の直しの確かめ)
- Plan: `docs/plans/active/2026-11-19-org-ssh-alias-host.md`
- Branch: `fix/org-ssh-alias-host`、HEAD `864e79a5`(計画だけの commit)、コードの最後は `55cdb7c3`、base は `origin/main` の `a8c5e289`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff origin/main...HEAD` の全体(27 ファイル、+1982 −183)の diff の質だけ。重点は `55cdb7c3`(`mergeSSHHostName`、`mergeSSHResolveRuns`、`resolveMergeAPIHost` の分岐、doctor の `readOrigin`、merge と cleanup の `--help`、その test)。テストと静的解析は走らせていない
- 番号: 前回の L1〜L7 の番号は変えない。Go の test のコメント 5 か所が、この report の `L1`・`L2`・`L3` を名指しで引いているので、新しい指摘は L8 から振る

## Evidence reviewed

- `55cdb7c3` の全 diff(コード 4 ファイルと test 4 ファイル)。`mergeorigin.go` は `ReadMergeOrigin` から `mergeSSHAliasHint` までを現在の HEAD で通して読んだ(`resolveMergeAPIHost`、`mergeSSHResolveRuns`、`mergeSSHHostName`、`mergeOriginIsSSH`)
- 呼び手: `mergeguarantee.go` の `readOrigin`・`newMergeGuaranteeRun`・`stopped`、`mergeorder.go` の `mergeOriginFromURL`(host の出どころ: `url.Hostname()` はユーザー情報・port・`[]` を除く、scp 形は最初の `:` の前から最後の `@` の後)、`internal/cli/org_cleanup.go` の `cleanupReadOrigin` と取り消しの文
- `ssh` を打つ場所の全数: `git grep '"ssh"'` で非 test の呼び出しは `mergeorigin.go:190` の 1 か所だけ。`resolveMergeAPIHost` の呼び手は `ReadMergeOrigin`(`:140`)と doctor の `readOrigin`(`mergeguarantee.go:311`)の 2 つ。L1 の検査はこの 1 か所に掛かる
- test(読んだだけ): `mergeorigin_test.go` の表と `TestMergeSSHHostName`、`mergeguarantee_alias_test.go` の `mgDoneOnceHostRead` と新しい 2 つのテスト、`org_merge_test.go`・`org_cleanup_steps_test.go` の `Help` の固定
- help と comment の掃き出し: `internal/cli/*.go` の `get-url origin`・`ssh -G`、`org_regate.go` の ci の段落、`doctor.go` の subprocess の一覧のコメント
- 文書: 計画の D6・Risks・Progress、README の doctor の行と `ralph org merge` の段落(ssh -G の文)、tech-debt の 2 行(RESOLVED の註と director skill の行)を `55cdb7c3` の前後で見比べた(`55cdb7c3` は register も README も触っていない)
- 探したもの: 直しが新しく生んだ「走らない分岐」の理由の文、`%q` と `%s` が混ざる所、古い規則(空白・制御文字・`/` だけを断る)を言う所、test の `map[bool]int` の複製

## 前回の指摘の扱い

| 前回 | 結果 | 根拠 |
| --- | --- | --- |
| L1(入力の host の検査が出力より緩い) | 解決 | 入力と出力が同じ述語 `mergeSSHHostName` を通る(`mergeorigin.go:179` と `:210`、定義 `:235`)。許可リストは空でなく、`-` で始まらず、全バイトが `A-Z a-z 0-9 . _ : -`。`$(id)`・`a;touch`・空白・`\|`・バッククォート・引用符・制御文字の host は ssh を打たず、origin の host のまま進み、理由が付く(表のテストの 8 行)。`ssh` の呼び出しは `mergeorigin.go:190` の 1 か所で、その前に必ずこの述語を通る。`ReadMergeOrigin` が `-` で始まる host を先に断る(`:137`)ので、`-` の枝は doctor の道だけで届く |
| L2(merge と cleanup の `--help` が古い) | 解決。ただし同じ漏れが別の場所に残る(L8) | `org_merge.go:58-67`・`org_cleanup.go:70-80` に、ssh の origin の host は `ssh -G <host>` の答えで、`Match exec` の命令が走ると書いた。どちらも `Help` のテストが文を固定する(newline を畳んだ連続した文) |
| L3(doctor が「ssh -G が走ったか」を条件から写す) | 解決 | `mergeSSHResolveRuns(originURL, host)` を 1 つ置き、解く関数(`:175`)と doctor(`mergeguarantee.go:312`)が同じ関数で決める。推奨は 3 つ目の戻り値だったが、述語の共有でも「2 つが食い違えない」は満たす。test は discriminating: `mgDoneOnceHostRead` は `c.Host` が入ったあとに親の ctx を終わらせ、`a;touch` の行は旧い doctor の条件(`mergeOriginIsSSH && !HasPrefix("-")`)なら ssh を名指しして落ちる。解く関数の表にも `mergeSSHResolveRuns` と `wantSSH` の一致を見る検査が入った |
| L5(最後の `switch` の 3 つ目の条件が重い) | 解決 | 3 行のクロージャと `\|\|` が `!mergeSSHHostName(resolved)` の 1 つに畳まれ、`unicode` と `utf8` の import も消えた。L1 の入力検査と同じ関数 |
| L4(`resolveMergeAPIHost` の形) | 残した。許容できる | 動きが変わらない。計画の slice を 1 つ増やす割に、直しは 8 呼び手と test の約 60 呼び出しに及ぶ。ただし `mergeSSHResolveRuns(originURL, host string)` が同じ「型の同じ引数が 2 つ並ぶ」形を 1 つ足した(3 呼び手: 解く関数、doctor、test)。同じ系統なので、L4 の払う時の条件に含める |
| L6(偽の ssh のプロセスグループの確かめが 3 か所) | 残した。許容できる | test だけで、本番の挙動に関わらない。3 か所のままで、`55cdb7c3` は増やしていない |
| L7(test の表の読みにくさ) | 残した。許容できる。ただし少し悪化 | (a) 位置指定 7 欄の表に約 15 行が足され、`notRun(...)` の行は読みやすいが欄の取り違えは同じ。(b) `map[bool]int{true: 1, false: 0}[tc.sshRuns]` が `mergeguarantee_alias_test.go:166` と、`55cdb7c3` が足した `:359` の 2 か所になった。どちらも test だけ |

## Findings

CRITICAL、HIGH、MEDIUM はない。LOW は、残した 3 件(L4、L6、L7)と新しい 3 件(L8、L9、L10)の 6 件。

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | L8: L2 の直しは `ssh -G` と `Match exec` を、ssh を打つ 4 つの場所のうち 2 つの `--help` にだけ書いた。(a) `ralph org regate` の ci の段落は、origin の URL・gh・PR を確かめると言うだけで、ssh の origin では `ssh -G` が走り `Match exec` の命令も走ることを書かない。regate(ci)は merge・cleanup と同じ `ReadMergeOrigin` を通る。(b) `doctor.go` の `runDoctorOpts` のコメントは、検査が起こす subprocess を数え上げる(「`git remote get-url origin` and up to four `gh api` calls under one 20 s deadline」)が、新しい `ssh -G`(専用の 5 秒、自分のプロセスグループ)が抜けている。(c) 2 つの help の文は「host は `ssh -G` が出す hostname」と言い切り、解けなかったとき(失敗、5 秒の期限、普通の host 名でない)に origin の host のまま進むことを書かない(README は書いている)。どれも `git diff` に入っていない。1 回目の L2 の推奨(「5 秒の期限」)も help に入っていない | `internal/cli/org_regate.go:164-166`、`internal/cli/doctor.go:80-85`、`internal/cli/org_merge.go:63-67`、`internal/cli/org_cleanup.go:74-78`。1 回目の report は L2 を merge と cleanup だけに限って挙げたので、regate と doctor は 1 回目の見落とし | (a) regate の ci の段落に、merge と同じ 1 文を足す。(b) `doctor.go` のコメントに `ssh -G`(5 秒、自分のプロセスグループ、origin が ssh のときだけ)を足す。(c) 余裕があれば、merge と cleanup の文に「解けなければ origin の host のまま」を 4〜6 語足す。help を固定する test は 2 つあるので、文を変えたら一緒に直す |
| LOW | security | L9: `55cdb7c3` が「走らない」理由の文の host を `%s` から `%q` にした(`mergeorigin.go:177`、`:179`)のは正しい。host が引用符や制御文字を持ちうるようになったため。ところが、その文はすぐ次の `mergeSSHAliasHint` の中で、同じ host を `%s` で 2 回(`apiHost`、`originHost`)そのまま出す。同じ Detail の 1 文の中で、1 つ目は escape され、2 つ目と 3 つ目は生のまま、制御文字(ESC など)が端末へ届く。scp 形の origin には制御文字が入れられる(`url.Parse` は拒むが、`git@a\x1bb:o/r` は `mergeOriginFromURL` を通る。新しい表の `a\x1bb` の行が示す)。host は利用者自身の git config なので、危険は低い。`unresolvedWhy` を作る他の文(`printed the hostname %q`)は `%q` か、`mergeGuaranteeErrText` を通る | `mergeorigin.go:271-273`(`%s` が 2 つ)、`mergeorigin.go:177-179`、test は `a;touch` の 1 行だけで生の出力を固定し(`mergeguarantee_alias_test.go` の「a host that is no plain host name runs no ssh」)、制御文字の行は解く関数の表(clause だけ)にある | hint の `apiHost` と `originHost` も `%q` にする(期待文の 1 行を `%q` の形に直す)。または、host は利用者の設定なので生で構わないと決め、`%q` を `%s` に戻して 2 か所をそろえる。どちらでもよいが、同じ 1 文の中で混ぜない |
| LOW | readability | L10: 述語を強くしたが、出す文を直していない。`mergeorigin.go:211` の「`ssh -G <host> printed the hostname "<x>", which gh cannot take as a host`」は、`55cdb7c3` で `;`・`$`・引用符・ASCII でない文字・`%`(IPv6 の zone)・`[]` にも出るようになった。gh はこれらの文字を断らない(同じ関数の doc `:166` が「gh itself does not check the hostname's characters」と書く)。断っているのは ralph で、IDN の `HostName` や `fe80::1%en0` を書いた利用者は、gh が受け付けないという誤った理由を読む。入力側の文(`:179`)は「holds characters a host name does not」と、断る者を言わない正確な形。README の doctor の行にも「prints no hostname that `gh` can take」がある | `mergeorigin.go:211` と `:166`、`mergeorigin_test.go:93`(期待文の helper `notAHost`、新しい 4 行を含む) | 文を「which is not a plain host name (ASCII letters, digits, `.`, `_`, `:`, `-`)」のように、断る基準を言う形に直す。入力側の文も同じ言い方にそろえると、2 つの理由が同じ述語を名指す。helper が 1 つなので test の直しは小さい。README の 1 句は `/sync-docs` に渡す |
| LOW | maintainability | L4(残した): `resolveMergeAPIHost` の引数 7 つ、`originURL, host string` の取り違え、`timeout` の 2 本とも同じ値、`sshRun == nil` の既定が 2 か所、「was interrupted」の枝が本番で読まれない。`55cdb7c3` で、同じ形の `mergeSSHResolveRuns(originURL, host string)` が 1 つ増え、「走らない」の理由(`:175-180`)は、`mergeOriginIsSSH` が真のあとに呼ぶ `mergeSSHResolveRuns` の偽を「host が普通の名前でない」と原因を排除法で決める(今は正しい。条件を足すと、文が嘘になる) | `mergeorigin.go:117,121-123,170,175-180,189-190,222-224` | 動きは変わらない。直すなら、host を `originURL` から中で出す、`sshRun` の既定を片方に置く。`:175` は `mergeSSHHostName(host)` を直接使うと、原因と条件が 1 行で結ばれる |
| LOW | maintainability | L6(残した): 偽の ssh を PATH に置いてプロセスグループを確かめる手順が 3 つの test ファイルに複製。変数名 `test` | `mergeorigin_test.go`、`mergeguarantee_alias_test.go`、`regate_ci_test.go` | `soFakeSSH` の隣に共通の確認を置く(L6 の元の推奨のまま) |
| LOW | readability | L7(残した): (a) 位置指定の 7 欄の表(`55cdb7c3` で約 15 行増えた)、(b) `map[bool]int{...}[tc.sshRuns]` が 2 か所(`:166`、`:359`)。tech-debt の org-merge-order の行の (e) が同じ形の `map[bool]string` を挙げている | `mergeorigin_test.go` の表、`mergeguarantee_alias_test.go:166,359` | (a) 欄名つきの struct literal、(b) `want := 0; if tc.sshRuns { want = 1 }`(L7 の元の推奨のまま) |

## 依頼された点の確認

| 問い | 結果 | 根拠 |
| --- | --- | --- |
| 直しは正しく完全か(L1) | 正しい。完全 | 入力と出力が 1 つの述語を通り、`ssh` を打つのは 1 か所。`-` で始まる host は許可リストの先頭の条件で落ちるので、`--` を付けなくても option に読まれない。許可リストは `Match exec` の `%h`・`%n` が shell に渡す文字(`$ ( ) ; \| & ' " \` < > * ? ~ ! = , + @ # \ 空白 制御文字`)をすべて落とす(`TestMergeSSHHostName` の表がそれらを含む 35 件を挙げる) |
| 以前は解けて、今は origin の host のままになる host | 8.2.1 からは 0 件。この PR の 1 回目の版(15066e7c)からは、ASCII でない host、`%` の zone を持つ IPv6、`[]` の host | 8.2.1 は ssh の別名を一切解かなかった。D6 が「今の版と同じ扱いになるだけで、悪くはならない」と書く。`mergeOriginFromURL` は `ssh://` の host から `[]`・port・ユーザー情報を外し、scp 形はユーザーを外すので、普通の別名(`github.com.emu`、`gh_work`、`gh-work`、IPv4、zone のない IPv6)は通る。GitHub・GHES・EMU の host 名は ASCII なので、実害は見当たらない。ただし、そのとき利用者に出る理由の文が誤り(L10) |
| `%q` にした 1 つの理由の文 | 文自体は正しい。同じ文の中に生の host が残る(L9) | `mergeorigin.go:177-179` は `%q`。隣の `printed the hostname %q`、`mergeGuaranteeErrText` を通る `failed (...)` と、host を引用する流儀がそろった。`cmd := "ssh -G " + host` は普通の host 名だけが届くので、生で安全 |
| help の文言 | merge と cleanup は正確。regate と doctor のコメントが抜けている(L8) | 「`ssh -G` connects to nothing, but it runs the commands of any `Match exec`」は、計画の Assumptions と README と evidence の言うとおり(OpenSSH の挙動は、この run では再確認していない)。`Help` の test が 2 つとも新しい文を固定する |
| tech-debt の register を diff と突き合わせた(両方向) | 問題なし | `55cdb7c3` は register も README も触らない。250 行の RESOLVED の註に、`mergeSSHHostName` に反する文はない。残した L4・L6・L7 は今は register に入れない(パイプラインはまだ 1 回目、`cycle-count.json` は `cycle: 1`) |

## Positive notes

- 入力と出力の host の検査を、論点が出た 1 か所の述語に集めた。`ssh` を打つ箇所が 1 つなので、「抜け道がない」ことを `git grep` 1 回で確かめられる。述語の doc が、許可リストの理由(`Match exec` の `%h`、gh の `-R`)と、何を断るかを言う
- `TestMergeSSHHostName` の表が、通る名前 11 件と断る名前 35 件を 1 か所に並べる。断る側に `--help`・`-oProxyCommand=x`・`fe80::1%en0`・`git\xffhub.com` のような、述語の穴になりやすい文字がそろっている
- L3 の test(`mgDoneOnceHostRead`)は、親の ctx が ssh のあとにだけ終わるという、`readOrigin` の中で届きにくい分岐に、疎な fake で届く。「走らなかった」3 種(普通の名前でない、`-` で始まる、https)で空を、走った別名で文を、それぞれ見る
- `55cdb7c3` の commit は計画の D6・Risks・Progress を同じ変更の中で直し、ずれ(`%q`)を Progress に書いた。許可リストの外に出る hostname(ASCII でない、`%`)が「今の版と同じ扱い」であることも D6 に書いてある

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(今は) | - | L4・L6・L7 は動きを変えない形・test の複製と書き方で、この PR の中でも直せる。パイプラインの 1 回目の中なので、register には入れない | 最終の cycle まで直されずに残ったら、1 行にまとめて register に足す(L4 と L7 は同じ test の表と `resolveMergeAPIHost` の周り、L6 は 3 つの test ファイル) | - |

この run は、依頼された範囲(report だけの commit)に合わせて `docs/tech-debt/README.md` を変えていない。

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はなし。LOW 6 件。前回の 7 件のうち 4 件が解決し、新しい 3 件を足した)
- Follow-ups:
  - L8 と L10 は利用者に見える文字列(help、`doctor.go` のコメント、理由の文)なので、「記録だけの修正」の例外に入らない。直すなら `/verify` の前にまとめて直す(`org_regate.go` の 1 文、`doctor.go` のコメント、`mergeorigin.go:211` と `:179` の文、`notAHost` と `notRun` の helper)。L9 は同じ `mergeorigin.go` の 1 関数なので一緒に直せる。直したら、この report をもう一度書き換える(L1 から L10 の番号は変えない)
  - 直さずに進めるなら、L8・L9・L10 を PR 本文の Known gaps に書く
  - README の doctor の行の「prints no hostname that `gh` can take」は、L10 を直したあとに `/sync-docs` で合わせる
  - 計画の Scope の 2 行(21・22 行目)は、まだ 1 回目の規則(解いた値が「空・`-` で始まる・空白や制御文字や `/`」)を言う。D6 と Risks が新しい規則を言い、ずれとして記録されているので、致命的ではないが、archive が残す記録なので、`/pr` の前に 2 行を D6 の規則に合わせると読み間違いがない
  - 計画の Progress checklist の「Review artifact created」は、この report で埋まる。印は呼び出し側が付ける
  - この run は、依頼された範囲(report だけの commit)に合わせて、insight event を書いていない。skill の手順のとおりに残すなら `./scripts/insights-append.sh --slug org-ssh-alias-host --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 6 --cycle auto --source skill`
- Known gaps:
  - `ssh -G` は既定の ssh の設定を読み、`core.sshCommand` と `GIT_SSH_COMMAND` の `-F` は見ない(gh も同じ)。別名が `-F` の設定にだけある checkout では、解けないまま案内に進む
  - gh の `Translator` との読み方の一致(最後の `hostname` の行、解いたあとの読み替え)は、計画の Assumptions を信じて、go-gh のソースを読み直していない
  - 本物の `Match exec` を書いた ssh の設定での実走は、していない(実プロセスのテストは偽の ssh)。`ssh -G` が `Match exec` を実行するという help の文も、この run では OpenSSH で再確認していない
  - 許可リストの外の hostname(IDN、zone 付きの IPv6)は、実在の設定では試していない。表のテストと `mergeOriginFromURL` の読みから、origin の host のまま進むと判断した
  - テスト、静的解析、spec の合致、文書のずれは見ていない(`/verify`、`/test`、`/sync-docs` の範囲)
