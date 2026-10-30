# Verify report: leader-sync-docs

- Date: 2026-10-30
- Plan: `docs/plans/active/2026-10-30-leader-sync-docs.md`(issue #268、AC1〜AC8、D1〜D5 と D2b・D2c、Progress の S1〜S3・F1〜F3・inline の 8556265e)
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 1)
- Scope: `git diff origin/main...HEAD`(origin/main 0351800d、HEAD 0dfc49f0、17 ファイル、+2511/−54)。0dfc49f0 は origin/main(#290、org-state-dir-local)の取り込み。spec の準拠と静的解析と文書のずれだけを見た。Go のテストと `./scripts/run-test.sh` は /test の担当なので打っていない
- Evidence: `docs/evidence/verify-2026-10-30-leader-sync-docs.log`(`.gitignore:61` の `docs/evidence/*.log` に当たるのでコミットしない)。run-static-verify 自身のログは `docs/evidence/verify-2026-10-30-091239.log`(同じく対象外)

## Spec compliance

| Acceptance criterion | Status | Evidence |
| --- | --- | --- |
| AC1 sync-docs は手順 9 の末尾、手順 10 の前。手順 10 の 1 行目は `` `ralph org report`` のまま | met | `internal/org/prompts/leader.md:333-345` の fetch と rebase のあと、`:348` の「rebase が済んだら(要らなかったときも)」から sync-docs が始まり、`:432` の手順 10 は `` 10. `ralph org report --org-id {{ORG_ID}} --state-dir <台帳>` `` で始まる。sync-docs のコミット(`:421-428`)は手順 11(`:449`)と 12 の gate(`:453`)より前 |
| AC2 doc-maintainer への委任、D2b の項目、変わったパスの確かめ、すべて戻す、1 コミット、定義がないときの skip | met | 呼び方 `leader.md:350-351`(`Task(subagent_type="doc-maintainer")`、`.codex/agents/doc-maintainer.toml`)。定義がないときの skip `:351-356`、呼び出しの失敗と report なしの枝 `:356-368`。D2b の項目は `:381-399` にすべてある(計画のパスと base、予約したパスと読み方、予約の中だけ、凍らせたファイル、写しの組、写しの検査、report のパスと `## 予約外の文書の follow-up`、insight event・ステージ・コミット・計画の変更・`ralph org` の禁止)。確かめと戻し方 `:401-412`、決まったメッセージの 1 コミット `:421-428`。定義の実体は `templates/base/.claude/agents/doc-maintainer.md` と `templates/base/.codex/agents/doc-maintainer.toml` にある |
| AC8 写しの組の片方だけが予約の中なら直さない。予約の外を見つけたら一部でなくすべて戻す | met | `leader.md:387-391`(組のすべてが予約の中のときだけ直し、組の写しを 1 つ 1 行ずつ follow-up に書く)、`:414-417`(一部だけを戻すことはしない、と理由)。`/org` skill は `.claude/skills/org/SKILL.md:1085-1089`、`:1111-1113` |
| AC3 `/org` の「座席内 fan-out」が leader の subagent を認め、「org の終わり方」に sync-docs がある | met | `.claude/skills/org/SKILL.md:1577-1583`(leader の doc-maintainer の子、manifest に出ない、`max_seats` に数えない、送信しない)。「org の終わり方」の 1 の末尾 `:1047-1128`。完了条件 `:1922-1934` にも sync-docs のコミットと `DOC_FOLLOWUP:` がある |
| AC4 RESULT の例の `DOC_FOLLOWUP:`、PR の本文の節、`/org` にも同じ欄 | met | 例 `leader.md:590-591`、PR の本文 `:558-567`、RESULT の本文と 2,000 字を超えたときの 1 行と skip の行 `:573-581`。`/org` は `.claude/skills/org/SKILL.md:1230-1250` |
| AC5 `/director` の RESULT の拾い方(note は件数と URL の 1 行・500 字)と、全機能のあとの文書の分割計画 | met | 「件の処理」の RESULT `.claude/skills/director/SKILL.md:655-675`(note の URL は merge-order の PR の列から取る、500 字)。新しい節「### 予約外の文書の follow-up」`:1086-1148`(`merged` か `closed` で始める、`inbox --all` と `gh pr view`、写しの組は組のすべてを Reserve、印 `元の分割計画:`、承認のあと今の手順で start、文書の分割計画の follow-up は報告だけ)。節が使う CLI の形はソースと合う(下の「Observational checks」) |
| AC6 文書の変更で落ちた gate は今の `GATE: fail` に乗り、やり直しで sync-docs がもう一度走り、これまでの sync-docs のコミットのファイルの和集合を凍らせる | met | `leader.md:521-526`(`GATE: fail` の項の追記)、`:427-431`(やり直しで sync-docs も走る、最後の回のものを使う)、`:368-380`(件名の完全一致の `git log` と `awk`、`git show --name-only --format=` の和集合、report は除く)。`/org` は `:1066-1079`、`:1124-1128`、`:1192-1196` |
| AC7 新しい固定のテスト、今のテスト 5 本は変えずに通る、`check-skill-sync.sh` と `check-sync.sh` が通る | partly | 新しいテスト `internal/org/prompts_leader_syncdocs_test.go`(529 行)と `internal/org/skill_doc_followup_test.go`(771 行)がある。名指された 5 本(`prompts_test.go`、`prompts_verification_test.go`、`prompts_leader_plan_test.go`、`director_skill_sync_test.go`、`org_skill_send_sync_test.go`)は `origin/main...HEAD` の diff に入っていない(変えていない)。`check-skill-sync.sh` と `check-sync.sh` は通った。テストが通るかは /test が確かめる |

## Static analysis

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | **fail**(rc=1) | 落ちたのは `scripts/check-template-purity.sh` だけ(V1)。ほかの shellcheck、`sh -n`、guard の awk の構文、`jq -e`、Codex hook の検査 3 つ、check-sync、check-pipeline-sync、check-skill-sync、tech-debt の plan の参照、gofmt、golangci-lint(`0 issues.`)、branch の secret scan(`0351800d..0dfc49f0`、clean)は通った |
| `./scripts/check-template-purity.sh`(HEAD) | **fail**(rc=1) | `templates/base/.claude/skills/org/SKILL.md:1090` と `templates/base/.agents/skills/org/SKILL.md:1090` の `scripts/check-sync.sh` |
| `./scripts/check-template-purity.sh`(`git archive origin/main` の写し) | pass(rc=0) | このブランチで入った失敗 |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が一致 |
| `./scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| `git diff --check origin/main...HEAD` | pass(rc=0、無出力) | |
| `go build ./...` / `go vet ./internal/org/` / `gofmt -l internal/org` | pass | どれも無出力 |
| `./scripts/plan-visual.sh digest <plan>` | `ab78e5f0645b` | 計画の `- Approved:` の digest と同じ |

### V1(HIGH、merge を止める): 雛形の写しが meta-repo だけの `scripts/check-sync.sh` を名指す

`/org` skill の sync-docs の段落(`.claude/skills/org/SKILL.md:1089-1090`)は、doc-maintainer への指示に「写しの検査(`scripts/check-skill-sync.sh`・`scripts/check-sync.sh`)があれば打ち」と書く。この段落は `templates/base/` の 2 つの写しにも入り、`check-template-purity.sh` の `FIXED_PATTERNS`(`scripts/check-template-purity.sh:53`)が `check-sync.sh` を雛形の中で禁じている。`scripts/check-sync.sh` は `templates/base/scripts/` に入らず、scaffold した repo にはない。

- 「あれば打ち」なので、downstream で読んでも害はない。ただし purity の検査は決まった判定で、`ALLOWLIST_PATHS` は空のまま運用している(見つかったものは直してきた、と `scripts/check-template-purity.sh:21-25` に書いてある)
- CI の `verify.yml` は `./scripts/run-verify.sh` を打ち(`.github/workflows/verify.yml:39`)、そこから `scripts/verify.local.sh:255-256` が purity を打つので、このままでは PR の CI が落ちる。`tests/test-template-purity.sh` の A(実の `templates/` が通る)もおそらく落ちる(未確認。/test の範囲)
- 同じ文は雛形 `internal/org/prompts/leader.md:392` にもある。`internal/org/prompts/` は purity の対象ではないが、binary に埋め込まれて downstream の leader にも渡る。固定しているテストは `internal/org/prompts_leader_syncdocs_test.go:141` と `internal/org/skill_doc_followup_test.go:141`
- 見落とした理由: S2・S3・F1〜F3 の検証のコマンドは check-skill-sync・check-sync・gofmt・vet・go test で、run-static-verify も purity も入っていなかった
- 直し方の案: 雛形と `/org` skill の文を、ファイル名を挙げない形(例「repo にある写しの検査(`scripts/check-skill-sync.sh` など)」)にそろえ、テスト 2 本の固定を合わせる。allowlist に入れる道もあるが、雛形の側にも同じ理由が当てはまるので、文を直すほうを勧める。直したら `./scripts/check-template-purity.sh` が exit 0 になることを確かめる。コードと文書の直しなので、記録だけの例外には当たらず、パイプラインは /self-review から回し直しになる

## Documentation drift

| Doc / contract | In sync? | Notes |
| --- | --- | --- |
| 雛形 `leader.md` と `/org` skill(コマンド、パス、節の名前、決まったメッセージ、戻し方) | yes | 15 の語句を grep で数えて比べた(`Task(subagent_type="doc-maintainer")`、2 つの定義のパス、`docs: sync docs for <機能の slug>`、report のパス、`## 予約外の文書の follow-up`、`## 戻した変更`、`git restore --source=HEAD --staged --worktree -- <パス>`、`git log … \| awk …` の 1 行、`git show --name-only --format= <コミット>`、`SYNC_DOCS: skipped (<理由>)`、`DOC_FOLLOWUP:` の 3 つの形、`git diff HEAD -- <パス>`)。どれも両方にあり、文字列は同じ。手順の番号は 9・10・12・14 と 1・2・4・6 で対応している |
| 同上(`<機能の slug>` の決め方) | info | 雛形には「タスクの `- 機能:` の行の名前」(`leader.md:394-395`)があり、skill にはない。skill の読み手は分割計画の `### <slug>` から分かり、`- 機能:` は `f.Slug`(`internal/org/feature.go:420`)なので食い違いはない |
| `/director` skill と雛形 | yes | `DOC_FOLLOWUP:` の 3 つの形、`SYNC_DOCS: skipped (<理由>)`、節の名前が同じ |
| `/org` skill「受信箱」の RESULT の resolve(`.claude/skills/org/SKILL.md:473-475` と写し 3 つ) | LOW(V2) | 「PR の URL を `--note` に書いて resolve する」のまま。`/director` は note を PR の URL と follow-up の件数の 1 行にし、URL は件の本文から写さず merge-order の PR の列から取る、と決めた(`.claude/skills/director/SKILL.md:662-672`)。食い違うとまでは言えないが、`/org` だけを読むと本文の URL を写しうる。/sync-docs で 1 文足せば足りる |
| 仕様 FR-4 の注(`docs/specs/2026-10-07-org-multi-org-director.md:93`) | yes | 置き場所、凍らせたファイル、すべて戻すこと、skip の 3 つの条件、director の集め方(`merged` か `closed`、印、写しの組)、FR-4 の印を変えないことが、雛形と skill と合う |
| 仕様 FR-7 の 2026-10-26 の注(同 `:126`) | info | 手順 9〜11 の中身を「reviewer のレポートのコミット、fetch と rebase、org の report と計画の archive のコミット、secret scan」と並べ、sync-docs がない。日付の付いた決定の記録で、同じ注が古い注を「それぞれの時点の決定として残す」としているので、直さなくてよい |
| README の `ralph org gate` の段落(`README.md:280`) | yes | sync-docs、follow-up、merged or closed、凍らせたファイルが入った |
| AGENTS.md の map | yes | `gate.go` の項は leader の手順を並べない |
| `docs/quality/quality-gates.md:82`(と `templates/base/docs/quality/quality-gates.md:81`) | yes | 手順 9〜11 の中身を並べない |
| origin/main の取り込み(#290)との食い違い | なし | #290 が雛形(`leader.md:122-132` の `--state-dir` の段落)と 2 つの skill(`RALPH_ORG_STATE_DIR` → `ralph.local.toml` の `[org].state_dir`)を書き換えた。このブランチの新しい文に `RALPH_ORG_STATE_DIR` も `state_dir` も出てこない(テストの `--state-dir <台帳>` の 2 行は今の雛形の文)。新しい文と食い違う文は見つからなかった |
| `docs/tech-debt/README.md` の 2 行 | yes | R2-2 と `SYNC_DOCS:` の控え、`awk` と `core.quotePath` の行は、計画の Progress(`:137`)と合う |
| 計画の AC の checkbox | 未記入 | 指示どおり印は付けていない |

## Observational checks

- `/director` の新しい節が使う CLI の形をソースで確かめた: `ralph org inbox --all` は resolved を含み(`internal/cli/org_inbox.go:183`)、列は `ID STATE ORG TYPE TASK_ID …`(`:414`)。`ralph org merge-order --all` は merged と closed を出し(`internal/cli/org_merge_order.go:170`)、PR の列は `#<number> <url>`、ORG の列は `<org_id>/<feature>`(`:184-207`)。ブランチは `<Type>/<slug>`(`internal/org/feature.go:209`)、Type の既定は `feat`(`internal/org/split.go:75`、`:623-624`)。機能の節のフィールド行でない行は本文になる(`split.go:525-531`)ので、`元の分割計画: <id>` の行は分割計画として読める
- 雛形の手順を leader が実機で走らせること、Codex の leader が pane の中で custom agent を呼べることは確かめていない(計画のリスクと FR-4 の注のとおり)

## Coverage gaps

- テストの結果(AC7 の後半)と `tests/test-template-purity.sh` の A は /test が確かめる
- `awk` を gawk と mawk で、`git show --name-only` を `core.quotePath` と署名つきのコミットで確かめていない(tech-debt に記録済み)
- 足すと効く検査: `templates/` に触れるスライスの検証のコマンドに `./scripts/check-template-purity.sh` を入れる。新しい verifier は要らない。run-static-verify が今回それを捕まえた

## Verdict

pass

- cycle 2(HEAD e21d89c3)の判定。cross-review cycle 1 の ACTION_REQUIRED #1(P2)を直した F5・F5b・F6 のあと、sync-docs の段落の `git status` は雛形と `/org` skill の両方で 5 か所とも `git status --porcelain --untracked-files=all` になった。`run-static-verify.sh` と `check-template-purity.sh` は rc=0。所見は LOW 1 件(V3 仕様 FR-4 の注のコマンドが古い)で、merge を止めない。V3 は cycle 2 の /sync-docs が扱う。下の「cycle 2(HEAD e21d89c3)」の節を見る
- re-run(HEAD feeb1e80)の判定は pass だった。V1 と V2 は d8e73710(F4)で直り、`run-static-verify.sh` と `check-template-purity.sh` は rc=0 になった。開いている所見は INFO 2 件だけで、merge を止めない。下の「re-run(HEAD feeb1e80)」の節を見る
- 初回(HEAD 0dfc49f0)の判定は fail だった。AC1・AC2・AC3・AC4・AC5・AC6・AC8 の文は雛形と skill の該当行にあり、check-skill-sync、check-sync、`git diff --check`、`go build`、`go vet`、gofmt、golangci-lint、secret scan は通り、計画の digest は承認時と同じだった。所見は HIGH 1 件(V1 `check-template-purity.sh` の失敗。CI も落ちる)、LOW 1 件(V2 `/org` skill の受信箱の resolve の文)、INFO 2 件。insight event は `--verdict fail --cycle auto --high 1 --low 1`
- Partially verified: AC7(静的の部分だけ)
- Not verified: Go のテストの結果(/test)、leader と director が実機でこの手順を走らせること

## re-run(HEAD feeb1e80)

- Date: 2026-10-30
- Verifier: verifier サブエージェント(`/verify` の re-run。パイプラインの cycle は 1 のまま。`cycle-count.json` は `"cycle": 1`)
- Scope: 初回の report(ad820f13)のあとの 4 コミット、`git diff ad820f13..HEAD`(11 ファイル、+123/−22)。d8e73710(F4: V1・V2)、3247c0af(計画の Progress)、d4bb1a78(self-review run 3)、feeb1e80(R3-1 に合わせた台帳の 2 行)。origin/main は 0351800d のままで、HEAD はそれを含む(`git merge-base --is-ancestor` rc=0)。テストは打っていない(/test)
- Evidence: 同じ `docs/evidence/verify-2026-10-30-leader-sync-docs.log` の末尾に足した(コミットしない)。run-static-verify 自身のログは `docs/evidence/verify-2026-10-30-094746.log`

### 打ったもの

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | `All verifiers passed.`。ログに `FAIL` の行はない。golangci-lint `0 issues.`、secret scan は `0351800d..feeb1e80` で clean |
| `./scripts/check-template-purity.sh` | pass(rc=0) | `PASS: no meta-repo-specific references found in templates.` |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が一致 |
| `./scripts/check-sync.sh` | pass | IDENTICAL 172、DRIFTED 0、ROOT_ONLY 0 |
| `git diff --check origin/main...HEAD` | pass(rc=0、無出力) | |
| `go build ./...` / `go vet ./internal/org/` / `gofmt -l internal/org` | pass | どれも無出力 |
| `grep -rn 'check-sync.sh'`(`internal/org/prompts/` と `/org` skill の 4 つの写し) | 0 件(rc=1) | 雛形にも skill にも残っていない |
| `./scripts/plan-visual.sh digest <plan>` | `ab78e5f0645b` | 計画の Progress に 3 行足したあとも、`- Approved:` の digest と同じ |

### 初回の所見

| ID | 状態 | Evidence |
| --- | --- | --- |
| V1(HIGH) | 直った | `internal/org/prompts/leader.md:392` と `.claude/skills/org/SKILL.md:1093`(と写し 3 つ)が「写しの検査(`scripts/check-skill-sync.sh` など)があれば打ち」になった。名指すのは `templates/base/scripts/check-skill-sync.sh` として配られる検査だけ。2 つのテストは共通の定数 `syncDocsCopyChecks`(`internal/org/prompts_leader_syncdocs_test.go:63`)で同じ文を固定し、雛形の全文に `check-sync.sh` がないことを確かめる否定の検査(`:157-159`)を足した |
| V2(LOW) | 直った | `.claude/skills/org/SKILL.md:473-478`(と写し 3 つ)が、note を「merge-order --all の PR の列から取った URL(本文の URL は写さない)と follow-up の件数の 1 行、follow-up そのものは書かない」にし、`/director` skill の「件の処理」の RESULT の項を名指す。`/director` 側の `-` の列のときの扱いは `/director` にだけあり、`/org` はそこを指すので食い違わない。新しいテスト `TestOrgSkill_InboxResolvesAPRResultWithTheDirectorsNote`(`internal/org/skill_doc_followup_test.go:791-800`)が文を固定する |
| INFO 2 件 | そのまま | `<機能の slug>` の括弧書き(`leader.md:394-395`)と、仕様 FR-7 の 2026-10-26 の注(`:126`)。どちらも直さなくてよい |

### AC と文書のずれの見直し

- AC1〜AC6・AC8: met のまま。`leader.md` は 1 行を同じ場所で書き換えただけで、初回に挙げた行番号は変わらない。`/org` skill は受信箱の 3 行が増えたので、`:473` より後ろの行番号が 3 つずれた(sync-docs の段落 `:1050-1131`、写しの検査 `:1093`、PR の本文 `:1234`、RESULT の本文 `:1246`、座席内 fan-out `:1580`)。中身は変わっていない
- AC7: partly のまま。名指された 5 本のテストは今も diff に入っていない。変えたのは新しい 2 本だけ。テストが通るかは /test が確かめる
- 雛形と `/org` skill の写しの検査の文は、`repo に` の有無のほかは同じ(テストの共通の定数で両方を固定している)
- 台帳(feeb1e80、`docs/tech-debt/README.md:404-405`): 1 行目の「`HEAD:` と `BASE:` は `.harness/state/gate-<org_id>.out` から読み直せる」は雛形の手順 12(`leader.md:482-486`)と合う。2 行目の quoting の主張は、使い捨ての repo(git 2.49.0)で確かめた。`core.quotePath=true`(git の既定)では、日本語のパスは `git show --name-only` も `git status --porcelain` も `"docs/\346\227\245\346\234\254.md"` と引用し、空白のあるパスは porcelain だけが `"docs/a b.md"` と引用する。この環境の利用者の設定は `core.quotePath=false` なので、probe では `-c core.quotePath=true` を付けた
- 計画(3247c0af): Progress の 3 行は verify cycle 1 の fail、V1・V2、F4 の中身と合う。AC の checkbox は未記入のまま
- feeb1e80 は self-review run 3(d4bb1a78)のあとのコミットで、台帳の 2 行の文だけを変えた。run 3 が R3-1 で求めた直し方のとおりになっている
- 新しい所見はない

### re-run の判定

pass。insight event は同じ jsonl に 1 行足した(`--verdict pass --cycle auto`、findings は 0)。

## cycle 2(HEAD e21d89c3)

- Date: 2026-10-30
- Verifier: verifier サブエージェント(`/verify`、パイプラインの cycle 2。`cycle-count.json` は `"cycle": 2`)
- Scope: re-run の report(91e82463)のあとの 10 コミット、`git diff 91e82463..HEAD`(14 ファイル、+739/−145)。a5e28500(test の report)、e5f90219(sync-docs が `/org` skill に足した 2 か所)、da707a0d(cross-review cycle 1 の triage、ACTION_REQUIRED 1 件 P2)、F5 4d0f70e4、F5b aec1381f、ef4a15a7(計画の D2 と Risks、承認し直し)、8bd985fb(self-review cycle 2、LOW 2、Merge yes)、94c10923(台帳の行)、F6 41468881、e21d89c3(計画の記録)。origin/main は 0351800d のままで、HEAD はそれを含む(`git merge-base --is-ancestor` rc=0)。テストは打っていない(/test)
- Evidence: 同じ `docs/evidence/verify-2026-10-30-leader-sync-docs.log` の末尾に足した(コミットしない)。run-static-verify 自身のログは `docs/evidence/verify-2026-10-30-113457.log`

### 打ったもの

| Command | Result | Notes |
| --- | --- | --- |
| `./scripts/run-static-verify.sh`(changed scope) | pass(rc=0) | `All verifiers passed.`。ログに `FAIL` の行はない。golangci-lint `0 issues.`、secret scan は `0351800d..e21d89c3` で clean |
| `./scripts/check-template-purity.sh` | pass(rc=0) | `PASS: no meta-repo-specific references found in templates.` |
| `./scripts/check-skill-sync.sh` | pass | 14 skill が一致 |
| `./scripts/check-sync.sh` | pass | DRIFTED 0、ROOT_ONLY 0 |
| `cmp`(`.claude/skills/org/SKILL.md` と写し 3 つ) | 同じ | `.agents/`、`templates/base/.claude/`、`templates/base/.agents/` |
| `git diff --check origin/main...HEAD` | pass(rc=0、無出力) | |
| `go build ./...` / `go vet ./internal/org/` / `gofmt -l internal/org` | pass | どれも無出力 |
| `grep -rn 'check-sync.sh'`(`internal/org/prompts/` と `templates/base/` の skill) | 0 件(rc=1) | |
| `./scripts/plan-visual.sh digest <plan>` | `220fa1f3994f` | 計画の `- Approved:` の digest と同じ(ef4a15a7 で付け直したもの) |

### sync-docs の段落の `git status` の形

雛形の手順 9(`internal/org/prompts/leader.md:348-447`)と `/org` skill の「org の終わり方」の 1(`.claude/skills/org/SKILL.md:1050-1150`)で、`git status` を名指す箇所はどちらも 5 つ。5 つとも `git status --porcelain --untracked-files=all` で、`--untracked-files=all` のない `git status --porcelain` は 0 か所だった。

| 場所 | 雛形 | `/org` skill |
| --- | --- | --- |
| 失敗の枝: 戻すパスの一覧 | `leader.md:359` | `SKILL.md:1061` |
| 失敗の枝: 戻したあと、何も出ないことの確かめ | `:365` | `:1067` |
| 呼ぶ前の確かめ(F5b) | `:369` | `:1071` |
| 終わったあとの照らし合わせ | `:411` | `:1113` |
| コミットのあとの確かめ | `:439` | `:1142` |

- 段落の行を連結して空白を除き、9 つの文(上の 5 つのコマンドを含む文、未追跡のファイルを「一覧に出たファイルを 1 つずつ消す(ディレクトリごとは消さない)」、呼ぶ前の確かめの理由、振り分けの「(per-plan では `PLAN:` のヘッダを付ける)」、`--untracked-files=all` を付ける理由)を雛形と skill で探した。どれも両方に 1 回ずつあった。呼ぶ前の確かめの振り分けの終わりは、雛形が手順 8 の往復と BLOCKED の上げ方を書き、skill が「reviewer にレビューを頼み直し、それ以外なら `BLOCKED` を escalate する」と縮めている。手順 9 の最初の段落の雛形と skill の書き分け(`SKILL.md:1031-1048`)と同じ縮め方で、食い違いはない
- 手順 9 の最初の段落の確かめ(`leader.md:325`、`SKILL.md:1032`)は `--untracked-files=all` のないまま。`prompts_test.go` が固定していて変えないこと、その穴を呼ぶ前の確かめが埋めることは、計画の F5b の記録(Progress)に書いてある
- `ralph org gate` の汚れた作業ツリーの判定も `git status --porcelain --untracked-files=all`(`SKILL.md:255` の gate の項)なので、leader の確かめは gate と同じ見方になった
- probe(使い捨ての repo、git 2.49.0): git の既定(`status.showUntrackedFiles=normal`)の `--porcelain` は、新しいディレクトリの 2 つのファイルを `?? docs/` の 1 行で出し、`--untracked-files=all` は 1 ファイルずつ出した。`status.showUntrackedFiles=no` では、`--porcelain` は何も出さず、`--untracked-files=all` は 1 ファイルずつ出した。ignore したディレクトリのファイルはどちらにも出ない。1 つずつ消したあとに残る空のディレクトリは、`--untracked-files=all` にも出ない。この環境の利用者の git の設定は `status.showUntrackedFiles=all`(`~/.config/git/config`)なので、probe では `-c status.showUntrackedFiles=normal` を付けた

### AC と文書のずれの見直し

- AC1〜AC6・AC8: met のまま。F5・F5b で文が増えたので行番号がずれた。雛形では sync-docs の段落が `:348-447`、doc-maintainer への指示が `:391-409`、写しの組が `:397-401`、一部だけ戻さないことが `:429-431`、1 コミットが `:435-442`、やり直しが `:444-447`、手順 10 が `:449`、手順 12 が `:470`、`GATE: fail` の追記が `:538-543`、PR の本文が `:576-584`、RESULT の例が `:607-608`。skill では sync-docs の段落が `:1050-1150`、`GATE: fail` が `:1214`、PR の本文が `:1253`、座席内 fan-out が `:1599`、完了条件が `:1945`。中身は変わっていない。AC2・AC8 の「変わったパスの確かめ」はファイルごとに照らし合わせるようになり、P2 の穴(新しいディレクトリの中の予約したファイルを外と読む、予約の外の兄弟のファイルが見えない)は塞がった
- AC7: partly のまま。名指された 5 本のテストは今も diff に入っていない。テストが通るかは /test が確かめる
- e5f90219 が `/org` skill に足した 2 か所(`SKILL.md:1772-1776` の「Leader 運用 2 経路」の 7、`:1924` の締めの順)は、雛形の手順 9〜14 の順と合う
- 計画(ef4a15a7、e21d89c3): D2 と Risks は `--untracked-files=all` と呼ぶ前の確かめになり、雛形と合う。Progress の F5・F5b・F6 の記録は diff と合う。AC の checkbox は未記入のまま
- 台帳(94c10923、`docs/tech-debt/README.md:405`): 照らし合わせる側を `git status --porcelain --untracked-files=all`、払い方を `-z` 付きの同じコマンドと書き、雛形と合う
- `/director` skill と README `:280` は `git status` を名指さないので、変える必要はない

### V3(LOW): 仕様 FR-4 の注のコマンドが F5 の前のまま

`docs/specs/2026-10-07-org-multi-org-director.md:93`(2026-10-30 に決めたこと)は「leader は `git status --porcelain` で変わったパスを読み」と書く。雛形と `/org` skill と計画の D2 は `git status --porcelain --untracked-files=all` になり、呼ぶ前の確かめも足された。注は要約なので呼ぶ前の確かめまで書く必要はないが、名指したコマンドは今の形に合わせるのがよい。cycle 2 の /sync-docs で 1 か所直せば足りる。

### cycle 2 の判定

pass。所見は LOW 1 件(V3)と、前からの INFO 2 件。insight event は同じ jsonl に 1 行足した(`--verdict pass --cycle auto --low 1`、cycle 2)。
