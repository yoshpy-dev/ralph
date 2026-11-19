# Self-review report: doctor-alias-herdr-guard

- Date: 2026-11-19(cycle 1。`cycle-count.json` は `cycle: 1`)
- Plan: `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`
- Branch: `fix/doctor-alias-herdr-guard`、HEAD `9e631cc1`、base は `main` の `a8c5e289`
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff a8c5e289...HEAD` の 12 ファイル(+1630 −76)の diff の質だけ。コードは S1 `faef9b12`(scanner、`shellAliasGuardTracker`)と S2 `1af2fcee`(`checkShellAliases`、Detail)。文書は S3 `754fd9a3`(recipe 2 つ、org skill の 4 写し)。計画書と記録の commit(`53c662b1`、`248e8a1b`、`b6359470`、`9e631cc1`)は、Progress の説明が差分と合うかだけ見た。テスト、静的解析、spec 適合、文書の drift は見ていない(`/verify` と `/test` の仕事)

## Evidence reviewed

- コード: `internal/cli/doctor_shell_alias.go` の全体(トークナイザ `shellAliasStatementsWithOps`、`shellAliasGuardTest`、`shellAliasGuardTracker` の全メソッド、`scanShellAliasFile`、`shellAliasRemedyClause`、`shellAliasPaneGuardedSentence`、`checkShellAliases` の分類と severity の `switch`)、`internal/cli/doctor.go` の comment
- テスト(読んだだけで、リポジトリのテストは走らせていない): `internal/cli/doctor_shell_alias_test.go` の追加分の見出しと、scanner 階層の 3 つの表(`MarksAliases…`、`LeavesAliasesAPaneCanDefineUnmarked`、`IfKeywordOutOfOrder…`)
- 文書: recipe 2 つと org skill(`.claude` 側)の差分。4 写しの同一性は `/verify` の `check-sync` に任せた
- register: `docs/tech-debt/README.md` の 136 行目(`checkShellAliases` の項)。この差分は `docs/tech-debt/` に触れていない
- 前の版との比較: `git show a8c5e289:internal/cli/doctor_shell_alias.go`(968 行)

## 依頼された点: 本当は pane で効く alias を info にする向き

方法。`doctor_shell_alias.go` を scratch の別 module(`package main` と `checkResult` の stub だけ足した写し、リポジトリの外)にコピーして build し、86 本の rc fixture を `scanShellAliasFile` に読ませた。同じ fixture を `HERDR_PANE_ID=wD:p1 zsh -f -c 'source <rc>; alias -L'`(zsh 5.9)で読み込み、pane で実際に定義される名前と、scanner の `PaneGuarded` を def ごとに突き合わせた。fixture は 2 組。1 組目 57 本は計画の AC2 と AC3 の形と、その近くの崩し方(前の行の `&&` と `\`、`|`、`&`、引用符、`{ }`、if の入れ子と `elif`、条件が 2 文の if、余分な `fi`、一行 if など)。2 組目 29 本は `!` や `if`、`while`、`time` だけの行のあとの判定、CRLF、tab、演算子の前後に空白のない形、`;;` のあと、判定のあとの `||` の連なり、入れ子の if の `else` の側など。

| 結果 | 数 | 内容 |
| --- | --- | --- |
| `PaneGuarded=true` かつ pane で定義される(印が間違い) | 4 本、3 通りの形 | 下の L1(`"[["` と引用符を付けた判定の語)、L2(rc が判定の前に `HERDR_PANE_ID` を書き換える、heredoc の中の `if`/`fi`)。どれも実在しうる形ではあるが、書く人は少ない。L2 の 2 つは計画の Non-goals と Risks に載っている |
| `PaneGuarded=true` で、pane でも定義されない(正しい印) | 多数 | 判定の 2 つの向き、`then`/`else` の側、`{ }` の頭、`;`・`&` で閉じた前の行、入れ子の if、`-z` の then の中の別の if。`[[ -n $H ]] \|\| alias codex=… && alias claude=…` は codex だけ印が付き、claude は付かない(zsh でも claude だけ pane で定義される)。`-z … && alias codex=… \|\| alias claude=…` も同じ |
| `PaneGuarded=false` で pane で定義される(正しい未印) | AC3 の反例すべて | 前の行の `&&`・`\|\|`・`\|`・行末の `\`、並びの頭にない判定、`-z` の if の else、`-n` の if の then、一行 if の `fi` のあとの alias、`fi \|\| alias`、`foo \` の次の行の `{ test -n … \|\| alias …` |
| `PaneGuarded=false` だが zsh でも pane で定義されない(保守的な読み) | 十数本 | パイプの右の判定、`\|\| false` を足した条件の if、`while`/`until` の中、`( … )` の中、`${HERDR_PANE_ID:-}`、リダイレクト付きの判定。どれも計画の Non-goals か、黙る向きを避けて印を付けない側 |

`!` だけの行の次の行の判定(`d01`)は、zsh が次の行を普通に実行し、pane では alias を定義しない(rc 全体を読ませた結果)ので、印は正しい。

`checkShellAliases` の側は、印のある def が `!d.Incomplete` のときだけ info の文に回り、codex の warn と claude の `--permission-mode` の warn の判定(`codexFound`、`claudeHasPermissionMode`)は印のない def だけから作られる。印のある def が、同じ rc の印のない def の warn を消す道は見つからなかった(AC4 の形)。

## Findings

この run の指摘は M1 と L1 から振る。CRITICAL と HIGH はない。pane で効く alias を黙らせる向きの誤りは、L1 と L2 の 4 本だけで、どれも普通の rc には出ない。

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | `docs/tech-debt/README.md` 136 行目の trigger「`checkShellAliases` の severity `switch` の次の変更」に、この差分が当たるのに、行が更新されていない。差分は `switch` に `paneGuardedFound` を足し、判定の種類を 1 つ増やしている。前の 2 回(#167 と 2026-11-05 の role-models)は、trigger に当たったと日付つきの carry-over の文を足し、2026-11-05 の分はファイルの行数(953 → 968)も書いた。今回は 968 → 1436 行(+468、`doctor_shell_alias.go`)で、800 行の目安の 1.8 倍になる。S3 の注記が「Known gaps に書く」と言う L1・L2 の穴も、register にはなく、PR 本文に書く予定だけが残っている(PR 本文は後から探しにくい) | `docs/tech-debt/README.md:136`(`Remaining trigger: the next change to checkShellAliases's severity switch …`、続けて `2026-11-05: … the file went from 953 to 968 lines`)。`git diff a8c5e289...HEAD --stat` に `docs/tech-debt/` は出ない。`doctor_shell_alias.go:1428-1431`(`case … \|\| paneGuardedFound \|\| …`)。`wc -l`: base 968、HEAD 1436 | 136 行目に、この PR が trigger に当たったこと、債務(config を読まない grading)は払っていないこと(`PaneGuarded` は config と無関係な 3 つ目の分類で、債務の外)、行数 968 → 1436 を書き足す。L1 と L2 の穴を同じ行か別の行に 1 つ足す。`archive-plan.sh` が書き換えるのは register の plan へのリンクだけなので、plan の Known gaps に残したままだと /pr のあと探す先がなくなる |
| L1 | LOW | security | 引用符を付けた判定の語(`"[["`、`\[\[`)でも `PaneGuarded` になるが、zsh は `command not found: [[` で、alias を pane でも定義する。トークナイザは引用符を落とした語だけを返し、`shellAliasGuardTest` は `words[0] == "[["` で見るため、引用符の有無が見えない。`"test" -n …` は組み込みの `test` として走るので問題ない(`c31` で確認)。実在する書き方では `"[["` は出ないので、黙る向きの誤りとしては小さい | scratch の probe。fixture `"[[" -n $HERDR_PANE_ID "]]" \|\| alias codex="codex -m x"` に対し、`HERDR_PANE_ID=wD:p1 zsh -f -c 'source rc; alias -L'` が `zhome/.zshrc:1: command not found: [[` のあと `alias codex='codex -m x'` を出す。同じ fixture を `checkShellAliases(herdr あり)` に通すと `info … not defined in herdr's panes … so no seat … gets its flags`。S3 の Progress に「引用符付きの `"[["` もガードと読む」と残した穴と同じもの。`shellAliasGuardTest`(`doctor_shell_alias.go:497`)の doc は「after quote removal」と書くだけで、この例外は書いていない | (a) トークナイザが語ごとに「引用符も `\` も使っていない」を返し(`shellAliasStatement` に `Plain []bool` を足す)、`shellAliasGuardTest` が `[[`、`[`、`test`、`-n`、`-z`、`]]`、`]` の語に限って plain を求める(値の語は引用符を許す)。(b) 直さないなら、L2 の comment と M1 の register に書く |
| L2 | LOW | readability | `shellAliasGuardTracker` の doc は「When in doubt it does not mark: a mark says the alias is not defined in a pane, so a wrong one would hide an alias that does break seats」と言う。確かめた誤った印は L1 のほかに 2 通りあり、どちらも comment に書いていない。(1) rc が判定の前に `HERDR_PANE_ID` を空にする(`HERDR_PANE_ID=` の代入、`unset HERDR_PANE_ID`)と、pane で判定が偽になり alias が定義される。(2) heredoc の中の `if`/`fi` は if の入れ子に数えられ、`fi` の数が合えば誤った側に印が付く。(1) は計画の Non-goals、(2) は Risks にあり、決めたこと自体は妥当。ただ、計画は /pr で archive に動き、recipe と skill の列挙は「ほかの形は条件のない alias と同じ」と言うだけで、「読み違える形」は書いていない | probe。`HERDR_PANE_ID=\n[[ -n $HERDR_PANE_ID ]] \|\| alias codex="codex -m x"` と、`unset HERDR_PANE_ID` のあとの `[[ -z $HERDR_PANE_ID ]] && alias …` は、どちらも zsh が `alias codex=…` を pane で定義し、scanner は印を付ける(`inline=true`)。`cat <<EOF`/`if [[ -z $HERDR_PANE_ID ]]; then`/`EOF` の次の行の alias、その後の `cat <<EOF`/`fi`/`EOF` の fixture は、zsh は定義、scanner は `byIf=true`。`doctor_shell_alias.go:569-571`(tracker の doc)、`:497` の手前の doc | `shellAliasGuardTracker` の doc の「When in doubt …」の次に、読み違える 3 つ(rc が `HERDR_PANE_ID` を書き換える、heredoc や複数行の文字列の中の `if`/`fi`、引用符付きの判定の語)を書く。「must not mark when in doubt」と「ここは読み違える」を同じ comment に並べる |
| L3 | LOW | maintainability | `doctor_shell_alias.go` は 968 → 1436 行になり、足した約 470 行のうち約 335 行(`shellAliasGuard` 型から `inlineGuarded` まで)は、rc の走査や severity と別の、自己完結した部品(if の入れ子と並びの頭の判定)。依存は `shellAliasStatementsWithOps`、`shellAliasAliasArgs`、`parseAliasWords`、`shellAliasStatementMarkers` の 4 つだけ。テストも 1718 → 2572 行(+854)で同じ形。800 行の目安を、すでに超えていたファイルがさらに伸びた | `wc -l`: base 968、HEAD 1436。`inlineGuarded` の終わりが 803 行目で、次の `shellAliasDef` から先は走査と severity。test は `git diff --stat` で +854 | `doctor_shell_alias_guard.go` と `_test.go` に切り出す(同じ package、名前はそのまま)。M1 の register に「切り出すなら次の変更で」と書いてもよい |
| L4 | LOW | maintainability | `scanShellAliasFile` は `assignments[i]`(`parseAliasStatements(joined)`)と `marks[i]`(`guards.line(joined)`)を添字でそろえる。後者は同じ行をもう一度トークナイズし、`parseAliasWords(args)` を数えるためだけに呼び直す。2 つが同じ並びで進むのは、共有の `shellAliasAliasArgs` と `parseAliasWords` があるからで、そのことは comment に書いてある。`if i < len(marks)` の bound check が守るのは `marks` が短いときだけで、`marks` が途中で 1 つ多くなると以後の def が別の def の印を受け取る(黙る向き)。いまは 2 つが同じなので起きないが、どちらかの解析を変える次の PR で、テストの表にない形から崩れる。近い点として、`shellAliasListHeadMarkers`(4 語)は `shellAliasStatementMarkers`(5 語)から `builtin` を引いただけで、2 つの表が並んでいる | `doctor_shell_alias.go:935-942`(bound check)、`:605-621`(`line` が `parseAliasWords` を数えるためだけに呼ぶ)、`:526`(`shellAliasListHeadMarkers`)と `:214-220`(`shellAliasStatementMarkers`)。テストに `len(marks) == len(assignments)` を直接確かめるものは見つからなかった | `parseAliasStatements` を、statement ごとの結果(def の数と、その statement の index)も返す形にして、tracker の印を statement に付ける。そこまでしないなら、`scanShellAliasFile` で `len(marks) != len(assignments)` のとき印を全部落とす(黙る向きを避ける)。2 つのマーカー表は、一方からもう一方を作る |
| L5 | LOW | readability | 案内の末尾(`shellAliasRemedyClause`)は、codex の warn と claude の `--permission-mode` の warn のすべてに付き、`[[ -n $HERDR_PANE_ID ]] \|\| alias codex=...` を勧める。fish の `config.fish` にある alias の warn にも付くが、fish に `[[` はなく、計画は fish を読まない(`TestScanShellAliasFile_FishFileIsNotReadForGuards`)ので、勧めたとおりに直しても warn は消えない。sentence 組み立ては file を知らない | scratch の probe。`~/.config/fish/config.fish` に `alias codex 'codex -m x'` だけの HOME で `checkShellAliases(herdr あり)` が `warn … or define it only outside herdr's panes with [[ -n $HERDR_PANE_ID ]] \|\| alias codex=..., one line per alias`。`doctor_shell_alias.go:1121-1124` | fish の def だけのとき案内の 3 つ目を省くか、「(bash と zsh の rc で。fish は読みません)」と足す。items にファイル名が入っているので、`.fish` の def があるかを sentence の引数に足せば済む |
| L6 | LOW | maintainability | 4 つの comment が `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md` を指す。/pr がこの plan を archive に動かすと、`archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` のリンクだけなので、comment の path は存在しなくなる。リポジトリには active を指す形(`internal/org/merge.go:19`)と archive を指す形(`internal/config/config.go:282`)の両方があり、直さなくても通るので LOW | `doctor_shell_alias.go:472`、`:568`、`:888`、`:1286`。`scripts/archive-plan.sh:54-65` | issue 番号(`#343`)だけに言い換えるか、archive の path を書く |

## 読んだが指摘にしないもの

- null 安全。`inlineGuarded` の `cur.Words[0]` と `rest[1:]`(`followIf`)は、statement が空でない語列のときだけ作られるので範囲外にならない。`shellAliasGuardTest` は語数を先に確かめる。`marks` の添字は bound check がある
- 例外の扱い。`scanShellAliasFile` の新しい経路にエラーを返す呼び出しはなく、`scanner.Err()` は今までどおり返る。`guards.broken` と `len(guards.frames) > 0` は、if の対応が崩れたファイルの if 判定を取り消す側(印を外す向き)に倒す。inline の印は残るが、if の入れ子とは独立に決まる(probe で確認。if の崩れ方のうち印が間違ったのは、`fi` の数が合ってしまう heredoc の `c53` だけ)
- 文の継続。`continuesPrevLine` と `prevLineEndedClosed` は、前の物理行の終わり(未閉じの引用符、行末の `\`、`&&`・`||`・`|`・`|&` の続き)が次の行の判定を並びの頭にしない向きで持ち越す。空行とコメント行が状態を変えないのも、zsh の読み方と合う(`false &&` の次が空行とコメントを挟んだ判定でも alias は pane で定義される、`d13`)
- `Incomplete` の def は、印があっても info に回さず warn の側に残す(値を読み切れていないため)。`shellAliasPaneGuardedSentence` に alias の値は入らない(`TestCheckShellAliases_PaneGuardedAliasValueNeverAppearsInDetail` の対象)
- 文字列の書き方。Detail の em dash は既存の文(`could not resolve home directory: … — shell alias check skipped`)と同じ。`shellAliasDetailSentence` が文を `". "` で分ける前提は、新しい sentence の中に `". "` を含まないこと(`0.7.5` は `". "` ではない)で成り立つ
- Detail と recipe と skill の「herdr 0.7.5 で確かめた」は、計画の Design decisions 4 で決めた書き方。文書の drift は `/sync-docs` の側
- 文書の例。recipe の `alias codex="codex -m ..."` と skill の `alias codex=…` は、doctor が読む形(`[[ -n $HERDR_PANE_ID ]] || alias …`、1 行に 1 つ)と同じ

## Positive notes

- 両方の読み手(`parseAliasStatements`、`shellAliasGuardTracker.line`)が `shellAliasAliasArgs` を共有するので、alias の statement の判定は 1 か所にある。`shellAliasStatements` は包みとして残り、既存の呼び出しとテストの結果を変えない
- 迷う形を、すべて印を付けない側に倒している。`!`、`-v`、`${…:-}`、別の変数、`{ }` の中、`elif`、条件が 2 文の if、前の物理行から続く文、if の対応の崩れ、fish。86 本の probe で、実在しうる rc の形に黙る向きの穴はなかった
- テストの表が AC3 の反例を 1 本ずつ名前つきで持ち、`TestScanShellAliasFile_UnclosedIfAtEOFDropsIfMarksButKeepsInlineMarks` などで EOF の取り消しの 2 層(if の印は落とし、inline の印は残す)を確かめている
- `checkShellAliases` の severity は、warn の条件(`codexFound`、`claudeHasPermissionMode`)に印のある def を入れず、info の条件にだけ `paneGuardedFound` を足している。warn の文は印のない finding だけを並べる

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `PaneGuarded` が誤って付く 3 つの形(引用符を付けた判定の語、rc が判定の前に `HERDR_PANE_ID` を書き換える形、heredoc や複数行の文字列の中の `if`/`fi`)。M1 と同じ行か別の行に書く | 該当する rc では、pane で定義される alias が info になり、warn が出ない | 計画の Non-goals と Risks で受け入れた。引用符の語は Progress の S3 の注記にある | 次に `shellAliasGuardTest` か `shellAliasGuardTracker` を変えるとき。または、この形の rc で warn が出ないという報告 | `docs/plans/active/2026-11-19-doctor-alias-herdr-guard.md`(Non-goals、Risks)、この報告の L1 と L2 |
| `doctor_shell_alias.go` が 1436 行(base 968)、同 test が 2572 行。`shellAliasGuardTracker` を別 file に切り出せる(L3) | 800 行の目安を大きく超える | この PR の scope は判定の追加で、切り出しは別の変更 | 次にこの file を変えるとき(M1 の行の trigger と同じ) | この報告の L3 |

_(上の 2 行は `docs/tech-debt/README.md` に足していない。この commit は報告と insight event だけで、register は M1 の直しの一部として、実装の側が 136 行目と合わせて直す。)_

## Recommendation

- Merge: yes(CRITICAL、HIGH はなし。MEDIUM 1 件(M1、register の更新)、LOW 6 件(L1 から L6))
- Follow-ups:
  - M1 は文書だけの直し(register の 136 行目と、上の 2 行)。plan の Known gaps に頼らず、register に載せる
  - L1 を直すなら、トークナイザに「引用符なし」の情報を足す小さなコード変更になる(L1 の (a))。直さずに進めるなら、L2 の comment と M1 の register に書く
  - L2、L3、L4、L5、L6 は comment と構造の直し。L4 の「`len(marks) != len(assignments)` なら印を全部落とす」は 3 行で足せ、黙る向きの将来の崩れを止める
  - どれも `/verify` を走らせる前なら、パイプラインの再走を足さずに直せる。直さずに進めるなら、PR 本文の Known gaps に書く(M1 の register の行は、PR に含めて直すほうがよい)
  - insight event の件数は CRITICAL 0、HIGH 0、MEDIUM 1、LOW 6
