# Self-review report: org-feature-worktree

- Date: 2026-10-09
- Plan: docs/plans/active/2026-10-09-org-feature-worktree.md(承認済み、digest 56e435bfa976)
- Reviewer: reviewer subagent (Claude)、パイプライン 1 回目(cycle 1、上限 2)。初回は 2a544af9 の時点、再実行は 8aae7ce2・2c1bb13b のあと(cross-review の前の直しなので cycle は増えない)
- Scope: diff の品質だけ。`git diff 765da6bd...HEAD`(9 コミット、27 ファイル、+6040/-367)。テスト・静的解析・仕様適合・文書の整合の検査は /test・/verify・/sync-docs の担当で、ここでは行っていない。ただし依頼された「雛形と /org skill の文言が、コードのしていることと合っているか」は、文言の根拠をコードで確かめた
- 番号の付け方: 初回の finding は M1〜M2(MEDIUM)、L1〜L5(LOW)、再実行で足した finding は N1〜N3(LOW)。/verify・/test・tech-debt が番号で指す場合は、この報告の番号を使う。報告を上書きしても番号は付け替えない

## Evidence reviewed

- `git log --format='%h parent=%p'` で 9 コミットが一直線(61b05d3d → a321d1a7 → 9d7a85ae → f27188d7 → 216b2cdd → 944daf6f → aaaef61a → af138af6 → d849e73d)であることを確かめた
- 全文を読んだ非テストのコード: `internal/org/feature.go`、`internal/org/split.go`、`internal/org/reserve.go` の差分、`internal/org/spawn.go` の差分(`checkSpawnInput`、`spawnPrecheckErr`、`idempotentRespawn`、`idempotentRespawnDecision`、`spawnCapacityErr`)、`internal/org/verbs.go` の差分(`reserveAgain`、`releasedReservation`)、`internal/org/statedir.go` の `FeatureRepoRoot`、`internal/cli/org.go` の差分(`newOrgStartCmd`、`checkOrgStartPlanInput`、`runOrgStartPlan`、status の表示)、`internal/org/prompts/leader.md`
- `scripts/ralph-worktree.sh` の `state_path`、`git_common_dir`、`ensure_worktree`、`cleanup_worktree` を読み、`feature.go` の呼び方(引数、cwd、stdout と stderr の扱い、記録の読み方)と突き合わせた
- `spawnPrecheckErr` と `Spawn` のロック下の閉包を並べて読み、順序(冪等な返り、retired key、envelope、permission、AC-2b、stale の補償、capacity)を 1 行ずつ比べた。`TestSpawnPrecheckErr_MatchesSpawn`(16 ケース)が何を固定しているかも読んだ
- `/org` skill の追加部分(「機能ごとの org」「担当範囲の予約」の追加行、動詞の表の `start` と `status` の行、「Leader 運用 2 経路」、サイクルの 1・2・7 番)を、`feature.go`・`split.go`・`identifier.go`・`spawn.go` の該当箇所と照合した。4 面は `cmp` で同一
- 機械的な確認: `git diff --check` は空。追加行に U+FFFD、`fmt.Print` 系のデバッグ出力、TODO、FIXME、secret らしい文字列はない。`docs/evidence/org-feature-worktree-live-2026-10-09.md` に `$HOME` の絶対パスはない(`/tmp/rs5` と `/opt/homebrew` のみ)
- `docs/tech-debt/README.md` は差分にない。台帳の行のうち、この差分が触れた関数を指すものを `grep` で探して照合した(L2)

## 依頼された点の結論

