package main

import (
	"fmt"
	"os"

	ralph "github.com/yoshpy-dev/harness-engineering-scaffolding-template"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/cli"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

func main() {
	// Inject build-time variables and embedded templates.
	cli.Version = Version
	cli.GitCommit = GitCommit
	cli.BuildDate = BuildDate
	scaffold.EmbeddedFS = ralph.TemplatesFS

	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
