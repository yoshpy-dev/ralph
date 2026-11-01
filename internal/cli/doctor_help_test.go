package cli

import (
	"regexp"
	"strings"
	"testing"
)

// AC5 of plan 2026-11-01-org-ledger-gaps (tech-debt 398). pflag takes the
// first back-quoted span of a flag's usage string as the name of its value,
// so a back-quoted `ralph upgrade` in the usage of the --strict flag made
// `ralph doctor --help` print the bool flag as "--strict ralph upgrade". The
// line of the flag must show no word between --strict and its usage, and the
// usage must still tell to run ralph upgrade first. The usage is matched with
// runs of white space read as one space.
func TestDoctorHelp_StrictFlagHasNoValueName(t *testing.T) {
	stdout, _, err := runRootCmdSplit(t, "doctor", "--help")
	if err != nil {
		t.Fatalf("doctor --help: %v", err)
	}
	var flagLine string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--strict ") {
			flagLine = line
			break
		}
	}
	if flagLine == "" {
		t.Fatalf("doctor --help has no line for the --strict flag:\n%s", stdout)
	}
	if want := regexp.MustCompile(`^\s+--strict {2,}fail \(exit 1\) on FR-9 `); !want.MatchString(flagLine) {
		t.Errorf("the --strict line of doctor --help = %q; want --strict, spaces, then the usage (no value name)", flagLine)
	}
	if strings.Contains(flagLine, "--strict ralph upgrade") {
		t.Errorf("the --strict line of doctor --help names its value `ralph upgrade`: %q", flagLine)
	}
	if want := "run ralph upgrade first, then doctor --strict"; !strings.Contains(strings.Join(strings.Fields(flagLine), " "), want) {
		t.Errorf("the --strict usage lacks %q:\n%s", want, flagLine)
	}
}
