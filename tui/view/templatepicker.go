package view

import (
	"github.com/terva-sh/git-ticket/tui"
)

// TemplatePicker offers the store's templates when n is pressed and
// the store defines any, per plan 4.2. The first option is always a
// blank ticket, because a template is something you may ask for, not a
// gate in front of creating: the person who wants no template presses
// Enter once and is where n used to put them.
type TemplatePicker struct {
	choices []TemplateChoice
	list    tui.List
}

// TemplateAction is what a key resolved to. Zero means the picker
// stays up.
type TemplateAction struct {
	// Choose opens the create form seeded from this template. Blank
	// opens the bare form. At most one is set.
	Choose *TemplateChoice
	Blank  bool
	Cancel bool
	Quit   bool
}

// NewTemplatePicker builds the picker over the store's choices.
func NewTemplatePicker(choices []TemplateChoice) *TemplatePicker {
	p := &TemplatePicker{choices: choices}
	p.list.SetTotal(len(choices) + 1) // row 0 is the blank ticket
	return p
}

// HandleKey routes one key, the picker escape ladder.
func (p *TemplatePicker) HandleKey(k tui.Key) TemplateAction {
	switch {
	case k.Kind == tui.KeyCtrlC:
		return TemplateAction{Quit: true}
	case k.Kind == tui.KeyEsc, k.Kind == tui.KeyRune && k.Rune == 'q':
		return TemplateAction{Cancel: true}
	case k.Kind == tui.KeyEnter:
		if p.list.Cursor() == 0 {
			return TemplateAction{Blank: true}
		}
		return TemplateAction{Choose: &p.choices[p.list.Cursor()-1]}
	}
	p.list.HandleKey(k)
	return TemplateAction{}
}

// Render lays the picker out for a cols x rows terminal.
func (p *TemplatePicker) Render(cols, rows int) []string {
	if rows < 5 {
		rows = 5
	}
	out := make([]string, 0, rows)
	out = append(out,
		"  \x1b[1mNew ticket\x1b[22m",
		dim("  start from:"),
		"")

	row := func(i int, label string) string {
		if i == p.list.Cursor() {
			return "\x1b[7m▸ " + label + "\x1b[27m"
		}
		return "  " + label
	}
	body := rows - len(out) - 1
	start, end := p.list.Window(body)
	for i := start; i < end; i++ {
		if i == 0 {
			out = append(out, row(0, "a blank ticket"))
			continue
		}
		out = append(out, row(i, "template: "+p.choices[i-1].Name))
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  ↑/↓ j/k move · Enter choose · Esc cancel"))
	return out
}
