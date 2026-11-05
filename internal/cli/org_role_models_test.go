package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for how `ralph org spawn`, `ralph org start` (<task> and --plan),
// and `ralph org director start` take a seat's or the director's driver and
// model from [org.role_models] (plan
// docs/plans/active/2026-11-05-org-role-models.md, AC5, AC6's display, AC7,
// and AC8's CLI side): --driver and --model may be omitted, the values come
// from the main worktree's ralph.toml by the max_orgs rule, a config that
// still sets [org].model_pool gets one line on stderr, a model of "" is shown
// as (CLI default), and no line about a --model fallback is printed.

// roleModelsLegacyPoolToml is an [org] table that still sets model_pool,
// which ralph no longer reads.
const roleModelsLegacyPoolToml = "[org]\nmodel_pool = [{ driver = \"claude\", model = \"sonnet\" }]\n"

// legacyModelPoolNotice is the line spawn, start, and director start print
// on stderr for a config whose file path sets [org].model_pool.
func legacyModelPoolNotice(path string) string {
	return "org: " + path + " sets [org].model_pool, which ralph no longer reads: [org.role_models] sets each role's driver and model; " +
		"remove model_pool to drop this note\n"
}

// assertNoModelFallbackWarning fails the test when stderr carries the
// warning of the --model fallback ralph no longer has (spec FR-3).
func assertNoModelFallbackWarning(t *testing.T, stderr string) {
	t.Helper()
	for _, stale := range []string{"--model omitted", "falling back"} {
		if strings.Contains(stderr, stale) {
			t.Errorf("stderr says %q; want no --model fallback warning:\n%s", stale, stderr)
		}
	}
}

// roleModelsConfig writes content as a ralph.toml under t.TempDir() and
// returns its path, for --config.
func roleModelsConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, path, content)
	return path
}

// TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel is AC5 for spawn
// (--dry-run): without --driver and --model the seat takes its role's
// [org.role_models] entry, the built-in claude set by default and the codex
// set when driver_pool has no claude; --model alone keeps the entry's
// driver; --driver alone with the entry's driver keeps its model, and with
// another driver passes when the entry's model is "" (the CLI's default); a
// role with no entry takes both flags. The spawned line shows the model,
// "(CLI default)" for "", the manifest records "" as it is, and nothing is
// printed on stderr: no --model fallback warning, and no model_pool note for
// a config that does not set it.
func TestOrgSpawn_RoleModels_DefaultsWithoutDriverOrModel(t *testing.T) {
	const guardedCodex = "[org.permissions]\ndefault = \"guarded\"\n"
	cases := []struct {
		name              string
		config            string // "" passes no --config
		role              string
		flags             []string
		wantDriver, model string
	}{
		{"reviewer under the default config", "", "reviewer", nil, "claude", "opus"},
		{"leader under the default config", "", org.LeaderIdentity, nil, "claude", "opus"},
		{"driver_pool codex only", "[org]\ndriver_pool = [\"codex\"]\n\n" + guardedCodex, "reviewer", nil, "codex", "gpt-6.1-sol"},
		{"--model only keeps the role's driver", "", "reviewer", []string{"--model", "sonnet"}, "claude", "sonnet"},
		{"--driver only, the role's driver", "", "implementer", []string{"--driver", "claude"}, "claude", "opus"},
		{"an entry with model \"\"", "[org.role_models]\nreviewer = { driver = \"claude\", model = \"\" }\n", "reviewer", nil, "claude", ""},
		{"--driver only, another driver, entry model \"\"",
			"[org.role_models]\nreviewer = { driver = \"claude\", model = \"\" }\n\n" + guardedCodex, "reviewer", []string{"--driver", "codex"}, "codex", ""},
		{"an empty --model is an omitted one", "", "reviewer", []string{"--model", ""}, "claude", "opus"},
		{"a custom role with both flags", "", "worker", []string{"--driver", "claude", "--model", "haiku"}, "claude", "haiku"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			t.Chdir(t.TempDir())
			stateDir := filepath.Join(t.TempDir(), "state")
			seat := "seat-1"
			if tc.role == org.LeaderIdentity {
				seat = org.LeaderIdentity
			}
			args := []string{"spawn", "--org-id", "org-a", "--id", seat, "--role", tc.role, "--cwd", t.TempDir(),
				"--scope", "test-scope", "--state-dir", stateDir, "--dry-run"}
			if tc.config != "" {
				args = append(args, "--config", roleModelsConfig(t, tc.config))
			}
			stdout, stderr, err := runOrgCmdStreams(t, append(args, tc.flags...)...)
			if err != nil {
				t.Fatalf("spawn %v: %v (stdout: %s stderr: %s)", tc.flags, err, stdout, stderr)
			}
			shown := tc.model
			if shown == "" {
				shown = "(CLI default)"
			}
			if want := "driver=" + tc.wantDriver + " model=" + shown + " pane_id="; !strings.Contains(stdout, want) {
				t.Errorf("stdout = %q; want it to contain %q", stdout, want)
			}
			if stderr != "" {
				t.Errorf("stderr = %q; want nothing", stderr)
			}
			last := lastSeatEvent(t, stateDir, "org-a", seat)
			if last.Event != org.EventSpawned || !last.DryRun || last.Driver != tc.wantDriver || last.Model != tc.model {
				t.Errorf("last event = %+v; want a dry-run spawned with %s / %q", last, tc.wantDriver, tc.model)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none for a dry run", lines)
			}
		})
	}
}

