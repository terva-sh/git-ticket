package view

import (
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// DetailView is one ticket, whole: the frontmatter that matters as a
// fixed header, then every non-empty body section under its heading,
// scrollable. The text is the ticket's own Markdown, wrapped to the
// pane and otherwise untouched, which plan 14 already promises reads
// correctly as written; styled rendering is a later lift and this view
// does not wait for it.
type DetailView struct {
	t  *ticket.Ticket
	vp tui.Viewport

	// body is the built section lines, cached against the width that
	// built them, because a Draw happens on every keypress and the
	// wrap is the expensive part.
	body     []string
	bodyCols int

	// msg is a one-render flash on the footer line, the same shape as
	// the list's: the copy confirmation lives here, and the next key
	// clears it back to the hints.
	msg string
}

// NewDetailView shows t. The caller keeps ownership of the ticket.
func NewDetailView(t *ticket.Ticket) *DetailView {
	return &DetailView{t: t}
}

// say puts m on the footer line until the next key.
func (d *DetailView) say(m string) { d.msg = m }

// HandleKey routes one key. back closes the view, quit ends the
// application. Esc, q, backspace, and h all go back, because every
// convention this view's users carry (tig, lazygit, vi) spells
// "leave" one of those ways. Ctrl+C is the one key that quits from
// anywhere. The y binding lives in the App, which holds the actions
// this view deliberately does not.
func (d *DetailView) HandleKey(k tui.Key) (back, quit bool) {
	d.msg = ""
	switch {
	case k.Kind == tui.KeyCtrlC:
		return false, true
	case k.Kind == tui.KeyEsc, k.Kind == tui.KeyBackspace,
		k.Kind == tui.KeyRune && (k.Rune == 'q' || k.Rune == 'h'):
		return true, false
	case k.Kind == tui.KeyRune && k.Rune == 'j':
		d.vp.HandleKey(tui.Key{Kind: tui.KeyDown})
	case k.Kind == tui.KeyRune && k.Rune == 'k':
		d.vp.HandleKey(tui.Key{Kind: tui.KeyUp})
	case k.Kind == tui.KeyRune && k.Rune == 'g':
		d.vp.Top()
	case k.Kind == tui.KeyRune && k.Rune == 'G':
		d.vp.Bottom()
	default:
		d.vp.HandleKey(k)
	}
	return false, false
}

// Render lays the ticket out for a cols x rows terminal: a fixed
// header, the scrolling body, and a footer of key hints.
func (d *DetailView) Render(cols, rows int) []string {
	if rows < 6 {
		rows = 6
	}
	header := d.header()
	paneRows := rows - len(header) - 1 // footer

	d.build(cols)
	d.vp.Fit(len(d.body), paneRows)

	out := make([]string, 0, rows)
	out = append(out, header...)
	start, end := d.vp.Window()
	for i := start; i < end; i++ {
		out = append(out, d.body[i])
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	if d.msg != "" {
		out = append(out, "  "+d.msg)
		return out
	}
	// The footer budget is 60 columns and it is full: the arrows and
	// now g/G go unhinted here and live on the ? page instead, because
	// t links earns the room more than a jump nobody reaches for
	// blind. The help page carries the complete detail key list.
	out = append(out, dim("  j/k scroll · y copy · t links · Esc back · Ctrl+C quit"))
	return out
}

func (d *DetailView) header() []string {
	t := d.t
	meta := []string{t.Status, t.Type, t.Priority}
	if len(t.Labels) > 0 {
		meta = append(meta, "labels: "+strings.Join(t.Labels, ", "))
	}
	if len(t.Assignees) > 0 {
		meta = append(meta, "assignees: "+strings.Join(t.Assignees, ", "))
	}
	if t.Milestone != nil && *t.Milestone != "" {
		meta = append(meta, "milestone: "+*t.Milestone)
	}
	if t.DueOn != nil && *t.DueOn != "" {
		meta = append(meta, "due: "+*t.DueOn)
	}
	if t.StatusReason != nil && *t.StatusReason != "" {
		meta = append(meta, "reason: "+*t.StatusReason)
	}
	return []string{
		dim("  " + t.ID),
		"  \x1b[1m" + t.Title + "\x1b[22m",
		dim("  " + strings.Join(meta, " · ")),
		"",
	}
}

// build renders the body sections once per width. The order is the
// render order of plan 5.3, Extra sections last in their own order.
func (d *DetailView) build(cols int) {
	if d.body != nil && d.bodyCols == cols {
		return
	}
	d.bodyCols = cols
	d.body = d.body[:0]
	limit := cols - 4 // two cells of margin each side
	if limit < 20 {
		limit = 20
	}

	b := d.t.Body
	sections := []struct{ heading, text string }{
		{"", b.Preamble},
		{"Description", b.Description},
		{"Acceptance criteria", b.AcceptanceCriteria},
		{"Definition of done", b.DefinitionOfDone},
		{"Implementation plan", b.ImplementationPlan},
		{"Notes", b.Notes},
		{"Comments", b.Comments},
		{"Summary", b.Summary},
	}
	for _, s := range b.Extra {
		sections = append(sections, struct{ heading, text string }{s.Heading, s.Text})
	}

	for _, s := range sections {
		text := strings.TrimRight(s.text, "\n")
		if strings.TrimSpace(text) == "" {
			continue
		}
		if len(d.body) > 0 {
			d.body = append(d.body, "")
		}
		if s.heading != "" {
			d.body = append(d.body, "  \x1b[1m"+s.heading+"\x1b[22m")
		}
		// Styled rendering fills the slot named when the view shipped:
		// RenderMarkdown styles the block, and the keep-style wrap folds
		// each styled line to the width without a colored span's tail
		// dropping to the default color at the fold.
		styled := tui.RenderMarkdown(text, tui.DefaultTheme, limit)
		for _, line := range strings.Split(styled, "\n") {
			if line == "" {
				d.body = append(d.body, "")
				continue
			}
			for _, w := range tui.WrapANSILineKeepStyle(line, limit) {
				d.body = append(d.body, "  "+w)
			}
		}
	}
	if len(d.body) == 0 {
		d.body = append(d.body, dim("  This ticket has no body sections."))
	}
}
