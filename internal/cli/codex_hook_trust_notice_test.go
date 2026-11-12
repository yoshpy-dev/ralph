package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for printCodexHookTrustNotice, the line `ralph org spawn`, both forms
// of `ralph org start`, and `ralph org director start` print on stderr when
// a codex agent would run without some of the hooks of the .codex/hooks.json
// codex reads (plan 2026-11-12-codex-hook-trust-check, AC6, D5, and D6;
// issue #310). Each points doctorCodexSandboxEnv at a codex home of its own,
// so none reads the real ~/.codex, and each runs the stub herdr and agmsg
// (setupOrgStubPATH). None runs in parallel: they chdir, set environment
// variables, and swap a package variable.

// setCodexHookTrustNoticeHome points doctorCodexSandboxEnv at a codex home
// under a temp dir until the test ends and returns the config.toml path,
// which is not written yet. The line names it as ~/.codex/config.toml.
func setCodexHookTrustNoticeHome(t *testing.T) (configPath string) {
	t.Helper()
	home := codexHookTrustTempDir(t)
	configPath = filepath.Join(home, ".codex", "config.toml")
	orig := doctorCodexSandboxEnv
	doctorCodexSandboxEnv = func() (codexSandboxEnv, error) {
		return codexSandboxEnv{Home: home, ConfigPath: configPath}, nil
	}
	t.Cleanup(func() { doctorCodexSandboxEnv = orig })
	return configPath
}

// initCodexHookTrustNoticeRepo makes dir a git repository (git init, with the
// developer's git config kept out) and writes hooks as its .codex/hooks.json
// unless hooks is "". It returns the hooks.json path.
func initCodexHookTrustNoticeRepo(t *testing.T, dir, hooks string) (hooksPath string) {
	t.Helper()
	isolateFeatureGitEnv(t)
	runLedgerTestGit(t, "init", "--quiet", dir)
	hooksPath = filepath.Join(dir, ".codex", "hooks.json")
	if hooks != "" {
		writeCodexHookTrustFile(t, hooksPath, hooks)
	}
	return hooksPath
}

// codexHookTrustNoticeUnapproved is the line for the two hooks of
// codexHookTrustDoctorHooks at hooksPath when no codex user config exists.
func codexHookTrustNoticeUnapproved(verb, seat, hooksPath string) string {
	return "note: " + verb + ": codex seat " + seat + ": there is no codex user config at ~/.codex/config.toml to approve the 2 hooks in " +
		hooksPath + ", so they do not run in the seat; run 'ralph doctor' for the keys and how to trust them\n"
}

