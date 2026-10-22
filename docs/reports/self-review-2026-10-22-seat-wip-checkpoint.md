# Self-review report: seat-wip-checkpoint

- Date: 2026-10-22
- Plan: `docs/plans/active/2026-10-22-seat-wip-checkpoint.md`(パイプライン cycle 1)
- Reviewer: reviewer サブエージェント(`/self-review`)
- Scope: `git diff c90d01b4...HEAD`(HEAD 1c78d731、39 ファイル、+1354/-257)。S1 88706897(WIP の hook 2 つと写し、`tests/test-wip-checkpoint-hooks.sh`)、S2 416993cf(`driver.SeatPaneEnv`、本番の組み立て、起動スクリプトの export、warning の文)、S3 3dedaef9(実機の evidence)、S4 b5b12ae9(leader の雛形の手順 14 と運用規律、文書)、S5 80cb817a と 075b2160(記録)、計画の 4 コミット。diff の質だけを見た

## Evidence reviewed

- hook 2 つの diff を全部読み、`.claude/hooks/precompact_checkpoint.sh` と `session_end_summary.sh` の全文と突き合わせた。変わったのは `git status --porcelain` が空でない枝の中だけで、`[ "${RALPH_WIP_CHECKPOINT:-}" = "off" ]` の `then` に skipped の 1 行、`else` に元の `git add -A` と `git commit` と失敗の記録をそのまま入れ子にした形。`case` の `main|master|unknown`、作業ツリーがきれいなときの枝、`off` 以外の値(`on`、`OFF`、空、未設定)の枝は、文字が変わっていない。`set -eu` のもとで `${VAR:-}` は未設定でも落ちない。`cmp` で root と `templates/base/` の hook 2 つ、`/org` と `/director` の skill 各 4 写し、`director-launch-cmd.md` の 2 写し、`git-commit-strategy.md` の 2 写しがバイト一致なことを確かめた
- `tests/test-wip-checkpoint-hooks.sh` を全部読んだ。trap は `EXIT` と `HUP INT TERM` を分けた形(`exit 1`)。一時 repo は `guard_temp_repo` が一時ルートの下で自分が top であることを確かめてから hook を打つ。repo 名は大文字小文字を区別しない file system のために連番。git の設定は `GIT_CONFIG_GLOBAL` と `GIT_CONFIG_NOSYSTEM` で隔て、`GIT_DIR` ほかを unset する。`off` の枝が常に commit する・常に飛ばす・`on` を飛ばす・大文字小文字を畳む、のどれにしても、`case_commits`(unset・`on`・`OFF`・空)か `case_off` のどちらかが赤になることを、判定を 1 つずつ変えたと仮定して読んだ(実行はしていない)
- Go の diff を全部読んだ。`driver/paneenv.go`(`SeatPaneEnv` は呼ぶたびに新しい slice を返す)、`herdr.go` の `warnNoPaneEnv`(消費する値は定数だけ。`paneEnvConsequence` は完全一致の switch)、`director_cmd.go` の起動スクリプト(名前は定数 2 つ、値は `shellQuote(value)` を通る。`strings.Cut` の第 3 戻り値を捨てるのは、一覧の要素が定数で `=` を持つため)、`internal/cli/org.go` と `org_director_headless.go`(本番の組み立て 2 か所が同じ一覧を使う)。`git grep AGMSG_SELF_NAME`(テスト・計画・記録を除く)で、pane の環境を書いた箇所に 2 つ目の値の書き漏れがないことを見た
- テストの diff を全部読んだ。`TestWIPCheckpointOff_IsWhatTheHooksRead` は hook の判定の文字列(`[ "${RALPH_WIP_CHECKPOINT:-}" = "off" ]`)を root と写しの 4 本で探し、`t.Fatalf` で落とす。`prompts_test.go` の 2 本は、仕様の文を `containsPhrase` で固め、`org: spawn step`・`seat left active`・`retry with a larger --timeout-ms` の 3 語を、実際の `Spawn` の誤りと `refuseStaleRespawn`・`spawnDeadlineHint` から引いて突き合わせている
- leader の雛形と skill の文を、コードの誤りの文字と突き合わせた。`spawn failed:` は `internal/cli/org.go:846`、`rejected:` は `:838`、`org: spawn step` と `seat left active` は `internal/org/spawn.go:2474`・`:2490`、`org: spawn: ... seat left active` は `:2607`、`retry with a larger --timeout-ms` は `:2498`。`stopSeatCommand`(`:2204`)は `--force` を書かない。手順 14 の「`<既定のブランチ>` は手順 9 の決め方」は手順 9(`leader.md` の 180〜186 行)に実在し、「件を上げる」の節は `leader.md:316` にある
- 記録の突き合わせ。`docs/evidence/seat-wip-checkpoint-2026-10-22.md` の「C-c を 1 回送り、間を置かず `herdr pane close`」は `internal/org/verbs.go:1203-1215` の `stopSeatPane` と合う。試行の訂正の時刻(05:56:46Z、05:56:52Z、05:57:43Z、05:57:55Z)は `docs/evidence/org-trial-2026-10-20/manifest.jsonl:72,73,74` と `inbox.jsonl:13` に実在する。`.codex/hooks.json` に `SessionEnd`・`PreCompact` はない。tech-debt の 192 行と 328 行を `re.split(r'(?<!\\)\|', ...)` で割ると 7 要素(5 欄)で、base の行と同じ形。取り消し線と `(RESOLVED ...)` と直上の RESOLVED コメント(191、327 行)がそろう。`fix/seat-spawn-close` は仕様・根拠の表・tech-debt から消え、`#249` になっている(残りは archive 済みの計画と過去の reports だけ)。仕様の PR 一覧は `#` の数で 32 本(241 と 242 を含む)で、本文の「32 本」と合う
- 追加行を `/Users/`、`TODO`・`FIXME`、`fmt.Print`・`println` で grep し、39 ファイルを U+FFFD で調べた。当たりなし。`git diff --check c90d01b4...HEAD` は空。templates 側の追加行に `internal/`・`cmd/`・`docs/plans`・`docs/evidence`・`docs/tech-debt` の参照はない
- skill と README の追加行を全角を 2 列に数えて測った(中央値 71〜74 列)。幅の外れは下の L1
- 確かめていない: Go とシェルのテストの実行(`/test` の範囲)、`go vet`・`gofmt`・shellcheck、仕様の `[x]` の数と根拠の表の突き合わせ(`/verify` の範囲)、実機の evidence の再現(herdr 0.7.5 と本物の claude が要る)、`scripts/check-sync.sh` と `check-skill-sync.sh`

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (L1) | readability | `/director` skill の再折り返しが半端。既存の行に新しい節を継ぎ足したため、103 列(全角 2 列)の行ができ、別の文の途中に 26 列の短い行が残る。同じブロックの中央値は 73 列。4 写し(`.claude/`、`.agents/`、`templates/base/.claude/`、`templates/base/.agents/`)で同じ | `.claude/skills/director/SKILL.md:228`(`compaction の前に ... そのあと、start を打った ralph の絶対パスで`)、`:284`(`届くのは \`ralph.toml\` の`) | 2 か所を折り直す。`sync-skills.sh` で写しを作り直し、`cmp` で 4 写しの一致を確かめる |
| LOW (L2) | maintainability | `git-commit-strategy.md` の Org Runtime Commits に足した 1 文が、指す先の節より狭い。「end-of-session WIP commit」とだけ書き、compaction の前の commit を落としている。「panes of seats」とだけ書くが、leader と herdr の headless の director の pane にも同じ値が付く(下の節の 47〜55 行はそう書く)。直前の段落(`progress.log` の文)に続けて空行なしで足したので、別の話に読めない | `.claude/rules/ralph/git-commit-strategy.md:29-31`(と `templates/base/` の写し)、同 `:47-55` | 「ralph turns off the WIP checkpoint commits below in the panes it creates」のように、下の節の語に揃える。同じ段落でなく別の段落にする |
| LOW (L3) | readability | leader の雛形の spawn 失敗の項で、場合分けの前に置いた文が無条件に読める。「BLOCKED は、TASK_ID を ...、EVIDENCE をエラーの文の要点にして、...上げ、返事が来るまでその座席の spawn を打たずに待ってください」は主題が BLOCKED で、いつ上げるかを言わない。続く (b) は「そのまま打ち直す」なので、場合分けを最後まで読まないと食い違って見える。座席に読ませる文なので、先頭の文だけ読んで動くと (b) の座席も待つ。また `/org` skill の手順 6 は `git log` を先に書き、そのあとに HEAD の比較を書く。雛形の手順 14 は HEAD の比較が先。結果は同じだが、2 つの文の順が逆 | `internal/org/prompts/leader.md:419-424`(この文は `prompts_test.go` の `TestRenderRolePrompt_Leader_SpawnFailedIsSortedByItsError` の最初の場合が一字一句固めている)、`.claude/skills/org/SKILL.md:891-897` と `leader.md:244-247` | 「BLOCKED を上げるときは、...」の形にして、場合分けの (a)・(c) が「上げる」側であることを (b) と並べて読めるようにする。直すなら固めているテストの文も同じコミットで直す。skill の順は雛形に揃える(4 写し) |
| LOW (L4) | maintainability | 試行の訂正と tech-debt の新しい行が、時刻だけを根拠にした推定を結論の形で書く。「So the `SessionEnd` hook of a seat that `ralph org stop` ended made the commit」。根拠は、commit の件名が session end の hook のものであること(これは確か)と、commit の時刻 05:56:51Z が implementer の stop(05:56:46Z)と reviewer の stop(05:56:52Z)のあいだに入ること。ただし 05:56:51Z は試行の notes の値で、同じ段落が「a record ではない」と認めている。座席のセッションが終わったことは時刻から読めるが、止められた座席だと断ずるには `claude -p` の子など別の session の終わりを除く記録がない。同じ PR の新しい evidence は「推定と合う(コミットを作ったセッションを特定したわけではない)」と弱く書いている | `docs/evidence/org-trial-2026-10-20.md:821-824`、`docs/tech-debt/README.md:328`(欄 1 の「So the hook of a seat that `ralph org stop` ended made it」)と `:327`(コメント)、対して `docs/evidence/seat-wip-checkpoint-2026-10-22.md` の「分かったこと」の 2 項目目 | 3 か所を「時刻が止められた座席の session end と合う(どの session かは記録にない)」の強さに揃える |
| LOW (L5) | maintainability | org-gate の batched 行の (o) が、この PR で前提を失った。(o) は「leader の手順 9 は reviewer のレポートを commit させるが、session end の hook が先に `wip:` で commit していることがある(2 回の live)ので、commit しない道を言うべき」と書く。新しい binary が `--env` のある herdr で作る pane では hook が commit しないので、その前提は新しい配備では出ない(古い herdr と片方だけ新しい配備では残る)。行にも PR にも注がない。(o) の後半(`git add docs/reports` が implementer の迷子のファイルを巻き込む)は残る | `docs/tech-debt/README.md:233`(欄 1 の (o))。この PR は同じ hook の 191・192・327・328 行だけ閉じた | (o) に「F11 の直し(fix/seat-wip-checkpoint)で、新しい pane では hook の commit が出なくなった。前半は古い herdr と片方だけ新しい配備に限る」と 1 文足す。足さないなら、PR の本文の既知の残りに書く |
| LOW (L6) | naming | `driver/paneenv.go` の名前と文が、使われ方より狭い・曖昧。(1) `SeatPaneEnv` は seat の pane だけでなく headless の director の pane と launch_cmd の起動スクリプトにも使われる(doc の 2 行目は「every pane it creates」と書く)。呼び出し側のコメントは「as a seat's does」と補っている。(2) `WIPCheckpointOff` の doc の最後の句「in a feature worktree the seats of an org share」が、「commit」にかかるのか「session」にかかるのか決まらない | `internal/org/driver/paneenv.go:9-12`、`:15-24`、`internal/cli/org_director_headless.go:786` | 名前は計画の D3 の語(pane に足す値の一覧)に合わせて `PaneEnv` 系の語にするか、doc に「seat と headless の director の pane」と書く。(2) は 2 文に分ける |
| LOW (L7) | readability | 値が 2 つになった warning の行が、区切りの `; ` を「2 つの帰結のあいだ」と「最後の助言の前」の両方に使う。1 行の中で 3 つの節が同じ区切りで並び、助言が 3 つ目の帰結に読める。文は 535 字の 1 行 | `internal/org/driver/herdr.go:339-341`、`paneenv.go:29-37`(固める文字は `herdr_test.go` の `noPaneEnvAdvice` ほか、`org_runtime_env_test.go` の `clause`) | 帰結を番号か `(1)`・`(2)` で区切るか、助言の前を `. ` にする。テストの文字も同じコミットで直す(`AgmsgSelfNameOff` だけのときの文は前の PR と一字一句同じ、と固めているので、そちらの扱いを決めてから) |

