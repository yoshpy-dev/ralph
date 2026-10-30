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

fail

- Verified: AC1・AC2・AC3・AC4・AC5・AC6・AC8 の文は雛形と skill の該当行にある。check-skill-sync、check-sync、`git diff --check`、`go build`、`go vet`、gofmt、golangci-lint、secret scan は通り、計画の digest は承認時と同じ
- Partially verified: AC7(静的の部分だけ)
- Not verified: Go のテストの結果(/test)、leader と director が実機でこの手順を走らせること
- 所見: HIGH 1 件(V1 `check-template-purity.sh` の失敗。CI も落ちる)、LOW 1 件(V2 `/org` skill の受信箱の resolve の文)、INFO 2 件。V1 を直してから /self-review から回し直す
- Insight event: `docs/insights/events/2026-10-30-leader-sync-docs.jsonl` に verify の行を 1 行足した(`--verdict fail --cycle auto --high 1 --low 1`)