// TestOrgSpawn_RoleModels_RefusedWithNothingRecorded is the refusal half of
// AC5 and AC3's CLI side: --driver alone with a driver other than the role's
// claude/opus entry is refused naming --model; a role with no entry and a
// missing flag is refused naming the role and its [org.role_models] key; an
// entry whose driver is not in driver_pool is refused naming the role and
// driver_pool (plan D9); a --model with a space is refused. Each refusal
// comes before Spawn: no manifest event, no receipt, no herdr call, and no
// --model fallback warning.
func TestOrgSpawn_RoleModels_RefusedWithNothingRecorded(t *testing.T) {
	cases := []struct {
		name   string
		config string // "" passes no --config
		role   string
		flags  []string
		want   []string
	}{
		{"--driver only, another driver than the entry's", "", "reviewer", []string{"--driver", "codex"},
			[]string{`org: role "reviewer" runs on claude in [org.role_models]; pass --model for --driver codex`}},
		{"a custom role with no flags", "", "worker", nil,
			[]string{`org: role "worker" has no [org.role_models].worker entry: pass both --driver and --model`, "[org.role_models].worker = { driver = "}},
		{"a custom role with --driver only", "", "worker", []string{"--driver", "claude"},
			[]string{`role "worker" has no [org.role_models].worker entry`}},
		{"a custom role with --model only", "", "worker", []string{"--model", "sonnet"},
			[]string{`role "worker" has no [org.role_models].worker entry`}},
		{"an entry whose driver is not in driver_pool",
			"[org]\ndriver_pool = [\"claude\"]\n\n[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n", "reviewer", nil,
			[]string{`org: role "reviewer" runs on driver "codex" in [org.role_models].reviewer, which is not in [org].driver_pool [claude]`}},
		{"a --model with a space", "", "reviewer", []string{"--model", "not a real model"}, []string{"org: --model: "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			t.Chdir(t.TempDir())
			stateDir := filepath.Join(t.TempDir(), "state")
			args := []string{"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", tc.role, "--cwd", t.TempDir(),
				"--scope", "test-scope", "--state-dir", stateDir}
			if tc.config != "" {
				args = append(args, "--config", roleModelsConfig(t, tc.config))
			}
			stdout, stderr, err := runOrgCmdStreams(t, append(args, tc.flags...)...)
			if err == nil {
				t.Fatalf("spawn %v succeeded; want refused (stdout: %s)", tc.flags, stdout)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			assertNoModelFallbackWarning(t, stderr)
			if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
				t.Errorf("manifest = %v; want nothing recorded", eventTypes(events))
			}
			if _, err := os.Stat(org.ReceiptsPathIn(stateDir)); !os.IsNotExist(err) {
				t.Errorf("stat receipts = %v; want no receipt written", err)
			}
			if lines := slices.Concat(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)); len(lines) != 0 {
				t.Errorf("herdr and agmsg calls = %v; want none", lines)
			}
		})
	}
}

