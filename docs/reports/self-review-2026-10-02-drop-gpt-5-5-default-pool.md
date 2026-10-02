# Self-review report: drop-gpt-5-5-default-pool

- Date: 2026-10-02
- Plan: docs/plans/active/2026-10-02-drop-gpt-5-5-default-pool.md
- Branch: chore/drop-gpt-5-5-default-pool(HEAD f8fce663、コードのコミットは 22df1937)
- Reviewer: reviewer subagent (Claude)、cycle 1
- Scope: diff の品質だけ(正しさ、命名、読みやすさ、不要な変更、安全性、保守性、文の正確さ)。仕様への適合、テストの網羅、文書のずれは /verify・/test・/sync-docs の担当。対象は `git diff main...HEAD` から `docs/plans/` を除いた 11 ファイル

## Evidence reviewed

- `git diff main...HEAD --stat`: 12 ファイル、+305 / -14(plan を除くと 11 ファイル)
- 写しの一致: org skill の 4 面は `cmp` で一致。`scripts/ralph-config.sh` と template も一致。追加行にデバッグ出力や TODO はない
- `internal/cli/org.go:41-56`: 10 の動詞(spawn / start / send / wait / read / stop / status / disband / report / watch)はすべて `newOrgRuntime` か `newOrgRuntimeAt` を通り、`resolveOrgConfig`(`:218-231`)で設定を読む。`--config` は persistent flag で全動詞に効く。`resolveOrgConfig` は `--config` が空のとき cwd の `./ralph.toml` だけを見る
- `internal/org/statedir.go:47-58`: state dir は flag → `RALPH_ORG_STATE_DIR` → git の toplevel → cwd の順。設定ファイルの場所は使わない
- 設定の検証: `internal/config/config.go:292-299`。エラー文は skill の段落とテストが引く文と一致する
- リリース済みの既定: `git tag --contains 3f9b4a01`(codex の既定を入れたコミット、2026-09-16)は空。`git show v5.1.0:internal/config/config.go` の `Default()` の `ModelPool` は claude の `opus` / `sonnet` / `haiku` の 3 つだけで、codex のエントリはない。v5.1.0 の `templates/base/ralph.toml` も同じ 3 つを明示している。v5.1.0 の `config.go:234-235` は今と同じ roles の検証を持つ
- `.goreleaser.yml:33-39`: changelog は `^docs:`、`^test:`、`^chore:` を除く。`gh release view v5.1.0` の本文はコミットの件名と "Merge pull request #N from …" の行を並べたもの
- `docs/tech-debt/README.md`: この diff が閉じる行も、無効にする行もない

### probe

scratchpad に HEAD(f8fce663)と v5.1.0 をビルドした。`HOME` を scratch に向け、`GIT_CONFIG_GLOBAL=/dev/null`、`GIT_CONFIG_NOSYSTEM=1`、`PATH` は git の symlink だけ(herdr と agmsg は見えない)。fixture は scratch の git repo で、toplevel に `[org.roles] reviewer = ["gpt-5.5"]` だけの `ralph.toml`、`.harness/state/org/manifest.jsonl` に codex `gpt-5.5` の座席 `seat-1`(spawned)を置いた。直したコピーは repo の外に `[org] max_seats = 5` だけで置いた。

| probe | 実行 | 結果 |
| --- | --- | --- |
| A | HEAD、toplevel で `org status --org-id org-a`(`--state-dir` なし) | rc 1、`org: load config: [org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool` |
| B | A に `--config <repo の外の直したコピー>` | rc 0、`seat-1 codex gpt-5.5 spawned (active)`(state dir は git の toplevel から) |
| C | HEAD、サブディレクトリで `--config` なし | rc 0、`seat-1` を表示(cwd に `ralph.toml` がないので組み込みの既定で動く) |
| D | HEAD、サブディレクトリで `--config ../ralph.toml` | rc 1、A と同じエラー |
| E | HEAD、`org report`(`--config` なし / あり) | rc 1 / rc 0 |
| F | HEAD、`ralph.toml` のない cwd で `org spawn --driver codex --model gpt-5.5 --dry-run` | rc 1、`org: model "gpt-5.5" not in [org].model_pool for driver "codex"` |
| G | v5.1.0 のビルド、A と同じ fixture と同じコマンド | rc 1、A と同じエラー |
| H | scratch の `internal/config` のコピーで `Default()` に `gpt-5.5` を戻し、`TestDefault_Org` を実行 | `config_test.go:133` の長さの `t.Fatalf` だけが出る。新しいループ(`:140-146`)は実行されない |