// TestCodexHookTrustNotice_SpawnPrintsOneLineForACodexSeat is AC6 for
// `ralph org spawn --dry-run` with the seat's cwd a git repository holding
// codexHookTrustDoctorHooks: a codex seat gets one line on stderr when a hook
// is not approved or is turned off, with the counts and the seat, and the
// spawn goes on with exit 0; no line when every hook is approved, when the
// seat runs on claude, when the cwd has no hooks.json, or when codex's user
// config cannot be read or decoded (doctor reports those).
func TestCodexHookTrustNotice_SpawnPrintsOneLineForACodexSeat(t *testing.T) {
	const pre, start = ":pre_tool_use:0:0", ":session_start:0:0"
	cases := []struct {
		name     string
		driver   []string                      // --driver and --model
		hooks    string                        // the cwd's hooks.json; "" writes none
		config   func(hooksPath string) string // codex's user config; nil writes none, "<dir>" makes it a directory
		relative bool                          // pass --cwd "." from inside the repository
		want     func(hooksPath string) string // the whole of stderr
	}{
		{name: "codex, no codex user config", hooks: codexHookTrustDoctorHooks,
			want: func(h string) string { return codexHookTrustNoticeUnapproved("org spawn", "implementer", h) }},
		{name: "codex, a config with no approvals", hooks: codexHookTrustDoctorHooks,
			config: func(string) string { return "model = \"gpt-6.1-sol\"\n" },
			want: func(h string) string {
				return "note: org spawn: codex seat implementer: 2 of 2 hooks in " + h + " are not approved in ~/.codex/config.toml, " +
					"so they do not run in the seat; run 'ralph doctor' for the keys and how to trust them\n"
			}},
		{name: "codex, one hook approved", hooks: codexHookTrustDoctorHooks,
			config: func(h string) string { return codexHookTrustApprovals(h + pre) },
			want: func(h string) string {
				return "note: org spawn: codex seat implementer: 1 of 2 hooks in " + h + " is not approved in ~/.codex/config.toml, " +
					"so it does not run in the seat; run 'ralph doctor' for the key and how to trust it\n"
			}},
		{name: "codex, a relative --cwd", hooks: codexHookTrustDoctorHooks, relative: true,
			want: func(h string) string { return codexHookTrustNoticeUnapproved("org spawn", "implementer", h) }},
		{name: "codex, one hook turned off", hooks: codexHookTrustDoctorHooks,
			config: func(h string) string {
				return codexHookTrustApprovals(h+pre) + codexHookTrustStateTable(h+start, "trusted_hash = \"sha256:b\"\nenabled = false")
			},
			want: func(h string) string {
				return "note: org spawn: codex seat implementer: 1 of 2 hooks in " + h + " is turned off (enabled = false) in ~/.codex/config.toml, " +
					"so it does not run in the seat; run 'ralph doctor' for the key and how to re-enable it\n"
			}},
		{name: "codex, one not approved and one turned off", hooks: codexHookTrustDoctorHooks,
			config: func(h string) string { return codexHookTrustStateTable(h+start, "enabled = false") },
			want: func(h string) string {
				return "note: org spawn: codex seat implementer: 1 of 2 hooks in " + h + " is not approved in ~/.codex/config.toml, 1 turned off (enabled = false), " +
					"so they do not run in the seat; run 'ralph doctor' for the keys and how to trust or re-enable them\n"
			}},
		{name: "codex, every hook approved", hooks: codexHookTrustDoctorHooks,
			config: func(h string) string { return codexHookTrustApprovals(h+pre, h+start) },
			want:   func(string) string { return "" }},
		{name: "claude", driver: []string{"--driver", "claude", "--model", "opus"}, hooks: codexHookTrustDoctorHooks,
			want: func(string) string { return "" }},
		{name: "codex, no hooks.json in the cwd",
			want: func(string) string { return "" }},
		{name: "codex, an unreadable config", hooks: codexHookTrustDoctorHooks,
			config: func(string) string { return "<dir>" },
			want:   func(string) string { return "" }},
		{name: "codex, an undecodable config", hooks: codexHookTrustDoctorHooks,
			config: func(string) string { return "[hooks\n" },
			want:   func(string) string { return "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			configPath := setCodexHookTrustNoticeHome(t)
			repo := filepath.Join(codexHookTrustTempDir(t), "repo")
			hooksPath := initCodexHookTrustNoticeRepo(t, repo, tc.hooks)
			if tc.config != nil {
				if content := tc.config(hooksPath); content == "<dir>" {
					mkdirCodexHookTrust(t, configPath)
				} else {
					writeCodexHookTrustFile(t, configPath, content)
				}
			}
			cwd := repo
			if tc.relative {
				t.Chdir(repo)
				cwd = "."
			} else {
				t.Chdir(t.TempDir())
			}
			driver := tc.driver
			if driver == nil {
				driver = []string{"--driver", "codex", "--model", "gpt-6.1-sol"}
			}
			stateDir := filepath.Join(t.TempDir(), "state")
			args := append([]string{"spawn", "--org-id", "org-a", "--id", "implementer", "--role", "implementer", "--cwd", cwd,
				"--scope", "test-scope", "--state-dir", stateDir, "--dry-run",
				"--config", roleModelsConfig(t, "[org.permissions]\ndefault = \"guarded\"\n")}, driver...)

			stdout, stderr, err := runOrgCmdStreams(t, args...)

			if err != nil {
				t.Fatalf("spawn: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if want := tc.want(hooksPath); stderr != want {
				t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
			}
			if !strings.HasPrefix(stdout, `spawned seat "implementer"`) {
				t.Errorf("stdout = %q; want the spawned line", stdout)
			}
			if last := lastSeatEvent(t, stateDir, "org-a", "implementer"); last.Event != org.EventSpawned || !last.DryRun {
				t.Errorf("last event = %+v; want a dry-run spawned", last)
			}
		})
	}
}

// TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings is AC9 on
// the launch side: a codex seat whose --cwd spells the main checkout through
// a symlink, absolutely or as "." with $PWD spelled through it, gets the line
// unless the hooks are approved under both the symlink's spelling and the
// real path. With only one of them approved the line counts every hook, in
// either direction; were the cwd resolved before the check, approvals under
// the real path alone would hide the missing ones under the symlink's.
func TestCodexHookTrustNotice_SpawnThroughASymlinkNeedsBothSpellings(t *testing.T) {
	const pre, start = ":pre_tool_use:0:0", ":session_start:0:0"
	cases := []struct {
		name     string
		keys     func(realHooks, linkHooks string) []string // the approved keys
		relative bool                                       // pass --cwd "." from inside the symlink
		wantLine bool
	}{
		{"real only", func(r, l string) []string { return []string{r + pre, r + start} }, false, true},
		{"link only", func(r, l string) []string { return []string{l + pre, l + start} }, false, true},
		{"both", func(r, l string) []string { return []string{r + pre, r + start, l + pre, l + start} }, false, false},
		{"relative, real only", func(r, l string) []string { return []string{r + pre, r + start} }, true, true},
		{"relative, link only", func(r, l string) []string { return []string{l + pre, l + start} }, true, true},
		{"relative, both", func(r, l string) []string { return []string{r + pre, r + start, l + pre, l + start} }, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			configPath := setCodexHookTrustNoticeHome(t)
			tmp := codexHookTrustTempDir(t)
			repo := filepath.Join(tmp, "repo")
			realHooks := initCodexHookTrustNoticeRepo(t, repo, codexHookTrustDoctorHooks)
			link := filepath.Join(tmp, "link-repo")
			symlinkCodexHookTrust(t, repo, link)
			linkHooks := filepath.Join(link, ".codex", "hooks.json")
			writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(tc.keys(realHooks, linkHooks)...))
			cwd := link
			if tc.relative {
				chdirCodexHookTrustAs(t, link)
				cwd = "."
			} else {
				t.Chdir(t.TempDir())
			}
			stateDir := filepath.Join(t.TempDir(), "state")

			stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "implementer", "--role", "implementer", "--cwd", cwd,
				"--scope", "test-scope", "--state-dir", stateDir, "--dry-run",
				"--config", roleModelsConfig(t, "[org.permissions]\ndefault = \"guarded\"\n"),
				"--driver", "codex", "--model", "gpt-6.1-sol")

			if err != nil {
				t.Fatalf("spawn: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			want := ""
			if tc.wantLine {
				want = "note: org spawn: codex seat implementer: 2 of 2 hooks in " + linkHooks +
					" are not approved in ~/.codex/config.toml, so they do not run in the seat; run 'ralph doctor' for the keys and how to trust them\n"
			}
			if stderr != want {
				t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
			}
		})
	}
}

