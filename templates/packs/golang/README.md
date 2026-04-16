# Go pack

Default verification order:
- gofmt / goimports
- go vet
- golangci-lint (if available)
- staticcheck (if available)
- go test

Customize this pack if your repo uses:
- custom golangci-lint config (.golangci.yml)
- build tags or constraint-gated tests
- integration tests behind a flag
- workspace mode (go.work)
