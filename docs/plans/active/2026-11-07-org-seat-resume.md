# org の座席を API エラーから続ける(`ralph org resume` と再試行の設定)

- Status: Approved
- Approved: 2026-11-07 sha256:0893ac30252e
- Owner: Claude Code
- Date: 2026-11-07
- Related request: docs/specs/2026-11-07-org-seat-resume.md(2026-11-07 に人が承認した仕様)
- Related issue: N/A
- Type: feat
- Branch: docs/spec-org-seat-resume

## Objective

通信断や利用上限で止まった herdr の座席を、人が director に「続けて」と言うだけで続けられるようにする。claude の座席には Claude Code の再試行の設定を足して止まる回数を減らし、それでも止まった座席は新しい動詞 `ralph org resume` が決まった規則で見つけて続行の文を打つ。仕様の FR-1〜FR-11 と AC-1〜AC-13 をこの計画で満たす。

## Scope

- `[org].api_wait_minutes`(既定 30、0 で無効、負は断る)と、driver が claude の座席と headless の director の環境への 3 つの変数(`CLAUDE_CODE_RETRY_WATCHDOG=1`、`CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS`、`CLAUDE_CODE_MAX_RETRIES`)
- 30 分に見合う `CLAUDE_CODE_MAX_RETRIES` の回数の実測と、分から回数への換算
- 画面の判定(止まった/Enter だけ/待ち中/一覧にない)と、文の一覧の 1 ファイル
- `ralph org resume`(`--org-id`、`--dry-run`、`--state-dir`)と manifest の `resumed`
- 座席の役割 prompt(leader、implementer、reviewer)の 1 行
- director と org の skill(4 つの写しずつ)、org runtime の仕様、AGENTS.md、テンプレートの `ralph.toml`
- 分離した herdr での実機の確認

## Non-goals

- 見張りが自分で resume を打つこと(段階 2 の候補)
- 上限のリセット時刻の解釈
- 再試行中の座席を `ralph status` に出すこと
- Claude Code の StopFailure の hook を使った検知
- promoted の director 自身の環境と、`launch_cmd` の director の pane への resume
- codex の再試行の設定の変更

## Assumptions

- herdr は 0.7.5。エラーでターンを終えた claude の pane は `idle` と報告される(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 2)。
- `herdr agent get` の出力は JSON の envelope で、`.result.agent.agent_status` に状態がある(同じ記録の 2〜4)。
- Claude Code は v2.1.295 以降で `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS` を読む。手元は 2.1.312。
- 既存の座席の環境は変わらない。新しい変数は、この版で spawn した座席から入る。

## Affected areas

- `internal/config/config.go` と既定値の一致のテスト(`internal/config/defaults_sync_test.go` ほか)、`templates/base/ralph.toml`
- `internal/org/driver/paneenv.go`(claude の再試行の変数を作る関数)
- `internal/org/spawn.go`(座席の pane の作成に driver が claude のときだけ変数を足す)
- `internal/org/director_start.go`(herdr の中の headless の director)、`internal/org/director_cmd.go`(`launch_cmd` の起動スクリプト)
- 新しいファイル: `internal/org/resume_screens.go`(文の一覧と画面の判定)、`internal/org/resume.go`(`Org.Resume`)、`internal/org/testdata/resume/`(実機の画面)
- `internal/org/seat.go` か `verbs.go`(`EventResumed`)
- `internal/cli/org_resume.go`(新しい cobra の動詞)、`internal/cli/org.go`(登録と `--org-id` の説明)
- `internal/org/prompts/leader.md`、`implementer.md`、`reviewer.md` と役割 prompt のテスト
- `.claude/skills/director/SKILL.md`、`.claude/skills/org/SKILL.md` と、その `.agents/` と `templates/base/` の写し
- `docs/specs/2026-08-01-org-runtime.md`、`AGENTS.md`、`docs/specs/2026-11-07-org-seat-resume.md`(FR と AC の印)
- 新しい記録: `docs/evidence/org-seat-resume-retries-2026-11-07.md`(回数の実測)、`docs/evidence/org-seat-resume-live-2026-11-07.md`(実機の確認)

## Visual review

- Page: `.harness/state/plan-visual/org-seat-resume.html`(やさしい図 3 枚: overview、easy-wait、easy-target。折りたたみの中に detail-overview、detail-sequence、detail-classify、detail-env)
- Lint: `./scripts/plan-visual.sh lint` exit 0
- Self-check: 全体、overview、ほかの 6 枚をそれぞれ `shot` で撮って読んだ。ファイルの地図の 2 か所のはみ出し、待ちの設定の図の ✓ と ✕ の線との重なり、冒頭の説明文の文字のままのバッククォートを直し、撮り直して問題がないことを確かめた

