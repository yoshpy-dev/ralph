# Verify report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md(`Status: Approved`。`./scripts/plan-visual.sh digest` は 41bff9980d8c で、`- Approved:` の digest と同じ。仕様 `docs/specs/2026-10-07-org-multi-org-director.md` の FR-7 のうち、merge のあとの取り直し。8 段目の 8a-3b)
- Verifier: verifier subagent (Claude)。パイプライン 3 回目(cross-review cycle 2 の ACTION_REQUIRED 2 件のあとの回し直し。上限を 3 に上げ、`.harness/state/standard-pipeline/cycle-count.json` の cycle は 2 のまま)
- Scope: `git diff 670ec219...HEAD`(HEAD 1c500537)。前回の verify(cycle 2 の 2 度目、e6af24b3)からの差分は、f630e216(テストだけ)、c60b2651(cycle 2 の sync-docs)、08c0a5d5(cross-review の triage)、ef2d859c(D4 と AC7 の書き直し、承認し直し)、556f73fb と 4129ae22(cross-review の直し)、cycle 3 の self-review(c5d32c4a)とその直し 82cf0d39・0e08999f・14252f06・ba84bccc、計画の注 927d6c26・1c500537。書き直された D4 と AC7 を重く見て、コードを読み、HEAD と直す前(ef2d859c)の binary で打った。ほかの AC は、差分がコードを変えていないことを確かめ、前回の結果を引き継ぐ。テストは走らせていない(/test の受け持ち)。行番号は HEAD 1c500537 のもの
- Evidence: `docs/evidence/verify-2026-10-16-org-regate.log` の「cycle 3」の節(前回までの節のあとに足した。run-static-verify の出力、Go の検査、2 つの移動の比べ、probe 7 のスクリプトと 2 つの binary の出力。`$PROBE7` は scratchpad の probe のディレクトリ)。wrapper の全ログは `docs/evidence/verify-2026-10-16-222633.log`。どちらも gitignore の対象

## 前回からの変化