新しい 2 つのテスト(`TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel`、`TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_Errors`)と `TestDefault_Org` を単独で 1 回ずつ実行し、pass を確認した。suite 全体は実行していない(/test の担当)。

## Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M-1 | MEDIUM | 文の正確さ(下流に配る文書) | 移行と復旧の段落は「`model_pool` を書かず `[org.roles]` で `gpt-5.5` を指定している project は、バイナリを更新すると `ralph.toml` の検証が通らなくなる」と書く。リリース済みのバイナリでは、これが起きない。`gpt-5.5` が既定に入ったのは 3f9b4a01(2026-09-16)で、どのタグにも含まれていない。Homebrew の v5.1.0 の既定は claude の 3 つだけなので、その設定は v5.1.0 でもすでに同じエラーで止まる(probe G)。v5.1.0 から次のリリースへの更新では、既定にエントリが足されるだけで(`fable` と codex の 4 つ)、外れるエントリはない。つまり、v5.1.0 で読めていた設定が更新後に読めなくなることはない。影響を受けるのは、3f9b4a01 からこの PR までの main をソースからビルドした場合だけになる。この段落は `templates/base/` の 2 面から全 scaffold に配られる。下流の運用者は、自分の既定に一度も入っていなかったスラッグについて日付つきの経緯を読み、移行として「`[org].model_pool` を明示して `gpt-5.5` を含める」ことを勧められる。それに従うと、退役予定のスラッグを明示のプールに固定し、以後の既定の更新も届かなくなる。spec の (e) の「下流 project は、バイナリ更新後に設定の検証エラーになり」も同じ前提に立っている。plan が release notes に載せる予定の 1 行(移行と復旧の要点)も、Homebrew の利用者全員にとって誤りになる。加えて、main 同士で比べた場合の影響範囲も段落の記述より広い。`--model` を明示する運用(skill の規則)で `spawn --model gpt-5.5` を使っていた場合も、既定のプールのままなら拒否される(probe F、`internal/org/envelope.go:56`) | `.claude/skills/org/SKILL.md:115-125`(4 面とも同じ)、`docs/specs/2026-08-01-org-runtime.md:23`、`git tag --contains 3f9b4a01`(空)、`git show v5.1.0:internal/config/config.go`(`ModelPool` は claude 3 つ)、probe A・F・G | skill の 4 面は、日付つきの経緯をやめ、版に依存しない復旧の手順にする。例: 「`model_pool` を書かずに `[org.roles]` で既定のプールにないモデルを指定すると、`ralph.toml` の検証が通らず、`ralph org` の動詞はどれも(`status` / `stop` / `disband` も)`[org.roles].<role> references model "<model>" not present in [org].model_pool` で止まる。直すには …、または直したコピーを `--config <path>` で渡す。state dir は設定ファイルの場所では変わらない」。この形なら、今後の既定の変更にもそのまま当てはまる。spec の (e) には、`gpt-5.5` を既定に含むリリースがないこと(`git tag --contains 3f9b4a01` が空)と、影響が main のソースビルドに限られることを書く。release notes への申し送りは、移行の警告ではなく、v5.1.0 の 3 エントリから 8 エントリへの拡張(#152 の変更)を説明する内容に変える |
| L-1 | LOW | テストのコメントと挙動 | `TestDefault_Org` に足したループのコメントは「a regression that re-adds it must fail here by name, not only via the length check」と書く。`gpt-5.5` を戻すと件数が 9 になり、`:133` の `t.Fatalf` でテストが止まるので、このループは実行されない(probe H)。別のエントリを `gpt-5.5` に置き換えた場合も、要素ごとの比較(`:135-139`)が先に名前つきで落ちる。どちらの場合も、ループは新しい情報を出さない。なお、`:133` の Fatalf はプール全体を表示するので、`gpt-5.5` の名前はもともと出力に現れる | `internal/config/config_test.go:132-146`、probe H | ループとコメントを消す。名前で落としたいなら、ループを長さの確認より前に置く |
| L-2 | LOW | テストが確かめる範囲 | `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` は 2 回とも `--state-dir` を明示している。そのため、テストのコメント(`:896-898`)と skill の段落が書く「state dir は設定ファイルの場所で変わらない」は、flag の段でしか確かめていない。flag の段は設定の場所と最初から無関係なので、この確認はほとんど意味を持たない。運用者がふつう使う経路(`--state-dir` なし、git の toplevel)は probe B で動くことを確かめたが、テストでは固定されていない。たとえば「`--config` があって `--state-dir` がないときは `filepath.Dir(configPath)` の下を使う」という変更が入っても、このテストは緑のまま | `internal/cli/org_test.go:899-959`(`:915`、`:950`)、`internal/org/statedir.go:47-58`、probe B | サブテストを 1 つ足し、`--state-dir` の代わりに `t.Setenv("RALPH_ORG_STATE_DIR", stateDir)` を使う。env の段なら git が要らず、`PATH=""` のままで動く |
| L-3 | LOW | 文の言い方 | (1) 「退役予告と "Legacy" の表示が付いたため」は、両方が今も付いているように読める。spec の (e) にあるとおり、2026-10-02 の観測では予告が消え、説明が "Legacy coding model." のまま残っていた。(2) state dir の決まり方を「`--state-dir`、`RALPH_ORG_STATE_DIR`、git の toplevel の順」と書いているが、4 番目の cwd の段が抜けている。git の外の project では cwd の段が使われる | `.claude/skills/org/SKILL.md:115-116`、`:123-124`、`docs/specs/2026-08-01-org-runtime.md:23`、`internal/org/statedir.go:56-57` | M-1 で段落を書き直すときに直す。(1) は経緯を skill から外せば消える。(2) は「…、git の toplevel、cwd の順」にする |
| L-4 | LOW | テストの名前 | `TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_Errors` は、後半で移行後の設定が読めること(エラーにならないこと)を確かめているが、名前は `_Errors` で終わる。後半の確認は、CLI のテストの 2 つ目のサブテストとも重なる | `internal/config/config_test.go:850-884` | 名前を `…_ErrorsUntilModelPoolIsExplicit` のように後半も表す形にするか、後半を別のテストに分ける |
| L-5 | LOW | 申し送りの経路 | plan は release notes の 1 行を PR の本文に申し送るとしている。しかし release notes の自動生成は `^chore:` を除く(`.goreleaser.yml:39`)。このため、22df1937(`chore: …`)は載らず、残るのは merge コミットの "Merge pull request #N from yoshpy-dev/chore/drop-gpt-5-5-default-pool" の行だけになる。#186 の受け入れ条件は手で書く release notes を前提にしているが、#186 の本文はこの PR の申し送りを挙げていない | `.goreleaser.yml:33-39`、`gh release view v5.1.0` の本文、#186 の本文 | M-1 で直した内容の 1 行を、#186 にコメントとして残す。PR の本文だけに置かない |

