# drop-org-shell-defaults

- Status: Approved
- Approved: 2026-10-28 sha256:2ec5763f7abf
- Owner: Claude Code
- Date: 2026-10-28
- Related request: https://github.com/yoshpy-dev/ralph/issues/273 (scripts/ralph-config.sh の RALPH_ORG_* の写しを消す)
- Related issue: #273
- Type: refactor
- Branch: refactor/drop-org-shell-defaults

## Objective

`scripts/ralph-config.sh` から、どの `ralph org` の動詞も読まない `RALPH_ORG_*` の既定値の写しを消す。
既定値の一致は、`internal/config/config.go` の `Default()` と `templates/base/ralph.toml` を直接照らし合わせるテストで守る。
2026-10-27 に決めた「設定は `ralph.toml` に置き、環境変数を使わない」方針の 1 段目に当たる。

## Scope

- `scripts/ralph-config.sh` と、その byte 一致の写し `templates/base/scripts/ralph-config.sh` から、`RALPH_ORG_AGMSG_HOME` を除く `RALPH_ORG_*` の 18 行と、それらの `export` を消す。ファイル冒頭とブロックのコメントを、残る中身に合わせて書き直す。
- `internal/config/defaults_sync_test.go` を組み直す。
  - `[org]` の既定値は `Default()` と `templates/base/ralph.toml` の 2 面で比べる。テンプレートは `Load` を通さず、ポインタの field を持つ probe struct に読んで、キーが書かれているかも確かめる(`Load` は書かれていないキーに `Default()` の値を残すので、キーが消えても一致してしまう)。
  - `config_test.go` の `TestLoad_TemplateRalphToml` にある `auto_merge` と `wake` の明示の文字列確認は、新しい照合が全キーの有無と値を見るので畳む(`wake_cmd` が空であることの確認は残す)。
  - `org.agmsg_home` だけは、`scripts/ralph-config.sh` の `RALPH_ORG_AGMSG_HOME` とも比べ続ける(#276 で扱う)。
  - `/plan` と `/cross-review` の SKILL.md の fallback と、`scripts/ralph-config.sh` の標準フローの既定値の比較は今のまま残す。
  - 比較を「テンプレートのバイト列と `Default()` を受けて、ずれたキーの一覧を返す」関数に切り出し、値を変えたテンプレートとキーを消したテンプレートで落ちることを単体テストで固定する。
- 写しに触れているコメントと文書を直す。`internal/config/config.go` の `OrgConfig` のコメント、`tests/test-ralph-config.sh` の冒頭コメント、`docs/specs/2026-08-01-org-runtime.md` の運用注記 (a) と FR-2、`docs/tech-debt/README.md` の model_pool 表の行、`.claude/rules/ralph/model-routing.md`(root のみ)の「Where the values live」。

## Non-goals

- `RALPH_ORG_AGMSG_HOME` の扱いを変えること(#276)。
- 標準フローの変数(`RALPH_STANDARD_MAX_PIPELINE_CYCLES`、`RALPH_CLAUDE_REVIEWER_MODEL`、`RALPH_CODEX_REVIEWER_MODEL`、`RALPH_CODEX_REASONING_EFFORT`)を `ralph.toml` へ移すこと(#276)。
- 既定値そのものを変えること(`codex_verified` と `auto_merge` は #274)。
- `[doctor]` の既定値を照合に足すこと(#272 が `require_*` を変えるので、そちらと衝突させない)。
- `/org` skill の「既定の model_pool」表をテストで縛ること(tech-debt の既存の行のまま)。
- 履歴の文書(`docs/specs/2026-08-01-org-runtime.md` の Research findings、`docs/specs/2026-10-07-org-multi-org-director.md`、`docs/evidence/`、`docs/reports/`、`docs/plans/archive/`)の書き換え。

## Assumptions

- `RALPH_ORG_AGMSG_HOME` 以外の `RALPH_ORG_*`(`STATE_DIR` を除く)は、Go のコードもスクリプトも環境から読まない。確認: `git grep -nE 'RALPH_ORG_(DRIVER_POOL|MODEL_POOL|MAX_SEATS|MAX_ORGS|MAX_TOTAL_SEATS|DEADMAN_MINUTES|PERMISSION_DEFAULT|PERMISSIONS_CODEX_VERIFIED|WATCHDOG_|DIRECTOR_|APPROVAL_MODE)'` の当たりは、`scripts/ralph-config.sh` とその写し、`internal/config/*_test.go`、文書だけ。export されている 6 つ(`DRIVER_POOL` など)を読む消費者もない。
- `RALPH_ORG_STATE_DIR` は Go が読む別の変数で、`ralph-config.sh` には書かれていない。この変更では触らない。
- `templates/base/ralph.toml` は、照合する 19 のキーをすべて明示している(`[org]` 7、`[org.permissions]` 2、`[org.watchdog]` 4、`[org.director]` 5、`[org.approval]` 1。`agmsg_home` を含む)。
- `scripts/ralph-config.sh` は core のファイルなので、`ralph upgrade` が下流の写しを置き換える。下流で `RALPH_ORG_*` を shell から読んでいたものがあっても、もともと `ralph org` には効いていなかった。

## Affected areas

- `scripts/ralph-config.sh`、`templates/base/scripts/ralph-config.sh`
- `internal/config/defaults_sync_test.go`、`internal/config/config_test.go`(`TestLoad_TemplateRalphToml` の重なる確認)、`internal/config/config.go`(コメント)
- `tests/test-ralph-config.sh`(コメント)
- `docs/specs/2026-08-01-org-runtime.md`、`docs/tech-debt/README.md`、`.claude/rules/ralph/model-routing.md`

## Visual review

None (テストの照合先を 3 面から 2 面に減らし、読まれない shell の行を消す変更で、図にする仕組みの変化がない)

## Design decisions

- D1: テンプレートは `Load` を通さず、19 キーをポインタの field で持つ probe struct に `toml.Unmarshal` で 1 回読む。nil ならキーがなく、nil でなければ値を `Default()` と比べる。`Load` は `Default()` から始めるので、テンプレートからキーが消えても `Default()` の値が残り、照合が素通りする。`codex_verified = false` や `auto_merge = false` のように既定がゼロ値のキーも、ポインタなら有無が分かる。この形は `config.go` の `orgPoolKeysPresent` がすでに使っている慣用句で、go-toml の `map[string]any` の入れ子の扱いに依らない(consult の指摘で `map[string]any` の二重読みから改めた)。今は `config_test.go` が `auto_merge` と `wake` についてだけ文字列で明示を確かめているが、全キーの照合に含まれるので畳む。
- D2: `RALPH_ORG_AGMSG_HOME` の行と、その 3 面の照合は残す。issue が #276 に回すと決めている。
- D3: 照合の入口は `TestDefaultsLockStep` の名前のまま、`t.Run` で「org」「shell の既定値と skill の fallback」を分ける。今は `scripts/ralph-config.sh` がないと org の照合まで飛ばされる。分ければ、org の照合はテンプレートだけを前提にできる。
- Critical forks: None

## Acceptance criteria

- [x] AC1: `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` に、`RALPH_ORG_AGMSG_HOME` を除く `RALPH_ORG_*` の代入と export がない(`grep -n 'RALPH_ORG_' scripts/ralph-config.sh | grep -v RALPH_ORG_AGMSG_HOME` が何も出さない)。2 つのファイルは byte 一致している。
- [x] AC2: `Default()` と `templates/base/ralph.toml` の `[org]` の既定値が 1 つでもずれると、`go test ./internal/config/` が落ちる。値を変えたテンプレートと、キーを消したテンプレートの両方で落ちることを、比較関数の単体テストで固定している。
- [x] AC3: `org.agmsg_home` の 3 面の照合と、SKILL.md の fallback の照合が残っている。
- [x] AC4: `scripts/check-sync.sh`、`tests/test-ralph-config.sh`、`go test ./internal/config/` が通る。
- [x] AC5: 写しの存在を前提にしたコメントと文書(Scope に挙げたもの)が、新しい形に合っている。履歴の文書を除き、`git grep` で消した変数名が残っていない。spec の FR-2 は 2 面の照合を述べている(変数名ではなくファイル名の言及なので、grep とは別に読んで確かめる)。

## Implementation outline

1. 照合のテストを組み直す: `internal/config/defaults_sync_test.go` に、テンプレートのバイト列と `Default()` を受けてずれの一覧を返す比較関数を足し、`[org]` の 19 キーを値とキーの有無で比べる。`TestDefaultsLockStep` を `t.Run` で org と shell/skill に分け、org の比較から shell を外す(`agmsg_home` の shell との比較は残す)。比較関数の単体テストに、実際のテンプレートで差がないこと、値を変えると落ちること、キーを消すと落ちることを入れる。`config_test.go` の `TestLoad_TemplateRalphToml` から、`RALPH_ORG_DIRECTOR_*` に触れるコメントと、`auto_merge` と `wake` の文字列での明示確認を外す(`wake_cmd` の確認は残す)。 — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
2. shell の写しを消す: `scripts/ralph-config.sh` から `RALPH_ORG_AGMSG_HOME` 以外の `RALPH_ORG_*` の代入とコメント、2 行の `export RALPH_ORG_*` を消し、冒頭のコメントと `[org]` ブロックのコメントを残る 1 行に合わせて書き直す。`templates/base/scripts/ralph-config.sh` に同じ内容を写す。`tests/test-ralph-config.sh` の冒頭コメントを直す。 — verify: `sh -n scripts/ralph-config.sh && cmp scripts/ralph-config.sh templates/base/scripts/ralph-config.sh && sh tests/test-ralph-config.sh && ./scripts/check-sync.sh && GOFLAGS="-timeout=30m $(go env GOFLAGS)" go test ./internal/config/`
3. 文書とコメントを合わせる: `internal/config/config.go` の `OrgConfig` のコメント、`docs/specs/2026-08-01-org-runtime.md` の運用注記 (a)(45 行)と FR-2(74 行、同 spec の既存の改訂の印と同じ形で「**(2026-10-28 #273 で `scripts/ralph-config.sh` を外し 2 面に)**」と書き添える)、`docs/tech-debt/README.md` の model_pool 表の行、`.claude/rules/ralph/model-routing.md` の「Where the values live」の `defaults_sync_test.go` の行を、2 面の照合に合わせる。Research findings(157 行)は履歴なので書き換えない。 — verify: `test -z "$(gofmt -l internal/config)" && go vet ./internal/config/ && ./scripts/check-sync.sh && ! git grep -nE 'RALPH_ORG_(DRIVER_POOL|MODEL_POOL|MAX_SEATS|MAX_ORGS|MAX_TOTAL_SEATS|DEADMAN_MINUTES|PERMISSION_DEFAULT|PERMISSIONS_CODEX_VERIFIED|WATCHDOG_|DIRECTOR_|APPROVAL_MODE)' -- ':!docs/evidence' ':!docs/reports' ':!docs/plans' ':!docs/insights'`

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt、go vet、golangci-lint、shellcheck、check-sync、check-skill-sync を含む)。
- Spec compliance criteria to confirm: AC1〜AC5。issue の受け入れ条件 3 つ(写しがない、ずれると落ちる、check-sync と既存のテストが通る)との対応。
- Documentation drift to check: Scope に挙げた文書と、`AGENTS.md` の地図(`ralph-config.sh` の説明が変わらないこと)。`.claude/rules/ralph/model-routing.md` は root と templates が KNOWN_DIFFS なので、root だけを直す。
- Evidence to capture: AC1 の grep、AC5 の `git grep` の結果、`cmp` の結果。

## Test plan

- Unit tests: 比較関数の単体テスト(実テンプレートで差なし、`max_seats` の値を変えると `org.max_seats` を名指しして落ちる、`auto_merge` の行を消すと「キーがない」で落ちる、`model_pool` の並びを変えると落ちる)。
- Integration tests: `go test ./internal/config/`(`TestDefaultsLockStep`、`TestLoad_TemplateRalphToml` ほか既存)。`sh tests/test-ralph-config.sh`。
- Regression tests: `./scripts/run-test.sh`(全体)。`TestLoad_TemplateRalphToml` から畳む `auto_merge` と `wake` の確認は、新しい照合の「キーを消すと落ちる」単体テストが置き換える(弱めではない)。`ralph-config.sh` は言語に分類されないので、全体の実行になる。
- Edge cases: `scripts/ralph-config.sh` がない環境(vendored)で、org の照合は走り、shell の照合だけ飛ぶ。`.claude/skills` がないときは skill の照合だけ飛ぶ。
- Evidence to capture: 比較関数のテストの出力(`-run` で絞った `-v`)、変異を入れたときに落ちることの確認。

## Risks and mitigations

- R1: 下流の project の shell スクリプトが `. scripts/ralph-config.sh` のあとに `RALPH_ORG_MAX_SEATS` などを読んでいた場合、空になる。軽減: もともと `ralph org` には効かない値で、repo の中に読み手はない。PR の本文の release ノートの項に 1 行書く。
- R2: 照合を 2 面にしたことで、`templates/base/ralph.toml` からキーが消えても気づけない状態に戻る。軽減: D1 でキーの有無を全キーで確かめ、その失敗を単体テストで固定する。
- R3: #274(同じ列の次の issue)が `codex_verified` と `auto_merge` の既定を変える。軽減: この PR のあとに入る順なので、#274 は Go と toml の 2 面を直すだけで済む。
- R4: 別のセッションの #272 が `[doctor]` の既定を変える。軽減: `[doctor]` は照合に入れない(Non-goals)。

## Rollout or rollback notes

- 1 つの PR で入れる。戻すときは revert すれば、写しと 3 面の照合が戻る。
- release のノートに「`scripts/ralph-config.sh` から `RALPH_ORG_*` の既定値の写しを消した。`ralph org` はもともと読んでいない。`RALPH_ORG_AGMSG_HOME` は残る」と書く(PR の本文に文を置く)。

## Open questions

- なし。

## Progress checklist

- [x] Plan reviewed (consult-plan-orgshell: 直してから進める → FR-2 を Scope に足し、D1 を probe struct に改めた。Codex plan advisory: no findings)
- [x] Plan approved — 自動承認(ユーザーの指示)。digest は Approved の行
- [x] Branch created (refactor/drop-org-shell-defaults, worktree .claude/worktrees/drop-org-shell-defaults)
- [x] Implementation started
  - S1 e948b85d(impl-273-s1): `orgTemplateDrift` と probe struct、`TestDefaultsLockStep` を `org` と `shell_and_skills` に分割、変異の単体テスト 6 件(handoff にない「wake の行を消す」を足した)。`TestLoad_TemplateRalphToml` の auto_merge と wake の確認を畳んだ(PR の本文に書く)。テンプレートの変異(max_orgs 10→11)で `TestDefaultsLockStep/org` が落ちることを確認。
  - S2 76c7595d(impl-273-s2): 18 変数と export の 3 行を削除、テンプレートの写しと byte 一致。S2 のあと `go test -run TestDefaultsLockStep -v` で `shell_and_skills` が `RALPH_ORG_AGMSG_HOME` の 1 行で通ることを orchestrator が確認。
  - S3 c56fa63c(impl-273-s3): config.go のコメント、defaults_sync_test.go:64 のコメント(S2 が見つけた範囲外のずれを含めた)、spec の注記 (a) と FR-2 の印、tech-debt:149、model-routing.md。消した変数名の git grep は履歴の外で 0 件。
  - 各スライスの検証コマンドはすべて通った(各 implementer の報告)。
- [x] Review artifact created — self-review 5de4cc6a(LOW 5 件、Merge: yes)。L1・L2・L3・L5 を修正スライス 7d59e278(impl-273-f1)で直し、再実行 6e02c5e1 で LOW 1 件(L4、Merge: yes)。L4(19 キーの数がコメントにしかない、組み直し前からの穴)は PR の Known gaps に書く。
  - Deviation(D2・AC3): `org.agmsg_home` の 3 面の照合は、直接の 3 者比較から「`shell_and_skills` の shell 対 `Default()`」と「`org` の テンプレート対 `Default()`」の合成に変わった(7d59e278、L3)。どの 1 面がずれても、どちらかのサブテストが面の名前つきで落ちる。テンプレートがないときも SKILL.md の照合が走るようになった(L2)。
  - Deviation(L1): `scripts/ralph-config.sh` のコメントに issue の URL を書かなかった。`scripts/check-template-purity.sh` が `templates/` の中で `yoshpy-dev` を禁じており、写しが下流に配られるため。
  - 2026-10-28 08:30Z ごろ、同じ Mac の stoic-app のセッションから「ローカル CI を約 90 分流すので重い処理の開始を控えて」と依頼があった。修正スライスは `internal/config` の 1 パッケージだけで検証した。/verify と /test は再開の連絡を待ってから回す。
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
