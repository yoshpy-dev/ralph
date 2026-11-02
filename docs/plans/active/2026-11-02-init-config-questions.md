# init-config-questions

- Status: Approved
- Approved: 2026-11-02 sha256:d1e86274a18e
- Owner: Claude Code
- Date: 2026-11-02
- Related request: ralph init: 使う CLI や自動の merge などを聞いて ralph.toml に書く
- Related issue: #277
- Type: feat
- Branch: feat/init-config-questions

## Objective

`ralph init` の対話で、プロジェクト名と言語パックのあとに、使う AI の CLI、計画を別の AI に見せるか、変更を別の AI にレビューさせるか、PR ごとにテストを走らせる CI があるか、テストの通った PR を聞かずに merge するかを、用語を使わずに聞く。答えは、init が書く `ralph.toml` の該当の key の値として書く。`--yes` は何も聞かず、雛形のままの `ralph.toml` を書く。

## Scope

- S1 答えを `ralph.toml` の文に反映する関数(新しい `internal/cli/init_settings.go`)
  - 答えの型 `initSettings`(使う CLI: `claude`・`codex`・`both`、計画の確認、変更のレビュー、CI、自動 merge)
  - 雛形の `ralph.toml` のバイト列と答えから、書く内容を返す関数。`[section]` の見出しから次の見出しまでの範囲で、コメントでない `key = value` の行の値だけを替える(`plan_advisory` と `cross_review` は `[standard]` と `[org.review]` の両方にあるので、section で区切って引く)。コメントと並びはそのまま残す
  - 使う CLI が 1 つのときは、`[org].model_pool` の配列の中の、もう一方の driver の行を `#` でコメントにする(あとで CLI を足す人が戻しやすいように、消さない)
  - 替える key か `model_pool` の配列が見つからないときはエラーを返す
- S2 `executeInit` が答えを書く(`internal/cli/init.go`)
  - `initConfig` に答え(`*initSettings`、nil なら雛形のまま)を足す
  - 書く内容は描き出しの前に計算し、エラーなら何も書かずに止める
  - `ralph.toml` は最初の書き込みから答えを反映した内容で書く。`scaffold.RenderFS` に渡す雛形の FS を、`ralph.toml` だけ答えの内容を返す薄い包み(`ReadFile` を持つ `fs.FS`)にして、雛形のまま書いてから書き直す順を作らない。雛形のまま書いたあとで描き出しが失敗すると(たとえば対象の `scripts` が普通のファイル)、既定の `auto_merge = true` などが残り、やり直しの init は `ralph.toml` があるので問いを出さず、答えが消えるため(計画助言の HIGH)
  - `ralph.toml` を作ったか上書きしたとき(Created か Overwritten)、「答えを書いた」1 行を出す
  - manifest の `ralph.toml` には、`TemplateHash` に雛形そのもののハッシュ、`DiskHash` に書いた内容のハッシュを残す(`SetFileOwned`、seed)。upgrade の seed の判定は `TemplateHash` と新しい雛形のハッシュだけを比べる(`internal/upgrade/replaceplan.go:307-317`)ので、書いた内容のハッシュを入れると、雛形が変わっていなくても init 直後の upgrade で advisory が出て、収束しない(`internal/cli/upgrade_v2.go:258-261`)
- S3 問いと 2 段のフォーム(`internal/cli/init.go`、`init_settings.go`)
  - 1 つ目のフォーム: 今の group(名前、言語パック)のあとに、使う CLI の問い(1 つを選ぶ)の group を足す
  - 2 つ目のフォーム: 1 つ目の答えから組み立てる。CLI が両方なら計画の確認・変更のレビュー・CI・自動 merge の 4 問、1 つなら CI と自動 merge の 2 問
  - `ralph.toml` が対象のディレクトリに既にあり `--force` がないときは、CLI の問いも 2 つ目のフォームも出さず、「ralph.toml があるので設定は聞かない」と 1 行で知らせ、答えを渡さない(nil)。manifest を見て聞かずに止める今の扱い(`init.go:130-137`)と同じ考え
  - 問いの組み立ては関数に分けて、テストから accessible モードで答えを流せるようにする
  - `init` の help(`Long`)と `README.md` のコマンドの表の `ralph init` の行に、新しい問いと、`ralph.toml` があるときは聞かないことを書く