CRITICAL と HIGH はない。

## 依頼された観点への回答

- すべての動詞が設定を読むか: 読む。10 の動詞とも `resolveOrgConfig` を通り、`--config` は persistent flag なので全動詞に効く(`stop`、`disband`、`watch`、`report` も同じ。probe E で `report` を確認)。ただし、`--config` がないときに読むのは cwd の `./ralph.toml` だけ。git の toplevel のサブディレクトリから動詞を実行すると、project の `ralph.toml` を読まずに組み込みの既定で動く(probe C)。したがって「どれも設定を読むので…止まる」が成り立つのは、`ralph.toml` のあるディレクトリで実行した場合に限られる。この非対称(設定は cwd、state dir は toplevel)は今回の diff より前からある。
- state dir の順序: コードの順序は flag → env → git の toplevel → cwd。段落は cwd の段を書いていない(L-3)。設定ファイルの場所が state dir に影響しないことは正しく、probe B で確かめた。
- 新しい CLI テストが復旧の経路を証明しているか: `--config` を無視する変更が入れば落ちる(cwd の `ralph.toml` が壊れた方なので)。エラー文の確認は role とモデル名まで含むので、別の原因のエラーでは通らない。`runOrgCmd` は呼ぶたびに `NewRootCmd()` を作るので、flag の値が呼び出しの間で残ることもない。`PATH=""` と明示の `--state-dir` によって、git と herdr にも触れない。弱い点は L-2(state dir の独立性を flag の段でしか見ていない)。
- `gpt-5.5` や 9 エントリ / codex 5 スラッグの記述の残り: 既定のプールとしての記述は残っていない。`git grep -n 'gpt-5\.5'` で、履歴と `*_test.go` を除いて残るのは次のものだけ。skill の新しい段落、spec の (c) と (e)、`.codex/config.toml`(+ template)の `model =`、`scripts/verify.local.sh` と `tests/*.sh` の codex 設定の fixture、`docs/specs/2026-05-07-codex-cli-parity.md`(codex CLI の既定の話)、`internal/cli/doctor_shell_alias.go:627` のコメントの例。件数の記述(`9 entries`、`5 スラッグ` など)は見つからなかった。「前提」節の「プール先頭(claude は `fable`、codex は `gpt-6-astra`)」と、`templates/base/ralph.toml` の `[org.roles]` のコメント例(`gpt-6-astra`)は今も正しい。
- 日本語の質: 文は短く、移行と復旧を括弧の条件で分けていて読みやすい。問題は書き方より中身で、下流にとって前提が成り立たない(M-1)。

