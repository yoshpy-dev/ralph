# codex-seat-permission-verification

- Status: In progress
- Owner: Claude Code
- Date: 2026-09-18
- Related request: `[org.permissions].codex_verified` は `false` のままで、codex 座席は `guarded` 以外を fail-closed で拒否する。`codex_verified = true` で有効になるマッピング(autonomous → `--sandbox workspace-write --ask-for-approval never`、edits → `--sandbox workspace-write`)は実機未検証。現行 codex CLI で実機確認し、herdr pane 内の codex 座席で typed TASK → RESULT の E2E を 1 回通し、検証ログを `docs/evidence/` に残す(issue #155)
- Related issue: 155
- Type: docs
- Branch: docs/codex-seat-permission-verification

## Objective

このマシン(codex-cli 0.154.0、herdr、agmsg 1.1.13、tmux)で、`codex_verified = true` を与えたスクラッチ config を使って codex 座席を autonomous と edits の 2 モードで実起動し、(a) autonomous 座席がツール実行時に承認ダイアログで止まらないこと、(b) edits 座席が workspace 内のファイル編集は自動承認し workspace 外の操作は問い合わせること、(c) typed TASK を送って RESULT が agmsg 経由で返ること、を観測して `docs/evidence/codex-seat-permissions-2026-09-18.md` に redact 済みで記録する。既定 `codex_verified` は **false のまま**(ユーザー決定)とし、検証済み手順を `docs/recipes/codex-seat-permissions.md` として root と `templates/base/` の両方に置く。`templates/base/ralph.toml` のコメント、`internal/org/permissions.go` のコメント、`/org` skill の「permission 作法」を「未検証のための暫定制約」から「マシンごとの検証を経て運用者が opt-in する設計」に書き換える。

## Scope

| # | 変更 | ファイル | 内容 |
|---|------|---------|------|
| 1 | 実機検証(autonomous) | `docs/evidence/codex-seat-permissions-2026-09-18.md`(新規) | worktree から `go build -o <scratch>/ralph ./cmd/ralph` した一時バイナリ(インストール済み 5.1.0 は repo より古い)を使う。スクラッチ cwd(`git init` した一時ディレクトリ、本 repo 外)とスクラッチ `ralph.toml`(`[org.permissions] default = "autonomous"`、`codex_verified = true`)を用意し、`ralph org spawn --org-id codex-perm-<日付> --id reviewer --role reviewer --driver codex --model gpt-5.5 --cwd <scratch> --scope "<scratch>/**" --config <scratch ralph.toml> --state-dir <scratch state>` を実行。`spawned` イベントの Details には論理モード(`permission_mode=autonomous`)しか記録されない(`spawnedEventDetails`、dry-run で確認済み)ので、実際の子プロセス引数 `--sandbox workspace-write --ask-for-approval never --model gpt-5.5` は `herdr pane process-info <pane>` または `ps -o args -p <pid>` で codex プロセスのコマンドラインから取って記録する。`herdr pane read` で起動直後の pane に承認 / trust / login ダイアログが出ていないことを確認。`ralph org send --to reviewer` で TASK(`TYPE: TASK` / `TASK_ID: t-1`、本文: スクラッチ cwd 内に `hello.txt` を shell で作り、その sha256 を EVIDENCE に入れて RESULT を返す)を送り、`ralph org wait --seat reviewer --until idle,done --timeout-ms 180000` → `ralph org read --seat reviewer --lines 80` で pane を確認 → agmsg の受信(`ralph org report` または agmsg history)で RESULT を確認。`ralph org stop --seat reviewer` → `ralph org disband` → `ralph org status` に active 座席なし |
| 2 | 実機検証(edits) | 同上 evidence | スクラッチ config の `[org.permissions.roles] reviewer = "edits"` で再 spawn(別 org_id)。`spawned` Details に `--sandbox workspace-write` のみ(`--ask-for-approval` なし)が記録されること。TASK: (i) cwd 内の `notes.md` を編集(自動承認されるはず)、(ii) codex の workspace-write sandbox の writable root(cwd と `/tmp` / `$TMPDIR` 系)の**外**にあることが明らかなパス `$HOME/codex-perm-outside-2026-09-18.txt`(事前に存在しないことを確認し、事後に存在しないことを独立に確認)への書き込みを試みる。pane で (ii) に承認プロンプトが出るか、sandbox で拒否されて model にエラーが返るかを観測し記録。結果は 3 値で判定する — **pass**: (i) がプロンプトなしで完了し、(ii) で承認プロンプトの文言が pane に観測された / **partial**: (i) は完了したが (ii) はプロンプトなしで sandbox 拒否(「問い合わせる」は未確立。recipe にはそのまま書き、opt-in の記述は「cwd 外は拒否される」までに留める)/ **inconclusive**: 座席が (ii) の tool call に到達しなかった、または pane が読めなかった。承認プロンプトが出た場合は `herdr pane send-keys` で拒否し、seat を stop |
| 2b | 実効設定の記録 | 同上 evidence | 省略時の approval policy はユーザー / プロジェクト config を継承するため、検証時の `~/.codex/config.toml` の `approval_policy` / `sandbox_mode` / `sandbox_workspace_write.*`(`writable_roots` 等)/ `[profiles.*]` の値と、スクラッチ cwd に `.codex/config.toml` が無いこと、`codex --version` を evidence に記録する。recipe の opt-in 手順は「この実効設定で検証した」旨と、config やバージョンを変えたら再検証する旨を含める |
| 3 | evidence の redact | 同上 evidence | ホームディレクトリ(`/Users/<user>` → `~`)、herdr / agmsg / codex の session UUID、pane id 以外の識別子を伏せる。生ログは scratchpad に置き、evidence には手順・コマンド・観測結果・manifest イベントの要点だけを書く(pointer 原則) |
| 4 | recipe | `docs/recipes/codex-seat-permissions.md`(新規)+ `templates/base/docs/recipes/codex-seat-permissions.md`(byte 同一) | 前提(codex CLI バージョン、herdr / agmsg)、スクラッチ config の書き方、autonomous / edits それぞれの spawn コマンドと「観測すべきこと」、本マシンでの結果(evidence へのポインタ)、自分の `ralph.toml` で `codex_verified = true` にする手順と注意(codex のバージョンが変わったら再検証)、後始末(stop / disband / status 確認)。`docs/recipes/codex-setup.md` の末尾に 1 行のポインタを追加(root と template を同時に) |
| 5 | コメント・文書の位置づけ更新 | `templates/base/ralph.toml`、`internal/org/permissions.go`(コメントのみ)、`.claude/skills/org/SKILL.md` + 3 ミラー | `ralph.toml` の `[org.permissions]` コメント 2 箇所の「see docs/tech-debt/README.md」(該当行は存在しない)を recipe へのポインタに置換し、「fail-closed until live-verified」を「既定は fail-closed。マシンごとに recipe の手順で検証してから `codex_verified = true` にする」に。`permissions.go` の fail-closed コメント(84-93 行付近)に「2026-09-18 に codex-cli 0.154.0 で検証済み(evidence 参照)。既定 false は運用者の opt-in 設計として維持」を追記。`/org` skill 「permission 作法」の「fail-closed。実機検証未了のための暫定制約」を「既定 fail-closed。`codex_verified = true`(recipe の手順で検証後)で autonomous / edits が有効」に書き換え、`scripts/sync-skills.sh` + cp で 4 面同期 |

## Non-goals

- `codex_verified` の既定変更(ユーザー決定: false のまま)
- `internal/org/permissions.go` のロジック変更(コメントのみ)
- codex のフラグマッピング自体の変更(検証結果が期待と異なった場合は evidence と recipe に事実を記録し、マッピング変更は別 issue にする)
- 検証の自動化(CI での codex 実起動)
- claude 座席の bypass 承諾ダイアログの扱い(既存の `/org` skill 記述のまま)
- tech-debt 行のクローズ(`codex_verified` に関する行は存在しないため、`ralph.toml` の誤ったポインタを直すだけ)

## Assumptions

- herdr は `ralph org spawn` が workspace / tab / agent を作れる状態(doctor pass 済み)。tmux セッションは herdr が管理する
- 座席の codex は `--model gpt-5.5`(issue 指定)。ユーザー shell の `codex` alias(`-m gpt-6-astra`)は herdr 経由の起動には適用されない
- codex の対話セッションは初回起動時に trust / login のダイアログを出しうる。出た場合は evidence に記録し、`herdr pane send-keys` で対処する(login が必要なら ~/.codex の既存認証が使われる想定)
- codex の workspace-write sandbox は cwd に加えて `/tmp` / `$TMPDIR` 系を writable root に含みうる(Codex plan advisory HIGH-1)。cwd 外の否定テストは `$HOME` 直下の使い捨てパスで行う
- 「承認ダイアログで止まらない」の判定は、TASK 送信後に `ralph org wait` が `idle,done` で返り、pane に承認待ちの表示がなく、RESULT が届いたこと。「問い合わせる」の判定は pane に approval prompt が表示されたこと(`herdr pane wait-output` で文字列待ちが可能なら使う)
- evidence は `docs/evidence/README.md` の慣例に従う markdown(既存の `codex-hooks-livefire-*.md` と同型)
- recipe は seed-once(下流には `ralph init` 時のコピーで届く)。root と `templates/base/` を byte 同一にして `check-sync.sh` を通す(`docs/recipes/` は ROOT_ONLY 除外対象外)

## Affected areas

- `docs/evidence/codex-seat-permissions-2026-09-18.md`(新規)
- `docs/recipes/codex-seat-permissions.md`、`templates/base/docs/recipes/codex-seat-permissions.md`(新規)、`docs/recipes/codex-setup.md` + template(1 行)
- `templates/base/ralph.toml`(コメント)
- `internal/org/permissions.go`(コメント)
- `.claude/skills/org/SKILL.md` + 3 ミラー

## Design decisions

Critical forks: 2 件、ユーザーと解決済み(2026-09-18、AskUserQuestion)。

- **実機検証をこのセッションで行う → 進める**。使い捨て org_id とスクラッチ cwd / state-dir を使い、承認ダイアログが出たら報告して止まる。終了後は disband で座席と pane を片付ける
- **`codex_verified` の既定 → false のまま + 手順を recipe 化**。fail-closed を維持し、下流は各自の codex バージョンで検証してから true にする。理由: codex の `--sandbox` / `--ask-for-approval` の挙動はバージョン依存で、メタリポでの 1 回の検証を全下流の既定にするのはリスクが大きい

既定として採った選択:

- 実機検証(Scope 1〜3)は orchestrator が inline で行う。herdr pane の観測・承認プロンプトへの応答・後始末は対話的で、implementer への handoff には向かない。Slice B(文書)は少量の散文なので同じく inline
- 期待と異なる観測(例: edits で cwd 外操作に問い合わせが出ず sandbox 拒否になる)は「検証失敗」ではなく事実として記録し、recipe には観測どおりに書く。マッピングの是非は別 issue
- Codex plan advisory(codex-cli 0.154.0)の 3 所見を採用: HIGH-1(否定テストの対象を writable root の外へ、結果を 3 値化)、HIGH-2(実効 approval policy / sandbox 設定を記録し、edits の「問い合わせる」は承認プロンプトの実観測を要件に。opt-in は検証時の config に条件付け)、MEDIUM-3(実引数は manifest ではなく子プロセスのコマンドラインから取る)
- E2E のタスクは reviewer 役の雛形に合わせ、成果物をファイル 1 つに限定した小さな作業にする(RESULT の EVIDENCE は sha256 と `file:line` のポインタ)

## Acceptance criteria

- [x] AC-1: `docs/evidence/codex-seat-permissions-2026-09-18.md` に autonomous 座席の記録がある: 使ったバージョン(codex / herdr / agmsg / ralph 一時バイナリの commit)、実効 codex config の該当キー、`spawned` イベントの `permission_mode`、子プロセスのコマンドライン(`--sandbox workspace-write --ask-for-approval never`)、判定(pass / fail / inconclusive)、起動直後の pane 抜粋(承認ダイアログなし)、送った TASK、`wait` の結果、受け取った RESULT の要点、stop / disband / status の結果。`grep -c '/Users/' docs/evidence/codex-seat-permissions-2026-09-18.md` が 0(redact 済み)
- [x] AC-2: 同 evidence に edits 座席の記録がある: 子プロセスのコマンドラインが `--sandbox workspace-write` のみ(`--ask-for-approval` なし)、cwd 内編集が承認なしで完了したこと、writable root 外への書き込みで観測された挙動と 3 値判定(pass / partial / inconclusive)、対象ファイルが事後に存在しないことの独立確認。recipe の「edits は cwd 外を問い合わせる」という記述は判定が pass のときだけ書く
- [x] AC-3: `docs/recipes/codex-seat-permissions.md` が root と `templates/base/` に byte 同一で存在し(`cmp`)、前提・スクラッチ config・2 モードの spawn コマンド・観測項目・本マシンの結果へのポインタ・自分の `ralph.toml` での opt-in 手順・後始末を含む。`docs/recipes/codex-setup.md` に recipe へのポインタ 1 行(root / template 同一)。`./scripts/check-sync.sh` pass
- [x] AC-4: `templates/base/ralph.toml` の `[org.permissions]` コメントに `docs/tech-debt/README.md` への参照がなく(`grep -c 'tech-debt' templates/base/ralph.toml` が 0)、recipe へのポインタがある。`internal/org/permissions.go` に検証日・codex バージョン・evidence への参照があり、`git diff main -- internal/org/permissions.go` がコメント行のみ。`/org` skill の permission 作法に「暫定制約」の語がなく(`grep -c '暫定制約' .claude/skills/org/SKILL.md` が 0)、`codex_verified` と recipe への言及がある。4 面 `cmp` 一致、`check-skill-sync.sh` pass
- [ ] AC-5: `go test ./internal/org/... -count=1` と `./scripts/run-verify.sh` が green(ロジック変更なし)
- [ ] AC-6: `codex_verified = true` への opt-in 手順が recipe で「検証時の実効 config とバージョン」に条件付けられ、config / バージョン変更時の再検証を求めている。検証で使った org の座席が残っていない(`ralph org status --org-id <各 id> --state-dir <scratch>` に active なし、`herdr agent list` / `herdr pane list` に該当 pane なし)。PR 本文は `Closes #155`

## Implementation outline

1. Slice A(inline、実機): 一時バイナリのビルド → スクラッチ cwd / config / state-dir → autonomous 座席の E2E(Scope 1)→ edits 座席の E2E(Scope 2)→ 後始末 → 生ログを scratchpad に保存 → redact して evidence 作成(Scope 3)→ commit `docs: record codex seat permission-mode live verification (autonomous / edits)`
2. Slice B(inline、文書): Scope 4〜5 → `sync-skills.sh` + cp → `check-skill-sync.sh` / `check-sync.sh` → commit `docs: add the codex seat permission recipe and reposition codex_verified as per-machine opt-in`
3. `./scripts/run-verify.sh` → post-implementation pipeline → `/cross-review` → `/pr`(`Closes #155`)

## Verify plan

- Static analysis checks: `./scripts/run-static-verify.sh`、`check-skill-sync.sh`、`check-sync.sh`
- Spec compliance criteria to confirm: AC-1〜AC-6 の grep / cmp。evidence の記述と manifest イベント(scratch state の `manifest.jsonl`)の整合
- Documentation drift to check: recipe の手順が evidence の実行内容と一致すること、`ralph.toml` / `permissions.go` / skill の 3 面が同じ位置づけ(既定 false、マシンごと opt-in)を語ること、`docs/specs/2026-08-01-org-runtime.md` の permission 記述と矛盾しないこと
- Evidence to capture: grep / cmp の出力、`git diff main -- internal/org/permissions.go` の出力

## Test plan

- Unit tests: 変更なし(`internal/org/permissions_test.go` の既存テストが fail-closed の維持を担保)。回帰として `go test ./internal/org/... -count=1`
- Integration tests: Slice A の実機 E2E そのもの(evidence が成果物)
- Regression tests: `./scripts/run-test.sh`
- Edge cases: (1) codex の初回 trust / login ダイアログ — 出たら記録して対処。(2) `ralph org wait` がタイムアウト — pane を読んで原因を記録し、TASK を簡略化して 1 回だけ再試行。(3) edits の cwd 外操作が sandbox で拒否される — 事実として記録。(4) herdr の pane が残る — `herdr pane close` で手動回収し evidence に記録
- Evidence to capture: `docs/evidence/codex-seat-permissions-2026-09-18.md`、`docs/reports/test-*.md`

## Risks and mitigations

- 実機 E2E がユーザー環境に副作用を残す(tmux pane、herdr workspace、agmsg team) → 使い捨て org_id とスクラッチ state-dir、終了時に disband + status / `herdr agent list` で残留確認
- codex 座席がスクラッチ cwd 外に書く → autonomous は `--sandbox workspace-write` で cwd 外は sandbox 拒否のはず。TASK の指示も cwd 内に限定。edits の cwd 外テストは `$TMPDIR` 配下の一時ファイルに限定
- codex クォータ消費 → TASK は小さく 2 回(モードごと 1 回)。再試行は 1 回まで
- 承認ダイアログが出て止まる → Design decisions どおり報告して止まる(autonomous で出た場合はそれ自体が検証結果)
- evidence に機微情報が混入 → redact のチェック(`grep '/Users/'`、UUID パターン)を AC-1 に含める

## Rollout or rollback notes

文書と evidence のみ。ロジック変更なし。recipe は seed-once で新規 `ralph init` にだけ届く(既存下流には `ralph upgrade` の seed advisory には出ない — 新規ファイルは advisory 対象外の可能性があるため、recipe の存在は release note で告知)。revert は PR 単位で安全。

## Open questions

なし。

## Deviation notes

- 2026-09-18 plan: dry-run で `spawned` Details が `permission_mode=<mode>` のみを記録することを確認(実引数なし)。Codex plan advisory の HIGH 2 件・MEDIUM 1 件を採用し Scope 1・2・2b、AC-1・2・6、Assumptions を改訂
- 2026-09-18 plan: `herdr workspace list` が socket 不在で失敗したため `herdr server`(ヘッドレス)をバックグラウンドで起動して検証する(herdr 0.7.5、protocol 17)。検証後に `herdr server stop` で止める
- 2026-09-18 work(実機、Scope 1〜3 = 7f299cc): 判明した環境依存の問題を evidence に P1〜P5 として記録。(P1) ユーザーの `~/.zshrc` の `alias codex="codex -m ..."` が herdr pane で展開され `--model` 二重エラーで spawn が 3 回失敗。`ZDOTDIR` 差し替えは `/etc/zshenv` が上書きするため無効、`~/.zshrc` は symlink で `sed -i` 不可(未変更)。最終的に herdr server を偽 HOME(alias なし rc、`.codex` に auth と最小 config、`.agents` / `.local` は symlink、`.config/herdr` のみ symlink)で起動して回避。(P2) idle 座席への `ralph org send` 本文が入力欄に残るため 3 秒後に Enter を追加送信。(P3) workspace-write 下で agmsg DB が readonly → `writable_roots` に `~/.agents/skills/agmsg/db` を追加して RESULT 到達。(P4) scratchpad が `/tmp` 配下で最初の cwd 外テストは無効 → 実ホームの絶対パスで再試行。(P5) gpt-5.5 の退役ダイアログ、実 config では `gpt-5.6-sol` へ自動移行
- 2026-09-18 work: 判定 — autonomous **pass**(A2)、edits **pass(機構)**: E1(実 config 複製、`approval_policy = "never"` 継承)はプロンプトなし = partial だが CLI の `--sandbox workspace-write` が `danger-full-access` を上書きすることを確認、E2(最小 config)は「回避しない」指示ではプロンプトなし = partial、E2b(escalation を要求させる指示)で承認プロンプトを観測し Esc で拒否 = pass。全 run で実ホームへの書き込みは拒否され、対象ファイルは存在しない
- 2026-09-18 work: Slice B(文書)= recipe を root と templates/base に byte 同一で追加、`codex-setup.md` に See also、`ralph.toml` コメント 2 箇所を recipe ポインタ + 継承の注意に置換、`permissions.go` コメントに検証日 / バージョン / evidence / opt-in 設計を追記(コード不変)、skill 4 面の permission 作法を更新。後始末: 実 `~/.codex/config.toml` を backup から復元(一時追記した trust 行を除去)、複製 auth.json 削除、herdr server 停止、`~/.zshrc` 無変更
- 2026-09-18 work: 派生 issue 候補(PR 後に起票): (a) `ralph doctor` で `codex` / `claude` の shell alias を検出して warn、(b) `ralph org send` の Enter timing、(c) codex 座席の agmsg DB writable root を ralph が `-c sandbox_workspace_write.writable_roots` で自動付与するか doctor で検査、(d) 退役モデルの自動移行を receipts の `honored` で捕捉

## Progress checklist

- [x] Plan reviewed
- [x] Branch created
- [x] Implementation started
- [ ] Review artifact created
- [ ] Verification artifact created
- [ ] Test artifact created
- [ ] PR created
