package protocol

import (
	"errors"
	"strings"
	"testing"
)

func TestParse_EveryEnumType_HeaderOnly(t *testing.T) {
	// Every TYPE parses successfully as a header-only message (TASK_ID
	// requirement is Validate's concern, not Parse's).
	for _, typ := range []string{
		TypeTask, TypeResult, TypeQuestion, TypeReview, TypeDecision,
		TypeBlocked, TypeContract, TypeHeartbeat, TypeStop, TypeHello, TypeAlert,
	} {
		t.Run(typ, func(t *testing.T) {
			m, err := Parse("TYPE: " + typ)
			if err != nil {
				t.Fatalf("Parse: unexpected error: %v", err)
			}
			if m.Type != typ {
				t.Fatalf("Type = %q, want %q", m.Type, typ)
			}
			if m.Body != "" {
				t.Fatalf("Body = %q, want empty for a header-only message", m.Body)
			}
		})
	}
}

func TestParse_MissingType_FirstLineNotTypeHeader(t *testing.T) {
	_, err := Parse("TASK_ID: t-1\nsome body")
	if !errors.Is(err, ErrMissingType) {
		t.Fatalf("expected ErrMissingType, got %v", err)
	}
}

func TestParse_EmptyMessage(t *testing.T) {
	_, err := Parse("")
	if !errors.Is(err, ErrMissingType) {
		t.Fatalf("expected ErrMissingType for an empty message, got %v", err)
	}
}

func TestParse_HeadersThenBlankLineThenBody(t *testing.T) {
	m, err := Parse("TYPE: TASK\nTASK_ID: t-1\nSEAT: reviewer\n\nline one\nline two")
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	if m.Type != "TASK" {
		t.Fatalf("Type = %q, want TASK", m.Type)
	}
	if m.TaskID != "t-1" {
		t.Fatalf("TaskID = %q, want t-1", m.TaskID)
	}
	if m.Fields["SEAT"] != "reviewer" {
		t.Fatalf("Fields[SEAT] = %q, want reviewer", m.Fields["SEAT"])
	}
	if _, ok := m.Fields["TYPE"]; ok {
		t.Fatalf("Fields must not duplicate the promoted TYPE key, got %+v", m.Fields)
	}
	if _, ok := m.Fields["TASK_ID"]; ok {
		t.Fatalf("Fields must not duplicate the promoted TASK_ID key, got %+v", m.Fields)
	}
	wantBody := "line one\nline two"
	if m.Body != wantBody {
		t.Fatalf("Body = %q, want %q", m.Body, wantBody)
	}
}

func TestParse_NonHeaderLineEndsHeaderBlock_BodyIncludesThatLine(t *testing.T) {
	// No blank-line separator: the first non-"KEY: value" line ends the
	// header block and becomes the first line of Body (inclusive).
	m, err := Parse("TYPE: HEARTBEAT\nthis is not a header line\nmore body")
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	wantBody := "this is not a header line\nmore body"
	if m.Body != wantBody {
		t.Fatalf("Body = %q, want %q", m.Body, wantBody)
	}
}

func TestParse_HeaderOnly_NoTrailingBlankLine(t *testing.T) {
	m, err := Parse("TYPE: STOP\nSEAT: reviewer")
	if err != nil {
		t.Fatalf("Parse: unexpected error: %v", err)
	}
	if m.Body != "" {
		t.Fatalf("Body = %q, want empty", m.Body)
	}
	if m.Fields["SEAT"] != "reviewer" {
		t.Fatalf("Fields[SEAT] = %q, want reviewer", m.Fields["SEAT"])
	}
}

func TestValidate_UnknownType(t *testing.T) {
	err := Validate(Message{Type: "NOT_A_TYPE"}, 0)
	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("expected ErrUnknownType, got %v", err)
	}
	// sortedTypeNames must list every enum member, including ALERT, in the
	// error message so an unknown-TYPE error is self-documenting.
	if !strings.Contains(err.Error(), TypeAlert) {
		t.Fatalf("expected error message to list %s among valid types, got %v", TypeAlert, err)
	}
}

func TestValidate_MissingTypeString(t *testing.T) {
	err := Validate(Message{Type: ""}, 0)
	if !errors.Is(err, ErrMissingType) {
		t.Fatalf("expected ErrMissingType, got %v", err)
	}
}

