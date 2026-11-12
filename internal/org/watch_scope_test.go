package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// The tests of the scope-change condition of an org that holds a
// reservation (plan 2026-11-12-watch-alert-dedupe, D5, D6, AC8 to AC12,
// AC14, and AC15).

func TestParsePorcelainZ_EntriesInOrderWithBothSidesOfARename(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{"empty", "", nil},
		{"modified in the worktree and in the index", " M src/a.go\x00M  src/b.go\x00MM src/c.go\x00",
			[]string{" M src/a.go", "M  src/b.go", "MM src/c.go"}},
		{"added and deleted", "A  src/new.go\x00 D other/x.go\x00D  other/y.go\x00",
			[]string{"A  src/new.go", " D other/x.go", "D  other/y.go"}},
		{"untracked files of a new directory, one by one", "?? newdir/sub/n.txt\x00?? newdir/m.txt\x00",
			[]string{"?? newdir/sub/n.txt", "?? newdir/m.txt"}},
		{"a rename gives the new path, then the original", "R  src/x.go\x00other/x.go\x00 M src/a.go\x00",
			[]string{"R  src/x.go", "R  other/x.go", " M src/a.go"}},
		{"a renamed and modified entry", "RM src/x.go\x00other/x.go\x00",
			[]string{"RM src/x.go", "RM other/x.go"}},
		{"a rename in the worktree", " R new.go\x00old.go\x00",
			[]string{" R new.go", " R old.go"}},
		{"a copy gives both paths", "C  b.go\x00a.go\x00",
			[]string{"C  b.go", "C  a.go"}},
		{"the original of a rename is never read as an entry", "R  new.go\x00?? weird\x00",
			[]string{"R  new.go", "R  ?? weird"}},
		{"a rename with no original field left", "R  x.go\x00",
			[]string{"R  x.go"}},
		{"spaces, non-ASCII, and a newline are kept as they are", "?? other/s p.go\x00?? docs/日本.md\x00?? other/a\nb.go\x00",
			[]string{"?? other/s p.go", "?? docs/日本.md", "?? other/a\nb.go"}},
		{"fields too short to be an entry are skipped", "x\x00?? a\x00\x00 M b\x00",
			[]string{"?? a", " M b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePorcelainZ(tt.out); !slices.Equal(got, tt.want) {
				t.Errorf("parsePorcelainZ(%q) = %q; want %q", tt.out, got, tt.want)
			}
		})
	}
}

func TestScopePathInside_FollowsTheReservationRules(t *testing.T) {
	tests := []struct {
		reserved []string
		path     string
		want     bool
	}{
		{[]string{"."}, "any/x.go", true},
		{[]string{"."}, "x", true},
		{[]string{"."}, "sub/", true},
		{[]string{"src/"}, "src/a.go", true},
		{[]string{"src/"}, "src/deep/x.go", true},
		{[]string{"src/"}, "srcx/a.go", false},
		{[]string{"src/"}, "src", false},
		{[]string{"src/"}, "other/a.go", false},
		{[]string{"src/main.go"}, "src/main.go", true},
		{[]string{"src/main.go"}, "src/main.go.bak", false},
		{[]string{"src/main.go"}, "src/main", false},
		// Only the prefix is the same: not inside.
		{[]string{"src/ab/"}, "src/a", false},
		{[]string{"src/ab/"}, "src/ab", false},
		{[]string{"src/a/"}, "src/ab.go", false},
		{[]string{"src/a/"}, "src/a/b.go", true},
		// A directory entry (an untracked repository inside the worktree).
		{[]string{"src/"}, "src/sub/", true},
		{[]string{"src/a/"}, "src/", false},
		{[]string{"src/a.go"}, "src/", false},
		{[]string{"docs/x.md", "src/"}, "docs/x.md", true},
		{[]string{"docs/x.md", "src/"}, "docs/y.md", false},
		{nil, "src/a.go", false},
	}
	for _, tt := range tests {
		if got := scopePathInside(tt.reserved, tt.path); got != tt.want {
			t.Errorf("scopePathInside(%q, %q) = %v; want %v", tt.reserved, tt.path, got, tt.want)
		}
	}
}

func TestOwnRecordCandidate_PlansAndReportsButNotTheirTemplates(t *testing.T) {
	for _, p := range []string{
		"docs/plans/active/2026-11-12-x.md",
		"docs/plans/archive/2026-11-12-x.md",
		"docs/reports/review-x.md",
		"docs/reports/sub/x.md",
		"docs/plans/templatesx/x.md",
	} {
		if !ownRecordCandidate(p) {
			t.Errorf("ownRecordCandidate(%q) = false; want true", p)
		}
	}
	for _, p := range []string{
		"docs/plans/templates/feature-plan.md",
		"docs/reports/templates/t.md",
		"docs/specs/x.md",
		"docs/evidence/x.md",
		"docs/plans",
		"docs/plansx/a.md",
		"docs/reports/",
		"docs/plans/sub/",
		"other/docs/plans/x.md",
		"src/x.go",
	} {
		if ownRecordCandidate(p) {
			t.Errorf("ownRecordCandidate(%q) = true; want false", p)
		}
	}
}

func TestScopeOutsideNote_EscapesEachPathAndFitsTheRecurredNote(t *testing.T) {
	t.Run("the paths in order", func(t *testing.T) {
		if got, want := scopeOutsideNote([]string{"other/x.go", "other/y.go"}), "outside reservation: other/x.go, other/y.go"; got != want {
			t.Errorf("scopeOutsideNote = %q; want %q", got, want)
		}
		if got := scopeOutsideNote(nil); got != "" {
			t.Errorf("scopeOutsideNote(nil) = %q; want no note", got)
		}
	})

	t.Run("control characters, line breaks, and bytes that are not UTF-8", func(t *testing.T) {
		paths := []string{"a\nb", "c\x1bd", "e\u2028f", "g\xffh", "日本/x.go", "tab\there", "cr\rx"}
		want := `outside reservation: a\nb, c\x1bd, e\u2028f, g\xffh, 日本/x.go, tab\there, cr\rx`
		got := scopeOutsideNote(paths)
		if got != want {
			t.Errorf("scopeOutsideNote = %q; want %q", got, want)
		}
		if err := validateInboxNoteLine("recurred", got); err != nil {
			t.Errorf("the note %q is refused as a recurred note: %v", got, err)
		}
	})

	t.Run("exactly the longest note is kept whole", func(t *testing.T) {
		path := strings.Repeat("日", InboxNoteMaxRunes-utf8.RuneCountInString(scopeNotePrefix))
		got := scopeOutsideNote([]string{path})
		if got != scopeNotePrefix+path {
			t.Errorf("a note of %d characters was cut: %q", InboxNoteMaxRunes, got)
		}
		if err := validateInboxNoteLine("recurred", got); err != nil {
			t.Errorf("the note is refused as a recurred note: %v", err)
		}
	})

	t.Run("one character more leaves the path out", func(t *testing.T) {
		path := strings.Repeat("a", InboxNoteMaxRunes-utf8.RuneCountInString(scopeNotePrefix)+1)
		if got, want := scopeOutsideNote([]string{path}), scopeNotePrefix+"… (+1 more)"; got != want {
			t.Errorf("scopeOutsideNote = %q; want %q", got, want)
		}
	})

	t.Run("many paths keep as many as fit and count the rest", func(t *testing.T) {
		var paths []string
		for i := range 100 {
			paths = append(paths, fmt.Sprintf("other/file-%03d.go", i))
		}
		got := scopeOutsideNote(paths)
		if n := utf8.RuneCountInString(got); n > InboxNoteMaxRunes {
			t.Fatalf("the note is %d characters; want at most %d: %q", n, InboxNoteMaxRunes, got)
		}
		if err := validateInboxNoteLine("recurred", got); err != nil {
			t.Fatalf("the note is refused as a recurred note: %v", err)
		}
		m := regexp.MustCompile(`^outside reservation: (.*), … \(\+(\d+) more\)$`).FindStringSubmatch(got)
		if m == nil {
			t.Fatalf("the note %q does not end with the count of the paths it left out", got)
		}
		kept := strings.Split(m[1], ", ")
		if !slices.Equal(kept, paths[:len(kept)]) {
			t.Errorf("the note keeps %q; want the first paths in order", kept)
		}
		if want := fmt.Sprint(len(paths) - len(kept)); m[2] != want {
			t.Errorf("the note says +%s more after %d paths; want +%s", m[2], len(kept), want)
		}
		// One more path would not have fit with the count of the rest.
		longer := scopeNotePrefix + strings.Join(paths[:len(kept)+1], ", ") + ", " + scopeNoteMore(len(paths)-len(kept)-1)
		if utf8.RuneCountInString(longer) <= InboxNoteMaxRunes {
			t.Errorf("the note keeps %d paths, but %d fit: %q", len(kept), len(kept)+1, longer)
		}
	})
}

