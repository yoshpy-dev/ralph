package org

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The words the tests below find the sync-docs paragraphs at the end of step 9
// by, and the name of the follow-up section that the sync-docs report, the PR
// body, and the RESULT's DOC_FOLLOWUP: lines share (plan
// 2026-10-30-leader-sync-docs).
const (
	syncDocsHandOff         = "rebase が済んだら(要らなかったときも)、sync-docs を doc-maintainer の subagent に任せる"
	syncDocsCheck           = "doc-maintainer が終わったら、`git status --porcelain` で変わったパスを読む"
	syncDocsFollowupHeading = "## 予約外の文書の follow-up"
	syncDocsSkipLine        = "`SYNC_DOCS: skipped (<理由>)`"
	docFollowupKey          = "DOC_FOLLOWUP"
	leaderStep14Marker      = "14. HEAD("
	leaderStep10FirstLine   = "10. `ralph org report --org-id org-a --state-dir <台帳>` を打ち(report"
)

// The names step 9 of the leader template and item 1 of the /org skill's
// ### org の終わり方 both use for the sync-docs commit and for what the
// leader does around it (plan 2026-10-30-leader-sync-docs, D2 and D2c): the
// commit's fixed message, the commands that list the branch's earlier
// sync-docs commits and the files each changed (the files a redo freezes),
// the start of the sync-docs reports those files leave out, the undo, and
// the report section that records what was undone.
const (
	syncDocsCommitMessage  = "`docs: sync docs for <機能の slug>`"
	syncDocsEarlierCommits = "`git log --format=%H --grep='^docs: sync docs for <機能の slug>$' origin/<既定のブランチ>..HEAD`"
	syncDocsCommitFiles    = "`git show --name-only --format= <コミット>`"
	syncDocsReportPrefix   = "docs/reports/sync-docs-"
	syncDocsRestore        = "`git restore --source=HEAD --staged --worktree -- <パス>`"
	syncDocsUndoneHeading  = "## 戻した変更"
)

