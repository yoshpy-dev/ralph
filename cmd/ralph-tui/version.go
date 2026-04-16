package main

import "fmt"

// Version, GitCommit, and BuildDate are set via ldflags at build time.
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func versionString() string {
	return fmt.Sprintf("ralph-tui %s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}