// TestCodexHookTrustNotice_SpawnInASubdirectoryCountsItsNestedHooks is AC10
// on the launch side: a codex seat whose --cwd is a subdirectory with its own
// .codex/hooks.json gets the line when that file's hook is not approved,
// although every hook of the root's hooks.json is, and the line names both
// files. With the nested hook approved too, there is no line.
func TestCodexHookTrustNotice_SpawnInASubdirectoryCountsItsNestedHooks(t *testing.T) {
	setupOrgStubPATH(t)
	configPath := setCodexHookTrustNoticeHome(t)
	repo := filepath.Join(codexHookTrustTempDir(t), "repo")
	hooksPath := initCodexHookTrustNoticeRepo(t, repo, codexHookTrustDoctorHooks)
	sub := filepath.Join(repo, "sub")
	subHooks := filepath.Join(sub, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, subHooks, codexHookTrustStopHooks)
	t.Chdir(t.TempDir())
	rootKeys := []string{hooksPath + ":pre_tool_use:0:0", hooksPath + ":session_start:0:0"}
	spawn := func(stateDir string) (string, string) {
		t.Helper()
		stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "implementer", "--role", "implementer", "--cwd", sub,
			"--scope", "test-scope", "--state-dir", stateDir, "--dry-run",
			"--config", roleModelsConfig(t, "[org.permissions]\ndefault = \"guarded\"\n"),
			"--driver", "codex", "--model", "gpt-6.1-sol")
		if err != nil {
			t.Fatalf("spawn: %v (stdout: %s stderr: %s)", err, stdout, stderr)
		}
		return stdout, stderr
	}

	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(rootKeys...))
	stdout, stderr := spawn(filepath.Join(t.TempDir(), "state"))
	want := "note: org spawn: codex seat implementer: 1 of 3 hooks in " + hooksPath + " and " + subHooks +
		" is not approved in ~/.codex/config.toml, so it does not run in the seat; run 'ralph doctor' for the key and how to trust it\n"
	if stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if !strings.HasPrefix(stdout, `spawned seat "implementer"`) {
		t.Errorf("stdout = %q; want the spawned line", stdout)
	}

	writeCodexHookTrustFile(t, configPath, codexHookTrustApprovals(rootKeys[0], rootKeys[1], subHooks+":stop:0:0"))
	if _, stderr := spawn(filepath.Join(t.TempDir(), "state")); stderr != "" {
		t.Errorf("stderr = %q; want nothing with every hook approved", stderr)
	}
}