1. 共有の先読み(`checkSpawnInput`、`spawnPrecheckErr`、`idempotentRespawnDecision`)がロック下の経路と同じ判定をするか: 同じ。`spawnPrecheckErr` の順序は `Spawn` の閉包と一致する(冪等な返り → `retiredRoleConfigErr` → `ValidateSpawnEnvelope` → `permissionArgsForDriver` → `autonomousScopeGateErr` → stale の座席を `spawn_failed` 1 件として数える → `spawnCapacityErr`)。`idempotentRespawn` は判定を `idempotentRespawnDecision` に切り出して呼ぶだけになり、先読みと同じ関数を通る。`reject()` が書く `rejected` と receipt は先読みにはない(書かない側が正しい)。`Spawn` の doc に「steps 1〜6 を変えたら `spawnPrecheckErr` も直す」と書いてあり、`TestSpawnPrecheckErr_MatchesSpawn` が同じ台帳で両者のエラー文字列の一致まで見る。ずれは見つからなかった
2. 結びつきの Details の書き出しと読み戻し(`scopeReservedDetails`、`reservationFromEvent`、`featureBindingFromTokens`): 閉じている。`key=value` でない最初の語(補償の注記は必ず `restored:` で始まる)で読むのを止めるので、注記の本文で結びつきを偽造できない。キーの欠け・空の値・重複は「不完全な結びつき」として読み、`sameFeature` はそれを何とも等しいとしない(fail closed)。`reservedPathsFromDetails` は最初の空白で切るままなので、古いバイナリも予約のパスを読める。`Worktree` を載せた org レベルのイベントは、`Roster`(状態イベントだけを見る)にも `watch.go` の座席ごとの `git status` にも読まれない
3. 補償(`reserveAgain`、`releasedReservation`): `releasedReservation` が `Reservation` ごと返し、`scopeReservedEvent` が結びつきのトークンと `Worktree` を書き戻す。`ActiveReservation(now) != nil`、`now[d] != before[d]`、`startsOrg` の走査という 3 層の条件は変わっていない。結びつきの有無で判定が分かれる箇所はなく、別の run の結びつきを戻す入力は見つからなかった
4. worktree の使い回しの検査とエラー文(`checkFeatureWorktreeReuse`): `canonical_ref`・パス(symlink を解決して比較)・ブランチ・kind と、実際のチェックアウトまで見る。記録が読めないときも拒否に倒れる。エラー文が出す `cleanup` の案内は、`cleanup_worktree` が worktree のディレクトリがない場合も扱える(`[ -d "$path" ]` で飛ばす)ので、どの拒否でも使える。ensure の失敗に付ける固定の案内は L1
5. スクリプトの呼び出し(`runWorktreeScript`): 引数は argv で渡り(shell を通さない)、`cmd.Dir` は main worktree のルート、stderr は取り込んで終了コードと一緒にエラーに入れる。スクリプトがないときは stat で先に止め、`errors.Is` で案内の文を分けている。`git_common_dir` は絶対パスを返すので、`Lookup` が読む記録のパスも cwd に依存しない
6. 雛形と skill の文言: 機構の挙動と合っていない点が 2 つある(M1、M2)。細かい食い違いは L3

## Findings(初回、2a544af9 の時点。現況は「再実行」の節)

