# mojibake-postedit-guard

- Status: Done — PR #25 (https://github.com/yoshpy-dev/harness-engineering-scaffolding-template/pull/25)
- Owner: Claude Code
- Date: 2026-04-17
- Related request: Claude Code の文字化け (U+FFFD 混入) 対策を記事（https://nyosegawa.com/posts/claude-code-mojibake-workaround/）に沿って導入
- Related issue: N/A
- Branch: chore/mojibake-postedit-guard

## Objective

Claude Code の `Write` / `Edit` / `MultiEdit` ツールが SSE チャンク境界でマルチバイト文字を壊し、ファイルに U+FFFD (`\xEF\xBF\xBD`, Unicode Replacement Character) を混入させる問題に対し、本リポジトリと本テンプレートからスキャフォールドされるプロジェクトの両方で **検出・再試行を強制するフック** を組み込む。恒久対策はアップストリーム修正 (GitHub Issue #43746 系) に委ね、当面の暫定対策として PostToolUse フックで検出→`exit 2` による Claude への actionable feedback を提供する。

## Scope

- `.claude/hooks/check_mojibake.sh` を新設（POSIX sh・既存 `lib_json.sh` 再利用）。
- `.claude/hooks/mojibake-allowlist` を新設（空ファイル＋コメントヘッダのみ）。glob パターンを 1 行 1 件で並べる opt-out 機構。
- `.claude/settings.json` の `PostToolUse` に同フックを登録し、`matcher` を `Edit|Write` → `Edit|Write|MultiEdit` に拡張。既存 `post_edit_verify.sh` と**連鎖実行**する（同一 matcher 配列に 2 エントリ）。
- `templates/base/.claude/hooks/check_mojibake.sh` / `templates/base/.claude/hooks/mojibake-allowlist` / `templates/base/.claude/settings.json` を同じ内容でミラーし、`scripts/check-sync.sh` でドリフトが出ないことを確認する。
- `scripts/verify.local.sh` を新設（**ルート限定**）し、`run-verify.sh` から自動で呼ばれるようにする。以下を実行：
  - `shellcheck .claude/hooks/*.sh`
  - `sh -n` / `bash -n` syntax check on all hook scripts
  - `jq -e . < .claude/settings.json` と templates 側の同 JSON
  - `tests/test-check-mojibake.sh` 実行（exit code 集約）
- `tests/test-check-mojibake.sh` を新設し、実ペイロードのスキーマに沿った JSON を `printf` で構築して hook に流す。`jq` 有り/無し両パスを `PATH` を制限して切り替える形で検証。
- `scripts/check-sync.sh` の `ROOT_ONLY_EXCLUSIONS` に `scripts/verify.local.sh` と `tests/test-check-mojibake.sh` を追加（これらは repo 専用、テンプレ非配布）。
- `AGENTS.md` の Repo map セクションに `.claude/hooks/check_mojibake.sh` と allowlist の 1 行注記を追加（撤去条件：Issue #43746 解決）。

## Non-goals

- Anthropic SDK 本体のパッチング、あるいは Claude Code のバージョン固定。恒久対策は本プランの対象外。
- Claude の**応答テキスト**そのもの、WebSearch/WebFetch の外部レスポンスに混入する U+FFFD への対策（記事の対策限界と同じ）。
- PreToolUse での先回り防止。Claude Code の PostToolUse 前に書き込みが完了しているため、PostToolUse で検出→再試行させる方針で十分。
- `~/.claude/settings.json`（ユーザーグローバル）の編集。プロジェクトスコープで閉じる。
- 自動修復（U+FFFD を元文字に復元する試み）。復元は不可能なので、Claude に再書き込みさせる方針。
- 編集前/編集後の差分ベース検出（diff-based introduced-byte 判定）。実装コストが allowlist より高く、今回は allowlist で代替。

## Assumptions

- 既存フックは `#!/usr/bin/env sh` 系で書かれており、`jq` があれば使い、無ければ sed フォールバックする（`lib_json.sh` が提供）。本フックは**jq 必須**に格上げし、`jq` が無い環境では fail-closed（stderr 警告＋exit 0 でパイプラインは通すが、warning を `.harness/state/mojibake-jq-missing` ファイルに残し、後段で可視化）する設計にする。`exit 2` で止めると CI 未整備環境を壊すため、fail-**open-with-warning** を採用。理由：既存ガード無しより弱化しないため。
- `grep` が UTF-8 バイト列 `\xEF\xBF\xBD` を `LC_ALL=C` 下でバイナリセーフに検索できる（macOS BSD grep / GNU grep いずれでも動く）。
- PostToolUse フックが exit 2 を返すと stderr が Claude にフィードバックされ、Claude はファイルを再編集できる（Claude Code 公式仕様）。
- Claude Code hook の PostToolUse payload スキーマは `tool_input.file_path` をトップレベルで持つ（Edit/Write/MultiEdit 共通、公式ドキュメント準拠）。Edit/Write/MultiEdit の実ペイロードサンプルを `tests/fixtures/payloads/` に保存し、将来スキーマ変更が入っても test が fail するようにする。
- スキャフォールドされるプロジェクトでも `.claude/hooks/` と `.claude/settings.json` がそのまま使えることをテンプレート同期前提とする（既存運用）。
- `verify.local.sh` は本リポジトリ専用（テンプレに配布しない）。スキャフォールド先プロジェクトは独自に `scripts/verify.local.sh` を書く前提で、`check-sync.sh` の除外に追加する。

## Affected areas

- `.claude/hooks/check_mojibake.sh` (新規、ルート＋templates の 2 箇所)
- `.claude/hooks/mojibake-allowlist` (新規、ルート＋templates の 2 箇所)
- `.claude/settings.json` (`PostToolUse` matcher 拡張 + hook 追加、ルート＋templates の 2 箇所)
- `scripts/verify.local.sh` (新規、**ルートのみ**)
- `tests/test-check-mojibake.sh` (新規、**ルートのみ**)
- `tests/fixtures/payloads/edit.json` / `write.json` / `multiedit.json` (新規、**ルートのみ**)
- `scripts/check-sync.sh` の `ROOT_ONLY_EXCLUSIONS` に上記 repo-only ファイルを追加
- `AGENTS.md`（Repo map セクションに 2 行追記）

## Acceptance criteria

- [ ] `./.claude/hooks/check_mojibake.sh` は POSIX sh で動作し、stdin から JSON を読んで `jq` で `tool_input.file_path` を抽出する（`jq` 必須、無い場合は warning＋exit 0）。
- [ ] `jq` が `PATH` に無いケースで `tests/test-check-mojibake.sh` を走らせたとき、exit 0 かつ `.harness/state/mojibake-jq-missing` が作成される（fail-open-with-warning）。
- [ ] 対象ファイルに U+FFFD が含まれ、かつ allowlist にマッチしない場合、stderr に「U+FFFD detected in <path>. Re-read and rewrite the corrupted sections.」を出して `exit 2`。
- [ ] allowlist (`.claude/hooks/mojibake-allowlist`) にマッチするパス（glob 1 行 1 件）、または対象ファイルが空 / 存在しない / U+FFFD 無しなら `exit 0`。
- [ ] フック自身と `tests/fixtures/**` は allowlist に既定で載り、自己検知ループが起きない。
- [ ] `.claude/settings.json` の `PostToolUse` に既存 `post_edit_verify.sh` と並んで 2 エントリ目として登録され、matcher が `Edit|Write|MultiEdit` に拡張されている。
- [ ] `templates/base/.claude/hooks/check_mojibake.sh` / `mojibake-allowlist` / `settings.json` がルートと byte-for-byte 同一で、`./scripts/check-sync.sh` PASS。
- [ ] `tests/test-check-mojibake.sh` が以下 **6 ケース**で green：(1) U+FFFD 入り fixture → exit 2、(2) クリーン UTF-8 fixture → exit 0、(3) 存在しないパス → exit 0、(4) allowlist マッチで U+FFFD 入りでも exit 0、(5) `jq` 欠落環境で exit 0 かつ warning marker 作成、(6) Edit/Write/MultiEdit 3 種類の payload fixture で `file_path` が正しく抽出される。
- [ ] `scripts/verify.local.sh` が新設され、以下を順に実行して exit code を集約：`shellcheck .claude/hooks/*.sh` → `sh -n` → `jq -e . < .claude/settings.json` × ルート/templates → `tests/test-check-mojibake.sh`。
- [ ] `./scripts/run-verify.sh` を叩いたとき、`verify.local.sh` が呼ばれ全チェック PASS。
- [ ] `./scripts/check-sync.sh` が PASS（`scripts/verify.local.sh` と `tests/test-check-mojibake.sh` と `tests/fixtures/payloads/` が ROOT_ONLY 除外に追加されている）。
- [ ] フック自身 (`.claude/hooks/check_mojibake.sh`) のソースに U+FFFD リテラルを含まない（`printf '\357\277\275'` など生成して比較）。
- [ ] AGENTS.md の Repo map に 2 行注記が入り、将来の読み手が意図（暫定対策・記事参照・Issue #43746 解決時に撤去）と allowlist の存在を把握できる。

## Implementation outline

1. **フック実装** (`.claude/hooks/check_mojibake.sh`)
   - shebang `#!/usr/bin/env sh`, `set -eu`.
   - `jq` presence check：`command -v jq >/dev/null 2>&1 || { warn+marker; exit 0; }`.
   - payload を stdin から読み、`jq -r '.tool_input.file_path // empty'` で抽出。空なら exit 0。
   - allowlist 読み込み（存在しなければ空配列扱い）、各行に対して `case` の glob マッチで skip 判定。
   - `FFFD="$(printf '\357\277\275')"` として literal byte を組み立て、`LC_ALL=C grep -q "$FFFD" "$file_path"` で検知。
   - 検知時 `printf` で stderr にメッセージを出して `exit 2`、未検知時 `exit 0`。
2. **allowlist ファイル** (`.claude/hooks/mojibake-allowlist`)
   - コメントヘッダ＋以下のデフォルトエントリ：
     - `.claude/hooks/check_mojibake.sh`（自己検知防止）
     - `tests/fixtures/**`（テストフィクスチャ）
     - `docs/plans/**/2026-04-17-mojibake-postedit-guard.md`（本プランが U+FFFD を含むケースに備え — 念のため）
3. **settings.json 登録**（ルート）
   - `PostToolUse` の該当 matcher を `Edit|Write` → `Edit|Write|MultiEdit` に変更。
   - `hooks` 配列に 2 エントリ目として `./.claude/hooks/check_mojibake.sh` を追加。`post_edit_verify.sh` が先・`check_mojibake.sh` が後。
4. **templates/base ミラー** — 上記 3 ファイルを `templates/base/.claude/` 配下にコピー／編集し、byte-for-byte 揃える。`chmod +x` を忘れない。
5. **実ペイロードフィクスチャ** (`tests/fixtures/payloads/`)
   - `edit.json` / `write.json` / `multiedit.json` を作成。Claude Code 公式ドキュメントのスキーマに基づき、`tool_name`、`tool_input.file_path`、`tool_input.content` 等を含む最小構造。エスケープされた `\"` を含む値も 1 ケース入れて sed フォールバック脆弱性を再現可能にする。
6. **verify.local.sh** (`scripts/verify.local.sh`) を新設
   - 各チェックを実行し、`|| fail=1` で集約。末尾で `exit "$fail"`.
   - 冒頭にコメントで「run-verify.sh から自動起動される」「テンプレには配布しない」と明記。
7. **tests/test-check-mojibake.sh**
   - 6 ケースを `assert_exit` ヘルパーで回す。`PATH` を一時的に `/nonexistent` に絞って `jq` 欠落をシミュレートするケースを含む。
   - `mktemp -d` でワークスペースを作り、後片付けは `trap`.
8. **check-sync.sh 更新** — `ROOT_ONLY_EXCLUSIONS` に `scripts/verify.local.sh`、`tests/test-check-mojibake.sh`、`tests/fixtures/payloads/` を追加。
9. **AGENTS.md 注記** — Repo map に `.claude/hooks/check_mojibake.sh` と `.claude/hooks/mojibake-allowlist` の 2 行を追加。末尾に「暫定対策 — Issue #43746 解決後に撤去」コメント。
10. **動作確認** — 実セッションで本プランファイル（日本語多め）を編集した直後、フックが exit 0 で静かに通ることを `run-verify.sh` と合わせて確認し、evidence に保存。

## Verify plan

- **Static analysis checks**:
  - `shellcheck .claude/hooks/check_mojibake.sh`
  - `shellcheck templates/base/.claude/hooks/check_mojibake.sh`
  - `shellcheck scripts/verify.local.sh`
  - `shellcheck tests/test-check-mojibake.sh`
  - `sh -n` / `bash -n` 構文チェックを verify.local.sh 内で実行
  - `jq -e . < .claude/settings.json`
  - `jq -e . < templates/base/.claude/settings.json`
- **Spec compliance criteria to confirm**: 受入基準 13 項目すべてチェック済み。POSIX sh で書かれ bash-ism を含まない。`jq` 必須化が実装に反映されている。allowlist 仕様が README/フック冒頭コメントに記載。
- **Documentation drift to check**:
  - AGENTS.md の Repo map に 2 行注記
  - `scripts/check-sync.sh` の `ROOT_ONLY_EXCLUSIONS` 更新
  - 差分がテンプレ/ルート間でゼロ
- **Evidence to capture**:
  - `./scripts/run-verify.sh` 実行ログ（verify.local.sh が起動したことが分かる行）
  - `./scripts/check-sync.sh` 出力
  - `shellcheck` 結果
  - `tests/test-check-mojibake.sh` の 6 ケース pass ログ

## Test plan

- **Unit tests** (`tests/test-check-mojibake.sh`)：
  - Case A: U+FFFD 入り fixture → exit 2、stderr に期待メッセージ
  - Case B: クリーン UTF-8 (日本語含む) fixture → exit 0
  - Case C: 存在しないパス → exit 0
  - Case D: allowlist マッチ（例：`tests/fixtures/...`）で U+FFFD 入りでも exit 0
  - Case E: `PATH=/nonexistent` で `jq` 欠落 → exit 0 かつ `.harness/state/mojibake-jq-missing` 作成
  - Case F: `tests/fixtures/payloads/{edit,write,multiedit}.json` を流し、`file_path` が正しく抽出・判定される
- **Integration tests**: 実セッションでこのプランファイル（日本語多め）の編集直後にフックが exit 0 で通ることを手動確認し、evidence 欄にログ断片を貼る。
- **Regression tests**: 既存 `post_edit_verify.sh` の挙動（`.harness/state/needs-verify` の touch、additionalContext 出力）が不変であることを、matcher 拡張後にも確認。`tool_failures.count` のリセットが継続動作すること。
- **Edge cases**: 空ファイル、バイナリ拡張子（`.png`/`.pdf`）、1 バイトのみのテキスト、JSON に `file_path` が無い payload、エスケープされた `\"` を含む値、`file_path` が相対・絶対・チルダ、フック自身のパス、allowlist グロブのワイルドカード。
- **Evidence to capture**: test レポートに 6 ユニット + 1 integration + edge cases の pass/fail を表で記録。

## Risks and mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| 既存 U+FFFD を含むファイル編集で無限ループ | High (Codex #1) | allowlist (`.claude/hooks/mojibake-allowlist`) 導入。自身と fixture は既定で除外。ユーザーは 1 行追加で任意パスを opt-out 可能。受入基準に明記。 |
| `jq` 欠落環境で silent no-op | High (Codex #2) | `jq` 必須化し、欠落時は fail-open-with-warning（stderr 警告＋marker ファイル作成）。警告メッセージは Claude 可読＋test でも検証。 |
| MultiEdit payload スキーマ変更 | Medium (Codex #2) | 実ペイロード fixture を `tests/fixtures/payloads/` に保存し、test で抽出結果を assert。スキーマ変化があれば CI で検知。 |
| run-verify.sh がフック test を走らせない | Medium (Codex #3) | `scripts/verify.local.sh` 新設。`run-verify.sh` が `verify.local.sh` を自動起動する既存仕組みを利用。受入基準に「verify.local.sh から呼ばれる」を明記。 |
| フック自身に U+FFFD リテラルが混入し自己検知 | High | `printf '\357\277\275'` でバイト列を生成。自身パスを allowlist に既定で入れる。 |
| macOS BSD grep と GNU grep の挙動差 | Low | `-q` と `LC_ALL=C` と literal byte だけを使用。shellcheck でカバー。 |
| PostToolUse exit 2 が Claude に伝わらない旧環境 | Low | Claude Code 公式仕様に準拠。前提バージョンを AGENTS.md に 1 行記載。 |
| `post_edit_verify.sh` との実行順副作用重複 | Low | 既存フックは `.harness/state/needs-verify` の touch のみ。check_mojibake.sh は read-only。 |
| テンプレと本体のドリフト | Medium | `check-sync.sh` で PR ごとに強制。ROOT_ONLY 除外は必要最小限に留める。 |
| allowlist の glob 表現が貧弱で過剰除外 | Low | glob は shell `case` マッチで `*` と `?` のみ。シンプルさを優先。複雑化が必要になったら別プランで対応。 |

## Rollout or rollback notes

- **Rollout**: 本 PR マージ後、次回セッションから自動で全 Edit/Write/MultiEdit に対してチェックが走る。既存ファイルに対するバックフィル検査は不要（影響は以後の編集のみ）。既存 U+FFFD を含むファイルがある場合は allowlist に 1 行追加するだけ。
- **Rollback**: `.claude/settings.json` と `templates/base/.claude/settings.json` の該当 hook エントリを削除し、`check_mojibake.sh` / `mojibake-allowlist` / `verify.local.sh` / test / fixture を両サイドから消す。State ファイル (`.harness/state/mojibake-jq-missing`) も削除。`check-sync.sh` の ROOT_ONLY_EXCLUSIONS も元に戻す。
- **恒久対策への移行トリガー**: GitHub Issue #43746 が Claude Code 安定版でクローズされ、手元で再現しないことを 1 週間確認できたら本フックを撤去。撤去時は AGENTS.md の注記も削除。
- **Partial failure scenarios**: 実装が途中で止まった場合、(a) `verify.local.sh` のみ作成済みで test 未作成 → run-verify.sh が exit 1 で明示失敗するので気付ける。(b) フックのみ作成済みで allowlist 未作成 → フックは allowlist 欠落時も exit 0 で通す設計にする（受入基準に追加）。(c) settings.json 登録まで済んで test 未実装 → 実害は出ないが verify.local.sh が warn を出す。

## Open questions

- 誤検知時の feedback メッセージに該当行番号を含めるべきか（初版はファイルパスのみ）。運用で必要性を見て改善。
- allowlist の保存先を `.claude/hooks/` ではなく `.claude/hooks/mojibake/allowlist.txt` 等のサブディレクトリに移すべきか。今回は 1 ファイルで十分と判断。
- `matcher` を `Edit|Write|MultiEdit` に拡張することで既存 `post_edit_verify.sh` が MultiEdit にも反応する点は意図した副次効果。self-review で明記。

## Progress checklist

- [x] Plan reviewed
- [x] Branch created (chore/mojibake-postedit-guard)
- [x] Implementation started
- [x] Review artifact created (`docs/reports/self-review-mojibake-postedit-guard.md`)
- [x] Verification artifact created (`docs/reports/verify-mojibake-postedit-guard.md`)
- [x] Test artifact created (`docs/reports/test-mojibake-postedit-guard.md`)
- [x] PR created (#25)
