#!/usr/bin/env sh
set -eu

# build-tui.sh — Build the ralph-tui binary with version info embedded via ldflags.
#
# Output: bin/ralph-tui

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="${SCRIPT_DIR}/.."

# Check Go is available
if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed or not in PATH." >&2
  echo "Install Go 1.22+ from https://go.dev/dl/" >&2
  exit 1
fi

# Check Go version (need 1.22+)
_go_version="$(go version | sed 's/.*go\([0-9]*\.[0-9]*\).*/\1/')"
_go_major="${_go_version%%.*}"
_go_minor="${_go_version#*.}"
if [ "$_go_major" -lt 1 ] || { [ "$_go_major" -eq 1 ] && [ "$_go_minor" -lt 22 ]; }; then
  echo "Error: Go 1.22+ is required (found go${_go_version})." >&2
  exit 1
fi

# Gather version metadata
_commit="$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")"
_date="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
_version="${RALPH_TUI_VERSION:-dev}"

# Build
_output="${REPO_ROOT}/bin/ralph-tui"
mkdir -p "$(dirname "$_output")"

echo "Building ralph-tui..."
echo "  Version:  ${_version}"
echo "  Commit:   ${_commit}"
echo "  Date:     ${_date}"

go build \
  -ldflags="-s -w -X main.Version=${_version} -X main.GitCommit=${_commit} -X main.BuildDate=${_date}" \
  -o "$_output" \
  "${REPO_ROOT}/cmd/ralph-tui"

chmod +x "$_output"

_size="$(wc -c < "$_output" | tr -d ' ')"
_size_mb="$(echo "scale=1; ${_size} / 1048576" | bc 2>/dev/null || echo "?")"

echo ""
echo "Built: ${_output} (${_size_mb} MB)"
echo "Run:   ${_output} --version"
