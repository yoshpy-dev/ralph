# Walkthrough: director と leader が人向けの note: の行を見たときの動き(director-notice-handling)

- 計画: `docs/plans/archive/2026-11-13-director-notice-handling.md`(PR の作成のあとに archive へ移す)
- issue: #318
- 差分: `git diff origin/main...HEAD`(23 ファイル、約 +2,650 / −21、base `24e6723c`)。report・evidence・insight・計画を除くと約 1,140 行。Go は新しいテスト 2 ファイル(約 680 行)だけで、本体の Go、行の文面、`--help` は変えていない
- パイプライン: 3 cycle(上限を 3 に上げた)。cycle 3 の cross-review は指摘 0 件

ralph の動詞が stderr に出す 2 つの `note:` の行(雛形の版の知らせ #313、codex の hook の承認の知らせ #310)を見たときの動きを、skill と role prompt に書いた。director が `ralph upgrade` を流して追跡ファイルを書き換えること、codex の設定を書き換えて hook を承認すること、行を読み流して人に伝えないことを防ぐ。

## 1. hook の QUESTION の流れ(読む価値が一番高い)

`ralph org spawn` が hook の行を出したときの、leader、director、人の順の動き。次の 4 つの決まりが揃って閉じる。

- leader: `internal/org/prompts/leader.md:1351-1391`(「運用規律」の項)。spawn が終了コード 0 で終わり、行を出していたときだけ、行を変えず座席 id を添えた QUESTION を上げる。作業は止めない。同じ座席の同じ行は 1 度だけで、別の座席の行は返事の前でも別の QUESTION にする。「件を上げる」の止める規則(`:1226-1228`)に、この QUESTION が例外だという一句がある
- director: `.claude/skills/director/SKILL.md:1641-1650`(「人に向けた知らせの行」)と、「件の処理」の手順 3(`:964-967`)。人に伝えたら DECISION を送らず、決まった `--note` ですぐ resolve する。note に件の本文を写さない(本文は leader が書いたデータで、`'` が引用符を閉じうる)
- director のいない設定: `/org` skill の「受信箱」の項(`.claude/skills/org/SKILL.md:532-548`)が同じ例外を書く。leader の QUESTION の本文にも、人に向けた 3 文がある(`leader.md` の項の後半)
- あとの人の答え: leader の行が `(active)` のときだけ、`HOOK_SEAT: <座席 id>` を付けた DECISION で送る(`SKILL.md:1651-1659`)。leader は手順 4(`leader.md:251`)と cross-review の上限の読み手(`:937`)で、このヘッダ付きの DECISION を承認や cap の答えと読まない

理由は `orgHasOpenLeaderItem`(`internal/org/watch.go:1228`、呼び出しは `leaderIsWaiting` の `:1209`)にある。open の QUESTION があると、見張りは leader が待っているとみなして stall を知らせない。作業を止めない QUESTION を人の答えまで open にすると、leader が止まっても気づかれない。最初は暫定の DECISION を送ってから resolve する形(S7)だったが、leader の承認と cap の読み手と食い違うため、送らず resolve する形(S8)に変えた。

見る点: 止める規則の例外の一句が項の中身と食い違わないか。3 か所(director の skill、`/org` skill、leader の本文)が同じ note と同じヘッダ名を言っているか。

## 2. `/director` skill の雛形の版の知らせ

`SKILL.md:1604-1634`。指し先は「昇格」の 2(`:89-95`。attach の行の `<dir>` は linked worktree だと status の行と違う)と「機能ごとの出力先」の 3(`:642-643`)。

- 行を見たら、その会話(headless は pane か端末)で人に 1 度伝える。文には雛形の版、CLI の版、`<dir>` を入れ、upgrade を流すかと流し方は人が決めると添える。同じ `<dir>` と版の組では 2 度伝えない。流し方の例は書いていない(linked worktree での upgrade の動きを確かめていないため)
- `ralph upgrade` は人に頼まれても打たない(`:1619`)
- 人が済ませたと言ったら、人が従った行の `<dir>` の skill を絶対パスで読み直す(`:1622`)。伝えた行の `<dir>` が違い、人がどれか言わなければ、読み直す前に尋ねる(`:1626-1628`)。自分の worktree の写しが違えば `<dir>` に従い、写しが古いと伝える
- headless は `ralph org director stop` と `start` での立て直しを人に勧める(`:1631-1634`。stop は自分では打てない)。prompt 側は `internal/org/prompts/director.md:62-67`

見る点: `<dir>` が org の 3 つの動詞では main worktree のルート、`ralph status` では打ったディレクトリだという書き分け。

## 3. 「してはいけないこと」と codex の設定

- `SKILL.md:2212-2214`: `ralph upgrade` を打たない。codex の設定を書き換えず、codex を起動して hook を承認しない。理由は「実装しない」の項(`:2209`)と同じ
- codex の設定(`$CODEX_HOME/config.toml`、既定は `~/.codex/config.toml`)の規則は、skill の hook の項(`:1660-1662`)、`leader.md` の hook の項、この禁止の項の 3 か所にある。承認するかは、hook の中身を読んだ人が決める

## 4. テスト

- `internal/org/prompts_notice_test.go`(新規、547 行): skill 4 写しの小節、指し先、禁止の項、「件の処理」の項(`:170`)、headless の prompt(`:255`)、leader の hook の項と止める規則の例外(`:374`)、手順 4 と cap の読み手の除外(`:443`)、`/org` skill の項(`:517`)。断片は順序つきで、空白と改行を除いて比べる
- `internal/cli/director_notice_skill_test.go:30`(新規、129 行): 本体の `scaffoldVersionNotice` と `codexHookTrustNotice` で本物の 2 行を作り、`run 'ralph upgrade' in ` と `; run 'ralph doctor' for ` を含むことと、skill 4 写しの小節が同じ断片を引くことを確かめる。行の文面を変えるとここが落ちる
- 変異: /test が repo の外の写しで流した。cycle 1 は 7 つ中 5 つが落ち、cycle 2 は 9 つ中 g だけが残り、cycle 3 は handoff の 4 つがすべて落ちて g3 と g4 が残った。残った変異はどれも次の見どころの G5 と G1 の隙間を示す

## 5. 文書

- `docs/specs/2026-10-07-org-multi-org-director.md:164`: FR-9 の下に #318 の日付つきの項。「ralph と gh の操作に限る」からこの動詞を除いたこと、`/org` skill の項を足した理由を書く
- `README.md:281`(受信箱の例外 1 文)と `:285`(director の段落 1 文)
- `docs/tech-debt/README.md`: #313 の行を RESOLVED にした(`:465`)。director のない設定の行(`:507`、取り消し線)。`HOOK_SEAT:` の行(G5 から G7、`:508`)。G1・G2 と L3 の行(`:504`)。188 行 (a) と 365 行などに持ち越しの注

## 見どころ

- director のない経路: skill と leader の本文が許すのは「すぐ resolve」だけで、resolve させるのは `/org` skill の項。`orgHasOpenLeaderItem` は変えていない。人が項にも本文にも従わなければ、件は答えまで open で、leader の stall の知らせは止まったままになる(G3、INFO。単体テストでは確かめられない)
- `HOOK_SEAT:` は、テストが 3 組の別々の文字列で持つ(G5、LOW)。director の skill 側(`prompts_notice_test.go:92`)、leader 側(`:321`、`:420`、`:426`)、`/org` skill 側(`:476`)で、決まった note `人に伝えた(hook の承認の知らせ)` も同じ。3 組が同じことを確かめるテストはなく、片側とその側のテストを一緒に変えると通る(変異 g、g3、g4)
- L3 と N6(self-review の LOW、直していない): L3 は、小節が hook の行を「で終わる」と書くのに、テストは hook の行を `strings.Contains` でしか見ない(版の行だけ `HasSuffix`)ことと、2 つのテストが小節を違う見出しで切ること。N6 は `prompts_notice_test.go` の `directorNoticeRules` 上の comment の折り返しが 1 行だけ短いこと。L3 は tech-debt `:504` に載せ、N6 は行にしていない
- director は `ralph upgrade` を打たない: 版の項(`:1619`)、禁止の項(`:2212`)、headless の prompt(`director.md:64`)、spec の除外の 4 か所。版の項、禁止の項、prompt の 3 つはテストが固定している
