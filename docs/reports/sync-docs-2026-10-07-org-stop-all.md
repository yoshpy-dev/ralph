# sync-docs report: org-stop-all

## Cycle 1

- Date: 2026-10-08(JST。insight event の ts は UTC の 2026-10-07)
- Plan: `docs/plans/active/2026-10-07-org-stop-all.md`
- Pipeline cycle: 1 of 2。差分は base `f423f230`(origin/main)から branch `feat/org-stop-all` の `ad3baa72` まで。この report の変更は `223258eb` の上に積んだ
- 先行 report(それぞれを追加した commit):
  `docs/reports/self-review-2026-10-07-org-stop-all.md`(`a103a05f` で追加、`eb7bf5e2` で S7 後に更新。Merge 可、CRITICAL・HIGH・MEDIUM なし、LOW 8 件 F-3〜F-10 ほか)、
  `docs/reports/verify-2026-10-07-org-stop-all.md`(`de154798`。pass、AC1〜AC15 Met、D-1〜D-4 と V-1〜V-3)、
  `docs/reports/test-2026-10-07-org-stop-all.md`(`ad3baa72`。pass、mutation 81 件中 80 件が red、テスト追加 `f63ae025`)

## Summary

verify が挙げた D-1〜D-4 をすべて直し、そのほかの文書を確かめた。古くなっていたのは `ralph org stop` / `disband` の `--help` の Long、`/org` skill の `disband` の行、plan のチェックボックス、tech-debt の台帳だった。README、仕様 FR-2、leader の雛形、recipe の 2 面、AGENTS.md の Repo map は実装と合っていて、変更していない。

`internal/cli/org.go` の変更は Long の文字列だけで、動作は変えていない。ただし code ファイルなので、self-review と verify の report はこの diff(`223258eb`)を見ていない。`/cross-review` の入力には入る。

## Changes made

| File | Change |
|------|--------|
| `internal/cli/org.go` | D-1。`stop` の Long に、C-c と close の前に pane の tab の label が座席 id、workspace の label が org_id であることを確かめ、確認に失敗した pane(label の不一致、または herdr に尋ねられない)には C-c も close も送らないこと(台帳の id が別の pane を指しているかもしれないため)を足した。`--force` の文に、確認に失敗した pane は閉じず `stopped` を記録するだけであることを足した。`disband` の Long に、pane と workspace を閉じる前の同じ確認と、失敗したものは失敗として数えること、`--force` でも確認に失敗したものは閉じず記録だけすることを足した。D-2 として、`--all` が「古い ralph の disband が workspace を閉じずに残した org」も対象にすることを足した。`ralph org stop --help` と `disband --help` を実際のバイナリで出して読み、折り返しを確かめた |
| `.claude/skills/org/SKILL.md` | D-2。`disband` の行の `--all` の括弧に 1 文「古い ralph の `disband` が workspace を閉じずに残した org も対象になる」を足した。`stop` のセルは触っていない(F-8) |
| `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 上の 3 つの写し。`scripts/sync-skills.sh` は `.agents/skills/org/SKILL.md` だけを作り直したので、`templates/base/` の 2 面は `cp` で揃えた。4 面の md5 は同じ(`766894da0775676b4fecc2ca6204dd9e`) |
| `docs/plans/active/2026-10-07-org-stop-all.md` | D-3。AC1〜AC15 と、Progress checklist の「Review artifact created」「Verification artifact created」「Test artifact created」にチェックを付けた(「PR created」は付けていない)。Progress checklist の末尾に、Verify plan の「AC1〜AC13」は S7 が AC14・AC15 を足す前の記述で、verify は AC1〜AC15 を確かめたことを 1 行足した。本文のほかの行は変えていない |
| `docs/tech-debt/README.md` | D-4。表の末尾に 6 行を足した(下記) |
| `docs/insights/events/2026-10-07-org-stop-all.jsonl` | `sync_docs` の event を 1 行追記(verdict pass、cycle 1) |
| `docs/reports/sync-docs-2026-10-07-org-stop-all.md` | この report |

## 実行した確認

| Command | Result |
|---------|--------|
| `gofmt -l internal/cli` | 出力なし(rc 0) |
| `go test ./internal/cli/... -count=1` | `ok  github.com/yoshpy-dev/ralph/internal/cli  67.700s` |
| `go test ./internal/scaffold/... ./internal/config/... -count=1` | `ok` 2 パッケージ(`templates/base/` の skill は go:embed の対象なので追加で流した) |
| `./scripts/check-skill-sync.sh` | `[ok] check-skill-sync: 13 skill(s) in lock-step` |
| `./scripts/check-sync.sh` | IDENTICAL 164、DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5。`PASS: all files in sync.` |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-07-org-stop-all.md` | `15eb2a79e2f6`。plan の `- Approved: 2026-10-07 sha256:15eb2a79e2f6` と一致 |