### 答えと書く設定

| 問い(英語、用語なし) | 答え | 書く設定 |
|---|---|---|
| Which AI coding CLI will you use? | Claude Code | `[org].driver_pool = ["claude"]`、`model_pool` の codex の行をコメントに、`[doctor].require_claude_cli = true`、`require_codex_cli = false` |
| | Codex | `[org].driver_pool = ["codex"]`、`model_pool` の claude の行をコメントに、`[doctor].require_codex_cli = true`、`require_claude_cli = false`、`[org.watchdog].watcher_enabled = false` |
| | Both | `driver_pool` と `model_pool` は雛形のまま、`require_claude_cli = true`、`require_codex_cli = true` |
| Have the other AI check each plan before work starts?(Both のときだけ) | Yes / No | `[standard].plan_advisory` と `[org.review].plan_advisory` に同じ値。1 つのときは false |
| Have the other AI review each change before its pull request?(Both のときだけ) | Yes / No | `[standard].cross_review` と `[org.review].cross_review` に同じ値。1 つのときは false |
| Does CI run your tests on every pull request? | Yes / No | `[org.gate].regate_checks` を `"ci"` / `"local"` |
| Merge pull requests whose checks pass without asking you? | Yes / No | `[org.director].auto_merge` |

文言は実装で整えてよいが、下の禁止語を使わない。

### 対話の初期値

- CLI: PATH に `claude` と `codex` のどちらがあるかを見て、片方だけならそれを、両方あるかどちらもないなら Both を最初の選択にする。見つかった CLI の選択肢(両方なら Both)に「(Recommended)」を添える(言語パックの問いと同じ扱い)
- 計画の確認: No、変更のレビュー: No、CI: No、自動 merge: Yes(雛形の既定と同じ)

## Non-goals

- `--yes` で答えを指定する flag(`--cli` など)
- 承認方式(`[org.approval].mode`)を聞くこと(issue で決めたとおり `per-plan` のまま)
- 既にある `ralph.toml` を書き換えること、`ralph.local.toml` に書くこと
- `ralph upgrade` で問いを出すこと
- 雛形の `ralph.toml` と `config.Default()` の既定値を変えること
- `[reviewer]`、`[org.watchdog].watcher_model` など、表にない key を問いから変えること(watcher_enabled を除く)

## Assumptions

- 雛形の `ralph.toml` で替える key は、すべて section の中の 1 行の `key = value` で書かれている(`templates/base/ralph.toml` を読んで確かめた)。`model_pool` だけが複数行の配列で、1 行に 1 つの `{ driver = "...", model = "..." },`
- `config.Load` は、`driver_pool` を狭めて `model_pool` に他の driver の行が残ると読み込みを拒む(`internal/config/config.go:886` 以降)。行をコメントにすれば読める
- huh v2.0.3 の accessible モード(`TERM=dumb`)は、group の `WithHideFunc` を見ずに全部の field を聞く(`form.go:720-736`)
- `ralph doctor` は claude と codex のどちらも使えなければ、require の値によらず fail にする。Both で両方 true にしても、新しい失敗は「片方がないとき」だけ増える
- watcher は `[org.watchdog].watcher_model` によらず claude を呼ぶ(`internal/org/watcher.go:295`)。呼べないときは失敗の receipt を残して判断なしで続ける

## Affected areas