## Design decisions

Critical forks: None。仕様で決まらなかった点は、ユーザーの方針(2026-10-11、分かれ目は自分で選ぶ)に従い、自律で次のとおり選んだ。

- D1: 1 本の PR にする。予防の変数(S1〜S3)と resume(S4〜S6)は独立して動くが、どちらも小さく、director の手順書(S8)が両方を前提に書くため。分けると手順書を 2 回直すことになる。
- D2: resume の打ち方は `Send` の打ち方(`idle`/`done` の待ち、本文、750ms、Enter を 1 回、送信の確認、`SendProgress`)を共有する。`Send` から打つ部分を内部の関数に切り出し、`Send` と resume の両方が呼ぶ。protocol の検査、watch gate、計画の承認の確認は `Send` の側だけに残す。記録するイベントは resume では `sent` ではなく `resumed` にする。打ち方を 2 か所に書くと、Enter の扱いの直しが片方に漏れるため。
- D3: `resumed` は状態を変えないイベント(`isStateEvent` が false)にする。座席の roster の状態と stall の時刻(`latestSeatEventTS` は種類を問わない)は、`sent` と同じ扱いになる。
- D4: 分から `CLAUDE_CODE_MAX_RETRIES` への換算は、S2 で測った立ち上がりの時間と 1 回あたりの間隔から決める式にし、`driver/paneenv.go` に置く。式と実測の値は記録と関数の doc comment に書く。この換算はすぐ失敗する通信断(名前解決の失敗、接続の拒否)の目安で、応答が返らない通信断では 1 回の再試行が `API_TIMEOUT_MS`(既定 10 分)近くかかるので長くなりうる(計画助言の指摘 2、仕様の FR-8 の改訂)。`API_TIMEOUT_MS` は遅いネットワークでの正しい応答も切るので変えない。
- D8: 打つ直前の割り込みを pane ごとのロックで防ぐ(計画助言の指摘 1、仕様の FR-5a)。ロックは台帳の state dir の下の pane ごとのファイル(既存の `acquireHeldFileLock` を使う)で、`Send` と resume の両方が、`idle`/`done` の待ちのあとに取り、Enter を押し終えるまで持つ。取れないまま待ちの期限が来たら、何も打たずに失敗する(resume では「飛ばした(ほかの送信中)」)。resume はロックの中で `pane read` と判定をやり直し、「止まった」でなければ打たない。文を打ったあと Enter の前に `pane read` で入力欄に文の先頭があることを確かめ、なければ Enter を押さない。`Send` はロックで直列になるだけで、判定や確かめは足さない(打つ文は呼び出し側が決めるため)。
- D5: 画面の判定は、`herdr pane read --lines 60` の text を使う。60 行は、記録の 2 の画面(30 行)に、待ちの行と長いエラー文の折り返しの余裕を足した値。
- D6: `ralph org resume` は `--org-id` を任意にする。`org.go` の `--org-id` の説明に resume を足す。
- D7: 終了コードは仕様の FR-7 のとおり(送信の失敗か台帳を読めないとき 1、それ以外 0)。判定で飛ばした座席だけなら 0。

## Acceptance criteria

仕様の AC をそのまま使う。

- [ ] AC-1: エラーで止まった画面(記録の 2)で `idle` のとき、続行の文を打って Enter を 1 回押し、`resumed` を 1 件記録し、結果は「続けた」
- [ ] AC-2: 再試行中の画面(記録の 3)は `working` でも `idle` でも打たず、結果は「待ち中」
- [ ] AC-3: codex のフック確認の 3 択(記録の 4)で `idle` のとき打たず、結果は「飛ばした(一覧にない)」
- [ ] AC-4: `Press enter to continue` には Enter だけを 1 回送る
- [ ] AC-5: 呼び出したプロセスの `HERDR_PANE_ID` の座席は飛ばす
- [ ] AC-6: `--dry-run` は `send-text` と `send-keys` を呼ばず、何も記録しない
- [ ] AC-7: implementer に打ってから leader に打つ
- [ ] AC-8: 既定の設定で claude の座席の pane の作成に 3 つの変数が入り、codex の座席には入らない。`api_wait_minutes = 0` なら入らない。headless の director にも driver ごとに同じ規則で入る
- [ ] AC-9: `api_wait_minutes` が負なら設定の読み込みが断る。既定値はテンプレートと一致する
- [ ] AC-10: 分離した herdr で、claude の座席を通信断で止め、戻したあと resume で作業が最後まで進む(記録を残す)
- [ ] AC-11: 30 分に見合う回数を実測し、記録と計画に書く
- [ ] AC-12: director の手順書に手順と禁止があり、skill の写しとテンプレートの一致の検査が通る
- [ ] AC-13: 3 つの役割 prompt に続行の指示の扱いの 1 行があり、テストが確かめる
- [ ] AC-14: 同じ pane への 2 つの resume(か resume と `Send`)は 1 つずつ打ち、2 つ目はロックの中で読み直して「止まった」でなければ打たない。判定のあと承認画面に変わったら文も Enter も送らない。打ったあと入力欄に文がなければ Enter を送らない
- [ ] AC-15: 分離した herdr で `api_wait_minutes` を小さくして claude の座席を接続の拒否で止め、ターンを終えるまでの時間と stall の ALERT が届くまでの時間を測って記録する

