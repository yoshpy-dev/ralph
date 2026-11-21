package cli

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org start --plan --issue` and the issue of a feature org
// in `ralph org status` and `ralph org merge-order --json` (plan
// docs/plans/active/2026-11-22-org-start-issue.md, the CLI side of AC1, AC3,
// AC5, AC7 and AC10). The org layer's checks of the number, the record, the
// leader's task and the merge-order binding are covered in internal/org.
// The tests that make a worktree run the repo's real scripts/ralph-worktree.sh
// in a temp git repo and chdir into it, so none of them runs in parallel.

// TestOrgStartIssue_RefusedWithoutPlanOrBlank is AC3's CLI half: --issue
// without --plan is refused, with or without a task argument, and so is a
// blank --issue with --plan, which the org layer would read as no issue.
// Each refusal comes before the state dir is resolved, so nothing is made
// and herdr is not called.
func TestOrgStartIssue_RefusedWithoutPlanOrBlank(t *testing.T) {
	const needsPlan = "org: start: --issue needs --plan and --feature: it names the issue a split plan feature was filed as, " +
		"and ralph org start <task> starts no feature"
	const blank = "org: start: --issue must not be blank: give the number of the issue the feature was filed as " +
		"(--issue 12 for issue #12), or leave --issue out for a feature that was not filed as an issue"
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"a task with --issue", []string{"--org-id", "x", "--cwd", ".", "--issue", "12", "task text"}, needsPlan},
		{"--issue and no task", []string{"--org-id", "x", "--cwd", ".", "--issue", "12"}, needsPlan},
		{"--issue alone", []string{"--issue", "12"}, needsPlan},
		{"two tasks with --issue", []string{"--org-id", "x", "--cwd", ".", "--issue", "12", "one", "two"}, needsPlan},
		{"a blank --issue with no --plan", []string{"--org-id", "x", "--cwd", ".", "--issue", "", "task text"}, needsPlan},
		{"--feature and --issue with no --plan", []string{"--feature", "a", "--issue", "12"},
			"org: start: --plan and --feature go together"},
		{"an empty --issue", []string{"--plan", "demo.md", "--feature", "a", "--issue", ""}, blank},
		{"an empty --issue=", []string{"--plan", "demo.md", "--feature", "a", "--issue="}, blank},
		{"a --issue of spaces", []string{"--plan", "demo.md", "--feature", "a", "--issue", "   "}, blank},
		{"a --issue of a tab", []string{"--plan", "demo.md", "--feature", "a", "--issue", "\t"}, blank},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")

			stdout, _, err := runOrgCmdStreams(t, append([]string{"start", "--state-dir", stateDir}, tc.args...)...)
			if err == nil {
				t.Fatalf("expected start refused, stdout: %s", stdout)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("expected the error to contain %q, got: %v", tc.want, err)
			}
			if _, statErr := os.Stat(stateDir); !os.IsNotExist(statErr) {
				t.Errorf("expected no state dir to be made, stat: %v", statErr)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
		})
	}
}

// TestOrgStartIssue_MalformedNumberRefusedWithNothingMade is the rest of
// AC3 through the CLI: a --issue that is not blank but not an issue number
// (0, a leading 0, a #, not decimal digits, a space around it, more than 10
// digits) reaches StartFeature as it was given, which refuses it before the
// watch gate, the herdr server gate and the worktree, so nothing is recorded,
// herdr is not called and no worktree is made.
func TestOrgStartIssue_MalformedNumberRefusedWithNothingMade(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	stateDir := filepath.Join(t.TempDir(), "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	wt := filepath.Join(root, ".claude", "worktrees", "org-b")

	for _, value := range []string{"0", "012", "#12", "twelve", "12a", " 12", "12 ", "１２", "12345678901", "+12"} {
		t.Run(value, func(t *testing.T) {
			stdout, _, err := startOrgFeature(t, plan, "b", "--state-dir", stateDir, "--issue", value)
			if err == nil {
				t.Fatalf("expected --issue %q refused, stdout: %s", value, stdout)
			}
			for _, want := range []string{
				`org: --issue "` + value + `" is not an issue number: give the number of the issue alone, `,
				"1 to 10 decimal digits with no leading zero and no # (--issue 12 for issue #12)",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected the error to contain %q, got: %v", want, err)
				}
			}
			if !strings.HasPrefix(stdout, "rejected: ") {
				t.Errorf("expected the rejected line on stdout, got: %s", stdout)
			}
			if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
				t.Errorf("expected nothing recorded, got %v", eventTypes(events))
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
			if _, statErr := os.Stat(wt); !os.IsNotExist(statErr) {
				t.Errorf("expected no worktree %s, stat: %v", wt, statErr)
			}
		})
	}
}

// issueOrgReservation returns orgID's latest scope_reserved under stateDir,
// nil when it has none.
func issueOrgReservation(t *testing.T, stateDir, orgID string) *org.ManifestEvent {
	t.Helper()
	var reserved *org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.OrgID == orgID && ev.SeatID == "" && ev.Event == org.EventScopeReserved && !ev.DryRun {
			reserved = &ev
		}
	}
	return reserved
}

// issueLeaderPrompt returns the leader prompt of orgID under stateDir.
func issueLeaderPrompt(t *testing.T, stateDir, orgID string) string {
	t.Helper()
	prompt, err := os.ReadFile(codexPromptPathFor(stateDir, orgID, org.LeaderIdentity))
	if err != nil {
		t.Fatalf("read %s's leader prompt: %v", orgID, err)
	}
	return string(prompt)
}

// TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask is AC1, AC5 and
// AC7 through the CLI and the real scripts/ralph-worktree.sh: start --plan
// --issue 12 records issue=12 at the end of the reservation's Details, after
// the feature digest, and writes the issue line after the line of the
// features it depends on in the leader's task; status shows `issue: #12`
// right after the feature line and --json the number 12 in the feature
// object. The same start again passes and records nothing, and a start
// again with another --issue or with none is refused, naming both, with
// nothing recorded. An org started without --issue (feature a) has no
// issue= token, no issue line in its task or status, and no issue key.
func TestOrgStartIssue_RecordedOnTheBindingAndTheLeadersTask(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state")
	approved := approvedFeatureSplitPlan(t)
	plan := writeFeatureSplitPlan(t, stateDir, approved)
	digest := org.PlanDigest([]byte(approved))
	wtB := filepath.Join(root, ".claude", "worktrees", "org-b")
	wtA := filepath.Join(root, ".claude", "worktrees", "org-a")

	stdout, stderr, err := startOrgFeature(t, plan, "b", "--state-dir", stateDir, "--issue", "12")
	if err != nil {
		t.Fatalf("start --plan --feature b --issue 12: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if !slices.Contains(outputLines(stdout), "worktree: "+wtB) {
		t.Errorf("expected the worktree %s, stdout:\n%s", wtB, stdout)
	}
	reserved := issueOrgReservation(t, stateDir, "b")
	tail := regexp.MustCompile(" branch=fix/b approval=per-plan feature_digest=[0-9a-f]{12} issue=12$")
	if reserved == nil || !tail.MatchString(reserved.Details) || strings.Count(reserved.Details, "issue=") != 1 {
		t.Fatalf("expected b's scope_reserved to end with the feature digest and then one issue=12, got %+v", reserved)
	}
	const issueLine = "- issue: #12(PR の本文に Closes #12 を書く)"
	if want := "\n- 依存する機能: a\n" + issueLine + "\n- 承認方式: per-plan\n"; !strings.Contains(issueLeaderPrompt(t, stateDir, "b"), want) {
		t.Errorf("expected the leader's task to have %q, got:\n%s", want, issueLeaderPrompt(t, stateDir, "b"))
	}

	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "b", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status b: %v", err)
	}
	wantFeature := "feature: demo/b branch fix/b worktree " + wtB
	lines := outputLines(stdout)
	if i := slices.Index(lines, wantFeature); i < 0 || i+1 >= len(lines) || lines[i+1] != "issue: #12" {
		t.Errorf("expected %q right after %q in status, got:\n%s", "issue: #12", wantFeature, stdout)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "b", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status b --json: %v", err)
	}
	var feature map[string]json.RawMessage
	if err := json.Unmarshal(orgStatusPayload(t, stdout)["feature"], &feature); err != nil {
		t.Fatalf("status --json feature: %v\n%s", err, stdout)
	}
	if got := string(feature["issue"]); got != "12" {
		t.Errorf("feature issue = %s, want the number 12\n%s", got, stdout)
	}
	if got := string(feature["digest"]); got != `"`+digest+`"` {
		t.Errorf("feature digest = %s, want %q", got, digest)
	}

	before := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	stdout, stderr, err = startOrgFeature(t, plan, "b", "--state-dir", stateDir, "--issue", "12")
	if err != nil {
		t.Fatalf("expected the same start with --issue 12 to pass again: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if !strings.Contains(stdout, `seat "leader" already spawned`) {
		t.Errorf("expected the existing leader, stdout:\n%s", stdout)
	}
	if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != before {
		t.Errorf("expected the same start to record nothing, %d -> %d events", before, got)
	}

	for _, tc := range []struct {
		name  string
		extra []string
		want  []string
	}{
		{"another --issue", []string{"--issue", "13"}, []string{
			`since org_id "b" was started with issue #12 and this start has issue #13`, "run it with --issue 12",
		}},
		{"no --issue", nil, []string{
			`since org_id "b" was started with issue #12 and this start has no issue`, "run it with --issue 12",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			herdrBefore := len(readLogLines(t, herdrLog))
			stdout, _, err := startOrgFeature(t, plan, "b", append([]string{"--state-dir", stateDir}, tc.extra...)...)
			if err == nil {
				t.Fatalf("expected the start refused, stdout: %s", stdout)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("expected the error to contain %q, got: %v", w, err)
				}
			}
			if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != before {
				t.Errorf("expected nothing recorded, %d -> %d events", before, got)
			}
			if got := readLogLines(t, herdrLog); slices.ContainsFunc(got[herdrBefore:], func(l string) bool {
				return strings.HasPrefix(l, "workspace create") || strings.HasPrefix(l, "pane split")
			}) {
				t.Errorf("expected no pane made, herdr calls: %v", got[herdrBefore:])
			}
		})
	}

	stdout, stderr, err = startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("start --plan --feature a: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if reserved := issueOrgReservation(t, stateDir, "a"); reserved == nil || strings.Contains(reserved.Details, "issue=") ||
		!regexp.MustCompile(" feature_digest=[0-9a-f]{12}$").MatchString(reserved.Details) {
		t.Errorf("expected a's scope_reserved to end with the feature digest and have no issue=, got %+v", reserved)
	}
	// The role prompt around the task may name the issue line in its steps,
	// so the check is on the task's own lines.
	if prompt := issueLeaderPrompt(t, stateDir, "a"); regexp.MustCompile(`(?m)^- issue: #[0-9]`).MatchString(prompt) ||
		!strings.Contains(prompt, "\n- 依存する機能: なし\n- 承認方式: per-plan\n") {
		t.Errorf("expected a's task with no issue line, got:\n%s", prompt)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status a: %v", err)
	}
	if lines := outputLines(stdout); lines[len(lines)-1] != "feature: demo/a branch feat/a worktree "+wtA ||
		slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "issue") }) {
		t.Errorf("expected a's status to end with its feature line and have no issue line, got:\n%s", stdout)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status a --json: %v", err)
	}
	var featureA map[string]json.RawMessage
	if err := json.Unmarshal(orgStatusPayload(t, stdout)["feature"], &featureA); err != nil {
		t.Fatalf("status a --json feature: %v\n%s", err, stdout)
	}
	if keys := slices.Sorted(maps.Keys(featureA)); !slices.Equal(keys, []string{"branch", "digest", "feature", "feature_digest", "split", "worktree"}) {
		t.Errorf("a's feature keys = %v, want no issue key", keys)
	}
}

