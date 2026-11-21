# Walkthrough: org-human-decision

- Date: 2026-11-22
- Plan: docs/plans/active/2026-11-21-org-human-decision.md
- Issue: #366
- Base: main 10e9322d

差分は 41 ファイル、3,678 行。そのうち約 1,100 行は報告、計画、insight で、Go は 16 ファイル(約 2,100 行、うちテストが約 1,700 行)。残りは雛形、skill の 4 写し、rule の 2 写し、仕様、tech-debt。下の順に読むと早い。

## 1. 何が足りなかったか

受信箱の件に答えるのでない人の決定(走っている org の方針を変える、など)を、走っている機能の org の leader に送る手順がなかった。director が DECISION を送っても、台帳には `sent` の要約しか残らず、leader を立て直す(`ralph org reseat`、`start --plan` のやり直し)と、新しい leader はその決定を知らない。

## 2. 送る形と記録

- `ORIGIN: human` のヘッダを 1 行だけ、leader 宛ての `TYPE: DECISION` に付けてよい(`checkSendOrigin`、`internal/org/send_origin.go:62`)。ほかの TYPE、ほかの値、ほかの座席、2 行、16 KiB を超える文は、何も打たず何も記録せずに断る(`--raw` と `--dry-run` も)
- 人の決定の `sent` は、Details が `origin=human ` で始まり(`raw=true` のあと)、本物の send なら全文を `ManifestEvent.Text`(`internal/org/manifest.go:70`)に持つ(`sentHumanText`、`send_origin.go:91`、`verbs.go:725`)
- ralph は誰が書いたかを見分けない。記録は「人の決定として送った」ことを言い、人が書いたことは言わない

## 3. 立て直した leader への引き継ぎ

- 立て直した leader のタスクの進捗の節に「人の決定」の項を足した(`reseatProgressHumanDecisionsLabel`、`internal/org/reseat_progress.go:51`)。org の今の命(最後の disband のあと)の、leader 宛ての人の決定の `sent` を、台帳の順に全文で並べる(`reseat_progress.go:228`)
- 進捗の節の本文はデータで指示ではない、という前置きのうち、この項だけは例外で指示として読む(`reseat_progress.go:40`)。leader の雛形も、人の決定に従い続けると言う

## 4. 送るのに失敗したとき

- 記録は、send が終了コード 0 で終わったときだけ残る(Enter のあとに `sent` を書く)。1 周目の cross-review の指摘で、人の決定に限って手の Enter で回復しないことにした。director の skill の「人の決定を送る」の 2 は、pane を読み、ほかの画面が出ていれば何も送らない、入力欄に文が残っていれば人に空けてもらってから同じファイルを送り直す、などの項で扱う
- CLI は人の決定かどうかを `SendResult.HumanOrigin`(`verbs.go:268`、`defer` で `verbs.go:513`)で知り、失敗したときは専用の note を出す(`orgSendHumanDecisionFailureNote`、`internal/cli/org.go:1591`)。note は手で Enter を押さないことと、rule の節と skill の手順を指すことだけを言う。人の決定でない send の note は変わらない

## 5. 先に読むテスト

- `TestCheckSendOrigin_Table`(`internal/org/send_origin_test.go:43`): 許す形と断る形
- `TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText`(`send_origin_test.go:156`)と `TestOrgSendOrigin_Refused_NothingTypedNothingRecorded`(`:244`)
- `TestReseatProgress_HumanDecisionsKeepTheLedgerOrder`(`internal/org/reseat_progress_decision_test.go:90`)、`…_AHumanDecisionBeforeTheLastDisbandIsNotListed`(`:117`)、`…_AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole`(`:318`)
- `TestOrgSend_HumanDecisionFails_PrintsItsOwnNote` と `TestOrgSendHumanDecisionFailureNote_Text`(`internal/cli/org_test.go`)
- `internal/org/prompts_leader_human_decision_test.go`: leader と director の雛形、director の skill の 4 写し、rule の 2 写しの文

## 6. 確かめたこと

- `./scripts/run-test.sh` rc 0(2 周目、HEAD b337156e、1,835 s)。変異 12 個がすべて落ちた(test report の Run 2)
- `./scripts/run-static-verify.sh` pass(verify report の Run 2)。main(10e9322d)を取り込んだあと、org、cli、config の絞ったテストが ok
- cross-review は 2 周。1 周目の ACTION_REQUIRED 1 件は直した。2 周目は WORTH_CONSIDERING 2 件で、上限の回なので直さず tech-debt の行にした

## 7. 残したもの

- tech-debt の新しい行(人の決定の記録と引き継ぎの 2 つの競合): 同じ leader への人の決定の send が 2 本重なると台帳の順が逆になりうる、send の Enter と記録の間に disband が重なると前の命の決定が次の命に渡りうる。どちらも数ミリ秒の窓で 2 つの操作が重なるときだけ起きる
- tech-debt の「Known gap of feat/org-human-decision (#366), L3」(compaction のあと leader が人の決定を読み直す手段がない)と、test report の穴の行
- 実機の director と leader の pane で、失敗した send の 4 つの項を通すことは確かめていない
