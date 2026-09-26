package view

import (
	"fmt"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// ReferencePicker lets a person choose exactly which resolved target to open.
// Opaque references remain visible and explain why Enter cannot open them.
type ReferencePicker struct {
	t    *ticket.Ticket
	refs []ReferenceTarget
	list tui.List
}

type ReferenceAction struct {
	OpenTarget string
	NoTarget   bool
	Cancel     bool
	Quit       bool
}

func NewReferencePicker(t *ticket.Ticket, refs []ReferenceTarget) *ReferencePicker {
	p := &ReferencePicker{t: t, refs: refs}
	p.list.SetTotal(len(refs))
	return p
}

func (p *ReferencePicker) HandleKey(k tui.Key) ReferenceAction {
	switch {
	case k.Kind == tui.KeyCtrlC:
		return ReferenceAction{Quit: true}
	case k.Kind == tui.KeyEsc, k.Kind == tui.KeyRune && k.Rune == 'q':
		return ReferenceAction{Cancel: true}
	case k.Kind == tui.KeyEnter:
		if len(p.refs) == 0 {
			return ReferenceAction{Cancel: true}
		}
		target := p.refs[p.list.Cursor()].Openable()
		if target == "" {
			return ReferenceAction{NoTarget: true}
		}
		return ReferenceAction{OpenTarget: target}
	}
	p.list.HandleKey(k)
	return ReferenceAction{}
}

func (p *ReferencePicker) Render(cols, rows int) []string {
	if rows < 5 {
		rows = 5
	}
	out := []string{
		dim("  " + p.t.ID),
		"  \x1b[1m" + p.t.Title + "\x1b[22m",
		dim("  reference targets:"),
		"",
	}
	if len(p.refs) == 0 {
		out = append(out, dim("  this ticket has no references"))
	}
	start, end := p.list.Window(rows - len(out) - 1)
	for i := start; i < end; i++ {
		ref := p.refs[i]
		target := ref.Openable()
		if target == "" {
			target = "opaque"
		}
		line := fmt.Sprintf("%s -> %s", ref.Reference.Ref, target)
		if i == p.list.Cursor() {
			out = append(out, "\x1b[7m▸ "+line+"\x1b[27m")
		} else {
			out = append(out, "  "+line)
		}
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  ↑/↓ j/k move · Enter open · Esc cancel"))
	return out
}
