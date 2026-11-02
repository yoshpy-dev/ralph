# Walkthrough: init-config-questions(issue #277)

- 計画: docs/plans/archive/2026-11-02-init-config-questions.md
- ブランチ: feat/init-config-questions(base 50f2ffc0。c8c8c0f1 から作り、f051d91c で origin/main を取り込んだ)
- PR: #300
- 差分: 17 ファイル、+2,578/−26。
  - Go の本体は `internal/cli/init.go` と、新しい `internal/cli/init_settings.go` の 2 ファイル。
  - テストは 3 ファイル(どれも新規)。
  - 残りは文書(README、codex-setup の recipe の 2 つの写し、spec の注、tech-debt の 1 行)と pipeline の記録。

## 読む順

1. `internal/cli/init_settings.go` の前半(`initSettings` から `initTOMLCommentModelPool` まで)。答えを雛形の `ralph.toml` に書き込む部分(S1)
2. `internal/cli/init.go` の `executeInit`。答えの計算、`initAnsweredFS`、manifest の 2 つのハッシュ(S2)
3. `internal/cli/init_settings.go` の後半(`askInitSettings` から `newSettingsForm` まで)と、`init.go` の `runInitInteractive`。問いと 2 段のフォーム(S3・S4)
4. テスト 3 ファイル(下の「テスト」)
5. 文書: `README.md` の `ralph init` と `ralph doctor` の行、`docs/recipes/codex-setup.md`、`docs/specs/2026-10-07-org-multi-org-director.md` の FR-8 の末尾の注

## 答えを書く部品(S1)

- `initSettings` は 5 つの答えを持つ。使う CLI は `initCLIClaude`・`initCLICodex`・`initCLIBoth` のどれかで、ほかの 4 つは計画の確認、変更のレビュー、CI、自動 merge の真偽。
- `initSettingsToTOML` は、答えを「表・key・値」の一覧に直す。CLI が 1 つのときは、計画の確認と変更のレビューを答えによらず false にする。codex だけなら `[org.watchdog].watcher_enabled = false` を足す。
- `applyInitSettings` の手順:
  - 雛形を行に分け、`initTOMLTables` が各行の属する表を決める。見出しは 0 桁目の `[name]` と `[[name]]`。配列を閉じる `]` や字下げした行は見出しにしない。
  - `initTOMLSet` が、その表の中のコメントでない `key = value` の行の値だけを替える。key の前の字下げ、`=` のまわりの空白、値のあとのコメントは残す。
  - CLI が 1 つのときは、`initTOMLCommentModelPool` が `[org]` の `model_pool` の配列の中で、もう一方の driver の行の字下げのあとに `# ` を入れる。行の数は変わらない。
  - 改行(LF・CRLF)と最後の改行の有無は雛形のまま。
- エラーになるもの: key が表にない、表に 2 行ある、値が 1 行に収まらない。CLI が 1 つのときは、`model_pool` がない、閉じる `]` が `[org]` の中にない、選んだ CLI の行が 1 つも残らない。どの場合も中身は返さない。

## init の書き込み(S2)

- `executeInit` の順:
  1. 既存の manifest を確かめる(前は `MkdirAll` のあとだった)
  2. `scaffold.BaseFS()` を読む
  3. 答えがあれば `answeredInitSettingsTOML` で書く内容と雛形のハッシュを計算する。エラーなら、対象のディレクトリも作らずに止まる
  4. `MkdirAll` し、`RenderFS` を呼ぶ
- `RenderFS` には、答えがあるとき `initAnsweredFS` を渡す。`ReadFile("ralph.toml")` だけが答えの内容を返し、ほかのパスと `Open` は雛形のまま。`RenderFS` は `fs.ReadFile` で読むので、`ralph.toml` は最初の書き込みから答えの内容になる(計画助言の HIGH、D10)。
- 描き出しのあと:
  - `ralph.toml` を作ったか上書きしたときは `✓ ralph.toml (wrote your answers)` を出し、書いた内容のハッシュを `DiskHash` に取っておく。
  - 既にあって書かなかったときは `⚠ ralph.toml: already exists, so your answers were not written` を出す。
- manifest の `ralph.toml` は、`TemplateHash` と `Hash` が雛形のハッシュ、`DiskHash` が書いた内容のハッシュ(`SetFileOwned`、seed)。upgrade の seed の判定は `TemplateHash` と新しい雛形のハッシュだけを比べるので、init 直後の upgrade は `ralph.toml` の advisory を出さない。
- 答えがないとき(`--yes`、または `ralph.toml` が既にある)は、前と同じ経路になる。

## 問いと 2 段のフォーム(S3・S4)