func TestValidate_TaskIDRequiredMatrix(t *testing.T) {
	cases := []struct {
		typ      string
		required bool
	}{
		{TypeTask, true},
		{TypeResult, true},
		{TypeReview, true},
		{TypeBlocked, true},
		{TypeContract, true},
		{TypeQuestion, false},
		{TypeDecision, false},
		{TypeHeartbeat, false},
		{TypeStop, false},
		{TypeHello, false},
		{TypeAlert, false},
	}
	for _, c := range cases {
		t.Run(c.typ+"_without_task_id", func(t *testing.T) {
			err := Validate(Message{Type: c.typ}, 0)
			if c.required && !errors.Is(err, ErrMissingTaskID) {
				t.Fatalf("expected ErrMissingTaskID for %s without TASK_ID, got %v", c.typ, err)
			}
			if !c.required && err != nil {
				t.Fatalf("expected no error for %s without TASK_ID, got %v", c.typ, err)
			}
		})
		t.Run(c.typ+"_with_task_id", func(t *testing.T) {
			err := Validate(Message{Type: c.typ, TaskID: "t-1"}, 0)
			if err != nil {
				t.Fatalf("expected no error for %s with TASK_ID set, got %v", c.typ, err)
			}
		})
		t.Run(c.typ+"_with_blank_task_id", func(t *testing.T) {
			// A TASK_ID header present but blank ("TASK_ID: ") must be
			// treated the same as absent for the required types.
			err := Validate(Message{Type: c.typ, TaskID: "   "}, 0)
			if c.required && !errors.Is(err, ErrMissingTaskID) {
				t.Fatalf("expected ErrMissingTaskID for %s with a blank TASK_ID, got %v", c.typ, err)
			}
		})
	}
}

func TestValidate_BodySizeCap_BoundaryAndOverLimit(t *testing.T) {
	atCap := strings.Repeat("a", 10)
	if err := Validate(Message{Type: TypeHeartbeat, Body: atCap}, 10); err != nil {
		t.Fatalf("expected body exactly at the cap to pass, got %v", err)
	}

	overCap := strings.Repeat("a", 11)
	err := Validate(Message{Type: TypeHeartbeat, Body: overCap}, 10)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge for a body 1 char over the cap, got %v", err)
	}
}

func TestValidate_BodySizeCap_ZeroFallsBackToDefault(t *testing.T) {
	atDefault := strings.Repeat("a", DefaultMaxBodyChars)
	if err := Validate(Message{Type: TypeHeartbeat, Body: atDefault}, 0); err != nil {
		t.Fatalf("expected body at DefaultMaxBodyChars to pass with maxBodyChars=0, got %v", err)
	}

	overDefault := strings.Repeat("a", DefaultMaxBodyChars+1)
	if err := Validate(Message{Type: TypeHeartbeat, Body: overDefault}, 0); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge over DefaultMaxBodyChars with maxBodyChars=0, got %v", err)
	}
}

func TestValidate_BodySizeCap_CountsRunesNotBytes(t *testing.T) {
	// Multi-byte characters (e.g. Japanese) must be counted as one char
	// each, not by UTF-8 byte length.
	body := strings.Repeat("あ", 10) // 10 runes, 30 bytes in UTF-8
	if err := Validate(Message{Type: TypeHeartbeat, Body: body}, 10); err != nil {
		t.Fatalf("expected a 10-rune body to pass a 10-char cap, got %v", err)
	}
}

func TestValidateText_ParseThenValidate_Convenience(t *testing.T) {
	if err := ValidateText("TYPE: RESULT\nTASK_ID: t-1\n\nEVIDENCE: commit abc123", 0); err != nil {
		t.Fatalf("expected a well-formed RESULT to pass, got %v", err)
	}

	err := ValidateText("TYPE: RESULT\n\nmissing task id", 0)
	if !errors.Is(err, ErrMissingTaskID) {
		t.Fatalf("expected ErrMissingTaskID to propagate through ValidateText, got %v", err)
	}

	err = ValidateText("not a valid header line at all", 0)
	if !errors.Is(err, ErrMissingType) {
		t.Fatalf("expected ErrMissingType to propagate through ValidateText for malformed input, got %v", err)
	}
}
