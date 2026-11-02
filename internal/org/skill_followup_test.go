package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file pin what the /org and the /director skills say
// about the follow-ups of issue #270 (plan 2026-10-30-org-followups, AC6 and
// AC7, D2 and D5): the verbs `ralph org followup add|list|done`, the
// follow-up sections of `ralph status` and `ralph org merge-order`, the two
// ways a follow-up is ready, and the director's steps (record it, read it
// when a merge is reported, mark it done). The verbs and the sections are the
// binary's (internal/cli/org_followup.go, status_followup.go,
// org_merge_order.go); these tests keep the four copies of each skill saying
// the same, and check the record's names and states against this package's
// constants.
//
// If one of these tests fails after a deliberate change to a rule, put the
// rule's new words, short and verbatim, in its fragment in the skill's order;
// otherwise restore the text in .claude/skills/<org or director>/SKILL.md and
// templates/base/.claude/skills/<org or director>/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the first test in
// org_skill_send_sync_test.go.

// followupSkillHeading is the subsection both skills give the follow-ups.
const followupSkillHeading = "### merge を待つ issue"

// checkSkillHeadingOrder fails t for each copy of a skill (copies, relative
// to the repository root) that does not have every heading of headings, each
// on a line of its own, after the one before it. It skips t when a copy is
// missing, as checkSkillPlaces does.
func checkSkillHeadingOrder(t *testing.T, copies []string, headings []string) {
	t.Helper()
	root := directorSkillRepoRoot(t)
	for _, rel := range copies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's heading order check (vendored repo?)", rel, err)
		}
		text, at := string(data), -1
		for _, h := range headings {
			i := strings.Index(text, "\n"+h+"\n")
			if i < 0 || i < at {
				t.Errorf("%s: expected the heading %q after the ones before it in %q -- see the comment at the top of this file", rel, h, headings)
				break
			}
			at = i
		}
	}
}

// TestOrgSkill_FollowupVerbRowsSayWhatEachRecordsOrPrints pins the /org
// skill's verb-table rows of followup add, list, and done (AC6, D3): add's
// required flags and their checks, the refusal that writes nothing, the
// record under followups.lock and the ID it prints, and a branch already
// merged needing no follow-up (D5); list's default, --all, --json and
// columns (with --all, DONE_AT and DONE_NOTE too: self-review L7), its
// damage counts on stderr, and that it shows the record only;
// done's note, its second run that changes nothing, and its exits; and the
// --org-id the three refuse. The note limit, the lock and the file are this
// package's.
func TestOrgSkill_FollowupVerbRowsSayWhatEachRecordsOrPrints(t *testing.T) {
	row := func(verb string) func(string) string {
		return func(text string) string { return orgSkillVerbRow(text, verb) }
	}
	checkOrgSkillPlaces(t, "followup verb rows", []skillPlace{
		{"the verb row of followup add", row("followup add"), []string{
			"follow-up として台帳に記録する(「機能ごとの org」節の「merge を待つ issue」)",
			"`--issue <番号>`(正の整数)と `--after-branch <ブランチ>`(空でなく、空白と制御文字を含まず、`-` で始まらない)が必須",
			"`--note <文>`(1 行、500 文字まで、制御文字なし)は任意",
			"検査に通らなければ何も書かずに終了コード 1",
			"`followups.lock` の下で台帳の `followups.jsonl` に記録し、stdout に `added <id>: issue #<番号> after <ブランチ>` を出す(ID は `f1`、`f2`、…)",
			"PR がもう merge されたブランチは follow-up にせず、issue をすぐ人に聞く(記録より前の merge は数えないので、着手できるにならない)",
			"`followup` の動詞は `--org-id` を拒否する",
		}},
		{"the verb row of followup list", row("followup list"), []string{
			"既定は open だけ、`--all` で done も、`--json` で機械向けの形(`items`・`corrupt_lines`・`ignored_events`)",
			"列は `ID` / `STATE` / `ISSUE` / `AFTER_BRANCH` / `ADDED_AT` / `NOTE`",
			"`--all` のときはその後に `DONE_AT` / `DONE_NOTE`(done の時刻と note。open の件は `-`)が続く",
			"読めない行と無視したイベントは数を stderr に出す",
			"出すのは記録だけで、着手できるかは出さない",
			"`ralph status` と `ralph org merge-order` の `follow-ups:` の節が出す",
			"記録が読めなければ終了コード 1",
		}},
		{"the verb row of followup done", row("followup done"), []string{
			"open の follow-up を done にする(人に issue を聞いた、または要らなくなった)",
			"`--note <文>` は任意で、決めたことや PR の URL を 1 行、500 文字まで、制御文字なしで書く",
			"stdout に `done <id>` を出す",
			"done の件にもう一度打つと何も変えずに `<id> is already done; nothing changed` を出して終了コード 0 で、最初の done の note が残る",
			"知らない ID と検査に通らない note は終了コード 1",
		}},
	})

	if InboxNoteMaxRunes != 500 {
		t.Errorf("InboxNoteMaxRunes = %d, but the /org skill's followup add and done rows say a note is at most 500 characters", InboxNoteMaxRunes)
	}
	if followupLockFile != "followups.lock" {
		t.Errorf("followupLockFile = %q, but the /org skill's followup add row names followups.lock", followupLockFile)
	}
	if got := filepath.Base(FollowupsPathIn(t.TempDir())); got != "followups.jsonl" {
		t.Errorf("FollowupsPathIn names %q, but the /org skill's followup add row names followups.jsonl", got)
	}
}