// TestCodexHookTrustNotice_PrintsNothingForAHooksJSONCodexRefuses is AC11 on
// the launch side, with D5: a hooks.json codex refuses for a null structure,
// the root's or a nested one, gives no line (ralph doctor reports it), even
// when hooks of another file are not approved.
func TestCodexHookTrustNotice_PrintsNothingForAHooksJSONCodexRefuses(t *testing.T) {
	setCodexHookTrustNoticeHome(t)
	repo, _ := newCodexHookTrustNoticeCheckout(t, `{"hooks": null}`)
	var out bytes.Buffer
	printCodexHookTrustNotice(&out, "codex", "org spawn", "implementer", repo)
	if out.Len() != 0 {
		t.Errorf("printed %q for a root hooks.json of null hooks; want nothing", out.String())
	}

	repo, _ = newCodexHookTrustNoticeCheckout(t, codexHookTrustDoctorHooks)
	writeCodexHookTrustFile(t, filepath.Join(repo, "sub", ".codex", "hooks.json"), `{"hooks": {"Stop": [{"hooks": [null]}]}}`)
	printCodexHookTrustNotice(&out, "codex", "org spawn", "implementer", filepath.Join(repo, "sub"))
	if out.Len() != 0 {
		t.Errorf("printed %q with a nested hooks.json of a null handler; want nothing", out.String())
	}
}