<!-- Area recommended values: naming, readability, unnecessary-change, typo,
     null-safety, debug-code, secrets, exception-handling, security, maintainability -->

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| M1 | MEDIUM | maintainability | 分割計画の slug は 30 文字まで通り、skill もそう書くが、既定の座席を立てられる長さは 20 文字まで。org_id の既定は slug なので、`herdr` の agent 名の上限(org_id + `_` + seat_id が 32 文字まで)に先に当たる。slug が 26〜30 文字だと start が leader の spawn で拒否される(エラーは seat_id="leader" の組み合わせを責めるが、skill の「30 文字まで」とは食い違う)。21〜25 文字だと leader は立ち、worktree もできたあとに、leader が雛形どおり `implementer`(11 文字)を spawn した時点で拒否される(`reviewer` なら 23 文字まで、skill の例の `reviewer-1` なら 21 文字まで) | `internal/org/spawn.go` の `checkSpawnInput`(`len(p.OrgID)+1+len(p.SeatID) > maxHerdrAgentNameLen`、32)。`internal/org/split.go` の `newSplitFeatureDraft` は `ValidateIdentifier`(`identifierPattern` は最大 30 文字)だけ見る。`.claude/skills/org/SKILL.md` 「分割計画」の「30 文字まで(既定の org_id になる)」。雛形と skill に 32 文字の組み合わせ上限の記述はない(`grep '32 文字\|agent-name'` は 0 件)。実機の確認(slug は `hello`)は短い値で、この範囲を通っていない | 計画の読み込みか start の先読みで、既定の座席 id(`implementer`、`reviewer`)との組み合わせを検査して早く拒否する。または slug の上限を 20 文字にして skill と同じ数にする。どちらにしても skill の「30 文字まで」は直す。`--org-id` で短い値を渡す逃げ道はエラー文に書くと親切 |
| M2 | MEDIUM | maintainability | 出荷する skill が、まだ存在しない director を 4 か所で説明している。「分割計画を承認するのは人。今後入る director も、下書きは書くが承認を代行しない」(「承認」)、「director の配下に置ける org」の節 1 つ(「今後入る director の配下に置けるのは、`start --plan` で立てた org だけになる」)、「Leader 運用 2 経路」の (A) と (B) の各 1 文(「director の配下にも置けない」)。ralph にはコードも CLI もなく、下流の利用者は使えない。skill は `templates/base/` 経由で `ralph init` が配る 4 面(`.claude/` と `.agents/`、root と `templates/base/`)すべてに入っている | `grep -n director .claude/skills/org/SKILL.md` が 4 か所(「承認」の節、「director の配下に置ける org」の節、「Leader 運用 2 経路」の (A) と (B))。差分の前の版(765da6bd)には 0 件。`grep -rn director internal --include='*.go'` に director の実装はない。計画の Scope(`/org` skill の項)は「分割計画の形式と例、承認の記録のしかた、`start --plan`、merge のあとの後始末、小さい変更は標準フロー」で、director の節は含まない。経路(B)の文は「director の配下にも置けない」と、使えない制約を出荷物に書いている | 「director の配下に置ける org」の節と、承認の節・2 経路の director への言及を skill から外す(`templates/base/` と `.agents/` の写しも `scripts/sync-skills.sh` で揃える)。director の決まりは仕様(`docs/specs/2026-10-07-org-multi-org-director.md`)に残っていて、入る段で skill に足せる。入れておく価値があると判断するなら、「未実装」と明記し、「承認を代行しない」のような将来の方針は書かない |
| L1 | LOW | exception-handling | `StartFeature` は `ensure` の失敗すべてに「main worktree を clean な default branch にして打ち直す」を付ける。`ensure_worktree` が返す失敗のうち、`branch already exists without matching state`、`worktree path already exists without matching state`、`state collision for id` は、main を clean にして打ち直しても通らない(別のブランチ名、残った worktree やブランチの片付けが要る)。計画の Edge cases に「slug と同じ名前のブランチがすでにある」が挙がっているが、この経路のテストはない | `internal/org/feature.go` の `StartFeature`(`fmt.Errorf("%w; make the main worktree %s a clean checkout of the default branch and run start again", err, root)`)。`scripts/ralph-worktree.sh` の `ensure_worktree` の `die` 3 種。`TestStartFeature_EnsureFailureRefused` は dirty の 1 種だけ。`grep 'already exists' internal/org/feature_test.go` は 0 件 | 案内の文を「スクリプトの文が main の状態を指しているときは」の条件付きにするか、clean/default branch を指す失敗(`validate_clean_base` の文言)のときだけ付ける。少なくとも文頭を「原因が main の checkout なら」にして、ほかの原因ではスクリプトの文に従うよう書く。ブランチの衝突のケースを 1 つ足す |
| L2 | LOW | maintainability | 台帳の行が指す関数をこの差分が動かし、行のトリガーを満たしたのに、行も該当の文も更新されていない。(a) 行「Wording findings of org-limits-reserve cycle 2」の C2-1: `idempotentRespawn` の doc が「reservation is decided first」と「max_orgs is decided first」を別の段で言う。この差分はその doc の第 1 段を書き換えた(結びつきの説明を足した)が、「decided first」はそのまま残り、同じ矛盾が新しい文にも続く。`Spawn` の doc の手順 1(「the reservation is decided first, after max_orgs」)も同じ。(b) 同じ行の C2-3 と、行「Test gaps left by the org-limits-reserve cycle 2 fix」の T2-1・T2-2 は `idempotentRespawn` の `if !seat.Active` を指すが、条件は `idempotentRespawnDecision` に移った。(c) 行「Code-shape findings of org-limits-reserve」の F-2 のトリガー「`scopeReservedEvent` の次の変更」は、この差分が引数を `Reservation` に変えたことで満たされた。空のタイムスタンプを渡す形(`scopeReservedEvent("", ..., true)` の dry-run 側)は残っている。F-6 のトリガー「`currentOrgLives` の次の変更」も同様 | `internal/org/spawn.go` の `idempotentRespawn` の doc(第 1 段「is decided first against the same locked events」、第 2 段「So max_orgs is decided first」)と `idempotentRespawnDecision`。`docs/tech-debt/README.md` の上記 3 行(`grep -n 'idempotentRespawn\|scopeReservedEvent\|currentOrgLives' docs/tech-debt/README.md`)。差分に `docs/tech-debt/README.md` はない | 第 1 パスでは、書き換えた第 1 段の「decided first」を「decided before the seat is returned」に直す(行 C2-1 の提案どおり、2 語)。`Spawn` の doc の手順 1 も同じ。行の (b) の関数名を `idempotentRespawnDecision` に直す。F-2 と F-6 は、触れたのに直さなかった理由を行に 1 文足すか、片方を直す。台帳の更新は /sync-docs に渡してよい |
| L3 | LOW | readability | 雛形の文の食い違いと体裁。(a) 「機能ごとの org」の導入は「変更は `- 予約したパス:` の中に収めてください」と言うが、手順は予約の外に書かせる(手順 1 の `docs/plans/active/`、手順 4 の `docs/reports/`、手順 5 の `docs/plans/archive/`)。予約は分割計画の `Reserve:` の値(機能のコードと文書のパス)で、機能の計画と report は含まれないのが普通。読んだとおりに従う座席は、計画も report も書けなくなる。(b) 手順 5 の「後始末(8)」は、同じ雛形のミッションの 8(disband)とも読める。節内の手順 8 を指す意図なら「この節の 8」と書く。(c) 手順 5 の 2 行が他の行より長い(「ほかのファイルを含む)をコミットする。report をコミットせずに残すと、worktree に未追跡の」)。部分的に折り直した跡 | `internal/org/prompts/leader.md` の「機能ごとの org」の冒頭と手順 1・4・5・8。`TestRenderRolePrompt_Leader_FeatureOrgProcedure` は文の順序と主要語を固定するが、この食い違いは見ない | 冒頭を「機能のコードと文書の変更は予約したパスに収める。機能の計画・report・archive は予約の対象外」のように書く。(b)(c) は語句と折り返しの調整 |
| L4 | LOW | null-safety | 壊れた結びつきの記録(キーの欠け、重複)を `ActiveFeature` は「読めた分だけ」返し、`printStatusTable` と `printStatusJSON` はそのまま出す。空のフィールドは `feature: / branch  worktree ` のように見え、記録が壊れていることを示さない。`FeatureBinding.String()` は不完全な記録を注記するが、status は使っていない。org の動作は fail closed で正しく、台帳の行 F-8(壊れた予約を `.` として出す)と同じ種類の「壊れた記録に名前がない」問題 | `internal/cli/org.go` の `printStatusTable`(`fmt.Fprintf(out, "feature: %s/%s branch %s worktree %s\n", ...)`)。`internal/org/reserve.go` の `featureBindingFromTokens`、`FeatureBinding.complete`。`grep -n -i 'incomplete\|damaged' internal/cli/org_feature_test.go` は 0 件(org 層のテストは `TestReservationFromEvent_DamagedBindingMatchesNothing`) | 不完全なときは status の行に `(incomplete record)` のような注記を足す。そこまでしない場合も、F-8 の行に同じ種類として 1 文足す |
| L5 | LOW | maintainability | 小さな形の問題。(a) `StartFeatureResult.Split` は本番に読む側がなく、テストだけが読む(`readStartFeature` が「エラーのときも plan を返す」のもこのため)。(b) `ResolveSplitPlanPath` が返す `id` を本番の呼び出し(`readStartFeature`)は捨て、`LoadSplitPlan` が同じ id をパスからもう一度求める。(c) `splitFeatureDraft.dependsLine` は `fieldLine[splitFieldDependsOn]` と同じ値の複製。(d) `internal/org/spawn.go` の `HerdrClient` の comment の最後の行が 91 桁で、ほかの行(78 桁以下)の幅に揃っていない。(e) `EnsureWorktree` は要求の構造体だが動詞の名前で、`Ensure(repoRoot, EnsureWorktree{...})` と読むと動作に見える | `grep -n 'res\.Split\|result\.Split' internal -r`(`feature_test.go` のみ)。`internal/org/feature.go` の `readStartFeature`、`internal/org/split.go` の `ResolveSplitPlanPath`・`LoadSplitPlan`・`splitFeatureDraft` | どれも挙動を変えない。直すなら (a) は結果のフィールドを残す理由を doc に 1 文、(b) は `LoadSplitPlan` に id を渡すか戻り値を落とす、(d) は折り返し、(e) は `EnsureWorktreeRequest`。直さないなら無視してよい |

