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

## Cycle 2

- Date: 2026-10-15(JST)
- Plan: `docs/plans/active/2026-10-15-org-director-start.md`(承認済み。`./scripts/plan-visual.sh digest` は `f65d7c59746c` で、`- Approved:` の行と同じ。この run で計画は触っていない。cycle 1 の節の `671c60416856` は、その後の 2 回の承認し直しの前の値)
- Pipeline cycle: 2。run 開始時の HEAD は `6556bce5`(run 開始時の値で、この節を含む commit の SHA ではない)。差分の base は `origin/main`(f21df66c)との merge base
- cycle 1 の sync-docs(e80ccafe)のあとに入った変更: 3243c499 と a539a765(start の `--config` を絶対パスにして、雛形の中で director が打つ `ralph org` のコマンドに付ける。雛形の stop の行と `ralph status` は `--state-dir` だけ。`{{CONFIG}}` を `{{CONFIG_LINE}}` に改名)、4ef33195(cycle 1 の Test gaps 1〜4 のテスト)
- 先行 report の cycle 2 の節: self-review(Merge: yes、LOW 5)、verify(pass、情報 C2-V1〜C2-V5)、test(pass、失敗 0。Test gaps 1〜6)
- 項目の番号: 「項目 N」は依頼の項目 N。C2-V は verify の cycle 2 の節の ID、K15・K16 は test の cycle 2 の mutation の ID
- 内容の commit: この節、直した文書、insight event を含む 1 つの commit(`docs: sync docs for org-director-start (cycle 2)`)
- コードとテストは変えていない。変えたのは Markdown と `.agents` の生成物だけ

### Summary

依頼の 3 項目をすべて直した。項目 1(C2-V3)は README、仕様の 8a-2 の注、`/org` skill の `director` の行に、start の `--config` が雛形に運ばれることを 1 句ずつ足した。項目 2(C2-V4)はコードで事実と確かめ、`/director` skill に 1 文、tech-debt に 1 行を足した。項目 3 は test の K15 と K16 を tech-debt の新しい 1 行にした。掃き出しでは、ほかに古くなった文書は見つからなかった。

### Changes made

| File | 項目 | Change |
|------|------|--------|
| `README.md`(258 行、director の段落) | 項目 1(C2-V3) | stop の説明の次に 1 文足した。start に渡した `--config <file>` は、絶対パスにして、director の役割の雛形が打たせる `ralph org` のコマンドに運ばれる。`ralph status`(`--config` がない)と、人に渡す stop のコマンドには運ばれない |
| `docs/specs/2026-10-07-org-multi-org-director.md`(FR-9 の 8a-2 の注) | 項目 1(C2-V3) | permission mode の文の次に 1 文足した。start の `--config` は絶対パスにして雛形の中で director が打つ `ralph org` のコマンドに付ける。理由は director が start と同じ設定を読むため。`ralph status` には `--config` がなく、stop の打ち方は ralph.toml を読まないので付けない |
| `.claude/skills/org/SKILL.md`(`director` の行) | 項目 1(C2-V3) | permission mode の文の次に 1 文足した。`--config` を渡した start は、その絶対パスを、雛形の中で director が打つ `ralph org` のコマンドに付ける(`ralph status` と人に頼む stop には付けない) |
| `templates/base/.claude/skills/org/SKILL.md`、`.agents/skills/org/SKILL.md`、`templates/base/.agents/skills/org/SKILL.md` | 項目 1(C2-V3) | `.claude` の 2 面は同じ edit を Edit で入れた。`.agents` の 2 面は `./scripts/sync-skills.sh` と `CLAUDE_ROOT=templates/base/.claude/skills CODEX_ROOT=templates/base/.agents/skills ./scripts/sync-skills.sh` で作り直した。4 面の md5 は `c1bb29c022619e5bf9b4572309ca7119` で同じ。`stop`・`disband` の行は触っていない(F-8 の行がセルの長さを書いているため) |
| `.claude/skills/director/SKILL.md`(「始める」の 3)と同じ 3 面 | 項目 2(C2-V4) | 「`ralph status` には `--config` がない。」を、1 文に直した。`ralph status` には `--config` がなく、leader の `ralph org escalate` も `--config` なしで読むので、この 2 つは start の `--config` のファイルでなく、`--config` なしで読む `ralph.toml` の `[org.director].enabled` を見る。そちらが false で `--config` のファイルが true なら、`ralph status` の `director:` の行は `disabled` と出て、leader の件は人へすぐ届く。直前の「cwd がどこでも、start と同じ台帳と設定を使うため」は、運ぶ文の理由なので、その文の直後に動かした。4 面の md5 は `52e76af939fd551478ee8aa969700d2e` で同じ。折り返しの行は全角を 2 として数えて 78 桁以下(この file の元の最大も 78) |
| `docs/tech-debt/README.md` 257 行(新しい行) | 項目 3(K15・K16) | 項目 3 の 2 件を 1 行にした。(a) K15: `checkDirectorStart` の `resolve the --config path` の拒否を固定するテストがない(隣の state dir と cwd の拒否もない。`absPath` を差し替えるのは `internal/org/spawn_test.go` だけ)。一番小さいテストは probe の `TestProbe_C2_AConfigPathThatCannotBeMadeAbsoluteIsRefused`(`ConfigPath` のときだけ `absPath` を失敗させ、error の前置き、herdr の呼び出しが 0、`director.jsonl` と `prompts` がないことを見る)。(b) K16: skill の「始める」の 3 の印 `(start の --config)` と `RenderDirectorPrompt` の印を突き合わせるテストがない。一番小さいテストは probe の `TestProbe_C2_DirectorSkillNamesTheConfigLineMarker`(`TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase` と同じ形で、4 つの SKILL.md が印を backtick で囲んで含むことを見る)。どちらも HEAD で通り、狙った mutant で落ちる。K15 の影響は cwd が読めないときだけとみる、と test report の推測のまま書いた。cycle の上限の文は入れていない(上限は上げられるので、古くなる)。既存の A12〜A14 の行(256 行)には足さず、別の行にした。A12〜A14 は seat のテストが止める話で、K15・K16 は止めるテストが 0 の話で、Trigger が違うため |
| `docs/tech-debt/README.md` 258 行(新しい行) | 項目 2(C2-V4) | `ralph status` に `--config` がないことの行。Debt は `--config` の start で director が `<file>` を読み、`ralph status` と leader の escalate が `--config` なしの ralph.toml を読むこと、見える向きは「`--config` なしで読む側が false、`<file>` が true」だけであること、逆向きは start が断るが、断りの文の「`ralph org escalate` はすでに人へ送る」は `<file>` の値で escalate が読む値ではないこと、警告は skill の 1 文だけであること。Impact は director が付いているのに先頭に立たず、件が人へすぐ届くこと、件は inbox に残ること。Why deferred は、8a-2 が `--config` を director 自身のコマンドにだけ運ぶこと。Trigger は、ずれの報告か、`ralph status` の `--config`(そのときは leader の escalate のファイルの出どころも決める)。evidence は verify の C2-5 |

