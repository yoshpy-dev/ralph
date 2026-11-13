package org

import (
	"errors"
	"strings"
)

// The bracketed paste markers (plan 2026-11-13-send-claude-paste, D1; issue
// #319). A terminal program that turned bracketed paste mode on, as Claude
// Code does, takes everything between bracketedPasteStart and
// bracketedPasteEnd as one paste, not as keys typed one at a time.
const (
	bracketedPasteStart = "\x1b[200~"
	bracketedPasteEnd   = "\x1b[201~"
)

// errSendPasteMarker is Send's refusal of a message that holds a bracketed
// paste marker (checkNoPasteMarkers).
var errSendPasteMarker = errors.New("org: send: the message contains a bracketed paste marker (ESC [200~ or ESC [201~), which a seat's terminal reads as the start or the end of a paste; nothing was sent")

// sendTextForDriver is the text Send types into the pane of a seat whose
// driver is driver (SeatStatus.Driver): text between the bracketed paste
// markers for "claude", and text unchanged for any other driver ("codex",
// "", or another value). Claude Code 2.1.316 dropped the start of a text of
// more than about 800 bytes typed plainly with `herdr pane send-text`, and
// took the same text whole when it came as a bracketed paste
// (docs/evidence/org-send-paste-probe-2026-11-13.md, A1, B1, and B2).
func sendTextForDriver(driver, text string) string {
	if driver != "claude" {
		return text
	}
	return bracketedPasteStart + text + bracketedPasteEnd
}

// checkNoPasteMarkers is Send's refusal of a message text that holds either
// bracketed paste marker anywhere (plan 2026-11-13-send-claude-paste, D3): an
// end marker inside a claude seat's paste would end it there and type the
// rest as keys. Send runs it right after the protocol validation, for Raw and
// DryRun too, before it looks at the seat, so the refusal does not depend on
// the driver and nothing is typed or recorded.
func checkNoPasteMarkers(text string) error {
	if strings.Contains(text, bracketedPasteStart) || strings.Contains(text, bracketedPasteEnd) {
		return errSendPasteMarker
	}
	return nil
}