## Positive notes

- 既定値の 3 面と template の `ralph-config.sh`、org skill の 4 面が、それぞれ byte 一致している。
- CLI のテストは、壊れた設定で失敗することを先に確かめてから、直した設定で座席が見えることを確かめる。最初の確認がなければ、何もしなくても通るテストになるところだった。
- spec の (d) の記録項目「既定 5 スラッグの有無」を個数のない言い方に変えたので、次に既定が変わっても古くならない。
- `TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder` の期待値も追従していて、driver_pool だけを書いた設定の経路も 4 スラッグで固定されている。

## Coverage gaps

- suite 全体、静的解析、`check-sync.sh` / `check-skill-sync.sh` は実行していない(/verify と /test の担当)。
- `stop` と `disband` は実際の herdr と agmsg を使わずには動かせないので、probe では `status` と `report` だけを使った。`--config` が効く仕組みは全動詞で共通(`newOrgRuntimeAt`)なので、同じ結果になると判断した。実行はしていない。
- サブディレクトリから実行すると組み込みの既定で動く件(probe C)は、この diff より前からある。tech-debt の行にするかどうかは /sync-docs かメンテナが判断する。

## Recommendation

- Merge: 重大度の基準では可(CRITICAL / HIGH なし)。ただし、M-1 は cycle 2 で直してからの merge を勧める。直す対象は文書だけ(skill の 4 面と spec の (e))で、直さないまま出すと、下流のすべての scaffold に誤った移行の指示が配られる。
- Follow-ups:
  - M-1 の書き直しにあわせて、L-3 を直す。
  - L-1、L-2、L-4 はテストの小さな修正なので、M-1 と同じ cycle でまとめて直せる。
  - L-5: 直した 1 行を #186 にコメントとして残す。
  - 設定は cwd の `./ralph.toml`、state dir は git の toplevel から決まる非対称(probe C)は、別 issue か tech-debt の行の候補。

