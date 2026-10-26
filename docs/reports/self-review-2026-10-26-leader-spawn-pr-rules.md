# Self-review report: leader-spawn-pr-rules

- Date: 2026-10-26
- Plan: `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md`(S1〜S5、AC1〜AC6、D1〜D6)
- Reviewer: reviewer サブエージェント(`/self-review`、pipeline cycle 1/2)
- Scope: `git diff 619cf5f8...HEAD`(HEAD f9fe07ea、12 ファイル、+354/-59)。コミットは 70cc1d77(計画)、efddd301(`feature.go`・`leader.md`・テスト 2 本)、c654d596(`/org` skill の 4 写し)、7d51236d(tech-debt・仕様・根拠の表)、f9fe07ea(計画の印)。テスト、静的検査、仕様の照合は `/verify` と `/test` の範囲で、この report では打っていない

## Evidence reviewed

- `featureLeaderTask` の本体と doc comment(`internal/org/feature.go:377-420`)を全文読み、`- 台帳:` の行(同 `:416` の `fmt.Sprintf("- 台帳: %s(ralph org のコマンドには必ず --state-dir %s を付ける)", stateDir, shellQuote(stateDir))`)と新しい `- 予約したパス:` の行(`:412`)を並べた。形は同じで、表示用の素の値と `shellQuote` の引用を 1 回ずつ書く。`shellQuote`(`:448`)は単一引用符を `'\''` にする POSIX の 1 語で、予約の値(パスを `, ` でつないだもの)に使って正しい。パスは空白とカンマを持てない(`normalizeReservePath`、`reserve.go:209`)ので、`, ` のつなぎは曖昧にならない。`parseSplitReserve`(`split.go:562`)は空の値を断り、`finishSplitPlan`(`:612`)は `Reserve` の行がない機能を断るので、`--scope ''` を書く経路はない
- doc comment の主張を 3 つ照らした。`autonomousScopeGateErr`(`spawn.go:1693`)は実在し、implementer と reviewer の座席には `--reserve` が使えない(エラー文「--reserve on the leader seat only」)ので「`--scope` のない座席は断られる」は合う。`normalizeReservePath` が `$`、バッククォート、引用符を通すことはテストの fixture 検査(`got != p` なら `t.Fatalf`)が毎回確かめる。`--scope` を読むコードは `prompts.go:112` の `{{SCOPE}}` の置換と `spawn.go:1711` の `scope=%s` の記録だけで、`scope=` を分けて読む処理は `internal/` にない(`grep -rn 'scope=' internal cmd --include='*.go'` の非テストの当たりは 3 行とも書く側)。`, ` を含む値を渡しても動きは変わらない
- `TestFeatureLeaderTask_ScopeIsOneShellWord`(`feature_test.go:593` から)を、兄弟の `TestFeatureLeaderTask_StateDirIsOneShellWord`(`:566` から)と並べた。解析は `HasPrefix` で行を探し、`CutPrefix` と `CutSuffix` で括弧の中を切る。予約のパスは空白を持たないので、`を付ける)` が予約の中に現れて切り損ねることはない。fixture の 7 組は、一つのパス、複数、`$`、バッククォート、単一引用符、二重引用符、バックスラッシュとグロブを覆う。`strings.SplitSeq` は go.mod の 1.25.8 で使え、`watch_test.go:3001` が前例
- `/bin/sh` の是非。`exec.Command("/bin/sh", ...)` は `internal/cli/org_director_headless_test.go:2319` と `org_director_cleanup_test.go:65` に、`exec.CommandContext(ctx, "/bin/sh", ...)` は `internal/org/prompts_leader_inbox_test.go:351` にあり、`internal/org/director_cmd_test.go` には `/bin/sh` が 20 行ある。そのうち `:810` は、引数(`$HOME`、`*`、バッククォート)が壊れずに届くことを `/bin/sh` で確かめる、今回のテストと同じ目的のもの。兄弟のテスト(`feature_test.go:576`)は PATH 引きの `"sh"` を使う。どちらの形も前例があり、herdr・agmsg・gh のような外の道具ではない。問題なし
- leader の雛形の差分(`leader.md`)の 4 か所を、周りの手順と並べて読んだ。ミッションの 2(`:19-27`)、手順 3(`:120-143`)、手順 5(`:153-170`)、手順 14(`:273` から)。`--scope` の値は 3 か所とも「タスクの `- 予約したパス:` の行の括弧にある引数」で、手順 5 が「書き換えずに付ける(引用符を足したり外したりしない)」と言い、手順 3 は「手順 5 と同じ flag」、ミッションは手順 5 を指す。「機能ごとの org」の導入(`:61-70`)の「予約の外に書く手順(計画、reviewer のレポート、report と計画の移動、計画の戻し)は予約に入っていなくてよい」と、手順 5 の「reviewer の雛形は report と gate の生成物を scope の外でも書いてよい」が合う(`reviewer.md:10-12`、`implementer.md:21` の `git add <path>...` は scope 内だけ)。導入の「`ralph org` のコマンドには `--state-dir` をそのまま付ける」(spawn を含む)と、手順 5 の `--scope` は別の flag で重ならない
- `/org` skill の差分を読み、4 写しを `cmp` で突き合わせた(3 つとも `.claude/skills/org/SKILL.md` と同一)。手順 2(`:1306-1313`)は `leader.md` の手順 3・5 と同じ向き、「org の終わり方」の手順 6(`:1002-1010`)は手順 14 と同じ向き。skill の手順 2 は `--cwd .` を昔から書かない(今回の差分で変わっていない)ので、承認前の reviewer の括弧「`--model` と `--scope` は上と同じにする」と `leader.md` 手順 3 の「`--cwd .`・`--model`・`--scope`」は食い違いではない
- 手順 14 が指す先を開いた。`.claude/skills/pr/SKILL.md:31` に「PR title and body must be written in Japanese」と題の prefix の決まりがあり、`template.md` は同じフォルダにある。`.agents/skills/pr/SKILL.md`、`templates/base/.claude/skills/pr/SKILL.md`、`templates/base/.agents/skills/pr/SKILL.md` も実在する。雛形の「`/pr` skill そのものは実行しない」の理由(`pr/SKILL.md` 手順 10 が task worktree と local branch を消す)は、今も当たる
- 記録の文を、コードと表に照らした。tech-debt の RC2・RC5 の行は `(?<!\\)\|` で分けると 5 セルで、取り消し線はセル 1 とセル 4、`<!-- RESOLVED ... -->` の注は前例(`:339`、`:343`、`:351`)と同じ位置と形。RESOLVED の注の主張(`featureLeaderTask` の行の形、手順 5・3・ミッションの 2、skill の 4 写し、テスト名と fixture の種類、`/pr` skill と `template.md` を変えていないこと、「実機では動かしていない」)をコードとテストで確かめて合った。行の「Related」のセルの `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md` は `/pr` の `archive-plan.sh` が tech-debt に限って書き換える(`pr/SKILL.md:57`)
- 仕様の `- [ ]` を数えた。行頭が `- [ ]` のものは base も HEAD も 5、`- [x]` は 27 で、新しい Status の段落の「`[ ]` は 5 項目のまま」と合う。RC の行を根拠の表で並べると RC2 と RC5 が直った行になり、RC1、RC3、RC4、RC6、RC9〜RC11 は「未定」、RC7 は先行の PR、RC8 と RC12 は行なし
- 秘密と debug の掃引。追加行に鍵・token・`/Users/`・メール・`console.log`・`fmt.Print`・`TODO`・コメントアウトしたコードはない。`featureLeaderTask` の出力は `shellQuote` を通るので、予約のパスの `$` やバッククォートを leader の shell が展開する経路はない(AC1b のテストが `/bin/sh` で確かめている)

