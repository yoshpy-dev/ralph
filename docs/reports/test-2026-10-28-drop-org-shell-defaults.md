# Test report: drop-org-shell-defaults

- Date: 2026-10-28(UTC 10:37〜13:21)
- Plan: `docs/plans/active/2026-10-28-drop-org-shell-defaults.md`(Test plan、AC2、AC4 のテストの部分)
- Tester: tester サブエージェント(Claude Opus 5.5)。pipeline の cycle 1
- Scope: HEAD ecf31b75(base 15e301c8)。behavioral test だけを実行した(静的解析は /verify の担当)。この branch が変えた Go のテスト以外のコードは `internal/config/config.go` のコメントだけで、`internal/org` には変更がない(`git diff 15e301c8..HEAD -- internal/org` が空)。環境は macOS(Darwin 25.5.0、arm64)、go1.27.3、git 2.50.1
- Evidence: `docs/evidence/test-2026-10-28-drop-org-shell-defaults.log`(名指しの実行、AC2 の変異 2 つの出力、端の場合と mutant の出力、使い捨ての probe の中身、`run-test.sh` の 2 回分の出力)。gitignore の対象で、コミットしない
- 追跡しているファイルは、この報告と insight event のほかは変えていない。AC2 の確認では worktree の `templates/base/ralph.toml` を一時的に書き換え、そのたびに `git checkout -- templates/base/ralph.toml` で戻して `git status --short` が空になることを確かめた。2 回の変異はどちらも `run-test.sh` を始める前に済ませた。端の場合と mutant は scratchpad の `git archive HEAD` の写しで走らせた

## Test execution

| # | Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `go test ./internal/config/ -run 'TestDefaultsLockStep\|TestOrgTemplateDrift\|TestLoad_TemplateRalphToml' -v -count=1` | トップレベル 3(サブテスト 8) | 3(8) | 0 | 0 | ok、0.501 秒 |
| 2 | `sh tests/test-ralph-config.sh` | 19 | 19 | 0 | 0 | rc 0 |
| 3 | AC2 の変異 2 つ(下の「AC2 の確認」) | 2 | 期待どおり 2 本とも落ちた | - | - | 1 回 1 秒未満 |
| 4 | 端の場合 4 つと mutant 6 つ(下の「端の場合」「mutant」) | 4 + 12 | 期待どおり | - | - | 1 回 1〜2 秒 |
| 5 | `./scripts/run-test.sh` 1 回目(scope は `full fallback (unclassified:scripts/ralph-config.sh)`、pack は golang) | shell 45 本 + Go 9 パッケージ | shell 45 本(4,768 件)、Go 8 パッケージ | Go 1(`internal/org` の `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`) | 0 | rc 1、2,914 秒(10:40:31〜11:29:05) |
| 6 | `./scripts/run-test.sh` 2 回目(5 の失敗のあとの打ち直し、同じ HEAD) | shell 45 本 + Go 9 パッケージ | shell 45 本(4,768 件)、Go 9 パッケージ | 0 | 0 | rc 0、2,985 秒(11:30:49〜12:20:34) |

- 1 の内訳は、`TestLoad_TemplateRalphToml`、`TestDefaultsLockStep`(サブテスト `org` と `shell_and_skills`)、`TestOrgTemplateDrift_DetectsChangedAndMissingKeys`(6 件: 実テンプレートに差がない、`max_seats` の変更、`auto_merge` の削除、`codex_verified` の削除、`wake` の削除、`model_pool` の 2 件の入れ替え)。すべて PASS
- 5 の Go はどのパッケージも `(cached)` ではなく実際に走った(`internal/cli` 879.5 秒、`internal/org` 1,196.9 秒、`internal/config` 4.5 秒、ほかは 1.6〜9.9 秒)
- 6 の Go で走ったのは `internal/config`(1.1 秒)と `internal/org`(1,044.6 秒)だけで、残りの 7 パッケージは 5 で同じ入力のまま通った結果の `(cached)` だった。5 と 6 は同じ HEAD、同じ worktree で、あいだに追跡しているファイルの変更はない
- 5 と 6 の shell の部分は、45 本すべての要約行(`PASS: n`、`FAIL: 0`、`n passed, 0 failed` など)が 1 行も違わなかった(`diff` で確かめた)。45 本は `tests/test-*.sh` の数と同じ。`SKIP:` の行はどれも 0
- 1 分の load average は、5 の始まりが 20.8、途中の 11:11 が 47〜73、終わりが 15.6。6 の始まりが 29.8、11:51 に 112.0 まで上がり、終わりが 61.3。ほかのセッションの処理と重なっていた