// realGitRun runs git with --no-optional-locks before the arguments, returns
// its output, and turns a non-zero exit into an error.
func TestWatchRealGitRun_RunsGitWithoutOptionalLocks(t *testing.T) {
	stubDir := t.TempDir()
	script := "#!/bin/sh\nprintf '%s|' \"$@\"\n[ \"$2\" = fail ] && exit 3\nexit 0\n"
	if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	o := &Org{DriverCallTimeout: 5 * time.Second}

	out, err := o.realGitRun(t.TempDir(), "status", "--porcelain=v1", "-z")
	if err != nil || out != "--no-optional-locks|status|--porcelain=v1|-z|" {
		t.Errorf("realGitRun = %q, %v; want git run with --no-optional-locks and the arguments", out, err)
	}
	if _, err := o.realGitRun(t.TempDir(), "fail"); err == nil {
		t.Error("realGitRun of a git that exits 3 succeeded; want an error")
	}
}

// scopeBaseFiles are the files of the base commit of scopeGitRepo, each
// with its own content so git pairs no rename by content by chance.
var scopeBaseFiles = []string{
	"src/keep.go",
	"src/old.go",
	"other/x.go",
	"other/y.go",
	"docs/plans/active/other-feature.md",
	"docs/plans/templates/feature-plan.md",
	"docs/reports/templates/t.md",
	"docs/reports/old-report.md",
}

// scopeGitRepo makes a git repository in a new temp dir whose one commit,
// the base, holds scopeBaseFiles, with git's configuration isolated from
// the user's. origin says which of origin's refs point at the base: "main"
// (refs/remotes/origin/main), "master", "origin/HEAD" (refs/remotes/origin/
// trunk, with refs/remotes/origin/HEAD pointing at it), or "" (none, so the
// merge base cannot be found), and "unborn" makes no commit at all. The
// test is skipped when git is not on PATH.
func scopeGitRepo(t *testing.T, origin string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	isolateGitEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // no user excludes file either
	repo := t.TempDir()
	runGit(t, "-c", "init.defaultBranch=main", "init", "--quiet", repo)
	for _, kv := range [][2]string{{"user.name", "t"}, {"user.email", "t@example.com"}, {"commit.gpgsign", "false"}} {
		scopeGit(t, repo, "config", kv[0], kv[1])
	}
	if origin == "unborn" {
		return repo
	}
	for _, rel := range scopeBaseFiles {
		writeGateFile(t, repo, rel, "// "+rel+"\n", 0o644)
	}
	scopeCommitAll(t, repo, "base")
	switch origin {
	case "main", "master":
		scopeGit(t, repo, "update-ref", "refs/remotes/origin/"+origin, "HEAD")
	case "origin/HEAD":
		scopeGit(t, repo, "update-ref", "refs/remotes/origin/trunk", "HEAD")
		scopeGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	case "":
	default:
		t.Fatalf("scopeGitRepo: unknown origin %q", origin)
	}
	return repo
}

// scopeGit runs git -C repo args and fails the test on a non-zero exit.
func scopeGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	runGit(t, append([]string{"-C", repo}, args...)...)
}

// scopeCommitAll stages everything in repo and commits it.
func scopeCommitAll(t *testing.T, repo, msg string) {
	t.Helper()
	scopeGit(t, repo, "add", "-A")
	scopeGit(t, repo, "commit", "--quiet", "-m", msg)
}

// scopeWrite writes a file at rel in repo, with content that names it and
// tag, so writing it again with another tag changes it.
func scopeWrite(t *testing.T, repo, rel, tag string) {
	t.Helper()
	writeGateFile(t, repo, rel, "// "+rel+" "+tag+"\n", 0o644)
}

// reservedScopeFixture is org-a with only its leader, which holds the
// reservation and whose worktree is a real repository, and a watchRun that
// reads git through the real GitRun (realGitRun). The test drives
// evaluateCycle itself, one fake-clock minute apart, so no stall comes up.
type reservedScopeFixture struct {
	o      *Org
	a      *fakeWatchAgmsg
	clk    *fakeClock
	dir    string
	stderr *bytes.Buffer
	run    *watchRun
	status *watchStatusFile
}

func newReservedScopeFixture(t *testing.T, repo string, reserve ...string) *reservedScopeFixture {
	t.Helper()
	return newReservedScopeFixtureAt(t, repo, reserve...)
}

// newReservedScopeFixtureAt is newReservedScopeFixture for a leader whose
// cwd is cwd, a directory of the repository (its root or a subdirectory).
func newReservedScopeFixtureAt(t *testing.T, cwd string, reserve ...string) *reservedScopeFixture {
	t.Helper()
	o, _, a, clk := testWatchOrg(t)
	p := watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)
	p.Cwd, p.Reserve = cwd, reserve
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn the leader with a reservation: %+v", r)
	}
	dir := t.TempDir()
	f := &reservedScopeFixture{o: o, a: a, clk: clk, dir: dir, stderr: &bytes.Buffer{}}
	f.run = &watchRun{
		org: o, cfg: o.Config,
		gitStatus: func(cwd string) (string, error) {
			t.Errorf("GitStatus was called for %s; an org with a reservation reads git through GitRun", cwd)
			return "", nil
		},
		gitRun:          o.realGitRun,
		escalateFn:      func(context.Context, string) error { return nil },
		stderr:          f.stderr,
		statusPath:      filepath.Join(dir, WatchStatusFileName("org-a")),
		escalationsPath: filepath.Join(dir, EscalationsRelName),
	}
	status, err := loadWatchStatus(f.run.statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}
	f.status = status
	return f
}

