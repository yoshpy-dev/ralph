// Package toolchain finds the places in a repository where its tools are
// measured and measures them: which node, package manager (pnpm, yarn, or
// npm), python3, and go a shell resolves in each place, and at which
// version. It also reads the versions a repository asks for: npm version
// ranges (engines.node), the specs of .node-version and .nvmrc, and the
// packageManager field of package.json.
//
// Two callers share it (plan 2026-11-12-gate-tool-versions, D8): the feature
// PR gate (`ralph org gate`, and `ralph org regate` when it runs the gate
// scripts; internal/org), which logs and records the versions before it runs
// the scripts, and the "Node toolchain" check of `ralph doctor`
// (internal/cli), which compares them with what the repository asks for. Both
// measure the same places (LayoutFromFiles).
//
// Measuring changes nothing (AC10). The package installs nothing and writes
// no file. The probe script runs only `command -v` and each tool's version
// command, with stdin from /dev/null, in an environment whose last entries
// (ProbeEnv) stop mise from installing a missing version
// (MISE_AUTO_INSTALL=0, MISE_NOT_FOUND_AUTO_INSTALL=0), corepack from
// downloading a package manager, asking to, or pinning one in package.json
// (COREPACK_ENABLE_NETWORK=0, COREPACK_ENABLE_DOWNLOAD_PROMPT=0,
// COREPACK_ENABLE_AUTO_PIN=0), npm from checking for its own update
// (NPM_CONFIG_UPDATE_NOTIFIER=false), and `go version` from downloading the
// toolchain a go.mod names into the module cache (GOTOOLCHAIN=local). A
// version that is not installed, or would have to be downloaded, therefore
// reads as unknown. A tool that ignores those variables can still write; for
// that, D7 has the gate compare the working tree before and after the probe.
package toolchain