## Positive notes

- hook の diff は、commit の道を 1 つの `if/else` で包んだだけで、ほかの道の文字を変えていない。skipped の行は commit するはずだった道にだけ足すので、きれいな木・main・`off` 以外の値の md はバイトで変わらない。判定は完全一致(`"off"`)で、空・`on`・`OFF` は今までどおり commit する。写しとの `cmp` も一致
- 起動スクリプトの export は、値を `shellQuote` に通し、名前は定数。一覧の順で出し、check-in の前に置く。warning の文は定数だけから作り、`PaneEnv` に秘密を入れない注意が doc に残っている
- テストが挙動と文字の両方を固めている。シェルのテストは、実 repo を触らせない保険(`guard_temp_repo`)、大文字小文字を区別しない file system、house の trap の形、`off`/`OFF`/空/`on`/未設定を別々の repo で打つ作りになっている。Go のテストは、`SeatPaneEnv` の順と複製、本番の組み立て、起動スクリプトの実行(偽の ralph と偽の agent に環境を書かせる)、warning の全文、leader の雛形の文と順序、hook の判定の文字列まで固める
- leader の雛形が引く 3 つの誤りの語と `rejected:` の行が、実際の `Spawn` の誤りと `printSpawnResult` に実在する。テストが、誤りの文を作る側から語を引いて突き合わせるので、誤りの文の書き換えが雛形の指す先を黙って外さない
- 記録は、できなかったこと(古い herdr、Codex の座席、feature org 全体の実機)を RESOLVED のコメントに書き、実機の確かめが `herdr tab create --env` を直接打ったもので `ralph org spawn` を通していないことも書いている。evidence は `$HOME` を `~`、一時ディレクトリを `<D>` にし、pane・session・commit の id を写していない

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| なし(L1〜L7 はすべて文面の LOW)。依頼どおりこの報告書以外のファイルは変えていないので、`docs/tech-debt/README.md` には足していない。最後の cycle で直さない LOW が残れば、1 行にまとめて足す | - | - | - | `docs/plans/active/2026-10-22-seat-wip-checkpoint.md` |

## Recommendation

- Merge: OK(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 7)
- Follow-ups: L1 と L3 は skill・雛形の文を直すので、直すなら 4 写しと固めているテストを同じコミットで直し、パイプラインを `/self-review` から回し直す(cycle 1 の次が最後の cycle)。L4 は記録の文字だけだが、直す先の `docs/evidence/org-trial-2026-10-20.md` と `docs/tech-debt/README.md` は、`post-implementation-pipeline.md` の例外(`/cross-review` だけが出す、この task の report と insight への修正)の対象ではないので、直すなら L1・L3 と同じ回し直しに入れる。L2・L5・L6・L7 は、直さないなら 1 行にまとめて tech-debt に足す。insight event は未実行(依頼が他のファイルを変えないとしたため): `./scripts/insights-append.sh --slug seat-wip-checkpoint --flow standard --phase self_review --verdict pass --critical 0 --high 0 --medium 0 --low 7 --cycle auto --source skill`
