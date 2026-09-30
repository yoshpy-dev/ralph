# Walkthrough: codex-exec-stdin-and-model (#184)

- Date: 2026-09-30
- Plan: docs/plans/active/2026-09-29-codex-exec-stdin-and-model.md
- Branch: fix/codex-exec-stdin-and-model(base main 6e26aaf)
- Diff: 28 files、+1036 / -81(plan・報告・insight を除くと 19 files、+522 / -81。skill 本文は 4 面のミラーで同じ変更が 4 回ずつ入る)

## 何を直したか

`/plan` の Codex plan advisory と `/cross-review` の codex reviewer は、Bash ツールから `codex exec` を呼ぶ。これまでの呼び出しには 3 つの問題があった。

1. stdin を閉じていないので、codex が「Reading additional input from stdin...」で止まることがあった(#153 で約 4 時間)。
2. model と reasoning effort を指定していないので、`.codex/config.toml` の model、ユーザー設定の effort、shell alias の組み合わせによっては API が 400 を返した(#162)。
3. 長い呼び出しに完了の判定がなかった。失敗や hang が「指摘なし」に見えうる。

## 読む順

1. `scripts/ralph-config.sh`(template と byte 一致): `RALPH_CODEX_REVIEWER_MODEL`(既定 `gpt-6-astra`)と `RALPH_CODEX_REASONING_EFFORT`(既定 `xhigh`)を追加して export した。`RALPH_CLAUDE_REVIEWER_MODEL` と同じ形。
2. `.claude/skills/plan/SKILL.md` の step 11.c と d: 呼び出しの 1 行と理由、完了の contract。
   - 呼び出しは `rm -f <out>; command codex -m … -c model_reasoning_effort=… exec … -o <out> </dev/null > <log> 2>&1 & cpid=$!; sleep 1200 & spid=$!; ( while kill -0 "$spid"; do sleep 1; done; kill "$cpid" ) & wpid=$!; wait "$cpid"; rc=$?; kill "$spid"; kill "$wpid"; echo "codex rc=$rc"`。
   - `sleep 1200` と codex を直接の兄弟として background にしている。wrapper の subshell の中に `sleep` を入れると、wrapper を kill しても `sleep` が孤児として残るため(実機で確認)。
   - 完了は「`codex rc=0` かつ `-o` のファイルが空でない」ときだけ。watchdog に TERM された codex は rc 0 で終わるので(codex-cli 0.154.0 で確認)、時間切れは rc ではなく `-o` の欠落で判定する。未完了のときは「Codex advisory: incomplete (<reason>)」と記録し、「Codex: no findings」とは書かない。
3. `.claude/skills/cross-review/SKILL.md`:
   - step 4 は同じ形の `exec review --base "$BASE"`。config の source と `$BASE` の算出は同じ Bash 呼び出しの中で行う。
   - 未完了のとき: triage report の header に `Reviewer status: incomplete (<reason>)` を書き、triage を省く。Step 8 は Case A〜C ではなく、再実行・`/pr` に known gap として記録・中止の 3 択にする。insight は `--verdict n/a` で、再実行では cycle-count を増やさない。
   - claude reviewer の経路も、exit 0 で結果が空でないときだけ complete とする。
4. ミラー 6 面(`.agents/skills/` の 2 つと `templates/base/` の 4 つ): `scripts/sync-skills.sh` で再生成した。`check-skill-sync.sh` と `check-sync.sh` が一致を保証する。
5. `tests/test-codex-exec-invocation.sh`(新規、100755、96 assertion): 4 面 × 2 skill の呼び出し行について次を検査する。
   - `command codex `、`-m "${RALPH_CODEX_REVIEWER_MODEL:-`、`model_reasoning_effort=${RALPH_CODEX_REASONING_EFFORT:-`、` -o `、`</dev/null` があること
   - 呼び出し行の数(plan 1、cross-review 2)
   - fallback が `ralph-config.sh` の既定と一致すること
   - cross-review に `Reviewer status: incomplete` があること
6. `internal/config/defaults_sync_test.go`: SKILL.md の fallback の検査を表駆動(5 組)にした。すべての出現を検査する。
7. 文書:
   - `model-routing.md`: sync note、値の所在、codex には alias がないこと
   - `docs/recipes/codex-setup.md`: agent の Bash から `codex exec` を呼ぶときの規則
   - triage template: `Reviewer status` 行
   - tech-debt: backfill が 0/0/0 の After-triage 行を pass と読む件

## 検証

| 項目 | 結果 |
|---|---|
| `sh tests/test-codex-exec-invocation.sh` | 96 / 96 |
| `sh tests/test-ralph-config.sh` | 19 / 19 |
| `go test ./internal/config/...`(`TMPDIR=/tmp` も) | ok |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` / `run-static-verify.sh` | green |
| check-skill-sync / check-sync | pass |
| 実機: skill の 1 行で ok の経路 | rc 0、約 15 秒、`-o` が `ok` |
| 実機: `sleep 3` で時間切れ | rc 0、`-o` なし(contract で未完了と判定) |
| 実機: cross-review の形 | rc 0、約 291 秒、`-o` 190 バイト |
| 実機: dash と `bash --posix` | どちらも完走 |
| 孤児プロセス、`.codex/config.toml` | どの実行でも残らない、変更なし |
| mutation(`</dev/null`、`-o`、`command`、fallback、既定、行分割) | すべて red |
| この PR の cross-review 自体 | 新しい形で実行、`codex rc=0`、`-o` 189 バイト、指摘 0 件 |

## pipeline の履歴

- plan: Codex advisory の MEDIUM 1 件(バックグラウンド実行に完了と失敗の contract がない)を受けて、`-o`、完了の contract、未完了の経路(AC-8)を追加した。
- self-review: MEDIUM 5 / LOW 10。Slice B で 14 件を直した。直す過程で、handoff に書いた watchdog の形の欠陥 2 つを implementer が実機で見つけて直した。完了 contract の段落の重複 1 件は設計判断で残した(各 skill は単独で読まれ、cross-review 側には未完了時の経路が要る)。
- verify pass → test pass → sync-docs で drift なし → cross-review で指摘 0 件。

## 残る gap

- `sleep 1200` を別の値に変えても、どのテストも捕まえない。上限の値はテストしていない。
- `.codex/config.toml` の `model = "gpt-5.5"`(2026-10-14 に退役予定)は #156 の観測で判断する。`exec` の呼び出しは `-m` を明示するようになったので、この値の影響を受けない。
- push 済みの branch で既定の scope が言語 pack を飛ばす件は #190 で扱う。
