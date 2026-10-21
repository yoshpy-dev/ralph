package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