## Implementation outline

1. S1 設定キー: `[org].api_wait_minutes` を `config.go` に足し(既定 30、負は `validate` で断る)、`templates/base/ralph.toml` の `[org]` にコメントつきで書き、既定値の一致のテストに足す。リポジトリの `ralph.toml` が `[org]` の既定値を写しているなら同じく足す。 — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. S2 回数の実測: `CLAUDE_CODE_RETRY_WATCHDOG=1` と `ANTHROPIC_BASE_URL=https://ralph-probe.invalid` で `claude -p --model haiku` を `CLAUDE_CODE_MAX_RETRIES` の 2 つの値(例: 10 と 20)で並べて走らせ、終わるまでの時間を測る。立ち上がりの時間と 1 回あたりの間隔を出し、30 分に見合う回数と換算の式を `docs/evidence/org-seat-resume-retries-2026-11-07.md` に書く。作業ファイルは `/tmp` か `$TMPDIR` に置く。 — verify: `test -s docs/evidence/org-seat-resume-retries-2026-11-07.md && grep -q 'CLAUDE_CODE_MAX_RETRIES' docs/evidence/org-seat-resume-retries-2026-11-07.md`
3. S3 pane の環境: `driver/paneenv.go` に分から 3 つの変数を作る関数(0 なら空)を足す。`spawn.go` は driver が claude の座席にだけ、`PrimaryCLIEntry` のあとの extraEnv として足す。`director_start.go`(herdr)と `director_cmd.go`(起動スクリプトの export)は、director の driver が claude のときだけ足す。warnNoPaneEnv の結果の文(`paneEnvConsequence`)にも 3 つの項を足す。 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/... && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/driver/ ./internal/org/ -run 'PaneEnv|Spawn|DirectorStart|DirectorCmd|Launch'`
4. S4 画面の判定: `internal/org/resume_screens.go` に driver ごとの文の一覧(止まった/Enter だけ/待ち中、各項目に出どころ)と、画面の text から種類と照合した行を返す関数を書く。折り返しをつなぎ、最後の入力欄より下の footer を読み飛ばし、入力欄が空で直前のまとまりが一覧の文のとき(状態行 1 行まで許す)だけ「止まった」にする。`internal/org/testdata/resume/` に記録の 2〜4 の画面と、ドキュメントの文から作った画面を置く。 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'ResumeScreen'`
5. S5 動詞 `Org.Resume`: `internal/org/resume.go`。対象(動いている座席、`--org-id` がなければ pane のある headless の director も)、順(leader 以外 → leader → director)、自分の pane を飛ばす、`herdr agent get` の状態、`pane read`、S4 の判定、`Send` から切り出した打ち方(D2)、pane ごとのロックとロックの中の読み直し、Enter の前の入力欄の確かめ(D8。`Send` にも同じロックを入れる)、Enter だけの道、`EventResumed`、`--dry-run`。1 つの座席の失敗で残りを止めない。 — verify: `test -z "$(gofmt -l internal/org)" && go vet ./internal/org/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Resume|Send'`
6. S6 CLI: `internal/cli/org_resume.go` と `org.go` の登録、`--org-id` の説明、座席ごとの結果の行、終了コード(D7)、help。 — verify: `test -z "$(gofmt -l internal/cli)" && go vet ./internal/cli/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/cli/ -run 'Resume|OrgHelp|Org'`
7. S7 役割 prompt: leader、implementer、reviewer の「スター型トポロジのルール」の近くに、pane に直接入った `ralph org resume` からの続行の指示は止まったところから作業を続ける指示として扱う 1 行を足し、テストで確かめる。 — verify: `GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'RenderRolePrompt'`
8. S8 skill: director の skill に「座席を続ける」の手順(人の頼み → resume → 報告)、ALERT の件で画面が API エラーのときの伝え方、「してはいけないこと」の追記を書く。org の skill の動詞リファレンスに resume を足す。`./scripts/sync-skills.sh` で `.agents/` を作り直し、`templates/base/` の写しをそろえる。 — verify: `./scripts/check-skill-sync.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/org/ -run 'Skill'`
9. S9 文書: org runtime の仕様に改訂の節と resume の動詞を足し、`AGENTS.md` の `internal/org/` の行に resume と画面の判定のファイルを足す。 — verify: `./scripts/check-sync.sh && ./scripts/run-static-verify.sh`
10. S10 実機の確認: `docs/recipes/isolated-herdr-server.md` の分離した herdr と scratch の repo で、claude の座席を `HTTPS_PROXY` に向けたローカルのプロキシ経由で動かし、プロキシを止めて座席を止め(`api_wait_minutes = 0`)、戻してから `ralph org resume` を打ち、座席が作業を終えるのを見る。`--dry-run` の出力、manifest の `resumed`、待ち中の座席を飛ばすことも見る。続けて `api_wait_minutes` を小さく(例: 2)、`[org.watchdog].stall_minutes` を小さくして同じように止め、座席がターンを終えるまでの時間と、見張りの stall の ALERT が届くまでの時間を測る(AC-15)。`docs/evidence/org-seat-resume-live-2026-11-07.md` に書き、ユーザーの herdr には触れない。 — verify: `test -s docs/evidence/org-seat-resume-live-2026-11-07.md`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、staticcheck、skill と template の一致)
- Spec compliance criteria to confirm: 仕様の FR-1〜FR-11 と非機能要件、AC-1〜AC-13 を、テストの名前と記録で 1 つずつ突き合わせる。FR-2 の「herdr の状態だけでは打たない」と FR-9 の「codex には足さない」を特に見る
- Documentation drift to check: director と org の skill の 4 つの写し、`internal/org/prompts/`、`AGENTS.md`、org runtime の仕様、`templates/base/ralph.toml` のコメントと `Default()`
- Evidence to capture: verify の report、S2 と S10 の記録