// TestCodexHookTrustNotice_StartPrintsOneLineForACodexLeader is AC6 for
// `ralph org start <task>` with --driver codex: the leader's --cwd is checked,
// the line names seat leader once, and the leader starts as before.
func TestCodexHookTrustNotice_StartPrintsOneLineForACodexLeader(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	setCodexHookTrustNoticeHome(t)
	repo := filepath.Join(codexHookTrustTempDir(t), "repo")
	hooksPath := initCodexHookTrustNoticeRepo(t, repo, codexHookTrustDoctorHooks)
	t.Chdir(t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--cwd", repo, "--scope", "org-a", "--state-dir", stateDir,
		"--driver", "codex", "--model", "gpt-6.1-sol", "task text")

	if err != nil {
		t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := codexHookTrustNoticeUnapproved("org start", org.LeaderIdentity, hooksPath); stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if last := lastSeatEvent(t, stateDir, "org-a", org.LeaderIdentity); last.Event != org.EventSpawned || last.Driver != "codex" {
		t.Errorf("leader's last event = %+v; want spawned with codex", last)
	}
	if !strings.Contains(strings.Join(readLogLines(t, herdrLog), "\n"), "agent start ") {
		t.Error("herdr had no agent start; want the leader started")
	}
}

// TestCodexHookTrustNotice_StartWithClaudePrintsNothing is AC6 for `ralph
// org start <task>` with a claude leader: its --cwd holds a hooks.json no
// approval covers, and stderr stays empty. The same repository with a codex
// leader does get the line, so the fixture reaches the check.
func TestCodexHookTrustNotice_StartWithClaudePrintsNothing(t *testing.T) {
	setupOrgStubPATH(t)
	setCodexHookTrustNoticeHome(t)
	repo := filepath.Join(codexHookTrustTempDir(t), "repo")
	initCodexHookTrustNoticeRepo(t, repo, codexHookTrustDoctorHooks)
	t.Chdir(t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--cwd", repo, "--scope", "org-a", "--state-dir", stateDir,
		"--driver", "claude", "--model", "opus", "task text")

	if err != nil {
		t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing for a claude leader", stderr)
	}
	codexStateDir := filepath.Join(t.TempDir(), "state")
	_, codexStderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-b", "--cwd", repo, "--scope", "org-b", "--state-dir", codexStateDir,
		"--driver", "codex", "--model", "gpt-6.1-sol", "task text")
	if err != nil {
		t.Fatalf("start with codex: %v", err)
	}
	if !strings.HasPrefix(codexStderr, "note: org start: codex seat leader: ") {
		t.Errorf("stderr with codex = %q; want the line for a codex leader", codexStderr)
	}
}

