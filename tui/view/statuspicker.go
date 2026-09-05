package view

import (
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// StatusPicker offers the transitions the lifecycle permits from the
// ticket's current status, per plan 6.2. The options come from
// ticket.PermittedTransitions, so the picker cannot offer a move the
// library would refuse, and a move the library later refuses anyway
// (a race, a rule the picker does not know) still lands as an error
// in the list footer.
//
// Choosing blocked opens a one-line reason prompt, because 6.2 makes
// the reason mandatory there and asking is cheaper than bouncing the
// refusal off the store.
type StatusPicker struct {
	t       *ticket.Ticket
	options []string
	list    tui.List

	// reason mode: set when a reason-requiring status was chosen and
	// the prompt is up. reason holds the text typed so far.
	asking bool
	chosen string
	reason string
}

// PickerAction is what a key resolved to. Zero means the picker stays
// up and nothing else happens.
type PickerAction struct {
	Apply          bool
	Status, Reason string
	Cancel         bool
	Quit           bool
}

// NewStatusPicker builds the picker for t.
func NewStatusPicker(t *ticket.Ticket) *StatusPicker {
	p := &StatusPicker{t: t, options: ticket.PermittedTransitions(t.Status)}
	p.list.SetTotal(len(p.options))
	return p
}

// HandleKey routes one key.
func (p *StatusPicker) HandleKey(k tui.Key) PickerAction {
	if k.Kind == tui.KeyCtrlC {
		return PickerAction{Quit: true}
	}
	if p.asking {
		return p.handleReasonKey(k)
	}
	switch k.Kind {
	case tui.KeyEsc:
		return PickerAction{Cancel: true}
	case tui.KeyEnter:
		if len(p.options) == 0 {
			return PickerAction{Cancel: true}
		}
		chosen := p.options[p.list.Cursor()]
		if chosen == ticket.StatusBlocked {
			p.asking, p.chosen = true, chosen
			return PickerAction{}
		}
		return PickerAction{Apply: true, Status: chosen}
	case tui.KeyRune:
		if k.Rune == 'q' {
			return PickerAction{Cancel: true}
		}
	}
	p.list.HandleKey(k)
	return PickerAction{}
}

// handleReasonKey is the one-line prompt: runes append, backspace
// deletes, Enter confirms, Esc returns to the options. It is not the
// editor lift; it is the smallest input that makes blocked reachable,
// and the full editor replaces it when the edit flows land.
func (p *StatusPicker) handleReasonKey(k tui.Key) PickerAction {
	switch k.Kind {
	case tui.KeyEsc:
		p.asking, p.reason = false, ""
		return PickerAction{}
	case tui.KeyBackspace:
		if p.reason != "" {
			r := []rune(p.reason)
			p.reason = string(r[:len(r)-1])
		}
		return PickerAction{}
	case tui.KeyEnter:
		reason := strings.TrimSpace(p.reason)
		if reason == "" {
			// 6.2 requires the reason; an empty confirm re-prompts
			// rather than sending a write the store will refuse.
			return PickerAction{}
		}
		return PickerAction{Apply: true, Status: p.chosen, Reason: reason}
	case tui.KeyPaste:
		p.reason += strings.ReplaceAll(k.Paste, "\n", " ")
		return PickerAction{}
	case tui.KeyRune:
		p.reason += string(k.Rune)
		return PickerAction{}
	}
	return PickerAction{}
}

// Render lays the picker out for a cols x rows terminal.
func (p *StatusPicker) Render(cols, rows int) []string {
	if rows < 5 {
		rows = 5
	}
	out := make([]string, 0, rows)
	out = append(out,
		dim("  "+p.t.ID),
		"  \x1b[1m"+p.t.Title+"\x1b[22m",
		dim("  move from "+p.t.Status+" to:"),
		"")

	if p.asking {
		out = append(out,
			"  a reason is required to enter "+p.chosen+":",
			"",
			"  > "+p.reason+"\x1b[7m \x1b[27m")
		for len(out) < rows-1 {
			out = append(out, "")
		}
		out = append(out, dim("  Enter apply · Esc back to the choices · Ctrl+C quit"))
		return out
	}

	if len(p.options) == 0 {
		out = append(out, dim("  no transition is permitted from "+p.t.Status))
	}
	body := rows - len(out) - 1
	start, end := p.list.Window(body)
	for i := start; i < end; i++ {
		if i == p.list.Cursor() {
			out = append(out, "\x1b[7m▸ "+p.options[i]+"\x1b[27m")
		} else {
			out = append(out, "  "+p.options[i])
		}
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  ↑/↓ j/k move · Enter apply · Esc cancel"))
	return out
}