## Findings

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW (F1) | maintainability | 記録の「テストで確かめた」が、テストの中身より強い。仕様の Status は「どちらもテストで確かめたが、実機では動かしていない」と RC2 と RC5 をひとまとめにし、根拠の表の FR-4 の行は RC2 を「`TestFeatureLeaderTask_ScopeIsOneShellWord` と `TestRenderRolePrompt_Leader_FeatureOrgProcedure` で確かめた」と書く。この 2 つのテストが固定するのは、タスクの行の形、`/bin/sh` が引数を 1 語に読むこと、雛形の文の字面で、leader がその文に従うことも、spawn の検査をその値が通ることも打っていない。RC5 のテストは文の字面の pin だけで、leader が日本語で書くことは何も確かめていない。同じ PR の tech-debt の注は「pins ... What is not done: ... not verified」と正確に書いており、3 つの記録の強さが食い違う | `docs/specs/2026-10-07-org-multi-org-director.md:37`(Status の段落の「どちらもテストで確かめたが」)、`docs/evidence/org-spec-evidence-2026-10-21.md:76` の FR-4 の行の末尾の「で確かめた)が、実機の記録がないので」、`docs/tech-debt/README.md:364`・`:368` の注 | 「テストで雛形の文面とタスクの行の引用を固定した。leader がそのとおりに打つかは実機で確かめていない」の形にそろえる。FR-4 の行も「で固定した」にする。`[ ]` を残す理由(実機の記録がない)はもう書いてあるので変えない |
| LOW (F2) | maintainability | 仕様の Status の段落が「RC1、RC3、RC4、RC6、RC9〜RC12 は直しておらず、開いたまま残る」と RC12 を含める。根拠の表の RC12 は「直す PR」も「tech-debt の行」も「なし(利用者の設定で、repo の欠陥ではない)」で、開いた項目がない(RC8 も行なしで、こちらは段落が数えていない)。読み手は RC12 に未解決の作業があると読む | `docs/specs/2026-10-07-org-multi-org-director.md:37`、`docs/evidence/org-spec-evidence-2026-10-21.md` の RC12 の行 | 「RC1、RC3、RC4、RC6、RC9〜RC11 は直しておらず開いたまま残る。RC12 は利用者の設定で repo の欠陥ではないので、行を作っていない」と分ける |

