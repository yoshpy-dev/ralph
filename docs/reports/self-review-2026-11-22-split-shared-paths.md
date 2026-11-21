# Self-review report: split-shared-paths

- Date: 2026-11-22
- Plan: `docs/plans/active/2026-11-22-split-shared-paths.md`(issue #363)
- Reviewer: reviewer サブエージェント(`/self-review`)。パイプライン 1 回目(`cycle-count.json` は cycle 1)
- Scope: `git diff origin/main...HEAD`(HEAD `477594d1`、base `origin/main` 9cd63d60)。コードの変更は S1 `35458c51`(`split.go`)、S2 `2bb50cd4`(`reserve.go`、`feature.go`)、S3 `89429da9`(`watch.go`、`watch_scope.go`、`feature.go`、`prompts/leader.md`)、S3b `c9e99c63`(`reseat.go`)、文書は S4 `eb43b1b7`(skill の 4 写し、README、AGENTS.md、2 つの仕様、skill のテスト)。見たのは diff の質(名前、読みやすさ、null 安全、エラーの扱い、安全性、保守性)だけ。仕様への適合、テストの網羅、文書の食い違いの網羅は見ていない(`/verify`、`/test`、`/sync-docs` の担当)。テストは 1 本も走らせていない

## Evidence reviewed

- `git diff origin/main...HEAD` のコード部分の全部(`split.go`、`reserve.go`、`feature.go`、`watch.go`、`watch_scope.go`、`reseat.go`、`prompts/leader.md`)と、skill の director 側の diff、org 側の変更行(1 行が長いので `-U0` と切り詰めで読んだ)
- 周辺の全文読み: `parseSplitPlan` の読みの本体(`split.go:343-415`)、`splitFeatureDraft.read`(`split.go:599-625`)、`NormalizeReservePaths` 一式(`reserve.go:274-360`)、`featureBindingFromTokens`(`reserve.go:459-492`)、`reservationDecision` の全体(`reserve.go:686-742`)、`PlanReseat`(`reseat.go:440-560`)、`idempotentRespawnDecision` と `spawnCapacityErr`(`spawn.go:1599-1708`)
- 洗い出し: `FeatureBinding{` の組み立て(非テストは `feature.go:513` の 1 か所と `featureBindingFromTokens` だけ)、`.Shared`・`SharedPaths()`・`featureDigest(` の非テストの参照、`ActiveReservation(` と `activeReservation(` の読み手(`verbs.go:1961` と `:1987`、`status_followup.go:84`、`cli/org.go:2100` は表示か復元で、共有のパスは要らない)、`FeatureBinding` を値で比べる箇所(無い。`sameFeature` と個々の field の比較だけ)
- 新しいテストの読み(質の目視): `feature_shared_task_test.go` 全体、`reserve_shared_test.go` の `TestReservationDecision_Shared`、`prompts_leader_shared_test.go`、`skill_shared_paths_test.go` の前半、`watch_scope_shared_test.go` の fixture、`split_test.go` と `reserve_test.go` と `watch_scope_test.go` と `reseat_test.go` の既存テストの変更行

### 頼まれた観点ごとの確認

| 観点 | 確認したこと | 結果 |
| --- | --- | --- |
| digest と Details の後方互換 | `featureDigest`(`split.go:727-745`)は `shared` が空のとき field を足さないので、`- Shared:` のない計画のバイト列は変わらない。`TestFeatureDigest_Encoding` が空の `[]string{}` と nil を含めて固定している。Details は `shared=` を最後のトークンに置き(`reserve.go:378`)、古い binary は最初の未知の鍵から先を note として読む。`featureBindingFromTokens` の鍵の表に足しただけなので、`shared=` のない記録は `Shared` が `""` で、`Complete()` にも `RequiresApproval()` にも関わらない | 問題なし。ただし `- Shared:` で始まる自由文をすでにヘッダに書いた計画は、新しい binary で読み込みを拒否される(空白を含む文は「空白を含むパス」で断られる)。鍵が #363 で初めて決まったものなので現実の影響は小さい |
| 損なわれた `shared=` の読み | 下の「損なわれた `shared=` の評価」 | 向きは妥当。細かい点は LOW 6 |
| 受け付けと並行 | 共有のパスの 2 方向の判定は `reservationDecision` の中で、呼び出しは `checkCapacityAndStart` の manifest lock の下(`spawn.go:1599-1618`、idempotent 側は `:1661-1708`)。dry-run と `spawnPrecheckErr` は lock なしの読みで、予測だけ。reseat の `reservationDecision(sim, ...)`(`reseat.go:539`)も計画の段階の予測で、本物の spawn が lock の下で決める | 問題なし。`want.Feature` と `held.Feature` が nil でも `SharedPaths()` が nil を返すので、nil 参照にならない |
| パスの正規化の一貫 | split(`parseSplitShared`)、`sharedFromToken`(記録の読みと `validateFeatureBinding`)、見張り、reseat のどれも `NormalizeReservePaths`(並べ替えと重複除去を含む)を通り、`strings.Join(",")` と `Split(",")` で往復する。`.` は split で断り、読みでは `""` に落とす。重なりの判定は split の `finishSplitPlan` と `reservationDecision` が同じ `reserveOverlaps` を使う。leader のタスクの `--scope`(`feature.go:588`、予約、共有の順)と reseat の `--scope`(`reseat.go:476`、同じ順)は並びも同じ | 問題なし |
| leader のタスクの shell の引用 | `shellQuote(scope)` は予約と共有を合わせた 1 つの語を作る。`$`、バッククォート、引用符、バックスラッシュ、glob を含む 4 つの組を `/bin/sh` で 1 語に読むテスト(`feature_shared_task_test.go:73-117`)がある。`- 共有のパス:` の行の生のパスは、予約の行と同じく表示用で、leader が写すのは `--scope` の引数のほう | 問題なし。文言は LOW 2 |
| エラー文 | 3 つの断り(計画の読み込み、共有と予約、予約と共有)は、どちらの側が持つパスか、重なりの組、直し方か待つ先を言う。`featureDigestRefusal` に共有のパスの前後の値を出す枝が足された | 言い回しの点は LOW 5 |
| skill が言うこととコード | director の「共有のパス」の各文(置き場所、`.` と重なりの拒否、digest が全部変わる、立て直しと reseat の拒否、`--scope`、受け付けの両順)と、org の「共有のパスの直し方」の手順を、コードと突き合わせた。ヘッダ以外の `- Shared:` が本文になること、共有どうしが重なってよいこと、reseat の `--scope`、「分割計画の承認」の 4 と「機能ごとの出力先」の 1・3 の番号の指し先は合っている | 偽の主張は見つからなかった。director の手順 1 が計画のファイルを読ませる点だけ LOW 7 |

### 損なわれた `shared=` の評価

`shared=` が読めない記録(値が空、鍵が 2 回、規則に反するパス、`.` を含む)を「共有のパスなし」と読む選択を、`.`(repo 全体)と読む案と比べた。

- 見張り: 「なし」と読むと、共有のはずのパスの変更に scope_change が立つ。うるさい側に倒れる。「repo 全体」と読むと、その org の変更を何も知らせなくなり、静かに緩む側に倒れる。予約のパスが読めないときに repo 全体と読むのは、受け付けで重なりを塞ぐ向きが目的で、見張りは同じ読みの副作用で黙る(既存)。共有のパスにまで同じ副作用を持ち込まなかった判断は妥当
- 受け付け: 「なし」と読むと、その org の共有のパスが別の org の予約から守られなくなる(`reserve.go:452-458` の doc が代償として書いている)。この側は静かだが、守られないパスを変えれば見張りが知らせるので、無音にはならない
- 損なわれた記録を誰が作れるか: manifest は ralph だけが書く。座席が台帳に書けることは台帳全体の前提で、この PR が広げるものではない
- 一部のパスだけ壊れた値は、全部を捨てて「なし」にする(`sharedFromToken` は 1 つでも断られたら `""`)。うるさい側に倒れるだけで、足りないものは出ない

評価は、向きは妥当で変更不要。残る点は LOW 6(痕跡が残らないことと、reseat の `--scope` が計画でなく記録から作られること)。

## Findings

<!-- Area recommended values: naming, readability, null-safety,
     exception-handling, security, maintainability -->

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | maintainability | `sameFeature` の doc が「ヘッダの行は feature digest に入らない」を前提にしているが、S2 の変更で `- Shared:` の値が入るようになり、前提が偽になった。結論(全体の digest が同じなら feature digest も同じ)は今も正しいが、同じ段落の次の文(S2 が足した「共有のパスはどちらの digest にも入る」)と矛盾して読める | `internal/org/reserve.go:206-207`(`and no header line goes into a feature digest`)。直後の `reserve.go:209-213` が S2 の追加。`split.go:57-58` の「ヘッダの自由な文は行き先がない」は `- Shared:` を自由な文に数えないので成り立つ | 「`- Shared:` の行の値のほかに、ヘッダの行は feature digest に入らず、その行は全体の digest にも入るので」の形にする |
| LOW | readability | leader のタスクの共有のパスの行の括弧「座席の spawn の --scope には予約したパスと合わせて渡す」は、leader に自分で合わせさせる指示に読める。直前の `- 予約したパス:` の行の `--scope` はすでに予約と共有の合算で、`leader.md:132-133` は「両方が入っています」と言い、手順 3 と 5 は引数を書き換えずに付けると言う。leader が `--scope '<予約>, <共有>'` を組み立て直しても動くが、RC2 の直しが避けた「leader が引数を作る」形に戻る | `internal/org/feature.go:598-599`。同じ文が `feature_shared_task_test.go:22` と `:125`、`prompts_leader_shared_test.go` の期待に固定されている | 「座席の spawn の --scope は、上の行の括弧のとおり予約したパスと合わせてある」のように、できている事実を言う文にする。固定している 3 か所を合わせて直す |
| LOW | maintainability | ヘッダの外に書いた `- Shared:` は黙って読まれず、前文なら本文として leader に渡る。機能の節に書くのは `- Reserve:` と並べやすい間違いで、機能ごとに共有を変えるという誤解(計画の Non-goals が機能ごとを外している)をそのまま通す。同じ種類の「digest が見ない行」は `- Status:` と `- Approved:` がヘッダの外に出ると断られる(`isStatusOrApprovedLine`)ので、扱いがそろっていない。結果は scope_change の ALERT が出る(静かに緩まない)ので重くないが、書いた人は何も共有していないと気づく手がかりが ALERT だけ | `internal/org/split.go:599-607`(`splitFeatureDraft.read` は未知の行を本文にする)、`split.go:378-382`(ヘッダの外の `isStatusOrApprovedLine` の断り)。決定は `split_shared_test.go:88-95` が固定、skill の director 側が注意を書く | 決定済みなので変更必須ではない。直すなら、フェンスの外の `- Shared:` 行を機能の節(と前文)で断る。引用したい本文はフェンスに入れる、と断りの文に書く。直さないなら、director skill の「前文の節か機能の節に書いた…」の文を残す現状でよい |
| LOW | maintainability | `ralph org start --help` の立て直しの条件が、共有のパスの変化を数えていない。今は「計画の前文と機能の節が変わっていなければ」と言うが、`- Shared:` の行が変わっても断られる(`featureDigestRefusal` の新しい枝)。この PR が作った条件なので、この PR の中で 1 句足すのが筋で、後回しにすると `/verify` の文書の食い違いの検査と `/sync-docs` が同じ指摘を出す。断りの文は共有のパスを名指すので、影響は小さい | `internal/cli/org.go:1094-1100`(`as long as the plan's preamble and the feature's own section did not change`)。条件は `reserve.go:761-765` | 「前文、機能の節、ヘッダの `- Shared:` の行が変わっていなければ」と 1 句足し、feature digest の説明(`org.go:1097-1100`)にも「と共有のパス」を足す。`internal/cli` に help のテストがあれば合わせる |
| LOW | readability | `reservationDecision` が 43 行から 57 行(`reserve.go:686-742`)になり、50 行の目安を超えた。ループの中の 3 つの断りが長い書式文字列を並べ、最初の 1 つ(予約どうし)と後の 2 つ(共有)で書き方も違う。3 つ目の文の「then the one split plan %q shares」は「その分割計画が共有するパス」と読めるまで引っかかる | `internal/org/reserve.go:722-741`、とくに `:736` | 3 つの断りを `reservationOverlapErr(other, want, held)` のような 1 つの関数に出し、ループは「どの組が重なるか」だけにする。3 つ目の文は「then the path split plan %q shares」にする(テストが文を固定しているので、`reserve_shared_test.go` の `sharedThere` も合わせる) |
| LOW | null-safety | 読めない `shared=` は「なし」と読まれ、「記録に無い」と区別がつかない。`feature_digest=` は `String()` が `no feature digest` と言い、`approval=` は `approvalModeUnreadable` を持つが、共有のパスにはどちらも無い。また reseat の非 leader の座席の `--scope` は記録の `binding.SharedPaths()` から作られ(`reseat.go:476`)、leader のタスクは計画の `plan.Shared` から作られる(`feature.go:588`)。通常は feature digest が同じなので一致するが、記録が損なわれたときは、feature digest が同じで再起動が通り、leader のタスクは共有のパスを言い、座席の `--scope` はそれを持たない。損なわれた台帳の中だけの話 | `internal/org/reserve.go:491`、`:452-458`(doc の「代償」)、`reseat.go:476`、`feature.go:588` | 変更なしでよい。`featureBindingFromTokens` の doc の「代償」の段落に「reseat の `--scope` も共有のパスを持たなくなる」と 1 文足せば、読み手が気づける |
| LOW | maintainability | director skill の手順 1 は、走っている機能の org の共有のパスを `ralph org status` が出さないので、`feature:` の行が名指す分割計画の台帳の `splits/<split>.md` の `- Shared:` の行を読ませる。org が持つ値は start のときの記録で、計画のファイルを承認し直して変えたあと(止めていない org は立て直しの検査を受けない)は食い違うことがある。start の断りが決め手なので、食い違いの影響は director の数え直しと説明の誤りまで。計画が言う未対応の「status は共有のパスを出さない」の裏返し | `.claude/skills/director/SKILL.md` の「機能ごとの出力先」の 1(手順 1 の追加部分)。状態の表示は `internal/cli/org.go:2095-2145`(`orgFeatureJSON` と `printStatusTable`)。`orgFeatureJSON` は `Digest` と `FeatureDigest` を出す | status の表示は、この PR に入れない(下の「範囲外の 2 件の扱い」)。skill の文には「ファイルの値は今のもので、org が start のときに記録した値と違うことがある。start の断りが決め手」と 1 句足す |
| LOW | readability | (a) `featureLeaderTask` が 1 つのスライス・リテラルを `lines` と末尾の 9 個の要素に分け、共有の行を条件で足したあと `strings.Join(append(lines, ...9 個...), "\n")` で戻す。行の順を読むのに 2 か所を行き来する。(b) `parseSplitShared` は `NormalizeReservePaths` の断りの文から `"org: reserve "` を前置きごと切る。兄弟の `parseSplitReserve` は `"org: "` を切るので、2 つで切り方が違い、`normalizeReservePath` の文面が変わると共有のほうだけ `org: reserve` が残る。「空白とカンマを含む」の文には共有の文脈で `a reservation cannot hold` が残る | (a) `internal/org/feature.go:590-612`、(b) `split.go:539` と `:641`、`reserve.go:306-312` | (a) 共有の行を入れる位置に `slices.Insert` を使うか、末尾を `tail` に名前を付けて `slices.Concat(lines, tail)` にする。(b) 切り方をそろえる(どちらも `"org: "` を切る)か、文面の変更でテストが落ちることを `TestLoadSplitPlan_SharedRejects` が固定していることを doc に書く |

## Positive notes

- `featureDigest` は「共有のパスがあるときだけ field を足す」1 点に絞った。`digestField` を使った書き直しで、既存の field の順とバイト列は変わらず、`TestFeatureDigest_Encoding` が文字列のレベルで固定している。AC2(既存計画の digest が同じ)の根拠が計算の形で見える
- 共有のパスを `FeatureBinding.Shared`(カンマでつないだ文字列)にして、`FeatureBinding` が `==` で比べられる性質を保った。`Reservation.Shared []string` にしなかった理由(`SpawnParams.reservation()` が範囲外で組み立てる)と、記録の read/write の対称性が `reserve.go` の doc に書いてある
- 受け付けの 2 方向の判定を manifest lock の中の 1 関数に置き、共有どうしだけを通す規則をテストの表(ファイルとディレクトリ、同じ prefix の別ディレクトリ、全ペアの名指し)で固定した。重なる計画の側が `finishSplitPlan` と受け付けで同じ `reserveOverlaps` を使うので、判定の規則が 1 つ
- 見張りは「外」を決める入力に共有のパスを足しただけで、`scopePathInside` の規則と ALERT の文を変えていない。`ActiveReservation` を `activeReservation` に替えた 1 行で、予約の長さの判定(`len(r.Paths) > 0`)は同じ
- 共有のパスのない計画の leader のタスクが 1 バイトも変わらないことを、共有なしの表(nil と空)でテストが固定している。leader の prompt のテストは、タスクの行の語(`featureSharedLead`)を prompt の文と突き合わせる
- 範囲外の発見(reseat の `--scope` が予約だけだった)を S3 で見つけて S3b に分け、同じ PR の中で閉じた。reseat の 3 か所(spawn、dry run、`reseatCustomRoleErr` のコマンド)が同じ `scope` 変数を共有している

### 範囲外の 2 件の扱い(plan の Progress の S4 の記録)

- `ralph org start --help` の立て直しの条件(LOW 4): **この PR に入れるべき**。この PR が条件を足したので、1 句で足りる
- `ralph status` と `ralph org status` が共有のパスを出さない: **この PR に入れなくてよい**。`orgFeatureJSON` の field と table の行という出力の形が増え、テストの組も増える。director は計画のファイルから読めて、start の断りが決め手になる。ただし skill の手順 1 の文に「ファイルの値と記録の値が違うことがある」の 1 句は足したい(LOW 7)。この PR で入れないなら、台帳に行を足す

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `ralph status` と `ralph org status`(table と `--json` の `feature` の object)が、機能の org の共有のパス(`FeatureBinding.Shared`)を出さない。director は台帳の `splits/<split>.md` を読んで代わりにするが、その値は start のときの記録と違うことがある | 走っている org の共有のパスを知る手段が計画のファイルだけで、承認し直したあとは台帳の値と食い違う。`reserved:` の行に共有が出ないので、予約の重なりを数える人が見落としうる | 出力の形(`--json` の field)が増え、この PR の範囲(計画の Non-goals と Progress S4 の記録)の外 | `ralph status` の出力を次に触る変更、または共有のパスの食い違いで director が重なる org を数え間違えた報告 | `docs/plans/active/2026-11-22-split-shared-paths.md` の Progress S4、この report の LOW 7 |

_`docs/tech-debt/README.md` には足していない。ほかのレーンの PR も同じ表の末尾に足すので、並行の merge で衝突しやすい。`/pr` の時点で、この PR が status の表示を入れないと決まっていれば、上の行を足す。_

## Recommendation

- Merge: yes(CRITICAL 0、HIGH 0、MEDIUM 0、LOW 8)
- Follow-ups:
  - 直すなら安い順に LOW 4(`--help` の 1 句、この PR に入れるべきもの)、LOW 1(`sameFeature` の doc の前提)、LOW 2(タスクの行の文言と固定している 3 か所)、LOW 7(skill の 1 句)
  - LOW 3 は決定済みの動きで、直すかどうかは人の判断。LOW 5・LOW 8 は読みやすさだけで、直さなくても動きは変わらない。LOW 6 は doc の 1 文で足りる
  - どれを直しても、コードか skill に触れるので記録だけの修正ではなく、`/verify` 以降の全体の再実行が要る
  - 計画の Verify plan にある文書の食い違い(AGENTS.md の `split.go` と `watch_scope.go` の説明、分割計画の形を書く仕様)は `/verify` と `/sync-docs` の担当で、この review では網羅していない
