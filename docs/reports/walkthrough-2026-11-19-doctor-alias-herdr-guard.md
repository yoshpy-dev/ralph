# Walkthrough: herdr の pane では定義されない alias を、doctor の Shell aliases が info にする(doctor-alias-herdr-guard)

- 計画: `docs/plans/archive/2026-11-19-doctor-alias-herdr-guard.md`(PR の作成のあとに archive へ移す)
- issue: #343
- 差分: `git diff main...HEAD`(この walkthrough を足す前で 19 ファイル、+2,106 / −79、base は main の 09d6b894。branch には origin/main を 7bbd6912 で取り込んである)。Go の本体は `internal/cli/doctor_shell_alias.go` が +530 / −50、`internal/cli/doctor.go` がコメント 2 行。Go のテストは `internal/cli/doctor_shell_alias_test.go` が +859。recipe 2 つと org skill の 4 写しが各 +7〜+19、tech-debt が 3 行。残りは report、計画、insight
- パイプライン: 1 cycle。verify は 1 回目で静的検査の 1 項目だけが落ち、直して 2 回目で pass。cross-review(codex)の指摘は 0 件

## 1. 文の区切りを残す(`shellAliasStatementsWithOps`、`doctor_shell_alias.go:283`)

- 今までの `shellAliasStatements` は、`;`、`|`、`&` で文を分けると区切りの種類を捨てていた。新しい関数は、文ごとに前の区切り(`""`、`;`、`&&`、`||`、`|`、`&` など)と、行の終わりに残った区切りを返す。`shellAliasStatements` はこれを包んで語だけを返すので、今の呼び手の結果は変わらない(`TestShellAliasStatements` と `TestShellAliasValueTokens` は変えていない)
- alias の文の語を取り出す部分は `shellAliasAliasArgs`(`:417`)にまとめた。`parseAliasStatements` と下の tracker が同じ関数を使うので、def と印の並びがずれない(self-review の L4。今は崩れないが、どちらかの解析を変える次の PR で注意する)

## 2. 判定の分類(`shellAliasGuardTest`、`:499`)

- 語が `[[ F V ]]`、`[ F V ]`、`test F V` のどれかに完全に一致するときだけ判定と読む。F は `-n`(pane の中で真)か `-z`(pane の中で偽)、V は `$HERDR_PANE_ID` か `${HERDR_PANE_ID}`。`!`、`-v`、`${HERDR_PANE_ID:-}`、ほかの変数、余分な語は判定ではない
- 引用符を外した語で見るので、`"[["` も判定と読む。zsh ではこれは予約語ではなくコマンドで、alias は pane でも定義される(tech-debt の 137 行の 1 つ目)

## 3. tracker(`shellAliasGuardTracker`、`:590`。読む価値が一番高い)