## Positive notes

- `- 予約したパス:` の行が `- 台帳:` の行と同じ形(素の値と `shellQuote` の引用を 1 回ずつ、括弧の中に「付ける flag」)で、leader の読み方がそろう。D6(Codex の計画助言 HIGH)の指摘どおり、leader に引用させず ralph が引用した語を書いている
- AC1b のテストは、`normalizeReservePath` が実際に通す文字だけを fixture にし(通らない fixture を `t.Fatalf` で止める)、`$`・バッククォート・単一引用符・二重引用符・バックスラッシュ・グロブを `/bin/sh` に読ませる。前提が崩れたときに、通らない入力で緑になることがない
- `TestRenderRolePrompt_Leader_FeatureOrgProcedure` は既存の pin(`--id` の文、手順 14 の 3 句、ミッションの「implementer 1 席と reviewer 1 席を既定」)を残して足している。`featureLeaderTask` の行の形そのものも pin(`prompts_test.go:557`)するので、行の形を変えると雛形のテストも落ちる
- 記録が「実機では動かしていない」と何度も書き、FR-4 と受け入れ条件 5 を `[ ]` のまま残している(印の規則と D4 に合う)。`[ ]` の数も 5 のまま

## Non-findings (取り上げない理由)

- 仕様の新しい Status の段落(`docs/specs/2026-10-07-org-multi-org-director.md:37`)が計画を `docs/plans/active/2026-10-26-leader-spawn-pr-rules.md` と書くこと。`/pr` 手順 8 の `archive-plan.sh` が書き換えるのは `docs/tech-debt/README.md` の参照だけ(`pr/SKILL.md:57`)なので、`/pr` のあとに仕様の参照は切れる。ただし同じ形は仕様の `:35`(`docs/plans/active/2026-10-24-org-recheck.md`)と根拠の表の 3 行目(`docs/plans/active/2026-10-21-org-spec-complete.md`)に前例があり、どちらも `docs/plans/archive/` に動いたまま残っている。evidence の見出しの active パスは前の review(`director-liveness-livefire`、`org-trial`)でも取り上げないと決めてある。tech-debt の 4 か所は script が直す。この repo の約束として取り上げない(直すなら 3 か所まとめて)
- `/bin/sh` を打つこと。前例が 2 つの package に多数あり、外の道具ではない(Evidence reviewed の 4 つ目)。兄弟のテストの `"sh"` との違いは前例のどちらの形にもある範囲で、そろえなくてよい
- `arg == ""` の確認(`feature_test.go:621`)は、`shellQuote` が空を返さないので到達しても意味がなく、`arg` が空なら直後の比較(`"0|"` と `"1|"+want`)が落ちる。害はなく、取らなくてもよい程度
- `featureLeaderTask` の doc comment の 1 段落目の折り返しが新しい語句のために 4 行ほど変わった。必要な範囲の整形
- RC2・RC5 の tech-debt の行を RESOLVED にし、実機の確かめを仕様の `[ ]` の印に残す形(D4)は、行が「何をまだしていないか」を注に書いているので、半分だけ直した行の取り消し(`pattern-techdebt-half-closure`)には当たらない。RC2 の行の文が挙げる欠陥は 1 つ(雛形が `--scope` を書かない)で、RC5 の行の 2 つの事実(言語の決まりが skill の手順にあり `template.md` にない、題の prefix は付いていた)は 1 つめが直り 2 つめは直す必要がない

## Tech debt identified

なし(F1・F2 は 1 文ずつの直しで、`docs/tech-debt/` には足さない)。この段が最後の cycle(2/2)で直さずに持ち越す場合は、F1・F2 を 1 行にまとめた行を足す。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |

_(行は足していない。)_

## Recommendation

- Merge: OK(CRITICAL なし、HIGH なし、MEDIUM なし、LOW 2 件。コードの直しは要らない)
- Follow-ups:
  - F1・F2 はどちらも `docs/specs/2026-10-07-org-multi-org-director.md` と `docs/evidence/org-spec-evidence-2026-10-21.md` の文の直し。`docs/reports/` の記録の例外(`.claude/rules/ralph/post-implementation-pipeline.md`)には入らず(仕様と根拠の表は対象外)、直すと全 pipeline の再実行になる。cycle 1/2 なので、直すなら次の run の前にまとめて、`/cross-review` のあとに出る指摘と一緒に直すのが安い
  - 確かめていないこと(この report の範囲外): Go のテストと `gofmt`、`check-skill-sync.sh`、`run-verify.sh`(`/verify` と `/test`)。leader が実際に `--scope` を付け、PR を日本語で書くか(実機。計画の R3、次の実行で確かめる)
