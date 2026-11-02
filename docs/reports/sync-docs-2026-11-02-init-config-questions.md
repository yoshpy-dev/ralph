# sync-docs report: init-config-questions

- Date: 2026-11-02
- Plan: `docs/plans/active/2026-11-02-init-config-questions.md`(issue #277、承認の digest `d1e86274a18e` はこの run の前後で同じ。`./scripts/plan-visual.sh digest` で確かめた。この run が変えたのは `## Progress checklist` の 2 項目だけ)
- Pipeline cycle: 1(`.harness/state/standard-pipeline/cycle-count.json` の値)。base は `c8c8c0f1`、branch は `feat/init-config-questions`。run 開始時の HEAD は `8a3fef5c`(この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-11-02-init-config-questions.md`(Run 1 が LOW 5 件、S4 1af26382 で 3 件を直し、Run 2 の 61caf1d3 は所見 0 で Merge yes)、`docs/reports/verify-2026-11-02-init-config-questions.md`(2fb3284d、pass、LOW V1 が 1 件、情報 2 件)、`docs/reports/test-2026-11-02-init-config-questions.md`(8a3fef5c、pass)
- 差分の大きさ(`git diff origin/main...HEAD --shortstat`、この run の前): 11 ファイル、+2419/-21。walkthrough を書くかどうかは `/pr` が決める

## Summary

この branch は issue #277 の対応で、対話の `ralph init` が言語パックのあとに、使う AI の CLI(Claude Code・Codex・両方)、計画の確認、変更のレビュー、PR ごとに CI がテストを走らせるか、テストの通った PR を聞かずに merge するかを聞き、答えを `ralph.toml` の該当の key に書くようになった。`ralph.toml` が既にあるとき(`--force` なし)と `--yes` では聞かず、`--yes` は雛形のままの `ralph.toml` を書く。コードの変更は `internal/cli/init.go`、新しい `init_settings.go` とそのテスト 3 本、`README.md` の `ralph init` の行で、help と README の init の行は実装のスライスの中で更新済みだった。

現在形の文書を洗った結果、直したのは verify の V1 だけである。対話の init は選んだ CLI の `[doctor].require_claude_cli`・`require_codex_cli` を `true` で書くので、「どちらか 1 つで doctor は通る」「`require_*` は既定で `false`」とだけ書く 3 か所(README の doctor の行、codex-setup recipe とその雛形側の写し)が、対話の init で作った project について言い足りなかった。`--yes` と `ralph.toml` が既にあった project では前の文のとおり正しいので、条件を足す形で直した。あわせて org の spec に、#264・#274 の注と同じ形で #277 の日付つきの注を足した。

## Changes made

| File | Change |
|------|--------|
| `README.md:134`(`ralph doctor` の行) | 「require_* が `true` の CLI が使えなければ fail」の文のあとに、「対話の `ralph init` は使う CLI を `true` で書く(両方を選んだら両方)ので、その project の doctor は選んだ CLI が使えないと fail する。`--yes` と、`ralph.toml` が既にあった project は両方 `false` のまま」を足した(V1) |
| `docs/recipes/codex-setup.md:20-22` | コードブロックのコメント「(either one passes)」を「(either one passes unless ralph.toml requires one)」に直し、折り返しを合わせた(V1) |
| `docs/recipes/codex-setup.md:34-42` | 「Codex だけの machine では `Claude Code CLI` の行は `warn` で doctor は通る」に、「`require_claude_cli` が `false` の間は。`ralph init --yes` はそれを書く。対話の init は Claude Code か両方を選ぶと `require_claude_cli`、Codex か両方を選ぶと `require_codex_cli` を `true` にし、必要な CLI が使えないと doctor は fail する。Codex だけの machine で使うときは `false` に戻す」を足した(V1) |
| `templates/base/docs/recipes/codex-setup.md` | 上の 2 か所と同じ編集。`cmp` で root の写しと同じ |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-8 の末尾、#274 の注のあと) | 「2026-11-02 に決めたこと(issue #277)」の注を足した。対話の init が聞く問い、書く key、CLI が 1 つのときの扱い(false、`model_pool` の行を `#` でコメント、Codex だけなら `watcher_enabled = false`)、`require_*` を true で書くこと、#264・#274 の「雛形は `local`・`auto_merge = true`」が雛形と `--yes` について今も正しいこと、既定値は変えていないこと、`ralph.toml` があるときは聞かないこと |
| `docs/plans/active/2026-11-02-init-config-questions.md` | `## Progress checklist` の「Verification artifact created」「Test artifact created」を `[x]` にし、report の SHA と verdict を書いた。ほかの節は触れていない(digest は同じ) |
| `docs/reports/sync-docs-2026-11-02-init-config-questions.md` | この report |
| `docs/insights/events/2026-11-02-init-config-questions.jsonl` | `sync_docs` の行を `insights-append.sh` で足した |

## Decision: spec に #277 の注を足した

verify の情報 Info は、足すかどうかを sync-docs で決めるとしていた。足した。理由は 2 つある。

1. FR-7 と FR-8 の注の系列(#264、#265、#267、#269、#274)は、`ralph.toml` の既定値や `ralph init` の書く値を決めるたびに日付つきの注を足してきた。#264 と #274 の注は「`ralph init` のテンプレートは `local`・`auto_merge = true` を書く」と読めるので、あとから読む人は `ralph init` がいつもその値を書くと受け取りうる。対話の init は答えで書き換えるので、その文が成り立つのは雛形と `--yes` だという限定が要る。
2. Codex だけのときの `watcher_enabled = false` と `model_pool` の行のコメント化は、issue の表にない key の書き換えで、plan の D8・D9 にだけ理由がある。plan は archive に移るので、spec に残す。

注は FR-8 の末尾に置いた。`auto_merge` の注(#274)が同じ場所にあり、`regate_checks` の注(#264)への参照を本文に入れたため。FR-7 と FR-8 のチェックボックスは変えていない(この変更は FR の達成度を動かさない)。

## Surfaces checked for drift

| Surface | 結果 | 根拠 |
|---------|------|------|
| `README.md:127`(`ralph init` の行) | 変更なし(同期済み) | 言語パックのあとの CLI の問い、「(Recommended)」の付き方、Both のときだけの 2 問、CI と自動 merge、`ralph.toml` があれば聞かないこと、`--force` は init の書くすべてのファイルを上書きすること、`--yes` は雛形の `ralph.toml` を書くことが、`init.go:41-54` の help と `askInitSettings` の 1 行の知らせに合う |
| `README.md:134`(`ralph doctor` の行) | 直した(V1) | 上の Changes made |
| `README.md:58-66`(Quick start) | 変更なし | `ralph init my-project` の対話の形は同じで、`ralph doctor # environment check (Claude Code or Codex)` は大づかみの説明のまま通る |
| `README.md:176-`(What `ralph init` scaffolds) | 変更なし | 配る tree の説明で、`ralph.toml` の中身には触れない |
| `docs/recipes/codex-setup.md`(root と `templates/base/` の写し) | 直した(V1) | 上の Changes made。`scripts/check-sync.sh` は 2 つを byte 比較し、通る |
| `docs/recipes/adding-a-language-pack.md`(root)と `templates/base/` の写し | 変更なし | 言語パックの問いと `--yes` の説明で、今も正しい。後ろの問いには触れないが、pack の recipe なので要らない(verify と同じ判断) |
| `docs/recipes/codex-seat-permissions.md:347` | 変更なし | 「`codex_verified = true` が `ralph init` の書く値」は雛形の値で、雛形は変えていない |
| `templates/base/ralph.toml` の `[doctor]` の注 | 変更なし | 「Set require_claude_cli or require_codex_cli to true to make doctor fail when that CLI is not usable」は雛形の既定(`false`)について正しい。対話の init は書き換える側で、雛形は defaults_sync_test.go が `Default()` と突き合わせるので触れない |
| `AGENTS.md`、`CLAUDE.md`、`templates/base/AGENTS.md`、`templates/base/CLAUDE.md`、`.ralph/core/` | 変更なし | `internal/cli/` の行(`AGENTS.md:86`)はサブコマンドを並べ、ファイルは並べない。`init_settings.go` は init の一部なので足す必要はない(verify と同じ判断)。`ralph init` はスキャフォールドの入口として挙げるだけ |
| `.claude/rules/ralph/`、`templates/base/.claude/rules/ralph/` | 変更なし | `ralph init` の当たりは `agent-messaging.md` の「scaffold に含まれない」の注記だけ。`ralph-workflow.md` と `post-implementation-pipeline.md` の `[standard].plan_advisory`・`cross_review` の「default off」は雛形と `config.Default()` の既定で、今も正しい(対話の init は Both のときに答えで true にしうるが、既定は変えていない)。`model-routing.md` の `[reviewer]` は init が触らない key |
| `.claude/skills/org/SKILL.md:212`(と `.agents/`、`templates/base/` の写し) | 変更なし | 「`ralph.toml` は seed-once(初回 `ralph init` で生成されたあと `ralph upgrade` は触らない)」は、答えを書いた `ralph.toml` も seed のままで、manifest の `TemplateHash` を雛形のハッシュにしたので upgrade の advisory も出ない(AC7)。今も正しい。skill は触れていないので `sync-skills.sh` は走らせていない |
| `.codex/README.md`、`.codex/config.toml`(と `templates/base/` の写し) | 変更なし | `ralph init` の当たりは Git hook の配置の説明だけ |
| `docs/architecture/repo-map.md`、`docs/quality/` | 変更なし | `repo-map.md` は `internal/cli` のファイルを並べない。`docs/quality/` に init の問いの記述はない |
| `docs/tech-debt/README.md` | 変更なし | init に触れる行(20、160、368)の Trigger はどれもこの変更で起きない(verify が 4 つ目の列を読んで確かめた)。`runInitInteractive` や #277 を名指す行はなく、足さなかった(下の Follow-ups) |
| `docs/specs/2026-10-07-org-multi-org-director.md`(#264・#274 の注) | 変更なし(#277 の注を足した) | 上の Decision |
| `docs/specs/2026-05-07-codex-cli-parity.md:13`、`:155` | 変更なし | 「`require_claude_cli = true` が既定」(`:13`)は当時の記述で、今の雛形(`templates/base/ralph.toml`)の既定は `false` になっている。この変更より前のずれで、日付つきの spec の初期の記述。「`ralph init` で CLI 選択」(`:155`)は ❌ の却下案だった。歴史の記録なので触らない |
| `docs/specs/2026-04-16-ralph-cli-tool.md:28` | 変更なし | 初期の spec の init の問い(Ralph Loop、TUI)。歴史の記録 |
| `scripts/`、`tests/`(現在形の呼び出し) | 変更なし | `ralph init` を呼ぶのは `tests/test-check-template.sh` の G(`--yes --pack`)で、`--yes` は雛形のままなので影響を受けない。`scripts/` の当たりは scaffold の説明のコメント |

## Info(履歴の記録として残したもの)

- verify の情報どおり、計画の `## Verify plan` は「Spec compliance criteria to confirm: AC1〜AC9」のままで、計画助言で足した AC10 が入っていない。この節は digest の対象なので書き換えない。verify report は AC10 も見ている。
- `docs/specs/2026-05-07-codex-cli-parity.md:13` は `[doctor] require_claude_cli = true` を既定と書く。#272 より前の記述で、この変更が作ったずれではない。直すなら別の作業で、日付つきの注を足す形が合う。

## Plan drift notes

- 計画の Scope・Design decisions・Acceptance criteria の本文は digest の対象で、この run は触れていない。AC1〜AC10 は実装のときに `[x]` が付いている。残る `[ ]` は Progress の「PR created」だけで、`/pr` が付ける。
- 実装中に計画の外へ出た点は、Progress の S1〜S3 と self-review の行に記録済み。この run で新しく見つかった差はない。

## Follow-ups(この PR では足していない)

- `runInitInteractive` の 2 つのフォームのつなぎ(`init.go:165-191`)を通す Go のテストがない。端末がないと届かない行で、絞った coverage でも 0 回だった(test report の Test gaps 1、verify の Coverage gaps)。pty の確かめと、`go build -overlay` で作った変異 4 つの binary がこの行を守っている。`docs/tech-debt/README.md` には行を足していない。足すかどうかは `/pr` の前に lead が決める。足すなら Trigger は「`runInitInteractive` に問いか group を足すとき」が自然である。直すなら、verify が書いたとおり 2 つの `Run()` を差し替えられる形(`func(*huh.Form) error`)にして、accessible モードで「CLI に 3、yes/no に 4 つ」を流し、`executeInit` に渡る `Settings` を当てるテストを 1 本足す。
- test report の Test gaps 2(`[org].model_pool = [` の行が 2 つある雛形のエラーを当てるテストがない)は影響が小さい(そういう雛形は `config.Load` も通らない)。
- self-review の L2(`executeInit` が 171 行)と L5(Codex だけのときの出力が `watcher_enabled = false` を説明しない)は、plan の Progress に直さない理由が書いてある。
- 2 つ目のフォームから 1 つ目に戻れない点は、plan のリスクの表にある既知の穴で、PR に書く。

## Release notes(`/pr` の本文に置く)

verify の Spec compliance、test report の実物の binary の確かめ、plan の Rollout に基づく。

1. 対話の `ralph init` が、言語パックの問いのあとに、使う AI の CLI(Claude Code・Codex・両方)を聞く。PATH で見つかった CLI(両方見つかれば Both)に「(Recommended)」が付く。両方を選ぶと、続けて計画を別の AI に見せるか、変更を別の AI にレビューさせるか、PR ごとに CI がテストを走らせるか、テストの通った PR を聞かずに merge するかの 4 問、1 つを選ぶと CI と自動 merge の 2 問になる。5 問(CLI が 1 つなら 3 問)の答えは `ralph.toml` に書かれる。
2. 書かれる key は、`[org].driver_pool`、`[doctor].require_claude_cli` と `require_codex_cli`、`[standard]` と `[org.review]` の `plan_advisory` と `cross_review`、`[org.gate].regate_checks`(CI ありなら `"ci"`、なしなら `"local"`)、`[org.director].auto_merge`。CLI が 1 つのとき、`plan_advisory` と `cross_review` は false で書かれ、`[org].model_pool` のもう一方の driver の行は消されずに `#` でコメントになる。Codex だけのときは `[org.watchdog].watcher_enabled = false` も書かれる(watcher は claude しか呼ばないため)。
3. 選んだ CLI の `require_*` が `true` で書かれるので、その CLI が使えない machine では `ralph doctor` が fail する。前の `ralph init` が書いた `false` のままの project は変わらない。
4. 聞かない場合がある。`--yes` は何も聞かず、雛形のままの `ralph.toml` を書く(前と同じバイト列)。`ralph.toml` が既にある対象のディレクトリでは、`--force` がなければ設定の問いを出さず、1 行で知らせて `ralph.toml` を変えない。`--force` なら聞いて、答えで上書きする。
5. 雛形と `config.Default()` の既定値は変えていない。答えを書いた `ralph.toml` の直後の `ralph upgrade --dry-run` は、`ralph.toml` の advisory を出さない。

PR の Notes for reviewers に入れる点: 2 つ目のフォームから 1 つ目に戻れない(言語パックを選び直すには Ctrl-C でやり直す)、`TERM=dumb`(accessible モード)では huh が description を出さない、上の Follow-ups の 1 つ目(`runInitInteractive` の配線は Go のテストがなく、pty の確かめだけ)。

## Verification

| Check | Result |
|-------|--------|
| `./scripts/check-sync.sh` | rc 0。`PASS: all files in sync.`(TEMPLATE_ONLY 10、KNOWN_DIFF 6)。`docs/recipes/codex-setup.md` と `templates/base/docs/recipes/codex-setup.md` は byte 比較の対象で、`cmp` でも同じ |
| `./scripts/plan-visual.sh digest docs/plans/active/2026-11-02-init-config-questions.md` | `d1e86274a18e`(`- Approved:` の行の値と同じ。Progress の編集の前後で同じ) |
| `git diff --check` | rc 0、出力なし(commit の前、追跡ファイルの変更に対して) |
| 現在形の文書の `grep` | `ralph init`、`require_claude_cli`、`require_codex_cli`、init が書く値を言う文(`ralph init writes`、`init's template`、key 名の近くの `scaffold`)を、README、AGENTS、CLAUDE、`docs/recipes`、`docs/quality`、`docs/architecture`、`templates/base/docs`、`templates/base/ralph.toml`、`.claude/rules`、`.claude/skills`、`.ralph/core` で打った。直した 3 か所のほかに、新しい振る舞いと食い違う文はなかった |
| `go test`、`go build`、golangci-lint、`run-*.sh` | 打っていない。依頼で Go とテストは触らず、この run が変えたのは文書、計画の Progress、この report、insight event で、コードの動きは変わらない |
| insight event | `./scripts/insights-append.sh --slug init-config-questions --flow standard --phase sync_docs --verdict pass --cycle auto --source skill` で追記した(rc 0、末尾の行が `phase":"sync_docs"`、`cycle":1`、`verdict":"pass"`) |
