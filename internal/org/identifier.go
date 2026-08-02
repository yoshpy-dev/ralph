package org

import (
	"fmt"
	"regexp"
)

// identifierPattern is the allowed shape for any user-supplied identifier
// that org turns into a filesystem path component (OrgID, SeatID):
// lowercase-letter-first, then up to 29 more lowercase letters, digits, or
// hyphens (30 chars total). This blocks path separators (`/`, `\`) and
// traversal sequences (`..`) before an identifier is ever used to build a
// path -- see promptFilePath's doc comment in spawn.go for the concrete
// path-traversal case this closes (--id '../../../../tmp/pwn' normalizing
// out of the state dir).
//
// The charset is deliberately narrower than "anything path-safe": it is
// scoped to what herdr's own agent-name validation accepts, confirmed by a
// live probe against herdr v0.7.5 (`^[a-z][a-z0-9_-]{0,31}$` -- lowercase
// start, lowercase letters/digits/hyphen/underscore, max 32 chars). ralph
// derives herdr agent names and prompt file names by joining OrgID and
// SeatID (see herdrAgentName/promptFilePath in spawn.go), so both IDs must
// already be herdr-legal on their own before the join even happens.
//
// Underscore is intentionally excluded from *this* pattern (even though
// herdr itself allows it) and reserved as the join separator: since no
// valid OrgID or SeatID can ever contain `_`, `<org>_<seat>` is guaranteed
// to split back into exactly one (org, seat) pair, closing the ambiguity a
// hyphen join has (`a-b`+`c` and `a`+`b-c` both joining to `a-b-c`). See the
// cross-review cycle-2 fix note in
// docs/plans/active/2026-08-02-org-runtime-seats.md, "Implementation notes
// (deviations)".
//
// The per-identifier cap is 30 (not herdr's 32) because the herdr agent name
// is `<org>_<seat>`: 32 minus the separator minus at least one character for
// the other identifier. The combined-length check in Spawn enforces the
// exact `len(org)+1+len(seat) <= 32` budget.
var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,29}$`)

// ValidateIdentifier rejects a value that is not shaped like a safe org/seat
// id. kind is used only in the returned error message (e.g. "org_id",
// "seat_id") so callers can produce distinguishable diagnostics. Called from
// two layers: (*Org).Spawn, before any manifest write is attempted, and the
// CLI's requireOrgID/seat-flag parsing (internal/cli/org.go), before an
// org.Org is even constructed -- both layers reject the same shape so a
// malformed id can never reach a path-building call site regardless of
// entry point.
func ValidateIdentifier(kind, value string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("org: invalid %s %q: must match %s", kind, value, identifierPattern.String())
	}
	return nil
}