// cycle runs one pulse cycle of org-a a minute after the previous one.
func (f *reservedScopeFixture) cycle(t *testing.T) {
	t.Helper()
	f.clk.Advance(time.Minute)
	if err := f.run.evaluateCycle("org-a", f.status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
}

// scopeAlerts returns the scope_change ALERTs the watch sent.
func (f *reservedScopeFixture) scopeAlerts() []string {
	var out []string
	for _, c := range watchdogAlerts(f.a) {
		if strings.Contains(c.message, "CONDITION: "+condScopeChange) {
			out = append(out, c.message)
		}
	}
	return out
}

// scopeLead is the line of a scope_change ALERT of the leader before the
// paths outside the reservation.
const scopeLead = "seat " + LeaderIdentity + " changed paths outside the org's reservation:\n"

// outsidePaths returns the paths a scope_change ALERT of the leader names,
// one per line after scopeLead, or fails the test when it has no such line.
func outsidePaths(t *testing.T, message string) []string {
	t.Helper()
	_, list, ok := strings.Cut(message, scopeLead)
	if !ok {
		t.Fatalf("the scope_change ALERT has no line %q:\n%s", strings.TrimSuffix(scopeLead, "\n"), message)
	}
	return strings.Split(list, "\n")
}

// wantOneScopeAlert fails the test unless the watch sent exactly one
// scope_change ALERT, naming exactly want.
func (f *reservedScopeFixture) wantOneScopeAlert(t *testing.T, want ...string) {
	t.Helper()
	alerts := f.scopeAlerts()
	if len(alerts) != 1 {
		t.Fatalf("scope_change ALERTs = %q; want one naming %q", alerts, want)
	}
	if got := outsidePaths(t, alerts[0]); !slices.Equal(got, want) {
		t.Errorf("the scope_change ALERT names %q; want %q:\n%s", got, want, alerts[0])
	}
}

// AC8: in an org that holds a reservation, changes inside it raise no
// scope_change, whether they are an edit, a new file in a new directory, a
// rename, or a commit. A change outside it afterwards still raises one.
func TestWatch_ReservedScope_ChangesInsideTheReservationRaiseNothing(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t) // the baseline
	steps := []struct {
		name string
		do   func()
	}{
		{"an edit", func() { scopeWrite(t, repo, "src/keep.go", "edited") }},
		{"a new file in a new directory", func() { scopeWrite(t, repo, "src/new/deep/n.go", "new") }},
		{"a rename", func() { scopeGit(t, repo, "mv", "src/old.go", "src/renamed.go") }},
		{"a commit", func() { scopeCommitAll(t, repo, "inside") }},
		{"a move inside committed in one cycle", func() {
			scopeGit(t, repo, "mv", "src/renamed.go", "src/again.go")
			scopeCommitAll(t, repo, "move inside")
		}},
	}
	for _, s := range steps {
		s.do()
		f.cycle(t)
		if alerts := f.scopeAlerts(); len(alerts) != 0 {
			t.Fatalf("after %s: scope_change ALERTs = %q; want none", s.name, alerts)
		}
	}
	snap := f.status.SeatSnapshots[LeaderIdentity]
	if head := gitOut(t, repo, "rev-parse", "HEAD"); snap == nil || !snap.ScopeOutsidePathsSeen || snap.ScopeHead != head || len(snap.ScopeOutsidePaths) != 0 {
		t.Fatalf("the leader's snapshot = %+v; want the baseline seen, at HEAD %s, with nothing outside", snap, head)
	}

	scopeWrite(t, repo, "other/x.go", "edited")
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/x.go")
}

// AC8 and D5: the org's own new plan and reports, which the role prompts
// have the seats write outside the reservation, raise no scope_change: new,
// committed, moved from active to archive, and that move committed. Each way
// of finding origin's default branch is tried.
func TestWatch_ReservedScope_TheOrgsOwnNewPlanAndReportsRaiseNothing(t *testing.T) {
	for _, origin := range []string{"main", "master", "origin/HEAD"} {
		t.Run(origin, func(t *testing.T) {
			repo := scopeGitRepo(t, origin)
			f := newReservedScopeFixture(t, repo, "src/")
			f.cycle(t)
			steps := []struct {
				name string
				do   func()
			}{
				{"a new plan and reports", func() {
					scopeWrite(t, repo, "docs/plans/active/2026-11-12-feat.md", "plan")
					scopeWrite(t, repo, "docs/reports/review-2026-11-12-org-a-reviewer.md", "review")
					scopeWrite(t, repo, "docs/reports/sync-docs-2026-11-12-feat.md", "sync-docs")
					scopeWrite(t, repo, "docs/reports/org-manifest-org-a-2026-11-12.md", "report")
				}},
				{"their commit", func() { scopeCommitAll(t, repo, "records") }},
				{"the plan moved to archive", func() {
					mkdirAll(t, filepath.Join(repo, "docs", "plans", "archive"))
					scopeGit(t, repo, "mv", "docs/plans/active/2026-11-12-feat.md", "docs/plans/archive/2026-11-12-feat.md")
				}},
				{"the move committed", func() { scopeCommitAll(t, repo, "archive") }},
				{"a report rewritten and committed in one cycle", func() {
					scopeWrite(t, repo, "docs/reports/review-2026-11-12-org-a-reviewer.md", "again")
					scopeCommitAll(t, repo, "review again")
				}},
			}
			for _, s := range steps {
				s.do()
				f.cycle(t)
				if alerts := f.scopeAlerts(); len(alerts) != 0 {
					t.Fatalf("after %s: scope_change ALERTs = %q; want none", s.name, alerts)
				}
			}

			scopeWrite(t, repo, "other/x.go", "edited")
			f.cycle(t)
			f.wantOneScopeAlert(t, "other/x.go")
		})
	}
}

// AC9 and D6: a change outside the reservation raises scope_change, and the
// ALERT names the paths outside it and none inside: an edit, a file in a
// new directory, the original side of a rename into the reservation, a
// commit since the previous cycle with a clean worktree, and a move into
// the reservation committed within one cycle (the plan advisory's finding
// 1: a diff with rename detection would name only the new path).
func TestWatch_ReservedScope_AChangeOutsideRaisesScopeChangeNamingItsPaths(t *testing.T) {
	tests := []struct {
		name  string
		do    func(t *testing.T, repo string)
		clean bool // the worktree is clean afterwards: only the commit shows the change
		want  []string
	}{
		{"an edit", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "other/x.go", "edited")
		}, false, []string{"other/x.go"}},
		{"a file in a new directory", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "other/newdir/deep/n.go", "new")
		}, false, []string{"other/newdir/deep/n.go"}},
		{"one side of a rename", func(t *testing.T, repo string) {
			scopeGit(t, repo, "mv", "other/y.go", "src/y.go")
		}, false, []string{"other/y.go"}},
		{"a commit since the previous cycle", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "other/x.go", "edited")
			scopeWrite(t, repo, "src/keep.go", "edited")
			scopeCommitAll(t, repo, "outside and inside")
		}, true, []string{"other/x.go"}},
		{"a move into the reservation committed in one cycle", func(t *testing.T, repo string) {
			scopeGit(t, repo, "mv", "other/x.go", "src/x.go")
			scopeCommitAll(t, repo, "move in")
		}, true, []string{"other/x.go"}},
		{"a path with a space and non-ASCII", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "other/s p/日本.go", "new")
		}, false, []string{"other/s p/日本.go"}},
		{"an edit outside next to the org's own new record", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/plans/active/2026-11-12-feat.md", "plan")
			scopeWrite(t, repo, "other/x.go", "edited")
		}, false, []string{"other/x.go"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := scopeGitRepo(t, "main")
			f := newReservedScopeFixture(t, repo, "src/")
			f.cycle(t)
			tt.do(t, repo)
			if dirty := gitOut(t, repo, "status", "--porcelain"); tt.clean && dirty != "" {
				t.Fatalf("the worktree is not clean after the commit:\n%s", dirty)
			}
			f.cycle(t)
			f.wantOneScopeAlert(t, tt.want...)
		})
	}
}

// AC14: a record that existed at the merge base is not the org's own, so
// changing or deleting one outside the reservation raises scope_change: a
// template, another feature's plan, an earlier report, and any new file
// under a templates directory. With no origin ref, the base is unknown, and
// a new plan or report outside the reservation raises it too.
func TestWatch_ReservedScope_RecordsTheBaseHadAndAnUnknownBaseRaiseScopeChange(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		do     func(t *testing.T, repo string)
		want   []string
	}{
		{"a template the base had", "main", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/reports/templates/t.md", "edited")
		}, []string{"docs/reports/templates/t.md"}},
		{"another feature's plan", "main", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/plans/active/other-feature.md", "edited")
		}, []string{"docs/plans/active/other-feature.md"}},
		{"an earlier report deleted", "main", func(t *testing.T, repo string) {
			scopeGit(t, repo, "rm", "--quiet", "docs/reports/old-report.md")
		}, []string{"docs/reports/old-report.md"}},
		{"another feature's plan edited and committed", "main", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/plans/active/other-feature.md", "edited")
			scopeCommitAll(t, repo, "edit another plan")
		}, []string{"docs/plans/active/other-feature.md"}},
		{"a new file under docs/plans/templates", "main", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/plans/templates/new.md", "new")
		}, []string{"docs/plans/templates/new.md"}},
		{"a template next to the org's own new report", "main", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/reports/review-2026-11-12-org-a-reviewer.md", "review")
			scopeWrite(t, repo, "docs/reports/templates/t.md", "edited")
		}, []string{"docs/reports/templates/t.md"}},
		{"a new report with no origin ref", "", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/reports/review-2026-11-12-org-a-reviewer.md", "review")
		}, []string{"docs/reports/review-2026-11-12-org-a-reviewer.md"}},
		{"a new plan with no origin ref", "", func(t *testing.T, repo string) {
			scopeWrite(t, repo, "docs/plans/active/2026-11-12-feat.md", "plan")
		}, []string{"docs/plans/active/2026-11-12-feat.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := scopeGitRepo(t, tt.origin)
			f := newReservedScopeFixture(t, repo, "src/")
			f.cycle(t)
			tt.do(t, repo)
			f.cycle(t)
			f.wantOneScopeAlert(t, tt.want...)
		})
	}
}