- `internal/cli/init.go`(`initConfig`、`runInitInteractive`、`executeInit`、help)
- `internal/cli/init_settings.go`(新規: 答えの型、書き換え、問いの組み立て、PATH の検出)
- `internal/cli/init_settings_test.go`(新規)
- `README.md`(コマンドの表の `ralph init` の行)
- 読むだけ: `templates/base/ralph.toml`、`internal/config/config.go`、`internal/scaffold/manifest.go`、`internal/upgrade/replaceplan.go`、`internal/cli/doctor.go`

## Visual review

- ページ: `.harness/state/plan-visual/init-config-questions.html`(やさしい図 2 枚: overview、easy-skip。たたみの中に detail-overview、detail-write、detail-keys)
- lint: `./scripts/plan-visual.sh lint` exit 0
- self-check: 全体、overview、easy-skip、detail-overview、detail-write、detail-keys の PNG を読んだ。detail-overview の `templates/base/ralph.toml` の箱が文字に足りず、幅を広げて撮り直した。計画助言の反映(D10)のあと detail-write を「包んだ FS で最初から答えの内容を書く」流れに直し、lint と撮り直しをした。ほかは崩れなし

## Design decisions

Critical forks: なし(下の分かれ目は consult `consult-plan-initq` に聞いたうえで推奨案を選んだ。ユーザーの指示により AskUserQuestion は出さない)。

- D1 書き方: 埋め込みの雛形の `ralph.toml` を行で書き換える。TOML に encode し直すとコメントが消え、雛形の説明が読めなくなる。雛形に placeholder を置くと、ルートの `ralph.toml` と `internal/config/defaults_sync_test.go` が読む雛形が壊れる。行の書き換えが将来の雛形の変更で壊れる危険は、実物の雛形に全組み合わせを当てるテスト(S1)と、key が見つからなければ何も書かずに止めること(S2)で抑える
- D2 フォームの形: 2 つに分ける。1 つのフォームで 2・3 問目の group を `WithHideFunc` で隠すと、accessible モードで隠した問いも聞かれる。2 つ目のフォームから 1 つ目に戻れないのは受け入れる(既知の穴に書く)
- D3 問いの言語: 英語。今のフォーム(Project name、Language packs)と CLI の出力が英語で、issue の日本語の問いは「例」。受け入れ条件は用語が出ないことなので、禁止語の一覧(大文字小文字を区別せず、語として): cross-review、cross review、crossreview、regate、gate、advisory、driver、pool、director、leader、seat、org、watcher、per-plan、toml。問いの title、description、選択肢の表示をテストで当てる
- D4 初期値: 上の「対話の初期値」。Both を選ぶと issue の「選んだ側を true」に従い require を両方 true にするので、Enter だけで進めた結果は `--yes`(両方 false)と同じにならない。PATH で初期値を決めるので、両方とも見つからないときだけ Both が初期値で require が両方 true になるが、そのとき doctor はもともと fail にする
- D5 `ralph.toml` が既にあるとき: `--force` がなければ設定の問いを出さない(consult の助言で、聞いてから「書かない」と言う案から変えた)。`--force` なら聞いて、上書きしたあとに答えを書く
- D6 manifest: `TemplateHash` は雛形のハッシュ、`DiskHash` は書いた内容のハッシュ(S2 の理由)
- D7 `--yes`: 書き換えの関数を通さず、雛形のバイト列のまま書く
- D8 codex だけのとき `[org.watchdog].watcher_enabled = false` を書く(consult が見つけたリスク)。watcher は claude しか呼ばないので、codex だけの project では毎回失敗の receipt を残すだけになる。issue の表にない key だが、CLI の答えから決まり、off にしても見張りの時間の層(pulse)は動く
- D9 model_pool の他の driver の行は消さずにコメントにする(consult の助言)
- D10 答えは `ralph.toml` の最初の書き込みに入れる(計画助言の HIGH)。雛形を書いてから書き直す順では、間で init が失敗すると既定の値が残り、やり直しで問いが出ないため答えが消える。描き出しの FS を包んで `ralph.toml` だけ答えの内容を返す。書き込みは今のほかのファイルと同じ `os.WriteFile` で、一時ファイルと rename による原子的な書き込みにはしない(途中で電源が落ちたときの切れたファイルは、`ralph doctor` が parse error として知らせる。init のほかのファイルも同じ扱い)

