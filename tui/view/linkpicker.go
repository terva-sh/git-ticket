package view

import (
	"fmt"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// LinkPicker offers a ticket's linked tickets, per the TKT-01M1RPZ0
// decision: the status-picker pattern rather than a focusable section,
// because a modal list reuses proven cursor semantics and leaves the
// scrolling viewport simple. Each row names the role the other ticket
// plays from the viewed ticket's side, then the columns the list view
// teaches, so nothing here reads differently from one screen up.
type LinkPicker struct {
	t     *ticket.Ticket
	links []Linked
	list  tui.List
}

// LinkAction is what a key resolved to. Zero means the picker stays up.
type LinkAction struct {
	// Open is the ticket to push onto the detail stack, nil when no
	// row was chosen.
	Open   *ticket.Ticket
	Cancel bool
	Quit   bool
}

// NewLinkPicker builds the picker over the links of t.
func NewLinkPicker(t *ticket.Ticket, links []Linked) *LinkPicker {
	p := &LinkPicker{t: t, links: links}
	p.list.SetTotal(len(links))
	return p
}

// HandleKey routes one key, the status picker's escape ladder.
func (p *LinkPicker) HandleKey(k tui.Key) LinkAction {
	switch {
	case k.Kind == tui.KeyCtrlC:
		return LinkAction{Quit: true}
	case k.Kind == tui.KeyEsc, k.Kind == tui.KeyRune && k.Rune == 'q':
		return LinkAction{Cancel: true}
	case k.Kind == tui.KeyEnter:
		if len(p.links) == 0 {
			return LinkAction{Cancel: true}
		}
		return LinkAction{Open: p.links[p.list.Cursor()].Ticket}
	}
	p.list.HandleKey(k)
	return LinkAction{}
}

// Render lays the picker out for a cols x rows terminal.
func (p *LinkPicker) Render(cols, rows int) []string {
	if rows < 5 {
		rows = 5
	}
	out := make([]string, 0, rows)
	out = append(out,
		dim("  "+p.t.ID),
		"  \x1b[1m"+p.t.Title+"\x1b[22m",
		dim("  linked tickets:"),
		"")

	if len(p.links) == 0 {
		out = append(out, dim("  this ticket links to nothing: no parent, no children, no dependencies"))
	}
	body := rows - len(out) - 1
	start, end := p.list.Window(body)
	for i := start; i < end; i++ {
		l := p.links[i]
		text := fmt.Sprintf("%-9s  %-11s  %-5s  %s", l.Role, l.Ticket.Status, l.Ticket.Type, l.Ticket.Title)
		if i == p.list.Cursor() {
			out = append(out, "\x1b[7m▸ "+text+"\x1b[27m")
		} else {
			out = append(out, "  "+text)
		}
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  ↑/↓ j/k move · Enter open · Esc cancel"))
	return out
}