// TestOrgStart_RoleModels_LeaderEntryWithoutDriverOrModel is AC5 for
// `ralph org start <task>`: with neither --driver nor --model the leader
// runs with [org.role_models].leader, claude/opus by default, the --config
// file's entry when it sets one, and codex/gpt-6.1-sol when driver_pool has
// no claude (so --driver no longer defaults to claude), through the stub
// herdr; and a config that still sets [org].model_pool gets the note once on
// stderr while the leader starts.
func TestOrgStart_RoleModels_LeaderEntryWithoutDriverOrModel(t *testing.T) {
	cases := []struct {
		name                  string
		config                string // "" passes no --config
		wantDriver, wantModel string
		withNotice            bool
	}{
		{"the default config", "", "claude", "opus", false},
		{"a config with a leader entry", "[org.role_models]\nleader = { driver = \"claude\", model = \"haiku\" }\n", "claude", "haiku", false},
		{"driver_pool codex only", "[org]\ndriver_pool = [\"codex\"]\n", "codex", "gpt-6.1-sol", false},
		{"a config that still sets model_pool", roleModelsLegacyPoolToml, "claude", "opus", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			t.Chdir(t.TempDir())
			stateDir := filepath.Join(t.TempDir(), "state")
			args := []string{"start", "--org-id", "org-a", "--cwd", t.TempDir(), "--scope", "org-a", "--state-dir", stateDir}
			var configPath string
			if tc.config != "" {
				configPath = roleModelsConfig(t, tc.config)
				args = append(args, "--config", configPath)
			}
			stdout, stderr, err := runOrgCmdStreams(t, append(args, "task text")...)
			if err != nil {
				t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			last := lastSeatEvent(t, stateDir, "org-a", org.LeaderIdentity)
			if last.Event != org.EventSpawned || last.Driver != tc.wantDriver || last.Model != tc.wantModel {
				t.Errorf("leader's last event = %+v; want spawned with %s / %s", last, tc.wantDriver, tc.wantModel)
			}
			if want := "driver=" + tc.wantDriver + " model=" + tc.wantModel + " pane_id="; !strings.Contains(stdout, want) {
				t.Errorf("stdout = %q; want it to contain %q", stdout, want)
			}
			wantStderr := ""
			if tc.withNotice {
				wantStderr = legacyModelPoolNotice(configPath)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, wantStderr)
			}
			agentStart := ""
			for _, line := range readLogLines(t, herdrLog) {
				if strings.HasPrefix(line, "agent start ") {
					agentStart = line
				}
			}
			if !strings.Contains(agentStart, " --kind "+tc.wantDriver+" ") || !strings.Contains(agentStart, " --model "+tc.wantModel+" ") {
				t.Errorf("agent start = %q; want --kind %s and --model %s", agentStart, tc.wantDriver, tc.wantModel)
			}
		})
	}
}

