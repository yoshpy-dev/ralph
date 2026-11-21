package org

import (
	"fmt"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The ORIGIN header of `ralph org send` (plan 2026-11-21-org-human-decision;
// issue #366). The director sends a human's decision to the leader of a
// running org as a DECISION with the header `ORIGIN: human`, with no inbox
// item behind it, and Send records such a send as a human's decision. ralph
// cannot tell who wrote the message: the record says the message was sent as
// a human's decision, not that a human wrote it.
//
// The header is read the way protocol.Parse reads every header: the key must
// be exactly sendOriginHeader once the spaces around it are trimmed (`Origin:`
// and `origin:` are other keys, which Send sends as before, unmarked), the
// value is trimmed and keeps its case, and only lines of the header block
// count (an `ORIGIN: human` line in the body is body text).
const (
	sendOriginHeader = "ORIGIN"
	sendOriginHuman  = "human"
	// sendOriginHumanDetails is the word a `sent` event's Details start with
	// (after `raw=true `) when the message carries ORIGIN: human, the shape
	// of `raw=true ` and `via=screen `.
	sendOriginHumanDetails = "origin=human "
	// sendOriginHumanMaxBytes bounds the text of a message with ORIGIN: human,
	// which the `sent` event of a real send keeps whole (ManifestEvent.Text).
	// The protocol's body cap does not hold for a --raw text, and a manifest
	// line longer than readJSONLines' 1 MiB buffer would stop every read of
	// the ledger. JSON writes a byte of text as at most six bytes (the escape
	// of `<`, or of a byte that is not UTF-8), so 16 KiB of text stays far
	// below that buffer.
	sendOriginHumanMaxBytes = 16 * 1024
)

// sendOriginRule is the rule every refusal of checkSendOrigin states.
const sendOriginRule = `ORIGIN is allowed only as one "ORIGIN: human" header line on a TYPE: DECISION sent to the leader (--to ` + LeaderIdentity + `)`

// sendOriginError is checkSendOrigin's refusal: what the check found, the
// rule, and that nothing was sent.
type sendOriginError struct{ found string }

func (e sendOriginError) Error() string {
	return "org: send: " + e.found + "; " + sendOriginRule + "; nothing was sent"
}

// checkSendOrigin is Send's check of the ORIGIN header of text, a message to
// seat to. It runs before the dry-run branch, for Raw and DryRun too, so a
// refusal types nothing and records nothing, and a dry run predicts it.
//
// A text with no ORIGIN header line passes, unmarked (human false). So does a
// text whose first line is not a `TYPE: <value>` header: such a text (only
// --raw lets one through) has no header block, and an `ORIGIN:` line in it is
// free text. Otherwise the text must have exactly one ORIGIN header line,
// whose value is sendOriginHuman, on a message whose TYPE (the first line's,
// as Parse reads it) is DECISION, sent to the leader seat (LeaderIdentity),
// and be at most sendOriginHumanMaxBytes long; human is then true. Any other
// ORIGIN header is refused with a sendOriginError, a --raw text's included.
func checkSendOrigin(text, to string) (human bool, err error) {
	values := protocol.HeaderValues(text, sendOriginHeader)
	if len(values) == 0 {
		return false, nil
	}
	if len(values) > 1 {
		return false, sendOriginError{found: fmt.Sprintf("the message has %d ORIGIN header lines", len(values))}
	}
	// HeaderValues found a header block, so Parse reads it without an error.
	// Were it ever to fail, msg.Type is "" and the TYPE check refuses.
	msg, _ := protocol.Parse(text)
	if msg.Type != protocol.TypeDecision {
		return false, sendOriginError{found: fmt.Sprintf("the message is TYPE: %s with an ORIGIN header", msg.Type)}
	}
	if values[0] != sendOriginHuman {
		return false, sendOriginError{found: fmt.Sprintf("the message has ORIGIN: %q, not %q", values[0], sendOriginHuman)}
	}
	if to != LeaderIdentity {
		return false, sendOriginError{found: fmt.Sprintf("the message with ORIGIN: human is for seat %q, not the leader", to)}
	}
	if len(text) > sendOriginHumanMaxBytes {
		return false, fmt.Errorf("org: send: the message with ORIGIN: human is %d bytes; the ledger keeps such a message whole, so it may be at most %d bytes; nothing was sent", len(text), sendOriginHumanMaxBytes)
	}
	return true, nil
}

// sentHumanText is the Text of the `sent` event of a real send of text:
// text itself when it carries ORIGIN: human (human, from checkSendOrigin),
// and "" for every other message.
func sentHumanText(human bool, text string) string {
	if !human {
		return ""
	}
	return text
}

// sentDetailsHumanOrigin reports whether details, the Details of a `sent`
// event, start with the mark Send writes for a message with ORIGIN: human:
// sendOriginHumanDetails, at the start or right after `raw=true `. The mark
// alone does not tell a human's decision: the Details of a --raw text that
// itself starts with `origin=human ` read the same, so a reader also needs
// the event's Text, which Send keeps only for a real send of a message with
// ORIGIN: human (the progress section of a restarted leader,
// readReseatProgress, reads both).
func sentDetailsHumanOrigin(details string) bool {
	return strings.HasPrefix(strings.TrimPrefix(details, "raw=true "), sendOriginHumanDetails)
}