## Positive notes

- 先読みと本実行の同一性が、コードの構造(共有関数)とテスト(16 ケース、エラー文字列の一致まで)の両方で守られている。`Spawn` の doc に「順序か判定を変えたら先読みも直す」という保守の指示がある
- 結びつきの Details は、読む側の規則(`key=value` でない最初の語で止める、重複・欠けは不完全として fail closed)が、書く側の検査(`validateFeatureBinding` が空白・`=`・`,`・制御文字を拒否)と対になっている。古いバイナリが最初の空白でパスを切って読める互換も doc に書いてある
- 分割計画の承認の検査が、digest が読まない行(2 行目以降の `- Status:` / `- Approved:`、`- Branch:`、`## Progress checklist`、`- [x]`)を読み込みで拒否する。承認のあとに書き換えられる本文は digest に必ず入る。CRLF の行は digest 側では `\r` を含めて比べ、解析側では 1 つ外して(より厳しく)拒否するので、安全な側にずれる
- `ResolveSplitPlanPath` は symlink を解決してから `splits/` の直下かを比べる。ファイル自体の symlink も同じ検査を通るので、`--plan` で外のファイルを読ませられない
- `seat.go` のパッケージ doc と `spawn.go` の `HerdrClient` の doc にあった「`internal/org` は `exec.Command` を使わない」は、差分の前から `statedir.go`・`watch.go` で偽だった。この差分は、`feature.go` を足すのに合わせて実態(git、ralph-worktree.sh、osascript、`claude -p`)に直した
- `/org` skill の拒否の一覧、`start` の手順 1〜5、worktree の後始末(`cleanup` の `--force-branch`)は、`feature.go` と `ralph-worktree.sh` の挙動と合っている。実機の記録(`docs/evidence/`)は、確かめていないこと(push と `gh pr create` が通る場合、codex の leader)を結論に書いている
- 相対の `--cwd` の修正(af138af6)は、実機で見つけた穴を入口の 1 か所(`checkSpawnInput`)で塞ぎ、dry-run・台帳・herdr・agmsg のテストを 1 つにまとめている。計画からのずれとして計画の進捗に記録されている