- 556f73fb: push だけの打ち直しと `reset --keep` の手は、記録が regate の書いたもの(`via=regate`)なら、HEAD が regate 自身の merge commit の形のときだけになった。形は `regateIsOwnMerge`(`internal/org/regate_merge.go:131-137`)が見る。親がちょうど 2 つで、第 1 親が fetch のあとの `origin/<b>`、第 2 親が記録の base。leader の記録(`via` なし)の push だけの道は、前と同じく `origin/<b>` が HEAD の祖先かだけを見る(`:80`)
- 4129ae22: push のあとに `refs/remotes/origin/<b>` を HEAD に向けるのは、push の直前に読んだ値(`:525`)から動いていないときだけになった(`regateSetTrackingRef`、`:582-602`)。test hook `afterPush` と `beforeTrackingUpdate`(`internal/org/regate.go:126-133`)を足した
- 0e08999f と 14252f06 はファイルの移動だけ。D3 の確かめ(`regateCheckInTheWay` から `regateSpellingOnDisk` まで)は `internal/org/regate_inway.go` に、gate の記録(`gateRecord`、`gateRecordFromDetails`、`gitObjectPattern`)は `internal/org/gate_record.go` に移った。元のファイルから消えた行と新しいファイルを `diff` で比べると、違いは package と import の行、ファイルの注、最後の空行だけで、関数の本体は 1 行も変わっていない(evidence の「the two moves」)。`regate_merge.go` は 613 行、`gate.go` は 709 行になった
- 82cf0d39 と ba84bccc は注と help の文だけ。前回の差分からのコードの変更は、上の 2 つの直しと 2 つの hook の欄だけ(`git diff e6af24b3 HEAD` で、注と文字列の行を除いて確かめた)
- 前回の V2-2〜V2-4 は c60b2651 で直った(仕様 7c-1 の注に 8a-3b の付け先の変更、`/org` skill と README に ignored のファイルの断り、tech-debt の 7c-1 の行の名前)。V2-5 は tech-debt の行(`docs/tech-debt/README.md:285`)になった

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 断る 12 の場合 | 満たす(前回の binary の確認のまま) | 差分はコードを変えていない。12 番目(D3)の関数は `internal/org/regate_inway.go` に移ったが、本体は同じ(`regateCheckInTheWay` は `:39`、`regateMergeWrites` は `:99`、`regatePathInTheWay` は `:169`、`regateSpellingOnDisk` は `:211`)。呼び出しは `regateMerged`(`internal/org/regate_merge.go:177`)のまま |
| AC2 fetch の失敗、origin にブランチがない、HEAD が `origin/<b>` と違う | 満たす(cycle 1 の binary の確認のまま) | 差分が触らない。`regateFetch`(`internal/org/regate_git.go:30`)、`regateCheckOriginHead`(`:82`)。probe 7 の B2 と A3-2 も「HEAD … has commits that origin's tip … does not」で断った |
| AC3 何もしない道と、壊れた記録のあとの取り直し | 満たす(前回の確認のまま) | 差分が触らない。`passedOn`(`internal/org/regate.go:433-440`) |
| AC4 取り込みの道 | 満たす(binary でも確認) | probe 7 の D1 で、merge して `GATE: pass`・`MERGED:`・`HEAD:`・`BASE:`・`LOG:`・`PUSHED:` を出し、終了コード 0。A1 の merge commit の親は `[merge の前の HEAD, origin/main の先頭]` |
| AC5 merge commit を作れない | 満たす | コードは変わっていない。probe 7 の C2-2 で、衝突を `conf.txt` と名指して断り、何も記録しなかった(manifest は 3 行のまま) |
| AC6 スクリプトの失敗、記録の前の中断、`head_moved` と `tree_changed` | 満たす(cycle 1 の確認のまま) | 差分が触らない |
| AC7 push の失敗と記録のあとの中断、打ち直し(書き直したもの) | 満たす(binary でも確認。V3-1 は文の外の残り) | コード: `regateHeadNotOnOrigin`(`internal/org/regate_merge.go:72-99`)は `passedOn` のあと、`via=regate` なら `regateIsOwnMerge`(記録の base を渡す)、`via` なしなら `regateIsAncestor` で push だけを許す。`regateResetHint`(`:111-123`)は、記録が読めて壊れた行が後になく、head が HEAD で `via=regate`、HEAD が同じ形のときだけ手を出す。D4 の文と一致する。push のあとの tracking ref は D4 の最後の文のとおり(`:525`、`:588`、`:594-597`)。probe 7(HEAD の binary): A1 は pre-receive が断った push で終了コード 1、記録と merge commit を残した。A2 は hook を外して打ち直し、`PUSH_ONLY:` で終了コード 0。A3 は push が落ちたあとに main が進み、断りに `reset --keep` の手が付いた。B は gate だけの regate が 1 つ親の commit で通ったあと origin を親に戻すと、断り、手を出さず、push しなかった(origin は親のまま)。D は push の直後に origin がさらに新しい commit を受け、worktree の repo に fetch されると、tracking ref はその新しい commit のまま残った。直す前の binary(ef2d859c)は B で push し直し(`PUSH_ONLY:`、origin は gate の head に戻った)、D で tracking ref を push した merge commit に巻き戻した。テスト(読んだだけ): `TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt`(`internal/org/regate_merge_test.go:1253`、3 case)、`TestRegate_AFailedPushIsNotRetriedPastWhatChanged`(`:1051`、3 subtest)、`TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush`(`:135`、3 case) |
| AC8 スクリプトの間の org か結びつきの変化 | 満たす(テストのコードで確認) | 差分が触らない。`recordRegate`(`internal/org/regate.go`) |
| AC9 gate だけの道 | 満たす(binary でも確認) | probe 7 の B1 と C0 は、merge も push もせずに gate を走らせ、`via=regate` の `gate_passed` を書いた |
| AC10 merge-order の理由の欄 | 満たす(前回の確認のまま) | 差分は `internal/org/mergeorder.go` の注の 4 行だけ(ba84bccc) |
| AC13 Spawn との排他(D9) | 満たす(テストのコードで確認) | 差分が触らない。help の文は 82cf0d39 で「spawn (not a --dry-run) and start (either form)」になり、コードと合う(dry-run は `internal/org/spawn.go:555` で返り、`regateSpawnErr` は `:1284`) |
| AC11 文書 | 一部(LOW のずれ 3 件) | 下の V3-2〜V3-4。help、`/director` skill、README、仕様の 8a-3b の注は、書き直した D4 とぶつからない |
| AC12 テストの隔離、go1.26 と go1.25.8 | 一部(静的な部分を確認) | 足したテストは `newRegateFixture` の bare origin と test hook だけを使い、URL、`osascript`、本物の `herdr` を打たない。`go build` と `go vet` は go1.26.0 と `GOTOOLCHAIN=go1.25.8` の両方で通った。テストの実行は /test |

