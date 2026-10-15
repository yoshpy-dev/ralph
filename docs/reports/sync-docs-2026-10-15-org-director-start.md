# sync-docs report: org-director-start

- Date: 2026-10-15(JST)
- Plan: `docs/plans/active/2026-10-15-org-director-start.md`(承認済み。`./scripts/plan-visual.sh digest` は `671c60416856` で、`- Approved:` の行と同じ。この run で計画は触っていない)
- Pipeline cycle: 1。差分の base は `origin/main`(f21df66c)との merge base、branch は `feat/org-director-start`。run 開始時の HEAD は `1cdddbbb`(run 開始時の値で、この report を含む commit の SHA ではない)
- 先行 report: `docs/reports/self-review-2026-10-15-org-director-start.md`(Merge: yes、CRITICAL・HIGH 0、MEDIUM 1、LOW 6)、`docs/reports/verify-2026-10-15-org-director-start.md`(pass。文書のずれ V1〜V6)、`docs/reports/test-2026-10-15-org-director-start.md`(pass、失敗 0。Test gaps 1〜10)
- 項目の番号: V1〜V5 は verify の「Documentation drift」の表の番号。D4・E1・E12・E13・P2 は test report の mutation の ID。「項目 N」は依頼の項目 N
- 内容の commit: この report、直した文書、insight event を含む 1 つの commit(`docs: sync docs for org-director-start`)
- コードとテストは変えていない。変えたのは Markdown、`templates/base/ralph.toml` のコメント、`.agents` の生成物だけ

## Summary

依頼の 6 項目をすべて直した。V1(`/org` skill)、V2(`ralph-workflow.md`)、V3(`ralph.toml` のコメントと tech-debt の同じ主張)、V5(tech-debt 189 行)、test gap の新しい行、self-review の「22 字」。あわせて掃き出しで、見張りの関門を挙げる文書 4 か所(`quality-gates.md` の 2 面、`repo-map.md`、`AGENTS.md`)と、`/org` skill の中の director の文 4 か所を直した。

## Changes made