// scopeAdvanceOrigin makes a commit on top of refs/remotes/origin/main that
// writes files, as a feature merged upstream would, moves that ref to it,
// and checks repo's branch out again. The worktree must be clean.
func scopeAdvanceOrigin(t *testing.T, repo string, files ...string) {
	t.Helper()
	branch := gitOut(t, repo, "branch", "--show-current")
	scopeGit(t, repo, "checkout", "--quiet", "--detach", "refs/remotes/origin/main")
	for _, rel := range files {
		scopeWrite(t, repo, rel, "upstream")
	}
	scopeCommitAll(t, repo, "upstream")
	scopeGit(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	scopeGit(t, repo, "checkout", "--quiet", branch)
}

// AC15 and D6: bringing origin's default branch into the branch, by the
// rebase the leader runs before its PR or by a merge, raises no
// scope_change for the upstream commits' paths outside the reservation,
// although HEAD moved past them since the previous cycle. A change outside
// afterwards still raises one.
func TestWatch_ReservedScope_UpstreamCommitsBroughtInRaiseNothing(t *testing.T) {
	for _, way := range []struct {
		name string
		args []string
	}{
		{"a rebase onto origin's default branch", []string{"rebase", "--quiet", "refs/remotes/origin/main"}},
		{"a merge of origin's default branch", []string{"merge", "--quiet", "--no-edit", "refs/remotes/origin/main"}},
	} {
		t.Run(way.name, func(t *testing.T) {
			repo := scopeGitRepo(t, "main")
			scopeWrite(t, repo, "src/x.go", "own")
			scopeCommitAll(t, repo, "own work")
			f := newReservedScopeFixture(t, repo, "src/")
			f.cycle(t) // the baseline
			prev := gitOut(t, repo, "rev-parse", "HEAD")

			scopeAdvanceOrigin(t, repo, "other/x.go", "other/new.go")
			scopeGit(t, repo, way.args...)
			if diff := gitOut(t, repo, "diff", "--name-only", prev, "HEAD"); !strings.Contains(diff, "other/new.go") {
				t.Fatalf("the diff from the previous HEAD = %q; want upstream's other/new.go in it", diff)
			}
			f.cycle(t)
			if alerts := f.scopeAlerts(); len(alerts) != 0 {
				t.Fatalf("scope_change ALERTs after %s = %q; want none", way.name, alerts)
			}
			if head, snap := gitOut(t, repo, "rev-parse", "HEAD"), f.status.SeatSnapshots[LeaderIdentity]; snap.ScopeHead != head {
				t.Errorf("ScopeHead = %q; want it moved on to HEAD %s", snap.ScopeHead, head)
			}

			scopeWrite(t, repo, "other/y.go", "edited")
			f.cycle(t)
			f.wantOneScopeAlert(t, "other/y.go")
		})
	}
}

// AC15 and D6: the org's own commit that a rebase onto origin's default
// branch rewrites is not on that branch, so its path outside the
// reservation raises scope_change, and upstream's paths stay out of the
// ALERT.
func TestWatch_ReservedScope_ARebasedOwnCommitOutsideRaisesScopeChange(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	scopeWrite(t, repo, "src/x.go", "own")
	scopeWrite(t, repo, "other/y.go", "own")
	scopeCommitAll(t, repo, "own work, one path outside")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t) // the baseline: a clean worktree

	scopeAdvanceOrigin(t, repo, "other/x.go", "other/new.go")
	scopeGit(t, repo, "rebase", "--quiet", "refs/remotes/origin/main")
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/y.go")
}

// AC9 and D6, the case the cycle 1 cross-review of plan
// 2026-11-12-watch-alert-dedupe found: a merge's own edit, here a conflict
// resolved in a path outside the reservation, is the org's write, so once
// the merge is committed and the worktree is clean it raises scope_change
// naming that path (the dense combined diff of the merge, --cc), and the
// path outside that the merge only brings in from upstream stays out.
func TestWatch_ReservedScope_AConflictResolvedOutsideInAMergeRaisesScopeChange(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	scopeWrite(t, repo, "other/x.go", "own")
	scopeCommitAll(t, repo, "own work, one path outside")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t) // the baseline: a clean worktree, with the own commit before it

	scopeAdvanceOrigin(t, repo, "other/x.go", "other/new.go")
	if out, err := exec.Command("git", "-C", repo, "merge", "--quiet", "--no-edit", "refs/remotes/origin/main").CombinedOutput(); err == nil {
		t.Fatalf("the merge of origin's default branch did not conflict:\n%s", out)
	}
	scopeWrite(t, repo, "other/x.go", "resolved")
	scopeGit(t, repo, "add", "other/x.go")
	scopeGit(t, repo, "commit", "--quiet", "--no-edit")
	if dirty := gitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("the worktree is not clean after the merge:\n%s", dirty)
	}
	if parents := strings.Fields(gitOut(t, repo, "rev-list", "--parents", "-n", "1", "HEAD")); len(parents) != 3 {
		t.Fatalf("HEAD and its parents = %q; want a merge commit", parents)
	}
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/x.go")
}

// A case the cycle 1 cross-review of plan 2026-11-12-watch-alert-dedupe
// found: an origin/HEAD that points to a ref that is gone
// (a default branch renamed upstream and its old remote-tracking branch
// pruned) is passed over. With origin/main, upstream's commits a merge
// brings in stay out and the org's own commit outside raises scope_change;
// with neither main nor master, the paths come from `git diff` of the two
// HEADs, so a commit outside raises it too. Before, every `git log` with
// the gone ref failed, and no commit was ever reported.
func TestWatch_ReservedScope_ADanglingOriginHEADFallsBackToMainThenToTheDiff(t *testing.T) {
	t.Run("with origin/main", func(t *testing.T) {
		repo := scopeGitRepo(t, "main")
		scopeGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
		f := newReservedScopeFixture(t, repo, "src/")
		f.cycle(t) // the baseline
		scopeAdvanceOrigin(t, repo, "other/new.go")
		scopeGit(t, repo, "merge", "--quiet", "--no-edit", "refs/remotes/origin/main")
		scopeWrite(t, repo, "other/y.go", "own")
		scopeCommitAll(t, repo, "own work outside")
		f.cycle(t)
		f.wantOneScopeAlert(t, "other/y.go")
		if head, snap := gitOut(t, repo, "rev-parse", "HEAD"), f.status.SeatSnapshots[LeaderIdentity]; snap.ScopeHead != head {
			t.Errorf("ScopeHead = %q; want it moved on to HEAD %s", snap.ScopeHead, head)
		}
	})
	t.Run("with neither main nor master", func(t *testing.T) {
		repo := scopeGitRepo(t, "")
		scopeGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/gone")
		f := newReservedScopeFixture(t, repo, "src/")
		f.cycle(t) // the baseline
		scopeWrite(t, repo, "other/x.go", "edited")
		scopeCommitAll(t, repo, "outside")
		f.cycle(t)
		f.wantOneScopeAlert(t, "other/x.go")
	})
}

