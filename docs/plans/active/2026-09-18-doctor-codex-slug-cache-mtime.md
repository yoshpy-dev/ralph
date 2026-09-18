# doctor-codex-slug-cache-mtime

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-18
- Related request: `ralph doctor` の「Org codex model slugs」Check はローカルの `models_cache.json` を読むだけで、その cache がいつ書かれたものかを出さない。warn が出た瞬間の出力に鮮度が見えないと、数日前の cache に基づく warn を運用者が真に受けて seed-once の `ralph.toml` からスラッグを誤って外すリスクが残る(2026-09-17 の一時的消失の事例)。#156 の文書化では散文で注意しているが、注意が効くべき場所は出力そのもの(issue #159、PR #160 の self-review Follow-ups 起点)
- Related issue: 159
- Type: feat
- Branch: feat/doctor-codex-slug-cache-mtime

## Objective

`checkCodexModelSlugs`(`internal/cli/doctor_codex_models.go`)の warn / pass の Detail に cache ファイルの書き込み時刻(mtime、UTC RFC3339)と経過時間を含め、mtime が 24 時間より古い場合は「cache may be stale; launch codex once to refresh, then re-run」の注記を添える。Status は変えない(warn は warn のまま、best-effort な Check の性質を維持)。テストは `os.Chtimes` で mtime を操作して、mtime が Detail に出ること・古い cache で注記が付き新しい cache では付かないことを固定する。あわせて `/org` skill(4 面)と org runtime spec の「更新も鮮度確認もしない」という記述を、実装に合わせて「更新はしないが、cache の書き込み時刻と stale 注記を Detail に出す」に更新する。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | mtime の取得と Detail への付与 | `internal/cli/doctor_codex_models.go` | `os.ReadFile` + 別 `os.Stat` ではなく、`os.Open` した **1 つのハンドル** で `f.Stat()`(読む前)→ `io.ReadAll(f)` → `f.Stat()`(読んだ後)の順に取り、前後の `ModTime` / `Size` が一致したときだけその mtime を採用する。不一致(読んでいる間に codex が cache を書き換えた)なら内容の判定はそのまま行い、Detail には ` (cache changed while reading; freshness unknown — re-run)` を付けて stale 注記は付けない。一致した場合は `cacheAgeNote := fmt.Sprintf(" (cache written %s, %s ago)", mtime.UTC().Format(time.RFC3339), humanAge)` を warn / pass 両方の Detail 末尾に付ける。`humanAge` は分未満切り捨てで `Nm` / `Nh` / `Nd`(小さなヘルパー `formatCacheAge(d time.Duration) string`)。`f.Stat()` が失敗した場合は mtime 節を省き、Status も Detail の本体も変えない(best-effort)。読み取りとメタデータ取得の間に差し込むための test seam として、パッケージ内非公開の `var codexCacheBetweenReadAndStat func()`(既定 nil、テストだけが設定)を `io.ReadAll` の直後・2 回目の `f.Stat()` の直前で呼ぶ |
| 2 | stale 注記 | `internal/cli/doctor_codex_models.go` | `time.Since(mtime) > codexCacheStaleAfter`(定数 `24 * time.Hour`)なら Detail 末尾に `; cache may be stale — launch codex once to refresh, then re-run` を追加。warn / pass どちらにも付ける(pass でも古い cache は「存在の証拠として弱い」ため)。Status は不変。doc comment の「Six deterministic outcomes」節に「warn / pass は cache の mtime と、24h 超の stale 注記を伴う」を追記 |
| 3 | テスト | `internal/cli/doctor_org_test.go` | (a) `TestCheckCodexModelSlugs_DetailCarriesExactCacheMtime`: 全スラッグありの cache に `os.Chtimes` で既知の秒揃え mtime(now − 5 分、`Truncate(time.Second)`)を設定 → pass、Detail に `cache written ` + その mtime の `UTC().Format(time.RFC3339)` を **完全一致の部分文字列** として含み、stale 注記を含まない。(b) `TestCheckCodexModelSlugs_StaleCache_WarnCarriesExactMtimeAndStaleNote`: 欠落スラッグありの cache に `os.Chtimes`(now − 48h、秒揃え)→ warn、Detail に同じく完全一致の RFC3339、`cache may be stale`、`2d ago` を含む。(c) `TestCheckCodexModelSlugs_StaleCache_PassAlsoCarriesStaleNote`: 全スラッグありの cache を 48h 古くして pass + stale 注記。(d) `TestFormatCacheAge` テーブル: −1m→`0m`、30s→`0m`、90m→`1h`、47h→`1d`、49h→`2d`。(e) `TestCheckCodexModelSlugs_CacheChangedWhileReading_FreshnessUnknown`: 欠落スラッグありの cache を 48h 古くし、`codexCacheBetweenReadAndStat` で cache を書き直す(`writeCodexModelsCache` で全スラッグあり、mtime は now)→ 判定は読んだ内容(warn、欠落スラッグ名)のまま、Detail に `freshness unknown` を含み `cache may be stale` と `cache written` を含まない。`t.Cleanup` で seam を nil に戻す。既存 8 テストは Detail の先頭一致で書かれているため変更不要(付与は末尾) |
| 4 | 文書更新 | `.claude/skills/org/SKILL.md`(+ `.agents/`、`templates/base/` の 3 ミラー) | 「既定の model_pool」節の段落中「Check はローカルの cache を読むだけで更新も鮮度確認もしないので」→「Check はローカルの cache を読むだけで更新はしない(cache の書き込み時刻を Detail に出し、24 時間より古ければ stale 注記が付く)ので」。以降の「codex を一度起動して cache を更新してから再確認」「warn 1 回で外さない」は据え置き |
| 5 | 文書更新 | `docs/specs/2026-08-01-org-runtime.md` | 「運用ノート」(d) の「`checkCodexModelSlugs` はローカルの `models_cache.json` を読むだけで、更新も鮮度確認もしない」→「…読むだけで更新はしない(書き込み時刻と 24h 超の stale 注記は Detail に出る、issue #159)」。観測手順の「`stat` で cache の mtime が観測時刻に更新されたことを確認」は「`ralph doctor` の Detail に出る `cache written` が観測時刻に更新されていることを確認(`stat` でも可)」に置き換え。記録項目の「cache mtime」はそのまま |

## Non-goals

- cache の自動更新(doctor から codex を起動しない。`--probe-models` が既にその役)
- stale 時に Status を変えること(warn → 別ステータス、pass → warn 等)。best-effort な Check の性質を保つ
- 閾値の設定化(`ralph.toml` / 環境変数)。24h の定数で始め、必要になったら別 issue
- 他の doctor Check への mtime 表示の横展開
- `docs/tech-debt/README.md` の変更(該当行なし)

## Assumptions

- `os.Stat().ModTime()` は macOS / Linux の CI(ubuntu)で `os.Chtimes` の設定値を返す。完全一致の assert は `Truncate(time.Second)` した値を `os.Chtimes` に渡し、RFC3339(秒精度)で比較するので、ファイルシステムのナノ秒精度差は影響しない
- 1 ハンドル方式で、codex が rename で cache を差し替えた場合はハンドルが旧 inode を指し続けるため内容と mtime は旧側で一貫する。in-place 書き換えの場合だけ前後の `Stat` 不一致で検知する(Codex plan advisory MEDIUM-1 を採用)
- 経過時間の計算は実時刻(`time.Now()`)で行う。テストは「書きたて(< 24h)」と「48h 前」の 2 点だけを使い、境界値(ちょうど 24h)はテストしない。clock 注入は不要
- 既存テストは Detail の先頭部分(`2 codex model_pool slug(s)`、`not found at <path>` 等)だけを検査しており、末尾への付与で壊れない(2026-09-18 に `grep 'r.Detail'` で確認)
- 4 面ミラーは `scripts/sync-skills.sh` + cp、`check-skill-sync.sh` / `check-sync.sh` で一致を検証する(#154/#156 と同じ手順)

## Affected areas

- `internal/cli/doctor_codex_models.go`
- `internal/cli/doctor_org_test.go`
- `.claude/skills/org/SKILL.md` と 3 ミラー
- `docs/specs/2026-08-01-org-runtime.md`

## Design decisions

Critical forks: None(閾値・Status 不変・Stat 失敗時の扱いはいずれも issue 本文と best-effort の既定で決まる)。

既定として採った選択:

- stale 注記は pass にも付ける。古い cache で pass しても「今も存在する」証拠としては弱く、#156 の観測手順(更新済み cache でのみ観測を成立とみなす)と整合させる
- 経過時間の表示は `Nm` / `Nh` / `Nd` の粗い粒度。運用者が見るのは「数分前か、数日前か」で、秒精度は不要
- `f.Stat()` 失敗は黙って mtime 節を省く。Check の本題(スラッグの有無)には影響しない
- test seam(`codexCacheBetweenReadAndStat`)は非公開のパッケージ変数 1 つ。`internal/cli` に clock 抽象や fs 抽象がなく、読み取り途中の書き換えを決定的に再現するにはこれが最小(Codex plan advisory MEDIUM-1 の「deterministic test」要求)
- 表示時刻の正しさは既知 mtime の完全一致で固定する(Codex plan advisory MEDIUM-2 を採用)
- テストは実時刻 + `os.Chtimes` で行い、clock 注入は入れない(`internal/cli` に既存の clock 抽象がなく、2 点判定で十分)

## Acceptance criteria

- [x] AC-1: warn と pass の Detail に `cache written <RFC3339 UTC>, <age> ago` が含まれ、その時刻は cache ファイルの mtime そのものである。テスト (a)(b) が `os.Chtimes` で設定した既知の mtime の RFC3339 表記を完全一致で assert して pass(`time.Now()` を表示する実装では fail する)
- [x] AC-2: mtime が 24h より古い cache では Detail に `cache may be stale` が付き、書きたての cache では付かない。warn / pass の両方で成立。テスト (b)(c) が pass、(a) が stale 注記の不在を assert
- [x] AC-3: `formatCacheAge` のテーブルテスト (d) が pass。Status は既存 8 テストのとおり不変(全 pass)
- [x] AC-4: 読み取り前後の `f.Stat()` が一致しない場合、Detail に `freshness unknown` が付き、stale 注記と `cache written` は付かず、スラッグ判定は読んだ内容に基づく。テスト (e) が test seam 経由で pass。`f.Stat()` 失敗時は mtime 節なしの従来 Detail になる(コード上の分岐として存在し、reviewer が確認。単体テストでは再現しないため Known gaps に記載)
- [x] AC-5: `/org` skill の段落が「鮮度確認もしない」を含まず(`grep -c '鮮度確認もしない' .claude/skills/org/SKILL.md` が 0)、`stale 注記` を含む。4 面 `cmp` 一致、`./scripts/check-skill-sync.sh` / `./scripts/check-sync.sh` pass
- [x] AC-6: spec (d) が「鮮度確認もしない」を含まず、`cache written` または `stale 注記` と issue #159 への参照を含む
- [ ] AC-7: `go test ./internal/cli/... -count=1` と `./scripts/run-verify.sh` が green。PR 本文は `Closes #159`

## Implementation outline

1. Slice A(Go、implementer 委譲): 項目 1〜3。`gofmt -l internal/`、`go vet ./internal/cli/...`、`go test ./internal/cli/ -run 'TestCheckCodexModelSlugs|TestFormatCacheAge' -count=1 -v`、`go test ./internal/cli/... -count=1`、`./scripts/run-verify.sh` → commit `feat: surface cache mtime and a stale note in the doctor codex slug check`
2. Slice B(docs、inline): 項目 4〜5。`sync-skills.sh` → cp → `check-skill-sync.sh` / `check-sync.sh` → commit `docs: describe the doctor slug check's cache mtime and stale note`
3. `./scripts/run-verify.sh` → post-implementation pipeline → `/cross-review` → `/pr`(`Closes #159`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`(gofmt / go vet / golangci-lint / staticcheck)、`check-skill-sync.sh`、`check-sync.sh`
- Spec compliance criteria to confirm: AC-1〜AC-7 の grep / cmp。Detail の文字列が doc comment の outcomes 記述と一致すること
- Documentation drift to check: skill 段落と spec (d) が新しい Detail の内容(mtime 表示、24h、stale 文言)と一致すること。#156 で書いた観測手順が新しい出力で簡略化されていること(`stat` は任意に)
- Evidence to capture: grep / cmp / sync スクリプトの出力

## Test plan

- Unit tests: `internal/cli` の `TestCheckCodexModelSlugs_*`(既存 8 + 新規 4)と `TestFormatCacheAge`
- Integration tests: なし(`ralph doctor` の CLI 経路は既存の doctor テストで担保)
- Regression tests: `go test ./internal/... -count=1`
- Edge cases: (1) `os.Chtimes` で未来の mtime を設定した場合、経過時間が負になる → `formatCacheAge` は負値を `0m` に丸める(テーブルに 1 ケース追加)。(2) Windows では `os.Chtimes` は動くのでスキップ不要。(3) cache が空の JSON(`{"models": []}`)で全スラッグ欠落 + stale → warn + stale 注記(既存 SomeMissing の派生、(b) で兼ねる)
- Evidence to capture: `go test -count=1 -v` の対象テスト出力、`docs/reports/test-*.md`

## Risks and mitigations

- Detail の末尾追加が既存テストの完全一致 assert を壊す → 確認済み: 完全一致は `no codex entries in model_pool`(codex エントリなし、mtime 節が付かない経路)のみ
- 24h の閾値が環境によっては厳しすぎる / 緩すぎる → Status は変えず注記のみなので誤検知の害は小さい。設定化は Non-goals
- ミラー取り残し → sync + 2 つの sync チェック

## Rollout or rollback notes

`ralph doctor` の出力文言の追加のみ。下流へは次回 release で配布(バイナリ)。skill の文言は core として `ralph upgrade` で届く。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: Codex plan advisory(codex-cli 0.154.0、`</dev/null` 付き)が MEDIUM 2 件(read/stat の分離で内容と鮮度が別バージョンになりうる、表示時刻が mtime である証明がない)を報告。両方採用し Scope 1・3、Assumptions、Design decisions、AC-1・AC-4 を改訂
- 2026-09-18 work: `./scripts/branch-name.sh from-plan` の issue 番号付きブランチ名ではなく、`/plan` が登録した worktree state の `feat/doctor-codex-slug-cache-mtime` を維持(手順 2d)
- 2026-09-18 work: Slice B(docs)を index 衝突回避のため Slice A より先に inline で実施 = fd1be42(skill 4 面 + spec (d)、sync ゲート pass)。Slice A = 3b0fdf4(implementer 委譲)。逸脱 1 件: golangci-lint の errcheck に合わせ `defer func() { _ = f.Close() }()`(`internal/insights` の既存慣例)。orchestrator 側で diff・18 テスト pass・AC ゲート・verify スクリプト exit 0(evidence: `docs/evidence/verify-2026-09-18-051002.log`、gitignored、lint 0 issues)を確認。AC-7 の `Closes #159` は `/pr` で満たす
- 2026-09-18 self-review(cycle 1): Merge 推奨、MEDIUM 1(M1: spec の観測手順が `cache written` を観測時刻と突き合わせるよう書いているが表示は UTC で JST と 9 時間ずれる)+ LOW 4(L2: `codexCacheFreshnessClause` の `now` 注入が未活用で 24h 境界と Stat 失敗の空文字列が未固定、L3: doc comment が in-place 書き換え検知を無条件に主張、L4: seam が最初の Stat 失敗の return より前に走る、L5: `24 * time.Hour` が閾値と表示粒度の 2 意味で重複)。全件 inline で修正 = 8f3f72b(spec に UTC / `<age> ago` の補足、skill 4 面に「UTC で」、`TestCodexCacheFreshnessClause` テーブル 5 行、doc の機構明記、seam の配置移動、`hoursPerDay` 定数)。verify スクリプト exit 0、lint 0 issues(evidence: `docs/evidence/verify-2026-09-18-052324.log`)。AC-4 の Known gap(Stat 失敗の空文字列)は L2 のテーブルで固定されたので解消
- 2026-09-18 plan drift: Design decisions の「clock 注入は入れない」は L2 で撤回。`codexCacheFreshnessClause` は `now` を引数に取っており、テーブルテストがそれを使って 24h 境界を固定する。`checkCodexModelSlugs` 経由のテストは引き続き実時刻 + `os.Chtimes`

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [x] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
