# Verify report: org-director-wait-bound

- Date: 2026-10-19
- Plan: `docs/plans/active/2026-10-19-org-director-wait-bound.md`(ステージ 8b-2c、パイプライン cycle 1)
- Verifier: verifier サブエージェント(`/verify`)
- Scope: `git diff 3a803a87...HEAD`(HEAD 83c0d34d、12 ファイル、+1011/-67)。受け入れ条件 AC1〜AC9、Verify plan の仕様適合と文書のずれの項目、Progress に書かれた計画からのずれ、self-review の再レビューと N1 の直し(de2b05d1)を照らし合わせた。テストは走らせていない(`/test` の担当)
- Evidence: `docs/evidence/verify-2026-10-19-org-director-wait-bound.log`(`docs/evidence/*.log` は gitignore の対象なのでコミットしない)。`run-static-verify.sh` 自身の出力は `docs/evidence/verify-2026-10-19-131731.log`

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1: Claude Code の箇条が `--timeout-ms 1500000` の wait を Monitor で見張り、`timeout_ms` に `1800000` を渡す。`--timeout-ms 0` が残らない | met | 4 つの写しの 487〜488 行(`cmp` で同じ)。`--timeout-ms 0` の grep は 4 つの写しで 0 件(log の 6) |
| AC2: Bash の `run_in_background` を代わりの手として書き、Move to background と SIGKILL のあとも残ること(25 分のうちに終わる)と、`note:` の行が wait の終わりまで届かないことを書く | met | 497〜501 行。`wait --inbox --director: ` で始まる行も終わりまで届かないと書く |
| AC3: 時間切れを Claude Code でも打ち直し、Monitor の期限切れの知らせのあとの signal での終わりを、ほかの signal での終わりと分けて書く | met | 495〜496 行(時間切れの文で打ち直す)、549〜551 行(終了コード 1 の箇条)、553〜555 行(失敗にしない終わりの列挙に Monitor の期限切れを足した)、565〜575 行(例外の段落)。段落は、signal での終わりを見た時点でその wait の Monitor の知らせが届いていれば打ち直し、届いていなければ失敗とし、あとから届けば打ち直す。ほかの signal、panic、時間切れの文のない終了コード 1 は失敗のまま。D3 にない「あとから届いたら打ち直す」は Progress の 129 行に記録がある |
| AC4: テストが 4 つの写しで `0 < N < M <= 1800000` と `--timeout-ms 0` がないことを確かめ、前の形に戻すと落ちる | met(静的に確認) | `internal/org/director_skill_sync_test.go` 357 行の `TestDirectorSkill_BoundsTheClaudeCodeWaitBelowTheMonitorCap` が 4 つの写しそれぞれに `directorSkillWaitBoundProblems`(298 行)を当てる。上限は 156 行の定数 1800000。377 行の表のテストに、main の箇条をそのまま写した `oldBullet`(391 行)の場合がある。テストの判定を Python に写して当てると(log の 8)、HEAD の 4 つの写しは問題 0 件で、main(3a803a87)の skill は 3 件(M がない、N=0、`--timeout-ms 0`)になる。`go test` の実行と mutant は `/test` に任せた |
| AC5: `check-skill-sync.sh` が通り、4 つの写しが同じ | met | `[ok] check-skill-sync: 14 skill(s) in lock-step`、rc 0。`cmp` は 3 組とも rc 0(log の 4・5) |
| AC6: 実機の確認 (a)〜(g) が evidence にあり、(e) は時間切れの打ち直しが 2 回以上続いてから件が返って ack され、(f) は期限切れのあとに打ち直し、(g) は外からの SIGTERM のあとに打ち直さない | met | `docs/evidence/director-wait-bound-livefire-2026-10-19.md` の表(37〜41 行)と mark log(66〜88 行)。(e) では時間切れのあとの打ち直しが 12:14:16Z、12:14:38Z、12:15:01Z と人の入力なしに 3 回続き、e1 が 12:15:13Z に返り、12:15:15Z に ack され、12:15:18Z にまた待った。(f) では 10 秒の Monitor の期限で 12:13:51Z に signal で終わり、12:13:54Z に打ち直した。(g) では `kill -TERM` で 12:18:11Z に終わり、そのあと 45 秒、新しい wait は始まらず、セッションは人に伝えた。確認が読んだのは 5aea657f の skill の文で、そのあとの dc98d5ec と de2b05d1 の文では打ち直していない。このことは evidence の 132〜140 行と仕様の注に書かれている |
| AC7: tech-debt の B の行が、Monitor と Bash で残る場合、25 分の上限、返した件の行き先、親の pid で見分けない理由を書き、閉じていない | met | `docs/tech-debt/README.md` 248 行。Monitor は claude の SIGKILL(落ちたとき)だけ残り、Bash は Move to background と `claude stop` のあとも残る。上限は Claude Code 25 分、codex 300 秒。返した件は open のまま次の wait が返し、ack されなければ期限で人に届く。親の pid については、`zsh -c` が pid 1 に付け替わり、段数はシェルの包み方に依り、codex は daemon であることを書く。`~~` と `RESOLVED` はなく、区切りの `|` は 6 個で見出し(14 行)と同じ。247 行は Trigger の欄だけが変わり、`|` は 6 個 |
| AC8: 仕様の FR-10 に 8b-2c の注がある | met | `docs/specs/2026-10-07-org-multi-org-director.md` 98 行。FR-10(91 行)の中で、8b-1 の注(97 行)のあと、FR-11(99 行)の前 |
| AC9: `GOFLAGS=-timeout=30m ./scripts/run-verify.sh` が通る | pending | orchestrator がこの worktree の背景で走らせている run-verify の結果待ちで、結果は orchestrator が記録する。計画の checkbox が未チェックなのはこの状態と合う |