// D6: with no ref of origin's default branch, the paths of the commits
// since the previous cycle come from `git diff` of the two HEADs, so a
// commit outside the reservation still raises scope_change.
func TestWatch_ReservedScope_WithNoOriginRefACommitOutsideRaisesScopeChange(t *testing.T) {
	repo := scopeGitRepo(t, "")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t)
	scopeWrite(t, repo, "other/x.go", "edited")
	scopeWrite(t, repo, "src/keep.go", "edited")
	scopeCommitAll(t, repo, "outside and inside")
	if dirty := gitOut(t, repo, "status", "--porcelain"); dirty != "" {
		t.Fatalf("the worktree is not clean after the commit:\n%s", dirty)
	}
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/x.go")
}

// D6 and AC8: the commit paths come back root-relative whatever the user's
// git config says. With diff.relative=true and the seat in a subdirectory, a
// git that listed them relative to the cwd would name a path inside the
// reservation without its directory (a false scope_change) and drop a path
// outside the subdirectory (a missed one). Both listings are checked: the log
// of the commits not on origin's default branch, and the diff of the two
// HEADs when there is no origin ref. The leader's own new plan, whose base is
// read with the cwd in the subdirectory too, still raises nothing when origin
// has a ref.
func TestWatch_ReservedScope_CommitPathsStayRootRelativeUnderDiffRelative(t *testing.T) {
	for _, tt := range []struct {
		name   string
		origin string
		plan   bool // the base is known, so the org's own new plan is dropped
	}{
		{"the log of the commits not on origin's default branch", "main", true},
		{"the diff of the two HEADs with no origin ref", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo := scopeGitRepo(t, tt.origin)
			scopeGit(t, repo, "config", "diff.relative", "true")
			scopeWrite(t, repo, "internal/org/x.go", "base")
			scopeWrite(t, repo, "cmd/y.go", "base")
			scopeCommitAll(t, repo, "the org's own files")
			f := newReservedScopeFixtureAt(t, filepath.Join(repo, "internal", "org"), "internal/org/")
			f.cycle(t) // the baseline

			scopeWrite(t, repo, "internal/org/x.go", "edited")
			scopeCommitAll(t, repo, "inside the reservation")
			f.cycle(t)
			if alerts := f.scopeAlerts(); len(alerts) != 0 {
				t.Fatalf("scope_change ALERTs after a commit inside the reservation = %q; want none", alerts)
			}

			if tt.plan {
				scopeWrite(t, repo, "docs/plans/active/2026-11-12-feat.md", "plan")
				f.cycle(t)
				if alerts := f.scopeAlerts(); len(alerts) != 0 {
					t.Fatalf("scope_change ALERTs after the org's own new plan = %q; want none", alerts)
				}
			}

			scopeWrite(t, repo, "cmd/y.go", "edited")
			scopeCommitAll(t, repo, "outside the reservation")
			f.cycle(t)
			f.wantOneScopeAlert(t, "cmd/y.go")
		})
	}
}

// A worktree with no commit yet has no HEAD: the porcelain is still
// compared, and no HEAD is recorded.
func TestWatch_ReservedScope_AWorktreeWithNoCommitComparesItsChanges(t *testing.T) {
	repo := scopeGitRepo(t, "unborn")
	scopeWrite(t, repo, "other/a.go", "new")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t)
	scopeWrite(t, repo, "src/b.go", "new")
	f.cycle(t)
	if alerts := f.scopeAlerts(); len(alerts) != 0 {
		t.Fatalf("scope_change ALERTs after a change inside = %q; want none", alerts)
	}
	scopeWrite(t, repo, "other/c.go", "new")
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/a.go", "other/c.go")
	if snap := f.status.SeatSnapshots[LeaderIdentity]; snap.ScopeHead != "" {
		t.Errorf("ScopeHead = %q; want none for a repository with no commit", snap.ScopeHead)
	}
}

// A previous HEAD that git no longer has (a rebase and a gc, or a status
// file of another repository) gives no commit paths and writes nothing on
// stderr; no later cycle could list from it (scopeHeadKnown), so HEAD moves
// on, and the next commit is compared with it.
func TestWatch_ReservedScope_APreviousHeadGitDoesNotHaveIsSkippedQuietly(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	f := newReservedScopeFixture(t, repo, "src/")
	f.cycle(t)
	snap := f.status.SeatSnapshots[LeaderIdentity]
	snap.ScopeHead = strings.Repeat("0", 39) + "1"

	scopeWrite(t, repo, "other/x.go", "edited")
	scopeCommitAll(t, repo, "outside")
	f.cycle(t)
	if alerts := f.scopeAlerts(); len(alerts) != 0 {
		t.Fatalf("scope_change ALERTs = %q; want none when the previous HEAD cannot be read", alerts)
	}
	if head := gitOut(t, repo, "rev-parse", "HEAD"); snap.ScopeHead != head {
		t.Errorf("ScopeHead = %q; want it moved on to HEAD %s", snap.ScopeHead, head)
	}
	if f.stderr.Len() != 0 {
		t.Errorf("stderr = %q; want nothing for a diff that failed", f.stderr.String())
	}

	scopeWrite(t, repo, "other/y.go", "edited")
	scopeCommitAll(t, repo, "outside again")
	f.cycle(t)
	f.wantOneScopeAlert(t, "other/y.go")
}

// L1 of the self-review: a listing of the commits since the previous cycle
// that fails while git still has the previous HEAD (a git killed at the
// deadline) keeps that HEAD, so the next cycle lists the same commits and
// reports the path outside the reservation one of them changed: the log
// with origin's default branch, and the diff without it.
func TestWatch_ReservedScope_AFailedCommitListingKeepsThePreviousHeadForTheNextCycle(t *testing.T) {
	for _, tc := range []struct{ name, origin, listing string }{
		{"the log", "main", "log"},
		{"the diff with no origin ref", "", "diff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := scopeGitRepo(t, tc.origin)
			f := newReservedScopeFixture(t, repo, "src/")
			failListing := false
			var failed []string
			f.run.gitRun = func(cwd string, args ...string) (string, error) {
				if failListing && scopeSubcommand(args) == tc.listing {
					failed = append(failed, strings.Join(args, " "))
					return "", errors.New("signal: killed")
				}
				return f.o.realGitRun(cwd, args...)
			}
			f.cycle(t) // the baseline
			prev := gitOut(t, repo, "rev-parse", "HEAD")

			scopeWrite(t, repo, "other/x.go", "edited")
			scopeCommitAll(t, repo, "outside")
			failListing = true
			f.cycle(t)
			failListing = false
			if len(failed) != 1 {
				t.Fatalf("the failed %s calls = %q; want one", tc.listing, failed)
			}
			if alerts := f.scopeAlerts(); len(alerts) != 0 {
				t.Fatalf("scope_change ALERTs while the listing failed = %q; want none (the worktree is clean)", alerts)
			}
			if snap := f.status.SeatSnapshots[LeaderIdentity]; snap.ScopeHead != prev {
				t.Fatalf("ScopeHead = %q; want the previous HEAD %s kept", snap.ScopeHead, prev)
			}

			f.cycle(t)
			f.wantOneScopeAlert(t, "other/x.go")
			if head, snap := gitOut(t, repo, "rev-parse", "HEAD"), f.status.SeatSnapshots[LeaderIdentity]; snap.ScopeHead != head {
				t.Errorf("ScopeHead = %q; want it moved on to HEAD %s", snap.ScopeHead, head)
			}
		})
	}
}

