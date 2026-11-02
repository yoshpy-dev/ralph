# Self-review report: init-config-questions

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-init-config-questions.md`(issue #277、D1〜D10、AC1〜AC10、スライス S1 82956f18、S2 f306d479、S3 d7316e5b)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base c8c8c0f1、HEAD e1d5f1ec、7 ファイル、+2096/-21)。コードは `internal/cli/init.go`、`init_settings.go`、新しいテスト 3 本(`init_settings_test.go`、`init_settings_write_test.go`、`init_settings_form_test.go`)、`README.md` の `ralph init` の行。計画の本文は読んだが、所見の対象にしていない。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は L1〜L5(すべて LOW)。CRITICAL、HIGH、MEDIUM はない。

## Evidence reviewed

- `git diff origin/main...HEAD` の全体と、`git log --oneline origin/main..HEAD` の 5 コミット。
- 依頼で名指しされた 5 点の読み方:
  - `applyInitSettings` の行ベースの書き換え(`init_settings.go:133-276`)。section の境界は `initTOMLTables` が見出しごとに行へ付ける表で決まり、`initTOMLSet` は「その表の中で、コメントでない `key = value` がちょうど 1 行」のときだけ値を替える。0 行も 2 行以上もエラーで、返り値は nil。見出しの正規表現(`:97`)は末尾にコメントのある見出しや引用符つきの key を認めないが、その場合は後続の行が前の表に入り、key が見つからない(または 2 行ある)エラーに倒れる。黙って別の表の値を替える経路は見つからなかった。値の形(`:104`)は真偽値、エスケープなしの文字列、同じ行で閉じる配列だけで、それ以外はエラー。CRLF は `\r` を外して照合し、行末を戻す。`model_pool` のコメントアウト(`:241-276`)は行数を変えず、閉じる `]` が表の外にあるとき、選んだ CLI の行が 1 つもないとき、`]` がないときにエラー。TOML に入る文字列は定数(`claude`、`codex`、`ci`、`local`)と真偽値だけで、project 名などの利用者の入力は届かない。
  - `initAnsweredFS`(`init_settings.go:79-91`)。埋め込みは `fs.FS` のインターフェースなので昇格するのは `Open` だけで、`fs.ReadFile` は包みの `ReadFile` に当たり、`fs.WalkDir` は元の FS の木を歩く。`scaffold.RenderFS`(`internal/scaffold/render.go:66`)は `fs.ReadFile(src, path)` で読み、そのバイト列を書いてハッシュを取る。返す値は複製で、呼び出し側の書き換えが次の読み取りに残らない。`reconcileBlockSurfaces` は包んでいない `baseFS` を読み、`ralph.toml` は block の面ではないので影響しない。
  - manifest のハッシュ(`init.go:390-405`、`:455-460`)。RenderFS は答え入りの内容をハッシュするので `hashes["ralph.toml"]` を雛形のハッシュに戻し、作った・上書きしたときだけ `DiskHash` に書いた内容のハッシュを入れて `SetFileOwned` で seed にする。Skipped のときは `DiskHash` なしで、答えなしの init と同じ記録。`SetFileOwned` は `Hash` と `TemplateHash` に同じ値を入れるので、`Hash` も雛形のハッシュになる。
  - `executeInit` の並べ替え(`init.go:343-377`)。manifest の確認、`scaffold.BaseFS()`、答えの計算、`MkdirAll` の順になった。答えが雛形に合わないとき、対象のディレクトリも作られない(テストが「新しい directory」と「空の既存 directory」で確かめる)。manifest がある場合は `MkdirAll` を通らなくなるが、manifest があれば対象の directory はすでにあるので結果は変わらない。
  - `askInitSettings` と `runInitInteractive`(`init_settings.go:283-292`、`init.go:146-191`)。`ralph.toml` があるとき(ファイル、symlink、dangling symlink、directory)は聞かず 1 行を出し、`--force` なら聞く。第 2 のフォームは `askSettings` が真のときだけ組み立て、CLI の答えが空になる経路はない(`cmp.Or(recommended, initCLIBoth)`)。フォームの欄が 0 件になる経路もない(CI と自動 merge は `bothOnly` でない)。
- 読んだテスト: S1 の 12 本(oracle が実装の表を写さず、行番号と行の中身で期待を作る)、S2 の `executeInit` と upgrade の dry-run、doctor、AC10 のテスト、S3 の accessible モードのテスト。テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | naming | L1: 型 `initTOMLSetting` と関数 `initTOMLSettings` の名前が末尾の `s` 1 文字しか違わない。型は 1 つのキーの書き込み(table、key、value)、関数は答え `initSettings` から書き込みの一覧を作る。`initSettings`(答えの型)を含めると、似た名前が 3 つ並ぶ。grep で `initTOMLSetting` を探すと型と関数の両方に当たり、呼び出し箇所(`init_settings.go:134`)でも `settings, keep, drop, err := initTOMLSettings(s)` と読む人は型の複数形と取り違えやすい | `init_settings.go:115-117`、`:156`、`:134` | 関数を `initTOMLSettingsFor(s)` か `initSettingsToTOML(s)` に変える(呼び出しは 1 か所、テストは関数を直接呼んでいない)。型は `initTOMLKeyValue` でもよい |
| LOW | maintainability | L2: `executeInit` が 130 行から 171 行になった(`git show origin/main:internal/cli/init.go` と比べた)。もともと長い関数に、答えの用意(`init.go:357-371`)、ハッシュと 1 行の表示(`:390-405`)、manifest の分岐(`:455-460`)が直接入った。コードレビューの基準(関数は 50 行未満)から外れた幅が広がり、`ralph.toml` に関する処理が 3 か所に散る。新しい処理の中身はすでに `init_settings.go` の関数(`answeredInitSettingsTOML`、`initAnsweredFS`)に分けてあるので、残りは継ぎ目だけ | `init.go:343` 以降 | 継ぎ目を関数にまとめる。たとえば `prepareInitSettings(baseFS, settings) (renderFS fs.FS, answered []byte, templateHash string, err error)` と、`recordInitSettings(result, answered, templateHash, hashes) (diskHash string)`(1a の分岐と表示)。manifest の分岐は、block 用の `blockDiskHashes` と同じ扱いに寄せられる(`owner == scaffold.OwnerBlock` の条件を外す必要がある)。急がないなら、次に `executeInit` を触るときでよい |
| LOW | readability(利用者向けの文) | L3: 「ralph.toml があるので聞かない」の 1 行が `--force` を勧める文面になっている: `ralph.toml already exists, so ralph init does not ask about its settings (edit ralph.toml to change them; --force overwrites it and asks).`。`--force` の説明は「overwrite existing files without prompting」で、`ralph.toml` だけでなく、雛形にある既存のファイル(`AGENTS.md`、`CLAUDE.md`、`.claude/` の設定など)をすべて上書きする。この文面は「it」を `ralph.toml` だけに読ませるので、5 つの問いに答えたいだけの人が、手で書いた `AGENTS.md` ごと失う操作を選びうる。`ralph.toml` が残っていて manifest がない場面は、前の init が途中で止まった場合(AC10)と、ralph を使っていない project に手で `ralph.toml` を置いた場合で、後者は `AGENTS.md` などを利用者が書いている可能性が高い。`init` の help(`init.go:46-47`)と README も同じ言い方 | `init_settings.go:290`、`init.go:93`(`--force` の説明)、`:46-47`、`README.md` の `ralph init` の行 | 文面を「`--force` overwrites every existing scaffold file, ralph.toml included, and asks」のように、範囲を書く形にする(help と README も同じ言い方に合わせる)。テストの `note`(`init_settings_form_test.go:367`)も合わせる |
| LOW | readability(利用者向けの文) | L4: CLI の問いの説明は 2 行目の `(Recommended): found on this machine's PATH.` を定数で持つので、PATH に claude も codex もない(`recommendedCLI` が `""` を返す)ときも出る。そのとき選択肢のどれにも `(Recommended)` が付かないので、説明が見せていない印を説明する形になる。テストも「説明が印を説明するので、説明自身の `(Recommended)` は数えない」と数え方で吸収している(`init_settings_form_test.go:217-219`)。CI の container や新しい machine で `ralph init` を回す人が、最初に見る問いで起きる | `init_settings.go:296-300`、`:314-326`、`:346-354`、`init_settings_form_test.go:217-219` | 説明の 2 行目を `recommended != ""` のときだけ足す(`newCLIQuestion` が組み立てる)。テストの数え方も、見つからないときの説明に印の語がないことを確かめる形にできる |
| LOW | maintainability(利用者の気づき) | L5: CLI の答えから決まる、問いに出ない変更が、出力にも `ralph.toml` にも説明なしで入る。codex だけを選ぶと `[org.watchdog].watcher_enabled = false`(D8)が書かれ、1 つだけ選ぶと `model_pool` のもう一方の行がコメントになる(D9)。init の出力は `✓ ralph.toml (wrote your answers)` の 1 行だけで、何を書いたかを言わない。あとで Claude Code を足す人は、`driver_pool` に足して `model_pool` の行を戻すことまでは雛形のコメントから読めるが、見張り(watcher)が init に止められたことは `ralph.toml` のどこにも書かれていない(`[org.watchdog]` の上の雛形のコメントは watcher の層を説明するだけで、init が替えたとは言わない)。行数を変えない設計(テストが固定)なので、コメント行を足す案は採りにくい | `init_settings.go:184-186`、`:236-241`、`init.go:401`、`templates/base/ralph.toml:231-238` | 出力の 1 行に、CLI の答えから決めた変更を足す。たとえば codex だけなら `(wrote your answers; also turned the watcher off, which only runs claude)`。コメント行を足さずに済み、テスト(`initSettingsWroteLine`)は文字列を合わせるだけで済む |

