# Sync-docs report: guard-bypass-and-hygiene

- Date: 2026-10-07
- Plan: docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md
- Agent: doc-maintainer subagent
- Branch: fix/guard-bypass-and-hygiene(base 2a22ba78、編集前の HEAD 37514051)
- 前段の report: `self-review-2026-10-07-guard-bypass-and-hygiene.md`(Merge 可、M-1 と L-1〜L-5)、`verify-2026-10-07-guard-bypass-and-hygiene.md`(pass、V-1〜V-9)、`test-2026-10-07-guard-bypass-and-hygiene.md`(pass、Test gaps 1〜9)

## Summary

verify の文書のずれ V-1〜V-7 と V-9 を直し、self-review の tech-debt 提案 2 行と test の Test gaps 1 行を `docs/tech-debt/README.md` に足した。V-8(AC4 の直し方の表示が plan の文言より細かい)は plan の本文が digest の対象なので直さず、Progress checklist に 1 行のメモを足した。`.codex/README.md`、`git-commit-strategy.md`、`quality-gates.md`、`definition-of-done.md` は新しい guard の挙動と食い違う記述がなく、変更していない。コードと tests は変えていない。plan の本文は変えておらず、digest は `c07adf402bdb` のまま(`./scripts/plan-visual.sh digest` で確認)。

## Files changed

| File | Change |
|------|--------|
| `docs/insights/README.md`、`templates/base/docs/insights/README.md` | insight event を書く skill の一覧に `/sync-docs` を足した。冒頭の段落(`:4`)と "Appending events"(`:118-119`)の 2 か所。root と template は `cmp` で一致。V-1 |
| `docs/tech-debt/README.md` | 下の表のとおり。127 行目(Codex の `ask`)、124 行目(コミットメッセージの guard)、128・129 行目(insight event)、157 行目(shellcheck の対象)を直し、末尾に 3 行足した。V-2〜V-6 |
| `.claude/hooks/pre_bash_guard.sh`、`templates/base/.claude/hooks/pre_bash_guard.sh` | 書き込み先の説明コメントに 2 行足した。sed fallback ではタブも 2 文字の `\t` のままなので、リダイレクトや `tee` と書き込み先の間のタブは見ない(jq の経路は見る)。コメントだけで、ロジックは変えていない。root と template は `cmp` で一致。V-7 |
| `scripts/verify.local.sh` | 冒頭コメントの test モードの説明を「every tests/test-*.sh(実行権限がないものは working tree でも index でも失敗として数える)」にした。static モードの行はこの PR の S4 で直っていて、tech-debt の plan 参照の検査を含んでいる。V-9 |
| `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` | `## Progress checklist` だけ更新した。Review / Verification / Test artifact created を `[x]` にし、verify と test と sync-docs の結果を 3 行のメモにした(V-8 の差はここに書いた)。本文は変えていない |
| `docs/reports/sync-docs-2026-10-07-guard-bypass-and-hygiene.md` | この report |
| `docs/insights/events/2026-10-06-guard-bypass-and-hygiene.jsonl` | `/sync-docs` の insight event を 1 行追記した(`--phase sync_docs --verdict pass --cycle auto`、cycle は 1)。ファイル名の日付は、このスラッグの既存ファイルが UTC の 10-06 で作られているため。この PR が足した手順の最初の実行 |

### `docs/tech-debt/README.md` の変更

