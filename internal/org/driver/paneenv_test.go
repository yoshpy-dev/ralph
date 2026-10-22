package driver

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestSeatPaneEnv_ListsBothEntriesInOrder pins AC2 of plan
// 2026-10-22-seat-wip-checkpoint: the panes ralph creates get
// AGMSG_SELF_NAME=off, then RALPH_WIP_CHECKPOINT=off.
func TestSeatPaneEnv_ListsBothEntriesInOrder(t *testing.T) {
	if want := []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"}; !reflect.DeepEqual(SeatPaneEnv(), want) {
		t.Fatalf("SeatPaneEnv() = %q; want %q", SeatPaneEnv(), want)
	}
	if WIPCheckpointEnv != "RALPH_WIP_CHECKPOINT" || WIPCheckpointOff != "RALPH_WIP_CHECKPOINT=off" {
		t.Fatalf("WIPCheckpointEnv, WIPCheckpointOff = %q, %q; want RALPH_WIP_CHECKPOINT, RALPH_WIP_CHECKPOINT=off",
			WIPCheckpointEnv, WIPCheckpointOff)
	}
}

// TestSeatPaneEnv_ReturnsACopy: a caller that changes the slice it got
// changes nothing a later call returns.
func TestSeatPaneEnv_ReturnsACopy(t *testing.T) {
	got := SeatPaneEnv()
	for i := range got {
		got[i] = "CHANGED=1"
	}
	if want := []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"}; !reflect.DeepEqual(SeatPaneEnv(), want) {
		t.Fatalf("SeatPaneEnv() after a caller changed its slice = %q; want %q", SeatPaneEnv(), want)
	}
}

// TestWIPCheckpointOff_IsWhatTheHooksRead: both WIP-checkpoint hooks, and
// their copies under templates/base, skip their commit on exactly the
// variable and value of WIPCheckpointOff, so the entry ralph adds to its
// panes is the one they read.
func TestWIPCheckpointOff_IsWhatTheHooksRead(t *testing.T) {
	name, value, ok := strings.Cut(WIPCheckpointOff, "=")
	if !ok || name != WIPCheckpointEnv {
		t.Fatalf("WIPCheckpointOff = %q; want %s=<value>", WIPCheckpointOff, WIPCheckpointEnv)
	}
	check := `[ "${` + name + `:-}" = "` + value + `" ]`
	root := filepath.Join("..", "..", "..")
	for _, dir := range []string{filepath.Join(root, ".claude", "hooks"), filepath.Join(root, "templates", "base", ".claude", "hooks")} {
		for _, hook := range []string{"session_end_summary.sh", "precompact_checkpoint.sh"} {
			path := filepath.Join(dir, hook)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if !strings.Contains(string(data), check) {
				t.Fatalf("%s does not test %s", path, check)
			}
		}
	}
}