## Positive notes

- 失敗の順序が計画どおりに作られている。答えの計算は `MkdirAll` より前で、雛形に合わない場合は何も書かずに止まり、テストが対象の directory の有無と出力の空を確かめる。`ralph.toml` は RenderFS の最初の書き込みで答え入りになるので、描き出しの途中で失敗しても答えが残る(AC10 のテストが、`scripts` を普通のファイルにして実際に確かめる)。
- 書き換えが「見つからなければエラー、2 つあればエラー、nil を返す」の形で一貫している。呼び出し側が部分的に書き換えた内容を受け取る経路がない。エラーの文言が `[table].key` を名指しする(`TestInitSettings_MissingKeyIsAnErrorNamingIt` が 10 の key で確かめる)。
- 型の持ち方: `initSettings` は「1 つのときは計画の確認と変更のレビューを false にする」を `both &&` で一か所に寄せていて、1 つの CLI に `PlanReview: true` が入っても `ralph.toml` には書かれない。
- `initAnsweredFS` の前提(RenderFS が `fs.ReadFile` で読む)は、別 package の挙動に頼る継ぎ目だが、テスト(`TestInitSettings_AnsweredFSServesTheAnswersForRalphTOMLOnly`)が RenderFS を実際に通して書かれた内容とハッシュを確かめる。前提が変わると落ちる。
- テスト: 実物の雛形に 48 の組み合わせを当てて `config.Load` に通し、答えが書かない値は雛形のまま読めることを `reflect.DeepEqual` で確かめる。期待の行は実装の表を写さず、雛形の行番号から作る。CRLF、最終行の改行なし、`[[...]]` の見出し、値の後ろのコメントを手で組んだ雛形で固定する。手で `TemplateHash` を答えのハッシュにした場合に upgrade の advisory が出ることまで確かめ、AC7 のテストが空振りしないようにしてある。

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (なし) | | | | |