// TestOrgSpawn_RoleModels_LegacyModelPoolNoticeOnce is AC8's model_pool half
// for spawn: a --config that still sets [org].model_pool loads, spawn prints
// the note naming that file once on stderr and goes on, and a config that
// does not set it prints nothing.
func TestOrgSpawn_RoleModels_LegacyModelPoolNoticeOnce(t *testing.T) {
	for _, withPool := range []bool{true, false} {
		name := "model_pool set"
		content := roleModelsLegacyPoolToml
		if !withPool {
			name, content = "model_pool not set", "[org]\nmax_seats = 5\n"
		}
		t.Run(name, func(t *testing.T) {
			setupOrgStubPATH(t)
			t.Chdir(t.TempDir())
			stateDir := filepath.Join(t.TempDir(), "state")
			configPath := roleModelsConfig(t, content)
			stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "reviewer",
				"--cwd", t.TempDir(), "--scope", "test-scope", "--state-dir", stateDir, "--config", configPath, "--dry-run")
			if err != nil {
				t.Fatalf("spawn: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			want := ""
			if withPool {
				want = legacyModelPoolNotice(configPath)
			}
			if stderr != want {
				t.Errorf("stderr = %q; want %q", stderr, want)
			}
			if last := lastSeatEvent(t, stateDir, "org-a", "seat-1"); last.Event != org.EventSpawned || last.Model != "opus" {
				t.Errorf("last event = %+v; want spawned with opus, the reviewer entry, not model_pool's sonnet", last)
			}
		})
	}
}

// TestOrgDirectorStart_RoleModels_NoFlagsLaunchesTheCLIDefault is AC5 and
// AC6's display for director start: with neither --driver nor --model the
// director takes [org.role_models].director, claude with a model of "", so
// herdr's agent start has no --model; the started line and director status
// show model=(CLI default), and the registration keeps "" (status --json).
// A config that still sets [org].model_pool gets its note once on stderr.
func TestOrgDirectorStart_RoleModels_NoFlagsLaunchesTheCLIDefault(t *testing.T) {
	for _, withPool := range []bool{false, true} {
		name := "the default config"
		if withPool {
			name = "a config that still sets model_pool"
		}
		t.Run(name, func(t *testing.T) {
			stateDir, herdrLog, cwd := directorStartLedger(t)
			args := []string{"director", "start", "--state-dir", stateDir}
			var configPath string
			if withPool {
				configPath = writeDirectorConfig(t, roleModelsLegacyPoolToml)
				args = append(args, "--config", configPath)
			}
			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			wantStderr := ""
			if withPool {
				wantStderr = legacyModelPoolNotice(configPath)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, wantStderr)
			}
			events := directorEventsIn(t, stateDir)
			if len(events) != 4 || events[0].Driver != "claude" || events[0].Model != "" {
				t.Fatalf("director.jsonl = %+v; want d1 attached with claude and a model of \"\", then three more events", events)
			}
			tok := events[0].Token
			if want := "started d1 (driver=claude, model=(CLI default), permission=autonomous, pane=pane-stub-1, workspace=ws-stub-1)\n"; !strings.HasPrefix(stdout, want) {
				t.Errorf("stdout = %q; want it to start with %q", stdout, want)
			}
			prompt := filepath.Join(stateDir, "prompts", "director_"+tok+".md")
			wantAgentStart := "agent start director_" + tok + " --kind claude --pane pane-stub-1 --timeout 60000 -- --permission-mode bypassPermissions " +
				org.PromptFilePointer(prompt)
			herdr := readLogLines(t, herdrLog)
			if len(herdr) != 3 || herdr[0] != "workspace create --cwd "+cwd+" --label ralph_director_"+tok || herdr[2] != wantAgentStart {
				t.Errorf("herdr calls =\n%s\nwant the third to be\n%s", strings.Join(herdr, "\n"), wantAgentStart)
			}

			if line := directorStatusLine(t, "--state-dir", stateDir); !strings.Contains(line, ", headless running, model=(CLI default), ") {
				t.Errorf("director line = %q; want model=(CLI default)", line)
			}
			jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
			if reg := directorReportOf(t, jsonOut).Registration; reg == nil || reg.Model != "" || reg.Driver != "claude" {
				t.Errorf("director status --json = %s; want the registration with claude and a model of \"\"", jsonOut)
			}
		})
	}
}