digest は、チェックボックスの `[x]` と `## Progress checklist` の節を数えない仕組み(`scripts/plan-visual.sh` の `digest_body`)なので、AC のチェックと進捗の 1 行で変わらないことを実行で確かめた。

## D-1〜D-4

| ID | 結果 | 場所 |
|----|------|------|
| D-1 | 直した | `internal/cli/org.go` の `stop` の Long(`:724-741`)と `disband` の Long(`:1000-1016`) |
| D-2 | 直した | `.claude/skills/org/SKILL.md:168` と 3 つの写し、`internal/cli/org.go` の `disband` の Long |
| D-3 | 直した | `docs/plans/active/2026-10-07-org-stop-all.md` の AC(`:80-94`)と Progress checklist |
| D-4 | 直した | `docs/tech-debt/README.md` の 163〜168 行目(下記) |

### 追加した tech-debt 行

| 行 | 内容 | 既存の行との関係 |
|----|------|------------------|
| 163 | `--all` の頑健さ。F-3(全体の時間の上限がない。herdr が応答しないと 1 座席あたり約 10 秒、30 座席で約 5 分、出力は最後)と F-4(`Disband` が台帳を 1 回読んでから `disbanded` を書くまでの間に、並行の `spawn` が足した座席が止まらない) | 重なる行はない |
| 164 | 持ち主の確認の抜け 4 件。F-9(`tab_not_found` を `NotFound()` に足したことで `PaneClose` の「閉じ済み」判定にも効く。`confirmSeatPane` が `%v` で包み直して連鎖を切る `verbs.go:693` に頼っている)、`compensateStale` が確認なしで C-c を送る `spawn.go:1845`(既存、plan の Non-goals)、V-2(`resolveWorkspace` が label を見ずに workspace を使い回す)、別の herdr セッションから打った本物の `pane_not_found`(id の空間が分かれているかは未確認) | tech-debt の既存の行には持ち主の確認に触れた行がない |
| 165 | コードの形。F-5(`withPrefixOnce`)、F-6(`Stop` の長さ、自分の pane / workspace の判定が 5 か所、`recordStop` / `failSeat` の写し、`holdsCaller` / `leaveOwnOrg` の名前)、F-7(`NewHerdrError` の export)、F-10(コメントの折り返し)。どれも動作は変わらない | 重なる行はない |
| 166 | 文書。F-8(skill の表の `stop` のセルが長い)と、`ralph org start --help` の `--scope` の行(usage に backtick があり、pflag が最初の backtick の中身を値の名前にするため、`--scope ralph org spawn --scope   optional ...` と出る) | `--scope` は main にある既存の問題で、この PR の diff にはない |
| 167 | 観測。`leaderActivityEventCount`(`internal/org/watch.go:816`)が `stop_failed` を数えない。herdr が止まっている間に leader が `stop` を打ち直すと `stop_failed` だけが増え、deadman が鳴りうる(効果は未測定) | verify の Coverage gaps から |
| 168 | テストの穴。台帳の読み込みの失敗、`stop_failed` / `disbanded` / `org_workspace_created` の追記の失敗、get の結果が JSON でない場合、本物の herdr を使う自動テストがない(CLI の stub は close のあとも状態を消さない)、`fakeWatchHerdr` がすべての get を not-found で返す、mutation M57 が生き残り(期限ちょうどに exit 0 で終わり孫がパイプを握る場合だけの差で、近似的に等価)、`tests/test-secret-scan.sh` が固定の `/tmp/ralph-secret-scan-test.out` と `.err` に書くので 2 本が重なると壊れうる | test report の Test gaps 1〜6 |

どの行も「なぜ今直さないか」は、`/sync-docs` が文書しか変えず、コードやテストを足すと post-implementation pipeline が `/self-review` からやり直しになるため、と書いた。Related には `docs/plans/active/2026-10-07-org-stop-all.md`(`/pr` の `archive-plan.sh` が `archive/` に書き換える)と、該当する report の finding id を使った。

## Surfaces checked for drift