| 行 | 変更 |
|----|------|
| 124(コミットメッセージの guard の誤検知) | Why deferred に、2026-10-07 の guard-bypass-and-hygiene も `pre_bash_guard.sh` に触れたが、この行の誤検知は plan の Non-goals に入れて意図的に残した、と足した。Trigger を「コミットメッセージの規則を直す次のタッチ」に書き直し、今回の接触は数えないと書いた。HEREDOC の形が今も deny になることは、新しい guard に payload を流して確かめた。V-4 |
| 127(Codex の `ask`) | debt item に新しい証拠を足した。codex-cli 0.160.0 の PreToolUse の入力スキーマが `permission_mode` を必須とし、値に `bypassPermissions` を含むので、この PR 以降は Codex でも bypass で `ask` を返さない。同じ実行ファイルに `unsupported permissionDecision:ask` の文字列があり、Codex はもともと `ask` を効かせていなかったとみられる(おそらく。未確認)。Codex が bypass 以外のモードで `ask` をどう扱うかと、`bypassPermissions` に当たる Codex の設定は、未確認のまま残した。Trigger は、`ask` の live-fire probe、Codex の `PreToolUse` 配線の変更、`codex exec` での予期しない block / pass の報告にし、古い Trigger(`pre_bash_guard.sh` の次のタッチ)がこの PR で発火したが `ask` は probe していないことを書いた。Related に self-review の L-2 を足した。V-3 |
| 128(HTML コメント)、129(行) | 「`/sync-docs` still writes no insight event」を直した。`/sync-docs` の 4 面に Insight event の節が入り、`tests/test-skill-insight-cycle.sh` が 5 skill × 4 面で見ていることを書いた。129 行目の Why deferred に、`/sync-docs` の手順は追加済みと足した。V-5 |
| 157(`insights-append.sh` の 2 件) | (a) を取り消し線にして RESOLVED とした。shellcheck の一覧が 9 / 36 本だったのを `scripts/*.sh` のグロブにしたこと、露出した SC2209 を直したこと、template 側の `insights-append.sh` は root と同一(`check-sync.sh` が保証)なので root の lint が覆うことを書いた。Impact と Trigger の (a) も取り消し線にした。(b)(cap を上げた `/cross-review` が `cycle-count.json` を増やさない)は開いたまま。Related に plan を足した。V-2、self-review L-5 |
| 158(新) | `pre_bash_guard.sh` が捕まえない書き込みと deny の綴り。(a) `cp`・`mv`・`install`・`sed -i`(M-1 の 5 例は旧 guard が偶然捕まえていた)、`>&`、コマンド置換を含むパス、空白を含む引用符付きのパス、行の継続。(b) `git commit -am "$(id)"`、`-m"$(id)"`、`--message "$(id)"`、`sudo` とタブ。(c) sed fallback でタブの後ろの書き込み先を見ない(L-1)。bypass では deny の規則だけが残るので (b) がそのまま素通りになる、と Impact に書いた。V-6 |
| 159(新) | 長いコマンドで guard が遅い(200 KB で 28 秒、旧 guard の jq の経路も同じ)。V-6 |
| 160(新) | test の Test gaps の要約。(a) `cat >.env<<EOF`、`/usr/bin/tee`、`<` の読み込みが続く形(mutation G8、G9、G14)と、`plan`・`acceptEdits`・`dontAsk` の未テスト。(b) `test-skill-insight-cycle.sh` が `--cycle auto` しか見ない(S2、S3)。(c) `TMPDIR` が git の work tree の中にあると `test-verify-local-hook-tests.sh` の Case 1 が落ちる。(d) マージ後でないと確かめられない 2 点(live の payload の `permission_mode`、`/pr` での実際の archive)。V-6 |

新しい行と 157 行目の plan の参照は `docs/plans/active/2026-10-07-guard-bypass-and-hygiene.md` のままにした。`verify.local.sh` の参照の検査は実在するパスを見るので、いま archive のパスを書くと落ちる。`/pr` の `archive-plan.sh` が移動と同時に README の参照を書き換える(この PR が足した手順)。

## Drift check results