// TestOrgStatus_IssueLineAndJSON is AC7's edges on seeded records: a
// binding with an issue shows `issue: #<number>` right after the feature
// line and the number in the JSON feature object, also when the binding was
// read from a damaged record (the feature line then ends with (incomplete
// record) and the object has "incomplete": true); an issue= token that is
// not an issue number, or that is given twice, reads as no issue, and a
// record with no issue= token shows the lines and the keys it showed before
// the issue.
func TestOrgStatus_IssueLineAndJSON(t *testing.T) {
	setupOrgStubPATH(t)
	const wt = "/repo/.claude/worktrees/org-x"
	const bound = "paths=docs/c.md split=demo feature=c digest=0123456789ab branch=fix/c approval=per-plan feature_digest=abcdef012345"
	const featureLine = "feature: demo/c branch fix/c worktree " + wt
	complete := map[string]any{
		"split": "demo", "feature": "c", "digest": "0123456789ab", "feature_digest": "abcdef012345", "branch": "fix/c", "worktree": wt,
	}
	withKeys := func(m map[string]any, kv ...any) map[string]any {
		out := maps.Clone(m)
		for i := 0; i+1 < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	for _, tc := range []struct {
		name, details string
		wantLines     []string
		wantJSON      map[string]any
	}{
		{"an issue", bound + " issue=12",
			[]string{"no seats", "reserved: docs/c.md", featureLine, "issue: #12"}, withKeys(complete, "issue", float64(12))},
		{"a 10-digit issue", bound + " issue=9999999999",
			[]string{"no seats", "reserved: docs/c.md", featureLine, "issue: #9999999999"}, withKeys(complete, "issue", float64(9999999999))},
		{"no issue", bound,
			[]string{"no seats", "reserved: docs/c.md", featureLine}, complete},
		{"an issue that is not a number", bound + " issue=abc",
			[]string{"no seats", "reserved: docs/c.md", featureLine}, complete},
		{"an issue with a leading zero", bound + " issue=012",
			[]string{"no seats", "reserved: docs/c.md", featureLine}, complete},
		{"an issue given twice", bound + " issue=12 issue=13",
			[]string{"no seats", "reserved: docs/c.md", featureLine}, complete},
		{"an issue on a damaged record", "paths=docs/c.md split=demo feature=c branch=fix/c issue=7",
			[]string{"no seats", "reserved: docs/c.md", featureLine + " (incomplete record)", "issue: #7"},
			map[string]any{"split": "demo", "feature": "c", "digest": "", "branch": "fix/c", "worktree": wt, "issue": float64(7), "incomplete": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "x", Event: org.EventScopeReserved, Worktree: wt, Details: tc.details})

			stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "x", "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status x: %v", err)
			}
			if got := outputLines(stdout); !slices.Equal(got, tc.wantLines) {
				t.Errorf("status = %q, want %q", got, tc.wantLines)
			}
			stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "x", "--state-dir", stateDir, "--json")
			if err != nil {
				t.Fatalf("status x --json: %v", err)
			}
			var feature map[string]any
			if err := json.Unmarshal(orgStatusPayload(t, stdout)["feature"], &feature); err != nil {
				t.Fatalf("status --json feature: %v\n%s", err, stdout)
			}
			if !maps.Equal(feature, tc.wantJSON) {
				t.Errorf("feature = %v, want %v", feature, tc.wantJSON)
			}
		})
	}
}