## Test plan

- Unit tests: 画面の判定(記録の画面、折り返し、footer、状態行、入力欄に文字が残っている、待ち中の 4 つの文、上限の文、`Press enter to continue`、一覧にない選択画面、空の画面)。分から 3 つの変数への換算(0、1、30、境界)。設定の負の値。
- Integration tests: 偽物の Runner で `Org.Resume` を回す(打つ座席・飛ばす座席・待ち中の座席が混ざる org、2 つの org、director あり、`--org-id`、`--dry-run`、自分の pane、`agent get` の失敗、送信の失敗のあとも残りを続ける、順番)。spawn の argv に `--env` が入る/入らない(claude、codex、0)。director start と起動スクリプト。CLI の出力と終了コード。
- Regression tests: `Send` の既存のテスト(切り出しで挙動が変わらない)、watch の stall の既存のテスト、役割 prompt と skill の一致のテスト。
- Edge cases: エラー文のあとに人が入力欄に文字を打ちかけている、herdr が `done` を返す、pane の行数が少ない、同じ座席に 2 回 resume を打つ(2 回目は打たない、送信後は `working` か入力欄に文字がないため)、codex の上限の文。
- Concurrency tests(AC-14): pane のロックを別の呼び出しが持っているあいだ、resume も `Send` も打たずに待ち、期限で何も打たずに終わる。判定のあと、ロックの中の読み直しで画面が承認画面に変わっていれば文も Enter も送らない。打ったあとの読み直しで入力欄に文がなければ Enter を送らない。偽物の Runner で順番を決めて確かめる。
- Evidence to capture: test の report、S10 の実機の記録

## Risks and mitigations