// syncDocsNoReportSkip is the start of the sentence, the same in step 9 of
// the leader template and in item 1 of the /org skill's ### org の終わり方,
// that takes a failed call and a doc-maintainer that ended without the
// report as one branch: undo every change, report included, check that
// git status --porcelain prints nothing, then skip (plan
// 2026-10-30-leader-sync-docs, D5). Each goes on with its own step numbers.
const syncDocsNoReportSkip = "呼び出しが失敗したとき(Codex で custom agent を呼べない、途中で止まった、時間切れになった、など)と、" +
	"doc-maintainer が report を書かずに終わったときは、同じ 1 つの枝で扱う。" +
	"どちらも、次の段落の戻し方で sync-docs の変更を report も含めてすべて戻し、" +
	"`git status --porcelain` が何も出さないことを確かめてから sync-docs を飛ばし、" + syncDocsSkipLine + " の行を"

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsEndsStep9 pins AC1 of plan
// 2026-10-30-leader-sync-docs (D1): sync-docs is two paragraphs at the end of
// step 9, after the reviewer's report is committed and the branch fetched and
// rebased and after the sentence that leaves a gate TASK's GATE: pass to step
// 12, so it comes before step 10's report and archive, the secret scan, and
// step 12's gate TASK. The steps are not renumbered: step 10's first line is
// the one the existing tests pin, and the intro lets the sync-docs report be
// written outside the reservation like the steps' other own writes.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsEndsStep9(t *testing.T) {
	section := leaderFeatureOrgSection(t)
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		"まず reviewer のレポート(reviewer が RESULT や BLOCKED に書いた、`docs/reports/` の下のファイル)をコミットし",
		"`git fetch origin <既定のブランチ>` を打つ",
		"`git rebase origin/<既定のブランチ>` を打つ",
		"gate の TASK(手順 12)への `GATE: pass` の RESULT は、この手順ではなく手順 12 で扱う(ここに戻ってコミットを足すと、push する HEAD が gate の HEAD からずれる)",
		syncDocsHandOff,
		"doc-maintainer への指示には、次のことを書く",
		syncDocsCheck,
		"手順 14 が使うのは最後の回のもの(最後の回の report か、最後の回で控えた `SYNC_DOCS:` の行)で、前の回の控えは使わない",
	})
	assertPhrasesInOrder(t, section, "the 機能ごとの org section", []string{
		step9Marker,
		syncDocsHandOff,
		syncDocsCheck,
		leaderStep10FirstLine,
		"./scripts/archive-plan.sh <機能の計画のパス>",
		"./scripts/secret-scan-branch.sh --strict",
		step12Marker,
		"ralph org gate --org-id org-a --state-dir <台帳>",
	})
	if !hasLine(section, leaderStep10FirstLine) {
		t.Errorf("expected step 10 to keep its first line %q, got section:\n%s", leaderStep10FirstLine, section)
	}

	intro, _, ok := strings.Cut(section, "\n"+leaderStep1Marker)
	if !ok || !containsPhrase(intro, "手順 1・4・6 の機能の計画、9 の reviewer のレポート、10 の report と計画の移動、12 の計画の戻しは予約の外に書きますが") ||
		!containsPhrase(intro, "手順 9 の終わりに sync-docs が書く report(`docs/reports/` の下)も同じく手順のうちなので、予約の外に書いてかまいません") {
		t.Errorf("expected the 機能ごとの org intro to let step 9's sync-docs report be written outside the reservation, with the steps' other own writes, got section:\n%s", section)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsHandsTheDocMaintainerItsLimits
// pins AC2 and AC8 of plan 2026-10-30-leader-sync-docs (D2b, D3): the leader
// hands sync-docs to the doc-maintainer subagent (Task on Claude Code, the
// custom agent on Codex) instead of editing docs itself, and tells it the
// plan, the base, the reserved paths read the way ralph reads a reservation,
// to edit only inside the reservation, to leave the frozen files alone, to
// edit a set of copies only when every copy is inside it and otherwise write
// every copy of the set to the follow-ups one per line (so a later docs-only
// feature can reserve them all), to run the repo's copy checks and undo what
// fails them, where the report and its follow-up section go, and to add no
// insight event, stage nothing, commit nothing, leave the plan alone, and run
// no ralph org command. The task lines the hand-off names are the ones
// featureLeaderTask writes, and the reading is the one reservePathsOverlap
// applies.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsHandsTheDocMaintainerItsLimits(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		syncDocsHandOff,
		"あなた自身のコンテキストを使わないためで、文書は自分では直さない",
		"Claude Code では `Task(subagent_type=\"doc-maintainer\")` で、Codex では custom agent の `.codex/agents/doc-maintainer.toml` で呼ぶ",
		"doc-maintainer への指示には、次のことを書く",
		"機能の計画のパスと、比べる base の `origin/<既定のブランチ>`(既定のブランチは上の決め方で決める)",
		"予約したパス(タスクの `- 予約したパス:` の行)と、その読み方(ralph の予約と同じく、ファイルはそのパス、`/` で終わるディレクトリはその下、`.` は repo 全体)",
		"直すのは予約の中のファイルだけにすること",
		"凍らせたファイルがあれば、その一覧と、そのファイルは予約の中でも直さず、残るずれは follow-up に書くこと",
		"repo の検査が中身をそろえる写しの組(skill の写しなど)は、組のすべてが予約の中にあるときだけ直し、1 つでも予約の外にあれば、" +
			"組のどれも直さずに、組の写しを 1 つ 1 行ずつ(予約の中のものも外のものもすべて)follow-up に書くこと" +
			"(あとで文書だけの機能が組のすべてを予約できるように)",
		"直したあと、repo に写しの検査(`scripts/check-skill-sync.sh`・`scripts/check-sync.sh`)があれば打ち、落ちる自分の変更を戻して follow-up に回すこと",
		"report を `docs/reports/sync-docs-<YYYY-MM-DD>-<機能の slug>.md`(`<機能の slug>` はタスクの `- 機能:` の行の名前)に書き",
		"予約の外で見つけた文書のずれを、その `" + syncDocsFollowupHeading + "` の節に 1 件 1 行(パスと、何が古いか)で書くこと(なければ `なし` の 1 行)",
		"insight event を付けないこと、ステージしないこと、コミットしないこと、機能の計画を変えないこと、`ralph org` のコマンドを打たないこと",
		syncDocsCheck,
	})

	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/", "b.md"}}, "/state", "/wt", "feat/a", "per-plan")
	for _, line := range []string{"\n- 機能: a\n", "\n- 予約したパス: a/, b.md("} {
		if !strings.Contains(task, line) {
			t.Fatalf("featureLeaderTask no longer writes %q, the line step 9's sync-docs hand-off names; got:\n%s", line, task)
		}
	}
	for _, c := range []struct {
		reserve, path string
		want          bool
	}{
		{"b.md", "b.md", true},
		{"b.md", "b.md/c.md", false},
		{"a/", "a/x/y.md", true},
		{"a/", "ab/y.md", false},
		{".", "docs/x.md", true},
	} {
		if got := reservePathsOverlap(c.reserve, c.path); got != c.want {
			t.Errorf("reservePathsOverlap(%q, %q) = %t, want %t: the reading of a reservation step 9 hands the doc-maintainer", c.reserve, c.path, got, c.want)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsUndoesEveryChangeOrNone pins
// the leader's check of AC2 and AC8 of plan 2026-10-30-leader-sync-docs (D2):
// after the doc-maintainer, the leader reads git status --porcelain, and when a
// path outside the reservation or the feature plan changed, it undoes every
// sync-docs change but the report, never only the outside half (a set of
// copies half changed fails the gate's copy check, which the implementer
// cannot fix outside the reservation). The undo restores tracked files and
// staged new files from HEAD in the index and the work tree (a plain
// git restore leaves a staged change in place for the commit) and deletes
// untracked files. What is left goes in one commit with the report under the
// fixed message the redo's git log --grep looks for, not an example of one;
// the report alone when nothing else changed, and nothing when nothing
// changed. The plan is undone because step 6's approval check refuses a TASK
// after the plan changed, which step 6 says.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsUndoesEveryChangeOrNone(t *testing.T) {
	step9 := leaderFeatureOrgItem(t, step9Marker)
	assertPhrasesInOrder(t, step9, leaderItemWhere(step9Marker), []string{
		syncDocsCheck,
		"(ここまでで何も出なくなっているので、出たものはすべて sync-docs の変更)",
		"report のほかに、予約の外のパスか機能の計画が 1 つでも変わっていれば、report のほかの sync-docs の変更をすべて戻す",
		"追跡しているファイルとステージした新しいファイルは " + syncDocsRestore + " で戻し" +
			"(ステージした新しいファイルは index からも作業ツリーからも消える)、未追跡のファイルは消す",
		"予約の外の分だけを戻すと、写しの組の片方だけが変わったまま残って gate の写しの検査が落ち、予約の外を直せない implementer も直せないので、一部だけを戻すことはしない",
		"計画を戻すのは、per-plan では承認のあとに計画が変わると implementer への TASK が断られるため(手順 6)",
		"どちらもなければ変更はそのまま残す",
		"残った変更と report を、決まったメッセージ " + syncDocsCommitMessage + " の 1 つのコミットにし" +
			"(上の `git log --grep` はこのメッセージで探すので、言い方を変えない)、`git status --porcelain` が何も出さないことを確かめる",
		"report のほかに変更がなければ、report だけで同じメッセージのコミットにする(report が前の回のコミットと同じで何も出なければ、コミットせずに手順 10 に進む)",
	})
	for _, old := range []string{"たとえば `docs: sync docs for", "追跡しているファイルは `git restore <パス>`"} {
		if containsPhrase(step9, old) {
			t.Errorf("expected step 9 to no longer say %q (the message is fixed, and a plain git restore leaves a staged change in the commit), got item:\n%s", old, step9)
		}
	}
	if item := leaderFeatureOrgItem(t, "6. (per-plan)"); !containsPhrase(item, "承認のあとに計画が変わると、implementer の spawn と TASK が断られる") {
		t.Errorf("expected step 6, which step 9's plan undo points at, to say a plan changed after the approval gets the implementer's TASK refused, got item:\n%s", item)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsRecordsTheUndoneChangesInTheReport
// pins L3 of the self-review of plan 2026-10-30-leader-sync-docs (D2, D3):
// before the undo the leader reads what each undone path changed, and after it
// writes the paths and what they changed in a ## 戻した変更 section of the
// report and adds them to the report's follow-up section before the commit, so
// the committed report does not read as if they were fixed and the record is
// in a file, not in the leader's memory. Step 14 reads the follow-ups for the
// PR body and the RESULT from the last sync-docs report, found from the newest
// sync-docs commit, and no longer adds undone changes kept in memory.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsRecordsTheUndoneChangesInTheReport(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		syncDocsCheck,
		"戻す前に、戻すパスごとに `git diff HEAD -- <パス>`(未追跡のファイルは中身)を読み、何を変えていたかを控える",
		syncDocsRestore,
		"戻したら、report に `" + syncDocsUndoneHeading + "` の節を足して戻したパスと変えていた中身の要点を 1 件 1 行で書き、" +
			"同じ項を report の `" + syncDocsFollowupHeading + "` の節にも足す(`なし` の行は消す)",
		"コミットした report が戻した変更を直したままのように読めず、手順 14 が記憶でなく report から読めるようにするため",
		"残った変更と report を、決まったメッセージ " + syncDocsCommitMessage,
	})
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	assertPhrasesInOrder(t, step14, leaderItemWhere(leaderStep14Marker), []string{
		"PR の本文には `" + syncDocsFollowupHeading + "` の節を置き、手順 9 で最後に走らせた sync-docs の report" +
			"(手順 9 と同じ `git log --grep` の 1 行目の、最も新しい sync-docs のコミットが入れた `" + syncDocsReportPrefix + "` のファイル)" +
			"をファイルから読み直して、その同じ名前の節の項を 1 件 1 行(パスと、何が古いか)で書く",
		"(手順 9 で戻した変更もその節に足してあるので、記憶からは足さない。なければ `なし`。",
		"本文には、PR の本文と同じく、その report の follow-up の節の項を、1 件 1 行の `" + docFollowupKey + ": <パス>: <ずれ>` で書く",
	})
	if containsPhrase(step14, "戻して控えた変更") {
		t.Errorf("expected step 14 to read the undone changes from the report instead of adding the ones step 9 kept in memory, got item:\n%s", step14)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezesEarlierSyncDocsFiles
// pins AC6 and D2c of plan 2026-10-30-leader-sync-docs (self-review M1):
// before it calls the doc-maintainer, the leader lists the branch's earlier
// sync-docs commits from git by their fixed message and the union of the
// files they changed, the sync-docs reports left out, and hands that union to
// the doc-maintainer as frozen files, so a GATE: fail caused by a sync-docs
// change does not come back with the same change put back. The list comes
// from git, not from the gate's LOG or the leader's memory. The commit the
// list finds is the one the next paragraph makes with that message.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezesEarlierSyncDocsFiles(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		syncDocsHandOff,
		"呼ぶ前に、このブランチのこれまでの sync-docs のコミット(メッセージは次の段落のとおり決まっている)を " + syncDocsEarlierCommits +
			" で引き、それぞれの " + syncDocsCommitFiles + " が出すファイルの和集合を作る" +
			"(`" + syncDocsReportPrefix + "` で始まる sync-docs の report は除く)",
		"和集合が空でなければ、そのファイルを凍らせたファイルとして doc-maintainer に渡す",
		"sync-docs の変更で `GATE: fail` になったあと、sync-docs が同じ変更を入れ直して同じ理由でまた落ちないためで、" +
			"gate の `LOG:` を読んで原因を推し量ることはしない",
		"一覧は記憶からでなく git から引くので、compaction や rebase のあとも同じものが出る",
		"doc-maintainer への指示には、次のことを書く",
		"凍らせたファイルがあれば、その一覧と、そのファイルは予約の中でも直さず、残るずれは follow-up に書くこと",
		syncDocsCheck,
		"決まったメッセージ " + syncDocsCommitMessage,
	})
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezeFindsTheFixedMessage
// runs step 9's two commands for the files a redo freezes, as the template
// writes them, in a repository whose branch has two sync-docs commits made
// with the fixed message, besides one on origin's default branch, one of a
// feature whose slug starts with this one's, and an implementer's commit:
// git log lists the two, newest first, and the union of what git show
// --name-only lists for them, less the sync-docs reports, is the files they
// changed. The slug a leader puts in the --grep pattern is one
// ValidateIdentifier accepts, which has no regular expression metacharacter.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsFreezeFindsTheFixedMessage(t *testing.T) {
	step9 := leaderFeatureOrgItem(t, step9Marker)
	for _, want := range []string{syncDocsEarlierCommits, syncDocsCommitFiles, syncDocsCommitMessage, "`" + syncDocsReportPrefix + "`"} {
		if !containsPhrase(step9, want) {
			t.Fatalf("expected step 9 to say %s, which this test runs, got item:\n%s", want, step9)
		}
	}

	dir := t.TempDir()
	runGit(t, "init", "--quiet", "--template=", dir)
	message := strings.ReplaceAll(strings.Trim(syncDocsCommitMessage, "`"), "<機能の slug>", "a")
	commit := func(msg string, files ...string) {
		t.Helper()
		for _, f := range files {
			writeGateFile(t, dir, f, msg+"\n", 0o644)
			runGit(t, "-C", dir, "add", f)
		}
		gateCommit(t, dir, msg)
	}
	commit(message, "old.md")
	runGit(t, "-C", dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	commit(message, "README.md", syncDocsReportPrefix+"2026-10-30-a.md")
	commit("feat: the feature", "a.go")
	commit("docs: sync docs for a-b", "b.md")
	commit(message+"\n\nThe redo's sync-docs.", "docs/x.md", syncDocsReportPrefix+"2026-10-31-a.md")
	newest := gitOut(t, dir, "rev-parse", "HEAD")

	shell := func(cmdline string) []string {
		t.Helper()
		cmd := exec.Command("sh", "-c", cmdline)
		cmd.Dir = dir
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("sh -c %q: %v", cmdline, err)
		}
		return strings.Fields(string(out))
	}
	logCmd := strings.NewReplacer("<機能の slug>", "a", "<既定のブランチ>", "main").Replace(strings.Trim(syncDocsEarlierCommits, "`"))
	commits := shell(logCmd)
	if len(commits) != 2 || commits[0] != newest {
		t.Fatalf("expected %q to list the branch's two sync-docs commits of a, the newest (%s) first, got %q", logCmd, newest, commits)
	}
	var frozen []string
	for _, c := range commits {
		for _, f := range shell(strings.ReplaceAll(strings.Trim(syncDocsCommitFiles, "`"), "<コミット>", c)) {
			if !strings.HasPrefix(f, syncDocsReportPrefix) && !slices.Contains(frozen, f) {
				frozen = append(frozen, f)
			}
		}
	}
	slices.Sort(frozen)
	if want := []string{"README.md", "docs/x.md"}; !slices.Equal(frozen, want) {
		t.Errorf("expected the frozen files of the two sync-docs commits, less the reports, to be %q, got %q", want, frozen)
	}

	for _, slug := range []string{"a.b", "a*", "a+", "a?", "a[b]", "a$", "a^b", `a\b`, "a|b", "a(b)"} {
		if ValidateIdentifier("feature slug", slug) == nil {
			t.Errorf("ValidateIdentifier accepted the feature slug %q, which would change what step 9's git log --grep pattern matches", slug)
		}
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsSkipsWithoutTheDocMaintainer
// pins D5 of plan 2026-10-30-leader-sync-docs: with no doc-maintainer
// definition in the worktree, the leader skips sync-docs without a call
// (nothing to undo) instead of editing docs itself and keeps
// SYNC_DOCS: skipped for step 14, which writes it in the RESULT instead of the
// DOC_FOLLOWUP: lines and puts its reason in the PR body. The definitions it
// looks for are the ones the scaffold ships, under the name the Task call and
// the custom agent use.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsSkipsWithoutTheDocMaintainer(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		"文書は自分では直さない",
		"その定義が worktree にない(Claude Code では `.claude/agents/doc-maintainer.md`、Codex では `.codex/agents/doc-maintainer.toml`)ときは、" +
			"呼ばずに sync-docs を飛ばし(戻す変更はない)、" + syncDocsSkipLine + " の行を手順 14 のために控えて手順 10 に進む",
	})
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	for _, want := range []string{
		"sync-docs を飛ばしたときは、控えた " + syncDocsSkipLine + " の理由",
		"sync-docs を飛ばしたときは、`" + docFollowupKey + ":` の行の代わりに、控えた " + syncDocsSkipLine + " の行を書く",
	} {
		if !containsPhrase(step14, want) {
			t.Errorf("expected step 14 to say %q, got item:\n%s", want, step14)
		}
	}

	root := reportNamesRepoRoot(t)
	for _, c := range []struct{ rel, nameLine string }{
		{"templates/base/.claude/agents/doc-maintainer.md", "name: doc-maintainer"},
		{"templates/base/.codex/agents/doc-maintainer.toml", `name = "doc-maintainer"`},
		{".claude/agents/doc-maintainer.md", "name: doc-maintainer"},
		{".codex/agents/doc-maintainer.toml", `name = "doc-maintainer"`},
	} {
		t.Run(c.rel, func(t *testing.T) {
			body, ok, err := readRepoFile(root, c.rel)
			if err != nil {
				t.Fatalf("read %s: %v", c.rel, err)
			}
			if !ok {
				t.Skipf("%s not found -- skipping the doc-maintainer definition check for it", c.rel)
			}
			if !hasLine(body, c.nameLine) {
				t.Errorf("expected %s to define the agent step 9 of the leader template calls, with the line %q, got:\n%s", c.rel, c.nameLine, body)
			}
		})
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsCountsANoReportEndAsASkip
// pins D5 of plan 2026-10-30-leader-sync-docs (self-review L2): a failed call
// (Codex cannot call the custom agent, the doc-maintainer stopped part way or
// timed out) and a doc-maintainer that ended without writing the report are
// one branch, so a call that fails after it wrote files cannot be read as a
// skip that undoes nothing: every change, the report included, is undone the
// way the next paragraph undoes them, git status --porcelain is checked to
// print nothing, and only then is sync-docs skipped with SYNC_DOCS: skipped
// kept for step 14. The sentence comes after the skip for a missing
// definition and before the hand-off's list, no other sentence of step 9
// speaks of a failed call, and the paragraph after it is the one that starts
// with the leader's check and holds the undo.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsCountsANoReportEndAsASkip(t *testing.T) {
	step9 := leaderFeatureOrgItem(t, step9Marker)
	assertPhrasesInOrder(t, step9, leaderItemWhere(step9Marker), []string{
		"ときは、呼ばずに sync-docs を飛ばし(戻す変更はない)",
		syncDocsNoReportSkip + "手順 14 のために控えて手順 10 に進む",
		"(ファイルを書いたあとで失敗したときも、汚れた作業ツリーのまま手順 10 に進まないため)",
		"doc-maintainer への指示には、次のことを書く",
		syncDocsCheck,
		syncDocsRestore,
	})
	if n := strings.Count(squashSpace(step9), "呼び出しが失敗"); n != 1 {
		t.Errorf("expected step 9 to speak of a failed call once, in the one branch it shares with an end without the report, got %d; item:\n%s", n, step9)
	}
	next, ok := skillParagraphAfter(step9, syncDocsNoReportSkip)
	if !ok || !strings.HasPrefix(squashSpace(next), squashSpace(syncDocsCheck)) || !containsPhrase(next, syncDocsRestore) {
		t.Errorf("expected the paragraph after the one that says %q to start with %q and hold the undo it points at, got item:\n%s", syncDocsNoReportSkip, syncDocsCheck, step9)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgSyncDocsRunsAgainOnARedo pins AC6 of
// plan 2026-10-30-leader-sync-docs: a GATE: fail, or a gate refusal whose fix
// needs the implementer, goes back through step 8 to step 9, so sync-docs runs
// again on the new head, and step 14 uses the last run's report or
// SYNC_DOCS: line. A GATE: fail in files the sync-docs commit changed (a copy
// check, say) takes the same path: the implementer fixes it inside the
// reservation, and the redo's sync-docs gets the earlier sync-docs commits'
// files frozen (step 9), so it does not put the same change back.
func TestRenderRolePrompt_Leader_FeatureOrgSyncDocsRunsAgainOnARedo(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		syncDocsCheck,
		"gate をやり直すとき(手順 12 の `GATE: fail` や、implementer の手が要る gate の断りのあと)は、手順 8 の往復を経てこの手順を通り直すので、sync-docs も新しい HEAD でもう一度走る",
		"手順 14 が使うのは最後の回のもの(最後の回の report か、最後の回で控えた `SYNC_DOCS:` の行)で、前の回の控えは使わない",
	})
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, gateFailMarker), leaderItemWhere(gateFailMarker), []string{
		"implementer への TASK に書いて直しを頼み、手順 8 の往復に戻る",
		"reviewer が通したら、手順 9 からやり直す",
		"手順 9 の sync-docs のコミットが変えたファイルで落ちたとき(写しの検査など)も同じで、implementer に予約の中で直しを頼む",
		"手順 9 からやり直すと、その終わりで sync-docs がもう一度走る",
		"そのとき、これまでの sync-docs のコミットが変えたファイルは凍らせて渡す(手順 9)ので、doc-maintainer に同じ変更を入れ直させない",
	})
	// The refusal that needs the implementer is the GATE: unrunnable item's,
	// which also comes back to step 9.
	if item := leaderFeatureOrgItem(t, gateUnrunnableMarker); !containsPhrase(item, "手順 8 の往復に戻る。reviewer が通したら、手順 9 からやり直す") {
		t.Errorf("expected the GATE: unrunnable item's refusal that needs the implementer to come back to step 9, where sync-docs runs again, got item:\n%s", item)
	}
}