_(L1〜L5 はこの PR の中で直せる大きさで、繰り延べる項目はない。計画の「既知の穴」(第 2 のフォームから第 1 に戻れない、`ralph.toml` の非原子的な書き込み、accessible モードで description が出ない)は計画の Risks と D10 に理由つきで書いてあり、PR の本文に書く扱い。非原子的な書き込みは、既存の行 `ralph init/upgrade lack transactional safety` が覆う。)_

## Recommendation

- Merge: yes(CRITICAL、HIGH、MEDIUM はない。LOW の 5 件は PR の前でも後でもよい)
- Follow-ups:
  - 先に直すなら L3(`--force` の文面)と L4(説明の 2 行目)。どちらも利用者が最初に読む文で、直しは文字列とテストの期待だけ。L5 の 1 行を足すなら L3 と同じ commit にできる。
  - L1 は名前の変更だけで済む。L2 は急がない。
  - 直す場合は計画のとおり、`/self-review` から `/cross-review` までを回し直す。

## Run 2 (S4, 1af26382)

- Date: 2026-11-02
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1 の 2 回目)
- Scope: `git show 1af26382`(4 ファイル、+72/-16: `README.md`、`internal/cli/init.go`、`internal/cli/init_settings.go`、`internal/cli/init_settings_form_test.go`)。Run 1 の L1、L3、L4 の直しと、その直しが持ち込む問題だけを見た。Run 1 の読み方(`applyInitSettings` の書き換え、`initAnsweredFS`、manifest のハッシュ、`executeInit` の順序)に触れる変更はこの commit にないので、読み直していない。
- ID: 新しい所見は 0 件(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 0)。

