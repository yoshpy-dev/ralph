# sync-docs report: drop-org-shell-defaults

- Date: 2026-10-28
- Plan: `docs/plans/active/2026-10-28-drop-org-shell-defaults.md`(承認の digest `2ec5763f7abf` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の 2 項目だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `15e301c8`、branch は `refactor/drop-org-shell-defaults`。run 開始時の HEAD は `339e465f`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-28-drop-org-shell-defaults.md`(5de4cc6a、再 run は 6e02c5e1、LOW 1 件、Merge yes)、`docs/reports/verify-2026-10-28-drop-org-shell-defaults.md`(ecf31b75、pass、情報 D-1〜D-4)、`docs/reports/test-2026-10-28-drop-org-shell-defaults.md`(339e465f、pass)
- 差分の大きさ(`git diff 15e301c8...HEAD --shortstat`、この run の前): 14 ファイル、+725/-315。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` から、`RALPH_ORG_AGMSG_HOME` を除く `RALPH_ORG_*` の 18 変数と 3 行の `export` を消した。`[org]` の既定値の一致は `internal/config/defaults_sync_test.go` が `config.Default()` と `templates/base/ralph.toml` を直接比べて守る(`orgTemplateDrift`、19 キー)。

verify が残した情報 D-1 と D-2 を直した。D-1 は `tests/test-ralph-config.sh` の冒頭コメントが `ralph-config.sh` に残る `RALPH_ORG_AGMSG_HOME` を挙げていなかった点で、D-2 は「各 `[org]` キーを検査する」と読める 2 つの文(`model-routing.md` と `config_test.go`)である。照合が実際に読むのは 19 キーで、`[org.roles]`、`[org.permissions].roles`、`[org.director].launch_cmd`、`wake_cmd` は読まない。どちらも文言の直しで、振る舞いは変えていない。

そのほかの面を洗った結果、直す文書は見つからなかった。消した 18 変数の名前は、履歴の文書(`docs/reports/`、`docs/evidence/`、`docs/plans/archive/`、`docs/insights/`)とこの計画のほかに、リポジトリに残っていない。

## Changes made

| File | Change |
|------|--------|
| `tests/test-ralph-config.sh` | 冒頭コメントの「surviving surface」の列挙に、`ralph` バイナリが環境から読む `[org]` の既定値 `RALPH_ORG_AGMSG_HOME`(export しない)を足した。コメントだけ(D-1) |
| `.claude/rules/ralph/model-routing.md`(root のみ) | 「Where the values live」の `defaults_sync_test.go` の行を、`orgTemplateDrift` が読む `[org]` のキーについて「each key set, same value」と書き、`[org.roles]`、`[org.permissions].roles`、`launch_cmd`、`wake_cmd` は対象外と明記した。`templates/base/` の写しはこの行を持たない(`check-sync.sh` の KNOWN_DIFF)ので触っていない(D-2) |
| `internal/config/config_test.go` | `TestLoad_TemplateRalphToml` のコメントを、`TestDefaultsLockStep` が `orgTemplateDrift` の読むキーを `Default()` と比べること、`[org.roles]`、`[org.permissions].roles`、`launch_cmd` は読まないこと、`wake_cmd` の空はこの関数の下で確かめることに直した。コメントだけで、`gofmt -l internal/config` は出力なし(D-2) |
| `docs/plans/active/2026-10-28-drop-org-shell-defaults.md` | `## Progress checklist` の「Verification artifact created」と「Test artifact created」を `[x]` にし、report の SHA と verdict、D-1〜D-4 の扱いを書いた。ほかの節は触れていない(digest は同じ) |
| `docs/reports/sync-docs-2026-10-28-drop-org-shell-defaults.md` | この report |

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| 消した 18 変数の名前の `git grep`(`docs/reports`、`docs/insights`、この計画を除く) | 履歴だけ | 当たりは `docs/evidence/test-2026-10-14-org-director-attach.md` と `docs/plans/archive/` の 6 本で、どれも当時の記録。計画の Non-goals が履歴の書き換えを除いている |
| `docs/specs/2026-08-01-org-runtime.md` の注記 (a)(`:45`)、FR-2(`:74`) | 変更なし(同期済み) | 2 面と、値の一致とキーの記述を検査する、と書く。FR-2 は既存の改訂の印と同じ形で #273 の印がある |
| 同 spec の注記 (c)(`:47`) | 変更なし | `ralph upgrade` が core(skill・`ralph-config.sh`)を置き換える、という文は正しい。`ralph-config.sh` が `model_pool` を持たなくなったので例としては不要だが、誤りではない(D-3)。依頼どおり残した |
| 同 spec の `:135`(Impact)、`:157`(Codebase analysis) | 変更なし | 「3 面ロックステップ」は 2026-08-01 時点の調査記録。計画の Non-goals が履歴として残すと決めた |
| `docs/specs/2026-10-07-org-multi-org-director.md`(`:213`、`:237`) | 変更なし | 履歴の spec。lock-step を求める FR はない |
| `docs/tech-debt/README.md:149`(model_pool 表の行) | 変更なし(同期済み) | 「two value surfaces (`Default()`, `templates/base/ralph.toml`)」。依頼どおり編集していない |
| `docs/tech-debt/README.md:51`(取り消し線の行) | 変更なし | `scripts/ralph-config.sh:20` に触れるが、解決済みの行で当時の記録 |
| `internal/config/config.go` の `OrgConfig` のコメント | 変更なし(同期済み) | `AgmsgHome` だけ `RALPH_ORG_AGMSG_HOME` とも比べる、と書く |
| `internal/config/defaults_sync_test.go` の冒頭コメント | 変更なし(同期済み) | 19 キー、`org` と `shell_and_skills` の 2 つのサブテスト、`agmsg_home` の例外が実装と合う |
| `internal/org/driver/agmsg.go:134-137`、`internal/org/prompts_test.go:258-259` | 変更なし | 前者は `agmsg_home` の 3 面(`Default()`、テンプレート、`ralph-config.sh`)で、この branch のあとも 3 面のまま。後者は「default model_pool は `defaults_sync_test.go` が別に固定する」で、いまも正しい |
| `scripts/ralph-config.sh` と `templates/base/scripts/ralph-config.sh` のコメント | 変更なし(同期済み) | 残るのは `RALPH_ORG_AGMSG_HOME` だけで、ほかの `[org]` の既定値は `config.Default()` と scaffold される `ralph.toml` にある、と書く。`cmp` で byte 一致(`check-sync.sh` も IDENTICAL) |
| `AGENTS.md:111`、`docs/architecture/repo-map.md:62` | 変更なし | `ralph-config.sh` を「標準フローの共有設定」として挙げるだけで、`[org]` の既定値には触れない |
| `README.md`、`docs/quality/*.md` | 変更なし | `ralph-config.sh` と消した変数への言及がない。`[org]` の記述は `ralph.toml` の設定と動詞の説明で、shell の既定値を前提にしていない |
| `docs/recipes/`(`director-launch-cmd.md`、`isolated-herdr-server.md`、`codex-seat-permissions.md`) | 変更なし | `RALPH_ORG_AGMSG_HOME` と `RALPH_ORG_STATE_DIR` の話で、どちらもこの branch のあとも有効。`codex-setup.md:96` は標準フローの codex 変数の話 |
| `.claude/skills/`、`.agents/skills/`、`templates/base/` の skill の写し | 変更なし | skill は触れていない。`check-skill-sync.sh` は 14 skill が一致 |
| `/org` skill の「既定の model_pool」表 | 変更なし | テストで縛られていない、という tech-debt の既存の行のまま(計画の Non-goals) |

## Plan drift notes

- Deviation(D2・AC3): `org.agmsg_home` の 3 面の照合が、直接の 3 者比較から 2 つのサブテストの合成に変わった。計画の Progress に記録済み(7d59e278)。
- Deviation(L1): `scripts/ralph-config.sh` のコメントに issue の URL を書かなかった(`check-template-purity.sh` が `templates/` の中の `yoshpy-dev` を禁じる)。Progress に記録済み。
- 計画の Implementation outline の step 2 は「2 行の `export RALPH_ORG_*`」と書くが、消したのは 3 行である(D-4)。計画の本文は digest の対象なので直さず、Progress の S2 の行(「3 行を削除」)を記録として残した。
- 計画の Acceptance criteria は AC1〜AC5 の `[x]` のまま変わらない。AC4 の `[x]` は test report の 2 回目の `run-test.sh` と `tests/test-ralph-config.sh` 19/19 が根拠になった。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。

## `/pr` と release に渡すこと

- 計画の Rollout のとおり、PR の本文に release ノートの文を置く。「`scripts/ralph-config.sh` から `RALPH_ORG_*` の既定値の写しを消した。`ralph org` はもともと読んでいない。`RALPH_ORG_AGMSG_HOME` は残る」。下流の shell が `ralph-config.sh` を source して `RALPH_ORG_MAX_SEATS` などを読んでいた場合は空になる(計画の R1)。
- `.goreleaser.yml` の changelog は `^docs:`、`^test:`、`^chore:` を除くので、この branch の `refactor: drop the unread RALPH_ORG_* default copies from ralph-config.sh` と `fix: address the drop-org-shell-defaults self-review findings` の 2 件が自動の changelog に載る。後者の題は内容を言わないので、release ノートの文を手で足す。
- `TestLoad_TemplateRalphToml` から `auto_merge` と `wake` の文字列での確認を畳んだこと(新しい照合が置き換える)と、self-review L4(19 という数がコメントにしかなく、`OrgConfig` に 20 個目の既定値を足したときにどのテストも気づかない、組み直し前からの穴)は、PR の本文に書く(Progress に記録済み)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | rc 0。`[ok] check-skill-sync: 14 skill(s) in lock-step` |
| `./scripts/check-template-purity.sh` | rc 0。`PASS: no meta-repo-specific references found in templates.` |
| `gofmt -l internal/config` | 出力なし |
| `sh -n tests/test-ralph-config.sh` | rc 0 |
| `git diff --check` | 出力なし |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-10-28-drop-org-shell-defaults.md` | `2ec5763f7abf`(`- Approved:` 行の値と同じ。Progress の編集の前後で同じ) |
| テスト、`go vet`、golangci-lint、`run-verify.sh` ほか | 打っていない。別のセッションが長い CI を回しているため、依頼で止められた。この run が変えたのはコメント 3 か所、rule の 1 文、計画の Progress、この report で、コードの動きは変わらない。コメントの直しのあとの `go test ./internal/config/` は未確認 |
| insight event | `./scripts/insights-append.sh --slug drop-org-shell-defaults --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記する(`docs/insights/events/2026-10-28-drop-org-shell-defaults.jsonl`) |