// L1 of the self-review: after a failed listing of the commits since the
// previous cycle, the watch asks git whether it still has the previous HEAD
// (`rev-parse --verify --quiet <prev>^{commit}`), keeps that HEAD when git
// has it, and moves on to the current HEAD when git does not.
func TestWatch_ReservedScope_AFailedCommitListingAsksWhetherGitHasThePreviousHead(t *testing.T) {
	for _, tc := range []struct {
		name  string
		known bool
	}{
		{"git has it", true},
		{"git no longer has it", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, _, _, clk := testWatchOrg(t)
			p := watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)
			p.Cwd, p.Reserve = "/wt", []string{"src/"}
			if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn: %+v", r)
			}
			prev, head := strings.Repeat("a", 40), strings.Repeat("b", 40)
			g := &scopeStubGit{answers: map[string]func([]string) (string, error){
				"status": func([]string) (string, error) { return "", nil },
				"rev-parse": func(args []string) (string, error) {
					switch last := args[len(args)-1]; {
					case last == "HEAD":
						return head + "\n", nil
					case last == prev+"^{commit}" && tc.known:
						return prev + "\n", nil
					}
					return "", errors.New("exit status 1")
				},
				// No answer for diff: the listing fails.
			}}
			dir := t.TempDir()
			run := &watchRun{
				org: o, cfg: o.Config, gitRun: g.run,
				escalateFn: func(context.Context, string) error { return nil }, stderr: &bytes.Buffer{},
				statusPath: filepath.Join(dir, WatchStatusFileName("org-a")), escalationsPath: filepath.Join(dir, EscalationsRelName),
			}
			status := &watchStatusFile{SeatSnapshots: map[string]*watchSeatSnapshot{
				LeaderIdentity: {ScopeHead: prev, ScopeOutsidePathsSeen: true},
			}}
			clk.Advance(time.Minute)
			if err := run.evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}

			if got, want := g.commands(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "rev-parse", "diff", "rev-parse"}; !slices.Equal(got, want) {
				t.Fatalf("the cycle runs %q; want %q", got, want)
			}
			if want := "/wt: rev-parse --verify --quiet " + prev + "^{commit}"; g.calls[len(g.calls)-1] != want {
				t.Errorf("the last call = %q; want %q", g.calls[len(g.calls)-1], want)
			}
			want := head
			if tc.known {
				want = prev
			}
			if got := status.SeatSnapshots[LeaderIdentity].ScopeHead; got != want {
				t.Errorf("ScopeHead = %q; want %q", got, want)
			}
		})
	}
}

// L2 of the self-review: scope_change rises only for a path outside the
// reservation that the previous cycle did not have, or for a commit since
// the previous cycle that changed one. A change put back while another
// stays (the set shrinks), and a `git add` of a path already outside (only
// its porcelain status changes), do not raise it. Each step after a raise
// follows a quiet cycle, so a rise there would be a new rising edge.
func TestWatch_ReservedScope_OnlyANewPathOrACommitOutsideRaisesScopeChange(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	f := newReservedScopeFixture(t, repo, "src/")
	key := conditionKey("org-a", LeaderIdentity, condScopeChange)
	f.cycle(t) // the baseline
	steps := []struct {
		name   string
		do     func()
		active bool
	}{
		{"two edits outside", func() {
			scopeWrite(t, repo, "other/x.go", "edited")
			scopeWrite(t, repo, "other/y.go", "edited")
		}, true},
		{"nothing new", func() {}, false},
		{"one of them put back", func() { scopeGit(t, repo, "checkout", "--", "other/x.go") }, false},
		{"the other staged", func() { scopeGit(t, repo, "add", "other/y.go") }, false},
		{"a new path outside", func() { scopeWrite(t, repo, "other/z.go", "new") }, true},
		{"nothing new again", func() {}, false},
		{"the paths outside committed", func() { scopeCommitAll(t, repo, "outside") }, true},
		{"a clean worktree", func() {}, false},
	}
	for _, s := range steps {
		s.do()
		f.cycle(t)
		rec := f.status.Conditions[key]
		if got := rec != nil && rec.Active; got != s.active {
			t.Fatalf("after %s: scope_change active = %t; want %t (the leader's snapshot %+v)", s.name, got, s.active, f.status.SeatSnapshots[LeaderIdentity])
		}
	}

	alerts := f.scopeAlerts()
	want := [][]string{{"other/x.go", "other/y.go"}, {"other/y.go", "other/z.go"}, {"other/y.go", "other/z.go"}}
	if len(alerts) != len(want) {
		t.Fatalf("scope_change ALERTs = %q; want %d, one for each rise", alerts, len(want))
	}
	for i, message := range alerts {
		if got := outsidePaths(t, message); !slices.Equal(got, want[i]) {
			t.Errorf("scope_change ALERT %d names %q; want %q", i+1, got, want[i])
		}
	}
}

// AC12: a status file written before the reservation baseline existed
// loads, and its first cycle records the baseline only, even with a change
// outside the reservation in the worktree. So does one that holds the
// earlier per-entry baseline (porcelain entries under scope_outside with
// scope_seen): its entry for the same change is not read as a baseline of
// paths, so the change is not a new path.
func TestWatch_ReservedScope_AStatusFileWithoutTheScopeBaselineLoads(t *testing.T) {
	for _, tc := range []struct {
		name     string
		snapshot func(head string) string
	}{
		{"written before the reservation baseline", func(string) string {
			return `{"agent_get": "ok", "agent_get_seen": true, "git_status": " M other/x.go\n", "git_status_seen": true}`
		}},
		{"holding the earlier per-entry baseline", func(head string) string {
			return fmt.Sprintf(`{"agent_get": "ok", "agent_get_seen": true, "scope_head": %q, "scope_outside": [" M other/x.go"], "scope_seen": true}`, head)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := scopeGitRepo(t, "main")
			f := newReservedScopeFixture(t, repo, "src/")
			old := `{"org_id": "org-a", "last_cycle_ts": "2025-12-31T23:59:00Z", "cycles": 7,
  "seat_snapshots": {"leader": ` + tc.snapshot(gitOut(t, repo, "rev-parse", "HEAD")) + `}}`
			if err := os.WriteFile(f.run.statusPath, []byte(old), 0o644); err != nil {
				t.Fatal(err)
			}
			status, err := loadWatchStatus(f.run.statusPath, "org-a")
			if err != nil {
				t.Fatalf("loadWatchStatus of a status file with no scope baseline: %v", err)
			}
			f.status = status

			scopeWrite(t, repo, "other/x.go", "edited")
			f.cycle(t)
			if alerts := f.scopeAlerts(); len(alerts) != 0 {
				t.Fatalf("scope_change ALERTs on the first cycle = %q; want the baseline only", alerts)
			}
			if snap := f.status.SeatSnapshots[LeaderIdentity]; !snap.ScopeOutsidePathsSeen || !slices.Equal(snap.ScopeOutsidePaths, []string{"other/x.go"}) {
				t.Fatalf("the leader's snapshot = %+v; want the baseline with other/x.go outside", snap)
			}
			saved := readWatchStatusFile(t, f.dir, "org-a")
			if snap := saved.SeatSnapshots[LeaderIdentity]; snap == nil || !snap.ScopeOutsidePathsSeen || snap.ScopeHead == "" {
				t.Fatalf("the saved snapshot = %+v; want the scope baseline written", snap)
			}

			scopeWrite(t, repo, "other/y.go", "edited")
			f.cycle(t)
			f.wantOneScopeAlert(t, "other/x.go", "other/y.go")
		})
	}
}

// A scope_change that fires again while its inbox item is open records the
// paths outside the reservation it found in the note of its recurred event
// (raiseOrClear's recurNote).
func TestWatch_ReservedScope_ARecurrenceNotesThePathsOutside(t *testing.T) {
	repo := scopeGitRepo(t, "main")
	f := newReservedScopeFixture(t, repo, "src/")
	inbox := NewInboxStore(f.dir)
	inbox.Now = f.clk.Now
	f.o.Inbox = inbox
	f.o.EscalationsPath = EscalationsPathIn(f.dir)
	f.o.DesktopNotify = func(context.Context, string) error { return nil }

	f.cycle(t)
	scopeWrite(t, repo, "other/x.go", "edited")
	f.cycle(t) // the ALERT, which is about the leader, so it is an inbox item at once
	f.cycle(t) // nothing new: the condition clears
	scopeWrite(t, repo, "other/y.go", "edited")
	f.cycle(t) // fires again while the item is open

	f.wantOneScopeAlert(t, "other/x.go")
	rr, err := inbox.Read()
	if err != nil {
		t.Fatalf("read the inbox: %v", err)
	}
	if len(rr.Items) != 1 {
		t.Fatalf("inbox items = %+v; want the one scope_change item", rr.Items)
	}
	item := rr.Items[0]
	var notes []string
	for _, ev := range item.Events {
		if ev.Event == InboxEventRecurred {
			notes = append(notes, ev.Note)
		}
	}
	if want := []string{"outside reservation: other/x.go, other/y.go"}; item.Recurrences != 1 || !slices.Equal(notes, want) {
		t.Errorf("the item's recurrences = %d with notes %q; want 1 with %q", item.Recurrences, notes, want)
	}
}

