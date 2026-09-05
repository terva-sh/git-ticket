package view

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// FormView is the create and edit form: a title line and a description
// body, both on the lifted tui.Editor. One form serves both flows,
// because the only difference between them is whether ref is set and
// which action the save calls.
//
// The key contract follows the editor's own: a bare Enter on the title
// moves to the description, a bare Enter in the description inserts a
// newline (the editor's submit is repurposed, since a form is not a
// chat composer), Tab switches fields, Ctrl+S saves, Esc cancels.
type FormView struct {
	ref      string // ID under edit; empty means create
	revision string // the revision the edit is preconditioned on
	title    *tui.Editor
	desc     *tui.Editor
	focus    int // 0 title, 1 description
	errMsg   string
	// template is the store template this create seeds from, per plan
	// 4.2, empty for a bare create and always empty for an edit. The
	// header names it, because an invisible mode reads as broken.
	template string
}

// FormAction is what a key resolved to. Zero means the form stays up.
type FormAction struct {
	Save   bool
	Cancel bool
	Quit   bool
}

// NewCreateForm is an empty form; save files a new ticket.
func NewCreateForm() *FormView {
	return &FormView{title: tui.NewEditor(""), desc: tui.NewEditor("")}
}

// NewCreateFormFrom is the create form seeded from a template, per
// plan 4.2: the description editor holds the template's skeleton so
// the person edits it instead of typing over an invisible one, and
// the template's name rides to the store with the save, where the
// remaining fields seed.
func NewCreateFormFrom(tc TemplateChoice) *FormView {
	f := NewCreateForm()
	f.template = tc.Name
	f.desc.SetValue(tc.Description)
	return f
}

// NewEditForm prefills from t and remembers the revision the edit is
// preconditioned on. The caller keeps ownership of t.
func NewEditForm(t *ticket.Ticket) *FormView {
	f := &FormView{ref: t.ID, revision: t.Revision, title: tui.NewEditor(""), desc: tui.NewEditor("")}
	f.title.SetValue(t.Title)
	f.desc.SetValue(strings.TrimRight(t.Body.Description, "\n"))
	return f
}

// Values returns the title flattened to one line and the description
// as typed. The title editor can only grow a newline through a
// modified Enter, and flattening at the seam beats refusing at the
// store, which enforces its own rules anyway.
func (f *FormView) Values() (title, description string) {
	return strings.Join(strings.Fields(f.title.Value()), " "), f.desc.Value()
}

// SetError shows msg above the fields until the next key replaces it.
func (f *FormView) SetError(msg string) { f.errMsg = msg }

// Refresh re-arms the revision precondition after a lost race. The
// text stays as typed: the person sees the conflict notice, and the
// next Ctrl+S is their deliberate decision to replace the other
// writer's version.
func (f *FormView) Refresh(revision string) { f.revision = revision }

// HandleKey routes one key.
func (f *FormView) HandleKey(k tui.Key) FormAction {
	switch {
	case k.Kind == tui.KeyCtrlC:
		return FormAction{Quit: true}
	case k.Kind == tui.KeyEsc:
		return FormAction{Cancel: true}
	case k.Kind == tui.KeyCtrlS:
		if title, _ := f.Values(); title == "" {
			f.errMsg = "a title is required"
			return FormAction{}
		}
		return FormAction{Save: true}
	case k.Kind == tui.KeyTab, k.Kind == tui.KeyShiftTab:
		f.focus = 1 - f.focus
		return FormAction{}
	}
	ed := f.title
	if f.focus == 1 {
		ed = f.desc
	}
	if ed.HandleKey(k) {
		// The editor's submit: a bare Enter. On the title it is "next
		// field"; in the description it is a newline, because prose
		// has paragraphs and the save chord is Ctrl+S.
		if f.focus == 0 {
			f.focus = 1
		} else {
			ed.Insert("\n")
		}
	}
	return FormAction{}
}

// Render lays the form out for a cols x rows terminal.
func (f *FormView) Render(cols, rows int) []string {
	if rows < 8 {
		rows = 8
	}
	width := cols - 4
	if width < 20 {
		width = 20
	}

	out := make([]string, 0, rows)
	switch {
	case f.ref != "":
		out = append(out, "  \x1b[1mEdit\x1b[22m "+dim(f.ref))
	case f.template != "":
		out = append(out, "  \x1b[1mNew ticket\x1b[22m "+dim("from template: "+f.template))
	default:
		out = append(out, "  \x1b[1mNew ticket\x1b[22m")
	}
	if f.errMsg != "" {
		out = append(out, "  ! "+f.errMsg)
	}
	out = append(out, "", dim("  Title"))

	titleLines, tRow, tCol := f.title.Render(width)
	out = append(out, indentWithCaret(titleLines, tRow, tCol, f.focus == 0)...)

	out = append(out, "", dim("  Description"))
	descLines, dRow, dCol := f.desc.Render(width)
	descLines = indentWithCaret(descLines, dRow, dCol, f.focus == 1)

	// Clip the description to the rows that remain, keeping the caret
	// row visible: the window slides only when the caret would leave it.
	room := rows - len(out) - 1
	if room < 1 {
		room = 1
	}
	if len(descLines) > room {
		start := 0
		if f.focus == 1 && dRow >= room {
			start = dRow - room + 1
		}
		descLines = descLines[start : start+room]
	}
	out = append(out, descLines...)

	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim("  Tab field · Enter next/newline · Ctrl+S save · Esc cancel"))
	return out
}

// indentWithCaret indents editor lines by two cells and paints an
// inverse-video caret at (row, col) when the field has focus. The
// editor renders plain text, so walking runes by width is sound.
func indentWithCaret(lines []string, row, col int, focused bool) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		if focused && i == row {
			out[i] = "  " + caretInto(line, col)
			continue
		}
		out[i] = "  " + line
	}
	if len(out) == 0 {
		if focused {
			return []string{"  " + caretInto("", 0)}
		}
		return []string{"  "}
	}
	return out
}

// caretInto inverts the cell at visual column col, or appends a block
// when the caret sits past the end of the line.
func caretInto(line string, col int) string {
	seen := 0
	for i, r := range line {
		w := runewidth.RuneWidth(r)
		if seen+w > col {
			return line[:i] + "\x1b[7m" + string(r) + "\x1b[27m" + line[i+len(string(r)):]
		}
		seen += w
	}
	return line + "\x1b[7m \x1b[27m"
}
