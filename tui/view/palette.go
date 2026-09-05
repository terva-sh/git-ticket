package view

import (
	"os"

	"github.com/terva-sh/git-ticket/ticket"
)

// Row palettes, decided in the TKT-01M1S022 session on 2026-09-05.
// The session picked priority as the default, and rather than deleting
// the other candidates it kept them: p cycles the palettes at runtime,
// so color composes with the filter and the sort the way the user
// mixes them. GIT_TICKET_UI_PALETTE still selects the starting
// palette, for a muscle memory that outlived the session.
//
// Two rules every palette obeys, per the ticket's constraints. The
// selected row keeps plain reverse-video and takes no palette color,
// so the cursor is legible over any palette. And no meaning rides on
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

// palettes is the cycle order of the p key, the default first. Each
// colors the exceptional and leaves the ordinary alone: a palette that
// colors everything makes color mean nothing.
var palettes = []rowPalette{
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
	// dim uses weight and no hue at all, for the day hue is noise.
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
	paletteOff,
}

// startPalette resolves the palette a view opens with: the index into
// palettes, and whether NO_COLOR pins the colors off. NO_COLOR wins by
// presence alone, per no-color.org, and it also locks the p key,
// because an environment that asked for no color should not be one
// keystroke from getting some. GIT_TICKET_UI_PALETTE selects the
// start; an unknown name is the default rather than an error, because
// a typo in an environment variable should not cost a usable list.
func startPalette() (idx int, locked bool) {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		for i, p := range palettes {
			if p.name == "off" {
				return i, true
			}
		}
	}
	want := os.Getenv("GIT_TICKET_UI_PALETTE")
	for i, p := range palettes {
		if p.name == want {
			return i, false
		}
	}
	return 0, false // priority, the session's pick
}
