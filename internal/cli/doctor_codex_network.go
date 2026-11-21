package cli

import (
	"fmt"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// codexSeatNetworkCheckName is the name of `ralph doctor`'s line for the
// network of codex seats (Check 11b-2).
const codexSeatNetworkCheckName = "Codex sandbox (seat network)"

// codexSeatNetworkReseatClause ends both the info and the warn of
// checkCodexSeatNetwork: ralph fixes a seat's arguments when it starts the
// seat, `ralph org spawn` does not start a running seat again, and
// `ralph org resume` only types into its pane, so a change to the key
// reaches only the seats started after it (plan 2026-11-21-codex-seat-network,
// the plan advisory's finding 1).
const codexSeatNetworkReseatClause = "a change to [org.permissions].codex_network_access reaches only the codex seats started after it: " +
	"a running seat keeps its arguments until it is started again, which ralph org reseat does for the seats of a feature org " +
	"(docs/recipes/codex-seat-permissions.md)"

// checkCodexSeatNetwork is `ralph doctor`'s "Codex sandbox (seat network)"
// check (plan 2026-11-21-codex-seat-network, issue #362): whether the codex
// seats ralph starts under --sandbox workspace-write may reach the network.
// internal/org's permissionArgsForDriver follows such a seat's sandbox flags
// with `-c sandbox_workspace_write.network_access=<bool>`, the value of
// [org.permissions].codex_network_access, so the key alone decides it. It
// reads only orgCfg and never spawns a process.
//
// Whether such a seat can be started at all is decided as the agmsg
// writable-root check (Check 11b) decides it, with the same predicates:
// codexSeatPossible (codex is in [org].driver_pool, which lets any role run
// on codex with --driver codex, whatever its [org.role_models] entry says)
// and codexSeatModesPossible (a role, the director's own entry aside,
// resolves to edits or autonomous in [org.permissions]), plus
// [org.permissions].codex_verified, without which permissionArgsForDriver
// refuses codex in edits and autonomous and a codex seat starts only in
// guarded, which passes no sandbox flag and follows the operator's codex
// config.
//
// Outcomes, in order:
//  1. codex is not in [org].driver_pool -> pass, not needed (the wording of
//     Check 11b and of the Codex hook trust check).
//  2. codex_verified is false, or no role resolves to edits or autonomous ->
//     pass, not needed, naming which (codexNotNeededWhyA, as Check 11b).
//  3. codex_network_access is true -> info: ralph lets those seats reach the
//     network, so a check that needs it (tflint's plugin init, a dependency
//     download) can reach it there.
//  4. codex_network_access is false -> warn: such a check fails inside those
//     seats; set the key to true, or run such checks outside codex seats.
//
// Both 3 and 4 end with codexSeatNetworkReseatClause. Like Check 11b it
// never returns "fail", and --strict, which raises only the scaffold
// checks, leaves its warn a warn. Like Check 11b it trusts orgCfg as given:
// after a load error of ralph.toml, runDoctorStateDir passes the defaults'
// [org], under which this check is the info of outcome 3.
func checkCodexSeatNetwork(orgCfg config.OrgConfig) checkResult {
	r := checkResult{Name: codexSeatNetworkCheckName}
	if !codexSeatPossible(orgCfg) {
		r.Status = "pass"
		r.Detail = "not needed: codex is not in [org].driver_pool, so no codex seat can be spawned"
		return r
	}
	workspaceWriteRole, _ := codexSeatModesPossible(orgCfg)
	if !orgCfg.Permissions.CodexVerified || !workspaceWriteRole {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("not needed: %s, so no codex seat runs under --sandbox workspace-write",
			codexNotNeededWhyA(orgCfg.Permissions.CodexVerified))
		return r
	}
	if orgCfg.Permissions.CodexNetworkAccess {
		r.Status = "info"
		r.Detail = "[org.permissions].codex_network_access = true: ralph passes -c sandbox_workspace_write.network_access=true " +
			"to codex seats in edits and autonomous (--sandbox workspace-write), so a check that needs the network, " +
			"such as tflint's plugin init or a dependency download, can reach it inside them; " + codexSeatNetworkReseatClause
		return r
	}
	r.Status = "warn"
	r.Detail = "[org.permissions].codex_network_access = false: ralph passes -c sandbox_workspace_write.network_access=false " +
		"to codex seats in edits and autonomous (--sandbox workspace-write), so a check that needs the network, " +
		"such as tflint's plugin init or a dependency download, fails inside them; " +
		"set the key to true, or run such checks outside codex seats; " + codexSeatNetworkReseatClause
	return r
}