// scopeSubcommand returns the git subcommand of args: the first argument
// after any leading `-c <key=value>` pairs, which the commit listing puts
// before it (`-c diff.relative=false log ...`).
func scopeSubcommand(args []string) string {
	i := 0
	for i+2 < len(args) && args[i] == "-c" {
		i += 2
	}
	return args[i]
}

// scopeStubGit is a GitRunFunc that records each call and answers from
// answers by the subcommand (scopeSubcommand); a subcommand with no answer
// fails.
type scopeStubGit struct {
	mu      sync.Mutex
	calls   []string
	answers map[string]func(args []string) (string, error)
}

func (g *scopeStubGit) run(cwd string, args ...string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls = append(g.calls, cwd+": "+strings.Join(args, " "))
	if answer, ok := g.answers[scopeSubcommand(args)]; ok {
		return answer(args)
	}
	return "", errors.New("exit status 128")
}

// commands returns the subcommand of each call.
func (g *scopeStubGit) commands() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, c := range g.calls {
		_, args, _ := strings.Cut(c, ": ")
		out = append(out, scopeSubcommand(strings.Fields(args)))
	}
	return out
}

// The git calls of a reserved org's cycle: the porcelain and HEAD every
// cycle, the commit paths only when HEAD moved from a previous HEAD that is
// a full object name (the log of the commits not on origin's default
// branch, or the diff when there is none), the merge base only when a path
// is a candidate record, and origin's default branch at most once, only
// when one of those two needs it, with origin/HEAD's target used only when
// it names a commit.
func TestWatch_ReservedScope_RunsOnlyTheGitCallsItNeeds(t *testing.T) {
	o, _, _, clk := testWatchOrg(t)
	p := watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)
	p.Cwd, p.Reserve = "/wt", []string{"src/"}
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn: %+v", r)
	}
	porcelain := "?? other/a.go\x00"
	head := strings.Repeat("a", 40)
	base := strings.Repeat("c", 40)
	originMain := false
	// originHEAD is where refs/remotes/origin/HEAD points ("" when it is
	// not a symbolic ref), and trunkExists whether that target names a
	// commit.
	originHEAD, trunkExists := "", false
	g := &scopeStubGit{answers: map[string]func([]string) (string, error){
		"status": func([]string) (string, error) { return porcelain, nil },
		"rev-parse": func(args []string) (string, error) {
			switch last := args[len(args)-1]; {
			case last == "HEAD":
				return head + "\n", nil
			case last == "refs/remotes/origin/main^{commit}" && originMain:
				return base + "\n", nil
			case last == "refs/remotes/origin/trunk^{commit}" && trunkExists:
				return base + "\n", nil
			}
			return "", errors.New("exit status 1")
		},
		"symbolic-ref": func([]string) (string, error) {
			if originHEAD == "" {
				return "", errors.New("exit status 1")
			}
			return originHEAD + "\n", nil
		},
		"diff":       func([]string) (string, error) { return "", nil },
		"log":        func([]string) (string, error) { return "", nil },
		"merge-base": func([]string) (string, error) { return base + "\n", nil },
		"ls-tree":    func([]string) (string, error) { return "", nil },
	}}
	dir := t.TempDir()
	run := &watchRun{
		org: o, cfg: o.Config, gitRun: g.run,
		escalateFn: func(context.Context, string) error { return nil }, stderr: &bytes.Buffer{},
		statusPath: filepath.Join(dir, WatchStatusFileName("org-a")), escalationsPath: filepath.Join(dir, EscalationsRelName),
	}
	status := &watchStatusFile{}
	cycle := func() []string {
		t.Helper()
		g.mu.Lock()
		g.calls = nil
		g.mu.Unlock()
		clk.Advance(time.Minute)
		if err := run.evaluateCycle("org-a", status); err != nil {
			t.Fatalf("evaluateCycle: %v", err)
		}
		return g.commands()
	}

	if got, want := cycle(), []string{"status", "rev-parse"}; !slices.Equal(got, want) {
		t.Errorf("the first cycle runs %q; want %q", got, want)
	}
	if got, want := cycle(), []string{"status", "rev-parse"}; !slices.Equal(got, want) {
		t.Errorf("a cycle with HEAD unchanged runs %q; want %q", got, want)
	}
	if want := "/wt: status --porcelain=v1 -z --untracked-files=all"; g.calls[0] != want {
		t.Errorf("the porcelain call = %q; want %q", g.calls[0], want)
	}

	// HEAD moved and there is no origin ref: the default branch (symbolic-ref,
	// then main, then master), then the diff of the two HEADs.
	head = strings.Repeat("b", 40)
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "rev-parse", "diff"}; !slices.Equal(got, want) {
		t.Errorf("a cycle after HEAD moved with no origin ref runs %q; want %q", got, want)
	}
	if want := "/wt: -c diff.relative=false diff --no-renames --name-only -z " + strings.Repeat("a", 40) + " " + head; len(g.calls) != 6 || g.calls[5] != want {
		t.Errorf("the calls = %q; want the diff call %q last", g.calls, want)
	}

	// A previous HEAD that is not a full object name never reaches git.
	status.SeatSnapshots[LeaderIdentity].ScopeHead = "--output=/tmp/x"
	if got, want := cycle(), []string{"status", "rev-parse"}; !slices.Equal(got, want) {
		t.Errorf("a cycle after a previous HEAD that is not an object name runs %q; want %q", got, want)
	}

	// A candidate record: the default branch (symbolic-ref, then main, then
	// master), the merge base, and the listing of the base.
	porcelain = "?? docs/reports/review-x.md\x00"
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "rev-parse"}; !slices.Equal(got, want) {
		t.Errorf("a cycle with a candidate record and no origin ref runs %q; want %q", got, want)
	}
	originMain = true
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "merge-base", "ls-tree"}; !slices.Equal(got, want) {
		t.Errorf("a cycle with a candidate record and origin/main runs %q; want %q", got, want)
	}
	if want := []string{
		"/wt: merge-base HEAD refs/remotes/origin/main",
		"/wt: ls-tree --full-tree --name-only -z " + base + " -- docs/reports/review-x.md",
	}; len(g.calls) != 6 || !slices.Equal(g.calls[4:], want) {
		t.Errorf("the calls = %q; want the merge base and the listing %q", g.calls, want)
	}

	// HEAD moved with origin/main and a candidate record: the default branch
	// is resolved once, for the log of the commits not on it and for the
	// merge base.
	prev := head
	head = strings.Repeat("d", 40)
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "log", "merge-base", "ls-tree"}; !slices.Equal(got, want) {
		t.Errorf("a cycle after HEAD moved with origin/main and a candidate record runs %q; want %q", got, want)
	}
	if want := "/wt: -c diff.relative=false log --no-show-signature --cc --no-renames --name-only -z --format= " + prev + ".." + head + " --not refs/remotes/origin/main --"; len(g.calls) != 7 || g.calls[4] != want {
		t.Errorf("the calls = %q; want the log call %q fifth", g.calls, want)
	}

	// origin/HEAD points to a ref that names no commit (a default branch
	// renamed upstream and its old ref pruned): its target is checked and
	// passed over, and origin/main is used.
	originHEAD = "refs/remotes/origin/trunk"
	prev, head = head, strings.Repeat("e", 40)
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "rev-parse", "log", "merge-base", "ls-tree"}; !slices.Equal(got, want) {
		t.Errorf("a cycle with a dangling origin/HEAD runs %q; want %q", got, want)
	}
	if want := []string{
		"/wt: rev-parse --verify --quiet refs/remotes/origin/trunk^{commit}",
		"/wt: rev-parse --verify --quiet refs/remotes/origin/main^{commit}",
		"/wt: -c diff.relative=false log --no-show-signature --cc --no-renames --name-only -z --format= " + prev + ".." + head + " --not refs/remotes/origin/main --",
	}; len(g.calls) != 8 || !slices.Equal(g.calls[3:6], want) {
		t.Errorf("the calls = %q; want the check of origin/HEAD's target, then origin/main's, then the log %q", g.calls, want)
	}

	// origin/HEAD points to a ref that names a commit: that ref is used, and
	// main and master are not asked about.
	trunkExists = true
	prev, head = head, strings.Repeat("f", 40)
	if got, want := cycle(), []string{"status", "rev-parse", "symbolic-ref", "rev-parse", "log", "merge-base", "ls-tree"}; !slices.Equal(got, want) {
		t.Errorf("a cycle with origin/HEAD pointing at a commit runs %q; want %q", got, want)
	}
	if want := "/wt: -c diff.relative=false log --no-show-signature --cc --no-renames --name-only -z --format= " + prev + ".." + head + " --not refs/remotes/origin/trunk --"; len(g.calls) != 7 || g.calls[4] != want {
		t.Errorf("the calls = %q; want the log call %q fifth", g.calls, want)
	}
}