| File | 項目 | Change |
|------|------|--------|
| `.claude/skills/org/SKILL.md`(「前提」節の古い台帳の項) | 項目 1(V1) | 古い台帳が動いている座席を持つとき止まる動詞の一覧に `director start` / `director stop` を足した。コードの `orgLedgerMutating` のコメントは、この PR が同じ 2 つを足している |
| 同上(動詞リファレンスの `director` の行) | 項目 1(V1) | director を「昇格させた対話セッションか、`director start` で立てる headless のセッション」と書き直した。登録のイベントに headless の `workspace_created`・`pane_created`・`launched`・`stopped` を足した。`director status` の欄に headless の状態・pane・workspace・model を足した。`director start`(herdr の外の端末、`HERDR_PANE_ID` で断る、見張りの起動、`attached` を先に書く順、workspace と tab の label、stdout の 1 行、登録がある間と `enabled = false` の拒否、`--replace` なし、失敗時の後始末、permission mode)と `director stop --id <id> [--force]`(label の確認、C-c と close、`stopped` と `detached` の同じ write、`--force`、自分の pane と workspace の中からの拒否)を足した。座席ではなく manifest に書かれず、`stop --all` が止めないことと `note:` の 1 行を書いた。例の欄に start と stop を足した。詳細は `/director` skill の「headless の director」節に送った。`stop`・`disband` の行は触っていない(tech-debt の F-8 の行が、その 2 つのセルの長さを書いているため) |
| 同上(「受信箱」節の 2 か所) | 項目 1(V1) | 件を読む director を「昇格させた対話セッションか、`director start` で立てた headless のセッション」にした。`enabled = false` のとき断られるのは attach だけでなく `director start` も、とした(`checkDirectorStart` が `enabled` を見る) |
| 同上(「permission 作法」節の `[org.permissions]` の項) | 項目 3(V3) | `[org.permissions.roles]` の説明に `director = "mode"` を足した。director は座席ではなく、`director start` が立てる headless の director の mode を決める。`guarded` は flag を渡さないので、claude も codex も人自身の承認の設定に従う、と書いた(`permissionArgsForDriver`)。この節の既存の文は座席の話で、director の分だけを足した |
| 同上(「permission 作法」節の見張りの項) | 掃き出し | 「`director` の動詞は見張りがなくても使える」は `director start` に当たらなくなっていた。`director start` を除き、`director start` は `ralph org start` と同じく見張りがなければ起動し、止まりかけ・印が古い・`--once` の見張りでは断られる、と書いた(`wireOrgStartWatchGate`、`TestOrgDirectorStart_WatchGate`) |
| `.agents/skills/org/SKILL.md`、`templates/base/.claude/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 項目 1(V1) | `.claude` の 2 面は同じ edit を Edit で入れた。`.agents` の 2 面は `./scripts/sync-skills.sh` と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で作り直した。4 面の md5 は `0fe994923d6ec7c734c8e51c1d43f8ed` で同じ |
| `.claude/rules/ralph/ralph-workflow.md`、`templates/base/.claude/rules/ralph/ralph-workflow.md` | 項目 2(V2) | `/director` の項に 2 行足した。「A human can instead launch a headless director with `ralph org director start` (same skill, ended with `director stop`).」。2 面は同一のまま(66 行から 68 行) |
| `templates/base/ralph.toml`(`[org.permissions.roles]` のコメント、director の注の終わり) | 項目 3(V3) | 「A guarded claude director stops at the permission prompt in its herdr pane, where the human talks to it.」を、guarded は permission の flag を渡さないので director は agent 自身の承認の設定(人の Claude Code の設定か `~/.codex/config.toml`)に従い、たとえば claude の `permissions.defaultMode = "bypassPermissions"` では prompt なしで動く、に直した。すぐ上の reviewer の注の「follows codex's own approval settings」と同じ言い方にした |
| `docs/tech-debt/README.md` 251 行(headless の director の permission の行、Impact 列と Related 列) | 項目 3(V3) | 同じ主張「`director = "guarded"` は claude の director が各 permission prompt で止まる」を直した。flag を渡さないので agent 自身の設定に従うこと、verify の実機の機械では `~/.claude/settings.json` が `bypassPermissions`(実機は空の `CLAUDE_CONFIG_DIR` で避けた)、`~/.codex/config.toml` が `approval_policy = "never"` で guarded の codex の director が prompt なしで動いたこと(L3)を書いた。Related に verify の evidence(L3 と「Restart of the dedicated server for claude」)を足した |
| `docs/tech-debt/README.md` 189 行(別の版の `ralph` の行、Trigger 列と Related 列) | 項目 4(V5) | 「the version skew can reach the director too ... (a judgment, not measured)」を測った結果に直した。codex の director の最初のコマンド `ralph org director status --state-dir '<ledger>'` が、pane の shell が見つけた Homebrew の `ralph` 5.1.0(`org director` がない)で動いて `ralph org` の help を返し、director は雛形の最初の手順で「id を確かめられない」と書いて止まった。dev build と古い install が並ぶ形で、同じ版を入れた利用者にはこの形は起きない。「trigger は人のいない director が始めるものに移る」の側は、測っていないので「(a judgment, not measured)」を残した。Related に verify の evidence(L3)を足した |
| `docs/tech-debt/README.md` 256 行(新しい行) | 項目 5 | test report の mutation の生き残り 5 件(D4、E1、E12、E13、P2)を 4 つの test gap の行にした。各 gap に、test report が提案する一番小さいテスト(D4 は `TestStartDirector_RefusesBeforeWritingOrCallingHerdr` に「director.jsonl unreadable」の行、E1 は 1 回目の write を ENOSPC にするテスト、E12・E13 は失敗した create が出力も返す `fakeHerdr` の包み、P2 は `startAttachedError` を直接呼ぶ表のテスト)と、probe の名前(PD4、PE1、PE12、PE13、PP2)を書いた。コードは関数名とテスト名で引いて、`file:line` は使っていない。A12〜A14 は seat のテストが止めているので gap にせず、行の Why deferred と Trigger に「director の stop が `confirmSeatPane` と `confirmOrgWorkspace` を使わなくなったら足す」と書いた。8a-2 の既存の行(251〜255)の直後に入れた |
| `docs/reports/self-review-2026-10-15-org-director-start.md`(「確かめていないもの」の項と Coverage gaps の項) | 項目 6 | label の長さ「22 字」を「27 字」に直し、「27 字は 15 + 12 で、初版の『22 字』は /sync-docs で訂正した」と添えた(`ralph_director_` は 15 字、token は 12 字)。2 行だけの変更で、finding の重大度、`Merge:` の行、Follow-ups は触っていない。sync-docs の skill はこの report を直すことを禁じていない。計画の同じ数字は b6c86188 で直っていて、verify の V4 は指摘の記録なので触らない |
| `docs/quality/quality-gates.md`、`templates/base/docs/quality/quality-gates.md`(「Cross-org watch gate」の行) | 掃き出し | 見張りが要る動詞に `ralph org director start` を足した。`start` に加えて `director start` が見張りを起動すること、見張りなしで使える動詞を `director` の動詞のうち `start` 以外にした。2 面は同じ 3 か所に同じ語を足した。`check-sync.sh` は DRIFTED 0 のまま |
| `docs/architecture/repo-map.md`(`.claude/skills/director/` の行) | 掃き出し | 「promote the current session」だけだったので、`ralph org director start` が herdr に立てる headless の director にも使う skill であることを足した |
| `AGENTS.md`(repo map の `internal/org/` の項、`watchgate.go` の句) | 掃き出し | 「refuses spawn, start, and TASK sends」を「spawn, start (and `director start`), and TASK sends」にした。1 句だけで、行数は増えていない |

## Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `README.md`(動詞の表 124 行、`ralph status` の行 125、director の段落 258 行) | 変更なし。124 行は `director start/stop` を、125 行は headless の状態・`open_headless` を、258 行は start・stop・`--force`・permission・`stop --all` の note・`launch_cmd` を後の段に送ったことを書いていて、コードと合う。256 行の「`ralph org start` launches it when none runs」は `ralph org start` の話で正しい(`director start` の見張りは 258 行が書く) |
| `AGENTS.md` の repo map | 上の 1 句のほかは変更なし。`director_start.go`・`director_stop.go` の足しは PR が入れたもので、コードと合う |
| `/director` skill(4 面) | 変更なし。PR が全面を書き直していて、`ralph org director start` の help と実機の出力(stdout の 1 行、label、拒否の文、`--force`、`stop --all` の note)と合う。permission の節に「prompt で止まる」の主張はない |
| `.claude/rules/ralph/`(ほかの rule、`CLAUDE.md`、`agent-messaging.md`) | 変更なし。`agent-messaging.md` は typed message の検査と TASK の関門を書くだけで、director の動詞を挙げない。`CLAUDE.md` の「手動の skill は `/release` だけ」は `/director` が自動の skill であることと合う |
| `docs/specs/2026-10-07-org-multi-org-director.md` | 変更なし。FR-9 の 8a-2 の注は PR が入れ、`launch_cmd` を 8a-2b に送ったことを書く。`docs/specs/2026-08-01-org-runtime.md` は後の動詞を挙げず後継 spec を指すので触らない |
| `docs/recipes/` | 変更なし。director を扱う記述がない(`codex-seat-permissions.md` の `director` は directory の部分文字列) |
| `docs/quality/definition-of-done.md`、`scripts/ralph-config.sh`、`.ralph/core/AGENTS.core.md` | 変更なし。org の記述がない、またはこの PR の語に触れない |
| `ralph org --help` と `director start|stop` の help | 変更なし。help の動詞・flag(`--driver`、`--model`、`--timeout-ms`、`--id`、`--force`)は実機の出力と合うと verify が確かめている。help は Go のコードで、触っていない |
| `templates/base/ralph.toml` の `[org.director]` のコメント | 変更なし。`enabled = false` が attach と start を断ることまでは書かないが、元から attach のことも書いていない(足りないだけで、誤りではない) |
| `docs/tech-debt/README.md` の列の数、`file:line`、計画の参照 | 変えた 3 行(189、251、256)はどれも 5 列(列を区切る `|` で 7 片)、backtick は列ごとに偶数、`file:line` は 0。`run-static-verify.sh` の「tech-debt README plan references」は OK。`docs/plans/active/2026-10-15-org-director-start.md` で引いているので、`/pr` の archive が書き換える |
| `docs/tech-debt/README.md` の stop・disband のセルの長さの行(F-8) | 変更なし。`/org` skill の `stop` と `disband` の行は触っていない |

## Left open

- 計画の Acceptance criteria と Progress checklist の checkbox(verify の V6)は、計画をオーケストレーターが持つので触っていない
- `templates/base/ralph.toml` の reviewer の注(「a guarded claude reviewer stops at the permission prompt」)は、director の注と同じ型の主張だが、この PR の差分の外の既存の文で、director の依頼に入っていないので直していない。`permissionArgsForDriver` の guarded の動きから、同じ書き直しが当てはまると考えられる
- Go のコメントにある `docs/plans/active/2026-10-15-org-director-start.md` は、`/pr` の archive のあと古くなる。8a-1 のときと同じ形で、既存の tech-debt の扱い(Go のコメントの `active/` は別の後始末)に従った
- test report の Test gaps 6〜10(通らない文、`USER` のない登録、claude の director が雛形どおり動くこと、`ralph init` の scaffold、macOS 以外)は、文書の話ではなく、tech-debt の行にもしていない。A12〜A14 は上の行の Why deferred に書いた
- `docs/quality/definition-of-done.md` の最後の項(committed な evidence は `$HOME` を `~` にする)は、`docs/evidence/test-2026-10-15-org-director-start.md` と `docs/evidence/verify-2026-10-15-org-director-start.md` のどちらも `/Users/` が 0 件で満たしている。前の PR で残っていた `/Users/<user>/...` の行はこの PR にはない
- 実機の動きは確かめていない。この run は文書の同期で、コードを変えていないので、verify と test の実機の結果はそのまま有効

## Verification

commit の前に、worktree のルートで打った。

- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `IDENTICAL: 169`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `go test ./internal/config/... ./internal/scaffold/... ./internal/org/ -run 'Skill|Sync|Defaults' -count=1`: `ok`(config 0.536s、scaffold `[no tests to run]`、org 1.286s)
- `go test ./internal/scaffold/... ./internal/config/... ./internal/upgrade/... -count=1`: 3 パッケージとも `ok`(`templates/base/ralph.toml` のコメントを pin するテストがないことの確認)
- `./scripts/run-static-verify.sh`(`templates/base/ralph.toml` の変更で言語の範囲は full): rc 0、`==> All verifiers passed.`。「tech-debt README plan references」は OK、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned f21df66c..1cdddbbb against origin/main: clean`。出力 425 行は scratchpad に落として読んだ。wrapper のログ(`docs/evidence/verify-2026-10-15-102652.log`)は gitignore の対象で、コミットしていない
- `git diff --check`: 出力なし
- 変えたファイルの U+FFFD は 0 件
- `/org` skill で足した折り返しの行は、全角を 2 として数えて 76 桁以下
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-org-director-start.md`: `671c60416856`(run の前も後も同じ)
- 変えた tech-debt の 3 行は、`git show HEAD:docs/tech-debt/README.md` と比べて、変わった行がこの 3 行だけ(189 と 251 は書き換え、256 は挿入)であること、列の数 5、backtick の偶数、`file:line` 0 を script で確かめた
- self-review report の差分は 2 行で、`Merge:`・重大度・Follow-ups の行を含まない
- `./scripts/insights-append.sh --slug org-director-start --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`(slug は日付なし): rc 0。`git status --short` は既存の `docs/insights/events/2026-10-15-org-director-start.jsonl` を `M` と示し、新しい `??` のファイルはない。足した行は cycle 1、verdict pass

## Verdict

Verdict: complete。依頼の 6 項目を直し、掃き出しで 4 か所の文書のずれを直した。静的検査と同期の検査は通った。コードとテストは触っていないので、verify と test の結果は変わらない。確かめていないのは、`/org` skill の director の行の文が実機の出力と 1 文字ずつ合うことで、help と `/director` skill の文から書き、実機の出力を取り直してはいない。
