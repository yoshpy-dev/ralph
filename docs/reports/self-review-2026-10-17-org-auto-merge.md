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

## Recommendation

- Merge: yes(CRITICAL と HIGH はない)。MEDIUM の M1 と M2 は、`/verify` の前に直すか、直さない理由を tech-debt に 1 行ずつ残す
- Follow-ups:
  - M2 は skill の文言の決め(待つか、止まるか)で、`/verify` が AC11 をどう読むかにもかかる。先に決めると pin テストと skill の 4 面をまとめて 1 回で直せる
  - L1、L3、L5、L6 はコードの小さな直しで、まとめて 1 コミットにできる。L2、L4、L7、L8 は注と文言。`/self-review` の再実行は不要な範囲だが、コードを触るなら cycle 2 の対象になる
  - tech-debt の 239 行目のセル 4 にある `regateRemoteGit` の置き場所(`internal/org/regate.go`。今は `regate_git.go`)は、base の `15d22b1a` にもある古い記述で、この PR の追加ではない。計画の Progress が `sync-docs` に回している項目と同じなので、そこで直す(L7 でセル 2 と 3 を書き直すときに一緒に直せる)
  - insight event(`scripts/insights-append.sh`)は、report 以外を編集しない指示のため書いていない。値は `--verdict pass --critical 0 --high 0 --medium 2 --low 8`