### 項目 2 の確かめ(C2-V4 は事実だった)

- `ralph status` に `--config` がない。`newStatusCmd` の flag は `--state-dir`、`--org-id`、`--json`。`statusOrgConfig` は `resolveOrgConfig("")`(cwd の `./ralph.toml`)で、`statusDirectorEnabled` は `withMainWorktreeDirectorEnabled(cfg, "", ...)` を呼ぶ。help の文も「`ralph org escalate` reads it without --config」と書く
- `ralph org escalate` は `--config` を受けるが、leader は付けない。`withMainWorktreeDirectorEnabled` は `configPath` が空で台帳が main worktree のものなら、main worktree の ralph.toml の `[org.director].enabled` を取る。`--config` があるか、台帳が main worktree のものでなければ、cfg をそのまま返す
- start は `--config` のファイルの `enabled` で判定する(`startDirector` が `withMainWorktreeDirectorStartConfig` のあと `cfg.Director.Enabled` を見る。`--config` があれば cfg は変わらない)。false なら断る。だから食い違いが見えるのは、`--config` なしで読む側が false、`<file>` が true の向きだけ。verify の C2-5 はこの向きを、`./ralph.toml` が false、`cfg/ralph.toml` が true で測っている
- 逆向きの断りの文(`startDirector`)は「`[org.director].enabled` is false in the --config file X, so ralph org escalate already sends every item to the human at once」と言う。`<file>` が false で repo の側が true なら、escalate は人へすぐ送らない。この文の食い違いは tech-debt の行に書き、コードは変えていない

### Surfaces checked for drift

| Surface | 結果 |
|---------|------|
| `ralph org director start --help` | 変更なし。`--config` の文(運ぶこと、stop の打ち方に付けないこと)は、README と `/org` skill に足した文と合う。verify の C2-6 が実機の出力で確かめている |
| `/director` skill の「立てる」の「start の設定」の箇条 | 変更なし。`--config` なしの規則(main worktree の ralph.toml)を書く箇条で、`--config` を運ぶことは「始める」の 3 が書く。依頼は 1 文だったので、そこに足した |
| `README.md`(動詞の表 124 行、`ralph status` の行 125) | 変更なし。124 行は `director start/stop`、125 行は headless の状態を書き、cycle 2 で変わった `--config` の運び方には触れない |
| `AGENTS.md`、`CLAUDE.md`、`.claude/rules/ralph/`、`docs/quality/`、`docs/architecture/repo-map.md` | 変更なし。director start の `--config` の運び方を書く文がない。cycle 1 で直した `watchgate.go` の句と見張りの関門の文は、cycle 2 の変更に触れない |
| `{{CONFIG}}`(改名前の名前) | `docs/reports/`、`docs/evidence/`、`docs/plans/` を除く Markdown、Go、toml、shell に 0 件。残るのは計画の Progress の 3243c499 の時点の記述(verify の C2-V1)で、計画は触っていない |
| `templates/base/ralph.toml` の `[org.director]` のコメント | 変更なし。`enabled` が escalate の動き(`true` は inbox に記録して director か ack の期限、`false` は人へすぐ)を書く規則は、新しい行の「escalate は inbox に記録する」と合う |
| `docs/tech-debt/README.md` の既存の行(245〜256) | `--config` を挙げる行は 0。256 行は 4ef33195 の縮め方のままで、挙げるテスト名は HEAD にある。F-8 の行は変更なし |
| `docs/recipes/` | 変更なし。director start を扱う記述がない |
| cycle 1 の節 | そのまま。「README 258 行は変更なし」などの記述は、cycle 1 の時点の判断として残した |

