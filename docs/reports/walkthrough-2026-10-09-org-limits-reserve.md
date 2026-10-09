# Walkthrough: org-limits-reserve

- Date: 2026-10-09
- Plan: docs/plans/archive/2026-10-08-org-limits-reserve.md(この PR の最後のコミットで archive に移す)
- Branch: feat/org-limits-reserve(merge-base 51855166。origin/main はこの間に c3a9242e へ進んだが、`git merge-tree` でぶつからない)
- Diff(HEAD 9972e67f、この walkthrough を除く): 40 files、+5,889 / -137。テスト以外の Go が +902 / -94、テストが +2,723、報告・insight・計画が +1,888
- PR: #212

## 何を変えたか

全 org をまたぐ上限 `max_orgs`(既定 10)と `max_total_seats`(既定 30)を、座席を立てるときに台帳のロックの下で守らせる。org の担当範囲を `--reserve` で予約でき、走っている他の org と重なれば start を拒否し、disband で解く。

## 読む順

1. `620458d7` S1 設定。`internal/config/config.go` の 2 つの項目、`templates/base/ralph.toml`、`scripts/ralph-config.sh` とその写し、同期テスト
2. `11fc2261` S2 org の層
   - `internal/org/reserve.go`(新規): パスの規則、重なり、走っている org の判定
   - `envelope.go`: 2 つの上限の判定
   - `spawn.go`: ロックの中の判定、予約の記録、すでに立っている leader への予約
   - `seat.go`: `scope_reserved`
   - `verbs_all.go`: disband の対象
   - `verbs.go`: 補償で予約を書き戻す
3. `8fe95acd` 上限が 0 のときも拒否する(`max_seats` と同じ fail-closed)。scope のゲートの文に `--reserve` を足した
4. `78e46f36` S3 CLI。`--reserve`、`ralph org status` の予約の表示、全体の上限を main worktree の `ralph.toml` から読むこと(`statedir.go` の `MainWorktreeRoot`)
5. `fd3e3b47` S4 文書(`/org` skill、README)
6. cross-review の指摘の直し(PR 本文の表を参照)
   - `975df92b` と `afcbc6c2`: 動いていない leader への予約で `max_orgs` を判定する
   - `a94c914f`: 補償を、台帳のロックの下で読み直してから判断する
   - `83aec44e`: 予約を書き戻す規則をそろえ、pane の経路も同じ規則を呼ぶ
7. 実装後のパイプライン(4 回)の報告とテストの追加: `c3a95c48`、`066bf282`、`69cb7d19` がテストの追加。ほかは報告、sync-docs、triage の記録

## 計画からの逸脱(本文は承認のまま。進捗の節に記録)

- 上限が 0 以下のときは拒否する(implementer の最初の案は「上限なし」だった)
- すでに立っている leader への予約の拒否は `rejected` を書かない(書くと leader が inactive に見える)
- 全体の上限を main worktree の `ralph.toml` から読むのは spawn と start だけ
- cross-review の 3 回の直し(上の 6)。どれも AC1 と AC15 にコードを合わせるもので、計画の本文は変えていない
- self-review の C4-1(`disband --force` のあとの stop の打ち直しでも予約を戻す)は、ユーザーが「戻す」を選んだ

## 変えていないもの

- org ごとの `max_seats`、watchdog(範囲の外への変更は今の scope_change の ALERT で知らせる)、台帳の座席の判定(`Roster`)、`ralph status`

## 下流への届き方

| 所有権 | ファイル | upgrade での扱い |
|---|---|---|
| バイナリ | `internal/org/`、`internal/cli/`、`internal/config/` | バイナリを更新すると効く |
| core | `/org` skill(`.claude/skills/`、`.agents/skills/`)、`scripts/ralph-config.sh`、`docs/quality/quality-gates.md` | 置き換わる |
| seed | `templates/base/ralph.toml`(新しい 2 つのキーとその説明) | `ralph init` の新しい scaffold に入る。既存の `ralph.toml` に書かなければ既定値が使われる |

## 確かめたこと

- `./scripts/run-static-verify.sh` は rc 0(golangci-lint 0 件)。`./scripts/run-test.sh` は 4 回とも shell 2,047 件、Go 8 パッケージで、すべて pass。`go test -race` も pass
- mutation: 各回で生き残りは等価なものだけ
- 実バイナリの dry-run(herdr と agmsg は PATH から外した)で、重なる予約の拒否、不正なパスの拒否、上限の拒否と disband の案内、status の表示を確かめた(verify)

## 確かめていないこと

- 本物の herdr と、別プロセスどうしの競合
- 補償の窓(close の失敗とほかの org の start が重なる場合)の実地の再現
- 古い ralph が実際に書いた台帳(テストは手で書いた記録で作った)
