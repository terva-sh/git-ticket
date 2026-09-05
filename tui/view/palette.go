package view

import (
	"os"

	"github.com/terva-sh/git-ticket/ticket"
)

// This file is session scaffolding for TKT-01M1S022: switchable
// candidate palettes for the decide-and-test session, selected per run
// with GIT_TICKET_UI_PALETTE. No palette is the shipped choice yet.
// When the session picks one, the winner moves into Theme fields in
// tui/theme.go, the losers are recorded on the ticket, and the
// environment variable goes away with this file.
//
// Two rules every candidate obeys, per the ticket's constraints. The
// selected row keeps plain reverse-video and takes no palette color,
// so the cursor is legible over any candidate. And no meaning rides on
// red-green alone: the hues in use are cyan, magenta, yellow, and
// blue, plus weight, so the common forms of color blindness keep every
// distinction that exists.

// sgr fragments. 256-color foregrounds, the space Theme already uses.
const (
	sgrDim     = "\x1b[2m"
	sgrBold    = "\x1b[1m"
	sgrCyan    = "\x1b[38;5;81m"
	sgrMagenta = "\x1b[38;5;176m"
	sgrYellow  = "\x1b[38;5;220m"
	sgrBlue    = "\x1b[38;5;111m"
)

// rowPalette maps a ticket to the SGR prefix its row opens with. An
// empty string is "leave the row alone", which is every palette's
// answer for the ordinary case: a candidate that colors everything
// makes color mean nothing.
type rowPalette struct {
	name  string
	color func(t *ticket.Ticket) string
}

// paletteOff is the baseline: the list as it ships today.
var paletteOff = rowPalette{name: "off", color: func(*ticket.Ticket) string { return "" }}

// palettes is the candidate set for the session, per the driving
// fields the ticket names. Each colors the exceptional and leaves the
// ordinary alone.
var palettes = []rowPalette{
	paletteOff,
	{name: "status", color: func(t *ticket.Ticket) string {
		switch t.Status {
		case ticket.StatusInProgress:
			return sgrCyan
		case ticket.StatusReview:
			return sgrMagenta
		case ticket.StatusBlocked:
			return sgrYellow
		case ticket.StatusDraft:
			return sgrDim
		case ticket.StatusDone, ticket.StatusArchived:
			return sgrDim
		}
		return "" // ready is the ordinary case
	}},
	{name: "priority", color: func(t *ticket.Ticket) string {
		switch t.Priority {
		case "urgent":
			return sgrBold + sgrYellow
		case "high":
			return sgrBlue
		case "low":
			return sgrDim
		}
		return "" // normal is the ordinary case
	}},
	{name: "type", color: func(t *ticket.Ticket) string {
		switch t.Type {
		case "epic":
			return sgrMagenta
		case "bug":
			return sgrYellow
		case "spike":
			return sgrCyan
		case "chore":
			return sgrDim
		}
		return "" // task is the ordinary case
	}},
	// dim uses weight and no hue at all: the control candidate, and
	// the answer if hue turns out to be noise.
	{name: "dim", color: func(t *ticket.Ticket) string {
		switch {
		case t.Status == ticket.StatusInProgress:
			return sgrBold
		case t.Status == ticket.StatusDraft, t.Status == ticket.StatusDone,
			t.Status == ticket.StatusArchived:
			return sgrDim
		}
		return ""
	}},
}

// activePalette reads GIT_TICKET_UI_PALETTE and honors NO_COLOR, whose
// presence wins over any selection, per the convention at
// no-color.org: the uncolored list is the baseline, not a degraded
// mode. An unknown name is off rather than an error, because a typo in
// an environment variable should not cost a usable list.
func activePalette() rowPalette {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return paletteOff
	}
	want := os.Getenv("GIT_TICKET_UI_PALETTE")
	for _, p := range palettes {
		if p.name == want {
			return p
		}
	}
	return paletteOff
}