### Run 1 の所見の扱い

| ID | 結果 | 確かめたこと |
| --- | --- | --- |
| L1 | resolved | 関数は `initSettingsToTOML` になった(`init_settings.go:153-156`、呼び出しは `:134` の 1 か所)。旧名 `initTOMLSettings` は repo 全体の grep で、この報告書の Run 1 の引用以外に残っていない(コード、コメント、テスト、docs のどこにもない)。型 `initTOMLSetting` との差は 1 文字ではなくなった |
| L3 | resolved | 文面が「`--force` asks, and overwrites every file init writes」に変わった(`init_settings.go:290`)。help(`init.go:46-47`)、README(`README.md:127`)、テストの `note`(`init_settings_form_test.go:409`)が同じ言い方でそろっている。旧文面の「overwrites it」は repo に残っていない。内容が挙動と合うかも見た: `--force` は `RenderFS` の `Overwrite: true`(`init.go:384`)で base の全ファイルを上書きし、AGENTS.md と .gitignore も RenderFS が書くので手書きの内容ごと置き換わる(`init.go:408-412` のコメント)。pack も `renderPackInto(..., force)`(`init.go:424`)で上書きされる。manifest があるときは、この文を出す前に `runInitInteractive` が抜ける(`init.go:152-155`)ので、何も上書きしない場面でこの文が出ることはない |
| L4 | resolved | `cliQuestionDescription`(`init_settings.go:330-338`)が、`recommended` が空なら 1 行目だけ、そうでなければ 2 行目(`initCLIRecommendedNote`)を足して返し、`newCLIQuestion` が使う(`init_settings.go:364`)。80 桁のテストと技術用語のテストの表に note を足してある(`init_settings_form_test.go:334`、`:380`)。新しいテスト `TestCLIQuestion_ExplainsRecommendedOnlyWhenAnOptionCarriesIt`(`:234-273`)が 4 通り(どちらも見つからない、claude だけ、codex だけ、両方)で、form の view の `(Recommended)` がラベル 1 + note 1 か 0 か、note の有無、accessible モードの数を確かめる。期待が production の関数ではなく定数と数から作られているので、説明の出し分けが壊れると落ちる |
| L2 | left (reason in the plan) | 依頼のとおり再提起しない |
| L5 | left (reason in the plan) | 依頼のとおり再提起しない |

### 直しが持ち込む問題の確認

- 変更は文字列の定数、説明を組み立てる関数 1 つ、関数名の変更、help と README の文、テストの期待。エラー処理、null の扱い、ファイルへの書き込み、利用者の入力が届く経路には触れていない。
- 説明の出し分けは分岐が 1 つで、既存の `recommendedCLI` の戻り値(`""` か 3 つの値)をそのまま使う。`cmp.Or(recommended, initCLIBoth)` の初期値の扱い(`init_settings.go:361`)は変わっていない。
- 既存のテスト `TestCLIQuestion_StartsOnTheCLIFoundAndRecommendsIt` は、`recommendedCLI` を呼ぶ回数を数える検査(`init_settings_form_test.go:198-201`)を `description` の計算(`:208`)より前に置いたままなので、追加の呼び出しで崩れない。
- 見送った点: `initSettingsToTOML` は書き込みの一覧を返し、`answeredInitSettingsTOML`(`init_settings.go:61`)は TOML の本体を返すので、「TOML」の語が並ぶ 2 つの関数の返すものが違う。ただし Run 1 が候補に挙げた名前で、doc comment(`:153-155`)と返り値の型から読み取れる。変えると `/self-review` からの回し直しになるため、所見にしていない。

### Recommendation (Run 2)

- Merge: yes(ブランチ全体で CRITICAL、HIGH、MEDIUM はない。LOW は Run 1 の L1、L3、L4 が解消、L2 と L5 が計画の理由つきで見送り、Run 2 の新しい所見は 0 件)
- Follow-ups: `/verify` に進める。L2(`executeInit` の長さ)と L5(`watcher_enabled` と `model_pool` のコメント行の説明)は計画の記述のとおり、この PR では直さない。