## AC2 の確認

worktree の `templates/base/ralph.toml` を書き換えて `go test ./internal/config/` の照合が落ちることを確かめ、そのつど戻した。

`max_seats = 5` を `max_seats = 6` にして `go test ./internal/config/ -run TestDefaultsLockStep -count=1` を実行すると、rc 1 で次のように落ちた。

```
--- FAIL: TestDefaultsLockStep (0.00s)
    --- FAIL: TestDefaultsLockStep/org (0.00s)
        defaults_sync_test.go:220: lock-step mismatch between templates/base/ralph.toml and config.Default():
              org.max_seats: template = 6, Default() = 5
FAIL
FAIL	github.com/yoshpy-dev/ralph/internal/config	0.523s
```

続けて、既定がゼロ値のキーの例として `auto_merge = false` の行を消し、`-run 'TestDefaultsLockStep|TestLoad_TemplateRalphToml' -v` を実行すると、rc 1 で次のように落ちた。

```
=== RUN   TestDefaultsLockStep/org
    defaults_sync_test.go:220: lock-step mismatch between templates/base/ralph.toml and config.Default():
          org.director.auto_merge: missing from the template (Default() = false)
--- FAIL: TestDefaultsLockStep (0.00s)
    --- FAIL: TestDefaultsLockStep/org (0.00s)
    --- PASS: TestDefaultsLockStep/shell_and_skills (0.00s)
```

- 2 つ目で、go-toml/v2 がキーのない pointer field を nil のまま残すことを実行で確かめた。verify の報告が「実行では確かめていない」とした点にあたる
- 同じ実行で `TestLoad_TemplateRalphToml` は PASS した。この branch で畳んだ `auto_merge` の明示の確認は、いまは `TestDefaultsLockStep/org` だけが受け持つ。計画の Test plan のとおりで、照合が弱まったわけではない
- どちらのあとも `git checkout -- templates/base/ralph.toml` で戻し、`git status --short` は空だった

## 端の場合

計画の Test plan の Edge cases を、`git archive HEAD` の写しからファイルを消して確かめた。

| 写し | `TestDefaultsLockStep/org` | `TestDefaultsLockStep/shell_and_skills` | 結果 |
| --- | --- | --- | --- |
| `scripts/ralph-config.sh` を消す | PASS | SKIP(`scripts/ralph-config.sh not found … (vendored repo?)`) | ok |
| `.claude/skills` を消す | PASS | SKIP(`.claude/skills not found …`) | ok |
| `templates/base/ralph.toml` を消す | SKIP(`templates/base/ralph.toml not found …`) | PASS | ok |
| `.claude/skills` を消し、`RALPH_ORG_AGMSG_HOME` の既定を `~/.agentsX/skills/agmsg` に変える | PASS | FAIL(`lock-step mismatch for org.agmsg_home`、そのあとに skill の SKIP の行) | rc 1 |

- 4 行目で、`.claude/skills` がないときに飛ぶのは SKILL.md の照合だけで、`org.agmsg_home` の shell と `Default()` の照合は走ることを確かめた(2 行目の SKIP の表示だけでは、どこで飛んだかが分からないため)

## mutant

`orgTemplateDrift`(`internal/config/defaults_sync_test.go:148-175`)の配線を写しの中で変え、`go test ./internal/config/ -count=1` を実行した。置き換えは、写しの中に 1 回だけ現れる字面を perl で変えた(0 回か 2 回以上なら止める)。対照(変えない写し)は通った。

| mutant | 変更 | コミット済みのテスト | 使い捨ての probe |
| --- | --- | --- | --- |
| M1 | `org.deadman_minutes` を `d.MaxOrgs` と比べる(どちらも既定 10) | 生き残る(rc 0) | 落ちる |
| M2 | `org.permissions.codex_verified` を `d.Director.AutoMerge` と比べる(どちらも既定 false) | 生き残る(rc 0) | 落ちる |
| M3 | `org.watchdog.watcher_enabled` を `d.Director.Enabled` と比べる(どちらも既定 true) | 生き残る(rc 0) | 落ちる |
| M4 | `org.max_seats` を `d.MaxOrgs` と比べる(5 と 10) | 落ちる(`TestDefaultsLockStep/org` ほか) | 落ちる |
| M5 | `org.approval.mode` の `valueDrift` の行を消す | 生き残る(rc 0) | 落ちる |
| M6 | `listDrift` を `slices.Equal` から長さの比較に変える | 落ちる(`two_model_pool_entries_swapped`) | 落ちる |