## Cycle 1 addendum (Slice B)

- Date: 2026-10-02
- Branch: chore/drop-gpt-5-5-default-pool(HEAD 972cd37b、Slice B のコミットは 67c20db0)
- Scope: `git diff 26b35264..HEAD -- .claude .agents templates docs/specs internal`(skill の 4 面、spec の (e)、`internal/config/config_test.go`、`internal/cli/org_test.go`)。plan の差分は対象外

### Evidence reviewed

- skill の 4 面は `cmp` で一致。出荷する skill(`.claude/skills`、`.agents/skills`、`templates/base/.claude`、`templates/base/.agents`)に `gpt-5.5` は 1 つも残っていない(`git grep -c` が rc 1)
- 段落の主張とコードの照合
  - 動詞が設定を読む: 10 の動詞すべてが `newOrgRuntime` / `newOrgRuntimeAt` を通る(cycle 1 の Evidence と同じ)
  - `--config` を省略したときの設定: `internal/cli/org.go:218-226`。cwd の `./ralph.toml` があればそれを読み、なければ組み込みの既定を使う
  - spawn の拒否: `internal/org/envelope.go:55-57`。同じ driver と model の組が `model_pool` にないときに `org: model %q not in [org].model_pool for driver %q` を返す。文面と条件は段落と一致する。driver が `driver_pool` にないときは、その前の `:52-54` で別のエラーになる
  - state dir の順序: `internal/org/statedir.go:47-58` の flag → env → git の toplevel → cwd と一致する
- spec の (e): `git fetch --tags` のあとも `git tag --contains 3f9b4a01` は空で、最新のタグは v5.1.0。3f9b4a01 は HEAD の祖先。v5.1.0 の既定が claude の 3 つだけであることは cycle 1 で確認済み
- mutation probe。いずれも scratch のコピー(`git archive HEAD`)で行った
  - `Default()` に `gpt-5.5` を戻す: `TestDefault_Org` は `config_test.go:137` で名前つきの Errorf を出し、そのあと `:141` の長さの Fatalf で止まる
  - `ResolveOrgStateDir` の env の段を無効にする: 新しいサブテストが `org_test.go:984` で落ち、`no seats` が出る
  - cycle 1 の L-2 で挙げた変更(`--config` があって `--state-dir` がないときに `filepath.Dir(configPath)` の下を使う)を `newOrgRuntime` に入れる: 同じく `:984` で落ちる
- 新しいサブテストを含む `TestOrgStatus_ConfigFlagRecoversAfterDefaultPoolDropsRoleModel` を単独で実行し、3 つのサブテストとも pass

### cycle 1 の指摘の扱い

| # | 状態 | 根拠 |
| --- | --- | --- |
| M-1 | 直った | 段落から日付つきの経緯と `gpt-5.5` が消え、版に依存しない復旧の手順になった。spec の (e) には、`gpt-5.5` を既定に含むリリースがないことと、影響がソースビルドに限られることが書かれた。release notes への申し送りは取り下げられた。言い回しの残りは A-L1 と A-L2 |
| L-1 | 直った | ループが長さの確認より前に移り、コメントの内容も挙動と合う(mutation probe)。メッセージは "retired" から "dropped" に変わり、`gpt-5.5` がまだ退役していない事実と合う |
| L-2 | 直った | env の段のサブテストが追加された。cycle 1 で挙げた変更も、env を無視する変更も、どちらでも落ちる。git の toplevel の段は `PATH=""` のため試せないと、コメント(`org_test.go:961-963`)に書いてある。cycle 1 の probe B で、この段が動くことは確認済み |
| L-3 | 直った | 退役の理由は skill から消えた。state dir の順序に cwd の段が入った(`SKILL.md:125-126`) |
| L-4 | 直った | 名前が `…_ErrorsUntilModelPoolIsExplicit` になった(`config_test.go:850`)。doc comment も前半と後半の両方を書いている |
| L-5 | 解消 | M-1 の訂正で、申し送る内容そのものがなくなった。plan の Non-goals に理由が書かれている |

