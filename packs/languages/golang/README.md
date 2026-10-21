# Go pack

Default verification order:
- gofmt / goimports
- go vet
- golangci-lint (if available)
- staticcheck (if available)
- go test, run with `-timeout=30m` first in `GOFLAGS` (go test's own
  default is 10 minutes). A timeout set in `GOFLAGS` or with
  `go env -w GOFLAGS=...` comes later and wins: `GOFLAGS=-timeout=10m`
  gives each package 10 minutes.

Customize this pack if your repo uses:
- custom golangci-lint config (.golangci.yml)
- build tags or constraint-gated tests
- integration tests behind a flag
- workspace mode (go.work)
