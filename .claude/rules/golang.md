---
paths:
  - "**/*.go"
  - "go.mod"
  - "go.sum"
---
# Go rules

- Prefer flat package layouts with clear naming over deep nesting.
- Keep error handling explicit; do not silently swallow errors.
- Use `go fmt`, `go vet`, and `go test ./...` before completion when the project supports them.
- Prefer interfaces at consumption sites, not at definition sites.
- Keep concurrency boundaries (goroutines, channels) visible and well-documented.
