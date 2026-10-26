package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

// go.mod's go line is a minimum, not a pin: with GOTOOLCHAIN=auto (the
// default) a newer local Go builds and tests the repo, while CI and the
// release build install go.mod's version (setup-go's go-version-file). The
// tests below make that drift fail locally instead of letting the local run
// and CI check two different Go versions.

// checkGoModToolchain returns nil when running, a runtime.Version() value,
// is the Go that goMod names: its toolchain line when it has one (such as
// "go1.27.5"), else its go line (such as "1.27.3", compared as "go1.27.3").
// Only the first space-separated field of running counts, so an experiment
// suffix such as "go1.27.3 X:nocoverageredesign" matches go1.27.3, and a
// "devel go1.28-..." build matches nothing.
func checkGoModToolchain(goMod, running string) error {
	var goLine, toolchainLine string
	for _, line := range strings.Split(goMod, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "go":
			goLine = fields[1]
		case "toolchain":
			toolchainLine = fields[1]
		}
	}
	if goLine == "" && toolchainLine == "" {
		return errors.New("go.mod has neither a toolchain line nor a go line, so the Go that CI and the release build install is unknown")
	}
	read, want := "toolchain", toolchainLine
	if want == "" {
		read, want = "go", "go"+goLine
	}
	if got, _, _ := strings.Cut(running, " "); got == want {
		return nil
	}
	return fmt.Errorf("the tests run on %s, but go.mod's %s line names %s, the Go that CI and the release build install (setup-go's go-version-file); raise that line to the running version in a PR, or run the tests with GOTOOLCHAIN=%s",
		running, read, want, want)
}

func TestCheckGoModToolchain(t *testing.T) {
	const goOnly = "module example.com/m\n\ngo 1.27.3\n\nrequire (\n\tgolang.org/x/sys v0.43.0 // indirect\n)\n"
	tests := []struct {
		name    string
		goMod   string
		running string
		// wantInErr lists what the error must say; nil means no error.
		wantInErr []string
	}{
		{name: "the running Go is the go line's version", goMod: goOnly, running: "go1.27.3"},
		{name: "a patch release apart", goMod: goOnly, running: "go1.27.4",
			wantInErr: []string{"go1.27.4", "go line names go1.27.3", "GOTOOLCHAIN=go1.27.3"}},
		{name: "a minor release apart", goMod: goOnly, running: "go1.28.0",
			wantInErr: []string{"go1.28.0", "go line names go1.27.3", "GOTOOLCHAIN=go1.27.3"}},
		{name: "the toolchain line wins over a go line that matches", goMod: goOnly + "toolchain go1.27.5\n", running: "go1.27.3",
			wantInErr: []string{"go1.27.3", "toolchain line names go1.27.5", "GOTOOLCHAIN=go1.27.5"}},
		{name: "the toolchain line matches and the go line is older", goMod: "module example.com/m\n\ngo 1.27.0\n\ntoolchain go1.27.3\n", running: "go1.27.3"},
		{name: "neither a go line nor a toolchain line", goMod: "module example.com/m\n", running: "go1.27.3",
			wantInErr: []string{"neither a toolchain line nor a go line"}},
		{name: "an experiment suffix after the version", goMod: goOnly, running: "go1.27.3 X:nocoverageredesign"},
		{name: "a devel build", goMod: goOnly, running: "devel go1.28-abc123 Mon Oct 26 2026",
			wantInErr: []string{"devel go1.28-abc123", "go line names go1.27.3", "GOTOOLCHAIN=go1.27.3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkGoModToolchain(tt.goMod, tt.running)
			if tt.wantInErr == nil {
				if err != nil {
					t.Fatalf("checkGoModToolchain(running %q) = %v, want nil", tt.running, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkGoModToolchain(running %q) = nil, want an error saying %q", tt.running, tt.wantInErr)
			}
			for _, want := range tt.wantInErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("checkGoModToolchain(running %q) = %q, want it to contain %q", tt.running, err, want)
				}
			}
		})
	}
}

// TestTheTestsRunOnTheGoThatGoModNames fails when the local Go is not the
// one CI and the release build use. go test runs it in cmd/ralph, so the
// repo's go.mod is two directories up.
func TestTheTestsRunOnTheGoThatGoModNames(t *testing.T) {
	goMod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkGoModToolchain(string(goMod), runtime.Version()); err != nil {
		t.Fatal(err)
	}
}
