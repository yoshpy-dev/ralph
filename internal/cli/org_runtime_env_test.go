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

// TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff pins AC3 of plan
// 2026-10-22-seat-agmsg-self-name on the wiring side: the herdr driver of
// the org verbs' runtime (the seat panes of spawn and start) and of director
// start (the headless director's pane) has exactly AGMSG_SELF_NAME=off as
// its PaneEnv. Two tab creates through each, run against a stand-in herdr
// on PATH, both carry it and check --env once.
func TestOrgRuntimes_HerdrPaneEnvIsSelfNameOff(t *testing.T) {
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
			if want := []string{"AGMSG_SELF_NAME=off"}; !reflect.DeepEqual(h.PaneEnv, want) {
				t.Fatalf("PaneEnv = %q; want %q", h.PaneEnv, want)
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
			tabCreate := "tab create --workspace w1 --cwd /tmp/cwd --label seat-1 --env AGMSG_SELF_NAME=off"
			if got, want := strings.Split(strings.TrimSpace(string(data)), "\n"), []string{"tab create --help", tabCreate, tabCreate}; !slices.Equal(got, want) {
				t.Fatalf("herdr calls = %q; want %q", got, want)
			}
		})
	}
}