- 1 行の中の形(`inlineGuarded`、`:787`): alias の文の直前の文が判定で、その判定が並びの頭にあり、区切りが `-n` なら `||`、`-z` なら `&&` のときだけ印を付ける。`{ … }` で始まる alias の文には付けない
- 並びの頭: 行の初め、`;` のあと、`then`・`else`・`do`・`{` で始まる文。行の初めを並びの頭と読むのは、前の物理行が閉じて終わったときだけ(`prevLineEndedClosed`、`:652`、`continuesPrevLine`、`:644`)。前の行が `&&`、`||`、`|`、行末の `\`、閉じない引用符で終わったら読まない。計画助言の codex が `false &&` ⏎ `[[ -n $HERDR_PANE_ID ]] || alias …` を zsh で確かめて見つけた穴で、空行とコメント行は前の行の状態を変えない
- if の形(`followIf`、`:709`、`inGuardedBranch`、`:758`): ファイルごとに if の入れ子を積む。`-z` の if の then の側と `-n` の if の else の側にある alias に印を付ける。if の条件は判定の 1 文だけで、`;` か改行で閉じたときに限る(`condClosedBeforeThen`、`:660`。`&&` や `||` で並べた条件、`&` で background にした条件、2 文以上の条件は「それ以外」)。consult の指摘で入れた
- 対応が崩れたら(ファイルの終わりで if が閉じていない、対応のない `fi`、条件の直後の `else`・`elif`・`fi`)、そのファイルの if による印をすべて消す(`scanShellAliasFile`、`:966`)。1 行の中の形の印は残す
- 印は `shellAliasDef.PaneGuarded`(`:837`)

## 4. severity と Detail(`checkShellAliases`、`:1302`)

- 印の付いた def は、値を最後まで読めていれば(`d.PaneGuarded && !d.Incomplete`、`:1371`)codex と claude の warn の一覧に入れず、別の一覧に入れる。warn の条件は今までどおり、印のない codex の finding か、印のない claude の `--permission-mode` だけを数える
- 別の一覧は `shellAliasPaneGuardedSentence`(`:1214`)の 1 文になる。「not defined in herdr's panes; … herdr sets HERDR_PANE_ID in the shell of each pane it opens (checked on herdr 0.7.5), so no seat, and no director started in herdr, gets its flags」。alias の値は出さない
- warn の文の最後の案内(`shellAliasRemedyClause`、`:1133`)に、3 つ目の手として「in a bash or zsh rc, define it only outside herdr's panes with `[[ -n $HERDR_PANE_ID ]] || alias <name>=...`」を足した。fish の rc は読まないので、bash と zsh に限ると断る(self-review の L5)

## 5. 文書

- `docs/recipes/codex-seat-permissions.md`(templates の写しも): 前提の「Shell aliases break every codex spawn」の項に 3 つ目の回避策、doctor が読む形、severity の括弧
- `docs/recipes/isolated-herdr-server.md`(templates の写しも): 「Shell aliases」節に 1 文
- org skill の 4 写しの「shell alias に注意」: 同じ回避策と doctor が読む形を日本語で。permission mode の「guarded」と取り違えないよう、「`HERDR_PANE_ID` を条件にした alias」と書いた
- `docs/tech-debt/README.md`: 136 行に carry-over の文、137 行(誤って印が付く 3 形)と 138 行(ファイルの大きさ)を新しく足した。417 行と 373 行には sync-docs が carry-over の文を足した。417 行は #346 も同じ行に文を足していたので、取り込みのときに両方の文を残した

## 6. テスト

- `internal/cli/doctor_shell_alias_test.go`: 区切りの出力、判定の分類、`scanShellAliasFile` の印(印を付ける形 28 件、付けない形 30 件ほど)、`checkShellAliases` の status と Detail(issue の 2 行、AC2 の形の表、AC3 の形の表、印のあるものとないものの混在、印のある claude の `--permission-mode`、herdr がない環境、値が Detail に出ないこと、案内の文)
- implementer が S1 で 23 個、S2 で 8 個の変異を当て、すべて落ちた。tester が見つけた G1(`if [[ -z $HERDR_PANE_ID ]] & then …` の判定を消す変異が生き残る)は ac0c403e の 1 行で埋め、overlay の変異で落ちることを確かめた
- `run-test.sh` は 27b08bc6 で rc 0(56/56)。`-run ShellAlias` の top-level 98 本が PASS。`doctor_shell_alias.go` の coverage は 98.9%

## 見どころ

- 誤って印を付ける向きは、座席に効く alias の warn を黙らせる。迷う形はすべて印を付けない側に倒した。印を付ける形は、implementer と reviewer が zsh 5.9 に `HERDR_PANE_ID=wD:p1` を入れて読ませ、pane では alias が定義されないことを確かめた
- 残した穴は tech-debt の 137 行(引用符付きの判定の語、rc が判定の前に `HERDR_PANE_ID` を書き換える形、heredoc や複数行の文字列の中の `if`/`fi`)。どれも実在の rc ではまず使わない形
- herdr の実機の pane に `HERDR_PANE_ID` が入ることは、このタスクでは確かめ直していない。根拠は issue の報告(herdr 0.7.5)と、`docs/evidence/test-2026-10-15-org-director-start.md` の実機の pane の `env`(`HERDR_PANE_ID=w1:p1`)