Verify plan の仕様適合の項目:

- FR-10 の「Claude Code は Monitor で待ち」(93 行)は、skill の Claude Code の箇条(487〜488 行、Monitor で見張る)と合う。区切って打ち直すことは 98 行の注が書く
- Codex の箇条(504〜507 行)は main と同じ(切り出した 5 行を `cmp` で比べて rc 0)
- `internal/org/prompts/director.md` は main から変わっていない(`git diff --quiet` rc 0)。42〜44 行は「skill のとおり `--timeout-ms` を足して打つ」と書き、数字を持たない
- Progress に書かれたずれ: 仕様の注の位置(127 行)は上の AC8 のとおり。L2 と N1 で D3・R1 の「知らせのあと」の前提を外したこと(128〜129 行)は、計画の本文を変えずに Progress にある。計画の digest は今も `505a695e5548` で、`- Approved:` の行と合う(本文は承認のあと変わっていない)

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(rc 0) | 変更した言語の範囲(golang)。shellcheck、`sh -n`、guard の awk、`jq -e`、Codex の guard 3 つ、`check-sync.sh`(DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`、`check-template-purity.sh`、tech-debt の計画の参照、gofmt ok、go vet(出力なし)、golangci-lint `0 issues.`、staticcheck(出力なし)、branch の secret scan(3a803a87..83c0d34d、clean) |
| `GOTOOLCHAIN=go1.25.8 go vet ./internal/org/` | pass(rc 0) | go.mod は `go 1.25.8`、手元は go1.26.0 |
| `gofmt -l internal/org/director_skill_sync_test.go` | pass | 出力なし |
| `cmp` で 4 つの写しを比べる | pass | 3 組とも rc 0 |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | `[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `git diff --check 3a803a87...HEAD` | pass(rc 0) | |
| `./scripts/plan-visual.sh digest <plan>` | pass | `505a695e5548`(記録と同じ) |

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| README の director の段落(258 行) | yes | 「waits on the inbox with `ralph org wait --inbox --director <id>`」とだけ書き、`--timeout-ms 0` を書かない。ファイルは main から変わっていない |
| `docs/recipes/director-wake.md`(7・271 行) | yes | `--timeout-ms 0` を書かない。変わっていない |
| `internal/org/prompts/director.md`(42〜44 行) | yes | 数字を持たず、skill に従う |
| 仕様 FR-10(93・98 行) | yes | 上の AC8 |
| tech-debt 247・248 行 | yes | 上の AC7。248 行の計画の道は `docs/plans/active/` を指し、`/pr` の archive が書き換える |
| `ralph org wait --help`(`internal/cli/org.go` 1243 行から) | yes | 計画の Non-goals のとおり変えていない。`--timeout-ms 0` を wait 全般の無期限として書くだけで、director の待ち方は書かない |
| `/org` skill の `wait` の行(4 つの写しの 175 行) | yes | 例の `ralph org wait --inbox --timeout-ms 0` は人が打つ受信箱の wait で、Non-goals(31 行)の範囲 |
| AGENTS.md と README の repo map | yes | Go のコードは変わっていない |

## Observational checks

- 2 つの evidence を全文読んだ。`/Users/`、`hiroki`、`/home/`、UUID、`token`、`director_<token>` の形、`sk-`、`api_key` はどちらにも 0 件(log の 11)。16 進は短い commit SHA だけ。19 桁の数は mark log の `wait_id` の後ろの開始時刻(Unix ナノ秒、pid と組)で、乱数ではない。パスは `~`、`/private/tmp/xbw`、`/private/tmp/x9m` だけで、session id は `<id>` に、shell snapshot の名前は `…` に伏せてある。pane id は書かれていない

## Findings

| ID | Severity | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| V1 | INFO | 計画の D6 は (a)〜(d) を claude(haiku)、(e)〜(g) を sonnet で打つと書くが、livefire の Setup は「model sonnet」だけを書き、(a)〜(d) のモデルを分けていない。Progress の S2 の記録(126 行)もモデルを変えたとは書かない。AC6 はモデルを名指さないので、受け入れ条件には響かない | 計画 62 行、`docs/evidence/director-wait-bound-livefire-2026-10-19.md` 14 行 | ずれなら Progress か Setup に 1 句足す。merge には要らない |
| V2 | INFO | Monitor の `timeout_ms` の最大 30 分と既定 5 分は、計画の Assumptions(38 行)ではツールの説明によるとされている。実験で測ったのは明示した 20000(m4)だけで、実機の確認も値を明示して渡した(15000、60000、900000、10000)。実験の evidence の Reading(83〜84 行)と、テストの定数の doc コメント(152〜161 行)は、この数がどこから来たかを書かない。248 行は「the real 25- and 30-minute numbers」を確かめていないことに挙げている | 計画 38 行、`docs/evidence/director-wait-monitor-2026-10-19.md` 83〜84 行、`internal/org/director_skill_sync_test.go` 152〜161 行 | 直すなら Reading に「ツールの説明による」と添える。merge には要らない |

CRITICAL、HIGH、MEDIUM、LOW はない。

## Coverage gaps

- テストの実行(`go test ./internal/org/ -run TestDirectorSkill -count=1`、`GOTOOLCHAIN=go1.25.8` でも)と、計画の Edge cases の mutant は `/test` の担当で、ここでは走らせていない。AC4 は、テストのコードを読んだことと、判定を Python に写して HEAD と main の skill に当てたことで確かめた
- AC9 は背景の run-verify の結果待ち
- 確認していない(likely but unverified): 本物の 25 分と 30 分の数字、Monitor の最大と既定の値(V2)、dc98d5ec と de2b05d1 の skill の文に従うセッションの振る舞い(実機は 5aea657f の文で打った)。verifier は Monitor ツールを持たないので、ツールの説明は読んでいない
- unknown: 本物の `/director` skill で何時間も動く director、headless の director、codex(どれも evidence と 248 行が確かめていないと書いている)

## Verdict

- Verdict: pass(AC1〜AC8 を満たし、静的解析はすべて通った。AC9 は背景の run-verify の結果待ちで、orchestrator が記録する。指摘は INFO の 2 件だけ)
- Verified: skill の 4 つの写しの Claude Code の箇条の数と文(AC1〜AC3)、4 つの写しが同じこと(AC5)、テストが前の形で落ちる判定を持つこと(AC4、静的)、実機の evidence の (a)〜(g) と mark log の対応(AC6)、tech-debt の 247・248 行と仕様の注(AC7、AC8)、Codex の箇条、director の雛形、README、recipe が変わっていないこと、計画の digest、evidence に個人の値がないこと、`run-static-verify.sh` と go1.25.8 の vet
- Partially verified: AC4(テストを走らせていない)、AC6(実機が読んだのは 5aea657f の文)
- Not verified: AC9(結果待ち)、テストの実行と mutant、本物の 25 分と 30 分の数字、Monitor の数の出どころ