// TestCodexHookTrustNotice_StartPlanChecksTheMainCheckout is AC6 for `ralph
// org start --plan` with --driver codex: the feature worktree does not exist
// when the line is printed, and a codex leader in it reads the main
// checkout's hooks, so the line names the main checkout's hooks.json, once,
// and the org starts as before.
func TestCodexHookTrustNotice_StartPlanChecksTheMainCheckout(t *testing.T) {
	setupOrgStubPATH(t)
	setCodexHookTrustNoticeHome(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	mkdirCodexHookTrust(t, filepath.Join(root, ".codex"))
	commitRalphTomls(t, root, map[string]string{filepath.Join(".codex", "hooks.json"): codexHookTrustDoctorHooks})
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	t.Chdir(root)

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--plan", plan, "--feature", "b", "--driver", "codex", "--model", "gpt-6.1-sol")

	if err != nil {
		t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want := codexHookTrustNoticeUnapproved("org start", org.LeaderIdentity, filepath.Join(root, ".codex", "hooks.json"))
	if stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if last := lastSeatEvent(t, stateDir, "b", org.LeaderIdentity); last.Event != org.EventSpawned || last.Driver != "codex" {
		t.Errorf("b's leader = %+v; want spawned with codex", last)
	}
	if !strings.Contains(stdout, "worktree: ") {
		t.Errorf("stdout = %q; want the worktree line", stdout)
	}
}

// TestCodexHookTrustNotice_StartPlanWithClaudePrintsNothing is AC6 for `ralph
// org start --plan` with a claude leader: the main checkout's hooks.json has
// no approval, and stderr names no codex line.
func TestCodexHookTrustNotice_StartPlanWithClaudePrintsNothing(t *testing.T) {
	setupOrgStubPATH(t)
	setCodexHookTrustNoticeHome(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	mkdirCodexHookTrust(t, filepath.Join(root, ".codex"))
	commitRalphTomls(t, root, map[string]string{filepath.Join(".codex", "hooks.json"): codexHookTrustDoctorHooks})
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	t.Chdir(root)

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--plan", plan, "--feature", "b", "--driver", "claude", "--model", "opus")

	if err != nil {
		t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if strings.Contains(stderr, "codex") {
		t.Errorf("stderr = %q; want no codex line for a claude leader", stderr)
	}
}

// TestCodexHookTrustNotice_DirectorStartPrintsOneLineForACodexDirector is
// AC6 for `ralph org director start` with --driver codex (guarded, as a codex
// director must be): the directory the director runs in (the current one,
// for a ledger that is no main worktree's) is checked, the line names the
// codex director once (not a seat: the director is not an org seat), and the
// director starts as before.
func TestCodexHookTrustNotice_DirectorStartPrintsOneLineForACodexDirector(t *testing.T) {
	stateDir, _, cwd := directorStartLedger(t)
	setCodexHookTrustNoticeHome(t)
	hooksPath := initCodexHookTrustNoticeRepo(t, cwd, codexHookTrustDoctorHooks)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--state-dir", stateDir, "--driver", "codex", "--model", "gpt-6-astra",
		"--config", writeDirectorConfig(t, "[org.permissions.roles]\ndirector = \"guarded\"\n"))

	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want := "note: org director start: codex director: there is no codex user config at ~/.codex/config.toml to approve the 2 hooks in " +
		hooksPath + ", so they do not run in the director; run 'ralph doctor' for the keys and how to trust them\n"
	if stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if !strings.HasPrefix(stdout, "started d1 (driver=codex, ") {
		t.Errorf("stdout = %q; want the started line of a codex director", stdout)
	}
}

// TestCodexHookTrustNotice_DirectorStartWithClaudePrintsNothing is AC6 for
// `ralph org director start` with a claude director: the directory it runs
// in holds a hooks.json no approval covers, stderr stays empty, and the
// director starts.
func TestCodexHookTrustNotice_DirectorStartWithClaudePrintsNothing(t *testing.T) {
	stateDir, _, cwd := directorStartLedger(t)
	setCodexHookTrustNoticeHome(t)
	initCodexHookTrustNoticeRepo(t, cwd, codexHookTrustDoctorHooks)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--state-dir", stateDir, "--driver", "claude", "--model", "opus")

	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing for a claude director", stderr)
	}
	if !strings.HasPrefix(stdout, "started d1 (driver=claude, ") {
		t.Errorf("stdout = %q; want a claude director started", stdout)
	}
}

// newCodexHookTrustNoticeCheckout is a main checkout under a temp dir (a
// .git directory holding HEAD, no git run) with hooks as its
// .codex/hooks.json. It returns the checkout and the hooks.json path.
func newCodexHookTrustNoticeCheckout(t *testing.T, hooks string) (repo, hooksPath string) {
	t.Helper()
	repo = filepath.Join(codexHookTrustTempDir(t), "repo")
	makeCodexHookTrustCheckout(t, repo)
	hooksPath = filepath.Join(repo, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, hooksPath, hooks)
	return repo, hooksPath
}

// TestCodexHookTrustNotice_PrintsNothingWithoutAConfigPathOrAHookToApprove
// covers two cases the CLI tests do not reach: the codex user config's path
// cannot be resolved (doctor reports it), and the hooks.json has no handler
// that needs approval.
func TestCodexHookTrustNotice_PrintsNothingWithoutAConfigPathOrAHookToApprove(t *testing.T) {
	repo, _ := newCodexHookTrustNoticeCheckout(t, codexHookTrustDoctorHooks)
	orig := doctorCodexSandboxEnv
	t.Cleanup(func() { doctorCodexSandboxEnv = orig })
	doctorCodexSandboxEnv = func() (codexSandboxEnv, error) { return codexSandboxEnv{}, errors.New("no home here") }
	var out bytes.Buffer
	printCodexHookTrustNotice(&out, "codex", "org spawn", "implementer", repo)
	if out.Len() != 0 {
		t.Errorf("printed %q with no codex config path; want nothing", out.String())
	}

	repo, _ = newCodexHookTrustNoticeCheckout(t, `{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": " "}]}]}}`)
	setCodexHookTrustNoticeHome(t)
	printCodexHookTrustNotice(&out, "codex", "org spawn", "implementer", repo)
	if out.Len() != 0 {
		t.Errorf("printed %q for a hooks.json with no hook that needs approval; want nothing", out.String())
	}
}

// TestCodexHookTrustNotice_WordsOneHookWithoutAConfig is the singular words
// of the line for a missing codex user config and a hooks.json with one hook.
func TestCodexHookTrustNotice_WordsOneHookWithoutAConfig(t *testing.T) {
	repo, hooksPath := newCodexHookTrustNoticeCheckout(t, `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`)
	setCodexHookTrustNoticeHome(t)
	var out bytes.Buffer
	printCodexHookTrustNotice(&out, "codex", "org spawn", "implementer", repo)
	want := "note: org spawn: codex seat implementer: there is no codex user config at ~/.codex/config.toml to approve the 1 hook in " + hooksPath +
		", so it does not run in the seat; run 'ralph doctor' for the key and how to trust it\n"
	if out.String() != want {
		t.Errorf("printed\n%q\nwant\n%q", out.String(), want)
	}
}

// TestCodexHookTrustNotice_NamesTheDirectorByTheVerb is the agent the line
// names: `ralph org director start` gets "codex director" and "in the
// director", since the director is not an org seat, while a seat that
// `ralph org spawn` names "director" is still "codex seat director".
func TestCodexHookTrustNotice_NamesTheDirectorByTheVerb(t *testing.T) {
	repo, hooksPath := newCodexHookTrustNoticeCheckout(t, `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`)
	setCodexHookTrustNoticeHome(t)
	tail := " there is no codex user config at ~/.codex/config.toml to approve the 1 hook in " + hooksPath + ", so it does not run in "
	for _, tc := range []struct{ verb, seat, want string }{
		{"org director start", org.DirectorRole,
			"note: org director start: codex director:" + tail + "the director; run 'ralph doctor' for the key and how to trust it\n"},
		{"org spawn", org.DirectorRole,
			"note: org spawn: codex seat director:" + tail + "the seat; run 'ralph doctor' for the key and how to trust it\n"},
	} {
		var out bytes.Buffer
		printCodexHookTrustNotice(&out, "codex", tc.verb, tc.seat, repo)
		if out.String() != tc.want {
			t.Errorf("%s, seat %s: printed\n%q\nwant\n%q", tc.verb, tc.seat, out.String(), tc.want)
		}
	}
}

// TestCodexHookTrustHelp_EachVerbDescribesTheNote is the help half of AC8:
// `ralph org spawn`, `ralph org start`, and `ralph org director start` each
// end their Long with orgCodexHookTrustNoticeHelp as a paragraph of its own,
// which names the user config it reads and the doctor check, and whose lines
// are wrapped like the rest of the help.
func TestCodexHookTrustHelp_EachVerbDescribesTheNote(t *testing.T) {
	for _, phrase := range []string{"($CODEX_HOME/config.toml, else ~/.codex/config.toml)", "The Codex hook trust check of\nralph doctor"} {
		if !strings.Contains(orgCodexHookTrustNoticeHelp, phrase) {
			t.Errorf("paragraph does not name %q:\n%s", phrase, orgCodexHookTrustNoticeHelp)
		}
	}
	for _, line := range strings.Split(orgCodexHookTrustNoticeHelp, "\n") {
		if len(line) > 76 {
			t.Errorf("help line of %d characters; want at most 76: %q", len(line), line)
		}
	}
	root := NewRootCmd()
	for _, path := range [][]string{{"org", "spawn"}, {"org", "start"}, {"org", "director", "start"}} {
		cmd, rest, err := root.Find(path)
		if err != nil || len(rest) != 0 || cmd.Name() != path[len(path)-1] {
			t.Fatalf("find %v = %v, %v, %v; want that command", path, cmd, rest, err)
		}
		if !strings.HasSuffix(cmd.Long, "\n\n"+orgCodexHookTrustNoticeHelp) {
			t.Errorf("%v --help does not end with the codex hook trust paragraph:\n%s", path, cmd.Long)
		}
	}
}
