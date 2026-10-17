# Self-review report: org-auto-merge

- Date: 2026-10-17
- Plan: `docs/plans/active/2026-10-17-org-auto-merge.md`(ステージ 8a-3a、パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff 15d22b1a...HEAD`(HEAD 8f5eed26、48 ファイル、+5999/-313)。diff の品質だけを見た。仕様適合、テストの網羅、文書のずれ、静的解析、テストの実行は `/verify` と `/test` に任せ、このレビューでは走らせていない。`gh` も実行していない

## Evidence reviewed

- 全文を読んだ: `internal/org/merge.go`(620 行)、`internal/org/merge_run.go`(385 行)、`internal/cli/org_merge.go`(212 行)
- diff を読んだ: `internal/org/mergeguarantee.go`(新規の `CheckMergeEnforcement` 一式)、`internal/org/mergeorder.go`(`pr_lookup`、`lookupMergePR` の `-R`)、`internal/org/lockfile.go`、`regate.go`、`spawn.go`、`seat.go`、`internal/config/config.go`、`internal/cli/org.go`、`org_director.go`、`status.go`、`org_regate.go`、`org_merge_order.go`、`scripts/ralph-config.sh`、`templates/base/ralph.toml`
- skill: `.claude/skills/director/SKILL.md` の merge の節、`.claude/skills/org/SKILL.md` の `merge` の行。4 面は `cmp` で一致(`.claude`、`.agents`、`templates/base` の 2 面。director と org の両方)
- 文書: `docs/tech-debt/README.md` を `15d22b1a` と行ごとに比べた(行数 223 から 228、変わった行は 236〜243、259、291〜295)。README、`AGENTS.md`、仕様の注、quality-gates は該当の diff を読んだ
- テスト: `director_skill_sync_test.go` の diff、`merge_test.go`、`merge_run_test.go`、`org_merge_test.go` は構成と fake の作りを読んだ(全ケースは読んでいない)
- 機械的な確認: `git diff --check` は空。diff の追加行に TODO・FIXME・print・debug・`$HOME` の実パスはない。コメント行の最大幅は新規 3 ファイルとも 80 以内
- argv の経路(依頼された重点): 下の「確認して問題がなかったもの」にまとめた

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

CRITICAL 0 件、HIGH 0 件、MEDIUM 2 件、LOW 8 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | M1: `internal/org/mergeguarantee.go` が 848 行から 1183 行に増えた。`CheckMergeEnforcement` 一式(約 300 行)が、doctor の読み取りと同じファイルに入っている。`mergeguarantee_test.go` も 1334 行から 1774 行。計画は merge.go が 800 行を超えないよう `merge_run.go` に分けたが、同じ配慮がここにはない。tech-debt の 292 行目は `merge_test.go` と `merge_run_test.go` の超過だけを載せ、この 2 ファイルは載っていない | `wc -l`(base 848、HEAD 1183)。新しい部分は `mergeguarantee.go:659`(`mergeEnforceFix`)から `:965`(`notEnforced` の終わり)。`MergeEnforcementParams`、`MergeEnforcement`、`CheckMergeEnforcement`、`enforce`、`enforceUnread`、`readEnforcementRepository`、`mergeQueueArgs`、`readMergeQueue`、`enforceRulesets`、`enforceClassic`、`notEnforced`。regate では同じ超過をファイル分割で直した前例がある(tech-debt 278〜280 行) | 同じ package のまま `mergeenforce.go` に移す。`mergeGuaranteeRun` を共有するだけなので動作は変わらない。移さないなら、292 行目に `mergeguarantee.go` と `mergeguarantee_test.go` を足す |
| MEDIUM | maintainability | M2: `/director` skill の auto_merge = true の手順 4 と「終了コード 1 なら止め、打ち直さない」の箇条が、regate の直後に起こるはずの断りに手を持たない。regate は branch に push して PR の CI をやり直させるので、手順 4 の `ralph org merge` はおそらく検査が pending(または 0 件)で断られる。確認はしていない。計画の Risks にも 0 件の挙動は未確認とある。動詞の断りの文は「pending を待ち、通ったら打ち直す」と言い、skill は「人に頼まれないまま打ち直さない」と言うので、2 つの指示が向き合っている。止まる向きなので安全だが、auto の流れは最初の 1 本の merge と 1 本の regate のあとで人に戻る可能性が高い | `.claude/skills/director/SKILL.md:468-475`(手順 2 と 4)、`:489-491`(止める箇条)。skill に `CI`・`pending`・`検査` の語は merge の節にない(`grep` で確認)。`internal/org/merge.go:553-556`(「wait for the pending ones ... and once all passed, run ... again」)。`merge.go:550-552`(0 件の断り)。pin する断片は `director_skill_sync_test.go` の `directorSkillAutoMergeSteps` の 4 番目と 6 番目 | skill で次のどちらかを決める。(a) 検査待ちの断りに限り、読み取りだけの待ち(merge-order と検査の再読み、回数と間隔を決める)のあとで 1 回だけ打ち直してよいと書く。(b) 検査待ちで止まるのは想定の動きだと手順 4 と「まだないもの」に書く。どちらでも pin の断片をそろえる。AC11 をどう読むかは `/verify` の判断でよい |
| LOW | unnecessary-change | L1: `mergePRViewFields` が `mergeCommit` を要求しているが、読む側の `ghPRView` に受ける field がない。OPEN の PR の `mergeCommit` は常に null で、merge のあとの読み直し(`mergeReadBackFields`)が別に読む | `merge.go:39`(`state,url,headRefOid,baseRefName,autoMergeRequest,mergeCommit`)、`merge.go:452-458`(`ghPRView` に `MergeCommit` なし)。文字列は `merge_test.go:441,445,648` に pin されている | `mergeCommit` を外し、テストの 3 か所を合わせる。残すなら理由を `mergePRViewFields` の注に 1 行書く |
| LOW | naming | L2: `pr_merged` の `auto_merge=` は設定の値で、その merge が自動で打たれたかは表さない。人に頼まれた merge でも、設定が true なら `auto_merge=true` が残る。`MergeResult.AutoMerge` の注と help は「報告するだけ」と書くが、永続する側の `EventPRMerged` の注には書いていない。台帳と report を追う人(仕様のユーザーストーリー 5)が「自動で merge された」と読みやすい | `merge_run.go:359-361`(Details の書式)、`seat.go:90-104`(`auto_merge=<true\|false>` とだけある)、`merge_run.go:91-93`(`MergeResult.AutoMerge` の注) | `seat.go` の注に「`[org.director].auto_merge` の値で、director が人に聞かずに打った merge かどうかは表さない」と足す。key 自体を変えるなら、永続する前の今が最後の機会 |
| LOW | exception-handling | L3: merge のあとの gh(読み直し、`--disable-auto`)は `context.WithoutCancel` で ctx の取り消しからは守られるが、ralph と同じプロセスグループにいる。端末の Ctrl-C と SIGHUP は前景のグループ全体に届くので、2 回目の Ctrl-C や、読み直しの最中に最初の Ctrl-C が届いた場合は、その gh も落ちる。help と doc は「merge が始まったあとに中断されても、読み直しと外しと記録を 1 回済ませる」と書く。テストは ctx を取り消すだけで、実際のシグナルは送っていない。落ちた場合の Failed の文は「merge されたか分からない」「予約が残るかもしれない」と状態を言うので、利用者が迷う形ではない | `mergeorder.go:1194-1212`(`realMergeOrderExec` は `SysProcAttr` を設定しない)。`merge_run.go:254-255`(読み直し)、`:306-307`(外し)。`org_merge.go:97-103`(help)、`merge_run.go:116-121`(doc)。`merge_run_test.go:454-560`(ctx の取り消しだけ)。前例は `gate_run.go:189` と `regate_git.go:180` の `Setpgid: true` | 直すなら、merge のあとの gh だけ別グループで走らせる(`gh` は端末を使わない設定なので移せる)。直さないなら help の「読み直す」に「2 回目の中断や読み直しの最中の中断では gh も止まり、その場合は状態が分からないと言う」と足し、tech-debt に 1 行 |
| LOW | naming | L4: `withMainWorktreeDirectorEnabled` が `AutoMerge` も運ぶようになったが、名前は `Enabled` のまま。`statusDirectorConfig` は `OrgDirectorConfig` 全体を返しつつ、doc で「`Enabled` と `AutoMerge` だけを読む前提」と断っている。escalate と `director attach` の設定エラーの文も「enabled and auto_merge」に変わり、auto_merge を読まない動詞にも出る | `internal/cli/org.go:217-232`(関数とエラーの文)、`internal/cli/status.go:256-258`(戻り値の型と doc)。呼び出し側は `org_merge.go:136`、escalate、attach、status | 関数を `withMainWorktreeDirectorSwitches` のように読み取る内容が分かる名前にするか、doc の先頭で「`Enabled` と `AutoMerge` の 2 つ」と名指す。`statusDirectorConfig` は 2 つの bool を返す小さな struct にすると、読んではいけない値が見えなくなる |
| LOW | readability | L5: `Merge` が regate の関数のエラーをそのまま包むので、前置きが混ざる。マーカーを空にする失敗は `org: merge: nothing was merged: org: regate: write the marker of ...` になる。ロックを取れない失敗はパスを 2 回言う(`since <path> could not be taken: org: acquire regate lock <path>: ...`) | `merge_run.go:138`(`regateLockPathIn(stateDir)` と、包んだ `err` の中のパス)、`merge_run.go:143-145`、`regate.go:511-515`(`org: regate: write the marker`) | 138 行の書式からパスを外す。マーカーの失敗は、regate の前置きが残ってよいなら注で言う |
| LOW | security | L6: `gh repo view <repoArg>` だけが、origin の URL から作った文字列を位置引数で渡す(`-R` と `--hostname` は値として取るので安全)。`mergeOriginFromURL` は host の先頭が `-` かを見ないので、`-` で始まる host は gh にフラグとして読まれうる。gh は失敗するだけで実行に至る道は見つからなかったが、gh の挙動は確かめていない。origin の URL は手元の git config 由来なので、攻撃には悪意のある `.git/config` が要る | `merge.go:571-572`(`"repo", "view", r.repoArg`)、`mergeorder.go:827-858`(scp 形式は `strings.Cut(s, ":")`、URL 形式は `u.Hostname()`。どちらも先頭の `-` を弾かない) | `mergeOriginFromURL` で host と owner と repo の先頭が `-` なら `ok=false` にする(2 行)。または `readOrigin` で同じ検査をする |
| LOW | maintainability | L7: tech-debt の 239 行目は、前半だけに取り消し線を引き、後半を同じ Debt のセルに残した形になった。Impact と Why deferred のセルは、取り消した前半(`--json` が lookup の結果を区別できない)を今も説明している。取り消しの注の「see the comment above」は、表の中に「above」にあたるものがなく、指す先がない | `docs/tech-debt/README.md:239`。セル 2 は「A program that reads the JSON ... reads a branch whose PR lookup failed ... as a branch with no PR」、セル 3 は「A field for the lookup's outcome is a change to the JSON contract」。セル 4 の Trigger だけが「resolves the JSON part」と更新されている | 取り消した文を残し、セル 2 と 3 は runner の重複(後半)だけの文に書き直す。「see the comment above」は「`MergeOrderEntry.PRLookup`(`internal/org/mergeorder.go`)」に置き換え、Related のセルにも同じシンボルを足す |
| LOW | typo | L8: 引数名をそのまま名詞に続けて、重複した語に読める箇所が 3 つある | `mergeguarantee.go:915`(「requires checks checks and branches」)、`merge_run_test.go:195`(「for the merge method method」)、`mergeguarantee_test.go:1358`(「permissions.admin admin」) | 「requires <checks> required checks」「for the merge method <method>」「with the permissions.admin value <admin>」のように書き直す |

## Positive notes

- 外部入力から argv を作る経路は、型で絞ってある。PR 番号は `int`(`strconv.Itoa`)、`--match-head-commit` の値は gate の記録の SHA(`gitObjectPattern` で検査済み)で、`viewPR` が PR の head と、`checkOriginUnmoved` が ls-remote の先頭と一致することも確かめる。方式の flag は固定の map(`mergeMethodFlags`)から出る。GraphQL の変数は `-f`(`-F` は `@` や数を読み替える)、REST の path は `url.PathEscape`
- gh の呼び出しはすべて origin の repository に固定されている。`gh pr list` / `view` / `checks` / `merge`(`--disable-auto` を含む)は `-R <host>/<owner>/<repo>`、`gh repo view` は引数、`gh api` は `--hostname`。PR の url は host と owner/repo を origin と比べ、host の大文字小文字は畳む(`isOriginsPR`)
- gh から来た文字列を出す経路は 1 か所に集まっている。`checkMerge` の最後で `PrintableInboxText`、Failed と Note は `failed()` と `mergeGHSaid`(行ごとに `mergeGuaranteeErrText`)、成功時の行は `printMergeLines` で個別に通す。`res.Method` だけは通さないが、値は固定の 3 語
- `gh pr merge` に `--auto`、`--admin`、`--delete-branch` がない。読み直しは終了コードによらず 1 回行い、MERGED のときだけ記録する。`Merged` と `Recorded` を分け、記録に失敗したときは「merge は済んでいる」と言う
- `regate.lock` は `defer` で必ず外れる。取った直後にマーカーを空にし、merge の間は書かない(D3)。ロックの待ちは regate と同じ定数を使う
- 想定外の値(知らない `current_user_can_bypass`、読めない保護、知らない merge 方式、型が違う JSON)はすべて断る側に倒れる
- テスト用の hook は `regateTestHooks` と同じ形で、本体は hook が nil なら何もしない。tech-debt の追加行(291〜295)は 5 セルで、`|` のエスケープ漏れはない
- 4 面の skill、`scripts/ralph-config.sh` の 2 面は `cmp` で一致。`defaults_sync_test.go` が `auto_merge` の 3 面を結ぶ

確認して問題がなかったもの(再指摘しない):

- 計画のパスを指す Go コメント(`docs/plans/active/2026-10-17-org-auto-merge.md`)は 13 か所ある。archive で古くなる点は前例(regate など)と同じで、D と AC の ID は計画の中で安定している
- 断りの文の中のコマンド(`ralph org merge --branch <b> again`、`gh pr create --head <b>`)は、ブランチ名をクォートしない。regate の `words.again` と同じ前例で、git の ref 名は空白と制御文字を含められない
- `--state-dir` を足さない案内も regate と同じ前例
- 最悪時間は、`git remote get-url` 15 秒、ls-remote 30 秒を 2 回、gh 15 秒を 6 回(pr list、view、checks、repo view、読み直し、外し)、保護 20 秒、merge 60 秒を足して約 4 分(245 秒)。skill の「前景で打ち、Bash の timeout は 10 分」と合う
- `pr_merged` は `RunningOrgs`、`orgsToDisband`、`leaderActivityEventCount` のどの条件にも入らない(コードを読んで確認。テストは `/test` が見る)
- `mergeTestHooks` と `regateTestHooks` はどちらも非並列のテストだけが書き換える
- `AutoMerge: false` を `Default()` に明示するのは冗長だが、ほかの director の値と並んで読める

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| M1 を直さない場合: `mergeguarantee.go`(1183 行)と `mergeguarantee_test.go`(1774 行)が 800 行の目安を超えている | 読みにくさだけで、動作は変わらない | 動作を変えない分割で、このスライスの外 | `CheckMergeEnforcement` か doctor の次の変更。tech-debt 292 行目に 2 ファイルを足す形でもよい | 計画の D11、この report の M1 |
| L3 を直さない場合: merge のあとの gh(読み直し、`--disable-auto`)が ralph のプロセスグループにいて、端末の 2 回目の中断で落ちる | 落ちると PR の状態が分からないと言って終わる。merge 済みでも記録されない | 外部コマンドの runner が 3 か所に重複している(239 行目の後半)ので、まとめて直すほうが安い | runner の統合、または中断の報告 | 計画の D6、この report の L3、tech-debt 239 行目 |

_(上の行は追加していない。依頼により、report 以外のファイルは編集しなかった。`docs/tech-debt/` への追記は、直さないと決めた finding の分だけ親側で行う。)_

## Re-review (S6・S7、29469c0e..e8299e07)

- Reviewer: reviewer サブエージェント(`/self-review` の再実行。パイプラインの cycle は増やさない)
- Scope: `git diff 29469c0e e8299e07`(HEAD e8299e07、34 ファイル、+2255/-920)。見たのは S6 の 4a39e909 と S7 の e478e350。54ec0d7e、53bef7e1、67ccd546、e8299e07 は計画と insight の記録だけなので、計画の本文は見ていない。diff の品質だけを見た。テストと `gh` は走らせていない
- 読んだもの:
  - M1 の移動: `git show 4a39e909` の消えた行(mergeguarantee.go 306 行、mergeguarantee_test.go 440 行)と増えた行(mergeenforce.go 326 行、mergeenforce_test.go 452 行)を `diff` で突き合わせた。違いは package と import、先頭の説明コメント、L8 のコメント(`enforceClassic` の doc、`meRepoJSON` の doc、テストの先頭の説明)だけで、コードの行は同じ。mergeguarantee.go が外した import は `bytes` と `strconv` の 2 つで、残りの import は本文で使われている。mergeenforce.go と mergeenforce_test.go の import も、すべて本文に使い道がある(`grep -c`)
  - S7: `merge_wait.go` と `merge_wait_test.go` の全文、`merge.go` と `merge_run.go` と `org_merge.go` の diff、`readChecks` と `checkMerge` の本文、`/director` と `/org` skill の diff。skill の 4 面(`.claude`、`.agents`、`templates/base` の 2 面。director と org)は `cmp` で一致
  - 機械的な確認: `git diff --check 29469c0e e8299e07` は空。追加行に TODO・FIXME・print・`/Users/` はない。tech-debt の変わった 6 行(239、243、292、293、295、296)は 5 セルで、`|` のエスケープ漏れはない。S6・S7 の新規ファイルはどれも 800 行未満(merge_wait.go 193、merge_wait_test.go 576、mergeenforce.go 326、mergeenforce_test.go 452)

### 前回の finding の判定

- M1(`mergeguarantee.go` の増え方): 直った。移動は純粋で、中身は変わっていない(上の突き合わせ)。mergeguarantee.go は 1183 から 877 行、mergeguarantee_test.go は 1774 から 1334 行に戻った。877 行と 1334 行はまだ 800 行を超えるが、tech-debt 292 行目が両方を載せ、base の 848 行と 1334 行も書いてある。参照の更新(`seat.go` の gh の一覧、AGENTS.md、tech-debt 243・293・295 行)も済み、`CheckMergeEnforcement` を `mergeguarantee.go` にあると書く箇所は残っていない
- M2(skill の auto 手順と検査待ちの断りの食い違い): 直った。D13 で動詞が待つ形にし、skill の手順 2 は `--wait-checks 60m` を付けて background で打つ。pin の断片(`directorSkillAutoMergeSteps`)も増えた。新しい問題は R1 から R4 だけ
- L1(`mergeCommit` の要求): 直った。`merge.go:41` から外れ、テストの 3 か所(`merge_test.go:457`、`:689` ほか)も合っている
- L2(`auto_merge=` の意味): 直った。`seat.go` の `EventPRMerged` の注に、設定の値であり director が聞かずに打ったかは表さないと書いた
- L3(merge のあとの gh が同じプロセスグループ): 直った。`realMergeDetachedExec`(`runMergeOrderCommand` の `ownGroup`)は `Setpgid` と、ctx 終了でグループごと SIGKILL で、`gate_run.go` と `regate_git.go` の前例と同じ形。使うのは読み直しと `--disable-auto` だけ(`ghAfterStart`)で、`gh pr merge` 自体は ralph のグループのままにしてあり、help と doc がそう書いている。テストは、差し替え先、pgid、ctx 終了での子の kill の 3 つを見る。新しい問題はない
- L4(`withMainWorktreeDirectorEnabled` の名前): 直った(doc の側)。関数の doc とエラーの文を「`[org.director]` の 2 つの値」に直し、`statusDirectorConfig` の doc は `readOrgDirectorReport` が `Enabled` と `AutoMerge` だけを読むと書く。`org_director.go:526` で確かめた。名前は変えていない
- L5(エラーの前置きとパスの重複): 直った。ロックを取れない失敗はパスを 1 回言い、マーカーを空にする失敗は `lock.f.Truncate(0)` で merge 自身のエラーにした。`writeRegateMarker(f, nil)` の中身(`Truncate(0)` と空の `WriteAt`)と同じ動きで、注がその対応を書いている
- L6(`gh repo view` の位置引数): 直った。`readOrigin` が host・owner・repo の先頭の `-` を断る(`merge.go:364-365`)。テストは scp 形式、URL 形式、owner、repo の 4 件
- L7(tech-debt 239 行目): 直った。取り消した前半の文は残し、Impact と Why deferred は runner の重複だけの文に変わり、「see the comment above」は `MergeOrderEntry.PRLookup` の名指しになり、Related にも同じシンボルを足した。5 セル
- L8(単語の重複): 直った。3 か所とも言い直した。`enforceClassic` の新しい文(「with checks the number of its required status checks」)は少し硬いが、意味は通る

前回の「Tech debt identified」の 2 行(M1 と L3 を直さない場合)は、どちらも直ったので不要になった。

### 依頼された点の確認(finding にならなかったもの)

- 待つ間にロックを外している: `Merge` の `for` ループ(`merge_run.go:142-151`)が `mergeLocked` を呼び、`mergeLocked` は `defer lock.release()` で外してから `wait` を返す。待ちはループ側の `w.wait` で、ロックを持たない。テストの fake clock は sleep のたびに `acquireHeldFileLock` で取れることを確かめる(`lockTaken`)
- 待ちのあと取り直して最初からやり直す: 次の周回で `acquireHeldFileLock`、マーカーを空にする、`checkMerge` の全 step の順。head と base が動けば最初の確かめが拾う(テストの「the branch moved」)
- 予算を sleep が超えない: `poll` は `min(w.interval, left)` だけ寝て、`left <= 0` で「使い切った」と返す。期限は最初の待ちの開始から数え、2 回目以降の待ちと間の確かめの時間も引かれる。最後の read(gh の timeout 15 秒)と取り直したあとの確かめは予算の外に出るが、help の「time is up」は範囲を言い切っていないので、問題にしない
- 中断: sleep 中も read 中も `wait` がエラーを返し、`Merge` は `MergeResult{}` で終わる。`gh pr merge` は走らず、何も記録しない(テストの `TestMerge_WaitChecksInterrupted`)
- 待つかどうかの決め方: `merge_run.go` と `merge_wait.go` に拒否の文字列を比べる箇所はなく、`kind.waits()` だけで決める
- r が漏れないか: `checkMerge` の switch の `refused != "" && !kind.waits()` が `mergeReady{}` を返し、`r` を返すのは待つ種類のときだけ。step 1 から 9 の拒否は `kind` が `mergeChecksOther` なので空になる。`TestCheckMerge_ReadsTheChecks` がこの対応を種類ごとに見る
- 0 件の文の読み取り(`readChecks`): 3 つ目の case は `err != nil && !listed` で、かつ `err.Error()` が `fmt.Sprintf(mergeNoChecksSentence, r.branch)` を含むときだけ。`realMergeOrderExec` は終了コードの文のあとに stderr を付けるので、文は `err.Error()` に入る。文が引用符で囲まれるので、別のブランチ名(`feat/a2`、`feat/b`)の文は一致しない(テストにある)。502 などの別のエラー、タイムアウト、gh 不在はこの case の前か後で `mergeChecksUnread` になる。exit 0 で stdout にこの文だけがあるときも `!listed` で `Unread`。残る前提は gh の文言で、gh 2.102.0 のソースを読んだ結果だと計画にあり(私は gh を打っていない)、tech-debt 296 行目が文言が変わった場合を載せている
- 進捗行の escape: `line` は `PrintableInboxText(mergePrefix+text, false)` を通す。repository の名前は origin の URL 由来で、start の行に入る(`cmd`)が escape される。待った時間を足す `waited()` と終了の行は `mergeWaitDuration` を通す。start の行だけ通っていない(R2)
- CLI の範囲の検査: `org_merge.go:151` で、`--org-id` と `--branch` の確認のあと、台帳を読む前にある。`Merge` の中でも同じ検査を先頭でやるので、空の `--branch` と範囲外の `--wait-checks` を同時に渡すと、CLI は branch の文、`Merge` を直に呼ぶと範囲の文になる。どちらも何も実行しないので、順の違いは問題にしない
- skill の背景実行の手順: `.harness/state/merge-<名前>.out`、`/` を `-` にする、終わるまでほかを打たない、が regate の手順(`SKILL.md:541`)と同じ形で、`<名前>` はその場で定義してある。待ちが検査の `pending` か 0 件のときだけで、60 分で終わらなければ断りで止まる、まで書いてある。CI のない repo で 60 分待って止まることは、help と D13 が書いている

## Re-review findings

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 4 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | R1: S7 で `--wait-checks` の待ちの間は Merge が `regate.lock` を外すようになったが、「merge はロックを run の間ずっと持つ」と書いた 3 か所のコメントが残っている。help、README、仕様、skill は直っているが、この 3 ファイルは S7 の diff に入っていない | `internal/org/lockfile.go:133`(「ralph org merge holds regate.lock too, for its whole run」)、`internal/org/regate.go:91`(「holds it too, for its whole run」と、続く「one Regate or Merge runs on a ledger at a time」)、`internal/org/spawn.go:1325`(「A `ralph org merge` holds the lock too, for its whole run」)。外す側は `merge_run.go:142-151` のループと `mergeLocked` の `defer lock.release()`。`git diff --stat 29469c0e e8299e07 -- internal/org/lockfile.go internal/org/regate.go internal/org/spawn.go` は空 | 3 か所の「whole run」を「checks から merge まで。`--wait-checks` で検査を待つ間は外していて、その間は regate が走れる(merge_wait.go)」に直す。spawn.go の文は、待つ間もロックが空くので spawn が通る点は変わらず、語だけ直せばよい |
| LOW | readability | R2: 2 回目以降の待ちの開始行の「up to <残り>」が丸められず、ナノ秒まで出る。1 回目は残りが予算そのものなので `1h0m0s` と出るが、2 回目は取り直した確かめに実時間がかかるので、`59m26.847291035s` のような値になる。同じ待ちの終了行(`mergeWaitDuration`)と、断りに足す `waited()` は秒に丸めている。テストは fake clock が sleep でしか進まないので、2 回目の行を `59m30s` と書いて通ってしまい、この差を拾えない | `merge_wait.go:127-128`(`max(w.deadline.Sub(start), 0)` を `%s` に渡す)、`merge_wait.go:130` と `:106`(`mergeWaitDuration` を通す)、`merge_wait.go:176-181`。`merge_wait_test.go:272`(`mwStart("has checks pending", "59m30s", "1h0m0s")`) | `max(w.deadline.Sub(start), 0)` を `mergeWaitDuration(...)` で包む。テストは、確かめの間に fake clock を 1 秒未満の端数だけ進めて(例: `exec.after` で `now` に 123ms を足す)、2 回目の行が丸まることを見る |
| LOW | maintainability | R3: 実時間のテストの `polls < 2` が、負荷で落ちうる。予算 60ms、間隔 10ms で、最初の sleep が 50ms 以上遅れると、次の周回の `left <= 0` で read が 1 回で終わり、`polls` が 1 になる。`-race` や `go test ./...` の並列では起きうる遅れ | `merge_wait_test.go:547-560`(`60*time.Millisecond`、`interval: 10 * time.Millisecond`、`polls < 2`) | 予算を 300ms ほどに広げるか、`polls < 1` にする。fake clock のテストが回数を正確に見ているので、この実時間のテストは「終わる」と「予算より短く終わらない」だけ見れば足りる |
| LOW | readability | R4: 小さな読みにくさが 5 つある。(a) `MergeResult.Refused` の doc の一覧に、範囲外の `--wait-checks` がない。(b) `poll` の空の `case kind.waits():` に注がなく、C の fall-through に見える。(c) `Merge` の doc に、`(mergeWait, D13). It then runs gh pr merge once` の短い行(整形の途中の形)と「the lock and every check are taken again」(「every check」は taken ではなく run)がある。(d) `Merge` の局所変数 `wait`(`*mergeWaitFor`)が、`w.wait(...)` と並んで読み違えやすい。(e) `wait` の中断のエラーは `mergeInterrupted` と同じ文を手で書いている | (a)`merge_run.go:52-55`。(b)`merge_wait.go:157`。(c)`merge_run.go:106-112` あたり。(d)`merge_run.go:144-148`。(e)`merge_wait.go:134-135` と `merge.go:336-337`。`mergeInterrupted(ctx, "waiting for the checks of PR #7, before gh pr merge ran")` が同じ文字列を作る | (a)に範囲外を足す。(b)に「待ち続ける」の 1 行を足す。(c)は文を直して整形し直す。(d)は `waitFor` のような名前にする。(e)は `mergeInterrupted` を呼ぶ。どれも動作は変わらない |

### Tech debt identified

新しく足す行はない。R1 から R4 はどれもコメントと小さな直しで、直さないなら 1 行にまとめて tech-debt に載せる(Trigger は `merge_wait.go` か `Merge` の次の変更)。800 行を超えるファイルのうち S6・S7 で増えたものは tech-debt 292 行目が載せている。載っていないのは `internal/org/mergeorder.go`(base 1152、HEAD 1238、S6 が +26)で、`realMergeDetachedExec` はこのファイルの runner の隣にあるので置き場所は妥当だが、行数の行はない。もとから 800 行を超えている大きなファイルが repo に多く(`spawn.go` 2333 行ほか)、この PR で新しく超えたものではないので finding にはしない。

_(上の変更は report だけで、`docs/tech-debt/` には追記していない。)_

## Cycle 2 re-review (S8、b1cbe054..298095c6)

- Reviewer: reviewer サブエージェント(`/self-review`、パイプライン cycle 2)
- Scope: `git diff b1cbe054..298095c6`(HEAD 298095c6、24 ファイル、+994/-395)。d8e6afb3(計画)、S8a 9506ec64、S8b b2f0b534、S8c 320b4d75、298095c6(`AGENTS.md` と計画)。cycle 1 のコードで S8 が触っていない部分は見ていない。diff の品質だけを見た。テスト、静的解析、`gh` は走らせていない

### Evidence reviewed

- 全文を読んだ: `internal/org/merge_wait.go`(271 行)。`merge_run.go` の `Merge`・`mergeLocked`・`mergeAndRecord`・`otherHeadMerged`、`merge.go` の `checkMerge`・`checkMergeBranch`・`runningRefusal`・`mergeRefusalKind`、`mergeenforce.go` の `enforce`・`readEnforcementRepository`・`enforceClassic`、`org_merge.go` の `printMergeResult`
- diff を読んだ: 上の 3 ファイルの残り、`internal/cli/org_merge.go`(help)、テスト(`merge_test.go`、`merge_run_test.go`、`merge_wait_test.go`、`mergeenforce_test.go`、`internal/cli/org_merge_test.go`、`director_skill_sync_test.go`)、README、`AGENTS.md`、仕様の注、tech-debt の 292・295 行目、計画。`/director` と `/org` の skill は変わった箇所と、`merge` の行の全文
- 機械的な確認: `git diff --check b1cbe054..HEAD` は空。追加行に TODO・FIXME・print・`/Users/` はない。skill の 4 面は director、org とも `cmp` で一致。tech-debt の変わった 2 行(292、295)は 5 セルで、`|` の escape 漏れはない。tech-debt 292 行目の行数(merge_test.go 911、merge_run_test.go 1006、merge_wait_test.go 800、mergeenforce.go 318、mergeenforce_test.go 483)は `wc -l` と合う
- 古い名前: `mergeChecksKind` と `mergeChecks{Other,Pending,Zero,NotPassed,Unread}` は `.go` に残っていない(`grep`)。残りは計画の S7 の進捗 1 行(195 行目。S8 の行が続けて改名を言う)と過去の report だけ。`mergeChecksText`(`mergeguarantee.go`)は別の関数

### 前回の finding の判定

- R1〜R4(S6・S7 の再レビュー): 97a1a062 で直っている。R1 の 3 か所は書き直され、R2 は `merge_wait.go:149` が `mergeWaitDuration` を通し、R4 の `waitFor` と `mergeInterrupted` は `merge_run.go:154-158` と `merge_wait.go:176` にある。R1 で書き直した文が S8c で足りなくなった点だけが新しい finding(C2-1)

### 依頼された点の確認(finding にならなかったもの)

1. `orgWaitEnd` がロックなしで台帳を読む: `checkMergeBranch` も `readMergeLedger(o.Manifest.Path())` をロックなしで呼ぶので、読み方は同じ。`readJSONLines` は読むだけで、壊れた行は数えて飛ばす。追記の途中の行に当たると、その周回は disband を見落とすだけで、30 秒後の次の周回が拾う。共有の状態を書く経路はない(`readMergeLedger`、`RunningOrgs`、`collectMergeTargets` は渡された events を読み、書くのは関数の中で作る map とスライスだけ。`mergeorder.go:423-495` を読んだ)。`checkMerge` は `corruptAfterGate` で壊れた行を見るが、`orgWaitEnd` が見るのは running と `orgRunsOn` だけなので影響しない
2. step 8 の部分的な `mergeReady`: `checkMergeBranch` が step 8 で返す `r` は branch、origin の host と repo、`orgID` だけで、`prNumber`・`head` は空。呼び出し側で読むものは `wait` の `f.r.orgID` と `f.r.branch`(`orgWaitEnd` の中)、`waitedOut` の `r.orgID` だけで、0 の PR 番号や空の head を使う場所はない。`orgRunsOn` の断り(kind は Other)は `checkMerge` が `mergeReady{}` に落とす(`!kind.waits()`)。`TestCheckMerge_OrgThatStillRunsIsWaitedFor` が `r.branch`・`r.orgID`・`r.repoArg` を、`TestCheckMerge_RefusesWithNothingAfterTheFailingCheck` が空の `r` と `kind.waits() == false` を見る。型の注と `wait` の組み立て方は C2-3
3. `waitedFor` の重複除去と文: `forWhat`(`the checks`、または `org_id "x" to be disbanded`)で除き、`" and for "` でつなぐので、「waited 1m10s for org_id "org-a" to be disbanded and for the checks (--wait-checks 1m10s)」と読める(テストが固定)。同じ待ちが 2 回目に起きても 1 回しか出ない。別の待ちを挟んで同じ org に戻ると、並びは初めて起きた順になる(まれで、文は崩れない)。開始行の `left` は `mergeWaitDuration(max(w.deadline.Sub(start), 0))` で、2 回目は残りを秒に丸める(テストは 40s)
4. S8b: `otherHeadMerged` は `res.Merged` を立てず、`res.Failed` だけを返す。`printMergeResult` は `res.Merged` のときだけ `MERGED:` の行を出すので、行は出ない。`headRefOid` が空か欠けているときは別の文(「with no head, so whether … is unknown」)で、同じ分岐に入る(テスト 2 件)。head の比較は `viewPR`(step 9)と同じ完全一致で、gate の head は `gitObjectPattern`(小文字)で検査済みなので、大文字小文字で誤って「別の head」になる道はない。gh から来た head は `failed()` の `PrintableInboxText` を通る(制御文字のテストがある)
5. S8a: `permissions.admin` を読む struct field、戻り値、`enforceClassic` の引数、`admin == nil` と `!*admin` の分岐は、コードから消えている。help、README、仕様の注、skill、tech-debt 295 行目、計画の D11・AC5 は enforce_admins を admin によらず求める形に揃っている。残りは古い fixture だけ(C2-4)
6. 改名: 上の「古い名前」のとおり。`mergeRefusalKind` の 6 つの定数の doc と `waits()` の doc は、新しい kind に合わせて書き直されている
7. skill の 4 面と動詞: director と org の `SKILL.md` は 4 面とも `cmp` で一致。org の `merge` の行は、org の待ちを「台帳だけを 30 秒ごとに読み、org が disband するか、ほかの機能で走り出すか、台帳が読めなくなるまで」と書き、`orgWaitEnd` の 3 つの終わり方と合う。時間切れの断りが `ralph org status --org-id <org>` を添え、disband は人が決める、という文は `waitedOut` と合う。director の skill は disband を打たずに人に伝えると書き、pin テスト(`directorSkillAutoMergeSteps`)が 2 文を固定する。「ほかの機能で走っている org は待たない」は、結びつきのあるブランチだけで成り立つ(C2-2)

## Cycle 2 findings

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 6 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C2-1: S8c で `regate.lock` を外す待ちが 2 つになったが、「待つ間は外す」と書いた 3 か所のコメントは「PR の検査を待つ間」のままで、org の disband を待つ間も外れることが読めない。3 か所とも cycle 1 の R1 の直し(97a1a062)が書いた文で、S8 の diff に入っていない。regate が走れる間を知りたい人に、org の待ちの間は走れないと読める。S6・S7 の R1 と同じ形(待ちを足した変更が、待ちの数を言い切った別ファイルの文を残す) | `internal/org/lockfile.go:136-137`(「releases it while it waits for the PR's checks (merge_wait.go)」)、`internal/org/regate.go:95-97`、`internal/org/spawn.go:1325-1326`(「but not while --wait-checks waits for the PR's checks」)。`git diff --stat b1cbe054 HEAD -- internal/org/lockfile.go internal/org/regate.go internal/org/spawn.go` は空。外す側は `merge_run.go` の `Merge` のループ(`wait` はロックの外) | 3 か所を「検査を待つ間と、org の disband を待つ間」に直す。次に待ちが増えても残らないよう、`merge_wait.go` の先頭に「`regate.lock` を外す待ちはこのファイルのものだけ」と書き、3 か所は `merge_wait.go` を指す形にしてもよい |
| LOW | maintainability | C2-2: 「ほかの機能で走っている org は待たない」は、結びつき(binding)のあるブランチでだけ成り立つ。`orgRunsOn` は `t.binding != nil` のときしか立たない(`collectMergeTargets`)ので、gate の記録だけを持つブランチは、その org がほかの機能で走っていても step 8 は待つ種類(`mergeRefusalOrgRunning`)になり、`orgWaitEnd` も「ほかの機能」を見分けられず、予算(director は 60 分)まで待つ。その間 director は skill のとおりほかの merge も regate も打たず、時間切れの断りは「its leader may still push to the branch」と書く(その org は別のブランチを走らせている)。計画は進捗の 1 行(200 行目)に「安全側」と書いたが、help、README、skill、tech-debt にはなく、計画は `/pr` で archive される。現行の writer では gate の記録の前に結びつきがあるので、起きるのは台帳を手で直したときなどまれ(`TestCheckMerge_OrgThatStillRunsIsWaitedFor` の「gate-only branch」は org-b が feat/b を走らせている形で、この場合そのもの) | `internal/org/mergeorder.go:488-492`(`orgRunning` と `orgRunsOn` は `t.binding != nil` が前提)、`merge_wait.go:238-242`(`orgWaitEnd` は `targets[i].orgRunsOn` だけを見る)、`merge.go:489-504`(`runningRefusal` の `t.binding == nil` は `whose` の言い方を変えるだけ)、`merge_test.go` の `TestCheckMerge_OrgThatStillRunsIsWaitedFor`(ledger は `moBind("org-b", "b")` と `moPassed("org-b", "feat/a", ...)`)、計画 200 行目の末尾、`docs/tech-debt/README.md` に該当の行なし | tech-debt に 1 行足す(下の「Tech debt identified」)。直すなら `orgRunsOn` を `mergeTargetOrg(t)` の org でも立てる(merge-order の分類が変わるので別の変更) |
| LOW | readability | C2-3: `wait` は、PR の検査の待ちの文(`the checks of PR #%d`、`f.r.prNumber`)を先に組み立ててから、org の待ちなら `what` と `forWhat` を上書きする。org の待ちでは `f.r.prNumber` は 0 で、「PR #0」の文字列が一度できる(使われはしない)。「何を待つか」の違いが `if f.kind == ...` の前後に散り、3 つ目の kind が待つ側に足されると、PR #0 の文がそのまま出る。`mergeReady` の型の doc は「every check passed のとき」のままで、待つ種類の断りでは途中までしか埋まらないこと(step 8 なら branch・origin・orgID だけ)が、`checkMerge` の doc にしか書かれていない。`mergeWaitFor.r` の doc にもない | `merge_wait.go:150`(`what, forWhat := fmt.Sprintf("the checks of PR #%d", f.r.prNumber), "the checks"`)と `:152-158`(org なら上書き)。`merge.go:94-96`(`mergeReady` の doc)、`merge_wait.go:80-85`(`mergeWaitFor`) | `switch f.kind` で `what`・`forWhat`・`read` を kind ごとに 1 か所で決める。`mergeReady` と `mergeWaitFor.r` の doc に「待つ種類の断りでは、断った step までに読んだ field だけが入る」と 1 文足す |
| LOW | maintainability | C2-4: `permissions.admin` を読まなくなったのに、テストの fixture がその値を渡し続け、値が効くように読める。`meClassicExec(admin bool)` の引数は、呼び出し 4 か所が `false`(table の `meBool(false)` と 345 行目)と `true`(397、440 行目)を渡すが、どちらでも結果は同じ。table の `classic *bool` と `meBool` は nil か否かを運ぶだけになった。CLI の fixture と `merge_test.go` の fixture も `admin:false` を送る。値を変えて確かめる意図があるのは `TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller` の 4 つの caller だけ | `internal/org/mergeenforce_test.go:94-96`(`meClassicExec`)、`:105`(`meBool`)、`:170`(`classic *bool`)、`:285`・`:345`・`:397`・`:440`、`:82`(`newMEExec` の doc の「the caller, no admin」)。`internal/cli/org_merge_test.go:130`(`"permissions":{"admin":false}`)。`internal/org/merge_test.go:174` | `meClassicExec` から `admin` を外し、table の `classic` は `bool` にして `meBool` を消す。`newMEExec` の doc から「no admin」を外す。CLI と `merge_test.go` の fixture は `meRepoJSON` の admin を省くか、残すなら「読まれないことを pin する」と 1 行書く |
| LOW | maintainability | C2-5: tech-debt 292 行目の Trigger は、S7 までは「merge のテストファイルの次の変更」だった。S8 がその次の変更になり、`merge_test.go` を 876 から 911、`merge_run_test.go` を 926 から 1006、`merge_wait_test.go` を 607 から 800 にしたが、fixture の移動は打たず、Trigger を「この計画のあとの次の変更(S8 はテストを足したが移動はしなかった)」に書き換えた。Trigger が起きるたびに書き換えると、行は動かない。`merge_wait_test.go` はちょうど 800 行で、次の 1 行で超える。code review の目安は 800 行未満 | `docs/tech-debt/README.md:292`(Debt、Trigger の両方に S8 の注)。`wc -l`(911、1006、800)と `wc -l` を b1cbe054 の各ファイルに打った値(876、926、607) | 移すのが機械的な作業(行に書いてある `merge_fake_test.go` への fake と fixture の移動)なら、cycle の中で打つ。打たないなら Trigger を「この PR が merge されたあとの最初の変更」のように、計画ではなく日付か出来事に結ぶ |
| LOW | readability | C2-6: 小さな読みにくさが 4 つある。(a) `merge_run.go` の先頭コメントに、整形の途中の短い行がある(「passed (D4). Its gh calls run」)。(b) `Merge` の doc の文「a merge --match-head-commit keeps this gh pr merge from making」は、関係詞(that)が抜けて読みにくい。(c) `merge_wait.go` の先頭と `merge_wait_test.go` の org の待ちのテストが、根拠を AC11(`/director` skill)と書くが、動詞の org の待ちを定めるのは AC13。(d) テスト名 `TestCheckMerge_OrgThatStillRunsIsWaitedFor` は、`checkMerge` が待つかのように読める(実際は待つ kind を返すだけ) | (a)`merge_run.go:22-25`(23 行目と 25 行目の間に 24 行目だけ短い)。(b)`merge_run.go:121-124`。(c)`merge_wait.go:15`、`merge_wait_test.go:504`、計画の AC11・AC13。(d)`merge_test.go` の該当テスト | (a)再整形。(b)「a merge that --match-head-commit keeps this gh pr merge from making」。(c)AC13 を足す(D5 と AC11 は残してよい)。(d)`..._OrgThatStillRunsIsOfAKindThatWaits` のように、kind を見るテストだと分かる名前にする。どれも動作は変わらない |

### Tech debt identified

直さない場合に載せる行の案(`docs/tech-debt/` にはこの report からは追記していない)。これが cap の最終 run なので、直さない LOW は 1 行にまとめて載せる。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C2-2: `ralph org merge --wait-checks` は、gate の記録だけを持ち結びつきのないブランチの org がほかの機能で走っているとき、それを見分けられず(`orgRunsOn` は `t.binding != nil` のときだけ立つ)、予算まで待ってから「leader がまだ push しうる」と断る | director が `--wait-checks 60m` で打つと、その間ほかの merge と regate を止めて 60 分待ち、断りの文も実情と違う。現行の writer では gate の記録の前に結びつきがあるので、台帳を手で直したときなどまれ | 見分けるには merge-order の分類(`orgRunsOn` の意味)を変える必要があり、このスライスの外。待つのは安全側 | `orgRunsOn` か merge-order の分類の次の変更、またはこの待ちが実際に起きたという報告 | 計画の D5・D13、この report の C2-2、`internal/org/mergeorder.go`(`collectMergeTargets`)、`internal/org/merge_wait.go`(`orgWaitEnd`) |
| C2-1・C2-3〜C2-6 を直さない場合: `regate.lock` の待ちのコメント 3 か所、`wait` の組み立て、`admin` の fixture、292 行目の Trigger、先頭コメントの整形と名前 | 読み違えるだけで動作は変わらない | どれもコメント、テスト、名前で、cap の最終 run では直す手間のほうが大きい | `merge_wait.go`、`mergeenforce_test.go`、`merge_test.go` の次の変更 | この report の C2-1、C2-3〜C2-6 |

_(上の変更は report だけで、`docs/tech-debt/` には追記していない。)_

## Cycle 3 re-review (S9・S10、31d27951..590214fb)

- Reviewer: reviewer サブエージェント(`/self-review`、パイプライン cycle 3。cap を 2 から 3 に上げた再実行で、`cycle-count.json` は 2 のまま)
- Scope: `git diff 31d27951..HEAD` から `docs/reports/`、`docs/insights/`、計画を除いたもの(HEAD 590214fb、25 ファイル、+1164/-827)。e5831d14(S9a)、7d0d08dd(S9b)、425a8543、56efae4a(sync-docs)、590214fb(S10)。cycle 2 までに見たコードで、この範囲が触っていない部分は読み直していない。diff の品質だけを見た。テスト、静的解析、`gh` は走らせていない

### Evidence reviewed

- 全文を読んだ: `internal/org/merge_wait.go`(291 行)、`internal/org/merge_readback_test.go`(122 行)、`merge_run.go` の `Merge`・`mergeLocked`・`mergeAndRecord`・`otherHeadMerged`・`otherBaseMerged`・`mergeReadBack`・`mergeInterruptedAfterStart`、`merge.go` の `mergeReady`・`checkMerge`・`checkMergeBranch`・`runningRefusal`・`mergeRefusalKind`・`waits`、`internal/cli/org_merge.go` の help と `printMergeResult`
- diff を読んだ: `lockfile.go`・`regate.go`・`spawn.go` のコメント、`mergeenforce_test.go`、`internal/cli/org_merge_test.go`、`merge_run_test.go`(S10 の 20 行)、`merge_fake_test.go` の S10 の差分、`merge_wait_test.go` の 425a8543、README、仕様の注、tech-debt の変わった 3 行(243、293、298)と HTML コメント(291)、skill の director と org の変わった箇所
- S9b の移動の確認: 7d0d08dd の `merge_test.go`・`merge_run_test.go`・`merge_wait_test.go` の削除行(648 行)と `merge_fake_test.go` の追加行(662 行)を sort して `comm` で比べた。片側にしかない行は、ファイル先頭の説明コメント、`package`・`import`、移ったあとの参照先を言い換えたコメント(「merge_fake_test.go」を指す 6 行)だけで、関数の本体に差はない。行の順は `comm` では見ていない。同じ commit の `TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` は移動ではなく新規のテスト
- 機械的な確認: `git diff --check` は空。追加行に TODO・FIXME・print・`/Users/`・`file.go:NNN` の行番号の引用はない。skill の director と org は 4 面とも `cmp` で一致。tech-debt の追加・変更した 3 行は 5 セルで、`|` の escape 漏れはない。`wc -l`: `merge.go` 737、`merge_run.go` 502、`merge_wait.go` 291、`merge_test.go` 626、`merge_run_test.go` 795、`merge_wait_test.go` 745、`merge_fake_test.go` 673、`merge_readback_test.go` 122、`mergeenforce_test.go` 480。どれも 800 未満
- 古い名前: `OrgThatStillRunsIsWaitedFor`、`meBool`、引数つきの `meClassicExec` は `.go` に残っていない。読み直しの古い field 列 `headRefOid,mergeCommit`(`baseRefName` なし)は、`.go`・skill・README・仕様・tech-debt に残っていない(残りは過去の report と計画だけ)

### 前回の finding の判定(C2-1〜C2-6)

- C2-1: 直っている。`lockfile.go`・`regate.go`・`spawn.go` の 3 か所は「検査を待つ間」から「`merge_wait.go` の待ちの間」に変わり、`merge_wait.go` の先頭に、3 か所の指す先はここ、と書いた。この新しい 1 文の言い方と、`lockfile.go` の整形が C3-6
- C2-2: 直っている(`runningRefusal`・`orgWaitEnd` の doc、help、README、skill 4 面、仕様の注、tech-debt 298 行目)。同じ事実を前提にしている `merge.go` の kind の定義 4 か所は、結びつきのないブランチの例外を書いていない(C3-4)
- C2-3: 直っている。`wait` は `switch f.kind` で `what`・`forWhat`・`first`・`read` を 1 か所で決め、`mergeReady` と `mergeWaitFor.r` の doc は「断った step までに読んだ field だけ」を書く。step 8 は branch、origin の field、`orgID`(`checkMergeBranch` の代入と合う)、step 10 は protection・method・methodFlag を除く全部(`checkMergeOnGitHub` と `checkMergeAfterChecks` の代入と合う)。`default:` が待つ kind のすべてを受ける点は残った(C3-5)
- C2-4: 直っている。`meClassicExec` の引数、`meBool`、table の `*bool` は消え、CLI と `merge_test.go` の fixture は `permissions` を送らない。`admin` を渡すのは `TestMergeEnforcement_ClassicNeedsEnforceAdminsForEveryCaller` の 2 caller だけで、`meRepoJSON` の doc の言うとおり
- C2-5: 直っている。fake と fixture が `merge_fake_test.go` に移り、3 つのテストファイルは 626・795・745 行。行は merge のテストの分だけ閉じ、残りは開いたまま。S10 のあとで行数が 1 つ古い(C3-7)
- C2-6: 4 つとも直っている。(a)は再整形、(b)は「a merge that」、(c)は AC13 を足した(`merge_wait.go` の先頭と `merge_wait_test.go`)、(d)は `TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits` に改名(参照する `merge_test.go` の 2 か所のコメントも追随)

### 依頼された点の確認(finding にならなかったもの)

1. S10 の比較: `back.BaseRefName != run.r.defaultBranch` の右辺は、`checkMergeBranch` が `remote.defaultBranch` から入れる値で、step 9(`viewPR`、`merge.go:551`)が `pr.BaseRefName != r.defaultBranch` で見るのと同じ field。step 9 は `BaseRefName == ""` も断るので、`defaultBranch` が空のまま「空と空が等しい」で記録に進む道はない。比較は完全一致で、step 9 と同じ
2. 優先順: `mergeAndRecord` の case は unknown、head 違い、base 違い、MERGED の順。head も base も違うときは head の文だけが出る(`merge_readback_test.go` の 2 行が固定)。文は `--json` に `baseRefName` を含めるので、人は base も確かめられる
3. 文: `otherBaseMerged` の 2 つの変種(別の base、base なし)は `Merged` を立てず、`failed()` が `PrintableInboxText` を通す(制御文字の行がテストにある)。`printMergeResult` は `res.Merged` のときだけ `MERGED:` を出すので、行は出ない。中断の経路は `mergeInterruptedAfterStart` が `res.Failed` を結果の文にする(`TestMerge_InterruptedThenMergedIntoAnotherBase`)
4. 文書の型: `MergeResult.Failed`・`Merged` の doc、`Merge` の doc の箇条書き、`mergeReadBack` の doc、`printMergeResult` の doc、help の記録の条件と終了コードの行、README は、head と base の両方を書いている。help と README の言い回しは同じ内容
5. `--json` の field 列: 利用者の文に出る 3 か所(unknown、別の head、別の base)は `state,headRefOid,baseRefName,mergeCommit` で揃っている。`autoMergeRequest` がないのは、確かめる人が見たい項目だけを並べた形で足りる。重複の手間は C3-2
6. S9b 以降の新しい 2 つの wait のテスト: 予算 70 秒、間隔 30 秒の割り当て(30 s、30 s、10 s)が期待値と合う。`TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` の「disbanded again」は、sleep 1 で disband と結びつきを足し、3 で disband を足して、検査の周回が `mxAll[:3]`、`mxAll`、merge、読み直しの順になる
7. 範囲外として扱った: merge-order の `classifyMergeTarget` が MERGED の PR を base を見ずに `merged` と分類する件(依頼の既知の件)

## Cycle 3 findings

CRITICAL 0 件、HIGH 0 件、MEDIUM 0 件、LOW 7 件。

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | C3-1: `/org` skill の `merge` の行(4 面)と仕様の注は、記録する条件を head だけで書き、S10 が足した base の条件がない。README と help は直っている。skill は「MERGED で、merge された head が gate の head なら、台帳に `pr_merged` を書き、… 終了コード 0。MERGED でも、merge された head が gate の head でないか読めなければ … 何も記録せず … 終了コード 1」と、記録と終了コード 0 を head だけで分けるので、gate の head のまま別の base に merge された PR は、記録されて終了コード 0 になると読める(実際は記録せず、`MERGED:` も出さず、終了コード 1)。director と leader はこの行で動詞の結果を読む。文書のずれは `/verify` の範囲だが、S10 が直前に偽にした文なので、依頼どおりここに載せる。`seat.go` の `EventPRMerged` の doc も、書く条件を「gh pr view read the PR back as MERGED」だけにしている(head の条件は S8b から書いていないので、新しいずれではない) | `.claude/skills/org/SKILL.md:189`(と `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` の同じ行)。`docs/specs/2026-10-07-org-multi-org-director.md:78`(「gate の head が merge されていたら台帳に org の記録 `pr_merged` を書く」と、head だけの例外の文)。`internal/org/seat.go:92-93`。比べる先は `README.md:266` と `internal/cli/org_merge.go:90-103`。`git show --stat 590214fb` に skill と仕様は入っていない | skill の 2 文を「head が gate の head で、base が既定のブランチなら記録する」「head が gate の head でないか読めない、または base が既定のブランチでないか読めなければ、記録せず、head か base を stderr に名指して終了コード 1」の形に直す。仕様の注も同じ 2 文。skill は `scripts/sync-skills.sh` と `templates/base` で 4 面を揃える。`seat.go` は「at the gate's head into the default branch」の 1 句を足す(任意) |
| LOW | readability | C3-2: `otherBaseMerged` は `otherHeadMerged` の末尾の `run.failed(fmt.Sprintf(...))`(「read PR #%d back as MERGED after %s %s, but %s, and nothing was recorded: check that merge yourself (%s --json ...), run ralph org merge-order again, and go on with the other branches」)をそのまま写し、違うのは `merged` の文だけ。確かめる field 列 `state,headRefOid,baseRefName,mergeCommit` は、unknown の文、`otherHeadMerged`、`otherBaseMerged` に手で 3 回書かれ、`mergeReadBackFields` が 4 つ目(`autoMergeRequest` つき)になる。S10 は 1 つの field のために、コード 3 か所とテスト 4 ファイルの field 列を書き換えた。次に field を足す人も同じ数を探し、1 か所を落とすと利用者に示すコマンドが古くなる | `internal/org/merge_run.go:348`(unknown の文)、`:411`(`otherHeadMerged`)、`:432`(`otherBaseMerged`)、`:37`(`mergeReadBackFields`)。テスト側は `merge_run_test.go`(5 行)、`merge_readback_test.go:18`、`internal/cli/org_merge_test.go`(4 行)、`merge_fake_test.go` | `func (run mergeRun) mergedElsewhere(merged string, a mergeAttempt) string` に末尾を寄せ、2 つの関数は `merged` の文だけを組み立てる。field 列は `const mergeViewFields = "state,headRefOid,baseRefName,mergeCommit"` のように 1 か所に置き、`mergeReadBackFields` は `mergeViewFields + ",autoMergeRequest"` と書く。テストは文字列を固定する役目なので、リテラルのまま残してよい |
| LOW | maintainability | C3-3: 新しいテストが、同じ commit で書き換えた隣のテストを使っていない。`mrNext` と `mrMergedAfter` は `merge_readback_test.go` の先頭に作られたが、`TestMerge_RecordsOnlyAMergeOfTheGatesHead` は S10 の差分でまさにその文字列(`next`)を書き換えたのに、同じ内容を手書きで持ち続け、表の 5 行が「org: merge: … read PR #7 back as MERGED after … failed, 」を毎回手で組む。2 つのテストは table の形も、ループの本体(`newMRFixture`、`res != want`、`appended`、`mrAssertLockFree`)も同じ。また wait の新しい 2 つのテストは、`before := mrRead(...)` と `strings.TrimPrefix(...)` の 3 行を手で書いているが、同じ働きの `addDuring` が `merge_fake_test.go` にある(「disbanded again」は sleep が 2 回、`ChecksThenOrg` は同じ hook で `setChecks` も打つので、今の `addDuring` では書けない) | `merge_readback_test.go:18-25`(`mrNext`、`mrMergedAfter`)、`merge_run_test.go:238-239` と `:345`(同じ文字列の手書き)、`:243-258`(`mrMergedAfter` に当たる頭)、`merge_wait_test.go:486-503` と `:563-575`、`merge_fake_test.go:663`(`addDuring`) | `mrNext` と `mrMergedAfter` を `merge_fake_test.go` の `mrViewCmd` の隣に移し、`TestMerge_RecordsOnlyAMergeOfTheGatesHead` と `:345` の行も使う。`addDuring` は sleep の番号ごとの lines と、同じ sleep で打つ関数を受けられる形にする。動作は変わらない |
| LOW | maintainability | C3-4: S9a は `runningRefusal` と `orgWaitEnd` に「結びつきのないブランチでは、ほかの機能で走る org も `mergeRefusalOrgRunning` になる(`orgRunsOn` は結びつきがあるときだけ立つ)」と書いたが、同じ事実を前提に「そのブランチの機能で走る org だけ」と書いた `merge.go` の定義 4 か所は直していない。kind の定義を読む人は、待つ kind はそのブランチの org だけだと信じる。C2-2 の直しが、断言している別の場所に届いていない形(cycle 2 の C2-1 と同じ) | `merge.go:595-596`(`mergeRefusalOrgRunning` の「and not another feature now」)、`:612-614`(`waits` の「only an org that still runs the branch's feature」)、`:586`(`mergeRefusalKind` の doc)、`:203-204`(`checkMerge` の doc)。比べる先は `merge.go:485-496`(`runningRefusal`)と `merge_wait.go:240-249`(`orgWaitEnd`)。`git diff 31d27951 HEAD -- internal/org/merge.go` は `mergeReady` と `runningRefusal` の 2 か所だけ | 4 か所に「ledger に結びつきのあるブランチでは」を足すか、`runningRefusal` の doc を指す 1 文にする |
| LOW | maintainability | C3-5: C2-3 の直しは、`wait` の組み立てを `switch f.kind` に集めたが、待つ kind が 2 つ目の case ではなく `default:` に入る。`waits()` に 3 つ目の kind が足されると、その kind は `default:` に落ち、`f.r.prNumber`(PR のない kind では 0)から「PR #0 has checks pending」の行を作る。C2-3 が書いた「3 つ目の kind が待つ側に足されると PR #0 の文がそのまま出る」は、形を変えて残る。`default:` の中のコメントは 2 つの kind を名指すが、何も確かめない | `internal/org/merge_wait.go:171-183`(`default:` と `f.kind == mergeRefusalChecksZero` の分岐)、`internal/org/merge.go:616-618`(`waits`) | `case mergeRefusalChecksPending, mergeRefusalChecksZero:` にして、`default:` は「この kind の待ち方は未定義」の error を返す(今は届かない)。または `waits()` が true の kind すべてを回して、`wait` の最初の行に「PR #0」が出ないことを見るテストを足す |
| LOW | readability | C3-6: 小さな読みにくさが 4 つある。(a) `merge_wait.go` の先頭に足した「A Merge waits only in this file, and lets go of regate.lock before it returns only while it waits here」は、Merge が regate.lock を `regateLockWait`(1 秒)待つこと(help も「it waits up to 1 s for regate.lock」と書く)と、manifest のロックを待つ `recordMerge` に合わず、「before it returns」は `mergeLocked` が返るたびに外す実装では全部に当たる。言いたいのは「ロックを途中で外すのはこのファイルの待ちだけ」。(b) `lockfile.go` の直した文に、短い行(「takes it again for the checks after the wait. The lock」)が残り、整形の途中の形。(c) help の `ralph org merge --help` に、短い行(59 桁。前後は 68〜72 桁)が出る。(d) `TestMerge_WaitChecksForTheOrgGoesOnWhenItStartsAgain` の doc は、主語と動詞が離れた 1 文で、読み取りにくい | (a)`merge_wait.go:37-40`、`regate.go:99-102`、`internal/cli/org_merge.go:56`。(b)`lockfile.go:136-139`。(c)`internal/cli/org_merge.go:130`(「refusal are not waited for. When the time is up and the org」)。(d)`merge_wait_test.go:477-483` | (a)「Merge が regate.lock を途中で外すのは、このファイルの待ちの間だけ」に直す。(b)(c)再整形。(d)「org が disband されて同じ org_id で同じ機能に起こし直されると、次の読みでその org は走っているので、待ちは続く」のように、主語を `org` に立てて 2 文に分ける。どれも動作は変わらない |
| LOW | maintainability | C3-7: tech-debt 291 行目の HTML コメント(merge のテストファイルの行の閉じ)は `merge_fake_test.go (662 lines)` と書くが、S10(590214fb)が `mrReadBackInto` を足して 673 行になった。コメントは S9b の記録なので、S10 の分だけ数が古くなる(shape: 後続の commit が行の数を動かす)。`merge_run_test.go` は 795 行で、800 行まで 5 行しかなく、行の Trigger はそこが次の追加で超えることを書かない(S10 は読み直しのテストを新しいファイル `merge_readback_test.go` に置いた)。コメントは 3 つのファイルを名指し、新しいこのファイルは載っていない | `docs/tech-debt/README.md:291`(`(662 lines)`、`(626)`、`(795)`、`(745)`)。`wc -l internal/org/merge_fake_test.go` は 673。`git show --stat 590214fb` に `docs/tech-debt/README.md` は入っていない | `(662 lines)` を行数なしの書き方にするか 673 にする。「merge のテストのファイル 5 つ(`merge_readback_test.go` を含む)は 800 行未満で、`merge_run_test.go` は 795」と書き、次にテストを足すなら新しいファイルに置く、と 1 文足す |

### Tech debt identified

これが cap の最終 run なので、直さない LOW は 1 行にまとめて載せる。C3-1 は利用者(director と leader の agent)が読む手順の記述なので、直す前提にし、直さないなら別の 1 行を足す。`docs/tech-debt/` にはこの report からは追記していない。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| C3-2〜C3-7 を直さない場合: `otherBaseMerged` と `otherHeadMerged` の末尾の写しと field 列の手書き 3 回、`mrNext`・`addDuring` の重複、`merge.go` の kind の doc 4 か所、`wait` の `default:`、小さな読みにくさ、291 行目の行数 | 読み違えるだけで動作は変わらない。read-back の field を足すたびに、コード 3 か所とテスト 4 ファイルを探す | どれもコメント、テスト、名前、小さな関数の切り出しで、cap の最終 run では直す手間のほうが大きい | `merge_run.go` の read-back か `waits()` の次の変更 | この report の C3-2〜C3-7 |
| C3-1 を直さない場合: `/org` skill の `merge` の行(4 面)と仕様の注が、記録の条件を head だけで書く | director と leader が、gate の head のまま別の base に merge された PR を、記録されて終了コード 0 になると読む。動詞は記録せず終了コード 1 にするので、安全側にずれる | skill と仕様は record だけの修正の例外に当たらず、cap の最終 run のあとに直すと全 pipeline の再実行になる | `merge` の行か、`otherBaseMerged` の次の変更 | この report の C3-1、`.claude/skills/org/SKILL.md` |

_(上の変更は report だけで、`docs/tech-debt/` には追記していない。)_

## Recommendation

- Merge: yes(cycle 3: CRITICAL、HIGH、MEDIUM はない。LOW 7 件)。S9a から S10 まではコードとして正しい。C2-1 から C2-6 は 6 件とも直っている。S10 は `baseRefName` を step 9 と同じ field と同じ完全一致で比べ、head の不一致を優先し、base の違う MERGED と base のない MERGED を `Failed` にして記録も `MERGED:` の行も出さない。新しい finding は文書とコメントの追随、複製、テストの重複で、動作を変えるものはない。C3-1(skill 4 面と仕様の注が記録の条件を head だけで書く)は、`/pr` の前に直すのが望ましい。ほかは任意で、直さないなら上の「Tech debt identified」の案を 1 行にして載せる
- Follow-ups(cycle 3 のあと):
  - C3-1 は skill の 1 行(4 面)と仕様の注の 2 文。直すなら skill と仕様を触るので、記録だけの修正の例外には当たらず、全 pipeline の再実行になる。cap(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` は 3 に上げた)の最終 run なので、直して回し直すか、known gap として `/pr` に載せるかは `/cross-review` の判断
  - C3-2 は小さな関数の切り出しと const。C3-3 から C3-7 はコメント、テスト、名前、register の数字で、まとめて 1 commit にできる
  - insight event(`scripts/insights-append.sh`)は、report 以外を編集しない指示のため書いていない。cycle 3 の値は `--verdict pass --critical 0 --high 0 --medium 0 --low 7`
- 経過(cycle 2 のレビューの結論。当時の Merge の行): CRITICAL、HIGH、MEDIUM はない。LOW 6 件(いまは C2-1 から C2-6 とも直っている)。S8a・S8b・S8c はコードとして正しい。`enforceClassic` は admin の値を読まず `enforce_admins` だけで決め、`mergeAndRecord` は MERGED でも head が違う(空を含む)と記録せず `Merged` も立てず、org の待ちは台帳だけをロックなしで読んで `checkMerge` と同じ読み方で終わる。新しい finding はどれも動作を変えない。C2-1(コメント 3 か所)と C2-2(tech-debt の行)は、`/verify` の前か `/pr` の前に 1 コミットでできる。ほかは任意で、直さないなら上の「Tech debt identified」の案を 1 行にして載せる
- Follow-ups(cycle 2 のあと):
  - C2-1 は 3 ファイルのコメントの 1 文ずつ。C2-2 は tech-debt に 1 行。C2-4 は fixture から引数を 1 つ外す。直すなら 1 コミットにまとめられる
  - cap(`RALPH_STANDARD_MAX_PIPELINE_CYCLES` の既定 2)の最終 run なので、直したあとに全 pipeline を回し直すかは `/cross-review` の判断。コメントとテストの名前だけの直しなら、`post-implementation-pipeline.md` の「記録だけの修正」の例外には当たらない(コードのファイルを触る)ので、回すなら全部になる
  - insight event(`scripts/insights-append.sh`)は、report 以外を編集しない指示のため書いていない。cycle 2 の値は `--verdict pass --critical 0 --high 0 --medium 0 --low 6`
- Follow-ups(S6・S7 の再レビューのあと。どれも 97a1a062 で済み):
  - 済: R1 は 3 ファイルのコメント 1 語ずつ、R2 は 1 行とテスト 1 件。R3 は実時間のテストの下限、R4 は 5 つの読みにくさ(cycle 2 の読みで 4 件とも直っていることを確かめた。R1 の新しい文は C2-1 で足りなくなった)
  - 済: insight event は report 以外を編集しない指示のため書いていない。再レビューの値は `--verdict pass --critical 0 --high 0 --medium 0 --low 4`
- Follow-ups(最初のレビューのあと):
  - 以下は最初のレビュー(再レビューの前)の Follow-ups で、どれも S6・S7 で済んだ(各行の頭の「済」)
  - 済: M2 は skill の文言の決め(待つか、止まるか)で、`/verify` が AC11 をどう読むかにもかかる。先に決めると pin テストと skill の 4 面をまとめて 1 回で直せる
  - 済: L1、L3、L5、L6 はコードの小さな直しで、まとめて 1 コミットにできる。L2、L4、L7、L8 は注と文言。`/self-review` の再実行は不要な範囲だが、コードを触るなら cycle 2 の対象になる
  - 済: tech-debt の 239 行目のセル 4 にある `regateRemoteGit` の置き場所(`internal/org/regate.go`。今は `regate_git.go`)は、base の `15d22b1a` にもある古い記述で、この PR の追加ではない。計画の Progress が `sync-docs` に回している項目と同じなので、そこで直す(L7 でセル 2 と 3 を書き直すときに一緒に直せる)
  - 済: insight event(`scripts/insights-append.sh`)は、report 以外を編集しない指示のため書いていない。値は `--verdict pass --critical 0 --high 0 --medium 2 --low 8`(53bef7e1 で記録された)