## Acceptance criteria

- [x] AC1 CLI に Both を選んだとき、対話で 5 問(CLI、計画の確認、変更のレビュー、CI、自動 merge)が聞かれ、答えが「答えと書く設定」の表のとおり `ralph.toml` に書かれる
- [x] AC2 CLI に 1 つを選んだとき、計画の確認と変更のレビューの 2 問が出ず、`[standard]` と `[org.review]` の `plan_advisory` と `cross_review` が false で書かれる。codex だけなら `[org.watchdog].watcher_enabled = false` も書かれる
- [x] AC3 問いの title、description、選択肢の表示に、Design decisions D3 の禁止語が出ない(テストで固定)
- [x] AC4 `--yes` では、雛形と同じバイト列の `ralph.toml` が書かれる
- [x] AC5 書かれた `ralph.toml` を `ralph doctor` が読める: 実物の雛形に答えの全組み合わせ(CLI 3 × 4 つの yes/no、CLI が 1 つのときは 2 問が off に決まる)を当てた結果を `config.Load` が読み、値が表のとおり。CLI ごとに、init が書いた `ralph.toml` で `runDoctorOpts` を回し、`ralph.toml` の parse error の warn 行が出ず、その CLI だけの偽の binary で doctor が fail しない(テストで固定)
- [x] AC6 `ralph.toml` が既にあり `--force` がないとき、設定の問いを出さず 1 行で知らせ、`ralph.toml` を変えない。`--force` なら聞いて、答えを書く
- [x] AC7 答えを書いた init の直後の `ralph upgrade --dry-run` で、`ralph.toml` の seed の advisory が出ない(manifest の `TemplateHash` は雛形のハッシュ、`DiskHash` は書いた内容のハッシュ)
- [x] AC8 書き換える key か `model_pool` の配列が雛形にないとき、init は何も書かずにエラーで止まる
- [x] AC9 `init` の help と `README.md` の `ralph init` の行が、新しい問いと、`ralph.toml` があるときは聞かないことを説明する
- [x] AC10 `ralph.toml` を書いたあとで描き出しが失敗しても、`ralph.toml` には答えが入っている。答えなしのやり直し(`ralph.toml` があるので問いを出さない)が成功したあとも答えが残り、manifest の `TemplateHash` は雛形のハッシュ(テストで固定: 対象の `scripts` を普通のファイルにして 1 回目を失敗させ、消してからやり直す)

## Implementation outline

1. S1 `initSettings` と、雛形の `ralph.toml` に答えを反映する関数、そのテスト(実物の雛形 `templates/base/ralph.toml` を読み、全組み合わせを `config.Load` に通して値を確かめる、コメントの行が残る、key がないとエラー、section をまたいで同じ名前の key を替えない)。AC2(値の部分)、AC5(Load の部分)、AC8(関数のエラー) — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'InitSettings'`
2. S2 `executeInit` が答えを `ralph.toml` の最初の書き込みに入れ(描き出しの FS を包む)、manifest に 2 つのハッシュを残す。テスト: nil なら雛形と同じバイト列、答えがあれば表のとおり、既にある `ralph.toml` は Skipped のまま変えない、key がない雛形では何も書かずにエラー、init 直後の upgrade の dry-run に `ralph.toml` の advisory が出ない、CLI ごとの `runDoctorOpts`、描き出しの失敗とやり直しで答えが残る。AC4、AC5(doctor の部分)、AC7、AC8、AC10 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'InitSettings|ExecuteInit|RunInitInteractive|RunDoctorOpts'`
3. S3 CLI の問い、2 つ目のフォーム、`runInitInteractive` の配線、help と README。テスト: accessible モードで 2 つ目のフォームに答えを流して、Both なら 4 問、1 つなら 2 問が聞かれ答えが入る、PATH の検出と「(Recommended)」の表示と初期値、禁止語、`ralph.toml` があり `--force` がないとき問いを出さない判定、TERM=dumb の既存のテストが通る。AC1、AC2(問いの部分)、AC3、AC6、AC9 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Init|PackQuestion|CLIQuestion|SettingsForm'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck)
- Spec compliance criteria to confirm: AC1〜AC9。issue の受け入れ条件 5 つとの対応(AC1↔1、AC2↔2、AC3↔3、AC4↔4、AC5↔5)
- Documentation drift to check: `README.md` の `ralph init` の行、`init` の help、`AGENTS.md` の地図(`internal/cli/` の行に init の新しいファイルが要るか)、`docs/recipes/` で `ralph init` の問いを説明している文
- Evidence to capture: `docs/reports/verify-*.md`

