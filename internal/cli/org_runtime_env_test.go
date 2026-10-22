package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff pins AC2 of plan
// 2026-10-22-seat-agmsg-self-name: the runtime the org verbs build runs
// agmsg through a Runner that implements driver.EnvRunner, so Agmsg.run
// adds AGMSG_SELF_NAME=off instead of falling back to Run. The second half
// runs a stand-in team.sh (a plain bash script, not agmsg) through that
// runtime and reads the value the script sees while ralph's own
// environment says on.
func TestNewOrgRuntimeAt_AgmsgRunnerAddsSelfNameOff(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/bash\nprintf '%s' \"$AGMSG_SELF_NAME\"\n"
	if err := os.WriteFile(filepath.Join(home, "scripts", "team.sh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(driver.AgmsgHomeEnv, home)
	t.Setenv("AGMSG_SELF_NAME", "on")

	rt, err := newOrgRuntimeAt(filepath.Join(t.TempDir(), "state"), "")
	if err != nil {
		t.Fatalf("newOrgRuntimeAt: %v", err)
	}
	agmsg, ok := rt.Agmsg.(driver.Agmsg)
	if !ok {
		t.Fatalf("rt.Agmsg is %T; want a driver.Agmsg", rt.Agmsg)
	}
	if _, ok := agmsg.R.(driver.EnvRunner); !ok {
		t.Fatalf("rt.Agmsg's Runner is %T; want a driver.EnvRunner", agmsg.R)
	}
	got, err := agmsg.TeamMembers(context.Background(), "ralph-org-1")
	if err != nil {
		t.Fatalf("TeamMembers: %v", err)
	}
	if got != "off" {
		t.Fatalf("team.sh saw AGMSG_SELF_NAME=%q; want %q", got, "off")
	}
}

// orgRuntimeEnvHerdrStub is a stand-in herdr (a plain sh script, not herdr)
// that logs its argv, answers `tab create --help` with a help listing --env
// as herdr 0.7.5 does, and answers anything else with a tab create envelope.
const orgRuntimeEnvHerdrStub = `#!/bin/sh
echo "$*" >> "$ORG_RUNTIME_ENV_HERDR_LOG"
if [ "$*" = "tab create --help" ]; then
  printf '%s\n' 'Usage: herdr tab create [OPTIONS]' '' 'Options:' '      --label <TEXT>' '      --env <KEY=VALUE>'
  exit 0
fi
echo '{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"w1:p2","tab_id":"w1:t2","workspace_id":"w1"},"tab":{"tab_id":"w1:t2"},"type":"tab_created"}}'
`

// TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv pins AC3 of plan
// 2026-10-22-seat-agmsg-self-name and AC2 of plan
// 2026-10-22-seat-wip-checkpoint on the wiring side: the herdr driver of
// the org verbs' runtime (the seat panes of spawn and start) and of director
// start (the headless director's pane) has exactly driver.SeatPaneEnv,
// AGMSG_SELF_NAME=off then RALPH_WIP_CHECKPOINT=off, as its PaneEnv. Two
// tab creates through each, run against a stand-in herdr on PATH, both
// carry them and check --env once.
func TestOrgRuntimes_HerdrPaneEnvIsSeatPaneEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "herdr"), []byte(orgRuntimeEnvHerdrStub), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	rt, err := newOrgRuntimeAt(filepath.Join(t.TempDir(), "state"), "")
	if err != nil {
		t.Fatalf("newOrgRuntimeAt: %v", err)
	}
	for name, client := range map[string]org.HerdrClient{
		"newOrgRuntimeAt":            rt.Herdr,
		"newOrgDirectorHerdrRuntime": newOrgDirectorHerdrRuntime(config.Default().Org).Herdr,
	} {
		t.Run(name, func(t *testing.T) {
			h, ok := client.(driver.Herdr)
			if !ok {
				t.Fatalf("Herdr is %T; want a driver.Herdr", client)
			}
			if want := []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"}; !reflect.DeepEqual(h.PaneEnv, want) {
				t.Fatalf("PaneEnv = %q; want %q", h.PaneEnv, want)
			}
			if want := driver.SeatPaneEnv(); !reflect.DeepEqual(h.PaneEnv, want) {
				t.Fatalf("PaneEnv = %q; want driver.SeatPaneEnv() %q", h.PaneEnv, want)
			}

			herdrLog := filepath.Join(t.TempDir(), "herdr.log")
			t.Setenv("ORG_RUNTIME_ENV_HERDR_LOG", herdrLog)
			for range 2 {
				if pane, err := client.TabCreate(context.Background(), "w1", "/tmp/cwd", "seat-1"); err != nil || pane != "w1:p2" {
					t.Fatalf("TabCreate = %q, %v; want w1:p2", pane, err)
				}
			}
			data, err := os.ReadFile(herdrLog)
			if err != nil {
				t.Fatal(err)
			}
			tabCreate := "tab create --workspace w1 --cwd /tmp/cwd --label seat-1 --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off"
			if got, want := strings.Split(strings.TrimSpace(string(data)), "\n"), []string{"tab create --help", tabCreate, tabCreate}; !slices.Equal(got, want) {
				t.Fatalf("herdr calls = %q; want %q", got, want)
			}
		})
	}
}

// TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr pins AC4 of plan
// 2026-10-22-seat-agmsg-self-name at the CLI: against setupOrgStubPATH's
// herdr in mode ORG_STUB_TAB_CREATE_HELP=noenv, whose `tab create --help`
// lists no --env, `ralph org spawn` and `ralph org director start` succeed,
// create their tab without --env, and write the one warning line to the
// process's stderr (the CLI builds its Herdr with Warn nil). It runs the
// verbs as main does, with os.Stdout and os.Stderr swapped, so it does not
// run in parallel.
func TestOrgVerbs_HerdrWithoutEnvFlagWarnsOnStderr(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T) (herdrLog string, args []string)
		wantOut string // the start of stdout
	}{
		{
			name: "spawn",
			setup: func(t *testing.T) (string, []string) {
				herdrLog, _ := setupOrgStubPATH(t)
				return herdrLog, []string{"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
					"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
					"--scope", "test-scope", "--state-dir", filepath.Join(t.TempDir(), "state")}
			},
			wantOut: `spawned seat "seat-1" `,
		},
		{
			name: "director start",
			setup: func(t *testing.T) (string, []string) {
				stateDir, herdrLog, _ := directorStartLedger(t)
				t.Setenv("USER", "alice")
				return herdrLog, []string{"director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir}
			},
			wantOut: "started d1 ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, args := tc.setup(t)
			t.Setenv("ORG_STUB_TAB_CREATE_HELP", "noenv")

			stdout, stderr, err := runOrgCmdSplitStreams(t, args...)
			if err != nil {
				t.Fatalf("%s: %v (stdout: %s stderr: %s)", tc.name, err, stdout, stderr)
			}
			if !strings.HasPrefix(stdout, tc.wantOut) {
				t.Fatalf("stdout = %q; want it to start with %q", stdout, tc.wantOut)
			}
			lines := readLogLines(t, herdrLog)
			if n := countLinesWithPrefix(lines, "tab create"); n != 1 {
				t.Fatalf("tab create calls = %d; want 1 (herdr calls %q)", n, lines)
			}
			for _, l := range lines {
				if strings.Contains(l, "--env") {
					t.Fatalf("herdr call %q has --env; want none on a herdr without it", l)
				}
			}
			const warning = "warning: herdr tab create has no --env flag, so ralph creates herdr panes without --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off: "
			if !strings.HasPrefix(stderr, warning) || strings.Count(stderr, "\n") != 1 || !strings.HasSuffix(stderr, "\n") {
				t.Fatalf("stderr = %q; want exactly the one line that starts with %q", stderr, warning)
			}
			// The line names the consequence of each entry (plan
			// 2026-10-22-seat-wip-checkpoint, AC4).
			for _, clause := range []string{"may rename the pane's herdr agent", "may commit a feature branch's uncommitted changes as 'wip:'"} {
				if !strings.Contains(stderr, clause) {
					t.Fatalf("stderr = %q; want it to say %q", stderr, clause)
				}
			}
		})
	}
}