### Left open

- 計画の Progress の行(verify の C2-V1)と、AC11 の文と雛形の stop の行の言い方(C2-V2)は、計画をオーケストレーターが持つので触っていない
- テストの名前 `TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig` の「Every」(C2-V5)は Go のテストなので、この run では触っていない
- start の断りの文の「`ralph org escalate` はすでに人へ送る」の言い方は Go のコードなので、直していない。tech-debt の 258 行に書いた
- `templates/base/ralph.toml` の reviewer の注(guarded の reviewer が permission prompt で止まる、と言う文)は、cycle 1 の Left open のまま
- test の cycle 2 の Test gaps 3〜5(本物の herdr と agent が `--config` つきの雛形のとおりに打つこと、台帳が main worktree のもので director が別の場所で動くときの相対パス、存在しないファイルを指す `--config`)は、文書の話ではないので行にしていない。test の Test gaps 6 は cycle 1 の 1〜4 を閉じた旨
- `docs/evidence/test-2026-10-15-org-director-start.md` と `verify-2026-10-15-org-director-start.md` の `/Users/` は 0 件で、`$HOME` の置き換えの項目は満たしている
- 実機の動きは確かめていない。コードを変えていないので、verify と test の実機の結果はそのまま有効

### Verification

commit の前に、worktree のルートで打った。

- `./scripts/check-skill-sync.sh`: `[ok] check-skill-sync: 14 skill(s) in lock-step`
- `./scripts/check-sync.sh`: `IDENTICAL: 169`、`DRIFTED: 0`、`ROOT_ONLY: 0`、`TEMPLATE_ONLY: 11`、`KNOWN_DIFF: 5`、`PASS: all files in sync.`
- `go test ./internal/config/... ./internal/scaffold/... ./internal/org/ -run 'Skill|Sync|Defaults' -count=1`: `ok`(config 1.093s、scaffold `[no tests to run]`、org 1.593s)
- `./scripts/run-static-verify.sh`: rc 0、`==> All verifiers passed.`。「tech-debt README plan references」は OK、`gofmt: ok`、`0 issues.`、`secret-scan-branch: scanned f21df66c..6556bce5 against origin/main: clean`。出力 425 行は scratchpad に落として読んだ。wrapper のログは gitignore の対象で、コミットしていない
- `git diff --check`: 出力なし。変えたファイルの U+FFFD は 0 件
- `git diff --numstat`(report と insight event を除く 11 ファイル): +32/-14。`docs/tech-debt/README.md` は +2/-0、ほかは skill の 4 面ごと、README、仕様の 1 行ずつ
- 変えた tech-debt の行は、`git show HEAD:docs/tech-debt/README.md` と比べて、増えた行が 257 と 258 の 2 行だけ(既存の行に変更なし)。2 行とも 5 列(列を区切る `|` で 7 片)、backtick は列ごとに偶数、`file:line` は 0
- skill の 4 面の md5: `/director` は `52e76af939fd551478ee8aa969700d2e`、`/org` は `c1bb29c022619e5bf9b4572309ca7119` で、それぞれ 4 面が同じ
- `./scripts/plan-visual.sh digest docs/plans/active/2026-10-15-org-director-start.md`: `f65d7c59746c`(run の前も後も同じ)
- `./scripts/insights-append.sh --slug org-director-start --flow standard --phase sync_docs --verdict pass --cycle auto --source skill`(slug は日付なし): rc 0。`git status --short` は既存の `docs/insights/events/2026-10-15-org-director-start.jsonl` を `M` と示し、新しい `??` のファイルはない。足した行は cycle 2、verdict pass

### Verdict

Verdict: complete。依頼の 3 項目を直した。C2-V4 はコードで事実と確かめ、skill に 1 文、tech-debt に 1 行を足した。K15 と K16 は tech-debt の 1 行にした。静的検査と同期の検査は通った。コードとテストは触っていないので、verify と test の結果は変わらない。確かめていないのは、足した文が本物の director の動きと合うことで、コードと verify の evidence(C2-5)から書き、実機を取り直してはいない。