## Coverage gaps

- テストコードは全文を読んでいない。読んだのは `TestSpawnPrecheckErr_MatchesSpawn`、`TestStartFeature_RealWorktreeScript`、`TestStartFeature_EnsureFailureRefused`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure`、`TestOrgSpawn_RelativeCwdResolvedAgainstCallerWorkingDir`。`split_test.go`(655 行)、`feature_test.go` の残り、`org_feature_test.go`、`spawn_feature_test.go`、`reserve_test.go` の追加分は、既存テストの削除行が機械的な署名変更だけなことの確認にとどめた
- M1 の数字(slug の上限と組み合わせ上限)は、コードを読んで求めた。再現の実行はしていない(テストと実行は /test の担当)
- 4 面の skill は `cmp` で同一なことを確かめ、1 面を全文読んだ。`scripts/check-skill-sync.sh` と `scripts/check-sync.sh` は実行していない
- 先読みと `ensure` の間、`ensure` と `Spawn` の間の競合(同じ機能の `start --plan` の同時実行など)は、計画の Risks と進捗メモ(4)に記録済みで、ここでは見直していない

## Tech debt identified

この回(cycle 1、上限 2)は直す回が 1 回残っているので、新しい行は足さない。M1、M2、L1〜L3 は cycle 2 で直せる大きさ。L4、L5 は挙動を変えない。cycle 2 でも直さずに終わる finding があれば、その回の報告で 1 行にまとめて台帳へ送る。

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (なし。上の注記のとおり) | | | | |

計画の進捗メモにある「実装中に見つけて送るもの」4 件((1) start を打った ralph と pane の中の ralph の版が違うと台帳が分かれる、(2) reviewer のレポートが Stop hook の `wip:` コミットで PR に入る、(3) 昇格した leader の `ralph org report` が未追跡のファイルを残す、(4) 同じ機能の `start --plan` の同時実行)は、まだ `docs/tech-debt/README.md` にない。/sync-docs で台帳に入れる予定と計画に書いてあるので、入れ忘れると計画の archive で失われる。

## /sync-docs に渡すもの

- L2 の台帳の行の更新(再実行後の状態: C2-1 は 8aae7ce2 で直ったので、行の (a) を直した旨にする。C2-3 と T2-1・T2-2 の `if !seat.Active` は `idempotentRespawnDecision` に移った。F-2・F-6 のトリガーは満たされたまま)
- 上の 4 件の新しい行
- 再実行で M2 は直った。README と `templates/base/` に director の言及がないことは確認済み
- N3: 計画の進捗に 8aae7ce2 の 1 行(20 文字の上限、ensure の失敗ごとの案内、status の `incomplete`)
- L5(`StartFeatureResult.Split`、`ResolveSplitPlanPath` の id、`dependsLine`、`HerdrClient` の comment の幅、`EnsureWorktree` の名前)のうち直すもの

## 再実行(8aae7ce2・2c1bb13b のあと)

対象は `git diff 2a544af9..HEAD`(2 コミット、15 ファイル、+452/-157)。`git log --format='%h parent=%p'` で 2a544af9 → 8aae7ce2 → 2c1bb13b の一直線を確かめた。`git diff --check` は空、追加行に U+FFFD、デバッグ出力、TODO はなく、コミットメッセージに帰属の行はない。skill の 4 面は `cmp` で同一。`grep -n director .claude/skills/org/SKILL.md` は 0 件。テストは実行していない。読んだ追加テストは `TestStartFeature_EnsureFailureRefused`、`TestEnsureFailureMessages_InWorktreeScript`、`TestStartFeature_OrgIDAtTheLengthLimit`、`TestOrgStatus_IncompleteFeatureBinding`、`TestRenderRolePrompt_Leader_FeatureOrgProcedure` の追加分、`split_test.go` の 2 ケース。

### 初回の指摘の現況

| ID | 現況 | 確かめたこと |
| --- | --- | --- |
| M1 | 直った | `maxFeatureOrgIDLen = maxHerdrAgentNameLen - 1 - len(implementerSeatID)`(20、定数式)を `newSplitFeatureDraft`(slug)と `readStartFeature`(`--org-id` を含む org_id)の両方で検査し、どちらも副作用の前に拒否する。エラー文は同じ `featureOrgIDLimit()` で上限の理由(32 文字の agent 名と `implementer`)を言う。leader の雛形の手順 2 は `--id` を `implementer` と `reviewer` に決め、理由を添える。skill は slug と `--org-id` の上限を 20 文字に直し、拒否の一覧にも足した。境界は 20 文字(通る、leader が `implementer` を spawn できるところまで)と 21 文字(拒否)をテストが固定する(`TestStartFeature_OrgIDAtTheLengthLimit`、`TestLoadSplitPlan_Fields`、`_Rejects`) |
| M2 | 直った | skill の 4 面から director の節と 3 か所の言及を外した。残した結びつきの事実(結びつきのない走っている org への `start --plan` は拒否)は、コードの挙動(`reservationDecision`)と合っている。承認の節は「分割計画を承認するのは人」で始まる。README、AGENTS.md、`templates/base/`、`internal/` に director の言及はない(`grep -rnw director` は 0 件) |
| L1 | 直った | `ensureFailureErr` が、`ensure` の失敗を文言で分ける。clean な default branch を求める失敗(`must start from clean default branch`、`has uncommitted changes`)だけに「main を clean にして打ち直す」を付け、状態の衝突は `cleanup` か別の `--org-id`、記録のないディレクトリは削除か別の `--org-id`、ブランチの衝突は名前の変更か削除(`--org-id` を変えても同じブランチと明記)を案内する。`.codex/config.toml` の書き換え、jq なし、default branch なしは、スクリプトの文のまま返す。照合する 6 つの文字列は定数で、`TestEnsureFailureMessages_InWorktreeScript` がスクリプトの本文にまだあることを見る。本物のスクリプトで、手で作ったブランチに当たる実行も足された。`.codex/config.toml` の文言(`ensureCodexRewriteMsg`)を `has uncommitted changes` より先に判定する順序も正しい |
| L2 | 一部直った | 書き換えた `idempotentRespawn` の doc と `Spawn` の手順 1 から、2 つの「decided first」が消え、「decided before the seat is returned」「max_orgs is decided before the reservation」になった(台帳の行 C2-1 の提案どおり)。`grep -n 'decided first' internal/org/*.go` は 0 件。台帳の行(C2-1 を直した旨、C2-3・T2-1・T2-2 の `if !seat.Active` が `idempotentRespawnDecision` に移った点、F-2・F-6 のトリガー)は未更新で、/sync-docs に回す |
| L3 | 直った(N2 が残る) | 冒頭が「機能のコードと文書の変更は予約したパスの中に収める。手順 1・4・5 の書き込みは予約の外でよい」になった。手順 5 は「この節の 8」と書き、折り返しも直った |
| L4 | 直った | `FeatureBinding.Complete()` を公開し、status の行の末尾に `(incomplete record)`、`--json` の `feature` に `"incomplete": true` を足した。完全な結びつきの JSON は従来の 5 キーのまま(`omitempty`)。`status` の help と skill の 2 か所(動詞の表、予約の節)に書いてあり、「そのような org では disband まで予約を渡す spawn と start がすべて拒否される」は `reservationDecision` の挙動と合う。3 通りの壊れ方のテスト(`TestOrgStatus_IncompleteFeatureBinding`)がある |
| L5 | 残る(依頼どおり /sync-docs へ) | 変更なし |

### 新しい指摘

| ID | Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- | --- |
| N1 | LOW | maintainability | 雛形のテストが、折り返しの位置を含む文字列を固定する。雛形の文を折り直すと、意味が変わらなくても落ちる。初回のテストは 1 行に収まる語句だけを見ていた | `internal/org/prompts_test.go` の `TestRenderRolePrompt_Leader_FeatureOrgProcedure`: `"手順 1 の\n機能の計画、4 の report、5 の計画の移動は予約の外に書きますが"`、`"`--id` は `implementer` と\n   `reviewer` にする"`。同じファイルの既存の固定(`markdownItem` が折り返しを 1 項目にまとめる)は改行を含まない | 改行をまたぐ語句は、`strings.Join(strings.Fields(...), " ")` で空白を 1 つにしてから比べるか、1 行に収まる語句(`implementer` と `reviewer` にする)だけを見る |
| N2 | LOW | readability | 冒頭の例外は leader 自身の手順 1・4・5 の書き込みだけを挙げる。実機の記録では reviewer が `docs/reports/` に自分の report を書いており、これも「機能のコードと文書」の外にある。座席の役割プロンプトは別にあるので害は小さい | `internal/org/prompts/leader.md` 「機能ごとの org」の冒頭。`docs/evidence/org-feature-worktree-live-2026-10-09.md` の Run 3(reviewer の self-review report) | 「機能の計画・report・計画の移動は予約の対象外」と手順の番号によらずに書く。直さなくても動きは変わらない |
| N3 | LOW | maintainability | 8aae7ce2 の計画からのずれが、計画の進捗に記録されていない(slug と `--org-id` の 20 文字の上限、ensure の失敗ごとの案内、status の `incomplete`)。計画は /pr で archive されるので、承認済みの Scope の文(「ensure の失敗は main を clean にして打ち直す案内を添える」)だけが残る。`ralph org start --help` と `--org-id` の usage は 20 文字の上限に触れていない(エラー文と skill には書いてある) | `docs/plans/active/2026-10-09-org-feature-worktree.md` の Progress checklist(S1〜S5 の記録のみ)。`internal/cli/org.go` の `newOrgStartCmd` の Long と `--org-id` の説明 | 進捗に 1 行足す(`## Progress checklist` は digest の外なので承認は崩れない)。help は、`--plan` の説明に「org_id は 20 文字まで」と 1 文足すのが最小 |

### 直しで新しく入った問題の確認

- ensure の失敗の分け方: 照合は部分文字列で、スクリプトの文言が変わったときは「何も足さない」側(`default` が err をそのまま返す)に倒れる。誤った案内を出す方向には倒れない。ensure 失敗のうち、`git worktree add` 自体の失敗(スクリプトが `set -e` で落とす)も `default` で、スクリプトの出力と終了コードだけが返る
- 20 文字の定数: `implementerSeatID` と leader の雛形の `--id implementer` は別々に書かれているが、雛形の文は `TestRenderRolePrompt_Leader_FeatureOrgProcedure` が固定し、テストのコメントが `maxFeatureOrgIDLen` との結びつきを言う。定数だけを変えるとテストが落ちる向きは守られている。雛形の `--id` を変えたときに定数が追いつく向きは、雛形の文の固定が間接的に守る
- `--org-id` の上限は `start --plan` だけで、`ralph org spawn`・`start <task>` の org_id は従来どおり 30 文字まで。skill は「機能ごとの org」の節に限った書き方になっており、食い違いはない
- skill から director を外したあとの文: 承認の節、2 経路の (A)(B)、「機能ごとの org」の導入のどれも、前後の文とつながっている。「分割計画の機能に結びつくのは `start --plan` で立てた org だけ」という追加文は、`reservationDecision` の挙動と合う
- `FeatureBinding.Complete` の公開: `String()`、`sameFeature`、CLI から使われ、doc は「ActiveFeature が読めた分だけ返し、status が使う」と理由を書いている。`complete` の旧名は残っていない(2c1bb13b の説明どおり、feature.go の中継も外れた)

## Recommendation

- Merge: 可。再実行後の finding は CRITICAL 0、HIGH 0、MEDIUM 0、LOW 4 件(L5、N1、N2、N3)。初回の MEDIUM 2 件(M1、M2)と LOW の L1・L3・L4 は直り、L2 は doc の部分が直った(台帳の行の更新が残る)。直しで入った問題は、LOW の N1〜N3 のほかに見つからなかった。先読みの同一性、結びつきの Details の読み書き、補償の写しには、初回から直すべき欠陥はない(初回の判定は「可(条件付き)」で、条件は M1・M2 の修正だった)
- Follow-ups: この先は /verify に進んでよい。L5・N1・N2・N3 は挙動を変えない。cross-review の fix の機会があれば N1(テストの折り返しへの依存)と N3 の進捗の 1 行を拾う。直さずに最後の回を終える場合は、その回の報告で 1 行にまとめて台帳へ送る。/sync-docs に渡すもの: 台帳の行の更新(L2 の残り: C2-1 は直った、C2-3・T2-1・T2-2 の関数名、F-2・F-6)、計画の進捗メモにある 4 件の新しい行、L5 のどれか、N3 の進捗の 1 行。`/test` には、M1 の境界(20・21 文字)と ensure の失敗ごとの案内が、実際のスクリプトの文で出ることを頼む(`TestEnsureFailureMessages_InWorktreeScript` はスクリプトの文字列の存在だけを見る)。計画の進捗の「Review artifact created」は未チェックのまま