// TestRenderRolePrompt_Leader_FeatureOrgResultAndPRBodyCarryTheDocFollowups
// pins AC4 of plan 2026-10-30-leader-sync-docs (D3): the PR body has the
// follow-up section the sync-docs report has, with the items of the last
// report's section, which holds the changes step 9 undid too (なし when
// none, the skip reason when skipped), and the RESULT carries the same items
// as one DOC_FOLLOWUP: line each in its body, DOC_FOLLOWUP: なし when none,
// or one line with the count pointing at the PR body when they do not fit
// the body cap. The example RESULT is one escalate accepts, with
// DOC_FOLLOWUP: lines of the <path>: <stale> shape in its body, not its header
// block.
func TestRenderRolePrompt_Leader_FeatureOrgResultAndPRBodyCarryTheDocFollowups(t *testing.T) {
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	assertPhrasesInOrder(t, step14, leaderItemWhere(leaderStep14Marker), []string{
		"`/pr` skill そのものは実行しない",
		"PR の本文には `" + syncDocsFollowupHeading + "` の節を置き、手順 9 で最後に走らせた sync-docs の report",
		"その同じ名前の節の項を 1 件 1 行(パスと、何が古いか)で書く",
		"PR ができたら、PR の URL を EVIDENCE に書いて `ralph org escalate` で RESULT を上げる",
		"RESULT のヘッダには、手順 12 で gate の出力のファイルから控えた `HEAD:` と `BASE:` の 2 行をそのまま写す",
		"本文には、PR の本文と同じく、その report の follow-up の節の項を、1 件 1 行の `" + docFollowupKey + ": <パス>: <ずれ>` で書く" +
			"(なければ `" + docFollowupKey + ": なし` の 1 行)",
		"本文が 2,000 字に収まらなければ、その行の代わりに、件数と PR の本文の節を指す 1 行(`" + docFollowupKey + ": 14 件(PR の本文の「予約外の文書の follow-up」の節)` など)を書く",
		"たとえば次のメッセージにする",
	})
	if protocol.DefaultMaxBodyChars != 2000 {
		t.Errorf("protocol.DefaultMaxBodyChars = %d, but step 14 of the leader template says the RESULT's body is capped at 2,000 characters", protocol.DefaultMaxBodyChars)
	}
	if step9 := leaderFeatureOrgItem(t, step9Marker); !containsPhrase(step9, "その `"+syncDocsFollowupHeading+"` の節に") {
		t.Errorf("expected step 9 to name the report's follow-up section %q, the one the PR body copies, got item:\n%s", syncDocsFollowupHeading, step9)
	}

	msg := fencedBlockAfter(t, step14, leaderStep14Marker, leaderItemWhere(leaderStep14Marker))
	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeResult || m.TaskID != "org-a" {
		t.Fatalf("expected escalate to accept the example as a RESULT with TASK_ID org-a, got TYPE %q TASK_ID %q, err %v; message:\n%s", m.Type, m.TaskID, err, msg)
	}
	if got := protocol.HeaderValues(msg, docFollowupKey); len(got) != 0 {
		t.Errorf("expected the example RESULT's %s: lines in its body, not its header block, got header values %q; message:\n%s", docFollowupKey, got, msg)
	}
	n := 0
	for _, line := range strings.Split(m.Body, "\n") {
		rest, ok := strings.CutPrefix(line, docFollowupKey+": ")
		if !ok {
			continue
		}
		n++
		if path, stale, ok := strings.Cut(rest, ": "); !ok || path == "" || strings.ContainsAny(path, " \t") || strings.TrimSpace(stale) == "" {
			t.Errorf("expected the example RESULT's %s: line %q to read <パス>: <ずれ>", docFollowupKey, line)
		}
	}
	if n < 2 {
		t.Errorf("expected the example RESULT's body to show one %s: line per follow-up (two or more), got %d; body:\n%s", docFollowupKey, n, m.Body)
	}
}