- 使い捨ての probe(`TestProbe_PerKeyDefaultPerturbation`、コミットしていない。中身は evidence にある)は、19 キーのそれぞれについて `Default()` のそのキーだけを別の値に変え、`orgTemplateDrift` がそのキーを名指しする 1 行だけを返すことを確かめる。HEAD では通り、M1〜M6 のすべてで落ちた
- M1〜M3 と M5 は、いまのテンプレートと `Default()` が一致しているかぎり害はない。害が出るのは、あとで `Default()` の片方だけを変え、テンプレートを直し忘れたときで、照合がそれを見逃す。#274 が `codex_verified` と `auto_merge` の既定を変える予定なので、M2 の形がその場面にあたる。下の Test gaps に書いた

## Coverage

- Statement: 測っていない(変わったのはテストのコードとシェルの代入の削除で、製品の Go のコードはコメントだけ)
- Branch: `orgTemplateDrift` の分岐(nil、値の違い、一致)と `listDrift` の分岐(nil、並びの違い、一致)は、1 の 6 件と AC2 の変異 2 つで全部通った
- Function: `orgTemplateDrift`、`valueDrift`、`listDrift`、`TestDefaultsLockStep` の 2 つのサブテストのすべての skip の経路(端の場合の表)を実行した
- Notes: 19 キーのそれぞれの配線(どのキーをどの `Default()` の field と比べるか)を落とせるテストはない。mutant の表のとおり、既定の値が同じキーどうしの取り違えと、1 キーの比較の削除が生き残る

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime`(`internal/org/director_cmd_test.go:310`、5 だけ) | `stopDirectorProcess = {outcome:ended signalled:true note:process=ended after SIGTERM err:<nil>}, want not ended within 300ms`(0.58 秒) | 既知の負荷による競合。テストが 100 ms 待ってから SIGTERM を送るが、負荷が高いと `sh -c "trap '' TERM; exec sleep 30"` の `trap` より先に SIGTERM が届く。`docs/tech-debt/README.md:285` の行に記録がある。この branch は `internal/org` を変えておらず、`internal/org` が import する `internal/config` の変更はコメントだけ | この branch では直さない。6 の打ち直しで `internal/org` は通った(1,044.6 秒)。tech-debt の行が数える 5 回目の発生で、その行の発火条件(次の失敗)がまた当たった。行が挙げる直し方(決まった時間待つ代わりに、`ps` で `sleep` に入ったことを確かめてから SIGTERM を送る)を別の作業で入れるか、行に 5 回目を書き足して持ち越すかは lead が決める |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| テンプレートから既定がゼロ値のキー(`auto_merge`、`codex_verified`)が消えても、`Load` で読むと `Default()` の値が残って照合が素通りする(計画の D1、R2) | 直っている | AC2 の 2 つ目の変異(実テンプレート)と、1 の削除の 2 件 |
| `scripts/ralph-config.sh` がないと `[org]` の照合まで飛ぶ(計画の D3) | 直っている | 端の場合の 1 行目(`org` は PASS) |
| `TestLoad_TemplateRalphToml` から畳んだ `auto_merge` と `wake` の確認 | 照合が受け持つ | AC2 の 2 つ目の変異で `TestLoad_TemplateRalphToml` は PASS、`TestDefaultsLockStep/org` が FAIL。1 の `a_removed_wake` |
| `tests/test-ralph-config.sh` の既存の 19 件 | 通る | 2、5、6 |

## Test gaps

- 19 キーの配線を固定するテストがない(mutant の M1〜M3、M5)。最小の追加は、使い捨ての probe と同じ形の表のテストで、`Default()` の 1 キーだけを変えて、そのキーを名指しする 1 行だけが返ることを 19 キーそれぞれで確かめるもの。1 キーの比較を消す形(M5)もこの表で落ちる。self-review の L4(19 という数がコメントにしかない)とは別の穴で、L4 は PR の Known gaps に入る予定
- `[doctor]` の既定値は照合に入っていない(計画の Non-goals、#272)
- この報告で走らせたのは macOS の go1.27.3 だけで、CI の Linux では走らせていない

## Verdict

- Pass: `go test ./internal/config/` の対象 3 本(サブテスト 8)、`tests/test-ralph-config.sh` 19/19、AC2 の変異 2 つがどちらもキー名つきで落ちること、端の場合 4 つ、`./scripts/run-test.sh` の 2 回目(rc 0、shell 45 本 4,768 件、Go 9 パッケージ)
- Fail: なし。1 回目の `run-test.sh` の `internal/org` の失敗は、tech-debt に記録のある既知の不安定なテストで、2 回目に通った
- Blocked: なし
- Verdict: pass