// TestOrgDirectorStart_RoleModels_LaunchCmdWithoutModel is AC6 for a
// launch_cmd director: with no flags the launch script execs the agent with
// the permission flags and the prompt pointer and no --model, its text says
// no --model, and the started line shows model=(CLI default).
func TestOrgDirectorStart_RoleModels_LaunchCmdWithoutModel(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	fakes := setDirectorLaunchSeams(t)
	killDirectorLaunchedProcesses(t, stateDir)
	configPath := writeDirectorConfig(t, directorLaunchCmdToml)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	events := directorEventsIn(t, stateDir)
	if len(events) != 2 || events[0].Launch != org.DirectorLaunchCmd || events[0].Model != "" || events[1].PID <= 0 {
		t.Fatalf("director.jsonl = %+v; want attached (launch cmd, a model of \"\") and launched with a pid", events)
	}
	tok := events[0].Token
	if want := "started d1 (driver=claude, model=(CLI default), permission=autonomous, launch=cmd, name=director_" + tok + ", "; !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout = %q; want it to start with %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing", stderr)
	}
	prompt := filepath.Join(stateDir, "prompts", "director_"+tok+".md")
	want := []string{"arg=--permission-mode", "arg=bypassPermissions", "arg=" + org.PromptFilePointer(prompt)}
	if got := waitForDirectorAgentMarker(t, fakes.agentMarker); len(got) < 2 || !slices.Equal(got[2:], want) {
		t.Errorf("the agent recorded\n%q\nwant its arguments to be\n%q", got, want)
	}
	script, err := os.ReadFile(filepath.Join(stateDir, "launch", "director_"+tok+".sh"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(script), "--model") {
		t.Errorf("the launch script passes --model; want none:\n%s", script)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none with launch_cmd", lines)
	}
}

// TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml is AC7 for spawn
// (--dry-run, from a linked worktree whose ralph.toml differs from the main
// worktree's): without --config, [org.role_models] and driver_pool come
// from the main worktree's ralph.toml, and with --config from that file.
// The stale copy case: the linked worktree's driver_pool has no codex, the
// main worktree's has, and the main worktree's ralph.local.toml puts the
// reviewer on codex, so the reviewer spawned without flags runs on codex
// (config.Load of the copy does not check an entry's driver against its
// driver_pool, plan D9). The record of the keys ralph no longer reads comes
// from the main worktree too: an [org.roles] or model_pool only the copy
// sets neither refuses nor prints the note, and the main worktree's
// model_pool prints the note naming its file, and its [org.roles] refuses
// naming that file.
func TestOrgSpawn_RoleModels_ReadFromMainWorktreeRalphToml(t *testing.T) {
	const guarded = "\n[org.permissions]\ndefault = \"guarded\"\n"
	const mainCodexReviewer = "[org]\ndriver_pool = [\"claude\", \"codex\"]\n\n[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n"
	const wtHaikuReviewer = "[org]\ndriver_pool = [\"claude\"]\n\n[org.role_models]\nreviewer = { driver = \"claude\", model = \"haiku\" }\n" + guarded
	cases := []struct {
		name                string
		mainToml, wtToml    string
		mainLocal           string // the main worktree's ralph.local.toml ("" writes none)
		config              bool   // pass --config <worktree>/ralph.toml
		wantDriver, model   string
		wantNotice, refusal func(r legacyLedgerRepo) string // "" for none
	}{
		{name: "no --config: the main worktree's entry and driver_pool", mainToml: mainCodexReviewer, wtToml: wtHaikuReviewer,
			wantDriver: "codex", model: "gpt-6.1-sol"},
		{name: "--config: that file's entry", mainToml: mainCodexReviewer, wtToml: wtHaikuReviewer, config: true,
			wantDriver: "claude", model: "haiku"},
		{name: "a stale copy of driver_pool and the main worktree's ralph.local.toml",
			mainToml: "[org]\ndriver_pool = [\"claude\", \"codex\"]\n", wtToml: "[org]\ndriver_pool = [\"claude\"]\n" + guarded,
			mainLocal:  "[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n",
			wantDriver: "codex", model: "gpt-6.1-sol"},
		{name: "[org.roles] and model_pool only in the copy", mainToml: "[org]\nmax_seats = 5\n",
			wtToml:     "[org]\nmodel_pool = [{ driver = \"claude\", model = \"sonnet\" }]\n\n[org.roles]\nreviewer = [\"sonnet\"]\n" + guarded,
			wantDriver: "claude", model: "opus"},
		{name: "model_pool in the main worktree", mainToml: roleModelsLegacyPoolToml, wtToml: guarded,
			wantDriver: "claude", model: "opus",
			wantNotice: func(r legacyLedgerRepo) string { return legacyModelPoolNotice(filepath.Join(r.mainRoot, "ralph.toml")) }},
		{name: "[org.roles] in the main worktree", mainToml: "[org.roles]\nreviewer = [\"sonnet\"]\n", wtToml: guarded,
			refusal: func(r legacyLedgerRepo) string {
				return "org: " + filepath.Join(r.mainRoot, "ralph.toml") + " sets [org.roles], which ralph no longer reads"
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t) // chdirs into the linked worktree
			writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), tc.mainToml)
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), tc.wtToml)
			if tc.mainLocal != "" {
				t.Setenv(config.LocalConfigEnv, "")
				writeTestFile(t, filepath.Join(r.mainRoot, config.LocalConfigFileName), tc.mainLocal)
			}
			args := []string{"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "reviewer", "--cwd", t.TempDir(),
				"--scope", "test-scope", "--dry-run"}
			if tc.config {
				args = append(args, "--config", filepath.Join(r.worktree, "ralph.toml"))
			}

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if tc.refusal != nil {
				if err == nil || !strings.Contains(err.Error(), tc.refusal(r)) {
					t.Fatalf("spawn = %v; want refused with %q (stdout: %s stderr: %s)", err, tc.refusal(r), stdout, stderr)
				}
				if events := readManifestEvents(t, org.ManifestPathIn(r.sharedDir)); len(events) != 0 {
					t.Errorf("manifest = %v; want nothing recorded", eventTypes(events))
				}
				return
			}
			if err != nil {
				t.Fatalf("spawn: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			last := lastSeatEvent(t, r.sharedDir, "org-a", "seat-1")
			if last.Event != org.EventSpawned || last.Driver != tc.wantDriver || last.Model != tc.model {
				t.Errorf("last event in the shared ledger = %+v; want spawned with %s / %s", last, tc.wantDriver, tc.model)
			}
			wantStderr := ""
			if tc.wantNotice != nil {
				wantStderr = tc.wantNotice(r)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, wantStderr)
			}
		})
	}
}

// TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml is AC7 and AC5
// for start --plan, run from pkg/sub whose ralph.toml sets driver_pool =
// ["codex"]: with neither --driver nor --model and no --config, the leader
// takes [org.role_models].leader and driver_pool from the main worktree's
// ralph.toml (claude/haiku); with --config, from that file (claude/opus),
// which also still sets model_pool, so its note is printed once.
func TestOrgStartPlan_RoleModels_ReadFromMainWorktreeRalphToml(t *testing.T) {
	cases := []struct {
		name       string
		configToml string // "" passes no --config
		wantModel  string
	}{
		{"no --config: the main worktree's", "", "haiku"},
		{"--config: that file's", "[org]\nmodel_pool = [{ driver = \"claude\", model = \"sonnet\" }]\n\n" +
			"[org.role_models]\nleader = { driver = \"claude\", model = \"opus\" }\n", "opus"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
			commitRalphTomls(t, root, map[string]string{
				"ralph.toml": "[org.role_models]\nleader = { driver = \"claude\", model = \"haiku\" }\n",
				filepath.Join("pkg", "sub", "ralph.toml"): "[org]\ndriver_pool = [\"codex\"]\n",
			})
			stateDir := filepath.Join(root, ".harness", "state", "org")
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			args := []string{"start", "--plan", plan, "--feature", "b"}
			var configPath string
			if tc.configToml != "" {
				configPath = filepath.Join(root, ".harness", "custom.toml") // .harness/ exists: the split plan is under it
				writeTestFile(t, configPath, tc.configToml)
				args = append(args, "--config", configPath)
			}
			t.Chdir(sub)

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			last := lastSeatEvent(t, stateDir, "b", org.LeaderIdentity)
			if last.Event != org.EventSpawned || last.Driver != "claude" || last.Model != tc.wantModel {
				t.Errorf("b's leader = %+v; want spawned with claude / %s", last, tc.wantModel)
			}
			wantStderr := ""
			if configPath != "" {
				wantStderr = legacyModelPoolNotice(configPath)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, wantStderr)
			}
			agentStart := ""
			for _, line := range readLogLines(t, herdrLog) {
				if strings.HasPrefix(line, "agent start ") {
					agentStart = line
				}
			}
			if !strings.Contains(agentStart, " --kind claude ") || !strings.Contains(agentStart, " --model "+tc.wantModel+" ") {
				t.Errorf("agent start = %q; want --kind claude and --model %s", agentStart, tc.wantModel)
			}
		})
	}
}