// TestOrgSkill_MergeOrderAndStatusRowsNameTheFollowupsAndTheReservedLine pins
// what the /org skill says the other verbs show of this change (AC6, AC7,
// D2, D4): the merge-order row ends its text with the follow-up section,
// puts the follow-ups in --json in an object with the keys of ralph status's
// (items with source gh, the damage counts, error: self-review L3), keeps
// the exit code, and runs
// gh for the follow-ups' after-branches even with no branch in the ledger;
// the status row says that ralph status shows each running org's reserved:
// line (an org with only a reservation too) and the ledger's follow-ups,
// which ralph org status does not; the reservation subsection says the same
// of ralph status; and the legacy-ledger lists of ## 前提 put followup add
// and done with the verbs that change the ledger and followup list with the
// ones that only read it.
func TestOrgSkill_MergeOrderAndStatusRowsNameTheFollowupsAndTheReservedLine(t *testing.T) {
	checkOrgSkillPlaces(t, "follow-up and reserved lines", []skillPlace{
		{"the verb row of merge-order", func(text string) string { return orgSkillVerbRow(text, "merge-order") }, []string{
			"続けて `note:` の行で注記を、最後に open の follow-up の `follow-ups:` の節" +
				"(`ID`・`ISSUE`・`AFTER_BRANCH`・`STATE`・`SOURCE`・`PR`・`DETAIL`。件がなければ出さない。「merge を待つ issue」)を出す",
			"follow-up は `followups` に入れる。`ralph status --json` の `followups` と同じキーの object で、`items`(各件の判定の道の " +
				"`source` は `gh`)、記録の傷の数の `corrupt_lines` と `ignored_events`、記録が読めなかった理由の `error`(読めたか記録がなければ空)を持つ",
			"分類と follow-up の状態によらず終了コード 0",
			"続けて注記と follow-up の節を出して終了コード 0",
			"open の follow-up があればその after-branch の `gh pr list` と origin の照合だけ",
		}},
		{"the verb row of status", func(text string) string { return orgSkillVerbRow(text, "status") }, []string{
			"`ralph status` も、走っている org ごとに同じ `reserved:` の行を出し(座席がなく予約だけがある org も。`--json` は `reservation`)",
			"org のあとに台帳全体の open の follow-up を `follow-ups:` の節に出す(「機能ごとの org」節の「merge を待つ issue」)",
			"follow-up は `ralph org status` には出ない",
		}},
		{"the subsection ### 担当範囲の予約(`--reserve`)", func(text string) string {
			return directorSkillSubsection(text, "### 担当範囲の予約(`--reserve`)")
		}, []string{
			"`ralph status` も、走っている org ごとに同じ `reserved:` の行を出す(`--json` は `reservation`)",
			"座席がなく予約だけがある org も、org_id の行とこの行を出す",
		}},
		{"the legacy-ledger lists of ## 前提", func(text string) string { return directorSkillSubsection(text, "## 前提") }, []string{
			"`director launched`(launch_cmd の起動スクリプトだけが打つ隠れた動詞)/ `followup add` / `followup done`)は終了コード 1 で止まり",
			"`merge-order` / `director status` / `followup list`、`ralph insights`)は",
		}},
	})

	if FollowupMergeSourceGH != "gh" {
		t.Errorf("FollowupMergeSourceGH = %q, but the /org skill's merge-order row says the follow-ups' source is gh", FollowupMergeSourceGH)
	}
}

