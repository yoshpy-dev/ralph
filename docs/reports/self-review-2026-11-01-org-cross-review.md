# Self-review report: org-cross-review

- Date: 2026-11-01
- Plan: `docs/plans/active/2026-11-01-org-cross-review.md`(issue #269、D1〜D10、D6b、D7b、AC1〜AC9、スライス S1 e3c04e18・586f7934、S1b a420eae0、S2 2478e069、S1c aa0a0fd5、S2b 82df1474、S3 93d58b65、S4 91c3798b、S5 106bc425)
- Reviewer: reviewer サブエージェント(`/self-review`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(base 4fef6b8c、HEAD e84c3054、31 ファイル、+3865/-160)。reviewer の雛形の `## cross-review の TASK`、leader の雛形の手順 9 の段落・`### cross-review`・ミッション 5・手順 12 の停滞・手順 14 の PR の本文と RESULT、計画助言の codex の `--disable apps`、`internal/org/driver/paneenv.go` の警告文と `spawn.go`・`config.go` のコメント、`templates/base/ralph.toml` のコメント、`/director` と `/org` の skill と写し、README・recipe・`docs/reports/README.md`・仕様の注、evidence、新しいテスト 3 本と変えたテスト 4 本。命名、読みやすさ、null の扱い、例外処理、security、保守性の 6 項目で見た。機械的な項目(不要な変更、誤字、デバッグ残り、秘密情報、slice 外のファイル)は実装側が見る範囲なので探していない。
- ID: 所見は M1(MEDIUM)と L1〜L7(LOW)。

## Evidence reviewed

- `git diff origin/main...HEAD` の全体と、`git log --oneline origin/main..HEAD` の 13 コミット。
- 雛形のコマンドの抜き出しと実行(scratchpad): leader の `### cross-review` の最初のコマンドを `{{ORG_ID}}` = `org-a` で抜き出し、main に org-a の complete の report c1・c2 があるリポジトリで新しいブランチを切って bash で打った(M1 の根拠)。テストもほかの静的検査も走らせていない(`/verify` と `/test` の範囲)。
- 写しの比較: `/director` と `/org` の skill の 4 つの写しは `.claude/skills/` とバイト単位で同じ(`cmp`)。`docs/recipes/codex-seat-permissions.md`、`docs/recipes/isolated-herdr-server.md`、`docs/reports/README.md` は `templates/base/` の写しと同じ。
- 古い記述の掃除: `git grep` で「seat's cross-review」「座席の cross-review」「treats the seat's driver as claude」「後の版から」を探した。残りは README の 278 行(計画が sync-docs に回すと書いた分)と、仕様の 2026-10-29 の決定の記録(日付つきの記録なので残してよい)だけだった。

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | 正しさ(回数の数え方) | M1: 上限に数える report を、このブランチが足したものに絞っていない。`git ls-files` は base から来た report も出すので、同じ org_id が前に使われてその PR が main に入っていると、前の org の report が `done:` と `last:` に入る。scratchpad で、main に org-a の complete の report c1・c2 がある状態で新しいブランチを切って打つと、`done: 2`・`cap: 2`・`since: changed` が出た。leader はこの org で cross-review を 1 回も送らずに、上限の QUESTION の 2 つ目(「最後の complete の回のあとに見せていない変更がある」)を人に上げる。前の org の report が 1 本なら、この org は 1 回しか回せず、それを誰も知らない。org_id の使い回しは repo が想定している場合で(`internal/org/cleanup.go:34-36` の「a worktree another feature made on the same org_id」、`internal/org/approval.go:373` の「an approval from an earlier life of the same org_id never counts」)、ここだけが前の org の記録を数える。手順 14 の `Cross-review:` の行に並べる report のパスにも、前の org の report が入る | `internal/org/prompts/leader.md:798-799`、`:835-846`(`last:` と `since:` の説明)、`:602-610` と `:989-994`(手順 14 の値と report のパス) | 数える対象を、このブランチで足した report にする。手順 9 の fetch のあとなので `origin/<既定のブランチ>` は読める。たとえば `git diff --name-only --no-renames --diff-filter=A "origin/<既定のブランチ>...HEAD" -- 'docs/reports/review-xreview-*.md'` を `git ls-files` の代わりに使う(コマンドに既定のブランチの決め方を入れるか、手順 9 で決めた名前を書き換える場所を作る)。`TestRenderRolePrompt_Leader_XreviewStateCommandCountsTheReviewersReports` に、base のブランチにある同じ org_id の complete の report を数えない場合を足す |
| LOW | 正しさ(上限を上げる答え) | L1: 上限の QUESTION に a で答えたときの新しい上限を、leader と director が別の式で決めている。director は人が数を言わないとき、`EVIDENCE:` の report の `- Cycle: <n>/<上限>` の `<上限>` + 1 を DECISION に書く。leader は 2 つ目の QUESTION の a で「すぐ TASK を送る」が、TASK の `CYCLE:` の `<上限>` は `cap:` の値で、それは cap のファイルを書く前に出した値なので、TASK は `CYCLE: 3/2` になり、reviewer はそのまま report に写す。この回のあとに上限の QUESTION がまた上がると、director は 2 + 1 = 3 を返し、`done:` が 3 のまま `cap: 3` になって同じ問いがすぐ戻る。人が 4 以上の数を言うまで a が効かない。leader の側の「書いていなければ `done:` + 1」は、D7b が同じ問いの繰り返しを避けるために選んだ式だが、director はいつも数を書くので使われない | `internal/org/prompts/leader.md:894-901`、`:922`、`.claude/skills/director/SKILL.md:869-874` | director は、人が数を言わなければ DECISION に数を書かない(leader が `done:` + 1 を使う)か、`<n>` + 1 にする。leader の 2 つ目の QUESTION の a は、cap のファイルを書いたあとに小節の初めのコマンドを打ち直してから TASK を送る、と書く。写しの 4 つと固定のテスト(`skill_xreview_test.go` の上限の節)も合わせる |
| LOW | 読みやすさ(director の読む場所) | L2: `/director` の「cross-review の上限」は `EVIDENCE:` の report を読めと言うが、どこで読むかを書いていない。report は機能のブランチにしかなく、director が動く main worktree のルートにはない。同じ skill の「計画の承認」は「機能の worktree(`.claude/worktrees/org-<org_id>`)で」と場所を書いている。director が report を開けなければ、1 の「どちらの件か」も、3 の今の上限も読めない | `.claude/skills/director/SKILL.md:854-857`、`:872-874`、比べた箇所 `:794` | 「機能の worktree(`.claude/worktrees/org-<org_id>`)の `EVIDENCE:` の report」と書く(写し 4 つ) |
| LOW | 読みやすさ(手順 9 の段落の頭) | L3: 差し込んだ段落も、そのあとの sync-docs の段落も「rebase が済んだら(要らなかったときも)」で始まる。差し込んだ段落は「cross-review が終わったら … 次の段落の sync-docs に進む」と言うが、sync-docs の段落の頭は sync-docs を rebase の直後に置いたままに読める。compaction のあとに sync-docs の段落から読み直した leader は、cross-review を飛ばしうる。`/org` skill も同じ形 | `internal/org/prompts/leader.md:351`、`:362`、`.claude/skills/org/SKILL.md:1074`、`:1106` | sync-docs の段落の頭を「cross-review が終わったら(off で送らなかったときも)」などに変え、その文を固定する `syncDocsHandOff`(`internal/org/prompts_leader_syncdocs_test.go:20`)と `orgSkillSyncDocsHandOff`(`internal/org/skill_doc_followup_test.go:19`)を合わせる |
| LOW | 保守性(理由の長さと行き先) | L4: incomplete の理由に入る `log:` の行は、log の 1 行をそのまま出し、長さを切らない。その値は report の `Reviewer status:`、reviewer の RESULT の `XREVIEW: incomplete (...)`(本文は 2,000 字まで)、leader の PR の本文の `Cross-review:` と RESULT の `CROSS_REVIEW:` に写る。codex の log の最後の行が API のエラーの JSON なら数百字になりうる。leader の RESULT の `DOC_FOLLOWUP:` には 2,000 字に収まらないときの書き方があるが、`CROSS_REVIEW:` にはない。PR の本文は公開されるので、log の行に入ったローカルの絶対パスもそのまま出る | `internal/org/prompts/reviewer.md:247`、`:276`、`:302-313` | コマンドの側で切る(`l=$(grep '[^[:space:]]' "$p.log" \| tail -n 1 \| cut -c1-200)`)。手順 4 に「理由は 1 行で、200 字まで」と書き、テストの `failLog` の比較はそのまま通る |
| LOW | 保守性(helper の説明) | L5: `pick_reviewer` の説明は呼び出し元を 3 つと書き、「All three callers pass the driver they resolved: the CLI running the skill …, which RALPH_PRIMARY_CLI … overrides」と言う。この diff で reviewer 座席の cross-review が 4 つ目の呼び出し元になり、渡すのは自分の CLI ではなく TASK の implementer の driver で、`RALPH_PRIMARY_CLI` は効かない(D6 が頼る「引数があれば env を読まない」性質)。説明だけを読んで helper の引数の扱いを変えると、この呼び出しの前提を確かめないまま変えうる | `scripts/xreview-helpers.sh:87-100`、`templates/base/scripts/xreview-helpers.sh:87-100` | 4 つ目の呼び出し元と、引数で implementer の driver を渡し env を読ませないことを、2 つの写しに足す。この slice の外として sync-docs に回すなら、その旨を計画の Progress に書く |
| LOW | security(evidence の伏せ字) | L6: codex の出力を写した行に、scratchpad の絶対パスがそのまま入っている。パスには利用者名と session の ID が含まれる。repo の evidence は `$HOME` を `~` に直して載せる約束で、この行だけが外れている | `docs/evidence/xreview-live-probe-2026-11-01.md:68` | `<scratchpad>/probe-269/internal/org/prompts/reviewer.md:229-231` のように伏せる。同じファイルのほかの行(`~/.codex/memories/MEMORY.md` など)はすでに伏せてある |
| LOW | 保守性(計画の記録) | L7: 計画の記録が diff に追いついていない。Affected areas は新しい `internal/org/prompts_xreview_test.go` と、`prompts_leader_syncdocs_test.go`・`director_skill_sync_test.go` の変更を挙げるが、実際に作ったのは `prompts_xreview_leader_test.go`(920 行)・`prompts_xreview_reviewer_test.go`(1,054 行)・`skill_xreview_test.go`(382 行)で、挙げた 2 つは変えていない。Progress はこの分け方を書いていない(S1c と S2b の outline の行にファイル名が出るだけ)。Risks の「`leader.md` が 975 行からさらに伸びる … 大きさは tech-debt で扱う」に当たる tech-debt の行もない(`leader.md` は 975 → 1,225 行、`reviewer.md` は 224 → 519 行) | `docs/plans/active/2026-11-01-org-cross-review.md:42-43`、`:113`、`docs/tech-debt/README.md`(行なし) | Affected areas を実際のファイルに直すか、Progress に分けた理由を書く。tech-debt に 1 行足す(案は下の Follow-ups) |

## Positive notes

- TASK の本文の行は両方の雛形で同じ並び(`IMPLEMENTER_DRIVER:`・`XREVIEW_BASE:`・`CYCLE:`・`REPORT:`・`REVIEWER_CODEX:`・`REVIEWER_CLAUDE:`)で、`reviewerXreviewTaskKeys` が reviewer の節から読んだ並びで leader の側を確かめる。`XREVIEW:` の値は、reviewer が返す 5 つ(pass、action_required、skipped、incomplete、git_changed)と、行が欠けたときの QUESTION のすべてに leader の枝がある(`leader.md:957-987`)。上限の QUESTION の `XREVIEW: cap_reached` のヘッダと `FORK 1:` の行は、`validateEscalation` と `protocol.HeaderValues` で読めることをテストが確かめ、`/director` の答えの `FORK 1: <記号> <案>` を leader が読む。
- reviewer のコマンド: `IMPLEMENTER_DRIVER:` の値を一重引用符の `case` で先に確かめるので、`$(echo claude)` のような値も展開されずに `driver: invalid` になる(テストあり)。CLI の確かめは `sh -c 'command -v claude'` で、zsh の alias を CLI とみなさない(alias だけの場合のテストあり)。codex の 3 回の呼び出しはどれも `--disable plugins --disable apps` を持ち、`-c` はすべて `exec` の前にある。`select(.enabled != false)` で、欄のない server も有効と数える(閉じる側に倒れる)。見張りの形と、前後の git の状態の比較(未追跡のファイルの中身の hash を含む)は計画助言と同じで、出力はすべて無視される `.harness/state/` に書く。
- leader のコマンド: `case "$rc:$v"` で 0・1・127 とそれ以外を分け、`cap:` は正の整数だけを受け、cap のファイルは 1 行目だけを読み、設定より下げない(18 の場合のテストあり)。report の名前は正規表現で日付と org_id を固定し、`${r##*-c}` は最後の `-c` を取るので、org_id に `-c` を含んでも番号を読み違えない(`org-a-cache` のテストあり)。`since:` は `git diff` が失敗すれば `changed` に倒れ、人に聞く側に寄る。
- テスト: 抜き出したコマンドを bash と `zsh -f` でスタブに対して打つ。`RALPH_PRIMARY_CLI` に implementer と同じ側の値を入れて、env を読む実装なら落ちるようにしてある。背景のジョブの通知が行の間に入っても通る `reviewerXreviewHasLines` で、負荷で落ちた 2 か所(S1c)を直した。見張りの確かめは見張りだけが出せる `rc=143` で見る。
- `paneenv.go` の警告文は、`leader.md:672` の形(`claude` か `codex` のときだけ env が自分の CLI に勝つ)と合う。新しいテストの 2 本は、警告に「a seat's cross-review」が戻らないことも確かめる。

## Coverage gaps

- テスト、静的検査、写しの検査は走らせていない(`/verify` と `/test` の範囲)。
- reviewer の codex のコマンドについて、`--disable apps` を知らない古い codex の場合(`mcp: not off (enabled: )` から `skipped (codex: MCP servers not off)` へ)を打つテストはない。計画助言の側には `STUB_OLD=apps` の場合がある。`/test` で見る点として残す。
- 座席の中(codex の sandbox と network、pane の環境)での実行は、計画どおり確かめていない(AC9 の known gap)。
- 新しいテストの 2 本は 800 行の目安を超える(920 行と 1,054 行)。同じパッケージには 6,000 行のテストもあり、話題ごとに分けてあるので所見にはしなかった。L7 の tech-debt の行に入れる案にした。

## Recommendation

- Merge: yes(CRITICAL と HIGH はない。M1 は同じ org_id の 2 回目の機能でだけ起き、その場合でも人に聞く側に倒れるが、PR の前に直すことを勧める)
- Follow-ups:
  - M1 と L1〜L7 の直し。M1 と L1 はコマンドと固定のテストの両方を直す。
  - tech-debt の行の案(L7): 「`internal/org/prompts/leader.md` は #269 で 975 行から 1,225 行、`reviewer.md` は 224 行から 519 行になった。`prompts_xreview_leader_test.go`(920 行)と `prompts_xreview_reviewer_test.go`(1,054 行)は 800 行の目安を超える。役割の雛形を節ごとのファイルに分けるか、座席が読む量を測ってから決める」。この commit は report と event だけに絞るので、tech-debt には書いていない。
  - L5 と README の 278 行(計画が sync-docs に回した分)は、sync-docs でまとめて見られる。