// TestOrgSpawnStartAndDirectorStartHelp_RoleModels pins the help of the
// three verbs that resolve a driver and model from [org.role_models] (plan
// 2026-11-05-org-role-models, FR-3): --driver is no longer required for
// spawn and director start, start's --driver has no "claude" default, the
// flags name the entry they default to, spawn and start explain the rules
// and where the values are read, and none of them names the model_pool
// fallback any more.
func TestOrgSpawnStartAndDirectorStartHelp_RoleModels(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"spawn", "--help"}, []string{
			"driver CLI the seat runs as: claude|codex (default: the driver of the role's [org.role_models] entry)",
			"model name or alias (default: the model of the role's [org.role_models] entry; \"\" there launches the CLI without --model, on its own default model)",
			"Without --driver and --model, the seat runs with the driver and model of its role's [org.role_models] entry;",
			"--model alone keeps the entry's driver. --driver alone, for a driver other than the entry's, also needs --model, unless the entry's model is \"\"",
			"A role with no entry needs both flags. A ralph.toml that still sets [org.roles] is refused; " +
				"one that still sets [org].model_pool, which ralph no longer reads, gets one line on stderr.",
			"Without --config, [org.role_models] and [org].driver_pool are read from the same ralph.toml as max_orgs and max_total_seats below.",
		}},
		{[]string{"start", "--help"}, []string{
			"driver CLI the leader seat runs as: claude|codex (default: the driver of [org.role_models].leader)",
			"model name or alias (default: the model of [org.role_models].leader;",
			"Without --driver and --model, the seat runs with the driver and model of its role's [org.role_models] entry;",
			"max_orgs and max_total_seats below. For start, the role is leader.",
		}},
		{[]string{"director", "start", "--help"}, []string{
			"the director's agent: claude|codex (default: the driver of [org.role_models].director)",
			"model name or alias (default: the model of [org.role_models].director;",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
		for _, stale := range []string{"model_pool entry permitted", "with a warning"} {
			if strings.Contains(help, stale) {
				t.Errorf("%v still says %q:\n%s", tc.args, stale, out)
			}
		}
		// The --driver line itself (not --leader-driver's, whose default
		// stays "claude"): neither required nor with a default value.
		driverLines := 0
		for _, line := range strings.Split(out, "\n") {
			if !strings.Contains(line, "--driver string") {
				continue
			}
			driverLines++
			if strings.Contains(line, "(required)") || strings.Contains(line, `(default "`) {
				t.Errorf("%v: the --driver line %q is required or has a default value; want neither", tc.args, line)
			}
		}
		if driverLines != 1 {
			t.Errorf("%v: %d --driver lines; want 1:\n%s", tc.args, driverLines, out)
		}
	}
}