| 文書 | 結果 |
|------|------|
| `.codex/README.md:114-117`(root、template) | 変更なし。「`deny` が実際にコマンドを止める」は、deny がどのモードでも残るので今も正しい。`ask` には触れていない。root と template は `cmp` で一致 |
| `.claude/rules/ralph/git-commit-strategy.md:70`(root、template) | 変更なし。「`pre_bash_guard.sh` blocks dangerous patterns」は deny の規則のことで、変わっていない。HEREDOC の推奨形が guard に deny される問題は、124 行目の既存の行で、この PR の範囲外 |
| `docs/quality/quality-gates.md` | 変更なし。`verify.local.sh` の個々の検査を列挙していない。実行権限の検査は test モード、tech-debt の参照の検査は static モードにあり、「static と test は重ならない」の規則(`:31-37`)は保たれる |
| `docs/quality/definition-of-done.md`(root、template) | 変更なし。guard、`verify.local.sh`、insight event に触れていない |
| `docs/architecture/repo-map.md:61,66` | 変更なし。`archive-plan.sh`、`verify.local.sh`、`insights-append.sh` と `tests/test-*.sh` の glob は載っていて、新しいテスト 2 本の名前を挙げる必要がない |
| `.claude/skills/pr/SKILL.md` step 8(4 面) | 変更なし。S4 で、`archive-plan.sh` が README を書き換えたら同じコミットに入れる、と書いてある(verify AC6 で確認済み) |
| `.claude/skills/sync-docs/SKILL.md`(4 面) | 変更なし。S3 で Insight event の節が入っている。この report の insight event は、その節のコマンドで書いた |
| `internal/org/prompts/implementer.md:27` | 変更なし。`-m "$(` の deny は残っている |
| `AGENTS.md`、`CLAUDE.md`、`README.md`、`.claude/rules/ralph/post-implementation-pipeline.md` | 変更なし。パイプラインの順序と skill の一覧は変わっていない |
| plan の Non-goals(`:51`)が後続候補とした、Go のコメントと fixture に残る `docs/plans/active/` の参照(約 25 か所) | 変更なし。plan の範囲外 |

## 残したもの

- V-8(AC4 の直し方の表示が plan の文言より細かい): plan の本文は digest の対象なので直していない。Progress checklist に、`git add --chmod=+x`(untracked)と `chmod +x`(index だけが 100644)を出すことをメモした。AC4 は満たしている。
- Test gaps 1〜5 のコードとテストの追加: この pass は文書とコメントだけを扱うので足していない。tech-debt の 160 行目に、足す 1 行の形とあわせて記録した。
- self-review L-1 の正規表現の修正(`redirect_lead` と `tee_lead` が 2 文字の `\t` を受け付ける): hook の変更になり、`/self-review` から回し直すことになるので、コメントと tech-debt 158 行目(c)にとどめた。

## 確認

| コマンド | 結果 |
|----------|------|
| `./scripts/check-sync.sh` | PASS(DRIFTED 0、ROOT_ONLY 0、TEMPLATE_ONLY 11、KNOWN_DIFF 5) |
| `./scripts/check-skill-sync.sh` | PASS(13 skill) |
| `bash scripts/check-template-purity.sh` | PASS |
| `HARNESS_VERIFY_MODE=static ./scripts/verify.local.sh` | rc 0(shellcheck、hook の `sh -n`、settings の `jq -e`、check-sync、check-pipeline-sync、check-skill-sync、check-template-purity、tech-debt README の plan 参照。すべて OK) |
| `sh tests/test-pre-bash-guard.sh` | 220 件 PASS、FAIL 0(guard のコメントだけを変えた確認) |
| `./scripts/plan-visual.sh digest <plan>` | `c07adf402bdb`(承認の行と一致) |
| `cmp`(guard、insights README の root と template) | 一致 |

テストの全体(`./scripts/run-test.sh`、`go test`)は、コードを変えていないので再実行していない。test report の結果(shell 39 ファイル 1,943 件、Go 8 パッケージ)が、この commit のコードに対してもそのまま当てはまる。

## Verdict

- Verdict: pass
- Verified: 上の確認表
- Not verified: Codex が `ask` をどう扱うか、live の bypass session で新しい guard が `ask` を返さないこと(どちらも tech-debt の 127 行目と 160 行目に記録)
