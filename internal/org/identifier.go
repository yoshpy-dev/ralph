package org

import (
	"fmt"
	"regexp"
)

// identifierPattern is the allowed shape for any user-supplied identifier
// that org turns into a filesystem path component (OrgID, SeatID):
// alphanumeric-first, then up to 63 more alphanumeric/underscore/hyphen
// characters. This blocks path separators (`/`, `\`) and traversal
// sequences (`..`) before an identifier is ever used to build a path -- see
// promptFilePath's doc comment in spawn.go for the concrete path-traversal
// case this closes (--id '../../../../tmp/pwn' normalizing out of the
// state dir).
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

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