- `askInitSettings`: `ralph.toml` が `os.Lstat` で見つかり(普通のファイル、symlink、ディレクトリ)、`--force` がないときは聞かない。そのときは 1 行を出す: `ralph.toml already exists, so ralph init does not ask about its settings (edit ralph.toml to change them; --force asks, and overwrites every file init writes).`
- `newCLIQuestion`(Select)の初期値と印:
  - `recommendedCLI` が PATH の `claude` と `codex` を探す。片方だけならその CLI、両方なら Both を初期値にし、選択肢に「(Recommended)」を付ける。
  - どちらもなければ初期値は Both で、印は付けない。その場合は `cliQuestionDescription` も「(Recommended)」の説明の行を出さない(S4 の L4)。
- `newSettingsForm` は `initSettingsQuestions` の表から Confirm を組む。Both なら 4 問、1 つなら CI と自動 merge の 2 問。初期値は No・No・No・Yes で、雛形の既定と同じ。
- `runInitInteractive` の流れ:
  1. 1 つ目のフォームは、今の group(名前、言語パック)と、聞くときだけ CLI の group。
  2. 1 つ目が終わってから、CLI の答えで 2 つ目のフォームを組んで走らせる。
  3. 答えを `cfg.Settings` に入れて `executeInit` に渡す。
- 1 つのフォームで group を隠さない理由: huh v2.0.3 の accessible モード(`TERM=dumb`)は、`WithHideFunc` を見ずに全部の field を聞く。
- 問いの文言は英語で、用語を使わない。
  - CLI の問い: "Which AI coding CLI will you use?"
  - 計画の確認: "Have the other AI check each plan before work starts?"
  - 変更のレビュー: "Have the other AI review each change before its pull request?"
  - CI: "Does CI run your tests on every pull request?"
  - 自動 merge: "Merge pull requests whose checks pass without asking you?"

## テスト

- `internal/cli/init_settings_test.go`(S1)
  - 実物の雛形 `templates/base/ralph.toml` をディスクから読む。テストの埋め込みの FS の `ralph.toml` は 4 行の偽物で、key がないため。
  - 答えの 48 通りを `config.Load` に通し、表の値と、ほかの値が雛形と同じことを確かめる(`TestInitSettings_EveryAnswerLoadsFromTheTemplateWithTheTableValues`)。
  - ほかに確かめること: コメントと行の数が保たれる、答えの行だけが変わる、key・表・`model_pool` の不足や重複がエラーになる、手作りの雛形で改行と表の境目が保たれる。
- `internal/cli/init_settings_write_test.go`(S2)
  - 答えがないときは、雛形と同じバイト列を書く(AC4)。
  - 答えがあるときは、最初の書き込みから答えの内容になり、manifest に 2 つのハッシュが入る。
  - 既にある `ralph.toml` は、`--force` なしなら変えず、ありなら答えで上書きする。
  - 雛形に合わないときは、何も作らない(AC8)。
  - init 直後の upgrade の dry-run に advisory が出ない。`TemplateHash` を書いた内容のハッシュにした対照では 1 件出る(AC7)。
  - CLI ごとに、その CLI だけの偽の binary で `runDoctorOpts` が通る(AC5)。
  - 対象の `scripts` を普通のファイルにして描き出しを失敗させても答えが残り、答えなしのやり直しのあとも残る(AC10)。
- `internal/cli/init_settings_form_test.go`(S3・S4)
  - 2 つ目のフォームに accessible モードで答えを流し、Both で 4 問、1 つで 2 問を聞いて答えが入ることを確かめる。入力は `iotest.OneByteReader` で 1 バイトずつ渡す。accessible モードが問いごとに新しい `bufio.Scanner` で読むため。
  - CLI の問い: PATH の 4 通りの初期値と「(Recommended)」、番号で選ぶ動き。
  - 表示する文字列に禁止語(cross-review、regate、gate、advisory、driver、pool、director、leader、seat、org、watcher、per-plan、toml)がない(AC3)。
  - description の行が 76 文字以内に収まる。
  - `askInitSettings` の判定(普通のファイル、symlink、ディレクトリ、`--force`)。

## 確かめ方と残り

- `./scripts/run-static-verify.sh` は rc 0、`./scripts/run-test.sh` は rc 0(2,951 秒、取り込みの前)。
- 実物の binary を expect の pty で 7 通り動かした(test report)。
- 取り込みのあと、`go build ./...`、`go vet ./internal/cli/`、init まわりの `go test` を打って通った。
- cross-review(codex)の所見は 0 件。
- 残り: `runInitInteractive` の 2 つのフォームのつなぎを通しで流す Go のテストがない(`docs/tech-debt/README.md` に 1 行)。ほかに、2 つ目のフォームから戻れない、accessible モードで description が出ない、"the main branch" の言い方、self-review の L2・L5(PR の Known gaps)。