- 一覧の文が CLI の版で変わり、止まった座席を拾えない → 文の一覧を 1 ファイルにまとめ、出どころの版を書く。拾えない座席は「飛ばした(一覧にない)」として結果に出るので、director が人に伝えられる。
- herdr が知らない画面を `idle` と報告し、そこに打つ → 画面の判定で、入力欄が空で直前が一覧の文のときだけ打つ。記録の 4 の画面をテストに入れる。
- `Send` の切り出しで既存の送信が変わる → 切り出しの前後で `Send` の既存のテストをすべて通す。打ち方の関数は振る舞いを変えずに移す。
- `CLAUDE_CODE_MAX_RETRIES` の回数と時間の対応がずれる → S2 で実測し、式と値を記録する。Claude Code の版が変わったら測り直すことを記録に書く。応答が返らない通信断では 30 分を超えうることを仕様と文書に書き、S10 で設定を有効にした止まり方と ALERT までの時間を測る(AC-15)。
- 判定と打つ間に画面が変わる、ほかの送信が割り込む → pane ごとのロック、ロックの中の読み直し、Enter の前の入力欄の確かめ(D8)。`Send` にロックを入れることで既存の送信が待つようになるので、`Send` の既存のテストと、ロックの待ちの期限のテストで確かめる。
- 待ちのあいだ stall の ALERT が出ず、止まっていることに気づきにくい → 上限 30 分で区切り、そのあとは既存の ALERT の経路に乗る。仕様の Out of scope に、待ち中の表示を段階 2 の候補として書いてある。
- 続行の文が座席にデータとして扱われる → 文に出どころを入れ、役割 prompt に扱いの 1 行を足す(S7)。S10 で実際に続くことを見る。

## Rollout or rollback notes

- `api_wait_minutes = 0` で、変数を足さない前の挙動に戻せる。`ralph.local.toml` で手元だけ変えられる。
- `ralph org resume` は新しい動詞で、打たなければ何も変わらない。戻すときはこの PR を revert する。
- 新しい変数は、この版で spawn した座席から入る。release のノートに「動いている org の座席には、spawn し直すまで入らない」と書く。

## Open questions

- 上限(429)に当たったときの claude と codex の画面と状態(仕様の Open questions と同じ。S10 でも当てられない)
- codex の通信待ちの上限と、そのあいだの stall の ALERT(director の手順書に「codex の `waiting for network` の ALERT は待つ」と書くかは S8 で決める)

## Progress checklist

- [x] Plan reviewed — plan advisory: ran with codex(gpt-6-astra、xhigh、rc=0、git の状態は前後で同じ)。指摘 2 件、どちらも聞かずに反映した(ユーザーの方針、2026-10-11)。[HIGH] 1: 判定と打つあいだの割り込みで承認画面に Enter が入りうる → D8(pane ごとのロックを Send と resume で共有、ロックの中の読み直し、Enter の前の入力欄の確かめ)、AC-14、S5、Test plan の Concurrency tests、Risks、仕様の FR-5a と AC-14。[HIGH] 2: 回数で区切る通信断の待ちは 30 分を保証できない(応答が返らない通信断は 1 回の再試行が約 10 分) → D4 に目安であることを書き、仕様の Desired state と FR-8 を改訂、AC-15 と S10 の設定を有効にした試験を足した
- [x] Plan approved — 2026-11-07、ユーザーが「推奨案を自動承認」「マージまで」と伝えたため、承認の確認を出さずに承認した
- [x] Branch created — `/spec` の worktree のブランチ `docs/spec-org-seat-resume` をそのまま使う(state id `spec-org-seat-resume`)
- [x] Implementation started
  - S1 `300b295b`(implementer、検証 ok)。`[org].api_wait_minutes`、既定 30、負は断る
  - S4 `1f298056`(implementer、検証 ok)。画面の判定。既知の穴: 普通の返事の最後のまとまりが `API Error:` を含むと stopped と判定しうる(tech-debt)
  - S2 `b8676ede`(inline、文書だけ)。実測は 10 回 189 秒、20 回 554 秒、40 回 1265 秒。式 N = max(1, 10 + round((m×60 − 189)/36))、30 分は 55 回。機械的な確かめ: 5 項目とも clean
  - S3 `d6a0a6bd`(implementer、検証 ok)。範囲外の `internal/cli` のテスト 3 ファイルは、既定 30 で期待値が変わったための更新(handoff で許可)
  - S5 `de55a4b2`(implementer、検証 ok、`internal/org` 全体も ok)。director の対象は `HerdrLaunch()` と `PaneID` で選ぶ(`HerdrPane` は呼び出し元の場所なので使わない)。director の push の wake はロックを取らない(tech-debt)
  - S6 `2024308f`(implementer、検証 ok)。skipped と failed の行は Line より Reason を出す
  - S7 `8b3e47a1`(implementer、検証 ok)
  - S8 `0b62b48c`(implementer、検証 ok)。新しい節は director の skill の `## 止まった座席を続ける`。計画の Open questions の 2 つ目(codex の `waiting for network` の ALERT)は「待つ」と書いた
  - S9 `387a290a`(implementer、検証 ok、`run-static-verify.sh` も ok)。tech-debt に 7 行
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