// TestOrgSkill_FollowupSubsectionSaysWhatIsRecordedAndHowReadyIsDecided pins
// the /org skill's subsection ### merge を待つ issue, placed after ### merge
// してよい順 and before ### 座席内 fan-out (AC6, AC7, D1, D2, D5): what the
// record holds (followups.jsonl, appended under followups.lock, added and
// done, open and done) and that it outlives a director and a summary; the
// two ways ready is decided, both counting only a merge after the follow-up
// was recorded: ralph status from the ledger's pr_merged or worktree_cleaned
// with no gh (source recorded, so a PR merged on GitHub and not cleaned up
// stays waiting, and one merged on GitHub before the follow-up and cleaned up
// after it is ready while merge-order says waiting, merge-order's answer
// being the one to use when one says ready and the other waiting, and
// status's, with merge-order's reason, when merge-order says unknown:
// self-review M1 and R2-1), and ralph org merge-order from
// GitHub (ready, closed, waiting, unknown with every reason the CLI can
// meet: L6, source gh); a branch already merged needing no follow-up; what
// the three outputs tell without anything else; and the done, with the done
// ones in list --all with their time and note (L7). The event and state
// names are this package's.
func TestOrgSkill_FollowupSubsectionSaysWhatIsRecordedAndHowReadyIsDecided(t *testing.T) {
	checkOrgSkillPlaces(t, "follow-up subsection", []skillPlace{
		{"the subsection " + followupSkillHeading, func(text string) string { return directorSkillSubsection(text, followupSkillHeading) }, []string{
			"その約束を follow-up として台帳に記録する(`/director` skill の「merge を待つ issue」)",
			"「org の終わり方」の予約外の文書の follow-up とは別のもの",
			"ralph org followup add --issue 270 --after-branch feat/auth-token",
			"ralph org followup done f1",
			"記録は台帳の下の `followups.jsonl` に追記だけで書き(`added` と `done`)、書き込みは `followups.lock` のロックの下で行う",
			"記録するのは issue の番号、待つブランチ(after-branch)、note、記録した時刻で、状態は open と done の 2 つ",
			"台帳全体のもので、どの org にも属さない",
			"プロセスや director が替わっても、会話を要約したあとも残る",
			"着手できる(`ready`)かを決める道は 2 つある",
			"どちらも、follow-up を記録した時刻より後の merge だけを数える",
			"前に merge された同じ名前のブランチは数えない",
			"`ralph status` は台帳の記録で決め、`gh` を打たない",
			"after-branch の `pr_merged`(`ralph org merge` が書く)か `worktree_cleaned`(merge された PR の `ralph org cleanup` が書く)があれば `ready`、なければ `waiting`",
			"SOURCE の列(`--json` は `source`)は `recorded`",
			"GitHub で merge して、まだ cleanup していない PR は `waiting` のままになる",
			"ただし、GitHub で記録より前に merge し、記録のあとに `ralph org cleanup` で片付けた PR は、`worktree_cleaned` が記録より後なので " +
				"`ready` になる(merge-order は `waiting`。`ready` と `waiting` で割れたら merge-order の答えを使い、" +
				"merge-order が `unknown` なら、その理由を添えて status の答えを使う)",
			"`ralph org merge-order` は GitHub で決める",
			"記録より後に merge された PR があれば `ready`、番号が最大の PR が merge されずに閉じていれば `closed`",
			"だけ merge されていれば `waiting`",
			"`gh` がない、`gh pr list` が失敗したか時間切れになった、PR が origin の repository のものでないか照らせない、" +
				"merge された PR の merge の時刻が読めない、`gh` が上限の 1000 本を返してどれも記録より後に merge されていない、" +
				"PR の状態が OPEN・MERGED・CLOSED のどれでもない、follow-up の記録の時刻が読めない、のどれかなら `unknown` と理由(DETAIL)",
			"SOURCE の列(`--json` は `source`)は `gh`",
			"PR がもう merge されたブランチは follow-up にしない",
			"`followup list` はどの issue がどのブランチを待つか(記録だけで、着手できるかは出さない)",
			"`ralph status` と `ralph org merge-order` の `follow-ups:` の節は open の件ごとの `STATE` と `DETAIL`(理由)を出す",
			"人に聞いたら(要らなくなったときも)`ralph org followup done <id>` で done にする",
			"done の件は `followup list --all` にだけ、done の時刻と note(`DONE_AT`・`DONE_NOTE`)とともに出る",
		}},
	})
	checkSkillHeadingOrder(t, orgSkillCopies, []string{"### merge してよい順", followupSkillHeading, "### 座席内 fan-out"})

	for _, c := range []struct{ name, got, want string }{
		{"FollowupEventAdded", FollowupEventAdded, "added"},
		{"FollowupEventDone", FollowupEventDone, "done"},
		{"FollowupStateOpen", FollowupStateOpen, "open"},
		{"FollowupStateDone", FollowupStateDone, "done"},
		{"EventPRMerged", EventPRMerged, "pr_merged"},
		{"EventWorktreeCleaned", EventWorktreeCleaned, "worktree_cleaned"},
		{"FollowupMergeReady", FollowupMergeReady, "ready"},
		{"FollowupMergeWaiting", FollowupMergeWaiting, "waiting"},
		{"FollowupMergeClosed", FollowupMergeClosed, "closed"},
		{"FollowupMergeUnknown", FollowupMergeUnknown, "unknown"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, but the /org skill's %s says %q", c.name, c.got, followupSkillHeading, c.want)
		}
	}
}