### Findings

| # | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| A-L1 | LOW | 文の正確さ(skill) | 段落の後半に、手順として足りない点が 3 つある。(1) 「直すには、`[org].model_pool` を明示してそのモデルを含めるか、role から外す」は、直前の spawn の文にも続けて読める。しかし spawn の拒否には「role から外す」が当てはまらず、別のモデルを `--model` に渡すという直し方が抜けている。(2) `start --model` も同じ `ValidateSpawnEnvelope` を通って同じ文面で拒否されるが、段落は `spawn` だけを挙げる。(3) `model_pool` を明示すると既定のプールを丸ごと置き換える。この段落だけを読んで 1 エントリだけ書くと、既定の残りのエントリが使えなくなる。前の段落の「明示した `model_pool` は自動では変わらない」は、この点を言っていない。ほかに、`driver_pool` だけを書いた設定の実効のプールは「既定のプール」ではなく、既定を driver で絞ったものになる(前の段落で説明はある)。また、「直すには…。`ralph.toml` を直すか…」と「直す」の文が続いて読みにくい | `.claude/skills/org/SKILL.md:115-124`(4 面とも同じ)、`internal/cli/org.go:310`、`:420`(どちらも `rt.Spawn`)、`internal/org/spawn.go:375`、`:502`、`internal/config/config.go:241-250` | 例: 「直すには、`[org].model_pool` を明示してそのモデルを含める(明示したプールは既定を置き換えるので、使う既定のエントリも書く)か、role から外す。`spawn` / `start` なら別のモデルを `--model` に渡してもよい。`ralph.toml` で直すか、直したコピーを `--config <path>` で渡す」 |
| A-L2 | LOW | 文の正確さ(spec) | (1) 「既定に入れた 3f9b4a01 はどのタグにも含まれず」は、次のタグを切った時点で事実でなくなる。3f9b4a01 は HEAD の祖先なので、merge のあとに main で切るタグはすべて 3f9b4a01 を含む。結論の「`gpt-5.5` を既定に含むリリースはない」は、除外がタグより先に入るので正しいままだが、根拠の文は、後から `git tag --contains` で確かめた人には誤りに見える。(2) 「影響を受けるのは、…`[org.roles]` で `gpt-5.5` を指定して `model_pool` を省略した人だけ」は、同じ期間のソースビルドで、既定のプールのまま `--model gpt-5.5` で spawn / start していた人を含まない。skill の段落は、この場合も挙げている | `docs/specs/2026-08-01-org-runtime.md:23`、`git merge-base --is-ancestor 3f9b4a01 HEAD`(真) | (1) 「3f9b4a01 からこの変更までの間にタグは切られていない(2026-10-02 時点の最新は v5.1.0)」のように、後から読んでも成り立つ形にする。(2) 「…省略した人と、既定のプールのまま `--model gpt-5.5` を渡していた人」に広げる。spawn / start の拒否は座席を作らないので、teardown が止まる話とは分けて書く |

CRITICAL、HIGH、MEDIUM はない。

### 対象外の気づき(件数に含めない)

- plan の Risks の行は、元の前提(「下流で…バイナリを更新すると設定の検証エラーになる」「課金が続きうる」)を先に書き、訂正をあとに足している。また「`ralph init` が作る `ralph.toml` は `model_pool` を明示していて `gpt-5.5` も含む」は、この PR の前の main の template の話になっている。v5.1.0 の template の `ralph.toml` は claude の 3 つだけを明示している。plan は archive されるだけで配布されないので、直すかどうかは orchestrator に任せる。

### Recommendation

- Merge: 可。M-1 と L-1〜L-4 は意図どおりに直っていて、残るのは LOW 2 件の言い回しだけ。
- Follow-ups: A-L1 と A-L2 は文書だけの修正で、4 面の同期と spec の 1 行で済む。同じ cycle で直すか、見送るかは orchestrator が判断する。見送る場合、配られる skill の段落は誤りではなく、手順が足りないだけになる。