// TestOrgMergeOrder_JSONBindingHasTheIssue is AC10 through the CLI: the
// binding of a branch whose org was started with --issue has the number in
// merge-order --json, after the org is disbanded too, a branch whose org was
// started without it has no issue key, and the text shows no issue.
func TestOrgMergeOrder_JSONBindingHasTheIssue(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	main := fx.tip(t, "main")
	issued := fx.branch(t, "feat/issued")
	plain := fx.branch(t, "feat/plain")
	fx.push(t, "feat/issued", "feat/plain")
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
		OrgID: "i", Event: org.EventScopeReserved, Worktree: fx.work,
		Details: "paths=internal/i/ split=demo feature=issued digest=0123456789ab branch=feat/issued approval=per-plan " +
			"feature_digest=abcdef012345 issue=365",
	})
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "i", Event: org.EventDisbanded})
	fx.bind(t, "p", "plain", "feat/plain")
	fx.gate(t, "i", "feat/issued", "", issued, main)
	fx.gate(t, "p", "feat/plain", "", plain, main)

	res, raw := fx.runJSON(t, "--all")
	byBranch := make(map[string]org.MergeOrderEntry)
	for _, e := range res.Entries {
		byBranch[e.Branch] = e
	}
	if b := byBranch["feat/issued"].Binding; b == nil || b.OrgID != "i" || b.Issue != 365 {
		t.Errorf("feat/issued's binding = %+v; want org i's with issue 365", b)
	}
	if b := byBranch["feat/plain"].Binding; b == nil || b.OrgID != "p" || b.Issue != 0 {
		t.Errorf("feat/plain's binding = %+v; want org p's with no issue", b)
	}
	var got struct {
		Entries []struct {
			Branch  string                     `json:"branch"`
			Binding map[string]json.RawMessage `json:"binding"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, e := range got.Entries {
		issue, ok := e.Binding["issue"]
		switch e.Branch {
		case "feat/issued":
			seen++
			if string(issue) != "365" {
				t.Errorf("feat/issued's binding issue = %s (present %v); want the number 365", issue, ok)
			}
		case "feat/plain":
			seen++
			if ok {
				t.Errorf("feat/plain's binding has issue %s; want the key left out", issue)
			}
		}
	}
	if seen != 2 {
		t.Fatalf("saw %d of the 2 branches in --json:\n%s", seen, raw)
	}

	stdout, stderr, err := fx.run(t, "--all")
	if err != nil || stderr != "" {
		t.Fatalf("merge-order --all: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	// The branch names hold the word "issued", so the check is on the fields:
	// each row is the one a branch with no issue gets, and no note names it.
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/issued", "-", short(issued), short(main), "i/issued", "the gate passed on " + short(issued)},
		{"ready", "feat/plain", "-", short(plain), short(main), "p/plain", "the gate passed on " + short(plain)},
	})
	for i, sha := range []string{issued, plain} {
		want := "the gate passed on " + short(sha) + ", origin's tip of the branch, with base " + short(main) + ", origin/main's tip"
		if len(out.rows) == 2 && out.rows[i][7] != want {
			t.Errorf("row %d's detail = %q; want %q", i+1, out.rows[i][7], want)
		}
	}
	for _, note := range out.notes {
		if strings.Contains(note, "#365") || strings.Contains(note, "issue ") {
			t.Errorf("expected no note to name the issue, got %q", note)
		}
	}
}

// TestOrgStartIssueHelp_DescribesTheFlag: `ralph org start --help` has the
// --issue paragraph and flag, `ralph org status --help` names the issue line
// and key, and `ralph org merge-order --help` names the binding's issue (the
// help of slice S2 of plan 2026-11-22-org-start-issue).
func TestOrgStartIssueHelp_DescribesTheFlag(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"start", "--help"}, []string{
			"ralph org start (<task> | --plan <split plan> --feature <slug> [--issue <number>])",
			"`ralph org start --plan <split plan> --feature <slug> --issue <number>` starts the org for a feature that was filed as an issue: " +
				"give the issue's number alone (12 for issue #12; 1 to 10 decimal digits with no leading zero and no #), " +
				"and a value of another form is refused with nothing made.",
			"The org's binding records the number, `ralph org status` shows it on an `issue: #<number>` line, " +
				"`ralph org merge-order --json` gives it in the branch's binding, also after the org is disbanded, " +
				"and the leader's task has an `- issue:` line that tells the leader to write `Closes #<number>` in the feature's PR body, " +
				"so that the merge of the PR closes the issue.",
			"Starting an org that was not disbanded again needs the --issue it was started with, or none when it was started without one: " +
				"another number, or a start that adds or leaves out --issue, is refused, and `ralph org reseat` starts its leader " +
				"with the recorded number. --issue needs --plan.",
			"--issue string number of the issue the feature was filed as, alone (12 for issue #12), when the org is started for one: " +
				"the leader writes Closes #<number> in the feature's PR body; needs --plan, and starting an org that was not " +
				"disbanded again needs the number it was started with",
		}},
		{[]string{"status", "--help"}, []string{
			"For an org started with `ralph org start --plan --issue`, an `issue: #<number>` line follows the feature line " +
				"and the object has `issue`, the number; an org started without --issue has neither.",
		}},
		{[]string{"merge-order", "--help"}, []string{
			"A binding of an org started with `ralph org start --plan --issue` also has issue, the issue's number, which the text " +
				"does not show: it is the binding's, so it stays after the org is disbanded, and a binding with no issue has no issue key.",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%s: %v", strings.Join(tc.args, " "), err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%s does not say %q:\n%s", strings.Join(tc.args, " "), want, out)
			}
		}
	}
}