D1〜D9:

- D4: 書き直した文とコードは一致する(上の AC7)。push だけの道の条件は「読める `gate_passed`、head が HEAD、base が今の既定のブランチの先頭」(`passedOn`)に、`via=regate` なら形の確かめ、`via` なしなら祖先の確かめを重ねる。`reset --keep` の手は `passedOn` を求めず、記録の base で形を見る(D4 の「HEAD が同じ merge commit の形なら」)。形で判じる以上、同じ形の人の merge commit は regate のものと見分けられない(V3-1)
- D3: 0e08999f は確かめを動かしただけ。判じ方は前回の verify が確かめたまま
- D6: 14252f06 は記録の型と読み書きを動かしただけ。`via` を `log` の前に置く書き方(`gateRecord.details`)は同じ
- D1、D2、D5、D7〜D9: 差分が触らない。前回までの結果のまま

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh` | pass(exit 0) | scope は changed、言語は golang。hook の shellcheck と `sh -n`、`check-sync.sh`(IDENTICAL 170、DRIFTED 0)、`check-pipeline-sync.sh`、`check-skill-sync.sh`(14 skill)、`check-template-purity.sh`、tech-debt の plan の参照、gofmt ok、golangci-lint `0 issues.`、`secret-scan-branch: scanned 670ec219..1c500537 against origin/main: clean` |
| `go vet ./...`(go1.26.0) | pass | 出力なし、rc 0 |
| `gofmt -l internal/ cmd/` | pass | 出力なし |
| `GOTOOLCHAIN=go1.25.8 go build ./...` | pass | go.mod の Go |
| `GOTOOLCHAIN=go1.25.8 go vet ./...` | pass | 出力なし、rc 0 |
| `./scripts/check-skill-sync.sh` | pass(rc 0) | 14 skill が lock-step |
| `./scripts/check-sync.sh` | pass(rc 0) | IDENTICAL 170、DRIFTED 0、KNOWN_DIFF 5 |
| `git diff --check 670ec219...HEAD` | pass | 出力なし |
| `./scripts/plan-visual.sh digest <plan>` | 一致 | 41bff9980d8c |

## Findings

| ID | Severity | 種類 | Finding | Evidence | 勧め |
| --- | --- | --- | --- | --- | --- |
| V3-1 | LOW | 振る舞い(D4 の形の判定の残り) | gate だけの regate が通した head が、それ自体 2 つ親の merge commit で、第 1 親が origin の先頭だった commit、第 2 親が記録の base なら、`regateIsOwnMerge` は regate 自身の merge と読む。この形は、`/org` skill の衝突の手順(`.claude/skills/org/SKILL.md:1035-1038`、「`git merge origin/<既定のブランチ>` を打って直し、commit して push してから、regate を打ち直す」)と、regate の衝突の断りの文(`internal/org/regate_merge.go:337`)が人に作らせる commit そのもの。GitHub の「Update branch」(merge の方式)もおそらく同じ形を作る(未確認)。そのあと origin のブランチをその第 1 親に戻す(force push)と、(1)既定のブランチが同じなら、push だけの道がその commit を push し直し、戻したことを黙って取り消す。(2)既定のブランチが進んでいれば、断りの文が「HEAD is the merge commit an earlier regate made」と言って `reset --keep` を勧め、従うと人が直した衝突の解消が worktree から消え、次の regate は同じ衝突で断る。cross-review cycle 2 の P2-1 と同じ害が、1 つ親でない head に残る。計画の AC7 は「1 つ親の commit」とだけ書くので、AC7 の文は満たす。D4 が理由に挙げる「人が消した commit を戻す」ことは、この形では防げていない。self-review の C3-5 は regate 自身の merge commit の場合だけを書き、その登録簿の案は gate だけの記録をテストが押さえたと書く | probe 7 の C(HEAD の binary): 人の merge commit(親は `[ブランチの先頭, origin/main の先頭]`、`conf.txt` の衝突を解いたもの)を gate だけの regate が通し(`via=regate`)、origin を第 1 親に戻すと、C1 は `PUSH_ONLY:` で終了コード 0、origin は人の merge commit に戻った。C2-1 は main を進めて断り、上の文と `reset --keep` を出した。従うと HEAD は第 1 親、`conf.txt` は `feature` に戻り、C2-2 は `conf.txt` の衝突で断った。直す前の binary も同じ結果(この 2 つの直しは触っていない)。コードは `internal/org/regate_merge.go:77-78`、`:113-118`、`:131-137` | この PR のコードは変えず、C3-5 の登録簿の行の(e)にこの場合を書き足す(/sync-docs)。直すときの案は 2 つ。記録に regate が merge したことを書く(tech-debt の `docs/tech-debt/README.md:280` が挙げる `merged=` の key)。記録の形を変えずに済ませるなら、`regateMergeTip` が書く件名 `Merge <base_ref の短い名前> <base の 12 桁> into <b>`(`:285`)も確かめる(probe 7 の人の merge commit の件名は git の既定の `Merge remote-tracking branch 'origin/main' into feat/auth` で、これと違う) |
| V3-2 | LOW | 文書(skill) | `/org` skill の 4 面の `regate` の行は、`reset --keep` の手の条件を「HEAD の第 1 親が origin の先頭」と書く。556f73fb のあとの条件は、親がちょうど 2 つで、第 1 親が origin の先頭、第 2 親がその記録の base(`regateIsOwnMerge`)。同じ行の push の打ち直しの文は、`via=regate` の記録ではこの形のときだけ push することを書かない | `.claude/skills/org/SKILL.md:188`(`.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` も同じ。4 面は cmp で同一)。`internal/org/regate_merge.go:111-137` | 4 面の条件を「HEAD の親がちょうど 2 つで、第 1 親が origin の先頭、第 2 親がその記録の base」に直す。push だけの道にも同じ条件を 1 文足すとよい(/sync-docs) |
| V3-3 | LOW | 文書(tech-debt) | 0e08999f のあと、4 つの行が D3 の関数を古いファイルに置く。`regateMergeWrites`(281 行の本文と最後の欄、282 行の最後の欄)、`regatePathInTheWay` と `regateSpellingOnDisk`(285 行の本文と最後の欄、286 行の本文と最後の欄)はどれも `internal/org/regate_merge.go` と書くが、今は `internal/org/regate_inway.go` にある(282 行と 287 行の `regateMergeTip` は `regate_merge.go` のまま)。286 行は、regate の間に断られるものを「the org's `spawn` and `start --plan` are refused (D9)」と書き、C3-1 で直した help(spawn は `--dry-run` でないもの、start はどちらの形も)と合わない | `docs/tech-debt/README.md:281`、`:282`、`:285`、`:286` | 4 行のファイル名を `internal/org/regate_inway.go` に、286 行の文を「the org's `spawn` (not a `--dry-run`) and `start` (either form) are refused (D9)」に直す。self-review の報告が足すよう勧める 1 行は、(a)〜(d) が 82cf0d39・0e08999f・14252f06 で直った(`regate_merge.go` は 613 行、`gate.go` は 709 行)ので、(e) と V3-1 だけを書く。(a) の「the row on the per-file `git ls-files` … says the same」も、286 行を直せば要らない(/sync-docs) |
| V3-4 | LOW | 文書(repo-map) | `AGENTS.md` の repo-map は regate のファイルを `regate.go`、`regate_merge.go`、`regate_git.go`、gate のファイルを `gate.go`、`gate_run.go` と名指すが、0e08999f と 14252f06 が足した `regate_inway.go`(merge の前の D3 の確かめ)と `gate_record.go`(gate と regate が書き、merge-order が読む gate の記録)を名指さない。この PR は 94ff3493 で `regate_git.go` を切り出したとき、a3cd529c で repo-map に足している | `AGENTS.md:90` | 2 つのファイルを repo-map の regate と gate の句に足す(/sync-docs) |

情報(ずれに数えない):

- I1 probe 7 の C2-1 の「unchanged: no」は、regate の fetch が `refs/remotes/origin/feat/auth` を origin の先頭に合わせた分だけで、HEAD、origin の 2 つの ref、作業ツリー、manifest の行数は前のまま。help の「the fetches change only refs/remotes/origin/<name>」のとおり
- I2 regate の help の「When origin's default branch or origin's branch moved in between, regate refuses and says what changed」(`internal/cli/org_regate.go:104-105`)は、origin のブランチを merge の第 1 親に戻した場合には当たらない(C3-5 の (2) と V3-1 では push し直す)。C3-5 の登録簿の行で扱う
- I3 `internal/org/regate_merge_test.go` の冒頭の注は「the merge commit, its undo, and the push」と書くが、D3 の確かめのテストもこのファイルにある。移動の前からこの書き方で、テストは動かしていない。テストのファイルを分けるときに直せばよい(tech-debt の 279 行)
- I4 計画の Progress(`:164-166`)は、cross-review cycle 2 の直しと cycle 3 の self-review の直しを書いている。AC1〜AC13 は `[x]`。digest は 41bff9980d8c

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| `/org` skill の 4 面 | 一部(V3-2) | 4 面は同一。`regate` の行の手の条件が 556f73fb の前のまま。spawn と start の行の regate の断りは c60b2651 で直った |
| `/director` skill の 4 面 | yes | 4 面は同一。手の条件と push だけの道の条件を書かないので、556f73fb とぶつからない |
| CLI の help(`ralph org regate`) | yes(情報 I2) | spawn と start の文は 82cf0d39 で直った。打ち直しの段落は条件を書かず「says what changed」とだけ言う |
| `AGENTS.md` の repo-map | 一部(V3-4) | `regate_inway.go` と `gate_record.go` がない |
| `README.md` | yes | c60b2651 で ignored のファイルの断りと spawn・start の文が入った。手の条件は書かない |
| 仕様(75 行の 8a-3b の注、77 行の 7c-1 の注) | yes | 打ち直しは「push だけをする」とだけ言う。7c-1 の注は 8a-3b の付け先の変更を書く |
| `docs/tech-debt/README.md` | 一部(V3-3) | 4 行のファイル名と 286 行の文。C3-5 の行はまだない(/pr の前に足す予定) |
| `docs/quality/quality-gates.md` の 2 面 | yes | 差分が触らない |
| Go の注(移動のあと) | yes | `regate_merge.go:12-17`、`regate_inway.go:14-20`、`gate.go:25`、`gate_record.go:9-15`、`mergeorder.go:24` は新しいファイルを指す |
| 計画の AC と Progress | yes | 情報 I4 |

## Observational checks

probe 7 は、HEAD 1c500537 の binary(`go build ./cmd/ralph`)と、直す前の ef2d859c から `git archive` で作った binary を、Homebrew の git 2.49.0 の symlink と `/usr/bin`・`/bin` だけの PATH で打った。gh は PATH にない。HOME と `GIT_CONFIG_GLOBAL` は probe の中の偽の場所、origin は probe の中の bare repository、台帳は scratch の main checkout の `.harness/state/org`(結びつきと `disbanded` の 2 行)。gate のスクリプトは `echo` と exit 0 だけの stub。場合ごとに作り直した。ネットワーク、herdr、本物の GitHub は打っていない。

- A(regate 自身の merge commit): A1 は pre-receive の hook が push を断り、終了コード 1、記録と merge commit が残った(親は `[merge の前の HEAD, main の先頭]`)。A2 は hook を外して `PUSH_ONLY:`、終了コード 0。A3 は push が落ちたあと main を進め、断りに `reset --keep` の手が付いた。両方の binary で同じ
- B(gate だけ、1 つ親の commit): HEAD の binary は origin を親に戻したあと断り、手を出さず、push しなかった。直す前の binary は `PUSH_ONLY:` で push し直した(556f73fb が変えたこと)
- C(gate だけ、衝突を解いた人の merge commit): 両方の binary で、origin を第 1 親に戻すと push し直し(C1)、main も進めると regate の merge と呼んで手を出した(C2-1)。手に従うと衝突の解消が消えた(C2-2)。V3-1
- D(push の直後の fetch。`remote.origin.fetch` なし、origin の post-receive の hook が新しい commit を足して worktree の repo に fetch する): HEAD の binary は tracking ref を fetch の値のまま残した。直す前の binary は push した merge commit に巻き戻した(4129ae22 が変えたこと)

## Coverage gaps

- テストは走らせていない。go1.26 と `GOTOOLCHAIN=go1.25.8` の `go test ./internal/org/ ./internal/cli/` は /test で確かめる
- `regateSetTrackingRef` の `update-ref` が落ちたあとの読み直しの分岐(ref が `before` から動いていれば誤りにしない)と、push の前の読みが落ちる道は、コードとテストを読んだだけ
- leader の記録(`via` なし)で origin を戻したあとの push し直し(C3-5 の (1))と、regate 自身の merge が届いたあと第 1 親に戻された場合(C3-5 の (2))は、binary で打っていない
- 前回までの Coverage gaps はそのまま残る: 中断の道、`head_moved` と `tree_changed`、hook が git を殺す場合、AC13 の spawn と `start --plan` の binary の確認、端末の Ctrl-C の実機、ssh の origin、本物の GitHub への push、Linux、本物の 2.38 より前の git、case を区別するファイルシステムでの D3

## Verdict

- Verdict: pass(MEDIUM 以上はない。LOW 4 件。V3-1 は、計画の AC7 の文の外に残る D4 の形の判定の穴で、force push で origin を戻したときだけ起き、tech-debt に回す。V3-2〜V3-4 は /sync-docs で直す文書のずれ)
- Verified: 承認の digest。書き直した D4 と AC7 のコード(`regateHeadNotOnOrigin`、`regateResetHint`、`regateIsOwnMerge`、`regatePush`、`regateSetTrackingRef`、`regateTrackingRef`)と計画の文の突き合わせ。2 つの移動が関数の本体を変えていないこと。HEAD の binary で、regate 自身の merge の push の失敗と打ち直し、手の出る条件、1 つ親の gate だけの head を push し直さないこと、push の直後に fetch された tracking ref を残すこと。直す前の binary との対照 2 件(B と D)。静的解析(run-static-verify、go vet、gofmt、go1.25.8 の build と vet、check-skill-sync、check-sync、`git diff --check`)
- Partially verified: AC6、AC8、AC13 のうちテストだけが固定する場合、AC11(LOW のずれ 3 件)、AC12 のテストの隔離(コードを読んだ範囲)
- Not verified: テストの実行、C3-5 の 2 つの場合の binary の確認、端末の Ctrl-C の実機、ssh と GitHub の origin、Linux
- Insight event: `insights-append.sh` で `--verdict pass --cycle 3 --medium 0 --low 4` を記録した