## Test plan

- Unit tests: S1 の書き換え(全組み合わせ、コメントの保持、section の区切り、エラー)、S3 の問いの組み立て(検出、初期値、表示、禁止語、2 つ目のフォームの問いの数)
- Integration tests: `executeInit` に答えを渡して書かれる `ralph.toml` と manifest、init のあとの `runDoctorOpts`、init のあとの upgrade の dry-run
- Regression tests: `--yes` の雛形のバイト列(AC4)、既存の `TestRunInitInteractive_DumbTermWithoutTerminalStopsBeforeTheForm`、言語パックの問いのテスト、`internal/config/defaults_sync_test.go`(雛形は変えないので通るはず)
- Edge cases: `ralph.toml` が既にある(`--force` あり・なし)、key を欠いた雛形、codex だけ(watcher_enabled)、PATH に CLI がどちらもない・片方だけ・両方
- Evidence to capture: `docs/reports/test-*.md`(`./scripts/run-test.sh` の結果)

## Risks and mitigations

| リスク | 影響 | 抑え方 |
|---|---|---|
| 雛形の key の並びや書き方が将来変わり、行の書き換えが当たらない | init が答えを書けない | S1 のテストで実物の雛形に全組み合わせを当てる。実行時は key が見つからなければ何も書かずにエラー(AC8) |
| accessible モードで、聞かないはずの問いが出る | CLI が 1 つでも 2 問が出る | 2 つのフォームに分ける(D2)。テストは accessible モードで流す |
| manifest の `TemplateHash` に書いた内容のハッシュが入る | init 直後の upgrade が毎回 advisory を出して収束しない | D6、AC7 のテスト |
| Both を選んだ人の PATH に片方がない | doctor が fail する | issue の決定どおり。PATH で初期値を決めるので、Enter だけで進めて起きるのは両方ないときだけで、そのとき doctor はもともと fail |
| `--force` で `ralph.toml` が symlink | 書き直しがリンクの先に書く | RenderFS の `--force` も同じくリンクをたどって書くので、扱いは変わらない |
| CI に Yes と答えたのに GitHub や gh がない | `ralph org regate` が CI の結果を読めない | 問いの description で「PR ごとに CI がテストを走らせる」場合に Yes と書く。regate のエラーは既存のまま(#264) |
| 2 つ目のフォームから 1 つ目に戻れない | 言語パックを選び直すには Ctrl-C でやり直す | 既知の穴として PR に書く |
| 雛形を書いたあとで init が失敗し、やり直しで問いが出ない | 「自動 merge しない」などの答えが消え、既定の値で動く | D10: 答えは `ralph.toml` の最初の書き込みに入れる。AC10 のテスト |
| `ralph.toml` の書き込みの途中で電源が落ちる | 切れたファイルが残り、やり直しで問いが出ない | init のほかのファイルと同じ扱い。`ralph doctor` が parse error を知らせる(D10) |

## Rollout or rollback notes

- release のノートに「`ralph init` の対話が、使う CLI など 5 問(CLI が 1 つなら 3 問)を聞いて `ralph.toml` に書く。`--yes` と、`ralph.toml` が既にある project では聞かない」と書く
- 戻すときは、この PR の revert で済む。書かれた `ralph.toml` は普通の設定のファイルなので、戻したあとも読める

## Open questions

- なし

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh)。スイッチは PATH の ralph 6.0.0 が `unknown command "config"`(exit 1)を返したので、この worktree の `go run ./cmd/ralph config get standard.plan_advisory` で読み直して true(#267・#271 と同じ扱い)。git の状態は前後で同じ。所見 1 件: [HIGH] 雛形を書いてから答えで書き直す順では、描き出しが間で失敗すると既定の `auto_merge = true` が残り、やり直しで問いが出ず答えが消える → 反映(S2 を最初の書き込みに答えを入れる形に、D10、AC10、リスクの 2 行)。原子的な書き込みにする助言は採らず、理由を D10 に書いた
- [x] Plan approved — 自動承認(ユーザーの指示)。digest sha256:d1e86274a18e(2026-11-02)。図解ページ `.harness/state/plan-visual/init-config-questions.html`
- [x] Branch created(feat/init-config-questions、worktree `.claude/worktrees/init-config-questions`、main c8c8c0f1 から)
- [x] Implementation started — branch は worktree の記録どおり `feat/init-config-questions`(`branch-name.sh from-plan` は issue 番号つきの `feat/277/init-config-questions` を返すが、どちらも validate を通り、計画と worktree の記録はこちら)。cycle 1
- スライス:
  - S1 82956f18(impl-277-s1): `applyInitSettings` と 12 のテスト(48 通りを `config.Load`)。検証 rc=0(2 回)、変異 14 個すべて検出。handoff からの違い: `[[name]]` も見出しに数える、1 行の値だけ替える(複数行ならエラー)、選んだ CLI の行が model_pool に 1 つもなければエラー、他の driver の行は `{` で始まり `driver = "<x>"` を含む行
  - S2 f306d479(impl-277-s2): `initAnsweredFS` で最初の書き込みから答えを入れ、manifest に 2 つのハッシュ。検証 rc=0、変異 6 個すべて検出。handoff からの違い: 既存の manifest の確かめを `MkdirAll` の前に動かした(既存の project の結果は変わらない)。答えを返すのは `ReadFile` だけで、`Open` は雛形のまま(RenderFS は `fs.ReadFile` で読む。テストで固定)
  - S3 d7316e5b(impl-277-s3): CLI の問い、2 つ目のフォーム、`askInitSettings`、help と README。検証 rc=0、変異 8 個すべて検出。実機の pty で TERM=dumb と全画面の両方を確かめた。accessible モードは description を出さない(huh の制約)
  - 自己点検の 5 項目はどのスライスも clean(S3 の一時ファイル zz_debug_s3_test.go はコミット前に消した)
  - AC1〜AC10 に印を付けたとき、python で計画の印を書き換えた(Edit を使う決まりに反する。中身は `- [ ]` を `- [x]` にしただけで、digest は d1e86274a18e のまま)
- self-review(cycle 1): Run 1 は LOW 5 件(Merge yes、a733d834)。L1・L3・L4 を S4 1af26382(impl-277-s4)で直し、Run 2 は所見 0(Merge yes、61caf1d3)。L2(`executeInit` が 171 行に伸びた)は直さない: 分けると今回の範囲外の作り直しになり、伸びたのは答えの計算と manifest の分岐で、どちらもコメントつきで読める。L5(codex だけのときの `watcher_enabled = false` とコメントにした model_pool の行を出力で説明しない)は直さない: 書いた `ralph.toml` の各 key のコメントが意味を説明し、コメントにした行はそのまま読める
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
- 計画の分かれ目: consult `consult-plan-initq` の判定は「進めてよい(D5 だけ変えて)」。D5 を「`ralph.toml` があり `--force` なしなら問いを出さない」に変え、D8(codex だけなら watcher_enabled = false)と D9(model_pool の行はコメントに)を足した