// TestDirectorSkill_RecordsAFollowupReadsItAtMergeReportsAndMarksItDone pins
// the /director skill's steps for a follow-up (AC6, AC7, D5): its subsection
// ### merge を待つ issue, placed after ### GitHub で merge した PR の後始末
// and before ### 予約外の文書の follow-up, records an issue put off until
// another feature's PR is merged with followup add (and none for a branch
// already merged), says the record outlives a director, reads the
// follow-up sections of ralph status and ralph org merge-order whenever a
// merge is reported and tells the human which issues are ready (and the
// reasons of closed and unknown), takes merge-order's answer when one says
// ready and the other waiting, either way, and status's, with merge-order's
// reason, when merge-order says unknown (self-review M1 and R2-1), and marks
// every open one of the issue done once the human decided, never starting
// the issue unasked (the question itself and the start are pinned in
// skill_split_questions_test.go);
// ## 進捗の報告 lists the follow-up sections, and ### merge と auto_merge
// reports the ready issues with a merge.
func TestDirectorSkill_RecordsAFollowupReadsItAtMergeReportsAndMarksItDone(t *testing.T) {
	step := func(n int) func(string) string {
		return func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, followupSkillHeading), n)
		}
	}
	checkDirectorSkillPlaces(t, "follow-up steps", []skillPlace{
		{"the subsection " + followupSkillHeading, func(text string) string { return directorSkillSubsection(text, followupSkillHeading) }, []string{
			"ほかの機能の PR の merge のあとに回すと決めることがある(先に始めると、その PR と衝突するときなど)",
			"その約束は会話に残すだけでなく、台帳に follow-up として記録する(`/org` skill の「merge を待つ issue」)",
			"記録は台帳にあるので、director が入れ替わっても、会話を要約したあとも残る",
			"新しい director は `ralph org followup list` で引き継ぐ",
		}},
		{"step 1 of " + followupSkillHeading, step(1), []string{
			"決めたときに `ralph org followup add --issue <番号> --after-branch <ブランチ>` を打つ",
			"`<ブランチ>` は待つ機能のブランチで、merge-order が出す名前を使う",
			"出た ID(`f1` など)を人に伝える",
			"そのブランチの PR がもう merge されていれば記録せず、issue をすぐ人に聞く(記録より前の merge は数えないので、着手できるにならない)",
		}},
		{"step 2 of " + followupSkillHeading, step(2), []string{
			"merge を人に報告するとき(「進捗の報告」と「merge と auto_merge」)は、`ralph status` と `ralph org merge-order` の `follow-ups:` の節も読み",
			"STATE が `ready` の件の issue を、着手できるものとして人に伝える",
			"`ralph status` の `ready` は台帳の merge の記録から(SOURCE が `recorded`)、merge-order の `ready` は GitHub の PR から決めたもの",
			"まだ片付けていない PR の follow-up は、status では `waiting` のままなので merge-order で確かめる",
			"逆に、GitHub で記録より前に merge し、記録のあとに片付けた PR の follow-up は、status では `ready`、merge-order では " +
				"`waiting` になる",
			"`ready` と `waiting` で割れたら merge-order の答えを使い、merge-order が `unknown` なら、その理由を添えて status の答えを使う",
			"`closed`(PR が merge されずに閉じた)と `unknown` の件は、DETAIL の理由をそのまま人に伝える",
		}},
		{"step 3 of " + followupSkillHeading, step(3), []string{
			"人がその issue をどうするか決めたら(着手する、後に回す、要らない)、その issue の open の follow-up ごとに " +
				"`ralph org followup done <id> --note <決めたこと>` を打ち、すべて done にする",
			"人に聞かないまま issue に着手しない",
		}},
		{"the follow-up bullet of ## 進捗の報告", func(text string) string {
			return directorSkillBullet(directorSkillSubsection(text, "## 進捗の報告"), "- `ralph status` と `ralph org merge-order` の最後の `follow-ups:` の節:")
		}, []string{
			"merge を待つ issue と、それぞれに着手できるか(下の「merge を待つ issue」)",
		}},
		{"the report bullet of ### merge と auto_merge", func(text string) string {
			return directorSkillBullet(directorSkillSubsection(text, "### merge と auto_merge"), "- merge を人に報告するときは")
		}, []string{
			"「merge を待つ issue」の 2 のとおり、着手できるようになった issue も伝える",
		}},
	})
	checkSkillHeadingOrder(t, directorSkillCopies,
		[]string{"### GitHub で merge した PR の後始末", followupSkillHeading, directorDocFollowupHeading})
}