| Surface | Finding |
|---------|---------|
| `README.md`(`:124` の Commands の表、`:245` の org 節の 1 文) | `stop` が pane を、`disband` が workspace を閉じること、`--all` が `--org-id` なしで全 org に効くこと、leader が最後に `report` → `disband` を打つことが書いてあり、実装と合う。持ち主の確認と `--force` は書いていないが、この表の粒度では不要。変更なし |
| `docs/specs/2026-10-07-org-multi-org-director.md` FR-2(`:36`) | 1 座席の `stop` と 1 org の `disband` も閉じること、`stop_failed` で動いているまま残ること、`--force` で `stopped` を書くことが、実装と合う。持ち主の確認は「詳細は 2 段目の計画に書いた」の先にある。変更なし。FR-2 のチェックボックスは PR がマージされるまで付けない |
| `internal/org/prompts/leader.md`(写しはない。`git ls-files` で 1 本だけ) | 手順 6(stop は pane を閉じるので、出力が要るなら先に `read`)、7(`report`)、8(最後のコマンドが `disband`、自分の pane も閉じる、終了コード 1 のときはセッションが続くので打ち直す)と「運用規律」が、実装(自分の pane と workspace は最後、ほかが失敗すれば手を付けない)と合う。変更なし |
| `docs/recipes/codex-seat-permissions.md` と `templates/base/` の写し | 2 面は `cmp` で同一。後片付けの節(`:280-297`)は `stop` が pane を、`disband` が workspace を閉じ、どちらかが終了コード 1 なら herdr が応答してから打ち直すと書き、実装と合う。reviewer 座席の probe の TASK は `reviewer.md` の使命に依存し、この PR は `reviewer.md` を変えていない(変えたのは `leader.md` で、recipe は leader を spawn しない)。変更なし。下の「Found but left」に 1 件 |
| `AGENTS.md` の Repo map、`.ralph/core/AGENTS.core.md`、`CLAUDE.md` | `internal/org/` の行は「herdr/agmsg driver adapters」までで、stop と disband の動作の粒度ではない。変更なし |
| `.claude/rules/ralph/model-routing.md`(と `templates/base/` の写し)の、`ralph org stop` が codex のモデルの receipt をもう一度探す記述 | 探す処理は `Stop` の herdr の呼び出しのあと、台帳に書く前に走り(`internal/org/verbs.go:856` の `stopSeatPane`、`:892` の `observeStopModelReceipt`、`:917` の event の決定)、読む先は codex のセッション記録(`$CODEX_HOME/sessions/`)で、pane を閉じても残る。記述と合う。変更なし |
| `.claude/rules/ralph/agent-messaging.md`、`docs/specs/2026-08-01-org-runtime.md`、`docs/architecture/repo-map.md`、`docs/quality/` | `ralph org stop` / `disband` の動作を書いた箇所がない(`git grep` で確認)。変更なし |
| 古い説明の残り(`C-c` / `sends C-c` / `stop.*C-c` を、plans・reports・tech-debt・evidence・Go のソースを除いて grep) | 残るのは `/org` skill の 4 面と FR-2 の新しい説明だけ |
| `/org` skill の 4 面 | md5 が同一(`766894da…`)。`check-skill-sync.sh` と `check-sync.sh` が通る |
| `docs/evidence/herdr-pane-close-2026-10-07.md` | Plan 行が `docs/plans/archive/...` を指すのは、`/pr` が plan を移すときに解消する先の参照。変更なし |

## Found but left

- skill の `stop` のセルの長さは、F-8 が書いた「1725 字」ではなく、セルの本文が 817 字(1,639 バイト)、`disband` のセルが 809 字(1,599 バイト)だった(Python の `len` で測った)。self-review の数は行全体のバイト数に近い。tech-debt の行には測った値(約 820 字、1,639 バイト)を書いた。F-8 の指示どおり `stop` のセルは触らず、`disband` の D-2 の文だけを足したので、`disband` のセルは 1 文ぶん伸びた(これも tech-debt の行に書いた)。
- recipe の後片付けの節の「どちらかが終了コード 1 なら、herdr が応答してから打ち直す」は、herdr が応答しない場合には正しい。持ち主の確認に失敗した場合(利用者が tab の名前を変えた、など)は打ち直しても同じ結果になり、stderr の理由を読む必要がある。recipe は scratch の座席を `ralph org spawn` で作ったばかりで、label は変わっていないので、手順の上では起きない。変更していない。
- plan の Verify plan の「AC1〜AC13」は、digest が変わるので書き換えず、Progress checklist の注記で補った(依頼どおり)。「PR created」は `/pr` で付く。
- `ralph org start --help` の `--scope` の表示(上の 166 行目)は、main にある既存の問題で、この PR の変更ではない。tech-debt に送り、`internal/cli/org.go:489` は直していない。
- `--help` に書いた動作(確認に失敗した pane と workspace は閉じない、`--force` は記録だけ)は、実装(`confirmSeatPane`、`confirmOrgWorkspace`、`stopSeatPane`、`closeOrgWorkspace`)とテストの内容に照らして書いた。本物の herdr では動かしていない。
- この `/sync-docs` は `internal/cli/org.go` の文字列を変えたので、self-review と verify はこの diff を見ていない(上の Summary)。パイプラインを `/self-review` から回し直すかどうかは、`/cross-review` の結果と合わせて判断してほしい。help の文字列だけの変更で、`go test ./internal/cli/...` は通った。