// AC10 and the GitRun seam: RunWatchAll reads a reserved org's worktree
// through WatchAllParams.GitRun only, and an org without a reservation
// through GitStatus only, with today's raw porcelain in its ALERT.
func TestWatch_ReservedScope_RunWatchAllReadsGitRunAndAnOrgWithoutOneKeepsGitStatus(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-1")
	lp := watchSpawnParams("org-b", LeaderIdentity, LeaderIdentity)
	lp.Cwd, lp.Reserve = "/wt-b", []string{"src/"}
	if r := f.o.Spawn(lp); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn org-b's leader: %+v", r)
	}
	statusCalls := 0
	g := &scopeStubGit{answers: map[string]func([]string) (string, error){
		"status": func([]string) (string, error) {
			statusCalls++
			return fmt.Sprintf("?? other/n%d.go\x00?? src/in%d.go\x00", statusCalls, statusCalls), nil
		},
	}}
	p := f.params(2)
	p.GitRun = g.run
	if err := runWatchAll(t, f.o, context.Background(), p, WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	f.mu.Lock()
	gitStatusCalls := maps.Clone(f.gitCalls)
	f.mu.Unlock()
	if want := map[string]int{"/tmp/watch-seat-seat-1": 2}; !maps.Equal(gitStatusCalls, want) {
		t.Errorf("GitStatus calls = %v; want only org-a's worktree, once a cycle", gitStatusCalls)
	}
	g.mu.Lock()
	runCalls := slices.Clone(g.calls)
	g.mu.Unlock()
	if len(runCalls) == 0 {
		t.Fatal("GitRun was never called for org-b")
	}
	for _, c := range runCalls {
		if !strings.HasPrefix(c, "/wt-b: ") {
			t.Errorf("GitRun call %q; want only org-b's worktree", c)
		}
	}

	var aAlerts, bAlerts []string
	for _, c := range watchdogAlerts(f.a) {
		if !strings.Contains(c.message, "CONDITION: "+condScopeChange) {
			continue
		}
		switch c.team {
		case agmsgTeam("org-a"):
			aAlerts = append(aAlerts, c.message)
		case agmsgTeam("org-b"):
			bAlerts = append(bAlerts, c.message)
		}
	}
	if len(aAlerts) != 1 || !strings.Contains(aAlerts[0], "seat seat-1 worktree scope changed:\n M file-2") {
		t.Errorf("org-a's scope_change ALERTs = %q; want one with the raw porcelain", aAlerts)
	}
	if len(bAlerts) != 1 {
		t.Fatalf("org-b's scope_change ALERTs = %q; want one", bAlerts)
	}
	if got := outsidePaths(t, bAlerts[0]); !slices.Equal(got, []string{"other/n2.go"}) {
		t.Errorf("org-b's ALERT names %q; want only the path outside the reservation", got)
	}
}

// promptDocsDirsNotWritten are the docs/ directories a role prompt names
// without having a seat write there, by prompt file, each with the reason.
// A prompt that names any other docs/ directory than docs/plans/ and
// docs/reports/ fails TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite.
var promptDocsDirsNotWritten = map[string]string{
	"leader.md docs/new/":        "an example of git's default output for a new directory, in the sync-docs step",
	"leader.md docs/recipes/":    "an example DOC_FOLLOWUP line of the RESULT: a doc the org leaves for later, not one it writes",
	"reviewer.md docs/evidence/": "the logs the gate scripts write there, which the scaffold's .gitignore ignores (checked below)",
}

// AC11: the rule that drops the org's own records (ownRecordCandidate, D5)
// covers the docs/ paths the role prompts have seats write outside the
// reservation. Every docs/ directory a prompt names is docs/plans/ or
// docs/reports/ (or a subdirectory), or is listed in
// promptDocsDirsNotWritten; the names the prompts give the plan and the
// reports are candidates, and the templates are not.
func TestOwnRecord_CoversTheDocsPathsTheRolePromptsWrite(t *testing.T) {
	files, err := fs.Glob(promptFS, "prompts/*.md")
	if err != nil || len(files) == 0 {
		t.Fatalf("the embedded role prompts: %v, %q", err, files)
	}
	mention := regexp.MustCompile(`docs/([A-Za-z0-9_.-]+)`)
	seen := map[string]bool{}
	for _, file := range files {
		data, err := promptFS.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		name := filepath.Base(file)
		for _, m := range mention.FindAllStringSubmatch(string(data), -1) {
			dir := "docs/" + m[1] + "/"
			if dir == "docs/plans/" || dir == "docs/reports/" {
				continue
			}
			key := name + " " + dir
			seen[key] = true
			if _, ok := promptDocsDirsNotWritten[key]; !ok {
				t.Errorf("%s names %s%s. If seats write there outside the reservation, the watch's scope_change "+
					"raises an ALERT for each such file: revisit D5 of plan 2026-11-12-watch-alert-dedupe "+
					"(ownRecordCandidate in watch_scope.go), or list it in promptDocsDirsNotWritten with the reason",
					file, "docs/", m[1])
			}
		}
	}
	for key := range promptDocsDirsNotWritten {
		if !seen[key] {
			t.Errorf("promptDocsDirsNotWritten lists %q, which no prompt names any more: remove it", key)
		}
	}

	gitignore := readRepoFileOrSkip(t, "templates/base/.gitignore")
	for _, line := range []string{"docs/evidence/*.log", "docs/evidence/*.json"} {
		if !hasLine(gitignore, line) {
			t.Errorf("templates/base/.gitignore has no line %s, so the gate's logs under docs/evidence/ "+
				"would show in git status outside the reservation", line)
		}
	}

	for _, p := range []string{
		"docs/plans/active/2026-01-01-x.md",
		"docs/plans/archive/2026-01-01-x.md",
		reviewerReportPrefix + "2026-01-01-org-seat.md",
		"docs/reports/review-xreview-2026-01-01-org-c1.md",
		"docs/reports/sync-docs-2026-01-01-x.md",
		"docs/reports/verify-x.md",
		defaultReportDir + "/" + orgReportFilePrefix + "org-2026-01-01.md",
	} {
		if !ownRecordCandidate(p) {
			t.Errorf("ownRecordCandidate(%q) = false; a record the prompts have seats write must be a candidate", p)
		}
	}
	for _, p := range []string{"docs/plans/templates/feature-plan.md", "docs/reports/templates/x.md"} {
		if ownRecordCandidate(p) {
			t.Errorf("ownRecordCandidate(%q) = true; a template is an input the skills read, never the org's record", p)
		}
	}
}
